package provisioning

import (
	"context"
	"testing"

	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
	"github.com/grafana/grafana/pkg/services/ngalert/models"
	"github.com/grafana/grafana/pkg/services/secrets"
	"github.com/grafana/grafana/pkg/services/secrets/database"
	"github.com/grafana/grafana/pkg/services/secrets/manager"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/prometheus/alertmanager/config"
	"github.com/stretchr/testify/require"
)

func TestContactPointService(t *testing.T) {
	sqlStore := sqlstore.InitTestDB(t)
	secretsService := manager.SetupTestService(t, database.ProvideSecretsStore(sqlStore))
	t.Run("service gets contact points from AM config", func(t *testing.T) {
		sut := createContactPointServiceSut(secretsService)

		cps, err := sut.GetContactPoints(context.Background(), 1)
		require.NoError(t, err)

		require.Len(t, cps, 1)
		require.Equal(t, "email receiver", cps[0].Name)
	})

	t.Run("service stitches contact point into org's AM config", func(t *testing.T) {
		sut := createContactPointServiceSut(secretsService)
		newCp := createTestContactPoint()

		_, err := sut.CreateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)
		require.NoError(t, err)

		cps, err := sut.GetContactPoints(context.Background(), 1)
		require.NoError(t, err)
		require.Len(t, cps, 2)
		require.Equal(t, "test-contact-point", cps[1].Name)
		require.Equal(t, "slack", cps[1].Type)
	})

	t.Run("it's possbile to use a custom uid", func(t *testing.T) {
		customUID := "1337"
		sut := createContactPointServiceSut(secretsService)
		newCp := createTestContactPoint()
		newCp.UID = customUID

		_, err := sut.CreateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)
		require.NoError(t, err)

		cps, err := sut.GetContactPoints(context.Background(), 1)
		require.NoError(t, err)
		require.Len(t, cps, 2)
		require.Equal(t, customUID, cps[1].UID)
	})

	t.Run("it's not possbile to use the same uid twice", func(t *testing.T) {
		customUID := "1337"
		sut := createContactPointServiceSut(secretsService)
		newCp := createTestContactPoint()
		newCp.UID = customUID

		_, err := sut.CreateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)
		require.NoError(t, err)

		_, err = sut.CreateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)
		require.Error(t, err)
	})

	t.Run("create rejects contact points that fail validation", func(t *testing.T) {
		sut := createContactPointServiceSut(secretsService)
		newCp := createTestContactPoint()
		newCp.Type = ""

		_, err := sut.CreateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)

		require.ErrorIs(t, err, ErrValidation)
	})

	t.Run("update rejects contact points with no settings", func(t *testing.T) {
		sut := createContactPointServiceSut(secretsService)
		newCp := createTestContactPoint()
		newCp, err := sut.CreateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)
		require.NoError(t, err)
		newCp.Settings = nil

		err = sut.UpdateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)

		require.ErrorIs(t, err, ErrValidation)
	})

	t.Run("update rejects contact points with no type", func(t *testing.T) {
		sut := createContactPointServiceSut(secretsService)
		newCp := createTestContactPoint()
		newCp, err := sut.CreateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)
		require.NoError(t, err)
		newCp.Type = ""

		err = sut.UpdateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)

		require.ErrorIs(t, err, ErrValidation)
	})

	t.Run("update rejects contact points which fail validation after merging", func(t *testing.T) {
		sut := createContactPointServiceSut(secretsService)
		newCp := createTestContactPoint()
		newCp, err := sut.CreateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)
		require.NoError(t, err)
		newCp.Settings, _ = simplejson.NewJson([]byte(`{}`))

		err = sut.UpdateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)

		require.ErrorIs(t, err, ErrValidation)
	})

	t.Run("default provenance of contact points is none", func(t *testing.T) {
		sut := createContactPointServiceSut(secretsService)

		cps, err := sut.GetContactPoints(context.Background(), 1)
		require.NoError(t, err)

		require.Equal(t, models.ProvenanceNone, models.Provenance(cps[0].Provenance))
	})

	t.Run("it's possible to update provenance from none to API", func(t *testing.T) {
		sut := createContactPointServiceSut(secretsService)
		newCp := createTestContactPoint()

		newCp, err := sut.CreateContactPoint(context.Background(), 1, newCp, models.ProvenanceNone)
		require.NoError(t, err)

		cps, err := sut.GetContactPoints(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, newCp.UID, cps[1].UID)
		require.Equal(t, models.ProvenanceNone, models.Provenance(cps[1].Provenance))

		err = sut.UpdateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)
		require.NoError(t, err)

		cps, err = sut.GetContactPoints(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, newCp.UID, cps[1].UID)
		require.Equal(t, models.ProvenanceAPI, models.Provenance(cps[1].Provenance))
	})

	t.Run("it's possible to update provenance from none to File", func(t *testing.T) {
		sut := createContactPointServiceSut(secretsService)
		newCp := createTestContactPoint()

		newCp, err := sut.CreateContactPoint(context.Background(), 1, newCp, models.ProvenanceNone)
		require.NoError(t, err)

		cps, err := sut.GetContactPoints(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, newCp.UID, cps[1].UID)
		require.Equal(t, models.ProvenanceNone, models.Provenance(cps[1].Provenance))

		err = sut.UpdateContactPoint(context.Background(), 1, newCp, models.ProvenanceFile)
		require.NoError(t, err)

		cps, err = sut.GetContactPoints(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, newCp.UID, cps[1].UID)
		require.Equal(t, models.ProvenanceFile, models.Provenance(cps[1].Provenance))
	})

	t.Run("it's not possible to update provenance from File to API", func(t *testing.T) {
		sut := createContactPointServiceSut(secretsService)
		newCp := createTestContactPoint()

		newCp, err := sut.CreateContactPoint(context.Background(), 1, newCp, models.ProvenanceFile)
		require.NoError(t, err)

		cps, err := sut.GetContactPoints(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, newCp.UID, cps[1].UID)
		require.Equal(t, models.ProvenanceFile, models.Provenance(cps[1].Provenance))

		err = sut.UpdateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)
		require.Error(t, err)
	})

	t.Run("it's not possible to update provenance from API to File", func(t *testing.T) {
		sut := createContactPointServiceSut(secretsService)
		newCp := createTestContactPoint()

		newCp, err := sut.CreateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)
		require.NoError(t, err)

		cps, err := sut.GetContactPoints(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, newCp.UID, cps[1].UID)
		require.Equal(t, models.ProvenanceAPI, models.Provenance(cps[1].Provenance))

		err = sut.UpdateContactPoint(context.Background(), 1, newCp, models.ProvenanceFile)
		require.Error(t, err)
	})

	t.Run("service respects concurrency token when updating", func(t *testing.T) {
		sut := createContactPointServiceSut(secretsService)
		newCp := createTestContactPoint()
		q := models.GetLatestAlertmanagerConfigurationQuery{
			OrgID: 1,
		}
		err := sut.amStore.GetLatestAlertmanagerConfiguration(context.Background(), &q)
		require.NoError(t, err)
		expectedConcurrencyToken := q.Result.ConfigurationHash

		_, err = sut.CreateContactPoint(context.Background(), 1, newCp, models.ProvenanceAPI)
		require.NoError(t, err)

		fake := sut.amStore.(*fakeAMConfigStore)
		intercepted := fake.lastSaveCommand
		require.Equal(t, expectedConcurrencyToken, intercepted.FetchedConfigurationHash)
	})
}

func TestContactPointInUse(t *testing.T) {
	result := isContactPointInUse("test", []*definitions.Route{
		{
			Receiver: "not-test",
			Routes: []*definitions.Route{
				{
					Receiver: "not-test",
				},
				{
					Receiver: "test",
				},
			},
		},
	})
	require.True(t, result)
	result = isContactPointInUse("test", []*definitions.Route{
		{
			Receiver: "not-test",
			Routes: []*definitions.Route{
				{
					Receiver: "not-test",
				},
				{
					Receiver: "not-test",
				},
			},
		},
	})
	require.False(t, result)
}

func createContactPointServiceSut(secretService secrets.Service) *ContactPointService {
	return &ContactPointService{
		amStore:           newFakeAMConfigStore(),
		provenanceStore:   NewFakeProvisioningStore(),
		xact:              newNopTransactionManager(),
		encryptionService: secretService,
		log:               log.NewNopLogger(),
	}
}

func createTestContactPoint() definitions.EmbeddedContactPoint {
	settings, _ := simplejson.NewJson([]byte(`{"recipient":"value_recipient","token":"value_token"}`))
	return definitions.EmbeddedContactPoint{
		Name:     "test-contact-point",
		Type:     "slack",
		Settings: settings,
	}
}

func TestStitchReceivers(t *testing.T) {
	type testCase struct {
		name        string
		initial     *definitions.PostableUserConfig
		new         *definitions.PostableGrafanaReceiver
		expModified bool
		expCfg      definitions.PostableApiAlertingConfig
	}

	cases := []testCase{
		{
			name: "non matching receiver by UID, no change",
			new: &definitions.PostableGrafanaReceiver{
				UID: "does not exist",
			},
			expModified: false,
			expCfg:      createTestConfigWithReceivers().AlertmanagerConfig,
		},
		{
			name: "matching receiver with unchanged name, replaces",
			new: &definitions.PostableGrafanaReceiver{
				UID:  "ghi",
				Name: "receiver-2",
				Type: "teams",
			},
			expModified: true,
			expCfg: definitions.PostableApiAlertingConfig{
				Receivers: []*definitions.PostableApiReceiver{
					{
						Receiver: config.Receiver{
							Name: "receiver-1",
						},
						PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
							GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
								{
									UID:  "abc",
									Name: "receiver-1",
									Type: "slack",
								},
							},
						},
					},
					{
						Receiver: config.Receiver{
							Name: "receiver-2",
						},
						PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
							GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
								{
									UID:  "def",
									Name: "receiver-2",
									Type: "slack",
								},
								{
									UID:  "ghi",
									Name: "receiver-2",
									Type: "teams",
								},
								{
									UID:  "jkl",
									Name: "receiver-2",
									Type: "discord",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "rename with only one receiver in group, renames group",
			new: &definitions.PostableGrafanaReceiver{
				UID:  "abc",
				Name: "new-receiver",
				Type: "slack",
			},
			expModified: true,
			expCfg: definitions.PostableApiAlertingConfig{
				Receivers: []*definitions.PostableApiReceiver{
					{
						Receiver: config.Receiver{
							Name: "new-receiver",
						},
						PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
							GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
								{
									UID:  "abc",
									Name: "new-receiver",
									Type: "slack",
								},
							},
						},
					},
					{
						Receiver: config.Receiver{
							Name: "receiver-2",
						},
						PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
							GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
								{
									UID:  "def",
									Name: "receiver-2",
									Type: "slack",
								},
								{
									UID:  "ghi",
									Name: "receiver-2",
									Type: "email",
								},
								{
									UID:  "jkl",
									Name: "receiver-2",
									Type: "discord",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "rename to another existing group, moves receiver",
			new: &definitions.PostableGrafanaReceiver{
				UID:  "def",
				Name: "receiver-1",
				Type: "slack",
			},
			expModified: true,
			expCfg: definitions.PostableApiAlertingConfig{
				Receivers: []*definitions.PostableApiReceiver{
					{
						Receiver: config.Receiver{
							Name: "receiver-1",
						},
						PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
							GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
								{
									UID:  "abc",
									Name: "receiver-1",
									Type: "slack",
								},
								{
									UID:  "def",
									Name: "receiver-1",
									Type: "slack",
								},
							},
						},
					},
					{
						Receiver: config.Receiver{
							Name: "receiver-2",
						},
						PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
							GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
								{
									UID:  "ghi",
									Name: "receiver-2",
									Type: "email",
								},
								{
									UID:  "jkl",
									Name: "receiver-2",
									Type: "discord",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "rename to a name that doesn't exist, creates new group and moves",
			new: &definitions.PostableGrafanaReceiver{
				UID:  "jkl",
				Name: "brand-new-group",
				Type: "opsgenie",
			},
			expModified: true,
			expCfg: definitions.PostableApiAlertingConfig{
				Receivers: []*definitions.PostableApiReceiver{
					{
						Receiver: config.Receiver{
							Name: "receiver-1",
						},
						PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
							GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
								{
									UID:  "abc",
									Name: "receiver-1",
									Type: "slack",
								},
							},
						},
					},
					{
						Receiver: config.Receiver{
							Name: "receiver-2",
						},
						PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
							GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
								{
									UID:  "def",
									Name: "receiver-2",
									Type: "slack",
								},
								{
									UID:  "ghi",
									Name: "receiver-2",
									Type: "email",
								},
							},
						},
					},
					{
						Receiver: config.Receiver{
							Name: "brand-new-group",
						},
						PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
							GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
								{
									UID:  "jkl",
									Name: "brand-new-group",
									Type: "opsgenie",
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "rename an inconsistent group in the database, algorithm fixes it",
			initial: createInconsistentTestConfigWithReceivers(),
			new: &definitions.PostableGrafanaReceiver{
				UID:  "ghi",
				Name: "brand-new-group",
				Type: "opsgenie",
			},
			expModified: true,
			expCfg: definitions.PostableApiAlertingConfig{
				Receivers: []*definitions.PostableApiReceiver{
					{
						Receiver: config.Receiver{
							Name: "receiver-1",
						},
						PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
							GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
								{
									UID:  "abc",
									Name: "receiver-1",
									Type: "slack",
								},
							},
						},
					},
					{
						Receiver: config.Receiver{
							Name: "receiver-2",
						},
						PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
							GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
								{
									UID:  "def",
									Name: "receiver-2",
									Type: "slack",
								},
								{
									UID:  "jkl",
									Name: "receiver-2",
									Type: "discord",
								},
							},
						},
					},
					{
						Receiver: config.Receiver{
							Name: "brand-new-group",
						},
						PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
							GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
								{
									UID:  "ghi",
									Name: "brand-new-group",
									Type: "opsgenie",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := createTestConfigWithReceivers()
			if c.initial != nil {
				cfg = c.initial
			}

			modified := stitchReceiver(cfg, c.new)

			require.Equal(t, c.expModified, modified)
			require.Equal(t, c.expCfg, cfg.AlertmanagerConfig)
		})
	}
}

func createTestConfigWithReceivers() *definitions.PostableUserConfig {
	return &definitions.PostableUserConfig{
		AlertmanagerConfig: definitions.PostableApiAlertingConfig{
			Receivers: []*definitions.PostableApiReceiver{
				{
					Receiver: config.Receiver{
						Name: "receiver-1",
					},
					PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
						GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
							{
								UID:  "abc",
								Name: "receiver-1",
								Type: "slack",
							},
						},
					},
				},
				{
					Receiver: config.Receiver{
						Name: "receiver-2",
					},
					PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
						GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
							{
								UID:  "def",
								Name: "receiver-2",
								Type: "slack",
							},
							{
								UID:  "ghi",
								Name: "receiver-2",
								Type: "email",
							},
							{
								UID:  "jkl",
								Name: "receiver-2",
								Type: "discord",
							},
						},
					},
				},
			},
		},
	}
}

// This is an invalid config, with inconsistently named receivers (intentionally).
func createInconsistentTestConfigWithReceivers() *definitions.PostableUserConfig {
	return &definitions.PostableUserConfig{
		AlertmanagerConfig: definitions.PostableApiAlertingConfig{
			Receivers: []*definitions.PostableApiReceiver{
				{
					Receiver: config.Receiver{
						Name: "receiver-1",
					},
					PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
						GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
							{
								UID:  "abc",
								Name: "receiver-1",
								Type: "slack",
							},
						},
					},
				},
				{
					Receiver: config.Receiver{
						Name: "receiver-2",
					},
					PostableGrafanaReceivers: definitions.PostableGrafanaReceivers{
						GrafanaManagedReceivers: []*definitions.PostableGrafanaReceiver{
							{
								UID:  "def",
								Name: "receiver-2",
								Type: "slack",
							},
							{
								UID:  "ghi",
								Name: "receiver-3",
								Type: "email",
							},
							{
								UID:  "jkl",
								Name: "receiver-2",
								Type: "discord",
							},
						},
					},
				},
			},
		},
	}
}

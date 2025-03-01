import { css, cx } from '@emotion/css';
import pluralize from 'pluralize';
import React, { useEffect, useState } from 'react';
import { connect, ConnectedProps } from 'react-redux';

import { GrafanaTheme2, OrgRole } from '@grafana/data';
import { Alert, ConfirmModal, FilterInput, Icon, LinkButton, RadioButtonGroup, Tooltip, useStyles2 } from '@grafana/ui';
import EmptyListCTA from 'app/core/components/EmptyListCTA/EmptyListCTA';
import { Page } from 'app/core/components/Page/Page';
import PageLoader from 'app/core/components/PageLoader/PageLoader';
import { contextSrv } from 'app/core/core';
import { StoreState, ServiceAccountDTO, AccessControlAction, ServiceAccountStateFilter } from 'app/types';

import { CreateTokenModal, ServiceAccountToken } from './components/CreateTokenModal';
import ServiceAccountListItem from './components/ServiceAccountsListItem';
import {
  changeQuery,
  fetchACOptions,
  fetchServiceAccounts,
  deleteServiceAccount,
  updateServiceAccount,
  changeStateFilter,
  createServiceAccountToken,
  getApiKeysMigrationStatus,
  getApiKeysMigrationInfo,
  closeApiKeysMigrationInfo,
} from './state/actions';

interface OwnProps {}

export type Props = OwnProps & ConnectedProps<typeof connector>;

function mapStateToProps(state: StoreState) {
  return {
    ...state.serviceAccounts,
  };
}

const mapDispatchToProps = {
  changeQuery,
  fetchACOptions,
  fetchServiceAccounts,
  deleteServiceAccount,
  updateServiceAccount,
  changeStateFilter,
  createServiceAccountToken,
  getApiKeysMigrationStatus,
  getApiKeysMigrationInfo,
  closeApiKeysMigrationInfo,
};

const connector = connect(mapStateToProps, mapDispatchToProps);

export const ServiceAccountsListPageUnconnected = ({
  serviceAccounts,
  isLoading,
  roleOptions,
  builtInRoles,
  query,
  serviceAccountStateFilter,
  apiKeysMigrated,
  showApiKeysMigrationInfo,
  changeQuery,
  fetchACOptions,
  fetchServiceAccounts,
  deleteServiceAccount,
  updateServiceAccount,
  changeStateFilter,
  createServiceAccountToken,
  getApiKeysMigrationStatus,
  getApiKeysMigrationInfo,
  closeApiKeysMigrationInfo,
}: Props): JSX.Element => {
  const styles = useStyles2(getStyles);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [isRemoveModalOpen, setIsRemoveModalOpen] = useState(false);
  const [isDisableModalOpen, setIsDisableModalOpen] = useState(false);
  const [newToken, setNewToken] = useState('');
  const [currentServiceAccount, setCurrentServiceAccount] = useState<ServiceAccountDTO | null>(null);

  useEffect(() => {
    fetchServiceAccounts({ withLoadingIndicator: true });
    getApiKeysMigrationStatus();
    getApiKeysMigrationInfo();
    if (contextSrv.licensedAccessControlEnabled()) {
      fetchACOptions();
    }
  }, [fetchACOptions, fetchServiceAccounts, getApiKeysMigrationStatus, getApiKeysMigrationInfo]);

  const noServiceAccountsCreated =
    serviceAccounts.length === 0 && serviceAccountStateFilter === ServiceAccountStateFilter.All && !query;

  const onRoleChange = async (role: OrgRole, serviceAccount: ServiceAccountDTO) => {
    const updatedServiceAccount = { ...serviceAccount, role: role };
    updateServiceAccount(updatedServiceAccount);
    if (contextSrv.licensedAccessControlEnabled()) {
      fetchACOptions();
    }
  };

  const onQueryChange = (value: string) => {
    changeQuery(value);
  };

  const onStateFilterChange = (value: ServiceAccountStateFilter) => {
    changeStateFilter(value);
  };

  const onRemoveButtonClick = (serviceAccount: ServiceAccountDTO) => {
    setCurrentServiceAccount(serviceAccount);
    setIsRemoveModalOpen(true);
  };

  const onServiceAccountRemove = async () => {
    if (currentServiceAccount) {
      deleteServiceAccount(currentServiceAccount.id);
    }
    onRemoveModalClose();
  };

  const onDisableButtonClick = (serviceAccount: ServiceAccountDTO) => {
    setCurrentServiceAccount(serviceAccount);
    setIsDisableModalOpen(true);
  };

  const onDisable = () => {
    if (currentServiceAccount) {
      updateServiceAccount({ ...currentServiceAccount, isDisabled: true });
    }
    onDisableModalClose();
  };

  const onEnable = (serviceAccount: ServiceAccountDTO) => {
    updateServiceAccount({ ...serviceAccount, isDisabled: false });
  };

  const onTokenAdd = (serviceAccount: ServiceAccountDTO) => {
    setCurrentServiceAccount(serviceAccount);
    setIsAddModalOpen(true);
  };

  const onTokenCreate = async (token: ServiceAccountToken) => {
    if (currentServiceAccount) {
      createServiceAccountToken(currentServiceAccount.id, token, setNewToken);
    }
  };

  const onAddModalClose = () => {
    setIsAddModalOpen(false);
    setCurrentServiceAccount(null);
    setNewToken('');
  };

  const onRemoveModalClose = () => {
    setIsRemoveModalOpen(false);
    setCurrentServiceAccount(null);
  };

  const onDisableModalClose = () => {
    setIsDisableModalOpen(false);
    setCurrentServiceAccount(null);
  };

  const onMigrationInfoClose = () => {
    closeApiKeysMigrationInfo();
  };

  return (
    <Page navId="serviceaccounts">
      <Page.Contents>
        {apiKeysMigrated && showApiKeysMigrationInfo && (
          <Alert
            title="API keys migrated to Service accounts. Your keys are now called tokens and live inside respective service
          accounts. Learn more."
            severity="success"
            onRemove={onMigrationInfoClose}
          ></Alert>
        )}
        <div className={styles.pageHeader}>
          <h2>Service accounts</h2>
          <div className={styles.apiKeyInfoLabel}>
            <Tooltip
              placement="bottom"
              interactive
              content={
                <>
                  API keys are now service accounts with tokens. Find out more{' '}
                  <a href="https://grafana.com/docs/grafana/latest/administration/service-accounts/">here.</a>
                </>
              }
            >
              <Icon name="question-circle" />
            </Tooltip>
            <span>Looking for API keys?</span>
          </div>
          {!noServiceAccountsCreated && contextSrv.hasPermission(AccessControlAction.ServiceAccountsCreate) && (
            <LinkButton href="org/serviceaccounts/create" variant="primary">
              Add service account
            </LinkButton>
          )}
        </div>
        <div className={styles.filterRow}>
          <FilterInput placeholder="Search service account by name" value={query} onChange={onQueryChange} width={50} />
          <div className={styles.filterDelimiter}></div>
          <RadioButtonGroup
            options={[
              { label: 'All', value: ServiceAccountStateFilter.All },
              { label: 'With expiring tokens', value: ServiceAccountStateFilter.WithExpiredTokens },
              { label: 'Disabled', value: ServiceAccountStateFilter.Disabled },
            ]}
            onChange={onStateFilterChange}
            value={serviceAccountStateFilter}
            className={styles.filter}
          />
        </div>
        {isLoading && <PageLoader />}
        {!isLoading && noServiceAccountsCreated && (
          <>
            <EmptyListCTA
              title="You haven't created any service accounts yet."
              buttonIcon="key-skeleton-alt"
              buttonLink="org/serviceaccounts/create"
              buttonTitle="Add service account"
              buttonDisabled={!contextSrv.hasPermission(AccessControlAction.ServiceAccountsCreate)}
              proTip="Remember, you can provide specific permissions for API access to other applications."
              proTipLink=""
              proTipLinkTitle=""
              proTipTarget="_blank"
            />
          </>
        )}

        <>
          <div className={cx(styles.table, 'admin-list-table')}>
            <table className="filter-table filter-table--hover">
              <thead>
                <tr>
                  <th></th>
                  <th>Account</th>
                  <th>ID</th>
                  <th>Roles</th>
                  <th>Tokens</th>
                  <th style={{ width: '34px' }} />
                </tr>
              </thead>
              <tbody>
                {!isLoading &&
                  serviceAccounts.length !== 0 &&
                  serviceAccounts.map((serviceAccount: ServiceAccountDTO) => (
                    <ServiceAccountListItem
                      serviceAccount={serviceAccount}
                      key={serviceAccount.id}
                      builtInRoles={builtInRoles}
                      roleOptions={roleOptions}
                      onRoleChange={onRoleChange}
                      onRemoveButtonClick={onRemoveButtonClick}
                      onDisable={onDisableButtonClick}
                      onEnable={onEnable}
                      onAddTokenClick={onTokenAdd}
                    />
                  ))}
              </tbody>
            </table>
          </div>
        </>
        {currentServiceAccount && (
          <>
            <ConfirmModal
              isOpen={isRemoveModalOpen}
              body={`Are you sure you want to delete '${currentServiceAccount.name}'${
                !!currentServiceAccount.tokens
                  ? ` and ${currentServiceAccount.tokens} accompanying ${pluralize(
                      'token',
                      currentServiceAccount.tokens
                    )}`
                  : ''
              }?`}
              confirmText="Delete"
              title="Delete service account"
              onConfirm={onServiceAccountRemove}
              onDismiss={onRemoveModalClose}
            />
            <ConfirmModal
              isOpen={isDisableModalOpen}
              title="Disable service account"
              body={`Are you sure you want to disable '${currentServiceAccount.name}'?`}
              confirmText="Disable service account"
              onConfirm={onDisable}
              onDismiss={onDisableModalClose}
            />
            <CreateTokenModal
              isOpen={isAddModalOpen}
              token={newToken}
              serviceAccountLogin={currentServiceAccount.login}
              onCreateToken={onTokenCreate}
              onClose={onAddModalClose}
            />
          </>
        )}
      </Page.Contents>
    </Page>
  );
};

export const getStyles = (theme: GrafanaTheme2) => {
  return {
    table: css`
      margin-top: ${theme.spacing(3)};
    `,
    filter: css`
      margin: 0 ${theme.spacing(1)};
    `,
    row: css`
      display: flex;
      align-items: center;
      height: 100% !important;

      a {
        padding: ${theme.spacing(0.5)} 0 !important;
      }
    `,
    unitTooltip: css`
      display: flex;
      flex-direction: column;
    `,
    unitItem: css`
      cursor: pointer;
      padding: ${theme.spacing(0.5)} 0;
      margin-right: ${theme.spacing(1)};
    `,
    disabled: css`
      color: ${theme.colors.text.disabled};
    `,
    link: css`
      color: inherit;
      cursor: pointer;
      text-decoration: underline;
    `,
    pageHeader: css`
      display: flex;
      margin-bottom: ${theme.spacing(2)};
    `,
    apiKeyInfoLabel: css`
      margin-left: ${theme.spacing(1)};
      line-height: 2.2;
      flex-grow: 1;
      color: ${theme.colors.text.secondary};

      span {
        padding: ${theme.spacing(0.5)};
      }
    `,
    filterRow: cx(
      'page-action-bar',
      css`
        display: flex;
        justifycontent: flex-end;
      `
    ),
    filterDelimiter: css`
      flex-grow: 1;
    `,
  };
};

const ServiceAccountsListPage = connector(ServiceAccountsListPageUnconnected);
export default ServiceAccountsListPage;

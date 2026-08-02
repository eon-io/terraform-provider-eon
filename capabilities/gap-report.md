# Eon SDK / Terraform provider capability gap report

SDK release: `github.com/eon-io/eon-sdk-go@v1.173.0`

| Total | Covered | Covered (internal) | Gaps | Skipped | Needs review |
|---|---|---|---|---|---|
| 104 | 58 | 1 | 27 | 18 | 2 |

## Gaps: proposed new provider surface

One group per capability; each group is expected to become one PR.

### `eon_backup_posture_control` (resource)

| Operation | Endpoint | Reasoning |
|---|---|---|
| `createBackupPostureControl` | `POST /v1/projects/{projectId}/backup-posture-controls` | Create with matching item read: a durable object with stable identity and a real lifecycle. |
| `deleteBackupPostureControl` | `DELETE /v1/projects/{projectId}/backup-posture-controls/{controlId}` | Read+write (+delete) on a stable path: a durable configuration object Terraform can reconcile and drift-detect. |
| `getBackupPostureControl` | `GET /v1/projects/{projectId}/backup-posture-controls/{controlId}` | Read+write (+delete) on a stable path: a durable configuration object Terraform can reconcile and drift-detect. |
| `updateBackupPostureControl` | `PUT /v1/projects/{projectId}/backup-posture-controls/{controlId}` | Read+write (+delete) on a stable path: a durable configuration object Terraform can reconcile and drift-detect. |

### `eon_backup_posture_controls` (data_source)

| Operation | Endpoint | Reasoning |
|---|---|---|
| `listBackupPostureControls` | `POST /v1/projects/{projectId}/backup-posture-controls/list` | List lookup mirroring the existing eon_backup_policies data source; ships alongside the eon_backup_posture_control resource. |

### `eon_idps` (data_source)

| Operation | Endpoint | Reasoning |
|---|---|---|
| `listIdps` | `POST /v1/idps/list` | Read-only lookup of configured identity providers; an IdP ID is required input for eon_idp_group role assignments. |

### `eon_permissions` (data_source)

| Operation | Endpoint | Reasoning |
|---|---|---|
| `listPermissions` | `GET /v1/permissions` | Read-only permission catalog; useful input for composing eon_role custom role permission lists. |

### `eon_resource_backup_exclusion` (resource)

| Operation | Endpoint | Reasoning |
|---|---|---|
| `cancelResourceBackupExclusion` | `PATCH /v1/projects/{projectId}/resources/{id}/include` | Exclude/include pair describes a durable per-object setting; model like the existing eon_volume_backup_exclusion (create = exclude, delete = include). |
| `excludeResourceFromBackup` | `PATCH /v1/projects/{projectId}/resources/{id}/exclude` | Exclude/include pair describes a durable per-object setting; model like the existing eon_volume_backup_exclusion (create = exclude, delete = include). |

### `eon_resource_data_classes_override` (resource)

| Operation | Endpoint | Reasoning |
|---|---|---|
| `overrideDataClasses` | `PATCH /v1/projects/{projectId}/resources/{id}/data-classifications` | Override set/remove pair for a resource's data classifications — a durable declarative setting; drift read comes from the parent resource's dataClasses. Model like eon_volume_backup_exclusion (create = override, delete = remove override). |
| `removeDataClassesOverride` | `DELETE /v1/projects/{projectId}/resources/{id}/data-classifications` | Delete half of the data classifications override pair; see overrideDataClasses. |

### `eon_resource_environment_override` (resource)

| Operation | Endpoint | Reasoning |
|---|---|---|
| `overrideEnvironment` | `PATCH /v1/projects/{projectId}/resources/{id}/environments` | Override set/remove pair for a resource's environment tag — a durable declarative setting; drift read comes from the parent resource's environment. Model like eon_volume_backup_exclusion (create = override, delete = remove override). |
| `removeEnvironmentOverride` | `DELETE /v1/projects/{projectId}/resources/{id}/environments` | Delete half of the environment override pair; see overrideEnvironment. |

### `eon_resource_snapshots` (data_source)

| Operation | Endpoint | Reasoning |
|---|---|---|
| `listResourceSnapshots` | `POST /v1/projects/{projectId}/resources/{id}/snapshots` | Read-only list of a resource's restorable snapshots — the natural input for eon_restore_job (selecting the snapshot to restore); complements the existing eon_snapshot by-ID lookup. |

### `eon_restore_account_metrics_config` (resource)

| Operation | Endpoint | Reasoning |
|---|---|---|
| `disableRestoreAccountMetricsConfig` | `DELETE /v1/projects/{projectId}/restore-accounts/{accountId}/metrics-config` | Read+write (+delete) on a stable path: a durable configuration object Terraform can reconcile and drift-detect. |
| `enableRestoreAccountMetricsConfig` | `PUT /v1/projects/{projectId}/restore-accounts/{accountId}/metrics-config` | Read+write (+delete) on a stable path: a durable configuration object Terraform can reconcile and drift-detect. |
| `getRestoreAccountMetricsConfig` | `GET /v1/projects/{projectId}/restore-accounts/{accountId}/metrics-config` | Read+write (+delete) on a stable path: a durable configuration object Terraform can reconcile and drift-detect. |

### `eon_restore_job` (resource)

| Operation | Endpoint | Reasoning |
|---|---|---|
| `restoreAzureDisk` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-azure-disk` | One-shot restore trigger; restores are modeled by the existing eon_restore_job resource — extend it with this restore type rather than adding new surface. |
| `restoreAzureSqlDatabase` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-azure-sql-database` | One-shot restore trigger; restores are modeled by the existing eon_restore_job resource — extend it with this restore type rather than adding new surface. |
| `restoreAzureVmInstance` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-azure-vm-instance` | One-shot restore trigger; restores are modeled by the existing eon_restore_job resource — extend it with this restore type rather than adding new surface. |
| `restoreDynamoDBTable` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-dynamo-db-table` | One-shot restore trigger; restores are modeled by the existing eon_restore_job resource — extend it with this restore type rather than adding new surface. |
| `restoreToEbsSnapshot` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/convert-ec2-ebs-snapshot` | One-shot restore trigger; restores are modeled by the existing eon_restore_job resource — extend it with this restore type rather than adding new surface. |

### `eon_snapshot_hold` (resource)

| Operation | Endpoint | Reasoning |
|---|---|---|
| `holdSnapshot` | `PATCH /v1/projects/{projectId}/snapshots/{id}/hold` | Hold/remove-hold pair describes a durable per-snapshot setting; model as a toggle resource (create = hold, delete = remove hold). |
| `removeSnapshotHold` | `PATCH /v1/projects/{projectId}/snapshots/{id}/remove-hold` | Hold/remove-hold pair describes a durable per-snapshot setting; model as a toggle resource (create = hold, delete = remove hold). |

### `eon_source_account_metrics_config` (resource)

| Operation | Endpoint | Reasoning |
|---|---|---|
| `disableSourceAccountMetricsConfig` | `DELETE /v1/projects/{projectId}/source-accounts/{accountId}/metrics-config` | Read+write (+delete) on a stable path: a durable configuration object Terraform can reconcile and drift-detect. |
| `enableSourceAccountMetricsConfig` | `PUT /v1/projects/{projectId}/source-accounts/{accountId}/metrics-config` | Read+write (+delete) on a stable path: a durable configuration object Terraform can reconcile and drift-detect. |
| `getSourceAccountMetricsConfig` | `GET /v1/projects/{projectId}/source-accounts/{accountId}/metrics-config` | Read+write (+delete) on a stable path: a durable configuration object Terraform can reconcile and drift-detect. |

## Skipped operations

Not exposed in Terraform; edit `capabilities/manifest.yaml` to override.

| Operation | Endpoint | Reason |
|---|---|---|
| `rotateCurrentApiClientSecret` | `POST /v1/api-credentials/current/rotate` | Authentication, token issuance, and credential rotation are session/security actions, not declarative infrastructure. |
| `rotateApiClientSecret` | `POST /v1/api-credentials/{clientId}/rotate` | Authentication, token issuance, and credential rotation are session/security actions, not declarative infrastructure. |
| `queryCostData` | `POST /v1/cost-data` | Cost and billing queries are pure reporting. |
| `getAccessTokenOAuth2` | `POST /v1/oauth2/token` | Authentication, token issuance, and credential rotation are session/security actions, not declarative infrastructure. |
| `getMyActionApprovalRequest` | `GET /v1/projects/{projectId}/action-approvals/my-requests/{requestId}` | Approval requests are one-shot workflow actions tied to a session, not durable configuration. |
| `cancelActionApprovalRequest` | `POST /v1/projects/{projectId}/action-approvals/my-requests/{requestId}/cancel` | Approval requests are one-shot workflow actions tied to a session, not durable configuration. |
| `createActionApprovalRequest` | `POST /v1/projects/{projectId}/action-approvals/my-requests/{requestId}/submit` | Approval requests are one-shot workflow actions tied to a session, not durable configuration. |
| `listBackupJobs` | `POST /v1/projects/{projectId}/backup-jobs` | Job listing/status polling is imperative monitoring of runtime activity; there is no desired state to reconcile. |
| `getBackupJob` | `GET /v1/projects/{projectId}/backup-jobs/{jobId}` | Job listing/status polling is imperative monitoring of runtime activity; there is no desired state to reconcile. |
| `getDailyStorageSummaries` | `GET /v1/projects/{projectId}/dashboard/daily-storage-summary` | Dashboard/metrics endpoints are pure reporting. |
| `getQueryResult` | `GET /v1/projects/{projectId}/queries/{queryId}/results` | Ad-hoc query execution against snapshots (run/poll/fetch results) is an imperative job with no durable state. |
| `getQueryStatus` | `GET /v1/projects/{projectId}/queries/{queryId}/status` | Ad-hoc query execution against snapshots (run/poll/fetch results) is an imperative job with no durable state. |
| `listResources` | `POST /v1/projects/{projectId}/resources` | Inventory listing of Eon-discovered cloud resources — pure inventory per triage policy. Revisit as a lookup data source if practitioners need resource IDs (inputs to eon_restore_job and exclusion resources) resolvable in HCL. |
| `takeSnapshot` | `POST /v1/projects/{projectId}/resources/{id}/take-snapshot` | Imperative one-shot action with no reconcilable state; lifecycle transitions belong inside the owning resource, not as standalone surface. |
| `reconnectRestoreAwsOrganizationalUnit` | `POST /v1/projects/{projectId}/restore-aws-organizational-units/{organizationalUnitId}/reconnect` | Imperative one-shot action with no reconcilable state; lifecycle transitions belong inside the owning resource, not as standalone surface. |
| `listRestoreJobs` | `POST /v1/projects/{projectId}/restore-jobs` | Job listing/status polling is imperative monitoring of runtime activity; there is no desired state to reconcile. |
| `runQuery` | `POST /v1/projects/{projectId}/snapshots/{snapshotId}/databases/query` | Ad-hoc query execution against snapshots (run/poll/fetch results) is an imperative job with no durable state. |
| `reconnectSourceAwsOrganizationalUnit` | `POST /v1/projects/{projectId}/source-aws-organizational-units/{organizationalUnitId}/reconnect` | Imperative one-shot action with no reconcilable state; lifecycle transitions belong inside the owning resource, not as standalone surface. |

## Covered operations

| Operation | Endpoint | Consumed by |
|---|---|---|
| `createIdpGroup` | `POST /v1/idp-groups` | eon_idp_group |
| `listIdpGroups` | `POST /v1/idp-groups/list` | eon_idp_groups |
| `deleteIdpGroup` | `DELETE /v1/idp-groups/{groupId}` | eon_idp_group |
| `getIdpGroup` | `GET /v1/idp-groups/{groupId}` | eon_idp_group |
| `updateIdpGroup` | `PUT /v1/idp-groups/{groupId}` | eon_idp_group |
| `createBackupPolicy` | `POST /v1/projects/{projectId}/backup-policies` | eon_backup_policy |
| `listBackupPolicies` | `POST /v1/projects/{projectId}/backup-policies/list` | eon_backup_policies |
| `deleteBackupPolicy` | `DELETE /v1/projects/{projectId}/backup-policies/{backupPolicyId}` | eon_backup_policy |
| `getBackupPolicy` | `GET /v1/projects/{projectId}/backup-policies/{backupPolicyId}` | eon_backup_policy |
| `updateBackupPolicy` | `PUT /v1/projects/{projectId}/backup-policies/{backupPolicyId}` | eon_backup_policy |
| `getResource` | `GET /v1/projects/{projectId}/resources/{id}` | eon_restore_job |
| `getObjectStoreScanMethod` | `GET /v1/projects/{projectId}/resources/{id}/object-store-scan-method` | eon_gcs_bucket_configuration |
| `updateObjectStoreScanMethod` | `PATCH /v1/projects/{projectId}/resources/{id}/object-store-scan-method` | eon_gcs_bucket_configuration |
| `restoreBigQueryDataset` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-bigquery-dataset` | eon_restore_job |
| `restoreBucket` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-bucket` | eon_restore_job |
| `restoreEbsVolume` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-ec2-ebs-volume` | eon_restore_job |
| `restoreEc2Instance` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-ec2-instance` | eon_restore_job |
| `restoreFiles` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-files` | eon_restore_job |
| `restoreGcpCloudSql` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-gcp-cloudsql` | eon_restore_job |
| `restoreGcpDisk` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-gcp-disk` | eon_restore_job |
| `restoreGcpVmInstance` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-gcp-vm-instance` | eon_restore_job |
| `restoreDatabase` | `POST /v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-rds-instance` | eon_restore_job |
| `excludeVolumeFromBackup` | `PATCH /v1/projects/{projectId}/resources/{id}/volumes/{volumeId}/exclude` | eon_volume_backup_exclusion |
| `cancelVolumeBackupExclusion` | `PATCH /v1/projects/{projectId}/resources/{id}/volumes/{volumeId}/include` | eon_volume_backup_exclusion |
| `connectRestoreAccount` | `POST /v1/projects/{projectId}/restore-accounts` | eon_restore_account |
| `listRestoreAccounts` | `POST /v1/projects/{projectId}/restore-accounts/list` | eon_restore_account, eon_restore_accounts |
| `DeleteRestoreAccountV1` | `DELETE /v1/projects/{projectId}/restore-accounts/{accountId}` | eon_restore_account |
| `getRestoreAccount` | `GET /v1/projects/{projectId}/restore-accounts/{accountId}` | eon_restore_account |
| `updateRestoreAccount` | `PATCH /v1/projects/{projectId}/restore-accounts/{accountId}` | eon_restore_account |
| `deleteRestoreAccountConnectivityConfig` | `DELETE /v1/projects/{projectId}/restore-accounts/{accountId}/connectivity-config` | eon_restore_account_connectivity_config |
| `getRestoreAccountConnectivityConfig` | `GET /v1/projects/{projectId}/restore-accounts/{accountId}/connectivity-config` | eon_restore_account_connectivity_config |
| `updateRestoreAccountConnectivityConfig` | `PUT /v1/projects/{projectId}/restore-accounts/{accountId}/connectivity-config` | eon_restore_account_connectivity_config |
| `DisconnectRestoreAccount` | `POST /v1/projects/{projectId}/restore-accounts/{accountId}/disconnect` | eon_restore_account |
| `ReconnectRestoreAccount` | `POST /v1/projects/{projectId}/restore-accounts/{accountId}/reconnect` | eon_restore_account |
| `connectRestoreAwsOrganizationalUnit` | `POST /v1/projects/{projectId}/restore-aws-organizational-units` | eon_restore_aws_organizational_unit |
| `listRestoreAwsOrganizationalUnits` | `POST /v1/projects/{projectId}/restore-aws-organizational-units/list` | eon_restore_aws_organizational_unit, eon_restore_aws_organizational_units |
| `disconnectRestoreAwsOrganizationalUnit` | `POST /v1/projects/{projectId}/restore-aws-organizational-units/{organizationalUnitId}/disconnect` | eon_restore_aws_organizational_unit |
| `getRestoreJob` | `GET /v1/projects/{projectId}/restore-jobs/{jobId}` | eon_restore_job |
| `getSnapshot` | `GET /v1/projects/{projectId}/snapshots/{id}` | eon_restore_job, eon_snapshot |
| `connectSourceAccount` | `POST /v1/projects/{projectId}/source-accounts` | eon_source_account |
| `listSourceAccounts` | `POST /v1/projects/{projectId}/source-accounts/list` | eon_source_account, eon_source_accounts |
| `deleteSourceAccount` | `DELETE /v1/projects/{projectId}/source-accounts/{accountId}` | eon_source_account |
| `getSourceAccount` | `GET /v1/projects/{projectId}/source-accounts/{accountId}` | eon_source_account |
| `updateSourceAccount` | `PATCH /v1/projects/{projectId}/source-accounts/{accountId}` | eon_source_account |
| `DisconnectSourceAccount` | `POST /v1/projects/{projectId}/source-accounts/{accountId}/disconnect` | eon_source_account |
| `ReconnectSourceAccount` | `POST /v1/projects/{projectId}/source-accounts/{accountId}/reconnect` | eon_source_account |
| `connectSourceAwsOrganizationalUnit` | `POST /v1/projects/{projectId}/source-aws-organizational-units` | eon_source_aws_organizational_unit |
| `listSourceAwsOrganizationalUnits` | `POST /v1/projects/{projectId}/source-aws-organizational-units/list` | eon_source_aws_organizational_unit, eon_source_aws_organizational_units |
| `disconnectSourceAwsOrganizationalUnit` | `POST /v1/projects/{projectId}/source-aws-organizational-units/{organizationalUnitId}/disconnect` | eon_source_aws_organizational_unit |
| `createVault` | `POST /v1/projects/{projectId}/vaults` | eon_vault |
| `listVaults` | `POST /v1/projects/{projectId}/vaults/list` | eon_vault, eon_vaults |
| `getVault` | `GET /v1/projects/{projectId}/vaults/{vaultId}` | eon_vault |
| `updateVault` | `PATCH /v1/projects/{projectId}/vaults/{vaultId}` | eon_vault |
| `createRole` | `POST /v1/roles` | eon_role |
| `listRoles` | `POST /v1/roles/list` | eon_roles |
| `deleteRole` | `DELETE /v1/roles/{roleId}` | eon_role |
| `getRole` | `GET /v1/roles/{roleId}` | eon_role |
| `updateRole` | `PUT /v1/roles/{roleId}` | eon_role |

## Covered internally

Consumed by provider plumbing rather than a specific resource or data source.

| Operation | Endpoint | Consumed by | Reason |
|---|---|---|---|
| `getAccessToken` | `POST /v1/token` | internal:client/token_refresher.go | Authentication, token issuance, and credential rotation are session/security actions, not declarative infrastructure. |


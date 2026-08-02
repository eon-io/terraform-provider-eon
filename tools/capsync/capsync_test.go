package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func op(id, method, path, tag string) SpecOperation {
	return SpecOperation{ID: id, Method: method, Path: path, Tag: tag, PathParams: pathParams(path)}
}

func TestClassify(t *testing.T) {
	t.Parallel()

	metricsOps := []SpecOperation{
		op("getSourceAccountMetricsConfig", "GET", "/v1/projects/{projectId}/source-accounts/{accountId}/metrics-config", "accounts"),
		op("enableSourceAccountMetricsConfig", "PUT", "/v1/projects/{projectId}/source-accounts/{accountId}/metrics-config", "accounts"),
		op("disableSourceAccountMetricsConfig", "DELETE", "/v1/projects/{projectId}/source-accounts/{accountId}/metrics-config", "accounts"),
	}
	postureOps := []SpecOperation{
		op("createBackupPostureControl", "POST", "/v1/projects/{projectId}/backup-posture-controls", "backupPostureControls"),
		op("getBackupPostureControl", "GET", "/v1/projects/{projectId}/backup-posture-controls/{controlId}", "backupPostureControls"),
		op("listBackupPostureControls", "POST", "/v1/projects/{projectId}/backup-posture-controls/list", "backupPostureControls"),
	}

	tests := []struct {
		name          string
		op            SpecOperation
		all           []SpecOperation
		wantClass     string
		wantName      string
		wantNeedsHint bool
	}{
		{
			name:      "auth is skipped",
			op:        op("getAccessToken", "POST", "/v1/token", "auth"),
			wantClass: ClassSkip,
		},
		{
			name:      "job polling is skipped",
			op:        op("getBackupJob", "GET", "/v1/projects/{projectId}/backup-jobs/{jobId}", "jobs"),
			wantClass: ClassSkip,
		},
		{
			name:      "one-shot snapshot trigger is skipped",
			op:        op("takeSnapshot", "POST", "/v1/projects/{projectId}/resources/{id}/take-snapshot", "backups"),
			wantClass: ClassSkip,
		},
		{
			name:      "reconnect action is skipped",
			op:        op("reconnectSourceAwsOrganizationalUnit", "POST", "/v1/projects/{projectId}/source-aws-organizational-units/{organizationalUnitId}/reconnect", "accounts"),
			wantClass: ClassSkip,
		},
		{
			name:      "agentic surface is skipped regardless of shape",
			op:        op("createAgent", "POST", "/v1/projects/{projectId}/agents", "agents"),
			wantClass: ClassSkip,
		},
		{
			name:      "billing reporting is skipped",
			op:        op("queryCostData", "POST", "/v1/cost-data", "billing"),
			wantClass: ClassSkip,
		},
		{
			name:      "singleton config with read+write is a resource",
			op:        metricsOps[1],
			all:       metricsOps,
			wantClass: ClassResource,
			wantName:  "eon_source_account_metrics_config",
		},
		{
			name:      "collection create with item read is a resource",
			op:        postureOps[0],
			all:       postureOps,
			wantClass: ClassResource,
			wantName:  "eon_backup_posture_control",
		},
		{
			name:      "POST list is a data source, not a create",
			op:        postureOps[2],
			all:       postureOps,
			wantClass: ClassDataSource,
			wantName:  "eon_backup_posture_controls",
		},
		{
			name:      "list by operation id prefix is a data source",
			op:        op("listResources", "POST", "/v1/projects/{projectId}/resources", "resources"),
			wantClass: ClassDataSource,
			wantName:  "eon_resources",
		},
		{
			name:      "exclude half of a toggle pair",
			op:        op("excludeResourceFromBackup", "PATCH", "/v1/projects/{projectId}/resources/{id}/exclude", "resources"),
			wantClass: ClassResource,
			wantName:  "eon_resource_backup_exclusion",
		},
		{
			name:      "include half lands on the same capability",
			op:        op("cancelResourceBackupExclusion", "PATCH", "/v1/projects/{projectId}/resources/{id}/include", "resources"),
			wantClass: ClassResource,
			wantName:  "eon_resource_backup_exclusion",
		},
		{
			name:      "hold and remove-hold land on the same capability",
			op:        op("removeSnapshotHold", "PATCH", "/v1/projects/{projectId}/snapshots/{id}/remove-hold", "snapshots"),
			wantClass: ClassResource,
			wantName:  "eon_snapshot_hold",
		},
		{
			name:      "restore trigger extends eon_restore_job",
			op:        op("restoreDynamoDBTable", "POST", "/v1/projects/{projectId}/resources/{id}/snapshots/{snapshotId}/restore-dynamo-db-table", "snapshots"),
			wantClass: ClassResource,
			wantName:  "eon_restore_job",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			all := tt.all
			if all == nil {
				all = []SpecOperation{tt.op}
			}
			got := classify(tt.op, all)
			assert.Equal(t, tt.wantClass, got.Classification, "classification")
			if tt.wantName != "" {
				assert.Equal(t, tt.wantName, got.TerraformName, "terraform name")
			}
			assert.NotEmpty(t, got.Reason, "every proposal must carry a reason")
		})
	}
}

func TestMatchRawCall(t *testing.T) {
	t.Parallel()
	ops := []SpecOperation{
		op("getObjectStoreScanMethod", "GET", "/v1/projects/{projectId}/resources/{id}/object-store-scan-method", "resources"),
		op("updateObjectStoreScanMethod", "PATCH", "/v1/projects/{projectId}/resources/{id}/object-store-scan-method", "resources"),
		op("excludeVolumeFromBackup", "PATCH", "/v1/projects/{projectId}/resources/{id}/volumes/{volumeId}/exclude", "resources"),
	}

	pattern := strings.Split("v1/projects/%s/resources/%s/object-store-scan-method", "/")
	assert.Equal(t, "updateObjectStoreScanMethod", matchRawCall(rawCall{method: "PATCH", urlPattern: pattern}, ops))
	assert.Equal(t, "getObjectStoreScanMethod", matchRawCall(rawCall{method: "GET", urlPattern: pattern}, ops))

	exclude := strings.Split("v1/projects/%s/resources/%s/volumes/%s/exclude", "/")
	assert.Equal(t, "excludeVolumeFromBackup", matchRawCall(rawCall{method: "PATCH", urlPattern: exclude}, ops))

	assert.Empty(t, matchRawCall(rawCall{method: "DELETE", urlPattern: pattern}, ops), "method mismatch should not match")
}

func TestParseSpecFile(t *testing.T) {
	t.Parallel()
	spec := `
openapi: 3.0.0
paths:
  /v1/projects/{projectId}/vaults:
    post:
      operationId: createVault
      summary: Create Vault
      tags: [vaults]
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateVaultRequest'
      responses:
        "201":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/BackupVault'
  /v1/projects/{projectId}/vaults/{vaultId}:
    get:
      operationId: getVault
      tags: [vaults]
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/BackupVault'
`
	path := filepath.Join(t.TempDir(), "openapi.yaml")
	require.NoError(t, os.WriteFile(path, []byte(spec), 0o644))

	ops, err := parseSpecFile(path)
	require.NoError(t, err)
	require.Len(t, ops, 2)

	create := ops[0]
	assert.Equal(t, "createVault", create.ID)
	assert.Equal(t, "POST", create.Method)
	assert.Equal(t, "CreateVaultRequest", create.RequestModel)
	assert.Equal(t, "BackupVault", create.ResponseModel)
	assert.Equal(t, []string{"projectId"}, create.PathParams)
	assert.Equal(t, "VaultsAPI", create.SDKService())
	assert.Equal(t, "CreateVault", create.SDKMethod())

	get := ops[1]
	assert.Equal(t, "getVault", get.ID)
	assert.Equal(t, []string{"projectId", "vaultId"}, get.PathParams)
}

// TestExtractCoverage_RealRepo runs the coverage scan against this repository
// and the SDK version pinned in go.mod, keeping the analyzer honest about the
// wrapper patterns it claims to understand.
func TestExtractCoverage_RealRepo(t *testing.T) {
	t.Parallel()
	providerDir := "../.."

	_, ops, err := loadSDKSpec(providerDir, "")
	require.NoError(t, err, "loading the go.mod SDK spec (requires module cache or network)")
	require.NotEmpty(t, ops)

	coverage, err := extractCoverage(providerDir, ops)
	require.NoError(t, err)

	// SDK-typed calls, two hops: resource -> EonClient wrapper -> SDK.
	assert.Contains(t, coverage.TerraformConsumers("createBackupPolicy"), "eon_backup_policy")
	assert.Contains(t, coverage.TerraformConsumers("listVaults"), "eon_vaults")

	// Raw fmt.Sprintf HTTP calls that bypass the generated SDK.
	assert.Contains(t, coverage.TerraformConsumers("excludeVolumeFromBackup"), "eon_volume_backup_exclusion")
	assert.Contains(t, coverage.TerraformConsumers("getObjectStoreScanMethod"), "eon_gcs_bucket_configuration")

	// Transitive wrapper call: WaitForRestoreJobCompletion -> GetRestoreJob.
	assert.Contains(t, coverage.TerraformConsumers("getRestoreJob"), "eon_restore_job")

	// Token refresh happens inside internal/client, not via any resource.
	assert.Empty(t, coverage.TerraformConsumers("getAccessToken"))
	assert.NotEmpty(t, coverage.Consumers["getAccessToken"], "token refresh should be detected as internal usage")
}

// TestManifestOverridesSurviveRuns is the contract behind reviewer overrides:
// once an operation is in the manifest, capsync must never change its
// classification or reason, and skipped items must never be re-proposed.
func TestManifestOverridesSurviveRuns(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")

	ops := []SpecOperation{
		op("frobnicateWidget", "POST", "/v1/widgets/{widgetId}/frobnicate", "widgets"),
		op("createWidget", "POST", "/v1/widgets", "widgets"),
		op("getWidget", "GET", "/v1/widgets/{widgetId}", "widgets"),
	}

	// First run: manifest is empty, everything gets seeded.
	manifest, err := loadManifest(path)
	require.NoError(t, err)
	report := buildReport("v1.0.0", ops, Coverage{Consumers: map[string][]string{}}, manifest)
	require.NoError(t, saveManifest(path, manifest, report))

	// Reviewer overrides: widgets must never become a resource.
	manifest, err = loadManifest(path)
	require.NoError(t, err)
	manifest.Operations["createWidget"].Classification = ClassSkip
	manifest.Operations["createWidget"].Reason = "reviewer says no"
	report = buildReport("v1.1.0", ops, Coverage{Consumers: map[string][]string{}}, manifest)
	require.NoError(t, saveManifest(path, manifest, report))

	// Second run after the override: the decision must stick.
	manifest, err = loadManifest(path)
	require.NoError(t, err)
	entry := manifest.Operations["createWidget"]
	assert.Equal(t, ClassSkip, entry.Classification)
	assert.Equal(t, "reviewer says no", entry.Reason)
	assert.Equal(t, "v1.0.0", entry.FirstSeen, "first_seen must not move on later runs")

	report = buildReport("v1.2.0", ops, Coverage{Consumers: map[string][]string{}}, manifest)
	for _, g := range report.Gaps {
		assert.NotContains(t, g.OperationIDs, "createWidget", "skipped operations must never be re-proposed")
	}

	// Operations that disappear from the SDK are reported, not silently dropped.
	report = buildReport("v1.3.0", ops[:2], Coverage{Consumers: map[string][]string{}}, manifest)
	assert.Contains(t, report.Removed, "getWidget")
}

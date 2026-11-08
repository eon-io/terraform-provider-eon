package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func googleWorkspacePlanType(t *testing.T) types.ObjectType {
	t.Helper()

	planType, diags := nestedObjectType(policySchemaType(t), "backup_plan")
	require.False(t, diags.HasError())
	googleWorkspaceType, diags := nestedObjectType(planType, "google_workspace_plan")
	require.False(t, diags.HasError())
	return googleWorkspaceType
}

func googleWorkspacePlanObject(t *testing.T, vaultID string, retentionDays int64) types.Object {
	t.Helper()

	object, diags := objectValue(googleWorkspacePlanType(t), map[string]attr.Value{
		"vault_id":       types.StringValue(vaultID),
		"retention_days": types.Int64Value(retentionDays),
	})
	require.False(t, diags.HasError())
	return object
}

func googleWorkspacePolicy(workspaceID, vaultID string, retentionDays int32) *externalEonSdkAPI.BackupPolicy {
	selector := externalEonSdkAPI.NewBackupPolicyResourceSelector(externalEonSdkAPI.RESOURCE_SELECTOR_MODE_NONE)
	selector.SetResourceInclusionOverride([]string{workspaceID})
	plan := externalEonSdkAPI.NewBackupPolicyPlan(externalEonSdkAPI.BACKUP_POLICY_TYPE_GOOGLE_WORKSPACE)
	plan.SetGoogleWorkspacePlan(*externalEonSdkAPI.NewGoogleWorkspaceBackupPolicyPlan(vaultID, retentionDays))
	return externalEonSdkAPI.NewBackupPolicy("policy-gws", "workspace nightly", true, *selector, *plan)
}

func TestBuildGoogleWorkspacePlan(t *testing.T) {
	t.Parallel()

	plan, diags := buildGoogleWorkspacePlan(context.Background(),
		map[string]attr.Value{"google_workspace_plan": googleWorkspacePlanObject(t, "vault-gcp", 30)})
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())

	assert.Equal(t, "vault-gcp", plan.VaultId)
	assert.Equal(t, int32(30), plan.BackupRetentionDays)
}

func TestBuildGoogleWorkspacePlan_MissingPlan(t *testing.T) {
	t.Parallel()

	_, diags := buildGoogleWorkspacePlan(context.Background(),
		map[string]attr.Value{"google_workspace_plan": types.ObjectNull(googleWorkspacePlanType(t).AttrTypes)})
	require.True(t, diags.HasError())
	assert.Contains(t, diags.Errors()[0].Detail(), "google_workspace_plan is required")
}

func TestGoogleWorkspacePlanValidator(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		retention int64
		wantError bool
	}{
		{name: "minimum", retention: 1},
		{name: "typical", retention: 30},
		{name: "zero", retention: 0, wantError: true},
		{name: "negative", retention: -5, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resp := &validator.ObjectResponse{}
			googleWorkspacePlanValidator{}.ValidateObject(context.Background(), validator.ObjectRequest{
				Path:        path.Root("backup_plan").AtName("google_workspace_plan"),
				ConfigValue: googleWorkspacePlanObject(t, "vault-gcp", tc.retention),
			}, resp)
			assert.Equal(t, tc.wantError, resp.Diagnostics.HasError(), "diagnostics: %v", resp.Diagnostics.Errors())
		})
	}
}

func TestGoogleWorkspacePlanValidator_SkipsUnknownAndNull(t *testing.T) {
	t.Parallel()

	attrTypes := googleWorkspacePlanType(t).AttrTypes
	for name, value := range map[string]types.Object{
		"null":    types.ObjectNull(attrTypes),
		"unknown": types.ObjectUnknown(attrTypes),
	} {
		resp := &validator.ObjectResponse{}
		googleWorkspacePlanValidator{}.ValidateObject(context.Background(), validator.ObjectRequest{
			Path:        path.Root("backup_plan").AtName("google_workspace_plan"),
			ConfigValue: value,
		}, resp)
		assert.False(t, resp.Diagnostics.HasError(), "%s object should not be validated", name)
	}
}

func TestFlattenBackupPolicy_GoogleWorkspacePlan(t *testing.T) {
	t.Parallel()

	var data BackupPolicyResourceModel
	diags := flattenBackupPolicy(context.Background(), googleWorkspacePolicy("ws-1", "vault-gcp", 30), policySchemaType(t), &data)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())

	assert.Equal(t, types.StringValue("GOOGLE_WORKSPACE"), nestedValue(t, data.BackupPlan, "backup_policy_type"))
	assert.Equal(t, types.StringValue("vault-gcp"), nestedValue(t, data.BackupPlan, "google_workspace_plan", "vault_id"))
	assert.Equal(t, types.Int64Value(30), nestedValue(t, data.BackupPlan, "google_workspace_plan", "retention_days"))
	for _, other := range []string{"standard_plan", "high_frequency_plan", "aws_native_pitr_plan", "aws_native_standard_plan"} {
		assert.True(t, nestedValue(t, data.BackupPlan, other).IsNull(), "%s should be null", other)
	}

	assert.Equal(t, types.StringValue("NONE"), nestedValue(t, data.ResourceSelector, "resource_selection_mode"))
	assert.Equal(t, types.ListValueMust(types.StringType, []attr.Value{types.StringValue("ws-1")}),
		nestedValue(t, data.ResourceSelector, "resource_inclusion_override"))
	assert.True(t, nestedValue(t, data.ResourceSelector, "resource_exclusion_override").IsNull())
	assert.True(t, nestedValue(t, data.ResourceSelector, "expression").IsNull())
}

func TestFlattenBackupPolicy_GoogleWorkspaceMissingPlan(t *testing.T) {
	t.Parallel()

	policy := googleWorkspacePolicy("ws-1", "vault-gcp", 30)
	policy.BackupPlan.GoogleWorkspacePlan.Unset()

	var data BackupPolicyResourceModel
	diags := flattenBackupPolicy(context.Background(), policy, policySchemaType(t), &data)
	require.True(t, diags.HasError())
	assert.Contains(t, diags.Errors()[0].Detail(), "without a Google Workspace plan")
}

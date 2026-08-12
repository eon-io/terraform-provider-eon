package provider

import (
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelToEnableSourceMetricsRequest(t *testing.T) {
	t.Parallel()

	t.Run("with aws region", func(t *testing.T) {
		t.Parallel()
		data := &SourceAccountMetricsConfigResourceModel{
			SourceAccountId: types.StringValue("acct-1"),
			Aws: &awsAccountMetricsDestModel{
				Region: types.StringValue("eu-west-1"),
			},
		}
		req := modelToEnableSourceMetricsRequest(data)
		require.True(t, req.HasAws())
		aws := req.GetAws()
		assert.Equal(t, "eu-west-1", aws.GetRegion())
	})

	t.Run("without aws block", func(t *testing.T) {
		t.Parallel()
		data := &SourceAccountMetricsConfigResourceModel{
			SourceAccountId: types.StringValue("acct-1"),
		}
		req := modelToEnableSourceMetricsRequest(data)
		assert.False(t, req.HasAws())
	})
}

func TestSourceMetricsConfigToModel(t *testing.T) {
	t.Parallel()

	destination := externalEonSdkAPI.NewAccountMetricsDestination()
	aws := externalEonSdkAPI.NewAwsAccountMetricsDestination()
	aws.SetRegion("ap-southeast-1")
	destination.SetAws(*aws)
	config := externalEonSdkAPI.NewSourceAccountMetricsConfig("acct-1", true, *destination)

	data := &SourceAccountMetricsConfigResourceModel{}
	diags := sourceMetricsConfigToModel(config, data)
	require.False(t, diags.HasError())
	require.NotNil(t, data.Aws)
	assert.Equal(t, "ap-southeast-1", data.Aws.Region.ValueString())
}

func TestActionApprovalRuleCreateRequest(t *testing.T) {
	t.Parallel()

	t.Run("minimal", func(t *testing.T) {
		t.Parallel()
		data := &ActionApprovalRuleResourceModel{
			Operation:               types.StringValue("ADD_RESTORE_ACCOUNT"),
			RequiredApprovals:       types.Int64Value(1),
			ApprovalWindowHours:     types.Int64Value(24),
			ExecutionWindowHours:    types.Int64Value(12),
			Description:             types.StringNull(),
			ResourceSelector:        types.ObjectNull(actionApprovalRuleSchema.Attributes["resource_selector"].GetType().(basetypes.ObjectType).AttrTypes),
			ApproverIdpId:           types.StringNull(),
			ApproverProviderGroupId: types.StringNull(),
			ExemptApiCredentials:    types.BoolValue(false),
		}
		req, diags := actionApprovalRuleCreateRequest(t.Context(), data)
		require.False(t, diags.HasError())
		assert.Equal(t, externalEonSdkAPI.ACTION_APPROVAL_OPERATION_ADD_RESTORE_ACCOUNT, req.GetOperation())
		assert.Equal(t, int32(1), req.GetRequiredApprovals())
		assert.Equal(t, int32(24), req.GetApprovalWindowHours())
		assert.Equal(t, int32(12), req.GetExecutionWindowHours())
		assert.False(t, req.HasResourceSelector())
	})

	t.Run("with description and exempt", func(t *testing.T) {
		t.Parallel()
		data := &ActionApprovalRuleResourceModel{
			Operation:               types.StringValue("RESTORE_RESOURCE"),
			RequiredApprovals:       types.Int64Value(2),
			ApprovalWindowHours:     types.Int64Value(48),
			ExecutionWindowHours:    types.Int64Value(6),
			Description:             types.StringValue("protect restores"),
			ResourceSelector:        types.ObjectNull(actionApprovalRuleSchema.Attributes["resource_selector"].GetType().(basetypes.ObjectType).AttrTypes),
			ApproverIdpId:           types.StringValue("idp-1"),
			ApproverProviderGroupId: types.StringValue("group-1"),
			ExemptApiCredentials:    types.BoolValue(true),
		}
		req, diags := actionApprovalRuleCreateRequest(t.Context(), data)
		require.False(t, diags.HasError())
		assert.Equal(t, "protect restores", req.GetDescription())
		assert.Equal(t, "idp-1", req.GetApproverIdpId())
		assert.Equal(t, "group-1", req.GetApproverProviderGroupId())
		assert.True(t, req.GetExemptApiCredentials())
	})
}

func TestActionApprovalRuleToState(t *testing.T) {
	t.Parallel()

	rule := externalEonSdkAPI.NewActionApprovalRule(
		"rule-1",
		"project-1",
		externalEonSdkAPI.ACTION_APPROVAL_OPERATION_CREATE_BACKUP_POLICY,
		2,
		24,
		12,
	)
	rule.SetDescription("policy create approval")
	rule.SetExemptApiCredentials(true)

	data := &ActionApprovalRuleResourceModel{}
	diags := actionApprovalRuleToState(t.Context(), rule, data)
	require.False(t, diags.HasError())
	assert.Equal(t, "rule-1", data.Id.ValueString())
	assert.Equal(t, "CREATE_BACKUP_POLICY", data.Operation.ValueString())
	assert.Equal(t, int64(2), data.RequiredApprovals.ValueInt64())
	assert.Equal(t, int64(24), data.ApprovalWindowHours.ValueInt64())
	assert.Equal(t, int64(12), data.ExecutionWindowHours.ValueInt64())
	assert.Equal(t, "policy create approval", data.Description.ValueString())
	assert.True(t, data.ExemptApiCredentials.ValueBool())
	assert.True(t, data.ResourceSelector.IsNull())
}

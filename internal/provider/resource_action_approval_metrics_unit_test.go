package provider

import (
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	t.Run("success with required fields", func(t *testing.T) {
		t.Parallel()
		data := &ActionApprovalRuleResourceModel{
			Operation:            types.StringValue("RESTORE_RESOURCE"),
			RequiredApprovals:    types.Int64Value(2),
			ApprovalWindowHours:  types.Int64Value(24),
			ExecutionWindowHours: types.Int64Value(48),
			Description:          types.StringValue("test rule"),
			ExemptApiCredentials: types.BoolValue(true),
			ResourceSelector:     types.ObjectNull(actionApprovalRuleSchema.Attributes["resource_selector"].GetType().(types.ObjectType).AttrTypes),
		}

		req, diags := actionApprovalRuleCreateRequest(t.Context(), data)
		require.False(t, diags.HasError())
		assert.Equal(t, externalEonSdkAPI.ACTION_APPROVAL_OPERATION_RESTORE_RESOURCE, req.GetOperation())
		assert.Equal(t, int32(2), req.GetRequiredApprovals())
		assert.Equal(t, int32(24), req.GetApprovalWindowHours())
		assert.Equal(t, int32(48), req.GetExecutionWindowHours())
		assert.Equal(t, "test rule", req.GetDescription())
		assert.True(t, req.GetExemptApiCredentials())
	})

	t.Run("invalid operation", func(t *testing.T) {
		t.Parallel()
		data := &ActionApprovalRuleResourceModel{
			Operation:            types.StringValue("NOT_A_REAL_OP"),
			ApprovalWindowHours:  types.Int64Value(24),
			ExecutionWindowHours: types.Int64Value(48),
			ResourceSelector:     types.ObjectNull(actionApprovalRuleSchema.Attributes["resource_selector"].GetType().(types.ObjectType).AttrTypes),
		}

		_, diags := actionApprovalRuleCreateRequest(t.Context(), data)
		require.True(t, diags.HasError())
	})

	t.Run("overflow approval window", func(t *testing.T) {
		t.Parallel()
		data := &ActionApprovalRuleResourceModel{
			Operation:            types.StringValue("RESTORE_RESOURCE"),
			ApprovalWindowHours:  types.Int64Value(1 << 33),
			ExecutionWindowHours: types.Int64Value(48),
			ResourceSelector:     types.ObjectNull(actionApprovalRuleSchema.Attributes["resource_selector"].GetType().(types.ObjectType).AttrTypes),
		}

		_, diags := actionApprovalRuleCreateRequest(t.Context(), data)
		require.True(t, diags.HasError())
	})
}

func TestActionApprovalRuleToState(t *testing.T) {
	t.Parallel()

	rule := externalEonSdkAPI.NewActionApprovalRule(
		"rule-1",
		"project-1",
		externalEonSdkAPI.ACTION_APPROVAL_OPERATION_CREATE_BACKUP_POLICY,
		1,
		12,
		24,
	)
	rule.SetDescription("desc")
	rule.SetExemptApiCredentials(true)
	selector := externalEonSdkAPI.NewActionApprovalRuleResourceSelector(externalEonSdkAPI.RESOURCE_SELECTOR_MODE_ALL)
	rule.SetResourceSelector(*selector)

	data := &ActionApprovalRuleResourceModel{}
	diags := actionApprovalRuleToState(t.Context(), rule, data)
	require.False(t, diags.HasError())
	assert.Equal(t, "rule-1", data.Id.ValueString())
	assert.Equal(t, "CREATE_BACKUP_POLICY", data.Operation.ValueString())
	assert.Equal(t, int64(1), data.RequiredApprovals.ValueInt64())
	assert.Equal(t, int64(12), data.ApprovalWindowHours.ValueInt64())
	assert.Equal(t, int64(24), data.ExecutionWindowHours.ValueInt64())
	assert.Equal(t, "desc", data.Description.ValueString())
	assert.True(t, data.ExemptApiCredentials.ValueBool())
	assert.False(t, data.ResourceSelector.IsNull())
}

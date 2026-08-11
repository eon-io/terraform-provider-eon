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

	tests := []struct {
		name    string
		data    ActionApprovalRuleResourceModel
		wantErr bool
	}{
		{
			name: "success minimal",
			data: ActionApprovalRuleResourceModel{
				Operation:               types.StringValue("ADD_RESTORE_ACCOUNT"),
				RequiredApprovals:       types.Int64Value(1),
				ApprovalWindowHours:     types.Int64Value(24),
				ExecutionWindowHours:    types.Int64Value(4),
				Description:             types.StringValue("desc"),
				ResourceSelector:        types.ObjectNull(actionApprovalRuleSchema.Attributes["resource_selector"].GetType().(types.ObjectType).AttrTypes),
				ApproverIdpId:           types.StringNull(),
				ApproverProviderGroupId: types.StringNull(),
				ExemptApiCredentials:    types.BoolNull(),
			},
		},
		{
			name: "invalid window",
			data: ActionApprovalRuleResourceModel{
				Operation:            types.StringValue("ADD_RESTORE_ACCOUNT"),
				RequiredApprovals:    types.Int64Value(1),
				ApprovalWindowHours:  types.Int64Value(int64(1) << 40),
				ExecutionWindowHours: types.Int64Value(4),
				ResourceSelector:     types.ObjectNull(actionApprovalRuleSchema.Attributes["resource_selector"].GetType().(types.ObjectType).AttrTypes),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req, diags := actionApprovalRuleCreateRequest(t.Context(), &tt.data)
			if tt.wantErr {
				assert.True(t, diags.HasError())
				return
			}
			require.False(t, diags.HasError())
			assert.Equal(t, externalEonSdkAPI.ACTION_APPROVAL_OPERATION_ADD_RESTORE_ACCOUNT, req.GetOperation())
			assert.Equal(t, int32(24), req.GetApprovalWindowHours())
			assert.Equal(t, int32(4), req.GetExecutionWindowHours())
			assert.Equal(t, "desc", req.GetDescription())
		})
	}
}

func TestActionApprovalRuleToState(t *testing.T) {
	t.Parallel()

	rule := externalEonSdkAPI.NewActionApprovalRule(
		"rule-1",
		"project-1",
		externalEonSdkAPI.ACTION_APPROVAL_OPERATION_ADD_RESTORE_ACCOUNT,
		2,
		48,
		8,
	)
	rule.SetDescription("from api")
	rule.SetExemptApiCredentials(true)

	data := &ActionApprovalRuleResourceModel{}
	diags := actionApprovalRuleToState(t.Context(), rule, data)
	require.False(t, diags.HasError())
	assert.Equal(t, "rule-1", data.Id.ValueString())
	assert.Equal(t, "ADD_RESTORE_ACCOUNT", data.Operation.ValueString())
	assert.Equal(t, int64(2), data.RequiredApprovals.ValueInt64())
	assert.Equal(t, int64(48), data.ApprovalWindowHours.ValueInt64())
	assert.Equal(t, int64(8), data.ExecutionWindowHours.ValueInt64())
	assert.Equal(t, "from api", data.Description.ValueString())
	assert.True(t, data.ExemptApiCredentials.ValueBool())
	assert.True(t, data.ResourceSelector.IsNull())
}

func TestActionApprovalRuleUpdateRequest(t *testing.T) {
	t.Parallel()

	data := &ActionApprovalRuleResourceModel{
		Operation:               types.StringValue("ADD_RESTORE_ACCOUNT"),
		RequiredApprovals:       types.Int64Value(3),
		ApprovalWindowHours:     types.Int64Value(12),
		ExecutionWindowHours:    types.Int64Value(2),
		Description:             types.StringValue("updated"),
		ResourceSelector:        types.ObjectNull(actionApprovalRuleSchema.Attributes["resource_selector"].GetType().(types.ObjectType).AttrTypes),
		ApproverIdpId:           types.StringNull(),
		ApproverProviderGroupId: types.StringNull(),
		ExemptApiCredentials:    types.BoolValue(false),
	}

	req, diags := actionApprovalRuleUpdateRequest(t.Context(), data)
	require.False(t, diags.HasError())
	assert.Equal(t, int32(3), req.GetRequiredApprovals())
	assert.Equal(t, int32(12), req.GetApprovalWindowHours())
	assert.Equal(t, int32(2), req.GetExecutionWindowHours())
	assert.Equal(t, "updated", req.GetDescription())
	assert.False(t, req.GetExemptApiCredentials())
}

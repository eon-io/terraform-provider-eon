package provider

import (
	"context"
	"encoding/json"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const actionApprovalRuleAPIResponse = `{
  "id": "rule-1",
  "projectId": "project-1",
  "operation": "RESTORE_RESOURCE",
  "requiredApprovals": 2,
  "approvalWindowHours": 24,
  "executionWindowHours": 12,
  "description": "require restore approval",
  "exemptApiCredentials": true,
  "resourceSelector": {
    "resourceSelectionMode": "ALL"
  }
}`

func TestActionApprovalRuleStateRoundTrip(t *testing.T) {
	t.Parallel()

	var rule externalEonSdkAPI.ActionApprovalRule
	require.NoError(t, json.Unmarshal([]byte(actionApprovalRuleAPIResponse), &rule))

	var state ActionApprovalRuleResourceModel
	diags := actionApprovalRuleToState(context.Background(), &rule, &state)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())

	assert.Equal(t, "rule-1", state.Id.ValueString())
	assert.Equal(t, "RESTORE_RESOURCE", state.Operation.ValueString())
	assert.Equal(t, int64(2), state.RequiredApprovals.ValueInt64())
	assert.Equal(t, int64(24), state.ApprovalWindowHours.ValueInt64())
	assert.Equal(t, int64(12), state.ExecutionWindowHours.ValueInt64())
	assert.Equal(t, "require restore approval", state.Description.ValueString())
	assert.True(t, state.ExemptApiCredentials.ValueBool())
	require.False(t, state.ResourceSelector.IsNull())

	createReq, diags := actionApprovalRuleCreateRequest(context.Background(), &state)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
	assert.Equal(t, externalEonSdkAPI.ACTION_APPROVAL_OPERATION_RESTORE_RESOURCE, createReq.GetOperation())
	assert.Equal(t, int32(2), createReq.GetRequiredApprovals())
	assert.Equal(t, int32(24), createReq.GetApprovalWindowHours())
	assert.Equal(t, int32(12), createReq.GetExecutionWindowHours())
	assert.True(t, createReq.GetExemptApiCredentials())
	require.True(t, createReq.HasResourceSelector())
	selector := createReq.GetResourceSelector()
	assert.Equal(t, externalEonSdkAPI.RESOURCE_SELECTOR_MODE_ALL, selector.GetResourceSelectionMode())
}

func TestActionApprovalRuleCreateRequestSafeIntConversion(t *testing.T) {
	t.Parallel()

	selectorType := actionApprovalRuleSchema.Attributes["resource_selector"].GetType()
	nullSelector, diags := apiValueToTF(nil, selectorType)
	require.False(t, diags.HasError())

	data := &ActionApprovalRuleResourceModel{
		Operation:            types.StringValue("RESTORE_RESOURCE"),
		RequiredApprovals:    types.Int64Value(1 << 40),
		ApprovalWindowHours:  types.Int64Value(24),
		ExecutionWindowHours: types.Int64Value(12),
		ExemptApiCredentials: types.BoolValue(false),
		ResourceSelector:     nullSelector.(types.Object),
	}

	_, diags = actionApprovalRuleCreateRequest(context.Background(), data)
	require.True(t, diags.HasError())
}

func TestActionApprovalRuleUpdateRequestClearsApprover(t *testing.T) {
	t.Parallel()

	selectorType := actionApprovalRuleSchema.Attributes["resource_selector"].GetType()
	nullSelector, diags := apiValueToTF(nil, selectorType)
	require.False(t, diags.HasError())

	data := &ActionApprovalRuleResourceModel{
		Operation:               types.StringValue("RESTORE_RESOURCE"),
		RequiredApprovals:       types.Int64Value(1),
		ApprovalWindowHours:     types.Int64Value(24),
		ExecutionWindowHours:    types.Int64Value(12),
		ExemptApiCredentials:    types.BoolValue(false),
		ApproverIdpId:           types.StringNull(),
		ApproverProviderGroupId: types.StringNull(),
		ResourceSelector:        nullSelector.(types.Object),
	}

	updateReq, diags := actionApprovalRuleUpdateRequest(context.Background(), data)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
	assert.True(t, updateReq.ApproverIdpId.IsSet())
	assert.Nil(t, updateReq.ApproverIdpId.Get())
	assert.True(t, updateReq.ApproverProviderGroupId.IsSet())
	assert.Nil(t, updateReq.ApproverProviderGroupId.Get())
}

func TestActionApprovalRuleToStateWithoutOptionalFields(t *testing.T) {
	t.Parallel()

	rule := externalEonSdkAPI.NewActionApprovalRule(
		"rule-2",
		"project-1",
		externalEonSdkAPI.ACTION_APPROVAL_OPERATION_ADD_RESTORE_ACCOUNT,
		1,
		8,
		4,
	)

	var state ActionApprovalRuleResourceModel
	diags := actionApprovalRuleToState(context.Background(), rule, &state)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
	assert.True(t, state.Description.IsNull())
	assert.True(t, state.ApproverIdpId.IsNull())
	assert.True(t, state.ResourceSelector.IsNull())
	assert.False(t, state.ExemptApiCredentials.ValueBool())
}

package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupPostureControlResource_Unit(t *testing.T) {
	t.Parallel()

	r := NewBackupPostureControlResource()
	assert.NotNil(t, r, "NewBackupPostureControlResource should return a resource")
}

func TestBackupPostureControlsDataSource_Unit(t *testing.T) {
	t.Parallel()

	d := NewBackupPostureControlsDataSource()
	assert.NotNil(t, d, "NewBackupPostureControlsDataSource should return a data source")
}

func newTestPostureControlRequest(name string) *externalEonSdkAPI.CreateBackupPostureControlRequest {
	selector := externalEonSdkAPI.NewBackupPostureControlResourceSelector(externalEonSdkAPI.ResourceSelectorMode("ALL"))
	rules := externalEonSdkAPI.NewBackupPostureControlRules()
	rules.SetCrossRegion(true)
	return externalEonSdkAPI.NewCreateBackupPostureControlRequest(
		name,
		externalEonSdkAPI.Severity("HIGH"),
		*externalEonSdkAPI.NewNullableBackupPostureControlResourceSelector(selector),
		*externalEonSdkAPI.NewNullableBackupPostureControlRules(rules),
	)
}

func TestBackupPostureControlResource_CreateWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		shouldFail bool
	}{
		{name: "successful create", shouldFail: false},
		{name: "failed create", shouldFail: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailPostureControlCreate = tt.shouldFail

			control, err := mockClient.CreateBackupPostureControl(context.Background(), *newTestPostureControlRequest("enforce-retention"))

			if tt.shouldFail {
				assert.Error(t, err, "Create should fail when the client errors")
				assert.Nil(t, control, "No control should be returned on failure")
			} else {
				assert.NoError(t, err, "Create should succeed")
				assert.NotEmpty(t, control.Id, "Created control should have an ID")
				assert.Equal(t, "enforce-retention", control.Name, "Created control should keep the requested name")
				assert.Equal(t, externalEonSdkAPI.Severity("HIGH"), control.Severity, "Created control should keep the requested severity")
			}
			assert.Equal(t, 1, mockClient.PostureControlCreateCalls, "Should have made one create call")
		})
	}
}

func TestBackupPostureControlResource_LifecycleWithMockClient(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	created, err := mockClient.CreateBackupPostureControl(context.Background(), *newTestPostureControlRequest("lifecycle"))
	require.NoError(t, err, "Create should succeed")

	read, err := mockClient.GetBackupPostureControl(context.Background(), created.Id)
	require.NoError(t, err, "Get should succeed")
	assert.Equal(t, created.Id, read.Id, "Get should return the created control")

	updateReq := externalEonSdkAPI.NewUpdateBackupPostureControlRequest(
		"lifecycle-renamed",
		externalEonSdkAPI.Severity("LOW"),
		created.ResourceSelector,
		created.Rules,
	)
	updated, err := mockClient.UpdateBackupPostureControl(context.Background(), created.Id, *updateReq)
	require.NoError(t, err, "Update should succeed")
	assert.Equal(t, "lifecycle-renamed", updated.Name, "Update should change the name")
	assert.Equal(t, externalEonSdkAPI.Severity("LOW"), updated.Severity, "Update should change the severity")

	controls, err := mockClient.ListBackupPostureControls(context.Background())
	require.NoError(t, err, "List should succeed")
	assert.Len(t, controls, 1, "List should return the single control")

	require.NoError(t, mockClient.DeleteBackupPostureControl(context.Background(), created.Id), "Delete should succeed")

	_, err = mockClient.GetBackupPostureControl(context.Background(), created.Id)
	assert.Error(t, err, "Get after delete should fail")
}

func TestPolicyExpressionToPostureExpression(t *testing.T) {
	t.Parallel()

	src := externalEonSdkAPI.NewBackupPolicyExpression()
	envCondition := externalEonSdkAPI.NewEnvironmentCondition(
		externalEonSdkAPI.ScalarOperators("IN"),
		[]externalEonSdkAPI.Environment{externalEonSdkAPI.Environment("PROD")},
	)

	operand := externalEonSdkAPI.NewBackupPolicyExpression()
	operand.SetEnvironment(*envCondition)

	tagOperand := externalEonSdkAPI.NewBackupPolicyExpression()
	tagOperand.SetTagKeys(*externalEonSdkAPI.NewTagKeysCondition(
		externalEonSdkAPI.ListOperators("CONTAINS_ANY_OF"),
		[]string{"team"},
	))

	src.SetGroup(*externalEonSdkAPI.NewBackupPolicyGroupCondition(
		externalEonSdkAPI.LogicalOperator("AND"),
		[]externalEonSdkAPI.BackupPolicyExpression{*operand, *tagOperand},
	))

	dst := policyExpressionToPostureExpression(src)
	require.NotNil(t, dst, "Conversion should not return nil")

	group := dst.Group.Get()
	require.NotNil(t, group, "Group should be converted")
	assert.Equal(t, externalEonSdkAPI.LogicalOperator("AND"), group.Operator, "Group operator should be preserved")
	require.Len(t, group.Operands, 2, "Both operands should be converted")

	convertedEnv := group.Operands[0].Environment.Get()
	require.NotNil(t, convertedEnv, "Environment condition should be preserved in the first operand")
	assert.Equal(t, *envCondition, *convertedEnv, "Environment condition should be copied verbatim")

	convertedTags := group.Operands[1].TagKeys.Get()
	require.NotNil(t, convertedTags, "Tag keys condition should be preserved in the second operand")
	assert.Equal(t, []string{"team"}, convertedTags.TagKeys, "Tag keys should be copied verbatim")

	assert.Nil(t, policyExpressionToPostureExpression(nil), "Nil input should convert to nil")
}

func postureControlRulesAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"minimum_retention": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
			"frequency":              types.StringType,
			"minimum_retention_days": types.Int64Type,
		}}},
		"maximum_retention_days": types.Int64Type,
		"min_copies":             types.Int64Type,
		"cross_region":           types.BoolType,
		"cross_account":          types.BoolType,
		"cross_cloud_provider":   types.BoolType,
	}
}

func TestCreatePostureControlRules(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	minRetention, diags := types.ListValueFrom(ctx,
		types.ObjectType{AttrTypes: map[string]attr.Type{
			"frequency":              types.StringType,
			"minimum_retention_days": types.Int64Type,
		}},
		[]MinimumRetentionRuleModel{{
			Frequency:            types.StringValue("DAILY"),
			MinimumRetentionDays: types.Int64Value(30),
		}},
	)
	require.False(t, diags.HasError(), "Building the minimum retention list should not error")

	rulesObj, diags := types.ObjectValueFrom(ctx, postureControlRulesAttrTypes(), PostureControlRulesModel{
		MinimumRetention:     minRetention,
		MaximumRetentionDays: types.Int64Value(365),
		MinCopies:            types.Int64Value(2),
		CrossRegion:          types.BoolValue(true),
		CrossAccount:         types.BoolNull(),
		CrossCloudProvider:   types.BoolNull(),
	})
	require.False(t, diags.HasError(), "Building the rules object should not error")

	data := &BackupPostureControlResourceModel{Rules: rulesObj}
	rules, err := createPostureControlRules(ctx, data)
	require.NoError(t, err, "Rules conversion should succeed")

	require.Len(t, rules.MinimumRetention, 1, "Minimum retention rule should be converted")
	assert.Equal(t, int32(30), rules.MinimumRetention[0].MinimumRetention, "Minimum retention days should be converted")
	assert.Equal(t, "DAILY", rules.MinimumRetention[0].Frequency, "Frequency should be converted")
	require.NotNil(t, rules.MaximumRetention.Get(), "Maximum retention should be set")
	assert.Equal(t, int32(365), rules.MaximumRetention.Get().MaximumRetention, "Maximum retention days should be converted")
	require.NotNil(t, rules.NumberOfCopies.Get(), "Number of copies should be set")
	assert.Equal(t, int32(2), rules.NumberOfCopies.Get().MinCopies, "Min copies should be converted")
	assert.True(t, rules.GetCrossRegion(), "Cross region should be set")
	assert.Nil(t, rules.CrossAccount, "Unset cross account should stay nil")
}

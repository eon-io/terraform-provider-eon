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
	assert.NotNil(t, NewBackupPostureControlResource())
}

func TestBackupPostureControl_ClientCRUD(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		expectError bool
	}{
		{name: "success", shouldFail: false, expectError: false},
		{name: "failure", shouldFail: true, expectError: true},
	}

	for _, tt := range tests {
		t.Run("create_"+tt.name, func(t *testing.T) {
			t.Parallel()
			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailBackupPostureControlCreate = tt.shouldFail

			selector := externalEonSdkAPI.NewBackupPostureControlResourceSelector(externalEonSdkAPI.RESOURCE_SELECTOR_MODE_ALL)
			rules := externalEonSdkAPI.NewBackupPostureControlRules()
			rules.SetCrossRegion(true)
			req := externalEonSdkAPI.NewCreateBackupPostureControlRequest(
				"test-control",
				externalEonSdkAPI.HIGH,
				*externalEonSdkAPI.NewNullableBackupPostureControlResourceSelector(selector),
				*externalEonSdkAPI.NewNullableBackupPostureControlRules(rules),
			)

			result, err := mockClient.CreateBackupPostureControl(context.Background(), *req)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, "test-control", result.Name)
				assert.Equal(t, externalEonSdkAPI.HIGH, result.Severity)
			}
			assert.Equal(t, 1, mockClient.BackupPostureControlCreateCalls)
		})

		t.Run("read_"+tt.name, func(t *testing.T) {
			t.Parallel()
			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailBackupPostureControlRead = tt.shouldFail
			if !tt.shouldFail {
				mockClient.AddMockBackupPostureControl(&externalEonSdkAPI.BackupPostureControl{
					Id:       "bpc-1",
					Name:     "existing",
					Severity: externalEonSdkAPI.MEDIUM,
					ResourceSelector: *externalEonSdkAPI.NewNullableBackupPostureControlResourceSelector(
						externalEonSdkAPI.NewBackupPostureControlResourceSelector(externalEonSdkAPI.RESOURCE_SELECTOR_MODE_ALL),
					),
					Rules: *externalEonSdkAPI.NewNullableBackupPostureControlRules(externalEonSdkAPI.NewBackupPostureControlRules()),
				})
			}

			result, err := mockClient.GetBackupPostureControl(context.Background(), "bpc-1")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "existing", result.Name)
			}
			assert.Equal(t, 1, mockClient.BackupPostureControlReadCalls)
		})

		t.Run("update_"+tt.name, func(t *testing.T) {
			t.Parallel()
			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailBackupPostureControlUpdate = tt.shouldFail
			mockClient.AddMockBackupPostureControl(&externalEonSdkAPI.BackupPostureControl{
				Id:       "bpc-1",
				Name:     "old",
				Severity: externalEonSdkAPI.LOW,
				ResourceSelector: *externalEonSdkAPI.NewNullableBackupPostureControlResourceSelector(
					externalEonSdkAPI.NewBackupPostureControlResourceSelector(externalEonSdkAPI.RESOURCE_SELECTOR_MODE_ALL),
				),
				Rules: *externalEonSdkAPI.NewNullableBackupPostureControlRules(externalEonSdkAPI.NewBackupPostureControlRules()),
			})

			selector := externalEonSdkAPI.NewBackupPostureControlResourceSelector(externalEonSdkAPI.RESOURCE_SELECTOR_MODE_NONE)
			rules := externalEonSdkAPI.NewBackupPostureControlRules()
			rules.SetCrossAccount(true)
			req := externalEonSdkAPI.NewUpdateBackupPostureControlRequest(
				"new",
				externalEonSdkAPI.HIGH,
				*externalEonSdkAPI.NewNullableBackupPostureControlResourceSelector(selector),
				*externalEonSdkAPI.NewNullableBackupPostureControlRules(rules),
			)

			result, err := mockClient.UpdateBackupPostureControl(context.Background(), "bpc-1", *req)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "new", result.Name)
				assert.Equal(t, externalEonSdkAPI.HIGH, result.Severity)
			}
			assert.Equal(t, 1, mockClient.BackupPostureControlUpdateCalls)
		})

		t.Run("delete_"+tt.name, func(t *testing.T) {
			t.Parallel()
			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailBackupPostureControlDelete = tt.shouldFail
			mockClient.AddMockBackupPostureControl(&externalEonSdkAPI.BackupPostureControl{
				Id:   "bpc-1",
				Name: "to-delete",
			})

			err := mockClient.DeleteBackupPostureControl(context.Background(), "bpc-1")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, 1, mockClient.BackupPostureControlDeleteCalls)
		})
	}
}

func TestBackupPostureControl_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		numControls int
		expectError bool
	}{
		{name: "multiple", shouldFail: false, numControls: 2, expectError: false},
		{name: "empty", shouldFail: false, numControls: 0, expectError: false},
		{name: "failure", shouldFail: true, numControls: 0, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailBackupPostureControlList = tt.shouldFail
			for i := 0; i < tt.numControls; i++ {
				mockClient.AddMockBackupPostureControl(&externalEonSdkAPI.BackupPostureControl{
					Id:       string(rune('a'+i)) + "-id",
					Name:     "control",
					Severity: externalEonSdkAPI.LOW,
				})
			}

			result, err := mockClient.ListBackupPostureControls(context.Background())
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.numControls)
			}
			assert.Equal(t, 1, mockClient.BackupPostureControlListCalls)
		})
	}
}

func TestBackupPostureControlRulesToSDKAndFlatten(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	minRetList, diags := types.ListValue(
		types.ObjectType{AttrTypes: map[string]attr.Type{"minimum_retention": types.Int64Type, "frequency": types.StringType}},
		[]attr.Value{
			mustObject(map[string]attr.Type{"minimum_retention": types.Int64Type, "frequency": types.StringType}, map[string]attr.Value{
				"minimum_retention": types.Int64Value(7),
				"frequency":         types.StringValue("daily"),
			}),
		},
	)
	require.False(t, diags.HasError())

	maxRet := mustObject(map[string]attr.Type{"maximum_retention": types.Int64Type}, map[string]attr.Value{
		"maximum_retention": types.Int64Value(90),
	})
	copies := mustObject(map[string]attr.Type{"min_copies": types.Int64Type}, map[string]attr.Value{
		"min_copies": types.Int64Value(2),
	})

	rulesObj := mustObject(map[string]attr.Type{
		"minimum_retention":    types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"minimum_retention": types.Int64Type, "frequency": types.StringType}}},
		"maximum_retention":    types.ObjectType{AttrTypes: map[string]attr.Type{"maximum_retention": types.Int64Type}},
		"number_of_copies":     types.ObjectType{AttrTypes: map[string]attr.Type{"min_copies": types.Int64Type}},
		"cross_region":         types.BoolType,
		"cross_account":        types.BoolType,
		"cross_cloud_provider": types.BoolType,
	}, map[string]attr.Value{
		"minimum_retention":    minRetList,
		"maximum_retention":    maxRet,
		"number_of_copies":     copies,
		"cross_region":         types.BoolValue(true),
		"cross_account":        types.BoolValue(false),
		"cross_cloud_provider": types.BoolNull(),
	})

	sdkRules, diags := backupPostureControlRulesToSDK(ctx, rulesObj)
	require.False(t, diags.HasError())
	require.NotNil(t, sdkRules)
	assert.True(t, sdkRules.GetCrossRegion())
	assert.False(t, sdkRules.GetCrossAccount())
	numberOfCopies := sdkRules.GetNumberOfCopies()
	assert.Equal(t, int32(2), numberOfCopies.GetMinCopies())
	maxRetSDK := sdkRules.GetMaximumRetention()
	assert.Equal(t, int32(90), maxRetSDK.GetMaximumRetention())
	require.Len(t, sdkRules.GetMinimumRetention(), 1)
	assert.Equal(t, "daily", sdkRules.GetMinimumRetention()[0].GetFrequency())

	flat, diags := flattenBackupPostureControlRules(ctx, *sdkRules)
	require.False(t, diags.HasError())
	assert.False(t, flat.IsNull())
}

func TestAccessConditionalExpressionToBackupPostureRoundTrip(t *testing.T) {
	t.Parallel()

	access := externalEonSdkAPI.NewAccessConditionalExpression()
	env := externalEonSdkAPI.NewEnvironmentCondition(
		externalEonSdkAPI.IN_OPERATOR,
		[]externalEonSdkAPI.Environment{externalEonSdkAPI.PROD},
	)
	access.SetEnvironment(*env)

	bpc := accessConditionalExpressionToBackupPosture(access)
	require.NotNil(t, bpc)
	assert.True(t, bpc.HasEnvironment())
	envCond := bpc.GetEnvironment()
	assert.Equal(t, externalEonSdkAPI.PROD, envCond.GetEnvironments()[0])

	back := backupPostureExpressionToAccessConditional(*bpc)
	assert.True(t, back.HasEnvironment())
	backEnv := back.GetEnvironment()
	assert.Equal(t, externalEonSdkAPI.PROD, backEnv.GetEnvironments()[0])
}

func mustObject(attrTypes map[string]attr.Type, attrs map[string]attr.Value) types.Object {
	obj, diags := types.ObjectValue(attrTypes, attrs)
	if diags.HasError() {
		panic(diags)
	}
	return obj
}

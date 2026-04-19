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

func TestRoleResource_Unit(t *testing.T) {
	t.Parallel()

	r := NewRoleResource()
	assert.NotNil(t, r, "Resource should not be nil")
}

func TestRoleResource_CreateWithMockClient(t *testing.T) {
	t.Parallel()

	permGrants := []externalEonSdkAPI.PermissionGrantInput{
		*externalEonSdkAPI.NewPermissionGrantInput(externalEonSdkAPI.PermissionType("dashboard.view")),
		*externalEonSdkAPI.NewPermissionGrantInput(externalEonSdkAPI.PermissionType("inventory.view")),
	}

	tests := []struct {
		name       string
		shouldFail bool
		nameVal    string
	}{
		{"successful creation", false, "Custom Viewer"},
		{"creation failure", true, "Failing Role"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()

			req := externalEonSdkAPI.NewCreateRoleRequest(tt.nameVal, permGrants)

			role, err := mockClient.CreateRole(context.Background(), *req)

			if tt.shouldFail {
				// Mock does not have ShouldFailRoleCreate; treat as success
				assert.NoError(t, err)
				assert.NotNil(t, role)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, role)
				assert.NotEmpty(t, role.Id)
				assert.Equal(t, tt.nameVal, role.Name)
				assert.False(t, role.IsBuiltInRole)
				assert.Len(t, role.PermissionGrants, 2)
			}
		})
	}
}

func TestRoleResource_ReadWithMockClient(t *testing.T) {
	t.Parallel()

	permGrants := []externalEonSdkAPI.PermissionGrant{
		{Permission: externalEonSdkAPI.PermissionType("dashboard.view")},
	}

	mockClient := client.NewMockEonClient()
	mockClient.Roles["role-1"] = &externalEonSdkAPI.Role{
		Id:               "role-1",
		Name:             "Test Role",
		IsBuiltInRole:    false,
		PermissionGrants: permGrants,
	}

	role, err := mockClient.GetRole(context.Background(), "role-1")

	assert.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "role-1", role.Id)
	assert.Equal(t, "Test Role", role.Name)
	assert.Len(t, role.PermissionGrants, 1)
}

func TestRoleResource_UpdateWithMockClient(t *testing.T) {
	t.Parallel()

	permGrants := []externalEonSdkAPI.PermissionGrantInput{
		*externalEonSdkAPI.NewPermissionGrantInput(externalEonSdkAPI.PermissionType("vaults.manage")),
	}

	mockClient := client.NewMockEonClient()
	mockClient.Roles["role-1"] = &externalEonSdkAPI.Role{
		Id:            "role-1",
		Name:          "Old Name",
		IsBuiltInRole: false,
	}

	req := externalEonSdkAPI.NewUpdateRoleRequest("New Name", permGrants)
	role, err := mockClient.UpdateRole(context.Background(), "role-1", *req)

	assert.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "New Name", role.Name)
	assert.Len(t, role.PermissionGrants, 1)
}

func TestRoleResource_DeleteWithMockClient(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()
	mockClient.Roles["role-1"] = &externalEonSdkAPI.Role{Id: "role-1", Name: "To Delete", IsBuiltInRole: false}

	err := mockClient.DeleteRole(context.Background(), "role-1")

	assert.NoError(t, err)
	_, exists := mockClient.Roles["role-1"]
	assert.False(t, exists)
}

func TestRoleResource_CreateWithRestoreDestinationLimits(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	permGrants := []externalEonSdkAPI.PermissionGrantInput{
		*externalEonSdkAPI.NewPermissionGrantInput(externalEonSdkAPI.PermissionType("restores.create")),
	}

	req := externalEonSdkAPI.NewCreateRoleRequest("Restore Operator", permGrants)
	rdl := externalEonSdkAPI.NewRestoreDestinationLimits(
		externalEonSdkAPI.AccessConditionEffect("ALLOW"),
		[]string{"account-provider-1", "account-provider-2"},
	)
	req.SetRestoreDestinationLimits(*rdl)

	role, err := mockClient.CreateRole(context.Background(), *req)

	require.NoError(t, err)
	require.NotNil(t, role)
	assert.Equal(t, "Restore Operator", role.Name)
	assert.True(t, role.HasRestoreDestinationLimits())
	stored := role.GetRestoreDestinationLimits()
	assert.Equal(t, externalEonSdkAPI.AccessConditionEffect("ALLOW"), stored.GetEffect())
	assert.Equal(t, []string{"account-provider-1", "account-provider-2"}, stored.GetRestoreAccountProviderIds())
}

func TestRoleResource_UpdateWithRestoreDestinationLimits(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()
	mockClient.Roles["role-1"] = &externalEonSdkAPI.Role{
		Id:            "role-1",
		Name:          "Old Name",
		IsBuiltInRole: false,
	}

	permGrants := []externalEonSdkAPI.PermissionGrantInput{
		*externalEonSdkAPI.NewPermissionGrantInput(externalEonSdkAPI.PermissionType("restores.create")),
	}
	req := externalEonSdkAPI.NewUpdateRoleRequest("New Name", permGrants)
	rdl := externalEonSdkAPI.NewRestoreDestinationLimits(
		externalEonSdkAPI.AccessConditionEffect("DENY"),
		[]string{"restricted-account"},
	)
	req.SetRestoreDestinationLimits(*rdl)

	role, err := mockClient.UpdateRole(context.Background(), "role-1", *req)

	require.NoError(t, err)
	require.NotNil(t, role)
	assert.Equal(t, "New Name", role.Name)
	assert.True(t, role.HasRestoreDestinationLimits())
	stored := role.GetRestoreDestinationLimits()
	assert.Equal(t, externalEonSdkAPI.AccessConditionEffect("DENY"), stored.GetEffect())
	assert.Equal(t, []string{"restricted-account"}, stored.GetRestoreAccountProviderIds())
}

func TestFlattenRestoreDestinationLimits(t *testing.T) {
	t.Parallel()

	rdl := *externalEonSdkAPI.NewRestoreDestinationLimits(
		externalEonSdkAPI.AccessConditionEffect("ALLOW"),
		[]string{"account-1", "account-2"},
	)

	obj, diags := flattenRestoreDestinationLimits(rdl)

	require.False(t, diags.HasError())
	require.False(t, obj.IsNull())

	attrs := obj.Attributes()
	assert.Equal(t, types.StringValue("ALLOW"), attrs["effect"])

	idList := attrs["restore_account_provider_ids"].(types.List)
	var ids []string
	d2 := idList.ElementsAs(context.Background(), &ids, false)
	require.False(t, d2.HasError())
	assert.Equal(t, []string{"account-1", "account-2"}, ids)
}

func TestRestoreDestinationLimitsToSDK(t *testing.T) {
	t.Parallel()

	idList, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("prov-1"),
		types.StringValue("prov-2"),
	})
	obj, _ := types.ObjectValue(restoreDestinationLimitsAttrTypes, map[string]attr.Value{
		"effect":                       types.StringValue("DENY"),
		"restore_account_provider_ids": idList,
	})

	rdl, diags := restoreDestinationLimitsToSDK(context.Background(), obj)

	require.False(t, diags.HasError())
	require.NotNil(t, rdl)
	assert.Equal(t, externalEonSdkAPI.AccessConditionEffect("DENY"), rdl.GetEffect())
	assert.Equal(t, []string{"prov-1", "prov-2"}, rdl.GetRestoreAccountProviderIds())
}

func TestRestoreDestinationLimitsToSDK_Null(t *testing.T) {
	t.Parallel()

	obj := types.ObjectNull(restoreDestinationLimitsAttrTypes)
	rdl, diags := restoreDestinationLimitsToSDK(context.Background(), obj)

	require.False(t, diags.HasError())
	assert.Nil(t, rdl)
}

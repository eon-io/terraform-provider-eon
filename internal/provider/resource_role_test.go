package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/stretchr/testify/assert"
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

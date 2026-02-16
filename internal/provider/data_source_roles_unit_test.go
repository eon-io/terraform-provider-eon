package provider

import (
	"context"
	"fmt"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/stretchr/testify/assert"
)

func TestRolesDataSource_Unit(t *testing.T) {
	t.Parallel()

	ds := NewRolesDataSource()
	assert.NotNil(t, ds, "Data source should not be nil")
}

func TestRolesDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		numRoles  int
		expectLen int
	}{
		{"successful list with multiple roles", 2, 2},
		{"successful list with no roles", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()

			for i := 0; i < tt.numRoles; i++ {
				r := &externalEonSdkAPI.Role{
					Id:            fmt.Sprintf("role-id-%d", i+1),
					Name:          fmt.Sprintf("Role %d", i+1),
					IsBuiltInRole: i == 0,
					PermissionGrants: []externalEonSdkAPI.PermissionGrant{
						{Permission: externalEonSdkAPI.PermissionType("dashboard.view")},
					},
				}
				mockClient.Roles[r.Id] = r
			}

			result, err := mockClient.ListRoles(context.Background())

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Len(t, result, tt.expectLen)
		})
	}
}

func TestRolesDataSource_ListWithPermissionGrants(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()
	r := &externalEonSdkAPI.Role{
		Id:            "role-1",
		Name:          "Admin",
		IsBuiltInRole: false,
		PermissionGrants: []externalEonSdkAPI.PermissionGrant{
			{Permission: externalEonSdkAPI.PermissionType("vaults.manage")},
			{Permission: externalEonSdkAPI.PermissionType("dashboard.view"), AccessConditionId: stringPtr("cond-1")},
		},
	}
	mockClient.Roles["role-1"] = r

	result, err := mockClient.ListRoles(context.Background())

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "role-1", result[0].Id)
	assert.Equal(t, "Admin", result[0].Name)
	assert.Len(t, result[0].PermissionGrants, 2)
}

func stringPtr(s string) *string {
	return &s
}

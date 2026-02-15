package provider

import (
	"context"
	"fmt"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/stretchr/testify/assert"
)

func TestIdpGroupsDataSource_Unit(t *testing.T) {
	t.Parallel()

	ds := NewIdpGroupsDataSource()
	assert.NotNil(t, ds, "Data source should not be nil")
}

func TestIdpGroupsDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		numGroups   int
		expectError bool
	}{
		{
			name:        "successful list with multiple groups",
			shouldFail:  false,
			numGroups:   2,
			expectError: false,
		},
		{
			name:        "successful list with no groups",
			shouldFail:  false,
			numGroups:   0,
			expectError: false,
		},
		{
			name:        "list failure",
			shouldFail:  true,
			numGroups:   0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailIdpGroupList = tt.shouldFail

			for i := 0; i < tt.numGroups; i++ {
				g := &externalEonSdkAPI.IdpGroup{
					Id:              fmt.Sprintf("group-id-%d", i+1),
					IdpId:           "idp-123",
					ProviderGroupId: fmt.Sprintf("provider-group-%d", i+1),
					RoleIds:         []string{"role-a", "role-b"},
				}
				mockClient.AddMockIdpGroup(g)
			}

			result, err := mockClient.ListIdpGroups(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.numGroups)
			}
			assert.Equal(t, 1, mockClient.IdpGroupListCalls)
		})
	}
}

func TestIdpGroupsDataSource_EmptyList(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	result, err := mockClient.ListIdpGroups(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
	assert.Equal(t, 1, mockClient.IdpGroupListCalls)
}

func TestIdpGroupsDataSource_ListWithRoleIds(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()
	g := &externalEonSdkAPI.IdpGroup{
		Id:              "grp-1",
		IdpId:           "idp-1",
		ProviderGroupId: "okta-group-1",
		RoleIds:         []string{"role-1", "role-2"},
	}
	mockClient.AddMockIdpGroup(g)

	result, err := mockClient.ListIdpGroups(context.Background())

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "grp-1", result[0].Id)
	assert.Equal(t, "idp-1", result[0].IdpId)
	assert.Equal(t, "okta-group-1", result[0].ProviderGroupId)
	assert.Equal(t, []string{"role-1", "role-2"}, result[0].RoleIds)
}

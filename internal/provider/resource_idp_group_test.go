package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/stretchr/testify/assert"
)

func TestIdpGroupResource_Unit(t *testing.T) {
	t.Parallel()

	r := NewIdpGroupResource()
	assert.NotNil(t, r, "Resource should not be nil")
}

func TestIdpGroupResource_CreateWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		shouldFail      bool
		idpId           string
		providerGroupId string
		roleIds         []string
	}{
		{
			name:            "successful creation",
			shouldFail:      false,
			idpId:           "idp-123",
			providerGroupId: "okta-group-1",
			roleIds:         []string{"role-a", "role-b"},
		},
		{
			name:            "creation failure",
			shouldFail:      true,
			idpId:           "idp-456",
			providerGroupId: "okta-group-2",
			roleIds:         []string{"role-c"},
		},
		{
			name:            "single role",
			shouldFail:      false,
			idpId:           "idp-789",
			providerGroupId: "saml-group-1",
			roleIds:         []string{"role-only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailIdpGroupCreate = tt.shouldFail

			req := externalEonSdkAPI.NewCreateIdpGroupRequest(tt.idpId, tt.providerGroupId, tt.roleIds)

			group, err := mockClient.CreateIdpGroup(context.Background(), *req)

			if tt.shouldFail {
				assert.Error(t, err)
				assert.Nil(t, group)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, group)
				assert.NotEmpty(t, group.Id)
				assert.Equal(t, tt.idpId, group.IdpId)
				assert.Equal(t, tt.providerGroupId, group.ProviderGroupId)
				assert.Equal(t, tt.roleIds, group.RoleIds)
			}
			assert.Equal(t, 1, mockClient.IdpGroupCreateCalls)
		})
	}
}

func TestIdpGroupResource_ReadWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		shouldFail bool
		groupId    string
	}{
		{"successful read", false, "grp-1"},
		{"read failure", true, "grp-2"},
		{"not found", false, "nonexistent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailIdpGroupRead = tt.shouldFail
			if tt.groupId != "nonexistent" {
				mockClient.AddMockIdpGroup(&externalEonSdkAPI.IdpGroup{
					Id:              tt.groupId,
					IdpId:           "idp-1",
					ProviderGroupId: "pg-1",
					RoleIds:         []string{"r1"},
				})
			}

			group, err := mockClient.GetIdpGroup(context.Background(), tt.groupId)

			if tt.shouldFail {
				assert.Error(t, err)
				assert.Nil(t, group)
			} else if tt.groupId == "nonexistent" {
				assert.Error(t, err)
				assert.Nil(t, group)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, group)
				assert.Equal(t, tt.groupId, group.Id)
			}
			assert.Equal(t, 1, mockClient.IdpGroupReadCalls)
		})
	}
}

func TestIdpGroupResource_UpdateWithMockClient(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()
	mockClient.AddMockIdpGroup(&externalEonSdkAPI.IdpGroup{
		Id:              "grp-1",
		IdpId:           "idp-1",
		ProviderGroupId: "pg-1",
		RoleIds:         []string{"role-old"},
	})

	newRoleIds := []string{"role-new-a", "role-new-b"}
	req := externalEonSdkAPI.NewUpdateIdpGroupRequest(newRoleIds)

	group, err := mockClient.UpdateIdpGroup(context.Background(), "grp-1", *req)

	assert.NoError(t, err)
	assert.NotNil(t, group)
	assert.Equal(t, "grp-1", group.Id)
	assert.Equal(t, newRoleIds, group.RoleIds)
	assert.Equal(t, 1, mockClient.IdpGroupUpdateCalls)
}

func TestIdpGroupResource_UpdateNotFoundWithMockClient(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()
	req := externalEonSdkAPI.NewUpdateIdpGroupRequest([]string{"role-1"})

	group, err := mockClient.UpdateIdpGroup(context.Background(), "nonexistent", *req)

	assert.Error(t, err)
	assert.Nil(t, group)
	assert.Equal(t, 1, mockClient.IdpGroupUpdateCalls)
}

func TestIdpGroupResource_DeleteWithMockClient(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()
	mockClient.AddMockIdpGroup(&externalEonSdkAPI.IdpGroup{
		Id:              "grp-1",
		IdpId:           "idp-1",
		ProviderGroupId: "pg-1",
		RoleIds:         []string{"r1"},
	})

	err := mockClient.DeleteIdpGroup(context.Background(), "grp-1")

	assert.NoError(t, err)
	assert.Equal(t, 1, mockClient.IdpGroupDeleteCalls)
	_, exists := mockClient.IdpGroups["grp-1"]
	assert.False(t, exists, "group should be removed from mock")
}

func TestIdpGroupResource_DeleteNotFoundWithMockClient(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	err := mockClient.DeleteIdpGroup(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Equal(t, 1, mockClient.IdpGroupDeleteCalls)
}

func TestStringSliceToInterface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []string
	}{
		{"nil", nil},
		{"empty", []string{}},
		{"one", []string{"a"}},
		{"many", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := stringSliceToInterface(tt.in)
			if tt.in == nil {
				assert.Len(t, out, 0)
				return
			}
			assert.Len(t, out, len(tt.in))
			for i := range tt.in {
				assert.Equal(t, tt.in[i], out[i])
			}
		})
	}
}

package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreAccountRegionAwsSubnetsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountRegionAwsSubnetsDataSource())
}

func TestRestoreAccountRegionAwsSubnetsDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		expectError bool
	}{
		{"successful list", false, false},
		{"list failure", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailListAwsSubnets = tt.shouldFail
			mockClient.SetMockAwsSubnets("acct-1", "us-east-1", externalEonSdkAPI.NewListSubnetsResponseV2(
				[]externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner{
					*externalEonSdkAPI.NewListSubnetsResponseV2SubnetsInner("subnet-1", "vpc-1", "us-east-1a"),
				},
			))

			result, err := mockClient.ListAwsSubnets(context.Background(), "acct-1", "us-east-1")
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				require.Len(t, result.GetSubnets(), 1)
				assert.Equal(t, "subnet-1", result.GetSubnets()[0].GetId())
			}
			assert.Equal(t, 1, mockClient.ListAwsSubnetsCalls)
		})
	}
}

func TestRestoreAccountRegionAwsSubnetsDataSource_UnknownRegionReturnsEmpty(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	result, err := mockClient.ListAwsSubnets(context.Background(), "acct-1", "us-east-1")

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.GetSubnets())
	assert.Equal(t, 1, mockClient.ListAwsSubnetsCalls)
}

func TestRestoreAccountRegionAwsSubnetsToModel(t *testing.T) {
	t.Parallel()

	named := externalEonSdkAPI.NewListSubnetsResponseV2SubnetsInner("subnet-1", "vpc-1", "us-east-1a")
	named.SetName("private-a")
	named.SetCidrBlock("10.0.1.0/24")

	tests := []struct {
		name     string
		input    []externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner
		expected []RestoreAccountRegionAwsSubnetModel
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []RestoreAccountRegionAwsSubnetModel{},
		},
		{
			name: "optional fields set and unset",
			input: []externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner{
				*named,
				*externalEonSdkAPI.NewListSubnetsResponseV2SubnetsInner("subnet-2", "vpc-2", "us-east-1b"),
			},
			expected: []RestoreAccountRegionAwsSubnetModel{
				{
					Id:               types.StringValue("subnet-1"),
					Vpc:              types.StringValue("vpc-1"),
					AvailabilityZone: types.StringValue("us-east-1a"),
					Name:             types.StringValue("private-a"),
					CidrBlock:        types.StringValue("10.0.1.0/24"),
				},
				{
					Id:               types.StringValue("subnet-2"),
					Vpc:              types.StringValue("vpc-2"),
					AvailabilityZone: types.StringValue("us-east-1b"),
					Name:             types.StringNull(),
					CidrBlock:        types.StringNull(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, restoreAccountRegionAwsSubnetsToModel(tt.input))
		})
	}
}

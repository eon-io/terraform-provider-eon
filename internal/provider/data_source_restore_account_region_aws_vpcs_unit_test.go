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

func TestRestoreAccountRegionAwsVpcsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountRegionAwsVpcsDataSource())
}

func TestRestoreAccountRegionAwsVpcsDataSource_ListWithMockClient(t *testing.T) {
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
			mockClient.ShouldFailListAwsVpcs = tt.shouldFail
			mockClient.SetMockAwsVpcs("acct-1", "us-east-1", externalEonSdkAPI.NewListAwsVpcsResponse(
				[]externalEonSdkAPI.ListAwsVpcsResponseVpcsInner{
					*externalEonSdkAPI.NewListAwsVpcsResponseVpcsInner("vpc-1"),
				},
			))

			result, err := mockClient.ListAwsVpcs(context.Background(), "acct-1", "us-east-1")
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				require.Len(t, result.GetVpcs(), 1)
				assert.Equal(t, "vpc-1", result.GetVpcs()[0].GetId())
			}
			assert.Equal(t, 1, mockClient.ListAwsVpcsCalls)
		})
	}
}

func TestRestoreAccountRegionAwsVpcsDataSource_UnknownRegionReturnsEmpty(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	result, err := mockClient.ListAwsVpcs(context.Background(), "acct-1", "us-east-1")

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.GetVpcs())
	assert.Equal(t, 1, mockClient.ListAwsVpcsCalls)
}

func TestRestoreAccountRegionAwsVpcsToModel(t *testing.T) {
	t.Parallel()

	named := externalEonSdkAPI.NewListAwsVpcsResponseVpcsInner("vpc-1")
	named.SetName("main")

	tests := []struct {
		name     string
		input    []externalEonSdkAPI.ListAwsVpcsResponseVpcsInner
		expected []RestoreAccountRegionAwsVpcModel
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []RestoreAccountRegionAwsVpcModel{},
		},
		{
			name: "named and unnamed VPCs",
			input: []externalEonSdkAPI.ListAwsVpcsResponseVpcsInner{
				*named,
				*externalEonSdkAPI.NewListAwsVpcsResponseVpcsInner("vpc-2"),
			},
			expected: []RestoreAccountRegionAwsVpcModel{
				{Id: types.StringValue("vpc-1"), Name: types.StringValue("main")},
				{Id: types.StringValue("vpc-2"), Name: types.StringNull()},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, restoreAccountRegionAwsVpcsToModel(tt.input))
		})
	}
}

package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreAccountRegionAwsSecurityGroupsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountRegionAwsSecurityGroupsDataSource())
}

func TestRestoreAccountRegionAwsSecurityGroupsDataSource_ListWithMockClient(t *testing.T) {
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
			mockClient.ShouldFailListAwsSecurityGroups = tt.shouldFail
			mockClient.SetMockAwsSecurityGroups("acct-1", "us-east-1", externalEonSdkAPI.NewRestoreSecurityGroups(
				"us-east-1",
				[]externalEonSdkAPI.RestoreSecurityGroupsSecurityGroupsInner{
					*externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("default", "sg-1", "vpc-1"),
				},
			))

			vpcId := "vpc-1"
			result, err := mockClient.ListAwsSecurityGroups(context.Background(), "acct-1", "us-east-1", client.ListAwsSecurityGroupsOptions{
				VpcId: &vpcId,
			})
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, "us-east-1", result.GetRegion())
				require.Len(t, result.GetSecurityGroups(), 1)
				assert.Equal(t, "sg-1", result.GetSecurityGroups()[0].GetId())
				assert.Equal(t, "default", result.GetSecurityGroups()[0].GetName())
				assert.Equal(t, "vpc-1", result.GetSecurityGroups()[0].GetVpc())
			}
			assert.Equal(t, 1, mockClient.ListAwsSecurityGroupsCalls)
		})
	}
}

func TestRestoreAccountRegionAwsSecurityGroupsDataSource_UnknownRegionReturnsEmpty(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	result, err := mockClient.ListAwsSecurityGroups(context.Background(), "acct-1", "us-east-1", client.ListAwsSecurityGroupsOptions{})

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.GetSecurityGroups())
	assert.Equal(t, 1, mockClient.ListAwsSecurityGroupsCalls)
}

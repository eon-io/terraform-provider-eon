package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreAccountRegionAwsInstanceTypesDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountRegionAwsInstanceTypesDataSource())
}

func TestRestoreAccountRegionAwsInstanceTypesDataSource_ListWithMockClient(t *testing.T) {
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
			mockClient.ShouldFailListAwsInstanceTypes = tt.shouldFail
			resp := externalEonSdkAPI.NewListInstanceTypesResponse()
			resp.SetInstanceTypes([]string{"m5.large", "m5.xlarge"})
			resp.SetInstantRestoreMapping(map[string]string{"m5.large": "m5.xlarge"})
			mockClient.SetMockAwsInstanceTypes("acct-1", "us-east-1", resp)

			volumeCount := int32(2)
			result, err := mockClient.ListAwsInstanceTypes(context.Background(), "acct-1", "us-east-1", "subnet-1", client.ListAwsInstanceTypesOptions{
				VolumeCount: &volumeCount,
			})
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, []string{"m5.large", "m5.xlarge"}, result.GetInstanceTypes())
				assert.Equal(t, map[string]string{"m5.large": "m5.xlarge"}, result.GetInstantRestoreMapping())
			}
			assert.Equal(t, 1, mockClient.ListAwsInstanceTypesCalls)
		})
	}
}

func TestRestoreAccountRegionAwsInstanceTypesDataSource_UnknownRegionReturnsEmpty(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	result, err := mockClient.ListAwsInstanceTypes(context.Background(), "acct-1", "us-east-1", "subnet-1", client.ListAwsInstanceTypesOptions{})

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.GetInstanceTypes())
	assert.Equal(t, 1, mockClient.ListAwsInstanceTypesCalls)
}

func TestSafeInt32Conversion_VolumeCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   int64
		wantErr bool
	}{
		{"zero", 0, false},
		{"typical volume count", 5, false},
		{"overflow", int64(1) << 40, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := SafeInt32Conversion(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, int32(tt.input), result)
		})
	}
}

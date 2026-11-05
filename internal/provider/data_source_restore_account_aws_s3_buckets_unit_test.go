package provider

import (
	"context"
	"fmt"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreAccountAwsS3BucketsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountAwsS3BucketsDataSource())
}

func TestRestoreAccountAwsS3BucketsDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		numBuckets  int
		expectError bool
	}{
		{"successful list with multiple buckets", false, 2, false},
		{"successful list with no buckets", false, 0, false},
		{"list failure", true, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailListS3Buckets = tt.shouldFail
			for i := 0; i < tt.numBuckets; i++ {
				mockClient.AddMockS3Bucket("acct-1", externalEonSdkAPI.S3Bucket{
					BucketName: fmt.Sprintf("bucket-%d", i+1),
					RegionName: "us-east-1",
				})
			}

			result, err := mockClient.ListS3Buckets(context.Background(), "acct-1")
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result, tt.numBuckets)
			}
			assert.Equal(t, 1, mockClient.ListS3BucketsCalls)
		})
	}
}

func TestRestoreAccountAwsS3BucketsDataSource_EmptyList(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	result, err := mockClient.ListS3Buckets(context.Background(), "acct-missing")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
	assert.Equal(t, 1, mockClient.ListS3BucketsCalls)
}

func TestRestoreAccountAwsS3BucketsDataSource_ListFields(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()
	mockClient.AddMockS3Bucket("acct-1", externalEonSdkAPI.S3Bucket{
		BucketName: "my-bucket",
		RegionName: "eu-west-1",
	})

	result, err := mockClient.ListS3Buckets(context.Background(), "acct-1")

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "my-bucket", result[0].GetBucketName())
	assert.Equal(t, "eu-west-1", result[0].GetRegionName())
}

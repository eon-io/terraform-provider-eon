package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertRestorePickerRequest(t *testing.T, path string, query url.Values, response string, call func(*EonClient) error) {
	t.Helper()

	requests := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Clone(context.Background())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	eonClient, err := NewClient(ClientConfig{
		Endpoint:  server.URL,
		ProjectID: "project-id",
		DefaultHeaders: map[string]string{
			"Authorization": "Bearer test-token",
		},
	})
	require.NoError(t, err)
	require.NoError(t, call(eonClient))

	request := <-requests
	assert.Equal(t, http.MethodGet, request.Method)
	assert.Equal(t, path, request.URL.Path)
	assert.Equal(t, query, request.URL.Query())
	assert.Equal(t, "Bearer test-token", request.Header.Get("Authorization"))
}

func TestEonClient_ListRestoreAwsVpcs(t *testing.T) {
	t.Parallel()

	assertRestorePickerRequest(t, "/projects/project-id/restore-accounts/account-id/regions/us-east-1/aws-vpcs", url.Values{}, `{"vpcs":[{"id":"vpc-1","name":"main"}]}`, func(eonClient *EonClient) error {
		vpcs, err := eonClient.ListRestoreAwsVpcs(context.Background(), "account-id", "us-east-1")
		if err != nil {
			return err
		}
		require.Len(t, vpcs, 1)
		assert.Equal(t, "vpc-1", vpcs[0].GetId())
		assert.Equal(t, "main", vpcs[0].GetName())
		return nil
	})
}

func TestEonClient_ListRestoreAwsSubnets(t *testing.T) {
	t.Parallel()

	assertRestorePickerRequest(t, "/projects/project-id/restore-accounts/account-id/regions/us-east-1/aws-subnets-v2", url.Values{}, `{"subnets":[{"id":"subnet-1","vpc":"vpc-1","availabilityZone":"us-east-1a"}]}`, func(eonClient *EonClient) error {
		subnets, err := eonClient.ListRestoreAwsSubnets(context.Background(), "account-id", "us-east-1")
		if err != nil {
			return err
		}
		require.Len(t, subnets, 1)
		assert.Equal(t, "subnet-1", subnets[0].GetId())
		assert.Equal(t, "vpc-1", subnets[0].GetVpc())
		return nil
	})
}

func TestEonClient_ListRestoreAwsSecurityGroups(t *testing.T) {
	t.Parallel()

	assertRestorePickerRequest(
		t,
		"/projects/project-id/restore-accounts/account-id/regions/us-east-1/aws-security-groups",
		url.Values{"subnetId": {"subnet-1"}, "vpcId": {"vpc-1"}},
		`{"region":"us-east-1","securityGroups":[{"id":"sg-1","name":"web","vpc":"vpc-1"}]}`,
		func(eonClient *EonClient) error {
			vpcID := "vpc-1"
			subnetID := "subnet-1"
			securityGroups, err := eonClient.ListRestoreAwsSecurityGroups(context.Background(), "account-id", "us-east-1", &vpcID, &subnetID)
			if err != nil {
				return err
			}
			require.Len(t, securityGroups, 1)
			assert.Equal(t, "sg-1", securityGroups[0].GetId())
			assert.Equal(t, "vpc-1", securityGroups[0].GetVpc())
			return nil
		},
	)
}

func TestEonClient_ListRestoreAwsInstanceTypes(t *testing.T) {
	t.Parallel()

	assertRestorePickerRequest(
		t,
		"/projects/project-id/restore-accounts/account-id/regions/us-east-1/aws-instance-types",
		url.Values{"subnetId": {"subnet-1"}, "totalVolumeBytes": {"1073741824"}, "volumeCount": {"2"}},
		`{"instanceTypes":["m5.large"]}`,
		func(eonClient *EonClient) error {
			volumeCount := int32(2)
			totalVolumeBytes := int64(1073741824)
			instanceTypes, err := eonClient.ListRestoreAwsInstanceTypes(context.Background(), "account-id", "us-east-1", "subnet-1", &volumeCount, &totalVolumeBytes)
			if err != nil {
				return err
			}
			assert.Equal(t, []string{"m5.large"}, instanceTypes)
			return nil
		},
	)
}

func TestEonClient_ListRestoreEncryptionKeys(t *testing.T) {
	t.Parallel()

	assertRestorePickerRequest(
		t,
		"/projects/project-id/restore-accounts/account-id/regions/us-east-1/encryption-keys",
		url.Values{"includeAwsManagedKeys": {"true"}},
		`{"keys":[{"id":"key-1","arn":"arn:aws:kms:us-east-1:123:key/key-1","aliasName":"alias/key"}]}`,
		func(eonClient *EonClient) error {
			includeAwsManagedKeys := true
			keys, err := eonClient.ListRestoreEncryptionKeys(context.Background(), "account-id", "us-east-1", &includeAwsManagedKeys)
			if err != nil {
				return err
			}
			require.Len(t, keys, 1)
			assert.Equal(t, "key-1", keys[0].GetId())
			assert.Equal(t, "alias/key", keys[0].GetAliasName())
			return nil
		},
	)
}

func TestEonClient_ListRestoreS3Buckets(t *testing.T) {
	t.Parallel()

	assertRestorePickerRequest(
		t,
		"/projects/project-id/restore-accounts/account-id/aws/s3-buckets",
		url.Values{},
		`{"buckets":[{"bucketName":"restore-bucket","regionName":"us-east-1"}]}`,
		func(eonClient *EonClient) error {
			buckets, err := eonClient.ListRestoreS3Buckets(context.Background(), "account-id")
			if err != nil {
				return err
			}
			require.Len(t, buckets, 1)
			assert.Equal(t, "restore-bucket", buckets[0].GetBucketName())
			assert.Equal(t, "us-east-1", buckets[0].GetRegionName())
			return nil
		},
	)
}

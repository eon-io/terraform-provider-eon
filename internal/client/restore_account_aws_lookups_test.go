package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const s3BucketsPath = "/api/projects/project-1/restore-accounts/account-1/aws/s3-buckets"

const s3BucketsResponse = `{
	"buckets": [
		{"bucketName": "bucket-a", "regionName": "us-east-1"},
		{"bucketName": "bucket-b", "regionName": "us-west-2"}
	]
}`

func TestListS3Buckets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantLen    int
	}{
		{name: "success", statusCode: http.StatusOK, body: s3BucketsResponse, wantLen: 2},
		{name: "empty", statusCode: http.StatusOK, body: `{"buckets":[]}`, wantLen: 0},
		{name: "not found", statusCode: http.StatusNotFound, body: `{"message":"not found"}`, wantErr: true},
		{name: "server error", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, s3BucketsPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			buckets, err := c.ListS3Buckets(context.Background(), "account-1")
			if tt.wantErr {
				require.Error(t, err)
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.statusCode, apiErr.StatusCode)
				assert.Nil(t, buckets)
				return
			}
			require.NoError(t, err)
			require.Len(t, buckets, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, "bucket-a", buckets[0].GetBucketName())
				assert.Equal(t, "us-east-1", buckets[0].GetRegionName())
			}
		})
	}
}

const instanceTypesPath = "/api/projects/project-1/restore-accounts/account-1/regions/us-east-1/aws-instance-types"

const instanceTypesResponse = `{
	"instanceTypes": ["m5.large", "m5.xlarge"],
	"instantRestoreMapping": {"m5.large": "i3.large"}
}`

func TestListAwsInstanceTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK, body: instanceTypesResponse},
		{name: "bad request", statusCode: http.StatusBadRequest, body: `{"message":"invalid"}`, wantErr: true},
		{name: "server error", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, instanceTypesPath, r.URL.Path)
				assert.Equal(t, "subnet-123", r.URL.Query().Get("subnetId"))
				assert.Equal(t, "2", r.URL.Query().Get("volumeCount"))
				assert.Equal(t, "1024", r.URL.Query().Get("totalVolumeBytes"))
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			volumeCount := int32(2)
			totalVolumeBytes := int64(1024)
			resp, err := c.ListAwsInstanceTypes(context.Background(), "account-1", "us-east-1", "subnet-123", ListAwsInstanceTypesOptions{
				VolumeCount:      &volumeCount,
				TotalVolumeBytes: &totalVolumeBytes,
			})
			if tt.wantErr {
				require.Error(t, err)
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.statusCode, apiErr.StatusCode)
				assert.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, []string{"m5.large", "m5.xlarge"}, resp.GetInstanceTypes())
			assert.Equal(t, map[string]string{"m5.large": "i3.large"}, resp.GetInstantRestoreMapping())
		})
	}
}

func TestListAwsInstanceTypes_WithoutOptionalParams(t *testing.T) {
	t.Parallel()

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "subnet-123", r.URL.Query().Get("subnetId"))
		assert.Empty(t, r.URL.Query().Get("volumeCount"))
		assert.Empty(t, r.URL.Query().Get("totalVolumeBytes"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(instanceTypesResponse)),
		}, nil
	}))

	resp, err := c.ListAwsInstanceTypes(context.Background(), "account-1", "us-east-1", "subnet-123", ListAwsInstanceTypesOptions{})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

const securityGroupsPath = "/api/projects/project-1/restore-accounts/account-1/regions/us-east-1/aws-security-groups"

const securityGroupsResponse = `{
	"region": "us-east-1",
	"securityGroups": [
		{"id": "sg-1", "name": "default", "vpc": "vpc-1"}
	]
}`

func TestListAwsSecurityGroups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK, body: securityGroupsResponse},
		{name: "not found", statusCode: http.StatusNotFound, body: `{"message":"not found"}`, wantErr: true},
		{name: "server error", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, securityGroupsPath, r.URL.Path)
				assert.Equal(t, "vpc-1", r.URL.Query().Get("vpcId"))
				assert.Equal(t, "subnet-1", r.URL.Query().Get("subnetId"))
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			vpcID := "vpc-1"
			subnetID := "subnet-1"
			resp, err := c.ListAwsSecurityGroups(context.Background(), "account-1", "us-east-1", ListAwsSecurityGroupsOptions{
				VpcId:    &vpcID,
				SubnetId: &subnetID,
			})
			if tt.wantErr {
				require.Error(t, err)
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.statusCode, apiErr.StatusCode)
				assert.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, "us-east-1", resp.GetRegion())
			require.Len(t, resp.GetSecurityGroups(), 1)
			assert.Equal(t, "sg-1", resp.GetSecurityGroups()[0].GetId())
		})
	}
}

func TestListAwsSecurityGroups_WithoutOptionalParams(t *testing.T) {
	t.Parallel()

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Empty(t, r.URL.Query().Get("vpcId"))
		assert.Empty(t, r.URL.Query().Get("subnetId"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(securityGroupsResponse)),
		}, nil
	}))

	resp, err := c.ListAwsSecurityGroups(context.Background(), "account-1", "us-east-1", ListAwsSecurityGroupsOptions{})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

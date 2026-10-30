package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const s3BucketsPath = "/api/projects/project-1/restore-accounts/acct-1/aws/s3-buckets"
const awsInstanceTypesPath = "/api/projects/project-1/restore-accounts/acct-1/regions/us-east-1/aws-instance-types"
const awsSecurityGroupsPath = "/api/projects/project-1/restore-accounts/acct-1/regions/us-east-1/aws-security-groups"

const s3BucketsResponse = `{
	"buckets": [
		{"bucketName": "bucket-a", "regionName": "us-east-1"},
		{"bucketName": "bucket-b", "regionName": "us-west-2"}
	]
}`

const instanceTypesResponse = `{
	"instanceTypes": ["m5.large", "m5.xlarge"],
	"instantRestoreMapping": {"m5.large": "m5.large-instant"}
}`

const securityGroupsResponse = `{
	"region": "us-east-1",
	"securityGroups": [
		{"id": "sg-1", "name": "default", "vpc": "vpc-1"}
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

			buckets, err := c.ListS3Buckets(context.Background(), "acct-1")
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, buckets)
				var apiErr *APIError
				require.True(t, errors.As(err, &apiErr), "expected *APIError, got %T", err)
				assert.Equal(t, tt.statusCode, apiErr.StatusCode)
				return
			}
			require.NoError(t, err)
			assert.Len(t, buckets, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, "bucket-a", buckets[0].GetBucketName())
				assert.Equal(t, "us-east-1", buckets[0].GetRegionName())
			}
		})
	}
}

func TestListAwsInstanceTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		statusCode       int
		body             string
		volumeCount      *int32
		totalVolumeBytes *int64
		wantErr          bool
	}{
		{name: "success without optional params", statusCode: http.StatusOK, body: instanceTypesResponse},
		{
			name:             "success with optional params",
			statusCode:       http.StatusOK,
			body:             instanceTypesResponse,
			volumeCount:      int32Ptr(3),
			totalVolumeBytes: int64Ptr(1024),
		},
		{name: "bad request", statusCode: http.StatusBadRequest, body: `{"message":"subnetId is required"}`, wantErr: true},
		{name: "server error", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, awsInstanceTypesPath, r.URL.Path)
				assert.Equal(t, "subnet-1", r.URL.Query().Get("subnetId"))
				if tt.volumeCount != nil {
					assert.Equal(t, "3", r.URL.Query().Get("volumeCount"))
				}
				if tt.totalVolumeBytes != nil {
					assert.Equal(t, "1024", r.URL.Query().Get("totalVolumeBytes"))
				}
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			result, err := c.ListAwsInstanceTypes(context.Background(), "acct-1", "us-east-1", "subnet-1", tt.volumeCount, tt.totalVolumeBytes)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
				var apiErr *APIError
				require.True(t, errors.As(err, &apiErr), "expected *APIError, got %T", err)
				assert.Equal(t, tt.statusCode, apiErr.StatusCode)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, []string{"m5.large", "m5.xlarge"}, result.GetInstanceTypes())
			assert.Equal(t, map[string]string{"m5.large": "m5.large-instant"}, result.GetInstantRestoreMapping())
		})
	}
}

func TestListAwsSecurityGroups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		vpcId      *string
		subnetId   *string
		wantErr    bool
	}{
		{name: "success without filters", statusCode: http.StatusOK, body: securityGroupsResponse},
		{
			name:       "success with filters",
			statusCode: http.StatusOK,
			body:       securityGroupsResponse,
			vpcId:      stringPtr("vpc-1"),
			subnetId:   stringPtr("subnet-1"),
		},
		{name: "not found", statusCode: http.StatusNotFound, body: `{"message":"not found"}`, wantErr: true},
		{name: "server error", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, awsSecurityGroupsPath, r.URL.Path)
				if tt.vpcId != nil {
					assert.Equal(t, *tt.vpcId, r.URL.Query().Get("vpcId"))
				} else {
					assert.Empty(t, r.URL.Query().Get("vpcId"))
				}
				if tt.subnetId != nil {
					assert.Equal(t, *tt.subnetId, r.URL.Query().Get("subnetId"))
				} else {
					assert.Empty(t, r.URL.Query().Get("subnetId"))
				}
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			result, err := c.ListAwsSecurityGroups(context.Background(), "acct-1", "us-east-1", tt.vpcId, tt.subnetId)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
				var apiErr *APIError
				require.True(t, errors.As(err, &apiErr), "expected *APIError, got %T", err)
				assert.Equal(t, tt.statusCode, apiErr.StatusCode)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, "us-east-1", result.GetRegion())
			require.Len(t, result.GetSecurityGroups(), 1)
			assert.Equal(t, "sg-1", result.GetSecurityGroups()[0].GetId())
			assert.Equal(t, "default", result.GetSecurityGroups()[0].GetName())
			assert.Equal(t, "vpc-1", result.GetSecurityGroups()[0].GetVpc())
		})
	}
}

func int32Ptr(v int32) *int32    { return &v }
func int64Ptr(v int64) *int64    { return &v }
func stringPtr(v string) *string { return &v }

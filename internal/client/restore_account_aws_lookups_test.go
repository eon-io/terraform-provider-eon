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

func TestListS3Buckets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantLen    int
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       `{"buckets": [{"bucketName": "bucket-1", "regionName": "us-east-1"}]}`,
			wantLen:    1,
		},
		{
			name:       "empty list",
			statusCode: http.StatusOK,
			body:       `{"buckets": []}`,
			wantLen:    0,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			body:       `{"message":"not found"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/projects/project-1/restore-accounts/acct-1/aws/s3-buckets", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			result, err := c.ListS3Buckets(context.Background(), "acct-1")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			require.Len(t, result, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, "bucket-1", result[0].GetBucketName())
				assert.Equal(t, "us-east-1", result[0].GetRegionName())
			}
		})
	}
}

func TestListAwsInstanceTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       `{"instanceTypes": ["m5.large", "m5.xlarge"], "instantRestoreMapping": {"m5.large": "m5.xlarge"}}`,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			body:       `{"message":"invalid"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/projects/project-1/restore-accounts/acct-1/regions/us-east-1/aws-instance-types", r.URL.Path)
				assert.Equal(t, "subnet-1", r.URL.Query().Get("subnetId"))
				assert.Equal(t, "3", r.URL.Query().Get("volumeCount"))
				assert.Equal(t, "1024", r.URL.Query().Get("totalVolumeBytes"))
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			volumeCount := int32(3)
			totalVolumeBytes := int64(1024)
			result, err := c.ListAwsInstanceTypes(context.Background(), "acct-1", "us-east-1", "subnet-1", ListAwsInstanceTypesOptions{
				VolumeCount:      &volumeCount,
				TotalVolumeBytes: &totalVolumeBytes,
			})
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, []string{"m5.large", "m5.xlarge"}, result.GetInstanceTypes())
			assert.Equal(t, map[string]string{"m5.large": "m5.xlarge"}, result.GetInstantRestoreMapping())
		})
	}
}

func TestListAwsInstanceTypes_WithoutOptionalFilters(t *testing.T) {
	t.Parallel()

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "subnet-1", r.URL.Query().Get("subnetId"))
		assert.Empty(t, r.URL.Query().Get("volumeCount"))
		assert.Empty(t, r.URL.Query().Get("totalVolumeBytes"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"instanceTypes": []}`)),
		}, nil
	}))

	result, err := c.ListAwsInstanceTypes(context.Background(), "acct-1", "us-east-1", "subnet-1", ListAwsInstanceTypesOptions{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.GetInstanceTypes())
}

func TestListAwsSecurityGroups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       `{"region": "us-east-1", "securityGroups": [{"id": "sg-1", "name": "default", "vpc": "vpc-1"}]}`,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			body:       `{"message":"not found"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/projects/project-1/restore-accounts/acct-1/regions/us-east-1/aws-security-groups", r.URL.Path)
				assert.Equal(t, "vpc-1", r.URL.Query().Get("vpcId"))
				assert.Equal(t, "subnet-1", r.URL.Query().Get("subnetId"))
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			vpcId := "vpc-1"
			subnetId := "subnet-1"
			result, err := c.ListAwsSecurityGroups(context.Background(), "acct-1", "us-east-1", ListAwsSecurityGroupsOptions{
				VpcId:    &vpcId,
				SubnetId: &subnetId,
			})
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
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

func TestListAwsSecurityGroups_WithoutOptionalFilters(t *testing.T) {
	t.Parallel()

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Empty(t, r.URL.Query().Get("vpcId"))
		assert.Empty(t, r.URL.Query().Get("subnetId"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"region": "us-east-1", "securityGroups": []}`)),
		}, nil
	}))

	result, err := c.ListAwsSecurityGroups(context.Background(), "acct-1", "us-east-1", ListAwsSecurityGroupsOptions{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.GetSecurityGroups())
}

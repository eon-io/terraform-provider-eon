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
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       `{"buckets":[{"bucketName":"restore-bucket","regionName":"us-east-1"}]}`,
		},
		{
			name:       "failure",
			statusCode: http.StatusInternalServerError,
			body:       `{"message":"boom"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/projects/project-1/restore-accounts/restore-acct-1/aws/s3-buckets", r.URL.Path)
				return response(tt.statusCode, tt.body), nil
			}))

			buckets, err := c.ListS3Buckets(context.Background(), "restore-acct-1")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, buckets)
				return
			}

			require.NoError(t, err)
			require.Len(t, buckets, 1)
			assert.Equal(t, "restore-bucket", buckets[0].GetBucketName())
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
			body:       `{"instanceTypes":["m5.large"],"instantRestoreMapping":{"m5.large":"i3.large"}}`,
		},
		{
			name:       "failure",
			statusCode: http.StatusBadRequest,
			body:       `{"message":"invalid subnet"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			volumeCount := int32(2)
			totalVolumeBytes := int64(107374182400)
			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/projects/project-1/restore-accounts/restore-acct-1/regions/us-east-1/aws-instance-types", r.URL.Path)
				assert.Equal(t, "subnet-1", r.URL.Query().Get("subnetId"))
				assert.Equal(t, "2", r.URL.Query().Get("volumeCount"))
				assert.Equal(t, "107374182400", r.URL.Query().Get("totalVolumeBytes"))
				return response(tt.statusCode, tt.body), nil
			}))

			instanceTypes, err := c.ListAwsInstanceTypes(
				context.Background(),
				"restore-acct-1",
				"us-east-1",
				"subnet-1",
				&volumeCount,
				&totalVolumeBytes,
			)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, instanceTypes)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, []string{"m5.large"}, instanceTypes.GetInstanceTypes())
			assert.Equal(t, "i3.large", instanceTypes.GetInstantRestoreMapping()["m5.large"])
		})
	}
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
			body:       `{"region":"us-east-1","securityGroups":[{"id":"sg-1","name":"restore","vpc":"vpc-1"}]}`,
		},
		{
			name:       "failure",
			statusCode: http.StatusNotFound,
			body:       `{"message":"not found"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vpcID := "vpc-1"
			subnetID := "subnet-1"
			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/projects/project-1/restore-accounts/restore-acct-1/regions/us-east-1/aws-security-groups", r.URL.Path)
				assert.Equal(t, "vpc-1", r.URL.Query().Get("vpcId"))
				assert.Equal(t, "subnet-1", r.URL.Query().Get("subnetId"))
				return response(tt.statusCode, tt.body), nil
			}))

			securityGroups, err := c.ListAwsSecurityGroups(
				context.Background(),
				"restore-acct-1",
				"us-east-1",
				&vpcID,
				&subnetID,
			)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, securityGroups)
				return
			}

			require.NoError(t, err)
			require.Len(t, securityGroups.GetSecurityGroups(), 1)
			assert.Equal(t, "sg-1", securityGroups.GetSecurityGroups()[0].GetId())
		})
	}
}

func response(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const restoreAccountS3BucketsPath = "/api/projects/project-1/restore-accounts/ra-1/aws/s3-buckets"
const restoreAccountInstanceTypesPath = "/api/projects/project-1/restore-accounts/ra-1/regions/us-east-1/aws-instance-types"
const restoreAccountSecurityGroupsPath = "/api/projects/project-1/restore-accounts/ra-1/regions/us-east-1/aws-security-groups"

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestListRestoreAccountS3Buckets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		statusCode     int
		body           string
		wantErr        bool
		wantStatusCode int
		wantBuckets    []string
	}{
		{
			name:        "success",
			statusCode:  http.StatusOK,
			body:        `{"buckets":[{"bucketName":"restore-a","regionName":"us-east-1"},{"bucketName":"restore-b","regionName":"eu-west-1"}]}`,
			wantBuckets: []string{"restore-a", "restore-b"},
		},
		{name: "empty list", statusCode: http.StatusOK, body: `{"buckets":[]}`, wantBuckets: []string{}},
		{name: "not found", statusCode: http.StatusNotFound, body: `{"message":"not found"}`, wantErr: true, wantStatusCode: http.StatusNotFound},
		{name: "server error", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantErr: true, wantStatusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, restoreAccountS3BucketsPath, r.URL.Path)
				return jsonResponse(tt.statusCode, tt.body), nil
			}))

			buckets, err := c.ListRestoreAccountS3Buckets(context.Background(), "ra-1")
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, buckets)
				var apiErr *APIError
				require.True(t, errors.As(err, &apiErr), "expected *APIError, got %T", err)
				assert.Equal(t, tt.wantStatusCode, apiErr.StatusCode)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, buckets)
			names := make([]string, 0, len(buckets))
			for _, b := range buckets {
				names = append(names, b.GetBucketName())
			}
			assert.Equal(t, tt.wantBuckets, names)
		})
	}
}

func TestListRestoreAccountS3Buckets_TransportError(t *testing.T) {
	t.Parallel()

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	}))

	buckets, err := c.ListRestoreAccountS3Buckets(context.Background(), "ra-1")
	require.Error(t, err)
	assert.Nil(t, buckets)
	assert.Contains(t, err.Error(), "failed to list restore account S3 buckets")
}

func TestListRestoreAccountAwsInstanceTypes(t *testing.T) {
	t.Parallel()

	volumeCount := int32(3)
	totalVolumeBytes := int64(107374182400)

	tests := []struct {
		name           string
		opts           ListAwsInstanceTypesOptions
		statusCode     int
		body           string
		wantQuery      map[string]string
		wantErr        bool
		wantStatusCode int
		wantTypes      []string
		wantMapping    map[string]string
	}{
		{
			name:        "success without sizing hints",
			statusCode:  http.StatusOK,
			body:        `{"instanceTypes":["t3.micro","m5.large"],"instantRestoreMapping":{"m5.large":"m5.xlarge"}}`,
			wantQuery:   map[string]string{"subnetId": "subnet-1"},
			wantTypes:   []string{"t3.micro", "m5.large"},
			wantMapping: map[string]string{"m5.large": "m5.xlarge"},
		},
		{
			name:       "success with sizing hints",
			opts:       ListAwsInstanceTypesOptions{VolumeCount: &volumeCount, TotalVolumeBytes: &totalVolumeBytes},
			statusCode: http.StatusOK,
			body:       `{"instanceTypes":["r5.large"]}`,
			wantQuery: map[string]string{
				"subnetId":         "subnet-1",
				"volumeCount":      "3",
				"totalVolumeBytes": "107374182400",
			},
			wantTypes: []string{"r5.large"},
		},
		{name: "bad request", statusCode: http.StatusBadRequest, body: `{"message":"invalid subnet"}`, wantQuery: map[string]string{"subnetId": "subnet-1"}, wantErr: true, wantStatusCode: http.StatusBadRequest},
		{name: "server error", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantQuery: map[string]string{"subnetId": "subnet-1"}, wantErr: true, wantStatusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, restoreAccountInstanceTypesPath, r.URL.Path)
				query := r.URL.Query()
				assert.Len(t, query, len(tt.wantQuery))
				for k, v := range tt.wantQuery {
					assert.Equal(t, v, query.Get(k), "query param %s", k)
				}
				return jsonResponse(tt.statusCode, tt.body), nil
			}))

			resp, err := c.ListRestoreAccountAwsInstanceTypes(context.Background(), "ra-1", "us-east-1", "subnet-1", tt.opts)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, resp)
				var apiErr *APIError
				require.True(t, errors.As(err, &apiErr), "expected *APIError, got %T", err)
				assert.Equal(t, tt.wantStatusCode, apiErr.StatusCode)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.wantTypes, resp.GetInstanceTypes())
			assert.Equal(t, tt.wantMapping, resp.GetInstantRestoreMapping())
		})
	}
}

func TestListRestoreAccountAwsSecurityGroups(t *testing.T) {
	t.Parallel()

	vpcID := "vpc-1"
	subnetID := "subnet-1"

	tests := []struct {
		name           string
		vpcId          *string
		subnetId       *string
		statusCode     int
		body           string
		wantQuery      map[string]string
		wantErr        bool
		wantStatusCode int
		wantIDs        []string
	}{
		{
			name:       "success without filters",
			statusCode: http.StatusOK,
			body:       `{"region":"us-east-1","securityGroups":[{"id":"sg-1","name":"default","vpc":"vpc-1"},{"id":"sg-2","name":"web","vpc":"vpc-2"}]}`,
			wantQuery:  map[string]string{},
			wantIDs:    []string{"sg-1", "sg-2"},
		},
		{
			name:       "success with filters",
			vpcId:      &vpcID,
			subnetId:   &subnetID,
			statusCode: http.StatusOK,
			body:       `{"region":"us-east-1","securityGroups":[{"id":"sg-1","name":"default","vpc":"vpc-1"}]}`,
			wantQuery:  map[string]string{"vpcId": "vpc-1", "subnetId": "subnet-1"},
			wantIDs:    []string{"sg-1"},
		},
		{name: "empty list", statusCode: http.StatusOK, body: `{"region":"us-east-1","securityGroups":[]}`, wantQuery: map[string]string{}, wantIDs: []string{}},
		{name: "not found", statusCode: http.StatusNotFound, body: `{"message":"not found"}`, wantQuery: map[string]string{}, wantErr: true, wantStatusCode: http.StatusNotFound},
		{name: "server error", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantQuery: map[string]string{}, wantErr: true, wantStatusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, restoreAccountSecurityGroupsPath, r.URL.Path)
				query := r.URL.Query()
				assert.Len(t, query, len(tt.wantQuery))
				for k, v := range tt.wantQuery {
					assert.Equal(t, v, query.Get(k), "query param %s", k)
				}
				return jsonResponse(tt.statusCode, tt.body), nil
			}))

			groups, err := c.ListRestoreAccountAwsSecurityGroups(context.Background(), "ra-1", "us-east-1", tt.vpcId, tt.subnetId)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, groups)
				var apiErr *APIError
				require.True(t, errors.As(err, &apiErr), "expected *APIError, got %T", err)
				assert.Equal(t, tt.wantStatusCode, apiErr.StatusCode)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, groups)
			ids := make([]string, 0, len(groups))
			for _, g := range groups {
				ids = append(ids, g.GetId())
			}
			assert.Equal(t, tt.wantIDs, ids)
		})
	}
}

func TestMockRestoreAccountTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		shouldFail bool
	}{
		{name: "success", shouldFail: false},
		{name: "failure", shouldFail: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewMockEonClient()
			m.ShouldFailListRestoreAccountS3Buckets = tt.shouldFail
			m.ShouldFailListRestoreAccountInstanceTypes = tt.shouldFail
			m.ShouldFailListRestoreAccountSecurityGroups = tt.shouldFail
			m.AddMockRestoreAccountS3Bucket("ra-1", *externalEonSdkAPI.NewS3Bucket("restore-a", "us-east-1"))
			instanceTypes := externalEonSdkAPI.NewListInstanceTypesResponse()
			instanceTypes.SetInstanceTypes([]string{"t3.micro"})
			m.SetMockRestoreAccountInstanceTypes("ra-1", "us-east-1", instanceTypes)
			m.AddMockRestoreAccountSecurityGroup("ra-1", "us-east-1", *externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("default", "sg-1", "vpc-1"))
			m.AddMockRestoreAccountSecurityGroup("ra-1", "us-east-1", *externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("web", "sg-2", "vpc-2"))

			buckets, bucketsErr := m.ListRestoreAccountS3Buckets(context.Background(), "ra-1")
			types, typesErr := m.ListRestoreAccountAwsInstanceTypes(context.Background(), "ra-1", "us-east-1", "subnet-1", ListAwsInstanceTypesOptions{})
			vpcID := "vpc-2"
			groups, groupsErr := m.ListRestoreAccountAwsSecurityGroups(context.Background(), "ra-1", "us-east-1", &vpcID, nil)

			assert.Equal(t, 1, m.ListRestoreAccountS3BucketsCalls)
			assert.Equal(t, 1, m.ListRestoreAccountInstanceTypesCalls)
			assert.Equal(t, 1, m.ListRestoreAccountSecurityGroupsCalls)

			if tt.shouldFail {
				assert.Error(t, bucketsErr)
				assert.Error(t, typesErr)
				assert.Error(t, groupsErr)
				return
			}
			require.NoError(t, bucketsErr)
			require.NoError(t, typesErr)
			require.NoError(t, groupsErr)
			assert.Len(t, buckets, 1)
			assert.Equal(t, []string{"t3.micro"}, types.GetInstanceTypes())
			require.Len(t, groups, 1)
			assert.Equal(t, "sg-2", groups[0].GetId())

			unknown, err := m.ListRestoreAccountAwsInstanceTypes(context.Background(), "ra-1", "eu-west-1", "subnet-1", ListAwsInstanceTypesOptions{})
			require.NoError(t, err)
			assert.Empty(t, unknown.GetInstanceTypes())
		})
	}
}

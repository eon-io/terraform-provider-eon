package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const metricsConfigPath = "/api/v1/projects/project-1/restore-accounts/account-1/metrics-config"

const metricsConfigResponse = `{
	"restoreAccountConfig": {
		"restoreAccountId": "account-1",
		"enabled": true,
		"destination": {
			"aws": {"region": "us-east-1"}
		}
	}
}`

func TestGetRestoreAccountMetricsConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantRegion string
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       metricsConfigResponse,
			wantRegion: "us-east-1",
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
				assert.Equal(t, metricsConfigPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			config, err := c.GetRestoreAccountMetricsConfig(context.Background(), "account-1")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "account-1", config.GetRestoreAccountId())
			assert.True(t, config.GetEnabled())
			dest := config.GetDestination()
			aws := dest.GetAws()
			assert.Equal(t, tt.wantRegion, aws.GetRegion())
		})
	}
}

func TestEnableRestoreAccountMetricsConfig(t *testing.T) {
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
			body:       metricsConfigResponse,
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
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, metricsConfigPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			aws := externalEonSdkAPI.NewAwsAccountMetricsDestination()
			aws.SetRegion("us-east-1")
			req := externalEonSdkAPI.EnableRestoreAccountMetricsConfigRequest{}
			req.SetAws(*aws)

			config, err := c.EnableRestoreAccountMetricsConfig(context.Background(), "account-1", req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "account-1", config.GetRestoreAccountId())
		})
	}
}

func TestDisableRestoreAccountMetricsConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusNoContent},
		{name: "success ok", statusCode: http.StatusOK},
		{name: "not found", statusCode: http.StatusNotFound, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, metricsConfigPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			}))

			err := c.DisableRestoreAccountMetricsConfig(context.Background(), "account-1")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestHoldSnapshot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK},
		{name: "bad request", statusCode: http.StatusBadRequest, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/snapshots/snap-1/hold", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			}))

			req := externalEonSdkAPI.NewHoldSnapshotRequest()
			req.SetDescription("compliance hold")
			err := c.HoldSnapshot(context.Background(), "snap-1", *req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestRemoveSnapshotHold(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK},
		{name: "not found", statusCode: http.StatusNotFound, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/snapshots/snap-1/remove-hold", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			}))

			err := c.RemoveSnapshotHold(context.Background(), "snap-1")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestStartDynamoDBTableRestore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantJobID  string
	}{
		{
			name:       "success",
			statusCode: http.StatusAccepted,
			body:       `{"jobId":"job-dynamo-1","actionApprovalRequest":null}`,
			wantJobID:  "job-dynamo-1",
		},
		{
			name:       "mpa intercepted",
			statusCode: http.StatusCreated,
			body:       `{"actionApprovalRequest":{"id":"mpa-1"}}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/resources/res-1/snapshots/snap-1/restore-dynamo-db-table", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			req := externalEonSdkAPI.RestoreDynamoDBTableRequest{
				RestoreAccountId: "acct-1",
				Destination: externalEonSdkAPI.DynamodbTableRestoreDestination{
					AwsDynamodb: &externalEonSdkAPI.AwsDynamoDBDestination{
						RestoreRegion: "us-east-1",
						RestoredName:  "restored-table",
					},
				},
			}
			jobID, err := c.StartDynamoDBTableRestore(context.Background(), "res-1", "snap-1", req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, jobID)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantJobID, jobID)
		})
	}
}

func TestStartAzureDiskRestore(t *testing.T) {
	t.Parallel()

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/projects/project-1/resources/res-1/snapshots/snap-1/restore-azure-disk", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"jobId":"job-azure-disk-1","actionApprovalRequest":null}`)),
		}, nil
	}))

	req := externalEonSdkAPI.RestoreAzureDiskRequest{
		ProviderDiskId:   "disk-1",
		RestoreAccountId: "acct-1",
		Destination: externalEonSdkAPI.AzureDiskRestoreDestination{
			AzureDisk: &externalEonSdkAPI.AzureDiskTarget{
				Region:            "eastus",
				ResourceGroupName: "rg-1",
				Settings: externalEonSdkAPI.AzureDiskSettings{
					Name: "disk-restored",
					Type: "Premium_LRS",
					Tier: "P10",
				},
			},
		},
	}
	jobID, err := c.StartAzureDiskRestore(context.Background(), "res-1", "snap-1", req)
	require.NoError(t, err)
	assert.Equal(t, "job-azure-disk-1", jobID)
}

func TestStartEbsSnapshotRestore(t *testing.T) {
	t.Parallel()

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/projects/project-1/resources/res-1/snapshots/snap-1/convert-ec2-ebs-snapshot", r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"jobId":"job-ebs-snap-1","actionApprovalRequest":null}`)),
		}, nil
	}))

	req := externalEonSdkAPI.RestoreVolumeToEbsSnapshotRequest{
		ProviderVolumeId: "vol-1",
		RestoreAccountId: "acct-1",
		Destination: externalEonSdkAPI.EbsSnapshotRestoreDestination{
			AwsEbs: &externalEonSdkAPI.EbsSnapshotTarget{
				Region:                  "us-east-1",
				SnapshotEncryptionKeyId: "alias/aws/ebs",
			},
		},
	}
	jobID, err := c.StartEbsSnapshotRestore(context.Background(), "res-1", "snap-1", req)
	require.NoError(t, err)
	assert.Equal(t, "job-ebs-snap-1", jobID)
}

const sourceMetricsConfigPath = "/api/v1/projects/project-1/source-accounts/account-1/metrics-config"

const sourceMetricsConfigResponse = `{
	"sourceAccountConfig": {
		"sourceAccountId": "account-1",
		"enabled": true,
		"destination": {
			"aws": {"region": "us-east-1"}
		}
	}
}`

func TestGetSourceAccountMetricsConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantRegion string
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       sourceMetricsConfigResponse,
			wantRegion: "us-east-1",
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
				assert.Equal(t, sourceMetricsConfigPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			config, err := c.GetSourceAccountMetricsConfig(context.Background(), "account-1")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "account-1", config.GetSourceAccountId())
			assert.True(t, config.GetEnabled())
			dest := config.GetDestination()
			aws := dest.GetAws()
			assert.Equal(t, tt.wantRegion, aws.GetRegion())
		})
	}
}

func TestEnableSourceAccountMetricsConfig(t *testing.T) {
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
			body:       sourceMetricsConfigResponse,
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
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, sourceMetricsConfigPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			aws := externalEonSdkAPI.NewAwsAccountMetricsDestination()
			aws.SetRegion("us-east-1")
			req := externalEonSdkAPI.EnableSourceAccountMetricsConfigRequest{}
			req.SetAws(*aws)

			config, err := c.EnableSourceAccountMetricsConfig(context.Background(), "account-1", req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "account-1", config.GetSourceAccountId())
		})
	}
}

func TestDisableSourceAccountMetricsConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusNoContent},
		{name: "success ok", statusCode: http.StatusOK},
		{name: "not found", statusCode: http.StatusNotFound, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, sourceMetricsConfigPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			}))

			err := c.DisableSourceAccountMetricsConfig(context.Background(), "account-1")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverrideEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		want       string
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       `{"environment": "PROD"}`,
			want:       "PROD",
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
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/resources/res-1/environments", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var got map[string]string
				require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
				assert.Equal(t, "PROD", got["environment"])

				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			result, err := c.OverrideEnvironment(context.Background(), "res-1", "PROD")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, result)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRemoveEnvironmentOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "success no content",
			statusCode: http.StatusNoContent,
			body:       "",
		},
		{
			name:       "success ok",
			statusCode: http.StatusOK,
			body:       `{}`,
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
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/resources/res-1/environments", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			err := c.RemoveEnvironmentOverride(context.Background(), "res-1")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestListResourceSnapshots(t *testing.T) {
	t.Parallel()

	created := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

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
			body: `{
				"snapshots": [
					{
						"id": "snap-1",
						"createdTime": "` + created.Format(time.RFC3339) + `",
						"resourceId": "res-1"
					}
				],
				"totalCount": 1
			}`,
			wantLen: 1,
		},
		{
			name:       "empty list",
			statusCode: http.StatusOK,
			body:       `{"snapshots": [], "totalCount": 0}`,
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
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/resources/res-1/snapshots", r.URL.Path)
				assert.Equal(t, "100", r.URL.Query().Get("pageSize"))
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			filters := externalEonSdkAPI.NewSnapshotFilterConditions()
			dateFilters := externalEonSdkAPI.NewSnapshotDateFilters()
			dateFilters.SetStartDate("2024-01-01")
			filters.SetPointInTime(*dateFilters)

			result, err := c.ListResourceSnapshots(context.Background(), "res-1", filters)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			require.Len(t, result, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, "snap-1", result[0].GetId())
				assert.Equal(t, "res-1", result[0].GetResourceId())
			}
		})
	}
}

func TestListResources(t *testing.T) {
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
			body: `{
				"resources": [
					{
						"id": "res-1",
						"backupStatus": "PROTECTED",
						"providerResourceId": "i-1",
						"resourceName": "demo",
						"providerAccountId": "123456789012",
						"snapshotStorage": {},
						"sourceStorage": {},
						"tags": {},
						"cloudProvider": "AWS",
						"resourceType": "AWS_EC2",
						"region": "us-east-1"
					}
				],
				"totalCount": 1
			}`,
			wantLen: 1,
		},
		{
			name:       "empty list",
			statusCode: http.StatusOK,
			body:       `{"resources": [], "totalCount": 0}`,
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
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/resources", r.URL.Path)
				assert.Equal(t, "100", r.URL.Query().Get("pageSize"))
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			filters := externalEonSdkAPI.NewInventoryFilterConditions()
			providerFilters := externalEonSdkAPI.NewResourceIdFilters()
			providerFilters.SetIn([]string{"i-1"})
			filters.SetProviderResourceId(*providerFilters)

			result, err := c.ListResources(context.Background(), filters)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			require.Len(t, result, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, "res-1", result[0].GetId())
				assert.Equal(t, "i-1", result[0].GetProviderResourceId())
			}
		})
	}
}

package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListPermissions(t *testing.T) {
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
				"permissions": [
					{"permissionType": "inventory.view", "description": "View inventory", "allowConditions": true},
					{"permissionType": "jobs.view", "description": "View jobs", "allowConditions": false}
				]
			}`,
			wantLen: 2,
		},
		{
			name:       "empty list",
			statusCode: http.StatusOK,
			body:       `{"permissions": []}`,
			wantLen:    0,
		},
		{
			name:       "server error",
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
				assert.Equal(t, "/api/v1/permissions", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			result, err := c.ListPermissions(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			require.Len(t, result, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, externalEonSdkAPI.INVENTORY_VIEW, result[0].GetPermissionType())
				assert.Equal(t, "View inventory", result[0].GetDescription())
				assert.True(t, result[0].GetAllowConditions())
			}
		})
	}
}

func TestExcludeResourceFromBackup(t *testing.T) {
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
			body:       `{"isExcluded": true}`,
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
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/resources/res-1/exclude", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			err := c.ExcludeResourceFromBackup(context.Background(), "res-1")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestCancelResourceBackupExclusion(t *testing.T) {
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
			body:       `{"isExclusionCanceled": true}`,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"message":"boom"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/resources/res-1/include", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			err := c.CancelResourceBackupExclusion(context.Background(), "res-1")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestOverrideDataClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		want       []string
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       `{"dataClasses": ["PII", "PCI"]}`,
			want:       []string{"PII", "PCI"},
		},
		{
			name:       "empty response",
			statusCode: http.StatusOK,
			body:       `{}`,
			want:       []string{},
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
				assert.Equal(t, "/api/v1/projects/project-1/resources/res-1/data-classifications", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var got map[string][]string
				require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
				assert.Equal(t, []string{"PII", "PCI"}, got["dataClasses"])

				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			result, err := c.OverrideDataClasses(context.Background(), "res-1", []string{"PII", "PCI"})
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRemoveDataClassesOverride(t *testing.T) {
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
				assert.Equal(t, "/api/v1/projects/project-1/resources/res-1/data-classifications", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			err := c.RemoveDataClassesOverride(context.Background(), "res-1")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

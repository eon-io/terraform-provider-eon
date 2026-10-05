package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const findingExclusionsPath = "/api/v1/projects/project-1/finding-exclusions"
const findingExclusionPath = "/api/v1/projects/project-1/finding-exclusions/fe-1"

const findingExclusionResponse = `{
	"findingExclusion": {
		"id": "fe-1",
		"scope": "RESOURCE",
		"providerResourceId": "i-0123456789abcdef0",
		"value": "/var/cache/app",
		"type": "PATH",
		"detector": "MALWARE",
		"updatedAt": "2024-01-01T00:00:00Z"
	}
}`

const findingExclusionsListResponse = `{
	"findingExclusions": [
		{
			"id": "fe-1",
			"scope": "ACCOUNT",
			"value": "staging_events",
			"type": "TABLE",
			"detector": "DATA_ANOMALY",
			"updatedAt": "2024-01-01T00:00:00Z"
		}
	],
	"totalCount": 1
}`

func TestCreateFindingExclusion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK, body: findingExclusionResponse},
		{name: "created", statusCode: http.StatusCreated, body: findingExclusionResponse},
		{name: "empty body", statusCode: http.StatusCreated, body: `{"findingExclusion":null}`, wantErr: true},
		{name: "bad request", statusCode: http.StatusBadRequest, body: `{"message":"invalid"}`, wantErr: true},
		{name: "conflict", statusCode: http.StatusConflict, body: `{"message":"already exists"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, findingExclusionsPath, r.URL.Path)

				var sent externalEonSdkAPI.CreateFindingExclusionRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&sent))
				assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE, sent.GetScope())
				assert.Equal(t, "i-0123456789abcdef0", sent.GetProviderResourceId())

				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			req := externalEonSdkAPI.NewCreateFindingExclusionRequest(
				externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE,
				"/var/cache/app",
				externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH,
				externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE,
			)
			req.SetProviderResourceId("i-0123456789abcdef0")

			exclusion, err := c.CreateFindingExclusion(context.Background(), *req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, exclusion)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "fe-1", exclusion.GetId())
			assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, exclusion.GetDetector())
			assert.Equal(t, "i-0123456789abcdef0", exclusion.GetProviderResourceId())
		})
	}
}

func TestGetFindingExclusion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		statusCode     int
		body           string
		wantErr        bool
		wantNotFound   bool
		wantStatusCode int
	}{
		{name: "success", statusCode: http.StatusOK, body: findingExclusionResponse},
		{name: "not found", statusCode: http.StatusNotFound, body: `{"message":"not found"}`, wantErr: true, wantNotFound: true},
		{name: "null exclusion treated as not found", statusCode: http.StatusOK, body: `{"findingExclusion":null}`, wantErr: true, wantNotFound: true},
		{name: "server error", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, findingExclusionPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			exclusion, err := c.GetFindingExclusion(context.Background(), "fe-1")
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, exclusion)
				if tt.wantNotFound {
					var apiErr *APIError
					require.True(t, errors.As(err, &apiErr), "expected *APIError, got %T", err)
					assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "fe-1", exclusion.GetId())
			assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE, exclusion.GetScope())
			assert.Equal(t, "/var/cache/app", exclusion.GetValue())
			assert.Equal(t, externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH, exclusion.GetType())
		})
	}
}

func TestUpdateFindingExclusion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK, body: findingExclusionResponse},
		{name: "empty body", statusCode: http.StatusOK, body: `{"findingExclusion":null}`, wantErr: true},
		{name: "bad request", statusCode: http.StatusBadRequest, body: `{"message":"invalid"}`, wantErr: true},
		{name: "not found", statusCode: http.StatusNotFound, body: `{"message":"not found"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, findingExclusionPath, r.URL.Path)

				var sent externalEonSdkAPI.UpdateFindingExclusionRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&sent))
				assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_ACCOUNT, sent.GetScope())
				assert.False(t, sent.HasProviderResourceId())

				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			req := externalEonSdkAPI.NewUpdateFindingExclusionRequest(
				externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_ACCOUNT,
				"staging_events",
				externalEonSdkAPI.FINDING_OBJECT_TYPE_TABLE,
				externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_DATA_ANOMALY,
			)
			exclusion, err := c.UpdateFindingExclusion(context.Background(), "fe-1", *req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, exclusion)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "fe-1", exclusion.GetId())
		})
	}
}

func TestDeleteFindingExclusion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusNoContent},
		{name: "success ok", statusCode: http.StatusOK},
		{name: "not found", statusCode: http.StatusNotFound, wantErr: true},
		{name: "server error", statusCode: http.StatusInternalServerError, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, findingExclusionPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			}))

			err := c.DeleteFindingExclusion(context.Background(), "fe-1")
			if tt.wantErr {
				require.Error(t, err)
				var apiErr *APIError
				require.True(t, errors.As(err, &apiErr), "expected *APIError, got %T", err)
				assert.Equal(t, tt.statusCode, apiErr.StatusCode)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestListFindingExclusions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantLen    int
	}{
		{name: "success", statusCode: http.StatusOK, body: findingExclusionsListResponse, wantLen: 1},
		{name: "empty", statusCode: http.StatusOK, body: `{"findingExclusions":[],"totalCount":0}`, wantLen: 0},
		{name: "failure", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, findingExclusionsPath+"/list", r.URL.Path)
				assert.Equal(t, "100", r.URL.Query().Get("pageSize"))
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			exclusions, err := c.ListFindingExclusions(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, exclusions)
				return
			}
			require.NoError(t, err)
			assert.Len(t, exclusions, tt.wantLen)
		})
	}
}

func TestListFindingExclusionsPaginates(t *testing.T) {
	t.Parallel()

	pages := map[string]string{
		"":       `{"findingExclusions":[{"id":"fe-1","scope":"ACCOUNT","value":"a","type":"TABLE","detector":"MALWARE","updatedAt":"2024-01-01T00:00:00Z"}],"totalCount":2,"nextPageToken":"page-2"}`,
		"page-2": `{"findingExclusions":[{"id":"fe-2","scope":"ACCOUNT","value":"b","type":"TABLE","detector":"MALWARE","updatedAt":"2024-01-01T00:00:00Z"}],"totalCount":2}`,
	}

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		body, ok := pages[r.URL.Query().Get("pageToken")]
		require.True(t, ok, "unexpected page token %q", r.URL.Query().Get("pageToken"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}))

	exclusions, err := c.ListFindingExclusions(context.Background())
	require.NoError(t, err)
	require.Len(t, exclusions, 2)
	assert.Equal(t, "fe-1", exclusions[0].GetId())
	assert.Equal(t, "fe-2", exclusions[1].GetId())
}

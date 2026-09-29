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

const findingExclusionsPath = "/api/v1/projects/project-1/finding-exclusions"
const findingExclusionPath = "/api/v1/projects/project-1/finding-exclusions/fx-1"

const findingExclusionResponse = `{
	"findingExclusion": {
		"id": "fx-1",
		"resourceId": "res-1",
		"value": "/data/tmp",
		"type": "PATH",
		"detector": "MALWARE",
		"updatedAt": "2026-01-02T03:04:05Z"
	}
}`

const findingExclusionsListResponse = `{
	"findingExclusions": [
		{
			"id": "fx-1",
			"resourceId": "res-1",
			"value": "/data/tmp",
			"type": "PATH",
			"detector": "MALWARE",
			"updatedAt": "2026-01-02T03:04:05Z"
		},
		{
			"id": "fx-2",
			"value": "customers",
			"type": "TABLE",
			"detector": "DATA_ANOMALY",
			"updatedAt": "2026-01-02T03:04:05Z"
		}
	],
	"totalCount": 2
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
		{name: "resource not found", statusCode: http.StatusNotFound, body: `{"message":"not found"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, findingExclusionsPath, r.URL.Path)

				var sent externalEonSdkAPI.CreateFindingExclusionRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&sent))
				assert.Equal(t, "/data/tmp", sent.GetValue())
				assert.Equal(t, externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH, sent.GetType())
				assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, sent.GetDetector())
				assert.Equal(t, "res-1", sent.GetResourceId())

				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			req := externalEonSdkAPI.NewCreateFindingExclusionRequest(
				"/data/tmp",
				externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH,
				externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE,
			)
			req.SetResourceId("res-1")
			exclusion, err := c.CreateFindingExclusion(context.Background(), *req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, exclusion)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "fx-1", exclusion.GetId())
			assert.Equal(t, "res-1", exclusion.GetResourceId())
			assert.Equal(t, externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH, exclusion.GetType())
			assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, exclusion.GetDetector())
		})
	}
}

func TestGetFindingExclusion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		statusCode    int
		body          string
		wantErr       bool
		wantNotFound  bool
		wantErrStatus int
	}{
		{name: "success", statusCode: http.StatusOK, body: findingExclusionResponse},
		{name: "not found", statusCode: http.StatusNotFound, body: `{"message":"not found"}`, wantErr: true, wantNotFound: true},
		{name: "empty body is not found", statusCode: http.StatusOK, body: `{"findingExclusion":null}`, wantErr: true, wantNotFound: true},
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

			exclusion, err := c.GetFindingExclusion(context.Background(), "fx-1")
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, exclusion)
				if tt.wantNotFound {
					var apiErr *APIError
					require.ErrorAs(t, err, &apiErr)
					assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "fx-1", exclusion.GetId())
			assert.Equal(t, "/data/tmp", exclusion.GetValue())
			assert.Equal(t, "2026-01-02T03:04:05Z", exclusion.GetUpdatedAt().UTC().Format("2006-01-02T15:04:05Z07:00"))
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
				assert.Equal(t, "/data/tmp", sent.GetValue())
				assert.False(t, sent.HasResourceId())

				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			req := externalEonSdkAPI.NewUpdateFindingExclusionRequest(
				"/data/tmp",
				externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH,
				externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE,
			)
			exclusion, err := c.UpdateFindingExclusion(context.Background(), "fx-1", *req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, exclusion)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "fx-1", exclusion.GetId())
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

			err := c.DeleteFindingExclusion(context.Background(), "fx-1")
			if tt.wantErr {
				require.Error(t, err)
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr)
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
		filters    *externalEonSdkAPI.FindingExclusionFilterConditions
		wantErr    bool
		wantLen    int
	}{
		{name: "success", statusCode: http.StatusOK, body: findingExclusionsListResponse, wantLen: 2},
		{name: "empty", statusCode: http.StatusOK, body: `{"findingExclusions":[],"totalCount":0}`, wantLen: 0},
		{
			name:       "with filters",
			statusCode: http.StatusOK,
			body:       findingExclusionsListResponse,
			filters: func() *externalEonSdkAPI.FindingExclusionFilterConditions {
				detector := externalEonSdkAPI.NewFindingExclusionDetectorFilters()
				detector.SetIn([]externalEonSdkAPI.FindingExclusionDetectorType{externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE})
				filters := externalEonSdkAPI.NewFindingExclusionFilterConditions()
				filters.SetDetector(*detector)
				return filters
			}(),
			wantLen: 2,
		},
		{name: "failure", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, findingExclusionsPath+"/list", r.URL.Path)
				assert.Equal(t, "100", r.URL.Query().Get("pageSize"))

				var sent externalEonSdkAPI.ListFindingExclusionsRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&sent))
				if tt.filters != nil {
					require.True(t, sent.HasFilters())
					sentFilters := sent.GetFilters()
					sentDetector := sentFilters.GetDetector()
					wantDetector := tt.filters.GetDetector()
					assert.Equal(t, wantDetector.GetIn(), sentDetector.GetIn())
				} else {
					assert.False(t, sent.HasFilters())
				}

				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			exclusions, err := c.ListFindingExclusions(context.Background(), tt.filters)
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
		"": `{
			"findingExclusions": [{"id": "fx-1", "value": "/a", "type": "PATH", "detector": "MALWARE", "updatedAt": "2026-01-02T03:04:05Z"}],
			"totalCount": 2,
			"nextPageToken": "page-2"
		}`,
		"page-2": `{
			"findingExclusions": [{"id": "fx-2", "value": "/b", "type": "PATH", "detector": "MALWARE", "updatedAt": "2026-01-02T03:04:05Z"}],
			"totalCount": 2
		}`,
	}

	var calls int
	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		body, ok := pages[r.URL.Query().Get("pageToken")]
		require.True(t, ok, "unexpected page token %q", r.URL.Query().Get("pageToken"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}))

	exclusions, err := c.ListFindingExclusions(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
	require.Len(t, exclusions, 2)
	assert.Equal(t, "fx-1", exclusions[0].GetId())
	assert.Equal(t, "fx-2", exclusions[1].GetId())
}

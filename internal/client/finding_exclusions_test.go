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

const findingExclusionsPath = "/api/v1/projects/project-1/finding-exclusions"
const findingExclusionPath = "/api/v1/projects/project-1/finding-exclusions/fe-1"
const findingExclusionsListPath = "/api/v1/projects/project-1/finding-exclusions/list"

const findingExclusionResponse = `{
	"findingExclusion": {
		"id": "fe-1",
		"resourceId": "res-1",
		"value": "/var/cache",
		"type": "PATH",
		"detector": "MALWARE",
		"updatedAt": "2026-01-02T03:04:05Z"
	}
}`

const findingExclusionsListResponse = `{
	"findingExclusions": [
		{
			"id": "fe-1",
			"value": "/var/cache",
			"type": "PATH",
			"detector": "MALWARE",
			"updatedAt": "2026-01-02T03:04:05Z"
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
		{name: "empty exclusion", statusCode: http.StatusOK, body: `{"findingExclusion":null}`, wantErr: true},
		{name: "bad request", statusCode: http.StatusBadRequest, body: `{"message":"invalid"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, findingExclusionsPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			req := externalEonSdkAPI.NewCreateFindingExclusionRequest(
				"/var/cache",
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
			assert.Equal(t, "fe-1", exclusion.GetId())
			assert.Equal(t, "/var/cache", exclusion.GetValue())
			assert.Equal(t, externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH, exclusion.GetType())
			assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, exclusion.GetDetector())
			assert.Equal(t, "res-1", exclusion.GetResourceId())
		})
	}
}

func TestGetFindingExclusion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantStatus int
	}{
		{name: "success", statusCode: http.StatusOK, body: findingExclusionResponse},
		{name: "not found", statusCode: http.StatusNotFound, body: `{"message":"not found"}`, wantErr: true, wantStatus: http.StatusNotFound},
		{name: "empty exclusion", statusCode: http.StatusOK, body: `{"findingExclusion":null}`, wantErr: true, wantStatus: http.StatusNotFound},
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
				assert.Error(t, err)
				assert.Nil(t, exclusion)
				if tt.wantStatus != 0 {
					var apiErr *APIError
					require.ErrorAs(t, err, &apiErr)
					assert.Equal(t, tt.wantStatus, apiErr.StatusCode)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "fe-1", exclusion.GetId())
			assert.Equal(t, "res-1", exclusion.GetResourceId())
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
		{name: "empty exclusion", statusCode: http.StatusOK, body: `{"findingExclusion":null}`, wantErr: true},
		{name: "bad request", statusCode: http.StatusBadRequest, body: `{"message":"invalid"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, findingExclusionPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			req := externalEonSdkAPI.NewUpdateFindingExclusionRequest(
				"orders",
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
			assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, exclusion.GetDetector())
		})
	}
}

func TestDeleteFindingExclusion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
		wantStatus int
	}{
		{name: "success", statusCode: http.StatusNoContent},
		{name: "success ok", statusCode: http.StatusOK},
		{name: "not found", statusCode: http.StatusNotFound, wantErr: true, wantStatus: http.StatusNotFound},
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
				assert.Error(t, err)
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.wantStatus, apiErr.StatusCode)
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
				assert.Equal(t, findingExclusionsListPath, r.URL.Path)
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

func TestListFindingExclusionsPagination(t *testing.T) {
	t.Parallel()

	const pageOne = `{
		"findingExclusions": [
			{"id":"fe-1","value":"/var/cache","type":"PATH","detector":"MALWARE","updatedAt":"2026-01-02T03:04:05Z"}
		],
		"totalCount": 2,
		"nextPageToken": "page-2"
	}`
	const pageTwo = `{
		"findingExclusions": [
			{"id":"fe-2","value":"orders","type":"TABLE","detector":"DATA_ANOMALY","updatedAt":"2026-01-02T03:04:05Z"}
		],
		"totalCount": 2
	}`

	var calls int
	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, findingExclusionsListPath, r.URL.Path)
		calls++
		body := pageOne
		if calls > 1 {
			assert.Equal(t, "page-2", r.URL.Query().Get("pageToken"))
			body = pageTwo
		}
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
	assert.Equal(t, 2, calls)
}

func TestFindingExclusionClientTransportError(t *testing.T) {
	t.Parallel()

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	}))

	_, err := c.GetFindingExclusion(context.Background(), "fe-1")
	assert.Error(t, err)
}

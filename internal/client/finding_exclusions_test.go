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

const findingExclusionsPath = "/api/v1/projects/project-1/finding-exclusions"
const findingExclusionPath = "/api/v1/projects/project-1/finding-exclusions/excl-1"
const findingExclusionsListPath = "/api/v1/projects/project-1/finding-exclusions/list"

const findingExclusionResponse = `{
	"findingExclusion": {
		"id": "excl-1",
		"scope": "RESOURCE",
		"providerResourceId": "i-abc",
		"value": "/var/cache",
		"type": "PATH",
		"detector": "MALWARE",
		"updatedAt": "2026-01-02T03:04:05Z"
	}
}`

const findingExclusionsListResponse = `{
	"findingExclusions": [
		{
			"id": "excl-1",
			"scope": "RESOURCE",
			"providerResourceId": "i-abc",
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
				externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE,
				"/var/cache",
				externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH,
				externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE,
			)
			req.SetProviderResourceId("i-abc")
			exclusion, err := c.CreateFindingExclusion(context.Background(), *req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, exclusion)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "excl-1", exclusion.GetId())
			assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE, exclusion.GetScope())
			assert.Equal(t, "i-abc", exclusion.GetProviderResourceId())
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
	}{
		{name: "success", statusCode: http.StatusOK, body: findingExclusionResponse},
		{name: "empty exclusion", statusCode: http.StatusOK, body: `{"findingExclusion":null}`, wantErr: true},
		{name: "not found", statusCode: http.StatusNotFound, body: `{"message":"not found"}`, wantErr: true},
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

			exclusion, err := c.GetFindingExclusion(context.Background(), "excl-1")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, exclusion)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "excl-1", exclusion.GetId())
			assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, exclusion.GetDetector())
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
				externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_ACCOUNT,
				"orders",
				externalEonSdkAPI.FINDING_OBJECT_TYPE_TABLE,
				externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_DATA_ANOMALY,
			)
			exclusion, err := c.UpdateFindingExclusion(context.Background(), "excl-1", *req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, exclusion)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "excl-1", exclusion.GetId())
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

			err := c.DeleteFindingExclusion(context.Background(), "excl-1")
			if tt.wantErr {
				assert.Error(t, err)
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

func TestListFindingExclusionsPaginates(t *testing.T) {
	t.Parallel()

	pageOne := `{
		"findingExclusions": [
			{
				"id": "excl-1",
				"scope": "ACCOUNT",
				"value": "orders",
				"type": "TABLE",
				"detector": "DATA_ANOMALY",
				"updatedAt": "2026-01-02T03:04:05Z"
			}
		],
		"totalCount": 2,
		"nextPageToken": "page-2"
	}`
	pageTwo := `{
		"findingExclusions": [
			{
				"id": "excl-2",
				"scope": "RESOURCE",
				"providerResourceId": "i-def",
				"value": "/tmp",
				"type": "PATH",
				"detector": "RANSOMWARE_BEHAVIOR",
				"updatedAt": "2026-01-02T03:04:05Z"
			}
		],
		"totalCount": 2
	}`

	var calls int
	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, findingExclusionsListPath, r.URL.Path)
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
	assert.Equal(t, "excl-1", exclusions[0].GetId())
	assert.Equal(t, "excl-2", exclusions[1].GetId())
	assert.Equal(t, 2, calls)
}

func TestListFindingExclusionsSecondPageFailure(t *testing.T) {
	t.Parallel()

	pageOne := `{
		"findingExclusions": [
			{
				"id": "excl-1",
				"scope": "ACCOUNT",
				"value": "orders",
				"type": "TABLE",
				"detector": "DATA_ANOMALY",
				"updatedAt": "2026-01-02T03:04:05Z"
			}
		],
		"totalCount": 2,
		"nextPageToken": "page-2"
	}`

	var calls int
	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		body := pageOne
		status := http.StatusOK
		if calls > 1 {
			body = `{"message":"boom"}`
			status = http.StatusInternalServerError
		}
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}))

	exclusions, err := c.ListFindingExclusions(context.Background())
	assert.Error(t, err)
	assert.Nil(t, exclusions)
}

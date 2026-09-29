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
const findingExclusionPath = "/api/v1/projects/project-1/finding-exclusions/exclusion-1"

const findingExclusionResponse = `{
	"findingExclusion": {
		"id": "exclusion-1",
		"scope": "RESOURCE",
		"providerResourceId": "i-0abc123def4567890",
		"value": "/var/cache",
		"type": "PATH",
		"detector": "MALWARE",
		"updatedAt": "2026-09-01T00:00:00Z"
	}
}`

func findingExclusionTestClient(t *testing.T, method, path string, statusCode int, body string, inspect func(*http.Request)) *EonClient {
	t.Helper()
	return newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, method, r.Method)
		assert.Equal(t, path, r.URL.Path)
		if inspect != nil {
			inspect(r)
		}
		return &http.Response{
			StatusCode: statusCode,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}))
}

func TestCreateFindingExclusion(t *testing.T) {
	t.Parallel()

	t.Run("sends the exclusion and returns the created one", func(t *testing.T) {
		t.Parallel()
		c := findingExclusionTestClient(t, http.MethodPost, findingExclusionsPath, http.StatusCreated, findingExclusionResponse, func(r *http.Request) {
			var sent map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&sent))
			assert.Equal(t, map[string]any{
				"scope":              "RESOURCE",
				"providerResourceId": "i-0abc123def4567890",
				"value":              "/var/cache",
				"type":               "PATH",
				"detector":           "MALWARE",
			}, sent)
		})

		req := externalEonSdkAPI.NewCreateFindingExclusionRequest(externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE, "/var/cache", externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE)
		req.SetProviderResourceId("i-0abc123def4567890")
		exclusion, err := c.CreateFindingExclusion(context.Background(), *req)

		require.NoError(t, err)
		assert.Equal(t, "exclusion-1", exclusion.GetId())
		assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, exclusion.GetDetector())
	})

	t.Run("a rejected exclusion is an error", func(t *testing.T) {
		t.Parallel()
		c := findingExclusionTestClient(t, http.MethodPost, findingExclusionsPath, http.StatusBadRequest, `{"error":"malware is not scanned in databases"}`, nil)

		req := externalEonSdkAPI.NewCreateFindingExclusionRequest(externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_ACCOUNT, "orders", externalEonSdkAPI.FINDING_OBJECT_TYPE_TABLE, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE)
		exclusion, err := c.CreateFindingExclusion(context.Background(), *req)

		assert.Error(t, err)
		assert.Nil(t, exclusion)
	})
}

func TestGetFindingExclusion(t *testing.T) {
	t.Parallel()

	t.Run("returns the exclusion", func(t *testing.T) {
		t.Parallel()
		c := findingExclusionTestClient(t, http.MethodGet, findingExclusionPath, http.StatusOK, findingExclusionResponse, nil)

		exclusion, err := c.GetFindingExclusion(context.Background(), "exclusion-1")

		require.NoError(t, err)
		assert.Equal(t, "/var/cache", exclusion.GetValue())
		assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE, exclusion.GetScope())
		assert.Equal(t, "i-0abc123def4567890", exclusion.GetProviderResourceId())
	})

	t.Run("a missing exclusion is a 404 API error", func(t *testing.T) {
		t.Parallel()
		c := findingExclusionTestClient(t, http.MethodGet, findingExclusionPath, http.StatusNotFound, `{"error":"finding exclusion pattern not found"}`, nil)

		_, err := c.GetFindingExclusion(context.Background(), "exclusion-1")

		var apiErr *APIError
		require.True(t, errors.As(err, &apiErr), "got %v", err)
		assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	})
}

func TestUpdateFindingExclusion(t *testing.T) {
	t.Parallel()

	c := findingExclusionTestClient(t, http.MethodPut, findingExclusionPath, http.StatusOK, findingExclusionResponse, func(r *http.Request) {
		var sent map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&sent))
		assert.Equal(t, map[string]any{"scope": "ACCOUNT", "value": "/var/cache", "type": "PATH", "detector": "MALWARE"}, sent)
	})

	req := externalEonSdkAPI.NewUpdateFindingExclusionRequest(externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_ACCOUNT, "/var/cache", externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE)
	exclusion, err := c.UpdateFindingExclusion(context.Background(), "exclusion-1", *req)

	require.NoError(t, err)
	assert.Equal(t, "exclusion-1", exclusion.GetId())
}

func TestDeleteFindingExclusion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantStatus int
	}{
		{name: "deleted", statusCode: http.StatusNoContent},
		{name: "not found", statusCode: http.StatusNotFound, wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := findingExclusionTestClient(t, http.MethodDelete, findingExclusionPath, tt.statusCode, "", nil)

			err := c.DeleteFindingExclusion(context.Background(), "exclusion-1")

			if tt.wantStatus == 0 {
				require.NoError(t, err)
				return
			}
			var apiErr *APIError
			require.True(t, errors.As(err, &apiErr), "got %v", err)
			assert.Equal(t, tt.wantStatus, apiErr.StatusCode)
		})
	}
}

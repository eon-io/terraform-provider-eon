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

type noopTokenRefresher struct{}

func (noopTokenRefresher) Initialize(_ *externalEonSdkAPI.APIClient) error {
	return nil
}

func (noopTokenRefresher) EnsureValidToken() error {
	return nil
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestClient(transport http.RoundTripper) *EonClient {
	cfg := externalEonSdkAPI.NewConfiguration()
	cfg.Servers = []externalEonSdkAPI.ServerConfiguration{{URL: "https://example.test/api"}}
	cfg.HTTPClient = &http.Client{Transport: transport}

	return &EonClient{
		client:         externalEonSdkAPI.NewAPIClient(cfg),
		projectID:      "project-1",
		tokenRefresher: noopTokenRefresher{},
	}
}

func TestGetCloudResourceConfiguration(t *testing.T) {
	t.Parallel()

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/projects/project-1/resources/resource-1/object-store-scan-method", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{
			"cdcBackup": {"systemControlled": true},
			"inventoryBackup": {"systemControlled": false, "enabled": false}
		}`)),
		}, nil
	}))

	config, err := c.GetCloudResourceConfiguration(context.Background(), "resource-1")
	require.NoError(t, err)
	require.NotNil(t, config.CdcBackup.SystemControlled)
	assert.True(t, *config.CdcBackup.SystemControlled)
	require.NotNil(t, config.InventoryBackup.SystemControlled)
	require.NotNil(t, config.InventoryBackup.Enabled)
	assert.False(t, *config.InventoryBackup.SystemControlled)
	assert.False(t, *config.InventoryBackup.Enabled)
}

func TestUpdateCloudResourceConfiguration(t *testing.T) {
	t.Parallel()

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/projects/project-1/resources/resource-1/object-store-scan-method", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		var got map[string]map[string]*bool
		err := json.NewDecoder(r.Body).Decode(&got)
		require.NoError(t, err)

		require.Contains(t, got, "cdcBackup")
		require.Contains(t, got, "inventoryBackup")
		assert.False(t, *got["cdcBackup"]["systemControlled"])
		assert.True(t, *got["cdcBackup"]["enabled"])
		assert.True(t, *got["inventoryBackup"]["systemControlled"])
		assert.Nil(t, got["inventoryBackup"]["enabled"])

		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}))

	enabled := true
	userControlled := false
	systemControlled := true
	err := c.UpdateCloudResourceConfiguration(context.Background(), "resource-1", UpdateCloudResourceConfigurationRequest{
		CdcBackup: &CloudResourceBackupMethodConfig{
			Enabled:          &enabled,
			SystemControlled: &userControlled,
		},
		InventoryBackup: &CloudResourceBackupMethodConfig{
			SystemControlled: &systemControlled,
		},
	})
	require.NoError(t, err)
}

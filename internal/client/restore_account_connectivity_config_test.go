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

// The SDK takes (projectId, accountId) in that order; passing them swapped produces a valid-looking
// URL that the API rejects with 403 (EON-14842), so these tests pin the exact request path.
const connectivityConfigPath = "/api/v1/projects/project-1/restore-accounts/account-1/connectivity-config"

const connectivityConfigResponse = `{
	"restoreAccountConfig": {
		"restoreAccountId": "account-1",
		"config": {}
	}
}`

func TestGetRestoreAccountConnectivityConfig(t *testing.T) {
	t.Parallel()

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, connectivityConfigPath, r.URL.Path)

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(connectivityConfigResponse)),
		}, nil
	}))

	config, err := c.GetRestoreAccountConnectivityConfig(context.Background(), "account-1")
	require.NoError(t, err)
	assert.Equal(t, "account-1", config.GetRestoreAccountId())
}

func TestUpdateRestoreAccountConnectivityConfig(t *testing.T) {
	t.Parallel()

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, connectivityConfigPath, r.URL.Path)

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(connectivityConfigResponse)),
		}, nil
	}))

	config, err := c.UpdateRestoreAccountConnectivityConfig(context.Background(), "account-1",
		externalEonSdkAPI.UpdateRestoreAccountConnectivityConfigRequest{})
	require.NoError(t, err)
	assert.Equal(t, "account-1", config.GetRestoreAccountId())
}

func TestDeleteRestoreAccountConnectivityConfig(t *testing.T) {
	t.Parallel()

	c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, connectivityConfigPath, r.URL.Path)

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	}))

	require.NoError(t, c.DeleteRestoreAccountConnectivityConfig(context.Background(), "account-1"))
}

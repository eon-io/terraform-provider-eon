package client

import (
	"fmt"
	"maps"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
)

// newClient is the common client creation logic
func newClient(cfg ClientConfig, refresher TokenRefresher) (*EonClient, error) {
	config := externalEonSdkAPI.NewConfiguration()
	config.Servers = []externalEonSdkAPI.ServerConfiguration{{URL: cfg.Endpoint}}

	// Apply any additional default headers from configuration
	if cfg.DefaultHeaders != nil {
		maps.Copy(config.DefaultHeader, cfg.DefaultHeaders)
	}

	apiClient := externalEonSdkAPI.NewAPIClient(config)

	if err := refresher.Initialize(apiClient); err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	return &EonClient{
		client:         apiClient,
		projectID:      cfg.ProjectID,
		tokenRefresher: refresher,
	}, nil
}

// NewClient creates an Eon API client with OAuth authentication
// If Authorization header is set in DefaultHeaders, OAuth is skipped
func NewClient(cfg ClientConfig) (*EonClient, error) {
	refresher := NewOAuthTokenRefresher(cfg.ClientID, cfg.ClientSecret)
	return newClient(cfg, refresher)
}

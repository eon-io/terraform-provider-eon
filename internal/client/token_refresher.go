package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
)

// staticAuthExpiry is used for externally managed auth that never needs refresh
const staticAuthExpiry = 100 * 365 * 24 * time.Hour

// TokenRefresher handles auth token lifecycle
type TokenRefresher interface {
	// Initialize sets up the refresher with the API client and performs initial auth if needed
	Initialize(apiClient *externalEonSdkAPI.APIClient) error
	// EnsureValidToken checks if the current token is valid and refreshes if necessary
	EnsureValidToken() error
}

// OAuthTokenRefresher handles OAuth token refresh
type OAuthTokenRefresher struct {
	apiClient    *externalEonSdkAPI.APIClient
	clientID     string
	clientSecret string
	tokenExpiry  time.Time
}

// NewOAuthTokenRefresher creates a new OAuth token refresher
func NewOAuthTokenRefresher(clientID, clientSecret string) *OAuthTokenRefresher {
	return &OAuthTokenRefresher{
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

// Initialize sets up the API client reference and performs initial authentication
func (r *OAuthTokenRefresher) Initialize(apiClient *externalEonSdkAPI.APIClient) error {
	r.apiClient = apiClient

	// If Authorization header is already set via DefaultHeaders, skip OAuth
	if _, exists := r.apiClient.GetConfig().DefaultHeader["Authorization"]; exists {
		// Never expires - externally managed auth
		r.tokenExpiry = time.Now().Add(staticAuthExpiry)
		return nil
	}

	// Otherwise, do OAuth
	return r.Authenticate()
}

// EnsureValidToken checks if the current token is valid and refreshes it if necessary
func (r *OAuthTokenRefresher) EnsureValidToken() error {
	// If token never expires, it's static auth from DefaultHeaders - don't refresh
	if r.tokenExpiry.After(time.Now().Add(staticAuthExpiry - 365*24*time.Hour)) {
		return nil
	}

	// Otherwise, check expiry and refresh if needed
	if time.Now().After(r.tokenExpiry.Add(-30 * time.Second)) {
		return r.Authenticate()
	}
	return nil
}

// Authenticate performs OAuth authentication with the Eon API
func (r *OAuthTokenRefresher) Authenticate() error {
	resp, httpResp, err := r.apiClient.AuthAPI.GetAccessToken(context.Background()).ApiCredentials(externalEonSdkAPI.ApiCredentials{
		ClientId:     r.clientID,
		ClientSecret: r.clientSecret,
	}).Execute()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("authentication failed with status %d: %s", httpResp.StatusCode, body)
	}

	r.tokenExpiry = time.Now().Add(time.Duration(resp.GetExpirationSeconds()) * time.Second)
	r.apiClient.GetConfig().DefaultHeader["Authorization"] = "Bearer " + resp.GetAccessToken()

	return nil
}

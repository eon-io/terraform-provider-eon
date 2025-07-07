package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
)

// EonClient wraps the Eon SDK client with authentication and configuration
type EonClient struct {
	client       *externalEonSdkAPI.APIClient
	ProjectID    string
	EonAccountID string
	authToken    string
	tokenExpiry  time.Time
	clientID     string
	clientSecret string
	endpoint     string
}

// NewEonClient creates a new Eon API client with the provided configuration
func NewEonClient(endpoint, clientID, clientSecret, projectID, eonAccountID string) (*EonClient, error) {
	config := externalEonSdkAPI.NewConfiguration()
	config.Servers = []externalEonSdkAPI.ServerConfiguration{
		{
			URL: endpoint,
		},
	}

	client := &EonClient{
		client:       externalEonSdkAPI.NewAPIClient(config),
		ProjectID:    projectID,
		EonAccountID: eonAccountID,
		clientID:     clientID,
		clientSecret: clientSecret,
		endpoint:     endpoint,
	}

	// Authenticate immediately to validate credentials
	if err := client.authenticate(); err != nil {
		return nil, fmt.Errorf("failed to authenticate with Eon API: %w", err)
	}

	return client, nil
}

// authenticate performs OAuth authentication with the Eon API
func (c *EonClient) authenticate() error {
	resp, httpResp, err := c.client.AuthAPI.GetAccessTokenOAuth2(context.Background()).
		ClientId(c.clientID).
		ClientSecret(c.clientSecret).
		Execute()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("authentication failed with status %d: %s", httpResp.StatusCode, body)
	}

	c.authToken = resp.GetAccessToken()
	c.tokenExpiry = time.Now().Add(time.Duration(resp.GetExpiresIn()) * time.Second)

	// Set the authorization header for future requests
	c.client.GetConfig().DefaultHeader["Authorization"] = "Bearer " + c.authToken

	return nil
}

// ensureValidToken checks if the current token is valid and refreshes it if necessary
func (c *EonClient) ensureValidToken() error {
	if time.Now().After(c.tokenExpiry.Add(-30 * time.Second)) {
		return c.authenticate()
	}
	return nil
}

// ListSourceAccounts retrieves all source accounts for the project
func (c *EonClient) ListSourceAccounts(ctx context.Context) ([]externalEonSdkAPI.SourceAccount, error) {
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ListSourceAccounts(ctx, c.ProjectID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list source accounts: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, body)
	}

	if resp.GetAccounts() == nil {
		return []externalEonSdkAPI.SourceAccount{}, nil
	}

	return resp.GetAccounts(), nil
}

// ListRestoreAccounts retrieves all restore accounts for the project
func (c *EonClient) ListRestoreAccounts(ctx context.Context) ([]externalEonSdkAPI.RestoreAccount, error) {
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ListRestoreAccounts(ctx, c.ProjectID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list restore accounts: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, body)
	}

	if resp.GetAccounts() == nil {
		return []externalEonSdkAPI.RestoreAccount{}, nil
	}

	return resp.GetAccounts(), nil
}

// ConnectSourceAccount connects a new source account
func (c *EonClient) ConnectSourceAccount(ctx context.Context, req externalEonSdkAPI.ConnectSourceAccountRequest) (*externalEonSdkAPI.SourceAccount, error) {
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ConnectSourceAccount(ctx, c.ProjectID).ConnectSourceAccountRequest(req).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to connect source account: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, body)
	}

	account := resp.GetSourceAccount()
	return &account, nil
}

// DisconnectSourceAccount disconnects a source account
func (c *EonClient) DisconnectSourceAccount(ctx context.Context, accountId string) error {
	if err := c.ensureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	_, httpResp, err := c.client.AccountsAPI.DisconnectSourceAccount(ctx, c.ProjectID, accountId).Execute()
	if err != nil {
		return fmt.Errorf("failed to disconnect source account: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, body)
	}

	return nil
}

// ConnectRestoreAccount connects a new restore account
func (c *EonClient) ConnectRestoreAccount(ctx context.Context, req externalEonSdkAPI.ConnectRestoreAccountRequest) (*externalEonSdkAPI.RestoreAccount, error) {
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ConnectRestoreAccount(ctx, c.ProjectID).ConnectRestoreAccountRequest(req).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to connect restore account: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, body)
	}

	account := resp.GetRestoreAccount()
	return &account, nil
}

// DisconnectRestoreAccount disconnects a restore account
func (c *EonClient) DisconnectRestoreAccount(ctx context.Context, accountId string) error {
	if err := c.ensureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	_, httpResp, err := c.client.AccountsAPI.DisconnectRestoreAccount(ctx, c.ProjectID, accountId).Execute()
	if err != nil {
		return fmt.Errorf("failed to disconnect restore account: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, body)
	}

	return nil
}

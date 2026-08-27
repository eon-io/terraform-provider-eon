package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
)

// APIError represents an error from the Eon API with HTTP status code
type APIError struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("API error %d: %s: %v", e.StatusCode, e.Message, e.Err)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// EonClient wraps the Eon SDK client with authentication and configuration
type EonClient struct {
	client         *externalEonSdkAPI.APIClient
	projectID      string
	tokenRefresher TokenRefresher
}

// CloudResourceBackupMethodConfig represents the auto/user-controlled backup method
// configuration used by GCS bucket scan detection and full-scan settings.
type CloudResourceBackupMethodConfig struct {
	Enabled          *bool `json:"enabled,omitempty"`
	SystemControlled *bool `json:"systemControlled,omitempty"`
}

// CloudResourceConfiguration represents the supported inventory resource configuration fields.
type CloudResourceConfiguration struct {
	CdcBackup       CloudResourceBackupMethodConfig `json:"cdcBackup"`
	InventoryBackup CloudResourceBackupMethodConfig `json:"inventoryBackup"`
}

// UpdateCloudResourceConfigurationRequest represents a partial inventory resource configuration update.
type UpdateCloudResourceConfigurationRequest struct {
	CdcBackup       *CloudResourceBackupMethodConfig `json:"cdcBackup,omitempty"`
	InventoryBackup *CloudResourceBackupMethodConfig `json:"inventoryBackup,omitempty"`
}

// GetProjectID returns the project ID
func (c *EonClient) GetProjectID() string {
	return c.projectID
}

// handleAPIError processes API errors and extracts detailed error information from HTTP responses
func (c *EonClient) handleAPIError(err error, httpResp *http.Response, baseErrorMsg string) error {
	if err != nil && httpResp != nil {
		defer func() { _ = httpResp.Body.Close() }()
		if body, readErr := io.ReadAll(httpResp.Body); readErr == nil && len(body) > 0 {
			return &APIError{
				StatusCode: httpResp.StatusCode,
				Message:    string(body),
				Err:        err,
			}
		}
		return &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    baseErrorMsg,
			Err:        err,
		}
	} else if err != nil {
		return fmt.Errorf("%s: %w", baseErrorMsg, err)
	}
	return nil
}

// GetSourceAccount retrieves a source account by ID via
// GET /v1/projects/{projectId}/source-accounts/{accountId}.
// Returns an *APIError with StatusCode 404 when the account does not exist.
func (c *EonClient) GetSourceAccount(ctx context.Context, accountId string) (*externalEonSdkAPI.SourceAccount, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.GetSourceAccount(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get source account"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	account := resp.GetSourceAccount()
	return &account, nil
}

// ListSourceAccounts retrieves all source accounts for the project.
// It paginates through all pages to return the complete list.
func (c *EonClient) ListSourceAccounts(ctx context.Context) ([]externalEonSdkAPI.SourceAccount, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	var allAccounts []externalEonSdkAPI.SourceAccount
	var pageToken string

	for {
		req := c.client.AccountsAPI.ListSourceAccounts(ctx, c.projectID).
			ListSourceAccountsRequest(externalEonSdkAPI.ListSourceAccountsRequest{})
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		resp, httpResp, err := req.Execute()

		if apiErr := c.handleAPIError(err, httpResp, "failed to list source accounts"); apiErr != nil {
			return nil, apiErr
		}
		defer func() { _ = httpResp.Body.Close() }()

		if httpResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(httpResp.Body)
			return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
		}

		accounts := resp.GetAccounts()
		if accounts != nil {
			allAccounts = append(allAccounts, accounts...)
		}

		if !resp.HasNextToken() {
			break
		}
		pageToken = resp.GetNextToken()
	}

	if allAccounts == nil {
		return []externalEonSdkAPI.SourceAccount{}, nil
	}

	return allAccounts, nil
}

// ListRestoreAccounts retrieves all restore accounts for the project.
// It paginates through all pages to return the complete list.
func (c *EonClient) ListRestoreAccounts(ctx context.Context) ([]externalEonSdkAPI.RestoreAccount, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	var allAccounts []externalEonSdkAPI.RestoreAccount
	var pageToken string

	for {
		req := c.client.AccountsAPI.ListRestoreAccounts(ctx, c.projectID).
			ListRestoreAccountsRequest(externalEonSdkAPI.ListRestoreAccountsRequest{})
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		resp, httpResp, err := req.Execute()

		if apiErr := c.handleAPIError(err, httpResp, "failed to list restore accounts"); apiErr != nil {
			return nil, apiErr
		}
		defer func() { _ = httpResp.Body.Close() }()

		if httpResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(httpResp.Body)
			return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
		}

		accounts := resp.GetAccounts()
		if accounts != nil {
			allAccounts = append(allAccounts, accounts...)
		}

		if !resp.HasNextToken() {
			break
		}
		pageToken = resp.GetNextToken()
	}

	if allAccounts == nil {
		return []externalEonSdkAPI.RestoreAccount{}, nil
	}

	return allAccounts, nil
}

// GetRestoreAccount retrieves a single restore account by ID.
func (c *EonClient) GetRestoreAccount(ctx context.Context, accountId string) (*externalEonSdkAPI.RestoreAccount, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.GetRestoreAccount(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get restore account"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	account := resp.GetRestoreAccount()
	return &account, nil
}

// ConnectSourceAccount connects a new source account
func (c *EonClient) ConnectSourceAccount(ctx context.Context, req externalEonSdkAPI.ConnectSourceAccountRequest) (*externalEonSdkAPI.SourceAccount, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ConnectSourceAccount(ctx, c.projectID).ConnectSourceAccountRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to connect source account"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	account := resp.GetSourceAccount()
	return &account, nil
}

// DisconnectSourceAccount disconnects a source account
func (c *EonClient) DisconnectSourceAccount(ctx context.Context, accountId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	_, httpResp, err := c.client.AccountsAPI.DisconnectSourceAccount(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to disconnect source account"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// DeleteSourceAccount fully removes a source account from the project.
// Callers should typically Disconnect first if the account is still connected.
func (c *EonClient) DeleteSourceAccount(ctx context.Context, accountId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	httpResp, err := c.client.AccountsAPI.DeleteSourceAccount(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to delete source account"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// UpdateSourceAccountRequest contains the fields that can be updated on a source account.
type UpdateSourceAccountRequest struct {
	Name                    *string
	SourceAccountAttributes *UpdateSourceAccountAttributes
}

// UpdateSourceAccountAttributes contains cloud-provider-specific attributes to update.
type UpdateSourceAccountAttributes struct {
	Aws *UpdateAwsSourceAccountAttributes
}

// UpdateAwsSourceAccountAttributes contains AWS-specific attributes to update.
type UpdateAwsSourceAccountAttributes struct {
	RoleArn *string
}

// UpdateSourceAccount updates mutable fields of a source account via
// PATCH /v1/projects/{projectId}/source-accounts/{accountId}.
func (c *EonClient) UpdateSourceAccount(ctx context.Context, accountId string, req UpdateSourceAccountRequest) (*externalEonSdkAPI.SourceAccount, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	sdkReq := externalEonSdkAPI.NewUpdateSourceAccountRequest()
	if req.Name != nil {
		sdkReq.SetName(*req.Name)
	}
	if req.SourceAccountAttributes != nil && req.SourceAccountAttributes.Aws != nil {
		attrs := externalEonSdkAPI.NewUpdateSourceAccountAttributesInput()
		awsAttrs := externalEonSdkAPI.UpdateAwsSourceAccountAttributes{}
		if req.SourceAccountAttributes.Aws.RoleArn != nil {
			awsAttrs.SetRoleArn(*req.SourceAccountAttributes.Aws.RoleArn)
		}
		attrs.SetAws(awsAttrs)
		sdkReq.SetSourceAccountAttributes(*attrs)
	}

	resp, httpResp, err := c.client.AccountsAPI.UpdateSourceAccount(ctx, c.projectID, accountId).
		UpdateSourceAccountRequest(*sdkReq).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to update source account"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	account := resp.GetSourceAccount()
	return &account, nil
}

// ReconnectSourceAccount reconnects a previously disconnected source account
func (c *EonClient) ReconnectSourceAccount(ctx context.Context, accountId string) (*externalEonSdkAPI.SourceAccount, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ReconnectSourceAccount(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to reconnect source account"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	account := resp.GetSourceAccount()
	return &account, nil
}

// ListSourceAwsOrganizationalUnits retrieves all source AWS organizational units for the project.
// It paginates through all pages to return the complete list.
func (c *EonClient) ListSourceAwsOrganizationalUnits(ctx context.Context) ([]externalEonSdkAPI.SourceAwsOrganizationalUnit, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	var allOUs []externalEonSdkAPI.SourceAwsOrganizationalUnit
	var pageToken string

	for {
		req := c.client.AccountsAPI.ListSourceAwsOrganizationalUnits(ctx, c.projectID)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		resp, httpResp, err := req.Execute()

		if apiErr := c.handleAPIError(err, httpResp, "failed to list source AWS organizational units"); apiErr != nil {
			return nil, apiErr
		}
		defer func() { _ = httpResp.Body.Close() }()

		if httpResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(httpResp.Body)
			return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
		}

		ous := resp.GetOrganizationalUnits()
		if ous != nil {
			allOUs = append(allOUs, ous...)
		}

		if !resp.HasNextToken() {
			break
		}
		pageToken = resp.GetNextToken()
	}

	if allOUs == nil {
		return []externalEonSdkAPI.SourceAwsOrganizationalUnit{}, nil
	}

	return allOUs, nil
}

// ConnectSourceAwsOrganizationalUnit connects a new source AWS organizational unit
func (c *EonClient) ConnectSourceAwsOrganizationalUnit(ctx context.Context, req externalEonSdkAPI.ConnectSourceAwsOrganizationalUnitRequest) (*externalEonSdkAPI.SourceAwsOrganizationalUnit, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ConnectSourceAwsOrganizationalUnit(ctx, c.projectID).ConnectSourceAwsOrganizationalUnitRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to connect source AWS organizational unit"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	ou := resp.GetOrganizationalUnit()
	return &ou, nil
}

// DisconnectSourceAwsOrganizationalUnit disconnects a source AWS organizational unit
func (c *EonClient) DisconnectSourceAwsOrganizationalUnit(ctx context.Context, organizationalUnitId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	_, httpResp, err := c.client.AccountsAPI.DisconnectSourceAwsOrganizationalUnit(ctx, c.projectID, organizationalUnitId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to disconnect source AWS organizational unit"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// ListRestoreAwsOrganizationalUnits retrieves all restore AWS organizational units for the project.
// It paginates through all pages to return the complete list.
func (c *EonClient) ListRestoreAwsOrganizationalUnits(ctx context.Context) ([]externalEonSdkAPI.RestoreAwsOrganizationalUnit, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	var allOUs []externalEonSdkAPI.RestoreAwsOrganizationalUnit
	var pageToken string

	for {
		req := c.client.AccountsAPI.ListRestoreAwsOrganizationalUnits(ctx, c.projectID)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		resp, httpResp, err := req.Execute()

		if apiErr := c.handleAPIError(err, httpResp, "failed to list restore AWS organizational units"); apiErr != nil {
			return nil, apiErr
		}
		defer func() { _ = httpResp.Body.Close() }()

		if httpResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(httpResp.Body)
			return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
		}

		ous := resp.GetOrganizationalUnits()
		if ous != nil {
			allOUs = append(allOUs, ous...)
		}

		if !resp.HasNextToken() {
			break
		}
		pageToken = resp.GetNextToken()
	}

	if allOUs == nil {
		return []externalEonSdkAPI.RestoreAwsOrganizationalUnit{}, nil
	}

	return allOUs, nil
}

// ConnectRestoreAwsOrganizationalUnit connects a new restore AWS organizational unit
func (c *EonClient) ConnectRestoreAwsOrganizationalUnit(ctx context.Context, req externalEonSdkAPI.ConnectRestoreAwsOrganizationalUnitRequest) (*externalEonSdkAPI.RestoreAwsOrganizationalUnit, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ConnectRestoreAwsOrganizationalUnit(ctx, c.projectID).ConnectRestoreAwsOrganizationalUnitRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to connect restore AWS organizational unit"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	ou := resp.GetOrganizationalUnit()
	return &ou, nil
}

// DisconnectRestoreAwsOrganizationalUnit disconnects a restore AWS organizational unit
func (c *EonClient) DisconnectRestoreAwsOrganizationalUnit(ctx context.Context, organizationalUnitId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	_, httpResp, err := c.client.AccountsAPI.DisconnectRestoreAwsOrganizationalUnit(ctx, c.projectID, organizationalUnitId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to disconnect restore AWS organizational unit"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// ConnectRestoreAccount connects a new restore account
func (c *EonClient) ConnectRestoreAccount(ctx context.Context, req externalEonSdkAPI.ConnectRestoreAccountRequest) (*externalEonSdkAPI.RestoreAccount, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ConnectRestoreAccount(ctx, c.projectID).ConnectRestoreAccountRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to connect restore account"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	account := resp.GetRestoreAccount()
	return &account, nil
}

// DisconnectRestoreAccount severs the cloud connection for a restore account.
// The account record remains in the project with status DISCONNECTED.
func (c *EonClient) DisconnectRestoreAccount(ctx context.Context, accountId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	_, httpResp, err := c.client.AccountsAPI.DisconnectRestoreAccount(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to disconnect restore account"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// DeleteRestoreAccount fully removes a restore account from the project.
// Callers should typically Disconnect first if the account is still connected.
func (c *EonClient) DeleteRestoreAccount(ctx context.Context, accountId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	httpResp, err := c.client.AccountsAPI.DeleteRestoreAccountV1(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to delete restore account"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// UpdateRestoreAccountRequest contains the fields that can be updated on a restore account.
type UpdateRestoreAccountRequest struct {
	Name                     *string
	RestoreAccountAttributes *UpdateRestoreAccountAttributes
}

// UpdateRestoreAccountAttributes contains cloud-provider-specific attributes to update.
// The API supports updating AWS (role_arn) and GCP (service_account); Azure attributes
// are immutable and require replacement.
type UpdateRestoreAccountAttributes struct {
	Aws *UpdateAwsRestoreAccountAttributes
	Gcp *UpdateGcpRestoreAccountAttributes
}

// UpdateAwsRestoreAccountAttributes contains AWS-specific attributes to update.
type UpdateAwsRestoreAccountAttributes struct {
	RoleArn *string
}

// UpdateGcpRestoreAccountAttributes contains GCP-specific attributes to update.
type UpdateGcpRestoreAccountAttributes struct {
	ServiceAccount *string
}

// UpdateRestoreAccount updates mutable fields of a restore account via
// PATCH /v1/projects/{projectId}/restore-accounts/{accountId}.
func (c *EonClient) UpdateRestoreAccount(ctx context.Context, accountId string, req UpdateRestoreAccountRequest) (*externalEonSdkAPI.RestoreAccount, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	sdkReq := externalEonSdkAPI.NewUpdateRestoreAccountRequest()
	if req.Name != nil {
		sdkReq.SetName(*req.Name)
	}
	if req.RestoreAccountAttributes != nil {
		attrs := externalEonSdkAPI.NewUpdateRestoreAccountAttributesInput()
		if req.RestoreAccountAttributes.Aws != nil {
			awsAttrs := externalEonSdkAPI.UpdateAwsRestoreAccountAttributes{}
			if req.RestoreAccountAttributes.Aws.RoleArn != nil {
				awsAttrs.SetRoleArn(*req.RestoreAccountAttributes.Aws.RoleArn)
			}
			attrs.SetAws(awsAttrs)
		}
		if req.RestoreAccountAttributes.Gcp != nil {
			gcpAttrs := externalEonSdkAPI.UpdateGcpRestoreAccountAttributes{}
			if req.RestoreAccountAttributes.Gcp.ServiceAccount != nil {
				gcpAttrs.SetServiceAccount(*req.RestoreAccountAttributes.Gcp.ServiceAccount)
			}
			attrs.SetGcp(gcpAttrs)
		}
		sdkReq.SetRestoreAccountAttributes(*attrs)
	}

	resp, httpResp, err := c.client.AccountsAPI.UpdateRestoreAccount(ctx, c.projectID, accountId).
		UpdateRestoreAccountRequest(*sdkReq).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to update restore account"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	account := resp.GetRestoreAccount()
	return &account, nil
}

// ReconnectRestoreAccount reconnects a previously disconnected restore account.
func (c *EonClient) ReconnectRestoreAccount(ctx context.Context, accountId string) (*externalEonSdkAPI.RestoreAccount, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ReconnectRestoreAccount(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to reconnect restore account"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	account := resp.GetRestoreAccount()
	return &account, nil
}

// GetRestoreAccountConnectivityConfig retrieves the connectivity configuration of a restore account.
func (c *EonClient) GetRestoreAccountConnectivityConfig(ctx context.Context, accountId string) (*externalEonSdkAPI.RestoreAccountConnectivityConfig, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.GetRestoreAccountConnectivityConfig(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get restore account connectivity config"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	config := resp.GetRestoreAccountConfig()
	return &config, nil
}

// UpdateRestoreAccountConnectivityConfig updates the connectivity configuration of a restore account.
func (c *EonClient) UpdateRestoreAccountConnectivityConfig(ctx context.Context, accountId string, req externalEonSdkAPI.UpdateRestoreAccountConnectivityConfigRequest) (*externalEonSdkAPI.RestoreAccountConnectivityConfig, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.UpdateRestoreAccountConnectivityConfig(ctx, c.projectID, accountId).UpdateRestoreAccountConnectivityConfigRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to update restore account connectivity config"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	config := resp.GetRestoreAccountConfig()
	return &config, nil
}

// DeleteRestoreAccountConnectivityConfig deletes the connectivity configuration of a restore
// account, reverting it to the default settings.
func (c *EonClient) DeleteRestoreAccountConnectivityConfig(ctx context.Context, accountId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	httpResp, err := c.client.AccountsAPI.DeleteRestoreAccountConnectivityConfig(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to delete restore account connectivity config"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// GetRestoreAccountMetricsConfig retrieves the metrics configuration of a restore account.
func (c *EonClient) GetRestoreAccountMetricsConfig(ctx context.Context, accountId string) (*externalEonSdkAPI.RestoreAccountMetricsConfig, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.GetRestoreAccountMetricsConfig(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get restore account metrics config"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	config := resp.GetRestoreAccountConfig()
	return &config, nil
}

// EnableRestoreAccountMetricsConfig enables and configures job metrics for a restore account.
func (c *EonClient) EnableRestoreAccountMetricsConfig(ctx context.Context, accountId string, req externalEonSdkAPI.EnableRestoreAccountMetricsConfigRequest) (*externalEonSdkAPI.RestoreAccountMetricsConfig, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.EnableRestoreAccountMetricsConfig(ctx, c.projectID, accountId).EnableRestoreAccountMetricsConfigRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to enable restore account metrics config"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	config := resp.GetRestoreAccountConfig()
	return &config, nil
}

// DisableRestoreAccountMetricsConfig disables job metrics for a restore account.
func (c *EonClient) DisableRestoreAccountMetricsConfig(ctx context.Context, accountId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	httpResp, err := c.client.AccountsAPI.DisableRestoreAccountMetricsConfig(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to disable restore account metrics config"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// GetSourceAccountMetricsConfig retrieves the metrics configuration of a source account.
func (c *EonClient) GetSourceAccountMetricsConfig(ctx context.Context, accountId string) (*externalEonSdkAPI.SourceAccountMetricsConfig, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.GetSourceAccountMetricsConfig(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get source account metrics config"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	config := resp.GetSourceAccountConfig()
	return &config, nil
}

// EnableSourceAccountMetricsConfig enables and configures job metrics for a source account.
func (c *EonClient) EnableSourceAccountMetricsConfig(ctx context.Context, accountId string, req externalEonSdkAPI.EnableSourceAccountMetricsConfigRequest) (*externalEonSdkAPI.SourceAccountMetricsConfig, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.EnableSourceAccountMetricsConfig(ctx, c.projectID, accountId).EnableSourceAccountMetricsConfigRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to enable source account metrics config"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	config := resp.GetSourceAccountConfig()
	return &config, nil
}

// DisableSourceAccountMetricsConfig disables job metrics for a source account.
func (c *EonClient) DisableSourceAccountMetricsConfig(ctx context.Context, accountId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	httpResp, err := c.client.AccountsAPI.DisableSourceAccountMetricsConfig(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to disable source account metrics config"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// HoldSnapshot places a retention hold on a snapshot so it is not deleted by retention policy.
func (c *EonClient) HoldSnapshot(ctx context.Context, snapshotId string, req externalEonSdkAPI.HoldSnapshotRequest) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	_, httpResp, err := c.client.SnapshotsAPI.HoldSnapshot(ctx, c.projectID, snapshotId).HoldSnapshotRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to hold snapshot"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// RemoveSnapshotHold removes a retention hold from a snapshot.
func (c *EonClient) RemoveSnapshotHold(ctx context.Context, snapshotId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	_, httpResp, err := c.client.SnapshotsAPI.RemoveSnapshotHold(ctx, c.projectID, snapshotId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to remove snapshot hold"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// GetCloudResourceConfiguration retrieves the configuration of an inventory resource.
func (c *EonClient) GetCloudResourceConfiguration(ctx context.Context, resourceId string) (*CloudResourceConfiguration, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	baseURL := c.client.GetConfig().Servers[0].URL
	url := fmt.Sprintf("%s/v1/projects/%s/resources/%s/object-store-scan-method", baseURL, c.projectID, resourceId)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get resource configuration request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")
	for key, value := range c.client.GetConfig().DefaultHeader {
		httpReq.Header.Set(key, value)
	}

	httpResp, err := c.client.GetConfig().HTTPClient.Do(httpReq) // #nosec G704 -- URL is built from trusted server configuration, not user input
	if apiErr := c.handleAPIError(err, httpResp, "failed to get resource configuration"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	var response struct {
		ObjectStoreScanMethod CloudResourceConfiguration `json:"objectStoreScanMethod"`
	}
	if err := json.NewDecoder(httpResp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode resource configuration response: %w", err)
	}

	return &response.ObjectStoreScanMethod, nil
}

// UpdateCloudResourceConfiguration partially updates the configuration of an inventory resource.
func (c *EonClient) UpdateCloudResourceConfiguration(ctx context.Context, resourceId string, req UpdateCloudResourceConfigurationRequest) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	baseURL := c.client.GetConfig().Servers[0].URL
	url := fmt.Sprintf("%s/v1/projects/%s/resources/%s/object-store-scan-method", baseURL, c.projectID, resourceId)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal resource configuration request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create update resource configuration request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	for key, value := range c.client.GetConfig().DefaultHeader {
		httpReq.Header.Set(key, value)
	}

	httpResp, err := c.client.GetConfig().HTTPClient.Do(httpReq) // #nosec G704 -- URL is built from trusted server configuration, not user input
	if apiErr := c.handleAPIError(err, httpResp, "failed to update resource configuration"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusNoContent && httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}

// GetRestoreJob retrieves a restore job by ID
func (c *EonClient) GetRestoreJob(ctx context.Context, jobId string) (*externalEonSdkAPI.RestoreJob, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.JobsAPI.GetRestoreJob(ctx, c.projectID, jobId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get restore job"); apiErr != nil {
		return nil, apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	job := resp.GetJob()
	return &job, nil
}

// restoreJobIDFromResponse interprets the response from a restore endpoint that
// may be intercepted by a Multi-Party Approval (MPA) policy. On HTTP 202 the
// restore job started and its ID is read from the response body. On HTTP 201 the
// operation was intercepted by an MPA policy and cannot proceed without approval.
func restoreJobIDFromResponse(resp *externalEonSdkAPI.RestoreJobInitiationResponse, httpResp *http.Response, action string) (string, error) {
	switch httpResp.StatusCode {
	case http.StatusAccepted:
		return resp.GetJobId(), nil
	case http.StatusCreated:
		mpaReq := resp.GetActionApprovalRequest()
		return "", fmt.Errorf("%s: operation requires Multi-Party Approval (MPA request ID %q); approve the request and retry", action, mpaReq.GetId())
	default:
		body, _ := io.ReadAll(httpResp.Body)
		return "", fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}
}

// StartVolumeRestore starts a volume restore job
func (c *EonClient) StartVolumeRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreVolumeToEbsRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreEbsVolume(ctx, c.projectID, resourceId, snapshotId).RestoreVolumeToEbsRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to start volume restore"); apiErr != nil {
		return "", apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	return restoreJobIDFromResponse(resp, httpResp, "failed to start volume restore")
}

// GetResourceById retrieves a resource by ID
func (c *EonClient) GetResourceById(ctx context.Context, resourceId string) (*externalEonSdkAPI.InventoryResource, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.ResourcesAPI.GetResource(ctx, c.projectID, resourceId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get resource"); apiErr != nil {
		return nil, apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	resource := resp.GetResource()
	return &resource, nil
}

// StartRdsRestore starts an RDS restore job
func (c *EonClient) StartRdsRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreDbToRdsInstanceRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreDatabase(ctx, c.projectID, resourceId, snapshotId).RestoreDbToRdsInstanceRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to start RDS restore"); apiErr != nil {
		return "", apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	return restoreJobIDFromResponse(resp, httpResp, "failed to start RDS restore")
}

// StartEc2InstanceRestore starts an EC2 instance restore job
func (c *EonClient) StartEc2InstanceRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreAwsEc2InstanceRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreEc2Instance(ctx, c.projectID, resourceId, snapshotId).RestoreAwsEc2InstanceRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to start EC2 instance restore"); apiErr != nil {
		return "", apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	return restoreJobIDFromResponse(resp, httpResp, "failed to start EC2 instance restore")
}

// StartS3BucketRestore starts an S3 bucket restore job
func (c *EonClient) StartS3BucketRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreBucketRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreBucket(ctx, c.projectID, resourceId, snapshotId).RestoreBucketRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to start S3 bucket restore"); apiErr != nil {
		return "", apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	return restoreJobIDFromResponse(resp, httpResp, "failed to start S3 bucket restore")
}

// StartS3FileRestore starts an S3 file restore job
func (c *EonClient) StartS3FileRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreFilesRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreFiles(ctx, c.projectID, resourceId, snapshotId).RestoreFilesRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to start S3 file restore"); apiErr != nil {
		return "", apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(httpResp.Body)
		return "", fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return resp.GetJobId(), nil
}

// StartGcpVmInstanceRestore starts a GCP VM instance restore job
func (c *EonClient) StartGcpVmInstanceRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreGcpVmInstanceRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreGcpVmInstance(ctx, c.projectID, resourceId, snapshotId).RestoreGcpVmInstanceRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to start GCP VM instance restore"); apiErr != nil {
		return "", apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	return restoreJobIDFromResponse(resp, httpResp, "failed to start GCP VM instance restore")
}

// StartGcpDiskRestore starts a GCP disk restore job
func (c *EonClient) StartGcpDiskRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreGcpDiskRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreGcpDisk(ctx, c.projectID, resourceId, snapshotId).RestoreGcpDiskRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to start GCP disk restore"); apiErr != nil {
		return "", apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	return restoreJobIDFromResponse(resp, httpResp, "failed to start GCP disk restore")
}

// StartGcpCloudSqlRestore starts a GCP Cloud SQL restore job
func (c *EonClient) StartGcpCloudSqlRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreGcpCloudSqlRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreGcpCloudSql(ctx, c.projectID, resourceId, snapshotId).RestoreGcpCloudSqlRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to start GCP Cloud SQL restore"); apiErr != nil {
		return "", apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	return restoreJobIDFromResponse(resp, httpResp, "failed to start GCP Cloud SQL restore")
}

// StartAzureDiskRestore starts an Azure disk restore job.
func (c *EonClient) StartAzureDiskRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreAzureDiskRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreAzureDisk(ctx, c.projectID, resourceId, snapshotId).RestoreAzureDiskRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to start Azure disk restore"); apiErr != nil {
		return "", apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	return restoreJobIDFromResponse(resp, httpResp, "failed to start Azure disk restore")
}

// StartAzureSqlDatabaseRestore starts an Azure SQL database restore job.
func (c *EonClient) StartAzureSqlDatabaseRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreAzureSqlDatabaseRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreAzureSqlDatabase(ctx, c.projectID, resourceId, snapshotId).RestoreAzureSqlDatabaseRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to start Azure SQL database restore"); apiErr != nil {
		return "", apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	return restoreJobIDFromResponse(resp, httpResp, "failed to start Azure SQL database restore")
}

// StartAzureVmInstanceRestore starts an Azure VM instance restore job.
func (c *EonClient) StartAzureVmInstanceRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreAzureVmInstanceRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreAzureVmInstance(ctx, c.projectID, resourceId, snapshotId).RestoreAzureVmInstanceRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to start Azure VM instance restore"); apiErr != nil {
		return "", apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	return restoreJobIDFromResponse(resp, httpResp, "failed to start Azure VM instance restore")
}

// StartDynamoDBTableRestore starts a DynamoDB table restore job.
func (c *EonClient) StartDynamoDBTableRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreDynamoDBTableRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreDynamoDBTable(ctx, c.projectID, resourceId, snapshotId).RestoreDynamoDBTableRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to start DynamoDB table restore"); apiErr != nil {
		return "", apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	return restoreJobIDFromResponse(resp, httpResp, "failed to start DynamoDB table restore")
}

// StartEbsSnapshotRestore starts a restore-to-EBS-snapshot job.
func (c *EonClient) StartEbsSnapshotRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreVolumeToEbsSnapshotRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreToEbsSnapshot(ctx, c.projectID, resourceId, snapshotId).RestoreVolumeToEbsSnapshotRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to start EBS snapshot restore"); apiErr != nil {
		return "", apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	return restoreJobIDFromResponse(resp, httpResp, "failed to start EBS snapshot restore")
}

// BigQueryRestoreDestination represents the destination for a BigQuery dataset restore
type BigQueryRestoreDestination struct {
	DatasetId string `json:"datasetId"`
	Location  string `json:"location"`
}

// BigQueryRestoreRequest represents the request body for a BigQuery dataset restore
type BigQueryRestoreRequest struct {
	RestoreAccountId string                     `json:"restoreAccountId"`
	Destination      BigQueryRestoreDestination `json:"destination"`
	Tables           []string                   `json:"tables,omitempty"`
}

// BigQueryRestoreResponse represents the response from a BigQuery restore job initiation
type BigQueryRestoreResponse struct {
	JobId string `json:"jobId"`
}

// StartBigQueryDatasetRestore starts a BigQuery dataset restore job
func (c *EonClient) StartBigQueryDatasetRestore(ctx context.Context, resourceId, snapshotId string, req BigQueryRestoreRequest) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	// Build the URL for BigQuery dataset restore
	baseURL := c.client.GetConfig().Servers[0].URL
	url := fmt.Sprintf("%s/v1/projects/%s/resources/%s/snapshots/%s/restore-bigquery-dataset", baseURL, c.projectID, resourceId, snapshotId)

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal BigQuery restore request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create BigQuery restore request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Copy auth and default headers from the SDK client config
	for key, value := range c.client.GetConfig().DefaultHeader {
		httpReq.Header.Set(key, value)
	}

	httpResp, err := c.client.GetConfig().HTTPClient.Do(httpReq) // #nosec G704 -- URL is built from trusted server configuration, not user input
	if apiErr := c.handleAPIError(err, httpResp, "failed to start BigQuery dataset restore"); apiErr != nil {
		return "", apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(httpResp.Body)
		return "", fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(respBody))
	}

	var restoreResp BigQueryRestoreResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&restoreResp); err != nil {
		return "", fmt.Errorf("failed to decode BigQuery restore response: %w", err)
	}

	return restoreResp.JobId, nil
}

// ListPermissions retrieves all available permissions for composing custom roles.
func (c *EonClient) ListPermissions(ctx context.Context) ([]externalEonSdkAPI.Permission, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.IamAPI.ListPermissions(ctx).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to list permissions"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	permissions := resp.GetPermissions()
	if permissions == nil {
		return []externalEonSdkAPI.Permission{}, nil
	}
	return permissions, nil
}

// ExcludeResourceFromBackup excludes an entire inventory resource from future backups.
func (c *EonClient) ExcludeResourceFromBackup(ctx context.Context, resourceId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	_, httpResp, err := c.client.ResourcesAPI.ExcludeResourceFromBackup(ctx, c.projectID, resourceId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to exclude resource from backup"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}

// CancelResourceBackupExclusion cancels a resource backup exclusion, including it in future backups.
func (c *EonClient) CancelResourceBackupExclusion(ctx context.Context, resourceId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	_, httpResp, err := c.client.ResourcesAPI.CancelResourceBackupExclusion(ctx, c.projectID, resourceId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to cancel resource backup exclusion"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}

// OverrideDataClasses manually sets a resource's data classes, overriding auto-classification.
func (c *EonClient) OverrideDataClasses(ctx context.Context, resourceId string, dataClasses []string) ([]string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	req := *externalEonSdkAPI.NewOverrideDataClassificationsRequest()
	req.SetDataClasses(dataClasses)

	resp, httpResp, err := c.client.ResourcesAPI.OverrideDataClasses(ctx, c.projectID, resourceId).OverrideDataClassificationsRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to override data classes"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	overridden := resp.GetDataClasses()
	if overridden == nil {
		return []string{}, nil
	}
	return overridden, nil
}

// RemoveDataClassesOverride removes a resource's data classes override, re-enabling auto-classification.
func (c *EonClient) RemoveDataClassesOverride(ctx context.Context, resourceId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	httpResp, err := c.client.ResourcesAPI.RemoveDataClassesOverride(ctx, c.projectID, resourceId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to remove data classes override"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}

// OverrideEnvironment manually sets a resource's environment, overriding auto-classification.
func (c *EonClient) OverrideEnvironment(ctx context.Context, resourceId string, environment string) (string, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	req := *externalEonSdkAPI.NewOverrideEnvironmentRequest()
	req.SetEnvironment(externalEonSdkAPI.Environment(environment))

	resp, httpResp, err := c.client.ResourcesAPI.OverrideEnvironment(ctx, c.projectID, resourceId).OverrideEnvironmentRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to override environment"); apiErr != nil {
		return "", apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return "", &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	return string(resp.GetEnvironment()), nil
}

// RemoveEnvironmentOverride removes a resource's environment override, re-enabling auto-classification.
func (c *EonClient) RemoveEnvironmentOverride(ctx context.Context, resourceId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	httpResp, err := c.client.ResourcesAPI.RemoveEnvironmentOverride(ctx, c.projectID, resourceId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to remove environment override"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}

// ListResourceSnapshots retrieves snapshots for an inventory resource, optionally filtered.
// It paginates through all pages to return the complete list.
func (c *EonClient) ListResourceSnapshots(ctx context.Context, resourceId string, filters *externalEonSdkAPI.SnapshotFilterConditions) ([]externalEonSdkAPI.Snapshot, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	listReq := *externalEonSdkAPI.NewListInventorySnapshotsRequest()
	if filters != nil {
		listReq.SetFilters(*filters)
	}

	var all []externalEonSdkAPI.Snapshot
	var pageToken string

	for {
		req := c.client.SnapshotsAPI.ListResourceSnapshots(ctx, c.projectID, resourceId).
			PageSize(100).
			ListInventorySnapshotsRequest(listReq)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		resp, httpResp, err := req.Execute()
		if apiErr := c.handleAPIError(err, httpResp, "failed to list resource snapshots"); apiErr != nil {
			if httpResp != nil {
				_ = httpResp.Body.Close()
			}
			return nil, apiErr
		}

		if httpResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(httpResp.Body)
			_ = httpResp.Body.Close()
			return nil, &APIError{
				StatusCode: httpResp.StatusCode,
				Message:    string(body),
			}
		}

		if resp.GetSnapshots() != nil {
			all = append(all, resp.GetSnapshots()...)
		}

		hasMore := resp.HasNextToken() && resp.GetNextToken() != ""
		_ = httpResp.Body.Close()
		if !hasMore {
			break
		}
		pageToken = resp.GetNextToken()
	}

	if all == nil {
		return []externalEonSdkAPI.Snapshot{}, nil
	}
	return all, nil
}

// ExcludeVolumeFromBackup excludes an EBS volume from future EC2 instance backups
func (c *EonClient) ExcludeVolumeFromBackup(ctx context.Context, resourceId, volumeId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	baseURL := c.client.GetConfig().Servers[0].URL
	url := fmt.Sprintf("%s/v1/projects/%s/resources/%s/volumes/%s/exclude", baseURL, c.projectID, resourceId, volumeId)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create exclude volume request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")
	for key, value := range c.client.GetConfig().DefaultHeader {
		httpReq.Header.Set(key, value)
	}

	httpResp, err := c.client.GetConfig().HTTPClient.Do(httpReq) // #nosec G704 -- URL is built from trusted server configuration, not user input
	if apiErr := c.handleAPIError(err, httpResp, "failed to exclude volume from backup"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(respBody))
	}

	return nil
}

// CancelVolumeBackupExclusion cancels the backup exclusion of a volume, including it in future backups
func (c *EonClient) CancelVolumeBackupExclusion(ctx context.Context, resourceId, volumeId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	baseURL := c.client.GetConfig().Servers[0].URL
	url := fmt.Sprintf("%s/v1/projects/%s/resources/%s/volumes/%s/include", baseURL, c.projectID, resourceId, volumeId)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create cancel volume exclusion request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")
	for key, value := range c.client.GetConfig().DefaultHeader {
		httpReq.Header.Set(key, value)
	}

	httpResp, err := c.client.GetConfig().HTTPClient.Do(httpReq) // #nosec G704 -- URL is built from trusted server configuration, not user input
	if apiErr := c.handleAPIError(err, httpResp, "failed to cancel volume backup exclusion"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(respBody))
	}

	return nil
}

// GetSnapshot retrieves a snapshot by ID
func (c *EonClient) GetSnapshot(ctx context.Context, snapshotId string) (*externalEonSdkAPI.Snapshot, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.GetSnapshot(ctx, c.projectID, snapshotId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get snapshot"); apiErr != nil {
		return nil, apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	snapshot := resp.GetSnapshot()
	return &snapshot, nil
}

// WaitForRestoreJobCompletion waits for a restore job to complete
func (c *EonClient) WaitForRestoreJobCompletion(ctx context.Context, jobId string, timeout time.Duration) (*externalEonSdkAPI.RestoreJob, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for restore job %s to complete", jobId)
		case <-ticker.C:
			job, err := c.GetRestoreJob(ctx, jobId)
			if err != nil {
				return nil, fmt.Errorf("failed to get restore job status: %w", err)
			}

			if job.GetJobExecutionDetails().Status.Ptr() == nil {
				continue
			}

			switch job.GetJobExecutionDetails().Status {
			case externalEonSdkAPI.JOB_COMPLETED, externalEonSdkAPI.JOB_PARTIAL:
				return job, nil
			case externalEonSdkAPI.JOB_FAILED, externalEonSdkAPI.JOB_CANCELED:
				errorMsg := "unknown error"
				if job.GetJobExecutionDetails().StatusMessage != nil {
					errorMsg = *job.GetJobExecutionDetails().StatusMessage
				}
				return job, fmt.Errorf("restore job failed with status: %s, error: %s", job.GetJobExecutionDetails().Status, errorMsg)
			}
		}
	}
}

// ListBackupPolicies retrieves all backup policies for the project
func (c *EonClient) ListBackupPolicies(ctx context.Context) ([]externalEonSdkAPI.BackupPolicy, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.BackupPoliciesAPI.ListBackupPolicies(ctx, c.projectID).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to list backup policies"); apiErr != nil {
		return nil, apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return resp.GetBackupPolicies(), nil
}

// GetBackupPolicy retrieves a backup policy by ID
func (c *EonClient) GetBackupPolicy(ctx context.Context, policyId string) (*externalEonSdkAPI.BackupPolicy, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.BackupPoliciesAPI.GetBackupPolicy(ctx, c.projectID, policyId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get backup policy"); apiErr != nil {
		return nil, apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	policy := resp.GetBackupPolicy()
	return &policy, nil
}

// CreateBackupPolicy creates a new backup policy
func (c *EonClient) CreateBackupPolicy(ctx context.Context, req externalEonSdkAPI.CreateBackupPolicyRequest) (*externalEonSdkAPI.BackupPolicy, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.BackupPoliciesAPI.CreateBackupPolicy(ctx, c.projectID).CreateBackupPolicyRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to create backup policy"); apiErr != nil {
		return nil, apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	policy := resp.GetBackupPolicy()
	return &policy, nil
}

// UpdateBackupPolicy updates an existing backup policy
func (c *EonClient) UpdateBackupPolicy(ctx context.Context, policyId string, req externalEonSdkAPI.UpdateBackupPolicyRequest) (*externalEonSdkAPI.BackupPolicy, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.BackupPoliciesAPI.UpdateBackupPolicy(ctx, c.projectID, policyId).UpdateBackupPolicyRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to update backup policy"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	policy := resp.GetBackupPolicy()
	return &policy, nil
}

// DeleteBackupPolicy deletes a backup policy
func (c *EonClient) DeleteBackupPolicy(ctx context.Context, policyId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	httpResp, err := c.client.BackupPoliciesAPI.DeleteBackupPolicy(ctx, c.projectID, policyId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to delete backup policy"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// GetBackupPostureControl retrieves a backup posture control by ID
func (c *EonClient) GetBackupPostureControl(ctx context.Context, controlId string) (*externalEonSdkAPI.BackupPostureControl, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.BackupPostureControlsAPI.GetBackupPostureControl(ctx, c.projectID, controlId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get backup posture control"); apiErr != nil {
		return nil, apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return resp, nil
}

// CreateBackupPostureControl creates a new backup posture control
func (c *EonClient) CreateBackupPostureControl(ctx context.Context, req externalEonSdkAPI.CreateBackupPostureControlRequest) (*externalEonSdkAPI.BackupPostureControl, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.BackupPostureControlsAPI.CreateBackupPostureControl(ctx, c.projectID).CreateBackupPostureControlRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to create backup posture control"); apiErr != nil {
		return nil, apiErr
	}

	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return resp, nil
}

// UpdateBackupPostureControl updates an existing backup posture control
func (c *EonClient) UpdateBackupPostureControl(ctx context.Context, controlId string, req externalEonSdkAPI.UpdateBackupPostureControlRequest) (*externalEonSdkAPI.BackupPostureControl, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.BackupPostureControlsAPI.UpdateBackupPostureControl(ctx, c.projectID, controlId).UpdateBackupPostureControlRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to update backup posture control"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return resp, nil
}

// DeleteBackupPostureControl deletes a backup posture control
func (c *EonClient) DeleteBackupPostureControl(ctx context.Context, controlId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	httpResp, err := c.client.BackupPostureControlsAPI.DeleteBackupPostureControl(ctx, c.projectID, controlId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to delete backup posture control"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// ListBackupPostureControls retrieves all backup posture controls for the project.
func (c *EonClient) ListBackupPostureControls(ctx context.Context) ([]externalEonSdkAPI.BackupPostureControl, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	var all []externalEonSdkAPI.BackupPostureControl
	var pageToken string

	for {
		req := c.client.BackupPostureControlsAPI.ListBackupPostureControls(ctx, c.projectID).
			ListBackupPostureControlsRequest(*externalEonSdkAPI.NewListBackupPostureControlsRequest()).
			PageSize(100)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		resp, httpResp, err := req.Execute()
		if apiErr := c.handleAPIError(err, httpResp, "failed to list backup posture controls"); apiErr != nil {
			if httpResp != nil {
				_ = httpResp.Body.Close()
			}
			return nil, apiErr
		}

		if httpResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(httpResp.Body)
			_ = httpResp.Body.Close()
			return nil, &APIError{
				StatusCode: httpResp.StatusCode,
				Message:    string(body),
			}
		}

		if resp.GetBackupPostureControls() != nil {
			all = append(all, resp.GetBackupPostureControls()...)
		}

		hasMore := resp.HasNextPageToken() && resp.GetNextPageToken() != ""
		_ = httpResp.Body.Close()
		if !hasMore {
			break
		}
		pageToken = resp.GetNextPageToken()
	}

	if all == nil {
		return []externalEonSdkAPI.BackupPostureControl{}, nil
	}
	return all, nil
}

// CreateVault creates a new backup vault
func (c *EonClient) CreateVault(ctx context.Context, req externalEonSdkAPI.CreateVaultRequest) (*externalEonSdkAPI.BackupVault, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.VaultsAPI.CreateVault(ctx, c.projectID).CreateVaultRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to create vault"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	vault := resp.GetVault()
	return &vault, nil
}

// GetVault retrieves a vault by ID
func (c *EonClient) GetVault(ctx context.Context, vaultId string) (*externalEonSdkAPI.BackupVault, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.VaultsAPI.GetVault(ctx, c.projectID, vaultId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get vault"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	vault := resp.GetVault()
	return &vault, nil
}

// UpdateVault updates a vault's display name
func (c *EonClient) UpdateVault(ctx context.Context, vaultId string, req externalEonSdkAPI.UpdateVaultRequest) (*externalEonSdkAPI.BackupVault, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.VaultsAPI.UpdateVault(ctx, c.projectID, vaultId).UpdateVaultRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to update vault"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	vault := resp.GetVault()
	return &vault, nil
}

// ListVaults retrieves all vaults for the project
func (c *EonClient) ListVaults(ctx context.Context) ([]externalEonSdkAPI.BackupVault, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	var allVaults []externalEonSdkAPI.BackupVault
	var pageToken *string

	// Handle pagination to fetch all vaults
	for {
		req := c.client.VaultsAPI.ListVaults(ctx, c.projectID)
		if pageToken != nil {
			req = req.PageToken(*pageToken)
		}

		resp, httpResp, err := req.Execute()
		if apiErr := c.handleAPIError(err, httpResp, "failed to list vaults"); apiErr != nil {
			if httpResp != nil {
				_ = httpResp.Body.Close()
			}
			return nil, apiErr
		}

		if httpResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(httpResp.Body)
			_ = httpResp.Body.Close()
			return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
		}

		if resp.GetVaults() != nil {
			allVaults = append(allVaults, resp.GetVaults()...)
		}

		// Check if there are more pages
		hasMorePages := resp.NextToken != nil && *resp.NextToken != ""

		// Close response body immediately after processing
		_ = httpResp.Body.Close()

		if !hasMorePages {
			break
		}
		pageToken = resp.NextToken
	}

	return allVaults, nil
}

// CreateIdpGroup creates a new IDP group and assigns roles to it.
func (c *EonClient) CreateIdpGroup(ctx context.Context, req externalEonSdkAPI.CreateIdpGroupRequest) (*externalEonSdkAPI.IdpGroup, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.IamAPI.CreateIdpGroup(ctx).CreateIdpGroupRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to create IDP group"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	group := resp.GetGroup()
	return &group, nil
}

// GetIdpGroup retrieves an IDP group by ID.
func (c *EonClient) GetIdpGroup(ctx context.Context, groupId string) (*externalEonSdkAPI.IdpGroup, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.IamAPI.GetIdpGroup(ctx, groupId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get IDP group"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	group := resp.GetGroup()
	return &group, nil
}

// ListIdpGroups retrieves all IDP groups for the account.
func (c *EonClient) ListIdpGroups(ctx context.Context) ([]externalEonSdkAPI.IdpGroup, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	var allGroups []externalEonSdkAPI.IdpGroup
	var pageToken *string

	for {
		req := c.client.IamAPI.ListIdpGroups(ctx).PageSize(100)
		if pageToken != nil {
			req = req.PageToken(*pageToken)
		}

		resp, httpResp, err := req.Execute()
		if apiErr := c.handleAPIError(err, httpResp, "failed to list IDP groups"); apiErr != nil {
			if httpResp != nil {
				_ = httpResp.Body.Close()
			}
			return nil, apiErr
		}

		if httpResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(httpResp.Body)
			_ = httpResp.Body.Close()
			return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
		}

		if resp.GetGroups() != nil {
			allGroups = append(allGroups, resp.GetGroups()...)
		}

		hasMorePages := resp.NextToken != nil && *resp.NextToken != ""
		_ = httpResp.Body.Close()

		if !hasMorePages {
			break
		}
		pageToken = resp.NextToken
	}

	if allGroups == nil {
		return []externalEonSdkAPI.IdpGroup{}, nil
	}
	return allGroups, nil
}

// ListIdps retrieves all identity providers for the account.
func (c *EonClient) ListIdps(ctx context.Context) ([]externalEonSdkAPI.Idp, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	var all []externalEonSdkAPI.Idp
	var pageToken *string

	for {
		req := c.client.IamAPI.ListIdps(ctx).PageSize(100)
		if pageToken != nil {
			req = req.PageToken(*pageToken)
		}

		resp, httpResp, err := req.Execute()
		if apiErr := c.handleAPIError(err, httpResp, "failed to list identity providers"); apiErr != nil {
			if httpResp != nil {
				_ = httpResp.Body.Close()
			}
			return nil, apiErr
		}

		if httpResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(httpResp.Body)
			_ = httpResp.Body.Close()
			return nil, &APIError{
				StatusCode: httpResp.StatusCode,
				Message:    string(body),
			}
		}

		if resp.GetIdps() != nil {
			all = append(all, resp.GetIdps()...)
		}

		hasMore := resp.HasNextToken() && resp.GetNextToken() != ""
		_ = httpResp.Body.Close()
		if !hasMore {
			break
		}
		token := resp.GetNextToken()
		pageToken = &token
	}

	if all == nil {
		return []externalEonSdkAPI.Idp{}, nil
	}
	return all, nil
}

// UpdateIdpGroup updates the role assignments for an IDP group.
func (c *EonClient) UpdateIdpGroup(ctx context.Context, groupId string, req externalEonSdkAPI.UpdateIdpGroupRequest) (*externalEonSdkAPI.IdpGroup, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.IamAPI.UpdateIdpGroup(ctx, groupId).UpdateIdpGroupRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to update IDP group"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	group := resp.GetGroup()
	return &group, nil
}

// DeleteIdpGroup deletes an IDP group and all its role assignments.
func (c *EonClient) DeleteIdpGroup(ctx context.Context, groupId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	httpResp, err := c.client.IamAPI.DeleteIdpGroup(ctx, groupId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to delete IDP group"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// ListRoles retrieves all roles for the account (paginated).
func (c *EonClient) ListRoles(ctx context.Context) ([]externalEonSdkAPI.Role, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	var allRoles []externalEonSdkAPI.Role
	var pageToken *string

	for {
		req := c.client.IamAPI.ListRoles(ctx).PageSize(100)
		if pageToken != nil {
			req = req.PageToken(*pageToken)
		}

		resp, httpResp, err := req.Execute()
		if apiErr := c.handleAPIError(err, httpResp, "failed to list roles"); apiErr != nil {
			if httpResp != nil {
				_ = httpResp.Body.Close()
			}
			return nil, apiErr
		}

		if httpResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(httpResp.Body)
			_ = httpResp.Body.Close()
			return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
		}

		if resp.GetRoles() != nil {
			allRoles = append(allRoles, resp.GetRoles()...)
		}

		hasMorePages := resp.NextToken != nil && *resp.NextToken != ""
		_ = httpResp.Body.Close()

		if !hasMorePages {
			break
		}
		pageToken = resp.NextToken
	}

	if allRoles == nil {
		return []externalEonSdkAPI.Role{}, nil
	}
	return allRoles, nil
}

// GetRole retrieves a role by ID.
func (c *EonClient) GetRole(ctx context.Context, roleId string) (*externalEonSdkAPI.Role, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.IamAPI.GetRole(ctx, roleId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get role"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	role := resp.GetRole()
	return &role, nil
}

// CreateRole creates a new custom role.
func (c *EonClient) CreateRole(ctx context.Context, req externalEonSdkAPI.CreateRoleRequest) (*externalEonSdkAPI.Role, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.IamAPI.CreateRole(ctx).CreateRoleRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to create role"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	role := resp.GetRole()
	return &role, nil
}

// UpdateRole updates an existing role.
func (c *EonClient) UpdateRole(ctx context.Context, roleId string, req externalEonSdkAPI.UpdateRoleRequest) (*externalEonSdkAPI.Role, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.IamAPI.UpdateRole(ctx, roleId).UpdateRoleRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to update role"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	role := resp.GetRole()
	return &role, nil
}

// DeleteRole deletes a role.
func (c *EonClient) DeleteRole(ctx context.Context, roleId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	httpResp, err := c.client.IamAPI.DeleteRole(ctx, roleId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to delete role"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// CreateActionApprovalRule creates a new action approval rule.
func (c *EonClient) CreateActionApprovalRule(ctx context.Context, req externalEonSdkAPI.CreateActionApprovalRuleRequest) (*externalEonSdkAPI.ActionApprovalRule, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.ActionApprovalsAPI.CreateActionApprovalRule(ctx, c.projectID).CreateActionApprovalRuleRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to create action approval rule"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	rule, ok := resp.GetActionApprovalRuleOk()
	if !ok || rule == nil {
		return nil, fmt.Errorf("API returned empty action approval rule")
	}
	return rule, nil
}

// GetActionApprovalRule retrieves an action approval rule by ID.
func (c *EonClient) GetActionApprovalRule(ctx context.Context, ruleId string) (*externalEonSdkAPI.ActionApprovalRule, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.ActionApprovalsAPI.GetActionApprovalRule(ctx, c.projectID, ruleId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get action approval rule"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	rule, ok := resp.GetActionApprovalRuleOk()
	if !ok || rule == nil {
		return nil, &APIError{StatusCode: http.StatusNotFound, Message: "action approval rule not found"}
	}
	return rule, nil
}

// UpdateActionApprovalRule updates an existing action approval rule.
func (c *EonClient) UpdateActionApprovalRule(ctx context.Context, ruleId string, req externalEonSdkAPI.UpdateActionApprovalRuleRequest) (*externalEonSdkAPI.ActionApprovalRule, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.ActionApprovalsAPI.UpdateActionApprovalRule(ctx, c.projectID, ruleId).UpdateActionApprovalRuleRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to update action approval rule"); apiErr != nil {
		return nil, apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	rule, ok := resp.GetActionApprovalRuleOk()
	if !ok || rule == nil {
		return nil, fmt.Errorf("API returned empty action approval rule")
	}
	return rule, nil
}

// DeleteActionApprovalRule deletes an action approval rule.
func (c *EonClient) DeleteActionApprovalRule(ctx context.Context, ruleId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	httpResp, err := c.client.ActionApprovalsAPI.DeleteActionApprovalRule(ctx, c.projectID, ruleId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to delete action approval rule"); apiErr != nil {
		return apiErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// ListActionApprovalRules retrieves all action approval rules for the project.
func (c *EonClient) ListActionApprovalRules(ctx context.Context) ([]externalEonSdkAPI.ActionApprovalRule, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	var all []externalEonSdkAPI.ActionApprovalRule
	var pageToken string

	for {
		req := c.client.ActionApprovalsAPI.ListActionApprovalRules(ctx, c.projectID).PageSize(100)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		resp, httpResp, err := req.Execute()
		if apiErr := c.handleAPIError(err, httpResp, "failed to list action approval rules"); apiErr != nil {
			if httpResp != nil {
				_ = httpResp.Body.Close()
			}
			return nil, apiErr
		}

		if httpResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(httpResp.Body)
			_ = httpResp.Body.Close()
			return nil, &APIError{
				StatusCode: httpResp.StatusCode,
				Message:    string(body),
			}
		}

		if resp.GetActionApprovalRules() != nil {
			all = append(all, resp.GetActionApprovalRules()...)
		}

		hasMore := resp.HasNextPageToken() && resp.GetNextPageToken() != ""
		_ = httpResp.Body.Close()
		if !hasMore {
			break
		}
		pageToken = resp.GetNextPageToken()
	}

	if all == nil {
		return []externalEonSdkAPI.ActionApprovalRule{}, nil
	}
	return all, nil
}

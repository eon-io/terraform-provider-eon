package client

import (
	"context"
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

// GetProjectID returns the project ID
func (c *EonClient) GetProjectID() string {
	return c.projectID
}

// handleAPIError processes API errors and extracts detailed error information from HTTP responses
func (c *EonClient) handleAPIError(err error, httpResp *http.Response, baseErrorMsg string) error {
	if err != nil && httpResp != nil {
		defer httpResp.Body.Close()
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

// ListSourceAccounts retrieves all source accounts for the project
func (c *EonClient) ListSourceAccounts(ctx context.Context) ([]externalEonSdkAPI.SourceAccount, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ListSourceAccounts(ctx, c.projectID).ListSourceAccountsRequest(externalEonSdkAPI.ListSourceAccountsRequest{}).Execute()

	if apiErr := c.handleAPIError(err, httpResp, "failed to list source accounts"); apiErr != nil {
		return nil, apiErr
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	if resp.GetAccounts() == nil {
		return []externalEonSdkAPI.SourceAccount{}, nil
	}

	return resp.GetAccounts(), nil
}

// ListRestoreAccounts retrieves all restore accounts for the project
func (c *EonClient) ListRestoreAccounts(ctx context.Context) ([]externalEonSdkAPI.RestoreAccount, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ListRestoreAccounts(ctx, c.projectID).ListRestoreAccountsRequest(externalEonSdkAPI.ListRestoreAccountsRequest{}).Execute()

	if apiErr := c.handleAPIError(err, httpResp, "failed to list restore accounts"); apiErr != nil {
		return nil, apiErr
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	if resp.GetAccounts() == nil {
		return []externalEonSdkAPI.RestoreAccount{}, nil
	}

	return resp.GetAccounts(), nil
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
	defer httpResp.Body.Close()

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
	defer httpResp.Body.Close()

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
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	account := resp.GetRestoreAccount()
	return &account, nil
}

// DisconnectRestoreAccount disconnects a restore account
func (c *EonClient) DisconnectRestoreAccount(ctx context.Context, accountId string) error {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	_, httpResp, err := c.client.AccountsAPI.DisconnectRestoreAccount(ctx, c.projectID, accountId).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to disconnect restore account"); apiErr != nil {
		return apiErr
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
}

// GetRestoreJob retrieves a restore job by ID
func (c *EonClient) GetRestoreJob(ctx context.Context, jobId string) (*externalEonSdkAPI.RestoreJob, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.JobsAPI.GetRestoreJob(ctx, jobId, c.projectID).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get restore job"); apiErr != nil {
		return nil, apiErr
	}

	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	job := resp.GetJob()
	return &job, nil
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

	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(httpResp.Body)
		return "", fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return resp.GetJobId(), nil
}

// GetResourceById retrieves a resource by ID
func (c *EonClient) GetResourceById(ctx context.Context, resourceId string) (*externalEonSdkAPI.InventoryResource, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.ResourcesAPI.GetResource(ctx, resourceId, c.projectID).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get resource"); apiErr != nil {
		return nil, apiErr
	}

	defer httpResp.Body.Close()

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

	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(httpResp.Body)
		return "", fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return resp.GetJobId(), nil
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

	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(httpResp.Body)
		return "", fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return resp.GetJobId(), nil
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

	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(httpResp.Body)
		return "", fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return resp.GetJobId(), nil
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

	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(httpResp.Body)
		return "", fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return resp.GetJobId(), nil
}

// GetSnapshot retrieves a snapshot by ID
func (c *EonClient) GetSnapshot(ctx context.Context, snapshotId string) (*externalEonSdkAPI.Snapshot, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.GetSnapshot(ctx, snapshotId, c.projectID).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get snapshot"); apiErr != nil {
		return nil, apiErr
	}

	defer httpResp.Body.Close()

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

	defer httpResp.Body.Close()

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

	resp, httpResp, err := c.client.BackupPoliciesAPI.GetBackupPolicy(ctx, policyId, c.projectID).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get backup policy"); apiErr != nil {
		return nil, apiErr
	}

	defer httpResp.Body.Close()

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

	defer httpResp.Body.Close()

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

	resp, httpResp, err := c.client.BackupPoliciesAPI.UpdateBackupPolicy(ctx, policyId, c.projectID).UpdateBackupPolicyRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to update backup policy"); apiErr != nil {
		return nil, apiErr
	}
	defer httpResp.Body.Close()

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

	httpResp, err := c.client.BackupPoliciesAPI.DeleteBackupPolicy(ctx, policyId, c.projectID).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to delete backup policy"); apiErr != nil {
		return apiErr
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	return nil
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
	defer httpResp.Body.Close()

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

	resp, httpResp, err := c.client.VaultsAPI.GetVault(ctx, vaultId, c.projectID).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to get vault"); apiErr != nil {
		return nil, apiErr
	}
	defer httpResp.Body.Close()

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

	resp, httpResp, err := c.client.VaultsAPI.UpdateVault(ctx, vaultId, c.projectID).UpdateVaultRequest(req).Execute()
	if apiErr := c.handleAPIError(err, httpResp, "failed to update vault"); apiErr != nil {
		return nil, apiErr
	}
	defer httpResp.Body.Close()

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

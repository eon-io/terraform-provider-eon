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
	authToken    string
	tokenExpiry  time.Time
	clientID     string
	clientSecret string
	endpoint     string
}

// NewEonClient creates a new Eon API client with the provided configuration
func NewEonClient(endpoint, clientID, clientSecret, projectID string) (*EonClient, error) {
	config := externalEonSdkAPI.NewConfiguration()
	config.Servers = []externalEonSdkAPI.ServerConfiguration{
		{
			URL: fmt.Sprintf("%s/api", endpoint),
		},
	}

	client := &EonClient{
		client:       externalEonSdkAPI.NewAPIClient(config),
		ProjectID:    projectID,
		clientID:     clientID,
		clientSecret: clientSecret,
		endpoint:     endpoint,
	}

	if err := client.authenticate(); err != nil {
		return nil, fmt.Errorf("failed to authenticate with Eon API: %w", err)
	}

	return client, nil
}

// authenticate performs OAuth authentication with the Eon API
func (c *EonClient) authenticate() error {
	resp, httpResp, err := c.client.AuthAPI.GetAccessToken(context.Background()).ApiCredentials(externalEonSdkAPI.ApiCredentials{
		ClientId:     c.clientID,
		ClientSecret: c.clientSecret,
	}).Execute()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("authentication failed with status %d: %s", httpResp.StatusCode, body)
	}

	c.authToken = resp.GetAccessToken()
	c.tokenExpiry = time.Now().Add(time.Duration(resp.GetExpirationSeconds()) * time.Second)

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

// handleAPIError processes API errors and extracts detailed error information from HTTP responses
func (c *EonClient) handleAPIError(err error, httpResp *http.Response, baseErrorMsg string) error {
	if err != nil && httpResp != nil {
		defer httpResp.Body.Close()
		if body, readErr := io.ReadAll(httpResp.Body); readErr == nil && len(body) > 0 {
			return fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
		}
		return fmt.Errorf("%s: %w", baseErrorMsg, err)
	} else if err != nil {
		return fmt.Errorf("%s: %w", baseErrorMsg, err)
	}
	return nil
}

// ListSourceAccounts retrieves all source accounts for the project
func (c *EonClient) ListSourceAccounts(ctx context.Context) ([]externalEonSdkAPI.SourceAccount, error) {
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ListSourceAccounts(ctx, c.ProjectID).ListSourceAccountsRequest(externalEonSdkAPI.ListSourceAccountsRequest{}).Execute()

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
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ListRestoreAccounts(ctx, c.ProjectID).ListRestoreAccountsRequest(externalEonSdkAPI.ListRestoreAccountsRequest{}).Execute()

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
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ConnectSourceAccount(ctx, c.ProjectID).ConnectSourceAccountRequest(req).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	_, httpResp, err := c.client.AccountsAPI.DisconnectSourceAccount(ctx, c.ProjectID, accountId).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.AccountsAPI.ConnectRestoreAccount(ctx, c.ProjectID).ConnectRestoreAccountRequest(req).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	_, httpResp, err := c.client.AccountsAPI.DisconnectRestoreAccount(ctx, c.ProjectID, accountId).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.JobsAPI.GetRestoreJob(ctx, jobId, c.ProjectID).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreEbsVolume(ctx, c.ProjectID, resourceId, snapshotId).RestoreVolumeToEbsRequest(req).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.ResourcesAPI.GetResource(ctx, resourceId, c.ProjectID).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreDatabase(ctx, c.ProjectID, resourceId, snapshotId).RestoreDbToRdsInstanceRequest(req).Execute()
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
func (c *EonClient) StartEc2InstanceRestore(ctx context.Context, resourceId, snapshotId string, req externalEonSdkAPI.RestoreInstanceInput) (string, error) {
	if err := c.ensureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreEc2Instance(ctx, c.ProjectID, resourceId, snapshotId).RestoreInstanceInput(req).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreBucket(ctx, c.ProjectID, resourceId, snapshotId).RestoreBucketRequest(req).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.RestoreFiles(ctx, c.ProjectID, resourceId, snapshotId).RestoreFilesRequest(req).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.SnapshotsAPI.GetSnapshot(ctx, snapshotId, c.ProjectID).Execute()
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
			case externalEonSdkAPI.JOB_FAILED, externalEonSdkAPI.JOB_CANCELLED:
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
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.BackupPoliciesAPI.ListBackupPolicies(ctx, c.ProjectID).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.BackupPoliciesAPI.GetBackupPolicy(ctx, policyId, c.ProjectID).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.BackupPoliciesAPI.CreateBackupPolicy(ctx, c.ProjectID).CreateBackupPolicyRequest(req).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	resp, httpResp, err := c.client.BackupPoliciesAPI.UpdateBackupPolicy(ctx, policyId, c.ProjectID).UpdateBackupPolicyRequest(req).Execute()
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
	if err := c.ensureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	httpResp, err := c.client.BackupPoliciesAPI.DeleteBackupPolicy(ctx, policyId, c.ProjectID).Execute()
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

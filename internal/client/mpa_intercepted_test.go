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

// mpaInterceptedBody is the body the API gateway returns with HTTP 201 when an
// action approval policy covers the request.
const mpaInterceptedBody = `{
	"actionApprovalRequest": {
		"id": "mpa-1",
		"operation": "MPA_OPERATION_CREATE_BACKUP_POLICY",
		"requesterEmail": "user@example.com",
		"status": "CREATED",
		"policyDetails": [],
		"createdTime": "2026-01-01T00:00:00Z"
	},
	"mpaPolicies": {}
}`

const mpaRequiredMessage = `operation requires Multi-Party Approval (MPA request ID "mpa-1"); ` +
	`after the request is approved, run the same operation again with the provider's action_approval_request_id set to "mpa-1"`

func mpaInterceptedTransport(t *testing.T, wantMethod, wantPath string) http.RoundTripper {
	return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, wantMethod, r.Method)
		assert.Equal(t, wantPath, r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(mpaInterceptedBody)),
		}, nil
	})
}

func TestMPAInterceptedOperationsReturnError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		path       string
		wantAction string
		call       func(ctx context.Context, c *EonClient) (any, error)
	}{
		{
			name:       "connect restore account",
			method:     http.MethodPost,
			path:       "/api/v1/projects/project-1/restore-accounts",
			wantAction: "failed to connect restore account",
			call: func(ctx context.Context, c *EonClient) (any, error) {
				return c.ConnectRestoreAccount(ctx, externalEonSdkAPI.ConnectRestoreAccountRequest{})
			},
		},
		{
			name:       "reconnect restore account",
			method:     http.MethodPost,
			path:       "/api/v1/projects/project-1/restore-accounts/acct-1/reconnect",
			wantAction: "failed to reconnect restore account",
			call: func(ctx context.Context, c *EonClient) (any, error) {
				return c.ReconnectRestoreAccount(ctx, "acct-1")
			},
		},
		{
			name:       "connect restore AWS organizational unit",
			method:     http.MethodPost,
			path:       "/api/v1/projects/project-1/restore-aws-organizational-units",
			wantAction: "failed to connect restore AWS organizational unit",
			call: func(ctx context.Context, c *EonClient) (any, error) {
				return c.ConnectRestoreAwsOrganizationalUnit(ctx, externalEonSdkAPI.ConnectRestoreAwsOrganizationalUnitRequest{})
			},
		},
		{
			name:       "create backup policy",
			method:     http.MethodPost,
			path:       "/api/v1/projects/project-1/backup-policies",
			wantAction: "failed to create backup policy",
			call: func(ctx context.Context, c *EonClient) (any, error) {
				return c.CreateBackupPolicy(ctx, externalEonSdkAPI.CreateBackupPolicyRequest{})
			},
		},
		{
			name:       "update backup policy",
			method:     http.MethodPut,
			path:       "/api/v1/projects/project-1/backup-policies/policy-1",
			wantAction: "failed to update backup policy",
			call: func(ctx context.Context, c *EonClient) (any, error) {
				return c.UpdateBackupPolicy(ctx, "policy-1", externalEonSdkAPI.UpdateBackupPolicyRequest{})
			},
		},
		{
			name:       "delete backup policy",
			method:     http.MethodDelete,
			path:       "/api/v1/projects/project-1/backup-policies/policy-1",
			wantAction: "failed to delete backup policy",
			call: func(ctx context.Context, c *EonClient) (any, error) {
				return nil, c.DeleteBackupPolicy(ctx, "policy-1")
			},
		},
		{
			name:       "remove snapshot hold",
			method:     http.MethodPatch,
			path:       "/api/v1/projects/project-1/snapshots/snap-1/remove-hold",
			wantAction: "failed to remove snapshot hold",
			call: func(ctx context.Context, c *EonClient) (any, error) {
				return nil, c.RemoveSnapshotHold(ctx, "snap-1")
			},
		},
		{
			name:       "start BigQuery dataset restore",
			method:     http.MethodPost,
			path:       "/api/v1/projects/project-1/resources/res-1/snapshots/snap-1/restore-bigquery-dataset",
			wantAction: "failed to start BigQuery dataset restore",
			call: func(ctx context.Context, c *EonClient) (any, error) {
				return c.StartBigQueryDatasetRestore(ctx, "res-1", "snap-1", BigQueryRestoreRequest{})
			},
		},
		{
			name:       "start DynamoDB table restore",
			method:     http.MethodPost,
			path:       "/api/v1/projects/project-1/resources/res-1/snapshots/snap-1/restore-dynamo-db-table",
			wantAction: "failed to start DynamoDB table restore",
			call: func(ctx context.Context, c *EonClient) (any, error) {
				return c.StartDynamoDBTableRestore(ctx, "res-1", "snap-1", externalEonSdkAPI.RestoreDynamoDBTableRequest{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(mpaInterceptedTransport(t, tt.method, tt.path))

			result, err := tt.call(context.Background(), c)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantAction)
			assert.Contains(t, err.Error(), mpaRequiredMessage)
			if jobID, ok := result.(string); ok {
				assert.Empty(t, jobID)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestDeleteBackupPolicySuccessStatuses(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: status,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(``)),
				}, nil
			}))

			assert.NoError(t, c.DeleteBackupPolicy(context.Background(), "policy-1"))
		})
	}
}

func TestMPARequestIDFromBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "full body", body: mpaInterceptedBody, want: "mpa-1"},
		{name: "partial request", body: `{"actionApprovalRequest":{"id":"mpa-2"}}`, want: "mpa-2"},
		{name: "no request", body: `{}`, want: ""},
		{name: "null request", body: `{"actionApprovalRequest":null}`, want: ""},
		{name: "invalid JSON", body: `not json`, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, mpaRequestIDFromBody(strings.NewReader(tt.body)))
		})
	}
}

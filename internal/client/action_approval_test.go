package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// actionApprovalOperations calls every client method whose API operation an
// action approval policy can protect.
var actionApprovalOperations = map[string]func(ctx context.Context, c *EonClient) error{
	"connect restore account": func(ctx context.Context, c *EonClient) error {
		_, err := c.ConnectRestoreAccount(ctx, externalEonSdkAPI.ConnectRestoreAccountRequest{})
		return err
	},
	"connect restore AWS organizational unit": func(ctx context.Context, c *EonClient) error {
		_, err := c.ConnectRestoreAwsOrganizationalUnit(ctx, externalEonSdkAPI.ConnectRestoreAwsOrganizationalUnitRequest{})
		return err
	},
	"reconnect restore account": func(ctx context.Context, c *EonClient) error {
		_, err := c.ReconnectRestoreAccount(ctx, "acct-1")
		return err
	},
	"remove snapshot hold": func(ctx context.Context, c *EonClient) error {
		return c.RemoveSnapshotHold(ctx, "snap-1")
	},
	"create backup policy": func(ctx context.Context, c *EonClient) error {
		_, err := c.CreateBackupPolicy(ctx, externalEonSdkAPI.CreateBackupPolicyRequest{})
		return err
	},
	"update backup policy": func(ctx context.Context, c *EonClient) error {
		_, err := c.UpdateBackupPolicy(ctx, "policy-1", externalEonSdkAPI.UpdateBackupPolicyRequest{})
		return err
	},
	"delete backup policy": func(ctx context.Context, c *EonClient) error {
		return c.DeleteBackupPolicy(ctx, "policy-1")
	},
	"EBS volume restore": func(ctx context.Context, c *EonClient) error {
		_, err := c.StartVolumeRestore(ctx, "res-1", "snap-1", externalEonSdkAPI.RestoreVolumeToEbsRequest{})
		return err
	},
	"EBS snapshot restore": func(ctx context.Context, c *EonClient) error {
		_, err := c.StartEbsSnapshotRestore(ctx, "res-1", "snap-1", externalEonSdkAPI.RestoreVolumeToEbsSnapshotRequest{})
		return err
	},
	"RDS restore": func(ctx context.Context, c *EonClient) error {
		_, err := c.StartRdsRestore(ctx, "res-1", "snap-1", externalEonSdkAPI.RestoreDbToRdsInstanceRequest{})
		return err
	},
	"EC2 instance restore": func(ctx context.Context, c *EonClient) error {
		_, err := c.StartEc2InstanceRestore(ctx, "res-1", "snap-1", externalEonSdkAPI.RestoreAwsEc2InstanceRequest{})
		return err
	},
	"S3 bucket restore": func(ctx context.Context, c *EonClient) error {
		_, err := c.StartS3BucketRestore(ctx, "res-1", "snap-1", externalEonSdkAPI.RestoreBucketRequest{})
		return err
	},
	"DynamoDB table restore": func(ctx context.Context, c *EonClient) error {
		_, err := c.StartDynamoDBTableRestore(ctx, "res-1", "snap-1", externalEonSdkAPI.RestoreDynamoDBTableRequest{})
		return err
	},
	"GCP VM instance restore": func(ctx context.Context, c *EonClient) error {
		_, err := c.StartGcpVmInstanceRestore(ctx, "res-1", "snap-1", externalEonSdkAPI.RestoreGcpVmInstanceRequest{})
		return err
	},
	"GCP disk restore": func(ctx context.Context, c *EonClient) error {
		_, err := c.StartGcpDiskRestore(ctx, "res-1", "snap-1", externalEonSdkAPI.RestoreGcpDiskRequest{})
		return err
	},
	"GCP Cloud SQL restore": func(ctx context.Context, c *EonClient) error {
		_, err := c.StartGcpCloudSqlRestore(ctx, "res-1", "snap-1", externalEonSdkAPI.RestoreGcpCloudSqlRequest{})
		return err
	},
	"BigQuery dataset restore": func(ctx context.Context, c *EonClient) error {
		_, err := c.StartBigQueryDatasetRestore(ctx, "res-1", "snap-1", BigQueryRestoreRequest{})
		return err
	},
	"Azure disk restore": func(ctx context.Context, c *EonClient) error {
		_, err := c.StartAzureDiskRestore(ctx, "res-1", "snap-1", externalEonSdkAPI.RestoreAzureDiskRequest{})
		return err
	},
	"Azure SQL database restore": func(ctx context.Context, c *EonClient) error {
		_, err := c.StartAzureSqlDatabaseRestore(ctx, "res-1", "snap-1", externalEonSdkAPI.RestoreAzureSqlDatabaseRequest{})
		return err
	},
	"Azure VM instance restore": func(ctx context.Context, c *EonClient) error {
		_, err := c.StartAzureVmInstanceRestore(ctx, "res-1", "snap-1", externalEonSdkAPI.RestoreAzureVmInstanceRequest{})
		return err
	},
}

// headerCapturingTransport records the X-Action-Approval-Request-Id header
// values it receives and fails every request, since only the request matters.
type headerCapturingTransport struct {
	mu     sync.Mutex
	values [][]string
}

func (h *headerCapturingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	h.mu.Lock()
	h.values = append(h.values, r.Header.Values(actionApprovalRequestIDHeader))
	h.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{}`)),
	}, nil
}

func TestActionApprovalRequestIDHeader(t *testing.T) {
	t.Parallel()

	const approvedID = "11111111-2222-3333-4444-555555555555"

	for name, call := range actionApprovalOperations {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			t.Run("sent when configured", func(t *testing.T) {
				t.Parallel()

				transport := &headerCapturingTransport{}
				c := newTestClient(transport)
				c.actionApprovalRequestID = approvedID

				require.Error(t, call(context.Background(), c))
				require.Len(t, transport.values, 1)
				assert.Equal(t, []string{approvedID}, transport.values[0])
			})

			t.Run("omitted when not configured", func(t *testing.T) {
				t.Parallel()

				transport := &headerCapturingTransport{}
				c := newTestClient(transport)

				require.Error(t, call(context.Background(), c))
				require.Len(t, transport.values, 1)
				assert.Empty(t, transport.values[0])
			})
		})
	}
}

func TestActionApprovalRequestIDHeaderNotSentOnUnprotectedOperations(t *testing.T) {
	t.Parallel()

	transport := &headerCapturingTransport{}
	c := newTestClient(transport)
	c.actionApprovalRequestID = "11111111-2222-3333-4444-555555555555"

	_, err := c.GetBackupPolicy(context.Background(), "policy-1")
	require.Error(t, err)
	require.Len(t, transport.values, 1)
	assert.Empty(t, transport.values[0])
}

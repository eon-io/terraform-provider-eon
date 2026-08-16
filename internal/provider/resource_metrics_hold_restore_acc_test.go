package provider

import (
	"fmt"
	"testing"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRestoreAccountMetricsConfig(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	resourceName := "eon_restore_account_metrics_config.test"
	accountID := "restore-acct-1"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_restore_account_metrics_config" "test" {
  restore_account_id = %q
  aws {
    region = "us-east-1"
  }
}
`, accountID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "restore_account_id", accountID),
					resource.TestCheckResourceAttr(resourceName, "id", accountID),
					resource.TestCheckResourceAttr(resourceName, "aws.region", "us-east-1"),
				),
			},
			{
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_restore_account_metrics_config" "test" {
  restore_account_id = %q
  aws {
    region = "us-west-2"
  }
}
`, accountID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "aws.region", "us-west-2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     accountID,
			},
			{
				PreConfig: func() {
					server.DeleteMetricsConfig(accountID)
				},
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_restore_account_metrics_config" "test" {
  restore_account_id = %q
  aws {
    region = "us-west-2"
  }
}
`, accountID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccSnapshotHold(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	resourceName := "eon_snapshot_hold.test"
	snapshotID := "snap-hold-1"

	snap := externalEonSdkAPI.NewSnapshot(snapshotID, time.Now().UTC(), "res-1")
	snap.SetOnHold(false)
	server.AddSnapshot("res-1", snap)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_snapshot_hold" "test" {
  snapshot_id = %q
  description = "compliance hold"
}
`, snapshotID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "snapshot_id", snapshotID),
					resource.TestCheckResourceAttr(resourceName, "id", snapshotID),
					resource.TestCheckResourceAttr(resourceName, "description", "compliance hold"),
				),
			},
			{
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_snapshot_hold" "test" {
  snapshot_id = %q
  description = "updated hold reason"
}
`, snapshotID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "updated hold reason"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     snapshotID,
			},
			{
				PreConfig: func() {
					server.RemoveHold(snapshotID)
				},
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_snapshot_hold" "test" {
  snapshot_id = %q
  description = "updated hold reason"
}
`, snapshotID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccRestoreJobDynamoDB(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	resourceName := "eon_restore_job.test"
	resourceID := "res-dynamo-1"
	snapshotID := "snap-dynamo-1"

	res := newTestInventoryResource(resourceID)
	res.ResourceType = externalEonSdkAPI.AWS_DYNAMO_DB
	server.AddResource(res)

	snap := externalEonSdkAPI.NewSnapshot(snapshotID, time.Now().UTC(), resourceID)
	server.AddSnapshot(resourceID, snap)

	config := server.providerConfig() + fmt.Sprintf(`
resource "eon_restore_job" "test" {
  restore_type         = "full"
  snapshot_id          = %q
  restore_account_id   = "restore-acct-1"
  wait_for_completion  = false

  dynamodb_config {
    restore_region = "us-east-1"
    restored_name  = "restored-table"
  }
}
`, snapshotID)

	var jobID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "snapshot_id", snapshotID),
					resource.TestCheckResourceAttr(resourceName, "restore_type", "full"),
					resource.TestCheckResourceAttrSet(resourceName, "job_id"),
					resource.TestCheckResourceAttr(resourceName, "dynamodb_config.restore_region", "us-east-1"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("resource not found: %s", resourceName)
						}
						jobID = rs.Primary.ID
						if jobID == "" {
							return fmt.Errorf("empty job id")
						}
						return nil
					},
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"restore_type", "snapshot_id", "restore_account_id", "wait_for_completion",
					"timeout_minutes", "dynamodb_config", "resource_id", "status", "status_message",
					"created_at", "started_at", "completed_at", "duration_seconds",
				},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return jobID, nil
				},
			},
			{
				PreConfig: func() {
					server.DeleteRestoreJob(jobID)
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccSourceAccountMetricsConfig(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	resourceName := "eon_source_account_metrics_config.test"
	accountID := "source-acct-1"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_source_account_metrics_config" "test" {
  source_account_id = %q
  aws {
    region = "us-east-1"
  }
}
`, accountID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "source_account_id", accountID),
					resource.TestCheckResourceAttr(resourceName, "id", accountID),
					resource.TestCheckResourceAttr(resourceName, "aws.region", "us-east-1"),
				),
			},
			{
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_source_account_metrics_config" "test" {
  source_account_id = %q
  aws {
    region = "us-west-2"
  }
}
`, accountID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "aws.region", "us-west-2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     accountID,
			},
			{
				PreConfig: func() {
					server.DeleteSourceMetricsConfig(accountID)
				},
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_source_account_metrics_config" "test" {
  source_account_id = %q
  aws {
    region = "us-west-2"
  }
}
`, accountID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

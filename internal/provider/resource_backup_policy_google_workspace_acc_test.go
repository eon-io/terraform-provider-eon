package provider

import (
	"fmt"
	"regexp"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func googleWorkspacePolicyConfig(workspaceID, vaultID string, retentionDays int) string {
	return fmt.Sprintf(`
resource "eon_backup_policy" "gws" {
  name    = "Google Workspace nightly"
  enabled = true

  resource_selector = {
    resource_selection_mode     = "NONE"
    resource_inclusion_override = [%q]
  }

  backup_plan = {
    backup_policy_type = "GOOGLE_WORKSPACE"

    google_workspace_plan = {
      vault_id       = %q
      retention_days = %d
    }
  }
}
`, workspaceID, vaultID, retentionDays)
}

// TestAccBackupPolicy_GoogleWorkspace drives a Google Workspace policy through create, update,
// import and refresh against the fake API, and checks that edits made outside Terraform to the
// workspace selector, vault, or retention show up as drift and are reverted on the next apply.
func TestAccBackupPolicy_GoogleWorkspace(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	resourceName := "eon_backup_policy.gws"
	var policyID string

	desiredConfig := server.providerConfig() + googleWorkspacePolicyConfig("ws-1", "vault-gcp", 14)
	desiredChecks := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(resourceName, "resource_selector.resource_selection_mode", "NONE"),
		resource.TestCheckResourceAttr(resourceName, "resource_selector.resource_inclusion_override.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "resource_selector.resource_inclusion_override.0", "ws-1"),
		resource.TestCheckResourceAttr(resourceName, "backup_plan.backup_policy_type", "GOOGLE_WORKSPACE"),
		resource.TestCheckResourceAttr(resourceName, "backup_plan.google_workspace_plan.vault_id", "vault-gcp"),
		resource.TestCheckResourceAttr(resourceName, "backup_plan.google_workspace_plan.retention_days", "14"),
		resource.TestCheckNoResourceAttr(resourceName, "backup_plan.standard_plan"),
	)

	driftStep := func(mutate func(policy *externalEonSdkAPI.BackupPolicy)) resource.TestStep {
		return resource.TestStep{
			PreConfig:          func() { server.MutateBackupPolicy(policyID, mutate) },
			Config:             desiredConfig,
			PlanOnly:           true,
			ExpectNonEmptyPlan: true,
		}
	}
	reconvergeStep := resource.TestStep{Config: desiredConfig, Check: desiredChecks}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + googleWorkspacePolicyConfig("ws-1", "vault-gcp", 30),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Google Workspace nightly"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "backup_plan.backup_policy_type", "GOOGLE_WORKSPACE"),
					resource.TestCheckResourceAttr(resourceName, "backup_plan.google_workspace_plan.vault_id", "vault-gcp"),
					resource.TestCheckResourceAttr(resourceName, "backup_plan.google_workspace_plan.retention_days", "30"),
					resource.TestCheckResourceAttr(resourceName, "resource_selector.resource_inclusion_override.0", "ws-1"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("resource not found: %s", resourceName)
						}
						policyID = rs.Primary.ID
						if policyID == "" {
							return fmt.Errorf("empty backup policy id")
						}
						return nil
					},
				),
			},
			{
				Config: desiredConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: desiredChecks,
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at", "updated_at"},
			},
			{
				Config:   desiredConfig,
				PlanOnly: true,
			},
			driftStep(func(policy *externalEonSdkAPI.BackupPolicy) {
				policy.BackupPlan.GoogleWorkspacePlan.Get().SetBackupRetentionDays(7)
			}),
			reconvergeStep,
			driftStep(func(policy *externalEonSdkAPI.BackupPolicy) {
				policy.BackupPlan.GoogleWorkspacePlan.Get().SetVaultId("vault-other")
			}),
			reconvergeStep,
			driftStep(func(policy *externalEonSdkAPI.BackupPolicy) {
				policy.ResourceSelector.SetResourceInclusionOverride([]string{"ws-other"})
			}),
			reconvergeStep,
		},
	})
}

func TestAccBackupPolicy_GoogleWorkspaceRejectsZeroRetention(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      server.providerConfig() + googleWorkspacePolicyConfig("ws-1", "vault-gcp", 0),
				ExpectError: regexp.MustCompile("retention_days must be at least 1"),
			},
		},
	})
}

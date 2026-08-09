package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

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

func TestAccActionApprovalRule(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	resourceName := "eon_action_approval_rule.test"
	var ruleID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
resource "eon_action_approval_rule" "test" {
  operation              = "RESTORE_RESOURCE"
  required_approvals     = 1
  approval_window_hours  = 24
  execution_window_hours = 48
  description            = "require restore approval"

  resource_selector = {
    resource_selection_mode = "ALL"
  }
}

data "eon_action_approval_rules" "all" {
  depends_on = [eon_action_approval_rule.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "operation", "RESTORE_RESOURCE"),
					resource.TestCheckResourceAttr(resourceName, "required_approvals", "1"),
					resource.TestCheckResourceAttr(resourceName, "approval_window_hours", "24"),
					resource.TestCheckResourceAttr(resourceName, "execution_window_hours", "48"),
					resource.TestCheckResourceAttr(resourceName, "description", "require restore approval"),
					resource.TestCheckResourceAttr(resourceName, "resource_selector.resource_selection_mode", "ALL"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet("data.eon_action_approval_rules.all", "rules.#"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("resource not found: %s", resourceName)
						}
						ruleID = rs.Primary.ID
						if ruleID == "" {
							return fmt.Errorf("empty rule id")
						}
						return nil
					},
				),
			},
			{
				Config: server.providerConfig() + `
resource "eon_action_approval_rule" "test" {
  operation              = "RESTORE_RESOURCE"
  required_approvals     = 2
  approval_window_hours  = 36
  execution_window_hours = 72
  description            = "updated restore approval"

  resource_selector = {
    resource_selection_mode = "ALL"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "required_approvals", "2"),
					resource.TestCheckResourceAttr(resourceName, "approval_window_hours", "36"),
					resource.TestCheckResourceAttr(resourceName, "execution_window_hours", "72"),
					resource.TestCheckResourceAttr(resourceName, "description", "updated restore approval"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("resource not found: %s", resourceName)
						}
						ruleID = rs.Primary.ID
						return nil
					},
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				PreConfig: func() {
					server.DeleteActionApprovalRule(ruleID)
				},
				Config: server.providerConfig() + `
resource "eon_action_approval_rule" "test" {
  operation              = "RESTORE_RESOURCE"
  required_approvals     = 2
  approval_window_hours  = 36
  execution_window_hours = 72
  description            = "updated restore approval"

  resource_selector = {
    resource_selection_mode = "ALL"
  }
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

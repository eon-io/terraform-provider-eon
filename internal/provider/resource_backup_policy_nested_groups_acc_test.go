package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccBackupPolicyNestedGroups creates a CONDITIONAL policy whose selector nests
// AND → OR → AND (type-scoped name excludes), matching the console resource selector UI.
// Requires live Eon credentials plus EON_TEST_VAULT_ID.
func TestAccBackupPolicyNestedGroups(t *testing.T) {
	testAccRealEnvPreCheck(t, false)
	vaultID := os.Getenv("EON_TEST_VAULT_ID")
	if vaultID == "" {
		t.Skip("EON_TEST_VAULT_ID must be set for backup policy acceptance tests")
	}

	resourceName := "eon_backup_policy.test"
	name := fmt.Sprintf("tf-acc-nested-groups-%s", acctest.RandString(6))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRealProviderConfig() + testAccBackupPolicyNestedGroupsConfig(name, vaultID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "resource_selector.resource_selection_mode", "CONDITIONAL"),
					resource.TestCheckResourceAttr(resourceName, "resource_selector.expression.group.operator", "AND"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					// Nested OR group is the last top-level operand.
					resource.TestCheckResourceAttr(resourceName, "resource_selector.expression.group.operands.2.group.operator", "OR"),
					resource.TestCheckResourceAttr(resourceName, "resource_selector.expression.group.operands.2.group.operands.0.group.operator", "AND"),
					resource.TestCheckResourceAttr(resourceName, "resource_selector.expression.group.operands.2.group.operands.0.group.operands.0.resource_type.resource_types.0", "AWS_RDS"),
					resource.TestCheckResourceAttr(resourceName, "resource_selector.expression.group.operands.2.group.operands.0.group.operands.1.resource_name.operator", "NOT_CONTAINS"),
					resource.TestCheckResourceAttr(resourceName, "resource_selector.expression.group.operands.2.group.operands.0.group.operands.1.resource_name.resource_names.0", "example-rds-instance"),
					resource.TestCheckResourceAttr(resourceName, "resource_selector.expression.group.operands.2.group.operands.1.group.operands.0.resource_type.resource_types.0", "AWS_DYNAMO_DB"),
					resource.TestCheckResourceAttr(resourceName, "resource_selector.expression.group.operands.2.group.operands.1.group.operands.1.resource_name.resource_names.0", "example-dynamodb-table"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created_at", "updated_at", "backup_plan.standard_plan.schedule_timezone"},
			},
		},
	})
}

func testAccBackupPolicyNestedGroupsConfig(name, vaultID string) string {
	return fmt.Sprintf(`
resource "eon_backup_policy" "test" {
  name    = %q
  enabled = true

  resource_selector = {
    resource_selection_mode = "CONDITIONAL"

    expression = {
      group = {
        operator = "AND"
        operands = [
          {
            source_region = {
              operator       = "IN"
              source_regions = ["us-east-1"]
            }
          },
          {
            resource_type = {
              operator       = "IN"
              resource_types = ["AWS_RDS", "AWS_DYNAMO_DB"]
            }
          },
          {
            group = {
              operator = "OR"
              operands = [
                {
                  group = {
                    operator = "AND"
                    operands = [
                      {
                        resource_type = {
                          operator       = "IN"
                          resource_types = ["AWS_RDS"]
                        }
                      },
                      {
                        resource_name = {
                          operator       = "NOT_CONTAINS"
                          resource_names = ["example-rds-instance"]
                        }
                      }
                    ]
                  }
                },
                {
                  group = {
                    operator = "AND"
                    operands = [
                      {
                        resource_type = {
                          operator       = "IN"
                          resource_types = ["AWS_DYNAMO_DB"]
                        }
                      },
                      {
                        resource_name = {
                          operator       = "NOT_CONTAINS"
                          resource_names = ["example-dynamodb-table"]
                        }
                      }
                    ]
                  }
                }
              ]
            }
          }
        ]
      }
    }
  }

  backup_plan = {
    backup_policy_type = "STANDARD"
    standard_plan = {
      schedule_timezone = "UTC"
      backup_schedules = [
        {
          vault_id       = %q
          retention_days = 7
          schedule_config = {
            frequency = "DAILY"
            daily_config = {
              time_of_day_hour     = 3
              time_of_day_minutes  = 0
              start_window_minutes = 240
            }
          }
        }
      ]
    }
  }
}
`, name, vaultID)
}

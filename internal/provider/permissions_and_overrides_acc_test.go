package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPermissions(t *testing.T) {
	testAccRealEnvPreCheck(t, false)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRealProviderConfig() + `
data "eon_permissions" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.eon_permissions.all", "permissions.#"),
					resource.TestMatchResourceAttr("data.eon_permissions.all", "permissions.0.permission_type", regexp.MustCompile(`\.`)),
					resource.TestCheckResourceAttrSet("data.eon_permissions.all", "permissions.0.description"),
					resource.TestCheckResourceAttrSet("data.eon_permissions.all", "permissions.0.allow_conditions"),
				),
			},
		},
	})
}

func TestAccResourceBackupExclusion(t *testing.T) {
	testAccRealEnvPreCheck(t, true)

	resourceID := os.Getenv("EON_TEST_RESOURCE_ID")
	resourceName := "eon_resource_backup_exclusion.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRealProviderConfig() + fmt.Sprintf(`
resource "eon_resource_backup_exclusion" "test" {
  resource_id = %q
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "resource_id", resourceID),
					resource.TestCheckResourceAttr(resourceName, "id", resourceID),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     resourceID,
			},
		},
	})
}

func TestAccResourceDataClassesOverride(t *testing.T) {
	testAccRealEnvPreCheck(t, true)

	resourceID := os.Getenv("EON_TEST_RESOURCE_ID")
	resourceName := "eon_resource_data_classes_override.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRealProviderConfig() + fmt.Sprintf(`
resource "eon_resource_data_classes_override" "test" {
  resource_id  = %q
  data_classes = ["PII", "PHI"]
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "resource_id", resourceID),
					resource.TestCheckResourceAttr(resourceName, "data_classes.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "data_classes.*", "PII"),
					resource.TestCheckTypeSetElemAttr(resourceName, "data_classes.*", "PHI"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("resource not found: %s", resourceName)
						}
						if rs.Primary.Attributes["resource_id"] == "" {
							return fmt.Errorf("empty resource_id")
						}
						return nil
					},
				),
			},
			{
				Config: testAccRealProviderConfig() + fmt.Sprintf(`
resource "eon_resource_data_classes_override" "test" {
  resource_id  = %q
  data_classes = ["PII", "FI"]
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "data_classes.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "data_classes.*", "PII"),
					resource.TestCheckTypeSetElemAttr(resourceName, "data_classes.*", "FI"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     resourceID,
			},
		},
	})
}

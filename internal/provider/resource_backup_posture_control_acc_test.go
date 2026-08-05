package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBackupPostureControl(t *testing.T) {
	testAccRealEnvPreCheck(t, false)

	resourceName := "eon_backup_posture_control.test"
	name := fmt.Sprintf("tf-acc-cross-region-%s", acctest.RandString(6))
	updatedName := name + "-updated"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRealProviderConfig() + fmt.Sprintf(`
resource "eon_backup_posture_control" "test" {
  name     = %q
  severity = "HIGH"
  resource_selector = {
    resource_selection_mode = "ALL"
  }
  rules = {
    cross_region = true
  }
}

data "eon_backup_posture_controls" "all" {
  depends_on = [eon_backup_posture_control.test]
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "severity", "HIGH"),
					resource.TestCheckResourceAttr(resourceName, "resource_selector.resource_selection_mode", "ALL"),
					resource.TestCheckResourceAttr(resourceName, "rules.cross_region", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet("data.eon_backup_posture_controls.all", "controls.#"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("resource not found: %s", resourceName)
						}
						if rs.Primary.ID == "" {
							return fmt.Errorf("empty control id")
						}
						return nil
					},
				),
			},
			{
				Config: testAccRealProviderConfig() + fmt.Sprintf(`
resource "eon_backup_posture_control" "test" {
  name     = %q
  severity = "MEDIUM"
  resource_selector = {
    resource_selection_mode = "ALL"
  }
  rules = {
    cross_region  = true
    cross_account = true
  }
}
`, updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "severity", "MEDIUM"),
					resource.TestCheckResourceAttr(resourceName, "rules.cross_region", "true"),
					resource.TestCheckResourceAttr(resourceName, "rules.cross_account", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccIdps(t *testing.T) {
	testAccRealEnvPreCheck(t, false)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRealProviderConfig() + `
data "eon_idps" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.eon_idps.all", "idps.#"),
					resource.TestMatchResourceAttr("data.eon_idps.all", "idps.0.id", regexp.MustCompile(`.+`)),
					resource.TestCheckResourceAttrSet("data.eon_idps.all", "idps.0.provider_name"),
				),
			},
		},
	})
}

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBackupPostureControl(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	t.Setenv("EON_USE_EXACT_ENDPOINT", "true")

	resourceName := "eon_backup_posture_control.test"
	var controlID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
resource "eon_backup_posture_control" "test" {
  name     = "acc-cross-region"
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
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "acc-cross-region"),
					resource.TestCheckResourceAttr(resourceName, "severity", "HIGH"),
					resource.TestCheckResourceAttr(resourceName, "resource_selector.resource_selection_mode", "ALL"),
					resource.TestCheckResourceAttr(resourceName, "rules.cross_region", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr("data.eon_backup_posture_controls.all", "controls.#", "1"),
					resource.TestCheckResourceAttr("data.eon_backup_posture_controls.all", "controls.0.name", "acc-cross-region"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("resource not found: %s", resourceName)
						}
						controlID = rs.Primary.ID
						if controlID == "" {
							return fmt.Errorf("empty control id")
						}
						return nil
					},
				),
			},
			{
				Config: server.providerConfig() + `
resource "eon_backup_posture_control" "test" {
  name     = "acc-cross-region-updated"
  severity = "MEDIUM"
  resource_selector = {
    resource_selection_mode = "ALL"
  }
  rules = {
    cross_region  = true
    cross_account = true
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "acc-cross-region-updated"),
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
			{
				PreConfig: func() {
					server.DeleteControl(controlID)
				},
				Config: server.providerConfig() + `
resource "eon_backup_posture_control" "test" {
  name     = "acc-cross-region-updated"
  severity = "MEDIUM"
  resource_selector = {
    resource_selection_mode = "ALL"
  }
  rules = {
    cross_region  = true
    cross_account = true
  }
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccIdps(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	t.Setenv("EON_USE_EXACT_ENDPOINT", "true")
	server.AddIdp(newTestIdp("idp-1", "Okta"))
	server.AddIdp(newTestIdp("idp-2", "Azure AD"))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_idps" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_idps.all", "idps.#", "2"),
					resource.TestMatchResourceAttr("data.eon_idps.all", "idps.0.id", regexp.MustCompile(`^idp-`)),
					resource.TestCheckResourceAttrSet("data.eon_idps.all", "idps.0.provider_name"),
				),
			},
		},
	})
}

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccFindingExclusion(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	resourceName := "eon_finding_exclusion.test"
	var exclusionID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
resource "eon_finding_exclusion" "test" {
  scope                = "RESOURCE"
  provider_resource_id = "i-0abc123def4567890"
  value                = "/var/cache"
  type                 = "PATH"
  detector             = "MALWARE"
}

data "eon_finding_exclusions" "all" {
  depends_on = [eon_finding_exclusion.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "scope", "RESOURCE"),
					resource.TestCheckResourceAttr(resourceName, "provider_resource_id", "i-0abc123def4567890"),
					resource.TestCheckResourceAttr(resourceName, "value", "/var/cache"),
					resource.TestCheckResourceAttr(resourceName, "type", "PATH"),
					resource.TestCheckResourceAttr(resourceName, "detector", "MALWARE"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.all", "finding_exclusions.#", "1"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.all", "finding_exclusions.0.scope", "RESOURCE"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.all", "finding_exclusions.0.provider_resource_id", "i-0abc123def4567890"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.all", "finding_exclusions.0.value", "/var/cache"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.all", "finding_exclusions.0.type", "PATH"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.all", "finding_exclusions.0.detector", "MALWARE"),
					resource.TestCheckResourceAttrSet("data.eon_finding_exclusions.all", "finding_exclusions.0.id"),
					resource.TestCheckResourceAttrSet("data.eon_finding_exclusions.all", "finding_exclusions.0.updated_at"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("resource not found: %s", resourceName)
						}
						exclusionID = rs.Primary.ID
						if exclusionID == "" {
							return fmt.Errorf("empty finding exclusion id")
						}
						return nil
					},
				),
			},
			{
				Config: server.providerConfig() + `
resource "eon_finding_exclusion" "test" {
  scope    = "ACCOUNT"
  value    = "orders"
  type     = "TABLE"
  detector = "DATA_ANOMALY"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "scope", "ACCOUNT"),
					resource.TestCheckNoResourceAttr(resourceName, "provider_resource_id"),
					resource.TestCheckResourceAttr(resourceName, "value", "orders"),
					resource.TestCheckResourceAttr(resourceName, "type", "TABLE"),
					resource.TestCheckResourceAttr(resourceName, "detector", "DATA_ANOMALY"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("resource not found: %s", resourceName)
						}
						exclusionID = rs.Primary.ID
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
					server.DeleteFindingExclusion(exclusionID)
				},
				Config: server.providerConfig() + `
resource "eon_finding_exclusion" "test" {
  scope    = "ACCOUNT"
  value    = "orders"
  type     = "TABLE"
  detector = "DATA_ANOMALY"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

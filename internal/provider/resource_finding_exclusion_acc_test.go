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
	dataSourceName := "data.eon_finding_exclusions.all"
	var exclusionID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
resource "eon_finding_exclusion" "test" {
  value    = "/var/cache"
  type     = "PATH"
  detector = "MALWARE"
}

data "eon_finding_exclusions" "all" {
  depends_on = [eon_finding_exclusion.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "value", "/var/cache"),
					resource.TestCheckResourceAttr(resourceName, "type", "PATH"),
					resource.TestCheckResourceAttr(resourceName, "detector", "MALWARE"),
					resource.TestCheckNoResourceAttr(resourceName, "resource_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
					resource.TestCheckResourceAttr(dataSourceName, "exclusions.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "exclusions.0.value", "/var/cache"),
					resource.TestCheckResourceAttr(dataSourceName, "exclusions.0.type", "PATH"),
					resource.TestCheckResourceAttr(dataSourceName, "exclusions.0.detector", "MALWARE"),
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
  resource_id = "11111111-1111-1111-1111-111111111111"
  value       = "orders"
  type        = "TABLE"
  detector    = "DATA_ANOMALY"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "resource_id", "11111111-1111-1111-1111-111111111111"),
					resource.TestCheckResourceAttr(resourceName, "value", "orders"),
					resource.TestCheckResourceAttr(resourceName, "type", "TABLE"),
					resource.TestCheckResourceAttr(resourceName, "detector", "DATA_ANOMALY"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("resource not found: %s", resourceName)
						}
						if rs.Primary.ID != exclusionID {
							return fmt.Errorf("finding exclusion id changed from %s to %s", exclusionID, rs.Primary.ID)
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
  resource_id = "11111111-1111-1111-1111-111111111111"
  value       = "orders"
  type        = "TABLE"
  detector    = "DATA_ANOMALY"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

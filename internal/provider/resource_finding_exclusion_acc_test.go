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
  scope                = "RESOURCE"
  provider_resource_id = "i-0123456789abcdef0"
  type                 = "PATH"
  value                = "/var/cache/app"
  detector             = "MALWARE"
}

data "eon_finding_exclusions" "all" {
  depends_on = [eon_finding_exclusion.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "scope", "RESOURCE"),
					resource.TestCheckResourceAttr(resourceName, "provider_resource_id", "i-0123456789abcdef0"),
					resource.TestCheckResourceAttr(resourceName, "type", "PATH"),
					resource.TestCheckResourceAttr(resourceName, "value", "/var/cache/app"),
					resource.TestCheckResourceAttr(resourceName, "detector", "MALWARE"),
					resource.TestCheckResourceAttr(resourceName, "updated_at", "2024-01-02T03:04:05Z"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "exclusions.#", "1"),
					resource.TestCheckResourceAttrPair(dataSourceName, "exclusions.0.id", resourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "exclusions.0.scope", "RESOURCE"),
					resource.TestCheckResourceAttr(dataSourceName, "exclusions.0.provider_resource_id", "i-0123456789abcdef0"),
					resource.TestCheckResourceAttr(dataSourceName, "exclusions.0.type", "PATH"),
					resource.TestCheckResourceAttr(dataSourceName, "exclusions.0.value", "/var/cache/app"),
					resource.TestCheckResourceAttr(dataSourceName, "exclusions.0.detector", "MALWARE"),
					resource.TestCheckResourceAttr(dataSourceName, "exclusions.0.updated_at", "2024-01-02T03:04:05Z"),
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
  type     = "TABLE"
  value    = "staging_events"
  detector = "DATA_ANOMALY"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "scope", "ACCOUNT"),
					resource.TestCheckNoResourceAttr(resourceName, "provider_resource_id"),
					resource.TestCheckResourceAttr(resourceName, "type", "TABLE"),
					resource.TestCheckResourceAttr(resourceName, "value", "staging_events"),
					resource.TestCheckResourceAttr(resourceName, "detector", "DATA_ANOMALY"),
					resource.TestCheckResourceAttr(resourceName, "updated_at", "2024-01-02T04:04:05Z"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("resource not found: %s", resourceName)
						}
						if rs.Primary.ID != exclusionID {
							return fmt.Errorf("expected in-place update to keep id %q, got %q", exclusionID, rs.Primary.ID)
						}
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
  type     = "TABLE"
  value    = "staging_events"
  detector = "DATA_ANOMALY"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

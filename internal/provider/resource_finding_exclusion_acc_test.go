package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccFindingExclusion(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	resourceName := "eon_finding_exclusion.test"
	accountWideName := "eon_finding_exclusion.account_wide"
	var exclusionID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
resource "eon_finding_exclusion" "test" {
  resource_id = "res-1"
  value       = "/data/tmp"
  type        = "PATH"
  detector    = "MALWARE"
}

resource "eon_finding_exclusion" "account_wide" {
  value    = "staging_events"
  type     = "TABLE"
  detector = "DATA_ANOMALY"
}

data "eon_finding_exclusions" "all" {
  depends_on = [eon_finding_exclusion.test, eon_finding_exclusion.account_wide]
}

data "eon_finding_exclusions" "malware" {
  detectors  = ["MALWARE"]
  depends_on = [eon_finding_exclusion.test, eon_finding_exclusion.account_wide]
}

data "eon_finding_exclusions" "account_wide" {
  account_wide = true
  depends_on   = [eon_finding_exclusion.test, eon_finding_exclusion.account_wide]
}

data "eon_finding_exclusions" "by_resource" {
  resource_ids = ["res-1"]
  types        = ["PATH"]
  depends_on   = [eon_finding_exclusion.test, eon_finding_exclusion.account_wide]
}

data "eon_finding_exclusions" "by_value" {
  value_contains = ["staging"]
  depends_on     = [eon_finding_exclusion.test, eon_finding_exclusion.account_wide]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "resource_id", "res-1"),
					resource.TestCheckResourceAttr(resourceName, "value", "/data/tmp"),
					resource.TestCheckResourceAttr(resourceName, "type", "PATH"),
					resource.TestCheckResourceAttr(resourceName, "detector", "MALWARE"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
					resource.TestCheckNoResourceAttr(accountWideName, "resource_id"),
					resource.TestCheckResourceAttr(accountWideName, "value", "staging_events"),
					resource.TestCheckResourceAttr(accountWideName, "type", "TABLE"),
					resource.TestCheckResourceAttr(accountWideName, "detector", "DATA_ANOMALY"),
					resource.TestCheckResourceAttrSet(accountWideName, "id"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.all", "finding_exclusions.#", "2"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.malware", "finding_exclusions.#", "1"),
					resource.TestCheckResourceAttrPair("data.eon_finding_exclusions.malware", "finding_exclusions.0.id", resourceName, "id"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.malware", "finding_exclusions.0.resource_id", "res-1"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.malware", "finding_exclusions.0.value", "/data/tmp"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.malware", "finding_exclusions.0.type", "PATH"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.malware", "finding_exclusions.0.detector", "MALWARE"),
					resource.TestCheckResourceAttrPair("data.eon_finding_exclusions.malware", "finding_exclusions.0.updated_at", resourceName, "updated_at"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.account_wide", "finding_exclusions.#", "1"),
					resource.TestCheckResourceAttrPair("data.eon_finding_exclusions.account_wide", "finding_exclusions.0.id", accountWideName, "id"),
					resource.TestCheckNoResourceAttr("data.eon_finding_exclusions.account_wide", "finding_exclusions.0.resource_id"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.by_resource", "finding_exclusions.#", "1"),
					resource.TestCheckResourceAttrPair("data.eon_finding_exclusions.by_resource", "finding_exclusions.0.id", resourceName, "id"),
					resource.TestCheckResourceAttr("data.eon_finding_exclusions.by_value", "finding_exclusions.#", "1"),
					resource.TestCheckResourceAttrPair("data.eon_finding_exclusions.by_value", "finding_exclusions.0.id", accountWideName, "id"),
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
				// In-place update: the value changes and the exclusion widens to account-wide.
				Config: server.providerConfig() + `
resource "eon_finding_exclusion" "test" {
  value    = "/data/cache"
  type     = "PATH"
  detector = "RANSOMWARE_BEHAVIOR"
}

resource "eon_finding_exclusion" "account_wide" {
  value    = "staging_events"
  type     = "TABLE"
  detector = "DATA_ANOMALY"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "resource_id"),
					resource.TestCheckResourceAttr(resourceName, "value", "/data/cache"),
					resource.TestCheckResourceAttr(resourceName, "type", "PATH"),
					resource.TestCheckResourceAttr(resourceName, "detector", "RANSOMWARE_BEHAVIOR"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
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
				ResourceName:      accountWideName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				PreConfig: func() {
					server.DeleteFindingExclusion(exclusionID)
				},
				Config: server.providerConfig() + `
resource "eon_finding_exclusion" "test" {
  value    = "/data/cache"
  type     = "PATH"
  detector = "RANSOMWARE_BEHAVIOR"
}

resource "eon_finding_exclusion" "account_wide" {
  value    = "staging_events"
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

func TestAccFindingExclusionRejectsMalwareTable(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
resource "eon_finding_exclusion" "invalid" {
  value    = "customers"
  type     = "TABLE"
  detector = "MALWARE"
}
`,
				ExpectError: regexp.MustCompile(`MALWARE exclusions must use the PATH type`),
			},
			{
				Config: server.providerConfig() + `
resource "eon_finding_exclusion" "invalid" {
  value    = "customers"
  type     = "FOLDER"
  detector = "MALWARE"
}
`,
				ExpectError: regexp.MustCompile(`invalid value 'FOLDER' for FindingObjectType`),
			},
		},
	})
}

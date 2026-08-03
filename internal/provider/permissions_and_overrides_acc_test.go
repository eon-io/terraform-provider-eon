package provider

import (
	"fmt"
	"regexp"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPermissions(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	t.Setenv("EON_USE_EXACT_ENDPOINT", "true")
	server.AddPermission(externalEonSdkAPI.NewPermission(
		externalEonSdkAPI.INVENTORY_VIEW,
		"View inventory resources",
		true,
	))
	server.AddPermission(externalEonSdkAPI.NewPermission(
		externalEonSdkAPI.JOBS_VIEW,
		"View jobs",
		false,
	))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_permissions" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_permissions.all", "permissions.#", "2"),
					resource.TestMatchResourceAttr("data.eon_permissions.all", "permissions.0.permission_type", regexp.MustCompile(`\.`)),
					resource.TestCheckResourceAttrSet("data.eon_permissions.all", "permissions.0.description"),
					resource.TestCheckResourceAttrSet("data.eon_permissions.all", "permissions.0.allow_conditions"),
				),
			},
		},
	})
}

func TestAccResourceBackupExclusion(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	t.Setenv("EON_USE_EXACT_ENDPOINT", "true")

	resourceID := "res-exclusion-1"
	server.AddResource(newTestInventoryResource(resourceID))
	resourceName := "eon_resource_backup_exclusion.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_resource_backup_exclusion" "test" {
  resource_id = %q
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "resource_id", resourceID),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     resourceID,
			},
			{
				PreConfig: func() {
					server.CancelResourceExclusion(resourceID)
				},
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_resource_backup_exclusion" "test" {
  resource_id = %q
}
`, resourceID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccResourceDataClassesOverride(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	t.Setenv("EON_USE_EXACT_ENDPOINT", "true")

	resourceID := "res-dataclass-1"
	server.AddResource(newTestInventoryResource(resourceID))
	resourceName := "eon_resource_data_classes_override.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_resource_data_classes_override" "test" {
  resource_id  = %q
  data_classes = ["PII", "PCI"]
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "resource_id", resourceID),
					resource.TestCheckResourceAttr(resourceName, "data_classes.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "data_classes.*", "PII"),
					resource.TestCheckTypeSetElemAttr(resourceName, "data_classes.*", "PCI"),
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
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_resource_data_classes_override" "test" {
  resource_id  = %q
  data_classes = ["PII", "PHI"]
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "data_classes.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "data_classes.*", "PII"),
					resource.TestCheckTypeSetElemAttr(resourceName, "data_classes.*", "PHI"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     resourceID,
			},
			{
				PreConfig: func() {
					server.RemoveDataClassesOverride(resourceID)
				},
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_resource_data_classes_override" "test" {
  resource_id  = %q
  data_classes = ["PII", "PHI"]
}
`, resourceID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

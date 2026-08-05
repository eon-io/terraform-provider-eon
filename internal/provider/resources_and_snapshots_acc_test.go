package provider

import (
	"fmt"
	"testing"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccResourceEnvironmentOverride(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	t.Setenv("EON_USE_EXACT_ENDPOINT", "true")

	resourceID := "res-environment-1"
	server.AddResource(newTestInventoryResource(resourceID))
	resourceName := "eon_resource_environment_override.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_resource_environment_override" "test" {
  resource_id = %q
  environment = "PROD"
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "resource_id", resourceID),
					resource.TestCheckResourceAttr(resourceName, "environment", "PROD"),
					resource.TestCheckResourceAttr(resourceName, "id", resourceID),
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
resource "eon_resource_environment_override" "test" {
  resource_id = %q
  environment = "STAGE"
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "environment", "STAGE"),
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
					server.RemoveEnvironmentOverride(resourceID)
				},
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_resource_environment_override" "test" {
  resource_id = %q
  environment = "STAGE"
}
`, resourceID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccResources(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	t.Setenv("EON_USE_EXACT_ENDPOINT", "true")

	server.AddResource(newTestInventoryResource("res-list-1"))
	server.AddResource(newTestInventoryResource("res-list-2"))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_resources" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_resources.all", "resources.#", "2"),
					resource.TestCheckResourceAttrSet("data.eon_resources.all", "resources.0.id"),
					resource.TestCheckResourceAttrSet("data.eon_resources.all", "resources.0.provider_resource_id"),
					resource.TestCheckResourceAttrSet("data.eon_resources.all", "resources.0.resource_name"),
					resource.TestCheckResourceAttrSet("data.eon_resources.all", "resources.0.cloud_provider"),
					resource.TestCheckResourceAttrSet("data.eon_resources.all", "resources.0.resource_type"),
				),
			},
			{
				Config: server.providerConfig() + `
data "eon_resources" "filtered" {
  ids = ["res-list-1"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_resources.filtered", "resources.#", "1"),
					resource.TestCheckResourceAttr("data.eon_resources.filtered", "resources.0.id", "res-list-1"),
					resource.TestCheckResourceAttr("data.eon_resources.filtered", "ids.#", "1"),
					resource.TestCheckResourceAttr("data.eon_resources.filtered", "ids.0", "res-list-1"),
				),
			},
		},
	})
}

func TestAccResourceSnapshots(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	t.Setenv("EON_USE_EXACT_ENDPOINT", "true")

	resourceID := "res-snapshots-1"
	server.AddResource(newTestInventoryResource(resourceID))

	created := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	pointInTime := time.Date(2024, 6, 15, 11, 0, 0, 0, time.UTC)
	snap := externalEonSdkAPI.NewSnapshot("snap-1", created, resourceID)
	snap.SetProjectId(testAccProjectID)
	snap.SetVaultId("vault-1")
	snap.SetPointInTime(pointInTime)
	snap.SetOnHold(false)
	server.AddSnapshot(resourceID, snap)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + fmt.Sprintf(`
data "eon_resource_snapshots" "demo" {
  resource_id = %q
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_resource_snapshots.demo", "resource_id", resourceID),
					resource.TestCheckResourceAttr("data.eon_resource_snapshots.demo", "snapshots.#", "1"),
					resource.TestCheckResourceAttr("data.eon_resource_snapshots.demo", "snapshots.0.id", "snap-1"),
					resource.TestCheckResourceAttr("data.eon_resource_snapshots.demo", "snapshots.0.resource_id", resourceID),
					resource.TestCheckResourceAttr("data.eon_resource_snapshots.demo", "snapshots.0.vault_id", "vault-1"),
					resource.TestCheckResourceAttr("data.eon_resource_snapshots.demo", "snapshots.0.on_hold", "false"),
					resource.TestCheckResourceAttrSet("data.eon_resource_snapshots.demo", "snapshots.0.created_at"),
					resource.TestCheckResourceAttrSet("data.eon_resource_snapshots.demo", "snapshots.0.point_in_time"),
				),
			},
		},
	})
}

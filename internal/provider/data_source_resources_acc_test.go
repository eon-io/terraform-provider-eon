package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResources(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	resourceOne := newTestInventoryResource("res-acc-1")
	resourceOne.SetProviderResourceId("i-filter-one")
	resourceTwo := newTestInventoryResource("res-acc-2")
	resourceTwo.SetProviderResourceId("i-filter-two")
	server.AddResource(resourceOne)
	server.AddResource(resourceTwo)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_resources" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_resources.all", "total_count", "2"),
					resource.TestCheckResourceAttr("data.eon_resources.all", "resources.#", "2"),
				),
			},
			{
				Config: server.providerConfig() + `
data "eon_resources" "filtered" {
  provider_resource_ids = ["i-filter-one"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_resources.filtered", "total_count", "1"),
					resource.TestCheckResourceAttr("data.eon_resources.filtered", "resources.#", "1"),
					resource.TestCheckResourceAttr("data.eon_resources.filtered", "resources.0.provider_resource_id", "i-filter-one"),
					resource.TestCheckResourceAttr("data.eon_resources.filtered", "resources.0.id", "res-acc-1"),
				),
			},
			{
				PreConfig: func() {
					server.DeleteResource("res-acc-2")
				},
				Config: server.providerConfig() + `
data "eon_resources" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_resources.all", "total_count", "1"),
					resource.TestCheckResourceAttr("data.eon_resources.all", "resources.#", "1"),
					resource.TestCheckResourceAttr("data.eon_resources.all", "resources.0.id", "res-acc-1"),
				),
			},
		},
	})
}

func TestAccResourcesRealEnv(t *testing.T) {
	testAccRealEnvPreCheck(t, false)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRealProviderConfig() + `
data "eon_resources" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.eon_resources.all", "total_count"),
					resource.TestCheckResourceAttrSet("data.eon_resources.all", "resources.#"),
				),
			},
		},
	})
}

func TestAccResourcesWithRelatedResource(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	resourceID := "res-related-1"
	server.AddResource(newTestInventoryResource(resourceID))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_resource_environment_override" "test" {
  resource_id = %q
  environment = "PROD"
}

data "eon_resources" "matched" {
  ids = [eon_resource_environment_override.test.resource_id]
  depends_on = [eon_resource_environment_override.test]
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_resources.matched", "total_count", "1"),
					resource.TestCheckResourceAttr("data.eon_resources.matched", "resources.0.id", resourceID),
					resource.TestCheckResourceAttr("data.eon_resources.matched", "resources.0.environment", "PROD"),
				),
			},
			{
				Config: server.providerConfig() + fmt.Sprintf(`
resource "eon_resource_environment_override" "test" {
  resource_id = %q
  environment = "STAGE"
}

data "eon_resources" "matched" {
  ids = [eon_resource_environment_override.test.resource_id]
  depends_on = [eon_resource_environment_override.test]
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_resources.matched", "resources.0.environment", "STAGE"),
				),
			},
			{
				PreConfig: func() {
					server.DeleteResource(resourceID)
				},
				Config: server.providerConfig() + fmt.Sprintf(`
data "eon_resources" "matched" {
  ids = [%q]
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_resources.matched", "total_count", "0"),
					resource.TestCheckResourceAttr("data.eon_resources.matched", "resources.#", "0"),
				),
			},
		},
	})
}

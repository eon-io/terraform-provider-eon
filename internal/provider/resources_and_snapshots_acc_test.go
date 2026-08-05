package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccRealEnvPreCheck(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)

	required := []string{
		"EON_ENDPOINT",
		"EON_CLIENT_ID",
		"EON_CLIENT_SECRET",
		"EON_PROJECT_ID",
		"EON_TEST_RESOURCE_ID",
	}
	for _, key := range required {
		if os.Getenv(key) == "" {
			t.Skipf("%s must be set for real-environment acceptance tests", key)
		}
	}
}

func testAccRealProviderConfig() string {
	return fmt.Sprintf(`
provider "eon" {
  endpoint      = %q
  client_id     = %q
  client_secret = %q
  project_id    = %q
}
`, os.Getenv("EON_ENDPOINT"), os.Getenv("EON_CLIENT_ID"), os.Getenv("EON_CLIENT_SECRET"), os.Getenv("EON_PROJECT_ID"))
}

func TestAccResourceEnvironmentOverride(t *testing.T) {
	testAccRealEnvPreCheck(t)

	resourceID := os.Getenv("EON_TEST_RESOURCE_ID")
	resourceName := "eon_resource_environment_override.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRealProviderConfig() + fmt.Sprintf(`
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
				Config: testAccRealProviderConfig() + fmt.Sprintf(`
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
		},
	})
}

func TestAccResourceSnapshots(t *testing.T) {
	testAccRealEnvPreCheck(t)

	resourceID := os.Getenv("EON_TEST_RESOURCE_ID")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRealProviderConfig() + fmt.Sprintf(`
data "eon_resource_snapshots" "demo" {
  resource_id = %q
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_resource_snapshots.demo", "resource_id", resourceID),
					resource.TestCheckResourceAttrSet("data.eon_resource_snapshots.demo", "snapshots.#"),
				),
			},
		},
	})
}

package provider

import (
	"fmt"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBackupPostureControl(t *testing.T) {
	testAccPreCheck(t)

	fake := newFakeEonServer()
	t.Cleanup(fake.Close)
	setupAccTestEnv(t, fake)

	resourceName := "eon_backup_posture_control.test"
	dataSourceName := "data.eon_backup_posture_controls.all"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccBackupPostureControlConfig("acc-control", "HIGH", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "acc-control"),
					resource.TestCheckResourceAttr(resourceName, "severity", "HIGH"),
					resource.TestCheckResourceAttr(resourceName, "resource_selector.resource_selection_mode", "ALL"),
					resource.TestCheckResourceAttr(resourceName, "rules.cross_region", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				Config: testAccBackupPostureControlConfig("acc-control-updated", "MEDIUM", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "acc-control-updated"),
					resource.TestCheckResourceAttr(resourceName, "severity", "MEDIUM"),
					resource.TestCheckResourceAttr(resourceName, "rules.cross_region", "false"),
					resource.TestCheckResourceAttr(resourceName, "rules.cross_account", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccBackupPostureControlConfigWithDataSource("acc-control-updated", "MEDIUM", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "controls.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "controls.0.name", "acc-control-updated"),
					resource.TestCheckResourceAttr(dataSourceName, "controls.0.severity", "MEDIUM"),
					resource.TestCheckResourceAttr(dataSourceName, "controls.0.resource_selection_mode", "ALL"),
				),
			},
			{
				PreConfig: func() {
					for _, id := range fake.BackupPostureControlIDs() {
						fake.DeleteBackupPostureControl(id)
					}
				},
				Config:             testAccBackupPostureControlConfigWithDataSource("acc-control-updated", "MEDIUM", false),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccBackupPostureControl_OutOfBandDelete(t *testing.T) {
	testAccPreCheck(t)

	fake := newFakeEonServer()
	t.Cleanup(fake.Close)
	setupAccTestEnv(t, fake)

	resourceName := "eon_backup_posture_control.test"
	var controlID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccBackupPostureControlConfig("acc-control-oob", "LOW", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("resource not found: %s", resourceName)
						}
						controlID = rs.Primary.ID
						return nil
					},
				),
			},
			{
				PreConfig: func() {
					fake.DeleteBackupPostureControl(controlID)
				},
				Config:             testAccBackupPostureControlConfig("acc-control-oob", "LOW", true),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccIdps(t *testing.T) {
	testAccPreCheck(t)

	fake := newFakeEonServer()
	t.Cleanup(fake.Close)
	setupAccTestEnv(t, fake)

	fake.SeedIdp(*externalEonSdkAPI.NewIdp("idp-1", "Okta"))
	fake.SeedIdp(*externalEonSdkAPI.NewIdp("idp-2", "Azure AD"))

	dataSourceName := "data.eon_idps.all"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `data "eon_idps" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "idps.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "idps.0.id", "idp-1"),
					resource.TestCheckResourceAttr(dataSourceName, "idps.0.provider_name", "Okta"),
					resource.TestCheckResourceAttr(dataSourceName, "idps.1.id", "idp-2"),
					resource.TestCheckResourceAttr(dataSourceName, "idps.1.provider_name", "Azure AD"),
				),
			},
		},
	})
}

func TestAccBackupPostureControls(t *testing.T) {
	testAccPreCheck(t)

	fake := newFakeEonServer()
	t.Cleanup(fake.Close)
	setupAccTestEnv(t, fake)

	resourceName := "eon_backup_posture_control.test"
	dataSourceName := "data.eon_backup_posture_controls.all"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccBackupPostureControlConfigWithDataSource("ds-control", "HIGH", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "ds-control"),
					resource.TestCheckResourceAttr(dataSourceName, "controls.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "controls.0.name", "ds-control"),
					resource.TestCheckResourceAttr(dataSourceName, "controls.0.severity", "HIGH"),
					func(s *terraform.State) error {
						if got := len(fake.BackupPostureControlIDs()); got != 1 {
							return fmt.Errorf("expected 1 control in fake server, got %d", got)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccBackupPostureControlConfig(name, severity string, crossRegion bool) string {
	crossAccount := "false"
	if !crossRegion {
		crossAccount = "true"
	}
	return fmt.Sprintf(`
resource "eon_backup_posture_control" "test" {
  name     = %q
  severity = %q

  resource_selector = {
    resource_selection_mode = "ALL"
  }

  rules = {
    cross_region  = %t
    cross_account = %s
  }
}
`, name, severity, crossRegion, crossAccount)
}

func testAccBackupPostureControlConfigWithDataSource(name, severity string, crossRegion bool) string {
	return testAccBackupPostureControlConfig(name, severity, crossRegion) + `
data "eon_backup_posture_controls" "all" {
  depends_on = [eon_backup_posture_control.test]
}
`
}

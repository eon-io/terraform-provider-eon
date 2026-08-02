package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccBackupPostureControl_lifecycle drives the full resource lifecycle
// (create, read, update, import, destroy) plus the plural data source through
// the real terraform CLI against the fake Eon API server. Runs only with
// TF_ACC=1 (make testacc).
func TestAccBackupPostureControl_lifecycle(t *testing.T) {
	fake := newFakeEonServer(t)
	providerConfig := accProviderConfig(t, fake)

	controlConfig := func(name, severity string, crossRegion bool) string {
		return providerConfig + fmt.Sprintf(`
resource "eon_backup_posture_control" "test" {
  name     = %q
  severity = %q

  resource_selector = {
    resource_selection_mode = "ALL"
  }

  rules = {
    minimum_retention = [
      {
        frequency              = "DAILY"
        minimum_retention_days = 30
      }
    ]
    cross_region = %t
  }
}
`, name, severity, crossRegion)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back.
			{
				Config: controlConfig("baseline-retention", "HIGH", true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("eon_backup_posture_control.test", tfjsonpath.New("name"), knownvalue.StringExact("baseline-retention")),
					statecheck.ExpectKnownValue("eon_backup_posture_control.test", tfjsonpath.New("severity"), knownvalue.StringExact("HIGH")),
					statecheck.ExpectKnownValue("eon_backup_posture_control.test", tfjsonpath.New("rules").AtMapKey("cross_region"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("eon_backup_posture_control.test", tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
			// In-place update of name, severity, and a rule.
			{
				Config: controlConfig("baseline-retention-v2", "MEDIUM", false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("eon_backup_posture_control.test", tfjsonpath.New("name"), knownvalue.StringExact("baseline-retention-v2")),
					statecheck.ExpectKnownValue("eon_backup_posture_control.test", tfjsonpath.New("severity"), knownvalue.StringExact("MEDIUM")),
					statecheck.ExpectKnownValue("eon_backup_posture_control.test", tfjsonpath.New("rules").AtMapKey("cross_region"), knownvalue.Bool(false)),
				},
			},
			// Import by ID round-trips the full state.
			{
				ResourceName:      "eon_backup_posture_control.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// The plural data source sees the managed control.
			{
				Config: controlConfig("baseline-retention-v2", "MEDIUM", false) + `
data "eon_backup_posture_controls" "all" {
  depends_on = [eon_backup_posture_control.test]
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.eon_backup_posture_controls.all", tfjsonpath.New("controls"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue("data.eon_backup_posture_controls.all",
						tfjsonpath.New("controls").AtSliceIndex(0).AtMapKey("name"),
						knownvalue.StringExact("baseline-retention-v2")),
				},
			},
		},
	})
}

// TestAccBackupPostureControl_disappears verifies the provider recovers when
// the control vanishes out-of-band: the next plan must propose re-creation
// rather than erroring.
func TestAccBackupPostureControl_disappears(t *testing.T) {
	fake := newFakeEonServer(t)
	providerConfig := accProviderConfig(t, fake)

	config := providerConfig + `
resource "eon_backup_posture_control" "test" {
  name     = "vanishing-control"
  severity = "LOW"

  resource_selector = {
    resource_selection_mode = "ALL"
  }

  rules = {
    cross_account = true
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: func() {
					fake.mu.Lock()
					defer fake.mu.Unlock()
					for id := range fake.postureControls {
						delete(fake.postureControls, id)
					}
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

terraform {
  required_providers {
    eon = {
      source = "eon-io/eon"
    }
  }
}

# Retrieve all backup posture controls in the project
data "eon_backup_posture_controls" "all" {}

output "high_severity_controls" {
  value = [
    for control in data.eon_backup_posture_controls.all.controls :
    control.name if control.severity == "HIGH"
  ]
}

# Example: List all backup posture controls
data "eon_backup_posture_controls" "all" {}

# Example: Filter high-severity controls
locals {
  high_severity_controls = [
    for control in data.eon_backup_posture_controls.all.controls :
    control if control.severity == "HIGH"
  ]
}

output "backup_posture_controls_count" {
  description = "Total number of backup posture controls"
  value       = length(data.eon_backup_posture_controls.all.controls)
}

output "high_severity_controls_count" {
  description = "Number of high-severity backup posture controls"
  value       = length(local.high_severity_controls)
}

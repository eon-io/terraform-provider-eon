# Example: List all backup posture controls
data "eon_backup_posture_controls" "all" {}

output "backup_posture_control_ids" {
  description = "IDs of all backup posture controls"
  value       = [for c in data.eon_backup_posture_controls.all.controls : c.id]
}

output "high_severity_controls" {
  description = "Names of HIGH severity backup posture controls"
  value = [
    for c in data.eon_backup_posture_controls.all.controls : c.name
    if c.severity == "HIGH"
  ]
}

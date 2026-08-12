# Example: List all action approval rules
data "eon_action_approval_rules" "all" {}

output "action_approval_rule_ids" {
  description = "IDs of all action approval rules"
  value       = [for r in data.eon_action_approval_rules.all.rules : r.id]
}

data "eon_action_approval_rules" "restore" {
  operation = "RESTORE_RESOURCE"
}

output "restore_approval_rules" {
  description = "Rules that protect restore operations"
  value       = [for r in data.eon_action_approval_rules.restore.rules : r.id]
}

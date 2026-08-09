# Example: List all action approval rules
data "eon_action_approval_rules" "all" {}

output "action_approval_rule_ids" {
  description = "IDs of all action approval rules"
  value       = [for r in data.eon_action_approval_rules.all.rules : r.id]
}

output "restore_resource_rules" {
  description = "Rules that protect restore operations"
  value = [
    for r in data.eon_action_approval_rules.all.rules : r.id
    if r.operation == "RESTORE_RESOURCE"
  ]
}

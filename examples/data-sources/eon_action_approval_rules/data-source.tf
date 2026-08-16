data "eon_action_approval_rules" "all" {}

output "action_approval_rule_ids" {
  value = [for rule in data.eon_action_approval_rules.all.rules : rule.id]
}

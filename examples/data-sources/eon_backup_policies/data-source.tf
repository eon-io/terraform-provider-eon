# Example: List all backup policies
data "eon_backup_policies" "all" {}

# Example: Filter enabled backup policies
locals {
  enabled_backup_policies = [
    for policy in data.eon_backup_policies.all.policies :
    policy if policy.enabled
  ]
}

# Example: Output backup policy information
output "backup_policies_count" {
  description = "Total number of backup policies"
  value       = length(data.eon_backup_policies.all.policies)
}

output "enabled_backup_policies_count" {
  description = "Number of enabled backup policies"
  value       = length(local.enabled_backup_policies)
}

output "backup_policies_summary" {
  description = "Summary of all backup policies"
  value = {
    for policy in data.eon_backup_policies.all.policies :
    policy.name => {
      id      = policy.id
      enabled = policy.enabled
    }
  }
}

# Example: Use backup policy data in other resources
resource "local_file" "backup_policy_report" {
  filename = "backup_policies_report.json"
  content = jsonencode({
    total_policies   = length(data.eon_backup_policies.all.policies)
    enabled_policies = length(local.enabled_backup_policies)
    policies = [
      for policy in data.eon_backup_policies.all.policies : {
        name    = policy.name
        id      = policy.id
        enabled = policy.enabled
      }
    ]
  })
} 
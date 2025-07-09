# Example: Basic backup policy
resource "eon_backup_policy" "daily_backup" {
  name                    = "Daily Production Backup"
  enabled                 = true
  resource_selection_mode = "ALL"
  backup_policy_type      = "STANDARD"

  resource_inclusion_override = []
  resource_exclusion_override = []
}

# Example: High frequency backup policy
resource "eon_backup_policy" "high_frequency_backup" {
  name                    = "High Frequency Backup"
  enabled                 = true
  resource_selection_mode = "ALL"
  backup_policy_type      = "HIGH_FREQUENCY"

  resource_inclusion_override = []
  resource_exclusion_override = []
}

# Example: Backup policy with specific resource inclusions
resource "eon_backup_policy" "specific_resources" {
  name                    = "Specific Resources Backup"
  enabled                 = true
  resource_selection_mode = "CONDITIONAL"
  backup_policy_type      = "STANDARD"

  resource_inclusion_override = [
    "i-1234567890abcdef0"
  ]

  resource_exclusion_override = [
    "i-0987654321fedcba0"
  ]
}

# Output examples
output "daily_backup_policy_id" {
  description = "ID of the daily backup policy"
  value       = eon_backup_policy.daily_backup.id
}

output "backup_policies_summary" {
  description = "Summary of all backup policies"
  value = {
    daily_backup = {
      id      = eon_backup_policy.daily_backup.id
      name    = eon_backup_policy.daily_backup.name
      enabled = eon_backup_policy.daily_backup.enabled
      type    = eon_backup_policy.daily_backup.backup_policy_type
    }
    high_frequency_backup = {
      id      = eon_backup_policy.high_frequency_backup.id
      name    = eon_backup_policy.high_frequency_backup.name
      enabled = eon_backup_policy.high_frequency_backup.enabled
      type    = eon_backup_policy.high_frequency_backup.backup_policy_type
    }
  }
} 
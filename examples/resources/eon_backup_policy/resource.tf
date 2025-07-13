# Example: Basic backup policy
resource "eon_backup_policy" "daily_backup" {
  name                    = "Daily Production Backup"
  enabled                 = true
  resource_selection_mode = "ALL"
  backup_policy_type      = "STANDARD"

  backup_schedules {
    vault_id       = "vault-12345678-1234-1234-1234-123456789012"
    retention_days = 30
  }

  schedule_frequency   = "DAILY"
  time_of_day_hour     = 2
  time_of_day_minutes  = 0
  start_window_minutes = 240

  resource_inclusion_override = []
  resource_exclusion_override = []
}

# Example: High frequency backup policy
resource "eon_backup_policy" "high_frequency_backup" {
  name                    = "High Frequency Backup"
  enabled                 = true
  resource_selection_mode = "ALL"
  backup_policy_type      = "HIGH_FREQUENCY"

  backup_schedules {
    vault_id       = "vault-12345678-1234-1234-1234-123456789012"
    retention_days = 7
  }

  schedule_frequency = "INTERVAL"
  interval_minutes   = 30

  high_frequency_resource_types = ["AWS_S3", "AWS_DYNAMO_DB"]

  resource_inclusion_override = []
  resource_exclusion_override = []
}

# Example: Multi-vault backup policy
resource "eon_backup_policy" "multi_vault_backup" {
  name                    = "Multi-Vault Production Backup"
  enabled                 = true
  resource_selection_mode = "ALL"
  backup_policy_type      = "STANDARD"

  backup_schedules {
    vault_id       = "vault-12345678-1234-1234-1234-123456789012"
    retention_days = 30
  }

  backup_schedules {
    vault_id       = "vault-87654321-4321-4321-4321-210987654321"
    retention_days = 90
  }

  schedule_frequency   = "WEEKLY"
  time_of_day_hour     = 3
  time_of_day_minutes  = 30
  start_window_minutes = 360

  resource_inclusion_override = []
  resource_exclusion_override = []
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
    multi_vault_backup = {
      id      = eon_backup_policy.multi_vault_backup.id
      name    = eon_backup_policy.multi_vault_backup.name
      enabled = eon_backup_policy.multi_vault_backup.enabled
      type    = eon_backup_policy.multi_vault_backup.backup_policy_type
    }
  }
} 
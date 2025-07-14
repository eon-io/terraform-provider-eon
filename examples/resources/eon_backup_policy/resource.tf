terraform {
  required_providers {
    eon = {
      source = "eon-io/eon"
    }
  }
}

# Example: Basic backup policy with daily schedule
resource "eon_backup_policy" "daily_backup" {
  name          = "Daily Production Backup"
  enabled       = true
  schedule_mode = "STANDARD"

  resource_selector = {
    resource_selection_mode = "ALL"
  }

  backup_plan = {
    backup_policy_type = "STANDARD"
    standard_plan = {
      backup_schedules = [
        {
          vault_id       = "vault-12345678-1234-1234-1234-123456789012"
          retention_days = 30
          schedule_config = {
            frequency = "DAILY"
            daily_config = {
              time_of_day_hour     = 2
              time_of_day_minutes  = 0
              start_window_minutes = 240
            }
          }
        }
      ]
    }
  }
}

# Example: High frequency backup policy
resource "eon_backup_policy" "high_frequency_backup" {
  name          = "High Frequency Critical Data Backup"
  enabled       = true
  schedule_mode = "STANDARD"

  resource_selector = {
    resource_selection_mode = "ALL"
  }

  backup_plan = {
    backup_policy_type = "HIGH_FREQUENCY"
    high_frequency_plan = {
      resource_types = ["RDS_INSTANCE", "DYNAMO_DB_TABLE"]
      backup_schedules = [
        {
          vault_id       = "vault-12345678-1234-1234-1234-123456789012"
          retention_days = 7
          schedule_config = {
            frequency = "INTERVAL"
            interval_config = {
              interval_hours = 1
            }
          }
        }
      ]
    }
  }
}

# Example: Weekly backup policy
resource "eon_backup_policy" "weekly_backup" {
  name          = "Weekly Production Backup"
  enabled       = true
  schedule_mode = "STANDARD"

  resource_selector = {
    resource_selection_mode = "ALL"
  }

  backup_plan = {
    backup_policy_type = "STANDARD"
    standard_plan = {
      backup_schedules = [
        {
          vault_id       = "vault-87654321-4321-4321-4321-210987654321"
          retention_days = 90
          schedule_config = {
            frequency = "WEEKLY"
            weekly_config = {
              days_of_week         = ["SUNDAY"]
              time_of_day_hour     = 3
              time_of_day_minutes  = 30
              start_window_minutes = 360
            }
          }
        }
      ]
    }
  }
}

# Example: Monthly backup policy
resource "eon_backup_policy" "monthly_backup" {
  name          = "Monthly Archive Backup"
  enabled       = true
  schedule_mode = "STANDARD"

  resource_selector = {
    resource_selection_mode = "ALL"
  }

  backup_plan = {
    backup_policy_type = "STANDARD"
    standard_plan = {
      backup_schedules = [
        {
          vault_id       = "vault-monthly-archive"
          retention_days = 365
          schedule_config = {
            frequency = "MONTHLY"
            monthly_config = {
              days_of_month        = [1, 15]
              time_of_day_hour     = 1
              time_of_day_minutes  = 0
              start_window_minutes = 480
            }
          }
        }
      ]
    }
  }
}

# Example: Conditional backup policy using new condition types
resource "eon_backup_policy" "conditional_backup" {
  name          = "Conditional Production Backup"
  enabled       = true
  schedule_mode = "STANDARD"

  resource_selector = {
    resource_selection_mode = "CONDITIONAL"

    expression = {
      group = {
        operator = "AND"
        operands = [
          {
            # Basic resource type condition
            resource_type = {
              operator       = "IN"
              resource_types = ["EC2_INSTANCE", "EBS_VOLUME"]
            }
          },
          {
            # Environment condition
            environment = {
              operator     = "IN"
              environments = ["production", "staging"]
            }
          },
          {
            # NEW: Data classes condition
            data_classes = {
              operator     = "CONTAINS"
              data_classes = ["PII", "CONFIDENTIAL"]
            }
          },
          {
            # NEW: Cloud provider condition
            cloud_provider = {
              operator        = "IN"
              cloud_providers = ["AWS", "AZURE"]
            }
          }
        ]
      }
    }
  }

  backup_plan = {
    backup_policy_type = "STANDARD"
    standard_plan = {
      backup_schedules = [
        {
          vault_id       = "vault-conditional"
          retention_days = 60
          schedule_config = {
            frequency = "DAILY"
            daily_config = {
              time_of_day_hour     = 2
              time_of_day_minutes  = 0
              start_window_minutes = 240
            }
          }
        }
      ]
    }
  }
}

# Example: Comprehensive condition types demonstration
resource "eon_backup_policy" "all_condition_types" {
  name          = "All Condition Types Demo"
  enabled       = true
  schedule_mode = "STANDARD"

  resource_selector = {
    resource_selection_mode = "CONDITIONAL"

    expression = {
      group = {
        operator = "AND"
        operands = [
          {
            # 1. Resource Type (existing)
            resource_type = {
              operator       = "IN"
              resource_types = ["EC2_INSTANCE", "EBS_VOLUME"]
            }
          },
          {
            # 2. Environment (existing)
            environment = {
              operator     = "IN"
              environments = ["production"]
            }
          },
          {
            # 3. Tag Keys (existing)
            tag_keys = {
              operator = "CONTAINS"
              tag_keys = ["Environment", "Owner"]
            }
          },
          {
            # 4. Tag Key Values (existing)
            tag_key_values = {
              operator       = "CONTAINS"
              tag_key_values = ["production", "critical"]
            }
          },
          {
            # 5. Data Classes (NEW)
            data_classes = {
              operator     = "CONTAINS"
              data_classes = ["PII", "CONFIDENTIAL"]
            }
          },
          {
            # 6. Apps (NEW)
            apps = {
              operator = "CONTAINS"
              apps     = ["web-app", "database"]
            }
          },
          {
            # 7. Cloud Provider (NEW)
            cloud_provider = {
              operator        = "IN"
              cloud_providers = ["AWS"]
            }
          },
          {
            # 8. Account ID (NEW)
            account_id = {
              operator    = "IN"
              account_ids = ["123456789012"]
            }
          },
          {
            # 9. Source Region (NEW)
            source_region = {
              operator       = "IN"
              source_regions = ["us-east-1", "us-west-2"]
            }
          },
          {
            # 10. VPC (NEW)
            vpc = {
              operator = "IN"
              vpcs     = ["vpc-production"]
            }
          },
          {
            # 11. Subnets (NEW)
            subnets = {
              operator = "CONTAINS"
              subnets  = ["subnet-web-tier", "subnet-db-tier"]
            }
          },
          {
            # 12. Resource Group Name (NEW)
            resource_group_name = {
              operator             = "CONTAINS"
              resource_group_names = ["production-rg"]
            }
          },
          {
            # 13. Resource Name (NEW)
            resource_name = {
              operator       = "CONTAINS"
              resource_names = ["prod-", "critical-"]
            }
          },
          {
            # 14. Resource ID (NEW)
            resource_id = {
              operator     = "IN"
              resource_ids = ["i-123456789abcdef0"]
            }
          }
        ]
      }
    }
  }

  backup_plan = {
    backup_policy_type = "STANDARD"
    standard_plan = {
      backup_schedules = [
        {
          vault_id       = "vault-all-conditions"
          retention_days = 180
          schedule_config = {
            frequency = "WEEKLY"
            weekly_config = {
              days_of_week         = ["SUNDAY", "WEDNESDAY"]
              time_of_day_hour     = 2
              time_of_day_minutes  = 30
              start_window_minutes = 180
            }
          }
        }
      ]
    }
  }
}

# Output examples
output "daily_backup_policy_id" {
  description = "ID of the daily backup policy"
  value       = eon_backup_policy.daily_backup.id
}

output "high_frequency_backup_policy_id" {
  description = "ID of the high frequency backup policy"
  value       = eon_backup_policy.high_frequency_backup.id
}

output "weekly_backup_policy_id" {
  description = "ID of the weekly backup policy"
  value       = eon_backup_policy.weekly_backup.id
}

output "monthly_backup_policy_id" {
  description = "ID of the monthly backup policy"
  value       = eon_backup_policy.monthly_backup.id
}

output "conditional_backup_policy_id" {
  description = "ID of the conditional backup policy"
  value       = eon_backup_policy.conditional_backup.id
}

output "all_condition_types_policy_id" {
  description = "ID of the policy demonstrating all condition types"
  value       = eon_backup_policy.all_condition_types.id
}

output "backup_policies_summary" {
  description = "Summary of all backup policies"
  value = {
    daily_backup = {
      id      = eon_backup_policy.daily_backup.id
      name    = eon_backup_policy.daily_backup.name
      enabled = eon_backup_policy.daily_backup.enabled
    }
    high_frequency_backup = {
      id      = eon_backup_policy.high_frequency_backup.id
      name    = eon_backup_policy.high_frequency_backup.name
      enabled = eon_backup_policy.high_frequency_backup.enabled
    }
    weekly_backup = {
      id      = eon_backup_policy.weekly_backup.id
      name    = eon_backup_policy.weekly_backup.name
      enabled = eon_backup_policy.weekly_backup.enabled
    }
    monthly_backup = {
      id      = eon_backup_policy.monthly_backup.id
      name    = eon_backup_policy.monthly_backup.name
      enabled = eon_backup_policy.monthly_backup.enabled
    }
    conditional_backup = {
      id      = eon_backup_policy.conditional_backup.id
      name    = eon_backup_policy.conditional_backup.name
      enabled = eon_backup_policy.conditional_backup.enabled
    }
    all_condition_types = {
      id      = eon_backup_policy.all_condition_types.id
      name    = eon_backup_policy.all_condition_types.name
      enabled = eon_backup_policy.all_condition_types.enabled
    }
  }
} 
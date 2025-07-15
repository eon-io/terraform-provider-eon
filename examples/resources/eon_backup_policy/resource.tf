terraform {
  required_providers {
    eon = {
      source = "eon-io/eon"
    }
  }
}

# Example: Basic backup policy with daily schedule
resource "eon_backup_policy" "daily_backup" {
  name    = "Daily Production Backup"
  enabled = true
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
  name    = "High Frequency Critical Data Backup"
  enabled = true
  resource_selector = {
    resource_selection_mode = "ALL"
  }

  backup_plan = {
    backup_policy_type = "HIGH_FREQUENCY"
    high_frequency_plan = {
      resource_types = ["AWS_S3", "AWS_DYNAMO_DB"]
      backup_schedules = [
        {
          vault_id       = "e19a6ad1-6a97-49a1-b7c9-9620977ea018"
          retention_days = 7
          schedule_config = {
            frequency = "INTERVAL"
            interval_config = {
              interval_minutes = 30
            }
          }
        }
      ]
    }
  }
}

# Example: Conditional backup policy using new condition types
resource "eon_backup_policy" "conditional_backup" {
  name    = "Conditional Production Backup"
  enabled = true
  resource_selector = {
    resource_selection_mode = "CONDITIONAL"

    expression = {
      group = {
        operator = "AND"
        operands = [
          {
            resource_type = {
              operator       = "IN"
              resource_types = ["AWS_EC2", "AWS_RDS"]
            }
          },
          {
            environment = {
              operator     = "IN"
              environments = ["PROD"]
            }
          },
          {
            tag_keys = {
              operator = "CONTAINS_ANY_OF"
              tag_keys = ["Environment", "Owner"]
            }
          },
          {
            tag_key_values = {
              operator = "CONTAINS_ANY_OF"
              tag_key_values = [
                {
                  key   = "Environment"
                  value = "Production"
                },
                {
                  key   = "Critical"
                  value = "true"
                }
              ]
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
          vault_id       = "e19a6ad1-6a97-49a1-b7c9-9620977ea018"
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
  name    = "All Condition Types Demo"
  enabled = true
  resource_selector = {
    resource_selection_mode = "CONDITIONAL"

    expression = {
      group = {
        operator = "AND"
        operands = [
          {
            resource_type = {
              operator       = "IN"
              resource_types = ["AWS_EC2", "AWS_RDS"]
            }
          },
          {
            environment = {
              operator     = "IN"
              environments = ["PROD"]
            }
          },
          {
            tag_keys = {
              operator = "CONTAINS_ANY_OF"
              tag_keys = ["Environment", "Owner"]
            }
          },
          {
            data_classes = {
              operator     = "CONTAINS_ANY_OF"
              data_classes = ["PII", "PHI"]
            }
          },
          {
            cloud_provider = {
              operator        = "IN"
              cloud_providers = ["AWS", "AZURE"]
            }
          },
          {
            apps = {
              operator = "CONTAINS_ANY_OF"
              apps     = ["web-app", "database"]
            }
          },
          {
            account_id = {
              operator    = "IN"
              account_ids = ["123456789012"]
            }
          },
          {
            source_region = {
              operator       = "IN"
              source_regions = ["us-east-1", "us-west-2"]
            }
          },
          {
            vpc = {
              operator = "IN"
              vpcs     = ["vpc-production"]
            }
          },
          {
            subnets = {
              operator = "CONTAINS_NONE_OF"
              subnets  = ["subnet-web-tier", "subnet-db-tier"]
            }
          },
          {
            resource_group_name = {
              operator             = "IN"
              resource_group_names = ["production-rg"]
            }
          },
          {
            resource_name = {
              operator       = "IN"
              resource_names = ["prod-", "critical-"]
            }
          },
          {
            resource_id = {
              operator     = "IN"
              resource_ids = ["i-123456789abcdef0"]
            }
          },
          {
            tag_key_values = {
              operator = "CONTAINS_ANY_OF"
              tag_key_values = [
                {
                  key   = "Environment"
                  value = "Production"
                },
                {
                  key   = "Critical"
                  value = "true"
                }
              ]
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
          vault_id       = "e19a6ad1-6a97-49a1-b7c9-9620977ea018"
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

# Output examples
output "daily_backup_policy_id" {
  description = "ID of the daily backup policy"
  value       = eon_backup_policy.daily_backup.id
}

output "high_frequency_backup_policy_id" {
  description = "ID of the high frequency backup policy"
  value       = eon_backup_policy.high_frequency_backup.id
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
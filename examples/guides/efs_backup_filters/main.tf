data "eon_vaults" "all" {}

locals {
  efs_vault_id = one([
    for v in data.eon_vaults.all.vaults :
    v.id if v.cloud_provider == "AWS" && v.region == "us-east-1"
  ])
}

resource "eon_backup_policy" "efs_conditional_backup" {
  name    = "Production EFS Backup - Critical Tier"
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
              resource_types = ["AWS_EFS"]
            }
          },
          {
            tag_key_values = {
              operator = "CONTAINS_ANY_OF"
              tag_key_values = [
                { key = "Tier", value = "Critical" }
              ]
            }
          },
          {
            source_region = {
              operator       = "IN"
              source_regions = ["us-east-1"]
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
          vault_id       = local.efs_vault_id
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

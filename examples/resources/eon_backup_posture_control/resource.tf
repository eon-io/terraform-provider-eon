terraform {
  required_providers {
    eon = {
      source = "eon-io/eon"
    }
  }
}

# Example: Require 30-day daily retention and a cross-region copy for all resources
resource "eon_backup_posture_control" "baseline_retention" {
  name     = "Baseline Retention"
  severity = "HIGH"

  resource_selector = {
    resource_selection_mode = "ALL"
  }

  rules = {
    minimum_retention = [
      {
        frequency              = "DAILY"
        minimum_retention_days = 30
      }
    ]
    cross_region = true
  }
}

# Example: Require redundant copies for production databases only
resource "eon_backup_posture_control" "prod_database_copies" {
  name     = "Production Database Copies"
  severity = "MEDIUM"

  resource_selector = {
    resource_selection_mode = "CONDITIONAL"
    expression = {
      group = {
        operator = "AND"
        operands = [
          {
            environment = {
              operator     = "IN"
              environments = ["PROD"]
            }
          },
          {
            resource_type = {
              operator       = "IN"
              resource_types = ["AWS_RDS"]
            }
          }
        ]
      }
    }
  }

  rules = {
    min_copies    = 2
    cross_account = true
  }
}

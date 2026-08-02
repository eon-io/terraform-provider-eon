terraform {
  required_providers {
    eon = {
      source = "eon-io/eon"
    }
  }
}

# Example: Backup posture control that applies to all resources and requires cross-region copies
resource "eon_backup_posture_control" "cross_region" {
  name     = "Require Cross-Region Backup"
  severity = "HIGH"

  resource_selector = {
    resource_selection_mode = "ALL"
  }

  rules = {
    cross_region     = true
    number_of_copies = { min_copies = 2 }
  }
}

# Example: Conditional control scoped to production environments
resource "eon_backup_posture_control" "prod_retention" {
  name     = "Production Minimum Retention"
  severity = "MEDIUM"

  resource_selector = {
    resource_selection_mode = "CONDITIONAL"
    expression = {
      environment = {
        operator     = "IN"
        environments = ["PROD"]
      }
    }
  }

  rules = {
    minimum_retention = [
      {
        frequency         = "daily"
        minimum_retention = 30
      },
    ]
    maximum_retention = {
      maximum_retention = 365
    }
  }
}

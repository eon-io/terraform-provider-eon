terraform {
  required_providers {
    eon = {
      source = "eon-io/eon"
    }
  }
}

# Example: production resources tagged team=core must keep 30 days of daily backups,
# in at least two copies, one of them in another region.
resource "eon_backup_posture_control" "production_3_2_1" {
  name     = "Production 3-2-1"
  severity = "HIGH"

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
            tag_key_values = {
              operator = "CONTAINS_ANY_OF"
              tag_key_values = [{
                key   = "team"
                value = "core"
              }]
            }
          },
        ]
      }
    }
  }

  rules = {
    minimum_retention = [{
      frequency         = "DAILY"
      minimum_retention = 30
    }]

    number_of_copies = {
      min_copies = 2
    }

    cross_region = true
  }
}

# Example: every resource must keep at least one backup copy, retained no longer than a year.
resource "eon_backup_posture_control" "max_retention" {
  name     = "Retention ceiling"
  severity = "LOW"

  resource_selector = {
    resource_selection_mode = "ALL"
  }

  rules = {
    maximum_retention = {
      maximum_retention = 365
    }
  }
}

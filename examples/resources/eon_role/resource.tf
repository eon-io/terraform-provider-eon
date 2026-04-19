terraform {
  required_providers {
    eon = {
      source = "eon-io/eon"
    }
  }
}

# Example: Custom role with permissions
resource "eon_role" "backup_viewer" {
  name = "Backup Viewer (Custom)"

  permission_grants = [
    { permission = "dashboard.view" },
    { permission = "inventory.view" },
  ]
}

# Example: Another custom role
resource "eon_role" "read_only" {
  name = "Read Only (Custom)"

  permission_grants = [
    { permission = "dashboard.view" },
    { permission = "inventory.view" },
    { permission = "jobs.view" },
    { permission = "snapshots.take_or_convert" },
  ]
}

# Example: IDP group that assigns the custom roles to users in the IdP group
resource "eon_idp_group" "viewers" {
  idp_id            = "your-idp-id"
  provider_group_id = "your-idp-group-id-for-viewers"

  # Reference eon_role resources by ID
  role_ids = [
    eon_role.backup_viewer.id,
    eon_role.read_only.id,
  ]
}

# Example: Role with access conditions (scope permissions to specific environments or resource types)
resource "eon_role" "production_viewer" {
  name = "Production Viewer (Custom)"

  access_conditions = [
    {
      id     = "prod-only"
      effect = "INCLUSIVE"
      expression = {
        environment = {
          operator     = "IN"
          environments = ["PROD"]
        }
      }
    },
  ]

  permission_grants = [
    { permission = "dashboard.view" },
    { permission = "inventory.view" },
    # Restrict this permission to production only via access_condition_id
    { permission = "snapshots.take_or_convert", access_condition_id = "prod-only" },
  ]
}

# Example: Role with resource_type condition (e.g. restrict to EC2 and RDS)
resource "eon_role" "ec2_rds_operator" {
  name = "EC2/RDS Operator (Custom)"

  access_conditions = [
    {
      id     = "ec2-rds-only"
      effect = "INCLUSIVE"
      expression = {
        resource_type = {
          operator       = "IN"
          resource_types = ["AWS_EC2", "AWS_RDS"]
        }
      }
    },
  ]

  permission_grants = [
    { permission = "inventory.view" },
    { permission = "snapshots.take_or_convert", access_condition_id = "ec2-rds-only" },
    { permission = "jobs.view" },
  ]
}

# Example: Role with grouped condition (AND of environment and resource type)
resource "eon_role" "prod_ec2_only" {
  name = "Production EC2 Only (Custom)"

  access_conditions = [
    {
      id     = "prod-ec2"
      effect = "INCLUSIVE"
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
                resource_types = ["AWS_EC2"]
              }
            },
          ]
        }
      }
    },
  ]

  permission_grants = [
    { permission = "inventory.view" },
    { permission = "snapshots.take_or_convert", access_condition_id = "prod-ec2" },
  ]
}

# Example: Role with restore_destination_limits (restrict which restore accounts can be used)
resource "eon_role" "restricted_restores" {
  name = "Restricted Restore Operator (Custom)"

  permission_grants = [
    { permission = "restores.create" },
    { permission = "inventory.view" },
  ]

  # Allow restores only to specific restore account providers
  restore_destination_limits = {
    effect                       = "ALLOW"
    restore_account_provider_ids = ["account-provider-id-1", "account-provider-id-2"]
  }
}

# For built-in roles in eon_idp_group, use the eon_builtin_roles data source (stable keys), not lookup by display name.

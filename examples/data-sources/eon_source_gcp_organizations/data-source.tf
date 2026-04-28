# Example: Retrieve all source GCP organizations
data "eon_source_gcp_organizations" "all" {}

# Example: Output all source GCP organizations information
output "all_source_gcp_organizations" {
  description = "Information about all connected source GCP organizations"
  value = {
    total_organizations = length(data.eon_source_gcp_organizations.all.organizations)
    organizations       = data.eon_source_gcp_organizations.all.organizations
  }
}

# Example: Filter active organizations using locals
locals {
  active_gcp_orgs = [
    for org in data.eon_source_gcp_organizations.all.organizations :
    org if org.state == "ACTIVE"
  ]
}

# Output filtered results
output "active_gcp_organizations" {
  description = "List of active source GCP organizations only"
  value       = local.active_gcp_orgs
}

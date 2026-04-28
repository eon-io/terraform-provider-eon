# Example: Connect a GCP organization for backup operations
resource "eon_source_gcp_organization" "production" {
  organization_id               = "123456789012"
  management_service_account_id = "eon-sa@my-management-project.iam.gserviceaccount.com"
}

# Example: Connect a GCP organization with project exclusion patterns
resource "eon_source_gcp_organization" "production_filtered" {
  organization_id               = "123456789012"
  management_service_account_id = "eon-sa@my-management-project.iam.gserviceaccount.com"

  exclude_project_patterns = [
    "internal-*",
    "test-*",
    "sandbox-*",
  ]
}

# Output the organization details
output "production_org" {
  description = "Details of the connected GCP production organization"
  value = {
    id                       = eon_source_gcp_organization.production.id
    name                     = eon_source_gcp_organization.production.name
    state                    = eon_source_gcp_organization.production.state
    management_project_id    = eon_source_gcp_organization.production.management_project_id
    exclude_project_patterns = eon_source_gcp_organization.production_filtered.exclude_project_patterns
  }
}

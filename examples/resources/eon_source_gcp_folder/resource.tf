# Example: Connect a GCP folder for backup operations
resource "eon_source_gcp_folder" "production" {
  organization_id               = "123456789012"
  folder_id                     = "987654321098"
  management_service_account_id = "eon-sa@my-management-project.iam.gserviceaccount.com"
}

# Example: Connect a GCP folder with project exclusion patterns
resource "eon_source_gcp_folder" "production_filtered" {
  organization_id               = "123456789012"
  folder_id                     = "987654321098"
  management_service_account_id = "eon-sa@my-management-project.iam.gserviceaccount.com"

  exclude_project_patterns = [
    "dev-*",
    "temp-*",
  ]
}

# Output the folder details
output "production_folder" {
  description = "Details of the connected GCP production folder"
  value = {
    id                       = eon_source_gcp_folder.production.id
    name                     = eon_source_gcp_folder.production.name
    state                    = eon_source_gcp_folder.production.state
    management_project_id    = eon_source_gcp_folder.production.management_project_id
    exclude_project_patterns = eon_source_gcp_folder.production_filtered.exclude_project_patterns
  }
}

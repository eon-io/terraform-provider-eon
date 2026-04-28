# Example: Retrieve all source GCP folders
data "eon_source_gcp_folders" "all" {}

# Example: Output all source GCP folders information
output "all_source_gcp_folders" {
  description = "Information about all connected source GCP folders"
  value = {
    total_folders = length(data.eon_source_gcp_folders.all.folders)
    folders       = data.eon_source_gcp_folders.all.folders
  }
}

# Example: Filter active folders using locals
locals {
  active_gcp_folders = [
    for folder in data.eon_source_gcp_folders.all.folders :
    folder if folder.state == "ACTIVE"
  ]
}

# Output filtered results
output "active_gcp_folders" {
  description = "List of active source GCP folders only"
  value       = local.active_gcp_folders
}

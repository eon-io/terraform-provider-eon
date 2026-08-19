# Example: List all inventory resources in the project
data "eon_resources" "all" {}

# Example: Resolve an Eon resource ID by cloud-provider resource ID
data "eon_resources" "by_provider_id" {
  provider_resource_ids = ["i-1234567890abcdef0"]
}

# Example: Filter by resource type and backup status
data "eon_resources" "protected_ec2" {
  resource_types  = ["AWS_EC2"]
  backup_statuses = ["PROTECTED"]
}

output "first_resource_id" {
  description = "Eon-assigned ID of the first listed resource, if any"
  value       = try(data.eon_resources.all.resources[0].id, null)
}

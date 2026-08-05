# Example: Retrieve inventory resources, optionally filtered
data "eon_resources" "aws_ec2" {
  cloud_providers = ["AWS"]
  resource_types  = ["AWS_EC2"]
}

# Example: Resolve a resource ID by provider ID for use in other resources
data "eon_resources" "by_provider_id" {
  provider_resource_ids = ["i-0123456789abcdef0"]
}

output "resolved_resource_id" {
  description = "Eon resource ID resolved from the cloud provider resource ID"
  value       = try(data.eon_resources.by_provider_id.resources[0].id, null)
}

output "aws_ec2_resources" {
  description = "AWS EC2 inventory resources in the project"
  value       = data.eon_resources.aws_ec2.resources
}

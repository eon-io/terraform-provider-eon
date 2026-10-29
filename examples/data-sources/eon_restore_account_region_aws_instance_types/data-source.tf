# Example: List EC2 instance types available for restore in a region and subnet
data "eon_restore_account_region_aws_instance_types" "demo" {
  restore_account_id = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  region             = "us-east-1"
  subnet_id          = "subnet-0123456789abcdef0"
}

# Example: Size the lookup for a restore of two 50 GiB volumes
data "eon_restore_account_region_aws_instance_types" "sized" {
  restore_account_id = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  region             = "us-east-1"
  subnet_id          = "subnet-0123456789abcdef0"
  volume_count       = 2
  total_volume_bytes = 107374182400
}

output "available_instance_types" {
  description = "EC2 instance types available for restore in the subnet"
  value       = data.eon_restore_account_region_aws_instance_types.demo.instance_types
}

output "instant_restore_host_for_m5_large" {
  description = "Instant-restore host instance type backing m5.large, if any"
  value       = try(data.eon_restore_account_region_aws_instance_types.demo.instant_restore_mapping["m5.large"], null)
}

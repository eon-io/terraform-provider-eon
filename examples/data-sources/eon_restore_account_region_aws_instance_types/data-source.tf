# Example: List the EC2 instance types available for restore in a subnet
data "eon_restore_account_region_aws_instance_types" "restore" {
  restore_account_id = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  region             = "us-east-1"
  subnet_id          = "subnet-0123456789abcdef0"
}

# Example: Size the instant-restore host reserve for the volumes being restored
data "eon_restore_account_region_aws_instance_types" "sized" {
  restore_account_id = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  region             = "us-east-1"
  subnet_id          = "subnet-0123456789abcdef0"
  volume_count       = 2
  total_volume_bytes = 214748364800
}

output "restore_instance_types" {
  description = "EC2 instance types available for restore"
  value       = data.eon_restore_account_region_aws_instance_types.restore.instance_types
}

output "instant_restore_instance_types" {
  description = "Instance types that support instant restore"
  value       = keys(data.eon_restore_account_region_aws_instance_types.sized.instant_restore_mapping)
}

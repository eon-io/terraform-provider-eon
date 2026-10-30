# Example: List EC2 instance types available for restore in a subnet and region
data "eon_restore_account_region_aws_instance_types" "demo" {
  restore_account_id = eon_restore_account.aws_disaster_recovery.id
  region             = "us-east-1"
  subnet_id          = "subnet-0123456789abcdef0"
}

output "available_instance_types" {
  description = "EC2 instance types available for restore in the subnet"
  value       = data.eon_restore_account_region_aws_instance_types.demo.instance_types
}

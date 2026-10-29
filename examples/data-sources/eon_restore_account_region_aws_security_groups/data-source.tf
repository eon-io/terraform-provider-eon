# Example: List all security groups available to a restore account in a region
data "eon_restore_account_region_aws_security_groups" "demo" {
  restore_account_id = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  region             = "us-east-1"
}

# Example: Narrow security groups to a specific VPC and subnet
data "eon_restore_account_region_aws_security_groups" "scoped" {
  restore_account_id = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  region             = "us-east-1"
  vpc_id             = "vpc-0123456789abcdef0"
  subnet_id          = "subnet-0123456789abcdef0"
}

output "security_group_ids" {
  description = "IDs of security groups available for restore, usable in eon_restore_job's ec2_config.security_group_ids"
  value       = [for sg in data.eon_restore_account_region_aws_security_groups.demo.security_groups : sg.id]
}

# Example: List the security groups available to Eon in a restore account region
data "eon_restore_account_region_aws_security_groups" "demo" {
  restore_account_id = eon_restore_account.aws_disaster_recovery.id
  region             = "us-east-1"
}

# Example: Narrow security groups to a specific VPC
data "eon_restore_account_region_aws_security_groups" "vpc_scoped" {
  restore_account_id = eon_restore_account.aws_disaster_recovery.id
  region             = "us-east-1"
  vpc_id             = "vpc-0123456789abcdef0"
}

output "security_group_ids" {
  description = "IDs of the security groups available for restore in the region"
  value       = [for sg in data.eon_restore_account_region_aws_security_groups.demo.security_groups : sg.id]
}

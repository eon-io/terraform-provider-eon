# Example: List the security groups available to Eon in a restore account region
data "eon_restore_account_region_aws_security_groups" "all" {
  restore_account_id = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  region             = "us-east-1"
}

# Example: Narrow security groups to a single VPC
data "eon_restore_account_region_aws_security_groups" "vpc" {
  restore_account_id = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  region             = "us-east-1"
  vpc_id             = "vpc-0123456789abcdef0"
}

output "restore_security_group_ids" {
  description = "IDs of the security groups in the VPC"
  value       = [for sg in data.eon_restore_account_region_aws_security_groups.vpc.security_groups : sg.id]
}

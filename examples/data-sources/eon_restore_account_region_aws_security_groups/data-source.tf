# Example: List security groups available in a region of a restore account
data "eon_restore_account_region_aws_security_groups" "demo" {
  account_id = "11111111-1111-1111-1111-111111111111"
  region     = "us-east-1"
}

# Example: Narrow security groups to a specific VPC
data "eon_restore_account_region_aws_security_groups" "by_vpc" {
  account_id = "11111111-1111-1111-1111-111111111111"
  region     = "us-east-1"
  vpc_id     = "vpc-0123456789abcdef0"
}

output "first_security_group_id" {
  description = "First listed security group ID for the region"
  value       = try(data.eon_restore_account_region_aws_security_groups.demo.security_groups[0].id, null)
}

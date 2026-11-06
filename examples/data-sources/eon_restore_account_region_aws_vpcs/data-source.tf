# Example: List VPCs available in a region of a restore account
data "eon_restore_account_region_aws_vpcs" "demo" {
  account_id = "11111111-1111-1111-1111-111111111111"
  region     = "us-east-1"
}

# Example: Use the first VPC to narrow security groups
data "eon_restore_account_region_aws_security_groups" "by_vpc" {
  account_id = "11111111-1111-1111-1111-111111111111"
  region     = "us-east-1"
  vpc_id     = data.eon_restore_account_region_aws_vpcs.demo.vpcs[0].id
}

output "vpc_ids" {
  description = "VPC IDs available for restore in the region"
  value       = [for vpc in data.eon_restore_account_region_aws_vpcs.demo.vpcs : vpc.id]
}

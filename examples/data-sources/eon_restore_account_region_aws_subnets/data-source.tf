# Example: List subnets available in a region of a restore account
data "eon_restore_account_region_aws_subnets" "demo" {
  account_id = "11111111-1111-1111-1111-111111111111"
  region     = "us-east-1"
}

output "subnet_ids_by_vpc" {
  description = "Subnet IDs grouped by VPC ID"
  value = {
    for subnet in data.eon_restore_account_region_aws_subnets.demo.subnets : subnet.vpc => subnet.id...
  }
}

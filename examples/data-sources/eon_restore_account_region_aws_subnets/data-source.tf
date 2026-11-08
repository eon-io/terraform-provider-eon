# Example: List subnets available in a region of a restore account
data "eon_restore_account_region_aws_subnets" "demo" {
  account_id = "11111111-1111-1111-1111-111111111111"
  region     = "us-east-1"
}

output "first_subnet_id" {
  description = "First listed subnet ID for the region"
  value       = try(data.eon_restore_account_region_aws_subnets.demo.subnets[0].id, null)
}

output "subnet_ids_in_us_east_1a" {
  description = "Subnet IDs in the us-east-1a availability zone"
  value       = [for s in data.eon_restore_account_region_aws_subnets.demo.subnets : s.id if s.availability_zone == "us-east-1a"]
}

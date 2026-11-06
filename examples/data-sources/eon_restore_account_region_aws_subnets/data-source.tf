# Example: List subnets available in a region of a restore account
data "eon_restore_account_region_aws_subnets" "demo" {
  account_id = "11111111-1111-1111-1111-111111111111"
  region     = "us-east-1"
}

output "vpc_subnet_ids" {
  description = "Subnet IDs in a specific VPC"
  value = [
    for subnet in data.eon_restore_account_region_aws_subnets.demo.subnets : subnet.id
    if subnet.vpc == "vpc-0123456789abcdef0"
  ]
}

output "first_subnet_id" {
  description = "First listed subnet ID for the region"
  value       = try(data.eon_restore_account_region_aws_subnets.demo.subnets[0].id, null)
}

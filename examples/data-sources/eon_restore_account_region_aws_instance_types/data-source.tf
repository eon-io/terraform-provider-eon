# Example: List EC2 instance types available for restore in a subnet and region
data "eon_restore_account_region_aws_instance_types" "demo" {
  account_id = "11111111-1111-1111-1111-111111111111"
  region     = "us-east-1"
  subnet_id  = "subnet-0123456789abcdef0"
}

output "first_instance_type" {
  description = "First listed EC2 instance type available for restore"
  value       = try(data.eon_restore_account_region_aws_instance_types.demo.instance_types[0], null)
}

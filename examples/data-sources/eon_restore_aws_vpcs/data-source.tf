data "eon_restore_aws_vpcs" "example" {
  restore_account_id = "restore-account-id"
  region             = "us-east-1"
}

output "restore_aws_vpcs" {
  value = data.eon_restore_aws_vpcs.example.vpcs
}

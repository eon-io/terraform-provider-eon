data "eon_restore_aws_subnets" "example" {
  restore_account_id = "restore-account-id"
  region             = "us-east-1"
}

output "restore_aws_subnets" {
  value = data.eon_restore_aws_subnets.example.subnets
}

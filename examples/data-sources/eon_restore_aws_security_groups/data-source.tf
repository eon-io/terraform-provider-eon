data "eon_restore_aws_security_groups" "example" {
  restore_account_id = "restore-account-id"
  region             = "us-east-1"
  vpc_id             = "vpc-example"
  subnet_id          = "subnet-example"
}

output "restore_aws_security_groups" {
  value = data.eon_restore_aws_security_groups.example.security_groups
}

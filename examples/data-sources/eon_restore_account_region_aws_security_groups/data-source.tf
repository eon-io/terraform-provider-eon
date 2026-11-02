data "eon_restore_account_region_aws_security_groups" "destination" {
  restore_account_id = "00000000-0000-0000-0000-000000000000"
  region             = "us-east-1"
  vpc_id             = "vpc-0123456789abcdef0"
  subnet_id          = "subnet-0123456789abcdef0"
}

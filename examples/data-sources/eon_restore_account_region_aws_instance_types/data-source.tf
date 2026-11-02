data "eon_restore_account_region_aws_instance_types" "destination" {
  restore_account_id = "00000000-0000-0000-0000-000000000000"
  region             = "us-east-1"
  subnet_id          = "subnet-0123456789abcdef0"
  volume_count       = 2
  total_volume_bytes = 107374182400
}

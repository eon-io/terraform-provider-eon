data "eon_restore_aws_instance_types" "example" {
  restore_account_id = "restore-account-id"
  region             = "us-east-1"
  subnet_id          = "subnet-example"
  volume_count       = 2
  total_volume_bytes = 1073741824
}

output "restore_aws_instance_types" {
  value = data.eon_restore_aws_instance_types.example.instance_types
}

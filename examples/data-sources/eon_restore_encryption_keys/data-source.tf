data "eon_restore_encryption_keys" "example" {
  restore_account_id       = "restore-account-id"
  region                   = "us-east-1"
  include_aws_managed_keys = true
}

output "restore_encryption_keys" {
  value = data.eon_restore_encryption_keys.example.keys
}

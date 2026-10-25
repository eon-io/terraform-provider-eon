data "eon_restore_s3_buckets" "example" {
  restore_account_id = "restore-account-id"
}

output "restore_s3_buckets" {
  value = data.eon_restore_s3_buckets.example.buckets
}

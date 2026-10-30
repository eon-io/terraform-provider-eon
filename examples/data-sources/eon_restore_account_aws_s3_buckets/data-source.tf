# Example: List the S3 buckets available to Eon in a restore account
data "eon_restore_account_aws_s3_buckets" "demo" {
  restore_account_id = eon_restore_account.aws_disaster_recovery.id
}

output "s3_bucket_names" {
  description = "Names of the S3 buckets available for restore in the account"
  value       = [for bucket in data.eon_restore_account_aws_s3_buckets.demo.buckets : bucket.bucket_name]
}

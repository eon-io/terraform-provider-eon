# Example: List the S3 buckets available to Eon in a restore account
data "eon_restore_account_aws_s3_buckets" "restore" {
  restore_account_id = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}

output "restore_bucket_names" {
  description = "Names of the S3 buckets Eon can restore to"
  value       = [for b in data.eon_restore_account_aws_s3_buckets.restore.buckets : b.bucket_name]
}

output "us_east_1_restore_buckets" {
  description = "S3 buckets in us-east-1"
  value       = [for b in data.eon_restore_account_aws_s3_buckets.restore.buckets : b.bucket_name if b.region == "us-east-1"]
}

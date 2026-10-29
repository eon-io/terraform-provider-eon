# Example: List S3 buckets available to an AWS restore account
data "eon_restore_account_aws_s3_buckets" "demo" {
  restore_account_id = "e696c7f0-17c6-4d9b-b589-591293c00d36"
}

output "s3_bucket_names" {
  description = "Names of the S3 buckets available for restore"
  value       = [for b in data.eon_restore_account_aws_s3_buckets.demo.buckets : b.bucket_name]
}

output "first_bucket_name" {
  description = "First listed S3 bucket name, usable as an eon_restore_job destination"
  value       = try(data.eon_restore_account_aws_s3_buckets.demo.buckets[0].bucket_name, null)
}

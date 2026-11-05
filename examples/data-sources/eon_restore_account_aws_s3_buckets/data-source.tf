# Example: List S3 buckets available to Eon in a restore account
data "eon_restore_account_aws_s3_buckets" "demo" {
  account_id = "11111111-1111-1111-1111-111111111111"
}

output "first_bucket_name" {
  description = "First listed S3 bucket name for the restore account"
  value       = try(data.eon_restore_account_aws_s3_buckets.demo.buckets[0].bucket_name, null)
}

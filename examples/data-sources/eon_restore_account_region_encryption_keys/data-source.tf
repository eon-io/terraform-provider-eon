# Example: List customer-managed encryption keys available in a region of a restore account
data "eon_restore_account_region_encryption_keys" "demo" {
  account_id = "11111111-1111-1111-1111-111111111111"
  region     = "us-east-1"
}

# Example: Include AWS-managed keys in the result
data "eon_restore_account_region_encryption_keys" "with_aws_managed" {
  account_id               = "11111111-1111-1111-1111-111111111111"
  region                   = "us-east-1"
  include_aws_managed_keys = true
}

output "first_encryption_key_arn" {
  description = "First listed encryption key ARN for the region"
  value       = try(data.eon_restore_account_region_encryption_keys.demo.keys[0].arn, null)
}

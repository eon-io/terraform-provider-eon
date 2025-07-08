# Example: Connect an AWS restore account for disaster recovery
resource "eon_restore_account" "aws_disaster_recovery" {
  name                = "Disaster Recovery AWS Account"
  cloud_provider      = "AWS"
  provider_account_id = "555666777888"
  role                = "arn:aws:iam::555666777888:role/EonRestoreRole"
  external_id         = "unique-restore-external-id-456" # Optional
}

# Example: Connect an AWS restore account for testing
resource "eon_restore_account" "aws_test_restore" {
  name                = "Test Restore AWS Account"
  cloud_provider      = "AWS"
  provider_account_id = "111222333444"
  role                = "arn:aws:iam::111222333444:role/EonTestRestoreRole"
}

# Output the account details
output "aws_disaster_recovery_account" {
  description = "Details of the connected AWS disaster recovery restore account"
  value = {
    id                  = eon_restore_account.aws_disaster_recovery.id
    name                = eon_restore_account.aws_disaster_recovery.name
    status              = eon_restore_account.aws_disaster_recovery.status
    provider_account_id = eon_restore_account.aws_disaster_recovery.provider_account_id
    cloud_provider      = eon_restore_account.aws_disaster_recovery.cloud_provider
    created_at          = eon_restore_account.aws_disaster_recovery.created_at
  }
}

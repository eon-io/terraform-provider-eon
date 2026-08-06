# Configure CloudWatch metrics for a restore account
resource "eon_restore_account_metrics_config" "aws" {
  restore_account_id = eon_restore_account.aws_disaster_recovery.id

  aws {
    region = "us-east-1"
  }
}

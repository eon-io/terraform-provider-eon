# Configure CloudWatch metrics for a source account
resource "eon_source_account_metrics_config" "aws" {
  source_account_id = eon_source_account.production.id

  aws {
    region = "us-east-1"
  }
}

# The metrics config attaches to a source account
resource "eon_source_account" "production" {
  name           = "Production AWS Account"
  cloud_provider = "AWS"

  aws {
    role_arn = "arn:aws:iam::123456789012:role/EonBackupRole"
  }
}

# Configure CloudWatch metrics for a source account
resource "eon_source_account_metrics_config" "aws" {
  source_account_id = eon_source_account.production.id

  aws {
    region = "us-east-1"
  }
}

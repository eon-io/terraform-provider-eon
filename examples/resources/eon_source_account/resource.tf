# Example: Connect an AWS source account for backup operations
resource "eon_source_account" "aws_production" {
  name                = "Production AWS Account"
  cloud_provider      = "AWS"
  provider_account_id = "123456789012"
  role                = "arn:aws:iam::123456789012:role/EonBackupRole"
  external_id         = "unique-external-id-123" # Optional
}

# Example: Connect an AWS source account without external ID
resource "eon_source_account" "aws_staging" {
  name                = "Staging AWS Account"
  cloud_provider      = "AWS"
  provider_account_id = "987654321098"
  role                = "arn:aws:iam::987654321098:role/EonBackupRole"
}

# Output the account details
output "aws_production_account" {
  description = "Details of the connected AWS production source account"
  value = {
    id                  = eon_source_account.aws_production.id
    name                = eon_source_account.aws_production.name
    status              = eon_source_account.aws_production.status
    provider_account_id = eon_source_account.aws_production.provider_account_id
    cloud_provider      = eon_source_account.aws_production.cloud_provider
    created_at          = eon_source_account.aws_production.created_at
  }
}

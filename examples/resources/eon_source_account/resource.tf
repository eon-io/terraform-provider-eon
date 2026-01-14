# Example: Connect an AWS source account using the new aws block (recommended)
resource "eon_source_account" "aws_production" {
  name           = "Production AWS Account"
  cloud_provider = "AWS"

  aws {
    role_arn = "arn:aws:iam::123456789012:role/EonBackupRole"
  }
}

# Example: Connect an Azure source account (subscription)
resource "eon_source_account" "azure_subscription" {
  name           = "Production Azure Subscription"
  cloud_provider = "AZURE"

  azure {
    tenant_id       = "00000000-0000-0000-0000-000000000000"
    subscription_id = "11111111-1111-1111-1111-111111111111"
  }
}

# Example: Connect a GCP source account (project)
resource "eon_source_account" "gcp_production" {
  name           = "Production GCP Project"
  cloud_provider = "GCP"

  gcp {
    project_id      = "my-gcp-project-id"
    service_account = "eon-backup@my-gcp-project-id.iam.gserviceaccount.com"
  }
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
  }
}

output "azure_subscription_account" {
  description = "Details of the connected Azure source account"
  value = {
    id                  = eon_source_account.azure_subscription.id
    name                = eon_source_account.azure_subscription.name
    status              = eon_source_account.azure_subscription.status
    provider_account_id = eon_source_account.azure_subscription.provider_account_id
    cloud_provider      = eon_source_account.azure_subscription.cloud_provider
  }
}

output "gcp_production_account" {
  description = "Details of the connected GCP source account"
  value = {
    id                  = eon_source_account.gcp_production.id
    name                = eon_source_account.gcp_production.name
    status              = eon_source_account.gcp_production.status
    provider_account_id = eon_source_account.gcp_production.provider_account_id
    cloud_provider      = eon_source_account.gcp_production.cloud_provider
  }
}

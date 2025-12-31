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
    tenant_id       = "ae5f2819-f24d-4e4b-990e-0e24fd4c5682"
    subscription_id = "cbb5ec02-4c52-4c6e-b262-d1c63effae51"
  }
}

# Example: Connect an Azure source account with resource group scoping
resource "eon_source_account" "azure_scoped" {
  name           = "Azure Scoped to Resource Group"
  cloud_provider = "AZURE"

  azure {
    tenant_id           = "ae5f2819-f24d-4e4b-990e-0e24fd4c5682"
    subscription_id     = "cbb5ec02-4c52-4c6e-b262-d1c63effae51"
    resource_group_name = "my-backup-resources"
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

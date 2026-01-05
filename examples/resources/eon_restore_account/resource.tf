# Example: Connect an AWS restore account using the new aws block (recommended)
resource "eon_restore_account" "aws_disaster_recovery" {
  name           = "Disaster Recovery AWS Account"
  cloud_provider = "AWS"

  aws {
    role_arn = "arn:aws:iam::555666777888:role/EonRestoreRole"
  }
}

# Example: Connect an Azure restore account (subscription)
resource "eon_restore_account" "azure_subscription" {
  name           = "Disaster Recovery Azure Subscription"
  cloud_provider = "AZURE"

  azure {
    tenant_id       = "ae5f2819-f24d-4e4b-990e-0e24fd4c5682"
    subscription_id = "cbb5ec02-4c52-4c6e-b262-d1c63effae51"
  }
}

# Example: Connect an Azure restore account with resource group scoping
resource "eon_restore_account" "azure_scoped" {
  name           = "Azure Restore to Specific RG"
  cloud_provider = "AZURE"

  azure {
    tenant_id           = "ae5f2819-f24d-4e4b-990e-0e24fd4c5682"
    subscription_id     = "cbb5ec02-4c52-4c6e-b262-d1c63effae51"
    resource_group_name = "my-restore-resources"
  }
}

# Example: Connect a GCP restore account (project)
resource "eon_restore_account" "gcp_disaster_recovery" {
  name           = "Disaster Recovery GCP Project"
  cloud_provider = "GCP"

  gcp {
    project_id      = "my-gcp-project-id"
    service_account = "eon-restore@my-gcp-project-id.iam.gserviceaccount.com"
  }
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
  }
}

output "azure_restore_account" {
  description = "Details of the connected Azure restore account"
  value = {
    id                  = eon_restore_account.azure_subscription.id
    name                = eon_restore_account.azure_subscription.name
    status              = eon_restore_account.azure_subscription.status
    provider_account_id = eon_restore_account.azure_subscription.provider_account_id
    cloud_provider      = eon_restore_account.azure_subscription.cloud_provider
  }
}

output "gcp_restore_account" {
  description = "Details of the connected GCP restore account"
  value = {
    id                  = eon_restore_account.gcp_disaster_recovery.id
    name                = eon_restore_account.gcp_disaster_recovery.name
    status              = eon_restore_account.gcp_disaster_recovery.status
    provider_account_id = eon_restore_account.gcp_disaster_recovery.provider_account_id
    cloud_provider      = eon_restore_account.gcp_disaster_recovery.cloud_provider
  }
}

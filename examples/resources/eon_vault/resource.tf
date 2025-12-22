# Example: Create a basic AWS vault with Eon-managed encryption
resource "eon_vault" "production_us_east" {
  name           = "Production US East 1"
  region         = "us-east-1"
  cloud_provider = "AWS"
}

# Example: Create an AWS vault with customer-managed KMS key
resource "eon_vault" "production_eu_central_cmk" {
  name            = "Production EU Central with CMK"
  region          = "eu-central-1"
  cloud_provider  = "AWS"
  aws_kms_key_arn = "arn:aws:kms:eu-central-1:123456789012:key/12345678-1234-1234-1234-123456789012"
}

# Example: Create vaults in multiple regions using for_each
resource "eon_vault" "multi_region" {
  for_each = toset(["us-east-1", "us-west-2", "eu-west-1"])

  name           = "Production ${each.value}"
  region         = each.value
  cloud_provider = "AWS"
}

# Output: Vault details
output "production_vault_id" {
  description = "ID of the production vault"
  value       = eon_vault.production_us_east.id
}

output "cmk_vault_details" {
  description = "Complete details of the CMK-encrypted vault"
  value = {
    id                  = eon_vault.production_eu_central_cmk.id
    name                = eon_vault.production_eu_central_cmk.name
    region              = eon_vault.production_eu_central_cmk.region
    vault_account_id    = eon_vault.production_eu_central_cmk.vault_account_id
    provider_account_id = eon_vault.production_eu_central_cmk.provider_account_id
    is_managed_by_eon   = eon_vault.production_eu_central_cmk.is_managed_by_eon
  }
}

output "multi_region_vault_ids" {
  description = "Map of region to vault ID for multi-region vaults"
  value = {
    for region, vault in eon_vault.multi_region :
    region => vault.id
  }
}

# IMPORTANT: Vaults are permanent and cannot be deleted.
# Running 'terraform destroy' will only remove vaults from Terraform state.
# The actual vaults will continue to exist in Eon permanently.
#
# IDEMPOTENT CREATION:
# If you try to create a vault that already exists (for example, after terraform destroy),
# the provider automatically imports it if the configuration matches.
# You'll see a warning message confirming the automatic import.
#
# This makes vault management safe and prevents errors when re-applying configurations.


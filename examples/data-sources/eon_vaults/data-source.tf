# List all vaults in the project
data "eon_vaults" "all" {}

# Output vault information
output "vault_count" {
  description = "Total number of vaults in the project"
  value       = length(data.eon_vaults.all.vaults)
}

output "vault_names" {
  description = "Names of all vaults"
  value       = [for v in data.eon_vaults.all.vaults : v.name]
}

output "vaults_by_region" {
  description = "Map of regions to vault names"
  value = {
    for v in data.eon_vaults.all.vaults :
    v.region => v.name...
  }
}

output "cmk_enabled_vaults" {
  description = "Vaults using customer-managed KMS keys"
  value = [
    for v in data.eon_vaults.all.vaults :
    {
      name    = v.name
      region  = v.region
      kms_arn = v.aws_kms_key_arn
    }
    if v.aws_kms_key_arn != null && v.aws_kms_key_arn != ""
  ]
}


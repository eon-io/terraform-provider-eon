# Example: Retrieve all restore accounts
data "eon_restore_accounts" "all" {}

# Example: Output all restore accounts information
output "all_restore_accounts" {
  description = "Information about all connected restore accounts"
  value = {
    total_accounts = length(data.eon_restore_accounts.all.accounts)
    accounts       = data.eon_restore_accounts.all.accounts
  }
}

# Example: Filter restore accounts using locals
locals {
  aws_restore_accounts = [
    for account in data.eon_restore_accounts.all.accounts :
    account if account.provider == "AWS"
  ]
  
  connected_restore_accounts = [
    for account in data.eon_restore_accounts.all.accounts :
    account if account.status == "CONNECTED"
  ]
}

# Output filtered results
output "aws_restore_accounts" {
  description = "List of AWS restore accounts only"
  value       = local.aws_restore_accounts
}

output "connected_restore_accounts" {
  description = "List of connected restore accounts only"
  value       = local.connected_restore_accounts
}

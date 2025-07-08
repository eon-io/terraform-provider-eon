# Example: Retrieve all source accounts
data "eon_source_accounts" "all" {}

# Example: Output all source accounts information
output "all_source_accounts" {
  description = "Information about all connected source accounts"
  value = {
    total_accounts = length(data.eon_source_accounts.all.accounts)
    accounts       = data.eon_source_accounts.all.accounts
  }
}

# Example: Filter AWS source accounts using locals
locals {
  aws_source_accounts = [
    for account in data.eon_source_accounts.all.accounts :
    account if account.provider == "AWS"
  ]

  connected_source_accounts = [
    for account in data.eon_source_accounts.all.accounts :
    account if account.status == "CONNECTED"
  ]
}

# Output filtered results
output "aws_source_accounts" {
  description = "List of AWS source accounts only"
  value       = local.aws_source_accounts
}

output "connected_source_accounts" {
  description = "List of connected source accounts only"
  value       = local.connected_source_accounts
}

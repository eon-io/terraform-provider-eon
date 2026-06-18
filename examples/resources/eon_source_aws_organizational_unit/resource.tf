# Connecting an OU registers it with Eon: Eon then DISCOVERS the member accounts
# and ASSUMES `EonSourceAccountRole` in each. It does NOT create that role —
# you must deploy it into every member account separately (a CloudFormation
# StackSet that auto-deploys to the OU). Eon ships this as `aws-organization.yml`
# (CloudFormation) and as the `source-account-org` Terraform onboarding module.
# Without it, members are discovered but have no permissions.
resource "eon_source_aws_organizational_unit" "production" {
  role_arn                        = "arn:aws:iam::123456789012:role/EonOrganizationAccountRole"
  provider_organizational_unit_id = "ou-abc1-23456789"
}

# Output the organizational unit details
output "production_ou" {
  description = "Details of the connected AWS production organizational unit"
  value = {
    id                              = eon_source_aws_organizational_unit.production.id
    name                            = eon_source_aws_organizational_unit.production.name
    status                          = eon_source_aws_organizational_unit.production.status
    provider_organizational_unit_id = eon_source_aws_organizational_unit.production.provider_organizational_unit_id
    provider_management_account_id  = eon_source_aws_organizational_unit.production.provider_management_account_id
    created_at                      = eon_source_aws_organizational_unit.production.created_at
  }
}

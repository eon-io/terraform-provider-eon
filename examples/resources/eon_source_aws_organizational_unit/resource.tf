# Example: Connect an AWS organizational unit for backup operations
resource "eon_source_aws_organizational_unit" "production" {
  role_arn                        = "arn:aws:iam::123456789012:role/EonOrgUnitRole"
  provider_organizational_unit_id = "ou-abc1-23456789"
}

# Example: Connect another AWS organizational unit
resource "eon_source_aws_organizational_unit" "staging" {
  role_arn                        = "arn:aws:iam::987654321098:role/EonOrgUnitRole"
  provider_organizational_unit_id = "ou-def2-98765432"
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

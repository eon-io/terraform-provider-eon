# Onboarding an AWS Organizational Unit is a TWO-part operation:
#
#   1. eon_source_aws_organizational_unit (below) registers the OU with Eon — Eon
#      then DISCOVERS the member accounts and ASSUMES `EonSourceAccountRole` in
#      each one.
#   2. That role must actually EXIST in every member account. Registration does
#      NOT create it; a CloudFormation StackSet does.
#
# Using this resource on its own leaves members discovered but with no
# permissions. For real onboarding use the `aws-ou-onboarding` module, which does
# BOTH parts (registration + the StackSet that creates the member roles, plus the
# management-account discovery role). See modules/aws-ou-onboarding/README.md:
#
#   module "eon_ou_onboarding" {
#     source = "github.com/eon-io/terraform-provider-eon//modules/aws-ou-onboarding"
#
#     eon_account_id          = "fde6adb5-38bc-45a3-919e-bd9ee17d9ba4"
#     scanning_account_id     = "388762879875"
#     organizational_unit_ids = ["ou-abc1-23456789", "ou-abc1-98765432"]
#   }

# Bare registration — only valid when EonSourceAccountRole already exists in every
# member account (e.g. deployed by the module above, a separate StackSet, or Eon's
# aws-organization.yml CloudFormation template).
resource "eon_source_aws_organizational_unit" "production" {
  role_arn                        = "arn:aws:iam::123456789012:role/EonOrgUnitRole"
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

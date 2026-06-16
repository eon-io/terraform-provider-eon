# Onboarding an AWS Organizational Unit to Eon is a TWO-part operation:
#
#   1. eon_source_aws_organizational_unit (below) registers the OU with Eon. Eon
#      then DISCOVERS every member account and ASSUMES a role named
#      `EonSourceAccountRole` in each one.
#   2. A role named `EonSourceAccountRole` must actually EXIST in every member
#      account. Registration does NOT create it. The service-managed
#      CloudFormation StackSet below creates it across the whole OU (and, via
#      auto-deployment, in any account that later joins the OU).
#
# Doing step 1 without step 2 leaves member accounts discovered but with no
# permissions. This mirrors Eon's official `aws-organization.yml` onboarding,
# which bundles the same StackSet.
#
# Prerequisite: enable CloudFormation trusted access with AWS Organizations once
# in the management account (`aws cloudformation activate-organizations-access`),
# otherwise the SERVICE_MANAGED StackSet cannot be created.

variable "eon_account_id" {
  type        = string
  description = "Eon-registered account ID (used as the STS ExternalId / confused-deputy guard)."
}

variable "organizational_unit_ids" {
  type        = set(string)
  description = "AWS Organizational Unit IDs to onboard (e.g. ou-abc1-23456789)."
}

variable "aws_region" {
  type        = string
  description = "Region to deploy the member-account roles in."
  default     = "us-east-1"
}

# (1) Discovery role in the management account: lets Eon enumerate the OU's
# accounts and assume EonSourceAccountRole in each member.
resource "aws_iam_role" "eon_org_unit" {
  name = "EonOrgUnitRole"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { AWS = ["arn:aws:iam::058264520728:root", "arn:aws:iam::010438478826:root"] }
      Action    = "sts:AssumeRole"
      Condition = { StringEquals = { "sts:ExternalId" = var.eon_account_id } }
    }]
  })

  inline_policy {
    name = "EonOrgUnitPolicy"
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Effect = "Allow"
          Action = [
            "organizations:ListAccountsForParent",
            "organizations:ListOrganizationalUnitsForParent",
            "organizations:DescribeOrganizationalUnit",
          ]
          Resource = "*"
        },
        {
          Effect   = "Allow"
          Action   = "sts:AssumeRole"
          Resource = "arn:aws:iam::*:role/EonSourceAccountRole"
        },
      ]
    })
  }
}

# (2) Create EonSourceAccountRole in every member account of the OU.
resource "aws_cloudformation_stack_set" "eon_source_members" {
  name             = "EonSourceAccountOrgDeployment"
  permission_model = "SERVICE_MANAGED"
  capabilities     = ["CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"]
  template_url     = "https://eon-public-b2b628cc-1d96-4fda-8dae-c3b1ad3ea03b.s3.amazonaws.com/source-account.yml"

  auto_deployment {
    enabled                          = true
    retain_stacks_on_account_removal = false
  }

  managed_execution { active = true }

  parameters = {
    EonAccountId = var.eon_account_id
    RoleName     = "EonSourceAccountRole"
    # source-account.yml exposes EnableS3CdcBackup, EnableEKS, EnableAwsBackup, …
    # capability flags — set them here as needed; sensible defaults apply otherwise.
  }

  lifecycle { ignore_changes = [administration_role_arn] }
}

resource "aws_cloudformation_stack_set_instance" "eon_source_members" {
  stack_set_name = aws_cloudformation_stack_set.eon_source_members.name

  deployment_targets {
    organizational_unit_ids = var.organizational_unit_ids
  }

  region = var.aws_region
}

# (1, cont.) Register each OU with Eon, only after member roles are being deployed.
resource "eon_source_aws_organizational_unit" "this" {
  for_each = var.organizational_unit_ids

  role_arn                        = aws_iam_role.eon_org_unit.arn
  provider_organizational_unit_id = each.value

  depends_on = [aws_cloudformation_stack_set_instance.eon_source_members]
}

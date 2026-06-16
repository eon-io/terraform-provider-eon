# Complete AWS Organizational Unit onboarding for Eon.
#
# Onboarding an OU is a TWO-part operation and this module does both:
#
#   1. Register each OU with Eon (eon_source_aws_organizational_unit). Eon then
#      DISCOVERS the member accounts and ASSUMES `source_role_name` in each.
#   2. CREATE that role in every member account, via a service-managed
#      CloudFormation StackSet that auto-deploys to the OU (and to accounts that
#      later join it).
#
# Doing only (1) — which is all the bare eon_source_aws_organizational_unit
# resource does — leaves members discovered but with no permissions. This module
# is the Terraform equivalent of Eon's official aws-organization.yml onboarding.
#
# PREREQUISITE (one-time, in the management account): enable CloudFormation
# trusted access with AWS Organizations, or the SERVICE_MANAGED StackSet cannot
# be created:
#   aws cloudformation activate-organizations-access

locals {
  create_discovery_role = var.discovery_role_arn == null
  discovery_role_arn    = local.create_discovery_role ? aws_iam_role.discovery[0].arn : var.discovery_role_arn

  permissions_boundary = trimspace(var.permissions_boundary_name)

  base_parameters = {
    EonAccountId            = var.eon_account_id
    RoleName                = var.source_role_name
    ServiceAccountId        = var.service_account_id
    ServiceDRAccountId      = var.service_dr_account_id
    ScanningAccountId       = var.scanning_account_id
    PermissionsBoundaryName = local.permissions_boundary
  }

  # Caller-supplied capability flags (EnableEKS, EnableS3CdcBackup, …) win over
  # the base set; anything omitted falls back to the template's own defaults.
  stackset_parameters = merge(local.base_parameters, var.capability_flags)
}

# (1) Management-account discovery role — created only when the caller did not
# pass an existing discovery_role_arn.
resource "aws_iam_role" "discovery" {
  count = local.create_discovery_role ? 1 : 0
  name  = var.discovery_role_name

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { AWS = [var.service_account_id, var.service_dr_account_id] }
        Action    = "sts:AssumeRole"
        Condition = { StringEquals = { "sts:ExternalId" = var.eon_account_id } }
      },
      {
        Effect    = "Allow"
        Principal = { AWS = ["arn:aws:iam::${var.service_account_id}:root", "arn:aws:iam::${var.service_dr_account_id}:root"] }
        Action    = "sts:TagSession"
      },
    ]
  })

  inline_policy {
    name = "${var.discovery_role_name}Policy"
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          # Enumerate the OU hierarchy and member accounts.
          Effect = "Allow"
          Action = [
            "organizations:ListAccounts",
            "organizations:ListAccountsForParent",
            "organizations:DescribeOrganizationalUnit",
            "organizations:ListChildren",
            "organizations:DescribeAccount",
            "organizations:ListParents",
            "organizations:ListOrganizationalUnitsForParent",
          ]
          Resource = "*"
        },
        {
          # Assume the per-member source role. Role name is constrained; the
          # account segment is a wildcard because the OU may hold any accounts.
          Effect   = "Allow"
          Action   = "sts:AssumeRole"
          Resource = "arn:aws:iam::*:role/${var.source_role_name}"
        },
      ]
    })
  }
}

# (2) Create source_role_name in every member account of the OUs.
resource "aws_cloudformation_stack_set" "source_members" {
  name             = "EonSourceAccountOrgDeployment"
  description      = "Deploys ${var.source_role_name} into Eon-onboarded OU member accounts"
  permission_model = "SERVICE_MANAGED"
  capabilities     = ["CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"]
  template_url     = var.source_account_template_url

  auto_deployment {
    enabled                          = true
    retain_stacks_on_account_removal = false
  }

  managed_execution { active = true }

  parameters = local.stackset_parameters

  lifecycle { ignore_changes = [administration_role_arn] }
}

resource "aws_cloudformation_stack_set_instance" "source_members" {
  stack_set_name = aws_cloudformation_stack_set.source_members.name

  deployment_targets {
    organizational_unit_ids = var.organizational_unit_ids
  }

  region = var.aws_region
}

# (1, cont.) Register each OU with Eon, only after member roles are deploying, so
# discovery doesn't transiently report missing permissions.
resource "eon_source_aws_organizational_unit" "this" {
  for_each = var.organizational_unit_ids

  role_arn                        = local.discovery_role_arn
  provider_organizational_unit_id = each.value

  depends_on = [aws_cloudformation_stack_set_instance.source_members]
}

variable "eon_account_id" {
  type        = string
  description = "Eon-registered account ID, used as the STS ExternalId (confused-deputy guard) on every Eon role."
}

variable "organizational_unit_ids" {
  type        = set(string)
  description = "AWS Organizational Unit IDs to onboard (e.g. ou-abc1-23456789). All current and future member accounts in each OU (and nested OUs) are covered."

  validation {
    condition     = length(var.organizational_unit_ids) > 0
    error_message = "Provide at least one organizational unit ID."
  }
}

variable "aws_region" {
  type        = string
  description = "Region in which the member-account roles are created by the StackSet."
  default     = "us-east-1"
}

variable "discovery_role_arn" {
  type        = string
  description = "ARN of an existing management-account role Eon assumes to enumerate the OU. Leave null to have this module create EonOrgUnitRole for you."
  default     = null
}

variable "discovery_role_name" {
  type        = string
  description = "Name of the management-account discovery role this module creates when discovery_role_arn is null."
  default     = "EonOrgUnitRole"
}

variable "source_role_name" {
  type        = string
  description = "Name of the per-member-account role Eon assumes to back up resources. Must match the role the StackSet creates."
  default     = "EonSourceAccountRole"
}

variable "service_account_id" {
  type        = string
  description = "Eon service account ID (trusted principal on the roles)."
  default     = "058264520728"
}

variable "service_dr_account_id" {
  type        = string
  description = "Eon service DR account ID (trusted principal on the roles)."
  default     = "010438478826"
}

variable "scanning_account_id" {
  type        = string
  description = "Eon scanning account ID (12 digits)."

  validation {
    condition     = can(regex("^[0-9]{12}$", var.scanning_account_id))
    error_message = "scanning_account_id must be a 12-digit AWS account ID."
  }
}

variable "permissions_boundary_name" {
  type        = string
  description = "Optional IAM permissions boundary policy name applied to the member roles. Must exist with this exact name in every target account. Empty string disables it."
  default     = ""
}

variable "source_account_template_url" {
  type        = string
  description = "URL of the Eon-hosted source-account CloudFormation template the StackSet deploys into each member account."
  default     = "https://eon-public-b2b628cc-1d96-4fda-8dae-c3b1ad3ea03b.s3.amazonaws.com/source-account.yml"
}

variable "capability_flags" {
  type        = map(string)
  description = "Optional overrides for source-account.yml capability parameters (e.g. { EnableEKS = \"false\" }). Anything omitted uses the template default. Values must be the strings \"true\"/\"false\"."
  default     = {}
}

# aws-ou-onboarding module

Onboards one or more AWS Organizational Units to Eon **completely** — both halves
of the operation, so member accounts actually end up with permissions.

Connecting an OU to Eon only makes Eon **discover** the member accounts and
**assume** a role (`EonSourceAccountRole`) in each. It does **not** create that
role. The bare `eon_source_aws_organizational_unit` resource does only the
registration; used on its own it leaves every member account discovered but
without an Eon role — the most common OU onboarding failure.

This module does both parts:

1. Registers each OU with Eon (`eon_source_aws_organizational_unit`).
2. Creates `EonSourceAccountRole` in every member account via a service-managed
   CloudFormation **StackSet** with auto-deployment, so current **and future**
   accounts in the OU are covered. (Optionally) creates the management-account
   discovery role too.

It is the Terraform equivalent of Eon's official `aws-organization.yml`
CloudFormation onboarding.

## Prerequisite

Enable CloudFormation trusted access with AWS Organizations once, in the
management account, or the `SERVICE_MANAGED` StackSet cannot be created:

```sh
aws cloudformation activate-organizations-access
```

(or Console → CloudFormation → StackSets → "Activate trusted access").

## Usage

```hcl
provider "aws" {
  region = "us-east-1"
  # credentials for the AWS Organization management account
}

provider "eon" {
  endpoint      = "https://<tenant>.console.eon.io"
  client_id     = var.eon_client_id
  client_secret = var.eon_client_secret
  project_id    = var.eon_project_id
}

module "eon_ou_onboarding" {
  source = "github.com/eon-io/terraform-provider-eon//modules/aws-ou-onboarding"

  eon_account_id      = "fde6adb5-38bc-45a3-919e-bd9ee17d9ba4"
  scanning_account_id = "388762879875"

  organizational_unit_ids = [
    "ou-abc1-23456789",
    "ou-abc1-98765432",
  ]

  # Optional: turn off capabilities you don't want the member role to grant.
  capability_flags = {
    EnableEKS = "false"
  }
}
```

If you already manage the management-account discovery role yourself, pass its
ARN and the module will skip creating one:

```hcl
  discovery_role_arn = aws_iam_role.my_existing_eon_org_role.arn
```

## What gets created

| Resource | Where | Purpose |
|----------|-------|---------|
| `aws_iam_role.discovery` (optional) | management account | role Eon assumes to enumerate the OU |
| `aws_cloudformation_stack_set.source_members` | management account | defines the member-role deployment |
| `aws_cloudformation_stack_set_instance.source_members` | OU members | creates `EonSourceAccountRole` in each account |
| `eon_source_aws_organizational_unit.this` | Eon | registers each OU for discovery |

## Inputs

See `variables.tf`. Required: `eon_account_id`, `scanning_account_id`,
`organizational_unit_ids`. Everything else has a sensible default.

## Notes

- The StackSet does not target the management account itself (service-managed
  StackSets only deploy to members) — correct, since Eon does not scan it.
- New accounts joining a registered OU automatically receive the role
  (`auto_deployment.enabled = true`); no re-apply needed.

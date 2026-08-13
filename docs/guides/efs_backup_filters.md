---
page_title: "AWS EFS Backup and Conditional Filtering Guide"
subcategory: ""
---

# AWS EFS Backup and Conditional Filtering Guide

This guide demonstrates how to configure backup policies for AWS EFS (Elastic
File System) resources using Eon's conditional resource filtering.

Eon lets you write fine-grained policies with `resource_selector` expressions so
you only back up EFS file systems that match specific criteria (such as
environment tags or AWS regions), instead of backing up everything.

## How Filtering Works

When you set `resource_selection_mode = "CONDITIONAL"`, you define an
`expression` block. This block can evaluate:

- **`resource_type`**: Set to `AWS_EFS` to target EFS resources.
- **`tag_key_values`**: Filter on specific AWS tags (e.g. `Tier = Critical`).
- **`source_region`**: Restrict backup to certain AWS regions.

These conditions combine using `group` and logical operators (`AND` / `OR`).

## Provider Setup

Configure the Eon provider. Supply credentials via environment variables
(`EON_ENDPOINT`, `EON_CLIENT_ID`, `EON_CLIENT_SECRET`, `EON_PROJECT_ID`) rather
than hardcoding them.

```terraform
terraform {
  required_providers {
    eon = {
      source  = "eon-io/eon"
      version = "~> 1.0"
    }
  }
}

provider "eon" {
}
```

## Backup Policy

This example targets `AWS_EFS` resources in `us-east-1` that are tagged
`Tier = Critical`. The destination vault is resolved dynamically with the
`eon_vaults` data source, so no vault UUID is hardcoded.

```terraform
data "eon_vaults" "all" {}

locals {
  efs_vault_id = one([
    for v in data.eon_vaults.all.vaults :
    v.id if v.cloud_provider == "AWS" && v.region == "us-east-1"
  ])
}

resource "eon_backup_policy" "efs_conditional_backup" {
  name    = "Production EFS Backup - Critical Tier"
  enabled = true

  resource_selector = {
    resource_selection_mode = "CONDITIONAL"
    expression = {
      group = {
        operator = "AND"
        operands = [
          {
            resource_type = {
              operator       = "IN"
              resource_types = ["AWS_EFS"]
            }
          },
          {
            tag_key_values = {
              operator = "CONTAINS_ANY_OF"
              tag_key_values = [
                { key = "Tier", value = "Critical" }
              ]
            }
          },
          {
            source_region = {
              operator       = "IN"
              source_regions = ["us-east-1"]
            }
          }
        ]
      }
    }
  }

  backup_plan = {
    backup_policy_type = "STANDARD"
    standard_plan = {
      backup_schedules = [
        {
          vault_id       = local.efs_vault_id
          retention_days = 60
          schedule_config = {
            frequency = "DAILY"
            daily_config = {
              time_of_day_hour     = 2
              time_of_day_minutes  = 0
              start_window_minutes = 240
            }
          }
        }
      ]
    }
  }
}
```

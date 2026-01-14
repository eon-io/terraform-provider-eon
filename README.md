# Terraform Provider for Eon

The Terraform provider for Eon allows you to manage your cloud backup and restore infrastructure using Infrastructure as Code (IaC). Connect cloud accounts, manage backup policies, and orchestrate disaster recovery workflows with Terraform.

## Features

- **Source Account Management**: Connect and manage cloud accounts containing resources to be backed up
- **Restore Account Management**: Connect and manage cloud accounts where backups can be restored
- **Backup Policy Management**: Create, update, and manage backup policies with schedules, retention, and notifications
- **Multi-Cloud Support**: AWS, Azure, and GCP
- **Data Sources**: Query existing source and restore accounts, backup policies, and snapshots

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22 (for development)

## Installation

### Terraform Registry (Recommended)

```hcl
terraform {
  required_providers {
    eon = {
      source  = "eon-io/eon"
      version = "~> 1.0"
    }
  }
}
```

### Manual Installation

1. Download the latest release from the [releases page](https://github.com/eon-io/terraform-provider-eon/releases)
2. Extract the binary to your Terraform plugins directory
3. Run `terraform init`

## Authentication

The provider supports authentication via OAuth2 client credentials. You can configure authentication in several ways:

### Environment Variables (Recommended)

```bash
export EON_ENDPOINT="https://your-eon-instance.eon.io"
export EON_CLIENT_ID="your-client-id"
export EON_CLIENT_SECRET="your-client-secret"
export EON_PROJECT_ID="your-project-id"
```

### Provider Configuration

```hcl
provider "eon" {
  endpoint      = "https://your-eon-instance.eon.io"
  client_id     = "your-client-id"
  client_secret = "your-client-secret"
  project_id    = "your-project-id"
}
```

## Usage Examples

### AWS Source Account

```hcl
# Connect an AWS source account
resource "eon_source_account" "aws_production" {
  name           = "Production AWS Account"
  cloud_provider = "AWS"

  aws {
    role_arn = "arn:aws:iam::123456789012:role/EonBackupRole"
  }
}
```

### Azure Source Account

```hcl
# Connect an Azure source account (subscription)
resource "eon_source_account" "azure_subscription" {
  name           = "Production Azure Subscription"
  cloud_provider = "AZURE"

  azure {
    tenant_id       = "00000000-0000-0000-0000-000000000000"
    subscription_id = "11111111-1111-1111-1111-111111111111"
  }
}
```

### AWS Restore Account

```hcl
# Connect an AWS restore account
resource "eon_restore_account" "aws_disaster_recovery" {
  name           = "Disaster Recovery AWS Account"
  cloud_provider = "AWS"

  aws {
    role_arn = "arn:aws:iam::987654321098:role/EonRestoreRole"
  }
}
```

### Azure Restore Account

```hcl
# Connect an Azure restore account (subscription)
resource "eon_restore_account" "azure_disaster_recovery" {
  name           = "Disaster Recovery Azure Subscription"
  cloud_provider = "AZURE"

  azure {
    tenant_id       = "00000000-0000-0000-0000-000000000000"
    subscription_id = "11111111-1111-1111-1111-111111111111"
  }
}
```

### GCP Source Account

```hcl
# Connect a GCP source account (project)
resource "eon_source_account" "gcp_production" {
  name           = "Production GCP Project"
  cloud_provider = "GCP"

  gcp {
    project_id      = "my-gcp-project-id"
    service_account = "eon-backup@my-gcp-project-id.iam.gserviceaccount.com"
  }
}
```

### GCP Restore Account

```hcl
# Connect a GCP restore account (project)
resource "eon_restore_account" "gcp_disaster_recovery" {
  name           = "Disaster Recovery GCP Project"
  cloud_provider = "GCP"

  gcp {
    project_id      = "my-dr-gcp-project-id"
    service_account = "eon-restore@my-dr-gcp-project-id.iam.gserviceaccount.com"
  }
}
```

### Basic Backup Policy

```hcl
# Create a backup policy
resource "eon_backup_policy" "all_resources" {
  name         = "All Resources Policy"
  enabled      = true
  resource_selector = {
    resource_selection_mode = "ALL"
  }

  backup_plan = {
    backup_policy_type = "STANDARD"
    standard_plan = {
      backup_schedules = [
        {
          vault_id = "vault-all"
          retention_days = 90
          schedule_config = {
            frequency = "DAILY"
            daily_config = {
              time_of_day_hour = 1
              time_of_day_minutes = 0
              start_window_minutes = 360
            }
          }
        }
      ]
    }
  }
}
```

### Data Sources

```hcl
# List all source accounts
data "eon_source_accounts" "all" {}

# List all restore accounts
data "eon_restore_accounts" "all" {}

# List all backup policies
data "eon_backup_policies" "all" {}

output "source_account_count" {
  value = length(data.eon_source_accounts.all.accounts)
}

output "backup_policy_count" {
  value = length(data.eon_backup_policies.all.policies)
}
```

## Resources

### `eon_source_account`

Manages source accounts for backup operations.

**Arguments:**

- `name` (Required) - Display name for the source account
- `cloud_provider` (Required) - Cloud provider: `AWS`, `AZURE`, or `GCP`

**Cloud Provider Blocks** (one required based on `cloud_provider`):

- `aws` block (for AWS accounts):
  - `role_arn` (Required) - ARN of the IAM role Eon assumes to access the account

- `azure` block (for Azure accounts):
  - `tenant_id` (Required) - Azure Active Directory tenant ID
  - `subscription_id` (Required) - Azure subscription ID

- `gcp` block (for GCP accounts):
  - `project_id` (Required) - GCP project ID
  - `service_account` (Required) - Email of the GCP service account Eon uses to access the project

**Attributes:**

- `id` - Source account identifier
- `provider_account_id` - Cloud provider account ID (computed)
- `status` - Connection status (`CONNECTED`, `DISCONNECTED`, `INSUFFICIENT_PERMISSIONS`)
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp

### `eon_restore_account`

Manages restore accounts for restore operations.

**Arguments:**

- `name` (Required) - Display name for the restore account
- `cloud_provider` (Required) - Cloud provider: `AWS`, `AZURE`, or `GCP`

**Cloud Provider Blocks** (one required based on `cloud_provider`):

- `aws` block (for AWS accounts):
  - `role_arn` (Required) - ARN of the IAM role Eon assumes to access the account

- `azure` block (for Azure accounts):
  - `tenant_id` (Required) - Azure Active Directory tenant ID
  - `subscription_id` (Required) - Azure subscription ID

- `gcp` block (for GCP accounts):
  - `project_id` (Required) - GCP project ID
  - `service_account` (Required) - Email of the GCP service account Eon uses to access the project

**Attributes:**

- `id` - Restore account identifier
- `provider_account_id` - Cloud provider account ID (computed)
- `status` - Connection status (`CONNECTED`, `DISCONNECTED`, `INSUFFICIENT_PERMISSIONS`)
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp

### `eon_backup_policy`

Manages backup policies for automated backup operations.

**Arguments:**

- `name` (Required) - Display name for the backup policy
- `enabled` (Required) - Whether the backup policy is enabled
- `resource_selection_mode` (Required) - Resource selection mode: 'ALL', 'NONE', or 'CONDITIONAL'
- `backup_policy_type` (Required) - Backup policy type: 'STANDARD', 'HIGH_FREQUENCY', or 'PITR'
- `vault_id` (Required) - Vault ID to associate with the backup policy
- `schedule_frequency` (Required) - Frequency for the backup schedule
- `retention_days` (Required) - Number of days to retain backups
- `resource_inclusion_override` (Optional) - List of resource IDs to include regardless of selection mode
- `resource_exclusion_override` (Optional) - List of resource IDs to exclude regardless of selection mode

**Attributes:**

- `id` - Backup policy identifier
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp

## Data Sources

### `eon_source_accounts`

Retrieves information about all source accounts.

**Attributes:**

- `accounts` - List of source account objects with `id`, `name`, `provider`, `provider_account_id`, and `status`

### `eon_restore_accounts`

Retrieves information about all restore accounts.

**Attributes:**

- `accounts` - List of restore account objects with `id`, `provider`, `provider_account_id`, `status`, and `regions`

### `eon_backup_policies`

Retrieves information about all backup policies.

**Attributes:**

- `policies` - List of backup policy objects with `id`, `name`, and `enabled`

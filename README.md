# Terraform Provider for Eon

The Terraform provider for Eon allows you to manage your cloud backup and restore infrastructure using Infrastructure as Code (IaC). Connect cloud accounts, manage backup policies, and orchestrate disaster recovery workflows with Terraform.

## Features

- **Source Account Management**: Connect and manage cloud accounts containing resources to be backed up
- **Restore Account Management**: Connect and manage cloud accounts where backups can be restored
- **Multi-Cloud Support**: AWS, Azure, and GCP (AWS fully supported, Azure and GCP in development)
- **Data Sources**: Query existing source and restore accounts

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
export EON_ACCOUNT_ID="your-account-id"
```

### Provider Configuration

```hcl
provider "eon" {
  endpoint       = "https://your-eon-instance.eon.io"
  client_id      = "your-client-id"
  client_secret  = "your-client-secret"
  project_id     = "your-project-id"
  eon_account_id = "your-account-id"
}
```

## Usage Examples

### Basic Source Account

```hcl
# Connect an AWS source account
resource "eon_source_account" "aws_production" {
  name                = "Production AWS Account"
  cloud_provider      = "AWS"
  provider_account_id = "123456789012"
  role               = "arn:aws:iam::123456789012:role/EonBackupRole"
}
```

### Basic Restore Account

```hcl
# Connect an AWS restore account
resource "eon_restore_account" "aws_disaster_recovery" {
  name                = "Disaster Recovery AWS Account"
  cloud_provider      = "AWS"
  provider_account_id = "987654321098"
  role               = "arn:aws:iam::987654321098:role/EonRestoreRole"
}
```

### Data Sources

```hcl
# List all source accounts
data "eon_source_accounts" "all" {}

# List all restore accounts
data "eon_restore_accounts" "all" {}

output "source_account_count" {
  value = length(data.eon_source_accounts.all.accounts)
}
```

## Resources

### `eon_source_account`

Manages source accounts for backup operations.

**Arguments:**
- `name` (Required) - Display name for the source account
- `cloud_provider` (Required) - Cloud provider (AWS, AZURE, GCP)
- `provider_account_id` (Required) - Cloud provider account ID
- `role` (Required) - IAM role ARN (AWS), service principal (Azure), or service account email (GCP)
- `external_id` (Optional) - External ID for AWS role assumption

**Attributes:**
- `id` - Source account identifier
- `status` - Connection status
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp

### `eon_restore_account`

Manages restore accounts for restore operations.

**Arguments:**
- `name` (Required) - Display name for the restore account
- `cloud_provider` (Required) - Cloud provider (AWS, AZURE, GCP)
- `provider_account_id` (Required) - Cloud provider account ID
- `role` (Required) - IAM role ARN (AWS), service principal (Azure), or service account email (GCP)
- `external_id` (Optional) - External ID for AWS role assumption

**Attributes:**
- `id` - Restore account identifier
- `status` - Connection status
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

## Development

### Building the Provider

```bash
git clone https://github.com/eon-io/terraform-provider-eon
cd terraform-provider-eon
go build -o terraform-provider-eon
```

### Running Tests

```bash
go test ./...
```

### Local Development

1. Build the provider: `go build -o terraform-provider-eon`
2. Create a `.terraformrc` file in your home directory:

```hcl
provider_installation {
  dev_overrides {
    "eon-io/eon" = "/path/to/terraform-provider-eon"
  }
  direct {}
}
```

3. Run `terraform init` and `terraform plan` in your test configuration

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

This project is licensed under the Mozilla Public License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- [Documentation](https://docs.eon.io)
- [GitHub Issues](https://github.com/eon-io/terraform-provider-eon/issues)
- [Community Forum](https://community.eon.io)

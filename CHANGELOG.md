# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2025-01-08

### Added
- Comprehensive examples for all resources and data sources
- Detailed documentation structure following HashiCorp standards
- Dependabot configuration for automated dependency updates
- Additional CI/CD workflows for linting, security scanning, and documentation
- golangci-lint configuration for enhanced code quality
- Contributing guide with detailed development workflows
- Templates for automated documentation generation
- Enhanced Makefile with comprehensive development targets
- Security scanning with gosec
- Terraform example validation in CI/CD

### Enhanced
- Provider documentation with detailed usage examples
- Resource and data source documentation with comprehensive guides
- Code quality standards and automated enforcement
- Development workflow automation

## [1.0.0] - 2025-01-08

### Added
- Initial release of Eon Terraform Provider
- Support for managing source and restore accounts
- AWS cloud account support (Azure and GCP in development)
- OAuth2 authentication with client credentials flow
- Data sources for querying existing accounts
- Comprehensive error handling and validation
- Clean provider implementation following HashiCorp best practices
- CI/CD workflows for testing and releasing
- Uses eon-sdk-go v1.3.1 for API communication

### Resources
- `eon_source_account` - Manages source accounts for backup operations
- `eon_restore_account` - Manages restore accounts for restore operations

### Data Sources
- `eon_source_accounts` - Retrieves information about source accounts
- `eon_restore_accounts` - Retrieves information about restore accounts

### Features
- Multi-cloud support framework (AWS fully implemented)
- Account connection and disconnection management
- Status monitoring and validation
- Import support for existing accounts
- Environment variable configuration support

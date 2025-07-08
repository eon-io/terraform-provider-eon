# Contributing to Terraform Provider for Eon

Thank you for your interest in contributing to the Terraform Provider for Eon! This document provides guidelines and information for contributors.

## Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) >= 1.23
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Git](https://git-scm.com/)

### Development Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/your-username/terraform-provider-eon.git
   cd terraform-provider-eon
   ```

3. Set up the development environment:
   ```bash
   make dev-setup
   ```

4. Build the provider:
   ```bash
   make build
   ```

5. Install the provider locally for testing:
   ```bash
   make install
   ```

## Development Workflow

### Making Changes

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the coding standards
3. Add or update tests as needed
4. Update documentation if necessary
5. Run the test suite:
   ```bash
   make check
   ```

### Testing

#### Unit Tests
```bash
make test
```

#### Acceptance Tests
```bash
make testacc
```

#### Test Coverage
```bash
make test-coverage
```

#### Validate Examples
```bash
make validate-examples
```

### Code Quality

#### Formatting
```bash
make fmt
```

#### Linting
```bash
make lint
```

#### Security Scan
```bash
make sec
```

### Documentation

Generate documentation:
```bash
make docs
```

The provider uses [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs) to generate documentation from the schema and examples.

## Coding Standards

### Go Code

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions and complex logic
- Handle errors appropriately
- Write unit tests for new functionality

### Terraform Code

- Format all Terraform code (`terraform fmt`)
- Use meaningful resource and variable names
- Add descriptions to variables and outputs
- Include examples for new resources and data sources

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `ci:` for CI/CD changes
- `test:` for test changes
- `refactor:` for code refactoring

Examples:
```
feat: add support for Azure source accounts
fix: handle authentication token expiry correctly
docs: update provider configuration examples
```

## Pull Request Process

1. Ensure your code passes all tests and checks:
   ```bash
   make check
   make testacc
   ```

2. Update documentation if needed
3. Create a pull request with:
   - Clear title and description
   - Reference to any related issues
   - Test results and validation steps

4. Respond to code review feedback
5. Ensure CI/CD checks pass

## License

By contributing to this project, you agree that your contributions will be licensed under the Mozilla Public License 2.0.

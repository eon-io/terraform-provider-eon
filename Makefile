# Terraform Provider Makefile

# Default target
.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build the provider
.PHONY: build
build: ## Build the provider binary
	go build -v ./...

# Install the provider locally
.PHONY: install
install: build ## Install the provider locally
	go install .

# Run tests
.PHONY: test
test: ## Run unit tests
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run acceptance tests (requires real credentials)
.PHONY: testacc
testacc: ## Run acceptance tests
	TF_ACC=1 go test -v ./... -timeout 120m

# Lint the code
.PHONY: lint
lint: ## Run linting
	golangci-lint run

# Format the code
.PHONY: fmt
fmt: ## Format Go code
	go fmt ./...
	terraform fmt -recursive ./examples/

# Security scan
.PHONY: security
security: ## Run security scan
	gosec ./...

# Validate examples
.PHONY: validate-examples
validate-examples: ## Validate Terraform examples
	@for dir in examples/*/; do \
		if [[ "$$(basename "$$dir")" == "provider" ]]; then \
			echo "Skipping provider directory"; \
			continue; \
		fi; \
		echo "Validating $$dir"; \
		cd "$$dir" && terraform init -backend=false && terraform validate && cd ../..; \
	done

# Clean build artifacts
.PHONY: clean
clean: ## Clean build artifacts
	rm -f terraform-provider-eon
	rm -f coverage.out coverage.html
	go clean

# Generate documentation
.PHONY: docs
docs: ## Generate documentation
	go generate ./...

# Development setup
.PHONY: dev-setup
dev-setup: ## Set up development environment
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Release (dry run)
.PHONY: release-dry-run
release-dry-run: ## Test release build without publishing
	goreleaser release --snapshot --clean --skip-sign

# Check if ready for release
.PHONY: release-check
release-check: lint test security validate-examples ## Run all checks before release
	@echo "All checks passed! Ready for release."

# Run all quality checks
.PHONY: check
check: fmt lint test security ## Run all quality checks

.PHONY: all
all: check build ## Run checks and build

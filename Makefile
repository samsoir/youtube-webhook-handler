# YouTube Webhook Project Makefile

# Variables
GO_VERSION := 1.23
FUNCTION_DIR := function
TERRAFORM_DIR := terraform
PROJECT_NAME := youtube-webhook
BINARY_NAME := webhook
CLI_BINARY := youtube-webhook

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# Default target
.DEFAULT_GOAL := help

.PHONY: help setup test test-cli test-all test-verbose test-coverage test-coverage-cli test-coverage-all clean lint fmt vet build deploy-function terraform-init terraform-plan terraform-apply terraform-destroy docker-build docker-run

help: ## Show this help message
	@echo "$(BLUE)YouTube Webhook Project$(NC)"
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

## Development Commands

setup: ## Setup development environment and download dependencies
	@echo "$(YELLOW)Setting up development environment...$(NC)"
	@cd $(FUNCTION_DIR) && go mod tidy
	@cd $(FUNCTION_DIR) && go mod download
	@echo "$(GREEN)✓ Go dependencies installed$(NC)"
	@echo "$(YELLOW)Verifying Terraform installation...$(NC)"
	@terraform version || (echo "$(RED)❌ Terraform not found. Please install Terraform.$(NC)" && exit 1)
	@echo "$(GREEN)✓ Development environment ready$(NC)"

## Go Commands

test: ## Run all tests
	@echo "$(YELLOW)Running tests...$(NC)"
	@cd $(FUNCTION_DIR) && go test -v ./...
	@echo "$(GREEN)✓ Function tests completed$(NC)"

test-cli: ## Run CLI tests
	@echo "$(YELLOW)Running CLI tests...$(NC)"
	@go test -v ./cli/...
	@go test -v ./cmd/youtube-webhook/...
	@echo "$(GREEN)✓ CLI tests completed$(NC)"

test-all: test test-cli ## Run all tests (function and CLI)
	@echo "$(GREEN)✓ All tests completed$(NC)"

test-verbose: ## Run tests with verbose output
	@echo "$(YELLOW)Running tests with verbose output...$(NC)"
	@cd $(FUNCTION_DIR) && go test -v -race ./...

test-coverage: ## Run tests with coverage report
	@echo "$(YELLOW)Running function tests with coverage...$(NC)"
	@cd $(FUNCTION_DIR) && go test -v -race -coverprofile=coverage.out ./...
	@cd $(FUNCTION_DIR) && go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Function coverage report generated: $(FUNCTION_DIR)/coverage.html$(NC)"

test-coverage-cli: ## Run CLI tests with coverage report
	@echo "$(YELLOW)Running CLI tests with coverage...$(NC)"
	@go test -v -race -coverprofile=cli-coverage.out ./cli/... ./cmd/youtube-webhook/...
	@go tool cover -html=cli-coverage.out -o cli-coverage.html
	@echo "$(GREEN)✓ CLI coverage report generated: cli-coverage.html$(NC)"

test-coverage-all: test-coverage test-coverage-cli ## Run all tests with coverage reports
	@echo "$(GREEN)✓ All coverage reports generated$(NC)"

test-watch: ## Watch for changes and run tests automatically (requires entr)
	@echo "$(YELLOW)Watching for changes... (Press Ctrl+C to stop)$(NC)"
	@cd $(FUNCTION_DIR) && find . -name "*.go" | entr -c go test -v ./...

benchmark: ## Run benchmark tests
	@echo "$(YELLOW)Running benchmarks...$(NC)"
	@cd $(FUNCTION_DIR) && go test -bench=. -benchmem ./...

## Code Quality Commands

fmt: ## Format Go code
	@echo "$(YELLOW)Formatting Go code...$(NC)"
	@cd $(FUNCTION_DIR) && go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

lint: ## Run golangci-lint on the code
	@echo "$(YELLOW)Running golangci-lint...$(NC)"
	@cd $(FUNCTION_DIR) && go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...

vet: ## Run go vet
	@echo "$(YELLOW)Running go vet...$(NC)"
	@cd $(FUNCTION_DIR) && go vet ./...
	@echo "$(GREEN)✓ Vet checks passed$(NC)"

check: fmt vet lint test ## Run all code quality checks

## Build Commands

build: ## Build the function locally
	@echo "$(YELLOW)Building function...$(NC)"
	@cd $(FUNCTION_DIR) && go build -o $(BINARY_NAME) .
	@echo "$(GREEN)✓ Function built: $(FUNCTION_DIR)/$(BINARY_NAME)$(NC)"

build-linux: ## Build for Linux (Cloud Functions target)
	@echo "$(YELLOW)Building for Linux...$(NC)"
	@cd $(FUNCTION_DIR) && GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux .
	@echo "$(GREEN)✓ Linux binary built: $(FUNCTION_DIR)/$(BINARY_NAME)-linux$(NC)"

build-cli: ## Build the CLI tool
	@echo "$(YELLOW)Building CLI tool...$(NC)"
	@go build -o $(CLI_BINARY) ./cmd/youtube-webhook
	@echo "$(GREEN)✓ CLI tool built: $(CLI_BINARY)$(NC)"

install-cli: build-cli ## Build and install the CLI tool to /usr/local/bin
	@echo "$(YELLOW)Installing CLI tool...$(NC)"
	@if [ -w /usr/local/bin ]; then \
		cp $(CLI_BINARY) /usr/local/bin/$(CLI_BINARY); \
		chmod +x /usr/local/bin/$(CLI_BINARY); \
	else \
		sudo cp $(CLI_BINARY) /usr/local/bin/$(CLI_BINARY); \
		sudo chmod +x /usr/local/bin/$(CLI_BINARY); \
	fi
	@echo "$(GREEN)✓ CLI tool installed to /usr/local/bin/$(CLI_BINARY)$(NC)"

## Local Development Commands

run-local: ## Run the function locally using Functions Framework
	@echo "$(YELLOW)Starting local development server...$(NC)"
	@echo "$(BLUE)Function will be available at: http://localhost:8080$(NC)"
	@go run ./cmd

test-local: ## Test the local function with a sample request
	@echo "$(YELLOW)Testing local function...$(NC)"
	@echo "$(BLUE)Sending verification challenge...$(NC)"
	@sleep 3
	@curl -X GET "http://localhost:8080/YouTubeWebhook?hub.challenge=test-challenge&hub.mode=subscribe&hub.topic=test" || echo "$(RED)❌ Local server not running? Try 'make run-local' first$(NC)"

## Terraform Commands

terraform-init: ## Initialize Terraform
	@echo "$(YELLOW)Initializing Terraform...$(NC)"
	@cd $(TERRAFORM_DIR) && terraform init
	@echo "$(GREEN)✓ Terraform initialized$(NC)"

terraform-validate: ## Validate Terraform configuration
	@echo "$(YELLOW)Validating Terraform configuration...$(NC)"
	@cd $(TERRAFORM_DIR) && terraform validate
	@echo "$(GREEN)✓ Terraform configuration is valid$(NC)"

terraform-fmt: ## Format Terraform files
	@echo "$(YELLOW)Formatting Terraform files...$(NC)"
	@cd $(TERRAFORM_DIR) && terraform fmt -recursive
	@echo "$(GREEN)✓ Terraform files formatted$(NC)"

terraform-plan: ## Plan Terraform deployment
	@echo "$(YELLOW)Planning Terraform deployment...$(NC)"
	@cd $(TERRAFORM_DIR) && terraform plan -out=tfplan
	@echo "$(GREEN)✓ Terraform plan created$(NC)"

terraform-apply: ## Apply Terraform configuration
	@echo "$(YELLOW)Applying Terraform configuration...$(NC)"
	@cd $(TERRAFORM_DIR) && terraform apply tfplan
	@echo "$(GREEN)✓ Infrastructure deployed$(NC)"

terraform-destroy: ## Destroy Terraform infrastructure
	@echo "$(RED)⚠️  This will destroy all infrastructure!$(NC)"
	@echo "$(YELLOW)Destroying Terraform infrastructure...$(NC)"
	@cd $(TERRAFORM_DIR) && terraform destroy
	@echo "$(GREEN)✓ Infrastructure destroyed$(NC)"

terraform-output: ## Show Terraform outputs
	@echo "$(YELLOW)Terraform outputs:$(NC)"
	@cd $(TERRAFORM_DIR) && terraform output

## Google Cloud Commands

deploy-function: build-linux ## Deploy function to Google Cloud
	@echo "$(YELLOW)Deploying function to Google Cloud...$(NC)"
	@cd $(FUNCTION_DIR) && gcloud functions deploy $(PROJECT_NAME) \
		--gen2 \
		--runtime go123 \
		--trigger-http \
		--allow-unauthenticated \
		--entry-point YouTubeWebhook \
		--memory 128Mi \
		--timeout 30s \
		--source .
	@echo "$(GREEN)✓ Function deployed$(NC)"

logs: ## View Cloud Function logs
	@echo "$(YELLOW)Fetching function logs...$(NC)"
	@gcloud functions logs read $(PROJECT_NAME) --limit=50

## Docker Commands (for local testing)

docker-build: ## Build Docker image for local testing
	@echo "$(YELLOW)Building Docker image...$(NC)"
	@docker build -t $(PROJECT_NAME):latest -f Dockerfile .
	@echo "$(GREEN)✓ Docker image built$(NC)"

docker-run: ## Run Docker container locally
	@echo "$(YELLOW)Running Docker container...$(NC)"
	@docker run -p 8080:8080 --env-file .env.local $(PROJECT_NAME):latest

## Utility Commands

clean: ## Clean build artifacts and caches
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@cd $(FUNCTION_DIR) && rm -f $(BINARY_NAME) $(BINARY_NAME)-linux
	@cd $(FUNCTION_DIR) && rm -f coverage.out coverage.html
	@cd $(TERRAFORM_DIR) && rm -f tfplan
	@cd $(TERRAFORM_DIR) && rm -f function-source.zip
	@echo "$(GREEN)✓ Clean completed$(NC)"

deps-update: ## Update Go dependencies
	@echo "$(YELLOW)Updating Go dependencies...$(NC)"
	@cd $(FUNCTION_DIR) && go get -u ./...
	@cd $(FUNCTION_DIR) && go mod tidy
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

security-scan: ## Run security scan on dependencies
	@echo "$(YELLOW)Running security scan...$(NC)"
	@cd $(FUNCTION_DIR) && command -v govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	@cd $(FUNCTION_DIR) && govulncheck ./...
	@echo "$(GREEN)✓ Security scan completed$(NC)"

## Development Workflow Commands

dev-setup: setup terraform-init ## Complete development setup
	@echo "$(GREEN)✓ Development environment fully set up$(NC)"
	@echo "$(BLUE)Next steps:$(NC)"
	@echo "  1. Create terraform/terraform.tfvars with your configuration values"
	@echo "  2. Run 'make terraform-plan' to plan infrastructure"
	@echo "  3. Run 'make test' to run tests"
	@echo "  4. Run 'make run-local' to start local development"

dev-test: test vet fmt ## Quick development test cycle
	@echo "$(GREEN)✓ Development checks passed$(NC)"

pre-commit: check test-coverage-all security-scan ## Complete pre-commit checks
	@echo "$(GREEN)✓ All pre-commit checks passed$(NC)"

pre-deploy: pre-commit terraform-validate ## Complete pre-deployment checks
	@echo "$(GREEN)✓ Ready for deployment$(NC)"

## CI/CD Commands

ci-test: ## CI-friendly test command
	@cd $(FUNCTION_DIR) && go test -v -race -coverprofile=coverage.out ./...
	@cd $(FUNCTION_DIR) && go tool cover -func=coverage.out
	@go test -v -race -coverprofile=cli-coverage.out ./cli/... ./cmd/youtube-webhook/...
	@go tool cover -func=cli-coverage.out

ci-build: ## CI-friendly build command
	@cd $(FUNCTION_DIR) && go build -v ./...

## Information Commands

status: ## Show project status
	@echo "$(BLUE)Project Status:$(NC)"
	@echo "  Go version: $$(cd $(FUNCTION_DIR) && go version 2>/dev/null || echo '$(RED)Not found$(NC)')"
	@echo "  Terraform version: $$(terraform version -json 2>/dev/null | jq -r '.terraform_version' 2>/dev/null || echo '$(RED)Not found$(NC)')"
	@echo "  Function directory: $(FUNCTION_DIR)"
	@echo "  Terraform directory: $(TERRAFORM_DIR)"
	@echo "  Last test run: $$(test -f $(FUNCTION_DIR)/coverage.out && stat -c %y $(FUNCTION_DIR)/coverage.out 2>/dev/null || echo 'Never')"

deps: ## Show dependency information
	@echo "$(BLUE)Dependencies:$(NC)"
	@echo "  Required: go ($(GO_VERSION)+), terraform, gcloud"
	@echo "  Optional: entr (for watch), docker, golint"
	@echo "  Go modules:"
	@cd $(FUNCTION_DIR) && go list -m all 2>/dev/null | head -10
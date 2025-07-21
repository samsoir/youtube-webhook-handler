# Contributing Guide

Thank you for your interest in contributing to the YouTube Webhook Service! This guide will help you get started with development and contributions.

## Architecture & Design

For detailed information about system design and planned features:
- **[SUBSCRIPTION_DESIGN.md](./SUBSCRIPTION_DESIGN.md)** - YouTube subscription management architecture
- **[README.md](./README.md)** - Project overview and setup
- **[TESTING.md](./TESTING.md)** - Testing strategy and guidelines

## Quick Start for Contributors

```bash
# 1. Clone and setup
git clone https://github.com/samsoir/youtube-webhook-handler.git
cd defreyssi.net-youtube-webhook
make dev-setup

# 2. Set up environment variables
export GITHUB_TOKEN="your-github-personal-access-token"
export REPO_OWNER="your-github-username"
export REPO_NAME="target-repository-name"
export ENVIRONMENT="dev"

# 3. Run tests to ensure everything works
make test

# 4. Start development server
make run-local
```

## Development Workflow

### Development Environment Setup

The project uses a Makefile-based workflow for consistency:

```bash
# Full development setup
make dev-setup               # Set up Go environment and dependencies
make test                   # Run all tests
make test-coverage          # Generate coverage report
make run-local              # Start local development server

# Quick development cycle
make test                   # Fast test run
make lint                   # Code formatting checks
make vet                    # Go vet analysis
make security-scan          # Security scanning

# Infrastructure
make terraform-validate     # Validate Terraform
make terraform-plan        # Plan infrastructure changes

# Cleanup
make clean                  # Clean build artifacts
```

### Code Quality Standards

#### Required Before Submitting PRs:

1. **Tests must pass**: `make test`
2. **Coverage must be ≥85%**: `make test-coverage`
3. **Security scan passes**: `make security-scan`
4. **Code properly formatted**: `make fmt` and `make vet`
5. **Terraform valid**: `make terraform-validate`

#### Testing Requirements:

- **Write tests** for new functionality
- **Test edge cases** and error conditions
- **Use mocks** for external GitHub API calls
- **Maintain coverage** above 85%

### Making Changes

#### 1. Create a Feature Branch
```bash
git checkout -b feature/your-feature-name
```

#### 2. Development Loop
```bash
# Make changes to code
# Run tests frequently
make test

# Check coverage before committing
make test-coverage

# Test with real local environment
make run-local
```

#### 3. Commit Guidelines

Use clear, descriptive commit messages:
```bash
git commit -m "Add comprehensive error handling for HTTP response writes

- Add proper error handling to all w.Write() calls to satisfy CWE-703
- Use fmt.Printf to log write errors without breaking HTTP response flow
- Maintain existing functionality while addressing static analysis security findings"
```

#### 4. Submit Pull Request

- **Base branch**: `main`
- **Include**: Description of changes and testing performed
- **Ensure**: All CI checks pass (branch protection enforced)

## Project Structure

```
defreyssi.net-youtube-webhook/
├── function/               # Go Cloud Function source code
│   ├── webhook.go         # Main webhook implementation
│   ├── webhook_test.go    # Comprehensive test suite
│   ├── go.mod            # Go module dependencies
│   └── go.sum            # Dependency checksums
├── terraform/             # Infrastructure as Code
│   ├── main.tf           # Main Terraform configuration
│   ├── variables.tf      # Input variables
│   ├── outputs.tf        # Output values
│   └── versions.tf       # Terraform and provider versions
├── .github/workflows/     # CI/CD pipelines
│   ├── ci.yml           # Continuous integration
│   └── deploy.yml       # Production deployment
├── Makefile              # Build automation (40+ targets)
├── TESTING.md           # Testing documentation
├── CONTRIBUTING.md      # This file
└── README.md            # Main project documentation
```

## Key Components

### Cloud Function (`function/webhook.go`)
- **Purpose**: Handle YouTube PubSubHubbub webhook notifications
- **Features**: Request validation, video filtering, GitHub API integration
- **Testing**: Comprehensive test suite with 87.8% coverage

### Infrastructure (`terraform/`)
- **Purpose**: Infrastructure as Code for Google Cloud Functions deployment
- **Features**: Service accounts, Cloud Functions, storage buckets
- **Validation**: Automated Terraform validation in CI

### CI/CD Pipeline (`.github/workflows/`)
- **CI Workflow**: Tests, linting, security scanning, Terraform validation
- **Deploy Workflow**: Automated deployment after CI success
- **Branch Protection**: Main branch requires PR reviews and passing CI

## Adding New Features

### 1. Webhook Functionality

When adding new webhook capabilities:

1. **Update handler** in `function/webhook.go`
2. **Add comprehensive tests** in `function/webhook_test.go`
3. **Maintain test coverage** above 85%
4. **Update documentation** in README and TESTING.md
5. **Test locally** with `make run-local`

### 2. Infrastructure Changes

When modifying infrastructure:

1. **Edit Terraform files** in `terraform/`
2. **Validate locally** with `make terraform-validate`
3. **Plan changes** with `make terraform-plan`
4. **Test in development environment** first
5. **Update documentation** if needed

### 3. CI/CD Pipeline Changes

When modifying workflows:

1. **Update workflow files** in `.github/workflows/`
2. **Test in feature branch** with actual PR
3. **Ensure backward compatibility**
4. **Document changes** in README

## Coding Standards

### Go Code

- **Style**: Follow standard Go formatting (`make fmt`)
- **Imports**: Use absolute imports where possible
- **Error Handling**: Always handle errors explicitly
- **Documentation**: Add comments for exported functions
- **Testing**: Write table-driven tests when appropriate

### Test Code

- **Naming**: Use descriptive test names explaining scenarios
- **Structure**: Follow Arrange-Act-Assert pattern
- **Mocking**: Mock external dependencies (GitHub API) consistently
- **Coverage**: Ensure new code paths are tested

### Infrastructure Code

- **Format**: Use `terraform fmt` for consistency
- **Variables**: Use descriptive variable names and documentation
- **Outputs**: Document all outputs clearly
- **Security**: Follow least-privilege principle

## Common Tasks

### Adding a Test

```go
func TestWebhook_NewFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "expected", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := NewFeature(tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Adding a Makefile Target

```makefile
new-target: ## Description of what this target does
	@echo "$(YELLOW)Running new target...$(NC)"
	@cd $(FUNCTION_DIR) && go run ./cmd/new-command
	@echo "$(GREEN)✓ New target completed$(NC)"
```

### Debugging Issues

```bash
# Run tests with verbose output
make test-verbose

# Check specific test coverage
make test-coverage

# Test local function with debugging
FUNCTION_TARGET=YouTubeWebhook make run-local

# Check security issues
make security-scan

# Validate infrastructure
make terraform-validate
```

## Getting Help

### Documentation
- **Main README**: Project overview and quick start
- **Testing Guide**: [TESTING.md](TESTING.md)
- **Makefile targets**: `make help`

### Common Issues

**Environment setup problems:**
```bash
make clean       # Clean build artifacts
make dev-setup   # Fresh setup
```

**Test failures:**
```bash
cd function && go mod tidy    # Update dependencies
make test-verbose            # Detailed test output
```

**Coverage issues:**
```bash
make test-coverage          # See detailed coverage report
```

**Local function issues:**
```bash
# Check environment variables
echo $GITHUB_TOKEN $REPO_OWNER $REPO_NAME

# Test with minimal setup
curl "http://localhost:8080?hub.challenge=test&hub.mode=subscribe&hub.topic=test"
```

## Security Considerations

### Required Security Practices

1. **Never commit secrets** to the repository
2. **Use environment variables** for sensitive configuration
3. **Validate all inputs** from external sources
4. **Handle errors gracefully** without exposing internal details
5. **Keep dependencies updated** and scan for vulnerabilities

### Security Testing

```bash
# Run comprehensive security scan
make security-scan

# Check for known vulnerabilities
cd function && govulncheck ./...

# Static analysis security scanner
cd function && gosec ./...
```

## Release Process

1. **All tests pass**: `make test`
2. **Coverage maintained**: `make test-coverage` (≥85%)
3. **Security scan passes**: `make security-scan`
4. **Documentation updated** if needed
5. **PR reviewed and approved** (branch protection enforced)
6. **Merge to main**
7. **Automatic deployment** via GitHub Actions

## Performance Guidelines

### Code Performance

- **Minimize allocations** in hot paths
- **Use appropriate data structures** for the use case
- **Profile with benchmarks** when optimizing
- **Test performance** with `make benchmark`

### Infrastructure Performance

- **Right-size Cloud Functions** memory allocation
- **Optimize cold start time** by minimizing dependencies
- **Monitor function performance** in production

## Code of Conduct

- **Be respectful** in all interactions
- **Test thoroughly** before submitting changes
- **Document clearly** any new features or changes
- **Follow established patterns** in the codebase
- **Ask questions** if anything is unclear
- **Provide constructive feedback** in reviews

## Questions?

- **Check documentation** first (README, TESTING.md)
- **Search existing issues** on GitHub
- **Create new issue** with detailed information
- **Include details**: error messages, steps to reproduce, environment info

## Helpful Resources

- **[Go Documentation](https://golang.org/doc/)**
- **[Google Cloud Functions](https://cloud.google.com/functions/docs)**
- **[Terraform Documentation](https://terraform.io/docs)**
- **[GitHub Actions](https://docs.github.com/en/actions)**
# Getting Started

## Prerequisites

### Required Software

- **Go** 1.23 or higher
- **Terraform** 1.12.2 or higher
- **Google Cloud SDK**
- **Make** (for build automation)
- **Git**

### Installation

#### macOS
```bash
# Install Go
brew install go

# Install Terraform
brew install terraform

# Install Google Cloud SDK
brew install --cask google-cloud-sdk
```

#### Linux (Ubuntu/Debian)
```bash
# Install Go
sudo snap install go --classic

# Install Terraform
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform

# Install Google Cloud SDK
echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key --keyring /usr/share/keyrings/cloud.google.gpg add -
sudo apt-get update && sudo apt-get install google-cloud-cli
```

#### Arch Linux
```bash
# Install Go
sudo pacman -S go

# Install Terraform
sudo pacman -S terraform

# Install Google Cloud SDK
yay -S google-cloud-cli
```

## Project Setup

### 1. Clone the Repository

```bash
git clone https://github.com/samsoir/youtube-webhook-handler.git
cd youtube-webhook-handler
```

### 2. Install Dependencies

```bash
# One-time development setup
make dev-setup

# This runs:
# - go mod download
# - go install testing tools
# - terraform init
```

### 3. Configure Environment

Create a `.env` file for local development:

```bash
# Required for local testing
export GITHUB_TOKEN="your-github-personal-access-token"
export REPO_OWNER="your-github-username"
export REPO_NAME="target-repository-name"
export ENVIRONMENT="dev"

# Optional configuration
export SUBSCRIPTION_BUCKET="test-bucket"
export FUNCTION_URL="http://localhost:8080"
export SUBSCRIPTION_LEASE_SECONDS="86400"
export RENEWAL_THRESHOLD_HOURS="12"
export MAX_RENEWAL_ATTEMPTS="3"
```

### 4. Configure Terraform

Create `terraform/terraform.tfvars`:

```hcl
project_id    = "your-google-cloud-project-id"
github_token  = "your-github-pat"
repo_owner    = "your-github-username"
repo_name     = "target-repository-name"
environment   = "dev"
region        = "us-central1"
```

## Local Development

### Running Locally

```bash
# Start the local server
make run-local

# The function will be available at:
# http://localhost:8080
```

### Testing the Local Server

```bash
# Test verification challenge
curl "http://localhost:8080?hub.challenge=test123&hub.mode=subscribe&hub.topic=test"

# Test subscription endpoint
curl -X POST "http://localhost:8080/subscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw"

# List subscriptions
curl "http://localhost:8080/subscriptions"
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run with race detection
make test-race

# Run specific test
go test -v ./function -run TestSubscribeToChannel
```

### Code Quality Checks

```bash
# Run all quality checks
make lint

# Format code
make fmt

# Run go vet
make vet

# Security scanning
make security-scan

# All pre-deployment checks
make pre-deploy
```

## Project Structure

```
.
├── function/              # Cloud Function source code
│   ├── webhook.go        # Main entry point
│   ├── storage_service.go # Storage layer
│   ├── notification_service.go # Notification processing
│   ├── github_client.go  # GitHub API client
│   └── *_test.go         # Test files
├── terraform/            # Infrastructure as Code
│   ├── main.tf          # Main configuration
│   ├── variables.tf     # Variable definitions
│   ├── outputs.tf       # Output values
│   └── scheduler.tf     # Cloud Scheduler config
├── docs/                # Documentation
│   ├── architecture/    # Architecture docs
│   ├── api/            # API documentation
│   ├── development/    # Development guides
│   ├── deployment/     # Deployment guides
│   └── operations/     # Operations guides
├── scripts/             # Utility scripts
├── .github/workflows/   # CI/CD pipelines
└── Makefile            # Build automation
```

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes

Follow the coding standards:
- Use `gofmt` for formatting
- Add tests for new functionality
- Update documentation as needed
- Follow SOLID principles

### 3. Test Your Changes

```bash
# Run tests
make test

# Check coverage (aim for >80%)
make test-coverage

# Run linting
make lint
```

### 4. Commit Your Changes

```bash
git add .
git commit -m "feat: Add new feature

- Detailed description of changes
- Any breaking changes
- Related issue numbers"
```

### 5. Push and Create PR

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub.

## Common Tasks

### Adding a New Endpoint

1. Add route handler in `webhook.go`
2. Implement business logic in appropriate service
3. Add tests for the new functionality
4. Update API documentation
5. Test locally before pushing

### Modifying Storage Schema

1. Update models in relevant files
2. Add migration logic if needed
3. Update tests
4. Document changes in architecture docs
5. Test with existing data

### Adding Environment Variables

1. Add to `.env` for local development
2. Update `terraform/variables.tf`
3. Add to GitHub Secrets for CI/CD
4. Document in relevant guides
5. Update configuration documentation

## Debugging

### Enable Verbose Logging

```go
// Add to your code temporarily
log.Printf("DEBUG: Variable value: %v", variable)
```

### View Cloud Function Logs

```bash
# View local logs
# Logs appear in terminal when running make run-local

# View production logs
gcloud functions logs read YouTubeWebhook --limit 50
```

### Common Issues

#### Port Already in Use
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>
```

#### Module Dependencies
```bash
# Clean module cache
go clean -modcache

# Re-download dependencies
go mod download
```

#### Test Failures
```bash
# Run test with verbose output
go test -v ./function -run TestName

# Debug with print statements
t.Logf("Debug: %v", variable)
```

## Next Steps

- Read the [Architecture Overview](../architecture/overview.md)
- Review [API Documentation](../api/endpoints.md)
- Learn about [Testing](./testing.md)
- Understand [Deployment](../deployment/cloud-functions.md)
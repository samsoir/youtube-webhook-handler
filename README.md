# YouTube Webhook Service

A serverless Google Cloud Function with CLI tool that processes YouTube PubSubHubbub webhook notifications and triggers GitHub Actions workflows when new videos are published.

[![Test Coverage](https://img.shields.io/badge/coverage-82.9%25-brightgreen)](docs/development/testing.md)
[![Go](https://img.shields.io/badge/go-1.23-blue)](https://golang.org/)
[![Terraform](https://img.shields.io/badge/terraform-1.12.2-blue)](https://terraform.io/)

## Quick Start

### Cloud Function
```bash
# Clone and setup
git clone https://github.com/samsoir/youtube-webhook-handler.git
cd youtube-webhook-handler
make dev-setup

# Configure (see docs/development/getting-started.md for details)
cp .env.example .env
# Edit .env with your configuration

# Run locally
make run-local

# Run tests
make test
```

### CLI Tool
```bash
# Build and install CLI
make install-cli

# Configure service URL
export YOUTUBE_WEBHOOK_URL=https://your-function.run.app

# Subscribe to a channel
youtube-webhook subscribe -channel UCXuqSBlHAE6Xw-yeJA0Tunw

# List subscriptions
youtube-webhook list

# Get help
youtube-webhook help
```

## Features

- 🔔 **Real-time Notifications** - Instant YouTube video notifications via PubSubHubbub
- 🔄 **Auto-renewal** - Automatic subscription renewal with Cloud Scheduler
- 🚀 **Serverless** - Auto-scaling with Cloud Functions Gen 2
- 🛡️ **Production Ready** - 82.9% test coverage with dependency injection architecture
- 📊 **Observable** - Structured logging and monitoring
- 🏗️ **Infrastructure as Code** - Complete Terraform configuration
- ⚡ **CLI Tool** - Command-line interface for subscription management

## Documentation

### 📚 Getting Started
- [**Quick Start Guide**](docs/development/getting-started.md) - Set up your development environment
- [**Architecture Overview**](docs/architecture/overview.md) - Understand the system design
- [**API Reference**](docs/api/endpoints.md) - Complete API documentation

### 🏗️ Development
- [**Testing Guide**](docs/development/testing.md) - Testing strategies and coverage
- [**Contributing**](CONTRIBUTING.md) - Contribution guidelines

### 🚀 Deployment
- [**Cloud Functions Deployment**](docs/deployment/cloud-functions.md) - Deploy to Google Cloud
- [**Terraform Guide**](docs/deployment/terraform.md) - Infrastructure as Code setup
- [**CI/CD Pipeline**](docs/deployment/ci-cd.md) - Automated deployment

### 🔧 Operations
- [**Monitoring**](docs/operations/monitoring.md) - Observability and alerting
- [**Renewal System**](docs/operations/renewal-system.md) - Auto-renewal configuration
- [**Troubleshooting**](docs/operations/troubleshooting.md) - Common issues and solutions

### 🏛️ Architecture
- [**System Architecture**](docs/architecture/overview.md) - High-level design
- [**Dependency Injection**](docs/architecture/dependency-injection.md) - DI architecture and patterns
- [**Subscription Management**](docs/architecture/subscription-management.md) - Subscription system details
- [**Webhook Processing**](docs/architecture/webhook-processing.md) - Notification handling

## Project Structure

```
.
├── function/           # Cloud Function source code
├── cli/               # CLI tool source code
│   ├── client/       # HTTP client for API
│   └── commands/     # Command implementations
├── cmd/              # CLI entry points
│   └── youtube-webhook/ # Main CLI application
├── terraform/         # Infrastructure configuration
├── docs/             # Comprehensive documentation
│   ├── architecture/ # System design docs
│   ├── api/         # API documentation
│   ├── development/ # Development guides
│   ├── deployment/  # Deployment guides
│   └── operations/  # Operations guides
├── scripts/          # Utility scripts
└── .github/         # CI/CD workflows
```

## Key Commands

### Development
```bash
make help           # Show all available commands
make test          # Run function tests
make test-cli      # Run CLI tests
make test-all      # Run all tests
make test-coverage # Run function tests with coverage
make lint          # Run linters
```

### Cloud Function
```bash
make run-local     # Start local server
make build-linux   # Build for Cloud Functions
make deploy-function # Deploy to Google Cloud
```

### CLI Tool
```bash
make build-cli     # Build CLI binary
make install-cli   # Build and install CLI
```

### Infrastructure
```bash
make terraform-plan  # Preview infrastructure changes
make terraform-apply # Apply infrastructure changes
```

## Requirements

- Go 1.23+
- Terraform 1.12.2+
- Google Cloud SDK
- GitHub Personal Access Token

## Configuration

Required environment variables:

```bash
GITHUB_TOKEN         # GitHub PAT with repo scope
REPO_OWNER          # GitHub username
REPO_NAME           # Target repository
SUBSCRIPTION_BUCKET # Cloud Storage bucket
FUNCTION_URL        # Cloud Function URL
```

See [Getting Started](docs/development/getting-started.md) for complete setup instructions.

## API Overview

### HTTP API
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | PubSubHubbub verification |
| `/` | POST | YouTube notifications |
| `/subscribe` | POST | Subscribe to channel |
| `/unsubscribe` | DELETE | Unsubscribe from channel |
| `/subscriptions` | GET | List subscriptions |
| `/renew` | POST | Renew subscriptions |

### CLI Commands
| Command | Description |
|---------|-----------|
| `subscribe -channel <ID>` | Subscribe to a YouTube channel |
| `unsubscribe -channel <ID>` | Unsubscribe from a channel |
| `list` | List all subscriptions |
| `renew` | Trigger renewal of expiring subscriptions |
| `help` | Show help information |

See [API Documentation](docs/api/endpoints.md) and [CLI README](cli/README.md) for complete details.

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

- 📖 [Documentation](docs/)
- 🐛 [Issue Tracker](https://github.com/samsoir/youtube-webhook-handler/issues)
- 💬 [Discussions](https://github.com/samsoir/youtube-webhook-handler/discussions)

---

Built with ❤️ using [Go](https://golang.org/) and deployed to [Google Cloud Functions](https://cloud.google.com/functions)
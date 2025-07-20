# YouTube Webhook Service

A serverless Google Cloud Function that processes YouTube PubSubHubbub webhook notifications and triggers GitHub Actions workflows when new videos are published.

[![Test Coverage](https://img.shields.io/badge/coverage-87.8%25-brightgreen)](TESTING.md)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](#deployment)
[![Go](https://img.shields.io/badge/go-1.23-blue)](https://golang.org/)
[![Terraform](https://img.shields.io/badge/terraform-1.12.2-blue)](https://terraform.io/)

## Table of Contents

- [🚀 Quick Start](#-quick-start)
- [✨ Features](#-features)
- [🔧 Setup](#-setup)
- [🏗️ Development](#️-development)
- [🚀 Deployment](#-deployment)
- [📚 Documentation](#-documentation)

## 🚀 Quick Start

```bash
# One-time setup
git clone <repository-url>
cd defreyssi.net-youtube-webhook
make dev-setup

# Configure secrets (required)
cat > terraform/terraform.tfvars << EOF
project_id    = "your-google-cloud-project-id"
github_token  = "your-github-pat"
repo_owner    = "your-github-username" 
repo_name     = "target-repository-name"
environment   = "dev"
EOF

# Start development
make test
make run-local
```

Visit `http://localhost:8080?hub.challenge=test&hub.mode=subscribe&hub.topic=test` to test your local function!

## ✨ Features

### 🎯 Core Functionality
- **YouTube Integration**: Handles PubSubHubbub webhook notifications
- **Smart Filtering**: Distinguishes new videos from updates using timestamp analysis
- **GitHub Integration**: Triggers repository dispatch events with video metadata
- **Real-time Updates**: Enables automated website updates when new videos are published

### 🛡️ Production Ready
- **87.8% test coverage** with comprehensive test suite
- **Security hardened** with Gosec scanning and vulnerability detection
- **CI/CD pipeline** with quality gates and automated deployment
- **Infrastructure as Code** with Terraform for reliable deployments

### 🏗️ Architecture
```
YouTube → PubSubHubbub → Cloud Function → GitHub API → Actions Workflow → Website Update
```

- **Language**: Go 1.23
- **Platform**: Google Cloud Functions (Gen 2)  
- **Infrastructure**: Terraform 1.12.2
- **Deployment**: GitHub Actions with branch protection

## 🔧 Setup

### Prerequisites

- **Go** (1.23+)
- **Terraform** (1.12.2+)
- **Google Cloud SDK**
- **GitHub Personal Access Token**

### Basic Setup

<details>
<summary>📖 Detailed Setup Instructions (click to expand)</summary>

#### Manual Installation

```bash
# Install Go (example for Arch Linux)
sudo pacman -S go

# Install Terraform
# See: https://developer.hashicorp.com/terraform/install

# Install Google Cloud SDK
# See: https://cloud.google.com/sdk/docs/install
```

#### GitHub Setup
1. Generate Personal Access Token with `repo` scope
2. Configure repository secrets (see [Deployment](#deployment))

#### Google Cloud Setup
1. Create or select a Google Cloud project
2. Enable required APIs (automatically handled by Terraform)
3. Create service account with appropriate permissions

</details>

### Environment Configuration

```bash
# Required for local development and deployment
export GITHUB_TOKEN="your-github-personal-access-token"
export REPO_OWNER="your-github-username"
export REPO_NAME="target-repository-name"
export ENVIRONMENT="dev"
```

## 🏗️ Development

### Common Commands

```bash
# Development workflow
make dev-setup        # Initial setup
make test            # Run tests
make test-coverage   # Coverage report
make run-local       # Start local server

# Code quality
make lint            # Code formatting
make vet            # Go vet checks
make security-scan   # Security analysis

# Build and deploy
make build-linux     # Build for Cloud Functions
make terraform-plan  # Plan infrastructure
```

See the [Contributing Guide](CONTRIBUTING.md) for detailed development workflow.

## 🚀 Deployment

### Automatic Deployment

The service automatically deploys to Google Cloud Functions when:
- ✅ Changes pushed to `main` branch
- ✅ All CI checks pass (tests, security, linting)
- ✅ Infrastructure validated with Terraform
- ✅ Branch protection requirements met

### Required GitHub Secrets

Configure in your repository settings:

**Secrets:**
- `GCP_CREDENTIALS` - Google Cloud service account JSON
- `GCP_PROJECT_ID` - Google Cloud project ID  
- `GH_WORKFLOW_TOKEN` - GitHub PAT with `repo` scope
- `GH_TARGET_REPO_NAME` - Target repository name

**Note:** GitHub reserves the `GITHUB_` prefix for system secrets, so we use `GH_` prefix for custom secrets.

<details>
<summary>🔧 Manual Deployment Setup (click to expand)</summary>

### Google Cloud Setup

1. Create service account with required permissions
2. Download service account JSON key
3. Add to GitHub secrets as `GCP_CREDENTIALS`
4. Configure other required secrets

### Local Deployment Testing

```bash
# Test full deployment process
make pre-deploy      # Run all pre-deployment checks
make terraform-plan  # Preview infrastructure changes

# Test with real environment
make build-linux     # Build production binary
make terraform-apply # Deploy infrastructure (with tfvars configured)
```

</details>

## 📚 Documentation

### Quick Links

- **[Testing Guide](TESTING.md)** - Comprehensive testing documentation
- **[Contributing Guide](CONTRIBUTING.md)** - Development workflow and standards
- **[API Documentation](#api-endpoints)** - Webhook endpoints and payload formats

### Project Structure

```
defreyssi.net-youtube-webhook/
├── function/           # Go Cloud Function source code
├── terraform/         # Infrastructure as Code  
├── .github/workflows/ # CI/CD pipelines
├── Makefile          # Build automation (40+ targets)
└── scripts/          # Utility scripts
```

### API Endpoints

<details>
<summary>📡 Webhook API Details</summary>

**GET /** - Verification Challenge
```bash
curl "https://your-function-url?hub.challenge=test&hub.mode=subscribe&hub.topic=test"
```

**POST /** - Video Notification  
Accepts YouTube Atom feed XML and triggers GitHub repository dispatch events.

**GitHub Repository Dispatch Event:**
```json
{
  "event_type": "youtube-video-published",
  "client_payload": {
    "video_id": "dQw4w9WgXcQ",
    "channel_id": "UCuAXFkgsw1L7xaCfnd5JJOw", 
    "title": "Video Title",
    "published": "2024-01-01T12:00:00Z",
    "video_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
  }
}
```

</details>

### Quality Assurance

- **Branch Protection**: Main branch requires PR reviews and passing CI
- **Security Scanning**: Automated vulnerability detection with Gosec and govulncheck  
- **Test Coverage**: 87.8% coverage with comprehensive test suite
- **Code Quality**: Automated linting, formatting, and vet checks

---

## Getting Help

- **📖 Documentation**: Check [TESTING.md](TESTING.md) and [CONTRIBUTING.md](CONTRIBUTING.md)
- **🐛 Issues**: [Create an issue](https://github.com/samsoir/youtube-webhook-handler/issues) 
- **💡 Questions**: Include error messages and steps to reproduce
- **📊 Monitoring**: Use `make logs` to view Cloud Function logs

Built with ❤️ using [Go](https://golang.org/) and deployed to [Google Cloud Functions](https://cloud.google.com/functions).
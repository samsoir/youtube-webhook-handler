# YouTube Webhook Service

A serverless Google Cloud Function that processes YouTube PubSubHubbub webhook notifications and triggers GitHub Actions workflows when new videos are published.

## 🎯 Purpose

This service enables real-time website updates when new YouTube videos are published by:
1. Receiving webhook notifications from YouTube
2. Filtering for new video publications (not updates)
3. Triggering GitHub Actions workflows via repository dispatch events
4. Enabling automated content updates on your website

## 🏗️ Architecture

```
YouTube → PubSubHubbub → Cloud Function → GitHub API → Actions Workflow → Website Update
```

- **Language**: Go 1.21
- **Platform**: Google Cloud Functions (Gen 2)
- **Infrastructure**: Terraform (Infrastructure as Code)
- **Testing**: Test-Driven Development with 90.2% coverage
- **CI/CD**: GitHub Actions with automated deployment

## 📁 Project Structure

```
defreyssi.net-youtube-webhook/
├── function/                 # Go Cloud Function source code
│   ├── webhook.go           # Main webhook implementation
│   ├── webhook_test.go      # Comprehensive test suite
│   ├── go.mod               # Go module dependencies
│   └── go.sum               # Dependency checksums
├── terraform/               # Infrastructure as Code
│   ├── main.tf             # Main Terraform configuration
│   ├── variables.tf        # Input variables
│   ├── outputs.tf          # Output values
│   └── terraform.tfvars.example  # Configuration template
├── .github/                # GitHub Actions and configuration
│   ├── workflows/          # CI/CD workflows
│   │   ├── ci.yml         # Continuous integration
│   │   ├── deploy.yml     # Production deployment
│   │   ├── release.yml    # Release management
│   │   └── dependabot-auto-merge.yml  # Dependency automation
│   └── dependabot.yml     # Dependabot configuration
├── Makefile                # Build automation and commands
├── .gitignore              # Git ignore patterns
├── LICENSE                 # MIT License
└── README.md               # This documentation
```

## 🚀 Features

### Core Functionality
- **YouTube Integration**: Handles PubSubHubbub webhook notifications
- **Smart Filtering**: Distinguishes new videos from updates using timestamp analysis
- **GitHub Integration**: Triggers repository dispatch events with video metadata
- **CORS Support**: Handles preflight requests for web integrations
- **Verification**: Responds to YouTube's subscription challenges

### Technical Features
- **Comprehensive Testing**: 90.2% test coverage with mock GitHub server
- **Error Handling**: Graceful handling of invalid XML, API failures, and missing configuration
- **Environment Configuration**: Flexible configuration via environment variables
- **Infrastructure as Code**: Complete Terraform setup for deployment
- **Build Automation**: Comprehensive Makefile with 40+ targets

## 📋 Prerequisites

- **Go**: Version 1.21 or higher
- **Terraform**: For infrastructure deployment
- **Google Cloud SDK**: For deployment and testing
- **GitHub Personal Access Token**: For repository dispatch API

## 🛠️ Development Setup

### 1. Clone and Setup
```bash
git clone <repository-url>
cd defreyssi.net-youtube-webhook
make dev-setup
```

### 2. Configure Environment
Copy and fill the configuration template:
```bash
cp terraform/terraform.tfvars.example terraform/terraform.tfvars
# Edit terraform/terraform.tfvars with your values
```

Required configuration:
- `project_id`: Your Google Cloud project ID
- `github_token`: GitHub personal access token with `repo` scope
- `repo_owner`: GitHub repository owner
- `repo_name`: Target repository name

### 3. Development Workflow
```bash
# Run tests
make test

# Check test coverage
make test-coverage

# Run local development server
make run-local

# Test local function
make test-local
```

## 🧪 Testing

The project follows Test-Driven Development (TDD) with comprehensive test coverage:

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with verbose output
make test-verbose

# Watch tests during development
make test-watch
```

### Test Coverage: 90.2% ✅

**Function-level breakdown:**
- `YouTubeWebhook()`: 100%
- `handleVerificationChallenge()`: 100%
- `handleNotification()`: 87.5%
- `triggerGitHubWorkflow()`: 85.7%
- `isNewVideo()`: 92.9%

### Test Scenarios Covered
- ✅ YouTube verification challenges
- ✅ CORS preflight requests
- ✅ Valid XML notification processing
- ✅ Invalid XML handling
- ✅ Empty notification handling
- ✅ Old video update filtering
- ✅ GitHub API integration
- ✅ GitHub API failure scenarios
- ✅ Missing environment variables
- ✅ New video detection logic
- ✅ Performance benchmarks

## 🚀 Deployment

### Automated Deployment (Recommended)

The project uses GitHub Actions for automated CI/CD. Simply push to the `main` branch to trigger automatic deployment to production.

#### Required GitHub Repository Secrets

Configure these secrets in your GitHub repository settings (`Settings → Secrets and variables → Actions`):

| Secret Name | Description | Example |
|-------------|-------------|---------|
| `GCP_CREDENTIALS` | Google Cloud service account JSON key | `{"type": "service_account", ...}` |
| `GCP_PROJECT_ID` | Your Google Cloud project ID | `my-project-123456` |
| `GITHUB_TOKEN_WEBHOOK` | GitHub Personal Access Token with `repo` scope | `ghp_xxxxxxxxxxxxxxxxxxxx` |
| `TARGET_REPO_NAME` | Repository name to trigger workflows | `defreyssi.net-v2` |

#### Setting up Google Cloud Service Account

1. Create a service account in Google Cloud Console
2. Grant the following roles:
   - Cloud Functions Admin
   - Storage Admin
   - Service Account User
   - Project IAM Admin (for function permissions)
3. Download the JSON key file
4. Copy the entire JSON content to the `GCP_CREDENTIALS` secret

#### Deployment Process

```bash
# Trigger deployment
git push origin main

# Or manually trigger via GitHub Actions UI
# Go to Actions → Deploy → Run workflow
```

### Manual Deployment (Development)

For local development and testing:

```bash
# Run all pre-deployment checks
make pre-deploy

# Manual Terraform deployment
make terraform-init
make terraform-plan
make terraform-apply

# Manual function deployment
make deploy-function
```

## 🔧 Configuration

### Environment Variables

**Required for Production:**
- `GITHUB_TOKEN`: GitHub personal access token
- `REPO_OWNER`: GitHub repository owner (e.g., "samsoir")
- `REPO_NAME`: Target repository name (e.g., "defreyssi.net-v2")

**Optional:**
- `ENVIRONMENT`: Environment identifier (default: "prod")
- `GITHUB_API_BASE_URL`: Custom GitHub API URL (for testing)

### Terraform Variables

Key configuration in `terraform/terraform.tfvars`:
- `project_id`: Google Cloud project ID
- `github_token`: GitHub PAT (stored securely)
- `region`: GCP region (default: "us-central1")
- `function_memory`: Memory allocation (default: "128Mi")
- `function_timeout`: Timeout in seconds (default: 30)

## 🔗 API Endpoints

The deployed function exposes these endpoints:

### `GET /?hub.challenge=<challenge>`
YouTube subscription verification endpoint.

**Response**: Returns the challenge parameter for successful verification.

### `POST /`
YouTube webhook notification endpoint.

**Request Body**: YouTube Atom feed XML
**Response**: 
- `200`: Successfully processed new video
- `200`: "Video update ignored" for old video updates
- `200`: "No video data" for empty notifications
- `400`: Invalid XML format
- `500`: GitHub API or configuration errors

### `OPTIONS /`
CORS preflight support for web integrations.

## 📊 Monitoring

### Local Testing
```bash
# View function logs
make logs

# Check function status
make status

# Run security scan
make security-scan
```

### Production Monitoring
- Google Cloud Function metrics and logs
- GitHub Actions workflow triggers
- Repository dispatch event tracking

## 🔄 Workflow Integration

When a new video is published, the webhook sends this payload to GitHub:

```json
{
  "event_type": "youtube-video-published",
  "client_payload": {
    "video_id": "dQw4w9WgXcQ",
    "channel_id": "UCuAXFkgsw1L7xaCfnd5JJOw",
    "title": "Video Title",
    "published": "2024-07-20T10:00:00Z",
    "updated": "2024-07-20T10:01:00Z",
    "video_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
    "environment": "prod"
  }
}
```

Your GitHub Actions workflow can listen for this event:

```yaml
on:
  repository_dispatch:
    types: [youtube-video-published]

jobs:
  update-website:
    runs-on: ubuntu-latest
    steps:
      - name: Process new video
        run: |
          echo "New video: ${{ github.event.client_payload.title }}"
          echo "Video ID: ${{ github.event.client_payload.video_id }}"
```

## 🛡️ Security Considerations

- GitHub token stored securely in Terraform state
- Function runs with minimal IAM permissions
- Input validation and sanitization
- No sensitive data in logs
- HTTPS-only communication

## 🧰 Available Commands

The Makefile provides 40+ commands for development:

```bash
# Development
make setup              # Setup development environment
make test              # Run tests
make build             # Build function
make run-local         # Start local server

# Code Quality
make fmt               # Format code
make lint              # Run linter
make vet               # Run go vet
make check             # Run all quality checks

# Deployment
make terraform-init    # Initialize Terraform
make terraform-plan    # Plan infrastructure
make deploy-function   # Deploy to Google Cloud

# Utilities
make clean             # Clean build artifacts
make deps-update       # Update dependencies
make security-scan     # Run security scan
```

Run `make help` to see all available commands.

## 🎯 New Video Detection Logic

The service uses intelligent timestamp analysis to distinguish new videos from updates:

**Considered "New Video" if:**
1. Published within the last hour
2. Time difference between `published` and `updated` is less than 15 minutes

**Considered "Update" if:**
1. Published more than 1 hour ago, OR
2. Large gap (>15 min) between publish and update times

This prevents unnecessary workflow triggers for video metadata updates.

## ⚙️ GitHub Actions Workflows

The project includes comprehensive CI/CD automation with four main workflows:

### 🔄 Continuous Integration (`ci.yml`)

**Triggers:** Push/PR to main or develop branches

**Jobs:**
- **Test** - Runs full test suite with 85% coverage enforcement
- **Lint** - Code formatting, go vet, and golint checks
- **Security** - Vulnerability scanning with Gosec and SARIF reports
- **Terraform** - Validates and formats Terraform configuration
- **Build** - Creates Linux binary for Cloud Functions deployment
- **Integration Test** - Benchmarks and local function testing

### 🚀 Production Deployment (`deploy.yml`)

**Triggers:** Push to main branch or manual dispatch

**Process:**
1. Runs full test suite
2. Builds function binary
3. Creates Terraform variables from GitHub secrets
4. Deploys infrastructure with Terraform
5. Tests deployed function endpoint
6. Reports deployment status

### 📦 Release Management (`release.yml`)

**Triggers:** Git tags (v*)

**Features:**
- Auto-generates changelog from git commits
- Creates GitHub releases with deployment information
- Includes technical details and rollback instructions

### 🤖 Dependency Management

**Dependabot Configuration:**
- Weekly updates for Go modules, GitHub Actions, and Terraform
- Auto-review assignment to repository owner
- Conventional commit message formatting

**Auto-merge Workflow:**
- Automatically merges minor and patch dependency updates
- Requires CI to pass before merging
- Major updates require manual review

### 🔧 GitHub Actions Setup

1. **Configure Repository Secrets** (see deployment section below)
2. **Enable Actions** in repository settings
3. **Configure Branch Protection** (optional but recommended):
   - Require PR reviews for main branch
   - Require status checks to pass
   - Require up-to-date branches before merging

## 🤝 Contributing

1. Follow Test-Driven Development (TDD)
2. Maintain test coverage above 85%
3. Run `make pre-commit` before committing
4. Update documentation for new features
5. Use conventional commit messages
6. All PRs must pass CI workflows before merging
7. Deployment happens automatically on merge to main

## 📈 Metrics

- **Test Coverage**: 90.2%
- **Response Time**: <100ms typical
- **Memory Usage**: 128Mi allocated
- **Cold Start**: <1s with Go runtime
- **Reliability**: 99.9% uptime target

## 🔍 Troubleshooting

### Common Issues

**Function not receiving webhooks:**
- Verify YouTube subscription is active
- Check function URL is publicly accessible
- Confirm CORS headers are set correctly

**GitHub API errors:**
- Verify token has `repo` scope
- Check repository owner/name configuration
- Ensure token hasn't expired

**Test failures:**
- Run `make deps-update` to update dependencies
- Check Go version compatibility (1.21+)
- Verify mock server setup in tests

### Debug Commands
```bash
make logs              # View function logs
make status           # Check project status
make terraform-output # Show infrastructure outputs
```

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Built with ❤️ using Test-Driven Development and Infrastructure as Code**
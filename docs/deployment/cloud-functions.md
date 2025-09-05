# Cloud Functions Deployment

## Overview

The YouTube Webhook Service deploys to Google Cloud Functions Gen 2, providing a serverless, auto-scaling solution.

## Prerequisites

### Google Cloud Setup

1. **Create Project**
   ```bash
   gcloud projects create your-project-id
   gcloud config set project your-project-id
   ```

2. **Enable APIs**
   ```bash
   gcloud services enable cloudfunctions.googleapis.com
   gcloud services enable cloudbuild.googleapis.com
   gcloud services enable storage.googleapis.com
   gcloud services enable cloudscheduler.googleapis.com
   ```

3. **Create Service Account**
   ```bash
   gcloud iam service-accounts create youtube-webhook-sa \
     --display-name="YouTube Webhook Service Account"
   ```

4. **Grant Permissions**
   ```bash
   gcloud projects add-iam-policy-binding your-project-id \
     --member="serviceAccount:youtube-webhook-sa@your-project-id.iam.gserviceaccount.com" \
     --role="roles/cloudfunctions.invoker" 
   
   gcloud projects add-iam-policy-binding your-project-id \
     --member="serviceAccount:youtube-webhook-sa@your-project-id.iam.gserviceaccount.com" \
     --role="roles/storage.objectAdmin"
   ```

## Deployment Methods

### Method 1: Terraform (Recommended)

#### Configure Variables

Create `terraform/terraform.tfvars`:

```hcl
project_id       = "your-project-id"
region          = "us-central1"
environment     = "production"
github_token    = "ghp_xxxxxxxxxxxx"
repo_owner      = "your-github-username"
repo_name       = "target-repo"
renewal_schedule = "0 */6 * * *"  # Every 6 hours
```

#### Deploy Infrastructure

```bash
# Initialize Terraform
cd terraform
terraform init

# Preview changes
terraform plan

# Apply changes
terraform apply

# Output function URL
terraform output function_url
```

### Method 2: GitHub Actions (CI/CD)

#### Setup Secrets

Add to GitHub repository secrets:

- `GCP_CREDENTIALS`: Service account JSON key
- `GCP_PROJECT_ID`: Google Cloud project ID
- `GH_WORKFLOW_TOKEN`: GitHub PAT with repo scope
- `GH_TARGET_REPO_NAME`: Target repository name

#### Automatic Deployment

Pushes to `main` branch trigger automatic deployment:

```yaml
name: Deploy to Cloud Functions

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: google-github-actions/auth@v1
        with:
          credentials_json: ${{ secrets.GCP_CREDENTIALS }}
      - uses: google-github-actions/deploy-cloud-functions@v1
        with:
          name: YouTubeWebhook
          runtime: go123
          entry_point: YouTubeWebhook
          source_dir: function
```

### Method 3: Manual Deployment

#### Using gcloud CLI

```bash
# Build the function
cd function
go mod download
GOOS=linux GOARCH=amd64 go build -o webhook

# Deploy to Cloud Functions
gcloud functions deploy YouTubeWebhook \
  --gen2 \
  --runtime=go123 \
  --region=us-central1 \
  --source=. \
  --entry-point=YouTubeWebhook \
  --trigger-http \
  --allow-unauthenticated \
  --set-env-vars="GITHUB_TOKEN=${GITHUB_TOKEN},REPO_OWNER=${REPO_OWNER},REPO_NAME=${REPO_NAME}" \
  --memory=256MB \
  --timeout=60s
```

## Configuration

### Environment Variables

Set via Terraform, gcloud, or Console:

```bash
# Required
GITHUB_TOKEN=ghp_xxxxxxxxxxxx
REPO_OWNER=your-github-username
REPO_NAME=target-repository
SUBSCRIPTION_BUCKET=your-bucket-name
FUNCTION_URL=https://region-project.cloudfunctions.net/YouTubeWebhook

# Optional
ENVIRONMENT=production
SUBSCRIPTION_LEASE_SECONDS=86400
RENEWAL_THRESHOLD_HOURS=12
MAX_RENEWAL_ATTEMPTS=3
```

### Function Settings

```hcl
# terraform/main.tf
resource "google_cloudfunctions2_function" "webhook" {
  name     = "YouTubeWebhook-${var.environment}"
  location = var.region

  build_config {
    runtime     = "go123"
    entry_point = "YouTubeWebhook"
    source {
      storage_source {
        bucket = google_storage_bucket.function_source.name
        object = google_storage_bucket_object.function_source.name
      }
    }
  }

  service_config {
    max_instance_count    = 100
    min_instance_count    = 0
    available_memory      = "256M"
    timeout_seconds       = 60
    ingress_settings      = "ALLOW_ALL"
    all_traffic_on_latest_revision = true
    
    environment_variables = {
      GITHUB_TOKEN         = var.github_token
      REPO_OWNER          = var.repo_owner
      REPO_NAME           = var.repo_name
      SUBSCRIPTION_BUCKET = google_storage_bucket.subscriptions.name
      FUNCTION_URL        = "https://${var.region}-${var.project_id}.cloudfunctions.net/YouTubeWebhook-${var.environment}"
    }
  }
}
```

## Post-Deployment

### 1. Verify Deployment

```bash
# Check function status
gcloud functions describe YouTubeWebhook --region=us-central1

# Test the function
curl "$(gcloud functions describe YouTubeWebhook --region=us-central1 --format='value(httpsTrigger.url)')?hub.challenge=test"
```

### 2. Subscribe to Channels

```bash
# Get function URL
FUNCTION_URL=$(gcloud functions describe YouTubeWebhook --region=us-central1 --format='value(httpsTrigger.url)')

# Subscribe to a channel
curl -X POST "$FUNCTION_URL/subscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw"
```

### 3. Verify Cloud Scheduler

```bash
# List scheduler jobs
gcloud scheduler jobs list --location=us-central1

# Run renewal manually
gcloud scheduler jobs run youtube-subscription-renewal-production --location=us-central1
```

### 4. Monitor Logs

```bash
# View recent logs
gcloud functions logs read YouTubeWebhook --limit=50

# Stream logs
gcloud functions logs read YouTubeWebhook --limit=50 --follow
```

## Rollback

### Using Terraform

```bash
# Revert to previous version
cd terraform
terraform plan -target=google_cloudfunctions2_function.webhook
terraform apply -target=google_cloudfunctions2_function.webhook
```

### Using gcloud

```bash
# List versions
gcloud functions versions list YouTubeWebhook --region=us-central1

# Rollback to specific version
gcloud functions deploy YouTubeWebhook \
  --region=us-central1 \
  --source-version=VERSION_ID
```

## Troubleshooting

### Deployment Failures

#### Build Errors
```bash
# Check Cloud Build logs
gcloud builds list --limit=5
gcloud builds log BUILD_ID
```

#### Permission Issues
```bash
# Check service account permissions
gcloud projects get-iam-policy your-project-id \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:youtube-webhook-sa@*"
```

### Runtime Issues

#### Function Not Responding
```bash
# Check function status
gcloud functions describe YouTubeWebhook --region=us-central1

# Check error logs
gcloud functions logs read YouTubeWebhook --limit=50 --filter="severity>=ERROR"
```

#### Memory Issues
```bash
# Increase memory allocation
gcloud functions deploy YouTubeWebhook \
  --memory=512MB
```

#### Timeout Issues
```bash
# Increase timeout
gcloud functions deploy YouTubeWebhook \
  --timeout=120s
```

## Performance Optimization

### Cold Start Mitigation

```hcl
# Keep minimum instances warm
service_config {
  min_instance_count = 1  # Keep 1 instance always warm
}
```

### Memory Tuning

Monitor memory usage and adjust:

```bash
# View metrics
gcloud monitoring metrics list --filter="metric.type:cloudfunctions.googleapis.com/function/execution_count"
```

### Connection Pooling

Already implemented in code:
- Singleton storage client
- Connection reuse
- 5-minute cache TTL

## Security

### Access Control

```bash
# Make function private
gcloud functions remove-iam-policy-binding YouTubeWebhook \
  --member="allUsers" \
  --role="roles/cloudfunctions.invoker"

# Grant specific access
gcloud functions add-iam-policy-binding YouTubeWebhook \
  --member="serviceAccount:scheduler-sa@project.iam.gserviceaccount.com" \
  --role="roles/cloudfunctions.invoker"
```

### Secret Management

```bash
# Use Secret Manager for sensitive data
gcloud secrets create github-token --data-file=token.txt

# Grant access to function
gcloud secrets add-iam-policy-binding github-token \
  --member="serviceAccount:youtube-webhook-sa@project.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

## Monitoring

### Set Up Alerts

```bash
# Create alert policy
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="Function Error Rate" \
  --condition-display-name="Error rate > 1%" \
  --condition-expression=' 
    resource.type="cloud_function"
    AND resource.labels.function_name="YouTubeWebhook"
    AND metric.type="cloudfunctions.googleapis.com/function/error_rate"
    AND value > 0.01'
```

### View Dashboards

Access Cloud Console:
1. Navigate to Cloud Functions
2. Select YouTubeWebhook
3. View Metrics tab

Key metrics:
- Invocations
- Execution time
- Memory usage
- Error rate

## Cost Optimization

### Estimate Costs

```bash
# Typical costs (as of 2025)
# - Invocations: $0.40 per million
# - Compute: $0.0000025 per GB-second
# - Networking: $0.12 per GB

# Example monthly cost for:
# - 100,000 invocations
# - 256MB memory
# - 1 second average execution
# Cost ≈ $0.04 + $0.06 = $0.10/month
```

### Reduce Costs

1. **Optimize Memory**: Use minimum required (256MB)
2. **Reduce Execution Time**: Cache responses
3. **Batch Operations**: Process multiple items together
4. **Set Max Instances**: Prevent runaway scaling

## Next Steps

- Configure [Terraform](./terraform.md) for infrastructure
- Set up [CI/CD](./ci-cd.md) pipelines
- Review [Monitoring](../operations/monitoring.md) setup
- Implement [Auto-renewal](../operations/renewal-system.md)

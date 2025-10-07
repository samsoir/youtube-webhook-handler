locals {
  function_name = "${var.function_name}-${var.environment}"
  # Sanitize bucket name to meet GCS requirements (lowercase, no spaces, start/end with alphanumeric)
  project_sanitized = replace(replace(replace(lower(var.project_id), " ", ""), "-", ""), "_", "")
  # Ensure bucket name starts with alphanumeric and is valid
  bucket_name = "gcs-${local.project_sanitized}-${var.function_name}-source"
  # Generate unique suffix for service account to avoid conflicts
  unique_suffix = substr(sha256("${var.project_id}-${var.function_name}-${var.environment}"), 0, 8)

  common_labels = merge(var.labels, {
    environment = var.environment
  })
}

# Configure the Google Cloud Provider
provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

# Enable required APIs
resource "google_project_service" "required_apis" {
  for_each = toset([
    "cloudresourcemanager.googleapis.com",
    "cloudfunctions.googleapis.com",
    "cloudbuild.googleapis.com",
    "artifactregistry.googleapis.com",
    "run.googleapis.com",
    "eventarc.googleapis.com",
    "storage.googleapis.com",
    "iam.googleapis.com"
  ])

  project = var.project_id
  service = each.value

  disable_on_destroy = false
}

# Storage bucket for Cloud Function source code
resource "google_storage_bucket" "function_source" {
  name     = local.bucket_name
  location = var.region
  project  = var.project_id

  uniform_bucket_level_access = true

  # Lifecycle management to clean up old versions
  lifecycle_rule {
    condition {
      age = 30
    }
    action {
      type = "Delete"
    }
  }

  lifecycle_rule {
    condition {
      num_newer_versions = 3
    }
    action {
      type = "Delete"
    }
  }

  versioning {
    enabled = true
  }

  labels = local.common_labels

  depends_on = [google_project_service.required_apis]
}

# Storage bucket for subscription state
resource "google_storage_bucket" "subscription_state" {
  name     = "${local.bucket_name}-subscriptions"
  location = var.region
  project  = var.project_id

  uniform_bucket_level_access = true

  # Lifecycle management for backup files
  lifecycle_rule {
    condition {
      age            = 90
      matches_prefix = ["subscriptions/backups/"]
    }
    action {
      type = "Delete"
    }
  }

  versioning {
    enabled = true
  }

  labels = merge(local.common_labels, {
    purpose = "subscription-state"
  })

  depends_on = [google_project_service.required_apis]
}

# Archive the function source code
data "archive_file" "function_source" {
  type        = "zip"
  source_dir  = "${path.module}/../function"
  output_path = "${path.module}/function-source.zip"
  excludes    = ["*.test", "go.sum"]
}

# Upload function source to Cloud Storage
resource "google_storage_bucket_object" "function_source" {
  name   = "function-source-${data.archive_file.function_source.output_md5}.zip"
  bucket = google_storage_bucket.function_source.name
  source = data.archive_file.function_source.output_path

  metadata = {
    md5Hash = data.archive_file.function_source.output_md5
  }
}

# Service account for the Cloud Function
resource "google_service_account" "function_sa" {
  account_id   = "yt-webhook-${var.environment}-${local.unique_suffix}"
  display_name = "YouTube Webhook Function Service Account (${var.environment})"
  description  = "Service account for the YouTube webhook Cloud Function (${var.environment})"
  project      = var.project_id

  depends_on = [google_project_service.required_apis]
}

# IAM binding for the service account
resource "google_project_iam_member" "function_sa_logging" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.function_sa.email}"

  depends_on = [google_project_service.required_apis]
}

# Grant function service account access to subscription state bucket
resource "google_storage_bucket_iam_member" "function_sa_subscription_bucket" {
  bucket = google_storage_bucket.subscription_state.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.function_sa.email}"

  depends_on = [google_storage_bucket.subscription_state]
}

# Cloud Function (Gen 2)
resource "google_cloudfunctions2_function" "youtube_webhook" {
  name     = local.function_name
  location = var.region
  project  = var.project_id

  description = "YouTube webhook handler for triggering website updates"

  build_config {
    runtime     = "go123"
    entry_point = "YouTubeWebhook"

    source {
      storage_source {
        bucket = google_storage_bucket.function_source.name
        object = google_storage_bucket_object.function_source.name
      }
    }

    environment_variables = {
      GOOS   = "linux"
      GOARCH = "amd64"
    }
  }

  service_config {
    max_instance_count    = var.max_instances
    min_instance_count    = var.min_instances
    available_memory      = var.function_memory
    timeout_seconds       = var.function_timeout
    service_account_email = google_service_account.function_sa.email

    environment_variables = {
      GITHUB_TOKEN               = var.github_token
      REPO_OWNER                 = var.repo_owner
      REPO_NAME                  = var.repo_name
      ENVIRONMENT                = var.environment
      SUBSCRIPTION_BUCKET        = google_storage_bucket.subscription_state.name
      RENEWAL_THRESHOLD_HOURS    = tostring(var.renewal_threshold_hours)
      MAX_RENEWAL_ATTEMPTS       = tostring(var.max_renewal_attempts)
      SUBSCRIPTION_LEASE_SECONDS = tostring(var.subscription_lease_seconds)
    }

    # Security settings
    ingress_settings               = "ALLOW_ALL"
    all_traffic_on_latest_revision = true
  }

  labels = local.common_labels

  depends_on = [
    google_project_service.required_apis,
    google_storage_bucket_object.function_source
  ]
}

# IAM member to allow public access to the webhook
# Gen 2 Cloud Functions use Cloud Run underneath, so we need to grant roles/run.invoker
# Using iam_member instead of iam_binding to avoid conflicts with scheduler_invoker
resource "google_cloud_run_service_iam_member" "webhook_public_invoker" {
  project  = google_cloudfunctions2_function.youtube_webhook.project
  location = google_cloudfunctions2_function.youtube_webhook.location
  service  = google_cloudfunctions2_function.youtube_webhook.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
# Cloud Scheduler for automatic subscription renewal

# Enable Cloud Scheduler API
resource "google_project_service" "scheduler_api" {
  project = var.project_id
  service = "cloudscheduler.googleapis.com"

  disable_on_destroy = false
}

# Service account for Cloud Scheduler (with necessary permissions)
resource "google_service_account" "scheduler_sa" {
  account_id   = "yt-scheduler-${var.environment}-${local.unique_suffix}"
  display_name = "YouTube Scheduler Service Account (${var.environment})"
  description  = "Service account for Cloud Scheduler to trigger subscription renewals (${var.environment})"
  project      = var.project_id

  depends_on = [google_project_service.scheduler_api]
}

# Grant the scheduler service account permission to invoke the function
resource "google_cloudfunctions2_function_iam_member" "scheduler_invoker" {
  project        = google_cloudfunctions2_function.youtube_webhook.project
  location       = google_cloudfunctions2_function.youtube_webhook.location
  cloud_function = google_cloudfunctions2_function.youtube_webhook.name
  role           = "roles/cloudfunctions.invoker"
  member         = "serviceAccount:${google_service_account.scheduler_sa.email}"

  depends_on = [google_service_account.scheduler_sa]
}

# Cloud Scheduler job for subscription renewal
resource "google_cloud_scheduler_job" "subscription_renewal" {
  name        = "youtube-subscription-renewal-${var.environment}"
  description = "Renew YouTube PubSubHubbub subscriptions automatically"
  schedule    = var.renewal_schedule
  time_zone   = var.renewal_timezone
  region      = var.region
  project     = var.project_id

  retry_config {
    retry_count          = 3
    max_retry_duration   = "300s"
    min_backoff_duration = "30s"
    max_backoff_duration = "300s"
    max_doublings        = 3
  }

  http_target {
    http_method = "POST"
    uri         = "${google_cloudfunctions2_function.youtube_webhook.url}/renew"

    headers = {
      "Content-Type" = "application/json"
      "User-Agent"   = "Google-Cloud-Scheduler/1.0"
    }

    body = base64encode(jsonencode({
      source    = "cloud-scheduler"
      timestamp = timestamp()
    }))

    oidc_token {
      service_account_email = google_service_account.scheduler_sa.email
      audience              = google_cloudfunctions2_function.youtube_webhook.url
    }
  }

  depends_on = [
    google_project_service.scheduler_api,
    google_cloudfunctions2_function.youtube_webhook,
    google_cloudfunctions2_function_iam_member.scheduler_invoker
  ]
}

# Output scheduler information
output "scheduler_job_name" {
  description = "Name of the Cloud Scheduler job"
  value       = google_cloud_scheduler_job.subscription_renewal.name
}

output "scheduler_service_account" {
  description = "Email of the scheduler service account"
  value       = google_service_account.scheduler_sa.email
}
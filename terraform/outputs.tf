output "webhook_url" {
  description = "The URL of the deployed Cloud Function webhook"
  value       = google_cloudfunctions2_function.youtube_webhook.url
}

output "function_name" {
  description = "The name of the deployed Cloud Function"
  value       = google_cloudfunctions2_function.youtube_webhook.name
}

output "service_account_email" {
  description = "Email of the service account used by the function"
  value       = google_service_account.function_sa.email
}

output "storage_bucket_name" {
  description = "Name of the Cloud Storage bucket for function source"
  value       = google_storage_bucket.function_source.name
}

output "subscription_bucket_name" {
  description = "Name of the Cloud Storage bucket for subscription state"
  value       = google_storage_bucket.subscription_state.name
}

output "project_id" {
  description = "The Google Cloud project ID"
  value       = var.project_id
}

output "region" {
  description = "The Google Cloud region"
  value       = var.region
}

output "environment" {
  description = "The deployment environment"
  value       = var.environment
}
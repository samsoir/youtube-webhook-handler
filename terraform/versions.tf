terraform {
  required_version = ">= 1.6"
  
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 5.0"
    }
  }
  
  # Backend configuration commented out for initial deployment
  # backend "gcs" {
  #   bucket = "your-terraform-state-bucket"
  #   prefix = "youtube-webhook"
  # }
}
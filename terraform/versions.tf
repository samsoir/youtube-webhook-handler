terraform {
  required_version = ">= 1.6"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 6.46"
    }
  }

  backend "gcs" {
    # Use the same bucket we create for function source, but different prefix for state
    prefix = "terraform-state"
  }
}
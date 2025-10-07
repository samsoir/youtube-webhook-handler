terraform {
  required_version = ">= 1.6"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.5"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 7.5"
    }
  }

  backend "gcs" {
    # Use the same bucket we create for function source, but different prefix for state
    prefix = "terraform-state"
  }
}
variable "project_id" {
  description = "Google Cloud Project ID"
  type        = string
}

variable "region" {
  description = "Google Cloud region for resources"
  type        = string
  default     = "us-central1"
}

variable "github_token" {
  description = "GitHub personal access token for triggering workflows"
  type        = string
  sensitive   = true
}

variable "repo_owner" {
  description = "GitHub repository owner (username or organization)"
  type        = string
  default     = "samsoir"
}

variable "repo_name" {
  description = "Target GitHub repository name for webhook triggers"
  type        = string
  default     = "defreyssi.net-v2"
}

variable "function_name" {
  description = "Name of the Cloud Function"
  type        = string
  default     = "youtube-webhook"
}

variable "function_memory" {
  description = "Memory allocation for the Cloud Function"
  type        = string
  default     = "128Mi"
}

variable "function_timeout" {
  description = "Timeout for the Cloud Function in seconds"
  type        = number
  default     = 30
}

variable "max_instances" {
  description = "Maximum number of function instances"
  type        = number
  default     = 10
}

variable "min_instances" {
  description = "Minimum number of function instances"
  type        = number
  default     = 0
}

variable "environment" {
  description = "Environment name (e.g., dev, staging, prod)"
  type        = string
  default     = "prod"
}

variable "labels" {
  description = "Labels to apply to resources"
  type        = map(string)
  default = {
    project     = "defreyssi-net"
    component   = "youtube-webhook"
    managed-by  = "terraform"
  }
}
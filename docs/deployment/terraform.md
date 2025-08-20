# Terraform Guide

## Overview

This guide provides instructions for setting up and managing the project's infrastructure using Terraform. The Terraform configuration handles the deployment of all necessary Google Cloud resources, including the Cloud Function, Cloud Storage bucket, and Cloud Scheduler job.

## Prerequisites

- [Terraform](https://learn.hashicorp.com/tutorials/terraform/install-cli) installed
- [Google Cloud SDK](https://cloud.google.com/sdk/docs/install) installed and authenticated
- A Google Cloud project with billing enabled

## Configuration

1.  **Navigate to the Terraform directory:**
    ```bash
    cd terraform
    ```

2.  **Create a `terraform.tfvars` file:**
    This file will contain your project-specific variables. You can copy the example file to get started:
    ```bash
    cp terraform.tfvars.example terraform.tfvars
    ```

3.  **Edit `terraform.tfvars`:**
    Update the file with your own values:
    ```hcl
    project_id       = "your-gcp-project-id"
    region           = "us-central1"
    environment      = "production"
    github_token     = "your-github-personal-access-token"
    repo_owner       = "your-github-username"
    repo_name        = "your-github-repo-name"
    renewal_schedule = "0 */6 * * *" # Every 6 hours
    ```

## Deployment

1.  **Initialize Terraform:**
    This will download the necessary providers and modules.
    ```bash
    make terraform-init
    ```

2.  **Plan the deployment:**
    This will show you what resources Terraform will create, modify, or destroy.
    ```bash
    make terraform-plan
    ```

3.  **Apply the changes:**
    This will create the resources in your Google Cloud project.
    ```bash
    make terraform-apply
    ```

## Destroying Infrastructure

To tear down all the resources created by Terraform, run:
```bash
make terraform-destroy
```
This is a destructive operation and will permanently delete your Cloud Function and other resources.

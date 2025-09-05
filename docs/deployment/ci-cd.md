# CI/CD Pipeline

## Overview

This project uses GitHub Actions for Continuous Integration (CI) and Continuous Deployment (CD). The workflows are defined in the `.github/workflows` directory and are triggered by pushes and pull requests to the `main` branch.

## Workflows

### `ci.yml`

This workflow runs on every push and pull request to the `main` branch. It performs the following checks:

- **Linting:** Checks the Go code for style issues.
- **Testing:** Runs the full test suite and calculates code coverage.
- **Security Scan:** Scans the code for potential security vulnerabilities.
- **Terraform Validation:** Validates the Terraform configuration.

### `deploy.yml`

This workflow is triggered after the `ci.yml` workflow successfully completes on the `main` branch. It deploys the Cloud Function to Google Cloud.

## Branch Protection

The `main` branch is protected by branch protection rules that require the `ci.yml` workflow to pass before any pull requests can be merged. This ensures that the code on the `main` branch is always in a deployable state.

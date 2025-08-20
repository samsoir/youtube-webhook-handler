# Subscription Renewal System

## Overview

The subscription renewal system is a critical component of the YouTube Webhook Service. It ensures that the subscriptions to YouTube channels remain active, allowing for continuous reception of video notifications.

## How it Works

1.  **Scheduled Trigger:** A Google Cloud Scheduler job is configured to trigger the renewal process at a regular interval (e.g., every 6 hours).
2.  **Renewal Endpoint:** The scheduler job sends a `POST` request to the `/renew` endpoint of the Cloud Function.
3.  **Subscription Check:** The function loads the current subscription state from Cloud Storage and identifies any subscriptions that are nearing their expiration date.
4.  **Renewal Request:** For each expiring subscription, the function sends a new subscription request to the PubSubHubbub hub.
5.  **State Update:** The subscription state is updated with the new expiration date and saved back to Cloud Storage.

## Configuration

The renewal system is configured using the following environment variables:

-   `RENEWAL_THRESHOLD_HOURS`: The number of hours before a subscription's expiration that the system should attempt to renew it. The default is `12`.
-   `MAX_RENEWAL_ATTEMPTS`: The maximum number of times the system will attempt to renew a subscription before marking it as failed. The default is `3`.

These variables can be set in the `terraform/terraform.tfvars` file.

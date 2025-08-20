# Troubleshooting

## Common Issues

### Function Not Receiving Notifications

-   **Check Subscription Status:** Use the `/subscriptions` endpoint to verify that the subscription for the channel is active.
-   **Check Cloud Function Logs:** Look for any errors in the Cloud Function logs that might indicate a problem with processing notifications.
-   **Check PubSubHubbub Hub:** Visit the [PubSubHubbub debug page](https://pubsubhubbub.appspot.com/topic-details) to check the status of your subscription.

### Function Failing to Deploy

-   **Check Cloud Build Logs:** If the deployment is failing, check the Cloud Build logs for any errors in the build process.
-   **Check Service Account Permissions:** Ensure that the service account used by the Cloud Function has the necessary permissions to access Cloud Storage and other required services.

### Terraform Errors

-   **Check `terraform.tfvars`:** Make sure that all the required variables are set correctly in your `terraform.tfvars` file.
-   **Run `terraform validate`:** Use the `make terraform-validate` command to check for any syntax errors in your Terraform configuration.

# Webhook Processing

## Overview

The webhook processing component is responsible for handling the incoming video notifications from YouTube's PubSubHubbub hub.

## How it Works

1.  **Notification Reception:** The Cloud Function receives a `POST` request to its root URL (`/`) from the PubSubHubbub hub.
2.  **XML Parsing:** The function parses the Atom XML feed from the request body to extract the video and channel information.
3.  **New Video Check:** The function checks if the video is new by comparing the `published` and `updated` timestamps in the feed. If the video is not new, the process stops.
4.  **Subscription Validation:** The function checks if there is an active subscription for the channel in the Cloud Storage bucket. If not, the process stops.
5.  **GitHub Workflow Trigger:** If the video is new and there is an active subscription, the function triggers a GitHub Actions workflow by sending a `repository_dispatch` event to the configured repository.

## Payload

The payload sent to the GitHub Actions workflow has the following structure:

```json
{
  "event_type": "youtube-video-published",
  "client_payload": {
    "video_id": "dQw4w9WgXcQ",
    "channel_id": "UCuAXFkgsw1L7xaCfnd5JJOw",
    "title": "Video Title",
    "published": "2025-01-21T12:00:00Z",
    "video_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
  }
}
```

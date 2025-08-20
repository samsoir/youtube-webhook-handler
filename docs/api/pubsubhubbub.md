# PubSubHubbub

## Overview

PubSubHubbub is a server-to-server web-hook-based pubsub (publish/subscribe) protocol. It is used by this project to receive real-time notifications from YouTube.

## Subscription Request

To subscribe to a YouTube channel's feed, you send a `POST` request to the PubSubHubbub hub.

**Request:**
```http
POST https://pubsubhubbub.appspot.com/subscribe
Content-Type: application/x-www-form-urlencoded

hub.callback=https://your-function-url/YouTubeWebhook
hub.topic=https://www.youtube.com/feeds/videos.xml?channel_id=CHANNEL_ID
hub.mode=subscribe
hub.verify=async
hub.lease_seconds=86400
```

-   `hub.callback`: The URL of your webhook function.
-   `hub.topic`: The URL of the YouTube channel's Atom feed.
-   `hub.mode`: Should be `subscribe`.
-   `hub.verify`: Should be `async`.
-   `hub.lease_seconds`: The number of seconds you want the subscription to be active. The maximum is 864000 (10 days).

## Unsubscription Request

To unsubscribe from a channel, you send a similar request with `hub.mode` set to `unsubscribe`.

**Request:**
```http
POST https://pubsubhubbub.appspot.com/subscribe
Content-Type: application/x-www-form-urlencoded

hub.callback=https://your-function-url/YouTubeWebhook
hub.topic=https://www.youtube.com/feeds/videos.xml?channel_id=CHANNEL_ID
hub.mode=unsubscribe
hub.verify=async
```

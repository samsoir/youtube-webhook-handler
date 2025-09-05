# Webhooks

## Overview

The YouTube Webhook Service uses webhooks to receive real-time notifications from YouTube. This is handled through the PubSubHubbub protocol.

## Verification Challenge

When you subscribe to a channel, the PubSubHubbub hub will send a `GET` request to your function's URL with a `hub.challenge` query parameter. Your function must respond with the value of this parameter to verify the subscription.

**Request:**
```http
GET /?hub.challenge=test123&hub.mode=subscribe&hub.topic=https://www.youtube.com/feeds/videos.xml?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw
```

**Response:**
```
200 OK
Content-Type: text/plain

test123
```

## Video Notification

When a new video is published to a subscribed channel, the hub will send a `POST` request to your function's URL with an Atom XML payload containing the video information.

**Request:**
```http
POST /
Content-Type: application/atom+xml

<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:yt="http://www.youtube.com/xml/schemas/2015">
  <entry>
    <id>yt:video:dQw4w9WgXcQ</id>
    <yt:videoId>dQw4w9WgXcQ</yt:videoId>
    <yt:channelId>UCuAXFkgsw1L7xaCfnd5JJOw</yt:channelId>
    <title>Video Title</title>
    <published>2025-01-21T12:00:00Z</published>
    <updated>2025-01-21T12:00:00Z</updated>
  </entry>
</feed>
```

**Response:**
```
204 No Content
```

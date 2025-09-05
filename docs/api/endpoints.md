# API Endpoints

## Overview

The YouTube Webhook Service exposes several HTTP endpoints for subscription management and webhook processing.

## Base URL

```
https://{region}-{project}.cloudfunctions.net/YouTubeWebhook
```

## Endpoints

### GET / - Verification Challenge

Handles PubSubHubbub verification challenges.

**Request:**
```http
GET /?hub.challenge=test123&hub.mode=subscribe&hub.topic=https://www.youtube.com/feeds/videos.xml?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw
```

**Query Parameters:**
- `hub.challenge` (required) - Verification challenge string
- `hub.mode` (required) - Subscription mode ("subscribe" or "unsubscribe")
- `hub.topic` (required) - YouTube channel feed URL

**Response:**
```
200 OK
Content-Type: text/plain

test123
```

---

### POST / - Video Notification

Receives YouTube video notifications from PubSubHubbub.

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

**GitHub Dispatch Event:**
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

---

### POST /subscribe

Subscribe to a YouTube channel for notifications.

**Request:**
```http
POST /subscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw
```

**Query Parameters:**
- `channel_id` (required) - YouTube channel ID

**Success Response (200 OK):**
```json
{
  "status": "success",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",
  "message": "Subscription initiated",
  "expires_at": "2025-01-22T10:30:00Z"
}
```

**Error Responses:**

**400 Bad Request - Invalid Channel ID:**
```json
{
  "status": "error",
  "channel_id": "invalid-id",
  "message": "Invalid channel ID format. Must be UC followed by 22 alphanumeric characters"
}
```

**400 Bad Request - Missing Parameter:**
```json
{
  "status": "error",
  "message": "channel_id parameter is required"
}
```

**409 Conflict - Already Subscribed:**
```json
{
  "status": "conflict",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",
  "message": "Already subscribed to this channel",
  "expires_at": "2025-01-22T10:30:00Z"
}
```

**502 Bad Gateway - Hub Unreachable:**
```json
{
  "status": "error",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",
  "message": "PubSubHubbub hub unreachable"
}
```

---

### DELETE /unsubscribe

Unsubscribe from a YouTube channel.

**Request:**
```http
DELETE /unsubscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw
```

**Query Parameters:**
- `channel_id` (required) - YouTube channel ID

**Success Response:**
```
204 No Content
```

**Error Responses:**

**400 Bad Request - Invalid Channel ID:**
```json
{
  "status": "error",
  "channel_id": "invalid-id",
  "message": "Invalid channel ID format"
}
```

**404 Not Found:**
```json
{
  "status": "error",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",
  "message": "Subscription not found for this channel"
}
```

---

### GET /subscriptions

List all active subscriptions.

**Request:**
```http
GET /subscriptions
```

**Success Response (200 OK):**
```json
{
  "subscriptions": [
    {
      "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",
      "status": "active",
      "expires_at": "2025-01-22T10:30:00Z",
      "days_until_expiry": 0.8
    },
    {
      "channel_id": "UCBJycsmduvYEL83R_U4JriQ",
      "status": "expired",
      "expires_at": "2025-01-20T10:30:00Z",
      "days_until_expiry": -1.2
    }
  ],
  "total": 2,
  "active": 1,
  "expired": 1
}
```

**Empty State Response (200 OK):**
```json
{
  "subscriptions": [],
  "total": 0,
  "active": 0,
  "expired": 0
}
```

---

### POST /renew

Trigger subscription renewal (called by Cloud Scheduler).

**Request:**
```http
POST /renew
Authorization: Bearer {OIDC_TOKEN}
```

**Headers:**
- `Authorization` - OIDC token from Cloud Scheduler

**Success Response (200 OK):**
```json
{
  "status": "success",
  "renewed": 2,
  "failed": 0,
  "message": "Renewed 2 expiring subscriptions"
}
```

**Partial Success Response (200 OK):**
```json
{
  "status": "partial",
  "renewed": 1,
  "failed": 1,
  "message": "Renewed 1 subscription, 1 failed",
  "errors": [
    {
      "channel_id": "UCBJycsmduvYEL83R_U4JriQ",
      "error": "Max renewal attempts exceeded"
    }
  ]
}
```

---

### OPTIONS /*

CORS preflight handler.

**Request:**
```http
OPTIONS /any-path
Origin: https://example.com
```

**Response:**
```http
200 OK
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

## Error Response Format

All error responses follow this structure:

```json
{
  "status": "error",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",  // Optional
  "message": "Human-readable error description"
}
```

## HTTP Status Codes

| Code | Description | Usage |
|------|-------------|-------|
| 200 | OK | Successful operations with response body |
| 204 | No Content | Successful operations without response body |
| 400 | Bad Request | Invalid input, validation errors |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Resource already exists |
| 500 | Internal Server Error | Server/storage errors |
| 502 | Bad Gateway | External service unreachable |
| 503 | Service Unavailable | External service down |
| 504 | Gateway Timeout | External service timeout |

## Rate Limiting

Currently no rate limiting is implemented. Consider adding:
- Per-IP rate limiting
- Per-channel subscription limits
- Webhook notification throttling

## Authentication

- Public endpoints: Verification challenges, webhook notifications
- Protected endpoints: Renewal endpoint (OIDC token required)
- Future: API key authentication for management endpoints
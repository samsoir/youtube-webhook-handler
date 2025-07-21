# YouTube Subscription Management Design

## Overview

This document outlines the design for YouTube PubSubHubbub subscription management, completing the integration with YouTube's real-time notification system.

## Architecture

### Components

1. **Subscription Endpoints** - HTTP endpoints for managing subscriptions
2. **State Persistence** - Cloud Storage for subscription state
3. **Auto-Renewal** - Cloud Scheduler for lease renewal
4. **Configuration** - Environment variables for target channels

## Detailed Design

### 1. Subscription State Management

#### Storage Format (Cloud Storage)
```json
{
  "subscriptions": {
    "UCXuqSBlHAE6Xw-yeJA0Tunw": {
      "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",
      "channel_name": "Tech Channel",
      "topic_url": "https://www.youtube.com/feeds/videos.xml?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw",
      "callback_url": "https://your-function-url/YouTubeWebhook",
      "status": "active",
      "lease_seconds": 86400,
      "subscribed_at": "2025-01-21T10:30:00Z",
      "expires_at": "2025-01-22T10:30:00Z",
      "last_renewal": "2025-01-21T10:30:00Z",
      "renewal_attempts": 0,
      "hub_response": "202 Accepted"
    }
  },
  "metadata": {
    "last_updated": "2025-01-21T10:30:00Z",
    "version": "1.0"
  }
}
```

#### File Location
- **Path**: `gs://your-bucket/subscriptions/state.json`
- **Backup**: `gs://your-bucket/subscriptions/backups/state-{timestamp}.json`

### 2. API Endpoints

#### POST /subscribe
Subscribe to a YouTube channel for real-time notifications.

**Request:**
- **Method**: `POST`
- **Path**: `/subscribe`
- **Query Parameters**: 
  - `channel_id` (required) - YouTube channel ID (format: `UC` + 22 alphanumeric chars)

**Request Flow:**
1. Validate channel_id format (`^UC[a-zA-Z0-9_-]{22}$`)
2. Check if already subscribed
3. Make PubSubHubbub subscription request to hub
4. Store subscription state in Cloud Storage
5. Return appropriate response

**Responses:**

**Success (200 OK):**
```json
{
  "status": "success",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",
  "message": "Subscription initiated",
  "expires_at": "2025-01-22T10:30:00Z"
}
```

**Already Subscribed (409 Conflict):**
```json
{
  "status": "conflict",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",
  "message": "Already subscribed to this channel",
  "expires_at": "2025-01-22T10:30:00Z"
}
```

**Invalid Channel ID (400 Bad Request):**
```json
{
  "status": "error",
  "channel_id": "invalid-id",
  "message": "Invalid channel ID format. Must be UC followed by 22 alphanumeric characters"
}
```

**Missing Parameter (400 Bad Request):**
```json
{
  "status": "error",
  "message": "channel_id parameter is required"
}
```

**Hub Unreachable (502 Bad Gateway):**
```json
{
  "status": "error",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",
  "message": "PubSubHubbub hub unreachable"
}
```

**Hub Error (5xx - Pass Through):**
```json
{
  "status": "error",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",
  "message": "PubSubHubbub hub error: 500 Internal Server Error"
}
```

**Request Timeout (504 Gateway Timeout):**
```json
{
  "status": "error",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",
  "message": "Request to PubSubHubbub hub timed out"
}
```

---

#### DELETE /unsubscribe
Unsubscribe from a YouTube channel.

**Request:**
- **Method**: `DELETE`
- **Path**: `/unsubscribe`
- **Query Parameters**: 
  - `channel_id` (required) - YouTube channel ID

**Request Flow:**
1. Validate channel_id format
2. Check if subscription exists in state
3. Make PubSubHubbub unsubscribe request to hub
4. Remove subscription from state (only if hub call succeeds)
5. Return appropriate response

**Responses:**

**Success (204 No Content):**
- **Body**: Empty (no response body)
- **Behavior**: Subscription removed from state

**Not Found (404 Not Found):**
```json
{
  "status": "error",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw", 
  "message": "Subscription not found for this channel"
}
```

**Invalid Channel ID (400 Bad Request):**
```json
{
  "status": "error",
  "channel_id": "invalid-id",
  "message": "Invalid channel ID format"
}
```

**Missing Parameter (400 Bad Request):**
```json
{
  "status": "error",
  "message": "channel_id parameter is required"
}
```

**Hub Failure (5xx):**
```json
{
  "status": "error",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",
  "message": "PubSubHubbub hub unreachable"
}
```
- **Behavior**: Subscription NOT removed from state (preserve consistency)

---

#### GET /subscriptions
List all current subscriptions and their status.

**Request:**
- **Method**: `GET`
- **Path**: `/subscriptions`
- **Query Parameters**: None

**Request Flow:**
1. Load subscription state from Cloud Storage
2. Calculate expiry status and days remaining
3. Generate summary statistics
4. Return subscription list with metadata

**Responses:**

**Success (200 OK):**
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

**Empty State (200 OK):**
```json
{
  "subscriptions": [],
  "total": 0,
  "active": 0,
  "expired": 0
}
```

**Storage Error (500 Internal Server Error):**
```json
{
  "status": "error",
  "message": "Unable to load subscription state from storage"
}
```

---

### API Design Decisions

#### HTTP Status Code Mapping
- **200 OK**: Successful operations with response data
- **204 No Content**: Successful deletion (no response body)
- **400 Bad Request**: Client errors (validation, missing parameters)
- **404 Not Found**: Resource doesn't exist (subscription not found)
- **409 Conflict**: Resource already exists (already subscribed)
- **500 Internal Server Error**: Server/storage errors
- **502 Bad Gateway**: External service (hub) unreachable
- **503 Service Unavailable**: External service (hub) returns 503
- **504 Gateway Timeout**: Timeout calling external service (hub)
- **5xx Pass Through**: Forward hub error codes when appropriate

#### Channel ID Validation
- **Format**: `^UC[a-zA-Z0-9_-]{22}$`
- **Length**: Exactly 24 characters
- **Prefix**: Must start with "UC"
- **Characters**: Alphanumeric, underscore, hyphen only

#### Error Response Format
```json
{
  "status": "error",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",  // Optional: included when relevant
  "message": "Descriptive error message"
}
```

#### Success Response Format
```json
{
  "status": "success",
  "channel_id": "UCXuqSBlHAE6Xw-yeJA0Tunw",
  "message": "Operation description", 
  "expires_at": "2025-01-22T10:30:00Z"  // ISO 8601 RFC3339 format
}
```

#### State Consistency Rules
1. **Subscribe**: Only store state after successful hub call
2. **Unsubscribe**: Only remove state after successful hub call  
3. **Network Failures**: Preserve existing state, return appropriate error
4. **Hub Errors**: Don't modify state, pass through error information

#### Timestamp Format
- **Standard**: ISO 8601 RFC3339 format (`2025-01-22T10:30:00Z`)
- **Timezone**: Always UTC (Z suffix)
- **Precision**: Second-level precision

#### Response Body Consistency
- **Always JSON**: Even error responses return structured JSON
- **Empty for 204**: No Content responses have completely empty body
- **Descriptive Messages**: Error messages provide actionable information

### 3. PubSubHubbub Integration

#### Subscription Request
```http
POST https://pubsubhubbub.appspot.com/subscribe
Content-Type: application/x-www-form-urlencoded

hub.callback=https://your-function-url/YouTubeWebhook
hub.topic=https://www.youtube.com/feeds/videos.xml?channel_id=CHANNEL_ID
hub.mode=subscribe
hub.verify=async
hub.lease_seconds=86400
hub.secret=optional-secret-for-verification
```

#### Verification Handling
Our existing webhook already handles verification challenges:
```go
// Already implemented in webhook.go
if challenge := r.URL.Query().Get("hub.challenge"); challenge != "" {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(challenge))
    return
}
```

### 4. Auto-Renewal System

#### Cloud Scheduler Configuration
```yaml
# terraform/scheduler.tf
resource "google_cloud_scheduler_job" "subscription_renewal" {
  name        = "youtube-subscription-renewal"
  description = "Renew YouTube PubSubHubbub subscriptions"
  schedule    = "0 */6 * * *"  # Every 6 hours
  time_zone   = "UTC"

  http_target {
    http_method = "POST"
    uri         = "${google_cloudfunctions_function.webhook.https_trigger_url}/renew"
    
    oidc_token {
      service_account_email = google_service_account.function_sa.email
    }
  }
}
```

#### Renewal Logic
1. **Trigger**: Every 6 hours via Cloud Scheduler
2. **Check**: Load subscription state from Cloud Storage  
3. **Identify**: Subscriptions expiring within 12 hours
4. **Renew**: Make new subscription requests for expiring subscriptions
5. **Update**: Save updated state back to Cloud Storage
6. **Alert**: Log renewal failures for monitoring

### 5. Configuration

#### Environment Variables
```bash
# Required for subscription management
YOUTUBE_CHANNELS=UCXuqSBlHAE6Xw-yeJA0Tunw,UCBJycsmduvYEL83R_U4JriQ
SUBSCRIPTION_BUCKET=your-subscription-state-bucket
FUNCTION_URL=https://your-region-your-project.cloudfunctions.net/YouTubeWebhook

# Optional configuration
SUBSCRIPTION_LEASE_SECONDS=86400
RENEWAL_THRESHOLD_HOURS=12
MAX_RENEWAL_ATTEMPTS=3
```

#### Channel Configuration
```go
type ChannelConfig struct {
    ID   string `json:"id"`
    Name string `json:"name,omitempty"`
}

// Parsed from YOUTUBE_CHANNELS environment variable
var targetChannels = []ChannelConfig{
    {ID: "UCXuqSBlHAE6Xw-yeJA0Tunw", Name: "Tech Channel"},
    {ID: "UCBJycsmduvYEL83R_U4JriQ", Name: "News Channel"},
}
```

### 6. Error Handling & Monitoring

#### Subscription Failures
- **Network errors**: Retry with exponential backoff
- **Hub rejections**: Log and alert, don't retry immediately  
- **Invalid channels**: Return 400 error, don't store state

#### Monitoring Metrics
- Active subscription count
- Failed subscription attempts
- Subscription renewal success rate
- Time until next expiry

#### Logging
```go
// Structured logging for observability
log.Printf("Subscription request: channel=%s, status=%s, hub_response=%d", 
    channelID, status, httpStatus)
```

## Implementation Plan

1. **Phase 1**: Add subscription state management and storage
2. **Phase 2**: Implement subscription/unsubscription endpoints  
3. **Phase 3**: Add auto-renewal with Cloud Scheduler
4. **Phase 4**: Add monitoring and alerting

## Testing Strategy

### Unit Tests
- Subscription state serialization/deserialization
- PubSubHubbub request formatting
- Error handling for various failure modes

### Integration Tests  
- End-to-end subscription flow
- Cloud Storage state persistence
- Hub verification challenge handling

### Load Tests
- Multiple concurrent subscriptions
- Renewal under load
- State file locking/consistency

## Security Considerations

- **Authentication**: Use service account for Cloud Storage access
- **Validation**: Strict channel ID format validation
- **Rate Limiting**: Prevent subscription spam
- **Secrets**: Optional hub.secret for request verification

## Future Enhancements

- **Web Dashboard**: Admin interface for subscription management
- **Webhook Secrets**: Enhanced security for hub verification
- **Multi-Region**: Distribute subscriptions across regions
- **Metrics Export**: Export metrics to monitoring systems
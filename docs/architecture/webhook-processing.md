# Webhook Processing

## Overview

The webhook processing component handles incoming video notifications from YouTube's PubSubHubbub hub using a dependency injection architecture for clean separation of concerns and improved testability.

## Architecture

The webhook processing system uses dependency injection to manage external services:

```go
type Dependencies struct {
    StorageClient StorageService       // Manages subscription state
    PubSubClient  PubSubClient         // Handles PubSubHubbub protocol
    GitHubClient  GitHubClientInterface // Triggers GitHub workflows
}
```

## Processing Flow

1. **Notification Reception:** The Cloud Function receives a `POST` request from the PubSubHubbub hub
2. **Dependency Resolution:** Handler retrieves dependencies via `GetDependencies()`
3. **XML Parsing:** Parses the Atom XML feed to extract video and channel information
4. **New Video Check:** `VideoProcessor` checks if the video is new by comparing timestamps
5. **Subscription Validation:** `StorageClient` loads subscription state from Cloud Storage
6. **GitHub Workflow Trigger:** `GitHubClient` triggers workflow if video is new and subscription is active

## Component Interactions

### Request Handler
```go
func handleNotification(w http.ResponseWriter, r *http.Request) {
    deps := GetDependencies()
    
    // Parse XML notification
    feed := parseAtomFeed(r.Body)
    
    // Check subscription status
    state, err := deps.StorageClient.LoadSubscriptionState(ctx)
    if !isSubscriptionActive(state, feed.ChannelID) {
        return // No active subscription
    }
    
    // Process new video
    processor := NewVideoProcessor()
    if processor.IsNewVideo(feed.Entry) {
        err := deps.GitHubClient.TriggerWorkflow(owner, repo, feed.Entry)
    }
}
```

### Verification Challenge
```go
func handleVerificationChallenge(w http.ResponseWriter, r *http.Request) {
    challenge := r.URL.Query().Get("hub.challenge")
    if challenge != "" {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(challenge))
    }
}
```

## Dependency Injection Benefits

### Testability
- Mock dependencies can be injected for unit testing
- No global state manipulation required
- Each test runs in isolation

### Flexibility
- Easy to swap implementations (e.g., different storage backends)
- Environment-specific configurations
- Feature toggles through dependency variants

### Example Test
```go
func TestWebhookProcessing(t *testing.T) {
    // Create test dependencies
    deps := &Dependencies{
        StorageClient: NewMockStorageService(),
        PubSubClient:  NewMockPubSubClient(),
        GitHubClient:  NewMockGitHubClient(),
    }
    
    // Configure mock behavior
    mockGitHub := deps.GitHubClient.(*MockGitHubClient)
    mockGitHub.TriggerFunc = func(owner, repo string, entry *Entry) error {
        return nil // Simulate success
    }
    
    // Inject dependencies
    SetDependencies(deps)
    defer SetDependencies(nil)
    
    // Test webhook processing
    req := createTestNotificationRequest()
    rec := httptest.NewRecorder()
    
    handleNotification(rec, req)
    
    // Verify workflow was triggered
    assert.Equal(t, 1, mockGitHub.TriggerCallCount)
}
```

## Payload Structure

The GitHub Actions workflow receives this payload:

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

## Error Handling

Each component handles errors gracefully:

1. **Storage Errors:** Logged and returns HTTP 500
2. **PubSub Errors:** Retries with exponential backoff
3. **GitHub API Errors:** Logged with details for debugging
4. **XML Parse Errors:** Returns HTTP 400 Bad Request

## Performance Optimizations

### Caching
- Storage client caches subscription state with TTL
- Reduces Cloud Storage API calls

### Connection Pooling
- HTTP clients reuse connections
- Reduces latency for external API calls

### Concurrent Processing
- Thread-safe dependency access
- Supports high-throughput webhook processing

## Related Documentation

- [Dependency Injection Architecture](./dependency-injection.md)
- [Storage Service](./storage-service.md)
- [API Endpoints](../api/endpoints.md)
- [Testing Guide](../development/testing.md)

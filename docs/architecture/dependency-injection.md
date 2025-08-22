# Dependency Injection Architecture

## Overview

The YouTube Webhook Service uses a dependency injection (DI) architecture to manage external dependencies, improve testability, and maintain clean separation of concerns. This architecture follows SOLID principles to ensure maintainable and extensible code.

## Core Components

### Dependencies Container

The central `Dependencies` struct acts as a service container, holding all external dependencies:

```go
// function/dependencies.go
type Dependencies struct {
    StorageClient StorageService       
    PubSubClient  PubSubClient
    GitHubClient  GitHubClientInterface
}
```

### Dependency Access Pattern

Dependencies are accessed through a singleton pattern with thread-safe initialization:

```go
func GetDependencies() *Dependencies {
    // Thread-safe singleton initialization
    // Returns production dependencies by default
}

func SetDependencies(deps *Dependencies) {
    // Used primarily for testing to inject mocks
}
```

## Interfaces

### StorageService

Manages subscription state persistence with caching optimization:

```go
type StorageService interface {
    LoadSubscriptionState(ctx context.Context) (*SubscriptionState, error)
    SaveSubscriptionState(ctx context.Context, state *SubscriptionState) error
}
```

**Implementations:**
- `CloudStorageService`: Production implementation with Google Cloud Storage
- `MockStorageService`: Test implementation with in-memory storage

### PubSubClient

Handles YouTube PubSubHubbub protocol interactions:

```go
type PubSubClient interface {
    Subscribe(channelID, callbackURL string) error
    Unsubscribe(channelID, callbackURL string) error
}
```

**Implementations:**
- `HTTPPubSubClient`: Production HTTP client for PubSubHubbub hub
- `MockPubSubClient`: Test implementation with configurable responses

### GitHubClientInterface

Manages GitHub API interactions for workflow dispatch:

```go
type GitHubClientInterface interface {
    TriggerWorkflow(repoOwner, repoName string, entry *Entry) error
}
```

**Implementations:**
- `GitHubClient`: Production implementation using GitHub API
- `MockGitHubClient`: Test implementation for workflow trigger simulation

## Dependency Creation

### Production Dependencies

Created automatically on first access:

```go
func CreateProductionDependencies() *Dependencies {
    return &Dependencies{
        StorageClient: NewCloudStorageService(),
        PubSubClient:  NewHTTPPubSubClient(),
        GitHubClient:  NewGitHubClient(),
    }
}
```

### Test Dependencies

Created explicitly in test setup:

```go
func CreateTestDependencies() *Dependencies {
    return &Dependencies{
        StorageClient: NewMockStorageService(),
        PubSubClient:  NewMockPubSubClient(),
        GitHubClient:  NewMockGitHubClient(),
    }
}
```

## Usage in Handlers

All HTTP handlers receive dependencies through the container:

```go
func handleSubscribe(w http.ResponseWriter, r *http.Request) {
    deps := GetDependencies()
    
    // Use storage service
    state, err := deps.StorageClient.LoadSubscriptionState(ctx)
    
    // Use PubSub client
    err = deps.PubSubClient.Subscribe(channelID, callbackURL)
    
    // Save state
    err = deps.StorageClient.SaveSubscriptionState(ctx, state)
}
```

## Testing Strategy

### Unit Testing with Mocks

Tests inject mock dependencies to isolate functionality:

```go
func TestSubscribeHandler(t *testing.T) {
    // Create test dependencies
    mockStorage := NewMockStorageService()
    mockPubSub := NewMockPubSubClient()
    
    deps := &Dependencies{
        StorageClient: mockStorage,
        PubSubClient:  mockPubSub,
    }
    
    // Inject test dependencies
    SetDependencies(deps)
    defer SetDependencies(nil) // Clean up
    
    // Configure mock behavior
    mockPubSub.SubscribeFunc = func(channelID, callback string) error {
        return nil // Simulate success
    }
    
    // Test handler
    req := httptest.NewRequest("POST", "/subscribe?channel_id=UC123", nil)
    rec := httptest.NewRecorder()
    
    handleSubscribe(rec, req)
    
    // Verify behavior
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.Equal(t, 1, mockPubSub.SubscribeCallCount)
}
```

### Integration Testing

Integration tests use real implementations where appropriate:

```go
func TestIntegration(t *testing.T) {
    // Use real storage with test bucket
    storage := NewCloudStorageService()
    storage.SetBucket("test-bucket")
    
    // Use mock external services
    deps := &Dependencies{
        StorageClient: storage,
        PubSubClient:  NewMockPubSubClient(),
        GitHubClient:  NewMockGitHubClient(),
    }
    
    SetDependencies(deps)
    defer SetDependencies(nil)
    
    // Run integration test
}
```

## Benefits

### 1. Testability
- Easy mock injection for unit tests
- No global state manipulation
- Isolated test execution

### 2. Maintainability
- Clear dependency boundaries
- Easy to add new dependencies
- Interface-based contracts

### 3. Flexibility
- Switch implementations without changing business logic
- Environment-specific configurations
- Feature toggles through dependency variants

### 4. SOLID Principles
- **Single Responsibility**: Each service has one purpose
- **Open/Closed**: Extend through new implementations
- **Liskov Substitution**: Mocks and real implementations are interchangeable
- **Interface Segregation**: Focused interfaces for each service
- **Dependency Inversion**: Depend on interfaces, not concrete types

## Best Practices

### 1. Always Use Interfaces
Define interfaces for all external dependencies:
```go
type MyService interface {
    DoSomething(ctx context.Context) error
}
```

### 2. Initialize Once
Create dependencies at startup, not per request:
```go
// Good: Created once
deps := GetDependencies()

// Bad: Created per request
func handler() {
    deps := CreateProductionDependencies() // Don't do this
}
```

### 3. Clean Up in Tests
Always restore original dependencies after tests:
```go
func TestSomething(t *testing.T) {
    original := GetDependencies()
    defer SetDependencies(original)
    
    // Test with custom dependencies
}
```

### 4. Configure Through Environment
Use environment variables for production configuration:
```go
func NewCloudStorageService() StorageService {
    bucket := os.Getenv("STORAGE_BUCKET")
    return &CloudStorageService{
        bucket: bucket,
    }
}
```

## Thread Safety

The dependency container uses mutex protection for thread-safe access:

```go
var (
    globalDependencies *Dependencies
    dependenciesMutex  sync.RWMutex
    dependenciesOnce   sync.Once
)
```

This ensures:
- Safe concurrent reads
- Protected writes during testing
- One-time initialization in production

## Performance Considerations

### Caching Strategy
Storage service includes built-in caching:
```go
type CloudStorageService struct {
    cache     *SubscriptionState
    cacheMu   sync.RWMutex
    cacheTime time.Time
    cacheTTL  time.Duration
}
```

### Connection Pooling
HTTP clients reuse connections:
```go
type HTTPPubSubClient struct {
    client *http.Client // Reused across requests
}
```

## Future Enhancements

### Potential Improvements
1. **Dependency Graph**: Automatic dependency resolution
2. **Lifecycle Management**: Init/shutdown hooks
3. **Configuration Injection**: Structured config objects
4. **Metrics Collection**: Performance monitoring per service
5. **Circuit Breakers**: Fault tolerance for external services

## Related Documentation

- [Testing Guide](../development/testing.md) - DI-based testing patterns
- [Webhook Processing](./webhook-processing.md) - How handlers use dependencies
- [Storage Service](./storage-service.md) - Storage implementation details
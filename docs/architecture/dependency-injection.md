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
- `MockStorageClient`: Test implementation with in-memory storage

#### CloudStorageService Architecture

The `CloudStorageService` uses a clean abstraction layer to prevent leaking implementation details:

```go
// Clean abstraction - no Google Cloud Storage types exposed
type CloudStorageOperations interface {
    GetObject(ctx context.Context, bucket, objectPath string) ([]byte, error)
    PutObject(ctx context.Context, bucket, objectPath string, data []byte) error
    Close() error
}

// Production implementation - Google Cloud Storage details hidden
type RealCloudStorageOperations struct {
    client *storage.Client  // Implementation detail not exposed
}

// Test implementation - in-memory storage for testing
type MockCloudStorageOperations struct {
    objects map[string][]byte
    getError error
    putError error
}
```

This pattern ensures that Google Cloud Storage implementation details (like `*storage.Client` or `stiface.Client`) never leak into the business logic.

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
    mockStorage := NewMockStorageClient()
    mockPubSub := NewMockPubSubClient()
    
    // Configure mock behavior
    mockStorage.SetState(&SubscriptionState{
        Subscriptions: make(map[string]*Subscription),
    })
    
    deps := &Dependencies{
        StorageClient: mockStorage,
        PubSubClient:  mockPubSub,
    }
    
    // Inject test dependencies
    SetDependencies(deps)
    defer SetDependencies(CreateProductionDependencies()) // Restore production deps
    
    // Test handler
    req := httptest.NewRequest("POST", "/subscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw", nil)
    rec := httptest.NewRecorder()
    
    handler := handleSubscribe(deps)
    handler(rec, req)
    
    // Verify behavior
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.Equal(t, 1, mockPubSub.GetSubscribeCount())
    assert.True(t, mockPubSub.IsSubscribed("UCXuqSBlHAE6Xw-yeJA0Tunw"))
}
```

### Testing CloudStorageService with Mock Operations

For testing the CloudStorageService itself, use the operations interface:

```go
func TestCloudStorageService_LoadSubscriptionState(t *testing.T) {
    // Create mock operations
    mockOps := &MockCloudStorageOperations{
        objects: make(map[string][]byte),
    }
    
    // Create service with mock operations
    service := NewCloudStorageServiceWithOperations(mockOps, "test-bucket")
    
    // Set up test data
    testState := &SubscriptionState{
        Subscriptions: map[string]*Subscription{
            "UC123": {
                ChannelID: "UC123",
                Status:    "active",
            },
        },
    }
    data, _ := json.Marshal(testState)
    mockOps.objects["subscriptions/state.json"] = data
    
    // Test load
    state, err := service.LoadSubscriptionState(context.Background())
    
    // Verify
    assert.NoError(t, err)
    assert.Equal(t, 1, len(state.Subscriptions))
    assert.Equal(t, "active", state.Subscriptions["UC123"].Status)
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

## Architectural Principles

### No Test-Mode Branching in Production Code

**Critical Principle**: Production code should never contain test-mode branching logic. All testing variations should be handled through dependency injection.

```go
// ❌ BAD: Test mode branching in production
func LoadState() (*State, error) {
    if testMode {  // Never do this!
        return mockStorage.Load()
    }
    return realStorage.Load()
}

// ✅ GOOD: Clean dependency injection
func (s *Service) LoadState() (*State, error) {
    return s.storage.Load()  // Storage is injected
}
```

### Clean Abstraction Layers

Keep implementation details hidden behind interfaces. Never leak third-party types into your business logic:

```go
// ❌ BAD: Leaking implementation details
type StorageService interface {
    GetClient() *storage.Client  // Exposes Google Cloud Storage type
}

// ✅ GOOD: Clean abstraction
type StorageService interface {
    LoadSubscriptionState(ctx context.Context) (*SubscriptionState, error)
    SaveSubscriptionState(ctx context.Context, state *SubscriptionState) error
}
```

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

### 5. Use Standard Testing Libraries
Don't reinvent the wheel - leverage existing testing libraries from the ecosystem:
```go
// Use Google Cloud Go testing utilities
import "cloud.google.com/go/storage"

// Use standard mocking patterns
import "github.com/stretchr/testify/assert"
import "github.com/stretchr/testify/mock"
```

### 6. Comprehensive Test Coverage
Aim for high test coverage through proper DI patterns:
- **Target**: 85% test coverage
- **Current**: 82.9% coverage achieved
- **Strategy**: Test all code paths using dependency injection to isolate components

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

## Lessons Learned

### Critical Production Fix (August 2024)

During a refactoring session, we discovered production was incorrectly using `MockStorageClient` instead of the real `CloudStorageService`. This critical issue meant:
- **Data was only stored in memory** - lost on function restart
- **No persistence** between function invocations
- **Subscription state was ephemeral**

**Root Cause**: Test-mode branching logic that incorrectly selected mock implementations in production.

**Solution**: Complete removal of test-mode branching, ensuring production always uses real implementations through proper dependency injection:

```go
// Before (WRONG):
func CreateProductionDependencies() *Dependencies {
    return &Dependencies{
        StorageClient: NewMockStorageClient(),  // ❌ Mock in production!
    }
}

// After (CORRECT):
func CreateProductionDependencies() *Dependencies {
    return &Dependencies{
        StorageClient: NewCloudStorageService(),  // ✅ Real implementation
    }
}
```

**Key Takeaway**: Never mix mock and production implementations in the same code path. Use dependency injection to completely separate test and production configurations.

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
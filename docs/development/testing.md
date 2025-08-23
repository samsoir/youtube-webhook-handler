# Testing Guide

## Overview

The project maintains **82.9% test coverage** for the function and **comprehensive CLI test coverage** with unit, integration, and load testing strategies using dependency injection for clean, isolated tests.

## Test Structure

```
function/
├── *_test.go           # Unit tests for each component
├── testutil/
│   └── common.go      # Shared test utilities
└── coverage.out       # Function coverage reports
cli/
├── client/
│   └── client_test.go # HTTP client tests
├── commands/
│   └── *_test.go      # Command handler tests
cmd/youtube-webhook/
└── main_test.go       # CLI integration tests
```

## Running Tests

### Basic Commands

```bash
# Run function tests only
make test

# Run CLI tests only
make test-cli

# Run all tests (function + CLI)
make test-all

# Run with coverage report
make test-coverage      # Function coverage
make test-coverage-cli  # CLI coverage
make test-coverage-all  # Combined coverage

# Run specific test file
go test -v ./function -run TestFileName
go test -v ./cli/client -run TestClient_Subscribe

# Run specific test case
go test -v ./function -run TestSubscribeToChannel/Success
```

### Coverage Analysis

```bash
# Generate function coverage report
make test-coverage

# Generate CLI coverage report
make test-coverage-cli

# Generate combined coverage reports
make test-coverage-all

# View coverage in browser
go tool cover -html=function/coverage.out     # Function coverage
go tool cover -html=cli-coverage.out          # CLI coverage

# Check coverage percentage
go test -cover ./function
go test -cover ./cli/...
```

Current coverage targets:
- **Function**: 82.9% (Minimum: 80%, Goal: 85%)
- **CLI Client**: 66.7%
- **CLI Commands**: 98.4%
- **Combined**: Comprehensive test coverage across all components

## Test Categories

The project includes tests for both the Cloud Function and CLI tool components.

### Function Tests

### Unit Tests

Test individual functions and methods in isolation.

**Example: Channel ID Validation**
```go
func TestValidateChannelID(t *testing.T) {
    tests := []struct {
        name     string
        channel  string
        expected bool
    }{
        {"valid", "UCXuqSBlHAE6Xw-yeJA0Tunw", true},
        {"too_short", "UC123", false},
        {"wrong_prefix", "ABXuqSBlHAE6Xw-yeJA0Tunw", false},
        {"invalid_chars", "UC!@#$%^&*()[]{}|\\<>? /", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := isValidChannelID(tt.channel)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Tests

Test interactions between components.

**Example: Storage Integration**
```go
func TestStorageIntegration(t *testing.T) {
    ctx := context.Background()
    
    // Create dependencies with real storage, mock external services
    deps := &Dependencies{
        StorageClient: NewCloudStorageService(), // Real storage
        PubSubClient:  NewMockPubSubClient(),   // Mock PubSub
        GitHubClient:  NewMockGitHubClient(),   // Mock GitHub
    }
    
    SetDependencies(deps)
    defer SetDependencies(nil)
    
    // Create test state
    state := &SubscriptionState{
        Subscriptions: map[string]*Subscription{
            "UCXuqSBlHAE6Xw-yeJA0Tunw": {
                ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
                Status:    "active",
            },
        },
    }
    
    // Save state through dependency
    err := deps.StorageClient.SaveSubscriptionState(ctx, state)
    assert.NoError(t, err)
    
    // Load state through dependency
    loaded, err := deps.StorageClient.LoadSubscriptionState(ctx)
    assert.NoError(t, err)
    assert.Equal(t, 1, len(loaded.Subscriptions))
}
```

### HTTP Handler Tests

Test HTTP endpoints and request/response handling.

**Example: Subscribe Endpoint**
```go
func TestSubscribeEndpoint(t *testing.T) {
    // Create and inject test dependencies
    deps := CreateTestDependencies()
    
    // Configure mock behavior
    mockPubSub := deps.PubSubClient.(*MockPubSubClient)
    mockPubSub.SubscribeFunc = func(channelID, callback string) error {
        return nil // Simulate successful subscription
    }
    
    SetDependencies(deps)
    defer SetDependencies(nil)
    
    // Test the endpoint
    req := httptest.NewRequest("POST", "/subscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw", nil)
    rec := httptest.NewRecorder()
    
    YouTubeWebhook(rec, req)
    
    assert.Equal(t, http.StatusOK, rec.Code)
    
    var response map[string]interface{}
    json.Unmarshal(rec.Body.Bytes(), &response)
    assert.Equal(t, "success", response["status"])
    
    // Verify mock was called
    assert.Equal(t, 1, mockPubSub.SubscribeCallCount)
}
```

### Concurrent Access Tests

Test thread safety and race conditions.

**Example: Concurrent State Access**
```go
func TestConcurrentStateAccess(t *testing.T) {
    // Create dependencies with thread-safe implementations
    deps := CreateProductionDependencies()
    SetDependencies(deps)
    defer SetDependencies(nil)
    
    ctx := context.Background()
    var wg sync.WaitGroup
    errors := make(chan error, 100)
    
    // Concurrent reads
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, err := deps.StorageClient.LoadSubscriptionState(ctx)
            if err != nil {
                errors <- err
            }
        }()
    }
    
    // Concurrent writes
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            state := &SubscriptionState{
                Subscriptions: map[string]*Subscription{
                    fmt.Sprintf("UC%d", id): {
                        ChannelID: fmt.Sprintf("UC%d", id),
                    },
                },
            }
            err := deps.StorageClient.SaveSubscriptionState(ctx, state)
            if err != nil {
                errors <- err
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    // Check for errors
    for err := range errors {
        t.Errorf("Concurrent access error: %v", err)
    }
}
```

### CLI Tests

#### CLI Client Tests
Test HTTP client interactions with the webhook service API.

```go
func TestClient_Subscribe_Success(t *testing.T) {
    // Create test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
            t.Errorf("Expected POST method, got %s", r.Method)
        }
        
        channelID := r.URL.Query().Get("channel_id")
        if channelID != "UCXuqSBlHAE6Xw-yeJA0Tunw" {
            t.Errorf("Expected specific channel ID, got %s", channelID)
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(webhook.APIResponse{
            Status:    "success",
            Message:   "Subscribed successfully",
            ExpiresAt: "2024-01-22T15:30:00Z",
        })
    }))
    defer server.Close()
    
    client := NewClient(server.URL, 30*time.Second)
    resp, err := client.Subscribe("UCXuqSBlHAE6Xw-yeJA0Tunw")
    
    assert.NoError(t, err)
    assert.Equal(t, "success", resp.Status)
}
```

#### CLI Command Tests
Test command implementations and user interactions.

```go
func TestSubscribe_Success(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(webhook.APIResponse{
            Status:    "success",
            Message:   "Subscribed successfully",
            ExpiresAt: "2024-01-22T15:30:00Z",
        })
    }))
    defer server.Close()
    
    config := SubscribeConfig{
        BaseURL:   server.URL,
        ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
        Timeout:   30 * time.Second,
    }
    
    err := Subscribe(config)
    assert.NoError(t, err)
}
```

#### CLI Integration Tests
Test the complete CLI binary with real subcommand execution.

```go
func TestIntegration_CompleteWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Build CLI binary for testing
    binaryPath := buildCLIBinary(t)
    defer os.Remove(binaryPath)
    
    // Setup test server with subscription tracking
    subscriptions := make(map[string]webhook.SubscriptionInfo)
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Handle subscribe, list, unsubscribe operations
        // Maintain state across requests
    }))
    defer server.Close()
    
    // Test complete workflow: subscribe → list → unsubscribe
    testChannelID := "UCXuqSBlHAE6Xw-yeJA0Tunw"
    
    // 1. Subscribe
    cmd := exec.Command(binaryPath, "subscribe", "-url", server.URL, "-channel", testChannelID)
    output, err := cmd.CombinedOutput()
    assert.NoError(t, err, "Subscribe command should succeed")
    
    // 2. Verify subscription in list
    cmd = exec.Command(binaryPath, "list", "-url", server.URL)
    output, err = cmd.CombinedOutput()
    assert.NoError(t, err)
    assert.Contains(t, string(output), testChannelID)
    
    // 3. Unsubscribe
    cmd = exec.Command(binaryPath, "unsubscribe", "-url", server.URL, "-channel", testChannelID)
    output, err = cmd.CombinedOutput()
    assert.NoError(t, err)
}
```

## Mocking Strategies

### Using Dependency Injection

```go
func TestWithDependencyInjection(t *testing.T) {
    // Create mock dependencies
    mockStorage := NewMockStorageService()
    mockPubSub := NewMockPubSubClient()
    mockGitHub := NewMockGitHubClient()
    
    deps := &Dependencies{
        StorageClient: mockStorage,
        PubSubClient:  mockPubSub,
        GitHubClient:  mockGitHub,
    }
    
    // Inject test dependencies
    SetDependencies(deps)
    defer SetDependencies(nil) // Clean up
    
    // Configure mock behavior
    mockStorage.State = &SubscriptionState{
        Subscriptions: make(map[string]*Subscription),
    }
    mockPubSub.SubscribeFunc = func(channelID, callback string) error {
        return nil
    }
    
    // Test code
    // ...
    
    // Verify mock interactions
    assert.Equal(t, 1, mockStorage.LoadCallCount)
    assert.Equal(t, 1, mockPubSub.SubscribeCallCount)
}
```

### Creating Test Dependencies

```go
func TestWithTestDependencies(t *testing.T) {
    // Create complete test dependencies
    deps := CreateTestDependencies()
    
    // Customize specific behavior
    mockPubSub := deps.PubSubClient.(*MockPubSubClient)
    mockPubSub.SubscribeFunc = func(channelID, callback string) error {
        if channelID == "invalid" {
            return errors.New("invalid channel")
        }
        return nil
    }
    
    // Inject dependencies
    SetDependencies(deps)
    defer SetDependencies(nil)
    
    // Run tests with isolated dependencies
    state, err := deps.StorageClient.LoadSubscriptionState(context.Background())
    assert.NoError(t, err)
    assert.NotNil(t, state)
}
```

## Test Data Management

### Test Fixtures

```go
func getTestSubscriptionState() *SubscriptionState {
    return &SubscriptionState{
        Subscriptions: map[string]*Subscription{
            "UCXuqSBlHAE6Xw-yeJA0Tunw": {
                ChannelID:    "UCXuqSBlHAE6Xw-yeJA0Tunw",
                Status:       "active",
                ExpiresAt:    time.Now().Add(24 * time.Hour),
                SubscribedAt: time.Now(),
            },
        },
        Metadata: &Metadata{
            LastUpdated: time.Now(),
            Version:     "1.0",
        },
    }
}
```

### Test Cleanup

```go
func TestWithCleanup(t *testing.T) {
    // Save original dependencies
    originalDeps := GetDependencies()
    
    // Create and inject test dependencies
    testDeps := CreateTestDependencies()
    SetDependencies(testDeps)
    
    // Cleanup
    t.Cleanup(func() {
        SetDependencies(originalDeps)
        // Clear any test state if needed
    })
    
    // Test code
    // ...
}
```

## Error Testing

### Testing Error Paths

```go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name        string
        setupDeps   func(*Dependencies)
        expectedErr string
    }{
        {
            name: "storage_load_error",
            setupDeps: func(deps *Dependencies) {
                mock := deps.StorageClient.(*MockStorageService)
                mock.LoadError = errors.New("storage unavailable")
            },
            expectedErr: "storage unavailable",
        },
        {
            name: "pubsub_subscribe_error",
            setupDeps: func(deps *Dependencies) {
                mock := deps.PubSubClient.(*MockPubSubClient)
                mock.SubscribeFunc = func(channelID, callback string) error {
                    return errors.New("subscription failed")
                }
            },
            expectedErr: "subscription failed",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create test dependencies
            deps := CreateTestDependencies()
            tt.setupDeps(deps)
            
            // Inject dependencies
            SetDependencies(deps)
            defer SetDependencies(nil)
            
            // Test error handling
            err := performOperation()
            assert.Contains(t, err.Error(), tt.expectedErr)
        })
    }
}
```

## Performance Testing

### Benchmark Tests

```go
func BenchmarkSubscriptionLoad(b *testing.B) {
    // Use production dependencies for realistic benchmarks
    deps := CreateProductionDependencies()
    SetDependencies(deps)
    defer SetDependencies(nil)
    
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = deps.StorageClient.LoadSubscriptionState(ctx)
    }
}
```

### Load Testing

```go
func TestHighLoad(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }
    
    // Use production dependencies for realistic load testing
    deps := CreateProductionDependencies()
    SetDependencies(deps)
    defer SetDependencies(nil)
    
    ctx := context.Background()
    start := time.Now()
    requests := 1000
    
    for i := 0; i < requests; i++ {
        go func() {
            deps.StorageClient.LoadSubscriptionState(ctx)
        }()
    }
    
    duration := time.Since(start)
    rps := float64(requests) / duration.Seconds()
    
    t.Logf("Processed %d requests in %v (%.2f req/s)", requests, duration, rps)
    assert.Greater(t, rps, 100.0, "Should handle >100 requests per second")
}
```

## Test Best Practices

### 1. Table-Driven Tests

Use table-driven tests for comprehensive coverage:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        // Test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

### 2. Isolation

Each test should be independent:

```go
func TestIsolated(t *testing.T) {
    // Create isolated test dependencies
    deps := CreateTestDependencies()
    SetDependencies(deps)
    defer SetDependencies(nil)
    
    // Test logic runs with isolated dependencies
}
```

### 3. Descriptive Names

Use clear, descriptive test names:

```go
func TestSubscribeToChannel_AlreadySubscribed_ReturnsConflict(t *testing.T) {
    // Test implementation
}
```

### 4. Assert Messages

Include helpful messages in assertions:

```go
assert.Equal(t, expected, actual, "Subscription count should match after renewal")
```

### 5. Test Coverage Goals

Focus on:
- Critical paths (100% coverage)
- Error handling (>90% coverage)
- Edge cases (>80% coverage)
- Happy paths (100% coverage)

## Continuous Integration

Tests run automatically on:
- Pull requests
- Commits to main branch
- Scheduled daily runs

CI checks:
- All function tests pass (82.9% coverage)
- All CLI tests pass (comprehensive coverage)
- No race conditions
- Linting passes
- Security scan clean

## Troubleshooting

### Common Issues

#### Race Condition Detected
```bash
# Run with race detector
go test -race ./function

# Fix by adding proper synchronization
var mu sync.Mutex
mu.Lock()
defer mu.Unlock()
```

#### Flaky Tests
```bash
# Run test multiple times
go test -count=10 -run TestFlaky

# Add delays or synchronization
time.Sleep(100 * time.Millisecond)
```

#### Coverage Gaps
```bash
# Find uncovered lines
go tool cover -html=coverage.out

# Add tests for uncovered code
```

## Next Steps

- Review [Dependency Injection Architecture](../architecture/dependency-injection.md)
- Review [CLI Documentation](../../cli/README.md)
- Review [Contributing Guide](./contributing.md)
- Learn about [Deployment](../deployment/cloud-functions.md)
- Understand [Monitoring](../operations/monitoring.md)

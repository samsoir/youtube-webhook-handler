# Testing Guide

## Overview

The project maintains **82.4% test coverage** with comprehensive unit, integration, and load testing strategies.

## Test Structure

```
function/
├── *_test.go           # Unit tests for each component
├── testutil/
│   └── common.go      # Shared test utilities
└── coverage.out       # Coverage reports
```

## Running Tests

### Basic Commands

```bash
# Run all tests
make test

# Run with coverage report
make test-coverage

# Run with race detection
make test-race

# Run specific test file
go test -v ./function -run TestFileName

# Run specific test case
go test -v ./function -run TestSubscribeToChannel/Success
```

### Coverage Analysis

```bash
# Generate coverage report
make test-coverage

# View coverage in browser
go tool cover -html=function/coverage.out

# Check coverage percentage
go test -cover ./function
```

Current coverage targets:
- Minimum: 80%
- Current: 82.4%
- Goal: 85%

## Test Categories

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
        {"invalid_chars", "UC!@#$%^&*()[]{}|\\<>?/", false},
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
    service := NewOptimizedCloudStorageService()
    
    // Create test state
    state := &SubscriptionState{
        Subscriptions: map[string]*Subscription{
            "UCXuqSBlHAE6Xw-yeJA0Tunw": {
                ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
                Status:    "active",
            },
        },
    }
    
    // Save state
    err := service.SaveSubscriptionState(ctx, state)
    assert.NoError(t, err)
    
    // Load state
    loaded, err := service.LoadSubscriptionState(ctx)
    assert.NoError(t, err)
    assert.Equal(t, 1, len(loaded.Subscriptions))
}
```

### HTTP Handler Tests

Test HTTP endpoints and request/response handling.

**Example: Subscribe Endpoint**
```go
func TestSubscribeEndpoint(t *testing.T) {
    req := httptest.NewRequest("POST", "/subscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw", nil)
    rec := httptest.NewRecorder()
    
    YouTubeWebhook(rec, req)
    
    assert.Equal(t, http.StatusOK, rec.Code)
    
    var response map[string]interface{}
    json.Unmarshal(rec.Body.Bytes(), &response)
    assert.Equal(t, "success", response["status"])
}
```

### Concurrent Access Tests

Test thread safety and race conditions.

**Example: Concurrent State Access**
```go
func TestConcurrentStateAccess(t *testing.T) {
    service := NewOptimizedCloudStorageService()
    ctx := context.Background()
    
    var wg sync.WaitGroup
    errors := make(chan error, 100)
    
    // Concurrent reads
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, err := service.LoadSubscriptionState(ctx)
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
            err := service.SaveSubscriptionState(ctx)
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

## Mocking Strategies

### Using Mock Storage Client

```go
func TestWithMockStorage(t *testing.T) {
    mockClient := NewMockStorageClient()
    
    // Set up mock behavior
    mockClient.LoadError = nil
    mockClient.SaveError = nil
    
    // Inject mock
    SetStorageClient(mockClient)
    defer RestoreStorageClient()
    
    // Test code that uses storage
    // ...
    
    // Verify mock interactions
    assert.Equal(t, 1, mockClient.LoadCallCount)
    assert.Equal(t, 1, mockClient.SaveCallCount)
}
```

### Using stiface for Cloud Storage Mocks

```go
func TestWithStifaceMock(t *testing.T) {
    // Create mock client
    mockClient := &MockStorageClient{
        state: &SubscriptionState{
            Subscriptions: make(map[string]*Subscription),
        },
    }
    
    // Use in tests
    state, err := mockClient.LoadSubscriptionState(context.Background())
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
    // Setup
    originalState := GetTestMode()
    SetTestMode(true)
    
    // Cleanup
    t.Cleanup(func() {
        SetTestMode(originalState)
        ClearTestState()
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
        setupMock   func(*MockStorageClient)
        expectedErr error
    }{
        {
            name: "load_error",
            setupMock: func(m *MockStorageClient) {
                m.LoadError = ErrMockLoadFailure
            },
            expectedErr: ErrMockLoadFailure,
        },
        {
            name: "save_error",
            setupMock: func(m *MockStorageClient) {
                m.SaveError = ErrMockSaveFailure
            },
            expectedErr: ErrMockSaveFailure,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mock := NewMockStorageClient()
            tt.setupMock(mock)
            
            // Test error handling
            err := someFunction(mock)
            assert.Equal(t, tt.expectedErr, err)
        })
    }
}
```

## Performance Testing

### Benchmark Tests

```go
func BenchmarkSubscriptionLoad(b *testing.B) {
    service := NewOptimizedCloudStorageService()
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = service.LoadSubscriptionState(ctx)
    }
}
```

### Load Testing

```go
func TestHighLoad(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }
    
    service := NewOptimizedCloudStorageService()
    ctx := context.Background()
    
    start := time.Now()
    requests := 1000
    
    for i := 0; i < requests; i++ {
        go func() {
            service.LoadSubscriptionState(ctx)
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
    // Save original state
    original := GetGlobalState()
    defer SetGlobalState(original)
    
    // Test logic
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
- All tests pass
- Coverage >80%
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

- Review [Contributing Guide](./contributing.md)
- Learn about [Deployment](../deployment/cloud-functions.md)
- Understand [Monitoring](../operations/monitoring.md)
package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCloudStorageOperations implements CloudStorageOperations for testing
type MockCloudStorageOperations struct {
	objects map[string][]byte
	getErr  error
	putErr  error
	closed  bool
}

func NewMockCloudStorageOperations() *MockCloudStorageOperations {
	return &MockCloudStorageOperations{
		objects: make(map[string][]byte),
	}
}

func (m *MockCloudStorageOperations) GetObject(ctx context.Context, bucket, objectPath string) ([]byte, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	
	key := bucket + "/" + objectPath
	data, exists := m.objects[key]
	if !exists {
		return nil, storage.ErrObjectNotExist
	}
	
	return data, nil
}

func (m *MockCloudStorageOperations) PutObject(ctx context.Context, bucket, objectPath string, data []byte) error {
	if m.putErr != nil {
		return m.putErr
	}
	
	key := bucket + "/" + objectPath
	m.objects[key] = make([]byte, len(data))
	copy(m.objects[key], data)
	return nil
}

func (m *MockCloudStorageOperations) Close() error {
	m.closed = true
	return nil
}

func (m *MockCloudStorageOperations) SetGetError(err error) {
	m.getErr = err
}

func (m *MockCloudStorageOperations) SetPutError(err error) {
	m.putErr = err
}

func (m *MockCloudStorageOperations) Reset() {
	m.objects = make(map[string][]byte)
	m.getErr = nil
	m.putErr = nil
	m.closed = false
}

func TestNewCloudStorageService(t *testing.T) {
	service := NewCloudStorageService()
	
	assert.NotNil(t, service)
	assert.Equal(t, "subscriptions/state.json", service.objectPath)
	assert.Equal(t, 5*time.Minute, service.cacheTTL)
	assert.Nil(t, service.storageOps) // Not initialized yet
	assert.Nil(t, service.cache)
}

func TestNewCloudStorageServiceWithOperations(t *testing.T) {
	mockOps := NewMockCloudStorageOperations()
	service := NewCloudStorageServiceWithOperations(mockOps, "test-bucket")
	
	assert.NotNil(t, service)
	assert.Equal(t, mockOps, service.storageOps)
	assert.Equal(t, "test-bucket", service.bucketName)
	assert.Equal(t, "subscriptions/state.json", service.objectPath)
	assert.Equal(t, 5*time.Minute, service.cacheTTL)
}

func TestCloudStorageService_InitializeErrors(t *testing.T) {
	t.Run("MissingBucketEnvironmentVariable", func(t *testing.T) {
		// Ensure SUBSCRIPTION_BUCKET is not set
		originalBucket := os.Getenv("SUBSCRIPTION_BUCKET")
		os.Unsetenv("SUBSCRIPTION_BUCKET")
		defer func() {
			if originalBucket != "" {
				os.Setenv("SUBSCRIPTION_BUCKET", originalBucket)
			}
		}()

		service := NewCloudStorageService()
		ctx := context.Background()
		
		_, err := service.LoadSubscriptionState(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SUBSCRIPTION_BUCKET environment variable not set")
	})

	t.Run("InitializeOnlyOnce", func(t *testing.T) {
		// Set up environment
		os.Setenv("SUBSCRIPTION_BUCKET", "test-bucket")
		defer os.Unsetenv("SUBSCRIPTION_BUCKET")

		// Use mock operations to avoid real client
		mockOps := NewMockCloudStorageOperations()
		service := NewCloudStorageServiceWithOperations(mockOps, "test-bucket")
		ctx := context.Background()

		// Call initialize multiple times
		err1 := service.initialize(ctx)
		err2 := service.initialize(ctx)
		
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, "test-bucket", service.bucketName)
	})
}

func TestCloudStorageService_CacheOperations(t *testing.T) {
	service := NewCloudStorageService()
	service.cacheTTL = 100 * time.Millisecond // Short TTL for testing

	t.Run("EmptyCache", func(t *testing.T) {
		cached := service.getCachedState()
		assert.Nil(t, cached)
	})

	t.Run("SetAndGetCache", func(t *testing.T) {
		testState := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				"test": {ChannelID: "UCtest", Status: "active"},
			},
		}
		testState.Metadata.Version = "1.0"
		testState.Metadata.LastUpdated = time.Now()

		// Set cache
		service.setCachedState(testState)
		
		// Get from cache
		cached := service.getCachedState()
		require.NotNil(t, cached)
		assert.Equal(t, "UCtest", cached.Subscriptions["test"].ChannelID)
		assert.Equal(t, "1.0", cached.Metadata.Version)
	})

	t.Run("CacheExpiration", func(t *testing.T) {
		testState := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				"test": {ChannelID: "UCtest", Status: "active"},
			},
		}

		// Set cache
		service.setCachedState(testState)
		
		// Should be cached
		cached := service.getCachedState()
		assert.NotNil(t, cached)

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)
		
		// Should be expired
		expired := service.getCachedState()
		assert.Nil(t, expired)
	})
}

func TestCloudStorageService_DeepCopyState(t *testing.T) {
	service := NewCloudStorageService()
	
	originalState := &SubscriptionState{
		Subscriptions: map[string]*Subscription{
			"test1": {ChannelID: "UCtest1", Status: "active"},
			"test2": {ChannelID: "UCtest2", Status: "expired"},
		},
	}
	originalState.Metadata.Version = "1.0"
	originalState.Metadata.LastUpdated = time.Now()

	// Create deep copy
	copied := service.deepCopyState(originalState)
	
	// Verify it's a deep copy
	require.NotNil(t, copied)
	assert.Equal(t, originalState.Metadata.Version, copied.Metadata.Version)
	assert.Equal(t, len(originalState.Subscriptions), len(copied.Subscriptions))
	
	// Modify copy and ensure original is unchanged
	copied.Subscriptions["test1"].Status = "modified"
	assert.Equal(t, "active", originalState.Subscriptions["test1"].Status)
	assert.Equal(t, "modified", copied.Subscriptions["test1"].Status)

	// Test nil input
	nilCopy := service.deepCopyState(nil)
	assert.Nil(t, nilCopy)
}

func TestCloudStorageService_UpdateMetadata(t *testing.T) {
	service := NewCloudStorageService()
	
	state := &SubscriptionState{
		Subscriptions: map[string]*Subscription{},
	}
	
	// Initially empty metadata
	assert.Equal(t, "", state.Metadata.Version)
	assert.True(t, state.Metadata.LastUpdated.IsZero())
	
	// Update metadata
	beforeUpdate := time.Now()
	service.updateMetadata(state)
	afterUpdate := time.Now()
	
	// Verify metadata was updated
	assert.Equal(t, "1.0", state.Metadata.Version)
	assert.True(t, state.Metadata.LastUpdated.After(beforeUpdate))
	assert.True(t, state.Metadata.LastUpdated.Before(afterUpdate))
}

func TestCloudStorageService_LoadSubscriptionState(t *testing.T) {
	t.Run("LoadFromStorage_Success", func(t *testing.T) {
		mockOps := NewMockCloudStorageOperations()
		service := NewCloudStorageServiceWithOperations(mockOps, "test-bucket")

		// Pre-populate storage with test data
		testState := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				"test": {ChannelID: "UCtest", Status: "active"},
			},
		}
		testState.Metadata.Version = "1.0"
		testState.Metadata.LastUpdated = time.Now()
		
		data, err := json.MarshalIndent(testState, "", "  ")
		require.NoError(t, err)
		_ = mockOps.PutObject(context.Background(), "test-bucket", "subscriptions/state.json", data)

		// Load should succeed
		ctx := context.Background()
		loaded, err := service.LoadSubscriptionState(ctx)
		require.NoError(t, err)
		require.NotNil(t, loaded)
		
		// Verify loaded data
		assert.Equal(t, "UCtest", loaded.Subscriptions["test"].ChannelID)
		assert.Equal(t, "1.0", loaded.Metadata.Version)
	})

	t.Run("LoadFromStorage_ObjectNotExist", func(t *testing.T) {
		mockOps := NewMockCloudStorageOperations()
		service := NewCloudStorageServiceWithOperations(mockOps, "test-bucket")

		ctx := context.Background()
		loaded, err := service.LoadSubscriptionState(ctx)
		require.NoError(t, err)
		require.NotNil(t, loaded)
		
		// Should return empty state
		assert.NotNil(t, loaded.Subscriptions)
		assert.Len(t, loaded.Subscriptions, 0)
		assert.Equal(t, "1.0", loaded.Metadata.Version)
	})

	t.Run("LoadFromStorage_Error", func(t *testing.T) {
		mockOps := NewMockCloudStorageOperations()
		mockOps.SetGetError(errors.New("storage error"))
		service := NewCloudStorageServiceWithOperations(mockOps, "test-bucket")

		ctx := context.Background()
		_, err := service.LoadSubscriptionState(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get storage object")
	})

	t.Run("LoadFromCache", func(t *testing.T) {
		mockOps := NewMockCloudStorageOperations()
		service := NewCloudStorageServiceWithOperations(mockOps, "test-bucket")

		// Set up cached state
		cachedState := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				"cached": {ChannelID: "UCcached", Status: "active"},
			},
		}
		cachedState.Metadata.Version = "1.0"
		service.setCachedState(cachedState)

		// Load should return cached version
		ctx := context.Background()
		loaded, err := service.LoadSubscriptionState(ctx)
		require.NoError(t, err)
		require.NotNil(t, loaded)
		
		// Verify it's the cached data
		assert.Equal(t, "UCcached", loaded.Subscriptions["cached"].ChannelID)
		
		// Verify it's a deep copy (modifying loaded shouldn't affect cache)
		loaded.Subscriptions["cached"].Status = "modified"
		cachedAgain := service.getCachedState()
		assert.Equal(t, "active", cachedAgain.Subscriptions["cached"].Status)
	})
}

func TestCloudStorageService_SaveSubscriptionState(t *testing.T) {
	t.Run("SaveToStorage_Success", func(t *testing.T) {
		mockOps := NewMockCloudStorageOperations()
		service := NewCloudStorageServiceWithOperations(mockOps, "test-bucket")

		testState := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				"save-test": {ChannelID: "UCsave", Status: "active"},
			},
		}

		ctx := context.Background()
		err := service.SaveSubscriptionState(ctx, testState)
		assert.NoError(t, err)

		// Verify metadata was updated
		assert.Equal(t, "1.0", testState.Metadata.Version)
		assert.False(t, testState.Metadata.LastUpdated.IsZero())

		// Verify cache was updated
		cached := service.getCachedState()
		require.NotNil(t, cached)
		assert.Equal(t, "UCsave", cached.Subscriptions["save-test"].ChannelID)

		// Verify data was saved to storage
		data, err := mockOps.GetObject(ctx, "test-bucket", "subscriptions/state.json")
		require.NoError(t, err)
		
		var savedState SubscriptionState
		err = json.Unmarshal(data, &savedState)
		require.NoError(t, err)
		assert.Equal(t, "UCsave", savedState.Subscriptions["save-test"].ChannelID)
	})

	t.Run("SaveToStorage_Error", func(t *testing.T) {
		mockOps := NewMockCloudStorageOperations()
		mockOps.SetPutError(errors.New("storage error"))
		service := NewCloudStorageServiceWithOperations(mockOps, "test-bucket")

		testState := &SubscriptionState{
			Subscriptions: map[string]*Subscription{},
		}

		ctx := context.Background()
		err := service.SaveSubscriptionState(ctx, testState)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to put storage object")
	})
}

func TestCloudStorageService_Close(t *testing.T) {
	mockOps := NewMockCloudStorageOperations()
	service := NewCloudStorageServiceWithOperations(mockOps, "test-bucket")
	
	// Set up cache
	testState := &SubscriptionState{
		Subscriptions: map[string]*Subscription{},
	}
	service.setCachedState(testState)
	
	// Verify cache exists
	assert.NotNil(t, service.getCachedState())
	
	// Close service
	err := service.Close()
	assert.NoError(t, err)
	
	// Verify cache is cleared
	service.cacheMutex.RLock()
	assert.Nil(t, service.cache)
	assert.True(t, service.cacheTime.IsZero())
	service.cacheMutex.RUnlock()

	// Verify storage operations were closed
	assert.True(t, mockOps.closed)
}

func TestCloudStorageService_ConcurrentAccess(t *testing.T) {
	mockOps := NewMockCloudStorageOperations()
	service := NewCloudStorageServiceWithOperations(mockOps, "test-bucket")
	
	// Test concurrent cache operations
	done := make(chan bool, 10)
	
	// Start multiple goroutines doing cache operations
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			testState := &SubscriptionState{
				Subscriptions: map[string]*Subscription{},
			}
			testState.Metadata.Version = "1.0"
			
			// Set and get cache concurrently
			service.setCachedState(testState)
			cached := service.getCachedState()
			
			if cached != nil {
				assert.Equal(t, "1.0", cached.Metadata.Version)
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCloudStorageService_InitializationErrorHandling(t *testing.T) {
	service := NewCloudStorageService()
	
	t.Run("ErrorPersistsAcrossMultipleCalls", func(t *testing.T) {
		// Force initialization error by not setting bucket
		originalBucket := os.Getenv("SUBSCRIPTION_BUCKET")
		os.Unsetenv("SUBSCRIPTION_BUCKET")
		defer func() {
			if originalBucket != "" {
				os.Setenv("SUBSCRIPTION_BUCKET", originalBucket)
			}
		}()
		
		ctx := context.Background()
		
		// First call should fail
		_, err1 := service.LoadSubscriptionState(ctx)
		assert.Error(t, err1)
		
		// Second call should fail with same error
		_, err2 := service.LoadSubscriptionState(ctx)  
		assert.Error(t, err2)
		assert.Equal(t, err1.Error(), err2.Error())
	})
}

func TestRealCloudStorageOperations(t *testing.T) {
	// These tests require real Google Cloud Storage credentials and would run against actual GCS
	// They are included to demonstrate how to test the real implementation
	t.Skip("Integration tests require real GCS credentials")
	
	t.Run("NewRealCloudStorageOperations", func(t *testing.T) {
		ctx := context.Background()
		ops, err := NewRealCloudStorageOperations(ctx)
		
		if err != nil {
			// This is expected in CI/testing environments without GCS credentials
			t.Logf("Could not create real storage operations (expected in test): %v", err)
			return
		}
		
		assert.NotNil(t, ops)
		assert.NotNil(t, ops.client)
		
		// Clean up
		err = ops.Close()
		assert.NoError(t, err)
	})
}

func TestCloudStorageService_Integration(t *testing.T) {
	// Integration test that exercises the full flow
	mockOps := NewMockCloudStorageOperations()
	service := NewCloudStorageServiceWithOperations(mockOps, "test-bucket")
	ctx := context.Background()

	// Test full cycle: Load empty -> Save data -> Load data -> Verify cache
	
	// 1. Load should return empty state
	state1, err := service.LoadSubscriptionState(ctx)
	require.NoError(t, err)
	assert.Len(t, state1.Subscriptions, 0)
	
	// 2. Add subscription and save
	state1.Subscriptions["test"] = &Subscription{
		ChannelID: "UCtest",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	
	err = service.SaveSubscriptionState(ctx, state1)
	require.NoError(t, err)
	
	// 3. Clear cache and load again
	service.cacheMutex.Lock()
	service.cache = nil
	service.cacheTime = time.Time{}
	service.cacheMutex.Unlock()
	
	state2, err := service.LoadSubscriptionState(ctx)
	require.NoError(t, err)
	assert.Len(t, state2.Subscriptions, 1)
	assert.Equal(t, "UCtest", state2.Subscriptions["test"].ChannelID)
	
	// 4. Verify cache was populated
	cached := service.getCachedState()
	assert.NotNil(t, cached)
	assert.Equal(t, "UCtest", cached.Subscriptions["test"].ChannelID)
}

func TestLegacyStorageService(t *testing.T) {
	t.Run("CreatesWithCloudStorageService", func(t *testing.T) {
		legacy := NewLegacyStorageService()
		
		assert.NotNil(t, legacy)
		assert.NotNil(t, legacy.optimized)
		assert.IsType(t, &CloudStorageService{}, legacy.optimized)
	})

	t.Run("DelegatesToCloudStorageService", func(t *testing.T) {
		// Create legacy service with mocked underlying service
		legacy := &LegacyStorageService{
			optimized: NewCloudStorageServiceWithOperations(
				NewMockCloudStorageOperations(), 
				"test-bucket",
			),
		}
		
		ctx := context.Background()

		// Test LoadSubscriptionState delegation
		_, err := legacy.LoadSubscriptionState(ctx)
		assert.NoError(t, err)

		// Test SaveSubscriptionState delegation
		testState := &SubscriptionState{
			Subscriptions: map[string]*Subscription{},
		}
		err = legacy.SaveSubscriptionState(ctx, testState)
		assert.NoError(t, err)
	})
}
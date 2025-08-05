package webhook

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestStorageServiceNonTestMode tests storage service functionality in non-test mode
// These tests will require Cloud Storage emulator or mock setup for full coverage
func TestStorageServiceNonTestMode(t *testing.T) {
	// Skip these tests in CI/CD environments or when Cloud Storage isn't available
	if testing.Short() {
		t.Skip("Skipping non-test mode storage tests in short mode")
	}

	// Set up environment
	originalBucket := os.Getenv("SUBSCRIPTION_BUCKET")
	originalTestMode := testMode
	
	os.Setenv("SUBSCRIPTION_BUCKET", "test-bucket")
	testMode = false
	
	defer func() {
		if originalBucket == "" {
			os.Unsetenv("SUBSCRIPTION_BUCKET")
		} else {
			os.Setenv("SUBSCRIPTION_BUCKET", originalBucket)
		}
		testMode = originalTestMode
	}()

	t.Run("initialize_with_missing_bucket", func(t *testing.T) {
		os.Unsetenv("SUBSCRIPTION_BUCKET")
		
		service := NewOptimizedCloudStorageService()
		service.testMode = false
		
		err := service.initialize(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SUBSCRIPTION_BUCKET environment variable not set")
	})

	t.Run("initialize_with_valid_bucket", func(t *testing.T) {
		os.Setenv("SUBSCRIPTION_BUCKET", "test-bucket")
		
		service := NewOptimizedCloudStorageService()
		service.testMode = false
		
		// This will fail in test environments without Cloud Storage access
		// but it tests the initialization path
		err := service.initialize(context.Background())
		
		// We expect either success or a Cloud Storage connection error
		// Both are valid outcomes depending on the test environment
		if err != nil {
			// Should be a storage client creation error, not a missing bucket error
			assert.NotContains(t, err.Error(), "SUBSCRIPTION_BUCKET environment variable not set")
			assert.Contains(t, err.Error(), "failed to create storage client")
		} else {
			// If successful, client should be set
			assert.NotNil(t, service.client)
		}
	})

	t.Run("load_subscription_state_non_test_mode", func(t *testing.T) {
		os.Setenv("SUBSCRIPTION_BUCKET", "test-bucket")
		
		service := NewOptimizedCloudStorageService()
		service.testMode = false
		
		// This will attempt to initialize the storage client
		// In most test environments, this will fail with a connection error
		// which is expected behavior
		_, err := service.LoadSubscriptionState(context.Background())
		
		// We expect either success or an initialization/connection error
		if err != nil {
			// Should be either initialization error or storage access error
			assert.True(t, 
				err.Error() == "SUBSCRIPTION_BUCKET environment variable not set" ||
				strings.Contains(err.Error(), "failed to create storage client") ||
				strings.Contains(err.Error(), "failed to open storage object"),
				"Unexpected error: %v", err)
		}
	})

	t.Run("save_subscription_state_non_test_mode", func(t *testing.T) {
		os.Setenv("SUBSCRIPTION_BUCKET", "test-bucket")
		
		service := NewOptimizedCloudStorageService()
		service.testMode = false
		
		state := &SubscriptionState{
			Subscriptions: make(map[string]*Subscription),
			Metadata: struct {
				LastUpdated time.Time `json:"last_updated"`
				Version     string    `json:"version"`
			}{
				Version: "1.0",
			},
		}
		
		// This will attempt to initialize and save to storage
		err := service.SaveSubscriptionState(context.Background(), state)
		
		// We expect either success or an initialization/connection error
		if err != nil {
			// Should be either initialization error or storage access error
			assert.True(t, 
				err.Error() == "SUBSCRIPTION_BUCKET environment variable not set" ||
				strings.Contains(err.Error(), "failed to create storage client") ||
				strings.Contains(err.Error(), "failed to marshal state") ||
				strings.Contains(err.Error(), "failed to write state data"),
				"Unexpected error: %v", err)
		}
	})
}


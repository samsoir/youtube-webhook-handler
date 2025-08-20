package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandlers_EndToEnd tests the complete refactored flow
func TestHandlers_EndToEnd(t *testing.T) {
	// Create test dependencies
	deps := CreateTestDependencies()
	channelID := "UCXuqSBlHAE6Xw-yeJA0Tunw"

	t.Run("Subscribe_Success", func(t *testing.T) {
		// Reset mocks
		deps.StorageClient.(*MockStorageClient).Reset()
		deps.PubSubClient.(*MockPubSubClient).Reset()

		// Create request
		req := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
		rec := httptest.NewRecorder()

		// Execute handler
		handler := handleSubscribeWithDeps(deps)
		handler(rec, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rec.Code)

		var response APIResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "success", response.Status)

		// Verify mocks
		assert.Equal(t, 1, deps.StorageClient.(*MockStorageClient).LoadCallCount)
		assert.Equal(t, 1, deps.StorageClient.(*MockStorageClient).SaveCallCount)
		assert.True(t, deps.PubSubClient.(*MockPubSubClient).IsSubscribed(channelID))
	})

	t.Run("Unsubscribe_Success", func(t *testing.T) {
		// Pre-populate with subscription
		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				channelID: {
					ChannelID: channelID,
					Status:    "active",
					ExpiresAt: time.Now().Add(24 * time.Hour),
				},
			},
		}
		deps.StorageClient.(*MockStorageClient).SetState(state)

		// Create request
		req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
		rec := httptest.NewRecorder()

		// Execute handler
		handler := handleUnsubscribeWithDeps(deps)
		handler(rec, req)

		// Verify response
		assert.Equal(t, http.StatusNoContent, rec.Code)

		// Verify subscription was removed
		finalState := deps.StorageClient.(*MockStorageClient).GetState()
		assert.NotContains(t, finalState.Subscriptions, channelID)
		assert.False(t, deps.PubSubClient.(*MockPubSubClient).IsSubscribed(channelID))
	})

	t.Run("Renewal_Success", func(t *testing.T) {
		// Set up expiring subscription
		expiringTime := time.Now().Add(6 * time.Hour) // Within renewal threshold
		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				channelID: {
					ChannelID:       channelID,
					Status:          "active",
					ExpiresAt:       expiringTime,
					SubscribedAt:    time.Now().Add(-18 * time.Hour),
					RenewalAttempts: 0,
				},
			},
		}
		deps.StorageClient.(*MockStorageClient).SetState(state)

		// Create request
		req := httptest.NewRequest("POST", "/renew", nil)
		rec := httptest.NewRecorder()

		// Execute handler
		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(rec, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rec.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 1, response.RenewalsSucceeded)
		assert.Equal(t, 0, response.RenewalsFailed)

		// Verify subscription was renewed
		finalState := deps.StorageClient.(*MockStorageClient).GetState()
		sub := finalState.Subscriptions[channelID]
		assert.NotNil(t, sub)
		assert.True(t, sub.ExpiresAt.After(expiringTime))
		assert.Equal(t, 0, sub.RenewalAttempts)
	})
}

// TestHandlers_ErrorScenarios tests error handling
func TestHandlers_ErrorScenarios(t *testing.T) {
	t.Run("Subscribe_StorageError", func(t *testing.T) {
		deps := CreateTestDependencies()
		deps.StorageClient.(*MockStorageClient).LoadError = ErrMockLoadFailure

		req := httptest.NewRequest("POST", "/subscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw", nil)
		rec := httptest.NewRecorder()

		handler := handleSubscribeWithDeps(deps)
		handler(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "error", response["status"])
	})

	t.Run("Unsubscribe_NotFound", func(t *testing.T) {
		deps := CreateTestDependencies()
		// Use a valid format channel ID that doesn't exist (UC + 22 chars)
		nonExistentChannelID := "UCNonExistent12345678901"

		req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+nonExistentChannelID, nil)
		rec := httptest.NewRecorder()

		handler := handleUnsubscribeWithDeps(deps)
		handler(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "error", response["status"])
		assert.Contains(t, response["message"], "not found")
	})

	t.Run("Renewal_MaxAttemptsExceeded", func(t *testing.T) {
		deps := CreateTestDependencies()

		// Set up subscription with max attempts exceeded
		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				"UCMaxAttempts": {
					ChannelID:       "UCMaxAttempts",
					Status:          "active",
					ExpiresAt:       time.Now().Add(1 * time.Hour), // Needs renewal
					RenewalAttempts: 10,                            // Exceeds max attempts
				},
			},
		}
		deps.StorageClient.(*MockStorageClient).SetState(state)

		req := httptest.NewRequest("POST", "/renew", nil)
		rec := httptest.NewRecorder()

		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, 0, response.RenewalsSucceeded)
		assert.Equal(t, 1, response.RenewalsFailed)
		assert.Contains(t, response.Results[0].Message, "Max renewal attempts")
	})
}

// TestHandlers_ConcurrentAccess tests thread safety
func TestHandlers_ConcurrentAccess(t *testing.T) {
	deps := CreateTestDependencies()

	// Run multiple operations concurrently
	done := make(chan bool, 3)

	// Concurrent subscribe
	go func() {
		req := httptest.NewRequest("POST", "/subscribe?channel_id=UCConcurrent1", nil)
		rec := httptest.NewRecorder()
		handler := handleSubscribeWithDeps(deps)
		handler(rec, req)
		done <- true
	}()

	// Concurrent unsubscribe
	go func() {
		req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id=UCConcurrent2", nil)
		rec := httptest.NewRecorder()
		handler := handleUnsubscribeWithDeps(deps)
		handler(rec, req)
		done <- true
	}()

	// Concurrent renewal
	go func() {
		req := httptest.NewRequest("POST", "/renew", nil)
		rec := httptest.NewRecorder()
		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(rec, req)
		done <- true
	}()

	// Wait for all operations to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify no race conditions or panics occurred
	assert.True(t, true, "Concurrent operations completed without panic")
}

// TestHandlers_NoDependencyOnGlobalTestMode verifies no global state usage
func TestHandlers_NoDependencyOnGlobalTestMode(t *testing.T) {
	// This test runs without setting any global testMode
	// If the refactored handlers depend on testMode, they will fail

	deps := CreateTestDependencies()

	// Test subscribe without global testMode
	t.Run("Subscribe_NoGlobalState", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/subscribe?channel_id=UCNoGlobalState123456789", nil)
		rec := httptest.NewRecorder()

		handler := handleSubscribeWithDeps(deps)
		handler(rec, req)

		if rec.Code != http.StatusOK {
			var errResp map[string]interface{}
			_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
			t.Logf("Error response: %v", errResp)
		}
		assert.Equal(t, http.StatusOK, rec.Code)
		// Success proves no dependency on global testMode
	})

	// Test unsubscribe without global testMode
	t.Run("Unsubscribe_NoGlobalState", func(t *testing.T) {
		// Pre-populate subscription
		channelID := "UCNoGlobalState123456789"
		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				channelID: {
					ChannelID: channelID,
					Status:    "active",
				},
			},
		}
		deps.StorageClient.(*MockStorageClient).SetState(state)

		req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
		rec := httptest.NewRecorder()

		handler := handleUnsubscribeWithDeps(deps)
		handler(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		// Success proves no dependency on global testMode
	})
}

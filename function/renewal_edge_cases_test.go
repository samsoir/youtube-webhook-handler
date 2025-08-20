package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenewalWithDeps_EdgeCases tests various edge cases for the renewal handler using dependency injection
func TestRenewalWithDeps_EdgeCases(t *testing.T) {
	t.Run("NoSubscriptions", func(t *testing.T) {
		deps := CreateTestDependencies()

		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 0, response.TotalChecked)
		assert.Equal(t, 0, response.RenewalsCandidates)
		assert.Equal(t, 0, response.RenewalsSucceeded)
		assert.Equal(t, 0, response.RenewalsFailed)
		assert.Empty(t, response.Results)
	})

	t.Run("NoSubscriptionsNeedRenewal", func(t *testing.T) {
		deps := CreateTestDependencies()

		// Create subscriptions that are not expiring soon
		now := time.Now()
		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				"UC1": createTestSubscriptionWithExpiry("UC1", now.Add(48*time.Hour)),
				"UC2": createTestSubscriptionWithExpiry("UC2", now.Add(24*time.Hour)),
				"UC3": createTestSubscriptionWithExpiry("UC3", now.Add(36*time.Hour)),
			},
		}
		deps.StorageClient.(*MockStorageClient).SetState(state)

		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 3, response.TotalChecked)
		assert.Equal(t, 0, response.RenewalsCandidates)
		assert.Equal(t, 0, response.RenewalsSucceeded)
		assert.Equal(t, 0, response.RenewalsFailed)
	})

	t.Run("MaxAttemptsExceeded", func(t *testing.T) {
		deps := CreateTestDependencies()

		// Create subscription with max attempts already exceeded
		now := time.Now()
		expiringSub := createTestSubscriptionWithExpiry("UCMaxAttempts", now.Add(1*time.Hour))
		expiringSub.RenewalAttempts = 10 // Exceeds default max of 3

		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				"UCMaxAttempts": expiringSub,
			},
		}
		deps.StorageClient.(*MockStorageClient).SetState(state)

		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 1, response.TotalChecked)
		assert.Equal(t, 1, response.RenewalsCandidates)
		assert.Equal(t, 0, response.RenewalsSucceeded)
		assert.Equal(t, 1, response.RenewalsFailed)
		
		require.Len(t, response.Results, 1)
		result := response.Results[0]
		assert.Equal(t, "UCMaxAttempts", result.ChannelID)
		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "Max renewal attempts")
		assert.Equal(t, 10, result.AttemptCount)
	})

	t.Run("PubSubRenewalFailure", func(t *testing.T) {
		deps := CreateTestDependencies()

		// Set PubSub to fail subscriptions
		mockPubSub := deps.PubSubClient.(*MockPubSubClient)
		mockPubSub.SetSubscribeError(fmt.Errorf("PubSubHubbub server error"))

		// Create subscription that needs renewal
		now := time.Now()
		expiringSub := createTestSubscriptionWithExpiry("UCPubSubFail", now.Add(1*time.Hour))
		expiringSub.RenewalAttempts = 1

		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				"UCPubSubFail": expiringSub,
			},
		}
		deps.StorageClient.(*MockStorageClient).SetState(state)

		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 1, response.RenewalsFailed)
		assert.Equal(t, 0, response.RenewalsSucceeded)
		
		require.Len(t, response.Results, 1)
		result := response.Results[0]
		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "PubSubHubbub renewal failed")
		assert.Equal(t, 2, result.AttemptCount) // Should increment

		// Verify attempt count was incremented in storage
		finalState := deps.StorageClient.(*MockStorageClient).GetState()
		sub := finalState.Subscriptions["UCPubSubFail"]
		assert.Equal(t, 2, sub.RenewalAttempts)
	})

	t.Run("SuccessfulRenewal", func(t *testing.T) {
		deps := CreateTestDependencies()

		// Create subscription that needs renewal
		now := time.Now()
		expiringSub := createTestSubscriptionWithExpiry("UCSuccess", now.Add(6*time.Hour))
		originalExpiryTime := expiringSub.ExpiresAt
		expiringSub.RenewalAttempts = 1

		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				"UCSuccess": expiringSub,
			},
		}
		deps.StorageClient.(*MockStorageClient).SetState(state)

		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 1, response.RenewalsSucceeded)
		assert.Equal(t, 0, response.RenewalsFailed)
		
		require.Len(t, response.Results, 1)
		result := response.Results[0]
		assert.True(t, result.Success)
		assert.Contains(t, result.Message, "Successfully renewed")
		assert.Equal(t, 0, result.AttemptCount) // Reset on success
		assert.NotEmpty(t, result.NewExpiryTime)

		// Verify subscription was updated in storage
		finalState := deps.StorageClient.(*MockStorageClient).GetState()
		sub := finalState.Subscriptions["UCSuccess"]
		assert.Equal(t, 0, sub.RenewalAttempts) // Should reset
		assert.True(t, sub.ExpiresAt.After(originalExpiryTime)) // Should extend
		assert.True(t, sub.LastRenewal.After(now.Add(-1*time.Minute))) // Should update
	})

	t.Run("MixedResults", func(t *testing.T) {
		deps := CreateTestDependencies()

		now := time.Now()
		
		// One will succeed
		successSub := createTestSubscriptionWithExpiry("UCSuccess", now.Add(6*time.Hour))
		successSub.RenewalAttempts = 0

		// One will fail due to max attempts
		maxAttemptsSub := createTestSubscriptionWithExpiry("UCMaxAttempts", now.Add(5*time.Hour))
		maxAttemptsSub.RenewalAttempts = 5 // Exceeds max

		// One will fail due to PubSub error
		pubsubFailSub := createTestSubscriptionWithExpiry("UCPubSubFail", now.Add(4*time.Hour))
		pubsubFailSub.RenewalAttempts = 1

		// One doesn't need renewal
		healthySub := createTestSubscriptionWithExpiry("UCHealthy", now.Add(48*time.Hour))

		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				"UCSuccess":      successSub,
				"UCMaxAttempts":  maxAttemptsSub,
				"UCPubSubFail":   pubsubFailSub,
				"UCHealthy":      healthySub,
			},
		}
		deps.StorageClient.(*MockStorageClient).SetState(state)

		// Set PubSub to fail for all channels
		mockPubSub := deps.PubSubClient.(*MockPubSubClient)
		mockPubSub.SetSubscribeError(fmt.Errorf("server error"))

		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 4, response.TotalChecked)
		assert.Equal(t, 3, response.RenewalsCandidates) // 3 need renewal
		assert.Equal(t, 0, response.RenewalsSucceeded)  // All fail due to PubSub error
		assert.Equal(t, 3, response.RenewalsFailed)     // All 3 candidates fail
		assert.Len(t, response.Results, 3)

		// Verify each result - all should fail due to PubSub error except max attempts
		resultsByChannel := make(map[string]RenewalResult)
		for _, result := range response.Results {
			resultsByChannel[result.ChannelID] = result
		}

		// All results should be failures now
		for channelID, result := range resultsByChannel {
			assert.False(t, result.Success, "Channel %s should fail", channelID)
			if channelID == "UCMaxAttempts" {
				assert.Contains(t, result.Message, "Max renewal attempts")
			} else {
				assert.Contains(t, result.Message, "PubSubHubbub renewal failed")
			}
		}
	})

	t.Run("StorageErrors", func(t *testing.T) {
		t.Run("LoadError", func(t *testing.T) {
			deps := CreateTestDependencies()
			deps.StorageClient.(*MockStorageClient).LoadError = fmt.Errorf("storage unavailable")

			req := httptest.NewRequest("POST", "/renew", nil)
			w := httptest.NewRecorder()

			handler := handleRenewSubscriptionsWithDeps(deps)
			handler(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code)

			var response APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "error", response.Status)
			assert.Contains(t, response.Message, "Failed to load subscription state")
		})

		t.Run("SaveError", func(t *testing.T) {
			deps := CreateTestDependencies()

			// Create subscription that needs renewal
			now := time.Now()
			expiringSub := createTestSubscriptionWithExpiry("UCSaveError", now.Add(1*time.Hour))
			state := &SubscriptionState{
				Subscriptions: map[string]*Subscription{
					"UCSaveError": expiringSub,
				},
			}
			deps.StorageClient.(*MockStorageClient).SetState(state)

			// Make save fail
			deps.StorageClient.(*MockStorageClient).SaveError = fmt.Errorf("storage write failed")

			req := httptest.NewRequest("POST", "/renew", nil)
			w := httptest.NewRecorder()

			handler := handleRenewSubscriptionsWithDeps(deps)
			handler(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code)

			var response APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "error", response.Status)
			assert.Contains(t, response.Message, "Failed to save subscription state")
		})
	})
}

// Helper function to create subscription with specific expiry time
func createTestSubscriptionWithExpiry(channelID string, expiresAt time.Time) *Subscription {
	now := time.Now()
	return &Subscription{
		ChannelID:       channelID,
		TopicURL:        "https://www.youtube.com/feeds/videos.xml?channel_id=" + channelID,
		CallbackURL:     "https://test-function-url",
		Status:          "active",
		LeaseSeconds:    86400,
		SubscribedAt:    now.Add(-12 * time.Hour),
		ExpiresAt:       expiresAt,
		LastRenewal:     now.Add(-12 * time.Hour),
		RenewalAttempts: 0,
		HubResponse:     "202 Accepted",
	}
}
package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHandleRenewSubscriptions(t *testing.T) {
	// Setup environment variables
	os.Setenv("RENEWAL_THRESHOLD_HOURS", "12")
	os.Setenv("MAX_RENEWAL_ATTEMPTS", "3")
	os.Setenv("SUBSCRIPTION_LEASE_SECONDS", "86400")
	os.Setenv("FUNCTION_URL", "https://test-function-url")
	defer func() {
		os.Unsetenv("RENEWAL_THRESHOLD_HOURS")
		os.Unsetenv("MAX_RENEWAL_ATTEMPTS")
		os.Unsetenv("SUBSCRIPTION_LEASE_SECONDS")
		os.Unsetenv("FUNCTION_URL")
	}()

	t.Run("successful_renewal_of_expiring_subscriptions", func(t *testing.T) {
		// Create test dependencies
		deps := CreateTestDependencies()

		// Setup test subscription state with expiring subscriptions
		now := time.Now()
		expiringSubscription := &Subscription{
			ChannelID:       "UCXuqSBlHAE6Xw-yeJA0Tunw",
			TopicURL:        "https://www.youtube.com/feeds/videos.xml?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw",
			CallbackURL:     "https://test-function-url",
			Status:          "active",
			LeaseSeconds:    86400,
			SubscribedAt:    now.Add(-20 * time.Hour),
			ExpiresAt:       now.Add(6 * time.Hour), // Expires in 6 hours (within 12h threshold)
			LastRenewal:     now.Add(-20 * time.Hour),
			RenewalAttempts: 0,
			HubResponse:     "202 Accepted",
		}

		validSubscription := &Subscription{
			ChannelID:       "UCBJycsmduvYEL83R_U4JriQ",
			TopicURL:        "https://www.youtube.com/feeds/videos.xml?channel_id=UCBJycsmduvYEL83R_U4JriQ",
			CallbackURL:     "https://test-function-url",
			Status:          "active",
			LeaseSeconds:    86400,
			SubscribedAt:    now.Add(-2 * time.Hour),
			ExpiresAt:       now.Add(20 * time.Hour), // Expires in 20 hours (outside 12h threshold)
			LastRenewal:     now.Add(-2 * time.Hour),
			RenewalAttempts: 0,
			HubResponse:     "202 Accepted",
		}

		// Create subscription state
		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				expiringSubscription.ChannelID: expiringSubscription,
				validSubscription.ChannelID:    validSubscription,
			},
		}
		state.Metadata.LastUpdated = now
		state.Metadata.Version = "1.0"

		// Set the state in our mock storage
		deps.StorageClient.(*MockStorageClient).SetState(state)

		// Create request
		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		// Use dependency injection handler
		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 2, response.TotalChecked)
		assert.Equal(t, 1, response.RenewalsCandidates)
		assert.Equal(t, 1, response.RenewalsSucceeded)
		assert.Equal(t, 0, response.RenewalsFailed)
		assert.Len(t, response.Results, 1)

		// Verify the expiring subscription was renewed
		result := response.Results[0]
		assert.Equal(t, expiringSubscription.ChannelID, result.ChannelID)
		assert.True(t, result.Success)
		assert.NotEmpty(t, result.NewExpiryTime)
		assert.Equal(t, 0, result.AttemptCount)
	})

	t.Run("no_subscriptions_need_renewal", func(t *testing.T) {
		// Create test dependencies
		deps := CreateTestDependencies()

		// Setup test subscription state with all valid subscriptions
		now := time.Now()
		validSubscription := &Subscription{
			ChannelID:       "UCXuqSBlHAE6Xw-yeJA0Tunw",
			Status:          "active",
			ExpiresAt:       now.Add(20 * time.Hour), // Expires in 20 hours (outside threshold)
			RenewalAttempts: 0,
		}

		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				validSubscription.ChannelID: validSubscription,
			},
		}
		state.Metadata.LastUpdated = now
		state.Metadata.Version = "1.0"

		deps.StorageClient.(*MockStorageClient).SetState(state)

		// Create request
		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		// Use dependency injection handler
		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 1, response.TotalChecked)
		assert.Equal(t, 0, response.RenewalsCandidates)
		assert.Equal(t, 0, response.RenewalsSucceeded)
		assert.Equal(t, 0, response.RenewalsFailed)
		assert.Len(t, response.Results, 0)
	})

	t.Run("renewal_failure_due_to_pubsub_error", func(t *testing.T) {
		// Create test dependencies with PubSub error
		deps := CreateTestDependencies()
		mockPubSub := deps.PubSubClient.(*MockPubSubClient)
		mockPubSub.SetSubscribeError(assert.AnError)

		// Setup test subscription state with expiring subscription
		now := time.Now()
		expiringSubscription := &Subscription{
			ChannelID:       "UCXuqSBlHAE6Xw-yeJA0Tunw",
			Status:          "active",
			ExpiresAt:       now.Add(6 * time.Hour), // Expires in 6 hours (within threshold)
			RenewalAttempts: 0,
		}

		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				expiringSubscription.ChannelID: expiringSubscription,
			},
		}
		state.Metadata.LastUpdated = now
		state.Metadata.Version = "1.0"

		deps.StorageClient.(*MockStorageClient).SetState(state)

		// Create request
		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		// Use dependency injection handler
		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 1, response.TotalChecked)
		assert.Equal(t, 1, response.RenewalsCandidates)
		assert.Equal(t, 0, response.RenewalsSucceeded)
		assert.Equal(t, 1, response.RenewalsFailed)
		assert.Len(t, response.Results, 1)

		// Verify the renewal failed
		result := response.Results[0]
		assert.Equal(t, expiringSubscription.ChannelID, result.ChannelID)
		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "PubSubHubbub renewal failed")
		assert.Equal(t, 1, result.AttemptCount)
	})

	t.Run("max_renewal_attempts_exceeded", func(t *testing.T) {
		// Create test dependencies
		deps := CreateTestDependencies()

		// Setup test subscription state with subscription that has max attempts
		now := time.Now()
		failedSubscription := &Subscription{
			ChannelID:       "UCXuqSBlHAE6Xw-yeJA0Tunw",
			Status:          "active",
			ExpiresAt:       now.Add(6 * time.Hour), // Expires in 6 hours (within threshold)
			RenewalAttempts: 3,                      // Max attempts reached
		}

		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				failedSubscription.ChannelID: failedSubscription,
			},
		}
		state.Metadata.LastUpdated = now
		state.Metadata.Version = "1.0"

		deps.StorageClient.(*MockStorageClient).SetState(state)

		// Create request
		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		// Use dependency injection handler
		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 1, response.TotalChecked)
		assert.Equal(t, 1, response.RenewalsCandidates)
		assert.Equal(t, 0, response.RenewalsSucceeded)
		assert.Equal(t, 1, response.RenewalsFailed)
		assert.Len(t, response.Results, 1)

		// Verify the renewal failed due to max attempts
		result := response.Results[0]
		assert.Equal(t, failedSubscription.ChannelID, result.ChannelID)
		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "Max renewal attempts (3) exceeded")
		assert.Equal(t, 3, result.AttemptCount)
	})

	t.Run("empty_subscription_state", func(t *testing.T) {
		// Create test dependencies with empty state
		deps := CreateTestDependencies()

		// Create request
		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		// Use dependency injection handler
		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 0, response.TotalChecked)
		assert.Equal(t, 0, response.RenewalsCandidates)
		assert.Equal(t, 0, response.RenewalsSucceeded)
		assert.Equal(t, 0, response.RenewalsFailed)
		assert.Len(t, response.Results, 0)
	})

	t.Run("storage_load_error", func(t *testing.T) {
		// Create test dependencies with storage error
		deps := CreateTestDependencies()
		deps.StorageClient.(*MockStorageClient).LoadError = assert.AnError

		// Create request
		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		// Use dependency injection handler
		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(w, req)

		// Verify response
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		body := w.Body.String()
		assert.Contains(t, body, "Failed to load subscription state")
	})

	t.Run("storage_save_error", func(t *testing.T) {
		// Create test dependencies with storage save error
		deps := CreateTestDependencies()
		deps.StorageClient.(*MockStorageClient).SaveError = assert.AnError

		// Setup test subscription state with expiring subscription
		now := time.Now()
		expiringSubscription := &Subscription{
			ChannelID:       "UCXuqSBlHAE6Xw-yeJA0Tunw",
			Status:          "active",
			ExpiresAt:       now.Add(6 * time.Hour), // Expires in 6 hours (within threshold)
			RenewalAttempts: 0,
		}

		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				expiringSubscription.ChannelID: expiringSubscription,
			},
		}
		state.Metadata.LastUpdated = now
		state.Metadata.Version = "1.0"

		deps.StorageClient.(*MockStorageClient).SetState(state)
		// Set error after setting state so Load works but Save fails
		deps.StorageClient.(*MockStorageClient).SaveError = assert.AnError

		// Create request
		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		// Use dependency injection handler
		handler := handleRenewSubscriptionsWithDeps(deps)
		handler(w, req)

		// Verify response
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		body := w.Body.String()
		assert.Contains(t, body, "Failed to save subscription state")
	})
}

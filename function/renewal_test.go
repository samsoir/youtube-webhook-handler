package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHandleRenewSubscriptions(t *testing.T) {
	// Setup test environment
	originalTestMode := GetTestMode()
	SetTestMode(true)
	defer SetTestMode(originalTestMode)

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
			ExpiresAt:       now.Add(22 * time.Hour), // Expires in 22 hours (beyond 12h threshold)
			LastRenewal:     now.Add(-2 * time.Hour),
			RenewalAttempts: 0,
			HubResponse:     "202 Accepted",
		}

		testState := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				expiringSubscription.ChannelID: expiringSubscription,
				validSubscription.ChannelID:    validSubscription,
			},
			Metadata: struct {
				LastUpdated time.Time `json:"last_updated"`
				Version     string    `json:"version"`
			}{
				LastUpdated: now,
				Version:     "1.0",
			},
		}

		SetTestSubscriptionState(testState)

		// Create request
		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		// Execute
		handleRenewSubscriptions(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 2, response.TotalChecked)
		assert.Equal(t, 1, response.RenewalsCandidates) // Only one subscription needs renewal
		assert.Equal(t, 1, response.RenewalsSucceeded)
		assert.Equal(t, 0, response.RenewalsFailed)
		assert.Len(t, response.Results, 1)

		// Verify the renewed subscription
		renewalResult := response.Results[0]
		assert.Equal(t, expiringSubscription.ChannelID, renewalResult.ChannelID)
		assert.True(t, renewalResult.Success)
		assert.Equal(t, "Subscription renewed successfully", renewalResult.Message)
		assert.NotEmpty(t, renewalResult.NewExpiryTime)
		assert.Equal(t, 0, renewalResult.AttemptCount)
	})

	t.Run("no_subscriptions_need_renewal", func(t *testing.T) {
		// Setup test subscription state with valid subscriptions
		now := time.Now()
		validSubscription := &Subscription{
			ChannelID:       "UCXuqSBlHAE6Xw-yeJA0Tunw",
			ExpiresAt:       now.Add(24 * time.Hour), // Expires in 24 hours
			RenewalAttempts: 0,
		}

		testState := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				validSubscription.ChannelID: validSubscription,
			},
		}

		SetTestSubscriptionState(testState)

		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		handleRenewSubscriptions(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 1, response.TotalChecked)
		assert.Equal(t, 0, response.RenewalsCandidates)
		assert.Equal(t, 0, response.RenewalsSucceeded)
		assert.Equal(t, 0, response.RenewalsFailed)
		assert.Empty(t, response.Results)
	})

	t.Run("renewal_failure_max_attempts_exceeded", func(t *testing.T) {
		// Setup subscription that has exceeded max attempts
		now := time.Now()
		failedSubscription := &Subscription{
			ChannelID:       "UCXuqSBlHAE6Xw-yeJA0Tunw",
			ExpiresAt:       now.Add(6 * time.Hour), // Within renewal threshold
			RenewalAttempts: 3,                      // Already at max attempts
		}

		testState := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				failedSubscription.ChannelID: failedSubscription,
			},
		}

		SetTestSubscriptionState(testState)

		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		handleRenewSubscriptions(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 1, response.RenewalsCandidates)
		assert.Equal(t, 0, response.RenewalsSucceeded)
		assert.Equal(t, 1, response.RenewalsFailed)

		renewalResult := response.Results[0]
		assert.False(t, renewalResult.Success)
		assert.Contains(t, renewalResult.Message, "Max renewal attempts")
		assert.Equal(t, 3, renewalResult.AttemptCount)
	})

	t.Run("storage_error_loading_state", func(t *testing.T) {
		// Clear bucket environment to trigger error
		originalBucket := os.Getenv("SUBSCRIPTION_BUCKET")
		os.Unsetenv("SUBSCRIPTION_BUCKET")
		defer func() {
			if originalBucket != "" {
				os.Setenv("SUBSCRIPTION_BUCKET", originalBucket)
			}
		}()

		// Switch to non-test mode to trigger storage error
		SetTestMode(false)
		defer SetTestMode(true)

		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		handleRenewSubscriptions(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "error", response.Status)
		assert.Contains(t, response.Message, "Failed to load subscription state")
	})
}

func TestRenewSubscription(t *testing.T) {
	// Setup test environment
	originalTestMode := GetTestMode()
	SetTestMode(true)
	defer SetTestMode(originalTestMode)

	os.Setenv("FUNCTION_URL", "https://test-function-url")
	os.Setenv("MAX_RENEWAL_ATTEMPTS", "3")
	os.Setenv("SUBSCRIPTION_LEASE_SECONDS", "86400")
	defer func() {
		os.Unsetenv("FUNCTION_URL")
		os.Unsetenv("MAX_RENEWAL_ATTEMPTS")
		os.Unsetenv("SUBSCRIPTION_LEASE_SECONDS")
	}()

	t.Run("successful_renewal", func(t *testing.T) {
		ctx := context.Background()
		now := time.Now()
		
		subscription := &Subscription{
			ChannelID:       "UCXuqSBlHAE6Xw-yeJA0Tunw",
			ExpiresAt:       now.Add(6 * time.Hour),
			LastRenewal:     now.Add(-18 * time.Hour),
			RenewalAttempts: 1,
		}

		state := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				subscription.ChannelID: subscription,
			},
		}

		result := renewSubscription(ctx, subscription.ChannelID, subscription, state)

		assert.True(t, result.Success)
		assert.Equal(t, subscription.ChannelID, result.ChannelID)
		assert.Equal(t, "Subscription renewed successfully", result.Message)
		assert.NotEmpty(t, result.NewExpiryTime)
		assert.Equal(t, 0, result.AttemptCount)

		// Verify subscription was updated
		assert.Equal(t, 0, subscription.RenewalAttempts)
		assert.True(t, subscription.ExpiresAt.After(now.Add(23*time.Hour))) // Should be ~24 hours from now
		assert.True(t, subscription.LastRenewal.After(now.Add(-1*time.Minute))) // Should be recent
		assert.Equal(t, "202 Accepted (Renewed)", subscription.HubResponse)
	})

	t.Run("max_attempts_exceeded", func(t *testing.T) {
		ctx := context.Background()
		
		subscription := &Subscription{
			ChannelID:       "UCXuqSBlHAE6Xw-yeJA0Tunw",
			RenewalAttempts: 3, // At max attempts
		}

		result := renewSubscription(ctx, subscription.ChannelID, subscription, nil)

		assert.False(t, result.Success)
		assert.Equal(t, subscription.ChannelID, result.ChannelID)
		assert.Contains(t, result.Message, "Max renewal attempts (3) exceeded")
		assert.Equal(t, 3, result.AttemptCount)
	})

	t.Run("pubsubhubbub_request_failure", func(t *testing.T) {
		// Temporarily disable test mode to trigger actual PubSubHubbub request
		SetTestMode(false)
		defer SetTestMode(true)
		
		// Clear FUNCTION_URL to trigger PubSubHubbub error
		os.Unsetenv("FUNCTION_URL")
		defer os.Setenv("FUNCTION_URL", "https://test-function-url")

		ctx := context.Background()
		subscription := &Subscription{
			ChannelID:       "UCXuqSBlHAE6Xw-yeJA0Tunw",
			RenewalAttempts: 1,
		}

		result := renewSubscription(ctx, subscription.ChannelID, subscription, nil)

		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "PubSubHubbub renewal failed")
		assert.Equal(t, 2, result.AttemptCount) // Should increment
	})
}

func TestConfigurationHelpers(t *testing.T) {
	t.Run("getRenewalThreshold", func(t *testing.T) {
		// Test default value
		os.Unsetenv("RENEWAL_THRESHOLD_HOURS")
		threshold := getRenewalThreshold()
		assert.Equal(t, 12*time.Hour, threshold)

		// Test custom value
		os.Setenv("RENEWAL_THRESHOLD_HOURS", "6")
		threshold = getRenewalThreshold()
		assert.Equal(t, 6*time.Hour, threshold)

		// Test invalid value falls back to default
		os.Setenv("RENEWAL_THRESHOLD_HOURS", "invalid")
		threshold = getRenewalThreshold()
		assert.Equal(t, 12*time.Hour, threshold)

		os.Unsetenv("RENEWAL_THRESHOLD_HOURS")
	})

	t.Run("getMaxRenewalAttempts", func(t *testing.T) {
		// Test default value
		os.Unsetenv("MAX_RENEWAL_ATTEMPTS")
		attempts := getMaxRenewalAttempts()
		assert.Equal(t, 3, attempts)

		// Test custom value
		os.Setenv("MAX_RENEWAL_ATTEMPTS", "5")
		attempts = getMaxRenewalAttempts()
		assert.Equal(t, 5, attempts)

		// Test invalid value falls back to default
		os.Setenv("MAX_RENEWAL_ATTEMPTS", "invalid")
		attempts = getMaxRenewalAttempts()
		assert.Equal(t, 3, attempts)

		// Test zero value falls back to default
		os.Setenv("MAX_RENEWAL_ATTEMPTS", "0")
		attempts = getMaxRenewalAttempts()
		assert.Equal(t, 3, attempts)

		os.Unsetenv("MAX_RENEWAL_ATTEMPTS")
	})

	t.Run("getLeaseSeconds", func(t *testing.T) {
		// Test default value
		os.Unsetenv("SUBSCRIPTION_LEASE_SECONDS")
		seconds := getLeaseSeconds()
		assert.Equal(t, 86400, seconds)

		// Test custom value
		os.Setenv("SUBSCRIPTION_LEASE_SECONDS", "43200") // 12 hours
		seconds = getLeaseSeconds()
		assert.Equal(t, 43200, seconds)

		// Test invalid value falls back to default
		os.Setenv("SUBSCRIPTION_LEASE_SECONDS", "invalid")
		seconds = getLeaseSeconds()
		assert.Equal(t, 86400, seconds)

		os.Unsetenv("SUBSCRIPTION_LEASE_SECONDS")
	})
}

func TestRenewalEndToEnd(t *testing.T) {
	// Setup test environment
	originalTestMode := GetTestMode()
	SetTestMode(true)
	defer SetTestMode(originalTestMode)

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

	t.Run("mixed_subscription_renewal_scenario", func(t *testing.T) {
		// Create a complex scenario with multiple subscription states
		now := time.Now()
		
		subscriptions := map[string]*Subscription{
			"UC1": {
				ChannelID:       "UC1",
				ExpiresAt:       now.Add(6 * time.Hour),  // Needs renewal
				RenewalAttempts: 0,
			},
			"UC2": {
				ChannelID:       "UC2", 
				ExpiresAt:       now.Add(24 * time.Hour), // Doesn't need renewal
				RenewalAttempts: 0,
			},
			"UC3": {
				ChannelID:       "UC3",
				ExpiresAt:       now.Add(2 * time.Hour),  // Needs renewal but at max attempts
				RenewalAttempts: 3,
			},
			"UC4": {
				ChannelID:       "UC4",
				ExpiresAt:       now.Add(8 * time.Hour),  // Needs renewal
				RenewalAttempts: 1,
			},
		}

		testState := &SubscriptionState{
			Subscriptions: subscriptions,
			Metadata: struct {
				LastUpdated time.Time `json:"last_updated"`
				Version     string    `json:"version"`
			}{
				LastUpdated: now,
				Version:     "1.0",
			},
		}

		SetTestSubscriptionState(testState)

		// Execute renewal
		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()
		handleRenewSubscriptions(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)

		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		
		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 4, response.TotalChecked)
		assert.Equal(t, 3, response.RenewalsCandidates) // UC1, UC3, UC4
		assert.Equal(t, 2, response.RenewalsSucceeded)  // UC1, UC4 (UC3 fails due to max attempts)
		assert.Equal(t, 1, response.RenewalsFailed)     // UC3
		assert.Len(t, response.Results, 3)

		// Verify individual results
		resultsByChannel := make(map[string]RenewalResult)
		for _, result := range response.Results {
			resultsByChannel[result.ChannelID] = result
		}

		// UC1 should succeed
		assert.True(t, resultsByChannel["UC1"].Success)
		assert.Equal(t, "Subscription renewed successfully", resultsByChannel["UC1"].Message)
		
		// UC3 should fail due to max attempts
		assert.False(t, resultsByChannel["UC3"].Success)
		assert.Contains(t, resultsByChannel["UC3"].Message, "Max renewal attempts")
		
		// UC4 should succeed
		assert.True(t, resultsByChannel["UC4"].Success)
		assert.Equal(t, "Subscription renewed successfully", resultsByChannel["UC4"].Message)
	})
}

func TestYouTubeWebhookRenewalRouting(t *testing.T) {
	// Test that the /renew endpoint is properly routed
	originalTestMode := GetTestMode()
	SetTestMode(true)
	defer SetTestMode(originalTestMode)

	// Setup minimal environment
	os.Setenv("FUNCTION_URL", "https://test-function-url")
	defer os.Unsetenv("FUNCTION_URL")

	// Setup empty test state
	SetTestSubscriptionState(&SubscriptionState{
		Subscriptions: make(map[string]*Subscription),
	})

	t.Run("renew_endpoint_routing", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/renew", nil)
		w := httptest.NewRecorder()

		YouTubeWebhook(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response RenewalSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response.Status)
		assert.Equal(t, 0, response.TotalChecked) // Empty state
	})

	t.Run("renew_wrong_method", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/renew", nil)
		w := httptest.NewRecorder()

		YouTubeWebhook(w, req)

		// Should fall through to verification challenge handler
		assert.Equal(t, http.StatusBadRequest, w.Code) // No challenge parameter
	})
}
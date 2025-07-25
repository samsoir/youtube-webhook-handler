package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)


// TestGetSubscriptions_Success tests listing all current subscriptions
func TestGetSubscriptions_Success(t *testing.T) {
	// Setup
	setupSubscriptionTest()
	defer teardownSubscriptionTest()
	
	// Test case: GET /subscriptions
	// Expected behavior:
	// 1. Load subscription state from storage
	// 2. Return list of all subscriptions with their status
	// 3. Include summary statistics (total, active, expired)

	// Pre-populate subscription state with test data
	now := time.Now()
	testSubscriptionState = &SubscriptionState{
		Subscriptions: map[string]*Subscription{
			"UCXuqSBlHAE6Xw-yeJA0Tunw": {
				ChannelID:    "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Status:       "active",
				ExpiresAt:    now.Add(12 * time.Hour), // Active, expires in 12 hours
				SubscribedAt: now.Add(-12 * time.Hour),
			},
			"UCBJycsmduvYEL83R_U4JriQ": {
				ChannelID:    "UCBJycsmduvYEL83R_U4JriQ", 
				Status:       "active",
				ExpiresAt:    now.Add(36 * time.Hour), // Active, expires in 36 hours
				SubscribedAt: now.Add(-12 * time.Hour),
			},
			"UC1234567890123456789012": {
				ChannelID:    "UC1234567890123456789012",
				Status:       "expired",
				ExpiresAt:    now.Add(-24 * time.Hour), // Expired 24 hours ago
				SubscribedAt: now.Add(-48 * time.Hour),
			},
		},
	}
	
	// Create request
	req := httptest.NewRequest("GET", "/subscriptions", nil)
	w := httptest.NewRecorder()
	
	// Execute
	handleGetSubscriptions(w, req)
	
	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK for successful listing")
	
	// Verify response body structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")
	
	// Verify top-level structure
	assert.Contains(t, response, "subscriptions", "Should include subscriptions array")
	assert.Contains(t, response, "total", "Should include total count")
	assert.Contains(t, response, "active", "Should include active count") 
	assert.Contains(t, response, "expired", "Should include expired count")
	
	// Verify summary statistics
	assert.Equal(t, float64(3), response["total"], "Should have 3 total subscriptions")
	assert.Equal(t, float64(2), response["active"], "Should have 2 active subscriptions")
	assert.Equal(t, float64(1), response["expired"], "Should have 1 expired subscription")
	
	// Verify subscriptions array structure
	subscriptions, ok := response["subscriptions"].([]interface{})
	require.True(t, ok, "Subscriptions should be an array")
	assert.Len(t, subscriptions, 3, "Should return 3 subscriptions")
	
	// Verify first subscription structure
	sub1, ok := subscriptions[0].(map[string]interface{})
	require.True(t, ok, "Each subscription should be an object")
	
	// Verify required fields for each subscription
	assert.Contains(t, sub1, "channel_id", "Should include channel_id")
	assert.Contains(t, sub1, "status", "Should include status")
	assert.Contains(t, sub1, "expires_at", "Should include expires_at")
	assert.Contains(t, sub1, "days_until_expiry", "Should include days_until_expiry")
	
	// Verify channel ID format
	channelID, ok := sub1["channel_id"].(string)
	require.True(t, ok, "Channel ID should be a string")
	assert.Regexp(t, `^UC[a-zA-Z0-9_-]{22}$`, channelID, "Channel ID should be valid format")
	
	// Verify status values
	status, ok := sub1["status"].(string)
	require.True(t, ok, "Status should be a string")
	assert.Contains(t, []string{"active", "expired"}, status, "Status should be 'active' or 'expired'")
	
	// Verify expires_at format
	expiresAt, ok := sub1["expires_at"].(string)
	require.True(t, ok, "expires_at should be a string")
	_, err = time.Parse(time.RFC3339, expiresAt)
	require.NoError(t, err, "expires_at should be valid RFC3339 timestamp")
	
	// Verify days_until_expiry is a number
	_, ok = sub1["days_until_expiry"].(float64)
	require.True(t, ok, "days_until_expiry should be a number")
}

// TestGetSubscriptions_Empty tests listing when no subscriptions exist
func TestGetSubscriptions_Empty(t *testing.T) {
	// Test case: GET /subscriptions (no subscriptions exist)
	// Expected behavior:
	// 1. Return empty array for subscriptions
	// 2. All counts should be zero
	// 3. Still return 200 OK (empty is valid state)

	// Setup
	setupSubscriptionTest()
	defer teardownSubscriptionTest()
	
	// Create request
	req := httptest.NewRequest("GET", "/subscriptions", nil)
	w := httptest.NewRecorder()
	
	// Execute
	handleGetSubscriptions(w, req)
	
	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK even with no subscriptions")
	
	// Verify response body structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")
	
	// Verify structure with zero values
	assert.Contains(t, response, "subscriptions", "Should include subscriptions array")
	assert.Contains(t, response, "total", "Should include total count")
	assert.Contains(t, response, "active", "Should include active count")
	assert.Contains(t, response, "expired", "Should include expired count")
	
	// Verify all counts are zero
	assert.Equal(t, float64(0), response["total"], "Should have 0 total subscriptions")
	assert.Equal(t, float64(0), response["active"], "Should have 0 active subscriptions")
	assert.Equal(t, float64(0), response["expired"], "Should have 0 expired subscriptions")
	
	// Verify empty array
	subscriptions, ok := response["subscriptions"].([]interface{})
	require.True(t, ok, "Subscriptions should be an array")
	assert.Len(t, subscriptions, 0, "Should return empty array")
}

// TestGetSubscriptions_StorageError tests handling of storage read failures
func TestGetSubscriptions_StorageError(t *testing.T) {
	// Storage error handling is comprehensively tested in TestGetSubscriptionsWithCloudStorageErrors
	// which provides proper mocking and covers multiple error scenarios
	t.Skip("Storage error handling is comprehensively tested in TestGetSubscriptionsWithCloudStorageErrors")
}

// TestGetSubscriptions_RealModeWithoutEnvVars tests subscriptions list endpoint in non-test mode
func TestGetSubscriptions_RealModeWithoutEnvVars(t *testing.T) {
	// Setup non-test mode
	setupNonTestMode()
	defer teardownNonTestMode()
	
	// Clear environment variables
	originalBucket := os.Getenv("SUBSCRIPTION_BUCKET")
	os.Unsetenv("SUBSCRIPTION_BUCKET")
	defer func() {
		if originalBucket != "" {
			os.Setenv("SUBSCRIPTION_BUCKET", originalBucket)
		}
	}()
	
	// Create request
	req := httptest.NewRequest("GET", "/subscriptions", nil)
	w := httptest.NewRecorder()
	
	// Execute
	handleGetSubscriptions(w, req)
	
	// Should return 500 error due to missing SUBSCRIPTION_BUCKET
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "error", response["status"])
	assert.Contains(t, response["message"], "Unable to load subscription state")
}

// TestGetSubscriptionsWithCloudStorageErrors tests comprehensive error scenarios for GetSubscriptions
func TestGetSubscriptionsWithCloudStorageErrors(t *testing.T) {
	
	t.Run("LoadSubscriptionState_Error", func(t *testing.T) {
		// Setup mock storage
		mockClient, origClient := setupMockStorage()
		defer teardownMockStorage(origClient)
		
		// Mock LoadSubscriptionState to return error
		mockClient.On("LoadSubscriptionState", mock.Anything).Return(
			(*SubscriptionState)(nil), 
			fmt.Errorf("authentication failed"),
		)
		
		req := httptest.NewRequest("GET", "/subscriptions", nil)
		w := httptest.NewRecorder()
		
		handleGetSubscriptions(w, req)
		
		// Verify error response
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "error", response.Status)
		assert.Contains(t, response.Message, "Unable to load subscription state from storage")
		
		// Verify mock was called
		mockClient.AssertExpectations(t)
	})
	
	t.Run("SuccessfulLoad_WithMultipleSubscriptions", func(t *testing.T) {
		// Setup mock storage
		mockClient, origClient := setupMockStorage()
		defer teardownMockStorage(origClient)
		
		now := time.Now()
		
		// Mock state with multiple subscriptions (active and expired)
		stateWithSubscriptions := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				"UCXuqSBlHAE6Xw-yeJA0Tunw": {
					ChannelID:    "UCXuqSBlHAE6Xw-yeJA0Tunw",
					Status:       "active",
					ExpiresAt:    now.Add(12 * time.Hour), // Active
					SubscribedAt: now.Add(-time.Hour),
				},
				"UCYuqSBlHAE6Xw-yeJA0Tunn": {
					ChannelID:    "UCYuqSBlHAE6Xw-yeJA0Tunn",
					Status:       "active",
					ExpiresAt:    now.Add(-time.Hour), // Expired
					SubscribedAt: now.Add(-25 * time.Hour),
				},
			},
			Metadata: struct {
				LastUpdated time.Time `json:"last_updated"`
				Version     string    `json:"version"`
			}{
				LastUpdated: now,
				Version:     "1.0",
			},
		}
		
		mockClient.On("LoadSubscriptionState", mock.Anything).Return(stateWithSubscriptions, nil)
		
		req := httptest.NewRequest("GET", "/subscriptions", nil)
		w := httptest.NewRecorder()
		
		handleGetSubscriptions(w, req)
		
		// Verify success response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response SubscriptionsListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, 2, response.Total)
		assert.Equal(t, 1, response.Active)
		assert.Equal(t, 1, response.Expired)
		assert.Len(t, response.Subscriptions, 2)
		
		// Verify active subscription
		activeFound := false
		expiredFound := false
		for _, sub := range response.Subscriptions {
			if sub.ChannelID == "UCXuqSBlHAE6Xw-yeJA0Tunw" {
				assert.Equal(t, "active", sub.Status)
				assert.True(t, sub.DaysUntilExpiry > 0)
				activeFound = true
			}
			if sub.ChannelID == "UCYuqSBlHAE6Xw-yeJA0Tunn" {
				assert.Equal(t, "expired", sub.Status)
				assert.True(t, sub.DaysUntilExpiry < 0)
				expiredFound = true
			}
		}
		assert.True(t, activeFound, "Active subscription should be found")
		assert.True(t, expiredFound, "Expired subscription should be found")
		
		// Verify mock was called
		mockClient.AssertExpectations(t)
	})
}
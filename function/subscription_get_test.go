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

// TestGetSubscriptions_Success tests listing all current subscriptions
func TestGetSubscriptions_Success(t *testing.T) {
	// Setup with dependency injection
	deps := CreateTestDependencies()

	// Test case: GET /subscriptions
	// Expected behavior:
	// 1. Load subscription state from storage
	// 2. Return list of all subscriptions with their status
	// 3. Include summary statistics (total, active, expired)

	// Pre-populate subscription state with test data
	now := time.Now()
	testState := &SubscriptionState{
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
	deps.StorageClient.(*MockStorageClient).SetState(testState)

	// Create request
	req := httptest.NewRequest("GET", "/subscriptions", nil)
	w := httptest.NewRecorder()

	// Execute using dependency injection
	handler := handleGetSubscriptions(deps)
	handler(w, req)

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

	// Verify status values
	statusValues := make(map[string]int)
	for _, sub := range subscriptions {
		subMap := sub.(map[string]interface{})
		status := subMap["status"].(string)
		statusValues[status]++
	}
	assert.Equal(t, 2, statusValues["active"], "Should have 2 active subscriptions")
	assert.Equal(t, 1, statusValues["expired"], "Should have 1 expired subscription")

	// Verify channel IDs are present
	channelIDs := make([]string, 0)
	for _, sub := range subscriptions {
		subMap := sub.(map[string]interface{})
		channelID := subMap["channel_id"].(string)
		channelIDs = append(channelIDs, channelID)
	}
	assert.Contains(t, channelIDs, "UCXuqSBlHAE6Xw-yeJA0Tunw")
	assert.Contains(t, channelIDs, "UCBJycsmduvYEL83R_U4JriQ")
	assert.Contains(t, channelIDs, "UC1234567890123456789012")

	// Verify expires_at is valid RFC3339
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

	// Setup with dependency injection
	deps := CreateTestDependencies()

	// Create request
	req := httptest.NewRequest("GET", "/subscriptions", nil)
	w := httptest.NewRecorder()

	// Execute using dependency injection
	handler := handleGetSubscriptions(deps)
	handler(w, req)

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

	// Verify zero counts
	assert.Equal(t, float64(0), response["total"], "Should have 0 total subscriptions")
	assert.Equal(t, float64(0), response["active"], "Should have 0 active subscriptions")
	assert.Equal(t, float64(0), response["expired"], "Should have 0 expired subscriptions")

	// Verify empty array
	subscriptions, ok := response["subscriptions"].([]interface{})
	require.True(t, ok, "Subscriptions should be an array")
	assert.Len(t, subscriptions, 0, "Should return empty array")
}
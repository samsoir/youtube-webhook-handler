package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/samsoir/youtube-webhook/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubscribeToChannel_Success tests successful subscription to a YouTube channel
func TestSubscribeToChannel_Success(t *testing.T) {
	// Test case: POST /subscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw
	// Expected behavior:
	// 1. Validate channel ID format
	// 2. Check if not already subscribed
	// 3. Make subscription request to PubSubHubbub hub
	// 4. Store subscription state in Cloud Storage
	// 5. Return success response with expiration time

	channelID := testutil.TestChannelIDs.Valid

	// Setup using dependency injection
	deps := CreateTestDependencies()

	// Create request
	req := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
	w := httptest.NewRecorder()

	// Execute using dependency injection
	handler := handleSubscribeWithDeps(deps)
	handler(w, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK for successful subscription")

	// Verify response body
	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")

	assert.Equal(t, "success", response.Status, "Status should be 'success'")
	assert.Equal(t, channelID, response.ChannelID, "Should return the channel ID")
	assert.Equal(t, "Subscription initiated", response.Message, "Should have correct message")
	assert.NotEmpty(t, response.ExpiresAt, "Should include expiration time")

	// Verify expiration time is in the future
	expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
	require.NoError(t, err, "ExpiresAt should be valid RFC3339")
	assert.True(t, expiresAt.After(time.Now()), "Expiration should be in the future")

	// Verify subscription was stored using dependency injection
	assert.Equal(t, 1, deps.StorageClient.(*MockStorageClient).LoadCallCount, "Should load state once")
	assert.Equal(t, 1, deps.StorageClient.(*MockStorageClient).SaveCallCount, "Should save state once")

	// Verify subscription details in storage
	savedState := deps.StorageClient.(*MockStorageClient).GetState()
	assert.NotNil(t, savedState)
	assert.Contains(t, savedState.Subscriptions, channelID, "Should store subscription")

	sub := savedState.Subscriptions[channelID]
	assert.Equal(t, "active", sub.Status, "Subscription should be active")
	assert.Equal(t, 86400, sub.LeaseSeconds, "Lease should be 24 hours")
	assert.NotZero(t, sub.SubscribedAt, "Should set subscription time")
}

// TestSubscribeToChannel_AlreadySubscribed tests subscribing to an already subscribed channel
func TestSubscribeToChannel_AlreadySubscribed(t *testing.T) {
	// Test case: POST /subscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw (already subscribed)
	// Expected behavior:
	// 1. Check subscription state
	// 2. Find existing subscription
	// 3. Return 409 Conflict with subscription details
	// 4. Do not make duplicate hub request

	channelID := testutil.TestChannelIDs.Valid

	// Setup using dependency injection with existing subscription
	deps := CreateTestDependencies()

	// Pre-populate subscription state
	existingSub := createTestSubscription(channelID)
	testState := createTestSubscriptionState(existingSub)
	deps.StorageClient.(*MockStorageClient).SetState(testState)

	// Create request
	req := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
	w := httptest.NewRecorder()

	// Execute using dependency injection
	handler := handleSubscribeWithDeps(deps)
	handler(w, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusConflict, w.Code, "Should return 409 Conflict for duplicate subscription")

	// Verify response body
	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")

	assert.Equal(t, "conflict", response.Status, "Status should be 'conflict'")
	assert.Equal(t, channelID, response.ChannelID, "Should return the channel ID")
	assert.Equal(t, "Already subscribed to this channel", response.Message)
	assert.NotEmpty(t, response.ExpiresAt, "Should include existing expiration time")

	// Verify no new subscription was created
	assert.Equal(t, 1, deps.StorageClient.(*MockStorageClient).LoadCallCount, "Should only load state once")
	assert.Equal(t, 0, deps.StorageClient.(*MockStorageClient).SaveCallCount, "Should not save state for duplicate")
}

// Helper functions for creating test data
func createTestSubscription(channelID string) *Subscription {
	now := time.Now()
	return &Subscription{
		ChannelID:       channelID,
		TopicURL:        "https://www.youtube.com/feeds/videos.xml?channel_id=" + channelID,
		CallbackURL:     "https://test-function-url",
		Status:          "active",
		LeaseSeconds:    86400,
		SubscribedAt:    now,
		ExpiresAt:       now.Add(24 * time.Hour),
		LastRenewal:     now,
		RenewalAttempts: 0,
		HubResponse:     "202 Accepted",
	}
}

func createTestSubscriptionState(subs ...*Subscription) *SubscriptionState {
	state := &SubscriptionState{
		Subscriptions: make(map[string]*Subscription),
	}
	for _, sub := range subs {
		state.Subscriptions[sub.ChannelID] = sub
	}
	return state
}
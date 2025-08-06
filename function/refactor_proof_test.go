package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Simple proof-of-concept test showing dependency injection works
func TestRefactoredSubscribeProofOfConcept(t *testing.T) {
	// Create dependencies with mocks
	deps := &Dependencies{
		StorageClient: NewRefactoredMockStorageClient(),
		PubSubClient:  NewMockPubSubClient(),
	}

	// Create test request
	channelID := "UCXuqSBlHAE6Xw-yeJA0Tunw"
	req := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
	rec := httptest.NewRecorder()

	// Call the refactored handler with dependency injection
	handler := handleSubscribeWithDeps(deps)
	handler(rec, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusOK, rec.Code)

	// Parse response
	var response APIResponse
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response content
	assert.Equal(t, "success", response.Status)
	assert.Equal(t, channelID, response.ChannelID)
	assert.Equal(t, "Subscription initiated", response.Message)

	// Verify mock interactions - NO GLOBAL testMode USED!
	mockStorage := deps.StorageClient
	mockPubSub := deps.PubSubClient.(*MockPubSubClient)

	assert.Equal(t, 1, mockStorage.LoadCallCount)
	assert.Equal(t, 1, mockStorage.SaveCallCount)
	assert.Equal(t, 1, mockPubSub.GetSubscribeCount())
	assert.Equal(t, channelID, mockPubSub.GetLastChannelID())

	// Verify subscription was created correctly
	savedState := mockStorage.GetState()
	subscription := savedState.Subscriptions[channelID]
	assert.NotNil(t, subscription)
	assert.Equal(t, "active", subscription.Status)
	assert.Equal(t, 86400, subscription.LeaseSeconds)
}
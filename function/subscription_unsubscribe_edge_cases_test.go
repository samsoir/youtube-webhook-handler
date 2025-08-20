package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samsoir/youtube-webhook/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnsubscribeWithDeps_EdgeCases tests various edge cases for the unsubscribe handler using dependency injection
func TestUnsubscribeWithDeps_EdgeCases(t *testing.T) {
	t.Run("InvalidChannelID", func(t *testing.T) {
		deps := CreateTestDependencies()

		testCases := []struct {
			name      string
			channelID string
			expected  string
		}{
			{
				name:      "Empty channel ID",
				channelID: "",
				expected:  "channel_id parameter is required",
			},
			{
				name:      "Invalid format - too short",
				channelID: "UC123",
				expected:  "Invalid channel ID format",
			},
			{
				name:      "Invalid format - wrong prefix",
				channelID: "AB1234567890123456789012",
				expected:  "Invalid channel ID format",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+tc.channelID, nil)
				w := httptest.NewRecorder()

				handler := handleUnsubscribeWithDeps(deps)
				handler(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)

				var response APIResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, "error", response.Status)
				assert.Contains(t, response.Message, tc.expected)
			})
		}
	})

	t.Run("SubscriptionNotFound", func(t *testing.T) {
		deps := CreateTestDependencies()
		
		// Use a valid channel ID that doesn't exist in subscriptions
		channelID := testutil.TestChannelIDs.Valid

		req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
		w := httptest.NewRecorder()

		handler := handleUnsubscribeWithDeps(deps)
		handler(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "error", response.Status)
		assert.Contains(t, response.Message, "Subscription not found for this channel")
	})

	t.Run("StorageErrors", func(t *testing.T) {
		t.Run("LoadError", func(t *testing.T) {
			deps := CreateTestDependencies()
			deps.StorageClient.(*MockStorageClient).LoadError = fmt.Errorf("storage unavailable")

			req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+testutil.TestChannelIDs.Valid, nil)
			w := httptest.NewRecorder()

			handler := handleUnsubscribeWithDeps(deps)
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
			channelID := testutil.TestChannelIDs.Valid

			// Pre-populate with a subscription
			existingSub := createTestSubscription(channelID)
			testState := createTestSubscriptionState(existingSub)
			deps.StorageClient.(*MockStorageClient).SetState(testState)

			// Make save operation fail
			deps.StorageClient.(*MockStorageClient).SaveError = fmt.Errorf("storage write failed")

			req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
			w := httptest.NewRecorder()

			handler := handleUnsubscribeWithDeps(deps)
			handler(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code)

			var response APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "error", response.Status)
			assert.Contains(t, response.Message, "Failed to save subscription state")
		})
	})

	t.Run("PubSubErrors", func(t *testing.T) {
		deps := CreateTestDependencies()
		channelID := testutil.TestChannelIDs.Valid

		// Pre-populate with a subscription
		existingSub := createTestSubscription(channelID)
		testState := createTestSubscriptionState(existingSub)
		deps.StorageClient.(*MockStorageClient).SetState(testState)

		// Set PubSub to return an error
		mockPubSub := deps.PubSubClient.(*MockPubSubClient)
		mockPubSub.SetUnsubscribeError(fmt.Errorf("PubSubHubbub server unavailable"))

		req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
		w := httptest.NewRecorder()

		handler := handleUnsubscribeWithDeps(deps)
		handler(w, req)

		assert.Equal(t, http.StatusBadGateway, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "error", response.Status)
		assert.Contains(t, response.Message, "PubSubHubbub unsubscribe failed")
	})

	t.Run("SuccessfulUnsubscribe", func(t *testing.T) {
		deps := CreateTestDependencies()
		channelID := testutil.TestChannelIDs.Valid

		// Pre-populate with a subscription
		existingSub := createTestSubscription(channelID)
		testState := createTestSubscriptionState(existingSub)
		deps.StorageClient.(*MockStorageClient).SetState(testState)

		req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
		w := httptest.NewRecorder()

		handler := handleUnsubscribeWithDeps(deps)
		handler(w, req)

		// Should return 204 No Content
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())

		// Verify subscription was removed from storage
		finalState := deps.StorageClient.(*MockStorageClient).GetState()
		assert.NotContains(t, finalState.Subscriptions, channelID)

		// Verify PubSub unsubscribe was called
		mockPubSub := deps.PubSubClient.(*MockPubSubClient)
		assert.False(t, mockPubSub.IsSubscribed(channelID))
	})

	t.Run("MultipleSubscriptionsOneRemoved", func(t *testing.T) {
		deps := CreateTestDependencies()
		channelID1 := testutil.TestChannelIDs.Valid
		channelID2 := "UCBJycsmduvYEL83R_U4JriQ" // Different valid channel

		// Pre-populate with two subscriptions
		sub1 := createTestSubscription(channelID1)
		sub2 := createTestSubscription(channelID2)
		testState := createTestSubscriptionState(sub1, sub2)
		deps.StorageClient.(*MockStorageClient).SetState(testState)

		req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID1, nil)
		w := httptest.NewRecorder()

		handler := handleUnsubscribeWithDeps(deps)
		handler(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify only the target subscription was removed
		finalState := deps.StorageClient.(*MockStorageClient).GetState()
		assert.NotContains(t, finalState.Subscriptions, channelID1)
		assert.Contains(t, finalState.Subscriptions, channelID2)
		assert.Len(t, finalState.Subscriptions, 1)
	})

	t.Run("ConcurrentUnsubscribe", func(t *testing.T) {
		// Test concurrent unsubscribe requests to the same channel
		deps := CreateTestDependencies()
		channelID := testutil.TestChannelIDs.Valid

		// Pre-populate with a subscription
		existingSub := createTestSubscription(channelID)
		testState := createTestSubscriptionState(existingSub)
		deps.StorageClient.(*MockStorageClient).SetState(testState)

		done := make(chan bool, 2)
		results := make([]int, 2)

		// Run two concurrent unsubscribe requests
		for i := 0; i < 2; i++ {
			go func(index int) {
				req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
				w := httptest.NewRecorder()

				handler := handleUnsubscribeWithDeps(deps)
				handler(w, req)

				results[index] = w.Code
				done <- true
			}(i)
		}

		// Wait for both to complete
		for i := 0; i < 2; i++ {
			<-done
		}

		// One should succeed (204), one should fail with not found (404)
		statusCodes := make(map[int]int)
		for _, code := range results {
			statusCodes[code]++
		}

		// Should have one success and one not found, or two successes (if very fast)
		assert.True(t, 
			(statusCodes[http.StatusNoContent] == 1 && statusCodes[http.StatusNotFound] == 1) ||
			statusCodes[http.StatusNoContent] == 2,
			"Expected one success + one not found, or two successes, got: %v", statusCodes)
	})
}

// TestUnsubscribeWithDeps_ErrorRecovery tests error recovery scenarios
func TestUnsubscribeWithDeps_ErrorRecovery(t *testing.T) {
	t.Run("RecoverFromTransientPubSubError", func(t *testing.T) {
		deps := CreateTestDependencies()
		channelID := testutil.TestChannelIDs.Valid

		// Pre-populate with a subscription
		existingSub := createTestSubscription(channelID)
		testState := createTestSubscriptionState(existingSub)
		deps.StorageClient.(*MockStorageClient).SetState(testState)

		mockPubSub := deps.PubSubClient.(*MockPubSubClient)

		// First request fails due to PubSub error
		mockPubSub.SetUnsubscribeError(fmt.Errorf("temporary PubSub error"))

		req1 := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
		w1 := httptest.NewRecorder()

		handler := handleUnsubscribeWithDeps(deps)
		handler(w1, req1)

		assert.Equal(t, http.StatusBadGateway, w1.Code)

		// Subscription should still exist since PubSub failed
		state := deps.StorageClient.(*MockStorageClient).GetState()
		assert.Contains(t, state.Subscriptions, channelID)

		// Clear the error and try again
		mockPubSub.SetUnsubscribeError(nil)

		req2 := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
		w2 := httptest.NewRecorder()

		handler(w2, req2)

		assert.Equal(t, http.StatusNoContent, w2.Code)

		// Subscription should now be removed
		finalState := deps.StorageClient.(*MockStorageClient).GetState()
		assert.NotContains(t, finalState.Subscriptions, channelID)
	})
}
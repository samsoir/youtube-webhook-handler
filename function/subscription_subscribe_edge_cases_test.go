package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/samsoir/youtube-webhook/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubscribe_EdgeCases tests various edge cases for the subscribe handler using dependency injection
func TestSubscribe_EdgeCases(t *testing.T) {
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
				expected:  "Invalid channel ID format. Must be UC followed by 22 alphanumeric characters",
			},
			{
				name:      "Invalid format - wrong prefix",
				channelID: "AB1234567890123456789012",
				expected:  "Invalid channel ID format. Must be UC followed by 22 alphanumeric characters",
			},
			{
				name:      "Invalid format - too long",
				channelID: "UC12345678901234567890123",
				expected:  "Invalid channel ID format. Must be UC followed by 22 alphanumeric characters",
			},
			{
				name:      "Invalid characters",
				channelID: "UC@#$%^&*()1234567890AB",
				expected:  "Invalid channel ID format. Must be UC followed by 22 alphanumeric characters",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("POST", "/subscribe?channel_id="+url.QueryEscape(tc.channelID), nil)
				w := httptest.NewRecorder()

				handler := handleSubscribe(deps)
				handler(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 for invalid channel ID")

				var response APIResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, "error", response.Status)
				assert.Contains(t, response.Message, tc.expected)
			})
		}
	})

	t.Run("StorageErrors", func(t *testing.T) {
		t.Run("LoadError", func(t *testing.T) {
			deps := CreateTestDependencies()
			deps.StorageClient.(*MockStorageClient).LoadError = fmt.Errorf("storage unavailable")

			req := httptest.NewRequest("POST", "/subscribe?channel_id="+testutil.TestChannelIDs.Valid, nil)
			w := httptest.NewRecorder()

			handler := handleSubscribe(deps)
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
			deps.StorageClient.(*MockStorageClient).SaveError = fmt.Errorf("storage write failed")

			req := httptest.NewRequest("POST", "/subscribe?channel_id="+testutil.TestChannelIDs.Valid, nil)
			w := httptest.NewRecorder()

			handler := handleSubscribe(deps)
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
		
		// Set PubSub to return an error
		mockPubSub := deps.PubSubClient.(*MockPubSubClient)
		mockPubSub.SetSubscribeError(fmt.Errorf("PubSubHubbub server unavailable"))

		req := httptest.NewRequest("POST", "/subscribe?channel_id="+testutil.TestChannelIDs.Valid, nil)
		w := httptest.NewRecorder()

		handler := handleSubscribe(deps)
		handler(w, req)

		assert.Equal(t, http.StatusBadGateway, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "error", response.Status)
		assert.Contains(t, response.Message, "PubSubHubbub subscription failed")
		assert.Contains(t, response.Message, "server unavailable")
	})

	t.Run("MissingChannelIDParameter", func(t *testing.T) {
		deps := CreateTestDependencies()

		// Request without channel_id parameter
		req := httptest.NewRequest("POST", "/subscribe", nil)
		w := httptest.NewRecorder()

		handler := handleSubscribe(deps)
		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "error", response.Status)
		assert.Equal(t, "channel_id parameter is required", response.Message)

		// Verify no storage operations were attempted
		assert.Equal(t, 0, deps.StorageClient.(*MockStorageClient).LoadCallCount)
		assert.Equal(t, 0, deps.StorageClient.(*MockStorageClient).SaveCallCount)
	})

	t.Run("ConcurrentSubscriptions", func(t *testing.T) {
		// Test concurrent subscribe requests to the same channel
		deps := CreateTestDependencies()
		channelID := testutil.TestChannelIDs.Valid

		done := make(chan bool, 2)
		results := make([]int, 2)

		// Run two concurrent subscribe requests
		for i := 0; i < 2; i++ {
			go func(index int) {
				req := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
				w := httptest.NewRecorder()

				handler := handleSubscribe(deps)
				handler(w, req)

				results[index] = w.Code
				done <- true
			}(i)
		}

		// Wait for both to complete
		for i := 0; i < 2; i++ {
			<-done
		}

		// One should succeed (200), one might be conflict (409) due to race condition
		// But both should be valid responses
		for i, code := range results {
			assert.True(t, code == http.StatusOK || code == http.StatusConflict, 
				"Request %d should return either 200 or 409, got %d", i, code)
		}
	})
}

// TestSubscribe_ErrorRecovery tests error recovery scenarios
func TestSubscribe_ErrorRecovery(t *testing.T) {
	t.Run("RecoverFromTransientStorageError", func(t *testing.T) {
		deps := CreateTestDependencies()
		channelID := testutil.TestChannelIDs.Valid

		// First request fails due to storage error
		deps.StorageClient.(*MockStorageClient).LoadError = fmt.Errorf("temporary storage error")

		req1 := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
		w1 := httptest.NewRecorder()

		handler := handleSubscribe(deps)
		handler(w1, req1)

		assert.Equal(t, http.StatusInternalServerError, w1.Code)

		// Clear the error and try again
		deps.StorageClient.(*MockStorageClient).LoadError = nil

		req2 := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
		w2 := httptest.NewRecorder()

		handler(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)

		var response APIResponse
		err := json.Unmarshal(w2.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "success", response.Status)
	})

	t.Run("RecoverFromTransientPubSubError", func(t *testing.T) {
		deps := CreateTestDependencies()
		channelID := testutil.TestChannelIDs.Valid
		mockPubSub := deps.PubSubClient.(*MockPubSubClient)

		// First request fails due to PubSub error
		mockPubSub.SetSubscribeError(fmt.Errorf("temporary PubSub error"))

		req1 := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
		w1 := httptest.NewRecorder()

		handler := handleSubscribe(deps)
		handler(w1, req1)

		assert.Equal(t, http.StatusBadGateway, w1.Code)

		// Clear the error and try again
		mockPubSub.SetSubscribeError(nil)

		req2 := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
		w2 := httptest.NewRecorder()

		handler(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)

		var response APIResponse
		err := json.Unmarshal(w2.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "success", response.Status)
	})
}

// TestMockPubSubClient_UncoveredMethods tests methods that were not covered in other tests
func TestMockPubSubClient_UncoveredMethods(t *testing.T) {
	mock := NewMockPubSubClient()
	
	// Test initial state
	assert.Equal(t, 0, mock.GetSubscribeCount())
	assert.Equal(t, 0, mock.GetUnsubscribeCount())
	assert.Equal(t, "", mock.GetLastChannelID())
	assert.Equal(t, "", mock.GetLastMode())
	
	// Test Subscribe tracking
	err := mock.Subscribe("UCTestChannel1")
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.GetSubscribeCount())
	assert.Equal(t, "UCTestChannel1", mock.GetLastChannelID())
	assert.Equal(t, "subscribe", mock.GetLastMode())
	
	// Test another Subscribe
	err = mock.Subscribe("UCTestChannel2")
	assert.NoError(t, err)
	assert.Equal(t, 2, mock.GetSubscribeCount())
	assert.Equal(t, "UCTestChannel2", mock.GetLastChannelID())
	assert.Equal(t, "subscribe", mock.GetLastMode())
	
	// Test Unsubscribe tracking
	err = mock.Unsubscribe("UCTestChannel1")
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.GetUnsubscribeCount())
	assert.Equal(t, "UCTestChannel1", mock.GetLastChannelID())
	assert.Equal(t, "unsubscribe", mock.GetLastMode())
	
	// Test Unsubscribe with error
	mock.SetUnsubscribeError(fmt.Errorf("unsubscribe failed"))
	err = mock.Unsubscribe("UCTestChannel3")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsubscribe failed")
	assert.Equal(t, 2, mock.GetUnsubscribeCount()) // Should still increment even on error
	assert.Equal(t, "UCTestChannel3", mock.GetLastChannelID())
	assert.Equal(t, "unsubscribe", mock.GetLastMode())
}
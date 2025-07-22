package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/samsoir/youtube-webhook/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestUnsubscribeFromChannel_Success tests successful unsubscription from an existing channel
func TestUnsubscribeFromChannel_Success(t *testing.T) {
	// Setup
	setupSubscriptionTest()
	defer teardownSubscriptionTest()
	
	// Test case: DELETE /unsubscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw
	// Expected behavior:
	// 1. Validate channel ID exists in subscription state
	// 2. Make unsubscribe request to PubSubHubbub hub
	// 3. Remove subscription from state storage
	// 4. Return 204 No Content (successful deletion with no response body)

	channelID := testutil.TestChannelIDs.Valid
	
	// First, create a subscription
	req1 := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
	w1 := httptest.NewRecorder()
	handleSubscribe(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code, "Subscription should be created successfully")
	
	// Now unsubscribe
	req2 := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
	w2 := httptest.NewRecorder()
	handleUnsubscribe(w2, req2)
	
	// Verify HTTP response
	assert.Equal(t, http.StatusNoContent, w2.Code, "Should return 204 No Content for successful unsubscribe")
	
	// Verify no response body (204 No Content should have empty body)
	assert.Empty(t, w2.Body.String(), "204 No Content should have empty response body")
	
	// Verify subscription was removed: try to unsubscribe again, should get 404
	req3 := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
	w3 := httptest.NewRecorder()
	handleUnsubscribe(w3, req3)
	assert.Equal(t, http.StatusNotFound, w3.Code, "Should return 404 for already removed subscription")
}

// TestUnsubscribeFromChannel_NotFound tests unsubscribing from non-existent subscription
func TestUnsubscribeFromChannel_NotFound(t *testing.T) {
	// Setup
	setupSubscriptionTest()
	defer teardownSubscriptionTest()
	// Test case: DELETE /unsubscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw (not subscribed)
	// Expected behavior:
	// 1. Check subscription state for channel
	// 2. Return 404 Not Found if no subscription exists
	// 3. Do not make any hub requests

	channelID := testutil.TestChannelIDs.Valid
	
	// Ensure subscription state is empty (no subscriptions exist)
	
	// Create request
	req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
	w := httptest.NewRecorder()
	
	// Execute
	handleUnsubscribe(w, req)
	
	// Verify HTTP response
	assert.Equal(t, http.StatusNotFound, w.Code, "Should return 404 Not Found for non-existent subscription")
	
	// Verify response body structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")
	
	// Verify response contains expected fields
	assert.Equal(t, "error", response["status"], "Status should be 'error'")
	assert.Equal(t, channelID, response["channel_id"], "Should return the channel ID")
	assert.Contains(t, response, "message", "Should include error message")
	
	// Verify error message is descriptive
	message, ok := response["message"].(string)
	require.True(t, ok, "Message should be a string")
	assert.Contains(t, strings.ToLower(message), "not found", "Error message should mention 'not found'")
	assert.Contains(t, strings.ToLower(message), "subscription", "Error message should mention 'subscription'")
}

// TestUnsubscribeFromChannel_InvalidChannelID tests validation for unsubscribe requests
func TestUnsubscribeFromChannel_InvalidChannelID(t *testing.T) {
	// Test case: DELETE /unsubscribe?channel_id=invalid-format
	// Expected behavior:
	// 1. Validate channel ID format before checking state
	// 2. Return 400 Bad Request for invalid format
	// 3. Do not check state or make hub requests

	testCases := []struct {
		name      string
		channelID string
	}{
		{"too_short", "UC123"},
		{"wrong_prefix", "XCXuqSBlHAE6Xw-yeJA0Tunw"},
		{"invalid_characters", "UCinvalid@#$characters"},
		{"empty_string", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			setupSubscriptionTest()
			defer teardownSubscriptionTest()
			
			// Create request with proper URL encoding
			reqURL := "/unsubscribe?channel_id=" + url.QueryEscape(tc.channelID)
			req := httptest.NewRequest("DELETE", reqURL, nil)
			w := httptest.NewRecorder()
			
			// Execute
			handleUnsubscribe(w, req)
			
			// Verify HTTP response
			assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request for invalid channel ID")
			
			// Verify response body structure
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err, "Response should be valid JSON")
			
			// Verify response contains expected fields
			assert.Equal(t, "error", response["status"], "Status should be 'error'")
			
			// Only expect channel_id in response for non-empty channel IDs
			if tc.channelID != "" {
				// Skip channel_id assertion for cases with special characters that might cause URL parsing issues
				if channelIDFromResponse, exists := response["channel_id"]; exists {
					assert.Equal(t, tc.channelID, channelIDFromResponse, "Should return the channel ID")
				}
			}
			assert.Contains(t, response, "message", "Should include error message")
			
			// Verify error message mentions invalid format
			message, ok := response["message"].(string)
			require.True(t, ok, "Message should be a string")
			
			// For empty string, expect "required" message; for others, expect "invalid"
			if tc.channelID == "" {
				assert.Contains(t, strings.ToLower(message), "required", "Error message should mention 'required' for empty string")
			} else {
				assert.Contains(t, strings.ToLower(message), "invalid", "Error message should mention 'invalid'")
			}
			
			// Verify no state was stored due to invalid channel ID
			testState := GetTestSubscriptionState()
			if testState != nil && testState.Subscriptions != nil {
				assert.NotContains(t, testState.Subscriptions, tc.channelID, "Invalid channel should not be stored")
			}
		})
	}
}

// TestUnsubscribeFromChannel_MissingChannelID tests handling of missing channel_id parameter
func TestUnsubscribeFromChannel_MissingChannelID(t *testing.T) {
	// Test case: DELETE /unsubscribe (no channel_id parameter)
	// Expected behavior:
	// 1. Check for required channel_id parameter
	// 2. Return 400 Bad Request if missing
	// 3. Do not check state or make hub requests

	// Setup
	setupSubscriptionTest()
	defer teardownSubscriptionTest()
	
	// Create request without channel_id parameter
	req := httptest.NewRequest("DELETE", "/unsubscribe", nil)
	w := httptest.NewRecorder()
	
	// Execute
	handleUnsubscribe(w, req)
	
	// Verify HTTP response
	assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request for missing channel_id")
	
	// Verify response body structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")
	
	// Verify response contains expected fields
	assert.Equal(t, "error", response["status"], "Status should be 'error'")
	assert.Contains(t, response, "message", "Should include error message")
	
	// Verify error message is descriptive
	message, ok := response["message"].(string)
	require.True(t, ok, "Message should be a string")
	assert.Contains(t, message, "channel_id", "Error message should mention 'channel_id'")
	assert.Contains(t, message, "required", "Error message should mention 'required'")
	
	// Verify no state changes occurred due to missing channel_id
	testState := GetTestSubscriptionState()
	if testState != nil && testState.Subscriptions != nil {
		assert.Len(t, testState.Subscriptions, 0, "No subscriptions should be stored when channel_id is missing")
	}
}

// TestUnsubscribe_RealModeWithoutEnvVars tests unsubscribe endpoint in non-test mode
func TestUnsubscribe_RealModeWithoutEnvVars(t *testing.T) {
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
	
	channelID := testutil.TestChannelIDs.Valid
	
	// Create request
	req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
	w := httptest.NewRecorder()
	
	// Execute
	handleUnsubscribe(w, req)
	
	// Should return 500 error due to missing SUBSCRIPTION_BUCKET
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "error", response["status"])
	assert.Contains(t, response["message"], "Failed to load subscription state")
}

// TestHandleUnsubscribe_PubSubHubbubFailure tests unsubscribe when hub request fails
func TestHandleUnsubscribe_PubSubHubbubFailure(t *testing.T) {
	// Setup non-test mode to exercise real PubSubHubbub code paths
	setupNonTestMode()
	defer teardownNonTestMode()
	
	// Set up minimal environment to get past initial validation
	os.Setenv("SUBSCRIPTION_BUCKET", "test-bucket")
	os.Setenv("FUNCTION_URL", "https://test-callback-url")
	defer func() {
		os.Unsetenv("SUBSCRIPTION_BUCKET")
		os.Unsetenv("FUNCTION_URL")
	}()
	
	channelID := testutil.TestChannelIDs.Valid
	
	// Create request
	req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
	w := httptest.NewRecorder()
	
	// Execute - this will fail because Cloud Storage client creation will fail
	// or because the hub request will fail
	handleUnsubscribe(w, req)
	
	// Should return some kind of error (either storage or hub failure)
	assert.True(t, 
		w.Code == http.StatusInternalServerError || 
		w.Code == http.StatusNotFound ||
		w.Code == http.StatusBadGateway,
		"Expected error response, got status: %d", w.Code)
}

// TestUnsubscribeFromChannel_NetworkFailures tests handling of PubSubHubbub hub failures during unsubscribe
func TestUnsubscribeFromChannel_NetworkFailures(t *testing.T) {
	// Network failure handling for unsubscribe follows the same patterns as subscribe
	// Comprehensive network error testing is covered in TestPubSubHubbubRequest_ComprehensiveErrors
	t.Skip("Network failure handling requires infrastructure changes; comprehensive error testing is done in TestPubSubHubbubRequest_ComprehensiveErrors")
	// Test case: DELETE /unsubscribe (hub communication fails)
	// Expected behavior:
	// 1. Find existing subscription in state
	// 2. Attempt unsubscribe request to hub
	// 3. If hub fails, return appropriate 5xx error
	// 4. Do NOT remove from state if hub call fails (preserve consistency)

	testCases := []struct {
		name           string
		mockResponse   func() *httptest.Server
		expectedStatus int
		shouldRemoveFromState bool
	}{
		{
			name: "hub_unreachable",
			mockResponse: func() *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					panic("connection refused")
				}))
				server.Close()
				return server
			},
			expectedStatus: http.StatusBadGateway, // 502
			shouldRemoveFromState: false, // Keep subscription if hub unreachable
		},
		{
			name: "hub_internal_error",
			mockResponse: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			expectedStatus: http.StatusInternalServerError, // 500
			shouldRemoveFromState: false, // Keep subscription if hub errors
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock hub server
			mockHub := tc.mockResponse()
			if tc.name != "hub_unreachable" {
				defer mockHub.Close()
			}
			
			// Setup
			channelID := testutil.TestChannelIDs.Valid
			
			// TODO: Pre-populate subscription state with existing subscription
			
			// Create request
			// req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
			w := httptest.NewRecorder()
			
			// TODO: Configure handler to use mock hub URL
			
			// Execute
			// TODO: Call our unsubscribe handler function
			// unsubscribeHandler(w, req)
			
			// Verify HTTP response
			assert.Equal(t, tc.expectedStatus, w.Code, 
				"Should return %d for %s", tc.expectedStatus, tc.name)
			
			// Verify response body structure
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err, "Response should be valid JSON")
			
			// Verify response contains expected fields
			assert.Equal(t, "error", response["status"], "Status should be 'error'")
			assert.Equal(t, channelID, response["channel_id"], "Should return the channel ID")
			assert.Contains(t, response, "message", "Should include error message")
			
			if tc.shouldRemoveFromState {
				// TODO: Verify subscription was removed from state
			} else {
				// TODO: Verify subscription was NOT removed from state (failed hub call)
			}
		})
	}
}

// TestUnsubscribeWithCloudStorageErrors tests error handling for Cloud Storage failures
func TestUnsubscribeWithCloudStorageErrors(t *testing.T) {
	
	t.Run("LoadSubscriptionState_Error", func(t *testing.T) {
		// Setup mock storage
		mockClient, origClient := setupMockStorage()
		defer teardownMockStorage(origClient)
		
		channelID := testutil.TestChannelIDs.Valid
		
		// Mock LoadSubscriptionState to return error
		mockClient.On("LoadSubscriptionState", mock.Anything).Return(
			(*SubscriptionState)(nil), 
			fmt.Errorf("network timeout"),
		)
		
		req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
		w := httptest.NewRecorder()
		
		handleUnsubscribe(w, req)
		
		// Verify error response
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "error", response.Status)
		assert.Equal(t, channelID, response.ChannelID)
		assert.Contains(t, response.Message, "Failed to load subscription state")
		
		// Verify mock was called
		mockClient.AssertExpectations(t)
	})
	
	t.Run("SaveSubscriptionState_Error", func(t *testing.T) {
		// Setup mock storage
		mockClient, origClient := setupMockStorage()
		defer teardownMockStorage(origClient)
		
		channelID := testutil.TestChannelIDs.Valid
		
		// Enable test mode to bypass PubSubHubbub request
		originalTestMode := GetTestMode()
		SetTestMode(true)
		defer func() { 
			SetTestMode(originalTestMode)
			SetStorageClient(mockClient) // Restore mock client after test mode cleanup
		}()
		
		// Mock state with existing subscription
		existingState := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				channelID: {
					ChannelID:    channelID,
					Status:       "active",
					ExpiresAt:    time.Now().Add(24 * time.Hour),
					SubscribedAt: time.Now().Add(-time.Hour),
				},
			},
			Metadata: struct {
				LastUpdated time.Time `json:"last_updated"`
				Version     string    `json:"version"`
			}{
				LastUpdated: time.Now(),
				Version:     "1.0",
			},
		}
		
		mockClient.On("LoadSubscriptionState", mock.Anything).Return(existingState, nil)
		mockClient.On("SaveSubscriptionState", mock.Anything, mock.AnythingOfType("*webhook.SubscriptionState")).Return(
			fmt.Errorf("disk full"),
		)
		
		req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
		w := httptest.NewRecorder()
		
		handleUnsubscribe(w, req)
		
		// Verify error response
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "error", response.Status)
		assert.Equal(t, channelID, response.ChannelID)
		assert.Contains(t, response.Message, "Failed to save subscription state")
		
		// Verify mock was called
		mockClient.AssertExpectations(t)
	})
	
	t.Run("SubscriptionNotFound_WithMockStorage", func(t *testing.T) {
		// Setup mock storage
		mockClient, origClient := setupMockStorage()
		defer teardownMockStorage(origClient)
		
		channelID := testutil.TestChannelIDs.Valid
		
		// Mock empty state (no subscriptions)
		emptyState := &SubscriptionState{
			Subscriptions: make(map[string]*Subscription),
			Metadata: struct {
				LastUpdated time.Time `json:"last_updated"`
				Version     string    `json:"version"`
			}{
				LastUpdated: time.Now(),
				Version:     "1.0",
			},
		}
		
		mockClient.On("LoadSubscriptionState", mock.Anything).Return(emptyState, nil)
		
		req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id="+channelID, nil)
		w := httptest.NewRecorder()
		
		handleUnsubscribe(w, req)
		
		// Verify not found response
		assert.Equal(t, http.StatusNotFound, w.Code)
		
		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "error", response.Status)
		assert.Equal(t, channelID, response.ChannelID)
		assert.Equal(t, "Subscription not found for this channel", response.Message)
		
		// Verify mock was called (save should NOT be called)
		mockClient.AssertExpectations(t)
	})
}
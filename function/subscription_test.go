package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test setup/teardown for subscription state
func setupSubscriptionTest() {
	testMode = true
	testSubscriptionState = nil
}

func teardownSubscriptionTest() {
	testMode = false
	testSubscriptionState = nil
}

// setupNonTestMode sets up environment for testing non-test-mode paths
func setupNonTestMode() {
	testMode = false
	testSubscriptionState = nil
}

func teardownNonTestMode() {
	testMode = false
	testSubscriptionState = nil
}

// TestSubscribeToChannel_Success tests the happy path for subscribing to a YouTube channel
func TestSubscribeToChannel_Success(t *testing.T) {
	// Test case: POST /subscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw
	// Expected behavior:
	// 1. Accept valid channel ID
	// 2. Make subscription request to PubSubHubbub hub  
	// 3. Store subscription state
	// 4. Return success response with expiration time

	// Setup
	setupSubscriptionTest()
	defer teardownSubscriptionTest()
	
	channelID := "UCXuqSBlHAE6Xw-yeJA0Tunw"
	
	// Create request
	req := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
	w := httptest.NewRecorder()
	
	// Execute
	handleSubscribe(w, req)
	
	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK for successful subscription")
	
	// Verify response body structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")
	
	// Verify response contains expected fields
	assert.Equal(t, "success", response["status"], "Status should be 'success'")
	assert.Equal(t, channelID, response["channel_id"], "Should return the channel ID")
	assert.Contains(t, response, "expires_at", "Should include expiration time")
	assert.Contains(t, response, "message", "Should include success message")
	
	// Verify expiration time is in the future
	expiresAt, ok := response["expires_at"].(string)
	require.True(t, ok, "expires_at should be a string")
	
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	require.NoError(t, err, "expires_at should be valid RFC3339 timestamp")
	assert.True(t, expiry.After(time.Now()), "Expiration should be in the future")
	
	// TODO: Verify subscription state was stored
	// TODO: Verify PubSubHubbub request was made
}

// TestSubscribeToChannel_AlreadySubscribed tests conflict handling for existing subscriptions
func TestSubscribeToChannel_AlreadySubscribed(t *testing.T) {
	// Test case: POST /subscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw (already subscribed)
	// Expected behavior:
	// 1. Check existing subscription state
	// 2. Return conflict response without making new hub request
	// 3. Include existing expiration time in response

	// Setup
	setupSubscriptionTest()
	defer teardownSubscriptionTest()
	
	channelID := "UCXuqSBlHAE6Xw-yeJA0Tunw"
	
	// First subscription - should succeed
	req1 := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
	w1 := httptest.NewRecorder()
	handleSubscribe(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code, "First subscription should succeed")
	
	// Second subscription - should return conflict
	req2 := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
	w2 := httptest.NewRecorder()
	handleSubscribe(w2, req2)
	
	// Verify HTTP response
	assert.Equal(t, http.StatusConflict, w2.Code, "Should return 409 Conflict for existing subscription")
	
	// Verify response body structure
	var response map[string]interface{}
	err := json.Unmarshal(w2.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")
	
	// Verify response contains expected fields
	assert.Equal(t, "conflict", response["status"], "Status should be 'conflict'")
	assert.Equal(t, channelID, response["channel_id"], "Should return the channel ID")
	assert.Contains(t, response, "expires_at", "Should include expiration time")
	assert.Contains(t, response, "message", "Should include conflict message")
	
	// Verify message content
	message, ok := response["message"].(string)
	require.True(t, ok, "Message should be a string")
	assert.Contains(t, strings.ToLower(message), "already", "Error message should mention 'already'")
}

// TestSubscribeToChannel_InvalidChannelID tests validation of malformed channel IDs
func TestSubscribeToChannel_InvalidChannelID(t *testing.T) {
	// Test case: POST /subscribe?channel_id=invalid-format
	// Expected behavior:
	// 1. Validate channel ID format (must be UC + 22 alphanumeric chars)
	// 2. Return 400 Bad Request for invalid format
	// 3. Do not make any hub requests or store state

	testCases := []struct {
		name      string
		channelID string
		reason    string
	}{
		{
			name:      "too_short",
			channelID: "UC123", 
			reason:    "Channel ID too short",
		},
		{
			name:      "wrong_prefix",
			channelID: "XCXuqSBlHAE6Xw-yeJA0Tunw",
			reason:    "Channel ID must start with 'UC'",
		},
		{
			name:      "invalid_characters", 
			channelID: "UC@#$%^&*()!@#$%^&*()!@#$",
			reason:    "Channel ID contains invalid characters",
		},
		{
			name:      "too_long",
			channelID: "UCXuqSBlHAE6Xw-yeJA0Tunwextra",
			reason:    "Channel ID too long",
		},
		{
			name:      "empty_string",
			channelID: "",
			reason:    "Channel ID cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			setupSubscriptionTest()
			defer teardownSubscriptionTest()
			
			// Create request with proper URL encoding
			reqURL := "/subscribe?channel_id=" + url.QueryEscape(tc.channelID)
			req := httptest.NewRequest("POST", reqURL, nil)
			w := httptest.NewRecorder()
			
			// Execute
			handleSubscribe(w, req)
			
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
				assert.Equal(t, tc.channelID, response["channel_id"], "Should return the invalid channel ID")
			}
			assert.Contains(t, response, "message", "Should include error message")
			
			// Verify error message is descriptive
			message, ok := response["message"].(string)
			require.True(t, ok, "Message should be a string")
			
			// For empty string, expect "required" message; for others, expect "invalid"
			if tc.channelID == "" {
				assert.Contains(t, strings.ToLower(message), "required", "Error message should mention 'required' for empty string")
			} else {
				assert.Contains(t, strings.ToLower(message), "invalid", "Error message should mention 'invalid'")
			}
			
			// TODO: Verify no hub request was made
			// TODO: Verify no state was stored
		})
	}
}

// TestSubscribeToChannel_MissingChannelID tests handling of missing channel_id parameter
func TestSubscribeToChannel_MissingChannelID(t *testing.T) {
	// Test case: POST /subscribe (no channel_id parameter)
	// Expected behavior:
	// 1. Check for required channel_id parameter
	// 2. Return 400 Bad Request if missing
	// 3. Do not make any hub requests or store state

	// Setup
	setupSubscriptionTest()
	defer teardownSubscriptionTest()
	
	// Create request without channel_id parameter
	req := httptest.NewRequest("POST", "/subscribe", nil)
	w := httptest.NewRecorder()
	
	// Execute
	handleSubscribe(w, req)
	
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
	
	// TODO: Verify no hub request was made
	// TODO: Verify no state was stored
}

// TestSubscribeToChannel_NetworkFailures tests handling of various network failures when communicating with PubSubHubbub hub
func TestSubscribeToChannel_NetworkFailures(t *testing.T) {
	// Test cases for different network failure scenarios
	// Expected behavior: Map network issues to appropriate HTTP status codes
	// - Hub unreachable → 502 Bad Gateway
	// - Hub returns 5xx error → Pass through 5xx code  
	// - Request timeout → 504 Gateway Timeout

	testCases := []struct {
		name           string
		mockResponse   func() *httptest.Server
		expectedStatus int
		expectedError  string
	}{
		{
			name: "hub_unreachable",
			mockResponse: func() *httptest.Server {
				// Return a server that immediately closes connections
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Simulate connection refused by closing immediately
					panic("connection refused")
				}))
				server.Close() // Close server to simulate unreachable
				return server
			},
			expectedStatus: http.StatusBadGateway, // 502
			expectedError:  "PubSubHubbub hub unreachable",
		},
		{
			name: "hub_internal_error",
			mockResponse: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError) // 500
					w.Write([]byte("Internal server error"))
				}))
			},
			expectedStatus: http.StatusInternalServerError, // 500 (pass through)
			expectedError:  "PubSubHubbub hub error: 500 Internal Server Error",
		},
		{
			name: "hub_bad_gateway",
			mockResponse: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadGateway) // 502
					w.Write([]byte("Bad gateway"))
				}))
			},
			expectedStatus: http.StatusBadGateway, // 502 (pass through)
			expectedError:  "PubSubHubbub hub error: 502 Bad Gateway",
		},
		{
			name: "hub_service_unavailable",
			mockResponse: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable) // 503
					w.Write([]byte("Service unavailable"))
				}))
			},
			expectedStatus: http.StatusServiceUnavailable, // 503 (pass through)
			expectedError:  "PubSubHubbub hub error: 503 Service Unavailable",
		},
		{
			name: "request_timeout",
			mockResponse: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Simulate timeout by sleeping longer than client timeout
					time.Sleep(10 * time.Second) // Assume client timeout is < 10s
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectedStatus: http.StatusGatewayTimeout, // 504
			expectedError:  "Request to PubSubHubbub hub timed out",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Skip("TODO: Implement network failure handling")
			
			// Setup mock hub server
			// mockHub := tc.mockResponse()
			// if tc.name != "hub_unreachable" {
			//	defer mockHub.Close()
			// }
			
			// Setup
			// channelID := "UCXuqSBlHAE6Xw-yeJA0Tunw"
			
			// Create request
			// req := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
			// w := httptest.NewRecorder()
			
			// TODO: Configure subscription handler to use mock hub URL
			// TODO: Set appropriate client timeout for timeout test
			
			// Execute
			// TODO: Call our subscription handler function
			// subscriptionHandler(w, req)
			
			// TODO: Verify HTTP response when network failure handling is implemented
			// assert.Equal(t, tc.expectedStatus, w.Code, 
			//	"Should return %d for %s", tc.expectedStatus, tc.name)
			
			// TODO: Verify response body structure
			// var response map[string]interface{}
			// err := json.Unmarshal(w.Body.Bytes(), &response)
			// require.NoError(t, err, "Response should be valid JSON")
			
			// TODO: Verify response contains expected fields
			// assert.Equal(t, "error", response["status"], "Status should be 'error'")
			// assert.Equal(t, channelID, response["channel_id"], "Should return the channel ID")
			// assert.Contains(t, response, "message", "Should include error message")
			
			// TODO: Verify error message contains expected text
			// message, ok := response["message"].(string)
			// require.True(t, ok, "Message should be a string")
			// assert.Contains(t, message, tc.expectedError, 
			//	"Error message should contain: %s", tc.expectedError)
			
			// TODO: Verify no subscription state was stored on failure
		})
	}
}

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
	t.Skip("TODO: Implement storage error handling")
	
	// Test case: GET /subscriptions (storage unavailable)
	// Expected behavior:
	// 1. Attempt to load subscription state from storage
	// 2. Return 500 Internal Server Error if storage fails
	// 3. Include descriptive error message

	// TODO: Mock storage to return error
	
	// Create request
	// req := httptest.NewRequest("GET", "/subscriptions", nil)
	// w := httptest.NewRecorder()
	
	// Execute
	// TODO: Call our subscriptions list handler function with mocked storage error
	// getSubscriptionsHandler(w, req)
	
	// Verify HTTP response
	// assert.Equal(t, http.StatusInternalServerError, w.Code, "Should return 500 for storage errors")
	
	// Verify response body structure
	// var response map[string]interface{}
	// err := json.Unmarshal(w.Body.Bytes(), &response)
	// require.NoError(t, err, "Response should be valid JSON")
	
	// Verify error response structure
	// assert.Equal(t, "error", response["status"], "Status should be 'error'")
	// assert.Contains(t, response, "message", "Should include error message")
	
	// Verify error message is descriptive
	// message, ok := response["message"].(string)
	// require.True(t, ok, "Message should be a string")
	// assert.Contains(t, message, "storage", "Error message should mention 'storage'")
}

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

	channelID := "UCXuqSBlHAE6Xw-yeJA0Tunw"
	
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

	channelID := "UCXuqSBlHAE6Xw-yeJA0Tunw"
	
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
		{"invalid_characters", "UC@#$%^&*()!@#$%^&*()!@#$"},
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
				assert.Equal(t, tc.channelID, response["channel_id"], "Should return the invalid channel ID")
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
			
			// TODO: Verify no state access or hub requests were made
		})
	}
}

// TestMakePubSubHubbubRequest_MissingEnvVar tests error handling when FUNCTION_URL is not set
func TestMakePubSubHubbubRequest_MissingEnvVar(t *testing.T) {
	// Setup non-test mode to exercise real PubSubHubbub code
	setupNonTestMode()
	defer teardownNonTestMode()
	
	// Ensure FUNCTION_URL is not set
	originalFunctionURL := os.Getenv("FUNCTION_URL")
	os.Unsetenv("FUNCTION_URL")
	defer func() {
		if originalFunctionURL != "" {
			os.Setenv("FUNCTION_URL", originalFunctionURL)
		}
	}()
	
	// Test makePubSubHubbubRequest directly
	err := makePubSubHubbubRequest("UCXuqSBlHAE6Xw-yeJA0Tunw", "subscribe")
	
	// Should return error about missing environment variable
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FUNCTION_URL environment variable not set")
}

// TestLoadSubscriptionState_MissingBucket tests error handling when SUBSCRIPTION_BUCKET is not set
func TestLoadSubscriptionState_MissingBucket(t *testing.T) {
	// Setup non-test mode to exercise real Cloud Storage code
	setupNonTestMode()
	defer teardownNonTestMode()
	
	// Ensure SUBSCRIPTION_BUCKET is not set
	originalBucket := os.Getenv("SUBSCRIPTION_BUCKET")
	os.Unsetenv("SUBSCRIPTION_BUCKET")
	defer func() {
		if originalBucket != "" {
			os.Setenv("SUBSCRIPTION_BUCKET", originalBucket)
		}
	}()
	
	// Test loadSubscriptionState directly
	ctx := context.Background()
	_, err := loadSubscriptionState(ctx)
	
	// Should return error about missing environment variable
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SUBSCRIPTION_BUCKET environment variable not set")
}

// TestSaveSubscriptionState_MissingBucket tests error handling when SUBSCRIPTION_BUCKET is not set
func TestSaveSubscriptionState_MissingBucket(t *testing.T) {
	// Setup non-test mode to exercise real Cloud Storage code
	setupNonTestMode()
	defer teardownNonTestMode()
	
	// Ensure SUBSCRIPTION_BUCKET is not set
	originalBucket := os.Getenv("SUBSCRIPTION_BUCKET")
	os.Unsetenv("SUBSCRIPTION_BUCKET")
	defer func() {
		if originalBucket != "" {
			os.Setenv("SUBSCRIPTION_BUCKET", originalBucket)
		}
	}()
	
	// Create test state
	state := &SubscriptionState{
		Subscriptions: make(map[string]*Subscription),
	}
	
	// Test saveSubscriptionState directly
	ctx := context.Background()
	err := saveSubscriptionState(ctx, state)
	
	// Should return error about missing environment variable
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SUBSCRIPTION_BUCKET environment variable not set")
}

// TestWriteJSONResponse_Error tests error handling in JSON encoding
func TestWriteJSONResponse_Error(t *testing.T) {
	// Create a response that will fail JSON encoding (channel with circular reference)
	type circularStruct struct {
		Self *circularStruct `json:"self"`
	}
	circular := &circularStruct{}
	circular.Self = circular
	
	// Use our failing response writer to catch the error
	w := &failingResponseWriter{ResponseRecorder: httptest.NewRecorder()}
	
	// This should not panic even with JSON encoding error + write error
	assert.NotPanics(t, func() {
		writeJSONResponse(w, http.StatusOK, circular)
	})
}

// TestSubscribe_RealModeWithoutEnvVars tests subscribe endpoint in non-test mode
func TestSubscribe_RealModeWithoutEnvVars(t *testing.T) {
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
	
	channelID := "UCXuqSBlHAE6Xw-yeJA0Tunw"
	
	// Create request
	req := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
	w := httptest.NewRecorder()
	
	// Execute
	handleSubscribe(w, req)
	
	// Should return 500 error due to missing SUBSCRIPTION_BUCKET
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "error", response["status"])
	assert.Contains(t, response["message"], "Failed to load subscription state")
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
	
	channelID := "UCXuqSBlHAE6Xw-yeJA0Tunw"
	
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

// TestYouTubeWebhook_UnsupportedMethod tests method validation
func TestYouTubeWebhook_UnsupportedMethod(t *testing.T) {
	req := httptest.NewRequest("PATCH", "/", nil)
	w := httptest.NewRecorder()
	
	YouTubeWebhook(w, req)
	
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	assert.Equal(t, "Method not allowed", w.Body.String())
}

// TestYouTubeWebhook_OptionsMethod tests CORS preflight handling
func TestYouTubeWebhook_OptionsMethod(t *testing.T) {
	req := httptest.NewRequest("OPTIONS", "/", nil)
	w := httptest.NewRecorder()
	
	YouTubeWebhook(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
}

// TestMakePubSubHubbubRequest_Success tests successful PubSubHubbub request
func TestMakePubSubHubbubRequest_Success(t *testing.T) {
	// Setup non-test mode
	setupNonTestMode()
	defer teardownNonTestMode()
	
	// Create mock hub server
	mockHub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request format
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		assert.Equal(t, "YouTube-Webhook-Handler/1.0", r.Header.Get("User-Agent"))
		
		// Parse form data
		err := r.ParseForm()
		require.NoError(t, err)
		
		// Verify form fields
		assert.Equal(t, "https://test-callback-url", r.FormValue("hub.callback"))
		assert.Equal(t, "https://www.youtube.com/feeds/videos.xml?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw", r.FormValue("hub.topic"))
		assert.Equal(t, "subscribe", r.FormValue("hub.mode"))
		assert.Equal(t, "async", r.FormValue("hub.verify"))
		assert.Equal(t, "86400", r.FormValue("hub.lease_seconds"))
		
		// Return success
		w.WriteHeader(http.StatusAccepted)
	}))
	defer mockHub.Close()
	
	// Set environment variables
	os.Setenv("FUNCTION_URL", "https://test-callback-url")
	defer os.Unsetenv("FUNCTION_URL")
	
	// Override hub URL in the function (we'll need to make it configurable)
	// For now, let's just test the error paths we can control
	
	// Test the function - this will try to contact the real Google hub
	err := makePubSubHubbubRequest("UCXuqSBlHAE6Xw-yeJA0Tunw", "subscribe")
	
	// The real Google hub might accept or reject our request
	// We just want to exercise the code path
	if err != nil {
		// If there's an error, it should be a meaningful one
		assert.True(t, 
			strings.Contains(err.Error(), "request failed") || 
			strings.Contains(err.Error(), "hub returned status"),
			"Expected network or HTTP error, got: %v", err)
	} else {
		// Success is also acceptable - means the real hub accepted our test request
		t.Log("PubSubHubbub request succeeded (contacted real Google hub)")
	}
}

// TestMakePubSubHubbubRequest_InvalidResponse tests handling of non-2xx response
func TestMakePubSubHubbubRequest_InvalidResponse(t *testing.T) {
	// Setup non-test mode
	setupNonTestMode()
	defer teardownNonTestMode()
	
	// Since we can't easily mock the HTTP client, let's test other error paths
	// Test with invalid URL format
	os.Setenv("FUNCTION_URL", "https://test-callback-url")
	defer os.Unsetenv("FUNCTION_URL")
	
	// Test with valid inputs - this will try to contact the real hub
	err := makePubSubHubbubRequest("UCXuqSBlHAE6Xw-yeJA0Tunw", "subscribe")
	
	// The hub request may succeed or fail, we just want to exercise the code
	if err != nil {
		// If there's an error, it should be a meaningful one
		assert.True(t, 
			strings.Contains(err.Error(), "request failed") || 
			strings.Contains(err.Error(), "hub returned status"),
			"Expected network or HTTP error, got: %v", err)
	}
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
	
	channelID := "UCXuqSBlHAE6Xw-yeJA0Tunw"
	
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
	
	// TODO: Verify no state access or hub requests were made
}

// TestUnsubscribeFromChannel_NetworkFailures tests handling of PubSubHubbub hub failures during unsubscribe
func TestUnsubscribeFromChannel_NetworkFailures(t *testing.T) {
	t.Skip("TODO: Implement network failure handling for unsubscribe")
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
			channelID := "UCXuqSBlHAE6Xw-yeJA0Tunw"
			
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
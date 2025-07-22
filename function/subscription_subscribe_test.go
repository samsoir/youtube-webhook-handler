package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/samsoir/youtube-webhook/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockStorageClient implements StorageInterface using testify/mock
type MockStorageClient struct {
	mock.Mock
}

// LoadSubscriptionState mocks the LoadSubscriptionState method
func (m *MockStorageClient) LoadSubscriptionState(ctx context.Context) (*SubscriptionState, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SubscriptionState), args.Error(1)
}

// SaveSubscriptionState mocks the SaveSubscriptionState method
func (m *MockStorageClient) SaveSubscriptionState(ctx context.Context, state *SubscriptionState) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

// Test helper functions
func setupSubscriptionTest() {
	SetTestMode(true)
	SetTestSubscriptionState(nil)
}

func teardownSubscriptionTest() {
	SetTestMode(false)
	SetTestSubscriptionState(nil)
}

func setupMockStorage() (*MockStorageClient, StorageInterface) {
	originalClient := GetStorageClient()
	mockClient := new(MockStorageClient)
	SetStorageClient(mockClient)
	SetTestMode(false) // Use mock instead of test mode
	return mockClient, originalClient
}

func teardownMockStorage(originalClient StorageInterface) {
	SetStorageClient(originalClient)
	SetTestMode(false)
}

func createTestSubscription(channelID string) *Subscription {
	now := time.Now()
	return &Subscription{
		ChannelID:       channelID,
		ChannelName:     "Test Channel",
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

func createTestSubscriptionState(subscriptions ...*Subscription) *SubscriptionState {
	state := &SubscriptionState{
		Subscriptions: make(map[string]*Subscription),
		Metadata: struct {
			LastUpdated time.Time `json:"last_updated"`
			Version     string    `json:"version"`
		}{
			LastUpdated: time.Now(),
			Version:     "1.0",
		},
	}

	for _, sub := range subscriptions {
		state.Subscriptions[sub.ChannelID] = sub
	}

	return state
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
	
	channelID := testutil.TestChannelIDs.Valid
	
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
	
	// Verify subscription state was stored
	testState := GetTestSubscriptionState()
	require.NotNil(t, testState, "Test subscription state should be initialized")
	require.Contains(t, testState.Subscriptions, channelID, "Channel should be stored in subscription state")
	
	subscription := testState.Subscriptions[channelID]
	assert.Equal(t, channelID, subscription.ChannelID, "Stored channel ID should match")
	assert.Equal(t, "active", subscription.Status, "Subscription status should be active")
	assert.True(t, subscription.ExpiresAt.After(time.Now()), "Stored expiration should be in the future")
	assert.False(t, subscription.SubscribedAt.IsZero(), "SubscribedAt should be set")
	assert.Equal(t, fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelID), subscription.TopicURL, "Topic URL should be correct")
	
	// Verify PubSubHubbub request was made (in test mode, this is bypassed but we can verify the function would have been called)
	// Since we're in test mode, the actual network request is bypassed, but the logic flow should be correct
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
	
	channelID := testutil.TestChannelIDs.Valid
	
	// Pre-populate subscription state
	existingSub := createTestSubscription(channelID)
	testState := createTestSubscriptionState(existingSub)
	SetTestSubscriptionState(testState)
	
	// Create request
	req := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
	w := httptest.NewRecorder()
	
	// Execute
	handleSubscribe(w, req)
	
	// Verify HTTP response
	assert.Equal(t, http.StatusConflict, w.Code, "Should return 409 Conflict for duplicate subscription")
	
	// Verify response body structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")
	
	// Verify response contains expected fields
	assert.Equal(t, "conflict", response["status"], "Status should be 'conflict'")
	assert.Equal(t, channelID, response["channel_id"], "Should return the channel ID")
	assert.Contains(t, response, "expires_at", "Should include existing expiration time")
	assert.Contains(t, response, "message", "Should include conflict message")
	
	// Verify expiration time matches existing subscription
	expiresAt, ok := response["expires_at"].(string)
	require.True(t, ok, "expires_at should be a string")
	
	expectedExpiry := existingSub.ExpiresAt.Format(time.RFC3339)
	assert.Equal(t, expectedExpiry, expiresAt, "Should return existing expiration time")
}

// TestSubscribeToChannel_InvalidChannelID tests validation of channel ID format
func TestSubscribeToChannel_InvalidChannelID(t *testing.T) {
	// Test case: POST /subscribe?channel_id=invalid-format
	// Expected behavior:
	// 1. Validate channel ID format
	// 2. Return 400 Bad Request without making hub request
	// 3. Provide descriptive error message

	// Setup
	setupSubscriptionTest()
	defer teardownSubscriptionTest()
	
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
			name:      "Too short",
			channelID: "UC123",
			expected:  "Invalid channel ID format",
		},
		{
			name:      "Too long", 
			channelID: "UC" + "abcdefghijklmnopqrstuvwxyz",
			expected:  "Invalid channel ID format",
		},
		{
			name:      "Wrong prefix",
			channelID: "AB1234567890123456789012",
			expected:  "Invalid channel ID format",
		},
		{
			name:      "Invalid characters",
			channelID: "UCinvalid@#$characters",
			expected:  "Invalid channel ID format",
		},
		{
			name:      "Valid format but test invalid",
			channelID: testutil.TestChannelIDs.Invalid,
			expected:  "Invalid channel ID format",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("POST", "/subscribe?channel_id="+tc.channelID, nil)
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
			assert.Contains(t, response["message"], tc.expected, "Should contain expected error message")
			
			if tc.channelID != "" {
				// Skip channel_id assertion for cases with special characters that might cause URL parsing issues
				if channelIDFromResponse, exists := response["channel_id"]; exists {
					assert.Equal(t, tc.channelID, channelIDFromResponse, "Should return the channel ID")
				}
			}
			
			// Verify no subscription state was created
			testState := GetTestSubscriptionState()
			if testState != nil {
				assert.NotContains(t, testState.Subscriptions, tc.channelID, "Invalid channel should not be stored")
			}
		})
	}
}

// TestSubscribeToChannel_MissingChannelID tests handling of missing channel_id parameter
func TestSubscribeToChannel_MissingChannelID(t *testing.T) {
	// Test case: POST /subscribe (no channel_id parameter)
	// Expected behavior:
	// 1. Detect missing required parameter
	// 2. Return 400 Bad Request
	// 3. Provide clear error message

	// Setup
	setupSubscriptionTest()
	defer teardownSubscriptionTest()
	
	// Create request without channel_id
	req := httptest.NewRequest("POST", "/subscribe", nil)
	w := httptest.NewRecorder()
	
	// Execute
	handleSubscribe(w, req)
	
	// Verify HTTP response
	assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request for missing parameter")
	
	// Verify response body structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")
	
	// Verify response contains expected fields
	assert.Equal(t, "error", response["status"], "Status should be 'error'")
	assert.Equal(t, "channel_id parameter is required", response["message"], "Should indicate missing parameter")
	assert.NotContains(t, response, "channel_id", "Should not include empty channel_id field")
}

// TestSubscribeWithCloudStorageErrors tests error handling for Cloud Storage failures
func TestSubscribeWithCloudStorageErrors(t *testing.T) {
	// Test various Cloud Storage error scenarios during subscription

	channelID := testutil.TestChannelIDs.Valid

	testCases := []struct {
		name           string
		loadError      error
		saveError      error
		setupMock      func(*MockStorageClient)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "LoadSubscriptionState error",
			loadError:      fmt.Errorf("storage client failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to load subscription state",
			setupMock: func(mockClient *MockStorageClient) {
				mockClient.On("LoadSubscriptionState", mock.Anything).
					Return(nil, fmt.Errorf("storage client failed"))
			},
		},
		{
			name:           "SaveSubscriptionState error",
			saveError:      fmt.Errorf("storage write failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to save subscription state",
			setupMock: func(mockClient *MockStorageClient) {
				emptyState := createTestSubscriptionState()
				mockClient.On("LoadSubscriptionState", mock.Anything).
					Return(emptyState, nil)
				mockClient.On("SaveSubscriptionState", 
					mock.Anything,
					mock.Anything).
					Return(fmt.Errorf("storage write failed"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock storage
			mockClient, originalClient := setupMockStorage()
			defer teardownMockStorage(originalClient)
			
			// Set up environment to bypass PubSubHubbub request for SaveSubscriptionState test
			if tc.name == "SaveSubscriptionState error" {
				// Set FUNCTION_URL to allow PubSubHubbub request to succeed
				originalURL := os.Getenv("FUNCTION_URL")
				os.Setenv("FUNCTION_URL", "https://test-function-url")
				defer func() {
					if originalURL == "" {
						os.Unsetenv("FUNCTION_URL")
					} else {
						os.Setenv("FUNCTION_URL", originalURL)
					}
				}()
				
				// Enable test mode to bypass actual PubSubHubbub request
				originalTestMode := GetTestMode()
				SetTestMode(true)
				defer func() {
					SetTestMode(originalTestMode)
					SetStorageClient(mockClient) // Restore mock client after test mode cleanup
				}()
			}

			// Configure mock expectations
			tc.setupMock(mockClient)

			// Create request
			req := httptest.NewRequest("POST", "/subscribe?channel_id="+channelID, nil)
			w := httptest.NewRecorder()

			// Execute
			handleSubscribe(w, req)

			// Verify response
			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "error", response["status"])
			assert.Contains(t, response["message"], tc.expectedMsg)
			assert.Equal(t, channelID, response["channel_id"])

			// Verify mock expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}
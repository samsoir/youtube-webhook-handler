package webhook

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestYouTubeWebhookRefactored_Subscribe(t *testing.T) {
	// Create test request
	req := httptest.NewRequest("POST", "/subscribe?channel_id=UCabcdefghijklmnopqrstuv", nil)
	rec := httptest.NewRecorder()

	// Call refactored router
	YouTubeWebhookRefactored(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check that response contains expected content
	body := rec.Body.String()
	if !strings.Contains(body, "success") {
		t.Errorf("Expected success response, got: %s", body)
	}
}

func TestYouTubeWebhookRefactored_Unsubscribe(t *testing.T) {
	// First create a subscription for testing
	deps := CreateTestDependencies()
	SetDependencies(deps)
	
	// Add a test subscription
	state, _ := deps.StorageClient.LoadSubscriptionState(nil)
	state.Subscriptions["UCabcdefghijklmnopqrstuv"] = &Subscription{
		ChannelID:    "UCabcdefghijklmnopqrstuv",
		Status:       "active",
		SubscribedAt: time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}
	deps.StorageClient.SaveSubscriptionState(nil, state)

	// Create test request
	req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id=UCabcdefghijklmnopqrstuv", nil)
	rec := httptest.NewRecorder()

	// Call refactored router
	YouTubeWebhookRefactored(rec, req)

	// Verify response
	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestYouTubeWebhookRefactored_GetSubscriptions(t *testing.T) {
	// Create test request
	req := httptest.NewRequest("GET", "/subscriptions", nil)
	rec := httptest.NewRecorder()

	// Call refactored router
	YouTubeWebhookRefactored(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check that response contains expected JSON structure
	body := rec.Body.String()
	if !strings.Contains(body, "subscriptions") {
		t.Errorf("Expected subscriptions in response, got: %s", body)
	}
}

func TestYouTubeWebhookRefactored_RenewSubscriptions(t *testing.T) {
	// Create test request
	req := httptest.NewRequest("POST", "/renew", nil)
	rec := httptest.NewRecorder()

	// Call refactored router
	YouTubeWebhookRefactored(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check that response contains expected content
	body := rec.Body.String()
	if !strings.Contains(body, "success") {
		t.Errorf("Expected success response, got: %s", body)
	}
}

func TestYouTubeWebhookRefactored_VerificationChallenge(t *testing.T) {
	// Create test request with challenge parameter
	req := httptest.NewRequest("GET", "/?hub.challenge=test-challenge-123", nil)
	rec := httptest.NewRecorder()

	// Call refactored router
	YouTubeWebhookRefactored(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check that response contains the challenge
	body := rec.Body.String()
	if body != "test-challenge-123" {
		t.Errorf("Expected challenge response 'test-challenge-123', got: %s", body)
	}
}

func TestYouTubeWebhookRefactored_Notification(t *testing.T) {
	// Set environment variables for GitHub integration
	os.Setenv("REPO_OWNER", "test-owner")
	os.Setenv("REPO_NAME", "test-repo")
	defer func() {
		os.Unsetenv("REPO_OWNER")
		os.Unsetenv("REPO_NAME")
	}()

	// Create test XML for new video (use recent timestamp for IsNewVideo logic)
	now := time.Now()
	published := now.Add(-10 * time.Minute).Format(time.RFC3339)
	updated := now.Add(-9 * time.Minute).Format(time.RFC3339)

	testXML := fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8'?>
<feed xmlns:yt="http://www.youtube.com/xml/schemas/2015"
      xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>yt:video:test123</id>
    <yt:videoId>test123</yt:videoId>
    <yt:channelId>UC123456789012345678901</yt:channelId>
    <title>Test Video</title>
    <published>%s</published>
    <updated>%s</updated>
  </entry>
</feed>`, published, updated)

	// Create test request
	req := httptest.NewRequest("POST", "/", strings.NewReader(testXML))
	rec := httptest.NewRecorder()

	// Call refactored router
	YouTubeWebhookRefactored(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestYouTubeWebhookRefactored_OptionsRequest(t *testing.T) {
	// Create test OPTIONS request
	req := httptest.NewRequest("OPTIONS", "/", nil)
	rec := httptest.NewRecorder()

	// Call refactored router
	YouTubeWebhookRefactored(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestYouTubeWebhookRefactored_MethodNotAllowed(t *testing.T) {
	// Create test request with unsupported method
	req := httptest.NewRequest("PATCH", "/", nil)
	rec := httptest.NewRecorder()

	// Call refactored router
	YouTubeWebhookRefactored(rec, req)

	// Verify response
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}

	// Check response body
	body := rec.Body.String()
	if body != "Method not allowed" {
		t.Errorf("Expected 'Method not allowed', got: %s", body)
	}
}

func TestYouTubeWebhookRefactored_CORSHeaders(t *testing.T) {
	// Create test request
	req := httptest.NewRequest("GET", "/subscriptions", nil)
	rec := httptest.NewRecorder()

	// Call refactored router
	YouTubeWebhookRefactored(rec, req)

	// Verify CORS headers are set
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected CORS origin header to be '*', got: %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	if rec.Header().Get("Access-Control-Allow-Methods") != "GET, POST, DELETE, OPTIONS" {
		t.Errorf("Expected CORS methods header, got: %s", rec.Header().Get("Access-Control-Allow-Methods"))
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type header to be 'application/json', got: %s", rec.Header().Get("Content-Type"))
	}
}

func TestHandleGetSubscriptionsWithDeps(t *testing.T) {
	// Create test dependencies
	deps := CreateTestDependencies()

	// Add some test subscriptions
	state, _ := deps.StorageClient.LoadSubscriptionState(nil)
	now := time.Now()
	state.Subscriptions["UCabcdefghijklmnopqrstuv"] = &Subscription{
		ChannelID:    "UCabcdefghijklmnopqrstuv",
		Status:       "active",
		SubscribedAt: now,
		ExpiresAt:    now.Add(24 * time.Hour),
	}
	state.Subscriptions["UCexpiredchannel123456789"] = &Subscription{
		ChannelID:    "UCexpiredchannel123456789",
		Status:       "expired",
		SubscribedAt: now.Add(-48 * time.Hour),
		ExpiresAt:    now.Add(-1 * time.Hour), // Expired
	}
	deps.StorageClient.SaveSubscriptionState(nil, state)

	// Create test request
	req := httptest.NewRequest("GET", "/subscriptions", nil)
	rec := httptest.NewRecorder()

	// Create and call handler
	handler := handleGetSubscriptionsWithDeps(deps)
	handler(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check response content
	body := rec.Body.String()
	if !strings.Contains(body, "UCabcdefghijklmnopqrstuv") {
		t.Errorf("Expected active subscription in response, got: %s", body)
	}
	if !strings.Contains(body, "UCexpiredchannel123456789") {
		t.Errorf("Expected expired subscription in response, got: %s", body)
	}
	if !strings.Contains(body, `"total":2`) {
		t.Errorf("Expected total:2 in response, got: %s", body)
	}
}

func TestHandleGetSubscriptionsRefactored_CompatibilityWrapper(t *testing.T) {
	// This test ensures the compatibility wrapper works correctly

	// Create test request
	req := httptest.NewRequest("GET", "/subscriptions", nil)
	rec := httptest.NewRecorder()

	// Call the compatibility wrapper
	handleGetSubscriptionsRefactored(rec, req)

	// Verify it behaves like a normal handler (status should be set)
	if rec.Code == 0 {
		t.Error("Expected non-zero status code from compatibility wrapper")
	}
}

func TestYouTubeWebhookRefactored_NoDependencyOnGlobalState(t *testing.T) {
	// This test verifies that the refactored router doesn't depend on global testMode

	// Save original test mode
	originalTestMode := GetTestMode()
	originalStorageClient := GetStorageClient()
	defer func() {
		SetTestMode(originalTestMode)
		SetStorageClient(originalStorageClient)
	}()

	// Set global state to different values
	SetTestMode(false) // Disable test mode globally
	SetStorageClient(&CloudStorageClient{}) // Use production storage client

	// Create test dependencies (should override global state)
	testDeps := CreateTestDependencies()
	SetDependencies(testDeps)

	// Create test request
	req := httptest.NewRequest("GET", "/subscriptions", nil)
	rec := httptest.NewRecorder()

	// Call refactored router - should use injected dependencies, not global state
	YouTubeWebhookRefactored(rec, req)

	// Verify response (should succeed using test dependencies, not global state)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Refactored router should use injected dependencies, not global state", http.StatusOK, rec.Code)
	}
}
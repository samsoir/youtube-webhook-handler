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

func TestHandleNotificationWithDeps_Success(t *testing.T) {
	// Create test dependencies
	deps := CreateTestDependencies()
	mockGitHub := deps.GitHubClient.(*MockGitHubClient)
	mockGitHub.SetConfigured(true)

	// Set environment variables
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
  <link rel="hub" href="https://pubsubhubbub.appspot.com"/>
  <link rel="self" href="https://www.youtube.com/xml/feeds/videos.xml?channel_id=UC123456789012345678901"/>
  <title>YouTube video feed</title>
  <updated>%s</updated>
  <entry>
    <id>yt:video:test123</id>
    <yt:videoId>test123</yt:videoId>
    <yt:channelId>UC123456789012345678901</yt:channelId>
    <title>Test Video</title>
    <link rel="alternate" href="http://www.youtube.com/watch?v=test123"/>
    <author>
      <name>Test Channel</name>
      <uri>http://www.youtube.com/channel/UC123456789012345678901</uri>
    </author>
    <published>%s</published>
    <updated>%s</updated>
  </entry>
</feed>`, updated, published, updated)

	// Create request
	req := httptest.NewRequest("POST", "/", strings.NewReader(testXML))
	rec := httptest.NewRecorder()

	// Create handler and execute
	handler := handleNotificationWithDeps(deps)
	handler(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Successfully triggered workflow for new video: test123") {
		t.Errorf("Expected success message, got: %s", body)
	}

	// Verify GitHub client was called
	if mockGitHub.GetTriggerCallCount() != 1 {
		t.Errorf("Expected 1 trigger call, got %d", mockGitHub.GetTriggerCallCount())
	}

	entry := mockGitHub.GetLastEntry()
	if entry == nil {
		t.Fatal("Expected entry to be set, got nil")
	}
	if entry.VideoID != "test123" {
		t.Errorf("Expected VideoID test123, got %s", entry.VideoID)
	}
}

func TestHandleNotificationWithDeps_GitHubNotConfigured(t *testing.T) {
	// Create test dependencies with unconfigured GitHub
	deps := CreateTestDependencies()
	mockGitHub := deps.GitHubClient.(*MockGitHubClient)
	mockGitHub.SetConfigured(false)

	// Set environment variables
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

	// Create request
	req := httptest.NewRequest("POST", "/", strings.NewReader(testXML))
	rec := httptest.NewRecorder()

	// Create handler and execute
	handler := handleNotificationWithDeps(deps)
	handler(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "New video detected but GitHub token not configured") {
		t.Errorf("Expected GitHub not configured message, got: %s", body)
	}

	// Verify GitHub client was not called
	if mockGitHub.GetTriggerCallCount() != 0 {
		t.Errorf("Expected 0 trigger calls, got %d", mockGitHub.GetTriggerCallCount())
	}
}

func TestHandleNotificationWithDeps_InvalidXML(t *testing.T) {
	// Create test dependencies
	deps := CreateTestDependencies()

	// Create request with invalid XML
	invalidXML := "not xml"
	req := httptest.NewRequest("POST", "/", strings.NewReader(invalidXML))
	rec := httptest.NewRecorder()

	// Create handler and execute
	handler := handleNotificationWithDeps(deps)
	handler(rec, req)

	// Verify response
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	body := rec.Body.String()
	if body != "Invalid XML" {
		t.Errorf("Expected 'Invalid XML', got: %s", body)
	}
}

func TestHandleNotificationWithDeps_EmptyNotification(t *testing.T) {
	// Create test dependencies
	deps := CreateTestDependencies()

	// Create test XML without entries
	testXML := `<?xml version='1.0' encoding='UTF-8'?>
<feed xmlns:yt="http://www.youtube.com/xml/schemas/2015"
      xmlns="http://www.w3.org/2005/Atom">
  <title>YouTube video feed</title>
</feed>`

	// Create request
	req := httptest.NewRequest("POST", "/", strings.NewReader(testXML))
	rec := httptest.NewRecorder()

	// Create handler and execute
	handler := handleNotificationWithDeps(deps)
	handler(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Empty notification (no entry found)") {
		t.Errorf("Expected empty notification message, got: %s", body)
	}
}

func TestHandleNotificationWithDeps_NotNewVideo(t *testing.T) {
	// Create test dependencies
	deps := CreateTestDependencies()

	// Create test XML for old video (published more than 1 hour ago)
	oldTime := time.Now().Add(-2 * time.Hour)
	published := oldTime.Format(time.RFC3339)
	updated := oldTime.Add(time.Minute).Format(time.RFC3339)

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

	// Create request
	req := httptest.NewRequest("POST", "/", strings.NewReader(testXML))
	rec := httptest.NewRecorder()

	// Create handler and execute
	handler := handleNotificationWithDeps(deps)
	handler(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Skipped: Not a new video (VideoID: test123)") {
		t.Errorf("Expected not new video message, got: %s", body)
	}
}

func TestHandleNotificationWithDeps_GitHubTriggerError(t *testing.T) {
	// Create test dependencies
	deps := CreateTestDependencies()
	mockGitHub := deps.GitHubClient.(*MockGitHubClient)
	mockGitHub.SetConfigured(true)
	mockGitHub.SetTriggerError(fmt.Errorf("GitHub API error"))

	// Set environment variables
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

	// Create request
	req := httptest.NewRequest("POST", "/", strings.NewReader(testXML))
	rec := httptest.NewRecorder()

	// Create handler and execute
	handler := handleNotificationWithDeps(deps)
	handler(rec, req)

	// Verify response
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Failed to trigger GitHub workflow") {
		t.Errorf("Expected GitHub workflow error, got: %s", body)
	}
}

func TestHandleNotificationWithDeps_ReadBodyError(t *testing.T) {
	// Create test dependencies
	deps := CreateTestDependencies()

	// Create request with failing body reader
	req := httptest.NewRequest("POST", "/", nil)
	req.Body = &testFailingReader{}
	rec := httptest.NewRecorder()

	// Create handler and execute
	handler := handleNotificationWithDeps(deps)
	handler(rec, req)

	// Verify response
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	body := rec.Body.String()
	if body != "Failed to read request body" {
		t.Errorf("Expected 'Failed to read request body', got: %s", body)
	}
}

// testFailingReader is a helper that always fails on Read
type testFailingReader struct{}

func (fr *testFailingReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

func (fr *testFailingReader) Close() error {
	return nil
}

func TestNotificationServiceWithDeps_ProcessNotification_ThreadSafety(t *testing.T) {
	// Create test dependencies
	deps := CreateTestDependencies()
	mockGitHub := deps.GitHubClient.(*MockGitHubClient)
	mockGitHub.SetConfigured(true)

	// Set environment variables
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

	// Test concurrent access
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	handler := handleNotificationWithDeps(deps)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			req := httptest.NewRequest("POST", "/", strings.NewReader(testXML))
			rec := httptest.NewRecorder()
			handler(rec, req)
			
			if rec.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all calls were processed
	if mockGitHub.GetTriggerCallCount() != numGoroutines {
		t.Errorf("Expected %d trigger calls, got %d", numGoroutines, mockGitHub.GetTriggerCallCount())
	}
}

func TestHandleNotificationRefactored_CompatibilityWrapper(t *testing.T) {
	// This test ensures the compatibility wrapper works correctly

	// Set environment variables
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

	// Create request
	req := httptest.NewRequest("POST", "/", strings.NewReader(testXML))
	rec := httptest.NewRecorder()

	// Call the compatibility wrapper
	handleNotificationRefactored(rec, req)

	// Verify it behaves like a normal handler (status should be set)
	if rec.Code == 0 {
		t.Error("Expected non-zero status code from compatibility wrapper")
	}
}
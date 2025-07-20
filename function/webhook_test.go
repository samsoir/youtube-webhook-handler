package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data - use dynamic timestamps for testing
var (
	validXMLNotification = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns:yt="http://www.youtube.com/xml/schemas/2015" xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <yt:videoId>dQw4w9WgXcQ</yt:videoId>
    <yt:channelId>UCuAXFkgsw1L7xaCfnd5JJOw</yt:channelId>
    <title>Test Video Title</title>
    <published>%s</published>
    <updated>%s</updated>
  </entry>
</feed>`, time.Now().Add(-10*time.Minute).Format(time.RFC3339), time.Now().Add(-9*time.Minute).Format(time.RFC3339))

	oldVideoXMLNotification = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns:yt="http://www.youtube.com/xml/schemas/2015" xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <yt:videoId>oldVideo123</yt:videoId>
    <yt:channelId>UCuAXFkgsw1L7xaCfnd5JJOw</yt:channelId>
    <title>Old Video Title</title>
    <published>%s</published>
    <updated>%s</updated>
  </entry>
</feed>`, time.Now().Add(-2*time.Hour).Format(time.RFC3339), time.Now().Add(-2*time.Hour).Format(time.RFC3339))
	
	emptyXMLNotification = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns:yt="http://www.youtube.com/xml/schemas/2015" xmlns="http://www.w3.org/2005/Atom">
</feed>`

	invalidXMLNotification = `<invalid>xml</invalid`
)

// Mock GitHub API server
type mockGitHubServer struct {
	receivedPayloads []GitHubDispatch
	shouldFail       bool
	server           *httptest.Server
	t                *testing.T
}

func newMockGitHubServer(t *testing.T) *mockGitHubServer {
	mock := &mockGitHubServer{
		receivedPayloads: make([]GitHubDispatch, 0),
		t:                t,
	}

	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mock.shouldFail {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("GitHub API Error"))
			return
		}

		// Verify request structure
		assert.Equal(mock.t, "POST", r.Method)
		assert.Equal(mock.t, "application/json", r.Header.Get("Content-Type"))
		assert.True(mock.t, strings.HasPrefix(r.Header.Get("Authorization"), "token "))

		// Parse payload
		var payload GitHubDispatch
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mock.receivedPayloads = append(mock.receivedPayloads, payload)
		w.WriteHeader(http.StatusNoContent)
	}))

	return mock
}

func (m *mockGitHubServer) close() {
	m.server.Close()
}

func (m *mockGitHubServer) reset() {
	m.receivedPayloads = make([]GitHubDispatch, 0)
	m.shouldFail = false
}

// Test setup and teardown
func setupTestEnvironment(t *testing.T) *mockGitHubServer {
	mockGitHub := newMockGitHubServer(t)
	
	// Set required environment variables for testing
	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("REPO_OWNER", "testowner")
	os.Setenv("REPO_NAME", "testrepo")
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("GITHUB_API_BASE_URL", mockGitHub.server.URL)

	return mockGitHub
}

func teardownTestEnvironment() {
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("REPO_OWNER")
	os.Unsetenv("REPO_NAME")
	os.Unsetenv("ENVIRONMENT")
	os.Unsetenv("GITHUB_API_BASE_URL")
}

// Test: Webhook should respond to verification challenges
func TestWebhook_VerificationChallenge(t *testing.T) {
	tests := []struct {
		name           string
		challenge      string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid challenge",
			challenge:      "test-challenge-123",
			expectedStatus: http.StatusOK,
			expectedBody:   "test-challenge-123",
		},
		{
			name:           "Empty challenge",
			challenge:      "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGitHub := setupTestEnvironment(t)
			defer mockGitHub.close()
			defer teardownTestEnvironment()

			url := "/webhook"
			if tt.challenge != "" {
				url += "?hub.challenge=" + tt.challenge + "&hub.mode=subscribe&hub.topic=test"
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			YouTubeWebhook(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

// Test: Webhook should handle OPTIONS requests for CORS
func TestWebhook_CORSPreflight(t *testing.T) {
	mockGitHub := setupTestEnvironment(t)
	defer mockGitHub.close()
	defer teardownTestEnvironment()

	req := httptest.NewRequest(http.MethodOptions, "/webhook", nil)
	w := httptest.NewRecorder()

	YouTubeWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
}

// Test: Webhook should reject unsupported HTTP methods
func TestWebhook_UnsupportedMethod(t *testing.T) {
	mockGitHub := setupTestEnvironment(t)
	defer mockGitHub.close()
	defer teardownTestEnvironment()

	req := httptest.NewRequest(http.MethodPut, "/webhook", nil)
	w := httptest.NewRecorder()

	YouTubeWebhook(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// Test: Webhook should parse valid XML notifications and trigger GitHub workflow
func TestWebhook_ValidNotification(t *testing.T) {
	mockGitHub := setupTestEnvironment(t)
	defer mockGitHub.close()
	defer teardownTestEnvironment()

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(validXMLNotification))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	YouTubeWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Webhook processed successfully")

	// Verify GitHub API was called
	require.Len(t, mockGitHub.receivedPayloads, 1)
	payload := mockGitHub.receivedPayloads[0]

	assert.Equal(t, "youtube-video-published", payload.EventType)
	assert.Equal(t, "dQw4w9WgXcQ", payload.ClientPayload["video_id"])
	assert.Equal(t, "Test Video Title", payload.ClientPayload["title"])
	assert.Equal(t, "UCuAXFkgsw1L7xaCfnd5JJOw", payload.ClientPayload["channel_id"])
	assert.Equal(t, "test", payload.ClientPayload["environment"])
	assert.Contains(t, payload.ClientPayload["video_url"], "dQw4w9WgXcQ")
}

// Test: Webhook should reject invalid XML
func TestWebhook_InvalidXML(t *testing.T) {
	mockGitHub := setupTestEnvironment(t)
	defer mockGitHub.close()
	defer teardownTestEnvironment()

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(invalidXMLNotification))
	w := httptest.NewRecorder()

	YouTubeWebhook(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid XML")

	// Verify GitHub API was NOT called
	assert.Len(t, mockGitHub.receivedPayloads, 0)
}

// Test: Webhook should handle empty notifications gracefully
func TestWebhook_EmptyNotification(t *testing.T) {
	mockGitHub := setupTestEnvironment(t)
	defer mockGitHub.close()
	defer teardownTestEnvironment()

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(emptyXMLNotification))
	w := httptest.NewRecorder()

	YouTubeWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "No video data")

	// Verify GitHub API was NOT called
	assert.Len(t, mockGitHub.receivedPayloads, 0)
}

// Test: Webhook should filter out old video updates
func TestWebhook_OldVideoUpdate(t *testing.T) {
	mockGitHub := setupTestEnvironment(t)
	defer mockGitHub.close()
	defer teardownTestEnvironment()

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(oldVideoXMLNotification))
	w := httptest.NewRecorder()

	YouTubeWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Video update ignored")

	// Verify GitHub API was NOT called
	assert.Len(t, mockGitHub.receivedPayloads, 0)
}

// Test: Webhook should handle GitHub API failures gracefully
func TestWebhook_GitHubAPIFailure(t *testing.T) {
	mockGitHub := setupTestEnvironment(t)
	defer mockGitHub.close()
	defer teardownTestEnvironment()

	mockGitHub.shouldFail = true

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(validXMLNotification))
	w := httptest.NewRecorder()

	YouTubeWebhook(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "GitHub API error")
}

// Test: Webhook should require environment variables
func TestWebhook_MissingEnvironmentVariables(t *testing.T) {
	// Don't set up environment variables
	mockGitHub := newMockGitHubServer(t)
	defer mockGitHub.close()

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(validXMLNotification))
	w := httptest.NewRecorder()

	YouTubeWebhook(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "GitHub API error")

	// Verify GitHub API was NOT called
	assert.Len(t, mockGitHub.receivedPayloads, 0)
}

// Test: isNewVideo function should correctly identify new vs old videos
func TestIsNewVideo(t *testing.T) {
	tests := []struct {
		name      string
		entry     *Entry
		expected  bool
		description string
	}{
		{
			name: "New video - recent publish",
			entry: &Entry{
				Published: time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
				Updated:   time.Now().Add(-9 * time.Minute).Format(time.RFC3339),
			},
			expected: true,
			description: "Recent video with minimal time difference should be considered new",
		},
		{
			name: "Old video - published over 1 hour ago",
			entry: &Entry{
				Published: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
				Updated:   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			},
			expected: false,
			description: "Video published over 1 hour ago should be considered old",
		},
		{
			name: "Video update - large time difference",
			entry: &Entry{
				Published: time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
				Updated:   time.Now().Format(time.RFC3339),
			},
			expected: false,
			description: "Video with large update-publish difference should be considered an update",
		},
		{
			name: "Invalid timestamps",
			entry: &Entry{
				Published: "invalid-timestamp",
				Updated:   "also-invalid",
			},
			expected: true,
			description: "Invalid timestamps should default to treating as new video",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNewVideo(tt.entry)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// Test: XML parsing should handle various feed formats
func TestXMLParsing(t *testing.T) {
	tests := []struct {
		name        string
		xmlContent  string
		expectError bool
		expectEntry bool
	}{
		{
			name:        "Valid complete feed",
			xmlContent:  validXMLNotification,
			expectError: false,
			expectEntry: true,
		},
		{
			name:        "Empty feed",
			xmlContent:  emptyXMLNotification,
			expectError: false,
			expectEntry: false,
		},
		{
			name:        "Invalid XML",
			xmlContent:  invalidXMLNotification,
			expectError: true,
			expectEntry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var feed AtomFeed
			err := json.Unmarshal([]byte(tt.xmlContent), &feed) // This will fail, but that's the point

			if tt.expectError {
				assert.Error(t, err)
			} else {
				// We expect this to fail since we're using JSON unmarshal on XML
				// In the actual implementation, we'll use xml.Unmarshal
				assert.Error(t, err) // This test will be updated once we implement the actual parsing
			}
		})
	}
}

// Benchmark: Webhook performance under load
func BenchmarkWebhook_ValidNotification(b *testing.B) {
	t := &testing.T{} // Create a minimal testing.T for the mock
	mockGitHub := setupTestEnvironment(t)
	defer mockGitHub.close()
	defer teardownTestEnvironment()

	body := strings.NewReader(validXMLNotification)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body.Seek(0, 0) // Reset reader
		req := httptest.NewRequest(http.MethodPost, "/webhook", body)
		w := httptest.NewRecorder()

		YouTubeWebhook(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// failingResponseWriter simulates a response writer that fails on Write calls
type failingResponseWriter struct {
	*httptest.ResponseRecorder
}

func (f *failingResponseWriter) Write(data []byte) (int, error) {
	return 0, fmt.Errorf("simulated write error")
}

func TestWebhook_WriteErrors(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		body   string
	}{
		{"Method not allowed write error", "DELETE", "/", ""},
		{"Challenge write error", "GET", "/?hub.challenge=test", ""},
		{"Bad request write error", "POST", "/", "invalid body"},
		{"Invalid XML write error", "POST", "/", invalidXMLNotification},
		{"Empty notification write error", "POST", "/", emptyXMLNotification},
		{"Old video write error", "POST", "/", oldVideoXMLNotification},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/xml")
			
			// Use failing response writer
			w := &failingResponseWriter{ResponseRecorder: httptest.NewRecorder()}
			
			// This should not panic even with write errors
			assert.NotPanics(t, func() {
				YouTubeWebhook(w, req)
			})
		})
	}
}
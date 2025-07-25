package webhook

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestHandleNotification_EdgeCases tests remaining edge cases in notification handling
func TestHandleNotification_EdgeCases(t *testing.T) {
	t.Run("invalid_xml_structure", func(t *testing.T) {
		// Test with XML that has correct structure but invalid content
		xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
			<entry>
				<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015"></yt:videoId>
				<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
				<title>Test Video</title>
				<published>invalid-date</published>
				<updated>invalid-date</updated>
			</entry>
		</feed>`
		
		req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
		w := httptest.NewRecorder()
		
		handleNotification(w, req)
		
		// Should process even with invalid dates (isNewVideo handles parsing errors)
		assert.Equal(t, http.StatusInternalServerError, w.Code) // Fails at GitHub API call due to missing env vars
	})
	
	t.Run("github_workflow_trigger_failure", func(t *testing.T) {
		// Test with valid XML but missing GitHub environment variables
		xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
			<entry>
				<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test_video_id</yt:videoId>
				<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
				<title>Test Video</title>
				<published>` + time.Now().Format(time.RFC3339) + `</published>
				<updated>` + time.Now().Format(time.RFC3339) + `</updated>
			</entry>
		</feed>`
		
		// Clear GitHub environment variables
		originalToken := os.Getenv("GITHUB_TOKEN")
		originalOwner := os.Getenv("REPO_OWNER")
		originalName := os.Getenv("REPO_NAME")
		
		os.Unsetenv("GITHUB_TOKEN")
		os.Unsetenv("REPO_OWNER") 
		os.Unsetenv("REPO_NAME")
		
		defer func() {
			setEnvOrUnset("GITHUB_TOKEN", originalToken)
			setEnvOrUnset("REPO_OWNER", originalOwner)
			setEnvOrUnset("REPO_NAME", originalName)
		}()
		
		req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
		w := httptest.NewRecorder()
		
		handleNotification(w, req)
		
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "GitHub API error", w.Body.String())
	})
}

// TestHandleNotification_BodyReadError tests body reading edge cases  
func TestHandleNotification_BodyReadError(t *testing.T) {
	// Create a request with a body that will cause read errors
	// Use a failing reader to simulate io.ReadAll errors
	req := httptest.NewRequest("POST", "/", &failingReader{})
	w := httptest.NewRecorder()
	
	handleNotification(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "Failed to read request body", w.Body.String())
}

// TestHandleNotification_MoreErrorPaths tests additional notification error paths
func TestHandleNotification_MoreErrorPaths(t *testing.T) {
	t.Run("xml_unmarshal_error", func(t *testing.T) {
		// Test with completely invalid XML
		invalidXML := "not xml at all"
		req := httptest.NewRequest("POST", "/", strings.NewReader(invalidXML))
		w := httptest.NewRecorder()
		
		handleNotification(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "Invalid XML", w.Body.String())
	})
	
	t.Run("successful_github_workflow_trigger", func(t *testing.T) {
		// Create server that returns success to test successful path
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		
		// Test with valid XML for new video
		xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
			<entry>
				<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test_video_id</yt:videoId>
				<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
				<title>Test Video</title>
				<published>` + time.Now().Format(time.RFC3339) + `</published>
				<updated>` + time.Now().Format(time.RFC3339) + `</updated>
			</entry>
		</feed>`
		
		originalToken := os.Getenv("GITHUB_TOKEN")
		originalOwner := os.Getenv("REPO_OWNER")
		originalName := os.Getenv("REPO_NAME")
		originalBaseURL := os.Getenv("GITHUB_API_BASE_URL")
		
		os.Setenv("GITHUB_TOKEN", "test-token")
		os.Setenv("REPO_OWNER", "test-owner")
		os.Setenv("REPO_NAME", "test-repo")
		os.Setenv("GITHUB_API_BASE_URL", server.URL)
		
		defer func() {
			setEnvOrUnset("GITHUB_TOKEN", originalToken)
			setEnvOrUnset("REPO_OWNER", originalOwner)
			setEnvOrUnset("REPO_NAME", originalName)
			setEnvOrUnset("GITHUB_API_BASE_URL", originalBaseURL)
		}()
		
		req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
		w := httptest.NewRecorder()
		
		handleNotification(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "Webhook processed successfully", w.Body.String())
	})
}

// TestIsNewVideo_TimestampEdgeCases tests edge cases in timestamp parsing
func TestIsNewVideo_TimestampEdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		published   string
		updated     string
		expected    bool
		description string
	}{
		{
			name:        "malformed_published_date",
			published:   "not-a-date",
			updated:     time.Now().Format(time.RFC3339),
			expected:    true,
			description: "Should return true when published date cannot be parsed",
		},
		{
			name:        "malformed_updated_date", 
			published:   time.Now().Format(time.RFC3339),
			updated:     "not-a-date",
			expected:    true,
			description: "Should return true when updated date cannot be parsed",
		},
		{
			name:        "both_dates_malformed",
			published:   "not-a-date-1",
			updated:     "not-a-date-2", 
			expected:    true,
			description: "Should return true when both dates cannot be parsed",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := &Entry{
				Published: tc.published,
				Updated:   tc.updated,
			}
			
			result := isNewVideo(entry)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}

// failingReader simulates a reader that always fails
type failingReader struct{}

func (f *failingReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}


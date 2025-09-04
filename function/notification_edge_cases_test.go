package webhook

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNotification_EdgeCases tests various edge cases for the notification handler using dependency injection
func TestNotification_EdgeCases(t *testing.T) {
	t.Run("InvalidXMLStructure", func(t *testing.T) {
		deps := CreateTestDependencies()

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

		handler := handleNotification(deps)
		handler(w, req)

		// Should handle invalid dates gracefully and skip processing
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Skipped: Not a new video")
	})

	t.Run("GitHubNotConfigured", func(t *testing.T) {
		deps := CreateTestDependencies()
		
		// Configure GitHub client to be not configured (empty token)
		mockGitHub := deps.GitHubClient.(*MockGitHubClient)
		mockGitHub.SetConfigured(false)

		// Test with valid XML for a new video
		now := time.Now()
		published := now.Add(-10 * time.Minute).Format(time.RFC3339)
		updated := now.Add(-9 * time.Minute).Format(time.RFC3339)

		xmlPayload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
			<entry>
				<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test_video_id</yt:videoId>
				<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
				<title>Test Video</title>
				<published>%s</published>
				<updated>%s</updated>
			</entry>
		</feed>`, published, updated)

		req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
		w := httptest.NewRecorder()

		handler := handleNotification(deps)
		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "GitHub token not configured")
	})

	t.Run("GitHubTriggerError", func(t *testing.T) {
		deps := CreateTestDependencies()
		
		// Configure GitHub client to fail
		mockGitHub := deps.GitHubClient.(*MockGitHubClient)
		mockGitHub.SetTriggerError(fmt.Errorf("GitHub API unavailable"))

		// Test with valid XML for a new video
		now := time.Now()
		published := now.Add(-10 * time.Minute).Format(time.RFC3339)
		updated := now.Add(-9 * time.Minute).Format(time.RFC3339)

		xmlPayload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
			<entry>
				<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test_video_id</yt:videoId>
				<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
				<title>Test Video</title>
				<published>%s</published>
				<updated>%s</updated>
			</entry>
		</feed>`, published, updated)

		req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
		w := httptest.NewRecorder()

		handler := handleNotification(deps)
		handler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to trigger GitHub workflow")
	})

	t.Run("InvalidXML", func(t *testing.T) {
		deps := CreateTestDependencies()

		// Test with completely invalid XML
		invalidXML := "not xml at all"
		req := httptest.NewRequest("POST", "/", strings.NewReader(invalidXML))
		w := httptest.NewRecorder()

		handler := handleNotification(deps)
		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid XML")
	})

	t.Run("EmptyNotification", func(t *testing.T) {
		deps := CreateTestDependencies()

		// Test with empty XML feed (no entry)
		emptyXML := `<?xml version="1.0" encoding="UTF-8"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
		</feed>`

		req := httptest.NewRequest("POST", "/", strings.NewReader(emptyXML))
		w := httptest.NewRecorder()

		handler := handleNotification(deps)
		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Empty notification (no entry found)")
	})

	t.Run("OldVideo", func(t *testing.T) {
		deps := CreateTestDependencies()

		// Test with old video (published over 1 hour ago)
		now := time.Now()
		published := now.Add(-2 * time.Hour).Format(time.RFC3339)
		updated := now.Add(-2 * time.Hour).Format(time.RFC3339)

		xmlPayload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
			<entry>
				<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">old_video_id</yt:videoId>
				<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
				<title>Old Video</title>
				<published>%s</published>
				<updated>%s</updated>
			</entry>
		</feed>`, published, updated)

		req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
		w := httptest.NewRecorder()

		handler := handleNotification(deps)
		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Skipped: Not a new video")
	})

	t.Run("SuccessfulNotification", func(t *testing.T) {
		deps := CreateTestDependencies()

		// Test with valid XML for a new video
		now := time.Now()
		published := now.Add(-10 * time.Minute).Format(time.RFC3339)
		updated := now.Add(-9 * time.Minute).Format(time.RFC3339)

		xmlPayload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
			<entry>
				<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">new_video_id</yt:videoId>
				<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
				<title>New Video Title</title>
				<published>%s</published>
				<updated>%s</updated>
			</entry>
		</feed>`, published, updated)

		req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
		w := httptest.NewRecorder()

		handler := handleNotification(deps)
		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Successfully triggered workflow for new video")

		// Verify GitHub workflow was triggered
		mockGitHub := deps.GitHubClient.(*MockGitHubClient)
		assert.Equal(t, 1, mockGitHub.GetTriggerCallCount())
		assert.Equal(t, "new_video_id", mockGitHub.GetLastEntry().VideoID)
		assert.Equal(t, "New Video Title", mockGitHub.GetLastEntry().Title)
	})

	t.Run("ReadBodyError", func(t *testing.T) {
		deps := CreateTestDependencies()

		// Create a request with a body that will cause read errors
		req := httptest.NewRequest("POST", "/", &failingReader{})
		w := httptest.NewRecorder()

		handler := handleNotification(deps)
		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to read request body")
	})

	t.Run("EmptyRequestBody", func(t *testing.T) {
		deps := CreateTestDependencies()

		req := httptest.NewRequest("POST", "/", strings.NewReader(""))
		w := httptest.NewRecorder()

		handler := handleNotification(deps)
		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid XML")
	})
}

// TestNotification_XMLParsingEdgeCases tests edge cases in XML parsing
func TestNotification_XMLParsingEdgeCases(t *testing.T) {
	t.Run("MalformedXMLWithValidStructure", func(t *testing.T) {
		deps := CreateTestDependencies()

		// XML with valid structure but malformed content
		xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
			<entry>
				<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test123</yt:videoId>
				<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015"></yt:channelId>
				<title></title>
				<published></published>
				<updated></updated>
			</entry>
		</feed>`

		req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
		w := httptest.NewRecorder()

		handler := handleNotification(deps)
		handler(w, req)

		// Should handle gracefully and skip due to empty fields
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Skipped: Not a new video")
	})

	t.Run("XMLWithUnsupportedEncoding", func(t *testing.T) {
		deps := CreateTestDependencies()

		// XML with different encoding - Go's XML parser doesn't handle ISO-8859-1 properly
		xmlPayload := `<?xml version="1.0" encoding="ISO-8859-1"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
			<entry>
				<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test123</yt:videoId>
				<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
				<title>Test Video</title>
				<published>` + time.Now().Add(-10*time.Minute).Format(time.RFC3339) + `</published>
				<updated>` + time.Now().Add(-9*time.Minute).Format(time.RFC3339) + `</updated>
			</entry>
		</feed>`

		req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
		w := httptest.NewRecorder()

		handler := handleNotification(deps)
		handler(w, req)

		// Go's XML parser may reject unsupported encoding
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid XML")
	})

	t.Run("XMLWithMissingNamespaces", func(t *testing.T) {
		deps := CreateTestDependencies()

		// XML without yt namespace - videoId and channelId won't be parsed correctly
		xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
			<entry>
				<videoId>test123</videoId>
				<channelId>UCXuqSBlHAE6Xw-yeJA0Tunw</channelId>
				<title>Test Video</title>
				<published>` + time.Now().Add(-10*time.Minute).Format(time.RFC3339) + `</published>
				<updated>` + time.Now().Add(-9*time.Minute).Format(time.RFC3339) + `</updated>
			</entry>
		</feed>`

		req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
		w := httptest.NewRecorder()

		handler := handleNotification(deps)
		handler(w, req)

		// Without proper yt namespace, entry exists but video processing succeeds
		// because the timestamp logic still works even with empty video/channel IDs
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Successfully triggered workflow for new video")
	})
}

// TestNotification_TimestampEdgeCases tests edge cases in timestamp handling
func TestNotification_TimestampEdgeCases(t *testing.T) {
	testCases := []struct {
		name      string
		published string
		updated   string
		expected  string
	}{
		{
			name:      "malformed_published_date",
			published: "not-a-date",
			updated:   time.Now().Format(time.RFC3339),
			expected:  "Skipped: Not a new video", // Skip processing on parse error for safety
		},
		{
			name:      "malformed_updated_date",
			published: time.Now().Format(time.RFC3339),
			updated:   "not-a-date",
			expected:  "Skipped: Not a new video", // Skip processing on parse error for safety
		},
		{
			name:      "both_dates_malformed",
			published: "not-a-date-1",
			updated:   "not-a-date-2",
			expected:  "Skipped: Not a new video", // Skip processing on parse error for safety
		},
		{
			name:      "future_dates",
			published: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			updated:   time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			expected:  "Skipped: Not a new video", // Future dates with large gaps should be skipped
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			deps := CreateTestDependencies()

			xmlPayload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
			<feed xmlns="http://www.w3.org/2005/Atom">
				<entry>
					<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test_%s</yt:videoId>
					<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
					<title>Test Video %s</title>
					<published>%s</published>
					<updated>%s</updated>
				</entry>
			</feed>`, tc.name, tc.name, tc.published, tc.updated)

			req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
			w := httptest.NewRecorder()

			handler := handleNotification(deps)
			handler(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), tc.expected)
		})
	}
}

// TestNotification_ConcurrentRequests tests thread safety
func TestNotification_ConcurrentRequests(t *testing.T) {
	deps := CreateTestDependencies()
	const numRequests = 10

	done := make(chan bool, numRequests)
	results := make([]int, numRequests)

	// Run multiple concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(index int) {
			xmlPayload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
			<feed xmlns="http://www.w3.org/2005/Atom">
				<entry>
					<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test_%d</yt:videoId>
					<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
					<title>Test Video %d</title>
					<published>%s</published>
					<updated>%s</updated>
				</entry>
			</feed>`, index, index,
				time.Now().Add(-10*time.Minute).Format(time.RFC3339),
				time.Now().Add(-9*time.Minute).Format(time.RFC3339))

			req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
			w := httptest.NewRecorder()

			handler := handleNotification(deps)
			handler(w, req)

			results[index] = w.Code
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}

	// All should succeed
	for i, code := range results {
		assert.Equal(t, http.StatusOK, code, "Request %d should succeed", i)
	}

	// Verify all requests were processed
	mockGitHub := deps.GitHubClient.(*MockGitHubClient)
	assert.Equal(t, numRequests, mockGitHub.GetTriggerCallCount())
}

// failingReader simulates a reader that always fails
type failingReader struct{}

func (f *failingReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}

// TestIsNewVideo_EdgeCases tests the isNewVideo function with edge cases
func TestIsNewVideo_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		entry       *Entry
		expected    bool
		description string
	}{
		{
			name: "new_video_recent_publish",
			entry: &Entry{
				Published: time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
				Updated:   time.Now().Add(-9 * time.Minute).Format(time.RFC3339),
			},
			expected:    true,
			description: "Recent video should be considered new",
		},
		{
			name: "old_video_published_hours_ago",
			entry: &Entry{
				Published: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
				Updated:   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			},
			expected:    false,
			description: "Video published hours ago should be old",
		},
		{
			name: "video_update_large_time_difference",
			entry: &Entry{
				Published: time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
				Updated:   time.Now().Format(time.RFC3339),
			},
			expected:    false,
			description: "Video with large update-publish difference should be an update",
		},
		{
			name: "invalid_timestamps_published",
			entry: &Entry{
				Published: "invalid-timestamp",
				Updated:   time.Now().Format(time.RFC3339),
			},
			expected:    false,
			description: "Invalid published timestamp should be skipped for safety",
		},
		{
			name: "invalid_timestamps_updated",
			entry: &Entry{
				Published: time.Now().Format(time.RFC3339),
				Updated:   "invalid-timestamp",
			},
			expected:    false,
			description: "Invalid updated timestamp should be skipped for safety",
		},
		{
			name: "both_timestamps_invalid",
			entry: &Entry{
				Published: "invalid-1",
				Updated:   "invalid-2",
			},
			expected:    false,
			description: "Both invalid timestamps should be skipped for safety",
		},
		{
			name: "empty_timestamps",
			entry: &Entry{
				Published: "",
				Updated:   "",
			},
			expected:    false,
			description: "Empty timestamps should be skipped for safety",
		},
		{
			name: "future_timestamps",
			entry: &Entry{
				Published: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
				Updated:   time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			},
			expected:    false,
			description: "Future timestamps with large update-publish gap should be skipped as suspicious",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isNewVideo(tc.entry)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}

// TestAtomFeedParsing_EdgeCases tests XML parsing with various edge cases
func TestAtomFeedParsing_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		xmlContent  string
		expectError bool
		expectEntry bool
		description string
	}{
		{
			name:        "valid_complete_feed",
			xmlContent:  `<?xml version="1.0" encoding="UTF-8"?><feed xmlns="http://www.w3.org/2005/Atom"><entry><yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test123</yt:videoId></entry></feed>`,
			expectError: false,
			expectEntry: true,
			description: "Valid feed should parse correctly",
		},
		{
			name:        "empty_feed",
			xmlContent:  `<?xml version="1.0" encoding="UTF-8"?><feed xmlns="http://www.w3.org/2005/Atom"></feed>`,
			expectError: false,
			expectEntry: false,
			description: "Empty feed should parse without error",
		},
		{
			name:        "invalid_xml",
			xmlContent:  `<invalid>xml</invalid`,
			expectError: true,
			expectEntry: false,
			description: "Invalid XML should cause parse error",
		},
		{
			name:        "feed_with_attributes",
			xmlContent:  `<?xml version="1.0" encoding="UTF-8"?><feed xmlns="http://www.w3.org/2005/Atom" xml:lang="en"><entry><yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test123</yt:videoId></entry></feed>`,
			expectError: false,
			expectEntry: true,
			description: "Feed with attributes should parse correctly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var feed AtomFeed
			err := xml.Unmarshal([]byte(tc.xmlContent), &feed)

			if tc.expectError {
				assert.Error(t, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
				if tc.expectEntry {
					assert.NotNil(t, feed.Entry, tc.description)
				} else {
					assert.Nil(t, feed.Entry, tc.description)
				}
			}
		})
	}
}
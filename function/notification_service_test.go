package webhook

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewNotificationService(t *testing.T) {
	// Set environment
	originalOwner := os.Getenv("REPO_OWNER")
	originalName := os.Getenv("REPO_NAME")
	
	os.Setenv("REPO_OWNER", "test-owner")
	os.Setenv("REPO_NAME", "test-repo")
	
	defer func() {
		if originalOwner == "" {
			os.Unsetenv("REPO_OWNER")
		} else {
			os.Setenv("REPO_OWNER", originalOwner)
		}
		if originalName == "" {
			os.Unsetenv("REPO_NAME")
		} else {
			os.Setenv("REPO_NAME", originalName)
		}
	}()

	service := NewNotificationService()
	
	assert.NotNil(t, service)
	assert.NotNil(t, service.VideoProcessor)
	assert.NotNil(t, service.GitHubClient)
	assert.Equal(t, "test-owner", service.RepoOwner)
	assert.Equal(t, "test-repo", service.RepoName)
}

func TestNotificationService_ProcessNotification_InvalidXML(t *testing.T) {
	service := NewNotificationService()
	
	// Create request with invalid XML
	req := httptest.NewRequest("POST", "/", strings.NewReader("invalid xml"))
	
	result, err := service.ProcessNotification(req)
	
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "error", result.Status)
	assert.Equal(t, "Invalid XML", result.Message)
}

func TestNotificationService_ProcessNotification_ReadError(t *testing.T) {
	service := NewNotificationService()
	
	// Use failing reader
	req := httptest.NewRequest("POST", "/", &failingReader{})
	
	result, err := service.ProcessNotification(req)
	
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "error", result.Status)
	assert.Equal(t, "Failed to read request body", result.Message)
}

func TestNotificationService_ProcessNotification_EmptyNotification(t *testing.T) {
	service := NewNotificationService()
	
	// Create XML with no entry
	xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
	<feed xmlns="http://www.w3.org/2005/Atom">
	</feed>`
	
	req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
	
	result, err := service.ProcessNotification(req)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "No video data", result.Message)
}

func TestNotificationService_ProcessNotification_InvalidEntry(t *testing.T) {
	service := NewNotificationService()
	
	// Create XML with invalid entry (missing video ID)
	xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
	<feed xmlns="http://www.w3.org/2005/Atom">
		<entry>
			<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015"></yt:videoId>
			<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
			<title>Test Video</title>
		</entry>
	</feed>`
	
	req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
	
	result, err := service.ProcessNotification(req)
	
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Message, "Invalid entry")
}

func TestNotificationService_ProcessNotification_OldVideo(t *testing.T) {
	service := NewNotificationService()
	
	// Create XML with old video (published 2 hours ago)
	oldTime := time.Now().Add(-2 * time.Hour)
	xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
	<feed xmlns="http://www.w3.org/2005/Atom">
		<entry>
			<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test_video_id</yt:videoId>
			<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
			<title>Test Video</title>
			<published>` + oldTime.Format(time.RFC3339) + `</published>
			<updated>` + oldTime.Add(time.Minute).Format(time.RFC3339) + `</updated>
		</entry>
	</feed>`
	
	req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
	
	result, err := service.ProcessNotification(req)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "Video update ignored", result.Message)
}

func TestNotificationService_ProcessNotification_MissingGitHubConfig(t *testing.T) {
	// Clear GitHub environment variables
	originalToken := os.Getenv("GITHUB_TOKEN")
	originalOwner := os.Getenv("REPO_OWNER")
	originalName := os.Getenv("REPO_NAME")
	
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("REPO_OWNER")
	os.Unsetenv("REPO_NAME")
	
	defer func() {
		if originalToken != "" {
			os.Setenv("GITHUB_TOKEN", originalToken)
		}
		if originalOwner != "" {
			os.Setenv("REPO_OWNER", originalOwner)
		}
		if originalName != "" {
			os.Setenv("REPO_NAME", originalName)
		}
	}()

	service := NewNotificationService()
	
	// Create XML with new video
	newTime := time.Now().Add(-30 * time.Minute)
	xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
	<feed xmlns="http://www.w3.org/2005/Atom">
		<entry>
			<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test_video_id</yt:videoId>
			<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
			<title>Test Video</title>
			<published>` + newTime.Format(time.RFC3339) + `</published>
			<updated>` + newTime.Add(time.Minute).Format(time.RFC3339) + `</updated>
		</entry>
	</feed>`
	
	req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
	
	result, err := service.ProcessNotification(req)
	
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "error", result.Status)
	assert.Equal(t, "GitHub API error", result.Message)
}

func TestNotificationService_ProcessNotification_GitHubError(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Set up environment
	originalToken := os.Getenv("GITHUB_TOKEN")
	originalOwner := os.Getenv("REPO_OWNER")
	originalName := os.Getenv("REPO_NAME")
	originalBaseURL := os.Getenv("GITHUB_API_BASE_URL")
	
	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("REPO_OWNER", "test-owner")
	os.Setenv("REPO_NAME", "test-repo")
	os.Setenv("GITHUB_API_BASE_URL", server.URL)
	
	defer func() {
		if originalToken == "" {
			os.Unsetenv("GITHUB_TOKEN")
		} else {
			os.Setenv("GITHUB_TOKEN", originalToken)
		}
		if originalOwner == "" {
			os.Unsetenv("REPO_OWNER")
		} else {
			os.Setenv("REPO_OWNER", originalOwner)
		}
		if originalName == "" {
			os.Unsetenv("REPO_NAME")
		} else {
			os.Setenv("REPO_NAME", originalName)
		}
		if originalBaseURL == "" {
			os.Unsetenv("GITHUB_API_BASE_URL")
		} else {
			os.Setenv("GITHUB_API_BASE_URL", originalBaseURL)
		}
	}()

	service := NewNotificationService()
	
	// Create XML with new video
	newTime := time.Now().Add(-30 * time.Minute)
	xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
	<feed xmlns="http://www.w3.org/2005/Atom">
		<entry>
			<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test_video_id</yt:videoId>
			<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
			<title>Test Video</title>
			<published>` + newTime.Format(time.RFC3339) + `</published>
			<updated>` + newTime.Add(time.Minute).Format(time.RFC3339) + `</updated>
		</entry>
	</feed>`
	
	req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
	
	result, err := service.ProcessNotification(req)
	
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "error", result.Status)
	assert.Equal(t, "GitHub API error", result.Message)
}

func TestNotificationService_ProcessNotification_Success(t *testing.T) {
	// Create mock server that returns success
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set up environment
	originalToken := os.Getenv("GITHUB_TOKEN")
	originalOwner := os.Getenv("REPO_OWNER")
	originalName := os.Getenv("REPO_NAME")
	originalBaseURL := os.Getenv("GITHUB_API_BASE_URL")
	
	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("REPO_OWNER", "test-owner")
	os.Setenv("REPO_NAME", "test-repo")
	os.Setenv("GITHUB_API_BASE_URL", server.URL)
	
	defer func() {
		if originalToken == "" {
			os.Unsetenv("GITHUB_TOKEN")
		} else {
			os.Setenv("GITHUB_TOKEN", originalToken)
		}
		if originalOwner == "" {
			os.Unsetenv("REPO_OWNER")
		} else {
			os.Setenv("REPO_OWNER", originalOwner)
		}
		if originalName == "" {
			os.Unsetenv("REPO_NAME")
		} else {
			os.Setenv("REPO_NAME", originalName)
		}
		if originalBaseURL == "" {
			os.Unsetenv("GITHUB_API_BASE_URL")
		} else {
			os.Setenv("GITHUB_API_BASE_URL", originalBaseURL)
		}
	}()

	service := NewNotificationService()
	
	// Create XML with new video
	newTime := time.Now().Add(-30 * time.Minute)
	xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
	<feed xmlns="http://www.w3.org/2005/Atom">
		<entry>
			<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test_video_id</yt:videoId>
			<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
			<title>Test Video</title>
			<published>` + newTime.Format(time.RFC3339) + `</published>
			<updated>` + newTime.Add(time.Minute).Format(time.RFC3339) + `</updated>
		</entry>
	</feed>`
	
	req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
	
	result, err := service.ProcessNotification(req)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "Webhook processed successfully", result.Message)
}

func TestNotificationService_parseNotification(t *testing.T) {
	service := NewNotificationService()

	t.Run("valid_xml", func(t *testing.T) {
		xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
			<entry>
				<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test_video_id</yt:videoId>
				<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
				<title>Test Video</title>
			</entry>
		</feed>`
		
		req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
		
		entry, err := service.parseNotification(req)
		
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, "test_video_id", entry.VideoID)
		assert.Equal(t, "UCXuqSBlHAE6Xw-yeJA0Tunw", entry.ChannelID)
		assert.Equal(t, "Test Video", entry.Title)
	})

	t.Run("empty_feed", func(t *testing.T) {
		xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
		</feed>`
		
		req := httptest.NewRequest("POST", "/", strings.NewReader(xmlPayload))
		
		entry, err := service.parseNotification(req)
		
		assert.NoError(t, err)
		assert.Nil(t, entry)
	})

	t.Run("invalid_xml", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader("invalid xml"))
		
		entry, err := service.parseNotification(req)
		
		assert.Error(t, err)
		assert.Nil(t, entry)
		assert.Contains(t, err.Error(), "invalid XML")
	})

	t.Run("read_error", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", &failingReader{})
		
		entry, err := service.parseNotification(req)
		
		assert.Error(t, err)
		assert.Nil(t, entry)
		assert.Contains(t, err.Error(), "failed to read request body")
	})
}

func TestNotificationService_ErrorMapping(t *testing.T) {
	service := NewNotificationService()

	// Test that error message mapping works correctly
	testCases := []struct {
		name         string
		errorMessage string
		expected     string
	}{
		{
			name:         "read_body_error",
			errorMessage: "failed to read request body",
			expected:     "Failed to read request body",
		},
		{
			name:         "xml_error",
			errorMessage: "invalid XML",
			expected:     "Invalid XML",
		},
		{
			name:         "other_error",
			errorMessage: "some other error",
			expected:     "some other error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This tests the error mapping logic in ProcessNotification
			// We can't directly test the private mapping, but we can verify
			// the behavior through integration
			if tc.errorMessage == "failed to read request body" {
				req := httptest.NewRequest("POST", "/", &failingReader{})
				result, err := service.ProcessNotification(req)
				assert.Error(t, err)
				assert.Equal(t, tc.expected, result.Message)
			} else if tc.errorMessage == "invalid XML" {
				req := httptest.NewRequest("POST", "/", strings.NewReader("invalid"))
				result, err := service.ProcessNotification(req)
				assert.Error(t, err)
				assert.Equal(t, tc.expected, result.Message)
			}
		})
	}
}


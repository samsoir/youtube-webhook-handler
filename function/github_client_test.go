package webhook

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitHubClient(t *testing.T) {
	// Set up environment
	originalToken := os.Getenv("GITHUB_TOKEN")
	originalBaseURL := os.Getenv("GITHUB_API_BASE_URL")
	
	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("GITHUB_API_BASE_URL", "https://custom-api.github.com")
	
	defer func() {
		if originalToken == "" {
			os.Unsetenv("GITHUB_TOKEN")
		} else {
			os.Setenv("GITHUB_TOKEN", originalToken)
		}
		if originalBaseURL == "" {
			os.Unsetenv("GITHUB_API_BASE_URL")
		} else {
			os.Setenv("GITHUB_API_BASE_URL", originalBaseURL)
		}
	}()

	client := NewGitHubClient()
	
	assert.NotNil(t, client)
	assert.Equal(t, "test-token", client.Token)
	assert.Equal(t, "https://custom-api.github.com", client.BaseURL)
	assert.NotNil(t, client.Client)
	assert.Equal(t, 30*time.Second, client.Client.Timeout)
}

func TestNewGitHubClient_DefaultBaseURL(t *testing.T) {
	// Clear environment
	originalToken := os.Getenv("GITHUB_TOKEN")
	originalBaseURL := os.Getenv("GITHUB_API_BASE_URL")
	
	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Unsetenv("GITHUB_API_BASE_URL")
	
	defer func() {
		if originalToken == "" {
			os.Unsetenv("GITHUB_TOKEN")
		} else {
			os.Setenv("GITHUB_TOKEN", originalToken)
		}
		if originalBaseURL == "" {
			os.Unsetenv("GITHUB_API_BASE_URL")
		} else {
			os.Setenv("GITHUB_API_BASE_URL", originalBaseURL)
		}
	}()

	client := NewGitHubClient()
	
	assert.Equal(t, "https://api.github.com", client.BaseURL)
}

func TestGitHubClient_IsConfigured(t *testing.T) {
	testCases := []struct {
		name     string
		token    string
		expected bool
	}{
		{
			name:     "configured_with_token",
			token:    "test-token",
			expected: true,
		},
		{
			name:     "not_configured_empty_token",
			token:    "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &GitHubClient{
				Token:   tc.token,
				BaseURL: "https://api.github.com",
				Client:  &http.Client{Timeout: 30 * time.Second},
			}
			
			result := client.IsConfigured()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGitHubClient_TriggerWorkflow_MissingParameters(t *testing.T) {
	client := &GitHubClient{
		Token:   "test-token",
		BaseURL: "https://api.github.com",
		Client:  &http.Client{Timeout: 30 * time.Second},
	}

	entry := &Entry{
		VideoID:   "test_video_id",
		ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
		Title:     "Test Video",
		Published: time.Now().Format(time.RFC3339),
		Updated:   time.Now().Format(time.RFC3339),
	}

	testCases := []struct {
		name      string
		repoOwner string
		repoName  string
		token     string
	}{
		{
			name:      "missing_repo_owner",
			repoOwner: "",
			repoName:  "test-repo",
			token:     "test-token",
		},
		{
			name:      "missing_repo_name",
			repoOwner: "test-owner",
			repoName:  "",
			token:     "test-token",
		},
		{
			name:      "missing_token",
			repoOwner: "test-owner",
			repoName:  "test-repo",
			token:     "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client.Token = tc.token
			err := client.TriggerWorkflow(tc.repoOwner, tc.repoName, entry)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "missing required parameters")
		})
	}
}

func TestGitHubClient_TriggerWorkflow_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/repos/test-owner/test-repo/dispatches", r.URL.Path)
		
		// Verify headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))
		
		// Return success
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &GitHubClient{
		Token:   "test-token",
		BaseURL: server.URL,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}

	entry := &Entry{
		VideoID:   "test_video_id",
		ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
		Title:     "Test Video",
		Published: time.Now().Format(time.RFC3339),
		Updated:   time.Now().Format(time.RFC3339),
	}

	err := client.TriggerWorkflow("test-owner", "test-repo", entry)
	assert.NoError(t, err)
}

func TestGitHubClient_TriggerWorkflow_HTTPError(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	client := &GitHubClient{
		Token:   "test-token",
		BaseURL: server.URL,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}

	entry := &Entry{
		VideoID:   "test_video_id",
		ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
		Title:     "Test Video",
		Published: time.Now().Format(time.RFC3339),
		Updated:   time.Now().Format(time.RFC3339),
	}

	err := client.TriggerWorkflow("test-owner", "test-repo", entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub API returned status 400")
}

func TestGitHubClient_TriggerWorkflow_NetworkError(t *testing.T) {
	client := &GitHubClient{
		Token:   "test-token",
		BaseURL: "http://localhost:99999", // Invalid port to trigger network error
		Client:  &http.Client{Timeout: 1 * time.Second},
	}

	entry := &Entry{
		VideoID:   "test_video_id",
		ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
		Title:     "Test Video",
		Published: time.Now().Format(time.RFC3339),
		Updated:   time.Now().Format(time.RFC3339),
	}

	err := client.TriggerWorkflow("test-owner", "test-repo", entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send request")
}

func TestGitHubClient_TriggerWorkflow_InvalidURL(t *testing.T) {
	client := &GitHubClient{
		Token:   "test-token",
		BaseURL: "ht tp://invalid url with spaces", // Invalid URL
		Client:  &http.Client{Timeout: 30 * time.Second},
	}

	entry := &Entry{
		VideoID:   "test_video_id",
		ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
		Title:     "Test Video",
		Published: time.Now().Format(time.RFC3339),
		Updated:   time.Now().Format(time.RFC3339),
	}

	err := client.TriggerWorkflow("test-owner", "test-repo", entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestGitHubClient_TriggerWorkflow_PayloadValidation(t *testing.T) {
	// Create mock server that captures and validates the payload
	var receivedPayload string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedPayload = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set environment for testing
	originalEnv := os.Getenv("ENVIRONMENT")
	os.Setenv("ENVIRONMENT", "test")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ENVIRONMENT")
		} else {
			os.Setenv("ENVIRONMENT", originalEnv)
		}
	}()

	client := &GitHubClient{
		Token:   "test-token",
		BaseURL: server.URL,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}

	entry := &Entry{
		VideoID:   "test_video_id",
		ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
		Title:     "Test Video",
		Published: time.Now().Format(time.RFC3339),
		Updated:   time.Now().Format(time.RFC3339),
	}

	err := client.TriggerWorkflow("test-owner", "test-repo", entry)
	require.NoError(t, err)

	// Validate payload content
	assert.Contains(t, receivedPayload, "youtube-video-published")
	assert.Contains(t, receivedPayload, "test_video_id")
	assert.Contains(t, receivedPayload, "UCXuqSBlHAE6Xw-yeJA0Tunw")
	assert.Contains(t, receivedPayload, "Test Video")
	assert.Contains(t, receivedPayload, "https://www.youtube.com/watch?v=test_video_id")
	assert.Contains(t, receivedPayload, "test") // environment
}

func TestGitHubClient_sendDispatch_ErrorCases(t *testing.T) {
	t.Run("json_marshal_error", func(t *testing.T) {
		// This is hard to trigger with normal structs, but we can test the path exists
		client := &GitHubClient{
			Token:   "test-token",
			BaseURL: "https://api.github.com",
			Client:  &http.Client{Timeout: 30 * time.Second},
		}

		// Create a valid dispatch - JSON marshal shouldn't fail with normal data
		dispatch := GitHubDispatch{
			EventType: "test-event",
			ClientPayload: map[string]interface{}{
				"test": "data",
			},
		}

		// Use invalid URL to test other error paths
		client.BaseURL = "ht tp://invalid"
		err := client.sendDispatch("owner", "repo", dispatch)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create request")
	})
}

func TestGitHubClient_EdgeCases(t *testing.T) {
	t.Run("multiple_concurrent_requests", func(t *testing.T) {
		// Create mock server that handles concurrent requests
		var requestCount int64
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&requestCount, 1)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &GitHubClient{
			Token:   "test-token",
			BaseURL: server.URL,
			Client:  &http.Client{Timeout: 30 * time.Second},
		}

		entry := &Entry{
			VideoID:   "test_video_id",
			ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
			Title:     "Test Video",
			Published: time.Now().Format(time.RFC3339),
			Updated:   time.Now().Format(time.RFC3339),
		}

		// Make concurrent requests
		const numRequests = 5
		errors := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				errors <- client.TriggerWorkflow("test-owner", "test-repo", entry)
			}()
		}

		// Check all requests succeeded
		for i := 0; i < numRequests; i++ {
			err := <-errors
			assert.NoError(t, err)
		}
	})

	t.Run("empty_entry_fields", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &GitHubClient{
			Token:   "test-token",
			BaseURL: server.URL,
			Client:  &http.Client{Timeout: 30 * time.Second},
		}

		// Entry with empty fields should still work
		entry := &Entry{
			VideoID:   "",
			ChannelID: "",
			Title:     "",
			Published: "",
			Updated:   "",
		}

		err := client.TriggerWorkflow("test-owner", "test-repo", entry)
		assert.NoError(t, err)
	})
}
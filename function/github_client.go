package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// GitHubClient handles GitHub API interactions
type GitHubClient struct {
	Token   string
	BaseURL string
	Client  *http.Client
}

// NewGitHubClient creates a new GitHub API client
func NewGitHubClient() *GitHubClient {
	token := os.Getenv("GITHUB_TOKEN")
	baseURL := os.Getenv("GITHUB_API_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	return &GitHubClient{
		Token:   token,
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// IsConfigured returns whether the GitHub client is configured with a token.
func (gc *GitHubClient) IsConfigured() bool {
	return gc.Token != ""
}

// TriggerWorkflow sends a repository dispatch event to trigger a GitHub workflow
func (gc *GitHubClient) TriggerWorkflow(repoOwner, repoName string, entry *Entry) error {
	if gc.Token == "" || repoOwner == "" || repoName == "" {
		return fmt.Errorf("missing required parameters for GitHub workflow trigger")
	}

	environment := os.Getenv("ENVIRONMENT")

	// Create dispatch payload
	dispatch := GitHubDispatch{
		EventType: "youtube-video-published",
		ClientPayload: map[string]interface{}{
			"video_id":    entry.VideoID,
			"channel_id":  entry.ChannelID,
			"title":       entry.Title,
			"published":   entry.Published,
			"updated":     entry.Updated,
			"video_url":   fmt.Sprintf("https://www.youtube.com/watch?v=%s", entry.VideoID),
			"environment": environment,
		},
	}

	return gc.sendDispatch(repoOwner, repoName, dispatch)
}

// sendDispatch performs the actual HTTP request to GitHub API
func (gc *GitHubClient) sendDispatch(repoOwner, repoName string, dispatch GitHubDispatch) error {
	// Marshal to JSON
	jsonData, err := json.Marshal(dispatch)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/repos/%s/%s/dispatches", gc.BaseURL, repoOwner, repoName)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", gc.Token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send request
	resp, err := gc.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	return nil
}

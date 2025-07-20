package webhook

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

// AtomFeed represents the structure of a YouTube Atom feed notification
type AtomFeed struct {
	XMLName xml.Name `xml:"feed"`
	Entry   *Entry   `xml:"entry"`
}

// Entry represents a single video entry in the YouTube Atom feed
type Entry struct {
	VideoID   string `xml:"http://www.youtube.com/xml/schemas/2015 videoId"`
	ChannelID string `xml:"http://www.youtube.com/xml/schemas/2015 channelId"`
	Title     string `xml:"title"`
	Published string `xml:"published"`
	Updated   string `xml:"updated"`
}

// GitHubDispatch represents the payload structure for GitHub repository dispatch events
type GitHubDispatch struct {
	EventType     string                 `json:"event_type"`
	ClientPayload map[string]interface{} `json:"client_payload"`
}

func init() {
	functions.HTTP("YouTubeWebhook", YouTubeWebhook)
}

// YouTubeWebhook handles YouTube PubSubHubbub notifications
func YouTubeWebhook(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for all requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	switch r.Method {
	case http.MethodGet:
		handleVerificationChallenge(w, r)
	case http.MethodPost:
		handleNotification(w, r)
	case http.MethodOptions:
		// CORS preflight request
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		if _, err := w.Write([]byte("Method not allowed")); err != nil {
			fmt.Printf("Error writing response: %v\n", err)
		}
	}
}

// handleVerificationChallenge handles YouTube's verification challenge
func handleVerificationChallenge(w http.ResponseWriter, r *http.Request) {
	challenge := r.URL.Query().Get("hub.challenge")
	if challenge == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(challenge)); err != nil {
		fmt.Printf("Error writing response: %v\n", err)
	}
}

// handleNotification processes YouTube webhook notifications
func handleNotification(w http.ResponseWriter, r *http.Request) {
	// Read and parse XML payload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("Failed to read request body")); err != nil {
			fmt.Printf("Error writing response: %v\n", err)
		}
		return
	}

	var feed AtomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("Invalid XML")); err != nil {
			fmt.Printf("Error writing response: %v\n", err)
		}
		return
	}

	// Handle empty notifications
	if feed.Entry == nil {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("No video data")); err != nil {
			fmt.Printf("Error writing response: %v\n", err)
		}
		return
	}

	// Check if this is a new video or just an update
	if !isNewVideo(feed.Entry) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("Video update ignored")); err != nil {
			fmt.Printf("Error writing response: %v\n", err)
		}
		return
	}

	// Trigger GitHub workflow
	if err := triggerGitHubWorkflow(feed.Entry); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("GitHub API error")); err != nil {
			fmt.Printf("Error writing response: %v\n", err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Webhook processed successfully")); err != nil {
		fmt.Printf("Error writing response: %v\n", err)
	}
}

// triggerGitHubWorkflow sends a repository dispatch event to GitHub
func triggerGitHubWorkflow(entry *Entry) error {
	token := os.Getenv("GITHUB_TOKEN")
	repoOwner := os.Getenv("REPO_OWNER")
	repoName := os.Getenv("REPO_NAME")
	environment := os.Getenv("ENVIRONMENT")

	if token == "" || repoOwner == "" || repoName == "" {
		return fmt.Errorf("missing required environment variables")
	}

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

	// Marshal to JSON
	jsonData, err := json.Marshal(dispatch)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// Create HTTP request - allow override for testing
	baseURL := os.Getenv("GITHUB_API_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	url := fmt.Sprintf("%s/repos/%s/%s/dispatches", baseURL, repoOwner, repoName)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	return nil
}

// isNewVideo determines if this is a new video or just an update
func isNewVideo(entry *Entry) bool {
	// Parse timestamps
	published, err := time.Parse(time.RFC3339, entry.Published)
	if err != nil {
		// If we can't parse the timestamp, assume it's new
		return true
	}

	updated, err := time.Parse(time.RFC3339, entry.Updated)
	if err != nil {
		// If we can't parse the timestamp, assume it's new
		return true
	}

	now := time.Now()

	// Consider a video "new" if:
	// 1. It was published within the last hour
	// 2. The difference between published and updated time is small (less than 15 minutes)
	timeSincePublished := now.Sub(published)
	updatePublishDiff := updated.Sub(published)

	// If published more than 1 hour ago, it's likely an old video update
	if timeSincePublished > time.Hour {
		return false
	}

	// If there's a large gap between publish and update, it's likely an update to an old video
	if updatePublishDiff > 15*time.Minute {
		return false
	}

	return true
}


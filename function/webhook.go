package webhook

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/storage"
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

// Subscription represents a YouTube channel subscription
type Subscription struct {
	ChannelID       string    `json:"channel_id"`
	ChannelName     string    `json:"channel_name,omitempty"`
	TopicURL        string    `json:"topic_url"`
	CallbackURL     string    `json:"callback_url"`
	Status          string    `json:"status"`
	LeaseSeconds    int       `json:"lease_seconds"`
	SubscribedAt    time.Time `json:"subscribed_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	LastRenewal     time.Time `json:"last_renewal"`
	RenewalAttempts int       `json:"renewal_attempts"`
	HubResponse     string    `json:"hub_response"`
}

// SubscriptionState represents the complete subscription state stored in Cloud Storage
type SubscriptionState struct {
	Subscriptions map[string]*Subscription `json:"subscriptions"`
	Metadata      struct {
		LastUpdated time.Time `json:"last_updated"`
		Version     string    `json:"version"`
	} `json:"metadata"`
}

// API Response types
type APIResponse struct {
	Status    string `json:"status"`
	ChannelID string `json:"channel_id,omitempty"`
	Message   string `json:"message,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type SubscriptionsListResponse struct {
	Subscriptions []SubscriptionInfo `json:"subscriptions"`
	Total         int                `json:"total"`
	Active        int                `json:"active"`
	Expired       int                `json:"expired"`
}

type SubscriptionInfo struct {
	ChannelID       string  `json:"channel_id"`
	Status          string  `json:"status"`
	ExpiresAt       string  `json:"expires_at"`
	DaysUntilExpiry float64 `json:"days_until_expiry"`
}

// Renewal Response types
type RenewalSummaryResponse struct {
	Status             string          `json:"status"`
	TotalChecked       int             `json:"total_checked"`
	RenewalsCandidates int             `json:"renewals_candidates"`
	RenewalsSucceeded  int             `json:"renewals_succeeded"`
	RenewalsFailed     int             `json:"renewals_failed"`
	Results            []RenewalResult `json:"results"`
}

type RenewalResult struct {
	ChannelID     string `json:"channel_id"`
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	NewExpiryTime string `json:"new_expiry_time,omitempty"`
	AttemptCount  int    `json:"attempt_count"`
}

// Channel ID validation regex
var channelIDRegex = regexp.MustCompile(`^UC[a-zA-Z0-9_-]{22}$`)

// StorageInterface defines the contract for subscription state storage operations
type StorageInterface interface {
	LoadSubscriptionState(ctx context.Context) (*SubscriptionState, error)
	SaveSubscriptionState(ctx context.Context, state *SubscriptionState) error
}

// CloudStorageClient implements StorageInterface using Google Cloud Storage
type CloudStorageClient struct{}

// CloudStorageClient is the production storage implementation
// For testing, use dependency injection with MockStorageClient

func init() {
	functions.HTTP("YouTubeWebhook", YouTubeWebhook)
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


// Backward compatibility functions for existing tests

// triggerGitHubWorkflow is a backward compatibility function that uses the new GitHubClient
func triggerGitHubWorkflow(entry *Entry) error {
	client := NewGitHubClient()
	repoOwner := os.Getenv("REPO_OWNER")
	repoName := os.Getenv("REPO_NAME")
	return client.TriggerWorkflow(repoOwner, repoName, entry)
}

// isNewVideo is a backward compatibility function that uses the new VideoProcessor
func isNewVideo(entry *Entry) bool {
	processor := NewVideoProcessor()
	return processor.IsNewVideo(entry)
}

// validateChannelID validates YouTube channel ID format
func validateChannelID(channelID string) bool {
	return channelIDRegex.MatchString(channelID)
}

// makePubSubHubbubRequest makes a subscription or unsubscription request to the PubSubHubbub hub
// Note: This legacy function is kept for backward compatibility but should use dependency injection instead
func makePubSubHubbubRequest(channelID, mode string) error {

	hubURL := "https://pubsubhubbub.appspot.com/subscribe"

	// Get callback URL from environment or construct default
	callbackURL := os.Getenv("FUNCTION_URL")
	if callbackURL == "" {
		return fmt.Errorf("FUNCTION_URL environment variable not set")
	}

	// Construct topic URL for the YouTube channel
	topicURL := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelID)

	// Prepare form data
	formData := url.Values{
		"hub.callback":      {callbackURL},
		"hub.topic":         {topicURL},
		"hub.mode":          {mode}, // "subscribe" or "unsubscribe"
		"hub.verify":        {"async"},
		"hub.lease_seconds": {"86400"}, // 24 hours
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", hubURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "YouTube-Webhook-Handler/1.0")

	// Send request with timeout
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hub returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// LoadSubscriptionState loads subscription state from Cloud Storage
func (c *CloudStorageClient) LoadSubscriptionState(ctx context.Context) (*SubscriptionState, error) {

	bucketName := os.Getenv("SUBSCRIPTION_BUCKET")
	if bucketName == "" {
		return nil, fmt.Errorf("SUBSCRIPTION_BUCKET environment variable not set")
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %v", err)
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)
	obj := bucket.Object("subscriptions/state.json")

	reader, err := obj.NewReader(ctx)
	if err != nil {
		// If file doesn't exist, return empty state
		if err == storage.ErrObjectNotExist {
			return &SubscriptionState{
				Subscriptions: make(map[string]*Subscription),
				Metadata: struct {
					LastUpdated time.Time `json:"last_updated"`
					Version     string    `json:"version"`
				}{
					LastUpdated: time.Now(),
					Version:     "1.0",
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read state data: %v", err)
	}

	var state SubscriptionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %v", err)
	}

	// Initialize subscriptions map if nil
	if state.Subscriptions == nil {
		state.Subscriptions = make(map[string]*Subscription)
	}

	return &state, nil
}

// SaveSubscriptionState saves subscription state to Cloud Storage
func (c *CloudStorageClient) SaveSubscriptionState(ctx context.Context, state *SubscriptionState) error {

	bucketName := os.Getenv("SUBSCRIPTION_BUCKET")
	if bucketName == "" {
		return fmt.Errorf("SUBSCRIPTION_BUCKET environment variable not set")
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %v", err)
	}
	defer client.Close()

	// Update metadata
	state.Metadata.LastUpdated = time.Now()
	if state.Metadata.Version == "" {
		state.Metadata.Version = "1.0"
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %v", err)
	}

	bucket := client.Bucket(bucketName)
	obj := bucket.Object("subscriptions/state.json")

	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/json"

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write state data: %v", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}

	return nil
}

// writeJSONResponse writes a JSON response with the given status code
func writeJSONResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf("Error encoding JSON response: %v\n", err)
	}
}

// writeErrorResponse writes a standardized error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, channelID, message string) {
	response := APIResponse{
		Status:  "error",
		Message: message,
	}
	if channelID != "" {
		response.ChannelID = channelID
	}
	writeJSONResponse(w, statusCode, response)
}






// Configuration helper functions


// getRenewalThreshold returns the time threshold for renewal
func getRenewalThreshold() time.Duration {
	thresholdHours := os.Getenv("RENEWAL_THRESHOLD_HOURS")
	if thresholdHours == "" {
		return 12 * time.Hour // Default: 12 hours
	}

	if hours, err := time.ParseDuration(thresholdHours + "h"); err == nil {
		return hours
	}
	return 12 * time.Hour
}

// getMaxRenewalAttempts returns the maximum number of renewal attempts
func getMaxRenewalAttempts() int {
	maxAttemptsStr := os.Getenv("MAX_RENEWAL_ATTEMPTS")
	if maxAttemptsStr == "" {
		return 3 // Default: 3 attempts
	}

	var attempts int
	if _, err := fmt.Sscanf(maxAttemptsStr, "%d", &attempts); err == nil && attempts > 0 {
		return attempts
	}
	return 3
}

// getLeaseSeconds returns the lease duration in seconds
func getLeaseSeconds() int {
	leaseSecondsStr := os.Getenv("SUBSCRIPTION_LEASE_SECONDS")
	if leaseSecondsStr == "" {
		return 86400 // Default: 24 hours
	}

	var seconds int
	if _, err := fmt.Sscanf(leaseSecondsStr, "%d", &seconds); err == nil && seconds > 0 {
		return seconds
	}
	return 86400
}

// Legacy functions removed - use dependency injection instead

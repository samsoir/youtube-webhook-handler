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
	"sync"
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
	ChannelID        string  `json:"channel_id"`
	Status           string  `json:"status"`
	ExpiresAt        string  `json:"expires_at"`
	DaysUntilExpiry  float64 `json:"days_until_expiry"`
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

// Global storage client and test state (for testing only)
var storageClient StorageInterface = &CloudStorageClient{}
var testSubscriptionState *SubscriptionState
var testSubscriptionStateMutex sync.RWMutex
var testMode bool

func init() {
	functions.HTTP("YouTubeWebhook", YouTubeWebhook)
}

// YouTubeWebhook handles YouTube PubSubHubbub notifications and subscription management
func YouTubeWebhook(w http.ResponseWriter, r *http.Request) {
	// Check if refactored router should be used
	if useRefactoredRouter() {
		YouTubeWebhookRefactored(w, r)
		return
	}

	// Original router implementation
	// Set CORS headers for all requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// Route based on path and method
	path := strings.TrimPrefix(r.URL.Path, "/")
	
	switch {
	case path == "subscribe" && r.Method == http.MethodPost:
		handleSubscribe(w, r)
	case path == "unsubscribe" && r.Method == http.MethodDelete:
		handleUnsubscribe(w, r)
	case path == "subscriptions" && r.Method == http.MethodGet:
		handleGetSubscriptions(w, r)
	case path == "renew" && r.Method == http.MethodPost:
		handleRenewSubscriptions(w, r)
	case r.Method == http.MethodGet:
		// Default GET behavior - YouTube verification challenge
		handleVerificationChallenge(w, r)
	case r.Method == http.MethodPost:
		// Default POST behavior - YouTube notifications
		handleNotification(w, r)
	case r.Method == http.MethodOptions:
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

// handleNotification processes YouTube webhook notifications using the notification service
func handleNotification(w http.ResponseWriter, r *http.Request) {
	notificationService := NewNotificationService()
	
	result, err := notificationService.ProcessNotification(r)
	if err != nil {
		if result.Message == "Failed to read request body" || result.Message == "Invalid XML" {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		if _, writeErr := w.Write([]byte(result.Message)); writeErr != nil {
			fmt.Printf("Error writing response: %v\n", writeErr)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(result.Message)); err != nil {
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
func makePubSubHubbubRequest(channelID, mode string) error {
	// Skip actual hub request in test mode
	if testMode {
		return nil
	}
	
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
		"hub.callback":     {callbackURL},
		"hub.topic":        {topicURL},
		"hub.mode":         {mode}, // "subscribe" or "unsubscribe"
		"hub.verify":       {"async"},
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
	// Use test state in test mode
	if testMode {
		if testSubscriptionState == nil {
			testSubscriptionState = &SubscriptionState{
				Subscriptions: make(map[string]*Subscription),
				Metadata: struct {
					LastUpdated time.Time `json:"last_updated"`
					Version     string    `json:"version"`
				}{
					LastUpdated: time.Now(),
					Version:     "1.0",
				},
			}
		}
		// Return a copy to avoid test interference
		stateCopy := *testSubscriptionState
		stateCopy.Subscriptions = make(map[string]*Subscription)
		for k, v := range testSubscriptionState.Subscriptions {
			subCopy := *v
			stateCopy.Subscriptions[k] = &subCopy
		}
		return &stateCopy, nil
	}
	
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
	// Use test state in test mode
	if testMode {
		if testSubscriptionState == nil {
			testSubscriptionState = &SubscriptionState{
				Subscriptions: make(map[string]*Subscription),
			}
		}
		// Update the test state
		state.Metadata.LastUpdated = time.Now()
		if state.Metadata.Version == "" {
			state.Metadata.Version = "1.0"
		}
		testSubscriptionState = state
		return nil
	}
	
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

// handleSubscribe handles POST /subscribe requests
func handleSubscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get and validate channel_id parameter
	channelID := r.URL.Query().Get("channel_id")
	if channelID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "", "channel_id parameter is required")
		return
	}

	// Validate channel ID format
	if !validateChannelID(channelID) {
		writeErrorResponse(w, http.StatusBadRequest, channelID, 
			"Invalid channel ID format. Must be UC followed by 22 alphanumeric characters")
		return
	}

	// Load current subscription state
	state, err := storageClient.LoadSubscriptionState(ctx)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, channelID, 
			fmt.Sprintf("Failed to load subscription state: %v", err))
		return
	}
	
	// Check if already subscribed
	if existing, exists := state.Subscriptions[channelID]; exists {
		// Return conflict response with existing expiration
		response := APIResponse{
			Status:    "conflict",
			ChannelID: channelID,
			Message:   "Already subscribed to this channel",
			ExpiresAt: existing.ExpiresAt.Format(time.RFC3339),
		}
		writeJSONResponse(w, http.StatusConflict, response)
		return
	}
	
	// Make PubSubHubbub subscription request
	if err := makePubSubHubbubRequest(channelID, "subscribe"); err != nil {
		writeErrorResponse(w, http.StatusBadGateway, channelID, 
			fmt.Sprintf("PubSubHubbub subscription failed: %v", err))
		return
	}
	
	// Create subscription record
	callbackURL := os.Getenv("FUNCTION_URL")
	if callbackURL == "" && testMode {
		callbackURL = "https://test-function-url"
	}
	topicURL := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelID)
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	
	subscription := &Subscription{
		ChannelID:       channelID,
		TopicURL:        topicURL,
		CallbackURL:     callbackURL,
		Status:          "active",
		LeaseSeconds:    86400,
		SubscribedAt:    now,
		ExpiresAt:       expiresAt,
		LastRenewal:     now,
		RenewalAttempts: 0,
		HubResponse:     "202 Accepted",
	}
	
	// Store subscription state
	state.Subscriptions[channelID] = subscription
	if err := storageClient.SaveSubscriptionState(ctx, state); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, channelID, 
			fmt.Sprintf("Failed to save subscription state: %v", err))
		return
	}
	
	// Return success response
	response := APIResponse{
		Status:    "success",
		ChannelID: channelID,
		Message:   "Subscription initiated",
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}
	writeJSONResponse(w, http.StatusOK, response)
}

// handleUnsubscribe handles DELETE /unsubscribe requests
func handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get and validate channel_id parameter
	channelID := r.URL.Query().Get("channel_id")
	if channelID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "", "channel_id parameter is required")
		return
	}

	// Validate channel ID format
	if !validateChannelID(channelID) {
		writeErrorResponse(w, http.StatusBadRequest, channelID, "Invalid channel ID format")
		return
	}

	// Load current subscription state
	state, err := storageClient.LoadSubscriptionState(ctx)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, channelID, 
			fmt.Sprintf("Failed to load subscription state: %v", err))
		return
	}
	
	// Check if subscription exists
	if _, exists := state.Subscriptions[channelID]; !exists {
		writeErrorResponse(w, http.StatusNotFound, channelID, 
			"Subscription not found for this channel")
		return
	}
	
	// Make PubSubHubbub unsubscribe request
	if err := makePubSubHubbubRequest(channelID, "unsubscribe"); err != nil {
		writeErrorResponse(w, http.StatusBadGateway, channelID, 
			fmt.Sprintf("PubSubHubbub unsubscribe failed: %v", err))
		return
	}
	
	// Remove from subscription state
	delete(state.Subscriptions, channelID)
	if err := storageClient.SaveSubscriptionState(ctx, state); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, channelID, 
			fmt.Sprintf("Failed to save subscription state: %v", err))
		return
	}
	
	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// handleGetSubscriptions handles GET /subscriptions requests
func handleGetSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Load subscription state from Cloud Storage
	state, err := storageClient.LoadSubscriptionState(ctx)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "", 
			fmt.Sprintf("Unable to load subscription state from storage: %v", err))
		return
	}
	
	// Calculate expiry status and statistics
	now := time.Now()
	subscriptions := make([]SubscriptionInfo, 0)
	total := 0
	active := 0
	expired := 0
	
	for _, sub := range state.Subscriptions {
		total++
		
		status := "active"
		daysUntilExpiry := sub.ExpiresAt.Sub(now).Hours() / 24
		
		if sub.ExpiresAt.Before(now) {
			status = "expired"
			expired++
		} else {
			active++
		}
		
		subscriptions = append(subscriptions, SubscriptionInfo{
			ChannelID:       sub.ChannelID,
			Status:          status,
			ExpiresAt:       sub.ExpiresAt.Format(time.RFC3339),
			DaysUntilExpiry: daysUntilExpiry,
		})
	}
	
	response := SubscriptionsListResponse{
		Subscriptions: subscriptions,
		Total:         total,
		Active:        active,
		Expired:       expired,
	}
	writeJSONResponse(w, http.StatusOK, response)
}

// handleRenewSubscriptions handles POST /renew requests from Cloud Scheduler
func handleRenewSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Load current subscription state
	state, err := storageClient.LoadSubscriptionState(ctx)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "", 
			fmt.Sprintf("Failed to load subscription state: %v", err))
		return
	}
	
	// Find subscriptions that need renewal
	renewalThreshold := getRenewalThreshold()
	now := time.Now()
	
	var renewalResults []RenewalResult
	var successCount, failureCount int
	
	for channelID, subscription := range state.Subscriptions {
		timeUntilExpiry := subscription.ExpiresAt.Sub(now)
		
		// Check if subscription needs renewal
		if timeUntilExpiry <= renewalThreshold {
			result := renewSubscription(ctx, channelID, subscription, state)
			renewalResults = append(renewalResults, result)
			
			if result.Success {
				successCount++
			} else {
				failureCount++
				// Increment failure count for monitoring
				subscription.RenewalAttempts++
			}
		}
	}
	
	// Save updated state if there were any changes
	if len(renewalResults) > 0 {
		if err := storageClient.SaveSubscriptionState(ctx, state); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "", 
				fmt.Sprintf("Failed to save subscription state: %v", err))
			return
		}
	}
	
	// Return renewal summary
	response := RenewalSummaryResponse{
		Status:           "success",
		TotalChecked:     len(state.Subscriptions),
		RenewalsCandidates: len(renewalResults),
		RenewalsSucceeded: successCount,
		RenewalsFailed:   failureCount,
		Results:          renewalResults,
	}
	
	writeJSONResponse(w, http.StatusOK, response)
}

// renewSubscription attempts to renew a single subscription
func renewSubscription(ctx context.Context, channelID string, subscription *Subscription, state *SubscriptionState) RenewalResult {
	maxAttempts := getMaxRenewalAttempts()
	
	// Check if we've exceeded max attempts
	if subscription.RenewalAttempts >= maxAttempts {
		return RenewalResult{
			ChannelID: channelID,
			Success:   false,
			Message:   fmt.Sprintf("Max renewal attempts (%d) exceeded", maxAttempts),
			AttemptCount: subscription.RenewalAttempts,
		}
	}
	
	// Attempt to renew the subscription
	err := makePubSubHubbubRequest(channelID, "subscribe")
	if err != nil {
		return RenewalResult{
			ChannelID: channelID,
			Success:   false,
			Message:   fmt.Sprintf("PubSubHubbub renewal failed: %v", err),
			AttemptCount: subscription.RenewalAttempts + 1,
		}
	}
	
	// Update subscription with new expiry time
	now := time.Now()
	leaseSeconds := getLeaseSeconds()
	subscription.ExpiresAt = now.Add(time.Duration(leaseSeconds) * time.Second)
	subscription.LastRenewal = now
	subscription.RenewalAttempts = 0 // Reset on successful renewal
	subscription.HubResponse = "202 Accepted (Renewed)"
	
	return RenewalResult{
		ChannelID: channelID,
		Success:   true,
		Message:   "Subscription renewed successfully",
		NewExpiryTime: subscription.ExpiresAt.Format(time.RFC3339),
		AttemptCount: 0,
	}
}

// Configuration helper functions

// useRefactoredRouter determines whether to use the refactored router with dependency injection
func useRefactoredRouter() bool {
	// Check environment variable to enable refactored router
	useRefactored := os.Getenv("USE_REFACTORED_ROUTER")
	return useRefactored == "true" || useRefactored == "1"
}

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

// Test helper functions for accessing private variables

// GetStorageClient returns the current storage client
func GetStorageClient() StorageInterface {
	return storageClient
}

// SetStorageClient sets the storage client (for testing)
func SetStorageClient(client StorageInterface) {
	storageClient = client
}

// GetTestMode returns the current test mode state
func GetTestMode() bool {
	return testMode
}

// SetTestMode sets the test mode state (for testing)
func SetTestMode(mode bool) {
	testMode = mode
}

// GetTestSubscriptionState returns the current test subscription state
func GetTestSubscriptionState() *SubscriptionState {
	return testSubscriptionState
}

// SetTestSubscriptionState sets the test subscription state (for testing)
func SetTestSubscriptionState(state *SubscriptionState) {
	testSubscriptionState = state
}


package webhook

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// handleSubscribeWithDeps handles POST /subscribe requests using dependency injection.
// This is the refactored version that doesn't rely on global testMode.
func handleSubscribeWithDeps(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// Load current subscription state using injected storage client
		state, err := deps.StorageClient.LoadSubscriptionState(ctx)
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

		// Make PubSubHubbub subscription request using injected client
		if err := deps.PubSubClient.Subscribe(channelID); err != nil {
			writeErrorResponse(w, http.StatusBadGateway, channelID,
				fmt.Sprintf("PubSubHubbub subscription failed: %v", err))
			return
		}

		// Create subscription record
		callbackURL := os.Getenv("FUNCTION_URL")
		if callbackURL == "" {
			callbackURL = "https://default-function-url"
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

		// Store subscription state using injected storage client
		state.Subscriptions[channelID] = subscription
		if err := deps.StorageClient.SaveSubscriptionState(ctx, state); err != nil {
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
}

// handleSubscribeRefactored is a compatibility wrapper that uses the refactored function.
// This allows us to test the refactored version while keeping the original intact.
func handleSubscribeRefactored(w http.ResponseWriter, r *http.Request) {
	deps := GetDependencies()
	handler := handleSubscribeWithDeps(deps)
	handler(w, r)
}

// handleUnsubscribeWithDeps handles DELETE /unsubscribe requests using dependency injection.
func handleUnsubscribeWithDeps(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// Load current subscription state using injected storage client
		state, err := deps.StorageClient.LoadSubscriptionState(ctx)
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

		// Make PubSubHubbub unsubscribe request using injected client
		if err := deps.PubSubClient.Unsubscribe(channelID); err != nil {
			writeErrorResponse(w, http.StatusBadGateway, channelID,
				fmt.Sprintf("PubSubHubbub unsubscribe failed: %v", err))
			return
		}

		// Remove from subscription state
		delete(state.Subscriptions, channelID)
		if err := deps.StorageClient.SaveSubscriptionState(ctx, state); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, channelID,
				fmt.Sprintf("Failed to save subscription state: %v", err))
			return
		}

		// Return 204 No Content
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleUnsubscribeRefactored is a compatibility wrapper that uses the refactored function.
func handleUnsubscribeRefactored(w http.ResponseWriter, r *http.Request) {
	deps := GetDependencies()
	handler := handleUnsubscribeWithDeps(deps)
	handler(w, r)
}

// handleRenewSubscriptionsWithDeps handles POST /renew requests using dependency injection.
func handleRenewSubscriptionsWithDeps(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Load current subscription state using injected storage client
		state, err := deps.StorageClient.LoadSubscriptionState(ctx)
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
				result := renewSubscriptionWithDeps(ctx, channelID, subscription, state, deps)
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
			if err := deps.StorageClient.SaveSubscriptionState(ctx, state); err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, "",
					fmt.Sprintf("Failed to save subscription state: %v", err))
				return
			}
		}

		// Return renewal summary
		response := RenewalSummaryResponse{
			Status:             "success",
			TotalChecked:       len(state.Subscriptions),
			RenewalsCandidates: len(renewalResults),
			RenewalsSucceeded:  successCount,
			RenewalsFailed:     failureCount,
			Results:            renewalResults,
		}

		writeJSONResponse(w, http.StatusOK, response)
	}
}

// renewSubscriptionWithDeps attempts to renew a single subscription using dependency injection.
func renewSubscriptionWithDeps(ctx context.Context, channelID string, subscription *Subscription, state *SubscriptionState, deps *Dependencies) RenewalResult {
	maxAttempts := getMaxRenewalAttempts()

	// Check if we've exceeded max attempts
	if subscription.RenewalAttempts >= maxAttempts {
		return RenewalResult{
			ChannelID:    channelID,
			Success:      false,
			Message:      fmt.Sprintf("Max renewal attempts (%d) exceeded", maxAttempts),
			AttemptCount: subscription.RenewalAttempts,
		}
	}

	// Attempt to renew the subscription using injected PubSub client
	err := deps.PubSubClient.Subscribe(channelID)
	if err != nil {
		return RenewalResult{
			ChannelID:    channelID,
			Success:      false,
			Message:      fmt.Sprintf("PubSubHubbub renewal failed: %v", err),
			AttemptCount: subscription.RenewalAttempts + 1,
		}
	}

	// Update subscription data
	subscription.LastRenewal = time.Now()
	subscription.ExpiresAt = time.Now().Add(time.Duration(getLeaseSeconds()) * time.Second)
	subscription.RenewalAttempts = 0

	return RenewalResult{
		ChannelID:     channelID,
		Success:       true,
		Message:       "Successfully renewed subscription",
		AttemptCount:  0,
		NewExpiryTime: subscription.ExpiresAt.Format(time.RFC3339),
	}
}

// handleRenewSubscriptionsRefactored is a compatibility wrapper that uses the refactored function.
func handleRenewSubscriptionsRefactored(w http.ResponseWriter, r *http.Request) {
	deps := GetDependencies()
	handler := handleRenewSubscriptionsWithDeps(deps)
	handler(w, r)
}

// handleNotificationWithDeps handles POST / requests (YouTube notifications) using dependency injection.
func handleNotificationWithDeps(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create notification service with injected dependencies
		notificationService := &NotificationServiceWithDeps{
			VideoProcessor: NewVideoProcessor(),
			GitHubClient:   deps.GitHubClient,
			RepoOwner:      os.Getenv("REPO_OWNER"),
			RepoName:       os.Getenv("REPO_NAME"),
		}

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
}

// NotificationServiceWithDeps is a version of NotificationService that uses dependency injection.
type NotificationServiceWithDeps struct {
	VideoProcessor *VideoProcessor
	GitHubClient   GitHubClientInterface
	RepoOwner      string
	RepoName       string
}

// ProcessNotification handles the complete notification processing workflow.
func (ns *NotificationServiceWithDeps) ProcessNotification(r *http.Request) (*NotificationResult, error) {
	// Parse the incoming XML notification
	entry, err := ns.parseNotification(r)
	if err != nil {
		// Map specific error messages to match original behavior
		var message string
		if err.Error() == "failed to read request body" {
			message = "Failed to read request body"
		} else if err.Error() == "invalid XML" {
			message = "Invalid XML"
		} else {
			message = err.Error()
		}
		return &NotificationResult{
			Status:  "error",
			Message: message,
		}, err
	}

	// Handle empty notifications
	if entry == nil {
		return &NotificationResult{
			Status:  "success",
			Message: "Empty notification (no entry found)",
		}, nil
	}

	// Check if it's a new video
	if !ns.VideoProcessor.IsNewVideo(entry) {
		return &NotificationResult{
			Status:  "success",
			Message: fmt.Sprintf("Skipped: Not a new video (VideoID: %s)", entry.VideoID),
		}, nil
	}

	// Check GitHub configuration
	if !ns.GitHubClient.IsConfigured() {
		return &NotificationResult{
			Status:  "success",
			Message: fmt.Sprintf("New video detected but GitHub token not configured (VideoID: %s)", entry.VideoID),
		}, nil
	}

	// Trigger GitHub workflow
	if err := ns.GitHubClient.TriggerWorkflow(ns.RepoOwner, ns.RepoName, entry); err != nil {
		return &NotificationResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to trigger GitHub workflow: %v", err),
		}, err
	}

	return &NotificationResult{
		Status:  "success",
		Message: fmt.Sprintf("Successfully triggered workflow for new video: %s", entry.VideoID),
	}, nil
}

// parseNotification parses the XML notification from the request body.
func (ns *NotificationServiceWithDeps) parseNotification(r *http.Request) (*Entry, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body")
	}

	var feed AtomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("invalid XML")
	}

	if feed.Entry == nil {
		return nil, nil
	}

	return feed.Entry, nil
}

// handleNotificationRefactored is a compatibility wrapper that uses the refactored function.
func handleNotificationRefactored(w http.ResponseWriter, r *http.Request) {
	deps := GetDependencies()
	handler := handleNotificationWithDeps(deps)
	handler(w, r)
}
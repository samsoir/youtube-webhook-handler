package webhook

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// YouTubeWebhook handles YouTube PubSubHubbub notifications and subscription management
// using dependency injection instead of global state
func YouTubeWebhook(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for all requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// Get dependencies for this request
	deps := GetDependencies()

	// Route based on path and method
	path := strings.TrimPrefix(r.URL.Path, "/")

	switch {
	case path == "subscribe" && r.Method == http.MethodPost:
		handler := handleSubscribe(deps)
		handler(w, r)
	case path == "unsubscribe" && r.Method == http.MethodDelete:
		handler := handleUnsubscribe(deps)
		handler(w, r)
	case path == "subscriptions" && r.Method == http.MethodGet:
		handler := handleGetSubscriptions(deps)
		handler(w, r)
	case path == "renew" && r.Method == http.MethodPost:
		handler := handleRenewSubscriptions(deps)
		handler(w, r)
	case r.Method == http.MethodGet:
		// Default GET behavior - YouTube verification challenge
		handleVerificationChallenge(w, r)
	case r.Method == http.MethodPost:
		// Default POST behavior - YouTube notifications
		handler := handleNotification(deps)
		handler(w, r)
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

// handleGetSubscriptions handles GET /subscriptions requests using dependency injection
func handleGetSubscriptions(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Load subscription state from injected storage client
		state, err := deps.StorageClient.LoadSubscriptionState(ctx)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "",
				fmt.Sprintf("Unable to load subscription state from storage: %v", err))
			return
		}

		// Calculate expiry status and statistics (same logic as original)
		now := getCurrentTime()
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
				ExpiresAt:       sub.ExpiresAt.Format(timeFormat()),
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
}


// Helper functions to make the code more testable by abstracting time and formats

// getCurrentTime returns the current time (can be mocked in tests)
func getCurrentTime() time.Time {
	return time.Now()
}

// timeFormat returns the time format to use (can be customized if needed)
func timeFormat() string {
	return time.RFC3339
}

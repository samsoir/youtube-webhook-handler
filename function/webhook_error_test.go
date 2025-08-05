package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestWebhookErrorPaths tests error handling paths in the main webhook functions
func TestWebhookErrorPaths(t *testing.T) {
	// Clear environment
	originalBucket := os.Getenv("SUBSCRIPTION_BUCKET")
	defer func() {
		if originalBucket == "" {
			os.Unsetenv("SUBSCRIPTION_BUCKET")
		} else {
			os.Setenv("SUBSCRIPTION_BUCKET", originalBucket)
		}
	}()

	t.Run("youtube_webhook_with_missing_bucket", func(t *testing.T) {
		// Clear bucket environment to trigger error
		os.Unsetenv("SUBSCRIPTION_BUCKET")
		
		// Create verification request
		req := httptest.NewRequest("GET", "/?hub.challenge=test&hub.mode=subscribe", nil)
		w := httptest.NewRecorder()
		
		// This should still work for verification challenges
		YouTubeWebhook(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "test", w.Body.String())
	})

	t.Run("youtube_webhook_notification_with_storage_error", func(t *testing.T) {
		// Clear bucket environment to trigger storage error
		os.Unsetenv("SUBSCRIPTION_BUCKET")
		
		// Create notification request
		xmlPayload := `<?xml version="1.0" encoding="UTF-8"?>
		<feed xmlns="http://www.w3.org/2005/Atom">
			<entry>
				<yt:videoId xmlns:yt="http://www.youtube.com/xml/schemas/2015">test_video_id</yt:videoId>
				<yt:channelId xmlns:yt="http://www.youtube.com/xml/schemas/2015">UCXuqSBlHAE6Xw-yeJA0Tunw</yt:channelId>
				<title>Test Video</title>
			</entry>
		</feed>`
		
		req := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(xmlPayload)))
		req.Header.Set("Content-Type", "application/atom+xml")
		w := httptest.NewRecorder()
		
		YouTubeWebhook(w, req)
		
		// Should return error due to missing storage configuration
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		// Check if response is JSON or plain text
		responseBody := w.Body.String()
		if len(responseBody) > 0 && responseBody[0] == '{' {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, "error", response["status"])
		} else {
			// Plain text error message
			assert.Contains(t, responseBody, "GitHub API error")
		}
	})

	t.Run("subscribe_with_storage_error", func(t *testing.T) {
		// Clear bucket environment to trigger storage error
		os.Unsetenv("SUBSCRIPTION_BUCKET")
		
		// Use query parameter instead of JSON body (matches actual API)
		req := httptest.NewRequest("POST", "/subscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw", nil)
		w := httptest.NewRecorder()
		
		handleSubscribe(w, req)
		
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "error", response["status"])
	})

	t.Run("unsubscribe_with_storage_error", func(t *testing.T) {
		// Clear bucket environment to trigger storage error
		os.Unsetenv("SUBSCRIPTION_BUCKET")
		
		// Use query parameter and DELETE method (matches actual API)
		req := httptest.NewRequest("DELETE", "/unsubscribe?channel_id=UCXuqSBlHAE6Xw-yeJA0Tunw", nil)
		w := httptest.NewRecorder()
		
		handleUnsubscribe(w, req)
		
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "error", response["status"])
	})

	t.Run("get_subscriptions_with_storage_error", func(t *testing.T) {
		// Clear bucket environment to trigger storage error
		os.Unsetenv("SUBSCRIPTION_BUCKET")
		
		req := httptest.NewRequest("GET", "/subscriptions", nil)
		w := httptest.NewRecorder()
		
		handleGetSubscriptions(w, req)
		
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "error", response["status"])
	})

	t.Run("load_subscription_state_error", func(t *testing.T) {
		// Clear bucket environment to trigger storage error
		os.Unsetenv("SUBSCRIPTION_BUCKET")
		
		// Use the storage client interface to test error path
		client := &CloudStorageClient{}
		ctx := context.Background()
		_, err := client.LoadSubscriptionState(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SUBSCRIPTION_BUCKET environment variable not set")
	})

	t.Run("save_subscription_state_error", func(t *testing.T) {
		// Clear bucket environment to trigger storage error
		os.Unsetenv("SUBSCRIPTION_BUCKET")
		
		state := &SubscriptionState{
			Subscriptions: make(map[string]*Subscription),
		}
		
		// Use the storage client interface to test error path
		client := &CloudStorageClient{}
		ctx := context.Background()
		err := client.SaveSubscriptionState(ctx, state)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SUBSCRIPTION_BUCKET environment variable not set")
	})

	t.Run("pubsubhubbub_request_with_missing_function_url", func(t *testing.T) {
		// This tests the error path in makePubSubHubbubRequest
		// Clear FUNCTION_URL to trigger error
		originalFunctionURL := os.Getenv("FUNCTION_URL")
		os.Unsetenv("FUNCTION_URL")
		defer func() {
			if originalFunctionURL != "" {
				os.Setenv("FUNCTION_URL", originalFunctionURL)
			}
		}()
		
		err := makePubSubHubbubRequest("UCXuqSBlHAE6Xw-yeJA0Tunw", "subscribe")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "FUNCTION_URL environment variable not set")
	})
}

// TestStorageServiceEdgeCases tests additional edge cases for storage service
func TestStorageServiceEdgeCases(t *testing.T) {
	t.Run("close_with_nil_client", func(t *testing.T) {
		service := NewOptimizedCloudStorageService()
		service.testMode = false
		service.client = nil // Explicitly set to nil
		
		err := service.Close()
		assert.NoError(t, err) // Should not error with nil client
	})

	t.Run("deep_copy_with_complex_state", func(t *testing.T) {
		service := NewOptimizedCloudStorageService()
		
		// Create a complex state with multiple subscriptions
		original := &SubscriptionState{
			Subscriptions: map[string]*Subscription{
				"channel1": {
					ChannelID: "channel1",
					Status:    "subscribed",
				},
				"channel2": {
					ChannelID: "channel2", 
					Status:    "unsubscribed",
				},
				"channel3": {
					ChannelID: "channel3",
					Status:    "failed",
				},
			},
			Metadata: struct {
				LastUpdated time.Time `json:"last_updated"`
				Version     string    `json:"version"`
			}{
				LastUpdated: time.Now(),
				Version:     "2.0",
			},
		}
		
		copy := service.deepCopyState(original)
		
		assert.NotNil(t, copy)
		assert.NotSame(t, original, copy)
		assert.Len(t, copy.Subscriptions, 3)
		
		// Verify all subscriptions are deep copied
		for key, originalSub := range original.Subscriptions {
			copySub, exists := copy.Subscriptions[key]
			assert.True(t, exists)
			assert.NotSame(t, originalSub, copySub)
			assert.Equal(t, originalSub.ChannelID, copySub.ChannelID)
			assert.Equal(t, originalSub.Status, copySub.Status)
		}
		
		// Modify original and verify copy is unchanged
		original.Subscriptions["channel1"].Status = "modified"
		assert.Equal(t, "subscribed", copy.Subscriptions["channel1"].Status)
	})
}
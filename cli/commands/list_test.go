package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	webhook "github.com/samsoir/youtube-webhook/function"
)

func TestList_Success(t *testing.T) {
	expectedResponse := webhook.SubscriptionsListResponse{
		Subscriptions: []webhook.SubscriptionInfo{
			{
				ChannelID:       "UCXuqSBlHAE6Xw-yeJA0Tunw",
				ExpiresAt:       "2024-01-22T15:30:00Z",
				Status:          "active",
				DaysUntilExpiry: 0.9,
			},
			{
				ChannelID:       "UCdQw4w9WgXcQ",
				ExpiresAt:       "2024-01-23T10:00:00Z",
				Status:          "active",
				DaysUntilExpiry: 1.4,
			},
			{
				ChannelID:       "UCabc123def456",
				ExpiresAt:       "2024-01-20T08:00:00Z",
				Status:          "expired",
				DaysUntilExpiry: -2.0,
			},
		},
		Total:   3,
		Active:  2,
		Expired: 1,
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		
		if r.URL.Path != "/subscriptions" {
			t.Errorf("Expected path /subscriptions, got %s", r.URL.Path)
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()
	
	config := ListConfig{
		BaseURL: server.URL,
		Timeout: 30 * time.Second,
		Format:  "table",
	}
	
	err := List(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestList_EmptyResponse(t *testing.T) {
	expectedResponse := webhook.SubscriptionsListResponse{
		Subscriptions: []webhook.SubscriptionInfo{},
		Total:         0,
		Active:        0,
		Expired:       0,
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()
	
	config := ListConfig{
		BaseURL: server.URL,
		Timeout: 30 * time.Second,
		Format:  "table",
	}
	
	err := List(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestList_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(webhook.APIResponse{
			Status:  "error",
			Message: "Internal server error",
		})
	}))
	defer server.Close()
	
	config := ListConfig{
		BaseURL: server.URL,
		Timeout: 30 * time.Second,
		Format:  "table",
	}
	
	err := List(config)
	if err == nil {
		t.Fatal("Expected error for server error, got nil")
	}
}
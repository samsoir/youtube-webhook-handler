package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	webhook "github.com/samsoir/youtube-webhook/function"
)

func TestNewClient(t *testing.T) {
	baseURL := "https://example.com/"
	timeout := 30 * time.Second
	
	client := NewClient(baseURL, timeout)
	
	if client.baseURL != "https://example.com" {
		t.Errorf("Expected baseURL to be 'https://example.com', got %s", client.baseURL)
	}
	
	if client.httpClient.Timeout != timeout {
		t.Errorf("Expected timeout to be %v, got %v", timeout, client.httpClient.Timeout)
	}
}

func TestClient_Subscribe_Success(t *testing.T) {
	expectedResponse := webhook.APIResponse{
		Status:    "success",
		Message:   "Subscribed successfully",
		ExpiresAt: "2024-01-22T15:30:00Z",
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		
		if r.URL.Path != "/subscribe" {
			t.Errorf("Expected path /subscribe, got %s", r.URL.Path)
		}
		
		channelID := r.URL.Query().Get("channel_id")
		if channelID != "UCXuqSBlHAE6Xw-yeJA0Tunw" {
			t.Errorf("Expected channel_id UCXuqSBlHAE6Xw-yeJA0Tunw, got %s", channelID)
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()
	
	client := NewClient(server.URL, 30*time.Second)
	resp, err := client.Subscribe("UCXuqSBlHAE6Xw-yeJA0Tunw")
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if resp.Status != expectedResponse.Status {
		t.Errorf("Expected status %s, got %s", expectedResponse.Status, resp.Status)
	}
	
	if resp.Message != expectedResponse.Message {
		t.Errorf("Expected message %s, got %s", expectedResponse.Message, resp.Message)
	}
	
	if resp.ExpiresAt != expectedResponse.ExpiresAt {
		t.Errorf("Expected expires_at %s, got %s", expectedResponse.ExpiresAt, resp.ExpiresAt)
	}
}

func TestClient_Subscribe_Conflict(t *testing.T) {
	conflictResponse := webhook.APIResponse{
		Status:    "conflict",
		Message:   "Already subscribed",
		ExpiresAt: "2024-01-22T15:30:00Z",
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(conflictResponse)
	}))
	defer server.Close()
	
	client := NewClient(server.URL, 30*time.Second)
	resp, err := client.Subscribe("UCXuqSBlHAE6Xw-yeJA0Tunw")
	
	if err == nil {
		t.Fatal("Expected error for conflict response, got nil")
	}
	
	if resp == nil {
		t.Fatal("Expected response to be returned even with error")
	}
	
	if resp.Status != "conflict" {
		t.Errorf("Expected status conflict, got %s", resp.Status)
	}
}

func TestClient_Subscribe_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(webhook.APIResponse{
			Status:  "error",
			Message: "Internal server error",
		})
	}))
	defer server.Close()
	
	client := NewClient(server.URL, 30*time.Second)
	_, err := client.Subscribe("UCXuqSBlHAE6Xw-yeJA0Tunw")
	
	if err == nil {
		t.Fatal("Expected error for server error response, got nil")
	}
	
	expectedError := "server error (500): Internal server error"
	if err.Error() != expectedError {
		t.Errorf("Expected error %s, got %s", expectedError, err.Error())
	}
}

func TestClient_Unsubscribe_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		
		if r.URL.Path != "/unsubscribe" {
			t.Errorf("Expected path /unsubscribe, got %s", r.URL.Path)
		}
		
		channelID := r.URL.Query().Get("channel_id")
		if channelID != "UCXuqSBlHAE6Xw-yeJA0Tunw" {
			t.Errorf("Expected channel_id UCXuqSBlHAE6Xw-yeJA0Tunw, got %s", channelID)
		}
		
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	
	client := NewClient(server.URL, 30*time.Second)
	err := client.Unsubscribe("UCXuqSBlHAE6Xw-yeJA0Tunw")
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestClient_Unsubscribe_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	
	client := NewClient(server.URL, 30*time.Second)
	err := client.Unsubscribe("UCXuqSBlHAE6Xw-yeJA0Tunw")
	
	if err == nil {
		t.Fatal("Expected error for not found response, got nil")
	}
	
	expectedError := "not subscribed to channel UCXuqSBlHAE6Xw-yeJA0Tunw"
	if err.Error() != expectedError {
		t.Errorf("Expected error %s, got %s", expectedError, err.Error())
	}
}

func TestClient_ListSubscriptions_Success(t *testing.T) {
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
		},
		Total:   2,
		Active:  2,
		Expired: 0,
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
	
	client := NewClient(server.URL, 30*time.Second)
	resp, err := client.ListSubscriptions()
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(resp.Subscriptions) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(resp.Subscriptions))
	}
	
	if resp.Total != 2 {
		t.Errorf("Expected total 2, got %d", resp.Total)
	}
	
	if resp.Active != 2 {
		t.Errorf("Expected active 2, got %d", resp.Active)
	}
}

func TestClient_RenewSubscriptions_Success(t *testing.T) {
	expectedResponse := webhook.RenewalSummaryResponse{
		Status:             "success",
		TotalChecked:       3,
		RenewalsCandidates: 1,
		RenewalsSucceeded:  1,
		RenewalsFailed:     0,
		Results: []webhook.RenewalResult{
			{
				ChannelID:     "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Success:       true,
				Message:       "Renewed successfully",
				AttemptCount:  1,
			},
		},
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		
		if r.URL.Path != "/renew" {
			t.Errorf("Expected path /renew, got %s", r.URL.Path)
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()
	
	client := NewClient(server.URL, 30*time.Second)
	resp, err := client.RenewSubscriptions()
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if resp.TotalChecked != 3 {
		t.Errorf("Expected checked 3, got %d", resp.TotalChecked)
	}
	
	if resp.RenewalsSucceeded != 1 {
		t.Errorf("Expected renewed 1, got %d", resp.RenewalsSucceeded)
	}
	
	if len(resp.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(resp.Results))
	}
}

func TestClient_RenewSubscriptions_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(webhook.APIResponse{
			Status:  "error",
			Message: "Renewal failed",
		})
	}))
	defer server.Close()
	
	client := NewClient(server.URL, 30*time.Second)
	_, err := client.RenewSubscriptions()
	
	if err == nil {
		t.Fatal("Expected error for server error response, got nil")
	}
	
	expectedError := "server error (500): Renewal failed"
	if err.Error() != expectedError {
		t.Errorf("Expected error %s, got %s", expectedError, err.Error())
	}
}
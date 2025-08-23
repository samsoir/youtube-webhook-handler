package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	webhook "github.com/samsoir/youtube-webhook/function"
)

func TestSubscribe_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		
		channelID := r.URL.Query().Get("channel_id")
		if channelID != "UCXuqSBlHAE6Xw-yeJA0Tunw" {
			t.Errorf("Expected channel_id UCXuqSBlHAE6Xw-yeJA0Tunw, got %s", channelID)
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(webhook.APIResponse{
			Status:    "success",
			Message:   "Subscribed successfully",
			ExpiresAt: "2024-01-22T15:30:00Z",
		})
	}))
	defer server.Close()
	
	config := SubscribeConfig{
		BaseURL:   server.URL,
		ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
		Timeout:   30 * time.Second,
	}
	
	err := Subscribe(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestSubscribe_AlreadySubscribed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(webhook.APIResponse{
			Status:    "conflict",
			Message:   "Already subscribed",
			ExpiresAt: "2024-01-22T15:30:00Z",
		})
	}))
	defer server.Close()
	
	config := SubscribeConfig{
		BaseURL:   server.URL,
		ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
		Timeout:   30 * time.Second,
	}
	
	// Should not return an error for conflict (already subscribed)
	err := Subscribe(config)
	if err != nil {
		t.Fatalf("Expected no error for already subscribed case, got %v", err)
	}
}

func TestSubscribe_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(webhook.APIResponse{
			Status:  "error",
			Message: "Internal server error",
		})
	}))
	defer server.Close()
	
	config := SubscribeConfig{
		BaseURL:   server.URL,
		ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
		Timeout:   30 * time.Second,
	}
	
	err := Subscribe(config)
	if err == nil {
		t.Fatal("Expected error for server error, got nil")
	}
}

func TestUnsubscribe_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		
		channelID := r.URL.Query().Get("channel_id")
		if channelID != "UCXuqSBlHAE6Xw-yeJA0Tunw" {
			t.Errorf("Expected channel_id UCXuqSBlHAE6Xw-yeJA0Tunw, got %s", channelID)
		}
		
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	
	config := UnsubscribeConfig{
		BaseURL:   server.URL,
		ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
		Timeout:   30 * time.Second,
	}
	
	err := Unsubscribe(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestUnsubscribe_NotSubscribed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	
	config := UnsubscribeConfig{
		BaseURL:   server.URL,
		ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
		Timeout:   30 * time.Second,
	}
	
	// Should not return an error for not found (not subscribed)
	err := Unsubscribe(config)
	if err != nil {
		t.Fatalf("Expected no error for not subscribed case, got %v", err)
	}
}

func TestUnsubscribe_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(webhook.APIResponse{
			Status:  "error",
			Message: "Internal server error",
		})
	}))
	defer server.Close()
	
	config := UnsubscribeConfig{
		BaseURL:   server.URL,
		ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
		Timeout:   30 * time.Second,
	}
	
	err := Unsubscribe(config)
	if err == nil {
		t.Fatal("Expected error for server error, got nil")
	}
}
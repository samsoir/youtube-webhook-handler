package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	webhook "github.com/samsoir/youtube-webhook/function"
)

func TestRenew_Success(t *testing.T) {
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
	
	config := RenewConfig{
		BaseURL: server.URL,
		Timeout: 60 * time.Second,
		Verbose: false,
	}
	
	err := Renew(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestRenew_SuccessVerbose(t *testing.T) {
	expectedResponse := webhook.RenewalSummaryResponse{
		Status:             "success",
		TotalChecked:       3,
		RenewalsCandidates: 3,
		RenewalsSucceeded:  2,
		RenewalsFailed:     1,
		Results: []webhook.RenewalResult{
			{
				ChannelID:     "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Success:       true,
				Message:       "Renewed successfully",
				AttemptCount:  1,
			},
			{
				ChannelID:     "UCdQw4w9WgXcQ",
				Success:       true,
				Message:       "Renewed successfully",
				AttemptCount:  1,
			},
			{
				ChannelID:     "UCabc123def456",
				Success:       false,
				Message:       "Hub returned error 404",
				AttemptCount:  1,
			},
		},
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()
	
	config := RenewConfig{
		BaseURL: server.URL,
		Timeout: 60 * time.Second,
		Verbose: true,
	}
	
	err := Renew(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestRenew_NoRenewalsNeeded(t *testing.T) {
	expectedResponse := webhook.RenewalSummaryResponse{
		Status:             "success",
		TotalChecked:       3,
		RenewalsCandidates: 0,
		RenewalsSucceeded:  0,
		RenewalsFailed:     0,
		Results:            []webhook.RenewalResult{},
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()
	
	config := RenewConfig{
		BaseURL: server.URL,
		Timeout: 60 * time.Second,
		Verbose: false,
	}
	
	err := Renew(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestRenew_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(webhook.APIResponse{
			Status:  "error",
			Message: "Renewal service unavailable",
		})
	}))
	defer server.Close()
	
	config := RenewConfig{
		BaseURL: server.URL,
		Timeout: 60 * time.Second,
		Verbose: false,
	}
	
	err := Renew(config)
	if err == nil {
		t.Fatal("Expected error for server error, got nil")
	}
}
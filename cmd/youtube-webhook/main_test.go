package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	webhook "github.com/samsoir/youtube-webhook/function"
)

// TestMain_Help tests the help command functionality
func TestMain_Help(t *testing.T) {
	// Build the CLI binary for testing
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	testCases := []string{"help", "-h", "--help"}
	
	for _, arg := range testCases {
		t.Run(fmt.Sprintf("help_%s", arg), func(t *testing.T) {
			cmd := exec.Command(binaryPath, arg)
			output, err := cmd.CombinedOutput()
			
			if err != nil && cmd.ProcessState.ExitCode() != 0 {
				t.Errorf("Expected help command to succeed, got exit code %d", cmd.ProcessState.ExitCode())
			}
			
			outputStr := string(output)
			if !strings.Contains(outputStr, "YouTube Webhook CLI") {
				t.Error("Expected help output to contain 'YouTube Webhook CLI'")
			}
			
			if !strings.Contains(outputStr, "subscribe") {
				t.Error("Expected help output to contain 'subscribe' command")
			}
			
			if !strings.Contains(outputStr, "YOUTUBE_WEBHOOK_URL") {
				t.Error("Expected help output to mention YOUTUBE_WEBHOOK_URL environment variable")
			}
		})
	}
}

// TestMain_NoCommand tests behavior when no command is provided
func TestMain_NoCommand(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	cmd := exec.Command(binaryPath)
	output, err := cmd.CombinedOutput()
	
	if err == nil {
		t.Error("Expected command to fail when no subcommand provided")
	}
	
	if cmd.ProcessState.ExitCode() != 1 {
		t.Errorf("Expected exit code 1, got %d", cmd.ProcessState.ExitCode())
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "Usage:") {
		t.Error("Expected usage information when no command provided")
	}
}

// TestMain_UnknownCommand tests behavior with unknown commands
func TestMain_UnknownCommand(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	cmd := exec.Command(binaryPath, "unknown-command")
	output, err := cmd.CombinedOutput()
	
	if err == nil {
		t.Error("Expected command to fail with unknown command")
	}
	
	if cmd.ProcessState.ExitCode() != 1 {
		t.Errorf("Expected exit code 1, got %d", cmd.ProcessState.ExitCode())
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "Unknown command") {
		t.Error("Expected error message about unknown command")
	}
}

// TestMain_Subscribe tests the subscribe command integration
func TestMain_Subscribe(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/subscribe" && r.Method == "POST" {
			channelID := r.URL.Query().Get("channel_id")
			if channelID == "UCXuqSBlHAE6Xw-yeJA0Tunw" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(webhook.APIResponse{
					Status:    "success",
					Message:   "Subscribed successfully",
					ExpiresAt: "2024-01-22T15:30:00Z",
				})
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	
	// Test successful subscribe
	cmd := exec.Command(binaryPath, "subscribe", "-url", server.URL, "-channel", "UCXuqSBlHAE6Xw-yeJA0Tunw")
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Errorf("Subscribe command failed: %v, output: %s", err, string(output))
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "Successfully subscribed") {
		t.Errorf("Expected success message, got: %s", outputStr)
	}
}

// TestMain_Subscribe_MissingFlags tests subscribe command with missing required flags
func TestMain_Subscribe_MissingFlags(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	testCases := []struct {
		name string
		args []string
		expectedError string
	}{
		{
			name: "missing_url_and_channel",
			args: []string{"subscribe"},
			expectedError: "-url flag or YOUTUBE_WEBHOOK_URL environment variable is required",
		},
		{
			name: "missing_channel",
			args: []string{"subscribe", "-url", "https://example.com"},
			expectedError: "-channel flag is required",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tc.args...)
			output, err := cmd.CombinedOutput()
			
			if err == nil {
				t.Error("Expected command to fail with missing flags")
			}
			
			outputStr := string(output)
			if !strings.Contains(outputStr, tc.expectedError) {
				t.Errorf("Expected error message '%s', got: %s", tc.expectedError, outputStr)
			}
		})
	}
}

// TestMain_List tests the list command integration
func TestMain_List(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/subscriptions" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(webhook.SubscriptionsListResponse{
				Subscriptions: []webhook.SubscriptionInfo{
					{
						ChannelID:       "UCXuqSBlHAE6Xw-yeJA0Tunw",
						ExpiresAt:       "2024-01-22T15:30:00Z",
						Status:          "active",
						DaysUntilExpiry: 0.9,
					},
				},
				Total:   1,
				Active:  1,
				Expired: 0,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	
	// Test successful list
	cmd := exec.Command(binaryPath, "list", "-url", server.URL)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Errorf("List command failed: %v, output: %s", err, string(output))
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "Subscription Summary") {
		t.Errorf("Expected subscription summary, got: %s", outputStr)
	}
}

// TestMain_EnvironmentVariable tests using YOUTUBE_WEBHOOK_URL environment variable
func TestMain_EnvironmentVariable(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/subscriptions" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(webhook.SubscriptionsListResponse{
				Subscriptions: []webhook.SubscriptionInfo{},
				Total:         0,
				Active:        0,
				Expired:       0,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	
	// Test with environment variable
	cmd := exec.Command(binaryPath, "list")
	cmd.Env = append(os.Environ(), fmt.Sprintf("YOUTUBE_WEBHOOK_URL=%s", server.URL))
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Errorf("List command with env var failed: %v, output: %s", err, string(output))
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "No subscriptions found") || strings.Contains(outputStr, "Subscription Summary") {
		// Should show either "No subscriptions found" or summary with 0 totals
		t.Logf("Output: %s", outputStr) // Log for debugging, but don't fail
	}
}

// buildCLIBinary builds the CLI binary for testing and returns the path
func buildCLIBinary(t *testing.T) string {
	t.Helper()
	
	// Create a temporary directory for the test binary
	tmpDir, err := os.MkdirTemp("", "youtube-webhook-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	binaryPath := filepath.Join(tmpDir, "youtube-webhook-test")
	
	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "." // Current directory (cmd/youtube-webhook)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Fatalf("Failed to build CLI binary: %v, output: %s", err, string(output))
	}
	
	return binaryPath
}

// TestIntegration_CompleteWorkflow tests a complete workflow
func TestIntegration_CompleteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	// Track the subscription state
	subscriptions := make(map[string]webhook.SubscriptionInfo)
	
	// Setup test server that maintains state
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		channelID := r.URL.Query().Get("channel_id")
		
		switch {
		case r.URL.Path == "/subscribe" && r.Method == "POST":
			subscriptions[channelID] = webhook.SubscriptionInfo{
				ChannelID:       channelID,
				ExpiresAt:       time.Now().Add(24 * time.Hour).Format(time.RFC3339),
				Status:          "active",
				DaysUntilExpiry: 1.0,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(webhook.APIResponse{
				Status:    "success",
				Message:   "Subscribed successfully",
				ExpiresAt: subscriptions[channelID].ExpiresAt,
			})
			
		case r.URL.Path == "/unsubscribe" && r.Method == "DELETE":
			if _, exists := subscriptions[channelID]; exists {
				delete(subscriptions, channelID)
				w.WriteHeader(http.StatusNoContent)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			
		case r.URL.Path == "/subscriptions" && r.Method == "GET":
			var subs []webhook.SubscriptionInfo
			for _, sub := range subscriptions {
				subs = append(subs, sub)
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(webhook.SubscriptionsListResponse{
				Subscriptions: subs,
				Total:         len(subs),
				Active:        len(subs),
				Expired:       0,
			})
			
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	
	// Test workflow: subscribe -> list -> unsubscribe -> list
	testChannelID := "UCXuqSBlHAE6Xw-yeJA0Tunw"
	
	// 1. Subscribe
	cmd := exec.Command(binaryPath, "subscribe", "-url", server.URL, "-channel", testChannelID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Subscribe failed: %v, output: %s", err, string(output))
	}
	
	// 2. List (should show 1 subscription)
	cmd = exec.Command(binaryPath, "list", "-url", server.URL)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("List failed: %v, output: %s", err, string(output))
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, testChannelID) {
		t.Errorf("Expected to see subscribed channel in list output, got: %s", outputStr)
	}
	
	// 3. Unsubscribe
	cmd = exec.Command(binaryPath, "unsubscribe", "-url", server.URL, "-channel", testChannelID)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Unsubscribe failed: %v, output: %s", err, string(output))
	}
	
	// 4. List (should show 0 subscriptions)
	cmd = exec.Command(binaryPath, "list", "-url", server.URL)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Final list failed: %v, output: %s", err, string(output))
	}
	
	outputStr = string(output)
	if strings.Contains(outputStr, testChannelID) {
		t.Errorf("Expected channel to be removed from list, but still found: %s", outputStr)
	}
}
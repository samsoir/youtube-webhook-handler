package webhook

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPPubSubClient(t *testing.T) {
	// Test with FUNCTION_URL set
	expectedURL := "https://test-function.cloudfunctions.net"
	os.Setenv("FUNCTION_URL", expectedURL)
	defer os.Unsetenv("FUNCTION_URL")

	client := NewHTTPPubSubClient()

	if client == nil {
		t.Fatal("NewHTTPPubSubClient returned nil")
	}

	if client.callbackURL != expectedURL {
		t.Errorf("Expected callbackURL %s, got %s", expectedURL, client.callbackURL)
	}

	if client.hubURL != "https://pubsubhubbub.appspot.com/subscribe" {
		t.Errorf("Expected hubURL https://pubsubhubbub.appspot.com/subscribe, got %s", client.hubURL)
	}

	if client.client == nil {
		t.Error("HTTP client is nil")
	}

	if client.client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.client.Timeout)
	}
}

func TestNewHTTPPubSubClient_DefaultURL(t *testing.T) {
	// Test without FUNCTION_URL set
	os.Unsetenv("FUNCTION_URL")

	client := NewHTTPPubSubClient()

	if client == nil {
		t.Fatal("NewHTTPPubSubClient returned nil")
	}

	if client.callbackURL != "https://default-function-url" {
		t.Errorf("Expected default callbackURL, got %s", client.callbackURL)
	}
}

func TestHTTPPubSubClient_Subscribe_Success(t *testing.T) {
	// Create a test server that returns success
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("Failed to parse form: %v", err)
			return
		}

		// Check required form fields
		if r.FormValue("hub.mode") != "subscribe" {
			t.Errorf("Expected hub.mode=subscribe, got %s", r.FormValue("hub.mode"))
		}

		if r.FormValue("hub.verify") != "async" {
			t.Errorf("Expected hub.verify=async, got %s", r.FormValue("hub.verify"))
		}

		if r.FormValue("hub.lease_seconds") != "86400" {
			t.Errorf("Expected hub.lease_seconds=86400, got %s", r.FormValue("hub.lease_seconds"))
		}

		expectedTopic := "https://www.youtube.com/feeds/videos.xml?channel_id=UC123"
		if r.FormValue("hub.topic") != expectedTopic {
			t.Errorf("Expected hub.topic=%s, got %s", expectedTopic, r.FormValue("hub.topic"))
		}

		// Return success
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with test server
	client := &HTTPPubSubClient{
		hubURL:      server.URL,
		callbackURL: "https://test-callback.com",
		client:      &http.Client{Timeout: 30 * time.Second},
	}

	err := client.Subscribe("UC123")
	if err != nil {
		t.Errorf("Subscribe failed: %v", err)
	}
}

func TestHTTPPubSubClient_Unsubscribe_Success(t *testing.T) {
	// Create a test server that returns success
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("Failed to parse form: %v", err)
			return
		}

		// Check that mode is unsubscribe
		if r.FormValue("hub.mode") != "unsubscribe" {
			t.Errorf("Expected hub.mode=unsubscribe, got %s", r.FormValue("hub.mode"))
		}

		// Return success
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with test server
	client := &HTTPPubSubClient{
		hubURL:      server.URL,
		callbackURL: "https://test-callback.com",
		client:      &http.Client{Timeout: 30 * time.Second},
	}

	err := client.Unsubscribe("UC456")
	if err != nil {
		t.Errorf("Unsubscribe failed: %v", err)
	}
}

func TestHTTPPubSubClient_Subscribe_HTTPError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := &HTTPPubSubClient{
		hubURL:      server.URL,
		callbackURL: "https://test-callback.com",
		client:      &http.Client{Timeout: 30 * time.Second},
	}

	err := client.Subscribe("UC123")
	if err == nil {
		t.Error("Expected error for HTTP 400 response")
	}

	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Expected error to contain status code 400, got: %v", err)
	}
}

func TestHTTPPubSubClient_Subscribe_NetworkError(t *testing.T) {
	// Use an invalid URL to trigger network error
	client := &HTTPPubSubClient{
		hubURL:      "http://invalid-url-that-does-not-exist.test",
		callbackURL: "https://test-callback.com",
		client:      &http.Client{Timeout: 1 * time.Second},
	}

	err := client.Subscribe("UC123")
	if err == nil {
		t.Error("Expected network error")
	}

	if !strings.Contains(err.Error(), "failed to make PubSubHubbub request") {
		t.Errorf("Expected network error message, got: %v", err)
	}
}

func TestHTTPPubSubClient_Unsubscribe_HTTPError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &HTTPPubSubClient{
		hubURL:      server.URL,
		callbackURL: "https://test-callback.com",
		client:      &http.Client{Timeout: 30 * time.Second},
	}

	err := client.Unsubscribe("UC456")
	if err == nil {
		t.Error("Expected error for HTTP 500 response")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Expected error to contain status code 500, got: %v", err)
	}
}

func TestHTTPPubSubClient_makePubSubHubbubRequest_StatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		expectError bool
	}{
		{"Success 200", http.StatusOK, false},
		{"Success 202", http.StatusAccepted, false},
		{"Client Error 400", http.StatusBadRequest, true},
		{"Client Error 404", http.StatusNotFound, true},
		{"Server Error 500", http.StatusInternalServerError, true},
		{"Redirection 301", http.StatusMovedPermanently, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			client := &HTTPPubSubClient{
				hubURL:      server.URL,
				callbackURL: "https://test-callback.com",
				client:      &http.Client{Timeout: 30 * time.Second},
			}

			err := client.makePubSubHubbubRequest("UC123", "subscribe")

			if tc.expectError && err == nil {
				t.Errorf("Expected error for status code %d", tc.statusCode)
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for status code %d: %v", tc.statusCode, err)
			}

			if tc.expectError && err != nil {
				expectedMsg := fmt.Sprintf("%d", tc.statusCode)
				if !strings.Contains(err.Error(), expectedMsg) {
					t.Errorf("Error message should contain status code %d, got: %v", tc.statusCode, err)
				}
			}
		})
	}
}

func TestHTTPPubSubClient_RequestFormat(t *testing.T) {
	var capturedForm map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse and capture form data immediately
		if err := r.ParseForm(); err != nil {
			t.Errorf("Failed to parse form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		capturedForm = make(map[string]string)
		for key, values := range r.Form {
			if len(values) > 0 {
				capturedForm[key] = values[0]
			}
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &HTTPPubSubClient{
		hubURL:      server.URL,
		callbackURL: "https://my-function.cloudfunctions.net/webhook",
		client:      &http.Client{Timeout: 30 * time.Second},
	}

	channelID := "UCaBcd123"
	err := client.Subscribe(channelID)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Verify captured form data
	if capturedForm == nil {
		t.Fatal("No form data was captured")
	}

	// Verify all form fields
	expectedFields := map[string]string{
		"hub.callback":      "https://my-function.cloudfunctions.net/webhook",
		"hub.topic":         fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelID),
		"hub.mode":          "subscribe",
		"hub.verify":        "async",
		"hub.lease_seconds": "86400",
	}

	for field, expectedValue := range expectedFields {
		actualValue := capturedForm[field]
		if actualValue != expectedValue {
			t.Errorf("Field %s: expected %s, got %s", field, expectedValue, actualValue)
		}
	}
}
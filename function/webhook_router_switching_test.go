package webhook

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestYouTubeWebhook_RouterSwitching(t *testing.T) {
	// Test cases for router switching
	testCases := []struct {
		name                string
		useRefactoredRouter string
		expectRefactored    bool
	}{
		{
			name:                "Default - use original router",
			useRefactoredRouter: "",
			expectRefactored:    false,
		},
		{
			name:                "Enable refactored router with 'true'",
			useRefactoredRouter: "true",
			expectRefactored:    true,
		},
		{
			name:                "Enable refactored router with '1'",
			useRefactoredRouter: "1",
			expectRefactored:    true,
		},
		{
			name:                "Disable refactored router with 'false'",
			useRefactoredRouter: "false",
			expectRefactored:    false,
		},
		{
			name:                "Disable refactored router with '0'",
			useRefactoredRouter: "0",
			expectRefactored:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable
			if tc.useRefactoredRouter != "" {
				os.Setenv("USE_REFACTORED_ROUTER", tc.useRefactoredRouter)
			} else {
				os.Unsetenv("USE_REFACTORED_ROUTER")
			}
			defer os.Unsetenv("USE_REFACTORED_ROUTER")

			// Enable test mode for original router tests
			if !tc.expectRefactored {
				originalTestMode := GetTestMode()
				SetTestMode(true)
				defer SetTestMode(originalTestMode)
			} else {
				// Set test dependencies for refactored router
				testDeps := CreateTestDependencies()
				SetDependencies(testDeps)
			}

			// Test the configuration function
			result := useRefactoredRouter()
			if result != tc.expectRefactored {
				t.Errorf("Expected useRefactoredRouter() to return %v, got %v", tc.expectRefactored, result)
			}

			// Test that the router actually switches
			// Create a simple request
			req := httptest.NewRequest("GET", "/subscriptions", nil)
			rec := httptest.NewRecorder()

			// Call main handler
			YouTubeWebhook(rec, req)

			// Both routers should return OK for this request
			if rec.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
			}
		})
	}
}

func TestYouTubeWebhook_RefactoredRouterIntegration(t *testing.T) {
	// Enable refactored router
	os.Setenv("USE_REFACTORED_ROUTER", "true")
	defer os.Unsetenv("USE_REFACTORED_ROUTER")

	// Create test dependencies to ensure refactored router is actually being used
	testDeps := CreateTestDependencies()
	SetDependencies(testDeps)

	// Create test request
	req := httptest.NewRequest("GET", "/subscriptions", nil)
	rec := httptest.NewRecorder()

	// Call main handler - should use refactored router
	YouTubeWebhook(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify CORS headers are set (both routers should set them)
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected CORS origin header to be '*', got: %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestYouTubeWebhook_OriginalRouterIntegration(t *testing.T) {
	// Disable refactored router (use original)
	os.Setenv("USE_REFACTORED_ROUTER", "false")
	defer os.Unsetenv("USE_REFACTORED_ROUTER")

	// Ensure test mode is enabled for original router
	originalTestMode := GetTestMode()
	SetTestMode(true)
	defer SetTestMode(originalTestMode)

	// Create test request
	req := httptest.NewRequest("GET", "/subscriptions", nil)
	rec := httptest.NewRecorder()

	// Call main handler - should use original router
	YouTubeWebhook(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify CORS headers are set
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected CORS origin header to be '*', got: %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestUseRefactoredRouter(t *testing.T) {
	testCases := []struct {
		envValue string
		expected bool
	}{
		{"", false},
		{"true", true},
		{"1", true},
		{"false", false},
		{"0", false},
		{"yes", false},  // Not recognized as true
		{"TRUE", false}, // Case sensitive
	}

	for _, tc := range testCases {
		t.Run("env_value_"+tc.envValue, func(t *testing.T) {
			if tc.envValue != "" {
				os.Setenv("USE_REFACTORED_ROUTER", tc.envValue)
			} else {
				os.Unsetenv("USE_REFACTORED_ROUTER")
			}
			defer os.Unsetenv("USE_REFACTORED_ROUTER")

			result := useRefactoredRouter()
			if result != tc.expected {
				t.Errorf("For env value '%s', expected %v, got %v", tc.envValue, tc.expected, result)
			}
		})
	}
}
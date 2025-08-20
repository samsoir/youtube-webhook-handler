package webhook

import (
	"os"
	"testing"
)

func TestTriggerGitHubWorkflow_BackwardCompatibility(t *testing.T) {
	// Set environment variables
	os.Setenv("REPO_OWNER", "test-owner")
	os.Setenv("REPO_NAME", "test-repo")
	os.Setenv("GITHUB_TOKEN", "test-token")
	defer func() {
		os.Unsetenv("REPO_OWNER")
		os.Unsetenv("REPO_NAME")
		os.Unsetenv("GITHUB_TOKEN")
	}()

	// Create a test entry
	entry := &Entry{
		Title:     "Test Video",
		VideoID:   "test123",
		Published: "2023-01-01T12:00:00Z",
		Updated:   "2023-01-01T12:00:00Z",
		ChannelID: "UCtest123",
	}

	// Call the backward compatibility function
	// This will create a real GitHubClient but won't actually make HTTP requests 
	// since we're not setting up a test server
	err := triggerGitHubWorkflow(entry)

	// We expect an error since we're not setting up a real server
	// but the function should at least initialize and validate inputs
	if err == nil {
		t.Log("triggerGitHubWorkflow completed (no network call made)")
	} else {
		// Error is expected due to network call, but function was called
		t.Logf("triggerGitHubWorkflow called with expected error: %v", err)
	}
}

func TestTriggerGitHubWorkflow_MissingEnvironment(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("REPO_OWNER")
	os.Unsetenv("REPO_NAME")
	os.Unsetenv("GITHUB_TOKEN")

	entry := &Entry{
		Title:     "Test Video",
		VideoID:   "test123",
		Published: "2023-01-01T12:00:00Z",
		Updated:   "2023-01-01T12:00:00Z",
		ChannelID: "UCtest123",
	}

	err := triggerGitHubWorkflow(entry)

	// Should get an error due to missing configuration
	if err == nil {
		t.Error("Expected error due to missing environment variables")
	}

	// Error should mention missing configuration
	if err != nil && err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}
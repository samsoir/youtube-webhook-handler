package webhook

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
)

// NotificationService handles YouTube webhook notifications
type NotificationService struct {
	VideoProcessor *VideoProcessor
	GitHubClient   *GitHubClient
	RepoOwner      string
	RepoName       string
}

// NewNotificationService creates a new notification service with dependencies
func NewNotificationService() *NotificationService {
	return &NotificationService{
		VideoProcessor: NewVideoProcessor(),
		GitHubClient:   NewGitHubClient(),
		RepoOwner:      os.Getenv("REPO_OWNER"),
		RepoName:       os.Getenv("REPO_NAME"),
	}
}

// ProcessNotification handles the complete notification processing workflow
func (ns *NotificationService) ProcessNotification(r *http.Request) (*NotificationResult, error) {
	// Parse the incoming XML notification
	entry, err := ns.parseNotification(r)
	if err != nil {
		// Map specific error messages to match original behavior
		var message string
		if err.Error() == "failed to read request body" {
			message = "Failed to read request body"
		} else if err.Error() == "invalid XML" {
			message = "Invalid XML"
		} else {
			message = err.Error()
		}
		return &NotificationResult{
			Status:  "error",
			Message: message,
		}, err
	}

	// Handle empty notifications
	if entry == nil {
		return &NotificationResult{
			Status:  "success",
			Message: "No video data",
		}, nil
	}

	// Validate entry data
	if err := ns.VideoProcessor.ValidateEntry(entry); err != nil {
		return &NotificationResult{
			Status:  "error",
			Message: fmt.Sprintf("Invalid entry: %v", err),
		}, err
	}

	// Check if this is a new video
	if !ns.VideoProcessor.IsNewVideo(entry) {
		return &NotificationResult{
			Status:  "success",
			Message: "Video update ignored",
		}, nil
	}

	// Trigger GitHub workflow if configured
	if ns.GitHubClient.IsConfigured() && ns.RepoOwner != "" && ns.RepoName != "" {
		if err := ns.GitHubClient.TriggerWorkflow(ns.RepoOwner, ns.RepoName, entry); err != nil {
			return &NotificationResult{
				Status:  "error",
				Message: "GitHub API error",
			}, fmt.Errorf("failed to trigger GitHub workflow: %v", err)
		}
	} else if !ns.GitHubClient.IsConfigured() || ns.RepoOwner == "" || ns.RepoName == "" {
		// If GitHub is not configured, return error like the original implementation
		return &NotificationResult{
			Status:  "error",
			Message: "GitHub API error",
		}, fmt.Errorf("missing required environment variables")
	}

	return &NotificationResult{
		Status:  "success",
		Message: "Webhook processed successfully",
	}, nil
}

// parseNotification extracts video entry from the XML notification
func (ns *NotificationService) parseNotification(r *http.Request) (*Entry, error) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body")
	}

	// Parse XML
	var feed AtomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("invalid XML")
	}

	return feed.Entry, nil
}

// NotificationResult represents the result of processing a notification
type NotificationResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
package webhook

import (
	"time"
)

// VideoProcessor handles video-related business logic
type VideoProcessor struct{}

// NewVideoProcessor creates a new video processor instance
func NewVideoProcessor() *VideoProcessor {
	return &VideoProcessor{}
}

// IsNewVideo determines if a video entry represents a new video publication
// rather than an update to an existing video
func (vp *VideoProcessor) IsNewVideo(entry *Entry) bool {
	// Parse timestamps
	published, err := time.Parse(time.RFC3339, entry.Published)
	if err != nil {
		// If we can't parse the timestamp, skip for safety (don't assume it's new)
		return false
	}

	updated, err := time.Parse(time.RFC3339, entry.Updated)
	if err != nil {
		// If we can't parse the timestamp, skip for safety (don't assume it's new)
		return false
	}

	now := time.Now()

	// Consider a video "new" if:
	// 1. It was published within the last hour
	// 2. The difference between published and updated time is small (less than 15 minutes)
	timeSincePublished := now.Sub(published)
	updatePublishDiff := updated.Sub(published)

	// If published more than 1 hour ago, it's likely an old video update
	if timeSincePublished > time.Hour {
		return false
	}

	// If there's a large gap between publish and update, it's likely an update to an old video
	if updatePublishDiff > 15*time.Minute {
		return false
	}

	return true
}

// ValidateEntry performs basic validation on video entry data
func (vp *VideoProcessor) ValidateEntry(entry *Entry) error {
	if entry == nil {
		return ErrInvalidEntry
	}

	if entry.VideoID == "" {
		return ErrMissingVideoID
	}

	if entry.ChannelID == "" {
		return ErrMissingChannelID
	}

	return nil
}

package webhook

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoProcessor(t *testing.T) {
	processor := NewVideoProcessor()
	require.NotNil(t, processor)
}

func TestVideoProcessor_IsNewVideo(t *testing.T) {
	processor := NewVideoProcessor()
	now := time.Now()

	testCases := []struct {
		name        string
		entry       *Entry
		expected    bool
		description string
	}{
		{
			name: "new_video_recent_publish",
			entry: &Entry{
				VideoID:   "test_video_id",
				ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Title:     "Test Video",
				Published: now.Add(-30 * time.Minute).Format(time.RFC3339),
				Updated:   now.Add(-29 * time.Minute).Format(time.RFC3339),
			},
			expected:    true,
			description: "Video published 30 minutes ago should be considered new",
		},
		{
			name: "old_video_published_over_hour_ago",
			entry: &Entry{
				VideoID:   "test_video_id",
				ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Title:     "Test Video",
				Published: now.Add(-2 * time.Hour).Format(time.RFC3339),
				Updated:   now.Add(-30 * time.Minute).Format(time.RFC3339),
			},
			expected:    false,
			description: "Video published 2 hours ago should not be considered new",
		},
		{
			name: "video_with_large_publish_update_diff",
			entry: &Entry{
				VideoID:   "test_video_id",
				ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Title:     "Test Video",
				Published: now.Add(-30 * time.Minute).Format(time.RFC3339),
				Updated:   now.Add(-10 * time.Minute).Format(time.RFC3339),
			},
			expected:    false,
			description: "Video with large publish-update diff should not be considered new",
		},
		{
			name: "video_with_invalid_published_date",
			entry: &Entry{
				VideoID:   "test_video_id",
				ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Title:     "Test Video",
				Published: "invalid-date",
				Updated:   now.Format(time.RFC3339),
			},
			expected:    true,
			description: "Video with invalid published date should be considered new (fail-safe)",
		},
		{
			name: "video_with_invalid_updated_date",
			entry: &Entry{
				VideoID:   "test_video_id",
				ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Title:     "Test Video",
				Published: now.Format(time.RFC3339),
				Updated:   "invalid-date",
			},
			expected:    true,
			description: "Video with invalid updated date should be considered new (fail-safe)",
		},
		{
			name: "video_with_both_invalid_dates",
			entry: &Entry{
				VideoID:   "test_video_id",
				ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Title:     "Test Video",
				Published: "invalid-date-1",
				Updated:   "invalid-date-2",
			},
			expected:    true,
			description: "Video with both invalid dates should be considered new (fail-safe)",
		},
		{
			name: "exactly_at_hour_boundary",
			entry: &Entry{
				VideoID:   "test_video_id",
				ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Title:     "Test Video",
				Published: now.Add(-60 * time.Minute).Format(time.RFC3339),
				Updated:   now.Add(-59 * time.Minute).Format(time.RFC3339),
			},
			expected:    false,
			description: "Video published exactly 1 hour ago should not be considered new",
		},
		{
			name: "exactly_at_15_minute_diff_boundary",
			entry: &Entry{
				VideoID:   "test_video_id",
				ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Title:     "Test Video",
				Published: now.Add(-30 * time.Minute).Format(time.RFC3339),
				Updated:   now.Add(-15 * time.Minute).Format(time.RFC3339),
			},
			expected:    true,
			description: "Video with exactly 15-minute publish-update diff should be considered new (boundary case)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := processor.IsNewVideo(tc.entry)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}

func TestVideoProcessor_ValidateEntry(t *testing.T) {
	processor := NewVideoProcessor()

	testCases := []struct {
		name        string
		entry       *Entry
		expectedErr error
		description string
	}{
		{
			name: "valid_entry",
			entry: &Entry{
				VideoID:   "test_video_id",
				ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Title:     "Test Video",
				Published: time.Now().Format(time.RFC3339),
				Updated:   time.Now().Format(time.RFC3339),
			},
			expectedErr: nil,
			description: "Valid entry should pass validation",
		},
		{
			name:        "nil_entry",
			entry:       nil,
			expectedErr: ErrInvalidEntry,
			description: "Nil entry should return ErrInvalidEntry",
		},
		{
			name: "missing_video_id",
			entry: &Entry{
				VideoID:   "",
				ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Title:     "Test Video",
				Published: time.Now().Format(time.RFC3339),
				Updated:   time.Now().Format(time.RFC3339),
			},
			expectedErr: ErrMissingVideoID,
			description: "Entry with missing video ID should return ErrMissingVideoID",
		},
		{
			name: "missing_channel_id",
			entry: &Entry{
				VideoID:   "test_video_id",
				ChannelID: "",
				Title:     "Test Video",
				Published: time.Now().Format(time.RFC3339),
				Updated:   time.Now().Format(time.RFC3339),
			},
			expectedErr: ErrMissingChannelID,
			description: "Entry with missing channel ID should return ErrMissingChannelID",
		},
		{
			name: "missing_both_ids",
			entry: &Entry{
				VideoID:   "",
				ChannelID: "",
				Title:     "Test Video",
				Published: time.Now().Format(time.RFC3339),
				Updated:   time.Now().Format(time.RFC3339),
			},
			expectedErr: ErrMissingVideoID,
			description: "Entry with missing both IDs should return ErrMissingVideoID (first validation error)",
		},
		{
			name: "valid_entry_with_empty_optional_fields",
			entry: &Entry{
				VideoID:   "test_video_id",
				ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
				Title:     "", // Empty title should be OK
				Published: "", // Empty timestamps should be OK for validation
				Updated:   "",
			},
			expectedErr: nil,
			description: "Entry with empty optional fields should pass validation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := processor.ValidateEntry(tc.entry)
			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedErr, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
			}
		})
	}
}

func TestVideoProcessor_EdgeCases(t *testing.T) {
	processor := NewVideoProcessor()

	t.Run("multiple_processor_instances", func(t *testing.T) {
		processor1 := NewVideoProcessor()
		processor2 := NewVideoProcessor()
		
		// Different instances should work independently
		entry := &Entry{
			VideoID:   "test_video_id",
			ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
			Title:     "Test Video",
			Published: time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
			Updated:   time.Now().Add(-29 * time.Minute).Format(time.RFC3339),
		}
		
		result1 := processor1.IsNewVideo(entry)
		result2 := processor2.IsNewVideo(entry)
		
		assert.Equal(t, result1, result2, "Different processor instances should give same results")
	})

	t.Run("concurrent_access", func(t *testing.T) {
		entry := &Entry{
			VideoID:   "test_video_id",
			ChannelID: "UCXuqSBlHAE6Xw-yeJA0Tunw",
			Title:     "Test Video",
			Published: time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
			Updated:   time.Now().Add(-29 * time.Minute).Format(time.RFC3339),
		}

		// Test concurrent access to the same processor
		const numGoroutines = 10
		results := make(chan bool, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				results <- processor.IsNewVideo(entry)
			}()
		}
		
		// All results should be the same
		firstResult := <-results
		for i := 1; i < numGoroutines; i++ {
			result := <-results
			assert.Equal(t, firstResult, result, "Concurrent access should give consistent results")
		}
	})
}
package commands

import (
	"fmt"
	"time"

	"github.com/samsoir/youtube-webhook/cli/client"
)

// SubscribeConfig holds the configuration for the subscribe command
type SubscribeConfig struct {
	BaseURL   string
	ChannelID string
	Timeout   time.Duration
}

// Subscribe subscribes to a YouTube channel
func Subscribe(config SubscribeConfig) error {
	c := client.NewClient(config.BaseURL, config.Timeout)
	
	resp, err := c.Subscribe(config.ChannelID)
	if err != nil {
		// Check if we got a conflict response (already subscribed)
		if resp != nil && resp.Status == "conflict" {
			fmt.Printf("ℹ️  Already subscribed to channel %s\n", config.ChannelID)
			if resp.ExpiresAt != "" {
				fmt.Printf("   Expires: %s\n", resp.ExpiresAt)
			}
			return nil
		}
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	fmt.Printf("✅ Successfully subscribed to channel %s\n", config.ChannelID)
	if resp.ExpiresAt != "" {
		fmt.Printf("   Expires: %s\n", resp.ExpiresAt)
	}
	
	return nil
}

// UnsubscribeConfig holds the configuration for the unsubscribe command
type UnsubscribeConfig struct {
	BaseURL   string
	ChannelID string
	Timeout   time.Duration
}

// Unsubscribe unsubscribes from a YouTube channel
func Unsubscribe(config UnsubscribeConfig) error {
	c := client.NewClient(config.BaseURL, config.Timeout)
	
	err := c.Unsubscribe(config.ChannelID)
	if err != nil {
		// Check if it's a not found error
		if err.Error() == fmt.Sprintf("not subscribed to channel %s", config.ChannelID) {
			fmt.Printf("ℹ️  Not subscribed to channel %s\n", config.ChannelID)
			return nil
		}
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	fmt.Printf("✅ Successfully unsubscribed from channel %s\n", config.ChannelID)
	return nil
}
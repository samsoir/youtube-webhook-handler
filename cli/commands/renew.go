package commands

import (
	"fmt"
	"time"

	"github.com/samsoir/youtube-webhook/cli/client"
)

// RenewConfig holds the configuration for the renew command
type RenewConfig struct {
	BaseURL string
	Timeout time.Duration
	Verbose bool
}

// Renew triggers renewal of expiring subscriptions
func Renew(config RenewConfig) error {
	c := client.NewClient(config.BaseURL, config.Timeout)
	
	resp, err := c.RenewSubscriptions()
	if err != nil {
		return fmt.Errorf("failed to renew subscriptions: %w", err)
	}

	// Print summary
	fmt.Printf("🔄 Renewal Summary\n")
	fmt.Printf("   Checked: %d | Candidates: %d | Succeeded: %d | Failed: %d\n\n",
		resp.TotalChecked, resp.RenewalsCandidates, 
		resp.RenewalsSucceeded, resp.RenewalsFailed)

	if len(resp.Results) == 0 {
		fmt.Println("No subscriptions needed renewal.")
		return nil
	}

	// Print results if verbose or if there were any failures
	if config.Verbose || resp.RenewalsFailed > 0 {
		fmt.Println("Results:")
		for _, result := range resp.Results {
			if result.Success {
				fmt.Printf("  ✅ %s - Renewed", result.ChannelID)
				if result.NewExpiryTime != "" {
					fmt.Printf(" (expires: %s)", result.NewExpiryTime)
				}
				fmt.Println()
			} else {
				fmt.Printf("  ❌ %s - Failed: %s\n", 
					result.ChannelID, result.Message)
			}
		}
	}
	
	return nil
}
package commands

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/samsoir/youtube-webhook/cli/client"
)

// ListConfig holds the configuration for the list command
type ListConfig struct {
	BaseURL string
	Timeout time.Duration
	Format  string // "table" or "json"
}

// List lists all subscriptions
func List(config ListConfig) error {
	c := client.NewClient(config.BaseURL, config.Timeout)
	
	resp, err := c.ListSubscriptions()
	if err != nil {
		return fmt.Errorf("failed to list subscriptions: %w", err)
	}

	// Print summary
	fmt.Printf("📊 Subscription Summary\n")
	fmt.Printf("   Total: %d | Active: %d | Expired: %d\n\n", 
		resp.Total, resp.Active, resp.Expired)

	if len(resp.Subscriptions) == 0 {
		fmt.Println("No subscriptions found.")
		return nil
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CHANNEL ID\tSTATUS\tEXPIRES\tDAYS LEFT")
	fmt.Fprintln(w, "----------\t------\t-------\t---------")
	
	for _, sub := range resp.Subscriptions {
		status := sub.Status
		if status == "active" {
			status = "✅ active"
		} else {
			status = "⚠️  expired"
		}
		
		daysLeft := fmt.Sprintf("%.1f", sub.DaysUntilExpiry)
		if sub.DaysUntilExpiry < 0 {
			daysLeft = "expired"
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", 
			sub.ChannelID, status, sub.ExpiresAt, daysLeft)
	}
	w.Flush()
	
	return nil
}
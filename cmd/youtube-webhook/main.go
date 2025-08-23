package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/samsoir/youtube-webhook/cli/commands"
)

const (
	defaultTimeout = 30 * time.Second
)

func main() {
	// Define subcommands
	subscribeCmd := flag.NewFlagSet("subscribe", flag.ExitOnError)
	unsubscribeCmd := flag.NewFlagSet("unsubscribe", flag.ExitOnError)
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	renewCmd := flag.NewFlagSet("renew", flag.ExitOnError)

	// Check if a subcommand is provided
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Get the base URL from environment or flag
	baseURL := os.Getenv("YOUTUBE_WEBHOOK_URL")

	switch os.Args[1] {
	case "subscribe":
		handleSubscribe(subscribeCmd, baseURL)
	case "unsubscribe":
		handleUnsubscribe(unsubscribeCmd, baseURL)
	case "list":
		handleList(listCmd, baseURL)
	case "renew":
		handleRenew(renewCmd, baseURL)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func handleSubscribe(cmd *flag.FlagSet, defaultURL string) {
	var (
		baseURL   = cmd.String("url", defaultURL, "Base URL of the webhook service (env: YOUTUBE_WEBHOOK_URL)")
		channelID = cmd.String("channel", "", "YouTube channel ID to subscribe to (required)")
		timeout   = cmd.Duration("timeout", defaultTimeout, "Request timeout")
	)

	cmd.Parse(os.Args[2:])

	if *baseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: -url flag or YOUTUBE_WEBHOOK_URL environment variable is required")
		cmd.Usage()
		os.Exit(1)
	}

	if *channelID == "" {
		fmt.Fprintln(os.Stderr, "Error: -channel flag is required")
		cmd.Usage()
		os.Exit(1)
	}

	config := commands.SubscribeConfig{
		BaseURL:   *baseURL,
		ChannelID: *channelID,
		Timeout:   *timeout,
	}

	if err := commands.Subscribe(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleUnsubscribe(cmd *flag.FlagSet, defaultURL string) {
	var (
		baseURL   = cmd.String("url", defaultURL, "Base URL of the webhook service (env: YOUTUBE_WEBHOOK_URL)")
		channelID = cmd.String("channel", "", "YouTube channel ID to unsubscribe from (required)")
		timeout   = cmd.Duration("timeout", defaultTimeout, "Request timeout")
	)

	cmd.Parse(os.Args[2:])

	if *baseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: -url flag or YOUTUBE_WEBHOOK_URL environment variable is required")
		cmd.Usage()
		os.Exit(1)
	}

	if *channelID == "" {
		fmt.Fprintln(os.Stderr, "Error: -channel flag is required")
		cmd.Usage()
		os.Exit(1)
	}

	config := commands.UnsubscribeConfig{
		BaseURL:   *baseURL,
		ChannelID: *channelID,
		Timeout:   *timeout,
	}

	if err := commands.Unsubscribe(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleList(cmd *flag.FlagSet, defaultURL string) {
	var (
		baseURL = cmd.String("url", defaultURL, "Base URL of the webhook service (env: YOUTUBE_WEBHOOK_URL)")
		timeout = cmd.Duration("timeout", defaultTimeout, "Request timeout")
		format  = cmd.String("format", "table", "Output format (table)")
	)

	cmd.Parse(os.Args[2:])

	if *baseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: -url flag or YOUTUBE_WEBHOOK_URL environment variable is required")
		cmd.Usage()
		os.Exit(1)
	}

	config := commands.ListConfig{
		BaseURL: *baseURL,
		Timeout: *timeout,
		Format:  *format,
	}

	if err := commands.List(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleRenew(cmd *flag.FlagSet, defaultURL string) {
	var (
		baseURL = cmd.String("url", defaultURL, "Base URL of the webhook service (env: YOUTUBE_WEBHOOK_URL)")
		timeout = cmd.Duration("timeout", 60*time.Second, "Request timeout")
		verbose = cmd.Bool("verbose", false, "Show detailed renewal results")
	)

	cmd.Parse(os.Args[2:])

	if *baseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: -url flag or YOUTUBE_WEBHOOK_URL environment variable is required")
		cmd.Usage()
		os.Exit(1)
	}

	config := commands.RenewConfig{
		BaseURL: *baseURL,
		Timeout: *timeout,
		Verbose: *verbose,
	}

	if err := commands.Renew(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("YouTube Webhook CLI - Manage YouTube PubSubHubbub subscriptions")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  youtube-webhook <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  subscribe    Subscribe to a YouTube channel")
	fmt.Println("  unsubscribe  Unsubscribe from a YouTube channel")
	fmt.Println("  list         List all subscriptions")
	fmt.Println("  renew        Trigger renewal of expiring subscriptions")
	fmt.Println("  help         Show this help message")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  YOUTUBE_WEBHOOK_URL  Base URL of the webhook service (can be overridden with -url flag)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Set the base URL via environment variable")
	fmt.Println("  export YOUTUBE_WEBHOOK_URL=https://your-function.run.app")
	fmt.Println()
	fmt.Println("  # Subscribe to a channel")
	fmt.Println("  youtube-webhook subscribe -channel UCXuqSBlHAE6Xw-yeJA0Tunw")
	fmt.Println()
	fmt.Println("  # List all subscriptions")
	fmt.Println("  youtube-webhook list")
	fmt.Println()
	fmt.Println("  # Unsubscribe from a channel")
	fmt.Println("  youtube-webhook unsubscribe -channel UCXuqSBlHAE6Xw-yeJA0Tunw")
	fmt.Println()
	fmt.Println("  # Renew expiring subscriptions (verbose output)")
	fmt.Println("  youtube-webhook renew -verbose")
	fmt.Println()
	fmt.Println("  # Override the URL for a specific command")
	fmt.Println("  youtube-webhook list -url https://different-function.run.app")
	fmt.Println()
	fmt.Println("Use '<command> -h' for more information about a command.")
}
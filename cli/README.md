# YouTube Webhook CLI

A command-line interface for managing YouTube PubSubHubbub subscriptions through the YouTube Webhook Service.

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/samsoir/youtube-webhook-handler.git
cd youtube-webhook-handler

# Build and install the CLI
make install-cli

# Or just build locally
make build-cli
```

### Using Go Install

```bash
go install github.com/samsoir/youtube-webhook/cmd/youtube-webhook@latest
```

## Configuration

The CLI requires the base URL of your deployed YouTube Webhook Service. You can provide this in two ways:

### Environment Variable (Recommended)

```bash
export YOUTUBE_WEBHOOK_URL=https://your-function.run.app
```

### Command Flag

```bash
youtube-webhook list -url https://your-function.run.app
```

## Usage

### Subscribe to a Channel

Subscribe to receive notifications for a YouTube channel:

```bash
youtube-webhook subscribe -channel UCXuqSBlHAE6Xw-yeJA0Tunw
```

Output:
```
✅ Successfully subscribed to channel UCXuqSBlHAE6Xw-yeJA0Tunw
   Expires: 2024-01-22T15:30:00Z
```

### List Subscriptions

View all active and expired subscriptions:

```bash
youtube-webhook list
```

Output:
```
📊 Subscription Summary
   Total: 3 | Active: 2 | Expired: 1

CHANNEL ID                  STATUS      EXPIRES                    DAYS LEFT
----------                  ------      -------                    ---------
UCXuqSBlHAE6Xw-yeJA0Tunw   ✅ active   2024-01-22T15:30:00Z      0.9
UCdQw4w9WgXcQ              ✅ active   2024-01-23T10:00:00Z      1.4
UCabc123def456             ⚠️  expired  2024-01-20T08:00:00Z      expired
```

### Unsubscribe from a Channel

Remove a subscription:

```bash
youtube-webhook unsubscribe -channel UCXuqSBlHAE6Xw-yeJA0Tunw
```

Output:
```
✅ Successfully unsubscribed from channel UCXuqSBlHAE6Xw-yeJA0Tunw
```

### Renew Expiring Subscriptions

Manually trigger renewal of subscriptions that are close to expiring:

```bash
youtube-webhook renew
```

Output:
```
🔄 Renewal Summary
   Checked: 3 | Candidates: 1 | Succeeded: 1 | Failed: 0

No subscriptions needed renewal.
```

For detailed renewal results:

```bash
youtube-webhook renew -verbose
```

## Command Reference

### Global Flags

All commands support these flags:

- `-url string`: Base URL of the webhook service (overrides YOUTUBE_WEBHOOK_URL)
- `-timeout duration`: Request timeout (default: 30s)
- `-h, -help`: Show help for the command

### subscribe

Subscribe to a YouTube channel.

```bash
youtube-webhook subscribe [flags]
```

Flags:
- `-channel string`: YouTube channel ID (required)
- `-url string`: Service URL
- `-timeout duration`: Request timeout

### unsubscribe

Unsubscribe from a YouTube channel.

```bash
youtube-webhook unsubscribe [flags]
```

Flags:
- `-channel string`: YouTube channel ID (required)
- `-url string`: Service URL
- `-timeout duration`: Request timeout

### list

List all subscriptions.

```bash
youtube-webhook list [flags]
```

Flags:
- `-url string`: Service URL
- `-timeout duration`: Request timeout
- `-format string`: Output format (currently only "table" supported)

### renew

Trigger renewal of expiring subscriptions.

```bash
youtube-webhook renew [flags]
```

Flags:
- `-url string`: Service URL
- `-timeout duration`: Request timeout (default: 60s)
- `-verbose bool`: Show detailed renewal results

## Finding YouTube Channel IDs

YouTube channel IDs always start with "UC" followed by 22 characters. You can find a channel ID by:

1. **From the channel URL**: 
   - Go to the channel page
   - Look for URLs like: `youtube.com/channel/UCXuqSBlHAE6Xw-yeJA0Tunw`
   - The ID is the part after `/channel/`

2. **From the channel page source**:
   - Right-click on the channel page
   - Select "View Page Source"
   - Search for "channelId"

3. **Using the YouTube API**:
   - Use the YouTube Data API to search for channels by name

## Error Handling

The CLI provides clear error messages for common issues:

- **Missing URL**: Configure YOUTUBE_WEBHOOK_URL or use -url flag
- **Invalid Channel ID**: Must start with "UC" and be 24 characters total
- **Already Subscribed**: Shows as info message with current expiration
- **Not Subscribed**: Shows as info message when unsubscribing
- **Network Errors**: Displays connection issues with the service

## Examples

### Complete Workflow

```bash
# Set the service URL
export YOUTUBE_WEBHOOK_URL=https://my-webhook.run.app

# Subscribe to multiple channels
youtube-webhook subscribe -channel UCXuqSBlHAE6Xw-yeJA0Tunw
youtube-webhook subscribe -channel UCdQw4w9WgXcQ

# Check subscription status
youtube-webhook list

# Renew expiring subscriptions
youtube-webhook renew -verbose

# Unsubscribe from a channel
youtube-webhook unsubscribe -channel UCXuqSBlHAE6Xw-yeJA0Tunw
```

### Using Different Environments

```bash
# Development
youtube-webhook list -url https://dev-webhook.run.app

# Production  
youtube-webhook list -url https://prod-webhook.run.app
```

## Troubleshooting

### "Error: -url flag or YOUTUBE_WEBHOOK_URL environment variable is required"

Set the YOUTUBE_WEBHOOK_URL environment variable or provide the -url flag with each command.

### "Invalid channel ID format"

Ensure the channel ID:
- Starts with "UC"
- Is exactly 24 characters long
- Contains only alphanumeric characters, hyphens, and underscores

### "Failed to subscribe: server error (502)"

The PubSubHubbub hub may be temporarily unavailable. Try again later.

## Development

To contribute to the CLI development:

```bash
# Run tests
go test ./cli/...

# Build locally
make build-cli

# Install to GOPATH/bin
make install-cli
```

## License

MIT License - see LICENSE file for details.
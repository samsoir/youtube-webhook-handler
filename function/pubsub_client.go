package webhook

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

// PubSubClient defines the interface for PubSubHubbub operations.
type PubSubClient interface {
	Subscribe(channelID string) error
	Unsubscribe(channelID string) error
}

// HTTPPubSubClient implements PubSubClient using HTTP requests.
type HTTPPubSubClient struct {
	hubURL      string
	callbackURL string
	client      *http.Client
}

// NewHTTPPubSubClient creates a new HTTP-based PubSub client.
func NewHTTPPubSubClient() *HTTPPubSubClient {
	callbackURL := os.Getenv("FUNCTION_URL")
	if callbackURL == "" {
		callbackURL = "https://default-function-url"
	}

	return &HTTPPubSubClient{
		hubURL:      "https://pubsubhubbub.appspot.com/subscribe",
		callbackURL: callbackURL,
		client:      &http.Client{Timeout: 30 * time.Second},
	}
}

// Subscribe subscribes to a YouTube channel via PubSubHubbub.
func (c *HTTPPubSubClient) Subscribe(channelID string) error {
	return c.makePubSubHubbubRequest(channelID, "subscribe")
}

// Unsubscribe unsubscribes from a YouTube channel via PubSubHubbub.
func (c *HTTPPubSubClient) Unsubscribe(channelID string) error {
	return c.makePubSubHubbubRequest(channelID, "unsubscribe")
}

// makePubSubHubbubRequest makes a subscription/unsubscription request to the hub.
func (c *HTTPPubSubClient) makePubSubHubbubRequest(channelID, mode string) error {
	topicURL := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelID)

	data := url.Values{}
	data.Set("hub.callback", c.callbackURL)
	data.Set("hub.topic", topicURL)
	data.Set("hub.mode", mode)
	data.Set("hub.verify", "async")
	data.Set("hub.lease_seconds", "86400")

	resp, err := c.client.PostForm(c.hubURL, data)
	if err != nil {
		return fmt.Errorf("failed to make PubSubHubbub request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("PubSubHubbub hub returned status: %d", resp.StatusCode)
	}

	return nil
}

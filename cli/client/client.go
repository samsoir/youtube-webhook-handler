package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	webhook "github.com/samsoir/youtube-webhook/function"
)

// Client provides methods to interact with the YouTube webhook service
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new webhook service client
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Subscribe subscribes to a YouTube channel
func (c *Client) Subscribe(channelID string) (*webhook.APIResponse, error) {
	url := fmt.Sprintf("%s/subscribe?channel_id=%s", c.baseURL, channelID)
	
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var apiResp webhook.APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode >= 400 {
		if apiResp.Message != "" {
			return &apiResp, fmt.Errorf("server error (%d): %s", resp.StatusCode, apiResp.Message)
		}
		return &apiResp, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return &apiResp, nil
}

// Unsubscribe unsubscribes from a YouTube channel
func (c *Client) Unsubscribe(channelID string) error {
	url := fmt.Sprintf("%s/unsubscribe?channel_id=%s", c.baseURL, channelID)
	
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil // Success
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not subscribed to channel %s", channelID)
	}

	// Try to parse error response
	body, _ := io.ReadAll(resp.Body)
	var apiResp webhook.APIResponse
	if err := json.Unmarshal(body, &apiResp); err == nil && apiResp.Message != "" {
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, apiResp.Message)
	}

	return fmt.Errorf("server returned status %d", resp.StatusCode)
}

// ListSubscriptions lists all subscriptions
func (c *Client) ListSubscriptions() (*webhook.SubscriptionsListResponse, error) {
	url := fmt.Sprintf("%s/subscriptions", c.baseURL)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var apiResp webhook.APIResponse
		if err := json.Unmarshal(body, &apiResp); err == nil && apiResp.Message != "" {
			return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, apiResp.Message)
		}
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var listResp webhook.SubscriptionsListResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &listResp, nil
}

// RenewSubscriptions triggers renewal of expiring subscriptions
func (c *Client) RenewSubscriptions() (*webhook.RenewalSummaryResponse, error) {
	url := fmt.Sprintf("%s/renew", c.baseURL)
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte{}))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiResp webhook.APIResponse
		if err := json.Unmarshal(body, &apiResp); err == nil && apiResp.Message != "" {
			return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, apiResp.Message)
		}
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var renewResp webhook.RenewalSummaryResponse
	if err := json.Unmarshal(body, &renewResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &renewResp, nil
}
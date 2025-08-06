package webhook

import "sync"

// MockPubSubClient implements PubSubClient for testing.
type MockPubSubClient struct {
	mu                sync.RWMutex
	subscribeError    error
	unsubscribeError  error
	subscribeCount    int
	unsubscribeCount  int
	lastChannelID     string
	lastMode          string
	subscriptions     map[string]bool
}

// NewMockPubSubClient creates a new mock PubSub client.
func NewMockPubSubClient() *MockPubSubClient {
	return &MockPubSubClient{
		subscriptions: make(map[string]bool),
	}
}

// Subscribe simulates subscribing to a channel.
func (m *MockPubSubClient) Subscribe(channelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.subscribeCount++
	m.lastChannelID = channelID
	m.lastMode = "subscribe"

	if m.subscribeError != nil {
		return m.subscribeError
	}

	m.subscriptions[channelID] = true
	return nil
}

// Unsubscribe simulates unsubscribing from a channel.
func (m *MockPubSubClient) Unsubscribe(channelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.unsubscribeCount++
	m.lastChannelID = channelID
	m.lastMode = "unsubscribe"

	if m.unsubscribeError != nil {
		return m.unsubscribeError
	}

	delete(m.subscriptions, channelID)
	return nil
}

// SetSubscribeError sets the error to return for subscribe operations.
func (m *MockPubSubClient) SetSubscribeError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribeError = err
}

// SetUnsubscribeError sets the error to return for unsubscribe operations.
func (m *MockPubSubClient) SetUnsubscribeError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unsubscribeError = err
}

// GetSubscribeCount returns the number of subscribe calls.
func (m *MockPubSubClient) GetSubscribeCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.subscribeCount
}

// GetUnsubscribeCount returns the number of unsubscribe calls.
func (m *MockPubSubClient) GetUnsubscribeCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.unsubscribeCount
}

// GetLastChannelID returns the last channel ID used in an operation.
func (m *MockPubSubClient) GetLastChannelID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastChannelID
}

// GetLastMode returns the last mode used in an operation.
func (m *MockPubSubClient) GetLastMode() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastMode
}

// IsSubscribed returns whether a channel is currently subscribed.
func (m *MockPubSubClient) IsSubscribed(channelID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.subscriptions[channelID]
}

// Reset resets the mock to initial state.
func (m *MockPubSubClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.subscribeError = nil
	m.unsubscribeError = nil
	m.subscribeCount = 0
	m.unsubscribeCount = 0
	m.lastChannelID = ""
	m.lastMode = ""
	m.subscriptions = make(map[string]bool)
}
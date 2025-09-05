package webhook

import "sync"

// GitHubClientInterface defines the interface for GitHub API operations.
type GitHubClientInterface interface {
	TriggerWorkflow(repoOwner, repoName string, entry *Entry) error
	IsConfigured() bool
}

// MockGitHubClient implements GitHubClientInterface for testing.
type MockGitHubClient struct {
	mu               sync.RWMutex
	triggerError     error
	isConfigured     bool
	triggerCallCount int
	lastRepoOwner    string
	lastRepoName     string
	lastEntry        *Entry
}

// NewMockGitHubClient creates a new mock GitHub client.
func NewMockGitHubClient() *MockGitHubClient {
	return &MockGitHubClient{
		isConfigured: true, // Default to configured for testing
	}
}

// TriggerWorkflow simulates triggering a GitHub workflow.
func (m *MockGitHubClient) TriggerWorkflow(repoOwner, repoName string, entry *Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.triggerCallCount++
	m.lastRepoOwner = repoOwner
	m.lastRepoName = repoName
	m.lastEntry = entry

	return m.triggerError
}

// IsConfigured returns whether the GitHub client is configured.
func (m *MockGitHubClient) IsConfigured() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isConfigured
}

// SetTriggerError sets the error to return from TriggerWorkflow.
func (m *MockGitHubClient) SetTriggerError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.triggerError = err
}

// SetConfigured sets whether the client is configured.
func (m *MockGitHubClient) SetConfigured(configured bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isConfigured = configured
}

// GetTriggerCallCount returns the number of TriggerWorkflow calls.
func (m *MockGitHubClient) GetTriggerCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.triggerCallCount
}

// GetLastEntry returns the last entry passed to TriggerWorkflow.
func (m *MockGitHubClient) GetLastEntry() *Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastEntry
}

// Reset resets the mock to initial state.
func (m *MockGitHubClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.triggerError = nil
	m.isConfigured = true
	m.triggerCallCount = 0
	m.lastRepoOwner = ""
	m.lastRepoName = ""
	m.lastEntry = nil
}

package webhook

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// RefactoredMockStorageClient implements StorageClient for testing.
type RefactoredMockStorageClient struct {
	mu    sync.RWMutex
	state *SubscriptionState
	
	// Control test behavior
	LoadError      error
	SaveError      error
	LoadCallCount  int
	SaveCallCount  int
	LastSavedState *SubscriptionState
}

// NewRefactoredMockStorageClient creates a new mock storage client.
func NewRefactoredMockStorageClient() *RefactoredMockStorageClient {
	return &RefactoredMockStorageClient{
		state: func() *SubscriptionState {
			state := &SubscriptionState{
				Subscriptions: make(map[string]*Subscription),
			}
			state.Metadata.LastUpdated = time.Now()
			state.Metadata.Version = "1.0"
			return state
		}(),
	}
}

// LoadSubscriptionState loads the subscription state from memory.
func (m *RefactoredMockStorageClient) LoadSubscriptionState(ctx context.Context) (*SubscriptionState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.LoadCallCount++
	
	if m.LoadError != nil {
		return nil, m.LoadError
	}
	
	if m.state == nil {
		state := &SubscriptionState{
			Subscriptions: make(map[string]*Subscription),
		}
		state.Metadata.LastUpdated = time.Now()
		state.Metadata.Version = "1.0"
		return state, nil
	}
	
	// Deep copy the state to prevent modifications
	return m.deepCopyState(m.state), nil
}

// SaveSubscriptionState saves the subscription state to memory.
func (m *RefactoredMockStorageClient) SaveSubscriptionState(ctx context.Context, state *SubscriptionState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.SaveCallCount++
	
	if m.SaveError != nil {
		return m.SaveError
	}
	
	// Update metadata  
	state.Metadata.LastUpdated = time.Now()
	state.Metadata.Version = "1.0"
	
	// Deep copy for storage
	m.state = m.deepCopyState(state)
	m.LastSavedState = m.deepCopyState(state)
	
	return nil
}

// Close is a no-op for the mock client.
func (m *RefactoredMockStorageClient) Close() error {
	return nil
}

// SetState sets the internal state for testing.
func (m *RefactoredMockStorageClient) SetState(state *SubscriptionState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = m.deepCopyState(state)
}

// GetState returns the current internal state for testing.
func (m *RefactoredMockStorageClient) GetState() *SubscriptionState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.deepCopyState(m.state)
}

// Reset resets the mock to initial state.
func (m *RefactoredMockStorageClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.state = func() *SubscriptionState {
		state := &SubscriptionState{
			Subscriptions: make(map[string]*Subscription),
		}
		state.Metadata.LastUpdated = time.Now()
		state.Metadata.Version = "1.0"
		return state
	}()
	m.LoadError = nil
	m.SaveError = nil
	m.LoadCallCount = 0
	m.SaveCallCount = 0
	m.LastSavedState = nil
}

// deepCopyState creates a deep copy of the subscription state.
func (m *RefactoredMockStorageClient) deepCopyState(state *SubscriptionState) *SubscriptionState {
	if state == nil {
		return nil
	}
	
	// Use JSON marshal/unmarshal for deep copy
	data, err := json.Marshal(state)
	if err != nil {
		return nil
	}
	
	var copy SubscriptionState
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil
	}
	
	// Ensure map is initialized
	if copy.Subscriptions == nil {
		copy.Subscriptions = make(map[string]*Subscription)
	}
	
	return &copy
}

// MockStorageError represents a custom error for testing.
type MockStorageError struct {
	Message string
}

func (e MockStorageError) Error() string {
	return e.Message
}

// Common test errors
var (
	ErrMockLoadFailure = MockStorageError{Message: "mock load failure"}
	ErrMockSaveFailure = MockStorageError{Message: "mock save failure"}
	ErrMockTimeout     = MockStorageError{Message: "mock timeout"}
)
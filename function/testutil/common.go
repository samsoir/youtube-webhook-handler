package testutil

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// StorageInterface defines the contract for subscription state storage operations
// This mirrors the interface from the main package to avoid import cycles
type StorageInterface interface {
	LoadSubscriptionState(ctx context.Context) (interface{}, error)
	SaveSubscriptionState(ctx context.Context, state interface{}) error
}

// MockStorageClient is a mock implementation of StorageInterface for testing
type MockStorageClient struct {
	mock.Mock
}

// LoadSubscriptionState mocks the LoadSubscriptionState method
func (m *MockStorageClient) LoadSubscriptionState(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

// SaveSubscriptionState mocks the SaveSubscriptionState method
func (m *MockStorageClient) SaveSubscriptionState(ctx context.Context, state interface{}) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

// TestChannelIDs provides commonly used test channel IDs
var TestChannelIDs = struct {
	Valid   string
	Valid2  string
	Invalid string
}{
	Valid:   "UCXuqSBlHAE6Xw-yeJA0Tunw",
	Valid2:  "UC_x5XG1OV2P6uZZ5FSM9Ttw",
	Invalid: "invalid-channel-id",
}

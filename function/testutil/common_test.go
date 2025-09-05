package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockStorageClient tests the mock storage client functionality
func TestMockStorageClient(t *testing.T) {
	mock := &MockStorageClient{}
	ctx := context.Background()

	// Test LoadSubscriptionState
	t.Run("LoadSubscriptionState", func(t *testing.T) {
		expectedState := map[string]interface{}{"test": "data"}
		mock.On("LoadSubscriptionState", ctx).Return(expectedState, nil)

		state, err := mock.LoadSubscriptionState(ctx)
		require.NoError(t, err)
		assert.Equal(t, expectedState, state)

		mock.AssertExpectations(t)
	})

	t.Run("LoadSubscriptionState_Error", func(t *testing.T) {
		mock.ExpectedCalls = nil // Clear previous expectations
		expectedErr := fmt.Errorf("load error")
		mock.On("LoadSubscriptionState", ctx).Return(nil, expectedErr)

		state, err := mock.LoadSubscriptionState(ctx)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, state)

		mock.AssertExpectations(t)
	})

	// Test SaveSubscriptionState
	t.Run("SaveSubscriptionState", func(t *testing.T) {
		mock.ExpectedCalls = nil // Clear previous expectations
		testState := map[string]interface{}{"test": "data"}
		mock.On("SaveSubscriptionState", ctx, testState).Return(nil)

		err := mock.SaveSubscriptionState(ctx, testState)
		require.NoError(t, err)

		mock.AssertExpectations(t)
	})

	t.Run("SaveSubscriptionState_Error", func(t *testing.T) {
		mock.ExpectedCalls = nil // Clear previous expectations
		testState := map[string]interface{}{"test": "data"}
		expectedErr := fmt.Errorf("save error")
		mock.On("SaveSubscriptionState", ctx, testState).Return(expectedErr)

		err := mock.SaveSubscriptionState(ctx, testState)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)

		mock.AssertExpectations(t)
	})
}

// TestChannelIDConstants tests the test channel ID constants
func TestChannelIDConstants(t *testing.T) {
	// Test that channel IDs are not empty and have expected formats
	assert.NotEmpty(t, TestChannelIDs.Valid)
	assert.NotEmpty(t, TestChannelIDs.Valid2)
	assert.NotEmpty(t, TestChannelIDs.Invalid)

	// Test that valid channel IDs start with UC and have correct length
	assert.True(t, len(TestChannelIDs.Valid) == 24)
	assert.True(t, TestChannelIDs.Valid[:2] == "UC")

	assert.True(t, len(TestChannelIDs.Valid2) == 24)
	assert.True(t, TestChannelIDs.Valid2[:2] == "UC")

	// Test that invalid channel ID is actually invalid
	assert.True(t, len(TestChannelIDs.Invalid) != 24 || TestChannelIDs.Invalid[:2] != "UC")

	// Test that all IDs are unique
	assert.NotEqual(t, TestChannelIDs.Valid, TestChannelIDs.Valid2)
	assert.NotEqual(t, TestChannelIDs.Valid, TestChannelIDs.Invalid)
	assert.NotEqual(t, TestChannelIDs.Valid2, TestChannelIDs.Invalid)
}
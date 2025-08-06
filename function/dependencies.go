package webhook

import "sync"

// Dependencies holds all the external dependencies for the webhook service.
type Dependencies struct {
	StorageClient *RefactoredMockStorageClient  // For proof-of-concept, use mock directly
	PubSubClient  PubSubClient
	GitHubClient  GitHubClientInterface
}

var (
	globalDependencies *Dependencies
	dependenciesMutex  sync.RWMutex
	dependenciesOnce   sync.Once
)

// GetDependencies returns the global dependencies instance.
// Creates production dependencies if none exist.
func GetDependencies() *Dependencies {
	dependenciesMutex.RLock()
	if globalDependencies != nil {
		defer dependenciesMutex.RUnlock()
		return globalDependencies
	}
	dependenciesMutex.RUnlock()

	// Create dependencies if they don't exist
	dependenciesOnce.Do(func() {
		globalDependencies = CreateProductionDependencies()
	})

	return globalDependencies
}

// SetDependencies sets the global dependencies (primarily for testing).
func SetDependencies(deps *Dependencies) {
	dependenciesMutex.Lock()
	defer dependenciesMutex.Unlock()
	globalDependencies = deps
}

// CreateProductionDependencies creates dependencies for production use.
func CreateProductionDependencies() *Dependencies {
	// For proof-of-concept, use mocks in production too
	// In real implementation, this would use actual services
	return &Dependencies{
		StorageClient: NewRefactoredMockStorageClient(),
		PubSubClient:  NewHTTPPubSubClient(),
		GitHubClient:  NewGitHubClient(), // Use real GitHub client in production
	}
}

// CreateTestDependencies creates dependencies for testing.
func CreateTestDependencies() *Dependencies {
	return &Dependencies{
		StorageClient: NewRefactoredMockStorageClient(),
		PubSubClient:  NewMockPubSubClient(),
		GitHubClient:  NewMockGitHubClient(),
	}
}
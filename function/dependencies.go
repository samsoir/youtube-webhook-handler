package webhook

import "sync"

// Dependencies holds all the external dependencies for the webhook service.
type Dependencies struct {
	StorageClient StorageService       // Use proper storage interface
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
	return &Dependencies{
		StorageClient: NewCloudStorageService(), // Use real Cloud Storage with caching
		PubSubClient:  NewHTTPPubSubClient(),    // Use real HTTP PubSub client
		GitHubClient:  NewGitHubClient(),        // Use real GitHub client
	}
}

// CreateTestDependencies creates dependencies for testing.
func CreateTestDependencies() *Dependencies {
	return &Dependencies{
		StorageClient: NewMockStorageClient(),  // Mock for testing only
		PubSubClient:  NewMockPubSubClient(),   // Mock for testing only  
		GitHubClient:  NewMockGitHubClient(),   // Mock for testing only
	}
}

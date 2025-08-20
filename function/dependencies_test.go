package webhook

import (
	"os"
	"sync"
	"testing"
)

func TestCreateProductionDependencies(t *testing.T) {
	// Set required environment variables for production dependencies
	os.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
	os.Setenv("STORAGE_BUCKET_NAME", "test-bucket")
	defer func() {
		os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		os.Unsetenv("STORAGE_BUCKET_NAME")
	}()

	deps := CreateProductionDependencies()

	if deps == nil {
		t.Fatal("CreateProductionDependencies returned nil")
	}

	if deps.StorageClient == nil {
		t.Error("StorageClient is nil")
	}

	if deps.PubSubClient == nil {
		t.Error("PubSubClient is nil")
	}

	if deps.GitHubClient == nil {
		t.Error("GitHubClient is nil")
	}

	// Verify types
	if _, ok := deps.StorageClient.(*CloudStorageService); !ok {
		t.Error("StorageClient is not CloudStorageService")
	}

	if _, ok := deps.PubSubClient.(*HTTPPubSubClient); !ok {
		t.Error("PubSubClient is not HTTPPubSubClient")
	}

	if _, ok := deps.GitHubClient.(*GitHubClient); !ok {
		t.Error("GitHubClient is not GitHubClient")
	}
}

func TestGetDependencies_CreatesProductionDependencies(t *testing.T) {
	// Reset global state
	dependenciesMutex.Lock()
	globalDependencies = nil
	dependenciesOnce = sync.Once{}
	dependenciesMutex.Unlock()

	// Set required environment variables
	os.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
	os.Setenv("STORAGE_BUCKET_NAME", "test-bucket")
	defer func() {
		os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		os.Unsetenv("STORAGE_BUCKET_NAME")
	}()

	deps := GetDependencies()

	if deps == nil {
		t.Fatal("GetDependencies returned nil")
	}

	// Should be production dependencies
	if _, ok := deps.StorageClient.(*CloudStorageService); !ok {
		t.Error("Expected CloudStorageService, got different type")
	}

	// Calling again should return the same instance
	deps2 := GetDependencies()
	if deps != deps2 {
		t.Error("GetDependencies should return the same instance")
	}
}

func TestGetDependencies_ReturnsExistingDependencies(t *testing.T) {
	// Set test dependencies
	testDeps := CreateTestDependencies()
	SetDependencies(testDeps)

	deps := GetDependencies()

	if deps != testDeps {
		t.Error("GetDependencies should return existing dependencies")
	}

	// Reset global state for other tests
	dependenciesMutex.Lock()
	globalDependencies = nil
	dependenciesOnce = sync.Once{}
	dependenciesMutex.Unlock()
}

func TestSetDependencies(t *testing.T) {
	// Create test dependencies
	testDeps := CreateTestDependencies()

	// Set them
	SetDependencies(testDeps)

	// Get them back
	deps := GetDependencies()

	if deps != testDeps {
		t.Error("SetDependencies/GetDependencies should preserve the same instance")
	}

	// Verify they are test dependencies
	if _, ok := deps.StorageClient.(*MockStorageClient); !ok {
		t.Error("Expected MockStorageClient after setting test dependencies")
	}

	// Reset global state for other tests
	dependenciesMutex.Lock()
	globalDependencies = nil
	dependenciesOnce = sync.Once{}
	dependenciesMutex.Unlock()
}

func TestDependencies_ConcurrentAccess(t *testing.T) {
	// Reset global state
	dependenciesMutex.Lock()
	globalDependencies = nil
	dependenciesOnce = sync.Once{}
	dependenciesMutex.Unlock()

	// Set required environment variables
	os.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
	os.Setenv("STORAGE_BUCKET_NAME", "test-bucket")
	defer func() {
		os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		os.Unsetenv("STORAGE_BUCKET_NAME")
	}()

	// Test concurrent access to GetDependencies
	const numGoroutines = 10
	results := make([]*Dependencies, numGoroutines)
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = GetDependencies()
		}(i)
	}

	wg.Wait()

	// All should return the same instance
	firstResult := results[0]
	for i := 1; i < numGoroutines; i++ {
		if results[i] != firstResult {
			t.Errorf("Concurrent GetDependencies calls should return the same instance")
		}
	}

	// Reset global state for other tests
	dependenciesMutex.Lock()
	globalDependencies = nil
	dependenciesOnce = sync.Once{}
	dependenciesMutex.Unlock()
}
package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/storage"
)

// StorageService defines the interface for subscription state storage operations
type StorageService interface {
	LoadSubscriptionState(ctx context.Context) (*SubscriptionState, error)
	SaveSubscriptionState(ctx context.Context, state *SubscriptionState) error
	Close() error
}

// OptimizedCloudStorageService provides an optimized Cloud Storage implementation
// with connection pooling and caching
type OptimizedCloudStorageService struct {
	client     *storage.Client
	bucketName string
	objectPath string
	
	// Cache layer
	cache      *SubscriptionState
	cacheTime  time.Time
	cacheTTL   time.Duration
	cacheMutex sync.RWMutex
	
	// Configuration
	testMode bool
	
	// Singleton pattern for connection reuse
	initOnce sync.Once
	initErr  error
}

// NewOptimizedCloudStorageService creates a new optimized storage service
func NewOptimizedCloudStorageService() *OptimizedCloudStorageService {
	return &OptimizedCloudStorageService{
		objectPath: "subscriptions/state.json",
		cacheTTL:   5 * time.Minute, // Cache for 5 minutes
		testMode:   testMode,
	}
}

// initialize sets up the storage client with proper error handling
func (s *OptimizedCloudStorageService) initialize(ctx context.Context) error {
	s.initOnce.Do(func() {
		s.bucketName = os.Getenv("SUBSCRIPTION_BUCKET")
		if s.bucketName == "" {
			s.initErr = fmt.Errorf("SUBSCRIPTION_BUCKET environment variable not set")
			return
		}

		// Only create client if not in test mode
		if !s.testMode {
			client, err := storage.NewClient(ctx)
			if err != nil {
				s.initErr = fmt.Errorf("failed to create storage client: %v", err)
				return
			}
			s.client = client
		}
	})
	return s.initErr
}

// LoadSubscriptionState loads subscription state with caching
func (s *OptimizedCloudStorageService) LoadSubscriptionState(ctx context.Context) (*SubscriptionState, error) {
	// Handle test mode
	if s.testMode {
		return s.loadTestModeState(), nil
	}

	// Check cache first
	if cachedState := s.getCachedState(); cachedState != nil {
		return s.deepCopyState(cachedState), nil
	}

	// Initialize client if needed
	if err := s.initialize(ctx); err != nil {
		return nil, err
	}

	// Load from Cloud Storage
	state, err := s.loadFromStorage(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache
	s.setCachedState(state)

	return s.deepCopyState(state), nil
}

// SaveSubscriptionState saves subscription state and updates cache
func (s *OptimizedCloudStorageService) SaveSubscriptionState(ctx context.Context, state *SubscriptionState) error {
	// Handle test mode
	if s.testMode {
		return s.saveTestModeState(state)
	}

	// Initialize client if needed
	if err := s.initialize(ctx); err != nil {
		return err
	}

	// Update metadata
	s.updateMetadata(state)

	// Save to Cloud Storage
	if err := s.saveToStorage(ctx, state); err != nil {
		return err
	}

	// Update cache after successful save
	s.setCachedState(state)

	return nil
}

// Close closes the storage client and clears cache
func (s *OptimizedCloudStorageService) Close() error {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	
	s.cache = nil
	s.cacheTime = time.Time{}
	
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// Private helper methods

func (s *OptimizedCloudStorageService) getCachedState() *SubscriptionState {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	
	if s.cache != nil && time.Since(s.cacheTime) < s.cacheTTL {
		return s.cache
	}
	return nil
}

func (s *OptimizedCloudStorageService) setCachedState(state *SubscriptionState) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	
	s.cache = s.deepCopyState(state)
	s.cacheTime = time.Now()
}

func (s *OptimizedCloudStorageService) loadFromStorage(ctx context.Context) (*SubscriptionState, error) {
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(s.objectPath)
	
	reader, err := obj.NewReader(ctx)
	if err != nil {
		// If file doesn't exist, return empty state
		if err == storage.ErrObjectNotExist {
			return s.createEmptyState(), nil
		}
		return nil, fmt.Errorf("failed to open storage object: %v", err)
	}
	defer reader.Close()
	
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage object: %v", err)
	}
	
	var state SubscriptionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %v", err)
	}
	
	// Ensure subscriptions map is initialized
	if state.Subscriptions == nil {
		state.Subscriptions = make(map[string]*Subscription)
	}
	
	return &state, nil
}

func (s *OptimizedCloudStorageService) saveToStorage(ctx context.Context, state *SubscriptionState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %v", err)
	}
	
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(s.objectPath)
	
	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/json"
	
	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write state data: %v", err)
	}
	
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close storage writer: %v", err)
	}
	
	return nil
}

func (s *OptimizedCloudStorageService) loadTestModeState() *SubscriptionState {
	if testSubscriptionState == nil {
		testSubscriptionState = s.createEmptyState()
	}
	return s.deepCopyState(testSubscriptionState)
}

func (s *OptimizedCloudStorageService) saveTestModeState(state *SubscriptionState) error {
	s.updateMetadata(state)
	testSubscriptionState = state
	return nil
}

func (s *OptimizedCloudStorageService) createEmptyState() *SubscriptionState {
	return &SubscriptionState{
		Subscriptions: make(map[string]*Subscription),
		Metadata: struct {
			LastUpdated time.Time `json:"last_updated"`
			Version     string    `json:"version"`
		}{
			LastUpdated: time.Now(),
			Version:     "1.0",
		},
	}
}

func (s *OptimizedCloudStorageService) updateMetadata(state *SubscriptionState) {
	state.Metadata.LastUpdated = time.Now()
	if state.Metadata.Version == "" {
		state.Metadata.Version = "1.0"
	}
}

func (s *OptimizedCloudStorageService) deepCopyState(original *SubscriptionState) *SubscriptionState {
	if original == nil {
		return nil
	}
	
	copy := &SubscriptionState{
		Subscriptions: make(map[string]*Subscription),
		Metadata:      original.Metadata,
	}
	
	for k, v := range original.Subscriptions {
		if v != nil {
			subCopy := *v
			copy.Subscriptions[k] = &subCopy
		}
	}
	
	return copy
}

// BackwardCompatibilityStorageService provides backward compatibility with the old CloudStorageClient
type BackwardCompatibilityStorageService struct {
	optimized *OptimizedCloudStorageService
}

// NewBackwardCompatibilityStorageService creates a service that maintains compatibility
func NewBackwardCompatibilityStorageService() *BackwardCompatibilityStorageService {
	return &BackwardCompatibilityStorageService{
		optimized: NewOptimizedCloudStorageService(),
	}
}

// LoadSubscriptionState provides backward compatibility
func (b *BackwardCompatibilityStorageService) LoadSubscriptionState(ctx context.Context) (*SubscriptionState, error) {
	return b.optimized.LoadSubscriptionState(ctx)
}

// SaveSubscriptionState provides backward compatibility  
func (b *BackwardCompatibilityStorageService) SaveSubscriptionState(ctx context.Context, state *SubscriptionState) error {
	return b.optimized.SaveSubscriptionState(ctx, state)
}
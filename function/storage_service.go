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

// CloudStorageOperations defines the interface for cloud storage operations
// This abstracts away the Google Cloud Storage implementation details
type CloudStorageOperations interface {
	GetObject(ctx context.Context, bucket, objectPath string) ([]byte, error)
	PutObject(ctx context.Context, bucket, objectPath string, data []byte) error
	Close() error
}

// CloudStorageService provides an optimized Cloud Storage implementation
// with connection pooling and caching
type CloudStorageService struct {
	storageOps CloudStorageOperations
	bucketName string
	objectPath string

	// Cache layer
	cache      *SubscriptionState
	cacheTime  time.Time
	cacheTTL   time.Duration
	cacheMutex sync.RWMutex

	// Initialization
	initOnce sync.Once
	initErr  error
}

// RealCloudStorageOperations implements CloudStorageOperations using Google Cloud Storage
type RealCloudStorageOperations struct {
	client *storage.Client
}

// NewRealCloudStorageOperations creates a real Cloud Storage operations implementation
func NewRealCloudStorageOperations(ctx context.Context) (*RealCloudStorageOperations, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %v", err)
	}
	return &RealCloudStorageOperations{client: client}, nil
}

// GetObject retrieves an object from Cloud Storage
func (r *RealCloudStorageOperations) GetObject(ctx context.Context, bucket, objectPath string) ([]byte, error) {
	bucketHandle := r.client.Bucket(bucket)
	obj := bucketHandle.Object(objectPath)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// PutObject stores an object in Cloud Storage
func (r *RealCloudStorageOperations) PutObject(ctx context.Context, bucket, objectPath string, data []byte) error {
	bucketHandle := r.client.Bucket(bucket)
	obj := bucketHandle.Object(objectPath)

	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/json"

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return err
	}

	return writer.Close()
}

// Close closes the storage client
func (r *RealCloudStorageOperations) Close() error {
	return r.client.Close()
}

// NewCloudStorageService creates a new Cloud Storage service with real operations
func NewCloudStorageService() *CloudStorageService {
	// storageOps will be created during initialization
	return &CloudStorageService{
		objectPath: "subscriptions/state.json",
		cacheTTL:   5 * time.Minute,
	}
}

// NewCloudStorageServiceWithOperations creates a service with custom storage operations (for testing)
func NewCloudStorageServiceWithOperations(ops CloudStorageOperations, bucketName string) *CloudStorageService {
	return &CloudStorageService{
		storageOps: ops,
		bucketName: bucketName,
		objectPath: "subscriptions/state.json",
		cacheTTL:   5 * time.Minute,
	}
}

// initialize sets up the storage operations with proper error handling
func (s *CloudStorageService) initialize(ctx context.Context) error {
	s.initOnce.Do(func() {
		if s.bucketName == "" {
			s.bucketName = os.Getenv("SUBSCRIPTION_BUCKET")
		}
		if s.bucketName == "" {
			s.initErr = fmt.Errorf("SUBSCRIPTION_BUCKET environment variable not set")
			return
		}

		// Only create storage operations if not already provided (e.g., in tests)
		if s.storageOps == nil {
			ops, err := NewRealCloudStorageOperations(ctx)
			if err != nil {
				s.initErr = fmt.Errorf("failed to create storage operations: %v", err)
				return
			}
			s.storageOps = ops
		}
	})
	return s.initErr
}

// LoadSubscriptionState loads subscription state with caching
func (s *CloudStorageService) LoadSubscriptionState(ctx context.Context) (*SubscriptionState, error) {

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
func (s *CloudStorageService) SaveSubscriptionState(ctx context.Context, state *SubscriptionState) error {

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

// Close closes the storage operations and clears cache
func (s *CloudStorageService) Close() error {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cache = nil
	s.cacheTime = time.Time{}

	if s.storageOps != nil {
		return s.storageOps.Close()
	}
	return nil
}

// Private helper methods

func (s *CloudStorageService) getCachedState() *SubscriptionState {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	if s.cache != nil && time.Since(s.cacheTime) < s.cacheTTL {
		return s.cache
	}
	return nil
}

func (s *CloudStorageService) setCachedState(state *SubscriptionState) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cache = s.deepCopyState(state)
	s.cacheTime = time.Now()
}

func (s *CloudStorageService) loadFromStorage(ctx context.Context) (*SubscriptionState, error) {
	data, err := s.storageOps.GetObject(ctx, s.bucketName, s.objectPath)
	if err != nil {
		// If file doesn't exist, return empty state
		if err == storage.ErrObjectNotExist {
			return s.createEmptyState(), nil
		}
		return nil, fmt.Errorf("failed to get storage object: %v", err)
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

func (s *CloudStorageService) saveToStorage(ctx context.Context, state *SubscriptionState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %v", err)
	}

	if err := s.storageOps.PutObject(ctx, s.bucketName, s.objectPath, data); err != nil {
		return fmt.Errorf("failed to put storage object: %v", err)
	}

	return nil
}

// Legacy testMode methods removed - use dependency injection instead

func (s *CloudStorageService) createEmptyState() *SubscriptionState {
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

func (s *CloudStorageService) updateMetadata(state *SubscriptionState) {
	state.Metadata.LastUpdated = time.Now()
	if state.Metadata.Version == "" {
		state.Metadata.Version = "1.0"
	}
}

func (s *CloudStorageService) deepCopyState(original *SubscriptionState) *SubscriptionState {
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

// LegacyStorageService provides backward compatibility with the old CloudStorageClient
type LegacyStorageService struct {
	optimized *CloudStorageService
}

// NewLegacyStorageService creates a service that maintains compatibility
func NewLegacyStorageService() *LegacyStorageService {
	return &LegacyStorageService{
		optimized: NewCloudStorageService(),
	}
}

// LoadSubscriptionState provides backward compatibility
func (b *LegacyStorageService) LoadSubscriptionState(ctx context.Context) (*SubscriptionState, error) {
	return b.optimized.LoadSubscriptionState(ctx)
}

// SaveSubscriptionState provides backward compatibility
func (b *LegacyStorageService) SaveSubscriptionState(ctx context.Context, state *SubscriptionState) error {
	return b.optimized.SaveSubscriptionState(ctx, state)
}

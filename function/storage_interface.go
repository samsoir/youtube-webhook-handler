package webhook

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
)

// StorageClient defines the interface for interacting with Cloud Storage.
// This interface allows for easy mocking in tests using stiface.
type StorageClient interface {
	LoadSubscriptionState(ctx context.Context) (*SubscriptionState, error)
	SaveSubscriptionState(ctx context.Context, state *SubscriptionState) error
	Close() error
}

// CloudStorageClient implements StorageClient using real Google Cloud Storage.
type CloudStorageClient struct {
	client     stiface.Client
	bucketName string
	objectName string
}

// NewCloudStorageClient creates a new Cloud Storage client.
func NewCloudStorageClient(ctx context.Context, bucketName string) (*CloudStorageClient, error) {
	if bucketName == "" {
		return nil, ErrMissingBucket
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &CloudStorageClient{
		client:     stiface.AdaptClient(client),
		bucketName: bucketName,
		objectName: "subscriptions/state.json",
	}, nil
}

// LoadSubscriptionState loads the subscription state from Cloud Storage.
func (c *CloudStorageClient) LoadSubscriptionState(ctx context.Context) (*SubscriptionState, error) {
	bucket := c.client.Bucket(c.bucketName)
	obj := bucket.Object(c.objectName)
	
	reader, err := obj.NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			// Return empty state if file doesn't exist
			return &SubscriptionState{
				Subscriptions: make(map[string]*Subscription),
				Metadata: &Metadata{
					LastUpdated: time.Now(),
					Version:     "1.0",
				},
			}, nil
		}
		return nil, err
	}
	defer reader.Close()

	var state SubscriptionState
	if err := json.NewDecoder(reader).Decode(&state); err != nil {
		return nil, err
	}

	// Initialize map if nil
	if state.Subscriptions == nil {
		state.Subscriptions = make(map[string]*Subscription)
	}

	return &state, nil
}

// SaveSubscriptionState saves the subscription state to Cloud Storage.
func (c *CloudStorageClient) SaveSubscriptionState(ctx context.Context, state *SubscriptionState) error {
	if state.Metadata == nil {
		state.Metadata = &Metadata{}
	}
	state.Metadata.LastUpdated = time.Now()
	state.Metadata.Version = "1.0"

	bucket := c.client.Bucket(c.bucketName)
	obj := bucket.Object(c.objectName)
	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/json"

	if err := json.NewEncoder(writer).Encode(state); err != nil {
		writer.Close()
		return err
	}

	return writer.Close()
}

// Close closes the storage client connection.
func (c *CloudStorageClient) Close() error {
	if realClient, ok := c.client.(*stiface.ClientAdapter); ok {
		return realClient.Close()
	}
	return nil
}
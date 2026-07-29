package storage

import (
	"context"
	"io"
	"time"
)

// StorageProvider defines the interface for storage implementations
type StorageProvider interface {
	// Upload uploads a file to storage and returns the storage key
	Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error

	// Delete removes a file from storage
	Delete(ctx context.Context, key string) error

	// GenerateSignedURL generates a temporary signed URL for accessing a file
	GenerateSignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)

	// Exists checks if a file exists in storage
	Exists(ctx context.Context, key string) (bool, error)

	// Get retrieves a file from storage
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

// UploadResult contains the result of an upload operation
type UploadResult struct {
	Key         string
	Size        int64
	ContentType string
	ETag        string
}

package domain

import (
	"context"
	"io"
)

// StorageService manages file storage for product assets.
type StorageService interface {
	UploadImage(ctx context.Context, objectName string, r io.Reader, contentType string) (publicURL string, err error)
	DeleteImage(ctx context.Context, objectName string) error
	// ObjectNameFromURL extracts the storage object path from a full public URL.
	// Returns ("", false) for URLs that don't belong to this storage backend.
	ObjectNameFromURL(url string) (string, bool)
}

// NoopStorage is used when GCS credentials are not configured.
// UploadImage always returns ErrStorageNotConfigured; the other methods are no-ops.
type NoopStorage struct{}

func (NoopStorage) UploadImage(_ context.Context, _ string, _ io.Reader, _ string) (string, error) {
	return "", ErrStorageNotConfigured
}

func (NoopStorage) DeleteImage(_ context.Context, _ string) error { return nil }

func (NoopStorage) ObjectNameFromURL(_ string) (string, bool) { return "", false }

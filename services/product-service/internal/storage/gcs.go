package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// GCSStorage implements domain.StorageService backed by Google Cloud Storage.
type GCSStorage struct {
	client     *storage.Client
	bucketName string
}

// NewGCSStorage creates a GCS client from a raw service-account JSON credential string.
// If credJSON is empty the SDK falls back to Application Default Credentials
// (GOOGLE_APPLICATION_CREDENTIALS env var or the GCP metadata server).
func NewGCSStorage(ctx context.Context, bucketName, credJSON string) (*GCSStorage, error) {
	var opts []option.ClientOption
	if credJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(credJSON)))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("gcs: failed to create client: %w", err)
	}

	return &GCSStorage{client: client, bucketName: bucketName}, nil
}

// UploadImage streams r to GCS at objectName and returns the public HTTPS URL.
func (g *GCSStorage) UploadImage(ctx context.Context, objectName string, r io.Reader, contentType string) (string, error) {
	obj := g.client.Bucket(g.bucketName).Object(objectName)
	w := obj.NewWriter(ctx)
	w.ContentType = contentType
	w.CacheControl = "public, max-age=31536000" // 1 year — objects are UUID-named, never stale

	if _, err := io.Copy(w, r); err != nil {
		_ = w.Close()
		return "", fmt.Errorf("gcs: upload failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("gcs: finalise upload failed: %w", err)
	}

	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", g.bucketName, objectName), nil
}

// DeleteImage removes objectName from GCS. Returns nil if the object does not exist.
func (g *GCSStorage) DeleteImage(ctx context.Context, objectName string) error {
	err := g.client.Bucket(g.bucketName).Object(objectName).Delete(ctx)
	if err == storage.ErrObjectNotExist {
		return nil
	}
	return err
}

// ObjectNameFromURL extracts the GCS object path from a full public URL.
// Returns ("", false) for URLs that don't match this bucket.
func (g *GCSStorage) ObjectNameFromURL(url string) (string, bool) {
	prefix := fmt.Sprintf("https://storage.googleapis.com/%s/", g.bucketName)
	if strings.HasPrefix(url, prefix) {
		return strings.TrimPrefix(url, prefix), true
	}
	return "", false
}

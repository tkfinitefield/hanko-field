package storage

import (
	"context"
	"errors"
	"strings"

	gcs "cloud.google.com/go/storage"
)

// Copier provides object copy operations between Cloud Storage locations.
type Copier struct {
	client *gcs.Client
}

// NewCopier constructs a Copier backed by the provided Cloud Storage client.
func NewCopier(client *gcs.Client) (*Copier, error) {
	if client == nil {
		return nil, errors.New("storage copier: client is required")
	}
	return &Copier{client: client}, nil
}

// CopyObject copies an object from the source bucket/path to the destination.
func (c *Copier) CopyObject(ctx context.Context, sourceBucket, sourceObject, destBucket, destObject string) error {
	if c == nil || c.client == nil {
		return errors.New("storage copier: client is not initialised")
	}

	srcBucket := strings.TrimSpace(sourceBucket)
	srcObject := strings.TrimSpace(sourceObject)
	dstBucket := strings.TrimSpace(destBucket)
	dstObject := strings.TrimSpace(destObject)

	if srcBucket == "" || srcObject == "" || dstBucket == "" || dstObject == "" {
		return errors.New("storage copier: source and destination must be provided")
	}
	if srcBucket == dstBucket && srcObject == dstObject {
		return nil
	}

	src := c.client.Bucket(srcBucket).Object(srcObject)
	dst := c.client.Bucket(dstBucket).Object(dstObject)
	_, err := dst.CopierFrom(src).Run(ctx)
	return err
}

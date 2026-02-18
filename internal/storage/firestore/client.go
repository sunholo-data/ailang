// Package firestore provides Firestore-backed implementations of AILANG storage interfaces.
package firestore

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// Client wraps a Firestore client with project-specific configuration.
type Client struct {
	fs        *firestore.Client
	projectID string
}

// NewClient creates a new Firestore client using Application Default Credentials.
// Requires GOOGLE_CLOUD_PROJECT to be set.
func NewClient(ctx context.Context) (*Client, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT must be set for Firestore backend")
	}

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client: %w", err)
	}

	return &Client{
		fs:        client,
		projectID: projectID,
	}, nil
}

// Close closes the underlying Firestore client.
func (c *Client) Close() error {
	return c.fs.Close()
}

// Collection returns a Firestore collection reference.
func (c *Client) Collection(name string) *firestore.CollectionRef {
	return c.fs.Collection(name)
}

// Doc returns a Firestore document reference.
func (c *Client) Doc(collection, id string) *firestore.DocumentRef {
	return c.fs.Collection(collection).Doc(id)
}

// RunTransaction runs a Firestore transaction.
func (c *Client) RunTransaction(ctx context.Context, fn func(context.Context, *firestore.Transaction) error) error {
	return c.fs.RunTransaction(ctx, fn)
}

// timeToFirestore converts a time.Time to a value suitable for Firestore.
// Zero time is stored as nil.
func timeToFirestore(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}

// timePtrToFirestore converts a *time.Time to a value suitable for Firestore.
func timePtrToFirestore(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

// snapshotToTime extracts a time.Time from a Firestore document snapshot field.
func snapshotToTime(data map[string]interface{}, key string) time.Time {
	v, ok := data[key]
	if !ok || v == nil {
		return time.Time{}
	}
	if t, ok := v.(time.Time); ok {
		return t
	}
	return time.Time{}
}

// snapshotToTimePtr extracts a *time.Time from a Firestore document snapshot field.
func snapshotToTimePtr(data map[string]interface{}, key string) *time.Time {
	v, ok := data[key]
	if !ok || v == nil {
		return nil
	}
	if t, ok := v.(time.Time); ok {
		return &t
	}
	return nil
}

// collectDocs collects all documents from an iterator into a slice of data maps.
func collectDocs(iter *firestore.DocumentIterator) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		results = append(results, doc.Data())
	}
	return results, nil
}

// Package pubsub wraps Google Cloud Pub/Sub for AILANG messaging.
// Pub/Sub serves as a notification/transport layer on top of Firestore storage.
// Messages are stored durably in Firestore; Pub/Sub provides instant push notification
// that new work is available (replacing SQLite polling).
package pubsub

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
)

// Client wraps the Google Cloud Pub/Sub client with AILANG topic naming conventions.
type Client struct {
	ps        *pubsub.Client
	projectID string
	prefix    string // topic name prefix (default: "ailang")
}

// NewClient creates a new Pub/Sub client using Application Default Credentials.
func NewClient(ctx context.Context, projectID, prefix string) (*Client, error) {
	if projectID == "" {
		return nil, fmt.Errorf("pubsub: project ID is required")
	}
	if prefix == "" {
		prefix = DefaultTopicPrefix
	}
	ps, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("pubsub.NewClient(project=%s): %w", projectID, err)
	}
	return &Client{ps: ps, projectID: projectID, prefix: prefix}, nil
}

// TopicName returns the full topic name with prefix (e.g., "ailang-messages").
func (c *Client) TopicName(baseName string) string {
	return fmt.Sprintf("%s-%s", c.prefix, baseName)
}

// SubscriptionName returns the full subscription name with prefix.
func (c *Client) SubscriptionName(baseName string) string {
	return fmt.Sprintf("%s-%s", c.prefix, baseName)
}

// Topic returns a handle to an existing topic. Does NOT create the topic —
// topics are managed by Terraform in ailang-multivac.
func (c *Client) Topic(baseName string) *pubsub.Topic {
	return c.ps.Topic(c.TopicName(baseName))
}

// Subscription returns a handle to an existing subscription.
func (c *Client) Subscription(baseName string) *pubsub.Subscription {
	return c.ps.Subscription(c.SubscriptionName(baseName))
}

// ProjectID returns the GCP project ID.
func (c *Client) ProjectID() string {
	return c.projectID
}

// Prefix returns the topic name prefix.
func (c *Client) Prefix() string {
	return c.prefix
}

// Close closes the underlying Pub/Sub client.
func (c *Client) Close() error {
	return c.ps.Close()
}

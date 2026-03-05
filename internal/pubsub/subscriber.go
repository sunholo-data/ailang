package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	gpubsub "cloud.google.com/go/pubsub"
)

// MessageHandler is called for each received Pub/Sub message.
// Returning nil acknowledges the message; returning an error causes a nack (retry).
type MessageHandler func(ctx context.Context, data []byte, attrs map[string]string) error

// Subscriber provides pull-based subscription for AILANG Pub/Sub topics.
type Subscriber struct {
	client *Client

	mu     sync.Mutex
	cancel map[string]context.CancelFunc // active subscription cancels
}

// NewSubscriber creates a subscriber from a client.
func NewSubscriber(client *Client) *Subscriber {
	return &Subscriber{
		client: client,
		cancel: make(map[string]context.CancelFunc),
	}
}

// Subscribe starts a blocking pull subscription. It calls handler for each
// message received on the named subscription. Subscribe blocks until ctx is
// cancelled or an unrecoverable error occurs.
//
// Messages are automatically acked when handler returns nil, or nacked on error.
func (s *Subscriber) Subscribe(ctx context.Context, subName string, handler MessageHandler) error {
	sub := s.client.Subscription(subName)

	// Register cancel so Stop() can shut us down.
	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel[subName] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.cancel, subName)
		s.mu.Unlock()
		cancel()
	}()

	return sub.Receive(ctx, func(ctx context.Context, msg *gpubsub.Message) {
		if err := handler(ctx, msg.Data, msg.Attributes); err != nil {
			msg.Nack()
			return
		}
		msg.Ack()
	})
}

// ReceiveOne pulls a single message from the subscription and returns it.
// Useful for CLI commands like `ailang messages watch --pubsub`.
// Returns the raw data, attributes, and any error.
func (s *Subscriber) ReceiveOne(ctx context.Context, subName string) ([]byte, map[string]string, error) {
	sub := s.client.Subscription(subName)

	// Limit to 1 message at a time.
	sub.ReceiveSettings.MaxOutstandingMessages = 1

	var (
		data  []byte
		attrs map[string]string
		once  sync.Once
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := sub.Receive(ctx, func(_ context.Context, msg *gpubsub.Message) {
		once.Do(func() {
			data = msg.Data
			attrs = msg.Attributes
			msg.Ack()
			cancel() // Stop after first message.
		})
	})

	// context.Canceled is expected — we cancel after the first message.
	if err != nil && ctx.Err() != nil {
		err = nil
	}

	if data == nil && err == nil {
		return nil, nil, fmt.Errorf("no message received")
	}

	return data, attrs, err
}

// Stop cancels all active subscriptions.
func (s *Subscriber) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, cancel := range s.cancel {
		cancel()
		delete(s.cancel, name)
	}
}

// DecodeMessageNotification decodes a MessageNotification from raw Pub/Sub data.
func DecodeMessageNotification(data []byte) (MessageNotification, error) {
	var n MessageNotification
	if err := json.Unmarshal(data, &n); err != nil {
		return n, fmt.Errorf("decode message notification: %w", err)
	}
	return n, nil
}

// DecodeTaskDispatch decodes a TaskDispatch from raw Pub/Sub data.
func DecodeTaskDispatch(data []byte) (TaskDispatch, error) {
	var d TaskDispatch
	if err := json.Unmarshal(data, &d); err != nil {
		return d, fmt.Errorf("decode task dispatch: %w", err)
	}
	return d, nil
}

// DecodeTaskCompletion decodes a TaskCompletion from raw Pub/Sub data.
func DecodeTaskCompletion(data []byte) (TaskCompletion, error) {
	var c TaskCompletion
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("decode task completion: %w", err)
	}
	return c, nil
}

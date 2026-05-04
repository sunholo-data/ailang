// Package daemon implements a long-running consumer that pulls AILANG cloud
// events from Pub/Sub subscriptions and surfaces them as native macOS
// notifications. Designed to run under launchd.
package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/notify"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// MessageHandler matches internal/pubsub.MessageHandler — exposed here so the
// daemon can unit-test against a fake Pub/Sub subscriber without depending on
// the gpubsub package.
type MessageHandler func(ctx context.Context, data []byte, attrs map[string]string) error

// EventSubscriber is the minimum surface the daemon needs from Pub/Sub.
// Production wiring uses an adapter around *pubsub.Subscriber; tests substitute
// an in-memory fake.
type EventSubscriber interface {
	Subscribe(ctx context.Context, subName string, handler MessageHandler) error
}

// MessageFetcher resolves a Pub/Sub MessageNotification (which carries only a
// MessageID) to the full InboxMessage. Production wiring reads Firestore via
// messaging.Store; tests substitute an in-memory fake.
type MessageFetcher interface {
	Fetch(ctx context.Context, messageID string) (*messaging.InboxMessage, error)
}

// Config holds runtime parameters for the daemon. Marshalled from
// ~/.ailang/config/daemon.yaml in production; constructed directly in tests.
type Config struct {
	EventsSub   string        // e.g. "events-laptop"
	MessagesSub string        // e.g. "messages-laptop"
	TaskWindow  time.Duration // dedup window for task events
	MsgWindow   time.Duration // dedup window for message events
	Excludes    []string      // substring matches against title/body
	DryRun      bool          // skip the notifier; log instead
	Logger      *log.Logger   // optional; defaults to log.Default()
}

// Daemon is the running pull loop.
type Daemon struct {
	cfg       Config
	sub       EventSubscriber
	fetcher   MessageFetcher
	notify    func(notify.Notification) error
	taskDedup *dedup
	msgDedup  *dedup
	log       *log.Logger
}

// New constructs a Daemon. Pass interface implementations for sub/fetcher so
// tests can substitute fakes.
func New(cfg Config, sub EventSubscriber, fetcher MessageFetcher, notifyFn func(notify.Notification) error) *Daemon {
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}
	return &Daemon{
		cfg:       cfg,
		sub:       sub,
		fetcher:   fetcher,
		notify:    notifyFn,
		taskDedup: newDedup(cfg.TaskWindow),
		msgDedup:  newDedup(cfg.MsgWindow),
		log:       logger,
	}
}

// Run blocks until ctx is cancelled, pulling from both subscriptions in
// parallel and firing notifications for relevant events. Returns nil on a
// clean shutdown; returns the first error from a subscription if one fails.
func (d *Daemon) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := d.sub.Subscribe(ctx, d.cfg.EventsSub, d.handleTaskEvent)
		if err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("events subscription: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := d.sub.Subscribe(ctx, d.cfg.MessagesSub, d.handleMessageEvent)
		if err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("messages subscription: %w", err)
		}
	}()

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Daemon) handleTaskEvent(_ context.Context, data []byte, _ map[string]string) error {
	var t pubsub.TaskCompletion
	if err := json.Unmarshal(data, &t); err != nil {
		d.log.Printf("daemon: malformed task event: %v", err)
		return nil // drop poison messages by acking; nack would loop forever
	}
	n, fire := taskNotification(t)
	if !fire {
		return nil
	}
	if d.taskDedup.seen(taskDedupKey(t.TaskID, t.Status)) {
		return nil
	}
	if shouldExclude(n, d.cfg.Excludes) {
		return nil
	}
	return d.fire(n)
}

func (d *Daemon) handleMessageEvent(ctx context.Context, data []byte, _ map[string]string) error {
	var m pubsub.MessageNotification
	if err := json.Unmarshal(data, &m); err != nil {
		d.log.Printf("daemon: malformed message event: %v", err)
		return nil
	}
	if d.msgDedup.seen(messageDedupKey(m.MessageID)) {
		return nil
	}
	full, err := d.fetcher.Fetch(ctx, m.MessageID)
	if err != nil {
		return fmt.Errorf("fetch message %s: %w", m.MessageID, err)
	}
	if full == nil {
		// Notification arrived before Firestore replication; nack to retry.
		return fmt.Errorf("message %s not yet visible", m.MessageID)
	}
	n, fire := messageNotification(full)
	if !fire {
		return nil
	}
	if shouldExclude(n, d.cfg.Excludes) {
		return nil
	}
	return d.fire(n)
}

// fire invokes the notifier (or logs in dry-run). Returning an error causes
// the upstream Pub/Sub handler to nack so the message is redelivered — this
// is the "ack only after notify success" guarantee in the design doc.
func (d *Daemon) fire(n notify.Notification) error {
	if d.cfg.DryRun {
		d.log.Printf("daemon[dry-run]: %s — %s", n.Title, n.Body)
		return nil
	}
	if err := d.notify(n); err != nil {
		d.log.Printf("daemon: notify failed: %v", err)
		return err
	}
	return nil
}

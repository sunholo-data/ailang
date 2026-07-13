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

// MessageSource is one project's inbox-message feed: a Pub/Sub subscriber, the
// Firestore fetcher scoped to THAT project, the base subscription name, and a
// human label for logs. A daemon fans out over N of these so a single process
// can watch both dev and prod (see cmd/ailang/daemon.go). Every source shares
// the daemon's notifier and dedup window — message IDs are globally unique
// (fb_*/msg_*), so a shared dedup is both correct and simpler than per-source.
type MessageSource struct {
	Sub     EventSubscriber
	Fetcher MessageFetcher
	SubName string // base sub name; the Pub/Sub client prepends the project prefix
	Label   string // e.g. "dev", "prod" — for startup/delivery logs only
}

// Daemon is the running pull loop.
type Daemon struct {
	cfg        Config
	sub        EventSubscriber // primary source's subscriber; task events use this
	msgSources []MessageSource // all inbox-message sources (includes the primary)
	notify     func(notify.Notification) error
	taskDedup  *dedup
	msgDedup   *dedup
	log        *log.Logger
}

// New constructs a single-source Daemon (dev-only, backward-compatible). Task
// events and inbox messages both flow through sub/fetcher. Pass interface
// implementations so tests can substitute fakes. To watch additional projects,
// build with New then AddMessageSource, or use NewMulti.
func New(cfg Config, sub EventSubscriber, fetcher MessageFetcher, notifyFn func(notify.Notification) error) *Daemon {
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}
	d := &Daemon{
		cfg:       cfg,
		sub:       sub,
		notify:    notifyFn,
		taskDedup: newDedup(cfg.TaskWindow),
		msgDedup:  newDedup(cfg.MsgWindow),
		log:       logger,
	}
	// The primary (task+message) source is also the first message source, using
	// the configured MessagesSub. Task events are pulled ONLY from this source
	// (the rig emits eval pings to dev; we never double-fan prod task events).
	d.msgSources = []MessageSource{{
		Sub:     sub,
		Fetcher: fetcher,
		SubName: cfg.MessagesSub,
		Label:   "primary",
	}}
	return d
}

// AddMessageSource registers an ADDITIONAL inbox-message source (e.g. prod)
// beyond the primary. Task events are NOT pulled from added sources — only
// messages. Each source's Fetcher MUST resolve against that source's project's
// Firestore (see cmd/ailang/daemon.go, where the prod fetcher is scoped to
// ailang-multivac without mutating the shared process env).
func (d *Daemon) AddMessageSource(src MessageSource) {
	d.msgSources = append(d.msgSources, src)
}

// Run blocks until ctx is cancelled, pulling task events from the primary
// source and inbox messages from EVERY registered message source in parallel.
// Returns nil on a clean shutdown; returns the first error from a subscription
// if one fails.
func (d *Daemon) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	// One goroutine per message source, plus one for the primary task events.
	errCh := make(chan error, len(d.msgSources)+1)

	// Task events: primary source only.
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := d.sub.Subscribe(ctx, d.cfg.EventsSub, d.handleTaskEvent)
		if err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("events subscription: %w", err)
		}
	}()

	// Inbox messages: every registered source.
	for _, src := range d.msgSources {
		src := src // capture per-iteration
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := src.Sub.Subscribe(ctx, src.SubName, d.messageHandlerFor(src))
			if err != nil && !errors.Is(err, context.Canceled) {
				errCh <- fmt.Errorf("messages subscription (%s): %w", src.Label, err)
			}
		}()
	}

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
	key := taskDedupKey(t.TaskID, t.Status)
	if d.taskDedup.seen(key) {
		return nil
	}
	if shouldExclude(n, d.cfg.Excludes) {
		return nil
	}
	if err := d.fire(n); err != nil {
		d.taskDedup.forget(key) // nack: let redelivery retry instead of being deduped
		return err
	}
	d.log.Printf("daemon: delivered task event %s/%s -> %q", t.TaskID, t.Status, n.Title)
	return nil
}

// messageHandlerFor returns a MessageHandler bound to src's project-scoped
// fetcher. Dedup is shared across sources (message IDs are globally unique), so
// a message that somehow arrives on two sources fires exactly once.
func (d *Daemon) messageHandlerFor(src MessageSource) MessageHandler {
	return func(ctx context.Context, data []byte, _ map[string]string) error {
		var m pubsub.MessageNotification
		if err := json.Unmarshal(data, &m); err != nil {
			d.log.Printf("daemon: malformed message event (%s): %v", src.Label, err)
			return nil
		}
		key := messageDedupKey(m.MessageID)
		if d.msgDedup.seen(key) {
			return nil
		}
		full, err := src.Fetcher.Fetch(ctx, m.MessageID)
		if err != nil {
			d.msgDedup.forget(key) // nack: let redelivery retry the fetch
			return fmt.Errorf("fetch message %s (%s): %w", m.MessageID, src.Label, err)
		}
		if full == nil {
			// Notification arrived before Firestore replication; nack to retry.
			d.msgDedup.forget(key)
			return fmt.Errorf("message %s not yet visible (%s)", m.MessageID, src.Label)
		}
		n, fire := messageNotification(full)
		if !fire {
			return nil
		}
		if shouldExclude(n, d.cfg.Excludes) {
			return nil
		}
		if err := d.fire(n); err != nil {
			d.msgDedup.forget(key) // nack: let redelivery retry delivery
			return err
		}
		d.log.Printf("daemon: delivered message %s [src=%s, from=%s, inbox=%s] -> %q", m.MessageID, src.Label, full.FromAgent, full.ToInbox, n.Title)
		return nil
	}
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

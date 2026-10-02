// Package daemon implements a long-running consumer that pulls AILANG cloud
// events from Pub/Sub subscriptions and surfaces them as native macOS
// notifications. Designed to run under launchd.
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

	// Notify overrides the daemon's notifier for this source; nil uses it.
	// SplitRemote sets it so per-device sources reach local channels only.
	Notify func(notify.Notification) error
	// DedupScope namespaces this source's dedup keys. The same message
	// arrives on a per-device source AND the shared remote source, and each
	// must deliver it once to its own channels — a shared key would let
	// whichever arrived first swallow the other.
	DedupScope string
}

// Daemon is the running pull loop.
type Daemon struct {
	cfg        Config
	sub        EventSubscriber // primary source's subscriber; task events use this
	msgSources []MessageSource // all inbox-message sources (includes the primary)
	notify     func(notify.Notification) error
	taskDedup  *dedup
	msgDedup   *dedup
	msgAbsent  *unresolved
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
		msgAbsent: newUnresolved(),
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

// SplitRemote moves remote channels (Discord) off the per-device subscriptions
// onto one subscription that every daemon shares.
//
// Each device pulls its own messages subscription so every machine can show
// its own banner, and Pub/Sub gives each subscription a full copy. Discord is
// one channel, so when two daemons could both post to it (from 2026-10-01,
// when the webhook moved to Secret Manager), every message reached it twice.
// A shared subscription work-steals: each message goes to one daemon, so it is
// posted once however many daemons are running.
//
// Every source registered so far, the per-device ones, delivers to local
// only; shared delivers to remote only. Task events keep the full notifier:
// their subscription is already shared.
func (d *Daemon) SplitRemote(shared MessageSource, local, remote func(notify.Notification) error) {
	for i := range d.msgSources {
		d.msgSources[i].Notify = local
	}
	shared.Notify = remote
	shared.DedupScope = "remote"
	d.msgSources = append(d.msgSources, shared)
}

// Run blocks until ctx is cancelled, pulling task events from the primary
// source and inbox messages from EVERY registered message source in parallel.
//
// Every subscription is SUPERVISED: one that returns is restarted with backoff
// and its outage announced. Previously each was started once and its error
// parked in a buffered channel that was only read after all the others exited —
// so a dead messages subscription left the process alive, silent, and receiving
// nothing for five days (see supervise.go).
//
// Returns nil on a clean shutdown.
func (d *Daemon) Run(ctx context.Context) error {
	runners := make([]subscriptionRunner, 0, len(d.msgSources)+1)

	// Task events: primary source only.
	runners = append(runners, subscriptionRunner{
		Label:   "task-events",
		SubName: d.cfg.EventsSub,
		Start: func(ctx context.Context) error {
			return d.sub.Subscribe(ctx, d.cfg.EventsSub, d.handleTaskEvent)
		},
	})

	// Inbox messages: every registered source.
	for _, src := range d.msgSources {
		src := src // capture per-iteration
		runners = append(runners, subscriptionRunner{
			Label:   src.Label,
			SubName: src.SubName,
			Start: func(ctx context.Context) error {
				return src.Sub.Subscribe(ctx, src.SubName, d.messageHandlerFor(src))
			},
		})
	}

	return d.runSupervised(ctx, runners)
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
		key := src.DedupScope + messageDedupKey(m.MessageID)
		if d.msgDedup.seen(key) {
			return nil
		}
		full, err := src.Fetcher.Fetch(ctx, m.MessageID)
		// Absence and failure are different things. Both backends can report
		// absence — Firestore wraps messaging.ErrMessageNotFound, SQLite returns
		// (nil, nil) — and absence is only worth retrying for as long as it could
		// still be replication lag. A retryable failure (network, permission,
		// deadline) is nacked for as long as it keeps failing.
		// A nil message means absence only when the fetch itself succeeded: a
		// failing backend returns (nil, err) too, and reading that as absence is
		// how a Firestore outage would turn into discarded notifications.
		absent := (err == nil && full == nil) || messaging.IsMessageNotFound(err)
		switch {
		case err != nil && !absent:
			d.msgDedup.forget(key) // nack: let redelivery retry the fetch
			d.log.Printf("daemon: RETRY message %s (%s): fetch failed: %v", m.MessageID, src.Label, err)
			return fmt.Errorf("fetch message %s (%s): %w", m.MessageID, src.Label, err)
		case absent:
			d.msgDedup.forget(key)
			// Already judged unresolvable: ack on sight. An ack does not reliably
			// retire the message (30s deadline, no exactly-once, several
			// deliveries in flight), so without the sticky verdict the same
			// message would re-time the full grace window every time it came back.
			if d.msgAbsent.gaveUp(m.MessageID) {
				return nil
			}
			age, firstSighting := d.msgAbsent.age(m.MessageID)
			if age >= absentGrace {
				// Permanently unresolvable. Ack it: a nack here puts it straight
				// back at the head of the backlog, where it blocks every real
				// message behind it (measured: 160,947 nacks to 10 acks).
				d.msgAbsent.giveUp(m.MessageID)
				d.log.Printf("daemon: DROPPING message %s (%s): unresolvable for %s — no such message in the store, acking so it stops blocking the subscription", m.MessageID, src.Label, age.Round(time.Second))
				return nil
			}
			// Log once, not on every redelivery: a poison message cycles several
			// times a second and would bury the traffic that matters.
			if firstSighting {
				d.log.Printf("daemon: RETRY message %s (%s): not yet visible in the store", m.MessageID, src.Label)
			}
			return fmt.Errorf("message %s not yet visible (%s)", m.MessageID, src.Label)
		}
		d.msgAbsent.clear(m.MessageID)
		n, fire := messageNotification(full)
		if !fire {
			return nil
		}
		if shouldExclude(n, d.cfg.Excludes) {
			return nil
		}
		if err := d.fireWith(src.Notify, n); err != nil {
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
	return d.fireWith(nil, n)
}

// fireWith is fire through notifyFn, or through the daemon's notifier when
// notifyFn is nil.
func (d *Daemon) fireWith(notifyFn func(notify.Notification) error, n notify.Notification) error {
	if notifyFn == nil {
		notifyFn = d.notify
	}
	if d.cfg.DryRun {
		d.log.Printf("daemon[dry-run]: %s — %s", n.Title, n.Body)
		return nil
	}
	if err := notifyFn(n); err != nil {
		d.log.Printf("daemon: notify failed: %v", err)
		return err
	}
	return nil
}

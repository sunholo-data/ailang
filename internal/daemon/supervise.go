package daemon

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/notify"
)

// Keeping a subscription alive, and being loud when it is not.
//
// Each subscription used to be started ONCE, in a goroutine, with no restart:
//
//	go func() {
//	    err := src.Sub.Subscribe(ctx, src.SubName, handler) // returns on stream error
//	    if err != nil { errCh <- err }                      // buffered: never blocks
//	}()                                                     // ...and the goroutine exits
//	wg.Wait()                                               // blocks: OTHER goroutines still live
//	close(errCh); for err := range errCh { ... }            // errors read only AFTER that
//
// `Receive` returns on any unrecoverable stream error. When the messages
// subscription hit one, its goroutine exited, the error landed in a BUFFERED
// channel, and `wg.Wait()` kept blocking because the task-events goroutine was
// still alive — so `Run` never returned, the error was never read, and the
// process stayed up with no messages subscription at all.
//
// Measured on the rig, reported 2026-09-11: daemon alive and "running", the
// subscription ACTIVE on Google's side, messages written correctly, and nothing
// delivered for five days. A restart flushed the backlog and re-stalled within
// minutes — a fresh Receive working until the next stream error. The watchdog
// meant to report the fleet being unwell was itself the thing failing silently.
//
// Two changes, and the second is the one that matters:
//
//  1. a subscription that returns is RESTARTED, with capped backoff;
//  2. an outage is ANNOUNCED — through the same notification fan-out the daemon
//     already owns — the moment it happens, rather than being parked in a
//     channel nobody reads until every goroutine has exited.
//
// (1) alone would have turned a five-day silence into a five-day flap, which
// looks healthy from the outside and is arguably worse.

// Vars, not consts, purely so tests can shrink them; production never writes
// these. A restart test that actually waited two minutes would be skipped, and
// a skipped test on a five-day outage is how this went unnoticed once already.
var (
	// superviseMinBackoffVar is the first retry delay. Short, because the common
	// case is a transient stream drop and the cost of retrying is one RPC.
	superviseMinBackoffVar = 1 * time.Second
	// superviseMaxBackoffVar caps it. A permanently-broken subscription must
	// keep retrying — it may be a revoked token that gets restored — but it must
	// not spin.
	superviseMaxBackoffVar = 2 * time.Minute
	// superviseStableAfterVar is how long a subscription must run before its
	// backoff resets. Without it, a subscription that dies immediately on every
	// attempt would retry at the minimum delay forever.
	superviseStableAfterVar = 5 * time.Minute
)

// subscriptionRunner is the subset of a source needed to (re)start a pull.
type subscriptionRunner struct {
	Label   string
	SubName string
	Start   func(ctx context.Context) error
}

// superviseSubscription runs a subscription until ctx is cancelled, restarting
// it whenever it returns.
//
// It never returns an error for the caller to swallow: a dead subscription is
// reported when it dies, not when the process eventually unwinds.
func (d *Daemon) superviseSubscription(ctx context.Context, r subscriptionRunner) {
	backoff := superviseMinBackoffVar
	// Announced once per OUTAGE, not once per retry: a flapping subscription
	// must not turn into a notification storm, which is its own kind of silence.
	announced := false

	for {
		started := time.Now()
		err := r.Start(ctx)

		// A cancelled context is a clean shutdown, not a failure.
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return
		}

		uptime := time.Since(started)
		if uptime >= superviseStableAfterVar {
			// It ran properly, so this is a fresh incident rather than a
			// continuing one: reset both the delay and the announcement.
			backoff = superviseMinBackoffVar
			announced = false
		}

		if err != nil {
			d.log.Printf("daemon: %s subscription (%s) STOPPED after %s: %v — restarting in %s",
				r.Label, r.SubName, uptime.Round(time.Second), err, backoff)
		} else {
			// Receive returning nil without a cancelled context still means the
			// pull has stopped. It is the exact case the old code treated as
			// success and dropped on the floor.
			d.log.Printf("daemon: %s subscription (%s) RETURNED after %s with no error — the pull has stopped; restarting in %s",
				r.Label, r.SubName, uptime.Round(time.Second), backoff)
		}

		if !announced {
			d.announceSubscriptionDown(r, uptime, err)
			announced = true
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		if backoff *= 2; backoff > superviseMaxBackoffVar {
			backoff = superviseMaxBackoffVar
		}
	}
}

// announceSubscriptionDown tells a human the daemon has stopped receiving.
//
// Through the notification fan-out rather than only the log, because the log is
// where the previous five days of this went unread. The fan-out is a different
// mechanism from the subscription — the pull is broken, the outbound path is
// not — so the daemon can still report its own deafness.
func (d *Daemon) announceSubscriptionDown(r subscriptionRunner, uptime time.Duration, cause error) {
	reason := "the pull returned with no error"
	if cause != nil {
		reason = cause.Error()
	}
	n := notify.Notification{
		// public-feedback is the EventType Discord's allow-list accepts. A
		// daemon that cannot report its own outage to the channel a human
		// watches is not reporting it at all.
		EventType: "public-feedback",
		Title:     "🚨 Notifier subscription DOWN",
		Body: fmt.Sprintf("%s (%s) stopped receiving after %s: %s\nRestarting with backoff; messages are queued, not lost.",
			r.Label, r.SubName, uptime.Round(time.Second), reason),
	}
	if err := d.fire(n); err != nil {
		// Nothing left to escalate to; the log is the last resort and says so.
		d.log.Printf("daemon: could not announce the %s subscription outage (%v) — the outage itself stands", r.Label, err)
	}
}

// runSupervised starts every subscription under supervision and blocks until
// ctx is cancelled.
//
// Returns nil: with restart-on-failure there is no longer a "first error" worth
// aborting the process for, and returning one would take the OTHER, healthy
// subscriptions down with it — the failure mode this replaces, inverted.
func (d *Daemon) runSupervised(ctx context.Context, runners []subscriptionRunner) error {
	var wg sync.WaitGroup
	for _, r := range runners {
		r := r
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.superviseSubscription(ctx, r)
		}()
	}
	wg.Wait()
	return nil
}

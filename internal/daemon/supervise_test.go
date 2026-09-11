package daemon

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/notify"
)

// The bug these pin, exactly as measured on the rig 2026-09-11: a subscription
// whose Receive returned was never restarted, and its error sat in a buffered
// channel that was only drained after EVERY goroutine exited — so one dead
// subscription meant a live process, no messages, and total silence for five
// days. Every test here fails against that code.

// fastBackoff shrinks the delays so a restart test runs in milliseconds. It
// restores the originals, so it cannot leak into another test.
func fastBackoff(t *testing.T) {
	t.Helper()
	minB, maxB, stable := superviseMinBackoffVar, superviseMaxBackoffVar, superviseStableAfterVar
	superviseMinBackoffVar = time.Millisecond
	superviseMaxBackoffVar = 2 * time.Millisecond
	superviseStableAfterVar = time.Hour // never "stable" in a fast test
	t.Cleanup(func() {
		superviseMinBackoffVar, superviseMaxBackoffVar, superviseStableAfterVar = minB, maxB, stable
	})
}

// quietLogger keeps the restart chatter out of test output; the assertions are
// on behaviour, not on what got logged.
func quietLogger() *log.Logger { return log.New(io.Discard, "", 0) }

func testDaemon(t *testing.T, fired *[]notify.Notification, mu *sync.Mutex) *Daemon {
	t.Helper()
	d := New(Config{EventsSub: "events", MessagesSub: "messages"}, nil, nil,
		func(n notify.Notification) error {
			mu.Lock()
			defer mu.Unlock()
			*fired = append(*fired, n)
			return nil
		})
	d.log = quietLogger()
	return d
}

func TestSupervise_RestartsASubscriptionThatReturns(t *testing.T) {
	fastBackoff(t)
	var mu sync.Mutex
	var fired []notify.Notification
	d := testDaemon(t, &fired, &mu)

	var starts int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		d.superviseSubscription(ctx, subscriptionRunner{
			Label: "messages", SubName: "messages-rig",
			Start: func(context.Context) error {
				// Dies immediately, every time — the rig's observed behaviour
				// once the stream had gone.
				if atomic.AddInt32(&starts, 1) >= 4 {
					cancel()
				}
				return errors.New("stream closed by server")
			},
		})
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("supervision did not stop on context cancel")
	}

	if got := atomic.LoadInt32(&starts); got < 4 {
		t.Errorf("subscription started %d times, want it RESTARTED (>=4); the old code started it once", got)
	}
}

func TestSupervise_AnnouncesTheOutageImmediately(t *testing.T) {
	fastBackoff(t)
	var mu sync.Mutex
	var fired []notify.Notification
	d := testDaemon(t, &fired, &mu)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var starts int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		d.superviseSubscription(ctx, subscriptionRunner{
			Label: "messages", SubName: "messages-rig",
			Start: func(context.Context) error {
				if atomic.AddInt32(&starts, 1) >= 3 {
					cancel()
				}
				return errors.New("stream closed by server")
			},
		})
	}()
	<-done

	mu.Lock()
	defer mu.Unlock()
	if len(fired) == 0 {
		t.Fatal("a dead subscription must be ANNOUNCED, not parked in a channel nobody reads")
	}
	// Once per outage, not once per retry: a flapping subscription that storms
	// the channel is its own kind of silence.
	if len(fired) != 1 {
		t.Errorf("announced %d times across %d restarts, want exactly 1 per outage", len(fired), atomic.LoadInt32(&starts))
	}
	n := fired[0]
	// It has to reach Discord, whose allow-list keys on EventType — an outage
	// that only reaches macOS is invisible when nobody is at the machine.
	if n.EventType != "public-feedback" {
		t.Errorf("EventType = %q; the outage must use the type Discord accepts", n.EventType)
	}
	for _, want := range []string{"messages-rig", "stream closed by server"} {
		if !strings.Contains(n.Body+n.Title, want) {
			t.Errorf("the announcement must name %q so it can be acted on:\n%s\n%s", want, n.Title, n.Body)
		}
	}
}

func TestSupervise_ReturnWithNoErrorIsStillAnOutage(t *testing.T) {
	fastBackoff(t)
	var mu sync.Mutex
	var fired []notify.Notification
	d := testDaemon(t, &fired, &mu)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var starts int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		d.superviseSubscription(ctx, subscriptionRunner{
			Label: "messages", SubName: "messages-rig",
			Start: func(context.Context) error {
				if atomic.AddInt32(&starts, 1) >= 2 {
					cancel()
				}
				// nil, not an error. The old code treated this as success and
				// let the goroutine exit — the pull stops and nothing says so.
				return nil
			},
		})
	}()
	<-done

	if got := atomic.LoadInt32(&starts); got < 2 {
		t.Errorf("a nil return still means the pull stopped; want a restart, got %d start(s)", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(fired) == 0 {
		t.Error("a silent stop is the worst case and must still be announced")
	}
}

func TestSupervise_CleanShutdownIsNotAnOutage(t *testing.T) {
	fastBackoff(t)
	var mu sync.Mutex
	var fired []notify.Notification
	d := testDaemon(t, &fired, &mu)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: an ordinary shutdown

	d.superviseSubscription(ctx, subscriptionRunner{
		Label: "messages", SubName: "messages-rig",
		Start: func(context.Context) error { return context.Canceled },
	})

	mu.Lock()
	defer mu.Unlock()
	if len(fired) != 0 {
		t.Errorf("a clean shutdown must not page anyone; got %d notification(s)", len(fired))
	}
}

func TestRunSupervised_OneDeadSubscriptionDoesNotStopTheOthers(t *testing.T) {
	fastBackoff(t)
	var mu sync.Mutex
	var fired []notify.Notification
	d := testDaemon(t, &fired, &mu)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var healthyRunning atomic.Bool
	var deadStarts int32

	go func() {
		// Let both settle, then shut down.
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	err := d.runSupervised(ctx, []subscriptionRunner{
		{Label: "task-events", SubName: "events", Start: func(ctx context.Context) error {
			// The healthy one: blocks until shutdown, as a real Receive does.
			healthyRunning.Store(true)
			<-ctx.Done()
			return context.Canceled
		}},
		{Label: "messages", SubName: "messages-rig", Start: func(context.Context) error {
			atomic.AddInt32(&deadStarts, 1)
			return errors.New("stream closed by server")
		}},
	})

	if err != nil {
		t.Errorf("a supervised shutdown returns nil, got %v", err)
	}
	if !healthyRunning.Load() {
		t.Error("the healthy subscription should have run")
	}
	// The whole point: the dead one kept being retried while the other lived.
	// The old code exited its goroutine once and blocked forever in wg.Wait().
	if got := atomic.LoadInt32(&deadStarts); got < 2 {
		t.Errorf("the dead subscription was started %d time(s); it must keep restarting alongside the healthy one", got)
	}
}

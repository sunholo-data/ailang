package motoko

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// A run whose session log stops growing is killed after the idle timeout —
// the 2026-10-08 stall waited out the 40-minute wall clock instead.
func TestWatchIdle_SilenceFiresOnce(t *testing.T) {
	var fired atomic.Int32
	done := make(chan struct{})
	go func() {
		watchIdle(context.Background(), 30*time.Millisecond, 5*time.Millisecond,
			func() int64 { return 42 }, func() { fired.Add(1) })
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watchIdle never returned on a silent run")
	}
	if fired.Load() != 1 {
		t.Fatalf("onIdle fired %d times, want 1", fired.Load())
	}
}

// Steady output (stream deltas, tool events) keeps the run alive.
func TestWatchIdle_ActivityKeepsAlive(t *testing.T) {
	var n atomic.Int64
	var fired atomic.Bool
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	watchIdle(ctx, 30*time.Millisecond, 5*time.Millisecond,
		func() int64 { return n.Add(1) }, func() { fired.Store(true) })
	if fired.Load() {
		t.Fatal("onIdle fired although the activity counter kept growing")
	}
}

// startIdleWatch cancels the run on idle and reports it; a run that ends first
// reports nothing.
func TestStartIdleWatch(t *testing.T) {
	saved := idlePollInterval
	idlePollInterval = 5 * time.Millisecond
	defer func() { idlePollInterval = saved }()
	runCtx, cancelRun := context.WithCancel(context.Background())
	stop := startIdleWatch(runCtx, 30*time.Millisecond, func() int64 { return 0 }, cancelRun)
	select {
	case <-runCtx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("idle run was not cancelled")
	}
	if note := stop(); !strings.Contains(note, "idle timeout") {
		t.Fatalf("stop() = %q, want an idle-timeout note", note)
	}

	quiet, cancelQuiet := context.WithCancel(context.Background())
	defer cancelQuiet()
	if note := startIdleWatch(quiet, time.Hour, func() int64 { return 0 }, cancelQuiet)(); note != "" {
		t.Fatalf("a run stopped before idling reported %q", note)
	}
}

package motoko

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// defaultIdleTimeout applies when the task sets none. Longer than pi's 3m:
// motoko runs tool calls (ailang_run is capped at 120s by the lane policy) and
// a compaction summary call that write nothing to the session log meanwhile.
const defaultIdleTimeout = 10 * time.Minute

// idlePollInterval is how often the session log is sampled.
var idlePollInterval = 10 * time.Second

// watchIdle calls onIdle once and returns when activity() has not changed for
// idle. activity is a monotone progress counter (bytes written); it is sampled
// every poll. It returns without calling onIdle when ctx ends first.
//
// The executor never read task.IdleTimeout: only the wall-clock bound could end
// a stalled run. Measured 2026-10-08: two lane tasks stalled mid-stream (31
// stream starts, 30 ends) and waited out the full 40-minute bound, where pi's
// executor would have killed them after its idle timeout.
func watchIdle(ctx context.Context, idle, poll time.Duration, activity func() int64, onIdle func()) {
	last := activity()
	lastChange := time.Now()
	t := time.NewTicker(poll)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if now := activity(); now != last {
				last, lastChange = now, time.Now()
				continue
			}
			if time.Since(lastChange) >= idle {
				onIdle()
				return
			}
		}
	}
}

// sessionActivity is the progress counter for a motoko run: the bytes in its
// session JSONL (motoko appends every stream delta and tool event) plus its
// stderr log. The JSONL path is looked up each time because motoko creates the
// file only once its logger starts.
func sessionActivity(workspace, sessionID, repo, stderrLogPath string) func() int64 {
	return func() int64 {
		var n int64
		if p, err := findSessionJSONL(workspace, sessionID, repo); err == nil {
			if fi, err := os.Stat(p); err == nil {
				n += fi.Size()
			}
		}
		if fi, err := os.Stat(stderrLogPath); err == nil {
			n += fi.Size()
		}
		return n
	}
}

// startIdleWatch runs watchIdle for one motoko run: on idle it cancels the
// run's parent context (cancelRun), so the process-group kill takes it down.
// The returned stop ends the watch and reports, as an error prefix, whether the
// run was killed for idling ("" when it was not).
func startIdleWatch(runCtx context.Context, idle time.Duration, activity func() int64, cancelRun context.CancelFunc) (stop func() string) {
	if idle <= 0 {
		idle = defaultIdleTimeout
	}
	var killed atomic.Bool
	watchCtx, stopWatch := context.WithCancel(runCtx)
	go watchIdle(watchCtx, idle, idlePollInterval, activity, func() {
		killed.Store(true)
		cancelRun()
	})
	return func() string {
		stopWatch()
		if killed.Load() {
			return fmt.Sprintf("motoko idle for %v (no session or stderr output): idle timeout — ", idle)
		}
		return ""
	}
}

package riglock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Cooperative yield (M-RIG-LOCK-YIELD).
//
// The rig lock had exactly two modes — Wait and NoWait — and no priority. That
// made a 10-hour batch job and a 40-second interactive classification equals,
// and the batch always won: it arrived first and never let go. Measured
// 2026-09-11: the nightly eval held the lock 03:00→~13:00 while Daneel's mail
// intake deferred on 83% of its runs (34/41) and the os-rotation-filler got one
// acquisition in a day. Nothing was broken; the protocol simply had no way to
// ASK a holder to step aside, which tools/daneel documented as filed-upstream.
//
// This is that ask. A short-job requester writes a HANDOFF marker naming itself
// and a deadline; the holder notices it at a checkpoint where the GPU is idle,
// releases, waits for the requester to finish, and re-acquires. Two properties
// keep it honest:
//
//   - The marker lives OUTSIDE the lock directory, so it survives the release.
//     Without that, the gap the holder opens is just a free lock, and the
//     background filler — which polls every 45 minutes with NoWait — would win
//     the race and hold it for a full chunk. NoWait acquirers that are not the
//     named requester are refused while a handoff is in force.
//   - Every handoff carries an expiry and a pid. A requester that dies between
//     asking and acquiring cannot strand the holder, and an expired marker is
//     removed by the next reader rather than blocking NoWait forever.
//
// The holder never blocks on a checkpoint when nothing is pending: the fast
// path is one stat of a file that usually does not exist.

const (
	// EnvHandoffFile overrides the handoff marker path (mirrors rig-lock.sh
	// RIG_HANDOFF_FILE). Defaults to rig.handoff beside the lock directory.
	EnvHandoffFile = "RIG_HANDOFF_FILE"

	// DefaultYieldWindow bounds how long a handoff stays in force. Sized for
	// the case it exists to serve — Daneel's intake classification is ~40s of
	// GPU — with generous headroom for a cold model load. A requester that
	// needs longer than this is a batch job and should queue like one.
	DefaultYieldWindow = 3 * time.Minute

	// yieldPollInterval is how often a yielding holder re-checks the handoff.
	// Short relative to the window: the whole point is to hand the GPU back
	// promptly once the short job is done.
	yieldPollInterval = 2 * time.Second
)

// checkpointMu serializes Checkpoint across goroutines in one process. Two
// callers releasing and re-acquiring the same lock directory concurrently would
// interleave into a state where one removes the directory the other just took.
var checkpointMu sync.Mutex

// Yield describes an in-force handoff request.
type Yield struct {
	Requester string
	PID       int
	Until     time.Time
}

func handoffPath() string {
	if p := os.Getenv(EnvHandoffFile); p != "" {
		return p
	}
	return filepath.Join(filepath.Dir(lockDir()), "rig.handoff")
}

// RequestYield asks the current lock holder to release at its next checkpoint.
// It does NOT acquire the lock — the caller still races for it normally once
// the holder steps aside, and should pass its own name to AcquireAs so the
// handoff guard does not refuse the very requester it exists to serve.
//
// window <= 0 uses DefaultYieldWindow. Re-requesting extends an existing
// handoff owned by the same requester rather than stacking a second one.
func RequestYield(requester string, window time.Duration) error {
	requester = strings.TrimSpace(requester)
	if requester == "" {
		return fmt.Errorf("riglock: RequestYield needs a requester name")
	}
	if strings.ContainsAny(requester, " \t\n") {
		return fmt.Errorf("riglock: requester name must not contain whitespace: %q", requester)
	}
	if window <= 0 {
		window = DefaultYieldWindow
	}
	path := handoffPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("riglock: cannot create state dir: %w", err)
	}
	// A handoff owned by SOMEONE ELSE is left alone: two short jobs queueing
	// behind one holder is fine, but the second must not silently inherit the
	// first one's grant and walk into a GPU the holder released for someone
	// else. It falls back to ordinary contention.
	if y, ok := PendingYield(); ok && y.Requester != requester {
		return fmt.Errorf("riglock: a handoff to %q is already in force until %s",
			y.Requester, y.Until.Format(time.RFC3339))
	}
	line := fmt.Sprintf("requester=%s pid=%d until=%s\n",
		requester, os.Getpid(), time.Now().Add(window).UTC().Format(time.RFC3339))
	return os.WriteFile(path, []byte(line), 0o644)
}

// ClearYield removes a handoff owned by requester. It is the requester's duty
// to call this as soon as it is done, which is what hands the GPU back early
// instead of making the holder wait out the whole window. Safe to call when no
// handoff exists, and a no-op against a handoff owned by anyone else.
func ClearYield(requester string) {
	y, ok := PendingYield()
	if !ok || y.Requester != strings.TrimSpace(requester) {
		return
	}
	_ = os.Remove(handoffPath())
}

// PendingYield returns the handoff currently in force. A marker that is
// expired, unparseable, or names a dead pid is treated as absent AND removed:
// leaving one of those in place would refuse every NoWait acquirer until a
// human noticed, which is a worse failure than the contention it prevents.
func PendingYield() (Yield, bool) {
	path := handoffPath()
	b, err := os.ReadFile(path)
	if err != nil {
		return Yield{}, false
	}
	y, parseErr := parseYield(string(b))
	if parseErr != nil || time.Now().After(y.Until) || (y.PID > 0 && !pidAlive(y.PID)) {
		_ = os.Remove(path)
		return Yield{}, false
	}
	return y, true
}

func parseYield(s string) (Yield, error) {
	var y Yield
	for _, f := range strings.Fields(strings.TrimSpace(s)) {
		k, v, found := strings.Cut(f, "=")
		if !found {
			continue
		}
		switch k {
		case "requester":
			y.Requester = v
		case "pid":
			y.PID, _ = strconv.Atoi(v)
		case "until":
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return Yield{}, fmt.Errorf("riglock: bad handoff until=%q: %w", v, err)
			}
			y.Until = t
		}
	}
	if y.Requester == "" || y.Until.IsZero() {
		return Yield{}, fmt.Errorf("riglock: incomplete handoff marker %q", strings.TrimSpace(s))
	}
	return y, nil
}

// Checkpoint is called by a lock HOLDER at a point where the GPU is idle — for
// eval-suite, between benchmarks. If a handoff is pending it releases the lock,
// waits for the requester to take and finish with it, then re-acquires and
// returns (true, elapsed). With nothing pending it returns (false, 0) after a
// single stat, so it is cheap enough to call in a loop.
//
// holder is this job's name; a handoff we requested ourselves is ignored.
//
// Callers MUST only invoke this when no work of their own is in flight. The
// lock says "nobody else is driving the GPU", and a checkpoint that fires while
// a benchmark is still streaming hands out a promise that is already false —
// which is why eval-suite checkpoints only at --parallel 1.
//
// This works whether the lock is held by this process or by an ancestor shell
// (AILANG_RIG_LOCK_HELD=1): the lock is a directory, not a process handle, so
// re-acquiring writes a fresh holder file and the ancestor's exit trap still
// removes it at the end of the job.
func Checkpoint(holder string) (bool, time.Duration) {
	// Only a holder may yield. EnvHeld is set both by an ancestor shell and by
	// this package's own Acquire, and cleared on release, so it is the one
	// signal that covers every way we can be holding — and, just as important,
	// keeps a cloud-only or --no-rig-lock run from releasing a lock it never
	// took. Cheap enough to sit in front of the stat.
	if !HeldByAncestor() {
		return false, 0
	}
	y, ok := PendingYield()
	if !ok || y.Requester == strings.TrimSpace(holder) {
		return false, 0
	}

	checkpointMu.Lock()
	defer checkpointMu.Unlock()
	// Re-read under the mutex: a sibling goroutine may have served this same
	// handoff while we were blocked, and yielding twice for one request would
	// give the filler a second gap to race into.
	if y, ok = PendingYield(); !ok || y.Requester == strings.TrimSpace(holder) {
		return false, 0
	}

	start := time.Now()
	_ = os.RemoveAll(lockDir())

	// Wait for the requester to finish. It signals that by clearing the
	// handoff; PendingYield also reports absent once the marker expires or the
	// requester's pid dies, so neither a crash nor an overrun strands us here.
	for {
		if _, still := PendingYield(); !still {
			break
		}
		time.Sleep(yieldPollInterval)
	}

	// Wait, not NoWait: the requester may still be finishing the run it took
	// the lock for, and the whole point was to let it complete.
	_, _, err := acquireDir(Wait)
	if err != nil {
		// Re-acquire failed (unwritable state dir). Say so rather than carrying
		// on as if we still held a lock we do not — a silent false hold is how
		// two jobs end up on one GPU.
		fmt.Fprintf(os.Stderr, "Warning: rig lock NOT re-acquired after yielding to %s: %v\n", y.Requester, err)
	}
	return true, time.Since(start)
}

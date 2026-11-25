package riglock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Reclaim marker: the yielding holder's claim on the gap it opened.
//
// A yield (yield.go) LENDS the GPU. The holder removes the lock directory, waits
// for the borrower to finish, then takes the directory back. Before this marker
// existed, "takes it back" was an ordinary mkdir race. The holder polled on the
// same 30-second loop as every other waiter, so any job that was queued in Wait
// mode, or the filler firing NoWait at the wrong moment, could take the lock
// the moment the borrower left. The holder then waited out that job's entire
// run. Measured 2026-11-25: a fine-tune eval (pilot5) yielded to a short job,
// an attended Inkling test that had been waiting took the gap, and the
// fine-tune sat idle for 2h42m. The same gap opened whenever a holder lent the
// GPU to the operator (operator.go).
//
// The holder writes this marker BEFORE it releases. While it is in force, every
// acquirer except the borrower a handoff names is refused (NoWait) or queues
// (Wait), and the holder re-acquires through acquireDir, which skips the guard.
// Like the handoff marker it lives OUTSIDE the lock directory, because the
// directory is exactly what is absent during the gap. tools/launchd/rig-lock.sh
// reads the same file and line format.
//
// A marker whose pid is dead, that has expired, or that cannot be parsed is
// treated as absent and removed. A crashed holder must not leave the rig
// refusing everyone. The expiry is the lock's own staleness window: a holder
// that has been lending longer than that is treated like a stale holder.

// Reclaim describes an in-force reclaim claim.
type Reclaim struct {
	Holder string
	PID    int
	Until  time.Time
}

func reclaimPath() string {
	return filepath.Join(filepath.Dir(handoffPath()), "rig.reclaim")
}

// writeReclaim records that this process is lending the lock and will take it back.
func writeReclaim(holder string) error {
	holder = strings.Join(strings.Fields(holder), "-")
	if holder == "" {
		holder = "holder"
	}
	line := fmt.Sprintf("holder=%s pid=%d until=%s\n",
		holder, os.Getpid(), time.Now().Add(staleWindow()).UTC().Format(time.RFC3339))
	return os.WriteFile(reclaimPath(), []byte(line), 0o644)
}

// clearReclaim removes the marker if this process wrote it.
func clearReclaim() {
	b, err := os.ReadFile(reclaimPath())
	if err != nil {
		return
	}
	if r, perr := parseReclaim(string(b)); perr == nil && r.PID != os.Getpid() {
		return // someone else's claim; not ours to drop
	}
	_ = os.Remove(reclaimPath())
}

// PendingReclaim returns the reclaim claim currently in force.
func PendingReclaim() (Reclaim, bool) {
	path := reclaimPath()
	b, err := os.ReadFile(path)
	if err != nil {
		return Reclaim{}, false
	}
	r, parseErr := parseReclaim(string(b))
	if parseErr != nil || time.Now().After(r.Until) || (r.PID > 0 && !pidAlive(r.PID)) {
		_ = os.Remove(path)
		return Reclaim{}, false
	}
	return r, true
}

func parseReclaim(s string) (Reclaim, error) {
	var r Reclaim
	for _, f := range strings.Fields(strings.TrimSpace(s)) {
		k, v, found := strings.Cut(f, "=")
		if !found {
			continue
		}
		switch k {
		case "holder":
			r.Holder = v
		case "pid":
			r.PID, _ = strconv.Atoi(v)
		case "until":
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return Reclaim{}, fmt.Errorf("riglock: bad reclaim until=%q: %w", v, err)
			}
			r.Until = t
		}
	}
	if r.Holder == "" || r.PID <= 0 || r.Until.IsZero() {
		return Reclaim{}, fmt.Errorf("riglock: incomplete reclaim marker %q", strings.TrimSpace(s))
	}
	return r, nil
}

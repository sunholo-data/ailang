package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The fleet-wide quota ledger.
//
// Routing today asks "is this lane up?" and never "can it afford to be used?". A
// 1-token probe answers the first; nothing answers the second, and on 2026-09-06 that
// cost half a codex bucket in a day.
//
// FLEET-WIDE (D-3), because that is what the bucket physically is: four missions drawing
// on one subscription. Per-mission accounting would let every loop believe it was within
// budget while the bucket emptied.

// Window is one of a bucket's limit periods. Both Anthropic and codex enforce TWO — a
// short rolling window and a longer cap — and they fail differently: codex's 5-hour
// window kept refilling on 2026-09-06 while its long cap stayed spent until the 12th,
// and Anthropic's long cap produced a 16-hour drought on 2026-08-16 (45 fires refused)
// that no 5-hour window can explain.
type Window string

const (
	// WindowShort is the ~5-hour rolling window.
	WindowShort Window = "5h"
	// WindowLong is the longer cap — the one the daily ration paces.
	WindowLong Window = "long"
)

// DailyRationFraction is D-1: spend at most 10% of a bucket per day.
//
// 100/7 = 14.3% would spend a weekly bucket exactly, with no margin. 10% leaves 30% of
// the week in reserve, and the reserve is not decoration — the measured bad day spent
// 50% of codex in one go.
const DailyRationFraction = 0.10

// windowDuration is how long a window lasts.
func windowDuration(w Window) time.Duration {
	if w == WindowShort {
		return 5 * time.Hour
	}
	return 7 * 24 * time.Hour
}

// rationApplies reports whether the daily ration paces this window.
//
// It paces the LONG window only. The daily fraction is a weekly-bucket pacing rule
// (the user derived it as 100/7); the 5-hour window is a REFILL RATE the provider
// enforces itself, and there are ~4.8 of them in a day. Applying 10%/day pro-rata to a
// 5h window would cap each one at ~2% of its capacity and idle the fleet for a limit
// nobody imposed. The short window is still tracked — it is what M5's controller
// reserve is computed against, and what makes a mid-window exhaustion legible — but it
// is not rationed by this rule, and Verdict says so rather than silently passing.
func rationApplies(w Window) bool { return w == WindowLong }

// BucketUsage is consumption of one bucket within one window.
type BucketUsage struct {
	Bucket      string    `json:"bucket"`
	Window      Window    `json:"window"`
	Tokens      int64     `json:"tokens"`
	Stages      int       `json:"stages"`
	WindowStart time.Time `json:"window_start"`
	// Capacity is the bucket's limit for this window. ZERO MEANS UNKNOWN, and an
	// unknown capacity is never rationed against — see Verdict. M3 fills this from
	// the provider endpoint; until then every bucket reads UNRATIONED and loud.
	Capacity int64 `json:"capacity,omitempty"`
	// ProviderReset is a reset instant the PROVIDER reported (codex prints "resets
	// 05:34"; Anthropic reports its own). ZERO MEANS UNKNOWN.
	//
	// When known it defines the window boundaries, and WindowStart is snapped to the
	// boundary containing the spend rather than derived from when we happened to
	// first record something. Deriving boundaries from our own first write is a
	// wall-clock guess that drifts from the limit actually being enforced — the
	// fleet would reset its counter hours away from the provider resetting the
	// bucket, and ration against a window nobody has.
	ProviderReset time.Time `json:"provider_reset,omitempty"`
}

// windowStartAt returns the start of the window containing `at`.
//
// With a provider reset known, boundaries are that instant stepped by whole window
// durations — so our window and the provider's are the same window. Without one, the
// caller's own start stands, and the ledger is honest that it is a local derivation.
func (u BucketUsage) windowStartAt(at time.Time) (time.Time, bool) {
	if u.ProviderReset.IsZero() {
		return time.Time{}, false
	}
	dur := windowDuration(u.Window)
	delta := at.Sub(u.ProviderReset)
	steps := int64(delta / dur)
	if delta < 0 && delta%dur != 0 {
		steps-- // floor toward -inf so a spend before the reset lands in the prior window
	}
	return u.ProviderReset.Add(time.Duration(steps) * dur), true
}

// Ledger is the fleet-wide record.
type Ledger struct {
	Updated time.Time     `json:"updated"`
	Usage   []BucketUsage `json:"usage"`
}

// RationState is what the ration rule concluded about one (bucket, window).
type RationState string

const (
	// RationOK — within the pro-rata allowance.
	RationOK RationState = "ok"
	// RationOver — the allowance is spent. M4 skips this rung.
	RationOver RationState = "over"
	// RationCapacityUnknown — no capacity for this bucket, so no allowance can be
	// computed. NOT a pass and NOT a fail: rationing on a guessed capacity would
	// either idle a healthy fleet or wave through an empty bucket, confidently.
	RationCapacityUnknown RationState = "capacity-unknown"
	// RationNotPaced — this window is not paced by the daily ration (the 5h window).
	RationNotPaced RationState = "not-paced"
)

// RationVerdict answers "can this bucket afford to be used right now?".
type RationVerdict struct {
	Bucket    string
	Window    Window
	State     RationState
	Consumed  int64
	Allowance int64
	Reason    string
}

// Over is the gate M4 calls. It is true ONLY for a computed exceedance — an unknown
// capacity or an unpaced window is never reported as over.
func (v RationVerdict) Over() bool { return v.State == RationOver }

// Verdict evaluates the daily ration against elapsed window time.
func (u BucketUsage) Verdict(now time.Time) RationVerdict {
	v := RationVerdict{Bucket: u.Bucket, Window: u.Window, Consumed: u.Tokens}

	if !rationApplies(u.Window) {
		v.State = RationNotPaced
		v.Reason = fmt.Sprintf("%d tok this %s window; the daily ration paces the %s window only",
			u.Tokens, u.Window, WindowLong)
		return v
	}
	if u.Capacity <= 0 {
		v.State = RationCapacityUnknown
		v.Reason = fmt.Sprintf("%d tok consumed, capacity unknown — left unrationed rather than rationed on a guess", u.Tokens)
		return v
	}

	dur := windowDuration(u.Window)
	elapsed := now.Sub(u.WindowStart)
	if elapsed < 0 {
		// Clock skew, or a window recorded in the future. Treat as the start of the
		// window: a skewed clock must not pause the fleet.
		elapsed = 0
	}
	if elapsed > dur {
		elapsed = dur
	}

	perDay := float64(u.Capacity) * DailyRationFraction
	allowance := perDay * (elapsed.Hours() / 24)
	// Floor at one day's ration, so the first hours of a window are not effectively a
	// zero budget that pauses every rung the moment a window rolls over.
	if allowance < perDay {
		allowance = perDay
	}
	v.Allowance = int64(allowance)

	if u.Tokens > v.Allowance {
		v.State = RationOver
		v.Reason = fmt.Sprintf("%d tok consumed vs %d allowed (%.0f%%/day, %s elapsed of %s window)",
			u.Tokens, v.Allowance, DailyRationFraction*100, elapsed.Round(time.Minute), u.Window)
	} else {
		v.State = RationOK
		v.Reason = fmt.Sprintf("%d of %d allowed (%.0f%%/day)", u.Tokens, v.Allowance, DailyRationFraction*100)
	}
	return v
}

// Get returns usage for a (bucket, window), and whether it was recorded.
func (l *Ledger) Get(bucket string, w Window) (BucketUsage, bool) {
	for _, u := range l.Usage {
		if u.Bucket == bucket && u.Window == w {
			return u, true
		}
	}
	return BucketUsage{Bucket: bucket, Window: w}, false
}

// Verdicts evaluates every recorded (bucket, window).
func (l *Ledger) Verdicts(now time.Time) []RationVerdict {
	out := make([]RationVerdict, 0, len(l.Usage))
	for _, u := range l.Usage {
		out = append(out, u.Verdict(now))
	}
	return out
}

// OverRation reports whether a bucket has exceeded its ration in any window, with the
// reason. Any window over is enough: the binding constraint is the tighter one.
func (l *Ledger) OverRation(bucket string, now time.Time) (bool, string) {
	for _, u := range l.Usage {
		if u.Bucket != bucket {
			continue
		}
		if v := u.Verdict(now); v.Over() {
			return true, fmt.Sprintf("%s/%s: %s", v.Bucket, v.Window, v.Reason)
		}
	}
	return false, ""
}

// add folds one spend event in, rolling the window over when the recorded one expired.
//
// Rolling on TOUCH rather than on a timer is deliberate: the ledger is written by
// whichever mission fires next and there is no daemon to tick it. A window that expired
// while the fleet was idle must reset when it is next read, or the first fire after a
// quiet night inherits a full window's consumption and pauses itself.
func (l *Ledger) add(bucket string, w Window, tokens int64, stages int, at time.Time) {
	for i := range l.Usage {
		u := &l.Usage[i]
		if u.Bucket != bucket || u.Window != w {
			continue
		}
		if start, ok := u.windowStartAt(at); ok {
			// Provider-defined boundary: reset when this spend belongs to a LATER
			// window than the one recorded, never on elapsed time we measured.
			if start.After(u.WindowStart) {
				u.WindowStart = start
				u.Tokens = 0
				u.Stages = 0
			}
		} else if at.Sub(u.WindowStart) >= windowDuration(w) {
			u.WindowStart = at
			u.Tokens = 0
			u.Stages = 0
		}
		u.Tokens += tokens
		u.Stages += stages
		return
	}
	u := BucketUsage{Bucket: bucket, Window: w, Tokens: tokens, Stages: stages, WindowStart: at}
	if start, ok := u.windowStartAt(at); ok {
		u.WindowStart = start
	}
	l.Usage = append(l.Usage, u)
}

// SetCapacity records a bucket's known capacity for a window (M3 supplies this).
func (l *Ledger) SetCapacity(bucket string, w Window, capacity int64, now time.Time) {
	for i := range l.Usage {
		if l.Usage[i].Bucket == bucket && l.Usage[i].Window == w {
			l.Usage[i].Capacity = capacity
			return
		}
	}
	l.Usage = append(l.Usage, BucketUsage{
		Bucket: bucket, Window: w, Capacity: capacity, WindowStart: now,
	})
}

// SetProviderReset pins a bucket's window boundaries to a provider-reported reset
// instant (M3 supplies these).
func (l *Ledger) SetProviderReset(bucket string, w Window, reset time.Time, now time.Time) {
	for i := range l.Usage {
		if l.Usage[i].Bucket == bucket && l.Usage[i].Window == w {
			l.Usage[i].ProviderReset = reset
			return
		}
	}
	l.Usage = append(l.Usage, BucketUsage{
		Bucket: bucket, Window: w, ProviderReset: reset, WindowStart: now,
	})
}

// BoundarySource says where a window's boundaries come from, so a reader can tell a
// provider-anchored window from a locally-derived one without inferring it.
func (u BucketUsage) BoundarySource() string {
	if u.ProviderReset.IsZero() {
		return "local"
	}
	return "provider"
}

// sortUsage gives the file and the CLI a stable order.
func (l *Ledger) sortUsage() {
	sort.Slice(l.Usage, func(i, j int) bool {
		if l.Usage[i].Bucket != l.Usage[j].Bucket {
			return l.Usage[i].Bucket < l.Usage[j].Bucket
		}
		return l.Usage[i].Window < l.Usage[j].Window
	})
}

// String renders a ledger for the CLI.
func (l *Ledger) String(now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "quota ledger (fleet-wide, %.0f%%/day on the %s window)\n",
		DailyRationFraction*100, WindowLong)
	if len(l.Usage) == 0 {
		b.WriteString("  (no spend recorded)\n")
		return b.String()
	}
	for _, u := range l.Usage {
		v := u.Verdict(now)
		fmt.Fprintf(&b, "  %-12s %-5s %-16s %s\n", u.Bucket, u.Window, v.State, v.Reason)
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Persistence.
//
// The write path is an APPEND JOURNAL, not a locked read-modify-write, and that is the
// whole answer to "a held lock must never wedge a fire". A fire appends one short line
// with O_APPEND (atomic on POSIX below PIPE_BUF) and takes no lock at all, so there is
// no lock for it to block on and nothing to fail open about — the spend is durable the
// instant it is recorded.
//
// Consolidation (journal -> ledger.json) is the only step that needs exclusion, and it
// is pure bookkeeping: skipping it loses nothing, because a reader folds any
// unconsolidated journal in memory. So its lock acquire is BOUNDED and gives up quietly.
// ---------------------------------------------------------------------------

// LedgerPath is the fleet-wide consolidated ledger. One file, because one bucket.
func LedgerPath(p Paths) string {
	return filepath.Join(p.Home, ".ailang", "state", "quota-ledger.json")
}

// journalPath is the append-only spend journal.
func journalPath(p Paths) string {
	return filepath.Join(p.Home, ".ailang", "state", "quota-ledger-journal.tsv")
}

// journalRotatedGlob matches segments a consolidation has claimed but not yet deleted.
func journalRotatedGlob(p Paths) string {
	return journalPath(p) + ".*"
}

// journalRetention is how long a journal segment is kept.
//
// It MUST exceed the longest window, because the journal is the sole authority for
// spend and ledger.json is only a cache of what the journal says. A segment deleted
// while its tokens still fall inside a live window would make the next rebuild forget
// them, and the ration would then measure low exactly as it does today.
const journalRetention = 8 * 24 * time.Hour

// lockGrace is how old the consolidation lock must be before it is treated as
// abandoned by a crashed process and stolen.
const lockGrace = 5 * time.Minute

// AppendSpend records one spend event. It takes NO LOCK and never blocks.
//
// Both windows are recorded from the one event: the same tokens count against the 5-hour
// window and the long cap simultaneously, because both providers enforce both.
func AppendSpend(p Paths, bucket string, tokens int64, stages int, at time.Time) error {
	if bucket == "" {
		return fmt.Errorf("quota ledger: bucket is required")
	}
	if tokens <= 0 {
		return fmt.Errorf("quota ledger: tokens must be positive (got %d); a zero append is indistinguishable from never reporting", tokens)
	}
	path := journalPath(p)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("quota ledger: create state dir: %w", err)
	}
	// LEADING newline, not just a trailing one. O_APPEND writes are atomic below
	// PIPE_BUF, but a process killed mid-write still leaves a partial line; without a
	// leading newline the NEXT append concatenates onto that fragment and both records
	// are lost instead of one. Blank lines are skipped by the parser, so the cost is a
	// leading blank line in the file.
	line := fmt.Sprintf("\n%d\t%s\t%d\t%d\n", at.UTC().Unix(), bucket, tokens, stages)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // fixed state path
	if err != nil {
		return fmt.Errorf("quota ledger: open journal: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("quota ledger: append journal: %w", err)
	}
	return nil
}

// journalEntry is one parsed spend event.
type journalEntry struct {
	At     time.Time
	Bucket string
	Tokens int64
	Stages int
}

// readJournalFiles parses every named journal file, skipping malformed lines.
//
// A malformed line is SKIPPED rather than fatal: the journal is appended by several
// processes, and a torn final line must not make the whole ledger unreadable and pause
// the fleet. The skipped count is returned so it is reported rather than hidden.
func readJournalFiles(paths []string) ([]journalEntry, int, error) {
	var out []journalEntry
	skipped := 0
	for _, path := range paths {
		body, err := os.ReadFile(path) //nolint:gosec // fixed state path
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, skipped, fmt.Errorf("quota ledger: read journal %s: %w", path, err)
		}
		for _, ln := range strings.Split(string(body), "\n") {
			if strings.TrimSpace(ln) == "" {
				continue
			}
			f := strings.Split(ln, "\t")
			if len(f) != 4 {
				skipped++
				continue
			}
			sec, err1 := strconv.ParseInt(f[0], 10, 64)
			tok, err2 := strconv.ParseInt(f[2], 10, 64)
			stg, err3 := strconv.Atoi(f[3])
			if err1 != nil || err2 != nil || err3 != nil || f[1] == "" || tok <= 0 {
				skipped++
				continue
			}
			out = append(out, journalEntry{
				At: time.Unix(sec, 0).UTC(), Bucket: f[1], Tokens: tok, Stages: stg,
			})
		}
	}
	// Fold in timestamp order so window rollover is evaluated chronologically. An
	// out-of-order fold would roll a live window on a late-arriving older entry and
	// discard the spend recorded before it.
	sort.Slice(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out, skipped, nil
}

// loadLedgerFile reads the consolidated ledger. A missing file is an EMPTY ledger, not
// an error: before the first consolidation there is simply nothing there, and failing
// would make every fire depend on a file that does not exist yet.
func loadLedgerFile(path string) (*Ledger, error) {
	body, err := os.ReadFile(path) //nolint:gosec // fixed state path
	if os.IsNotExist(err) {
		return &Ledger{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("quota ledger: read %s: %w", path, err)
	}
	var l Ledger
	if err := json.Unmarshal(body, &l); err != nil {
		return nil, fmt.Errorf("quota ledger: %s is malformed: %w", path, err)
	}
	return &l, nil
}

// journalFiles lists every journal file: the retired segments plus the live one.
func journalFiles(p Paths) ([]string, error) {
	segs, err := filepath.Glob(journalRotatedGlob(p))
	if err != nil {
		return nil, fmt.Errorf("quota ledger: scan journal segments: %w", err)
	}
	sort.Strings(segs) // segment names are nanosecond stamps, so this is chronological
	return append(segs, journalPath(p)), nil
}

// rebuild computes the ledger from scratch: capacities from the cache file, every token
// from the journal.
//
// TOTALS ARE NEVER CARRIED FORWARD from the cache. The journal is the sole authority for
// spend; ledger.json only memoises the result and holds the capacities (which M3 writes
// and the journal does not carry). Folding the journal ON TOP of cached totals would
// double-count every entry on every pass — which is exactly why the counters are zeroed
// here rather than incremented.
func rebuild(p Paths, now time.Time) (*Ledger, int, error) {
	l, err := loadLedgerFile(LedgerPath(p))
	if err != nil {
		return nil, 0, err
	}
	// Clearing WindowStart is what discards the cached totals: add() sees an expired
	// window on the first entry for each (bucket, window) and rolls it over, so the
	// counters restart from the journal and the window start is derived from the
	// journal's own first entry rather than inherited from the cache. Zeroing the
	// counters here as well would be redundant — and untestable, since removing it
	// changes nothing.
	for i := range l.Usage {
		l.Usage[i].WindowStart = time.Time{}
	}
	files, err := journalFiles(p)
	if err != nil {
		return nil, 0, err
	}
	entries, skipped, err := readJournalFiles(files)
	if err != nil {
		return nil, skipped, err
	}
	for _, e := range entries {
		l.add(e.Bucket, WindowShort, e.Tokens, e.Stages, e.At)
		l.add(e.Bucket, WindowLong, e.Tokens, e.Stages, e.At)
	}
	l.expire(now)
	l.sortUsage()
	return l, skipped, nil
}

// LoadLedger returns the fleet ledger as of `now`. It takes no lock and never blocks, so
// a routing decision is never delayed by bookkeeping.
func LoadLedger(p Paths, now time.Time) (*Ledger, error) {
	l, _, err := rebuild(p, now)
	return l, err
}

// expire resets any window whose period has elapsed as of `now`, preserving capacity.
//
// A window that expired while the fleet was idle must read as reset, or the first fire
// after a quiet night inherits a full window of consumption and pauses itself.
func (l *Ledger) expire(now time.Time) {
	for i := range l.Usage {
		u := &l.Usage[i]
		if u.WindowStart.IsZero() {
			continue
		}
		if start, ok := u.windowStartAt(now); ok {
			if start.After(u.WindowStart) {
				u.WindowStart = start
				u.Tokens = 0
				u.Stages = 0
			}
			continue
		}
		if now.Sub(u.WindowStart) >= windowDuration(u.Window) {
			u.WindowStart = now
			u.Tokens = 0
			u.Stages = 0
		}
	}
}

// Consolidate recomputes the ledger cache and retires aged-out journal segments.
//
// It is BEST-EFFORT by design. If another process holds the consolidation lock this
// returns (false, nil) immediately — no wait, no error, no fire delayed — because the
// journal is already durable and every reader rebuilds from it anyway. Making a fire
// fail on bookkeeping contention would be the wrong trade in exactly the situation this
// milestone exists to survive.
//
// The bool reports whether consolidation actually ran.
func Consolidate(p Paths, now time.Time) (bool, error) {
	lockPath := LedgerPath(p) + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o750); err != nil {
		return false, fmt.Errorf("quota ledger: create state dir: %w", err)
	}
	release, ok := acquireLedgerLock(lockPath, now)
	if !ok {
		return false, nil
	}
	defer release()

	// Roll the live journal into a segment so it stops growing without bound. Appends
	// after this land in a fresh journal and are folded on the next pass. Nothing is
	// read between the rename and the fold that could miss them, because the fold reads
	// the segment we just created along with all the others.
	if err := os.Rename(journalPath(p), fmt.Sprintf("%s.%d", journalPath(p), now.UTC().UnixNano())); err != nil {
		if !os.IsNotExist(err) {
			return false, fmt.Errorf("quota ledger: roll journal: %w", err)
		}
	}

	l, skipped, err := rebuild(p, now)
	if err != nil {
		return false, err
	}
	l.Updated = now.UTC()
	body, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return false, fmt.Errorf("quota ledger: encode: %w", err)
	}
	if err := writeAtomic(LedgerPath(p), append(body, '\n'), 0o600); err != nil {
		return false, err
	}

	// Retire only segments older than the longest window. See journalRetention.
	segs, err := filepath.Glob(journalRotatedGlob(p))
	if err != nil {
		return true, fmt.Errorf("quota ledger: scan journal segments: %w", err)
	}
	for _, seg := range segs {
		fi, err := os.Stat(seg)
		if err != nil {
			continue
		}
		if now.Sub(fi.ModTime()) > journalRetention {
			_ = os.Remove(seg)
		}
	}
	if skipped > 0 {
		return true, fmt.Errorf("quota ledger: consolidated, but skipped %d malformed journal line(s)", skipped)
	}
	return true, nil
}

// SaveCapacities writes capacity values into the ledger cache (M3 supplies them).
//
// Capacity is the ONE thing the journal does not carry, so it lives in the cache file
// and every rebuild preserves it. Like Consolidate, this gives up quietly rather than
// waiting: a missed capacity update leaves the bucket UNRATIONED and loud, which is the
// documented safe state.
func SaveCapacities(p Paths, caps map[string]map[Window]int64, now time.Time) (bool, error) {
	lockPath := LedgerPath(p) + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o750); err != nil {
		return false, fmt.Errorf("quota ledger: create state dir: %w", err)
	}
	release, ok := acquireLedgerLock(lockPath, now)
	if !ok {
		return false, nil
	}
	defer release()

	l, _, err := rebuild(p, now)
	if err != nil {
		return false, err
	}
	for bucket, byWindow := range caps {
		for w, c := range byWindow {
			l.SetCapacity(bucket, w, c, now)
		}
	}
	l.sortUsage()
	l.Updated = now.UTC()
	body, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return false, fmt.Errorf("quota ledger: encode: %w", err)
	}
	return true, writeAtomic(LedgerPath(p), append(body, '\n'), 0o600)
}

// acquireLedgerLock takes the consolidation lock by atomic mkdir (portable; macOS has no
// flock, the same reason internal/riglock uses mkdir).
//
// It does NOT wait. A lock older than lockGrace was left by a crashed process and is
// stolen, so a dead consolidator cannot stop the ledger being consolidated ever again.
func acquireLedgerLock(path string, now time.Time) (func(), bool) {
	if err := os.Mkdir(path, 0o750); err == nil {
		return func() { _ = os.Remove(path) }, true
	}
	fi, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if now.Sub(fi.ModTime()) <= lockGrace {
		return nil, false // someone is consolidating right now
	}
	if err := os.Remove(path); err != nil {
		return nil, false
	}
	if err := os.Mkdir(path, 0o750); err != nil {
		return nil, false
	}
	return func() { _ = os.Remove(path) }, true
}

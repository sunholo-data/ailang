package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testPaths(t *testing.T) Paths {
	t.Helper()
	return Paths{Home: t.TempDir()}
}

var refTime = time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

// --- ration policy -------------------------------------------------------

// The daily ration paces the LONG window only. Applying 10%/day pro-rata to the 5-hour
// window would cap each of the ~4.8 daily windows at ~2% of its capacity and idle the
// fleet for a limit nobody imposed.
func TestVerdict_ShortWindowIsNotPacedByTheDailyRation(t *testing.T) {
	u := BucketUsage{
		Bucket: "codex", Window: WindowShort,
		Capacity: 1000, Tokens: 900, // 90% of a 5h bucket
		WindowStart: refTime.Add(-4 * time.Hour),
	}
	v := u.Verdict(refTime)
	if v.State != RationNotPaced {
		t.Fatalf("short window state = %q, want %q", v.State, RationNotPaced)
	}
	if v.Over() {
		t.Fatal("short window reported over ration; the daily rule does not pace it")
	}
	// Guard: the same numbers on the LONG window MUST be over. Without this the test
	// passes even if Verdict returns not-paced for every window.
	u.Window = WindowLong
	if got := u.Verdict(refTime); !got.Over() {
		t.Fatalf("control failed: 900/1000 on the long window should be over, got %q", got.State)
	}
}

// An unknown capacity is neither a pass nor a fail. Rationing against a guess would
// either idle a healthy fleet or wave through an empty bucket, confidently.
func TestVerdict_UnknownCapacityIsNeitherOverNorOK(t *testing.T) {
	u := BucketUsage{
		Bucket: "anthropic", Window: WindowLong,
		Tokens: 1 << 40, WindowStart: refTime.Add(-24 * time.Hour),
	}
	v := u.Verdict(refTime)
	if v.State != RationCapacityUnknown {
		t.Fatalf("state = %q, want %q", v.State, RationCapacityUnknown)
	}
	if v.Over() {
		t.Fatal("unknown capacity reported as over ration")
	}
	if v.State == RationOK {
		t.Fatal("unknown capacity reported as ok")
	}
	// Control: the identical usage WITH a capacity must be over, proving the unknown
	// state comes from the missing capacity and not from the numbers.
	u.Capacity = 1000
	if got := u.Verdict(refTime); !got.Over() {
		t.Fatalf("control failed: want over with capacity set, got %q", got.State)
	}
}

// 10%/day pro-rata: the allowance grows with elapsed window time, floored at one day.
func TestVerdict_AllowanceIsProRataWithAOneDayFloor(t *testing.T) {
	cases := []struct {
		name    string
		elapsed time.Duration
		want    int64
	}{
		{"fresh window floors at one day", 0, 100},
		{"half a day still floors at one day", 12 * time.Hour, 100},
		{"three days", 72 * time.Hour, 300},
		{"full week caps at seven days", 14 * 24 * time.Hour, 700},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := BucketUsage{
				Bucket: "codex", Window: WindowLong, Capacity: 1000,
				WindowStart: refTime.Add(-tc.elapsed),
			}
			if got := u.Verdict(refTime).Allowance; got != tc.want {
				t.Fatalf("allowance after %s = %d, want %d", tc.elapsed, got, tc.want)
			}
		})
	}
}

// A week at 10%/day spends 70%, leaving 30% in reserve. That reserve is the point of
// D-1 (the measured bad day spent 50% of codex in one go), so pin it.
func TestVerdict_AWeekOfFullRationLeavesThirtyPercentInReserve(t *testing.T) {
	u := BucketUsage{
		Bucket: "codex", Window: WindowLong, Capacity: 1000,
		WindowStart: refTime.Add(-7 * 24 * time.Hour),
	}
	if got := u.Verdict(refTime).Allowance; got != 700 {
		t.Fatalf("a full week's allowance = %d of 1000, want 700 (30%% reserve)", got)
	}
}

// A clock skewed into the future must not pause the fleet.
func TestVerdict_FutureWindowStartDoesNotPauseTheFleet(t *testing.T) {
	u := BucketUsage{
		Bucket: "codex", Window: WindowLong, Capacity: 1000, Tokens: 50,
		WindowStart: refTime.Add(48 * time.Hour),
	}
	v := u.Verdict(refTime)
	if v.Over() {
		t.Fatalf("skewed clock paused the bucket: %s", v.Reason)
	}
	if v.Allowance != 100 {
		t.Fatalf("allowance = %d, want the one-day floor of 100", v.Allowance)
	}
	// The allowance floor hides a negative elapsed from the ARITHMETIC, but not from
	// the operator. Spend past the floor so the verdict takes the over-ration branch,
	// which is the one that renders elapsed: unclamped, it reads "-48h0m0s elapsed" in
	// the reason an engineer sees when the lane is skipped.
	u.Tokens = 500
	over := u.Verdict(refTime)
	if !over.Over() {
		t.Fatalf("control failed: 500 tok against a floored allowance of 100 should be over, got %q", over.State)
	}
	if strings.Contains(over.Reason, "-") {
		t.Fatalf("reason reports a negative elapsed time: %q", over.Reason)
	}
}

// --- journal / rebuild ---------------------------------------------------

func TestAppendSpend_RejectsZeroAndEmptyBucket(t *testing.T) {
	p := testPaths(t)
	if err := AppendSpend(p, "codex", 0, 1, refTime); err == nil {
		t.Fatal("a zero append was accepted; it is indistinguishable from never reporting")
	}
	if err := AppendSpend(p, "", 10, 1, refTime); err == nil {
		t.Fatal("an empty bucket was accepted")
	}
	// Control: the valid form must work, or the guards above prove nothing.
	if err := AppendSpend(p, "codex", 10, 1, refTime); err != nil {
		t.Fatalf("control append failed: %v", err)
	}
}

// One event counts against BOTH windows: providers enforce both simultaneously.
func TestAppendSpend_RecordsAgainstBothWindows(t *testing.T) {
	p := testPaths(t)
	if err := AppendSpend(p, "codex", 500, 1, refTime); err != nil {
		t.Fatal(err)
	}
	l, err := LoadLedger(p, refTime)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []Window{WindowShort, WindowLong} {
		u, ok := l.Get("codex", w)
		if !ok {
			t.Fatalf("no usage recorded for window %s", w)
		}
		if u.Tokens != 500 {
			t.Fatalf("window %s tokens = %d, want 500", w, u.Tokens)
		}
	}
}

// The ledger file is a CACHE. Rebuilding must recompute totals from the journal, never
// add the journal on top of the cached totals — that would double-count every entry on
// every read and every consolidation.
func TestLoadLedger_DoesNotDoubleCountAfterConsolidation(t *testing.T) {
	p := testPaths(t)
	if err := AppendSpend(p, "codex", 300, 1, refTime); err != nil {
		t.Fatal(err)
	}
	if ran, err := Consolidate(p, refTime); err != nil || !ran {
		t.Fatalf("consolidate ran=%v err=%v", ran, err)
	}
	// Verify the cache really was written, so the read below is exercising the
	// cache+journal path and not an empty-file shortcut.
	if _, err := os.Stat(LedgerPath(p)); err != nil {
		t.Fatalf("consolidation wrote no ledger file: %v", err)
	}
	for i := 0; i < 3; i++ {
		l, err := LoadLedger(p, refTime)
		if err != nil {
			t.Fatal(err)
		}
		u, _ := l.Get("codex", WindowLong)
		if u.Tokens != 300 {
			t.Fatalf("read %d: tokens = %d, want 300 (double-counted?)", i, u.Tokens)
		}
	}
	// And consolidating again must not inflate it either.
	if _, err := Consolidate(p, refTime); err != nil {
		t.Fatal(err)
	}
	l, err := LoadLedger(p, refTime)
	if err != nil {
		t.Fatal(err)
	}
	if u, _ := l.Get("codex", WindowLong); u.Tokens != 300 {
		t.Fatalf("after a second consolidation tokens = %d, want 300", u.Tokens)
	}
}

// Consolidation retires the live journal into a segment; the tokens in that segment
// must survive, because the journal is the sole authority for spend.
func TestConsolidate_RolledSegmentsStillCount(t *testing.T) {
	p := testPaths(t)
	if err := AppendSpend(p, "codex", 100, 1, refTime); err != nil {
		t.Fatal(err)
	}
	if _, err := Consolidate(p, refTime); err != nil {
		t.Fatal(err)
	}
	// The live journal must be gone (rolled), or this test would pass without the
	// segment being read at all.
	if _, err := os.Stat(journalPath(p)); !os.IsNotExist(err) {
		t.Fatalf("live journal was not rolled: %v", err)
	}
	if err := AppendSpend(p, "codex", 50, 1, refTime.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	l, err := LoadLedger(p, refTime.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if u, _ := l.Get("codex", WindowLong); u.Tokens != 150 {
		t.Fatalf("tokens = %d, want 150 (100 in a rolled segment + 50 live)", u.Tokens)
	}
}

// A segment must not be deleted while its tokens still fall inside a live window.
func TestConsolidate_RetainsSegmentsLongerThanTheLongestWindow(t *testing.T) {
	p := testPaths(t)
	if err := AppendSpend(p, "codex", 100, 1, refTime); err != nil {
		t.Fatal(err)
	}
	if _, err := Consolidate(p, refTime); err != nil {
		t.Fatal(err)
	}
	// Six days later the long window is still live; the segment must survive.
	sixDays := refTime.Add(6 * 24 * time.Hour)
	if _, err := Consolidate(p, sixDays); err != nil {
		t.Fatal(err)
	}
	segs, _ := filepath.Glob(journalRotatedGlob(p))
	if len(segs) == 0 {
		t.Fatal("segment deleted while the long window that contains it is still live")
	}
	if journalRetention <= windowDuration(WindowLong) {
		t.Fatalf("journalRetention (%s) must exceed the longest window (%s)",
			journalRetention, windowDuration(WindowLong))
	}
}

// A torn or garbage line must not make the ledger unreadable and pause the fleet.
func TestReadJournal_SkipsMalformedLinesAndKeepsTheRest(t *testing.T) {
	p := testPaths(t)
	if err := AppendSpend(p, "codex", 100, 1, refTime); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(journalPath(p), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	// Garbage, a non-numeric record, and a TORN line (killed mid-write, no trailing
	// newline). The torn line must corrupt only itself: the next append must survive.
	if _, err := f.WriteString("nonsense\nbad\tcodex\tx\t1\n1788\t"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if err := AppendSpend(p, "codex", 25, 1, refTime.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	l, err := LoadLedger(p, refTime.Add(time.Minute))
	if err != nil {
		t.Fatalf("a malformed line made the whole ledger unreadable: %v", err)
	}
	if u, _ := l.Get("codex", WindowLong); u.Tokens != 125 {
		t.Fatalf("tokens = %d, want 125 (both good lines, no garbage)", u.Tokens)
	}
}

// A window that expired while the fleet was idle must read as reset, or the first fire
// after a quiet night inherits a spent window and pauses itself.
func TestLoadLedger_ExpiredWindowResets(t *testing.T) {
	p := testPaths(t)
	if err := AppendSpend(p, "codex", 900, 1, refTime); err != nil {
		t.Fatal(err)
	}
	// Still inside the 5h window: the spend is visible.
	l, err := LoadLedger(p, refTime.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if u, _ := l.Get("codex", WindowShort); u.Tokens != 900 {
		t.Fatalf("inside the window tokens = %d, want 900", u.Tokens)
	}
	// Past it: reset.
	l, err = LoadLedger(p, refTime.Add(6*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if u, _ := l.Get("codex", WindowShort); u.Tokens != 0 {
		t.Fatalf("after the 5h window elapsed tokens = %d, want 0", u.Tokens)
	}
	// The long window has NOT elapsed and must still hold the spend.
	if u, _ := l.Get("codex", WindowLong); u.Tokens != 900 {
		t.Fatalf("long window tokens = %d, want 900 (only the 5h window expired)", u.Tokens)
	}
}

// Capacity lives only in the cache file; every rebuild must preserve it or M3's work is
// erased by the next fire.
func TestRebuild_PreservesCapacityAcrossConsolidation(t *testing.T) {
	p := testPaths(t)
	if err := AppendSpend(p, "codex", 10, 1, refTime); err != nil {
		t.Fatal(err)
	}
	ok, err := SaveCapacities(p, map[string]map[Window]int64{
		"codex": {WindowLong: 5000},
	}, refTime)
	if err != nil || !ok {
		t.Fatalf("SaveCapacities ok=%v err=%v", ok, err)
	}
	if err := AppendSpend(p, "codex", 20, 1, refTime.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := Consolidate(p, refTime.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	l, err := LoadLedger(p, refTime.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	u, _ := l.Get("codex", WindowLong)
	if u.Capacity != 5000 {
		t.Fatalf("capacity = %d after consolidation, want 5000 (M3's value erased)", u.Capacity)
	}
	if u.Tokens != 30 {
		t.Fatalf("tokens = %d, want 30", u.Tokens)
	}
}

// --- concurrency ---------------------------------------------------------

// Concurrent missions must not corrupt the ledger, and no fire may be lost.
func TestAppendSpend_ConcurrentWritersLoseNothing(t *testing.T) {
	p := testPaths(t)
	const writers, each = 8, 25
	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < each; i++ {
				if err := AppendSpend(p, "codex", 1, 1, refTime.Add(time.Duration(i)*time.Second)); err != nil {
					t.Errorf("writer %d: %v", w, err)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	l, err := LoadLedger(p, refTime.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	u, _ := l.Get("codex", WindowLong)
	if want := int64(writers * each); u.Tokens != want {
		t.Fatalf("tokens = %d, want %d — concurrent appends were lost or torn", u.Tokens, want)
	}
}

// A held consolidation lock must NEVER wedge a fire: the write path takes no lock, and
// consolidation gives up immediately rather than waiting.
func TestConsolidate_HeldLockNeverBlocksAFire(t *testing.T) {
	p := testPaths(t)
	lockPath := LedgerPath(p) + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(lockPath, 0o750); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		// The two things a fire does. Neither may block on the held lock.
		if err := AppendSpend(p, "codex", 42, 1, refTime); err != nil {
			t.Errorf("append blocked or failed under a held lock: %v", err)
		}
		ran, err := Consolidate(p, refTime)
		if err != nil {
			t.Errorf("consolidate errored under a held lock instead of giving up quietly: %v", err)
		}
		if ran {
			t.Error("consolidate claimed to run while another process held the lock")
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a held lock wedged the fire path")
	}

	// And the spend is still visible without any consolidation having happened.
	l, err := LoadLedger(p, refTime)
	if err != nil {
		t.Fatal(err)
	}
	if u, _ := l.Get("codex", WindowLong); u.Tokens != 42 {
		t.Fatalf("tokens = %d, want 42 — an unconsolidated journal must still be readable", u.Tokens)
	}
}

// A lock left by a crashed process must not stop consolidation forever.
func TestConsolidate_StealsAnAbandonedLock(t *testing.T) {
	p := testPaths(t)
	lockPath := LedgerPath(p) + ".lock"
	if err := os.MkdirAll(lockPath, 0o750); err != nil {
		t.Fatal(err)
	}
	// The lock's age is its mtime, which is what a crashed process actually leaves
	// behind. Pin it to the test's clock so the comparison is against refTime and not
	// against whatever the wall clock happens to be when the suite runs.
	if err := os.Chtimes(lockPath, refTime, refTime); err != nil {
		t.Fatal(err)
	}
	if err := AppendSpend(p, "codex", 7, 1, refTime); err != nil {
		t.Fatal(err)
	}
	// Fresh lock: not stolen.
	if ran, _ := Consolidate(p, refTime); ran {
		t.Fatal("a fresh lock was stolen")
	}
	// Aged past the grace: stolen.
	ran, err := Consolidate(p, refTime.Add(lockGrace+time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("an abandoned lock was never stolen; consolidation is wedged forever")
	}
}

// --- reporting -----------------------------------------------------------

func TestOverRation_AnyWindowOverIsEnough(t *testing.T) {
	l := &Ledger{Usage: []BucketUsage{
		{Bucket: "codex", Window: WindowShort, Capacity: 1000, Tokens: 999, WindowStart: refTime},
		{Bucket: "codex", Window: WindowLong, Capacity: 1000, Tokens: 50, WindowStart: refTime},
	}}
	// The short window is not paced, so 999/1000 there is NOT over.
	if over, why := l.OverRation("codex", refTime); over {
		t.Fatalf("short-window spend reported over ration: %s", why)
	}
	// Push the long window over and it must be reported, with a reason naming it.
	l.Usage[1].Tokens = 5000
	over, why := l.OverRation("codex", refTime)
	if !over {
		t.Fatal("long window over its allowance was not reported")
	}
	if !strings.Contains(why, string(WindowLong)) {
		t.Fatalf("reason %q does not name the window that is over", why)
	}
}

func TestLedgerString_NamesEveryStateItReports(t *testing.T) {
	l := &Ledger{Usage: []BucketUsage{
		{Bucket: "anthropic", Window: WindowLong, Tokens: 10, WindowStart: refTime},
		{Bucket: "codex", Window: WindowLong, Capacity: 1000, Tokens: 5000, WindowStart: refTime},
		{Bucket: "codex", Window: WindowShort, Capacity: 1000, Tokens: 10, WindowStart: refTime},
	}}
	out := l.String(refTime)
	for _, want := range []string{string(RationCapacityUnknown), string(RationOver), string(RationNotPaced)} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendering omits state %q:\n%s", want, out)
		}
	}
}

// The on-disk shape is read by other tools; pin it.
func TestConsolidate_WritesReadableJSON(t *testing.T) {
	p := testPaths(t)
	if err := AppendSpend(p, "codex", 11, 2, refTime); err != nil {
		t.Fatal(err)
	}
	if _, err := Consolidate(p, refTime); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(LedgerPath(p))
	if err != nil {
		t.Fatal(err)
	}
	var l Ledger
	if err := json.Unmarshal(body, &l); err != nil {
		t.Fatalf("ledger file is not valid JSON: %v\n%s", err, body)
	}
	if l.Updated.IsZero() {
		t.Fatal("ledger has no updated timestamp")
	}
	if len(l.Usage) != 2 {
		t.Fatalf("usage rows = %d, want 2 (one per window)", len(l.Usage))
	}
	if got := fmt.Sprintf("%s/%s", l.Usage[0].Bucket, l.Usage[0].Window); got != "codex/5h" {
		t.Fatalf("first row = %s, want codex/5h (rows must be sorted)", got)
	}
}

// --- provider-derived window boundaries ----------------------------------

// With a provider reset known, our window and the provider's must be the SAME window.
// Deriving the boundary from our own first write drifts from the limit actually being
// enforced, so the counter resets hours away from the bucket refilling.
func TestWindowStart_SnapsToTheProviderBoundary(t *testing.T) {
	// codex prints "resets 05:34"; take that as the anchor.
	reset := time.Date(2026, 9, 6, 5, 34, 0, 0, time.UTC)
	u := BucketUsage{Bucket: "codex", Window: WindowShort, ProviderReset: reset}

	cases := []struct {
		at   time.Time
		want time.Time
	}{
		{reset, reset},
		{reset.Add(4 * time.Hour), reset},                       // same window
		{reset.Add(5 * time.Hour), reset.Add(5 * time.Hour)},    // next boundary exactly
		{reset.Add(7 * time.Hour), reset.Add(5 * time.Hour)},    // into the next window
		{reset.Add(-time.Hour), reset.Add(-5 * time.Hour)},      // before the anchor
		{reset.Add(-6 * time.Hour), reset.Add(-10 * time.Hour)}, // two windows before
	}
	for _, tc := range cases {
		got, ok := u.windowStartAt(tc.at)
		if !ok {
			t.Fatalf("no boundary for %s despite a known provider reset", tc.at)
		}
		if !got.Equal(tc.want) {
			t.Errorf("windowStartAt(%s) = %s, want %s", tc.at, got, tc.want)
		}
	}
}

// Without a provider reset the ledger must say so rather than imply the boundary is
// authoritative.
func TestBoundarySource_DistinguishesProviderFromLocal(t *testing.T) {
	local := BucketUsage{Bucket: "codex", Window: WindowShort}
	if got := local.BoundarySource(); got != "local" {
		t.Errorf("BoundarySource() = %q, want local", got)
	}
	if _, ok := local.windowStartAt(refTime); ok {
		t.Error("an unset provider reset produced a boundary")
	}
	pinned := BucketUsage{Bucket: "codex", Window: WindowShort, ProviderReset: refTime}
	if got := pinned.BoundarySource(); got != "provider" {
		t.Errorf("BoundarySource() = %q, want provider", got)
	}
}

// Spend either side of a provider boundary must land in different windows.
func TestLoadLedger_ProviderResetSplitsTheWindows(t *testing.T) {
	p := testPaths(t)
	reset := time.Date(2026, 9, 6, 5, 34, 0, 0, time.UTC)

	if err := AppendSpend(p, "codex", 100, 1, reset.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	l, err := LoadLedger(p, reset.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	l.SetProviderReset("codex", WindowShort, reset, reset)
	body, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(LedgerPath(p)), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(LedgerPath(p), body, 0o600); err != nil {
		t.Fatal(err)
	}

	// A second spend 5h later is past the provider boundary: the 5h window resets,
	// while the long window (no reset pinned) keeps both.
	if err := AppendSpend(p, "codex", 40, 1, reset.Add(6*time.Hour)); err != nil {
		t.Fatal(err)
	}
	l, err = LoadLedger(p, reset.Add(6*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	short, _ := l.Get("codex", WindowShort)
	if short.Tokens != 40 {
		t.Errorf("5h window tokens = %d, want 40 (the provider boundary reset it)", short.Tokens)
	}
	if !short.WindowStart.Equal(reset.Add(5 * time.Hour)) {
		t.Errorf("window start = %s, want the provider boundary %s", short.WindowStart, reset.Add(5*time.Hour))
	}
	if long, _ := l.Get("codex", WindowLong); long.Tokens != 140 {
		t.Errorf("long window tokens = %d, want 140 (its window did not roll)", long.Tokens)
	}
}

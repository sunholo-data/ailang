package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func quotaFixture(t *testing.T, at time.Time, primary, secondary any) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{"timestamp": at, "type": "event_msg", "payload": map[string]any{"type": "token_count", "rate_limits": map[string]any{"limit_id": "codex", "primary": primary, "secondary": secondary}}})
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func quotaWindow(at time.Time, used float64, minutes int64) map[string]any {
	return map[string]any{"used_percent": used, "window_minutes": minutes, "resets_at": at.Add(time.Duration(minutes)*time.Minute - time.Hour).Unix()}
}
func TestCodexQuotaProviderWindows(t *testing.T) {
	now := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name               string
		primary, secondary any
		age                time.Duration
		want               string
	}{
		{"weekly primary", quotaWindow(now, 17, 10080), nil, 0, "over"},
		{"weekly secondary", quotaWindow(now, 20, 300), quotaWindow(now, 17, 10080), 0, "over"},
		{"zero is real", quotaWindow(now, 0, 10080), nil, 0, "ok"},
		{"exact first day allowance", quotaWindow(now, 10, 10080), nil, 0, "ok"},
		{"short exhausted", quotaWindow(now, 100, 300), quotaWindow(now, 1, 10080), 0, "over"},
		{"short only", quotaWindow(now, 1, 300), nil, 0, "unknown"},
		{"missing percentage", map[string]any{"window_minutes": 10080, "resets_at": now.Add(7 * 24 * time.Hour).Unix()}, nil, 0, "unknown"},
		{"invalid second clears first", quotaWindow(now, 1, 10080), map[string]any{"used_percent": 101, "window_minutes": 300, "resets_at": now.Add(time.Hour).Unix()}, 0, "unknown"},
		{"stale", quotaWindow(now, 1, 10080), nil, 16 * time.Minute, "stale"},
		{"expired", map[string]any{"used_percent": 1, "window_minutes": 10080, "resets_at": now.Add(-time.Minute).Unix()}, nil, 2 * time.Minute, "expired"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := parseCodexQuota(quotaFixture(t, now.Add(-tc.age), tc.primary, tc.secondary), now)
			if o == nil {
				t.Fatal("missing observation")
			}
			o.evaluate(now)
			if o.State != tc.want {
				t.Fatalf("got %+v want %s", o, tc.want)
			}
			if o.Blocked() != (tc.want != "ok") {
				t.Fatal("admission mismatch")
			}
		})
	}
}
func TestObserveCodexQuotaLatestAndPartialTail(t *testing.T) {
	now := time.Now().UTC()
	home := t.TempDir()
	dir := filepath.Join(home, "sessions", now.Format("2006/01/02"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	old := quotaFixture(t, now.Add(-time.Minute), quotaWindow(now, 5, 10080), nil)
	latest := quotaFixture(t, now, quotaWindow(now, 25, 10080), nil)
	// A huge irrelevant record is bounded away, and an in-flight record is ignored.
	data := strings.Repeat("x", 3<<20) + "\n" + string(old) + "\n" + string(latest) + "\n" + `{"type":"event_msg"`
	if err := os.WriteFile(filepath.Join(dir, "rollout-test.jsonl"), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	o := observeCodexSessions(home, now)
	if o.State != "over" || o.Windows[0].UsedPercent != 25 {
		t.Fatalf("%+v", o)
	}
	// Re-reading cannot accumulate percentages or mutate records.
	again := observeCodexSessions(home, now)
	if again.Windows[0].UsedPercent != 25 {
		t.Fatal(again)
	}
}
func TestCodexQuotaMissingAndWrongLimit(t *testing.T) {
	now := time.Now()
	o := observeCodexSessions(t.TempDir(), now)
	if !o.Blocked() || o.State != "unknown" {
		t.Fatal(o)
	}
	line := quotaFixture(t, now, quotaWindow(now, 1, 10080), nil)
	line = []byte(strings.Replace(string(line), `"limit_id":"codex"`, `"limit_id":"other"`, 1))
	if parseCodexQuota(line, now) != nil {
		t.Fatal("accepted other bucket")
	}
}

func TestCodexQuotaMalformedDoesNotExposeOlderPass(t *testing.T) {
	now := time.Now().UTC()
	home := t.TempDir()
	dir := filepath.Join(home, "sessions", now.Format("2006/01/02"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	good := quotaFixture(t, now.Add(-time.Minute), quotaWindow(now, 1, 10080), nil)
	bad := quotaFixture(t, now, quotaWindow(now, 99, 10080), nil)
	bad = []byte(strings.Replace(string(bad), `"used_percent":99`, `"used_percent":"99"`, 1))
	if err := os.WriteFile(filepath.Join(dir, "rollout-invalid.jsonl"), append(append(append(good, '\n'), bad...), '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	o := observeCodexSessions(home, now)
	if !o.Blocked() {
		t.Fatalf("invalid newest record uncovered old pass: %+v", o)
	}
}

func TestCodexQuotaValidRefreshRecoversMalformedRecord(t *testing.T) {
	now := time.Now().UTC()
	home := t.TempDir()
	dir := filepath.Join(home, "sessions", now.Format("2006/01/02"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	bad := quotaFixture(t, now.Add(-time.Minute), quotaWindow(now, 99, 10080), nil)
	bad = []byte(strings.Replace(string(bad), `"used_percent":99`, `"used_percent":"99"`, 1))
	good := quotaFixture(t, now, quotaWindow(now, 1, 10080), nil)
	if err := os.WriteFile(filepath.Join(dir, "rollout-refresh.jsonl"), append(append(append(bad, '\n'), good...), '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	o := observeCodexSessions(home, now)
	if o.Blocked() {
		t.Fatalf("valid refresh did not recover: %+v", o)
	}
}

func TestCodexQuotaBoundedScanRequiresNewerObservation(t *testing.T) {
	now := time.Now().UTC()
	home := t.TempDir()
	dir := filepath.Join(home, "sessions", now.Format("2006/01/02"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 129; i++ {
		path := filepath.Join(dir, fmt.Sprintf("rollout-%03d.jsonl", i))
		data := append(quotaFixture(t, now.Add(-time.Hour), quotaWindow(now, 1, 10080), nil), '\n')
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		stamp := now.Add(-30 * time.Minute)
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	if o := observeCodexSessions(home, now); o.State != "unknown" {
		t.Fatalf("omitted newer files should block: %+v", o)
	}
	path := filepath.Join(dir, "rollout-newest.jsonl")
	if err := os.WriteFile(path, append(quotaFixture(t, now, quotaWindow(now, 17, 10080), nil), '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	o := observeCodexSessions(home, now)
	if o.State != "over" || o.Windows[0].UsedPercent != 17 {
		t.Fatalf("fresh observation should win safely: %+v", o)
	}
}

// A blocked observation must still report a TRUTHFUL allowance.
//
// The allowance used to be computed after the staleness and expiry early returns, so on
// exactly the paths an operator reads when something is wrong, the report printed an
// uninitialised `0.0% allowed`. Live on 2026-09-14 that rendered as "73.0% used / 0.0%
// allowed" against a real allowance of 20.1%, and the reason named only the staleness —
// pointing at a refresh that could not have helped, because the bucket was 3.6x over.
//
// The reset is placed 5 days out on a 7-day window so the window STARTED 2 days ago,
// reproducing the measured 20% allowance rather than the 10% first-day floor.
func TestCodexQuotaBlockedStatesStillReportAllowance(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	twoDaysIn := func(used float64) map[string]any {
		return map[string]any{"used_percent": used, "window_minutes": int64(10080), "resets_at": now.Add(5 * 24 * time.Hour).Unix()}
	}
	for _, tc := range []struct {
		name          string
		used          float64
		age           time.Duration
		wantState     string
		wantAllowance float64
		wantOverNote  bool
	}{
		{"stale and over ration", 73, 16 * time.Minute, "stale", 20, true},
		{"stale but within ration", 5, 16 * time.Minute, "stale", 20, false},
		{"fresh and over ration", 73, 0, "over", 20, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := parseCodexQuota(quotaFixture(t, now.Add(-tc.age), twoDaysIn(tc.used), nil), now)
			if o == nil {
				t.Fatal("missing observation")
			}
			o.evaluate(now)
			if o.State != tc.wantState {
				t.Fatalf("state = %q, want %q (reason: %s)", o.State, tc.wantState, o.Reason)
			}
			if !o.Blocked() {
				t.Fatal("every case here must block")
			}
			if len(o.Windows) != 1 {
				t.Fatalf("want 1 window, got %d", len(o.Windows))
			}
			// The regression: this was 0 on the stale path.
			if got := o.Windows[0].AllowancePercent; got < tc.wantAllowance-0.5 || got > tc.wantAllowance+0.5 {
				t.Fatalf("allowance = %.2f%%, want ~%.0f%% — an uninitialised zero reads as a computed hard block", got, tc.wantAllowance)
			}
			// A stale reading that is ALSO over ration must say so, or the operator is
			// told to refresh an observation that will block again on arrival.
			const note = "ALSO over ration"
			if strings.Contains(o.Reason, note) != tc.wantOverNote {
				t.Fatalf("reason %q: want over-ration note = %v", o.Reason, tc.wantOverNote)
			}
		})
	}
}

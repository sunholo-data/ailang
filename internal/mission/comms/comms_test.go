package comms

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// M-MISSION-COMMS-P1 / M2.
//
// Two things are pinned here. First, mission identity: today the driver passes
// MISSION_GH_ISSUE around as a bare env string across six call sites, so "which
// issue does this mission post to" is answered six times. Second, the report cap:
// the v1 bookkeeping thread measured 27 comments / 52,677 chars, mean 1,951 — a
// firehose in which the human's total input was six characters. The renderer
// exists to make that structurally impossible, so the cap is a test, not a habit.
//
// Issue numbers are NOT hardcoded. The bookkeeping threads rotate WEEKLY (v1 was
// 972 for the week of 2026-08-31; docs was already 979 and motoko 987 while this
// was written), so a constant in Go would be stale within days. They are read from
// the same $STATE_DIR/mission-<name>-gh-issue files the driver reads.

func withStateDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("AILANG_STATE_DIR", dir)
	t.Setenv("MISSION_GH_ISSUE", "")
	return dir
}

func TestResolveMission_ReadsIssueFromStateNotAConstant(t *testing.T) {
	withStateDir(t, map[string]string{
		"mission-v1-gh-issue":     "972\n",
		"mission-world-gh-issue":  "107\n",
		"mission-docs-gh-issue":   "979\n",
		"mission-motoko-gh-issue": "987\n",
	})
	for _, tc := range []struct {
		name      string
		wantRepo  string
		wantIssue int
	}{
		{"v1", "sunholo-data/ailang", 972},
		{"world", "sunholo-data/ailang-world", 107},
		{"docs", "sunholo-data/ailang", 979},
		{"motoko", "sunholo-data/ailang", 987},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := ResolveMission(tc.name)
			if err != nil {
				t.Fatalf("ResolveMission(%q): %v", tc.name, err)
			}
			if m.Repo != tc.wantRepo {
				t.Errorf("repo = %q, want %q", m.Repo, tc.wantRepo)
			}
			if m.Issue != tc.wantIssue {
				t.Errorf("issue = %d, want %d", m.Issue, tc.wantIssue)
			}
		})
	}
}

func TestResolveMission_V1UsesTheNamespacedFileNotTheBareOne(t *testing.T) {
	// A documented past defect (mission-control.sh:78-83): v1 alone read the
	// fleet-shared bare `mission-gh-issue`, which holds a CLOSED thread, while
	// `mission-v1-gh-issue` holds the live one. Reintroducing that would post every
	// v1 report into a dead thread, silently.
	withStateDir(t, map[string]string{
		"mission-gh-issue":    "745\n", // the closed one
		"mission-v1-gh-issue": "972\n", // the live one
	})
	m, err := ResolveMission("v1")
	if err != nil {
		t.Fatal(err)
	}
	if m.Issue == 745 {
		t.Fatal("v1 read the bare mission-gh-issue (closed thread 745) — the namespaced file must win")
	}
	if m.Issue != 972 {
		t.Fatalf("issue = %d, want 972 from mission-v1-gh-issue", m.Issue)
	}
}

func TestResolveMission_EnvOverrideWins(t *testing.T) {
	// The driver supports MISSION_GH_ISSUE as an override; the port must too, or
	// a cutover rehearsal against a scratch issue is impossible.
	withStateDir(t, map[string]string{"mission-v1-gh-issue": "972\n"})
	t.Setenv("MISSION_GH_ISSUE", "12345")
	m, err := ResolveMission("v1")
	if err != nil {
		t.Fatal(err)
	}
	if m.Issue != 12345 {
		t.Fatalf("issue = %d, want the env override 12345", m.Issue)
	}
}

func TestResolveMission_UnknownIsLoudNotDefaulted(t *testing.T) {
	// Critical Principle 2. Silently defaulting an unknown mission to v1 would
	// post one mission's report onto another mission's thread.
	withStateDir(t, map[string]string{"mission-v1-gh-issue": "972\n"})
	if _, err := ResolveMission("nope"); err == nil {
		t.Fatal("unknown mission returned nil error — it must fail loudly, not default")
	}
	if _, err := ResolveMission(""); err == nil {
		t.Fatal("empty mission name returned nil error")
	}
}

func TestResolveMission_MissingIssueIsAnErrorNotZero(t *testing.T) {
	// Issue 0 would be posted as `gh issue comment 0` — a confusing remote failure
	// instead of a clear local one.
	withStateDir(t, map[string]string{})
	if _, err := ResolveMission("v1"); err == nil {
		t.Fatal("missing state file returned nil error — must fail loudly")
	}
}

func TestRenderReport_NeverExceedsCap(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   Report
	}{
		{"typical", Report{Mission: "v1", Iteration: 325, Landed: "M3+M4 of m-spawn-pin-enforcement", GoalDistance: "N=12 unmoved", CostUSD: 0, LogCommit: "70e453060"}},
		{"empty", Report{Mission: "v1"}},
		{"over-long", Report{
			Mission:      "v1",
			Iteration:    325,
			Landed:       strings.Repeat("a very long description of what landed ", 40),
			GoalDistance: strings.Repeat("distance ", 40),
			CostUSD:      12.34,
			LogCommit:    "70e453060",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := RenderReport(tc.in)
			if len(got) > MaxReportChars {
				t.Fatalf("len = %d, want <= %d\n%s", len(got), MaxReportChars, got)
			}
		})
	}
}

func TestRenderReport_TruncationIsMarkedNotSilent(t *testing.T) {
	// A truncated report must SAY it was truncated. A report silently cut at 400
	// chars is one the reader trusts and shouldn't.
	got := RenderReport(Report{Mission: "v1", Iteration: 1, Landed: strings.Repeat("x", 2000)})

	// Assert against a LITERAL, not truncationMark. Asserting the constant against
	// itself is vacuous: strings.Contains(got, "") is always true, so emptying the
	// constant would leave this test green while truncation became silent. That
	// mutant survived the first run of this suite, which is how it was caught.
	if truncationMark == "" {
		t.Fatal("truncationMark is empty — truncation would be silent")
	}
	if !strings.Contains(got, "[truncated]") {
		t.Fatalf("truncated output carries no truncation marker:\n%s", got)
	}
}

func TestRenderReport_TruncationIsRuneSafe(t *testing.T) {
	// The reports are full of em dashes and arrows. Cutting on a byte boundary
	// would emit invalid UTF-8 into a GitHub comment.
	got := RenderReport(Report{Mission: "v1", Iteration: 1, Landed: strings.Repeat("é—→", 500)})
	if !utf8.ValidString(got) {
		t.Fatalf("truncation produced invalid UTF-8: %q", got)
	}
	if len(got) > MaxReportChars {
		t.Fatalf("len = %d, want <= %d", len(got), MaxReportChars)
	}
}

func TestRenderReport_Deterministic(t *testing.T) {
	// A1. The same iteration state must produce a byte-identical artifact, or the
	// comms path stops being replayable.
	r := Report{Mission: "docs", Iteration: 7, Landed: "docs-3 landed", GoalDistance: "1 item left", CostUSD: 0.12, LogCommit: "663237dc7"}
	first := RenderReport(r)
	for i := 0; i < 50; i++ {
		if got := RenderReport(r); got != first {
			t.Fatalf("render %d differs:\n%q\nvs\n%q", i, got, first)
		}
	}
}

func TestRenderReport_CarriesTheLoadBearingFields(t *testing.T) {
	got := RenderReport(Report{
		Mission: "v1", Iteration: 325,
		Landed: "the hook is ARMED", GoalDistance: "N=12", CostUSD: 0.50, LogCommit: "70e453060",
	})
	for _, want := range []string{"325", "the hook is ARMED", "N=12", "70e453060", "0.50"} {
		if !strings.Contains(got, want) {
			t.Errorf("report is missing %q:\n%s", want, got)
		}
	}
}

func TestMaxReportChars_IsFourHundred(t *testing.T) {
	// Pinned deliberately: the design doc's success metric is a drop from ~52,677
	// chars/week to <8,000, which assumes ~400 per report. Changing this constant
	// changes that metric, so it should not be changeable by accident.
	if MaxReportChars != 400 {
		t.Fatalf("MaxReportChars = %d, want 400", MaxReportChars)
	}
}

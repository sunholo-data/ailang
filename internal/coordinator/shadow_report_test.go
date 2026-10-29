package coordinator

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

var updateShadowGolden = flag.Bool("update-shadow-golden", false, "rewrite testdata/shadow_report.golden.md")

var reportT0 = time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)

// reportFixture covers each row the report distinguishes: a shadow win with a
// reviewer note, a primary win against a circular merge with base drift, a tie,
// an ungraded comparison, and one still in progress.
func reportFixture() []*ShadowComparison {
	run := func(id, prov string, cost float64, diff string) *ShadowRunResult {
		return &ShadowRunResult{TaskID: id, Provider: prov, CostUSD: cost, DurationMS: 90000, NumTurns: 12, Diff: diff,
			SessionID: "sess-" + id, ArtifactGCSPath: "tasks/" + id, DiffStat: " m.ail | 3 ++-\n 1 file changed, 2 insertions(+), 1 deletion(-)"}
	}
	mech := func(p, s bool) *ShadowMechanicalGrade {
		return &ShadowMechanicalGrade{Primary: ShadowSideGrade{Pass: p}, Shadow: ShadowSideGrade{Pass: s}}
	}
	return []*ShadowComparison{
		{ParentID: "task-a", Primary: run("task-a", "pi", 0.40, "d"), Shadow: run("task-a-shadow", "motoko", 0.30, "d"),
			Mechanical: mech(true, true), GradeState: ShadowGradeDone, Verdict: ShadowVerdictShadowBetter, UngradedReason: "reviewer",
			Review: &ShadowReview{Winner: "shadow", Confidence: 0.8, StrongestObjection: "primary drops the error case"}},
		{ParentID: "task-b", Primary: run("task-b", "motoko", 0.20, "d"), Shadow: run("task-b-shadow", "pi", 0.10, ""),
			Mechanical: mech(true, false), GradeState: ShadowGradeDone, Verdict: ShadowVerdictPrimaryBetter, BaseDrift: true,
			Merged: &ShadowMergedFix{State: "merged", Number: 12, Circular: true}},
		{ParentID: "task-c", Primary: run("task-c", "pi", 0.10, "d"), Shadow: run("task-c-shadow", "motoko", 0.10, "d"),
			Mechanical: mech(true, true), GradeState: ShadowGradeDone, Verdict: ShadowVerdictEquivalent},
		{ParentID: "task-d", Primary: run("task-d", "pi", 0.10, "d"), Shadow: run("task-d-shadow", "motoko", 0.10, "d"),
			Mechanical: &ShadowMechanicalGrade{Primary: ShadowSideGrade{Ungraded: "apply_failed"}, Shadow: ShadowSideGrade{Pass: true}},
			GradeState: ShadowGradeDone, Verdict: ShadowVerdictUngraded, UngradedReason: "primary: apply_failed"},
		{ParentID: "task-e", Primary: run("task-e", "pi", 0.05, "d"), Shadow: run("task-e-shadow", "motoko", 0.05, "d"),
			GradeState: ShadowGradeGraded},
	}
}

func TestShadowReportGolden(t *testing.T) {
	got := BuildShadowReport(reportFixture(), reportT0.Add(-7*24*time.Hour), reportT0).Markdown()
	path := filepath.Join("testdata", "shadow_report.golden.md")
	if *updateShadowGolden {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update-shadow-golden to create it)", err)
	}
	if got != string(want) {
		t.Fatalf("report differs from %s:\n--- got\n%s", path, got)
	}
}

func TestShadowReportCounts(t *testing.T) {
	r := BuildShadowReport(reportFixture(), reportT0, reportT0)
	if r.Comparisons != 5 || r.Done != 4 || r.InProgress != 1 || r.BaseDrift != 1 || len(r.Disagreements) != 2 {
		t.Fatalf("counts: %+v", r)
	}
	if r.Wins["motoko"] != 2 || r.Wins["pi"] != 0 {
		t.Fatalf("wins = %v (task-a: motoko as shadow; task-b: motoko as primary)", r.Wins)
	}
	if r.Ungraded["apply_failed"] != 1 {
		t.Fatalf("ungraded tally = %v", r.Ungraded)
	}
	byName := map[string]ShadowHarnessStats{}
	for _, h := range r.Harnesses {
		byName[h.Harness] = h
	}
	// pi: graded on a, b(shadow, failed), c; d's primary was ungraded; e not graded yet.
	if pi := byName["pi"]; pi.Runs != 5 || pi.Graded != 3 || pi.Passed != 2 || pi.NoOutput != 1 {
		t.Fatalf("pi stats: %+v", pi)
	}
	if empty := BuildShadowReport(nil, reportT0, reportT0).Markdown(); empty == "" {
		t.Fatal("an empty window must still say so")
	}
}

// The report reads back what the stores wrote, in the window only.
func TestListShadowComparisonsSince(t *testing.T) {
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if _, err := s.RecordShadowRun(ctx, "task-a", ShadowSidePrimary, ShadowRunResult{TaskID: "task-a"}); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListShadowComparisonsSince(ctx, time.Now().Add(-time.Hour))
	if err != nil || len(got) != 1 {
		t.Fatalf("in window: %d, %v", len(got), err)
	}
	if got, _ = s.ListShadowComparisonsSince(ctx, time.Now().Add(time.Hour)); len(got) != 0 {
		t.Fatalf("future window returned %d", len(got))
	}
}

// The weekly post: Mondays from 09:00 UTC, once per ISO week however often the
// check runs (the coordinator scales to zero, so memory cannot be the guard),
// and only when some agent is shadowed.
func TestWeeklyShadowReport(t *testing.T) {
	monday := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC) // a Monday
	setup := func(t *testing.T, shadowed bool) (*Daemon, *SQLiteStore, *messaging.Store) {
		d, s := gradeSweepDaemon(t, &shadowRecordingDispatcher{})
		d.agentRegistry.GetAgentByID("pkg-sunholo-ailang-parse").Shadow.Enabled = shadowed
		msgs, err := messaging.OpenStore(filepath.Join(t.TempDir(), "m.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = msgs.Close() })
		d.msgStore = msgs
		return d, s, msgs
	}
	posts := func(t *testing.T, msgs *messaging.Store) int {
		list, err := msgs.ListInboxMessages(messaging.InboxListOptions{Inbox: "user"})
		if err != nil {
			t.Fatal(err)
		}
		return len(list)
	}
	t.Run("not monday", func(t *testing.T) {
		d, s, msgs := setup(t, true)
		d.postWeeklyShadowReport(s, monday.Add(24*time.Hour))
		if n := posts(t, msgs); n != 0 {
			t.Fatalf("posted %d on a Tuesday", n)
		}
	})
	t.Run("too early monday", func(t *testing.T) {
		d, s, msgs := setup(t, true)
		d.postWeeklyShadowReport(s, monday.Add(-2*time.Hour))
		if n := posts(t, msgs); n != 0 {
			t.Fatalf("posted %d before 09:00", n)
		}
	})
	t.Run("once per week across restarts", func(t *testing.T) {
		d, s, msgs := setup(t, true)
		d.postWeeklyShadowReport(s, monday)
		d.lastShadowReportCheck = time.Time{} // a restarted instance
		d.postWeeklyShadowReport(s, monday.Add(3*time.Hour))
		if n := posts(t, msgs); n != 1 {
			t.Fatalf("posted %d times in one week, want 1", n)
		}
	})
	t.Run("nothing shadowed", func(t *testing.T) {
		d, s, msgs := setup(t, false)
		d.postWeeklyShadowReport(s, monday)
		if n := posts(t, msgs); n != 0 {
			t.Fatalf("posted %d with no agent shadowed", n)
		}
	})
}

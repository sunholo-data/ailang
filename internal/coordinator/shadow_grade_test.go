package coordinator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

func gradeSweepDaemon(t *testing.T, rec *shadowRecordingDispatcher) (*Daemon, *SQLiteStore) {
	t.Helper()
	d, s := shadowTestDaemon(t, rec)
	policy := filepath.Join(t.TempDir(), "policy.toml")
	if err := os.WriteFile(policy, []byte("fs_sandbox = \"${WORKSPACE}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := NewAgentRegistry()
	a := laneAgent(motokoShadow())
	a.PolicyPath, a.Workspace = policy, "sunholo-data/ailang-parse"
	if err := reg.Register(a); err != nil {
		t.Fatal(err)
	}
	d.agentRegistry = reg
	return d, s
}

func seedComparison(t *testing.T, s *SQLiteStore, parent string, sides ...string) {
	t.Helper()
	for _, side := range sides {
		if _, err := s.RecordShadowRun(context.Background(), parent, side,
			ShadowRunResult{TaskID: parent, AgentID: "pkg-sunholo-ailang-parse", Diff: "d", BaseCommit: "b"}); err != nil {
			t.Fatal(err)
		}
	}
}

// A complete comparison gets exactly one grade job, in shadow_grade mode with
// the agent's policy and no model; it is claimed before dispatch, so the next
// tick does not dispatch it again. An incomplete one waits.
func TestShadowGradeSweepDispatchesOnce(t *testing.T) {
	rec := &shadowRecordingDispatcher{}
	d, s := gradeSweepDaemon(t, rec)
	seedComparison(t, s, "task-a", ShadowSidePrimary, ShadowSideShadow)
	seedComparison(t, s, "task-b", ShadowSidePrimary)

	d.dispatchShadowGrades()
	d.dispatchShadowGrades()

	if len(rec.calls) != 1 {
		t.Fatalf("grade dispatches = %d, want 1", len(rec.calls))
	}
	p := rec.calls[0]
	if p.JobMode != config.JobModeShadowGrade || p.ShadowParent != "task-a" || p.TaskID != "task-a-mechanical" || p.ShadowPhase != config.ShadowPhaseMechanical {
		t.Errorf("grade identity wrong: %+v", p)
	}
	if p.PolicyTOML == "" || p.ToolPolicy != "ailang_only" || p.Model != "" || p.Shadow || p.PushBranch != "" {
		t.Errorf("grade job must carry the agent's policy and nothing to write with: %+v", p)
	}
	if p.RepoURL == "" || p.ExecutorVariant != "pi" || p.Provider != "pi" {
		t.Errorf("grade job routing: repo=%q variant=%q provider=%q", p.RepoURL, p.ExecutorVariant, p.Provider)
	}
	c, _ := s.GetShadowComparison(context.Background(), "task-a")
	if c.GradeState != ShadowGradeDispatched || c.GradeDispatchedAt.IsZero() {
		t.Errorf("task-a state = %q", c.GradeState)
	}
}

func TestShadowGradeSweepSettlesWhatItCannotGrade(t *testing.T) {
	t.Run("dispatch fails", func(t *testing.T) {
		rec := &shadowRecordingDispatcher{err: errors.New("quota")}
		d, s := gradeSweepDaemon(t, rec)
		seedComparison(t, s, "task-a", ShadowSidePrimary, ShadowSideShadow)
		d.dispatchShadowGrades()
		c, _ := s.GetShadowComparison(context.Background(), "task-a")
		if c.Verdict != ShadowVerdictUngraded || c.GradeState != ShadowGradeGraded {
			t.Fatalf("verdict=%q state=%q", c.Verdict, c.GradeState)
		}
	})
	t.Run("unknown agent", func(t *testing.T) {
		rec := &shadowRecordingDispatcher{}
		d, s := gradeSweepDaemon(t, rec)
		if _, err := s.RecordShadowRun(context.Background(), "task-a", ShadowSidePrimary, ShadowRunResult{AgentID: "gone"}); err != nil {
			t.Fatal(err)
		}
		seedComparison(t, s, "task-a", ShadowSideShadow)
		d.dispatchShadowGrades()
		c, _ := s.GetShadowComparison(context.Background(), "task-a")
		if len(rec.calls) != 0 || c.UngradedReason != "agent_unknown" {
			t.Fatalf("calls=%d reason=%q", len(rec.calls), c.UngradedReason)
		}
	})
	t.Run("job never reported", func(t *testing.T) {
		rec := &shadowRecordingDispatcher{}
		d, s := gradeSweepDaemon(t, rec)
		seedComparison(t, s, "task-a", ShadowSidePrimary, ShadowSideShadow)
		if _, err := s.UpdateShadowComparison(context.Background(), "task-a", func(c *ShadowComparison) bool {
			c.GradeState, c.GradeDispatchedAt = ShadowGradeDispatched, time.Now().Add(-time.Hour)
			return true
		}); err != nil {
			t.Fatal(err)
		}
		d.dispatchShadowGrades()
		c, _ := s.GetShadowComparison(context.Background(), "task-a")
		if c.UngradedReason != "mechanical_timeout" || len(rec.calls) != 0 {
			t.Fatalf("reason=%q calls=%d", c.UngradedReason, len(rec.calls))
		}
	})
}

func TestShadowVerdictFromMechanical(t *testing.T) {
	pass, fail := ShadowSideGrade{Pass: true}, ShadowSideGrade{}
	for _, c := range []struct {
		p, s ShadowSideGrade
		want ShadowVerdict
	}{
		{pass, pass, ShadowVerdictEquivalent},
		{pass, fail, ShadowVerdictPrimaryBetter},
		{fail, pass, ShadowVerdictShadowBetter},
		{fail, fail, ShadowVerdictBothFailed},
		{ShadowSideGrade{Ungraded: "timeout"}, pass, ShadowVerdictUngraded},
		{pass, ShadowSideGrade{Pass: true, Ungraded: "apply_failed"}, ShadowVerdictUngraded},
	} {
		if got, _ := ShadowVerdictFromMechanical(ShadowMechanicalGrade{Primary: c.p, Shadow: c.s}); got != c.want {
			t.Errorf("%+v / %+v: got %q, want %q", c.p, c.s, got, c.want)
		}
	}
}

// Two instances list the same comparison: only the one that wins the claim
// dispatches. The loser's snapshot still says "not dispatched".
func TestShadowGradeClaimRace(t *testing.T) {
	rec := &shadowRecordingDispatcher{}
	d, s := gradeSweepDaemon(t, rec)
	seedComparison(t, s, "task-a", ShadowSidePrimary, ShadowSideShadow)
	snapshot, _ := s.GetShadowComparison(context.Background(), "task-a")
	d.dispatchShadowPhase(s, snapshot, shadowPhases[0], time.Now())
	d.dispatchShadowPhase(s, snapshot, shadowPhases[0], time.Now()) // the other instance, same stale view
	if len(rec.calls) != 1 {
		t.Fatalf("grade dispatches = %d, want 1", len(rec.calls))
	}
}

// A job that reports between the sweep's listing and its settle wins: the
// settle sees the stage has moved and leaves the result alone.
func TestSettleShadowNeverOverwritesALandedResult(t *testing.T) {
	d, s := gradeSweepDaemon(t, &shadowRecordingDispatcher{})
	seedComparison(t, s, "task-a", ShadowSidePrimary, ShadowSideShadow)
	if _, err := s.UpdateShadowComparison(context.Background(), "task-a", func(c *ShadowComparison) bool {
		c.GradeState, c.Verdict = ShadowGradeGraded, ShadowVerdictPrimaryBetter // the job's result
		return true
	}); err != nil {
		t.Fatal(err)
	}
	d.settleShadow(s, "task-a", ShadowGradeDispatched, "mechanical_timeout", shadowPhases[0].onAbandon)
	c, _ := s.GetShadowComparison(context.Background(), "task-a")
	if c.Verdict != ShadowVerdictPrimaryBetter || c.UngradedReason != "" {
		t.Fatalf("a late settle overwrote the job's result: verdict=%q reason=%q", c.Verdict, c.UngradedReason)
	}
}

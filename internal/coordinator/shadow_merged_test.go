package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// stubGitHub replaces both GitHub reads for one test.
func stubGitHub(t *testing.T, pr *LandedPR, head string, files []string) {
	t.Helper()
	oldPR, oldDetails := settledPRLookup, mergedPRDetailsLookup
	t.Cleanup(func() { settledPRLookup, mergedPRDetailsLookup = oldPR, oldDetails })
	settledPRLookup = func(context.Context, string, string, string) (*LandedPR, error) { return pr, nil }
	mergedPRDetailsLookup = func(context.Context, string, string, int) (string, string, []string, error) {
		return head, "merge-sha", files, nil
	}
	t.Setenv("GITHUB_TOKEN", "test-token")
}

// gradedComparison is a comparison the mechanical grade has finished with.
func gradedComparison(t *testing.T, s *SQLiteStore, primaryPass, shadowPass bool) {
	t.Helper()
	ctx := context.Background()
	for side, files := range map[string][]string{
		ShadowSidePrimary: {"pkg/m.ail", "pkg/m_test.ail"},
		ShadowSideShadow:  {"pkg/m.ail"},
	} {
		if _, err := s.RecordShadowRun(ctx, "task-a", side, ShadowRunResult{
			TaskID: "task-a", AgentID: "pkg-sunholo-ailang-parse", Diff: "d", BaseCommit: "b",
			HeadCommit: "head-" + side, ChangedFiles: files,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.UpdateShadowComparison(ctx, "task-a", func(c *ShadowComparison) bool {
		c.Mechanical = &ShadowMechanicalGrade{Primary: ShadowSideGrade{Pass: primaryPass}, Shadow: ShadowSideGrade{Pass: shadowPass}}
		c.Verdict, c.UngradedReason = ShadowVerdictFromMechanical(*c.Mechanical)
		c.GradeState = ShadowGradeGraded
		return true
	}); err != nil {
		t.Fatal(err)
	}
}

func getComparison(t *testing.T, s *SQLiteStore) *ShadowComparison {
	t.Helper()
	c, err := s.GetShadowComparison(context.Background(), "task-a")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestShadowMergedSweep(t *testing.T) {
	merged := &LandedPR{Number: 7, State: "MERGED"}
	cases := []struct {
		name      string
		pr        *LandedPR
		head      string
		files     []string
		wantState string
		wantStage string
		circular  bool
	}{
		{"still open", nil, "", nil, "", ShadowGradeGraded, false},
		{"closed unmerged", &LandedPR{Number: 7, State: "CLOSED"}, "", nil, "closed", ShadowGradeReview, false},
		{"merged as produced, with tests", merged, "head-primary", []string{"pkg/m.ail", "pkg/m_test.ail"}, "merged", ShadowGradeMergedTests, true},
		{"merged after human edits", merged, "head-edited", []string{"pkg/m.ail", "pkg/m_test.ail"}, "merged", ShadowGradeMergedTests, false},
		{"merged without tests", merged, "head-edited", []string{"pkg/m.ail"}, "merged", ShadowGradeReview, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, s := gradeSweepDaemon(t, &shadowRecordingDispatcher{})
			gradedComparison(t, s, true, false)
			stubGitHub(t, tc.pr, tc.head, tc.files)
			d.sweepShadowMerged(s, time.Now())
			c := getComparison(t, s)
			if c.GradeState != tc.wantStage {
				t.Fatalf("stage = %q, want %q", c.GradeState, tc.wantStage)
			}
			if tc.wantState == "" {
				if c.Merged != nil {
					t.Fatalf("an open PR recorded a merged fix: %+v", c.Merged)
				}
				return
			}
			if c.Merged == nil || c.Merged.State != tc.wantState || c.Merged.Circular != tc.circular {
				t.Fatalf("merged = %+v, want state %q circular %v", c.Merged, tc.wantState, tc.circular)
			}
			if tc.wantState == "merged" && (c.Merged.PrimaryOverlap != jaccard([]string{"pkg/m.ail", "pkg/m_test.ail"}, tc.files) ||
				c.Merged.ShadowOverlap != jaccard([]string{"pkg/m.ail"}, tc.files)) {
				t.Errorf("overlap primary=%v shadow=%v", c.Merged.PrimaryOverlap, c.Merged.ShadowOverlap)
			}
		})
	}
}

// A PR nobody settles is given up on, not waited for forever.
func TestShadowMergedSweepGivesUp(t *testing.T) {
	d, s := gradeSweepDaemon(t, &shadowRecordingDispatcher{})
	gradedComparison(t, s, true, false)
	stubGitHub(t, nil, "", nil)
	d.sweepShadowMerged(s, time.Now().Add(shadowMergedGiveUpAfter+time.Hour))
	if c := getComparison(t, s); c.Merged == nil || c.Merged.State != "unsettled" || c.GradeState != ShadowGradeReview {
		t.Fatalf("merged=%+v stage=%q", c.Merged, c.GradeState)
	}
}

// The sweep reads GitHub at most once per interval.
func TestShadowMergedSweepThrottled(t *testing.T) {
	d, s := gradeSweepDaemon(t, &shadowRecordingDispatcher{})
	gradedComparison(t, s, true, false)
	calls := 0
	old := settledPRLookup
	t.Cleanup(func() { settledPRLookup = old })
	settledPRLookup = func(context.Context, string, string, string) (*LandedPR, error) { calls++; return nil, nil }
	t.Setenv("GITHUB_TOKEN", "test-token")
	now := time.Now()
	d.sweepShadowMerged(s, now)
	d.sweepShadowMerged(s, now.Add(time.Minute))
	if calls != 1 {
		t.Fatalf("PR lookups = %d, want 1 within the interval", calls)
	}
}

// The review stage: a decisive verdict needs no reviewer and finishes; a tie
// with no reviewer configured finishes on the layered verdict; a tie with a
// reviewer dispatches the review phase with the model, the cap and the task.
func TestShadowReviewStage(t *testing.T) {
	ctx := context.Background()
	toReview := func(t *testing.T, s *SQLiteStore) {
		if _, err := s.UpdateShadowComparison(ctx, "task-a", func(c *ShadowComparison) bool {
			c.GradeState = ShadowGradeReview
			return true
		}); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("decisive", func(t *testing.T) {
		rec := &shadowRecordingDispatcher{}
		d, s := gradeSweepDaemon(t, rec)
		gradedComparison(t, s, true, false)
		toReview(t, s)
		d.dispatchShadowGrades()
		if c := getComparison(t, s); c.GradeState != ShadowGradeDone || c.Verdict != ShadowVerdictPrimaryBetter || len(rec.calls) != 0 {
			t.Fatalf("stage=%q verdict=%q calls=%d", c.GradeState, c.Verdict, len(rec.calls))
		}
	})
	t.Run("decisive, reviewer configured", func(t *testing.T) {
		rec := &shadowRecordingDispatcher{}
		d, s := gradeSweepDaemon(t, rec)
		d.agentRegistry.GetAgentByID("pkg-sunholo-ailang-parse").Shadow.ReviewerModel = "gpt6-1-sol"
		gradedComparison(t, s, true, false)
		toReview(t, s)
		d.dispatchShadowGrades()
		if c := getComparison(t, s); c.GradeState != ShadowGradeDone || len(rec.calls) != 0 {
			t.Fatalf("a decisive verdict spent a review: stage=%q calls=%d", c.GradeState, len(rec.calls))
		}
	})
	t.Run("tie, no reviewer", func(t *testing.T) {
		rec := &shadowRecordingDispatcher{}
		d, s := gradeSweepDaemon(t, rec)
		gradedComparison(t, s, true, true)
		toReview(t, s)
		d.dispatchShadowGrades()
		if c := getComparison(t, s); c.GradeState != ShadowGradeDone || c.Verdict != ShadowVerdictEquivalent || len(rec.calls) != 0 {
			t.Fatalf("stage=%q verdict=%q calls=%d", c.GradeState, c.Verdict, len(rec.calls))
		}
	})
	t.Run("tie, reviewer", func(t *testing.T) {
		rec := &shadowRecordingDispatcher{}
		d, s := gradeSweepDaemon(t, rec)
		d.agentRegistry.GetAgentByID("pkg-sunholo-ailang-parse").Shadow.ReviewerModel = "gpt6-1-sol"
		if err := s.CreateTask(ctx, &TaskRecord{ID: "task-a", Title: "x", Content: "fix the parser"}); err != nil {
			t.Fatal(err)
		}
		gradedComparison(t, s, false, false)
		toReview(t, s)
		d.dispatchShadowGrades()
		if len(rec.calls) != 1 {
			t.Fatalf("review dispatches = %d, want 1", len(rec.calls))
		}
		p := rec.calls[0]
		if p.ShadowPhase != config.ShadowPhaseReview || p.Model != "gpt6-1-sol" || p.MaxCostUSD != defaultShadowReviewerMaxCostUSD || p.Directive != "fix the parser" {
			t.Fatalf("review dispatch: phase=%q model=%q cap=%v directive=%q", p.ShadowPhase, p.Model, p.MaxCostUSD, p.Directive)
		}
		if c := getComparison(t, s); c.GradeState != ShadowGradeReviewing {
			t.Fatalf("stage = %q", c.GradeState)
		}
	})
	t.Run("reviewer never reports", func(t *testing.T) {
		d, s := gradeSweepDaemon(t, &shadowRecordingDispatcher{})
		gradedComparison(t, s, true, true)
		if _, err := s.UpdateShadowComparison(ctx, "task-a", func(c *ShadowComparison) bool {
			c.GradeState, c.GradeDispatchedAt = ShadowGradeReviewing, time.Now().Add(-time.Hour)
			return true
		}); err != nil {
			t.Fatal(err)
		}
		d.dispatchShadowGrades()
		c := getComparison(t, s)
		if c.GradeState != ShadowGradeDone || c.Verdict != ShadowVerdictUngraded || c.UngradedReason != "review_timeout" {
			t.Fatalf("stage=%q verdict=%q reason=%q", c.GradeState, c.Verdict, c.UngradedReason)
		}
	})
}

func TestLayeredAndFinalShadowVerdict(t *testing.T) {
	pass, fail := ShadowSideGrade{Pass: true}, ShadowSideGrade{}
	mech := func(p, s ShadowSideGrade) *ShadowMechanicalGrade {
		return &ShadowMechanicalGrade{Primary: p, Shadow: s}
	}
	cases := []struct {
		name       string
		c          ShadowComparison
		want       ShadowVerdict
		wantReview bool
		final      ShadowVerdict
	}{
		{"mechanical decisive", ShadowComparison{Mechanical: mech(pass, fail)}, ShadowVerdictPrimaryBetter, false, ShadowVerdictPrimaryBetter},
		{"tie goes to review", ShadowComparison{Mechanical: mech(pass, pass), Review: &ShadowReview{Winner: "shadow"}}, ShadowVerdictEquivalent, true, ShadowVerdictShadowBetter},
		{"merged tests outrank", ShadowComparison{Mechanical: mech(pass, pass), MergedTests: mech(fail, pass)}, ShadowVerdictShadowBetter, false, ShadowVerdictShadowBetter},
		{"circular merge is ignored", ShadowComparison{Mechanical: mech(fail, pass), MergedTests: mech(pass, fail), Merged: &ShadowMergedFix{Circular: true}}, ShadowVerdictShadowBetter, false, ShadowVerdictShadowBetter},
		{"layers disagree", ShadowComparison{Mechanical: mech(pass, fail), MergedTests: mech(fail, pass), Review: &ShadowReview{Winner: "primary"}}, ShadowVerdictShadowBetter, true, ShadowVerdictPrimaryBetter},
		{"needed reviewer failed", ShadowComparison{Mechanical: mech(pass, pass), Review: &ShadowReview{Ungraded: "reviewer_unavailable: x"}}, ShadowVerdictEquivalent, true, ShadowVerdictUngraded},
		{"ungraded merged tests fall back", ShadowComparison{Mechanical: mech(pass, fail), MergedTests: mech(ShadowSideGrade{Ungraded: "apply_failed"}, pass)}, ShadowVerdictPrimaryBetter, false, ShadowVerdictPrimaryBetter},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, _, needs := LayeredShadowVerdict(&tc.c)
			if v != tc.want || needs != tc.wantReview {
				t.Fatalf("layered = %q needs=%v, want %q %v", v, needs, tc.want, tc.wantReview)
			}
			if f, _ := FinalShadowVerdict(&tc.c); f != tc.final {
				t.Fatalf("final = %q, want %q", f, tc.final)
			}
		})
	}
}

func TestJaccard(t *testing.T) {
	if j := jaccard([]string{"a", "b"}, []string{"b", "c"}); j != 1.0/3 {
		t.Fatalf("jaccard = %v", j)
	}
	if jaccard(nil, nil) != 0 || jaccard([]string{"a"}, []string{"a"}) != 1 {
		t.Fatal("jaccard edge cases")
	}
}

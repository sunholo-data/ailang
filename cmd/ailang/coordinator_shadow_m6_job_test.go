package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/policytool"
)

// pushCommit makes the "merged PR": a commit on the remote with these edits,
// returning its SHA.
func (f *gradeFixture) pushCommit(t *testing.T, edits map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "clone", "-q", f.remote, ".")
	for rel, content := range edits {
		writeGradeFile(t, dir, rel, content)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "merged fix")
	runGit(t, dir, "push", "-q", "origin", "HEAD:refs/heads/merged")
	return runGit(t, dir, "rev-parse", "HEAD")
}

func (f *gradeFixture) toStage(t *testing.T, stage string, fn func(*coordinator.ShadowComparison)) {
	t.Helper()
	if _, err := f.store.UpdateShadowComparison(context.Background(), "task-p", func(c *coordinator.ShadowComparison) bool {
		c.GradeState = stage
		if fn != nil {
			fn(c)
		}
		return true
	}); err != nil {
		t.Fatal(err)
	}
}

// The merged PR's test file is laid over each candidate and run: a candidate
// whose module the accepted tests reject fails, the other passes; only test
// ops run, and the candidate's own module is never overwritten.
func TestShadowMergedTestsPhase(t *testing.T) {
	f := newGradeFixture(t)
	mergeSHA := f.pushCommit(t, map[string]string{"pkg/m.ail": "module pkg/m\nmerged\n", "pkg/m_test.ail": "tests for m\n"})
	f.seed(t, f.diff(t, "task-p", map[string]string{"pkg/m.ail": "module pkg/m\nprimary fix\n"}),
		f.diff(t, "task-p-shadow", map[string]string{"pkg/m.ail": "module pkg/m\nBAD shadow fix\n"}))
	f.toStage(t, coordinator.ShadowGradeMergedTesting, func(c *coordinator.ShadowComparison) {
		c.Mechanical = &coordinator.ShadowMechanicalGrade{Primary: coordinator.ShadowSideGrade{Pass: true}, Shadow: coordinator.ShadowSideGrade{Pass: true}}
		c.Merged = &coordinator.ShadowMergedFix{State: "merged", MergeCommit: mergeSHA, TestFiles: []string{"pkg/m_test.ail", "pkg/gone_test.ail"}}
	})
	var ran []string
	run := func(dir, _ string, req policytool.Request) policytool.Response {
		ran = append(ran, req.Op+":"+req.Path)
		mod, _ := os.ReadFile(filepath.Join(dir, "m.ail"))
		if strings.Contains(string(mod), "merged") {
			t.Errorf("the merged module overwrote the candidate's own fix")
		}
		if strings.Contains(string(mod), "BAD") {
			return policytool.Response{OK: false, ExitCode: 1}
		}
		return policytool.Response{OK: true}
	}
	if err := gradeShadowMergedTests(context.Background(), f.store, "task-p", f.url, gradeTestPolicy, t.TempDir(), run); err != nil {
		t.Fatal(err)
	}
	c, _ := f.store.GetShadowComparison(context.Background(), "task-p")
	if c.GradeState != coordinator.ShadowGradeReview || c.MergedTests == nil {
		t.Fatalf("stage=%q merged_tests=%+v", c.GradeState, c.MergedTests)
	}
	if !c.MergedTests.Primary.Pass || c.MergedTests.Shadow.Pass {
		t.Fatalf("merged tests: primary %+v shadow %+v", c.MergedTests.Primary, c.MergedTests.Shadow)
	}
	if c.Verdict != coordinator.ShadowVerdictPrimaryBetter {
		t.Fatalf("verdict = %q: the accepted fix's tests should outrank the tied mechanical layer", c.Verdict)
	}
	for _, r := range ran {
		if r != "test:m_test.ail" {
			t.Fatalf("ran %q; only the merged test file, relative to its package", r)
		}
	}
}

func fakeReviewer(answer string, cost float64, err error, calls *int) func(string) (*shadowReviewer, error) {
	return func(string) (*shadowReviewer, error) {
		return &shadowReviewer{
			call: func(_, _, _ string) (string, float64, error) {
				*calls++
				return answer, cost, err
			},
			estimate: func(in, out int) float64 { return float64(in+out) * 1e-6 },
			maxOut:   reviewExpectedOutputTokens,
		}, nil
	}
}

// The reviewer's A/B answer maps back to primary/shadow through the
// per-comparison order, and its call settles a tie.
func TestShadowReviewPhase(t *testing.T) {
	answerFor := func(primary bool) string {
		letter := "B"
		if primary == reviewPrimaryIsA("task-p") {
			letter = "A"
		}
		return `{"winner":"` + letter + `","strongest_objection":"misses the empty case","confidence":0.7}`
	}
	cases := []struct {
		name     string
		answer   string
		err      error
		maxCost  string
		resolve  error
		want     coordinator.ShadowVerdict
		ungraded string
		calls    int
	}{
		{"picks shadow", answerFor(false), nil, "0.25", nil, coordinator.ShadowVerdictShadowBetter, "", 1},
		{"picks primary", "```json\n" + answerFor(true) + "\n```", nil, "0.25", nil, coordinator.ShadowVerdictPrimaryBetter, "", 1},
		{"unparseable", "I think A", nil, "0.25", nil, coordinator.ShadowVerdictUngraded, "reviewer_invalid", 1},
		{"no objection", `{"winner":"A","strongest_objection":"","confidence":0.9}`, nil, "0.25", nil, coordinator.ShadowVerdictUngraded, "reviewer_invalid", 1},
		{"unreachable", "", errors.New("401"), "0.25", nil, coordinator.ShadowVerdictUngraded, "reviewer_unavailable", 1},
		{"over budget", answerFor(true), nil, "0.000001", nil, coordinator.ShadowVerdictUngraded, "reviewer_over_budget", 0},
		{"unknown model", "", nil, "0.25", errors.New("model not in models.yml"), coordinator.ShadowVerdictUngraded, "reviewer_unavailable", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newGradeFixture(t)
			f.seed(t, f.diff(t, "task-p", map[string]string{"pkg/m.ail": "primary\n"}), f.diff(t, "task-p-shadow", map[string]string{"pkg/m.ail": "shadow\n"}))
			f.toStage(t, coordinator.ShadowGradeReviewing, func(c *coordinator.ShadowComparison) {
				c.Mechanical = &coordinator.ShadowMechanicalGrade{Primary: coordinator.ShadowSideGrade{Pass: true}, Shadow: coordinator.ShadowSideGrade{Pass: true}}
			})
			calls := 0
			resolve := fakeReviewer(tc.answer, 0.01, tc.err, &calls)
			if tc.resolve != nil {
				resolve = func(string) (*shadowReviewer, error) { return nil, tc.resolve }
			}
			if err := reviewShadowComparison(context.Background(), f.store, "task-p", "fix the parser", "gpt6-1-sol", tc.maxCost, resolve); err != nil {
				t.Fatal(err)
			}
			c, _ := f.store.GetShadowComparison(context.Background(), "task-p")
			if c.GradeState != coordinator.ShadowGradeDone || c.Verdict != tc.want || !strings.HasPrefix(c.UngradedReason, tc.ungraded) || calls != tc.calls {
				t.Fatalf("stage=%q verdict=%q reason=%q calls=%d", c.GradeState, c.Verdict, c.UngradedReason, calls)
			}
		})
	}
}

// The prompt carries the task and both diffs under the per-comparison order.
func TestShadowReviewPromptOrder(t *testing.T) {
	c := &coordinator.ShadowComparison{ParentID: "task-p",
		Primary: &coordinator.ShadowRunResult{Diff: "PRIMARY-DIFF"}, Shadow: &coordinator.ShadowRunResult{Diff: "SHADOW-DIFF"}}
	for _, primaryIsA := range []bool{true, false} {
		p := shadowReviewPrompt(c, "the task", primaryIsA)
		a, b := strings.Index(p, "PRIMARY-DIFF"), strings.Index(p, "SHADOW-DIFF")
		if !strings.Contains(p, "the task") || (a < b) != primaryIsA {
			t.Fatalf("primaryIsA=%v: order wrong\n%s", primaryIsA, p)
		}
	}
	// Over many IDs the order is not always the same.
	seen := map[bool]bool{}
	for i := 0; i < 50; i++ {
		seen[reviewPrimaryIsA("task-"+time.Duration(i).String())] = true
	}
	if len(seen) != 2 {
		t.Fatal("the A/B order never varies: position bias would read as a harness preference")
	}
}

// The reviewer end to end through design-quorum's resolver and a real model.
// Opt-in (it spends a few cents):
//
//	AILANG_SHADOW_REVIEW_LIVE_MODEL=gemini-4-argon go test ./cmd/ailang/ -run ShadowReviewLive -count=1 -v
func TestShadowReviewLive(t *testing.T) {
	model := os.Getenv("AILANG_SHADOW_REVIEW_LIVE_MODEL") //nolint:forbidigo // opt-in live test knob, never read by the product
	if model == "" {
		t.Skip("set AILANG_SHADOW_REVIEW_LIVE_MODEL to a models.yml reviewer")
	}
	c := &coordinator.ShadowComparison{ParentID: "task-live",
		Primary:    &coordinator.ShadowRunResult{Status: "completed", Diff: "--- a/m.ail\n+++ b/m.ail\n@@ -1,3 +1,3 @@\n export pure func safeDiv(a: int, b: int) -> int {\n-  a / b\n+  if b == 0 then 0 else a / b\n }\n"},
		Shadow:     &coordinator.ShadowRunResult{Status: "completed", Diff: "--- a/m.ail\n+++ b/m.ail\n@@ -1,3 +1,3 @@\n export pure func safeDiv(a: int, b: int) -> int {\n-  a / b\n+  a / (b + 1)\n }\n"},
		Mechanical: &coordinator.ShadowMechanicalGrade{Primary: coordinator.ShadowSideGrade{Pass: true}, Shadow: coordinator.ShadowSideGrade{Pass: true}},
	}
	r := runShadowReview(c, "safeDiv must return 0 when dividing by zero instead of crashing; otherwise behave like a / b.", model, "0.25", resolveShadowReviewer)
	t.Logf("review: %+v", r)
	if r.Ungraded != "" {
		t.Fatalf("reviewer gave no call: %s", r.Ungraded)
	}
	if r.Winner != coordinator.ShadowSidePrimary {
		t.Fatalf("winner = %q; the shadow's a/(b+1) changes every result", r.Winner)
	}
}

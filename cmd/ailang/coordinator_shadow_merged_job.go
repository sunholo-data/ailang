package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/gitexec"
)

// The merged_tests phase (M-AILANG-LANE-EXECUTOR-SHADOW M6, design layer 2):
// run the merged PR's *_test.ail files against each candidate. Each side is set
// up exactly as for the mechanical grade, then the accepted PR's test files are
// laid over it from the merge commit and run. Only test files are overlaid —
// never modules, which would overwrite the candidate's own fix.
func gradeShadowMergedTests(ctx context.Context, store coordinator.ShadowComparisonStore, parent, repoURL, policyTOML, root string, run gateRunner) error {
	if err := relaxModuleNames(); err != nil {
		return err
	}
	c, err := store.GetShadowComparison(ctx, parent)
	if err != nil {
		return fmt.Errorf("shadow merged tests: %w", err)
	}
	if c.Merged == nil || c.Merged.MergeCommit == "" || len(c.Merged.TestFiles) == 0 {
		return fmt.Errorf("shadow merged tests: %s has no merged test files", parent)
	}
	side := func(name string, r *coordinator.ShadowRunResult) coordinator.ShadowSideGrade {
		dir := filepath.Join(root, name)
		if g, done := prepareShadowSide(ctx, dir, r, repoURL); done {
			return g
		}
		tests, err := overlayMergedTests(ctx, dir, c.Merged.MergeCommit, c.Merged.TestFiles)
		if err != nil {
			return coordinator.ShadowSideGrade{Ungraded: "merge_fetch_failed"}
		}
		if len(tests) == 0 {
			return coordinator.ShadowSideGrade{Ungraded: "no_merged_tests"}
		}
		return runGatePlan(ctx, dir, policyTOML, mergedTestPlan(dir, tests), run)
	}
	m := coordinator.ShadowMechanicalGrade{Primary: side("primary", c.Primary), Shadow: side("shadow", c.Shadow), GradedAt: time.Now().UTC()}
	return writeShadowPhase(ctx, store, parent, coordinator.ShadowGradeMergedTesting, func(cur *coordinator.ShadowComparison) {
		cur.MergedTests = &m
		cur.Verdict, cur.UngradedReason, _ = coordinator.LayeredShadowVerdict(cur)
		cur.GradeState = coordinator.ShadowGradeReview
	})
}

// overlayMergedTests checks the merged PR's test files out of the merge commit
// into dir, returning the ones that exist there (a test the PR deleted has
// nothing to run).
func overlayMergedTests(ctx context.Context, dir, mergeCommit string, files []string) ([]string, error) {
	if out, err := gitexec.CommandContext(ctx, "-C", dir, "fetch", "-q", "--depth", "1", "origin", mergeCommit).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: shadow merged tests: fetch %s: %v: %s\n", mergeCommit, err, out)
		return nil, err
	}
	var present []string
	for _, f := range files {
		if err := gitexec.CommandContext(ctx, "-C", dir, "checkout", mergeCommit, "--", f).Run(); err == nil {
			present = append(present, f)
		}
	}
	sort.Strings(present)
	return present, nil
}

// mergedTestPlan runs each test file on its own (`test <file>`), sandboxed to
// its package directory like every other gate.
func mergedTestPlan(dir string, tests []string) []gateGroup {
	byRoot := map[string][]string{}
	for _, t := range tests {
		root, ok := packageRoot(dir, t)
		if !ok {
			root = "."
		}
		rel := t
		if root != "." {
			rel = strings.TrimPrefix(t, root+"/")
		}
		byRoot[root] = append(byRoot[root], rel)
	}
	roots := make([]string, 0, len(byRoot))
	for r := range byRoot {
		roots = append(roots, r)
	}
	sort.Strings(roots)
	plan := make([]gateGroup, 0, len(roots))
	for _, r := range roots {
		g := gateGroup{root: r}
		for _, t := range byRoot[r] {
			g.reqs = append(g.reqs, testRequest(t, ""))
		}
		plan = append(plan, g)
	}
	return plan
}

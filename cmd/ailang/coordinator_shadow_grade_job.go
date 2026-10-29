package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/gitexec"
	"github.com/sunholo-data/ailang/internal/policytool"
	fsstore "github.com/sunholo-data/ailang/internal/storage/firestore"
)

// The shadow grade job (M-AILANG-LANE-EXECUTOR-SHADOW M5, design layer 1):
// AILANG_JOB_MODE=shadow_grade on the agent executor image. No agent runs and
// nothing is pushed. For each side of the comparison it clones the repo at that
// run's base commit, applies the run's diff, and runs only AILANG gates —
// `check` on each changed module, `test` on each changed package — through the
// policy tool under the agent's own restricted policy, sandboxed to the
// module's package directory. Repo `make` targets are
// never run: they would execute repo code outside the policy.
//
// Bounds: the policy tool kills each gate at 120s; the job stops starting gates
// at shadowGradeJobBound; the coordinator marks a job that never reports
// ungraded(grade_timeout) after 30 minutes.

const shadowGradeJobBound = 900 * time.Second

// gateRunner runs one policy-tool request in a sandbox. The job's runner is the
// real policy tool; tests substitute one that needs no ailang binary.
type gateRunner func(dir, policyPath string, req policytool.Request) policytool.Response

// policyToolGateRunner runs gates with this binary (the job's ailang).
func policyToolGateRunner(dir, policyPath string, req policytool.Request) policytool.Response {
	exe, _ := os.Executable()
	return policyToolGateRunnerWith(exe)(dir, policyPath, req)
}

// policyToolGateRunnerWith runs gates with the named ailang binary. It ignores
// dir: the materialised policy's fs_sandbox already names it, and the policy,
// not the caller, decides where a gate may look.
func policyToolGateRunnerWith(bin string) gateRunner {
	return func(_, policyPath string, req policytool.Request) policytool.Response {
		host, err := policytool.Open(policyPath)
		if err != nil {
			return policytool.Response{Refused: err.Error()}
		}
		defer host.Close()
		if bin != "" {
			host.SetBinary(bin)
		}
		return host.Dispatch(req)
	}
}

// runShadowGradeJob is execute-job's shadow_grade mode.
func runShadowGradeJob() error {
	parent := config.ShadowParent()
	if parent == "" {
		return fmt.Errorf("shadow grade: %s is required", config.EnvShadowParent)
	}
	ctx, cancel := context.WithTimeout(context.Background(), shadowGradeJobBound)
	defer cancel()
	project, err := config.CloudProject(ctx)
	if err != nil {
		return fmt.Errorf("shadow grade: %w", err)
	}
	client, err := fsstore.NewClientForProject(ctx, project)
	if err != nil {
		return fmt.Errorf("shadow grade: %w", err)
	}
	defer client.Close()
	if config.GitHubToken() != "" {
		_ = gitexec.CommandContext(ctx, "config", "--global", "credential.helper", envTokenCredentialHelper).Run()
	}

	store := fsstore.NewCoordinatorStore(client)
	root := filepath.Join(cloudWorkspaceRoot, parent+"-"+config.ShadowPhase())
	switch phase := config.ShadowPhase(); phase {
	case config.ShadowPhaseMechanical:
		return gradeShadowComparison(ctx, store, parent, config.RepoURL(), config.AgentPolicyTOML(), root, policyToolGateRunner)
	case config.ShadowPhaseMergedTests:
		return gradeShadowMergedTests(ctx, store, parent, config.RepoURL(), config.AgentPolicyTOML(), root, policyToolGateRunner)
	case config.ShadowPhaseReview:
		directive, err := resolveJobDirective(ctx, config.DirectiveSource(), config.TaskID(), openDirectiveStore(project))
		if err != nil {
			directive = "" // the reviewer still compares the diffs; the prompt says the task is unknown
		}
		return reviewShadowComparison(ctx, store, parent, directive, config.Model(), config.MaxCostUSD(), resolveShadowReviewer)
	default:
		return fmt.Errorf("shadow grade: unknown %s %q", config.EnvShadowPhase, phase)
	}
}

// relaxModuleNames grades what the gates are for — does it type-check, do its
// tests pass — not the module-path naming rule: lane programs are loose files
// whose module names need not mirror their path, and the lane does not enforce
// it. The policy tool's ailang child inherits the variable. Measured 2026-10-19:
// the first dev comparison failed a correct lane program on MOD010.
func relaxModuleNames() error {
	if err := os.Setenv(config.EnvRelaxModules, "1"); err != nil {
		return fmt.Errorf("shadow grade: %w", err)
	}
	return nil
}

// writeShadowPhase lands a phase's result, only while the comparison is still
// in the stage this job was dispatched for: if the coordinator already gave up
// on the job, the late result is dropped rather than overwriting a settled one.
// The write gets its own deadline, because a timeout is a result too.
func writeShadowPhase(ctx context.Context, store coordinator.ShadowComparisonStore, parent, inFlight string, fn func(*coordinator.ShadowComparison)) error {
	wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	wrote, err := store.UpdateShadowComparison(wctx, parent, func(cur *coordinator.ShadowComparison) bool {
		if cur.GradeState != inFlight {
			return false
		}
		fn(cur)
		return true
	})
	if err != nil {
		return fmt.Errorf("shadow grade: writing %s: %w", parent, err)
	}
	fmt.Printf("execute-job: shadow %s %s written=%v\n", config.ShadowPhase(), parent, wrote)
	return nil
}

// gradeShadowComparison grades both sides and writes the result back. The write
// only lands while the comparison is still "dispatched": if the coordinator
// already gave up on this job (grade_timeout), the late grade is dropped rather
// than overwriting a settled verdict.
func gradeShadowComparison(ctx context.Context, store coordinator.ShadowComparisonStore, parent, repoURL, policyTOML, root string, run gateRunner) error {
	if err := relaxModuleNames(); err != nil {
		return err
	}
	c, err := store.GetShadowComparison(ctx, parent)
	if err != nil {
		return fmt.Errorf("shadow grade: %w", err)
	}
	if !c.Complete() {
		return fmt.Errorf("shadow grade: %s is not complete", parent)
	}
	m := coordinator.ShadowMechanicalGrade{
		Primary: gradeShadowSide(ctx, filepath.Join(root, "primary"), c.Primary, repoURL, policyTOML, run),
		Shadow:  gradeShadowSide(ctx, filepath.Join(root, "shadow"), c.Shadow, repoURL, policyTOML, run),
	}
	m.GradedAt = time.Now().UTC()
	verdict, reason := coordinator.ShadowVerdictFromMechanical(m)
	fmt.Printf("execute-job: shadow grade %s: verdict=%s %s\n", parent, verdict, reason)
	return writeShadowPhase(ctx, store, parent, coordinator.ShadowGradeDispatched, func(cur *coordinator.ShadowComparison) {
		cur.Mechanical, cur.Verdict, cur.UngradedReason = &m, verdict, reason
		cur.GradeState = coordinator.ShadowGradeGraded
	})
}

// gradeShadowSide runs the gates for one side. A run that produced no diff
// failed its task, which is a real result; a diff we could not set up to test
// is ungraded, which is not.
func gradeShadowSide(ctx context.Context, dir string, r *coordinator.ShadowRunResult, repoURL, policyTOML string, run gateRunner) coordinator.ShadowSideGrade {
	if g, done := prepareShadowSide(ctx, dir, r, repoURL); done {
		return g
	}
	modules := changedAILModules(dir, r.ChangedFiles)
	if len(modules) == 0 {
		return coordinator.ShadowSideGrade{Ungraded: "no_ail_changes"}
	}
	return runGatePlan(ctx, dir, policyTOML, shadowGatePlan(dir, modules), run)
}

// prepareShadowSide checks out the run's base commit and applies its diff.
// done=true returns the side's grade without running gates.
func prepareShadowSide(ctx context.Context, dir string, r *coordinator.ShadowRunResult, repoURL string) (coordinator.ShadowSideGrade, bool) {
	switch {
	case r != nil && strings.Contains(r.ErrorMsg, shadowKeyRefusal):
		// Refused before the agent started (no shadow key on this plane): the
		// harness never ran, so this is not a harness failing.
		return coordinator.ShadowSideGrade{Ungraded: "shadow_key_missing"}, true
	case r != nil && r.Status == "failed" && r.NumTurns == 0 && strings.TrimSpace(r.Diff) == "":
		// Failed before the agent's first turn: the image, the lane check or
		// the dispatch, not the code the harness writes. Kept visible in the
		// ungraded tally, never scored as the harness losing (measured
		// 2026-10-19: both motoko runs of the first dev smoke died on an image
		// defect and would have read as motoko writing worse AILANG).
		return coordinator.ShadowSideGrade{Ungraded: "did_not_start"}, true
	case r == nil || strings.TrimSpace(r.Diff) == "":
		return coordinator.ShadowSideGrade{Pass: false}, true
	case ctx.Err() != nil:
		return coordinator.ShadowSideGrade{Ungraded: "timeout"}, true
	case r.BaseCommit == "":
		return coordinator.ShadowSideGrade{Ungraded: "no_base_commit"}, true
	}
	if err := checkoutCommit(ctx, dir, repoURL, r.BaseCommit); err != nil {
		return coordinator.ShadowSideGrade{Ungraded: "clone_failed"}, true
	}
	if err := applyDiff(ctx, dir, r.Diff); err != nil {
		return coordinator.ShadowSideGrade{Ungraded: "apply_failed"}, true
	}
	return coordinator.ShadowSideGrade{}, false
}

// runGatePlan runs every group's gates with that group's package directory as
// the policy sandbox: the agent's own view (the lane scopes the sandbox to the
// package), and the only root a module inside a package resolves from.
func runGatePlan(ctx context.Context, dir, policyTOML string, plan []gateGroup, run gateRunner) coordinator.ShadowSideGrade {
	g := coordinator.ShadowSideGrade{Pass: true}
	for _, grp := range plan {
		sandbox := filepath.Join(dir, filepath.FromSlash(grp.root))
		policyPath, err := executor.MaterializeAgentPolicy(policyTOML, sandbox)
		if err != nil || policyPath == "" {
			return coordinator.ShadowSideGrade{Ungraded: "no_policy", Gates: g.Gates}
		}
		for _, req := range grp.reqs {
			if ctx.Err() != nil {
				return coordinator.ShadowSideGrade{Ungraded: "timeout", Gates: g.Gates}
			}
			resp := run(sandbox, policyPath, req)
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return coordinator.ShadowSideGrade{Ungraded: "timeout", Gates: g.Gates}
			}
			gate := coordinator.ShadowGateResult{Op: req.Op, Path: path.Join(grp.root, req.Path), Package: req.Flags["package"],
				OK: gatePassed(req, resp), ExitCode: resp.ExitCode,
				Refused: resp.Refused, Output: tailString(resp.Stderr+resp.Stdout, 2000)}
			if gate.Package != "" {
				gate.Package = grp.root
			}
			g.Gates = append(g.Gates, gate)
			if resp.Refused != "" {
				// The policy refused to run the gate: that says nothing about
				// the harness that wrote the code.
				return coordinator.ShadowSideGrade{Ungraded: "gate_refused", Gates: g.Gates}
			}
			g.Pass = g.Pass && gate.OK
		}
	}
	return g
}

// gatePassed reads a gate's response. A `test` of a module with no tests exits
// 1 ("No tests found"), which is not a failing test: with --json the counts
// say which it was, and zero tests run with zero failed is vacuous, so it passes.
func gatePassed(req policytool.Request, resp policytool.Response) bool {
	if resp.Refused != "" {
		return false
	}
	if resp.ExitCode == 0 && resp.OK {
		return true
	}
	if req.Op != "test" {
		return false
	}
	var counts struct {
		Total  *int `json:"total_tests"`
		Failed int  `json:"failed_tests"`
	}
	if json.Unmarshal([]byte(strings.TrimSpace(resp.Stdout)), &counts) != nil || counts.Total == nil {
		return false
	}
	return *counts.Total == 0 && counts.Failed == 0
}

// gateGroup is the gates for the modules under one sandbox root (a package
// directory, or the clone root for modules in no package), relative to it.
type gateGroup struct {
	root string
	reqs []policytool.Request
}

// shadowGatePlan: per module, `check` it and `test` it (its inline tests);
// per package that ships *_test.ail files, `test --package .` as well.
// Measured 2026-10-19: `test --package` runs only *_test.ail files and exits 0
// on a package with none, so on its own it would pass a broken inline test.
func shadowGatePlan(dir string, modules []string) []gateGroup {
	byRoot := map[string][]string{}
	for _, m := range modules {
		root, ok := packageRoot(dir, m)
		if !ok {
			root = "."
		}
		rel := m
		if root != "." {
			rel = strings.TrimPrefix(m, root+"/")
		}
		byRoot[root] = append(byRoot[root], rel)
	}
	roots := make([]string, 0, len(byRoot))
	for r := range byRoot {
		roots = append(roots, r)
	}
	sort.Strings(roots)
	var plan []gateGroup
	for _, r := range roots {
		g := gateGroup{root: r}
		for _, m := range byRoot[r] {
			g.reqs = append(g.reqs, policytool.Request{Op: "check", Path: m}, testRequest(m, ""))
		}
		if r != "." && hasTestFiles(filepath.Join(dir, filepath.FromSlash(r))) {
			g.reqs = append(g.reqs, testRequest("", "."))
		}
		plan = append(plan, g)
	}
	return plan
}

// testRequest is a `test` gate with --json, which gatePassed reads.
func testRequest(path, pkg string) policytool.Request {
	flags := map[string]string{"json": ""}
	if pkg != "" {
		flags["package"] = pkg
	}
	return policytool.Request{Op: "test", Path: path, Flags: flags}
}

func hasTestFiles(pkgDir string) bool {
	found := false
	_ = filepath.Walk(pkgDir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(p, "_test.ail") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// packageRoot walks up from a module to the nearest ailang.toml inside dir.
func packageRoot(dir, rel string) (string, bool) {
	for d := filepath.Dir(rel); ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(dir, d, "ailang.toml")); err == nil {
			return filepath.ToSlash(d), true
		}
		if d == "." || d == string(filepath.Separator) {
			return "", false
		}
	}
}

// changedAILModules keeps the changed .ail files that exist after the diff
// (a deleted module has nothing to check).
func changedAILModules(dir string, changed []string) []string {
	var out []string
	for _, f := range changed {
		if !strings.HasSuffix(f, ".ail") {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(f))); err == nil {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}

// checkoutCommit fetches exactly one commit: the run's recorded base, not the
// branch tip, so the grade sees the code the agent saw.
func checkoutCommit(ctx context.Context, dir, repoURL, sha string) error {
	for _, args := range [][]string{
		{"init", "-q", dir},
		{"-C", dir, "remote", "add", "origin", repoURL},
		{"-C", dir, "fetch", "-q", "--depth", "1", "origin", sha},
		{"-C", dir, "checkout", "-q", "FETCH_HEAD"},
	} {
		if out, err := gitexec.CommandContext(ctx, args...).CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "execute-job: shadow grade: git %v: %v: %s\n", args, err, out)
			return err
		}
	}
	return nil
}

// applyDiff applies a patch kept outside the sandbox. A truncated diff (the job
// caps it) fails here and grades as apply_failed.
func applyDiff(ctx context.Context, dir, diff string) error {
	f, err := os.CreateTemp("", "shadow-grade-*.patch")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	// The job's evidence trims git's output (collectGitEvidence), so a stored
	// patch has lost its final newline, and git apply calls that a corrupt
	// patch. Measured 2026-10-19: the first real primary diff graded
	// apply_failed for exactly this.
	if !strings.HasSuffix(diff, "\n") {
		diff += "\n"
	}
	if _, err := f.WriteString(diff); err != nil {
		f.Close()
		return err
	}
	f.Close()
	out, err := gitexec.CommandContext(ctx, "-C", dir, "apply", "--whitespace=nowarn", f.Name()).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: shadow grade: git apply: %v: %s\n", err, out)
	}
	return err
}

func tailString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

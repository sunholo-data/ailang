package main

import (
	"context"
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
	return gradeShadowComparison(ctx, fsstore.NewCoordinatorStore(client), parent, config.RepoURL(),
		config.AgentPolicyTOML(), filepath.Join(cloudWorkspaceRoot, parent+"-grade"), policyToolGateRunner)
}

// gradeShadowComparison grades both sides and writes the result back. The write
// only lands while the comparison is still "dispatched": if the coordinator
// already gave up on this job (grade_timeout), the late grade is dropped rather
// than overwriting a settled verdict.
func gradeShadowComparison(ctx context.Context, store coordinator.ShadowComparisonStore, parent, repoURL, policyTOML, root string, run gateRunner) error {
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
	// The grade must land even when the job's bound is what ended it: a
	// timeout is a result. So the write gets its own short deadline.
	wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	wrote, err := store.UpdateShadowComparison(wctx, parent, func(cur *coordinator.ShadowComparison) bool {
		if cur.GradeState != coordinator.ShadowGradeDispatched {
			return false
		}
		cur.Mechanical, cur.Verdict, cur.UngradedReason = &m, verdict, reason
		cur.GradeState = coordinator.ShadowGradeGraded
		return true
	})
	if err != nil {
		return fmt.Errorf("shadow grade: writing %s: %w", parent, err)
	}
	fmt.Printf("execute-job: shadow grade %s: verdict=%s %s (written=%v)\n", parent, verdict, reason, wrote)
	return nil
}

// gradeShadowSide runs the gates for one side. A run that produced no diff
// failed its task, which is a real result; a diff we could not set up to test
// is ungraded, which is not.
func gradeShadowSide(ctx context.Context, dir string, r *coordinator.ShadowRunResult, repoURL, policyTOML string, run gateRunner) coordinator.ShadowSideGrade {
	if r == nil || strings.TrimSpace(r.Diff) == "" {
		return coordinator.ShadowSideGrade{Pass: false}
	}
	if ctx.Err() != nil {
		return coordinator.ShadowSideGrade{Ungraded: "timeout"}
	}
	if r.BaseCommit == "" {
		return coordinator.ShadowSideGrade{Ungraded: "no_base_commit"}
	}
	if err := checkoutCommit(ctx, dir, repoURL, r.BaseCommit); err != nil {
		return coordinator.ShadowSideGrade{Ungraded: "clone_failed"}
	}
	if err := applyDiff(ctx, dir, r.Diff); err != nil {
		return coordinator.ShadowSideGrade{Ungraded: "apply_failed"}
	}
	modules := changedAILModules(dir, r.ChangedFiles)
	if len(modules) == 0 {
		return coordinator.ShadowSideGrade{Ungraded: "no_ail_changes"}
	}
	g := coordinator.ShadowSideGrade{Pass: true}
	for _, grp := range shadowGatePlan(dir, modules) {
		// Each group runs with its package directory as the policy sandbox:
		// that is the agent's own view (the lane scopes the sandbox to the
		// package), and a module inside a package only resolves from there.
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
				OK: resp.OK && resp.Refused == "" && resp.ExitCode == 0, ExitCode: resp.ExitCode,
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
			g.reqs = append(g.reqs, policytool.Request{Op: "check", Path: m}, policytool.Request{Op: "test", Path: m})
		}
		if r != "." && hasTestFiles(filepath.Join(dir, filepath.FromSlash(r))) {
			g.reqs = append(g.reqs, policytool.Request{Op: "test", Flags: map[string]string{"package": "."}})
		}
		plan = append(plan, g)
	}
	return plan
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

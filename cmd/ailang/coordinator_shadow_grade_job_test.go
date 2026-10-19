package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/policytool"
	"github.com/sunholo-data/ailang/internal/testutil"
)

const gradeTestPolicy = "fs_sandbox = \"${WORKSPACE}\"\n"

// gradeFixture is a bare remote holding a package (pkg/ with ailang.toml and
// m.ail) and a loose module, plus helpers that turn edits into the diff a run
// would have banked.
type gradeFixture struct {
	remote, url, base string
	store             *coordinator.SQLiteStore
	ops               []string
	ctx               context.Context // the grade's context, which HANG waits out
}

func newGradeFixture(t *testing.T) *gradeFixture {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv(config.EnvRelaxModules, "") // restored after; the grade sets it
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	// The materialised policy is read-only by design; let TempDir remove it.
	t.Cleanup(func() {
		pol := filepath.Join(home, ".ailang", "agent-policy")
		_ = os.Chmod(pol, 0o755)
		_ = os.Chmod(filepath.Join(pol, "agent-policy.toml"), 0o644)
	})
	seed := t.TempDir()
	runGit(t, seed, "init", "-b", "dev")
	writeGradeFile(t, seed, "pkg/ailang.toml", "[package]\nname = \"p\"\n")
	writeGradeFile(t, seed, "pkg/m.ail", "module pkg/m\n")
	writeGradeFile(t, seed, "loose.ail", "module loose\n")
	runGit(t, seed, "add", ".")
	runGit(t, seed, "commit", "-m", "seed")
	f := &gradeFixture{base: runGit(t, seed, "rev-parse", "HEAD")}
	f.remote = filepath.Join(t.TempDir(), "remote.git")
	runGit(t, seed, "clone", "--bare", seed, f.remote)
	// Fetching a bare SHA needs this on a file:// remote (GitHub allows it).
	runGit(t, f.remote, "config", "uploadpack.allowAnySHA1InWant", "true")
	path := filepath.ToSlash(f.remote)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	f.url = "file://" + path
	store, err := coordinator.NewSQLiteStore(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	f.store = store
	return f
}

func writeGradeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// diff edits a fresh clone and returns the run result a completion would carry.
func (f *gradeFixture) diff(t *testing.T, id string, edits map[string]string) coordinator.ShadowRunResult {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "clone", "-q", f.remote, ".")
	var files []string
	for rel, content := range edits {
		writeGradeFile(t, dir, rel, content)
		files = append(files, rel)
	}
	runGit(t, dir, "add", ".")
	return coordinator.ShadowRunResult{TaskID: id, Status: "completed", BaseCommit: f.base,
		// runGit trims, exactly as the job's evidence collector does: the
		// stored patch has no final newline.
		ChangedFiles: files, Diff: runGit(t, dir, "diff", "--cached", f.base)}
}

// seed records both sides and puts the comparison where the coordinator's
// sweep leaves it: dispatched.
func (f *gradeFixture) seed(t *testing.T, primary, shadow coordinator.ShadowRunResult) {
	t.Helper()
	ctx := context.Background()
	for side, r := range map[string]coordinator.ShadowRunResult{coordinator.ShadowSidePrimary: primary, coordinator.ShadowSideShadow: shadow} {
		if _, err := f.store.RecordShadowRun(ctx, "task-p", side, r); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.store.UpdateShadowComparison(ctx, "task-p", func(c *coordinator.ShadowComparison) bool {
		c.GradeState = coordinator.ShadowGradeDispatched
		return true
	}); err != nil {
		t.Fatal(err)
	}
}

// runner stands in for the policy tool: a module containing BAD fails check, a
// package containing BADTEST fails test, HANG outlives the job's deadline.
func (f *gradeFixture) runner(dir, _ string, req policytool.Request) policytool.Response {
	f.ops = append(f.ops, req.Op)
	target := req.Path
	if p := req.Flags["package"]; p != "" {
		target = p
	}
	var body strings.Builder
	_ = filepath.Walk(filepath.Join(dir, filepath.FromSlash(target)), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(p, ".ail") {
			b, _ := os.ReadFile(p)
			body.Write(b)
		}
		return nil
	})
	switch {
	case strings.Contains(body.String(), "HANG"):
		<-f.ctx.Done() // a gate that outlives the job's bound
		return policytool.Response{OK: true}
	case req.Op == "check" && strings.Contains(body.String(), "BAD") && !strings.Contains(body.String(), "BADTEST"):
		return policytool.Response{OK: true, ExitCode: 1, Stderr: "type error"}
	case req.Op == "test" && strings.Contains(body.String(), "BADTEST"):
		return policytool.Response{OK: true, ExitCode: 1, Stderr: "1 failed"}
	}
	return policytool.Response{OK: true}
}

func (f *gradeFixture) grade(t *testing.T, timeout time.Duration) *coordinator.ShadowComparison {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	f.ctx = ctx
	if err := gradeShadowComparison(ctx, f.store, "task-p", f.url, gradeTestPolicy, t.TempDir(), f.runner); err != nil {
		t.Fatal(err)
	}
	c, err := f.store.GetShadowComparison(context.Background(), "task-p")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestShadowGradeVerdicts(t *testing.T) {
	good := map[string]string{"pkg/m.ail": "module pkg/m\n-- fixed\n"}
	cases := []struct {
		name       string
		primary    map[string]string
		shadow     map[string]string // nil = the shadow run produced no diff
		want       coordinator.ShadowVerdict
		wantReason string
	}{
		{"shadow fails check", good, map[string]string{"pkg/m.ail": "module pkg/m\nBAD\n"}, coordinator.ShadowVerdictPrimaryBetter, ""},
		{"primary fails package test", map[string]string{"pkg/m.ail": "module pkg/m\nBADTEST\n"}, good, coordinator.ShadowVerdictShadowBetter, ""},
		{"both pass", good, map[string]string{"loose.ail": "module loose\n-- ok\n"}, coordinator.ShadowVerdictEquivalent, ""},
		{"shadow produced nothing", good, nil, coordinator.ShadowVerdictPrimaryBetter, ""},
		{"no AILANG changed", good, map[string]string{"README.md": "x\n"}, coordinator.ShadowVerdictUngraded, "shadow: no_ail_changes"},
		{"shadow refused for want of its key", good, map[string]string{}, coordinator.ShadowVerdictUngraded, "shadow: shadow_key_missing"},
		{"shadow never started", good, map[string]string{"": ""}, coordinator.ShadowVerdictUngraded, "shadow: did_not_start"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newGradeFixture(t)
			// An agent that worked (turns) and produced nothing: a real failure.
			shadow := coordinator.ShadowRunResult{TaskID: "task-p-shadow", Status: "failed", BaseCommit: f.base, NumTurns: 12}
			if _, never := tc.shadow[""]; !never && len(tc.shadow) > 0 {
				shadow = f.diff(t, "task-p-shadow", tc.shadow)
			}
			if tc.shadow != nil && len(tc.shadow) == 0 {
				shadow.ErrorMsg = "executor failed: " + shadowKeyRefusal + ": OPENROUTER_API_KEY_SHADOW is not set"
			}
			if _, never := tc.shadow[""]; never {
				shadow = coordinator.ShadowRunResult{TaskID: "task-p-shadow", Status: "failed", BaseCommit: f.base,
					ErrorMsg: "executor failed: motoko execution failed: could not resolve motoko profile"}
			}
			f.seed(t, f.diff(t, "task-p", tc.primary), shadow)
			c := f.grade(t, time.Minute)
			if c.Verdict != tc.want || c.UngradedReason != tc.wantReason || c.GradeState != coordinator.ShadowGradeGraded {
				t.Fatalf("verdict=%q reason=%q state=%q; want %q %q graded\nmechanical=%+v",
					c.Verdict, c.UngradedReason, c.GradeState, tc.want, tc.wantReason, c.Mechanical)
			}
			for _, op := range f.ops {
				if op != "check" && op != "test" {
					t.Fatalf("grade ran op %q; only AILANG check/test may run", op)
				}
			}
		})
	}
}

// Gates run with the module's package directory as the sandbox, paths relative
// to it; every module gets check + its inline tests, and only a package that
// ships *_test.ail files gets `test --package .` (which passes vacuously
// otherwise).
func TestShadowGatePlan(t *testing.T) {
	dir := t.TempDir()
	writeGradeFile(t, dir, "pkg/ailang.toml", "")
	writeGradeFile(t, dir, "pkg/sub/a.ail", "")
	writeGradeFile(t, dir, "pkg/b.ail", "")
	writeGradeFile(t, dir, "lib/ailang.toml", "")
	writeGradeFile(t, dir, "lib/c.ail", "")
	writeGradeFile(t, dir, "lib/c_test.ail", "")
	writeGradeFile(t, dir, "loose.ail", "")
	plan := shadowGatePlan(dir, []string{"lib/c.ail", "loose.ail", "pkg/b.ail", "pkg/sub/a.ail"})
	got := map[string][]string{}
	for _, g := range plan {
		for _, r := range g.reqs {
			got[g.root] = append(got[g.root], r.Op+":"+r.Path+r.Flags["package"])
		}
	}
	want := map[string][]string{
		".":   {"check:loose.ail", "test:loose.ail"},
		"lib": {"check:c.ail", "test:c.ail", "test:."},
		"pkg": {"check:b.ail", "test:b.ail", "check:sub/a.ail", "test:sub/a.ail"},
	}
	if len(got) != len(want) {
		t.Fatalf("groups = %v", got)
	}
	for root, reqs := range want {
		if strings.Join(got[root], ",") != strings.Join(reqs, ",") {
			t.Errorf("%s: got %v, want %v", root, got[root], reqs)
		}
	}
}

func TestShadowGradeUnappliableDiffIsUngraded(t *testing.T) {
	f := newGradeFixture(t)
	bad := f.diff(t, "task-p-shadow", map[string]string{"pkg/m.ail": "x\n"})
	bad.Diff = bad.Diff[:len(bad.Diff)/2] // truncated, as the job's cap would leave it
	f.seed(t, f.diff(t, "task-p", map[string]string{"pkg/m.ail": "ok\n"}), bad)
	if c := f.grade(t, time.Minute); c.Verdict != coordinator.ShadowVerdictUngraded || c.UngradedReason != "shadow: apply_failed" {
		t.Fatalf("verdict=%q reason=%q", c.Verdict, c.UngradedReason)
	}
}

func TestShadowGradeHangIsUngradedTimeout(t *testing.T) {
	f := newGradeFixture(t)
	f.seed(t, f.diff(t, "task-p", map[string]string{"pkg/m.ail": "HANG\n"}), f.diff(t, "task-p-shadow", map[string]string{"loose.ail": "ok\n"}))
	if c := f.grade(t, 5*time.Second); c.Verdict != coordinator.ShadowVerdictUngraded || c.UngradedReason != "primary: timeout" {
		t.Fatalf("verdict=%q reason=%q", c.Verdict, c.UngradedReason)
	}
}

// A grade that arrives after the coordinator gave up must not overwrite the
// settled verdict.
func TestShadowGradeLateResultDropped(t *testing.T) {
	f := newGradeFixture(t)
	f.seed(t, f.diff(t, "task-p", map[string]string{"pkg/m.ail": "ok\n"}), f.diff(t, "task-p-shadow", map[string]string{"pkg/m.ail": "BAD\n"}))
	if _, err := f.store.UpdateShadowComparison(context.Background(), "task-p", func(c *coordinator.ShadowComparison) bool {
		c.GradeState, c.Verdict, c.UngradedReason = coordinator.ShadowGradeGraded, coordinator.ShadowVerdictUngraded, "grade_timeout"
		return true
	}); err != nil {
		t.Fatal(err)
	}
	if c := f.grade(t, time.Minute); c.UngradedReason != "grade_timeout" || c.Mechanical != nil {
		t.Fatalf("late grade overwrote the settled verdict: %+v", c)
	}
}

// A gate the policy refuses says nothing about the harness: ungraded, not failed.
func TestShadowGradeRefusedGateIsUngraded(t *testing.T) {
	f := newGradeFixture(t)
	f.seed(t, f.diff(t, "task-p", map[string]string{"pkg/m.ail": "ok\n"}), f.diff(t, "task-p-shadow", map[string]string{"loose.ail": "ok\n"}))
	ctx := context.Background()
	refuse := func(string, string, policytool.Request) policytool.Response {
		return policytool.Response{Refused: "outside fs_sandbox"}
	}
	if err := gradeShadowComparison(ctx, f.store, "task-p", f.url, gradeTestPolicy, t.TempDir(), refuse); err != nil {
		t.Fatal(err)
	}
	c, _ := f.store.GetShadowComparison(ctx, "task-p")
	if c.Verdict != coordinator.ShadowVerdictUngraded || c.UngradedReason != "primary: gate_refused" {
		t.Fatalf("verdict=%q reason=%q", c.Verdict, c.UngradedReason)
	}
}

// An unknown job mode is refused before anything runs.
func TestExecuteJobUnknownModeRefused(t *testing.T) {
	t.Setenv(config.EnvJobMode, "grade_everything")
	if err := coordinatorExecuteJob(nil); err == nil || !strings.Contains(err.Error(), "grade_everything") {
		t.Fatalf("err = %v, want the unknown mode named", err)
	}
}

// The grade end to end through the REAL policy tool and a real ailang binary,
// under the real package-lane policy shape. Opt-in, because the test binary is
// not ailang:
//
//	AILANG_SHADOW_GRADE_LIVE_BIN=$(which ailang) go test ./cmd/ailang/ -run ShadowGradeLive -count=1
func TestShadowGradeLive(t *testing.T) {
	bin := os.Getenv("AILANG_SHADOW_GRADE_LIVE_BIN") //nolint:forbidigo // opt-in live test knob, never read by the product
	if bin == "" {
		t.Skip("set AILANG_SHADOW_GRADE_LIVE_BIN to an ailang binary")
	}
	const policy = `allowed_caps = ["IO", "FS", "Process"]
fs_sandbox = "${WORKSPACE}"
process_allow = ["git:status"]
security_mode = "restricted"
timeout_ms = 120000
entry = "main"
`
	good := "module m\n\nexport pure func double(x: int) -> int\n  tests [(2, 4), (3, 6)]\n{\n  x * 2\n}\n"
	for _, tc := range []struct {
		name   string
		shadow string
		want   coordinator.ShadowVerdict
	}{
		{"broken inline test", strings.Replace(good, "(3, 6)", "(3, 7)", 1), coordinator.ShadowVerdictPrimaryBetter},
		{"type error", strings.Replace(good, "x * 2", "x * \"2\"", 1), coordinator.ShadowVerdictPrimaryBetter},
		{"both good", strings.Replace(good, "(2, 4)", "(1, 2)", 1), coordinator.ShadowVerdictEquivalent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newGradeFixture(t)
			manifest := "[package]\nname = \"sunholo/gradesmoke\"\nversion = \"0.1.0\"\nedition = \"1\"\n"
			primary := f.diff(t, "task-p", map[string]string{"pkg/ailang.toml": manifest, "pkg/m.ail": good})
			shadow := f.diff(t, "task-p-shadow", map[string]string{"pkg/ailang.toml": manifest, "pkg/m.ail": tc.shadow})
			f.seed(t, primary, shadow)
			ctx := context.Background()
			if err := gradeShadowComparison(ctx, f.store, "task-p", f.url, policy, t.TempDir(), policyToolGateRunnerWith(bin)); err != nil {
				t.Fatal(err)
			}
			c, _ := f.store.GetShadowComparison(ctx, "task-p")
			if c.Verdict != tc.want {
				t.Fatalf("verdict = %q (%s), want %q\nprimary gates: %+v\nshadow gates: %+v", c.Verdict, c.UngradedReason, tc.want,
					c.Mechanical.Primary.Gates, c.Mechanical.Shadow.Gates)
			}
		})
	}
	// A lane-report program: a loose file whose module name does not mirror its
	// path, with no tests. Both conditions are fine in the lane, so it passes.
	t.Run("loose lane program, no tests", func(t *testing.T) {
		f := newGradeFixture(t)
		prog := "module scan\n\nexport func main() -> () ! {IO} {\n  println(\"ok\")\n}\n"
		f.seed(t, f.diff(t, "task-p", map[string]string{"lane-reports/x/scan.ail": prog}),
			f.diff(t, "task-p-shadow", map[string]string{"lane-reports/x/scan.ail": prog}))
		ctx := context.Background()
		if err := gradeShadowComparison(ctx, f.store, "task-p", f.url, policy, t.TempDir(), policyToolGateRunnerWith(bin)); err != nil {
			t.Fatal(err)
		}
		c, _ := f.store.GetShadowComparison(ctx, "task-p")
		if c.Verdict != coordinator.ShadowVerdictEquivalent {
			t.Fatalf("verdict = %q (%s)\nprimary gates: %+v", c.Verdict, c.UngradedReason, c.Mechanical.Primary.Gates)
		}
	})
}

// gatePassed: a test run that found no tests is vacuous, not failing; any
// other non-zero exit fails; a refusal never passes.
func TestGatePassed(t *testing.T) {
	test := testRequest("m.ail", "")
	cases := []struct {
		name string
		req  policytool.Request
		resp policytool.Response
		want bool
	}{
		{"clean exit", test, policytool.Response{OK: true}, true},
		{"no tests found", test, policytool.Response{ExitCode: 1, Stdout: `{"total_tests":0,"failed_tests":0}`}, true},
		{"a failing test", test, policytool.Response{ExitCode: 1, Stdout: `{"total_tests":2,"failed_tests":1}`}, false},
		{"unparseable test output", test, policytool.Response{ExitCode: 1, Stdout: "boom"}, false},
		{"check failure is never vacuous", policytool.Request{Op: "check"}, policytool.Response{ExitCode: 1, Stdout: `{"total_tests":0}`}, false},
		{"refused", test, policytool.Response{OK: true, Refused: "outside"}, false},
	}
	for _, c := range cases {
		if got := gatePassed(c.req, c.resp); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	if _, ok := test.Flags["json"]; !ok {
		t.Error("test gates must ask for --json, which gatePassed reads")
	}
}

// Grading runs with relaxed module names, as the lane does.
func TestShadowGradeRelaxesModuleNames(t *testing.T) {
	f := newGradeFixture(t)
	f.seed(t, f.diff(t, "task-p", map[string]string{"pkg/m.ail": "ok\n"}), f.diff(t, "task-p-shadow", map[string]string{"pkg/m.ail": "ok\n"}))
	var during string
	run := func(string, string, policytool.Request) policytool.Response {
		during = os.Getenv(config.EnvRelaxModules)
		return policytool.Response{OK: true}
	}
	if err := gradeShadowComparison(context.Background(), f.store, "task-p", f.url, gradeTestPolicy, t.TempDir(), run); err != nil {
		t.Fatal(err)
	}
	if during != "1" {
		t.Fatalf("%s = %q during the gates, want 1", config.EnvRelaxModules, during)
	}
}

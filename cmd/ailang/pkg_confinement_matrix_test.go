package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/policytool"
)

// rowKind is what a confined row must do besides staying I/O-free.
type rowKind int

const (
	mustRefuse rowKind = iota // stderr carries the canonical refusal, exit != 0
	mustSucceed
	anyExit // only the I/O invariants are asserted
)

type matrixRow struct {
	args   []string
	kind   rowKind
	ledger []string // sandbox paths the op may change (its declared writes)
}

const confinedText = "confined by AILANG_AGENT_POLICY"

// pkgVerbMatrix is one or more confined rows per pkg verb. It is keyed by
// verb, and TestPkgConfinementMatrix iterates pkgSubcommands(): a verb added
// without a row here FAILS, so no verb ships without being run confined.
var pkgVerbMatrix = map[string][]matrixRow{
	"docs":           {{args: []string{"pkg", "docs", "test/lib"}, kind: mustRefuse}},
	"add":            {{args: []string{"pkg", "add", "--git", "file:///nonexistent", "--tag", "v1"}, kind: mustRefuse}},
	"lock":           {{args: []string{"pkg", "lock"}, kind: mustRefuse}, {args: []string{"pkg", "lock", "--check"}, kind: mustRefuse}},
	"tree":           {{args: []string{"pkg", "tree"}, kind: mustSucceed}},
	"install":        {{args: []string{"pkg", "install", "test/lib@0.1.0"}, kind: mustRefuse}, {args: []string{"pkg", "install", "--path", "."}, kind: mustRefuse}},
	"bin":            {{args: []string{"pkg", "bin", "uninstall", "app"}, kind: mustRefuse}, {args: []string{"pkg", "bin", "list"}, kind: mustSucceed}},
	"search":         {{args: []string{"pkg", "search", "lib"}, kind: mustRefuse}},
	"publish":        {{args: []string{"pkg", "publish"}, kind: mustRefuse}},
	"unpublish":      {{args: []string{"pkg", "unpublish", "test/lib@0.1.0"}, kind: mustRefuse}},
	"quality":        {{args: []string{"pkg", "quality", "."}, kind: anyExit}},
	"info":           {{args: []string{"pkg", "info", "test/lib"}, kind: mustRefuse}},
	"versions":       {{args: []string{"pkg", "versions", "test/lib"}, kind: mustRefuse}},
	"stats":          {{args: []string{"pkg", "stats"}, kind: mustRefuse}},
	"provenance":     {{args: []string{"pkg", "provenance", "test/lib@0.1.0"}, kind: mustRefuse}},
	"history":        {{args: []string{"pkg", "history", "test/lib@0.1.0"}, kind: mustRefuse}},
	"notify-upgrade": {{args: []string{"pkg", "notify-upgrade", "test/lib@0.2.0"}, kind: mustRefuse}},
	"affected-by":    {{args: []string{"pkg", "affected-by", "test/lib"}, kind: anyExit}},
	"cascade":        {{args: []string{"pkg", "cascade", "status", "test/lib@0.1.0"}, kind: mustRefuse}},
	"key":            {{args: []string{"pkg", "key", "list"}, kind: mustRefuse}},
}

// loaderRows reach the package loader rather than a pkg verb. The git
// dependency is locked but not provisioned in this HOME, so D4 makes check
// fail loudly — what matters here is that nothing is fetched or written.
var loaderRows = []matrixRow{
	{args: []string{"lock"}, kind: mustRefuse}, // bare legacy spelling
	{args: []string{"check", "main.ail"}, kind: anyExit},
	{args: []string{"ai-check", "main.ail"}, kind: anyExit},
	{args: []string{"test", "main.ail"}, kind: anyExit},
	{args: []string{"iface", "test/app/main"}, kind: anyExit},
	{args: []string{"run", "--entry", "main", "main.ail"}, kind: anyExit},
}

// TestPkgConfinementMatrix is #1720 AC1: every pkg verb, plus the loader
// entry points, run under AILANG_AGENT_POLICY with a git dependency. Each row
// must make zero recorder requests, zero network git invocations, and leave
// HOME, an outside canary and the sandbox byte-identical.
func TestPkgConfinementMatrix(t *testing.T) {
	f := newConfinementFixture(t)
	var rows []matrixRow
	for _, sub := range pkgSubcommands() {
		vr, ok := pkgVerbMatrix[sub.Name]
		if !ok {
			t.Errorf("pkg verb %q has no confinement matrix row — add one to pkgVerbMatrix (#1720)", sub.Name)
			continue
		}
		rows = append(rows, vr...)
	}
	rows = append(rows, loaderRows...)
	for _, row := range rows {
		row := row
		t.Run(strings.Join(row.args, "_"), func(t *testing.T) {
			f.assertConfinedRow(t, f.template, nil, row)
		})
	}
}

func (f *confinementFixture) assertConfinedRow(t *testing.T, template string, extraEnv map[string]string, row matrixRow) *confinementRun {
	t.Helper()
	tplSnap := treeSnapshot(t, template)
	r := f.run(t, true, template, "", extraEnv, row.args...)
	if len(r.requests) != 0 {
		t.Errorf("recorder saw %d request(s): %v", len(r.requests), r.requests)
	}
	if len(r.netGit) != 0 {
		t.Errorf("network git invoked: %v", r.netGit)
	}
	if s := treeSnapshot(t, r.home); len(s) != 1 {
		t.Errorf("HOME changed (was empty): %v", s)
	}
	if s := treeSnapshot(t, r.canary); len(s) != 2 || readFile(t, filepath.Join(r.canary, "canary.txt")) != "outside the sandbox\n" {
		t.Errorf("outside canary changed: %v", s)
	}
	if d := snapshotDiff(tplSnap, treeSnapshot(t, r.sandbox), row.ledger); len(d) != 0 {
		t.Errorf("sandbox changed outside the ledger %v: %v", row.ledger, d)
	}
	switch row.kind {
	case mustRefuse:
		if r.exit == 0 || !strings.Contains(r.stderr, confinedText) {
			t.Errorf("want a confinement refusal, got exit %d\nstdout: %s\nstderr: %s", r.exit, r.stdout, r.stderr)
		}
	case mustSucceed:
		if r.exit != 0 {
			t.Errorf("want exit 0, got %d\nstdout: %s\nstderr: %s", r.exit, r.stdout, r.stderr)
		}
	}
	return r
}

// AC2a: the same fixture unconfined MUST light the instruments, or the
// confined matrix proves nothing ("instrument blind").
func TestPkgConfinementMatrix_UnconfinedControl(t *testing.T) {
	f := newConfinementFixture(t)
	noLock := filepath.Join(t.TempDir(), "nolock")
	copyTree(t, f.template, noLock)
	if err := os.Remove(filepath.Join(noLock, pkg.LockFileName)); err != nil {
		t.Fatal(err)
	}
	withBin := filepath.Join(t.TempDir(), "withbin")
	copyTree(t, noLock, withBin)
	mustWrite(t, filepath.Join(withBin, "ailang.toml"), readFile(t, filepath.Join(noLock, "ailang.toml"))+"[bin]\napp = { module = \"test/app/main\", entry = \"main\" }\n", 0o644)

	for _, c := range []struct {
		name     string
		template string
		args     []string
	}{
		{"lock", noLock, []string{"lock"}},
		{"lock with prior lock", f.template, []string{"lock"}},
		{"lock --check", f.template, []string{"lock", "--check"}},
		{"add --git", noLock, []string{"add", "--git", "file://" + f.upstream, "--tag", "v0.1.0"}},
		{"install --path", withBin, []string{"install", "--path", "."}},
		{"pkg info", noLock, []string{"pkg", "info", "test/lib"}},
		{"pkg history", noLock, []string{"pkg", "history", "test/lib@0.1.0"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := f.run(t, false, c.template, "", nil, c.args...)
			if len(r.requests) == 0 && len(r.netGit) == 0 {
				t.Errorf("instrument blind: unconfined %v made no request and no network git call\nstderr: %s", c.args, r.stderr)
			}
			if len(r.netGit) > 0 && len(treeSnapshot(t, r.home)) <= 1 {
				t.Errorf("instrument blind: unconfined %v cloned but HOME is empty", c.args)
			}
		})
	}
}

// AC2c + AC6: an operator-provisioned git package under AILANG_PACKAGE_ROOT
// (copied from the operator's clone; root read-only) gives confined check
// exit 0 and confined lock exit 0 with only ailang.lock changed — and HOME,
// including ~/.ailang/cache/git, is never touched.
func TestPkgConfinementMatrix_ProvisionedControl(t *testing.T) {
	f := newConfinementFixture(t)
	clones, err := filepath.Glob(filepath.Join(f.opHome, ".ailang", "cache", "git", "*"))
	if err != nil || len(clones) != 1 {
		t.Fatalf("operator clone: %v %v", clones, err)
	}
	root := filepath.Join(t.TempDir(), "pkgroot")
	provisioned := filepath.Join(root, "test", "gitlib", "0.1.0")
	copyTree(t, clones[0], provisioned)
	if err := os.RemoveAll(filepath.Join(provisioned, ".git")); err != nil {
		t.Fatal(err)
	}
	chmodTree(t, root, 0o555)
	t.Cleanup(func() { chmodTree(t, root, 0o755) })
	rootSnap := treeSnapshot(t, root)
	env := map[string]string{"AILANG_PACKAGE_ROOT": root}

	f.assertConfinedRow(t, f.template, env, matrixRow{args: []string{"check", "main.ail"}, kind: mustSucceed})
	f.assertConfinedRow(t, f.template, env, matrixRow{args: []string{"lock"}, kind: mustSucceed, ledger: []string{pkg.LockFileName}})
	f.assertConfinedRow(t, f.template, env, matrixRow{args: []string{"lock", "--check"}, kind: mustSucceed})
	f.assertConfinedRow(t, f.template, env, matrixRow{args: []string{"pkg", "tree"}, kind: mustSucceed})
	if d := snapshotDiff(rootSnap, treeSnapshot(t, root), nil); len(d) != 0 {
		t.Fatalf("package root changed: %v", d)
	}

	// AC5 at the binary: tampered provisioned content fails check loudly.
	chmodTree(t, root, 0o755)
	mustWrite(t, filepath.Join(provisioned, "core.ail"), "module test/gitlib/core\nexport pure func answer() -> int = 7\n", 0o644)
	r := f.run(t, true, f.template, "", env, "check", "main.ail")
	if r.exit == 0 || !strings.Contains(r.stderr, "test/gitlib content changed") {
		t.Fatalf("tampered provisioned git dep: exit %d\n%s", r.exit, r.stderr)
	}

	// AC6: a missing package is the named not-provisioned error.
	if err := os.RemoveAll(provisioned); err != nil {
		t.Fatal(err)
	}
	r = f.run(t, true, f.template, "", env, "check", "main.ail")
	if r.exit == 0 || !strings.Contains(r.stderr, "not provisioned in AILANG_PACKAGE_ROOT") {
		t.Fatalf("missing provisioned git dep: exit %d\n%s", r.exit, r.stderr)
	}
}

// AC4: the policy-tool route. `{"op":"lock"}` with cli_allow = ["lock"]
// runs the real binary as a confined child; it must refuse and leave HOME
// and the sandbox (bar the declared ailang.lock ledger) untouched.
func TestPkgConfinementPolicyToolLock(t *testing.T) {
	f := newConfinementFixture(t)
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	sandbox := filepath.Join(dir, "sandbox")
	copyTree(t, f.template, sandbox)
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	shimLog := filepath.Join(dir, "git.log")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("PATH", f.shimDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GIT_SHIM_LOG", shimLog)
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("AILANG_REGISTRY", f.recorder.URL)
	t.Setenv("AILANG_REGISTRY_VALIDATOR", f.recorder.URL)
	t.Setenv("AILANG_PACKAGE_ROOT", "")
	t.Setenv("AILANG_STATE_DIR", "")
	policyPath := filepath.Join(dir, "policy.toml")
	mustWrite(t, policyPath, "allowed_caps = [\"IO\", \"FS\"]\nfs_sandbox = \""+filepath.ToSlash(sandbox)+"\"\nentry = \"main\"\ncli_allow = [\"lock\"]\n", 0o644)
	host, err := policytool.Open(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	host.SetBinary(f.bin)

	before := treeSnapshot(t, sandbox)
	resp := host.Dispatch(policytool.Request{Op: "lock"})
	if resp.ExitCode == 0 || !strings.Contains(resp.Stderr, confinedText) {
		t.Fatalf("policy-tool lock not refused: %+v", resp)
	}
	if d := snapshotDiff(before, treeSnapshot(t, sandbox), []string{pkg.LockFileName}); len(d) != 0 {
		t.Errorf("sandbox changed: %v", d)
	}
	if s := treeSnapshot(t, home); len(s) != 1 {
		t.Errorf("HOME changed: %v", s)
	}
	if calls := networkGitCalls(t, shimLog); len(calls) != 0 {
		t.Errorf("network git invoked: %v", calls)
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func chmodTree(t *testing.T, root string, dirMode os.FileMode) {
	t.Helper()
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			_ = os.Chmod(p, dirMode)
		}
		return nil
	})
}

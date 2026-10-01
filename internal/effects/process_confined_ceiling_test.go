//go:build !js

package effects

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// Security audit 2026-10-01 F-A1: confined git walked UP from the fs_sandbox
// and read an ancestor repository's history when the sandbox was a repo
// subdirectory. Discovery is now bounded with GIT_CEILING_DIRECTORIES =
// the sandbox's parent (a ceiling equal to the cwd is ignored by git).

func needGit(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("posix git fixture")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=ParentAuthor", "GIT_AUTHOR_EMAIL=parent@secret.example", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// confinedCtx is a restricted-mode effect context rooted at sandbox.
func confinedCtx(t *testing.T, sandbox string) *EffContext {
	t.Helper()
	ctx := NewEffContext(nil)
	ctx.Env.Sandbox = sandbox
	ctx.Env.ProtectGitDir = true
	ctx.Grant(NewCapability("IO"))
	ctx.Grant(NewCapability("FS"))
	ctx.Grant(NewCapability("Process"))
	pc := NewProcessContext()
	pc.Timeout = 20 * time.Second
	if err := pc.ResolveAllowlist("git:status,git:diff,git:log"); err != nil {
		t.Fatal(err)
	}
	pc.Confined = true
	ctx.Process = pc
	t.Cleanup(func() { _ = ctx.CloseFSRoot() })
	return ctx
}

func TestGitCeiling_IsResolvedParentOfSandbox(t *testing.T) {
	dir := t.TempDir() // macOS: /var/folders/... → /private/var/folders/...
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	sb := filepath.Join(dir, "sandbox")
	if err := os.Mkdir(sb, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := gitCeiling(sb)
	if err != nil || got != real {
		t.Fatalf("gitCeiling(%s) = %q, %v; want the symlink-resolved parent %q", sb, got, err, real)
	}
	// A symlinked sandbox: the ceiling is the parent of the TARGET — the
	// directory git sees as its cwd — not of the link.
	target := filepath.Join(dir, "elsewhere", "clone")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if got, err := gitCeiling(link); err != nil || got != filepath.Join(real, "elsewhere") {
		t.Fatalf("gitCeiling(symlink) = %q, %v; want %q", got, err, filepath.Join(real, "elsewhere"))
	}
}

func TestGitCeiling_RefusesWhatItCannotBind(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix-only: restricted mode (and its confined git) is refused on Windows")
	}
	if _, err := gitCeiling(""); err == nil || !strings.Contains(err.Error(), "fs_sandbox") {
		t.Errorf("no sandbox must be refused naming fs_sandbox, got %v", err)
	}
	if _, err := gitCeiling(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("a sandbox that does not exist must be refused")
	}
	colon := filepath.Join(t.TempDir(), "a:b")
	if err := os.MkdirAll(filepath.Join(colon, "sb"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := gitCeiling(filepath.Join(colon, "sb")); err == nil || !strings.Contains(err.Error(), "GIT_CEILING_DIRECTORIES") {
		t.Errorf("a parent containing ':' would split into ceilings that do not bind; must be refused, got %v", err)
	}
}

func TestConfinedEnv_CeilingSetAndCallerGITStripped(t *testing.T) {
	sb := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", "/caller-chosen")
	t.Setenv("GIT_DISCOVERY_ACROSS_FILESYSTEM", "1")
	t.Setenv("GIT_DIR", "/elsewhere/.git")
	env, err := confinedEnv(sb)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := gitCeiling(sb)
	var ceilings []string
	for _, kv := range env {
		if strings.HasPrefix(kv, "GIT_CEILING_DIRECTORIES=") {
			ceilings = append(ceilings, kv)
		}
		if strings.HasPrefix(kv, "GIT_DIR=") || strings.HasPrefix(kv, "GIT_DISCOVERY_ACROSS_FILESYSTEM=") {
			t.Errorf("caller's %s reached the child", kv)
		}
	}
	if len(ceilings) != 1 || ceilings[0] != "GIT_CEILING_DIRECTORIES="+want {
		t.Fatalf("exactly one ceiling, the sandbox's parent %q; got %v", want, ceilings)
	}
	if _, err := confinedEnv(""); err == nil {
		t.Fatal("confinedEnv without a sandbox must fail")
	}
}

// F-A8: hooks stay disabled, spelled as a path no one can create; implicit
// bare repos are refused; diff-rendering subcommands force off external
// diff and textconv.
func TestConfinedGit_HardeningArgv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix-only: restricted mode (and its confined git) is refused on Windows")
	}
	if err := os.MkdirAll(gitHooksDisabledPath, 0o755); err == nil {
		t.Fatalf("%s must be impossible to create", gitHooksDisabledPath)
	}
	for _, sub := range []string{"status", "diff", "log"} {
		argv, err := confineGit([]string{sub})
		if err != nil {
			t.Fatal(err)
		}
		joined := strings.Join(argv, " ")
		for _, must := range []string{"core.hooksPath=" + gitHooksDisabledPath, "safe.bareRepository=explicit", "core.fsmonitor=false"} {
			if !strings.Contains(joined, must) {
				t.Errorf("git %s argv lacks %s: %v", sub, must, argv)
			}
		}
		i := slices.Index(argv, sub)
		forced := sub == "diff" || sub == "log"
		if hasNoExt := slices.Contains(argv[i:], "--no-ext-diff") && slices.Contains(argv[i:], "--no-textconv"); hasNoExt != forced {
			t.Errorf("git %s: --no-ext-diff/--no-textconv forced=%v, want %v: %v", sub, hasNoExt, forced, argv)
		}
	}
}

// The hooks setting is a control in its own right: a hook planted in the
// repo's hooks dir must not run under the hardening (proved with commit,
// which runs pre-commit — the confined trio never would).
func TestConfinedGit_HooksPathDisablesPlantedHook(t *testing.T) {
	needGit(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	marker := filepath.Join(t.TempDir(), "HOOK-RAN")
	hook := filepath.Join(repo, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	env, err := confinedEnv(repo)
	if err != nil {
		t.Fatal(err)
	}
	args := append(append([]string{}, gitHardening...), "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "x")
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit under hardening: %v\n%s", err, out)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("CONFINEMENT FAILURE: a planted pre-commit hook ran under core.hooksPath=" + gitHooksDisabledPath)
	}
	// Control: without the hardening the same hook does run.
	plain := exec.Command("git", "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "y")
	plain.Dir = repo
	plain.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := plain.CombinedOutput(); err != nil {
		t.Fatalf("control commit: %v\n%s", err, out)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("control: the planted hook should run without the hardening (fixture broken)")
	}
}

// parentRepoWithSubdir builds the audit's probe: a repository with commits
// whose subdirectory (no .git of its own) is the sandbox. The sandbox path is
// left UNRESOLVED (macOS /var → /private/var) to cover the symlink case.
func parentRepoWithSubdir(t *testing.T) (repo, sub string) {
	t.Helper()
	needGit(t)
	repo = t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "top.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", "top.txt")
	gitIn(t, repo, "commit", "-q", "-m", "PARENT-SECRET-COMMIT-1")
	if err := os.WriteFile(filepath.Join(repo, "top.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "commit", "-q", "-am", "PARENT-SECRET-COMMIT-2")
	sub = filepath.Join(repo, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "local.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo, sub
}

// The audit's live probe: sandbox = a repo subdirectory. log and status must
// fail "not a git repository" (git's own verdict under the ceiling) and print
// nothing of the parent; diff is refused before spawning (see
// TestConfinedGit_DiffOutsideRepoIsNotNoIndex).
func TestConfinedGit_RepoSubdirSandboxDoesNotWalkUp(t *testing.T) {
	_, sub := parentRepoWithSubdir(t)
	ctx := confinedCtx(t, sub)
	v, err := gitExec(ctx, "diff", "HEAD~1", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if ctor, _, msg := resultText(t, v); ctor != "Err" || !strings.Contains(msg, "NotAllowed") || strings.Contains(msg, "PARENT-SECRET") {
		t.Errorf("git diff from a non-repository sandbox must be NotAllowed, got %s %s", ctor, msg)
	}
	for _, args := range [][]string{
		{"log", "--format=%an <%ae> %s"},
		{"log", "--oneline", "-n", "5", "--stat"},
		{"status", "--porcelain"},
	} {
		v, err := gitExec(ctx, args...)
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		ctor, out, errb := resultText(t, v)
		if ctor != "Ok" {
			t.Fatalf("git %v: want a completed process (exit 128), got %s %s", args, ctor, errb)
		}
		exit := v.(*eval.TaggedValue).Fields[0].(*eval.RecordValue).Fields["exitCode"].(*eval.IntValue).Value
		if exit == 0 || !strings.Contains(errb, "not a git repository") {
			t.Errorf("git %v from a repo-subdir sandbox must fail 'not a git repository', got exit %d stderr %q", args, exit, errb)
		}
		for _, leak := range []string{"PARENT-SECRET", "parent@secret.example", "ParentAuthor", "top.txt", "sub/"} {
			if strings.Contains(out, leak) || strings.Contains(errb, leak) {
				t.Errorf("CONFINEMENT FAILURE: git %v leaked %q from the ancestor repository: out=%q err=%q", args, leak, out, errb)
			}
		}
	}
}

// Positive control: sandbox == repo root (the clone-root case every deployed
// policy uses) still runs status, log and a real patch diff.
func TestConfinedGit_CloneRootSandboxStillWorks(t *testing.T) {
	repo, _ := parentRepoWithSubdir(t)
	if err := os.WriteFile(filepath.Join(repo, "top.txt"), []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := confinedCtx(t, repo)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"log", "--format=%s"}, "PARENT-SECRET-COMMIT-2"},
		{[]string{"status", "--porcelain"}, " M top.txt"},
		{[]string{"diff"}, "+three"},
		{[]string{"diff", "HEAD~1", "HEAD"}, "+two"},
		{[]string{"diff", "--stat", "--", "top.txt"}, "top.txt"},
	} {
		v, err := gitExec(ctx, tc.args...)
		if err != nil {
			t.Fatalf("git %v: %v", tc.args, err)
		}
		if ctor, out, errb := resultText(t, v); ctor != "Ok" || !strings.Contains(out, tc.want) {
			t.Errorf("git %v at the clone root: want %q, got %s out=%q err=%q", tc.args, tc.want, ctor, out, errb)
		}
	}
}

// Within the bound, the sandbox itself must not be usable as a planted
// repository: a sandbox that is not a clone root is agent-writable, and
// HEAD/objects/refs/config written there would make it an implicit bare
// repo with the agent's config. safe.bareRepository=explicit refuses it.
func TestConfinedGit_PlantedBareRepoInSandboxRefused(t *testing.T) {
	repo, sub := parentRepoWithSubdir(t)
	for _, name := range []string{"HEAD", "objects", "refs"} {
		if out, err := exec.Command("cp", "-R", filepath.Join(repo, ".git", name), filepath.Join(sub, name)).CombinedOutput(); err != nil {
			t.Fatalf("cp %s: %v\n%s", name, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(sub, "config"), []byte("[core]\n\tbare = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := confinedCtx(t, sub)
	v, err := gitExec(ctx, "log", "--format=%s")
	if err != nil {
		t.Fatal(err)
	}
	_, out, errb := resultText(t, v)
	if strings.Contains(out, "PARENT-SECRET") || !strings.Contains(errb, "bare repository") {
		t.Fatalf("a planted bare repo at the sandbox must be refused by safe.bareRepository=explicit: out=%q err=%q", out, errb)
	}
}

// Defence in depth for the policy refusal: a confined context with no
// sandbox never spawns git.
func TestConfinedGit_NoSandboxRefused(t *testing.T) {
	needGit(t)
	ctx := confinedCtx(t, "")
	v, err := gitExec(ctx, "log", "--oneline")
	if err != nil {
		t.Fatal(err)
	}
	if ctor, _, msg := resultText(t, v); ctor != "Err" || !strings.Contains(msg, "NotAllowed") || !strings.Contains(msg, "fs_sandbox") {
		t.Fatalf("confined git without a sandbox must be NotAllowed naming fs_sandbox, got %s %s", ctor, msg)
	}
}

// `.git` protection is case-folded: on a case-insensitive filesystem
// (macOS APFS) `.GIT/config` is `.git/config`.
func TestFSCheckMutation_GitDirCaseFolded(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Env.Sandbox = "/sb"
	ctx.Env.ProtectGitDir = true
	for _, p := range []string{".GIT/config", ".Git/hooks/pre-commit", "sub/.gIt", "/sb/.GIT/config"} {
		if err := ctx.fsCheckMutation(p); err == nil {
			t.Errorf("%s must be protected", p)
		}
	}
	for _, p := range []string{".gitignore", ".GITATTRIBUTES", "git/config", "a.git/x"} {
		if err := ctx.fsCheckMutation(p); err != nil {
			t.Errorf("%s is an ordinary path: %v", p, err)
		}
	}
}

// Outside a repository `git diff <a> <b>` is an implicit --no-index diff of
// two paths — with rev-shaped tokens that climb (`d/../../etc/hosts`) it read
// arbitrary host files. Both the traversal-shaped rev and diff outside a
// repository root are refused before any process exists.
func TestConfinedGit_DiffOutsideRepoIsNotNoIndex(t *testing.T) {
	_, sub := parentRepoWithSubdir(t)
	if err := os.Mkdir(filepath.Join(sub, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := confinedCtx(t, sub)
	up := "d" + strings.Repeat("/..", 40)
	for _, args := range [][]string{
		{"diff", up + "/etc/hosts", up + "/etc/shells"},
		{"diff", "local.txt", "d"},
		{"log", up + "/etc/hosts"},
	} {
		v, err := gitExec(ctx, args...)
		if err != nil {
			t.Fatal(err)
		}
		if ctor, out, msg := resultText(t, v); ctor != "Err" || !strings.Contains(msg, "NotAllowed") {
			t.Errorf("git %v must be NotAllowed, got %s out=%q %s", args, ctor, out, msg)
		}
	}
	// Rev ranges are untouched by the traversal rule.
	if hasDotDotSegment("HEAD~1..HEAD") || hasDotDotSegment("main...feature/x") || !hasDotDotSegment("a/../b") || !hasDotDotSegment("..") {
		t.Error("hasDotDotSegment must refuse path segments '..' and only those")
	}
}

//go:build !js

package effects

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M-EXECUTOR-POLICY-HARDENING M6 — the confined Process adapter for git.
//
// Restricted mode admits Process only for entries with a confined schema
// (git:status, git:diff, git:log). The schema builds the argv in Go, the
// invocation is hardened (config from the launcher's clone only, no hooks,
// no fsmonitor, no pager, no GIT_* from the caller), and `.git/` is
// read-only to the agent so the config cannot be planted.

// confinedRepo makes a real git repo inside a sandbox with one commit.
func confinedRepo(t *testing.T) (sandbox string, ctx *EffContext) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("posix git fixture")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	sandbox = t.TempDir()
	if real, err := filepath.EvalSymlinks(sandbox); err == nil {
		sandbox = real
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = sandbox
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(sandbox, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-q", "-m", "first")
	if err := os.WriteFile(filepath.Join(sandbox, "a.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx = NewEffContext(nil)
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
	return sandbox, ctx
}

func gitExec(ctx *EffContext, args ...string) (eval.Value, error) {
	list := make([]eval.Value, len(args))
	for i, a := range args {
		list[i] = &eval.StringValue{Value: a}
	}
	return Call(ctx, "Process", "exec", []eval.Value{&eval.StringValue{Value: "git"}, &eval.ListValue{Elements: list}})
}

// resultText returns (ctor, stdout, stderr) of a Process result value.
func resultText(t *testing.T, v eval.Value) (string, string, string) {
	t.Helper()
	tv, ok := v.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("not a tagged value: %v", v)
	}
	if tv.CtorName != "Ok" {
		msg := ""
		if len(tv.Fields) > 0 {
			if inner, ok := tv.Fields[0].(*eval.TaggedValue); ok && len(inner.Fields) > 0 {
				msg = inner.CtorName + ":" + inner.Fields[0].String()
			} else {
				msg = tv.Fields[0].String()
			}
		}
		return tv.CtorName, "", msg
	}
	rec := tv.Fields[0].(*eval.RecordValue)
	out := string(rec.Fields["stdout"].(*eval.BytesValue).Value)
	errb := string(rec.Fields["stderr"].(*eval.BytesValue).Value)
	return "Ok", out, errb
}

// The RCE the design names: a repo config planted with core.fsmonitor runs a
// command on every `git status`. With the confined adapter it must not.
func TestConfinedGit_FsmonitorInRepoConfigDoesNotRun(t *testing.T) {
	sandbox, ctx := confinedRepo(t)
	marker := filepath.Join(sandbox, "..", "pwned-"+filepath.Base(sandbox))
	cfg := filepath.Join(sandbox, ".git", "config")
	b, _ := os.ReadFile(cfg)
	if err := os.WriteFile(cfg, append(b, []byte("[core]\n\tfsmonitor = \"touch "+marker+"\"\n\thooksPath = /nonexistent\n[diff]\n\texternal = touch "+marker+"\n[core]\n\tpager = touch "+marker+"\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"status", "--porcelain"}, {"diff"}, {"log", "--oneline", "-n", "1"}} {
		_, err := gitExec(ctx, args...)
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	time.Sleep(200 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("CONFINEMENT FAILURE: a command from the repo config ran under git status/diff/log")
	}
}

// Flags that reach outside the clone are refused by the schema, before any
// process starts.
func TestConfinedGit_OutsideReachingFlagsRefused(t *testing.T) {
	sandbox, ctx := confinedRepo(t)
	outside := filepath.Join(sandbox, "..", "leak-"+filepath.Base(sandbox))
	cases := [][]string{
		{"diff", "--no-index", "/etc/passwd", "/dev/null"},
		{"diff", "--output=" + outside},
		{"diff", "--output", outside},
		{"log", "--output=" + outside},
		{"-c", "core.fsmonitor=touch " + outside, "status"},
		{"--git-dir=/etc", "status"},
		{"-C", "/", "status"},
		{"--exec-path=/tmp", "status"},
		{"status", "--porcelain", "--", "../"},
		{"diff", "--ext-diff"},
		{"log", "--format=%H", "--", "/etc/passwd"},
		{"log", "-n", "1", "--stdin"},
		{"status", "--porcelain", "-c"},
	}
	for _, args := range cases {
		v, err := gitExec(ctx, args...)
		if err != nil {
			t.Fatalf("git %v: Go error %v", args, err)
		}
		ctor, _, msg := resultText(t, v)
		if ctor != "Err" || !strings.Contains(msg, "NotAllowed") {
			t.Errorf("git %v must be NotAllowed, got %s %s", args, ctor, msg)
		}
	}
	if _, err := os.Stat(outside); err == nil {
		t.Fatal("a refused invocation wrote outside the sandbox")
	}
}

// Positive control: the read-only trio returns real output, and the
// caller's GIT_* environment does not reach the child.
func TestConfinedGit_StatusDiffLogWork(t *testing.T) {
	_, ctx := confinedRepo(t)
	t.Setenv("GIT_EXTERNAL_DIFF", "/bin/false")
	t.Setenv("GIT_DIR", "/nonexistent")
	v, err := gitExec(ctx, "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if ctor, out, msg := resultText(t, v); ctor != "Ok" || !strings.Contains(out, " M a.txt") {
		t.Fatalf("status: %s %q %s", ctor, out, msg)
	}
	v, _ = gitExec(ctx, "diff", "--stat", "--", "a.txt")
	if ctor, out, msg := resultText(t, v); ctor != "Ok" || !strings.Contains(out, "a.txt") {
		t.Fatalf("diff: %s %q %s", ctor, out, msg)
	}
	v, _ = gitExec(ctx, "log", "--oneline", "-n", "1")
	if ctor, out, msg := resultText(t, v); ctor != "Ok" || !strings.Contains(out, "first") {
		t.Fatalf("log: %s %q %s", ctor, out, msg)
	}
	v, _ = gitExec(ctx, "log", "--format=%s", "-n1", "HEAD", "--", "a.txt")
	if ctor, out, _ := resultText(t, v); ctor != "Ok" || !strings.Contains(out, "first") {
		t.Fatalf("log rev + pathspec: %s %q", ctor, out)
	}
}

// Confined mode is exec-only: spawnProcess / asyncExecProcess have no
// hardened shape and are refused.
func TestConfinedGit_SpawnAndAsyncRefused(t *testing.T) {
	_, ctx := confinedRepo(t)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	t.Cleanup(ctx.Stream.CloseAll)
	_, err := Call(ctx, "Stream", "asyncExecProcess", []eval.Value{
		&eval.StringValue{Value: "git"},
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "status"}}},
		&eval.StringValue{Value: "s"}, &eval.IntValue{Value: 1}, &eval.IntValue{Value: 64},
	})
	if err == nil || !strings.Contains(err.Error(), "confined") {
		t.Fatalf("asyncExecProcess under confinement must be refused: %v", err)
	}
	v, err := Call(ctx, "Process", "spawnProcess", []eval.Value{
		&eval.StringValue{Value: "git"},
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "status"}}},
	})
	if err == nil {
		if ctor, _, msg := resultText(t, v); ctor == "Ok" || !strings.Contains(msg, "confined") {
			t.Fatalf("spawnProcess under confinement must be refused: %s %s", ctor, msg)
		}
	}
}

// `.git/` is read-only to the agent's programs in restricted mode: the config
// cannot be planted through the FS effect.
func TestConfinedGit_DotGitIsReadOnlyToFS(t *testing.T) {
	sandbox, ctx := confinedRepo(t)
	before, _ := os.ReadFile(filepath.Join(sandbox, ".git", "config"))
	for _, args := range [][]eval.Value{
		{&eval.StringValue{Value: ".git/config"}, &eval.StringValue{Value: "[core]\n\tfsmonitor = evil\n"}},
		{&eval.StringValue{Value: ".git/hooks/pre-commit"}, &eval.StringValue{Value: "#!/bin/sh\n"}},
		{&eval.StringValue{Value: "sub/../.git/config"}, &eval.StringValue{Value: "x"}},
		{&eval.StringValue{Value: filepath.Join(sandbox, ".git", "config")}, &eval.StringValue{Value: "x"}},
	} {
		for _, op := range []string{"writeFile", "appendFile", "writeFileResult", "appendFileResult"} {
			v, err := Call(ctx, "FS", op, args)
			if err == nil {
				if tv, ok := v.(*eval.TaggedValue); !ok || tv.CtorName != "Err" {
					t.Errorf("%s(%s) must be refused, got %v", op, args[0], v)
				}
			}
		}
	}
	for _, op := range []string{"removeFile", "removeFileResult", "mkdir", "mkdirResult"} {
		if v, err := Call(ctx, "FS", op, []eval.Value{&eval.StringValue{Value: ".git/hooks"}}); err == nil {
			if tv, ok := v.(*eval.TaggedValue); !ok || tv.CtorName != "Err" {
				t.Errorf("%s(.git/hooks) must be refused, got %v", op, v)
			}
		}
	}
	if v, err := Call(ctx, "FS", "renameFile", []eval.Value{&eval.StringValue{Value: "a.txt"}, &eval.StringValue{Value: ".git/config"}}); err == nil {
		t.Errorf("rename into .git must be refused, got %v", v)
	}
	if v, err := Call(ctx, "FS", "renameFile", []eval.Value{&eval.StringValue{Value: ".git/config"}, &eval.StringValue{Value: "b.txt"}}); err == nil {
		t.Errorf("rename out of .git must be refused, got %v", v)
	}
	after, _ := os.ReadFile(filepath.Join(sandbox, ".git", "config"))
	if string(before) != string(after) {
		t.Fatal(".git/config changed")
	}
	// Reading .git is fine; .gitignore is an ordinary file.
	if _, err := Call(ctx, "FS", "readFile", []eval.Value{&eval.StringValue{Value: ".git/HEAD"}}); err != nil {
		t.Fatalf("reading .git must still work: %v", err)
	}
	if _, err := Call(ctx, "FS", "writeFile", []eval.Value{&eval.StringValue{Value: ".gitignore"}, &eval.StringValue{Value: "x\n"}}); err != nil {
		t.Fatalf(".gitignore must stay writable: %v", err)
	}
}

// ConfinedProcessEntry is what policy.Resolve consults.
func TestConfinedProcessEntry(t *testing.T) {
	for entry, want := range map[string]bool{
		"git:status": true, "git:diff": true, "git:log": true,
		"git": false, "git:*": false, "git:push": false, "git:commit": false, "sh": false, "echo": false, "gh:pr:list": false,
	} {
		if got := ConfinedProcessEntry(entry); got != want {
			t.Errorf("ConfinedProcessEntry(%q) = %v, want %v", entry, got, want)
		}
	}
}

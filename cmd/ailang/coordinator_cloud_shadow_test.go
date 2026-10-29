package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
)

// shadowJobFixture is a bare "remote" with one commit on dev, and the job's
// seams pointed at a temp workspace, an agent that edits a file and then tries
// to push on its own, and a PR opener that records its calls.
type shadowJobFixture struct {
	remote    string
	repoURL   string
	prCalls   int
	agentPush error // the agent's own push attempt
}

func newShadowJobFixture(t *testing.T) *shadowJobFixture {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(k, "Shadow test")
	}
	for _, k := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "test@example.com")
	}
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv(config.EnvTaskInputs, "")
	t.Setenv(config.EnvPushBranch, "")

	f := &shadowJobFixture{}
	seed := t.TempDir()
	runGit(t, seed, "init", "-b", "dev")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, seed, "add", ".")
	runGit(t, seed, "commit", "-m", "seed")
	f.remote = filepath.Join(t.TempDir(), "remote.git")
	runGit(t, seed, "clone", "--bare", seed, f.remote)
	path := filepath.ToSlash(f.remote)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path // file:///C:/... on Windows
	}
	f.repoURL = "file://" + path

	oldRoot, oldExec, oldPR := cloudWorkspaceRoot, runExecutorFn, openPullRequestFn
	t.Cleanup(func() { cloudWorkspaceRoot, runExecutorFn, openPullRequestFn = oldRoot, oldExec, oldPR })
	cloudWorkspaceRoot = t.TempDir()
	runExecutorFn = func(ctx context.Context, workDir, _, _, _, _, _, _, _ string) (*executor.Result, error) {
		if err := os.WriteFile(filepath.Join(workDir, "fix.ail"), []byte("-- fix\n"), 0o644); err != nil {
			return nil, err
		}
		// The negative test: the agent tries to publish on its own.
		push := exec.CommandContext(ctx, "git", "-C", workDir, "push", "origin", "HEAD:refs/heads/agent-push")
		f.agentPush = push.Run()
		return &executor.Result{NumTurns: 1}, nil
	}
	openPullRequestFn = func(context.Context, string, string, string, string, string, string) error {
		f.prCalls++
		return nil
	}
	return f
}

func (f *shadowJobFixture) remoteRefs(t *testing.T) string {
	t.Helper()
	return runGit(t, f.remote, "for-each-ref", "--format=%(refname)", "refs/heads/")
}

// A shadow run keeps its diff and pushes nothing: no coordinator branch, no
// branch the agent tried to push itself, and no PR call.
func TestShadowRunKeepsEvidenceAndPublishesNothing(t *testing.T) {
	f := newShadowJobFixture(t)
	t.Setenv(config.EnvTaskShadow, "true")
	t.Setenv(config.EnvOpenRouterAPIKey, "fleet-key")
	t.Setenv(config.EnvOpenRouterKeyShadow, "shadow-key")
	var keyDuringRun string
	inner := runExecutorFn
	runExecutorFn = func(ctx context.Context, workDir, a, b, c, d, e, g, h string) (*executor.Result, error) {
		keyDuringRun = os.Getenv(config.EnvOpenRouterAPIKey)
		return inner(ctx, workDir, a, b, c, d, e, g, h)
	}

	_, result, ev, err := executeCloudTask(context.Background(), "task-p-shadow", "pkg-x", f.repoURL, "dev", "fix it", "motoko", "", "m", "1m")
	if err != nil {
		t.Fatalf("shadow run failed: %v", err)
	}
	if result == nil || ev.BaseCommit == "" || !strings.Contains(strings.Join(ev.ChangedFiles, ","), "fix.ail") || !strings.Contains(ev.Diff, "fix.ail") {
		t.Fatalf("shadow evidence missing: base=%q files=%v diff=%d bytes", ev.BaseCommit, ev.ChangedFiles, len(ev.Diff))
	}
	if refs := f.remoteRefs(t); refs != "refs/heads/dev" {
		t.Fatalf("remote refs after a shadow run = %q, want only refs/heads/dev", refs)
	}
	if f.agentPush == nil {
		t.Fatal("the agent's own push succeeded in a shadow run")
	}
	if f.prCalls != 0 {
		t.Fatalf("PR opened %d times for a shadow run", f.prCalls)
	}
	if keyDuringRun != "shadow-key" {
		t.Fatalf("agent ran with OPENROUTER_API_KEY=%q, want the shadow key", keyDuringRun)
	}
}

// No shadow key: the run is refused before anything is cloned or spent.
func TestShadowRunWithoutShadowKeyRefused(t *testing.T) {
	f := newShadowJobFixture(t)
	t.Setenv(config.EnvTaskShadow, "true")
	t.Setenv(config.EnvOpenRouterAPIKey, "fleet-key")
	t.Setenv(config.EnvOpenRouterKeyShadow, "")
	ran := false
	runExecutorFn = func(context.Context, string, string, string, string, string, string, string, string) (*executor.Result, error) {
		ran = true
		return &executor.Result{}, nil
	}
	_, _, _, err := executeCloudTask(context.Background(), "task-p-shadow", "pkg-x", f.repoURL, "dev", "fix", "motoko", "", "m", "1m")
	if err == nil || !strings.Contains(err.Error(), "OPENROUTER_API_KEY_SHADOW") || ran {
		t.Fatalf("err=%v ran=%v; want a refusal naming the shadow key and no run", err, ran)
	}
	if os.Getenv(config.EnvOpenRouterAPIKey) != "fleet-key" {
		t.Fatal("the fleet key was changed by a refused shadow run")
	}
}

// Regression: an ordinary run still pushes its branch and opens the PR.
func TestNonShadowRunStillPushesAndOpensPR(t *testing.T) {
	f := newShadowJobFixture(t)
	t.Setenv(config.EnvTaskShadow, "false")

	if _, _, _, err := executeCloudTask(context.Background(), "task-p", "pkg-x", f.repoURL, "dev", "fix it", "pi", "", "m", "1m"); err != nil {
		t.Fatalf("ordinary run failed: %v", err)
	}
	if refs := f.remoteRefs(t); !strings.Contains(refs, "refs/heads/coordinator/task-p") {
		t.Fatalf("remote refs = %q, want the coordinator branch pushed", refs)
	}
	if f.prCalls != 1 {
		t.Fatalf("PR calls = %d, want 1", f.prCalls)
	}
}

// With a write-capable token present, a shadow run's agent still gets no git
// credential file.
func TestShadowRunWithholdsChildGitCredential(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "write-capable-token")
	t.Setenv("AILANG_CHILD_GIT_CREDENTIALS", "repo")
	for _, shadow := range []string{"true", "false"} {
		t.Setenv(config.EnvTaskShadow, shadow)
		task := &executor.Task{Workspace: t.TempDir()}
		cleanup, err := childGitCredential(task, "https://github.com/org/repo.git")
		if err != nil {
			t.Fatal(err)
		}
		got := task.GitCredentialFile != ""
		cleanup()
		if want := shadow == "false"; got != want {
			t.Fatalf("shadow=%s: credential file written = %v, want %v", shadow, got, want)
		}
	}
}

func TestNeedsDeployKeyWriteCheck(t *testing.T) {
	for _, c := range []struct {
		shadow, repo string
		want         bool
	}{
		{"false", "org/repo", true},
		{"true", "org/repo", false},
		{"false", "", false},
	} {
		t.Setenv(config.EnvTaskShadow, c.shadow)
		if got := needsDeployKeyWriteCheck(c.repo); got != c.want {
			t.Errorf("shadow=%s repo=%q: got %v, want %v", c.shadow, c.repo, got, c.want)
		}
	}
}

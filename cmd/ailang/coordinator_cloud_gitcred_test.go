package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// M-EXECUTOR-ENV-HARDENING E5: the global helper a cloud job installs must
// hold no secret, answer the parent (which has GITHUB_TOKEN) and give the
// agent child (which does not) nothing.
func TestEnvTokenCredentialHelper_ReadsTheCallersEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cloud jobs are Linux; the helper is POSIX sh")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	if strings.Contains(envTokenCredentialHelper, "ghp_") || !strings.Contains(envTokenCredentialHelper, "$GITHUB_TOKEN") {
		t.Fatal("the helper must reference $GITHUB_TOKEN, not embed a token")
	}
	gitconfig := filepath.Join(t.TempDir(), "gitconfig")
	if out, err := exec.Command("git", "config", "--file", gitconfig, "credential.helper", envTokenCredentialHelper).CombinedOutput(); err != nil {
		t.Fatalf("git config: %v: %s", err, out)
	}
	fill := func(extra ...string) string {
		cmd := exec.Command("git", "credential", "fill")
		cmd.Env = append([]string{"PATH=" + os.Getenv("PATH"), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + gitconfig, "GIT_TERMINAL_PROMPT=0"}, extra...)
		cmd.Stdin = strings.NewReader("protocol=https\nhost=github.com\npath=owner/repo.git\n\n")
		out, _ := cmd.Output()
		return string(out)
	}
	if out := fill("GITHUB_TOKEN=tok-parent\n"); !strings.Contains(out, "password=tok-parent\n") {
		t.Errorf("parent (GITHUB_TOKEN set, Secret Manager newline): fill = %q", out)
	}
	if out := fill(); strings.Contains(out, "password=") {
		t.Errorf("child (no GITHUB_TOKEN) must get nothing: fill = %q", out)
	}
}

func TestChildGitCredential_Modes(t *testing.T) {
	const repo = "https://github.com/owner/repo.git"
	t.Setenv("GITHUB_TOKEN", "tok")

	t.Setenv("AILANG_CHILD_GIT_CREDENTIALS", "")
	task := &executor.Task{}
	cleanup, err := childGitCredential(task, repo)
	if err != nil {
		t.Fatal(err)
	}
	if task.GitCredentialFile == "" || len(task.GitCredentialScopes) != 2 {
		t.Fatalf("default mode must hand the child a repo-scoped file: %+v", task)
	}
	file := task.GitCredentialFile
	cleanup()
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("cleanup must remove the file")
	}

	t.Setenv("AILANG_CHILD_GIT_CREDENTIALS", "none")
	task = &executor.Task{}
	if _, err := childGitCredential(task, repo); err != nil || task.GitCredentialFile != "" {
		t.Errorf("none: file=%q err=%v, want no file", task.GitCredentialFile, err)
	}

	t.Setenv("AILANG_CHILD_GIT_CREDENTIALS", "")
	task = &executor.Task{}
	if _, err := childGitCredential(task, "git@gh-deploy-key:owner/repo.git"); err != nil || task.GitCredentialFile != "" {
		t.Errorf("SSH deploy-key clone: file=%q err=%v, want no file", task.GitCredentialFile, err)
	}

	t.Setenv("AILANG_CHILD_GIT_CREDENTIALS", "everything")
	if _, err := childGitCredential(&executor.Task{}, repo); err == nil || !strings.Contains(err.Error(), "AILANG_CHILD_GIT_CREDENTIALS") {
		t.Errorf("unknown mode: err = %v, want an error naming the variable", err)
	}
}

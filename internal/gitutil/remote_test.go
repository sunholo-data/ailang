package gitutil

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitHubOwnerRepo(t *testing.T) {
	tests := []struct {
		name, remote, owner, repo string
	}{
		{"https", "https://github.com/acme/docs.git", "acme", "docs"},
		{"ssh", "git@github.com:acme/docs", "acme", "docs"},
		{"non-github", "https://gitlab.com/acme/docs.git", "", ""},
		{"malformed", "git@github.com:acme", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initRepo(t, tt.remote)
			owner, repo, err := GitHubOwnerRepo(context.Background(), dir)
			if err != nil || owner != tt.owner || repo != tt.repo {
				t.Fatalf("got %q/%q, %v; want %q/%q", owner, repo, err, tt.owner, tt.repo)
			}
		})
	}
}

func TestGitHubOwnerRepoGitError(t *testing.T) {
	_, _, err := GitHubOwnerRepo(context.Background(), filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected git command error")
	}
}

func initRepo(t *testing.T, remote string) string {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	cmd := exec.Command("git", "-C", dir, "remote", "add", "origin", remote)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v: %s", err, out)
	}
	return dir
}

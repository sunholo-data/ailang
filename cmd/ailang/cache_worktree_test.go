package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Agent work in this repo is deliberately worktree-isolated. Because
// .ailang/state/{evaluations,sprints}/*.json are git-TRACKED, every linked
// worktree materialises a .ailang/ directory — which used to stop the
// project-root walk at the worktree and open an empty brain, shadowing the main
// checkout's. These tests pin the redirect so that regression can't return.

// mkMainRepo builds a main worktree: .git/ (dir) + .ailang/state/.
func mkMainRepo(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir()) // macOS /var -> /private/var
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{".git", filepath.Join(".ailang", "state")} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// mkLinkedWorktree builds a linked worktree under main/.claude/worktrees/<name>,
// mirroring the real layout: a .git FILE plus a materialised .ailang/state/.
func mkLinkedWorktree(t *testing.T, main, name string) string {
	t.Helper()
	wt := filepath.Join(main, ".claude", "worktrees", name)
	if err := os.MkdirAll(filepath.Join(wt, ".ailang", "state"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitdir := filepath.Join(main, ".git", "worktrees", name)
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: "+gitdir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return wt
}

func TestMainWorktreeRoot(t *testing.T) {
	main := mkMainRepo(t)
	wt := mkLinkedWorktree(t, main, "sprint-x")

	t.Run("linked worktree resolves to main root", func(t *testing.T) {
		if got := mainWorktreeRoot(wt); got != main {
			t.Errorf("got %q want %q", got, main)
		}
	})

	t.Run("main worktree (.git is a directory) returns empty", func(t *testing.T) {
		if got := mainWorktreeRoot(main); got != "" {
			t.Errorf("main worktree must not redirect; got %q", got)
		}
	})

	t.Run("no .git at all returns empty", func(t *testing.T) {
		bare := t.TempDir()
		if got := mainWorktreeRoot(bare); got != "" {
			t.Errorf("got %q want empty", got)
		}
	})

	// A submodule .git file also starts with "gitdir:" but points at
	// .git/modules/<name> — it must NOT be treated as a worktree.
	t.Run("submodule pointer returns empty", func(t *testing.T) {
		sub := t.TempDir()
		body := "gitdir: " + filepath.Join(main, ".git", "modules", "vendorlib") + "\n"
		if err := os.WriteFile(filepath.Join(sub, ".git"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := mainWorktreeRoot(sub); got != "" {
			t.Errorf("submodule must not redirect; got %q", got)
		}
	})

	t.Run("malformed .git file returns empty", func(t *testing.T) {
		bad := t.TempDir()
		if err := os.WriteFile(filepath.Join(bad, ".git"), []byte("not a gitdir line\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := mainWorktreeRoot(bad); got != "" {
			t.Errorf("got %q want empty", got)
		}
	})
}

func TestGetProjectBrainPath_WorktreeSharesMainBrain(t *testing.T) {
	main := mkMainRepo(t)
	wt := mkLinkedWorktree(t, main, "sprint-y")
	want := filepath.Join(main, ".ailang", "state", "brain.db")

	t.Run("from worktree root", func(t *testing.T) {
		t.Chdir(wt)
		if got := getProjectBrainPath(); got != want {
			t.Errorf("worktree opened its own brain\n got: %s\nwant: %s", got, want)
		}
	})

	t.Run("from a subdirectory inside the worktree", func(t *testing.T) {
		deep := filepath.Join(wt, "internal", "microrag")
		if err := os.MkdirAll(deep, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Chdir(deep)
		if got := getProjectBrainPath(); got != want {
			t.Errorf("got %s want %s", got, want)
		}
	})

	t.Run("main checkout is unaffected", func(t *testing.T) {
		t.Chdir(main)
		if got := getProjectBrainPath(); got != want {
			t.Errorf("main checkout must keep its own brain\n got: %s\nwant: %s", got, want)
		}
	})
}

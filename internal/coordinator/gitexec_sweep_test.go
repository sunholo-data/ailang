package coordinator

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitexecSweep_MergeOperations(t *testing.T) {
	repoDir := filepath.Join(t.TempDir(), "repo")
	if err := initTestGitRepo(repoDir); err != nil {
		t.Fatalf("initialize repository: %v", err)
	}
	if err := runGit(repoDir, "checkout", "-b", "feature"); err != nil {
		t.Fatalf("create feature branch: %v", err)
	}
	writeAndCommitTestFile(t, repoDir, "feature.txt", "feature\n", "add feature")

	ctx := context.Background()
	diff, err := GetWorktreeDiff(ctx, repoDir, "main", "")
	if err != nil {
		t.Fatalf("get worktree diff: %v", err)
	}
	if !strings.Contains(diff, "feature.txt") {
		t.Fatalf("diff does not name feature.txt: %q", diff)
	}

	branch, err := getWorktreeBranch(ctx, repoDir)
	if err != nil {
		t.Fatalf("get worktree branch: %v", err)
	}
	if branch != "feature" {
		t.Fatalf("branch = %q, want feature", branch)
	}

	files, err := getChangedFiles(ctx, repoDir, "main", "")
	if err != nil {
		t.Fatalf("get changed files: %v", err)
	}
	if len(files) != 1 || files[0] != "feature.txt" {
		t.Fatalf("changed files = %v, want [feature.txt]", files)
	}

	linkedWorktree := filepath.Join(filepath.Dir(repoDir), "linked-worktree")
	if err := runGit(repoDir, "worktree", "add", "-b", "probe", linkedWorktree, "main"); err != nil {
		t.Fatalf("create linked worktree: %v", err)
	}
	t.Cleanup(func() { _ = runGit(repoDir, "worktree", "remove", "--force", linkedWorktree) })
	gotMainRepo := getMainRepoPath(linkedWorktree)
	gotInfo, gotErr := os.Stat(gotMainRepo)
	wantInfo, wantErr := os.Stat(repoDir)
	if gotErr != nil || wantErr != nil || !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("main repo path = %q, want filesystem identity with %q (got err %v, want err %v)", gotMainRepo, repoDir, gotErr, wantErr)
	}

	if err := gitCheckout(ctx, repoDir, "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}
	if output, err := gitMerge(ctx, repoDir, "feature"); err != nil {
		t.Fatalf("merge feature: %v\n%s", err, output)
	}
	head, err := getHeadCommit(ctx, repoDir)
	if err != nil {
		t.Fatalf("get head commit: %v", err)
	}
	if head == "" {
		t.Fatal("head commit is empty")
	}
	if err := gitMergeAbort(ctx, repoDir); err == nil {
		t.Fatal("merge abort unexpectedly succeeded without an active merge")
	}
}

func TestGitexecSweep_WorktreeInspectionAndCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := initTestGitRepo(repoDir); err != nil {
		t.Fatalf("initialize repository: %v", err)
	}

	wm, err := NewWorktreeManager(repoDir, filepath.Join(tmpDir, "worktrees"), 2)
	if err != nil {
		t.Fatalf("create worktree manager: %v", err)
	}
	wt, err := wm.CreateWorktree("coverage-task", "main")
	if err != nil {
		t.Fatalf("create worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wt.Path, "changed.txt"), []byte("changed\n"), 0644); err != nil {
		t.Fatalf("write worktree change: %v", err)
	}

	hasChanges, err := wm.HasChanges("coverage-task")
	if err != nil {
		t.Fatalf("check worktree changes: %v", err)
	}
	if !hasChanges {
		t.Fatal("worktree change was not detected")
	}
	summary, err := wm.GetChangeSummary("coverage-task")
	if err != nil {
		t.Fatalf("get worktree summary: %v", err)
	}
	if len(summary.FilesChanged) != 1 || summary.FilesChanged[0] != "changed.txt" {
		t.Fatalf("changed files = %v, want [changed.txt]", summary.FilesChanged)
	}
	if got := GetDefaultBranch(repoDir); got != "main" {
		t.Fatalf("default branch = %q, want main", got)
	}

	if err := wm.CleanupAll(); err != nil {
		t.Fatalf("cleanup worktrees: %v", err)
	}
	if wm.Count() != 0 {
		t.Fatalf("worktree count = %d after cleanup, want 0", wm.Count())
	}
	if _, err := os.Stat(wt.Path); !os.IsNotExist(err) {
		t.Fatalf("worktree path still exists after cleanup: %v", err)
	}
}

func TestGitexecSweep_AutoCommitPaths(t *testing.T) {
	repoDir := filepath.Join(t.TempDir(), "repo")
	if err := initTestGitRepo(repoDir); err != nil {
		t.Fatalf("initialize repository: %v", err)
	}
	logger := log.New(io.Discard, "", 0)

	if err := os.WriteFile(filepath.Join(repoDir, "daemon.txt"), []byte("daemon\n"), 0644); err != nil {
		t.Fatalf("write daemon change: %v", err)
	}
	if err := autoCommitWorktree(repoDir, "daemon path", logger); err != nil {
		t.Fatalf("daemon auto-commit: %v", err)
	}
	assertGitWorktreeClean(t, repoDir)

	if err := os.WriteFile(filepath.Join(repoDir, "approval.txt"), []byte("approval\n"), 0644); err != nil {
		t.Fatalf("write approval change: %v", err)
	}
	if err := autoCommitWorktreeChanges(repoDir, "approval path"); err != nil {
		t.Fatalf("approval auto-commit: %v", err)
	}
	assertGitWorktreeClean(t, repoDir)
}

func writeAndCommitTestFile(t *testing.T, repoDir, name, contents, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repoDir, name), []byte(contents), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	if err := runGit(repoDir, "add", name); err != nil {
		t.Fatalf("stage %s: %v", name, err)
	}
	if err := runGit(repoDir, "commit", "-m", message); err != nil {
		t.Fatalf("commit %s: %v", name, err)
	}
}

func assertGitWorktreeClean(t *testing.T, repoDir string) {
	t.Helper()
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	if len(output) != 0 {
		t.Fatalf("worktree is dirty: %s", output)
	}
}

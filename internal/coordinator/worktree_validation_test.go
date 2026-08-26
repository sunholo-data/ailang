package coordinator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// M-MESSAGE-PLANE-FAIL-LOUD M1.
//
// NewWorktreeManager used to validate its repo path ONLY when the caller passed
// an empty string (the findGitRoot branch). A supplied-but-unresolvable path was
// accepted without complaint, the daemon logged "Worktree manager ready", and the
// failure surfaced later on EVERY CleanupOrphaned tick instead.
//
// Measured 2026-08-26: the rig logged
//
//	failed to prune worktrees: chdir sunholo-data/ailang: no such file or directory
//
// every 30 seconds for ~3.5 hours, because an agent's `workspace` held a GitHub
// coordinate (relative) while the daemon's CWD was $HOME. The constructor had
// already reported success.
func TestNewWorktreeManager_RejectsNonexistentRepoDir(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "worktrees")

	wm, err := NewWorktreeManager(filepath.Join(t.TempDir(), "definitely-not-here"), baseDir, 3)
	if err == nil {
		t.Fatal("expected an error for a nonexistent repoDir, got nil (this is the 3.5-hour-loop regression)")
	}
	if wm != nil {
		t.Error("expected a nil manager when construction fails")
	}
}

// A RELATIVE coordinate is the exact shape that caused the incident: it is not
// absolute, so it resolves against the daemon's CWD rather than naming a repo.
func TestNewWorktreeManager_RejectsRelativeRepoCoordinate(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "worktrees")

	_, err := NewWorktreeManager("sunholo-data/ailang", baseDir, 3)
	if err == nil {
		t.Fatal("expected an error for a bare org/repo coordinate, got nil")
	}
	if !strings.Contains(err.Error(), "sunholo-data/ailang") {
		t.Errorf("error should name the offending path so the operator can fix it; got: %v", err)
	}
}

// A real directory that is not a git repository must also be refused: worktree
// operations shell out to git and would fail per-call otherwise.
func TestNewWorktreeManager_RejectsNonGitDirectory(t *testing.T) {
	notARepo := t.TempDir()
	baseDir := filepath.Join(t.TempDir(), "worktrees")

	_, err := NewWorktreeManager(notARepo, baseDir, 3)
	if err == nil {
		t.Fatal("expected an error for a directory that is not a git repository, got nil")
	}
}

// Control: a genuine repository still constructs. Without this the tests above
// would pass on a constructor that rejects everything.
func TestNewWorktreeManager_AcceptsRealRepo(t *testing.T) {
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init").CombinedOutput(); err != nil {
		t.Skipf("git init unavailable: %v (%s)", err, out)
	}
	baseDir := filepath.Join(t.TempDir(), "worktrees")

	wm, err := NewWorktreeManager(repo, baseDir, 3)
	if err != nil {
		t.Fatalf("a real git repo must be accepted, got: %v", err)
	}
	if wm == nil {
		t.Fatal("worktree manager is nil")
	}
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		t.Error("base directory was not created")
	}
}

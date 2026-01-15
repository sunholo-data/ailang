package coordinator

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSanitizeTaskID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple-task", "simple-task"},
		{"task_with_underscore", "task_with_underscore"},
		{"Task With Spaces", "Task-With-Spaces"},
		{"task/with/slashes", "task-with-slashes"},
		{"task@special#chars!", "task-special-chars-"},
		{"", ""},
		{"a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeTaskID(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeTaskID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeTaskIDLength(t *testing.T) {
	longID := "this-is-a-very-long-task-id-that-should-be-truncated-to-fifty-characters-maximum"
	result := sanitizeTaskID(longID)

	if len(result) > 50 {
		t.Errorf("sanitizeTaskID should truncate to 50 chars, got %d", len(result))
	}
}

func TestNewWorktreeManager(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "worktrees")

	// Skip if not in a git repo
	if _, err := findGitRoot(); err != nil {
		t.Skip("not in a git repository")
	}

	wm, err := NewWorktreeManager("", baseDir, 3)
	if err != nil {
		t.Fatalf("failed to create worktree manager: %v", err)
	}

	if wm == nil {
		t.Fatal("worktree manager is nil")
	}

	if wm.maxWorktrees != 3 {
		t.Errorf("expected max worktrees 3, got %d", wm.maxWorktrees)
	}

	// Base directory should be created
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		t.Error("base directory was not created")
	}
}

func TestWorktreeManagerMaxLimit(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "worktrees")

	// Initialize a test git repo
	repoDir := filepath.Join(tmpDir, "repo")
	if err := initTestGitRepo(repoDir); err != nil {
		t.Fatalf("failed to init test repo: %v", err)
	}

	wm, err := NewWorktreeManager(repoDir, baseDir, 2)
	if err != nil {
		t.Fatalf("failed to create worktree manager: %v", err)
	}

	// Create first worktree
	wt1, err := wm.CreateWorktree("task-1", "main")
	if err != nil {
		t.Fatalf("failed to create first worktree: %v", err)
	}
	defer func() { _ = wm.RemoveWorktree("task-1") }()

	if wt1.TaskID != "task-1" {
		t.Errorf("expected task ID 'task-1', got %q", wt1.TaskID)
	}

	// Create second worktree
	wt2, err := wm.CreateWorktree("task-2", "main")
	if err != nil {
		t.Fatalf("failed to create second worktree: %v", err)
	}
	defer func() { _ = wm.RemoveWorktree("task-2") }()

	if wt2.TaskID != "task-2" {
		t.Errorf("expected task ID 'task-2', got %q", wt2.TaskID)
	}

	// Third should fail due to limit
	_, err = wm.CreateWorktree("task-3", "main")
	if err == nil {
		t.Error("expected error when exceeding max worktrees limit")
		_ = wm.RemoveWorktree("task-3")
	}
}

func TestWorktreeManagerCreateAndRemove(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "worktrees")

	// Initialize a test git repo
	repoDir := filepath.Join(tmpDir, "repo")
	if err := initTestGitRepo(repoDir); err != nil {
		t.Fatalf("failed to init test repo: %v", err)
	}

	wm, err := NewWorktreeManager(repoDir, baseDir, 5)
	if err != nil {
		t.Fatalf("failed to create worktree manager: %v", err)
	}

	// Create worktree
	wt, err := wm.CreateWorktree("test-task", "main")
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
		t.Error("worktree directory was not created")
	}

	// Count should be 1
	if wm.Count() != 1 {
		t.Errorf("expected count 1, got %d", wm.Count())
	}

	// Get worktree
	got, ok := wm.GetWorktree("test-task")
	if !ok {
		t.Error("worktree not found")
	}
	if got.TaskID != "test-task" {
		t.Errorf("expected task ID 'test-task', got %q", got.TaskID)
	}

	// Remove worktree
	if err := wm.RemoveWorktree("test-task"); err != nil {
		t.Fatalf("failed to remove worktree: %v", err)
	}

	// Count should be 0
	if wm.Count() != 0 {
		t.Errorf("expected count 0 after removal, got %d", wm.Count())
	}

	// Should not find it anymore
	_, ok = wm.GetWorktree("test-task")
	if ok {
		t.Error("worktree should not be found after removal")
	}
}

func TestWorktreeManagerListWorktrees(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "worktrees")

	// Initialize a test git repo
	repoDir := filepath.Join(tmpDir, "repo")
	if err := initTestGitRepo(repoDir); err != nil {
		t.Fatalf("failed to init test repo: %v", err)
	}

	wm, err := NewWorktreeManager(repoDir, baseDir, 5)
	if err != nil {
		t.Fatalf("failed to create worktree manager: %v", err)
	}

	// Create multiple worktrees
	for _, id := range []string{"task-a", "task-b"} {
		_, err := wm.CreateWorktree(id, "main")
		if err != nil {
			t.Fatalf("failed to create worktree %s: %v", id, err)
		}
		defer func(id string) { _ = wm.RemoveWorktree(id) }(id)
	}

	// List should return both
	list := wm.ListWorktrees()
	if len(list) != 2 {
		t.Errorf("expected 2 worktrees, got %d", len(list))
	}
}

func TestWorktreeManagerDuplicateCreate(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "worktrees")

	// Initialize a test git repo
	repoDir := filepath.Join(tmpDir, "repo")
	if err := initTestGitRepo(repoDir); err != nil {
		t.Fatalf("failed to init test repo: %v", err)
	}

	wm, err := NewWorktreeManager(repoDir, baseDir, 5)
	if err != nil {
		t.Fatalf("failed to create worktree manager: %v", err)
	}

	// Create worktree
	wt1, err := wm.CreateWorktree("duplicate-task", "main")
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}
	defer func() { _ = wm.RemoveWorktree("duplicate-task") }()

	// Create again should return existing
	wt2, err := wm.CreateWorktree("duplicate-task", "main")
	if err != nil {
		t.Fatalf("duplicate create returned error: %v", err)
	}

	if wt1.Path != wt2.Path {
		t.Error("duplicate create should return same worktree")
	}

	// Count should still be 1
	if wm.Count() != 1 {
		t.Errorf("expected count 1 after duplicate create, got %d", wm.Count())
	}
}

func TestWorktreeManagerCleanupOrphaned(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "worktrees")

	// Initialize a test git repo
	repoDir := filepath.Join(tmpDir, "repo")
	if err := initTestGitRepo(repoDir); err != nil {
		t.Fatalf("failed to init test repo: %v", err)
	}

	wm, err := NewWorktreeManager(repoDir, baseDir, 5)
	if err != nil {
		t.Fatalf("failed to create worktree manager: %v", err)
	}

	// Create worktree
	wt, err := wm.CreateWorktree("orphan-task", "main")
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Manually remove the directory (simulating orphan)
	if err := os.RemoveAll(wt.Path); err != nil {
		t.Fatalf("failed to remove worktree directory: %v", err)
	}

	// Cleanup should find the orphan
	cleaned, err := wm.CleanupOrphaned()
	if err != nil {
		t.Fatalf("cleanup orphaned failed: %v", err)
	}

	if cleaned != 1 {
		t.Errorf("expected 1 orphan cleaned, got %d", cleaned)
	}

	if wm.Count() != 0 {
		t.Errorf("expected count 0 after cleanup, got %d", wm.Count())
	}
}

// initTestGitRepo initializes a minimal git repository for testing
func initTestGitRepo(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return err
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return err
	}

	// Create initial commit
	testFile := filepath.Join(path, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repo\n"), 0644); err != nil {
		return err
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = path
	return cmd.Run()
}

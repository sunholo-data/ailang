package coordinator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Worktree represents a git worktree for task isolation
type Worktree struct {
	TaskID    string
	Path      string
	Branch    string
	CreatedAt time.Time
}

// WorktreeManager manages git worktrees for isolated task execution
type WorktreeManager struct {
	baseDir      string // Base directory for worktrees
	repoDir      string // Main repository directory
	maxWorktrees int
	worktrees    map[string]*Worktree
	mu           sync.RWMutex
}

// NewWorktreeManager creates a new worktree manager
func NewWorktreeManager(repoDir, baseDir string, maxWorktrees int) (*WorktreeManager, error) {
	if repoDir == "" {
		// Try to find git root
		var err error
		repoDir, err = findGitRoot()
		if err != nil {
			return nil, fmt.Errorf("failed to find git root: %w", err)
		}
	}

	if baseDir == "" {
		baseDir = filepath.Join(os.TempDir(), "ailang_worktrees")
	}

	if maxWorktrees <= 0 {
		maxWorktrees = 3
	}

	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create worktree base directory: %w", err)
	}

	wm := &WorktreeManager{
		baseDir:      baseDir,
		repoDir:      repoDir,
		maxWorktrees: maxWorktrees,
		worktrees:    make(map[string]*Worktree),
	}

	// Load existing worktrees
	if err := wm.loadExisting(); err != nil {
		// Non-fatal, just log
		fmt.Fprintf(os.Stderr, "warning: failed to load existing worktrees: %v\n", err)
	}

	return wm, nil
}

// CreateWorktree creates a new worktree for a task
func (wm *WorktreeManager) CreateWorktree(taskID string) (*Worktree, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Check if already exists
	if wt, ok := wm.worktrees[taskID]; ok {
		return wt, nil
	}

	// Check max limit
	if len(wm.worktrees) >= wm.maxWorktrees {
		return nil, fmt.Errorf("max worktrees limit reached (%d)", wm.maxWorktrees)
	}

	// Create branch name and path
	branchName := fmt.Sprintf("coordinator/%s", sanitizeTaskID(taskID))
	worktreePath := filepath.Join(wm.baseDir, sanitizeTaskID(taskID))

	// Create the worktree with a new branch
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath)
	cmd.Dir = wm.repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w\nOutput: %s", err, output)
	}

	wt := &Worktree{
		TaskID:    taskID,
		Path:      worktreePath,
		Branch:    branchName,
		CreatedAt: time.Now(),
	}

	wm.worktrees[taskID] = wt
	return wt, nil
}

// RemoveWorktree removes a worktree
func (wm *WorktreeManager) RemoveWorktree(taskID string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wt, ok := wm.worktrees[taskID]
	if !ok {
		return fmt.Errorf("worktree not found for task: %s", taskID)
	}

	// Remove the worktree
	cmd := exec.Command("git", "worktree", "remove", "--force", wt.Path)
	cmd.Dir = wm.repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w\nOutput: %s", err, output)
	}

	// Delete the branch
	cmd = exec.Command("git", "branch", "-D", wt.Branch)
	cmd.Dir = wm.repoDir
	_ = cmd.Run() // Ignore errors - branch might already be deleted

	delete(wm.worktrees, taskID)
	return nil
}

// GetWorktree returns a worktree by task ID
func (wm *WorktreeManager) GetWorktree(taskID string) (*Worktree, bool) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	wt, ok := wm.worktrees[taskID]
	return wt, ok
}

// ListWorktrees returns all active worktrees
func (wm *WorktreeManager) ListWorktrees() []*Worktree {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	worktrees := make([]*Worktree, 0, len(wm.worktrees))
	for _, wt := range wm.worktrees {
		worktrees = append(worktrees, wt)
	}
	return worktrees
}

// Count returns the number of active worktrees
func (wm *WorktreeManager) Count() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return len(wm.worktrees)
}

// CleanupOrphaned removes worktrees that no longer exist on disk
func (wm *WorktreeManager) CleanupOrphaned() (int, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Run git worktree prune
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = wm.repoDir
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("failed to prune worktrees: %w", err)
	}

	// Check which of our tracked worktrees still exist
	cleaned := 0
	for taskID, wt := range wm.worktrees {
		if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
			delete(wm.worktrees, taskID)
			cleaned++
		}
	}

	return cleaned, nil
}

// CleanupAll removes all worktrees
func (wm *WorktreeManager) CleanupAll() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	var lastErr error
	for taskID, wt := range wm.worktrees {
		cmd := exec.Command("git", "worktree", "remove", "--force", wt.Path)
		cmd.Dir = wm.repoDir
		if err := cmd.Run(); err != nil {
			lastErr = err
		}

		cmd = exec.Command("git", "branch", "-D", wt.Branch)
		cmd.Dir = wm.repoDir
		_ = cmd.Run()

		delete(wm.worktrees, taskID)
	}

	return lastErr
}

// loadExisting loads existing worktrees from git
func (wm *WorktreeManager) loadExisting() error {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = wm.repoDir
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	// Parse porcelain output
	lines := strings.Split(string(output), "\n")
	var currentPath, currentBranch string

	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") {
			currentBranch = strings.TrimPrefix(line, "branch refs/heads/")

			// Check if it's a coordinator worktree
			if strings.HasPrefix(currentBranch, "coordinator/") && currentPath != wm.repoDir {
				taskID := strings.TrimPrefix(currentBranch, "coordinator/")
				wm.worktrees[taskID] = &Worktree{
					TaskID:    taskID,
					Path:      currentPath,
					Branch:    currentBranch,
					CreatedAt: time.Now(), // Unknown, use current time
				}
			}
			currentPath = ""
			// Note: currentBranch is overwritten on next "branch" line
		}
	}

	return nil
}

// findGitRoot finds the root of the current git repository
func findGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// HasChanges checks if a worktree has uncommitted changes
func (wm *WorktreeManager) HasChanges(taskID string) (bool, error) {
	wm.mu.RLock()
	wt, ok := wm.worktrees[taskID]
	wm.mu.RUnlock()

	if !ok {
		return false, fmt.Errorf("worktree not found for task: %s", taskID)
	}

	// Check for uncommitted changes using git status
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = wt.Path
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	// If output is non-empty, there are changes
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// GetChangeSummary returns a summary of changes in the worktree
func (wm *WorktreeManager) GetChangeSummary(taskID string) (*WorktreeChanges, error) {
	wm.mu.RLock()
	wt, ok := wm.worktrees[taskID]
	wm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("worktree not found for task: %s", taskID)
	}

	changes := &WorktreeChanges{
		TaskID: taskID,
		Path:   wt.Path,
		Branch: wt.Branch,
	}

	// Get list of changed files
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = wt.Path
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) > 3 {
			changes.FilesChanged = append(changes.FilesChanged, strings.TrimSpace(line[3:]))
		}
	}

	// Get diff summary
	cmd = exec.Command("git", "diff", "--stat", "HEAD")
	cmd.Dir = wt.Path
	diffOutput, _ := cmd.Output() // Ignore errors
	changes.DiffSummary = strings.TrimSpace(string(diffOutput))

	// Get commit count ahead of origin
	cmd = exec.Command("git", "rev-list", "--count", "origin/dev..HEAD")
	cmd.Dir = wt.Path
	countOutput, _ := cmd.Output()
	if count := strings.TrimSpace(string(countOutput)); count != "" {
		changes.CommitsAhead = count
	}

	return changes, nil
}

// WorktreeChanges describes the changes in a worktree
type WorktreeChanges struct {
	TaskID       string   `json:"task_id"`
	Path         string   `json:"path"`
	Branch       string   `json:"branch"`
	FilesChanged []string `json:"files_changed"`
	DiffSummary  string   `json:"diff_summary"`
	CommitsAhead string   `json:"commits_ahead"`
}

// sanitizeTaskID makes a task ID safe for use as a directory/branch name
func sanitizeTaskID(taskID string) string {
	// Replace unsafe characters
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, taskID)

	// Limit length
	if len(safe) > 50 {
		safe = safe[:50]
	}

	return safe
}

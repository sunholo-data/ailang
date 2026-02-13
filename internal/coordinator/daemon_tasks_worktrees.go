package coordinator

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// syncWorktreeState syncs worktree manager in-memory state with actual git worktrees.
// This handles cases where worktrees are removed externally (e.g., via CLI rejection/cleanup).
func (d *Daemon) syncWorktreeState() {
	for agentID, wm := range d.worktreeManagers {
		cleaned, err := wm.CleanupOrphaned()
		if err != nil {
			d.logger.Printf("Warning: Failed to sync worktrees for agent %s: %v", agentID, err)
		} else if cleaned > 0 {
			d.logger.Printf("Synced worktrees for agent %s: removed %d orphaned slot(s)", agentID, cleaned)
		}
	}
}

// cleanupWorktreesForTerminalTasks removes worktrees for tasks in terminal states.
// This is needed because RecoverStaleTasks marks tasks as cancelled but doesn't
// clean up their worktrees, leading to "max worktrees limit reached" errors.
// It handles two cases:
// 1. Tasks with WorktreePath set in database - clean up by task ID
// 2. Orphaned worktree directories on disk for terminal tasks (WorktreePath may be empty)
func (d *Daemon) cleanupWorktreesForTerminalTasks() {
	if d.taskStore == nil {
		return
	}

	terminalStatuses := []TaskStatus{
		TaskStatusCancelled,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusRejected,
	}

	cleanedTotal := 0

	// First pass: clean up by WorktreePath in database
	for _, status := range terminalStatuses {
		tasks, err := d.taskStore.ListTasks(d.ctx, &TaskFilter{Status: []TaskStatus{status}})
		if err != nil {
			continue
		}

		for _, task := range tasks {
			if task.WorktreePath == "" {
				continue
			}

			agentID := task.AgentID
			if agentID == "" {
				agentID = "coordinator"
			}

			if wm, ok := d.worktreeManagers[agentID]; ok {
				if err := wm.RemoveWorktree(task.ID); err == nil {
					cleanedTotal++
					d.logger.Printf("Cleaned up stale worktree for %s task %s", status, task.ID)
				}
			}
		}
	}

	// Second pass: scan worktree directories for orphans
	// This handles cases where WorktreePath was never set (task crashed early)
	for agentID, wm := range d.worktreeManagers {
		// Get base directory for this agent's worktrees
		baseDir := filepath.Join(d.config.StateDir, "worktrees", agentID)
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			taskID := entry.Name()

			// Check if this task exists and is in a terminal state
			task, err := d.taskStore.GetTask(d.ctx, taskID)
			if err != nil {
				// Task doesn't exist in DB - this is truly orphaned, clean it up
				if err := wm.RemoveWorktree(taskID); err == nil {
					cleanedTotal++
					d.logger.Printf("Cleaned up orphaned worktree (no task record) %s/%s", agentID, taskID)
				}
				continue
			}

			// Check if task is in terminal state
			isTerminal := false
			for _, ts := range terminalStatuses {
				if task.Status == ts {
					isTerminal = true
					break
				}
			}

			if isTerminal {
				if err := wm.RemoveWorktree(taskID); err == nil {
					cleanedTotal++
					d.logger.Printf("Cleaned up orphaned worktree for %s task %s/%s", task.Status, agentID, taskID)
				}
			}
		}
	}

	if cleanedTotal > 0 {
		d.logger.Printf("Cleaned up %d stale worktree(s) for terminal tasks", cleanedTotal)
	}
}

// autoCommitWorktree commits any uncommitted changes in the worktree.
// This ensures agent work is captured even if the agent forgot to commit.
func autoCommitWorktree(worktreePath, taskTitle string, logger *log.Logger) error {
	// Check for uncommitted changes (including untracked files)
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = worktreePath
	output, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}

	// No changes to commit
	if strings.TrimSpace(string(output)) == "" {
		logger.Printf("No uncommitted changes in worktree: %s", worktreePath)
		return nil
	}

	logger.Printf("Auto-committing changes in worktree: %s", worktreePath)
	logger.Printf("Changes:\n%s", string(output))

	// Add all changes (including untracked files)
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = worktreePath
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Create commit message
	commitMsg := fmt.Sprintf("Auto-commit: %s\n\nCommitted by AILANG coordinator after agent completion.\n\n🤖 Generated with Claude Code", taskTitle)

	// Commit the changes
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = worktreePath
	commitOutput, err := commitCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %w\nOutput: %s", err, string(commitOutput))
	}

	logger.Printf("Auto-commit successful: %s", strings.TrimSpace(string(commitOutput)))
	return nil
}

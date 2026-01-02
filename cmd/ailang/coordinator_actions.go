package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
)

func coordinatorApprove(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ailang coordinator approve <task-id>")
	}

	taskID := args[0]
	stateDir := ""
	skipMerge := false
	keepWorktree := false

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		case "--skip-merge":
			skipMerge = true
		case "--keep-worktree":
			keepWorktree = true
		case "--help", "-h":
			printCoordinatorApproveHelp()
			return nil
		}
	}

	cfg := coordinator.DefaultConfig()
	if stateDir != "" {
		cfg.StateDir = stateDir
	}

	// Open the coordinator database
	dbPath := filepath.Join(cfg.StateDir, "coordinator.db")
	store, err := coordinator.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open coordinator database: %w", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Get the task to verify status and get worktree path
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Verify task is pending approval
	if task.Status != coordinator.TaskStatusPendingApproval {
		return fmt.Errorf("task %s is not pending approval (status: %s)", taskID, task.Status)
	}

	// Resolve the approval request in database
	if err := store.ResolveApprovalRequestByTask(ctx, taskID, "approved", "cli-user"); err != nil {
		return fmt.Errorf("failed to approve task: %w", err)
	}

	fmt.Println(green("✓"), "Task approved:", taskID)

	// Skip merge if requested or no worktree
	if skipMerge {
		fmt.Println(yellow("!"), "Merge skipped (--skip-merge)")
		return nil
	}

	if task.WorktreePath == "" {
		fmt.Println(yellow("!"), "No worktree path - nothing to merge")
		return nil
	}

	// Check worktree exists
	if _, err := os.Stat(task.WorktreePath); os.IsNotExist(err) {
		fmt.Println(yellow("!"), "Worktree no longer exists:", task.WorktreePath)
		return nil
	}

	// Auto-commit any uncommitted changes in the worktree
	if err := autoCommitWorktreeChanges(task.WorktreePath, task.Title); err != nil {
		fmt.Println(yellow("!"), "Warning: Failed to auto-commit:", err)
		// Continue anyway - user can manually commit
	}

	// Perform the merge
	fmt.Println(cyan("→"), "Merging changes to dev branch...")
	fmt.Printf("  Worktree: %s\n", task.WorktreePath)

	mergeResult, err := coordinator.MergeWorktree(ctx, task.WorktreePath, "dev")
	if err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	if !mergeResult.Success {
		if len(mergeResult.ConflictFiles) > 0 {
			fmt.Println(red("✗"), "Merge conflicts detected:")
			for _, f := range mergeResult.ConflictFiles {
				fmt.Printf("    - %s\n", f)
			}
			fmt.Println()
			fmt.Println("Resolve conflicts manually in the worktree, then retry approval.")
			return fmt.Errorf("merge conflicts: %v", mergeResult.ConflictFiles)
		}
		return fmt.Errorf("merge failed: %s", mergeResult.Error)
	}

	// Update task status to completed
	if err := store.MarkTaskCompleted(ctx, taskID, &coordinator.ExecuteResult{
		Success: true,
		Output:  fmt.Sprintf("Merged to dev (commit: %s)", mergeResult.CommitHash),
	}); err != nil {
		fmt.Println(yellow("!"), "Warning: Failed to update task status:", err)
	}

	fmt.Println(green("✓"), "Changes merged successfully")
	fmt.Printf("  Commit: %s\n", mergeResult.CommitHash)
	if len(mergeResult.MergedFiles) > 0 {
		fmt.Printf("  Files: %s\n", strings.Join(mergeResult.MergedFiles, ", "))
	}

	// Clean up worktree after successful merge (unless --keep-worktree)
	if keepWorktree {
		fmt.Println()
		fmt.Printf("Worktree preserved at: %s\n", task.WorktreePath)
		fmt.Println("To remove: git worktree remove", task.WorktreePath)
	} else {
		// Get the branch name before removing worktree
		branchCmd := exec.Command("git", "-C", task.WorktreePath, "rev-parse", "--abbrev-ref", "HEAD")
		branchOutput, _ := branchCmd.Output()
		branchName := strings.TrimSpace(string(branchOutput))

		// Remove the worktree
		fmt.Println(cyan("→"), "Cleaning up worktree...")
		removeCmd := exec.Command("git", "worktree", "remove", task.WorktreePath, "--force")
		if output, err := removeCmd.CombinedOutput(); err != nil {
			fmt.Println(yellow("!"), "Warning: Failed to remove worktree:", string(output))
		} else {
			fmt.Println(green("✓"), "Worktree removed")
		}

		// Also delete the branch
		if branchName != "" && branchName != "HEAD" {
			deleteCmd := exec.Command("git", "branch", "-D", branchName)
			if _, err := deleteCmd.CombinedOutput(); err == nil {
				fmt.Printf("  Deleted branch: %s\n", branchName)
			}
		}
	}

	return nil
}

func coordinatorReject(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ailang coordinator reject <task-id>")
	}

	taskID := args[0]
	stateDir := ""

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		case "--help", "-h":
			printCoordinatorRejectHelp()
			return nil
		}
	}

	cfg := coordinator.DefaultConfig()
	if stateDir != "" {
		cfg.StateDir = stateDir
	}

	// Open the coordinator database to resolve approval
	dbPath := filepath.Join(cfg.StateDir, "coordinator.db")
	store, err := coordinator.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open coordinator database: %w", err)
	}
	defer store.Close()

	// Resolve the approval request in database
	ctx := context.Background()
	if err := store.ResolveApprovalRequestByTask(ctx, taskID, "rejected", "cli-user"); err != nil {
		return fmt.Errorf("failed to reject task: %w", err)
	}

	fmt.Println(green("✓"), "Task rejected:", taskID)
	return nil
}

func coordinatorReopen(args []string) error {
	// Check for help first
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			fmt.Println("Usage: ailang coordinator reopen <task-id> [options]")
			fmt.Println("")
			fmt.Println("Reopen a rejected or cancelled task for re-approval.")
			fmt.Println("Useful if you accidentally rejected a task.")
			fmt.Println("")
			fmt.Println("Options:")
			fmt.Println("  --state-dir DIR  Use custom state directory")
			return nil
		}
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: ailang coordinator reopen <task-id>")
	}

	taskID := args[0]
	stateDir := ""

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		}
	}

	cfg := coordinator.DefaultConfig()
	if stateDir != "" {
		cfg.StateDir = stateDir
	}

	// Open the coordinator database
	dbPath := filepath.Join(cfg.StateDir, "coordinator.db")
	store, err := coordinator.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open coordinator database: %w", err)
	}
	defer store.Close()

	// Reopen the task
	ctx := context.Background()
	if err := store.ReopenTask(ctx, taskID); err != nil {
		return fmt.Errorf("failed to reopen task: %w", err)
	}

	fmt.Println(green("✓"), "Task reopened:", taskID)
	fmt.Println("  Use 'ailang coordinator pending' to see it in the approval queue")
	return nil
}

func coordinatorRetry(args []string) error {
	// Check for help first
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			fmt.Println("Usage: ailang coordinator retry <task-id|--all> [options]")
			fmt.Println("")
			fmt.Println("Reset failed tasks to pending so they will be retried.")
			fmt.Println("")
			fmt.Println("Options:")
			fmt.Println("  --all             Retry all failed tasks")
			fmt.Println("  --state-dir DIR   Use custom state directory")
			fmt.Println("")
			fmt.Println("Examples:")
			fmt.Println("  ailang coordinator retry task-123    # Retry specific task")
			fmt.Println("  ailang coordinator retry --all       # Retry all failed tasks")
			return nil
		}
	}

	stateDir := ""
	retryAll := false
	var taskID string

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			retryAll = true
		case "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		default:
			if taskID == "" && !strings.HasPrefix(args[i], "-") {
				taskID = args[i]
			}
		}
	}

	if !retryAll && taskID == "" {
		return fmt.Errorf("usage: ailang coordinator retry <task-id|--all>")
	}

	cfg := coordinator.DefaultConfig()
	if stateDir != "" {
		cfg.StateDir = stateDir
	}

	// Open the coordinator database
	dbPath := filepath.Join(cfg.StateDir, "coordinator.db")
	store, err := coordinator.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open coordinator database: %w", err)
	}
	defer store.Close()

	ctx := context.Background()

	if retryAll {
		// Reset all failed tasks
		count, err := store.RetryAllFailedTasks(ctx)
		if err != nil {
			return fmt.Errorf("failed to retry failed tasks: %w", err)
		}
		if count == 0 {
			fmt.Println(green("✓"), "No failed tasks to retry")
		} else {
			fmt.Printf("%s Reset %d failed task(s) to pending\n", green("✓"), count)
		}
		return nil
	}

	// Reset specific task
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.Status != coordinator.TaskStatusFailed {
		return fmt.Errorf("task %s is not failed (status: %s)", taskID, task.Status)
	}

	// Reset to pending
	if err := store.RequeueTask(ctx, taskID); err != nil {
		return fmt.Errorf("failed to retry task: %w", err)
	}

	fmt.Println(green("✓"), "Task reset to pending:", taskID)
	fmt.Println("  It will be picked up on the next poll cycle.")
	return nil
}

func coordinatorCleanup(args []string) error {
	stateDir := ""
	staleThreshold := 5 * time.Minute // Default: 5 minutes
	dryRun := false

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		case "--threshold":
			if i+1 < len(args) {
				dur, err := time.ParseDuration(args[i+1])
				if err != nil {
					return fmt.Errorf("invalid threshold: %w", err)
				}
				staleThreshold = dur
				i++
			}
		case "--dry-run":
			dryRun = true
		case "--help", "-h":
			printCoordinatorCleanupHelp()
			return nil
		}
	}

	cfg := coordinator.DefaultConfig()
	if stateDir != "" {
		cfg.StateDir = stateDir
	}

	// Open the coordinator database
	dbPath := filepath.Join(cfg.StateDir, "coordinator.db")
	store, err := coordinator.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open coordinator database: %w", err)
	}
	defer store.Close()

	ctx := context.Background()

	if dryRun {
		// Just show what would be cleaned up
		fmt.Printf("Dry run: would cancel tasks older than %s\n", staleThreshold)
		// TODO: Add a method to list stale tasks without cancelling
		return nil
	}

	// Cancel stale tasks
	count, err := store.RecoverStaleTasks(ctx, staleThreshold)
	if err != nil {
		return fmt.Errorf("failed to cleanup stale tasks: %w", err)
	}

	if count == 0 {
		fmt.Println(green("✓"), "No stale tasks to clean up")
	} else {
		fmt.Printf("%s Cleaned up %d stale task(s)\n", green("✓"), count)
	}
	return nil
}

func printCoordinatorApproveHelp() {
	fmt.Println("Usage: ailang coordinator approve <task-id> [options]")
	fmt.Println("")
	fmt.Println("Approve a pending task and merge its changes to the dev branch.")
	fmt.Println("Automatically cleans up the worktree after merge.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --skip-merge      Approve without merging (mark as approved only)")
	fmt.Println("  --keep-worktree   Don't remove worktree after merge")
	fmt.Println("  --state-dir DIR   State directory (default: ~/.ailang/state)")
	fmt.Println("  --help, -h        Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator approve task-123")
	fmt.Println("  ailang coordinator approve task-123 --keep-worktree")
}

func printCoordinatorRejectHelp() {
	fmt.Println("Usage: ailang coordinator reject <task-id> [options]")
	fmt.Println("")
	fmt.Println("Reject a pending task")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --state-dir DIR   State directory (default: ~/.ailang/state)")
	fmt.Println("  --help, -h        Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator reject task-123")
}

func printCoordinatorCleanupHelp() {
	fmt.Println("Usage: ailang coordinator cleanup [options]")
	fmt.Println("")
	fmt.Println("Cancel stale running/queued tasks from previous daemon runs")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --threshold DURATION   Tasks older than this are considered stale (default: 5m)")
	fmt.Println("  --state-dir DIR        State directory (default: ~/.ailang/state)")
	fmt.Println("  --dry-run              Show what would be cleaned without doing it")
	fmt.Println("  --help, -h             Show this help message")
	fmt.Println("")
	fmt.Println("Note: The daemon automatically runs this on startup.")
}

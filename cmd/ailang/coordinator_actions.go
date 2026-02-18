package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/observatory"
)

func coordinatorApprove(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ailang coordinator approve <task-id|approval-id>")
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

	// Load agent registry for per-agent merge branch lookup
	agentRegistry, _ := coordinator.LoadAgentRegistry()

	// Create GitHubPoster for label operations (M-GENERIC-PIPELINE)
	var githubPoster *coordinator.GitHubPoster
	if poster, err := coordinator.NewGitHubPoster(); err == nil {
		githubPoster = poster
	}
	// If poster creation fails, continue without it - labels won't be updated

	// Open observatory database for chain status updates (M-CHAINS-SIMPLIFY)
	obsDBPath := filepath.Join(cfg.StateDir, "observatory.db")
	obsBackend, _ := observatory.NewSQLiteBackendFromPath(obsDBPath)
	if obsBackend != nil {
		defer obsBackend.Close()
	}
	// If observatory fails to open, continue without it - chain status won't be updated

	// Open messaging store for handoff messages (M-CHAINS-SIMPLIFY)
	var msgStore messaging.MessageStore
	if ms, err := messaging.OpenStore(messaging.GetDefaultDatabasePath()); err == nil {
		msgStore = ms
		defer msgStore.Close()
	}
	// If messaging store fails to open, continue without it - handoffs won't trigger

	ctx := context.Background()

	// Use unified approval processor
	// Note: MergeBranch is resolved by processor from AgentRegistry or defaults
	result, err := coordinator.ProcessApprovalRequest(ctx, &coordinator.ApprovalParams{
		TaskID:        taskID,
		Action:        "approve",
		ApprovedBy:    "cli-user",
		Channel:       "cli",
		SkipMerge:     skipMerge,
		KeepWorktree:  keepWorktree,
		Store:         store,
		AgentRegistry: agentRegistry,
		GitHubPoster:  githubPoster,
		ObsBackend:    obsBackend,
		MsgStore:      msgStore,
	})
	if err != nil {
		return err
	}

	// Display result
	if result.Success {
		fmt.Println(green("✓"), result.Message)
		if result.MergeCommit != "" {
			fmt.Printf("  Commit: %s\n", result.MergeCommit)
		}
		if len(result.MergedFiles) > 0 {
			fmt.Printf("  Files: %s\n", strings.Join(result.MergedFiles, ", "))
		}
	} else {
		if len(result.ConflictFiles) > 0 {
			fmt.Println(red("✗"), "Merge conflicts detected:")
			for _, f := range result.ConflictFiles {
				fmt.Printf("    - %s\n", f)
			}
			fmt.Println()
			fmt.Println("Resolve conflicts manually in the worktree, then retry approval.")
		} else {
			fmt.Println(red("✗"), result.Error)
		}
		return fmt.Errorf("approval failed: %s", result.Error)
	}

	return nil
}

func coordinatorReject(args []string) error {
	// Check for help flag first
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printCoordinatorRejectHelp()
			return nil
		}
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: ailang coordinator reject <task-id|approval-id>")
	}

	taskID := args[0]
	stateDir := ""
	feedbackText := ""
	skipPrompt := false
	noRetrigger := false

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		case "--feedback", "-f":
			if i+1 < len(args) {
				feedbackText = args[i+1]
				i++
			}
		case "--no-prompt":
			skipPrompt = true
		case "--no-retrigger":
			noRetrigger = true
		case "--help", "-h":
			printCoordinatorRejectHelp()
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

	// Prompt for feedback if not provided and not skipped
	if feedbackText == "" && !skipPrompt {
		fmt.Print(yellow("📝"), " Feedback for agent (why was this rejected?): ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read feedback: %w", err)
		}
		feedbackText = strings.TrimSpace(input)
	}

	// Open messaging store for feedback loop
	var msgStore messaging.MessageStore
	if !noRetrigger {
		msgStore, err = messaging.OpenStore(messaging.GetDefaultDatabasePath())
		if err != nil {
			fmt.Println(yellow("!"), "Warning: Could not open messaging store:", err)
		} else {
			defer msgStore.Close()
		}
	}

	// Create GitHub poster for issue updates
	var githubPoster *coordinator.GitHubPoster
	poster, err := coordinator.NewGitHubPoster()
	if err == nil {
		githubPoster = poster
	}

	// Load agent registry for per-agent merge branch lookup
	agentRegistry, _ := coordinator.LoadAgentRegistry()

	// Open observatory database for chain status updates (M-CHAINS-SIMPLIFY)
	obsDBPath := filepath.Join(cfg.StateDir, "observatory.db")
	obsBackend, _ := observatory.NewSQLiteBackendFromPath(obsDBPath)
	if obsBackend != nil {
		defer obsBackend.Close()
	}
	// If observatory fails to open, continue without it - chain status won't be updated

	ctx := context.Background()

	// Use unified approval processor
	result, err := coordinator.ProcessApprovalRequest(ctx, &coordinator.ApprovalParams{
		TaskID:            taskID,
		Action:            "reject",
		ApprovedBy:        "cli-user",
		Channel:           "cli",
		Feedback:          feedbackText,
		RetriggerOnReject: !noRetrigger,
		Store:             store,
		MsgStore:          msgStore,
		GitHubPoster:      githubPoster,
		AgentRegistry:     agentRegistry,
		ObsBackend:        obsBackend,
	})
	if err != nil {
		return err
	}

	// Display result
	fmt.Println(green("✓"), result.Message)
	if result.NewTaskID != "" {
		fmt.Println(cyan("→"), "Session will resume with context preserved (--resume)")
	}

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
	fmt.Println("Reject a pending task with feedback and optional re-trigger.")
	fmt.Println("")
	fmt.Println("By default, rejection prompts for feedback and re-queues the task")
	fmt.Println("for another attempt (up to 3 iterations). The agent will receive")
	fmt.Println("the feedback and use --resume to continue with full context.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --feedback, -f TEXT   Provide feedback directly (skip prompt)")
	fmt.Println("  --no-prompt           Skip feedback prompt (use default message)")
	fmt.Println("  --no-retrigger        Permanently reject without re-triggering")
	fmt.Println("  --state-dir DIR       State directory (default: ~/.ailang/state)")
	fmt.Println("  --help, -h            Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator reject task-123")
	fmt.Println("  ailang coordinator reject task-123 -f 'Need better error handling'")
	fmt.Println("  ailang coordinator reject task-123 --no-retrigger  # permanent rejection")
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

// coordinatorSyncThreads syncs thread target_agent from coordinator tasks to collaboration.db
// This repairs data inconsistencies where tasks were re-routed but threads weren't updated.
func coordinatorSyncThreads(args []string) error {
	stateDir := ""
	dryRun := false

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		case "--dry-run":
			dryRun = true
		case "--help", "-h":
			printCoordinatorSyncThreadsHelp()
			return nil
		}
	}

	cfg := coordinator.DefaultConfig()
	if stateDir != "" {
		cfg.StateDir = stateDir
	}

	// Open coordinator database
	dbPath := filepath.Join(cfg.StateDir, "coordinator.db")
	store, err := coordinator.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open coordinator database: %w", err)
	}
	defer store.Close()

	// Open collaboration database
	collabPath := filepath.Join(cfg.StateDir, "collaboration.db")
	msgStore, err := messaging.OpenStore(collabPath)
	if err != nil {
		return fmt.Errorf("failed to open collaboration database: %w", err)
	}
	defer msgStore.Close()

	ctx := context.Background()

	// Get all tasks
	filter := &coordinator.TaskFilter{Limit: 1000}
	tasks, err := store.ListTasks(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	synced := 0
	for _, task := range tasks {
		if task.ThreadID == "" || task.AgentID == "" {
			continue
		}

		// Get the thread
		thread, err := msgStore.GetThread(task.ThreadID)
		if err != nil || thread == nil {
			continue
		}

		// Check if they match
		if thread.TargetAgent != task.AgentID {
			if dryRun {
				fmt.Printf("Would sync thread %s: %q -> %q (task: %s)\n",
					task.ThreadID, thread.TargetAgent, task.AgentID, task.Title)
			} else {
				if err := msgStore.SetThreadTargetAgent(task.ThreadID, task.AgentID); err != nil {
					fmt.Printf("Failed to sync %s: %v\n", task.ThreadID, err)
				} else {
					fmt.Printf("Synced thread %s: %q -> %q\n",
						task.ThreadID, thread.TargetAgent, task.AgentID)
				}
			}
			synced++
		}
	}

	if synced == 0 {
		fmt.Println(green("✓"), "All threads are in sync")
	} else if dryRun {
		fmt.Printf("Would sync %d thread(s). Run without --dry-run to apply.\n", synced)
	} else {
		fmt.Printf("%s Synced %d thread(s)\n", green("✓"), synced)
	}

	return nil
}

func printCoordinatorSyncThreadsHelp() {
	fmt.Println("Usage: ailang coordinator sync-threads [options]")
	fmt.Println("")
	fmt.Println("Sync thread target_agent from coordinator tasks to collaboration.db")
	fmt.Println("This repairs data inconsistencies where tasks were re-routed but threads weren't updated.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --dry-run              Show what would be changed without making changes")
	fmt.Println("  --state-dir DIR        State directory (default: ~/.ailang/state)")
	fmt.Println("  --help, -h             Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator sync-threads --dry-run")
	fmt.Println("  ailang coordinator sync-threads")
}

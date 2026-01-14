package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/messaging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func coordinatorApprove(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ailang coordinator approve <task-id|approval-id>")
	}

	taskID := args[0]

	// Accept both task IDs (task-xxx) and approval IDs (apr-xxx)
	// Convert approval ID to task ID if needed
	if strings.HasPrefix(taskID, "apr-") {
		taskID = "task-" + strings.TrimPrefix(taskID, "apr-")
	}
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

	// Create OTEL span for human approval (M-TRANSCRIPT, M-DASHBOARD-APPROVAL-INTEGRATION)
	tracer := otel.Tracer("coordinator.cli")
	ctx, span := tracer.Start(ctx, "approval.decision",
		trace.WithAttributes(
			attribute.String("task.id", taskID),
			attribute.String("approval.action", "approve"),
			attribute.String("approval.channel", "cli"),
			attribute.String("approval.by", "cli-user"),
		),
	)
	defer span.End()

	// Get the task to verify status and get worktree path
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Add task iteration to span (M-TRANSCRIPT)
	span.SetAttributes(attribute.Int("task.iteration", task.Iteration))

	// Verify task is pending approval
	if task.Status != coordinator.TaskStatusPendingApproval {
		return fmt.Errorf("task %s is not pending approval (status: %s)", taskID, task.Status)
	}

	// Store approval event for audit trail (M-TRANSCRIPT)
	if err := coordinator.StoreApprovalEvent(ctx, store, taskID, "cli-user"); err != nil {
		// Log but don't fail - approval can still proceed
		fmt.Println(yellow("!"), "Warning: Failed to store approval event:", err)
	}

	// Resolve the approval request in database
	if err := store.ResolveApprovalRequestByTask(ctx, taskID, "approved", "cli-user"); err != nil {
		span.RecordError(err)
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

	// Accept both task IDs (task-xxx) and approval IDs (apr-xxx)
	// Convert approval ID to task ID if needed
	if strings.HasPrefix(taskID, "apr-") {
		taskID = "task-" + strings.TrimPrefix(taskID, "apr-")
	}

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

	// Open the coordinator database to resolve approval
	dbPath := filepath.Join(cfg.StateDir, "coordinator.db")
	store, err := coordinator.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open coordinator database: %w", err)
	}
	defer store.Close()

	// Use enhanced rejection with feedback (unless --no-retrigger specified)
	if noRetrigger {
		// Legacy behavior: immediate rejection without feedback loop
		ctx := context.Background()
		if err := store.ResolveApprovalRequestByTask(ctx, taskID, "rejected", "cli-user"); err != nil {
			return fmt.Errorf("failed to reject task: %w", err)
		}

		fmt.Println(green("✓"), "Task rejected:", taskID)

		// Clean up worktree if it exists
		task, err := store.GetTask(ctx, taskID)
		if err == nil && task != nil && task.WorktreePath != "" {
			if _, statErr := os.Stat(task.WorktreePath); statErr == nil {
				branchCmd := exec.Command("git", "-C", task.WorktreePath, "rev-parse", "--abbrev-ref", "HEAD")
				branchOutput, _ := branchCmd.Output()
				branchName := strings.TrimSpace(string(branchOutput))

				fmt.Println(cyan("→"), "Cleaning up worktree...")
				removeCmd := exec.Command("git", "worktree", "remove", task.WorktreePath, "--force")
				if output, err := removeCmd.CombinedOutput(); err != nil {
					fmt.Println(yellow("!"), "Warning: Failed to remove worktree:", string(output))
				} else {
					fmt.Println(green("✓"), "Worktree removed")
				}

				if branchName != "" && branchName != "HEAD" {
					deleteCmd := exec.Command("git", "branch", "-D", branchName)
					if _, err := deleteCmd.CombinedOutput(); err == nil {
						fmt.Printf("  Deleted branch: %s\n", branchName)
					}
				}
			}
		}
		return nil
	}

	// Enhanced rejection with feedback and re-trigger support
	return coordinatorRejectWithFeedback(store, taskID, skipPrompt, feedbackText)
}

// coordinatorRejectWithFeedback handles rejection with feedback, message sending, and re-triggering.
// This implements the M-TRANSCRIPT feedback loop:
// 1. Prompts for feedback reason
// 2. Stores feedback as human_feedback event
// 3. Sends message to agent inbox
// 4. Re-triggers task with iteration+1 (if under MaxIterations)
func coordinatorRejectWithFeedback(store *coordinator.SQLiteStore, taskID string, skipPrompt bool, feedbackText string) error {
	ctx := context.Background()

	// Get the task
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Create OTEL span for rejection decision (M-DASHBOARD-APPROVAL-INTEGRATION)
	tracer := otel.Tracer("coordinator.cli")
	ctx, span := tracer.Start(ctx, "approval.decision",
		trace.WithAttributes(
			attribute.String("task.id", taskID),
			attribute.String("approval.action", "reject"),
			attribute.String("approval.channel", "cli"),
			attribute.String("approval.by", "cli-user"),
			attribute.Int("task.iteration", task.Iteration),
		),
	)
	defer span.End()

	// Prompt for feedback if not provided
	if feedbackText == "" && !skipPrompt {
		fmt.Print(yellow("📝"), " Feedback for agent (why was this rejected?): ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read feedback: %w", err)
		}
		feedbackText = strings.TrimSpace(input)
	}

	// If still no feedback after prompting, use default
	if feedbackText == "" {
		feedbackText = "Rejected without specific feedback"
	}

	// Store feedback as human_feedback event
	feedback := &coordinator.HumanFeedback{
		TaskID:    taskID,
		Iteration: task.Iteration,
		Feedback:  feedbackText,
		Action:    "reject",
		Timestamp: time.Now(),
		UserID:    "cli-user",
	}
	if err := coordinator.StoreFeedbackEvent(ctx, store, feedback); err != nil {
		fmt.Println(yellow("!"), "Warning: Failed to store feedback event:", err)
		// Continue anyway
	}

	// Post feedback to GitHub if task has a linked issue (M-DASHBOARD-APPROVAL-INTEGRATION)
	if task.GithubIssue > 0 {
		iteration := task.Iteration
		if iteration < 1 {
			iteration = 1
		}
		poster, err := coordinator.NewGitHubPoster() // Uses default repo from config
		if err != nil {
			fmt.Println(yellow("!"), "Warning: Could not create GitHub poster:", err)
		} else {
			if err := poster.PostFeedback(task.GithubIssue, feedbackText, iteration, "cli"); err != nil {
				fmt.Println(yellow("!"), fmt.Sprintf("Warning: Failed to post feedback to GitHub issue #%d: %v", task.GithubIssue, err))
			} else {
				fmt.Println(green("✓"), fmt.Sprintf("Feedback posted to GitHub issue #%d", task.GithubIssue))
			}
		}
	}

	// Check if we can re-trigger (under max iterations)
	canRetrigger := coordinator.CanRetrigger(task)

	if canRetrigger {
		// Prepare task for re-trigger
		coordinator.PrepareTaskForRetrigger(task, feedbackText)

		// Store iteration start event
		if err := coordinator.StoreIterationStartEvent(ctx, store, taskID, task.Iteration); err != nil {
			fmt.Println(yellow("!"), "Warning: Failed to store iteration start:", err)
		}

		// Update task in database
		if err := store.UpdateTask(ctx, task); err != nil {
			return fmt.Errorf("failed to update task for re-trigger: %w", err)
		}

		// Send feedback message to agent inbox
		msgStore, err := messaging.OpenStore(messaging.GetDefaultDatabasePath())
		if err != nil {
			fmt.Println(yellow("!"), "Warning: Could not send message to agent:", err)
		} else {
			defer msgStore.Close()

			agentInbox := task.AgentID
			if agentInbox == "" {
				agentInbox = "coordinator" // Default fallback
			}

			msg := &messaging.InboxMessage{
				FromAgent:     "cli-user",
				ToInbox:       agentInbox,
				MessageType:   messaging.InboxTypeNotification,
				Title:         fmt.Sprintf("Feedback: %s (iteration %d)", truncateString(task.Title, 30), task.Iteration),
				Payload:       fmt.Sprintf("Task rejected with feedback:\n\n%s\n\nTask will be re-run with this feedback incorporated.", feedbackText),
				CorrelationID: taskID, // Link to original task
			}
			if err := msgStore.InsertInboxMessage(msg); err != nil {
				fmt.Println(yellow("!"), "Warning: Failed to send message:", err)
			} else {
				fmt.Println(green("✓"), "Feedback sent to agent inbox:", agentInbox)
			}
		}

		fmt.Println(green("✓"), fmt.Sprintf("Task re-queued for iteration %d (max: %d)", task.Iteration, coordinator.MaxIterations))
		fmt.Println(cyan("→"), "Session will resume with context preserved (--resume)")

	} else {
		// Max iterations reached - permanent rejection
		if err := store.ResolveApprovalRequestByTask(ctx, taskID, "rejected", "cli-user"); err != nil {
			return fmt.Errorf("failed to reject task: %w", err)
		}

		// Update task status
		task.Status = coordinator.TaskStatusRejected
		if err := store.UpdateTask(ctx, task); err != nil {
			fmt.Println(yellow("!"), "Warning: Failed to update task status:", err)
		}

		fmt.Println(yellow("!"), fmt.Sprintf("Max iterations (%d) reached - task permanently rejected", coordinator.MaxIterations))

		// Clean up worktree
		if task.WorktreePath != "" {
			if _, statErr := os.Stat(task.WorktreePath); statErr == nil {
				fmt.Println(cyan("→"), "Cleaning up worktree...")
				removeCmd := exec.Command("git", "worktree", "remove", task.WorktreePath, "--force")
				if output, err := removeCmd.CombinedOutput(); err != nil {
					fmt.Println(yellow("!"), "Warning: Failed to remove worktree:", string(output))
				} else {
					fmt.Println(green("✓"), "Worktree removed")
				}
			}
		}
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

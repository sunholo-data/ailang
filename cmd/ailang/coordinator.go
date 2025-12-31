package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
)

func coordinatorCommand(args []string) error {
	if len(args) == 0 {
		printCoordinatorHelp()
		return nil
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "start":
		return coordinatorStart(subargs)
	case "stop":
		return coordinatorStop(subargs)
	case "status":
		return coordinatorStatus(subargs)
	case "pending":
		return coordinatorPending(subargs)
	case "list":
		return coordinatorList(subargs)
	case "approve":
		return coordinatorApprove(subargs)
	case "reject":
		return coordinatorReject(subargs)
	case "reopen":
		return coordinatorReopen(subargs)
	case "diff":
		return coordinatorDiff(subargs)
	case "logs":
		return coordinatorLogs(subargs)
	case "cleanup":
		return coordinatorCleanup(subargs)
	case "help", "--help", "-h":
		printCoordinatorHelp()
		return nil
	default:
		return fmt.Errorf("unknown coordinator subcommand: %s", subcommand)
	}
}

func coordinatorStart(args []string) error {
	cfg := coordinator.DefaultConfig()

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--poll-interval":
			if i+1 < len(args) {
				duration, err := time.ParseDuration(args[i+1])
				if err != nil {
					return fmt.Errorf("invalid poll-interval: %w", err)
				}
				cfg.PollInterval = duration
				i++
			}
		case "--max-worktrees":
			if i+1 < len(args) {
				n, err := strconv.Atoi(args[i+1])
				if err != nil {
					return fmt.Errorf("invalid max-worktrees: %w", err)
				}
				cfg.MaxWorktrees = n
				i++
			}
		case "--state-dir":
			if i+1 < len(args) {
				cfg.StateDir = args[i+1]
				cfg.PIDFile = filepath.Join(args[i+1], "coordinator.pid")
				i++
			}
		case "--log-file":
			if i+1 < len(args) {
				cfg.LogFile = args[i+1]
				i++
			}
		case "--help", "-h":
			printCoordinatorStartHelp()
			return nil
		}
	}

	// Create daemon
	daemon, err := coordinator.NewDaemon(cfg)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// Check if already running
	status, _ := daemon.Status()
	if status.Running {
		fmt.Println(yellow("⚠"), "Coordinator is already running")
		fmt.Printf("  PID: %d\n", status.PID)
		return nil
	}

	// Start daemon
	fmt.Println(cyan("→"), "Starting coordinator daemon...")
	fmt.Printf("  PID file: %s\n", cfg.PIDFile)
	fmt.Printf("  Log file: %s\n", cfg.LogFile)
	fmt.Printf("  Poll interval: %s\n", cfg.PollInterval)
	fmt.Printf("  Max worktrees: %d\n", cfg.MaxWorktrees)

	if err := daemon.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	fmt.Println(green("✓"), "Coordinator daemon started")
	return nil
}

func coordinatorStop(args []string) error {
	cfg := coordinator.DefaultConfig()

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				cfg.StateDir = args[i+1]
				cfg.PIDFile = filepath.Join(args[i+1], "coordinator.pid")
				i++
			}
		case "--help", "-h":
			printCoordinatorStopHelp()
			return nil
		}
	}

	daemon, err := coordinator.NewDaemon(cfg)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// Check if running
	status, _ := daemon.Status()
	if !status.Running {
		fmt.Println(yellow("⚠"), "Coordinator is not running")
		return nil
	}

	// Stop daemon
	fmt.Println(cyan("→"), "Stopping coordinator daemon...")
	if err := daemon.Stop(); err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}

	fmt.Println(green("✓"), "Coordinator daemon stopped")
	return nil
}

func coordinatorStatus(args []string) error {
	cfg := coordinator.DefaultConfig()
	jsonOutput := false
	watchMode := false

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				cfg.StateDir = args[i+1]
				cfg.PIDFile = filepath.Join(args[i+1], "coordinator.pid")
				i++
			}
		case "--json":
			jsonOutput = true
		case "--watch", "-w":
			watchMode = true
		case "--help", "-h":
			printCoordinatorStatusHelp()
			return nil
		}
	}

	daemon, err := coordinator.NewDaemon(cfg)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	if watchMode {
		return coordinatorStatusWatch(daemon)
	}

	status, err := daemon.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(status)
	}

	printCoordinatorStatusOutput(status)
	return nil
}

// coordinatorStatusWatch shows live status updates
func coordinatorStatusWatch(daemon *coordinator.Daemon) error {
	fmt.Println(bold("Coordinator Status (live, press Ctrl+C to exit)"))
	fmt.Println()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Print initial status
	status, _ := daemon.Status()
	printCoordinatorStatusOutput(status)

	for range ticker.C {
		// Clear screen and move cursor to top
		fmt.Print("\033[2J\033[H")

		fmt.Println(bold("Coordinator Status (live, press Ctrl+C to exit)"))
		fmt.Println()

		status, err := daemon.Status()
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		printCoordinatorStatusOutput(status)
	}

	return nil
}

// printCoordinatorStatusOutput prints the status in human-readable format
func printCoordinatorStatusOutput(status *coordinator.Status) {
	if status.Running {
		fmt.Printf("  State:      %s %s\n", "▶", green("running"))
		fmt.Printf("  PID:        %d\n", status.PID)
		if !status.StartedAt.IsZero() {
			fmt.Printf("  Started:    %s\n", status.StartedAt.Format("2006-01-02 15:04:05"))
		}
		if status.Uptime != "" {
			fmt.Printf("  Uptime:     %s\n", status.Uptime)
		}
	} else {
		fmt.Printf("  State:      %s %s\n", "⏹", red("stopped"))
	}

	fmt.Println()
	fmt.Println(bold("Task Statistics"))
	fmt.Printf("  Completed:  %s\n", green(fmt.Sprintf("%d", status.TasksRun)))
	if status.PendingTasks > 0 {
		fmt.Printf("  Pending:    %s\n", yellow(fmt.Sprintf("%d", status.PendingTasks)))
	}
	if status.RunningTasks > 0 {
		fmt.Printf("  Running:    %s\n", cyan(fmt.Sprintf("%d", status.RunningTasks)))
	}
	if status.PendingApprovals > 0 {
		fmt.Printf("  Approvals:  %s %s\n", yellow(fmt.Sprintf("%d", status.PendingApprovals)), "(use 'ailang coordinator pending' to review)")
	}
	if status.FailedTasks > 0 {
		fmt.Printf("  Failed:     %s\n", red(fmt.Sprintf("%d", status.FailedTasks)))
	}
	if status.TotalCost > 0 {
		fmt.Printf("  Total Cost: $%.4f\n", status.TotalCost)
	}
	if status.TotalTokens > 0 {
		fmt.Printf("  Tokens:     %d\n", status.TotalTokens)
	}
}

func printCoordinatorHelp() {
	fmt.Println("Usage: ailang coordinator <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  start     Start the coordinator daemon")
	fmt.Println("  stop      Stop the coordinator daemon")
	fmt.Println("  status    Show coordinator status (summary)")
	fmt.Println("  list      List all tasks (with filters)")
	fmt.Println("  pending   List tasks awaiting approval (interactive)")
	fmt.Println("  diff      Show changes made by a task (git diff)")
	fmt.Println("  logs      Show streaming logs/events for a task")
	fmt.Println("  approve   Approve a pending task")
	fmt.Println("  reject    Reject a pending task")
	fmt.Println("  reopen    Reopen a rejected/cancelled task for re-approval")
	fmt.Println("  cleanup   Cancel stale running/queued tasks")
	fmt.Println("  help      Show this help message")
	fmt.Println("")
	fmt.Println("The coordinator daemon watches for incoming messages and executes tasks")
	fmt.Println("using Claude Code or Gemini in isolated git worktrees.")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator start")
	fmt.Println("  ailang coordinator start --poll-interval 60s --max-worktrees 2")
	fmt.Println("  ailang coordinator status --json")
	fmt.Println("  ailang coordinator list --running      # See running tasks")
	fmt.Println("  ailang coordinator list --pending      # See pending tasks")
	fmt.Println("  ailang coordinator pending             # Interactive approval queue")
	fmt.Println("  ailang coordinator approve task-abc123")
	fmt.Println("  ailang coordinator reject task-abc123")
	fmt.Println("  ailang coordinator stop")
}

func printCoordinatorStartHelp() {
	fmt.Println("Usage: ailang coordinator start [options]")
	fmt.Println("")
	fmt.Println("Start the coordinator daemon")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --poll-interval DURATION   How often to check for new messages (default: 30s)")
	fmt.Println("  --max-worktrees N          Maximum concurrent worktrees (default: 3)")
	fmt.Println("  --state-dir DIR            State directory (default: ~/.ailang/state)")
	fmt.Println("  --log-file PATH            Log file path (default: ~/.ailang/logs/coordinator.log)")
	fmt.Println("  --help, -h                 Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator start")
	fmt.Println("  ailang coordinator start --poll-interval 60s")
	fmt.Println("  ailang coordinator start --max-worktrees 5")
}

func printCoordinatorStopHelp() {
	fmt.Println("Usage: ailang coordinator stop [options]")
	fmt.Println("")
	fmt.Println("Stop the coordinator daemon")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --state-dir DIR   State directory (default: ~/.ailang/state)")
	fmt.Println("  --help, -h        Show this help message")
}

func printCoordinatorStatusHelp() {
	fmt.Println("Usage: ailang coordinator status [options]")
	fmt.Println("")
	fmt.Println("Show coordinator daemon status")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --state-dir DIR   State directory (default: ~/.ailang/state)")
	fmt.Println("  --json            Output status as JSON")
	fmt.Println("  --watch, -w       Watch mode: continuously update status")
	fmt.Println("  --help, -h        Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator status")
	fmt.Println("  ailang coordinator status --json")
	fmt.Println("  ailang coordinator status --watch")
}

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

func coordinatorDiff(args []string) error {
	// Check for help first
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			fmt.Println("Usage: ailang coordinator diff <task-id> [options]")
			fmt.Println("")
			fmt.Println("Show the git diff for changes made by a task.")
			fmt.Println("Use this to review changes before approving or rejecting.")
			fmt.Println("")
			fmt.Println("Options:")
			fmt.Println("  --stat           Show diffstat only (summary of changes)")
			fmt.Println("  --state-dir DIR  Use custom state directory")
			return nil
		}
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: ailang coordinator diff <task-id>")
	}

	taskID := args[0]
	stateDir := ""
	statOnly := false

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		case "--stat":
			statOnly = true
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

	// Get the task to find its worktree path
	ctx := context.Background()
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task.WorktreePath == "" {
		return fmt.Errorf("task %s has no worktree path (may have been executed without isolation)", taskID)
	}

	// Check if worktree exists
	if _, err := os.Stat(task.WorktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree no longer exists: %s", task.WorktreePath)
	}

	// Show task info
	fmt.Printf("%s %s\n", bold("Task:"), task.ID)
	fmt.Printf("%s %s\n", bold("Title:"), task.Title)
	fmt.Printf("%s %s\n", bold("Status:"), task.Status)
	fmt.Printf("%s %s\n", bold("Worktree:"), task.WorktreePath)
	fmt.Println()

	// Run git diff in the worktree
	var cmd *exec.Cmd
	if statOnly {
		cmd = exec.Command("git", "-C", task.WorktreePath, "diff", "--stat", "HEAD~1")
	} else {
		cmd = exec.Command("git", "-C", task.WorktreePath, "diff", "--color=always", "HEAD~1")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// If HEAD~1 doesn't exist, try diff against origin/dev
		if statOnly {
			cmd = exec.Command("git", "-C", task.WorktreePath, "diff", "--stat", "origin/dev")
		} else {
			cmd = exec.Command("git", "-C", task.WorktreePath, "diff", "--color=always", "origin/dev")
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run git diff: %w", err)
		}
	}

	fmt.Println()
	fmt.Printf("To approve: %s\n", green("ailang coordinator approve "+taskID))
	fmt.Printf("To reject:  %s\n", red("ailang coordinator reject "+taskID))

	return nil
}

func coordinatorLogs(args []string) error {
	// Check for help first
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			fmt.Println("Usage: ailang coordinator logs <task-id> [options]")
			fmt.Println("")
			fmt.Println("Show streaming logs/events for a task (running or completed).")
			fmt.Println("Useful for monitoring in-progress tasks or reviewing what happened.")
			fmt.Println("")
			fmt.Println("Options:")
			fmt.Println("  --limit N        Show last N events (default: 50)")
			fmt.Println("  --json           Output as JSON")
			fmt.Println("  --state-dir DIR  Use custom state directory")
			return nil
		}
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: ailang coordinator logs <task-id>")
	}

	taskID := args[0]
	stateDir := ""
	limit := 50
	jsonOutput := false

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		case "--limit":
			if i+1 < len(args) {
				n, err := strconv.Atoi(args[i+1])
				if err == nil && n > 0 {
					limit = n
				}
				i++
			}
		case "--json":
			jsonOutput = true
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

	// Get task info first
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Get events for the task
	events, err := store.GetTaskEvents(ctx, taskID, limit)
	if err != nil {
		return fmt.Errorf("failed to get task events: %w", err)
	}

	if jsonOutput {
		data := struct {
			Task   *coordinator.TaskRecord        `json:"task"`
			Events []*coordinator.TaskEventRecord `json:"events"`
		}{Task: task, Events: events}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	// Display task info
	fmt.Printf("%s %s\n", bold("Task:"), task.ID)
	fmt.Printf("%s %s\n", bold("Title:"), task.Title)
	fmt.Printf("%s %s\n", bold("Status:"), string(task.Status))
	if task.Provider != "" {
		fmt.Printf("%s %s\n", bold("Provider:"), task.Provider)
	}
	fmt.Println()

	if len(events) == 0 {
		fmt.Println(yellow("No events recorded for this task."))
		fmt.Println("Note: Events are only recorded while the coordinator daemon is running.")
		return nil
	}

	fmt.Printf("%s (showing %d events)\n", bold("Events:"), len(events))
	fmt.Println(strings.Repeat("─", 60))

	for _, event := range events {
		timestamp := event.CreatedAt.Format("15:04:05")

		switch event.StreamType {
		case "turn_start":
			fmt.Printf("%s %s Turn %d started\n", dim(timestamp), blue("◆"), event.TurnNum)
		case "turn_end":
			fmt.Printf("%s %s Turn %d ended\n", dim(timestamp), blue("◇"), event.TurnNum)
		case "text":
			// Truncate long text
			text := event.Text
			if len(text) > 120 {
				text = text[:120] + "..."
			}
			// Remove newlines for cleaner display
			text = strings.ReplaceAll(text, "\n", " ")
			fmt.Printf("%s %s\n", dim(timestamp), text)
		case "tool_use":
			fmt.Printf("%s %s %s\n", dim(timestamp), cyan("🔧"), event.ToolName)
		case "tool_result":
			output := event.ToolOutput
			if len(output) > 80 {
				output = output[:80] + "..."
			}
			fmt.Printf("%s %s %s\n", dim(timestamp), green("→"), output)
		case "error":
			fmt.Printf("%s %s %s\n", dim(timestamp), red("✗"), event.ErrorMsg)
		case "status":
			fmt.Printf("%s %s %s\n", dim(timestamp), yellow("●"), event.Status)
		default:
			if event.Text != "" {
				fmt.Printf("%s %s\n", dim(timestamp), event.Text)
			}
		}
	}

	fmt.Println(strings.Repeat("─", 60))

	if task.Cost > 0 {
		fmt.Printf("%s $%.4f (%d tokens)\n", bold("Cost:"), task.Cost, task.TokensUsed)
	}

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

func coordinatorPending(args []string) error {
	stateDir := ""
	jsonOutput := false

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		case "--json":
			jsonOutput = true
		case "--help", "-h":
			printCoordinatorPendingHelp()
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

	// List pending approvals
	ctx := context.Background()
	pending, err := store.ListPendingApprovals(ctx)
	if err != nil {
		return fmt.Errorf("failed to list pending approvals: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(pending)
	}

	if len(pending) == 0 {
		fmt.Println(green("✓"), "No pending approval requests")
		return nil
	}

	fmt.Println(bold("Pending Approval Requests"))
	fmt.Println()
	for i, req := range pending {
		// Get task details for worktree info
		task, _ := store.GetTask(ctx, req.TaskID)
		fmt.Printf("  %s [%d] %s\n", yellow("⏳"), i+1, req.TaskID)
		fmt.Printf("       Title: %s\n", req.Description)
		if task != nil && task.WorktreePath != "" {
			if _, err := os.Stat(task.WorktreePath); err == nil {
				fmt.Printf("       Worktree: %s\n", task.WorktreePath)
			} else {
				fmt.Printf("       Worktree: %s\n", red("(deleted)"))
			}
		}
		fmt.Printf("       Created: %s\n", req.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	fmt.Println(bold("Actions:"))
	fmt.Println("  [1-" + strconv.Itoa(len(pending)) + "]  Select task number")
	fmt.Println("  [q]    Quit")
	fmt.Println()
	fmt.Print("Select task to review: ")

	// Read user input
	var input string
	fmt.Scanln(&input)

	if input == "" || input == "q" || input == "Q" {
		return nil
	}

	// Parse task number
	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num > len(pending) {
		return fmt.Errorf("invalid selection: %s", input)
	}

	selectedReq := pending[num-1]
	selectedTask, _ := store.GetTask(ctx, selectedReq.TaskID)

	// Show task menu
	fmt.Println()
	fmt.Println(bold("Task: ") + selectedReq.TaskID)
	fmt.Println(bold("Title: ") + selectedReq.Description)
	fmt.Println()
	fmt.Println(bold("Actions:"))
	fmt.Println("  [d]  View diff")
	fmt.Println("  [s]  View diff summary (--stat)")
	fmt.Println("  [a]  Approve")
	fmt.Println("  [r]  Reject")
	fmt.Println("  [q]  Cancel")
	fmt.Println()
	fmt.Print("Action: ")

	fmt.Scanln(&input)

	switch strings.ToLower(input) {
	case "d":
		// Show diff
		if selectedTask == nil || selectedTask.WorktreePath == "" {
			fmt.Println(red("✗"), "No worktree available for this task")
			return nil
		}
		cmd := exec.Command("git", "-C", selectedTask.WorktreePath, "diff", "--color=always", "origin/dev")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		// After showing diff, prompt for action
		fmt.Println()
		fmt.Print("Approve [a], Reject [r], or Cancel [q]: ")
		fmt.Scanln(&input)
		if strings.ToLower(input) == "a" {
			// TODO: implement approve flow
			fmt.Println(yellow("!"), "Use 'ailang coordinator approve", selectedReq.TaskID+"'")
		} else if strings.ToLower(input) == "r" {
			if err := store.ResolveApprovalRequestByTask(ctx, selectedReq.TaskID, "rejected", "cli-user"); err != nil {
				return fmt.Errorf("failed to reject: %w", err)
			}
			fmt.Println(green("✓"), "Task rejected:", selectedReq.TaskID)
		}
	case "s":
		// Show diff stat
		if selectedTask == nil || selectedTask.WorktreePath == "" {
			fmt.Println(red("✗"), "No worktree available for this task")
			return nil
		}
		cmd := exec.Command("git", "-C", selectedTask.WorktreePath, "diff", "--stat", "origin/dev")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	case "a":
		// TODO: implement approve flow (needs merge logic)
		fmt.Println(yellow("!"), "Use 'ailang coordinator approve", selectedReq.TaskID+"'")
	case "r":
		if err := store.ResolveApprovalRequestByTask(ctx, selectedReq.TaskID, "rejected", "cli-user"); err != nil {
			return fmt.Errorf("failed to reject: %w", err)
		}
		fmt.Println(green("✓"), "Task rejected:", selectedReq.TaskID)
	case "q", "":
		return nil
	default:
		fmt.Println("Unknown action:", input)
	}

	return nil
}

func printCoordinatorPendingHelp() {
	fmt.Println("Usage: ailang coordinator pending [options]")
	fmt.Println("")
	fmt.Println("List tasks awaiting human approval")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --state-dir DIR   State directory (default: ~/.ailang/state)")
	fmt.Println("  --json            Output as JSON")
	fmt.Println("  --help, -h        Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator pending")
	fmt.Println("  ailang coordinator pending --json")
}

func coordinatorList(args []string) error {
	stateDir := ""
	jsonOutput := false
	limit := 10
	var statusFilters []coordinator.TaskStatus

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		case "--json":
			jsonOutput = true
		case "--limit":
			if i+1 < len(args) {
				n, err := strconv.Atoi(args[i+1])
				if err == nil && n > 0 {
					limit = n
				}
				i++
			}
		case "--status":
			if i+1 < len(args) {
				for _, s := range strings.Split(args[i+1], ",") {
					statusFilters = append(statusFilters, coordinator.TaskStatus(s))
				}
				i++
			}
		case "--running":
			statusFilters = append(statusFilters, coordinator.TaskStatusRunning)
		case "--pending":
			statusFilters = append(statusFilters, coordinator.TaskStatusPending, coordinator.TaskStatusQueued, coordinator.TaskStatusPendingApproval)
		case "--completed":
			statusFilters = append(statusFilters, coordinator.TaskStatusCompleted)
		case "--failed":
			statusFilters = append(statusFilters, coordinator.TaskStatusFailed, coordinator.TaskStatusRejected, coordinator.TaskStatusCancelled)
		case "--help", "-h":
			printCoordinatorListHelp()
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

	// Build filter
	filter := &coordinator.TaskFilter{
		Limit:     limit,
		OrderBy:   "created_at",
		OrderDesc: true,
	}
	if len(statusFilters) > 0 {
		filter.Status = statusFilters
	}

	tasks, err := store.ListTasks(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tasks)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	fmt.Println(bold("Tasks"))
	fmt.Println()

	// Table header
	fmt.Printf("  %-15s %-12s %-10s %-40s %s\n",
		dim("ID"), dim("STATUS"), dim("TYPE"), dim("TITLE"), dim("CREATED"))
	fmt.Println("  " + strings.Repeat("─", 95))

	for _, task := range tasks {
		statusIcon, statusStr := formatTaskStatus(task.Status)
		title := task.Title
		if len(title) > 38 {
			title = title[:35] + "..."
		}

		// Shorten ID for display (first 12 chars)
		shortID := task.ID
		if len(shortID) > 15 {
			shortID = shortID[:12] + "..."
		}

		created := task.CreatedAt.Format("Jan 02 15:04")

		fmt.Printf("  %-15s %s %-11s %-10s %-40s %s\n",
			shortID, statusIcon, statusStr, task.Type, title, dim(created))

		// Show extra info for certain statuses
		if task.Status == coordinator.TaskStatusRunning && task.Provider != "" {
			fmt.Printf("       %s Provider: %s\n", dim("└"), task.Provider)
		}
		if task.Status == coordinator.TaskStatusFailed && task.Error != "" {
			errMsg := task.Error
			if len(errMsg) > 70 {
				errMsg = errMsg[:67] + "..."
			}
			fmt.Printf("       %s Error: %s\n", dim("└"), red(errMsg))
		}
		if task.Cost > 0 {
			fmt.Printf("       %s Cost: $%.4f (%d tokens)\n", dim("└"), task.Cost, task.TokensUsed)
		}
	}

	fmt.Println()
	fmt.Printf("Showing %d task(s). Use --limit N to see more.\n", len(tasks))
	fmt.Println()

	// Interactive mode - select a task to explore
	fmt.Println(bold("Actions:"))
	fmt.Println("  [1-" + strconv.Itoa(len(tasks)) + "]  Select task number to explore")
	fmt.Println("  [q]      Quit")
	fmt.Println()
	fmt.Print("Select task (or press Enter to quit): ")

	var input string
	fmt.Scanln(&input)

	if input == "" || input == "q" || input == "Q" {
		return nil
	}

	// Parse task number
	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num > len(tasks) {
		return fmt.Errorf("invalid selection: %s", input)
	}

	selectedTask := tasks[num-1]
	return showTaskDetail(ctx, store, selectedTask)
}

// formatTaskStatus returns an icon and colored status string
func formatTaskStatus(status coordinator.TaskStatus) (string, string) {
	switch status {
	case coordinator.TaskStatusPending:
		return yellow("○"), "pending"
	case coordinator.TaskStatusQueued:
		return yellow("◎"), "queued"
	case coordinator.TaskStatusRunning:
		return cyan("▶"), cyan("running")
	case coordinator.TaskStatusPendingApproval:
		return magenta("⏳"), magenta("approval")
	case coordinator.TaskStatusCompleted:
		return green("✓"), green("completed")
	case coordinator.TaskStatusFailed:
		return red("✗"), red("failed")
	case coordinator.TaskStatusRejected:
		return red("⊘"), red("rejected")
	case coordinator.TaskStatusCancelled:
		return dim("⊗"), dim("cancelled")
	case coordinator.TaskStatusDuplicate:
		return dim("⊜"), dim("duplicate")
	default:
		return "?", string(status)
	}
}

func printCoordinatorListHelp() {
	fmt.Println("Usage: ailang coordinator list [options]")
	fmt.Println("")
	fmt.Println("List all coordinator tasks")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --status STATUS   Filter by status (comma-separated: pending,running,completed)")
	fmt.Println("  --running         Show only running tasks")
	fmt.Println("  --pending         Show pending tasks (includes queued and approval)")
	fmt.Println("  --completed       Show only completed tasks")
	fmt.Println("  --failed          Show failed/rejected/cancelled tasks")
	fmt.Println("  --limit N         Maximum tasks to show (default: 50)")
	fmt.Println("  --json            Output as JSON")
	fmt.Println("  --state-dir DIR   State directory (default: ~/.ailang/state)")
	fmt.Println("  --help, -h        Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator list                    # Show recent tasks")
	fmt.Println("  ailang coordinator list --running          # Show only running tasks")
	fmt.Println("  ailang coordinator list --pending          # Show all pending tasks")
	fmt.Println("  ailang coordinator list --status running,pending_approval")
	fmt.Println("  ailang coordinator list --limit 100 --json # JSON output")
}

// showTaskDetail shows detailed information about a task with interactive options
func showTaskDetail(ctx context.Context, store *coordinator.SQLiteStore, task *coordinator.TaskRecord) error {
	for {
		// Clear and show task details
		fmt.Println()
		fmt.Println(strings.Repeat("═", 70))
		fmt.Printf("%s %s\n", bold("Task:"), task.ID)
		fmt.Printf("%s %s\n", bold("Title:"), task.Title)
		statusIcon, statusStr := formatTaskStatus(task.Status)
		fmt.Printf("%s %s %s\n", bold("Status:"), statusIcon, statusStr)
		fmt.Printf("%s %s\n", bold("Type:"), task.Type)
		fmt.Printf("%s %s\n", bold("Created:"), task.CreatedAt.Format("2006-01-02 15:04:05"))
		if task.Provider != "" {
			fmt.Printf("%s %s\n", bold("Provider:"), task.Provider)
		}
		if task.Cost > 0 {
			fmt.Printf("%s $%.4f (%d tokens)\n", bold("Cost:"), task.Cost, task.TokensUsed)
		}
		if task.WorktreePath != "" {
			if _, err := os.Stat(task.WorktreePath); err == nil {
				fmt.Printf("%s %s\n", bold("Worktree:"), task.WorktreePath)
			} else {
				fmt.Printf("%s %s\n", bold("Worktree:"), red("(deleted)"))
			}
		}
		if task.Error != "" {
			fmt.Printf("%s %s\n", bold("Error:"), red(task.Error))
		}
		fmt.Println(strings.Repeat("─", 70))
		fmt.Println()

		// Show available actions based on task state
		fmt.Println(bold("Actions:"))
		hasWorktree := task.WorktreePath != "" && fileExists(task.WorktreePath)

		if hasWorktree {
			fmt.Println("  [d]  View diff (full)")
			fmt.Println("  [s]  View diff summary (--stat)")
			fmt.Println("  [f]  Browse files changed")
			fmt.Println("  [b]  Browse worktree directory")
		}
		fmt.Println("  [l]  View execution logs")
		if task.Status == coordinator.TaskStatusPendingApproval {
			fmt.Println("  [a]  " + green("Approve and merge"))
			fmt.Println("  [r]  " + red("Reject"))
		}
		fmt.Println("  [q]  Back to list")
		fmt.Println()
		fmt.Print("Action: ")

		var input string
		fmt.Scanln(&input)

		switch strings.ToLower(input) {
		case "d":
			if !hasWorktree {
				fmt.Println(red("✗"), "No worktree available")
				continue
			}
			showWorktreeDiff(task.WorktreePath, false)

		case "s":
			if !hasWorktree {
				fmt.Println(red("✗"), "No worktree available")
				continue
			}
			showWorktreeDiff(task.WorktreePath, true)

		case "f":
			if !hasWorktree {
				fmt.Println(red("✗"), "No worktree available")
				continue
			}
			browseChangedFiles(task.WorktreePath)

		case "b":
			if !hasWorktree {
				fmt.Println(red("✗"), "No worktree available")
				continue
			}
			browseWorktreeDirectory(task.WorktreePath, "")

		case "l":
			showTaskLogs(ctx, store, task.ID)

		case "a":
			if task.Status != coordinator.TaskStatusPendingApproval {
				fmt.Println(yellow("!"), "Task is not pending approval")
				continue
			}
			// Call the approve function
			return coordinatorApprove([]string{task.ID})

		case "r":
			if task.Status != coordinator.TaskStatusPendingApproval {
				fmt.Println(yellow("!"), "Task is not pending approval")
				continue
			}
			return coordinatorReject([]string{task.ID})

		case "q", "":
			return nil

		default:
			fmt.Println("Unknown action:", input)
		}
	}
}

// showWorktreeDiff shows the git diff for a worktree
func showWorktreeDiff(worktreePath string, statOnly bool) {
	var cmd *exec.Cmd
	if statOnly {
		cmd = exec.Command("git", "-C", worktreePath, "diff", "--stat", "origin/dev")
	} else {
		cmd = exec.Command("git", "-C", worktreePath, "diff", "--color=always", "origin/dev")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	fmt.Println()
	fmt.Print("Press Enter to continue...")
	fmt.Scanln()
}

// browseChangedFiles shows a list of changed files and lets user view them
func browseChangedFiles(worktreePath string) {
	// Get list of changed files
	cmd := exec.Command("git", "-C", worktreePath, "diff", "--name-status", "origin/dev")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println(red("✗"), "Failed to get changed files:", err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println(yellow("!"), "No changed files found")
		fmt.Print("Press Enter to continue...")
		fmt.Scanln()
		return
	}

	// Parse files with their status
	type changedFile struct {
		status string
		path   string
	}
	var files []changedFile
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			files = append(files, changedFile{
				status: parts[0],
				path:   parts[1],
			})
		}
	}

	for {
		fmt.Println()
		fmt.Println(bold("Changed Files:"))
		fmt.Println()
		for i, f := range files {
			var statusStr string
			switch f.status {
			case "A":
				statusStr = green(f.status)
			case "M":
				statusStr = yellow(f.status)
			case "D":
				statusStr = red(f.status)
			default:
				statusStr = dim(f.status)
			}
			fmt.Printf("  [%d] %s %s\n", i+1, statusStr, f.path)
		}
		fmt.Println()
		fmt.Println("  [q] Back")
		fmt.Println()
		fmt.Print("Select file to view (or q to go back): ")

		var input string
		fmt.Scanln(&input)

		if input == "q" || input == "Q" || input == "" {
			return
		}

		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(files) {
			fmt.Println("Invalid selection")
			continue
		}

		selectedFile := files[num-1]
		showFileDiff(worktreePath, selectedFile.path)
	}
}

// showFileDiff shows the diff for a specific file
func showFileDiff(worktreePath, filePath string) {
	fmt.Println()
	fmt.Println(bold("Diff for:"), filePath)
	fmt.Println(strings.Repeat("─", 60))

	cmd := exec.Command("git", "-C", worktreePath, "diff", "--color=always", "origin/dev", "--", filePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	fmt.Println()
	fmt.Print("Press Enter to continue...")
	fmt.Scanln()
}

// browseWorktreeDirectory lets user browse the worktree directory
func browseWorktreeDirectory(worktreePath, subPath string) {
	currentPath := filepath.Join(worktreePath, subPath)

	for {
		// List directory contents
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			fmt.Println(red("✗"), "Failed to read directory:", err)
			return
		}

		fmt.Println()
		if subPath == "" {
			fmt.Println(bold("Worktree Root:"), worktreePath)
		} else {
			fmt.Println(bold("Directory:"), subPath)
		}
		fmt.Println(strings.Repeat("─", 60))

		// Separate dirs and files
		var dirs, files []os.DirEntry
		for _, e := range entries {
			if e.Name() == ".git" {
				continue // Skip .git
			}
			if e.IsDir() {
				dirs = append(dirs, e)
			} else {
				files = append(files, e)
			}
		}

		// Show directories first
		idx := 1
		entryMap := make(map[int]os.DirEntry)
		for _, d := range dirs {
			fmt.Printf("  [%d] %s/\n", idx, cyan(d.Name()))
			entryMap[idx] = d
			idx++
		}
		// Then files
		for _, f := range files {
			info, _ := f.Info()
			size := ""
			if info != nil {
				size = fmt.Sprintf(" (%d bytes)", info.Size())
			}
			fmt.Printf("  [%d] %s%s\n", idx, f.Name(), dim(size))
			entryMap[idx] = f
			idx++
		}

		fmt.Println()
		if subPath != "" {
			fmt.Println("  [u] Go up")
		}
		fmt.Println("  [q] Back to task")
		fmt.Println()
		fmt.Print("Select entry (or q to go back): ")

		var input string
		fmt.Scanln(&input)

		switch strings.ToLower(input) {
		case "q", "":
			return
		case "u":
			if subPath != "" {
				subPath = filepath.Dir(subPath)
				if subPath == "." {
					subPath = ""
				}
				currentPath = filepath.Join(worktreePath, subPath)
			}
		default:
			num, err := strconv.Atoi(input)
			if err != nil || num < 1 || num >= idx {
				fmt.Println("Invalid selection")
				continue
			}

			entry := entryMap[num]
			entryPath := filepath.Join(subPath, entry.Name())

			if entry.IsDir() {
				subPath = entryPath
				currentPath = filepath.Join(worktreePath, subPath)
			} else {
				// Show file contents
				showFileContents(worktreePath, entryPath)
			}
		}
	}
}

// showFileContents displays the contents of a file
func showFileContents(worktreePath, filePath string) {
	fullPath := filepath.Join(worktreePath, filePath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		fmt.Println(red("✗"), "Failed to read file:", err)
		fmt.Print("Press Enter to continue...")
		fmt.Scanln()
		return
	}

	fmt.Println()
	fmt.Println(bold("File:"), filePath)
	fmt.Println(strings.Repeat("─", 60))

	// Show file contents (limit to reasonable size)
	lines := strings.Split(string(content), "\n")
	maxLines := 50
	if len(lines) > maxLines {
		for i, line := range lines[:maxLines] {
			fmt.Printf("%4d│ %s\n", i+1, line)
		}
		fmt.Printf("\n... and %d more lines (file truncated)\n", len(lines)-maxLines)
	} else {
		for i, line := range lines {
			fmt.Printf("%4d│ %s\n", i+1, line)
		}
	}

	fmt.Println()
	fmt.Print("Press Enter to continue...")
	fmt.Scanln()
}

// showTaskLogs shows the execution logs for a task
func showTaskLogs(ctx context.Context, store *coordinator.SQLiteStore, taskID string) {
	events, err := store.GetTaskEvents(ctx, taskID, 100)
	if err != nil {
		fmt.Println(red("✗"), "Failed to get logs:", err)
		fmt.Print("Press Enter to continue...")
		fmt.Scanln()
		return
	}

	fmt.Println()
	fmt.Println(bold("Execution Logs"))
	fmt.Println(strings.Repeat("─", 60))

	if len(events) == 0 {
		fmt.Println(yellow("No events recorded for this task."))
	} else {
		for _, event := range events {
			timestamp := event.CreatedAt.Format("15:04:05")

			switch event.StreamType {
			case "turn_start":
				fmt.Printf("%s %s Turn %d started\n", dim(timestamp), blue("◆"), event.TurnNum)
			case "turn_end":
				fmt.Printf("%s %s Turn %d ended\n", dim(timestamp), blue("◇"), event.TurnNum)
			case "text":
				text := event.Text
				if len(text) > 100 {
					text = text[:100] + "..."
				}
				text = strings.ReplaceAll(text, "\n", " ")
				fmt.Printf("%s %s\n", dim(timestamp), text)
			case "tool_use":
				fmt.Printf("%s %s %s\n", dim(timestamp), cyan("🔧"), event.ToolName)
			case "tool_result":
				output := event.ToolOutput
				if len(output) > 60 {
					output = output[:60] + "..."
				}
				fmt.Printf("%s %s %s\n", dim(timestamp), green("→"), output)
			case "error":
				fmt.Printf("%s %s %s\n", dim(timestamp), red("✗"), event.ErrorMsg)
			case "status":
				fmt.Printf("%s %s %s\n", dim(timestamp), yellow("●"), event.Status)
			default:
				if event.Text != "" {
					fmt.Printf("%s %s\n", dim(timestamp), event.Text)
				}
			}
		}
	}

	fmt.Println()
	fmt.Print("Press Enter to continue...")
	fmt.Scanln()
}

// fileExists checks if a file/directory exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// autoCommitWorktreeChanges checks if there are uncommitted changes in the worktree
// and commits them automatically. This handles the case where the agent creates
// files but doesn't commit them.
func autoCommitWorktreeChanges(worktreePath, taskTitle string) error {
	// Check for uncommitted changes (untracked or modified)
	statusCmd := exec.Command("git", "-C", worktreePath, "status", "--porcelain")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	// If no changes, nothing to do
	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		return nil
	}

	fmt.Println(cyan("→"), "Auto-committing uncommitted changes in worktree...")

	// Add all changes
	addCmd := exec.Command("git", "-C", worktreePath, "add", "-A")
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add changes: %w\n%s", err, output)
	}

	// Commit with a descriptive message
	commitMsg := fmt.Sprintf("Changes for task: %s\n\nAuto-committed by coordinator on approval.", taskTitle)
	commitCmd := exec.Command("git", "-C", worktreePath, "commit", "-m", commitMsg)
	commitOutput, err := commitCmd.CombinedOutput()
	if err != nil {
		// Check if it's just "nothing to commit"
		if strings.Contains(string(commitOutput), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("failed to commit: %w\n%s", err, commitOutput)
	}

	fmt.Println(green("✓"), "Auto-committed changes")
	return nil
}

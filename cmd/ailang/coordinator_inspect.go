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
		// If HEAD~1 doesn't exist, try diff against origin/dev to HEAD (committed changes)
		if statOnly {
			cmd = exec.Command("git", "-C", task.WorktreePath, "diff", "--stat", "origin/dev", "HEAD")
		} else {
			cmd = exec.Command("git", "-C", task.WorktreePath, "diff", "--color=always", "origin/dev", "HEAD")
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
	fmt.Println("────────────────────────────────────────────────────────────")

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

	fmt.Println("────────────────────────────────────────────────────────────")

	if task.Cost > 0 {
		fmt.Printf("%s $%.4f (%d tokens)\n", bold("Cost:"), task.Cost, task.TokensUsed)
	}

	return nil
}

func coordinatorWatcherStatus(args []string) error {
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
			printCoordinatorWatcherStatusHelp()
			return nil
		}
	}

	cfg := coordinator.DefaultConfig()
	if stateDir != "" {
		cfg.StateDir = stateDir
	}

	daemon, err := coordinator.NewDaemon(cfg)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// Get watcher status from daemon
	status := daemon.GetWatcherStatus()

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(status)
	}

	// Human-readable output
	fmt.Println(bold("ApprovalWatcher Status"))
	fmt.Println()

	if status.Running {
		fmt.Printf("  State:         %s %s\n", "▶", green("running"))
	} else {
		fmt.Printf("  State:         %s %s\n", "⏹", red("stopped"))
	}

	fmt.Printf("  Poll Interval: %s\n", status.PollInterval)

	if !status.LastPoll.IsZero() {
		ago := time.Since(status.LastPoll).Round(time.Second)
		fmt.Printf("  Last Poll:     %s (%s ago)\n", status.LastPoll.Format("15:04:05"), ago)
	} else {
		fmt.Printf("  Last Poll:     %s\n", dim("never"))
	}

	fmt.Println()
	fmt.Printf("  Watched Issues: %d\n", len(status.WatchedIssues))

	if len(status.WatchedIssues) > 0 {
		fmt.Println()
		for issueNum, taskID := range status.WatchedIssues {
			fmt.Printf("    #%d → %s\n", issueNum, taskID)
		}
	}

	fmt.Println()
	fmt.Println(dim("Tip: Set DEBUG_APPROVAL_WATCHER=1 for verbose polling logs"))

	return nil
}

func printCoordinatorWatcherStatusHelp() {
	fmt.Println("Usage: ailang coordinator watcher-status [options]")
	fmt.Println("")
	fmt.Println("Show the status of the ApprovalWatcher (GitHub label polling)")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --state-dir DIR   State directory (default: ~/.ailang/state)")
	fmt.Println("  --json            Output as JSON")
	fmt.Println("  --help, -h        Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator watcher-status")
	fmt.Println("  ailang coordinator watcher-status --json")
}

func coordinatorWorktree(args []string) error {
	if len(args) == 0 {
		printCoordinatorWorktreeHelp()
		return nil
	}

	// Check for help flag first
	if args[0] == "--help" || args[0] == "-h" {
		printCoordinatorWorktreeHelp()
		return nil
	}

	taskID := args[0]
	stateDir := ""
	openDir := false
	cdShell := false

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		case "--open", "-o":
			openDir = true
		case "--cd":
			cdShell = true
		case "--help", "-h":
			printCoordinatorWorktreeHelp()
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

	// Get the task
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.WorktreePath == "" {
		return fmt.Errorf("no worktree associated with task: %s", taskID)
	}

	// Check if worktree exists
	if _, err := os.Stat(task.WorktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree no longer exists: %s", task.WorktreePath)
	}

	// Output format depends on flags
	if cdShell {
		// Just print the path for shell integration: eval $(ailang coordinator worktree ID --cd)
		fmt.Printf("cd '%s'\n", task.WorktreePath)
		return nil
	}

	if openDir {
		openInFinder(task.WorktreePath)
		return nil
	}

	// Default: just print the path
	fmt.Println(task.WorktreePath)
	return nil
}

func printCoordinatorWorktreeHelp() {
	fmt.Println("Usage: ailang coordinator worktree <task-id> [options]")
	fmt.Println("")
	fmt.Println("Show or open the worktree directory for a task")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --open, -o        Open worktree in file manager (Finder)")
	fmt.Println("  --cd              Output shell command to cd into worktree")
	fmt.Println("  --state-dir DIR   State directory (default: ~/.ailang/state)")
	fmt.Println("  --help, -h        Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator worktree task-abc123")
	fmt.Println("  ailang coordinator worktree task-abc123 --open")
	fmt.Println("  cd $(ailang coordinator worktree task-abc123)")
}

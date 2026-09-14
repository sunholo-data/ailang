package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/display"
)

func coordinatorList(args []string) error {
	stateDir := ""
	remote := ""
	jsonOutput := false
	limit := 10
	var statusFilters []coordinator.TaskStatus

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--remote":
			if i+1 < len(args) {
				remote = args[i+1]
				i++
			}
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

	// The plane this command answers about.
	//
	// It used to open a hardcoded local SQLite path and accept --remote without
	// reading it, so asking for production returned a stale local task from May
	// 2026 — a confident wrong answer, which is worse than an error (measured
	// 2026-09-14 while waiting on a live cloud task).
	//
	// openCoordinatorStore is the resolver approve/reject/approvals already use.
	// Sharing it is what stops the two halves of this CLI disagreeing about
	// which coordinator exists.
	ctx := context.Background()
	bundle, err := openCoordinatorStore(ctx, remote, stateDir)
	if err != nil {
		return err
	}
	defer bundle.Close()
	store := bundle.Store
	localStore, isLocal := store.(*coordinator.SQLiteStore)

	// Name the plane on every listing. Two planes' task lists are
	// indistinguishable by content, and reading one while believing it is the
	// other is the whole failure class.
	if !jsonOutput {
		fmt.Printf("store: %s\n", bundle.Mode)
	}

	// Build filter
	filter := &coordinator.TaskFilter{
		Limit:     limit,
		OrderBy:   "created_at",
		OrderDesc: true,
	}
	if len(statusFilters) > 0 {
		filter.Status = statusFilters
	}

	// JSON output mode - no interactive loop
	if jsonOutput {
		tasks, err := store.ListTasks(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to list tasks: %w", err)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tasks)
	}

	// Interactive loop - show list, select task, return to list
	for {
		tasks, err := store.ListTasks(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to list tasks: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		fmt.Println()
		fmt.Println(bold("Tasks"))
		fmt.Println()

		// Table header
		fmt.Printf("  %-15s %-12s %-10s %-40s %s\n",
			dim("ID"), dim("STATUS"), dim("TYPE"), dim("TITLE"), dim("CREATED"))
		fmt.Println("  " + strings.Repeat("─", 95))

		for i, task := range tasks {
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

			// Show number prefix for selection
			fmt.Printf("%2d %-15s %s %-11s %-10s %-40s %s\n",
				i+1, shortID, statusIcon, statusStr, task.Type, title, dim(created))

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

		// Interactive drill-down needs the concrete SQLite store (worktree
		// diffs, chat history, handoff approval all take *SQLiteStore). Rather
		// than degrade to a local answer under a remote flag — the exact bug
		// above — say so and name the command that does work remotely.
		if !isLocal {
			fmt.Println(dim("  (listing only: interactive drill-down is local-only on this store)"))
			fmt.Println("  Remote equivalents:")
			fmt.Println("    ailang coordinator approvals --remote " + firstWord(bundle.Mode) + " [--full]")
			fmt.Println("    ailang coordinator list --remote " + firstWord(bundle.Mode) + " --json")
			return nil
		}

		// Interactive mode - select a task to explore
		fmt.Print("Select task [1-" + strconv.Itoa(len(tasks)) + "] or [q]uit: ")

		var input string
		fmt.Scanln(&input)

		if input == "" || input == "q" || input == "Q" {
			return nil
		}

		// Parse task number
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(tasks) {
			fmt.Println(red("Invalid selection:"), input)
			continue
		}

		selectedTask := tasks[num-1]
		if err := showTaskDetail(ctx, localStore, selectedTask); err != nil {
			fmt.Println(red("Error:"), err)
		}
		// Loop back to show list again
	}
}

// formatTaskStatus returns an icon and colored status string.
// Uses shared display.StatusDisplay for consistency with API responses.
func formatTaskStatus(status coordinator.TaskStatus) (string, string) {
	sd := display.TaskStatusDisplay(string(status))
	// Apply terminal colors based on severity
	coloredLabel := sd.Label
	switch sd.Severity {
	case display.SeveritySuccess:
		coloredLabel = green(sd.Label)
	case display.SeverityError:
		coloredLabel = red(sd.Label)
	case display.SeverityWarning:
		coloredLabel = magenta(sd.Label)
	case display.SeverityInfo:
		if string(status) == "running" {
			coloredLabel = cyan(sd.Label)
		} else {
			coloredLabel = yellow(sd.Label)
		}
	case display.SeverityMuted:
		coloredLabel = dim(sd.Label)
	}
	return sd.Icon, coloredLabel
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
			fmt.Println("  [o]  Open worktree in Finder")
		}
		fmt.Println("  [l]  View execution logs")
		fmt.Println("  [c]  View chat history")
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

		case "o":
			if !hasWorktree {
				fmt.Println(red("✗"), "No worktree available")
				continue
			}
			openInFinder(task.WorktreePath)

		case "l":
			showTaskLogs(ctx, store, task)

		case "c":
			showTaskChatHistory(store, task.ID)

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

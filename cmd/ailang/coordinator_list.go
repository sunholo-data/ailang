package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/messaging"
)

func coordinatorPending(args []string) error {
	stateDir := ""
	jsonOutput := false
	approveAll := false
	approveID := ""

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
		case "--approve-all":
			approveAll = true
		case "--approve":
			if i+1 < len(args) {
				approveID = args[i+1]
				i++
			}
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

	ctx := context.Background()

	// JSON output mode - no interactive loop
	if jsonOutput {
		pending, err := store.ListPendingApprovals(ctx)
		if err != nil {
			return fmt.Errorf("failed to list pending approvals: %w", err)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(pending)
	}

	// Non-interactive mode: --approve <id> or --approve-all
	if approveID != "" || approveAll {
		pending, err := store.ListPendingApprovals(ctx)
		if err != nil {
			return fmt.Errorf("failed to list pending approvals: %w", err)
		}

		if len(pending) == 0 {
			fmt.Println(green("✓"), "No pending approval requests")
			return nil
		}

		for _, req := range pending {
			// If --approve <id>, only approve matching ID
			if approveID != "" && req.ID != approveID && req.TaskID != approveID {
				// Also check if approveID is a prefix (like apr-xxx matches task-xxx)
				if !strings.HasPrefix(req.ID, approveID) && !strings.HasPrefix(req.TaskID, approveID) {
					continue
				}
			}

			if req.Type == "handoff" {
				// Handle handoff approval
				if err := coordinatorApproveHandoff(store, req); err != nil {
					fmt.Println(red("✗"), "Failed to approve handoff:", err)
				} else {
					fmt.Println(green("✓"), "Handoff approved:", req.ID)
				}
			} else {
				// Handle merge approval - use the approve command
				if err := coordinatorApprove([]string{req.TaskID}); err != nil {
					fmt.Println(red("✗"), "Failed to approve:", req.TaskID, err)
				}
			}

			// If single ID mode, we're done
			if approveID != "" {
				return nil
			}
		}

		fmt.Println(green("✓"), "All pending approvals processed")
		return nil
	}

	// OUTER LOOP: Task selection - allows returning to list after actions
taskList:
	for {
		// Refresh pending list each iteration
		pending, err := store.ListPendingApprovals(ctx)
		if err != nil {
			return fmt.Errorf("failed to list pending approvals: %w", err)
		}

		if len(pending) == 0 {
			fmt.Println(green("✓"), "No pending approval requests")
			return nil
		}

		fmt.Println()
		fmt.Println(bold("Pending Approval Requests"))
		fmt.Println()
		for i, req := range pending {
			// Get task details for worktree info
			task, _ := store.GetTask(ctx, req.TaskID)
			// Show type indicator
			typeLabel := "[merge]"
			if req.Type == "handoff" {
				typeLabel = "[handoff]"
			}
			fmt.Printf("  %s [%d] %s %s\n", yellow("⏳"), i+1, cyan(typeLabel), req.TaskID)
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
			fmt.Println(red("Invalid selection:"), input)
			continue // Stay in task list
		}

		selectedReq := pending[num-1]
		selectedTask, _ := store.GetTask(ctx, selectedReq.TaskID)
		isHandoff := selectedReq.Type == "handoff"

		// INNER LOOP: Action menu for selected task
		for {
			// Show task menu
			fmt.Println()
			if isHandoff {
				fmt.Println(bold("Type: ") + cyan("Handoff Approval"))
			} else {
				fmt.Println(bold("Type: ") + cyan("Merge Approval"))
			}
			fmt.Println(bold("Task: ") + selectedReq.TaskID)
			fmt.Println(bold("Title: ") + selectedReq.Description)
			// Show handoff context if available
			if isHandoff && selectedReq.ContextJSON != "" {
				var ctxJSON map[string]interface{}
				if err := json.Unmarshal([]byte(selectedReq.ContextJSON), &ctxJSON); err == nil {
					if src, ok := ctxJSON["source_agent_id"].(string); ok {
						if tgt, ok := ctxJSON["target_agent_id"].(string); ok {
							fmt.Println(bold("Handoff: ") + src + " → " + tgt)
						}
					}
				}
			}
			fmt.Println()
			fmt.Println(bold("Actions:"))
			if !isHandoff {
				fmt.Println("  [d]  View diff (full)")
				fmt.Println("  [s]  View diff summary (--stat)")
				fmt.Println("  [f]  Browse changed files")
				fmt.Println("  [o]  Open worktree in Finder")
				fmt.Println("  [a]  " + green("Approve and merge"))
			} else {
				fmt.Println("  [a]  " + green("Approve handoff (send to next agent)"))
			}
			fmt.Println("  [c]  View chat history")
			fmt.Println("  [r]  " + red("Reject"))
			fmt.Println("  [q]  Back to list")
			fmt.Println()
			fmt.Print("Action: ")

			fmt.Scanln(&input)

			switch strings.ToLower(input) {
			case "d":
				// Show full diff (committed + uncommitted changes)
				if selectedTask == nil || selectedTask.WorktreePath == "" {
					fmt.Println(red("✗"), "No worktree available for this task")
					continue // Stay in action menu
				}
				showWorktreeDiff(selectedTask.WorktreePath, false)
				// Stay in action menu to allow further actions

			case "s":
				// Show diff stat (committed + uncommitted changes)
				if selectedTask == nil || selectedTask.WorktreePath == "" {
					fmt.Println(red("✗"), "No worktree available for this task")
					continue // Stay in action menu
				}
				showWorktreeDiff(selectedTask.WorktreePath, true)
				// Stay in action menu

			case "f":
				// Browse changed files
				if selectedTask == nil || selectedTask.WorktreePath == "" {
					fmt.Println(red("✗"), "No worktree available for this task")
					continue // Stay in action menu
				}
				browseChangedFiles(selectedTask.WorktreePath)
				// Stay in action menu

			case "o":
				// Open worktree in Finder
				if selectedTask == nil || selectedTask.WorktreePath == "" {
					fmt.Println(red("✗"), "No worktree available for this task")
					continue // Stay in action menu
				}
				openInFinder(selectedTask.WorktreePath)
				// Stay in action menu

			case "c":
				// View chat history
				if selectedTask == nil {
					fmt.Println(red("✗"), "No task selected")
					continue
				}
				showTaskChatHistory(store, selectedTask.ID)
				// Stay in action menu

			case "a":
				if isHandoff {
					// Approve handoff - resolve and send to target agent
					return coordinatorApproveHandoff(store, selectedReq)
				}
				// Approve and merge
				return coordinatorApprove([]string{selectedReq.TaskID})

			case "r":
				if isHandoff {
					// Reject handoff by ID (not by task) - no feedback loop for handoffs
					if err := store.ResolveApprovalRequest(ctx, selectedReq.ID, "rejected", "cli-user"); err != nil {
						return fmt.Errorf("failed to reject handoff: %w", err)
					}
					fmt.Println(green("✓"), "Handoff rejected:", selectedReq.ID)
				} else {
					// Use enhanced rejection with feedback loop
					if err := coordinatorRejectWithFeedback(store, selectedReq.TaskID, false, ""); err != nil {
						return fmt.Errorf("failed to reject: %w", err)
					}
				}
				// Return to task list after rejection
				continue taskList

			case "q", "":
				// Return to task list
				continue taskList

			default:
				fmt.Println("Unknown action:", input)
				// Stay in action menu
			}
		}
	}
}

// coordinatorApproveHandoff approves a handoff request and sends the message to the target agent.
func coordinatorApproveHandoff(coordStore *coordinator.SQLiteStore, req *coordinator.ApprovalRequestRecord) error {
	ctx := context.Background()

	// Parse context to get handoff details
	var handoffCtx struct {
		SourceAgentID  string `json:"source_agent_id"`
		TargetAgentID  string `json:"target_agent_id"`
		SessionID      string `json:"session_id"`
		HandoffMessage string `json:"handoff_message"`
	}
	if err := json.Unmarshal([]byte(req.ContextJSON), &handoffCtx); err != nil {
		return fmt.Errorf("failed to parse handoff context: %w", err)
	}

	if handoffCtx.TargetAgentID == "" {
		return fmt.Errorf("handoff missing target_agent_id")
	}

	// Resolve the handoff approval
	if err := coordStore.ResolveApprovalRequest(ctx, req.ID, "approved", "cli-user"); err != nil {
		return fmt.Errorf("failed to resolve handoff approval: %w", err)
	}

	// Get task details for the handoff message
	task, err := coordStore.GetTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Send message to target agent's inbox
	messageContent := handoffCtx.HandoffMessage
	if messageContent == "" {
		messageContent = fmt.Sprintf("**Approved Handoff**\n\nTask: %s\nFrom: %s\n\nThis handoff was approved by a human reviewer.",
			req.TaskID, handoffCtx.SourceAgentID)
	}

	// Build the title
	title := fmt.Sprintf("Handoff: %s (approved)", task.Title)
	if len(title) > 80 {
		title = title[:77] + "..."
	}

	// Open the messages database and send
	msgStore, err := openStore()
	if err != nil {
		return fmt.Errorf("failed to open message store: %w", err)
	}
	defer msgStore.Close()

	// Create the inbox message
	msg := &messaging.InboxMessage{
		FromAgent:    "coordinator",
		ToInbox:      handoffCtx.TargetAgentID,
		MessageType:  messaging.InboxTypeNotification,
		Title:        title,
		Payload:      messageContent,
		ParentTaskID: req.TaskID, // Link to parent task for hierarchy tracking
	}

	if err := msgStore.InsertInboxMessage(msg); err != nil {
		return fmt.Errorf("failed to send handoff message: %w", err)
	}

	fmt.Println(green("✓"), "Handoff approved:", handoffCtx.SourceAgentID, "→", handoffCtx.TargetAgentID)
	fmt.Println("  ", "Message sent to inbox:", handoffCtx.TargetAgentID)

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
		if err := showTaskDetail(ctx, store, selectedTask); err != nil {
			fmt.Println(red("Error:"), err)
		}
		// Loop back to show list again
	}
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

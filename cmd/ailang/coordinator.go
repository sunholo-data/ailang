package main

import (
	"fmt"
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
	case "retry":
		return coordinatorRetry(subargs)
	case "diff":
		return coordinatorDiff(subargs)
	case "logs":
		return coordinatorLogs(subargs)
	case "cleanup":
		return coordinatorCleanup(subargs)
	case "worktree":
		return coordinatorWorktree(subargs)
	case "watcher-status":
		return coordinatorWatcherStatus(subargs)
	case "sync-threads":
		return coordinatorSyncThreads(subargs)
	case "execute-job":
		return coordinatorExecuteJob(subargs)
	case "workers":
		return coordinatorWorkers(subargs)
	case "help", "--help", "-h":
		printCoordinatorHelp()
		return nil
	default:
		return fmt.Errorf("unknown coordinator subcommand: %s", subcommand)
	}
}

func printCoordinatorHelp() {
	fmt.Println("Usage: ailang coordinator <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  start          Start the coordinator daemon")
	fmt.Println("  stop           Stop the coordinator daemon")
	fmt.Println("  status         Show coordinator status (summary)")
	fmt.Println("  watcher-status Show ApprovalWatcher status (GitHub polling)")
	fmt.Println("  list           List all tasks (with filters)")
	fmt.Println("  pending        List tasks awaiting approval (interactive)")
	fmt.Println("  diff           Show changes made by a task (git diff)")
	fmt.Println("  logs           Show streaming logs/events for a task")
	fmt.Println("  worktree       Show/open worktree directory for a task")
	fmt.Println("  approve        Approve a pending task")
	fmt.Println("  reject         Reject a pending task")
	fmt.Println("  reopen         Reopen a rejected/cancelled task for re-approval")
	fmt.Println("  retry          Reset failed tasks to pending for retry")
	fmt.Println("  cleanup        Cancel stale running/queued tasks")
	fmt.Println("  sync-threads   Sync thread agent from coordinator tasks")
	fmt.Println("  execute-job    Execute a task in Cloud Run Job (M-PUBSUB)")
	fmt.Println("  workers        List/probe worker hosts (bare-metal + Cloud Run; M-COORD-MULTI-HOST-WORKERS)")
	fmt.Println("  help           Show this help message")
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
	fmt.Println("  ailang coordinator worktree task-abc --open")
	fmt.Println("  ailang coordinator approve task-abc123")
	fmt.Println("  ailang coordinator reject task-abc123")
	fmt.Println("  ailang coordinator stop")
}

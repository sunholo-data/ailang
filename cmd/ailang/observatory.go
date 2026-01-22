package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// normalizeWorkspacePath converts raw workspace paths to clean, aggregated names.
// - Eval workspaces (.Eval_workspace) -> "Eval"
// - Task worktrees (Worktrees/) -> "Tasks"
// - Regular workspaces -> project name (last meaningful directory)
func normalizeWorkspacePath(path string) string {
	if path == "" || path == "unknown" {
		return "unknown"
	}

	// Check for eval workspace
	if strings.Contains(path, ".Eval_workspace") || strings.Contains(path, ".eval_workspace") {
		return "Eval"
	}

	// Check for coordinator task worktree (case-insensitive)
	lowerPath := strings.ToLower(path)
	if strings.Contains(lowerPath, "/worktrees/") || strings.Contains(lowerPath, "/.ailang/state/worktrees/") {
		return "Tasks"
	}

	// Extract project name from regular workspace path
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if part == "" {
			continue
		}
		// Skip hidden directories
		if strings.HasPrefix(part, ".") {
			continue
		}
		// Skip common non-project directories
		switch part {
		case "Users", "home", "var", "tmp", "temp", "Worktrees":
			continue
		}
		// Skip numeric-looking temp dirs (timestamps)
		if len(part) > 10 && part[0] >= '0' && part[0] <= '9' {
			continue
		}
		// Found a good project name
		return part
	}

	// Fallback to last segment
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

func observatoryCommand() {
	if flag.NArg() < 2 {
		fmt.Println("Usage: ailang observatory <subcommand> [options]")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  hierarchy   View unified task/message/span hierarchy (auto-detects ID type)")
		fmt.Println("  metrics     View session/workspace metrics (LOC, commits, cache savings)")
		fmt.Println("  seed        Generate test data for dashboard development")
		fmt.Println("  backfill    Link existing spans to tasks by time correlation")
		fmt.Println("  cleanup     Delete old/noise spans based on retention policy")
		fmt.Println("  heatmap     Get activity heatmap data (daily aggregates)")
		fmt.Println("  evolution   Get task evolution data (cumulative metrics over time)")
		fmt.Println("  usage       Get usage time series data (bucketed by hour/day/week)")
		fmt.Println("  tokens      Get token distribution histogram")
		fmt.Println("  outliers    Detect statistical outliers in spans within a task")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang observatory hierarchy task-29404032  # View task hierarchy")
		fmt.Println("  ailang observatory hierarchy 0ebf5e64bb...  # View by trace ID")
		fmt.Println("  ailang observatory metrics --session <id>   # Session metrics summary")
		fmt.Println("  ailang observatory metrics --list           # List all OTLP metrics")
		fmt.Println("  ailang observatory seed                     # Default: realistic workload")
		fmt.Println("  ailang observatory seed --minimal           # Quick: 1 workspace, 2 tasks")
		fmt.Println("  ailang observatory cleanup --dry-run        # Preview what would be deleted")
		fmt.Println("  ailang observatory cleanup --vacuum         # Delete and reclaim disk space")
		fmt.Println("  ailang observatory heatmap --days 30 --format ascii")
		fmt.Println("  ailang observatory evolution --metric cost --limit 5")
		fmt.Println("  ailang observatory usage --interval day --split-by provider")
		fmt.Println("  ailang observatory tokens --format ascii")
		fmt.Println("  ailang observatory outliers --task TASK_ID --threshold 2.0")
		return
	}

	subcommand := flag.Arg(1)
	switch subcommand {
	case "hierarchy":
		observatoryHierarchyCommand()
	case "metrics":
		observatoryMetricsCommand()
	case "seed":
		observatorySeedCommand()
	case "backfill":
		observatoryBackfillCommand()
	case "cleanup":
		observatoryCleanupCommand()
	case "heatmap":
		observatoryHeatmapCommand()
	case "evolution":
		observatoryEvolutionCommand()
	case "usage":
		observatoryUsageCommand()
	case "tokens":
		observatoryTokensCommand()
	case "outliers":
		observatoryOutliersCommand()
	default:
		fmt.Fprintf(os.Stderr, "Unknown observatory subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

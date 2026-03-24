package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

func observatoryBackfillCommand() {
	fs := flag.NewFlagSet("observatory backfill", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview what would be linked without making changes")
	window := fs.Duration("window", 5*time.Minute, "Time window for matching spans to tasks")
	taskID := fs.String("task", "", "Only backfill spans for a specific task")
	verbose := fs.Bool("verbose", false, "Show detailed output for each span linked")

	// Skip "ailang observatory backfill" args
	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Get database path from config
	dbPath := observatory.DefaultDatabasePath()

	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	opts := BackfillOptions{
		DryRun:  *dryRun,
		Window:  *window,
		TaskID:  *taskID,
		Verbose: *verbose,
	}

	result, err := runBackfill(ctx, backend, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during backfill: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Println()
	if *dryRun {
		fmt.Println("=== Backfill Preview (dry-run) ===")
	} else {
		fmt.Println("=== Backfill Complete ===")
	}
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("Tasks scanned:       %d\n", result.TasksScanned)
	fmt.Printf("Spans scanned:       %d\n", result.SpansScanned)
	fmt.Printf("Spans already linked:%d\n", result.SpansAlreadyLinked)
	fmt.Printf("Spans linked:        %d\n", result.SpansLinked)
	if result.Errors > 0 {
		fmt.Printf("Errors:              %d\n", result.Errors)
	}

	if *dryRun {
		fmt.Println()
		fmt.Println("Run without --dry-run to apply these changes.")
	}
}

// observatoryCleanupCommand deletes old/noise spans based on retention policy.
// M-DB-CLEANUP: Implements the retention strategy from design_docs/planned/m-db-cleanup.md
func observatoryCleanupCommand() {
	fs := flag.NewFlagSet("observatory cleanup", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview what would be deleted without making changes")
	vacuum := fs.Bool("vacuum", false, "Run VACUUM after cleanup to reclaim disk space")
	noiseRetention := fs.Int("noise-days", 7, "Days to keep orphan noise spans (default: 7)")
	toolRetention := fs.Int("tool-days", 30, "Days to keep tool usage spans (default: 30)")
	compileRetention := fs.Int("compile-days", 90, "Days to keep compilation spans (default: 90)")
	verbose := fs.Bool("verbose", false, "Show detailed output for each category")

	// Skip "ailang observatory cleanup" args
	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Get database path
	dbPath := observatory.DefaultDatabasePath()

	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	db := backend.DB()
	if db == nil {
		fmt.Fprintf(os.Stderr, "Error: database connection is nil\n")
		os.Exit(1)
	}

	// Get current span count and size
	var totalSpans int
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM spans").Scan(&totalSpans)

	fmt.Println("=== Observatory Cleanup ===")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Database: %s\n", dbPath)
	fmt.Printf("Total spans before: %d\n\n", totalSpans)

	// Define cleanup categories with their retention policies
	type CleanupCategory struct {
		Name       string
		Days       int
		Condition  string
		Exclusions []string // Span names to exclude from deletion
	}

	categories := []CleanupCategory{
		{
			Name:      "Orphan noise spans (no task_id)",
			Days:      *noiseRetention,
			Condition: "task_id IS NULL",
			Exclusions: []string{
				// Keep LLM call spans - they have cost data
				"anthropic.%", "openai.%", "gemini.%", "ollama.%",
			},
		},
		{
			Name:      "Server HTTP spans (ailang-server)",
			Days:      *noiseRetention,
			Condition: "name = 'ailang-server'",
		},
		{
			Name:      "Tool usage spans (claude_code.tool.*)",
			Days:      *toolRetention,
			Condition: "name LIKE 'claude_code.tool.%'",
		},
		{
			Name:      "Compilation spans (compile.*)",
			Days:      *compileRetention,
			Condition: "name LIKE 'compile.%'",
		},
	}

	var totalDeleted int

	for _, cat := range categories {
		// Build query with exclusions
		whereClause := cat.Condition + fmt.Sprintf(" AND created_at < datetime('now', '-%d days')", cat.Days)
		for _, excl := range cat.Exclusions {
			whereClause += fmt.Sprintf(" AND name NOT LIKE '%s'", excl)
		}

		// Count spans to delete
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM spans WHERE %s", whereClause)
		var count int
		if err := db.QueryRowContext(ctx, countQuery).Scan(&count); err != nil {
			fmt.Fprintf(os.Stderr, "Error counting %s: %v\n", cat.Name, err)
			continue
		}

		if *verbose || count > 0 {
			fmt.Printf("%-45s %6d spans (>%d days old)\n", cat.Name+":", count, cat.Days)
		}

		if count > 0 && !*dryRun {
			deleteQuery := fmt.Sprintf("DELETE FROM spans WHERE %s", whereClause)
			result, err := db.ExecContext(ctx, deleteQuery)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error deleting %s: %v\n", cat.Name, err)
				continue
			}
			deleted, _ := result.RowsAffected()
			totalDeleted += int(deleted)
		} else if count > 0 {
			totalDeleted += count
		}
	}

	fmt.Println(strings.Repeat("─", 60))

	if *dryRun {
		fmt.Printf("Would delete: %d spans\n", totalDeleted)
		fmt.Println()
		fmt.Println("Run without --dry-run to apply these changes.")
	} else {
		fmt.Printf("Deleted: %d spans\n", totalDeleted)

		// Get new span count
		var newTotal int
		db.QueryRowContext(ctx, "SELECT COUNT(*) FROM spans").Scan(&newTotal)
		fmt.Printf("Total spans after: %d\n", newTotal)

		// Vacuum if requested
		if *vacuum {
			fmt.Println()
			fmt.Println("Running VACUUM to reclaim disk space...")
			if _, err := db.ExecContext(ctx, "VACUUM"); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: VACUUM failed: %v\n", err)
			} else {
				fmt.Println("VACUUM complete.")
			}
		}
	}

	// M-OBS-RETENTION: Also run full retention on all tables (chat, metrics, summaries, tools)
	fmt.Println()
	fmt.Println("Running full retention (7d spans/metrics, 30d chat/tools)...")
	obsStore, err := observatory.OpenDefaultStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open store for retention: %v\n", err)
		return
	}
	defer obsStore.Close()
	stats, err := obsStore.RunRetention(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: retention failed: %v\n", err)
	} else {
		fmt.Printf("Retention: %s (total: %d rows)\n", stats, stats.Total())
	}
}

// BackfillOptions configures the backfill operation.
type BackfillOptions struct {
	DryRun  bool
	Window  time.Duration
	TaskID  string
	Verbose bool
}

// BackfillResult contains statistics from the backfill operation.
type BackfillResult struct {
	TasksScanned       int
	SpansScanned       int
	SpansAlreadyLinked int
	SpansLinked        int
	Errors             int
}

// runBackfill links unlinked spans to tasks based on time correlation.
func runBackfill(ctx context.Context, backend observatory.Backend, opts BackfillOptions) (*BackfillResult, error) {
	result := &BackfillResult{}

	// Track which tasks need aggregate recalculation
	tasksToRecalculate := make(map[string]bool)

	// Get all tasks or just the specified one
	var tasks []*observatory.Task
	var err error

	if opts.TaskID != "" {
		task, err := backend.GetTask(ctx, opts.TaskID)
		if err != nil {
			return nil, fmt.Errorf("get task: %w", err)
		}
		if task == nil {
			return nil, fmt.Errorf("task not found: %s", opts.TaskID)
		}
		tasks = []*observatory.Task{task}
	} else {
		tasks, err = backend.ListTasks(ctx, observatory.TaskListOptions{Limit: 1000})
		if err != nil {
			return nil, fmt.Errorf("list tasks: %w", err)
		}
	}

	result.TasksScanned = len(tasks)

	if opts.Verbose {
		fmt.Printf("Scanning %d tasks with window=%s\n", len(tasks), opts.Window)
	}

	// For each task, find spans that overlap in time
	for _, task := range tasks {
		assignments, err := backend.ListAgentAssignments(ctx, task.ID)
		if err != nil {
			result.Errors++
			if opts.Verbose {
				fmt.Printf("  Error listing assignments for task %s: %v\n", task.ID, err)
			}
			continue
		}

		for _, assignment := range assignments {
			// Determine time window for this assignment
			startTime := assignment.AssignedAt.Add(-opts.Window)
			endTime := time.Now()
			if assignment.CompletedAt != nil {
				endTime = assignment.CompletedAt.Add(opts.Window)
			}

			if opts.Verbose {
				fmt.Printf("  Assignment %s: provider=%s, window=%s to %s\n",
					assignment.ID, assignment.Provider, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
			}

			// Find unlinked spans in this time window
			// Use large limit to ensure we get all spans (pagination would be complex)
			spans, err := backend.ListSpans(ctx, observatory.SpanListOptions{
				StartAfter:  startTime,
				StartBefore: endTime,
				Limit:       100000,
			})
			if err != nil {
				result.Errors++
				if opts.Verbose {
					fmt.Printf("  Error listing spans for assignment %s: %v\n", assignment.ID, err)
				}
				continue
			}

			if opts.Verbose && len(spans) > 0 {
				fmt.Printf("    Found %d spans in window\n", len(spans))
			}

			for _, span := range spans {
				result.SpansScanned++

				// Skip already linked spans
				if span.TaskID != "" || span.AgentAssignmentID != "" {
					result.SpansAlreadyLinked++
					continue
				}

				// Check if span matches this task (by task ID from cwd path or resource attrs)
				if !spanMatchesTask(span, task, assignment) {
					// Debug: check why it didn't match
					if opts.Verbose {
						cwd := ""
						if span.ResourceAttributes != nil {
							if c, ok := span.ResourceAttributes["process.cwd"].(string); ok {
								cwd = c
							}
						}
						extractedTaskID := extractTaskIDFromCwd(cwd)
						fmt.Printf("    Span %s: cwd=%s, extracted=%s, expected=%s\n",
							span.ID[:8], cwd[max(0, len(cwd)-50):], extractedTaskID, task.ID)
					}
					continue
				}

				if opts.Verbose {
					fmt.Printf("  Linking span %s (%s) to task %s, agent %s\n",
						span.ID[:8], span.Name, task.ID, assignment.AgentID)
				}

				if !opts.DryRun {
					// Update span with task and assignment IDs
					span.TaskID = task.ID
					span.AgentAssignmentID = assignment.ID
					if err := backend.UpdateSpanLinks(ctx, span.ID, task.ID, assignment.ID); err != nil {
						result.Errors++
						if opts.Verbose {
							fmt.Printf("    Error updating span: %v\n", err)
						}
						continue
					}
					// Mark task for aggregate recalculation
					tasksToRecalculate[task.ID] = true
				}

				result.SpansLinked++
			}
		}
	}

	// Recalculate aggregates for tasks that had spans linked
	if !opts.DryRun && len(tasksToRecalculate) > 0 {
		if opts.Verbose {
			fmt.Printf("Recalculating aggregates for %d tasks...\n", len(tasksToRecalculate))
		}
		for taskID := range tasksToRecalculate {
			if err := backend.RecalculateTaskAggregates(ctx, taskID); err != nil {
				result.Errors++
				if opts.Verbose {
					fmt.Printf("  Error recalculating aggregates for task %s: %v\n", taskID, err)
				}
			}
		}
	}

	return result, nil
}

// spanMatchesTask checks if a span came from a specific task.
// Matches spans that have task context either from:
// 1. ailang.task_id in resource_attributes (set by coordinator via OTEL_RESOURCE_ATTRIBUTES)
// 2. process.cwd containing worktree path (fallback - Claude Code doesn't pass env to subprocesses)
// Also verifies the task ID matches the expected task.
func spanMatchesTask(span *observatory.Span, task *observatory.Task, assignment *observatory.AgentAssignment) bool {
	if span.ResourceAttributes == nil {
		return false
	}

	// Try explicit ailang.task_id first
	spanTaskID, _ := span.ResourceAttributes["ailang.task_id"].(string)

	// Fallback: extract from process.cwd worktree path
	if spanTaskID == "" {
		if cwd, ok := span.ResourceAttributes["process.cwd"].(string); ok {
			spanTaskID = extractTaskIDFromCwd(cwd)
		}
	}

	if spanTaskID == "" {
		// No task context - this span wasn't from a coordinator task
		return false
	}

	// Verify task ID matches the task we're backfilling
	if spanTaskID != task.ID {
		return false
	}

	// Check for assignment_id match if present
	if assignmentID, ok := span.ResourceAttributes["ailang.assignment_id"].(string); ok && assignmentID != "" {
		// Exact assignment match
		return assignmentID == assignment.ID
	}

	// Task ID matches - check service name for additional verification
	// Accept AILANG services (they're our tooling running within the task)
	// Also accept provider-specific services as a sanity check
	if serviceName, ok := span.ResourceAttributes["service.name"].(string); ok {
		// AILANG tooling spans (ailang-check, ailang-messages, etc.)
		if strings.HasPrefix(serviceName, "ailang") {
			return true
		}
		// Provider-specific services
		switch assignment.Provider {
		case observatory.ProviderClaude:
			return strings.Contains(serviceName, "claude") || strings.Contains(serviceName, "anthropic")
		case observatory.ProviderGemini:
			return strings.Contains(serviceName, "gemini") || strings.Contains(serviceName, "google")
		}
	}

	// Task ID matched but no recognizable service name - still link it
	// (the task ID from cwd is strong enough evidence)
	return true
}

// extractTaskIDFromCwd extracts task ID from worktree path.
// Path format: /Users/.../worktrees/coordinator/task-XXXXXXXX
func extractTaskIDFromCwd(cwd string) string {
	const taskPrefix = "task-"
	idx := strings.Index(cwd, "/worktrees/")
	if idx == -1 {
		return ""
	}

	remainder := cwd[idx:]
	taskIdx := strings.Index(remainder, taskPrefix)
	if taskIdx == -1 {
		return ""
	}

	// Find end of task ID (next / or end of string)
	start := taskIdx
	end := start + len(taskPrefix) + 8
	if end > len(remainder) {
		nextSlash := strings.Index(remainder[start:], "/")
		if nextSlash > 0 {
			end = start + nextSlash
		} else {
			end = len(remainder)
		}
	}

	taskID := remainder[start:end]
	if strings.HasPrefix(taskID, taskPrefix) {
		return taskID
	}
	return ""
}

// observatorySeedCommand generates test data for the observatory dashboard.

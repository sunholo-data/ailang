package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
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
func observatorySeedCommand() {
	fs := flag.NewFlagSet("observatory seed", flag.ExitOnError)
	minimal := fs.Bool("minimal", false, "Generate minimal test data (1 workspace, 2 tasks, 5 spans)")
	stress := fs.Bool("stress", false, "Generate stress test data (10 workspaces, 100 tasks, 1000+ spans)")
	clean := fs.Bool("clean", false, "Delete all existing data before seeding")
	dbPath := fs.String("db", "", "Custom database path (default: ~/.ailang/state/observatory.db)")
	verbose := fs.Bool("verbose", false, "Show detailed output during seeding")

	// Skip "ailang observatory seed" args
	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Get database path
	path := *dbPath
	if path == "" {
		path = observatory.DefaultDatabasePath()
	}

	// Clean database if requested
	if *clean {
		if *verbose {
			fmt.Printf("Cleaning database at %s...\n", path)
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: could not remove database: %v\n", err)
		}
	}

	backend, err := observatory.NewSQLiteBackendFromPath(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	// Select configuration
	var cfg observatory.SeedConfig
	switch {
	case *minimal:
		cfg = observatory.MinimalSeedConfig()
	case *stress:
		cfg = observatory.StressSeedConfig()
	default:
		cfg = observatory.DefaultSeedConfig()
	}
	cfg.Verbose = *verbose
	cfg.CleanFirst = *clean

	// Run seeding
	fmt.Println("Seeding observatory database...")
	if *verbose {
		fmt.Printf("  Database: %s\n", path)
		fmt.Printf("  Workspaces: %d\n", cfg.NumWorkspaces)
		fmt.Printf("  Tasks per workspace: %d-%d\n", cfg.TasksPerWorkspace.Min, cfg.TasksPerWorkspace.Max)
		fmt.Printf("  Spans per task: %d-%d\n", cfg.SpansPerTask.Min, cfg.SpansPerTask.Max)
	}

	result, err := observatory.SeedDatabase(ctx, backend, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error seeding database: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Println()
	fmt.Println("=== Seed Complete ===")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("Workspaces created:   %d\n", result.WorkspacesCreated)
	fmt.Printf("Tasks created:        %d\n", result.TasksCreated)
	fmt.Printf("Assignments created:  %d\n", result.AssignmentsCreated)
	fmt.Printf("Spans created:        %d\n", result.SpansCreated)
	fmt.Printf("Events created:       %d\n", result.EventsCreated)
	fmt.Printf("Messages created:     %d\n", result.MessagesCreated)
	fmt.Println()
	fmt.Println("View the data at: http://localhost:1957 (run 'ailang serve' first)")
}

// observatoryHeatmapCommand outputs activity heatmap data.
func observatoryHeatmapCommand() {
	fs := flag.NewFlagSet("observatory heatmap", flag.ExitOnError)
	days := fs.Int("days", 90, "Number of days of history to show (ignored if --since is set)")
	format := fs.String("format", "json", "Output format: json or ascii")
	provider := fs.String("provider", "", "Filter by provider")
	workspace := fs.String("workspace", "", "Filter by workspace")
	since := fs.String("since", "", "Start date (YYYY-MM-DD)")
	until := fs.String("until", "", "End date (YYYY-MM-DD)")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	dbPath := observatory.DefaultDatabasePath()

	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	// Query for daily aggregates
	db := backend.DB()
	if db == nil {
		fmt.Fprintf(os.Stderr, "Error: database connection is nil\n")
		os.Exit(1)
	}

	// Determine date range
	var startDate, endDate time.Time
	if *since != "" {
		if t, err := time.Parse("2006-01-02", *since); err == nil {
			startDate = t
		} else {
			fmt.Fprintf(os.Stderr, "Invalid --since date format (use YYYY-MM-DD): %v\n", err)
			os.Exit(1)
		}
	} else {
		startDate = time.Now().AddDate(0, 0, -*days)
	}
	if *until != "" {
		if t, err := time.Parse("2006-01-02", *until); err == nil {
			endDate = t.AddDate(0, 0, 1) // Include the end date
		} else {
			fmt.Fprintf(os.Stderr, "Invalid --until date format (use YYYY-MM-DD): %v\n", err)
			os.Exit(1)
		}
	} else {
		endDate = time.Now().AddDate(0, 0, 1)
	}

	// Build filter conditions
	conditions := []string{"start_time >= ?", "start_time < ?"}
	args := []interface{}{startDate.Format("2006-01-02 15:04:05"), endDate.Format("2006-01-02 15:04:05")}

	if *provider != "" {
		conditions = append(conditions, "provider = ?")
		args = append(args, *provider)
	}
	if *workspace != "" {
		conditions = append(conditions, "json_extract(resource_attributes, '$.\"process.cwd\"') = ?")
		args = append(args, *workspace)
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT
			date(start_time) as day,
			COUNT(*) as span_count,
			SUM(COALESCE(cost_usd, 0)) as total_cost,
			SUM(COALESCE(tokens_in, 0) + COALESCE(tokens_out, 0)) as total_tokens
		FROM spans
		WHERE %s
		GROUP BY date(start_time)
		ORDER BY day
	`, whereClause)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying database: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	type DayData struct {
		Day       string  `json:"day"`
		SpanCount int     `json:"span_count"`
		Cost      float64 `json:"cost"`
		Tokens    int64   `json:"tokens"`
	}

	var data []DayData
	for rows.Next() {
		var d DayData
		if err := rows.Scan(&d.Day, &d.SpanCount, &d.Cost, &d.Tokens); err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning row: %v\n", err)
			continue
		}
		data = append(data, d)
	}

	if *format == "ascii" {
		// Build CLI command for display
		cliCmd := "ailang observatory heatmap"
		if *since != "" {
			cliCmd += fmt.Sprintf(" --since %s", *since)
		} else {
			cliCmd += fmt.Sprintf(" --days %d", *days)
		}
		if *until != "" {
			cliCmd += fmt.Sprintf(" --until %s", *until)
		}
		if *provider != "" {
			cliCmd += fmt.Sprintf(" --provider %s", *provider)
		}
		if *workspace != "" {
			cliCmd += fmt.Sprintf(" --workspace %s", *workspace)
		}
		cliCmd += " --format ascii"

		fmt.Println("=== Activity Heatmap ===")
		fmt.Println(strings.Repeat("─", 60))

		if len(data) == 0 {
			fmt.Println("No data found for the specified period.")
			return
		}

		// Find max for scaling
		maxCount := 0
		for _, d := range data {
			if d.SpanCount > maxCount {
				maxCount = d.SpanCount
			}
		}

		// Heatmap characters from low to high intensity
		heatChars := []rune{'░', '▒', '▓', '█'}

		// Group by week for display
		fmt.Printf("\nDaily Activity (last %d days):\n\n", *days)

		// Show last 12 weeks in a grid format
		currentDate := time.Now()
		dataMap := make(map[string]DayData)
		for _, d := range data {
			dataMap[d.Day] = d
		}

		// Print week headers
		fmt.Print("       ")
		for i := 0; i < 7; i++ {
			fmt.Printf(" %s", []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}[i])
		}
		fmt.Println()

		// Print 12 weeks
		for week := 11; week >= 0; week-- {
			weekStart := currentDate.AddDate(0, 0, -week*7-int(currentDate.Weekday()))
			fmt.Printf("W%-2d    ", 12-week)

			for day := 0; day < 7; day++ {
				d := weekStart.AddDate(0, 0, day)
				dayStr := d.Format("2006-01-02")
				if dd, ok := dataMap[dayStr]; ok {
					// Scale to heatmap character
					level := 0
					if maxCount > 0 {
						level = int(float64(dd.SpanCount) / float64(maxCount) * float64(len(heatChars)-1))
						if level >= len(heatChars) {
							level = len(heatChars) - 1
						}
					}
					fmt.Printf(" %c ", heatChars[level])
				} else if d.After(time.Now()) {
					fmt.Print(" · ")
				} else {
					fmt.Print(" · ")
				}
			}
			fmt.Println()
		}

		fmt.Println()
		fmt.Printf("Legend: · = no activity, ░ = low, ▒ = medium, ▓ = high, █ = max\n")
		fmt.Println()
		fmt.Printf("CLI: %s\n", cliCmd)
	} else {
		// JSON output - build CLI command with filters
		jsonCliCmd := "ailang observatory heatmap"
		if *since != "" {
			jsonCliCmd += fmt.Sprintf(" --since %s", *since)
		} else {
			jsonCliCmd += fmt.Sprintf(" --days %d", *days)
		}
		if *until != "" {
			jsonCliCmd += fmt.Sprintf(" --until %s", *until)
		}
		if *provider != "" {
			jsonCliCmd += fmt.Sprintf(" --provider %s", *provider)
		}
		if *workspace != "" {
			jsonCliCmd += fmt.Sprintf(" --workspace %s", *workspace)
		}
		jsonCliCmd += " --format json"

		output := map[string]interface{}{
			"days":        data,
			"cli_command": jsonCliCmd,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(output)
	}
}

// observatoryEvolutionCommand outputs task evolution data for line charts.
func observatoryEvolutionCommand() {
	fs := flag.NewFlagSet("observatory evolution", flag.ExitOnError)
	metric := fs.String("metric", "cost", "Metric to track: cost, tokens, turns, spans")
	limit := fs.Int("limit", 10, "Maximum number of tasks to return")
	format := fs.String("format", "json", "Output format: json or ascii")
	provider := fs.String("provider", "", "Filter by provider")
	since := fs.String("since", "", "Start date (YYYY-MM-DD)")
	until := fs.String("until", "", "End date (YYYY-MM-DD)")
	workspace := fs.String("workspace", "", "Filter by workspace path")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
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

	// Get recent tasks
	taskQuery := `
		SELECT id, title, source_type, status,
		       COALESCE(total_cost_usd, 0) as total_cost,
		       COALESCE(total_tokens_in + total_tokens_out, 0) as total_tokens,
		       COALESCE(agent_count, 0) as total_turns,
		       COALESCE(span_count, 0) as total_spans
		FROM tasks
		WHERE 1=1
	`
	args := []interface{}{}

	if *provider != "" {
		taskQuery += " AND source_type = ?"
		args = append(args, *provider)
	}
	if *since != "" {
		taskQuery += " AND date(created_at) >= ?"
		args = append(args, *since)
	}
	if *until != "" {
		taskQuery += " AND date(created_at) <= ?"
		args = append(args, *until)
	}
	if *workspace != "" {
		taskQuery += " AND workspace = ?"
		args = append(args, *workspace)
	}

	taskQuery += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, *limit)

	rows, err := db.QueryContext(ctx, taskQuery, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying tasks: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	type TaskEvolution struct {
		TaskID      string  `json:"task_id"`
		Title       string  `json:"title"`
		SourceType  string  `json:"source_type"`
		Status      string  `json:"status"`
		TotalCost   float64 `json:"total_cost"`
		TotalTokens int64   `json:"total_tokens"`
		TotalTurns  int     `json:"total_turns"`
		TotalSpans  int     `json:"total_spans"`
	}

	var tasks []TaskEvolution
	for rows.Next() {
		var t TaskEvolution
		if err := rows.Scan(&t.TaskID, &t.Title, &t.SourceType, &t.Status,
			&t.TotalCost, &t.TotalTokens, &t.TotalTurns, &t.TotalSpans); err != nil {
			continue
		}
		tasks = append(tasks, t)
	}

	cliCmd := fmt.Sprintf("ailang observatory evolution --metric %s --limit %d", *metric, *limit)
	if *since != "" {
		cliCmd += fmt.Sprintf(" --since %s", *since)
	}
	if *until != "" {
		cliCmd += fmt.Sprintf(" --until %s", *until)
	}
	if *workspace != "" {
		cliCmd += fmt.Sprintf(" --workspace %s", *workspace)
	}
	if *provider != "" {
		cliCmd += fmt.Sprintf(" --provider %s", *provider)
	}

	if *format == "ascii" {
		fmt.Println("=== Task Evolution ===")
		fmt.Println(strings.Repeat("─", 60))
		fmt.Printf("Metric: %s\n\n", *metric)

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return
		}

		// Find max value for scaling
		maxVal := 0.0
		for _, t := range tasks {
			var val float64
			switch *metric {
			case "cost":
				val = t.TotalCost
			case "tokens":
				val = float64(t.TotalTokens)
			case "turns":
				val = float64(t.TotalTurns)
			case "spans":
				val = float64(t.TotalSpans)
			}
			if val > maxVal {
				maxVal = val
			}
		}

		// Display as sparklines
		barWidth := 30
		for _, t := range tasks {
			var val float64
			var suffix string
			switch *metric {
			case "cost":
				val = t.TotalCost
				suffix = fmt.Sprintf("$%.2f", val)
			case "tokens":
				val = float64(t.TotalTokens)
				if val > 1000 {
					suffix = fmt.Sprintf("%.1fK", val/1000)
				} else {
					suffix = fmt.Sprintf("%.0f", val)
				}
			case "turns":
				val = float64(t.TotalTurns)
				suffix = fmt.Sprintf("%d", t.TotalTurns)
			case "spans":
				val = float64(t.TotalSpans)
				suffix = fmt.Sprintf("%d", t.TotalSpans)
			}

			// Render bar
			filled := 0
			if maxVal > 0 {
				filled = int(val / maxVal * float64(barWidth))
			}
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

			// Truncate title
			title := t.Title
			if len(title) > 25 {
				title = title[:22] + "..."
			}

			fmt.Printf("%-25s %s %s\n", title, bar, suffix)
		}

		fmt.Println()
		cliCmd += " --format ascii"
		fmt.Printf("CLI: %s\n", cliCmd)
	} else {
		output := map[string]interface{}{
			"tasks":       tasks,
			"metric":      *metric,
			"cli_command": cliCmd + " --format json",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(output)
	}
}

// observatoryUsageCommand outputs time-bucketed usage data for column charts.
func observatoryUsageCommand() {
	fs := flag.NewFlagSet("observatory usage", flag.ExitOnError)
	metric := fs.String("metric", "cost", "Metric to aggregate: cost, tokens, turns, spans")
	interval := fs.String("interval", "day", "Time interval: hour, day, week")
	splitBy := fs.String("split-by", "", "Split by dimension: provider, model, workspace")
	format := fs.String("format", "json", "Output format: json or ascii")
	days := fs.Int("days", 30, "Number of days of history (ignored if --since is set)")
	since := fs.String("since", "", "Start date (YYYY-MM-DD)")
	until := fs.String("until", "", "End date (YYYY-MM-DD)")
	workspace := fs.String("workspace", "", "Filter by workspace path")
	provider := fs.String("provider", "", "Filter by provider")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
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

	// Determine date range
	var startDate, endDate time.Time
	if *since != "" {
		if t, err := time.Parse("2006-01-02", *since); err == nil {
			startDate = t
		} else {
			fmt.Fprintf(os.Stderr, "Invalid --since date format (use YYYY-MM-DD): %v\n", err)
			os.Exit(1)
		}
	} else {
		startDate = time.Now().AddDate(0, 0, -*days)
	}
	if *until != "" {
		if t, err := time.Parse("2006-01-02", *until); err == nil {
			// Add 1 day to include the full end date
			endDate = t.AddDate(0, 0, 1)
		} else {
			fmt.Fprintf(os.Stderr, "Invalid --until date format (use YYYY-MM-DD): %v\n", err)
			os.Exit(1)
		}
	} else {
		endDate = time.Now().AddDate(0, 0, 1) // Include today
	}

	// Build date grouping expression based on interval
	var dateExpr string
	switch *interval {
	case "hour":
		dateExpr = "strftime('%Y-%m-%d %H:00', start_time)"
	case "week":
		dateExpr = "strftime('%Y-W%W', start_time)"
	default: // day
		dateExpr = "date(start_time)"
	}

	type UsagePoint struct {
		Bucket      string             `json:"bucket"`
		SpanCount   int                `json:"span_count"`
		Cost        float64            `json:"cost"`
		Tokens      int64              `json:"tokens"`
		ByDimension map[string]float64 `json:"by_dimension,omitempty"`
	}

	var points []UsagePoint

	// Build filter conditions
	conditions := []string{"start_time >= ?", "start_time < ?"}
	args := []interface{}{startDate.Format("2006-01-02 15:04:05"), endDate.Format("2006-01-02 15:04:05")}

	if *workspace != "" {
		conditions = append(conditions, "json_extract(resource_attributes, '$.\"process.cwd\"') = ?")
		args = append(args, *workspace)
	}
	if *provider != "" {
		conditions = append(conditions, "provider = ?")
		args = append(args, *provider)
	}

	whereClause := strings.Join(conditions, " AND ")

	if *splitBy != "" {
		// Query with dimension split
		var splitCol string
		switch *splitBy {
		case "provider":
			splitCol = "COALESCE(provider, 'unknown')"
		case "model":
			splitCol = "COALESCE(model, 'unknown')"
		case "workspace":
			splitCol = "COALESCE(json_extract(resource_attributes, '$.\"process.cwd\"'), 'unknown')"
		default:
			splitCol = "COALESCE(provider, 'unknown')"
		}

		query := fmt.Sprintf(`
			SELECT
				%s as bucket,
				%s as dimension,
				COUNT(*) as span_count,
				SUM(COALESCE(cost_usd, 0)) as total_cost,
				SUM(COALESCE(tokens_in, 0) + COALESCE(tokens_out, 0)) as total_tokens
			FROM spans
			WHERE %s
			GROUP BY bucket, dimension
			ORDER BY bucket
		`, dateExpr, splitCol, whereClause)

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error querying database: %v\n", err)
			os.Exit(1)
		}
		defer rows.Close()

		// Aggregate by bucket with dimension breakdown
		bucketMap := make(map[string]*UsagePoint)
		for rows.Next() {
			var bucket, dimension string
			var spanCount int
			var cost float64
			var tokens int64
			if err := rows.Scan(&bucket, &dimension, &spanCount, &cost, &tokens); err != nil {
				continue
			}

			// Normalize workspace paths
			if *splitBy == "workspace" {
				dimension = normalizeWorkspacePath(dimension)
			}

			if bucketMap[bucket] == nil {
				bucketMap[bucket] = &UsagePoint{
					Bucket:      bucket,
					ByDimension: make(map[string]float64),
				}
			}
			p := bucketMap[bucket]
			p.SpanCount += spanCount
			p.Cost += cost
			p.Tokens += tokens
			p.ByDimension[dimension] += cost // Aggregate by normalized dimension
		}

		// Convert map to sorted slice
		buckets := make([]string, 0, len(bucketMap))
		for b := range bucketMap {
			buckets = append(buckets, b)
		}
		sort.Strings(buckets)
		for _, b := range buckets {
			points = append(points, *bucketMap[b])
		}
	} else {
		// Query without split
		query := fmt.Sprintf(`
			SELECT
				%s as bucket,
				COUNT(*) as span_count,
				SUM(COALESCE(cost_usd, 0)) as total_cost,
				SUM(COALESCE(tokens_in, 0) + COALESCE(tokens_out, 0)) as total_tokens
			FROM spans
			WHERE %s
			GROUP BY bucket
			ORDER BY bucket
		`, dateExpr, whereClause)

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error querying database: %v\n", err)
			os.Exit(1)
		}
		defer rows.Close()

		for rows.Next() {
			var p UsagePoint
			if err := rows.Scan(&p.Bucket, &p.SpanCount, &p.Cost, &p.Tokens); err != nil {
				continue
			}
			points = append(points, p)
		}
	}

	cliCmd := fmt.Sprintf("ailang observatory usage --metric %s --interval %s", *metric, *interval)
	if *since != "" {
		cliCmd += fmt.Sprintf(" --since %s", *since)
	} else {
		cliCmd += fmt.Sprintf(" --days %d", *days)
	}
	if *until != "" {
		cliCmd += fmt.Sprintf(" --until %s", *until)
	}
	if *workspace != "" {
		cliCmd += fmt.Sprintf(" --workspace %s", *workspace)
	}
	if *provider != "" {
		cliCmd += fmt.Sprintf(" --provider %s", *provider)
	}
	if *splitBy != "" {
		cliCmd += fmt.Sprintf(" --split-by %s", *splitBy)
	}

	if *format == "ascii" {
		fmt.Println("=== Usage Time Series ===")
		fmt.Println(strings.Repeat("─", 60))
		fmt.Printf("Metric: %s, Interval: %s\n\n", *metric, *interval)

		if len(points) == 0 {
			fmt.Println("No data found.")
			return
		}

		// Find max for scaling
		maxVal := 0.0
		for _, p := range points {
			var val float64
			switch *metric {
			case "cost":
				val = p.Cost
			case "tokens":
				val = float64(p.Tokens)
			case "turns", "spans":
				val = float64(p.SpanCount)
			}
			if val > maxVal {
				maxVal = val
			}
		}

		barWidth := 40
		for _, p := range points {
			var val float64
			var suffix string
			switch *metric {
			case "cost":
				val = p.Cost
				suffix = fmt.Sprintf("$%.2f", val)
			case "tokens":
				val = float64(p.Tokens)
				if val > 1000000 {
					suffix = fmt.Sprintf("%.1fM", val/1000000)
				} else if val > 1000 {
					suffix = fmt.Sprintf("%.1fK", val/1000)
				} else {
					suffix = fmt.Sprintf("%.0f", val)
				}
			case "turns", "spans":
				val = float64(p.SpanCount)
				suffix = fmt.Sprintf("%d", p.SpanCount)
			}

			filled := 0
			if maxVal > 0 {
				filled = int(val / maxVal * float64(barWidth))
			}
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

			fmt.Printf("%-12s %s %s\n", p.Bucket, bar, suffix)
		}

		fmt.Println()
		cliCmd += " --format ascii"
		fmt.Printf("CLI: %s\n", cliCmd)
	} else {
		output := map[string]interface{}{
			"points":      points,
			"metric":      *metric,
			"interval":    *interval,
			"cli_command": cliCmd + " --format json",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(output)
	}
}

// observatoryTokensCommand outputs token distribution histogram.
func observatoryTokensCommand() {
	fs := flag.NewFlagSet("observatory tokens", flag.ExitOnError)
	format := fs.String("format", "json", "Output format: json or ascii")
	days := fs.Int("days", 30, "Number of days of history (ignored if --since is set)")
	since := fs.String("since", "", "Start date (YYYY-MM-DD)")
	until := fs.String("until", "", "End date (YYYY-MM-DD)")
	workspace := fs.String("workspace", "", "Filter by workspace path")
	provider := fs.String("provider", "", "Filter by provider")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
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

	// Determine date range
	var startDate, endDate time.Time
	if *since != "" {
		if t, err := time.Parse("2006-01-02", *since); err == nil {
			startDate = t
		} else {
			fmt.Fprintf(os.Stderr, "Invalid --since date format (use YYYY-MM-DD): %v\n", err)
			os.Exit(1)
		}
	} else {
		startDate = time.Now().AddDate(0, 0, -*days)
	}
	if *until != "" {
		if t, err := time.Parse("2006-01-02", *until); err == nil {
			endDate = t.AddDate(0, 0, 1) // Include the end date
		} else {
			fmt.Fprintf(os.Stderr, "Invalid --until date format (use YYYY-MM-DD): %v\n", err)
			os.Exit(1)
		}
	} else {
		endDate = time.Now().AddDate(0, 0, 1)
	}

	// Build filter conditions
	conditions := []string{"start_time >= ?", "start_time < ?"}
	baseArgs := []interface{}{startDate.Format("2006-01-02 15:04:05"), endDate.Format("2006-01-02 15:04:05")}

	if *workspace != "" {
		conditions = append(conditions, "json_extract(resource_attributes, '$.\"process.cwd\"') = ?")
		baseArgs = append(baseArgs, *workspace)
	}
	if *provider != "" {
		conditions = append(conditions, "provider = ?")
		baseArgs = append(baseArgs, *provider)
	}

	whereClause := strings.Join(conditions, " AND ")

	// Define buckets for token distribution
	buckets := []struct {
		label string
		min   int64
		max   int64
	}{
		{"0-1K", 0, 1000},
		{"1K-5K", 1000, 5000},
		{"5K-20K", 5000, 20000},
		{"20K-50K", 20000, 50000},
		{"50K-100K", 50000, 100000},
		{"100K+", 100000, 999999999},
	}

	type BucketCount struct {
		Label string `json:"label"`
		Min   int64  `json:"min"`
		Max   int64  `json:"max"`
		Count int    `json:"count"`
	}

	var results []BucketCount

	for _, b := range buckets {
		query := fmt.Sprintf(`
			SELECT COUNT(*) FROM spans
			WHERE %s
			AND (COALESCE(tokens_in, 0) + COALESCE(tokens_out, 0)) >= ?
			AND (COALESCE(tokens_in, 0) + COALESCE(tokens_out, 0)) < ?
		`, whereClause)
		args := append(baseArgs, b.min, b.max)
		var count int
		err := db.QueryRowContext(ctx, query, args...).Scan(&count)
		if err != nil {
			continue
		}
		results = append(results, BucketCount{
			Label: b.label,
			Min:   b.min,
			Max:   b.max,
			Count: count,
		})
	}

	// Build CLI command
	cliCmd := "ailang observatory tokens"
	if *since != "" {
		cliCmd += fmt.Sprintf(" --since %s", *since)
	} else {
		cliCmd += fmt.Sprintf(" --days %d", *days)
	}
	if *until != "" {
		cliCmd += fmt.Sprintf(" --until %s", *until)
	}
	if *workspace != "" {
		cliCmd += fmt.Sprintf(" --workspace %s", *workspace)
	}
	if *provider != "" {
		cliCmd += fmt.Sprintf(" --provider %s", *provider)
	}

	if *format == "ascii" {
		fmt.Println("=== Token Distribution ===")
		fmt.Println(strings.Repeat("─", 60))
		fmt.Printf("Period: last %d days\n\n", *days)

		// Find max for scaling
		maxCount := 0
		for _, r := range results {
			if r.Count > maxCount {
				maxCount = r.Count
			}
		}

		barWidth := 40
		for _, r := range results {
			filled := 0
			if maxCount > 0 {
				filled = int(float64(r.Count) / float64(maxCount) * float64(barWidth))
			}
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			fmt.Printf("%-10s %s %d spans\n", r.Label, bar, r.Count)
		}

		fmt.Println()
		cliCmd += " --format ascii"
		fmt.Printf("CLI: %s\n", cliCmd)
	} else {
		output := map[string]interface{}{
			"buckets":     results,
			"cli_command": cliCmd + " --format json",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(output)
	}
}

// observatoryOutliersCommand detects statistical outliers in spans within a task.
func observatoryOutliersCommand() {
	fs := flag.NewFlagSet("observatory outliers", flag.ExitOnError)
	taskID := fs.String("task", "", "Task ID to analyze (required)")
	metric := fs.String("metric", "", "Filter to specific metric: cost, duration, tokens (default: all)")
	threshold := fs.Float64("threshold", 2.0, "Z-score threshold for outlier detection")
	format := fs.String("format", "ascii", "Output format: json or ascii")
	showRate := fs.Bool("show-rate", false, "Include rate-of-change analysis")
	limit := fs.Int("limit", 10, "Maximum number of outliers to show")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *taskID == "" {
		fmt.Fprintf(os.Stderr, "Error: --task is required\n")
		fmt.Fprintf(os.Stderr, "Usage: ailang observatory outliers --task TASK_ID [options]\n")
		os.Exit(1)
	}

	ctx := context.Background()
	dbPath := observatory.DefaultDatabasePath()

	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	opts := observatory.OutlierOptions{
		Threshold: *threshold,
		Metric:    *metric,
		ShowRate:  *showRate,
		Limit:     *limit,
	}

	analysis, err := observatory.AnalyzeTaskOutliers(ctx, backend, *taskID, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing task: %v\n", err)
		os.Exit(1)
	}

	// Build CLI command for display
	cliCmd := fmt.Sprintf("ailang observatory outliers --task %s --threshold %.1f", *taskID, *threshold)
	if *metric != "" {
		cliCmd += fmt.Sprintf(" --metric %s", *metric)
	}
	if *showRate {
		cliCmd += " --show-rate"
	}
	if *limit != 10 {
		cliCmd += fmt.Sprintf(" --limit %d", *limit)
	}

	if *format == "ascii" {
		printOutliersASCII(analysis, cliCmd)
	} else {
		output := map[string]interface{}{
			"analysis":    analysis,
			"cli_command": cliCmd + " --format json",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(output)
	}
}

// printOutliersASCII formats outlier analysis for terminal display.
func printOutliersASCII(analysis *observatory.OutlierAnalysis, cliCmd string) {
	fmt.Println("=== Span Outlier Analysis ===")
	fmt.Println(strings.Repeat("═", 70))
	fmt.Printf("Task: %s (%s)\n", analysis.TaskID, analysis.TaskTitle)
	fmt.Printf("Spans: %d | Threshold: %.1fσ\n", analysis.SpanCount, analysis.Threshold)
	fmt.Println(strings.Repeat("─", 70))

	// Print metric statistics
	fmt.Println("\nMetric Statistics:")
	fmt.Printf("%-12s %8s %12s %12s %12s\n", "", "Count", "Sum", "Mean", "StdDev")
	for _, stat := range analysis.Stats {
		var sumStr, meanStr, stdStr string
		switch stat.Metric {
		case "cost_usd":
			sumStr = fmt.Sprintf("$%.4f", stat.Sum)
			meanStr = fmt.Sprintf("$%.4f", stat.Mean)
			stdStr = fmt.Sprintf("$%.4f", stat.StdDev)
		case "duration_ms":
			sumStr = formatDurationMs(stat.Sum)
			meanStr = formatDurationMs(stat.Mean)
			stdStr = formatDurationMs(stat.StdDev)
		case "tokens":
			sumStr = formatTokenCount(stat.Sum)
			meanStr = formatTokenCount(stat.Mean)
			stdStr = formatTokenCount(stat.StdDev)
		}
		fmt.Printf("  %-10s %8d %12s %12s %12s\n", stat.Metric, stat.Count, sumStr, meanStr, stdStr)
	}

	// Print outliers
	if len(analysis.Outliers) == 0 {
		fmt.Println("\nNo outliers detected (all spans within threshold).")
	} else {
		fmt.Printf("\nOutliers Detected (%d spans):\n", len(analysis.Outliers))
		fmt.Println(strings.Repeat("─", 70))
		for i, outlier := range analysis.Outliers {
			var valueStr string
			switch outlier.Metric {
			case "cost_usd":
				valueStr = fmt.Sprintf("$%.4f", outlier.Value)
			case "duration_ms":
				valueStr = formatDurationMs(outlier.Value)
			case "tokens":
				valueStr = formatTokenCount(outlier.Value)
			}
			fmt.Printf("  #%d %s\n", i+1, outlier.Span.Name)
			fmt.Printf("     %s: %s (z=%.2f, %.1f%% of total)\n",
				outlier.Metric, valueStr, outlier.ZScore, outlier.PercentOfTotal)
			if outlier.Span.Model != "" {
				fmt.Printf("     Model: %s | Provider: %s\n", outlier.Span.Model, outlier.Span.Provider)
			}
		}
	}

	// Print rate of change if available
	if analysis.RateOfChange != nil {
		fmt.Println("\nRate of Change (Top Contributors):")
		fmt.Println(strings.Repeat("─", 70))

		// Show top cost contributors
		if len(analysis.RateOfChange.CumulativeCost) > 0 {
			fmt.Println("  Cost:")
			printTopContributors(analysis.RateOfChange.CumulativeCost, "cost")
		}
		if len(analysis.RateOfChange.CumulativeTokens) > 0 {
			fmt.Println("  Tokens:")
			printTopContributors(analysis.RateOfChange.CumulativeTokens, "tokens")
		}
		if len(analysis.RateOfChange.CumulativeDuration) > 0 {
			fmt.Println("  Duration:")
			printTopContributors(analysis.RateOfChange.CumulativeDuration, "duration")
		}
	}

	fmt.Println()
	cliCmd += " --format ascii"
	fmt.Printf("CLI: %s\n", cliCmd)
}

// printTopContributors shows the top spans contributing to a metric.
func printTopContributors(points []observatory.CumulativePoint, metricType string) {
	// Sort by delta percent descending to find top contributors
	sorted := make([]observatory.CumulativePoint, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].DeltaPercent > sorted[j].DeltaPercent
	})

	// Show top 3
	limit := 3
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for i := 0; i < limit; i++ {
		p := sorted[i]
		var valueStr string
		switch metricType {
		case "cost":
			valueStr = fmt.Sprintf("$%.4f", p.Value)
		case "tokens":
			valueStr = formatTokenCount(p.Value)
		case "duration":
			valueStr = formatDurationMs(p.Value)
		}
		fmt.Printf("    • %s: %s (%.1f%%)\n", truncateSpanName(p.SpanName, 35), valueStr, p.DeltaPercent)
	}
}

// formatDurationMs formats milliseconds for display.
func formatDurationMs(ms float64) string {
	if ms >= 60000 {
		return fmt.Sprintf("%.1fm", ms/60000)
	} else if ms >= 1000 {
		return fmt.Sprintf("%.1fs", ms/1000)
	}
	return fmt.Sprintf("%.0fms", ms)
}

// formatTokenCount formats token counts for display.
func formatTokenCount(tokens float64) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", tokens/1000000)
	} else if tokens >= 1000 {
		return fmt.Sprintf("%.1fK", tokens/1000)
	}
	return fmt.Sprintf("%.0f", tokens)
}

// truncateSpanName truncates a string to max length with ellipsis.
func truncateSpanName(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// observatoryHierarchyCommand displays unified task/message/span hierarchy.
// Auto-detects ID type from the input argument.
func observatoryHierarchyCommand() {
	fs := flag.NewFlagSet("observatory hierarchy", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	includeSpans := fs.Bool("spans", true, "Include individual spans in output")
	groupByTurns := fs.Bool("turns", false, "Group spans by conversation turn (Session → Turn 1 → Turn 2 → ...)")

	// Skip "ailang observatory hierarchy" args
	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Get ID from positional argument
	if fs.NArg() < 1 {
		fmt.Println("Usage: ailang observatory hierarchy [options] <id>")
		fmt.Println()
		fmt.Println("The <id> argument auto-detects type:")
		fmt.Println("  task-XXXXXXXX   Coordinator task ID")
		fmt.Println("  32-char hex     Trace ID (e.g., 0ebf5e64bb654fcc1d19256b59f05ae3)")
		fmt.Println("  UUID            Session ID (e.g., 4df60536-caed-4e2f-af2c-e386c361f4e7)")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -json           Output as JSON")
		fmt.Println("  -spans=false    Hide individual span details")
		fmt.Println("  -turns          Group spans by conversation turn")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang observatory hierarchy task-29404032")
		fmt.Println("  ailang observatory hierarchy -json task-29404032")
		fmt.Println("  ailang observatory hierarchy -turns task-29404032")
		return
	}

	id := fs.Arg(0)
	idType := detectIDType(id)

	ctx := context.Background()

	// Open observatory database
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening observatory database: %v\n", err)
		fmt.Fprintf(os.Stderr, "Database path: %s\n", dbPath)
		os.Exit(1)
	}
	defer backend.Close()

	// Build unified hierarchy based on ID type
	result, err := buildUnifiedHierarchy(ctx, backend, id, idType, *includeSpans)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building hierarchy: %v\n", err)
		os.Exit(1)
	}

	// Apply turn grouping if requested
	if *groupByTurns && result.SpanNodes != nil {
		result.TurnGrouped = observatory.GroupSpansByTurn(result.SpanNodes)
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Pretty print hierarchy
	if *groupByTurns && result.TurnGrouped != nil {
		printTurnGroupedHierarchy(result)
	} else {
		printUnifiedHierarchy(result)
	}
}

// IDType represents the detected type of an ID
type IDType string

const (
	IDTypeTask    IDType = "task"
	IDTypeTrace   IDType = "trace"
	IDTypeSession IDType = "session"
	IDTypeUnknown IDType = "unknown"
)

// detectIDType determines the type of ID from its format
func detectIDType(id string) IDType {
	// task-XXXXXXXX format
	if strings.HasPrefix(id, "task-") {
		return IDTypeTask
	}

	// 32-char hex = trace_id
	if len(id) == 32 && isHexString(id) {
		return IDTypeTrace
	}

	// UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
	if len(id) == 36 && strings.Count(id, "-") == 4 {
		return IDTypeSession
	}

	// eval-timestamp format
	if strings.HasPrefix(id, "eval-") {
		return IDTypeTask // treat as task-like
	}

	return IDTypeUnknown
}

// isHexString checks if a string contains only hex characters
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// UnifiedHierarchy represents the complete hierarchy for display
type UnifiedHierarchy struct {
	IDType      IDType                            `json:"id_type"`
	ID          string                            `json:"id"`
	Task        *unifiedTask                      `json:"task,omitempty"`
	Message     *unifiedMessage                   `json:"message,omitempty"`
	Spans       []*unifiedSpan                    `json:"spans,omitempty"`
	SpanNodes   []*observatory.SpanNode           `json:"-"` // Raw span nodes for grouping (not serialized)
	TurnGrouped *observatory.TurnGroupedHierarchy `json:"turn_grouped,omitempty"`
	Handoffs    []*unifiedHandoff                 `json:"handoffs,omitempty"`
	ParentChain []*unifiedTaskSummary             `json:"parent_chain,omitempty"`
	Stats       *unifiedStats                     `json:"stats"`
}

type unifiedTask struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	AgentID        string  `json:"agent_id,omitempty"`
	Status         string  `json:"status"`
	ParentTaskID   string  `json:"parent_task_id,omitempty"`
	Cost           float64 `json:"cost"`
	DurationMs     int64   `json:"duration_ms"`
	TokensIn       int64   `json:"tokens_in"`
	TokensOut      int64   `json:"tokens_out"`
	ApprovalStatus string  `json:"approval_status,omitempty"`
}

type unifiedMessage struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	FromAgent    string `json:"from_agent"`
	ToInbox      string `json:"to_inbox"`
	ParentTaskID string `json:"parent_task_id,omitempty"`
	Status       string `json:"status"`
}

type unifiedSpan struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	DurationMs int64          `json:"duration_ms"`
	Cost       float64        `json:"cost,omitempty"`
	TokensIn   int64          `json:"tokens_in,omitempty"`
	TokensOut  int64          `json:"tokens_out,omitempty"`
	Status     string         `json:"status"`
	Children   []*unifiedSpan `json:"children,omitempty"`
}

type unifiedHandoff struct {
	TargetTaskID string `json:"target_task_id"`
	TargetAgent  string `json:"target_agent"`
	Status       string `json:"status"`
}

type unifiedTaskSummary struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	AgentID string `json:"agent_id,omitempty"`
	Status  string `json:"status"`
}

type unifiedStats struct {
	TotalSpans       int     `json:"total_spans"`
	TotalCost        float64 `json:"total_cost"`
	TotalTokens      int64   `json:"total_tokens"`
	TotalDurationMs  int64   `json:"total_duration_ms"`
	HandoffCount     int     `json:"handoff_count"`
	PendingApprovals int     `json:"pending_approvals"`
}

// buildUnifiedHierarchy builds the complete hierarchy based on ID type
func buildUnifiedHierarchy(ctx context.Context, backend observatory.Backend, id string, idType IDType, includeSpans bool) (*UnifiedHierarchy, error) {
	result := &UnifiedHierarchy{
		IDType: idType,
		ID:     id,
		Stats:  &unifiedStats{},
	}

	switch idType {
	case IDTypeTask:
		return buildHierarchyFromTask(ctx, backend, id, includeSpans)
	case IDTypeTrace:
		return buildHierarchyFromTrace(ctx, backend, id, includeSpans)
	case IDTypeSession:
		return buildHierarchyFromSession(ctx, backend, id, includeSpans)
	default:
		return result, fmt.Errorf("unknown ID type for: %s", id)
	}
}

// buildHierarchyFromTask builds hierarchy starting from a task ID
func buildHierarchyFromTask(ctx context.Context, backend observatory.Backend, taskID string, includeSpans bool) (*UnifiedHierarchy, error) {
	result := &UnifiedHierarchy{
		IDType: IDTypeTask,
		ID:     taskID,
		Stats:  &unifiedStats{},
	}

	// Get task hierarchy using existing function
	opts := observatory.HierarchyOptions{
		IncludeSpans: includeSpans,
	}
	hierarchy, err := observatory.GetTaskHierarchy(ctx, backend, taskID, opts)
	if err != nil {
		return nil, fmt.Errorf("get task hierarchy: %w", err)
	}

	if hierarchy.Task != nil {
		result.Task = &unifiedTask{
			ID:           hierarchy.Task.ID,
			Title:        hierarchy.Task.Title,
			Status:       string(hierarchy.Task.Status),
			ParentTaskID: hierarchy.Task.ParentTaskID,
			Cost:         hierarchy.Task.TotalCostUSD,
			TokensIn:     hierarchy.Task.TotalTokensIn,
			TokensOut:    hierarchy.Task.TotalTokensOut,
		}
		result.Stats.TotalCost = hierarchy.Task.TotalCostUSD
		result.Stats.TotalTokens = hierarchy.Task.TotalTokensIn + hierarchy.Task.TotalTokensOut
	}

	// Build span tree from agents
	if includeSpans {
		for _, agent := range hierarchy.Agents {
			if result.Task != nil && result.Task.AgentID == "" && agent.Agent != nil {
				result.Task.AgentID = agent.Agent.AgentID
			}
			for _, traceHierarchy := range agent.Traces {
				for _, spanNode := range traceHierarchy.Spans {
					result.Spans = append(result.Spans, convertSpanNode(spanNode))
					result.SpanNodes = append(result.SpanNodes, spanNode) // Keep raw nodes for turn grouping
					result.Stats.TotalSpans++
				}
			}
		}
	}

	// Get child tasks (handoffs) from observatory tasks table
	childTasks, err := backend.ListTasks(ctx, observatory.TaskListOptions{
		ParentTaskID: taskID,
		Limit:        100,
	})
	if err == nil {
		for _, child := range childTasks {
			handoff := &unifiedHandoff{
				TargetTaskID: child.ID,
				Status:       string(child.Status),
			}
			result.Handoffs = append(result.Handoffs, handoff)
			result.Stats.HandoffCount++
		}
	}

	// Build parent chain (walk up parent_task_id)
	if hierarchy.Task != nil && hierarchy.Task.ParentTaskID != "" {
		result.ParentChain = buildParentChain(ctx, backend, hierarchy.Task.ParentTaskID)
	}

	return result, nil
}

// buildParentChain walks up the parent_task_id chain
func buildParentChain(ctx context.Context, backend observatory.Backend, parentTaskID string) []*unifiedTaskSummary {
	var chain []*unifiedTaskSummary
	currentID := parentTaskID
	visited := make(map[string]bool)

	for currentID != "" && !visited[currentID] && len(chain) < 10 {
		visited[currentID] = true
		task, err := backend.GetTask(ctx, currentID)
		if err != nil || task == nil {
			break
		}
		chain = append(chain, &unifiedTaskSummary{
			ID:     task.ID,
			Title:  task.Title,
			Status: string(task.Status),
		})
		currentID = task.ParentTaskID
	}

	return chain
}

// buildHierarchyFromTrace builds hierarchy from a trace ID
func buildHierarchyFromTrace(ctx context.Context, backend observatory.Backend, traceID string, includeSpans bool) (*UnifiedHierarchy, error) {
	result := &UnifiedHierarchy{
		IDType: IDTypeTrace,
		ID:     traceID,
		Stats:  &unifiedStats{},
	}

	// Get spans for this trace
	spans, err := backend.ListSpans(ctx, observatory.SpanListOptions{
		TraceID: traceID,
		Limit:   1000,
	})
	if err != nil {
		return nil, fmt.Errorf("list spans by trace: %w", err)
	}

	if len(spans) == 0 {
		return result, nil
	}

	// Build span tree
	if includeSpans {
		result.Spans = buildSpanTreeFromList(spans)
		result.SpanNodes = buildSpanNodeTreeFromList(spans) // For turn grouping
		result.Stats.TotalSpans = len(spans)
	}

	// Calculate stats
	for _, span := range spans {
		result.Stats.TotalCost += span.CostUSD
		result.Stats.TotalTokens += span.TokensIn + span.TokensOut
		if span.DurationMs > result.Stats.TotalDurationMs {
			result.Stats.TotalDurationMs = span.DurationMs
		}
	}

	// Try to find linked task
	for _, span := range spans {
		if span.TaskID != "" {
			// Found a task link - fetch task info
			task, err := backend.GetTask(ctx, span.TaskID)
			if err == nil && task != nil {
				result.Task = &unifiedTask{
					ID:           task.ID,
					Title:        task.Title,
					Status:       string(task.Status),
					ParentTaskID: task.ParentTaskID,
				}
				break
			}
		}
	}

	return result, nil
}

// buildHierarchyFromSession builds hierarchy from a session UUID
func buildHierarchyFromSession(ctx context.Context, backend observatory.Backend, sessionID string, includeSpans bool) (*UnifiedHierarchy, error) {
	result := &UnifiedHierarchy{
		IDType: IDTypeSession,
		ID:     sessionID,
		Stats:  &unifiedStats{},
	}

	// For session IDs, we query spans by the session.id attribute
	// The session ID might also be stored as task_id for Claude Code sessions
	spans, err := backend.ListSpans(ctx, observatory.SpanListOptions{
		TaskID: sessionID, // Claude Code uses session ID as task_id
		Limit:  1000,
	})
	if err != nil {
		return nil, fmt.Errorf("list spans by session: %w", err)
	}

	if len(spans) == 0 {
		return result, nil
	}

	// Build span tree
	if includeSpans {
		result.Spans = buildSpanTreeFromList(spans)
		result.SpanNodes = buildSpanNodeTreeFromList(spans) // For turn grouping
		result.Stats.TotalSpans = len(spans)
	}

	// Calculate stats
	for _, span := range spans {
		result.Stats.TotalCost += span.CostUSD
		result.Stats.TotalTokens += span.TokensIn + span.TokensOut
		if span.DurationMs > result.Stats.TotalDurationMs {
			result.Stats.TotalDurationMs = span.DurationMs
		}
	}

	return result, nil
}

// buildSpanNodeTreeFromList converts a flat span list to observatory.SpanNode tree.
// This is needed for turn-based grouping which uses the observatory types.
func buildSpanNodeTreeFromList(spans []*observatory.Span) []*observatory.SpanNode {
	if len(spans) == 0 {
		return nil
	}

	// Build node map
	nodeMap := make(map[string]*observatory.SpanNode)
	for _, span := range spans {
		nodeMap[span.ID] = &observatory.SpanNode{Span: span}
	}

	// Build parent-child relationships
	var roots []*observatory.SpanNode
	for _, span := range spans {
		node := nodeMap[span.ID]
		if span.ParentSpanID == "" {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[span.ParentSpanID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// Parent not in our set, treat as root
			roots = append(roots, node)
		}
	}

	return roots
}

// buildSpanTreeFromList converts a flat span list to a tree
func buildSpanTreeFromList(spans []*observatory.Span) []*unifiedSpan {
	if len(spans) == 0 {
		return nil
	}

	// Build node map
	nodeMap := make(map[string]*unifiedSpan)
	for _, span := range spans {
		nodeMap[span.ID] = &unifiedSpan{
			ID:         span.ID,
			Name:       span.Name,
			DurationMs: span.DurationMs,
			Cost:       span.CostUSD,
			TokensIn:   span.TokensIn,
			TokensOut:  span.TokensOut,
			Status:     string(span.Status),
		}
	}

	// Build parent-child relationships
	var roots []*unifiedSpan
	for _, span := range spans {
		node := nodeMap[span.ID]
		if span.ParentSpanID == "" {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[span.ParentSpanID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// Parent not in our set, treat as root
			roots = append(roots, node)
		}
	}

	return roots
}

// convertSpanNode converts observatory SpanNode to unifiedSpan
func convertSpanNode(node *observatory.SpanNode) *unifiedSpan {
	if node == nil || node.Span == nil {
		return nil
	}

	result := &unifiedSpan{
		ID:         node.Span.ID,
		Name:       node.Span.Name,
		DurationMs: node.Span.DurationMs,
		Cost:       node.Span.CostUSD,
		TokensIn:   node.Span.TokensIn,
		TokensOut:  node.Span.TokensOut,
		Status:     string(node.Span.Status),
	}

	for _, child := range node.Children {
		if converted := convertSpanNode(child); converted != nil {
			result.Children = append(result.Children, converted)
		}
	}

	return result
}

// printUnifiedHierarchy prints the hierarchy as a tree
func printUnifiedHierarchy(h *UnifiedHierarchy) {
	fmt.Println(strings.Repeat("═", 80))
	fmt.Printf("Hierarchy for %s (%s)\n", h.ID, h.IDType)
	fmt.Println(strings.Repeat("═", 80))

	// Print task info
	if h.Task != nil {
		agentInfo := ""
		if h.Task.AgentID != "" {
			agentInfo = fmt.Sprintf(" (%s)", h.Task.AgentID)
		}
		statusBadge := fmt.Sprintf(" [%s]", h.Task.Status)

		fmt.Printf("\n⬢ Task: %s%s%s\n", h.Task.ID, agentInfo, statusBadge)
		if h.Task.Title != "" && h.Task.Title != h.Task.ID {
			fmt.Printf("  Title: %s\n", truncateString(h.Task.Title, 70))
		}
		if h.Task.ParentTaskID != "" {
			fmt.Printf("  Parent: %s\n", h.Task.ParentTaskID)
		}
		fmt.Printf("  Cost: $%.4f | Tokens: %d in / %d out\n",
			h.Task.Cost, h.Task.TokensIn, h.Task.TokensOut)
	}

	// Print parent chain (ancestry)
	if len(h.ParentChain) > 0 {
		fmt.Printf("\n┌─ Parent Chain (%d levels):\n", len(h.ParentChain))
		for i, parent := range h.ParentChain {
			var prefix string
			if i == len(h.ParentChain)-1 {
				prefix = "└─ "
			} else {
				prefix = "├─ "
			}
			fmt.Printf("%s%s [%s]\n", prefix, parent.ID, parent.Status)
		}
	}

	// Print message info
	if h.Message != nil {
		fmt.Printf("\n📨 Message: %s\n", h.Message.ID)
		fmt.Printf("   From: %s → To: %s\n", h.Message.FromAgent, h.Message.ToInbox)
		if h.Message.ParentTaskID != "" {
			fmt.Printf("   Parent Task: %s\n", h.Message.ParentTaskID)
		}
	}

	// Print spans
	if len(h.Spans) > 0 {
		fmt.Printf("\n├─ Spans: %d total\n", h.Stats.TotalSpans)
		for i, span := range h.Spans {
			isLast := i == len(h.Spans)-1 && len(h.Handoffs) == 0
			printSpanTree(span, "│  ", isLast)
		}
	}

	// Print handoffs
	if len(h.Handoffs) > 0 {
		fmt.Printf("\n└─ Handoffs: %d\n", len(h.Handoffs))
		for i, handoff := range h.Handoffs {
			prefix := "   ├─ "
			if i == len(h.Handoffs)-1 {
				prefix = "   └─ "
			}
			agent := handoff.TargetAgent
			if agent == "" {
				agent = "?"
			}
			fmt.Printf("%s→ %s (%s) [%s]\n", prefix, handoff.TargetTaskID, agent, handoff.Status)
		}
	}

	// Print stats summary
	fmt.Println()
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("Stats: %d spans | $%.4f | %d tokens | %s\n",
		h.Stats.TotalSpans,
		h.Stats.TotalCost,
		h.Stats.TotalTokens,
		formatHierarchyDuration(h.Stats.TotalDurationMs))
	if h.Stats.HandoffCount > 0 {
		fmt.Printf("       %d handoffs", h.Stats.HandoffCount)
		if h.Stats.PendingApprovals > 0 {
			fmt.Printf(" | %d pending approvals", h.Stats.PendingApprovals)
		}
		fmt.Println()
	}
}

// printTurnGroupedHierarchy prints the hierarchy organized by conversation turns.
func printTurnGroupedHierarchy(h *UnifiedHierarchy) {
	fmt.Println(strings.Repeat("═", 80))
	fmt.Printf("Turn-Grouped Hierarchy for %s (%s)\n", h.ID, h.IDType)
	fmt.Println(strings.Repeat("═", 80))

	// Print task info if available
	if h.Task != nil {
		agentInfo := ""
		if h.Task.AgentID != "" {
			agentInfo = fmt.Sprintf(" (%s)", h.Task.AgentID)
		}
		statusBadge := fmt.Sprintf(" [%s]", h.Task.Status)
		fmt.Printf("\n⬢ Task: %s%s%s\n", h.Task.ID, agentInfo, statusBadge)
		if h.Task.Title != "" && h.Task.Title != h.Task.ID {
			fmt.Printf("  Title: %s\n", truncateString(h.Task.Title, 70))
		}
	}

	tg := h.TurnGrouped
	if tg == nil {
		fmt.Println("\n(No turn data available)")
		return
	}

	// Print session info
	if tg.Session != nil {
		fmt.Printf("\n┌─ Session: %s\n", tg.Session.Name)
		if tg.Session.Provider != "" || tg.Session.Model != "" {
			fmt.Printf("│  Provider: %s | Model: %s\n", tg.Session.Provider, tg.Session.Model)
		}
		fmt.Printf("│  Duration: %s | Cost: $%.4f | Tokens: %d in / %d out\n",
			formatHierarchyDuration(tg.Session.DurationMs),
			tg.Session.Cost,
			tg.Session.TokensIn,
			tg.Session.TokensOut)
	}

	// Print turns
	if len(tg.Turns) > 0 {
		fmt.Printf("│\n")
		for i, turn := range tg.Turns {
			isLast := i == len(tg.Turns)-1

			// Turn header
			prefix := "├"
			nextPrefix := "│"
			if isLast {
				prefix = "└"
				nextPrefix = " "
			}

			fmt.Printf("%s─ Turn %d [%s] $%.4f [%d→%d tokens]\n",
				prefix,
				turn.TurnNumber,
				formatHierarchyDuration(turn.DurationMs),
				turn.Cost,
				turn.TokensIn,
				turn.TokensOut)

			// Print tools within turn
			if len(turn.Tools) > 0 {
				for j, tool := range turn.Tools {
					toolIsLast := j == len(turn.Tools)-1
					toolPrefix := "├─"
					if toolIsLast {
						toolPrefix = "└─"
					}

					statusIcon := "✓"
					if tool.Status == "error" {
						statusIcon = "✗"
					}

					toolName := tool.ToolName
					if toolName == "" {
						toolName = tool.Name
					}

					fmt.Printf("%s  %s %s %s [%s]\n",
						nextPrefix,
						toolPrefix,
						toolName,
						statusIcon,
						formatHierarchyDuration(tool.DurationMs))
				}
			}

			// Add spacing between turns
			if !isLast {
				fmt.Printf("%s\n", nextPrefix)
			}
		}
	}

	// Print stats summary
	fmt.Println()
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("Stats: %d turns | %d tools | $%.4f | %d tokens | %s\n",
		tg.Stats.TotalTurns,
		tg.Stats.TotalTools,
		tg.Stats.TotalCost,
		tg.Stats.TotalTokens,
		formatHierarchyDuration(tg.Stats.DurationMs))

	// Print handoffs if any
	if len(h.Handoffs) > 0 {
		fmt.Printf("       %d handoffs\n", len(h.Handoffs))
	}
}

// printSpanTree prints a span and its children as a tree
func printSpanTree(span *unifiedSpan, prefix string, isLast bool) {
	if span == nil {
		return
	}

	connector := "├─ "
	if isLast {
		connector = "└─ "
	}

	// Format metrics
	metrics := ""
	if span.DurationMs > 0 {
		metrics += fmt.Sprintf(" %s", formatHierarchyDuration(span.DurationMs))
	}
	if span.Cost > 0 {
		metrics += fmt.Sprintf(" $%.4f", span.Cost)
	}
	if span.TokensIn > 0 || span.TokensOut > 0 {
		metrics += fmt.Sprintf(" [%d→%d]", span.TokensIn, span.TokensOut)
	}

	fmt.Printf("%s%s%s%s\n", prefix, connector, span.Name, metrics)

	// Print children
	childPrefix := prefix
	if isLast {
		childPrefix += "   "
	} else {
		childPrefix += "│  "
	}

	for i, child := range span.Children {
		printSpanTree(child, childPrefix, i == len(span.Children)-1)
	}
}

// formatHierarchyDuration formats milliseconds as human-readable duration for hierarchy output
func formatHierarchyDuration(ms int64) string {
	if ms >= 60000 {
		return fmt.Sprintf("%.1fm", float64(ms)/60000)
	} else if ms >= 1000 {
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	}
	return fmt.Sprintf("%dms", ms)
}

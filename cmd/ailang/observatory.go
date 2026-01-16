package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

func observatoryCommand() {
	if flag.NArg() < 2 {
		fmt.Println("Usage: ailang observatory <subcommand> [options]")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  seed        Generate test data for dashboard development")
		fmt.Println("  backfill    Link existing spans to tasks by time correlation")
		fmt.Println("  heatmap     Get activity heatmap data (daily aggregates)")
		fmt.Println("  evolution   Get task evolution data (cumulative metrics over time)")
		fmt.Println("  usage       Get usage time series data (bucketed by hour/day/week)")
		fmt.Println("  tokens      Get token distribution histogram")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang observatory seed                    # Default: realistic workload")
		fmt.Println("  ailang observatory seed --minimal          # Quick: 1 workspace, 2 tasks")
		fmt.Println("  ailang observatory heatmap --days 30 --format ascii")
		fmt.Println("  ailang observatory evolution --metric cost --limit 5")
		fmt.Println("  ailang observatory usage --interval day --split-by provider")
		fmt.Println("  ailang observatory tokens --format ascii")
		return
	}

	subcommand := flag.Arg(1)
	switch subcommand {
	case "seed":
		observatorySeedCommand()
	case "backfill":
		observatoryBackfillCommand()
	case "heatmap":
		observatoryHeatmapCommand()
	case "evolution":
		observatoryEvolutionCommand()
	case "usage":
		observatoryUsageCommand()
	case "tokens":
		observatoryTokensCommand()
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
	days := fs.Int("days", 90, "Number of days of history to show")
	format := fs.String("format", "json", "Output format: json or ascii")
	provider := fs.String("provider", "", "Filter by provider")
	workspace := fs.String("workspace", "", "Filter by workspace")

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

	startDate := time.Now().AddDate(0, 0, -*days)

	query := `
		SELECT
			date(start_time) as day,
			COUNT(*) as span_count,
			SUM(COALESCE(cost_usd, 0)) as total_cost,
			SUM(COALESCE(tokens_in, 0) + COALESCE(tokens_out, 0)) as total_tokens
		FROM spans
		WHERE start_time >= ?
	`
	args := []interface{}{startDate.Format("2006-01-02")}

	if *provider != "" {
		query += " AND provider = ?"
		args = append(args, *provider)
	}
	if *workspace != "" {
		query += " AND workspace = ?"
		args = append(args, *workspace)
	}

	query += " GROUP BY date(start_time) ORDER BY day"

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
		cliCmd := fmt.Sprintf("ailang observatory heatmap --days %d", *days)
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
		// JSON output
		output := map[string]interface{}{
			"days": data,
			"cli_command": fmt.Sprintf("ailang observatory heatmap --days %d --format json",
				*days),
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
	days := fs.Int("days", 30, "Number of days of history")

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

	startDate := time.Now().AddDate(0, 0, -*days)

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

	query := fmt.Sprintf(`
		SELECT
			%s as bucket,
			COUNT(*) as span_count,
			SUM(COALESCE(cost_usd, 0)) as total_cost,
			SUM(COALESCE(tokens_in, 0) + COALESCE(tokens_out, 0)) as total_tokens
		FROM spans
		WHERE start_time >= ?
		GROUP BY bucket
		ORDER BY bucket
	`, dateExpr)

	rows, err := db.QueryContext(ctx, query, startDate.Format("2006-01-02"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying database: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	type UsagePoint struct {
		Bucket    string  `json:"bucket"`
		SpanCount int     `json:"span_count"`
		Cost      float64 `json:"cost"`
		Tokens    int64   `json:"tokens"`
	}

	var points []UsagePoint
	for rows.Next() {
		var p UsagePoint
		if err := rows.Scan(&p.Bucket, &p.SpanCount, &p.Cost, &p.Tokens); err != nil {
			continue
		}
		points = append(points, p)
	}

	cliCmd := fmt.Sprintf("ailang observatory usage --metric %s --interval %s --days %d",
		*metric, *interval, *days)
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
	days := fs.Int("days", 30, "Number of days of history")

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

	startDate := time.Now().AddDate(0, 0, -*days)

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
		query := `
			SELECT COUNT(*) FROM spans
			WHERE start_time >= ?
			AND (COALESCE(tokens_in, 0) + COALESCE(tokens_out, 0)) >= ?
			AND (COALESCE(tokens_in, 0) + COALESCE(tokens_out, 0)) < ?
		`
		var count int
		err := db.QueryRowContext(ctx, query, startDate.Format("2006-01-02"), b.min, b.max).Scan(&count)
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

	cliCmd := fmt.Sprintf("ailang observatory tokens --days %d", *days)

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

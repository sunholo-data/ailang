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

func observatoryCommand() {
	if flag.NArg() < 2 {
		fmt.Println("Usage: ailang observatory <subcommand> [options]")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  seed        Generate test data for dashboard development")
		fmt.Println("  backfill    Link existing spans to tasks by time correlation")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang observatory seed                    # Default: realistic workload")
		fmt.Println("  ailang observatory seed --minimal          # Quick: 1 workspace, 2 tasks")
		fmt.Println("  ailang observatory seed --stress           # Load test: 10 workspaces, 1000 spans")
		fmt.Println("  ailang observatory seed --clean            # Wipe all data first")
		fmt.Println("  ailang observatory backfill --dry-run")
		fmt.Println("  ailang observatory backfill --window 5m")
		return
	}

	subcommand := flag.Arg(1)
	switch subcommand {
	case "seed":
		observatorySeedCommand()
	case "backfill":
		observatoryBackfillCommand()
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

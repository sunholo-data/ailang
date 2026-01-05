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
		fmt.Println("  backfill    Link existing spans to tasks by time correlation")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang observatory backfill --dry-run")
		fmt.Println("  ailang observatory backfill --window 5m")
		return
	}

	subcommand := flag.Arg(1)
	switch subcommand {
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

			// Find unlinked spans in this time window
			spans, err := backend.ListSpans(ctx, observatory.SpanListOptions{
				StartAfter:  startTime,
				StartBefore: endTime,
				Limit:       1000,
			})
			if err != nil {
				result.Errors++
				if opts.Verbose {
					fmt.Printf("  Error listing spans for assignment %s: %v\n", assignment.ID, err)
				}
				continue
			}

			for _, span := range spans {
				result.SpansScanned++

				// Skip already linked spans
				if span.TaskID != "" || span.AgentAssignmentID != "" {
					result.SpansAlreadyLinked++
					continue
				}

				// Check if span matches the agent (by provider or name patterns)
				if !spanMatchesAgent(span, assignment) {
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
				}

				result.SpansLinked++
			}
		}
	}

	return result, nil
}

// spanMatchesAgent checks if a span likely came from a specific agent assignment.
func spanMatchesAgent(span *observatory.Span, assignment *observatory.AgentAssignment) bool {
	// Match by provider
	if span.Provider != "" && string(span.Provider) == string(assignment.Provider) {
		return true
	}

	// Match by span name patterns
	switch assignment.Provider {
	case observatory.ProviderClaude:
		if strings.Contains(span.Name, "anthropic") || strings.Contains(span.Name, "claude") {
			return true
		}
	case observatory.ProviderGemini:
		if strings.Contains(span.Name, "gemini") || strings.Contains(span.Name, "google") {
			return true
		}
	}

	// Match by resource attributes
	if span.ResourceAttributes != nil {
		if serviceName, ok := span.ResourceAttributes["service.name"].(string); ok {
			if strings.Contains(serviceName, "claude") && assignment.Provider == observatory.ProviderClaude {
				return true
			}
			if strings.Contains(serviceName, "gemini") && assignment.Provider == observatory.ProviderGemini {
				return true
			}
		}
	}

	return false
}

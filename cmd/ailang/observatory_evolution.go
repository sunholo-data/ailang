package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/observatory"
)

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

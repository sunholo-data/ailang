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

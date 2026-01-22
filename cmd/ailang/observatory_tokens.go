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

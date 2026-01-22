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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

func observatoryMetricsCommand() {
	fs := flag.NewFlagSet("observatory metrics", flag.ExitOnError)

	// Mode flags (mutually exclusive)
	sessionID := fs.String("session", "", "Show metrics for a specific session ID")
	listMetrics := fs.Bool("list", false, "List raw OTLP metrics")

	// List filters
	name := fs.String("name", "", "Filter metrics by name (e.g., claude_code.lines_of_code.count)")
	workspace := fs.String("workspace", "", "Filter metrics by workspace")
	limit := fs.Int("limit", 50, "Maximum number of metrics to show")

	// Output format
	jsonOutput := fs.Bool("json", false, "Output in JSON format")

	fs.Usage = func() {
		fmt.Println("Usage: ailang observatory metrics [options]")
		fmt.Println()
		fmt.Println("View Claude Code telemetry metrics (LOC, commits, cache savings, etc.)")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang observatory metrics --session 4df60536-caed-4e2f-af2c-e386c361f4e7")
		fmt.Println("  ailang observatory metrics --list --limit 20")
		fmt.Println("  ailang observatory metrics --list --name claude_code.lines_of_code.count")
		fmt.Println("  ailang observatory metrics --list --json")
	}

	// Skip "ailang observatory metrics" args
	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Open database
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	// Determine mode
	if *sessionID != "" {
		showSessionMetrics(ctx, backend, *sessionID, *jsonOutput)
	} else if *listMetrics {
		listTelemetryMetrics(ctx, backend, *name, *workspace, *sessionID, *limit, *jsonOutput)
	} else {
		// Default: show recent sessions summary
		showRecentSessionsSummary(ctx, backend, *limit, *jsonOutput)
	}
}

func showSessionMetrics(ctx context.Context, backend *observatory.SQLiteBackend, sessionID string, jsonOutput bool) {
	summary, err := backend.GetSessionMetricsSummary(ctx, sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting session metrics: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(summary)
		return
	}

	// Human-readable output
	fmt.Printf("Session Metrics: %s\n", sessionID)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	// Token usage
	fmt.Println("Token Usage:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  Input tokens:\t%d\n", summary.TokensIn)
	fmt.Fprintf(w, "  Output tokens:\t%d\n", summary.TokensOut)
	fmt.Fprintf(w, "  Cache read tokens:\t%d\n", summary.CacheReadTokens)
	fmt.Fprintf(w, "  Cache creation tokens:\t%d\n", summary.CacheCreationTokens)
	w.Flush()
	fmt.Println()

	// Costs
	fmt.Println("Costs:")
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  Total cost:\t$%.4f\n", summary.TotalCostUSD)
	fmt.Fprintf(w, "  Cache savings:\t$%.4f\n", summary.CacheSavingsUSD)
	if summary.TotalCostUSD > 0 {
		savingsPercent := (summary.CacheSavingsUSD / (summary.TotalCostUSD + summary.CacheSavingsUSD)) * 100
		fmt.Fprintf(w, "  Savings %% of original:\t%.1f%%\n", savingsPercent)
	}
	w.Flush()
	fmt.Println()

	// Lines of Code
	if summary.LinesAdded > 0 || summary.LinesRemoved > 0 {
		fmt.Println("Lines of Code:")
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "  Lines added:\t%d\n", summary.LinesAdded)
		fmt.Fprintf(w, "  Lines removed:\t%d\n", summary.LinesRemoved)
		fmt.Fprintf(w, "  Net change:\t%+d\n", summary.LinesAdded-summary.LinesRemoved)
		w.Flush()
		fmt.Println()
	}

	// Activity
	fmt.Println("Activity:")
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  Turns:\t%d\n", summary.TurnCount)
	fmt.Fprintf(w, "  Tool calls:\t%d\n", summary.ToolCalls)
	fmt.Fprintf(w, "  Spans:\t%d\n", summary.SpanCount)
	fmt.Fprintf(w, "  Errors:\t%d\n", summary.ErrorCount)
	if summary.SpanCount > 0 {
		fmt.Fprintf(w, "  Success rate:\t%.1f%%\n", summary.SuccessRate*100)
	}
	w.Flush()
	fmt.Println()

	// Git activity
	if summary.CommitCount > 0 || summary.PullRequestCount > 0 {
		fmt.Println("Git Activity:")
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "  Commits:\t%d\n", summary.CommitCount)
		fmt.Fprintf(w, "  Pull requests:\t%d\n", summary.PullRequestCount)
		w.Flush()
		fmt.Println()
	}

	// Duration
	if summary.DurationMs > 0 {
		duration := time.Duration(summary.DurationMs) * time.Millisecond
		fmt.Printf("Duration: %s\n", formatMetricsDuration(duration))
	}
	if summary.ActiveTimeMs > 0 {
		activeTime := time.Duration(summary.ActiveTimeMs) * time.Millisecond
		fmt.Printf("Active time: %s\n", formatMetricsDuration(activeTime))
	}
}

func listTelemetryMetrics(ctx context.Context, backend *observatory.SQLiteBackend, name, workspace, sessionID string, limit int, jsonOutput bool) {
	opts := observatory.MetricListOptions{
		Name:      name,
		Workspace: workspace,
		SessionID: sessionID,
		Limit:     limit,
	}

	metrics, err := backend.ListMetrics(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing metrics: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]any{
			"metrics": metrics,
			"count":   len(metrics),
		})
		return
	}

	if len(metrics) == 0 {
		fmt.Println("No metrics found.")
		return
	}

	// Human-readable table
	fmt.Printf("OTLP Metrics (%d results)\n", len(metrics))
	fmt.Println(strings.Repeat("=", 80))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "NAME\tTYPE\tVALUE\tLABELS\tTIME\n")
	fmt.Fprintf(w, "----\t----\t-----\t------\t----\n")

	for _, m := range metrics {
		// Format value
		var value string
		if m.ValueInt != 0 {
			value = fmt.Sprintf("%d", m.ValueInt)
		} else if m.ValueFloat != 0 {
			value = fmt.Sprintf("%.2f", m.ValueFloat)
		} else {
			value = "0"
		}

		// Format labels
		var labels []string
		if m.LabelType != "" {
			labels = append(labels, fmt.Sprintf("type=%s", m.LabelType))
		}
		if m.LabelTool != "" {
			labels = append(labels, fmt.Sprintf("tool=%s", m.LabelTool))
		}
		if m.LabelLanguage != "" {
			labels = append(labels, fmt.Sprintf("lang=%s", m.LabelLanguage))
		}
		if m.LabelModel != "" {
			labels = append(labels, fmt.Sprintf("model=%s", m.LabelModel))
		}
		labelStr := strings.Join(labels, ",")
		if labelStr == "" {
			labelStr = "-"
		}

		// Format time (relative)
		timeStr := formatRelativeTime(m.Timestamp)

		// Truncate name if too long
		displayName := m.Name
		if len(displayName) > 35 {
			displayName = displayName[:32] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", displayName, m.Type, value, labelStr, timeStr)
	}
	w.Flush()
}

func showRecentSessionsSummary(ctx context.Context, backend *observatory.SQLiteBackend, limit int, jsonOutput bool) {
	// Query recent sessions from the database
	db := backend.DB()
	if db == nil {
		fmt.Fprintf(os.Stderr, "Error: database not available\n")
		os.Exit(1)
	}

	rows, err := db.QueryContext(ctx, `
		SELECT session_id, workspace, started_at, ended_at, turn_count
		FROM sessions
		ORDER BY started_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying sessions: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	type sessionInfo struct {
		SessionID string
		Workspace string
		StartedAt time.Time
		EndedAt   *time.Time
		TurnCount int
		Summary   *observatory.SessionMetricsSummary
	}

	var sessions []sessionInfo
	for rows.Next() {
		var s sessionInfo
		var endedAt *time.Time
		if err := rows.Scan(&s.SessionID, &s.Workspace, &s.StartedAt, &endedAt, &s.TurnCount); err != nil {
			continue
		}
		s.EndedAt = endedAt

		// Get summary for each session
		summary, err := backend.GetSessionMetricsSummary(ctx, s.SessionID)
		if err == nil {
			s.Summary = summary
		}
		sessions = append(sessions, s)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]any{
			"sessions": sessions,
			"count":    len(sessions),
		})
		return
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found. Start a Claude Code session with telemetry enabled:")
		fmt.Println("  CLAUDE_CODE_ENABLE_TELEMETRY=1 claude")
		return
	}

	fmt.Printf("Recent Sessions (%d)\n", len(sessions))
	fmt.Println(strings.Repeat("=", 100))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "SESSION ID (short)\tWORKSPACE\tTURNS\tTOKENS\tCOST\tLOC\tSTARTED\n")
	fmt.Fprintf(w, "------------------\t---------\t-----\t------\t----\t---\t-------\n")

	for _, s := range sessions {
		// Short session ID
		shortID := s.SessionID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}

		// Workspace name
		workspace := normalizeWorkspacePath(s.Workspace)
		if len(workspace) > 15 {
			workspace = workspace[:12] + "..."
		}

		// Metrics
		var tokens, cost, loc string
		if s.Summary != nil {
			totalTokens := s.Summary.TokensIn + s.Summary.TokensOut
			if totalTokens > 1000000 {
				tokens = fmt.Sprintf("%.1fM", float64(totalTokens)/1000000)
			} else if totalTokens > 1000 {
				tokens = fmt.Sprintf("%.1fK", float64(totalTokens)/1000)
			} else {
				tokens = fmt.Sprintf("%d", totalTokens)
			}
			cost = fmt.Sprintf("$%.3f", s.Summary.TotalCostUSD)
			netLOC := s.Summary.LinesAdded - s.Summary.LinesRemoved
			if netLOC != 0 {
				loc = fmt.Sprintf("%+d", netLOC)
			} else {
				loc = "-"
			}
		} else {
			tokens = "-"
			cost = "-"
			loc = "-"
		}

		timeStr := formatRelativeTime(s.StartedAt)

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
			shortID, workspace, s.TurnCount, tokens, cost, loc, timeStr)
	}
	w.Flush()

	fmt.Println()
	fmt.Println("Use --session <id> for detailed metrics of a specific session.")
}

func formatMetricsDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

func formatRelativeTime(t time.Time) string {
	diff := time.Since(t)
	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	} else if diff < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
	return t.Format("Jan 2")
}

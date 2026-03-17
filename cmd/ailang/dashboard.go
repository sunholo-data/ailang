package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Default dashboard server URL
const defaultDashboardURL = "http://localhost:1957"

// HTTP client with timeout for dashboard requests
var dashboardHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

// getDashboardURL returns the dashboard server URL from flag, env, or default
func getDashboardURL(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if envURL := os.Getenv("AILANG_DASHBOARD_URL"); envURL != "" {
		return envURL
	}
	return defaultDashboardURL
}

func dashboardCommand() {
	if flag.NArg() < 2 {
		fmt.Println("Usage: ailang dashboard <subcommand> [options]")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  spans      Query observatory spans with filters")
		fmt.Println("  inbox      Query unified inbox (messages + claude code events)")
		fmt.Println("  traces     Query trace summaries")
		fmt.Println("  hierarchy  Show exec task hierarchy (message → exec → turn → tool)")
		fmt.Println("  sessions   List Claude Code sessions with workspace info")
		fmt.Println("  tools      Show tool usage for a session (file paths, patterns)")
		fmt.Println("  stats      Query aggregation statistics")
		fmt.Println("  health     Check server health")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang dashboard spans --provider gemini")
		fmt.Println("  ailang dashboard spans --workspace /path/to/repo")
		fmt.Println("  ailang dashboard inbox --model gemini-2.5-flash")
		fmt.Println("  ailang dashboard inbox --status unread")
		fmt.Println("  ailang dashboard traces --trace-id abc123")
		fmt.Println("  ailang dashboard hierarchy --limit 10")
		fmt.Println("  ailang dashboard sessions --limit 10")
		fmt.Println("  ailang dashboard tools <session-id> --summary")
		fmt.Println("  ailang dashboard stats --start 2026-01-01")
		fmt.Println("  ailang dashboard health")
		fmt.Println()
		fmt.Println("Environment:")
		fmt.Println("  AILANG_DASHBOARD_URL  Server URL (default: http://localhost:1957)")
		return
	}

	subcommand := flag.Arg(1)
	switch subcommand {
	case "spans":
		dashboardSpansCommand()
	case "inbox":
		dashboardInboxCommand()
	case "traces":
		dashboardTracesCommand()
	case "hierarchy":
		dashboardHierarchyCommand()
	case "sessions":
		dashboardSessionsCommand()
	case "tools":
		dashboardToolsCommand()
	case "stats":
		dashboardStatsCommand()
	case "health":
		dashboardHealthCommand()
	default:
		fmt.Fprintf(os.Stderr, "Unknown dashboard subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

// Helper functions for dashboard commands

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func truncateID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12] + "..."
}

func formatDuration(ms float64) string {
	if ms == 0 {
		return "-"
	}
	if ms < 1000 {
		return fmt.Sprintf("%.0fms", ms)
	}
	return fmt.Sprintf("%.1fs", ms/1000)
}

func formatTimestampAge(timestamp string) string {
	if timestamp == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		// Try other formats
		t, err = time.Parse("2006-01-02T15:04:05Z", timestamp)
		if err != nil {
			return timestamp[:10] // Just date
		}
	}

	age := time.Since(t)
	if age < time.Minute {
		return fmt.Sprintf("%ds ago", int(age.Seconds()))
	}
	if age < time.Hour {
		return fmt.Sprintf("%dm ago", int(age.Minutes()))
	}
	if age < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(age.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(age.Hours()/24))
}

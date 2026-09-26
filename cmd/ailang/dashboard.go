package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
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
	if envURL := config.DashboardURL(); envURL != "" {
		return envURL
	}
	return defaultDashboardURL
}

// dashboardCommand is `ailang dashboard <subcommand>` — a CLI client for the
// dashboard server's HTTP API, not the web UI. (The web UI reads the same data
// through internal/server's own handlers; it never shells out to this binary.)
//
// M-V1-SIMPLIFY-S5 M3 folded it into `chains`: the canonical spelling is
// `ailang chains dashboard <subcommand>` and this one survives as an alias for
// one release (D1).
func dashboardCommand() {
	if flag.NArg() < 2 {
		printDashboardHelp()
		// Exit 1, not 0 — see the note in observatoryCommand.
		os.Exit(1)
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

func printDashboardHelp() {
	fmt.Println("Usage: ailang chains dashboard <subcommand> [options]")
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
	fmt.Println("  ailang chains dashboard spans --provider gemini")
	fmt.Println("  ailang chains dashboard spans --workspace /path/to/repo")
	fmt.Println("  ailang chains dashboard inbox --model gemini-2.5-flash")
	fmt.Println("  ailang chains dashboard inbox --status unread")
	fmt.Println("  ailang chains dashboard traces --trace-id abc123")
	fmt.Println("  ailang chains dashboard hierarchy --limit 10")
	fmt.Println("  ailang chains dashboard sessions --limit 10")
	fmt.Println("  ailang chains dashboard tools <session-id> --summary")
	fmt.Println("  ailang chains dashboard stats --start 2026-01-01")
	fmt.Println("  ailang chains dashboard health")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  AILANG_DASHBOARD_URL  Server URL (default: http://localhost:1957)")
	fmt.Println()
	fmt.Println("`ailang dashboard <subcommand>` remains an alias for one release.")
}

// Helper functions for dashboard commands

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getInt reads a numeric field however the map came to hold it: float64
// from encoding/json, int64 from a Firestore document or a json.Number
// from a decoder with UseNumber. It used to accept float64 only, so every
// other representation read as 0 (M-V1-SIMPLIFY-S3 M5).
func getInt(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
		if f, err := v.Float64(); err == nil {
			return int(f)
		}
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	}
	return 0
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

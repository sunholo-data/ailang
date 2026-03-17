package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// dashboardHierarchyCommand shows exec task hierarchy.
func dashboardHierarchyCommand() {
	fs := flag.NewFlagSet("dashboard hierarchy", flag.ExitOnError)
	server := fs.String("server", "", "Dashboard server URL")
	limit := fs.Int("limit", 50, "Maximum exec tasks to show")
	includeMessages := fs.Bool("messages", true, "Group by triggering messages (4-level hierarchy)")
	jsonOutput := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	baseURL := getDashboardURL(*server)
	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", *limit))
	if *includeMessages {
		params.Set("include_messages", "true")
	}

	apiURL := fmt.Sprintf("%s/api/controlplane/exec-hierarchy?%s", baseURL, params.Encode())
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to dashboard: %v\n", err)
		fmt.Fprintf(os.Stderr, "Is the server running? Start with: ailang serve\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error from server: %s\n%s\n", resp.Status, string(body))
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		// Pretty print JSON
		var parsed interface{}
		json.Unmarshal(body, &parsed)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(parsed)
		return
	}

	// Parse and display hierarchy
	if *includeMessages {
		displayMessageHierarchy(body)
	} else {
		displayExecHierarchy(body)
	}
}

// displayMessageHierarchy shows message-grouped hierarchy.
func displayMessageHierarchy(body []byte) {
	var result struct {
		Messages []struct {
			MessageID string `json:"message_id"`
			Title     string `json:"title"`
			FromAgent string `json:"from_agent"`
			CreatedAt string `json:"created_at"`
			Execs     []struct {
				TaskID     string `json:"task_id"`
				Command    string `json:"command"`
				Provider   string `json:"provider"`
				Status     string `json:"status"`
				DurationMs int    `json:"duration_ms"`
				Children   []struct {
					Command    string `json:"command"`
					TurnNumber int    `json:"turn_number,omitempty"`
					ToolName   string `json:"tool_name,omitempty"`
					Status     string `json:"status"`
					DurationMs int    `json:"duration_ms"`
				} `json:"children"`
			} `json:"execs"`
		} `json:"messages"`
		UnlinkedExecs []struct {
			TaskID     string `json:"task_id"`
			Command    string `json:"command"`
			Provider   string `json:"provider"`
			Status     string `json:"status"`
			DurationMs int    `json:"duration_ms"`
		} `json:"unlinked_execs"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing hierarchy: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("EXEC TASK HIERARCHY")
	fmt.Println(strings.Repeat("─", 60))

	// Display message-grouped hierarchy
	for _, msg := range result.Messages {
		title := msg.Title
		if title == "" {
			title = "(no title)"
		}
		age := formatTimestampAge(msg.CreatedAt)
		fmt.Printf("\n◉ Message: %s [%s]\n", truncate(title, 40), age)
		fmt.Printf("  From: %s | ID: %s\n", msg.FromAgent, truncateID(msg.MessageID))

		for _, exec := range msg.Execs {
			status := statusIcon(exec.Status)
			dur := formatDuration(float64(exec.DurationMs))
			fmt.Printf("  ├─▶ exec (%s) %s %s\n", exec.Provider, status, dur)

			for i, child := range exec.Children {
				prefix := "  │  ├─"
				if i == len(exec.Children)-1 {
					prefix = "  │  └─"
				}
				if child.TurnNumber > 0 {
					fmt.Printf("%s turn %d %s %s\n", prefix, child.TurnNumber, statusIcon(child.Status), formatDuration(float64(child.DurationMs)))
				} else if child.ToolName != "" {
					fmt.Printf("%s %s %s %s\n", prefix, child.ToolName, statusIcon(child.Status), formatDuration(float64(child.DurationMs)))
				} else {
					fmt.Printf("%s %s %s %s\n", prefix, child.Command, statusIcon(child.Status), formatDuration(float64(child.DurationMs)))
				}
			}
		}
	}

	// Display unlinked execs
	if len(result.UnlinkedExecs) > 0 {
		fmt.Printf("\n◎ Unlinked Execs (%d):\n", len(result.UnlinkedExecs))
		for _, exec := range result.UnlinkedExecs {
			status := statusIcon(exec.Status)
			dur := formatDuration(float64(exec.DurationMs))
			fmt.Printf("  ├─▶ exec (%s) %s %s\n", exec.Provider, status, dur)
		}
	}

	totalExecs := 0
	for _, msg := range result.Messages {
		totalExecs += len(msg.Execs)
	}
	totalExecs += len(result.UnlinkedExecs)
	fmt.Printf("\nTotal: %d messages, %d exec tasks\n", len(result.Messages), totalExecs)
}

// displayExecHierarchy shows flat exec hierarchy (backward compat).
func displayExecHierarchy(body []byte) {
	var result struct {
		Hierarchy []struct {
			TaskID     string `json:"task_id"`
			Command    string `json:"command"`
			Provider   string `json:"provider"`
			Status     string `json:"status"`
			DurationMs int    `json:"duration_ms"`
			Children   []struct {
				Command    string `json:"command"`
				TurnNumber int    `json:"turn_number,omitempty"`
				ToolName   string `json:"tool_name,omitempty"`
				Status     string `json:"status"`
				DurationMs int    `json:"duration_ms"`
			} `json:"children"`
		} `json:"hierarchy"`
		Count int `json:"count"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing hierarchy: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("EXEC TASK HIERARCHY")
	fmt.Println(strings.Repeat("─", 60))

	for _, exec := range result.Hierarchy {
		status := statusIcon(exec.Status)
		dur := formatDuration(float64(exec.DurationMs))
		fmt.Printf("\n▶ exec (%s) %s %s\n", exec.Provider, status, dur)
		fmt.Printf("  Task ID: %s\n", truncateID(exec.TaskID))

		for i, child := range exec.Children {
			prefix := "  ├─"
			if i == len(exec.Children)-1 {
				prefix = "  └─"
			}
			if child.TurnNumber > 0 {
				fmt.Printf("%s turn %d %s %s\n", prefix, child.TurnNumber, statusIcon(child.Status), formatDuration(float64(child.DurationMs)))
			} else if child.ToolName != "" {
				fmt.Printf("%s %s %s %s\n", prefix, child.ToolName, statusIcon(child.Status), formatDuration(float64(child.DurationMs)))
			} else {
				fmt.Printf("%s %s %s %s\n", prefix, child.Command, statusIcon(child.Status), formatDuration(float64(child.DurationMs)))
			}
		}
	}

	fmt.Printf("\nTotal: %d exec tasks\n", result.Count)
}

// statusIcon returns a status indicator character.
func statusIcon(status string) string {
	switch status {
	case "ok", "completed", "success":
		return "✓"
	case "error", "failed":
		return "✕"
	case "running", "in_progress":
		return "▶"
	default:
		return "○"
	}
}

// dashboardStatsCommand queries aggregation statistics.
func dashboardStatsCommand() {
	fs := flag.NewFlagSet("dashboard stats", flag.ExitOnError)
	server := fs.String("server", "", "Dashboard server URL")
	startDate := fs.String("start", "", "Start date (YYYY-MM-DD)")
	endDate := fs.String("end", "", "End date (YYYY-MM-DD)")
	workspace := fs.String("workspace", "", "Filter by workspace path")
	sourceType := fs.String("source", "", "Filter by source type (github, eval, coordinator, user_session, messaging, cli, direct_api)")
	jsonOutput := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	baseURL := getDashboardURL(*server)
	params := url.Values{}
	if *startDate != "" {
		params.Set("start_date", *startDate)
	}
	if *endDate != "" {
		params.Set("end_date", *endDate)
	}
	if *workspace != "" {
		params.Set("workspace", *workspace)
	}
	if *sourceType != "" {
		params.Set("source_type", *sourceType)
	}

	apiURL := fmt.Sprintf("%s/api/controlplane/stats/breakdown?%s", baseURL, params.Encode())
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to dashboard: %v\n", err)
		fmt.Fprintf(os.Stderr, "Is the server running? Start with: ailang serve\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error from server: %s\n%s\n", resp.Status, string(body))
		os.Exit(1)
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(stats)
		return
	}

	// Format human-readable output
	dateRange := ""
	if *startDate != "" && *endDate != "" {
		dateRange = fmt.Sprintf(" (%s to %s)", *startDate, *endDate)
	}
	fmt.Printf("AGGREGATION STATISTICS%s\n", dateRange)
	fmt.Println(strings.Repeat("─", 50))

	// By Provider
	if providers, ok := stats["by_provider"].([]interface{}); ok && len(providers) > 0 {
		fmt.Println("\nBy Provider:")
		for _, p := range providers {
			prov, _ := p.(map[string]interface{})
			name := getString(prov, "label")
			if name == "" {
				name = getString(prov, "id")
			}
			spans := getInt(prov, "span_count")
			cost := getFloat(prov, "cost_usd")
			pct := getFloat(prov, "percentage")
			fmt.Printf("  %-12s %5d spans   $%.2f  (%.1f%%)\n", name, spans, cost, pct)
		}
	}

	// By Model
	if models, ok := stats["by_model"].([]interface{}); ok && len(models) > 0 {
		fmt.Println("\nBy Model:")
		for _, m := range models {
			mod, _ := m.(map[string]interface{})
			name := truncate(getString(mod, "label"), 28)
			if name == "" {
				name = truncate(getString(mod, "id"), 28)
			}
			spans := getInt(mod, "span_count")
			cost := getFloat(mod, "cost_usd")
			fmt.Printf("  %-28s %5d spans   $%.2f\n", name, spans, cost)
		}
	}

	// By Source Type
	if sources, ok := stats["by_source_type"].([]interface{}); ok && len(sources) > 0 {
		fmt.Println("\nBy Source Type:")
		for _, s := range sources {
			src, _ := s.(map[string]interface{})
			name := getString(src, "label")
			if name == "" {
				name = getString(src, "id")
			}
			spans := getInt(src, "span_count")
			fmt.Printf("  %-15s %5d spans\n", name, spans)
		}
	}

	// Totals
	if totals, ok := stats["totals"].(map[string]interface{}); ok {
		fmt.Println("\nTotals:")
		fmt.Printf("  Spans:      %d\n", getInt(totals, "total_spans"))
		fmt.Printf("  Cost:       $%.2f\n", getFloat(totals, "total_cost_usd"))
		fmt.Printf("  Tokens In:  %d\n", getInt(totals, "total_tokens_in"))
		fmt.Printf("  Tokens Out: %d\n", getInt(totals, "total_tokens_out"))
	}
}

// dashboardHealthCommand checks server health.
func dashboardHealthCommand() {
	fs := flag.NewFlagSet("dashboard health", flag.ExitOnError)
	server := fs.String("server", "", "Dashboard server URL")
	jsonOutput := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	baseURL := getDashboardURL(*server)

	// Check health endpoint
	healthURL := fmt.Sprintf("%s/health", baseURL)
	resp, err := http.Get(healthURL)
	if err != nil {
		if *jsonOutput {
			json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			})
		} else {
			fmt.Printf("Server:      %s (%s)\n", baseURL, red("not running"))
			fmt.Println()
			fmt.Println("Start the server with: ailang serve")
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&health)

	if *jsonOutput {
		health["url"] = baseURL
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(health)
		return
	}

	fmt.Printf("Server:      %s (%s)\n", baseURL, green("running"))

	if dbPath, ok := health["database"].(string); ok {
		fmt.Printf("Database:    %s\n", dbPath)
	}

	if obsEnabled, ok := health["observatory_enabled"].(bool); ok && obsEnabled {
		fmt.Printf("Observatory: %s\n", green("enabled"))
	}

	wsURL := strings.Replace(baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	fmt.Printf("WebSocket:   %s/ws\n", wsURL)

	// Get recent activity counts from spans API
	spansResp, err := http.Get(fmt.Sprintf("%s/api/observatory/spans?limit=1000", baseURL))
	if err == nil {
		defer spansResp.Body.Close()
		var spans []interface{}
		json.NewDecoder(spansResp.Body).Decode(&spans)

		// Count spans from last 24 hours
		recentCount := 0
		cutoff := time.Now().Add(-24 * time.Hour)
		for _, s := range spans {
			span, _ := s.(map[string]interface{})
			if startTime := getString(span, "start_time"); startTime != "" {
				if t, err := time.Parse(time.RFC3339, startTime); err == nil {
					if t.After(cutoff) {
						recentCount++
					}
				}
			}
		}

		fmt.Println()
		fmt.Println("Recent Activity (24h):")
		fmt.Printf("  Spans:     %d\n", recentCount)
	}
}

// dashboardSessionsCommand lists Claude Code sessions.
func dashboardSessionsCommand() {
	fs := flag.NewFlagSet("dashboard sessions", flag.ExitOnError)
	server := fs.String("server", "", "Dashboard server URL")
	limit := fs.Int("limit", 20, "Maximum results")
	jsonOutput := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	baseURL := getDashboardURL(*server)
	apiURL := fmt.Sprintf("%s/api/observatory/sessions?limit=%d", baseURL, *limit)

	resp, err := dashboardHTTPClient.Get(apiURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to dashboard: %v\n", err)
		fmt.Fprintf(os.Stderr, "Is the server running? Start with: ailang serve\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error from server: %s\n%s\n", resp.Status, string(body))
		os.Exit(1)
	}

	var response struct {
		Sessions []map[string]interface{} `json:"sessions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}
	sessions := response.Sessions

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(response) // Output full response object
		return
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found")
		fmt.Println()
		fmt.Println("Sessions are captured via Claude Code hooks.")
		fmt.Println("Make sure the telemetry hook is configured in ~/.claude/settings.json")
		return
	}

	// Table output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SESSION ID\tWORKSPACE\tSTARTED\tTURNS")
	fmt.Fprintln(w, strings.Repeat("-", 12)+"\t"+strings.Repeat("-", 30)+"\t"+strings.Repeat("-", 10)+"\t"+strings.Repeat("-", 5))

	for _, session := range sessions {
		id := truncateID(getString(session, "session_id"))
		workspace := getString(session, "workspace")
		// Show just the project name from the path
		if idx := strings.LastIndex(workspace, "/"); idx > 0 {
			workspace = ".../" + workspace[idx+1:]
		}
		workspace = truncate(workspace, 30)
		started := formatTimestampAge(getString(session, "started_at"))
		turns := getInt(session, "turn_count")

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", id, workspace, started, turns)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d sessions\n", len(sessions))
	fmt.Println()
	fmt.Println("View tools for a session: ailang dashboard tools <session-id>")
}

// dashboardToolsCommand shows tool usage for a session.
func dashboardToolsCommand() {
	fs := flag.NewFlagSet("dashboard tools", flag.ExitOnError)
	server := fs.String("server", "", "Dashboard server URL")
	summary := fs.Bool("summary", false, "Show aggregated summary by tool type")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	workspace := fs.String("workspace", "", "Filter by workspace path")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Session ID is the first positional arg after flags
	args := fs.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: ailang dashboard tools [--summary] [--json] [--workspace PATH] <session-id>\n")
		fmt.Fprintf(os.Stderr, "\nGet session IDs from: ailang dashboard sessions\n")
		os.Exit(1)
	}
	sessionID := args[0]

	baseURL := getDashboardURL(*server)
	var apiURL string
	if *summary {
		apiURL = fmt.Sprintf("%s/api/observatory/sessions/%s/tools/summary", baseURL, sessionID)
	} else {
		apiURL = fmt.Sprintf("%s/api/observatory/sessions/%s/tools", baseURL, sessionID)
	}
	if *workspace != "" {
		apiURL += "?workspace=" + url.QueryEscape(*workspace)
	}

	resp, err := dashboardHTTPClient.Get(apiURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to dashboard: %v\n", err)
		fmt.Fprintf(os.Stderr, "Is the server running? Start with: ailang serve\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error from server: %s\n%s\n", resp.Status, string(body))
		os.Exit(1)
	}

	body, _ := io.ReadAll(resp.Body)

	if *jsonOutput {
		var parsed interface{}
		json.Unmarshal(body, &parsed)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(parsed)
		return
	}

	if *summary {
		displayToolsSummary(body, sessionID)
	} else {
		displayToolsList(body, sessionID)
	}
}

func displayToolsSummary(body []byte, sessionID string) {
	// API returns {tools: [{tool_name, count, success_count, details}, ...]}
	var response struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	if len(response.Tools) == 0 {
		fmt.Printf("No tools found for session %s\n", truncateID(sessionID))
		return
	}

	fmt.Printf("TOOL USAGE SUMMARY for session %s\n", truncateID(sessionID))
	fmt.Println(strings.Repeat("─", 60))

	// Print tools from array
	for _, data := range response.Tools {
		toolName := getString(data, "tool_name")
		count := getInt(data, "count")
		success := getInt(data, "success_count")
		failed := count - success

		statusStr := ""
		if success > 0 || failed > 0 {
			statusStr = fmt.Sprintf(" (%d ok, %d fail)", success, failed)
		}

		fmt.Printf("\n%s: %d calls%s\n", toolName, count, statusStr)

		// Show details (combined files, patterns, commands)
		if details, ok := data["details"].([]interface{}); ok && len(details) > 0 {
			fmt.Println("  Details:")
			for i, d := range details {
				if i >= 5 {
					fmt.Printf("    ... and %d more\n", len(details)-5)
					break
				}
				detail := fmt.Sprintf("%v", d)
				// Shorten long paths
				if len(detail) > 60 {
					detail = "..." + detail[len(detail)-57:]
				}
				fmt.Printf("    %s\n", detail)
			}
		}
	}
}

func displayToolsList(body []byte, sessionID string) {
	var tools []map[string]interface{}
	if err := json.Unmarshal(body, &tools); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	if len(tools) == 0 {
		fmt.Printf("No tools found for session %s\n", truncateID(sessionID))
		return
	}

	fmt.Printf("TOOLS for session %s (%d total)\n", truncateID(sessionID), len(tools))
	fmt.Println(strings.Repeat("─", 70))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TOOL\tDURATION\tDETAILS")
	fmt.Fprintln(w, strings.Repeat("-", 15)+"\t"+strings.Repeat("-", 10)+"\t"+strings.Repeat("-", 40))

	for _, tool := range tools {
		name := getString(tool, "tool_name")
		dur := formatDuration(getFloat(tool, "duration_ms"))

		// Extract details from metadata
		details := ""
		if metadata, ok := tool["metadata"].(map[string]interface{}); ok {
			if filePath, ok := metadata["file_path"].(string); ok {
				// Shorten path
				if len(filePath) > 40 {
					filePath = "..." + filePath[len(filePath)-37:]
				}
				details = filePath
			} else if pattern, ok := metadata["pattern"].(string); ok {
				details = "grep: " + truncate(pattern, 35)
			} else if cmd, ok := metadata["command"].(string); ok {
				details = truncate(cmd, 40)
			} else if desc, ok := metadata["description"].(string); ok {
				// Task: prefer description over prompt
				details = truncate(desc, 50)
			} else if skill, ok := metadata["skill"].(string); ok {
				// Skill: show skill name
				details = "skill: " + skill
			} else if prompt, ok := metadata["prompt"].(string); ok {
				details = truncate(prompt, 40)
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n", name, dur, details)
	}
	w.Flush()
}

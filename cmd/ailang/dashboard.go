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

// dashboardSpansCommand queries observatory spans with filters.
func dashboardSpansCommand() {
	fs := flag.NewFlagSet("dashboard spans", flag.ExitOnError)
	server := fs.String("server", "", "Dashboard server URL")
	provider := fs.String("provider", "", "Filter by provider (claude, gemini, openai)")
	model := fs.String("model", "", "Filter by model name")
	status := fs.String("status", "", "Filter by status (ok, error)")
	workspace := fs.String("workspace", "", "Filter by workspace path")
	taskID := fs.String("task-id", "", "Filter by task ID")
	traceID := fs.String("trace-id", "", "Filter by trace ID")
	limit := fs.Int("limit", 50, "Maximum results")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	enriched := fs.Bool("enriched", false, "Enrich spans with tool metadata (file paths, commands, etc.)")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	baseURL := getDashboardURL(*server)
	params := url.Values{}
	if *provider != "" {
		params.Set("provider", *provider)
	}
	if *model != "" {
		params.Set("model", *model)
	}
	if *status != "" {
		params.Set("status", *status)
	}
	if *workspace != "" {
		params.Set("workspace", *workspace)
	}
	if *taskID != "" {
		params.Set("task_id", *taskID)
	}
	if *traceID != "" {
		params.Set("trace_id", *traceID)
	}
	params.Set("limit", fmt.Sprintf("%d", *limit))

	// Use enriched endpoint if flag is set
	endpoint := "spans"
	if *enriched {
		endpoint = "spans/enriched"
	}
	apiURL := fmt.Sprintf("%s/api/observatory/%s?%s", baseURL, endpoint, params.Encode())
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

	var spans []map[string]interface{}
	if *enriched {
		// Enriched endpoint returns {spans: [...], enriched: true}
		var response struct {
			Spans    []map[string]interface{} `json:"spans"`
			Enriched bool                     `json:"enriched"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
			os.Exit(1)
		}
		spans = response.Spans
	} else {
		// Regular endpoint returns array directly
		if err := json.NewDecoder(resp.Body).Decode(&spans); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
			os.Exit(1)
		}
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(spans)
		return
	}

	if len(spans) == 0 {
		fmt.Println("No spans found matching criteria")
		return
	}

	// Table output - use display_name for enriched output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if *enriched {
		fmt.Fprintln(w, "ID\tDISPLAY NAME\tPROVIDER\tMODEL\tSTATUS\tDURATION")
	} else {
		fmt.Fprintln(w, "ID\tNAME\tPROVIDER\tMODEL\tSTATUS\tDURATION")
	}
	fmt.Fprintln(w, strings.Repeat("-", 12)+"\t"+strings.Repeat("-", 40)+"\t"+strings.Repeat("-", 10)+"\t"+strings.Repeat("-", 20)+"\t"+strings.Repeat("-", 6)+"\t"+strings.Repeat("-", 10))

	for _, span := range spans {
		id := truncateID(getString(span, "id"))
		// Use display_name if enriched and available, otherwise use name
		name := getString(span, "display_name")
		if name == "" {
			name = getString(span, "name")
		}
		name = truncate(name, 40)
		prov := getString(span, "provider")
		if prov == "" {
			prov = "-"
		}
		mod := truncate(getString(span, "model"), 20)
		if mod == "" {
			mod = "-"
		}
		stat := getString(span, "status")
		dur := formatDuration(getFloat(span, "duration_ms"))

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", id, name, prov, mod, stat, dur)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d spans\n", len(spans))
}

// dashboardInboxCommand queries the unified inbox.
func dashboardInboxCommand() {
	fs := flag.NewFlagSet("dashboard inbox", flag.ExitOnError)
	server := fs.String("server", "", "Dashboard server URL")
	provider := fs.String("provider", "", "Filter by provider (claude, gemini)")
	model := fs.String("model", "", "Filter by model name")
	inbox := fs.String("inbox", "", "Filter by inbox name")
	sourceType := fs.String("source", "", "Filter by source type (github, eval, coordinator, user_session, messaging, cli, direct_api)")
	startDate := fs.String("start", "", "Start date (YYYY-MM-DD)")
	endDate := fs.String("end", "", "End date (YYYY-MM-DD)")
	workspace := fs.String("workspace", "", "Filter by workspace path")
	status := fs.String("status", "", "Filter by status (unread, read)")
	sortBy := fs.String("sort", "timestamp", "Sort by: timestamp, turns, cost, tokens, duration")
	sortOrder := fs.String("order", "desc", "Sort order: asc, desc")
	limit := fs.Int("limit", 50, "Maximum results")
	jsonOutput := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	baseURL := getDashboardURL(*server)
	params := url.Values{}
	if *provider != "" {
		params.Set("provider", *provider)
	}
	if *model != "" {
		params.Set("model", *model)
	}
	if *inbox != "" {
		params.Set("inbox", *inbox)
	}
	if *sourceType != "" {
		params.Set("source_type", *sourceType)
	}
	if *startDate != "" {
		params.Set("start_date", *startDate)
	}
	if *endDate != "" {
		params.Set("end_date", *endDate)
	}
	if *workspace != "" {
		params.Set("workspace", *workspace)
	}
	if *status != "" {
		params.Set("status", *status)
	}
	params.Set("sort", *sortBy)
	params.Set("order", *sortOrder)
	params.Set("limit", fmt.Sprintf("%d", *limit))

	apiURL := fmt.Sprintf("%s/api/inbox?%s", baseURL, params.Encode())
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

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	messages, _ := result["messages"].([]interface{})

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(messages)
		return
	}

	if len(messages) == 0 {
		fmt.Println("No events found matching criteria")
		return
	}

	// Table output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tFROM\tCOST\tTOKENS\tTURNS\tAGE")
	fmt.Fprintln(w, strings.Repeat("-", 12)+"\t"+strings.Repeat("-", 12)+"\t"+strings.Repeat("-", 12)+"\t"+strings.Repeat("-", 8)+"\t"+strings.Repeat("-", 8)+"\t"+strings.Repeat("-", 5)+"\t"+strings.Repeat("-", 8))

	msgCount := 0
	ccCount := 0
	for _, m := range messages {
		msg, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		id := truncateID(getString(msg, "id"))
		msgType := getString(msg, "source")
		if msgType == "" {
			msgType = getString(msg, "message_type")
		}
		from := truncate(getString(msg, "from_agent"), 12)
		age := formatTimestampAge(getString(msg, "created_at"))

		// Cost and token info (only for claude_code events)
		cost := "-"
		tokens := "-"
		turns := "-"
		if costVal := getFloat(msg, "cost_usd"); costVal > 0 {
			cost = fmt.Sprintf("$%.2f", costVal)
		}
		tokensIn := getInt(msg, "tokens_in")
		tokensOut := getInt(msg, "tokens_out")
		if tokensIn > 0 || tokensOut > 0 {
			totalTokens := tokensIn + tokensOut
			if totalTokens >= 1000 {
				tokens = fmt.Sprintf("%.1fk", float64(totalTokens)/1000)
			} else {
				tokens = fmt.Sprintf("%d", totalTokens)
			}
		}
		if turnCount := getInt(msg, "turn_count"); turnCount > 0 {
			turns = fmt.Sprintf("%d", turnCount)
		}

		if msgType == "claude_code" || msgType == "claude_code_session" {
			ccCount++
			if from == "" {
				from = "claude-code"
			}
		} else {
			msgCount++
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", id, msgType, from, cost, tokens, turns, age)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d events (%d messages, %d claude_code)\n", len(messages), msgCount, ccCount)
}

// dashboardTracesCommand queries trace summaries.
func dashboardTracesCommand() {
	fs := flag.NewFlagSet("dashboard traces", flag.ExitOnError)
	server := fs.String("server", "", "Dashboard server URL")
	taskID := fs.String("task-id", "", "Filter by task ID")
	traceID := fs.String("trace-id", "", "Filter by trace ID")
	limit := fs.Int("limit", 20, "Maximum results")
	jsonOutput := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	baseURL := getDashboardURL(*server)
	params := url.Values{}
	if *taskID != "" {
		params.Set("task_id", *taskID)
	}
	if *traceID != "" {
		params.Set("trace_id", *traceID)
	}
	params.Set("limit", fmt.Sprintf("%d", *limit))

	apiURL := fmt.Sprintf("%s/api/observatory/traces?%s", baseURL, params.Encode())
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

	var traces []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(traces)
		return
	}

	if len(traces) == 0 {
		fmt.Println("No traces found")
		return
	}

	// Table output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TRACE ID\tROOT SPAN\tSERVICE\tSPANS\tDURATION\tSTATUS")
	fmt.Fprintln(w, strings.Repeat("-", 16)+"\t"+strings.Repeat("-", 20)+"\t"+strings.Repeat("-", 15)+"\t"+strings.Repeat("-", 5)+"\t"+strings.Repeat("-", 10)+"\t"+strings.Repeat("-", 6))

	for _, trace := range traces {
		trID := truncateID(getString(trace, "trace_id"))
		rootSpan := truncate(getString(trace, "root_span"), 20)
		service := truncate(getString(trace, "service_name"), 15)
		if service == "" {
			service = "-"
		}
		spanCount := getInt(trace, "span_count")
		dur := formatDuration(getFloat(trace, "duration_ms"))
		stat := getString(trace, "status")

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n", trID, rootSpan, service, spanCount, dur, stat)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d traces\n", len(traces))
}

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

// Helper functions

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

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Session ID is the first positional arg after flags
	args := fs.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: ailang dashboard tools <session-id> [--summary] [--json]\n")
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

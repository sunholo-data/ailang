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
)

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
	includeChat := fs.Bool("include-chat", false, "Include chat context (user prompt, assistant response) for each span")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// --include-chat requires the enriched endpoint
	if *includeChat {
		*enriched = true
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
	if *includeChat {
		params.Set("include_chat", "true")
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

		// Show chat context below the span row when --include-chat is set
		if *includeChat {
			if chatCtx, ok := span["chat_context"].(map[string]interface{}); ok && chatCtx != nil {
				if prompt, _ := chatCtx["user_prompt"].(string); prompt != "" {
					fmt.Fprintf(w, "\t  💬 User: %s\n", truncate(prompt, 80))
				}
				if response, _ := chatCtx["assistant_response"].(string); response != "" {
					fmt.Fprintf(w, "\t  🤖 Asst: %s\n", truncate(response, 80))
				}
				turn, _ := chatCtx["turn_number"].(float64)
				thinking, _ := chatCtx["has_thinking"].(bool)
				if turn > 0 || thinking {
					extra := fmt.Sprintf("\t  Turn: %.0f", turn)
					if thinking {
						extra += " (with thinking)"
					}
					fmt.Fprintln(w, extra)
				}
			}
		}
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

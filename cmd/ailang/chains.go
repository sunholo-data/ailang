package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sunholo/ailang/internal/observatory"
)

func chainsCommand() {
	if flag.NArg() < 2 {
		// No subcommand - show interactive mode if terminal, else show help
		if isTerminal() {
			runChainsInteractive()
			return
		}
		fmt.Println("Usage: ailang chains <subcommand> [options]")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  list    List execution chains")
		fmt.Println("  view    View a chain with all stages")
		fmt.Println("  tree    ASCII tree view of chain hierarchy")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang chains list                  # List all chains")
		fmt.Println("  ailang chains list --status active  # Filter by status")
		fmt.Println("  ailang chains view <chain-id>       # View chain details")
		fmt.Println("  ailang chains tree <chain-id>       # View as tree")
		fmt.Println()
		fmt.Println("Run 'ailang chains' in a terminal for interactive mode.")
		os.Exit(1)
	}

	subcommand := flag.Arg(1)
	switch subcommand {
	case "list":
		chainsListCommand()
	case "view":
		chainsViewCommand()
	case "tree":
		chainsTreeCommand()
	default:
		fmt.Printf("Unknown subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func chainsListCommand() {
	fs := flag.NewFlagSet("chains list", flag.ExitOnError)
	status := fs.String("status", "", "Filter by status (active, pending_approval, completed, failed)")
	sourceType := fs.String("source", "", "Filter by source type (github_issue, message, manual)")
	limit := fs.Int("limit", 20, "Maximum number of chains to show")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	fullIDs := fs.Bool("full", false, "Show full chain IDs (for copy-paste)")
	fs.Parse(flag.Args()[2:])

	// Connect to observatory database
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()
	opts := observatory.ChainListOptions{
		Limit: *limit,
	}
	if *status != "" {
		opts.Status = observatory.ChainStatus(*status)
	}
	if *sourceType != "" {
		opts.SourceType = *sourceType
	}

	chains, err := backend.ListChains(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to list chains: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(chains)
		return
	}

	if len(chains) == 0 {
		fmt.Println("No execution chains found.")
		return
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tSOURCE\tSTAGES\tCOST\tCREATED")
	for _, chain := range chains {
		source := chain.SourceType
		if chain.SourceRef != "" {
			source = fmt.Sprintf("%s:%s", chain.SourceType, chain.SourceRef)
		}
		cost := fmt.Sprintf("$%.2f", chain.TotalCost)
		created := chain.CreatedAt.Format("2006-01-02 15:04")

		// Show full or truncated ID based on flag
		chainID := truncateChainID(chain.ID)
		if *fullIDs {
			chainID = chain.ID
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
			chainID,
			chain.Status,
			source,
			chain.StagesCompleted,
			cost,
			created,
		)
	}
	w.Flush()
}

func chainsViewCommand() {
	fs := flag.NewFlagSet("chains view", flag.ExitOnError)
	includeSpans := fs.Bool("spans", false, "Include spans for each stage")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	fs.Parse(flag.Args()[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: ailang chains view <chain-id>")
		os.Exit(1)
	}

	chainIDPrefix := fs.Arg(0)

	// Connect to observatory database
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()

	// Resolve short ID prefix to full ID
	chainID, err := resolveChainID(backend, ctx, chainIDPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	opts := observatory.ChainReadOptions{
		IncludeStages: true,
		IncludeSpans:  *includeSpans,
	}

	chain, err := backend.GetChain(ctx, chainID, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get chain: %v\n", err)
		os.Exit(1)
	}
	if chain == nil {
		fmt.Fprintf(os.Stderr, "Error: chain not found: %s\n", chainID)
		os.Exit(1)
	}

	// Get stages
	stages, err := backend.GetChainStages(ctx, chainID, opts)
	if err == nil {
		chain.Stages = stages
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(chain)
		return
	}

	// Print chain details
	fmt.Printf("Chain: %s\n", chain.ID)
	fmt.Printf("Status: %s\n", colorizeStatus(string(chain.Status)))
	fmt.Printf("Source: %s", chain.SourceType)
	if chain.SourceRef != "" {
		fmt.Printf(" (%s)", chain.SourceRef)
	}
	fmt.Println()
	if chain.GitHubRepo != "" {
		fmt.Printf("GitHub: %s#%d\n", chain.GitHubRepo, chain.GitHubIssueNumber)
	}
	fmt.Printf("Created: %s\n", chain.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Total Cost: $%.4f\n", chain.TotalCost)
	fmt.Printf("Total Tokens: %d\n", chain.TotalTokens)
	fmt.Println()

	// Print stages
	if len(chain.Stages) > 0 {
		fmt.Println("Stages:")
		for i, stage := range chain.Stages {
			fmt.Printf("  %d. %s [%s]\n", i+1, stage.AgentID, colorizeStatus(string(stage.Status)))
			if stage.TaskID != "" {
				fmt.Printf("     Task: %s\n", stage.TaskID)
			}
			if stage.SessionID != "" {
				fmt.Printf("     Session: %s\n", truncateChainID(stage.SessionID))
			}
			if stage.ApprovalStatus != "" {
				fmt.Printf("     Approval: %s\n", stage.ApprovalStatus)
			}
			if stage.Cost > 0 {
				fmt.Printf("     Cost: $%.4f (%d tokens in, %d tokens out)\n",
					stage.Cost, stage.TokensIn, stage.TokensOut)
			}
			if stage.HandoffTo != "" {
				fmt.Printf("     Handoff: -> %s\n", stage.HandoffTo)
			}
		}
	}
}

func chainsTreeCommand() {
	fs := flag.NewFlagSet("chains tree", flag.ExitOnError)
	detailed := fs.Bool("detailed", false, "Show execution details (turns, tools)")
	jsonOutput := fs.Bool("json", false, "Output raw JSON with full event data")
	summary := fs.Bool("summary", false, "JSON summary: turn numbers and tools only (no content)")
	toolsOnly := fs.Bool("tools", false, "JSON: show only tool calls (skip text/thinking)")
	turnNum := fs.Int("turn", 0, "JSON: show only specific turn number")
	stageNum := fs.Int("stage", 0, "Filter to specific stage (1-based)")
	sessionID := fs.String("session", "", "Session ID to get full chat data (bypasses chain lookup)")
	fs.Parse(flag.Args()[2:])

	// Direct session query mode
	if *sessionID != "" {
		messages := getChatMessages(*sessionID)
		if len(messages) == 0 {
			fmt.Fprintf(os.Stderr, "No chat messages found for session: %s\n", *sessionID)
			fmt.Fprintf(os.Stderr, "Try: ailang observatory sync-chat --status to see imported sessions\n")
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]interface{}{
			"session_id": *sessionID,
			"messages":   messages,
		})
		return
	}

	if fs.NArg() < 1 {
		fmt.Println("Usage: ailang chains tree [--detailed] [--json] <chain-id>")
		fmt.Println("       ailang chains tree --session <session-id>  # Get full chat for session")
		os.Exit(1)
	}

	chainIDPrefix := fs.Arg(0)

	// Connect to observatory database
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()

	// Resolve short ID prefix to full ID
	chainID, err := resolveChainID(backend, ctx, chainIDPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	opts := observatory.ChainReadOptions{
		IncludeStages: true,
	}

	chain, err := backend.GetChain(ctx, chainID, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get chain: %v\n", err)
		os.Exit(1)
	}
	if chain == nil {
		fmt.Fprintf(os.Stderr, "Error: chain not found: %s\n", chainID)
		os.Exit(1)
	}

	stages, err := backend.GetChainStages(ctx, chainID, opts)
	if err == nil {
		chain.Stages = stages
	}

	// JSON output with filtering options
	if *jsonOutput || *summary {
		opts := jsonExportOptions{
			SummaryOnly: *summary,
			ToolsOnly:   *toolsOnly,
			TurnFilter:  *turnNum,
			StageFilter: *stageNum,
		}
		printChainJSONFiltered(chain, opts)
		return
	}

	// Print tree
	printChainTreeDetailed(ctx, backend, chain, *detailed)
}

func printChainTree(chain *observatory.ExecutionChain) {
	printChainTreeDetailed(context.Background(), nil, chain, false)
}

func printChainTreeDetailed(ctx context.Context, backend *observatory.SQLiteBackend, chain *observatory.ExecutionChain, detailed bool) {
	// Source node
	sourceLabel := string(chain.SourceType)
	if chain.GitHubRepo != "" {
		sourceLabel = fmt.Sprintf("%s (%s#%d)", chain.SourceType, chain.GitHubRepo, chain.GitHubIssueNumber)
	} else if chain.SourceRef != "" {
		sourceLabel = fmt.Sprintf("%s (%s)", chain.SourceType, chain.SourceRef)
	}
	fmt.Printf("%s %s\n", colorizeStatus(string(chain.Status)), sourceLabel)

	// Stages
	for i, stage := range chain.Stages {
		isLast := i == len(chain.Stages)-1
		prefix := "├──"
		childPrefix := "│   "
		if isLast {
			prefix = "└──"
			childPrefix = "    "
		}

		statusIcon := getStatusIcon(string(stage.Status))
		fmt.Printf("%s %s %s", prefix, statusIcon, stage.AgentID)

		// Add stage details
		details := []string{}
		if stage.Turns > 0 {
			details = append(details, fmt.Sprintf("%d turns", stage.Turns))
		}
		if stage.Cost > 0 {
			details = append(details, fmt.Sprintf("$%.2f", stage.Cost))
		}
		if stage.ApprovalStatus == "pending" {
			details = append(details, yellow("[awaiting approval]"))
		}
		if len(details) > 0 {
			fmt.Printf(" (%s)", strings.Join(details, ", "))
		}
		fmt.Println()

		// Show execution details if requested and backend available
		if detailed && backend != nil && stage.TaskID != "" {
			printStageDetails(ctx, backend, stage.TaskID, childPrefix)
		}

		// Show handoff arrow if applicable
		if stage.HandoffTo != "" && !isLast {
			fmt.Printf("%s└── -> %s\n", childPrefix, stage.HandoffTo)
		}
	}
}

func printStageDetails(ctx context.Context, backend *observatory.SQLiteBackend, taskID, prefix string) {
	// First try to get session info (including time range) from coordinator.db tasks
	sessionInfo := getSessionInfoFromTask(taskID)

	// If we have session_id, show full chat data with tools (filtered by task time range)
	if sessionInfo != nil && sessionInfo.SessionID != "" {
		messages := getChatMessagesInRange(sessionInfo.SessionID, sessionInfo.StartedAt, sessionInfo.CompletedAt)
		if len(messages) > 0 {
			printChatTurnDetails(messages, prefix)
			return
		}
	}

	// Fallback: Try coordinator events (streaming chunks, less detail)
	events := getTaskEvents(taskID)
	if len(events) > 0 {
		printDetailedTurnEvents(events, prefix)
		return
	}

	// Final fallback: Get spans for this task (basic info only)
	spans, err := backend.ListSpans(ctx, observatory.SpanListOptions{
		TaskID: taskID,
		Limit:  100,
	})
	if err != nil || len(spans) == 0 {
		return
	}

	// Collect turn info and duration from spans
	turns := []string{}
	var totalDuration int64
	for _, span := range spans {
		if span.Name == "exec.turn" {
			turnNum := 0
			if v, ok := span.Attributes["turn.number"]; ok {
				switch n := v.(type) {
				case float64:
					turnNum = int(n)
				case int:
					turnNum = n
				}
			}
			if turnNum > 0 {
				turns = append(turns, fmt.Sprintf("T%d", turnNum))
			}
		}
		if span.Name == "claude.execute" || span.Name == "gemini.execute" {
			totalDuration = span.DurationMs
		}
	}

	// Print summary line (fallback)
	if len(turns) > 0 || totalDuration > 0 {
		details := []string{}
		if len(turns) > 0 {
			details = append(details, fmt.Sprintf("%d turns", len(turns)))
		}
		if totalDuration > 0 {
			details = append(details, formatChainDuration(totalDuration))
		}
		fmt.Printf("%s└── %s\n", prefix, strings.Join(details, " | "))
	}
}

// printChatTurnDetails prints turn-by-turn details from chat messages with full tool info
func printChatTurnDetails(messages []chatMessageExport, prefix string) {
	// Group messages by turn
	type turnData struct {
		tools      []string
		toolInputs map[string]string // tool name -> preview of input
		textParts  []string
	}
	turns := make(map[int]*turnData)
	maxTurn := 0

	for _, msg := range messages {
		tn := msg.TurnNumber
		if tn > maxTurn {
			maxTurn = tn
		}
		if tn == 0 {
			continue
		}

		if turns[tn] == nil {
			turns[tn] = &turnData{toolInputs: make(map[string]string)}
		}
		td := turns[tn]

		for _, block := range msg.Content {
			switch block.Type {
			case "tool_use":
				if block.ToolUse != nil {
					td.tools = append(td.tools, block.ToolUse.Name)
					// Store input preview for important tools
					if block.ToolUse.Input != nil {
						inputStr := formatToolInputPreview(block.ToolUse.Name, block.ToolUse.Input)
						if inputStr != "" {
							td.toolInputs[block.ToolUse.Name] = inputStr
						}
					}
				}
			case "text":
				if block.Text != "" && len(block.Text) > 20 {
					td.textParts = append(td.textParts, block.Text)
				}
			}
		}
	}

	// Print each turn
	turnNums := make([]int, 0, len(turns))
	for tn := range turns {
		turnNums = append(turnNums, tn)
	}
	sort.Ints(turnNums)

	for i, tn := range turnNums {
		td := turns[tn]
		isLast := i == len(turnNums)-1
		turnPrefix := "├──"
		childPfx := "│   "
		if isLast {
			turnPrefix = "└──"
			childPfx = "    "
		}

		// Build turn line with tools
		turnLine := fmt.Sprintf("Turn %d", tn)
		if len(td.tools) > 0 {
			uniqueTools := dedupeTools(td.tools)
			turnLine += fmt.Sprintf(": %s", cyan(strings.Join(uniqueTools, ", ")))
		}
		fmt.Printf("%s%s %s\n", prefix, turnPrefix, turnLine)

		// Show tool input previews (most useful for debugging)
		for toolName, input := range td.toolInputs {
			fmt.Printf("%s%s   %s: %s\n", prefix, childPfx, dim(toolName), input)
		}

		// Show text preview if no tool inputs shown
		if len(td.toolInputs) == 0 && len(td.textParts) > 0 {
			preview := getTextPreview(td.textParts[0], 100)
			if preview != "" {
				fmt.Printf("%s%s   %s\n", prefix, childPfx, dim(preview))
			}
		}
	}
}

// formatToolInputPreview formats a tool input for display
func formatToolInputPreview(toolName string, input interface{}) string {
	switch toolName {
	case "Read":
		if m, ok := input.(map[string]interface{}); ok {
			if path, ok := m["file_path"].(string); ok {
				return truncateString(path, 60)
			}
		}
	case "Write":
		if m, ok := input.(map[string]interface{}); ok {
			if path, ok := m["file_path"].(string); ok {
				return truncateString(path, 60)
			}
		}
	case "Edit":
		if m, ok := input.(map[string]interface{}); ok {
			if path, ok := m["file_path"].(string); ok {
				return truncateString(path, 60)
			}
		}
	case "Bash":
		if m, ok := input.(map[string]interface{}); ok {
			if cmd, ok := m["command"].(string); ok {
				return truncateString(cmd, 80)
			}
		}
	case "Grep", "Glob":
		if m, ok := input.(map[string]interface{}); ok {
			if pattern, ok := m["pattern"].(string); ok {
				return truncateString(pattern, 60)
			}
		}
	case "Task":
		if m, ok := input.(map[string]interface{}); ok {
			if desc, ok := m["description"].(string); ok {
				return truncateString(desc, 60)
			}
		}
	}
	return ""
}

// taskEvent represents an event from coordinator task_events table (basic)
type taskEvent struct {
	StreamType string
	TurnNum    int
	Text       string
	ToolName   string
	ToolInput  string
}

// getTaskEvents queries coordinator.db for basic task events (for tree display)
func getTaskEvents(taskID string) []taskEvent {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dbPath := filepath.Join(homeDir, ".ailang", "state", "coordinator.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT stream_type, turn_num, COALESCE(text, ''), COALESCE(tool_name, ''), COALESCE(tool_input, '')
		FROM task_events
		WHERE task_id = ?
		ORDER BY id ASC
	`, taskID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var events []taskEvent
	for rows.Next() {
		var e taskEvent
		if err := rows.Scan(&e.StreamType, &e.TurnNum, &e.Text, &e.ToolName, &e.ToolInput); err != nil {
			continue
		}
		events = append(events, e)
	}
	return events
}

// chainExport is the full JSON export structure
type chainExport struct {
	Chain  *observatory.ExecutionChain `json:"chain"`
	Stages []stageExport               `json:"stages"`
}

type stageExport struct {
	Stage       observatory.ChainStage `json:"stage"`
	Messages    []chatMessageExport    `json:"messages,omitempty"`     // Full chat with tool data
	DataStatus  string                 `json:"data_status,omitempty"`  // "available", "no_session_id", "not_synced"
	DataMessage string                 `json:"data_message,omitempty"` // Human-readable explanation
}

// chatMessageExport represents a chat message with full tool data
type chatMessageExport struct {
	ID         string         `json:"id"`
	SessionID  string         `json:"session_id"`
	TurnNumber int            `json:"turn_number"`
	Role       string         `json:"role"`
	Model      string         `json:"model,omitempty"`
	TokensIn   int            `json:"tokens_in,omitempty"`
	TokensOut  int            `json:"tokens_out,omitempty"`
	Timestamp  string         `json:"timestamp"`
	Content    []contentBlock `json:"content"` // Parsed content blocks with tools
}

// contentBlock represents a content block from Claude Code
type contentBlock struct {
	Type string `json:"type"` // "text", "thinking", "tool_use", "tool_result"

	// For type="text"
	Text string `json:"text,omitempty"`

	// For type="thinking"
	Thinking string `json:"thinking,omitempty"`

	// For type="tool_use"
	ToolUse *toolUseBlock `json:"tool_use,omitempty"`

	// For type="tool_result"
	ToolResult *toolResultBlock `json:"tool_result,omitempty"`
}

type toolUseBlock struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Input interface{} `json:"input"` // Full tool input (varies by tool)
}

type toolResultBlock struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"` // Full tool output
	IsError   bool   `json:"is_error"`
}

// jsonExportOptions controls JSON output filtering
type jsonExportOptions struct {
	SummaryOnly bool // Only show turn numbers and tool names
	ToolsOnly   bool // Only show tool_use blocks
	TurnFilter  int  // Show only specific turn (0 = all)
	StageFilter int  // Show only specific stage (0 = all, 1-based)
}

// printChainJSONFiltered outputs filtered chain data as JSON
func printChainJSONFiltered(chain *observatory.ExecutionChain, opts jsonExportOptions) {
	if opts.SummaryOnly {
		printChainJSONSummary(chain, opts)
		return
	}
	printChainJSONFull(chain, opts)
}

// printChainJSONSummary outputs a compact summary (turns and tools only)
func printChainJSONSummary(chain *observatory.ExecutionChain, opts jsonExportOptions) {
	type turnSummary struct {
		TurnNumber int      `json:"turn"`
		Tools      []string `json:"tools,omitempty"`
		ToolCount  int      `json:"tool_count"`
		TextLength int      `json:"text_length,omitempty"`
	}
	type stageSummary struct {
		StageNum   int           `json:"stage"`
		AgentID    string        `json:"agent"`
		Status     string        `json:"status"`
		TaskID     string        `json:"task_id,omitempty"`
		TurnCount  int           `json:"turns"`
		ToolCalls  int           `json:"tool_calls"`
		TurnDetail []turnSummary `json:"turn_detail,omitempty"`
	}

	summary := struct {
		ChainID    string         `json:"chain_id"`
		Status     string         `json:"status"`
		Source     string         `json:"source"`
		TotalCost  float64        `json:"total_cost"`
		StageCount int            `json:"stage_count"`
		Stages     []stageSummary `json:"stages"`
	}{
		ChainID:    chain.ID,
		Status:     string(chain.Status),
		Source:     string(chain.SourceType),
		TotalCost:  chain.TotalCost,
		StageCount: len(chain.Stages),
		Stages:     make([]stageSummary, 0),
	}

	for i, stage := range chain.Stages {
		stageIdx := i + 1
		if opts.StageFilter > 0 && stageIdx != opts.StageFilter {
			continue
		}

		ss := stageSummary{
			StageNum: stageIdx,
			AgentID:  stage.AgentID,
			Status:   string(stage.Status),
			TaskID:   stage.TaskID,
		}

		// Get session info and messages
		var sessionInfo *taskSessionInfo
		if stage.TaskID != "" {
			sessionInfo = getSessionInfoFromTask(stage.TaskID)
		}

		var messages []chatMessageExport
		if sessionInfo != nil && sessionInfo.SessionID != "" {
			messages = getChatMessagesInRange(sessionInfo.SessionID, sessionInfo.StartedAt, sessionInfo.CompletedAt)
		}

		// Build turn summaries from chat messages
		turnMap := make(map[int]*turnSummary)
		if len(messages) > 0 {
			for _, msg := range messages {
				tn := msg.TurnNumber
				if opts.TurnFilter > 0 && tn != opts.TurnFilter {
					continue
				}
				if turnMap[tn] == nil {
					turnMap[tn] = &turnSummary{TurnNumber: tn}
				}
				ts := turnMap[tn]

				for _, block := range msg.Content {
					switch block.Type {
					case "tool_use":
						if block.ToolUse != nil {
							ts.Tools = append(ts.Tools, block.ToolUse.Name)
							ts.ToolCount++
						}
					case "text":
						ts.TextLength += len(block.Text)
					}
				}
			}
		} else if stage.TaskID != "" {
			// Fallback to streaming events
			events := getTaskEvents(stage.TaskID)
			for _, e := range events {
				tn := e.TurnNum
				if tn == 0 {
					continue
				}
				if opts.TurnFilter > 0 && tn != opts.TurnFilter {
					continue
				}
				if turnMap[tn] == nil {
					turnMap[tn] = &turnSummary{TurnNumber: tn}
				}
				ts := turnMap[tn]

				switch e.StreamType {
				case "tool_use":
					if e.ToolName != "" {
						ts.Tools = append(ts.Tools, e.ToolName)
						ts.ToolCount++
					}
				case "text":
					ts.TextLength += len(e.Text)
				}
			}
		}

		// Convert map to sorted slice
		for tn := 1; tn <= len(turnMap)+100; tn++ { // scan reasonable range
			if ts, ok := turnMap[tn]; ok {
				ts.Tools = dedupeTools(ts.Tools)
				ss.TurnDetail = append(ss.TurnDetail, *ts)
				ss.ToolCalls += ts.ToolCount
			}
		}
		ss.TurnCount = len(ss.TurnDetail)

		summary.Stages = append(summary.Stages, ss)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(summary)
}

// printChainJSONFull outputs full chain data with filtering
func printChainJSONFull(chain *observatory.ExecutionChain, opts jsonExportOptions) {
	// Filter stages if StageFilter is set
	stagesToProcess := chain.Stages
	if opts.StageFilter > 0 {
		var filteredStages []*observatory.ChainStage
		for i, stage := range chain.Stages {
			if i+1 == opts.StageFilter {
				filteredStages = append(filteredStages, stage)
			}
		}
		stagesToProcess = filteredStages
	}

	// Create a copy of chain with filtered stages
	chainCopy := *chain
	chainCopy.Stages = stagesToProcess

	export := chainExport{
		Chain:  &chainCopy,
		Stages: make([]stageExport, 0, len(stagesToProcess)),
	}

	for _, stage := range stagesToProcess {

		se := stageExport{
			Stage: *stage,
		}

		// Resolve session info
		var sessionInfo *taskSessionInfo
		if stage.SessionID != "" {
			sessionInfo = &taskSessionInfo{SessionID: stage.SessionID}
		}
		if stage.TaskID != "" {
			if taskInfo := getSessionInfoFromTask(stage.TaskID); taskInfo != nil {
				sessionInfo = taskInfo
			}
		}

		// Get filtered messages
		if sessionInfo != nil && sessionInfo.SessionID != "" {
			allMessages := getChatMessagesInRange(sessionInfo.SessionID, sessionInfo.StartedAt, sessionInfo.CompletedAt)

			// Apply turn and tools filters
			for _, msg := range allMessages {
				if opts.TurnFilter > 0 && msg.TurnNumber != opts.TurnFilter {
					continue
				}

				if opts.ToolsOnly {
					// Filter to only tool_use and tool_result blocks
					var filteredContent []contentBlock
					for _, block := range msg.Content {
						if block.Type == "tool_use" || block.Type == "tool_result" {
							filteredContent = append(filteredContent, block)
						}
					}
					if len(filteredContent) == 0 {
						continue // Skip messages with no tool content
					}
					msg.Content = filteredContent
				}

				se.Messages = append(se.Messages, msg)
			}
		}

		// Set data status
		if len(se.Messages) > 0 {
			se.DataStatus = "available"
		} else if sessionInfo == nil || sessionInfo.SessionID == "" {
			se.DataStatus = "no_session_id"
			se.DataMessage = "Task has no linked session. Run 'ailang observatory sync-chat' after session completes."
		} else {
			se.DataStatus = "not_synced"
			se.DataMessage = fmt.Sprintf("Session %s exists but no chat messages found. Run 'ailang observatory sync-chat' to import.", sessionInfo.SessionID)
		}

		export.Stages = append(export.Stages, se)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(export)
}

// printChainJSON outputs full chain data as JSON (legacy wrapper)
// printChainJSON outputs full chain data as JSON (legacy wrapper)
func printChainJSON(chain *observatory.ExecutionChain) {
	printChainJSONFull(chain, jsonExportOptions{})
}

// taskSessionInfo contains session_id and time range for filtering
type taskSessionInfo struct {
	SessionID   string
	StartedAt   time.Time
	CompletedAt time.Time
}

// getSessionInfoFromTask looks up session_id and time range from coordinator.db tasks table
func getSessionInfoFromTask(taskID string) *taskSessionInfo {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dbPath := filepath.Join(homeDir, ".ailang", "state", "coordinator.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil
	}
	defer db.Close()

	var sessionID sql.NullString
	var startedAt, completedAt sql.NullTime
	err = db.QueryRow(`
		SELECT session_id, started_at, completed_at
		FROM tasks WHERE id = ?
	`, taskID).Scan(&sessionID, &startedAt, &completedAt)
	if err != nil || !sessionID.Valid {
		return nil
	}

	info := &taskSessionInfo{SessionID: sessionID.String}
	if startedAt.Valid {
		info.StartedAt = startedAt.Time
	}
	if completedAt.Valid {
		info.CompletedAt = completedAt.Time
	}
	return info
}

// getSessionIDFromTask looks up session_id from coordinator.db tasks table (legacy helper)
func getSessionIDFromTask(taskID string) string {
	info := getSessionInfoFromTask(taskID)
	if info == nil {
		return ""
	}
	return info.SessionID
}

// getChatMessages fetches chat messages from observatory.db with full content
func getChatMessages(sessionID string) []chatMessageExport {
	return getChatMessagesInRange(sessionID, time.Time{}, time.Time{})
}

// getChatMessagesInRange fetches chat messages within a time range
func getChatMessagesInRange(sessionID string, startTime, endTime time.Time) []chatMessageExport {
	dbPath := observatory.DefaultDatabasePath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil
	}
	defer db.Close()

	// Build query with optional time filtering
	query := `
		SELECT id, session_id, turn_number, role, content_json,
		       tokens_in, tokens_out, model, timestamp
		FROM chat_messages
		WHERE session_id = ?
	`
	args := []interface{}{sessionID}

	// Add time range filter if provided
	if !startTime.IsZero() {
		// Add a buffer before start time (1 min) to catch setup messages
		bufferedStart := startTime.Add(-1 * time.Minute)
		query += " AND timestamp >= ?"
		args = append(args, bufferedStart)
	}
	if !endTime.IsZero() {
		// Add a buffer after end time (1 min) to catch completion messages
		bufferedEnd := endTime.Add(1 * time.Minute)
		query += " AND timestamp <= ?"
		args = append(args, bufferedEnd)
	}

	query += " ORDER BY turn_number, timestamp"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var messages []chatMessageExport
	for rows.Next() {
		var msg chatMessageExport
		var contentJSON, model sql.NullString
		var timestamp time.Time

		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.TurnNumber, &msg.Role,
			&contentJSON, &msg.TokensIn, &msg.TokensOut, &model, &timestamp); err != nil {
			continue
		}

		msg.Model = model.String
		msg.Timestamp = timestamp.Format(time.RFC3339)

		// Parse content_json to get full tool data
		if contentJSON.Valid && contentJSON.String != "" {
			msg.Content = parseContentJSON(contentJSON.String)
		}

		messages = append(messages, msg)
	}

	return messages
}

// parseContentJSON parses the content_json into structured blocks
func parseContentJSON(jsonStr string) []contentBlock {
	// The actual structure has nested tool_use/tool_result objects
	var rawBlocks []struct {
		Type     string `json:"type"`
		Text     string `json:"text,omitempty"`
		Thinking string `json:"thinking,omitempty"`
		// Nested structure for tool_use
		ToolUse *struct {
			ID    string      `json:"id"`
			Name  string      `json:"name"`
			Input interface{} `json:"input"`
		} `json:"tool_use,omitempty"`
		// Nested structure for tool_result
		ToolResult *struct {
			ToolUseID string `json:"tool_use_id"`
			Content   string `json:"content"`
			IsError   bool   `json:"is_error"`
		} `json:"tool_result,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawBlocks); err != nil {
		return nil
	}

	blocks := make([]contentBlock, 0, len(rawBlocks))
	for _, raw := range rawBlocks {
		block := contentBlock{Type: raw.Type}

		switch raw.Type {
		case "text":
			block.Text = raw.Text
		case "thinking":
			block.Thinking = raw.Thinking
		case "tool_use":
			if raw.ToolUse != nil {
				block.ToolUse = &toolUseBlock{
					ID:    raw.ToolUse.ID,
					Name:  raw.ToolUse.Name,
					Input: raw.ToolUse.Input,
				}
			}
		case "tool_result":
			if raw.ToolResult != nil {
				block.ToolResult = &toolResultBlock{
					ToolUseID: raw.ToolResult.ToolUseID,
					Content:   raw.ToolResult.Content,
					IsError:   raw.ToolResult.IsError,
				}
			}
		}

		blocks = append(blocks, block)
	}

	return blocks
}

// printDetailedTurnEvents prints turn-by-turn details with tools and text
func printDetailedTurnEvents(events []taskEvent, prefix string) {
	// Group events by turn
	type turnData struct {
		tools    []string
		textPart strings.Builder
	}
	turns := make(map[int]*turnData)
	maxTurn := 0

	for _, e := range events {
		if e.TurnNum > maxTurn {
			maxTurn = e.TurnNum
		}
		if e.TurnNum == 0 {
			continue // Skip turn 0 events (status, etc.)
		}

		if turns[e.TurnNum] == nil {
			turns[e.TurnNum] = &turnData{}
		}
		td := turns[e.TurnNum]

		switch e.StreamType {
		case "tool_use":
			if e.ToolName != "" {
				td.tools = append(td.tools, e.ToolName)
			}
		case "text":
			if e.Text != "" {
				td.textPart.WriteString(e.Text)
			}
		}
	}

	// Print each turn
	turnNums := make([]int, 0, len(turns))
	for tn := range turns {
		turnNums = append(turnNums, tn)
	}
	sort.Ints(turnNums)

	for i, tn := range turnNums {
		td := turns[tn]
		isLast := i == len(turnNums)-1
		turnPrefix := "├──"
		if isLast {
			turnPrefix = "└──"
		}

		// Build turn line
		turnLine := fmt.Sprintf("Turn %d", tn)
		if len(td.tools) > 0 {
			// Deduplicate consecutive tools
			uniqueTools := dedupeTools(td.tools)
			turnLine += fmt.Sprintf(": %s", cyan(strings.Join(uniqueTools, ", ")))
		}
		fmt.Printf("%s%s %s\n", prefix, turnPrefix, turnLine)

		// Show text preview (first meaningful chunk)
		text := strings.TrimSpace(td.textPart.String())
		if len(text) > 0 {
			// Get first line or first 120 chars
			preview := getTextPreview(text, 120)
			if preview != "" {
				childPrefix := "│   "
				if isLast {
					childPrefix = "    "
				}
				fmt.Printf("%s%s   %s\n", prefix, childPrefix, dim(preview))
			}
		}
	}
}

// dedupeTools removes consecutive duplicate tool names
func dedupeTools(tools []string) []string {
	if len(tools) == 0 {
		return tools
	}
	result := []string{tools[0]}
	for i := 1; i < len(tools); i++ {
		if tools[i] != tools[i-1] {
			result = append(result, tools[i])
		}
	}
	return result
}

// getTextPreview returns a preview of text (first line or truncated)
func getTextPreview(text string, maxLen int) string {
	// Skip if mostly whitespace
	if len(strings.TrimSpace(text)) < 10 {
		return ""
	}
	// Get first non-empty line
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 10 && !strings.HasPrefix(line, "```") {
			if len(line) > maxLen {
				return line[:maxLen-3] + "..."
			}
			return line
		}
	}
	return ""
}

func formatChainDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	} else if ms < 60000 {
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	} else {
		mins := ms / 60000
		secs := (ms % 60000) / 1000
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
}

func truncateChainID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12] + "..."
}

func colorizeStatus(status string) string {
	switch status {
	case "active", "running":
		return cyan(status)
	case "completed":
		return green(status)
	case "failed":
		return red(status)
	case "pending", "pending_approval", "awaiting_approval":
		return yellow(status)
	default:
		return status
	}
}

func getStatusIcon(status string) string {
	switch status {
	case "completed":
		return green("✓")
	case "running", "active":
		return cyan("●")
	case "pending", "pending_approval", "awaiting_approval":
		return yellow("○")
	case "failed":
		return red("✗")
	default:
		return "○"
	}
}

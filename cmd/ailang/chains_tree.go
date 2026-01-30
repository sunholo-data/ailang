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

func chainsTreeCommand() {
	fs := flag.NewFlagSet("chains tree", flag.ExitOnError)
	detailed := fs.Bool("detailed", false, "Show execution details (turns, tools)")
	jsonOutput := fs.Bool("json", false, "Output raw JSON with full event data")
	summary := fs.Bool("summary", false, "JSON summary: turn numbers and tools only (no content)")
	toolsOnly := fs.Bool("tools", false, "JSON: show only tool calls (skip text/thinking)")
	turnNum := fs.Int("turn", 0, "JSON: show only specific turn number")
	stageNum := fs.Int("stage", 0, "Filter to specific stage (1-based)")
	sessionID := fs.String("session", "", "Session ID to get full chat data (bypasses chain lookup)")
	// Compact output modes
	errorsOnly := fs.Bool("errors", false, "Only show failed or stuck stages (human-readable)")
	handoffsOnly := fs.Bool("handoffs", false, "Show handoff summary only (compact view)")
	lastTurnOnly := fs.Bool("last-turn", false, "Show only the final turn per stage")
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

	// Compact output modes
	if *errorsOnly {
		printChainErrorsOnly(ctx, backend, chain)
		return
	}
	if *handoffsOnly {
		printChainHandoffsOnly(chain)
		return
	}
	if *lastTurnOnly {
		printChainLastTurnOnly(ctx, backend, chain)
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
	// PREFERRED: Try deterministic task_id query first (M-DETERMINISTIC-CHAT-LINKING)
	// This works when sync-chat has propagated correlation IDs from sessions to chat_messages
	messages := getChatMessagesForTask(taskID)
	if len(messages) > 0 {
		printChatTurnDetails(messages, prefix)
		return
	}

	// FALLBACK: Get session info and use timestamp filtering (legacy approach)
	sessionInfo := getSessionInfoFromTask(taskID)
	if sessionInfo != nil && sessionInfo.SessionID != "" {
		messages = getChatMessagesInRange(sessionInfo.SessionID, sessionInfo.StartedAt, sessionInfo.CompletedAt)
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

// printChainErrorsOnly shows only failed or stuck stages (compact for AI)
func printChainErrorsOnly(ctx context.Context, backend *observatory.SQLiteBackend, chain *observatory.ExecutionChain) {
	fmt.Printf("Chain %s [%s]\n", truncateChainID(chain.ID), colorizeStatus(string(chain.Status)))

	hasIssues := false
	for i, stage := range chain.Stages {
		// Check for issues
		issues := []string{}

		if stage.Status == "failed" {
			issues = append(issues, "FAILED")
		}
		if stage.Status == "awaiting_approval" || stage.ApprovalStatus == "pending" {
			if stage.StartedAt != nil {
				waitTime := time.Since(*stage.StartedAt)
				if waitTime > time.Hour {
					issues = append(issues, fmt.Sprintf("stuck %s", formatDurationHuman(waitTime)))
				}
			}
		}

		// Check for missing data
		if stage.TaskID != "" {
			messages := getChatMessagesForTask(stage.TaskID)
			if len(messages) == 0 && stage.Status != "pending" {
				issues = append(issues, "no chat linked")
			}
		}
		if stage.SessionID == "" && stage.Status != "pending" && stage.Status != "failed" {
			issues = append(issues, "no session")
		}

		if len(issues) > 0 {
			hasIssues = true
			fmt.Printf("  Stage %d: %s [%s]\n", i+1, stage.AgentID, string(stage.Status))
			for _, issue := range issues {
				fmt.Printf("    ⚠ %s\n", issue)
			}
		}
	}

	if !hasIssues {
		fmt.Println(green("  ✓ No issues detected"))
	}
}

// printChainHandoffsOnly shows a compact handoff summary (one line per stage)
func printChainHandoffsOnly(chain *observatory.ExecutionChain) {
	// Header line
	var parts []string
	parts = append(parts, fmt.Sprintf("Chain %s [%s]", truncateChainID(chain.ID), colorizeStatus(string(chain.Status))))

	// Calculate duration and cost
	if !chain.CreatedAt.IsZero() {
		var endTime time.Time
		if chain.CompletedAt != nil && !chain.CompletedAt.IsZero() {
			endTime = *chain.CompletedAt
		} else {
			endTime = time.Now()
		}
		parts = append(parts, formatDurationHuman(endTime.Sub(chain.CreatedAt)))
	}
	if chain.TotalCost > 0 {
		parts = append(parts, fmt.Sprintf("$%.2f", chain.TotalCost))
	}
	fmt.Println(strings.Join(parts, " "))

	// Stage flow: 1. agent ✓ → 2. agent ⏳ → 3. agent
	var stages []string
	for i, stage := range chain.Stages {
		icon := getStatusIcon(string(stage.Status))
		entry := fmt.Sprintf("%d. %s %s", i+1, stage.AgentID, icon)
		stages = append(stages, entry)
	}
	fmt.Printf("  %s\n", strings.Join(stages, " → "))
}

// printChainLastTurnOnly shows only the final turn per stage (for quick context)
func printChainLastTurnOnly(ctx context.Context, backend *observatory.SQLiteBackend, chain *observatory.ExecutionChain) {
	fmt.Printf("Chain %s [%s] - Last Turn Summary\n", truncateChainID(chain.ID), colorizeStatus(string(chain.Status)))
	fmt.Println()

	for i, stage := range chain.Stages {
		fmt.Printf("Stage %d: %s [%s]\n", i+1, stage.AgentID, colorizeStatus(string(stage.Status)))

		// Get chat messages
		var messages []chatMessageExport
		if stage.TaskID != "" {
			messages = getChatMessagesForTask(stage.TaskID)
		} else if stage.SessionID != "" {
			messages = getChatMessages(stage.SessionID)
		}

		if len(messages) == 0 {
			fmt.Println("  (no chat data available)")
			fmt.Println()
			continue
		}

		// Find the last assistant message (the final turn)
		var lastAssistant *chatMessageExport
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "assistant" {
				lastAssistant = &messages[i]
				break
			}
		}

		if lastAssistant == nil {
			fmt.Println("  (no assistant response found)")
			fmt.Println()
			continue
		}

		// Extract tools and text from content blocks
		tools := []string{}
		var textContent string
		for _, block := range lastAssistant.Content {
			if block.Type == "tool_use" && block.ToolUse != nil {
				tools = append(tools, block.ToolUse.Name)
			}
			if block.Type == "text" && block.Text != "" {
				textContent = block.Text
			}
		}

		// Print summary of last turn
		fmt.Printf("  Turn %d | Tools: ", lastAssistant.TurnNumber)
		if len(tools) > 0 {
			fmt.Println(strings.Join(tools, ", "))
		} else {
			fmt.Println("(none)")
		}

		// Print truncated content
		content := strings.TrimSpace(textContent)
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		if content != "" {
			// Indent the content
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				if len(line) > 100 {
					line = line[:100] + "..."
				}
				fmt.Printf("  %s\n", line)
			}
		}
		fmt.Println()
	}
}

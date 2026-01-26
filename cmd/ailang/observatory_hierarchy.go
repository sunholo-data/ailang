package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/claudehistory"
	"github.com/sunholo/ailang/internal/observatory"
)

func observatoryHierarchyCommand() {
	fs := flag.NewFlagSet("observatory hierarchy", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	includeSpans := fs.Bool("spans", true, "Include individual spans in output")
	groupByTurns := fs.Bool("turns", false, "Group spans by conversation turn (Session → Turn 1 → Turn 2 → ...)")
	includeChat := fs.Bool("chat", false, "Include conversation history (user prompts, assistant responses)")

	// Skip "ailang observatory hierarchy" args
	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Get ID from positional argument
	if fs.NArg() < 1 {
		fmt.Println("Usage: ailang observatory hierarchy [options] <id>")
		fmt.Println()
		fmt.Println("The <id> argument auto-detects type:")
		fmt.Println("  task-XXXXXXXX   Coordinator task ID")
		fmt.Println("  32-char hex     Trace ID (e.g., 0ebf5e64bb654fcc1d19256b59f05ae3)")
		fmt.Println("  UUID            Session ID (e.g., 4df60536-caed-4e2f-af2c-e386c361f4e7)")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -json           Output as JSON")
		fmt.Println("  -spans=false    Hide individual span details")
		fmt.Println("  -turns          Group spans by conversation turn")
		fmt.Println("  -chat           Include conversation history (prompts/responses)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang observatory hierarchy task-29404032")
		fmt.Println("  ailang observatory hierarchy -json task-29404032")
		fmt.Println("  ailang observatory hierarchy -turns task-29404032")
		fmt.Println("  ailang observatory hierarchy -chat <session-uuid>")
		return
	}

	id := fs.Arg(0)
	idType := detectIDType(id)

	ctx := context.Background()

	// Open observatory database
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening observatory database: %v\n", err)
		fmt.Fprintf(os.Stderr, "Database path: %s\n", dbPath)
		os.Exit(1)
	}
	defer backend.Close()

	// Build unified hierarchy based on ID type
	result, err := buildUnifiedHierarchy(ctx, backend, id, idType, *includeSpans)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building hierarchy: %v\n", err)
		os.Exit(1)
	}

	// Apply turn grouping if requested
	if *groupByTurns && result.SpanNodes != nil {
		result.TurnGrouped = observatory.GroupSpansByTurn(result.SpanNodes)
	}

	// Fetch chat history if requested
	if *includeChat {
		sessionID := ""
		// Determine session ID based on ID type
		switch idType {
		case IDTypeSession:
			sessionID = id
		case IDTypeTask, IDTypeTrace:
			// Try to extract session ID from spans
			for _, span := range result.Spans {
				if span != nil && span.ID != "" {
					// Look up the span to get session.id attribute
					// For now, use the ID directly if it looks like a session
					break
				}
			}
		}

		if sessionID != "" {
			importer := claudehistory.NewImporter(backend.DB())
			messages, err := importer.GetChatMessages(ctx, sessionID)
			if err == nil && len(messages) > 0 {
				result.ChatMessages = messages
			}
		}
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Pretty print hierarchy
	if *groupByTurns && result.TurnGrouped != nil {
		printTurnGroupedHierarchy(result)
	} else {
		printUnifiedHierarchy(result)
	}
}

// IDType represents the detected type of an ID
type IDType string

const (
	IDTypeTask    IDType = "task"
	IDTypeTrace   IDType = "trace"
	IDTypeSession IDType = "session"
	IDTypeUnknown IDType = "unknown"
)

// detectIDType determines the type of ID from its format
func detectIDType(id string) IDType {
	// task-XXXXXXXX format
	if strings.HasPrefix(id, "task-") {
		return IDTypeTask
	}

	// 32-char hex = trace_id
	if len(id) == 32 && isHexString(id) {
		return IDTypeTrace
	}

	// UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
	if len(id) == 36 && strings.Count(id, "-") == 4 {
		return IDTypeSession
	}

	// eval-timestamp format
	if strings.HasPrefix(id, "eval-") {
		return IDTypeTask // treat as task-like
	}

	return IDTypeUnknown
}

// isHexString checks if a string contains only hex characters
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// UnifiedHierarchy represents the complete hierarchy for display
type UnifiedHierarchy struct {
	IDType       IDType                            `json:"id_type"`
	ID           string                            `json:"id"`
	Task         *unifiedTask                      `json:"task,omitempty"`
	Message      *unifiedMessage                   `json:"message,omitempty"`
	Spans        []*unifiedSpan                    `json:"spans,omitempty"`
	SpanNodes    []*observatory.SpanNode           `json:"-"` // Raw span nodes for grouping (not serialized)
	TurnGrouped  *observatory.TurnGroupedHierarchy `json:"turn_grouped,omitempty"`
	Handoffs     []*unifiedHandoff                 `json:"handoffs,omitempty"`
	ParentChain  []*unifiedTaskSummary             `json:"parent_chain,omitempty"`
	ChatMessages []*claudehistory.ChatMessage      `json:"chat_messages,omitempty"` // Conversation history
	Stats        *unifiedStats                     `json:"stats"`
}

type unifiedTask struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	AgentID        string  `json:"agent_id,omitempty"`
	Status         string  `json:"status"`
	ParentTaskID   string  `json:"parent_task_id,omitempty"`
	Cost           float64 `json:"cost"`
	DurationMs     int64   `json:"duration_ms"`
	TokensIn       int64   `json:"tokens_in"`
	TokensOut      int64   `json:"tokens_out"`
	ApprovalStatus string  `json:"approval_status,omitempty"`
}

type unifiedMessage struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	FromAgent    string `json:"from_agent"`
	ToInbox      string `json:"to_inbox"`
	ParentTaskID string `json:"parent_task_id,omitempty"`
	Status       string `json:"status"`
}

type unifiedSpan struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	DurationMs int64          `json:"duration_ms"`
	Cost       float64        `json:"cost,omitempty"`
	TokensIn   int64          `json:"tokens_in,omitempty"`
	TokensOut  int64          `json:"tokens_out,omitempty"`
	Status     string         `json:"status"`
	Children   []*unifiedSpan `json:"children,omitempty"`
}

type unifiedHandoff struct {
	TargetTaskID string `json:"target_task_id"`
	TargetAgent  string `json:"target_agent"`
	Status       string `json:"status"`
}

type unifiedTaskSummary struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	AgentID string `json:"agent_id,omitempty"`
	Status  string `json:"status"`
}

type unifiedStats struct {
	TotalSpans       int     `json:"total_spans"`
	TotalCost        float64 `json:"total_cost"`
	TotalTokens      int64   `json:"total_tokens"`
	TotalDurationMs  int64   `json:"total_duration_ms"`
	HandoffCount     int     `json:"handoff_count"`
	PendingApprovals int     `json:"pending_approvals"`
}

// buildUnifiedHierarchy builds the complete hierarchy based on ID type
func buildUnifiedHierarchy(ctx context.Context, backend observatory.Backend, id string, idType IDType, includeSpans bool) (*UnifiedHierarchy, error) {
	result := &UnifiedHierarchy{
		IDType: idType,
		ID:     id,
		Stats:  &unifiedStats{},
	}

	switch idType {
	case IDTypeTask:
		return buildHierarchyFromTask(ctx, backend, id, includeSpans)
	case IDTypeTrace:
		return buildHierarchyFromTrace(ctx, backend, id, includeSpans)
	case IDTypeSession:
		return buildHierarchyFromSession(ctx, backend, id, includeSpans)
	default:
		return result, fmt.Errorf("unknown ID type for: %s", id)
	}
}

// buildHierarchyFromTask builds hierarchy starting from a task ID
func buildHierarchyFromTask(ctx context.Context, backend observatory.Backend, taskID string, includeSpans bool) (*UnifiedHierarchy, error) {
	result := &UnifiedHierarchy{
		IDType: IDTypeTask,
		ID:     taskID,
		Stats:  &unifiedStats{},
	}

	// Get task hierarchy using existing function
	opts := observatory.HierarchyOptions{
		IncludeSpans: includeSpans,
	}
	hierarchy, err := observatory.GetTaskHierarchy(ctx, backend, taskID, opts)
	if err != nil {
		return nil, fmt.Errorf("get task hierarchy: %w", err)
	}

	if hierarchy.Task != nil {
		result.Task = &unifiedTask{
			ID:           hierarchy.Task.ID,
			Title:        hierarchy.Task.Title,
			Status:       string(hierarchy.Task.Status),
			ParentTaskID: hierarchy.Task.ParentTaskID,
			Cost:         hierarchy.Task.TotalCostUSD,
			TokensIn:     hierarchy.Task.TotalTokensIn,
			TokensOut:    hierarchy.Task.TotalTokensOut,
		}
		result.Stats.TotalCost = hierarchy.Task.TotalCostUSD
		result.Stats.TotalTokens = hierarchy.Task.TotalTokensIn + hierarchy.Task.TotalTokensOut
	}

	// Build span tree from agents
	if includeSpans {
		for _, agent := range hierarchy.Agents {
			if result.Task != nil && result.Task.AgentID == "" && agent.Agent != nil {
				result.Task.AgentID = agent.Agent.AgentID
			}
			for _, traceHierarchy := range agent.Traces {
				for _, spanNode := range traceHierarchy.Spans {
					result.Spans = append(result.Spans, convertSpanNode(spanNode))
					result.SpanNodes = append(result.SpanNodes, spanNode) // Keep raw nodes for turn grouping
					result.Stats.TotalSpans++
				}
			}
		}
	}

	// Get child tasks (handoffs) from observatory tasks table
	childTasks, err := backend.ListTasks(ctx, observatory.TaskListOptions{
		ParentTaskID: taskID,
		Limit:        100,
	})
	if err == nil {
		for _, child := range childTasks {
			handoff := &unifiedHandoff{
				TargetTaskID: child.ID,
				Status:       string(child.Status),
			}
			result.Handoffs = append(result.Handoffs, handoff)
			result.Stats.HandoffCount++
		}
	}

	// Build parent chain (walk up parent_task_id)
	if hierarchy.Task != nil && hierarchy.Task.ParentTaskID != "" {
		result.ParentChain = buildParentChain(ctx, backend, hierarchy.Task.ParentTaskID)
	}

	return result, nil
}

// buildParentChain walks up the parent_task_id chain
func buildParentChain(ctx context.Context, backend observatory.Backend, parentTaskID string) []*unifiedTaskSummary {
	var chain []*unifiedTaskSummary
	currentID := parentTaskID
	visited := make(map[string]bool)

	for currentID != "" && !visited[currentID] && len(chain) < 10 {
		visited[currentID] = true
		task, err := backend.GetTask(ctx, currentID)
		if err != nil || task == nil {
			break
		}
		chain = append(chain, &unifiedTaskSummary{
			ID:     task.ID,
			Title:  task.Title,
			Status: string(task.Status),
		})
		currentID = task.ParentTaskID
	}

	return chain
}

// buildHierarchyFromTrace builds hierarchy from a trace ID
func buildHierarchyFromTrace(ctx context.Context, backend observatory.Backend, traceID string, includeSpans bool) (*UnifiedHierarchy, error) {
	result := &UnifiedHierarchy{
		IDType: IDTypeTrace,
		ID:     traceID,
		Stats:  &unifiedStats{},
	}

	// Get spans for this trace
	spans, err := backend.ListSpans(ctx, observatory.SpanListOptions{
		TraceID: traceID,
		Limit:   1000,
	})
	if err != nil {
		return nil, fmt.Errorf("list spans by trace: %w", err)
	}

	if len(spans) == 0 {
		return result, nil
	}

	// Build span tree
	if includeSpans {
		result.Spans = buildSpanTreeFromList(spans)
		result.SpanNodes = buildSpanNodeTreeFromList(spans) // For turn grouping
		result.Stats.TotalSpans = len(spans)
	}

	// Calculate stats
	for _, span := range spans {
		result.Stats.TotalCost += span.CostUSD
		result.Stats.TotalTokens += span.TokensIn + span.TokensOut
		if span.DurationMs > result.Stats.TotalDurationMs {
			result.Stats.TotalDurationMs = span.DurationMs
		}
	}

	// Try to find linked task
	for _, span := range spans {
		if span.TaskID != "" {
			// Found a task link - fetch task info
			task, err := backend.GetTask(ctx, span.TaskID)
			if err == nil && task != nil {
				result.Task = &unifiedTask{
					ID:           task.ID,
					Title:        task.Title,
					Status:       string(task.Status),
					ParentTaskID: task.ParentTaskID,
				}
				break
			}
		}
	}

	return result, nil
}

// buildHierarchyFromSession builds hierarchy from a session UUID
func buildHierarchyFromSession(ctx context.Context, backend observatory.Backend, sessionID string, includeSpans bool) (*UnifiedHierarchy, error) {
	result := &UnifiedHierarchy{
		IDType: IDTypeSession,
		ID:     sessionID,
		Stats:  &unifiedStats{},
	}

	// For session IDs, we query spans by the session.id attribute
	// The session ID might also be stored as task_id for Claude Code sessions
	spans, err := backend.ListSpans(ctx, observatory.SpanListOptions{
		TaskID: sessionID, // Claude Code uses session ID as task_id
		Limit:  1000,
	})
	if err != nil {
		return nil, fmt.Errorf("list spans by session: %w", err)
	}

	if len(spans) == 0 {
		return result, nil
	}

	// Build span tree
	if includeSpans {
		result.Spans = buildSpanTreeFromList(spans)
		result.SpanNodes = buildSpanNodeTreeFromList(spans) // For turn grouping
		result.Stats.TotalSpans = len(spans)
	}

	// Calculate stats
	for _, span := range spans {
		result.Stats.TotalCost += span.CostUSD
		result.Stats.TotalTokens += span.TokensIn + span.TokensOut
		if span.DurationMs > result.Stats.TotalDurationMs {
			result.Stats.TotalDurationMs = span.DurationMs
		}
	}

	return result, nil
}

// buildSpanNodeTreeFromList converts a flat span list to observatory.SpanNode tree.
// This is needed for turn-based grouping which uses the observatory types.
func buildSpanNodeTreeFromList(spans []*observatory.Span) []*observatory.SpanNode {
	if len(spans) == 0 {
		return nil
	}

	// Build node map
	nodeMap := make(map[string]*observatory.SpanNode)
	for _, span := range spans {
		nodeMap[span.ID] = &observatory.SpanNode{Span: span}
	}

	// Build parent-child relationships
	var roots []*observatory.SpanNode
	for _, span := range spans {
		node := nodeMap[span.ID]
		if span.ParentSpanID == "" {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[span.ParentSpanID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// Parent not in our set, treat as root
			roots = append(roots, node)
		}
	}

	return roots
}

// buildSpanTreeFromList converts a flat span list to a tree
func buildSpanTreeFromList(spans []*observatory.Span) []*unifiedSpan {
	if len(spans) == 0 {
		return nil
	}

	// Build node map
	nodeMap := make(map[string]*unifiedSpan)
	for _, span := range spans {
		nodeMap[span.ID] = &unifiedSpan{
			ID:         span.ID,
			Name:       span.Name,
			DurationMs: span.DurationMs,
			Cost:       span.CostUSD,
			TokensIn:   span.TokensIn,
			TokensOut:  span.TokensOut,
			Status:     string(span.Status),
		}
	}

	// Build parent-child relationships
	var roots []*unifiedSpan
	for _, span := range spans {
		node := nodeMap[span.ID]
		if span.ParentSpanID == "" {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[span.ParentSpanID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// Parent not in our set, treat as root
			roots = append(roots, node)
		}
	}

	return roots
}

// convertSpanNode converts observatory SpanNode to unifiedSpan
func convertSpanNode(node *observatory.SpanNode) *unifiedSpan {
	if node == nil || node.Span == nil {
		return nil
	}

	result := &unifiedSpan{
		ID:         node.Span.ID,
		Name:       node.Span.Name,
		DurationMs: node.Span.DurationMs,
		Cost:       node.Span.CostUSD,
		TokensIn:   node.Span.TokensIn,
		TokensOut:  node.Span.TokensOut,
		Status:     string(node.Span.Status),
	}

	for _, child := range node.Children {
		if converted := convertSpanNode(child); converted != nil {
			result.Children = append(result.Children, converted)
		}
	}

	return result
}

// printUnifiedHierarchy prints the hierarchy as a tree
func printUnifiedHierarchy(h *UnifiedHierarchy) {
	fmt.Println(strings.Repeat("═", 80))
	fmt.Printf("Hierarchy for %s (%s)\n", h.ID, h.IDType)
	fmt.Println(strings.Repeat("═", 80))

	// Print task info
	if h.Task != nil {
		agentInfo := ""
		if h.Task.AgentID != "" {
			agentInfo = fmt.Sprintf(" (%s)", h.Task.AgentID)
		}
		statusBadge := fmt.Sprintf(" [%s]", h.Task.Status)

		fmt.Printf("\n⬢ Task: %s%s%s\n", h.Task.ID, agentInfo, statusBadge)
		if h.Task.Title != "" && h.Task.Title != h.Task.ID {
			fmt.Printf("  Title: %s\n", truncateString(h.Task.Title, 70))
		}
		if h.Task.ParentTaskID != "" {
			fmt.Printf("  Parent: %s\n", h.Task.ParentTaskID)
		}
		fmt.Printf("  Cost: $%.4f | Tokens: %d in / %d out\n",
			h.Task.Cost, h.Task.TokensIn, h.Task.TokensOut)
	}

	// Print parent chain (ancestry)
	if len(h.ParentChain) > 0 {
		fmt.Printf("\n┌─ Parent Chain (%d levels):\n", len(h.ParentChain))
		for i, parent := range h.ParentChain {
			var prefix string
			if i == len(h.ParentChain)-1 {
				prefix = "└─ "
			} else {
				prefix = "├─ "
			}
			fmt.Printf("%s%s [%s]\n", prefix, parent.ID, parent.Status)
		}
	}

	// Print message info
	if h.Message != nil {
		fmt.Printf("\n📨 Message: %s\n", h.Message.ID)
		fmt.Printf("   From: %s → To: %s\n", h.Message.FromAgent, h.Message.ToInbox)
		if h.Message.ParentTaskID != "" {
			fmt.Printf("   Parent Task: %s\n", h.Message.ParentTaskID)
		}
	}

	// Print spans
	if len(h.Spans) > 0 {
		fmt.Printf("\n├─ Spans: %d total\n", h.Stats.TotalSpans)
		for i, span := range h.Spans {
			isLast := i == len(h.Spans)-1 && len(h.Handoffs) == 0
			printSpanTree(span, "│  ", isLast)
		}
	}

	// Print handoffs
	if len(h.Handoffs) > 0 {
		fmt.Printf("\n└─ Handoffs: %d\n", len(h.Handoffs))
		for i, handoff := range h.Handoffs {
			prefix := "   ├─ "
			if i == len(h.Handoffs)-1 {
				prefix = "   └─ "
			}
			agent := handoff.TargetAgent
			if agent == "" {
				agent = "?"
			}
			fmt.Printf("%s→ %s (%s) [%s]\n", prefix, handoff.TargetTaskID, agent, handoff.Status)
		}
	}

	// Print chat messages if available
	if len(h.ChatMessages) > 0 {
		fmt.Printf("\n💬 Conversation History: %d messages\n", len(h.ChatMessages))
		fmt.Println(strings.Repeat("─", 80))
		for _, msg := range h.ChatMessages {
			roleIcon := "👤"
			if msg.Role == "assistant" {
				roleIcon = "🤖"
			}
			fmt.Printf("\n%s [Turn %d] %s\n", roleIcon, msg.TurnNumber, msg.Role)
			if msg.ContentText != "" {
				// Truncate long messages for display
				content := msg.ContentText
				if len(content) > 500 {
					content = content[:500] + "..."
				}
				fmt.Printf("   %s\n", strings.ReplaceAll(content, "\n", "\n   "))
			}
			if msg.ContentThinking != "" {
				thinking := msg.ContentThinking
				if len(thinking) > 200 {
					thinking = thinking[:200] + "..."
				}
				fmt.Printf("   💭 %s\n", strings.ReplaceAll(thinking, "\n", "\n   "))
			}
			if msg.TokensIn > 0 || msg.TokensOut > 0 {
				fmt.Printf("   📊 Tokens: %d in / %d out", msg.TokensIn, msg.TokensOut)
				if msg.Model != "" {
					fmt.Printf(" | Model: %s", msg.Model)
				}
				fmt.Println()
			}
		}
	}

	// Print stats summary
	fmt.Println()
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("Stats: %d spans | $%.4f | %d tokens | %s\n",
		h.Stats.TotalSpans,
		h.Stats.TotalCost,
		h.Stats.TotalTokens,
		formatHierarchyDuration(h.Stats.TotalDurationMs))
	if h.Stats.HandoffCount > 0 {
		fmt.Printf("       %d handoffs", h.Stats.HandoffCount)
		if h.Stats.PendingApprovals > 0 {
			fmt.Printf(" | %d pending approvals", h.Stats.PendingApprovals)
		}
		fmt.Println()
	}
}

// printTurnGroupedHierarchy prints the hierarchy organized by conversation turns.
func printTurnGroupedHierarchy(h *UnifiedHierarchy) {
	fmt.Println(strings.Repeat("═", 80))
	fmt.Printf("Turn-Grouped Hierarchy for %s (%s)\n", h.ID, h.IDType)
	fmt.Println(strings.Repeat("═", 80))

	// Print task info if available
	if h.Task != nil {
		agentInfo := ""
		if h.Task.AgentID != "" {
			agentInfo = fmt.Sprintf(" (%s)", h.Task.AgentID)
		}
		statusBadge := fmt.Sprintf(" [%s]", h.Task.Status)
		fmt.Printf("\n⬢ Task: %s%s%s\n", h.Task.ID, agentInfo, statusBadge)
		if h.Task.Title != "" && h.Task.Title != h.Task.ID {
			fmt.Printf("  Title: %s\n", truncateString(h.Task.Title, 70))
		}
	}

	tg := h.TurnGrouped
	if tg == nil {
		fmt.Println("\n(No turn data available)")
		return
	}

	// Print session info
	if tg.Session != nil {
		fmt.Printf("\n┌─ Session: %s\n", tg.Session.Name)
		if tg.Session.Provider != "" || tg.Session.Model != "" {
			fmt.Printf("│  Provider: %s | Model: %s\n", tg.Session.Provider, tg.Session.Model)
		}
		fmt.Printf("│  Duration: %s | Cost: $%.4f | Tokens: %d in / %d out\n",
			formatHierarchyDuration(tg.Session.DurationMs),
			tg.Session.Cost,
			tg.Session.TokensIn,
			tg.Session.TokensOut)
	}

	// Print turns
	if len(tg.Turns) > 0 {
		fmt.Printf("│\n")
		for i, turn := range tg.Turns {
			isLast := i == len(tg.Turns)-1

			// Turn header
			prefix := "├"
			nextPrefix := "│"
			if isLast {
				prefix = "└"
				nextPrefix = " "
			}

			fmt.Printf("%s─ Turn %d [%s] $%.4f [%d→%d tokens]\n",
				prefix,
				turn.TurnNumber,
				formatHierarchyDuration(turn.DurationMs),
				turn.Cost,
				turn.TokensIn,
				turn.TokensOut)

			// Print tools within turn
			if len(turn.Tools) > 0 {
				for j, tool := range turn.Tools {
					toolIsLast := j == len(turn.Tools)-1
					toolPrefix := "├─"
					if toolIsLast {
						toolPrefix = "└─"
					}

					statusIcon := "✓"
					if tool.Status == "error" {
						statusIcon = "✗"
					}

					toolName := tool.ToolName
					if toolName == "" {
						toolName = tool.Name
					}

					fmt.Printf("%s  %s %s %s [%s]\n",
						nextPrefix,
						toolPrefix,
						toolName,
						statusIcon,
						formatHierarchyDuration(tool.DurationMs))
				}
			}

			// Add spacing between turns
			if !isLast {
				fmt.Printf("%s\n", nextPrefix)
			}
		}
	}

	// Print stats summary
	fmt.Println()
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("Stats: %d turns | %d tools | $%.4f | %d tokens | %s\n",
		tg.Stats.TotalTurns,
		tg.Stats.TotalTools,
		tg.Stats.TotalCost,
		tg.Stats.TotalTokens,
		formatHierarchyDuration(tg.Stats.DurationMs))

	// Print handoffs if any
	if len(h.Handoffs) > 0 {
		fmt.Printf("       %d handoffs\n", len(h.Handoffs))
	}
}

// printSpanTree prints a span and its children as a tree
func printSpanTree(span *unifiedSpan, prefix string, isLast bool) {
	if span == nil {
		return
	}

	connector := "├─ "
	if isLast {
		connector = "└─ "
	}

	// Format metrics
	metrics := ""
	if span.DurationMs > 0 {
		metrics += fmt.Sprintf(" %s", formatHierarchyDuration(span.DurationMs))
	}
	if span.Cost > 0 {
		metrics += fmt.Sprintf(" $%.4f", span.Cost)
	}
	if span.TokensIn > 0 || span.TokensOut > 0 {
		metrics += fmt.Sprintf(" [%d→%d]", span.TokensIn, span.TokensOut)
	}

	fmt.Printf("%s%s%s%s\n", prefix, connector, span.Name, metrics)

	// Print children
	childPrefix := prefix
	if isLast {
		childPrefix += "   "
	} else {
		childPrefix += "│  "
	}

	for i, child := range span.Children {
		printSpanTree(child, childPrefix, i == len(span.Children)-1)
	}
}

// formatHierarchyDuration formats milliseconds as human-readable duration for hierarchy output
func formatHierarchyDuration(ms int64) string {
	if ms >= 60000 {
		return fmt.Sprintf("%.1fm", float64(ms)/60000)
	} else if ms >= 1000 {
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	}
	return fmt.Sprintf("%dms", ms)
}

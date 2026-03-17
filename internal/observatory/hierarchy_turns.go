// Package observatory provides turn-based span grouping for hierarchy views.
package observatory

import (
	"fmt"
	"sort"
	"strings"
)

// =============================================================================
// Turn-Based Grouping: Structure spans by conversation turns
// =============================================================================
// When viewing execution hierarchies, it's useful to see spans grouped by
// conversation turn rather than raw parent-child relationships. This creates
// a more intuitive view: Session → Turn 1 → Turn 2 → Turn 3, with tools
// nested under their respective turns.
// =============================================================================

// TurnGroupedHierarchy represents spans organized by conversation turns.
type TurnGroupedHierarchy struct {
	Session *TurnGroupSession `json:"session,omitempty"`
	Turns   []*TurnGroup      `json:"turns"`
	Stats   *TurnGroupStats   `json:"stats"`
}

// TurnGroupSession represents the top-level session/executor span.
type TurnGroupSession struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	DurationMs int64   `json:"duration_ms"`
	Cost       float64 `json:"cost"`
	TokensIn   int64   `json:"tokens_in"`
	TokensOut  int64   `json:"tokens_out"`
	Provider   string  `json:"provider,omitempty"`
	Model      string  `json:"model,omitempty"`
}

// TurnGroup represents a single conversation turn with its tools.
type TurnGroup struct {
	TurnNumber int         `json:"turn_number"`
	SpanID     string      `json:"span_id"`
	DurationMs int64       `json:"duration_ms"`
	Cost       float64     `json:"cost"`
	TokensIn   int64       `json:"tokens_in"`
	TokensOut  int64       `json:"tokens_out"`
	Tools      []*TurnTool `json:"tools,omitempty"`
}

// TurnTool represents a tool call within a turn.
type TurnTool struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	ToolName   string  `json:"tool_name,omitempty"` // Extracted tool name (e.g., "Read", "Bash")
	DurationMs int64   `json:"duration_ms"`
	Cost       float64 `json:"cost,omitempty"`
	Status     string  `json:"status"`
}

// TurnGroupStats contains aggregate statistics for the turn-grouped view.
type TurnGroupStats struct {
	TotalTurns  int     `json:"total_turns"`
	TotalTools  int     `json:"total_tools"`
	TotalCost   float64 `json:"total_cost"`
	TotalTokens int64   `json:"total_tokens"`
	DurationMs  int64   `json:"duration_ms"`
}

// GroupSpansByTurn transforms a span tree into a turn-based hierarchy.
// This is useful for displaying execution in a more intuitive way.
func GroupSpansByTurn(spans []*SpanNode) *TurnGroupedHierarchy {
	result := &TurnGroupedHierarchy{
		Turns: make([]*TurnGroup, 0),
		Stats: &TurnGroupStats{},
	}

	if len(spans) == 0 {
		return result
	}

	// Find the root session/executor span
	var sessionNode *SpanNode
	for _, node := range spans {
		if node.Span != nil && isSessionOrExecutorSpan(node.Span) {
			sessionNode = node
			break
		}
	}

	// If no session found, use the first root span
	if sessionNode == nil && len(spans) > 0 {
		sessionNode = spans[0]
	}

	if sessionNode != nil && sessionNode.Span != nil {
		result.Session = &TurnGroupSession{
			ID:         sessionNode.Span.ID,
			Name:       sessionNode.Span.Name,
			DurationMs: sessionNode.Span.DurationMs,
			Cost:       sessionNode.Span.CostUSD,
			TokensIn:   sessionNode.Span.TokensIn,
			TokensOut:  sessionNode.Span.TokensOut,
			Provider:   string(sessionNode.Span.Provider),
			Model:      sessionNode.Span.Model,
		}
		result.Stats.DurationMs = sessionNode.Span.DurationMs
	}

	// Collect all turns recursively from the span tree
	turnsMap := make(map[int]*turnCollector)
	collectTurnsForGrouping(spans, turnsMap)

	// Convert map to sorted slice
	turnNumbers := make([]int, 0, len(turnsMap))
	for num := range turnsMap {
		turnNumbers = append(turnNumbers, num)
	}
	sort.Ints(turnNumbers)

	// Build turn groups
	for _, turnNum := range turnNumbers {
		tc := turnsMap[turnNum]
		turn := &TurnGroup{
			TurnNumber: turnNum,
			SpanID:     tc.spanID,
			DurationMs: tc.durationMs,
			Cost:       tc.cost,
			TokensIn:   tc.tokensIn,
			TokensOut:  tc.tokensOut,
			Tools:      make([]*TurnTool, 0, len(tc.tools)),
		}

		// Add tools sorted by start time (already sorted during collection)
		for _, tool := range tc.tools {
			turn.Tools = append(turn.Tools, tool)
			result.Stats.TotalTools++
		}

		result.Turns = append(result.Turns, turn)
		result.Stats.TotalTurns++
		result.Stats.TotalCost += tc.cost
		result.Stats.TotalTokens += tc.tokensIn + tc.tokensOut
	}

	return result
}

// turnCollector accumulates data for a single turn during traversal.
type turnCollector struct {
	spanID     string
	durationMs int64
	cost       float64
	tokensIn   int64
	tokensOut  int64
	startTime  int64 // For sorting tools
	tools      []*TurnTool
}

// collectTurnsForGrouping recursively traverses spans to collect turn data.
func collectTurnsForGrouping(nodes []*SpanNode, turnsMap map[int]*turnCollector) {
	for _, node := range nodes {
		if node == nil || node.Span == nil {
			continue
		}

		// Check if this is a turn span
		if isTurnSpanForGrouping(node.Span) {
			turnNum := extractTurnNumber(node.Span)
			if turnNum > 0 {
				tc := &turnCollector{
					spanID:     node.Span.ID,
					durationMs: node.Span.DurationMs,
					cost:       node.Span.CostUSD,
					tokensIn:   node.Span.TokensIn,
					tokensOut:  node.Span.TokensOut,
					startTime:  node.Span.StartTime.UnixMilli(),
					tools:      make([]*TurnTool, 0),
				}

				// Collect tool children
				collectToolsFromChildren(node.Children, tc)

				turnsMap[turnNum] = tc
			}
		}

		// Recurse into children
		collectTurnsForGrouping(node.Children, turnsMap)
	}
}

// collectToolsFromChildren collects tool spans from a turn's children.
func collectToolsFromChildren(children []*SpanNode, tc *turnCollector) {
	for _, child := range children {
		if child == nil || child.Span == nil {
			continue
		}

		if isToolSpanForGrouping(child.Span) {
			tool := &TurnTool{
				ID:         child.Span.ID,
				Name:       child.Span.Name,
				ToolName:   extractToolName(child.Span),
				DurationMs: child.Span.DurationMs,
				Cost:       child.Span.CostUSD,
				Status:     string(child.Span.Status),
			}
			tc.tools = append(tc.tools, tool)
		}

		// Also check nested children (tools might have children too)
		collectToolsFromChildren(child.Children, tc)
	}
}

// isSessionOrExecutorSpan returns true if this is a session or executor span.
func isSessionOrExecutorSpan(span *Span) bool {
	name := span.Name
	return name == "claude.execute" ||
		name == "gemini.execute" ||
		name == "coordinator.task.execute" ||
		strings.HasPrefix(name, "exec.session") ||
		strings.HasPrefix(name, "session.")
}

// isTurnSpanForGrouping returns true if this is a turn span.
func isTurnSpanForGrouping(span *Span) bool {
	name := span.Name
	return strings.Contains(name, "turn") ||
		strings.HasPrefix(name, "exec.turn")
}

// isToolSpanForGrouping returns true if this is a tool span.
func isToolSpanForGrouping(span *Span) bool {
	name := span.Name
	return strings.HasPrefix(name, "tool.") ||
		strings.HasPrefix(name, "exec.tool") ||
		strings.Contains(name, "tool_use")
}

// extractTurnNumber extracts the turn number from a span.
func extractTurnNumber(span *Span) int {
	// Try to extract from span name (e.g., "exec.turn.3" or "turn.3")
	name := span.Name

	// Pattern: exec.turn.N or turn.N
	if strings.Contains(name, "turn") {
		parts := strings.Split(name, ".")
		for i, part := range parts {
			if part == "turn" && i+1 < len(parts) {
				if num, err := parseIntSafe(parts[i+1]); err == nil && num > 0 {
					return num
				}
			}
		}
	}

	// Try attributes
	if span.Attributes != nil {
		// Check turn.number attribute
		if turnNumAttr, ok := span.Attributes["turn.number"]; ok {
			if num, ok := turnNumAttr.(float64); ok {
				return int(num)
			}
			if num, ok := turnNumAttr.(int); ok {
				return num
			}
			if numStr, ok := turnNumAttr.(string); ok {
				if num, err := parseIntSafe(numStr); err == nil {
					return num
				}
			}
		}
	}

	return 0
}

// extractToolName extracts the tool name from a span.
func extractToolName(span *Span) string {
	name := span.Name

	// Pattern: tool.Read, exec.tool_use.Bash, etc.
	if strings.HasPrefix(name, "tool.") {
		return strings.TrimPrefix(name, "tool.")
	}
	if strings.Contains(name, "tool_use") {
		// exec.tool_use or claude_code.tool_use
		if span.Attributes != nil {
			if toolName, ok := span.Attributes["tool.name"].(string); ok {
				return toolName
			}
		}
		// Try to extract from name
		parts := strings.Split(name, ".")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	return name
}

// parseIntSafe safely parses an integer string without strconv.
func parseIntSafe(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %s", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

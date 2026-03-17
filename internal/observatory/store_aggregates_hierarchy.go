// Package observatory provides span hierarchy query operations.
package observatory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// GetSpanHierarchy returns a hierarchical tree of spans using parent_span_id relationships.
// Unlike GetExecTaskHierarchy which requires custom attributes, this works with standard OTEL parenting.
// Returns roots (coordinator/executor spans), session groups, and stats.
func (s *Store) GetSpanHierarchy(limit int) (*SpanHierarchyResult, error) {
	if limit <= 0 {
		limit = 100
	}

	// Step 1: Query root spans (those with no parent or executor roots)
	// We look for coordinator.task.execute, claude.execute, gemini.execute as roots
	rootQuery := `
		SELECT id, parent_span_id, name, start_time, duration_ms,
			   tokens_in, tokens_out, cache_read_tokens, cache_creation_tokens,
			   cost_usd, status, provider, attributes
		FROM spans
		WHERE (parent_span_id IS NULL OR parent_span_id = '')
		  AND name IN ('coordinator.task.execute', 'claude.execute', 'gemini.execute', 'ailang.exec')
		ORDER BY start_time DESC
		LIMIT ?
	`
	rootRows, err := s.db.Query(rootQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("query root spans: %w", err)
	}
	defer rootRows.Close()

	// Collect root span IDs
	rootIDs := make([]string, 0)
	rootNodes := make(map[string]*SpanHierarchyNode)

	for rootRows.Next() {
		node, err := scanSpanHierarchyNode(rootRows)
		if err != nil {
			continue
		}
		node.Depth = 0
		rootIDs = append(rootIDs, node.ID)
		rootNodes[node.ID] = node
	}

	if len(rootIDs) == 0 {
		// Return empty result if no roots found
		return &SpanHierarchyResult{
			Roots:    []*SpanHierarchyNode{},
			Sessions: make(map[string]int),
		}, nil
	}

	// Step 2: Query all children using recursive CTE
	// Build placeholder string for root IDs
	placeholders := make([]string, len(rootIDs))
	args := make([]interface{}, len(rootIDs)+1) // +1 for depth limit
	for i, id := range rootIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	maxDepth := 5
	args[len(rootIDs)] = maxDepth

	childQuery := `
		WITH RECURSIVE children AS (
			-- Base case: direct children of root spans
			SELECT id, parent_span_id, name, start_time, duration_ms,
				   tokens_in, tokens_out, cache_read_tokens, cache_creation_tokens,
				   cost_usd, status, provider, attributes,
				   1 as depth
			FROM spans
			WHERE parent_span_id IN (` + strings.Join(placeholders, ",") + `)

			UNION ALL

			-- Recursive case: children of children
			SELECT s.id, s.parent_span_id, s.name, s.start_time, s.duration_ms,
				   s.tokens_in, s.tokens_out, s.cache_read_tokens, s.cache_creation_tokens,
				   s.cost_usd, s.status, s.provider, s.attributes,
				   c.depth + 1
			FROM spans s
			JOIN children c ON s.parent_span_id = c.id
			WHERE c.depth < ?
		)
		SELECT id, parent_span_id, name, start_time, duration_ms,
			   tokens_in, tokens_out, cache_read_tokens, cache_creation_tokens,
			   cost_usd, status, provider, attributes, depth
		FROM children
		ORDER BY depth, start_time
	`

	childRows, err := s.db.Query(childQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query child spans: %w", err)
	}
	defer childRows.Close()

	// Build map of all nodes
	allNodes := make(map[string]*SpanHierarchyNode)
	for id, node := range rootNodes {
		allNodes[id] = node
	}

	// Sessions tracking
	sessions := make(map[string]int)

	// Scan children and add to map
	for childRows.Next() {
		var id, parentID, name, status, provider, attrsJSON string
		var startTime sql.NullTime
		var durationMs, tokensIn, tokensOut, cacheReadTokens, cacheCreationTokens sql.NullInt64
		var costUSD sql.NullFloat64
		var depth int

		if err := childRows.Scan(&id, &parentID, &name, &startTime, &durationMs,
			&tokensIn, &tokensOut, &cacheReadTokens, &cacheCreationTokens,
			&costUSD, &status, &provider, &attrsJSON, &depth); err != nil {
			continue
		}

		node := &SpanHierarchyNode{
			ID:       id,
			ParentID: parentID,
			Name:     name,
			Status:   SpanStatus(status),
			Provider: Provider(provider),
			Depth:    depth,
			Children: []*SpanHierarchyNode{},
		}
		if startTime.Valid {
			node.StartTime = startTime.Time
		}
		if durationMs.Valid {
			node.DurationMs = durationMs.Int64
		}
		if tokensIn.Valid {
			node.TokensIn = tokensIn.Int64
		}
		if tokensOut.Valid {
			node.TokensOut = tokensOut.Int64
		}
		if cacheReadTokens.Valid {
			node.CacheReadTokens = cacheReadTokens.Int64
		}
		if cacheCreationTokens.Valid {
			node.CacheCreationTokens = cacheCreationTokens.Int64
		}
		if costUSD.Valid {
			node.CostUSD = costUSD.Float64
		}

		// Parse attributes to extract session.id and turn.number
		node.parseAttributesForHierarchy(attrsJSON, sessions)
		node.NodeType = classifySpanNodeType(name)

		allNodes[id] = node
	}

	// Step 3: Build parent-child relationships
	for _, node := range allNodes {
		if node.ParentID != "" {
			if parent, ok := allNodes[node.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	// Sort children by start time
	for _, node := range allNodes {
		sortChildrenByStartTime(node.Children)
	}

	// Step 4: Build result with roots in order
	roots := make([]*SpanHierarchyNode, 0, len(rootIDs))
	for _, id := range rootIDs {
		if node, ok := rootNodes[id]; ok {
			roots = append(roots, node)
		}
	}

	// Step 5: Calculate stats
	stats := calculateHierarchyStats(roots, allNodes)

	return &SpanHierarchyResult{
		Roots:    roots,
		Sessions: sessions,
		Stats:    stats,
	}, nil
}

// scanSpanHierarchyNode scans a row into a SpanHierarchyNode
func scanSpanHierarchyNode(rows *sql.Rows) (*SpanHierarchyNode, error) {
	var id, name string
	var parentID, status, provider, attrsJSON sql.NullString
	var startTime sql.NullTime
	var durationMs, tokensIn, tokensOut, cacheReadTokens, cacheCreationTokens sql.NullInt64
	var costUSD sql.NullFloat64

	if err := rows.Scan(&id, &parentID, &name, &startTime, &durationMs,
		&tokensIn, &tokensOut, &cacheReadTokens, &cacheCreationTokens,
		&costUSD, &status, &provider, &attrsJSON); err != nil {
		return nil, err
	}

	node := &SpanHierarchyNode{
		ID:       id,
		ParentID: parentID.String,
		Name:     name,
		Status:   SpanStatus(status.String),
		Provider: Provider(provider.String),
		Children: []*SpanHierarchyNode{},
	}
	if startTime.Valid {
		node.StartTime = startTime.Time
	}
	if durationMs.Valid {
		node.DurationMs = durationMs.Int64
	}
	if tokensIn.Valid {
		node.TokensIn = tokensIn.Int64
	}
	if tokensOut.Valid {
		node.TokensOut = tokensOut.Int64
	}
	if cacheReadTokens.Valid {
		node.CacheReadTokens = cacheReadTokens.Int64
	}
	if cacheCreationTokens.Valid {
		node.CacheCreationTokens = cacheCreationTokens.Int64
	}
	if costUSD.Valid {
		node.CostUSD = costUSD.Float64
	}

	// Parse attributes
	sessions := make(map[string]int) // Dummy for parsing
	node.parseAttributesForHierarchy(attrsJSON.String, sessions)
	node.NodeType = classifySpanNodeType(name)

	return node, nil
}

// parseAttributesForHierarchy extracts session.id and turn.number from attributes JSON
func (n *SpanHierarchyNode) parseAttributesForHierarchy(attrsJSON string, sessions map[string]int) {
	if attrsJSON == "" || attrsJSON == "{}" {
		return
	}

	var attrs map[string]interface{}
	if err := json.Unmarshal([]byte(attrsJSON), &attrs); err != nil {
		return
	}

	// Extract session.id
	if sessionID, ok := attrs["session.id"].(string); ok && sessionID != "" {
		n.SessionID = sessionID
	}

	// Extract turn.number
	if turnNum, ok := attrs["turn.number"].(float64); ok {
		n.TurnNumber = int(turnNum)
		// Track session turn count
		if n.SessionID != "" {
			if current, ok := sessions[n.SessionID]; !ok || n.TurnNumber > current {
				sessions[n.SessionID] = n.TurnNumber
			}
		}
	}

	// Extract tool.name for tool spans
	if toolName, ok := attrs["tool.name"].(string); ok {
		n.ToolName = toolName
	}

	// Store selected useful attributes
	n.Attributes = make(map[string]interface{})
	for key, val := range attrs {
		// Keep only useful attributes for display
		switch key {
		case "task.id", "task.title", "task.iteration", "task.turns",
			"executor.name", "executor.model", "tool.input", "tool.output":
			n.Attributes[key] = val
		}
	}
}

// classifySpanNodeType determines the node type based on span name
func classifySpanNodeType(name string) SpanHierarchyNodeType {
	switch {
	case name == "coordinator.task.execute":
		return NodeTypeCoordinator
	case name == "claude.execute" || name == "gemini.execute" || name == "ailang.exec":
		return NodeTypeExecutor
	case name == "exec.turn":
		return NodeTypeTurn
	case strings.HasPrefix(name, "claude_code.tool.") || strings.HasPrefix(name, "exec.tool_use"):
		return NodeTypeTool
	default:
		return NodeTypeOther
	}
}

// sortChildrenByStartTime sorts children by start time
func sortChildrenByStartTime(children []*SpanHierarchyNode) {
	sort.Slice(children, func(i, j int) bool {
		return children[i].StartTime.Before(children[j].StartTime)
	})
}

// calculateHierarchyStats calculates aggregate stats for the hierarchy
func calculateHierarchyStats(roots []*SpanHierarchyNode, allNodes map[string]*SpanHierarchyNode) SpanHierarchyStats {
	stats := SpanHierarchyStats{
		TotalSpans: len(allNodes),
	}

	var minTime, maxTime time.Time
	maxDepth := 0

	for _, node := range allNodes {
		stats.TotalCost += node.CostUSD
		stats.TotalTokens.In += node.TokensIn
		stats.TotalTokens.Out += node.TokensOut
		stats.TotalTokens.CacheRead += node.CacheReadTokens
		stats.TotalTokens.CacheCreation += node.CacheCreationTokens

		if node.Depth > maxDepth {
			maxDepth = node.Depth
		}

		if !node.StartTime.IsZero() {
			if minTime.IsZero() || node.StartTime.Before(minTime) {
				minTime = node.StartTime
			}
			endTime := node.StartTime.Add(time.Duration(node.DurationMs) * time.Millisecond)
			if maxTime.IsZero() || endTime.After(maxTime) {
				maxTime = endTime
			}
		}
	}

	stats.TimeRange.Start = minTime
	stats.TimeRange.End = maxTime
	stats.MaxDepth = maxDepth

	return stats
}

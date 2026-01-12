// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ExecTaskNode represents a node in the exec task hierarchy
type ExecTaskNode struct {
	TaskID       string          `json:"task_id"`
	ParentTaskID string          `json:"parent_task_id"`
	Command      string          `json:"command"`   // exec, run, check, turn, tool_use
	Provider     string          `json:"provider"`  // for exec: claude, gemini, etc.
	Workspace    string          `json:"workspace"` // for exec
	FilePath     string          `json:"file_path"` // for run, check
	Status       string          `json:"status"`
	StartTime    *time.Time      `json:"start_time,omitempty"`
	DurationMs   int             `json:"duration_ms,omitempty"`
	Children     []*ExecTaskNode `json:"children,omitempty"`
	// Turn/tool specific fields
	TurnNumber  int    `json:"turn_number,omitempty"`  // for turn spans
	ToolName    string `json:"tool_name,omitempty"`    // for tool_use spans
	ToolInput   string `json:"tool_input,omitempty"`   // for tool_use spans
	ToolOutput  string `json:"tool_output,omitempty"`  // for tool_use spans
	DisplayName string `json:"display_name,omitempty"` // enriched name from session_tools (e.g., "Read: /path/file.go")
}

// MessageNode represents a message that triggered coordinator tasks
type MessageNode struct {
	MessageID   string          `json:"message_id"`
	Title       string          `json:"title"`
	FromAgent   string          `json:"from_agent"`
	ToInbox     string          `json:"to_inbox"`
	MessageType string          `json:"message_type"`
	Status      string          `json:"status"`
	CreatedAt   *time.Time      `json:"created_at,omitempty"`
	Execs       []*ExecTaskNode `json:"execs,omitempty"`
}

// ExecHierarchyWithMessages groups exec tasks by their triggering messages
type ExecHierarchyWithMessages struct {
	Messages []*MessageNode  `json:"messages,omitempty"` // Messages that triggered execs
	Orphan   []*ExecTaskNode `json:"orphan,omitempty"`   // Execs without a triggering message
	Count    int             `json:"count"`
}

// GetMetricsSummary returns global metrics.
func (s *Store) GetMetricsSummary() (*MetricsSummary, error) {
	summary := &MetricsSummary{}

	// Count workspaces
	s.db.QueryRow("SELECT COUNT(*) FROM workspaces").Scan(&summary.TotalWorkspaces)

	// Count tasks
	s.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&summary.TotalTasks)

	// Count spans
	s.db.QueryRow("SELECT COUNT(*) FROM spans").Scan(&summary.TotalSpans)

	// Count unique agents
	s.db.QueryRow("SELECT COUNT(DISTINCT agent_id) FROM agent_assignments").Scan(&summary.TotalAgents)

	// Sum tokens and cost
	s.db.QueryRow(`
		SELECT COALESCE(SUM(tokens_in), 0), COALESCE(SUM(tokens_out), 0), COALESCE(SUM(cost_usd), 0)
		FROM spans
	`).Scan(&summary.TotalTokensIn, &summary.TotalTokensOut, &summary.TotalCostUSD)

	// Calculate success rate
	var completed, failed int
	s.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'completed'").Scan(&completed)
	s.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'failed'").Scan(&failed)
	if completed+failed > 0 {
		summary.SuccessRate = float64(completed) / float64(completed+failed) * 100
	}

	return summary, nil
}

// GetProviderComparison returns comparison metrics across providers.
func (s *Store) GetProviderComparison() ([]*ProviderComparison, error) {
	rows, err := s.db.Query(`
		SELECT provider, total_executions, total_tokens_in, total_tokens_out,
		       total_cost, avg_duration_ms, success_rate
		FROM provider_comparison
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comparisons []*ProviderComparison
	for rows.Next() {
		pc := &ProviderComparison{}
		if err := rows.Scan(&pc.Provider, &pc.TotalExecutions, &pc.TotalTokensIn,
			&pc.TotalTokensOut, &pc.TotalCost, &pc.AvgDurationMs, &pc.SuccessRate); err != nil {
			return nil, err
		}
		comparisons = append(comparisons, pc)
	}
	return comparisons, rows.Err()
}

// GetTaskTimeline returns timeline data for a task.
func (s *Store) GetTaskTimeline(taskID string) ([]*TaskTimeline, error) {
	rows, err := s.db.Query(`
		SELECT task_id, title, status, span_id, span_name, start_time, end_time,
		       duration_ms, span_status, tokens_in, tokens_out, cost_usd, provider
		FROM task_timeline WHERE task_id = ?
		ORDER BY start_time ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var timeline []*TaskTimeline
	for rows.Next() {
		tl := &TaskTimeline{}
		var spanID, spanName sql.NullString
		var startTime, endTime sql.NullTime
		var spanStatus, provider sql.NullString
		if err := rows.Scan(&tl.TaskID, &tl.Title, &tl.Status, &spanID, &spanName,
			&startTime, &endTime, &tl.DurationMs, &spanStatus,
			&tl.TokensIn, &tl.TokensOut, &tl.CostUSD, &provider); err != nil {
			return nil, err
		}
		if spanID.Valid {
			tl.SpanID = spanID.String
		}
		if spanName.Valid {
			tl.SpanName = spanName.String
		}
		if startTime.Valid {
			tl.StartTime = &startTime.Time
		}
		if endTime.Valid {
			tl.EndTime = &endTime.Time
		}
		if spanStatus.Valid {
			tl.SpanStatus = SpanStatus(spanStatus.String)
		}
		if provider.Valid {
			tl.Provider = Provider(provider.String)
		}
		timeline = append(timeline, tl)
	}
	return timeline, rows.Err()
}

// GetExecTaskHierarchy returns the hierarchy of ailang commands (exec, run, check) from span attributes
// including turn and tool_use child spans for complete hierarchy visualization
func (s *Store) GetExecTaskHierarchy(limit int) ([]*ExecTaskNode, error) {
	if limit <= 0 {
		limit = 100
	}

	// Query ailang command spans (exec, run, check) and extract hierarchy attributes
	// Note: attributes use dot-notation keys like "exec.task_id" not nested "exec"."task_id"
	// Spans: ailang.exec, ailang.check, "ailang run: <filename>"
	rows, err := s.db.Query(`
		SELECT
			id,
			name,
			JSON_EXTRACT(attributes, '$."exec.task_id"') as task_id,
			JSON_EXTRACT(attributes, '$."exec.parent_task_id"') as parent_task_id,
			JSON_EXTRACT(attributes, '$."exec.provider"') as provider,
			JSON_EXTRACT(attributes, '$."exec.workspace"') as workspace,
			JSON_EXTRACT(attributes, '$."file.path"') as file_path,
			status,
			start_time,
			duration_ms
		FROM spans
		WHERE name = 'ailang.exec'
		   OR name = 'ailang.check'
		   OR name LIKE 'ailang run:%'
		ORDER BY start_time DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build flat list of nodes
	nodeMap := make(map[string]*ExecTaskNode)
	var nodes []*ExecTaskNode

	for rows.Next() {
		var spanID, spanName string
		var taskID, parentTaskID, provider, workspace, filePath, status sql.NullString
		var startTime sql.NullTime
		var durationMs sql.NullInt64

		if err := rows.Scan(&spanID, &spanName, &taskID, &parentTaskID, &provider, &workspace, &filePath, &status, &startTime, &durationMs); err != nil {
			return nil, err
		}

		// Determine command type from span name
		command := "exec"
		if spanName == "ailang.check" {
			command = "check"
		} else if strings.HasPrefix(spanName, "ailang run:") {
			command = "run"
		}

		node := &ExecTaskNode{
			TaskID:   taskID.String,
			Command:  command,
			Status:   status.String,
			Children: []*ExecTaskNode{},
		}
		if parentTaskID.Valid {
			node.ParentTaskID = parentTaskID.String
		}
		if provider.Valid {
			node.Provider = provider.String
		}
		if workspace.Valid {
			node.Workspace = workspace.String
		}
		if filePath.Valid {
			node.FilePath = filePath.String
		}
		if startTime.Valid {
			node.StartTime = &startTime.Time
		}
		if durationMs.Valid {
			node.DurationMs = int(durationMs.Int64)
		}

		// Only add if we have a task_id
		if node.TaskID != "" {
			nodeMap[node.TaskID] = node
			nodes = append(nodes, node)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Query turn and tool_use spans and attach to their parent exec nodes
	turnToolRows, err := s.db.Query(`
		SELECT
			id,
			name,
			JSON_EXTRACT(attributes, '$."exec.task_id"') as parent_task_id,
			JSON_EXTRACT(attributes, '$."turn.number"') as turn_number,
			JSON_EXTRACT(attributes, '$."tool.name"') as tool_name,
			JSON_EXTRACT(attributes, '$."tool.input"') as tool_input,
			JSON_EXTRACT(attributes, '$."tool.output"') as tool_output,
			status,
			start_time,
			duration_ms
		FROM spans
		WHERE name = 'exec.turn' OR name = 'exec.tool_use'
		ORDER BY start_time ASC
	`)
	if err != nil {
		return nil, err
	}
	defer turnToolRows.Close()

	// Attach turn/tool spans to their parent exec nodes
	for turnToolRows.Next() {
		var spanID, spanName string
		var parentTaskID, toolName, toolInput, toolOutput, status sql.NullString
		var turnNumber sql.NullInt64
		var startTime sql.NullTime
		var durationMs sql.NullInt64

		if err := turnToolRows.Scan(&spanID, &spanName, &parentTaskID, &turnNumber, &toolName, &toolInput, &toolOutput, &status, &startTime, &durationMs); err != nil {
			return nil, err
		}

		// Skip if no parent task ID
		if !parentTaskID.Valid || parentTaskID.String == "" {
			continue
		}

		// Find parent exec node
		parent, ok := nodeMap[parentTaskID.String]
		if !ok {
			continue // Parent exec not in result set
		}

		// Create child node
		command := "turn"
		if spanName == "exec.tool_use" {
			command = "tool_use"
		}

		child := &ExecTaskNode{
			TaskID:       spanID, // Use span ID for uniqueness
			ParentTaskID: parentTaskID.String,
			Command:      command,
			Status:       status.String,
			Children:     []*ExecTaskNode{},
		}
		if turnNumber.Valid {
			child.TurnNumber = int(turnNumber.Int64)
		}
		if toolName.Valid {
			child.ToolName = toolName.String
		}
		if toolInput.Valid {
			child.ToolInput = toolInput.String
		}
		if toolOutput.Valid {
			child.ToolOutput = toolOutput.String
		}
		if startTime.Valid {
			child.StartTime = &startTime.Time
		}
		if durationMs.Valid {
			child.DurationMs = int(durationMs.Int64)
		}

		parent.Children = append(parent.Children, child)
	}

	if err := turnToolRows.Err(); err != nil {
		return nil, err
	}

	// Build tree structure for exec/run/check nodes
	var roots []*ExecTaskNode
	for _, node := range nodes {
		if node.ParentTaskID == "" || node.ParentTaskID == "root" {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[node.ParentTaskID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// Parent not in result set, treat as root
			roots = append(roots, node)
		}
	}

	return roots, nil
}

// GetExecTaskHierarchyWithMessages returns the exec hierarchy grouped by triggering messages.
// This provides a 4-level view: Messages -> Execs -> Turns -> Tool Uses
func (s *Store) GetExecTaskHierarchyWithMessages(limit int) (*ExecHierarchyWithMessages, error) {
	if limit <= 0 {
		limit = 100
	}

	// First, get the regular exec hierarchy
	execNodes, err := s.GetExecTaskHierarchy(limit)
	if err != nil {
		return nil, fmt.Errorf("get exec hierarchy: %w", err)
	}

	if len(execNodes) == 0 {
		return &ExecHierarchyWithMessages{Count: 0}, nil
	}

	// Collect all task IDs from root exec nodes
	taskIDs := make([]string, 0, len(execNodes))
	taskToExec := make(map[string]*ExecTaskNode)
	for _, node := range execNodes {
		if node.TaskID != "" {
			taskIDs = append(taskIDs, node.TaskID)
			taskToExec[node.TaskID] = node
		}
	}

	if len(taskIDs) == 0 {
		// No task IDs found, return all as orphan
		return &ExecHierarchyWithMessages{
			Orphan: execNodes,
			Count:  len(execNodes),
		}, nil
	}

	// Query tasks to get source_ref (message_id) for each
	// Build placeholder string for IN clause
	placeholders := make([]string, len(taskIDs))
	args := make([]interface{}, len(taskIDs))
	for i, id := range taskIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	taskQuery := `
		SELECT id, source_type, source_ref
		FROM tasks
		WHERE id IN (` + strings.Join(placeholders, ",") + `)
	`
	taskRows, err := s.db.Query(taskQuery, args...)
	if err != nil {
		// If query fails, return hierarchy without message grouping
		return &ExecHierarchyWithMessages{
			Orphan: execNodes,
			Count:  len(execNodes),
		}, nil
	}
	defer taskRows.Close()

	// Map task_id -> message_id (source_ref)
	taskToMessage := make(map[string]string)
	for taskRows.Next() {
		var taskID string
		var sourceType, sourceRef sql.NullString
		if err := taskRows.Scan(&taskID, &sourceType, &sourceRef); err != nil {
			continue
		}
		// Only include if source_type is 'message' and source_ref is set
		if sourceType.Valid && sourceType.String == "message" && sourceRef.Valid && sourceRef.String != "" {
			taskToMessage[taskID] = sourceRef.String
		}
	}

	// Group execs by message
	messageToExecs := make(map[string][]*ExecTaskNode)
	var orphanExecs []*ExecTaskNode
	for _, node := range execNodes {
		if msgID, ok := taskToMessage[node.TaskID]; ok {
			messageToExecs[msgID] = append(messageToExecs[msgID], node)
		} else {
			orphanExecs = append(orphanExecs, node)
		}
	}

	// Fetch message details for all unique message IDs
	result := &ExecHierarchyWithMessages{
		Orphan: orphanExecs,
	}

	if len(messageToExecs) > 0 {
		// Build message ID list
		msgPlaceholders := make([]string, 0, len(messageToExecs))
		msgArgs := make([]interface{}, 0, len(messageToExecs))
		for msgID := range messageToExecs {
			msgPlaceholders = append(msgPlaceholders, "?")
			msgArgs = append(msgArgs, msgID)
		}

		// Query messages
		msgQuery := `
			SELECT id, inbox, from_agent, title, message_type, status, created_at
			FROM messages
			WHERE id IN (` + strings.Join(msgPlaceholders, ",") + `)
		`
		msgRows, err := s.db.Query(msgQuery, msgArgs...)
		if err == nil {
			defer msgRows.Close()

			msgDetails := make(map[string]*MessageNode)
			for msgRows.Next() {
				var msgID, inbox, fromAgent, title, msgType, status string
				var createdAt sql.NullTime
				if err := msgRows.Scan(&msgID, &inbox, &fromAgent, &title, &msgType, &status, &createdAt); err != nil {
					continue
				}
				node := &MessageNode{
					MessageID:   msgID,
					Title:       title,
					FromAgent:   fromAgent,
					ToInbox:     inbox,
					MessageType: msgType,
					Status:      status,
				}
				if createdAt.Valid {
					node.CreatedAt = &createdAt.Time
				}
				msgDetails[msgID] = node
			}

			// Build message nodes with their execs
			for msgID, execs := range messageToExecs {
				var msgNode *MessageNode
				if details, ok := msgDetails[msgID]; ok {
					msgNode = details
				} else {
					// Message not found in DB, create minimal node
					msgNode = &MessageNode{
						MessageID: msgID,
						Title:     msgID,
						Status:    "unknown",
					}
				}
				msgNode.Execs = execs
				result.Messages = append(result.Messages, msgNode)
			}
		} else {
			// Couldn't fetch messages, put execs as orphans
			for _, execs := range messageToExecs {
				result.Orphan = append(result.Orphan, execs...)
			}
		}
	}

	// Count total
	result.Count = len(result.Messages) + len(result.Orphan)
	return result, nil
}

// RecalculateTaskAggregates recalculates all aggregate metrics for a task from its spans.
// Use this for backfill operations or to fix inconsistent aggregates.
func (s *Store) RecalculateTaskAggregates(taskID string) error {
	_, err := s.db.Exec(`
		UPDATE tasks SET
			span_count = COALESCE((SELECT COUNT(*) FROM spans WHERE task_id = ?), 0),
			total_duration_ms = COALESCE((SELECT SUM(duration_ms) FROM spans WHERE task_id = ?), 0),
			total_tokens_in = COALESCE((SELECT SUM(tokens_in) FROM spans WHERE task_id = ?), 0),
			total_tokens_out = COALESCE((SELECT SUM(tokens_out) FROM spans WHERE task_id = ?), 0),
			total_cost_usd = COALESCE((SELECT SUM(cost_usd) FROM spans WHERE task_id = ?), 0),
			error_count = COALESCE((SELECT COUNT(*) FROM spans WHERE task_id = ? AND status = 'error'), 0)
		WHERE id = ?
	`, taskID, taskID, taskID, taskID, taskID, taskID, taskID)

	if err != nil {
		return fmt.Errorf("recalculate task aggregates: %w", err)
	}
	return nil
}

// RecalculateAgentAssignmentAggregates recalculates all aggregate metrics for an agent assignment.
// Use this for backfill operations or to fix inconsistent aggregates.
func (s *Store) RecalculateAgentAssignmentAggregates(assignmentID string) error {
	_, err := s.db.Exec(`
		UPDATE agent_assignments SET
			duration_ms = COALESCE((SELECT SUM(duration_ms) FROM spans WHERE agent_assignment_id = ?), 0),
			tokens_in = COALESCE((SELECT SUM(tokens_in) FROM spans WHERE agent_assignment_id = ?), 0),
			tokens_out = COALESCE((SELECT SUM(tokens_out) FROM spans WHERE agent_assignment_id = ?), 0),
			cost_usd = COALESCE((SELECT SUM(cost_usd) FROM spans WHERE agent_assignment_id = ?), 0),
			tool_calls = COALESCE((SELECT COUNT(*) FROM spans WHERE agent_assignment_id = ? AND name LIKE '%tool.%'), 0)
		WHERE id = ?
	`, assignmentID, assignmentID, assignmentID, assignmentID, assignmentID, assignmentID)

	if err != nil {
		return fmt.Errorf("recalculate agent assignment aggregates: %w", err)
	}
	return nil
}

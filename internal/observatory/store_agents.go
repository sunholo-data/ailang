// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"database/sql"
)

// CreateAgentAssignment inserts a new agent assignment.
func (s *Store) CreateAgentAssignment(a *AgentAssignment) error {
	// Convert empty strings to NULL for foreign key columns
	var parentID interface{}
	if a.ParentAssignmentID != "" {
		parentID = a.ParentAssignmentID
	}

	_, err := s.db.Exec(`
		INSERT INTO agent_assignments (id, task_id, agent_id, provider, status,
		                               assigned_at, started_at, completed_at,
		                               parent_assignment_id, duration_ms,
		                               tokens_in, tokens_out, cost_usd, tool_calls, turns)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, a.ID, a.TaskID, a.AgentID, a.Provider, a.Status,
		a.AssignedAt, a.StartedAt, a.CompletedAt,
		parentID, a.DurationMs,
		a.TokensIn, a.TokensOut, a.CostUSD, a.ToolCalls, a.Turns)
	return err
}

// GetAgentAssignment retrieves an agent assignment by ID.
func (s *Store) GetAgentAssignment(id string) (*AgentAssignment, error) {
	a := &AgentAssignment{}
	var parentID sql.NullString
	var startedAt, completedAt sql.NullTime
	err := s.db.QueryRow(`
		SELECT id, task_id, agent_id, provider, status,
		       assigned_at, started_at, completed_at,
		       parent_assignment_id, duration_ms,
		       tokens_in, tokens_out, cost_usd, tool_calls, turns
		FROM agent_assignments WHERE id = ?
	`, id).Scan(&a.ID, &a.TaskID, &a.AgentID, &a.Provider, &a.Status,
		&a.AssignedAt, &startedAt, &completedAt,
		&parentID, &a.DurationMs,
		&a.TokensIn, &a.TokensOut, &a.CostUSD, &a.ToolCalls, &a.Turns)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		a.ParentAssignmentID = parentID.String
	}
	if startedAt.Valid {
		a.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		a.CompletedAt = &completedAt.Time
	}
	return a, nil
}

// ListAgentAssignments returns assignments for a task.
func (s *Store) ListAgentAssignments(taskID string) ([]*AgentAssignment, error) {
	rows, err := s.db.Query(`
		SELECT id, task_id, agent_id, provider, status,
		       assigned_at, started_at, completed_at,
		       parent_assignment_id, duration_ms,
		       tokens_in, tokens_out, cost_usd, tool_calls, turns
		FROM agent_assignments WHERE task_id = ?
		ORDER BY assigned_at ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []*AgentAssignment
	for rows.Next() {
		a := &AgentAssignment{}
		var parentID sql.NullString
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(&a.ID, &a.TaskID, &a.AgentID, &a.Provider, &a.Status,
			&a.AssignedAt, &startedAt, &completedAt,
			&parentID, &a.DurationMs,
			&a.TokensIn, &a.TokensOut, &a.CostUSD, &a.ToolCalls, &a.Turns); err != nil {
			return nil, err
		}
		if parentID.Valid {
			a.ParentAssignmentID = parentID.String
		}
		if startedAt.Valid {
			a.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			a.CompletedAt = &completedAt.Time
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}

// UpdateAgentAssignment updates an existing assignment.
func (s *Store) UpdateAgentAssignment(a *AgentAssignment) error {
	_, err := s.db.Exec(`
		UPDATE agent_assignments SET status = ?, started_at = ?, completed_at = ?,
		                             duration_ms = ?, tokens_in = ?, tokens_out = ?,
		                             cost_usd = ?, tool_calls = ?, turns = ?
		WHERE id = ?
	`, a.Status, a.StartedAt, a.CompletedAt,
		a.DurationMs, a.TokensIn, a.TokensOut,
		a.CostUSD, a.ToolCalls, a.Turns, a.ID)
	return err
}

// DeleteAgentAssignment removes an assignment by ID.
func (s *Store) DeleteAgentAssignment(id string) error {
	_, err := s.db.Exec("DELETE FROM agent_assignments WHERE id = ?", id)
	return err
}

// GetAgentStats returns aggregated stats for an agent.
func (s *Store) GetAgentStats(agentID string) (*AgentStats, error) {
	stats := &AgentStats{}
	err := s.db.QueryRow(`
		SELECT agent_id, provider, execution_count, total_duration_ms,
		       avg_duration_ms, total_tokens_in, total_tokens_out,
		       total_cost, total_tool_calls, success_rate
		FROM agent_stats WHERE agent_id = ?
	`, agentID).Scan(
		&stats.AgentID, &stats.Provider, &stats.ExecutionCount, &stats.TotalDurationMs,
		&stats.AvgDurationMs, &stats.TotalTokensIn, &stats.TotalTokensOut,
		&stats.TotalCost, &stats.TotalToolCalls, &stats.SuccessRate,
	)
	return stats, err
}

package messaging

import (
	"database/sql"
	"fmt"
	"time"
)

// ApprovalHistoryEntry represents a single audit entry for an approval action
type ApprovalHistoryEntry struct {
	ID              string   `json:"id"`
	ApprovalID      string   `json:"approval_id"`
	ThreadID        string   `json:"thread_id"`
	AgentID         string   `json:"agent_id"`
	Action          string   `json:"action"` // created, approved, rejected, expired
	Actor           string   `json:"actor"`
	Proposal        string   `json:"proposal,omitempty"`
	Impact          string   `json:"impact,omitempty"`
	EstimatedCost   *float64 `json:"estimated_cost,omitempty"`
	CapabilityToken string   `json:"capability_token,omitempty"`
	CreatedAt       int64    `json:"created_at"`
}

// InstanceHistoryEntry represents lifecycle data for an agent instance
type InstanceHistoryEntry struct {
	ID            string `json:"id"`
	AgentID       string `json:"agent_id"`
	InstanceID    string `json:"instance_id"`
	StartedAt     int64  `json:"started_at"`
	EndedAt       *int64 `json:"ended_at,omitempty"`
	ExitCode      *int   `json:"exit_code,omitempty"`
	TotalTokens   int    `json:"total_tokens"`
	TotalCostCent int    `json:"total_cost_cents"`
	ThreadCount   int    `json:"thread_count"`
}

// RecordApprovalHistory adds an audit entry for an approval action
func (s *Store) RecordApprovalHistory(approvalID, threadID, agentID, action, actor, proposal, impact string, estimatedCost *float64, capabilityToken string) error {
	id := generateID("ahist")
	now := time.Now().UnixMilli()

	_, err := s.db.Exec(`
		INSERT INTO approval_history (
			id, approval_id, thread_id, agent_id, action, actor,
			proposal, impact, estimated_cost, capability_token, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, approvalID, threadID, agentID, action, actor, proposal, impact, estimatedCost, capabilityToken, now)

	if err != nil {
		return fmt.Errorf("failed to record approval history: %w", err)
	}
	return nil
}

// GetApprovalHistory returns approval history entries, optionally filtered by thread
func (s *Store) GetApprovalHistory(threadID string, limit int) ([]ApprovalHistoryEntry, error) {
	var rows *sql.Rows
	var err error

	if threadID != "" {
		rows, err = s.db.Query(`
			SELECT id, approval_id, thread_id, agent_id, action, actor,
				   proposal, impact, estimated_cost, capability_token, created_at
			FROM approval_history
			WHERE thread_id = ?
			ORDER BY created_at DESC
			LIMIT ?
		`, threadID, limit)
	} else {
		rows, err = s.db.Query(`
			SELECT id, approval_id, thread_id, agent_id, action, actor,
				   proposal, impact, estimated_cost, capability_token, created_at
			FROM approval_history
			ORDER BY created_at DESC
			LIMIT ?
		`, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query approval history: %w", err)
	}
	defer rows.Close()

	var entries []ApprovalHistoryEntry
	for rows.Next() {
		var e ApprovalHistoryEntry
		var proposal, impact, capToken sql.NullString
		var cost sql.NullFloat64

		err := rows.Scan(&e.ID, &e.ApprovalID, &e.ThreadID, &e.AgentID, &e.Action, &e.Actor,
			&proposal, &impact, &cost, &capToken, &e.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan approval history row: %w", err)
		}

		if proposal.Valid {
			e.Proposal = proposal.String
		}
		if impact.Valid {
			e.Impact = impact.String
		}
		if cost.Valid {
			e.EstimatedCost = &cost.Float64
		}
		if capToken.Valid {
			e.CapabilityToken = capToken.String
		}

		entries = append(entries, e)
	}

	return entries, nil
}

// RecordInstanceStart records the start of an agent instance
func (s *Store) RecordInstanceStart(agentID, instanceID string) error {
	id := generateID("inst")
	now := time.Now().UnixMilli()

	_, err := s.db.Exec(`
		INSERT INTO instance_history (
			id, agent_id, instance_id, started_at, total_tokens, total_cost_cents, thread_count
		) VALUES (?, ?, ?, ?, 0, 0, 0)
	`, id, agentID, instanceID, now)

	if err != nil {
		return fmt.Errorf("failed to record instance start: %w", err)
	}
	return nil
}

// RecordInstanceEnd records the end of an agent instance with final stats
func (s *Store) RecordInstanceEnd(instanceID string, exitCode int, totalTokens, totalCostCents, threadCount int) error {
	now := time.Now().UnixMilli()

	result, err := s.db.Exec(`
		UPDATE instance_history
		SET ended_at = ?, exit_code = ?, total_tokens = ?, total_cost_cents = ?, thread_count = ?
		WHERE instance_id = ? AND ended_at IS NULL
	`, now, exitCode, totalTokens, totalCostCents, threadCount, instanceID)

	if err != nil {
		return fmt.Errorf("failed to record instance end: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		// No matching instance found - this might be an instance not tracked
		return nil
	}

	return nil
}

// GetInstanceHistory returns instance history entries, optionally filtered by agent
func (s *Store) GetInstanceHistory(agentID string, limit int) ([]InstanceHistoryEntry, error) {
	var rows *sql.Rows
	var err error

	if agentID != "" {
		rows, err = s.db.Query(`
			SELECT id, agent_id, instance_id, started_at, ended_at, exit_code,
				   total_tokens, total_cost_cents, thread_count
			FROM instance_history
			WHERE agent_id = ?
			ORDER BY started_at DESC
			LIMIT ?
		`, agentID, limit)
	} else {
		rows, err = s.db.Query(`
			SELECT id, agent_id, instance_id, started_at, ended_at, exit_code,
				   total_tokens, total_cost_cents, thread_count
			FROM instance_history
			ORDER BY started_at DESC
			LIMIT ?
		`, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query instance history: %w", err)
	}
	defer rows.Close()

	var entries []InstanceHistoryEntry
	for rows.Next() {
		var e InstanceHistoryEntry
		var endedAt sql.NullInt64
		var exitCode sql.NullInt64

		err := rows.Scan(&e.ID, &e.AgentID, &e.InstanceID, &e.StartedAt, &endedAt, &exitCode,
			&e.TotalTokens, &e.TotalCostCent, &e.ThreadCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan instance history row: %w", err)
		}

		if endedAt.Valid {
			v := endedAt.Int64
			e.EndedAt = &v
		}
		if exitCode.Valid {
			v := int(exitCode.Int64)
			e.ExitCode = &v
		}

		entries = append(entries, e)
	}

	return entries, nil
}

// CleanupOldHistory removes history entries older than the retention period
func (s *Store) CleanupOldHistory(retentionDays int) (int64, int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays).UnixMilli()

	// Clean approval history
	result1, err := s.db.Exec(`DELETE FROM approval_history WHERE created_at < ?`, cutoff)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to cleanup approval history: %w", err)
	}
	approvalDeleted, _ := result1.RowsAffected()

	// Clean instance history
	result2, err := s.db.Exec(`DELETE FROM instance_history WHERE started_at < ?`, cutoff)
	if err != nil {
		return approvalDeleted, 0, fmt.Errorf("failed to cleanup instance history: %w", err)
	}
	instanceDeleted, _ := result2.RowsAffected()

	return approvalDeleted, instanceDeleted, nil
}

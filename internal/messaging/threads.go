package messaging

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Thread represents a conversation thread
type Thread struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	CreatedAt     time.Time `json:"created_at"`
	CreatedByType string    `json:"created_by_type"`
	CreatedByID   string    `json:"created_by_id"`
	Status        string    `json:"status"`
	ContextJSON   string    `json:"context_json,omitempty"`
	TargetAgent   string    `json:"target_agent,omitempty"` // Which agent this conversation is with
	Workspace     string    `json:"workspace,omitempty"`    // Working directory for this thread
	LastSeq       int       `json:"last_seq"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ThreadContext represents the parsed context_json for a thread
type ThreadContext struct {
	TargetAgent string `json:"target_agent,omitempty"`
	Workspace   string `json:"workspace,omitempty"`
}

// CreateThread creates a new thread in the database.
// targetAgent specifies which agent this conversation is with (optional).
func (s *Store) CreateThread(title, createdByType, createdByID, targetAgent string) (*Thread, error) {
	now := time.Now()
	threadID := fmt.Sprintf("thread_%d_%s", now.UnixMilli(), generateRandomID(8))

	// Store target_agent in context_json
	var contextJSON *string
	if targetAgent != "" {
		ctx := fmt.Sprintf(`{"target_agent":"%s"}`, targetAgent)
		contextJSON = &ctx
	}

	_, err := s.db.Exec(`
		INSERT INTO threads (id, title, created_at, created_by_type, created_by_id, status, context_json, last_seq, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, threadID, title, now.UnixMilli(), createdByType, createdByID, "active", contextJSON, 0, now.UnixMilli())

	if err != nil {
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}

	return &Thread{
		ID:            threadID,
		Title:         title,
		CreatedAt:     now,
		CreatedByType: createdByType,
		CreatedByID:   createdByID,
		Status:        "active",
		TargetAgent:   targetAgent,
		LastSeq:       0,
		UpdatedAt:     now,
	}, nil
}

// GetThread retrieves a thread by ID.
func (s *Store) GetThread(threadID string) (*Thread, error) {
	var thread Thread
	var createdAtMs, updatedAtMs int64
	var contextJSON sql.NullString

	err := s.db.QueryRow(`
		SELECT id, title, created_at, created_by_type, created_by_id, status, context_json, last_seq, updated_at
		FROM threads
		WHERE id = ?
	`, threadID).Scan(
		&thread.ID, &thread.Title, &createdAtMs, &thread.CreatedByType, &thread.CreatedByID,
		&thread.Status, &contextJSON, &thread.LastSeq, &updatedAtMs,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("thread not found: %s", threadID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	thread.CreatedAt = time.UnixMilli(createdAtMs)
	thread.UpdatedAt = time.UnixMilli(updatedAtMs)
	if contextJSON.Valid {
		thread.ContextJSON = contextJSON.String
		ctx := parseThreadContext(contextJSON.String)
		thread.TargetAgent = ctx.TargetAgent
		thread.Workspace = ctx.Workspace
	}

	return &thread, nil
}

// parseThreadContext extracts thread context from context_json
func parseThreadContext(contextJSON string) ThreadContext {
	if contextJSON == "" {
		return ThreadContext{}
	}
	var ctx ThreadContext
	if err := json.Unmarshal([]byte(contextJSON), &ctx); err != nil {
		return ThreadContext{}
	}
	return ctx
}

// SetThreadWorkspace updates the workspace path for a thread.
// This persists the workspace so all messages in the thread use the same working directory.
func (s *Store) SetThreadWorkspace(threadID, workspace string) error {
	// Get existing context
	thread, err := s.GetThread(threadID)
	if err != nil {
		return err
	}

	// Parse existing context or create new one
	ctx := parseThreadContext(thread.ContextJSON)
	ctx.Workspace = workspace

	// Serialize back to JSON
	contextBytes, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	// Update thread
	_, err = s.db.Exec(`
		UPDATE threads SET context_json = ?, updated_at = ? WHERE id = ?
	`, string(contextBytes), time.Now().UnixMilli(), threadID)

	if err != nil {
		return fmt.Errorf("failed to update thread workspace: %w", err)
	}

	return nil
}

// GetThreadWorkspace returns the workspace path for a thread.
func (s *Store) GetThreadWorkspace(threadID string) (string, error) {
	thread, err := s.GetThread(threadID)
	if err != nil {
		return "", err
	}
	return thread.Workspace, nil
}

// UpdateThreadTitle updates the title of a thread.
func (s *Store) UpdateThreadTitle(threadID, title string) error {
	result, err := s.db.Exec(`
		UPDATE threads SET title = ?, updated_at = ? WHERE id = ?
	`, title, time.Now().UnixMilli(), threadID)

	if err != nil {
		return fmt.Errorf("failed to update thread title: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("thread not found: %s", threadID)
	}

	return nil
}

// DeleteThread deletes a thread and all its messages.
func (s *Store) DeleteThread(threadID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Delete all messages in the thread
	_, err = tx.Exec(`DELETE FROM messages WHERE thread_id = ?`, threadID)
	if err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	// Delete all subscriptions to the thread
	_, err = tx.Exec(`DELETE FROM subscriptions WHERE thread_id = ?`, threadID)
	if err != nil {
		return fmt.Errorf("failed to delete subscriptions: %w", err)
	}

	// Delete all approvals for the thread
	_, err = tx.Exec(`DELETE FROM approvals WHERE thread_id = ?`, threadID)
	if err != nil {
		return fmt.Errorf("failed to delete approvals: %w", err)
	}

	// Delete the thread itself
	result, err := tx.Exec(`DELETE FROM threads WHERE id = ?`, threadID)
	if err != nil {
		return fmt.Errorf("failed to delete thread: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("thread not found: %s", threadID)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetThreadsByStatus returns all threads matching a status filter
// If status is empty, returns all threads
func (s *Store) GetThreadsByStatus(status string, limit int) ([]Thread, error) {
	var query string
	var args []interface{}

	if status == "" {
		query = `
			SELECT id, title, created_at, created_by_type, created_by_id, status, context_json, last_seq, updated_at
			FROM threads
			ORDER BY updated_at DESC
			LIMIT ?
		`
		args = []interface{}{limit}
	} else {
		query = `
			SELECT id, title, created_at, created_by_type, created_by_id, status, context_json, last_seq, updated_at
			FROM threads
			WHERE status = ?
			ORDER BY updated_at DESC
			LIMIT ?
		`
		args = []interface{}{status, limit}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query threads: %w", err)
	}
	defer rows.Close()

	// Initialize as empty slice (not nil) so JSON marshals to [] instead of null
	threads := []Thread{}
	for rows.Next() {
		var thread Thread
		var createdAtMs, updatedAtMs int64
		var contextJSON sql.NullString

		err := rows.Scan(
			&thread.ID, &thread.Title, &createdAtMs, &thread.CreatedByType, &thread.CreatedByID,
			&thread.Status, &contextJSON, &thread.LastSeq, &updatedAtMs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan thread: %w", err)
		}

		thread.CreatedAt = time.UnixMilli(createdAtMs)
		thread.UpdatedAt = time.UnixMilli(updatedAtMs)
		if contextJSON.Valid {
			thread.ContextJSON = contextJSON.String
			ctx := parseThreadContext(contextJSON.String)
			thread.TargetAgent = ctx.TargetAgent
			thread.Workspace = ctx.Workspace
		}

		threads = append(threads, thread)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating threads: %w", err)
	}

	return threads, nil
}

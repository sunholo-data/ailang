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
	return s.CreateThreadWithWorkspace(title, createdByType, createdByID, targetAgent, "")
}

// CreateThreadWithWorkspace creates a new thread with an optional workspace context.
// workspace is the source project/directory that originated this thread.
func (s *Store) CreateThreadWithWorkspace(title, createdByType, createdByID, targetAgent, workspace string) (*Thread, error) {
	now := time.Now()
	threadID := fmt.Sprintf("thread_%d_%s", now.UnixMilli(), generateRandomID(8))

	// Store target_agent and workspace in context_json
	var contextJSON *string
	ctx := ThreadContext{
		TargetAgent: targetAgent,
		Workspace:   workspace,
	}
	if targetAgent != "" || workspace != "" {
		contextBytes, err := json.Marshal(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal context: %w", err)
		}
		contextStr := string(contextBytes)
		contextJSON = &contextStr
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
		Workspace:     workspace,
		LastSeq:       0,
		UpdatedAt:     now,
	}, nil
}

// GetOrCreateThreadWithWorkspace returns an existing thread with the same title and target agent,
// or creates a new one if none exists. This prevents duplicate threads for the same task.
func (s *Store) GetOrCreateThreadWithWorkspace(title, createdByType, createdByID, targetAgent, workspace string) (*Thread, bool, error) {
	// First, check if a thread with this title and target agent already exists
	existing, err := s.GetThreadByTitleAndAgent(title, targetAgent)
	if err == nil && existing != nil {
		// Thread already exists, return it
		return existing, false, nil
	}

	// Create new thread
	thread, err := s.CreateThreadWithWorkspace(title, createdByType, createdByID, targetAgent, workspace)
	if err != nil {
		return nil, false, err
	}
	return thread, true, nil
}

// GetThreadByTitleAndAgent finds a thread by title and target agent.
// Returns nil, nil if no matching thread is found.
func (s *Store) GetThreadByTitleAndAgent(title, targetAgent string) (*Thread, error) {
	var thread Thread
	var createdAtMs, updatedAtMs int64
	var contextJSON sql.NullString

	// Query threads and filter by context_json containing the target agent
	// We search for threads with matching title and parse context to check target_agent
	rows, err := s.db.Query(`
		SELECT id, title, created_at, created_by_type, created_by_id, status, context_json, last_seq, updated_at
		FROM threads
		WHERE title = ? AND status = 'active'
		ORDER BY created_at DESC
	`, title)
	if err != nil {
		return nil, fmt.Errorf("failed to query threads: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&thread.ID, &thread.Title, &createdAtMs, &thread.CreatedByType, &thread.CreatedByID,
			&thread.Status, &contextJSON, &thread.LastSeq, &updatedAtMs,
		)
		if err != nil {
			continue
		}

		thread.CreatedAt = time.UnixMilli(createdAtMs)
		thread.UpdatedAt = time.UnixMilli(updatedAtMs)
		if contextJSON.Valid {
			thread.ContextJSON = contextJSON.String
			ctx := parseThreadContext(contextJSON.String)
			thread.TargetAgent = ctx.TargetAgent
			thread.Workspace = ctx.Workspace

			// Check if target agent matches
			if ctx.TargetAgent == targetAgent {
				return &thread, nil
			}
		} else if targetAgent == "" {
			// No context and no target agent requested - match
			return &thread, nil
		}
	}

	return nil, nil // No matching thread found
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

// SetThreadTargetAgent updates the target agent for a thread.
// This should be called when a task is re-routed to a different agent.
func (s *Store) SetThreadTargetAgent(threadID, targetAgent string) error {
	// Get existing context
	thread, err := s.GetThread(threadID)
	if err != nil {
		return err
	}

	// Parse existing context or create new one
	ctx := parseThreadContext(thread.ContextJSON)
	ctx.TargetAgent = targetAgent

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
		return fmt.Errorf("failed to update thread target agent: %w", err)
	}

	return nil
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

// ThreadFilter represents filter options for querying threads
type ThreadFilter struct {
	Status    string // Filter by status (active, paused, resolved, archived)
	Workspace string // Filter by workspace
	Limit     int    // Maximum number of results
}

// NewThreadFilter creates a new ThreadFilter with the given parameters
func (s *Store) NewThreadFilter(status, workspace string, limit int) ThreadFilter {
	return ThreadFilter{
		Status:    status,
		Workspace: workspace,
		Limit:     limit,
	}
}

// GetThreadsFiltered returns threads matching the given filter
func (s *Store) GetThreadsFiltered(filter ThreadFilter) ([]Thread, error) {
	if filter.Limit <= 0 {
		filter.Limit = 100
	}

	// Get all threads first, then filter by workspace (since it's in context_json)
	// For better performance with large datasets, consider adding a workspace column
	threads, err := s.GetThreadsByStatus(filter.Status, 1000) // Get more to account for filtering
	if err != nil {
		return nil, err
	}

	// Filter by workspace if specified
	if filter.Workspace != "" {
		filtered := make([]Thread, 0)
		for _, t := range threads {
			if t.Workspace == filter.Workspace {
				filtered = append(filtered, t)
				if len(filtered) >= filter.Limit {
					break
				}
			}
		}
		return filtered, nil
	}

	// Apply limit
	if len(threads) > filter.Limit {
		return threads[:filter.Limit], nil
	}
	return threads, nil
}

// GetDistinctWorkspaces returns a list of all unique workspaces from threads
func (s *Store) GetDistinctWorkspaces() ([]string, error) {
	// Get all threads and extract unique workspaces
	threads, err := s.GetThreadsByStatus("", 1000)
	if err != nil {
		return nil, err
	}

	workspaceSet := make(map[string]bool)
	for _, t := range threads {
		if t.Workspace != "" {
			workspaceSet[t.Workspace] = true
		}
	}

	workspaces := make([]string, 0, len(workspaceSet))
	for w := range workspaceSet {
		workspaces = append(workspaces, w)
	}
	return workspaces, nil
}

// ThreadAggregateStats provides aggregate statistics about threads
type ThreadAggregateStats struct {
	TotalThreads int            `json:"total_threads"`
	ByStatus     map[string]int `json:"by_status"`
	ByWorkspace  map[string]int `json:"by_workspace"`
}

// GetThreadAggregateStats returns aggregate statistics about threads
func (s *Store) GetThreadAggregateStats() (*ThreadAggregateStats, error) {
	threads, err := s.GetThreadsByStatus("", 1000)
	if err != nil {
		return nil, err
	}

	stats := &ThreadAggregateStats{
		TotalThreads: len(threads),
		ByStatus:     make(map[string]int),
		ByWorkspace:  make(map[string]int),
	}

	for _, t := range threads {
		// Count by status
		stats.ByStatus[t.Status]++

		// Count by workspace
		workspace := t.Workspace
		if workspace == "" {
			workspace = "(no workspace)"
		}
		stats.ByWorkspace[workspace]++
	}

	return stats, nil
}

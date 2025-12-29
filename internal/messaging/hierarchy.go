package messaging

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// AgentInfo represents information about a known agent
type AgentInfo struct {
	ID         string `json:"id"`
	LastActive int64  `json:"last_active,omitempty"`
	Status     string `json:"status,omitempty"`
	Label      string `json:"label,omitempty"`
}

// Badge represents a status badge on a hierarchy node
type Badge struct {
	Type  string `json:"type"`  // "unread", "pending", "running"
	Count int    `json:"count"` // Number of items
}

// HierarchyNode represents a node in the agent/thread hierarchy tree
type HierarchyNode struct {
	Type     string          `json:"type"`               // "root", "agent", "thread"
	ID       string          `json:"id"`                 // Unique identifier
	Label    string          `json:"label"`              // Display label
	Status   string          `json:"status,omitempty"`   // "active", "idle", "pending"
	Badges   []Badge         `json:"badges,omitempty"`   // Status badges
	Children []HierarchyNode `json:"children,omitempty"` // Child nodes
}

// AgentStats represents detailed statistics for a single agent
type AgentStats struct {
	AgentID          string        `json:"agent_id"`
	Status           string        `json:"status"` // "active", "idle", "pending"
	ThreadCount      int           `json:"thread_count"`
	UnreadMessages   int           `json:"unread_messages"`
	PendingApprovals int           `json:"pending_approvals"`
	RunningProcesses int           `json:"running_processes"`
	LastActivity     string        `json:"last_activity,omitempty"`
	Threads          []ThreadStats `json:"threads,omitempty"`
}

// ThreadStats represents statistics for a thread within an agent
type ThreadStats struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	UnreadCount      int    `json:"unread_count"`
	PendingApprovals int    `json:"pending_approvals"`
	RunningProcesses int    `json:"running_processes"`
	LastMessageAt    string `json:"last_message_at,omitempty"`
}

// ExecutionStats represents aggregated execution statistics
type ExecutionStats struct {
	TotalExecutions        int     `json:"total_executions"`
	SuccessfulExecutions   int     `json:"successful_executions"`
	FailedExecutions       int     `json:"failed_executions"`
	TotalDurationMS        int64   `json:"total_duration_ms"`
	TotalCost              float64 `json:"total_cost"`
	TotalInputTokens       int     `json:"total_input_tokens"`
	TotalOutputTokens      int     `json:"total_output_tokens"`
	TotalCacheReadTokens   int     `json:"total_cache_read_tokens"`
	TotalCacheCreateTokens int     `json:"total_cache_create_tokens"`
	TotalFilesCreated      int     `json:"total_files_created"`
}

// AggregateStats represents overall statistics across all agents
type AggregateStats struct {
	TotalAgents      int            `json:"total_agents"`
	ActiveAgents     int            `json:"active_agents"`
	IdleAgents       int            `json:"idle_agents"`
	PendingApprovals int            `json:"pending_approvals"`
	RunningProcesses int            `json:"running_processes"`
	TotalThreads     int            `json:"total_threads"`
	Execution        ExecutionStats `json:"execution"`
}

// HierarchyResponse is the response for the /api/hierarchy endpoint
type HierarchyResponse struct {
	Root      HierarchyNode  `json:"root"`
	Aggregate AggregateStats `json:"aggregate"`
}

// ExecutionMetadata contains per-message execution details including file list
type ExecutionMetadata struct {
	Success             bool     `json:"success"`
	DurationMS          int      `json:"duration_ms"`
	NumTurns            int      `json:"num_turns"`
	Cost                float64  `json:"cost"`
	SessionID           string   `json:"session_id"`
	InputTokens         int      `json:"input_tokens"`
	OutputTokens        int      `json:"output_tokens"`
	CacheReadTokens     int      `json:"cache_read_tokens"`
	CacheCreationTokens int      `json:"cache_creation_tokens"`
	FilesCreatedCount   int      `json:"files_created_count"`
	FilesCreated        []string `json:"files_created"`
	Workspace           string   `json:"workspace"`
}

// parseExecutionMetadataFromMessage extracts full execution metadata from a message's metadata_json
func parseExecutionMetadataFromMessage(metadataJSON string) *ExecutionMetadata {
	if metadataJSON == "" {
		return nil
	}

	var metadata struct {
		ExecutionStats ExecutionMetadata `json:"execution_stats"`
	}

	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return nil
	}

	stats := metadata.ExecutionStats
	// Check if we actually have execution stats (not empty struct)
	if stats.DurationMS == 0 && stats.Cost == 0 && stats.InputTokens == 0 && stats.OutputTokens == 0 {
		return nil
	}

	return &stats
}

// parseExecutionStatsFromMetadata extracts execution stats from a message's metadata_json
func parseExecutionStatsFromMetadata(metadataJSON string) *ExecutionStats {
	meta := parseExecutionMetadataFromMessage(metadataJSON)
	if meta == nil {
		return nil
	}

	execStats := &ExecutionStats{
		TotalExecutions:        1,
		TotalDurationMS:        int64(meta.DurationMS),
		TotalCost:              meta.Cost,
		TotalInputTokens:       meta.InputTokens,
		TotalOutputTokens:      meta.OutputTokens,
		TotalCacheReadTokens:   meta.CacheReadTokens,
		TotalCacheCreateTokens: meta.CacheCreationTokens,
		TotalFilesCreated:      meta.FilesCreatedCount,
	}
	if meta.Success {
		execStats.SuccessfulExecutions = 1
	} else {
		execStats.FailedExecutions = 1
	}

	return execStats
}

// GetAggregatedExecutionStats aggregates execution stats from all result messages
func (s *Store) GetAggregatedExecutionStats() (*ExecutionStats, error) {
	// Query all result messages with metadata
	rows, err := s.db.Query(`
		SELECT metadata_json
		FROM messages
		WHERE kind = 'result'
		  AND metadata_json IS NOT NULL
		  AND metadata_json != ''
		  AND deleted_at IS NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query result messages: %w", err)
	}
	defer rows.Close()

	aggregate := &ExecutionStats{}

	for rows.Next() {
		var metadataJSON string
		if err := rows.Scan(&metadataJSON); err != nil {
			continue
		}

		stats := parseExecutionStatsFromMetadata(metadataJSON)
		if stats != nil {
			aggregate.TotalExecutions += stats.TotalExecutions
			aggregate.SuccessfulExecutions += stats.SuccessfulExecutions
			aggregate.FailedExecutions += stats.FailedExecutions
			aggregate.TotalDurationMS += stats.TotalDurationMS
			aggregate.TotalCost += stats.TotalCost
			aggregate.TotalInputTokens += stats.TotalInputTokens
			aggregate.TotalOutputTokens += stats.TotalOutputTokens
			aggregate.TotalCacheReadTokens += stats.TotalCacheReadTokens
			aggregate.TotalCacheCreateTokens += stats.TotalCacheCreateTokens
			aggregate.TotalFilesCreated += stats.TotalFilesCreated
		}
	}

	return aggregate, rows.Err()
}

// GetExecutionStatsByThread aggregates execution stats for a specific thread
func (s *Store) GetExecutionStatsByThread(threadID string) (*ExecutionStats, error) {
	rows, err := s.db.Query(`
		SELECT metadata_json
		FROM messages
		WHERE thread_id = ?
		  AND kind = 'result'
		  AND metadata_json IS NOT NULL
		  AND metadata_json != ''
		  AND deleted_at IS NULL
	`, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to query result messages: %w", err)
	}
	defer rows.Close()

	aggregate := &ExecutionStats{}

	for rows.Next() {
		var metadataJSON string
		if err := rows.Scan(&metadataJSON); err != nil {
			continue
		}

		stats := parseExecutionStatsFromMetadata(metadataJSON)
		if stats != nil {
			aggregate.TotalExecutions += stats.TotalExecutions
			aggregate.SuccessfulExecutions += stats.SuccessfulExecutions
			aggregate.FailedExecutions += stats.FailedExecutions
			aggregate.TotalDurationMS += stats.TotalDurationMS
			aggregate.TotalCost += stats.TotalCost
			aggregate.TotalInputTokens += stats.TotalInputTokens
			aggregate.TotalOutputTokens += stats.TotalOutputTokens
			aggregate.TotalCacheReadTokens += stats.TotalCacheReadTokens
			aggregate.TotalCacheCreateTokens += stats.TotalCacheCreateTokens
			aggregate.TotalFilesCreated += stats.TotalFilesCreated
		}
	}

	return aggregate, rows.Err()
}

// GetHierarchy returns the complete agent/thread hierarchy tree
func (s *Store) GetHierarchy() (*HierarchyResponse, error) {
	// Get all agents
	agents, err := s.GetKnownAgents()
	if err != nil {
		return nil, fmt.Errorf("failed to get agents: %w", err)
	}

	// Get all threads
	threads, err := s.GetThreadsByStatus("", 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get threads: %w", err)
	}

	// Get all pending approvals
	pendingApprovals, err := s.GetApprovalsByStatus("pending", 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending approvals: %w", err)
	}

	// Build approval counts by thread
	approvalsByThread := make(map[string]int)
	for _, approval := range pendingApprovals {
		approvalsByThread[approval.ThreadID]++
	}

	// Build thread nodes grouped by agent
	threadsByAgent := make(map[string][]HierarchyNode)
	for _, thread := range threads {
		agentID := thread.TargetAgent
		if agentID == "" {
			agentID = "unassigned"
		}

		var badges []Badge
		pendingCount := approvalsByThread[thread.ID]
		if pendingCount > 0 {
			badges = append(badges, Badge{Type: "pending", Count: pendingCount})
		}

		threadNode := HierarchyNode{
			Type:   "thread",
			ID:     thread.ID,
			Label:  thread.Title,
			Badges: badges,
		}
		threadsByAgent[agentID] = append(threadsByAgent[agentID], threadNode)
	}

	// Build set of known agent IDs for lookup
	knownAgents := make(map[string]bool)
	for _, agent := range agents {
		knownAgents[agent.ID] = true
	}

	// Build agent nodes
	var agentNodes []HierarchyNode
	activeCount := 0
	idleCount := 0

	for _, agent := range agents {
		childThreads := threadsByAgent[agent.ID]

		// Calculate agent status based on pending approvals
		status := "idle"
		var agentBadges []Badge
		pendingCount := 0
		for _, threadNode := range childThreads {
			for _, badge := range threadNode.Badges {
				if badge.Type == "pending" {
					pendingCount += badge.Count
				}
			}
		}
		if pendingCount > 0 {
			status = "pending"
			agentBadges = append(agentBadges, Badge{Type: "pending", Count: pendingCount})
		}

		if status == "idle" {
			idleCount++
		} else {
			activeCount++
		}

		agentNode := HierarchyNode{
			Type:     "agent",
			ID:       agent.ID,
			Label:    agent.ID,
			Status:   status,
			Badges:   agentBadges,
			Children: childThreads,
		}
		agentNodes = append(agentNodes, agentNode)
	}

	// Add threads for unknown agents (agents referenced by threads but not in database)
	for agentID, childThreads := range threadsByAgent {
		if agentID == "unassigned" {
			continue // Handle separately below
		}
		if !knownAgents[agentID] && len(childThreads) > 0 {
			// This is an unknown agent with threads - create a node for it
			agentNodes = append(agentNodes, HierarchyNode{
				Type:     "agent",
				ID:       agentID,
				Label:    agentID,
				Status:   "idle",
				Children: childThreads,
			})
			idleCount++
		}
	}

	// Add unassigned threads if any
	if unassignedThreads, ok := threadsByAgent["unassigned"]; ok && len(unassignedThreads) > 0 {
		agentNodes = append(agentNodes, HierarchyNode{
			Type:     "agent",
			ID:       "unassigned",
			Label:    "Unassigned",
			Status:   "idle",
			Children: unassignedThreads,
		})
	}

	// Build root node
	root := HierarchyNode{
		Type:     "root",
		ID:       "all",
		Label:    "All Agents",
		Children: agentNodes,
	}

	// Get aggregated execution stats
	execStats, err := s.GetAggregatedExecutionStats()
	if err != nil {
		// Non-fatal - use empty stats
		execStats = &ExecutionStats{}
	}

	// Build aggregate stats
	aggregate := AggregateStats{
		TotalAgents:      len(agents),
		ActiveAgents:     activeCount,
		IdleAgents:       idleCount,
		PendingApprovals: len(pendingApprovals),
		RunningProcesses: 0, // TODO: Add process tracking
		TotalThreads:     len(threads),
		Execution:        *execStats,
	}

	return &HierarchyResponse{
		Root:      root,
		Aggregate: aggregate,
	}, nil
}

// GetAgentStats returns detailed statistics for a single agent
func (s *Store) GetAgentStats(agentID string) (*AgentStats, error) {
	// Get threads for this agent
	threads, err := s.GetThreadsByStatus("", 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get threads: %w", err)
	}

	// Filter threads for this agent
	var agentThreads []Thread
	for _, thread := range threads {
		if thread.TargetAgent == agentID || (thread.TargetAgent == "" && agentID == "unassigned") {
			agentThreads = append(agentThreads, thread)
		}
	}

	// Get pending approvals
	pendingApprovals, err := s.GetApprovalsByStatus("pending", 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending approvals: %w", err)
	}

	// Build approval counts by thread
	approvalsByThread := make(map[string]int)
	for _, approval := range pendingApprovals {
		approvalsByThread[approval.ThreadID]++
	}

	// Build thread stats
	var threadStats []ThreadStats
	totalPending := 0
	for _, thread := range agentThreads {
		pendingCount := approvalsByThread[thread.ID]
		totalPending += pendingCount

		threadStats = append(threadStats, ThreadStats{
			ID:               thread.ID,
			Title:            thread.Title,
			UnreadCount:      0, // TODO: Calculate unread
			PendingApprovals: pendingCount,
			RunningProcesses: 0, // TODO: Add process tracking
			LastMessageAt:    thread.UpdatedAt.Format(time.RFC3339),
		})
	}

	// Determine status
	status := "idle"
	if totalPending > 0 {
		status = "pending"
	}

	return &AgentStats{
		AgentID:          agentID,
		Status:           status,
		ThreadCount:      len(agentThreads),
		UnreadMessages:   0, // TODO: Calculate unread
		PendingApprovals: totalPending,
		RunningProcesses: 0, // TODO: Add process tracking
		Threads:          threadStats,
	}, nil
}

// GetKnownAgents returns a list of known agent IDs from the database
func (s *Store) GetKnownAgents() ([]AgentInfo, error) {
	// Query for distinct agent IDs from multiple sources:
	// 1. Registered agents (agents table)
	// 2. Messages sent to ailang_instance (to_id)
	// 3. Subscriptions (instance_id)
	// 4. Agents that sent messages
	query := `
		SELECT DISTINCT agent_id, MAX(last_active) as last_active, MAX(status) as status, MAX(label) as label FROM (
			-- Registered agents (coordinator, etc.)
			SELECT id as agent_id, last_active_at as last_active, status, label
			FROM agents

			UNION ALL

			-- Agents that received messages
			SELECT DISTINCT to_id as agent_id, created_at as last_active, NULL as status, NULL as label
			FROM messages
			WHERE to_type = 'ailang_instance' AND to_id IS NOT NULL AND to_id != ''

			UNION ALL

			-- Agents with subscriptions
			SELECT DISTINCT instance_id as agent_id, subscribed_at as last_active, NULL as status, NULL as label
			FROM subscriptions
			WHERE instance_id IS NOT NULL AND instance_id != ''

			UNION ALL

			-- Agents that sent messages
			SELECT DISTINCT from_id as agent_id, created_at as last_active, NULL as status, NULL as label
			FROM messages
			WHERE from_type = 'ailang_instance' AND from_id IS NOT NULL AND from_id != ''
		)
		GROUP BY agent_id
		ORDER BY last_active DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	var agents []AgentInfo
	for rows.Next() {
		var agent AgentInfo
		var lastActive sql.NullInt64
		var status, label sql.NullString
		if err := rows.Scan(&agent.ID, &lastActive, &status, &label); err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		if lastActive.Valid {
			agent.LastActive = lastActive.Int64
		}
		if status.Valid {
			agent.Status = status.String
		}
		if label.Valid {
			agent.Label = label.String
		}
		agents = append(agents, agent)
	}

	// Always include a default agent if none found
	if len(agents) == 0 {
		agents = append(agents, AgentInfo{ID: "my-agent"})
	}

	return agents, rows.Err()
}

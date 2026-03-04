package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/observatory"
)

// ============================================================================
// Unified Stats API (Observatory + Coordinator)
// ============================================================================

// UnifiedStatsResponse combines Observatory telemetry with Coordinator runtime state
type UnifiedStatsResponse struct {
	// Observatory metrics (canonical source of truth for telemetry)
	Observatory *ObservatoryStats `json:"observatory"`

	// Coordinator runtime state (subset of observatory - delegated tasks only)
	Coordinator *CoordinatorRuntimeStats `json:"coordinator"`

	// Metadata about data sources
	Sources DataSources `json:"sources"`
}

// ObservatoryStats holds metrics from the Observatory database
type ObservatoryStats struct {
	TotalSpans      int     `json:"total_spans"`
	TotalTasks      int     `json:"total_tasks"`
	TotalWorkspaces int     `json:"total_workspaces"`
	TotalAgents     int     `json:"total_agents"`
	TotalTokensIn   int64   `json:"total_tokens_in"`
	TotalTokensOut  int64   `json:"total_tokens_out"`
	TotalCostUSD    float64 `json:"total_cost_usd"`
	SuccessRate     float64 `json:"success_rate"`

	// Cache metrics (from spans)
	TotalCacheReadTokens     int64   `json:"total_cache_read_tokens"`
	TotalCacheCreationTokens int64   `json:"total_cache_creation_tokens"`
	CacheSavingsUSD          float64 `json:"cache_savings_usd"`

	// Lines of Code metrics (from metrics table)
	LinesAdded   int64 `json:"lines_added"`
	LinesRemoved int64 `json:"lines_removed"`

	// Activity metrics (from metrics table)
	CommitCount      int64 `json:"commit_count"`
	PullRequestCount int64 `json:"pull_request_count"`
	ActiveTimeMs     int64 `json:"active_time_ms"`

	// Session metrics
	TurnCount  int `json:"turn_count"`
	ToolCalls  int `json:"tool_calls"`
	ErrorCount int `json:"error_count"`
}

// CoordinatorRuntimeStats holds live coordinator state
type CoordinatorRuntimeStats struct {
	Running          bool    `json:"running"`
	CompletedTasks   int     `json:"completed_tasks"`
	PendingTasks     int     `json:"pending_tasks"`
	RunningTasks     int     `json:"running_tasks"`
	FailedTasks      int     `json:"failed_tasks"`
	PendingApprovals int     `json:"pending_approvals"`
	ActiveAgents     int     `json:"active_agents"`
	TotalCost        float64 `json:"total_cost"`   // Coordinator-tracked subset
	TotalTokens      int     `json:"total_tokens"` // Coordinator-tracked subset
}

// DataSources documents where each metric comes from
type DataSources struct {
	ObservatoryDB string `json:"observatory_db"` // Path to observatory.db
	CoordinatorDB string `json:"coordinator_db"` // Path to coordinator.db
	ObservatoryOK bool   `json:"observatory_ok"` // Whether observatory is available
	CoordinatorOK bool   `json:"coordinator_ok"` // Whether coordinator is available
}

// parseControlPlaneFilter extracts filter parameters from query string
func parseControlPlaneFilter(r *http.Request) *observatory.ControlPlaneFilter {
	q := r.URL.Query()
	filter := &observatory.ControlPlaneFilter{
		SourceType: q.Get("source_type"),
		Provider:   q.Get("provider"),
		Model:      q.Get("model"),
		Workspace:  q.Get("workspace"),
		StartDate:  q.Get("start_date"), // YYYY-MM-DD format
		EndDate:    q.Get("end_date"),   // YYYY-MM-DD format
	}
	return filter
}

// GET /api/controlplane/stats - Get unified stats from Observatory + Coordinator
func (s *Server) handleControlPlaneStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	filter := parseControlPlaneFilter(r)

	response := UnifiedStatsResponse{
		Sources: DataSources{
			ObservatoryDB: "~/.ailang/state/observatory.db",
			CoordinatorDB: "~/.ailang/state/coordinator.db",
		},
	}

	// Load workspace config for reverse mapping (workspace ID -> path patterns)
	wsConfig := coordinator.LoadWorkspacesConfig()

	// Get Observatory metrics (canonical source of truth)
	if s.obsBackend != nil {
		// Use filtered metrics if SQLite backend with filter support
		var metrics *observatory.MetricsSummary
		var err error

		if sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend); ok && !filter.IsEmpty() {
			metrics, err = sqliteBackend.GetFilteredMetricsSummary(ctx, filter, wsConfig)
		} else {
			metrics, err = s.obsBackend.GetMetricsSummary(ctx)
		}

		if err != nil {
			log.Printf("Failed to get observatory metrics: %v", err)
		} else if metrics != nil {
			response.Observatory = &ObservatoryStats{
				TotalSpans:               metrics.TotalSpans,
				TotalTasks:               metrics.TotalTasks,
				TotalWorkspaces:          metrics.TotalWorkspaces,
				TotalAgents:              metrics.TotalAgents,
				TotalTokensIn:            metrics.TotalTokensIn,
				TotalTokensOut:           metrics.TotalTokensOut,
				TotalCostUSD:             metrics.TotalCostUSD,
				SuccessRate:              metrics.SuccessRate,
				TotalCacheReadTokens:     metrics.TotalCacheReadTokens,
				TotalCacheCreationTokens: metrics.TotalCacheCreationTokens,
				CacheSavingsUSD:          metrics.CacheSavingsUSD,
				LinesAdded:               metrics.LinesAdded,
				LinesRemoved:             metrics.LinesRemoved,
				CommitCount:              metrics.CommitCount,
				PullRequestCount:         metrics.PullRequestCount,
				ActiveTimeMs:             metrics.ActiveTimeMs,
				TurnCount:                metrics.TurnCount,
				ToolCalls:                metrics.ToolCalls,
				ErrorCount:               metrics.ErrorCount,
			}
			response.Sources.ObservatoryOK = true
		}
	}

	// Get Coordinator runtime state (subset - delegated tasks only)
	coordStore := s.getCoordStoreForControlPlane()
	if coordStore != nil {
		// Get basic stats from coordinator store interface
		if s.coordStore != nil {
			coordStats, err := s.coordStore.GetCoordinatorStats()
			if err == nil && coordStats != nil {
				response.Coordinator = &CoordinatorRuntimeStats{
					Running:        coordStats.Running,
					CompletedTasks: coordStats.TasksRun,
					PendingTasks:   coordStats.PendingTasks,
					RunningTasks:   coordStats.RunningTasks,
					FailedTasks:    coordStats.FailedTasks,
					TotalCost:      coordStats.TotalCost,
					TotalTokens:    coordStats.TotalTokens,
				}
				response.Sources.CoordinatorOK = true

				// Get additional stats (pending approvals, active agents)
				taskStats, err := coordStore.GetTaskStats(ctx)
				if err == nil && taskStats != nil {
					response.Coordinator.PendingApprovals = taskStats.PendingApprovals
				}

				// Count active agents (agents with running tasks)
				runningTasks, err := coordStore.ListTasks(ctx, &coordinator.TaskFilter{
					Status: []coordinator.TaskStatus{coordinator.TaskStatusRunning},
				})
				if err == nil {
					agentSet := make(map[string]bool)
					for _, task := range runningTasks {
						if task.AgentID != "" {
							agentSet[task.AgentID] = true
						}
					}
					response.Coordinator.ActiveAgents = len(agentSet)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode unified stats response: %v", err)
	}
}

// ============================================================================
// Breakdown/Drill-Down API
// ============================================================================

// BreakdownItem represents a single item in a breakdown
type BreakdownItem struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	SpanCount  int     `json:"span_count"`
	TaskCount  int     `json:"task_count,omitempty"`
	TokensIn   int64   `json:"tokens_in"`
	TokensOut  int64   `json:"tokens_out"`
	CostUSD    float64 `json:"cost_usd"`
	DurationMs int64   `json:"duration_ms"`          // Total execution time in ms
	Percentage float64 `json:"percentage,omitempty"` // Percentage of total cost

	// Cache metrics
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	CacheSavingsUSD     float64 `json:"cache_savings_usd"`
}

// BreakdownResponse contains hierarchical breakdown data
type BreakdownResponse struct {
	// By provider (claude, gemini, openai)
	ByProvider []BreakdownItem `json:"by_provider"`

	// By source type (inferred from span name patterns)
	BySourceType []BreakdownItem `json:"by_source_type"`

	// By model
	ByModel []BreakdownItem `json:"by_model"`

	// By workspace
	ByWorkspace []BreakdownItem `json:"by_workspace"`

	// Totals for percentage calculations
	TotalCost float64 `json:"total_cost"`
}

// GET /api/controlplane/stats/breakdown - Get breakdown data for drill-down
func (s *Server) handleControlPlaneStatsBreakdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := BreakdownResponse{}

	// Get Observatory backend for direct SQL queries
	if s.obsBackend == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	ctx := r.Context()
	filter := parseControlPlaneFilter(r)

	// Type assert to SQLiteBackend which has breakdown methods
	sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend)
	if !ok {
		log.Printf("Observatory backend does not support breakdown queries (type: %T)", s.obsBackend)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Load workspace config for path-to-ID mapping and reverse mapping
	wsConfig := coordinator.LoadWorkspacesConfig()

	// Get total cost (filtered if filter is set)
	var metrics *observatory.MetricsSummary
	var err error
	if !filter.IsEmpty() {
		metrics, err = sqliteBackend.GetFilteredMetricsSummary(ctx, filter, wsConfig)
	} else {
		metrics, err = s.obsBackend.GetMetricsSummary(ctx)
	}
	if err == nil && metrics != nil {
		response.TotalCost = metrics.TotalCostUSD
	}

	// Get breakdowns sequentially (SQLite serializes concurrent reads on same connection).
	if !filter.IsEmpty() {
		if items, err := sqliteBackend.GetFilteredBreakdownByProvider(ctx, filter, wsConfig); err == nil {
			response.ByProvider = convertBreakdownItems(items)
		}
		if items, err := sqliteBackend.GetFilteredBreakdownBySourceType(ctx, filter, wsConfig); err == nil {
			response.BySourceType = convertBreakdownItems(items)
		}
		if items, err := sqliteBackend.GetFilteredBreakdownByModel(ctx, filter, wsConfig); err == nil {
			response.ByModel = convertBreakdownItems(items)
		}
		if items, err := sqliteBackend.GetFilteredBreakdownByWorkspaceWithMapping(ctx, filter, wsConfig, wsConfig); err == nil {
			response.ByWorkspace = convertBreakdownItems(items)
		}
	} else {
		if items, err := sqliteBackend.GetBreakdownByProvider(ctx); err == nil {
			response.ByProvider = convertBreakdownItems(items)
		}
		if items, err := sqliteBackend.GetBreakdownBySourceType(ctx); err == nil {
			response.BySourceType = convertBreakdownItems(items)
		}
		if items, err := sqliteBackend.GetBreakdownByModel(ctx); err == nil {
			response.ByModel = convertBreakdownItems(items)
		}
		if items, err := sqliteBackend.GetFilteredBreakdownByWorkspaceWithMapping(ctx, nil, wsConfig, wsConfig); err == nil {
			response.ByWorkspace = convertBreakdownItems(items)
		}
	}

	// Merge inbox message counts into source type breakdown
	if s.store != nil {
		inboxCounts := s.countInboxMessagesBySourceType()
		response.BySourceType = mergeInboxCountsIntoBreakdown(response.BySourceType, inboxCounts)
	}

	// Calculate percentages
	if response.TotalCost > 0 {
		for i := range response.ByProvider {
			response.ByProvider[i].Percentage = (response.ByProvider[i].CostUSD / response.TotalCost) * 100
		}
		for i := range response.BySourceType {
			response.BySourceType[i].Percentage = (response.BySourceType[i].CostUSD / response.TotalCost) * 100
		}
		for i := range response.ByModel {
			response.ByModel[i].Percentage = (response.ByModel[i].CostUSD / response.TotalCost) * 100
		}
		for i := range response.ByWorkspace {
			response.ByWorkspace[i].Percentage = (response.ByWorkspace[i].CostUSD / response.TotalCost) * 100
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode breakdown response: %v", err)
	}
}

// convertBreakdownItems converts observatory.BreakdownItem to server.BreakdownItem
func convertBreakdownItems(items []observatory.BreakdownItem) []BreakdownItem {
	result := make([]BreakdownItem, len(items))
	for i, item := range items {
		result[i] = BreakdownItem{
			ID:                  item.ID,
			Label:               item.Label,
			SpanCount:           item.SpanCount,
			TaskCount:           item.TaskCount,
			TokensIn:            item.TokensIn,
			TokensOut:           item.TokensOut,
			CostUSD:             item.CostUSD,
			DurationMs:          item.DurationMs,
			CacheReadTokens:     item.CacheReadTokens,
			CacheCreationTokens: item.CacheCreationTokens,
			CacheSavingsUSD:     item.CacheSavingsUSD,
		}
	}
	return result
}

// InboxSourceCount holds message count for a source type
type InboxSourceCount struct {
	ID           string
	Label        string
	MessageCount int
}

// countInboxMessagesBySourceType counts inbox messages grouped by source type.
// Uses InferInboxSourceType from handlers_inbox.go for consistent taxonomy.
func (s *Server) countInboxMessagesBySourceType() []InboxSourceCount {
	// Get all recent messages (limit to last 1000 for performance)
	messages, err := s.store.ListInboxMessages(messaging.InboxListOptions{Limit: 1000})
	if err != nil {
		return nil
	}

	// Count by source type
	counts := make(map[string]int)
	for _, msg := range messages {
		sourceType := InferInboxSourceType(msg.FromAgent, msg.ToInbox)
		counts[sourceType]++
	}

	// Convert to slice with labels
	labelMap := map[string]string{
		"github":       "GitHub",
		"eval":         "Eval Benchmarks",
		"coordinator":  "Coordinator Tasks",
		"user_session": "User Sessions",
		"messaging":    "Messaging",
		"cli":          "CLI Usage",
		"direct_api":   "Direct API",
		"other":        "Other",
	}

	var result []InboxSourceCount
	for id, count := range counts {
		label := labelMap[id]
		if label == "" {
			label = id
		}
		result = append(result, InboxSourceCount{
			ID:           id,
			Label:        label,
			MessageCount: count,
		})
	}

	return result
}

// mergeInboxCountsIntoBreakdown adds inbox message counts to the span-based breakdown.
// If a source type exists in both, adds the message count.
// If a source type only exists in inbox (e.g., GitHub), adds it as a new entry.
func mergeInboxCountsIntoBreakdown(breakdown []BreakdownItem, inboxCounts []InboxSourceCount) []BreakdownItem {
	if len(inboxCounts) == 0 {
		return breakdown
	}

	// Create map of existing breakdown items
	existing := make(map[string]int) // id -> index
	for i, item := range breakdown {
		existing[item.ID] = i
	}

	// Merge inbox counts
	for _, inbox := range inboxCounts {
		if idx, ok := existing[inbox.ID]; ok {
			// Source exists in breakdown - add message count
			breakdown[idx].TaskCount += inbox.MessageCount
		} else {
			// Source only exists in inbox (e.g., GitHub) - add new entry
			breakdown = append(breakdown, BreakdownItem{
				ID:        inbox.ID,
				Label:     inbox.Label,
				SpanCount: 0,
				TaskCount: inbox.MessageCount,
				TokensIn:  0,
				TokensOut: 0,
				CostUSD:   0, // Inbox messages have no cost
			})
		}
	}

	return breakdown
}

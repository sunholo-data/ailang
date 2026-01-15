package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/observatory"
)

// ============================================================================
// Heatmap API Types
// ============================================================================

// HeatmapCell represents a single day's activity data
type HeatmapCell struct {
	Date        string  `json:"date"` // YYYY-MM-DD
	TaskCount   int     `json:"taskCount"`
	Cost        float64 `json:"cost"`
	SuccessRate float64 `json:"successRate"` // 0.0 to 1.0
}

// HeatmapResponse is the response for GET /api/controlplane/heatmap
type HeatmapResponse struct {
	Cells  []HeatmapCell `json:"cells"`
	Totals struct {
		Tasks int     `json:"tasks"`
		Cost  float64 `json:"cost"`
	} `json:"totals"`
}

// HeatmapGridCell is a cell in the grid format response
type HeatmapGridCell struct {
	Date        string  `json:"date"`
	TaskCount   int     `json:"count"`
	Cost        float64 `json:"cost"`
	SuccessRate float64 `json:"successRate"`
	Intensity   float64 `json:"intensity"` // 0.0-1.0 for coloring
	DayOfWeek   int     `json:"dayOfWeek"` // 0=Sunday, 6=Saturday
}

// HeatmapMonthLabel is a month label for the grid header
type HeatmapMonthLabel struct {
	Name      string `json:"name"`      // "Jan", "Feb", etc.
	WeekIndex int    `json:"weekIndex"` // 0-based week column index
}

// HeatmapGridResponse is the grid format response for heatmap
type HeatmapGridResponse struct {
	Weeks       [][]HeatmapGridCell `json:"weeks"`       // weeks[weekIndex][dayIndex]
	MonthLabels []HeatmapMonthLabel `json:"monthLabels"` // month markers
	Totals      struct {
		Tasks int     `json:"tasks"`
		Cost  float64 `json:"cost"`
	} `json:"totals"`
	DateRange struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"dateRange"`
}

// ============================================================================
// Topology API Types
// ============================================================================

// TopologyAgent represents an agent in the topology graph
type TopologyAgent struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Status     string  `json:"status"` // idle, busy, blocked, error
	TrustScore int     `json:"trustScore"`
	TaskCount  int     `json:"taskCount"`
	Cost       float64 `json:"cost"`
}

// TopologyEdge represents a connection between agents
type TopologyEdge struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	MessageCount int    `json:"messageCount"`
	LastActivity string `json:"lastActivity,omitempty"`
}

// TopologySink represents a terminal node (approval, main branch)
type TopologySink struct {
	ID           string `json:"id"`
	Label        string `json:"label,omitempty"`
	PendingCount int    `json:"pendingCount,omitempty"`
}

// TopologyResponse is the response for GET /api/controlplane/topology
type TopologyResponse struct {
	Agents []TopologyAgent `json:"agents"`
	Edges  []TopologyEdge  `json:"edges"`
	Sinks  []TopologySink  `json:"sinks"`
}

// ============================================================================
// Handlers
// ============================================================================

// GET /api/controlplane/heatmap - Get daily activity data for heatmap visualization
// Uses Observatory spans for canonical telemetry data with filter support.
// Query params:
//   - days: Number of days to include (default: 90, max: 365)
//   - source_type: Filter by source (eval, coordinator, direct_api, local, other)
//   - provider: Filter by provider (claude, gemini, openai)
//   - model: Filter by model name
//   - start_date: Filter start date (YYYY-MM-DD)
//   - end_date: Filter end date (YYYY-MM-DD)
func (s *Server) handleControlPlaneHeatmap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse days parameter (default: 90)
	days := 90
	if daysParam := r.URL.Query().Get("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	// Parse filters
	filter := parseControlPlaneFilter(r)

	// Initialize response
	var cells []HeatmapCell
	var totalTasks int
	var totalCost float64

	// Try Observatory backend first (canonical source)
	if s.obsBackend != nil {
		if sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend); ok {
			points, err := sqliteBackend.GetFilteredHeatmapData(ctx, filter, days)
			if err != nil {
				log.Printf("Failed to get observatory heatmap data: %v", err)
			} else {
				// Convert observatory data points to heatmap cells
				for _, point := range points {
					cells = append(cells, HeatmapCell{
						Date:        point.Date,
						TaskCount:   point.SpanCount, // Use span count as activity indicator
						Cost:        point.Cost,
						SuccessRate: point.SuccessRate,
					})
					totalTasks += point.SpanCount
					totalCost += point.Cost
				}
			}
		}
	}

	// If no observatory data, fall back to coordinator store (unfiltered, coordinator only)
	if len(cells) == 0 {
		now := time.Now()
		startDate := now.AddDate(0, 0, -days)

		coordStore := s.getCoordStoreForControlPlane()
		if coordStore != nil {
			coordFilter := &coordinator.TaskFilter{
				Since:     &startDate,
				OrderBy:   "created_at",
				OrderDesc: false,
			}

			tasks, err := coordStore.ListTasks(ctx, coordFilter)
			if err != nil {
				log.Printf("Failed to get tasks for heatmap: %v", err)
			} else {
				// Group tasks by date
				dateMap := make(map[string]struct {
					count     int
					cost      float64
					completed int
				})

				for _, task := range tasks {
					dateStr := task.CreatedAt.Format("2006-01-02")
					entry := dateMap[dateStr]
					entry.count++
					entry.cost += task.Cost
					if task.Status == coordinator.TaskStatusCompleted {
						entry.completed++
					}
					dateMap[dateStr] = entry
					totalTasks++
					totalCost += task.Cost
				}

				// Generate cells for all days in range
				for d := startDate; !d.After(now); d = d.AddDate(0, 0, 1) {
					dateStr := d.Format("2006-01-02")
					entry := dateMap[dateStr]
					successRate := 0.0
					if entry.count > 0 {
						successRate = float64(entry.completed) / float64(entry.count)
					}
					cells = append(cells, HeatmapCell{
						Date:        dateStr,
						TaskCount:   entry.count,
						Cost:        entry.cost,
						SuccessRate: successRate,
					})
				}
			}
		} else {
			// No data source - return empty data for all days
			for d := startDate; !d.After(now); d = d.AddDate(0, 0, 1) {
				cells = append(cells, HeatmapCell{
					Date:        d.Format("2006-01-02"),
					TaskCount:   0,
					Cost:        0,
					SuccessRate: 0,
				})
			}
		}
	}

	// Check format parameter - grid is now the default
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "grid" // Changed default per plan
	}

	if format == "grid" {
		gridResponse := buildHeatmapGrid(cells, totalTasks, totalCost, days)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(gridResponse); err != nil {
			log.Printf("Failed to encode heatmap grid response: %v", err)
		}
		return
	}

	// Legacy flat format
	response := HeatmapResponse{
		Cells: cells,
	}
	response.Totals.Tasks = totalTasks
	response.Totals.Cost = totalCost

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode heatmap response: %v", err)
	}
}

// buildHeatmapGrid builds a week-by-week grid structure from flat cells
func buildHeatmapGrid(cells []HeatmapCell, totalTasks int, totalCost float64, days int) HeatmapGridResponse {
	now := time.Now()
	endDate := now
	startDate := now.AddDate(0, 0, -days)

	// Build a map for O(1) lookup
	cellMap := make(map[string]HeatmapCell)
	maxCount := 0
	for _, cell := range cells {
		cellMap[cell.Date] = cell
		if cell.TaskCount > maxCount {
			maxCount = cell.TaskCount
		}
	}

	// Align to Monday start
	for startDate.Weekday() != time.Monday {
		startDate = startDate.AddDate(0, 0, -1)
	}

	// Build weeks array
	var weeks [][]HeatmapGridCell
	var monthLabels []HeatmapMonthLabel
	lastMonth := -1

	for d := startDate; !d.After(endDate); {
		week := make([]HeatmapGridCell, 7)
		weekIndex := len(weeks)

		for i := 0; i < 7; i++ {
			dateStr := d.Format("2006-01-02")
			cell := cellMap[dateStr]

			// Calculate intensity (0-1) for coloring
			intensity := 0.0
			if maxCount > 0 && cell.TaskCount > 0 {
				intensity = float64(cell.TaskCount) / float64(maxCount)
			}

			week[i] = HeatmapGridCell{
				Date:        dateStr,
				TaskCount:   cell.TaskCount,
				Cost:        cell.Cost,
				SuccessRate: cell.SuccessRate,
				Intensity:   intensity,
				DayOfWeek:   int(d.Weekday()),
			}

			// Track month labels
			month := int(d.Month())
			if month != lastMonth && d.Day() <= 7 {
				monthLabels = append(monthLabels, HeatmapMonthLabel{
					Name:      d.Format("Jan"),
					WeekIndex: weekIndex,
				})
				lastMonth = month
			}

			d = d.AddDate(0, 0, 1)
		}
		weeks = append(weeks, week)
	}

	response := HeatmapGridResponse{
		Weeks:       weeks,
		MonthLabels: monthLabels,
	}
	response.Totals.Tasks = totalTasks
	response.Totals.Cost = totalCost
	response.DateRange.Start = startDate.Format("2006-01-02")
	response.DateRange.End = endDate.Format("2006-01-02")

	return response
}

// GET /api/controlplane/topology - Get agent topology graph
func (s *Server) handleControlPlaneTopology(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load agent configuration
	cfg, err := coordinator.LoadCoordinatorConfig()
	if err != nil {
		log.Printf("Failed to load coordinator config: %v", err)
		cfg = coordinator.DefaultCoordinatorConfig()
	}

	// Get coordinator store for queries
	coordStore := s.getCoordStoreForControlPlane()

	// Get running tasks to determine agent status
	runningAgents := make(map[string]bool)
	if coordStore != nil {
		ctx := context.Background()
		filter := &coordinator.TaskFilter{
			Status: []coordinator.TaskStatus{coordinator.TaskStatusRunning, coordinator.TaskStatusQueued},
		}
		tasks, err := coordStore.ListTasks(ctx, filter)
		if err == nil {
			for _, task := range tasks {
				if task.AgentID != "" {
					runningAgents[task.AgentID] = true
				}
			}
		}
	}

	// Get task stats per agent
	agentStats := make(map[string]struct {
		taskCount int
		cost      float64
	})
	if coordStore != nil {
		ctx := context.Background()
		tasks, err := coordStore.ListTasks(ctx, &coordinator.TaskFilter{})
		if err == nil {
			for _, task := range tasks {
				if task.AgentID != "" {
					stats := agentStats[task.AgentID]
					stats.taskCount++
					stats.cost += task.Cost
					agentStats[task.AgentID] = stats
				}
			}
		}
	}

	// Get pending approvals count
	pendingApprovals := 0
	if coordStore != nil {
		ctx := context.Background()
		stats, err := coordStore.GetTaskStats(ctx)
		if err == nil {
			pendingApprovals = stats.PendingApprovals
		}
	}

	// Build topology response
	var agents []TopologyAgent
	var edges []TopologyEdge
	edgeSet := make(map[string]bool) // Track unique edges

	for _, agentCfg := range cfg.Agents {
		// Determine status
		status := "idle"
		if runningAgents[agentCfg.ID] {
			status = "busy"
		}

		// Get stats for this agent
		stats := agentStats[agentCfg.ID]

		// Default trust score (placeholder until trust system is implemented)
		trustScore := 75

		agents = append(agents, TopologyAgent{
			ID:         agentCfg.ID,
			Label:      agentCfg.Label,
			Status:     status,
			TrustScore: trustScore,
			TaskCount:  stats.taskCount,
			Cost:       stats.cost,
		})

		// Build edges from trigger_on_complete
		for _, targetID := range agentCfg.TriggerOnComplete {
			edgeKey := agentCfg.ID + "->" + targetID
			if !edgeSet[edgeKey] {
				edges = append(edges, TopologyEdge{
					Source:       agentCfg.ID,
					Target:       targetID,
					MessageCount: 0, // TODO: Count handoff messages
				})
				edgeSet[edgeKey] = true
			}
		}
	}

	// Add source node (GitHub)
	// Note: This is a fixed source for the AILANG workflow
	// Edge from github to first agent(s) that have no incoming edges
	hasIncomingEdge := make(map[string]bool)
	for _, edge := range edges {
		hasIncomingEdge[edge.Target] = true
	}

	for _, agent := range agents {
		if !hasIncomingEdge[agent.ID] {
			edges = append(edges, TopologyEdge{
				Source:       "github",
				Target:       agent.ID,
				MessageCount: 0,
			})
		}
	}

	// Add sink nodes
	sinks := []TopologySink{
		{
			ID:           "approval",
			Label:        "Approval Queue",
			PendingCount: pendingApprovals,
		},
		{
			ID:    "main",
			Label: "main branch",
		},
	}

	// Add edges to approval sink from agents with no outgoing edges
	hasOutgoingEdge := make(map[string]bool)
	for _, edge := range edges {
		hasOutgoingEdge[edge.Source] = true
	}

	for _, agent := range agents {
		if !hasOutgoingEdge[agent.ID] {
			edges = append(edges, TopologyEdge{
				Source:       agent.ID,
				Target:       "approval",
				MessageCount: 0,
			})
		}
	}

	// Add edge from approval to main
	edges = append(edges, TopologyEdge{
		Source:       "approval",
		Target:       "main",
		MessageCount: 0,
	})

	response := TopologyResponse{
		Agents: agents,
		Edges:  edges,
		Sinks:  sinks,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode topology response: %v", err)
	}
}

// Helper to get the full coordinator store interface
func (s *Server) getCoordStoreForControlPlane() coordinator.Store {
	return s.coordStoreRaw
}

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

	// Get Observatory metrics (canonical source of truth)
	if s.obsBackend != nil {
		// Use filtered metrics if SQLite backend with filter support
		var metrics *observatory.MetricsSummary
		var err error

		if sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend); ok && !filter.IsEmpty() {
			metrics, err = sqliteBackend.GetFilteredMetricsSummary(ctx, filter)
		} else {
			metrics, err = s.obsBackend.GetMetricsSummary(ctx)
		}

		if err != nil {
			log.Printf("Failed to get observatory metrics: %v", err)
		} else if metrics != nil {
			response.Observatory = &ObservatoryStats{
				TotalSpans:      metrics.TotalSpans,
				TotalTasks:      metrics.TotalTasks,
				TotalWorkspaces: metrics.TotalWorkspaces,
				TotalAgents:     metrics.TotalAgents,
				TotalTokensIn:   metrics.TotalTokensIn,
				TotalTokensOut:  metrics.TotalTokensOut,
				TotalCostUSD:    metrics.TotalCostUSD,
				SuccessRate:     metrics.SuccessRate,
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
	Percentage float64 `json:"percentage,omitempty"` // Percentage of total cost
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

	// Get total cost (filtered if filter is set)
	var metrics *observatory.MetricsSummary
	var err error
	if !filter.IsEmpty() {
		metrics, err = sqliteBackend.GetFilteredMetricsSummary(ctx, filter)
	} else {
		metrics, err = s.obsBackend.GetMetricsSummary(ctx)
	}
	if err == nil && metrics != nil {
		response.TotalCost = metrics.TotalCostUSD
	}

	// Get breakdowns (filtered if filter is set)
	if !filter.IsEmpty() {
		if items, err := sqliteBackend.GetFilteredBreakdownByProvider(ctx, filter); err == nil {
			response.ByProvider = convertBreakdownItems(items)
		}
		if items, err := sqliteBackend.GetFilteredBreakdownBySourceType(ctx, filter); err == nil {
			response.BySourceType = convertBreakdownItems(items)
		}
		if items, err := sqliteBackend.GetFilteredBreakdownByModel(ctx, filter); err == nil {
			response.ByModel = convertBreakdownItems(items)
		}
		// Workspace breakdown also respects filters (shows workspaces matching current filter)
		if items, err := sqliteBackend.GetFilteredBreakdownByWorkspace(ctx, filter); err == nil {
			response.ByWorkspace = convertBreakdownItems(items)
		}
	} else {
		// No filter - use original methods
		if items, err := sqliteBackend.GetBreakdownByProvider(ctx); err == nil {
			response.ByProvider = convertBreakdownItems(items)
		}
		if items, err := sqliteBackend.GetBreakdownBySourceType(ctx); err == nil {
			response.BySourceType = convertBreakdownItems(items)
		}
		if items, err := sqliteBackend.GetBreakdownByModel(ctx); err == nil {
			response.ByModel = convertBreakdownItems(items)
		}
		if items, err := sqliteBackend.GetFilteredBreakdownByWorkspace(ctx, filter); err == nil {
			response.ByWorkspace = convertBreakdownItems(items)
		}
	}

	// Merge inbox message counts into source type breakdown
	// This ensures GitHub (inbox-only) appears in the sidebar
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
			ID:        item.ID,
			Label:     item.Label,
			SpanCount: item.SpanCount,
			TaskCount: item.TaskCount,
			TokensIn:  item.TokensIn,
			TokensOut: item.TokensOut,
			CostUSD:   item.CostUSD,
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
			// TaskCount is used to track message count for inbox-only sources
			breakdown[idx].TaskCount += inbox.MessageCount
		} else {
			// Source only exists in inbox (e.g., GitHub) - add new entry
			breakdown = append(breakdown, BreakdownItem{
				ID:        inbox.ID,
				Label:     inbox.Label,
				SpanCount: 0,
				TaskCount: inbox.MessageCount, // Use TaskCount for message count
				TokensIn:  0,
				TokensOut: 0,
				CostUSD:   0, // Inbox messages have no cost
			})
		}
	}

	return breakdown
}

// ============================================================================
// Observed Topology API - Data-Driven Graph
// ============================================================================

// ObservedTopologyNode represents a node in the observed topology
type ObservedTopologyNode struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	NodeType     string `json:"node_type"` // agent, source, sink
	MessagesSent int    `json:"messages_sent"`
	MessagesRecv int    `json:"messages_recv"`
	LastActivity string `json:"last_activity,omitempty"`
}

// ObservedTopologyEdge represents an edge derived from actual message flows
type ObservedTopologyEdge struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	MessageCount int    `json:"message_count"`
	LastActivity string `json:"last_activity,omitempty"`
	Active       bool   `json:"active"`
}

// ObservedTopologyResponse is the response for GET /api/controlplane/topology/observed
type ObservedTopologyResponse struct {
	Nodes   []ObservedTopologyNode `json:"nodes"`
	Edges   []ObservedTopologyEdge `json:"edges"`
	IsEmpty bool                   `json:"is_empty"`
}

// GET /api/controlplane/topology/observed - Get topology derived from actual message flows
// This returns a data-driven graph based on from_agent → to_inbox message relationships
func (s *Server) handleControlPlaneTopologyObserved(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := ObservedTopologyResponse{
		Nodes: []ObservedTopologyNode{},
		Edges: []ObservedTopologyEdge{},
	}

	// Get message flow edges from the messaging store
	edges, err := s.store.GetMessageFlowEdges()
	if err != nil {
		log.Printf("Failed to get message flow edges: %v", err)
	}

	// Get active agents from the messaging store
	agents, err := s.store.GetActiveAgents()
	if err != nil {
		log.Printf("Failed to get active agents: %v", err)
	}

	// Build nodes from active agents
	nodeMap := make(map[string]bool)
	for _, agent := range agents {
		nodeMap[agent.ID] = true
		response.Nodes = append(response.Nodes, ObservedTopologyNode{
			ID:           agent.ID,
			Label:        formatAgentLabel(agent.ID),
			NodeType:     "agent",
			MessagesSent: agent.MessagesSent,
			MessagesRecv: agent.MessagesRecv,
			LastActivity: agent.LastActivity,
		})
	}

	// Build edges and ensure all nodes exist
	for _, edge := range edges {
		// Add source node if not already present
		if !nodeMap[edge.FromAgent] {
			nodeMap[edge.FromAgent] = true
			response.Nodes = append(response.Nodes, ObservedTopologyNode{
				ID:       edge.FromAgent,
				Label:    formatAgentLabel(edge.FromAgent),
				NodeType: "agent",
			})
		}

		// Add target node if not already present
		if !nodeMap[edge.ToInbox] {
			nodeMap[edge.ToInbox] = true
			response.Nodes = append(response.Nodes, ObservedTopologyNode{
				ID:       edge.ToInbox,
				Label:    formatAgentLabel(edge.ToInbox),
				NodeType: "agent",
			})
		}

		// Determine if edge is active (activity in last 5 minutes)
		active := false
		if edge.LastActivity != "" {
			if t, err := time.Parse(time.RFC3339, edge.LastActivity); err == nil {
				active = time.Since(t) < 5*time.Minute
			}
		}

		response.Edges = append(response.Edges, ObservedTopologyEdge{
			Source:       edge.FromAgent,
			Target:       edge.ToInbox,
			MessageCount: edge.MessageCount,
			LastActivity: edge.LastActivity,
			Active:       active,
		})
	}

	// Detect node types based on edge topology
	hasIncoming := make(map[string]bool)
	hasOutgoing := make(map[string]bool)
	for _, edge := range response.Edges {
		hasIncoming[edge.Target] = true
		hasOutgoing[edge.Source] = true
	}

	// Update node types: sources have no incoming, sinks have no outgoing
	for i := range response.Nodes {
		nodeID := response.Nodes[i].ID
		if !hasIncoming[nodeID] && hasOutgoing[nodeID] {
			response.Nodes[i].NodeType = "source"
		} else if hasIncoming[nodeID] && !hasOutgoing[nodeID] {
			response.Nodes[i].NodeType = "sink"
		}
	}

	response.IsEmpty = len(response.Nodes) == 0

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode observed topology response: %v", err)
	}
}

// formatAgentLabel converts an agent ID to a human-readable label
func formatAgentLabel(agentID string) string {
	// Handle special cases
	switch agentID {
	case "github":
		return "GitHub Issues"
	case "approval":
		return "Approval Queue"
	case "main":
		return "Main Branch"
	case "user":
		return "User"
	}

	// Convert kebab-case to Title Case
	parts := strings.Split(agentID, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

// GET /api/controlplane/exec-hierarchy - Get exec task hierarchy from span attributes
// Returns tree structure of ailang exec tasks with parent/child relationships
// Query params:
//   - limit: Maximum number of exec spans to query (default: 100)
//   - include_messages: If "true", groups execs by triggering messages (4-level hierarchy)
func (s *Server) handleControlPlaneExecHierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse limit parameter
	limit := 100
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	// Parse include_messages parameter
	includeMessages := r.URL.Query().Get("include_messages") == "true"

	// Get SQLite backend for direct store access
	sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend)
	if !ok {
		http.Error(w, "Exec hierarchy requires SQLite backend", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if includeMessages {
		// Return hierarchy grouped by messages (4-level: Message -> Exec -> Turn -> Tool)
		result, err := sqliteBackend.Store().GetExecTaskHierarchyWithMessages(limit)
		if err != nil {
			log.Printf("Failed to get exec hierarchy with messages: %v", err)
			http.Error(w, "Failed to get exec hierarchy", http.StatusInternalServerError)
			return
		}
		// Enrich all exec hierarchies within messages
		for _, msg := range result.Messages {
			enrichExecHierarchy(r.Context(), sqliteBackend.Store(), msg.Execs)
		}
		enrichExecHierarchy(r.Context(), sqliteBackend.Store(), result.Orphan)
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("Failed to encode exec hierarchy response: %v", err)
		}
		return
	}

	// Return flat hierarchy (backward compatible)
	hierarchy, err := sqliteBackend.Store().GetExecTaskHierarchy(limit)
	if err != nil {
		log.Printf("Failed to get exec hierarchy: %v", err)
		http.Error(w, "Failed to get exec hierarchy", http.StatusInternalServerError)
		return
	}

	// Enrich hierarchy with display names from session_tools
	enrichExecHierarchy(r.Context(), sqliteBackend.Store(), hierarchy)

	response := struct {
		Hierarchy []*observatory.ExecTaskNode `json:"hierarchy"`
		Count     int                         `json:"count"`
	}{
		Hierarchy: hierarchy,
		Count:     len(hierarchy),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode exec hierarchy response: %v", err)
	}
}

// enrichExecHierarchy adds display_name to tool_use nodes using session_tools data.
// This correlates OTEL spans with hook-captured tool metadata for richer display.
func enrichExecHierarchy(ctx context.Context, store *observatory.Store, hierarchy []*observatory.ExecTaskNode) {
	if len(hierarchy) == 0 {
		return
	}

	// Collect all tool_use nodes with their timestamps
	var toolNodes []*observatory.ExecTaskNode
	var minTime, maxTime time.Time

	var collect func(nodes []*observatory.ExecTaskNode)
	collect = func(nodes []*observatory.ExecTaskNode) {
		for _, node := range nodes {
			if node.Command == "tool_use" && node.StartTime != nil {
				toolNodes = append(toolNodes, node)
				if minTime.IsZero() || node.StartTime.Before(minTime) {
					minTime = *node.StartTime
				}
				endTime := node.StartTime.Add(time.Duration(node.DurationMs) * time.Millisecond)
				if maxTime.IsZero() || endTime.After(maxTime) {
					maxTime = endTime
				}
			}
			if len(node.Children) > 0 {
				collect(node.Children)
			}
		}
	}
	collect(hierarchy)

	if len(toolNodes) == 0 {
		return
	}

	// Expand time window for matching
	minTime = minTime.Add(-30 * time.Second)
	maxTime = maxTime.Add(30 * time.Second)

	// Fetch session tools in range
	tools, err := store.GetToolsByTimestampRange(ctx, minTime, maxTime, "")
	if err != nil || len(tools) == 0 {
		return
	}

	// Build lookup map by tool name + approximate timestamp
	const tolerance = 10 * time.Second

	// Enrich each tool node
	for _, node := range toolNodes {
		if node.ToolName == "" || node.DisplayName != "" {
			continue
		}

		// Find matching tool by name and timestamp
		for _, tool := range tools {
			if tool.ToolName != node.ToolName {
				continue
			}

			// Check timestamp match
			toolStart := tool.StartTime
			nodeStart := *node.StartTime
			diff := toolStart.Sub(nodeStart)
			if diff < 0 {
				diff = -diff
			}
			if diff > tolerance {
				continue
			}

			// Generate display name from tool metadata
			displayName := generateToolDisplayName(tool.ToolName, tool.ToolInput)
			if displayName != "" {
				node.DisplayName = displayName
				break
			}
		}
	}
}

// generateToolDisplayName creates a rich display name from tool name and input.
func generateToolDisplayName(toolName string, toolInput json.RawMessage) string {
	if len(toolInput) == 0 {
		return toolName
	}

	var data map[string]any
	if err := json.Unmarshal(toolInput, &data); err != nil {
		return toolName
	}

	switch toolName {
	case "Read":
		if path, ok := data["file_path"].(string); ok {
			return "Read: " + truncateForDisplay(path, 50)
		}
	case "Edit":
		if path, ok := data["file_path"].(string); ok {
			return "Edit: " + truncateForDisplay(path, 50)
		}
	case "Write":
		if path, ok := data["file_path"].(string); ok {
			return "Write: " + truncateForDisplay(path, 50)
		}
	case "Glob":
		if pattern, ok := data["pattern"].(string); ok {
			return "Glob: " + truncateForDisplay(pattern, 50)
		}
	case "Grep":
		if pattern, ok := data["pattern"].(string); ok {
			return "Grep: " + truncateForDisplay(pattern, 50)
		}
	case "Bash":
		if cmd, ok := data["command"].(string); ok {
			return "Bash: " + truncateForDisplay(cmd, 50)
		}
		if desc, ok := data["description"].(string); ok {
			return "Bash: " + truncateForDisplay(desc, 50)
		}
	case "Task":
		if desc, ok := data["description"].(string); ok && desc != "" {
			return "Task: " + truncateForDisplay(desc, 50)
		}
		if subType, ok := data["subagent_type"].(string); ok {
			return "Task: " + subType
		}
	case "Skill":
		if skill, ok := data["skill"].(string); ok {
			return "Skill: " + skill
		}
	}

	return toolName
}

// truncateForDisplay truncates a string to maxLen characters.
func truncateForDisplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

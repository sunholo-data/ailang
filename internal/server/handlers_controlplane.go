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
		// Use AILANG bridge if enabled, falls back to Go
		gridResponse := GetAILANGBridge().BuildHeatmapGrid(cells, totalTasks, totalCost, days)
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
	DurationMs int64   `json:"duration_ms"`          // Total execution time in ms
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
			ID:         item.ID,
			Label:      item.Label,
			SpanCount:  item.SpanCount,
			TaskCount:  item.TaskCount,
			TokensIn:   item.TokensIn,
			TokensOut:  item.TokensOut,
			CostUSD:    item.CostUSD,
			DurationMs: item.DurationMs,
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

// GET /api/controlplane/span-hierarchy - Get span hierarchy using parent_span_id relationships
// This works with standard OTEL span parenting, not custom attributes.
// Query params:
//   - limit: Maximum number of root spans to query (default: 100)
func (s *Server) handleSpanHierarchy(w http.ResponseWriter, r *http.Request) {
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

	// Get SQLite backend for direct store access
	sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend)
	if !ok {
		http.Error(w, "Span hierarchy requires SQLite backend", http.StatusServiceUnavailable)
		return
	}

	result, err := sqliteBackend.Store().GetSpanHierarchy(limit)
	if err != nil {
		log.Printf("Failed to get span hierarchy: %v", err)
		http.Error(w, "Failed to get span hierarchy", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Failed to encode span hierarchy response: %v", err)
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

// TaskHierarchyNode represents a task with its relationships for cross-task visualization.
type TaskHierarchyNode struct {
	ID             string               `json:"id"`
	Title          string               `json:"title"`
	AgentID        string               `json:"agent_id,omitempty"`
	ParentTaskID   string               `json:"parent_task_id,omitempty"`
	SessionID      string               `json:"session_id,omitempty"`
	Status         string               `json:"status"`
	ApprovalStatus string               `json:"approval_status,omitempty"` // "pending", "approved", "rejected", ""
	ApprovalType   string               `json:"approval_type,omitempty"`   // "merge", "merge_handoff", etc.
	Iteration      int                  `json:"iteration,omitempty"`
	Cost           float64              `json:"cost"`
	TokensIn       int                  `json:"tokens_in"`
	TokensOut      int                  `json:"tokens_out"`
	Turns          int                  `json:"turns,omitempty"`
	DurationMs     int64                `json:"duration_ms"`
	CreatedAt      time.Time            `json:"created_at"`
	Provider       string               `json:"provider,omitempty"`
	Workspace      string               `json:"workspace,omitempty"`
	Children       []*TaskHierarchyNode `json:"children,omitempty"` // Child tasks (via parent_task_id)
	// Execution spans nested within this task (from observatory.db)
	Spans []*TaskSpanNode `json:"spans,omitempty"`
	// Turn-grouped hierarchy (when group_by=turns is requested)
	TurnGrouped *observatory.TurnGroupedHierarchy `json:"turn_grouped,omitempty"`
}

// TaskSpanNode represents a span within a task for the unified task hierarchy view.
// Simplified version of SpanHierarchyNode for API response.
type TaskSpanNode struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	NodeType   string          `json:"node_type"` // coordinator, executor, turn, tool, other
	DurationMs int64           `json:"duration_ms"`
	TokensIn   int64           `json:"tokens_in,omitempty"`
	TokensOut  int64           `json:"tokens_out,omitempty"`
	CostUSD    float64         `json:"cost_usd,omitempty"`
	TurnNumber int             `json:"turn_number,omitempty"`
	ToolName   string          `json:"tool_name,omitempty"`
	Status     string          `json:"status"`
	Children   []*TaskSpanNode `json:"children,omitempty"`
}

// TaskHierarchyEdge represents a relationship between tasks.
type TaskHierarchyEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // "handoff" (parent_task_id) or "session" (shared session_id)
}

// TaskHierarchyResult contains the full cross-task hierarchy.
type TaskHierarchyResult struct {
	Tasks []*TaskHierarchyNode `json:"tasks"`
	Edges []TaskHierarchyEdge  `json:"edges"`
	Stats struct {
		TotalTasks       int     `json:"total_tasks"`
		TotalSpans       int     `json:"total_spans"`
		PendingApprovals int     `json:"pending_approvals"`
		TotalCost        float64 `json:"total_cost"`
	} `json:"stats"`
}

// GET /api/controlplane/task-hierarchy - Get cross-task hierarchy with relationships
// Returns tasks with parent_task_id chains, session continuity, and approval status.
// Query params:
//   - limit: Maximum number of tasks (default: 50)
//   - status: Filter by status (optional, comma-separated)
//   - workspace: Filter by workspace path (optional)
//   - provider: Filter by provider (optional)
//   - task_id: Filter to specific task and its handoff chain (optional)
//   - trace_id: Filter to tasks with spans in this trace (optional)
//   - group_by: "turns" to group spans by conversation turn (Session → Turn 1 → Turn 2 → ...)
func (s *Server) handleTaskHierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get coordinator store
	coordStore := s.getCoordStoreForControlPlane()
	if coordStore == nil {
		http.Error(w, "Coordinator store not available", http.StatusServiceUnavailable)
		return
	}

	// Parse limit parameter
	limit := 50
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	// Parse status filter
	var statusFilter []coordinator.TaskStatus
	if statusParam := r.URL.Query().Get("status"); statusParam != "" {
		for _, s := range strings.Split(statusParam, ",") {
			statusFilter = append(statusFilter, coordinator.TaskStatus(strings.TrimSpace(s)))
		}
	}

	// Parse workspace filter
	workspace := r.URL.Query().Get("workspace")

	// Parse provider filter
	provider := r.URL.Query().Get("provider")

	// Parse task_id filter (filter to specific task and its chain)
	filterTaskID := r.URL.Query().Get("task_id")

	// Parse task_ids filter (filter to multiple specific task IDs, comma-separated)
	var filterTaskIDs []string
	if taskIDsParam := r.URL.Query().Get("task_ids"); taskIDsParam != "" {
		for _, id := range strings.Split(taskIDsParam, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				filterTaskIDs = append(filterTaskIDs, trimmed)
			}
		}
	}

	// Parse trace_id filter (filter to tasks with spans in this trace)
	filterTraceID := r.URL.Query().Get("trace_id")

	// Parse group_by parameter (turns = group spans by conversation turn)
	groupBy := r.URL.Query().Get("group_by")

	// If trace_id filter is provided, find task_ids from spans first
	var traceTaskIDs map[string]bool
	if filterTraceID != "" && s.obsBackend != nil {
		spans, err := s.obsBackend.ListSpans(ctx, observatory.SpanListOptions{
			TraceID: filterTraceID,
			Limit:   1000,
		})
		if err != nil {
			log.Printf("Failed to query spans for trace %s: %v", filterTraceID, err)
		} else {
			traceTaskIDs = make(map[string]bool)
			for _, span := range spans {
				if span.TaskID != "" {
					traceTaskIDs[span.TaskID] = true
				}
			}
		}
	}

	// Fetch tasks
	filter := &coordinator.TaskFilter{
		Limit:     limit,
		OrderBy:   "created_at",
		OrderDesc: true,
		Workspace: workspace,
		Provider:  provider,
	}
	if len(statusFilter) > 0 {
		filter.Status = statusFilter
	}

	tasks, err := coordStore.ListTasks(ctx, filter)
	if err != nil {
		log.Printf("Failed to list tasks for hierarchy: %v", err)
		http.Error(w, "Failed to fetch tasks", http.StatusInternalServerError)
		return
	}

	// Filter to specific task and its handoff chain if task_id is provided
	if filterTaskID != "" {
		// Build task lookup for filtering
		allTasksMap := make(map[string]*coordinator.TaskRecord)
		for _, t := range tasks {
			allTasksMap[t.ID] = t
		}

		// Collect task IDs that should be included (the task + its chain)
		includeIDs := make(map[string]bool)

		// Include the requested task
		includeIDs[filterTaskID] = true

		// Walk up parent chain
		currentID := filterTaskID
		for {
			if t, ok := allTasksMap[currentID]; ok && t.ParentTaskID != "" {
				includeIDs[t.ParentTaskID] = true
				currentID = t.ParentTaskID
			} else {
				break
			}
		}

		// Include child tasks (tasks where parent_task_id = any included task)
		changed := true
		for changed {
			changed = false
			for _, t := range tasks {
				if t.ParentTaskID != "" && includeIDs[t.ParentTaskID] && !includeIDs[t.ID] {
					includeIDs[t.ID] = true
					changed = true
				}
			}
		}

		// Filter tasks list
		var filteredTasks []*coordinator.TaskRecord
		for _, t := range tasks {
			if includeIDs[t.ID] {
				filteredTasks = append(filteredTasks, t)
			}
		}
		tasks = filteredTasks
	}

	// Filter by trace_id if provided (tasks with spans in that trace)
	if traceTaskIDs != nil {
		var filteredTasks []*coordinator.TaskRecord
		for _, t := range tasks {
			if traceTaskIDs[t.ID] {
				filteredTasks = append(filteredTasks, t)
			}
		}
		tasks = filteredTasks
	}

	// Filter by task_ids if provided (multiple specific task IDs)
	if len(filterTaskIDs) > 0 {
		taskIDSet := make(map[string]bool)
		for _, id := range filterTaskIDs {
			taskIDSet[id] = true
		}
		var filteredTasks []*coordinator.TaskRecord
		for _, t := range tasks {
			if taskIDSet[t.ID] {
				filteredTasks = append(filteredTasks, t)
			}
		}
		tasks = filteredTasks
	}

	// Fetch pending approvals
	pendingApprovals, err := coordStore.ListPendingApprovals(ctx)
	if err != nil {
		log.Printf("Failed to list pending approvals: %v", err)
		// Continue without approvals
		pendingApprovals = nil
	}

	// Build approval lookup map: task_id -> approval
	approvalMap := make(map[string]*coordinator.ApprovalRequestRecord)
	for _, apr := range pendingApprovals {
		approvalMap[apr.TaskID] = apr
	}

	// Build result (initialize slices to avoid null in JSON)
	result := TaskHierarchyResult{
		Tasks: []*TaskHierarchyNode{},
		Edges: []TaskHierarchyEdge{},
	}
	taskMap := make(map[string]*TaskHierarchyNode)
	sessionTasks := make(map[string][]string) // session_id -> task_ids

	for _, task := range tasks {
		node := &TaskHierarchyNode{
			ID:           task.ID,
			Title:        task.Title,
			AgentID:      task.AgentID,
			ParentTaskID: task.ParentTaskID,
			SessionID:    task.SessionID,
			Status:       string(task.Status),
			Cost:         task.Cost,
			TokensIn:     task.InputTokens,
			TokensOut:    task.OutputTokens,
			CreatedAt:    task.CreatedAt,
			Provider:     task.Provider,
			Workspace:    task.Workspace,
			Iteration:    task.Iteration,
		}

		// Calculate duration
		if task.StartedAt != nil && task.CompletedAt != nil {
			node.DurationMs = task.CompletedAt.Sub(*task.StartedAt).Milliseconds()
		} else if task.Duration > 0 {
			node.DurationMs = task.Duration.Milliseconds()
		}

		// Add approval status if exists
		if apr, ok := approvalMap[task.ID]; ok {
			node.ApprovalStatus = apr.Status
			node.ApprovalType = apr.Type
		} else if task.Status == coordinator.TaskStatusPendingApproval {
			node.ApprovalStatus = "pending"
		}

		taskMap[task.ID] = node
		result.Tasks = append(result.Tasks, node)

		// Track session relationships
		if task.SessionID != "" {
			sessionTasks[task.SessionID] = append(sessionTasks[task.SessionID], task.ID)
		}

		// Update stats
		result.Stats.TotalTasks++
		result.Stats.TotalCost += task.Cost
		if node.ApprovalStatus == "pending" {
			result.Stats.PendingApprovals++
		}
	}

	// Fetch spans from observatory.db for each task
	if s.obsBackend != nil {
		for _, node := range result.Tasks {
			spans, err := s.obsBackend.ListSpans(ctx, observatory.SpanListOptions{
				TaskID: node.ID,
				Limit:  500, // Limit spans per task
			})
			if err != nil {
				log.Printf("Failed to fetch spans for task %s: %v", node.ID, err)
				continue
			}
			if len(spans) > 0 {
				// Build span hierarchy from flat list
				node.Spans = buildSpanHierarchyForTask(spans)
				result.Stats.TotalSpans += len(spans)

				// Apply turn grouping if requested
				if groupBy == "turns" {
					spanNodes := buildSpanNodeTreeFromFlat(spans)
					node.TurnGrouped = observatory.GroupSpansByTurn(spanNodes)
				}
			}
		}
	}

	// Build edges (only include edges where BOTH source and target exist in result)
	for _, task := range result.Tasks {
		// Handoff edges (parent_task_id)
		if task.ParentTaskID != "" {
			// Only add edge if parent task exists in filtered result
			if _, parentExists := taskMap[task.ParentTaskID]; parentExists {
				result.Edges = append(result.Edges, TaskHierarchyEdge{
					Source: task.ParentTaskID,
					Target: task.ID,
					Type:   "handoff",
				})
			}
		}
	}

	// Session edges (shared session_id - only add between tasks that exist)
	for sessionID, taskIDs := range sessionTasks {
		if len(taskIDs) > 1 && sessionID != "" {
			// Both tasks must exist in the result
			if _, ok1 := taskMap[taskIDs[0]]; ok1 {
				if _, ok2 := taskMap[taskIDs[1]]; ok2 {
					result.Edges = append(result.Edges, TaskHierarchyEdge{
						Source: taskIDs[0],
						Target: taskIDs[1],
						Type:   "session",
					})
				}
			}
		}
	}

	// Build children arrays for tree structure
	for _, task := range result.Tasks {
		if task.ParentTaskID != "" {
			if parent, ok := taskMap[task.ParentTaskID]; ok {
				parent.Children = append(parent.Children, task)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Failed to encode task hierarchy response: %v", err)
	}
}

// buildSpanHierarchyForTask converts a flat list of spans into a tree structure
// using parent_span_id relationships. Returns root spans with nested children.
func buildSpanHierarchyForTask(spans []*observatory.Span) []*TaskSpanNode {
	if len(spans) == 0 {
		return nil
	}

	// Convert spans to TaskSpanNode and build lookup map
	nodeMap := make(map[string]*TaskSpanNode)
	for _, span := range spans {
		node := &TaskSpanNode{
			ID:         span.ID,
			Name:       span.Name,
			NodeType:   classifySpanNodeType(span.Name),
			DurationMs: span.DurationMs,
			TokensIn:   span.TokensIn,
			TokensOut:  span.TokensOut,
			CostUSD:    span.CostUSD,
			Status:     string(span.Status),
		}

		// Extract turn number from attributes if present
		if span.Attributes != nil {
			if turnNum, ok := span.Attributes["turn.number"]; ok {
				if tn, ok := turnNum.(float64); ok {
					node.TurnNumber = int(tn)
				}
			}
			if toolName, ok := span.Attributes["tool.name"]; ok {
				if tn, ok := toolName.(string); ok {
					node.ToolName = tn
				}
			}
		}

		nodeMap[span.ID] = node
	}

	// Build parent-child relationships
	var roots []*TaskSpanNode
	for _, span := range spans {
		node := nodeMap[span.ID]
		if span.ParentSpanID != "" {
			if parent, ok := nodeMap[span.ParentSpanID]; ok {
				parent.Children = append(parent.Children, node)
			} else {
				// Parent not in this task's spans - treat as root
				roots = append(roots, node)
			}
		} else {
			roots = append(roots, node)
		}
	}

	return roots
}

// classifySpanNodeType determines the node type based on span name.
func classifySpanNodeType(name string) string {
	switch {
	case strings.HasPrefix(name, "coordinator."):
		return "coordinator"
	case strings.HasPrefix(name, "claude.") || strings.HasPrefix(name, "gemini.") ||
		strings.HasPrefix(name, "openai.") || name == "ailang.exec":
		return "executor"
	case strings.HasPrefix(name, "exec.turn") || strings.HasPrefix(name, "turn."):
		return "turn"
	case strings.HasPrefix(name, "exec.tool_use") || strings.HasPrefix(name, "tool."):
		return "tool"
	default:
		return "other"
	}
}

// buildSpanNodeTreeFromFlat converts a flat list of spans into observatory.SpanNode tree.
// This is used for turn grouping which requires the SpanNode tree structure.
func buildSpanNodeTreeFromFlat(spans []*observatory.Span) []*observatory.SpanNode {
	if len(spans) == 0 {
		return nil
	}

	// Build node map
	nodeMap := make(map[string]*observatory.SpanNode)
	for _, span := range spans {
		nodeMap[span.ID] = &observatory.SpanNode{Span: span}
	}

	// Build parent-child relationships
	var roots []*observatory.SpanNode
	for _, span := range spans {
		node := nodeMap[span.ID]
		if span.ParentSpanID == "" {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[span.ParentSpanID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// Parent not in our set, treat as root
			roots = append(roots, node)
		}
	}

	return roots
}

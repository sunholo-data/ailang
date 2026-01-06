package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
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

// GET /api/controlplane/heatmap - Get daily task activity for heatmap visualization
func (s *Server) handleControlPlaneHeatmap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse days parameter (default: 90)
	days := 90
	if daysParam := r.URL.Query().Get("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	// Calculate date range
	now := time.Now()
	startDate := now.AddDate(0, 0, -days)

	// Get tasks from coordinator store
	var cells []HeatmapCell
	var totalTasks int
	var totalCost float64

	coordStore := s.getCoordStoreForControlPlane()
	if coordStore != nil {
		// Query tasks within date range
		ctx := context.Background()
		filter := &coordinator.TaskFilter{
			Since:     &startDate,
			OrderBy:   "created_at",
			OrderDesc: false,
		}

		tasks, err := coordStore.ListTasks(ctx, filter)
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
		// No coordinator store - return empty data for all days
		for d := startDate; !d.After(now); d = d.AddDate(0, 0, 1) {
			cells = append(cells, HeatmapCell{
				Date:        d.Format("2006-01-02"),
				TaskCount:   0,
				Cost:        0,
				SuccessRate: 0,
			})
		}
	}

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

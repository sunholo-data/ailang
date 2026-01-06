package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/sunholo/ailang/internal/coordinator"
)

// StatisticsResponse represents aggregate statistics for the dashboard
type StatisticsResponse struct {
	Threads     ThreadStatistics    `json:"threads"`
	Coordinator *CoordinatorSummary `json:"coordinator,omitempty"`
}

// ThreadStatistics provides thread-level statistics
type ThreadStatistics struct {
	Total       int            `json:"total"`
	ByStatus    map[string]int `json:"by_status"`
	ByWorkspace map[string]int `json:"by_workspace"`
}

// CoordinatorSummary provides coordinator task statistics
type CoordinatorSummary struct {
	TotalTasks       int            `json:"total_tasks"`
	PendingTasks     int            `json:"pending_tasks"`
	RunningTasks     int            `json:"running_tasks"`
	CompletedTasks   int            `json:"completed_tasks"`
	FailedTasks      int            `json:"failed_tasks"`
	ByProvider       map[string]int `json:"by_provider,omitempty"`
	ByWorkspace      map[string]int `json:"by_workspace,omitempty"`
	TotalCost        float64        `json:"total_cost"`
	TotalTokens      int            `json:"total_tokens"`
	ActiveAgents     int            `json:"active_agents"`     // Count of agents with running tasks
	PendingApprovals int            `json:"pending_approvals"` // Tasks awaiting human approval
	SuccessRate      float64        `json:"success_rate"`      // Completed / (Completed + Failed)
}

// GET /api/statistics - Get aggregate statistics
func (s *Server) handleStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get thread statistics from messaging store
	threadStats, err := s.store.GetThreadAggregateStats()
	if err != nil {
		log.Printf("Failed to get thread stats: %v", err)
		http.Error(w, "Failed to get statistics", http.StatusInternalServerError)
		return
	}

	response := StatisticsResponse{
		Threads: ThreadStatistics{
			Total:       threadStats.TotalThreads,
			ByStatus:    threadStats.ByStatus,
			ByWorkspace: threadStats.ByWorkspace,
		},
	}

	// Try to get coordinator stats if available
	if s.coordStore != nil {
		coordStats, err := s.coordStore.GetCoordinatorStats()
		if err == nil && coordStats != nil {
			// Calculate success rate
			successRate := 0.0
			totalResolved := coordStats.TasksRun + coordStats.FailedTasks
			if totalResolved > 0 {
				successRate = float64(coordStats.TasksRun) / float64(totalResolved)
			}

			// Count active agents (agents with running tasks)
			activeAgents := 0
			pendingApprovals := 0

			// Try to get detailed stats from full store interface
			if store := s.getCoordStoreForStats(); store != nil {
				ctx := r.Context()
				if taskStats, err := store.GetTaskStats(ctx); err == nil {
					pendingApprovals = taskStats.PendingApprovals
					// Count unique agents with running tasks
					if runningTasks, err := store.ListTasks(ctx, &coordinator.TaskFilter{
						Status: []coordinator.TaskStatus{coordinator.TaskStatusRunning},
					}); err == nil {
						agentSet := make(map[string]bool)
						for _, task := range runningTasks {
							if task.AgentID != "" {
								agentSet[task.AgentID] = true
							}
						}
						activeAgents = len(agentSet)
					}
				}
			}

			response.Coordinator = &CoordinatorSummary{
				TotalTasks:       coordStats.TasksRun + coordStats.PendingTasks + coordStats.RunningTasks,
				PendingTasks:     coordStats.PendingTasks,
				RunningTasks:     coordStats.RunningTasks,
				CompletedTasks:   coordStats.TasksRun,
				FailedTasks:      coordStats.FailedTasks,
				TotalCost:        coordStats.TotalCost,
				TotalTokens:      coordStats.TotalTokens,
				ActiveAgents:     activeAgents,
				PendingApprovals: pendingApprovals,
				SuccessRate:      successRate,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode statistics response: %v", err)
	}
}

// Helper to get the full coordinator store interface for detailed stats
func (s *Server) getCoordStoreForStats() coordinator.Store {
	return s.coordStoreRaw
}

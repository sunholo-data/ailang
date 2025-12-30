package server

import (
	"encoding/json"
	"log"
	"net/http"
)

// StatisticsResponse represents aggregate statistics for the dashboard
type StatisticsResponse struct {
	Threads     ThreadStatistics     `json:"threads"`
	Coordinator *CoordinatorSummary  `json:"coordinator,omitempty"`
}

// ThreadStatistics provides thread-level statistics
type ThreadStatistics struct {
	Total       int            `json:"total"`
	ByStatus    map[string]int `json:"by_status"`
	ByWorkspace map[string]int `json:"by_workspace"`
}

// CoordinatorSummary provides coordinator task statistics
type CoordinatorSummary struct {
	TotalTasks     int            `json:"total_tasks"`
	PendingTasks   int            `json:"pending_tasks"`
	RunningTasks   int            `json:"running_tasks"`
	CompletedTasks int            `json:"completed_tasks"`
	FailedTasks    int            `json:"failed_tasks"`
	ByProvider     map[string]int `json:"by_provider,omitempty"`
	ByWorkspace    map[string]int `json:"by_workspace,omitempty"`
	TotalCost      float64        `json:"total_cost"`
	TotalTokens    int            `json:"total_tokens"`
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
			response.Coordinator = &CoordinatorSummary{
				TotalTasks:     coordStats.TasksRun + coordStats.PendingTasks + coordStats.RunningTasks,
				PendingTasks:   coordStats.PendingTasks,
				RunningTasks:   coordStats.RunningTasks,
				CompletedTasks: coordStats.TasksRun,
				FailedTasks:    coordStats.FailedTasks,
				TotalCost:      coordStats.TotalCost,
				TotalTokens:    coordStats.TotalTokens,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode statistics response: %v", err)
	}
}

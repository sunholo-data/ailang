package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "healthy",
		"version":     s.version,
		"connections": s.wsServer.GetConnectionCount(),
		"timestamp":   time.Now().Unix(),
	}); err != nil {
		log.Printf("Failed to encode health response: %v", err)
	}
}

// handleVersion returns the AILANG version
// GET /api/version
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"version": s.version,
	}); err != nil {
		log.Printf("Failed to encode version response: %v", err)
	}
}

// handleHierarchy returns the complete agent/thread hierarchy tree
// GET /api/hierarchy
func (s *Server) handleHierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hierarchy, err := s.store.GetHierarchy()
	if err != nil {
		log.Printf("Failed to get hierarchy: %v", err)
		http.Error(w, "Failed to get hierarchy", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(hierarchy); err != nil {
		log.Printf("Failed to encode hierarchy response: %v", err)
	}
}

// handleAgentStats returns detailed statistics for a single agent
// GET /api/agent-stats/{agentID}
func (s *Server) handleAgentStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/agent-stats/")
	if path == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	stats, err := s.store.GetAgentStats(path)
	if err != nil {
		log.Printf("Failed to get agent stats: %v", err)
		http.Error(w, "Failed to get agent stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Failed to encode agent stats response: %v", err)
	}
}

// handleMetrics returns global aggregated metrics
// GET /api/metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics, err := s.store.GetGlobalMetrics()
	if err != nil {
		log.Printf("Failed to get global metrics: %v", err)
		http.Error(w, "Failed to get metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		log.Printf("Failed to encode metrics response: %v", err)
	}
}

// handleMetricsScope returns metrics for a specific scope (agent or thread)
// GET /api/metrics/agent/{id} - Agent-level metrics
// GET /api/metrics/thread/{id} - Thread-level metrics
// GET /api/metrics/trends/{scope}/{id}?period={minute|hour|day}&limit={n} - Time-series trends
func (s *Server) handleMetricsScope(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/metrics/{scope}/{id} or /api/metrics/trends/{scope}/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/metrics/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		http.Error(w, "Invalid path: expected /api/metrics/{scope}/{id}", http.StatusBadRequest)
		return
	}

	// Handle trends endpoint
	if parts[0] == "trends" {
		if len(parts) < 3 {
			http.Error(w, "Invalid path: expected /api/metrics/trends/{scope}/{id}", http.StatusBadRequest)
			return
		}
		scopeType := parts[1]
		scopeID := parts[2]

		// Parse query params
		period := r.URL.Query().Get("period")
		if period == "" {
			period = "hour"
		}
		limitStr := r.URL.Query().Get("limit")
		limit := 24 // Default to 24 data points
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}

		trends, err := s.store.GetMetricsTrends(scopeType, scopeID, period, limit)
		if err != nil {
			log.Printf("Failed to get metrics trends: %v", err)
			http.Error(w, "Failed to get trends", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(trends); err != nil {
			log.Printf("Failed to encode trends response: %v", err)
		}
		return
	}

	// Handle scope metrics
	scopeType := parts[0]
	scopeID := parts[1]

	if scopeType != "agent" && scopeType != "thread" {
		http.Error(w, "Invalid scope type: expected 'agent' or 'thread'", http.StatusBadRequest)
		return
	}

	metrics, err := s.store.GetMetrics(scopeType, scopeID)
	if err != nil {
		log.Printf("Failed to get %s metrics: %v", scopeType, err)
		http.Error(w, "Failed to get metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		log.Printf("Failed to encode metrics response: %v", err)
	}
}

// handleInstanceHistory returns instance history entries
// GET /api/instances/history?agent_id={id}&limit={n}
func (s *Server) handleInstanceHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // Default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	entries, err := s.store.GetInstanceHistory(agentID, limit)
	if err != nil {
		log.Printf("Failed to get instance history: %v", err)
		http.Error(w, "Failed to get instance history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		log.Printf("Failed to encode instance history response: %v", err)
	}
}

// TaskMetrics represents live metrics for a running task
type TaskMetrics struct {
	TaskID      string  `json:"task_id"`
	ThreadID    string  `json:"thread_id,omitempty"`
	Status      string  `json:"status"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryMB    float64 `json:"memory_mb"`
	TokensIn    int     `json:"tokens_in"`
	TokensOut   int     `json:"tokens_out"`
	Cost        float64 `json:"cost"`
	DurationSec int     `json:"duration_sec"`
	PeakCPU     float64 `json:"peak_cpu"`
	PeakMemory  float64 `json:"peak_memory_mb"`
	TurnNum     int     `json:"turn_num,omitempty"`
	LastEvent   string  `json:"last_event,omitempty"`
}

// CoordinatorStatus represents the coordinator daemon status
type CoordinatorStatus struct {
	Running      bool           `json:"running"`
	PID          int            `json:"pid,omitempty"`
	Uptime       string         `json:"uptime,omitempty"`
	TasksRun     int            `json:"tasks_run"`
	PendingTasks int            `json:"pending_tasks"`
	RunningTasks int            `json:"running_tasks"`
	FailedTasks  int            `json:"failed_tasks"`
	TotalCost    float64        `json:"total_cost"`
	TotalTokens  int            `json:"total_tokens"`
	ActiveTasks  []*TaskMetrics `json:"active_tasks,omitempty"`
}

// handleCoordinatorStatus returns the coordinator daemon status with active task metrics
// GET /api/coordinator/status
func (s *Server) handleCoordinatorStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := &CoordinatorStatus{
		Running: false,
	}

	// Get coordinator status from store if available
	if s.coordStore != nil {
		stats, err := s.coordStore.GetCoordinatorStats()
		if err == nil {
			status.Running = stats.Running
			status.PID = stats.PID
			status.Uptime = stats.Uptime
			status.TasksRun = stats.TasksRun
			status.PendingTasks = stats.PendingTasks
			status.RunningTasks = stats.RunningTasks
			status.FailedTasks = stats.FailedTasks
			status.TotalCost = stats.TotalCost
			status.TotalTokens = stats.TotalTokens
		}
	}

	// Get active task metrics from resource registry
	if s.resourceRegistry != nil {
		metrics := s.resourceRegistry.GetAllMetrics()
		for _, m := range metrics {
			status.ActiveTasks = append(status.ActiveTasks, &TaskMetrics{
				TaskID:      m.TaskID,
				ThreadID:    m.ThreadID,
				Status:      "running",
				CPUPercent:  m.CPUPercent,
				MemoryMB:    m.MemoryMB,
				TokensIn:    m.TokensIn,
				TokensOut:   m.TokensOut,
				Cost:        m.Cost,
				DurationSec: m.DurationSec,
				PeakCPU:     m.PeakCPU,
				PeakMemory:  m.PeakMemory,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Failed to encode coordinator status response: %v", err)
	}
}

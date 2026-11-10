package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/websocket"
)

// TelemetryReportRequest is the JSON body for POST /api/telemetry
type TelemetryReportRequest struct {
	PID        int     `json:"pid"`         // Process ID (required)
	InstanceID string  `json:"instance_id"` // Optional friendly name
	Turns      int     `json:"turns"`
	TokensIn   int     `json:"tokens_in"`
	TokensOut  int     `json:"tokens_out"`
	Cost       float64 `json:"cost"`
	Status     string  `json:"status"` // running, completed, error
}

// handleTelemetryReport handles POST /api/telemetry: eval-suite and pipeline
// self-reports (internal/eval_harness/telemetry_reporter.go,
// internal/pipeline/metrics.go). A report is broadcast to /ws clients, and a
// completed run's token/cost totals are recorded in the messaging store's
// metrics. (The per-PID cache the process monitor read is gone with the
// monitor.)
func (s *Server) handleTelemetryReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TelemetryReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// PID is required
	if req.PID == 0 {
		// Use current process PID if not specified
		req.PID = os.Getpid()
	}

	// Generate instance ID if not provided
	if req.InstanceID == "" {
		req.InstanceID = fmt.Sprintf("external_%d", req.PID)
	}

	// Default status
	if req.Status == "" {
		req.Status = "running"
	}

	// Broadcast to WebSocket clients
	s.wsServer.BroadcastTelemetry(&websocket.TelemetryEvent{
		InstanceID:  req.InstanceID,
		PID:         req.PID,
		Turns:       req.Turns,
		TokensIn:    req.TokensIn,
		TokensOut:   req.TokensOut,
		Cost:        req.Cost,
		Status:      req.Status,
		DurationSec: 0,
	})

	// Record metrics when a run completes
	// This bridges telemetry reports from evals/agents to the metrics aggregation system
	if req.Status == "completed" && (req.TokensIn > 0 || req.TokensOut > 0) {
		stats := &messaging.MessageExecutionStats{
			DurationMS:   0, // Not tracked in telemetry request
			InputTokens:  req.TokensIn,
			OutputTokens: req.TokensOut,
			CostCents:    int(req.Cost * 100), // Convert dollars to cents
			FilesCreated: nil,
		}
		// Use instance_id as agent_id, create a synthetic thread for telemetry runs
		threadID := fmt.Sprintf("telemetry_%s", req.InstanceID)
		_ = s.store.RecordMetrics(threadID, req.InstanceID, stats)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

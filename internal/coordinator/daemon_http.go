package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

// startHealthServer starts an HTTP server on the given port with health and status endpoints.
// Called from Run() when PORT env var is set (Cloud Run convention).
func (d *Daemon) startHealthServer(port string) {
	mux := http.NewServeMux()

	// M3: Health endpoint (Cloud Run startup/liveness probe)
	mux.HandleFunc("/health", d.handleHealth)

	// M4: Status API endpoints
	mux.HandleFunc("/status", d.handleStatus)
	mux.HandleFunc("/chains/active", d.handleChainsActive)
	mux.HandleFunc("/chains/stats", d.handleChainsStats)
	mux.HandleFunc("/pending", d.handlePending)

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	d.logger.Printf("Health server starting on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		d.logger.Printf("Health server error: %v", err)
	}
}

// handleHealth returns a simple health check response.
// Used by Cloud Run startup probes (TCP check) and liveness probes.
func (d *Daemon) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"status":    "ok",
		"component": "coordinator",
		"uptime":    time.Since(d.startedAt).Round(time.Second).String(),
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleStatus returns the coordinator's current state with task counts.
func (d *Daemon) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	status, err := d.Status()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Enrich with task stats from the store if available
	if d.taskStore != nil {
		stats, err := d.taskStore.GetTaskStats(ctx)
		if err == nil {
			status.TasksRun = stats.CompletedTasks
			status.PendingTasks = stats.PendingTasks
			status.RunningTasks = stats.RunningTasks
			status.PendingApprovals = stats.PendingApprovals
			status.FailedTasks = stats.FailedTasks
			status.TotalCost = stats.TotalCost
			status.TotalTokens = stats.TotalTokens
		}
	}

	writeJSON(w, http.StatusOK, status)
}

// handleChainsActive returns currently running execution chains.
func (d *Daemon) handleChainsActive(w http.ResponseWriter, r *http.Request) {
	if d.obsBackend == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	chains, err := d.obsBackend.ListChains(ctx, observatory.ChainListOptions{
		Status: observatory.ChainStatusActive,
		Limit:  50,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, chains)
}

// handleChainsStats returns aggregate chain metrics.
// Accepts ?hours=N query parameter (default: 168 = 1 week).
func (d *Daemon) handleChainsStats(w http.ResponseWriter, r *http.Request) {
	if d.obsBackend == nil {
		writeJSON(w, http.StatusOK, &observatory.ChainStatusCounts{})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// Parse hours query parameter
	hours := 168 // default: 1 week
	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil && parsed > 0 {
			hours = parsed
		}
	}

	createdAfter := time.Now().Add(-time.Duration(hours) * time.Hour)
	counts, err := d.obsBackend.GetChainStatusCounts(ctx, &createdAfter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, counts)
}

// handlePending returns pending approval requests.
func (d *Daemon) handlePending(w http.ResponseWriter, r *http.Request) {
	if d.obsBackend == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	approvals, err := d.obsBackend.ListPendingApprovals(ctx, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, approvals)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

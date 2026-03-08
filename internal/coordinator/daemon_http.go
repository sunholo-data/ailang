package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
	"github.com/sunholo/ailang/internal/pubsub"
)

// requireAPIKey returns middleware that checks for a valid Bearer token.
// When COORDINATOR_API_KEY is unset, all requests pass through (local mode).
func (d *Daemon) requireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := os.Getenv("COORDINATOR_API_KEY")
		if key == "" {
			next(w, r) // No key configured = open (local mode)
			return
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// startHealthServer starts an HTTP server on the given port with health and status endpoints.
// Called from Run() when PORT env var is set (Cloud Run convention).
func (d *Daemon) startHealthServer(port string) {
	mux := http.NewServeMux()

	// M3: Health endpoint (Cloud Run startup/liveness probe) — always public
	mux.HandleFunc("/health", d.handleHealth)

	// M4: Status API endpoints — protected by API key when configured
	mux.HandleFunc("/status", d.requireAPIKey(d.handleStatus))
	mux.HandleFunc("/chains/active", d.requireAPIKey(d.handleChainsActive))
	mux.HandleFunc("/chains/stats", d.requireAPIKey(d.handleChainsStats))
	mux.HandleFunc("/pending", d.requireAPIKey(d.handlePending))

	// M-CLOUD-PUSH: Pub/Sub push endpoints for cloud mode.
	// Pub/Sub delivers messages via HTTP POST instead of pull subscriptions.
	if os.Getenv("COORDINATOR_MODE") == CoordinatorModeCloud {
		mux.HandleFunc("/pubsub/push", d.handlePushMessage)
		mux.HandleFunc("/pubsub/completions", d.handlePushCompletion)
		d.logger.Println("Pub/Sub push endpoints registered: /pubsub/push, /pubsub/completions")

		// M-CLOUD-WEBHOOK: GitHub webhook endpoint replaces polling-based
		// ApprovalWatcher and GitHub sync in cloud mode.
		mux.HandleFunc("/github/webhook", d.handleGitHubWebhook)
		d.logger.Println("GitHub webhook endpoint registered: /github/webhook")
	}

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second, // Increased for Firestore fetch in push handlers
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

// handlePushMessage receives Pub/Sub push messages from the messages topic.
// Pub/Sub POSTs a JSON envelope containing base64-encoded data and attributes.
// HTTP 200 = ack, 5xx = nack (Pub/Sub retries).
func (d *Daemon) handlePushMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	data, attrs, msgID, err := pubsub.DecodePushEnvelope(r.Body)
	if err != nil {
		d.logger.Printf("Push /pubsub/push: bad envelope: %v (acking to prevent retry)", err)
		w.WriteHeader(http.StatusOK) // Ack malformed messages to prevent infinite retry.
		return
	}

	if d.cloudInboxAdapter == nil {
		d.logger.Printf("Push /pubsub/push: no inbox adapter configured (msg=%s)", msgID)
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := d.cloudInboxAdapter.HandleNotification(data, attrs); err != nil {
		d.logger.Printf("Push /pubsub/push: handler error for %s: %v", msgID, err)
		w.WriteHeader(http.StatusInternalServerError) // Nack → Pub/Sub retries.
		return
	}

	d.logger.Printf("Push /pubsub/push: processed message %s", msgID)

	// M-CLOUD-WEBHOOK: Immediately process and dispatch (no 30s ticker wait).
	// This makes cloud mode fully push-driven — message arrives, task dispatches.
	if d.msgAdapter != nil {
		if err := d.pollAndProcessTasks(); err != nil {
			d.logger.Printf("Push /pubsub/push: poll error: %v", err)
		}
	}
	if err := d.executeTaskQueue(); err != nil {
		d.logger.Printf("Push /pubsub/push: dispatch error: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}

// handlePushCompletion receives Pub/Sub push messages from the completions topic.
func (d *Daemon) handlePushCompletion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	data, attrs, msgID, err := pubsub.DecodePushEnvelope(r.Body)
	if err != nil {
		d.logger.Printf("Push /pubsub/completions: bad envelope: %v (acking to prevent retry)", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	if d.completionHandler == nil {
		d.logger.Printf("Push /pubsub/completions: no completion handler configured (msg=%s)", msgID)
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := d.completionHandler.HandleCompletion(data, attrs); err != nil {
		d.logger.Printf("Push /pubsub/completions: handler error for %s: %v", msgID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	d.logger.Printf("Push /pubsub/completions: processed completion %s", msgID)
	w.WriteHeader(http.StatusOK)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

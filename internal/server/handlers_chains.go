package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// handleListChains returns a list of execution chains with summaries.
// GET /api/chains
// Query params:
//   - status: filter by status (active, pending_approval, completed, failed)
//   - source_type: filter by source (github_issue, message, manual)
//   - workspace_id: filter by workspace
//   - github_repo: filter by owner/repository
//   - limit: max results (default 50)
//   - offset: pagination offset
//   - since: positive integer hours (omit for all time)
func (s *Server) handleListChains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	// Parse filter options
	opts := observatory.ChainListOptions{
		Limit:  50,
		Offset: 0,
	}

	if status := q.Get("status"); status != "" {
		opts.Status = observatory.ChainStatus(status)
	}
	if sourceType := q.Get("source_type"); sourceType != "" {
		opts.SourceType = sourceType
	}
	var err error
	opts.Limit, err = chainPageParameter(q, "limit", 50, 1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	opts.Offset, err = chainPageParameter(q, "offset", 0, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	opts.WorkspaceID = q.Get("workspace_id")
	opts.GitHubRepo = q.Get("github_repo")
	if agentID := q.Get("agent_id"); agentID != "" {
		opts.AgentID = agentID
	}
	if q.Has("since") {
		hours, err := chainPageParameter(q, "since", 0, 1)
		// Bound before multiplying so time.Duration cannot wrap into a future cutoff.
		const maxSinceHours = int64(1<<63-1) / int64(time.Hour)
		if err != nil || int64(hours) > maxSinceHours {
			http.Error(w, "since must be a positive integer number of hours within the supported duration range", http.StatusBadRequest)
			return
		}
		t := time.Now().Add(-time.Duration(hours) * time.Hour)
		opts.CreatedAfter = &t
	}

	chains, err := s.obsBackend.ListChains(ctx, opts)
	if err != nil {
		log.Printf("Failed to list chains: %v", err)
		writeChainReadError(w, err)
		return
	}

	if chains == nil {
		chains = []*observatory.ChainSummary{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chains); err != nil {
		log.Printf("Failed to encode chains: %v", err)
	}
}

// handleGetChain returns a single execution chain with all stages.
// GET /api/chains/{id}
// Query params:
//   - include_spans: include spans for each stage (default false)
//   - include_chat: include chat context in spans (default false)
func (s *Server) handleGetChain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	// Extract chain ID from path
	path := r.URL.Path
	chainID := strings.TrimPrefix(path, "/api/chains/")
	if chainID == "" || strings.Contains(chainID, "/") {
		http.Error(w, "Invalid chain ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	// Parse read options
	opts := observatory.ChainReadOptions{
		IncludeStages:   true, // Always include stages for a single chain request
		IncludeSpans:    q.Get("include_spans") == "true",
		IncludeSessions: q.Get("include_sessions") == "true",
	}

	chain, err := s.obsBackend.GetChain(ctx, chainID, opts)
	if err != nil {
		log.Printf("Failed to get chain %s: %v", chainID, err)
		http.Error(w, "Chain not found", http.StatusNotFound)
		return
	}
	// Note: GetChain with IncludeStages=true already loads stages via GetChainStages internally.
	// No need to call GetChainStages again — that was a double-fetch bug (M-PERF-OBSERVATORY).

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chain); err != nil {
		log.Printf("Failed to encode chain: %v", err)
	}
}

// handleGetChainByMessage returns a chain by its source message ID.
// GET /api/chains/by-message/{messageId}
func (s *Server) handleGetChainByMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	// Extract message ID from path
	path := r.URL.Path
	messageID := strings.TrimPrefix(path, "/api/chains/by-message/")
	if messageID == "" {
		http.Error(w, "Missing message ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	chain, err := s.obsBackend.GetChainByMessageID(ctx, messageID)
	if err != nil || chain == nil {
		http.Error(w, "Chain not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chain); err != nil {
		log.Printf("Failed to encode chain: %v", err)
	}
}

// handleGetChainByTask returns a chain by a task ID in any of its stages.
// GET /api/chains/by-task/{taskId}
func (s *Server) handleGetChainByTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	// Extract task ID from path
	path := r.URL.Path
	taskID := strings.TrimPrefix(path, "/api/chains/by-task/")
	if taskID == "" {
		http.Error(w, "Missing task ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	chain, err := s.obsBackend.GetChainByTaskID(ctx, taskID)
	if err != nil || chain == nil {
		// No execution chain found — return 404.
		// The frontend handles 404 by synthesizing a virtual chain from spans,
		// which is the correct behavior for user sessions, evals, etc.
		// Previously this returned a TaskSpanSummary with 200 OK, but that
		// different shape caused the frontend to parse it as ChainData and
		// then try to fetch /api/chains/undefined (M-AUDIT-OBSERVATORY).
		http.Error(w, "Chain not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chain); err != nil {
		log.Printf("Failed to encode chain: %v", err)
	}
}

// handleGetChainByGitHub returns a chain by GitHub repo and issue number.
// GET /api/chains/by-github/{repo}/{issueNumber}
func (s *Server) handleGetChainByGitHub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	// Extract repo and issue number from path: /api/chains/by-github/owner/repo/123
	path := strings.TrimPrefix(r.URL.Path, "/api/chains/by-github/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		http.Error(w, "Expected: /api/chains/by-github/{owner}/{repo}/{issueNumber}", http.StatusBadRequest)
		return
	}

	repo := parts[0] + "/" + parts[1]
	issueNumber, err := strconv.Atoi(parts[2])
	if err != nil || issueNumber <= 0 {
		http.Error(w, "Invalid issue number", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	chain, err := s.obsBackend.GetChainByGitHubIssue(ctx, repo, issueNumber)
	if err != nil || chain == nil {
		http.Error(w, "Chain not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chain); err != nil {
		log.Printf("Failed to encode chain: %v", err)
	}
}

// handleListPendingApprovals returns stages awaiting approval.
// GET /api/chains/pending
// Query params:
//   - limit: max results (default 20)
func (s *Server) handleListPendingApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	limit := 20
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	approvals, err := s.obsBackend.ListPendingApprovals(ctx, limit)
	if err != nil {
		log.Printf("Failed to list pending approvals: %v", err)
		http.Error(w, "Failed to list pending approvals", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(approvals); err != nil {
		log.Printf("Failed to encode pending approvals: %v", err)
	}
}

// handleCreateChain creates a new execution chain.
// POST /api/chains
// Body: { "source_type": "github_issue", "source_ref": "#123", "github_repo": "owner/repo", "github_issue_number": 123 }
func (s *Server) handleCreateChain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	var req observatory.ChainCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SourceType == "" {
		http.Error(w, "source_type is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	chain, err := s.obsBackend.CreateChain(ctx, &req)
	if err != nil {
		log.Printf("Failed to create chain: %v", err)
		http.Error(w, "Failed to create chain", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(chain); err != nil {
		log.Printf("Failed to encode chain: %v", err)
	}
}

// handleCreateStage creates a new stage in an execution chain.
// POST /api/chains/{id}/stages
// Body: { "agent_id": "design-doc-creator", "message_id": "...", "task_id": "..." }
func (s *Server) handleCreateStage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	// Extract chain ID from path
	path := r.URL.Path
	// Path: /api/chains/{id}/stages
	parts := strings.Split(strings.TrimPrefix(path, "/api/chains/"), "/")
	if len(parts) < 2 || parts[1] != "stages" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	chainID := parts[0]

	var req observatory.StageCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.ChainID = chainID
	if req.AgentID == "" {
		http.Error(w, "agent_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	stage, err := s.obsBackend.CreateStage(ctx, &req)
	if err != nil {
		log.Printf("Failed to create stage: %v", err)
		http.Error(w, "Failed to create stage", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(stage); err != nil {
		log.Printf("Failed to encode stage: %v", err)
	}
}

// handleUpdateStageStatus updates a stage's status.
// PATCH /api/chains/{chainId}/stages/{stageId}/status
// Body: { "status": "running" }
func (s *Server) handleUpdateStageStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	// Extract IDs from path
	path := r.URL.Path
	// Path: /api/chains/{chainId}/stages/{stageId}/status
	path = strings.TrimPrefix(path, "/api/chains/")
	path = strings.TrimSuffix(path, "/status")
	parts := strings.Split(path, "/stages/")
	if len(parts) != 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	stageID := parts[1]

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	if err := s.obsBackend.UpdateStageStatus(ctx, stageID, observatory.ChainStageStatus(req.Status)); err != nil {
		log.Printf("Failed to update stage status: %v", err)
		http.Error(w, "Failed to update stage status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleChainsStats returns aggregated chain statistics.
// GET /api/chains/stats
// Query params:
//   - hours: time window in hours (0 = all time)
//   - by_agent: include per-agent breakdown (default false)
func (s *Server) handleChainsStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	// Parse time window
	var hours int
	if h := q.Get("hours"); h != "" {
		if n, err := strconv.Atoi(h); err == nil && n > 0 {
			hours = n
		}
	}
	byAgent := q.Get("by_agent") == "true"

	// Compute time cutoff
	var createdAfter *time.Time
	timeWindow := "all time"
	if hours > 0 {
		t := time.Now().Add(-time.Duration(hours) * time.Hour)
		createdAfter = &t
		timeWindow = strconv.Itoa(hours) + " hours"
	}

	// M-COST-PER-SUCCESS-KPI (M2): additive headline-KPI surface. When requested,
	// return the canonical cost-per-verified-success result — the EXACT same
	// struct the CLI (--json) and latest.json publisher serialize, computed by
	// the SAME observatory rollup (no SQL/cost logic duplicated in the handler).
	if q.Get("cost_per_verified_success") == "true" {
		s.handleCostPerVerifiedSuccess(w, r, createdAfter)
		return
	}

	// Single SQL query for chain counts by status (replaces fetch-all + Go loop, M-PERF-OBSERVATORY)
	counts, err := s.obsBackend.GetChainStatusCounts(ctx, createdAfter)
	if err != nil {
		log.Printf("Failed to get chain status counts: %v", err)
		http.Error(w, "Failed to compute stats", http.StatusInternalServerError)
		return
	}

	result := map[string]interface{}{
		"time_window":      timeWindow,
		"total_chains":     counts.Total,
		"completed":        counts.Completed,
		"active":           counts.Active,
		"pending_approval": counts.Pending,
		"failed":           counts.Failed,
		"total_cost":       counts.TotalCost,
		"total_tokens":     counts.TotalTokens,
	}
	if counts.Total > 0 {
		result["avg_cost_per_chain"] = counts.TotalCost / float64(counts.Total)
	}

	// Single SQL query for per-agent stats (replaces N+1, M-PERF-OBSERVATORY)
	if byAgent {
		agentStats, err := s.obsBackend.GetChainStatsByAgent(ctx, createdAfter)
		if err != nil {
			log.Printf("Failed to get agent stats: %v", err)
		} else {
			result["by_agent"] = agentStats
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Failed to encode chain stats: %v", err)
	}
}

// handleCostPerVerifiedSuccess serves the additive headline KPI
// (M-COST-PER-SUCCESS-KPI, M2) for GET /api/chains/stats?cost_per_verified_success=true.
// Query params:
//   - baseline: frozen-cohort id / chains.source_ref prefix (required, e.g. "v1.0")
//   - hours:    optional cohort window (createdAfter is passed by the caller)
//
// It calls the SAME observatory rollup as the CLI and publisher and serializes
// the identical struct. The KPI's own `available`/`reason` fields carry
// completeness (never a silent $0); an unavailable KPI is still HTTP 200 with
// available=false so the dashboard can render the Incomplete state.
func (s *Server) handleCostPerVerifiedSuccess(w http.ResponseWriter, r *http.Request, createdAfter *time.Time) {
	sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend)
	if !ok {
		http.Error(w, "Cost-per-verified-success requires the SQLite observatory backend", http.StatusServiceUnavailable)
		return
	}

	baseline := r.URL.Query().Get("baseline")
	if baseline == "" {
		http.Error(w, "baseline query param is required (frozen cohort source_ref prefix, e.g. v1.0)", http.StatusBadRequest)
		return
	}

	// Normalize the baseline id into a delimited source_ref prefix so "v1.0"
	// never accidentally matches "v1.05".
	sourceRef := baseline
	if sourceRef[len(sourceRef)-1] != '/' {
		sourceRef += "/"
	}

	res, err := sqliteBackend.Store().CostPerVerifiedSuccess(r.Context(), observatory.CostPerVerifiedSuccessOptions{
		BaselineID:   baseline,
		SourceRef:    sourceRef,
		CreatedAfter: createdAfter,
	})
	if err != nil {
		log.Printf("Failed to compute cost-per-verified-success: %v", err)
		http.Error(w, "Failed to compute cost-per-verified-success", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Printf("Failed to encode cost-per-verified-success: %v", err)
	}
}

// handleChainsActive returns currently active chains.
// GET /api/chains/active
// Query params:
//   - limit: max results (default 20)
func (s *Server) handleChainsActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	limit := 20
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	chains, err := s.obsBackend.ListChains(ctx, observatory.ChainListOptions{
		Status: observatory.ChainStatusActive,
		Limit:  limit,
	})
	if err != nil {
		log.Printf("Failed to list active chains: %v", err)
		http.Error(w, "Failed to list active chains", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chains); err != nil {
		log.Printf("Failed to encode active chains: %v", err)
	}
}

// handleStageSpans returns paginated lightweight spans for a specific stage.
// GET /api/chains/{chainId}/stages/{stageId}/spans
// Query params:
//   - limit: max spans (default 200)
//   - offset: pagination offset (default 0)
//
// This endpoint returns SpanLite records (no attributes/resource_attributes columns),
// which avoids reading the 3.9GB of attribute data (M-PERF-OBSERVATORY Level 2).
func (s *Server) handleStageSpans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	q := r.URL.Query()
	limit, err := chainPageParameter(q, "limit", 200, 1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	offset, err := chainPageParameter(q, "offset", 0, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	stage, ok := s.readOwnedStage(w, r, "spans")
	if !ok {
		return
	}
	stageID := stage.ID
	ctx := r.Context()

	page, err := s.obsBackend.GetSpanLitesByStageID(ctx, stageID, limit, offset)
	if err != nil {
		log.Printf("Failed to get spans for stage %s: %v", stageID, err)
		writeChainReadError(w, err)
		return
	}

	if page == nil {
		writeChainReadError(w, fmt.Errorf("backend returned nil span page"))
		return
	}
	if page.Spans == nil {
		page.Spans = []*observatory.SpanLite{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(page); err != nil {
		log.Printf("Failed to encode stage spans: %v", err)
	}
}

// handleGetSpanDetail returns a single span with full attributes.
// GET /api/spans/{spanId}
// This is Level 3 loading — only called when user clicks a specific span (M-PERF-OBSERVATORY).
func (s *Server) handleGetSpanDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	spanID := strings.TrimPrefix(r.URL.Path, "/api/spans/")
	if spanID == "" {
		http.Error(w, "Missing span ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	span, err := s.obsBackend.GetSpan(ctx, spanID)
	if err != nil || span == nil {
		http.Error(w, "Span not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(span); err != nil {
		log.Printf("Failed to encode span: %v", err)
	}
}

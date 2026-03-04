package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

// handleListChains returns a list of execution chains with summaries.
// GET /api/chains
// Query params:
//   - status: filter by status (active, pending_approval, completed, failed)
//   - source_type: filter by source (github_issue, message, manual)
//   - limit: max results (default 50)
//   - offset: pagination offset
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
	if limit := q.Get("limit"); limit != "" {
		if n, err := strconv.Atoi(limit); err == nil && n > 0 {
			opts.Limit = n
		}
	}
	if offset := q.Get("offset"); offset != "" {
		if n, err := strconv.Atoi(offset); err == nil && n >= 0 {
			opts.Offset = n
		}
	}
	if agentID := q.Get("agent_id"); agentID != "" {
		opts.AgentID = agentID
	}
	if since := q.Get("since"); since != "" {
		// Parse as hours
		if n, err := strconv.Atoi(since); err == nil && n > 0 {
			t := time.Now().Add(-time.Duration(n) * time.Hour)
			opts.CreatedAfter = &t
		}
	}

	chains, err := s.obsBackend.ListChains(ctx, opts)
	if err != nil {
		log.Printf("Failed to list chains: %v", err)
		http.Error(w, "Failed to list chains", http.StatusInternalServerError)
		return
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
		// No execution chain — fall back to span summary (user sessions, evals, etc.)
		if sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend); ok {
			summary, summaryErr := sqliteBackend.GetTaskSpanSummary(ctx, taskID)
			if summaryErr == nil && summary != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(summary)
				return
			}
		}
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

	// Parse path: /api/chains/{chainId}/stages/{stageId}/spans
	path := strings.TrimPrefix(r.URL.Path, "/api/chains/")
	parts := strings.Split(path, "/stages/")
	if len(parts) != 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	stageAndSuffix := strings.TrimSuffix(parts[1], "/spans")
	stageID := stageAndSuffix
	if stageID == "" {
		http.Error(w, "Missing stage ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	limit := 200
	offset := 0
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if o := q.Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	page, err := s.obsBackend.GetSpanLitesByStageID(ctx, stageID, limit, offset)
	if err != nil {
		log.Printf("Failed to get spans for stage %s: %v", stageID, err)
		http.Error(w, "Failed to get spans", http.StatusInternalServerError)
		return
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

// handleStageChat returns chat messages for a specific stage.
// GET /api/chains/{chainId}/stages/{stageId}/chat
// Query params:
//   - limit: max messages (default 50)
//   - offset: pagination offset (default 0)
//
// This is Level 4 loading — only called when user clicks the "Chat" tab (M-PERF-OBSERVATORY).
func (s *Server) handleStageChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	// Parse path: /api/chains/{chainId}/stages/{stageId}/chat
	path := strings.TrimPrefix(r.URL.Path, "/api/chains/")
	parts := strings.Split(path, "/stages/")
	if len(parts) != 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	stageID := strings.TrimSuffix(parts[1], "/chat")
	if stageID == "" {
		http.Error(w, "Missing stage ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Look up the stage to get its task_id or session_id
	stage, err := s.obsBackend.GetStage(ctx, stageID)
	if err != nil || stage == nil {
		http.Error(w, "Stage not found", http.StatusNotFound)
		return
	}

	var messages []*observatory.ChatMessage

	// Try task_id first, fall back to session_id
	if stage.TaskID != "" {
		messages, err = s.obsBackend.GetChatMessagesByTaskID(ctx, stage.TaskID)
	} else if stage.SessionID != "" {
		var startTime, endTime time.Time
		if stage.StartedAt != nil {
			startTime = *stage.StartedAt
		}
		if stage.CompletedAt != nil {
			endTime = *stage.CompletedAt
		} else {
			endTime = time.Now()
		}
		messages, err = s.obsBackend.GetChatMessagesBySession(ctx, stage.SessionID, startTime, endTime)
	}

	if err != nil {
		log.Printf("Failed to get chat for stage %s: %v", stageID, err)
		http.Error(w, "Failed to get chat messages", http.StatusInternalServerError)
		return
	}

	if messages == nil {
		messages = []*observatory.ChatMessage{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
		"total":    len(messages),
		"stage_id": stageID,
	}); err != nil {
		log.Printf("Failed to encode chat messages: %v", err)
	}
}

// handleChainJourney returns a pre-computed narrative journey for a chain.
// GET /api/chains/{id}/journey
func (s *Server) handleChainJourney(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not configured", http.StatusServiceUnavailable)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/chains/")
	chainID := strings.TrimSuffix(path, "/journey")
	if chainID == "" {
		http.Error(w, "Chain ID required", http.StatusBadRequest)
		return
	}

	sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend)
	if !ok {
		http.Error(w, "Journey not supported for this backend", http.StatusNotImplemented)
		return
	}

	journey, err := sqliteBackend.GetChainJourney(r.Context(), chainID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(journey)
}

// registerChainRoutes registers all chain-related API routes.
func (s *Server) registerChainRoutes(mux *http.ServeMux) {
	// List and create chains
	mux.HandleFunc("/api/chains", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.handleListChains(w, r)
		case http.MethodPost:
			s.handleCreateChain(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Stats endpoint (must be before /api/chains/{id} catch-all)
	mux.HandleFunc("/api/chains/stats", s.handleChainsStats)

	// Active chains (must be before /api/chains/{id} catch-all)
	mux.HandleFunc("/api/chains/active", s.handleChainsActive)

	// Pending approvals (must be before /api/chains/{id} to match correctly)
	mux.HandleFunc("/api/chains/pending", s.handleListPendingApprovals)

	// Lookup by message
	mux.HandleFunc("/api/chains/by-message/", s.handleGetChainByMessage)

	// Lookup by task
	mux.HandleFunc("/api/chains/by-task/", s.handleGetChainByTask)

	// Lookup by GitHub issue
	mux.HandleFunc("/api/chains/by-github/", s.handleGetChainByGitHub)

	// Span detail endpoint (must be before /api/chains/ catch-all)
	mux.HandleFunc("/api/spans/", s.handleGetSpanDetail)

	// Single chain operations - this catches /api/chains/{id} and /api/chains/{id}/stages/*
	mux.HandleFunc("/api/chains/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/chains/")

		// Handle /api/chains/{id}/journey (M-CHAIN-JOURNEY)
		if strings.HasSuffix(path, "/journey") {
			s.handleChainJourney(w, r)
			return
		}

		// Handle /api/chains/{id}/stages/{stageId}/spans (M-PERF-OBSERVATORY L2)
		if strings.HasSuffix(path, "/spans") && strings.Contains(path, "/stages/") {
			s.handleStageSpans(w, r)
			return
		}

		// Handle /api/chains/{id}/stages/{stageId}/chat (M-PERF-OBSERVATORY L4)
		if strings.HasSuffix(path, "/chat") && strings.Contains(path, "/stages/") {
			s.handleStageChat(w, r)
			return
		}

		// Handle /api/chains/{id}/stages/{stageId}/status
		if strings.Contains(path, "/stages") {
			if strings.HasSuffix(path, "/status") {
				s.handleUpdateStageStatus(w, r)
			} else if r.Method == http.MethodPost {
				s.handleCreateStage(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		// Handle /api/chains/{id}
		s.handleGetChain(w, r)
	})
}

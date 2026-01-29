package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

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

	// If chain found but we also want stages, fetch them
	if chain != nil && opts.IncludeSpans {
		stages, err := s.obsBackend.GetChainStages(ctx, chainID, opts)
		if err != nil {
			log.Printf("Failed to get stages for chain %s: %v", chainID, err)
		} else {
			chain.Stages = stages
		}
	}

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

	// Pending approvals (must be before /api/chains/{id} to match correctly)
	mux.HandleFunc("/api/chains/pending", s.handleListPendingApprovals)

	// Lookup by message
	mux.HandleFunc("/api/chains/by-message/", s.handleGetChainByMessage)

	// Lookup by task
	mux.HandleFunc("/api/chains/by-task/", s.handleGetChainByTask)

	// Single chain operations - this catches /api/chains/{id} and /api/chains/{id}/stages
	mux.HandleFunc("/api/chains/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/chains/")

		// Handle /api/chains/{id}/stages
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

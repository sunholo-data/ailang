package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

// handleStageChat returns chat messages for a specific stage.
// GET /api/chains/{id}/stages/{stageId}/chat
// Query params:
//   - limit: max results (default 100)
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

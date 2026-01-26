package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/claudehistory"
)

// ClaudeHistoryHandler provides HTTP handlers for Claude Code conversation history.
type ClaudeHistoryHandler struct {
	reader *claudehistory.Reader
}

// NewClaudeHistoryHandler creates a new handler with a claudehistory reader.
func NewClaudeHistoryHandler() *ClaudeHistoryHandler {
	return &ClaudeHistoryHandler{
		reader: claudehistory.NewReader(),
	}
}

// handleClaudeHistoryProjects handles GET /api/claude-history/projects
func (s *Server) handleClaudeHistoryProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	handler := NewClaudeHistoryHandler()
	projects, err := handler.reader.ListProjects()
	if err != nil {
		http.Error(w, "Failed to list projects: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(projects); err != nil {
		log.Printf("Failed to encode projects response: %v", err)
	}
}

// handleClaudeHistorySessions handles GET /api/claude-history/sessions?project=X
func (s *Server) handleClaudeHistorySessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectPath := r.URL.Query().Get("project")
	if projectPath == "" {
		http.Error(w, "project parameter required", http.StatusBadRequest)
		return
	}

	handler := NewClaudeHistoryHandler()
	sessions, err := handler.reader.ListSessions(projectPath)
	if err != nil {
		http.Error(w, "Failed to list sessions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sessions); err != nil {
		log.Printf("Failed to encode sessions response: %v", err)
	}
}

// handleClaudeHistorySession handles GET /api/claude-history/session/{id}
// Supports pagination with ?offset=N&limit=M query parameters
func (s *Server) handleClaudeHistorySession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract session ID from path: /api/claude-history/session/abc-123
	path := r.URL.Path
	prefix := "/api/claude-history/session/"
	if !strings.HasPrefix(path, prefix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	sessionID := strings.TrimPrefix(path, prefix)
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	handler := NewClaudeHistoryHandler()
	session, err := handler.reader.GetSession(sessionID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Session not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get session: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Support pagination for large sessions
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if limit > 0 && len(session.Messages) > offset {
		end := offset + limit
		if end > len(session.Messages) {
			end = len(session.Messages)
		}
		session.Messages = session.Messages[offset:end]
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(session); err != nil {
		log.Printf("Failed to encode session response: %v", err)
	}
}

// handleClaudeHistoryBySpan handles GET /api/claude-history/by-span/{spanId}
// Correlates a span to its corresponding chat context via session.id attribute
func (s *Server) handleClaudeHistoryBySpan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract span ID from path
	path := r.URL.Path
	prefix := "/api/claude-history/by-span/"
	if !strings.HasPrefix(path, prefix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	spanID := strings.TrimPrefix(path, prefix)
	if spanID == "" {
		http.Error(w, "Span ID required", http.StatusBadRequest)
		return
	}

	// Get span from observatory to extract session.id
	if s.obsBackend == nil {
		http.Error(w, "Observatory not configured", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	span, err := s.obsBackend.GetSpan(ctx, spanID)
	if err != nil {
		http.Error(w, "Span not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Extract session.id from span attributes
	sessionID, ok := span.Attributes["session.id"].(string)
	if !ok || sessionID == "" {
		http.Error(w, "Span has no session.id attribute", http.StatusNotFound)
		return
	}

	handler := NewClaudeHistoryHandler()
	session, err := handler.reader.GetSession(sessionID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Session not found for span", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get session: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Filter messages to span's time window if we have start/end times
	var spanStart, spanEnd time.Time
	spanStart = span.StartTime
	if span.EndTime != nil {
		spanEnd = *span.EndTime
	}

	if !spanStart.IsZero() && !spanEnd.IsZero() {
		// Add 5 minute buffer before and after for context
		start := spanStart.Add(-5 * time.Minute)
		end := spanEnd.Add(5 * time.Minute)

		var filtered []claudehistory.Message
		for _, msg := range session.Messages {
			if (msg.Timestamp.Equal(start) || msg.Timestamp.After(start)) &&
				(msg.Timestamp.Equal(end) || msg.Timestamp.Before(end)) {
				filtered = append(filtered, msg)
			}
		}
		session.Messages = filtered
	}

	// Return response with correlation metadata
	response := struct {
		*claudehistory.Session
		SpanID    string    `json:"span_id"`
		SpanStart time.Time `json:"span_start"`
		SpanEnd   time.Time `json:"span_end"`
	}{
		Session:   session,
		SpanID:    spanID,
		SpanStart: spanStart,
		SpanEnd:   spanEnd,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode by-span response: %v", err)
	}
}

// handleClaudeHistorySearch handles GET /api/claude-history/search?q=query
func (s *Server) handleClaudeHistorySearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "q (query) parameter required", http.StatusBadRequest)
		return
	}

	// Optional filters
	project := r.URL.Query().Get("project")
	model := r.URL.Query().Get("model")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	handler := NewClaudeHistoryHandler()

	// Simple text search implementation
	// TODO: Implement proper SimHash-based search in M5
	type SearchResult struct {
		SessionID   string    `json:"session_id"`
		ProjectPath string    `json:"project_path"`
		ProjectName string    `json:"project_name"`
		Model       string    `json:"model"`
		Timestamp   time.Time `json:"timestamp"`
		Snippet     string    `json:"snippet"`
		TurnCount   int       `json:"turn_count"`
	}

	var results []SearchResult
	queryLower := strings.ToLower(query)

	projects, err := handler.reader.ListProjects()
	if err != nil {
		http.Error(w, "Failed to list projects: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for _, proj := range projects {
		// Filter by project if specified
		if project != "" && proj.Path != project {
			continue
		}

		sessions, err := handler.reader.ListSessions(proj.Path)
		if err != nil {
			continue
		}

		for _, sessionMeta := range sessions {
			// Filter by model if specified
			if model != "" && sessionMeta.Model != model {
				continue
			}

			// Load session to search content
			session, err := handler.reader.GetSession(sessionMeta.ID)
			if err != nil {
				continue
			}

			// Search through messages
			for _, msg := range session.Messages {
				for _, block := range msg.Content {
					var text string
					switch block.Type {
					case "text":
						text = block.Text
					case "thinking":
						text = block.Thinking
					}

					if text != "" && strings.Contains(strings.ToLower(text), queryLower) {
						// Create snippet
						idx := strings.Index(strings.ToLower(text), queryLower)
						start := idx - 50
						if start < 0 {
							start = 0
						}
						end := idx + len(query) + 50
						if end > len(text) {
							end = len(text)
						}
						snippet := text[start:end]
						if start > 0 {
							snippet = "..." + snippet
						}
						if end < len(text) {
							snippet = snippet + "..."
						}

						results = append(results, SearchResult{
							SessionID:   session.ID,
							ProjectPath: session.ProjectPath,
							ProjectName: session.ProjectName,
							Model:       session.Model,
							Timestamp:   msg.Timestamp,
							Snippet:     snippet,
							TurnCount:   session.TurnCount,
						})

						// Limit results per session
						if len(results) >= limit {
							goto done
						}
						break // One match per message is enough
					}
				}
			}
		}
	}
done:

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Printf("Failed to encode search response: %v", err)
	}
}

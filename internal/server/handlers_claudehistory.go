package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/claudehistory"
	"github.com/sunholo-data/ailang/internal/observatory"
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

// ============================================================================
// Database-backed chat history endpoints (M-CHAT-HISTORY-DB)
// ============================================================================

// handleClaudeHistorySync handles POST /api/claude-history/sync
// Triggers import of JSONL files to observatory.db
func (s *Server) handleClaudeHistorySync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get SQLite backend for database access
	sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend)
	if !ok {
		http.Error(w, "Chat sync requires SQLite backend", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	importer := claudehistory.NewImporter(sqliteBackend.DB())

	// Check for session_id parameter (sync specific session)
	sessionID := r.URL.Query().Get("session_id")
	if sessionID != "" {
		msgCount, err := importer.SyncSession(ctx, sessionID)
		if err != nil {
			http.Error(w, "Sync failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := struct {
			SessionID        string `json:"session_id"`
			MessagesImported int    `json:"messages_imported"`
		}{
			SessionID:        sessionID,
			MessagesImported: msgCount,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Sync all sessions
	stats, err := importer.SyncAll(ctx)
	if err != nil {
		http.Error(w, "Sync failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Failed to encode sync response: %v", err)
	}
}

// handleClaudeHistorySyncStatus handles GET /api/claude-history/sync-status
// Returns the import status for all sessions
func (s *Server) handleClaudeHistorySyncStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get SQLite backend for database access
	sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend)
	if !ok {
		http.Error(w, "Chat sync status requires SQLite backend", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	importer := claudehistory.NewImporter(sqliteBackend.DB())

	// Check for session_id parameter
	sessionID := r.URL.Query().Get("session_id")
	if sessionID != "" {
		status, err := importer.GetImportStatus(ctx, sessionID)
		if err != nil {
			http.Error(w, "Failed to get status: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if status == nil {
			http.Error(w, "Session not imported", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
		return
	}

	// Get all import statuses
	statuses, err := importer.GetAllImportStatus(ctx)
	if err != nil {
		http.Error(w, "Failed to get status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Statuses []*claudehistory.ImportStatus `json:"statuses"`
		Count    int                           `json:"count"`
	}{
		Statuses: statuses,
		Count:    len(statuses),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode sync status response: %v", err)
	}
}

// handleClaudeHistoryDBSession handles GET /api/claude-history/db/session/{id}
// Gets chat messages from the database (imported from JSONL)
func (s *Server) handleClaudeHistoryDBSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract session ID from path
	path := r.URL.Path
	prefix := "/api/claude-history/db/session/"
	if !strings.HasPrefix(path, prefix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	sessionID := strings.TrimPrefix(path, prefix)
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	// Get SQLite backend for database access
	sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend)
	if !ok {
		http.Error(w, "DB session requires SQLite backend", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	importer := claudehistory.NewImporter(sqliteBackend.DB())

	messages, err := importer.GetChatMessages(ctx, sessionID)
	if err != nil {
		http.Error(w, "Failed to get messages: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(messages) == 0 {
		http.Error(w, "Session not found in database", http.StatusNotFound)
		return
	}

	// Support pagination
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	totalCount := len(messages)
	if limit > 0 && len(messages) > offset {
		end := offset + limit
		if end > len(messages) {
			end = len(messages)
		}
		messages = messages[offset:end]
	}

	response := struct {
		SessionID  string                       `json:"session_id"`
		Messages   []*claudehistory.ChatMessage `json:"messages"`
		TotalCount int                          `json:"total_count"`
		Offset     int                          `json:"offset"`
		Limit      int                          `json:"limit"`
	}{
		SessionID:  sessionID,
		Messages:   messages,
		TotalCount: totalCount,
		Offset:     offset,
		Limit:      limit,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode DB session response: %v", err)
	}
}

// handleClaudeHistoryDBBySpan handles GET /api/claude-history/db/by-span/{spanId}
// Gets chat context for a span from the database, with time-range filtering
func (s *Server) handleClaudeHistoryDBBySpan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract span ID from path
	path := r.URL.Path
	prefix := "/api/claude-history/db/by-span/"
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

	// Get SQLite backend for database access
	sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend)
	if !ok {
		http.Error(w, "DB by-span requires SQLite backend", http.StatusServiceUnavailable)
		return
	}

	importer := claudehistory.NewImporter(sqliteBackend.DB())

	// Get messages, optionally filtered by span time window
	var messages []*claudehistory.ChatMessage
	spanStart := span.StartTime
	var spanEnd time.Time
	if span.EndTime != nil {
		spanEnd = *span.EndTime
	}

	if !spanStart.IsZero() && !spanEnd.IsZero() {
		// Add 5 minute buffer for context
		start := spanStart.Add(-5 * time.Minute)
		end := spanEnd.Add(5 * time.Minute)
		messages, err = importer.GetChatMessagesByTimeRange(ctx, sessionID, start, end)
	} else {
		messages, err = importer.GetChatMessages(ctx, sessionID)
	}

	if err != nil {
		http.Error(w, "Failed to get messages: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		SessionID  string                       `json:"session_id"`
		SpanID     string                       `json:"span_id"`
		SpanStart  time.Time                    `json:"span_start"`
		SpanEnd    time.Time                    `json:"span_end"`
		Messages   []*claudehistory.ChatMessage `json:"messages"`
		TotalCount int                          `json:"total_count"`
	}{
		SessionID:  sessionID,
		SpanID:     spanID,
		SpanStart:  spanStart,
		SpanEnd:    spanEnd,
		Messages:   messages,
		TotalCount: len(messages),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode DB by-span response: %v", err)
	}
}

// ChatContextPreview represents a preview of chat context for embedding in span responses
type ChatContextPreview struct {
	UserPrompt        string `json:"user_prompt,omitempty"`        // First 500 chars of user prompt
	AssistantResponse string `json:"assistant_response,omitempty"` // First 500 chars of response
	HasThinking       bool   `json:"has_thinking"`
	TurnNumber        int    `json:"turn_number"`
	FullChatURL       string `json:"full_chat_url"` // Link to full conversation
}

// GetChatContextForSpan retrieves chat context preview for a span from the database.
// This is used to enrich span responses with conversation context.
func (s *Server) GetChatContextForSpan(ctx context.Context, span *observatory.Span) *ChatContextPreview {
	if s.obsBackend == nil {
		return nil
	}

	// Extract session.id from span attributes
	sessionID, ok := span.Attributes["session.id"].(string)
	if !ok || sessionID == "" {
		return nil
	}

	// Get SQLite backend for database access
	sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend)
	if !ok {
		return nil
	}

	importer := claudehistory.NewImporter(sqliteBackend.DB())

	// Get messages around the span time
	var messages []*claudehistory.ChatMessage
	var err error

	spanStart := span.StartTime
	var spanEnd time.Time
	if span.EndTime != nil {
		spanEnd = *span.EndTime
	}

	if !spanStart.IsZero() && !spanEnd.IsZero() {
		// Get messages within span time window
		start := spanStart.Add(-2 * time.Minute)
		end := spanEnd.Add(2 * time.Minute)
		messages, err = importer.GetChatMessagesByTimeRange(ctx, sessionID, start, end)
	} else {
		// Get all messages for session (limited)
		messages, err = importer.GetChatMessages(ctx, sessionID)
		if len(messages) > 10 {
			messages = messages[:10] // Limit preview to first 10
		}
	}

	if err != nil || len(messages) == 0 {
		return nil
	}

	// Build preview from messages
	preview := &ChatContextPreview{
		FullChatURL: "/api/claude-history/db/session/" + sessionID,
	}

	for _, msg := range messages {
		if msg.Role == "user" && preview.UserPrompt == "" {
			preview.UserPrompt = truncateString(msg.ContentText, 500)
			preview.TurnNumber = msg.TurnNumber
		} else if msg.Role == "assistant" {
			if preview.AssistantResponse == "" {
				preview.AssistantResponse = truncateString(msg.ContentText, 500)
			}
			if msg.ContentThinking != "" {
				preview.HasThinking = true
			}
		}
	}

	return preview
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

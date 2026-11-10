package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
)

// UnifiedEvent represents either an inbox message or a Claude Code event
// This provides a consistent format for the Event Queue in the dashboard
type UnifiedEvent struct {
	ID            string `json:"id"`
	CreatedAt     string `json:"created_at"`               // ISO8601 timestamp
	Type          string `json:"message_type"`             // e.g., "notification", "handoff", "claude_code_turn"
	FromAgent     string `json:"from_agent"`               // e.g., "eval-suite", "claude-code"
	ToInbox       string `json:"to_inbox"`                 // e.g., "user", "coordinator"
	Title         string `json:"title"`                    // Display title
	TaskID        string `json:"task_id"`                  // For linking to hierarchy/waterfall
	Status        string `json:"status"`                   // e.g., "unread", "read"
	Payload       string `json:"payload,omitempty"`        // Message payload (inbox messages only)
	CorrelationID string `json:"correlation_id,omitempty"` // For linking related messages
	Source        string `json:"source"`                   // "inbox" or "claude_code"

	// Claude Code specific fields
	CostUSD        float64 `json:"cost_usd,omitempty"`
	TokensIn       int64   `json:"tokens_in,omitempty"`
	TokensOut      int64   `json:"tokens_out,omitempty"`
	TurnCount      int     `json:"turn_count,omitempty"`
	DurationMs     int     `json:"duration_ms,omitempty"`
	Workspace      string  `json:"workspace,omitempty"`       // Working directory for Claude Code events
	Model          string  `json:"model,omitempty"`           // AI model used (e.g., "claude-opus-4-5-20251101")
	Provider       string  `json:"provider,omitempty"`        // AI provider (e.g., "claude", "gemini")
	SourceType     string  `json:"source_type,omitempty"`     // Source type: coordinator, eval, user_session, etc.
	Directive      string  `json:"directive,omitempty"`       // Initial user prompt (truncated preview)
	DirectiveFull  string  `json:"directive_full,omitempty"`  // Full directive (for detail views)
	MetricsSummary string  `json:"metrics_summary,omitempty"` // "3 turns • $0.42 • 12.5s"
}

// GET /api/inbox - list inbox messages merged with Claude Code events.
// Read by `ailang chains dashboard inbox` (cmd/ailang/dashboard_spans.go). The
// write routes the UI used (send, ack, delete, ack-all, cleanup) are gone:
// messages are written through `ailang messages` and the coordinator.
func (s *Server) handleInbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()

	opts := messaging.InboxListOptions{
		Inbox:      q.Get("inbox"),
		FromAgent:  q.Get("from"),
		UnreadOnly: q.Get("unread") == "true",
		StartDate:  q.Get("start_date"),
		EndDate:    q.Get("end_date"),
		Limit:      0, // 0 = no limit (pagination handles display)
	}

	// Parse limit from query parameter (0 = no limit)
	if limitStr := q.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			opts.Limit = limit
		}
	}

	if q.Get("status") != "" {
		opts.Status = q.Get("status")
	}

	// Parse observatory filters for filtering Claude Code events
	providerFilter := q.Get("provider")
	modelFilter := q.Get("model")
	workspaceFilter := q.Get("workspace")
	sourceTypeFilter := q.Get("source_type") // coordinator, user_session, eval, etc.

	// Sorting parameters
	sortBy := q.Get("sort")     // timestamp (default), turns, cost, tokens, duration
	sortOrder := q.Get("order") // asc, desc (default: desc)
	if sortBy == "" {
		sortBy = "timestamp"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	messages, err := s.store.ListInboxMessages(opts)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list messages: %v", err), http.StatusInternalServerError)
		return
	}

	// Also get counts for the response
	counts, _ := s.store.CountInboxMessagesByStatus(opts.Inbox)

	// Check if we should include Claude Code events
	// Include by default unless:
	// - Filtering by specific inbox/from agent
	// - Provider filter is set and != "claude" (Claude Code is always provider=claude)
	// - Source type filter is set to something that excludes Claude Code events
	includeClaudeCode := opts.Inbox == "" && opts.FromAgent == ""
	if providerFilter != "" && providerFilter != "claude" {
		includeClaudeCode = false
	}
	// Source types that ARE Claude Code events
	if sourceTypeFilter != "" && sourceTypeFilter != "coordinator" && sourceTypeFilter != "user_session" {
		// eval, direct_api, exec, local, other - these are NOT Claude Code sessions
		includeClaudeCode = false
	}

	// Convert inbox messages to unified event format
	// Apply source_type filter to inbox messages based on from_agent
	// Apply workspace filter - inbox messages don't have workspace data, so they're excluded when workspace filter is active
	events := make([]UnifiedEvent, 0, len(messages)+50)
	for _, msg := range messages {
		// Filter inbox messages by source_type
		if sourceTypeFilter != "" && !inboxMessageMatchesSourceType(msg.FromAgent, msg.ToInbox, sourceTypeFilter) {
			continue
		}
		// Inbox messages don't have workspace data - exclude when workspace filter is active
		// unless it's "No Workspace" which means "show items without workspace"
		if workspaceFilter != "" && workspaceFilter != "No Workspace" && workspaceFilter != "unknown" {
			continue // Skip inbox messages when filtering by specific workspace
		}
		events = append(events, UnifiedEvent{
			ID:            msg.MessageID,
			CreatedAt:     msg.CreatedAt.Format(time.RFC3339),
			Type:          msg.MessageType,
			FromAgent:     msg.FromAgent,
			ToInbox:       msg.ToInbox,
			Title:         msg.Title,
			TaskID:        msg.CorrelationID, // Use correlation_id as task_id for linking
			Status:        msg.Status,
			Payload:       msg.Payload,
			CorrelationID: msg.CorrelationID,
			Source:        "inbox",
			SourceType:    InferInboxSourceType(msg.FromAgent, msg.ToInbox),
		})
	}

	// Merge Claude Code events from observatory if available
	if includeClaudeCode && s.obsBackend != nil {
		if sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend); ok {
			ctx := context.Background()

			// Load workspace config for reverse mapping (workspace ID → path patterns)
			wsConfig := coordinator.LoadWorkspacesConfig()

			// Use lookup function if coordinator store is available
			// This resolves task_id → agent info for proper event classification
			var ccEvents []observatory.ClaudeCodeEvent
			var err error

			if s.coordStoreRaw != nil {
				// Use lookup to resolve coordinator task → agent info
				// Pass source_type and workspace filters to apply at SQL level (before LIMIT)
				ccEvents, err = sqliteBackend.GetClaudeCodeEventsWithLookup(ctx, opts.Limit, s.coordStoreRaw.GetTaskAgentInfo, sourceTypeFilter, workspaceFilter, wsConfig)
			} else {
				// Fallback: no coordinator store, use defaults (claude-code → user)
				// Still apply workspace filter at SQL level
				ccEvents, err = sqliteBackend.GetClaudeCodeEventsWithLookup(ctx, opts.Limit, nil, sourceTypeFilter, workspaceFilter, wsConfig)
			}

			if err != nil {
				log.Printf("Warning: Failed to get Claude Code events: %v", err)
			} else {
				for _, cc := range ccEvents {
					// Apply model filter if specified
					if modelFilter != "" && cc.Model != modelFilter {
						continue
					}
					// Apply workspace filter if specified
					if workspaceFilter != "" {
						// Special case: "No Workspace" filter matches events with no workspace
						if workspaceFilter == "No Workspace" || workspaceFilter == "unknown" {
							if cc.Workspace != "" {
								continue // Has workspace, but filter wants none → skip
							}
							// cc.Workspace is empty, matches "No Workspace" filter → include
						} else {
							// Normal workspace filter - hide events with no workspace data
							if cc.Workspace == "" {
								continue // No workspace data → skip (user wants specific workspace)
							}
							if !matchesWorkspace(cc.Workspace, workspaceFilter, wsConfig) {
								continue // Has workspace but doesn't match → skip
							}
						}
					}
					events = append(events, UnifiedEvent{
						ID:             cc.ID,
						CreatedAt:      cc.CreatedAt,
						Type:           cc.Type,
						FromAgent:      cc.FromAgent,
						ToInbox:        cc.ToInbox,
						Title:          cc.Title,
						TaskID:         cc.TaskID,
						Status:         cc.Status,
						CostUSD:        cc.CostUSD,
						TokensIn:       cc.TokensIn,
						TokensOut:      cc.TokensOut,
						TurnCount:      cc.TurnCount,
						DurationMs:     cc.DurationMs,
						Source:         "claude_code",
						Model:          cc.Model,
						Provider:       cc.Provider,
						Workspace:      cc.Workspace,
						SourceType:     InferInboxSourceType(cc.FromAgent, cc.ToInbox),
						Directive:      cc.Directive,
						DirectiveFull:  cc.DirectiveFull,
						MetricsSummary: cc.MetricsSummary,
					})
				}
			}
		}
	}

	// Sort events by requested field
	sortEvents(events, sortBy, sortOrder)

	// Apply limit after merging (0 = no limit)
	if opts.Limit > 0 && len(events) > opts.Limit {
		events = events[:opts.Limit]
	}

	response := struct {
		Messages []UnifiedEvent   `json:"messages"`
		Counts   map[string]int64 `json:"counts"`
	}{
		Messages: events,
		Counts:   counts,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode inbox response: %v", err)
	}
}

// matchesWorkspace checks if a full path matches a workspace filter.
// The filter can be:
// - A workspace ID: "MarkEdmondson1234/TwilightGame" (uses wsConfig for reverse mapping)
// - A full path: "/Users/mark/dev/sunholo/ailang"
// - A project name: "ailang" (matches paths ending with /ailang)
// - A special grouping: "Eval Benchmarks", "Coordinator Tasks", "Unknown Workspace"
func matchesWorkspace(fullPath, filter string, wsConfig *coordinator.WorkspacesConfig) bool {
	if fullPath == "" || filter == "" {
		return true
	}

	// Exact match
	if fullPath == filter {
		return true
	}

	// Handle special groupings
	switch filter {
	case "Eval Benchmarks", "eval_workspace":
		return strings.Contains(fullPath, ".eval_workspace") || strings.Contains(fullPath, "eval_workspace")
	case "Coordinator Tasks", "coordinator_worktrees":
		return strings.Contains(fullPath, "/worktrees/")
	case "No Workspace", "unknown":
		return fullPath == "" || fullPath == "unknown"
	}

	// Use workspace config for reverse mapping (workspace ID → path patterns)
	// This handles workspace IDs like "MarkEdmondson1234/TwilightGame"
	if wsConfig != nil {
		patterns := wsConfig.GetPathPatternsForWorkspace(filter)
		for _, pattern := range patterns {
			// Convert SQL LIKE pattern to Go matching
			// Pattern uses % for wildcards, convert to simple contains/prefix/suffix
			p := strings.ReplaceAll(pattern, "%", "")
			if p != "" && strings.Contains(fullPath, p) {
				return true
			}
		}
	}

	// Project name match: filter "ailang" matches "/Users/mark/dev/sunholo/ailang"
	// Check if path ends with /filter or /filter/something
	if strings.HasSuffix(fullPath, "/"+filter) {
		return true
	}
	if strings.Contains(fullPath, "/"+filter+"/") {
		return true
	}

	// Direct substring match for simple cases
	return strings.Contains(fullPath, filter)
}

// InferInboxSourceType derives source_type from inbox message fields.
// This uses the SAME taxonomy as GetBreakdownBySourceType for consistency.
//
// Canonical source types (matching breakdown):
// - "github": Messages from GitHub sync/webhooks
// - "eval": Messages from eval suite
// - "coordinator": Messages from/to coordinator agents
// - "user_session": Messages from user/dashboard (manual sends)
// - "messaging": General agent-to-agent messaging
// - "cli": CLI-related messages (rarely used for inbox)
// - "other": Catch-all
func InferInboxSourceType(fromAgent, toInbox string) string {
	fromLower := strings.ToLower(fromAgent)
	toLower := strings.ToLower(toInbox)

	// Priority order matches GetBreakdownBySourceType logic
	switch {
	// 1. GitHub messages
	case strings.Contains(fromLower, "github") || strings.HasPrefix(fromLower, "gh-"):
		return "github"

	// 2. Eval suite messages
	case strings.Contains(fromLower, "eval") || strings.Contains(toLower, "eval"):
		return "eval"

	// 3. Coordinator agent messages
	case isCoordinatorInbox(fromLower) || isCoordinatorInbox(toLower):
		return "coordinator"

	// 4. User-initiated messages
	case fromLower == "user" || fromLower == "dashboard" || fromLower == "claude-code" || toLower == "user":
		return "user_session"

	// 5. CLI messages (rare for inbox)
	case strings.HasPrefix(fromLower, "ailang"):
		return "cli"

	// 6. Default: messaging between agents
	case fromAgent != "" && toInbox != "":
		return "messaging"

	default:
		return "other"
	}
}

// isCoordinatorInbox checks if an inbox name belongs to coordinator
func isCoordinatorInbox(inbox string) bool {
	coordinatorInboxes := map[string]bool{
		"coordinator":        true,
		"design-doc-creator": true,
		"sprint-planner":     true,
		"sprint-executor":    true,
		"eval-runner":        true,
	}
	return coordinatorInboxes[inbox]
}

// inboxMessageMatchesSourceType checks if an inbox message matches a source_type filter.
func inboxMessageMatchesSourceType(fromAgent, toInbox, sourceType string) bool {
	if sourceType == "" {
		return true
	}
	return InferInboxSourceType(fromAgent, toInbox) == sourceType
}

// sortEvents sorts unified events by the specified field and order.
// Supported sort fields: timestamp (default), turns, cost, tokens, duration
// When sorting by turns/cost/tokens, inbox messages (which have 0 values) sort to the end.
func sortEvents(events []UnifiedEvent, sortBy, order string) {
	descending := order != "asc"

	sort.Slice(events, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "turns":
			less = events[i].TurnCount < events[j].TurnCount
		case "cost":
			less = events[i].CostUSD < events[j].CostUSD
		case "tokens":
			totalI := events[i].TokensIn + events[i].TokensOut
			totalJ := events[j].TokensIn + events[j].TokensOut
			less = totalI < totalJ
		case "duration":
			less = events[i].DurationMs < events[j].DurationMs
		default: // timestamp
			ti, _ := time.Parse(time.RFC3339, events[i].CreatedAt)
			tj, _ := time.Parse(time.RFC3339, events[j].CreatedAt)
			less = ti.Before(tj)
		}
		if descending {
			return !less
		}
		return less
	})
}

package observatory

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// ===== Aggregate Handlers =====

// handleGetMetricsSummary returns global aggregates.
// DEPRECATED: Use /api/controlplane/stats instead, which provides
// unified Observatory + Coordinator stats with filtering options.
// This endpoint will be removed in a future version.
func (a *API) handleGetMetricsSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := a.backend.GetMetricsSummary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (a *API) handleGetProviderComparison(w http.ResponseWriter, r *http.Request) {
	comparison, err := a.backend.GetProviderComparison(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, comparison)
}

func (a *API) handleGetTaskTimeline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	timeline, err := a.backend.GetTaskTimeline(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, timeline)
}

// ===== Hierarchy Handler (M-TASK-HIERARCHY) =====

// enrichHierarchySpans adds display_name to spans in a hierarchy using session_tools data.
// This correlates OTEL spans with hook-captured tool metadata for richer display.
func (a *API) enrichHierarchySpans(ctx context.Context, hierarchy *TaskHierarchy) {
	if hierarchy == nil {
		return
	}

	sqliteBackend, ok := a.backend.(*SQLiteBackend)
	if !ok {
		return
	}

	// Collect all spans from the hierarchy structure
	// TaskHierarchy -> AgentHierarchy -> TraceHierarchy -> SpanNode
	var allSpans []*Span
	var collectFromSpanNodes func(nodes []*SpanNode)
	collectFromSpanNodes = func(nodes []*SpanNode) {
		for _, node := range nodes {
			if node != nil && node.Span != nil {
				allSpans = append(allSpans, node.Span)
			}
			if node != nil && len(node.Children) > 0 {
				collectFromSpanNodes(node.Children)
			}
		}
	}

	// Traverse the hierarchy
	for _, agent := range hierarchy.Agents {
		if agent == nil {
			continue
		}
		for _, trace := range agent.Traces {
			if trace == nil {
				continue
			}
			// Collect from flat spans list
			collectFromSpanNodes(trace.Spans)
			// Also collect from root span if present
			if trace.RootSpan != nil {
				collectFromSpanNodes([]*SpanNode{trace.RootSpan})
			}
		}
	}

	if len(allSpans) == 0 {
		return
	}

	// Find time range
	var minTime, maxTime time.Time
	for _, span := range allSpans {
		if minTime.IsZero() || span.StartTime.Before(minTime) {
			minTime = span.StartTime
		}
		endTime := span.StartTime.Add(time.Duration(span.DurationMs) * time.Millisecond)
		if maxTime.IsZero() || endTime.After(maxTime) {
			maxTime = endTime
		}
	}

	// Expand time window
	minTime = minTime.Add(-30 * time.Second)
	maxTime = maxTime.Add(30 * time.Second)

	// Fetch all tools in range
	tools, _ := sqliteBackend.store.GetToolsByTimestampRange(ctx, minTime, maxTime, "")

	// Enrich spans with display_name
	for _, span := range allSpans {
		if span == nil || span.DisplayName != "" {
			continue
		}

		toolName := extractToolNameFromSpan(span)
		if toolName == "" {
			// Try fallback display name extraction
			span.DisplayName = extractDisplayNameFromSpan(span)
			continue
		}

		// Find matching tool
		displayName := findMatchingToolByTimestamp(span, tools, toolName)
		if displayName != "" {
			span.DisplayName = displayName
		} else {
			span.DisplayName = extractDisplayNameFromSpan(span)
		}
	}
}

// enrichHierarchySpansWithChat populates ChatContext for all spans in a TaskHierarchy.
// Reuses enrichSpansWithChat from api_spans.go which queries chat_messages by session.id + time range.
func (a *API) enrichHierarchySpansWithChat(ctx context.Context, hierarchy *TaskHierarchy) {
	if hierarchy == nil {
		return
	}

	// Collect all spans from hierarchy (same traversal as enrichHierarchySpans)
	var allSpans []*Span
	var collectFromSpanNodes func(nodes []*SpanNode)
	collectFromSpanNodes = func(nodes []*SpanNode) {
		for _, node := range nodes {
			if node != nil && node.Span != nil {
				allSpans = append(allSpans, node.Span)
			}
			if node != nil && len(node.Children) > 0 {
				collectFromSpanNodes(node.Children)
			}
		}
	}

	for _, agent := range hierarchy.Agents {
		if agent == nil {
			continue
		}
		for _, trace := range agent.Traces {
			if trace == nil {
				continue
			}
			collectFromSpanNodes(trace.Spans)
			if trace.RootSpan != nil {
				collectFromSpanNodes([]*SpanNode{trace.RootSpan})
			}
		}
	}

	if len(allSpans) == 0 {
		return
	}

	// Reuse the existing enrichment logic from api_spans.go
	a.enrichSpansWithChat(ctx, allSpans)
}

func (a *API) handleGetTaskHierarchy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Parse query parameters
	opts := DefaultHierarchyOptions()

	if depth := r.URL.Query().Get("depth"); depth != "" {
		if d, err := strconv.Atoi(depth); err == nil && d > 0 {
			opts.MaxDepth = d
		}
	}

	if includeSpans := r.URL.Query().Get("include_spans"); includeSpans == "false" {
		opts.IncludeSpans = false
	}

	// Parse workspace filter to prevent cross-workspace span bleeding
	if workspace := r.URL.Query().Get("workspace"); workspace != "" {
		opts.Workspace = workspace
	}

	hierarchy, err := GetTaskHierarchy(r.Context(), a.backend, id, opts)
	if err != nil {
		if isNotFoundError(err) {
			// Fallback: Try Claude Code hierarchy if task not found
			// This handles the case where "id" is a Claude Code span_id rather than a task_id
			if sqliteBackend, ok := a.backend.(*SQLiteBackend); ok {
				ccHierarchy, ccErr := sqliteBackend.GetClaudeCodeHierarchy(r.Context(), id)
				if ccErr == nil {
					a.enrichHierarchySpans(r.Context(), ccHierarchy)
					if r.URL.Query().Get("include_chat") == "true" {
						a.enrichHierarchySpansWithChat(r.Context(), ccHierarchy)
					}
					writeJSON(w, http.StatusOK, ccHierarchy)
					return
				}
			}
			writeError(w, http.StatusNotFound, "task not found: "+id)
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	a.enrichHierarchySpans(r.Context(), hierarchy)

	// Enrich with chat context if requested
	if r.URL.Query().Get("include_chat") == "true" {
		a.enrichHierarchySpansWithChat(r.Context(), hierarchy)
	}

	writeJSON(w, http.StatusOK, hierarchy)
}

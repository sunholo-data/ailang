package observatory

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ===== Telemetry Ingest Handlers =====

func (a *API) handleIngestClaude(w http.ResponseWriter, r *http.Request) {
	var metrics ClaudeMetrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	normalizer := NewProviderNormalizer()
	span, err := normalizer.NormalizeClaudeMetrics(&metrics)
	if err != nil {
		writeError(w, http.StatusBadRequest, "normalization failed: "+err.Error())
		return
	}

	if err := a.backend.CreateSpan(r.Context(), span); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create span events
	for _, event := range span.Events {
		if err := a.backend.CreateSpanEvent(r.Context(), &event); err != nil {
			// Log but don't fail - span was created
			fmt.Printf("warning: failed to create span event: %v\n", err)
		}
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"span_id":  span.ID,
		"trace_id": span.TraceID,
	})
}

func (a *API) handleIngestOTEL(w http.ResponseWriter, r *http.Request) {
	// Parse OTEL spans (could be single or array)
	body, err := json.Marshal(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	spans, err := ParseGeminiTraceJSON(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid OTEL JSON: "+err.Error())
		return
	}

	// Get task ID from query param or header
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		taskID = r.Header.Get("X-Task-ID")
	}

	normalizer := NewProviderNormalizer()
	normalized, err := normalizer.NormalizeGeminiTrace(spans, taskID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "normalization failed: "+err.Error())
		return
	}

	var spanIDs []string
	for _, span := range normalized {
		if err := a.backend.CreateSpan(r.Context(), span); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		spanIDs = append(spanIDs, span.ID)

		// Create span events
		for _, event := range span.Events {
			if err := a.backend.CreateSpanEvent(r.Context(), &event); err != nil {
				fmt.Printf("warning: failed to create span event: %v\n", err)
			}
		}
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"span_ids": spanIDs,
		"count":    len(spanIDs),
	})
}

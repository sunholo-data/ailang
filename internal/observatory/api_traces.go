package observatory

import (
	"net/http"
	"strconv"
	"time"
)

// ===== Trace Handlers =====

func (a *API) handleListTraces(w http.ResponseWriter, r *http.Request) {
	opts := TraceQuery{
		TaskID:  r.URL.Query().Get("task_id"),
		TraceID: r.URL.Query().Get("trace_id"),
	}

	// Parse time range
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")
	if startTimeStr != "" || endTimeStr != "" {
		opts.TimeRange = &TimeRange{}
		if startTimeStr != "" {
			if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
				opts.TimeRange.Start = t
			}
		}
		if endTimeStr != "" {
			if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
				opts.TimeRange.End = t
			}
		}
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			opts.Limit = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			opts.Offset = o
		}
	}

	traces, err := a.backend.ListTraces(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, traces)
}

func (a *API) handleGetTrace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	trace, err := a.backend.GetTrace(r.Context(), id)
	if err != nil {
		if isNotFoundError(err) {
			writeError(w, http.StatusNotFound, "trace not found: "+id)
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, trace)
}

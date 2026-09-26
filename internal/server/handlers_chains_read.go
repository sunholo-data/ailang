package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/observatory"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// chainPageParameter rejects supplied invalid values rather than silently changing the page.
func chainPageParameter(q url.Values, name string, fallback, minimum int) (int, error) {
	values, supplied := q[name]
	if !supplied {
		return fallback, nil
	}
	if len(values) != 1 {
		return 0, fmt.Errorf("%s must be supplied once", name)
	}
	n, err := strconv.Atoi(values[0])
	if err != nil || n < minimum {
		return 0, fmt.Errorf("%s must be an integer >= %d", name, minimum)
	}
	return n, nil
}

// writeChainReadError exposes stable categories without backend URLs or credentials.
func writeChainReadError(w http.ResponseWriter, err error) {
	httpStatus, code, message, retryable := http.StatusInternalServerError, "query_failed", "Could not read work data", false
	switch {
	case errors.Is(err, observatory.ErrNotFound), status.Code(err) == codes.NotFound:
		httpStatus, code, message = http.StatusNotFound, "not_found", "Requested work evidence was not found"
	case status.Code(err) == codes.FailedPrecondition:
		httpStatus, code, message, retryable = http.StatusServiceUnavailable, "query_not_ready", "Work query is not ready; required backend indexes may be unavailable", true
	case status.Code(err) == codes.PermissionDenied:
		httpStatus, code, message = http.StatusForbidden, "permission_denied", "Work query permission denied"
	case status.Code(err) == codes.Unauthenticated:
		httpStatus, code, message = http.StatusUnauthorized, "unauthenticated", "Work query authentication required"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled), status.Code(err) == codes.Unavailable, status.Code(err) == codes.DeadlineExceeded, status.Code(err) == codes.ResourceExhausted, status.Code(err) == codes.Canceled:
		httpStatus, code, message, retryable = http.StatusServiceUnavailable, "backend_unavailable", "Work data backend is temporarily unavailable", true
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": code, "message": message, "retryable": retryable})
}

// readOwnedStage verifies the complete path before loading any evidence.
func (s *Server) readOwnedStage(w http.ResponseWriter, r *http.Request, suffix string) (*observatory.ChainStage, bool) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/chains/"), "/")
	if len(parts) != 4 || parts[0] == "" || parts[1] != "stages" || parts[2] == "" || parts[3] != suffix {
		http.Error(w, "Invalid stage path", http.StatusBadRequest)
		return nil, false
	}
	stage, err := s.obsBackend.GetStage(r.Context(), parts[2])
	if err != nil {
		writeChainReadError(w, err)
		return nil, false
	}
	if stage == nil || stage.ChainID != parts[0] {
		writeChainReadError(w, observatory.ErrNotFound)
		return nil, false
	}
	return stage, true
}

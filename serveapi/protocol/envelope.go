package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

var ErrCallbackCapacity = errors.New("host callback capacity exceeded")

func RequestID(body []byte) json.RawMessage {
	var request struct {
		ID json.RawMessage `json:"id"`
	}
	if json.Unmarshal(body, &request) != nil || len(request.ID) == 0 || !json.Valid(request.ID) {
		return json.RawMessage("null")
	}
	return append(json.RawMessage(nil), request.ID...)
}
func CallbackMessage(err error) string {
	switch {
	case errors.Is(err, ErrCallbackCapacity):
		return "host callback capacity exceeded"
	case errors.Is(err, context.DeadlineExceeded):
		return "host callback timed out"
	case errors.Is(err, context.Canceled):
		return "host callback canceled"
	default:
		return "host callback failed"
	}
}
func WriteMCPEnvelope(w http.ResponseWriter, id json.RawMessage, message string) {
	if len(id) == 0 || !json.Valid(id) {
		id = json.RawMessage("null")
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Error   struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{JSONRPC: "2.0", ID: id, Error: struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{Code: -32603, Message: message}})
}
func AuthorizationStatus(err error) int {
	var statusError interface{ HTTPStatus() int }
	if errors.As(err, &statusError) {
		status := statusError.HTTPStatus()
		if status == http.StatusUnauthorized || status == http.StatusForbidden {
			return status
		}
	}
	return 0
}

package protocol

import (
	"encoding/json"
	"net/http"
)

type A2ARequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	ID      json.RawMessage `json:"id"`
	Params  json.RawMessage `json:"params"`
}
type A2ATaskSendParams struct {
	ID       string         `json:"id"`
	Message  A2AMessage     `json:"message"`
	Metadata map[string]any `json:"metadata"`
}
type A2AMessage struct {
	Role  string       `json:"role"`
	Parts []A2AContent `json:"parts"`
}
type A2AContent struct {
	Type string         `json:"type"`
	Text string         `json:"text,omitempty"`
	Data map[string]any `json:"data,omitempty"`
}

func A2AError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	resp := map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": msg}}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
func A2AResult(w http.ResponseWriter, id json.RawMessage, result any) {
	resp := map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

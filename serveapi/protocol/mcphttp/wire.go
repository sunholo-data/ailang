package mcphttp

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strings"
)

// JSON-RPC 2.0 error codes used on this surface.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
)

// MaxRequestBodyBytes caps a POST body. It matches the value the MCP SDK's
// streamable handler used before this package replaced it (4 MiB).
const MaxRequestBodyBytes = 4 << 20

// message is one inbound JSON-RPC object. A request has a method and an id; a
// notification has a method and no id; a response has neither method nor params.
type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

func (m message) isRequest() bool      { return m.Method != "" && len(m.ID) > 0 }
func (m message) isNotification() bool { return m.Method != "" && len(m.ID) == 0 }
func (m message) isResponse() bool {
	return m.Method == "" && len(m.ID) > 0 && (len(m.Result) > 0 || len(m.Error) > 0)
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

func resultResponse(id json.RawMessage, result any) response {
	return response{JSONRPC: "2.0", ID: id, Result: result}
}

func errorResponse(id json.RawMessage, code int, format string, args ...any) response {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: fmt.Sprintf(format, args...)}}
}

// label sets the headers every response on this surface carries (#603): an
// explicit Content-Type and nosniff, so no reflecting body is ever sniffed.
func label(w http.ResponseWriter, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// writeTransportError answers a request the transport refuses before dispatch.
func writeTransportError(w http.ResponseWriter, status int, resp response) {
	label(w, "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// writeSSE answers with a single `message` event carrying payload, then ends
// the stream: this server is stateless and never sends anything else.
func writeSSE(w http.ResponseWriter, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		writeTransportError(w, http.StatusInternalServerError,
			errorResponse(nil, -32603, "encode response: %v", err))
		return
	}
	label(w, "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
}

func writeAccepted(w http.ResponseWriter) {
	label(w, "application/json")
	w.WriteHeader(http.StatusAccepted)
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	label(w, "text/plain; charset=utf-8")
	w.Header().Set("Allow", http.MethodPost)
	w.WriteHeader(http.StatusMethodNotAllowed)
	_, _ = fmt.Fprintln(w, "Method Not Allowed")
}

// acceptsBoth reports whether the Accept header lists both media types the
// Streamable HTTP transport requires of a POST.
func acceptsBoth(r *http.Request) bool {
	var json, sse bool
	for _, value := range r.Header.Values("Accept") {
		for _, part := range strings.Split(value, ",") {
			mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(part))
			if err != nil {
				continue
			}
			switch mediaType {
			case "application/json":
				json = true
			case "text/event-stream":
				sse = true
			}
		}
	}
	return json && sse
}

func isJSONContent(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mediaType == "application/json"
}

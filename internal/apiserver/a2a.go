package apiserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/embed"
)

// handleA2AAgentCard serves the A2A Agent Card at /.well-known/agent.json.
func (s *Server) handleA2AAgentCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET only"})
		return
	}

	card := s.buildAgentCard(r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(card)
}

// buildAgentCard creates an A2A Agent Card from loaded modules.
func (s *Server) buildAgentCard(r *http.Request) map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build skills list from loaded modules.
	var skills []map[string]any

	modPaths := make([]string, 0, len(s.modules))
	for k := range s.modules {
		modPaths = append(modPaths, k)
	}
	sort.Strings(modPaths)

	for _, modPath := range modPaths {
		modInfo := s.modules[modPath]
		for _, export := range modInfo.Exports {
			if export.Arity < 0 {
				continue
			}

			skillID := modPath + "." + export.Name
			skillID = strings.ReplaceAll(skillID, "/", ".")

			desc := export.Type
			if export.Pure {
				desc += " [pure]"
			}

			tags := []string{modPath}
			if export.Pure {
				tags = append(tags, "pure")
			}

			skills = append(skills, map[string]any{
				"id":          skillID,
				"name":        export.Name,
				"description": desc,
				"tags":        tags,
				"examples":    []string{},
			})
		}
	}

	// Determine server URL from request.
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	serverURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	return map[string]any{
		"name":        "AILANG Function Server",
		"description": "AILANG module exports as callable functions via A2A protocol",
		"url":         serverURL,
		"version":     "0.8.1",
		"capabilities": map[string]any{
			"streaming":              false,
			"pushNotifications":      false,
			"stateTransitionHistory": false,
		},
		"defaultInputModes":  []string{"application/json"},
		"defaultOutputModes": []string{"application/json"},
		"skills":             skills,
	}
}

// handleA2ATask handles JSON-RPC 2.0 requests at /a2a/.
func (s *Server) handleA2ATask(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST only"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		a2aError(w, nil, -32700, "failed to read request body")
		return
	}

	var req a2aRequest
	if err := json.Unmarshal(body, &req); err != nil {
		a2aError(w, nil, -32700, "invalid JSON: "+err.Error())
		return
	}

	if req.JSONRPC != "2.0" {
		a2aError(w, req.ID, -32600, "expected jsonrpc: \"2.0\"")
		return
	}

	switch req.Method {
	case "tasks/send":
		s.handleA2ATaskSend(w, &req)
	case "tasks/get":
		s.handleA2ATaskGet(w, &req)
	default:
		a2aError(w, req.ID, -32601, "method not found: "+req.Method)
	}
}

// handleA2ATaskSend handles tasks/send - executes an AILANG function.
func (s *Server) handleA2ATaskSend(w http.ResponseWriter, req *a2aRequest) {
	var params a2aTaskSendParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		a2aError(w, req.ID, -32602, "invalid params: "+err.Error())
		return
	}

	// Determine which function to call from metadata.skill_id or message text.
	skillID := ""
	if params.Metadata != nil {
		if sid, ok := params.Metadata["skill_id"].(string); ok {
			skillID = sid
		}
	}

	if skillID == "" {
		a2aError(w, req.ID, -32602, "missing metadata.skill_id (format: module.path.function)")
		return
	}

	// Parse skill_id into module path and function name.
	// e.g., "api.math.add" → module="api/math", func="add"
	parts := strings.Split(skillID, ".")
	if len(parts) < 2 {
		a2aError(w, req.ID, -32602, "invalid skill_id format: expected module.function")
		return
	}

	funcName := parts[len(parts)-1]
	modulePath := strings.Join(parts[:len(parts)-1], "/")

	// Verify module and function exist.
	s.mu.RLock()
	modInfo, ok := s.modules[modulePath]
	s.mu.RUnlock()
	if !ok {
		a2aError(w, req.ID, -32602, fmt.Sprintf("module %q not loaded", modulePath))
		return
	}

	var found bool
	for _, e := range modInfo.Exports {
		if e.Name == funcName {
			found = true
			break
		}
	}
	if !found {
		a2aError(w, req.ID, -32602, fmt.Sprintf("function %q not found in module %q", funcName, modulePath))
		return
	}

	// Extract arguments from message text (try JSON array).
	var args []any
	if params.Message.Parts != nil {
		for _, part := range params.Message.Parts {
			if part.Type == "data" && part.Data != nil {
				if argsSlice, ok := part.Data["args"].([]any); ok {
					args = argsSlice
				}
			}
		}
	}

	// Call the function.
	result, callErr := s.engine.Call(modulePath, funcName, args...)

	taskID := params.ID
	if taskID == "" {
		taskID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}

	if callErr != nil {
		a2aResult(w, req.ID, map[string]any{
			"id":     taskID,
			"status": map[string]any{"state": "failed", "message": callErr.Error()},
		})
		return
	}

	goResult, err := embed.ToGo(result)
	if err != nil {
		a2aResult(w, req.ID, map[string]any{
			"id":     taskID,
			"status": map[string]any{"state": "failed", "message": "result conversion: " + err.Error()},
		})
		return
	}

	resultJSON, _ := json.Marshal(goResult)
	a2aResult(w, req.ID, map[string]any{
		"id":     taskID,
		"status": map[string]any{"state": "completed"},
		"artifacts": []map[string]any{
			{
				"parts": []map[string]any{
					{"type": "data", "data": map[string]any{"result": goResult}},
					{"type": "text", "text": string(resultJSON)},
				},
			},
		},
	})
}

// handleA2ATaskGet handles tasks/get - returns task status.
// Since all tasks are synchronous, this always returns "not found".
func (s *Server) handleA2ATaskGet(w http.ResponseWriter, req *a2aRequest) {
	a2aError(w, req.ID, -32602, "tasks/get not supported (all tasks complete synchronously)")
}

// A2A JSON-RPC types.

type a2aRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	ID      json.RawMessage `json:"id"`
	Params  json.RawMessage `json:"params"`
}

type a2aTaskSendParams struct {
	ID       string         `json:"id"`
	Message  a2aMessage     `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type a2aMessage struct {
	Role  string       `json:"role"`
	Parts []a2aContent `json:"parts"`
}

type a2aContent struct {
	Type string         `json:"type"`
	Text string         `json:"text,omitempty"`
	Data map[string]any `json:"data,omitempty"`
}

// a2aError writes a JSON-RPC 2.0 error response.
func a2aError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]any{"code": code, "message": msg},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC always returns 200.
	_ = json.NewEncoder(w).Encode(resp)
}

// a2aResult writes a JSON-RPC 2.0 success response.
func a2aResult(w http.ResponseWriter, id json.RawMessage, result any) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

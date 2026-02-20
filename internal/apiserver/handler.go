package apiserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/embed"
)

// FunctionCallRequest is the JSON body for calling an AILANG function.
// Use "args" for positional arguments (array), or provide a flat JSON object
// which is passed as a single record argument.
type FunctionCallRequest struct {
	Args []interface{} `json:"args,omitempty"`
}

// FunctionCallResponse is the JSON response from a function call.
type FunctionCallResponse struct {
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	Module    string      `json:"module"`
	Func      string      `json:"func"`
	ElapsedMs int64       `json:"elapsed_ms"`
}

// handleFunctionCall is the generic handler for calling any AILANG exported function.
// URL pattern: POST /api/{modulePath}/{functionName}
func (s *Server) handleFunctionCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, FunctionCallResponse{
			Error: "use POST with JSON body to call functions",
		})
		return
	}

	// Parse module path and function name from URL
	// /api/ecommerce/api/handlers/successResponse
	// → module: "ecommerce/api/handlers", func: "successResponse"
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	path = strings.TrimSuffix(path, "/")

	if path == "" || strings.HasPrefix(path, "_") {
		writeJSON(w, http.StatusNotFound, FunctionCallResponse{
			Error: "not found",
		})
		return
	}

	// Split into module path and function name (last segment is function)
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash < 0 {
		writeJSON(w, http.StatusBadRequest, FunctionCallResponse{
			Error: "invalid path: expected /api/{module}/{function}",
		})
		return
	}

	modulePath := path[:lastSlash]
	funcName := path[lastSlash+1:]

	// Validate module exists
	s.mu.RLock()
	modInfo, ok := s.modules[modulePath]
	s.mu.RUnlock()

	if !ok {
		writeJSON(w, http.StatusNotFound, FunctionCallResponse{
			Module: modulePath,
			Func:   funcName,
			Error:  fmt.Sprintf("module %q not loaded", modulePath),
		})
		return
	}

	// Validate function exists in module
	var exportInfo *ExportInfo
	for i := range modInfo.Exports {
		if modInfo.Exports[i].Name == funcName {
			exportInfo = &modInfo.Exports[i]
			break
		}
	}
	if exportInfo == nil {
		available := make([]string, len(modInfo.Exports))
		for i, e := range modInfo.Exports {
			available[i] = e.Name
		}
		writeJSON(w, http.StatusNotFound, FunctionCallResponse{
			Module: modulePath,
			Func:   funcName,
			Error:  fmt.Sprintf("function %q not found in module %q (available: %v)", funcName, modulePath, available),
		})
		return
	}

	// Read request body
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		writeJSON(w, http.StatusBadRequest, FunctionCallResponse{
			Module: modulePath,
			Func:   funcName,
			Error:  "failed to read request body",
		})
		return
	}

	// Parse arguments
	args, err := parseArgs(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, FunctionCallResponse{
			Module: modulePath,
			Func:   funcName,
			Error:  fmt.Sprintf("invalid arguments: %v", err),
		})
		return
	}

	// Call AILANG function (preserve floats since JSON has no int/float distinction —
	// 100.0 must remain FloatValue, not become IntValue).
	start := time.Now()
	result, callErr := s.engine.CallPreserveFloats(modulePath, funcName, args...)
	elapsed := time.Since(start).Milliseconds()

	if callErr != nil {
		log.Printf("[API] %s/%s failed: %v", modulePath, funcName, callErr)
		writeJSON(w, http.StatusInternalServerError, FunctionCallResponse{
			Module:    modulePath,
			Func:      funcName,
			Error:     callErr.Error(),
			ElapsedMs: elapsed,
		})
		return
	}

	// Convert result to Go value
	goResult, err := embed.ToGo(result)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, FunctionCallResponse{
			Module:    modulePath,
			Func:      funcName,
			Error:     fmt.Sprintf("result conversion failed: %v", err),
			ElapsedMs: elapsed,
		})
		return
	}

	writeJSON(w, http.StatusOK, FunctionCallResponse{
		Module:    modulePath,
		Func:      funcName,
		Result:    goResult,
		ElapsedMs: elapsed,
	})
}

// parseArgs extracts function arguments from the JSON body.
// Supports two formats:
//  1. {"args": [arg1, arg2, ...]} - positional arguments
//  2. Any other JSON value - passed as a single argument
//  3. Empty body - no arguments
func parseArgs(body []byte) ([]interface{}, error) {
	if len(body) == 0 {
		return nil, nil
	}

	// Try the structured {"args": [...]} format first
	var req FunctionCallRequest
	if err := json.Unmarshal(body, &req); err == nil && req.Args != nil {
		return req.Args, nil
	}

	// Otherwise, try to parse as a raw JSON value and pass as single arg
	var singleArg interface{}
	if err := json.Unmarshal(body, &singleArg); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return []interface{}{singleArg}, nil
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[API] failed to write response: %v", err)
	}
}

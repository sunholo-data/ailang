package apiserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	if r.Method != "POST" && r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, FunctionCallResponse{
			Error: "use GET with query params or POST with JSON body to call functions",
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

	// Validate function exists in module and get param names
	var foundExport *ExportInfo
	for i := range modInfo.Exports {
		if modInfo.Exports[i].Name == funcName {
			foundExport = &modInfo.Exports[i]
			break
		}
	}
	if foundExport == nil {
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

	// Delegate to shared function caller with param names for named binding
	s.callFunction(w, r, modulePath, funcName, callOpts{ParamNames: foundExport.ParamNames})
}

// camelToSnake converts a camelCase string to snake_case.
// e.g., "outputFormat" -> "output_format", "maxSize" -> "max_size"
func camelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(r + ('a' - 'A'))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// parseNamedArgs maps JSON object keys to function parameter names and returns
// positional args in parameter order. Returns nil if no parameters match.
//
// Matching rules:
//  1. Exact match: JSON key matches param name exactly
//  2. Snake-case match: JSON key matches camelToSnake(paramName)
//
// Unmatched JSON keys are silently ignored (forward-compatible).
func parseNamedArgs(body map[string]interface{}, paramNames []string) []interface{} {
	if len(paramNames) == 0 {
		return nil
	}
	args := make([]interface{}, len(paramNames))
	matched := 0
	for i, name := range paramNames {
		// Try exact match first
		if val, ok := body[name]; ok {
			args[i] = val
			matched++
			continue
		}
		// Try snake_case version of camelCase param name
		snake := camelToSnake(name)
		if snake != name {
			if val, ok := body[snake]; ok {
				args[i] = val
				matched++
				continue
			}
		}
	}
	if matched == 0 {
		return nil // no matches, caller should fall back
	}
	return args
}

// parseArgsWithNames tries named JSON parameter binding before falling back to parseArgs.
//
// Precedence:
//  1. {"args": [...]} — positional (existing behavior)
//  2. JSON object with keys matching paramNames — named binding
//  3. Any other JSON value — single argument (existing behavior)
func parseArgsWithNames(body []byte, paramNames []string) ([]interface{}, error) {
	if len(body) == 0 || len(paramNames) == 0 {
		return parseArgs(body)
	}

	// Quick check: try structured {"args": [...]} first (backward compat)
	var req FunctionCallRequest
	if err := json.Unmarshal(body, &req); err == nil && req.Args != nil {
		return req.Args, nil
	}

	// Try named binding: parse as JSON object and match keys to param names
	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err == nil && len(obj) > 0 {
		if named := parseNamedArgs(obj, paramNames); named != nil {
			return named, nil
		}
	}

	// Fall back to single-arg parsing
	return parseArgs(body)
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

// parseQueryArgs extracts function arguments from URL query parameters.
// Supports two conventions:
//  1. Positional: ?args=val1&args=val2 → ["val1", "val2"]
//  2. Named: ?name=Alice&age=30 → [{name: "Alice", age: 30}] (single record arg)
func parseQueryArgs(query url.Values) []interface{} {
	if len(query) == 0 {
		return nil
	}

	// Convention 1: positional via ?args=...&args=...
	if positional, ok := query["args"]; ok {
		result := make([]interface{}, len(positional))
		for i, a := range positional {
			result[i] = tryParseJSON(a)
		}
		return result
	}

	// Convention 2: named params → single record arg
	record := make(map[string]interface{})
	for key, values := range query {
		if len(values) == 1 {
			record[key] = tryParseJSON(values[0])
		} else {
			parsed := make([]interface{}, len(values))
			for i, v := range values {
				parsed[i] = tryParseJSON(v)
			}
			record[key] = parsed
		}
	}
	if len(record) > 0 {
		return []interface{}{record}
	}
	return nil
}

// tryParseJSON attempts to parse a string as a JSON value (number, bool, null).
// Falls back to returning the string as-is.
func tryParseJSON(s string) interface{} {
	// Try integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return float64(i) // JSON numbers are float64
	}
	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	// Try bool
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	// Try JSON object/array
	if (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) {
		var v interface{}
		if err := json.Unmarshal([]byte(s), &v); err == nil {
			return v
		}
	}
	return s
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[API] failed to write response: %v", err)
	}
}

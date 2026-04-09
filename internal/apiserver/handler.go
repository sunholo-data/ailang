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
//
// For error responses, the flat `Error` field is preserved for backward
// compatibility, and new clients should prefer the structured `ErrorDetail`
// envelope which carries a stable error code and optional did-you-mean hints.
// The flat `Error` string always mirrors `ErrorDetail.Message` when both are set.
type FunctionCallResponse struct {
	Result      interface{}        `json:"result,omitempty"`
	Error       string             `json:"error,omitempty"`
	ErrorDetail *RouterErrorDetail `json:"error_detail,omitempty"`
	Module      string             `json:"module"`
	Func        string             `json:"func"`
	ElapsedMs   int64              `json:"elapsed_ms"`
}

// handleFunctionCall is the generic handler for calling any AILANG exported function.
// URL pattern: POST /api/{modulePath}/{functionName}
func (s *Server) handleFunctionCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" && r.Method != "GET" {
		writeRouterError(w, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"use GET with query params or POST with JSON body to call functions",
			"", nil)
		return
	}

	// Parse module path and function name from URL
	// /api/ecommerce/api/handlers/successResponse
	// → module: "ecommerce/api/handlers", func: "successResponse"
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	path = strings.TrimSuffix(path, "/")

	if path == "" || strings.HasPrefix(path, "_") {
		writeRouterError(w, http.StatusNotFound, ErrCodeRouteNotFound, "not found", "", nil)
		return
	}

	// Split into module path and function name (last segment is function)
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash < 0 {
		writeRouterError(w, http.StatusBadRequest,
			ErrCodeRouteNotFound,
			"invalid path: expected /api/{module}/{function}",
			"", nil)
		return
	}

	modulePath := path[:lastSlash]
	funcName := path[lastSlash+1:]

	// Validate module exists. s.modules is keyed by PhysicalPath; look
	// up by the URL-shaped RelPath projection instead.
	s.mu.RLock()
	modInfo, ok := s.findModuleByRelPath(modulePath)
	s.mu.RUnlock()

	if !ok {
		// Fallback 1: check if the URL matches a custom @route from a package
		// module. Package modules use paths like "pkg/owner/repo/mod" which
		// don't match the URL-based module/function parsing above.
		if route := s.findRouteByPath(r.URL.Path); route != nil {
			if r.Method == route.Method || r.Method == "OPTIONS" {
				s.callFunction(w, r, route.Module, route.Function, callOpts{
					Raw:        route.IsRaw,
					Nowrap:     route.IsNowrap,
					ParamNames: route.ParamNames,
					ParamTypes: route.ParamTypes,
				})
				return
			}
		}

		// 3-way discrimination for the 404 response:
		//
		//   Case A: server has @routes registered and none matched.
		//     → ROUTE_NOT_FOUND with did-you-mean suggestions. This is the
		//       common case for route-driven deployments (e.g. docparse).
		//
		//   Case C: server has zero @routes (legacy module/func-only server)
		//       AND the parsed modulePath doesn't resolve.
		//     → MODULE_NOT_LOADED. Preserves historical behavior for non-
		//       @route deployments that dispatch via /api/{module}/{func}.
		customRoutes := s.getCustomRoutes()
		if len(customRoutes) > 0 {
			suggestedFix, available := suggestRoutes(r.Method, r.URL.Path, customRoutes)
			msg := fmt.Sprintf("No route registered for %s %s", r.Method, r.URL.Path)
			writeRouterError(w, http.StatusNotFound, ErrCodeRouteNotFound, msg, suggestedFix, available)
			return
		}
		// Case C: legacy module/func dispatch on a no-@route server.
		writeRouterErrorWithDispatch(w, http.StatusNotFound,
			ErrCodeModuleNotLoaded,
			fmt.Sprintf("module %q not loaded", modulePath),
			"Ensure the module is reachable from an --entry file or passed via --load",
			modulePath, funcName)
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
		exports := make([]string, len(modInfo.Exports))
		for i, e := range modInfo.Exports {
			exports[i] = e.Name
		}
		writeRouterErrorWithDispatch(w, http.StatusNotFound,
			ErrCodeFunctionNotFound,
			fmt.Sprintf("function %q not found in module %q (available: %v)", funcName, modulePath, exports),
			"",
			modulePath, funcName)
		return
	}

	// Check if function is hidden from HTTP via @noexpose or --routes-only.
	// Intentionally use the same FUNCTION_NOT_FOUND code so @noexpose stays
	// indistinguishable from a genuinely missing function.
	if !s.isExposed(*foundExport) {
		writeRouterErrorWithDispatch(w, http.StatusNotFound,
			ErrCodeFunctionNotFound,
			fmt.Sprintf("function %q not found in module %q", funcName, modulePath),
			"",
			modulePath, funcName)
		return
	}

	// Delegate to shared function caller with param names for named binding
	s.callFunction(w, r, modulePath, funcName, callOpts{ParamNames: foundExport.ParamNames, ParamTypes: foundExport.ParamTypes})
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

// zeroValueForType returns the zero-value for a given AILANG type name.
// Used to pad missing named parameters so functions receive typed defaults
// instead of unit values that crash on type-specific operations.
func zeroValueForType(typeName string) interface{} {
	switch typeName {
	case "string":
		return ""
	case "int":
		return float64(0) // JSON numbers are float64
	case "float":
		return float64(0)
	case "bool":
		return false
	case "list", "array":
		return []interface{}{}
	case "record":
		return map[string]interface{}{}
	default:
		return nil // unknown types remain nil → unit (current behavior)
	}
}

// parseNamedArgs maps JSON object keys to function parameter names and returns
// positional args in parameter order. Returns nil if no parameters match.
//
// Matching rules:
//  1. Exact match: JSON key matches param name exactly
//  2. Snake-case match: JSON key matches camelToSnake(paramName)
//
// Unmatched JSON keys are silently ignored (forward-compatible).
// Missing parameters are padded with type-appropriate zero-values when
// paramTypes is available, enabling functions to validate inputs instead
// of crashing on unit values.
func parseNamedArgs(body map[string]interface{}, paramNames []string, paramTypes []string) []interface{} {
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
		// Pad missing param with type-appropriate zero-value
		if i < len(paramTypes) {
			args[i] = zeroValueForType(paramTypes[i])
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
//  1. {"args": [...]} — positional (existing behavior, with zero-value padding)
//  2. JSON object with keys matching paramNames — named binding
//  3. Any other JSON value — single argument (existing behavior)
func parseArgsWithNames(body []byte, paramNames []string, paramTypes []string) ([]interface{}, error) {
	if len(paramNames) == 0 {
		return parseArgs(body)
	}
	if len(body) == 0 {
		// Empty body + declared params → pad with type zero-values,
		// matching the behavior of POST {} (UnmatchedKeysZeroValuePadding).
		// This lets user code run its normal validation instead of
		// crashing inside builtins on UnitValue or hitting arity errors.
		if len(paramTypes) == 0 {
			return parseArgs(body)
		}
		args := make([]interface{}, len(paramNames))
		for i := range paramNames {
			if i < len(paramTypes) {
				args[i] = zeroValueForType(paramTypes[i])
			}
		}
		return args, nil
	}

	// Quick check: try structured {"args": [...]} first (backward compat)
	var req FunctionCallRequest
	if err := json.Unmarshal(body, &req); err == nil && req.Args != nil {
		// Pad positional args if fewer than expected parameters
		if len(req.Args) < len(paramNames) && len(paramTypes) > 0 {
			padded := make([]interface{}, len(paramNames))
			copy(padded, req.Args)
			for i := len(req.Args); i < len(paramNames); i++ {
				if i < len(paramTypes) {
					padded[i] = zeroValueForType(paramTypes[i])
				}
			}
			return padded, nil
		}
		return req.Args, nil
	}

	// Try named binding: parse as JSON object and match keys to param names
	var obj map[string]interface{}
	bodyIsObject := json.Unmarshal(body, &obj) == nil
	if bodyIsObject && len(obj) > 0 {
		if named := parseNamedArgs(obj, paramNames, paramTypes); named != nil {
			return named, nil
		}
	}

	// If body is a JSON object but no keys matched declared params,
	// return zero-value-padded args instead of passing the raw object as a
	// single argument (which would give e.g. a Record to a string param).
	// Non-object bodies (strings, numbers, arrays) still fall through to parseArgs.
	if bodyIsObject && len(paramNames) > 0 && len(paramTypes) > 0 {
		args := make([]interface{}, len(paramNames))
		for i := range paramNames {
			if i < len(paramTypes) {
				args[i] = zeroValueForType(paramTypes[i])
			}
		}
		return args, nil
	}

	// Fall back to single-arg parsing (non-object body or no declared params)
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
	// Only try numeric parsing if the string looks like a plain number.
	// Go's strconv.ParseFloat accepts underscores as digit separators
	// (e.g., "2026_04" → 202604.0), which silently corrupts string args
	// that happen to contain underscores between digits.
	if looksNumeric(s) {
		// Try integer
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return float64(i) // JSON numbers are float64
		}
		// Try float
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
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

// looksNumeric returns true if s looks like a plain JSON number
// (digits, optional leading minus, optional decimal point, optional exponent).
// Rejects strings with underscores, letters (other than e/E for exponent), etc.
func looksNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, c := range s {
		switch {
		case c >= '0' && c <= '9':
			continue
		case c == '-' && i == 0:
			continue
		case c == '+' && i > 0:
			continue
		case c == '.':
			continue
		case (c == 'e' || c == 'E') && i > 0:
			continue
		default:
			return false
		}
	}
	return true
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[API] failed to write response: %v", err)
	}
}

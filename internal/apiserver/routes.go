package apiserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/embed"
	"github.com/sunholo/ailang/internal/eval"
)

// goroutineID extracts the goroutine ID for debug logging.
func goroutineID() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	s := string(buf[:n])
	s = strings.TrimPrefix(s, "goroutine ")
	var id int
	fmt.Sscanf(s, "%d", &id)
	return id
}

// RouteEntry represents a custom route defined by a @route annotation.
type RouteEntry struct {
	Method     string   // "GET", "POST", etc.
	Path       string   // "/general/v0/general"
	Module     string   // module path
	Function   string   // function name
	IsRaw      bool     // @raw: pass full HttpRequest record instead of parsed args
	IsNowrap   bool     // @nowrap: skip FunctionCallResponse envelope, return raw JSON
	ParamNames []string // parameter names for named JSON binding
	ParamTypes []string // parameter type strings for zero-value padding
}

// extractParamInfo populates ExportInfo.ParamNames and ExportInfo.ParamTypes
// from the parsed AST for all exported functions. This enables named JSON
// parameter binding and zero-value padding for missing parameters.
func extractParamInfo(modInfo *ModuleInfo, file *ast.File) {
	for _, fn := range file.Funcs {
		if !fn.IsExport {
			continue
		}
		names := make([]string, len(fn.Params))
		types := make([]string, len(fn.Params))
		for i, p := range fn.Params {
			names[i] = p.Name
			types[i] = paramTypeToString(p.Type)
		}
		for i := range modInfo.Exports {
			if modInfo.Exports[i].Name == fn.Name {
				modInfo.Exports[i].ParamNames = names
				modInfo.Exports[i].ParamTypes = types
				break
			}
		}
	}
}

// paramTypeToString converts an ast.Type to a simple type name string
// used for zero-value padding of missing parameters.
func paramTypeToString(t ast.Type) string {
	if t == nil {
		return "unknown"
	}
	switch v := t.(type) {
	case *ast.SimpleType:
		return v.Name
	case *ast.ListType:
		return "list"
	case *ast.ArrayType:
		return "array"
	case *ast.RecordType:
		return "record"
	default:
		return "unknown"
	}
}

// extractRouteAnnotations populates ExportInfo.RouteMethod/RoutePath from
// @route annotations found in the parsed AST.
func extractRouteAnnotations(modInfo *ModuleInfo, file *ast.File) {
	for _, fn := range file.Funcs {
		routeAnn := fn.GetAnnotation("route")
		if routeAnn == nil || len(routeAnn.Args) < 2 {
			continue
		}
		methodLit, ok1 := routeAnn.Args[0].(*ast.Literal)
		pathLit, ok2 := routeAnn.Args[1].(*ast.Literal)
		if !ok1 || !ok2 || methodLit.Kind != ast.StringLit || pathLit.Kind != ast.StringLit {
			continue
		}
		method := methodLit.Value.(string)
		path := pathLit.Value.(string)
		isRaw := fn.GetAnnotation("raw") != nil
		isNowrap := fn.GetAnnotation("nowrap") != nil

		// Find matching export and set route info
		for i := range modInfo.Exports {
			if modInfo.Exports[i].Name == fn.Name {
				modInfo.Exports[i].RouteMethod = method
				modInfo.Exports[i].RoutePath = path
				modInfo.Exports[i].IsRaw = isRaw
				modInfo.Exports[i].IsNowrap = isNowrap
				modInfo.Exports[i].IsNoExpose = false // @route overrides @noexpose
				flags := ""
				if isRaw {
					flags += " raw"
				}
				if isNowrap {
					flags += " nowrap"
				}
				if flags != "" {
					log.Printf("    Route: %s %s -> %s (%s)", method, path, fn.Name, strings.TrimSpace(flags))
				} else {
					log.Printf("    Route: %s %s -> %s", method, path, fn.Name)
				}
				break
			}
		}
	}
}

// extractNoExposeAnnotations marks exported functions with @noexpose as hidden
// from HTTP endpoints. Functions with @route are never hidden (route overrides noexpose).
func extractNoExposeAnnotations(modInfo *ModuleInfo, file *ast.File) {
	for _, fn := range file.Funcs {
		if fn.GetAnnotation("noexpose") == nil {
			continue
		}
		for i := range modInfo.Exports {
			if modInfo.Exports[i].Name == fn.Name && modInfo.Exports[i].RoutePath == "" {
				modInfo.Exports[i].IsNoExpose = true
				break
			}
		}
	}
}

// isExposed returns true if the export should be visible as an HTTP endpoint,
// considering the server's routesOnly setting and the export's annotations.
func (s *Server) isExposed(exp ExportInfo) bool {
	if exp.IsNoExpose {
		return false
	}
	if s.routesOnly && exp.RoutePath == "" {
		return false
	}
	return true
}

// isValidJSONObjectOrArray checks if a string is a valid JSON object ({...}) or array ([...]).
// Only these compound types are unwrapped — bare strings, numbers, and booleans are NOT.
func isValidJSONObjectOrArray(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	if s[0] != '{' && s[0] != '[' {
		return false
	}
	return json.Valid([]byte(s))
}

// findRouteByPath finds a custom route matching the given URL path.
// Used as a fallback in the catch-all handler for package module routes.
func (s *Server) findRouteByPath(urlPath string) *RouteEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, mod := range s.modules {
		for _, exp := range mod.Exports {
			if exp.RoutePath == urlPath {
				return &RouteEntry{
					Method:     exp.RouteMethod,
					Path:       exp.RoutePath,
					Module:     mod.Path,
					Function:   exp.Name,
					IsRaw:      exp.IsRaw,
					IsNowrap:   exp.IsNowrap,
					ParamNames: exp.ParamNames,
					ParamTypes: exp.ParamTypes,
				}
			}
		}
	}
	return nil
}

// getCustomRoutes returns all custom routes from loaded modules.
func (s *Server) getCustomRoutes() []RouteEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var routes []RouteEntry
	for _, mod := range s.modules {
		for _, exp := range mod.Exports {
			if exp.RoutePath != "" {
				routes = append(routes, RouteEntry{
					Method:     exp.RouteMethod,
					Path:       exp.RoutePath,
					Module:     mod.Path,
					Function:   exp.Name,
					IsRaw:      exp.IsRaw,
					IsNowrap:   exp.IsNowrap,
					ParamNames: exp.ParamNames,
					ParamTypes: exp.ParamTypes,
				})
			}
		}
	}
	return routes
}

// registerCustomRoutes adds custom route handlers to the mux.
// Custom routes are registered BEFORE the catch-all /api/ handler.
// builtinPaths contains paths already registered by buildRoutes(); any @route
// annotation that collides with a built-in path is skipped with a warning
// (Go 1.22+ ServeMux panics on duplicate patterns).
func (s *Server) registerCustomRoutes(mux *http.ServeMux, builtinPaths map[string]bool) {
	routes := s.getCustomRoutes()
	registered := map[string]bool{} // track registered paths to avoid Go 1.22+ duplicate panics
	for _, route := range routes {
		if builtinPaths[route.Path] {
			log.Printf("  WARNING: @route %s %s collides with built-in route, skipping (use built-in handler instead)", route.Method, route.Path)
			continue
		}
		if registered[route.Path] {
			log.Printf("  WARNING: @route %s %s already registered, skipping duplicate from %s", route.Method, route.Path, route.Module)
			continue
		}
		r := route // capture for closure
		handler := func(w http.ResponseWriter, req *http.Request) {
			// Enforce HTTP method
			if req.Method != r.Method && req.Method != "OPTIONS" {
				writeJSON(w, http.StatusMethodNotAllowed, FunctionCallResponse{
					Error: fmt.Sprintf("this endpoint only accepts %s requests", r.Method),
				})
				return
			}
			s.callFunction(w, req, r.Module, r.Function, callOpts{Raw: r.IsRaw, Nowrap: r.IsNowrap, ParamNames: r.ParamNames, ParamTypes: r.ParamTypes})
		}
		mux.HandleFunc(r.Path, s.corsWrap(s.authMiddleware(handler)))
		registered[route.Path] = true
		log.Printf("  Custom route: %s %s -> %s/%s", r.Method, r.Path, r.Module, r.Function)
	}
}

// callOpts controls per-route behavior for callFunction.
type callOpts struct {
	Raw        bool     // @raw: pass full HttpRequest record instead of parsed args
	Nowrap     bool     // @nowrap: skip FunctionCallResponse envelope, return raw JSON
	ParamNames []string // parameter names for named JSON binding
	ParamTypes []string // parameter type strings for zero-value padding
}

// callFunction executes an AILANG function and writes the response.
// Shared by both the catch-all handler and custom route handlers.
func (s *Server) callFunction(w http.ResponseWriter, r *http.Request, modulePath, funcName string, opts ...callOpts) {
	if os.Getenv("DEBUG_CONCURRENCY") == "1" {
		log.Printf("[CONCURRENCY] callFunction entered: %s/%s (goroutine %d)", modulePath, funcName, goroutineID())
	}
	// Parse arguments based on content type
	var args []interface{}
	var opt callOpts
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.Raw {
		// @raw routes: pass full HttpRequest record instead of parsed args
		body, err := readRequestBody(r, 1<<20) // 1MB limit
		if err != nil {
			writeJSON(w, http.StatusBadRequest, FunctionCallResponse{
				Module: modulePath,
				Func:   funcName,
				Error:  "failed to read request body",
			})
			return
		}
		args = []interface{}{buildHttpRequestRecord(r, body)}
	} else if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		// Handle file uploads via multipart/form-data
		maxSize := s.maxUploadSize
		if maxSize == 0 {
			maxSize = 50 << 20 // 50MB default
		}
		if err := r.ParseMultipartForm(maxSize); err != nil {
			writeJSON(w, http.StatusRequestEntityTooLarge, FunctionCallResponse{
				Module: modulePath,
				Func:   funcName,
				Error:  fmt.Sprintf("multipart parse error: %v", err),
			})
			return
		}
		var cleanup func()
		var parseErr error
		args, cleanup, parseErr = parseMultipartArgsWithNames(r, maxSize, opt.ParamNames, opt.ParamTypes)
		if cleanup != nil {
			defer cleanup()
		}
		if parseErr != nil {
			writeJSON(w, http.StatusBadRequest, FunctionCallResponse{
				Module: modulePath,
				Func:   funcName,
				Error:  fmt.Sprintf("failed to parse multipart: %v", parseErr),
			})
			return
		}
	} else {
		// Default: JSON body
		body, err := readRequestBody(r, 1<<20) // 1MB limit
		if err != nil {
			writeJSON(w, http.StatusBadRequest, FunctionCallResponse{
				Module: modulePath,
				Func:   funcName,
				Error:  "failed to read request body",
			})
			return
		}

		var parseErr error
		args, parseErr = parseArgsWithNames(body, opt.ParamNames, opt.ParamTypes)
		if parseErr != nil {
			writeJSON(w, http.StatusBadRequest, FunctionCallResponse{
				Module: modulePath,
				Func:   funcName,
				Error:  fmt.Sprintf("invalid arguments: %v", parseErr),
			})
			return
		}
	}

	// Fall back to query parameters when body args are empty (e.g., GET requests)
	if len(args) == 0 && len(r.URL.Query()) > 0 {
		args = parseQueryArgs(r.URL.Query())
	}

	// Flush Debug ghost effect output after each request
	defer s.flushDebugOutput()

	// Call AILANG function
	// Use Call (not CallPreserveFloats) so JSON-decoded whole numbers (float64)
	// are converted to IntValue when they fit. CallPreserveFloats kept them as
	// FloatValue, which broke int-typed record fields from cross-package calls.
	debugConc := os.Getenv("DEBUG_CONCURRENCY") == "1"
	if debugConc {
		log.Printf("[CONCURRENCY] calling engine.Call %s/%s (goroutine %d)", modulePath, funcName, goroutineID())
	}
	start := time.Now()
	result, callErr := s.engine.Call(modulePath, funcName, args...)
	elapsed := time.Since(start).Milliseconds()
	if debugConc {
		log.Printf("[CONCURRENCY] engine.Call returned %s/%s (goroutine %d, err=%v, %dms)", modulePath, funcName, goroutineID(), callErr, elapsed)
	}

	// Fix: zero-arg functions in AILANG internally compile to take a unit parameter.
	// If the call fails with "expects 1 arguments, got 0", retry with a unit arg.
	if callErr != nil && len(args) == 0 && strings.Contains(callErr.Error(), "expects 1 arguments, got 0") {
		start = time.Now()
		result, callErr = s.engine.Call(modulePath, funcName, nil)
		elapsed = time.Since(start).Milliseconds()
	}

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

	// Check if result is a raw response record (has _body field)
	if rec, ok := result.(*eval.RecordValue); ok {
		if _, hasBody := rec.Fields["_body"]; hasBody {
			writeRawResponse(w, rec, elapsed)
			return
		}
	}

	// Check if result is a Result.Err — map to non-200 HTTP status.
	// Must inspect the raw eval.Value BEFORE ToGo conversion so we can
	// extract _status from RecordValue fields with proper typing.
	if errStatus, errPayload, isErr := resultErrStatus(result); isErr {
		goErr, convErr := embed.ToGo(errPayload)
		if convErr != nil {
			goErr = fmt.Sprintf("result conversion failed: %v", convErr)
		}

		if opt.Nowrap {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Elapsed-Ms", fmt.Sprintf("%d", elapsed))
			w.WriteHeader(errStatus)
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			_ = enc.Encode(goErr)
			return
		}

		writeJSON(w, errStatus, FunctionCallResponse{
			Module:    modulePath,
			Func:      funcName,
			Error:     fmt.Sprintf("%v", goErr),
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

	// Unwrap Result.Ok — return the inner value, not the Ok wrapper.
	if tagged, ok := result.(*eval.TaggedValue); ok && tagged.CtorName == "Ok" {
		goResult, err = embed.ToGo(tagged.Fields[0])
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, FunctionCallResponse{
				Module:    modulePath,
				Func:      funcName,
				Error:     fmt.Sprintf("result conversion failed: %v", err),
				ElapsedMs: elapsed,
			})
			return
		}
	}

	// @nowrap: return raw JSON without FunctionCallResponse envelope
	if opt.Nowrap {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Elapsed-Ms", fmt.Sprintf("%d", elapsed))

		// Extract _headers from Go result map and set as HTTP headers
		if m, ok := goResult.(map[string]interface{}); ok {
			if headersVal, ok := m["_headers"]; ok {
				if headers, ok := headersVal.(map[string]interface{}); ok {
					for k, v := range headers {
						if sv, ok := v.(string); ok {
							w.Header().Set(k, sv)
						}
					}
				}
				delete(m, "_headers")
			}
		}

		w.WriteHeader(http.StatusOK)

		// Auto-unwrap: if result is a string containing a valid JSON object or array,
		// write it as raw bytes instead of double-encoding through json.Encoder.
		// This handles the common pattern: encode(jo([...])) -> '{"key":"val"}'
		if s, ok := goResult.(string); ok && isValidJSONObjectOrArray(s) {
			_, _ = w.Write([]byte(s))
			_, _ = w.Write([]byte("\n"))
			return
		}

		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(goResult)
		return
	}

	// Default: JSON-wrapped response
	writeJSON(w, http.StatusOK, FunctionCallResponse{
		Module:    modulePath,
		Func:      funcName,
		Result:    goResult,
		ElapsedMs: elapsed,
	})
}

// resultErrStatus checks if a value is a Result.Err variant and returns the
// HTTP status code and error payload. If the Err payload is a record containing
// a _status field, that value is used as the HTTP status code and stripped from
// the payload. Otherwise the default status is 400 (Bad Request).
func resultErrStatus(v eval.Value) (status int, payload eval.Value, isErr bool) {
	tagged, ok := v.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Err" {
		return 0, nil, false
	}

	// Err() with no fields — return 400 with empty error
	if len(tagged.Fields) == 0 {
		return http.StatusBadRequest, &eval.StringValue{Value: "error"}, true
	}

	payload = tagged.Fields[0]
	status = http.StatusBadRequest

	// If Err payload is a record with _status, extract and strip it
	if rec, ok := payload.(*eval.RecordValue); ok {
		if statusVal, ok := rec.Fields["_status"]; ok {
			if iv, ok := statusVal.(*eval.IntValue); ok {
				status = iv.Value
			}
			// Strip _status from payload — it's metadata, not response content
			stripped := &eval.RecordValue{Fields: make(map[string]eval.Value, len(rec.Fields)-1)}
			for k, v := range rec.Fields {
				if k != "_status" {
					stripped.Fields[k] = v
				}
			}
			payload = stripped
		}
	}

	return status, payload, true
}

// writeRawResponse writes a raw HTTP response from a record with _body, _status, _headers fields.
// This enables AILANG functions to return binary files with custom content types.
func writeRawResponse(w http.ResponseWriter, rec *eval.RecordValue, elapsedMs int64) {
	// Set custom headers from _headers field
	if headersVal, ok := rec.Fields["_headers"]; ok {
		if headersRec, ok := headersVal.(*eval.RecordValue); ok {
			for k, v := range headersRec.Fields {
				if sv, ok := v.(*eval.StringValue); ok {
					w.Header().Set(k, sv.Value)
				}
			}
		}
	}

	// Add timing header
	w.Header().Set("X-Elapsed-Ms", fmt.Sprintf("%d", elapsedMs))

	// Set status code from _status field
	status := http.StatusOK
	if statusVal, ok := rec.Fields["_status"]; ok {
		if iv, ok := statusVal.(*eval.IntValue); ok {
			status = iv.Value
		}
	}
	w.WriteHeader(status)

	// Write body from _body field
	body := rec.Fields["_body"]
	switch b := body.(type) {
	case *eval.BytesValue:
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		_, _ = w.Write(b.Value)
	case *eval.StringValue:
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}
		_, _ = w.Write([]byte(b.Value))
	default:
		// Fall back to JSON for other types
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		goVal, _ := embed.ToGo(body)
		_ = writeJSONBody(w, goVal)
	}
}

// writeJSONBody writes a JSON body to the response writer.
func writeJSONBody(w http.ResponseWriter, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

// readRequestBody reads the request body with a size limit.
func readRequestBody(r *http.Request, maxSize int64) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	return io.ReadAll(io.LimitReader(r.Body, maxSize))
}

// parseMultipartArgs extracts function arguments from a multipart/form-data request.
// File fields become *eval.BytesValue, non-file fields become strings.
func parseMultipartArgs(r *http.Request, maxSize int64) ([]interface{}, error) {
	if r.MultipartForm == nil {
		return nil, nil
	}

	var args []interface{}

	// File fields become BytesValue
	for _, fileHeaders := range r.MultipartForm.File {
		for _, fh := range fileHeaders {
			f, err := fh.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open uploaded file %q: %w", fh.Filename, err)
			}
			data, err := io.ReadAll(io.LimitReader(f, maxSize))
			f.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read uploaded file %q: %w", fh.Filename, err)
			}
			args = append(args, &eval.BytesValue{
				Value:    data,
				Filename: fh.Filename,
				MimeType: fh.Header.Get("Content-Type"),
			})
		}
	}

	// Non-file form fields become string args
	for _, values := range r.MultipartForm.Value {
		for _, v := range values {
			args = append(args, v)
		}
	}

	return args, nil
}

// writeTempFile writes data to a temp file preserving the original filename.
// Creates a unique temp directory and writes the file with its original name,
// so filepath.Base(path) returns the real filename (e.g. "report.docx").
// Caller is responsible for cleanup (remove the parent directory).
func writeTempFile(data []byte, originalFilename string) (string, error) {
	dir, err := os.MkdirTemp("", "ailang-upload-*")
	if err != nil {
		return "", err
	}
	name := originalFilename
	if name == "" {
		name = "upload"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return path, nil
}

// parseMultipartArgsWithNames maps multipart fields to function parameters by name.
// File fields become *eval.BytesValue or temp file paths (if target param is string).
// Non-file fields become strings. Unmatched params get zero-values.
// Returns args, a cleanup function for temp files, and any error.
// Falls back to positional parseMultipartArgs when no paramNames are provided.
func parseMultipartArgsWithNames(r *http.Request, maxSize int64, paramNames []string, paramTypes []string) ([]interface{}, func(), error) {
	if r.MultipartForm == nil || len(paramNames) == 0 {
		args, err := parseMultipartArgs(r, maxSize)
		return args, func() {}, err
	}

	args := make([]interface{}, len(paramNames))
	var tempFiles []string
	matchedFiles := map[string]bool{} // track which multipart file fields were consumed

	// Pass 1: exact name matching (field name == param name)
	for i, name := range paramNames {
		paramType := ""
		if i < len(paramTypes) {
			paramType = paramTypes[i]
		}

		// Check file fields first
		if fileHeaders, ok := r.MultipartForm.File[name]; ok && len(fileHeaders) > 0 {
			fh := fileHeaders[0]
			data, err := readMultipartFile(fh, maxSize)
			if err != nil {
				removeTempFiles(tempFiles)
				return nil, nil, err
			}
			matchedFiles[name] = true

			if paramType == "string" {
				tmpPath, err := writeTempFile(data, fh.Filename)
				if err != nil {
					removeTempFiles(tempFiles)
					return nil, nil, fmt.Errorf("failed to write temp file: %w", err)
				}
				tempFiles = append(tempFiles, tmpPath)
				args[i] = tmpPath
			} else {
				args[i] = &eval.BytesValue{
					Value:    data,
					Filename: fh.Filename,
					MimeType: fh.Header.Get("Content-Type"),
				}
			}
			continue
		}

		// Check non-file form fields
		if values, ok := r.MultipartForm.Value[name]; ok && len(values) > 0 {
			args[i] = values[0]
			continue
		}

		// Not matched yet — leave nil for now (filled in pass 2 or zero-padded)
	}

	// Pass 2: assign unmatched file uploads to unmatched file-accepting params.
	// This handles curl -F 'file=@doc.docx' when the param is named 'filepath'.
	var unmatchedFiles []*multipart.FileHeader
	for fieldName, fileHeaders := range r.MultipartForm.File {
		if !matchedFiles[fieldName] && len(fileHeaders) > 0 {
			unmatchedFiles = append(unmatchedFiles, fileHeaders[0])
		}
	}

	if len(unmatchedFiles) > 0 {
		fileIdx := 0
		for i := range paramNames {
			if args[i] != nil || fileIdx >= len(unmatchedFiles) {
				continue
			}
			paramType := ""
			if i < len(paramTypes) {
				paramType = paramTypes[i]
			}
			// Only assign to string or bytes params that weren't matched in pass 1
			if paramType == "string" || paramType == "bytes" || paramType == "" {
				fh := unmatchedFiles[fileIdx]
				data, err := readMultipartFile(fh, maxSize)
				if err != nil {
					removeTempFiles(tempFiles)
					return nil, nil, err
				}
				fileIdx++

				if paramType == "string" {
					tmpPath, err := writeTempFile(data, fh.Filename)
					if err != nil {
						removeTempFiles(tempFiles)
						return nil, nil, fmt.Errorf("failed to write temp file: %w", err)
					}
					tempFiles = append(tempFiles, tmpPath)
					args[i] = tmpPath
				} else {
					args[i] = &eval.BytesValue{
						Value:    data,
						Filename: fh.Filename,
						MimeType: fh.Header.Get("Content-Type"),
					}
				}
			}
		}
	}

	// Pass 3: zero-value pad any remaining unmatched params
	for i := range args {
		if args[i] == nil {
			paramType := ""
			if i < len(paramTypes) {
				paramType = paramTypes[i]
			}
			args[i] = zeroValueForType(paramType)
		}
	}

	cleanup := func() {
		removeTempFiles(tempFiles)
	}
	return args, cleanup, nil
}

// readMultipartFile opens and reads a multipart file header, respecting the size limit.
func readMultipartFile(fh *multipart.FileHeader, maxSize int64) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file %q: %w", fh.Filename, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read uploaded file %q: %w", fh.Filename, err)
	}
	return data, nil
}

// removeTempFiles removes temporary upload files and their parent directories.
func removeTempFiles(paths []string) {
	for _, p := range paths {
		os.RemoveAll(filepath.Dir(p))
	}
}

// buildHttpRequestRecord constructs a map representing an HttpRequest record
// from an http.Request and its already-read body. Used by @raw routes.
// Headers and query are JObject (Json ADT) so handlers can use std/json.get()
// for dynamic key access (e.g., hyphenated header names like "Stripe-Signature").
func buildHttpRequestRecord(r *http.Request, body []byte) map[string]interface{} {
	return map[string]interface{}{
		"body":    string(body),
		"headers": stringMapToJObject(r.Header),
		"method":  r.Method,
		"path":    r.URL.Path,
		"query":   stringMapToJObject(r.URL.Query()),
	}
}

// stringMapToJObject converts an http.Header or url.Values (map[string][]string)
// to a JObject TaggedValue: JObject(List[{key: string, value: JString(string)}]).
func stringMapToJObject(m map[string][]string) *eval.TaggedValue {
	kvPairs := make([]eval.Value, 0, len(m))
	for k, v := range m {
		if len(v) > 0 {
			kvPairs = append(kvPairs, &eval.RecordValue{
				Fields: map[string]eval.Value{
					"key": &eval.StringValue{Value: k},
					"value": &eval.TaggedValue{
						ModulePath: "std/json", TypeName: "Json", CtorName: "JString",
						Fields: []eval.Value{&eval.StringValue{Value: v[0]}},
					},
				},
			})
		}
	}
	return &eval.TaggedValue{
		ModulePath: "std/json", TypeName: "Json", CtorName: "JObject",
		Fields: []eval.Value{&eval.ListValue{Elements: kvPairs}},
	}
}

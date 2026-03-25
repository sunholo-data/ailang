package apiserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
	Method   string // "GET", "POST", etc.
	Path     string // "/general/v0/general"
	Module   string // module path
	Function string // function name
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

		// Find matching export and set route info
		for i := range modInfo.Exports {
			if modInfo.Exports[i].Name == fn.Name {
				modInfo.Exports[i].RouteMethod = method
				modInfo.Exports[i].RoutePath = path
				log.Printf("    Route: %s %s -> %s", method, path, fn.Name)
				break
			}
		}
	}
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
					Method:   exp.RouteMethod,
					Path:     exp.RoutePath,
					Module:   mod.Path,
					Function: exp.Name,
				})
			}
		}
	}
	return routes
}

// registerCustomRoutes adds custom route handlers to the mux.
// Custom routes are registered BEFORE the catch-all /api/ handler.
func (s *Server) registerCustomRoutes(mux *http.ServeMux) {
	routes := s.getCustomRoutes()
	for _, route := range routes {
		r := route // capture for closure
		handler := func(w http.ResponseWriter, req *http.Request) {
			// Enforce HTTP method
			if req.Method != r.Method && req.Method != "OPTIONS" {
				writeJSON(w, http.StatusMethodNotAllowed, FunctionCallResponse{
					Error: fmt.Sprintf("this endpoint only accepts %s requests", r.Method),
				})
				return
			}
			s.callFunction(w, req, r.Module, r.Function)
		}
		mux.HandleFunc(r.Path, s.corsWrap(s.authMiddleware(handler)))
		log.Printf("  Custom route: %s %s -> %s/%s", r.Method, r.Path, r.Module, r.Function)
	}
}

// callFunction executes an AILANG function and writes the response.
// Shared by both the catch-all handler and custom route handlers.
func (s *Server) callFunction(w http.ResponseWriter, r *http.Request, modulePath, funcName string) {
	if os.Getenv("DEBUG_CONCURRENCY") == "1" {
		log.Printf("[CONCURRENCY] callFunction entered: %s/%s (goroutine %d)", modulePath, funcName, goroutineID())
	}
	// Parse arguments based on content type
	var args []interface{}
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
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
		var parseErr error
		args, parseErr = parseMultipartArgs(r, maxSize)
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
		args, parseErr = parseArgs(body)
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
	debugConc := os.Getenv("DEBUG_CONCURRENCY") == "1"
	if debugConc {
		log.Printf("[CONCURRENCY] calling engine.CallPreserveFloats %s/%s (goroutine %d)", modulePath, funcName, goroutineID())
	}
	start := time.Now()
	result, callErr := s.engine.CallPreserveFloats(modulePath, funcName, args...)
	elapsed := time.Since(start).Milliseconds()
	if debugConc {
		log.Printf("[CONCURRENCY] engine.CallPreserveFloats returned %s/%s (goroutine %d, err=%v, %dms)", modulePath, funcName, goroutineID(), callErr, elapsed)
	}

	// Fix: zero-arg functions in AILANG internally compile to take a unit parameter.
	// If the call fails with "expects 1 arguments, got 0", retry with a unit arg.
	if callErr != nil && len(args) == 0 && strings.Contains(callErr.Error(), "expects 1 arguments, got 0") {
		start = time.Now()
		result, callErr = s.engine.CallPreserveFloats(modulePath, funcName, nil)
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

	// Default: JSON-wrapped response
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

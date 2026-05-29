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
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/embed"
	"github.com/sunholo-data/ailang/internal/eval"
)

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

		// resolveArgs owns the precedence: body → query (when body source
		// is not Real) → zero-value padding. The provenance-based fallback
		// guard prevents zero-padding from silently shadowing query args
		// for GET handlers with declared params.
		// See: design_docs/planned/v0_21_0/m-serveapi-get-query-shadow.md
		var parseErr error
		args, _, parseErr = resolveArgs(r, body, opt.ParamNames, opt.ParamTypes)
		if parseErr != nil {
			writeJSON(w, http.StatusBadRequest, FunctionCallResponse{
				Module: modulePath,
				Func:   funcName,
				Error:  fmt.Sprintf("invalid arguments: %v", parseErr),
			})
			return
		}
	}

	// Inject request headers for @route handlers declaring a _headers parameter.
	// This allows @route functions to access HTTP headers (e.g., auth tokens)
	// without switching to @raw (which loses multipart parsing).
	if !opt.Raw && len(opt.ParamNames) > 0 {
		for i, name := range opt.ParamNames {
			if name == "_headers" && i < len(args) {
				args[i] = stringMapToJObject(r.Header)
			}
		}
	}

	// Multipart and raw paths don't route through resolveArgs; preserve the
	// historical "empty args + query present" fallback for them so that
	// query data still reaches handlers that bypassed JSON-body parsing.
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
	// filepath.Base strips any directory components the client may have
	// embedded in the multipart filename (e.g. "../../etc/passwd" or
	// "sub/dir/report.docx"). Without this, filepath.Join would happily
	// resolve the traversal and write outside `dir`. Base still returns
	// ".." for a literal "..", so reject that explicitly too.
	name := filepath.Base(originalFilename)
	if name == "" || name == "." || name == ".." || name == string(filepath.Separator) {
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
	// A warning is logged so server operators can see the fallback was used.
	type unmatchedFile struct {
		fieldName string
		header    *multipart.FileHeader
	}
	var unmatchedFiles []unmatchedFile
	for fieldName, fileHeaders := range r.MultipartForm.File {
		if !matchedFiles[fieldName] && len(fileHeaders) > 0 {
			unmatchedFiles = append(unmatchedFiles, unmatchedFile{fieldName, fileHeaders[0]})
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
				uf := unmatchedFiles[fileIdx]
				log.Printf("  WARNING: multipart field %q does not match param %q — assigning by position (use -F '%s=@file' for exact match)", uf.fieldName, paramNames[i], paramNames[i])
				data, err := readMultipartFile(uf.header, maxSize)
				if err != nil {
					removeTempFiles(tempFiles)
					return nil, nil, err
				}
				fileIdx++

				if paramType == "string" {
					tmpPath, err := writeTempFile(data, uf.header.Filename)
					if err != nil {
						removeTempFiles(tempFiles)
						return nil, nil, fmt.Errorf("failed to write temp file: %w", err)
					}
					tempFiles = append(tempFiles, tmpPath)
					args[i] = tmpPath
				} else {
					args[i] = &eval.BytesValue{
						Value:    data,
						Filename: uf.header.Filename,
						MimeType: uf.header.Header.Get("Content-Type"),
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

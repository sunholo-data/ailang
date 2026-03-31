# M-MULTIPART-BYTES: Multipart Upload BytesValue Handling in serve-api

**Status**: Implemented
**Target**: v0.10.1
**Priority**: P1 — Blocks all multipart file upload parsing on live endpoints
**Estimated**: 0.5 days (~4 hours)
**Dependencies**: None (BytesValue type and bytes builtins already exist)
**Milestone ID**: M-MULTIPART-BYTES
**Created**: 2026-03-31
**Source**: DocParse agent message `87af52ec` (serve-api multipart _str_len BytesValue error)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Runtime HTTP layer change, outside deterministic core |
| A2: Replayability | 0 | No change to trace/replay semantics |
| A3: Effect Legibility | +1 | Makes the bytes→string boundary explicit instead of crashing silently |
| A4: Explicit Authority | 0 | No new capabilities — uses existing IO effect for temp file writes |
| A5: Bounded Verification | +1 | Named param matching for multipart enables compile-time-like validation |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Prevents agent-facing crashes on file upload; enables autonomous docparse pipeline |
| A8: Minimal Syntax | 0 | No new language syntax |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | +1 | Multipart args compose with existing named parameter binding contract |
| A11: Structured Failure | +1 | Returns typed error instead of runtime type assertion crash |
| A12: System Boundary | +1 | Fixes the HTTP multipart → AILANG type system boundary |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — temp file paths are deterministic per request
- [x] A3 (Effects): No hidden side effects — temp file creation is within existing IO effect boundary
- [x] A4 (Authority): No ambient access granted — uses existing HTTP/IO capabilities
- [x] A7 (Machines First): Directly unblocks autonomous agent file processing

---

## Problem Statement

Multipart file uploads to serve-api endpoints crash at runtime with:

```
_str_len: expected String, got *eval.BytesValue
```

### Root Cause

When a file is uploaded via `multipart/form-data` (e.g., `curl -F 'file=@sample.docx'`), the serve-api `parseMultipartArgs()` function (`internal/apiserver/routes.go:502-539`) correctly creates `*eval.BytesValue` for file fields. However:

1. **No named parameter matching** — Unlike JSON parsing (`parseArgsWithNames`), multipart parsing ignores `paramNames`/`paramTypes` entirely. Args are collected in arbitrary map iteration order.

2. **No type coercion at the boundary** — When a function declares `func parseFileSecure(filepath: string, outputFormat: string, apiKey: string)`, the multipart `file` field arrives as `BytesValue` but is passed directly as the first positional arg — which the function expects to be a string filepath.

3. **@raw and multipart are mutually exclusive** — The if/else chain at `routes.go:276-311` means `@raw` handlers never see parsed multipart data, and multipart handlers don't get the `@raw` HttpRequest record. There's no way to declare "I want multipart data parsed AND access to request metadata."

### Reproduction

```bash
curl -X POST https://ailang-dev-docparse-api-ejjw6zt3bq-ew.a.run.app/api/v1/parse \
  -F 'file=@any.docx' \
  -F 'output_format=markdown' \
  -F 'api_key=dp_...'
# Returns 500: _str_len: expected String, got *eval.BytesValue
```

### Current State

- `parseMultipartArgs()` iterates `r.MultipartForm.File` and `r.MultipartForm.Value` maps
- File fields → `*eval.BytesValue{Value, Filename, MimeType}`
- Non-file fields → raw Go strings
- Args are appended in map iteration order (non-deterministic!)
- No mapping of field names to function parameter names
- Existing bytes builtins (`_bytes_to_string`, `_bytes_length`, `_bytes_filename`, `_bytes_mime_type`) exist in `internal/builtins/bytes.go` but functions receiving bytes must be written to expect `bytes` type

### Impact

- **All multipart file upload endpoints are broken** on live Cloud Run
- DocParse production pipeline cannot process uploaded documents
- No workaround available (JSON base64 encoding works but defeats multipart purpose)

---

## Goals

**Primary Goal:** Make multipart file uploads work with AILANG functions that declare string parameters, while preserving BytesValue for functions that explicitly accept bytes.

**Success Metrics:**
- `func parse(filepath: string, format: string, key: string)` with multipart `curl -F 'file=@doc.pdf' -F 'format=markdown' -F 'key=abc'` correctly maps field names to parameter names
- File field mapped to a `string` param → temp file written, path passed as string
- File field mapped to a `bytes` param → BytesValue passed directly
- Non-file fields mapped by name (not position)
- Args delivered in parameter order, not map iteration order
- Unmatched params receive zero-values (consistent with JSON path)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Temp file for string params vs error | Determines UX: auto-convert or require bytes type | agent | design | med |
| Named matching vs positional | Determines if field order matters | agent | design | low |
| Temp file cleanup responsibility | Who deletes temp files — runtime or user code | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Named parameter matching for multipart (consistent with JSON named binding)
- [x] File field + string param → write to temp file, pass path (most useful for existing functions)
- [x] File field + bytes param → pass BytesValue directly (zero conversion overhead)
- [x] Temp files cleaned up after function returns (runtime responsibility)
- [x] Unmatched multipart params get zero-values (consistent with v0.9.5 contract)

---

## Solution Design

### Overview

Three focused changes:

1. **Named multipart parsing** — new `parseMultipartArgsWithNames()` that maps field names to function params
2. **Bytes-to-tempfile coercion** — when a BytesValue is bound to a string-typed param, write to temp file and pass path
3. **Callsite integration** — wire the new parser into the multipart branch of `callFunction()`

### Architecture

**Component 1: Named multipart arg parsing** (`internal/apiserver/routes.go`)

Replace positional `parseMultipartArgs()` with a named variant:

```go
// parseMultipartArgsWithNames maps multipart fields to function parameters by name.
// File fields become *eval.BytesValue or temp file paths (if target param is string).
// Non-file fields become strings.
// Unmatched params get zero-values.
func parseMultipartArgsWithNames(r *http.Request, maxSize int64, paramNames []string, paramTypes []string) ([]interface{}, func(), error) {
    if r.MultipartForm == nil || len(paramNames) == 0 {
        // Fall back to positional parsing (backward compat for no-param functions)
        args, err := parseMultipartArgs(r, maxSize)
        return args, func() {}, err
    }

    args := make([]interface{}, len(paramNames))
    var tempFiles []string // track for cleanup

    for i, name := range paramNames {
        paramType := ""
        if i < len(paramTypes) {
            paramType = paramTypes[i]
        }

        // Check file fields first
        if fileHeaders, ok := r.MultipartForm.File[name]; ok && len(fileHeaders) > 0 {
            fh := fileHeaders[0]
            f, err := fh.Open()
            if err != nil {
                return nil, nil, fmt.Errorf("failed to open uploaded file %q: %w", fh.Filename, err)
            }
            data, err := io.ReadAll(io.LimitReader(f, maxSize))
            f.Close()
            if err != nil {
                return nil, nil, fmt.Errorf("failed to read uploaded file %q: %w", fh.Filename, err)
            }

            if paramType == "string" {
                // Write to temp file, pass path
                tmpPath, err := writeTempFile(data, fh.Filename)
                if err != nil {
                    return nil, nil, fmt.Errorf("failed to write temp file: %w", err)
                }
                tempFiles = append(tempFiles, tmpPath)
                args[i] = tmpPath
            } else {
                // Pass as BytesValue (bytes type or untyped)
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

        // No match — zero-value pad
        args[i] = zeroValueForType(paramType)
    }

    cleanup := func() {
        for _, path := range tempFiles {
            os.Remove(path)
        }
    }

    return args, cleanup, nil
}
```

**Component 2: Temp file helper** (`internal/apiserver/routes.go`)

```go
// writeTempFile writes data to a temp file preserving the original extension.
// Returns the temp file path. Caller is responsible for cleanup.
func writeTempFile(data []byte, originalFilename string) (string, error) {
    ext := filepath.Ext(originalFilename)
    pattern := "ailang-upload-*" + ext
    f, err := os.CreateTemp("", pattern)
    if err != nil {
        return "", err
    }
    defer f.Close()
    if _, err := f.Write(data); err != nil {
        os.Remove(f.Name())
        return "", err
    }
    return f.Name(), nil
}
```

**Component 3: Callsite integration** (`internal/apiserver/routes.go`)

Update the multipart branch in `callFunction()`:

```go
} else if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
    maxSize := s.maxUploadSize
    if maxSize == 0 {
        maxSize = 50 << 20
    }
    if err := r.ParseMultipartForm(maxSize); err != nil {
        // ... existing error handling
        return
    }
    var cleanup func()
    var parseErr error
    args, cleanup, parseErr = parseMultipartArgsWithNames(r, maxSize, opt.ParamNames, opt.ParamTypes)
    if cleanup != nil {
        defer cleanup() // remove temp files after function returns
    }
    if parseErr != nil {
        // ... existing error handling
        return
    }
}
```

### Implementation Plan

**Phase 1: Named multipart parsing** (~2 hours)
- [ ] Add `parseMultipartArgsWithNames()` with field→param name matching
- [ ] Add `writeTempFile()` helper with extension preservation
- [ ] Add `zeroValueForType` padding for unmatched multipart params
- [ ] Keep existing `parseMultipartArgs()` as fallback for no-param functions

**Phase 2: Callsite integration** (~1 hour)
- [ ] Update multipart branch in `callFunction()` to use new parser
- [ ] Pass `opt.ParamNames` and `opt.ParamTypes` to multipart parsing
- [ ] Wire cleanup function with `defer` for temp file removal

**Phase 3: Tests** (~1 hour)
- [ ] Unit test: file field + string param → temp file path returned
- [ ] Unit test: file field + bytes param → BytesValue returned
- [ ] Unit test: non-file field + string param → string value returned
- [ ] Unit test: named matching — fields map to params by name, not position
- [ ] Unit test: unmatched param → zero-value padding
- [ ] Unit test: no paramNames → falls back to positional `parseMultipartArgs`
- [ ] Regression test: existing multipart without named params still works
- [ ] `make test`, `make lint`, `make verify-examples`

### Files to Modify

**Modified files:**
- `internal/apiserver/routes.go` — Add `parseMultipartArgsWithNames()`, `writeTempFile()`, update multipart branch (~60 LOC)
- `internal/apiserver/routes_test.go` or `internal/apiserver/multipart_test.go` — Named multipart tests (~80 LOC)

**No new files** (test file may be new if `multipart_test.go` doesn't exist yet).

---

## Examples

### Example 1: File upload to string-typed param (the bug fix)

**Before (crashes):**
```bash
# Endpoint: func parseFileSecure(filepath: string, outputFormat: string, apiKey: string)
curl -X POST http://localhost:8080/parseFileSecure \
  -F 'filepath=@document.docx' \
  -F 'outputFormat=markdown' \
  -F 'apiKey=dp_xxx'
# 500: _str_len: expected String, got *eval.BytesValue
```

**After (works):**
```bash
curl -X POST http://localhost:8080/parseFileSecure \
  -F 'filepath=@document.docx' \
  -F 'outputFormat=markdown' \
  -F 'apiKey=dp_xxx'
# filepath receives "/tmp/ailang-upload-123456.docx" (temp file)
# outputFormat receives "markdown"
# apiKey receives "dp_xxx"
# Temp file cleaned up after function returns
```

### Example 2: File upload to bytes-typed param (zero-copy)

```bash
# Endpoint: func processFile(data: bytes, format: string) -> string
curl -X POST http://localhost:8080/processFile \
  -F 'data=@image.png' \
  -F 'format=png'
# data receives BytesValue{Value: [...], Filename: "image.png", MimeType: "image/png"}
# format receives "png"
# Can use _bytes_length(data), _bytes_to_string(data), _bytes_filename(data), etc.
```

### Example 3: Unmatched fields get zero-values

```bash
# Endpoint: func process(file: bytes, apiKey: string)
curl -X POST http://localhost:8080/process \
  -F 'file=@doc.pdf'
# file receives BytesValue
# apiKey receives "" (zero-value, no crash)
```

### Example 4: Backward compat — no paramNames (positional)

```bash
# Endpoint with no param type info (legacy)
curl -X POST http://localhost:8080/legacyUpload \
  -F 'file=@doc.pdf'
# Falls back to existing parseMultipartArgs() — positional BytesValue
```

---

## Success Criteria

- [ ] Multipart file field + string param → temp file path (not BytesValue)
- [ ] Multipart file field + bytes param → BytesValue (no conversion)
- [ ] Multipart non-file fields matched by name to params
- [ ] Args delivered in param declaration order, not map iteration order
- [ ] Unmatched params get zero-values (consistent with JSON path)
- [ ] Temp files cleaned up after function returns
- [ ] Existing positional multipart parsing preserved for paramless functions
- [ ] All tests passing (`make test`)
- [ ] Lint passing (`make lint`)
- [ ] Examples verified (`make verify-examples`)

---

## Testing Strategy

**Unit tests (named multipart parsing):**
- `parseMultipartArgsWithNames` with file field matching string param → temp file path
- `parseMultipartArgsWithNames` with file field matching bytes param → BytesValue
- `parseMultipartArgsWithNames` with non-file field matching string param → string
- `parseMultipartArgsWithNames` with unmatched param → zero-value
- `parseMultipartArgsWithNames` with no paramNames → falls back to positional
- `writeTempFile` preserves file extension
- Cleanup function removes all temp files

**Integration test:**
- Full HTTP round-trip: multipart POST → AILANG function → response (no crash)

**Manual testing:**
- `curl -F 'file=@test.docx' -F 'format=md'` against local `ailang serve`
- Verify temp file is created and cleaned up (check `/tmp/ailang-upload-*` before/after)

---

## Deferred Decisions

- Whether `@raw` endpoints should also get parsed multipart access (separate design)
- Whether to support multi-file arrays (`file[]=@a.pdf&file[]=@b.pdf`) — future work
- Whether temp file directory should be configurable — agent may choose `/tmp` or `os.TempDir()`

---

## Non-Goals

**Not attempted in this feature:**
- `@raw` + multipart interaction — `@raw` always gets raw body; this is by design
- Streaming upload support — out of scope
- File size validation per field — existing `maxUploadSize` applies globally
- MIME type validation — pass through as metadata, let AILANG code validate
- Multi-file per param — only first file header used per field name

---

## Timeline

**Day 1** (~4 hours):
- Phase 1: Named multipart parsing (2h)
- Phase 2: Callsite integration (1h)
- Phase 3: Tests (1h)

**Total: ~4 hours, single day**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Temp file leak on panic | Med | `defer cleanup()` in callsite; also consider os.TempDir cleanup on server shutdown |
| Disk space exhaustion from large uploads | Med | Existing `maxUploadSize` limit applies; temp files are short-lived (cleaned after function returns) |
| Map iteration order changes between Go versions | Low | Named matching eliminates dependence on map order entirely |
| Existing multipart endpoints break | Med | Falls back to positional parsing when no paramNames — zero behavioral change for legacy endpoints |

---

## Related Documents

**Implemented (directly relevant):**
- [design_docs/implemented/v0_9_5/m-serve-api-zero-value-padding.md](design_docs/implemented/v0_9_5/m-serve-api-zero-value-padding.md) — Zero-value padding contract this extends to multipart
- [design_docs/implemented/v0_10_0/m-serve-api-agent-enhancements.md](design_docs/implemented/v0_10_0/m-serve-api-agent-enhancements.md) — Named parameter binding for JSON

**Planned (same release):**
- [design_docs/planned/v0_10_1/m-file-handling-improvements.md](design_docs/planned/v0_10_1/m-file-handling-improvements.md) — fileData/fileUri and POST param fix (complementary, no overlap)

**Planned (check for overlap):**
- [design_docs/planned/v0_11_0/m-serve-api-dx.md](design_docs/planned/v0_11_0/m-serve-api-dx.md) — Broader serve-api DX improvements

---

## References

- DocParse agent message `87af52ec` — Original bug report
- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- `internal/builtins/bytes.go` — Existing bytes builtins (_bytes_to_string, _bytes_length, etc.)
- `internal/eval/value.go` — BytesValue and StringValue type definitions

---

## Future Work

- **@raw multipart access** — Allow @raw handlers to access parsed multipart fields via HttpRequest record
- **Multi-file support** — Accept arrays of files per field name
- **Temp file directory config** — Server-level `--temp-dir` flag for upload staging
- **Streaming multipart** — Process large files without reading fully into memory

---

**Document created**: 2026-03-31
**Last updated**: 2026-03-31

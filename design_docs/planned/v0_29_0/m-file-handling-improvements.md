# M-FILE-HANDLING: fileData/fileUri Support & serve-api POST Param Fix

**Status**: Planned
**Target**: v0.10.1
**Priority**: P1 — Blocks large-file multimodal pipelines; fixes serve-api crash path
**Estimated**: 0.5 days (~4 hours)
**Dependencies**: None (Gemini handler and serve-api both exist and are stable)
**Milestone ID**: M-FILE-HANDLING
**Created**: 2026-03-31
**Source**: DocParse agent messages `f78cf3a9` (fileData) and `d8506494` (POST param)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Both changes are in runtime HTTP layers, outside the deterministic core |
| A2: Replayability | 0 | No change to trace/replay semantics |
| A3: Effect Legibility | +1 | fileUri makes the "where data lives" distinction explicit in the API call |
| A4: Explicit Authority | 0 | No new capabilities needed — AI effect already requires `--caps AI` |
| A5: Bounded Verification | +1 | serve-api fix enables compile-time-like param validation at HTTP boundary |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | fileUri reduces token waste (220MB→<1KB for large PDFs); zero-value fix prevents agent crashes |
| A8: Minimal Syntax | 0 | No new language syntax — changes are in runtime handlers only |
| A9: Cost Visibility | +1 | fileUri makes the cost difference visible: base64 inline vs URI reference |
| A10: Composability | +1 | fileData composes with existing multimodal JSON contract; zero-value fix composes with named binding |
| A11: Structured Failure | +1 | serve-api returns typed zero-values instead of crashing on unit |
| A12: System Boundary | +1 | Both changes improve system boundary handling (GCS↔Gemini API, HTTP↔AILANG) |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — `fileUri` produces same API request every time
- [x] A3 (Effects): No hidden side effects — both changes are in existing effect boundaries
- [x] A4 (Authority): No ambient access granted — uses existing AI and HTTP capabilities
- [x] A7 (Machines First): Directly reduces AI agent costs and prevents agent-facing crashes

---

## Problem Statement

Two related issues at the file-handling boundary of AILANG's runtime:

### Problem 1: Gemini multimodal handler only supports inlineData (base64)

`buildParts()` in `internal/ai/gemini/generate.go` only constructs `inlineData` parts. For large files (PDFs 5+ MB), this means:

- A 20-page PDF (~10MB) becomes ~14MB base64, sent with **every** AI call
- Multi-turn conversations re-send the same 14MB payload each turn
- A typical docparse pipeline: 220MB+ of redundant traffic, 12+ minutes wall time

The Gemini API natively supports `fileData` parts that reference files by URI (`gs://` for Vertex AI, Files API URIs for AI Studio). The external `sunholo/gemini_files` package already handles uploads and returns URIs — the missing piece is runtime support in AILANG's Gemini handler.

### Problem 2: serve-api passes raw Record to single-param POST functions

When a POST endpoint declares `func foo(apiKey: string)` and receives `{}` (empty body or body with no matching keys):

- `parseNamedArgs()` returns `nil` (matched == 0) at `handler.go:204-205`
- Falls through to `parseArgs()` which wraps the entire body as a single argument
- Single-param function receives a `Record` value instead of a string
- Causes runtime crashes in `sha256Hex`, `length()`, etc.

This is a gap in the zero-value padding implemented in v0.9.5 (M-SERVE-API-ZERO-VALUE-PADDING). That design correctly pads *matched* params but returns `nil` when *no* params match, triggering a raw-body fallback that was designed for intentional Record-typed endpoints.

**Current State:**
- `buildParts()` only checks for `"data"` field → always produces `inlineData`
- `parseArgsWithNames()` falls back to `parseArgs()` when named binding returns nil, regardless of whether params exist

**Impact:**
- Large-file multimodal pipelines are 100-200x slower than necessary
- Single-param POST endpoints crash instead of receiving zero-values
- Both issues affect the docparse production pipeline

## Goals

**Primary Goal:** Support file URI references in Gemini multimodal calls and fix serve-api parameter validation for single-param POST endpoints.

**Success Metrics:**
- `{"mode": "multimodal", "fileUri": "gs://bucket/file.pdf", "mimeType": "application/pdf"}` produces a `fileData` part
- `{"mode": "multimodal", "data": "base64...", "mimeType": "image/png"}` continues producing `inlineData` (backward compat)
- When both `fileUri` and `data` are present, `fileUri` takes precedence
- POST to `func foo(apiKey: string)` with `{}` receives `""` (empty string), not a Record
- POST to `func bar(config: record)` with `{"key": "val"}` still receives the Record (backward compat)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| fileUri precedence over data | Determines behavior when both fields present; affects all callers | agent | design | low |
| Zero-value vs error for unmatched params | Determines whether missing params error or default; must match v0.9.5 behavior | agent | design | low |
| URI validation (gs:// only vs any) | Whether to validate URI scheme or pass through | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] fileUri takes precedence when both `fileUri` and `data` are present (matches Gemini API behavior)
- [x] Unmatched params get zero-value padding (consistent with v0.9.5 design)
- [x] No URI scheme validation — pass through to Gemini API (supports both gs:// and Files API URIs)

## Solution Design

### Overview

Two focused changes in existing files, no new files needed:

1. **Gemini handler**: Add `fileData` struct to `types.go`, extend `buildParts()` to check for `fileUri` field
2. **serve-api handler**: When `parseNamedArgs` returns nil but `paramNames` exists, return zero-value-padded args instead of falling through to raw single-arg parsing

### Architecture

**Component 1: fileData part type** (`internal/ai/gemini/types.go`)

Add a new struct and field to the existing `part`:

```go
// fileData represents a reference to a file stored externally (GCS, Files API).
type fileData struct {
    MimeType string `json:"mimeType"` // e.g., "application/pdf"
    FileUri  string `json:"fileUri"`  // gs://bucket/file.pdf or Files API URI
}

// Updated part struct:
type part struct {
    Text       string      `json:"text,omitempty"`
    InlineData *inlineData `json:"inlineData,omitempty"`
    FileData   *fileData   `json:"fileData,omitempty"` // NEW
}
```

**Component 2: buildParts fileUri branch** (`internal/ai/gemini/generate.go`)

Extend the multimodal JSON detection in `buildParts()`:

```go
// Inside the multimodal detection block:
if obj["mode"] == "multimodal" && obj["mimeType"] != "" {
    // fileUri takes precedence over data
    if obj["fileUri"] != "" {
        parts := []part{
            {FileData: &fileData{
                MimeType: obj["mimeType"],
                FileUri:  obj["fileUri"],
            }},
        }
        // ... add text prompt as before
        return parts
    }
    if obj["data"] != "" {
        // ... existing inlineData logic (unchanged)
    }
}
```

**Component 3: parseArgsWithNames fallback fix** (`internal/apiserver/handler.go`)

Change the fallback at line 246-247:

```go
// Current (buggy):
// Fall back to single-arg parsing
return parseArgs(body)

// Fixed:
// If named binding failed but function has declared params,
// return zero-value-padded args (not the raw body as a single arg).
if len(paramNames) > 0 && len(paramTypes) > 0 {
    args := make([]interface{}, len(paramNames))
    for i, t := range paramTypes {
        if i < len(args) {
            args[i] = zeroValueForType(t)
        }
    }
    return args, nil
}
return parseArgs(body)
```

### Implementation Plan

**Phase 1: fileData support** (~2 hours)
- [ ] Add `fileData` struct to `types.go`
- [ ] Add `FileData *fileData` field to `part` struct
- [ ] Extend `buildParts()` to check `fileUri` field with precedence over `data`
- [ ] Add unit tests for: fileUri-only, data-only, both-present, neither-present
- [ ] Add test for gs:// URI and Files API URI passthrough

**Phase 2: serve-api param fix** (~1.5 hours)
- [ ] Fix `parseArgsWithNames()` fallback when `parseNamedArgs` returns nil
- [ ] Add unit tests: single-param with empty body, single-param with non-matching keys
- [ ] Add regression test: multi-param with non-matching keys
- [ ] Verify backward compat: Record-typed param still receives raw body

**Phase 3: Verification** (~0.5 hours)
- [ ] `make test`
- [ ] `make lint`
- [ ] `make verify-examples`
- [ ] Manual test with `ailang serve` endpoint

### Files to Modify

**Modified files:**
- `internal/ai/gemini/types.go` — Add `fileData` struct, update `part` struct (~8 LOC)
- `internal/ai/gemini/generate.go` — Extend `buildParts()` with fileUri branch (~12 LOC)
- `internal/ai/gemini/client_test.go` — Add multimodal fileUri tests (~40 LOC)
- `internal/apiserver/handler.go` — Fix `parseArgsWithNames` fallback (~8 LOC)
- `internal/apiserver/handler_test.go` — Add param validation tests (~30 LOC)

**No new files.**

## Examples

### Example 1: fileData — Large PDF via GCS URI

**Before (base64 inline — 14MB per call):**
```json
{
  "mode": "multimodal",
  "mimeType": "application/pdf",
  "data": "JVBERi0xLjQNCjEgMCBvYmoNC...<14MB base64>...",
  "prompt": "Summarize this document"
}
```
Gemini API receives:
```json
{"contents": [{"parts": [
  {"inlineData": {"mimeType": "application/pdf", "data": "JVBERi0xLjQNC..."}},
  {"text": "Summarize this document"}
]}]}
```

**After (fileUri — <1KB per call):**
```json
{
  "mode": "multimodal",
  "mimeType": "application/pdf",
  "fileUri": "gs://my-bucket/documents/report.pdf",
  "prompt": "Summarize this document"
}
```
Gemini API receives:
```json
{"contents": [{"parts": [
  {"fileData": {"mimeType": "application/pdf", "fileUri": "gs://my-bucket/documents/report.pdf"}},
  {"text": "Summarize this document"}
]}]}
```

### Example 2: serve-api — Single-param POST with empty body

**Before (crashes):**
```bash
# Endpoint: func listApiKeys(apiKey: string) -> string
curl -X POST http://localhost:8080/listApiKeys -d '{}'
# Runtime crash: sha256Hex received Record instead of string
```

**After (zero-value):**
```bash
curl -X POST http://localhost:8080/listApiKeys -d '{}'
# Function receives apiKey = "" (empty string)
# Function can validate: if apiKey == "" then Err("missing apiKey")
```

### Example 3: Backward compatibility — Record param still works

```bash
# Endpoint: func processConfig(config: record) -> string
curl -X POST http://localhost:8080/processConfig -d '{"key": "value"}'
# Still receives {key: "value"} as before — no change
```

## Success Criteria

- [ ] `buildParts()` produces `fileData` part when input has `fileUri` field
- [ ] `buildParts()` produces `inlineData` part when input has `data` field (unchanged)
- [ ] `fileUri` takes precedence when both `fileUri` and `data` are present
- [ ] Both `gs://` and Gemini Files API URIs pass through without validation
- [ ] POST with empty body to typed-param endpoint returns zero-values, not raw Record
- [ ] POST with non-matching keys to typed-param endpoint returns zero-values
- [ ] Record-typed single-param endpoints still receive raw body
- [ ] All tests passing (`make test`)
- [ ] Lint passing (`make lint`)
- [ ] Examples verified (`make verify-examples`)

## Testing Strategy

**Unit tests (fileData):**
- `buildParts()` with `fileUri` only → produces `fileData` part
- `buildParts()` with `data` only → produces `inlineData` part (regression)
- `buildParts()` with both → produces `fileData` (precedence)
- `buildParts()` with neither `data` nor `fileUri` but `mode: multimodal` → falls through to text
- JSON marshaling of `part` with `FileData` produces correct Gemini API format

**Unit tests (serve-api):**
- `parseArgsWithNames(empty_body, ["apiKey"], ["string"])` → `[""]`
- `parseArgsWithNames(non_matching_body, ["apiKey"], ["string"])` → `[""]`
- `parseArgsWithNames(matching_body, ["apiKey"], ["string"])` → `["value"]` (regression)
- `parseArgsWithNames(body, ["a","b"], ["string","int"])` with no matches → `["", 0]`
- `parseArgsWithNames(body, [], [])` → falls through to `parseArgs` (no params = raw passthrough)

**Manual testing:**
- Run `ailang serve` with a single-param endpoint, POST empty body
- Verify response is typed error message, not crash

## Deferred Decisions

The following are intentionally left open for the implementer:

- Whether to add a `fileUri` builtin to AILANG stdlib (separate feature) — agent may defer
- Log/trace formatting for fileUri parts — agent may choose
- Whether `buildParts()` should validate mimeType is non-empty for fileData — agent may choose

## Non-Goals

**Not attempted in this feature:**
- File upload from AILANG to GCS/Files API — handled by external `sunholo/gemini_files` package
- Multi-part file arrays (multiple fileData parts in one call) — future work
- Streaming upload support — out of scope
- serve-api error responses for missing required params — separate design needed; zero-values are the v0.9.5 contract
- OpenAI/Anthropic provider fileUri support — different API shape, separate design

## Timeline

**Day 1** (~4 hours):
- Phase 1: fileData support (2h)
- Phase 2: serve-api fix (1.5h)
- Phase 3: Verification (0.5h)

**Total: ~4 hours, single day**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Gemini API rejects fileData format | Med | Test with real GCS URI before release; format matches official API spec |
| serve-api zero-value fallback breaks Record-typed endpoints | High | Guard: only apply when `len(paramNames) > 0 && len(paramTypes) > 0`; existing `parseArgs` path preserved for no-param functions |
| External callers depend on current crash behavior | Low | Crash is never desirable; zero-value is strictly better |

## Related Documents

**Implemented (directly relevant):**
- [design_docs/implemented/v0_9_5/m-serve-api-zero-value-padding.md](design_docs/implemented/v0_9_5/m-serve-api-zero-value-padding.md) — The zero-value padding design this fix extends
- [design_docs/implemented/v0_10_0/m-serve-api-agent-enhancements.md](design_docs/implemented/v0_10_0/m-serve-api-agent-enhancements.md) — Named parameter binding that created the fallback path

**Planned (check for overlap):**
- [design_docs/planned/v0_11_0/m-serve-api-dx.md](design_docs/planned/v0_11_0/m-serve-api-dx.md) — Broader serve-api developer experience improvements

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [Gemini API: fileData parts](https://cloud.google.com/vertex-ai/generative-ai/docs/multimodal/send-multimodal-prompts) — Official Vertex AI docs
- DocParse agent message `f78cf3a9` — Original fileData feature request
- DocParse agent message `d8506494` — Original POST param bug report

## Future Work

- **Multi-file support**: Allow arrays of `fileData` parts for batch document processing
- **AILANG stdlib `fileUri` builtin**: Upload files to GCS/Files API from AILANG code directly
- **Required param validation**: Return HTTP 400 errors for missing required params instead of zero-values (needs `@required` annotation design)
- **OpenAI/Anthropic file reference support**: Equivalent file URI handling for other providers

---

**Document created**: 2026-03-31
**Last updated**: 2026-03-31

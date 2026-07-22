# Sprint Plan: M-FILE-HANDLING

## Summary
Add fileData/fileUri support to the Gemini multimodal handler and fix serve-api single-param POST validation. Two focused changes that unblock large-file pipelines and prevent serve-api crashes.

**Duration:** 0.5 days (~4 hours)
**Dependencies:** None
**Risk Level:** Low

## Current Status Analysis

### Completed Recently
- cons (::) expression support: ~200 LOC in 1 day
- exit(code) builtin: ~150 LOC in 1 day
- bitwise operators: ~300 LOC in 1 day
- serve-api zero-value padding (v0.9.5): ~200 LOC in 1 day

### Velocity
- Recent average: ~200 LOC/day (focused changes with tests)
- Estimated capacity: ~100 LOC for this sprint (half-day)
- This sprint: ~98 LOC total — well within capacity

### Remaining from Design Doc
- M1: fileData support (~60 LOC impl + tests)
- M2: serve-api param fix (~38 LOC impl + tests)

## Proposed Milestones

### Milestone 1: M1_FILEDATA_SUPPORT — fileData/fileUri in Gemini handler
**Goal:** Allow Gemini multimodal calls to reference files by URI instead of base64 inline
**Estimated:** 20 LOC implementation + 40 LOC tests = 60 LOC
**Duration:** ~2 hours

**Tasks:**
1. Add `fileData` struct to `internal/ai/gemini/types.go`
2. Add `FileData *fileData` field to `part` struct
3. Extend `buildParts()` in `generate.go` to check `fileUri` field (precedence over `data`)
4. Add unit tests: fileUri-only, data-only, both-present, text-fallback
5. Add JSON marshaling test verifying correct Gemini API format

**Acceptance Criteria:**
- [ ] `buildParts()` produces `fileData` part when input has `fileUri` field
- [ ] `buildParts()` produces `inlineData` part when input has `data` field (regression)
- [ ] `fileUri` takes precedence when both fields are present
- [ ] Both `gs://` and Gemini Files API URIs pass through
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- JSON field naming mismatch with Gemini API — Mitigation: verified against official API spec (`fileData.fileUri`)

### Milestone 2: M2_SERVE_API_PARAM_FIX — Zero-value fallback for unmatched params
**Goal:** When parseNamedArgs returns nil but function has declared params, return zero-value-padded args instead of raw body
**Estimated:** 8 LOC implementation + 30 LOC tests = 38 LOC
**Duration:** ~1.5 hours

**Tasks:**
1. Fix `parseArgsWithNames()` fallback in `internal/apiserver/handler.go` (line 246-247)
2. Add guard: when `len(paramNames) > 0 && len(paramTypes) > 0`, return zero-padded args
3. Add tests: single-param empty body, single-param non-matching keys, multi-param non-matching
4. Add regression test: Record-typed param still receives raw body
5. Add regression test: no-param function still uses parseArgs path

**Acceptance Criteria:**
- [ ] POST with `{}` to `func foo(apiKey: string)` receives `""` not Record
- [ ] POST with non-matching keys to typed-param endpoint receives zero-values
- [ ] Record-typed single-param endpoints still receive raw body
- [ ] Zero-arg functions still work via parseArgs fallback
- [ ] `make test` passes
- [ ] `make lint` passes

## Success Metrics
- All tests passing: `make test`
- Lint clean: `make lint`
- Examples verified: `make verify-examples`

## Dependencies
- None — both changes are self-contained

## Open Questions
- None — all design decisions frozen in design doc

## Notes
- M1 and M2 are independent and could be done in either order
- Both are small enough to ship in a single commit each
- After completion, ack docparse messages `f78cf3a9` and `d8506494`

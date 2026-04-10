# M-MCP-HOSTED-MODE-FIXES: Fix mcpAccount LOCAL_MODE + mcpParse sample_id Resolution

**Status**: Planned
**Target**: ailang-parse v0.9.4 (requires AILANG v0.10.14 for serve-api enhancement)
**Priority**: P1 - Blocks SDK auth-interceptor work and all three published bridges
**Estimated**: 4 hours
**Dependencies**: None (can fix ailang-parse independently; serve-api enhancement optional)
**GitHub Issue**: #153

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Bug fix; no change to determinism guarantees |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | +1 | Makes Env effect usage more explicit — hosted mode detection visible |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Better error messages help agents self-recover (SAMPLE_NOT_FOUND with valid list) |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | 0 | No composition changes |
| A11: Structured Failure | +1 | Distinguishes SAMPLE_NOT_FOUND from FILE_NOT_FOUND; no more empty error messages |
| A12: System Boundary | +1 | Hosted/local boundary detection becomes explicit |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine analysis with structured errors

## Problem Statement

Two bugs in the ailang-parse MCP server block SDK development for all three published bridges (`@ailang/parse@0.3.0`, `ailang-parse 0.3.0`, `ailang-parse-go`):

### Bug 1: mcpAccount returns LOCAL_MODE on the hosted MCP endpoint

**Root cause:** All MCP tools use `getEnvOr("DOCPARSE_MODE", "local")` which defaults to `"local"`. The hosted Cloud Run deployment at `https://docparse.ailang.sunholo.com` does not have `DOCPARSE_MODE=hosted` in its environment configuration. Even if fixed in deployment, this is fragile — any future deployment that forgets this env var silently degrades to local mode.

**Affected functions (6 total):**
- `mcpAccount` (account.ail:15)
- `mcpAuth` (auth.ail:16)
- `mcpAuthPoll` (auth.ail:38)
- `mcpParse` (tools.ail:44)
- `mcpConvert` (tools.ail:269)
- `mcpEstimate` (estimate.ail:17)

**Current State:**
```
→ Call mcpAccount(apiKey="dp_valid_key", action="info") on hosted MCP
← {"error":"LOCAL_MODE","message":"Account management is not available in local mode...","suggested_fix":"Use the hosted API at https://docparse.ailang.sunholo.com for account features."}
```
The suggested_fix tells the caller to do what they're already doing — a dead-end loop.

### Bug 2: mcpParse FILE_NOT_FOUND for documented sample_id `sample_docx_basic`

**Root cause (two issues):**

1. **Renamed without alias:** `sample_docx_basic` was renamed to `sample_docx_formatting` (confirmed in release notes EML: `sample_docx_basic → sample_docx_formatting`). The `sampleResolvePath()` function was updated but no backward-compatible alias was added. All three SDKs, the landing page, API docs, integration tests, and the v0.8.0/v0.9.0 design docs still reference `sample_docx_basic`.

2. **Silent empty-string fallback:** `sampleResolvePath()` returns `""` for unknown IDs. The caller then runs `fileExists("")` → false → error "File not found: " with nothing after the colon. This violates the No Silent Fallbacks principle — it should return a distinct `SAMPLE_NOT_FOUND` error with the list of valid IDs.

**Current State:**
```
→ Call mcpParse(filepath="sample_docx_basic", outputFormat="blocks", apiKey="any", requestId="")
← {"error":"FILE_NOT_FOUND","message":"File not found: ","suggested_fix":"Check the file path or use a sample_id from mcpFormats"}
```

**Impact:**
- SDK auth-interceptor testing blocked (can't trigger AUTH_REQUIRED reliably)
- End-to-end validation broken for all three published bridges
- All SDK integration tests use `sample_docx_basic` → all fail

## Goals

**Primary Goal:** Fix both MCP server bugs so SDK bridges can authenticate and parse correctly.

**Success Metrics:**
- `mcpAccount(action="status", apiKey="dp_valid")` returns account info on hosted endpoint (not LOCAL_MODE)
- `mcpParse(filepath="sample_docx_basic")` resolves to the correct file path
- Unknown sample IDs return `SAMPLE_NOT_FOUND` error with valid sample list
- All three SDK integration test suites pass (`npm test`, `pytest`, `go test`)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Mode detection strategy: env var only vs. env + URL inference | Determines deployment robustness | agent | design | low |
| Backward-compat aliases vs. rename in SDKs | Affects published SDK versions | agent | design | low |
| SAMPLE_NOT_FOUND as new error type | SDK error handling contracts | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Mode detection: Add `DOCPARSE_API_URL` as secondary hosted-mode signal (if API URL is explicitly set, infer hosted mode)
- [x] Sample aliases: Add backward-compatible alias in `sampleResolvePath()` (cheaper than updating 3 SDKs + docs)
- [x] Error type: Add `SAMPLE_NOT_FOUND` distinct from `FILE_NOT_FOUND`

## Solution Design

### Overview

Three targeted fixes in the ailang-parse codebase, plus one optional serve-api enhancement:

### Fix 1: Robust Hosted Mode Detection

**Strategy:** Check both `DOCPARSE_MODE` and `DOCPARSE_API_URL`. If either signals hosted, use hosted mode. Also add a `CLOUD_RUN_SERVICE` env var check (automatically set by Cloud Run).

**File:** `docparse/services/mcp/account.ail` (and 5 other files)

Extract a shared helper:

```ailang
-- In docparse/services/mcp/mode.ail (new file)
module docparse/services/mcp/mode

import std/string (length)
import std/env (getEnvOr)

-- Detect whether we're running in hosted mode.
-- Checks DOCPARSE_MODE first, then infers from DOCPARSE_API_URL and
-- CLOUD_RUN_SERVICE (auto-set by Cloud Run).
export func isHostedMode() -> bool ! {Env} {
  let explicit = getEnvOr("DOCPARSE_MODE", "");
  if explicit == "hosted" then true
  else if explicit == "local" then false
  else {
    -- Infer: if API URL is set or running on Cloud Run, assume hosted
    let apiUrl = getEnvOr("DOCPARSE_API_URL", "");
    let cloudRun = getEnvOr("K_SERVICE", "");
    length(apiUrl) > 0 || length(cloudRun) > 0
  }
}
```

Then replace all 6 occurrences of:
```ailang
let mode = getEnvOr("DOCPARSE_MODE", "local");
if mode == "local" then ...
```
with:
```ailang
if not isHostedMode() then ...
```

### Fix 2: Add sample_docx_basic Alias

**File:** `docparse/services/samples.ail`

Add alias at the top of `sampleResolvePath()`:

```ailang
export pure func sampleResolvePath(sampleId: string) -> string =
  -- Backward-compatible aliases (renamed samples)
  if sampleId == "sample_docx_basic" then "data/test_files/challenge/challenge_formatting.docx"
  else if sampleId == "sample_docx_formatting" then "data/test_files/challenge/challenge_formatting.docx"
  -- ... rest unchanged
```

### Fix 3: SAMPLE_NOT_FOUND Error with Valid IDs

**File:** `docparse/services/mcp/tools.ail` (mcpParseLocal, mcpConvert)

Replace the empty-string check with explicit sample validation:

```ailang
-- Current (broken):
let resolved = if contains(filepath, ".") then filepath
               else sampleResolvePath(filepath);
if not fileExists(resolved) then ...

-- Fixed:
let isSampleId = not contains(filepath, ".");
let resolved = if isSampleId then sampleResolvePath(filepath) else filepath;

if isSampleId && length(resolved) == 0 then
  encode(jo([
    kv("error", js("SAMPLE_NOT_FOUND")),
    kv("message", js("Unknown sample_id: " ++ filepath)),
    kv("valid_samples", ja(map(js, sampleIds()))),
    kv("suggested_fix", js("Use a sample_id from this list, or pass a file path with extension."))
  ]))
else if not fileExists(resolved) then
  encode(jo([
    kv("error", js("FILE_NOT_FOUND")),
    kv("message", js("File not found: " ++ resolved)),
    kv("suggested_fix", js("Check the file path or use a sample_id from mcpFormats"))
  ]))
else ...
```

Also add a `sampleIds()` helper to `samples.ail`:

```ailang
export pure func sampleIds() -> [string] =
  ["sample_docx_basic", "sample_docx_formatting", "sample_docx_tables",
   "sample_docx_comments", "sample_docx_track_changes", "sample_docx_footnotes",
   "sample_docx_hyperlinks", "sample_docx_real_world",
   "sample_pptx_notes", "sample_pptx_formatting",
   "sample_xlsx_merged", "sample_xlsx_formulas", "sample_xlsx_formats",
   "sample_csv", "sample_markdown", "sample_html",
   "sample_odt", "sample_odp", "sample_ods", "sample_epub",
   "sample_eml_welcome", "sample_eml_release", "sample_eml_bug",
   "sample_mbox_thread", "sample_pdf", "sample_mp3", "sample_mp4"]
```

### Implementation Plan

**Phase 1: Mode Detection Fix** (~1 hour)
- [ ] Create `docparse/services/mcp/mode.ail` with `isHostedMode()`
- [ ] Update `account.ail` to use `isHostedMode()`
- [ ] Update `auth.ail` (2 functions) to use `isHostedMode()`
- [ ] Update `tools.ail` (2 functions) to use `isHostedMode()`
- [ ] Update `estimate.ail` to use `isHostedMode()`
- [ ] Add `docparse/services/mcp/mode` to `ailang.toml` exports

**Phase 2: Sample Resolution Fix** (~1 hour)
- [ ] Add `sample_docx_basic` alias to `sampleResolvePath()` in `samples.ail`
- [ ] Add `sampleIds()` helper to `samples.ail`
- [ ] Update `mcpParseLocal()` to emit `SAMPLE_NOT_FOUND` for unknown sample IDs
- [ ] Update `mcpConvert()` with same pattern
- [ ] Import `sampleIds` in `tools.ail`

**Phase 3: Testing & Docs** (~1.5 hours)
- [ ] Add inline tests to `mode.ail` for K_SERVICE/DOCPARSE_API_URL inference
- [ ] Add inline test for `sample_docx_basic` alias resolution
- [ ] Add inline test for `SAMPLE_NOT_FOUND` error on unknown ID
- [ ] Verify SDK integration tests pass: `npm test`, `pytest`, `go test`
- [ ] Update CHANGELOG.md in ailang-parse
- [ ] Deploy to Cloud Run (set `DOCPARSE_MODE=hosted` as belt-and-suspenders)

**Phase 4: Cloud Run Deployment Config** (~30 min)
- [ ] Add `DOCPARSE_MODE=hosted` to Cloud Run service env vars
- [ ] Verify mcpAccount returns account info (not LOCAL_MODE)
- [ ] Verify mcpParse with `sample_docx_basic` returns blocks

### Files to Modify/Create

**New files (ailang-parse):**
- `docparse/services/mcp/mode.ail` - Hosted mode detection helper, ~20 LOC

**Modified files (ailang-parse):**
- `docparse/services/mcp/account.ail` - Use `isHostedMode()`, ~5 LOC change
- `docparse/services/mcp/auth.ail` - Use `isHostedMode()` (2 functions), ~10 LOC change
- `docparse/services/mcp/tools.ail` - Use `isHostedMode()` + SAMPLE_NOT_FOUND, ~30 LOC change
- `docparse/services/mcp/estimate.ail` - Use `isHostedMode()`, ~5 LOC change
- `docparse/services/samples.ail` - Add alias + `sampleIds()`, ~10 LOC change
- `ailang.toml` - Add `docparse/services/mcp/mode` to exports, ~1 LOC

## Examples

### Example 1: mcpAccount on hosted endpoint (Bug 1 fix)

**Before:**
```json
{"error":"LOCAL_MODE","message":"Account management is not available in local mode..."}
```

**After (with valid key):**
```json
{"tier":"free","quota_remaining":847,"monthly_limit":1000,"ai_parses_remaining":42}
```

**After (without key):**
```json
{"error":"AUTH_REQUIRED","message":"An API key is required for account actions. Use mcpAuth to get one."}
```

### Example 2: mcpParse with sample_docx_basic (Bug 2 fix)

**Before:**
```json
{"error":"FILE_NOT_FOUND","message":"File not found: ","suggested_fix":"Check the file path or use a sample_id from mcpFormats"}
```

**After:**
```json
{"document":{"format":"docx","filename":"data/test_files/challenge/challenge_formatting.docx",...},"warnings":[],"aiCallsUsed":0}
```

### Example 3: mcpParse with unknown sample ID (new SAMPLE_NOT_FOUND)

**Before:**
```json
{"error":"FILE_NOT_FOUND","message":"File not found: ","suggested_fix":"..."}
```

**After:**
```json
{"error":"SAMPLE_NOT_FOUND","message":"Unknown sample_id: sample_docx_nonexistent","valid_samples":["sample_docx_basic","sample_docx_formatting",...],"suggested_fix":"Use a sample_id from this list, or pass a file path with extension."}
```

## Success Criteria

- [ ] `mcpAccount(action="status", apiKey="dp_...")` returns account data on hosted endpoint
- [ ] `mcpAuth()` initiates device auth on hosted endpoint (not LOCAL_MODE)
- [ ] `mcpParse(filepath="sample_docx_basic")` resolves and parses successfully
- [ ] `mcpParse(filepath="sample_nonexistent")` returns SAMPLE_NOT_FOUND with valid list
- [ ] Mode detection works via `K_SERVICE`, `DOCPARSE_API_URL`, or `DOCPARSE_MODE`
- [ ] All 6 mode-checking functions use `isHostedMode()` (no raw env var checks)
- [ ] SDK integration tests pass: JS (`npm test`), Python (`pytest`), Go (`go test`)
- [ ] All tests passing (`make test` in ailang-parse)
- [ ] Documentation updated (CHANGELOG)

## Testing Strategy

**Unit tests (inline AILANG tests):**
- `mode.ail`: Test `isHostedMode()` with various env var combinations
- `samples.ail`: Test `sampleResolvePath("sample_docx_basic")` returns correct path
- `samples.ail`: Test `sampleResolvePath("nonexistent")` returns empty string
- `samples.ail`: Test `sampleIds()` returns complete list

**Integration tests:**
- SDK JS: `npx vitest` — verify `sample_docx_basic` parse succeeds
- SDK Python: `pytest` — verify `sample_docx_basic` parse succeeds
- SDK Go: `go test ./...` — verify `sample_docx_basic` parse succeeds

**Manual testing:**
- Hit `https://docparse.ailang.sunholo.com/mcp/` with MCP client, verify mcpAccount works
- Verify mcpAuth returns device auth URL (not LOCAL_MODE)
- Verify mcpParse with `sample_docx_basic` returns parsed blocks

## Deferred Decisions

- Deprecation timeline for `sample_docx_basic` alias — agent may add a `deprecated_aliases` field in mcpFormats output in a future release
- Whether to inject `_server_mode` from serve-api framework — optional enhancement, not needed for this fix

## Non-Goals

- **Serve-api framework changes** — The fix is entirely in ailang-parse .ail code; no Go changes needed
- **SDK version bumps** — The SDKs reference `sample_docx_basic` which will work again with the alias
- **Cloud Run infrastructure changes** — We fix in code; deployment env var is belt-and-suspenders

## Timeline

**Day 1** (4 hours):
- Phase 1: Mode detection (1h)
- Phase 2: Sample resolution (1h)
- Phase 3: Testing & docs (1.5h)
- Phase 4: Deploy (0.5h)

**Total: ~4 hours, single day**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| K_SERVICE env var not available in all Cloud Run revisions | Low | Belt-and-suspenders: also set DOCPARSE_MODE=hosted explicitly |
| Old clients cache LOCAL_MODE errors | Low | MCP clients don't cache tool results; fix is immediate |
| sample_docx_basic alias confuses new users | Low | Document as alias in mcpFormats output |

## Related Documents

**ailang-parse design docs:**
- `design_docs/implemented/v0_9_0/mcp_server.md` — Original MCP server design (references sample_docx_basic)
- `design_docs/implemented/v0_8_0/agent_friendly_api.md` — API design (references sample_docx_basic)

**AILANG serve-api:**
- `internal/apiserver/mcp.go` — MCP server framework
- `internal/apiserver/routes.go` — `_headers` injection mechanism

## References

- [GitHub Issue #153](https://github.com/sunholo-data/ailang/issues/153) — Original bug report
- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- `ailang-parse/docparse/services/mcp/account.ail` — Bug 1 source
- `ailang-parse/docparse/services/mcp/tools.ail` — Bug 2 source
- `ailang-parse/docparse/services/samples.ail` — Sample ID registry

## Future Work

- Add a `_server_context` parameter in serve-api that MCP tools can optionally accept to detect transport mode (HTTP vs stdio) without env vars
- Deprecate `sample_docx_basic` alias after SDK major version bump
- Consider a sample registry backed by a data file (TOML/JSON) instead of hardcoded if-else chain

---

**Document created**: 2026-04-10
**Last updated**: 2026-04-10

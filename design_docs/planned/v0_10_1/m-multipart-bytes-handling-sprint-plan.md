# Sprint Plan: M-MULTIPART-BYTES

## Summary
Fix multipart file upload crash in serve-api by adding named parameter matching and type-aware coercion (BytesValue→temp file path for string params, BytesValue passthrough for bytes params).

**Duration:** 1 day (~4 hours)
**Dependencies:** None — BytesValue type, bytes builtins, and named arg parsing all exist
**Risk Level:** Low
**Design Doc:** [m-multipart-bytes-handling.md](m-multipart-bytes-handling.md)

## Current Status Analysis

### Completed Recently
- ✅ fileData/fileUri Gemini handler: ~80 LOC in 1 day (commit 8697c9d0)
- ✅ serve-api POST param validation fix: ~40 LOC in 1 day (commit 8697c9d0)
- ✅ cons (::) expression support: ~200 LOC in 1 day (commit 55e9c656)
- ✅ Bitwise operators: ~300 LOC in 1 day (commit e4991339)

### Velocity
- Recent average: ~200-400 LOC/day (focused features)
- This sprint: ~140 LOC implementation + ~80 LOC tests = ~220 LOC total
- Well within single-day capacity

### Remaining from Design Doc
- ⏳ M-MULTIPART-BYTES: Named multipart parsing + type coercion (~220 LOC)

## Proposed Milestones

### Milestone 1: M1_NAMED_MULTIPART_PARSER
**Goal:** Add `parseMultipartArgsWithNames()` that maps multipart field names to function parameter names, with type-aware coercion and temp file management.
**Estimated:** 90 LOC implementation + 50 LOC tests = 140 LOC
**Duration:** ~2 hours

**Tasks:**
1. Add `writeTempFile()` helper to `internal/apiserver/routes.go` — writes bytes to temp file preserving extension
2. Add `parseMultipartArgsWithNames()` to `internal/apiserver/routes.go`:
   - Match multipart field names to `paramNames`
   - File field + string param → write temp file, return path
   - File field + bytes param → return BytesValue directly
   - Non-file field → return string value
   - Unmatched params → zero-value padding via `zeroValueForType()`
   - Return cleanup function for temp file removal
3. Unit tests for `parseMultipartArgsWithNames()`:
   - File field + string param → temp file path
   - File field + bytes param → BytesValue
   - Non-file field matching → string value
   - Unmatched param → zero-value
   - No paramNames → falls back to positional `parseMultipartArgs`
   - Cleanup function removes temp files
4. Unit test for `writeTempFile()` — extension preserved, file written

**Acceptance Criteria:**
- [ ] `parseMultipartArgsWithNames` maps fields by name, not position
- [ ] File field + string param produces temp file path (not BytesValue)
- [ ] File field + bytes param produces BytesValue (no conversion)
- [ ] Unmatched params get zero-values
- [ ] Cleanup function removes all temp files
- [ ] Falls back to positional `parseMultipartArgs` when no paramNames

**Risks:**
- None significant — follows established `parseArgsWithNames` pattern

### Milestone 2: M2_CALLSITE_INTEGRATION
**Goal:** Wire `parseMultipartArgsWithNames` into the multipart branch of `callFunction()` and verify end-to-end.
**Estimated:** 20 LOC implementation + 30 LOC tests = 50 LOC
**Duration:** ~1 hour

**Tasks:**
1. Update multipart branch in `callFunction()` (`routes.go:288-311`):
   - Call `parseMultipartArgsWithNames(r, maxSize, opt.ParamNames, opt.ParamTypes)`
   - Wire `defer cleanup()` for temp file removal
2. Integration test: multipart POST → AILANG string-typed param → no crash
3. Regression test: multipart POST without paramNames → existing behavior preserved
4. Run `make test`, `make lint`, `make verify-examples`

**Acceptance Criteria:**
- [ ] Multipart POST to string-typed endpoint works (no `_str_len` crash)
- [ ] Multipart POST to bytes-typed endpoint passes BytesValue
- [ ] Existing multipart behavior preserved for legacy endpoints
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- Temp file leak if panic before defer runs — low risk, OS cleans `/tmp` periodically

### Milestone 3: M3_VERIFICATION
**Goal:** Final verification, cleanup, and documentation.
**Estimated:** 30 LOC (CHANGELOG + example updates)
**Duration:** ~30 minutes

**Tasks:**
1. Update CHANGELOG.md with fix description
2. Run full `make ci` verification
3. Verify no temp file leaks with manual test

**Acceptance Criteria:**
- [ ] CHANGELOG.md updated
- [ ] `make ci` passes
- [ ] No temp files left after request completes

## Success Metrics
- All tests passing: `make test` ✅
- All linting passing: `make lint` ✅
- Examples verified: `make verify-examples` ✅
- DocParse multipart upload scenario works (string param receives temp file path)
- No regression for existing multipart or JSON endpoints

## Dependencies
- None — all required infrastructure exists

## Open Questions
- None — design decisions resolved in design doc

## Notes
- This is a focused bug fix sprint, not a feature sprint
- The fix follows the established pattern from `parseArgsWithNames` (JSON named binding)
- Existing `parseMultipartArgs` is preserved as fallback for backward compat

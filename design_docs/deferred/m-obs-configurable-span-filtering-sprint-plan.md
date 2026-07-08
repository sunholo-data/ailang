# Sprint Plan: M-OBS-CONFIGURABLE-SPAN-FILTERING

## Summary

Extract hard-coded span filtering in `shouldFilterSpan()` into a configurable `SpanFilterConfig` struct loaded from environment variables, enabling operators to override allow/deny rules without recompiling.

**Duration:** 1 session (~4 hours)
**Dependencies:** None
**Risk Level:** Low (small refactor of pure function, good existing test coverage)

## Current Status Analysis

### Completed Recently
- v0.9.1.1: Z3 verification fixes, stdlib embedding (~400 LOC in 2 days)
- v0.9.1: SMT-V2 verification + cloud plugin skills (~2000 LOC in 5 days)

### Velocity
- Recent average: ~300-400 LOC/day
- This sprint: ~180 LOC total (well within single-session capacity)

### Remaining from Design Doc
- ⏳ Phase 1: Extract config struct (~80 LOC)
- ⏳ Phase 2: Refactor shouldFilterSpan (~30 LOC)
- ⏳ Phase 3: Tests (~100 LOC)

## Proposed Milestones

### M1: SpanFilterConfig Struct & Loader
**Goal:** Create the config struct, pattern types, default builder, and env var parser
**Estimated:** 80 LOC implementation = 80 LOC
**Duration:** ~1.5 hours

**Tasks:**
1. Add `SpanFilterConfig` and `FilterPattern` structs to `otlp_receiver.go`
2. Add `filterConfig` field to `OTLPReceiver` struct
3. Implement `DefaultSpanFilterConfig()` — encodes all current hard-coded rules
4. Implement `loadSpanFilterConfig()` — parses `AILANG_SPAN_FILTER_ALLOW`, `AILANG_SPAN_FILTER_DENY`, `AILANG_SPAN_FILTER_DISABLE`
5. Update `NewOTLPReceiver()` to call `loadSpanFilterConfig()` and log active config
6. Implement pattern parsing: `name` (exact), `name*` (prefix), `*name` (suffix), `service:name` (service-scoped)

**Acceptance Criteria:**
- [ ] `SpanFilterConfig` struct defined with `AllowPatterns`, `DenyPatterns`, `DisableAll`
- [ ] `DefaultSpanFilterConfig()` returns config matching current hard-coded rules exactly
- [ ] Env var parsing handles comma-separated patterns with prefix/exact/suffix/service syntax
- [ ] Config logged at startup (pattern counts)
- [ ] `make build` passes

### M2: Refactor shouldFilterSpan to Use Config
**Goal:** Replace hard-coded checks with config-driven matching; allow takes priority over deny
**Estimated:** 30 LOC changed
**Duration:** ~1 hour

**Tasks:**
1. Change `shouldFilterSpan` from package-level function to method on `*OTLPReceiver`
2. Implement matching logic: check `DisableAll` first → check allow-list → check deny-list → default keep
3. Add `matchesPattern(name string, resourceAttrs map[string]any, pattern FilterPattern) bool` helper
4. Update call site in `processResourceSpans` (line 691)
5. Run `make test` to verify backward compatibility

**Acceptance Criteria:**
- [ ] `shouldFilterSpan` uses `r.filterConfig` instead of hard-coded lists
- [ ] Allow-list takes priority over deny-list
- [ ] `AILANG_SPAN_FILTER_DISABLE=true` passes all spans through
- [ ] No behavioral change when no env vars are set
- [ ] `make test` passes (existing `TestShouldFilterSpan` still green)

### M3: Tests
**Goal:** Comprehensive tests for config loading and filter behavior
**Estimated:** 100 LOC tests
**Duration:** ~1.5 hours

**Tasks:**
1. `TestDefaultSpanFilterConfig` — verify defaults produce identical filtering to old hard-coded logic
2. `TestSpanFilterAllow` — allow-listed patterns bypass deny
3. `TestSpanFilterDeny` — custom deny patterns block spans
4. `TestSpanFilterDisable` — disable mode passes everything
5. `TestSpanFilterServiceScoped` — `service:name` syntax works correctly
6. `TestLoadSpanFilterConfig` — env var parsing with various formats
7. `TestSpanFilterAllowOverridesDeny` — allow takes priority
8. Run `make lint` and `make test` to verify clean

**Acceptance Criteria:**
- [ ] 7+ new test cases covering all filter modes
- [ ] `make test` all green
- [ ] `make lint` clean
- [ ] Backward compatibility confirmed (no env vars = same behavior)

## Success Metrics
- All existing tests passing: `make test`
- Linting clean: `make lint`
- 7+ new test cases for filter config
- No behavioral change without env vars (backward compatible)
- Total LOC: ~180 (80 impl + 100 tests)

## Dependencies
- None — self-contained change in `internal/observatory/`

## Open Questions
- None — design doc fully specified

## Notes
- `shouldFilterSpan` is currently a package-level function — making it a method is the cleanest approach
- Existing `TestShouldFilterSpan` has 8 test cases — all must continue passing
- The `otlp_receiver.go` file is 1043 lines — this change adds ~80 LOC, keeping it under the 1200 line threshold

# Sprint Plan: M-SERVE-API-ZERO-VALUE-PADDING

## Summary

Implement zero-value padding for missing named parameters in serve-api so AILANG functions can validate inputs and return structured errors instead of crashing on unit values.

**Duration:** 1 day (~4 hours)
**Dependencies:** M-SERVE-API-AGENT-ENHANCEMENTS (already implemented)
**Risk Level:** Low
**Design Doc:** [m-serve-api-zero-value-padding.md](m-serve-api-zero-value-padding.md)

## Current Status Analysis

### Completed Recently
- M-SERVE-API-AGENT-ENHANCEMENTS: Named JSON binding + @nowrap headers (v0.9.2)
- M-AI-IMAGE: AI image generation in std/ai

### Velocity
- Recent average: ~200-300 LOC/day (serve-api sprint was ~400 LOC in 1 day)
- Estimated capacity: 150 LOC is well within a single session

### Remaining from Design Doc
- All work is new — no partial implementation exists

## Proposed Milestones

### M1: EXTRACT_PARAM_TYPES
**Goal:** Extract per-parameter type strings from AST alongside param names
**Estimated:** ~35 LOC implementation + ~25 LOC tests = ~60 LOC total
**Duration:** ~1 hour

**Tasks:**
1. Add `ParamTypes []string` field to `ExportInfo` in `server.go`
2. Add `ParamTypes []string` field to `callOpts` in `routes.go`
3. Rename `extractParamNames()` → `extractParamInfo()` in `routes.go`
4. Add `typeToString(ast.Type) string` helper in `routes.go`
5. Wire `ParamTypes` through `callOpts` at all 3 call sites (handler.go:100, routes.go:120, routes.go:149)
6. Update `TestExtractParamNames` to verify types are extracted

**Acceptance Criteria:**
- [ ] `ExportInfo.ParamTypes` populated for exported functions with type annotations
- [ ] `typeToString` handles string, int, float, bool, list, array, record → correct name
- [ ] Unknown/complex types return "unknown"
- [ ] `make test` passes (existing tests still pass)

**Risks:**
- `ast.Param.Type` may be nil for inferred params → Mitigation: default to "unknown"

### M2: ZERO_VALUE_PADDING
**Goal:** Pad missing named params and short positional args with type-appropriate zero-values
**Estimated:** ~40 LOC implementation + ~40 LOC tests = ~80 LOC total
**Duration:** ~1.5 hours

**Tasks:**
1. Add `zeroValueForType(typeName string) interface{}` in `handler.go`
2. Update `parseNamedArgs()` signature to accept `paramTypes []string`
3. Pad unmatched slots with `zeroValueForType(paramTypes[i])` instead of leaving nil
4. Update `parseArgsWithNames()` to pass `paramTypes` through
5. Add positional arg padding when `{"args": [...]}` has fewer elements than params
6. Update existing `TestParseNamedArgs` "partial match" test (now expects `""` not nil)
7. Add new tests: zero-value per type, positional padding, unknown type stays nil

**Acceptance Criteria:**
- [ ] Missing string param → `""` (not nil/unit)
- [ ] Missing int param → `0`
- [ ] Missing bool param → `false`
- [ ] Missing list param → `[]`
- [ ] Unknown type → nil (backward compatible)
- [ ] Positional `{"args": ["x"]}` with 3-param function → pads remaining with zero-values
- [ ] Full named binding (no missing params) unchanged
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- Changing `parseNamedArgs` signature breaks callers → Mitigation: only 1 caller (`parseArgsWithNames`), update in same commit

### M3: DOCS_AND_OPENAPI
**Goal:** Update OpenAPI spec and documentation to reflect zero-value padding behavior
**Estimated:** ~15 LOC code + ~20 lines docs = ~35 LOC total
**Duration:** ~30 minutes

**Tasks:**
1. Add `x-ailang-param-types` to OpenAPI operation metadata
2. Mark params with known zero-values as not required in schema
3. Update `docs/docs/guides/serve-api.md` with zero-value padding section
4. Update `changelogs/v0.9-current.md`

**Acceptance Criteria:**
- [ ] OpenAPI `/openapi.json` includes `x-ailang-param-types`
- [ ] Changelog documents the DX fix
- [ ] serve-api guide explains zero-value behavior
- [ ] `make lint` passes

**Risks:**
- None — documentation-only changes

## Success Metrics
- Test coverage: >80% for new code paths
- All existing apiserver tests still pass
- `make test` clean
- `make lint` clean
- Changelog updated
- serve-api docs updated

## Open Questions
- None — design doc covers all decisions

## Notes
- This is a direct follow-on to M-SERVE-API-AGENT-ENHANCEMENTS
- The "no silent fallbacks" principle is addressed: zero-values are explicit, documented, and functions MUST validate — this is not a silent fallback but a typed default at the HTTP boundary
- Source: docparse agent message `08f8af96`

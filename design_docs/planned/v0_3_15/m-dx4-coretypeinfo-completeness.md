# M-DX4: CoreTypeInfo Completeness & Type-Guided Lowering

**Status**: Planned
**Target**: v0.3.15
**Priority**: P0 - High (Compiler Correctness)
**Estimated**: 6 hours (4h implementation + 1.5h testing + 0.5h docs)
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Internal compiler improvement, no syntax changes |
| Preserve Semantic Clarity | + | +1 | Better error messages make semantics clearer |
| Increase Determinism | + | +1 | Eliminates silent fallbacks and runtime panics |
| Lower Token Cost | + | +1 | Fewer debugging cycles = lower conversation cost |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

CoreTypeInfo is currently a partial map (Core NodeID → principal type), but type-guided lowering assumes it's total. Missing entries cause cryptic runtime panics or silent fallbacks that hide bugs.

**Current State:**
- Missing CoreTypeInfo entries for Float, Bool, nested lets, and comparison operators
- Lowering code uses fallbacks when types are absent
- Errors like "cannot lower unknown variant" with no context
- No validation that CoreTypeInfo is complete before lowering
- Developer has no way to inspect or debug missing type information

**Impact:**
- **Who is affected?** All AILANG developers (AI and human)
- **How significant?** P0 - Compiler panics and wrong code generation are critical bugs
- **Example**: Float literals in lambda bodies crash with "unknown variant" at runtime

## Goals

**Primary Goal:** Make CoreTypeInfo a total function: every Core NodeID → principal type, validated before lowering.

**Success Metrics:**
- 100% CoreTypeInfo coverage for all Core nodes (validated by CI)
- Zero "unknown variant" runtime errors in lowering
- All errors include NodeID, expression kind, and suggested fixes
- Developers can inspect CoreTypeInfo with `ailang debug types --show-gaps`

## Solution Design

### Overview

Implement a validation pass that walks the entire Core AST and ensures every node has a CoreTypeInfo entry. Run this validation before lowering and fail compilation if gaps are found. Improve error messages to include full context.

### Architecture

**Components:**
1. **CoreTypeInfo Validator** (`internal/pipeline/validate_coretypeinfo.go`): Walk Core AST, check that every node exists in CoreTypeInfo map
2. **CoreTypeInfo Populator** (`internal/elaborate/populate_coretypeinfo.go`): Ensure all Core node variants populate CoreTypeInfo during elaboration
3. **Enhanced Error Diagnostics** (`internal/errors/lowering_errors.go`): Wrap errors with NodeID, expression kind, type hint, and position

### Implementation Plan

**Phase 1: Validation Pass** (~2 hours)
- [ ] Create `Pipeline.validateCoreTypeInfo()` walker function
- [ ] Walk all Core node variants (Var, Let, Lit, Intrinsic, App, Lam, If, Case, Handle, ...)
- [ ] For each NodeID, check existence in CoreTypeInfo map
- [ ] Collect missing NodeIDs with their AST context
- [ ] Return error with list of missing entries
- [ ] Add validation call in `Pipeline.Run()` before lowering

**Phase 2: Complete CoreTypeInfo Population** (~2 hours)
- [ ] Fix Float literal CoreTypeInfo (add to `elaborate/literals.go`)
- [ ] Fix Bool literal CoreTypeInfo (add to `elaborate/literals.go`)
- [ ] Fix nested let CoreTypeInfo (verify `elaborate/let.go`)
- [ ] Fix comparison operators in lambda bodies (verify `elaborate/operators.go`)
- [ ] Fix all Intrinsic nodes (verify `elaborate/intrinsics.go`)
- [ ] Audit all Core node construction sites for CoreTypeInfo population

**Phase 3: Error Diagnostics** (~1 hour)
- [ ] Create `errors.NewLoweringError(nodeID, exprKind, pos, hint)`
- [ ] Replace all `fmt.Errorf` in lowering with contextual errors
- [ ] Include "check type with: ailang debug types --show-gaps" in hints
- [ ] Add color-coded output for error messages

**Phase 4: CI Enforcement** (~0.5 hours)
- [ ] Add validation gate in CI: fail build if CoreTypeInfo coverage < 100%
- [ ] Add regression tests for Float, Bool, nested lets, operators
- [ ] Test that validation catches missing entries before lowering

### Files to Modify/Create

**New files:**
- `internal/pipeline/validate_coretypeinfo.go` - CoreTypeInfo validator (~150 LOC)
- `internal/errors/lowering_errors.go` - Enhanced error types (~100 LOC)

**Modified files:**
- `internal/pipeline/pipeline.go` - Add validation call (~10 LOC)
- `internal/elaborate/literals.go` - Fix Float/Bool CoreTypeInfo (~20 LOC)
- `internal/elaborate/operators.go` - Fix comparison operators (~15 LOC)
- `internal/lower/lower.go` - Use enhanced errors (~30 LOC changes)
- `.github/workflows/ci.yml` - Add validation check (~5 LOC)

## Examples

### Example 1: Missing Float CoreTypeInfo

**Before:**
```
λ> (\x -> x + 0.5) 42
panic: cannot lower unknown variant for NodeID 1337
```

**After:**
```
λ> (\x -> x + 0.5) 42
Error: Missing type information for Core node
  NodeID: 1337
  Expression: Float literal (0.5)
  Location: lambda body at line 1:14

Hint: This is a compiler bug. CoreTypeInfo should be populated during elaboration.
Debug with: ailang debug types --show-gaps
```

### Example 2: Validation Catches Gap Before Lowering

**Before:**
```
# Compilation proceeds silently, crashes later in lowering
```

**After:**
```
Error: CoreTypeInfo validation failed - found 3 nodes without type information:
  NodeID 1337: Float literal at line 4:10
  NodeID 1338: Comparison operator (<=) at line 5:15
  NodeID 1339: Nested let binding at line 6:5

This is a compiler bug. All Core nodes must be typed before lowering.
Debug with: ailang debug types --show-gaps
```

## Success Criteria

- [ ] 100% CoreTypeInfo coverage validated by walker (all Core nodes checked)
- [ ] CI fails if validation finds missing entries
- [ ] All existing tests pass with validation enabled
- [ ] Regression tests for Float, Bool, comparison operators, nested lets
- [ ] Error messages include NodeID, expression kind, position, and actionable hints
- [ ] All tests passing
- [ ] Documentation updated (docs/architecture/types.md)
- [ ] Examples added (test cases in `internal/pipeline/validate_coretypeinfo_test.go`)

## Testing Strategy

**Unit tests:**
- Test validator catches missing Float CoreTypeInfo
- Test validator catches missing Bool CoreTypeInfo
- Test validator catches missing comparison operator types
- Test validator reports multiple missing entries
- Test enhanced error messages include all required fields

**Integration tests:**
- Compile lambda with Float literal: `(\x -> x + 0.5) 42`
- Compile lambda with comparison: `(\x -> x <= 5) 3`
- Compile nested let: `let x = 1 in let y = 2 in x + y`
- Verify all examples pass through validator without errors

**Manual testing:**
- Run `make test` with validation enabled
- Check CI catches validation failures
- Verify error messages are actionable

## Non-Goals

**Not in this feature:**
- Type inference improvements - This is about validation, not inference
- Reflection/quasiquotes - Out of scope for v0.3.15
- `ailang debug types` CLI - Deferred to M-DX5
- Performance optimization of validator - Correctness first, optimize later if needed

## Timeline

**Day 1** (4 hours):
- Phase 1: Validation pass implementation
- Phase 2: CoreTypeInfo population fixes

**Day 2** (2 hours):
- Phase 3: Error diagnostics
- Phase 4: CI enforcement
- Testing and documentation

**Total: ~6 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Validator false positives | High | Comprehensive tests for all Core node variants |
| Performance regression | Low | Validation is single-pass, O(n) in AST size |
| Breaking existing code | Medium | All existing tests must pass with validation enabled |

## References

- Field report: User's "AI-first DX reflection" from October 2025
- Related: M-DX1 (Developer Experience) - Similar compiler correctness focus
- Related: M-DX5 (Inference Visibility) - Will add debug CLI later
- [CLAUDE.md](../../../CLAUDE.md#2-no-silent-fallbacks---fail-loudly) - Fail loudly principle

## Future Work

- M-DX5: Add `ailang debug types --trace-inference` to show inference flow
- M-DX5: Add `ailang debug types --show-gaps` to list missing CoreTypeInfo
- Performance profiling: Measure validation overhead (likely negligible)
- Literal resolution table: Document Int↔Float promotion rules (M-DX5)

---

**Document created**: 2025-10-22
**Last updated**: 2025-10-22

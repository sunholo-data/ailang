# M-DX8: Silent Failure Prevention

**Status**: Planned
**Target**: v0.4.1
**Priority**: P1 (Medium-High)
**Estimated**: 3-4 days
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | Neutral | 0 | Internal tooling, no syntax impact |
| Preserve Semantic Clarity | Positive | +2 | **Prevents silent bugs** - failures become loud and clear |
| Increase Determinism | Positive | +2 | **Eliminates silent fallbacks** - behavior is predictable |
| Lower Token Cost | Positive | +1 | Reduces debug time → fewer context-switching tokens for AI |
| **Net Score** | | **+5** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Rationale**: Silent failures violate AILANG's determinism principle. When code falls through to default cases or uses wrong type variants (TVar vs TVar2), debugging takes hours instead of minutes. Making failures loud preserves semantic clarity (+2), increases determinism (+2), and reduces token cost for AI debugging (+1).

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Discovered during**: M-POLY-B Phase 1 implementation (October 2025)

While implementing var-bound polymorphic lambda support, we discovered **five distinct silent failures** that cost 4-6 hours of debugging time:

1. **TVar/TVar2 duality**: Pattern matching on `*types.TVar` when actual type is `*types.TVar2` → silent failure
2. **Incomplete switch statements**: `cloneExpr()` missing `Let` case → default returns unchanged → silent
3. **Wrong map keys**: Building `typeSubst["x"]` but looking up `typeSubst["α2"]` → not found, silent
4. **Default cases hide bugs**: Both `specializeExpr` and `cloneExpr` return unchanged on unknown types → silent

**Current State:**
- **26 files** use `*types.TVar` pattern matching (found via `grep -r "case \*types.TVar"`)
- **Unknown** how many also check `*types.TVar2` (manual audit required)
- **Multiple switch statements** with default cases that return expressions unchanged
- **Zero compile-time warnings** when a case is missing
- **Zero runtime warnings** in production mode when default case is hit

**Impact:**
- **Who**: AI assistants and human developers working on AILANG compiler
- **Significance**: High - M-POLY-B debugging took 6 hours, could have been 2-3 hours with better diagnostics
- **Scope**: Affects type system, monomorphization, AST traversal, operator lowering
- **Current workaround**: Add verbose debug logging (`DEBUG_MONO_VERBOSE=1`), manual inspection

**Pain Points from M-POLY-B:**
- Spent 2+ hours debugging `extractParamTVars()` returning `[]` (was checking TVar, should check TVar2)
- Spent 1.5 hours discovering `cloneExpr()` stopped at `Let` nodes (missing case)
- Spent 1 hour tracing type substitution map key mismatch (`"x"` vs `"α2"`)
- Total: **4.5 hours lost to silent failures**, 33% of total debug time

## Goals

**Primary Goal:** Eliminate silent failures in type system and AST traversal code, reducing AI debugging time by 50-70%.

**Success Metrics:**
- **Zero silent TVar/TVar2 failures** - all sites checked for both variants
- **Zero silent switch statement gaps** - incomplete switches fail loudly in debug mode
- **Reduce debug time** from 6 hours → 2-3 hours for similar bugs
- **Catch 80%+ of bugs** via compile-time or debug-mode checks
- **AI confidence** - Claude can trust that missing cases will be caught

## Solution Design

### Overview

**Three-pronged approach:**

1. **TVar/TVar2 Unification** - Migrate to single type, add helper functions, document usage
2. **AST Coverage Testing** - Ensure all Core node types handled by traversal functions
3. **Debug Mode Enforcement** - Make default cases fail loudly when `DEBUG_STRICT=1`

**Strategy**: Start with quick wins (debug mode, helpers), then systematic migration (TVar2), then long-term prevention (coverage tests).

### Architecture

**Component 1: TVar/TVar2 Unification**

**Goal**: Choose one type variable representation and migrate to it.

**Recommendation**: Standardize on `TVar2` (newer, more complete)

**Approach**:
1. Add helper function to abstract the duality:
   ```go
   // internal/types/helpers.go (NEW)
   func ExtractTVarName(t Type) (string, bool) {
       switch tv := t.(type) {
       case *TVar:
           return tv.Name, true
       case *TVar2:
           return tv.Name, true
       default:
           return "", false
       }
   }

   func IsTVar(t Type) bool {
       _, ok := ExtractTVarName(t)
       return ok
   }
   ```

2. Create migration guide in `docs/development/type-variable-migration.md`

3. Add deprecation warnings:
   ```go
   // In TVar.String()
   func (t *TVar) String() string {
       if os.Getenv("DEBUG_STRICT") != "" {
           log.Printf("WARNING: TVar is deprecated, use TVar2. Location: %s",
               debug.Stack())
       }
       return fmt.Sprintf("α_%s", t.Name)
   }
   ```

4. Systematic migration:
   - Phase 1: Use helpers in new code
   - Phase 2: Migrate internal/types/ package
   - Phase 3: Migrate internal/pipeline/ package
   - Phase 4: Migrate remaining packages
   - Phase 5: Remove TVar entirely (v0.5.0+)

**Component 2: AST Coverage Testing**

**Goal**: Ensure all traversal functions handle all Core node types.

**Approach**:
1. Generate list of all Core node types via reflection
2. For each traversal function, create coverage test
3. Test ensures no node hits default case

**Example test**:
```go
// internal/pipeline/specialize_coverage_test.go (NEW)
func TestCloneExprCoversAllNodeTypes(t *testing.T) {
    // Get all Core node types
    allNodeTypes := core.AllNodeTypes() // generates via reflection

    s := NewSpecializer(&types.CoreTypeInfo{})
    typeSubst := make(map[string]types.Type)

    for _, nodeType := range allNodeTypes {
        // Create minimal valid node of this type
        node := core.NewMinimalNode(nodeType)

        // Ensure cloneExpr doesn't hit default case
        cloned, err := s.cloneExpr(node, typeSubst)

        // Check that node was actually cloned (has fresh NodeID)
        require.NoError(t, err, "cloneExpr failed for %T", node)
        require.NotEqual(t, node.ID(), cloned.ID(),
            "cloneExpr returned unchanged node for %T - missing case?", node)
    }
}
```

**Component 3: Debug Mode Enforcement**

**Goal**: Make silent failures loud when `DEBUG_STRICT=1` is set.

**Approach**:
1. Add `DEBUG_STRICT` environment variable check
2. Modify default cases to panic in strict mode
3. Document in CLAUDE.md for AI developers

**Example**:
```go
// internal/pipeline/specialize.go
func (s *Specializer) cloneExpr(expr core.CoreExpr, typeSubst map[string]types.Type) (core.CoreExpr, error) {
    switch e := expr.(type) {
    case *core.Var:
        // ... handle Var
    case *core.Let:
        // ... handle Let
    // ... other cases

    default:
        if os.Getenv("DEBUG_STRICT") != "" {
            // In strict mode, panic to force developer to add case
            panic(fmt.Sprintf("cloneExpr: unhandled node type %T (NodeID %d). "+
                "Add a case for this type or explicitly mark as unsupported.",
                expr, expr.ID()))
        }

        // In production, log warning and return unchanged
        if os.Getenv("DEBUG_VERBOSE") != "" {
            fmt.Fprintf(os.Stderr, "[WARN] cloneExpr: unhandled type %T (NodeID %d), "+
                "returning unchanged\n", expr, expr.ID())
        }
        return expr, nil
    }
}
```

**Component 4: Linter Rules** (Optional, Phase 2)

**Goal**: Catch pattern matching issues at compile time.

**Approach**:
1. Create custom `golangci-lint` plugin
2. Check for `case *types.TVar:` without `case *types.TVar2:`
3. Warn on switch statements with only `default:` case

### Implementation Plan

**Phase 1: Quick Wins** (~1 day / 8 hours)
- [ ] Add `ExtractTVarName` and `IsTVar` helpers (1h)
- [ ] Add `DEBUG_STRICT` mode to key functions (2h)
  - `cloneExpr()`
  - `specializeExpr()`
  - `substituteType()`
  - `extractParamTVars()`
- [ ] Document `DEBUG_STRICT` in CLAUDE.md (0.5h)
- [ ] Add TVar deprecation warnings in debug mode (1h)
- [ ] Test with M-POLY-B test cases (0.5h)
- [ ] Write migration guide `docs/development/type-variable-migration.md` (1h)
- [ ] Create GitHub issue for systematic TVar→TVar2 migration (0.5h)
- [ ] Buffer (1.5h)

**Phase 2: AST Coverage Tests** (~1 day / 8 hours)
- [ ] Create `core.AllNodeTypes()` reflection function (2h)
- [ ] Create `core.NewMinimalNode()` factory (2h)
- [ ] Write `TestCloneExprCoversAllNodeTypes` (1.5h)
- [ ] Write `TestSpecializeExprCoversAllNodeTypes` (1.5h)
- [ ] Run coverage tests, fix any gaps found (1h)

**Phase 3: Systematic TVar2 Migration** (~1-2 days / 8-16 hours)
- [ ] Audit all `case *types.TVar:` sites (2h)
  - `grep -rn "case \*types.TVar:" internal/`
  - Document which need TVar2 handling
- [ ] Migrate internal/types/ package (3h)
- [ ] Migrate internal/pipeline/ package (3h)
- [ ] Migrate remaining packages (2h)
- [ ] Remove TVar deprecation warnings (0.5h)
- [ ] Update tests (2h)
- [ ] Buffer (3.5h)

**Phase 4: Linter Rules** (Optional, ~1 day / 8 hours)
- [ ] Research golangci-lint plugin development (2h)
- [ ] Implement TVar/TVar2 check rule (3h)
- [ ] Implement default-only switch warning (2h)
- [ ] Integrate into `make lint` (1h)

### Files to Modify/Create

**New files:**
- `internal/types/helpers.go` - TVar helper functions (~50 LOC)
- `internal/core/reflection.go` - AllNodeTypes, NewMinimalNode (~150 LOC)
- `internal/pipeline/specialize_coverage_test.go` - AST coverage tests (~200 LOC)
- `docs/development/type-variable-migration.md` - Migration guide (~300 LOC)
- `.github/workflows/lint-custom.yml` - Custom linter config (optional, ~50 LOC)

**Modified files:**
- `internal/pipeline/specialize.go` - Add DEBUG_STRICT checks (~50 LOC)
- `internal/types/types.go` - Add deprecation warnings to TVar (~20 LOC)
- `internal/types/*.go` - Migrate TVar → TVar2 (~200 LOC changes)
- `internal/pipeline/*.go` - Migrate TVar → TVar2 (~100 LOC changes)
- `CLAUDE.md` - Document DEBUG_STRICT mode (~50 LOC)

**Total new code:** ~750 LOC
**Total modified code:** ~420 LOC
**Total: ~1,170 LOC**

## Examples

### Example 1: TVar/TVar2 Silent Failure (Before M-DX8)

**Before:**
```go
// internal/pipeline/specialize.go (BUGGY)
func extractParamTVars(funcType types.Type, expectedParams int) []string {
    switch ft := funcType.(type) {
    case *types.TFunc2:
        for _, paramType := range ft.Params {
            if tvar, ok := paramType.(*types.TVar); ok {  // ❌ Only checks TVar
                tvars = append(tvars, tvar.Name)
            }
            // TVar2 falls through, returns empty string
        }
    }
    return tvars
}

// Result: extractParamTVars returns [] even though type is α2 -> α2 -> α2
// Spent 2 hours debugging why typeSubst map was empty!
```

**After (with helpers):**
```go
// internal/pipeline/specialize.go (FIXED)
func extractParamTVars(funcType types.Type, expectedParams int) []string {
    switch ft := funcType.(type) {
    case *types.TFunc2:
        for _, paramType := range ft.Params {
            if name, ok := types.ExtractTVarName(paramType); ok {  // ✅ Handles both!
                tvars = append(tvars, name)
            }
        }
    }
    return tvars
}

// Result: Works for both TVar and TVar2, no silent failures
```

### Example 2: Incomplete Switch Statement (Before M-DX8)

**Before:**
```go
// internal/pipeline/specialize.go (BUGGY)
func (s *Specializer) cloneExpr(expr core.CoreExpr, typeSubst map[string]types.Type) (core.CoreExpr, error) {
    switch e := expr.(type) {
    case *core.Var:
        // ... handle Var
    case *core.Lambda:
        // ... handle Lambda
    // ❌ Missing case for *core.Let!

    default:
        return expr, nil  // ❌ Silent fallback - Let nodes returned unchanged!
    }
}

// Result: Cloning stops at Let boundaries, operators never specialized
// Spent 1.5 hours discovering this with DEBUG_MONO_VERBOSE logging
```

**After (with DEBUG_STRICT):**
```go
// internal/pipeline/specialize.go (FIXED)
func (s *Specializer) cloneExpr(expr core.CoreExpr, typeSubst map[string]types.Type) (core.CoreExpr, error) {
    switch e := expr.(type) {
    case *core.Var:
        // ... handle Var
    case *core.Lambda:
        // ... handle Lambda
    case *core.Let:  // ✅ Now handled!
        // ... clone Let

    default:
        if os.Getenv("DEBUG_STRICT") != "" {
            panic(fmt.Sprintf("cloneExpr: unhandled type %T", expr))  // ✅ Loud failure!
        }
        return expr, nil
    }
}

// Running with DEBUG_STRICT=1:
// $ DEBUG_STRICT=1 ailang run test.ail
// panic: cloneExpr: unhandled type *core.Let (NodeID 42)
//   Add a case for this type or explicitly mark as unsupported.
//
// Result: Bug discovered immediately, not after 1.5 hours
```

### Example 3: AST Coverage Test (New)

**Test that prevents Example 2:**
```go
// internal/pipeline/specialize_coverage_test.go (NEW)
func TestCloneExprCoversAllNodeTypes(t *testing.T) {
    allNodeTypes := core.AllNodeTypes()  // [*Var, *Let, *Lambda, *If, ...]

    s := NewSpecializer(&types.CoreTypeInfo{})
    typeSubst := make(map[string]types.Type)

    for _, nodeType := range allNodeTypes {
        node := core.NewMinimalNode(nodeType)
        cloned, err := s.cloneExpr(node, typeSubst)

        require.NoError(t, err, "cloneExpr failed for %T", node)
        require.NotEqual(t, node.ID(), cloned.ID(),
            "cloneExpr returned unchanged node for %T - missing case?", node)
    }
}

// If we forget to add Let case:
// $ make test
// --- FAIL: TestCloneExprCoversAllNodeTypes (0.00s)
//     specialize_coverage_test.go:42:
//         cloneExpr returned unchanged node for *core.Let - missing case?
//
// Result: Test fails at commit time, not at production time
```

## Success Criteria

- [ ] `ExtractTVarName` helper function available and tested
- [ ] `DEBUG_STRICT` mode implemented in 4+ key functions
- [ ] All TVar pattern matches also check TVar2 (or use helper)
- [ ] AST coverage tests pass for `cloneExpr` and `specializeExpr`
- [ ] Migration guide written and reviewed
- [ ] Zero silent failures in M-POLY-B test cases when run with `DEBUG_STRICT=1`
- [ ] All tests passing (including new coverage tests)
- [ ] Documentation updated (CLAUDE.md, migration guide)

## Testing Strategy

**Unit tests:**
- `TestExtractTVarName` - helper works for TVar and TVar2
- `TestIsTVar` - correctly identifies type variables
- `TestCloneExprCoversAllNodeTypes` - no missing cases
- `TestSpecializeExprCoversAllNodeTypes` - no missing cases
- `TestDebugStrictModeCloneExpr` - panics on unhandled types
- `TestDebugStrictModeSpecializeExpr` - panics on unhandled types

**Integration tests:**
- Run M-POLY-B test cases with `DEBUG_STRICT=1` - should pass
- Run with intentionally broken code (missing case) - should panic
- Run full test suite with `DEBUG_STRICT=1` - should pass

**Manual testing:**
- Remove a case from `cloneExpr`, run with `DEBUG_STRICT=1` - should panic
- Use TVar where TVar2 expected with helper - should work
- Use TVar where TVar2 expected without helper - should fail loudly in debug

**Regression prevention:**
- Add M-DX8 coverage tests to CI pipeline
- Require `DEBUG_STRICT=1` tests to pass before merge

## Non-Goals

**Not in this feature:**
- **Complete TVar removal** - Only migration to TVar2 (removal in v0.5.0+)
- **Exhaustive switch checks at compile time** - Requires language changes (optional linter only)
- **Silent failure prevention in eval** - Only compiler phases (eval is separate milestone)
- **Automated migration tool** - Manual migration with helpers is sufficient

**Why deferred:**
- Complete TVar removal requires broad coordination (v0.5.0+ roadmap item)
- Compile-time exhaustiveness checking requires Go language features we don't have
- Eval phase has different failure modes (runtime vs compile-time)
- Automated migration adds complexity without enough benefit (manual is feasible)

## Timeline

**Week 1** (~8 hours):
- Days 1-2: Phase 1 (Quick Wins) - Helpers, DEBUG_STRICT, documentation

**Week 2** (~8 hours):
- Days 1-2: Phase 2 (AST Coverage Tests) - Reflection, minimal nodes, tests

**Week 3** (~8-16 hours):
- Days 1-3: Phase 3 (Systematic Migration) - TVar → TVar2 across codebase
- Day 4 (optional): Phase 4 (Linter Rules) - Custom golangci-lint checks

**Total: ~24-32 hours across 3 weeks (3-4 days of work)**

**Buffer:** 50% applied (16h → 24h)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| TVar2 migration breaks existing code | High | Systematic migration with test suite validation at each step |
| DEBUG_STRICT too noisy | Medium | Only enable in CI and local dev, not production |
| AST coverage tests miss edge cases | Medium | Start with happy path, iterate based on real bugs |
| Linter rules reject valid code | Low | Make linter warnings, not errors; provide escape hatch |
| Migration takes longer than estimated | Medium | Phase 3 is optional for v0.4.1 - can defer to v0.4.2 |

## References

- **Trigger**: M-POLY-B Phase 1 implementation
  - [M-POLY-B Complete Report](../../../M-POLY-B-PHASE1-COMPLETE.md)
  - Bug #3: TVar2 not handled in extractParamTVars
  - Bug #4: TVar2 not handled in substituteType
  - Bug #5: Let not handled in cloneExpr
- **Related design docs**:
  - [M-DX6: Pipeline Visualization](../v0_3_18/m-dx6-pipeline-visualization.md) - Debugging guides
  - [M-DX4: CoreTypeInfo Validation](../v0_3_18/m-dx4-coreti-population-gaps.md) - Similar validation pattern
- **Existing assessments**:
  - [AILANG Ease of Use Assessment](../v0_3_18/AILANG_EASE_OF_USE_ASSESSMENT.md) - Mentions TVar/TVar2 confusion
- **Prior art**:
  - Rust's `#[non_exhaustive]` attribute
  - Haskell's `-fwarn-incomplete-patterns`
  - OCaml's exhaustiveness checking

## Future Work (v0.5.0+)

**Beyond v0.4.1:**

1. **Complete TVar removal** (v0.5.0)
   - After migration complete, remove TVar entirely
   - Single type variable representation (TVar2 only)
   - Update all documentation

2. **Compile-time exhaustiveness checking** (v0.5.0+)
   - Explore static analysis tools for Go
   - Generate exhaustiveness proofs per function
   - Reject code with incomplete switches at build time

3. **Silent failure audit** (v0.4.2)
   - Audit eval phase for silent failures
   - Audit effects system for silent failures
   - Apply same pattern (helpers, DEBUG_STRICT, tests)

4. **Automated migration tool** (v0.5.0+)
   - AST rewriter to convert TVar → TVar2
   - Useful if other dual-representation issues arise
   - Low priority (manual migration worked well)

---

**Document created**: 2025-10-23
**Last updated**: 2025-10-23

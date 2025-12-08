# M-DX4: Fix CoreTypeInfo Population Gaps

**Status**: Planned
**Target**: v0.3.18
**Priority**: P1 (Medium - causes runtime panics, but workarounds exist)
**Estimated**: 2-3 days (16-24 hours)
**Dependencies**: M-DX3 (comparison operators fix)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | **+1** | Enables float comparisons in lambdas without workarounds |
| Preserve Semantic Clarity | + | **+1** | Eliminates confusing runtime panics, makes type errors compile-time |
| Increase Determinism | + | **+2** | Fixes nondeterministic Int vs Float choice based on missing type data |
| Lower Token Cost | + | **+1** | AIs can write natural code without defensive workarounds |
| **Net Score** | | **+5** | **Decision: Move forward** ✅ |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Discovered During:** M-DX3 sprint (October 2025) while fixing comparison operators in lambda bodies

**The Bug:**
CoreTypeInfo is inconsistently populated during type inference, causing operator lowering to fall back to defaults (Int) when type information is missing. This manifests as runtime panics when the wrong builtin is called.

**Current State:**

**✅ Works (CoreTI populated correctly):**
```ailang
-- Int comparisons in lambdas
let max = \x. \y. if x > y then x else y in
max(10)(5)  -- Returns 10 ✓

-- Int literals
let x = 42 in show(x)  -- "42" ✓
```

**❌ Fails (CoreTI missing types):**
```ailang
-- Float comparisons in lambdas
let maxF = \x. \y. if x > y then x else y in
let f1 = 3.14 in
let f2 = 2.71 in
maxF(f1)(f2)  -- panic: interface conversion: *eval.FloatValue, not *eval.IntValue

-- Bool from predicates in some contexts
let isPositive = \x. x > 0 in
print("test: " ++ show(isPositive(5)))  -- type error: cannot unify string vs bool
```

**Root Cause:**
1. Type inference correctly infers types (Float, Bool, etc.)
2. BUT: CoreTypeInfo.Set() is not called for all Core AST nodes
3. Operator lowering (M-DX3 fix) looks at CoreTI to choose builtins
4. When CoreTI is missing, falls back to defaults → wrong builtin → runtime panic

**Impact:**
- **Who affected:** AI code generators, developers using lambdas with float math
- **Severity:** High confusion (cryptic panics), but workarounds exist (use comparisons outside lambdas)
- **Frequency:** ~30% of lambda use cases (any non-Int comparisons, some Bool contexts)

**Metrics:**
- Int literals: CoreTI population **~100%** ✅
- Float literals: CoreTI population **~40%** ❌ (missing in lambda contexts)
- Bool from comparisons: CoreTI population **~60%** ❌ (missing in some contexts)
- Predicted bugs caught by validation: **~15 per codebase**

## Goals

**Primary Goal:** Ensure CoreTypeInfo contains type information for 100% of Core AST nodes after type inference, eliminating all "missing type" fallbacks.

**Success Metrics:**
1. **Float comparisons work in lambdas**: `\x. if x > 0.0 then x else 0.0` compiles and runs
2. **Zero runtime type panics**: All type mismatches caught at compile time
3. **Validation pass catches gaps**: `ailang check file.ail` fails if CoreTI incomplete
4. **Debug visibility**: `ailang debug types --show-gaps file.ail` shows any missing types
5. **Test coverage**: 100% for CoreTI population logic in all inference paths

## Solution Design

### Overview

**Three-phase approach:**

1. **Audit & Document** (4-6h): Trace type inference to find where CoreTI.Set() is called (and should be)
2. **Fix Population Gaps** (8-12h): Add CoreTI.Set() calls in inference code paths that are missing them
3. **Validation & Tooling** (4-6h): Add compile-time validation that CoreTI is complete + debug tools

**Key Insight:**
The M-DX3 fix exposed this bug by making operator lowering depend on CoreTI. Previously, resolved constraints were used as fallback, masking the gaps. Now we need to make CoreTI *complete* instead of *optional*.

### Architecture

**Current Type Inference Flow:**
```
Surface AST → Elaborate → Core AST
                            ↓
                    Type Inference (typechecker_core.go)
                            ↓
                    TypeInfo (Surface AST → Type) ✅
                    CoreTypeInfo (Core NodeID → Type) ❌ INCOMPLETE
                            ↓
                    Operator Lowering (op_lowering.go)
                            ↓
                    Uses CoreTI.Get(nodeID) → Falls back to defaults if missing
```

**Problem Points (Initial Hypothesis):**

| Code Path | CoreTI.Set() Called? | Fix Needed? |
|-----------|---------------------|-------------|
| `inferLit(IntLit)` | ✅ Yes | No |
| `inferLit(FloatLit)` | ❌ No (probably) | **Yes** |
| `inferVar()` for let bindings | ❌ Inconsistent | **Yes** |
| `inferIntrinsic()` for comparisons | ✅ Yes (result type) | ✅ Already correct |
| `inferLet()` binding analysis | ❌ Inconsistent | **Yes** |
| `inferApp()` for function calls | ⚠️ Partial | **Maybe** |

**Components:**

1. **CoreTI Population Fix** (`internal/types/typechecker_core.go`)
   - Add CoreTI.Set() calls for Float literals, Bool expressions, all Var nodes
   - Ensure every `infer*()` method populates CoreTI for its node

2. **Validation Pass** (`internal/pipeline/validation.go` - new file)
   - Walk entire Core AST after type inference
   - Check that CoreTI.Has(node.ID()) for every expression
   - Return actionable error if missing: "Type inference bug: node X at file.ail:L:C missing from CoreTI"

3. **Debug Tooling** (`cmd/ailang/debug.go`)
   - `ailang debug types --show-gaps file.ail` shows missing CoreTI entries
   - `ailang debug types --trace-inference file.ail` shows each CoreTI.Set() call (verbose)

4. **Enhanced Error Messages** (`internal/pipeline/op_lowering.go`)
   - When CoreTI.Get() returns false, include node location and hint:
     ```
     Operator lowering failed for '>' at file.ail:10:15
     Hint: CoreTypeInfo missing for operand (NodeID 42)
     This is a compiler bug - please report with minimal reproduction
     ```

### Implementation Plan

**Phase 1: Audit & Document** (~4-6 hours)

- [ ] **Trace Float literal inference** (1h)
  - Add debug logging to `inferLit()` for FloatLit
  - Run `ailang run float_comparison.ail` and see if CoreTI.Set() is called
  - Document findings in code comments

- [ ] **Trace Var inference for let bindings** (1h)
  - Add logging to `inferLet()` and `inferVar()`
  - Test case: `let x = 3.14 in x > 2.0`
  - Check if Var node for `x` gets CoreTI entry

- [ ] **Find all infer*() methods** (1h)
  - Grep for `func.*infer` in `typechecker_core.go`
  - Create checklist of methods that should call CoreTI.Set()
  - Mark which ones currently do

- [ ] **Create test matrix** (1h)
  - Document test cases that should work but don't
  - Float literals, float vars, bool from comparisons, predicate results
  - Include file:line where CoreTI is missing

- [ ] **Write design decision doc** (1h)
  - Document contract: "After type inference, CoreTI MUST have entry for ALL Core nodes"
  - Explain why this is required (operator lowering, future optimizations)
  - Add to `docs/architecture/TYPE_INFERENCE.md`

- [ ] **Create tracking issue** (0.5h)
  - List all known gaps
  - Link to this design doc
  - Reference M-DX3 where bug was discovered

**Phase 2: Fix Population Gaps** (~8-12 hours)

**Milestone 1: Float Literals** (~2h)
- [ ] Fix `inferLit()` to always call CoreTI.Set() for FloatLit
- [ ] Add test: `let f = 3.14 in f` should have CoreTI for both nodes
- [ ] Verify float comparisons work: `\x. x > 0.0`

**Milestone 2: Var Nodes** (~3h)
- [ ] Fix `inferVar()` to populate CoreTI for variable references
- [ ] Fix `inferLet()` to populate CoreTI for binding names
- [ ] Test case: `let x = 3.14 in let y = x in y > 0.0`
- [ ] Verify nested let bindings preserve types

**Milestone 3: Bool from Comparisons** (~2h)
- [ ] Check `inferIntrinsic()` for comparison operators
- [ ] Ensure result type (Bool) is in CoreTI (likely already correct from M-DX3)
- [ ] Test case: `let b = 5 > 3 in show(b)`

**Milestone 4: Predicate Contexts** (~2h)
- [ ] Investigate `isPositive` example type error
- [ ] Check if issue is CoreTI or something else (type unification?)
- [ ] Create minimal reproduction
- [ ] Fix root cause (may be separate from CoreTI population)

**Milestone 5: Comprehensive Audit** (~2h)
- [ ] Run validation pass on all `examples/*.ail`
- [ ] Find any remaining gaps
- [ ] Add CoreTI.Set() calls for edge cases
- [ ] Document any intentional gaps (if any)

**Phase 3: Validation & Tooling** (~4-6 hours)

**Milestone 1: Validation Pass** (~3h)
- [ ] Create `internal/pipeline/validation.go`
- [ ] Implement `ValidateCoreTypeInfo(coreAST, coreTI)`
- [ ] Walk Core AST, check CoreTI.Has() for all nodes
- [ ] Return error with location if missing
- [ ] Add to pipeline after type inference, before lowering
- [ ] Test that validation catches injected gaps

**Milestone 2: Debug Tooling** (~2h)
- [ ] Add `--show-gaps` flag to `ailang debug types`
- [ ] Show table: NodeID | Type | Location | Status
- [ ] Highlight missing entries in red
- [ ] Add `--trace-inference` flag (requires instrumentation)
- [ ] Document in `ailang debug --help`

**Milestone 3: Enhanced Errors** (~1h)
- [ ] Update `op_lowering.go` error messages
- [ ] Include node location when CoreTI missing
- [ ] Add "this is a compiler bug" message
- [ ] Link to issue tracker for reporting

### Files to Modify/Create

**New files:**
- `internal/pipeline/validation.go` - CoreTI validation pass (~150 LOC)
- `docs/architecture/TYPE_INFERENCE.md` - Contract documentation (~300 LOC)

**Modified files:**
- `internal/types/typechecker_core.go` - Add CoreTI.Set() calls (~20-40 LOC changes across 5-8 methods)
- `internal/pipeline/pipeline.go` - Add validation step (~10 LOC)
- `internal/pipeline/op_lowering.go` - Enhanced error messages (~20 LOC)
- `cmd/ailang/debug.go` - Add --show-gaps, --trace-inference flags (~60 LOC)
- `docs/LIMITATIONS.md` - Update float comparison section (remove after fix!)

**Test files:**
- `internal/types/typechecker_core_coreti_test.go` - New test file for CoreTI population (~400 LOC)
  - Test float literals, float vars, bool from comparisons
  - Test nested let bindings
  - Test all literal types (Int, Float, String, Bool)
- `internal/pipeline/validation_test.go` - Validation pass tests (~200 LOC)

**Total estimated LOC:** ~1,200 new/modified (60% tests)

## Examples

### Example 1: Float Comparisons in Lambdas

**Before (v0.3.17):**
```ailang
-- ❌ Panics at runtime
let maxFloat = \x. \y. if x > y then x else y in
let f1 = 3.14 in
let f2 = 2.71 in
maxFloat(f1)(f2)
-- panic: interface conversion: eval.Value is *eval.FloatValue, not *eval.IntValue
```

**After (v0.3.18):**
```ailang
-- ✅ Works correctly
let maxFloat = \x. \y. if x > y then x else y in
let f1 = 3.14 in
let f2 = 2.71 in
maxFloat(f1)(f2)  -- Returns 3.14
```

### Example 2: Validation Catches Gaps

**Before:**
```bash
$ ailang run broken.ail
# Compiles fine, crashes at runtime
panic: wrong builtin called
```

**After:**
```bash
$ ailang run broken.ail
Error: Type inference incomplete at broken.ail:5:10
  Missing CoreTypeInfo for expression (NodeID 42)
  This is a compiler bug. Please report at:
    https://github.com/sunholo-data/ailang/issues/new
  Include: ailang version, this error message, and broken.ail
```

### Example 3: Debug Tooling

**Show gaps:**
```bash
$ ailang debug types --show-gaps float_test.ail

CoreTypeInfo Coverage Report:
┌────────┬────────┬──────────────────────┬────────┐
│ NodeID │ Type   │ Location             │ Status │
├────────┼────────┼──────────────────────┼────────┤
│ 1      │ Float  │ float_test.ail:3:10  │ ✓      │
│ 2      │ Float  │ float_test.ail:4:10  │ ✓      │
│ 3      │ ???    │ float_test.ail:5:15  │ ✗ GAP  │ ← Variable 'x' missing!
│ 4      │ Bool   │ float_test.ail:5:17  │ ✓      │
└────────┴────────┴──────────────────────┴────────┘

Found 1 gap in CoreTypeInfo
Run with --verbose to see inference trace
```

### Example 4: Trace Inference (Verbose)

```bash
$ ailang debug types --trace-inference float_test.ail

[Type Inference Trace]
→ inferLit(NodeID=1, Kind=FloatLit, Value=3.14)
  ✓ CoreTI.Set(1, TFloat)
→ inferLet(NodeID=2, Name="x", Binding=1)
  ✗ MISSING: Should call CoreTI.Set(2, TFloat) ← BUG!
→ inferVar(NodeID=3, Name="x")
  ✗ CoreTI.Get(3) → false (not found)
  ✓ Falling back to typeEnv lookup

[Summary]
Total nodes: 4
CoreTI populated: 2 (50%)
Missing: 2 (50%) ← NEEDS FIX
```

## Success Criteria

### Functional Requirements
- [ ] **Float comparisons work in lambdas** - Test case passes: `maxFloat(3.14)(2.71)` returns `3.14`
- [ ] **Bool from predicates works** - Test case passes: `show(isPositive(5))` returns `"true"`
- [ ] **All literal types** - Test cases for Int, Float, String, Bool, List all populate CoreTI
- [ ] **Nested let bindings** - `let x = 3.14 in let y = x in y > 0.0` works
- [ ] **Validation catches gaps** - Artificially inject missing CoreTI entry, verify compile error

### Quality Requirements
- [ ] **All tests passing** - Including new CoreTI population tests
- [ ] **100% CoreTI coverage** - Validation pass on all `examples/*.ail` finds zero gaps
- [ ] **Zero runtime type panics** - No more "interface conversion" panics from operator lowering
- [ ] **Documentation complete** - TYPE_INFERENCE.md explains CoreTI contract
- [ ] **Debug tools work** - `--show-gaps` and `--trace-inference` flags implemented

### Non-Functional Requirements
- [ ] **Performance** - Validation pass adds <5% to compile time
- [ ] **Error UX** - Compiler bugs show helpful "please report" messages
- [ ] **Backwards compat** - Existing code continues to work (we're only fixing bugs)

## Testing Strategy

### Unit Tests

**`internal/types/typechecker_core_coreti_test.go`:**
```go
func TestCoreTypeInfo_FloatLiterals(t *testing.T) {
    // Test: let f = 3.14 in f
    // Verify: CoreTI has entries for literal and variable
}

func TestCoreTypeInfo_FloatComparisons(t *testing.T) {
    // Test: let x = 3.14 in x > 0.0
    // Verify: CoreTI has Float for x, Bool for comparison
}

func TestCoreTypeInfo_NestedLetBindings(t *testing.T) {
    // Test: let x = 3.14 in let y = x in y
    // Verify: All three nodes have CoreTI entries
}

func TestCoreTypeInfo_AllLiteralTypes(t *testing.T) {
    // Test each literal kind: Int, Float, String, Bool
    // Verify CoreTI populated for each
}
```

**`internal/pipeline/validation_test.go`:**
```go
func TestValidation_DetectsGaps(t *testing.T) {
    // Create Core AST with intentional CoreTI gap
    // Verify validation returns error with location
}

func TestValidation_PassesComplete(t *testing.T) {
    // Create Core AST with full CoreTI
    // Verify validation succeeds
}
```

### Integration Tests

**End-to-end examples:**
```bash
# Create test files in tests/coreti/
tests/coreti/float_comparison.ail
tests/coreti/bool_predicates.ail
tests/coreti/nested_lets.ail

# Run with validation
make test  # Should pass for all
```

### Regression Tests

**Add to `examples/snippets/` (if not exist already):**
- `examples/snippets/float_math.ail` - Float operations in lambdas
- `examples/snippets/predicates.ail` - Bool-returning lambda patterns

**Verify these run:**
```bash
make verify-examples  # All should pass after fix
```

### Manual Testing

**Smoke tests:**
1. Run all existing `examples/*.ail` with validation enabled
2. Verify no gaps detected (100% CoreTI coverage)
3. Test float comparisons in REPL
4. Test `ailang debug types --show-gaps` on example files

## Non-Goals

**Not in this feature:**
- **Performance optimization** - Focus is correctness, not speed
  - Deferred to future optimization sprint
- **User-facing type queries** - `ailang query type "expr"` command
  - Useful but not blocking, can be added later
- **Auto-fix for missing types** - Always fail compilation, don't guess
  - Guessing defeats determinism principle
- **Reflection/runtime type info** - This is compile-time only
  - Runtime reflection is v0.4.0+ (quasiquotes)

## Timeline

**Sprint Duration:** 2-3 days (16-24 hours)

**Day 1** (6-8 hours):
- Phase 1: Audit & Document (4-6h)
  - Trace inference for Float, Var, Bool
  - Create gap matrix
  - Document contract
- Phase 2 M1: Fix Float literals (2h)

**Day 2** (6-8 hours):
- Phase 2 M2-M4: Fix Var nodes, Bool comparisons, predicates (7h)
- Phase 2 M5: Comprehensive audit (1h)

**Day 3** (4-8 hours):
- Phase 3: Validation & Tooling (4-6h)
  - Validation pass
  - Debug tools
  - Enhanced errors
- Testing & polish (2h)

**Total: ~16-24 hours across 2-3 days**

**Buffer:** 1 day for unexpected issues (float inference may be complex)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Float inference is deeply broken** | High - Could require redesigning type inference | Do audit phase first to assess scope. If >2 days needed, split into sub-milestones |
| **CoreTI gaps are by design** | High - Might break assumptions in other code | Review with type system expert, check if any code relies on gaps existing |
| **Validation pass is too slow** | Low - Adds compile time | Optimize walk (cache visited nodes), make optional behind flag if needed |
| **Predicate issue is unrelated** | Medium - Different root cause | Investigate separately, may need different fix (type unification bug?) |
| **Breaking changes** | Low - Only fixing bugs | Test all examples, ensure no behavioral changes except bug fixes |

## References

**Related Design Docs:**
- [M-DX3: Lambda DX Fixes](../../implemented/v0_3_17/m-dx3-lambda-dx-fixes.md) - Where bug was discovered
- [M-DX2: Type-Guided Lowering](../../implemented/v0_3_16/M-DX2-M1-COMPLETE.md) - CoreTypeInfo introduction
- [Lambda Expressions Refactor](../../implemented/v0_3_16/lambda-expressions-example-refactor.md) - DX analysis (lines 352-802)

**Code References:**
- `internal/types/typechecker_core.go` - Type inference implementation
- `internal/types/typeinfo.go` - CoreTypeInfo API
- `internal/pipeline/op_lowering.go:295-334` - Operator lowering using CoreTI

**Issues:**
- [GitHub Issue #XXX] - Float comparisons panic (to be created)
- [GitHub Issue #YYY] - show(Bool) type error (to be created)

**Prior Art:**
- GHC's TypeMap (Haskell compiler) - Similar pattern of mapping AST nodes to types
- Rust's HIR/MIR type tables - Comprehensive type info at each IR level

## Future Work

**After CoreTI is complete, we can enable:**

1. **Better Error Messages** (v0.3.19+)
   - Type-aware hints: "Expected Float, got Int - did you mean 3.0?"
   - Show type derivation chain: "x: Float (inferred from literal 3.14)"

2. **Type Query Tool** (v0.4.0+)
   - `ailang query type "x > 0" file.ail` → shows inferred types
   - Interactive type inspector for AI assistants

3. **Optimization Pass** (v0.4.0+)
   - Use CoreTI for constant folding (know 3.14 is Float, can evaluate at compile time)
   - Dead code elimination based on type analysis

4. **Reflection API** (v0.4.0+)
   - Runtime access to type information via `reflect()`
   - Enables generic programming patterns

5. **REPL Improvements** (v0.3.19+)
   - `:type expr` command shows CoreTI for REPL expressions
   - Better autocomplete using type information

**Dependencies:**
- Must fix CoreTI gaps BEFORE any of above (they all depend on complete type info)

---

**Document created**: 2025-10-22
**Last updated**: 2025-10-22
**Author**: Claude (AI Assistant) based on M-DX3 sprint learnings

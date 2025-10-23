# Sprint Plan: M-DX4 CoreTypeInfo Population Gaps

## Summary

Fix operator lowering for float comparisons in lambdas by ensuring CoreTypeInfo contains complete type information for all operands, eliminating runtime panics when wrong builtins are called.

**Duration:** 1-2 days (8-16 hours)
**Dependencies:** M-DX3 (comparison operators fix) - ✅ COMPLETE
**Risk Level:** Medium (needs careful debugging, but scope is well-defined)

## Current Status Analysis

### What's Already Complete ✅

From v0.3.17 (Oct 22, 2025):
- ✅ **CoreTypeInfo validation infrastructure** (360 LOC implementation + 534 LOC tests)
  - `internal/pipeline/validate_coretypeinfo.go` - Validates 100% CoreTI coverage
  - Integrated in 3 pipeline locations (file, module, REPL)
  - Comprehensive error diagnostics with NodeID, position, hints
  - Performance: O(n) linear, zero allocations (191ns for 10 nodes)

- ✅ **Type-guided operator lowering** (from M-DX3)
  - `internal/pipeline/op_lowering.go` - Uses CoreTI to choose builtins
  - Lines 304-334: CoreTI lookup for operand types
  - Fallback chain: CoreTI → ResolvedConstraints → Default

### The Bug Still Exists ❌

**Reproduction:**
```bash
$ cat > /tmp/test_float.ail << 'EOF'
let maxF = \x. \y. if x > y then x else y in
let f1 = 3.14 in
let f2 = 2.71 in
maxF(f1)(f2)
EOF

$ ailang run --entry main --caps IO /tmp/test_float.ail
panic: interface conversion: eval.Value is *eval.FloatValue, not *eval.IntValue
```

**Root Cause:**
Line 298 in `op_lowering.go`:
```go
typeNode = intrinsic.Args[0].ID()  // Get first operand's NodeID
```

Then line 306:
```go
if inferredType, ok := l.CoreTI.Get(typeNode); ok {
```

**Two distinct issues:**
1. **Operand's type is missing from CoreTI** - The validation pass only checks that the *intrinsic node* has a type, but doesn't verify that *operands* have types
2. **Var-bound polymorphic lambda panic** - When polymorphic lambda is bound to a var, monomorphization doesn't re-elaborate the specialized body (that's M-POLY-B)

**This sprint fixes issue #1 only.** Issue #2 (var-bound polymorphic) is out of scope - it belongs to M-POLY-B (operator re-linking after specialization).

### Recent Velocity

From CHANGELOG analysis (last 14 days):
- **M-DX4 validation (v0.3.17)**: 360 LOC + 534 test LOC = 894 total (~3 hours actual)
- **M-POLY-A (v0.3.17)**: 1130 LOC + 790 test LOC = 1920 total (~4-5 days estimate, likely 2-3 actual)
- **Average**: ~200-400 LOC/day for implementation + tests
- **Test ratio**: 1:1 to 1.5:1 (test-heavy, good for compiler work)

**Velocity estimate for this sprint:**
- Implementation: ~150-250 LOC (focused bug fix in typechecker)
- Tests: ~200-300 LOC (comprehensive operand type coverage)
- **Total**: ~350-550 LOC
- **Time**: 1-2 days at current velocity

### Remaining from Design Doc

The original design doc had 3 phases:
- ✅ Phase 1: Audit & Document (~4-6h) - PARTIALLY DONE (we know the issue now)
- ❌ Phase 2: Fix Population Gaps (~8-12h) - **THIS SPRINT**
- ✅ Phase 3: Validation & Tooling (~4-6h) - COMPLETE in v0.3.17

**What's left:**
1. Fix CoreTI.Set() calls for operands (not just result types)
2. Verify all `infer*()` methods populate operand types (including TVars for polymorphic code)
3. Add tests for float/bool operands in various contexts
4. Update validation to check operand types too (accepting TVars as valid)
5. Add fallback telemetry to track when CoreTI lookups miss
6. Document CoreTI contract: "total for all nodes, TVars allowed pre-mono, concrete heads required for lowering"

## Proposed Milestones

### Milestone 1: Audit Operand Type Population + Add Telemetry
**Goal:** Understand exactly where operand types are missing from CoreTI AND add telemetry to track fallbacks
**Estimated:** 150 LOC debug code + telemetry + logging = 150 LOC
**Duration:** 3-4 hours (half day)

**Tasks:**
- **Hour 1: Add fallback telemetry**
  - [ ] Add debug counter in `op_lowering.go` (gated by `--debug-compile` flag)
    - Track: CoreTI hit, ResolvedConstraints fallback, default fallback
    - Track: operator type, NodeID, source location
  - [ ] Emit summary at end of compilation: "CoreTI misses: 3 (OpGt at line 5, OpLe at line 7)"
  - [ ] Test: Run existing tests with `--debug-compile`, count current fallbacks

- **Hour 2: Add debug logging**
  - [ ] Add logging to `lowerIntrinsic()` in `op_lowering.go`
    - Log: `typeNode = %d, CoreTI.Get() = %v, fallback = %s` for each operand
  - [ ] Run test_float.ail with debug logging enabled
  - [ ] Identify which NodeIDs are missing from CoreTI vs which have TVars
  - [ ] Check if those nodes are in ResolvedConstraints as fallback

- **Hour 3: Trace type inference**
  - [ ] Add logging to `inferApp()` in `typechecker_core.go`
    - When lambda is applied, does it populate arg types in CoreTI? (even TVars)
  - [ ] Add logging to `inferLam()` in `typechecker_core.go`
    - When lambda body is inferred, are param types (including TVars) in CoreTI?
  - [ ] Document findings: distinguish missing entries from TVar entries

- **Hour 4: Create test matrix + document contract**
  - [ ] List all failing cases: float lit, float var, bool from comparison
  - [ ] Document CoreTI contract in code comments:
    - "CoreTI is total for all nodes"
    - "Types may be TVars prior to specialization"
    - "Lowering of overloaded operators requires non-TVar heads"
    - "If absent, that is a compiler bug or missing specialization"
  - [ ] Create minimal reproduction for each case
  - [ ] Write hypothesis: "Operands not in CoreTI" vs "Operands have TVars but lowering needs concrete"

**Acceptance Criteria:**
- [ ] Clear documentation of which `infer*()` methods are missing CoreTI.Set() calls
- [ ] Minimal reproduction test cases for each missing type
- [ ] Debug output showing NodeID gaps vs TVar entries in CoreTI
- [ ] Telemetry tracks all CoreTI misses during lowering (with location info)
- [ ] CoreTI contract documented in `internal/types/typechecker_core.go` header
- [ ] Hypothesis distinguishes "missing" from "polymorphic (TVar)"

**Risks:**
- Gaps might be in multiple places (not just lambda params) - Mitigation: Systematic audit of all infer methods
- TVar entries might be incorrectly treated as "missing" - Mitigation: Distinguish presence from concreteness
- Telemetry might have false positives - Mitigation: Check if fallback to TVar-aware ResolvedConstraints is intentional

---

### Milestone 2: Fix CoreTI Population in Type Inference
**Goal:** Ensure all operand types are populated in CoreTI during type inference (including TVars for polymorphic code)
**Estimated:** 150 LOC fixes + 250 LOC tests = 400 LOC
**Duration:** 5-6 hours (half to full day)

**Tasks:**
- **Hour 1-2: Fix inferApp() for lambda applications**
  - [ ] When applying `(\x. body) arg`, ensure:
    - CoreTI.Set(arg.ID(), argType) ← Use POST-substitution type (after unification)
    - CoreTI.Set(x.ID(), argType) ← Param gets type from arg (reuse ApplySubstitution)
  - [ ] For polymorphic applications, ensure TVar is set (not missing)
  - [ ] Test: `(\x. x > 0.0)(3.14)` should populate CoreTI for `3.14` and `x` with Float
  - [ ] Test: `(\x. x)(42)` should populate CoreTI with TVar (pre-mono) or Int (post-mono)
  - [ ] Verify existing tests still pass

- **Hour 2-3: Fix inferLam() for lambda parameters**
  - [ ] When inferring `\x. body`, ensure:
    - CoreTI.Set(x.ID(), paramType) where paramType is from annotation or TVar
  - [ ] Use TVar for unannotated polymorphic lambdas (not missing!)
  - [ ] Test: `let f = \x. x + 1 in f(42)` populates CoreTI for `x` (TVar → Int after application)

- **Hour 3-4: Fix inferVar() for variable references**
  - [ ] When inferring `Var(name)`, ensure:
    - CoreTI.Set(var.ID(), lookupType(name))
  - [ ] Include both the var node AND its binding (if ANF creates intermediate)
  - [ ] Test: `let x = 3.14 in x > 0.0` populates CoreTI for both `x` references

- **Hour 4-5: Fix inferLit() for all literal types**
  - [ ] Verify FloatLit calls CoreTI.Set() (hypothesis: this is missing!)
  - [ ] Check BoolLit, StringLit, IntLit all populate CoreTI
  - [ ] Test: `3.14 > 2.71` populates CoreTI for both float literals
  - [ ] Test: ANF-transformed literals (if bound to vars) have CoreTI entries

- **Hour 5-6: Add golden tests for TVar contract**
  - [ ] **Golden A (TVar allowed)**: `\x. x > x` → CoreTI has TVar on operands; passes pre-mono
  - [ ] **Golden B (concrete required)**: Same lambda after specialization → requires Float/Int heads
    - Mark as `skip` for now (will unskip in M-POLY-B when re-elab exists)
    - Add test helper to simulate specialization by substituting types
  - [ ] Run all existing tests with CoreTI validation enabled
  - [ ] Document any intentional gaps (should be zero)

**Acceptance Criteria:**
- [ ] `test_float.ail` compiles without CoreTI gaps (may still panic at runtime - that's M-POLY-B)
- [ ] All literal types (Int, Float, Bool, String) have CoreTI entries
- [ ] All variable references have CoreTI entries (including both binding and usage sites)
- [ ] Lambda parameters have CoreTI entries (TVar for polymorphic, concrete after specialization)
- [ ] Golden tests pin TVar contract (allowed pre-mono, concrete post-mono)
- [ ] All existing tests pass
- [ ] New unit tests for each fix (6+ test cases covering TVar and concrete paths)
- [ ] Telemetry shows zero CoreTI misses for non-polymorphic code

**Risks:**
- Fixing one inference path might break another - Mitigation: Run full test suite after each change
- TVar vs concrete distinction might be subtle - Mitigation: Golden tests with explicit pre/post-mono separation
- ApplySubstitution might not be idempotent - Mitigation: Verify substitution is applied exactly once per node

---

### Milestone 3: Enhanced Validation & Testing (TVar-Aware)
**Goal:** Strengthen CoreTI validation to catch operand type gaps while accepting TVars as valid
**Estimated:** 120 LOC validation + 250 LOC tests = 370 LOC
**Duration:** 3-4 hours (half day)

**Tasks:**
- **Hour 1-2: Enhance validation for operands (TVar-aware)**
  - [ ] Modify `ValidateCoreTypeInfo` to check:
    - For each `Intrinsic` node, verify ALL args have CoreTI entries (TVars OK pre-mono)
    - For each `App` node, verify function and args have CoreTI entries (TVars OK)
    - For each `BinOp`/`UnOp`, verify operands have CoreTI entries (TVars OK)
  - [ ] Add phase-aware validation:
    - Pre-mono: Accept TVar as valid (check presence, not concreteness)
    - Post-mono (future): Add `--strict-lowering` flag to require concrete heads for operators
  - [ ] Add specific error messages with operator, arg index, parent NodeID:
    - "Missing operand type for Intrinsic(OpGt) arg 0 (parent NodeID 1234, line 5)"
    - Include hint: "This is a compiler bug - operands should be typed before lowering"
  - [ ] Test that validation catches injected operand gaps (missing entries)
  - [ ] Test that validation accepts TVars (polymorphic code is valid pre-mono)

- **Hour 2-3: Add comprehensive test suite**
  - [ ] Test file: `internal/types/typechecker_core_operands_test.go`
  - [ ] Test cases (non-polymorphic):
    - Float literals in comparisons: `3.14 > 2.71`
    - Float vars in comparisons: `let x = 3.14 in x > 0.0`
    - Direct lambda app with float: `(\x. x > 0.0)(3.14)`
    - Bool from predicates: `let b = 5 > 3 in show(b)`
    - Nested let bindings: `let x = 3.14 in let y = x in y > 0.0`
  - [ ] Test cases (polymorphic - TVar expected):
    - Unapplied polymorphic lambda: `\x. x > x` (TVar operands OK)
    - Var-bound polymorphic lambda: `let f = \x. x > x in ...` (TVar OK pre-mono)
  - [ ] All tests verify: CoreTI.Has(operand.ID()) == true (presence check)
  - [ ] Subset of tests verify: Head(CoreTI.Get(operand.ID())) is concrete (when expected)

- **Hour 3-4: Integration testing + nice-to-have tooling**
  - [ ] Add examples to `examples/` directory:
    - `examples/snippets/float_comparison.ail`
    - `examples/snippets/bool_predicates.ail`
    - `examples/snippets/polymorphic_comparison.ail` (var-bound, will panic at runtime until M-POLY-B)
  - [ ] Run `make verify-examples` to ensure they compile (no CoreTI gaps)
  - [ ] Update example verification status
  - [ ] **Nice-to-have**: Add `ailang debug types --show-gaps` mode
    - Highlight operands missing in CoreTI with asterisk in printout
    - Format: `OpGt(arg0: Float, arg1: *MISSING*)` vs `OpGt(arg0: Float, arg1: Float)`

**Acceptance Criteria:**
- [ ] Validation catches missing operand entries (presence check)
- [ ] Validation accepts TVar entries as valid (pre-mono polymorphic code OK)
- [ ] Error messages include operator, arg index, parent NodeID, source location
- [ ] 100% test coverage for operand type population (both concrete and TVar paths)
- [ ] All examples in `examples/snippets/` compile without CoreTI gaps
- [ ] Polymorphic examples compile (may panic at runtime - that's M-POLY-B scope)
- [ ] Clear error messages guide users to report compiler bugs
- [ ] Telemetry confirms zero unexpected defaulting in test suite

**Risks:**
- Validation might be too strict (rejects valid TVar entries) - Mitigation: Check presence, not concreteness
- Phase detection (pre/post mono) might be ambiguous - Mitigation: Use flag `--strict-lowering` (default off for now)
- Tests might not cover all edge cases - Mitigation: Golden tests with explicit TVar vs concrete expectations

---

## Success Metrics

### Functional Requirements
- [ ] Direct float comparisons compile: `3.14 > 2.71` (no CoreTI gaps)
- [ ] Float vars in comparisons compile: `let x = 3.14 in x > 0.0` (no CoreTI gaps)
- [ ] Direct lambda applications compile: `(\x. x > 0.0)(3.14)` (no CoreTI gaps)
- [ ] Var-bound lambdas compile: `let f = \x. x > 0.0 in f(3.14)` (no CoreTI gaps, may panic at runtime - M-POLY-B)
- [ ] Bool from predicates works: `let b = 5 > 3 in show(b)` compiles (no CoreTI gaps)
- [ ] All literal types populate CoreTI: Int, Float, String, Bool, List
- [ ] Nested let bindings preserve types: `let x = 3.14 in let y = x in y > 0.0`
- [ ] Validation catches gaps: Inject missing CoreTI entry → compile error with NodeID/location
- [ ] Polymorphic code accepted: `\x. x > x` compiles with TVar operands (validation passes)

### Quality Requirements
- [ ] All tests passing (including new operand type tests)
- [ ] 100% CoreTI presence for operands (validation on `examples/*.ail` and test suite)
- [ ] Zero unexpected defaulting (telemetry confirms CoreTI hits, not fallbacks)
- [ ] Test coverage: >95% for new code
- [ ] Lint clean: `make lint` passes
- [ ] Golden tests pin TVar contract (pre-mono accepts TVars, post-mono requires concrete)

### Performance Requirements
- [ ] Validation adds <5% to compile time
- [ ] No performance regression in type inference
- [ ] CoreTI memory overhead acceptable (<10% increase)
- [ ] Telemetry overhead negligible when `--debug-compile` disabled

### Documentation Requirements
- [ ] CoreTI contract documented in `internal/types/typechecker_core.go` header:
  - "CoreTI is total for all nodes"
  - "Types may be TVars prior to specialization"
  - "Lowering of overloaded operators requires non-TVar heads"
  - "If absent, that is a compiler bug or missing specialization"
- [ ] Update `docs/LIMITATIONS.md`:
  - Note: Direct float comparisons now work
  - Note: Var-bound polymorphic lambdas still panic (M-POLY-B will fix)
- [ ] Update CHANGELOG.md with M-DX4 completion details
- [ ] Code comments in `op_lowering.go` explain fallback chain and telemetry
- [ ] Example files demonstrate working float comparisons (non-polymorphic)

## Timeline

### Day 1 (7-9 hours)
**Morning (3-4h): Milestone 1 - Audit + Telemetry**
- Add fallback telemetry to op_lowering (gated by --debug-compile)
- Add debug logging to op_lowering and typechecker
- Run failing test cases, identify missing NodeIDs vs TVar entries
- Document CoreTI contract in code comments
- Create test matrix distinguishing "missing" from "polymorphic"

**Afternoon (4-5h): Start Milestone 2**
- Fix inferApp() for lambda applications (use post-substitution types)
- Fix inferLam() for lambda parameters (TVar for polymorphic)
- Add golden tests for TVar contract (Golden A passing, Golden B skipped)
- Test: Direct float comparisons compile without CoreTI gaps

### Day 2 (4-7 hours)
**Morning (2-3h): Finish Milestone 2**
- Fix inferVar() for variable references (including binding + usage sites)
- Fix inferLit() for float literals (including ANF intermediates)
- Comprehensive audit of all infer methods
- Verify telemetry shows zero unexpected defaulting
- All existing tests pass

**Afternoon (2-4h): Milestone 3**
- Enhance validation for operand types (TVar-aware: check presence, not concreteness)
- Add comprehensive test suite (concrete + polymorphic paths)
- Integration testing with examples (including polymorphic.ail that compiles but may panic)
- Optional: Add `ailang debug types --show-gaps` mode
- Polish and documentation

**Total: 11-16 hours across 1.5-2 days**

## Dependencies

### Completed Dependencies
- ✅ M-DX3: Comparison operators (v0.3.17) - Operator lowering infrastructure exists
- ✅ M-DX4 Phase 3: Validation (v0.3.17) - Validation framework ready to extend

### External Dependencies
- None - self-contained bug fix

### Internal Dependencies
- Access to `internal/types/typechecker_core.go` (462 LOC - well-sized ✅)
- Access to `internal/pipeline/op_lowering.go` (370 LOC - well-sized ✅)
- Test infrastructure in `internal/types/*_test.go`

## M-POLY-B Boundary (Critical: Out of Scope!)

**This sprint DOES NOT fix var-bound polymorphic lambda panics.** That's M-POLY-B.

### What This Sprint Fixes ✅
- Direct float comparisons: `3.14 > 2.71` ← Will work!
- Float vars: `let x = 3.14 in x > 0.0` ← Will work!
- Direct lambda apps: `(\x. x > 0.0)(3.14)` ← Will work!

### What This Sprint DOES NOT Fix ❌ (M-POLY-B)
- Var-bound polymorphic: `let maxF = \x. \y. if x > y then x else y in maxF(3.14)(2.71)` ← Still panics!
- Reason: Monomorphization specializes the lambda but doesn't re-elaborate the body to fix operator intrinsics
- Solution (M-POLY-B): After specialization, re-elaborate the specialized body to re-link operators with concrete types

### Why the Boundary Matters
- **This sprint**: Ensures all operands have CoreTI entries (including TVars for polymorphic code)
- **M-POLY-B**: Ensures specialized bodies get re-elaborated so operators use concrete builtins

### Test Strategy for Boundary
- Add `examples/snippets/polymorphic_comparison.ail` (var-bound lambda)
- It should **compile** without CoreTI gaps (this sprint's success)
- It will still **panic at runtime** (expected until M-POLY-B)
- Add comment in example: "// KNOWN ISSUE: Runtime panic until M-POLY-B (operator re-linking)"

### Acceptance Criteria Adjustment
- This sprint is successful if CoreTI gaps are eliminated
- Runtime panics for var-bound polymorphic are expected and documented
- Do not attempt to fix M-POLY-B issues during this sprint (scope creep risk!)

## Open Questions

### For User Input
1. **Scope**: Should we fix ONLY float comparisons, or all operand types?
   - Recommendation: Fix all operand types (Float, Bool, String) for completeness
   - Rationale: Same root cause, same fix, minimal extra effort

2. **Validation strictness**: Should we fail compilation on missing operand types?
   - Recommendation: Yes, fail fast with clear error (but accept TVars as valid)
   - Rationale: Better than silent fallback to wrong default, catches compiler bugs early

3. **Backwards compatibility**: Any concerns about breaking existing code?
   - Analysis: Only fixes bugs, no behavioral changes for working code
   - Risk: Very low (validation only catches compiler bugs, not user errors)

### Technical Questions (to resolve during implementation)
1. **Polymorphic lambda params**: Should they have TVar in CoreTI or concrete type?
   - Hypothesis: TVar during inference, specialized to concrete after application
   - Need to verify: Does monomorphization update CoreTI?

2. **Resolved constraints fallback**: Keep it or remove it?
   - Current: Line 336-343 in op_lowering.go falls back to ResolvedConstraints
   - Recommendation: Keep as safety net, but log warning if CoreTI missing
   - Rationale: Defense in depth, easier debugging

3. **Package size**: `internal/types` is 11,858 LOC total across ~30 files
   - Individual files are well-sized (largest is inference.go at 780 LOC)
   - `typechecker_core.go` is only 462 LOC (well under 800 line limit ✅)
   - No refactoring needed - package is already well-organized

## Notes

### Assumptions
- The validation framework (v0.3.17) works correctly for intrinsic result types
- The issue is specifically operand types, not result types
- ResolvedConstraints are populated correctly (fallback works)
- CoreTI.Set() is cheap (no performance concerns about extra calls)

### Lessons from v0.3.17
- Validation was faster than expected (~3h vs 4-6h estimate)
- Why? Because ApplySubstitution() already fixed most gaps
- **Key insight**: May find that some fixes already exist, just need wiring

### Code Organization
- `typechecker_core.go` is only 462 LOC ✅ (well under 800 line limit)
- The entire `internal/types` package is 11,858 LOC across ~30 well-organized files
- Largest file is `inference.go` at 780 LOC (still acceptable)
- No structural refactoring needed - code is already maintainable
- This sprint: ~50 LOC of targeted edits (mostly CoreTI.Set() calls)

### Testing Strategy
- Unit tests for each `infer*()` method fix
- Integration tests for end-to-end scenarios
- Regression tests to ensure existing code still works
- Performance benchmarks to check overhead

### Metrics to Track
- Number of CoreTI.Set() calls added
- Number of test cases covering operand types
- Compilation time before/after (should be <5% increase)
- CoreTI coverage percentage (should be 100%)

---

## Key Revisions (Based on Expert Feedback)

### Critical Architectural Adjustments

1. **TVar Acceptance in Validation** ✅
   - Original: Validation required concrete types
   - Revised: Accept TVars as valid (check presence, not concreteness)
   - Rationale: Polymorphic code is valid pre-monomorphization
   - Implementation: Golden tests distinguish TVar (pre-mono) from concrete (post-mono)

2. **M-POLY-B Boundary** ✅
   - Original: Attempted to fix all polymorphic lambda panics
   - Revised: Only fix CoreTI population gaps; var-bound polymorphic panics are M-POLY-B scope
   - Rationale: Separate concerns - this sprint ensures operands have types, M-POLY-B re-elaborates specialized bodies
   - Success criteria: Compilation without CoreTI gaps (runtime panics for var-bound polymorphic are expected)

3. **Fallback Telemetry** ✅
   - Added: Debug counter (gated by `--debug-compile`) tracks CoreTI hits vs fallbacks
   - Rationale: Proves sprint reduces unexpected defaulting
   - Metrics: "CoreTI misses: 3 (OpGt at line 5, OpLe at line 7)"

4. **Post-Substitution Types** ✅
   - Original: Didn't specify when to capture types
   - Revised: Use post-substitution types (after unification) when calling CoreTI.Set()
   - Implementation: Reuse existing ApplySubstitution() in typechecker
   - Rationale: Ensures types are fully resolved, not partial

5. **Phase-Aware Validation** ✅
   - Added: `--strict-lowering` flag (default off) for future post-mono validation
   - Current: Pre-mono accepts TVars, validation checks presence only
   - Future (M-POLY-B): Post-mono requires concrete heads for operators
   - Rationale: Defense in depth without breaking polymorphic code

6. **Enhanced Error Diagnostics** ✅
   - Original: "Missing operand type"
   - Revised: Include operator, arg index, parent NodeID, source location
   - Example: "Missing operand type for Intrinsic(OpGt) arg 0 (parent NodeID 1234, line 5)"
   - Rationale: Shortens root-cause debugging time

7. **Golden Tests for Contract** ✅
   - Golden A: `\x. x > x` compiles with TVar operands (pre-mono valid)
   - Golden B: Same lambda post-specialization requires concrete heads (skipped for now, unskip in M-POLY-B)
   - Rationale: Pins TVar contract, prevents future regressions

### Implementation Notes

- File sizes are well-organized (462 LOC for typechecker_core.go, not 11,858)
- No structural refactoring needed
- ~50 LOC of targeted edits (CoreTI.Set() calls)
- Test-heavy approach (1:1.5 ratio implementation:tests)
- Estimated: 11-16 hours across 1.5-2 days

### Out of Scope (Explicitly)

- Var-bound polymorphic lambda panics (M-POLY-B)
- Re-elaboration of specialized bodies (M-POLY-B)
- Strict post-mono validation (future)
- Removal of ResolvedConstraints fallback (kept as safety net)

---

**Document created**: 2025-10-23
**Last updated**: 2025-10-23
**Revisions**: Incorporated expert feedback on TVar handling, M-POLY-B boundary, telemetry
**Author**: Claude (sprint-planner skill)
**Reviewer feedback**: User (2025-10-23)
**Based on**: `design_docs/planned/v0_3_18/m-dx4-coreti-population-gaps.md`

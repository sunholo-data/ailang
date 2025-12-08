# Sprint Plan: M-SOUNDNESS Effect Checking Implementation

## Summary
Implement effect validation to catch missing effect declarations at compile time, fixing a critical type system soundness bug where functions using `print` (or other effectful builtins) can omit `! {IO}` from their signatures.

**Duration:** 1.5-2 days
**Dependencies:** None (all required infrastructure exists)
**Risk Level:** Low (well-scoped, clear requirements, existing helper functions)

## Current Status Analysis

### Completed Infrastructure
- ✅ Effect subsumption function exists: `types.SubsumeEffectRows()` (internal/types/effects.go:107-125)
- ✅ Helper functions exist: `EffectRowDifference()`, `FormatEffectRow()`
- ✅ CoreTypeInfo validation enforced (M-DX4) - ensures type info available for validation
- ✅ Pipeline integration points ready (after type checking, before lowering)

### Current Bug
- ❌ Functions using `print` compile without declaring `! {IO}` effect
- ❌ Skipped test: `TestPrelude_EntryModuleMissingIO` (internal/pipeline/prelude_test.go:178)
- ❌ Failing benchmark: `print_missing_effect` (expects compilation failure, but compiles successfully)
- ❌ Type system soundness violated: Effect tracking not enforced

### Recent Velocity
- v0.4.3 (Nov 5): M-DX10 string parsing - ~370 LOC in 1-2 days
- v0.4.2 (Nov 2): S-CALL0 hotfix - ~165 LOC in 1 day
- Average: ~200-300 LOC/day for focused features

### Estimated Capacity
- **This Sprint:** ~350-450 LOC (implementation + comprehensive tests)
- **Timeline:** 1.5-2 days (includes testing, integration, documentation)

## Implementation Plan

### Day 1: Core Validation Pass (4-5 hours)

#### Morning: Create Validation Pass
**Task 1.1: Create `internal/pipeline/validate_effects.go`** (~150-200 LOC)

**Functions to implement:**
```go
// Main validation function
func ValidateEffects(coreProg *core.Program, coreTypeInfo map[core.NodeID]types.Type) error

// Extract declared effects from function signature
func extractDeclaredEffects(decl *core.Decl, typeInfo map[core.NodeID]types.Type) *types.Row

// Collect required effects from function body (recursive AST walk)
func collectRequiredEffects(expr core.Expr, typeInfo map[core.NodeID]types.Type) *types.Row

// Build helpful error message
func formatEffectError(decl *core.Decl, required *types.Row, declared *types.Row) error
```

**Algorithm:**
1. For each function in `coreProg.Decls`:
   - Look up function type in `coreTypeInfo`
   - Extract declared effect row from function type
   - Walk function body (Core AST)
   - For each `core.App` node (function call):
     - Look up callee type in `coreTypeInfo`
     - Extract effect row from callee's function type
     - Union with accumulated required effects
   - Check subsumption: `types.SubsumeEffectRows(required, declared)`
   - If false, return detailed error with suggestion

**Acceptance Criteria:**
- [ ] ValidateEffects function walks all function declarations
- [ ] Extracts declared effects from function signatures
- [ ] Collects required effects from function bodies (recursive)
- [ ] Uses existing `types.SubsumeEffectRows()` for validation
- [ ] Returns helpful error with location and fix suggestion
- [ ] Handles pure functions (nil effect rows)

#### Afternoon: Pipeline Integration
**Task 1.2: Integrate into `internal/pipeline/pipeline.go`** (~20-30 LOC)

**Integration points:**
1. **File pipeline** (internal/pipeline/pipeline.go ~line 228):
   ```go
   // After CoreTypeInfo validation
   if err := ValidateCoreTypeInfo(coreProg, typeChecker.CoreTI); err != nil {
       return result, err
   }

   // NEW: Validate effects
   if err := ValidateEffects(coreProg, typeChecker.CoreTI); err != nil {
       return result, fmt.Errorf("effect checking failed: %w", err)
   }

   // Continue to lowering
   linked, err := linker.Link(coreProg, typeChecker.CoreTI)
   ```

2. **Module pipeline** (internal/pipeline/pipeline.go ~line 631):
   Same integration after type checking

3. **REPL** (internal/repl/repl_eval.go ~line 113):
   Same integration for interactive evaluation

**Acceptance Criteria:**
- [ ] Effect validation runs after CoreTypeInfo validation
- [ ] Effect validation runs before lowering in all 3 pipelines
- [ ] Errors are caught and reported with clear messages
- [ ] All existing tests still pass

### Day 2: Testing & Documentation (4-5 hours)

#### Morning: Comprehensive Testing
**Task 2.1: Create `internal/pipeline/validate_effects_test.go`** (~150-200 LOC)

**Test cases (8 scenarios):**

1. **TestEffectValidation_Pure** - Pure function with no effects
   ```go
   func add(x: int, y: int) -> int { x + y }
   ```
   Expected: ✅ Pass (no effects used, none declared)

2. **TestEffectValidation_MissingIO** - Uses print without declaring IO
   ```go
   export func main() -> () { print("hi") }
   ```
   Expected: ❌ Fail with error containing "! {IO}"

3. **TestEffectValidation_DeclaredIO** - Correctly declares IO
   ```go
   export func main() -> () ! {IO} { print("hi") }
   ```
   Expected: ✅ Pass

4. **TestEffectValidation_MissingFS** - Uses readFile without declaring FS
   ```go
   func f() -> () ! {IO} { readFile("x.txt") }
   ```
   Expected: ❌ Fail (missing FS effect)

5. **TestEffectValidation_MultipleEffects** - Declares and uses IO+FS
   ```go
   func f() -> () ! {IO, FS} {
     print("reading");
     readFile("x.txt")
   }
   ```
   Expected: ✅ Pass

6. **TestEffectValidation_OverDeclared** - Declares more effects than needed
   ```go
   func f() -> () ! {IO, FS} { print("hi") }
   ```
   Expected: ✅ Pass (subsumption allows over-declaration)

7. **TestEffectValidation_NestedCalls** - Helper function with effects
   ```go
   func helper() -> () ! {IO} { print("hi") }
   func main() -> () { helper() }
   ```
   Expected: ❌ Fail (main missing IO effect)

8. **TestEffectValidation_LibraryModule** - Non-entry module
   ```go
   module foo
   func bar() -> () { print("hi") }  // No prelude, should still validate
   ```
   Expected: Test that validation works for library modules too

**Acceptance Criteria:**
- [ ] All 8 test cases implemented
- [ ] Tests verify error messages contain suggestions
- [ ] Tests verify subsumption rules (over-declaration OK)
- [ ] Tests cover nested calls (transitive effects)
- [ ] All tests passing
- [ ] Test coverage >90% for validate_effects.go

#### Afternoon: Fix Skipped Test & Benchmark
**Task 2.2: Un-skip `TestPrelude_EntryModuleMissingIO`** (~20 LOC)

Update test in `internal/pipeline/prelude_test.go`:
```go
func TestPrelude_EntryModuleMissingIO(t *testing.T) {
    // Remove t.Skip()

    code := `
module tmp/test_missing_effect

export func main() -> () {
    print("Hello!")
}
`
    result := RunModule(code, "main", []string{})

    // Should fail with effect error
    require.False(t, result.Success)
    require.Contains(t, result.Stderr, "! {IO}")
    require.Contains(t, result.Stderr, "print requires IO effect")
}
```

**Task 2.3: Verify `print_missing_effect` benchmark passes** (~10 min)

After implementation, run:
```bash
ailang eval-suite --models gpt5-mini --benchmarks print_missing_effect
```

Expected result:
- If AI generates `export func main() -> () ! {IO}` → ✅ Compiles (correct!)
- If AI generates `export func main() -> ()` → ❌ Compilation fails with "! {IO}" error (expected!)
- Benchmark passes because compilation behavior matches spec

**Acceptance Criteria:**
- [ ] Skipped test un-skipped and passing
- [ ] Test verifies error message is helpful
- [ ] Benchmark `print_missing_effect` passes
- [ ] No regression in other tests

#### Late Afternoon: Documentation & Examples
**Task 2.4: Update documentation** (~30 min)

**Files to update:**

1. **CHANGELOG.md** - Add entry for v0.4.3:
   ```markdown
   ### Fixed - Effect Checking Soundness (M-SOUNDNESS) ⚠️ Type Safety

   **User Impact**: Functions using `print` or other effectful builtins now MUST declare effects in their signatures. Previously silent violations now caught at compile time.

   **What Was Fixed**:
   - Effect validation pass added to pipeline (validates all function declarations)
   - Helpful error messages pointing to missing effects
   - Leverages existing `SubsumeEffectRows` function (internal/types/effects.go)

   **Example Error:**
   ```
   Error: function main uses IO effect but signature does not declare it
     at test.ail:3:5 (print call)

     Current signature: export func main() -> ()
     Suggested fix:     export func main() -> () ! {IO}
   ```

   **Files Added**:
   - internal/pipeline/validate_effects.go: ~180 LOC (validation pass)
   - internal/pipeline/validate_effects_test.go: ~200 LOC (8 test scenarios)

   **Files Modified**:
   - internal/pipeline/pipeline.go: +3 LOC (integrate validation)
   - internal/pipeline/prelude_test.go: Un-skipped test

   **Benchmark Impact**: `print_missing_effect` now passes (compilation correctly rejects missing effects)

   **Resolves**: M-SOUNDNESS (P0 - Type System Soundness)
   ```

2. **design_docs/planned/v0_4_3/m-soundness-effect-checking-prelude.md** - Add completion section:
   ```markdown
   ## Implementation Report

   **Status:** ✅ COMPLETE (v0.4.3)
   **Actual Duration:** 1.5 days
   **Actual LOC:** ~400 LOC (180 impl + 200 tests + 20 integration)

   **What Was Built:**
   - Effect validation pass in internal/pipeline/validate_effects.go
   - Comprehensive test suite (8 scenarios, >90% coverage)
   - Pipeline integration (file, module, REPL)
   - Helpful error messages with fix suggestions

   **Deviations from Plan:**
   - None - implementation followed design doc exactly

   **Lessons Learned:**
   - Existing helper functions (SubsumeEffectRows, etc.) made implementation straightforward
   - CoreTypeInfo validation (M-DX4) was essential prerequisite
   - Error message quality matters - included fix suggestions
   ```

3. **examples/** - Create `examples/effect_checking_error.ail`:
   ```ailang
   -- This example demonstrates effect checking errors
   -- Run: ailang check effect_checking_error.ail
   -- Expected: Compilation error (missing ! {IO} effect)

   module examples/effect_checking_error

   -- ❌ WRONG: Uses print but doesn't declare IO effect
   export func main() -> () {
       print("This will fail to compile!")
   }

   -- ✅ CORRECT: Declares IO effect
   -- export func main() -> () ! {IO} {
   --     print("This compiles successfully!")
   -- }
   ```

**Acceptance Criteria:**
- [ ] CHANGELOG.md updated with feature details
- [ ] Design doc moved to `implemented/v0_4_3/` with completion report
- [ ] Example file created demonstrating error case
- [ ] Documentation links in error messages (future: add URL to effect guide)

## Success Metrics

### Code Quality
- [ ] **Test coverage:** >90% for validate_effects.go
- [ ] **All tests passing:** 400+ existing tests + 8 new tests
- [ ] **Linting clean:** `make lint` passes
- [ ] **Examples verified:** Effect checking example works as expected

### Functionality
- [ ] **Effect violations caught:** Code without `! {IO}` fails to compile
- [ ] **Error messages helpful:** Include location, problem, and fix suggestion
- [ ] **Subsumption correct:** Over-declared effects are accepted
- [ ] **No false positives:** Pure code still compiles

### Benchmarks
- [ ] **print_missing_effect passes:** Benchmark now validates correctly
- [ ] **No regressions:** All other effect-related benchmarks still pass
- [ ] **Eval baseline:** Run baseline to verify no new failures

### Documentation
- [ ] **CHANGELOG.md updated:** v0.4.3 entry with metrics
- [ ] **Design doc updated:** Moved to implemented/ with completion report
- [ ] **Examples created:** Demonstrate effect checking behavior

## Risk Mitigation

### Risk 1: False Positives (Functions incorrectly rejected)
**Likelihood:** Low (subsumption logic is well-tested)
**Impact:** High (breaks valid code)
**Mitigation:**
- Comprehensive test suite covering edge cases
- Manual testing with existing effect-heavy code
- Review subsumption implementation before integrating

### Risk 2: Performance Impact (Validation slows compilation)
**Likelihood:** Low (single O(n) AST walk)
**Impact:** Low (negligible for typical programs)
**Mitigation:**
- Validation only walks function bodies once
- No complex algorithms (just map lookups + unions)
- Measure compilation time before/after (expect <5% impact)

### Risk 3: Error Messages Confusing
**Likelihood:** Medium (effect system can be subtle)
**Impact:** Medium (users don't understand errors)
**Mitigation:**
- Include fix suggestions in every error
- Point to exact call site that requires effect
- Future: add documentation links

### Risk 4: Breaks Existing Code
**Likelihood:** Low (AI models already write correct code)
**Impact:** Medium (users need to add effect declarations)
**Mitigation:**
- This is a bug fix, not a breaking change (code was always wrong)
- AI prompts already document `! {IO}` requirement
- Clear migration path: add `! {IO}` to signatures

## Dependencies

### Prerequisites (All Met)
- ✅ CoreTypeInfo validation (M-DX4) - ensures type info available
- ✅ Effect subsumption function (internal/types/effects.go)
- ✅ Helper functions (EffectRowDifference, FormatEffectRow)
- ✅ Pipeline integration points (after type checking)

### No Blocking Issues
- Implementation can start immediately
- All required infrastructure exists
- No dependency on other in-flight work

## Open Questions

### Q1: Should we validate REPL inputs?
**Answer:** Yes - REPL also gets prelude, so same rules apply
**Implementation:** Add validation to repl_eval.go after type checking

### Q2: Should we add documentation URLs to error messages?
**Answer:** Future enhancement (v0.4.4+) - not blocking for this sprint
**Reason:** Need to create effect system documentation page first

### Q3: Should we infer effects instead of requiring declarations?
**Answer:** Future feature (v0.5.0+) - out of scope for this sprint
**Reason:** Effect inference is complex, requires bidirectional type checking
**For now:** Require explicit declarations (consistent with AILANG's explicit philosophy)

## Notes

### Why This Is Critical
- **Type system soundness:** AILANG claims static effect tracking, but it's not enforced
- **User trust:** Silently wrong code erodes confidence in type system
- **AI training:** AI models need accurate error feedback to learn correct code

### Why This Is Low Risk
- **Small scope:** 350-450 LOC with clear requirements
- **Existing infrastructure:** Subsumption function already exists and tested
- **No breaking changes:** Only catches bugs that should have been caught
- **Recent similar work:** M-DX4 (CoreTypeInfo validation) used similar integration pattern

### Implementation Notes
- **Use existing helpers:** Don't reimplement subsumption or formatting
- **Follow M-DX4 pattern:** Similar validation pass structure
- **Error message quality:** Include location, problem, fix suggestion
- **Test edge cases:** Nested calls, over-declaration, pure functions

### Post-Sprint Work (Future)
- **Better diagnostics:** Point to exact call site (not just function)
- **Effect inference:** Don't require explicit declarations
- **Documentation:** Create effect system guide with examples
- **Capability budgets:** `! {IO @limit=10}` for resource tracking

## Timeline

### Day 1 (4-5 hours)
- **Morning:** Create validate_effects.go (150-200 LOC)
- **Afternoon:** Integrate into pipeline (20-30 LOC)
- **Checkpoint:** Validation pass works, compiles, basic manual testing

### Day 2 (4-5 hours)
- **Morning:** Write comprehensive tests (150-200 LOC)
- **Afternoon:** Un-skip test, verify benchmark, update docs
- **Checkpoint:** All tests passing, documentation complete, ready for PR

### Total: 1.5-2 days (8-10 hours)

## Approval

**Ready to implement?** Please confirm:
- [ ] Approach looks reasonable (use existing SubsumeEffectRows)
- [ ] Timeline is realistic (1.5-2 days)
- [ ] Success metrics are clear
- [ ] No concerns about breaking changes

Once approved, implementation will follow this plan with daily progress updates.

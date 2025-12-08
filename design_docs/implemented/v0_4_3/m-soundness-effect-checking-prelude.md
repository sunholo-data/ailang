# M-SOUNDNESS: Effect Checking for Entry Module Prelude

**Status:** ✅ IMPLEMENTED (2025-11-06)
**Version:** v0.4.3
**Priority:** **CRITICAL** - Type system soundness issue
**Actual Effort:** ~3 hours (1 session)

> **Implementation Report:** See [M-SOUNDNESS-COMPLETION-REPORT.md](M-SOUNDNESS-COMPLETION-REPORT.md) for complete details.

## Problem Statement

The entry-module prelude system (v0.3.16) injects `print : string -> () ! {IO}` into the type environment, but **does not validate** that functions using `print` declare the IO effect in their signature.

**Soundness Violation:**
```ailang
module tmp/test_missing_effect

export func main() -> () {  -- ❌ Missing ! {IO}
  print("Hello!")           -- ✅ Compiles and runs successfully
}
```

**Expected behavior:** Compilation should **fail** with error like:
```
Error: print requires IO effect
  Add ! {IO} to main signature: export func main() -> () ! {IO}
```

**Actual behavior:** Compiles and runs successfully, **silently violating type safety**.

## Discovery Context

**Benchmark:** `print_missing_effect`
- Spec says: `expect_failure: true`, `expected_stderr_contains: "! {IO}"`
- Caps: `[]` (intentionally no capabilities)
- AI generated: `export func main() -> () ! {IO}` (correctly added IO)
- **Issue:** Benchmark can't test effect checking because AI writes correct code

**Manual test:** Created code WITHOUT IO effect declaration
- Result: Compiles and runs successfully
- Confirms effect checking is not implemented

## Root Cause Analysis

### 1. Prelude Injection (v0.3.16)

**File:** [internal/pipeline/prelude.go](internal/pipeline/prelude.go)

```go
// Injects print : string -> () ! {IO} into type environment
func injectPrelude(env *types.Env, isEntry bool) {
    if isEntry {
        env.AddBuiltin("print", printType)  // Adds to type env
    }
}
```

**Problem:** Only adds `print` to type environment, doesn't check if callers declare IO effect.

### 2. Missing Effect Subsumption Check

**File:** [internal/types/effects.go:107-125](internal/types/effects.go#L107-L125)

```go
// SubsumeEffectRows checks if effect row 'a' is subsumed by effect row 'b'
// Returns true if all effects in 'a' are present in 'b'
func SubsumeEffectRows(a, b *Row) bool {
    if a == nil {
        return true // Pure is subsumed by anything
    }
    if b == nil {
        return a == nil // Only pure is subsumed by pure
    }

    // All labels in 'a' must be in 'b'
    for k := range a.Labels {
        if _, ok := b.Labels[k]; !ok {
            return false
        }
    }
    return true
}
```

**Problem:** Function exists but **is never called**. Effect subsumption is not validated anywhere.

### 3. Skipped Test

**File:** [internal/pipeline/prelude_test.go:178-187](internal/pipeline/prelude_test.go#L178-L187)

```go
func TestPrelude_EntryModuleMissingIO(t *testing.T) {
    t.Skip("TODO: Effect diagnostic test needs runModule path with prelude - testing via manual verification for now")

    // When implemented, this should verify:
    // 1. Entry module with print("x") but main() -> () (no ! {IO})
    // 2. Should get helpful error: "print requires IO effect. Add ! {IO} to main signature"
}
```

**Problem:** Test is skipped with TODO - effect checking was **never implemented**, only noted as future work.

## Impact Assessment

### Type System Soundness

**Critical:** AILANG claims to have static effect tracking, but it's not enforced for prelude builtins.

**Affected code:**
- Any entry module using `print` without declaring `! {IO}`
- Potentially other builtins if they have effects

**Scope:**
- Entry modules only (library modules don't get prelude)
- REPL (also has prelude injection)

### Eval Failures

**Direct impact:** 1 benchmark (`print_missing_effect`)
- Benchmark expects compilation failure
- AILANG allows it, so benchmark fails

**Indirect impact:** Potentially more if AI models start omitting effect declarations
- Currently AI models correctly add `! {IO}` (trained on correct code)
- But this is luck, not enforcement

## Design Goals

1. **Sound effect checking** - Functions using effectful builtins MUST declare those effects
2. **Clear error messages** - Point to exact location and suggest fix
3. **Retroactive validation** - Check after type inference completes
4. **No breaking changes** - Only catch errors that should have been caught

## Proposed Solution

### Phase 1: Function-Level Effect Validation (v0.4.3)

**When:** After type checking, before lowering

**What:** For each function declaration:
1. Collect all effects used in function body (from function calls)
2. Compare against declared effect row in signature
3. Error if required effects are missing

**Where:** New validation pass in pipeline

```go
// internal/pipeline/validate_effects.go
func ValidateEffects(prog *core.Program, typeInfo map[core.NodeID]types.Type) error {
    for _, decl := range prog.Decls {
        // Get declared effects from function signature
        declaredEffects := extractDeclaredEffects(decl, typeInfo)

        // Collect required effects from function body
        requiredEffects := collectRequiredEffects(decl.Body, typeInfo)

        // Check subsumption
        if !types.SubsumeEffectRows(requiredEffects, declaredEffects) {
            missing := types.EffectRowDifference(requiredEffects, declaredEffects)
            return fmt.Errorf(
                "function %s uses effects %v but signature declares %v\n" +
                "  Suggestion: Add ! {%s} to function signature",
                decl.Name, formatEffects(requiredEffects),
                formatEffects(declaredEffects), strings.Join(missing, ", "))
        }
    }
    return nil
}
```

**Algorithm:**
1. Walk function body (Core AST)
2. For each App node (function call):
   - Look up callee type in CoreTypeInfo
   - Extract effect row from function type
   - Union with accumulated effects
3. Compare accumulated effects against declared effects
4. Error if required effect is missing

**Error messages:**
```
Error: function main uses IO effect but signature does not declare it
  at test.ail:3:5 (print call)

  Current signature: export func main() -> ()
  Suggested fix:     export func main() -> () ! {IO}
```

### Phase 2: Better Diagnostics (v0.4.4+)

**Enhancements:**
- Point to exact call site that needs effect
- Suggest minimal effect set to add
- Link to docs explaining effects
- Special handling for common cases (print, println, readFile)

### Phase 3: Effect Polymorphism (Future)

**Out of scope for v0.4.3**, but noted for future:
- Effect variables in signatures: `func f() -> () ! {e}`
- Effect constraints
- Effect inference (don't require explicit declarations)

## Implementation Plan

### Step 1: Add Validation Pass (2-3 hours)

**Create:** `internal/pipeline/validate_effects.go`
```go
// ValidateEffects ensures declared effects subsume required effects
func ValidateEffects(coreProg *core.Program, coreTypeInfo map[core.NodeID]types.Type) error
```

**Functions needed:**
- `extractDeclaredEffects(decl, typeInfo)` - Get effect row from function signature
- `collectRequiredEffects(expr, typeInfo)` - Walk AST, collect effects from calls
- `formatEffectError(decl, required, declared)` - Build helpful error message

### Step 2: Integrate into Pipeline (1 hour)

**File:** `internal/pipeline/pipeline.go`

Add after type checking, before lowering:
```go
// After type checking
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

### Step 3: Write Tests (2-3 hours)

**File:** `internal/pipeline/validate_effects_test.go`

**Test cases:**
1. **Pure function** - No effects used, no effects declared → Pass
2. **Pure declared, IO used** - Uses print, declares `() -> ()` → **Fail**
3. **IO declared, IO used** - Uses print, declares `() -> () ! {IO}` → Pass
4. **IO declared, FS used** - Uses readFile, declares `! {IO}` → **Fail**
5. **IO+FS declared, IO+FS used** - Uses print+readFile, declares `! {IO, FS}` → Pass
6. **More effects than needed** - Declares `! {IO, FS}` but only uses IO → Pass (subsumption)
7. **Nested calls** - `main` calls `helper` which uses print → Require IO in both
8. **Library module** - Should work same as entry module (prelude not involved)

### Step 4: Fix Benchmark (30 minutes)

**File:** `benchmarks/print_missing_effect.yml`

Current spec is correct - expects compilation failure.

**Result after fix:** Benchmark will pass because:
- AI generates: `export func main() -> () ! {IO}`
- Effect checker validates: Required IO ⊆ Declared IO → Pass
- If AI forgets: `export func main() -> ()`
- Effect checker validates: Required IO ⊈ Declared ∅ → **Fail** (expected!)

### Step 5: Update Documentation (1 hour)

**Files to update:**
- `CHANGELOG.md` - Note effect checking fix in v0.4.3
- `docs/guides/effects.md` - Document effect checking rules
- `prompts/v0.4.3.md` - No changes needed (already documents `! {IO}` requirement)

## Testing Strategy

### Unit Tests
- `internal/pipeline/validate_effects_test.go` - 8 test cases above
- `internal/types/effects_test.go` - Test SubsumeEffectRows (if not already covered)

### Integration Tests
- Create manual test files in `tests/effects/`
- Verify error messages are helpful

### Regression Tests
- Run full eval baseline to ensure no new failures
- Expect `print_missing_effect` to pass

## Success Metrics

- ✅ Code with missing effects fails to compile
- ✅ Error message points to problem and suggests fix
- ✅ `print_missing_effect` benchmark passes
- ✅ All existing benchmarks still pass
- ✅ No false positives (valid code rejected)
- ✅ Test coverage >90% for new validation code

## Migration Notes

**For AI models:**
- **No change needed** - Already generating `! {IO}` correctly
- Prompt already documents effect requirements

**For existing code:**
- **Breaking change:** Code that was silently wrong will now fail
- But this is a bug fix - code was always incorrect, just not caught
- Very few real programs affected (AI models write correct code)

## Risks & Mitigations

### Risk 1: False Positives
**Mitigation:** Comprehensive test suite, manual testing

### Risk 2: Performance Impact
**Mitigation:** Single pass, O(n) in AST size, should be negligible

### Risk 3: Complex Error Messages
**Mitigation:** Iterate on error formatting, add examples

## Related Issues

**Exhaustiveness checking:** Similar issue with `no_runtime_crashes_option` - non-exhaustive patterns not caught at compile time. May need similar validation pass.

## Timeline

- **Design:** 1 hour (done)
- **Implementation:** 3-4 hours (validation pass + pipeline integration)
- **Testing:** 2-3 hours (unit + integration tests)
- **Documentation:** 1 hour
- **Total:** 7-9 hours

## Dependencies

- CoreTypeInfo populated (already done in M-DX4)
- Type inference complete (already done)
- SubsumeEffectRows function (already exists, just needs to be used)

## Future Work (Out of Scope)

1. **Effect inference** - Don't require explicit `! {IO}` declarations
2. **Effect polymorphism** - Generic over effects
3. **Effect handlers** - Algebraic effects and handlers
4. **Capability budgets** - `! {IO @limit=10}` for resource bounds

## Conclusion

This is a **critical soundness bug** that's been present since v0.3.16 when the prelude system was added. The fix is straightforward (use existing `SubsumeEffectRows` function), and the impact is minimal (only catches bugs that should have been caught).

**Priority: CRITICAL** - This affects type system soundness, a core value proposition of AILANG.

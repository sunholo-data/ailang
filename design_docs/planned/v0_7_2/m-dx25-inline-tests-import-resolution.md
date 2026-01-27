# Fix: Inline Tests Fail with Complex Imports

**Status**: Planned (New)
**Target**: v0.7.2
**Priority**: P1 - Critical bug blocking inline tests
**Estimated**: 2-3 days
**Depends**: M-TESTING-INLINE (v0.4.7, completed)

## Problem Statement

**Inline tests fail with "harness evaluation failed: cannot apply non-function value: <nil>" when testing functions that call imported functions.**

### Reproduction Steps

1. Create a module that imports `std/fs`, `std/net`, or `std/json`
2. Add a pure function that calls an imported function
3. Add inline tests to that function
4. Run `ailang test module.ail`
5. All tests fail with nil function error

### Example Failure

```ailang
module test_with_imports
import std/fs

pure func read_and_process(path: string) -> string {
    let content = fs.read(path)
    content
}

tests [
    (read_and_process("test.txt"), "hello")
]
```

**Error**: `harness evaluation failed: cannot apply non-function value: <nil>`

### Why This Matters

- Inline tests are the primary testing mechanism for AILANG
- Many modules import stdlib functions (std/fs, std/net, std/json, etc.)
- Users cannot test functions that use imported capabilities
- This blocks practical testing of real-world code

### Current Workaround

None - users must avoid importing stdlib if they want to use inline tests.

## Root Cause Analysis

### Issue Identification: Environment Variable Capture

The test harness evaluation has a critical flaw in how it resolves imported functions:

**File**: `internal/testing/executor.go`
**Method**: `injectModuleBindings()` (lines 726-779)
**Problem**: Environment Variable Capture in FunctionValue

```go
func (e *Executor) injectModuleBindings(evaluator *eval.CoreEvaluator, env *eval.Environment) {
    for _, mod := range e.modules {
        for _, decl := range mod.Core.Decls {
            case *core.Let:
                if lambda, ok := d.Value.(*core.Lambda); ok {
                    funcVal := &eval.FunctionValue{
                        Params: extractLambdaParams(lambda),
                        Body:   lambda.Body,
                        Env:    env,              // ← BUG: Captures environment reference
                        Typed:  true,
                    }
                    env.Set(d.Name, funcVal)
                }
        }
    }
}
```

### Why This Breaks

1. **FunctionValue Closure Capture**: The `Env` field captures a reference to the current environment at the time the FunctionValue is created
2. **Transitive Function Calls**: When a function from the test file calls an imported function:
   - The test harness wraps the user function in a LetRec
   - The user function's body references an imported function
   - At evaluation time, the imported function was injected as a FunctionValue with a captured Env
3. **Resolution Failure**: When the imported function's body executes:
   - It tries to resolve references within its body
   - But those references are resolved using its captured environment
   - If the captured environment doesn't have nested dependencies, resolution fails
   - The resolver returns nil instead of a proper error

### Systemic Issue Check

**Question**: Is this isolated to imported functions, or does it affect any transitive function calls?

**Answer**: **This affects ANY function that calls other functions during test evaluation**, but the symptom is most visible with:
- Imported functions (from stdlib) ✓ Reported bug
- User-defined helper functions ✓ Likely affected
- Builtin functions ✓ Should work (special case)
- ADT constructors ✓ Should work (injected separately)

**Systemic Fix Required**: Fix the entire environment resolution chain, not just imported functions.

## Solution Design

### Root Fix: Use CombinedResolver for All Function Calls

Instead of capturing environment references in FunctionValue, all function calls should use the CombinedResolver which has access to:
1. Builtins registry
2. Module cache
3. Current environment
4. ADT constructors

### Implementation Strategy

**Phase 1: Fix Module Binding Injection (0.5 days)**
- Change `injectModuleBindings()` to NOT capture the environment in FunctionValue
- Instead, rely on the CombinedResolver at evaluation time
- This requires modifying how eval.FunctionValue stores and uses environment

**Phase 2: Fix Transitive Resolution (1 day)**
- Ensure GlobalResolver is used for ALL variable references, not just global ones
- Add test case for nested function calls
- Verify ADT constructors still work

**Phase 3: Test Coverage (0.5 days)**
- Add regression test: inline tests with imported functions
- Add regression test: inline tests with helper functions
- Add regression test: inline tests with complex call chains
- Verify existing tests still pass

### Key Changes Required

**File: `internal/testing/executor.go`**

```go
// Current (broken):
funcVal := &eval.FunctionValue{
    Params: extractLambdaParams(lambda),
    Body:   lambda.Body,
    Env:    env,              // ← Captured reference (WRONG)
    Typed:  true,
}
env.Set(d.Name, funcVal)

// Fixed:
// Option 1: Don't capture environment, let resolver handle it
funcVal := &eval.FunctionValue{
    Params: extractLambdaParams(lambda),
    Body:   lambda.Body,
    Env:    nil,              // ← Nil = use global resolver
    Typed:  true,
}
env.Set(d.Name, funcVal)

// Option 2: Capture environment AFTER ALL bindings injected
// (requires two-pass injection)
```

**File: `internal/eval/eval_evaluator.go`**

- Verify that GlobalResolver is consulted for all Var and GlobalRef evaluations
- Add special case handling for nil Env in FunctionValue (use global resolver)

**File: `internal/testing/integration_test.go`**

Add regression tests:
```go
func TestIntegration_InlineTestsWithImportedFunctions(t *testing.T) {
    // Test that functions calling imported stdlib functions work in inline tests
    // Example: pure func that calls std/fs.read
}

func TestIntegration_InlineTestsWithHelperFunctions(t *testing.T) {
    // Test that functions calling other user-defined functions work
}

func TestIntegration_InlineTestsWithComplexCallChains(t *testing.T) {
    // Test deep call chains: main -> helper1 -> helper2 -> stdlib.func
}
```

## Technical Details

### How GlobalResolver Works

```go
type CombinedResolver struct {
    Builtins *BuiltinRegistry
    Env      *Environment
    Modules  map[string]*LoadedModule
}

func (r *CombinedResolver) ResolveValue(ref core.GlobalRef) (eval.Value, error) {
    // 1. Try builtins first
    // 2. Try environment
    // 3. Try modules map
    // 4. Return error if not found
}
```

This resolver has access to:
- All builtin functions
- All injected environment bindings
- All loaded modules (for qualified references like std/fs.read)

### Environment Lifecycle in Test Context

```
EvaluateInlineTestsWithHarness
├── evaluator := NewCoreEvaluator()
│   └── env: Fresh environment with registerBuiltins()
├── injectModuleBindings(evaluator, env)
│   └── For each module function:
│       └── Create FunctionValue with Env: nil (use global resolver)
│       └── env.Set(name, funcVal)
├── injectADTConstructors(evaluator)
├── resolver := CombinedResolver{
│       Builtins: builtinRegistry,
│       Env:      env,
│       Modules:  e.modules,
│   }
├── evaluator.SetGlobalResolver(resolver)
├── result := evaluator.EvalCoreProgram(coreProg)
│   └── During evaluation:
│       └── When evaluating App(f, arg):
│           └── If f is FunctionValue with Env:nil
│               └── Call Body using global resolver
└── Return result
```

## Testing Plan

### Unit Tests

1. **Test Module Binding Injection**
   - Create FunctionValue from imported stdlib function
   - Verify it has Env: nil (or can use global resolver)
   - Verify resolution works during evaluation

2. **Test Transitive Function Calls**
   - User function → imported function
   - User function → helper function → imported function
   - Deep call chains (3+ levels)

3. **Test ADT Constructors Still Work**
   - Inline tests with ADT pattern matching
   - Constructors from source file
   - Constructors from imported modules (if any)

### Integration Tests

1. **Regression: Inline Tests with std/fs**
   - Module imports std/fs
   - Function calls fs.readLine or similar
   - Inline tests call that function
   - Tests pass

2. **Regression: Inline Tests with std/net**
   - Module imports std/net
   - Function uses net.dial or similar
   - Inline tests call that function
   - Tests pass

3. **Regression: Inline Tests with std/json**
   - Module imports std/json
   - Function calls json.decode or similar
   - Inline tests call that function
   - Tests pass

4. **Regression: Inline Tests with Helper Functions**
   - Module with multiple user-defined functions
   - Main function calls helper
   - Inline tests on main function
   - Tests pass

### Example Files

Create `examples/testing_with_imports.ail`:
```ailang
module examples/testing_with_imports

// Test inline tests with imported functions
pure func format_number(n: int) -> string {
    let str = _int_to_string(n)
    str
}

tests [
    (format_number(42), "42"),
    (format_number(-1), "-1"),
    (format_number(0), "0")
]
```

## Acceptance Criteria

- [ ] All inline tests pass with `std/fs` imports
- [ ] All inline tests pass with `std/net` imports
- [ ] All inline tests pass with `std/json` imports
- [ ] Helper function calls work in inline tests
- [ ] Transitive function calls (3+ levels) work
- [ ] Existing inline test tests still pass
- [ ] No regression in non-imported modules
- [ ] Error messages are clear when resolution fails
- [ ] Example files created and verified working

## Risk Analysis

### Medium Risk Areas

1. **Changing FunctionValue Environment Capture**: This is used in multiple places
   - Check all callers of FunctionValue creation
   - Verify they don't rely on captured Env
   - Risk: Silent failures in other code paths

2. **GlobalResolver Behavior**: Making it the primary resolver
   - Must handle all reference types (Var, GlobalRef, ConstructorRef)
   - Must not break existing tests
   - Risk: Subtle type resolution failures

### Mitigation

- Run full test suite after each change
- Add regression tests for known issues
- Use DEBUG_STRICT=1 to catch panics
- Test with actual modules (gcp_auth.ail, ga4_queries.ail)

## Estimated LOC

- `executor.go` changes: ~50 LOC (modify injectModuleBindings)
- `eval_evaluator.go` changes: ~30 LOC (handle nil Env)
- Test additions: ~150 LOC (3 integration tests, examples)
- **Total**: ~230 LOC implementation, ~150 LOC tests

## References

**Related Bugs/Docs**:
- M-BUG-ADT-TEST-HARNESS-SCOPE (v0.5.0) - Similar scoping issue with ADT constructors
- M-TESTING-INLINE-CORE-EVALUATION (v0.4.7) - Original inline test implementation
- M-DX23-INLINE-TESTS-DOCUMENTATION (v0.7.1) - Documentation of inline test feature

**Key Files**:
- `internal/testing/executor.go:726-779` - injectModuleBindings
- `internal/testing/executor.go:22-84` - CombinedResolver
- `internal/testing/integration_test.go:259-268` - Skipped regression test
- `internal/eval/eval_evaluator.go` - GlobalResolver handling

## Timeline

- Day 1 (0.5 days): Understand current resolver flow, write minimal fix
- Day 1 (0.5 days): Run existing tests, identify breakage
- Day 2 (1 day): Add regression tests, verify fix works
- Day 3 (0.5 days): Test with real modules (gcp_auth, ga4_queries)
- Day 3 (0.5 days): Documentation, example files, cleanup

## Version Target

**v0.7.2** - Bug fix release (after v0.7.1 with inline test documentation)

## Open Questions

1. Should we preserve captured environment for performance, or always use global resolver?
   - Current answer: Use global resolver (correctness over micro-optimization)

2. Should we add a "strict mode" flag to catch missing functions earlier?
   - Current answer: Leave for future improvement (M-DX26)

3. Do we need to handle module-qualified function calls (e.g., `std/fs.read`)?
   - Current answer: Yes, CombinedResolver already supports this

# M-DX26: Property Test "Empty Program" Bug

**Status:** Planned
**Target:** v0.7.2
**Priority:** P1 (High - blocks property-based testing)
**Estimated:** 4-6 hours
**Dependencies:** None
**Created:** 2026-01-28

## Problem Statement

Property tests auto-generated from `requires`/`ensures` contracts fail with "empty program" error, blocking property-based testing functionality.

**Current behavior:**
```bash
$ ailang test examples/contracts_demo.ail

Tests:
  ✓ divide_test_1 (inline tests work)
  ✓ divide_test_2
  ✓ divide_test_3

Properties:
  ✗ divide_property_1 (1 cases, 138.666µs)
      test 0: evaluation failed: empty program
  ✗ divide_property_2 (1 cases, 142.333µs)
      test 0: evaluation failed: empty program
```

**Impact:**
- Critical - property-based testing completely broken
- Discovered during ecommerce demo (feedback message a7dd508c)
- Blocks M-VERIFY adoption (requires/ensures useless without property tests)
- User workaround: manually write inline tests instead of using contracts

## Root Cause Analysis

### The Execution Flow

1. **Property collection** (`internal/testing/collector.go:137`):
   - Extracts `forall` clauses from `requires`/`ensures`
   - Creates `PropertyCase` with AST expression

2. **Value generation** (`internal/testing/runner.go:183-188`):
   - Generates random values using type-based generators
   - Binds values to property parameters

3. **Expression evaluation** (`internal/testing/runner.go:194`):
   - Calls `r.executor.EvaluateExpression(boundExpr)`
   - **THIS IS WHERE IT FAILS**

4. **Source reconstruction** (`internal/testing/executor.go:116-147`):
   ```go
   func (e *Executor) EvaluateExpression(expr ast.Expr) (eval.Value, error) {
       var sourceParts []string

       // Reconstruct pure function sources
       for _, f := range e.sourceFile.Funcs {
           funcSrc := fmt.Sprintf("pure func %s(", f.Name)
           // ... build params ...
           funcSrc += fmt.Sprintf(") -> %v {\n", f.ReturnType)
           funcSrc += "  " + fmt.Sprintf("%v", f.Body) + "\n}\n\n"
           sourceParts = append(sourceParts, funcSrc)
       }

       // Add test expression
       sourceParts = append(sourceParts, fmt.Sprintf("%v", expr))

       // Concatenate and run through pipeline
       source := strings.Join(sourceParts, "")
       result, err := pipeline.Run(cfg, src)
   }
   ```

5. **Pipeline rejection** (`internal/pipeline/pipeline_single.go:297-300`):
   ```go
   if len(program.Items) == 0 {
       return result, fmt.Errorf("empty program: expected at least one item after parse")
   }
   ```

### The Bug

**Using `fmt.Sprintf("%v", ast.Expr)` to reconstruct source code is fundamentally broken!**

AST nodes may not have proper `String()` methods, resulting in:
- Empty strings
- Invalid AILANG syntax
- Incomplete expressions
- Missing operators or parentheses

**Evidence from code:**
- Line 135: `fmt.Sprintf(") -> %v {\n", f.ReturnType)` - may print empty
- Line 136: `fmt.Sprintf("%v", f.Body)` - may print empty or invalid syntax
- Line 142: `fmt.Sprintf("%v", expr)` - may print empty

When the reconstructed source is empty or unparseable, the pipeline sees an "empty program" and rejects it.

### Why Inline Tests Work

Inline tests use a different path (`EvaluateInlineTestsWithHarness` at runner.go:88) that:
1. Extracts Core binding directly from typed AST
2. Builds synthetic Core program with test harness
3. Evaluates Core directly (no source reconstruction)

This avoids the AST→Source conversion entirely!

## Comparison: Inline Tests vs Property Tests

| Aspect | Inline Tests (✓ Works) | Property Tests (✗ Broken) |
|--------|----------------------|--------------------------|
| Entry point | `EvaluateInlineTestsWithHarness()` | `EvaluateExpression()` |
| Input | Core binding + test cases | AST expression |
| Approach | Direct Core evaluation | AST→Source reconstruction |
| Pipeline | Bypassed (uses Core directly) | Full pipeline run |
| Source generation | Synthetic Core AST | `fmt.Sprintf("%v", ast)` |
| Result | ✓ Works | ✗ Empty program error |

## Proposed Solutions

### Option A: Direct Core Evaluation (Recommended)

**Make property tests use the same path as inline tests** - evaluate Core AST directly instead of reconstructing source.

```go
// internal/testing/executor.go - NEW METHOD
func (e *Executor) EvaluatePropertyExpression(expr ast.Expr) (eval.Value, error) {
    // 1. Type check the expression
    typeChecker := types.NewTypeChecker(e.coreProg, e.imports)
    typ, err := typeChecker.InferExpr(expr)
    if err != nil {
        return nil, fmt.Errorf("type inference failed: %w", err)
    }

    // 2. Elaborate to Core
    elaborator := elaborate.NewElaborator()
    coreExpr, err := elaborator.ElaborateExpr(expr)
    if err != nil {
        return nil, fmt.Errorf("elaboration failed: %w", err)
    }

    // 3. Evaluate Core expression directly (no source reconstruction)
    effCtx := effects.NewMockEffContext()
    evaluator := eval.NewEvaluator(e.coreProg, effCtx)
    return evaluator.EvalExpr(coreExpr)
}
```

**Pros:**
- Avoids AST→Source conversion entirely
- Consistent with inline test path
- Simpler and more reliable
- No dependency on AST String() methods

**Cons:**
- Requires adding `InferExpr()`, `ElaborateExpr()`, `EvalExpr()` entry points
- More work than fixing String() methods

**Estimated effort:** 4-6 hours

### Option B: Implement Proper AST→Source Unparser

Add a proper unparser that converts AST back to valid AILANG source code.

```go
// internal/unparse/unparse.go - NEW PACKAGE
func Unparse(node ast.Node) string {
    var buf strings.Builder
    unparseLet(&buf, node.(*ast.Let), 0)
    return buf.String()
}

func unparseLet(buf *strings.Builder, let *ast.Let, indent int) {
    buf.WriteString("let ")
    buf.WriteString(let.Name)
    buf.WriteString(" = ")
    unparseExpr(buf, let.Value, indent)
    buf.WriteString(" in\n")
    unparseExpr(buf, let.Body, indent)
}

// ... unparse functions for all AST node types
```

**Pros:**
- Useful for other features (code formatters, refactoring tools)
- Preserves current architecture

**Cons:**
- Large effort (need unparse for ALL AST node types)
- Fragile (must update when AST changes)
- Still doing unnecessary roundtrip (AST→Source→Parse→Core→Eval)

**Estimated effort:** 12-16 hours

### Option C: Fix AST String() Methods

Implement proper `String()` methods on all AST node types.

**Pros:**
- Minimal code changes
- Useful for debugging

**Cons:**
- Same fragility as Option B (must maintain for all node types)
- Still doing unnecessary AST→Source→AST roundtrip
- Doesn't address fundamental architectural issue

**Estimated effort:** 8-10 hours

## Recommendation

**Option A (Direct Core Evaluation)** - eliminates the root cause by avoiding source reconstruction entirely. This is the same approach that made inline tests work reliably.

## Implementation Plan

### Phase 1: Add Expression Entry Points (2 hours)

**File: `internal/types/checker.go`**
- [ ] Add `InferExpr(expr ast.Expr) (types.Type, error)` method
- [ ] Type check expression in isolation (empty initial environment)
- [ ] Tests: literals, binops, lambdas, let bindings

**File: `internal/elaborate/elaborate.go`**
- [ ] Add `ElaborateExpr(expr ast.Expr) (core.Expr, error)` method
- [ ] Convert Surface AST expression to Core
- [ ] Tests: all expression types

**File: `internal/eval/eval.go`**
- [ ] Add `EvalExpr(expr core.Expr) (Value, error)` method
- [ ] Evaluate Core expression directly
- [ ] Tests: arithmetic, closures, conditionals

### Phase 2: Refactor Property Test Execution (1.5 hours)

**File: `internal/testing/executor.go`**
- [ ] Add `EvaluatePropertyExpression(expr ast.Expr)` method
- [ ] Use new entry points (InferExpr → ElaborateExpr → EvalExpr)
- [ ] Include function definitions from source file in Core environment
- [ ] Remove old `EvaluateExpression()` or mark deprecated

**File: `internal/testing/runner.go`**
- [ ] Update `runProperty()` to call `EvaluatePropertyExpression()`
- [ ] No changes to property collection or value generation

### Phase 3: Testing & Validation (1.5 hours)

- [ ] Test with ecommerce demo contracts
- [ ] Test with builtin contracts from std/
- [ ] Verify property tests pass for all supported types (int, float, bool, string, lists)
- [ ] Update documentation in `docs/docs/guides/testing.md`

### Phase 4: Cleanup (1 hour)

- [ ] Remove or deprecate old `EvaluateExpression()` method
- [ ] Add warnings if AST String() methods produce empty results
- [ ] Update CHANGELOG.md
- [ ] Close related GitHub issues

## Test Cases

```ailang
module examples/property_test_demo

-- Test case 1: Simple arithmetic property
pure func add(x: int, y: int) -> int
  requires forall(x: int, y: int) => add(x, y) == add(y, x)  -- Commutativity
  requires forall(x: int) => add(x, 0) == x                   -- Identity
{
  x + y
}

-- Test case 2: Division with contracts
pure func divide(x: int, y: int) -> int
  requires y != 0
  ensures forall(x: int, y: int) => y != 0 ==> x == divide(x, y) * y + (x % y)
{
  x / y
}

-- Test case 3: List length property
pure func listLength(xs: [int]) -> int
  requires forall(xs: [int]) => listLength(xs) >= 0
  ensures forall(xs: [int], x: int) => listLength([x] ++ xs) == 1 + listLength(xs)
{
  match xs {
    [] => 0,
    [_, ...rest] => 1 + listLength(rest)
  }
}
```

**Expected output after fix:**
```bash
$ ailang test examples/property_test_demo.ail

Tests:
  (no inline tests)

Properties:
  ✓ add_property_1 (100 cases, 45.2ms)  # Commutativity
  ✓ add_property_2 (100 cases, 38.1ms)  # Identity
  ✓ divide_property_1 (100 cases, 52.7ms)
  ✓ listLength_property_1 (100 cases, 61.3ms)
  ✓ listLength_property_2 (100 cases, 58.9ms)

Summary: 5 properties passed, 0 failed (500 test cases, 256.2ms)
```

## Success Criteria

- [ ] Property tests evaluate successfully without "empty program" error
- [ ] All ecommerce demo contracts produce passing property tests
- [ ] Test execution time comparable to inline tests (<100ms for 100 cases)
- [ ] No AST→Source reconstruction in property test path
- [ ] Documentation updated in testing guide

## Files to Modify

| File | Changes | LOC |
|------|---------|-----|
| `internal/types/checker.go` | Add `InferExpr()` entry point | ~40 |
| `internal/elaborate/elaborate.go` | Add `ElaborateExpr()` entry point | ~30 |
| `internal/eval/eval.go` | Add `EvalExpr()` entry point | ~25 |
| `internal/testing/executor.go` | Add `EvaluatePropertyExpression()` | ~50 |
| `internal/testing/runner.go` | Update `runProperty()` | ~10 |
| `internal/testing/executor_test.go` | Tests for new methods | ~80 |
| `docs/docs/guides/testing.md` | Update property test docs | ~20 |
| `CHANGELOG.md` | Document fix | ~10 |
| **Total** | | ~265 |

## Related Issues

- **Message ID**: a7dd508c-a2a6-412e-81c5-85c7f5996f13 (demos/ecommerce)
- **User report**: "Property tests fail because AILANG's test framework tries to load _test.ail internally but that module doesn't exist"
  - **NOTE**: This was partially correct - the _test.ail issue was fixed in executor.go:158 by setting IsREPL: true
  - **However**: A deeper issue remains (this bug) - property test source reconstruction produces empty programs

## Related Documents

- [M-VERIFY Implementation](../../implemented/v0_7_0/m-verify-requires-ensures.md) - Contracts feature
- [M-TESTING Integration](../../archive/v0_3_21_m-testing-integration-complete.md) - Test framework
- [Testing Guide](../../../docs/docs/guides/testing.md) - Property test documentation

## Alternatives Considered

### Alternative 1: Skip Property Tests for Now

Mark property tests as "not implemented" and skip them during test execution.

**Pros:**
- Zero implementation effort
- Unblocks users who can use inline tests instead

**Cons:**
- Defeats purpose of M-VERIFY (contracts need property tests)
- Poor user experience (contracts silently ignored)
- Technical debt

**Rejected:** Property tests are core to M-VERIFY value proposition.

### Alternative 2: Use Python/Go Property Test Libraries

Generate property tests as external Python (Hypothesis) or Go (gopter) code.

**Pros:**
- Leverage mature property testing libraries
- More powerful generators/shrinking

**Cons:**
- Cross-language complexity
- Loses AILANG-native testing story
- Hard to integrate with `ailang test` command

**Rejected:** We want AILANG-native property testing.

## Notes

- This bug was discovered during ecommerce demo session
- Inline test fix (IsREPL: true) was applied in same session but didn't fix property tests
- Property tests use fundamentally different execution path than inline tests
- The issue is NOT with module loading - it's with AST→Source reconstruction
- Similar issues may exist elsewhere in codebase (check for other uses of `fmt.Sprintf("%v", ast)`)

## Migration Notes

**For users:**
- No migration needed - property tests will "just work" after fix
- Existing contracts will automatically generate working property tests

**For developers:**
- If you have custom test executors, update to use new `EvaluatePropertyExpression()` method
- Old `EvaluateExpression()` will be deprecated but not removed in v0.7.2

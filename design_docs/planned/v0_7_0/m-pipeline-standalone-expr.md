# M-PIPELINE-STANDALONE-EXPR: Standalone Expression Evaluation

**Status:** Planned
**Target:** v0.7.0
**Priority:** P2 (Low)
**Estimated:** 4 hours
**Dependencies:** None
**Created:** 2026-01-15

## Problem Statement

The AILANG embed API cannot evaluate standalone expressions. While `Engine.Call()` works perfectly for module functions, `Engine.Eval()` fails for simple expressions like `1 + 2`.

**Current behavior:**
```go
engine := embed.New(".")

// Works - calling a module function
result, _ := engine.Call("my/module", "add", 1, 2)

// FAILS - standalone expression
result, err := engine.Eval("1 + 2")
// Error: "empty program: expected at least one item after parse"
```

**Impact:**
- Minor - workaround exists via `Engine.Call()`
- Mostly affects REPL-like use cases and quick prototyping
- Discovered during dashboard dogfooding (GAP-5)

## Root Cause Analysis

The pipeline in `internal/pipeline/pipeline_single.go` requires module context:

1. **Parser expects module declaration** - Line 297-300 rejects programs without items
2. **Type checking needs module** - Uses module-level bindings
3. **Evaluation expects Core program** - Not designed for ad-hoc expressions

```go
// internal/pipeline/pipeline_single.go:297-300
if len(program.Items) == 0 {
    return result, fmt.Errorf("empty program: expected at least one item after parse")
}
```

## Proposed Solution

### Option A: Expression Pipeline (Recommended)

Add a separate lightweight pipeline for expression evaluation:

```go
// internal/pipeline/expr_pipeline.go
func (p *Pipeline) EvalExpr(input string) (eval.Value, types.Type, error) {
    // 1. Parse as expression (not module)
    expr, err := parser.ParseExpr(input)

    // 2. Type check in empty environment
    typ, err := types.InferExpr(expr)

    // 3. Elaborate to Core expression
    coreExpr, err := elaborate.ElaborateExpr(expr)

    // 4. Evaluate
    return eval.EvalExpr(coreExpr)
}
```

**Changes required:**
- `internal/parser/parser.go` - Add `ParseExpr()` entry point (~20 LOC)
- `internal/types/checker.go` - Add `InferExpr()` entry point (~30 LOC)
- `internal/elaborate/elaborate.go` - Add `ElaborateExpr()` entry point (~20 LOC)
- `internal/eval/eval.go` - Add `EvalExpr()` entry point (~20 LOC)
- `internal/pipeline/expr_pipeline.go` - Wire it together (~50 LOC)
- `internal/embed/embed.go` - Update `Eval()` to use new pipeline (~10 LOC)

**Total:** ~150 LOC

**Pros:**
- Clean separation of concerns
- No modification to existing module pipeline
- Simple and focused

**Cons:**
- Some code duplication with module pipeline
- Expressions can't reference module bindings

### Option B: Synthetic Module Wrapper

Wrap expressions in a synthetic module:

```go
func (e *Engine) Eval(expr string) (any, error) {
    // Wrap expression in synthetic module
    syntheticCode := fmt.Sprintf(`
        module __eval__
        let __result__ = %s
        __result__
    `, expr)

    // Use existing pipeline
    return e.evalModule(syntheticCode)
}
```

**Pros:**
- Reuses existing pipeline
- Minimal code changes (~30 LOC)

**Cons:**
- Hacky - creates fake module context
- Error messages reference synthetic code, confusing for users
- Parse errors have wrong line numbers

### Recommendation

**Option A (Expression Pipeline)** - cleaner design, better error messages, and establishes pattern for future REPL improvements.

## Implementation Plan

### Phase 1: Parser Entry Point (1 hour)
- [ ] Add `parser.ParseExpr()` that parses expression without module
- [ ] Handle all expression types (literals, binops, lambdas, records, etc.)
- [ ] Return descriptive errors for statements-in-expression-position

### Phase 2: Type Inference Entry Point (1 hour)
- [ ] Add `types.InferExpr()` for standalone expressions
- [ ] Empty initial environment (no module bindings)
- [ ] Prelude access for basic operations

### Phase 3: Evaluation Entry Point (1 hour)
- [ ] Add `elaborate.ElaborateExpr()` for expression → Core
- [ ] Add `eval.EvalExpr()` for Core expression evaluation
- [ ] Wire into `pipeline.EvalExpr()`

### Phase 4: Integration & Testing (1 hour)
- [ ] Update `embed.Engine.Eval()` to use new pipeline
- [ ] Add tests for various expression types
- [ ] Update GAPS_DISCOVERED.md to mark GAP-5 fixed
- [ ] Un-skip embed tests that rely on Eval()

## Test Cases

```go
func TestEvalExpressions(t *testing.T) {
    engine := embed.New(".")

    // Literals
    assert(engine.Eval("42") == 42)
    assert(engine.Eval("3.14") == 3.14)
    assert(engine.Eval(`"hello"`) == "hello")
    assert(engine.Eval("true") == true)

    // Arithmetic
    assert(engine.Eval("1 + 2") == 3)
    assert(engine.Eval("10 * 5") == 50)
    assert(engine.Eval("2.0 / 4.0") == 0.5)

    // Lambda application
    assert(engine.Eval(`(\x. x + 1)(5)`) == 6)

    // Records
    assert(engine.Eval(`{name: "Alice"}.name`) == "Alice")

    // Lists
    assert(engine.Eval(`[1, 2, 3]`) == []int{1, 2, 3})
}
```

## Success Criteria

- [ ] `engine.Eval("1 + 2")` returns `3` without error
- [ ] All expression types supported (literals, binops, lambdas, records, lists)
- [ ] Error messages show correct line/column for expression
- [ ] Embed tests un-skipped and passing
- [ ] GAPS_DISCOVERED.md updated

## Files to Modify

| File | Changes | LOC |
|------|---------|-----|
| `internal/parser/parser.go` | Add `ParseExpr()` | ~20 |
| `internal/types/checker.go` | Add `InferExpr()` | ~30 |
| `internal/elaborate/elaborate.go` | Add `ElaborateExpr()` | ~20 |
| `internal/eval/eval.go` | Add `EvalExpr()` | ~20 |
| `internal/pipeline/expr_pipeline.go` | New file | ~50 |
| `internal/embed/embed.go` | Update `Eval()` | ~10 |
| `internal/embed/embed_test.go` | Un-skip tests | ~5 |
| **Total** | | ~155 |

## Related Documents

- [GAPS_DISCOVERED.md](../../../internal/dashboard_transforms/GAPS_DISCOVERED.md) - GAP-5 tracking
- [m-gap5-stdlib-repeat-sprint-plan.md](../../implemented/v0_6_4/m-gap5-stdlib-repeat-sprint-plan.md) - Sprint plan format reference

## Notes

- This is a low-priority enhancement discovered during dashboard dogfooding
- The workaround (`Engine.Call()`) covers all practical use cases
- Implementation provides foundation for future REPL improvements

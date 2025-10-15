# v0.3.5: Functional Completeness Sprint

**Status**: 🔄 IN PROGRESS (Started: 2025-10-13)
**Priority**: P0 (CRITICAL - Language Completeness)
**Target Duration**: 5-7 days
**Target Release**: v0.3.5
**Theme**: "From research-grade to practically usable by both humans and AI"

---

## Executive Summary

v0.3.4 has a **solid foundation** (effect system, type inference, modules) but lacks **functional programming completeness**. This sprint fixes the critical P0 blocker (func expressions) and delivers quick-win P1 features that dramatically improve both human UX and AI codegen success rates.

**Goal**: Move M-EVAL success rate from **38.9% → 50%+** by unblocking higher-order functions, improving REPL ergonomics, and fixing numeric handling.

**Strategic Alignment**: This is the inflection point where AILANG transitions from "research-grade prototype" to "practically usable functional language." After this, we can invest in metaprogramming (v0.4.0+).

---

## Problem Statement

### Current Blockers (v0.3.4)

**P0 CRITICAL**: Anonymous function syntax doesn't work
```ailang
-- ❌ FAILS: Parse error
let double = func(x: int) -> int { x * 2 };

-- ✅ WORKAROUND: Must use backslash lambda
let double = \x. x * 2;
```

**Impact**:
- Blocks **15/90 benchmarks** (all higher-order function code)
- Every AI model expects `func(x) -> y { ... }` syntax
- Makes functional programming painful
- Users report confusion ("why doesn't func work here?")

**Evidence from M-EVAL**:
- `higher_order_functions`: 5/5 models fail with parse errors
- `pipeline`: Similar failures due to compose functions
- Root cause: Parser only allows `func` at top-level, not in expressions

### Additional Pain Points

**P1 REPL Limitations**:
```ailang
-- ❌ Can't write recursive lambdas in REPL
λ> let fib = \n. if n < 2 then n else fib(n-1) + fib(n-2) in fib(10)
Error: undefined variable: fib
```

**P1 Numeric Type Friction**:
```ailang
-- ❌ Mixed numeric types fail
λ> 1 + 2.5
Runtime error: builtin add_Int expects Int arguments

-- ✅ Must manually convert
λ> intToFloat(1) + 2.5  -- But these functions don't exist!
```

**P1 Possible Runtime Bug**:
- `list_comprehension` benchmark outputs `0` instead of `220`
- May be modulo operator or fold accumulator issue
- Needs investigation

---

## Strategic Context

### Marginal Impact per LOC Analysis

| Feature | LOC | M-EVAL Impact | Human UX Impact | AI Friendliness | Priority |
|---------|-----|---------------|-----------------|-----------------|----------|
| **func expressions** | ~150 | +10-15 benchmarks | Very High | Very High | **P0** |
| **letrec keyword** | ~200 | +2-3 benchmarks | High | High | **P1** |
| **numeric conversion** | ~100 | +3-5 benchmarks | Medium | High | **P1** |
| **modulo fix** | ~50 | +3-5 benchmarks | Low | Medium | **P1** |
| Record update syntax | ~300 | +0-2 benchmarks | Medium | Medium | P2 (defer) |
| Capability inference | ~500 | +0-1 benchmarks | High | Low | P2 (defer) |

**Analysis**: P0+P1 items deliver **~500 LOC for 20-25 benchmark improvements** (marginal cost: 20-25 LOC per benchmark). Excellent ROI.

### AI-Friendliness Lens

**Why AI models fail on v0.3.4:**
1. **Syntax mismatch**: Every AI training corpus has `func(x) { ... }` patterns
2. **Mental model mismatch**: AIs expect "higher-order functions just work"
3. **Error recovery**: Parse errors harder for AIs to fix than type errors
4. **Token efficiency**: Backslash lambda `\x. ...` unfamiliar, wastes repair tokens

**After v0.3.5:**
- AIs generate correct syntax on first try (↓ repair iterations)
- Functional patterns work naturally (↑ benchmark success)
- REPL exploration improves (↑ interactive debugging)

---

## Sprint Goals

### Success Criteria

**Quantitative**:
- M-EVAL success: 38.9% → **50%+** (35/90 → 45/90 benchmarks)
- Examples passing: 72.7% → **75%+** (48/66 → 50/66)
- Parse error benchmarks: 15 → **0** (all unblocked)
- REPL satisfaction: Enable recursive lambdas

**Qualitative**:
- "Functional programming feels natural"
- "AI-generated code works on first try"
- "REPL is great for exploration"

### Non-Goals (Explicitly Deferred)

- ❌ Record update syntax `{r | field: v}` → v0.3.6
- ❌ Capability auto-inference → v0.3.6
- ❌ Error propagation `?` → v0.3.7
- ❌ List comprehensions → v0.3.7
- ❌ Typed quasiquotes → v0.4.0+
- ❌ CSP concurrency → v0.4.0+

**Rationale**: Focus on **language completeness** (P0/P1) before **convenience features** (P2/P3) or **advanced features** (P4).

---

## Implementation Plan

### Phase 1: P0 - Anonymous Function Syntax (Day 1)

**Goal**: Make `func(x: int) -> int { x * 2 }` work in expression position.

#### Design

**Current State**:
```go
// internal/parser/parser.go
func (p *Parser) parseExpression() ast.Expr {
    switch p.curToken.Type {
    case token.FUNC:
        return nil, fmt.Errorf("func not allowed in expression")
    // ...
    }
}
```

**Proposed**:
```go
// Add FuncLit AST node
type FuncLit struct {
    Params []Param
    ReturnType Type
    Body Expr
    Pos Pos
}

// Parser: Allow FUNC token in expression position
case token.FUNC:
    return p.parseFuncLit()  // Similar to parseFuncDecl but returns expression

// Elaboration: Desugar FuncLit -> core.Lambda
func (e *Elaborator) elaborate(node ast.Expr) core.CoreExpr {
    case *ast.FuncLit:
        return e.elaborateFuncLit(node)  // Extract params, elaborate body
}
```

#### Implementation Steps

1. **Add AST node** (`internal/ast/ast.go`, ~30 LOC)
   - `type FuncLit struct { ... }`
   - Implement `exprNode()`, `Position()` methods

2. **Parser changes** (`internal/parser/parser.go`, ~80 LOC)
   - Modify `parseExpression()` to handle `token.FUNC`
   - Implement `parseFuncLit()` - similar to `parseFuncDecl()`
   - Handle type annotations, multiple params, effect annotations

3. **Elaboration** (`internal/elaborate/elaborate.go`, ~40 LOC)
   - Add case for `*ast.FuncLit`
   - Desugar to `core.Lambda` (extract param names/types, elaborate body)
   - Preserve type annotations for type checker

4. **Type checking** (likely works automatically)
   - Type checker already handles `core.Lambda`
   - May need to wire through type annotations

5. **Tests** (`internal/parser/parser_test.go`, ~50 LOC)
   - Parse `func(x: int) -> int { x + 1 }`
   - Parse multi-param: `func(x: int, y: int) -> int { x + y }`
   - Parse with effects: `func() -> () ! {IO} { println("hi") }`
   - Integration test: higher_order_functions example

#### Acceptance Criteria

- [ ] `let f = func(x: int) -> int { x * 2 } in f(5)` evaluates to `10`
- [ ] Multi-param works: `func(x: int, y: int) -> int { x + y }`
- [ ] Type annotations optional: `func(x, y) { x + y }` (inferred)
- [ ] Effects work: `func() -> () ! {IO} { println("hi") }`
- [ ] `higher_order_functions` benchmark passes for all 5 models
- [ ] All existing tests still pass

**Estimated**: 6-8 hours, ~200 LOC

---

### Phase 2: P1a - Add `letrec` Keyword (Day 2 Morning)

**Goal**: Enable recursive lambdas in REPL and module code.

#### Design

**Syntax**:
```ailang
-- Single recursive binding
letrec fib = \n. if n < 2 then n else fib(n-1) + fib(n-2) in fib(10)

-- With type annotation
letrec sum: [int] -> int = \xs. match xs {
  [] => 0,
  [x, ...rest] => x + sum(rest)
} in sum([1,2,3])
```

**Implementation** (see [20251013_letrec_surface_syntax.md](20251013_letrec_surface_syntax.md)):

1. **Lexer** - Add `LETREC` token, add to keywords map (~10 LOC)
2. **Parser** - Add `ast.LetRec` node, parse `letrec name = value in body` (~50 LOC)
3. **Elaborate** - Lower to existing `core.LetRec` (~20 LOC)
4. **Tests** - Fibonacci, factorial, list sum (~30 LOC)

#### Acceptance Criteria

- [ ] `letrec fib = \n. ... in fib(10)` works in REPL
- [ ] Type annotations work
- [ ] All existing letrec-using code still works
- [ ] REPL example in docs updated

**Estimated**: 3-4 hours, ~200 LOC

---

### Phase 3: P1b - Numeric Conversion Builtins (Day 2 Afternoon)

**Goal**: Add explicit conversion functions to unblock mixed numeric types.

#### Design

**API** (see [20251013_numeric_coercion.md](20251013_numeric_coercion.md) Option 1):
```ailang
-- stdlib/std/prelude.ail
export func intToFloat(x: int) -> float { $builtin.intToFloat(x) }
export func floatToInt(x: float) -> int { $builtin.floatToInt(x) }

-- Usage
λ> intToFloat(1) + 2.5
3.5 :: Float
```

#### Implementation

1. **Builtins** (`internal/builtins/registry.go`, ~40 LOC)
   - Register `intToFloat` and `floatToInt`
   - Type signatures: `int -> float` and `float -> int`

2. **Evaluator** (`internal/eval/builtins.go`, ~30 LOC)
   - Implement runtime conversion
   - `intToFloat`: Convert Go int64 → float64
   - `floatToInt`: Truncate float64 → int64

3. **Stdlib** (`stdlib/std/prelude.ail`, ~10 LOC)
   - Export wrapper functions
   - Document usage

4. **Tests** (~20 LOC)
   - `intToFloat(1)` returns `1.0`
   - `floatToInt(3.7)` returns `3` (truncates)
   - Mixed arithmetic: `intToFloat(1) + 2.5` → `3.5`

5. **Documentation** (`prompts/v0.3.0.md`, ~20 LOC)
   - Teach AIs to use these functions
   - Example: "For mixed numeric types, use intToFloat()"

#### Acceptance Criteria

- [ ] `intToFloat(1) + 2.5` evaluates to `3.5`
- [ ] `floatToInt(3.9)` evaluates to `3`
- [ ] Functions exported from std/prelude
- [ ] Documentation updated

**Estimated**: 2-3 hours, ~120 LOC

---

### Phase 4: P1c - Debug Modulo/Logic Issues (Day 3)

**Goal**: Investigate why `list_comprehension` outputs `0` instead of `220`.

#### Investigation Steps

1. **Reproduce locally**:
   ```bash
   # Get AI-generated code from eval results
   cat eval_results/*list_comprehension*claude*.json | jq -r '.code' > test_list.ail

   # Run it
   ailang run --caps IO --entry main test_list.ail
   ```

2. **Debug hypotheses**:
   - Modulo operator: Test `n % 2 == 0` for evens
   - Fold accumulator: Check if fold(add, 0, xs) works correctly
   - Type inference: Verify Int vs Float types
   - List construction: Verify ADT constructors work

3. **Create minimal repro**:
   ```ailang
   -- Test modulo
   export func testMod() -> () ! {IO} {
     println(show(4 % 2));  -- Should be 0
     println(show(5 % 2))   -- Should be 1
   }

   -- Test fold
   export func testFold() -> () ! {IO} {
     let nums = Cons(2, Cons(4, Cons(6, Nil)));
     let sum = fold(\acc. \n. acc + n, 0, nums);
     println(show(sum))  -- Should be 12
   }
   ```

4. **Fix root cause**:
   - If modulo bug: Fix in `internal/eval/builtins.go`
   - If fold bug: Fix in generated code or stdlib
   - If type bug: Fix in type checker

5. **Verify fix**:
   ```bash
   ailang eval-validate list_comprehension
   ```

#### Acceptance Criteria

- [ ] `list_comprehension` outputs `220` for all models
- [ ] Root cause identified and documented
- [ ] Regression test added
- [ ] Related benchmarks re-validated

**Estimated**: 2-4 hours (depends on bug complexity)

---

### Phase 5: Documentation & Examples (Day 4)

**Goal**: Update all user-facing docs to teach new features.

#### Tasks

1. **Update Teaching Prompt** (`prompts/v0.3.0.md`)
   - Add func expression syntax examples
   - Add letrec examples
   - Add numeric conversion examples
   - Update "Common Mistakes" section

2. **Update Playground** (`docs/docs/playground.mdx`)
   - Add func expression example
   - Add letrec example
   - Remove broken examples

3. **Create Examples** (`examples/`)
   - `examples/func_expressions.ail` - Higher-order functions
   - `examples/letrec_recursion.ail` - Recursive lambdas
   - `examples/numeric_conversion.ail` - Type conversions

4. **Update README** (`README.md`)
   - Update feature checklist
   - Update example counts
   - Update M-EVAL success rate

5. **Update CHANGELOG** (`CHANGELOG.md`)
   - Document v0.3.5 changes
   - Include metrics (LOC, benchmarks fixed)
   - Migration notes if needed

#### Acceptance Criteria

- [ ] All docs mention func expressions
- [ ] Teaching prompt has 3+ new examples
- [ ] Playground examples all work
- [ ] 3 new example files created and verified
- [ ] README metrics updated

**Estimated**: 4-6 hours

---

### Phase 6: M-EVAL Validation (Day 5)

**Goal**: Validate improvements with comprehensive eval run.

#### Process

1. **Baseline current state**:
   ```bash
   make eval-suite
   ailang eval-summary eval_results/ > v0.3.4-final.jsonl
   ```

2. **Run full eval**:
   ```bash
   make eval-suite  # After all changes
   ailang eval-summary eval_results/ > v0.3.5.jsonl
   ```

3. **Generate comparison report**:
   ```bash
   ailang eval-compare v0.3.4-final v0.3.5 > v0.3.5-report.md
   ```

4. **Analyze results**:
   - Count new passing benchmarks
   - Identify remaining failures
   - Document any regressions
   - Calculate success rate

5. **Validate specific benchmarks**:
   ```bash
   ailang eval-validate higher_order_functions
   ailang eval-validate list_comprehension
   ailang eval-validate pipeline
   ```

6. **Create release notes**:
   - Summarize improvements
   - List breaking changes (if any)
   - Include benchmark metrics

#### Success Targets

- [ ] M-EVAL success: 50%+ (45/90+ benchmarks)
- [ ] `higher_order_functions`: 5/5 models pass
- [ ] `list_comprehension`: 5/5 models pass
- [ ] `pipeline`: 3/5+ models pass
- [ ] No regressions in previously passing benchmarks

**Estimated**: 4-6 hours (mostly waiting for eval runs)

---

## Timeline & Effort

### Optimistic (5 days)

| Day | Phase | Hours | Cumulative |
|-----|-------|-------|------------|
| 1 | P0: func expressions | 8 | 8 |
| 2 | P1a: letrec (4h) + P1b: numeric (3h) | 7 | 15 |
| 3 | P1c: modulo debug | 4 | 19 |
| 4 | Documentation | 6 | 25 |
| 5 | M-EVAL validation | 6 | 31 |

**Total**: 31 hours (~4 full days)

### Realistic (7 days)

| Day | Phase | Hours | Cumulative |
|-----|-------|-------|------------|
| 1-2 | P0: func expressions | 12 | 12 |
| 3 | P1a: letrec | 6 | 18 |
| 4 | P1b: numeric + P1c: debug | 8 | 26 |
| 5 | Documentation | 8 | 34 |
| 6 | M-EVAL validation | 8 | 42 |
| 7 | Buffer for issues | 4 | 46 |

**Total**: 46 hours (~6 working days)

---

## Metrics & KPIs

### Before (v0.3.4)

| Metric | Value |
|--------|-------|
| Examples passing | 48/66 (72.7%) |
| M-EVAL success | 35/90 (38.9%) |
| Parse error benchmarks | 15 |
| Logic error benchmarks | 35 |
| Compile error benchmarks | 5 |
| REPL recursive lambdas | ❌ Not supported |
| Mixed numeric types | ❌ Requires manual conversion |

### After (v0.3.5 Target)

| Metric | Value | Change |
|--------|-------|--------|
| Examples passing | 50+/66 (75%+) | +2-4 examples |
| M-EVAL success | 45+/90 (50%+) | +10-15 benchmarks |
| Parse error benchmarks | 0 | -15 ✅ |
| Logic error benchmarks | 30-32 | -3-5 (modulo fix) |
| Compile error benchmarks | 3-5 | ~0 |
| REPL recursive lambdas | ✅ Supported | NEW |
| Mixed numeric types | ✅ Explicit conversion | NEW |

### Code Size

| Component | LOC Added | LOC Changed | Total |
|-----------|-----------|-------------|-------|
| Parser | +150 | +50 | 200 |
| Elaborator | +60 | +20 | 80 |
| Evaluator | +30 | +20 | 50 |
| Builtins | +70 | +10 | 80 |
| Stdlib | +20 | +10 | 30 |
| Tests | +130 | +30 | 160 |
| Docs | +100 | +50 | 150 |
| **Total** | **~560** | **~190** | **~750** |

---

## Risk Assessment

### High Risk

**Risk**: func expression syntax conflicts with existing parser
- **Mitigation**: Thorough testing, isolated parser changes
- **Contingency**: Revert to backslash lambda only, update docs

**Risk**: Breaking changes to existing code
- **Mitigation**: Run all examples, verify no regressions
- **Contingency**: Add compatibility mode, migration guide

### Medium Risk

**Risk**: M-EVAL improvements don't reach 50%
- **Mitigation**: Incremental validation per phase
- **Contingency**: Acceptable if >45%, analyze remaining failures

**Risk**: Modulo bug is deep in evaluator
- **Mitigation**: Budget extra time for debugging
- **Contingency**: Document as known issue, fix in v0.3.6

### Low Risk

**Risk**: letrec implementation issues
- **Mitigation**: core.LetRec already exists and works
- **Contingency**: Defer to v0.3.6 if blocking

**Risk**: Documentation lag
- **Mitigation**: Write docs alongside code
- **Contingency**: Ship code, update docs post-release

---

## Dependencies

### Upstream (Must Complete First)

- ✅ v0.3.4 release (complete)
- ✅ M-EVAL-LOOP v2.0 (complete)
- ✅ Design audits (complete)

### Downstream (Enabled by This Sprint)

- → v0.3.6: Record update syntax (P2)
- → v0.3.6: Capability inference (P2)
- → v0.3.7: Error propagation `?` (P3)
- → v0.4.0: Typed quasiquotes (P4)
- → v0.4.0: CSP concurrency (P4)

---

## Post-Sprint Review

### Questions to Answer

1. Did we hit 50% M-EVAL success? If not, why?
2. Which benchmarks are still failing? What patterns?
3. Are there new language gaps discovered?
4. How do users (human + AI) report the UX?
5. Any technical debt introduced?

### Retro Topics

- Was the P0/P1 prioritization correct?
- Should we have included P2 items?
- Did we defer the right features?
- Is the strategic direction still correct?

### Next Steps (v0.3.6 Planning)

Based on v0.3.5 results, decide:
- Continue with P2 ergonomic features?
- Address new M-EVAL failures?
- Start v0.4.0 planning (metaprogramming)?

---

## Alignment with Vision

### Original Vision Progress

| Principle | v0.3.4 | v0.3.5 | v0.4.0 Target |
|-----------|--------|--------|---------------|
| Explicit Effects | ✅ Complete | ✅ Complete | ✅ Complete |
| Typed Expressions | ✅ Complete | ✅ Complete | ✅ Complete |
| Type-Safe Metaprogramming | ❌ | ❌ | ✅ Quasiquotes |
| Deterministic Execution | ⚠️ Partial | ⚠️ Partial | ✅ Complete |
| CSP Concurrency | ❌ | ❌ | ✅ Complete |

### Strategic Positioning

**v0.3.x** = "Practical functional language with effect tracking"
- Target: Real programs, AI-assisted coding
- Philosophy: Safety + Ergonomics + Explicitness
- **v0.3.5 delivers**: Functional completeness

**v0.4.x** = "AI-first metaprogramming language"
- Target: Code generation, training data, deterministic execution
- Philosophy: Machine-decidability + Reproducibility
- **Requires v0.3.5 foundation**: Stable core before advanced features

---

## Conclusion

v0.3.5 is the **inflection point** where AILANG becomes **practically usable**. After this sprint:

✅ Higher-order functions work naturally
✅ REPL enables full exploration
✅ AI models generate correct code
✅ Numeric handling has explicit paths
✅ M-EVAL success demonstrates AI-friendliness

**Next milestone**: v0.3.6-v0.3.7 polish (P2/P3), then v0.4.0 metaprogramming layer.

**Strategic bet**: Get basics right first, then invest in advanced features. This sprint validates that bet.

---

**Approval Required**: Review priorities, timeline, and resource allocation before starting implementation.

**Start Trigger**: Approval + commit to focused 5-7 day sprint + pause other features.

**Success Signal**: M-EVAL ≥50%, community feedback "functional programming feels natural", AI codegen success.

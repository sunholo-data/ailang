# M-EVAL-SHORT-CIRCUIT-BOOL: Short-Circuit Evaluation for `&&` and `||`

**Status**: Implemented (v0.11.3)
**Target**: v0.11.3 (hotfix)
**Priority**: **P0 — Correctness bug** (silent runtime crashes from idiomatic code)
**Estimated**: 0.5–1 day (~4 hours implementation + testing)
**Dependencies**: None
**Bug Report**: ailang-parse msg `ce6e078e` (2026-04-14, tex_parser.ail development)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Evaluation order becomes more predictable (LHS always before RHS); no nondeterminism |
| A2: Replayability | 0 | Traces unchanged in shape; RHS may be absent from trace when short-circuited |
| A3: Effect Legibility | +1 | Effectful RHS no longer runs when LHS gates it — matches written intent |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Z3 encoding of `&&`/`||` already assumes short-circuit semantics; evaluator will match |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | AI agents overwhelmingly expect short-circuit from FP languages; surprise bugs are a major DX hazard |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Avoiding needless RHS evaluation is a direct cost win on hot paths |
| A10: Composability | +1 | Idiomatic guard patterns (`len>0 && head(xs) == x`) compose correctly |
| A11: Structured Failure | +1 | Removes a whole class of spurious panics from bounds-checked indexing guarded by `&&` |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — evaluation order strictly LHS-then-RHS
- [x] A3 (Effects): Effect ordering becomes *more* legible, not less
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Aligns with universal FP expectation; removes footgun

## Problem Statement

AILANG's `&&` and `||` operators **do not short-circuit**. Both operands are fully evaluated
before the boolean operation is applied. This is a correctness bug that silently miscompiles
idiomatic guard code and causes runtime panics.

### Concrete Repro (from msg ce6e078e)

In `tex_parser.ail` (~570 LoC, ailang-parse project):

```ailang
-- Guard: only look back if i > 0
if i > 0 && charAt(s, i-1) == "\\" then ...
```

**Expected** (standard FP / JS / Python semantics): if `i == 0`, RHS is not evaluated.
**Actual**: both sides evaluate; `charAt(s, -1)` panics at runtime with
`index -1 out of bounds for string of length N`.

Similarly:

```ailang
if cmdEnd < length(s) && charAt(s, cmdEnd) == "{" then ...
-- Crashes at cmdEnd == length(s) with "index N out of bounds for string of length N"
```

The user had to rewrite every such guard as a nested `if`:

```ailang
if i > 0 then
  if charAt(s, i-1) == "\\" then ... else ...
else ...
```

### Why This Is Urgent

1. **The type-checker accepts it.** There is no warning, no error, no hint. The program
   compiles, runs, and crashes on the first input that exercises the guard's false branch.
2. **Every FP language on earth short-circuits `&&`/`||`.** Haskell, OCaml, Elm, F#, Scala,
   Rust, Go, JS, Python — all of them. AILANG is the outlier.
3. **AI code synthesis overwhelmingly produces guard patterns.** The entire codegen corpus
   (training data for LLMs targeting AILANG) assumes short-circuit. Models will keep
   generating broken guards until this is fixed.
4. **Silent miscompilation of a fundamental operator undermines trust in the language.**
   A 570-line file that type-checks cleanly on first submission but crashes at runtime is
   a worst-case DX outcome — *worse* than a type error, because the error surfaces far
   from the cause.

### Root Cause

`internal/eval/eval_operations.go:405-418` handles `&&`/`||` in `applyBinOp`, but
`applyBinOp` is called **after** both arguments are already evaluated. The evaluator
reaches this point via the standard `App` path (`intrinsic` → `applyBinOp`), which
eagerly evaluates all arguments before dispatch.

```go
// internal/eval/eval_operations.go:405
if op == "&&" || op == "||" {
    lBool, lOk := left.(*BoolValue)   // already evaluated
    rBool, rOk := right.(*BoolValue)  // already evaluated — too late
    ...
}
```

The fix must intercept `&&`/`||` **before** RHS evaluation, likely in the Core
evaluator's `App`/`Intrinsic` path, so the RHS is a thunk evaluated only if needed.

## Goals

**Primary Goal:** `&&` and `||` short-circuit: RHS is not evaluated if LHS determines the result.

**Success Metrics:**
- `(i > 0 && charAt(s, i-1) == "x")` with `i == 0` returns `false` and does not panic
- `(i == 0 || charAt(s, i) == "x")` with `i == 0` returns `true` and does not panic
- No regression on any existing example or test
- Tex parser from msg ce6e078e can use single `&&` guards (no nested-if workaround)
- New `examples/short_circuit.ail` demonstrating guarded indexing, guarded division, etc.
- Z3/SMT backend `&&`/`||` encoding remains consistent (already short-circuit in SMT)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Intercept at Core evaluator vs at elaboration (desugar to `if`) | Evaluator intercept is surgical and local; desugar-to-if is more invasive but guarantees all downstream phases (bytecode VM, SMT, codegen) see it for free | human | design | high |
| Bytecode VM: add `JumpIfFalse`/`JumpIfTrue` or inline via `IfElse` lowering | Determines whether bytecode ISA grows or whether `&&`/`||` lower to branch instructions | human | compile | med |
| Whether to deprecate the `&&`/`||` intrinsic path entirely or keep for dictionary-based fallback | Simpler evaluator vs backwards-compat with existing dict-elaborated code | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Approach**: Desugar `a && b` → `if a then b else false` (and `a || b` → `if a then true else b`) at **elaboration time**, NOT evaluator time. Rationale: every downstream phase (evaluator, bytecode VM, SMT, future codegen) gets correct semantics for free, matching the canonical FP encoding.
- [ ] Elaboration must preserve the `Bool` type of the RHS (type-check before desugar).
- [ ] `applyBinOp`'s `&&`/`||` case remains as a fail-fast guard: reaching it indicates a missed desugar (panic in `DEBUG_STRICT=1`, log-and-evaluate otherwise).
- [ ] Tests must cover: effectful RHS skipped, short-circuit traces recorded correctly, SMT proofs unchanged.

## Solution Design

### Overview

**Desugar `&&`/`||` at elaboration time** into `if`-expressions. The evaluator, bytecode VM,
SMT backend, and any future codegen all inherit correct semantics from the existing `if`
lowering — which already short-circuits correctly.

```
BEFORE (elaborated Core):
  App(BinOp("&&"), [lhs, rhs])

AFTER (elaborated Core):
  If(lhs, rhs, BoolLit(false))     -- for &&
  If(lhs, BoolLit(true), rhs)      -- for ||
```

### Architecture

**Current flow (broken):**
```
Parser → AST BinOp("&&", a, b)
     → Elaborator → Core App(BinOp("&&"), [a, b])
     → Evaluator eagerly evaluates both a and b
     → applyBinOp("&&", aVal, bVal) → BoolValue
```

**Proposed flow:**
```
Parser → AST BinOp("&&", a, b)
     → Elaborator (new pass or inline in existing lowering):
       if op ∈ {"&&", "||"} && both args typed as Bool:
         emit If(lhs, rhs, BoolLit(false))     for &&
         emit If(lhs, BoolLit(true), rhs)      for ||
     → Evaluator follows standard If semantics → short-circuits naturally
```

### Components

1. **Elaboration desugar** (`internal/elaborate/` — likely a new small pass or hook in
   existing BinOp lowering, ~40 LOC)
   - Match `ast.BinaryOp` nodes where `Op == "&&"` or `Op == "||"`
   - After type-checking confirms both operands are `Bool`, rewrite to `core.If`
   - Preserve source position and type annotation

2. **Evaluator fail-fast guard** (`internal/eval/eval_operations.go`, ~10 LOC change)
   - Keep existing `&&`/`||` branch in `applyBinOp`
   - Add: `if os.Getenv("DEBUG_STRICT") == "1" { panic("&&/|| should have been desugared to If") }`
   - Otherwise: log a one-time warning and evaluate eagerly (preserves behavior during
     migration; we expect this path to be cold after the elaborator fix)

3. **Bytecode VM** (`internal/bytecode/` — verify, no change expected)
   - `If` already lowers to conditional jumps; short-circuit is free
   - Add a bytecode test: compile `a && sideEffect(b)` where LHS false → verify
     `sideEffect` does not execute

4. **SMT backend** — no change (already emits `(and a b)` which Z3 treats as short-circuit
   semantically; verify tests still pass)

5. **Examples & regression tests**
   - `examples/short_circuit_and.ail` — guarded indexing pattern
   - `examples/short_circuit_or.ail` — null-check pattern
   - `internal/eval/short_circuit_test.go` — effect ordering, no-panic guards,
     nested `&&`/`||` chains

### Implementation Plan

**Phase 1: Elaboration desugar** (~2 hours)
- [ ] Locate where `ast.BinaryOp("&&" | "||")` is lowered to Core in `internal/elaborate/`
- [ ] Rewrite to `core.If` after type check
- [ ] Preserve position, attach `Bool` type annotation
- [ ] Unit test: elaborate `a && b` → assert Core shape is `If(a, b, False)`

**Phase 2: Tests + examples** (~1 hour)
- [ ] `examples/short_circuit_and.ail` — guarded `charAt`
- [ ] `examples/short_circuit_or.ail` — guarded `head`
- [ ] `internal/eval/short_circuit_test.go` — effect ordering, `make test` regression
- [ ] Regression: run the `tex_parser.ail` pattern from msg ce6e078e end-to-end

**Phase 3: Evaluator guard + cleanup** (~1 hour)
- [ ] Add `DEBUG_STRICT=1` panic in `applyBinOp` `&&`/`||` branch
- [ ] Verify no existing test triggers the panic
- [ ] CHANGELOG entry under **Bug Fixes (correctness)**

### Files to Modify/Create

**New:**
- `examples/short_circuit_and.ail` (~15 LOC)
- `examples/short_circuit_or.ail` (~15 LOC)
- `internal/eval/short_circuit_test.go` (~80 LOC)

**Modified:**
- `internal/elaborate/*` — add desugar pass for `&&`/`||` (~40 LOC)
- `internal/eval/eval_operations.go` — fail-fast guard (~10 LOC)
- `CHANGELOG.md` — bug fix entry

**Estimated total: ~160 LOC (mostly tests)**

## Examples

### Example 1: Guarded Indexing (the reported bug)

```ailang
-- BEFORE (required nested if as workaround):
func lookBehind(s: string, i: int) -> bool {
  if i > 0 then
    if charAt(s, i-1) == "\\" then true else false
  else false
}

-- AFTER (natural single-line guard):
func lookBehind(s: string, i: int) -> bool =
  i > 0 && charAt(s, i-1) == "\\"
```

### Example 2: Guarded Division

```ailang
-- Works correctly after fix; currently divides-by-zero even when guarded:
func safeDiv(a: int, b: int) -> bool =
  b != 0 && (a / b) > 0
```

### Example 3: Effect Ordering

```ailang
-- After fix: sideEffect() only runs if check() is true.
-- Before fix: sideEffect() always runs.
if check() && sideEffect() then ... else ...
```

## Success Criteria

- [ ] `a && b` elaborates to `If(a, b, False)`; `a || b` to `If(a, True, b)`
- [ ] Reported bug from msg ce6e078e: `i > 0 && charAt(s, i-1) == "\\"` with `i == 0` returns `false`, no panic
- [ ] Effectful RHS is **not** executed when LHS short-circuits (verified by effect trace)
- [ ] All existing tests pass (`make test`, `make verify-examples`)
- [ ] `DEBUG_STRICT=1` verifies no codepath reaches the old `applyBinOp` `&&`/`||` branch
- [ ] Bytecode VM runs the new examples identically to the tree-walking evaluator
- [ ] CHANGELOG entry under **v0.11.3 → Bug Fixes (correctness)**
- [ ] ailang-parse agent can remove the nested-`if` workarounds in `tex_parser.ail`

## Testing Strategy

**Unit tests:**
- Elaboration: `&&`/`||` → `If` shape, position preserved, type preserved
- Evaluator: LHS `false` short-circuits RHS; LHS `true` short-circuits RHS (for `||`)
- Effect trace: RHS absent from trace when short-circuited

**Integration tests:**
- `examples/short_circuit_*.ail` via `make verify-examples`
- Regression: bytecode VM mirrors tree-walk behavior
- SMT: contracts using `&&`/`||` still verify (no regression)

**End-to-end:**
- Port the `tex_parser.ail` guard patterns from msg ce6e078e, confirm no panic

## Deferred Decisions

- Whether to warn/lint when an effectful expression appears as RHS of `&&`/`||`
  (user's msg suggested "typechecker warns when RHS uses a potentially-unsafe index") —
  nice-to-have but *this design doc focuses on making the common case correct*.
  A dedicated linter pass can come later.
- Whether to add `andThen`/`orElse` combinator functions (lazy via closure) for cases
  where users want explicit short-circuit without the syntactic sugar — out of scope.

## Non-Goals

- Linting for "dangerous" RHS (bounds checks, division) — separate future work
- Lazy evaluation in general — this is *specifically* `&&` and `||`
- Changing `&` or `|` (bitwise operators, see m-bitwise-operators.md)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Desugar happens before type-check completes → wrong shape | Medium | Place pass AFTER type-check / dictionary elaboration, confirmed both operands typed `Bool` |
| Users relied on the eager-eval bug (unlikely but possible) | Low | Explicit CHANGELOG note; eager-eval is not documented anywhere |
| SMT proofs silently change due to different Core shape | Low | SMT emits `(and a b)` from either shape; add regression test on existing verified contracts |
| Bytecode VM regression | Low | `If` lowering is already exercised by every example; add specific test |

## Related Documents

**Implemented (prior art):**
- [design_docs/implemented/v0_11_0/m-bytecode-vm.md](design_docs/implemented/v0_11_0/m-bytecode-vm.md) — `If` lowering is already short-circuit
- [design_docs/implemented/v0_6_2/m-pattern-guards.md](design_docs/implemented/v0_6_2/m-pattern-guards.md) — Pattern guards also benefit from short-circuit semantics

**Related:**
- [design_docs/planned/v0_11_0/m-bitwise-operators.md](design_docs/planned/v0_11_0/m-bitwise-operators.md) — `&`/`|` are separate operators, unchanged

## References

- Originating message: ailang-parse msg `ce6e078e` (`ailang messages read ce6e078e-fdd1-4f64-aff4-7c7a0041ba67`)
- Evaluator bug location: `internal/eval/eval_operations.go:405-418`
- Standard FP short-circuit semantics: Haskell `&&`/`||`, OCaml `&&`/`||`, Elm `&&`/`||`

---

**Document created**: 2026-04-14
**Last updated**: 2026-04-14

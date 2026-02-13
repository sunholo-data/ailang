# M-SMT-BOUNDED-RECURSION: Bounded Unrolling for Recursive Function Verification

**Status**: Planned
**Target**: v0.8.0
**Priority**: P1 - Medium
**Estimated**: 2 days (~16 hours)
**Dependencies**: M-SMT-RECORDS (implemented), M-SMT-STRINGS (implemented), M-SMT-LISTS (implemented)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Bounded unrolling is fully deterministic — same depth produces identical SMT output |
| A2: Replayability | +1 | Verification results are reproducible given same depth parameter |
| A3: Effect Legibility | 0 | Only applies to pure functions (already enforced by fragment checker) |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Core axiom — extends local bounded verification to recursive functions |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Enables automated verification of recursive functions without human intervention |
| A8: Minimal Syntax | +1 | No new syntax — only a CLI flag (`--verify-recursive-depth N`) |
| A9: Cost Visibility | +1 | Depth parameter makes solver cost explicit; output labels "bounded: depth N" |
| A10: Composability | +1 | Composes with existing contract system, cross-function calls, and all SMT-encodable types |
| A11: Structured Failure | 0 | Verification results already structured |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): Bounded unrolling is fully deterministic
- [x] A3 (Effects): Only verifies pure functions
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Designed for automated machine verification

## Problem Statement

`ailang verify` currently rejects **all** recursive functions with `RejectRecursive: "Function is recursive — SMT verification requires non-recursive functions"`. This is the largest remaining gap in the decidable fragment.

**Current State:**
- Recursive functions are the #1 rejection reason for user-written contracts
- Functions like `factorial`, `sumTo`, `fib`, and recursive list operations cannot be verified at all
- The fragment checker in `IsSMTEncodable()` permanently rejects any function where `containsRef(body, funcName)` returns true
- Users have no workaround — they must restructure code to avoid recursion, which is often unnatural

**Impact:**
- Any developer writing contracts on recursive functions is blocked
- Standard library functions that are naturally recursive (e.g., list operations) cannot have verified contracts
- Limits AILANG's verification story to a small subset of programs

## Research: Bounded Unfolding Approaches

Three established approaches exist in formal verification:

| System | Mechanism | Complexity | Fit for AILANG |
|--------|-----------|------------|----------------|
| **F*** | Fuel sort (`ZFuel/SFuel`) as quantifier pattern guards | High — requires universally quantified axioms + E-matching | Overkill |
| **Liquid Haskell** | Refinement reflection + PLE (selective unfolding) | Very High — requires refinement type infrastructure | Wrong paradigm |
| **Dafny** | Simple `define-fun` chains at fixed depth | Low — reuses existing infrastructure | **Best fit** |

### F* Fuel Pattern
F* introduces a `Fuel` sort with constructors `ZFuel` (zero) and `SFuel` (successor). Recursive function definitions become universally quantified axioms guarded by fuel patterns. Z3's E-matching triggers unfolding when fuel is available. Default initial fuel is 2, max fuel is 8. In practice, F* encourages fuel 0 + explicit lemma calls.

**Why not for AILANG**: Requires universally quantified axioms and E-matching, which are outside AILANG's quantifier-free decidable fragment. The complexity is disproportionate to the benefit.

### Liquid Haskell PLE
Liquid Haskell uses refinement reflection to lift Haskell functions into the refinement logic, then PLE (Proof by Logical Evaluation) selectively unfolds function definitions at call sites. Functions remain uninterpreted by default; unfolding happens through the type system.

**Why not for AILANG**: Requires a refinement type infrastructure that AILANG doesn't have. The paradigm assumes a different type system architecture.

### Dafny Simple Unrolling (Selected)
Dafny generates `define-fun` chains at a fixed depth. Default depth is 2 for assertions. The `{:fuel n}` attribute overrides per-function. Higher depths have `opaque`/`reveal` mechanisms.

**Why this fits AILANG**:
- AILANG already has `define-fun` infrastructure for cross-function calls (`callee_resolver.go`)
- No quantifiers needed — stays within the decidable fragment
- Simple, deterministic, and machine-friendly (A7)
- Matches AILANG's "bounded verification" axiom (A5)

## Goals

**Primary Goal:** Enable SMT verification of recursive functions via bounded unrolling at configurable depth.

**Success Metrics:**
- `factorial(n)` with `requires { n >= 0 }, ensures { result >= 1 }` verifies at depth 2
- `sumTo(n)` with `requires { n >= 0 }, ensures { result >= 0 }` verifies at depth 2
- `fib(n)` with `requires { n >= 0 }, ensures { result >= 0 }` verifies at depth 3
- All existing non-recursive verification examples remain unchanged
- Bounded results clearly labeled in output: `VERIFIED (bounded: depth N)`

## Solution Design

### Overview

For a recursive function `f(n)` at depth N, generate N+1 `define-fun` declarations:

```smt2
(declare-fun f_0 (Int) Int)           ; Level 0: uninterpreted (conservative)
(define-fun f_1 ((n Int)) Int B[f_0]) ; Level 1: self-calls → f_0
(define-fun f_2 ((n Int)) Int B[f_1]) ; Level 2: self-calls → f_1
```

Level 0 is an **uninterpreted function** (`declare-fun`). Z3 only verifies properties that hold regardless of what `f_0` returns — sound by construction (same soundness strategy as F*'s ZFuel).

The top-level name `f_N` is used in the verification conditions. This means the verifier checks the property assuming at most N levels of recursive calls.

### Architecture

**Components:**
1. **AST Rewriter** (`replaceSelfCalls`): Walks Core AST, replaces self-references with level-N-1 name. Follows `containsRef()` pattern from `encodable.go`. Creates new nodes (immutable). Respects Lambda/Let shadowing.
2. **Unroll Engine** (`UnrollRecursiveFunction`): Generates the `declare-fun`/`define-fun` chain. Validates depth bounds. Returns declarations + top-level name.
3. **CLI Integration**: `--verify-recursive-depth N` flag. Default depth 2 (matching Dafny's default for assertions). Filters `RejectRecursive` rejections when depth > 0. Labels output distinctly.

### Default Depth: 2

The default recursion depth is **2** (matching Dafny's default for assertion verification). This means:
- When `ailang verify` encounters a recursive function with contracts, it automatically attempts bounded verification at depth 2
- Users can override with `--verify-recursive-depth N` (range 1-10)
- `--verify-recursive-depth 0` disables bounded recursion (only verifies non-recursive functions)
- Depth 2 is sufficient for most common properties (e.g., `result >= 0`, `result >= 1` for non-negative inputs)

**Why 2?** Dafny's experience shows depth 2 handles the vast majority of assertion-level properties. Depth 1 is too shallow for base-case + one recursive step. Depth 3+ is needed only for properties that depend on deeper structural induction.

### Implementation Plan

**Phase 1: Core Unrolling Logic** (~6 hours)
- [ ] Create `internal/smt/unroll.go` with `replaceSelfCalls()` and `UnrollRecursiveFunction()`
- [ ] Handle all 16+ Core node types in AST rewriter (Var, VarGlobal, Lambda, App, If, Let, LetRec, Match, BinOp, UnOp, Intrinsic, Record, RecordAccess, List, Tuple, DictApp, DictAbs, DictRef)
- [ ] Respect Lambda/Let/LetRec shadowing (if `funcName` is rebound, stop replacing)
- [ ] Validate depth bounds (1-10), return error for out-of-range
- [ ] Write unit tests for `replaceSelfCalls` and `UnrollRecursiveFunction`

**Phase 2: Codegen + CLI Integration** (~5 hours)
- [ ] Add `RecursiveDepth int` to `EncodeFunctionOpts` in `codegen.go`
- [ ] Wire unrolling into `EncodeFunction` after callee resolution
- [ ] Add `--verify-recursive-depth` flag to `verify.go` (default 2, max 10)
- [ ] Filter `RejectRecursive` rejections when depth > 0
- [ ] Add `BoundedDepth int` to `verifyResult` struct
- [ ] Label output: `VERIFIED (bounded: depth N)` with explanatory note
- [ ] Write integration tests

**Phase 3: Examples + Documentation** (~3 hours)
- [ ] Create `examples/runnable/contracts/recursive_verify.ail` with factorial, sumTo, fib
- [ ] Update `docs/docs/guides/contracts.mdx` with bounded recursion section
- [ ] Update CHANGELOG.md
- [ ] Z3 integration tests (gated on Z3 availability)

### Files to Modify/Create

**New files:**
- `internal/smt/unroll.go` - AST rewriter + unroll engine, ~150 LOC
- `internal/smt/unroll_test.go` - Unit + integration tests, ~200 LOC
- `examples/runnable/contracts/recursive_verify.ail` - 3 recursive examples, ~40 LOC

**Modified files:**
- `internal/smt/codegen.go` - Add `RecursiveDepth` to opts, wire unrolling, ~30 LOC
- `cmd/ailang/verify.go` - CLI flag, rejection filtering, output labeling, ~40 LOC
- `docs/docs/guides/contracts.mdx` - Bounded recursion section + status update, ~40 LOC

**Reuse (no changes needed):**
- `internal/smt/encodable.go` - `containsRef()` as template for `replaceSelfCalls`
- `internal/smt/callee_resolver.go` - `buildDefineFun()` for define-fun emission, `ResolveCallees` already skips self-calls

**Total: ~500 LOC** (150 impl + 200 tests + 40 CLI + 40 example + 40 docs + 30 codegen)

## Examples

### Example 1: Factorial Verification

**Before:**
```
$ ailang verify examples/runnable/contracts/recursive_verify.ail
⊘ SKIPPED  factorial  [RECURSIVE] Function "factorial" is recursive
⊘ SKIPPED  sumTo     [RECURSIVE] Function "sumTo" is recursive
⊘ SKIPPED  fib       [RECURSIVE] Function "fib" is recursive
```

**After:**
```
$ ailang verify examples/runnable/contracts/recursive_verify.ail
✓ VERIFIED (bounded: depth 2)  factorial   38ms
✓ VERIFIED (bounded: depth 2)  sumTo       42ms
✓ VERIFIED (bounded: depth 2)  fib         51ms

Note: "bounded: depth N" means the property was verified assuming at most N
levels of recursion. This is sound but not a full inductive proof.
```

### Example 2: AILANG Source Code

```ailang
-- examples/runnable/contracts/recursive_verify.ail
module examples/runnable/contracts/recursive_verify

export func factorial(n: int) -> int ! {}
requires { n >= 0 }
ensures { result >= 1 }
{
  if n == 0 then 1
  else n * factorial(n - 1)
}

export func sumTo(n: int) -> int ! {}
requires { n >= 0 }
ensures { result >= 0 }
{
  if n == 0 then 0
  else n + sumTo(n - 1)
}

export func fib(n: int) -> int ! {}
requires { n >= 0 }
ensures { result >= 0 }
{
  if n <= 1 then n
  else fib(n - 1) + fib(n - 2)
}

export func main() -> int ! {}
{
  factorial(5) + sumTo(10) + fib(7)
}
```

### Example 3: Explicit Depth Override

```bash
# Use deeper unrolling for complex properties
$ ailang verify --verify-recursive-depth 5 examples/runnable/contracts/recursive_verify.ail
✓ VERIFIED (bounded: depth 5)  factorial   85ms
✓ VERIFIED (bounded: depth 5)  sumTo       91ms
✓ VERIFIED (bounded: depth 5)  fib         120ms

# Disable bounded recursion (only verify non-recursive functions)
$ ailang verify --verify-recursive-depth 0 examples/runnable/contracts/recursive_verify.ail
⊘ SKIPPED  factorial  [RECURSIVE] Function "factorial" is recursive
⊘ SKIPPED  sumTo     [RECURSIVE] Function "sumTo" is recursive
⊘ SKIPPED  fib       [RECURSIVE] Function "fib" is recursive
```

## SMT-LIB Output Example

For `factorial` at depth 2:

```smt2
; Level 0: uninterpreted (conservative base case)
(declare-fun factorial_0 (Int) Int)

; Level 1: one unfolding (self-calls → factorial_0)
(define-fun factorial_1 ((n Int)) Int
  (ite (<= n 0) 1 (* n (factorial_0 (- n 1)))))

; Level 2: two unfoldings (self-calls → factorial_1)
(define-fun factorial_2 ((n Int)) Int
  (ite (<= n 0) 1 (* n (factorial_1 (- n 1)))))

; Verification condition uses factorial_2
(declare-const n Int)
(assert (>= n 0))                           ; requires
(assert (not (>= (factorial_2 n) 1)))        ; negated ensures
(check-sat)                                   ; expect: unsat (verified)
```

## Success Criteria

- [ ] `replaceSelfCalls` correctly rewrites all Core AST node types with shadowing
- [ ] `UnrollRecursiveFunction` generates correct define-fun chains at depths 1-10
- [ ] Factorial, sumTo, fib verify at default depth 2
- [ ] `--verify-recursive-depth 0` disables bounded recursion (backward compatible)
- [ ] Existing non-recursive examples produce identical output
- [ ] Output clearly labeled "bounded: depth N" to distinguish from full proofs
- [ ] All tests passing (`go test ./internal/smt/...`)
- [ ] Documentation updated (contracts.mdx, CHANGELOG)
- [ ] Example file added and working

## Testing Strategy

**Unit tests:**
- `replaceSelfCalls`: simple Var replacement, VarGlobal replacement, shadowed by Let binding, shadowed by Lambda parameter, nested in If/Match/BinOp, original AST unmodified (immutability check)
- `UnrollRecursiveFunction`: depth 1 (2 declarations), depth 3 (4 declarations), factorial encoding correctness, naming convention (`funcName_0..N`), depth validation (0 → error, >10 → error), non-recursive function (all levels identical body)

**Integration tests:**
- `EncodeFunction` with `RecursiveDepth=3` for factorial produces valid SMT-LIB
- Recursive function + cross-function callee (declaration ordering correct)
- Default depth 2 applied when recursive function encountered

**Z3 integration tests (gated):**
- factorial verified at depth 2 and 5
- sumTo verified at depth 2
- fib verified at depth 3
- Non-recursive functions unchanged

**Manual testing:**
- `ailang verify` with recursive examples
- Verify output labels are clear and informative
- Check solver timeout behavior at depth 8-10

## Non-Goals

**Not in this feature:**
- **Mutual recursion** (f calls g calls f) — Self-recursion covers 90%+ of practical cases; mutual recursion requires call graph analysis
- **Induction proofs** (proving for ALL inputs) — Bounded verification is sound but incomplete; full induction requires lemma infrastructure
- **F*-style fuel sorts / quantified axioms** — Overkill for AILANG's quantifier-free fragment
- **Z3 `define-fun-rec`** — Less control over depth and labeling than explicit unrolling
- **`opaque`/`reveal` mechanisms** — Dafny feature for large proofs; out of scope for initial support
- **Per-function depth attributes** — All functions use the same depth; per-function control deferred

## Timeline

**Day 1** (~8 hours):
- Phase 1: Core unrolling logic + unit tests
- Phase 2 start: Codegen integration

**Day 2** (~8 hours):
- Phase 2 complete: CLI integration + integration tests
- Phase 3: Examples, documentation, Z3 integration tests

**Total: ~16 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Z3 timeout at high depths | Medium | Max depth 10; exponential blowup is inherent — document in output |
| Unsound at insufficient depth | Low | Bounded verification is always sound (uninterpreted base); just incomplete |
| Cross-function + recursion ordering | Medium | `ResolveCallees` already skips self-calls; unroll defs go between callee defs and param defs |
| User confusion: "bounded" vs full proof | Medium | Clear labeling in output + explanatory note; docs explain the difference |

## Design Decisions

| Decision | Choice | Why |
|----------|--------|-----|
| Base case strategy | Uninterpreted function (`declare-fun`) | Sound — Z3 only verifies properties independent of base behavior |
| Default depth | 2 (matching Dafny) | Handles most common properties; depth 1 too shallow for base+recursive step |
| Max depth | 10 | Prevents solver timeout explosion; deeper needs induction lemmas |
| Mutual recursion | Out of scope | Self-recursion covers 90%+ of cases |
| Output labeling | "VERIFIED (bounded: depth N)" | Prevents confusion with full proofs |
| Fragment checker changes | None — CLI filters rejections | Keeps `IsSMTEncodable()` pure; policy decision lives in CLI layer |
| AST rewriter strategy | Create new nodes (immutable) | Follows AILANG's functional style; prevents mutation bugs |

## Related Documents

<!-- Auto-populated by Ollama neural search on "smt bounded recursion" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_4_8/m-bug-recursion-depth.md](design_docs/implemented/v0_4_8/m-bug-recursion-depth.md) (0.48) — Runtime recursion depth overflow; different problem (runtime vs compile-time) but shows recursion handling history
- [design_docs/implemented/v0_6_2/m-capability-budgets.md](design_docs/implemented/v0_6_2/m-capability-budgets.md) (0.41) — Bounded resource tracking; related concept of bounding
- [design_docs/implemented/v0_7_0/concat-operator-type-inference-bug.md](design_docs/implemented/v0_7_0/concat-operator-type-inference-bug.md) (0.38)

**Planned (check for overlap):**
- [design_docs/planned/v0_7_4/m-bug-letrec-single-call.md](design_docs/planned/v0_7_4/m-bug-letrec-single-call.md) (0.47) — LetRec single-call bug; may affect recursive function encoding
- [design_docs/planned/v0_8_0/m-smt-fragment-expansion.md](design_docs/planned/v0_8_0/m-smt-fragment-expansion.md) (0.40) — Parent design doc; Phase E sketches this feature
- [design_docs/planned/v0_8_0/m-smt-lists-sprint-plan.md](design_docs/planned/v0_8_0/m-smt-lists-sprint-plan.md) (0.38) — List SMT support; recursive list functions are a key use case

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why
- [F* Tutorial: Fuel](https://www.fstar-lang.org/tutorial/) - F*'s fuel-based approach
- [Liquid Haskell: Refinement Reflection](https://ucsd-progsys.github.io/liquidhaskell/) - PLE approach
- [Dafny Reference: Fuel Attribute](https://dafny.org/dafny/DafnyRef/DafnyRef) - Dafny's `{:fuel}` attribute
- [Z3 SMT-LIB: define-fun-rec](https://smtlib.cs.uiowa.edu/) - Z3 recursive function support

## Future Work

- **Per-function depth attributes**: `@fuel(5)` annotation to override default depth for specific functions
- **Mutual recursion**: Extend unrolling to handle mutually recursive function groups
- **Induction hints**: User-supplied lemmas that help Z3 prove properties beyond bounded depth
- **Automatic depth selection**: Heuristic to choose depth based on function structure
- **`opaque`/`reveal` mechanisms**: Fine-grained control over which functions are unfolded

---

**Document created**: 2026-02-13
**Last updated**: 2026-02-13

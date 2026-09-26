# Inline-Test Harness: Multi-Argument and Tuple-Valued Test Rows

**Status**: Planned
**Target**: v0.39.0
**Priority**: P1 (Medium) — unblocks AILANG World v1 DecisionPacket freeze (cross-mission)
**Estimated**: 2 days
**Dependencies**: None
**Issue**: [#715](https://github.com/sunholo-data/ailang/issues/715) (world mission; re-verified still broken at HEAD 2026-09-13)

> **Scope note**: this is a **test-harness semantics change** (parser row grammar + collector
> desugaring + harness call construction). It touches `internal/parser/parser_testing.go`,
> `internal/testing/collector.go`, and `internal/testing/harness.go`. No language, type-system,
> or effect-semantics changes.

## Problem Statement

The inline-test harness cannot express test cases for functions of more than one logical input,
and reports tuple-input cases in the most expensive failure shape — collected, then dead.

**Current state (re-verified at HEAD 3e23644e, 2026-09-13; issue was v0.30.0 commit e37b370):**

1. **Flat multi-argument row does not parse.** `tests [(1, 2, 3)]` on a 2-arg function
   produces `PAR_UNEXPECTED_TOKEN` / `PAR_INFINITE_LOOP` cascades and a synthetic
   `{"name": "parse", "status": "fail"}` JSON entry with no location. The parser only accepts
   the nested form `tests [((1, 2), 3)]`.
2. **Tuple-valued input is collected, then dies at runtime.** For a single-parameter function
   `func addPair(p: (int, int)) -> int`, the row `tests [((1, 2), 3)]` parses, is collected as
   `addPair_test_1` with a real location and duration, then fails with
   `harness evaluation failed: harness evaluation failed: no pattern matched in match expression`
   (note the doubled prefix — a separate cosmetic defect).
3. **The working multi-arg form is undiscoverable.** The nested `((a, b), expected)` form
   *does* parse and pass at HEAD (verified: `add3_test_1` with `((1, 2, 3), 6)` passes;
   see `examples/inline_tests_arithmetic.ail`). The failure is that (a) nothing tells an
   author the flat form is wrong or what the right form is, and (b) the nested form is
   *syntactically identical* to a tuple-input row, so the harness resolves it by guessing.

**Impact:** AILANG World freezes five Z3-proven laws (`validEscalation(old, new, recordedNow,
newDeadlineAt)`, …) whose natural test surface is multi-argument. Every law is instead routed
through a hand-written single-int `caseId` dispatcher, adding indirection per law and making
name-pinned tests (39 in that repo) pin dispatcher names instead of the law under test.
The collected-then-dropped shape (2) is the worst kind of failure: it looks like a test that
ran and failed, not a harness that cannot express the case.

## Root-Cause Analysis (from code inspection)

### Why flat rows don't parse — `internal/parser/parser_testing.go` (`parseTestCase`)

The row grammar is hard-coded as a 2-tuple `(input, expected)`:

- If the first token inside the row's `(` is another `(`, the parser takes the **multi-arg
  branch** and requires exactly `((a1, ..., aN), expected)`.
- Otherwise it takes the **single-arg branch** `(input, expected)` and errors on any second
  comma: `expected ) to close test case`.

A flat `(1, 2, 3)` row hits the single-arg branch, parses `1` as the input, then sees the
second comma and emits `PAR_UNEXPECTED_TOKEN`; the outer `parseTestsBlock` loop then reports
`PAR_INFINITE_LOOP`. The diagnostics cascade but never say "use the nested form".

### Why tuple rows collect then die — `internal/testing/harness.go` (`buildFunctionCall`)

`parseTestCase` wraps multiple inputs in an `ast.Tuple`, and `collectInlineTests`
(`collector.go:98-119`) stores each row as the body tuple `(input, expected)`. At
harness-build time, `buildFunctionCall` (harness.go:118) applies **one rule only**:

```go
if tuple, ok := inputExpr.(*ast.Tuple); ok && len(tuple.Elements) > 1 {
    // Multi-arg function: f(a, b, c) becomes App(f, [a, b, c])
```

Any multi-element tuple input is splatted into `App(f, [a, b, c])`, **regardless of the
function's declared parameter count**. For `addPair(p: (int, int))` the row input `(1, 2)` is
indistinguishable from a 2-arg row input, so the harness builds `App(addPair, [1, 2])` against
a 1-parameter lambda whose parameter is a tuple pattern. The evaluator's pattern machinery
(`internal/eval/eval_patterns.go:100`, `decision_tree.go:114`) fails the tuple pattern against
the mis-arity application: **`no pattern matched in match expression`**. Collection succeeded,
runtime dies — the failure shape the issue calls "collected-then-dropped".

There is no place that *swallows* the tuple silently by intention; the bug is that the
splat/direct decision is made from **input shape alone** when it must be made from the
**function's declared arity**, which is available in the `ast.FuncDecl` at collection time.

### The ambiguity, stated plainly

`tests [((1, 2), 3)]` is syntactically identical for:

- 2-argument function → `f(1, 2) == 3`
- 1-argument tuple function → `f((1, 2)) == 3`

Shape alone cannot disambiguate. Arity can.

## Design Ruling

### Ruling 1 — Row grammar: keep `(inputs, expected)`, never flat, never curried

A `tests [...]` row is **always a 2-tuple `(input, expected)`** where `input` is:

- a **single expression** for a 1-parameter function (the expression may itself be a tuple
  literal), or
- a **tuple literal `(a1, ..., aN)`** whose arity N matches the function's parameter count
  for an N-parameter function.

**Rejected alternatives:**
- **Flat rows `(a, b, expected)`** — rejected because the parser cannot distinguish the last
  argument from the expected value without knowing the function's arity, and the parser runs
  before function declarations are resolved within the same signature. A grammar that parses
  only when you already know the answer is a nondeterminism trap in a language whose selling
  point is decidability. (The flat form stays **refused** — see Ruling 3 for the required
  diagnostic quality.)
- **Curried inputs `[(a)(b)]` / per-argument rows** — rejected: AILANG multi-parameter
  functions are single lambdas applied with one `App` (`harness_test.go:225` asserts exactly
  this for ensures harnesses); introducing currying only for test rows would test a calling
  convention the language does not have.

**Why this ruling:** it keeps the grammar arity-independent and deterministic, it is already
the only working form at HEAD, and it makes the migration cost zero for every existing
passing corpus.

### Ruling 2 — Tuple-input rows desugar by declared arity, not input shape

At **collection time** (`collectInlineTests`, which has the `ast.FuncDecl` in hand):

- `arity(f) == 1` and the row input is a tuple literal → desugar to a **single-argument
  application**: `App(f, [Tuple(a1..aN)])` — the tuple is passed as one value, the function's
  parameter pattern destructures it.
- `arity(f) == N > 1` and the row input is a tuple literal of exactly N elements → desugar to
  **direct parameter application**: `App(f, [a1, ..., aN])` (no currying).
- `arity(f) == 0` (nullary) → row input must be `()` (unit); any other input is refused at
  collection (see Ruling 3).

Mechanically: thread the declared parameter count from `FuncDecl` into `TestCase` (e.g. an
`Arity int` field set by the collector) and change `buildFunctionCall` to take it:

```go
// sketch
if arity == 1 {
    return &core.App{Func: f, Args: []core.CoreExpr{astExprToCore(inputExpr)}}
}
// arity > 1: inputExpr must be a Tuple of exactly `arity` elements (validated at collection)
```

The arity is a static property of the declaration, so the desugar is deterministic and needs
no type inference. (If a future variant type-checks rows, the type checker is the right place
to catch arity≠N mismatches; the design here only needs declaration arity, which is available
without it.)

### Ruling 3 — Unsupportable rows are refused at parse/collection time, never runtime death

An inexpressible case must never be reported as a case that ran and failed. The harness adds
a collection-time gate for:

- flat rows (more than one top-level comma in the row) → `PAR_UNEXPECTED_TOKEN` with hint
  *"a tests row is (input, expected); for multi-argument functions use ((a, b), expected)"*;
- tuple input whose element count ≠ declared arity → diagnostic naming the arity and the
  mismatch;
- tuple input with 1 element (`((x,), e)`) → same gate;
- any shape `buildFunctionCall` cannot construct → refuse with a diagnostic rather than the
  current `panic`/runtime pattern-match failure (also removes the doubled
  `harness evaluation failed:` prefix by wrapping once at the executor boundary).

New diagnostics should be catalogued (e.g. `TST` family codes) so agents get actionable
hints instead of `PAR_INFINITE_LOOP` cascades.

## Migration / Compatibility for Existing Corpora

- **Single-argument scalar rows** `(input, expected)` — the entire existing passing corpus —
  are untouched: same parse path, same desugar (`App(f, [input])`).
- **Nested multi-arg rows** `((a, b), expected)` on N-ary functions — already pass at HEAD —
  keep passing; only the construction site changes from shape-guess to arity-based.
- **Tuple-input rows on 1-arg functions** — currently always fail at runtime (verified:
  `fst_test_1` → `no pattern matched`), so fixing them changes no working behavior; the
  previously-dying rows start passing or produce an honest collection-time diagnostic.
- **Flat rows and mis-arity rows** — previously parse errors or runtime deaths; now refuse
  with a targeted diagnostic. Strictly better; no silent-fallback concern.
- Downstream (AILANG World): the `caseId` dispatchers can be deleted in favor of direct
  multi-arg rows once this ships; that migration is the world mission's to perform, not
  gated on this repo.

## Acceptance Criteria

- [ ] `tests [((1, 2), 3)]` on `func addPair(p: (int, int)) -> int` (both direct and
      nested `match` forms) passes.
- [ ] `tests [((1, 2, 3), 6)]` on a 3-arg function passes (regression guard for Ruling 2
      arity>1 path).
- [ ] `tests [(1, 2, 3)]` (flat) is refused at parse/collection with a diagnostic that names
      the correct nested form — no `PAR_INFINITE_LOOP`, no synthetic `parse` entry needed.
- [ ] Tuple input with wrong element count vs declared arity is refused at collection with a
      diagnostic naming both counts.
- [ ] No test case ever reaches `EvaluateInlineTestsWithHarness` evaluation with a shape the
      harness cannot construct (no runtime `no pattern matched` from harness mis-application;
      property tests' own match failures are unaffected).
- [ ] The doubled `harness evaluation failed:` prefix is gone (single wrap at executor).
- [ ] Existing corpus (`make test`, examples in `examples/inline_tests_*.ail`) unchanged in
      outcomes.
- [ ] Corpus unit tests added at collector and harness level, including an
      arity-disambiguation pair (same row text, 1-arg vs 2-arg function, both correct).

## Implementation Notes (for the sprint plan)

- Files: `internal/parser/parser_testing.go` (row refusal + diagnostics),
  `internal/testing/collector.go` (thread arity, validate rows),
  `internal/testing/harness.go` (arity-based `buildFunctionCall`, remove panic paths),
  `internal/testing/executor.go` (single error-wrap boundary).
- `TestCase.Arity` or equivalent must survive into `runner.go`'s comparison path so the
  Go-side expected-vs-actual comparison is unchanged.
- Estimated 2 days: 0.5 parser + diagnostics, 0.75 collector/harness + unit tests,
  0.25 regression sweep, 0.5 downstream corpus updates and JSON-output spot checks.
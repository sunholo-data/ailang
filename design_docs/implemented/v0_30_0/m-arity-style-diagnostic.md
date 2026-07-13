# Arity-Style Diagnostic (m-arity-style-diagnostic)

**Status**: Implemented (mission iteration 21, 2026-07-13 — PASS 97/100 round 1)
**Target**: v1.0.0 (clause-3 accessibility — fleet-tier footgun burn-down)
**Priority**: P1
**Estimated**: 1–2 days
**Dependencies**: None (self-contained diagnostic-quality change in `internal/types`)
**Mission**: V1 clause-3 (`design_docs/v1-mission.md`), item R4c. Cheapest of the NEW-DOC
footgun fixes; full inner-loop sprint (NOT bookkeeping).

---

## Problem Statement

AILANG has **strict arity matching and no partial application** (`\x y. x + y` is arity-2,
strict — see `design_docs/planned/v1_1_0/m-pipe-operator.md:80`). Fleet-tier models arriving with
ML/Haskell habits routinely (a) partially apply, (b) over-supply, or (c) under-supply arguments.
All three currently produce the **same weak, un-actionable diagnostic**.

**Current State** (reproduced live on binary `d6b22b75d`, files in `/tmp/mc20/`):

| Footgun | Source | Current output |
|---|---|---|
| Partial application | `let add = \x y. x+y; let inc = add(1); inc(5)` | `type unification failed at [function application at F:L:C]: function arity mismatch: 2 vs 1` |
| Too many args | `add(1, 2, 3)` | `… function arity mismatch: 2 vs 3` |
| Too few args | `add(1)` | `… function arity mismatch: 2 vs 1` |

Full first-line for the partial case:
```
Error: type error in test/partial (decl 0): type unification failed at [function application at /tmp/mc20/partial.ail:4:16]: function arity mismatch: 2 vs 1
```

**Four concrete weaknesses** (each a clause-3 accessibility deficit):

1. **No public error code.** The user sees a bare `type error` / `type unification failed`.
   Clause-3 explicitly requires an **error-code gate** on footgun fixes so a model (or a human)
   can look up the failure class. Record errors already carry codes (`TC_REC_001`…`TC_REC_004`,
   `internal/types/errors.go:389-392`); arity has none.
2. **Directionally ambiguous.** `2 vs 1` never says which number is the function's declared arity
   and which is the call site — the model cannot tell "too many" from "too few".
3. **No `Suggestion:` line.** Every other structured type error emits one
   (`TypeCheckError.Error()`, `errors.go:88-89`); arity emits none.
4. **No style guidance.** The partial-application case is the single most common fleet-tier ML
   habit, yet the message never states that AILANG has no partial application, nor suggests
   "call with all N arguments" or "wrap in a lambda".

**Impact**: affects AI models (fleet tier especially) and humans. Not a crash, but a
convergence-killer: a model that partially applies and gets `arity mismatch: 2 vs 1` has no signal
to correct toward AILANG's strict-arity calling convention, so it loops.

---

## Goals

**Primary goal**: Turn the three arity footguns into a single, coded, directional, style-aware
diagnostic — WITHOUT changing arity *semantics* (strict arity and no-partial-application stay).

**Success metrics** (measurable):
1. Each of the three `/tmp/mc20/{partial,toomany,toofew}.ail` cases emits a message containing
   (a) the code `TC_ARITY_001`, (b) directional wording ("expected N … got M"), and
   (c) a `Suggestion:`-style hint — verified by a golden test.
2. The too-few / partial-application case's hint explicitly names AILANG's no-partial-application
   rule (e.g. "AILANG has no partial application — supply all N arguments or wrap in a lambda").
3. **Zero regressions**: all `curry_unify_test.go` positive cases (7) stay green; the two
   positive-control programs (`/tmp/mc20/ok.ail`, `/tmp/mc20/curry.ail`) still `✓ No errors`.
4. `make test` + `make verify-examples` green; `higher_order_functions` and `fold_reduce`
   benchmarks unaffected.

---

## Solution Design

### Overview

Two verified facts pin the approach:

1. The failing message is emitted as a **bare `fmt.Errorf`** at
   `internal/types/unification_types.go:39`, *inside* `unifyFunctions`, in the `else` branch that
   runs only **after** the M-DOCPARSE-DX M1 curry-flatten attempt (lines 30-41) has already failed
   to reconcile the two arities.
2. That error is then wrapped by `internal/types/inference_helpers.go:187` —
   `fmt.Errorf("type unification failed at %v: %w", constraint.Path, err)` — a **plain `%w` wrap**.
   Nothing in `internal/types`, `internal/pipeline`, or `cmd/` uses `errors.As` to recover a
   `*TypeCheckError` from the chain (verified: `grep -rn "errors.As" … TypeCheckError` = empty).

**Consequence**: a `*TypeCheckError` constructed at the emission site would be **flattened to plain
text** by the `%w` wrap, so its `Suggestion` field would never render. Therefore the code + hint
must live **inline in the message string** — which is exactly the proven `TC_REC_00X` convention
(`fmt.Sprintf("%s: %s", TC_REC_001, msg)`, `errors.go:409`), and survives arbitrary plain wrapping.

### Architecture

1. **Allocate the code.** Add `TC_ARITY_001 = "TC_ARITY_001"` next to the `TC_REC_00X` block in
   `internal/types/errors.go` (verified free: `grep -rn "TC_ARITY" internal/ cmd/` = empty).
2. **Build the message at the emission site.** Replace the bare
   `fmt.Errorf("function arity mismatch: %d vs %d", len(fp1), len(fp2))` at
   `unification_types.go:39` with a helper (e.g. `arityMismatchMsg(expected, actual int) string`)
   that produces:
   ```
   TC_ARITY_001: function expects <expected> argument(s), but <actual> provided
     Suggestion: <direction-specific hint>
   ```
   where the hint branches on direction:
   - `actual < expected` → "AILANG has no partial application — call with all <expected>
     arguments, or wrap in a lambda `\a b. f(a, b)`."
   - `actual > expected` → "Remove the extra <actual-expected> argument(s); this function takes
     <expected>."
   The `Suggestion:` prefix mirrors `TypeCheckError.Error()`'s render so output is visually
   consistent even though we emit a string (not a struct) here.
3. **Get the direction right (see Conflict Surface).** The emission site currently prints
   `len(fp1) vs len(fp2)` with no expected/actual semantics. The implementer MUST read the
   constraint construction to determine which of `fp1`/`fp2` is the callee's declared arity vs the
   call-site arity, and pin it with a test — do NOT guess.

### Implementation Plan

**M1 — Coded, directional message (0.5d)**
- Add `TC_ARITY_001` constant.
- Add `arityMismatchMsg(expected, actual int) string` helper (in `errors.go`, next to the arity
  constructor) producing the coded + directional + hint text.
- Determine expected/actual direction at `unification_types.go:39` by reading how `unifyFunctions`
  is reached from `Unify`/constraint solving; wire the helper in.
- Golden tests: the three `/tmp/mc20` cases → assert code + direction + hint.

**M2 — Regression lock + docs (0.5–1d)**
- Add regression assertions: all 7 `curry_unify_test.go` tests green; positive controls green.
- `make verify-examples`, `make test`, targeted benchmark spot-check.
- One-paragraph note in the DX/error-quality reference (mirror `m-match-xcheck-error-quality`).

### Files to Modify

| File | Change | ~LOC |
|---|---|---|
| `internal/types/errors.go` | Add `TC_ARITY_001` const + `arityMismatchMsg` helper | +15 |
| `internal/types/unification_types.go` | Use helper at line 39 (direction-correct) | ~5 |
| `internal/types/arity_diagnostic_test.go` (new) | Golden tests for 3 cases + regression asserts | +80 |

---

## Conflict Surface

This change touches `internal/types/` (unification). Section mandatory.

### Syntactic / semantic positions touched

- **No grammar/AST change.** The only touched position is the **error-emission path** inside
  `unifyFunctions` (`unification_types.go:24-41`) and the arity-error text in `errors.go`.
- The message is produced in the `else` branch at line 38-40, reached **only when**
  `len(fp1) != len(fp2)` *after* curry-flattening (lines 33-37) fails to make the arities equal.

### What else lives here — and must NOT regress

| Path through `unifyFunctions` | Existing behavior | Must stay |
|---|---|---|
| Equal arity (`len(p1)==len(p2)`) | skips the whole block, unifies params | unchanged |
| Curried↔tupled reconcilable (`a->b->c` vs `(a,b)->c`) | flatten succeeds (lines 33-37), unifies | unchanged — the new message is in the `else` only |
| Genuinely irreconcilable arity | bare `fmt.Errorf` (line 39) | **this is the only line we change** |
| TCon arity mismatch (`unification_types.go:187`) | separate, already directional (`"%s expects %d args, got %d"`) | out of scope; do NOT touch |

Because the change is confined to the `else` branch that *already* signaled a hard failure, it
cannot alter which programs type-check — only the *text* of an already-guaranteed failure.

### Disambiguation strategy

None required — no new syntax, no new lookahead. The direction (expected vs actual) is resolved by
reading the constraint that feeds `unifyFunctions`, not by parsing. The Conflict Surface risk here
is **semantic (wrong direction label)**, not syntactic; it is pinned by the golden tests in M1
(the too-many case must say "got 3", the too-few case "got 1").

### Programs that MUST still work (regression fixtures)

1. `internal/types/curry_unify_test.go` — all 7 tests, esp. `TestCurriedLambdaUnifiesWithMultiParam`,
   `TestTripleCurriedUnifies`, `TestMultiParamStillWorks`, `TestSymmetricCurriedUnification`,
   `TestPartialCurriedUnification` (positive), and `TestCurriedMismatchStillFails` (negative — its
   error *text* changes but it must still FAIL).
2. `examples/lambdas_higher_order.ail`
3. `examples/no_loops_fold.ail`
4. `benchmarks/higher_order_functions.yml`, `benchmarks/fold_reduce.yml` (fleet corpus)
5. `/tmp/mc20/ok.ail` + `/tmp/mc20/curry.ail` (positive controls — both `✓ No errors` today)

### What deliberately changes

- The **text** of the irreconcilable-arity error (adds code, direction, hint).
- `TestCurriedMismatchStillFails` asserts on the old bare string → its expected message must be
  updated to the new coded form. This is the ONE intentional test-text change; it must still
  assert a failure, not a pass.
- **No semantic change**: strict arity and no-partial-application are preserved; no program that
  compiled before stops compiling, and no program that failed before starts compiling.

---

## Testing Strategy

**Unit / golden tests** (`arity_diagnostic_test.go`, new):
- Partial application (`add(1)` then apply) → message contains `TC_ARITY_001`, "expects 2",
  "but 1 provided", and the no-partial-application hint.
- Too many (`add(1,2,3)`) → `TC_ARITY_001`, "expects 2", "but 3 provided", "remove … argument".
- Too few (`add(1)`) → same family as partial.

**Regression-surface tests** (REQUIRED — Conflict Surface was filled):
- One assertion per "Programs that MUST still work" entry: 7 curry tests green; 2 examples +
  2 benchmarks unaffected; 2 positive controls still `✓ No errors`.

**Manual**:
- Re-run the three `/tmp/mc20` repros; eyeball the rendered `Suggestion:` line.

---

## Non-Goals

- Changing arity **semantics** (adding partial application / auto-currying) — explicitly out.
- Touching the TCon arity site (`unification_types.go:187`) — already directional.
- Upgrading the wrap at `inference_helpers.go:187` to preserve structured errors via `errors.As`
  — a larger cross-cutting refactor; the inline-message approach delivers the clause-3 win without
  it. (Noted as a possible future consolidation if more type errors need structured rendering.)
- A dedicated `ailang explain TC_ARITY_001` entry — nice-to-have, deferred.

---

## Axiom Compliance (net positive)

- **A1 (machine-decidable)** +: a stable, greppable code (`TC_ARITY_001`) replaces free-text.
- **A3 (semantic transparency)** +: directional wording states the actual failure.
- **A7 (AI-friendly)** ++: the style hint targets the #1 fleet-tier ML habit directly.
- No axiom regressions (no semantic/syntactic change). Net: strongly positive.

---

## Verification Log

All claims below verified live on binary `d6b22b75d` (`bin/ailang`, matches `git describe`).

| Claim | How verified | Result |
|---|---|---|
| AILANG has NO partial application | `ailang check /tmp/mc20/partial.ail` | FAILS with arity mismatch (partial app rejected) ✓ |
| Correct 2-arg call compiles | `ailang check /tmp/mc20/ok.ail` | `✓ No errors` ✓ |
| Curried↔tupled unification is legitimate (must not regress) | `ailang check /tmp/mc20/curry.ail` | `✓ No errors` ✓ |
| Emission site is a bare `fmt.Errorf` | read `unification_types.go:39` | confirmed, in post-flatten `else` ✓ |
| A structured arity constructor exists but is unused at the site | read `errors.go:336` (`NewArityMismatchError`, no Suggestion) | confirmed ✓ |
| The error is wrapped by a plain `%w` (flattens structs) | read `inference_helpers.go:187` | confirmed `"type unification failed at %v: %w"` ✓ |
| Nothing recovers `*TypeCheckError` via `errors.As` | `grep -rn "errors.As" internal/types internal/pipeline cmd \| grep TypeCheckError` | empty → inline-message approach required ✓ |
| Code `TC_ARITY_001` is unallocated | `grep -rn "TC_ARITY" internal/ cmd/` (minus tests) | empty → FREE ✓ |
| Existing code convention embeds code in message text | read `errors.go:389-392, 409` (`TC_REC_00X`) | confirmed pattern ✓ |

---

## Related Documents

- `design_docs/planned/v1_1_0/m-pipe-operator.md` — documents strict arity / no partial application
  (the semantic invariant this diagnostic teaches). Distinct: that doc *adds a feature* (`|>`);
  this doc only improves the *diagnostic* for the existing invariant.
- `m-match-xcheck-error-quality` (LANDED mission iter 15) — sibling clause-3 error-quality fix;
  same pattern (code + directional wording + suggestion). This doc mirrors its shape.
- `internal/types/errors.go` `TC_REC_00X` (record errors) — the code-embedding convention reused here.

---

**Created**: 2026-07-13
**Author**: mission-control (iteration 20)

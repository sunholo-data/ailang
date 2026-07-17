# M-DX-EXPECTED-FAIL-FIXES: Fix Remaining Expected-Fail Examples

**Status**: GHOST-CLOSED (2026-07-17, mission iteration 40) — see addendum below
**Target**: v0.9.5
**Priority**: P2 (DX — broken examples erode confidence)
**Estimated**: 1-2 days
**Dependencies**: None
**Milestone ID**: M-DX-EXPECTED-FAIL-FIXES
**Created**: 2026-03-25
**Source**: Audit of `examples/expected_fail/` directory

---

## GHOST-CLOSE ADDENDUM (2026-07-17, mission iteration 40)

A Gate-2 live-repro at origin/dev `1ee919386` found the four "bugs" below were **largely ghosts** —
**0 of 4 required a language fix**. The design doc's diagnoses pre-dated shipped teaching
diagnostics and (for Bug 4) contained a reproduction error. Findings:

| Doc bug | File(s) | Reality at HEAD | Resolution |
|---|---|---|---|
| Bug 1 `\x -> ` not parsed | `contracts/hof_verify.ail` | GHOST — AILANG lambda is `\x. expr`; parser emits a clean teaching diagnostic ("lambda body separator is '.' not '->'"). Example used non-canonical arrow syntax. | Fixed example to `\x.`, promoted to `examples/runnable/contracts/` as a passing guard. |
| Bug 2 multiple `requires` rejected | `contracts/list_recursive_verify.ail` | GHOST — single-block is the intended design; parser teaches "combine with commas". | Merged to one comma-separated `requires {...}`, promoted to runnable. |
| Bug 3 `let` in `@raw` handler fails | `serve_api_webhook.ail` | MOSTLY GHOST — example omitted the `;`/`in` statement separator after a block-RHS `let = match {...}`, and used deprecated string `++`. | Fixed to canonical syntax + `"${...}"` interpolation, promoted to runnable. Underlying parser ASI inconsistency split out (below). |
| Bug 4 `@limit=N` breaks cap checking | `effect_budgets*.ail` (4) | GHOST — the doc's repro placed `--caps` AFTER the filename, where it is ignored. With `ailang run --caps IO,Clock <file>` (flag first), `@limit` enforcement fires at runtime ("effect 'IO' budget exhausted: semantic limit=3, used=3"). | README corrected; files kept as budget-enforcement demonstrations. |
| (README: intended rejections) | `match_foreign_constructor_{option,result}.ail` | CORRECT — excellent cross-ADT constructor diagnostics; already documented as intended type-rejections. | Unchanged. |

Also de-drifted `examples/manifest.json` (two `contracts/` entries were mispathed to
`expected_fail/`, not phantom — the files live at `runnable/contracts/`).

**Split-out backlog item:** the block-RHS-`let` separator inconsistency (a simple-RHS `let x = e`
tolerates omitting the statement separator before a trailing expression, but a block-RHS
`let x = match{...}` does not) is a real but minor parser footgun. Per PROGRAM.md's
default-bias-not-core, it is filed as `m-parser-block-let-separator` (planned/v0_30_0) for
evidence-gated evaluation, NOT folded into this doc/example cleanup.

**Executor** Opus / **evaluator** Sonnet (generator≠judge) **PASS 92/100 round 1**. Zero Go/parser
changes; `make test` + `make verify-examples` green.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change |
| A2: Replayability | 0 | No change |
| A3: Effect Legibility | +1 | Budget annotations should work correctly with capability system |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | 0 | No change |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +1 | Parser should accept documented syntax; AI agents rely on examples |
| A8: Minimal Syntax | +1 | Lambda shorthand and multiple requires are natural syntax |
| A9: Cost Visibility | 0 | No change |

**Net Score: +3** → **Decision: Move forward**

---

## Problem Statement

7 examples in `examples/expected_fail/` demonstrate features that should work but don't. These fall into two categories: parser gaps (3 files) and a runtime bug (4 files). All are documented features whose examples were written but never verified against the actual implementation.

### Impact

- AI agents that read examples will attempt these patterns and fail silently
- Broken examples erode trust in the language documentation
- Effect budgets (`@limit=N`) are a documented feature that doesn't work at runtime

---

## Bug 1: Lambda Syntax `\x -> expr` Not Parsed

**File**: `examples/expected_fail/contracts/hof_verify.ail`
**Error**: `expected '.' after lambda parameter`
**Line**: `map(\x -> x + 1, xs)`

### Root Cause

The parser expects backslash-lambda to use dot syntax (`\x. x + 1`) but the example uses arrow syntax (`\x -> x + 1`). This is a parser limitation — arrow lambdas are a natural syntax that many languages support.

### Investigation Needed

1. Does the parser support `\x. expr` currently? If so, the example just needs updating (not a bug).
2. If we want both `\x. expr` and `\x -> expr`, is the grammar unambiguous?
3. Check if `->` conflicts with function return type annotation in this context.

### Proposed Fix Options

**Option A** (minimal): Update the example to use `\x. x + 1` if dot-lambda works.
**Option B** (enhanced): Add `\x -> expr` as alternative lambda syntax in the parser.

### Key Files
- `internal/parser/parser_expr.go` — lambda parsing
- `internal/lexer/lexer.go` — `->` token

---

## Bug 2: Multiple `requires` Blocks Rejected

**File**: `examples/expected_fail/contracts/list_recursive_verify.ail`
**Error**: `PAR_DUPLICATE_REQUIRES: only one requires block per function; combine with commas`

### Root Cause

The parser enforces a single `requires` block per function. The example uses:

```ailang
requires { offset >= 0 }
requires { len >= 0 }
requires { offset + len <= listLength(xs) }
```

But the parser requires:

```ailang
requires { offset >= 0, len >= 0, offset + len <= listLength(xs) }
```

### Investigation Needed

1. Is the single-`requires` restriction intentional (simplicity) or accidental?
2. Multiple blocks read more clearly for complex preconditions — is there a DX case for allowing them?
3. Do `ensures` blocks have the same restriction?

### Proposed Fix Options

**Option A** (minimal): Update the example to use comma-separated single block.
**Option B** (enhanced): Allow multiple `requires`/`ensures` blocks, merging them internally.

### Key Files
- `internal/parser/parser_decl.go` — contract block parsing (`PAR_DUPLICATE_REQUIRES` error)

---

## Bug 3: `@raw` Route Handler Parse Failure

**File**: `examples/expected_fail/serve_api_webhook.ail`
**Error**: `expected next token to be }, got STRING instead` at line 19

### Root Cause

The `@raw` handler has a function body with a `let` binding:

```ailang
export func handleWebhook(request: {body: string, method: string, path: string}) -> string ! {IO} {
  let body = request.body
  "received webhook: " ++ request.method ++ " " ++ ...
}
```

The parser fails at the `let` after the opening `{`. This may be a conflict between the record type annotation `{body: string, ...}` and the function body `{`.

### Investigation Needed

1. Does the parser confuse the function body `{` with a record literal?
2. Is this specific to `@raw`/`@route` annotated functions or all functions with record params?
3. Test with a simpler function that has a record parameter and `let` in the body.

### Proposed Fix

Debug the parser to determine why `let` is unexpected after `{` in this context. Likely a lookahead/precedence issue in the function body parser.

### Key Files
- `internal/parser/parser_decl.go` — `@route`/`@raw` annotation parsing
- `internal/parser/parser_expr.go` — function body / block expression parsing

---

## Bug 4: Effect Budget `@limit=N` Breaks Capability Checking

**Files**: 4 `effect_budgets*.ail` files
**Error**: `effect 'IO' requires capability, but none provided` (even with `--caps IO`)

### Root Cause

When a function declares `! {IO @limit=3}`, the `--caps IO` flag is not recognized as satisfying the capability requirement. The budget annotation (`@limit=N`) appears to interfere with the normal capability matching path.

### Reproduction

```bash
# This SHOULD work but fails:
ailang run examples/expected_fail/effect_budgets.ail --caps IO,Clock

# Error: effect 'IO' requires capability, but none provided
# Hint: Run with --caps IO   ← but we already passed --caps IO!
```

### Investigation Needed

1. How does the runtime match `--caps IO` against `! {IO @limit=3}`? Is the budget annotation included in the effect name string comparison?
2. Where does capability checking happen? Does it see `IO @limit=3` as the effect name instead of `IO`?
3. Is the budget annotation stripped before capability matching, or does it need to be?

### Proposed Fix

The capability checker must match the BASE effect name (`IO`) regardless of annotations (`@limit=N`). The budget is an enforcement constraint, not part of the effect identity.

### Key Files
- `internal/eval/eval_operations.go` — capability checking at runtime
- `internal/eval/value.go` — `FunctionValue.EffectBudgets` field
- `internal/runtime/builtins.go` — `RequireCapWithBudget` call path
- `cmd/ailang/run.go` — `--caps` flag parsing and EffContext setup

---

## Implementation Plan

### Milestone 1: Investigate and Fix Parser Issues (~100 LOC)

1. Test if `\x. expr` works — if yes, update `hof_verify.ail` (Option A)
2. Update `list_recursive_verify.ail` to use single `requires` block (Option A)
3. Debug `serve_api_webhook.ail` parse failure — determine if record-param + let-body is the trigger
4. Fix parser if needed, or update example if syntax is wrong

### Milestone 2: Fix Effect Budget Capability Matching (~50 LOC)

1. Trace `--caps IO` through to where it's checked against `! {IO @limit=3}`
2. Identify where budget annotation interferes with capability matching
3. Fix: strip annotation before matching, or match on base effect name
4. Verify all 4 effect_budgets examples pass
5. Move all fixed examples back to `examples/runnable/`

### Milestone 3: Verify and Clean Up

1. Run `verify-examples` — should be 149+ passing, 0 failing
2. Delete `examples/expected_fail/` directory if empty
3. Update `examples/expected_fail/README.md` or remove it
4. Update CHANGELOG

---

## Success Criteria

- [ ] All 7 expected_fail examples either fixed and moved to `runnable/`, or have clear "not yet implemented" justification
- [ ] `verify-examples` reports 0 failures
- [ ] `make test` passes
- [ ] `make lint` clean
- [ ] No regressions in existing examples

---

## Testing Strategy

- Each fix verified by running the specific example
- Full `verify-examples` run after all fixes
- `make test` for regression testing
- Parser fixes need targeted unit tests in `internal/parser/`

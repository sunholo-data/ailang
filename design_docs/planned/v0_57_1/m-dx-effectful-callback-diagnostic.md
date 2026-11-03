# M-DX-EFFECTFUL-CALLBACK-DIAGNOSTIC: effectful lambda passed to `std/list` `map` — diagnostic blames the import, not the effect

**Status**: Planned
**Target**: v0.57.1
**Priority**: P2 (Medium — message-only DX; every agent that passes an effectful callback to a pure combinator hits it, and the current leading note sends it to a fix that cannot work)
**Estimated**: 2 days
**Dependencies**: None.

**Reported from**: cli via `ailang messages`, 2026-10-19, AILANG v0.52.0-17-g790169359 ("Hit … writing a docx image checker (two effectful loops over zip entries)"). Triage: [ailang-core-triage/effectful-callback-misread-as-name-collision.md](../ailang-core-triage/effectful-callback-misread-as-name-collision.md). Reproduced verbatim on v0.57.0 (b23cadc) — see Verification Log V1.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The diagnostic is a pure function of the constraint graph (same input → same message); no behavior or acceptance decision changes |
| A2: Replayability | 0 | No trace/trace-format change |
| A3: Effect Legibility | +1 | The error now names the *effect-row restriction* as the cause, in plain language, instead of burying it in solver vocabulary |
| A4: Explicit Authority | 0 | No capability change; the restriction enforced is the same one as today |
| A5: Bounded Verification | 0 | Same local check; decoration happens at the existing constraint-failure wrap |
| A6: Safe Concurrency | 0 | No concurrency surface touched |
| A7: Machines First | +1 | The message leads with the actionable fact (pure-callback restriction + the `mapE` remedy) instead of a red-herring alias hint; agents stop burning fix cycles on `import std/option (map as mapAlt)` |
| A8: Minimal Syntax | +1 | No syntax added; message-only change |
| A9: Cost Visibility | 0 | No cost accounting touched |
| A10: Composability | 0 | No semantic composition change |
| A11: Structured Failure | +1 | Failure is classified by a typed sentinel (`EffectRowMismatchError`) instead of string-matching, and the solver detail is preserved verbatim as a trailing segment |
| A12: System Boundary | 0 | No boundary crossing added; stdlib knowledge stays behind the existing injection seam |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism (message derived only from the failing constraint + rows)
- [x] A3 (Effects): no hidden side effects (rejection set unchanged — programs that fail today fail tomorrow, with a better message)
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): this *is* the machines-first fix (the current note is the human-unhelpful one; the reporter is an agent harness)

---

## Problem Statement

**Current State.** Passing an effectful lambda to a pure combinator fails with a diagnostic whose *leading* note points at the wrong fix:

```ailang
module eff
import std/list (map)
import std/io (println)
export func main() -> () ! {IO} {
  let _ = map(\x. println(x), ["a", "b"]);
  ()
}
```

v0.57.0 actual output (V1):

```
Error: type error in tmp/repro/eff (decl 0): type unification failed at [function application of map at /tmp/repro/eff.ail:5:14 note: `map` is also exported by std/option, std/result — if you meant one of those, import it with an alias, e.g. `import std/option (map as mapAlt)`]: failed to unify parameter 0: failed to unify effect rows: row {IO, ...ε6} carries labels [IO] which the closed row {} does not permit
```

Two defects, in order of appearance:

1. **The leading note is a red herring.** `collisionHint` (`internal/types/import_hint.go:74-92`) is attached by `applicationDiagnosticPath` (`internal/types/typechecker_app_diagnostics.go:8-21`) to *every* failing application whose callee is a bare name exported by ≥2 stdlib modules — it never inspects *why* the application failed. But `std/option`'s `map` (`std/option.ail:13`, `export pure func map[a, b](f: a -> b, …)`) and `std/result`'s `map` (`std/result.ail:13`, same pure callback) carry **the same purity restriction** (V5). Re-importing under an alias cannot fix an effect mismatch — the suggested fix is a dead end in all three modules.
2. **The real cause is in the tail, in solver vocabulary.** `failed to unify parameter 0` (`unification_types.go:49-53`) wrapping the row-unification failure (`row_unification.go:94/101/118`) is correct but unreadable to the report's audience: what failed is that *the callback argument's effect row carries labels the parameter's declared row does not permit*. Nothing in the message says "this combinator takes a pure function; your lambda performs IO".

**Impact.** Any AI agent (the language's primary author) that passes an effectful callback to `map`/`filter`/`foldl`-style combinators — the natural shape for "loop over entries doing IO" (the reporter's docx zip-entry case) — gets a wrong instruction first and the truth last. The reporter explicitly asked: *"when unification fails on an effect row of a callback argument, say so plainly first … and drop the also-exported note when the failure is an effect mismatch rather than a type mismatch."*

**Scope of the misleading-note family (verified, V7/V8):** `filter` with an effectful lambda shows the same note (`filter` is also exported by std/option); a *named* effectful function value (`map(println, …)`) shows the same failure via the closed-rows branch (`incompatible closed rows: r1 has extra labels [], r2 has extra labels [IO]`). The fix must cover both row-failure shapes, not just the reported lambda case.

## Goals

**Primary Goal:** When a function application fails because an argument's effect row is incompatible with the parameter's declared effect row, the error leads with a plain-language effect diagnosis (combinator name, parameter position, declared row, performed labels, argument span) and suppresses the import-alias collision note; all other failures keep today's message byte-for-byte.

**Success Metrics:**
1. The reporter's repro produces a message whose first sentence names the purity restriction and the performed effects (`IO`) — no `also exported by` note.
2. A type mismatch on the same call shape (e.g. `map(\x. x, "abc")`) keeps the collision note byte-for-byte (today's behavior).
3. All existing solver-tail substring assertions (`incompatible closed rows`, `type unification failed`, `failed to unify parameter 0`) still match — the solver detail is preserved verbatim as a trailing segment.
4. Local and deferred (whole-program) solving of the same failing constraint produce identical decorated messages (parity invariant, `TestApplicationDiagnosticNames`).
5. No acceptance decision changes anywhere: `make test-core` green, the #386 must-accept fixtures stay green, the must-reject fixtures stay rejected.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Suppress-only vs. plain-language lead — do BOTH (lead sentence + drop note on effect mismatch) | The report asks for both; suppress-only leaves the agent without a remedy, and a lead without suppression keeps the red herring | human (reporter, explicit) | design | low |
| D2: Where the effect-mismatch classification lives: sentinel raised in `unifyFunctions`' effect-row block (function-type unification), not in `row_unification.go` | That block sees both rows and only ever sees *effect* rows (record rows never enter it), so record-row errors are untouched by construction; `row_unification.go`'s three branches stay byte-identical | agent (per this doc; triage left it open) | design | med |
| D3: How app context reaches the failure wrap: optional typed field `TypeEq.App` (`AppConstraintInfo`), not `Path[0]` string parsing | All 48 `TypeEq{…}` constructions use named fields (V15), so adding a field is compile-safe; string-parsing our own message format is the brittle path the repo rejects elsewhere | agent | design | med |
| D4: Decoration applied inside a shared wrap helper used by BOTH solve loops (`solveEqualitiesInto` + `SolveConstraints`), not at `inferApp`'s local-solve return | The local==deferred parity invariant (`TestApplicationDiagnosticNames` asserts `err.Error()` equality) must hold for the decorated message too; one helper used twice guarantees it structurally | agent | design | med |
| D5: The "use `mapE`" remedy comes from an injected stdlib suggester (`types.EffectfulVariantSuggester` ← `stdlibindex.EffectVariants`), not a hardcoded table in `internal/types` | `internal/types` must not depend on stdlib contents (established `ImportSuggester` doctrine, `import_hint.go:11-13`, wired in `cmd/ailang/diagnostics_wiring.go`); nil hook → generic fallback wording, library use unchanged | agent | design | low |
| D6: Keep the solver vocabulary as a trailing `(solver: …)` segment instead of dropping it | Downstream tests and greppability pin those substrings (`named_batch_test.go:143`, `row_unification_regression_test.go:56,71`, pipeline must-reject `wantSubstr`); agents debugging solver internals need the raw chain | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — decided in this doc (report-specified).
- [x] D2 — decided: `unifyFunctions` effect-row block.
- [x] D3 — decided: typed `TypeEq.App` field.
- [x] D4 — decided: shared wrap helper, both loops.
- [x] D5 — decided: injected suggester + generic fallback.
- [x] D6 — decided: keep solver tail.

No unchecked items — sprint-executor may proceed without pausing.

---

## Solution Design

### Overview

A typed sentinel (`EffectRowMismatchError`) classifies effect-row unification failures inside function-type unification, carrying both rows and the parameter position. The application constraint (`TypeEq`) gains optional `AppConstraintInfo` (callee name/module, spans, argument lambdaness) populated at creation in `inferApp`. A shared wrap helper — used by both equality-solving loops — intercepts the failure: when the sentinel is present and the constraint is an application constraint, it emits a plain-language lead sentence, rebuilds the diagnostic path *without* the collision note, and appends today's full solver chain verbatim as a trailing segment. A CLI-wired stdlib suggester contributes the "std/list also exports `mapE`" remedy sentence; the hook stays nil in library use.

**Nothing about which programs type-check changes.** This is a message-only fix; the rejection set is identical.

### Architecture

**Components:**

1. **Sentinel error** — new `internal/types/effect_row_mismatch.go`:

   ```go
   // EffectRowMismatchError marks a failed unification of two effect rows
   // inside a function-type unification (M-DX-EFFECTFUL-CALLBACK-DIAGNOSTIC).
   type EffectRowMismatchError struct {
       DeclaredRow *Row   // t1-side row (callee's declared row for this position)
       ActualRow   *Row    // t2-side row (the argument's inferred row)
       ParamIdx     int    // -1 = the function's own row; >=0 = parameter position (post-flatten)
       detail       string // exact solver text this error replaces ("failed to unify effect rows: …")
       cause        error  // the raw row-unification error
   }
   func (e *EffectRowMismatchError) Error() string { return e.detail }
   func (e *EffectRowMismatchError) Unwrap() error  { return e.cause }
   ```

   Raised in `unifyFunctions` (`unification_types.go`): the `Unify effect rows` block replaces `fmt.Errorf("failed to unify effect rows: %w", err)` with the sentinel whose `detail` is the *same* string — so every downstream substring assertion on today's text still matches. `DeclaredRow`/`ActualRow` are `eff1`/`eff2` (both default to `EmptyEffectRow()` exactly as today, so both are always non-nil `EffectRow` rows — record rows never enter this block). Orientation note: for the application constraint `Unify(Left=calleeType, Right=expectedFuncType)` there is no var-head swap (both sides concrete), so `eff1` is reliably the *declared* side; the wrap helper nevertheless derives the message symmetrically from the two rows' label sets (see templates) so a swapped orientation degrades to correct wording, not a wrong claim.

   The parameter loop (same function) annotates `ParamIdx`: when `u.Unify(p1[i], p2[i], sub)` fails and `errors.As` finds the sentinel, set `ParamIdx = i` before wrapping with the existing `fmt.Errorf("failed to unify parameter %d: %w", i, err)` (text unchanged).

2. **App context on the constraint** — `internal/types/inference.go`:

   ```go
   // AppConstraintInfo carries application context for decorating failed
   // application constraints. Optional; nil on every non-application TypeEq.
   type AppConstraintInfo struct {
       Callee      string    // bare callee name, "" if anonymous
       FromModule  string    // resolved module ("" for local core.Var)
       Span        ast.Span  // application span (already in Path[0])
       ArgSpans    []ast.Span  // per-argument spans (parallel to Args)
       ArgIsLambda []bool      // per-argument: is the arg a lambda?
   }
   ```

   `TypeEq` gains `App *AppConstraintInfo` (named-field constructions only — V15). `inferApp` (`typechecker_functions.go`, the `addConstraint` at ~line 601) populates it alongside `Path: applicationDiagnosticPath(app)` from the same `coreCalleeName(app.Func)` call, `app.Args` spans, and `isLambda`-style kind checks.

3. **Shared wrap helper** — `internal/types/inference_helpers.go` (or a split file if that exceeds the AI-maintainability line; see Files):

   ```go
   // wrapTypeEqFailure renders a failed application/other equality constraint.
   func wrapTypeEqFailure(constraint TypeEq, err error) error {
       var mm *EffectRowMismatchError
       if errors.As(err, &mm) == false || constraint.App == nil {
           return fmt.Errorf("type unification failed at %v: %w", constraint.Path, err) // unchanged
       }
       lead := effectMismatchLead(constraint.App, mm)      // plain sentence(s) first
       path := applicationDiagnosticPathNoNote(constraint.App) // note-free path
       return fmt.Errorf("%s (solver: type unification failed at %v: %w)", lead, path, err)
   }
   ```

   Both loops route through it: `solveEqualitiesInto` (line 205 — serves the application-local solve *and* deferred replay) and `SolveConstraints` (line 237). RowEq branches untouched (callback effect mismatches surface through TypeEq; annotation-row failures keep today's message).

   Decoration cases:
   - **Sentinel + `App != nil` + `ParamIdx >= 0`** (the callback case): plain lead first, note dropped, solver tail kept.
   - **Sentinel + `App != nil` + `ParamIdx < 0`** (callee's own row vs call row — defensive; unreachable for concrete callees today because the call-side row is a fresh row variable): drop the note (it is an effect mismatch, per the report), no callback lead, otherwise today's text.
   - **No sentinel** (type mismatch, arity mismatch, record rows, non-app constraints): byte-for-byte today's message, note included.

4. **Lead message templates** (deterministic; label sets sorted; rows rendered with the existing `Row.String()`):

   Let `extraActual = labels(ActualRow) − labels(DeclaredRow)` and `extraDeclared = labels(DeclaredRow) − labels(ActualRow)`; `argWord = "lambda"` if `ArgIsLambda[ParamIdx]` else `"argument"`.

   - Common case (`extraActual ≠ ∅`, `extraDeclared = ∅` — the report's case):
     ```
     `<callee>` requires parameter <N> to be a function with effect row `<DeclaredRow>` (pure), but this <lambda|argument> (at <ArgSpans[N]>) performs effects {<extraActual>}. <REMEDY>
     ```
   - Both-closed mismatch (`extraActual ≠ ∅` and `extraDeclared ≠ ∅`):
     ```
     `<callee>` requires parameter <N> to be a function with effect row `<DeclaredRow>`, but this <lambda|argument> (at <ArgSpans[N]>) has effect row `<ActualRow>` (performs {<extraActual>}, missing {<extraDeclared>}). <REMEDY>
     ```
   - `Callee == ""` (anonymous): same template with "this call requires parameter <N> …" in place of `` `<callee>` requires ``.

   `REMEDY` = ``Use an effect-accepting combinator if one is exported (e.g. `<module>`'s `<name>E` variants), or rewrite the traversal as a `func` recursion whose declared effects cover {<extraActual>}.`` — the `<name>E` clause is emitted **only** when `EffectfulVariantSuggester` returns matches (D5); otherwise the generic clause `…if one is exported…` is used with no module claim.

5. **Stdlib variant suggester** — `internal/stdlibindex/index.go`:

   ```go
   // EffectVariants returns same-module exports named name+"E" (the stdlib
   // effect-accepting-combinator convention), sorted; empty if none.
   func EffectVariants(module, name string) []string
   ```

   Implemented off the existing `idx` (`Modules(name+"E")` ∩ `module`). Convention verified: std/list exports `mapE`/`filterE`/`foldlE`/`flatMapE`/`forEachE` (V3); no other std module exports any `…E[a, b, e]` combinator (V20), so the lookup is honest without a hardcoded table. Wired in `cmd/ailang/diagnostics_wiring.go` init(): `types.EffectfulVariantSuggester = stdlibindex.EffectVariants`.

   **Verified remedies for the report's exact case (V3/V4):** `std/list.mapE` (`std/list.ail:205`, `export func mapE[a, b, e](f: a -> b ! {e}, xs: [a]) -> [b] ! {e}`) — the repro with `mapE` type-checks clean. `forEachE` (:262) and `foldlE` (:237) exist for loop-shaped needs. (The report's proposed "foldl/forEach" wording was checked against the real inventory: pure `foldl` (:64) will not accept the callback; `forEach` does not exist — the doc's message names the *`E`* variants, discovered from the index, not a hand-list.)

### Implementation Plan

**Phase 1: Sentinel + classification** (~4 hours)
- [ ] Add `EffectRowMismatchError` (`effect_row_mismatch.go`) with `Error()` returning the exact current wrap text.
- [ ] Raise it in `unifyFunctions`' effect-row block; annotate `ParamIdx` in the parameter loop.
- [ ] Unit tests: sentinel present (errors.As) for both row-failure shapes; `ParamIdx` correct; record-row failures produce **no** sentinel; `Error()` text byte-identical to today's.

**Phase 2: App context + wrap helper** (~6 hours)
- [ ] Add `AppConstraintInfo`; `TypeEq.App`; populate in `inferApp` (all `applicationDiagnosticPath` callers stay in sync — it is the only app-constraint creator, V16).
- [ ] Add `wrapTypeEqFailure`; route both TypeEq wrap sites through it.
- [ ] Unit tests: lead renders for lambda/closed-rows/anonymous cases; note dropped iff sentinel; solver tail preserved; **local==deferred parity** for the decorated message (extend the `TestApplicationDiagnosticNames` pattern); `TestApplicationDiagnosticCollisionHint` untouched and green (type mismatch keeps note).

**Phase 3: Remedy suggester + integration** (~4 hours)
- [ ] `stdlibindex.EffectVariants` + test (map→mapE on std/list; no false positive for, e.g., std/io).
- [ ] Wire `types.EffectfulVariantSuggester` in `cmd/ailang/diagnostics_wiring.go`; nil-hook fallback wording tested in `internal/types`.
- [ ] Pipeline integration tests: reporter's repro (exact new message, no note), `filter` variant, named-function variant, `mapE` version still accepted, `map(\x. x, "abc")` keeps note; `make test-core`, `make fmt`, `make lint`, `make check-boundaries`.

### Files to Modify/Create

**New files:**
- `internal/types/effect_row_mismatch.go` — sentinel + lead builder + `EffectfulVariantSuggester` var (~120 LOC)
- `internal/types/app_effect_diagnostic_test.go` — unit tests for sentinel/wrap/parity (~180 LOC)

**Modified files:**
- `internal/types/unification_types.go` — raise sentinel in effect block, annotate `ParamIdx` in param loop (~+25 LOC)
- `internal/types/inference.go` — `AppConstraintInfo` + `TypeEq.App` (~+20 LOC)
- `internal/types/typechecker_functions.go` — populate `App` at the app constraint (~+12 LOC)
- `internal/types/inference_helpers.go` — both loops route through `wrapTypeEqFailure` (~+15 LOC, −2 changed lines)
- `internal/types/typechecker_app_diagnostics.go` — note-free path variant used by the helper (~+15 LOC)
- `internal/stdlibindex/index.go` — `EffectVariants` (~+25 LOC)
- `cmd/ailang/diagnostics_wiring.go` — wire the suggester (~+2 LOC)
- `internal/pipeline/` — integration fixtures for the decorated rejection (~+80 LOC)

---

## Conflict Surface

*(Required: this change touches `internal/types/`.)*

**1. Positions extended.** The failure wrap of `TypeEq` constraints in both equality-solving loops (`solveEqualitiesInto`, `SolveConstraints`) — and, upstream of it, the *text* of two wraps inside `unifyFunctions` (effect-row block, parameter loop).

**2. Other valid constructs already in those positions.**
- *Type mismatches* on application parameters (`cannot unify list[α] with string`) — must keep the collision note (today's contract, pinned by `TestApplicationDiagnosticCollisionHint`).
- *Arity mismatches* (`arityMismatchMsg`) — different branch, untouched.
- *Record-row* unification failures (`failed to unify label <f>`, `newRecordRowError` family) — never enter the TFunc2 effect-row block (D2), so no sentinel, no suppression.
- *RowEq constraints* (annotation rows, `row unification failed at …`) — wrap sites untouched.
- *Class/instance constraints* (`instances_eq_synth.go:286-287`) — different constraint kind, untouched.
- *Anonymous callees* (`\x. … (args)`) — no collision note exists today; decoration degrades to the anonymous lead wording only.

**3. Disambiguation.** The classification is not textual: the sentinel exists iff the failing unification passed through `unifyFunctions`' effect-row block (function-typed operand, effect rows only); the note is suppressed iff `errors.As` finds that sentinel *and* the constraint carries `App != nil`. `Path[0]` format strings are never parsed (D3).

**4. Programs that MUST still work post-change (fixtures):**
- `examples/match_hof_lambda.ail` — block-body lambdas passed to `map`/`foldl` (pure) — accepted today, accepted after (message changes only on failure). *Exists: verified by `ls` (V14).*
- `examples/poly_arith_lambda.ail`, `examples/effect_budget_demo.ail` — same class of pure-callback use.
- The #386 must-accept fixtures (`internal/pipeline/effect_row_show_interp_test.go`, 9 cases incl. effectful `foldlE` + pure `flatMap`) — green today, must stay green.
- `TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback` (`effect_pure_row_overgeneralization_test.go:307`) — `flatMap` with an `! {FS}` lambda is *accepted* today (V6: verified live via `ailang check`, exit 0); the fix must not start rejecting it (it must not start *classifying* it at all — it type-checks).
- Reporter's repro with `mapE` — accepted (V4: exit 0).

**5. Programs that must still FAIL (non-vacuity):**
- The repro itself — still rejected, new message.
- `internal/pipeline/effect_row_show_interp_test.go` must-reject fixtures (incl. `inline_lambda_annotation_misses_io`, `genuine_missing_io_literal`) — still rejected; their `wantSubstr` ("IO") still matches inside the decorated message.

**What deliberately changes:** the *error text* for effect-row mismatches on function-application constraints: a new leading sentence, no collision note, solver chain demoted to a trailing `(solver: …)` segment. No existing test asserts the old composed text for this case (V13: the only note-asserting test is the type-mismatch one; solver-tail assertions match the preserved tail).

---

## Examples

### Example 1: The reporter's repro

**Before (v0.57.0, V1):**
```
Error: type error in tmp/repro/eff (decl 0): type unification failed at [function application of map at /tmp/repro/eff.ail:5:14 note: `map` is also exported by std/option, std/result — if you meant one of those, import it with an alias, e.g. `import std/option (map as mapAlt)`]: failed to unify parameter 0: failed to unify effect rows: row {IO, ...ε6} carries labels [IO] which the closed row {} does not permit
```

**After (target):**
```
Error: type error in tmp/repro/eff (decl 0): `map` (from std/list) requires parameter 0 to be a function with effect row `{}` (pure), but this lambda (at eff.ail:6:14) performs effects {IO}. std/list also exports `mapE`, which accepts an effectful callback; or rewrite the traversal as a `func` recursion whose declared effects cover {IO}. (solver: type unification failed at [function application of map at eff.ail:5:14]: failed to unify parameter 0: failed to unify effect rows: row {IO, ...ε6} carries labels [IO] which the closed row {} does not permit)
```

The agent's next action changes from a doomed alias-import to either the one-token switch (`map` → `mapE`) or an explicit `func` recursion — both verified to type-check (V4).

### Example 2: Type mismatch keeps today's message (note survives)

```ailang
let _ = map(\x. x, "abc");   -- string where [a] expected
```
Unchanged (V2): the collision note stays, because the failure is a type mismatch, not an effect mismatch — here the note is *legitimate* (std/option's `map` does take an `Option[a]`).

---

## Success Criteria

- [ ] Reporter's repro message leads with the plain effect diagnosis (callee, parameter, declared row, performed labels, argument span) and contains no `also exported by` note.
- [ ] `map(\x. x, "abc")` (type mismatch) keeps the collision note byte-for-byte.
- [ ] Named-function argument (`map(println, …)`) and closed-rows shape get the same plain lead (argument wording).
- [ ] Solver chain preserved verbatim inside `(solver: …)`; `internal/testing/named_batch_test.go:143` ("type unification failed") and `row_unification_regression_test.go` assertions still pass.
- [ ] Local == deferred message parity for decorated failures.
- [ ] `EffectfulVariantSuggester = nil` (library/default) yields the generic remedy clause; `internal/types` acquires no stdlib dependency (`make check-boundaries` clean).
- [ ] All tests passing (`make test`, `make test-core` < 15s), `make fmt`/`make lint` clean.
- [ ] Documentation updated: this doc → implemented on ship; CHANGELOG entry at release.

## Testing Strategy

**Unit tests (`internal/types`):**
- Sentinel: raised for both row-failure shapes; `ParamIdx`; record rows produce none; `Error()` text identical to today's wrap.
- Wrap helper: table over {lambda open-row, named-fn closed-rows, both-extras, anonymous callee, non-app constraint, no-sentinel} — lead wording, note presence, tail preservation.
- Parity: decorated local failure == deferred `SolveConstraints` failure (string equality).
- Suggester nil-fallback wording.

**Integration tests (`internal/pipeline`):**
- The repro (module `eff`) asserting the new lead and the absence of `also exported by`.
- `filter` repro (V7) and named-function repro (V8).
- `mapE` repro stays accepted; #386 must-accept/must-reject suites untouched and green.

**Manual testing:**
- `ailang check` on the repro in a temp dir (MOD010 path-relaxation unaffected); LSP surfaces the same text (same pipeline, no separate formatting).

## Deferred Decisions

- Exact lead phrasing details (backticks vs quotes, label-set braces) — agent may finalize while writing tests; requirements: callee name, parameter index, both rows or their label sets, argument span, deterministic output.
- Whether the lead also names the *module* of non-ambiguous local callees — agent decides (template handles `FromModule == ""`).
- File placement of the wrap helper (`inference_helpers.go` vs a new split file) — agent, respecting the ~800-line AI-maintainability convention that motivated `typechecker_effect_row_issue386.go`.
- Whether to extend `Row.String()` rendering for label sets in the lead — agent; must not change `Row.String()` output used by existing tests.

## Non-Goals

- **Not** making pure combinators effect-polymorphic or changing any acceptance decision — `std/list.map` stays pure; only the diagnostic changes.
- **Not** seeding `Row.Provenance` from source spans (to name the inner `println` call): provenance is only ever unioned/cloned during inference, never seeded (V12) — naming the inner call is a separate, larger change.
- **Not** reviving `NewRowMismatchError`/`TypeCheckError` for this path: it has no production callers today (V17); the sentinel composes with the existing wrap chain instead of replacing it.
- **Not** renaming stdlib exports or adding new error codes (no code allocated — the existing "type error in …" surface carries none, V18).
- **Not** touching record-row diagnostics, `RowEq` constraint wraps, or the deferred class-instance solver.

## Timeline

**Week 1** (2 days):
- Phase 1 + Phase 2 implementation and unit tests
**Week 2** (1 day):
- Phase 3, integration fixtures, full `make test`/lint/boundaries, docs

**Total: ~3 days** (2× the honest 1.5-day estimate)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Decorated text breaks a downstream assertion not found by grep | Med | Phase-1 requirement that sentinel `Error()` be byte-identical limits the delta to *additive* lead + demoted tail; full `make test` (not just test-core) gates |
| `ParamIdx` from flattened curried forms misleads ("parameter 2" of a curried chain) | Low | ParamIdx is the post-flatten position exactly as today's `failed to unify parameter N` already reports — no regression; documented in sentinel |
| Suggester drift (new std module exports `fooE` with different meaning) | Low | Convention-gated on the same `E` suffix + effect param the stdlib already documents (`std/list.ail:196-197` "Effectful Combinators"); hint is advisory text, never an acceptance change |
| Message growth hurts agent token budgets | Low | Lead is one sentence + one remedy clause; the old note (~30 tokens of dead end) is removed |

## Related Documents

**Implemented (this design extends their contracts):**
- [m-length-collision-dx](../../implemented/v0_28_0/m-length-collision-dx.md) (v0.28.0) — introduced `collisionHint` with the deliberate "attach on any failed application, cause-blind" contract; this doc makes that contract cause-aware for effect mismatches only.
- [m-effect-row-show-interp](../../implemented/v0_31_0/m-effect-row-show-interp.md) (v0.31.0, #386) — the application-local effect-row solve the failing constraint flows through; its parity/fixture suite is the regression backbone here.

**Planned (triage for this report):**
- [ailang-core-triage/effectful-callback-misread-as-name-collision](../ailang-core-triage/effectful-callback-misread-as-name-collision.md) — locates the mechanism, flags the design choices this doc freezes (D2), and corrects the report's remedy to the real inventory (`mapE`, not `forEach`).

*(Neural/SimHash auto-search on "dx effectful callback diagnostic" found no matches; the related set above was assembled by reading the triage doc's citations and verifying each file exists.)*

## References

- [Design Axioms](/docs/references/axioms)
- `internal/types/import_hint.go` — `collisionHint`, `ImportSuggester` injection doctrine
- `internal/types/typechecker_app_diagnostics.go` — `applicationDiagnosticPath`, `coreCalleeName`
- `internal/types/unification_types.go`, `internal/types/row_unification.go` — failure chain
- `internal/types/inference_helpers.go:205,237` — the two wrap sites
- `std/list.ail:196-262` — Effectful Combinators inventory

## Verification Log

Every load-bearing claim above, with its check. All repros run on v0.57.0 (b23cadc), the working tree of this doc.

| # | Claim | Check | Result |
|---|-------|-------|--------|
| V1 | The repro reproduces on v0.57.0 with the misleading note leading | `ailang check /tmp/repro/eff.ail` (transcript in Problem Statement) | Confirmed, exit 1, note leads |
| V2 | Type mismatch on same shape keeps note and is *legitimate* there | `ailang check` on `map(\x. x, "abc")` | Confirmed: `cannot unify list[α3] with string`, note present |
| V3 | std/list exports effect-accepting combinators | `grep -n "export func.*E\[" std/list.ail` | Confirmed: `mapE`:205, `filterE`:220, `foldlE`:237, `flatMapE`:248, `forEachE`:262 |
| V4 | `mapE` fixes the reporter's repro | `ailang check` on the repro with `mapE` | Confirmed: `✓ No errors found!`, exit 0 |
| V5 | `std/option.map` and `std/result.map` are also pure (aliasing cannot fix) | `grep -n "export.*func map" std/option.ail std/result.ail` | Confirmed: both `:13`, callbacks `f: a -> b` (no effect row) |
| V6 | `std/list.flatMap` (no effect annotation) *accepts* an `! {FS}` lambda today — must not start failing | `ailang check` on `flatMap(\p. readOne(p), paths)` variant | Confirmed: exit 0 (matches `TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback`, read in full) |
| V7 | `filter` + effectful lambda shows the same misleading note (`filter` also in std/option) | `ailang check` t2.ail | Confirmed: note names std/option |
| V8 | Named-function argument (`map(println, …)`) hits the closed-rows branch | `ailang check` t4.ail | Confirmed: `incompatible closed rows: r1 has extra labels [], r2 has extra labels [IO]` |
| V9 | `collisionHint` is attached cause-blind by `applicationDiagnosticPath` | Read `typechecker_app_diagnostics.go:8-21`, `import_hint.go:68-92` | Confirmed: note appended whenever name is multi-exported, failure kind never consulted |
| V10 | The failure chain is `unification_types.go` param loop → effect block → `row_unification.go` branches | Read `unification_types.go:49-71`, `row_unification.go:88-120` | Confirmed: wrap sites at `:53` (param) and `:70` (effect rows); raw messages at `:94/:101/:118` |
| V11 | The two `TypeEq` failure wraps are `inference_helpers.go:205,237`; the local app solve flows through `solveEqualitiesInto` | Read `inference_helpers.go:180-250`, `typechecker_functions.go:556-650` | Confirmed: `solveLocalEqualities` → `solveEqualitiesInto` (:205); `SolveConstraints` loop (:237) |
| V12 | **Negative:** `Row.Provenance` is never seeded from source spans during inference (naming `println` is not cheaply available) | `grep -rn "Provenance" internal/types/*.go` minus tests: only union (:412-418 `typechecker_data.go`), tail-copy (:211 `row_unification.go`), clone (:120 `app_effects`), lambda-open copy (:53) — no span-seeding write; none in parser/elaborate (grep empty) | Confirmed (hence Non-Goal) |
| V13 | **Negative:** no test asserts the collision-note text *together with* an effect failure (so suppression breaks nothing) | `grep -rn "also exported\|mapAlt" --include="*_test.go" internal/` | Confirmed: only `TestApplicationDiagnosticCollisionHint` (:64), which is a type-mismatch case (read: `length` on a string literal) |
| V14 | Cited regression fixtures exist | `ls examples/match_hof_lambda.ail examples/poly_arith_lambda.ail examples/effect_budget_demo.ail`; read `match_hof_lambda.ail:9-30` (pure lambdas to `map`) | Confirmed |
| V15 | Adding a field to `TypeEq` is compile-safe: all 48 constructions use named fields | `grep -rn "TypeEq{" internal/types/*.go` (48 hits, all `Left:/Right:/Path:` named) | Confirmed |
| V16 | `inferApp` (via `typechecker_functions.go:605`) is the only creator of application-path `TypeEq` constraints | `grep -rn "applicationDiagnosticPath" internal/` (non-test) | Confirmed: one producer (`typechecker_functions.go:605`), one formatter (`typechecker_app_diagnostics.go:8`) |
| V17 | **Negative:** `NewRowMismatchError` has no production callers (we extend the live chain, not a dead one) | `grep -rn "NewRowMismatchError" internal/` minus `errors.go`/tests | Confirmed: empty |
| V18 | **No new error code is proposed or needed** (none allocated, none consumed) | The decorated message rides the existing un-coded `type error in …` surface; `grep` for code prefixes in the touched files shows none to collide with | Confirmed (statement of intent) |
| V19 | Local==deferred parity is an enforced invariant of app diagnostics | Read `TestApplicationDiagnosticNames` (`typechecker_app_diagnostics_test.go:11-42`): asserts `deferredErr.Error() == err.Error()` | Confirmed (D4 preserves it) |
| V20 | **Negative:** no std module other than std/list exports `name+E` effect-combinators (suggester is honest) | `grep -rn "export func.*E\[" std/*.ail` | Confirmed: only `std/list.ail` |
| V21 | Solver-tail substring assertions survive (V-metric 3) | `named_batch_test.go:143` contains "type unification failed"; `row_unification_regression_test.go:56,71` contain "incompatible closed rows"; must-reject `wantSubstr: "IO"` | Confirmed: all match inside the preserved `(solver: …)` tail |

---

**Document created**: 2026-11-03
**Last updated**: 2026-11-03
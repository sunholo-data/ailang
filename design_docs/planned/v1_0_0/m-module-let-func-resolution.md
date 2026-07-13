# M-MODULE-LET-FUNC-RESOLUTION: Module-level lets cannot reference module funcs (false "undefined variable"; hint cites closed #327)

**Status**: Planned
**Target**: v0.30.0 (v1.0.0 queue, clause-3 DX/soundness)
**Priority**: P1 (false diagnostic on a natural idiom; the hint actively misroutes agents; broke nightly `higher_order_functions` 2026-07-13)
**Estimated**: 2–3 days (SCC unification 1–1.5d, semantics pinning + fixtures 1d)
**Dependencies**: None. GitHub: [#366](https://github.com/sunholo-data/ailang/issues/366). Predecessors: #327 (v0_29_0 m-record-update-local-resolution), M-BUG-MODULE-LET-SCOPE (v0_4_9).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change |
| A2: Replayability | 0 | No change |
| A3: Effect Legibility | 0 | No change (module lets stay pure-value positions; effect rules unchanged) |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +1 | Restores local reasoning: a module-scope name resolves the same in every DECL class, not just every expression position |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +2 | The natural idiom fails with a FALSE diagnostic whose "workaround" is a no-op for this shape — an agent following it loops forever (observed: nightly higher_order_functions thrash 2026-07-13) |
| A8: Minimal Syntax | +1 | Removes a decl-class exception to uniform scoping; no new syntax |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Module lets compose with module funcs like any other binding |
| A11: Structured Failure | +1 | The interim hint stops lying (no closed-bug citation, no inapplicable workaround) |
| A12: System Boundary | 0 | No change |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): this fix exists FOR machine repair-loop sanity

## Problem Statement

Found via nightly-eval regression triage 2026-07-13: `higher_order_functions`
(opencode-qwen3-5) went solid→broken because the model spontaneously wrote
module-level `let`/`letrec` declarations — a shape the elaborator can never
resolve against module funcs — and the diagnostic's workaround ("bind it with
let first") is a no-op for that shape, so the model thrashed to failure.

**Current State (all live-verified at HEAD `v0.29.2-115-g20e0fe4f1`, 2026-07-13):**

| Shape | Result |
|---|---|
| module `let` → earlier module `let` | ✅ resolves |
| module `func` → module `let` (M-BUG-MODULE-LET-SCOPE, v0.4.9) | ✅ resolves |
| module `let` → IMPORTED func (`examples/runnable/array_basic.ail`: `A.fromList`) | ✅ resolves |
| module `let` → module `func`, func declared FIRST | ❌ false "undefined variable" |
| module `let` → module `func`, func declared AFTER | ❌ false "undefined variable" |
| module `let` = immediate call (`let four = double(2)`) | ❌ false "undefined variable" |
| module `letrec` → itself | ❌ false "undefined variable" (no hint) |
| `export let` | dedicated `PAR_UNSUPPORTED_EXPORT_LET` — honest, unchanged |
| `let` and `func` with the SAME name in one module | compiles SILENTLY — semantics unpinned ⚠ |

Minimal repro (fails, both declaration orders):

```ailang
module benchmark/solution
export func double(x: int) -> int = x * 2
let quad = \y. double(double(y))
export func main() -> int = quad(4)
```

```
Error: type error in ... (decl 0): undefined variable: double at 3:23
(double is defined in this module but not resolvable in this position —
known bug #327; workaround: bind it with let first)
```

The hint lies twice: **#327 is CLOSED** (fixed v0.29.0), and the reference
already IS in a let, so the suggested workaround cannot terminate an agent's
repair loop.

**Impact:**
- Any AI-written module using top-level `let` constants/combinators that call
  local helpers hits a wall with a misleading exit sign (A7 violation in effect).
- Benchmarks: `higher_order_functions` regression 2026-07-13 (both trials);
  same dialect shape appears in the docx non-convergence family.

## Mechanism (verified by reading the code path, not inferred from output)

1. `BuildCallGraph(funcs, …)` at `internal/elaborate/scc.go:111` takes **funcs
   only**. Module-level lets are never call-graph nodes → no ordering edges.
2. Module lets are instead handled by `collectModuleLets` (file.go:130) +
   `wrapInLets` (file.go:279–302): **every** core decl (each func) is wrapped
   INSIDE the full chain of module lets. A let's value is therefore
   type-checked OUTSIDE any func binding — it can see the global env (imports,
   builtins) and earlier lets in the chain, but **no module func, ever**. This
   explains order-independence of the failure and why imported funcs work.
3. `wrapInLets` emits plain `core.Let` (file.go:351), so module `letrec` gets
   no recursive binding → self-reference also fails.
4. Secondary smell: let values are re-elaborated once per wrapped decl
   (file.go:286 and 316–330) — n_lets × n_decls elaborations, duplicated core
   subtrees for every decl (also duplicates evaluation cost per-decl at
   runtime under the current wrapping).
5. The lying hint is `internal/types/import_hint.go:50` (test:
   `local_resolution_hint_test.go:37` asserts the "#327" text).

This is the 4th member of the "#323/#327 resolution diverges by syntactic
position" family, at the **decl class** level rather than expression-position
level: the v0.29.0 fix made `findReferences` exhaustive over expression
positions but only wired **func→func** edges.

## Goals

**Primary Goal:** a module-level `let`/`letrec` value resolves module-scope
names exactly as a func body does — one unified declaration ordering.

**Success Metrics:**
- All ❌ rows in the matrix above compile green (repros v3/v4/v7/v8 preserved
  as tests).
- `internal/types/import_hint.go` no longer cites a closed bug anywhere.
- `make verify-examples` baseline unchanged (185 pass / 5 pre-existing #341).
- `higher_order_functions` nightly benchmark passes with the top-level-let
  solution shape (offline replay of the 2026-07-13 failing solution, minus the
  model's unrelated `multiply`-undefined and `_`-placeholder mistakes).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Unified SCC over lets+funcs (replace wrapInLets) vs. keep wrapping + special-case | Determines whether this bug family can recur for the next decl class | agent (design below; executor validates) | design | high |
| let/func same-name collision semantics (today: silent) | Silent shadowing is a soundness trap; must be pinned by test or made an error | agent → propose DUPLICATE-NAME error, human may veto in review | compile | med |
| `letrec` self/mutual recursion at module level: support via core.LetRec or reject with honest diagnostic | Parser accepts `letrec` today; silent false error is the worst of the three options | agent (support if ≤0.5d, else honest diagnostic + issue) | compile | med |
| Interim hint text (if fix slips): stop citing #327, cite #366, state the REAL workaround (`func` form) | The current hint actively misroutes agents | agent | design | low |

### Design Freeze

- [ ] Decl-ordering approach confirmed: unified call graph over funcs + module
  lets, emitting lets as first-class core decls in topological order
  (executor must verify the link/eval layer accepts non-lambda module decls
  before committing; fallback documented in Risks)
- [ ] Name-collision behavior decided and test-pinned (proposed: duplicate
  module-scope name = compile error)

## Solution Design

### Overview

Make module-level lets first-class citizens of the existing SCC machinery
instead of a wrapping special case: add them as call-graph nodes, run the
(already exhaustive, post-#327) `findReferences` over their value expressions,
and emit them as their own core decls in topological order interleaved with
funcs. Delete `wrapInLets`.

### Components

1. **Call-graph extension** (`internal/elaborate/scc.go`): `BuildCallGraph`
   accepts module lets (name + value expr) alongside `FuncSig`s; edges from
   let→func, func→let, let→let via `findReferences(value)`. The DEBUG_STRICT
   exhaustiveness panic already guards traversal gaps.
2. **Decl emission** (`internal/elaborate/file.go`): for each SCC in topo
   order, emit `core.Let` (or `core.LetRec` for self/mutual-recursive groups
   that include lets, if supported — see Design Freeze) with the same
   `let NAME = VALUE in VAR(NAME)` shape funcs use today. Remove
   `wrapInLets` and both re-elaboration loops (file.go:279–302, 316–333).
   Non-func statements (file.go:305+) follow the same ordering.
3. **Collision gate**: duplicate module-scope name (let vs func, let vs let)
   → compile error with both positions (today: silent, semantics undefined).
4. **Hint truth pass** (`internal/types/import_hint.go:50` + its test): the
   "defined in this module but not resolvable" path should no longer be
   reachable for this class post-fix; the residual hint text (kept for any
   future member of the family) cites #366, drops the closed-#327 citation,
   and states the verified workaround: *declare it as a `func`* (verified
   green at HEAD) — NOT "bind it with let first".

### Implementation Plan

**Phase 1: Unify decl ordering** (~1–1.5d)
- [ ] Extend call graph to let nodes; topo-emit lets as core decls
- [ ] Delete `wrapInLets` + duplicate elaboration loops
- [ ] Verify link/eval layer handles non-lambda module decls (spike FIRST — this
  is the fallback trigger)

**Phase 2: Semantics pinning** (~0.5d)
- [ ] Duplicate-name compile error + test (v10 shape)
- [ ] `letrec` decision: core.LetRec support or honest diagnostic (v7 shape)

**Phase 3: Diagnostics + fixtures** (~0.5–1d)
- [ ] Hint truth pass + update `local_resolution_hint_test.go`
- [ ] Test file with the full behavior matrix (v1–v10 shapes)
- [ ] `examples/module_let_helpers.ail` (new runnable example: let→func)
- [ ] CHANGELOG, docs/reference errors section, footguns.md row update

### Files to Modify/Create

**Modified:**
- `internal/elaborate/scc.go` — let nodes in call graph (~40 LOC)
- `internal/elaborate/file.go` — topo emission, delete wrapInLets (~-60/+80 LOC)
- `internal/types/import_hint.go` — truthful hint (~10 LOC)
- `internal/types/local_resolution_hint_test.go` — updated assertion
- `internal/diag/footguns.md` — row update (#327 row → retired/#366)

**New:**
- `internal/elaborate/module_let_resolution_test.go` — behavior matrix (~150 LOC)
- `examples/module_let_helpers.ail` — runnable example (~20 LOC)

## Conflict Surface (REQUIRED — touches internal/elaborate)

1. **Positions extended**: module-level declaration ORDERING (the interleave of
   `let`/`letrec`/`func` decls). No syntactic positions change — parser
   untouched.
2. **Other constructs in those positions**: type decls (processed in a separate
   earlier pass, file.go:120–126 — unaffected); imports (globalEnv, unaffected);
   non-func statements (file.go:305–335 — currently ALSO wrapped in lets; must
   see lets via the same ordering post-fix); func contracts/DeclMeta
   (per-func, keyed by name — let decls must NOT collide with meta handling);
   named test blocks (elaborated via the func path — executor verifies).
3. **Disambiguation**: none needed syntactically; this is semantic ordering
   only. Tarjan handles cycles; a let↔func cycle becomes an SCC → LetRec (or
   an honest error, per Design Freeze).
4. **Programs that MUST still work** (all exist, all pass at HEAD):
   - `examples/runnable/fnv1a.ail` — func→let constants (the v0.4.9 direction)
   - `examples/runnable/array_basic.ail` — module let calling IMPORTED funcs
   - `examples/deriving_eq.ail` — module lets with ADT constructors + funcs after
   - `examples/runnable/list_sum.ail` — let…in expression form (NOT module-let;
     must stay untouched)
   - full `make verify-examples` baseline: 185 pass / 5 pre-existing (#341)
5. **Deliberate changes**: (a) previously-erroring programs (let→func, letrec
   self-ref if supported) now compile — strictly widening; (b) duplicate
   module-scope names become a compile error (today silent — narrowing, called
   out in CHANGELOG); (c) evaluation order/cost of module lets changes from
   per-decl re-evaluation (wrapped copies) to once-per-decl-group — observable
   only via effects, and module lets are pure-value positions, so no observable
   change (executor adds a test if any effectful module-let path exists).

## Verification Log (claims → evidence)

| Claim | Evidence |
|---|---|
| let→func fails both orders; immediate call fails; letrec self-ref fails | live `ailang check` v3/v4/v8/v7 at HEAD 20e0fe4f1, 2026-07-13 (transcripts in #366) |
| let→let, func→let, let→imported-func all work | live `ailang check` v2/v6 + `array_basic.ail` in passing examples baseline |
| `export let` unsupported with honest error | live check v9: `PAR_UNSUPPORTED_EXPORT_LET` |
| let/func same name compiles silently | live check v10: `✓ No errors found!` |
| Mechanism = wrapInLets outside func bindings; funcs-only call graph | READ `internal/elaborate/file.go:279–302,305–335,340–356` + `scc.go:111–134` (not inferred from output) |
| Hint text location + its test | `internal/types/import_hint.go:50`, `local_resolution_hint_test.go:37` |
| #327 is closed/fixed | `gh issue view 327` → CLOSED 2026-07-09; fix M-DOGFOOD-FIXES M2 (scc.go findReferences exhaustive) |
| No new error code proposed | n/a — fix REMOVES a false error; hint edit reuses existing text path (no MODxxx/TCxxx allocation). If Phase 2 adds the duplicate-name error, executor greps for a free code at implementation time and records it |
| Cited regression fixtures exist | `ls` verified 2026-07-13: fnv1a.ail, array_basic.ail, deriving_eq.ail, list_sum.ail all present |
| Frequency claim | 1 nightly benchmark regression (2 trials, 2026-07-13) + dialect-confusion family precedent; NOT claimed to be a top-N failure cause |

## Examples

### Example 1: helper combinators (the nightly shape)

**Before (false error, misleading hint):**
```ailang
module benchmark/solution
export func subtract(x: int, y: int) -> int = x - y
let sub4 = \y. subtract(4, y)      -- ❌ undefined variable: subtract … known bug #327
export func main() -> int = sub4(11)
```

**After:**
```ailang
-- identical source compiles; sub4 ordered after subtract by the unified SCC
✓ No errors found!
```

### Example 2: interim honest workaround (already works today, hint should say so)

```ailang
func sub4(y: int) -> int = subtract(4, y)   -- ✅ verified green at HEAD
```

## Success Criteria

- [ ] Behavior-matrix test green (all v1–v10 shapes asserted per Design Freeze decisions)
- [ ] Base-binary non-vacuity: matrix repros FAIL at pre-fix HEAD, PASS post-fix
- [ ] `grep -rn "known bug #327" internal/` → 0 hits
- [ ] `make verify-examples` baseline unchanged
- [ ] `go test ./internal/... -count=1` green
- [ ] New runnable example + CHANGELOG + footguns row + errors reference updated

## Testing Strategy

**Unit tests:** behavior matrix in `module_let_resolution_test.go`; SCC
ordering with let nodes (incl. cycle → SCC grouping); duplicate-name error.

**Integration tests:** verify-examples full run; offline replay of the
2026-07-13 `higher_order_functions` solution shape (cleaned of the model's
unrelated mistakes).

**Manual:** `ailang check` the v1–v10 matrix; run the new example.

## Deferred Decisions

- Free error-code choice for the duplicate-name diagnostic (Phase 2) — agent
  greps and records at implementation time (claim-class 1).
- `letrec` module-level support vs honest rejection — agent decides by spike
  cost (≤0.5d supports; otherwise honest diagnostic + follow-up issue).
- Whether the residual not-resolvable hint survives at all post-fix — agent
  may delete it if the path is provably unreachable.

## Non-Goals

- `export let` support — separate feature, honest error today, unchanged.
- Effectful module-level initializers — module lets remain pure-value
  positions; no semantics extension here.
- Cross-module let visibility — out of scope (no export).

## Timeline

**Day 1–1.5:** Phase 1 (spike link/eval acceptance FIRST, then unify ordering)
**Day 2:** Phase 2 (semantics pinning) + Phase 3 start
**Day 2.5–3:** fixtures, docs, CHANGELOG, eval replay

**Total: ~2–3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Link/eval layer assumes module decls are lambdas (runtime break on value decls) | High | Phase-1 spike BEFORE deleting wrapInLets; fallback: keep wrapping but ALSO add funcs into the let-value env via a pre-pass env extension (smaller, uglier, recurrence-prone — documented as fallback only) |
| Evaluation-order change observable | Med | Module lets are pure-value positions (no effect rows accepted there); matrix test pins; executor greps for any effectful module-let acceptance first |
| Duplicate-name error breaks an existing example/package silently relying on shadowing | Med | verify-examples + ailang-packages grep before enabling; if hits exist, downgrade to warning + issue |
| SCC change destabilizes #327's 40-cell position matrix | Low | `record_update_positions_test.go` runs in CI; run explicitly in Phase 1 |

## Related Documents

<!-- Neural search SKIPPED (rig busy: qwen3.6 eval-suite mid-run, GPU rule) — grep + SimHash + manual gate used instead; distinctions verified by reading each doc. -->

**Implemented (may inform design):**
1. [m-record-update-local-resolution](../../implemented/v0_29_0/m-record-update-local-resolution.md) — #327: same family, but expression-POSITION edges for func→func only; this doc is the decl-CLASS member. Its fix (exhaustive findReferences + DEBUG_STRICT) is reused as-is.
2. [v0_4_9_m-bug-module-let-scope](../../archive/v0_4_9_m-bug-module-let-scope.md) — introduced `wrapInLets` (func→let direction); this doc replaces that mechanism with unified ordering while preserving its guarantee (fnv1a.ail fixture).
3. [m-dogfood-fixes (v0_29_0)](../../implemented/v0_29_0/) — M2 shipped the exhaustiveness contract this design extends to decl nodes.

**Planned (check for overlap):**
- None. Grep of `design_docs/planned/**` for module-let/resolution: no hits (2026-07-13). SimHash top hits (bytecode-interpreter, pipe-operator) are keyword noise, verified unrelated.

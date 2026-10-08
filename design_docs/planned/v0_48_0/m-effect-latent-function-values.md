# M-EFFECT-LATENT-FUNCTION-VALUES — Charge the latent effects of function-valued arguments and record fields

**Status**: Planned
**Target**: v0.53.0 (next minor release; roll to v0.54.0 if the v0.53.x release train is closed; soundness fix that rejects programs that compile today)
**Priority**: P0 (High) — a `pure` signature can be bypassed by `ailang check`; the runtime capability layer is the only backstop
**Estimated**: 4–6 days (Phase 1 ≈ 2 days, Phase 2 ≈ 1 day, Phase 3 ≈ 1–2 day spike + build, Phase 4 ≈ 1 day)
**Dependencies**: No semantic prerequisite. Scheduling: land this fix first, then rebase #616 onto its per-App metadata and canonical effect-label representation. Independent of the historically parked [M-EFFECT-ROW-VAR-UNIFICATION](../v1_0_0/m-effect-row-var-unification.md) (#616) — see "Relation to prior art".
**Source**: GitHub issues [#1326](https://github.com/sunholo-data/ailang/issues/1326) (effect leak through any higher-order function) and [#573](https://github.com/sunholo-data/ailang/issues/573) (effect checker not transitive through function-valued record-field calls, incl. the 2026-08-26 addendum on record-update / constructor-payload positions)
**Base measured**: `origin/dev` = `1fcc479f1`, binary built from this worktree (`go build -o /tmp/ailang-ds ./cmd/ailang`; the version stamp shows the main checkout's HEAD — known worktree stamping quirk — so provenance was verified behaviourally, see V1–V4).

**Scheduling update (2026-10-08)**: Mark approved scheduling from P0 triage; the design was approved and merged in #1378. Triage reverified both leaks on `origin/dev` `658ff76a3`. Implementation remains pending. See [sprint plan](../v0_53_0/m-effect-latent-function-values-sprint-plan.md). Refs #1326, Refs #573; sequencing Refs #616. Historical measurements and prototype results below are retained as evidence, not claims about the current tree.

## Summary

Both issues are the same defect: **the effect-checking pass (`internal/pipeline/validate_effects.go`) is
the only place a declared effect row is enforced, and it reads a function *value's* latent effects from
two sources that are silently empty.**

1. **#1326** — a same-module function *name* used as a value (`applyTo(logIt, x)`, `sortBy(cmpLog, xs)`)
   is a `*core.Var`, and `collectRequiredEffects` returns `nil` for every `*core.Var`
   (`validate_effects.go:311-314`). The two sibling forms of a function value are charged:
   an imported name (`*core.VarGlobal`, `:316-325`) and a lambda literal (`*core.Lambda`, `:330-335`).
   Three spellings of the same program, three answers (V5).
2. **#573** — a call *through* a value (`p.f(0)`, `callback(x)`) is charged with the concrete labels of
   the value's type. But the elaborator converts every function-type *annotation* — parameter types,
   return types, record fields of type declarations, ADT constructor payloads, `let` annotations — with
   `(*Elaborator).astTypeToInternalType`, whose `*ast.FuncType` case (`internal/elaborate/file_funcs.go:386-403`)
   **ignores `typ.Effects` entirely** and emits an anonymous open row `{| ε_annotN}`. So `f: (int) -> () ! {IO}`
   inside the body is `int -> () ! {...ρ}` (V9), `extractEffectFromType` normalises a label-less tail row to
   *pure* (`validate_effects.go:276,286`), and the call is charged nothing.

The fix is two small, independent rules in the existing pass plus one prerequisite, all prototyped in this
session: **zero regressions over 425 `examples/`, 49/49 `std/` modules, and the `internal/{pipeline,types,elaborate,iface}`
test suites** (V17–V20), while every repro in both issues flips to a rejection naming the missing effect (V16).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime or evaluation-order change; compile-time check only |
| A2: Replayability | 0 | Traces unaffected |
| A3: Effect Legibility | +1 | Restores "the signature tells you the effects": today a `pure` function prints (V2, V3, V7) |
| A4: Explicit Authority | +1 | Declared rows feed capability reasoning (motoko's `declared_vs_performed` gate, package effect ceilings); a bypass is ambient authority |
| A5: Bounded Verification | +1 | The rule stays local to one function body + callee signatures; no whole-program analysis |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | An agent can trust a declared row instead of re-deriving it; the runtime hermeticity probe motoko runs to compensate (#573) becomes redundant |
| A8: Minimal Syntax | 0 | No new syntax; existing annotations start meaning what they say |
| A9: Cost Visibility | 0 | — |
| A10: Composability | 0 | Effect-polymorphic combinators keep working (pinned tests stay green, V19); callers of them become responsible for the callback's effects, which is the composable reading |
| A11: Structured Failure | 0 | Reuses the existing "Missing effects:" diagnostic; no new error code |
| A12: System Boundary | +1 | Record-of-functions ports (motoko `ExtPorts`) are exactly a boundary; calls across it become visible |

**Net Score: +6** → **Proceed.** Hard-violation check: A1 ✓ (none), A3 ✓ (this *removes* a hidden side effect), A4 ✓, A7 ✓.

## Problem Statement

### The repros (this session, base `1fcc479f1`)

Each file below was written, formatted by the repo's `ailang fmt` hook, and run exactly as shown.

**R1 — #1326, local higher-order function** (`hof_local.ail`)
```ailang
module hof_local
import std/io (println)

func logIt(x: int) -> int ! {IO} = {
  println("side effect")
  x
}

func applyTo(f: int -> int, x: int) -> int = f(x)

pure func sneaky(x: int) -> int = applyTo(logIt, x)

export func main() -> () ! {IO} = println(show(sneaky(1)))
```
```
$ ailang check hof_local.ail
→ Type checking hof_local.ail...
→ Effect checking...

✓ No errors found!
$ ailang run --caps IO --entry main hof_local.ail
✓ Running hof_local.ail
side effect
1
```

**R2 — #1326, stdlib `sortBy`** (`hof_sortby.ail`)
```ailang
module hof_sortby
import std/io (println)
import std/list (sortBy)

func cmpLog(a: int, b: int) -> int ! {IO} = {
  println("compared")
  a - b
}

pure func s(xs: [int]) -> [int] = sortBy(cmpLog, xs)

export func main() -> () ! {IO} = println(show(s([3, 1, 2])))
```
```
$ ailang check hof_sortby.ail
✓ No errors found!
$ ailang run --caps IO --entry main hof_sortby.ail
compared
compared
compared
[1, 2, 3]
```

**R3 — #573, rowless record-field call** (`field_call.ail`)
```ailang
module field_call
import std/io (println)

export type P = { f: int -> () ! {IO} }

func rowless(p: P) -> () {
  p.f(0)
}

func effectful(_n: int) -> () ! {IO} {
  println("EFFECT PERFORMED")
}

export func main() -> () {
  let p = { f: effectful }
  rowless(p)
}
```
```
$ ailang check field_call.ail
✓ No errors found!
$ ailang run --caps IO --entry main field_call.ail
✓ Running field_call.ail
EFFECT PERFORMED
$ ailang run --entry main field_call.ail          # no caps: the runtime backstop is the only thing that notices
Error: execution failed: effect 'IO' requires capability, but none provided
```

**R0 — control: the direct call is rejected** (`control_direct.ail`: same as R1 with
`pure func direct(x: int) -> int = logIt(x)`)
```
Error: effect checking failed in control_direct: Effect checking failed for function 'direct'
  Function uses effects not declared in signature

  Missing effects: IO

  Current signature: func direct(...) -> T
  Suggested fix:     func direct(...) -> T ! {IO}
```

The bytecode VM path behaves identically (the effect pass runs before either backend):
`ailang run --bytecode --caps IO --entry main hof_local.ail` prints `side effect` / `1`, and
`field_call.ail` prints `EFFECT PERFORMED` (V4).

### How wide the hole is (measured, not estimated)

**#1326 shape — which stdlib HOFs let a `pure` caller perform IO through a *named* callback** (one probe
per function: `pure func p(xs: [int]) -> … = <hof>(logX, xs)` with `logX ! {IO}`):

| std/list function | base `check` | base `run` prints side effect | why |
|---|---|---|---|
| `any`, `findIndex`, `foldr`, `sortBy`, `zipWith`, `flatMap` | **accepted** | **yes** | callback row open; caller not charged |
| `map`, `filter`, `foldl`, `takeMap` | rejected (type error: `incompatible closed rows … [IO]`) | no | builtin callback row is closed-pure |
| `mapE` | rejected (`Missing effects: IO`) | no | declared `! {e}` path |

So the "pure" list combinators are **inconsistent today**: four refuse effectful callbacks outright, six
accept them *and* leak (V7). The same leak reproduces with user modules imported cross-module whenever the
HOF is recursive (`anyR`), forwards the callback to another HOF (`applyVia`), or forwards to a stdlib HOF
(`fm = flatMap(f, xs)`) — V8.

**#573 shape — calls through annotated function values** (all accepted at base, all perform the effect):

| Variant | base |
|---|---|
| `p.f(0)`, `p: P` (named record type) — the issue's repro | accepted, prints |
| `p: { f: (int) -> () ! {IO} }` (inline record type) | accepted, prints |
| `let g = p.f; g(0)` | accepted, prints |
| `pure func rowless(p: P) -> () { p.f(0) }` | accepted, prints |
| `func rowless(p: P) -> () ! {FS} { p.f(0) }` — **declares the wrong effect** | accepted (only `main` is blamed, for `FS`) |
| `func callIt(f: (int) -> () ! {IO}) -> () { f(0) }` — plain annotated **parameter**, no record | accepted, prints |
| ADT payload: `type Handler = OnInt(int -> () ! {IO}) \| NoHandler`; rowless `fireH` matches `OnInt(f) => f(1)` (addendum "case G" position) | accepted, prints |
| record update: `fireP({ base() \| f: loud })` where rowless `fireP(p) = p.f(p.n)` (addendum "case E" position) | accepted, prints |

The `callIt` row matters: #573 is not about records. Any function-typed *annotation* loses its labels.

## Root Cause

### V-log: what the checker sees (instrumented with a temporary `DEBUG_EFFECTS` probe, then reverted)

`DEBUG_EFFECTS=1` plus a one-line probe printing `typeInfo.Get(e.Func.ID())` at every App
(passing inputs must use `AILANG_NO_CACHE=1`; a passing file is content-cached and prints nothing — the V38
trap recorded in #616's doc, re-hit here):

```
# R1, function `sneaky`
Callee declared effects (from signature): []
TMPDBG callee CoreTI type: (int -> int ! {IO}, int) -> int ! {IO}      <- the TYPE CHECKER got it right
Var(logIt) -> []                                                       <- the argument contributes nothing
App total effects: []

# R2, function `s`
TMPDBG callee CoreTI type: ((int, int) -> int ! {IO}, list[int]) -> list[int]
Callee type effects (from CoreTypeInfo): []
Var(cmpLog) -> []

# R3, function `rowless`
TMPDBG let $tmp2 value(*core.RecordAccess) CoreTI: int -> () ! {...ρ3}  <- field's {IO} is gone
TMPDBG   receiver CoreTI: P
Callee type effects (from CoreTypeInfo): []

# R3 variant with an inline record param type
TMPDBG   receiver CoreTI: { f: int -> () ! {...ρ3} }                   <- the PARAMETER ANNOTATION lost {IO}
```

### Cause 1 (#1326): `*core.Var` values carry no latent effect

`internal/pipeline/validate_effects.go` `collectRequiredEffects`:

| Line | Case | Contribution of a function **value** |
|---|---|---|
| 311-314 | `*core.Var` (same-module names, params, locals) | **`nil`, always** |
| 316-325 | `*core.VarGlobal` (imported names) | `extractEffectFromType(CoreTI)` — its latent labels |
| 330-335 | `*core.Lambda` (literal) | its body's effects |

A function value's latent effect is therefore charged at the point it is *referenced* — unless it is
spelled as a same-module name. V5 shows the three spellings of R2 side by side: imported `logC` → rejected,
lambda literal → rejected, same-module `cmpLog` → accepted.

The App case (`:337-379`) cannot recover it either: for a same-module callee it prefers the declared row
(`:344-353`, introduced by `71b610d68` to avoid CoreTypeInfo contamination on recursive calls), and
`applyTo`'s declared row is empty. The instantiated `(int -> int ! {IO}, int) -> int ! {IO}` in CoreTypeInfo
is correct but discarded. For std HOFs (R2) even CoreTypeInfo's outer row is closed-pure, because the
callback row in their exported schemes is *not* linked to the outer row (V6) — contrary to the mechanism
stated in `TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback`'s comment; that test passes because
its callback is a *lambda literal*, which the `*core.Lambda` case charges, not because of row sharing (V6).

### Cause 2 (#573): function-type annotations drop their effect labels

`internal/elaborate/file_funcs.go:386-403`, `(*Elaborator).astTypeToInternalType`, `case *ast.FuncType`:

```go
// Use an open effect row when the annotation doesn't specify effects.
// ...
e.freshVarNum++
openEffectRow := &types.Row{
    Kind:   types.EffectRow,
    Labels: make(map[string]types.Type),
    Tail:   &types.RowVar{Name: fmt.Sprintf("ε_annot%d", e.freshVarNum), Kind: types.EffectRow},
}
```

The comment says "when the annotation doesn't specify effects", but the code runs **unconditionally**:
`typ.Effects` (`internal/ast/ast_type.go:63`) is never read. This converter feeds 19 call sites: parameter
annotations (`file_funcs.go:33`, `:290`), return annotations (`:53`, `:309`), ADT constructor fields and
type aliases for record types (`:182`, `:205`, `:220`, `:242`), and `let` annotations
(`expr_control.go:163`, `:181`, `:283`). The two *other* AST→type converters do read the labels —
`internal/iface/builder.go:47-74` and `internal/types/typechecker.go:216-243` — which is why the call-site
type of `rowless` seen from `main` shows `{ f: int -> () ! {IO} } -> ()` while the body sees `! {...ρ3}`
(a one-concept-two-implementations seam).

Downstream, `extractEffectFromType` (`validate_effects.go:270-300`) returns `nil` for any row with no labels
— including `{| ρ}` — so the call `p.f(0)` is charged nothing (the same tail-dropping behaviour #616's doc
recorded as its V36).

### Do #1326 and #573 share a root cause? — **Yes, one defect class, two code sites**

Both are *"a function value's latent effect is invisible to the only pass that enforces declared rows"*:
the pass either never asks (`*core.Var`, cause 1) or asks a type whose labels the elaborator threw away
(cause 2), and in both cases a residual row variable is read as "pure". Fixing only one site leaves the
other issue open — the prototype arms isolate them cleanly (V16: L2 alone flips every #1326 repro and no
#573 repro; L1 alone the reverse).

### Two further defects found on the way (in scope because the fix leans on the same machinery)

- **D3 — effect-label payload mismatch.** Effect rows built by the builtin `Builder`
  (`internal/types/builder.go:246`: `Labels[eff] = &TCon{Name: eff}`) and by `ElaborateEffectRow*`
  (`internal/types/effects.go:327,426`: `Labels[name] = Unit()`) disagree on the label payload, and
  `RowUnifier` unifies payloads (`internal/types/row_unification.go:74-80`). Today this never fires because
  annotation labels are dropped (cause 2). The moment they are kept, `std/dom.subscribe` →
  `_dom_subscribe` fails: `failed to unify label DOM: cannot unify type constructors: DOM vs ()` (V14).
  **Cause 2 cannot be fixed without fixing D3 first.**
- **D4 — `declaredEffects` is keyed by bare name, ignoring shadowing.** `ValidateEffects` builds
  `declaredEffects[funcDecl.Name]` (`validate_effects.go:117`) and the App case consults it for any
  `*core.Var` callee. A local that shadows a top-level function inherits that function's row:
  `pure func applyStep(step: int -> int, x: int) -> int = step(x)` next to a top-level
  `func step(x: int) -> int ! {IO}` is **rejected today** with `Missing effects: IO` (V15, a false
  positive). The first L2 prototype hit the same bug on `std/ai.dispatchToolCalls` (a pattern variable
  `call` shadows the legacy top-level `call ! {AI}`), which is how it was found.

### Also measured: a pre-existing width hole (not in either issue)

A function declared `! {IO, Env}` can be stored into a field typed `! {IO}`, and a function declared
exactly `! {IO}` then performs `Env` through it (V13, file `e5_wider_into_field.ail`: check passes, run
prints `$HOME`). This is cause 2 again (the field's closed `{IO}` became an open `{| ε}`), but closing it
needs effect *subsumption*, which is the Phase-3 decision below.

## Relation to prior art

| Doc / issue | Relationship |
|---|---|
| [M-EFFECT-ROW-VAR-UNIFICATION](../v1_0_0/m-effect-row-var-unification.md) (#616, **parked**) | Different defect: declared `! {e}` tails are not *discharged* at call sites (its V31–V33: parameter `e` and result `e` are separate instantiated variables). **This design does not require row-variable unification**: both rules charge *concrete labels* that are known without solving any tail. It leaves `UnionEffectRows`/`DiffEffectRows` tail semantics untouched. When #616 lands its A3 interface (`CallEffects[appID]`), rule L2 can be re-expressed as "the instantiated call row", and the `LatentParamMask` proposed here is designed to travel the same channel (one per-App publication, not two). No conflict with its frozen decisions: it freezes the App-case callee-row selection only for tail-bearing declarations; L2 adds argument contributions, it does not change callee selection. |
| #616 doc's "CORRECTION" (from #1091) | Re-confirmed and sharpened: the `VarGlobal` path is correct for *imported named callbacks* (charged at reference, V5), but std HOF schemes do not link callback rows to their outer rows (V6), so "effect-polymorphic by inferred sharing" is not what makes `flatMap` accept effectful lambdas across modules. |
| [M-EFFECT-PURE-ROW-OVERGENERALIZATION](../v0_35_3/m-effect-pure-row-overgeneralization.md) (#1091) | Unaffected. `closeDeclaredEffectRow` was disabled in a probe (V10) and the leaks persist, so it is not the cause; the fix does not touch it. |
| M-EFFECT-ROW-SHOW-INTERP (#386) lambda-annotation sub-pass (`validateLambdaAnnotations`) | Unaffected; it still enforces closed inline-lambda annotations. |
| `design_docs/implemented/v0_29_0/m-effect-row-poly-params.md` | Type-layer unification of lambda closed rows vs. concrete rows; no overlap. |
| `design_docs/implemented/v0_6_2/m-bug-effect-checker-conflation.md` / commit `71b610d68` | Origin of "prefer declared rows for same-module callees". Kept; D4 fixes its name-keying. |
| `TestEffectCeiling_OwnTypesAndCallbacksNotCharged` (package effect ceilings) | Pins that a *stored, never-called* callback and a hook typed wider than its body are **not** charged. Both rules below respect it (V19); a naive "charge every reference" rule breaks it (V18). |

## Goals

**Primary goal:** a function whose declared row omits an effect cannot perform that effect through a
function value — whether the value was passed as an argument, read from a record field, or received as a
parameter.

**Success metrics:**
- R1, R2, R3 and every row of both "how wide" tables are rejected by `ailang check` with `Missing effects:` naming the effect.
- The pinned polymorphism tests stay green: `TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback`, `TestEffectCeiling_OwnTypesAndCallbacksNotCharged` (3 subtests), `TestPureImporter_StillRejectedForGenuineEffect`.
- Zero regressions over `examples/` (425 files) and all 49 `std/` modules; zero std signature changes.
- D4's false positive (V15) becomes an acceptance.

## Non-Goals

- **Discharging declared row variables (`! {e}`)** — that is #616. Nothing here reads or solves a tail.
- **Making std pure combinators uniform.** `map`/`filter`/`foldl`/`takeMap` reject effectful callbacks while six siblings accept them (V7). Whether they should all be effect-transparent or all closed is a library-API decision; this doc only makes the transparent ones *sound*. Filed as a follow-up.
- **Runtime changes.** The capability layer stays the backstop; no evaluator/VM change.
- **Effect inference for unannotated top-level functions** (a rowless `func` still means "pure").

## Solution Design

### Overview — two rules and a prerequisite

| Id | Rule | Fixes | Site |
|---|---|---|---|
| **L0** | One canonical effect-label payload; effect-row label payloads are not unified | prerequisite for L1 (D3) | `internal/types/builder.go:246`, `internal/types/row_unification.go:74-80` |
| **L1** | A function-type annotation keeps its declared labels | #573, addendum constructor/record-update positions, `callIt` | `internal/elaborate/file_funcs.go:386-403` |
| **L2** | Passing a function *value* to an **effect-polymorphic parameter** charges the caller with the value's latent labels | #1326 (local and std HOFs) | `internal/pipeline/validate_effects.go` App case (`:367-373`) |
| **D4** | Resolve `declaredEffects` by binding, not by bare name | V15 false positive; prerequisite for L2 | `validate_effects.go:110-119, 344-353` |

### L1 — keep annotation labels (the #573 fix)

In `(*Elaborator).astTypeToInternalType`, `case *ast.FuncType`: when `len(typ.Effects) > 0`, build the row
from the annotation (labels + params/budgets via `types.ElaborateEffectRowWithBudgets`, row-variable tail if
the author wrote `! {e}`); only an **unannotated** arrow keeps today's fresh `ε_annotN` tail.

Whether the resulting row is **open** (`{IO | ε}`) or **closed** (`{IO}`) is the one real design choice here —
see Phase 3 and the Design Freeze. The prototype measured both:

| Variant | #573 repros | Width hole V13 | `TestEffectCeiling_OwnTypesAndCallbacksNotCharged` | examples / std |
|---|---|---|---|---|
| **L1-open**: labels + fresh tail | rejected ✓ | still accepted ✗ | green ✓ | 0 regressions ✓ |
| **L1-closed**: labels, no tail | rejected ✓ | rejected ✓ | **3 subtests red** ✗ ("hook typed wider than its body" needs a narrower lambda to fit a wider slot) | not swept |

L1-open is sound for #573 as reported: the call is charged *at least* the declared labels, which is exactly
what the rowless caller omitted. It is not an upper bound, so it cannot catch V13. Upper-bound semantics
need **effect subsumption at the point a function value flows into an annotated slot** — the Koka
"open on use" discipline: every reference to a function value (and every lambda) gets its latent row
extended with a fresh tail, so `{IO | ρ}` unifies with a closed `{IO, Process}` (ρ := `{Process}`) but not
with `{Process}`. That makes L1-closed safe for the pinned hook shapes. It changes `inferVar` /
`inferVarGlobal` / `inferLambda` (`internal/types/typechecker_literals.go:59-160`,
`typechecker_functions.go:22-150`) and interacts with #1091's closure and #386's local solve, so it is
**Phase 3, behind a spike**, not Phase 1.

### L2 — charge latent effects at effect-polymorphic parameters (the #1326 fix)

In the App case, for each argument that is a function value, add that value's **latent labels** to the
application's required row **iff** the callee's corresponding parameter, in the callee's *generalised*
(pre-instantiation) type, has an effect row with an open tail — i.e. an unannotated callback
(`f: a -> b`) or a declared `! {e}` / `! {IO, e}`. Latent labels by argument form:

| Argument form | Latent labels |
|---|---|
| same-module function name (`*core.Var` bound at top level — see D4) | its declared row (`declaredEffects`, contamination-safe per `71b610d68`) |
| imported name (`*core.VarGlobal`) | already charged at reference today (`:316-325`); unchanged |
| lambda literal | already charged (body effects, `:330-335`); unchanged |
| local / parameter / field value | concrete labels of its type (non-empty once L1 lands) |

**Why "at a polymorphic parameter" and not "everywhere a name is referenced":** referencing is not
performing. The naive rule (charge every same-module `*core.Var` of function type, prototype P1) turns
`TestEffectCeiling_OwnTypesAndCallbacksNotCharged/caller-supplied callback stored, never called` red (V18),
and rejects `e3_closed_param_store.ail` — a pure `build() = mkBox(logEvent)` where `mkBox` takes a
**closed-annotated** `f: (string) -> () ! {IO}` and only stores it (V12). A closed-annotated parameter is a
contract the *callee* discharges: once L1 lands, the callee's own `f(x)` is charged `{IO}` in its own body.
An open parameter is the only case where the callee's body sees nothing but a tail, so the only sound place
to charge the effect is the caller.

**Where the callee's generalised parameter rows come from.** CoreTypeInfo holds the *instantiated* callee
type, in which an open callback row has already been unified with the argument's closed row — the
distinction is gone. So the type checker publishes it explicitly, following the existing
`DeclaredLambdaEffectRow` lookup pattern (`validate_effects.go:101`):

- `CoreTypeChecker.LatentParamMask(appID) ([]bool, bool)` — for each argument position, whether the callee's
  scheme type has an open latent row at that parameter. Recorded in `inferApp` from the callee node's
  pre-instantiation scheme (`inferVar`/`inferVarGlobal` already hold it).
- Missing entry for a successfully typed App whose callee is a function = internal invariant violation,
  fail loudly (no silent default — CLAUDE.md principle 2).

This mask is the argument-side half of the per-App publication #616 proposes (`CallEffects[appID]`); the
two should share one struct when #616 is unparked.

### D4 — binding-aware callee resolution

`declaredEffects` lookups (both the existing callee path and L2's argument path) must only fire when the
`*core.Var` refers to the **top-level** binding. Implementation choice deferred (thread a set of locally
bound names through `collectRequiredEffects`, or have the elaborator mark same-module top-level references);
the requirement is that `applyStep` (V15) is accepted and `sneaky` (R1) is rejected.

### L0 — canonical label payload

Pick `Unit()` (used by `ElaborateEffectRow*`, `iface`, and `typechecker.astTypeToType` — 3 of 4 builders) and
change `Builder.Build` (`builder.go:246`) to it; additionally make `RowUnifier` skip payload unification for
`EffectRow`-kind rows, since effect labels carry no payload (params/budgets are separate fields and already
checked by `effectParamsCompatible`). The prototype used only the unifier half (V14); doing both is
belt-and-braces and removes the seam.

### What the stdlib needs

**No std signature changes** (V17: 49/49 modules compile under L0+L1-open+L2). Census of `std/` (script over
all 49 modules): 50 functions take ≥1 function-typed parameter (48 exported); 33 have ≥1 *unannotated*
callback; 26 are exported with an unannotated callback and no outer row (the "pure combinator" family:
`std/list` 11, `std/option` 3, `std/result` 3, `std/string` 3, `std/xml` 6); 13 declare a row variable
(`mapE` family, `std/stream`, `std/smoke`, `std/ai/streaming`). Under L2 every unannotated callback parameter
is effect-polymorphic by definition, so callers of these 33 — not the functions themselves — carry the
callback's effects. Behaviour change visible to std users: a same-module *named* effectful callback passed to
`any`/`findIndex`/`foldr`/`sortBy`/`zipWith`/`flatMap` (and, unprobed, `takeFlatMap`, `std/string` and
`std/xml` folds) now requires the caller to declare the effect — which lambda-literal and imported-name
callbacks already required.

### Migration / breakage

Measured with the prototype (`L0-unifier + L1-open + naive-L2`, i.e. an *upper bound* on L2's breakage since
the naive rule over-charges):

| Corpus | Base | Prototype | Delta |
|---|---|---|---|
| `examples/**/*.ail` (425 files) | 341 pass / 84 fail | 341 pass / 84 fail | **0 files changed status** (V17) |
| `std/` (49 modules, one import probe each) | 49 compile | 49 compile | 0 |
| `go test ./internal/{pipeline,types,elaborate,iface}` | green | green | 0 |

Nothing in `examples/` relies on the hole. **Not measured** (no access in this session, stated as an
estimate): external packages — motoko_agent extensions (`ExtCtx.ports` is a record of effectful
function-valued fields; any rowless hook that calls a port will now be rejected, which is the point of
#573), docparse (`flatMap` with an `! {FS}` *lambda* — unaffected, V19), ailang-parse. Expected breakage is
limited to code that genuinely performs an undeclared effect; the fix for each is the `Suggested fix:`
line the diagnostic already prints. Ship with a CHANGELOG "Breaking (soundness)" entry.

### Implementation Plan

**Phase 0 — prerequisites (~0.5 day)**
- [ ] D4: binding-aware `declaredEffects` resolution; V15 flips to accept; R0 stays rejected.
- [ ] L0: canonical label payload (`builder.go:246`) + payload-free effect-label unification (`row_unification.go`).

**Phase 1 — L1-open, the #573 fix (~1 day)**
- [ ] `file_funcs.go:386-403` builds the row from `typ.Effects` (labels + fresh tail; named `e` tail when written).
- [ ] Regression tests: R3, the six #573 variants, addendum constructor-payload and record-update positions.

**Phase 2 — L2, the #1326 fix (~1 day)**
- [ ] `LatentParamMask(appID)` published by the type checker (fail-loud on a missing entry).
- [ ] App case: charge latent labels of function-valued args at open-row parameters only.
- [ ] Regression tests: R1, R2, the six leaky std HOFs, the three cross-module shapes (`anyR`, `applyVia`, `fm`), `e3` (must be **accepted**), the ceiling test.

**Phase 3 — upper-bound annotations (spike ~0.5 day, then ~1 day if green)**
- [ ] Spike "open on use" for function-value references and lambdas; switch L1 to closed rows; gate on V13 rejected + ceiling test green + examples/std sweep unchanged.
- [ ] If the spike fails the gate, ship Phases 0–2 and file V13 as its own issue with this measurement attached.

**Phase 4 — docs (~0.5 day)**
- [ ] CHANGELOG (Breaking — soundness), `docs/LIMITATIONS.md` if it lists effect caveats, teaching-prompt note: "passing a function to a higher-order function counts as performing its effects".
- [ ] Correct the mechanism comment on `TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback` (V6).
- [ ] `examples/runnable/effect_latent_function_values.ail` + manifest entry (accept arms only; reject arms live in Go tests).

### Files to Modify/Create

- `internal/elaborate/file_funcs.go` — L1 in the `*ast.FuncType` case (~+20 LOC)
- `internal/types/builder.go` — L0 label payload (~2 LOC)
- `internal/types/row_unification.go` — L0 effect-label payload skip (~+5 LOC)
- `internal/pipeline/validate_effects.go` — L2 argument charging, D4 binding-aware lookup (~+60 LOC)
- `internal/types/typechecker_functions.go` — record `LatentParamMask` in `inferApp` (~+40 LOC)
- `internal/types/typechecker_literals.go` — expose the callee scheme to `inferApp` (~+10 LOC); Phase 3 "open on use" (~+40 LOC)
- `internal/pipeline/pipeline_single.go` and `internal/pipeline/pipeline_module_compile.go` — pass the new lookup to `ValidateEffects` (~+4 LOC)
- `internal/pipeline/effect_latent_function_values_test.go` — new, the arm matrix (~300 LOC)
- `examples/runnable/effect_latent_function_values.ail` — new (~30 LOC) + `examples/manifest.json`
- `internal/pipeline/effect_pure_row_overgeneralization_test.go` — comment correction only

## Conflict Surface

Touches `internal/elaborate/`, `internal/types/`, `internal/pipeline/validate_effects.go`. No parser/lexer/AST,
codegen, eval or VM change.

### Syntactic positions touched

None. `! {…}` on a function-type annotation already parses (`ast.FuncType.Effects`); it starts being honoured.

### Semantic positions touched, and what else lives there

| Position | Existing occupants | Interaction | Evidence |
|---|---|---|---|
| `astTypeToInternalType` `*ast.FuncType` | 19 call sites: params, returns, ADT ctor fields, record-type aliases, `let` annotations | All start carrying labels. Unannotated arrows unchanged (fresh tail). Width-subsumption users (hooks wider than body) rely on the *dropped* labels today → L1-open keeps them green; L1-closed needs Phase 3 | V9, V18, V19 |
| Effect-row label payload | Builtins (`TCon`) vs everything else (`Unit`) | Collides as soon as L1 lands (`std/dom.subscribe`) → L0 | V14 |
| App-case argument effects | `argEffects` union of arg sub-expressions | L2 adds latent labels for function values at open parameters only | V12, V16, V18 |
| `declaredEffects` name map | Callee resolution for same-module `*core.Var` | D4 restricts to top-level bindings; fixes a live false positive | V15 |
| `*core.VarGlobal` reference charging | Imported names charged at reference, anywhere | **Unchanged** (keeps its existing over-approximation for stored imported functions; noted, not widened) | V5 |
| `*core.Lambda` body charging | Literal callbacks charged at creation | Unchanged; this is what keeps `flatMap(\p. readOne(p))` in an `! {FS}` function working | V19 |
| Package effect ceiling (`validateEffectCeiling`) | Runs on inferred types before this pass | Reads the same CoreTypeInfo; L1 makes annotated rows visible to it. Ceiling tests green under the prototype | V19 |
| `validateLambdaAnnotations` (#386) | Closed inline-lambda annotations | Unchanged | code read `:152-200` |
| Ghost effects (`Debug`) | `eraseGhostEffects` runs on `required` | L2 contributions pass through it like any other; `Debug` stays transparent | code read `:31-40` |

### Programs that MUST still work (all green under the prototype)

1. `TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback` — `flatMap(\p. readOne(p), paths)` in `! {FS}`.
2. `TestEffectCeiling_OwnTypesAndCallbacksNotCharged` — hook wider than body; stored-never-called callback; sibling reference.
3. `examples/runnable/effectful_list_t1_mapE_basic.ail` — declared `! {e}` combinators.
4. `examples/cognitive_os/single_agent_replay.ail` — `std/dom.subscribe` with a `! {DOM, Cog}` callback (the L0 collision).
5. `e2_hook_builder.ail` — rowless `mkHooks() = { onEvent: logEvent }` (stores, never applies) stays accepted.

### What deliberately changes

- Every program in the two "how wide" tables is rejected (intended).
- A same-module named effectful function passed to an unannotated-callback HOF requires the caller to
  declare its effects — the same rule lambda literals and imported names already follow.
- A function that *applies* an annotated function value (`callIt(f: … ! {IO}) = f(0)`) must declare `IO`.
- (Phase 3 only) Storing a function into a narrower-annotated slot is rejected at the store.

## Examples

**Before → after (R1):**
```
$ ailang check hof_local.ail
✓ No errors found!                      # before
Error: effect checking failed in hof_local: Effect checking failed for function 'sneaky'
  Missing effects: IO                   # after (prototype output)
```

**Still accepted after (storing is not performing)** — excerpt of `e2_hook_builder.ail`, accepted at base and under the prototype:
```ailang
export type Hooks = { onEvent: string -> () ! {IO} }
func logEvent(s: string) -> () ! {IO} = println(s)
export func mkHooks() -> Hooks = { onEvent: logEvent }
```

## Acceptance Criteria

All reject arms: `ailang check` exits 1 and the output contains `Missing effects:` naming the effect **and**
the function; all accept arms exit 0. Every arm runs with `AILANG_NO_CACHE=1` (a passing file is
content-cached, V2 note) and from a fresh temp dir.

- [ ] **AC1** (#1326 local): R1 rejected, names `sneaky`, `IO`.
- [ ] **AC2** (#1326 std): R2 rejected; same for `any`, `findIndex`, `foldr`, `zipWith`, `flatMap` probes.
- [ ] **AC3** (#1326 cross-module): `anyR`, `applyVia`, `fm` shapes rejected at the pure importer.
- [ ] **AC4** (#1326 issue AC3): a caller declaring `! {FS}` that passes an `! {IO}` named callback to `applyTo` is rejected naming `IO`.
- [ ] **AC5** (#573): R3 rejected naming `rowless`; the five other variants in the #573 table rejected, incl. `! {FS}`-declared `rowless` blamed for `IO` (not `main` for `FS`).
- [ ] **AC6** (#573 addendum): `e7_ctor_payload` (ADT constructor payload) and `e8_record_update` (`{ base() | f: loud }`) are rejected naming `fireH` / `fireP` (prototype already does, V21).
- [ ] **AC7** (no over-charging): `e2_hook_builder` accepted; `e3_closed_param_store` accepted; `TestEffectCeiling_OwnTypesAndCallbacksNotCharged` green.
- [ ] **AC8** (polymorphism preserved): `TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback`, `TestPureImporter_StillRejectedForGenuineEffect`, `examples/runnable/effectful_list_t1_mapE_basic.ail` green.
- [ ] **AC9** (D4): `applyStep` (V15) accepted; R0 still rejected.
- [ ] **AC10** (L0): `examples/cognitive_os/single_agent_replay.ail` passes.
- [ ] **AC11** (bytecode): R1 and R3 are rejected before `run --bytecode` executes (no `side effect` / `EFFECT PERFORMED` on stdout).
- [ ] **AC12** (no regressions): the 425-file `examples/` sweep and the 49-module std probe give the base pass/fail sets exactly; `make test`, `make verify-examples` green.
- [ ] **AC13** (Phase 3, only if shipped): V13 rejected at `mk()`; hook-wider test still green.

### Mutation checks (revert the fix, watch the test fail)

The implementer must show each of these RED with the named change reverted and GREEN with it applied:

| Revert | Must turn RED | Must stay GREEN |
|---|---|---|
| L2 argument charging | AC1–AC4 | AC5, AC7, AC8 |
| L1 annotation labels | AC5, AC6 | AC1–AC3 |
| L2 open-parameter gate (charge every function arg) | AC7 (`e3`, ceiling test) | AC1–AC5 |
| D4 binding-aware lookup | AC9 | AC1 |
| L0 payload canonicalisation | AC10 | AC1–AC3 |

The prototype already demonstrated rows 1, 2 and 3 in both directions (V16, V18).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Latent effects are charged **at application** (and at hand-off to an effect-polymorphic parameter), never at mere reference | Settles the stored-hook question motoko pins; reference-charging breaks the ceiling test | design (evidence: V18) | design | high |
| Unannotated callback parameters are effect-polymorphic and their effects belong to the **caller** | Makes 33 std HOFs sound without signature changes; the alternative (close them) breaks docparse's pinned `flatMap` use | design | design | high |
| Annotation rows open (L1-open) vs closed + open-on-use (L1-closed) | Closed is the only variant that makes an annotation an upper bound (V13); it changes inference of every function reference | **human** | before Phase 3 | high |
| Type checker publishes `LatentParamMask(appID)` rather than the validator re-deriving openness | CoreTypeInfo cannot answer it (instantiated); an explicit interface is testable and is the channel #616 wants | design | compile | med |
| Canonical effect-label payload = `Unit()` | 3 of 4 builders already use it | agent | compile | low |

### Design Freeze

- [x] Charge at application / at polymorphic hand-off, not at reference (V18 settles it).
- [x] Unannotated callbacks are caller-charged; no std signature change in this milestone.
- [x] `LatentParamMask` is an explicit, fail-loud type-checker output.
- [ ] **Human:** ship L1-open in Phase 1 and treat the width hole (V13) as Phase 3 behind a spike — or hold the release for L1-closed. (Recommendation: ship Phases 0–2; both reported issues close without Phase 3.)
- [ ] **Human:** confirm the breaking-change posture (minor bump, CHANGELOG "Breaking — soundness", no opt-out flag).

## Deferred Decisions

- D4 mechanism (scope set vs elaborator-marked top-level refs) — agent may choose.
- Exact shape of `LatentParamMask` (bool slice vs row per param) — agent may choose; keep it mergeable with #616's `CallEffects`.
- Whether `extractEffectFromType` should stop mapping `{| ρ}` to pure — **not needed** by this design; leave to #616.
- Whether std's closed-callback combinators (`map`/`filter`/`foldl`/`takeMap`) become effect-transparent — human, separate issue.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| External packages (motoko ext, docparse, ailang-parse) contain rowless code that applies ports/callbacks | Med — they start failing `check` | That code performs undeclared effects; the diagnostic's `Suggested fix:` is correct. Announce in CHANGELOG; motoko is the reporter and asked for this |
| Re-introducing `71b610d68` CoreTypeInfo contamination | High | L2 reads same-module latent rows from **declared** rows, never CoreTypeInfo; the `PureFunctionNotContaminatedByContext` test stays |
| L0 payload change interacts with effect params/budgets | Med | Params/budgets live in separate `Row` fields checked by `effectParamsCompatible`; add a test unifying a builtin row with an annotation row carrying `Rand[mode=seeded]` |
| Phase 3 "open on use" destabilises #1091/#386 row machinery | High | Spike-gated; Phases 0–2 ship independently |
| Content cache hides a regression in tests/sweeps | Med | All arms use `AILANG_NO_CACHE=1` / fresh paths (V2 note) |

## Verification Log

Base: this worktree reset to `origin/dev` = `1fcc479f1`; binary `/tmp/ailang-ds` from `go build ./cmd/ailang`.
Temporary probes and the prototype were built into separate binaries (`/tmp/ailang-dbg`, `/tmp/ailang-proto`)
and **reverted from the tree** before this doc was committed (`git status` shows no source change). The
prototype diff (88 lines, env-gated: `PROTO_LATENT_VAR`, `PROTO_ANNOT_LABELS`, `PROTO_ANNOT_OPEN`,
`PROTO_EFF_PAYLOAD_FREE`) is described in each row; it is evidence, not the implementation.

- **V1** — R1/R2/R3 accepted by `check`, side effect printed by `run --caps IO` (transcripts above).
- **V2** — R0 rejected with `Missing effects: IO`. A passing file re-checked unchanged prints no `DEBUG_EFFECTS` lines (content cache); all probes used `AILANG_NO_CACHE=1`.
- **V3** — `pure func` local HOF (`applyP`) leaks identically to rowless `applyTo`.
- **V4** — `run --bytecode` prints `side effect`/`1` (R1) and `EFFECT PERFORMED` (R3); with the prototype, R1 is rejected before execution.
- **V5** — three spellings of R2: imported `logC` (`xm/logs`) → `Missing effects: IO`; lambda literal → `Missing effects: IO`; same-module `cmpLog` → accepted. Code: `validate_effects.go:311-335`.
- **V6** — CoreTI at an importer's call site: `sortBy` `((int, int) -> int ! {IO}, list[int]) -> list[int]`; `flatMap` `(int -> list[int] ! {IO}, list[int]) -> list[int]` — outer rows closed, callback rows not linked. Within `xm/lib`, `applyTo`/`applyP`/`applyG` link (`… ! {...ρ3}) -> … ! {...ρ3}`) and are correctly rejected cross-module; recursive `anyR` is unlinked even in-module.
- **V7** — std/list probe table (11 functions) as in "How wide the hole is".
- **V8** — cross-module `anyR`, `applyVia`, `fm` from `xm/lib2`: accepted, print `E`.
- **V9** — #573 variants table; debug shows `receiver CoreTI: { f: int -> () ! {...ρ3} }` for an inline param annotation `{ f: (int) -> () ! {IO} }`. Code: `file_funcs.go:386-403` never reads `typ.Effects`; `iface/builder.go:47-74` and `types/typechecker.go:216-243` do.
- **V10** — `closeDeclaredEffectRowForBinding` disabled via probe: `u_via`, `u_anyR`, `h_sortBy` still accepted → #1091's closure is not the cause.
- **V11** — std census: 50 HOFs / 48 exported / 33 with an unannotated callback / 26 exported pure-or-rowless with one / 13 with a row variable.
- **V12** — naive L2 (charge every same-module function-valued argument): `e3_closed_param_store` (`build() = mkBox(logEvent)`, closed `! {IO}` storing param) rejected → false positive; motivates the open-parameter gate.
- **V13** — `e5_wider_into_field.ail`: base check passes, `run --caps IO,Env` prints `$HOME` from `fire ! {IO}`. L1-open: still accepted. L1-closed: rejected at `mk()` (`incompatible closed rows`).
- **V14** — L1 without L0: `single_agent_replay.ail` fails `failed to unify label DOM: cannot unify type constructors: DOM vs ()`; payload sources `builder.go:246` (`TCon`) vs `effects.go:327,426` (`Unit()`), unified at `row_unification.go:74-80`. With the unifier skipping effect-label payloads: passes.
- **V15** — `e6_name_shadow.ail` (`pure func applyStep(step: int -> int, x) = step(x)` beside top-level `step ! {IO}`): base **rejects** with `Missing effects: IO`. Same mechanism made the first prototype reject `std/ai.dispatchToolCalls` (pattern var `call` vs top-level `call ! {AI}`); fixed in the prototype by requiring the arg to be function-typed, properly by D4.
- **V16** — prototype arm matrix over 31 repro files: L2 alone flips R1, R2, `applyP`, `anyR`, `applyVia`, `fm`, six std HOFs; L1 alone flips R3 and the five #573 variants and `callIt`; L1+L2 flips all; no previously-rejected file becomes accepted.
- **V17** — sweep of all 425 `examples/**/*.ail` under base / L2 / L1 / L1+L2: `341 pass / 84 fail` in every arm, identical per file. std probe (one import per module, 49 modules): 49/49 pass in base and L1+L2.
- **V18** — `go test ./internal/pipeline/` with L1-closed: `TestEffectCeiling_OwnTypesAndCallbacksNotCharged` 3 subtests red; with reference-charging P1: the "stored, never called" subtest red.
- **V19** — `go test ./internal/{pipeline,types,elaborate,iface}` with L0+L1-open+L2: all `ok`.
- **V21** — addendum positions: `e7_ctor_payload.ail` (constructor payload) and `e8_record_update.ail` (field installed by `{ base() | f: loud }`) — base: check passes, run prints `E`; L0+L1-open+L2: rejected, `Missing effects: IO` naming `fireH` / `fireP`.
- **V20** — negative-existence: no existing design doc covers #1326/#573 (`grep -rli "1326\|issues/573" design_docs` → none; neural search top match 0.40 < 0.45 gate). No new error code is introduced (the fix reuses `Effect checking failed … Missing effects:`), so no code-allocation check is needed.

## Quorum

Trigger 1 fires (two open Design-Freeze items for a human) and trigger 2 fires (L0 overrides shared row
unification). `ailang design-quorum` was **not** run in this session; run it before sprint planning.

## Related Documents

- [M-EFFECT-ROW-VAR-UNIFICATION](../v1_0_0/m-effect-row-var-unification.md) — #616, parked; complementary (tails vs labels)
- [M-EFFECT-PURE-ROW-OVERGENERALIZATION](../v0_35_3/m-effect-pure-row-overgeneralization.md) — #1091
- [m-bug-effect-checker-conflation](../../implemented/v0_6_2/m-bug-effect-checker-conflation.md) — origin of declared-row preference
- [m-effectful-list-combinators](../../implemented/v0_7_3/m-effectful-list-combinators.md) — `mapE` family
- [m-effect-row-poly-params](../../implemented/v0_29_0/m-effect-row-poly-params.md) — lambda rows vs concrete rows (distinct)
- Issues: [#1326](https://github.com/sunholo-data/ailang/issues/1326), [#573](https://github.com/sunholo-data/ailang/issues/573), [#616](https://github.com/sunholo-data/ailang/issues/616), [#1091](https://github.com/sunholo-data/ailang/issues/1091), [#386](https://github.com/sunholo-data/ailang/issues/386)

## Future Work

- Uniform std combinator policy (all effect-transparent vs all closed).
- Fold `LatentParamMask` into #616's per-App `CallEffects` when that doc is unparked.
- Retire motoko's runtime hermeticity probe once AC5/AC6 ship (their call).

---

**Document created**: 2026-09-28
**Last updated**: 2026-09-28

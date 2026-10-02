# M-ITERATIVE-LIST-REMAINING: complete M-ITERATIVE-LIST — `any`/`findIndex` (O(n²) on the VM, RT_REC_003 on the interpreter) and `foldr` + the `maximum`/`minimum` family (O(n²) + frame caps) delegate to iterative builtins; the Go-codegen engine already runs them iteratively

**Status**: IMPLEMENTED (v0.51.1) — delivered by the std/list iterative-helpers change (`_list_any`, `_list_findIndex`, `_list_foldr`, right-fold extremes; see [m-foldl-cons-cost-model.md](m-foldl-cons-cost-model.md) Implementation Report and changelog `2026-10-02-list-iterative-helpers`), written in parallel with this doc. Original status: PLANNED. Quorum not runnable in this authoring container (no off-Anthropic reviewer credentials; `ailang docs search --neural` falls back to SimHash — see V28). Every code claim below is backed by a source read at the cited lines (tree `07846355`, single-commit clone) or by a live run against the reporter's exact binary (`/usr/local/bin/ailang`, v0.51.0 `b99dd25c…`, matches the reporter's md5 `ed0478cc…` by commit). Run `ailang design-quorum` on this doc before sprint-planning when a credentialed runner is available (convention: [m-foldl-cons-cost-model](../v0_51_2/m-foldl-cons-cost-model.md) header).
**Target**: v0.51.3 (same release train as this folder's sibling [m-vm-adt-tag-check-lowering](m-vm-adt-tag-check-lowering.md))
**Priority**: P1 (High) — real cost bug (12.5 s for `any` over 80k ints on the VM; 81 s over 320k records) + a VM/interpreter divergence + docs that teach an O(n²) call as a short-circuit helper; not a hard blocker (the reporter worked around it)
**Estimated**: ~3 days raw, 2× where honest → ~1 week (Phase 1: 1 d · Phase 2: 0.5 d · Phase 3: 1 d · Phase 4: 0.5 d)
**Dependencies**: None hard. **Coordination required**: [m-interpreter-tail-call-elimination](../v0_51_1/m-interpreter-tail-call-elimination.md) (its code is already in this tree — see V10 — and it changes the divergence half of this bug report); [m-foldl-cons-cost-model](../v0_51_2/m-foldl-cons-cost-model.md) edits the same `std/list.ail` header comments (merge coordination, see Conflict Surface item 6).
**Bug report**: AILANG v0.51.0 (b99dd25, darwin arm64, binary md5 `ed0478cc…`), related to [#1501](https://github.com/sunholo-data/ailang/issues/1501) (consing foldl) and [#676](https://github.com/sunholo-data/ailang/issues/676) (O(1) cons substrate, parked). Downstream consumer: `stapledons-godot` M1.2b-T4 (catalogue tool; **avoided** the bug via `head(filter(...))` / `head(flatMap(...))` — 0.25 s VM / 0.71 s interpreter over 320k records, vs 81–83 s for `any`/`findIndex`, reporter's numbers, V27).
**Author**: design-doc-creator, unattended coordinator session (`AILANG_TASK_TITLE` set; inbox skipped per CLAUDE.md).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Same results, same callback evaluation order (left-to-right with identical early exit; right-to-left for `foldr`), both engines, by construction and pinned by parity + order-recording tests |
| A2: Replayability | 0 | Trace shape changes per the shipped map/filter precedent (helper frames drop, predicate-call events identical in count/order); no effect/trace semantics added |
| A3: Effect Legibility | +1 | All new builtins `IsPure: true`; exported signatures unchanged, so effect rows of every caller are unchanged |
| A4: Explicit Authority | 0 | Pure in-process computation; no capabilities |
| A5: Bounded Verification | 0 | Registry types via the existing builder; no new type-system surface |
| A6: Safe Concurrency | 0 | Single-threaded loops |
| A7: Machines First | +1 | Removes a complexity cliff AI-generated code reliably walks into (predicate helpers at 20k+ elements); docs/`ailang docs` state the true cost |
| A8: Minimal Syntax | +1 | Zero new syntax; delegation is transparent at the existing call sites |
| A9: Cost Visibility | +1 | The headline fix: O(n²)→O(n) on the VM, depth-cap independence on the interpreter; plus D7 makes *which engine ran* visible under `--quiet` |
| A10: Composability | +1 | `any(p, xs)` / `findIndex` compose as before; short-circuit semantics preserved without the `head(filter(...))` workaround |
| A11: Structured Failure | +1 | Type errors are structured per the `_list_map` pattern; callback errors carry the index; empty-list guards stay in typed AILANG wrappers |
| A12: System Boundary | 0 | No boundary crossing |

**Net Score: +6** → **Decision: Proceed to implementation.**

### Hard Violation Check

- [x] A1 (Determinism): no new nondeterminism; evaluation order preserved and tested
- [x] A3 (Effects): pure builtins only; effectful `*E` family untouched
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): removes a machine-facing cliff, adds no human-only convenience

## Routing Lane (per design_docs/PROGRAM.md §4)

**Lane: AILANG fix.** The defect is in the substrate: `std/list.ail` implements `any`/`findIndex`/`foldr`/`maximum*`/`minimum*` as per-element recursion, and on the bytecode VM every `[x, ...rest]` bind lowers to `_list_tail`, which **copies the remaining suffix** (V7, V9) — O(n²) time and allocation churn for the idiomatic predicate helpers the module's own header recommends. No motoko extension (prompt shaping, tool shaping, retrieval) can change that copy or the evaluator's recursion accounting; the reporter's consumer is not behind motoko. This is the documented continuation of [M-ITERATIVE-LIST](../../implemented/v0_9_2/m-iterative-list-builtins.md), whose Non-Goals listed "`foldr`, `flatMap`, `sortBy`: less commonly used at scale. Can add later following the same pattern" — `flatMap` and `sortBy` have since been delegated (#1318); this doc is the "later" for the rest of the list.

## Problem Statement

The reporter's minimal repro (no FS, no packages; reproduced first-party, V1–V3):

```ailang
module anyrepro
import std/list (any, findIndex, range)
import std/io (println)
export func main(n: int) -> () ! {IO} {
  let xs = range(0, n);
  println(if any(\x. x < 0, xs) then "found" else "none");
  println(match findIndex(\x. x < 0, xs) { Some(i) => "found", None => "none" })
}
```

| Run | n=20,000 | n=40,000 | n=80,000 | Shape |
|---|---|---|---|---|
| VM (`--bytecode`), reporter's darwin binary | 0.92 s | 3.24 s | 12.52 s | ×3.5–4 per doubling → O(n²) |
| VM, this container (same binary) | 5.57 s | 28.99 s | — | ×5.2 per doubling → O(n²) (V2) |
| interpreter, reporter / this container | **RT_REC_003 at depth 10000** | — | — | diverges from the VM (V1) |
| interpreter below the cap (n = 2k/4k/8k/9.5k) | 0.47 / 0.60 / 0.74 / 0.71 s | | | ≈ linear (V3) |
| `head(filter(p, xs))` over **320k records** (reporter) | 0.25 s | | | linear — the workaround |

**Mechanism — three engines, three answers for the same helper** (all verified by code read or live run):

1. **Bytecode VM: O(n²) time, O(1) frames.** `any`'s recursive call sits in tail position (`if p(x) then true else any(p, rest)`), so the compiler emits `OpTailCall` and frames are reused — the program *runs* at any n. But the `[x, ...rest]` bind lowers to `_list_tail` (gen/lower/match.go:521-527), whose VM implementation copies the whole suffix on every bind: `tail := make(...); copy(tail, elems[n:])` (internal/vm/builtins.go:373-374, V7). n binds × (n−i) elements each = O(n²) copy + allocation churn (V9).
2. **Tree-walking interpreter: O(n) time, Go-frame depth.** The evaluator's pattern matcher shares the backing array — `tailElements := listVal.Elements[len(p.Elements):]` (internal/eval/eval_patterns.go:245-246, V8) — so traversal is linear, but pre-TCE each recursive call nested a Go frame: RT_REC_003 at the default 10,000 (the reporter's binary, reproduced V1). **At this tree's HEAD, #1486's tail-call elimination is already implemented** (internal/eval/eval_apply.go:1-16, trampoline at :81-187, V10), so `any`/`findIndex` should now run flat on the interpreter too — the *runnability* divergence is fixed by that doc, but both the O(n²) VM cost and the O(n)-with-heavy-frames interpreter cost remain.
3. **Go-codegen engine: already iterative.** `std/list.findIndex` and `std/list.foldr` calls resolve through the GoCodegenSpec registry to hand-written iterative Go helpers (`FindIndex`, `Foldr`, internal/builtins/registry_codegen_list.go:268-280 and :66-77, V12). The compiled-to-Go engine was never quadratic — meaning the three engines disagree with each other *and* the fastest one is the least used.

**The same-shape audit of the rest of std/list** (the reporter asked to check `last`, `nth?`, `contains`, `member`, `dedup`… — all of those already delegate; the *remaining* recursive pure helpers are the scope of this doc, V14/V15):

| Helper | std/list.ail | State |
|---|---|---|
| `any` | :137-141 | **recursive**, tail; VM O(n²), interpreter capped pre-TCE |
| `findIndex` (+ private `findIndexHelper`) | :145-153 | **recursive**, tail; same |
| `foldr` | :53-57 | **recursive**, **non-tail** (`f(x, foldr(...))`); VM O(n²) **and** frame-capped; interpreter depth-capped |
| `maximumInt/minimumInt/maximumFloat/minimumFloat/maximumString/minimumString` | :259-331 | **recursive**, non-tail (scrutinee of an inner `match`); same class as `foldr` |
| `mapE/filterE/foldlE/flatMapE/forEachE` | :198-252 | recursive, **documented** ("Stack-safe for lists up to ~10,000 elements", :193); effectful — **out of scope** (Non-Goals) |
| `reverse` `zip` `map` `filter` `foldl` `sortBy` `take` `range` `drop` `contains` `nth` `last` `flatMap` `takeFlatMap` `takeMap` `member` `dedup` `intersect` `union` `difference` | various | already iterative builtins (V15) — nothing to do |

**Secondary finding — "the VM and interpreter agree" only by a silent fallback.** `foldr`/`maximum*` are non-tail, so on the VM they nest frames until `MaxStack` — which is `DefaultMaxStack = 1000` (internal/vm/vm.go:11-13, hardcoded at `NewVM`, :63-67, **no CLI override exists**, V25). Measured live (V4, V5): `--bytecode --strict-bytecode` at n=1500 dies with `vm: stack overflow`; plain `--bytecode --quiet` at n=1500 *succeeds* — because the runner silently falls back to the evaluator when the VM errors (internal/runner/entrypoint.go:145-157), and the `⚠ bytecode path unavailable … falling back to evaluator` warning is **suppressed by `--quiet`** (:152-154). At n=20,000 the same run prints the *evaluator's* RT_REC_003 — a user benchmarking "the VM" is actually benchmarking the tree-walker. That is a no-silent-fallbacks violation (CLAUDE.md principle 2) hiding a VM/interpreter divergence, addressed as a small severable item (Phase 4) plus the stale comment at vm.go:11-12 claiming `DefaultMaxStack` "matches the evaluator's recursion limit" (it is 1000 vs 10,000, V25).

**Docs claim (ask 2).** `any`'s entry says "short-circuits on first match" with no complexity or depth caveat (std/list.ail:135, rendered verbatim by `ailang docs std/list`, V16), and the module header (:3-7) steers every reader toward "iterative, O(n)" helpers without noting that `any`/`findIndex`/`foldr` are not among them. The current teaching prompt lists `any`/`findIndex` the same way (`ailang prompt`, lines ~632/667-668, V16).

## Goals

**Primary Goal:** Make the remaining recursive pure `std/list` helpers (`any`, `findIndex`, `foldr`, the six `maximum`/`minimum` functions) iterative Go builtins on all three engines, so the same program has O(n) cost and identical behavior on the VM, the interpreter, and Go codegen — completing M-ITERATIVE-LIST.

**Success Metrics:**
- Reporter's repro: `any` + `findIndex` at n=80,000 complete in <1 s on the VM (today: 12.5 s reporter / quadratic, V2/V27); at n=320,000 in single-digit seconds (today: 81–83 s on records).
- The repro at n=20,000 **runs on both engines** with identical output (today: VM 5.6 s, interpreter RT_REC_003 on the reporter's binary).
- `foldr` at n=200,000 completes on both engines (today: VM dies at ~1000 frames or silently falls back; interpreter RT_REC_003 at 10,000).
- Scaling test: doubling n roughly doubles time (measured ratio in [1.5, 3]) for all newly delegated helpers on both engines.
- All existing tests pass (`make test`, `make verify-examples`), with the one deliberate change class (trace shape) updated per Conflict Surface item 7.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Delegation, not in-AILANG rewriting — each helper becomes a thin `std/list.ail` export over a `$builtin` registry builtin, per the shipped M-ITERATIVE-LIST pattern (map/filter/foldl, v0.9.3) | The alternative (hand-rewritten iterative AILANG, e.g. index loop via `nth`) keeps evaluator-frame-per-step cost and adds a second idiom to teach; delegation also gives the VM native loops instead of per-element frame ops | agent (pattern already human-approved twice: M-ITERATIVE-LIST, #1318) | design | low |
| D2: `_list_findIndex` returns `Option[int]` **directly** (no −1 sentinel) | Both engines have ADT-construction precedents (eval: `TaggedValue{CtorName:"Some"}`, internal/builtins/bytes.go:251-258; VM: `NewADT(optionTagSome/None)`, internal/vm/builtins_string.go:473-477, V21), and the existing GoCodegenSpec for `_list_findIndex` already returns Option — a sentinel would create a runtime/codegen signature mismatch | agent | design | low |
| D3: `any`/`findIndex`/`foldr` get **native VM HOF ports** (`__list_any`, `__list_findIndex`, `__list_foldr` in `HOFBuiltinNames`/`HOFBuiltinTable` + impls in internal/vm/builtins_hof.go + parity test) | They take closures and are polymorphic → not adapter-eligible per #1447's static rule (D2 of m-vm-pure-builtin-coverage); without a native port the VM keeps running the recursive AILANG and the fix does nothing for the reporter's `--bytecode` numbers | agent (mechanically forced by #1447 buckets) | design | med |
| D4: The six extremes become **monomorphic raw-value builtins** (`_list_maximumInt(xs) -> int` etc.) with the AILANG wrapper doing the empty-list check via `_list_length` — the `nth`/`last` precedent (std/list.ail:96-105) | Monomorphic, closure-free, ADT-free ⇒ **adapter-eligible** on the VM (zero VM-side code, covered by `AdaptedBuiltinNames`); keeps `Option` out of Go/VM builtin bodies for these six | agent | design | low |
| D5: Tie behavior of the extremes: the iterative loop replaces the current value on `>=` (maximum) / `<=` (minimum), matching today's "later occurrence wins on ties" (std/list.ail:259-269, tie line at :265: `if x > m then Some(x) else Some(m)` keeps `m` from the rest) | Ties are value-identical for int/float/string (unobservable), but pinning the rule keeps the parity test honest and documents intent | agent | design | low |
| D6: `findIndexHelper` is **deleted**, not kept beside the builtin | Dead code after delegation; nothing references it (V24); keeping it doubles the doc surface for the next reader | agent | design | low |
| D7: The VM→evaluator fallback warning stops being `--quiet`-suppressed, and the stale `DefaultMaxStack` comment is corrected (Phase 4) | `--quiet` currently hides *which engine ran the program* (V5) — measured to mislead benchmarking; the comment claims parity with a limit it does not match (V25) | human (observable CLI behavior change, severable if disputed) | design | low |

### Design Freeze

- [x] D1 delegation (pattern shipped twice)
- [x] D2 `Option[int]` return for `_list_findIndex`
- [x] D3 native HOF ports for `any`/`findIndex`/`foldr`
- [x] D4 adapter path for the six extremes (nth-style wrapper)
- [x] D5 tie rule `>=`/`<=` (later-wins), pinned by test
- [x] D6 delete `findIndexHelper`
- [ ] D7 fallback-visibility change — **severable**: if the human prefers a separate doc (it touches `internal/runner`, outside `std/list`), Phase 4 item (a) moves out and this doc keeps only the comment fix (b). Default proposed: land it here, one line + one test.

## Conflict Surface

This change touches `internal/builtins/`, `internal/vm/`, `internal/bytecode/`, `internal/runner/` (Phase 4a only), and `std/list.ail`. Required section per the design-doc-creator skill.

1. **Positions extended and their existing tenants.**
   - `HOFBuiltinNames` (internal/bytecode/builtin_names.go:134-144, 9 entries) / `HOFBuiltinTable` (internal/vm/builtins.go:30-38): order-pinned, validated at init by `validateBuiltinTables` (builtins_adapted.go). New entries append; reordering is a bug, not a choice.
   - Registry name space `$builtin::_list_*`: today's 22 `_list_*` entries enumerated in V11; **no `_list_any`/`_list_findIndex`/`_list_foldr`/`_list_maximum*` exists** (V11, negative verified) — no collision.
   - IR name mangling: the lower pass prepends `_` to `$builtin` references (gen/lower/expr.go:229-233), so registry `_list_any` surfaces as `__list_any` — the doubling is the mechanism, not a typo (V18).
   - `GoCodegenSpec`s keyed by builtin name and `StdlibName`: `_list_foldr` (:66-77) and `_list_findIndex` (:268-280) **already exist with iterative bodies and Option-returning `findIndex`** (V12) — D2 makes the runtime builtin match them instead of colliding. `_list_any` and the six extremes have **no spec today** (V13) — Phase 1/3 add them.
   - `AdaptedBuiltinNames` (computed, internal/bytecode/builtin_adapt.go:170): the extremes join this bucket; `TestPureBuiltinCoverage`'s `unportedPure` ratchet (internal/vm/builtin_coverage_test.go:15-40) must not gain entries and gains none — the new builtins are pure and bucketed (V22).
2. **uint8 index budget.** `OpBuiltinCall` indexes native+adapted in one uint8; the compiler panics above 256 (compiler/builtins.go:34-36). Current estimate ≈102 native + ≈77 adapted ≈ 179 (V26); +6 adapted leaves headroom. The HOF index is a separate uint8 (9 → 12 of 255). The panic is loud if wrong — no silent overflow.
3. **Trace/observability (deliberate change).** Traces record one `function_enter` per AILANG call. Today `any(xs)` emits n `any` frames; after delegation it emits one frame for the wrapper plus one per predicate-closure invocation — the same count and order of *predicate* calls (left-to-right until first true), minus the `any`/`findIndexHelper` frames. Identical precedent: map/filter/foldl delegation in v0.9.3 (A2 scored 0 there). `examples/traces/list_helpers.jsonl` is a committed sample containing `findIndexHelper` events and **will** change; no test compares it (the otel integration test pins `recursion_fibonacci.jsonl` only, V23) — regenerate the sample in the same PR.
4. **Contract/SMT surface: unchanged.** Only `_list_contains` has an SMT sequence rule (internal/smt/codegen_apps.go:317-318); no rule exists for `any`/`findIndex`/`foldr`/extremes (V17) — `ensures` clauses mentioning them were already unencodable and remain so. Delegation must not add an SMT rule (not attempted).
5. **Typechecker surface: unchanged.** Registry types are declared with `types.NewBuilder()` exactly like `_list_map` (forall a b …); the exported `std/list` signatures are bit-identical to today (V18 pattern), so `ailang check` results for every existing program are unchanged. `findIndex` keeps `Option[int]`; extremes keep `Option[t]`.
6. **Same-file coordination.** `std/list.ail` header comments are also edited by [m-foldl-cons-cost-model](../v0_51_2/m-foldl-cons-cost-model.md) Phase 1 (different lines: header vs. per-function comments, but the same hunk region for `foldr`/`concat`). Land order or a merge: whoever lands second rebases; both docs' edits are comments + one-line bodies, so conflicts are textual only.
7. **Programs that MUST still work (regression fixtures, all verified to exist):**
   - `tests/codegen-harness/list_ops.ail:12` — `testAny` over `[int]` (goes through delegation).
   - `examples/runnable/list_helpers.ail:43-48` — `any` ×3 (verify-examples).
   - `internal/vm/builtins_list_poly_test.go`, `builtins_hof_test.go` — the native↔evaluator parity pattern the new ports must join.
   - `tests/golden/codegen/` golden corpus and `tests/golden/bytecode/` — the delegated bodies lower to `BuiltinCall`s the corpus already exercises for map/filter.
   - Any user code calling `any`/`findIndex`/`foldr`/`maximum*` polymorphically over records/ADTs — the HOF ports move `Value`s without inspecting them (`CallClosure` per element), same as `__list_map` (internal/vm/builtins_hof.go:15-35).
8. **What deliberately changes:** per-element helper frames disappear from traces (item 7's counterpart); the O(n²) cost class disappears; `--quiet` no longer hides an engine fallback (D7, if not severed); `DefaultMaxStack`'s comment stops claiming evaluator parity.

## Solution Design

### Overview

Five pure helpers/families become iterative Go builtins following the four-surface pattern established by M-ITERATIVE-LIST (registry Impl + VM port + GoCodegenSpec + thin `std/list.ail` delegation). `any`/`findIndex`/`foldr` take closures → native VM HOF ports; the extremes are monomorphic → VM adapter bucket, `nth`-style wrapper. Evaluation order of user callbacks is preserved exactly (see each builtin), so results, errors, and trace-visible callback order are identical to the recursive implementations.

### Architecture

**New builtins** (evaluator side in a new `internal/builtins/list_search.go`, registration via `RegisterEffectBuiltin`, `IsPure: true`, types via `types.NewBuilder()` — the `_list_map` template, internal/builtins/list_iterative.go:29-77):

| Builtin | Type | Body (Go) | Callback order |
|---|---|---|---|
| `_list_any` | `forall a. (a -> bool, [a]) -> bool` | `for i, e := range elems { if FnCaller(fn, e) → true, return true }; return false` | left-to-right, **stops at first true** — identical to recursive |
| `_list_findIndex` | `forall a. (a -> bool, [a]) -> Option[int]` | same loop; on true return `Some(int64(i))`; else `None` (ADT construction per D2) | identical |
| `_list_foldr` | `forall a b. ((a, b) -> b, b, [a]) -> b` | `for i := len-1; i >= 0; i-- { acc = FnCallerN(fn, [elems[i], acc]) }` | right-to-left, innermost-first — **identical order to the recursive `f(x, foldr(...))`** |
| `_list_maximumInt` / `minimumInt` / `maximumFloat` / `minimumFloat` / `maximumString` / `minimumString` | `[t] -> t` (monomorphic; errors on empty) | single max/min loop, `>=`/`<=` replace (D5) | no callbacks |

The `any`/`findIndex` loops use the `_list_takeFlatMap` early-exit idiom (break/return inside the loop, internal/builtins/list_bounded.go:186-201, V20); `callbackErr` wraps callback failures and passes RT_REC_003 through unchanged (internal/builtins/callback_error.go:11, V19).

**std/list.ail delegations** (Option wrappers follow the shipped `nth`/`last` style, std/list.ail:96-105):

```ailang
-- any: Check if any element satisfies predicate. Iterative builtin
-- (M-ITERATIVE-LIST-REMAINING): O(n) time, O(1) stack on both engines,
-- short-circuits on the first element where p holds.
export pure func any[a](p: a -> bool, xs: [a]) -> bool = _list_any(p, xs)

-- findIndex: index of the first element matching p. Iterative builtin:
-- O(n), O(1) stack, stops at the first match.
export pure func findIndex[a](p: a -> bool, xs: [a]) -> Option[int] = _list_findIndex(p, xs)

export pure func foldr[a, b](f: (a, b) -> b, acc: b, xs: [a]) -> b = _list_foldr(f, acc, xs)

-- nth-style guard for the extremes (empty → None, never calls the erroring builtin)
export pure func maximumInt(xs: [int]) -> Option[int] {
  if _list_length(xs) == 0 then None else Some(_list_maximumInt(xs))
}
-- (×6, same shape)
```

`findIndexHelper` (std/list.ail:149-153) is deleted (D6).

**VM side:**
- Native ports (D3): append `__list_any`, `__list_findIndex`, `__list_foldr` to `HOFBuiltinNames` (internal/bytecode/builtin_names.go) **and** `HOFBuiltinTable` (internal/vm/builtins.go), implementations in `internal/vm/builtins_hof.go` using `caller.CallClosure` — `hofBuiltinListFoldr` mirrors `hofBuiltinListFoldl` with the reversed loop; `__list_findIndex` returns `bytecode.NewADT(optionTagSome/None, …)` (builtins_string.go:473-477 idiom, V21). Add them to the parity test (`builtins_hof_test.go` / the `builtins_list_poly_test.go` mirror) so evaluator and VM stay locked.
- Extremes (D4): **no VM code** — pure, monomorphic, closure-free, ADT-free ⇒ they land in the `AdaptedBuiltinNames` bucket automatically; `TestPureBuiltinCoverage` passes with `unportedPure` unchanged (V22).
- `__list_takeMap`/`__list_takeFlatMap` remain in `unportedPure` (`ReasonClosure`) — **not** widened here (Future Work).

**Go-codegen side:** add `GoCodegenSpec`s for `_list_any` (early-exit loop) and the six extremes (plain loop) in `internal/builtins/registry_codegen_list.go`, mirroring the existing `_list_findIndex`/`_list_foldr` entries (V12); the `findIndex`/`foldr` specs already match the new runtime shapes (D2).

**Phase 4 (severable, D7):** (a) internal/runner/entrypoint.go:152-154 — the `⚠ … falling back to evaluator` line moves out from under `if !params.Quiet` (quiet may drop banners, not *which engine executed*); (b) fix the stale comment at internal/vm/vm.go:11-12 to state the actual caps (VM 1000, evaluator 10,000) and that non-strict `--bytecode` falls back to the evaluator past the VM cap.

### Implementation Plan

**Phase 1: `any` + `findIndex`** (~1 day)
- [ ] Registry Impls + types (`internal/builtins/list_search.go` + `_test.go`)
- [ ] VM HOF ports + parity tests; ratchet check (`TestPureBuiltinCoverage`)
- [ ] `GoCodegenSpec` for `_list_any`; verify `_list_findIndex` spec resolves direct-name calls
- [ ] `std/list.ail` delegation + comment rewrite; delete `findIndexHelper`
- [ ] Fixtures: `list_ops.ail`, `list_helpers.ail` still pass; regenerate `examples/traces/list_helpers.jsonl`
- [ ] Bench: reporter's repro at 20k/40k/80k/320k, both engines, ratios recorded

**Phase 2: `foldr`** (~0.5 day)
- [ ] Registry Impl (FnCallerN, right-to-left); VM HOF port + parity; codegen spec exists — add direct-name test
- [ ] `std/list.ail` delegation; fold-order test (non-commutative f, e.g. string concat) proving identical result and callback order

**Phase 3: extremes ×6** (~1 day)
- [ ] Six monomorphic Impls (shared Go helper, `>=`/`<=` rule); adapter eligibility verified by `TestPureBuiltinCoverage`
- [ ] `GoCodegenSpec`s; `std/list.ail` wrappers (nth-style guards); tie test; empty-list test

**Phase 4: observability + docs** (~0.5 day)
- [ ] D7(a) fallback line + test asserting it appears under `--quiet` (or the human severs it — see Freeze)
- [ ] D7(b) stale comment fix
- [ ] `std/list.ail` header: name the full iterative set; complexity notes on `any`/`findIndex`/`foldr`/extremes
- [ ] Teaching prompt: one-line complexity note for `any`/`findIndex` — routed to prompt-manager lane per M-FOLDL-CONS precedent (this doc does not edit prompts)
- [ ] CHANGELOG entry; `make simplicity-audit` re-run (no new opcodes — `OpBuiltinCallHOF` already exists — but builtin count moves)

### Files to Modify/Create

**New files:**
- `internal/builtins/list_search.go` — registry Impls + types for 3+6 builtins (~220 LOC)
- `internal/builtins/list_search_test.go` — unit + 200k stress + callback-error + tie/empty tests (~250 LOC)

**Modified files:**
- `internal/bytecode/builtin_names.go` — +3 `HOFBuiltinNames` entries (~3 LOC)
- `internal/vm/builtins.go` — +3 `HOFBuiltinTable` entries (~3 LOC)
- `internal/vm/builtins_hof.go` — `hofBuiltinListAny`/`FindIndex`/`Foldr` (~90 LOC)
- `internal/vm/builtins_hof_test.go` or `builtins_list_poly_test.go` — parity tests (~80 LOC)
- `internal/builtins/registry_codegen_list.go` — `_list_any` + 6 extremes specs (~70 LOC)
- `std/list.ail` — 5 delegations + 6 wrappers + comments; delete `findIndexHelper` (~40 LOC)
- `internal/runner/entrypoint.go` — D7(a) (~2 LOC)
- `internal/vm/vm.go` — D7(b) comment (~2 LOC)
- `examples/traces/list_helpers.jsonl` — regenerate (trace shape)
- CHANGELOG.md / changelogs/v0.32-current.md

**Estimated total:** ~760 LOC (≈420 implementation, ≈330 tests), all following shipped patterns.

## Examples

### The reporter's repro, after

```console
$ ailang run --quiet --bytecode --caps IO --args-json 80000 anyrepro.ail   # ~0.3 s (was 12.5 s)
$ ailang run --quiet --caps IO --args-json 20000 anyrepro.ail              # "none/none" (was RT_REC_003)
```

Both engines, same output, same wall-clock class — the ask-(3) agreement, plus the cost fix (ask 1).

### Before / after in std/list.ail

```ailang
-- Before (recursive, tail; VM copies the suffix every bind, O(n²)):
export pure func any[a](p: a -> bool, xs: [a]) -> bool {
  match xs { [] => false, [x, ...rest] => if p(x) then true else any(p, rest) }
}

-- After (iterative builtin; short-circuit preserved, O(n), O(1) stack):
export pure func any[a](p: a -> bool, xs: [a]) -> bool = _list_any(p, xs)
```

### Workaround that becomes unnecessary

`head(filter(p, xs))` scans and *allocates* the whole filtered list (no early exit); `_list_any`/`_list_findIndex` return on the first match — strictly cheaper, and the docs stop steering users to the workaround.

## Success Criteria

- [ ] AC-0: re-derive V1–V6 measurements with a freshly built binary from the PR branch (this doc's container had no Go toolchain — V28)
- [ ] `any`/`findIndex` at n=80,000 < 1 s on `--bytecode` (was 12.5 s); scaling ratio per doubling in [1.5, 3]
- [ ] Same repro at n=20,000 produces identical output on both engines (interpreter no longer diverges — re-verify against HEAD where #1486 TCE already flattened `any`; the builtin path removes the dependence entirely)
- [ ] `foldr` at n=200,000 completes on `--bytecode --strict-bytecode` (no stack overflow, no silent fallback)
- [ ] Extremes over 200k elements complete on both engines; `TestPureBuiltinCoverage` unchanged-or-smaller `unportedPure`
- [ ] Parity tests: VM ports vs evaluator Impls agree on values, callback order, and error text for every new builtin (incl. non-commutative `foldr` order test)
- [ ] `make test`, `make verify-examples`, `make lint`, `make check-boundaries` pass; golden corpus unchanged except the regenerated trace sample
- [ ] Docs: `ailang docs std/list` shows complexity notes for the delegated helpers; header names the iterative set
- [ ] CHANGELOG updated

## Testing Strategy

**Unit tests** (`list_search_test.go`): empty/single/many; short-circuit invocation-count test for `any`/`findIndex` (counting predicate asserts exactly k+1 calls for a match at index k); `foldr` right-to-left order with a non-commutative step and an order-recording callback; extremes tie (`[5,5]`, `["a","a"]`) and empty (wrapper returns None, builtin never called); type-error cases (non-list, non-closure); callback error propagates with index in message; 200k stress per builtin (depth-free).

**Integration:** both engines on the reporter's repro at 20k/40k/80k; `--strict-bytecode` variants (no fallback allowed); `--emit-trace` diff of predicate-call events before/after (count and order identical); `tests/codegen-harness/list_ops.ail` and `examples/runnable/list_helpers.ail`.

**Parity:** VM HOF port vs evaluator Impl on shared cases (the `builtins_list_poly_test.go` mirror contract, V22).

**Manual:** `ailang docs std/list` renders the new comments; `ailang builtins list` shows the new `_list_*` names; `ailang prompt` unchanged this PR (routed separately).

## Deferred Decisions

- File split for the new registry code (`list_search.go` vs extending `list_iterative_more.go`) — agent may choose.
- Whether the six extremes share one Go impl behind a small comparison-func table or six explicit loops — agent may choose (prefer one table).
- Exact `LongDesc`/`Tags` metadata wording for the new builtin specs — agent may choose (follow `_list_map`'s).
- Whether `tail` (std/list.ail:33-37) also gets a complexity note now (`[_, ...rest] => rest` is one O(n) copy on the VM) — agent may choose; no builtin change.
- If D7(a) is severed: whether the fallback line deserves its own tiny doc — human decides at review.

## Non-Goals

- **Effectful combinators (`mapE`/`filterE`/`foldlE`/`flatMapE`/`forEachE`)** — recursive and documented as such (std/list.ail:193); iterative *effectful* builtins need effect-context threading through callbacks and remain M-ITERATIVE-LIST's explicit non-goal. (Their GoCodegenSpecs already delegate to iterative helpers, registry_codegen_list.go:282-300.)
- **The O(1)-cons substrate** (`::`/`++` copy cost) — parked on a human decision as D-19 of [m-list-cons-quadratic](../m-list-cons-quadratic.md); this doc does not reopen it. `foldr` with a consing step stays O(n²) *time* for the same reason `foldl` does (see [m-foldl-cons-cost-model](../v0_51_2/m-foldl-cons-cost-model.md)); this doc fixes the *traversal* cost, not the step cost.
- **Interpreter tail-call elimination** — already implemented in this tree (#1486); this doc only re-measures its interaction.
- **Changing the VM's `MaxStack` policy** (raising 1000 to 10000, or adding a flag) — flagged, not designed; the iterative builtins make stdlib users independent of the cap, which is the right fix for this report.
- **`_list_takeMap`/`_list_takeFlatMap` VM ports** — same `ReasonClosure` bucket, independent value; Future Work.

## Timeline

**Week 1** (~3 days): Phases 1–2 (any/findIndex/foldr end-to-end on all engines + benches).
**Week 2** (~1.5 days): Phase 3 extremes + Phase 4 observability/docs; full test matrix, simplicity-audit, CHANGELOG.
**Total: ~4.5 focused days; padded ×2 ≈ 1.5 weeks** (matches the 2× rule for a four-surface change with parity tests).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| uint8 index budget exceeded (native+adapted > 256) | Med | Estimate ≈185/256 after +6 (V26); the compiler panics loudly at 256 (compiler/builtins.go:34-36) — a build-time, not runtime, failure; fallback if hit: extremes get native ports in the same space or Phase 3 defers |
| Trace-shape change breaks a consumer pinning helper frames | Low | Precedent: map/filter/foldl delegation shipped identically (v0.9.3); the committed sample regenerates in-PR; no CI test pins `list_helpers.jsonl` (V23) |
| `#1486` TCE interaction (builtin path vs trampoline) | Low | The builtin path never enters AILANG recursion — no interaction; AC re-verifies `any` at HEAD on both engines with the default limit |
| D7(a) turns a cosmetic warning into noise for `--quiet` users | Low | It only prints when the engine *changed*, which is exactly what `--quiet` must not hide; human may sever (Freeze) |
| Merge conflict with m-foldl-cons-cost-model on `std/list.ail` comments | Low | Textual only; both docs' hunks are disjoint lines in the same file (header vs function bodies); second-lander rebases |
| Adapter eligibility for extremes changes shape (someone later adds an ADT to their result) | Low | D2's static rule rejects it at compile time, loudly (unportedPure bucket grows visibly) |

## Related Documents

**Implemented (informs design):**
- [m-iterative-list-builtins](../../implemented/v0_9_2/m-iterative-list-builtins.md) (v0.9.2) — the delegation pattern, `FnCaller`/`FnCallerN`, and the non-goal this doc completes
- [m-vm-pure-builtin-coverage](../../implemented/v0_51_0/m-vm-pure-builtin-coverage.md) (v0.50.2/v0.51.0) — the native/adapted/unported bucket machinery and ratchet this doc rides
- [m-gap6-stdlib-maximum](../../implemented/v0_7_0/m-gap6-stdlib-maximum.md) (v0.7.0) — where the recursive `maximum`/`minimum` family came from

**Planned (check for overlap):**
- [m-interpreter-tail-call-elimination](../v0_51_1/m-interpreter-tail-call-elimination.md) — fixes the RT_REC_003 *runnability* half of this bug report for tail shapes (in-tree); this doc removes the cost half and the non-tail helpers. Distinct, coordinated.
- [m-foldl-cons-cost-model](../v0_51_2/m-foldl-cons-cost-model.md) — the *step* cost of fold-consing + docs honesty for foldl/mapAccumL; distinct (traversal vs step), same file edited.
- [m-list-cons-quadratic](../m-list-cons-quadratic.md) (PARKED) — the substrate representation decision; explicitly not reopened.

## References

- [Design Axioms](/docs/references/axioms) — scoring below
- [#1501](https://github.com/sunholo-data/ailang/issues/1501), [#676](https://github.com/sunholo-data/ailang/issues/676)
- `changelogs/v0.32-current.md` — #1317 (recursion-depth memory), #1447 strict-VM builtin coverage

## Verification Log

Every load-bearing claim above, with how it was checked in this session. "Reporter" = inherited from the bug report (binary md5-matched by commit `b99dd25`); "first-party" = run/read in this container; "read" = source lines at tree `07846355`.

| # | Claim | How verified |
|---|-------|---------------|
| V1 | Interpreter `any`/`findIndex` at n=20,000 → `RT_REC_003: max recursion depth 10000 exceeded` | First-party: reporter's repro (verbatim, module line fixed) run against `/usr/local/bin/ailang` v0.51.0 b99dd25; full error transcript in session |
| V2 | VM timings quadratic: 5.57 s @20k, 28.99 s @40k (×5.2 per doubling) | First-party, same binary, `time` around `ailang run --quiet --bytecode`; matches reporter's 0.92/3.24/12.52 s shape on slower container CPU |
| V3 | Interpreter below cap ≈ linear: 0.47/0.60/0.74/0.71 s at n=2k/4k/8k/9.5k | First-party timings (≈0.25 s process startup included) |
| V4 | `foldr`+`maximumInt` at n=1500, `--bytecode --strict-bytecode` → `Error: bytecode execution failed: vm: vm: stack overflow` | First-party (foldrepro.ail in session) |
| V5 | Same at n=1500 non-strict `--quiet` → succeeds via silent evaluator fallback; at n=20,000 → evaluator's RT_REC_003; warning suppressed by `--quiet` | First-party runs + read: internal/runner/entrypoint.go:145-157, `if !params.Quiet` at :152 |
| V6 | `any`/`findIndex` compile and run natively on the VM (quadratic cost is the VM's own) | First-party: strict run fails only on `println` (EvalOnly, no bridge in strict) — error names `anyrepro.main` `CALL ip 18`; plus V2's quadratic shape matching the `_list_tail` copy |
| V7 | VM `_list_tail` copies the suffix | Read: internal/vm/builtins.go:354-375 (`tail := make(...)`/`copy(tail, elems[n:])` at :373-374) |
| V8 | Interpreter pattern bind shares the backing array (no copy) | Read: internal/eval/eval_patterns.go:245-246 |
| V9 | VM-path list-pattern rest binding lowers to `_list_tail` | Read: internal/gen/lower/match.go:521-527 (bindings), :405-443 (length/tag conds via `_len`/`_list_get`) |
| V10 | Interpreter TCE (#1486) is implemented at this tree | Read: internal/eval/eval_apply.go:1-16 (header comment cites the v0_51_1 doc) + trampoline :81-187 (`evalCoreT`, `tailCall`, frame replacement); cannot execute HEAD (no Go toolchain, V28) — implementer re-verifies as AC-0 |
| V11 | No `_list_any`/`_list_findIndex`/`_list_foldr`/`_list_maximum*`/`_list_minimum*` runtime builtin exists; today's `_list_*` set enumerated (22 entries) | First-party: `ailang builtins list \| grep _list` (transcript in session); all pure, `$builtin` module |
| V12 | GoCodegenSpecs already exist and are iterative for `_list_foldr` and `_list_findIndex` (Option-returning) | Read: internal/builtins/registry_codegen_list.go:66-77, :268-280 (also `_list_last` :258-266; `mapE`/`forEachE` :282-300) |
| V13 | No `GoCodegenSpec` for `_list_any` or the six extremes | Grep: `"__list_any"\|"_list_any"\|_list_maximum"` over internal/builtins/registry_codegen*.go → no hits (negative existence) |
| V14 | Recursive helpers + line numbers: `any` :137-141, `findIndex` :145-147 + `findIndexHelper` :149-153, `foldr` :53-57, extremes :259-331, `*E` :198-252 with NOTE :193 | Read: std/list.ail (full file) |
| V15 | Reporter's suspects already iterative: `reverse` :29-30, `zip` :42-43, `map` :48, `filter` :50, `foldl` :52, `contains` :88, `nth` :96-98, `last` :100-104, `zipWith` :164, `flatMap` :175, `takeFlatMap` :182, `takeMap` :188, `member` :334, `dedup` :337, `intersect`/`union`/`difference` :340-348 | Read: std/list.ail (line numbers from `grep -n "export pure func" std/list.ail`) |
| V16 | Docs say `any` "short-circuits on first match" with no complexity note; prompt lists any/findIndex likewise | First-party: `ailang docs std/list` output (session) + std/list.ail:135-136; `ailang prompt` lines ~632, 667-668 |
| V17 | Contract/SMT surface unchanged: only `_list_contains` has a sequence rule; none for any/findIndex/foldr/extremes | Read/grep: internal/smt/codegen_apps.go:317-318; `"any"` in internal/smt → comments only |
| V18 | HOF wiring: 9 `HOFBuiltinNames` order-matched to `HOFBuiltinTable` (init-validated); lower mangles `$builtin._x` → `__x`; uint8 index panics > 256; registry/types pattern = `_list_map` | Read: internal/bytecode/builtin_names.go:130-144; internal/vm/builtins.go:25-38; internal/gen/lower/expr.go:229-233; internal/bytecode/compiler/builtins.go:16-38; internal/builtins/list_iterative.go:29-77 |
| V19 | `callbackErr` passes RT_REC_003 through; `FnCaller`/`FnCallerN` nil-checks are the established error shape | Read: internal/builtins/callback_error.go:11; list_iterative.go:74-77, :209-212 |
| V20 | Early-exit loop precedent exists (`_list_takeFlatMap` breaks mid-loop) | Read: internal/builtins/list_bounded.go:186-201 |
| V21 | Option construction precedents both engines: eval `TaggedValue{std/option, Some}`; VM `NewADT(optionTagSome/None)` | Read: internal/builtins/bytes.go:251-258; internal/vm/builtins_string.go:473-477 + tags :461-462 |
| V22 | Coverage ratchet exists and must not grow: `TestPureBuiltinCoverage` + `unportedPure` (24 entries incl. `__list_takeMap`/`__list_takeFlatMap`, `ReasonClosure`) | Read: internal/vm/builtin_coverage_test.go:15-60 |
| V23 | Fixtures exist and are exercised: `tests/codegen-harness/list_ops.ail:12` (`testAny`), `examples/runnable/list_helpers.ail:43-48`; trace sample `examples/traces/list_helpers.jsonl` contains `findIndexHelper` events and is compared by no test | Read: files + grep for `list_helpers.jsonl` (only the otel test reads traces, and it pins `recursion_fibonacci.jsonl`, internal/trace/otel_emitter_integration_test.go:45) |
| V24 | Nothing outside std/list.ail references `findIndexHelper` | Grep over std/, internal/, examples/, tests/ → only the trace sample (data, regenerated) — the private helper is deletable |
| V25 | VM `DefaultMaxStack = 1000`, hardcoded at `NewVM`, no override anywhere; comment claims it matches the evaluator's limit (10,000) | Read: internal/vm/vm.go:11-13, :63-67; repo-wide grep `MaxStack` → vm.go only; RT_REC_003 default 10,000 from V1's message (internal/eval/recursion_limit_error.go) |
| V26 | Index headroom ≈ 102 native + ≈77 adapted ≈ 179/256 | Counted `BuiltinNames` entries (~102) + #1447's counted "~121 of 203 IsPure unported pre-fix" minus the 24-entry ratchet (≈77 adapted) — estimate; the >256 panic (V18) is the loud gate |
| V27 | Reporter's absolute numbers (0.92/3.24/12.52 s @20/40/80k; 81 s findIndex + 83 s any over 320k records; `head(filter)` 0.25 s VM / 0.71 s interpreter) | Reporter-derived (binary commit-matched); container CPU differs (V2) — implementer re-derives at AC-0 |
| V28 | Authoring conditions: no Go toolchain (`which go` empty); live binary = reporter's b99dd25 (`ailang version`); neural doc-search unavailable (SimHash fallback), so the duplicate gate ran as SimHash + direct grep of ITERATIVE/foldl/tail-call docs | Session transcripts; duplicate gate: no ≥0.75 planned/implemented match on this topic (closest are the three coordinated siblings, distinct scopes — see Related Documents) |

## Future Work

- Iterative **effectful** combinators (`_list_mapE`/`_list_foldlE`/…) — needs effect-context threading through callbacks; the GoCodegenSpecs already show the target shape (V12).
- Native VM ports for `__list_takeMap`/`__list_takeFlatMap` (shrink `unportedPure` further; same HOF pattern as this doc).
- VM `MaxStack` policy (flag or evaluator-parity) — only matters for *user* non-tail recursion now; pair with a decision on the fallback visibility (D7).
- `zipWith` avoids one intermediate pair-list by fusing with `zip`+`map` (linear today; constant-factor only).

---

**Document created**: 2026-10-02
**Last updated**: 2026-10-02

# M-VM-LIST-TAIL-SHARE: O(1) list-tail binding on the bytecode VM (cons-pattern walks are O(n²))

**Status**: Planned — quorum attempted 2026-10-09 (`task-006b0353`): **all three external reviewers absent** (`gemini-3-1-pro` unreachable; `oc-kimi-k3` unreachable; `gpt6-1-sol` auth), recorded by name in `.ailang/state/mission-quorum/m-vm-list-tail-share-2026-10-09T11-00-54Z.json`; controller verdict **pass** — degraded to controller-only, **not quorum-cleared**. The 26-row first-party Verification Log is the load-bearing evidence; re-run quorum when a reviewer route is available (same convention as [m-bytecode-nested-pattern-lowering](../../implemented/v0_52_0/m-bytecode-nested-pattern-lowering.md)).
**Target**: v0.53.0
**Priority**: P0 — a 52× engine divergence on the canonical functional list-walk idiom, reported from a downstream consumer mid-PR
**Estimated**: ~2 days (fix is one line ×3 sites + tests + bench; the cost is measurement discipline), 2× where honest → 4 days
**Dependencies**: None. **Coordination required** (text only): [m-list-cons-quadratic](../m-list-cons-quadratic.md) stays PARKED on D-19 — this doc does not touch `::`/`++` construction or the arena decision; `docs/LIMITATIONS.md` rows 41–42 and `std/list.ail`'s header both gain corrected text and must not contradict D-19's pending state.
**Planner-Lane**: opus-required (VM builtin semantics + allocation behavior)
**Bug report**: AILANG v0.52.0 (`bf2436a`, darwin arm64, binary md5 `2dd082832fbadf861b49a1a432a0130c`), `--bytecode`. Downstream consumer `stapledons-godot` (Edenhofer dust-map pass, R1-ISM-DUST PR B); workaround in flight: `foldl`.
**Author**: design-doc-creator, unattended coordinator session (`task-006b0353`). This container has a live `ailang` binary (v0.52.5, commit `7200786`, md5 `8e4464db…` — near-lineage but **not** built from this tree, and **no Go toolchain / no `make`**, shallow single-commit clone). Wall-clock numbers below are first-party runs against that binary; source-line claims are reads at this tree (`59feeccf`, v0.53.0-dev). Provenance marks in the Verification Log.

## Routing Lane (per design_docs/PROGRAM.md §4)

**Lane: AILANG fix.** Runtime cost bug in `internal/vm/`, not reachable by any motoko extension
(prompt shaping cannot change what the VM's `_list_tail` builtin allocates), and the reporter
is not behind motoko. Not the core-floor lane: no motoko-core crash/overflow, no re-freeze. It
carries runtime-core discipline internally (Conflict Surface, verification log, named
regression fixtures) because it touches `internal/vm/builtins.go`.

## Problem Statement

The canonical functional way to walk a list — tail recursion over a cons pattern,
`match xs { [] => …, x :: r => f(r, …) }` — is **O(n²) on the bytecode VM** while the same
program is **O(n) on the interpreter**, because the two engines bind the pattern's tail
variable differently:

- The **pattern lower pass** (`internal/gen/lower/match_pattern.go`) emits
  `_list_tail(s, n)` for every list-pattern tail binding (V4). Its own comments treat
  each `_list_tail` call as a cost to minimize (":145 — the flat form needs one length
  check and no nested `_list_tail` copies") without questioning the copy itself.
- On the **tree-walking interpreter**, list patterns never call that builtin —
  `matchPattern` binds the tail as an **O(1) subslice alias** of the scrutinee
  (`internal/eval/eval_patterns.go:255-257`, V2). The stdlib delegation test says it in
  one line: "_list_tail … is unnecessary in the interpreter because std/list.tail is
  O(1) pattern matching" (V15).
- On the **Go-codegen runtime**, the lowered call compiles to `ListTail`, which returns
  `l[1:]` — an O(1) shared subslice (`internal/gen/golang/codegen_runtime_collections.go:60-67`,
  V3).
- On the **bytecode VM**, `_list_tail` is a native builtin whose implementation **copies
  the remaining elements**: `tail := make(…); copy(tail, elems[n:])`
  (`internal/vm/builtins.go`, `builtinListTail`, V1). One full copy per recursive step →
  O(n²) total. The disassembly of the walk loop shows it plainly: one
  `BUILTIN_CALL builtin#515 (_list_tail), argc=2` per iteration (V16).

**Current State** (all first-party, this session, binary v0.52.5@`7200786`; the reporter's
machine is ~5× faster, so ratios transfer, absolutes do not):

Walk-only repro — same `x :: r` tail recursion as the report, list built **linearly** by
`std/list.range` so construction is out of the picture (V6–V8; program type-checked with
`ailang check`, V24):

```ailang
pure func countGt(xs: [int], thr: int, acc: int) -> int =
  match xs {
    [] => acc,
    x :: r => countGt(r, thr, if x > thr then acc + 1 else acc)
  };
```

| Engine | n=40,000 | n=80,000 | n=160,000 | scaling |
|---|---|---|---|---|
| VM (`--bytecode`; also `--bytecode --strict-bytecode`) | 6.39 s | 24.7 s | **90.9 s** | ×3.9 per doubling → **O(n²)** |
| interpreter (no flag) | 0.85 s | 1.10 s | 1.74 s | ~linear |
| VM, `foldl` with the same step | — | 0.36 s total (≈ 0.34 s hello-world startup baseline) | — | linear, ~free |

The VM is **52× slower than its own interpreter** on this shape at 160k, and ~three orders
of magnitude off `foldl` at scale. The reporter measured the same class on the real
workload: a 786,432-element list decoded by `std/embedding.decodeF32LE` (built in one
pass, V25) walked by cons-pattern recursion took **427 s**; the same count via `foldl`
took **0.08 s** [P].

The reporter's full repro also builds the list with `mk(n, acc) = … 0.25 :: acc` — that
**construction half** is separately quadratic on both engines (`OpCons`/`listConsImpl`
copy the tail: 40k = 6.9 s, 80k = 28.4 s on the VM, V9). That half is **not this doc's
to fix**: it is the parked [m-list-cons-quadratic](../m-list-cons-quadratic.md)
representation decision D-19, whose Non-Goals explicitly freeze VM `OpCons` at O(n) and
keep pattern-tail aliasing untouched. This doc fixes the **consumption half** — the walk —
which is the half that bit the consumer (their list was built linearly by a builtin and
then walked).

**The reporter's alternative hypothesis is refuted**: the tail call is *not* leaking
frames. The walk compiles to `TAIL_CALL` (disassembly, V16), `OpTailCall` reuses the
current frame, and the 160,000-deep walk completes under `DefaultMaxStack = 10,000`
(V10) — constant space, quadratic *time*, purely from the copy.

**Impact**: every AILANG program (human- or AI-written — and this language's primary
audience writes the idiomatic form first) that consumes a list by pattern recursion on
the VM pays n²/2 element copies and the GC cycles they trigger. The consumer's
dust-map pass hit exactly this and had to restructure to `foldl` as a workaround.

**A second, adjacent defect surfaced while reproducing this one** (V11): `ailang run
--strict-bytecode` **without** `--bytecode` silently runs the evaluator — the strict flag
only qualifies `--bytecode` (`BytecodeMode` stays false, `internal/runner/entrypoint.go:144`),
while `ailang test --strict-bytecode` *does* imply the VM (`commands_language.go:284` ORs
the two flags). My own first strict-mode measurements in this session were evaluator
numbers wearing a strict label (they matched the interpreter to the millisecond before I
caught it), and the report's parenthetical "(also --strict-bytecode)" suggests the same
trap. A user asking for the strict VM gets a silent fallback — the exact failure mode
CLAUDE.md's "no silent fallbacks" principle names.

**This is systemic, not a one-off** (audit per CLAUDE.md principle 3): the defect class is
"suffix/prefix views of an immutable list are copied on the VM". Same-class residents:
`builtinListTail` (pattern tails, `std/list.tail` — P0), `builtinListDrop`
(`append([]bytecode.Value{}, xs[n:]...)`), `builtinListTake`
(`append([]bytecode.Value(nil), xs[:n]...)`) — `internal/vm/builtins_list_poly.go:55-87`
(V21). All three are one-line changes to the same sharing discipline the other two
engines already follow; this doc fixes the class, not the reported instance.

## Verification Log

Provenance: **[R]** = read first-party at this tree (`59feeccf`), command/line below;
**[M]** = measured this session against the container's binary (v0.52.5, commit `7200786`,
md5 `8e4464db…`, `/usr/local/bin/ailang`); **[P]** = the reporter's measurement
(v0.52.0 `bf2436a`, md5 `2dd0828…`), inherited, not re-run; **[N]** = negative-existence
claim, instrument included per the skill's rule.

| # | Claim | Command / location | Observed |
|---|---|---|---|
| V1 [R] | VM `_list_tail` copies the remaining list | `internal/vm/builtins.go`, `builtinListTail` (registered at `builtins.go:77`, name at `internal/bytecode/builtin_names.go:20`) | `tail := make([]bytecode.Value, len(elems)-n); copy(tail, elems[n:]); NewList(tail)` — O(n) per call |
| V2 [R] | Interpreter binds pattern tails as an O(1) subslice alias | `sed -n '255,257p' internal/eval/eval_patterns.go` | `tailElements := listVal.Elements[len(p.Elements):]` → `&ListValue{Elements: tailElements}` — no copy |
| V3 [R] | Go-codegen runtime `ListTail` returns a shared subslice | `internal/gen/golang/codegen_runtime_collections.go:60-67` (`ListTail` emitter) | `if l, ok := list.([]interface{}); ok … { return l[1:] }` — O(1) alias; compiled programs run this today |
| V4 [R] | The lower pass emits `_list_tail` for every list-pattern tail | `internal/gen/lower/match_pattern.go:27-28` (access-path table), `:85` (cond), `:133` (binding), `:191-193` (`listTail` → `BuiltinCall{Name: "_list_tail"}`) | `x :: r` lowers to `_len(s) ≥ 1 && …` then binds `x = _list_get(s, 0)`, `r = _list_tail(s, 1)` |
| V5 [R] | `a :: b :: rest` chains flatten before lowering | `internal/gen/lower/match_pattern.go:140-162` (`normalizeListPattern`) | `x :: r` is the flat `[x] + tail r` form; one length check, one `_list_tail` per match — the per-step cost is structural, not an artifact of nesting |
| V6 [M] | Walk is quadratic on the VM | `ailang run --quiet --bytecode --caps IO --args-json N walk_only.ail` (repro files per Examples) | 40k = 6.39 s; 80k = 24.7 s; 160k = 90.9 s — ×3.9 per doubling |
| V7 [M] | Same walk is linear on the interpreter | same file, no `--bytecode` | 40k = 0.85 s; 80k = 1.10 s; 160k = 1.74 s — 52× faster than the VM at 160k |
| V8 [M] | `foldl` with the identical step is ~free on the VM | `walk_foldl.ail`, `--bytecode` | 80k = 0.364 s total vs 0.341 s hello-world baseline → net ≈ 0.02 s |
| V9 [M] | The construction half (`0.25 :: acc`) is quadratic on the VM — parked doc's territory | `build_only.ail`, `--bytecode` | 40k = 6.9 s; 80k = 28.4 s — ×4.1 per doubling (`OpCons` copy; `internal/vm/vm.go` `OpCons`: `make([]bytecode.Value, 0, len(old)+1)` + `append(elems, old...)`) |
| V10 [R/M] | Tail calls are constant-space on the VM — the reporter's alternative hypothesis is false | disassembly shows `TAIL_CALL r8, args=3` in the loop body; `internal/vm/vm.go:320-345` (`OpTailCall` reuses the frame); `DefaultMaxStack = 10000` (`vm.go:12-14`); the 160k-deep walk completed (V6) | 160,000 tail calls under a 10,000-frame cap ⇒ frames are reused, not stacked; `internal/vm/frame_reuse_test.go` pins TAIL_CALL recycling |
| V11 [R/M] | `--strict-bytecode` alone silently runs the evaluator | `ailang run --strict-bytecode … walk_only.ail`: 20k = 0.52 s, 40k = 0.73 s, 160k = 1.75 s — matches V7's interpreter times, not V6's VM times. Source: `cmd/ailang/main_run.go:137` (help: "With --bytecode: fail instead of falling back"), `internal/runner/entrypoint.go:144` (gates on `params.BytecodeMode` only), vs `cmd/ailang/commands_language.go:284` (`test`: `Bytecode: *bytecodeFlag \|\| *strictBytecodeFlag`) | `run --strict-bytecode` without `--bytecode` is a no-op: `BytecodeMode` stays false, entry runs on the evaluator, no warning; `--bytecode --strict-bytecode` together = the real strict VM (20k = 1.55 s, 40k = 6.19 s, same quadratic as plain `--bytecode`) |
| V12 [N] | **No VM path mutates a list's backing slice in place** (the safety precondition for sharing) | `grep -rnE 'AsList\(\)\[[^]]*\] *=\|append\((elems\|src\|lst\|args\[[0-9]+\]\.AsList\(\))' internal/vm/ --include='*.go' \| grep -v _test` → 0 hits into shared slices; every index-write site writes into a fresh `make` (`vm.go` `OpMakeList`; `builtins_array_poly.go` `array_make`), a copied slice (`boxedElems` allocates fresh; `array_set` writes only after `append([]float64(nil), a.Floats…)` / fresh-box), or a pre-sized result (`builtins_hof.go`); `__list_sortBy` copies before sorting (`builtins_hof.go:221-231`: `result := make(…); copy(result, src)`); `__array_from_list` copies (`append([]bytecode.Value{}, xs...)`) | No aliased write or shared-slice append exists — every list-producing op is already copy-on-write |
| V13 [N] | No test pins `_list_tail`'s copying (only its error text) | `grep -rn "_list_tail" internal/ --include='*_test.go'` | 4 hits, all error-message/delegation pins (`internal/eval/builtin_errors_test.go:92,240`; `internal/builtins/stdlib_delegation_test.go:52`) — behavior-preserving under the fix |
| V14 [R] | `std/list.ail` documents the VM tail copy as a standing limitation | `std/list.ail:9-11` | "Matching `[x, …rest]` also copies `rest` on the bytecode VM, so recursion that walks a list that way is O(n²) there" — the doc sentence this doc deletes |
| V15 [R] | The interpreter's O(1) tail is the settled reference semantics | `internal/builtins/stdlib_delegation_test.go:52` | "_list_tail … codegen-only helper is unnecessary in the interpreter because std/list.tail is O(1) pattern matching" |
| V16 [M] | The walk's per-iteration VM code is: length check, `_list_get`, **`_list_tail`**, closure, TAIL_CALL | `ailang disasm walk_only.ail` → `walk_only.countGt` (proto 94): `0015 BUILTIN_CALL r3, builtin#514, argc=2` (`_list_get`), `0019 BUILTIN_CALL r4, builtin#515, argc=2` (`_list_tail`), `0027 CLOSURE`, `0031 TAIL_CALL r8, args=3` | `_list_tail` is native (no bridge trap); the copy is the only superlinear op in the loop |
| V17 [P] | Reporter's measurements (v0.52.0 `bf2436a`, md5 `2dd0828…`) | inherited: full repro 10k = 0.21 s, 100k = 13.6 s (×65 for ×10 — both halves quadratic); 786,432-element walk 427 s vs 0.08 s `foldl` | ratios reproduce at V6–V9 on the container's binary; absolutes differ (machine speed) |
| V18 [N] | Container toolchain limits + binary provenance | `which go make` → absent; `git log --oneline \| wc -l` → 1 (shallow squashed clone, no blame history for `builtinListTail`); binary v0.52.5@`7200786` not in this tree's history (`git cat-file -t 7200786` → error); `ailang version` → v0.52.5, `-dirty` | Source claims are reads at `59feeccf`; measurements at v0.52.5@`7200786`. The mechanism agrees across both (V1's copy ≡ V6's quadratic + V16's native dispatch). **AC-0 re-derives on a build of this tree** |
| V19 [R] | `NewList` wraps a slice in a shared heap object — sharing is a slice op, values stay immutable | `internal/bytecode/value.go:208-210` | `Value{Tag: TagList, Obj: &ListObj{Elems: elems}}` |
| V20 [R] | Regression fixtures exist and exercise cons/spread patterns | `examples/first_non_repeat.ail`, `examples/record_cons_pattern.ail`, `examples/url_route_dispatch.ail` (`grep -rln ':: rest\|\.\.\.rest' examples/*.ail`) | listed fixtures exist; sprint re-runs them under `--bytecode` and the evaluator and diffs outputs |
| V21 [R] | `take`/`drop` are the same copy-class on the VM | `internal/vm/builtins_list_poly.go:55-87` | `builtinListTake`: `append([]bytecode.Value(nil), xs[:n]…)`; `builtinListDrop`: `append([]bytecode.Value{}, xs[n:]…)` — prefix/suffix views, same one-line fix |
| V22 [R] | Bench layout precedent | `bench/vm_hof_callbacks/` (`consrepro.ail`, `sumfold.ail`, `mapfilt.ail`, `run.sh` min-of-N) | the pattern this doc's bench follows |
| V23 [R] | Coverage: the parked doc explicitly keeps pattern tails untouched and defers VM parity | [m-list-cons-quadratic](../m-list-cons-quadratic.md) Non-Goals + V15/V18 there | "VM `OpCons` — stays O(n) … parity deferred"; "Arena propagation through pattern tails — future work; Sprint 1 freezes it off"; its V15 documents the interpreter alias as existing fact, not as something to change |
| V24 [M] | The repro constructs type-check | `ailang check` on `full_repro.ail`, `walk_only.ail`, `walk_foldl.ail`, `build_only.ail` | `✓ No errors found!` ×4 — `x :: r` cons-pattern syntax, `match` with two arms, tail recursion, `foldl` with an if-step |
| V25 [R] | The reporter's real path builds the list linearly | `std/embedding.ail:18-19` | `_embedding_decode(bytes) -> Option[list[float]]` — Go builtin, one-pass build; the walk, not the build, was their 427 s |
| V26 [R] | The eval↔VM bridge deep-copies in both directions — sharing cannot leak across engines | `internal/vm/convert.go:40-51` (`bytecodeToEval` list case: fresh `dst := make(…)` per element) | a shared tail crossing the bridge is converted elementwise into a fresh slice |

### Workarounds note (W-1)

The container's binary is not a build of this tree, and there is no Go toolchain (V18), so:
(a) the fix itself is unimplemented and unmeasured here — the "after" numbers in Success
Criteria are AC targets, not results; (b) `go test` cannot run in this session; the
implementer runs the full test plan as AC-0/AC-5. Quorum was attempted and degraded to
controller-only with all three reviewer routes recorded absent by name (Status header) —
re-run when a route is available.

## Goals

**Primary Goal:** make consuming a list by pattern recursion (`x :: r` / `[x, …rest]`)
cost the same asymptotics on the bytecode VM as on the interpreter and Go codegen — O(n),
no copy per step — without changing a single observable value.

**Success Metrics:**
- `walk_only.ail` on `--bytecode`: ≤ 2× the interpreter at n = 160,000 (today: 52×), scaling ×≤2 per doubling (today ×3.9) (AC-2).
- The reporter's shape — cons-pattern walk over a 786k-element decoded list — completes in ≤ 2 s on the VM (reporter baseline: 427 s; AC re-derived) (AC-2b).
- Byte-identical outputs on every fixture and test (AC-5).
- `ailang run --strict-bytecode` without `--bytecode` no longer silently runs the evaluator (AC-4).
- A committed benchmark (`bench/vm_tail_walk/`) that pins the regression mechanically (AC-3).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| **Do NOT re-open the representation question (D-19).** `::`/`++` construction stays quadratic until the parked doc's human decision; this doc changes only how *existing* lists are viewed | Two docs claiming the same fix forks the decision record; the parked doc is quorum-blocked on a named human decision | human (already recorded, D-19) | — | high (avoided) |
| The fix is **sharing, at the builtin**: `_list_tail` returns `NewList(elems[n:])` — the exact discipline the interpreter (V2) and Go codegen (V3) already apply | Representation-preserving, one line, precedent on two engines; no arena, no `ListValue`/`ListObj` change | this doc | design | low |
| **`take`/`drop` ride along** in the same commit (same suffix/prefix class, V21) | One class, one fix — the systemic pattern CLAUDE.md principle 3 requires; three one-line changes reviewed under one safety argument | this doc | design | low |
| **`--strict-bytecode` implies `--bytecode` on `run`** (match the `test` command's existing OR, V11) | A silent evaluator fallback behind a flag named "strict" is the exact anti-pattern CLAUDE.md principle 2 names; `test` already settled the intended semantics | this doc | design | low |
| Retention trade-off is **accepted and documented**, not mitigated: a shared tail pins its source array (bounded by the list's own size) | Identical to interpreter/Go-codegen behavior since inception (V2/V3); today's copy is O(n²) allocation vs O(n) retained — strictly better for walks | this doc | design | low (doc) |

### Design Freeze

- [x] Fix shape frozen: subslice share in `builtinListTail` (+ `builtinListTake`/`builtinListDrop`), clamping semantics (`n<0→0`, `n>len→len`) byte-identical.
- [x] No engine-semantics change: outputs, errors, error text, and determinism guarantees unchanged; only allocation/sharing and cost change.
- [x] `--strict-bytecode`-implies-`--bytecode` is in scope (V11), flagged as an intentional behavior change in Conflict Surface §5.
- [x] `mk`/`::`/`++` construction (the report's other half) is out of scope and stays parked (V23, D-19).
- [x] Bench committed as `bench/vm_tail_walk/` following the `vm_hof_callbacks` min-of-N pattern (V22).

## Solution Design

### Overview

Three small phases, independently shippable, smallest-first:

1. **Phase 1 — O(1) tail/prefix/suffix views on the VM (P0).** In `internal/vm/builtins.go`,
   `builtinListTail` becomes:

   ```go
   // before                                          // after
   tail := make([]bytecode.Value, len(elems)-n)       return bytecode.NewList(elems[n:]), nil
   copy(tail, elems[n:])
   return bytecode.NewList(tail), nil
   ```

   with the existing clamping (`n<0→0`, `n>len→len`) unchanged — note `elems[n:]` after
   clamping is always well-formed. `builtinListTake` → `NewList(xs[:n])`,
   `builtinListDrop` → `NewList(xs[n:])` (same clamping, V21). Lists are immutable at the
   language level and the audit (V12) shows no VM path writes into a list's backing slice,
   so the shared array can never be observed to change. The bridge deep-copies on engine
   crossings (V26), so sharing stays VM-internal.

2. **Phase 2 — `--strict-bytecode` implies the VM on `run` (P1, one line).** In
   `cmd/ailang/main_run.go`, treat `--strict-bytecode` as implying `--bytecode`
   (`bytecodeMode := *bytecodeFlag || *strictBytecodeFlag` before building
   `runner.Options`), exactly as `ailang test` already does (`commands_language.go:284`,
   V11). The strict flag's own help text ("With --bytecode: fail instead of falling
   back") already promises this pairing; after the change the pairing is enforced instead
   of assumed.

3. **Phase 3 — docs honesty + bench (P1).** `std/list.ail:9-11` deletes/rewrites the
   "Matching `[x, …rest]` also copies `rest` on the bytecode VM" sentence (V14) — after
   Phase 1 it is false; replace with the sharing statement (and keep the `::`/`++`
   construction warning, which stays true until D-19). `docs/LIMITATIONS.md` gains a row
   for the *retention* trade-off (small view pins its source array, bounded by list size,
   interpreter-parity). Commit `bench/vm_tail_walk/` with `walk.ail` (cons-pattern
   recursion), `build.ail` (`::` construction, the parked half, kept as the documented
   contrast), `foldl.ail`, and a min-of-N `run.sh` mirroring `vm_hof_callbacks` (V22).

### Implementation Plan

**Phase 1 (the fix):**
- [ ] `internal/vm/builtins.go` `builtinListTail`: copy → `NewList(elems[n:])`
- [ ] `internal/vm/builtins_list_poly.go`: `builtinListTake`, `builtinListDrop` → share
- [ ] Go unit tests (below, Testing Strategy): aliasing-safety, allocation-count, clamp parity, error text unchanged
- [ ] AC-0 baseline first: rebuild this tree, re-derive V6–V9 tables, record in the sprint log

**Phase 2 (strict flag):**
- [ ] `cmd/ailang/main_run.go`: strict implies bytecode; update the flag help ("Implies --bytecode")
- [ ] Test: `ailang run --strict-bytecode` on a file with an evaluator-only entry now *fails loudly* instead of silently evaluating; `--bytecode --strict-bytecode` unchanged

**Phase 3 (docs + bench):**
- [ ] `std/list.ail:9-11` header rewrite; `tail`/`take`/`drop` comments updated
- [ ] `docs/LIMITATIONS.md` retention row; reconcile with rows 41–42 (D-19 pending)
- [ ] `bench/vm_tail_walk/` + `run.sh` committed; run before/after, min-of-5, both engines
- [ ] Changelog fragment `changelogs/unreleased/`

**Files to modify/create:** `internal/vm/builtins.go` (+2/−2), `internal/vm/builtins_list_poly.go` (+2/−2), `cmd/ailang/main_run.go` (+2/−2), `std/list.ail` (comment ~4 lines), `docs/LIMITATIONS.md` (+1 row), `internal/vm/list_tail_share_test.go` (new, ~120 LOC), `bench/vm_tail_walk/{walk.ail,build.ail,foldl.ail,run.sh}` (new).

## Conflict Surface

Touches `internal/vm/` and `cmd/ailang/` — required. The positions extended: (a) the
result of the `_list_tail`/`__list_take`/`__list_drop` builtins, (b) the `--strict-bytecode`
flag's meaning on `run`, (c) user-facing cost-model prose in `std/list.ail` +
`docs/LIMITATIONS.md`.

1. **Position (a): who else reads/consumes list-suffix views?** The pattern lower pass is
   the dominant caller (`_list_tail` from every list-pattern tail, V4); `std/list.tail`
   lowers to the same builtin (disassembly, V16 area); `take`/`drop` are stdlib exports.
   Nothing else calls them. Disambiguation: all callers receive the same *values* — the
   only change is which allocation backs them.
2. **Aliasing hazards in position (a):** the exhaustive audit (V12) — `OpCons` builds
   fresh, `MAKE_LIST` copies from registers, `sortBy` copies before sorting, `mapAccumL`/
   `map`/`filter` pre-size fresh results, array ops are a different type with
   copy-on-write `set`, `array_from_list` copies, the bridge deep-copies (V26). The one
   *new* aliasing class this introduces (tail shares head-storage) is the class the
   interpreter has run in production since inception (V2) and every `--emit-go` program
   runs today (V3).
3. **HOF/table coupling:** none — the changed builtins keep their indices and signatures;
   `HOFBuiltinNames`/`HOFBuiltinTable` untouched (no new builtins).
4. **Cost-model prose is load-bearing (V14):** `std/list.ail`'s header currently teaches
   the copy as a workaround-generating fact ("index with nth or use a helper below").
   After Phase 1 that sentence is false and must change *in the same release* — a stale
   cost warning is a new bug of the same class (A9).
5. **`--strict-bytecode` behavior change (intentional):** users who (mis)used
   `--strict-bytecode` alone to get evaluator behavior now get the strict VM — including
   hard failures on evaluator-only entries. That is what the flag has always documented
   ("fail instead of falling back") and what `ailang test` already does (V11); anything
   else is a silent fallback. Flagged in the changelog as a behavior fix.
6. **Retention:** a small tail now pins its source array (e.g. keeping `drop(1_000_000,
   xs)` alive retains ~1M elements instead of ~0). Bounded by the source list's own size;
   identical to interpreter/Go-codegen behavior; documented (Phase 3) rather than
   mitigated (Design Freeze).

### Programs that MUST still work

1. `examples/first_non_repeat.ail`, `examples/record_cons_pattern.ail`,
   `examples/url_route_dispatch.ail` (V20) — cons/spread patterns; outputs byte-identical
   on `--bytecode` and the evaluator.
2. The reporter's `countAbove`/`synth` repro and `walk_only.ail` — same values (0 / 19999
   / …), only faster.
3. `std/list.tail`/`head`/`take`/`drop` on both engines — including edge cases (`tail([])`
   → `[]`, `take(0, xs)`, `drop(n > len)` → `[]`), clamping byte-identical.
4. `_list_tail` error text on non-list input (`builtin_errors_test.go:92,240`, V13).
5. Every VM HOF dispatch — tables untouched (Conflict Surface §3).

### What deliberately changes

- Allocation behavior of `_list_tail`/`take`/`drop` on the VM (shared backing arrays).
- Wall-clock on cons-pattern walks: O(n²) → O(n) — the point.
- `ailang run --strict-bytecode` alone: silent evaluator → strict VM (fails loudly).
- Comment text in `std/list.ail`, one `docs/LIMITATIONS.md` row, flag help text.
- Nothing else. `::`/`++` construction stays exactly as it is (D-19).

## Examples

**Before (this session's measurements, V6):** the walk on the VM — 90.9 s at 160k, ×3.9
per doubling; the same code on the interpreter: 1.74 s.

```ailang
-- The canonical walk. On --bytecode today: O(n^2). After this doc: O(n),
-- same output, same determinism, same traces.
pure func countAbove(xs: [float], thr: float, acc: int) -> int =
  match xs {
    [] => acc,
    x :: r => countAbove(r, thr, if x > thr then acc + 1 else acc)
  };
```

**After:** identical program, no rewrite needed — the whole point. The consumer's
`foldl` workaround becomes unnecessary but stays valid; `foldl` remains the right tool
when a *callback* shape is natural, and `mapAccumL` when stateful emission is needed
(those recommendations are unaffected — they were about the construction half and
per-step cost, both still true).

## Success Criteria (Acceptance)

- [ ] **AC-0** (gate): build this tree, re-derive the V6–V9 measurement tables, record in the sprint log (this container could not: V18).
- [ ] **AC-1** `builtinListTail`/`builtinListTake`/`builtinListDrop` return shared subslices; clamping and error behavior byte-identical (`internal/eval/builtin_errors_test.go`, `internal/vm` parity tests green).
- [ ] **AC-2** `walk_only.ail` on `--bytecode`: n=160,000 ≤ 2× the interpreter wall (base: 52×), ×≤2 per doubling from 80k (base ×3.9); net time ≈ foldl-class. **AC-2b** derived target: the reporter's 786k walk ≤ 2 s on a machine where hello-world startup is ~0.35 s.
- [ ] **AC-3** `bench/vm_tail_walk/run.sh` committed; before/after tables in the sprint report; the Go allocation test (`testing.AllocsPerRun` on `_list_tail` ≈ 0 allocs) green.
- [ ] **AC-4** `ailang run --strict-bytecode` without `--bytecode` runs the VM strictly (an evaluator-only entry fails loudly with the strict error, not a silent evaluator run); `ailang test --strict-bytecode` unchanged.
- [ ] **AC-5** `make test`, `make verify-examples` green; Conflict Surface §"MUST still work" fixtures byte-identical outputs.
- [ ] **AC-6** `std/list.ail:9-11` no longer claims the copy; `docs/LIMITATIONS.md` carries the retention trade-off; changelog fragment with the strict-flag behavior note.

## Testing Strategy

- **Allocation-pinning Go test** (the regression the bench guards at the unit level):
  `testing.AllocsPerRun` on `_list_tail(list, 1)` over a large list ≈ 0 (today: n).
- **Aliasing-safety tests**: tail of list → `OpCons` onto the tail → assert the original
  list's values unchanged (copy-on-write discipline); `sortBy` on a tail; `array_from_list`
  on a tail. Same tests against the interpreter's outputs (parity).
- **Clamp/edge parity**: `_list_tail(xs, n)` for n ∈ {0, 1, len, len+1, −1} on both engines,
  error text for non-list input byte-identical (V13 pins).
- **Strict-flag test**: `run --strict-bytecode` on an evaluator-only entry → the strict
  error, exit ≠ 0 (today: silently succeeds on the evaluator, exit 0 — V11).
- **Bench**: `bench/vm_tail_walk/run.sh` min-of-5, both engines, n ∈ {40k, 80k, 160k};
  include `build.ail` (the parked construction half) so the two halves' costs stay
  honestly separated in future reports.

## Non-Goals

- **`::`/`++` construction, `OpCons`, the arena, D-19** — parked doc's territory, human decision pending (V23). The report's `mk` half stays quadratic after this doc; the docs say so.
- **Interpreter changes** — it is the reference semantics and already O(1) (V2); its `take`/`drop` Go builtins may copy, which is a (smaller) cost question for a separate audit if a consumer hits it.
- **Go-codegen changes** — already shares (V3).
- **Tail-call elimination** — already present on the VM (V10); the interpreter's TCO shipped separately (m-eval-tail-calls).
- **General VM↔interpreter performance parity program** — only this shape; the construction-side parity belongs to D-19, per [m-foldl-cons-cost-model](../../implemented/v0_52_0/m-foldl-cons-cost-model.md) Phase 3's account.

## Timeline

**Day 1**: AC-0 baselines; Phase 1 fix + unit tests; AC-1. **Day 2**: Phase 2 + tests;
Phase 3 docs + bench; AC-2/AC-3/AC-4 measurements; changelog. **Total ~2 days** (×2 honest
= 4, dominated by measurement discipline, as every sibling sprint has found).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| An undiscovered write-into-shared-list path makes sharing observable | High (silent wrong values) | The exhaustive audit (V12) with instruments named; two-engine parity differential tests over the full fixture corpus; the aliasing-safety tests in Testing Strategy; interpreter/Go-codegen have run the identical aliasing class in production (V2/V3) — the burden of proof is on "a write exists", not on "sharing is safe" |
| Retention: a long-lived small tail pins a large array | Med (memory) | Bounded by the source list's size (O(n) worst case vs today's O(n²) allocation); identical to interpreter/Go-codegen; documented in LIMITATIONS (Phase 3); a consumer holding `drop(10⁶, xs)` forever can copy explicitly via `take(len, …)` if it ever bites |
| `--strict-bytecode` implies-VM breaks a workflow that relied on the no-op | Med (behavior) | The flag never documented that meaning; `test` already ORs the flags (V11); changelog behavior note; `--bytecode` semantics unchanged |
| Stale docs contradict the fix (std/list.ail header) | Low but real (A9) | Phase 3 lands in the same sprint; AC-6 checks the sentence is gone |
| Reporter timings don't reproduce on the implementer's machine | Low | AC-0 re-derives; thresholds are ratios (×/doubling, VM:interpreter), not absolute seconds |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Outputs, evaluation order, traces unchanged; only allocation identity changes, which AILANG cannot observe (V12/V13/V26) |
| A2: Replayability | 0 | Same instruction streams (the fix touches no compiler output), same values |
| A3: Effect Legibility | 0 | Pure builtins; no effects touched |
| A4: Explicit Authority | 0 | No capabilities touched |
| A5: Bounded Verification | +1 | The claim is mechanically checkable: an allocation-count unit test + a min-of-N bench pin it forever |
| A6: Safe Concurrency | 0 | VM is single-goroutine; bridge deep-copies crossings (V26); lists immutable |
| A7: Machines First | +1 | AI-written idiomatic recursion currently pays a hidden 52×; the fix removes the punishment for the canonical form instead of teaching a workaround |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Removes a quadratic cost from a documented idiom *and* deletes the now-false limitation sentence; the strict-flag fix removes a silent fallback |
| A10: Composability | 0 | No API changes |
| A11: Structured Failure | +1 | `--strict-bytecode` fails loudly instead of silently running the wrong engine (V11) |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Proceed**

### Hard Violation Check

- [x] A1 (Determinism): no nondeterminism; sharing is unobservable (V12/V13/V26)
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): the change exists to stop the substrate punishing machine-idiomatic code

## Duplicate / Coverage Gate (explicit, per skill §4)

The create script's auto-search returned no hits (no doc index in this container); the manual
gate, from reading the top candidates:

- [m-list-cons-quadratic](../m-list-cons-quadratic.md) (planned, **PARKED — needs-human-review**, same defect family, high similarity). **Distinct and not duplicative**: that doc is the *construction* fix (`::`/`++` copy → amortized O(1) prepend via arena, D-19 human decision A/B). Its Non-Goals explicitly freeze pattern-tail aliasing untouched, defer VM `OpCons` parity, and its V15 records the interpreter's O(1) pattern tail as *existing settled behavior*. This doc is the *consumption* fix (pattern-tail views on the VM), representation-preserving, no D-19 dependency: if D-19 later lands as arena (Option B), shared tails remain correct (nil-arena views, exactly as that doc's "Pattern-match tail alias" row already assumes); if it lands as cons cells, this fix is superseded harmlessly. The two docs fix different halves of the reporter's repro; this doc's V9 explicitly attributes the `mk` half to the parked doc.
- [m-foldl-cons-cost-model](../../implemented/v0_52_0/m-foldl-cons-cost-model.md) (implemented v0.52.0–v0.52.5). **Distinct**: docs-honesty for *construction* idioms, the `mapAccumL` primitive, and VM per-callback overhead (H1/H2). Its scope statement routed the substrate half to D-19 and explicitly did not touch pattern-tail destructuring; Phase 3's profile attributed consrepro to `OpCons` bytes, not `_list_tail`. This doc extends its cost-model work to the walk side and cites its profile as the GC-dominance mechanism.
- [m-vm-match-lowering](../../implemented/v0_52_0/m-vm-match-lowering.md) lineage (implemented v0.51.1) — the recursive pattern lowering that *introduced* the `_list_tail` emission path this doc optimizes; it fixed correctness (unbound/nested bindings), not cost. Sibling, not duplicate.
- [m-eval-tail-calls](../../implemented/v0_52_0/m-eval-tail-calls.md) (implemented) — interpreter TCO; different engine, different mechanism (V10 shows VM TCO already works).
- [m-perf4-bytecode-interpreter](../v1_1_0/m-perf4-bytecode-interpreter.md) (planned, P3 exploratory) — whole-engine performance program, not this shape.

## Related Documents

- [m-list-cons-quadratic](../m-list-cons-quadratic.md) — the parked construction-half fix (D-19); this doc's `mk` measurement (V9) is its evidence class, and its Non-Goals define this doc's boundary.
- [m-foldl-cons-cost-model](../../implemented/v0_52_0/m-foldl-cons-cost-model.md) — the cost-model honesty precedent, the GC-dominance profile, and the frame-recycling/H2 work that makes the post-fix walk fast.
- [m-vm-match-lowering](../../implemented/v0_52_0/m-vm-match-lowering.md) (v0.51.1; supersedes [m-bytecode-nested-pattern-lowering](../../implemented/v0_52_0/m-bytecode-nested-pattern-lowering.md)) — the recursive pattern lowering that emits `_list_tail`; correctness baseline for the pattern paths this doc must not perturb.
- [m-eval-tail-calls](../../implemented/v0_52_0/m-eval-tail-calls.md) — interpreter tail calls; establishes the tail-recursion-is-constant-space contract this doc's V10 confirms on the VM.
- M-ITERATIVE-LIST ([implemented/v0_9_2](../../implemented/v0_9_2/m-iterative-list-builtins.md)) — the Go-delegation pattern; the walk fix keeps hand-written recursion a first-class alternative rather than a documented trap.

## References

- **Source report**: coordinator task `task-006b0353` (this doc's directive), AILANG v0.52.0 `bf2436a`, binary md5 `2dd0828…`; consumer `stapledons-godot` R1-ISM-DUST PR B; related #676.
- Repro and measurement scripts (this session): `walk_only.ail`, `walk_foldl.ail`, `build_only.ail`, `full_repro.ail` (folded into Phase 3's `bench/vm_tail_walk/`); disassembly via `ailang disasm`.
- [design_docs/PROGRAM.md](../../PROGRAM.md) §4 — routing lanes. · [Design Axioms](/docs/references/axioms).

---

**Document created**: 2026-10-09 (design-doc-creator, unattended coordinator session `task-006b0353`; live v0.52.5 binary available for measurements, no Go toolchain / no build of this tree — Verification Log provenance marks and W-1 apply)
**Last updated**: 2026-10-09

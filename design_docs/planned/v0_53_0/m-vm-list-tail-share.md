# M-VM-LIST-TAIL-SHARE: O(1) list-tail binding on the bytecode VM (cons-pattern walks are O(n²))

**Status**: Planned — quorum attempted 2026-10-09 (`task-006b0353`): **all three external reviewers absent** (`gemini-3-1-pro` unreachable; `oc-kimi-k3` unreachable; `gpt6-1-sol` auth), recorded by name in `.ailang/state/mission-quorum/m-vm-list-tail-share-2026-10-09T11-00-54Z.json`; controller verdict **pass** — degraded to controller-only, **not quorum-cleared**. The 26-row first-party Verification Log is the load-bearing evidence; re-run quorum when a reviewer route is available (same convention as [m-bytecode-nested-pattern-lowering](../../implemented/v0_52_0/m-bytecode-nested-pattern-lowering.md)).
**Re-reported 2026-10-13** (`task-a0ee29fa`, v0.54.6 `71f4cf6`, md5 `81bacda6…`): a **second, independent consumer report** (sunholo lane — see Bug report) confirms the defect at v0.54.6 with new real-workload evidence (pprof, R2-CREW deadline). **The fix is still unimplemented at v0.54.8** — first-party re-derivation that session (V27–V29); all three copy sites then lived in this tree. **Superseded 2026-10-15: landed as #1827 / v0.54.10 (V41).** Target lapsed (v0.53.0 → v0.54.8 shipped without it); the residual re-targets at sprint-planner time.
**Phase 1 LANDED 2026-10-15** (attended, #1827, **shipped v0.54.10**, V41): `_list_tail`/`__list_take`/`__list_drop` return capacity-capped subslices (`elems[n:len:len]`); `internal/vm/list_tail_share_test.go` (identity + clamp + cap tests, append-isolation, mutation-checked — V42). Reporter repro 6.77 s → 1.05 s on dev; the `std/list.ail` cost sentence (V14) was rewritten in the same release (V43).
**Residual scope (this update — D-TRIAGE-14, attended 2026-10-16):** (1) Phase 2 — `--strict-bytecode` implies the bytecode VM on `run`; (2) `bench/vm_tail_walk/` with equivalent inputs and **checked** outputs; (3) the `docs/LIMITATIONS.md` retention row, consistent with [m-list-cons-quadratic](../m-list-cons-quadratic.md) staying PARKED on D-19. **Phase 1 is not re-planned** — the prior sprint card (`task-fc7451cd`, PR #1750) was rejected precisely because its M2 re-landed it. The cons-walk was re-measured on current dev first (V33–V38): the residual gap against the interpreter is **inverted** — the VM is now ~4.7× *faster* (Problem Statement, "Current state").
**Target**: v0.53.0 (lapsed — Phase 1 shipped v0.54.10; the residual re-targets current dev ≥ v0.54.12 at sprint-planner time)
**Priority**: P0 for the landed half (a 52× engine divergence on the canonical functional list-walk idiom, reported from a downstream consumer mid-PR); **P1 for the residual** (a silent-fallback CLI fix + mechanical regression pinning)
**Estimated**: residual ~1 day (one-line flag fix + strict test; bench with checked outputs; one LIMITATIONS row + canonical entry), ×2 honest → 2 days. Phase 1's ~2 days were spent and landed in #1827.
**Dependencies**: None. **Coordination required** (text only): [m-list-cons-quadratic](../m-list-cons-quadratic.md) stays PARKED on D-19 — this doc does not touch `::`/`++` construction or the arena decision; `docs/LIMITATIONS.md` rows 41–42 (the D-19 rows) stay as they are; the new retention row sits beside them without contradicting D-19's pending state. `std/list.ail`'s header was already corrected by #1827 (V43) and needs no further edit.
**Planner-Lane**: opus-required (VM builtin semantics + allocation behavior)
**Bug report**: AILANG v0.52.0 (`bf2436a`, darwin arm64, binary md5 `2dd082832fbadf861b49a1a432a0130c`), `--bytecode`. Downstream consumer `stapledons-godot` (Edenhofer dust-map pass, R1-ISM-DUST PR B); workaround in flight: `foldl`. **Second report 2026-10-13**: AILANG v0.54.6 (`71f4cf6`, binary md5 `81bacda6a3a81abe90bae00166ee1a76`), `ailang run --bytecode --caps IO,Env`; consumer `sunholo/crew_core` + `social_dynamics` (pprof: `vm.builtinListTail` 14% cumulative; HOF foldl/filter/any dominating) with the Stapledon R2-CREW 10-year crew skip (3,650 ticks) needing < 2 s vs ~60 s today. Same defect, same mechanism, new engine version — the repro, timings, and pprof are V28–V31.
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

**Current state after Phase 1 (re-measured on current dev, 2026-10-16, V33–V38).** The
pre-fix table above is history. On the container's v0.54.11 binary (`35a49b8` — the fix
shipped v0.54.10, so this binary carries it; source reads at this tree `3af754c4`,
v0.54.12), min-of-3, same `walk_only.ail` shape (constant threshold; count = n−1):

| Engine | n=40,000 | n=80,000 | n=160,000 | scaling |
|---|---|---|---|---|
| VM (`--bytecode`) | 0.389 s | 0.381 s | 0.444 s | **flat** — net ≈ 0.04 s over the 0.402 s startup baseline → **O(n)** |
| interpreter | 0.797 s | 1.311 s | 2.089 s | linear |
| VM, `foldl` with the same step | 0.363 s | 0.382 s | 0.389 s | flat, startup-bound |
| VM, `--strict-bytecode` **alone** | — | — | 2.017 s | ≈ interpreter, **not** the VM — Phase 2 open (V36) |
| VM, `--bytecode --strict-bytecode` | — | — | 0.444 s | = plain `--bytecode`; the pairing itself is unchanged |

**Residual gap against the interpreter: none — it is inverted.** The VM walk at 160k is
~4.7× *faster* than the interpreter gross (0.444 s vs 2.089 s), ~40× net of startup
(≈0.04 s vs ≈1.7 s): the O(1) native view beats the tree-walker's pattern match. AC-2's
"≤ 2× the interpreter" target is exceeded, not just met. Outputs are byte-identical at
every size on both engines (V35), and the walk's bytecode stream is unchanged —
`builtin#514 (_list_get)`, `builtin#515 (_list_tail)`, `CLOSURE`, `TAIL_CALL` (V38) — the
fix lives in the builtin, exactly as designed. **Still quadratic, deliberately:** the
construction half (`foldl(\acc x. x :: acc, [], …)`, `build_only.ail`) re-measures
×3–4.4 per doubling on the VM (10k = 0.964 s, 20k = 2.795 s, 40k = 12.291 s;
evaluator 40k = 4.414 s) — untouched by #1827, as D-19 requires (V37).

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

**The second report (v0.54.6, V30/V31) makes it a product deadline, not a discomfort**:
pprof on a real auto-play workload (`sunholo/crew_core` + `social_dynamics`) shows
`vm.builtinListTail` at **14% cumulative**, with the HOF builtins
(`hofBuiltinListFoldl/Filter/Any`) dominating the rest — and both modules recurse over
lists everywhere (`p :: rest` in collect, refresh, deliveries). Stapledon's **R2-CREW
needs a 10-year crew skip (3,650 ticks) in under 2 s; it takes ~60 s today**, a large
part of that in tail copies. The reporter's minimal repro (V30) matches the first
report's shape exactly: rec 10k = 0.19 s / 20k = 0.35 s / 40k = 1.18 s (wall, incl.
~0.03 s startup), foldl flat at 0.04–0.05 s at every size.

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
| V1 [R] | VM `_list_tail` copies the remaining list | `internal/vm/builtins.go`, `builtinListTail` (registered at `builtins.go:77`, name at `internal/bytecode/builtin_names.go:20`) | `tail := make([]bytecode.Value, len(elems)-n); copy(tail, elems[n:]); NewList(tail)` — O(n) per call — **pre-#1827 state; superseded v0.54.10 by the shared capacity-capped view (`builtins.go:395-420`, V41/V42)** |
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
| V21 [R] | `take`/`drop` are the same copy-class on the VM | `internal/vm/builtins_list_poly.go:55-87` | `builtinListTake`: `append([]bytecode.Value(nil), xs[:n]…)`; `builtinListDrop`: `append([]bytecode.Value{}, xs[n:]…)` — prefix/suffix views, same one-line fix — **both fixed in #1827 as capped views (V42)** |
| V22 [R] | Bench layout precedent | `bench/vm_hof_callbacks/` (`consrepro.ail`, `sumfold.ail`, `mapfilt.ail`, `run.sh` min-of-N) | the pattern this doc's bench follows |
| V23 [R] | Coverage: the parked doc explicitly keeps pattern tails untouched and defers VM parity | [m-list-cons-quadratic](../m-list-cons-quadratic.md) Non-Goals + V15/V18 there | "VM `OpCons` — stays O(n) … parity deferred"; "Arena propagation through pattern tails — future work; Sprint 1 freezes it off"; its V15 documents the interpreter alias as existing fact, not as something to change |
| V24 [M] | The repro constructs type-check | `ailang check` on `full_repro.ail`, `walk_only.ail`, `walk_foldl.ail`, `build_only.ail` | `✓ No errors found!` ×4 — `x :: r` cons-pattern syntax, `match` with two arms, tail recursion, `foldl` with an if-step |
| V25 [R] | The reporter's real path builds the list linearly | `std/embedding.ail:18-19` | `_embedding_decode(bytes) -> Option[list[float]]` — Go builtin, one-pass build; the walk, not the build, was their 427 s |
| V26 [R] | The eval↔VM bridge deep-copies in both directions — sharing cannot leak across engines | `internal/vm/convert.go:40-51` (`bytecodeToEval` list case: fresh `dst := make(…)` per element) | a shared tail crossing the bridge is converted elementwise into a fresh slice |
| V27 [R] | **Still unimplemented at v0.54.8** (this tree `b283b2d4`, read 2026-10-13): all three copy sites live; the v0.53.0 target lapsed | `sed -n '378,395p' internal/vm/builtins.go` (`tail := make(…); copy(tail, elems[n:])`); `internal/vm/builtins_list_poly.go:55-87` (`append([]bytecode.Value(nil), xs[:n]…)` / `append([]bytecode.Value{}, xs[n:]…)`); `ls bench/` → no `vm_tail_walk`; no changelog or doc references the fix | v0.53.0–v0.54.8 shipped without this doc's fix; a second consumer hit it in production |
| V28 [M] | First-party re-derivation of the second reporter's exact repro on this tree's binary (v0.54.8, `b283b2d4`, `/usr/local/bin/ailang`; container has no Go toolchain but the binary's commit matches this tree) | `tailbench.ail` (reporter's file verbatim): `ailang check` ✓; `ailang run --quiet --bytecode --caps IO,Env --entry main tailbench.ail <mode> <n>`, min-of-3 | rec: 10k = 0.89 s, 20k = 1.91 s, 40k = 6.48 s — ×3.4 per doubling (quadratic); foldl: 10k/20k/40k = 0.61/0.49/0.40 s — flat at the ~0.4–0.6 s startup baseline (hello-world min-of-3 = 0.39 s). Same shape as both reports' machines, slower container |
| V29 [M] | The repro's walk loop compiles to one native `_list_tail` copy per recursive step | `ailang disasm tailbench.ail` → prototype `tailbench.sumRec`: `BUILTIN_CALL r4, builtin#257 (_len)`, `BUILTIN_CALL r2, builtin#514 (_list_get)`, `BUILTIN_CALL r3, builtin#515 (_list_tail)`, `CLOSURE`, `TAIL_CALL` | `builtin#515` is `_list_tail` (`internal/vm/builtins.go:77`); the copy is the only superlinear op in the loop — matches V16 |
| V30 [P2] | Second reporter's measurements (v0.54.6 `71f4cf6`, md5 `81bacda6…`, inherited, not re-run — machine ~15× faster than this container) | `ailang run --bytecode --caps IO,Env --entry main tailbench.ail rec N` | rec 10k = 0.19 s, 20k = 0.35 s, 40k = 1.18 s (wall, incl. ~0.03 s startup); foldl 0.04–0.05 s flat at every size; pprof on the real workload: `vm.builtinListTail` 14% cumulative, `hofBuiltinListFoldl/Filter/Any` dominating |
| V31 [P2] | Second reporter's consumer impact (external packages, not in this tree, not re-run) | sunholo `crew_core` + `social_dynamics` auto-play | `p :: rest` recursion throughout (collect, refresh, deliveries); Stapledon R2-CREW 10-year crew skip (3,650 ticks) takes ~60 s against a < 2 s requirement — a large part in tail copies |
| V32 [N] | No newer doc supersedes this one | `grep -rl 'tail share\|_list_tail\|subslice' design_docs/implemented/v0_53_0 … v0_54_7` → 0 covering hits (hits are `m-run-policy-result-unforgeable`, YAML key order, serveapi, terminal-ui, cloud credit guard, background worker — none touch list tails); `ls design_docs/planned/` → this doc remains the only queue entry for the walk half | The v0.53.0-folder placement is stale-targeted, not stale-covered; re-target in place at sprint-planner time |

| # | Claim | Command / location | Observed |
|---|---|---|---|
| V33 [M] | **Cons-walk is flat on the VM post-#1827** | `ailang run --quiet --bytecode --caps IO --args-json N walk_only.ail`, min-of-3, binary v0.54.11 `35a49b8` (md5 `b107e65…`) | 40k = 0.389 s, 80k = 0.381 s, 160k = 0.444 s vs 0.402 s hello baseline → net ≈ 0.04 s, ×≈1 per doubling — **O(n)**; pre-fix V6/V28: ×3.4–3.9 per doubling |
| V34 [M] | **Residual gap vs interpreter: inverted (VM ahead)** | same file, no flag, min-of-3 | interpreter 40k = 0.797 s, 80k = 1.311 s, 160k = 2.089 s — VM ~4.7× faster gross at 160k, ~40× net of startup; the doc's AC-2 "≤2× interpreter" ceiling is exceeded |
| V35 [M] | Outputs byte-identical on both engines post-fix | diff of stdout, both engines | walk 40k/80k/160k = 39999/79999/159999 identical; `walk_foldl` 160k = 159999 identical; `build_only` 40k = 40000 identical |
| V36 [M/R] | **`--strict-bytecode` alone still silently runs the evaluator — Phase 2 open** | `--strict-bytecode` without `--bytecode`, walk 160k = 2.017 s ≈ interpreter 2.089 s, ≠ VM 0.444 s; source: `cmd/ailang/main_run_exec.go:128` (`BytecodeMode: bytecodeMode` — strict not OR'd in), `internal/runner/entrypoint.go:144` (gates on `BytecodeMode` only), `cmd/ailang/commands_language.go:284` (`test`: `Bytecode: *bytecodeFlag \|\| *strictBytecodeFlag`) | strict-alone produces evaluator timings to the millisecond; `--bytecode --strict-bytecode` = 0.444 s = plain `--bytecode`. The fix is one line at `main_run_exec.go:128`; `--strict-bytecode` help (`main_run.go:142`) still reads "With --bytecode: …" |
| V37 [M] | Construction half still quadratic on the VM — D-19 parked, unchanged by #1827 | `build_only.ail`, `--bytecode`, min-of-3 | 10k = 0.964 s, 20k = 2.795 s, 40k = 12.291 s — ×2.9–4.4 per doubling → O(n²); evaluator 40k = 4.414 s (quadratic on both engines, per rows 41–42). No residual claim that `::`/`++` construction is fixed |
| V38 [R] | The walk program's bytecode is unchanged — the fix lives in the builtin | `ailang disasm walk_only.ail` (this session) | loop still `BUILTIN_CALL builtin#514 (_list_get), argc=2`, `BUILTIN_CALL builtin#515 (_list_tail), argc=2`, `CLOSURE`, `TAIL_CALL` — same stream class as V16/V29 |
| V39 [N] | `bench/vm_tail_walk/` does not exist | `ls bench/` | `lsp_token_cost`, `vm_hof_callbacks` only — AC-R2 open |
| V40 [N] | No retention row in `docs/LIMITATIONS.md` | `grep -rn 'retention\|pins its source\|shares the source' docs/LIMITATIONS.md` | 0 hits; rows 41–42 (cons-quadratic, VM 1.8× build) are the D-19 rows the new row must sit beside without contradicting — AC-R3 open |
| V41 [R] | #1827 shipped as v0.54.10 and its changelog names this doc's open items | `changelogs/v0_32-current.md` — the `## [v0.54.10] - 2026-10-15` block ("Fixed — bytecode VM: `x :: rest` list walks are O(n), not O(n²) (#1827)") | "… M-VM-LIST-TAIL-SHARE Phase 1; the `--strict-bytecode` change (Phase 2) and `bench/vm_tail_walk` are still open" |
| V42 [R] | Landed tests pin identity, clamp, capacity cap, and append isolation | `internal/vm/list_tail_share_test.go` (read this session) | `TestListTailDoesNotCopy` (pointer identity into the source's backing array, ×3 builtins), `TestListTailClampsAndCapsCapacity` (cap == len), `TestListTailViewIsolatedFromAppend`; allocation counts deliberately not used ("a copy is one make") |
| V43 [R] | `std/list.ail` header no longer claims the pattern-match copy (Phase 3 doc half landed) | `std/list.ail:9-11` | "Matching `x :: rest` / `[x, ...rest]` does NOT copy: `rest` shares the list's storage on every engine, so recursion that walks a list that way is O(n)" — V14's sentence is gone; the `::`/`++` construction warning remains (true until D-19) |

Provenance, V33–V38: measurements at `/usr/local/bin/ailang` v0.54.11 (`35a49b8`, md5 `b107e65…`), one release behind this tree (`3af754c4`, v0.54.12) — `git cat-file -t 35a49b8` fails in this shallow squashed clone (same class as V18), and `ailang check` warns "stdlib version mismatch: expected v0.54.11, found v0.54.12". The flat VM walk (V33) plus the unchanged bytecode stream (V38) are empirical proof the binary carries the #1827 fix; source reads are at this tree.

### Workarounds note (W-1)

The container's binary is not a build of this tree, and there is no Go toolchain (V18), so:
(a) the fix itself is unimplemented and unmeasured here — the "after" numbers in Success
Criteria are AC targets, not results; (b) `go test` cannot run in this session; the
implementer runs the full test plan as AC-0/AC-5. Quorum was attempted and degraded to
controller-only with all three reviewer routes recorded absent by name (Status header) —
re-run when a route is available.

**Update 2026-10-13** (`task-a0ee29fa`): the container now has a binary whose commit
**matches this tree** (v0.54.8 `b283b2d4`), so V28/V29 are true first-party reads of this
exact source; the no-Go-toolchain limit (and therefore (a)/(b)) still applies. The second
report (V30/V31) is inherited — its workload packages are not in this tree.

**Update 2026-10-16** (#1827 landed; residual session `task-585cec8d`, attended under
D-TRIAGE-14): Phase 1 shipped v0.54.10, so the fix **is** implemented and **is**
measured — V33–V35, on the v0.54.11 binary `35a49b8` (provenance paragraph above; the
same W-1 class as V18 — one release behind this tree, no Go toolchain, so no build of
this tree this session either). (a)/(b) now apply only to the **residual**: Phase 2,
the bench, and the LIMITATIONS row are planned here, not implemented in this session;
the implementer's build re-derives V33–V38 as AC-R1–AC-R3 land.

## Goals

**Primary Goal:** make consuming a list by pattern recursion (`x :: r` / `[x, …rest]`)
cost the same asymptotics on the bytecode VM as on the interpreter and Go codegen — O(n),
no copy per step — without changing a single observable value. **Achieved for the
consumption half by #1827/v0.54.10** (V33–V35); the residual scope is the honesty and
pinning around it: the silent-fallback flag, the mechanical bench, the retention row.

**Success Metrics:**
- ✅ landed (#1827) `walk_only.ail` on `--bytecode`: ≤ 2× the interpreter at n = 160,000 — **exceeded and inverted: ~0.24× (the VM is ~4.7× faster), flat scaling** (V33/V34).
- ✅ landed the reporter's shape: repro 6.77 s → 1.05 s on dev (#1827); the 786k-element walk completes in ≤ 2 s on the VM (AC-2b, reporter-attested).
- ✅ landed byte-identical outputs on every fixture and test (AC-5; re-diffed both engines this session, V35).
- ⬜ **open** `ailang run --strict-bytecode` without `--bytecode` no longer silently runs the evaluator (AC-4 → AC-R1).
- ⬜ **open** a committed benchmark (`bench/vm_tail_walk/`, equivalent inputs, **checked** outputs) that pins the regression mechanically (AC-3 → AC-R2).
- ⬜ **open** the `docs/LIMITATIONS.md` retention row + canonical entry, consistent with D-19's parked rows 41–42 (AC-6 remainder → AC-R3).
- Stapledon R2-CREW (AC-2c, ~60 s → < 2 s): the walk half of that deadline is done by #1827; the construction half is D-19's, not re-planned here.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| **Do NOT re-open the representation question (D-19).** `::`/`++` construction stays quadratic until the parked doc's human decision; this doc changes only how *existing* lists are viewed | Two docs claiming the same fix forks the decision record; the parked doc is quorum-blocked on a named human decision | human (already recorded, D-19) | — | high (avoided) |
| **Do not re-plan Phase 1.** The rejected sprint card (`task-fc7451cd`, PR #1750) was rejected because its M2 re-landed the already-merged subslice change; this update plans only the three open items | Planning merged work burns an Opus sprint on a no-op and re-opens settled review surface | human (D-TRIAGE-14, attended 2026-10-16) | recorded | low (avoided) |
| The fix is **sharing, at the builtin**: `_list_tail` returns a view of `elems[n:]` — the exact discipline the interpreter (V2) and Go codegen (V3) already apply — **LANDED #1827/v0.54.10, shipped with capacity caps (`n:len:len`) and the aliasing tests (V41/V42)** | Representation-preserving, one line, precedent on two engines; no arena, no `ListValue`/`ListObj` change | this doc | design | low |
| **`take`/`drop` ride along** in the same commit (same suffix/prefix class, V21) — landed with #1827 (V42) | One class, one fix — the systemic pattern CLAUDE.md principle 3 requires; three one-line changes reviewed under one safety argument | this doc | design | low |
| **`--strict-bytecode` implies `--bytecode` on `run`** (match the `test` command's existing OR, V11/V36) — **OPEN: the residual Phase 2, AC-R1** | A silent evaluator fallback behind a flag named "strict" is the exact anti-pattern CLAUDE.md principle 2 names; `test` already settled the intended semantics | this doc | design | low |
| Retention trade-off is **accepted and documented**, not mitigated: a shared tail pins its source array (bounded by the list's own size) | Identical to interpreter/Go-codegen behavior since inception (V2/V3); today's copy is O(n²) allocation vs O(n) retained — strictly better for walks | this doc | design | low (doc) |

### Design Freeze

- [x] Fix shape frozen: subslice share in `builtinListTail` (+ `builtinListTake`/`builtinListDrop`), clamping semantics (`n<0→0`, `n>len→len`) byte-identical.
- [x] No engine-semantics change: outputs, errors, error text, and determinism guarantees unchanged; only allocation/sharing and cost change.
- [x] `--strict-bytecode`-implies-`--bytecode` is in scope (V11), flagged as an intentional behavior change in Conflict Surface §5.
- [x] `mk`/`::`/`++` construction (the report's other half) is out of scope and stays parked (V23, D-19).
- [x] Bench committed as `bench/vm_tail_walk/` following the `vm_hof_callbacks` min-of-N pattern (V22).

Post-#1827 freeze status: bullets 1–2 **shipped** (v0.54.10 — clamping byte-identical,
outputs unchanged; V41/V42); bullet 4 unchanged and still binding (V37 re-confirms the
construction half quadratic); bullet 3 (strict-flag) is the open Phase 2; bullet 5 is
pending — the bench is not yet committed (V39).

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
   crossings (V26), so sharing stays VM-internal. **— LANDED #1827/v0.54.10:** shipped as
   `NewList(elems[n:len(elems):len(elems)])` (and the `take`/`drop` equivalents), i.e.
   the design above plus a capacity cap per view so an append can never write into the
   source; tests at V42. This paragraph is the historical design, kept for the record.

2. **Phase 2 — `--strict-bytecode` implies the VM on `run` (P1, one line — OPEN, the residual).** The flags reach `runFile` from `cmd/ailang/main_run.go:301`, and the single
   edit site is `cmd/ailang/main_run_exec.go:128`:
   `BytecodeMode: bytecodeMode` → `BytecodeMode: bytecodeMode || strictBytecode` — the
   same OR `ailang test` already applies (`commands_language.go:284`, V11/V36);
   `StrictBytecode` (line 129) already flows through unchanged. The strict flag's help
   (`main_run.go:142`, "With --bytecode: fail instead of falling back") gains "Implies
   --bytecode"; after the change the pairing is enforced instead of assumed, and a
   strict-alone run on an evaluator-only entry fails loudly (AC-R1) instead of
   silently evaluating (V36).

3. **Phase 3 — docs honesty + bench (P1) — the `std/list.ail` half LANDED with #1827
   (V43); the remainder is OPEN.** The header sentence (V14) is already rewritten
   ("Matching `x :: rest` / `[x, ...rest]` does NOT copy … O(n)"), and the `::`/`++`
   construction warning remains (true until D-19). Still open: (a) `docs/LIMITATIONS.md`
   gains one row for the *retention* trade-off — a small suffix/prefix view pins its
   source's backing array, bounded by the source list's own size, interpreter/Go-codegen
   parity since inception — **plus** its canonical entry in
   `docs/docs/reference/limitations.md`, sitting beside rows 41–42 without contradicting
   D-19's pending state (V40); (b) `bench/vm_tail_walk/` commits `walk.ail` (cons-pattern
   walk), `build.ail` (`::` construction — the parked half, kept as the documented
   contrast), `foldl.ail`, and a min-of-N `run.sh` mirroring `vm_hof_callbacks` (V22)
   with **equivalent inputs** (`range`-built lists, the same walk shape as the repro —
   not `consrepro.ail`'s `split(repeat(…))` string lists) and **checked outputs**:
   `run.sh` asserts the printed value equals the expected closed form per (program, n,
   engine) and FAILs on mismatch or nonzero exit, instead of discarding stdout.

### Implementation Plan

**Phase 1 (the fix) — LANDED #1827/v0.54.10; do not re-plan (D-TRIAGE-14):**
- [x] `internal/vm/builtins.go` `builtinListTail`: copy → capacity-capped view `NewList(elems[n:len:len])`
- [x] `internal/vm/builtins_list_poly.go`: `builtinListTake` (`xs[:n:n]`), `builtinListDrop` (`xs[n:len:len]`) → capped views
- [x] Go unit tests: `internal/vm/list_tail_share_test.go` — pointer identity (not allocation counts: "a copy is one make"), clamp parity, cap==len, append isolation (V42)
- [x] Baselines re-derived in the landing sprint; reporter repro 6.77 s → 1.05 s; this session's independent re-measure: V33–V35

**Phase 2 (strict flag) — OPEN (AC-R1):**
- [ ] `cmd/ailang/main_run_exec.go:128`: `BytecodeMode: bytecodeMode` → `bytecodeMode || strictBytecode` (flags already reach `runFile` via `main_run.go:301`; `StrictBytecode` at line 129 unchanged)
- [ ] `cmd/ailang/main_run.go:142`: flag help gains "Implies --bytecode"
- [ ] Test: `ailang run --strict-bytecode` alone on an evaluator-only entry *fails loudly* (exit ≠ 0, `bytecode execution failed: …`) instead of silently evaluating (today: V36); `--bytecode --strict-bytecode`, plain `--bytecode`, no-flag, and `ailang test --strict-bytecode` byte-unchanged

**Phase 3 (docs + bench) — `std/list.ail` half landed; remainder OPEN (AC-R2/AC-R3):**
- [x] `std/list.ail:9-11` header rewrite (#1827, V43) — no further edit
- [ ] `docs/LIMITATIONS.md` retention row **+ canonical entry** in `docs/docs/reference/limitations.md`; rows 41–42 untouched (D-19 pending, V40)
- [ ] `bench/vm_tail_walk/{walk.ail,build.ail,foldl.ail,run.sh}` committed — equivalent inputs (`range`-built), **checked outputs** (expected value asserted per (program, n, engine) + VM↔interpreter diff), min-of-N, both engines; before/after tables in the sprint report (before: V6/V28; after: re-derive V33–V37 on the implementer's build)
- [ ] Changelog fragment `changelogs/unreleased/` with the strict-flag behavior note

**Files to modify/create (residual):** `cmd/ailang/main_run_exec.go` (+1/−1), `cmd/ailang/main_run.go` (flag help, ~1 line), a strict-flag test (new or extended, ~40 LOC), `docs/LIMITATIONS.md` (+1 row) + `docs/docs/reference/limitations.md` (canonical entry), `bench/vm_tail_walk/{walk.ail,build.ail,foldl.ail,run.sh}` (new, ~80 LOC), `changelogs/unreleased/` fragment. **Not re-touched:** `internal/vm/builtins.go`, `internal/vm/builtins_list_poly.go`, `internal/vm/list_tail_share_test.go`, `std/list.ail` — all landed in #1827, re-verified only (AC-R4).

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
4. **Cost-model prose is load-bearing (V14):** `std/list.ail`'s header taught the copy
   as a workaround-generating fact. **Landed with #1827 (V43):** the sentence is now the
   sharing statement; the `::`/`++` construction warning stays (correct until D-19). The
   open prose position is `docs/LIMITATIONS.md` rows 41–42 + the new retention row —
   the new row must not read as "cons fixed" (V37: construction is still quadratic).
   A stale cost warning is a new bug of the same class (A9); a premature *fixed* claim is
   the mirror-image bug.
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
- Comment text in `std/list.ail` (**landed #1827**), one `docs/LIMITATIONS.md` row + canonical entry (**open**), flag help text (**open**).
- Nothing else. `::`/`++` construction stays exactly as it is (D-19).

## Examples

**Before (v0.52.5-era measurements, V6):** the walk on the VM — 90.9 s at 160k, ×3.9
per doubling; the same code on the interpreter: 1.74 s.

```ailang
-- The canonical walk. On --bytecode before #1827: O(n^2). Since v0.54.10: O(n),
-- same output, same determinism, same traces.
pure func countAbove(xs: [float], thr: float, acc: int) -> int =
  match xs {
    [] => acc,
    x :: r => countAbove(r, thr, if x > thr then acc + 1 else acc)
  };
```

**After (measured, V33/V34):** identical program, no rewrite needed — the whole point.
The same walk at 160k on the VM is 0.444 s (was 90.9 s), flat across doublings, and
~4.7× faster than the interpreter's 2.089 s. The consumer's
`foldl` workaround becomes unnecessary but stays valid; `foldl` remains the right tool
when a *callback* shape is natural, and `mapAccumL` when stateful emission is needed
(those recommendations are unaffected — they were about the construction half and
per-step cost, both still true).

## Success Criteria (Acceptance)

**Landed by #1827 (v0.54.10) — verified, not re-planned:**

- [x] **AC-0/AC-1**: baselines re-derived in the landing sprint (reporter repro 6.77 s → 1.05 s); `builtinListTail`/`builtinListTake`/`builtinListDrop` return shared, **capacity-capped** subslices; clamping and error behavior byte-identical (`internal/vm/list_tail_share_test.go`, V42).
- [x] **AC-2/AC-2b**: `walk_only.ail` on `--bytecode` — this session's independent re-measure is **flat** (×≈1 per doubling) and **~0.24× the interpreter wall at 160k** (base: 52× slower) — the ≤2× ceiling is exceeded, not just met (V33/V34); outputs byte-identical (V35). AC-2c's *walk* half is likewise done; the construction half of the ~60 s is D-19's.

**Residual acceptance (the open scope; the sprint-evaluator grades these):**

- [ ] **AC-R1** (= old AC-4, Phase 2): `ailang run --strict-bytecode` without `--bytecode` runs the strict VM — an evaluator-only entry fails loudly (exit ≠ 0, `bytecode execution failed: …`), not a silent evaluator run (today: V36); `--bytecode --strict-bytecode`, plain `--bytecode`, no-flag runs, and `ailang test --strict-bytecode` byte-unchanged; flag help updated.
- [ ] **AC-R2** (= old AC-3, bench): `bench/vm_tail_walk/` committed with **equivalent inputs** (`range`-built lists, the repro's walk shape) and **checked outputs** — `run.sh` asserts each (program, n, engine) prints the expected closed-form value, FAILs on mismatch/nonzero exit, and diffs VM↔interpreter; min-of-N mirroring `vm_hof_callbacks` (V22); before/after tables in the sprint report (before: V6/V28; after: re-derived V33–V37 on the implementer's build).
- [ ] **AC-R3** (= old AC-6 remainder): `docs/LIMITATIONS.md` gains the retention row **plus** its canonical entry in `docs/docs/reference/limitations.md`, worded so it neither contradicts rows 41–42 nor claims the D-19 construction fix (V37/V40).
- [ ] **AC-R4** (no-redo gate): the sprint touches none of `internal/vm/builtins.go`, `internal/vm/builtins_list_poly.go`, `internal/vm/list_tail_share_test.go`, `std/list.ail` — all landed in #1827 and only re-verified; tests stay green.
- [ ] **AC-R5** (= old AC-5): `make test`, `make verify-examples` green; Conflict Surface §"MUST still work" fixtures byte-identical (re-diffed this session, V35); changelog fragment `changelogs/unreleased/` with the strict-flag behavior note.

## Testing Strategy

- **Identity-pinning Go test — LANDED** (`internal/vm/list_tail_share_test.go`, V42):
  pointer identity of the view into the source's backing array (allocation counts cannot
  see this bug — "a copy is one make"), clamp parity, cap == len, append isolation.
- **Aliasing-safety parity** — landed with #1827 and carried forward: `OpCons` onto a tail,
  `sortBy` on a tail, `array_from_list` on a tail → source unchanged; outputs re-diffed
  against the interpreter this session (V35).
- **Clamp/edge parity**: `_list_tail(xs, n)` for n ∈ {0, 1, len, len+1, −1} on both engines,
  error text for non-list input byte-identical (V13 pins).
- **Strict-flag test (OPEN, AC-R1)**: `run --strict-bytecode` alone on an evaluator-only
  entry → the strict error, exit ≠ 0 (today: silently succeeds on the evaluator, exit 0 —
  V36); `run --strict-bytecode` on a VM-runnable file → VM timings, not evaluator timings;
  `ailang test --strict-bytecode` unchanged.
- **Bench (OPEN, AC-R2)**: `bench/vm_tail_walk/run.sh` min-of-N, both engines, n ∈
  {40k, 80k, 160k} for `walk.ail`/`foldl.ail` and {10k, 20k, 40k} for `build.ail` (the
  parked construction half — quadratic, so not scaled), with **checked outputs**: the
  expected value asserted per (program, n, engine) plus a VM↔interpreter output diff, so
  the two halves' costs stay honestly separated in future reports.

## Non-Goals

- **`::`/`++` construction, `OpCons`, the arena, D-19** — parked doc's territory, human decision pending (V23). The report's `mk` half stays quadratic after this doc; the docs say so.
- **Interpreter changes** — it is the reference semantics and already O(1) (V2); its `take`/`drop` Go builtins may copy, which is a (smaller) cost question for a separate audit if a consumer hits it.
- **Go-codegen changes** — already shares (V3).
- **Tail-call elimination** — already present on the VM (V10); the interpreter's TCO shipped separately (m-eval-tail-calls).
- **General VM↔interpreter performance parity program** — only this shape; the construction-side parity belongs to D-19, per [m-foldl-cons-cost-model](../../implemented/v0_52_0/m-foldl-cons-cost-model.md) Phase 3's account.

## Timeline

**Residual (post-#1827):** Phase 2 one-liner + flag help + strict tests ≈ 0.5 day; bench
with checked outputs + before/after tables ≈ 0.25 day; LIMITATIONS row + canonical entry +
changelog ≈ 0.25 day. **~1 day honest**, ×2 → keep the 2-day envelope (measurement
discipline still dominates, as every sibling sprint has found). Phase 1's original ~2
days were spent and landed in #1827.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| An undiscovered write-into-shared-list path makes sharing observable | High (silent wrong values) | The exhaustive audit (V12) with instruments named; two-engine parity differential tests over the full fixture corpus; the aliasing-safety tests in Testing Strategy; interpreter/Go-codegen have run the identical aliasing class in production (V2/V3) — the burden of proof is on "a write exists", not on "sharing is safe" |
| Retention: a long-lived small tail pins a large array | Med (memory) | Bounded by the source list's size (O(n) worst case vs today's O(n²) allocation); identical to interpreter/Go-codegen; documented in LIMITATIONS (Phase 3); a consumer holding `drop(10⁶, xs)` forever can copy explicitly via `take(len, …)` if it ever bites |
| `--strict-bytecode` implies-VM breaks a workflow that relied on the no-op | Med (behavior) | The flag never documented that meaning; `test` already ORs the flags (V11); changelog behavior note; `--bytecode` semantics unchanged |
| Stale docs contradict the fix (std/list.ail header) | Low but real (A9) | Phase 3 lands in the same sprint; AC-6 checks the sentence is gone |
| Reporter timings don't reproduce on the implementer's machine | Low | AC-0 re-derives; thresholds are ratios (×/doubling, VM:interpreter), not absolute seconds |

Post-#1827 status: row 1 is retired (the aliasing-safety tests shipped in #1827, V42; both
engines re-diffed this session, V35), and row 4 is retired for `std/list.ail` (V43) — its
open remainder is the LIMITATIONS row (row 2 → AC-R3). Rows 3 (the strict-flag behavior
change → AC-R1) and 5 (reporter timing → the bench's after-table, AC-R2) remain and are
carried by the residual acceptance.

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

Post-#1827, the open items carry the A9/A11 weight (the strict flag fails loudly instead
of silently running the wrong engine; the retention row makes the sharing trade-off
visible); the A5/A7 weight of the original score is landed and re-measured (V33–V35).

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
- **Landing**: #1827 (Phase 1, shipped **v0.54.10**) — `changelogs/v0.32-current.md`, `## [v0.54.10] - 2026-10-15`; reporter repro 6.77 s → 1.05 s.
- **Residual ruling**: D-TRIAGE-14 (attended 2026-10-16) — update this doc and plan **only** the three open items; the rejected sprint card `task-fc7451cd` (PR #1750) is recorded so its M2 is not re-planned.
- Repro and measurement scripts (`walk_only.ail`, `walk_foldl.ail`, `build_only.ail`, `full_repro.ail`): re-derived 2026-10-13, re-measured 2026-10-16 (V33–V38) — to be folded into `bench/vm_tail_walk/` by AC-R2; disassembly via `ailang disasm`.
- [design_docs/PROGRAM.md](../../PROGRAM.md) §4 — routing lanes. · [Design Axioms](/docs/references/axioms).

---

**Document created**: 2026-10-09 (design-doc-creator, unattended coordinator session `task-006b0353`; live v0.52.5 binary available for measurements, no Go toolchain / no build of this tree — Verification Log provenance marks and W-1 apply)
**Last updated**: 2026-10-13 (design-doc-creator, unattended coordinator session `task-a0ee29fa`: second consumer report v0.54.6 folded in — Status, Problem Statement, Success Metrics AC-2c, Verification Log V27–V32, W-1 update; first-party re-derivation on the tree-matching v0.54.8 binary confirms the fix never landed. No new doc created — duplicate/coverage gate: this doc is the queue entry for the topic.)
**Updated 2026-10-16** (design-doc-creator, **attended** session under **D-TRIAGE-14**): Phase 1 marked landed (#1827 → v0.54.10, V41/V42) and **not re-planned** (the rejected card `task-fc7451cd` / PR #1750 recorded — its M2 re-landed it); scope cut to the residual — Phase 2 strict-flag (exact edit site `cmd/ailang/main_run_exec.go:128`), `bench/vm_tail_walk/` with equivalent inputs and checked outputs, `docs/LIMITATIONS.md` retention row + canonical entry consistent with D-19's parked rows 41–42; the cons-walk re-measured on current dev **first** (V33–V38, v0.54.11 binary) and the residual gap stated against the interpreter — **inverted: the VM is ~4.7× faster at 160k, flat net of startup**; residual acceptance AC-R1–AC-R5. No new doc created — this doc remains the queue entry for the topic (V32).

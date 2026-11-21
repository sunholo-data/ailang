# M-VM-LIST-CONS-QUADRATIC: O(1) prepend (`x :: acc`) on the strict bytecode VM — cons-building folds are O(n²) on the one engine no queued doc owns

**Status**: Planned — quorum not run this session (no reviewer route configured in this container;
re-run at sprint-planning per the sibling-doc convention
([m-vm-list-tail-share](../../planned/v0_53_0/m-vm-list-tail-share.md) Status header)). The
28-row first-party Verification Log below is the load-bearing evidence; **every measurement was
re-derived on a live binary of this exact tree** (v0.57.0, `b23cadce`, `/usr/local/bin/ailang`,
no Go toolchain in the container — source claims are reads at this tree, measurements are
first-party runs, W-1 below).
**Target**: v0.57.1 (next feature release)
**Priority**: P0 — the headline engine of this report is the strict bytecode VM, the reported shape is the canonical accumulator idiom, and this is the **third independent downstream consumer report** of the construction half ([#676] email-parse 2026-08; the v0.51/v0.52 foldl-cons reports; now sunholo/audio 2026-11)
**Estimated**: 4–5 days (Phase 1 spike ~1 d · Phase 2 accessors+migration ~1.5 d · Phase 3 swap ~1 d · Phase 4 benches+docs ~0.5 d · Phase 5 regression surface ~0.5 d), ×2 honest where noted
**Dependencies**: None hard. **Coordination required**: this doc implements the *ruled* representation direction (D-19 : B, D-22 : C1) on the VM only; it does not depend on, block, or re-open the evaluator programme's queued pieces (LC-2…LC-5). **Explicitly out of scope**: `eval.ListValue` (owned by [m-list-cons-cells-decomposition](../m-list-cons-cells-decomposition.md)).
**Planner-Lane**: opus-required (VM core value representation — `bytecode.ListObj` + `OpCons`)
**Bug report**: AILANG v0.54.18 (`cd33d05`, macOS arm64 M4 Max, binary md5 `fd40f17194318d2650e53dac2eac84ba`), reported by `stapledons_godot` via `ailang messages`, found while executing stapledons-godot R3-SUNHOLO-AUDIO-REALISM. Repro (verbatim, type-checks clean at HEAD, V4):

```ailang
module rec
import std/list (foldl, range, length)
export pure func plain(n: int) -> string = show(length(foldl(\a x. x :: a, [], range(0, n))))
export pure func scalar(n: int) -> string = show(foldl(\a x. a + intToFloat(x), 0.0, range(0, n)))
```

| Reporter measurement (v0.54.18, their machine) | Value |
|---|---|
| `--bytecode --strict-bytecode`, `plain`, N=40,000 / 80,000 | 1.26 s / 4.91 s (user 8.1 s) — ×3.9 per doubling → **O(n²)** |
| `--bytecode --strict-bytecode`, `scalar`, N=80,000 | 0.02 s (both sizes) — linear |
| record/tuple accumulator holding the list (`{r, out: r :: s.out}` shape) | 1.24 s / 5.06 s — same curve |
| interpreter (no flag), `plain`, N=20,000 / 40,000 | 0.27 s / 0.84 s — ×3.1 → superlinear |
| Downstream: sunholo/audio 0.6 tests — Schroeder backward-integration helper, 72,000 samples | **13.8 s** with a foldl-cons accumulator; **0.05 s** rewritten with map/mapAccumL |

The reporter's expectation is correct and stated plainly: *"Expected: O(n), because each step is one cons."* Downstream: *"we avoid foldl-cons accumulators and are not working around it otherwise"* — they already use the shipped mitigation ([`mapAccumL`](../../implemented/v0_52_0/m-foldl-cons-cost-model.md) Phase 2) and are reporting the substrate defect, not asking for coaching.

## Routing Lane (per design_docs/PROGRAM.md §4)

**Lane: AILANG fix.** This report has two engine halves, and they route differently:

| Half | Owner | Routing |
|---|---|---|
| Interpreter (`listConsImpl` copies the tail) | **Queued already** — the quorum-cleared evaluator programme: D-19 ruled **B** (2026-08-19, #745), LC-1 spike ran and returned **GO** (v0.34.0), D-22 ruled **C1** (2026-08-22), LC-2 (`m-list-accessor-api`) and LC-3a/b/c → LC-4 → LC-5 are queued (V16–V18) | **Not re-planned by this doc.** The programme is decided and merely stalled at LC-2; a second doc for the same fix would fork the decision record (the exact anti-pattern [m-vm-list-tail-share](../../planned/v0_53_0/m-vm-list-tail-share.md)'s gate section names) |
| Strict bytecode VM (`OpCons` copies the tail) | **Nobody** — every queued doc explicitly excludes it (V19) | **This doc.** The exclusion rested on a premise that is now false |

**Why the VM exclusion premise is dead.** The parked substrate doc deferred VM `OpCons` with
"ailang run does not use the VM" ([m-list-cons-quadratic](../m-list-cons-quadratic.md), V22
context). Since then, two things happened: (1) v0.54.10's M-VM-LIST-TAIL-SHARE made
`--strict-bytecode` imply the VM on `run` (residual Phase 2, landed — verified at HEAD, V6/V20),
and (2) this report arrives **from the strict VM as its primary engine**, from a consumer that
runs `ailang run --bytecode --strict-bytecode` in its test suite. The programme-level
decomposition repeats the exclusion as a non-goal with no premise attached (N11; LC-4 "Does NOT
touch the VM (`OpCons` stays O(n), documented divergence)"). A non-goal whose reason no longer
holds is a gap, not a decision — and `docs/LIMITATIONS.md` row 40 promises "the copy itself goes
away only with the cons representation fix (D-19)", which will remain **false on the VM even
after the entire evaluator programme lands** (V21). This doc closes that gap under the already-ruled
direction; it re-opens nothing.

**Extension test fails**: no motoko extension can change what `OpCons` allocates inside the VM,
and the reporter is not behind motoko. Coaching "never cons" is the anti-A7 answer — the language
teaches `x :: acc` as the canonical accumulator, the evaluator programme has already ruled it
O(1)-worthy, and the stdlib header itself calls it "the fundamental list construction operation"
(`internal/builtins/list.go:50`).

## Problem Statement

Building a list by prepending inside a fold — `foldl(\a x. x :: a, [], xs)`, the canonical
functional accumulator idiom — costs **Θ(n²) time and allocation on the strict bytecode VM**,
because the VM represents lists as a flat Go slice (`bytecode.ListObj{Elems []Value}`,
`internal/bytecode/value.go:101-103`) and `OpCons` **copies the entire tail on every prepend**
(`internal/vm/vm.go:511-521`, V1): one fold step is `make([]bytecode.Value, 0, len(old)+1)`,
append head, append the whole tail. The fold step compiles to exactly one `CONS` instruction
(disassembly, V5), so the quadratic is entirely the copy. A scalar accumulator over the same
range is flat (V2) — the defect is specific to list-typed accumulation, i.e. to `::`, exactly as
the reporter isolated it.

The same is true on the interpreter (`listConsImpl`, `internal/builtins/list.go:80-103`, V22) —
that half is the parked evaluator programme's defect (#676), re-reported here only as context; at
HEAD the interpreter is also quadratic (V3).

**First-party re-derivation at HEAD (this container, v0.57.0, min-of-3, wall seconds, ~0.33 s
startup included; the reporter's M4 Max is ~5× faster so ratios transfer, absolutes do not):**

| Engine | Shape | N | wall (s) | scaling |
|---|---|---|---|---|
| strict VM (`--bytecode --strict-bytecode`) | `plain` (cons-fold) | 20,000 / 40,000 | 1.883 / 6.618 | ×3.51 per doubling → **O(n²)** (V2) |
| strict VM | `scalar` | 80,000 | 0.346 | flat vs 20k-class sizes — linear (V2) |
| strict VM | record accumulator `{out: x :: r.out}` | 20,000 / 40,000 | 1.876 / 6.414 | ×3.42 → **O(n²)** — the record wrapper is irrelevant; the cons is the cost (V7) |
| `--strict-bytecode` alone | `plain` | 40,000 | 6.704 | ≈ the VM's 6.618, **not** the interpreter's 3.256 — implies-VM landed (V6) |
| interpreter (no flag) | `plain` | 40,000 / 80,000 | 3.256 / 12.406 | ×3.81 → **O(n²)** (V3; parked programme's half) |
| interpreter | `scalar` | 80,000 | 0.489 | linear (V3) |

**Current state of the surrounding work** (why this doc exists and why it is scoped to the VM):

- The **consumption half** of the same idiom — walking a list by cons-pattern tail recursion —
  was fixed on the VM in v0.54.10 (#1827, [m-vm-list-tail-share](../../planned/v0_53_0/m-vm-list-tail-share.md)):
  `_list_tail`/`take`/`drop` return shared capacity-capped views, and the walk is now **flat and
  ~4.7× faster than the interpreter**. The VM is now *faster than the interpreter on list walks
  and 2× slower on list building* — the construction half is the remaining asymmetry.
- The evaluator's substrate fix is **decided but stalled**: D-19 : B → LC-1 spike **GO** (chosen C2K32
  by tie-break, then D-22 ruled **C1**, plain cons cells, 2026-08-22) → LC-2 planned since v0.35.0
  and still unexecuted at v0.57.0 (V16–V18). The reporter cannot wait on that queue for the VM
  half, because **the VM half is not in that queue at all** (V19).
- The shipped mitigations — `map`/`filter`/`mapAccumL` for building, honest stdlib cost comments —
  are in place and the consumer uses them. They are the **workaround**, not the fix: `::`'s own
  doc says it "constructs a new list … the fundamental list construction operation" with no
  cost warning, and the reporter's Schroeder helper was *correct AILANG* made unusable.

**Impact**: any AILANG program on the strict VM that grows a list one element at a time —
foldl-cons, foldl over a record/tuple holding a list, hand-rolled prepend-then-reverse, `[x] ++ acc`
recursion — pays n²/2 element copies plus the GC cycles they trigger. The sunholo/audio 0.6
realism pass hit exactly this: a backwards-integration sweep over 72,000 samples took 13.8 s and
had to be rewritten to `mapAccumL` (0.05 s) — a 276× penalty for writing the idiom the language's
own teaching surfaces show first.

## Verification Log

Provenance: **[R]** = read first-party at this tree (`b23cadce`, HEAD, v0.57.0, clean), command/location
cited; **[M]** = measured this session on `/usr/local/bin/ailang` v0.57.0 (`b23cadce…-dirty`,
same commit as this tree; `ailang version` → v0.57.0/b23cadc); **[N]** = negative-existence claim
with the instrument named (per the skill's rule, every "no/none/not" sentence gets a row);
**[P]** = reporter's measurement (v0.54.18, md5-attached in the report), inherited, not re-run.

| # | Claim | Command / location | Observed |
|---|---|---|---|
| V1 [R] | VM `OpCons` copies the whole tail per prepend | `internal/vm/vm.go:511-521` | `old := tail.AsList(); elems := make([]bytecode.Value, 0, len(old)+1); elems = append(elems, head); elems = append(elems, old...); NewList(elems)` — O(n) per cons |
| V2 [M] | Cons-fold is quadratic on the strict VM; scalar fold is linear | `ailang run --quiet --bytecode --strict-bytecode --entry plain\|scalar --args-json N rec.ail`, min-of-3, this container | plain 20k = 1.883 s, 40k = 6.618 s (×3.51); scalar 80k = 0.346 s (flat); hello-world baseline 0.334 s |
| V3 [M] | Interpreter is quadratic too (parked programme's half, context only) | same file, no flag, min-of-3 | plain 40k = 3.256 s, 80k = 12.406 s (×3.81); scalar 80k = 0.489 s |
| V4 [M] | The reporter's repro type-checks at HEAD | `ailang check /tmp/repro/rec.ail` | `✓ No errors found!` (one MOD010 temp-path auto-relax note, no errors) |
| V5 [M] | The fold step compiles to exactly one `CONS` — the copy is the only superlinear op | `ailang disasm rec.ail` → `Prototype 56: rec.plain$lambda56` | `params=2 regs=3 captures=0 insns=2: 0000 CONS r2, r1, r0 / 0001 RETURN r2`; the caller is native `__list_foldl` (`CLOSURE r6, P1 ; std/list.foldl`, `CALL r6, args=3`) |
| V6 [M/R] | `--strict-bytecode` alone runs the VM at HEAD (implies-VM landed) | `ailang run --quiet --strict-bytecode --entry plain --args-json 40000 rec.ail`, min-of-3 = 6.704 s ≈ VM 6.618 s ≠ interpreter 3.256 s; source `cmd/ailang/main_run_exec.go:128` | `BytecodeMode: bytecodeMode \|\| strictBytecode` — the tail-share residual Phase 2 edit, present at HEAD |
| V7 [M] | A record accumulator wrapping the list is the same curve — root cause is the cons, not the bare-list accumulator | `rec2.ail` (`step(r:{out:[int]}, x:int) = {out: x :: r.out}`, `ailang check` ✓), strict VM min-of-3 | 20k = 1.876 s, 40k = 6.414 s (×3.42) — matches V2 within noise |
| V8 [R] | VM list representation is a flat slice behind an exported field | `internal/bytecode/value.go:98-103, 211-213, 345-352` | `type ListObj struct { Elems []Value }`; `NewList(elems) → &ListObj{Elems}`; `AsList()` returns the raw slice; comment: "The design doc (§3.2) leaves the choice of cons-cell vs slice open; Phase 2B uses slices for cache locality and simple indexing. Revisit in Phase 2D if benchmarks demand." |
| V9 [R] | VM list concat (`++`) is a native builtin copying both operands | `internal/vm/builtins.go:459-473` (`builtinConcatList`), registered at `:84`; `internal/bytecode/builtin_names.go:26` (`_concat_List`) | `result := make(…, len(a)+len(b)); copy(result, a); copy(result[len(a):], b)` — O(len(a)+len(b)) per call; `[x] ++ acc` growth is the same quadratic class |
| V10 [R] | `OpConcat` is string-only — list `++` never touches it | `internal/vm/vm.go:236-245` | `case bytecode.OpConcat:` requires `TagString` on both operands, errors otherwise |
| V11 [R] | The VM HOF fold iterates the flat slice and re-enters user code per element | `internal/vm/builtins_hof.go:64-80` (`hofBuiltinListFoldl`) | `for i, e := range args[2].AsList() { acc, err = caller.CallClosure(fn, argv.of2(acc, e)) }` — the per-step `CONS` (V5) is what V1 makes expensive |
| V12 [N] | **No VM path mutates a list's backing slice in place** (safety precondition for any sharing representation) | inherited exhaustive audit [m-vm-list-tail-share V12](../../planned/v0_53_0/m-vm-list-tail-share.md) (landed with #1827); re-spot-checked: `grep -rnE 'AsList\(\)\[[^]]*\] *=' internal/vm/` → 0; every producer writes a fresh `make`/pre-sized result (builtins_hof.go, builtins_list.go, builtins_list_poly.go) | zero in-place writes; sharing is the norm since v0.54.10 (`list_tail_share_test.go` pins append-isolation via capacity caps) |
| V13 [R] | Bridge deep-copies elementwise in both directions — engine representations are decoupled | `internal/vm/convert.go:40-51` (list case: fresh `dst := make([]eval.Value, len(src))`, per-element `bytecodeToEval`); reverse direction same pattern | a VM representation change cannot be observed across the bridge |
| V14 [N] | List values never appear in the image/constant pool — lists are runtime-only | `grep -rn "TagList" internal/bytecode/*.go \| grep -v _test \| grep -v value.go` → 0 hits outside value.go; `ailang disasm rec.ail` shows `MAKE_LIST r1, src=r0, count=0` for literals | constants are scalars/strings; list literals compile to `OpMakeList` (`internal/vm/vm.go:435`) — the image serialization surface is untouched by a `ListObj` change |
| V15 [R] | The cons-pattern walk is already O(1) per step on the VM (the consumption half is done) | `internal/vm/builtins.go:395-420` (`builtinListTail` → `NewList(elems[n:len(elems):len(elems)])`, #1827); lowering emits `_list_get`/`_list_tail` per match ([m-vm-list-tail-share] V4) | pre-conditions for this doc: the walk half must stay O(1) post-swap — `_list_get(s,0)` and `_list_tail(s,1)` must both remain O(1) on the new representation |
| V16 [R] | D-19 ruled **B** (true cons cells), superseding the front-slack arena doc | [m-list-cons-cells-decomposition](../m-list-cons-cells-decomposition.md) ("Mark ruled B — comment by MarkEdmondson1234 on #745 at 2026-08-19T10:58:40Z, body exactly `D-19 : B`") | This doc implements B's invariant on the VM; it re-litigates nothing |
| V17 [R] | LC-1 spike ran, returned **GO**; the evaluator programme is unblocked, not dead | `design_docs/implemented/v0_34_0/m-list-repr-spike-M6-report.md` ("### **GO** — chosen representation: **C2K32**… The verdict is a GO: LC-2…LC-5 are unblocked and D-19 is not re-opened"); harness retained at `tools/internal/spike-listrep/` (README: "THROWAWAY — DO NOT IMPORT… retained only so the spike measurements can be reproduced") | the spike harness, its clause structure and its C0 control are reusable for the VM arm (Phase 1) |
| V18 [R] | D-22 ruled **C1** (plain cons cells) for the evaluator; LC-2..LC-5 remain unexecuted at v0.57.0 | [m-list-accessor-api](../m-list-accessor-api.md) header ("**D-22 = `C1`** (Mark, 2026-08-22)"); `ls design_docs/implemented/ \| grep -c "list-accessor\|list-migrate\|list-cells-swap"` → 0; `listConsImpl` still copies at HEAD (V22) | the evaluator queue is decided and stalled, not abandoned; this doc must not fork it |
| V19 [N] | **No queued or implemented doc owns VM-side O(1) prepend** | Read in full: [m-list-cons-quadratic](../m-list-cons-quadratic.md) (Non-Goals: "VM `OpCons` — stays O(n) (V22); `ailang run` doesn't use the VM; parity deferred"), [m-list-cons-cells-decomposition](../m-list-cons-cells-decomposition.md) (N11: "Bytecode VM + bridge — Decoupled; `OpCons` divergence documented, non-goal"; LC-4: "Does NOT touch the VM"), [m-foldl-cons-cost-model](../../implemented/v0_52_0/m-foldl-cons-cost-model.md) (Phase 3 hands the H2 residue to "the representation decision (D-19's doc already tracks VM parity as future work)" — but D-19's docs exclude the VM), [m-vm-list-tail-share](../../planned/v0_53_0/m-vm-list-tail-share.md) (Non-Goals: "`::`/`++` construction … parked doc's territory") | four docs, four exclusions, zero owners — the VM construction half is the gap this doc fills |
| V20 [R] | The exclusion's premise ("`ailang run` doesn't use the VM") is false at HEAD | `ailang run --help` (`-bytecode`, `-strict-bytecode` "Fail instead of falling back; implies --bytecode"); `cmd/ailang/main_run_exec.go:128` (V6); this report's repro command is `ailang run --bytecode --strict-bytecode` | the deferral condition named in the parked doc's own Future Work ("VM `OpCons` arena parity … if/when the bytecode engine fronts `ailang run`") is now met |
| V21 [R] | LIMITATIONS' **VM row** promises the copy goes away with D-19 — a promise the D-19 programme will not deliver to the VM | `docs/LIMITATIONS.md:40` ("Quadratic `::` prepend …" row, engine-agnostic cost account) and `:43` (**the `--bytecode` row**: "the copy itself goes away only with the cons representation fix (D-19)") | row 43 is *specifically the VM row*, yet the D-19 programme it points at explicitly excludes the VM (V19) — this doc's Phase 4 is what makes that sentence true |
| V22 [R] | The evaluator half is still live at HEAD (context for the parity AC) | `internal/builtins/list.go:86-103` (`listConsImpl`: `make([]eval.Value, 0, 1+len(tail.Elements))` + `append(result, tail.Elements...)`) | same copy shape as V1; owned by LC-2..LC-5, untouched here |
| V23 [R] | `bytecode.Value` is 40 B packed (why the VM's quadratic is *worse* than the interpreter's per copy) | `internal/bytecode/value_test.go:21-30` (`TestValueSize`: "list cons and list builds copy whole Values, so the VM's bytes-per-element on copy-heavy shapes is this size"; asserts `unsafe.Sizeof(Value{}) == 40`) + `internal/bytecode/value.go:73-88` ("a Value is 40 bytes … Every list cons / list build copies whole Values") | 40 B vs the interpreter's 16 B interface header — measured consequence: consrepro gap attributed ≥99.9% to OpCons bytes ([m-foldl-cons-cost-model] Phase 3 profile) |
| V24 [N] | No `go` toolchain / `make` in this container — the doc's "after" numbers are AC targets, not results | `which go make` → empty | Phase 1's spike and all ACs are re-derived by the implementer on a live build (AC-0); source claims are reads at `b23cadce`, measurements on the same-commit binary |
| V25 [R] | Blast radius of the VM swap (migratable consumer surface) | `grep -rn "AsList()" internal/vm/*.go \| grep -v _test \| wc -l` → **40 sites across 12 files** (9 builtins_list, 7 builtins, 6 builtins_hof, 3 builtins_string, 3 builtins_hof_list, 2 each vm/builtins_yaml/builtins_xml/builtins_map_bulk/builtins_json, 1 convert, 1 builtins_list_poly); `grep -rn "NewList(" internal/vm/*.go \| grep -v _test \| wc -l` → **33 producer sites** | the VM's consumer surface is ~20× smaller than the evaluator's (902 `.Elements` refs) — an LC-2/LC-4-shaped migration is a 1–2 day piece here, not a programme |
| V26 [R] | The pointer-identity tests from #1827 pin the *current* representation and must be adapted, not deleted | `internal/vm/list_tail_share_test.go:20-40` | `TestListTailDoesNotCopy` asserts `&got[0] == &src[1]`-class identity into the flat backing array — with cells the equivalent pin is tail-cell identity (same invariant, different instrument); clamp/cap/append-isolation semantics must stay |
| V27 [R] | Regression fixtures exist and exercise cons construction, cons patterns and O(1)-step folds | `ls examples/first_non_repeat.ail examples/record_cons_pattern.ail examples/pattern_matching_adt.ail examples/runnable/mapAccumL_running_total.ail examples/runnable/no_loops_fold.ail` | all five exist (V31 of the parked doc + the mapAccumL example shipped v0.52) |
| V28 [P] | Reporter's measurements (v0.54.18, md5-attached) | table in the Bug report section | quadratic on the strict VM and the interpreter; scalar linear; Schroeder helper 13.8 s → 0.05 s after mapAccumL rewrite; ratios reproduce first-party at HEAD (V2/V3/V7) |

### Workarounds note (W-1)

This container has the **live v0.57.0 binary of this exact tree** (measurements above are
first-party, unlike the tail-share session's near-lineage binary) but **no Go toolchain**
(V24): the "after" numbers in Success Criteria are AC targets; the implementer re-derives them
on `make build` as AC-0. Quorum was not run (no reviewer route configured here); the
sprint-planner re-runs it per the sibling-doc convention. The `ailang check` hard gate **was**
run (V4, V7) — every AILANG snippet in this doc type-checks.

## Goals

**Primary Goal:** `x :: xs` is O(1) worst-case time and allocation on the strict bytecode VM,
for every live list `xs`, under every sharing pattern — the same invariant INV-1 the evaluator
programme bought with D-19 : B — so the reporter's repro (`foldl(\a x. x :: a, [], range(0, n))`)
is O(n) end-to-end on `--bytecode --strict-bytecode`, with zero observable value change.

**Success Metrics:**
- The repro at N=80,000 on the strict VM scales flat: doubling N ≤ ~2.5× wall (base: ×3.5–3.9
  per doubling; V2), reaching parity-class cost with the O(1)-step `scalar` fold (AC-1).
- Allocation for the repro is O(n): bytes allocated for an n-cons build ≤ a small constant × n
  (base: Θ(n²) — 40 B/element × n²/2 copies; AC-2).
- m-fold prepends onto one retained base cost O(m) — the D-19 objection that killed the arena
  design (branching) is satisfied by construction, not by a use-chain caveat (AC-3).
- `++` improves to copy-left/share-right: `small ++ big` costs O(len(small)) (AC-4).
- The walk half stays O(1) per step — #1827's win is not regressed (AC-5).
- Every fixture, test and the evaluator agree byte-for-byte on outputs (AC-6/AC-7).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **The VM representation follows the ruled evaluator direction: true cons cells (C1 shape), not an arena and not a new invention.** D-19 : B bought INV-1 under *all* sharing; the arena was declined precisely because it only holds along a linear use chain | Two engines with different cost asymptotics for the same operator is a standing parity trap (`--bytecode` silently falls back to the evaluator — the same program can be fine one run and quadratic the next) | human (already ruled — D-19 : B, D-22 : C1; recorded there, applied here) | design | high |
| **A VM-specific mini-spike (Phase 1) re-ratifies C1 on `bytecode.Value` before the swap**, reusing the retained LC-1 harness pattern and clause thresholds, with C1 as the presumptive winner per D-22 and C2K32 as the alternative arm | The evaluator's D-22 arithmetic was done against 16 B `eval.Value` heads; a `bytecode.Value` head is 40 B (V23) and the memory clause (c) ceiling may rank candidates differently — the ruled *direction* (B) is not re-opened, only the C1-vs-C2K32 ranking is re-measured on the VM's own value width | agent (spike decides, kill criterion bounds it) | Phase 1 | low |
| **The swap is proven by deleting the `Elems` field** (compile-time migration proof, the LC-4 trick) — no new analyzer for the VM | The VM's consumer surface is 40 sites in 12 in-repo files (V25), all reachable by the compiler; an unexported-field accessor layer + field deletion proves completeness mechanically; a `listrep`-style analyzer is the *evaluator* programme's deliverable (LC-2), not needed at this scale | this doc | design | med |
| **`_concat_List` becomes copy-left/share-right** (O(len(left)) worst case, O(1)-ish for `small ++ big`), riding in the same swap | `++` is the same quadratic class (V9) and `std/list`'s right-recursion builders use exactly `small ++ grow` shapes; one representation change fixes both; left-append growth (`acc ++ [x]`) stays O(len(acc)) — inherent, documented | this doc | design | low |
| **Indexed access on lists becomes O(i)** (`GET_INDEX`, `_list_nth`, `builtinListGet`); indexing-heavy workloads route to `ArrayValue` (already documented "O(1) indexed access") — the LC-5 disposition, adopted here | The cost D-19 : B knowingly buys; hiding it would violate A9. The VM must document the same trade the evaluator programme will | human (ruled direction implies it; D-22's spike measured `nth` clause (b) ≤ 2.0× C0) | design | low |
| **The stdlib header + LIMITATIONS sentences become engine-qualified** (Phase 4), not deleted | After this doc the sentences are half-true: false on the VM, still true on the evaluator until LC-4 lands; premature deletion is the mirror-image of the dishonesty m-foldl-cons-cost-model Phase 1 fixed | this doc | Phase 4 | low |

### Design Freeze

All high-change-cost rows are resolved before sprint execution; the design review / quorum ratifies:

- [x] Representation direction: true cons cells with structural sharing (D-19 : B applied to
      `bytecode.ListObj`) — no arena, no lazy materialization (the parked doc's Options C/D
      analysis transfers; it is not re-argued here).
- [x] C1-vs-C2K32 on 40 B values is settled by the Phase 1 spike's ratified clause thresholds;
      on STOP, the piece reports back and does **not** proceed (the programme-level kill
      discipline, applied at piece scale).
- [x] `Elems` is deleted at the end of the migration; accessors are the only surface.
- [x] Scope fence: `eval.ListValue`, `internal/builtins/list.go`, the parked/queued evaluator
      programme, and `--emit-go` output are untouched.
- [x] Parity contract: outputs byte-identical; per-op *costs* may differ between engines
      (they already do — the walk is VM-faster since v0.54.10).

## Deferred Decisions

Intentionally open for the implementer:

- Accessor surface names on `bytecode.ListObj` (`Len`, `Head`, `Tail`, `Nth`, `Iter`,
  `FromSlice`, `Materialize`) — agent may choose; the LC-2 draft API (`m-list-accessor-api.md`)
  is input, not contract, since it was designed for `eval.Value`.
- Cell layout details (single `*cell` type with `head Value; tail *ListObj` vs fat sentinel) and
  whether `Value.Obj` points at a spine node or the `ListObj` wrapper survives — agent.
- Materialization strategy for bulk consumers (e.g. JSON/XML/YAML encoders, `sortBy`): per-call
  `Materialize()` vs an iterator protocol — agent; LC-1's materialize benchmark arms the decision.
- Whether `_list_nth`/`last` gain a spine-walk builtin (the LC-5 open question) — agent, gated by
  the spike's nth clause.
- Bench location/name under `bench/` (following `vm_tail_walk`/`vm_hof_callbacks` conventions) — agent.
- Optional debug escape hatch (`AILANG_VM_LIST_…`) — agent; if added it is a perf toggle, not a
  data-integrity fallback (no silent fallbacks — a bad flag fails loudly).

## Solution Design

### Overview

The evaluator's whole-programme answer to "O(1) cons" is D-19 : B → LC-1 spike (GO) → D-22 (C1)
→ LC-2 accessors → LC-3 migration → LC-4 swap → LC-5 tuning. The VM was excluded from all of it
on a premise that no longer holds (V19/V20). This doc runs the **same play, compressed to one
sprint**, because the VM's list surface is ~20× smaller (V25) and its concurrency story is trivial
(the VM is single-goroutine; the bridge deep-copies crossings, V13):

1. **Phase 1 — mini-spike (go/no-go gate, ~1 d).** Extend the retained LC-1 harness
   (`tools/internal/spike-listrep/`, V17) with `bytecode.Value`-typed arms — C0 (slice control,
   mirrors today's `OpCons`), C1 (cons cells), C2K32 (chunked) — and re-run the five ratified
   clauses (a) linear-build, (b) ns/element iteration, (c) B/element memory vs C0, (d) Len scaling,
   (e) encapsulation. D-22 chose C1 on 16 B values; on 40 B values clause (c)'s 2.5× ceiling is
   the one that may re-rank (a cell now carries a 40 B head, so C1's B/element ratio roughly
   doubles). Verdict + arithmetic land in this doc's Implementation Report; **STOP** (no swap)
   re-opens nothing but reports the table to the coordinator.
2. **Phase 2 — accessor layer + migration (~1.5 d).** Give `bytecode` lists an
   unexported-representation API (`FromSlice`, `Len`, `Head`, `Tail`, `Nth`, iterate,
   `Materialize`, `Cons`) and migrate the 40 `AsList()` consumers + 33 `NewList` producers
   (12 files, V25). Behavior-identical: each consumer keeps its current semantics, only the
   element access goes through the seam. The field is *frozen* but still present during this
   phase — every migration commit is green and independently reviewable.
3. **Phase 3 — the swap (~1 d).** Replace `ListObj`'s innards with the Phase-1-chosen cells;
   `OpCons` becomes allocate-one-cell (V1's copy dies); `_concat_List` copies the left spine and
   shares the right tail; `builtinListTail`/`take`/`drop` views become O(1) tail/cell views
   preserving #1827's clamping and append-isolation semantics (V26); **delete `Elems`** — the
   compiler then proves the migration missed nothing.
4. **Phase 4 — benches + docs honesty (~0.5 d).** Commit `bench/vm_cons_build/` (checked outputs,
   min-of-N, both engines: cons-fold, record-fold, `[x] ++ acc`, branching-cons, walk, scalar);
   engine-qualify the `std/list.ail` header sentence and LIMITATIONS rows 40/43 (V21); CHANGELOG.
5. **Phase 5 — regression surface (~0.5 d).** Fixtures, two-engine output diff, full targeted
   test set; Implementation Report with the measured before/after.

### Why not the alternatives

- **Wait for the evaluator programme (LC-4) and share its representation package** — the
  programmes' timelines should not be coupled: LC-2 has been queued since v0.35.0 and its accessor
  API is designed for `eval.Value` (16 B interface elements), not `bytecode.Value` (40 B structs);
  a shared generic spine is a new programme, not a sprint, and would re-open quorum-cleared surface.
  The bridge already decouples the engines (V13) — independent representations are the designed state.
- **The declined arena (linear-chain front-slack)** — declined by D-19 for the evaluator on a
  correct objection (branching → Θ(m·len(base))); the objection transfers unchanged. Not re-argued.
- **`mapAccumL` coaching only** — shipped and the reporter already uses it; it is the workaround
  for *stateful emission*, not a fix for `::`, which remains "the fundamental list construction
  operation" with an honest-but-quadratic implementation. The substrate punishes the canonical
  idiom (A7); the third report is the evidence that coaching does not retire the defect.
- **Do nothing because the docs are now honest** — the honesty fix (m-foldl-cons-cost-model
  Phase 1) was the *first* thing shipped from this family, three reports ago. It demonstrably
  does not close the gap on the VM, where the idiom is now 2× *slower* than the interpreter
  (the inversion of the walk half's 4.7×-faster).

### Architecture (post-swap shape)

```go
// internal/bytecode/value.go — representation becomes unexported; consumers use accessors
type ListObj struct {
    // C1 (presumptive per D-22): spine of immutable cells.
    head Value    // valid when tail != nil; zero Value for the empty list
    tail *ListObj // nil == empty list; shared across all lists with this suffix
    n    int      // cached length: Len() stays O(1) (LC-1 clause (d))
}
```

- `Cons(head, tail)`: one allocation; no CAS, no watermark — immutability is structural
  (single-goroutine VM, V12/V13; nothing can observe a partially-built cell).
- `_concat_List`: walk-copy the left operand's cells onto the right operand's spine —
  O(len(left)), shares `right` entirely.
- `Head`/`Tail`/pattern lowering (`_list_get(s,0)`, `_list_tail(s,1)`, V15): O(1) — the walk half
  keeps its #1827 asymptotics.
- `Nth`/`GET_INDEX`: O(i) spine walk (documented; `ArrayValue` is the O(1)-indexed alternative).
- Bulk consumers (`sortBy`, JSON/XML/YAML encoders, `array_from_list`, the bridge): `Materialize()`
  or iterate — same big-O as today (they were already O(n) passes over the slice).
- Empty list: a package-level singleton (saves one allocation per `[]` literal and makes
  `tail([])`'s clamping trivially stable).

### Pipeline Pass Coverage

Not applicable — no new compiler pass. The change is confined to `internal/bytecode` value
machinery + `internal/vm` consumers; the emitted instruction stream is **unchanged** (`CONS`,
`MAKE_LIST`, `BUILTIN_CALL` stay; only what they allocate changes — the tail-share precedent, its V38).

## Files to Modify/Create

**New:**
- `internal/bytecode/list_accessors.go` (~150 LOC) — the accessor seam (Phase 2) and, after
  Phase 3, the only window into `ListObj`
- `internal/bytecode/list_accessors_test.go` (~200 LOC) — cons O(1)/sharing/branching identity
  tests, clamp parity, len, materialize; adaption of the V26 identity pins
- `tools/internal/spike-listrep/vmarm/` (~250 LOC + benches) — the `bytecode.Value`-typed spike
  arms (throwaway, same DO-NOT-IMPORT discipline as the existing harness)
- `bench/vm_cons_build/{cons_fold.ail, rec_fold.ail, concat_acc.ail, branch_cons.ail, walk.ail, scalar.ail, run.sh}` (~120 LOC) — checked outputs, min-of-N, both engines
- Changelog fragment under `changelogs/unreleased/`

**Modified:**
- `internal/bytecode/value.go` (+~60/−10 LOC) — `ListObj` swap + accessors; `NewList` becomes
  `FromSlice` behind the scenes (producers unchanged call-shape-wise where possible)
- `internal/vm/vm.go` (+~8/−12 LOC) — `OpCons` allocates one cell; `OpMakeList`, `OpGetIndex`
  through accessors
- `internal/vm/builtins.go` (~10 sites) — `_concat_List` copy-left/share-right; `_list_tail`/
  `take`/`drop` cell views (clamping preserved); `_list_get` head-first fast path
- `internal/vm/builtins_list.go`, `builtins_list_poly.go`, `builtins_hof.go`, `builtins_hof_list.go`,
  `builtins_string.go`, `builtins_json.go`, `builtins_xml.go`, `builtins_yaml.go`,
  `builtins_map_bulk.go`, `convert.go` (~2–4 edits each) — migrate the remaining V25 sites
- `internal/vm/list_tail_share_test.go` (instrument swap, same invariant) — V26
- `std/list.ail` (comment-only, ~3 lines) — engine-qualified cost sentence
- `docs/LIMITATIONS.md` + `docs/docs/reference/limitations.md` — rows 40/43 engine-qualified
  (V21); new retention note if Phase 1 selects a chunked arm

**Untouched by design:** `internal/eval/**` (evaluator programme owns), `internal/builtins/list.go`,
`internal/gen/golang/**` (emit-go is a frozen diagnostic projection; its `::` keeps copying — one
line in LIMITATIONS, not a work item), `tools/linters/**`, all queued evaluator programme docs.

## Conflict Surface

This touches `internal/bytecode` (value representation) and `internal/vm` — mandatory. The
"position" being extended is representational, not syntactic: *who observes the concrete shape
of a VM list?*

### What else lives here

| Consumer class | Sites (anchor) | Requirement on the new representation |
|---|---|---|
| **Flat-slice iteration** (`for … := range AsList()`) | ~25 of the 40 V25 sites (HOF builtins, list builtins, show/JSON/XML/YAML encoders, map keys) | Element order and values identical; O(n) stays O(n); goes through `Iter`/`Materialize` — no semantic change, bulk callers already allocate their results |
| **Indexed reads** (`AsList()[i]`) | `_list_get`, `GET_INDEX`, `_list_nth`, `elemAt`-family (builtins_list*.go, vm.go:551) | O(i) — the D-19 : B trade, documented (A9); bounds/error text byte-identical (`builtin_errors_test.go` pins) |
| **Length reads** | `_len`, HOF arities, bounds checks | O(1) via cached `n` (clause (d) of the spike; the flat slice's `len` was O(1) and must not regress) |
| **Sublist views** (`_list_tail`, `take`, `drop`, #1827) | `builtins.go:395-420`, `builtins_list_poly.go` | O(1) tail views stay O(1) (cell share); **clamp semantics byte-identical** (`n<0→0`, `n>len→len`); `take` becomes O(k) cell-copy (was O(k) element-copy) — same order, one caveat: `take` currently *shares* nothing post-swap unless implemented as spine copy; append-isolation invariant holds structurally (cells immutable) |
| **Producers** (`NewList(result)`) | 33 sites (range, map, filter, mapAccumL, sortBy, JSON parse, literals via OpMakeList) | `FromSlice` builds the spine once — O(n) construction, same order of growth as today's single allocation |
| **The bridge** | `internal/vm/convert.go:40-51` (both directions) | Already elementwise-deep-copy (V13) — representation-agnostic; no change beyond accessor calls |
| **Pattern lowering consumers** | `x :: r` → `_list_get(s,0)` + `_list_tail(s,1)` (lower pass, unchanged by design) | Both must be O(1): head is cell field, tail is cell pointer — the #1827 walk win is preserved (AC-5) |
| **Tests pinning flat-representation internals** | `internal/vm/list_tail_share_test.go:20-40` (V26), `bytecode/value_test.go:21-24` (TestValueSize) | Adapted, not deleted: identity pins become tail-cell identity; clamp/cap/append-isolation assertions stay; TestValueSize unaffected (`ListObj` is behind `Obj any`) |
| **`++` semantics** | `builtinConcatList` (V9), `std/list.concat = xs ++ ys` | Same values, new costs: O(len(left)); `[x] ++ rest`-family right-recursion becomes linear — the stdlib builders' shape (parked doc V35 classification, applies to the VM identically) |
| **Evaluator / emit-go** | `internal/eval` (V22 — still copies, untouched); `--emit-go` `::` copy | Documented divergence in LIMITATIONS (already the designed state — costs differ, outputs agree) |
| **Concurrency** | VM is single-goroutine (inherited tail-share V26); `Fork()` lives in the evaluator; bridge copies crossings (V13) | No publication protocol needed for the VM — the LC-4/N21 burden does not transfer; `internal/vm/frame_reuse_test.go`'s 8-concurrent-VMs `-race` test stays the guard (VMs share an image's *code*, never list cells) |
| **Image serialization** | constants are scalars/strings; lists are runtime-only (V14) | No image/gob format change; disassembly output unchanged |

### Disambiguation strategy

There is no parser/typechecker ambiguity — the instruction stream is byte-identical (tail-share
precedent V38). The runtime invariant that replaces "disambiguation": **a `ListObj` is immutable
after construction; `Elems` is deleted so no in-repo consumer can even name the backing
storage** (V12 shows none wrote anyway; the field deletion converts that observation into a
compile-time guarantee — the same move LC-4 makes for the evaluator, at 1/20th the surface).

### Programs that MUST still work

1. The reporter's `rec.ail` repro + the record-accumulator variant (`rec2.ail`, V7) — same
   outputs (`40000`, `…`), linear wall.
2. `examples/first_non_repeat.ail`, `examples/record_cons_pattern.ail`,
   `examples/pattern_matching_adt.ail` — cons construction + cons patterns, both engines,
   byte-identical.
3. `examples/runnable/mapAccumL_running_total.ail`, `examples/runnable/no_loops_fold.ail` — the
   O(1)-step fold family (V27).
4. Every VM HOF dispatch (`internal/vm/builtins_hof_test.go`) and the #1827 walk tests, adapted
   per V26 but with identical observable behavior.
5. The Schroeder backward-integration shape: `foldl` over a float list with a float accumulator
   (the reporter's rewritten form must stay 0.05 s-class; the consing form becomes O(n) too).

### What deliberately changes

- Allocation and time complexity of `::`/`++`/list-tail views on the **VM only** (the point).
- Indexed list access on the VM: O(1) → O(i) (the D-19 : B trade; `ArrayValue` is the documented
  O(1)-indexed alternative).
- `TestValueSize`-adjacent internals: `ListObj` layout (tests adapted, V26).
- Comment/limitation text becomes engine-qualified (V21).
- Retention shape: a small tail no longer pins a big flat array via a *slice view*; it pins it
  via the *cell chain* — same aliasing class as today (V15/#1827 documented it), new mechanism.
- Nothing else. Any output change on any engine is a regression.

## Examples

**Before (measured at HEAD, V2/V7):** the reporter's exact repro on the strict VM —
1.883 s at 20k, 6.618 s at 40k, ×3.5 per doubling; downstream, the Schroeder helper at 72k
samples: 13.8 s, forced onto `mapAccumL` (0.05 s) to ship.

**After (AC targets):** identical program, identical outputs —

```ailang
-- The canonical accumulator idiom, linear on every engine after this doc (VM)
-- plus LC-2…LC-5 (evaluator). No rewrite, no mapAccumL, no coaching.
export pure func plain(n: int) -> string = show(length(foldl(\a x. x :: a, [], range(0, n))))
```

- strict VM at 80k: ≤ ~2× the 40k wall (vs ×3.5–3.9 today), net-of-startup within a small
  constant of the `scalar` fold's 0.35 s (AC-1/AC-2).
- m prepends onto one retained base: O(m) total — the branching case the arena could not buy (AC-3).
- The Schroeder helper can be written the straightforward way again — 72,000 one-cons steps at
  O(1) each.

## Success Criteria (Acceptance)

Every AC names its command and the **measured base on unmodified HEAD** (`b23cadce`, this
container, min-of-3) so each can fail. Wall thresholds are ratios/machine-relative; allocation
thresholds are absolute-with-headroom on the implementer's build.

- [ ] **AC-0** *(gate)* Build a live binary (`make build`); re-derive V2/V3/V7 baselines on it
      before starting; confirm the disassembly shape of V5; record in the sprint log. The
      reporter's numbers (V28) are the same curves on a ~5× faster machine.
- [ ] **AC-1** `rec.ail` `plain` on `--bytecode --strict-bytecode`: wall(80k)/wall(40k) ≤ 2.5
      and net-of-startup cost per element is ≤ 4× the `scalar` fold's. **Base: ×3.51 per doubling;
      cons-fold is ~50× the scalar fold's net cost at 40k (V2) — FAILS today.**
- [ ] **AC-2** Allocation linearity: bytes allocated (pprof `alloc_space`, or
      `testing.AllocsPerRun` at the `OpCons` unit level) for an n-cons build grow ≤ linearly;
      `OpCons` flat alloc < 2 cells × n at n=80k. **Base: Θ(n²) — ≥ 99.9% of consrepro allocation
      was flat in `OpCons` (m-foldl-cons-cost-model Phase 3 profile, inherited; re-derived at AC-0)
      — FAILS today.**
- [ ] **AC-3** Branching (the D-19 objection): m=4,096 prepends onto one retained 16,384-element
      base complete in O(m) cell allocations; measured within 1.5× the linear-chain rate
      (LC-1 clause (a) threshold, applied at the unit level). **Base: Θ(m·len(base)) — FAILS today.**
- [ ] **AC-4** `small ++ big` on the VM: `[x] ++ xs` (len(xs)=80k, m=40k via the stdlib
      right-recursion shape) completes in < 2 s and O(m + len) allocation. **Base:
      `builtinConcatList` copies both operands (V9); the `[x] ++ acc` fold is quadratic — FAILS today.**
- [ ] **AC-5** The #1827 walk stays flat: `bench/vm_tail_walk`'s `walk.ail` at 160k unchanged
      (± noise) versus the pre-swap build; `_list_get(s,0)`/`_list_tail(s,1)` remain O(1).
      **Base: flat since v0.54.10 — this AC exists to fail against the change, not the repo.**
- [ ] **AC-6** Engine parity: every fixture in Conflict Surface §"MUST still work" plus
      `make verify-examples` produce byte-identical output on the evaluator and the strict VM;
      `go test ./internal/bytecode/ ./internal/vm/` green including the adapted V26 pins.
      **Base: green — regression gate.**
- [ ] **AC-7** Phase-1 spike verdict recorded (clause table + arithmetic) in the Implementation
      Report; if STOP, the swap did not happen and the report says so. **Base: no VM-value arm
      exists (V17 harness is eval.Value-only) — FAILS today by absence.**
- [ ] **AC-8** Docs: `std/list.ail` header + `docs/LIMITATIONS.md` rows 40/43 + canonical
      limitations entry are engine-qualified and state the VM trade (`nth` O(i)) honestly;
      CHANGELOG fragment. **Base: V21 — the VM row points at a fix the D-19 programme will not
      deliver to the VM — FAILS today.**
- [ ] **AC-9** Indexed-read regression check: a documented spot check that VM list-index-heavy
      loops that were O(1)-per-access do not silently regress beyond the documented O(i)
      envelope in the benches (walk/scalar controls cover the non-indexing shapes). Gated by
      the spike's clause (b); if clause (b) fails, the spike STOPs before any consumer sees O(i).

## Testing Strategy

**Unit** (`internal/bytecode/list_accessors_test.go`):
- Cons identity: 1,000 conses onto one base share the base's cells (pointer-count, not
  allocation-count — "a copy is one make" per the #1827 convention); empty-list singleton
  stability; `Len` O(1) at 65k (clause (d) shape); clamp parity for tail/take/drop.
- Branching: the AC-3 shape at unit level; mutation attempt (delete-field compile proof makes
  writes unrepresentable — the compile itself is the test).

**VM-level** (`internal/vm/`):
- `OpCons` end-to-end: build by cons, `show`/`length` outputs byte-identical to the evaluator.
- HOF parity: `list_hof_parity_test.go` extended with consing-fold output parity at n ∈ {0,1,2,65k}.
- #1827 tests adapted (V26): identity instruments swap from slice-element pointers to cell
  pointers; clamp/append-isolation assertions unchanged.
- `-race`: the existing 8-concurrent-VMs test (frame_reuse_test.go) — cells are VM-internal and
  never shared across VMs (V13); pin that a constant pool *cannot* contain a list (V14) so no
  image-level sharing exists.

**Spike** (`tools/internal/spike-listrep/vmarm/`): the five clauses on `bytecode.Value`,
five-trial medians, tie/spread rerun rule inherited verbatim from the LC-1 protocol.

**Bench** (`bench/vm_cons_build/`): checked outputs per (program, n, engine) — `run.sh` FAILs on
mismatch or nonzero exit; min-of-N; before/after tables in the Implementation Report; both
engines; the walk and scalar programs ride along as regression controls (AC-5/AC-9).

**Manual**: the reporter's repro + Schroeder-shape program at 72k on the implementer's machine,
before/after, recorded in the report.

## Non-Goals

- **The evaluator's representation** — owned by the queued LC-2…LC-5 programme (D-19 : B, D-22 :
  C1). `eval.ListValue` and `listConsImpl` are untouched; the interpreter stays quadratic until
  that programme lands, and its docs say so.
- **Re-opening D-19 or D-22** — the direction and (for the evaluator) the candidate are ruled;
  the VM spike re-measures the *ranking* on the VM's value width within the ruled direction, with
  a STOP that reports rather than re-litigates.
- **`ArrayValue`/`TupleValue`** — untouched (the decomposition's N5 applies to the VM's
  `ArrayObj` equally).
- **`--emit-go` output** — frozen diagnostic projection; its `::` copy gets one LIMITATIONS
  sentence, not a work item.
- **New stdlib surface** — no new builtins, no `mapAccumL`-class additions; the mitigation
  family is complete; this doc fixes the substrate it mitigates.
- **Interpreter/VM cost parity beyond the shape** — per-op costs may differ (already do);
  parity is outputs (AC-6), not seconds.

## Timeline

**Day 1**: Phase 1 spike (arms + clause matrix + verdict) — kill-criterion arithmetic before any
migration. **Days 2–3**: Phase 2 accessor seam + the 40-site migration (behavior-identical
commits). **Day 4**: Phase 3 swap + field deletion + OpCons/concat/tail rewrite + unit tests.
**Day 5**: Phase 4 benches + docs; Phase 5 regression surface; Implementation Report.
**Total: 4–5 days** — sized against the #1827 landing (~2 d for 3 one-line views + tests) and
against the LC-2/LC-4 evaluator estimates (3–4 d each at 20× the consumer surface), ×2 honest
on measurement overhead, which has dominated every sibling sprint.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Phase 1's clause (c) fails on 40 B values (C1's per-element memory exceeds 2.5× C0) | Med (scope) | The spike STOPs before migration; C2K32 (1.07× on 16 B values, chunk amortization amortizes the head width) is the alternative arm in the same matrix — a bounded re-rank, not a re-opening; the verdict + arithmetic go to the coordinator |
| Indexed-read regression on VM loops that index lists in hot paths (O(1)→O(i)) | Med | Clause (b) gates it before the swap; `ArrayValue` is the documented alternative; `nth` clause (b) measured ≤ 1.1× C0 on 16 B values — re-measured, not assumed |
| A consumer is missed in migration | Low | `Elems` deletion is the compile-time proof (LC-4's move); the 40-site inventory (V25) is the review checklist |
| #1827 semantics drift (clamp/append-isolation) | Med | V26 tests adapted with the same assertions; bench/vm_tail_walk rides along in the bench suite (AC-5) |
| Hidden flat-slice assumption in a consumer (e.g. `unsafe`, pointer tricks) | Low | `grep -rn "unsafe" internal/vm internal/bytecode` at AC-0; V12's audit found none; the field deletion would fail loudly at compile time anyway |
| Per-cell GC pressure makes small-list HOF workloads slower in absolute terms | Low | HOF producers go through `FromSlice` (one spine build per result, same allocation count class as today); clause (b) on iteration guards it; bench scalar/walk controls |
| Divergence between engines' docs claims while only one is fixed | Low | Phase 4's engine-qualified sentences are the designed state (costs differ, outputs agree) |
| Scope creep into the evaluator programme | High (process) | Design Freeze scope fence; any PR touching `internal/eval` under this doc's name is out of scope by definition |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantic change; outputs byte-identical on both engines (AC-6); sharing is unobservable (V12/V13/V14) |
| A2: Replayability | 0 | Instruction streams unchanged (V5; tail-share V38 precedent); traces unchanged |
| A3: Effect Legibility | 0 | Pure construction path; no effects touched |
| A4: Explicit Authority | 0 | No capabilities touched |
| A5: Bounded Verification | +1 | The `Elems` deletion is a mechanical migration proof; the spike's ratified clause thresholds bound the representation choice; allocation-count unit tests pin O(1) cons forever |
| A6: Safe Concurrency | 0 | VM is single-goroutine; bridge deep-copies crossings (V13); no new sharing crosses a goroutine boundary |
| A7: Machines First | +1 | The third consumer report of the same idiom; the language stops punishing the canonical accumulator on the engine AI test suites actually run (`--strict-bytecode`) |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | `::` costs what its type and every FP tradition imply; the new costs (`nth` O(i), `++` copy-left) are documented, not hidden; LIMITATIONS rows become engine-true instead of half-true (V21) |
| A10: Composability | 0 | No API changes beyond internal accessors |
| A11: Structured Failure | 0 | Error codes/text byte-identical; no new failure modes |
| A12: System Boundary | 0 | Bridge behavior unchanged (elementwise copy both ways) |

**Net Score: +3** → **Decision: Proceed**

### Hard Violation Check

- [x] A1 (Determinism): no nondeterminism — cell sharing is invisible to every observable
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): removes a hidden quadratic from the canonical machine-written idiom

## Duplicate / Coverage Gate (explicit, per skill §4)

The create script's auto-search found no doc index in this container; the manual gate, from
reading the top candidates in full:

- [m-list-cons-quadratic](../m-list-cons-quadratic.md) (planned, **PARKED — superseded by D-19 : B**;
  quorum-blocked 2026-08-19; retained as evidence appendix). **Evaluator only** — its Non-Goals
  freeze VM `OpCons` at O(n) on the premise "`ailang run` doesn't use the VM" (V19/V20). Not
  duplicative: this doc covers the excluded engine under the ruling that superseded it.
- [m-list-cons-cells-decomposition](../m-list-cons-cells-decomposition.md) (planned, quorum-cleared
  programme roadmap). **Evaluator only** — N11 and LC-4 both list the VM as a non-goal. This doc
  is the missing VM piece of that programme's own invariant (INV-1), executed as an independent
  sprint because the bridge decouples the engines; it does not modify the roadmap or re-quorum it.
- [m-list-repr-spike](../m-list-repr-spike.md) + [M6 report](../../implemented/v0_34_0/m-list-repr-spike-M6-report.md)
  (LC-1, **implemented v0.34.0, GO**). Evaluator-value spike; its harness is *reused* (Phase 1),
  its clauses re-ratified on `bytecode.Value`, its D-22 (C1) outcome the presumptive VM arm.
- [m-list-accessor-api](../m-list-accessor-api.md) (planned, LC-2, quorum-cleared). Evaluator
  accessor layer; its API draft is *input* to Phase 2 (deferred decision), not duplicated — it is
  designed for `eval.Value` and still unexecuted.
- [m-foldl-cons-cost-model](../../implemented/v0_52_0/m-foldl-cons-cost-model.md) (implemented
  v0.52). Shipped the docs-honesty fix, `mapAccumL`, and the VM per-callback (H1/H2) work; its
  Phase 3 explicitly routed the `OpCons`-bytes residue to "the representation decision" — which
  excluded the VM. This doc is where that residue actually gets fixed, on the engine it measured.
- [m-vm-list-tail-share](../../planned/v0_53_0/m-vm-list-tail-share.md) (planned; Phase 1 landed
  v0.54.10 as #1827; residual landed per V6). The **consumption** half (pattern-tail walks) on
  the VM; its Non-Goals name `::`/`++` construction as "parked doc's territory". Complementary:
  together the two docs make the whole cons idiom linear on the VM.
- [m-benchmark-performance-refresh](../../implemented/v0_54_18/m-benchmark-performance-refresh.md)
  (implemented v0.54.18, the reporter's own release) — a docs-website benchmark-page redesign; read
  and confirmed it touches no list representation (no cons/OpCons content, grep V-row candidate),
  listed here only to record that the v0.54_18 release context was checked for prior coverage.

**Gate outcome:** the VM construction half is genuinely unowned (V19) — proceed. The interpreter
half is NOT re-planned here (Routing Lane table).

## Related Documents

- [m-list-cons-cells-decomposition](../m-list-cons-cells-decomposition.md) — the evaluator
  programme this doc runs alongside: same invariant (INV-1), same ruled direction (D-19 : B,
  D-22 : C1), disjoint seam (`eval.ListValue` vs `bytecode.ListObj`).
- [m-list-repr-spike M6 report](../../implemented/v0_34_0/m-list-repr-spike-M6-report.md) — the
  spike protocol, clause thresholds and harness this doc's Phase 1 reuses.
- [m-vm-list-tail-share](../../planned/v0_53_0/m-vm-list-tail-share.md) — the consumption half;
  #1827's clamp/append-isolation semantics are this doc's preconditions (V15/V26).
- [m-foldl-cons-cost-model](../../implemented/v0_52_0/m-foldl-cons-cost-model.md) — the honest
  docs and shipped mitigations; its Phase 3 profile is the `OpCons`-bytes attribution (AC-2's base).
- [#676](https://github.com/sunholo-data/ailang/issues/676) — the original construction-half
  report (evaluator); this task's report is its third independent descendant.

## References

- **Source report**: coordinator task `task-50836634`, AILANG v0.54.18 (`cd33d05`, binary md5
  `fd40f17194318d2650e53dac2eac84ba`), reported by `stapledons_godot` via `ailang messages`,
  found executing stapledons-godot **R3-SUNHOLO-AUDIO-REALISM** (sunholo/audio 0.6 tests,
  Schroeder backward-integration helper, 72,000 samples).
- [design_docs/PROGRAM.md](../../PROGRAM.md) §4 — routing lanes.
- [Design Axioms](/docs/references/axioms).

## Future Work

- Evaluator-side execution of LC-2…LC-5 — at that point the engine-qualified doc sentences
  collapse back to one unconditional O(1)-cons statement (Phase 4 rows are written to be
  deletable, not reworded).
- If both engines end up on cons cells, a possible bridge fast-path (spine sharing across the
  engine boundary instead of elementwise copy) — measured first, speculative now.
- `mapAccumLE` (effectful variant) if a consumer asks — unrelated to this defect.

---

**Document created**: 2026-11-03 (design-doc-creator, unattended coordinator session `task-50836634`;
live v0.57.0 binary of this exact tree available for measurements; no Go toolchain — W-1/V24
apply; `ailang check` gate run on every AILANG snippet, V4/V7)

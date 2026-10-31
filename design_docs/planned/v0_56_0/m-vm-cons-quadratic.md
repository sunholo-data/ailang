# M-VM-CONS-QUADRATIC: O(1) `::` on the strict bytecode VM — the D-19 programme's missing half

**Status**: Planned
**Target**: v0.56.0
**Priority**: P0 — same defect class and same P0 as the queued D-19 programme, but on the engine
the reporting consumers actually run (`--bytecode` / `--strict-bytecode`)
**Estimated**: ~3 days (M1 seam 1 d · M2 swap 1.5 d · M3 docs/bench 0.5 d), ×2 where honest → ~6 d
**Dependencies**: None hard. **Coordination required**: [m-list-cons-cells-decomposition](../../m-list-cons-cells-decomposition.md)
(the D-19/LC programme — this doc is its VM-side piece; see [Triage](#triage--the-duplicate-gate-verdict)),
[m-persistent-collections](../v0_55_0/m-persistent-collections.md) Phases 2–3 (shared files, see Conflict Surface),
[m-vm-list-tail-share](../v0_53_0/m-vm-list-tail-share.md) (#1827 fixtures must not regress).
**Planner-Lane**: opus-required (runtime value representation in `internal/bytecode` + `internal/vm`)

**Bug report**: AILANG v0.54.18 (`cd33d05`, darwin arm64/M4 Max, binary md5 `fd40f171…`):
`foldl(\a x. x :: a, [], range(0, n))` grows ×3.9 per doubling on `--bytecode --strict-bytecode`
(40,000 → 1.26 s; 80,000 → 4.91 s, user 8.1 s), same shape on the interpreter (20k 0.27 s → 40k 0.84 s),
record/tuple accumulator holding the list identical (`{s | out: x :: s.out}`: 1.24 s / 5.06 s); a scalar
accumulator stays linear (0.02 s at both sizes). Consumer: sunholo/audio 0.6 tests — a Schroeder
backward-integration helper over 72,000 samples took 13.8 s with foldl-cons vs 0.05 s with map/mapAccumL.
Found executing stapledons-godot R3-SUNHOLO-AUDIO-REALISM. The reporter already avoids foldl-cons
accumulators downstream ("and are not working around it otherwise").

**Author**: design-doc-creator, unattended coordinator session (`task-806bedd8`). This container has a
live `ailang` binary (v0.55.0, commit `e863a1e`, `/usr/local/bin/ailang`) but **no Go toolchain / no
`make`** — so every wall-clock number below was re-derived first-party by running that binary, every
code-structure claim is a read at this tree (`490fb730`, clean, no uncommitted work), and no claim
about post-fix behavior is asserted as measured. The repro type-checks (`ailang check` → `✓ No errors
found!`, V-R). Provenance marks: **[R]** read first-party, **[P]** first-party run against the v0.55.0
binary in this container, **[N]** negative-existence claim with its grep.

## Routing Lane (per design_docs/PROGRAM.md §4)

**Lane: AILANG fix.** Same routing as the D-19 programme itself
([m-list-cons-cells-decomposition](../../m-list-cons-cells-decomposition.md) argues it in full): no
motoko extension — prompt, tool, or retrieval shaping — can change what the VM's `OpCons` allocates,
and neither reporting consumer (stapledons-godot, sunholo) is behind motoko. Not the core-floor lane:
no motoko-core crash/overflow, no re-freeze. It carries runtime-core discipline *internally* because
it touches a value representation (`bytecode.ListObj`) and the VM dispatch — hence the mandatory
[Conflict Surface](#conflict-surface) below.

## Triage — the duplicate-gate verdict

The skill's coverage gate requires this before designing anything. The report's ask — "foldl that
conses onto a list accumulator should be O(n)" — splits into two halves with different verdicts:

| Half | Verdict | Where |
|---|---|---|
| **Interpreter** (`listConsImpl` copies the tail) | **Already covered — do not re-design here.** The D-19 programme owns it: quorum-cleared decomposition ([m-list-cons-cells-decomposition](../../m-list-cons-cells-decomposition.md), iteration 234), representation ruled by Mark (D-22 = C1 plain cons cells), spike run and GO ([m-list-repr-spike-M6-report](../v0_34_0/m-list-repr-spike-M6-report.md), 420 trials), LC-2 accessor design + sprint plan written (4.5 d accepted, D-24). Parked behind the D-28 bar-first ordering, P0 unchanged. | INV-1, LC-2…LC-5 |
| **Shipped mitigation** (stateful list-building without cons) | **Already shipped — verified live** (V10, V-P4): `mapAccumL` over 80k elements = 0.37 s on the strict VM vs 52.11 s for the consing foldl in the same container. `std/list.ail`'s header teaches the honest cost model and the linear path (V10). The reporter uses exactly this (0.05 s on their machine). | v0.52.0, [m-foldl-cons-cost-model](../v0_52_0/m-foldl-cons-cost-model.md) |
| **Strict bytecode VM** (`OpCons` copies the tail) | **NOT covered anywhere — this doc owns it.** | this doc |

Evidence for the third row (the load-bearing one):

- **[R]** `m-list-cons-quadratic.md:643` (Non-Goals): "**VM `OpCons`** — stays O(n); `ailang run`
  doesn't use the VM; parity deferred with the divergence documented." The deferred-parity rationale
  is **stale**: `ailang run --strict-bytecode` fails instead of falling back and implies `--bytecode`
  (live `ailang run --help`, V8), #1827's residual Phase 2 made the strict flag select the VM on
  `run`, and both consumer reports in this defect class (stapledons-godot M1.2b, this one) run the VM.
- **[R]** The LC programme inherits the exclusion: LC-4 "Does NOT: touch the VM (`OpCons` stays O(n),
  documented divergence, N11)"; N11: "the VM's own `bytecode.Value` repr is out of scope". So when
  the queued programme completes, the interpreter's cons is O(1) but **this report's headline command
  still reproduces, quadratic**, on `--strict-bytecode`.
- **[N]** No queued row names the VM half: `grep -n "OpCons\|VM.*cons" design_docs/v1-mission.md
  design_docs/planned/ailang-core-backlog.md` → no hits (V12). The skill's dual doc search
  (SimHash + neural via `create_planned_doc.sh`) returned "(none found)" both sides this session
  (query "vm cons quadratic"; the index appears empty in this container), so the coverage verdict
  rests on the manual read of all five neighbouring docs above.
- **[R]** The gap is not denial: `bench/vm_tail_walk/build.ail` — the #1827 bench — already carries
  the *still-quadratic* cons-construction shape as its named "D-19 construction contrast"
  (README, V11), i.e. the programme's own bench documents that VM cons is unfixed.

**This doc is therefore not a fork of the D-19 programme.** It implements the same invariant (INV-1,
below) on the other engine, under the same human-ruled representation (D-22 = C1, mirrored), and
inherits the programme's queue position under D-28 (it does **not** jump the bar — see High-Impact
Decisions). The interpreter half stays with LC-4; nothing here re-opens it.

## Problem Statement

The canonical functional accumulator idiom — `foldl(\acc x. x :: acc, [], xs)` — costs **Θ(n²) time
and allocation on the strict bytecode VM**, because `bytecode.ListObj` is a flat slice
(`internal/bytecode/value.go:101`) and the `OpCons` opcode copies the entire tail into a fresh slice
on every cons (`internal/vm/vm.go:511-521`, V1):

```go
case bytecode.OpCons:
    head := frame.Regs[inst.B()]
    tail := frame.Regs[inst.C()]
    ...
    old := tail.AsList()
    elems := make([]bytecode.Value, 0, len(old)+1)   // ← O(len(tail)) per cons
    elems = append(elems, head)
    elems = append(elems, old...)
```

Every cons is O(n); a fold that conses n times is Θ(n²/2) `Value` copies. At 80k elements that is
≈3.2·10⁹ copied slots × ~40 B/slot — GC-churn-dominated, matching the reporter's "user 8.1 s vs wall 4.91 s" split, and the VM's own struct comment
("a Value is 40 bytes… this width is what the VM pays per element on copy-heavy shapes",
`value.go:78-82`). A scalar accumulator copies O(1) per step and stays linear, which
is exactly the report's control. The interpreter is the same class (`listConsImpl`,
`internal/builtins/list.go:98-103`, V2) — owned by LC-4.

**Current state — all numbers first-party in this container** (v0.55.0 binary `e863a1e`; this
container is ~10× slower than the reporter's M4 Max, so absolute walls do not transfer — the
per-doubling **ratios** do, and they are what the defect is):

Repro (type-checks clean, V-R; verbatim from the report, plus the record and mapAccumL variants):

```ailang
module rec
import std/list (foldl, range, length, mapAccumL)

export pure func plain(n: int) -> string = show(length(foldl(\a x. x :: a, [], range(0, n))))
export pure func scalar(n: int) -> string = show(foldl(\a x. a + intToFloat(x), 0.0, range(0, n)))
export pure func recacc(n: int) -> string =
  show(length(foldl(\s x. {s | out: x :: s.out}, {out: []}, range(0, n)).out))
export pure func accum(n: int) -> string =
  show(match mapAccumL(\s x. (x, s), 0, range(0, n)) { (xs, _) => length(xs) })
```

| Run ([P], this container) | wall | per-doubling |
|---|---:|---:|
| VM strict, `plain` N=40,000 | 12.58 s | — |
| VM strict, `plain` N=80,000 | 52.11 s | **×4.14 → quadratic** |
| VM strict, `scalar` N=80,000 | 0.44 s | linear (control) |
| VM strict, `recacc` N=40,000 | 12.15 s | same class as `plain` at 40k |
| VM strict, `accum` (mapAccumL) N=80,000 | 0.37 s | **linear — the shipped mitigation works** |
| interpreter, `plain` N=20,000 | 1.39 s | — |
| interpreter, `plain` N=40,000 | 4.30 s | ×3.09 |
| interpreter, `plain` N=80,000 | 19.11 s | ×4.45 |

Reporter's numbers ([report], M4 Max, v0.54.18): VM 40k 1.26 s → 80k 4.91 s (×3.9); scalar 0.02 s
at both; record variant 1.24 s / 5.06 s; interpreter 20k 0.27 s → 40k 0.84 s. Same ratios, faster
machine. Consistent.

**Impact.** Consumers on `--bytecode`/`--strict-bytecode` (both bug reports in this class) pay the
quadratic on their primary engine. `std/list`'s header already teaches the workaround
(`mapAccumL`, `map`/`filter`), and the reporter follows it — but a language whose *fundamental list
construction operation* (`::`, documented in `std/list` as exactly that) is quadratic on its strict
engine teaches every future consumer a trap the docs must constantly patch over
(`docs/LIMITATIONS.md` rows 40 and 43 exist solely to carry this caveat, V10).

**This is systemic, not a one-off** (audit per CLAUDE.md principle 3): the defect class is "list
construction cost on a flat-slice representation", it appears at two independent sites (`OpCons` in
the VM, `listConsImpl` in the interpreter), and the already-queued LC programme measured the same
class on the interpreter with a full decomposition. The unified fix is one representation decision —
cons cells with structural sharing — applied to the remaining engine. This doc does not patch
around `OpCons`; it removes the copy.

## Verification Log

| # | Claim | Command / location | Observed |
|---|---|---|---|
| V1 [R] | VM `OpCons` copies the whole tail | `sed -n '511,521p' internal/vm/vm.go` | `old := tail.AsList(); elems := make(…, 0, len(old)+1); append(elems, head); append(elems, old…)` — O(n) per cons |
| V2 [R] | Interpreter `::` copies the whole tail (LC-4's half) | `sed -n '87,107p' internal/builtins/list.go` | `listConsImpl`: `make([]eval.Value, 0, 1+len(tail.Elements))` + `append(result, tail.Elements…)` |
| V3 [R] | On the VM, `::` reaches **only** `OpCons` — no builtin path | `internal/bytecode/builtin_adapt.go:129-138` (`IsCallableBuiltin` — "Symbolic entries (e.g. `"::"`) lower to dedicated opcodes"); `internal/bytecode/compiler/collections.go:212` (single `OpCons` emit) | one emit site, one dispatch arm; `grep -n "__list_cons" internal/bytecode/builtin_names.go` → **no hits** (exit 1) — no cons builtin exists |
| V4 [R] | `ListObj` is slice-backed, and the representation question was left open **with a revisit clause** | `sed -n '98,103p' internal/bytecode/value.go` | `ListObj{Elems []Value}`; comment: "The design doc (§3.2) leaves the choice of cons-cell vs slice open; Phase 2B uses slices… **Revisit in Phase 2D if benchmarks demand**" — the benchmarks now demand (V-P) |
| V5 [R] | The migration surface is the `NewList`/`AsList` seam + exactly one raw poke | `grep -rn 'ListObj{' internal/ cmd/ --include=*.go \| grep -v internal/bytecode/` → empty; `grep -rn 'ListObj).Elems' internal/vm/` → **one** hit | `ListObj` constructed only inside `internal/bytecode`; the single raw field access outside it is `internal/vm/vm_arith.go:80` (list equality). All other VM consumers go through `AsList()` (41 non-test call sites across 12 files) and `NewList` (31 non-test sites) |
| V6 [R] | D-19 programme state: decomposition quorum-cleared, representation human-ruled, spike GO | `head -40 design_docs/planned/m-list-cons-cells-decomposition.md`; `design_docs/implemented/v0_34_0/m-list-repr-spike-M6-report.md`; v1-mission D-22 row | LC-0…LC-5 planned; D-22 = **C1** plain cons cells `{head, tail *cell, n}` (chunked `C2K32` declined); spike verdict GO, 420 trials, `tools/internal/spike-listrep/testdata/m6-matrix.json` exists |
| V7 [R] | LC-4 explicitly excludes the VM | decomposition LC-4 scope: "Does NOT: touch the VM (`OpCons` stays O(n), documented divergence, N11)" | confirmed — no LC piece owns `OpCons` |
| V8 [R] | The stale deferral rationale: "`ailang run` doesn't use the VM" | `design_docs/planned/m-list-cons-quadratic.md:643`; `ailang run --help` (live) | Non-Goal as quoted; help says `--strict-bytecode` "Fail instead of falling back to the evaluator; **implies --bytecode**" — `run` does use the VM, and both consumer reports run it |
| V9 [N] | No queued row owns the VM cons half | `grep -n "OpCons\|VM.*cons\|cons.*VM" design_docs/v1-mission.md design_docs/planned/ailang-core-backlog.md` | no hits (only `consider`-word false positives) |
| V10 [R] | Mitigation shipped + stdlib cost model honest | `std/list.ail:1-16` (header), `:62-64` (foldl comment), `:70-76` (mapAccumL); `changelogs/v0.32-current.md` v0.52.0 | header states the O(n²) class and the linear paths; `mapAccumL` present; `docs/LIMITATIONS.md` rows 40 & 43 carry the caveat and name D-19 |
| V11 [R] | #1827 fixtures exist and already bench the *unfixed* cons-build shape | `bench/vm_tail_walk/` (README, `walk.ail`, `foldl.ail`, `build.ail`, `run.sh`); `internal/vm/list_tail_share_test.go` | README: "Build … prints length `n`: **it is the still-quadratic D-19 construction contrast**"; tests `TestListTailDoesNotCopy` / `TestListTailClampsAndCapsCapacity` / `TestListTailViewIsolatedFromAppend` — the regression suite this piece must not break |
| V12 [R] | D-28 bar-first ordering still governs the cons-cells programme | v1-mission D-28 row | "items serving an open bar clause … OUTRANK the cons-cells programme until all five clauses close" — this piece inherits that queue position, it does not re-order it |
| V13 [R] | VM concurrency shape: single-goroutine per VM; images shared across goroutines | `internal/vm/frame.go:46` ("per-VM; a VM is single-goroutine"); `internal/vm/vm.go:84` ("an image is shared across goroutines") | cells never cross goroutines through the VM today; image **constants** are materialized at build (constructor-only), so no cell is ever image-shared |
| V14 [R] | The eval↔VM bridge deep-copies per element both ways (isolation boundary) | `internal/vm/convert.go:38-48` (TagList → per-element `bytecodeToEval`), `:114-196` (`EvalToBytecode` per element) | bridge unchanged by this piece; the N11 divergence stands until a follow-up shares cells across it (Future Work) |
| V15 [R] | No image-format change is needed | `internal/bytecode/opcode.go:79` (`OpCons` ABC encoding unchanged); list **constants** are `NewList`-materialized | same opcodes, same encodings, same constant forms → images byte-compatible, `disasm` unchanged |
| V16 [R] | Regression fixtures exist (Conflict Surface §"Programs that MUST still work") | `ls` of `examples/runnable/`, `tests/stdlib/` | `ai_offline_cli.ail`, `no_loops_fold.ail`, `no_loops_filter_map.ail`, `record_update.ail`, `letrec_local_shadow.ail`, `effectful_list_t3_foldlE_acc.ail`, `tests/stdlib/list_iterative_test.ail` — all present |
| V-P1 [P] | The defect reproduces first-party, strict VM, quadratic | `ailang run --quiet --strict-bytecode --entry plain --args-json N rec.ail` (N=40k, 80k) | 12.58 s → 52.11 s = **×4.14 per doubling** |
| V-P2 [P] | Scalar control linear on the same engine/size | `--entry scalar --args-json 80000 --strict-bytecode` | 0.44 s |
| V-P3 [P] | Record-accumulator variant same class (the cons inside the record is the cost) | `--entry recacc --args-json 40000 --strict-bytecode` | 12.15 s ≈ `plain` at 40k (12.58 s) |
| V-P4 [P] | Shipped mitigation linear on the same engine/size | `--entry accum --args-json 80000 --strict-bytecode` | 0.37 s |
| V-P5 [P] | Interpreter superlinear too (LC-4's half, confirmed here for completeness) | `--entry plain`, no VM flags, N=20k/40k/80k | 1.39 / 4.30 / 19.11 s = ×3.09, ×4.45 |
| V-R [R] | The repro is valid AILANG at HEAD | `ailang check rec.ail` (prompt v0.16.8 idioms) | `✓ No errors found!` |

## Goals

**Primary Goal:** `x :: xs` is O(1) worst-case in time and allocation on the strict bytecode VM —
INV-1's VM clause — with **byte-identical observable behavior** (outputs, errors, disassembly, image
format), and no change to the interpreter-side programme.

**Success Metrics:**
1. The report's headline command (`--entry plain --strict-bytecode`) at N=80,000 drops from
   **52.11 s** (this container, V-P1) to **< 1 s**, and its per-doubling ratio from ×4.14 to ≤ ~2.2×
   (start-up/compile constant included).
2. `bench/vm_tail_walk/build.ail` (the programme's own cons-build contrast) at its 40k bound drops
   to the `walk.ail`/`foldl.ail` order of magnitude, with a ≤ ~2.2× per-doubling ratio recorded by
   `run.sh`; the checked oracle (`n`) and engine equality still hold.
3. **INV-1 under branching, on the VM**: m prepends onto one retained base cost O(m), not
   O(m·len(base)) — the exact shape that killed the arena (D-19), converted into a regression gate
   (m6 spike clause, `branching_bench_test.go` analog).
4. No regression on the linear controls: `scalar` and `accum` (mapAccumL) at 80k stay within ~1.5×
   of V-P2/V-P4; `#1827` walk/tail fixtures (`list_tail_share_test.go`, `bench/vm_tail_walk/walk.ail`)
   unchanged in wall and byte-identical in output.
5. Per-element retained memory of a cons-built list on the VM within **~2.5×** of the slice form
   (m6 clause (c) inherited; expected better — a `bytecode.Value` is ~40 B in *both* forms, `value.go:78-82`),
   and iteration through a cons-built list within **~2×** of the slice control (m6 clause (b)).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| **Representation = D-22's C1 (plain cons cells, cached length), mirrored in `bytecode.ListObj`** — not re-derived, not chunked | Two engines, two representations for the same invariant doubles every future seam (bridge, docs, tests); D-22 already ruled C1 for the eval side and declined `C2K32` | **human (ruled — D-22, #745 2026-08-22)** | design (inherited) | high (avoided: not re-litigated here) |
| **Scope = the VM half only.** The interpreter half stays with LC-4; this doc does not touch `eval.ListValue`, and does not re-open the arena (declined by D-19) | Re-scoping mid-flight forks the decision record a second time (the arena/cons-cells fork was D-19 itself) | human (D-19 scope) + this doc | design | high (avoided) |
| **The piece inherits the D-28 queue position** — it does not jump the bar-first ordering, and does not use the live consumer report to re-prioritise the programme | A doc that shipped a consumer-urgency argument could be read as re-ordering a human-ruled queue | human (D-28) | design | low |
| **No new opcode, no image-format change**: `OpCons` keeps its encoding and semantics; only its implementation cost changes | Image/disasm/constant compatibility is load-bearing for the strict VM's parity guarantees (V15) | compiler (semantics) + this doc | design | med |
| **Cell reads walk, they do not materialize**: `OpGetIndex`, `_list_nth`, `_list_tail`/`take`/`drop` on cell chains walk cells without allocating | A per-index materialization is O(n) and re-creates the quadratic on indexed access; `#1827`'s O(1) tail binding must not regress (V11) | this doc | design | med |
| **`AsList()` materializes on demand (walk + build), memoization deferred** (see Deferred Decisions) | Memoization writes into a shared immutable object — a memory-model property that needs its own race-test obligation (N21 analog) if ever shipped | agent | design | med |

### Design Freeze

- [x] Scope: `internal/bytecode` (`ListObj`) + `internal/vm` (dispatch arms, list builtins,
      equality). No eval-side, no bridge, no stdlib, no new opcodes.
- [x] Representation: C1 mirror — cons form `{Head *Value, Tail *ListObj, N int}` alongside the
      materialized `Elems` store (ArrayObj dual-store precedent); `N` cached at construction.
- [x] Observable contract: outputs, error strings, disassembly, image bytes — all identical;
      only cost changes. Anything that differs is a regression, not an intentional change.
- [x] INV-1 (VM clause) is the acceptance invariant, including the branching shape (Goal 3).
- [x] Queue position: with the D-19 programme under D-28; sequenced against LC-3c/LC-4 and
      persistent-collections Phase 2 by the first-lands-first rule (Conflict Surface).
- [ ] AsList memoization: **deferred** (default = do not memoize in M2; ship only with a measured
      hot loop **and** the N21-analog race obligation).

## Deferred Decisions

- **`AsList()` memoization** — agent may choose at M2 time; default is *no memoization* (walk +
  build per call, no write after construction). Rationale: the foldl-cons pattern pays one
  materialization per consumer (`length`/`show`), so a memo buys little; a memo is a
  nil→non-nil transition on a possibly-shared object and would need the N21-style
  publication-path→happens-before→race-test table. Only ship it with a measured hot loop.
- Accessor/constructor **naming and file layout** inside `internal/bytecode` — agent may choose
  (`NewCons`, `Len`, `IsCell`, `Walk` are the working names here).
- Whether `__list_nth`'s cell path caches the walked prefix — agent; parity with LC-5's
  disposition when that piece is picked.
- Bench size extension for `build.ail` (40k bound can rise once linear; keep checked oracles) — agent.
- `docs/LIMITATIONS.md` row 43 wording — agent, but it **must** stop claiming both engines are
  unfixed once this lands (say which engine is fixed and which still awaits LC-4).

## Solution Design

### Overview

Mirror the eval-side strangler — but at VM scale, where it is one sprint, not a programme. The
discovery that makes this cheap (V5): the VM's entire list surface already sits behind a
two-function seam (`bytecode.NewList` / `Value.AsList`), with exactly **one** raw field access
outside `internal/bytecode` (`vm_arith.go:80`, list equality). So there is no 900-site migration to
decompose: M1 formalizes the seam as package-private accessors, M2 swaps the representation behind
it, and `OpCons` — the single construction opcode — becomes a cell allocation.

The invariant bought (INV-1's VM clause): `x :: xs` is O(1) worst-case for every live list `xs`,
regardless of sharing (m-fold onto a retained base included), and N lists sharing a spine retain
O(total distinct cells); pattern-match tail extraction (`_list_tail(s, 1)` — the `#1827` surface)
stays O(1) because it returns the tail **cell**, not a slice offset.

### Architecture

**Components:**

1. **`internal/bytecode` — the cell (new, ~120 LOC + tests).**
   `ListObj` becomes a dual-store value, following the `ArrayObj` precedent three fields above it
   (`value.go:109-113`, "two stores and exactly one is used… Neither slice is ever mutated after
   construction"):

   ```go
   type ListObj struct {
       Elems []Value  // materialized form (the only form today); never mutated after construction
       Head  *Value   // cons form: Head/Tail/N live, Elems nil. Head is a pointer so a
       Tail  *ListObj // zero Value (unit) is still a valid head.
       N     int      // cached length, set at construction.
   }
   ```

   - `NewList(elems)` — unchanged (materialized; constants, `range`, literals).
   - `NewCons(head Value, tail *ListObj) Value` — O(1): one `ListObj`, `N = tail.Len() + 1`.
   - `(*ListObj).Len() int` — `N` for cells, `len(Elems)` otherwise; O(1) both.
   - `(*ListObj).IsCell() bool`; `(*ListObj).Walk(yield func(int, Value) bool)` — iterate cells,
     allocation-free, shared-prefix safe.
   - `AsList()` — materialized: today's O(1) return; cell: walk once, build, return. *No
     memoization* under the deferred default (write-free).
   - `Value.Equal` TagList branch (`value.go:422`) and `Value.String` TagList branch (`:520`):
     unchanged logic over `AsList()`/`Len()`.
   - A `listrep`-style invariant guard is **not** needed at VM scope (the eval-side 900-site
     risk does not exist here), but `go vet`-visible construction via the two constructors is: the
     fields stay unexported to `internal/vm` (same package boundary as today — `vm` already cannot
     name `Elems` except through the one type assertion, which M2 removes, V5).

2. **`internal/vm` — the swap (~60 LOC across 6 files).**
   - `vm.go:511` `OpCons`: `frame.Regs[inst.A()] = bytecode.NewCons(head, tailListObj)`; the
     "CONS: tail is %s, not List" error and its `errAt` position byte-identical.
   - `vm.go` `OpGetIndex` TagList arm: walk `i` cells (no allocation, no materialization);
     materialized lists keep today's `elems[i]`.
   - `vm_arith.go:80` runtimeEq TagList arm: migrate the raw `.Elems` poke to
     `Len()`+`AsList()` (or a `Walk`-based compare for cells).
   - `vm_arith.go` `OpConcat` TagList arm: left materialized/copied O(len(left)), right **shared**
     as `Tail` — parity with LC-4's concat semantics ("O(len(left)) sharing the right operand").
   - `builtins.go` `_len` TagList arm: `Len()` (O(1) both forms; today it is `len(AsList())`).
   - `builtins_list.go` `__list_nth` and `builtins_list_poly.go` `__list_take`/`__list_drop`/
     `_list_tail`: cell paths — tail = the `Tail` cell (O(1), `#1827` semantics preserved);
     `take k`/`drop n` walk their literal counts; bounds errors byte-identical.
   - HOF builtins (`__list_foldl`, `map`, `filter`, `mapAccumL`): unchanged — they consume
     `AsList()` of the *input* (materialized from `range`/literals in practice) and never touch the
     accumulator's representation (`builtins_hof.go:64-84` reads `args[2].AsList()` only).

3. **Not touched:** the eval↔VM bridge (`convert.go`, per-element deep copy both ways — V14), the
   compiler (`compileCons` emits the same `OpCons` ABC — V15), the disassembler, image constants
   (materialized at build), `std/`, the interpreter.

### Implementation Plan

**Phase M1 — the seam (no behavior change)** (~1 d)
- [ ] Add `NewCons`, `Len`, `IsCell`, `Walk` to `internal/bytecode/value.go`; unit tests asserting
      slice-equivalent semantics for each (red if any wrapper diverges from `NewList`/`AsList`).
- [ ] Migrate `vm_arith.go:80` to the accessors (the one raw poke, V5); no other caller changes.
- [ ] `go test ./internal/bytecode/ ./internal/vm/` green; `bench/vm_tail_walk/run.sh` outputs
      byte-identical (this phase must be a no-op to the oracle).

**Phase M2 — the swap** (~1.5 d)
- [ ] `OpCons` → `NewCons`; `OpGetIndex`/`_len`/`__list_nth`/`take`/`drop`/`_list_tail` cell paths.
- [ ] `OpConcat` TagList arm: copy-left/share-right.
- [ ] Identity tests in `list_tail_share_test.go` style, for cells: tail-of-cell shares cells
      (`TestListTailDoesNotCopy` analog), m prepends onto a retained base = O(m)
      (`TestConsBranchingIsLinear` — timing-gate like `vm_arith_fast_test.go`), append-isolation.
- [ ] `bench/vm_tail_walk/build.ail` re-run: per-doubling ratio ≤ ~2.2×, wall at 40k bound in the
      walk/foldl order of magnitude (Goal 2); `walk.ail`/`foldl.ail` unchanged.
- [ ] `make test`, strict-alone smokes (`run.sh` runs them), full parity suites green.

**Phase M3 — docs and bench** (~0.5 d)
- [ ] Extend `bench/vm_tail_walk` with the ratio row (and raise `build.ail` sizes if linear).
- [ ] `docs/LIMITATIONS.md` row 43: name which engine is fixed (VM) and which still awaits LC-4;
      row 40 unchanged (it describes the interpreter).
- [ ] `CHANGELOG` entry; pointer from [m-list-cons-cells-decomposition](../../m-list-cons-cells-decomposition.md)
      (a "VM-half piece" note in its piece list — a doc-text edit, not a programme re-scope); this
      doc's Status → Implemented when the sprint evaluator confirms.

### Files to Modify/Create

**Modified:**
- `internal/bytecode/value.go` (+~120/−~10) — `ListObj` dual store, `NewCons`/`Len`/`IsCell`/`Walk`,
  `Equal`/`String` TagList branches over accessors
- `internal/vm/vm.go` (+~15/−~8) — `OpCons` arm, `OpGetIndex` TagList walk
- `internal/vm/vm_arith.go` (+~12/−~3) — runtimeEq TagList via accessors, `OpConcat` TagList arm
- `internal/vm/builtins.go` (+~4/−~2) — `_len` → `Len()`
- `internal/vm/builtins_list.go` (+~15/−~5) — `__list_nth` cell walk
- `internal/vm/builtins_list_poly.go` (+~25/−~8) — `take`/`drop`/`_list_tail` cell paths
- `internal/bytecode/value_test.go` (+~100) — representation unit tests
- `internal/vm/list_tail_share_test.go` (+~150) — cell identity/clamp/isolation + branching gate
- `bench/vm_tail_walk/run.sh` (+~15) — ratio row, size extension
- `docs/LIMITATIONS.md` (+~6/−~3) — row 43 update
- `changelogs/v0.32-current.md` (+~10) — release note

**Not modified** (load-bearing negatives, V14/V15/V5): `internal/vm/convert.go`,
`cmd/ailang/bytecode_bridge.go`, `internal/bytecode/compiler/**`, `internal/bytecode/disasm.go`,
`internal/eval/**`, `std/**`.

## Examples

**Before** (`vm.go:511`, V1) — Θ(n) per cons, Θ(n²) per consing fold:

```go
old := tail.AsList()
elems := make([]bytecode.Value, 0, len(old)+1)
elems = append(elems, head)
elems = append(elems, old...)          // copies the whole accumulator, every step
frame.Regs[inst.A()] = bytecode.NewList(elems)
```

**After** — O(1) per cons, the accumulator's spine shared, never copied:

```go
frame.Regs[inst.A()] = bytecode.NewCons(head, tailListObj)
// internal/bytecode: &ListObj{Head: &head, Tail: t, N: t.Len() + 1} — one allocation
```

**Effect on the report's workload** (this container, V-P1 → Goal 1):

| | before | after (target) |
|---|---:|---:|
| `plain` 80k, strict VM | 52.11 s | < 1 s |
| per-doubling ratio | ×4.14 | ≤ ~2.2× |
| `accum` (mapAccumL) 80k | 0.37 s | ≤ ~0.5 s (no regression) |

## Conflict Surface

### Syntactic positions touched

No grammar, AST, type, elaboration, or lowering surface changes. The compiler emits the same
`OpCons` ABC for the same `stmt.Cons` (V15). The positions touched are all **runtime dispatch and
representation**:

- `internal/bytecode/value.go:101` — the `ListObj` struct shape (dual store).
- `internal/vm/vm.go:511-521` — the `OpCons` dispatch arm (implementation only).
- `internal/vm/vm.go:551-570` — the `OpGetIndex` TagList arm.
- `internal/vm/vm_arith.go:79-82` — runtimeEq TagList; `OpConcat` TagList arm.
- `internal/vm/builtins*.go` — the `AsList()`/`NewList` call sites enumerated in V5 (41 + 31
  non-test sites across 12 files) — *unchanged* calls, changed backing cost.

### What else lives there

| Position | Existing valid forms | Disposition |
|---|---|---|
| `ListObj` construction | `NewList` (constants, `range`, literals, bridge conversion, every list builtin's result) | stays materialized — byte-identical paths |
| `ListObj` consumption | `AsList()` everywhere; one raw `.Elems` (equality, V5) | equality migrates to accessors; `AsList` contract ("caller must not mutate") unchanged |
| `OpCons` semantics | `R[A] = R[B] :: R[C]`, error `CONS: tail is %s, not List` | same value, same error, O(1) |
| `_list_tail`/`take`/`drop` | `#1827` capacity-capped subslices (`elems[n:len:len]`), identity tests live | materialized path **byte-identical**; cell path returns the tail cell — identity *strengthened*, never weakened |
| Image constants | `NewList`-materialized at build | unchanged, never cells (V13) |

### Disambiguation strategy

Not applicable at the parser level. At the representation level the discriminator is structural and
package-private: a cell has `Head != nil`; a materialized list has `Head == nil`. `Head` is a
pointer precisely so that a zero `Value{}` (unit) head cannot alias "materialized". Empty lists are
always materialized (`NewList(nil)`), so `[]` has one representation forever.

### Programs that MUST still work

Regression fixtures (all exist, V16; outputs must stay byte-identical, walls within the stated
bounds):

- `bench/vm_tail_walk/walk.ail` + `foldl.ail` + `build.ail` — checked oracles (`max(n-1, 0)` /
  `n`), engine equality, strict-alone smokes (V11).
- `examples/runnable/ai_offline_cli.ail` — cons *pattern* binding (`first :: _`).
- `examples/runnable/no_loops_fold.ail`, `no_loops_filter_map.ail`, `effectful_list_t3_foldlE_acc.ail`
  — O(1)-step folds (unaffected by construction cost).
- `examples/runnable/record_update.ail` — record update inside a lambda (the `recacc` shape).
- `examples/runnable/letrec_local_shadow.ail` — the v0.55.0 VM fixture (local letrec).
- `tests/stdlib/list_iterative_test.ail` — `mapAccumL` semantics.
- `internal/vm/list_tail_share_test.go` — the three `#1827` tests, plus this piece's new cell tests.

### What deliberately changes

**Nothing observable.** Outputs, error strings, disassembly, image bytes: identical. The changes are
cost (O(n)→O(1) per cons), allocation volume (Θ(n²)→Θ(n) per consing fold), and retained-memory
shape (contiguous vs per-cell; bounded by Goal 5). `--bytecode` remains non-strict fallback to the
evaluator for uncompilable code — unchanged. The interpreter remains quadratic until LC-4 (V-P5);
the engines will diverge on cons cost between this piece landing and LC-4 landing, and
`docs/LIMITATIONS.md` must say so explicitly (M3).

## Testing Strategy

**Unit tests** (`internal/bytecode/value_test.go`): dual-store invariants — construction forms,
`Len` O(1) both, `AsList` slice-equivalence vs `NewList`, `Equal`/`String` identical across forms,
`Walk` order/shared-prefix safety, empty-list single representation.

**Regression-surface tests** (`internal/vm/list_tail_share_test.go`, extended):
- cell-tail identity (`TestListTailDoesNotCopy` analog: tail of a cell chain shares cells),
- clamp/cap behavior on the materialized path (existing three tests, untouched),
- append-isolation for cells,
- **branching gate**: m prepends onto one retained base measured O(m) — the D-19 objection
  converted to a test (timing-gate pattern per `vm_arith_fast_test.go`).

**Integration:** `bench/vm_tail_walk/run.sh` (module checks, exact stdout, engine equality,
strict-alone smokes, ratio row); the repro from this doc (`rec.ail`) as a manual AC; `make test`,
`make verify-examples`, `make check-boundaries` (no boundary changes expected).

**Manual:** the Problem Statement table re-run on the sprint container before/after (the [P]
protocol from this doc), both engines, recorded in the implementation report.

## Non-Goals

- **The interpreter's representation** — LC-4 owns it ([m-list-cons-cells-decomposition](../../m-list-cons-cells-decomposition.md)); this doc touches no `internal/eval` file.
- **The arena** (front-slack or any variant) — declined by D-19; not re-opened at VM scope either.
- **Bridge cell sharing / killing the per-element deep copy** — `convert.go` and `bytecode_bridge.go` unchanged (V14); a future follow-up can make cells cross the bridge once *both* engines are on C1.
- **`nth`-cost disposition** beyond parity (O(i) walk, `ArrayValue` remains the documented O(1) indexed structure) — mirrors LC-5's disposition; nothing new decided here.
- **Tail-call elimination** — `m-eval-tail-calls` (queued); untouched.
- **Re-ordering the D-28 bar** — this piece queues with the programme; a consumer-urgency escalation is Mark's call, not this doc's.
- **New stdlib surface** — none needed; `mapAccumL` already covers stateful building (V-P4).

## Timeline

- **Day 1 (M1)**: accessors + equality migration + no-op proof (bench byte-identical).
- **Day 2 (M2)**: `OpCons`/`OpGetIndex`/`OpConcat`/list-builtin cell paths + identity/branching tests.
- **Day 3 (M3)**: bench ratio row, LIMITATIONS row 43, CHANGELOG, decomposition pointer, re-run of
  the full Problem Statement table.
- Buffer ×2 → **~6 d honest**, inside the LC programme's per-piece band (D-24 precedent).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Cell walks slow hot *materialized* paths (every builtin result is materialized) | Medium | Dual store: materialized lists keep today's byte-identical paths; only cell chains walk. m6 clause (b) bound (~2×) gates it; `bench/vm_tail_walk` compares within one run. |
| `#1827` tail-sharing semantics regress (walk instead of subslice) | High | Cell path returns the tail *cell* (O(1)); materialized path unchanged; the existing three `list_tail_share_test.go` tests run untouched. |
| Equality/show divergence between forms | Medium | Identity tests across forms in `value_test.go`; `bench/vm_tail_walk` checks engine equality on every timed repetition. |
| Allocation/GC pattern shift (per-cell allocs vs chunked slices) | Medium | Goal 5 bounds retained memory (~2.5×) and iteration (~2×); m6 already measured C1 within both on the eval side (0.950× iteration); measure on VM before M2 exit. |
| Merge conflicts with persistent-collections Phase 2 (also edits `bytecode/value.go`, `vm/builtins*.go`) and LC-3c (`bytecode_bridge.go`) | Medium | Sequencing rule (same convention as [m-persistent-collections](../v0_55_0/m-persistent-collections.md) "Distinction" section): first-lands-first, the other rebases; this piece's diff is small and representation-local. |
| Image-format creep (temptation to add an opcode) | Low | Frozen by decision (V15): same encoding, same constants; `disasm_test.go` and image fixtures are the gate. |
| Queue-position drift (a live consumer report used to jump the D-28 bar) | Low | Stated in High-Impact Decisions; any re-ordering goes to Mark as a directive, not into this doc. |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Cost change only; outputs, errors, traversal order byte-identical (Conflict Surface "What deliberately changes") |
| A2: Replayability | 0 | No trace/effect changes |
| A3: Effect Legibility | 0 | No effect-surface changes; `::` is pure and stays pure |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | The seam is two package-private constructors/accessors; the full surface is unit-testable locally without a VM |
| A6: Safe Concurrency | 0 | Single-goroutine per VM unchanged (V13); no memoization under the default |
| A7: Machines First | +1 | The canonical functional accumulator idiom becomes *actually correct to emit* on the strict engine — the docs stop teaching a workaround for a substrate defect |
| A8: Minimal Syntax | +1 | No new syntax; existing programs get faster unchanged |
| A9: Cost Visibility | +1 | `std/list`'s and `LIMITATIONS`' cost claims become true per-engine; the bench carries a ratio row |
| A10: Composability | +1 | INV-1 holds under all sharing shapes (m-fold onto retained base), matching the eval-side invariant |
| A11: Structured Failure | 0 | Error strings byte-identical |
| A12: System Boundary | 0 | Bridge unchanged |

**Net Score: +6** ✅ Proceed to implementation.

### Hard Violation Check

- [x] A1: no nondeterminism introduced (deterministic walk order, no races under the default)
- [x] A3: no hidden effects
- [x] A4: no ambient authority
- [x] A7: not optimizing human convenience over machine analysis — the idiom is the machine's

## References

- **Bug report**: this task (v0.54.18 `cd33d05`, sunholo/audio consumer, R3-SUNHOLO-AUDIO-REALISM)
- **Programme**: [m-list-cons-cells-decomposition](../../m-list-cons-cells-decomposition.md) (D-19
  ruling, INV-1, LC-0…LC-5); [m-list-repr-spike-M6-report](../v0_34_0/m-list-repr-spike-M6-report.md)
  (420-trial matrix, GO verdict); D-22 (C1) and D-24/D-28 rows in [v1-mission](../../v1-mission.md)
- **Predecessors**: [m-list-cons-quadratic](../../m-list-cons-quadratic.md) (superseded arena doc,
  its Non-Goal line is why this half was unowned — V7/V8);
  [m-foldl-cons-cost-model](../v0_52_0/m-foldl-cons-cost-model.md) (docs honesty + `mapAccumL`,
  shipped v0.52.0); [m-vm-list-tail-share](../v0_53_0/m-vm-list-tail-share.md) (#1827 — the
  tail-sharing fixtures this piece must not regress)
- **Neighbour**: [m-persistent-collections](../v0_55_0/m-persistent-collections.md) (Map/Set/Array;
  explicitly not lists — sequencing rule shared with this piece)
- **Code**: `internal/vm/vm.go:511` (OpCons), `internal/bytecode/value.go:98` (ListObj + its own
  "revisit" comment, V4), `internal/builtins/list.go:87` (the interpreter's twin, LC-4's),
  `bench/vm_tail_walk/` (the AC harness)
- **Issue trail**: [#676](https://github.com/sunholo-data/ailang/issues/676) (email-parse OOM —
  interpreter half), [#745](https://github.com/sunholo-data/ailang/issues/745) (D-19/D-22 rulings),
  #1827 (v0.54.10 tail share)
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- **Bridge cell sharing**: once LC-4 lands C1 on the interpreter, `EvalToBytecode`/`bytecodeToEval`
  can convert list *spines* lazily instead of per-element — the N11 divergence closes and both
  engines share one representation end-to-end. Needs its own design pass (memory-model obligations).
- **`AsList` memoization** with the N21-analog publication-path/race-test table, if a measured hot
  loop demands it (Deferred Decisions).
- **LC-5-analog post-swap tuning** for the VM (iterator-based JSON/effects encoders over cells).
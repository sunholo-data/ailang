# M-FOLDL-CONS-COST-MODEL: the documented-safe list-building path — `foldl` with a consing step, and "prepend then reverse" — is O(n²); std/list's cost claims must say so, the language needs one linear stateful-accumulate primitive, and the VM runs the shape ~2.4× slower than the interpreter

**Status**: PLANNED. Quorum not runnable in this authoring container (no `ailang` binary, no Go toolchain — see W-1/V20/V21); the 25-row Verification Log below is the load-bearing evidence, every code claim backed by a source read at cited lines, every timing marked as the reporter's. Re-run `ailang design-quorum` on this doc before sprint-planning when a runner with a live binary is available (convention: [m-vm-var-pattern-default-arm.md](m-vm-var-pattern-default-arm.md) header).
**Target**: v0.51.2 (docs-honesty + additive stdlib primitive + VM investigation); the same release train as this folder's sibling VM doc
**Priority**: P0 for the docs-honesty fix (Phase 1 — the stdlib actively teaches an O(n²) idiom as "iterative, O(n)"); P1 for the `mapAccumL` primitive (Phase 2) and the VM per-step overhead (Phase 3)
**Estimated**: ~4 days total (Phase 1: 0.5 d · Phase 2: 1.5 d · Phase 3: 2 d, profile-first), 2× where honest
**Dependencies**: None hard. **Coordination required**: [m-interpreter-tail-call-elimination](../v0_51_1/m-interpreter-tail-call-elimination.md) also edits the `RT_REC_003` message text (its High-Impact Decisions table) — see Conflict Surface item 4. **Explicitly blocked on a human elsewhere**: the O(1)-cons substrate fix is D-19 in [m-list-cons-quadratic](../m-list-cons-quadratic.md) and is NOT re-opened by this doc.
**Bug report**: AILANG v0.51.0 (b99dd25, darwin arm64, binary md5 `ed0478cc…`), related to [#676](https://github.com/sunholo-data/ailang/issues/676) but the *documented-safe* path; downstream consumer `stapledons-godot` M1.2b-T3 (331,312-row catalogue transform, `acc.rows = row :: acc.rows`, the idiom std/list's own docs recommend); consumer's own mitigation scheduled as M1.2b-T4 (map/filter + reverse).
**Author**: design-doc-creator, unattended coordinator session. This container has **no Go toolchain and no `ailang` binary** (V20/V21), so wall-clock numbers are the reporter's (run against the real v0.51.0 binary, md5-attached) and every structural claim is verified by reading the cited source lines at this tree (`4460d91b`, clean, v0.51.0-era). The implementer must re-derive the measurements with a live binary as AC-0.

## Routing Lane (per design_docs/PROGRAM.md §4)

**Lane: AILANG fix.** Three sub-asks, three different routings inside that lane:

| Ask | Routing | Why |
|---|---|---|
| (2) "correct the std/list docs: foldl is O(n) only if the step is O(1)" | This doc, Phase 1 | Doc-text honesty bug in `std/` + one error message; no semantics change; unblockable by a single session |
| (1) "make cons onto a uniquely-owned accumulator O(1) amortised in foldl (or a persistent cons list)" | **Split.** The substrate half stays with the parked [m-list-cons-quadratic](../m-list-cons-quadratic.md) (D-19, a named human decision, quorum-blocked on representation direction). This doc ships the *stdlib* half (Phase 2): one additive Go builtin that makes the stateful-accumulate idiom expressible in O(n) **today**, without touching `eval.ListValue` | The substrate change is parked on a human decision; consumers burn now. The gap is real and measured: no linear stateful list-building primitive exists anywhere in the language (V13–V15) |
| (3) "the VM being slower than the interpreter on this shape looks like a separate VM perf bug" | This doc, Phase 3, profile-first | The parked doc explicitly deferred VM `OpCons` as a non-goal (V18); the per-step overhead mechanism is VM-local (`CallClosure` frame allocation, V11) and orthogonal to representation |

The docs-honesty half is not an extension (no motoko prompt shaping can fix `std/list.ail`'s own comments), and the consumer is not behind motoko. The prompt teaching-surface update is routed to the prompt-manager lane (Phase 1, item d).

## Problem Statement

`std/list`'s header tells every reader — human and AI — the documented-safe way to build lists:

> "Build with map/filter/foldl (iterative, O(n)). If you must recurse, prepend then reverse" — `std/list.ail:6-7`

> "Prefer foldl/map; else `x :: acc` then `reverse`" — `std/list.ail:39`

Both claims are wrong in the way that matters. `foldl` **is** iterative (O(1) stack, V3), but its per-element step is arbitrary user code, and the accumulator idiom the language teaches — `\acc x. x :: acc` — executes `::`, which copies the entire flat backing array (V1/V2, V8/V9). The fold is therefore O(n) **only if the step is O(1)**; a consing step is O(n) per element, making the fold O(n²) overall. "Prepend then reverse" is the same quadratic shape in time (the header's own next line says "still not linear", but the `concat` comment's "Prefer foldl … else `x :: acc` then `reverse`" and its 0.36 s / 5,000-record measurement read as an endorsement). The `RT_REC_003` error message repeats the advice ("carry partial results in an accumulator argument, use an iterative std/list helper such as foldl", V6), and the current teaching prompt says "**Prefer map/filter/foldl over manual recursion**" with foldl examples that are all O(1)-step — so the caveat is nowhere a model would see it (V7).

The reporter's minimal repro (no FS, no packages; type-checked and run against the v0.51.0 binary, V17):

```ailang
module consrepro
import std/list (foldl, length)
import std/io (println)
import std/string (split, repeat, intToStr)
pure func rev[a](xs: [a]) -> [a] = foldl(\acc x. x :: acc, [], xs)
export func main(n: int) -> () ! {IO} {
  let xs = split(repeat("ab,", n), ",");
  println(intToStr(length(rev(xs))))
}
```

| Measurement (reporter, v0.51.0 b99dd25) | Value |
|---|---|
| `--bytecode` (VM), N=40,000 / 80,000 / 160,000 | 1.8 s / 5.8 s / **23.4 s** (×4 per doubling → quadratic) |
| interpreter (no `--bytecode`), N=80,000 / 160,000 | 2.5 s / 9.6 s (also quadratic) |
| VM : interpreter wall ratio on this shape | **~2.4× slower on the VM** |
| `std/list.reverse` on the same list | instant (iterative Go builtin, native on the VM since v0.51.0, V16) |

**Impact**: `stapledons-godot` M1.2b-T3 folds 331,312 CSV rows with `acc.rows = row :: acc.rows` — the accumulate-then-reverse idiom, which is what the docs teach. Full GCNS scan: 7 min 46 s on the VM, scaling ~n^1.5–2 (20k rows 4.7 s → 40k 11.7 s → 80k 33 s). Their T4 rewrite (map/filter + reverse) will fix their workload — but every future consumer that needs *stateful* accumulation (a fold whose step depends on the running value: dedup-while-building, stateful filtering, running aggregates with output) cannot use map/filter, and today has **no linear option at all**: hand-rolled recursion is quadratic + depth-capped, foldl+cons is quadratic, `std/array.append` is also O(n) per call (V14), `std/iter` is only a Stop/Continue signal type (V15). That gap is this doc's Phase 2.

**This is systemic, not a one-off** (audit per CLAUDE.md principle 3): the defect class is "cost-model claims about list-building idioms", and the false claims appear in four places — the `std/list.ail` header (V4), the `std/list.ail` `concat` comment (V5), the `RT_REC_003` message (V6), and the teaching prompt (V7). Phase 1 fixes all four with one cost-model statement, not four ad-hoc patches.

## Verification Log

Provenance: **[R]** = read first-party at this tree (`4460d91b`), command and observed result below; **[P]** = the reporter's measurement against their binary (v0.51.0 `b99dd25`, md5 `ed0478cc…`), inherited, not re-run here (W-1); **[N]** = negative-existence claim, grep/read included per the skill's rule.

| # | Claim | Command / location | Observed |
|---|---|---|---|
| V1 [R] | `eval.ListValue` is a flat Go slice | `sed -n '84,95p' internal/eval/value.go` | `type ListValue struct { Elements []Value }` — no arena fields (the parked doc's Option B never landed) |
| V2 [R] | `::` copies the whole tail in the interpreter | `sed -n '86,103p' internal/builtins/list.go` | `listConsImpl`: `make([]eval.Value, 0, 1+len(tail.Elements))`, `append(result, tail.Elements...)` — O(n) per cons |
| V3 [R] | `_list_foldl` itself is iterative (O(1) stack) — the loop is fine, the step is not | `sed -n '202,219p' internal/builtins/list_iterative.go` | plain `for i, elem := range list.Elements` calling `ctx.FnCallerN(fn, …)`; no recursion |
| V4 [R] | `std/list` header claims foldl is the safe builder | `sed -n '1,8p' std/list.ail` | "BOTH `x :: xs` and `xs ++ ys` copy … Build with map/filter/foldl (iterative, O(n)). If you must recurse, prepend then reverse" — no step-cost caveat anywhere in the file (grep "only if the step" → 0) |
| V5 [R] | `concat` comment endorses prepend-then-reverse and foldl | `sed -n '36,39p' std/list.ail` | "the same list built by prepend-then-reverse = 0.36 s / 294 MB). Prefer foldl/map; else `x :: acc` then `reverse`" — the 5,000-record number carries no quadratic qualifier |
| V6 [R] | `RT_REC_003` message recommends the accumulator idiom + foldl, uncaveated | `cat internal/eval/recursion_limit_error.go` | "…carry partial results in an accumulator argument, use an iterative std/list helper such as foldl or map…" |
| V7 [R] | Current teaching prompt teaches foldl without the step-cost caveat | `grep -n "Prefer \`map\`/\`filter\`/\`foldl\`" cmd/ailang/prompts/v0.16.6.md` + `sed -n '267,292p'` | line 686: "**Prefer `map`/`filter`/`foldl` over manual recursion**"; all foldl examples (lines 278–293) are O(1)-step (`acc + x`, string concat); prompts are versioned and frozen (cmd/ailang/prompts/versions.json `frozen.at`), so an in-place edit is not the mechanism — see Phase 1d |
| V8 [R] | VM `OpCons` copies the whole tail | `sed -n '463,473p' internal/vm/vm.go` | `make([]bytecode.Value, 0, len(old)+1)`; `append(elems, old...)` — O(n) per cons, same class as V2 |
| V9 [R] | Source `x :: acc` reaches `OpCons` on the VM (not the eval adapter) | `sed -n '186,199p' internal/gen/lower/expr.go`; `sed -n '145,163p' internal/bytecode/compiler/collections.go` | saturated and curried `App(VarGlobal("::"))` lower to `stmt.Cons` → `compileCons` emits `OpCons`; `IsCallableBuiltin` explicitly excludes `::` ("symbolic entries lower to dedicated opcodes", `internal/bytecode/builtin_adapt.go:129-138`) |
| V10 [R] | VM foldl path: builtin → `CallClosure` per element | `internal/bytecode/builtin_names.go:133-141` (`"__list_foldl"` in `HOFBuiltinNames`); `internal/vm/builtins_hof.go:62-78` (`hofBuiltinListFoldl`: `for i, e := range args[2].AsList() { acc, err = caller.CallClosure(...) }`) | per-element VM re-entry confirmed |
| V11 [R] | Each `CallClosure` allocates a fresh `Frame` + register slab | `sed -n '92,120p' internal/vm/vm.go`; `sed -n '38,45p' internal/vm/frame.go` | `newFrame` does `make([]bytecode.Value, proto.NumRegs)` and the Frame struct per call; `vm.Stack = append(...)`; contrast `reuseFor` (frame.go:50-63) which exists for TAIL_CALL reuse but is **not** used by `CallClosure` |
| V12 [R] | VM element copy width ≈ 2.5× interpreter's | `sed -n '73,79p' internal/bytecode/value.go` | `bytecode.Value{Tag, Int, Flt, Bool, Obj any}` ≈ 40 B/element vs `eval.Value` interface header 16 B — a constant-factor candidate for the 2.4× (hypothesis H2, Phase 3 profiles it; not asserted as measured) |
| V13 [N] | No stateful linear list-building primitive exists (`mapAccumL`/`scanl`-family) | `grep -rin "mapaccum\|map_accum\|scanl" std/ internal/builtins/ internal/vm/` | only false-positive hits (`migration_validator.go` "scanLegacyLocation"); no such builtin — the V13 gap this doc fills |
| V14 [R] | `std/array.append` is also O(n) per call (no mutable builder escape hatch) | `sed -n '84,86p' std/array.ail` | "Append element to end of array, returning NEW array. O(n) due to copy." + "TIP: For bulk building, prefer fromList" |
| V15 [R] | `std/iter` is only a Stop/Continue signal (35 lines) | `cat std/iter.ail` | `FoldStep[a] = Continue(a) \| Stop(a)` + two helpers; no builder |
| V16 [R] | `reverse` is instant on both engines (consistent with the report) | `std/list.ail:28-29` (`= _list_reverse(xs)`); `internal/bytecode/builtin_names.go:127` (`"__list_reverse"` native); changelogs/v0.32-current.md v0.51.0 "Added — native VM ports of polymorphic list builtins (#1447)" | O(n) Go builtin, native VM port since v0.51.0 |
| V17 [P] | Reporter's timings (table above) | `ailang run --quiet --bytecode --caps IO --args-json N consrepro.ail`, real v0.51.0 binary, md5-attached in the report | quadratic on both engines; VM ≈ 2.4× interpreter; ×4 per doubling on the VM |
| V18 [R] | The O(1)-cons substrate design exists, is parked on a human decision, and explicitly excluded the VM | Read [m-list-cons-quadratic](../m-list-cons-quadratic.md) (whole) | Status "PARKED — needs-human-review", D-19 (A: linear-chain arena vs B: true cons cells) open; Non-Goals: "VM `OpCons` — stays O(n); parity deferred" |
| V19 [R] | The tail-call doc also edits the `RT_REC_003` message | [m-interpreter-tail-call-elimination](../v0_51_1/m-interpreter-tail-call-elimination.md), High-Impact Decisions | "RT_REC_003 message gains a remedy that now exists…" — text-edit collision, coordinated in Conflict Surface item 4 |
| V20 [N] | No Go toolchain / make in this container | `which go` / `go build` → "command not found"; `command -v make` → empty | measurements inherited from the reporter; code claims verified by source reads (this table) |
| V21 [N] | No `ailang` binary in this container (quorum, `ailang check`, doc-search all unavailable) | `command -v ailang`; `find -maxdepth 3 -name ailang -type f` | absent — see W-1 |
| V22 [R] | Tuple syntax needed by Phase 2 is attested in shipped, runnable code (types, literals, match patterns) | `grep -n "(a, b)" std/list.ail` (foldr's `(a, b) -> b` param type, zip's `[(a, b)]`); `sed -n '242,258p' internal/parser/parser_literals.go` (`ast.Tuple` literals); `examples/runnable/guards_basic.ail:30` (`(x, y) if x > y => …` tuple match arm) | tuple types, literals and match-arm patterns all parse and are elaborated (`internal/elaborate/expr_data.go:146`, `patterns.go:169-179`) — the archive fixture `examples/archive/nested_match_variants/*` is a *parse-failure* repro and is deliberately NOT cited; final spelling still gated by AC-0 |
| V23 [R] | New pure builtins are CI-ratcheted: a builtin in no coverage bucket fails the build | changelogs/v0.32-current.md v0.51.0 (#1447) "TestPureBuiltinCoverage … fails when a new pure builtin lands in none"; `internal/vm/builtin_coverage_test.go` | Phase 2's builtin must land in the HOF-native bucket, by design |
| V24 [R] | Regression fixtures exist and use O(1)-step folds (must keep passing, byte-identical) | `grep -n "foldl" examples/runnable/no_loops_fold.ail examples/runnable/effectful_list_t3_foldlE_acc.ail` | both exist; steps are `acc + x`, string concat — unaffected by any phase here |
| V25 [R] | HOF builtin tables are order-coupled | `internal/bytecode/builtin_names.go:127-141` comment "Order MUST match vm.HOFBuiltinTable" | adding `__list_mapAccumL` must append to BOTH tables in the same position (Conflict Surface item 2) |

### Workarounds note (W-1)

This authoring container has no Go toolchain and no `ailang` binary (V20/V21), so: (a) wall-clock numbers are the reporter's, marked [P]; (b) the skill's live-`ailang check` hard gate could not be run — every language-syntax claim below is instead attested by shipped, type-checked source (`std/list.ail`, `examples/` per V22/V24), and the sprint-executor MUST run `ailang check` on the new `std/list.ail` and the benchmark files as AC-0, before anything else; (c) quorum was not run — re-run before sprint-planning (Status header). Code-structure claims are all first-party reads at `4460d91b` and cite file:line.

## Goals

**Primary Goal:** every cost claim the language makes about building lists tells the truth, and every list-building idiom the language teaches has a linear implementation available on both engines.

**Success Metrics:**
- A reader of `std/list.ail`'s header can correctly predict the cost of `foldl(\acc x. x :: acc, [], xs)` — the file states the O(1)-step condition explicitly (AC-1).
- A stateful list-building fold has a linear path: `mapAccumL` over 160k elements completes in < 1 s on the interpreter and the VM (AC-3), where the consing-foldl shape takes 9.6 s / 23.4 s (V17).
- The VM : interpreter wall ratio on the consrepro shape drops from ~2.4× toward parity, or the residual is fully explained by a committed profile (AC-4).
- `stapledons-godot`'s M1.2b-T4-class rewrite becomes guidance the docs themselves give: map/filter for elementwise, `mapAccumL` for stateful (AC-1 example text).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Do NOT re-open the O(1)-cons representation question here.** The substrate fix (ask 1's first half) stays with the parked [m-list-cons-quadratic](../m-list-cons-quadratic.md) doc and its open human decision D-19 (arena vs cons cells) | Re-litigating a quorum-parked, human-blocked design in a second doc forks the decision record; two docs would claim the same fix | human (already decided — D-19, recorded there) | — | high (avoided) |
| Phase 2's answer to ask (1) is an **additive stdlib primitive** (`mapAccumL`, Go-delegated), not a runtime representation change | Gives the consumer-class workload a linear path now, without the 900-site blast radius the parked doc measured; compositional, optional, no semantics change | this doc | design | med |
| Phase 1 is **docs honesty first and unconditionally** — it does not wait for Phase 2/3 | The stdlib is actively teaching an O(n²) idiom as O(n); every day it ships, another consumer builds the quadratic shape (this report is exactly that) | this doc | design | low |
| Phase 3 is **profile-first**: no `CallClosure` rewrite is merged before pprof attributes the per-step cost | The 2.4× has two plausible mechanisms (per-call frame alloc, V11; copy width, V12) with different fixes; guessing picks the wrong one | this doc | design | med |
| `RT_REC_003` text edits are **coordinated with, and textually merged into, whichever of this doc / the tail-call doc lands second** | Both docs edit the same string; the tail-call doc's message change is semantically larger (its remedy then exists) | agent | design | low |
| Teaching-prompt update goes through a **new prompt version** (prompt-manager lane), not an in-place edit of `v0.16.6.md` | Prompts are versioned and frozen (`versions.json` `frozen.at`); in-place edits break the version contract (V7) | agent | design | low |

### Design Freeze

- [x] Phase 1 scope fixed: four doc surfaces (`std/list.ail` header, `std/list.ail` concat comment, `RT_REC_003` message, new-prompt-version note) — one cost-model statement, no semantics.
- [x] `mapAccumL` semantics fixed: pure, `f: (s, a) -> (b, s)`, single pass, output list built in one Go-side allocation; no effectful variant in this sprint (future work).
- [x] VM port of `mapAccumL` is in-scope and mandatory (the consumer runs `--bytecode`); HOF-table append-at-end (V25).
- [x] Phase 3 deliverable is *either* a measured fix *or* a measured account — "we profiled it and here is why" is an acceptable terminal state; a speculative rewrite is not.

## Solution Design

### Overview

Three phases, independently shippable, smallest-first:

1. **Phase 1 — Tell the truth about step cost (P0, ~0.5 d).** Rewrite the two `std/list.ail` comment blocks so the cost model is unconditional: *any* shape that grows a list one element at a time is O(n²) in time — hand-rolled recursion, foldl-with-consing-step, prepend-then-reverse — because `::` and `++` copy (V1/V2/V8/V9). `foldl` is O(n) **only if the step is O(1)**. State the three linear alternatives by shape (map/filter; `mapAccumL` once Phase 2 lands; non-list accumulators are always fine). Give the `concat` comment's 0.36 s / 5,000-record number its missing scaling law (≈×4 per doubling — the reporter's 23.4 s at 160k is that curve). Add the same one-line caveat to the `RT_REC_003` message. Route a prompt-version bump through prompt-manager.
2. **Phase 2 — `mapAccumL`: the linear stateful-accumulate primitive (~1.5 d).** The general shape "fold that also emits one element per input, with state" is what consing foldl actually implements, minus the quadratic copies. Add, following the exact M-ITERATIVE-LIST delegation pattern (`std/list.ail:28,51` → `_list_*` builtins):

   ```ailang
   -- mapAccumL: stateful map in one pass, O(n).
   -- f: (state, element) -> (output, newState). Output list is built in a
   -- single allocation (no per-step copy). The linear replacement for
   -- foldl(\acc x. … x :: acc…, ([], s0), xs)-style accumulation.
   export pure func mapAccumL[a, b, s](f: (s, a) -> (b, s), s0: s, xs: [a]) -> ([b], s)
     = _list_mapAccumL(f, s0, xs)
   ```

   Go side (`internal/builtins/list_iterative.go` family): iterate `xs`, call `f(state, elem)` via `ctx.FnCallerN` (V3 pattern), destructure the returned pair, append the output element to a pre-sized `[]Value`, thread the state. One allocation, O(n) total. VM side: `hofBuiltinListMapAccumL` (V10 pattern — `CallClosure` per element into a pre-sized slice), registered in `HOFBuiltinNames`/`HOFBuiltinTable` appended at the end (V25) so existing indices are untouched; it lands in the HOF-native coverage bucket, keeping the #1447 ratchet green (V23). Tuple return type is attested syntax (V22); the implementer re-checks with `ailang check` (AC-0).

   This does **not** fix `::` — a consing `foldl` stays quadratic (that is D-19's to fix) — but it makes the *idiom the docs teach* expressible linearly, which is what the consumer needs.
3. **Phase 3 — VM per-step overhead on HOF folds (profile-first, ~2 d).** Reproduce consrepro at N ∈ {40k, 80k} on both engines, take CPU+alloc profiles of the VM run, and attribute the 2.4× before touching anything. Candidate mechanisms, both verified as present (not yet as dominant):
   - **H1 — per-callback frame allocation**: `CallClosure` does `newFrame` (Frame + `make([]bytecode.Value, NumRegs)`) and a `vm.Stack` append per element (V11), while a `reuseFor` mechanism exists in the same file but is unused by this path — fix shape: a per-VM free-list/pool of frames for `CallClosure`, with re-entrancy care (a closure may itself call a HOF builtin; the pool must handle nesting, not recursion-depth assumptions).
   - **H2 — copy width**: `OpCons` moves ~40 B/element (V12) vs the interpreter's 16 B interface headers — inherent to the tagged-value representation; not fixable at this layer; would cap achievable parity at ~2.5× on copy-bound shapes if H1 is not dominant.
   The sprint merges a fix only for whatever pprof actually shows; otherwise it ships the profile + a LIMITATIONS.md note ("VM HOF folds pay per-callback frame allocation; interpreter-favored on copy-heavy fold steps") and hands the H2 residue to the representation decision (D-19's doc already tracks VM parity as future work).

### Implementation Plan

**Phase 1 (docs honesty, P0):**
- [ ] `std/list.ail:1-8` header rewrite per Overview (state the O(1)-step condition; classify the three shapes and their linear implementations)
- [ ] `std/list.ail:36-39` concat comment: add the scaling law to the 0.36 s measurement; replace "Prefer foldl/map" with the step-conditional form
- [ ] `internal/eval/recursion_limit_error.go:22`: qualify the foldl advice — "use an iterative std/list helper such as foldl or map **(a foldl step that conses onto a list accumulator is itself O(n) per step — use map/filter/mapAccumL to build lists)**"; merge textually with the tail-call doc's edit if it lands first (V19)
- [ ] Prompt-manager lane: draft the caveat paragraph for the next prompt version (do not edit `v0.16.6.md` in place, V7); deliver as a note in the sprint report, not a code change
- [ ] `make verify-examples` + stdlib goldens (comment text feeds the stdlib doc pipeline; confirm no golden pins these comment lines — if one does, update it in the same commit)

**Phase 2 (`mapAccumL`):**
- [ ] `internal/builtins/list_iterative.go`: register + implement `_list_mapAccumL` (type builder `(s, a) -> (b, s)`; ~80 LOC incl. metadata)
- [ ] `internal/vm/builtins_hof.go`: `hofBuiltinListMapAccumL` (~40 LOC)
- [ ] `internal/bytecode/builtin_names.go` `HOFBuiltinNames` + `internal/vm/builtins.go` `HOFBuiltinTable`: append at end, same position in both (V25)
- [ ] `std/list.ail`: export `mapAccumL` with an O(n) comment + a worked before/after example
- [ ] Coverage ratchet: confirm `TestPureBuiltinCoverage` places it in the HOF-native bucket (V23)
- [ ] Tests: Go unit tests both engines (incl. empty list, state never used, output not a list-slice alias); an AILANG-level example `examples/runnable/mapAccumL_running_total.ail` (stateful filter: keep rows above the running mean — the shape map/filter cannot express)
- [ ] `ailang check` on the edited `std/list.ail` and the new example (AC-0)

**Phase 3 (VM profile-first):**
- [ ] Benchmarks: commit the reporter's consrepro + a mapAccumL variant under `bench/` (check existing bench layout first; `ls bench/`), with a run script for both engines
- [ ] `go tool pprof` CPU + alloc on the VM run at N=80k; attribute per-step cost to H1/H2/dispatch/other with numbers
- [ ] If H1 dominant (or co-dominant): frame pool for `CallClosure` with nesting-safe re-entrancy + `-race` test; measure again
- [ ] Either the merged fix (AC-4a) or the committed profile + LIMITATIONS.md account (AC-4b); a benchmark note in the doc's Implementation Report either way

### Files to Modify/Create

**Modified:** `std/list.ail` (comment rewrites + one export, ~+18/−8 LOC), `internal/eval/recursion_limit_error.go` (+1/−1 LOC), `internal/builtins/list_iterative.go` (+~80 LOC), `internal/vm/builtins_hof.go` (+~40 LOC), `internal/bytecode/builtin_names.go` (+1 LOC), `internal/vm/builtins.go` (+1 LOC), `docs/LIMITATIONS.md` (+~6 LOC if AC-4b).
**New:** `examples/runnable/mapAccumL_running_total.ail` (~30 LOC), VM frame-pool files only if Phase 3 profiling says so, bench scripts under `bench/`.

## Conflict Surface

This touches `internal/eval/` (message text), `internal/vm/`, `internal/bytecode/` (builtin tables), `internal/builtins/` — the section is required. The "positions" being extended: (a) the pure-builtin registry + VM HOF table, (b) user-facing cost-model prose in three surfaces, (c) `CallClosure`'s allocation behavior.

### What else lives in those positions, and disambiguation

1. **Builtin namespace + coverage ratchet.** Every new pure builtin must land in exactly one bucket of `TestPureBuiltinCoverage` (#1447) or the build fails (V23). Other residents of the `$builtin`-module `_list_*` family: `_list_map`, `_list_filter`, `_list_foldl`, `_list_sortBy`… (list_iterative.go). Name `_list_mapAccumL` follows the convention; no collision (grep, V13). Disambiguation: registry name uniqueness + the ratchet test is the mechanical check.
2. **VM HOF tables are order-coupled.** `HOFBuiltinNames` order MUST match `vm.HOFBuiltinTable` (V25); indices feed `OpBuiltinCallHOF`. Existing entries `__list_map`, `__list_filter`, `__list_foldl`, `__str_*`, `__xml_parseFold`, `__list_sortBy`, `__list_flatMap` must keep their indices — therefore **append-only**. Regression proof: the existing HOF tests (`internal/vm/builtins_hof_test.go`) pin dispatch by name, and `validateBuiltinTables` runs at package init.
3. **Cost-model prose is load-bearing, not decorative.** The `std/list.ail` header is consumed by AI doc tooling and teaches every model that reads the stdlib (that is how this bug shipped: the consumer *followed* it). Other residents of the "teaching" position: `docs/docs/reference/no-loops.md` (fold laws, "Approach 2: Fold Combinators" — its guarantees list omits step cost; Phase 1 adds one sentence there too), the prompt (frozen, versioned — new version only, V7), `RT_REC_003` (item 4). Disambiguation: one canonical cost-model sentence, reused verbatim across surfaces, so they cannot drift again: *"foldl is O(n) only if the step is O(1); a step that conses onto (or concatenates to) a list accumulator is O(n) per element, making the fold O(n²)."*
4. **`RT_REC_003` message text is edited by two planned docs.** [m-interpreter-tail-call-elimination](../v0_51_1/m-interpreter-tail-call-elimination.md) changes the same string (adds the tail-call remedy, V19); this doc adds the step-cost caveat. Resolution: the later-landing doc carries the merged text; both docs' ACs must reference the *merged* string so neither can revert the other. Pinned tests: `rt_rec_003_message_test.go` bans the phrase "enable tail recursion" (its lines 118–139, per the tail-call doc) and `recursion_test.go` asserts only the `RT_REC_003` substring — both survive this edit (checked by the tail-call doc's inventory and the parked doc's V30).
5. **`CallClosure` is on the concurrency path.** Frames are per-VM; `Fork()`-style sharing of module state means a HOF callback can re-enter the VM. Any frame pool (Phase 3, only if profiled) must be nesting-safe: a callback that itself calls `__list_map` acquires a second frame while the first is live. Disambiguation: pool = free-list (LIFO stack of returned frames), never "the one scratch frame"; the `-race` test must cover nested HOF calls.
6. **Fold laws / equational reasoning are unaffected.** `mapAccumL` is a new combinators, not a change to `foldl`; the fusion law in no-loops.md keeps its truth conditions (V4's caveat is about *cost*, not the laws).

### Programs that MUST still work

1. `examples/runnable/no_loops_fold.ail` — O(1)-step folds (V24); byte-identical behavior, Phase 1 only touches its file's sibling comments in `std/list.ail`.
2. `examples/runnable/effectful_list_t3_foldlE_acc.ail` — foldlE type-inference regression (V24).
3. The reporter's `consrepro` — must produce the same output at the same semantics, just with honest docs about its cost.
4. Every existing VM HOF dispatch (indices unchanged, V25) — `internal/vm/builtins_hof_test.go` green.
5. `std/list.reverse` / `_list_reverse` on both engines (V16) — untouched.

### What deliberately changes

- Comment text in `std/list.ail`, one error-message string, one sentence in `no-loops.md`, a new prompt *version* (not an edit to a frozen one).
- New stdlib surface: `std/list.mapAccumL` (+ its VM-native implementation).
- Possibly (Phase 3, profile-gated): `CallClosure` allocates from a pool — allocation behavior only, no observable semantics.
- Nothing else. A consing `foldl` is still quadratic after this doc — fixing that is D-19.

## Examples

**Before (the documented-safe path, quadratic — V17 timings):**

```ailang
pure func rev[a](xs: [a]) -> [a] = foldl(\acc x. x :: acc, [], xs)
-- 160,000 elements: 9.6 s interpreter / 23.4 s VM (reporter, V17) — and after
-- this doc's Phase 1, the stdlib finally says so instead of recommending it.
```

**After Phase 2 (stateful accumulate, linear):** running totals — the step
needs the accumulated state *and* emits one element per input, which is
exactly the shape `map`/`filter` cannot express and the shape a consing
`foldl` was being used for:

```ailang
import std/list (mapAccumL)
-- mapAccumL(f, s0, xs): one pass, f: (state, element) -> (output, newState)
-- runningTotals([1, 2, 3]) = [1, 3, 6] — built in ONE allocation, O(n)
pure func runningTotals(xs: [int]) -> [int] {
  match mapAccumL(\st x. (st + x, st + x), 0, xs) {
    (outs, _) => outs
  }
}
```

What Phase 2 promises (the contract the tests pin): `mapAccumL(f, s0, xs)`
visits elements left-to-right once; `f` returns an `(output, newState)` pair;
outputs are assembled in a single pre-sized allocation; total O(n) time, O(1)
evaluator stack; empty input → `([], s0)`. Syntax attestation for every
construct used above: tuple *types* in signatures (`std/list.foldr`'s
`(a, b) -> b`, `std/list.ail:53`), tuple *values* flowing out of
`std/list.zip` (`[(a, b)]`, `std/list.ail:43,50`), tuple *literals* parsed at
`internal/parser/parser_literals.go:242-258` and elaborated at
`internal/elaborate/expr_data.go:146`, and tuple *patterns* in match arms
(`examples/runnable/guards_basic.ail:30`). Per W-1 the implementer re-runs
`ailang check` on the shipped example (AC-0) — this doc fixes the semantics
and cost contract, not the final spelling.

## Success Criteria (Acceptance)

- [ ] **AC-0** *(gate for everything)* Build a live binary (`make build`), run `ailang check` on the edited `std/list.ail` + new example + benchmark files, and re-derive the reporter's table row for N=40,000 both engines before starting (this container could not: V20/V21). Baseline recorded in the sprint log.
- [ ] **AC-1** `std/list.ail` contains the canonical cost-model sentence (foldl O(n) only if step O(1); consing/concat step → O(n²)); the `concat` comment's 0.36 s measurement carries its scaling law; `RT_REC_003` message carries the same caveat. **Fails today** (V4/V5/V6: the claims are present without the conditions).
- [ ] **AC-2** `mapAccumL` type-checks and runs on both engines: `mapAccumL` over a 160,000-element list completes in < 1 s on interpreter and VM (reporter's baseline for the consing shape: 9.6 s / 23.4 s at N=160k, V17 — this AC fails today *by absence* of the primitive, V13).
- [ ] **AC-3** `TestPureBuiltinCoverage` green with `_list_mapAccumL` in the HOF-native bucket; `internal/vm/builtins_hof_test.go` green; both HOF tables appended in the same position (V23/V25).
- [ ] **AC-4** VM:interpreter wall ratio on consrepro at N=40,000: **(a)** ≤ 1.5× after a profile-attributed fix, **or (b)** a committed pprof artifact + LIMITATIONS.md note attributing ≥ 90% of the gap to named mechanisms. **Base: ~2.4× (reporter, V17) — (a) fails today.**
- [ ] **AC-5** `make test`, `make verify-examples` green; fixtures in Conflict Surface §"MUST still work" byte-identical.
- [ ] **AC-6** Docs updated: CHANGELOG entry under the target version; no-loops.md gains the one-sentence cost caveat; prompt-version note delivered to prompt-manager (not an in-place frozen-prompt edit, V7).

## Testing Strategy

- **Go unit tests**, both engines: `mapAccumL` (empty list → `([], s0)`; state threading; output length = input length; no aliasing between output list and input; `sortBy`-adjacent regression not affected).
- **VM table lockstep**: appended-position test (compile a module using `mapAccumL` under `--bytecode`, run, compare interpreter output) — the existing HOF test pattern.
- **Phase 3**: `-race` with nested HOF callbacks if the pool lands; benchmark script committed with min-of-3 convention (repo precedent: parked doc's AC measurements).
- **Docs**: grep-based assertions are brittle; instead the AC-1 check is a review step in the sprint-evaluator with the exact sentences quoted in the Implementation Report.

## Non-Goals

- **Fixing `::` / the representation** — parked doc + D-19 (V18). A consing foldl remains quadratic after this doc; the docs will finally say so.
- **Tail-call elimination** — [m-interpreter-tail-call-elimination](../v0_51_1/m-interpreter-tail-call-elimination.md), v0.51.1.
- **`mapAccumLE` / effectful variant** — future work; the pattern is established and trivially extensible, but this sprint ships the pure shape only.
- **Array builders / mutable arrays** — `std/array`'s cost model is already honest (V14).
- **General VM↔interpreter perf parity program** — only this shape's per-step overhead; the representation-width residue (H2) belongs to the representation decision.

## Timeline

**Day 1**: AC-0 baseline; Phase 1 complete (all four surfaces), `make verify-examples` + goldens.
**Day 2–3**: Phase 2 (builtin + VM port + tables + tests + example); AC-2/AC-3 measured on both engines.
**Day 4**: Phase 3 profile; fix-or-account decision; AC-4; CHANGELOG; Implementation Report.
**Total: ~4 days** (2× where honest: the estimate already includes the measurement overhead that burned previous sprints).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-------------|
| Phase 3's frame pool introduces re-entrancy/concurrency unsoundness in the VM | High | Profile-gated: pool only if H1 is measured dominant; LIFO free-list (never single scratch frame); nested-HOF + `-race` tests mandatory; kill-switch = revert to `newFrame` per call (base behavior) |
| `RT_REC_003` text merge with the tail-call doc lands contradictory text | Med | Conflict Surface item 4: later doc merges; both docs' ACs reference the merged string |
| Comment rewrites break stdlib doc-pipeline goldens | Low | `make verify-examples` first; golden updates in the same commit; goldens are generated artifacts, not contracts |
| `mapAccumL` teaches a new idiom models won't find | Low | Phase 1's rewritten header names it as *the* stateful path; prompt-version note (prompt-manager lane) |
| Reporter timings don't reproduce (different machine) | Low | AC-0 re-derives first; thresholds are ratios and asymptotics (×4/doubling), not absolute seconds |
| Scope creep into the representation fix | High (process) | Design Freeze row 1; Non-Goals first bullet; any PR touching `eval.ListValue` fields under this doc's name is out of scope by definition |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantic change; `mapAccumL` is pure; pool affects allocation only |
| A2: Replayability | 0 | Traces unchanged (new builtin logs like its siblings) |
| A3: Effect Legibility | 0 | `mapAccumL` is pure; no new effects |
| A4: Explicit Authority | 0 | No capabilities touched |
| A5: Bounded Verification | +1 | One mechanical coverage ratchet already covers the new builtin (V23); the doc's claims are locally checkable (grep the sentence, run the bench) |
| A6: Safe Concurrency | 0 | Pool is per-VM; `-race` gate if it lands |
| A7: Machines First | +1 | The bug class is "the teaching surfaces lie to AI code-writers"; fixing the cost model is precisely a machines-first fix (the reporter's team followed the docs) |
| A8: Minimal Syntax | 0 | No syntax; one stdlib function in an established family |
| A9: Cost Visibility | +1 | The core axiom of this doc: hidden quadratic costs in documented-idiom clothing are removed from four teaching surfaces |
| A10: Composability | 0 | Additive combinator; folds and laws untouched |
| A11: Structured Failure | 0 | Only message-text improvement (caveat), same error code |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Proceed**

### Hard Violation Check

- [x] A1 (Determinism): no nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): the change exists to stop the substrate punishing machine-idiomatic code

## Duplicate / Coverage Gate (explicit, per skill §4)

The auto-search could not run (V21); the manual top match is decisive:

- [m-list-cons-quadratic](../m-list-cons-quadratic.md) (planned, **PARKED — needs-human-review**, similarity: same defect family, high). **Distinct and not duplicative**: that doc is the *substrate* fix (value representation), quorum-blocked on a named human decision (D-19: arena vs cons cells) with VM `OpCons` explicitly a non-goal; this doc is the *cost-model honesty* fix (Phase 1), the *additive stdlib primitive* that unblocks consumers while D-19 is pending (Phase 2), and the *VM per-step overhead* investigation the parked doc explicitly excluded (Phase 3, its V22/Non-Goals). It does not re-adjudicate any parked decision. If D-19 lands, Phase 2 remains useful (stateful emit) and Phase 1 remains true.
- [m-interpreter-tail-call-elimination](../v0_51_1/m-interpreter-tail-call-elimination.md) (planned, v0.51.1): tail calls, not step cost; one shared artifact (the `RT_REC_003` string), coordinated (Conflict Surface item 4).
- [m-perf4-bytecode-interpreter](../v1_1_0/m-perf4-bytecode-interpreter.md) (planned, P3 exploratory, targets v0.9.0+): a whole-engine rewrite program, not this shape's per-step overhead; no overlap in scope or horizon.
- M-ITERATIVE-LIST ([implemented/v0_9_2](../implemented/v0_9_2/m-iterative-list-builtins.md)): the delegation pattern Phase 2 *extends*; its docs (`std/list.ail` header) are the artifact Phase 1 corrects — an implementation's cost claim was generalized beyond what the implementation guarantees.

## Related Documents

- [m-list-cons-quadratic](../m-list-cons-quadratic.md) — the parked substrate design (D-19 decision A/B pending: linear-chain arena vs true cons cells). This doc routes ask (1)'s representation half there and ships the stdlib half.
- [m-interpreter-tail-call-elimination](../v0_51_1/m-interpreter-tail-call-elimination.md) — sibling; shares the `RT_REC_003` string (merged-text contract above).
- [m-vm-var-pattern-default-arm](m-vm-var-pattern-default-arm.md) — sibling in this folder; same authoring-container constraints, same re-run-quorum convention.
- [M-ITERATIVE-LIST](../implemented/v0_9_2/m-iterative-list-builtins.md) — the Go-delegation pattern Phase 2 follows; the doc whose "iterative, O(n)" claim Phase 1 makes conditional.
- [M-VM-PURE-BUILTIN-COVERAGE](../implemented/v0_51_0/m-vm-pure-builtin-coverage.md) — the coverage ratchet Phase 2 must satisfy (V23).

## References

- **Source report**: coordinator task on the consrepro report (v0.51.0 b99dd25, binary md5 `ed0478cc…`), related to [#676](https://github.com/sunholo-data/ailang/issues/676); consumer `stapledons-godot` M1.2b-T3/T4.
- [design_docs/PROGRAM.md](../PROGRAM.md) §4 — routing lanes.
- [Design Axioms](/docs/references/axioms).

## Future Work

- `mapAccumLE` (effectful variant) once a consumer needs it — mechanical extension of the Phase 2 pattern.
- A `FoldStep`-bounded `mapAccumLStepWhile` for early-exit stateful scans (combine with `std/iter`), if the XML bounded-scan consumers ask.
- D-19 (in the parked doc): the representation fix that would make consing foldl itself linear — until it lands, `mapAccumL` is the only linear stateful-accumulate path.
- VM representation-width residue (H2) if Phase 3 accounts for the gap that way.

---

**Document created**: 2026-10-02 (design-doc-creator, unattended coordinator session; no Go toolchain/binary in container — Verification Log provenance marks apply)
**Last updated**: 2026-10-02

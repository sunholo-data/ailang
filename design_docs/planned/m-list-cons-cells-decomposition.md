# M-LIST-CONS-CELLS-DECOMPOSITION: True Cons Cells / Structural Sharing — Programme Roadmap

**Status**: Planned — **quorum-cleared via the narrow-refinement carve-out at mission iteration 234**
(round 1 blocked 2-of-2 → designer revision; round 2 blocked 2-of-2 → both premises measured, carve-out
applied; ratified iter-98). Each piece below still gets its own design doc + quorum when picked.
**Target**: v0.35.0 programme (multi-sprint; first piece can land in v0.34.0)
**Priority**: P0 — implements Mark's D-19 ruling; permanent fix for [#676](https://github.com/sunholo-data/ailang/issues/676)
**Estimated**: 8 pieces, 16.5–22.5 person-days sequential; ~13–17 days wall-clock with the parallel lanes
**Dependencies**: D-19 (RESOLVED — see below). No code dependencies.
**Planner-Lane**: opus-required for LC-1/LC-4 (runtime core value representation); mechanical lanes eligible for cheaper routing

## What this doc is

Mission-control Gate 3 requires that multi-week strategic items are **decomposed into sprint-sized
design docs (≤3–4 days each), queued individually** — not executed. This doc is that decomposition
for the true-cons-cells programme. It names the pieces, their seams, ordering, and kill criteria.
**It deliberately does not design any piece**; each piece runs its own design-doc-creator pass and
its own quorum when it reaches the head of the queue.

## The ruling being implemented

`::` is O(n), so every AILANG list built by prepending is Θ(n²) in time and allocation (#676; full
first-party measurement record in
[m-list-cons-quadratic.md](m-list-cons-quadratic.md), V1–V38, at `88631976e`). The arena design
that doc chose was quorum-blocked on a correct objection: a front-slack arena is amortized O(1)
only along a **linear use chain** — under persistent branching (`x :: base` repeated while `base`
stays live) every prepend after the first loses the CAS and degrades to a full copy, Θ(m·len(base)).
Decision D-19 put the direction to Mark:

- **A** — accept the arena as a linear-chain optimization; **B** — true cons cells / structural
  sharing: O(1) under **all** sharing, correct by construction, a much larger programme needing
  decomposition first.

**Mark ruled `B`** — verified first-party this session: comment by `MarkEdmondson1234` on
[#745](https://github.com/sunholo-data/ailang/issues/745) at `2026-08-19T10:58:40Z`, body exactly
`D-19 : B` (N1). The ruling itself names decomposition as the precondition; this doc is that
deliverable. **Naming note to avoid an inversion:** the superseded doc's *Option A* ("true cons
cells with structural sharing") is what D-19 answer **B** selects; that doc's *Option B* (the
front-slack arena, which it marked CHOSEN) is now **DECLINED**.

## The invariant the programme buys

> **INV-1: `x :: xs` is O(1) worst-case in time and allocation for every live list `xs`,
> regardless of sharing** — including m-fold prepending onto the same retained base, tails obtained
> by pattern match, and concurrent prepends from Fork-forked goroutines — **and N lists sharing a
> spine retain O(total distinct cells) memory** (structural sharing), with pattern-match tail
> extraction remaining O(1).

This is precisely what the declined arena did not deliver (it was amortized O(1) along a linear use
chain only). Immutability and safe publication are **required properties, not established facts**
until LC-4's enforcement and publication checks pass. Cells must sit behind a narrowly scoped
package/API with an unexported representation and constructors/read-only accessors; the permanent
`listrep` analyzer rejects assignments to cell fields outside constructor initialization. Deleting
the raw `Elements` field proves migration completeness, not immutability. Every piece below is
either **[INV]** (directly delivers/protects
INV-1), **[ENABLING]** (makes an [INV] piece possible/safe), or **[MITIGATION]** (interim relief
for #676, not traceable to INV-1 — marked honestly as such).

Costs the invariant does **not** hide (each owned by a named piece): indexed access `_list_nth`
becomes O(i) (LC-1 measures, LC-5 disposes — AILANG already has `ArrayValue` with O(1) indexing as
the documented alternative, N13); per-element retained memory rises from 16 B contiguous to an
estimated ~32 B per heap cell plus allocator overhead (a **derivation, not a measurement** — LC-1
measures it); `++` becomes O(len(left)) sharing the right operand, which makes the whole
`[x] ++ rest` / `concat(small, recurse)` family in `std/list` linear (9 builders, N14) but leaves
`reverse`'s `concat(growing, [x])` shape quadratic — which is why the already-queued
`m-stdlib-reverse-delegates-to-builtin` remains **required**, not optional, under cons cells (N15).

## Verification Log

Rows N1–N20 were re-derived first-party this session at `dedf3b91f` (= `origin/dev`, N2), in the
iteration worktree. N21 is explicitly a pending LC-4 verification obligation, not observed evidence.
Scope travels with every count. Inherited-but-not-rerun claims are NOT here — they are in [Unverified
/ needs measurement](#unverified--needs-measurement).

| # | Claim | Command | Observed |
|---|---|---|---|
| N1 | Mark's D-19 ruling is `B` | `gh api repos/sunholo-data/ailang/issues/745/comments --paginate --jq '.[] \| select(.created_at=="2026-08-19T10:58:40Z") …'` | `{"body":"D-19 : B","created_at":"2026-08-19T10:58:40Z","user":"MarkEdmondson1234"}` |
| N2 | Tree state | `git log -1 --format='%H %D'`; `git merge-base HEAD origin/dev` | `dedf3b91f021ae65eaa4a23e651cd4ac4ad67736 HEAD -> docs/mission-iter229-decomposition, origin/dev…`; merge-base = same SHA; `git status --short` empty before authoring |
| N3 | Textual blast radius, **in `internal/` and `cmd/`, `*.go` including tests**, with same-scope control | `test -d internal && test -d cmd` → "dirs exist"; `grep -rn '\.Elements' … \| wc -l`; `grep -rn 'ListValue{' … \| wc -l`; control `grep -rn 'func ' … \| wc -l` | **902** `.Elements` refs; **386** `ListValue{` constructions; control 20,361 `func` hits — matches the controller's same-scope numbers exactly |
| N4 | The 902 is an over-approximation: `Elements` fields exist on many non-eval types | `grep -rn 'Elements *\[\]' internal/ cmd/ --include='*.go' \| grep -v _test.go` | **23 declaration-site matches across 9 packages** — `internal/types` (TupleType + 2 JSON shims), `internal/core` ×5, `internal/ast` ×5, `internal/typedast` ×5, `internal/eval` ×3, plus 1 local-var false positive (`elaborate/dictionaries.go:298` `newElements`) |
| N5 | Only three `Elements []Value` owners are eval values; ListValue has exactly 2 methods today (no accessor layer; control = the 2 methods the grep must see) | Read `internal/eval/value.go:84-104,251`; `grep -rn 'func ([a-z]* \*ListValue)' internal/eval/` | `ListValue` (line 85), `ArrayValue` (104), `TupleValue` (251), each `Elements []Value`; methods: `Type()` (88), `String()` (89) — nothing else |
| N6 | Per-package `.Elements` distribution, **same scope as N3, including tests** | `grep -rn '\.Elements' … \| cut -d: -f1 \| sed -E 's#^(internal/[^/]+\|cmd/[^/]+)/.*#\1#' \| sort \| uniq -c \| sort -rn` | builtins 408, testing 103, **types 74**, effects 47, eval 43, **gen 36**, **pipeline 34**, **smt 29**, ast 16, elaborate 15, runtime 14, cmd/ailang 14, parser 12, vm 11, embed 11, format 7, core 7, typedast 4, lsp 4, apiserver 3, cmd/wasm 3, linked/link/iface 2+2+2, telemetry 1 (bold = compile-time `Elements`, not eval values) |
| N7 | Non-test subset, same scope | same pipeline + `grep -v _test.go` | **522** total; builtins 138, types 74, eval 43, testing 38, gen 36, effects 33, pipeline 31, smt 29, … |
| N8 | `ListValue` (the type name) per-package distribution, same scope incl tests | `grep -rn 'ListValue' … \| cut -d: -f1 \| sed …dirs… \| uniq -c` | 750 total: **builtins 444, effects 154, eval 45, testing 39, core 17 (helper names like `IsListValue` on CoreExpr — compile-time), embed 16, vm 10, runtime 10**, cmd/wasm 4, apiserver 3, cmd/ailang 3, repl 2, elaborate 2, telemetry 1 |
| N9 | `::` copies the whole tail; `++` copies both operands | Read `internal/builtins/list.go:87-107` (`listConsImpl`) and `:161-184` (`listConcatImpl`) | cons: `make([]eval.Value, 0, 1+len(tail.Elements))` + `append(result, tail.Elements...)`; concat: `make(…, len(l1)+len(l2))` + two spread appends |
| N10 | Pattern-match tail is an O(1) subslice alias today (must not regress) | Read `internal/eval/eval_patterns.go:254-255` | `tailElements := listVal.Elements[len(p.Elements):]` then `&ListValue{Elements: tailElements}` |
| N11 | VM `OpCons` is independently O(n); the eval↔VM bridge deep-copies per element in both directions (decoupled — divergence documented, non-goal) | Read `internal/vm/vm.go:429-440`; `cmd/ailang/bytecode_bridge.go:132-142,191-200` | OpCons: `make([]bytecode.Value, 0, len(old)+1)` + `append(elems, old...)`; bridge: per-element conversion loops both ways |
| N12 | Fork shares module-level environments read-only across goroutines (the concurrency constraint) | Read `internal/eval/eval_evaluator.go:166-170`, `internal/eval/env.go:1-14` | Fork comment: "shares read-only state … each HTTP request goroutine gets its own Fork"; env comment: "module-level environments are shared (read) across goroutines" over `sync.RWMutex` |
| N13 | Raw-`Elements` escape sites: **3 currently known syntax-matched escapes** (wording per `gpt5-6-sol` round-2 `proposed_fix`; the exhaustive audit is LC-2's type-aware analyzer, not these greps), one invisible to the `, nil` pattern (instrument note: a `.Elements, nil` grep alone finds 2 — `GetList` returns without the error value) | `grep -rn '\.Elements, nil' internal/ cmd/ --include='*.go' \| grep -v _test.go` → 2; `grep -rn 'GetList' internal/effects/testctx/` + read `mock_context.go:353-355` | `builtins/safe_cast.go:97`, `embed/convert.go:346`, and `testctx/mock_context.go:353-355` (`return v.(*eval.ListValue).Elements`). Also confirmed: `_list_nth` registered at `list.go:278`, bounds-checked via `len(list.Elements)` (`:320`); `ArrayValue` documented "O(1) indexed access" (`value.go:103`) |
| N14 | `std/list` right-recursion builder census (two shapes; a literal-`++` grep alone misses the second) | `grep -nF '++' std/list.ail`; `grep -n 'flatMap' std/list.ail` + read lines 202-206, 263-270 | 7 literal `++` sites: concat:40, zip:49, mergeBy:92-93, take:103, mapE:235, filterE:246; plus 2 `concat(small, recursive-call)` sites: flatMap:205, flatMapE:268 → **9 right-recursion builders**, all linear under cons cells |
| N15 | `reverse` is the sole left-append builder and stays quadratic under cons cells; the iterative builtin exists with 0 std callers (control fires) | Read `std/list.ail:29-35`; `grep -n '_list_reverse' internal/builtins/list.go std/*.ail`; control `grep -c '_list_map' std/list.ail` | `reverse` = `concat(reverse(rest), [x])` (growing **left** operand — cons-concat copies the left spine each step); `_list_reverse` registered `list.go:546-567`, **zero** hits in `std/*.ail`; control = 1 |
| N16 | Symmetric-switch surface, file granularity (the inherited "~15 switches" does not reproduce as stated). **CORRECTED at iteration 234** — the round-1 form intersected ListValue×ArrayValue only and omitted `TupleValue`, so it under-counted by more than 2× (`gemini-3-1-pro` round-2 objection, CONFIRMED) | `grep -rlE 'case \*(eval\.)?(List\|Array\|Tuple)Value' internal/ cmd/ --include='*.go' \| sort`, pairwise `comm -12`, both scopes | **Non-test:** L=**15**, A=**3**, T=**8**; L∩A=**3**, L∩T=**7**, L∩A∩T=**3**, **L∩(A∪T)=7**. **Incl. tests** (N16's original scope): L=**16**, A=**3**, T=**9**; L∩A=**3**, L∩T=**8**, **L∩(A∪T)=8**. The 7 non-test symmetric files: `cmd/ailang/bytecode_bridge.go`, `builtins/canonical_key.go`, `builtins/list.go`, `embed/convert.go`, `eval/eval_patterns.go`, `testing/executor_helpers.go`, `testing/value_splice.go`. Statement-level count remains unmeasured → LC-2's analyzer |
| N17 | Go version supports range-over-func iterators for the accessor API | `head -4 go.mod` | `go 1.26.6` (≥1.23) |
| N18 | RT_REC_003 still advertises a nonexistent option; fix is ALREADY QUEUED separately | `grep -rn 'enable tail recursion' internal/ cmd/ --include='*.go'`; `grep -n 'm-rt-rec-003' design_docs/v1-mission.md` | one site, `internal/eval/eval_operations.go:58`; backlog row `m-rt-rec-003-advertises-nonexistent-option` exists — **not duplicated here** |
| N19 | `m-stdlib-reverse-delegates-to-builtin` and `m-eval-tail-calls` are already queued backlog rows | `grep -n 'm-stdlib-reverse-delegates-to-builtin\|m-eval-tail-calls' design_docs/v1-mission.md` | rows at lines 2220 and 2239; the TCO row says "answer **B** (cons cells) may change what the evaluator work has to preserve" |
| N20 | Duplicate/coverage gate | `ailang docs search "cons cells structural sharing persistent list" --neural` | Neural unavailable this session — CLI fell back to SimHash (output header says "SimHash search"); SimHash top hits are keyword noise (e.g. an unrelated match-error doc at 1.00). The one genuinely overlapping doc is `m-list-cons-quadratic.md` — the superseded doc this programme replaces; distinction is the entire subject of this doc. The old doc's own neural search on this topic returned nothing above 0.29 |
| N21 | **LC-4 required verification (pending, not a current-code fact):** enumerate every publication path for shared lists and its Go happens-before edge, then race-test every enumerated path | In LC-4's worktree: `rg -n 'go |\.Fork\(|chan |<-|Lock\(|Unlock\(|RLock\(|RUnlock\(' internal/eval internal/effects internal/runtime internal/telemetry internal/builtins cmd --glob '*.go'`; record a reviewed path→edge table; run one named `go test -race` target per row | **PENDING LC-4.** The row is incomplete until every publication path, its concrete edge (mutex, channel, initialization-before-goroutine-start, etc.), and the corresponding passing race test are recorded. Until then this doc treats immutability and safe publication as required properties, not established facts |
| N22 | **Escape-site caller/aliasing audit** (`gpt5-6-sol` round-2 objection, measured rather than forwarded): every in-repo caller of the three escape APIs, and whether any mutates the returned slice | `grep -rn 'SafeAsList(\|ToList(\|testctx.GetList(' internal/ cmd/ --include='*.go'`, then `grep -A3` for index-assignment / `append` into the result; scope + module check `head -1 go.mod` | **Zero production callers — every in-repo caller is a test.** `SafeAsList` → `builtins/safe_cast_test.go` only (control: 8 total hits incl. def); `embed.ToList` → `embed/embed_test.go` only (the `internal/gen/golang/codegen_runtime_collections.go:177` hit is a *string literal* emitted into generated code, not a caller; control: 6); `testctx.GetList` → `builtins/list_test.go` only, and it is a test helper by construction (control: 16). **No call site mutates the returned slice** — no index-assignment or `append`-into-result at any of them (same-scope known-positive control: `builtins/array.go:121,:483` match the pattern). **All three live under `internal/`** in module `github.com/sunholo-data/ailang`, so the Go toolchain forbids import from outside this module — they are NOT external Go-facing APIs and no versioning/deprecation path is owed |

## The decomposition

The programme is a strangler: an accessor seam goes in over the **unchanged** slice representation,
consumers migrate mechanically behind a type-aware ratchet, and only then does the representation
swap inside the narrowly scoped cell package/API — with the deletion of the `Elements` field as the
compile-time proof that migration was complete. The spike runs first because it is the cheap
experiment that can kill
or re-scope the whole programme before any migration effort is spent.

A structural fact discovered while verifying (N4/N6): **grep cannot size this migration.**
`.Elements` fields exist on ≥22 struct types across 9 packages (AST, core, typed-AST, TupleType,
eval's three value types…), so the 902/522 textual counts are upper bounds contaminated by
compile-time structures (`internal/types` 74, `internal/gen` 36, `internal/smt` 29 …), and even
within value-handling packages the grep cannot distinguish `ListValue.Elements` from
`TupleValue.Elements`/`ArrayValue.Elements` — only `ListValue` changes. The true migration surface
must be measured by a type-checker-driven `go/analysis` tool; producing that number is LC-2's first
deliverable, and every migration AC is denominated in the analyzer's count, not grep's.

### LC-0 `m-list-interim-communication` — **[MITIGATION]** (0.5 day)

**Scope.** The only piece that touches nothing structural: (a) a `docs/LIMITATIONS.md` entry for
the quadratic-prepend defect and the 10,000-frame depth cap (absent at base per the old doc's V28),
with the honest workarounds (build via the iterative builtin-backed `std/list` functions —
`map`/`foldl`/`takeMap` allocate once and never cons; `--max-recursion-depth` for the depth wall);
(b) a status comment on #676 stating the D-19 outcome, the programme shape, and the workarounds.
Any AILANG snippet in either text must be `ailang check`-verified by the piece (none is written in
this umbrella).
**Seam owned:** `docs/LIMITATIONS.md` + the #676 issue thread. **Does NOT:** touch code; duplicate
the two already-queued interim rows (N18/N19). **Depends on:** nothing; can run immediately, in
parallel with everything. **Estimate:** 0.5 d.
**AC shape:** the LIMITATIONS entry exists and names both defects with measured numbers (would red
if the entry were missing or cited unmeasured claims); the #676 comment posted; any embedded
AILANG has an `ailang check` transcript.

**What happens to #676 in the meantime — stated honestly.** This programme is multi-week; #676 is a
live user-reported OOM. The interim lane is: LC-0 (communication + honest docs) plus the two
**already-queued, D-19-independent** rows — `m-stdlib-reverse-delegates-to-builtin` (fixes the
worst measured stdlib case: 10.2 s / 172.7 MB at n=2,000) and
`m-rt-rec-003-advertises-nonexistent-option` (stops the runtime advertising a fix that does not
exist) (N18/N19). **I recommend AGAINST resurrecting the arena as a bounded interim**: it is ~4–5
days of work that LC-2/LC-4 then rewrite (same files: `value.go`, `list.go`), i.e. throwaway plus
merge churn in exchange for compressing #676's memory relief by roughly two weeks. But this is a
judgement call the D-19 ruling did **not** decide — Mark declined the arena as the *permanent*
answer, which leaves a bounded interim formally open. If the ~3–4-week wait for LC-4 is
unacceptable for the #676 reporter, an interim-arena piece needs **Mark's explicit assent** (a
directive on #745); absent that, this decomposition proceeds without it.

### LC-1 `m-list-repr-spike` — **[INV]** — THE RISKIEST-PIECE KILLER, RUNS FIRST (2–3 days)

**Scope.** A throwaway, off-to-the-side benchmark harness (scratch package or worktree branch — no
production wiring) implementing 2–3 candidate representations against the current slice as control:
(i) plain cons cell with cached length `{head Value, tail *List, n int}`; (ii) chunked/unrolled
cons (fixed chunk ≤ ~32 elements; prepending into a shared chunk copies at most one chunk = O(1)
worst-case with a constant, recovering slice-like locality and amortizing the per-element
overhead). Measures, per candidate: worst-case prepend **under branching** (m prepends onto one
retained base — the exact shape that killed the arena), linear build, iteration/sum throughput,
`nth` sweep, materialize-to-slice, per-element retained bytes (the ~32 B/cell figure above is a
derivation only), allocation count, and GC behavior on a #676-repro-shaped workload. Outputs: the
chosen representation, the measured table, a draft accessor API spec sized against real call-site
needs (N17: `iter.Seq` available), and an explicit go/no-go.
**Seam owned:** none in production — that is the point. **Does NOT:** modify `internal/eval`,
`internal/builtins`, or any consumer; commit to an API. **Depends on:** nothing. **Estimate:** 2–3 d.
**AC shape:** each candidate has all cells of the benchmark matrix filled with commands + numbers
(would red if a cell were asserted from theory); the branching benchmark shows O(m), not
O(m·len(base)), for the chosen candidate.

**Kill / re-scope criterion (programme-level, provisional — ratified in the spike's own doc):** if
NO candidate simultaneously achieves (a) O(1) worst-case prepend under sharing, (b) iteration
throughput within ~2× of the slice baseline on the builtin-family benchmarks, (c) per-element
retained memory within ~2.5× of the slice's 16 B/element, (d) O(1) `length`, and (e) encapsulation
behind the narrow constructor/read-only API with mechanically distinguishable constructor
initialization for the permanent analyzer — the programme
STOPS and returns to Mark with the data: D-19 gets re-opened with a measured case for either the
chunked hybrid at relaxed bounds or a rescoped answer A. Spending ~3 days to earn the right to
spend ~16 more is the point of ordering this first. **Estimate remains 2–3 d:** this enforcement
feasibility check is folded into the already-required candidate implementation and draft API spec,
not added as a production analyzer deliverable (LC-2 owns that work).

### LC-2 `m-list-accessor-api` — **[ENABLING]** (3–4 days)

**Scope.** Two halves, one seam. (1) The accessor layer on `*eval.ListValue`, implemented over the
**unchanged** slice representation as thin wrappers (today the type has only `Type()`/`String()`,
N5): length, indexed get, range-over-func iteration (N17), materialize-to-slice (documented O(n)
copy), and constructors (`from slice`, `empty`, `cons`) — exact surface comes from LC-1's draft
spec, decided in this piece's own doc. (2) The `listrep` type-aware `go/analysis` tool: reports
every `.Elements` selector whose receiver type-checks as `*eval.ListValue` (grep cannot do this,
N4), emits the **true** migration count and a per-package baseline file, and gates CI as a ratchet
(count may only decrease; a seeded-violation fixture proves the gate fires — the empty-result trap
rule: a gate that has never fired is a claim, not a gate). The analyzer also permanently rejects
assignments to cell fields outside constructor initialization unless a mechanically enforced
alternative is demonstrated; after LC-4 deletes `Elements`, only the migration-count ratchet
retires to "field must not exist" while the cell-assignment rule remains.
**Seam owned:** `internal/eval/value.go` (additive methods only) + `tools/linters/listrep/` +
`make/ci.mk` wiring. **Does NOT:** change representation or behavior anywhere; migrate any
consumer. **Depends on:** LC-1 (API shape); the analyzer half has no dependency and MAY start in
parallel with LC-1. **Estimate:** 3–4 d (increased by 1 day for the permanent assignment rule and
its seeded positive/negative fixtures).
**AC shape:** accessor unit tests assert slice-equivalent semantics (red if any wrapper diverges);
the analyzer's count at HEAD is recorded as the baseline (red if it disagrees with a hand-audit of
one sampled package); CI fails on a seeded new `.Elements` reference and on a seeded assignment to
a cell field outside constructor initialization, while constructor initialization remains accepted.

### LC-3a `m-list-migrate-builtins` — **[ENABLING]** (2–3 days)

**Scope.** Migrate the largest cluster — `internal/builtins` (444 `ListValue` refs incl. tests, N8;
408 textual `.Elements` incl. tests, N6) — to the LC-2 accessors: `list.go`, `list_iterative.go`,
the set/JSON/canonical-key/SMT-mirror builtins, `safe_cast.go`. Includes this lane's members of the
**7** symmetric List/Array/**Tuple** switch files (N16, corrected at iteration 234): `canonical_key.go`
and `list.go`. Purely mechanical, analyzer-guided, and behavior-preserving **except** for the escape
site named next, which is an intentional and separately-gated semantic change.
The `safe_cast.go` raw-slice escape (N13) migrates to the materialize accessor in this lane. Per
`gpt5-6-sol`'s round-2 `proposed_fix` option (2), this is classified as an **intentional API semantic
change** — `SafeAsList` returns a snapshot, not an alias — documented as such, with a
mutation-focused compatibility test. Option (1) (retain alias semantics + version/deprecate) is NOT
taken, and the reason is measured rather than asserted: N22 records zero production callers and zero
mutating call sites, and `internal/` forbids external import, so there is no consumer to version for.
**Seam owned:** `internal/builtins/**`. **Does NOT:** change representation, semantics, or
performance; touch other packages. **Depends on:** LC-2. **Parallel with:** LC-3b/LC-3c (disjoint
packages; worktree isolation per the shared-checkout policy). **Estimate:** 2–3 d.
**AC shape:** `listrep` count for `internal/builtins` = 0 (red if any site remains); `go test
./internal/builtins/` + `make verify-examples` green; a **mutation-focused compatibility test** pins `SafeAsList`'s
snapshot semantics (mutate the returned slice, assert the source list is unchanged — red if the
alias is restored); no output diff on the old doc's five named regression fixtures.

### LC-3b `m-list-migrate-runtime-effects` — **[ENABLING]** (2–3 days)

**Scope.** Migrate the runtime cluster: `internal/eval`'s own consumers outside the value-core
(patterns incl. `eval_patterns.go:254-255`'s tail construction, equality, `String()`),
`internal/effects` (154 `ListValue` refs incl. tests, N8 — the effects ABI encoders),
`internal/runtime` (10), `internal/telemetry` (1 — verify the inherited "length-only" claim while
there). The pattern-tail site gets an accessor that today returns the O(1) subslice and tomorrow
returns the O(1) shared tail (N10) — the seam where structural sharing will actually pay.
The `internal/effects/testctx/mock_context.go` raw-slice escape (N13) migrates to the materialize
accessor in this lane, classified per `gpt5-6-sol`'s round-2 option (2) as an intentional
snapshot-not-alias semantic change with a mutation-focused compatibility test (see N22 and LC-3a).
This lane also owns **1** of the 7 symmetric List/Array/Tuple switch files (N16): `eval_patterns.go`.
**Seam owned:** `internal/eval` (non-core files), `internal/effects/**`, `internal/runtime`,
`internal/telemetry`. **Does NOT:** touch `value.go`'s representation. **Depends on:** LC-2.
**Parallel with:** LC-3a/LC-3c. **Estimate:** 2–3 d.
**AC shape:** `listrep` count for these packages = 0; pattern-match microbench shows tail
extraction unchanged (red if the accessor added a copy); a **mutation-focused compatibility test** pins `testctx.GetList`'s
snapshot semantics (red if the alias is restored); full `go test ./internal/...` green.

### LC-3c `m-list-migrate-periphery` — **[ENABLING]** (2 days)

**Scope.** Everything else the analyzer flags: `internal/testing` (39 `ListValue` refs incl. the
shrinker at `testing/shrink.go`), `internal/embed` (16, incl. the `ToList` escape at
`convert.go:346`), `cmd/ailang` (bytecode bridge, N11), `cmd/wasm`, `internal/vm` (bridge-facing
eval refs only — the VM's own `bytecode.Value` repr is out of scope, N11), `apiserver`, `repl`,
`elaborate`, `lsp`, plus whatever the analyzer shows in `core`/`link`/`linked`/`iface` (N8's
textual hits there are mostly compile-time helpers — the analyzer decides, not grep). Only the
`internal/embed/convert.go` raw-slice escape (N13) migrates to the materialize accessor in this lane,
classified per `gpt5-6-sol`'s round-2 option (2) as an intentional snapshot-not-alias semantic change
with a mutation-focused compatibility test (see N22 and LC-3a). This lane owns **4** of the 7
symmetric List/Array/Tuple switch files (N16): `cmd/ailang/bytecode_bridge.go`, `embed/convert.go`,
`testing/executor_helpers.go`, `testing/value_splice.go`.
**Seam owned:** all remaining flagged packages. **Does NOT:** change the VM's representation or the
bridge's deep-copy semantics. **Depends on:** LC-2. **Parallel with:** LC-3a/LC-3b. **Estimate:** 2 d.
**AC shape:** `listrep` global count = 0 outside `internal/eval`'s value-core allowlist (red
otherwise); bridge round-trip tests green; a **mutation-focused compatibility test** pins
`embed.ToList`'s snapshot semantics (red if the alias is restored).

### LC-4 `m-list-cells-swap` — **[INV] — THE RISKIEST PIECE** (3–4 days)

**Scope.** The payoff: swap `ListValue`'s internals to LC-1's chosen representation behind a
narrowly scoped package/API with unexported representation and constructors/read-only accessors,
and **delete the `Elements` field** — the compile-time proof that LC-3a–c missed nothing (any
straggler becomes a build break, not a latent corruption). `::` becomes cell
allocation (O(1) worst-case under all sharing — INV-1), `++` becomes left-spine-copy +
shared-right, pattern-tail becomes the shared tail pointer. Immutability and safe publication remain
required properties, not established facts, until the permanent `listrep` cell-assignment rule
passes and N21 is completed. LC-4 must enumerate every publication path for shared lists and its
specific Go happens-before edge (mutex, channel, initialization-before-goroutine-start, etc.), add
a race test covering each path, and record the results in N21. A memory-model comment and those
`-race` tests are mandatory deliverables, as is a **branching benchmark AC**: m prepends onto one
retained base costs O(m) — the gpt5-6-sol objection that killed the arena, converted into a
regression gate. Perf ACs are inherited from the old doc's measured bases: repro n=12,800 (with
`--max-recursion-depth 200000`) from 1,467.9 MB to the piece-doc's target; takebench from
784.5 MB; linearity ratio from 3.72.
**Seam owned:** `internal/eval/value.go` + the new narrowly scoped cell/spine package/API + `listConsImpl`/
`listConcatImpl` internals. **Does NOT:** touch the VM (`OpCons` stays O(n), documented divergence,
N11); tune non-cons builtins (LC-5); fix the depth cap (that is `m-eval-tail-calls`, N19).
**Depends on:** LC-1 (repr), LC-2, LC-3a, LC-3b, LC-3c — **all complete**. **Estimate:** 3–4 d.
**AC shape:** the field is gone and the repo compiles (red = a missed consumer); the permanent
analyzer rejects every seeded post-construction cell-field assignment and accepts constructor
initialization; N21 contains the complete publication-path→happens-before-edge→race-test table and
every named race test is green; branching benchmark O(m); repro/takebench RSS within piece-doc
bounds; `make test` + `make verify-examples` green; five named fixtures byte-identical output.

### LC-5 `m-list-post-swap-tuning` — **[INV — protects the envelope]** (2–3 days)

**Scope.** The honest-cost pass after the swap: re-tune iterative builtins for the new
representation (build-forward strategies vs materialize-then-convert); iterator-based JSON/effects
encoding where the analyzer shows materialize hot spots; the `nth`-cost disposition (document O(i);
point indexing workloads at `ArrayValue`, which is already documented "O(1) indexed access", N13 —
whether `std/list.nth`/`last` warrant a spine-walk builtin is this piece's decision); full
benchmark matrix vs the pre-swap bases; `docs/LIMITATIONS.md` + CHANGELOG + the VM-divergence note.
**Seam owned:** `internal/builtins` hot paths + docs. **Does NOT:** change semantics; re-open the
representation. **Depends on:** LC-4. **Estimate:** 2–3 d.
**AC shape:** the benchmark matrix has no cell regressed beyond the spike-ratified bounds (red
otherwise); docs updated with measured, not asserted, numbers.

## Ordering & parallelism

```
LC-0 ──────────────────────────────────────────────► (anytime, independent)
LC-1 ──► LC-2 ──► { LC-3a ∥ LC-3b ∥ LC-3c } ──► LC-4 ──► LC-5
          ▲ (analyzer half of LC-2 may start ∥ LC-1)
m-eval-tail-calls (already queued): unblocked by the D-19 answer; runs ∥ LC-2/LC-3
```

**Rationale.** LC-1 first is riskiest-first discipline: it is the cheap experiment that can kill
the programme before migration effort is spent, and its output (the API spec) feeds LC-2. LC-2
before any migration because the ratchet must exist before the count starts moving. LC-3a/b/c are
package-disjoint and parallelizable in isolated worktrees: LC-3a owns `safe_cast.go`, LC-3b owns
`effects/testctx/mock_context.go`, and LC-3c owns only the `embed/convert.go` escape. LC-4 requires
the analyzer to read zero
everywhere first — field deletion is only a *proof* if the tree already compiles without stragglers.
LC-5 must follow LC-4 because its baselines are the swap's own numbers.
**Interaction with `m-eval-tail-calls`** (queued, was blocked on D-19, now answerable — N19): it
touches evaluator control flow, not representation, so it can proceed in parallel with LC-2/LC-3;
its doc should re-state what it must preserve given answer B, and it should not land between LC-4
and LC-5 without re-running LC-4's perf bases. Honest residual for both programmes: TCO eliminates
*self-tail-calls* — the #676 `gen` shape is tail-recursive and will benefit, but `std/list`'s
right-recursion builders (`[x] ++ concat(rest, ys)` — the recursive call is *inside* the `++`) are
not tail calls and stay depth-capped at 10,000 even after both land. That is a `std/list` shape
issue (iterative delegation per function), named here so nobody re-derives it as a surprise.

**Totals:** 8 pieces; 16.5–22.5 person-days sequential; ~13–17 days wall-clock with the LC-3 lanes
parallel. Every piece is ≤ 3–4 days per Gate 3.

## Riskiest piece & the de-risking spike

**LC-4 (the swap) is the riskiest piece**: it changes the runtime's central value type, and its
failure modes (perf-envelope collapse on iteration/index-heavy workloads, memory blow-up from
per-cell overhead, mutation outside construction, incomplete safe-publication reasoning, or a
subtle sharing bug under Fork) are exactly the ones a compile-green tree can hide. Immutability and
safe publication are required properties, not established facts, until the permanent analyzer and
the complete N21 publication-path/race-test matrix pass. **LC-1 is the cheap experiment that de-risks it FIRST** — 2–3 days of throwaway benchmarks
answering the questions LC-4 would otherwise discover in week three. The explicit abandonment
criterion lives in LC-1's section above: no candidate inside the (a)–(e) bounds → the programme
stops and D-19 is re-opened with measurements, not vibes. Secondary risk — migration behavior
drift across ~500+ sites — is bounded mechanically: the analyzer ratchet, per-cluster zero-counts,
and the field deletion as terminal proof.

## Conflict Surface (programme-level)

The programme touches `internal/eval` (value representation) — this section is mandatory. Each
piece-doc must carry its own sharpened version; this is the programme-level inventory, re-verified
at `dedf3b91f` where marked. **The central inversion vs the declined design:** the arena's core
conflict-avoidance move was "`Elements` stays a materialized slice, so all ~902 textual consumers
stay valid." Answer B abandons exactly that — which is why the migration pieces exist, and why the
consumer classes below map to pieces rather than to "unchanged".

| Consumer class | Evidence | Programme disposition |
|---|---|---|
| Direct `.Elements` reads/constructions (upper bounds: 902 textual refs / 386 constructions incl. tests, in `internal/` + `cmd/` `*.go`, N3; true `ListValue`-typed count unknown until LC-2) | N3/N4/N6/N7 | Migrated to accessors (LC-3a–c), field deleted (LC-4). Grep cannot attribute types (N4) — the analyzer is the instrument |
| Pattern-match tail alias, O(1) today | N10 | Becomes the O(1) shared tail pointer — the seam where sharing pays (LC-3b/LC-4) |
| Fork-shared module-level lists across goroutines | N12; required verification N21 | Immutability and safe publication are required, not established: permanent analyzer enforcement plus an LC-4 census of every publication path, its concrete happens-before edge, and a race test per path |
| Raw-slice escapes ×3 (`SafeAsList`, `embed.ToList`, `testctx.GetList`) | N13 | Migrate to materialize accessor in the package-owning lane: LC-3a / LC-3c / LC-3b respectively — escaped slices become private copies by construction |
| Symmetric List/Array/Tuple switch arms | N16 **(corrected iter-234)**: 15 non-test files w/ ListValue arms; **7** symmetric with Array **and/or Tuple** (was reported as 3 when Tuple was omitted from the intersection) | Arms diverge at LC-4. Lane ownership of the 7: **LC-3a** ×2 (`builtins/canonical_key.go`, `builtins/list.go`), **LC-3b** ×1 (`eval/eval_patterns.go`), **LC-3c** ×4 (`cmd/ailang/bytecode_bridge.go`, `embed/convert.go`, `testing/executor_helpers.go`, `testing/value_splice.go`). ⚠ The round-1 claim that they are "all in LC-3a/LC-3c clusters" is **refuted** by `eval_patterns.go`, which is LC-3b's — so all three migration lanes carry symmetric-switch work, not two. Statement-level census = LC-2 analyzer output |
| Indexed builtins (`_list_nth`, bounds checks via `len`) | N13 | O(i) post-swap; disposition + `ArrayValue` guidance in LC-5 |
| `std/list` builder family | N14/N15: 9 right-recursion builders; `reverse` sole left-append | Right-recursion goes linear via O(len(left)) concat; `reverse` needs the already-queued delegation (N19) |
| Bytecode VM + bridge | N11 | Decoupled (deep-copy bridge); `OpCons` divergence documented, non-goal |
| `ArrayValue`/`TupleValue` | N5 | Untouched — they keep `Elements []Value`; only `ListValue` changes |
| Compile-time `Elements` owners (types/ast/core/typedast/gen/smt/pipeline…) | N4/N6 | **Out of scope entirely** — the analyzer excludes them by type, and no piece may touch them |

**Programs that MUST still work:** the five fixtures named and existence-verified in the old doc
(V31) — `examples/first_non_repeat.ail`, `examples/inline_tests_recursive.ail`,
`examples/record_cons_pattern.ail`, `examples/record_list_extraction.ail`,
`examples/pattern_matching_adt.ail` — plus `make verify-examples` and the stdlib test surface;
byte-identical output is an AC of every migration piece and of LC-4.

**What deliberately changes:** allocation/memory behavior of `::`/`++`/pattern-tails (the point);
the three escape helpers return materialized copies (observable only to a caller mutating through
the escaped slice — the behavior being made impossible); nothing else — any other observable
change is a regression.

## Disposition of the superseded doc (recommendation — the controller enacts)

`design_docs/planned/m-list-cons-quadratic.md` should be **retitled in place, not deleted and not
moved**: Status header → "SUPERSEDED — D-19 ruled B (2026-08-19, #745); retained as the evidence
appendix for m-list-cons-cells-decomposition", with a pointer to this doc. Its 38-row Verification
Log is a genuine asset and most of it is measurement of HEAD, not arena advocacy:

- **Carry over as programme evidence (independent of the chosen fix):** V1–V4 (representation, no
  TCO), V7 (blast radius — now known to be a textual upper bound, N4), V8–V21 (repro, RSS curves,
  pprof attribution, stdlib shapes, reverse base cost — these are LC-4/LC-5's benchmark bases),
  V22–V26 (VM/bridge/Fork/escape/telemetry inventory — N10–N13 re-verified four of them this
  session), V27–V31 (test-surface and fixture existence), V32/V33 (zero aliased writes at HEAD —
  still relevant during the migration window), V34–V36 (escape census, builder classification,
  takebench base), V38 (`-race` base).
- **Do not carry:** V5/V6 (already retired in-doc), V37 (the old `listmut`-absence conclusion is
  superseded by LC-2's permanently retained `listrep` cell-assignment rule plus its temporary
  migration-count ratchet), and every arena-specific design section (seeding, bootstrap, CAS protocol,
  AC-9/AC-10 as written).
- Its Sprint-2 recommendation (`m-eval-tail-calls`) survives and is already queued (N19); its
  quorum artifacts under `design_docs/planned/quorum/` stay where they are — they are the recorded
  reasoning that produced D-19.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1 Determinism | 0 | No semantic change in any piece; outputs byte-identical (AC of every piece) |
| A2 Replayability | 0 | Traces unchanged (telemetry reads length only — inherited V26, re-check owned by LC-3b) |
| A3 Effect Legibility | 0 | No effect changes |
| A4 Explicit Authority | 0 | No capability changes |
| A5 Bounded Verification | +1 | Required immutability is mechanically gated by the permanently retained assignment analyzer; field deletion separately proves migration completeness |
| A6 Safe Concurrency | 0 | Immutability and safe publication are required, not established; LC-4 must complete N21's path→happens-before-edge→race-test matrix before claiming them |
| A7 Machines First | +1 | The canonical accumulator idiom models write stops being punished — under ALL sharing patterns, which is what B buys over A |
| A8 Minimal Syntax | 0 | No syntax |
| A9 Cost Visibility | +1 | `::` costs what every FP tradition implies; the new costs (`nth` O(i), per-cell memory) are documented, not hidden (LC-5) |
| A10 Composability | 0 | No composition changes |
| A11 Structured Failure | 0 | RT_REC_003 honesty is a separate queued row (N18), not claimed here |
| A12 System Boundary | 0 | Escape boundaries materialize; embedding API semantics preserved |

**Net Score: +3** → **Decision: Proceed** (hard-violation check: A1/A3/A4/A7 all non-negative).

## Unverified / needs measurement

Named gaps, each with an owner. A named gap is cheap; an assumed one is not.

| Claim | Status | Owner |
|---|---|---|
| True count of `eval.ListValue`-typed `.Elements` selector sites (902/522 are contaminated textual upper bounds, N4) | Unmeasurable by grep | LC-2 analyzer (first deliverable) |
| Statement-granularity symmetric-switch census (inherited "~15"; file-granularity partially reproduces, N16) | Unverified as stated | LC-2 analyzer |
| Per-element retained memory (~32 B/cell), iteration/nth/GC costs of candidate representations | Derivation only | LC-1 spike |
| Whether list-pattern compilation requires O(1) `length` (assumed yes — drives the cached-length field) | Assumed | LC-1/LC-2 (read `eval_patterns.go` match paths) |
| Telemetry reads list length only (inherited V26 [A✓]; corroborated by the 1-textual-ref count N6, not re-read) | Inherited | LC-3b |
| No identity comparison of `*ListValue` pointers anywhere (inherited inventory claim; sharing makes pointer identity newly observable if it exists) | Inherited, unverified | LC-4 (mandatory re-audit before swap) |
| The spike's (a)–(d) numeric bounds (2× iteration, 2.5× memory) | Provisional judgement | Ratified in LC-1's own doc + quorum |
| `opencode`/eval-harness workloads' sensitivity to `nth`/indexing cost | Unknown | LC-5 benchmark matrix |
| Whether every chosen cell representation can be confined behind the narrow constructor/read-only API and checked by the permanent assignment rule | Unestablished required property | LC-1 feasibility gate; LC-2 analyzer; LC-4 seeded enforcement checks |
| Complete census of shared-list publication paths and the concrete Go happens-before edge for each | Unverified; N12 establishes a concurrency constraint, not the complete publication mechanism | LC-4, recorded in N21 |
| Race safety of every enumerated shared-list publication path | Unverified required property | LC-4, one named `go test -race` target per N21 path |
| Whether the 3 known escapes are the ONLY `*eval.ListValue.Elements` escape paths (N13's greps are syntax-specific and cannot prove exhaustiveness — `gpt5-6-sol` round-2, upheld) | Unverified; "3 currently known syntax-matched escapes" is the honest wording until the audit runs | LC-2's type-aware analyzer (the same instrument N4 already makes authoritative over grep) |
| Statement-level symmetric-switch census across List/Array/Tuple (N16 is file-granularity only) | Unverified | LC-2 analyzer output |

## Related Documents

- [m-list-cons-quadratic.md](m-list-cons-quadratic.md) — the superseded arena design; its
  Verification Log is this programme's evidence base (see Disposition). Its "Options considered"
  argument **against** cons cells (blast radius, switch divergence, indexed-builtin costs) is
  treated here as the problem list the decomposition solves: blast radius → LC-2/LC-3 strangler +
  analyzer; switch divergence → measured at N16 and owned by LC-2/LC-4; indexed costs → LC-1
  measures, LC-5 disposes. Mark has ruled; the argument is a work plan now, not a veto.
- `design_docs/v1-mission.md` — D-19 ledger row; iteration-228 STATUS stamp (fullest narrative);
  queued rows `m-stdlib-reverse-delegates-to-builtin`, `m-rt-rec-003-advertises-nonexistent-option`,
  `m-eval-tail-calls` (N18/N19).
- Duplicate gate: see N20 — neural search unavailable this session (SimHash fallback, noise); the
  only overlapping doc is the superseded one above, and this doc exists because of it.

## References

- [#676](https://github.com/sunholo-data/ailang/issues/676) — the user-reported defect (email-parse M9)
- [#745](https://github.com/sunholo-data/ailang/issues/745) — the mission issue carrying Mark's `D-19 : B` ruling (N1)
- [design_docs/PROGRAM.md](../PROGRAM.md) §4 — routing lane: AILANG fix (argued in the superseded doc; unchanged by the ruling)

---

**Document created**: 2026-08-19 (mission iteration 229)
**Authored at**: `dedf3b91f` (= `origin/dev`), branch `docs/mission-iter229-decomposition`

---

## Quorum verification log (mission iteration 229)

**Round 1 — `2026-08-19T11:53:59Z` — SYNTHESIS: BLOCKED.** Artifact (untracked, `.gitignore:82`
excludes `.ailang/`): `.ailang/state/mission-quorum/m-list-cons-cells-decomposition-2026-08-19T11-53-59Z.json`.
Cost $0.0884 (22,190 in / 576 out tok). **`absent_reviewers` is EMPTY and both external reviewers
report `present=true`** — so this is a genuine 2-of-2 external block, not an N−1 degrade, and the
`#651` presentCount-inflation trap does not apply (the controller verdict is recorded separately
under `controller_in_session`).

| Reviewer | Verdict | Cost |
|---|---|---|
| `gpt5-6-sol` | **reject** | $0.0618 |
| `gemini-3-1-pro` | **reject** | $0.0266 |
| controller (in-session, not an API call) | pass | $0 |

### Blocking objection 1 — `gpt5-6-sol` (the safety premise)

> The core safety premise is false: deleting `ListValue.Elements` does not make cons cells
> immutable "by the Go compiler." Go has no immutable fields; code in the package owning the
> proposed cell type can still mutate `head`, `tail`, or cached length after publication.
> Therefore INV-1's concurrency justification and the claim that no permanent mutation enforcement
> is needed are unsupported.

**Catch:** LC-4 relies on compiler-enforced immutability to dismiss mutation protocols and
analyzers, yet neither the representation nor an enforcement mechanism has been designed or
verified. The separate claim that publication happens-before is carried by "whatever channel passes
the value" is also too vague to establish safe Fork sharing.

**Reviewer-authored `proposed_fix` (verbatim):** Replace the compiler-enforcement claim with an
explicit enforcement design: place cells behind a narrowly scoped package/API with unexported
representation and constructors/read-only accessors, add a `go/analysis` rule that rejects
assignments to cell fields outside constructor initialization, and retain that rule permanently
unless a mechanically enforced alternative is demonstrated. Add an LC-4 verification-log row
enumerating every publication path for shared lists and the corresponding Go happens-before edge
(mutex, channel, initialization-before-goroutine-start, etc.), with race tests covering each path.
Until those checks pass, describe immutability and safe publication as required properties, not
established facts.

**Controller assessment:** the objection is CORRECT on Go semantics and is not disputable —
unexporting a field confines mutation to the owning package, it does not eliminate it, and
`internal/eval` is precisely the package that would own both the cell type and the evaluator. It
disputes a stated JUSTIFICATION (why the `listrep` analyzer can be retired), **not the design
DIRECTION** (cons cells), and it arrives with a complete, concrete, reviewer-authored fix. Note
this is the same reviewer, and the same class of catch, as round 1 of the superseded
`m-list-cons-quadratic.md` — where its "the grep cannot see an aliased write" objection was also
correct about the instrument.

### Blocking objection 2 — `gemini-3-1-pro` (parallel-lane file overlap)

> The decomposition assigns overlapping files to parallel workstreams, breaking its own isolation
> constraints. LC-3c is instructed to migrate 'The three raw-slice escape sites (N13)' to the
> materialize accessor. However, N13 lists 'builtins/safe_cast.go' and
> 'effects/testctx/mock_context.go', which reside in packages explicitly owned by LC-3a
> ('internal/builtins') and LC-3b ('internal/effects'). Executing LC-3c in parallel with LC-3a/b in
> 'isolated worktrees' as dictated by the Rationale will guarantee merge conflicts and duplicated
> migration work.

**Reviewer-authored `proposed_fix` (verbatim):** Remove 'The three raw-slice escape sites (N13)
migrate to the materialize accessor here' from LC-3c's Scope. Instead, distribute the
materialize-accessor instruction to each package's rightful owner: instruct LC-3a to handle the
escape in 'safe_cast.go', LC-3b for 'mock_context.go', and leave only the 'embed/convert.go' escape
in LC-3c.

**Controller assessment:** purely mechanical file-ownership completeness — the doc contradicts its
own isolation constraint. No design direction is disputed and the fix is fully specified.

### Disposition — ROUND-1 REVISION APPLIED; RE-QUORUM RAN (see round 2 below)

Both objections are narrow-refinement shaped (concrete reviewer-authored fix, no dispute of the
design DIRECTION), so Gate 2's normal flow applies: **one designer revision, then one re-quorum.**
That revision was not run in iteration 229 because of a routing constraint rather than a design
one:

- The designer ROTATION's next entry, `codex:gpt-5.6-sol`, probed **rc=1** — *"You've hit your usage
  limit … try again at Aug 20th, 2026 5:34 AM"* — the same exhaustion iteration 228 measured.
- The entry after it (gemini / managed_agents) is **read-only under `CapRemoteSandbox`** and
  structurally cannot author a doc.
- So the rotation resolved to `claude:claude-fable-5` as a **fallback**, and the Fable discipline
  caps that lane at **ONE** bounded sub-agent run per iteration. Iteration 228 spent two (create +
  revision) and FLAGGED it; iteration 229 declined to repeat that.

The lane returned: `codex exec --model gpt-5.6-sol` probed **rc=0 at 2026-08-20 06:12 CEST** — the
`PARKED-ON-LANE` resume predicate, re-run as a command rather than transcribed (Standing rule 8(d)).
The document is no longer `PARKED-ON-LANE`; mission iteration 234 ran the codex designer, which
applied the two round-1 fixes below. The re-quorum then ran and **blocked again on two NEW
objections** — see *Quorum verification log — round 2* below, where both were measured and resolved
under the narrow-refinement carve-out.

### Round-1 revision applied (mission iteration 234)

**Objection 1 (`gpt5-6-sol`) — reviewer-authored fix applied.**

- Replaced the compiler-enforcement premise in **The invariant the programme buys** (lines 39–55)
  and clarified in **The decomposition** (lines 97–113) and LC-4 (lines 246–272) that deleting
  `Elements` proves migration completeness, not immutability.
- Added the prescribed narrowly scoped package/API, unexported representation,
  constructors/read-only accessors, and permanent `go/analysis` assignment rule to INV-1, LC-2,
  and LC-4; updated LC-2's AC and retained the rule unless a mechanically enforced alternative is
  demonstrated.
- Added N21 (line 95), the prescribed pending LC-4 verification row for the complete publication-path → Go
  happens-before-edge → race-test inventory. Updated LC-4's scope/AC, **Conflict Surface**, and A6
  so immutability and safe publication remain required properties rather than established facts
  until the checks pass.
- Repaired the resulting stale text in **Riskiest piece & the de-risking spike** (lines 315–327),
  **Disposition of the superseded doc** (lines 362–382), **Axiom Compliance** (lines 384–401), and
  **Unverified / needs measurement** (lines 403–419).
- **YOUR inference from the reviewer's mechanism:** extended LC-1's kill criterion with
  encapsulation/analyzer feasibility, while deliberately retaining LC-1's 2–3 d estimate because
  this fits its existing candidate/API spike. Increased LC-2 from 2–3 d to 3–4 d for the permanent
  rule and fixtures, and consequently updated the header and programme totals by one day.
- Deliberately did **not** alter D-19:B, the true-cons-cell direction, or claim the pending N21
  checks passed; neither is authorized by the objection.

**Objection 2 (`gemini-3-1-pro`) — reviewer-authored fix applied.**

- Removed the three-site instruction from LC-3c (lines 231–244). LC-3a (lines 199–212) now owns the
  `safe_cast.go` materialization, LC-3b (lines 214–229) owns `effects/testctx/mock_context.go`, and
  LC-3c retains only `embed/convert.go`.
- Updated each lane's AC to cover only its owned escape and repaired **Ordering & parallelism**
  (lines 286–313) and **Conflict Surface** (lines 329–360) to reflect the now-disjoint package
  ownership.
- **YOUR inference from the reviewer's ownership fix:** named unchanged-value-through-copy behavior
  separately in all three ACs so each owner has a local red/green obligation.
- Deliberately did **not** reorder the parallel lanes or change their representation/semantic scope;
  the objection required ownership repair only.

## Quorum verification log — round 2 (mission iteration 234)

**Round 2 — `2026-08-20T04:22:39Z` — SYNTHESIS: BLOCKED (rc=3).** Artifact (untracked, `.gitignore:82`
excludes `.ailang/`): `.ailang/state/mission-quorum/m-list-cons-cells-decomposition-2026-08-20T04-22-39Z.json`.
Cost **$0.1089** (28,040 in / 545 out tok). **`absent_reviewers` is EMPTY and both external reviewers
report `present=true`** — a genuine 2-of-2 external block, not an N−1 degrade, so the iteration-175
budget-degrade trap does not apply (the cap was raised to $0.25 pre-emptively because the doc grew).

| Reviewer | Verdict | Cost |
|---|---|---|
| `gpt5-6-sol` | **reject** | $0.0772 |
| `gemini-3-1-pro` | **reject** | $0.0317 |
| controller (in-session, not an API call) | pass | $0 |

### Disposition — NARROW-REFINEMENT CARVE-OUT (ratified iter-98), both premises MEASURED not forwarded

Both round-2 objections carry a concrete reviewer-authored `proposed_fix` and **neither disputes the
design DIRECTION** (cons cells / `D-19 : B`); both are premise/completeness objections. Gate 2's
carve-out therefore applies after the one re-quorum, and Gate 2 rule 3f applies first: *a reviewer's
objection is a claim too* — the controller MEASURES a premise objection rather than buying a revision
round to answer a question one command settles. Both were measured first-party at
`779746352`, and the two came out **opposite ways**.

**Objection 2 — `gemini-3-1-pro` — CONFIRMED, and materially worse than filed.**

> N16's verification command performed a `comm -12` intersection exclusively between ListValue and
> ArrayValue files, completely omitting TupleValue from the measurement. The premise that only 3
> files contain symmetric switches is unverified for Tuples.

Measured (N16, rewritten): including `TupleValue` takes the symmetric surface from **3 to 7**
non-test files (**8** including tests, which is N16's original scope) — more than a 2× under-count.
The consequence is structural, not cosmetic: the round-1 doc asserted the symmetric files are *"all
in LC-3a/LC-3c clusters"*, and one of the four newly-surfaced files, `internal/eval/eval_patterns.go`,
is **LC-3b's**. So all three migration lanes carry symmetric-switch work, not two, and the Conflict
Surface row said otherwise. The reviewer's `proposed_fix` is applied in full: N16's command now covers
all three constructors, the true overlap is recorded in both scopes with the files enumerated, and the
Conflict Surface row carries the corrected number plus per-lane ownership.

**Objection 1 — `gpt5-6-sol` — PARTIALLY REFUTED by measurement, which selects the reviewer's own
cheaper option and drops the expensive one.**

> …it neither verifies that no caller relies on aliasing nor establishes that aliasing is outside the
> APIs' contracts. Byte-identical fixture output cannot prove compatibility for these Go-facing APIs.

The reviewer is right that the doc had not verified it. The audit is now N22, and it refutes the
"Go-facing APIs" framing on which the expensive half of the fix rested: all three escapes live under
`internal/` in module `github.com/sunholo-data/ailang`, which the Go toolchain forbids importing from
outside this module; **every in-repo caller of all three is a test** (zero production callers); and
**no call site mutates the returned slice** (same-scope known-positive control firing). There is
therefore no consumer to version for, and the reviewer's `proposed_fix` option **(1)** — retain alias
semantics until LC-4, version/deprecate the API — is not taken. Option **(2)** is applied in full:
the returned slices are documented as snapshots, each owning lane gains a mutation-focused
compatibility test as an AC, and the copy is classified as an intentional semantic change rather than
smuggled under "zero behavior change" (LC-3a's scope paragraph no longer makes that claim
unqualified). The reviewer's wording fix is also applied verbatim: N13 now reads **"3 currently known
syntax-matched escapes"**, and the residual — that syntax-specific greps cannot prove exhaustiveness
— is recorded in *Unverified / needs measurement* against LC-2's type-aware analyzer, which N4 had
already made the authoritative instrument over grep.

**Controller-applied, not designer-applied**, per the carve-out's letter (the fixes are the
reviewers' own text, plus measurements that only narrow them). **No third quorum** is run: the
carve-out is one bounded revision after the one re-quorum, and re-running would be the
re-litigation Gate 2 forbids. Both fixes are recorded here and in the Gate-4 routing-evidence row.
**Residual carried honestly**: the exhaustive escape audit and the statement-level symmetric census
are LC-2 analyzer deliverables, named in *Unverified*, not claimed closed here.

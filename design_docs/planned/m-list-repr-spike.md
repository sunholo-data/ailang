# M-LIST-REPR-SPIKE: LC-1 — List-Representation Benchmark Spike (Go/No-Go Gate for Cons Cells)

**Status**: Planned — LC-1 of the quorum-cleared
[m-list-cons-cells-decomposition](m-list-cons-cells-decomposition.md) programme (cleared 2-of-2 at
mission iteration 234; this piece still gets its own quorum, per the roadmap's own rule).
**Target**: v0.34.0 (first programme piece; roadmap line 6 allows the first piece to land in v0.34.0)
**Priority**: P0 — carries the kill criterion for the whole ~16-day remainder of the programme
**Estimated**: 2–3 days (matches the roadmap; no silent absorption — see Timeline)
**Dependencies**: None. D-19 is RESOLVED (`D-19 : B`, roadmap N1). No code dependencies.
**Planner-Lane**: opus-required (roadmap line 10 declares `opus-required for LC-1/LC-4`; V15)

## What this doc is

The roadmap's LC-1 section (`m-list-cons-cells-decomposition.md:147-176`, V15) is the authoritative
scope statement: a **throwaway** benchmark spike, no production wiring, that implements 2–3
candidate list representations against the current slice as control, fills a measured benchmark
matrix, ratifies the programme's kill criterion, and emits an explicit **go/no-go** on the whole
cons-cells programme. This doc refines that scope into a routable sprint. It does **not**
re-litigate the programme, does not re-open `D-19 : B` (Mark's 2026-08-19 ruling), and does not
design LC-2's production accessor layer or `listrep` analyzer — LC-2 owns both.

Why this piece runs first: everything downstream (LC-2…LC-5, ~16 more person-days) is gated on
this spike passing. Spending ~3 days to earn the right to spend ~16 more is the entire point of
the ordering. The designed exit on failure is a STOP that re-opens D-19 **with measurements** —
which is different from arguing the ruling was wrong.

## Problem statement (inherited, one paragraph)

`::` copies the whole tail (`internal/builtins/list.go:98-103`, V4), so prepend-built lists are
Θ(n²) in time and allocation; the #676 repro peaks at 1,467.9 MB at n=12,800 with 94.97% of
allocation flat in `listConsImpl` (inherited [R] measurements at `88631976e`, V14). Mark ruled
`D-19 : B`: true cons cells / structural sharing, O(1) prepend under **all** sharing (INV-1). The
declined arena failed precisely under *branching* — m prepends onto one retained base degrade to
Θ(m·len(base)) — so that shape is this spike's load-bearing benchmark, not an afterthought. Before
any migration effort is spent, the spike must show a representation that actually delivers INV-1's
costs on this machine, at measured constants, behind a feasibly-narrow API.

## Deliverables

| # | Deliverable | Form |
|---|---|---|
| D1 | Scratch benchmark package `tools/internal/spike-listrep/` — candidates + control + full matrix | Go code on the sprint branch; every matrix cell = command + number |
| D2 | Ratified kill criterion + explicit **go/no-go verdict** with the arithmetic shown | Section of the implemented-doc report; on STOP, a comment on #745 re-opening D-19 with the table |
| D3 | Draft accessor API spec sized against real call-site needs (LC-2 builds the seam from it) | Table in the implemented-doc report (draft status explicit) |

## Solution Design

### Where the spike lives, and why that location

New package **`tools/internal/spike-listrep/`** (does not exist today, V11). Rationale:

- **Compiler-enforced production boundary.** Go's nested-`internal` rule permits imports only from
  code rooted at the parent of that `internal/`: placing the spike below `tools/internal/` lets
  packages below `tools/` use it while mechanically forbidding production packages below `cmd/` or
  the repo-root `internal/` from importing it (V16). This is stronger than a visual or README-only
  boundary. It also remains outside the `check-file-sizes` CI gate, whose scan is exactly `find
  internal cmd -name "*.go"` and therefore does not traverse `tools/internal/`
  (`make/code-health.mk:122-128`, V10). Files stay <500 lines anyway per repo hygiene.
- `tools/` already hosts standalone Go programs (`tools/eval-elo/main.go`,
  `tools/govulncheck-filter/main.go`, V11), so a Go package here is established practice.
- The package **imports `internal/eval` read-only** (legal because the parent of the repo-root
  `internal/` is the repository root, which contains `tools/`; V16) so the control arm is the
  literal production type `*eval.ListValue` and candidates hold real `eval.Value` elements —
  faithful 16-byte interface headers in the memory accounting. Importing is not modifying; the
  anti-goal ("do not touch `internal/eval`") is enforced by an AC that `git diff` against the merge
  base shows zero changes under `internal/` and `cmd/` (AC-7).
- The slice-cons control mirrors `listConsImpl`'s copy locally (the 5-line
  `make(…, 0, 1+len(tail))` + two appends, `list.go:98-105`, V4) rather than calling the builtin,
  avoiding `EffContext` plumbing. The mirror is quoted next to the original in the spike's README
  so drift is visible.

**Disposition (refinement over the roadmap, with reason).** The roadmap says "throwaway … scratch
package or worktree branch". This doc keeps the package merged on `dev`, README-marked
`THROWAWAY — DO NOT IMPORT`, because LC-4's representation swap needs a before/after instrument
and this harness *is* that instrument at the Go level; deleting it makes the spike's numbers
irreproducible. Deletion (or retention as a permanent regression harness) is decided in LC-4's
doc. Cost of keeping it: `make test` compiles and unit-tests it in CI — see Conflict Surface.

### Candidates

Per the roadmap, 2 candidates + control. No third candidate: neither quorum objection demands one,
a third costs ~0.5–1 day against a 2–3-day box, and if both candidates fail the kill criterion
fires regardless. The chunk-size sweep below gives the exploratory breadth a third candidate would
have bought.

- **C0 — control: current slice.** Literally `*eval.ListValue` (`internal/eval/value.go:84-86`,
  V2) with the mirrored copy-on-cons. Baseline for every ratio in the kill criterion.
- **C1 — plain cons cell with cached length.**
  `type cell struct { head eval.Value; tail *cell; n int }` — 16 B interface header + 8 B pointer +
  8 B int = 32 B/cell before allocator rounding (32 B is an exact Go size class; B6 measures rather
  than trusts this). Prepend = one cell allocation, O(1) under all sharing by construction. `nth` =
  O(i); iteration = pointer-chasing (the locality cost B3 exists to price).
- **C2 — chunked/unrolled cons, chunk capacity K, swept at K ∈ {8, 32}.** Nodes hold up to K
  elements plus a tail pointer and cached total length; elements fill leftward so prepend into a
  chunk with front slack is a slot write. When the front slot is contended (a second list sharing
  the chunk prepends) or full, the prepender copies **at most one chunk** (≤ K elements) into a
  fresh node — O(K) = O(1) worst-case with a constant, recovering slice-like locality within
  chunks and amortizing per-element overhead across the chunk. The K sweep is a parameter study of
  C2, not a separate candidate; the report records both columns and names which K (if either)
  passes.

All candidates are implemented with **unexported representation** behind exported
constructors/read-only accessors, and the benchmarks live in an external `package spikelistrep_test`
that can only use the public API — this is the kill criterion's clause (e) feasibility check,
folded in as designed (see below), not a production analyzer.

### Representation/API overlap and reuse decision

Added at round 2 on `gpt5-6-sol`'s objection: the Conflict Surface analysed *directory* boundaries
and CI effects but never asked **what machinery this proposal overlaps**, so building C1/C2 and the
D3 API from scratch rested on an unverified absence. The inventory below was measured by the
controller at `8322d22b7` (V17–V20); each row states whether C0/C1/C2 can reuse the mechanism and,
where not, the concrete incompatibility.

| Mechanism inspected | Found? | Reusable for C0/C1/C2? | Concrete reason if not |
|---|---|---|---|
| Methods on `*ListValue` (pointer **and** value receivers, any receiver name, repo-wide) | 2 — `Type()`, `String()` (V3) | No | Both are display/tagging, not access: neither exposes length, indexing, iteration, or construction. Nothing to preserve, nothing to extend. |
| Free functions over `*ListValue` | 2 — `encodeJSONArray`, `encodeJSONObject` (V17) | No | They *consume* `.Elements` to emit JSON. They are migration **sites** owned by LC-3x, not an accessor layer; reusing them would invert the dependency (encoder → representation). |
| Persistent / copy-on-write / structural-sharing sequence implementations | **0** (V19) | n/a | None exists. This is the gap the programme is for. |
| Finger trees / ropes / existing cons cells | **0** (V19) | n/a | None exists. |
| Generic collection types in-repo | 2, neither a sequence (V20) | No | `ttlCache[T any]` is a TTL map; `callbackResult[T any]` is a one-shot result box. Neither has sequence semantics. |
| Iterator machinery (`iter.Seq`, range-over-func) | **0 in-repo uses**; stdlib available at `go 1.26.6` (V20) | **Yes — adopt, do not invent** | D3's iteration accessor uses stdlib `iter.Seq`. The spike would be its first in-repo use, so the D3 spec must say so explicitly rather than implying an existing house idiom. |
| Module dependencies offering a collection/persistent data structure | **0 of 99 direct requires** (V20) | n/a | The dependency set is GCP, OTel, LSP, sqlite, MCP, ollama, testify. Adding one would be a dependency-policy decision this throwaway spike has no standing to make. |

**Decision.** C1 and C2 are built from scratch because nothing reusable exists — measured, not
assumed. The single exception runs the other way: iteration is expressed with the **stdlib**
`iter.Seq`, not a bespoke iterator, even though the repo has no precedent for it. The two free
functions found are recorded as LC-3x migration sites so that a later lane does not rediscover them.

### Benchmark matrix

Columns: C0, C1, C2(K=8), C2(K=32). Every cell in the report = the `go test -bench` command and
the number it printed; a cell asserted from theory is a red AC. Rows:

| Row | Benchmark | Design | Metric |
|---|---|---|---|
| B1 | **Prepend under branching** (load-bearing: the shape that killed the arena) | m prepends each onto ONE retained base of length L; all m results kept live (`runtime.KeepAlive` on the result slice). Grid: m ∈ {1024, 4096} × L ∈ {1024, 4096, 16384} | ns/op and allocs/op per grid point |
| B2 | Linear build (#676 `gen` shape, V14) | fold n prepends, each onto the newest list; n ∈ {1600, 3200, 6400, 12800} (mirrors the repro ladder) | total ns, total bytes allocated |
| B3 | Iteration/sum throughput | sum n int elements via each representation's iterator; n ∈ {4096, 65536} (small = in-cache, large = locality-sensitive) | ns/element |
| B4 | `nth` sweep | index i ∈ {0, n/4, n/2, n−1} at n=4096 | ns/op per i (informs LC-5's `_list_nth` disposition, `list.go:278` V6 — measured, not gated) |
| B5 | Materialize-to-slice | full copy-out at n=4096 | ns/op, allocs/op |
| B6 | Per-element retained bytes | dedicated non-`testing.B` subprocess builds n=100,000 list of ONE shared singleton element (so element cost cancels and structure cost remains); in each fresh process, force `runtime.GC()` twice, measure a same-process empty-workload baseline, build + `KeepAlive` the list, force GC twice again, and report the baseline-adjusted retained-heap delta ÷ n | B/element, incl. measured C0 baseline (the 16 B figure is verified by measurement here, not assumed — it is currently a derivation the superseded doc matched to 3.2%, V14) |
| B7 | Allocation count | read off B1/B2 `-benchmem` allocs/op | allocs/op |
| B8 | GC behaviour, #676-repro-shaped | B2's n=12,800 workload once per candidate with `runtime.ReadMemStats` immediately before and after: deltas for `NumGC`, `PauseTotalNs`, and endpoint `HeapAlloc` (before and after, not a claimed peak) | counts + bytes |

**Methodology.** Go benchmarks are established here — 22 files under `internal/` define
`^func Benchmark` (scope: `--include='*_test.go'`, V8), including
`internal/builtins/xml_fold_bench_test.go` and
`internal/pipeline/validate_coretypeinfo_bench_test.go`. The spike deliberately does not inherit
the `bench-phase2a` target's three-replicate, single-command convention: the kill gate needs the
fresh-process five-trial protocol below. The `BenchmarkListRep_` prefix cannot collide with that
target's `Benchmark(Native|Eval)_` regex, which is additionally scoped to `./internal/eval/`.
Run on the quiet recorded rig (darwin/arm64), and record `go version` plus machine identity in the
report header. No new make target: every invocation and result is recorded verbatim in the matrix.

### Mandatory measurement and adjudication protocol

This protocol sits underneath every numeric observable in clauses (a)–(d); it does not replace or
weaken any threshold or C0's known-positive control.

1. Execute each benchmark cell in a **fresh process for five trials** on the recorded rig. Each Go
   benchmark invocation uses `-count=1`, an explicit `-timeout=10m`, and exactly one selected cell;
   the runner also imposes a 12-minute subprocess deadline so a wedged command is killed and the
   trial is reported as invalid rather than silently omitted. B6 uses its dedicated non-`testing.B`
   measurement subprocess with the same explicit deadlines. B8 uses five fresh processes as well.
2. Pair candidate and C0 trials by ordinal on the same rig (`candidate_i / control_i`), and compute
   every threshold ratio from the **median of the five paired ratios**. For clause (a)'s within-arm
   L-ratio and clause (d)'s within-arm n-ratio, pair the two sizes by ordinal in the same way. Report
   all raw operands, all five paired ratios, their sorted order, and the median arithmetic.
3. A five-trial median strictly inside the permitted side of its threshold is ordinarily final.
   If the median is **exactly** on a threshold, or the five individual paired ratios span the
   threshold (values occur on both sides, with equality treated as touching both), the result is
   **STOP pending the predeclared rerun**: run five additional fresh-process trials under the same
   protocol. The median of all ten paired ratios is then final; equality passes a `<=` clause and
   passes a `>=` control clause, exactly as their written operators specify. No discretionary
   reruns, dropped trials, or alternate aggregation are allowed.
4. B6's hard operand is the median baseline-adjusted retained-heap delta across five fresh
   processes for each arm, paired candidate/control by ordinal before applying clause (c). Its raw
   empty baseline, post-workload counters, adjusted delta, and B/element are all reported. The same
   threshold-touch/crossing rule triggers five more fresh processes and a final combined median.
5. B8 reports only defined before/after `MemStats` counters. Endpoint snapshots cannot establish a
   peak, so the unsupported peak-`HeapAlloc` claim is removed rather than adding a sampler whose
   timing and implementation would enlarge this throwaway spike.

### The kill criterion — ratified here, as the roadmap requires

The roadmap marks the thresholds **provisional judgement, "ratified in the spike's own doc +
quorum"** (`m-list-cons-cells-decomposition.md:431`, V15). This section is that ratification: each
clause is adopted or adjusted **with a stated rationale**, and each is operationalized as a
measurable observable that can fail. If **no candidate simultaneously satisfies (a)–(e)**, the
programme **STOPS**: LC-2…LC-5 do not run, and D-19 is re-opened on #745 with the measured table
and a case for either the chunked hybrid at relaxed bounds or a rescoped answer A.

- **(a) O(1) worst-case prepend under sharing — ADOPTED unchanged.** This is INV-1 verbatim; it is
  the ruling, not a tunable. *Observable (B1):* for the candidate, at fixed m,
  time(m, L=16384) ÷ time(m, L=1024) ≤ **1.5**, at both m points (a shape claim needs at least two
  points and a ratio); the C0 control must show ≥ **8×** on the same ratio (expected ~16×,
  L-proportional), proving the instrument can see the failure it gates on. Fails if the candidate's
  ratio exceeds 1.5 — i.e. asserting O(m) from theory cannot pass this.
- **(b) Iteration throughput within ~2× of slice — ADOPTED at 2.0×, with rationale.** Iteration
  is the dominant list access pattern in the builtin family (`map`/`fold`/`show` all walk the
  spine). 2× at the Go microbench level is deliberately **conservative in the safe direction**:
  the tree-walking evaluator's per-element interpretation overhead dilutes representation-level
  slowdown end-to-end, so a Go-level 2× bound can only produce false STOPs, never false GOs — the
  correct bias for a gate whose false-GO costs 16 days. *Observable (B3):* candidate ns/element ÷
  C0 ns/element ≤ 2.0 at **both** n=4096 and n=65536 (the large size is where pointer-chasing
  actually hurts; passing only in-cache would be a false pass).
- **(c) Per-element retained memory within 2.5× of slice — ADOPTED, denominated in the measured
  baseline.** Refinement: the ratio uses B6's **measured** C0 B/element, not the assumed 16 B
  (if the measurement disagrees with the derivation, the measurement wins; the superseded doc's
  16 B derivation matched observed RSS to 3.2%, V14, so ~40 B/element is the expected ceiling).
  Rationale for 2.5×: #676 is a memory defect (1.47 GB peak, real-pipeline OOM at 10.5 GB); the fix
  must not trade quadratic *transient* allocation for a large *permanent* residency multiplier.
  2.5× admits C1's ~32 B/cell with margin for allocator rounding and rejects fat representations;
  even at the 2.5× ceiling the repro's retained memory at n=12,800 is ~0.5 MB vs the current
  quadratic ~1,250 MB allocation volume. *Observable (B6):* candidate B/element ÷ C0 B/element
  ≤ 2.5.
- **(d) O(1) `length` — ADOPTED, and upgraded from "assumed" to verified-needed.** The roadmap
  carried "whether list-pattern compilation requires O(1) length" as an *assumption* (line 428,
  owner LC-1/LC-2). Settled first-party this session: exact-length list patterns compare full
  runtime length — `len(p.Elements) != len(listVal.Elements)` at `internal/eval/eval_patterns.go:218`,
  and `[a, ...rest]` patterns bounds-check at `:238` (V5). Under a cons representation without
  cached length, every exact-pattern match attempt becomes O(n). So (d) stays a hard clause.
  *Observable:* `Len()` at n=4096 and n=65536 within measurement noise of each other (ratio ≤ 1.2);
  an O(n) length shows ~16×.
- **(e) Encapsulation feasibility — ADOPTED with its scope pinned.** The candidate must live
  behind a narrow constructor/read-only API with **mechanically distinguishable constructor
  initialization** — i.e. it must be possible for LC-2's analyzer to tell "field write inside a
  constructor" from "field write anywhere else". *Observable, folded into the spike as the roadmap
  requires (its Estimate clause, lines 174-176):* (1) representation structs have unexported
  fields; (2) ALL benchmarks compile in an external `_test` package that can only reach the public
  API — if any benchmark needs a raw field, the API was insufficient and this clause **fails**;
  (3) all field writes sit inside constructor functions in one file, recorded by a grep in the
  report (a syntax-level record for LC-2's type-aware analyzer to supersede — NOT the analyzer
  itself, which stays LC-2's deliverable so the estimate holds).

**Verdict procedure (D2).** A per-candidate pass/fail table over (a)–(e) with the arithmetic
inline. ≥1 candidate passes all five → **GO**, naming the chosen representation (ties broken by
(c) then (b)). Zero candidates pass → **STOP**, comment posted on #745 with the full matrix,
explicitly re-opening D-19 as the roadmap designed. The verdict may not be "partial go".

### Draft accessor API spec (D3) — input to LC-2, not a commitment

Sized against verified call-site needs; every row cites the call-site class that demands it. LC-2's
own doc decides the final surface — divergence there is expected and cheap, duplication here is
not, so this stays a table, not an implementation.

| Operation (draft) | Complexity slice → cons | Call-site justification |
|---|---|---|
| `Len() int` | O(1) → O(1) (cached) | exact-length pattern match `eval_patterns.go:218` (V5); `[x, ...rest]` bounds check `:238` |
| `At(i int) (eval.Value, bool)` | O(1) → O(i) | `_list_nth` builtin, bounds-checked at `list.go:320` (V6, body read — not merely the registration site); indexed head-element access in patterns at `eval_patterns.go:224` and `:244` (V18) |
| `All() iter.Seq[eval.Value]` | O(n) → O(n) | every spine walk (`String()` at `value.go:89-99` V3, map/fold family); range-over-func available at `go 1.26.6` (V7) |
| `ToSlice() []eval.Value` | O(n) copy → O(n) copy, **documented** | escape/materialize sites (roadmap N13's three known escapes); embedding conversions |
| `FromSlice([]eval.Value)`, `Empty()` | O(n) / O(1) | literal construction, builtin results (386 `ListValue{` construction sites at roadmap N3 scope — LC-3's problem, but the constructors must exist) |
| `Cons(head, tail)` | O(n) today → O(1) | `::` (`listConsImpl`, V4) — the point of the programme |
| `Uncons() (head eval.Value, tail List, ok bool)` | O(1) → O(1) | pattern-match tail extraction is an O(1) alias today and must not regress (`eval_patterns.go:255-256`, V5; roadmap N10) |
| `DropPrefix(k int) List` | O(1) alias → O(k) | multi-element `[a, b, ...rest]` patterns take a tail at offset `len(p.Elements)` (`eval_patterns.go:255`, V5) |

Draft-status caveats recorded for LC-2: whether `At` returns `(Value, bool)` or errors; whether
`DropPrefix` folds into `Uncons` iteration; naming. The spike's own candidates implement exactly
this surface (that is clause (e)'s test), so infeasibility surfaces here, not in LC-2.

## Conflict Surface

LC-1 touches **no production code** — but per the hard gate, here is *why*, and what would change it:

1. **Files written:** only `tools/internal/spike-listrep/**` (new directory, V11) and this doc's
   implemented-report update. AC-7 makes this checkable: `git diff --name-only <merge-base>..HEAD
   -- internal/ cmd/ std/ examples/` must print nothing.
2. **CI does compile and unit-test the spike.** `make test` runs `go test` over
   `$(go list ./...)` excluding only `/scripts` and `/examples/agents`
   (`make/test.mk:27-30`, V9) — so the new package is built, vetted, and its (minimal) unit tests
   run in CI from the merge onward. Benchmarks themselves do **not** execute under plain `go test`
   (no `-bench` flag in the target, V9) — they only compile. This is the one real surface: a
   broken spike test reds CI for everyone. Mitigation: unit tests are smoke-level (each candidate
   round-trips a small list), and the package imports nothing that moves.
3. **Import direction:** `tools/internal/spike-listrep` → `internal/eval` (read-only) remains legal
   because the repo-root `internal/` admits all importers rooted at the repository. The reverse
   production dependency is **enforced mechanically by the Go compiler**: placing the package under
   `tools/internal/` guarantees that production code in `cmd/` or the repo-root `internal/` is
   structurally forbidden from importing it, replacing social README enforcement with a compiler
   gate (negative arm + positive control in V16). The README still marks its throwaway intent.
4. **What would change the answer:** if GC measurement (B8) were deemed to require an in-process
   evaluator hook — running real AILANG through `internal/eval` with instrumentation — that would
   put `internal/eval` in scope and change this piece's risk class. **Explicitly rejected:** B8
   measures GC at the Go level on the repro *shape*; the end-to-end AILANG-level before/after is
   LC-4's acceptance work (the roadmap's LC-4 owns the real `repro.ail` re-measurement), where this
   harness serves as the Go-level control.
5. **No parser/typechecker/codegen surface** — no syntax, no semantics, no `.ail` files.

## Example (sketch, not implementation)

```go
// tools/internal/spike-listrep/conslist.go — C1 sketch. Fields UNEXPORTED (kill-criterion clause e).
type ConsList struct{ head eval.Value; tail *ConsList; n int }

func Cons(h eval.Value, t *ConsList) *ConsList { // all field writes live in constructors
    return &ConsList{head: h, tail: t, n: t.Len() + 1}
}
func (l *ConsList) Len() int { if l == nil { return 0 }; return l.n }
```

```go
// tools/internal/spike-listrep/branching_bench_test.go — B1 shape (external test package: API-only access).
// m prepends onto ONE retained base; slice control is Θ(m·L), a passing candidate is O(m).
func BenchmarkListRep_BranchingPrepend_Cons_m4096_L16384(b *testing.B) { … }
```

Sample of the report's matrix form (numbers are what make each cell an AC, per roadmap AC shape):

| B1 branching, m=4096 | L=1024 | L=16384 | ratio | verdict vs (a) |
|---|---|---|---|---|
| C0 slice | _ns/op + cmd_ | _ns/op + cmd_ | expect ~16× | control fires |
| C1 cons | _ns/op + cmd_ | _ns/op + cmd_ | must be ≤1.5× | pass/fail |

## Success Criteria

Each is falsifiable; the refuting observation is named.

- [ ] **AC-1 (matrix complete):** every cell of B1–B8 × {C0, C1, C2(8), C2(32)} filled with the
  benchmark command and its printed number. *Red if:* any cell is empty, theoretical, or lacks its
  command.
- [ ] **AC-2 (branching shape, kill (a)):** candidate L-ratio ≤ 1.5 at both m ∈ {1024, 4096}
  AND C0 control L-ratio ≥ 8 on the same grid. *Red if:* candidate ratio > 1.5, or the control
  fails to show the quadratic blowup (instrument can't see a positive → B1 is invalid).
- [ ] **AC-3 (kill (b)/(c)/(d) arithmetic):** the report shows the three ratios with both operands
  measured (B3 at both n; B6 with measured C0 baseline; Len at both n), using five fresh-process
  trials, ordinal pairing, the median of paired ratios, and the predeclared five-trial tie/spread
  rerun exactly as specified. B6 additionally reports every same-process empty baseline and
  baseline-adjusted retained-heap delta. *Red if:* any ratio uses an assumed operand (e.g. the 16 B
  derivation instead of B6's measurement), a different aggregation, or an unreported/dropped trial.
- [ ] **AC-4 (kill (e) folded check):** all benchmarks compile in the external `_test` package;
  representation fields unexported; the report quotes the grep showing field writes confined to
  constructor files. *Red if:* any benchmark needed a raw field or an internal test package.
- [ ] **AC-5 (verdict):** a single explicit GO (with chosen representation) or STOP (with the #745
  comment URL) — the arithmetic for every candidate against every clause is in the table, derived
  only by the fixed five-trial aggregation and mandatory five-more tie/spread procedure, with
  explicit command timeouts. *Red if:* verdict is hedged, partial, uses discretionary adjudication,
  omits raw trials/median arithmetic, or a STOP lacks the posted D-19 re-open comment.
- [ ] **AC-6 (API draft):** D3's table present with a call-site citation per row. *Red if:* any
  row lacks a `file:line` justification.
- [ ] **AC-7 (anti-goal proof):** `git diff --name-only <merge-base>..HEAD -- internal/ cmd/ std/
  examples/` prints nothing. This remains correct after relocation: `tools/internal/` is not the
  repo-root `internal/` selected by that pathspec. *Red if:* any production path appears.
- [ ] **AC-8 (CI):** `make test` green locally with the spike package included. *Red if:* the new
  package breaks build/vet/test.

## Timeline

Within the roadmap's 2–3 days. The fresh-process runner, B6 baseline protocol, and deterministic
adjudication/reporting are D2 work on Day 3 and replace the former three-replicate procedure; they
do not add a candidate or production component. Thus they consume the existing Day-3 measurement
and report budget rather than pushing the estimate. Clause (e) remains folded into candidate
implementation (per roadmap lines 174-176), and D3 remains a table, not code.

- **Day 1:** package scaffold + C0 mirror + C1; B1/B2 running end-to-end with the C0 control
  showing the quadratic positive (AC-2's control leg — proves the instrument before trusting it).
- **Day 2:** C2 with K sweep; fill B1–B5, B7; start B6/B8 (memory runs are long-tail).
- **Day 3:** finish five-fresh-process B6/B8 runs and any predeclared tie/spread reruns; emit raw
  trials, paired medians, kill-criterion arithmetic + verdict; D3 API table; report writeup.
  On STOP: draft the #745 comment.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|---|---|
| A1 Determinism | +1 | No language semantics touched; the programme gate itself has fixed trials, aggregation, tie handling, and timeouts |
| A2 Replayability | 0 | No trace changes |
| A3 Effect Legibility | 0 | No effects |
| A4 Explicit Authority | 0 | No capabilities |
| A5 Bounded Verification | +1 | Converts the programme's provisional judgement into a mechanically checkable gate (five observables, each with a refuting value) |
| A6 Safe Concurrency | 0 | Concurrency claims stay LC-4's (roadmap N21); the spike makes none |
| A7 Machines First | 0 | Indirect only (the programme scores this; the spike itself ships no model-facing change) |
| A8 Minimal Syntax | 0 | No syntax |
| A9 Cost Visibility | +2 | The entire deliverable is turning derived/assumed costs (32 B/cell, iteration penalty, GC shape) into measured ones |
| A10 Composability | 0 | None |
| A11 Structured Failure | 0 | None |
| A12 System Boundary | 0 | Spike is outside the boundary by construction (AC-7) |

**Net: +4** → Proceed (hard-violation check A1/A3/A4/A7: all non-negative).

## Verification Log

All rows first-party this session in the iteration worktree at
`8322d22b7` (= `origin/dev`; `git log -1 --format='%H %D'` →
`8322d22b75adfce7a4aa284eaf3ad99afdd4b570 HEAD -> sprint/iter236-list-repr-spike, origin/dev, origin/HEAD, dev`;
`git status --short` clean), except V14, which is inherited and labeled so. Scope travels with
every count.

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 | Tree state as above | `git log -1 --format='%H %D'`; `git status --short` | as quoted above; status empty |
| V2 | `ListValue` is a bare slice wrapper | `grep -rn "type ListValue struct" internal/` | `internal/eval/value.go:84:type ListValue struct {` — body `Elements []Value` (read, lines 84-86) |
| V3 | `*ListValue` has exactly 2 METHODS repo-wide (pointer and value receivers, any receiver name); there is no existing *method* accessor surface to preserve. **Narrowed at round 2** per `gpt5-6-sol`: the original method-only grep could not support the broader "no accessor machinery anywhere" claim — see V17 for the free-function result and V18–V20 for the reuse inventory | `grep -rnE 'func \([A-Za-z_][A-Za-z0-9_]* \*ListValue\)' --include='*.go' .` and the value-receiver twin `func \([A-Za-z_][A-Za-z0-9_]* ListValue\)`; control `... \*ArrayValue` | pointer **2** (`Type()` :88, `String()` :89), value **0**, repo-wide (not just `internal/eval/`); control **4** ArrayValue pointer methods, so the instrument sees positives at the same scope |
| V4 | `::` copies the whole tail (the defect + the mirror the spike copies) | read `internal/builtins/list.go:87-107` | `result := make([]eval.Value, 0, 1+len(tail.Elements))` + `append(result, head)` + `append(result, tail.Elements...)` |
| V5 | Pattern matching needs runtime length (exact patterns) and O(1) tail alias (settles roadmap line-428 assumption) | `grep -n 'len(listVal.Elements)\|len(p.Elements)' internal/eval/eval_patterns.go`; read `:215-256` | `:218` `len(p.Elements) != len(listVal.Elements)` (exact match), `:238` `<` bounds check, `:255` `tailElements := listVal.Elements[len(p.Elements):]` O(1) alias |
| V6 | `_list_nth` is a live builtin AND is genuinely bounds-checked (drives B4 + D3's `At`). **Expanded at round 2** per `gemini-3-1-pro`: the round-1 command captured only the name and the registration guard, never the checking logic it was cited for | `grep -n '_list_nth' internal/builtins/list.go`, then read the impl body `sed -n '305,325p' internal/builtins/list.go` | `:278` `Name: "_list_nth"`, `:298` registration panic guard, and the checking logic itself: `:312` non-List arg refused, `:316` non-Int index refused, `:320` `_list_nth: index %d out of bounds for list of length %d` — an explicit bounds refusal, so "bounds-checked" is now supported by the log rather than asserted |
| V7 | Range-over-func (`iter.Seq`) available | `head -4 go.mod` | `module github.com/sunholo-data/ailang` … `go 1.26.6` (≥ 1.23) |
| V8 | `go test -bench` is established here: **22 files under `internal/`** define benchmarks (scope: `--include='*_test.go'`); named neighbours exist; a make target uses a historical three-replicate convention that this kill gate explicitly does **not** adopt | `grep -rln '^func Benchmark' internal/ --include='*_test.go' \| wc -l`; `ls internal/builtins/xml_fold_bench_test.go internal/pipeline/validate_coretypeinfo_bench_test.go`; read `make/eval.mk:163-175` | **22**; both files exist; `bench-phase2a` uses three repetitions in one benchmark command; LC-1 instead mandates five separate `-count=1` fresh-process invocations |
| V9 | CI compiles + unit-tests any new module package; benchmarks don't execute there | read `make/test.mk:27-30` | `test:` runs `$(GOTEST) -v $$($(GOCMD) list ./... \| grep -v /scripts \| grep -v /examples/agents)` — no `-bench` flag; only `/scripts` and `/examples/agents` excluded |
| V10 | `check-file-sizes` does NOT scan `tools/` (negative claim; instrument = the gate's own definition) | read `make/code-health.mk:122-138` | loop is `for file in $$(find internal cmd -name "*.go")` — the enumeration names `internal` and `cmd` only; gate exits 1 on breach (it can fire) |
| V11 | `tools/internal/spike-listrep/` does not exist; `tools/` hosts Go packages (negative + same-scope control) | `ls tools/internal/spike-listrep`; control `ls -d tools/eval-elo`; `find tools -name '*.go' \| head` | `No such file or directory`; control exists; `tools/eval-elo/main.go`, `tools/govulncheck-filter/main.go`, … |
| V12 | Real element payload types exist for benchmarks | `grep -n 'type IntValue\|type StringValue' internal/eval/value.go` | `:16 type IntValue struct`, `:39 type StringValue struct` |
| V13 | Duplicate/coverage gate | `ailang docs search "list representation benchmark spike cons cells" --neural` | Neural unavailable — CLI fell back (header: `🔍 SimHash search`, 1410 docs scanned); top hits are keyword noise (e.g. `m-match-xcheck-error-quality-sprint-plan.md` at 1.00, unrelated). The genuinely related docs are the parent roadmap and the superseded `m-list-cons-quadratic.md`, both in Related Documents. Same fallback the parent recorded at N20 |
| V14 | **Inherited [R], not re-run:** #676 repro shape + measurements (Θ(n²), 1,467.9 MB at n=12,800, 94.97% alloc in `listConsImpl`, 16 B/element prediction within 3.2% of observed) | `grep -n '16 B\|repro' design_docs/planned/m-list-cons-quadratic.md`; read `:55-95` | repro program quoted (prepend-fold `gen`), measurement table V12/V13/V21 at `88631976e` — the spike's B2/B8 sizes mirror this ladder; the 16 B figure is a derivation there, which is exactly why B6 measures it |
| V15 | Roadmap: LC-1 scope, provisional-threshold marker, planner lane | read `m-list-cons-cells-decomposition.md:147-176`; `:431`; `:10` | LC-1 section as summarized; line 431: "The spike's (a)–(d) numeric bounds (2× iteration, 2.5× memory) \| Provisional judgement \| Ratified in LC-1's own doc + quorum"; line 10: `**Planner-Lane**: opus-required for LC-1/LC-4` |
| V16 | A nested `tools/internal/` package mechanically rejects production importers while accepting importers below `tools/`; measured by the controller this session at `8322d22b7` (scratch packages removed afterward) | negative: create `tools/internal/spikeprobe/probe.go` and `internal/spikeprobe_consumer/consumer.go` importing it, then `go build ./internal/spikeprobe_consumer/` with rc captured directly; positive: create `tools/scratchprobe_ok/ok.go` importing the same package, then `go build ./tools/scratchprobe_ok/` with rc captured directly | negative `rc=1`: `internal/spikeprobe_consumer/consumer.go:3:8: use of internal package github.com/sunholo-data/ailang/tools/internal/spikeprobe not allowed`; positive `rc=0`, zero bytes output. Confirms the production-import prohibition and that packages rooted below `tools/` may import it |
| V17 | Free functions (not methods) operating on `*ListValue` DO exist — 2 of them — which is the surface `gpt5-6-sol` correctly said a method-only grep excludes. Neither is an accessor | `grep -rnE '^func [A-Za-z_][A-Za-z0-9_]*\(.*\*ListValue' --include='*.go' internal/` | **2**: `internal/eval/builtins_json.go:193 encodeJSONArray(list *ListValue)`, `:220 encodeJSONObject(list *ListValue)`. Both are JSON encoders that walk `.Elements`; neither offers length, indexing, iteration or construction, so they are migration SITES for LC-3x, not reusable accessor machinery |
| V18 | Patterns really do perform INDEXED head-element access (the D3 `At` justification `gemini-3-1-pro` flagged as absent from the log) | `grep -nE 'Elements\[[0-9a-zA-Z_]+\]' internal/eval/eval_patterns.go`; control `grep -c 'Elements' internal/eval/eval_patterns.go` | **5** indexed accesses, of which the two list-pattern ones are `:224` `listVal.Elements[i]` (exact-match arm) and `:244` `listVal.Elements[i]` (tail-pattern head elements); control **18** total `Elements` references in the same file, so the narrower pattern is a measurement, not an artifact |
| V19 | No existing persistent-sequence / copy-on-write / structural-sharing machinery exists in the repo to reuse (negative claim + firing control) | `grep -rniE '<term>' --include='*.go' internal/ cmd/` for each of copy-on-write/copyonwrite/CopyOnWrite, finger-tree/FingerTree, rope/Rope, cons-cell/consCell, structural-sharing; control the same command for `ListValue` | **0** for every term; control **753** `ListValue` hits at the same scope. The 69 `persistent` and 33 `immutable` hits were inspected by file and are storage/session persistence (`sharedmem_sqlite.go`, `opencode_test.go`, `eval_trend.go`) and effect/env-semantics prose (`effects/env.go`, `builtins/map.go`, `builtins/array.go`) — no sequence implementation among them |
| V20 | Neither the repo's generics nor its dependencies supply a reusable collection to build on | `grep -rnE 'type [A-Za-z_][A-Za-z0-9_]*\[[A-Za-z_]+ any\] struct' --include='*.go' internal/`; control `grep -rcE '^type [A-Za-z]' --include='*.go' internal/`; plus a read of all 99 `require` lines in `go.mod`; plus `grep -rn 'iter\.Seq' --include='*.go' internal/ cmd/` | generic struct types **2** (`internal/storage/firestore/cache.go:10 ttlCache[T any]`, `internal/apiserver/callbacks.go:32 callbackResult[T any]`) against a control of **2142** declared types — neither is a sequence. No collection/persistent-data-structure library among the 99 direct requires (they are GCP, OTel, LSP, sqlite, MCP, ollama, testify). `iter.Seq` usage in-repo: **0**, though `go.mod` declares `go 1.26.6` so the stdlib facility IS available — the spike would be its first in-repo use |

## Unverified / needs measurement

| Claim | Status | Owner |
|---|---|---|
| 32 B/cell for C1, ~16-18 B/element for C2 | Derivation from struct layout + Go size classes | B6 measures |
| The measured slice baseline is ≈16 B/element | Derivation matched to RSS within 3.2% in the superseded doc (V14) | B6 measures first-party |
| C2's shared-chunk prepend is implementable at O(K) worst-case without locks | Design argument | Day-2 implementation; failure ⇒ C2 columns marked infeasible (which the kill criterion tolerates if C1 passes) |
| Actual B8 `HeapAlloc` peak during the workload | Not measured: endpoint `ReadMemStats` snapshots cannot establish a peak, so the metric was removed from B8 | Out of scope; B8 reports defined before/after endpoints only |
| Evaluator-level (end-to-end AILANG) impact of the chosen representation | Out of scope by design — no production wiring | LC-4's before/after using this harness as the Go-level control |
| Concurrency/publication safety of any candidate | Not claimed; the spike is single-goroutine except the chunk-contention benchmark's setup | LC-4 (roadmap N21) |

## Quorum verification log

Two metered rounds, `gpt5-6-sol` + `gemini-3-1-pro`. **Both reviewers were PRESENT in both rounds and
`absent_reviewers` was EMPTY both times**, so neither the N−1 budget-degrade trap nor a
controller-satisfied `presentCount` applies — the verdicts are two real external eyes, twice.

**Round 1 — BLOCKED 2-of-2** (`m-list-repr-spike-2026-08-20T10-45-40Z.json`, $0.0757, 18,355 in / 543 out).
- `gpt5-6-sol`: the go/no-go gate was **not deterministic** — `-count=3` with no defined replicate,
  aggregation, boundary-crossing or noise-adjudication rule, so identical recorded numbers could
  produce different verdicts on the gate that decides ~16 days of work. Its further catch: B8 could
  not obtain a *peak* `HeapAlloc` from before/after `ReadMemStats` endpoints. **Applied**: five
  fresh-process trials, median of paired ratios, predeclared tie/spread rerun, B6 isolation +
  same-process baseline, explicit timeouts; the unsupported peak metric was **removed** (endpoint
  counters only) rather than papered over, and its removal is recorded under *Unverified*.
- `gemini-3-1-pro`: Conflict Surface #3 rested on the **false premise** that "Go gives no mechanism
  to forbid same-module imports". **The controller MEASURED this rather than forwarding it**
  (both arms, exit codes captured to file rather than read through a pipe — see V16): a nested
  `tools/internal/` package is refused to production importers (`rc=1`, *use of internal package …
  not allowed*) while compiling cleanly for importers below `tools/` (`rc=0`, zero output). The
  premise was false; the package was relocated to `tools/internal/spike-listrep/` at all 7 path
  sites, and social README enforcement was replaced by a compiler gate.

**Round 2 — BLOCKED 2-of-2** (`m-list-repr-spike-2026-08-20T10-53-42Z.json`, $0.0849, 20,791 in / 620 out).
Both round-1 fixes were accepted; these are *new*, and both are verification-completeness defects
rather than disputes about the design direction:
- `gpt5-6-sol`: no verified inventory of existing persistent-sequence / copy-on-write / list-view /
  iterator / collection machinery, and V3's method-only grep could not support "no accessor surface
  exists". **Measured** (V17–V20) → the broad claim mostly HOLDS (0 COW / finger-tree / rope /
  cons-cell / structural-sharing hits against a firing control of 753 `ListValue` hits; 2 generic
  types, neither a sequence, against a control of 2142 declared types; 0 of 99 dependencies) — but
  the reviewer was RIGHT that the instrument was too narrow: **2 free functions over `*ListValue`
  do exist** (`encodeJSONArray`, `encodeJSONObject`). V3's conclusion is therefore **narrowed** to
  the method surface, the free functions are recorded as LC-3x migration sites, and the new
  *Representation/API overlap and reuse decision* subsection states reuse-or-why-not per mechanism.
  One finding runs the other way and is adopted: `iter.Seq` has **0** in-repo uses, so D3's iterator
  is stdlib-adopted and the doc now says so instead of implying a house idiom.
- `gemini-3-1-pro`: the D3 `At(i int)` justification asserted `_list_nth` was "bounds-checked" and
  that patterns do indexed head-element access, while the cited V6 captured only the builtin's name
  and registration guard and no row covered indexed access at all. **Measured**: both claims are
  TRUE and were simply unsupported — `list.go:320` is an explicit
  `index %d out of bounds for list of length %d` refusal, and `eval_patterns.go:224`/`:244` are the
  two indexed head accesses (5 indexed of 18 total `Elements` references in that file). V6 expanded
  to read the body; V18 added; the D3 citation repointed from the registration site to the checking
  logic.

**Disposition — narrow-refinement carve-out** (ratified iteration 98). Every surviving objection
carried a concrete reviewer-authored `proposed_fix` and none disputed the design DIRECTION (cons
cells, spike-first ordering, the kill criterion). The controller therefore applied the reviewers'
own fixes plus first-party measurements in one bounded second revision and routed onward, rather
than buying a third metered round to re-litigate fixes both reviewers had already specified. No
threshold, observable, or control was weakened: the clause literals (1.5, ≥8×, 2.0, 2.5, 1.2) are
unchanged, verified by count before and after each revision.

## Related Documents

- [m-list-cons-cells-decomposition.md](m-list-cons-cells-decomposition.md) — parent roadmap;
  LC-1 scope at lines 147-176 is authoritative; this doc refines, never widens, it.
- [m-list-cons-quadratic.md](m-list-cons-quadratic.md) — superseded arena design; source of the
  inherited measurement record (V14) and the B2/B8 size ladder.
- LC-2 (`m-list-accessor-api`, not yet authored) — consumes D3; owns the real analyzer and the
  production accessor seam. Deliberately NOT designed here (anti-goal).

## References

- [#676](https://github.com/sunholo-data/ailang/issues/676) — the user-reported OOM (email-parse M9)
- [#745](https://github.com/sunholo-data/ailang/issues/745) — carries `D-19 : B`; destination of a STOP verdict
- [design_docs/PROGRAM.md](../PROGRAM.md) — routing lane: AILANG fix

---

**Document created**: 2026-08-20 (mission iteration 236)
**Authored at**: `8322d22b7` (= `origin/dev`), branch `sprint/iter236-list-repr-spike`

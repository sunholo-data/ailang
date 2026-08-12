# M-TAKE-FLATMAP-PEAK-MEMORY: `take(n, flatMap(f, xs))` Cannot Bound Peak Memory — Expose the Fused Primitive That Already Exists, Fix Its Latent Type Bug, and Teach the Trap

**Status**: Planned
**Target**: v0.34.0
**Priority**: P1 (two production host OOMs in `sunholo/ailang_parse`; silent, inverted failure mode — the code *looks* defensive)
**Estimated**: 3–4 days (round 1 cut the `takeMap` export and revised down to 3; round 2 restored it on measured evidence — see Round 2 section)
**Dependencies**: None
**Source**: GitHub issue [#617](https://github.com/sunholo-data/ailang/issues/617)
**Base SHA for all measurements**: `eabab0611c90dce93508bf9dfbdfad136f16daf8` (worktree binary `v0.33.0-143-geabab0611`)

---

## TL;DR — the design-changing discovery

Issue #617 proposes three resolutions: (1) a fused `takeFlatMap` primitive, (2) a check-time
lint, (3) documentation. **Resolution (1) was already built and shipped in v0.10.0** —
`_list_takeFlatMap` and `_list_takeMap` exist in `internal/builtins/list_bounded.go` with a
full unit-test suite (M-EVAL-BOUNDED-PIPELINE, commit `d41e43894`, motivated by the *same
DocParse Moby Dick OOM* now reported in #617) — **and then became unreachable in practice**,
for three independently verified reasons:

1. **No stdlib surface.** `import std/list (takeFlatMap)` fails with `IMP010` (V4). The
   builtins were registered under `$builtin` but the `std/list.ail` export was never added.
2. **A latent type-registration bug makes the builtin unusable behind any `int` annotation.**
   `list_bounded.go` registers the count parameter as `TCon{Name: "Int"}` (capital I);
   the surface language's int is `TCon{Name: "int"}` (V5, V-BUILDER). Bare numeric literals
   unify (they're polymorphic), which is why the shipped unit tests pass — but the moment `n`
   flows through an annotated variable or a stdlib wrapper (`n: int`), unification fails:
   `cannot unify type constructors: Int vs int`. The one call shape a stdlib export requires
   is exactly the shape that has never worked.
3. **Zero discoverability.** The fused builtins appear in no teaching-prompt line (V13), no
   LIMITATIONS entry (V14), no example, and no `internal/diag/footguns.md` row (V15). The
   compiler note promised by M-EVAL-BOUNDED-PIPELINE's own success criteria ("Compiler note
   emitted for `take(N, flatMap(f, xs))` pattern") was **never implemented** (V11).

So the same downstream consumer OOM'd a host twice, five months after the fix for its exact
pipeline was merged, because the fix was never exported, never taught, and broken for
annotated arguments. This doc's recommendation is therefore not "build a fused primitive" —
it is **expose + repair + teach**, in that order, with the lint scoped by what a syntactic
note can actually see (Phase 3 — round 2 corrected a round-1 over-read of V7 here).

The trap itself is re-derived first-party at base SHA (V1/V2): the same 5-element result
costs **81.9 s wall / 425 MB peak RSS** through `take(5, flatMap(...))` and
**0.06 s / 89 MB** through `_list_takeFlatMap(5, ...)` — a 1365× wall and 4.8× peak-RSS gap,
with byte-identical output. `take` bounds the *result*, never the *peak*. **And neither does
`takeFlatMap`** — it removes the dominant term (the materialised outputs of unvisited
inputs), not the source list's residency nor a single `f(x)`'s size (V21–V23; corrected
cost model below, used throughout this revision). Round 2 measured the **same trap for
`take(n, map(f, xs))` whenever `f` allocates** — 5.5× peak RSS and 235× wall on the
reviewer-prescribed fixture (V25/V26) — so **both** fused builtins now ship stdlib exports:
`takeFlatMap` and `takeMap`.

## Round 1 objections and how they were resolved

The first quorum (gpt5-6-sol, gemini-3-1-pro) returned BLOCKED with two objections. Both
were **accepted** — the first was independently measured true by the controller before this
revision; the second is correct on this doc's own V7 measurement.

### Objection 1 (gpt5-6-sol, verbatim) — the cost-model claim was wrong

> "The design repeatedly presents `takeFlatMap` as bounding peak memory and claims 'Peak
> allocation is O(n) outputs + O(1) per visited input element,' but the callback returns a
> strict list. A single visited call to `f(x)` can materialize an arbitrarily large inner
> list before the builtin takes only the needed prefix. The source list is also already
> resident. Thus the primitive avoids materializing outputs for unvisited inputs but does
> not generally bound peak memory by n; the central cost-model claim is unverified and
> likely false, violating A9 cost visibility."

The controller measured this rather than forwarding it, and the reviewer is right on both
points. All three arms call `_list_takeFlatMap` **directly** with a literal cap of 5 (base
SHA `eabab0611`, rebuilt binary reporting `v0.33.0-141`, `--version` checked; interpreter
floor measured separately: 46,104,576 B):

| arm | cap | source length | size of ONE f(x) | wall | peak RSS | above floor |
|-----|-----|---------------|------------------|-------:|--------------:|------------:|
| q | 5 | 3 | 10 | 0.02 s | 49,872,896 B | ~3.8 MB |
| p | 5 | 3 | 500,000 | 2.23 s | 137,789,440 B | ~91.7 MB |
| s | 5 | 500,000 | 1 | 2.17 s | 141,213,696 B | ~95.1 MB |

Arm p vs arm q: cap and source length identical, **only the size of a single `f(x)`
varies** → +92 MB (the reviewer's first point). Arm s vs arm q: cap identical, `f(x)`
tiny, **only the source length varies** → +95 MB (the second point). All three rc=0,
outputs asserted, zero stack overflows. An earlier attempt that built the 500k-element
list by direct `rep(300000, …)` AILANG recursion died in a Go `fatal error: stack
overflow` at 1.49 GB — a peak that belonged to the list **builder**, not the primitive —
so that contaminated pair was discarded and the clean arms build their large lists via the
fused builtin itself (V24 records the discard so the 1.49 GB is not re-attributed).

**The corrected cost model, used everywhere in this doc:**

    peak = O(source list, already resident) + O(largest single f(x)) + O(n retained outputs)

`takeFlatMap` does **not** bound peak memory by n. What it buys — and it is the real,
still-large win — is eliminating the unfused form's dominant term: the materialised
outputs of every **unvisited** input. In V1/V2's own numbers: the unfused arm holds all
200 × 2000 = 400,000 materialised output elements at peak (425 MB); the fused arm holds
the 200-element source + one element's 2,000 outputs + the 5 retained ≈ 2,205 live output
elements (89 MB) — 379 MB vs 43 MB above the 46 MB interpreter floor, an ~8.8× above-floor
reduction on the identical workload. In the motivating incident's terms: DocParse's
unfused pipeline held ~2,800 document elements' token lists simultaneously (#617); the
fused form holds one element's tokens plus the retained prefix. If a *single* `f(x)` can
itself be unbounded — the general case — the user must bound it **inside `f`**: #617's own
budgeted-walk workaround applies there unchanged (e.g. a capped per-element tokenizer).
The LIMITATIONS entry and prompt copy state all of this explicitly.

Changes made: the false "Peak allocation is O(n) outputs + O(1) per visited input element"
claim deleted everywhere it appeared; corrected model written into the export's doc
comment, the LIMITATIONS entry, the prompt block, the Problem Statement, and AC-3's
framing; A9's +2 re-justified against the corrected model; rows V21–V24 (including the
discarded pair) added to the Verification Log.

### Objection 2 (gemini-3-1-pro, verbatim) — `takeMap` had no motivating problem

> "Exposing takeMap violates the Minimal Frozen Core axiom. The document proves in V7 that
> the unfused take(n, map(f, xs)) does not suffer from peak memory amplification. Exporting
> takeMap simply because the builtin 'already exists and costs ~4 LOC to expose' needlessly
> expands the standard library surface area without a motivating problem, directly
> contradicting the minimal core mandate."

Accepted on this doc's own evidence — V7 is our measurement, and existence-plus-cheapness
is not a motivating problem under PROGRAM.md's minimal-core routing bias. Changes made:
`takeMap` dropped from the Phase 1 export set (`takeFlatMap` ships alone); AC-1/AC-2,
Files to Modify, Conflict Surface, Testing Strategy, and the effort estimate updated to
match. The `Int`→`int` repair is **kept for both builtins** — they share the registration
path in `list_bounded.go`, and a latent type bug is worth fixing where it sits — but
`_list_takeMap` stays an unexported builtin. "Export `takeMap`" moved to Future Work
behind a stated evidence bar; `takeMap` added to Non-Goals so the next iteration does not
re-propose it, noting `_list_take` sits in the same unexposed state under the same bar.

*(Round-2 note: this resolution's premise — "V7 shows map has no amplification" — was
itself an over-read, caught by the same reviewer in round 2. The cut is REVERSED below;
the round-1 record above is kept as history, not as a live claim.)*

## Round 2 objections and how they were resolved

The second quorum (same reviewers) returned BLOCKED with two objections. Both were
**accepted, and the reviewers' own proposed fixes applied verbatim** under the mission's
narrow-refinement carve-out: neither objection disputes the expose + repair + teach
direction — objection 3 corrects an acceptance criterion's *instrument*, objection 4 a
premise's *attribution*.

### Objection 3 (gpt5-6-sol, verbatim) — AC-3a was neither verified nor bounded

> "AC-3a is neither verified nor bounded: it proposes running the n=200 unfused interpreter
> workload in CI even though V1 measures that workload at about 81.9 seconds, and it asserts
> a TotalAlloc ratio of at least 20x without ever measuring TotalAlloc. The cited 425 MB
> versus 89 MB RSS figures are peak-residency measurements, not cumulative-allocation
> evidence, and imply only a 4.8x floor-inclusive RSS ratio. A process-global
> runtime.ReadMemStats delta is also GC-sensitive, so this acceptance gate violates bounded
> verification and deterministic-test requirements."

Proposed fix (verbatim): *"Replace AC-3a with a fast Go test using an instrumented callback
and a small fixed input. Assert that `_list_takeFlatMap` returns the expected prefix and
invokes `f` only through the first input needed to produce n outputs, while an unfused
reference invokes `f` for every input. This deterministically kills an implementation
rewritten as `take(n, flatMap(...))` and directly verifies removal of the unvisited-output
term. Give the test an explicit short timeout. Retain `/usr/bin/time -l` as non-CI release
evidence for peak RSS. If an allocation gate is still desired, first add a verification-log
row measuring the exact reduced fixture across repeated isolated runs, then choose a
threshold supported by those measurements rather than by RSS."*

Applied verbatim: AC-3 replaced, Testing Strategy and Files to Modify updated to match.
The objection caught a **category error** of exactly the kind this mission keeps closing:
RSS (peak residency) was quoted as evidence for a TotalAlloc (cumulative allocation)
threshold. Those are different quantities, and nothing in this doc now derives one from the
other — the old AC-3a sentence that inferred cumulative allocation from the V1/V2 RSS
figures is deleted, not reworded.

### Objection 4 (gemini-3-1-pro, verbatim) — "map has no amplification" was falsely generalized

> "The premise that `take(n, map(f, xs))` has no peak memory amplification (derived from V7
> and used to drop `takeMap`) is falsely generalized. V7 tests `map(\x. x + 1, big)`, where
> `f(x)` is a scalar operation that allocates no heap objects. Because AILANG is strict,
> `map` fully evaluates `f(x)` for all unvisited elements. If `f(x)` returns a large object
> (e.g., parsing a string to an AST), `take(n, map(f, xs))` will materialize all N large
> objects simultaneously before `take` discards any, causing the exact same peak-memory OOM
> as `flatMap`. `takeMap` eliminates this identical 'unvisited outputs' term."

Catch (verbatim): *"You accepted a flawed Round-1 objection by using a zero-allocation
`f(x)` to prove a strict map doesn't amplify memory."*

Proposed fix (verbatim): *"Update V7 to test an allocating map:
`take(5, map(\x. rep(2000, x), big))` vs `_list_takeMap(5, \x. rep(2000, x), big)`. This
will correctly demonstrate that strict `map` amplifies peak memory proportionally to
`f(x)`'s size. Then, either restore the `takeMap` export and its lint, or keep it deferred
but correct the justification in Non-Goals to 'Production OOMs were specific to flatMap'
rather than the false claim that 'map has no amplification'."*

The controller ran the reviewer's prescribed measurement rather than forwarding the
objection (base SHA `eabab0611`, rebuilt binary reporting `v0.33.0-141`, `--version`
checked). `src()` is a 2,000-element list built via
`_list_takeFlatMap(2000, \x. rep(100, x), upto(50))` so no arm uses deep AILANG recursion
(the V24 rule). Both arms print `5` (identical output), literal cap of 5 in both, zero
stack overflows (asserted):

| arm | pipeline | wall | peak RSS |
|---|---|---:|---:|
| fused | `_list_takeMap(5, \x. rep(2000, x), src())` | 0.08 s | 101,203,968 B |
| unfused | `take(5, map(\x. rep(2000, x), src()))` | 18.78 s | 558,907,392 B |

**5.5× peak RSS and 235× wall, same output, same cap, same source.** Interpreter floor
46,104,576 B, so the above-floor payload is ~55 MB vs ~513 MB. Recorded as V25/V26.

**How the two gemini objections reconcile — the most useful history in this doc for
whoever plans the sprint.** Round 1's objection 2 cut the `takeMap` export for having "no
motivating problem"; round 2's objection 4 supplies one, by the reviewer's own prescribed
measurement. The two objections pointed in opposite directions on `takeMap`, but they are
not in conflict once the measurement exists: **round 1 demanded evidence, round 2 supplied
it.** This doc's own Future Work bar for exporting `takeMap` was "a measured motivating
case" — that bar is now met by measurement, not by argument. The FIRST branch of the
reviewer's proposed fix is therefore taken: **the `takeMap` export is restored**, which is
what satisfies BOTH objections, and the minimal-core principle is restated as an
**evidence bar, not a headcount** (Non-Goals).

Changes made in this revision:
- **V7 relabelled** to what it actually shows — a NON-ALLOCATING `f` does not amplify —
  with its command and observed output untouched; rows **V25/V26 added**. The contrast
  between V7 and V26 IS the finding: strict `map` amplifies peak proportionally to
  `f(x)`'s size, and a scalar `f` is the degenerate case, not the general one.
- **`takeMap` restored everywhere round 1 cut it**: the Phase 1 export, Files to Modify,
  Conflict Surface, AC-1 (which now asserts the import SUCCEEDS, reversing round 1),
  AC-2's wrapper arm, AC-4's parity arm, AC-5's docs rows, Testing Strategy, and the
  effort estimate (3 → 3–4 days).
- **Non-Goals rewritten** on the evidence-bar principle: `takeFilter` and further fused
  forms stay out (still no measured motivating case), `_list_take` stays unexported under
  the same bar, and the false "map has no amplification" justification is removed.
- **Phase 3's lint scoping restated** in terms of what a syntactic note can actually see,
  with an explicit false-positive/false-negative profile — the old scoping ("do not warn
  on `take(n, map(...))`: no amplification") was derived from the over-read V7.
- **AC-3 replaced** per objection 3 (instrumented-callback invocation count in CI;
  `/usr/bin/time -l` retained as non-CI release evidence).

## Corrections to prior triage (recorded per the mission loop's own rules)

- `design_docs/v1-mission.md`'s queue row for #617 says *"stdlib has NO `flatMap` (grep 0,
  control firing) so the class is user-written eager flatMaps"*. **False.** `std/list.ail:202`
  exports `flatMap` (recursive, via `concat`), `std/list.ail:250` exports the effectful twin
  `flatMapE` (V16). The earlier grep ran against a non-existent `stdlib/` directory. The
  consequence is material: both halves of the trap are AILANG's **own exported, taught
  stdlib**, which moves the lane from "docs + lint for user code" to "stdlib/builtin fix".
- Issue #617 says the fused shape "cannot currently [be] express[ed] without hand-rolling
  recursion". **Half-false**: `_list_takeFlatMap` is directly callable today and solves the
  repro (V2) — but only with literal arguments (V5), it is documented nowhere a user would
  look, and the `_`-prefixed name is not part of any taught surface. Practically the issue's
  claim was true for its author; mechanically the primitive exists.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | `takeFlatMap(n, f, xs)` ≡ `take(n, flatMap(f, xs))` for pure `f` — same elements, same order (existing parity tests + new cross-check AC-4). No new nondeterminism; the pure-only type (V6) means "which effects run" cannot silently change. |
| A2: Replayability | 0 | Pure functions; no trace-surface change. |
| A3: Effect Legibility | +1 | The type checker already **rejects** effectful `f` on the fused builtin (V6: `failed to unify effect rows … extra labels [IO]`), so exposing it opens no effect hole. The effectful twin — whose short-circuiting would change *which effects run* — is explicitly deferred to its own design (see Non-Goals), keeping the effect surface exactly as legible as today. |
| A4: Explicit Authority | 0 | No capabilities involved. |
| A5: Bounded Verification | 0 | No new type complexity; fixes an existing mis-registered type to the canonical `int` constructor. |
| A6: Safe Concurrency | 0 | Single-threaded iteration, immutable inputs. |
| A7: Machines First | +1 | The footgun diagnostic (Phase 3) is error-time teaching per `internal/diag`'s charter: the model that writes `take(n, flatMap(...))` gets the fix (`takeFlatMap`) in the diagnostic, enabling one-round self-repair instead of a per-run prompt tax. |
| A8: Minimal Syntax | +1 | Zero new syntax. Two `std/list` exports delegating to already-registered builtins — the exact `map`→`_list_map` idiom at `std/list.ail:56`. (Round 1 cut `takeMap` to one on objection 2's no-evidence ground; round 2's measurement [V25/V26] supplied the evidence and restored it. The minimal-core principle is an evidence bar, not a headcount — see Non-Goals.) |
| A9: Cost Visibility | +2 | **This axiom is the whole item — and round 1 of this doc itself violated it**, claiming the fused form's peak is "O(n) outputs by construction"; the quorum caught it and the controller measured it false (V21–V23). Today the cost model of `take∘flatMap` is invisible *and inverted* — the code reads as a memory cap and is the opposite (V1: cap of 5, 425 MB peak). After: the export's doc comment, the LIMITATIONS entry, and the prompt block state the **corrected, measured** model — peak = source residency + largest single `f(x)` + n retained outputs — including what `takeFlatMap` does *not* bound and what the user must still bound inside `f`. The +2 is retained only after this correction, and not for the primitive's asymptotics: the deliverable is a true, measured cost model (V1/V2 + V21–V23 tables, all shipped in the teaching text) where today there is a false one. A doc shipping the round-1 claim would have deserved a negative score here, for exactly the reason the objection gives. |
| A10: Composability | +1 | `dedup(takeFlatMap(n, f, xs))` composes directly — verified live (V18) — so the motivating DocParse line rewrites 1:1. |
| A11: Structured Failure | +1 | Removes the amplification-driven failure modes (host OOM; `RT_REC_003` that worsens when `--max-recursion-depth` is raised) for the observed class. The fused form can still OOM when the source list or one `f(x)` is itself huge (V22/V23) — that residual is documented, not hidden. Builtin errors remain structured (`callback must return a List`, etc. — existing tests). |
| A12: System Boundary | 0 | No FFI/registration-path change; same `RegisterEffectBuiltin` mechanism. |

**Net Score: +8** → **Decision: Move forward.**

### Hard Violation Check

- [x] A1 (Determinism): pure, order-preserving, result-identical to the unfused form
- [x] A3 (Effects): pure-only at the type level, verified by a live rejection (V6)
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): diagnostic carries the concrete fix; footgun row is a CI-fixtured contract

---

## Problem Statement

AILANG is strict: `flatMap` materialises **every** element's output before `take` discards
any. A cap written as `take(10000, flatMap(tokenize, elements))` bounds the result, never the
peak. In `sunholo/ailang_parse` this shipped as an explicit, commented mitigation
("Cap at 10000 words") and still exhausted host memory twice; raising
`--max-recursion-depth` — the natural response to the `RT_REC_003` symptom — made it worse.

The failure mode is silent and inverted: a reviewer sees `take(10000, ...)` plus an OOM
comment and concludes memory is handled. There is no warning, and the symptom appears far
from the cause, only on large inputs, so it survives every small-input test.

The fix this doc ships is honest about its own limits: `takeFlatMap` eliminates the
dominant term — the materialised outputs of every unvisited input — but does **not** bound
peak by n either. The source list is already resident, and each visited `f(x)` is fully
materialised before the prefix is taken (V21–V23). The teaching text carries both halves,
because guidance that said "takeFlatMap bounds peak memory" would recreate #617's exact
failure mode — a documented cap that does not cap — one layer up.

### First-party repro at base SHA (V1, V2)

Two `ailang check`-clean programs computing the same value (full sources in Appendix A):

| arm | pipeline | wall | peak RSS | output |
|---|---|---:|---:|---|
| A (the #617 shape) | `take(5, flatMap(\x. rep(2000, x), upto(200)))` | 81.93 s | 425,361,408 B | `[1, 1, 1, 1, 1]` |
| D (existing builtin) | `_list_takeFlatMap(5, \x. rep(2000, x), upto(200))` | 0.06 s | 89,374,720 B | `[1, 1, 1, 1, 1]` |

The wall-time gap is not just allocation: `flatMap` at `std/list.ail:202` concatenates via
the AILANG-recursive `concat` (`std/list.ail:37`, `[x] ++ concat(rest, ys)`), which is
quadratic in the accumulated output. The controller's independent measurements at n=50/100/200
(5.84 s / 21.10 s / 78.57 s) show the ~quadratic growth in source length; the cap is 5
throughout.

### Which neighbours share the trap? Measured, not assumed (V7, V8, V9 — corrected by V25, V26)

All round-1 arms share a 400,000-element input `big` (built via the fused builtin so
construction cost is identical), against a control that only does `take(5, big)`:

| arm | pipeline | wall | peak RSS | verdict |
|---|---|---:|---:|---|
| control | `take(5, big)` | 4.15 s | 185,925,632 B | baseline: holding `big` costs ~186 MB |
| take-after-map | `take(5, map(\x. x+1, big))` | 3.11 s | 185,729,024 B | **no amplification with a NON-ALLOCATING `f`** — peak within noise of control (see below) |
| take-after-filter | `take(5, filter(\x. x>0, big))` | 3.19 s | 186,466,304 B | **no amplification** |
| take-after-concat | `take(5, concat(big, [1]))` | 2.31 s | (died) | **different failure**: `RT_REC_003: max recursion depth 10000 exceeded` — a stack limit in `concat` itself, not a take-ordering memory trap |

Round 1 over-read the map row as "map has no amplification" — the round-2 quorum caught
that its `f` (`\x. x+1`) is a scalar that allocates nothing, so the row only shows the
degenerate case. Strict `map` fully evaluates `f(x)` for **every** element before `take`
discards any; when `f` allocates, the unvisited-outputs term is identical to `flatMap`'s.
Measured on the reviewer's prescribed fixture (`src()` = 2,000-element fused-built list;
both arms print `5`, identical output; zero stack overflows; V25/V26):

| arm | pipeline | wall | peak RSS |
|---|---|---:|---:|
| fused | `_list_takeMap(5, \x. rep(2000, x), src())` | 0.08 s | 101,203,968 B |
| unfused | `take(5, map(\x. rep(2000, x), src()))` | 18.78 s | 558,907,392 B |

**5.5× peak RSS and 235× wall** — above the 46,104,576 B interpreter floor, ~55 MB vs
~513 MB. Corrected interpretation: the trap is the strict materialisation of unvisited
outputs. It is structural for `flatMap` (output-length amplification, here 2000×) and real
for `map` exactly when `f(x)` allocates (size amplification, invisible in V7's scalar
case). `filter` measured no amplification (V8) — its retained elements are the input's own
elements, and predicate results are not retained. Consequences: **both** `takeFlatMap` and
`takeMap` ship exports (round 2 restored the latter); no `takeFilter` is needed; and the
lint's scope is restated in Phase 3 in terms of what a syntactic note can actually see — it
cannot see whether `f` allocates. `concat`'s recursion limit is real but orthogonal — it
fails fast with a structured error at 10k elements regardless of `take`, and is noted in
Future Work.

---

## Verification Log

Every codebase claim in this doc, one row each. All commands run in the worktree at
`eabab0611`; `./bin/ailang` built from that SHA. Negative/empty findings carry an in-call
known-positive control (glob-shaped flags quoted — this rig's zsh aborts on unquoted globs).
Rows V21–V24 were run by the **controller** during round-1 review (same base SHA; rebuilt
binary reporting `v0.33.0-141`, `--version` checked) and are recorded verbatim, including
the discarded contaminated pair. Rows V25–V26 were run by the **controller** during round-2
review (same base SHA and binary), executing objection 4's prescribed measurement.
Existing rows are untouched and unrenumbered; the one permitted change is V7's **claim**
column, relabelled in round 2 to what its measurement actually shows — its command and
observed output are unchanged.

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 | The trap reproduces: cap of 5, 425 MB peak, 81.9 s | `/usr/bin/time -l ./bin/ailang run --caps IO arm_a.ail` (Appendix A) | `[1, 1, 1, 1, 1]` · `81.93 real` · `425361408 maximum resident set size` |
| V2 | The existing builtin solves it today | `/usr/bin/time -l ./bin/ailang run --caps IO arm_d.ail` | `[1, 1, 1, 1, 1]` · `0.06 real` · `89374720 max RSS` |
| V3 | Both fused builtins are registered | `./bin/ailang builtins list \| grep -in "take\|range"` | `132: _list_take · 133: _list_takeFlatMap · 134: _list_takeMap` all `[pure] $builtin` (control: grep also matched `_list_take`) |
| V4 | No stdlib surface | `./bin/ailang check arm_i.ail` (imports `std/list (takeFlatMap)`) | `Error: IMP010: symbol 'takeFlatMap' not exported by 'std/list' — did you mean 'take'?` (control: same file's `std/io (println)` import resolves; `take` named as exported) |
| V5 | Type-ctor bug: annotated `int` cannot reach the builtin | `./bin/ailang run --caps IO arm_w.ail` (wrapper `takeFlatMap[a,b](n: int, ...) = _list_takeFlatMap(n, f, xs)`) | `type error … cannot unify type constructors: Int vs int` at the delegation site. Positive control: same builtin with a bare literal (V2) and with `let k = 2` (arm_v → `[1, 1]`) both run |
| V-BUILDER | The canonical int constructor is lowercase; `list_bounded.go` deviates | `sed -n '25,45p' internal/types/builder.go` · `grep -n 'TCon{Name: "Int"}' internal/builtins/*.go` | `Builder.Int() = &TCon{Name: "int"}`; `list_bounded.go` builds `intT := &types.TCon{Name: "Int"}` (used at lines 68–69, 150–151) |
| V6 | Effectful `f` is rejected by the type checker (no accidental effect hole) | `./bin/ailang check arm_e.ail` (passes `noisy: (int) -> [int] ! {IO}` to `_list_takeFlatMap`) | `type error … failed to unify effect rows: incompatible closed rows: r1 has extra labels [], r2 has extra labels [IO]` |
| V7 | take-after-map with a NON-ALLOCATING scalar `f` (`\x. x+1`) does NOT amplify peak — relabelled in round 2: this row was over-read as "map does not amplify"; the contrast with V26 is the finding | `/usr/bin/time -l … arm_m.ail` vs control `arm_0.ail` | 185,729,024 B vs control 185,925,632 B (Δ ≈ −0.1%) |
| V8 | take-after-filter does NOT amplify peak | `/usr/bin/time -l … arm_ff.ail` | 186,466,304 B (Δ ≈ +0.3% vs control) |
| V9 | take-after-concat fails on stack, not memory | `./bin/ailang run --caps IO arm_c.ail` (concat of 400k-element list) | `RT_REC_003: max recursion depth 10000 exceeded` |
| V10 | Aliasing defeats any syntactic lint | `./bin/ailang check arm_l.ail` (`let t = take; let g = flatMap; t(2, g(...))`) | `✓ No errors found!` — the aliased trap type-checks clean; no current or proposed diagnostic can see it |
| V11 | The promised compiler note was never shipped; `--max-memory` was | `grep -rn "materializes large intermediate\|consider takeFlatMap" internal/ cmd/ --include='*.go'` (empty) with in-call control `grep -rn "RT_REC_003" internal/ --include='*.go'` (3 hits); `./bin/ailang run --help \| grep -in memory` | note-grep: **0 hits**, control fires; help line 50: `-max-memory string  Memory limit (e.g., 256MB, 1GB)…`, impl at `cmd/ailang/memory_limit.go` |
| V12 | Ship provenance of the builtins | `git show -s --format='%h %ci %s' d41e43894; git tag --contains d41e43894 \| sort -V \| head -3` | `d41e43894 2026-03-19 … M-EVAL-BOUNDED-PIPELINE`; first tag **v0.10.0** (the builtin metadata says `Since: "v0.9.4"` — wrong, cleanup in Phase 1; the design doc sits in `implemented/v0_29_0/`) |
| V13 | Teaching prompt never mentions the trap or the fused forms | `./bin/ailang prompt --source=embedded \| grep -in "strict\|materiali\|takeFlatMap"` → hits are unrelated (streaming-XML/short-circuit-`&&` lines); control `grep -cin "flatMap\|std/list"` → **19** | prompt teaches `flatMap` 19 times, teaches the trap 0 times; source file `prompts/v0.16.2.md`: target grep 0, control `flatMap` 5 |
| V14 | LIMITATIONS says nothing about strictness/materialisation | `grep -cin "strict\|materiali\|peak memory" docs/LIMITATIONS.md docs/docs/reference/limitations.md` with control `wc -l` both files | **0** and **0**; controls: 92 and 345 lines (both files non-empty, greps ran) |
| V15 | No footgun-table row exists for the trap | `grep -n "take.*flatMap\|flatMap.*take" internal/diag/footguns.md` (rc=1) with control `grep -c "\| covered\|\| inventoried"` → **12** rows | absent; table live and populated |
| V16 | stdlib DOES export `flatMap` + `flatMapE` (charter row false) | Read `std/list.ail` | `flatMap` at :202 (recursive via `concat` :37), `flatMapE` at :250, `take` at :99 (recursive), `map`/`filter`/`foldl` delegate to builtins at :56–60; effectful-combinator contract comment at :212–214 |
| V17 | Existing builtin tests are green at base | `go test ./internal/builtins -run 'TakeFlatMap\|TakeMap' -count=1` | `ok … 0.333s` (22 test funcs + 2 benchmarks in `list_bounded_test.go`) |
| V18 | `dedup` composes with the fused form (the DocParse shape) | `./bin/ailang run --caps IO arm_x.ail` (`dedup(_list_takeFlatMap(4, \x. [x,x], [1,2,3]))`) | `[1, 2]` |
| V19 | Conflict check: no other planned doc touches `list_bounded.go` | `grep -rln "list_bounded" design_docs/planned/` (rc=1) with control: same grep over `design_docs/implemented/` finds `v0_29_0/m-eval-bounded-pipeline.md` | planned: 0 docs; `m-parmap-effectful.md` (planned/v0_30_0) does plan `std/list.ail` edits (its lines 191, 255) |
| V20 | verify-examples green at base | `make verify-examples` | `187 modules checked, 0 drift, 1 missing-on-disk` · `✅ verify-examples: all examples pass and manifest is in sync` (the pre-existing `1 missing-on-disk` is baseline, not ours) |
| V21 | Fused-call floor: cap 5, 3-element source, f(x) of 10 — everything small | `/usr/bin/time -l ./bin/ailang run --caps IO arm_q.ail` (controller; interpreter floor measured separately: 46,104,576 B) | rc=0, output asserted · `0.02 real` · 49,872,896 B max RSS (~3.8 MB above floor) |
| V22 | A single large `f(x)` alone drives peak — cap and source length held fixed | `/usr/bin/time -l ./bin/ailang run --caps IO arm_p.ail` (controller; cap 5, source 3, ONE f(x) of 500,000 elements) | rc=0 · `2.23 real` · 137,789,440 B max RSS (~91.7 MB above floor — +92 MB vs V21 with only the size of one f(x) varied) |
| V23 | Source-list residency alone drives peak — cap and f(x) held tiny | `/usr/bin/time -l ./bin/ailang run --caps IO arm_s.ail` (controller; cap 5, 500,000-element source, f(x) of 1) | rc=0 · `2.17 real` · 141,213,696 B max RSS (~95.1 MB above floor — +95 MB vs V21 with only source length varied) |
| V24 | DISCARDED contaminated pair, recorded so the 1.49 GB is not re-attributed to the primitive | earlier controller arms built the 500k list by direct `rep(300000, …)` AILANG recursion | Go `fatal error: stack overflow` at 1.49 GB peak — the peak belonged to the recursive list BUILDER, not `_list_takeFlatMap`; both arms of that pair discarded. Clean arms (V22/V23) build the list via `_list_takeFlatMap(500000, \x. rep(1000, x), upto(500))` so no clean arm uses deep AILANG recursion |
| V25 | Fused `takeMap` with an ALLOCATING `f`: fused cost is flat | `/usr/bin/time -l ./bin/ailang run --caps IO <fused round-2 arm>` (controller, round 2; `_list_takeMap(5, \x. rep(2000, x), src())` where `src()` = 2,000-element list built via `_list_takeFlatMap(2000, \x. rep(100, x), upto(50))` — no deep AILANG recursion, the V24 rule) | rc=0, prints `5` · `0.08 real` · 101,203,968 B max RSS (~55 MB above the 46,104,576 B floor) · zero stack overflows (asserted) |
| V26 | Unfused `take(5, map(f, …))` with the SAME allocating `f` amplifies: 5.5× peak RSS, 235× wall — strict map materialises every `f(x)` before `take` discards | `/usr/bin/time -l ./bin/ailang run --caps IO <unfused round-2 arm>` (controller, round 2; `take(5, map(\x. rep(2000, x), src()))`, same `src()`, same literal cap of 5) | rc=0, prints `5` (identical output to V25) · `18.78 real` · 558,907,392 B max RSS (~513 MB above floor) · zero stack overflows (asserted) |

---

## Goals

**Primary Goal:** Make the fused pipelines the *easy, taught* path: `takeFlatMap` **and**
`takeMap` importable from `std/list` (the latter restored in round 2 on measured
amplification, V25/V26), working with annotated arguments, documented in LIMITATIONS and
the teaching prompt **with the corrected cost model** — what the fused forms remove and
what they do not bound — plus a check-time diagnostic on the direct `take∘flatMap` trap
pattern, its scope stated with its blind spots (Phase 3).

**Success Metrics:**
- The #617 repro rewritten with `takeFlatMap` runs in <1 s and <150 MB peak RSS
  (vs 81.9 s / 425 MB measured, V1) with identical output.
- The allocating-`f` map pipeline rewritten with `takeMap` runs at fused cost —
  0.08 s / 101 MB measured (V25) vs 18.78 s / 559 MB unfused (V26) — with identical output.
- `import std/list (takeFlatMap)` and `import std/list (takeMap)` both compile
  (today: IMP010 for `takeFlatMap`, V4; `takeMap` absent from the export list, V16; AC-1).
- The annotated-argument form works (today: `Int vs int` type error, V5).
- `ailang check` on the direct `take(n, flatMap(f, xs))` pattern emits a fix-carrying,
  non-blocking note naming `takeFlatMap` (today: silent, V1's check output).
- LIMITATIONS (canonical + root), the teaching prompt, and `internal/diag/footguns.md` each
  carry the strict-materialisation entry with the corrected cost model — including what
  `takeFlatMap` does NOT bound, the bound-inside-`f` guidance, and the allocating-`f` map
  half with its `takeMap` rewrite (today: 0 mentions each, V13–V15).

---

## Answers to the four design questions the routing directive posed

**1. Does the fused form compose the way users actually write this?**
Yes — measured. The motivating line is `dedup(take(10000, flatMap(f, elements)))`; the fused
rewrite is the 1:1 `dedup(takeFlatMap(10000, f, elements))`, and `dedup∘takeFlatMap`
composition is verified live (V18). Argument order `(n, f, xs)` matches both the existing
builtin and `take(n, xs)`'s n-first convention.

**2. Does the same trap exist for map/filter/concat?**
For `map`: **yes, whenever `f` allocates** — round 1 answered "no" here by over-reading V7,
whose `f` (`\x. x+1`) allocates nothing. Strict `map` materialises `f(x)` for every element
before `take` discards any; measured with `f = \x. rep(2000, x)`, the unfused form costs
5.5× peak RSS and 235× wall vs `_list_takeMap` (V25/V26). That measurement is what restored
the `takeMap` export in round 2. For `filter`: no (V8) — retained elements are the input's
own elements. `concat` fails on recursion depth (`RT_REC_003`), a different (pre-existing,
structured) failure orthogonal to take-ordering. The lint still fires only on the
`flatMap` form — not because `take∘map` is safe, but because a syntactic note cannot see
whether `f` allocates; see question 4 and Phase 3 for the stated profile. No `takeFilter`
ships (Non-Goals: evidence bar).

**3. What is the story for `flatMapE`, the effectful twin?**
Deferred, and **safely** so — this is the sharpest question and the type system has already
answered the dangerous half: the fused builtin is pure-only at the type level, and passing an
effectful `f` fails effect-row unification today (V6). So exposing `takeFlatMap` cannot
silently change which effects run — misuse is a compile error, not a semantic surprise.
A future `takeFlatMapE` genuinely would change observable behaviour: it runs `f`'s effects
only for the input prefix needed to produce `n` outputs, making the *count* of executed
effects data-dependent, and it must be squared with the contract comment at
`std/list.ail:212-214` ("All effectful combinators evaluate elements left-to-right,
sequentially" — order is preserved by a prefix stop; "all elements" is not). AILANG has
precedent for name-explicit opt-in short-circuiting (M-EVAL-BOUNDED-PIPELINE's ratified
decision; `foldChildrenStep`'s `Stop` in std/xml), so the door is open — but there is zero
demonstrated demand (DocParse's tokenizer is pure), and the semantic surface deserves its own
axiom scoring. Shipping the pure form now forecloses nothing. One in-scope cleanup: the
builtin's registered `LongDesc` currently *claims* the effectful short-circuit behaviour
("For effectful f, only evaluates f for as many input elements as needed") — behaviour that
is unreachable through its own pure type (V6). Phase 1 corrects the metadata.

**4. Is the lint even decidable?**
Only partially, and the doc says exactly where. `take` and `flatMap` are ordinary exported
functions; the diagnostic pattern-matches direct application of the resolved `std/list`
globals — `App(take, [n, App(flatMap, [f, xs])])` — after import resolution (so
`import std/list (flatMap as fm)` still resolves to the same global and is caught, but a
**value-level alias** `let g = flatMap` is invisible: measured, the aliased trap type-checks
with zero diagnostics, V10; partial application and HOF-passed values are likewise
invisible). This is acceptable because the observed production instance and the shape a code
generator emits is the direct form, and because the diagnostic is a *note* (never blocks) —
its job is teaching at the moment of writing, with documented blind spots, not soundness.
The footgun-table row records the blind spots explicitly.

---

## Solution Design

### Phase 1 — Repair + expose (Day 1)

1. **Fix the type registration** in `internal/builtins/list_bounded.go`: replace both
   `intT := &types.TCon{Name: "Int"}` sites (`makeTakeMapType`, `makeTakeFlatMapType`) with
   the builder's canonical `T.Int()` (V-BUILDER). Two-line fix. The repair covers **both**
   builtins — they share the registration path — and both now get a stdlib surface too:
   round 2 restored the `takeMap` export on measured amplification (V25/V26).
2. **Add a Go regression test** that exercises the builtin *through an annotated `int`* (the
   shape V5 proves broken) via the pipeline, so the bug class — literal-only unification
   masking a wrong constructor case — cannot silently return. This test must be red against
   unfixed base.
3. **Export from `std/list.ail`** (~16 LOC, the exact `map`→`_list_map` delegation idiom) —
   `takeFlatMap` **and** `takeMap` (the latter restored in round 2, V25/V26):

   ```ailang
   -- takeFlatMap: fused take∘flatMap — collects at most n outputs and stops
   -- visiting inputs once it has them. It does NOT bound peak memory by n:
   --   peak = O(source list, already resident) + O(largest single f(x)) + O(n retained)
   -- What it removes is the unfused take(n, flatMap(f, xs))'s dominant term under
   -- strict evaluation: the materialised outputs of every UNVISITED input
   -- (425 MB -> 89 MB on the #617 shape; see LIMITATIONS "strict evaluation").
   -- If one f(x) can itself be huge, bound it inside f.
   export pure func takeFlatMap[a, b](n: int, f: (a) -> [b], xs: [a]) -> [b]
     = _list_takeFlatMap(n, f, xs)

   -- takeMap: fused take∘map — invokes f at most n times. Strict map evaluates
   -- f(x) for EVERY element first, so when f allocates, take(n, map(f, xs))
   -- materialises every output before taking n (measured 5.5x peak RSS / 235x
   -- wall on a 2,000-element source with a 2,000-element f(x); see LIMITATIONS).
   -- takeMap removes that unvisited-outputs term; it does NOT shrink the source
   -- list or a single f(x). With a non-allocating scalar f the unfused form is
   -- fine (measured: no amplification).
   export pure func takeMap[a, b](n: int, f: (a) -> b, xs: [a]) -> [b]
     = _list_takeMap(n, f, xs)
   ```

   (This delegation shape is what V5 was measured on; the arm_w file — both wrappers — is
   the sprint's ready-made fixture, and it compiles once the constructor case is fixed.)
4. **Correct builtin metadata**: `Since: "v0.9.4"` → the actual first tag `v0.10.0` (V12);
   `LongDesc` effectful-`f` claim → pure-only wording with "effectful variant deferred".
5. **Example**: `examples/bounded_take_flatmap.ail` — the unfused/fused pairs side by side
   (`take∘flatMap` vs `takeFlatMap`, and the allocating-`f` `take∘map` vs `takeMap`) with
   the cost comments; registered in the manifest (verify-examples is green at base, V20, so
   drift is attributable).

### Phase 2 — Teach (Day 2)

6. **Canonical LIMITATIONS entry** (`docs/docs/reference/limitations.md`, + summary row in
   root `docs/LIMITATIONS.md` per the M-V1-STABILITY-PROMISE entry policy: repro transcript +
   verified-at version): *"Strict evaluation: `take(n, flatMap(f, xs))` bounds the result,
   not the peak"* — with the V1/V2 measured table and the `takeFlatMap` rewrite. Kind:
   design constraint (strictness is ratified; see Non-Goals). Include the
   `--max-recursion-depth` anti-pattern (raising it makes the memory failure worse).
   **The entry MUST carry the corrected cost model** — peak = source residency + largest
   single `f(x)` + n retained outputs, with the V21–V23 table — and say plainly that
   `takeFlatMap` is *not* a peak-memory cap: it removes the unvisited-outputs term (in the
   incident: ~2,800 elements' token lists held at once → one element's tokens plus the
   retained prefix). Teaching it as a cap would recreate #617's failure mode — a documented
   cap that does not cap — one layer up. The entry must also state what the user still
   bounds themselves: if a single `f(x)` can be unbounded, apply #617's budgeted-walk
   workaround *inside* `f` (e.g. a capped per-element tokenizer). **And it must carry the
   map half** (round 2): strict `map` amplifies peak proportionally to `f(x)`'s size —
   `take(n, map(f, xs))` with an allocating `f` measured 5.5× peak / 235× wall vs `takeMap`
   (V25/V26), while a non-allocating scalar `f` measured no amplification (V7). The docs
   are the channel that CAN state this "if `f` allocates" conditional; the lint cannot
   (Phase 3).
7. **Teaching prompt** (`prompts/v0.16.2.md`, via the prompt-manager conventions): one
   Common-Mistakes block: `take(n, flatMap(f, xs))` materialises everything under strict
   evaluation → use `takeFlatMap(n, f, xs)`, which skips unvisited inputs' outputs but does
   NOT shrink the source list or a single `f(x)` — bound a huge `f(x)` inside `f`; same for
   `take(n, map(f, xs))` when `f` allocates → `takeMap(n, f, xs)`. Keep it to a few lines;
   the diagnostic (Phase 3) is the durable channel, per `internal/diag`'s
   prompt-tax-to-diagnostic doctrine.
8. **Footgun row** in `internal/diag/footguns.md`: trigger snippet, current diagnostic
   (today: none — file checks clean, V1), target diagnostic (Phase 3's note), fixture link,
   prompt lines it would let us delete; status `shipped-this-sprint` once Phase 3 lands.
   The row's blind-spot column records BOTH known false negatives: value-level aliases
   (V10) and the allocating-`f` `take∘map` form (V25/V26) — with `takeMap` named as the
   fix the note cannot deliver there.

### Phase 3 — Diagnose (Days 3–4)

9. **Check-time note** on the direct pattern. Site: the post-import-resolution pass where
   global refs are resolved (candidate: alongside the existing warning channel used by MOD010
   in `internal/pipeline/pipeline_module.go`; exact hook chosen by the sprint-planner —
   the pattern needs resolved `std/list.take` / `std/list.flatMap` references, so it must run
   at or after elaboration). Shape:

   ```
   note LIST_TAKE_AFTER_FLATMAP: take(n, flatMap(f, xs)) materialises the ENTIRE flatMap
   result before taking n (strict evaluation — the cap bounds the result, not peak memory).
   Fix: takeFlatMap(n, f, xs) from std/list stops after n outputs.
   ```

   Non-blocking (exit code unchanged); fires only on the direct `take(n, flatMap(...))`
   form. **Scope, restated in round 2 in terms of what the instrument can actually see**
   (the round-1 scoping — "do not warn on `take(n, map(...))`: no amplification" — was
   derived from the over-read V7 and is withdrawn): a syntactic note sees the combinator
   identity, never whether `f` allocates. For `flatMap`, the materialised intermediate is
   output-length-amplified by construction, so the note's claim is true wherever it fires;
   its false positives are only small-input cases, which the note text already concedes.
   For `map`, the identical trap is real **only when `f` allocates** (V25/V26: 5.5× peak,
   235× wall) — a condition invisible to a syntactic pass. **Proposed rule: fire on the
   direct `flatMap` form only; do not fire on `take(n, map(...))`.** Its profile, stated
   honestly: *false negative* — the allocating-`f` map trap gets no note, and is carried
   instead by the teaching text (LIMITATIONS + prompt, which CAN state the "if `f`
   allocates" conditional) and by the restored `takeMap` export, with the footgun row
   recording the blind spot alongside V10's aliases; *rejected alternative* — firing on
   every `take(n, map(...))` would false-positive on the common non-allocating scalar-`f`
   case (V7: measured harmless), eroding trust in the note channel. CI contract via a
   `footgun_fixtures_test.go` fixture asserting the code and the fix substring
   `takeFlatMap`, per that file's existing convention, plus two negative fixtures (fused
   form silent; take-after-map silent — the latter pins the stated rule, not a safety
   claim about `take∘map`).
10. If Phase 3 runs over budget, it is the **explicit cut line** — Phases 1–2 alone resolve
    the issue's primary ask, and the footgun row remains `inventoried` with the target
    diagnostic recorded. Cutting it is a recorded decision, not a silent drop.

### Files to Modify/Create

- `internal/builtins/list_bounded.go` — 2-line type fix + metadata corrections (~10 LOC Δ)
- `internal/builtins/list_bounded_test.go` — annotated-int regression (both repaired registration sites) + instrumented-callback invocation-count tests for both fused builtins (AC-3a, explicit short timeouts) (~80 LOC)
- `std/list.ail` — two exports + doc comments (~16 LOC)
- `examples/bounded_take_flatmap.ail` — new (~30 LOC) + manifest entry
- `docs/docs/reference/limitations.md` — new entry (~40 LOC); `docs/LIMITATIONS.md` — summary row (~5 LOC)
- `prompts/v0.16.2.md` — Common-Mistakes block (~8 LOC)
- `internal/diag/footguns.md` — one row; `internal/diag/footgun_fixtures_test.go` — 3 fixtures (~60 LOC)
- `internal/pipeline/` (exact file per sprint-planner; candidate `pipeline_module.go` warning channel) — diagnostic (~60 LOC)
- `CHANGELOG.md` / `changelogs/v0.18-current.md` — entry quoting the V1/V2 deltas

---

## Conflict Surface

This touches `internal/builtins/`, `std/`, `internal/pipeline/` (diagnostic), the teaching
prompt, and the limitations pages.

**Files a sprint edits, and who else is in flight there:**
- `std/list.ail` — **[m-parmap-effectful](v0_30_0/m-parmap-effectful.md)** (planned) intends
  to append `parMap`/`parMapN`/`parMapResult` here or create `std/par.ail` (its lines 191,
  255), and leans on the `std/list.ail:212-214` effectful-combinator contract comment that
  this doc also cites. Both changes are additive exports at different sites; textual conflict
  risk low, semantic conflict none — but whichever lands second rebases. No other planned doc
  touches `list_bounded.go` (V19).
- `internal/diag/footguns.md` + `footgun_fixtures_test.go` — owned by
  [m-diagnostic-coverage](../implemented/v0_29_0/m-diagnostic-coverage.md) (implemented; the
  table is live with 12 covered/inventoried rows, V15). We follow its fixture contract
  exactly; adding a row is the mechanism it was built for.
- `docs/docs/reference/limitations.md` — entry policy owned by M-V1-STABILITY-PROMISE
  (planned/v1_0_0): every entry needs a repro transcript + verified-at date. Our entry ships
  with the V1/V2 transcript at `v0.33.0-143-geabab0611`.
- `prompts/v0.16.2.md` — the footgun table's own "prompt lines to delete" column references
  this file; our block must be added in a way that Phase 3's diagnostic can later justify
  trimming (note the line range in the footgun row).
- `internal/pipeline/` — iteration 180's #616 work (parked, `D-10`) is probing
  `internal/pipeline/validate_effects.go` and `internal/types` row unification. Our
  diagnostic is a *note* in the module-warning channel, not an effect/type change; no shared
  lines expected, but if #616's third revision is authorized concurrently, coordinate on
  `internal/pipeline/` merge order.

**What other constructs live in the position we extend?**
- The `std/list` export namespace: `takeFlatMap` collides with nothing exported today
  (V4's IMP010 proves absence; `take` is the nearest name and the IMP010 suggester already
  bridges the two). `takeMap` — restored in round 2 — likewise collides with nothing:
  V16's full-file read of `std/list.ail` lists every export and `takeMap` is not among
  them. User modules defining their own `takeFlatMap`/`takeMap` shadow the import per
  normal scoping — no new mechanism.
- The `$builtin` registry: `_list_takeFlatMap`/`_list_takeMap` are already registered (V3);
  no registration-path change, so no golden builtin-count snapshot churn beyond metadata.
- The diagnostic's position: check output currently carries MOD010-class warnings; adding a
  note must not flip exit codes — fixture asserts `rc=0` with the note present.

**Programs that MUST still work unchanged:** direct `_list_takeFlatMap` literal calls
(arm_d, V2 — external code may already use the underscore name); `take(n, flatMap(f, xs))`
itself (still correct on small inputs — it only gains a note); every `std/list` consumer
(`make verify-examples`: 187 modules green at base, V20); `sortBy`, which calls `take`
internally at `std/list.ail:78` (must NOT trigger the note — it has no flatMap inside).

**What deliberately changes:** the `Int` → `int` constructor case in the two builtin type
makers. Any hypothetical caller depending on the *broken* behaviour would have to be passing
a value typed with a user-defined `Int` constructor into the count slot — no such caller can
exist in `std/` or `examples/` (verify-examples green at base with zero uses, V4/V20).

---

## Acceptance Criteria

Every AC names its command, its **measured baseline on the unmodified worktree**, and the
mutation that kills it. (Instrument note, inherited from iteration 180: `ailang check`
caches by content and hides passing arms on re-runs — every check-based AC must run on a
fresh file or state the run is first-touch.)

- **AC-1 (surface).** `./bin/ailang check <fresh file importing std/list (takeFlatMap)>`
  exits 0, AND a second fresh file importing `std/list (takeMap)` **also exits 0** —
  reversed from round 1, which asserted the `takeMap` import must still fail; round 2's
  measured amplification (V25/V26) restored the export. **Baseline: rc=1, `IMP010: symbol
  'takeFlatMap' not exported` (V4); `takeMap` likewise absent from `std/list.ail`'s export
  list (V16, full-file read).** Killed by: dropping either export.
- **AC-2 (type fix).** The arm_w wrapper file — restored to its full shape with **both**
  annotated wrappers (`n: int`, Appendix A) — runs; its `takeFlatMap` half prints `[1, 2]`
  and its `takeMap` half's expected prefix is asserted alongside. **Baseline: type error
  `cannot unify type constructors: Int vs int` (V5).** Killed by: reverting either
  `T.Int()` site (`makeTakeFlatMapType` or `makeTakeMapType`). The paired red-at-base Go
  regression in `list_bounded_test.go` exercises **both** repaired sites through an
  annotated `int`, so each site is also killed at the Go layer.
- **AC-3 (the unvisited-outputs term is gone — direct instrument; replaced in round 2 per
  objection 3's proposed fix, applied verbatim).** Two-layer:
  (a) CI: a **fast Go test using an instrumented callback and a small fixed input** (a
  counting `f` over a fixture-sized source; explicit short test timeout). It asserts that
  `_list_takeFlatMap` returns the expected prefix and invokes `f` only through the first
  inputs needed to produce n outputs, while an unfused reference invokes `f` for every
  input; the same test shape covers `_list_takeMap`. **Named mutation this kills:**
  re-implementing `takeFlatMap` as sugar for `take(n, flatMap(f, xs))` (likewise `takeMap`
  as `take(n, map(f, xs))`) — output stays byte-identical under that mutation, but `f`
  runs on every input, so the invocation count fails deterministically. The invocation
  count is a **better instrument than an allocation ratio for this mutation, because it
  is a direct observation of the mechanism — the unvisited-outputs term (`f` never runs
  on unvisited inputs) — rather than a proxy for it**: it is GC-independent,
  input-size-independent, and runs in milliseconds. (Round 1's version of this AC asserted
  a TotalAlloc ratio ≥ 20× justified by RSS figures — a category error, since peak
  residency is not cumulative allocation; no allocation figure in this doc is derived from
  RSS, and no threshold ships unmeasured. If an allocation gate is still desired later,
  the path is: first add a Verification Log row measuring the exact reduced fixture across
  repeated isolated runs, then choose a threshold supported by those measurements — not by
  RSS.) **Baseline:** the existing builtin suite is green at base (V17); this new test's
  job is to pin the early-exit mechanism explicitly against the named mutation, which no
  output-equality test can detect.
  (b) Release evidence (non-CI, retained per the proposed fix): `/usr/bin/time -l` on the
  rewritten repros recorded in the CHANGELOG — fused `takeFlatMap` arm < 150 MB max RSS
  and < 1 s against the banked 425 MB / 81.9 s baseline (V1/V2), and the `takeMap` pair
  against its measured 5.5× / 235× baseline (V25/V26). Peak RSS stays a release-evidence
  instrument, not a CI gate.
- **AC-4 (semantics parity).** An `.ail` integration test asserts
  `takeFlatMap(n, f, xs) == take(n, flatMap(f, xs))` for pure `f` across n=0, n>total,
  empty inner lists, and the dedup composition (V18's shape) — AND the restored parity arm
  `takeMap(n, f, xs) == take(n, map(f, xs))` for pure `f` across the same edge cases.
  **Baseline: inexpressible (AC-1's imports fail).** Killed by: any off-by-one in the
  early-exit loops (e.g. `>` for `>=` at `list_bounded.go:190` yields n+1 elements).
- **AC-5 (docs land).** `grep -ci "takeFlatMap" docs/docs/reference/limitations.md
  prompts/v0.16.2.md internal/diag/footguns.md` ≥ 1 each, AND `grep -ci "takeMap"` ≥ 1 in
  each of the same three files (the allocating-`f` map half of the trap and its fused fix
  must be taught too — V25/V26; note `takeMap` is not a substring of `takeFlatMap`, so the
  greps are independent). **Baseline: 0 / 0 / 0 (V13–V15, with per-file non-empty
  controls).** Killed by: shipping code without the teaching half — which is precisely how
  M-EVAL-BOUNDED-PIPELINE's fix stayed invisible for 5 months (V11/V12).
- **AC-6 (diagnostic, if Phase 3 ships).** Three fixtures in `footgun_fixtures_test.go`:
  the direct trap file → note code + substring `takeFlatMap`, rc still 0; the fused file →
  no note; a `take(n, map(...))` file → no note. Fixture 3 pins the **stated scope rule**
  (the note fires on the flatMap form only, because a syntactic pass cannot see whether
  `f` allocates — Phase 3) — it is NOT a claim that `take∘map` is safe; the allocating-`f`
  map case is a documented lint false negative carried by AC-5's teaching text (V25/V26).
  **Baseline: trap file checks fully clean — `✓ No errors found!` (V1's check run).**
  Killed by: removing the pass (fixture 1) or over-matching (fixtures 2–3).
- **AC-7 (regression gates, reach-verified).** `go test ./internal/builtins -run
  'TakeFlatMap|TakeMap' -count=1` and `make verify-examples` stay green. **Baselines: both
  green at base (V17: `ok 0.333s`; V20: 187 modules, and the pre-existing `1
  missing-on-disk` count must not grow).** These are anti-regression only — new-code reach
  is carried by AC-1..AC-6, so "suite green" cannot stand in for them.

---

## Testing Strategy

- **Unit (Go):** annotated-int regression covering both repaired registration sites
  (AC-2); instrumented-callback invocation-count tests for both fused builtins, small
  fixed inputs, explicit short timeouts (AC-3a — replaced in round 2; no allocation-ratio
  test ships without a measured threshold, per objection 3's fix); existing 22
  `list_bounded_test.go` tests stay green (V17).
- **Integration (.ail):** parity suite for both fused/unfused pairs (AC-4); the example
  file under verify-examples.
- **Diagnostic fixtures:** per `internal/diag` convention — code + fix-substring assertions,
  positive and negative (AC-6). All check-based fixtures run on fresh content (cache hazard).
- **Manual, recorded:** the `/usr/bin/time -l` before/after table in the CHANGELOG (AC-3b).

---

## Non-Goals

- **General lazy evaluation / lazy lists.** A reviewer will ask why N fused primitives
  instead of one evaluation-order change, so argued, not assumed: (1) laziness changes *when
  and whether effects run* — `take(10, flatMap(f, xs))` under lazy lists means "run `f`
  until 10 outputs", a semantic shift that collides with A1/A3 and was **explicitly ratified
  against** by a human decision in M-EVAL-BOUNDED-PIPELINE ("Fused builtins vs general
  laziness … Decided: fused builtins"); (2) the measured trap surface does not justify it —
  the trap is the strict materialisation of unvisited outputs, structural for `flatMap` and
  real for `map` exactly when `f` allocates (V25/V26), so "N fused primitives" is in fact
  N=2, both already written and both now exported; (3) the cost would be interpreter +
  codegen + builtin dual-mode support (that doc's surface-area argument, unchanged since).
  Revisit only via the Iterator-Protocol design sketched there, which makes materialisation
  boundaries explicit types.
- **`takeFilter`, `_list_take` exposure, or further fused forms.** The governing principle,
  restated in round 2: **the minimal core is satisfied by an evidence bar, not a
  headcount.** `takeMap` clears the bar — 5.5× peak / 235× wall measured on the reviewer's
  own prescribed fixture (V25/V26) — and ships; `takeFilter` does not (`filter` retains
  the input's own elements and measured no amplification, V8), and `_list_take` (V3,
  registered and unused) has no motivating incident. Round 1's justification for cutting
  `takeMap` — "map has no amplification", read off V7 — was falsely generalized: V7's `f`
  is a non-allocating scalar, and the accurate statement is that the *production OOMs were
  specific to `flatMap`* while the map trap is real whenever `f` allocates. The corrected
  record lives in the Round 2 section. M-EVAL-BOUNDED-PIPELINE's "add more fused forms
  only when proven needed" stands — V25/V26 is what "proven needed" looks like.
- **`takeFlatMapE` / `takeMapE` (effectful twins).** Deferred with an argument — see design
  question 3. The type system enforces the deferral (V6): no user can wander into
  effect-order surprise via the pure form.
- **Fixing `concat`/`take`'s own recursion limits** (V9, and `take` at `std/list.ail:99` is
  still AILANG-recursive with `_list_take` sitting unused in the registry — same exposure
  pattern, no motivating incident). Future Work.
- **Streaming/chunked JSON decode** (Category B of the original OOM). Still separate
  (M-JSON-STREAMING, never routed).
- **Making the lint see aliases** (V10). Documented blind spot; a data-flow lint is not
  worth its weight for a note-level teaching diagnostic.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `Int`→`int` fix breaks an unknown external caller of the raw builtin | Low | Only literal-arg calls work today (V5), and literals unify with `int` too; the fix strictly widens. AC-7 regression gates. |
| Diagnostic site turns out not to have resolved-global access where planned | Med | The sprint-planner picks the hook with the elaborated AST in hand; Phase 3 is the pre-declared cut line (step 10) — Phases 1–2 alone close #617's primary ask. |
| Note fires on `sortBy` (internal `take`) or other stdlib-internal patterns | Med | Negative fixtures (AC-6); pattern requires a *direct* `flatMap` application in argument position, which `sortBy` lacks. |
| Prompt entry bloats the token budget the prompt team just fought to cut | Low | ≤8 lines, and the footgun row marks it deletable once the diagnostic is proven (the `internal/diag` doctrine). |
| `take(n, flatMap(...))` note annoys legitimate small-input users | Low | Note-level, never blocks; text states the unfused form is correct for small inputs. |
| `take(n, map(f, ...))` with an allocating `f` gets no note — the lint's stated false negative (Phase 3) | Med | The trap is taught where the "if `f` allocates" conditional CAN be stated (LIMITATIONS + prompt, AC-5), the fix (`takeMap`) is exported and importable (AC-1), and the footgun row records the blind spot. |

## Related Documents

- [m-eval-bounded-pipeline](../implemented/v0_29_0/m-eval-bounded-pipeline.md) — built the
  builtins this doc exposes (shipped v0.10.0, V12). **Distinction, per the duplicate gate:**
  that doc's delivered scope ended at the registry; its stdlib surface never existed, its
  compiler-note success criterion was never implemented (V11), and its type registration has
  the V5 bug. This doc is the missing last mile plus the repairs, not a re-build.
- [m-iterative-list-builtins](../implemented/v0_9_2/m-iterative-list-builtins.md) — the
  `std/list`→builtin delegation idiom Phase 1 copies (its A9 row is this doc's ancestor).
- [m-stdlib-xml-walk-perf](../implemented/v0_19_2/m-stdlib-xml-walk-perf.md) — second
  precedent; its `_xml_flatMapChildren` comment explicitly "mirrors `_list_takeFlatMap`'s
  iterative pattern", and it served the same downstream consumer as #617.
- [m-diagnostic-coverage](../implemented/v0_29_0/m-diagnostic-coverage.md) — the footgun
  table + fixture contract Phase 3 plugs into.
- [m-parmap-effectful](v0_30_0/m-parmap-effectful.md) — in-flight `std/list.ail` neighbour
  (Conflict Surface).
- [m-v1-stability-promise](v1_0_0/m-v1-stability-promise.md) — limitations entry policy.

## Future Work

- `takeFlatMapE` with explicit effect-prefix semantics, if demand materialises (own doc).
  (Round 1's "export `takeMap` behind a measured motivating case" entry is gone from this
  list because its bar was met — V25/V26 — and the export moved into scope; see Round 2.)
- Iterative `concat`/`take` delegation (`_list_take` already registered and unused) — same
  expose-the-existing-builtin pattern, pending a measured motivating case: the same
  evidence bar that admitted `takeMap` in round 2.
- `foldWhile` bounded reducers (carried from M-EVAL-BOUNDED-PIPELINE's future work).

---

## Appendix A — measurement arm sources

All arms `ailang check` clean at base (first-touch runs; V-log). `rep`/`upto` are the shared
scaffolding; `big` is 400,000 elements.

```ailang
-- shared scaffolding (identical in every arm)
pure func rep(k: int, x: int) -> [int] =
  if k <= 0 then [] else [x] ++ rep(k - 1, x)

pure func upto(n: int) -> [int] =
  if n <= 0 then [] else concat(upto(n - 1), [n])
```

| arm | body (inside `main() -> () ! {IO}`) |
|---|---|
| arm_a (V1) | `println(show(take(5, flatMap(\x. rep(2000, x), upto(200)))))` |
| arm_d (V2) | `println(show(_list_takeFlatMap(5, \x. rep(2000, x), upto(200))))` |
| arm_i (V4) | file imports `std/list (takeFlatMap)` |
| arm_w (V5/AC-2) | wrapper `pure func takeFlatMap[a, b](n: int, f: (a) -> [b], xs: [a]) -> [b] = _list_takeFlatMap(n, f, xs)` + `dedup`/`takeMap` calls |
| arm_e (V6) | passes `noisy: (int) -> [int] ! {IO}` to `_list_takeFlatMap` |
| arm_0/arm_m/arm_ff/arm_c (V7–V9) | `let big = _list_takeFlatMap(400000, \x. rep(2000, x), upto(200));` then `take(5, big)` / `take(5, map(\x. x + 1, big))` / `take(5, filter(\x. x > 0, big))` / `take(5, concat(big, [1]))` |
| arm_l (V10) | `let t = take; let g = flatMap; t(2, g(\x. [x, x], [1, 2, 3]))` |
| arm_x (V18) | `dedup(_list_takeFlatMap(4, \x. [x, x], [1, 2, 3]))` → `[1, 2]` |
| arm_q (V21) | controller round-1 arm: `_list_takeFlatMap`, literal cap 5, 3-element source, f(x) of 10 elements — everything small |
| arm_p (V22) | controller round-1 arm: literal cap 5, 3-element source, ONE f(x) of 500,000 elements (the fused-built list below) |
| arm_s (V23) | controller round-1 arm: literal cap 5, 500,000-element source (fused-built), f(x) of 1 element |
| discarded pair (V24) | earlier arms whose 500k list came from direct `rep(300000, …)` recursion — Go `fatal error: stack overflow` at 1.49 GB; the peak was the BUILDER's, so the pair was excluded and the cause recorded |
| fused round-2 arm (V25) | controller round-2 arm (objection 4's prescribed fixture): `_list_takeMap(5, \x. rep(2000, x), src())` — prints `5` |
| unfused round-2 arm (V26) | controller round-2 arm: `take(5, map(\x. rep(2000, x), src()))` — prints `5`, identical output |

The round-1 arms (V21–V23) build their 500,000-element list via
`_list_takeFlatMap(500000, \x. rep(1000, x), upto(500))` so that no clean arm relies on
deep AILANG recursion — the discarded pair is the counterexample that forced this rule:
a recursive builder's stack/peak would otherwise be mis-credited to the primitive under
measurement. The round-2 arms (V25/V26) follow the same rule: their shared `src()` is a
2,000-element list built via `_list_takeFlatMap(2000, \x. rep(100, x), upto(50))`, literal
cap of 5 in both arms, zero stack overflows asserted in both.

---

**Document created**: 2026-08-12
**Last updated**: 2026-08-12 (round-2 revision: both quorum objections accepted with the
reviewers' own fixes applied verbatim — AC-3 replaced with the instrumented-callback
invocation-count instrument [objection 3], and the `takeMap` export RESTORED after the
reviewer-prescribed measurement supplied the motivating case round 1 demanded [objection 4,
V25/V26]. Round-1 revision: cost model corrected and measured [V21–V24], `takeMap` export
dropped — since reversed.)

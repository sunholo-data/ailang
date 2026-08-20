# LC-1 `m-list-repr-spike` — M6 report: full matrix, kill-criterion arithmetic, and the programme verdict

> **Every number below was produced by `tools/internal/spike-listrep/cmd/matrix` in one pass and
> is reproducible from `tools/internal/spike-listrep/testdata/m6-matrix.json`, which records each
> cell's exact command argv, the raw stdout line the number was parsed from, and all five trials.
> This document is GENERATED from that JSON — no value in it was transcribed by hand.**

## Run header

| field | value |
|---|---|
| Go | `go1.26.6` |
| Platform | `darwin/arm64` |
| Machine | `Voights-Mac-Studio.local` |
| Started | `2026-08-20T22:55:50Z` |
| Elapsed (full pass) | **11m21.603622708s** |
| AC-1 measured points | **76** |
| B-LEN points (outside AC-1) | **8** |
| Trials recorded | **420** |
| `rig.lock` before / after | free / free (observed, **never acquired** — it is a GPU mutex and this spike never touches the GPU) |

The plan estimated ~30–90 minutes for a full pass. The measured elapsed is well under that: the
five trials of a cell share a warm build cache, so the marginal cost of a `go test` trial is ~1.6 s
rather than the ~8 s a cold single invocation costs. The two `go run` rows (B6, B8) dominate what
remains, at ~9 s per trial, because `go run` relinks every time.

## 1. AC-2 — the control leg, which is what makes this gate falsifiable

The kill criterion is only worth running if the instrument can see the failure it gates on. Clause
(a) requires the **C0 control** to show ≥ 8× on the same within-arm L-ratio the candidates must keep
under 1.5×. If it does not, **no verdict may be emitted at all** — and the driver enforces that as a
refusal (`control_leg_failed`), not as a note.

| m | C0 `time(L=16384)/time(L=1024)` paired ratios | median | ≥ 8 |
|---|---|---:|---|
| 1024 | 9.854, 9.985, 9.429, 10.044, 9.547 | **9.854** | ✅ |
| 4096 | 10.384, 10.307, 10.713, 10.389, 10.300 | **10.384** | ✅ |

Absolute cost of the heaviest control cell, for scale: C0 at m=4096, L=16384 runs at
**54.8 ms/op** against
**73 µs/op** for C1 — the
quadratic behaviour `#676` reports, reproduced in the harness.

## 2. The kill criterion, clause by clause

Thresholds are the doc's ratified literals (doc:185-240) and none was altered. `(a)` and `(d)` are
**within-arm** ratios; `(b)` and `(c)` are **cross-arm** against C0. Every median is the median of
five ordinal-paired ratios. **No clause fired the tie/spread rerun** (`rerun_required` is false
throughout), so every verdict below rests on the initial five-trial batch.

### C1 — **PASS**

| clause | observable | threshold | measured median(s) | verdict |
|---|---|---|---|---|
| (a) | B1 within-arm `t(L=16384)/t(L=1024)`, m ∈ {1024, 4096} | ≤ 1.5 | 1.2416, 1.2229 | ✅ |
| (b) | B3 ns/element ÷ C0, n ∈ {4096, 65536} | ≤ 2.0 | 1.1065, 0.9496 | ✅ |
| (c) | B6 B/element ÷ **measured** C0 B/element | ≤ 2.5 | 1.9524 | ✅ |
| (d) | B-LEN within-arm `Len()` n=65536 ÷ n=4096 | ≤ 1.2 | 0.9987 | ✅ |
| (e) | encapsulation feasibility (3 legs, §4) | — | `pass` | ✅ |

### C2K8 — **PASS**

| clause | observable | threshold | measured median(s) | verdict |
|---|---|---|---|---|
| (a) | B1 within-arm `t(L=16384)/t(L=1024)`, m ∈ {1024, 4096} | ≤ 1.5 | 1.0386, 1.0445 | ✅ |
| (b) | B3 ns/element ÷ C0, n ∈ {4096, 65536} | ≤ 2.0 | 1.1087, 1.1067 | ✅ |
| (c) | B6 B/element ÷ **measured** C0 B/element | ≤ 2.5 | 1.3404 | ✅ |
| (d) | B-LEN within-arm `Len()` n=65536 ÷ n=4096 | ≤ 1.2 | 1.0014 | ✅ |
| (e) | encapsulation feasibility (3 legs, §4) | — | `pass` | ✅ |

### C2K32 — **PASS**

| clause | observable | threshold | measured median(s) | verdict |
|---|---|---|---|---|
| (a) | B1 within-arm `t(L=16384)/t(L=1024)`, m ∈ {1024, 4096} | ≤ 1.5 | 1.0138, 1.0303 | ✅ |
| (b) | B3 ns/element ÷ C0, n ∈ {4096, 65536} | ≤ 2.0 | 1.0857, 1.0810 | ✅ |
| (c) | B6 B/element ÷ **measured** C0 B/element | ≤ 2.5 | 1.0699 | ✅ |
| (d) | B-LEN within-arm `Len()` n=65536 ÷ n=4096 | ≤ 1.2 | 0.9982 | ✅ |
| (e) | encapsulation feasibility (3 legs, §4) | — | `pass` | ✅ |

### Clause (c)'s denominator is MEASURED, never the 16 B derivation

The doc is explicit that if the measurement disagrees with the derivation, the measurement wins.
It does not disagree — which is itself worth recording, because it is the assumption the whole
memory case rests on.

| arm | B6 B/element (5 trials) | median | ÷ C0 |
|---|---|---:|---:|
| C0 | 16.36, 16.42, 16.46, 16.47, 16.29 | **16.418** | 1.0000 |
| C1 | 32.00, 32.05, 32.05, 32.00, 31.95 | **32.001** | 1.9491 |
| C2K8 | 22.06, 22.01, 22.06, 22.06, 22.11 | **22.056** | 1.3434 |
| C2K32 | 17.62, 17.57, 17.57, 17.61, 17.51 | **17.570** | 1.0701 |

C0's measured **16.418 B/element** against the doc's ~16 B derivation (doc:209-211) — a
**2.6%** gap, consistent with the superseded doc's 3.2% RSS agreement (V14).

## 3. B8 — GC shape, and the parser that had never read it

`-kind=gcshape` returned `cannot unmarshal string into Go value of type float64` on **every** trial
since it was written: the metric parser unmarshalled into `map[string]float64` while the gcshape
report carries a string `arm` beside its counters. No kill clause reads B8, so the only thing that
ever asked for it was AC-1's completeness floor — which is where it surfaced, ten minutes into the
first full pass. Fixed, pinned, and the numbers it unblocked are not a footnote:

| arm | `num_gc_delta` over the n=12800 workload |
|---|---|
| C0 | 407, 426, 417, 411, 419 |
| C1 | 0, 0, 0, 0, 0 |
| C2K8 | 0, 0, 0, 0, 0 |
| C2K32 | 2, 2, 2, 2, 2 |

C0 triggers ~**417** collections where C1 and C2(K=8) trigger **zero** and C2(K=32) triggers **2**.
That is the allocation-volume half of `#676` — the quadratic copying is what feeds the collector —
and it is a **secondary observation, not a kill clause**: no threshold was defined for it in advance,
so it is reported and not adjudicated.

## 4. AC-4 — clause (e), all three legs

**Leg 1 — unexported fields.** `SliceList{value}`, `ConsList{head,tail,n}`, `ChunkList{elems,tail,
total,k}`: zero exported field names. Control: the same matcher finds **7** exported fields on
`protocol.Trial`, so the empty result is a measurement rather than a broken pattern.

**Leg 2 — every benchmark compiles in an external `_test` package.** All 8 `*_test.go` files in the
spike read `package spikelistrep_test`; none needs a raw field.

**Leg 3 — field writes confined to constructors.** The regex the plan originally carried matched
`==` (because `=` is a prefix of `==`) and returned 3 hits on correct code, all comparisons outside
any constructor — so read literally it **failed on a tree with zero violations**. Iteration 238
corrected it to the three-part form used here:

```
(i)   assignment-only sweep  -> 0 hits   (for THIS clause an empty result is the PASS)
(ii)  deliberate probe       -> 1 hit    (proves the instrument fires; (i) cannot prove itself)
(iii) constructor check      -> 3 hits   (&SliceList{ &ConsList{ &ChunkList{ — one per file)
```

(iii) is a separate check because constructor writes are **composite literals**, which use `:` and
are invisible to any `=`-shaped grep. Clause (e) is therefore **PASS** for all three candidates.

## 5. Verdict (AC-5)

### **GO** — chosen representation: **C2K32**

All three candidates satisfy all five clauses. The doc's procedure breaks ties by **(c) then (b)**:

| candidate | (c) B/element ratio | (b) worst-n ratio | rank |
|---|---:|---:|---|
| C2K32 | 1.0699 | 1.0810 | 1 ← chosen |
| C2K8 | 1.3404 | 1.1067 | 2 |
| C1 | 1.9524 | 0.9496 | 3 |

The verdict is a **GO**: LC-2…LC-5 are unblocked and `D-19` is not re-opened.

### ⚠ The winner is not the representation the remaining ~16 person-days were estimated against

This is the one result that needs a human eye rather than a table. The decomposition
(`m-list-cons-cells-decomposition.md`, 15.5–21.5 person-days across LC-2…LC-5) was scoped around
**plain cons cells — C1**. The doc's own tie-break selects **C2(K=32)**, the chunked hybrid, because
its per-element memory is 1.070× C0 against C1's
1.952×.

Both readings are defensible and the spike deliberately does not choose between them:

- **Take the tie-break literally → C2(K=32).** It is the doc's ratified rule, and the memory margin
  is real: 17.6 vs 32.0 B/element. But LC-4 then implements a chunked structure with a leading-chunk
  copy, which is more code than the cons cells LC-2…LC-5 were estimated against, and every estimate
  downstream is against the wrong shape.
- **Take the cheapest passing candidate → C1.** It passes all five clauses with margin — (c) at
  1.95× against a 2.5× ceiling — and it is what the ~16 days were planned for. The tie-break exists
  to pick a winner when several pass, not to force the most complex one.

**This is a scope decision, not a measurement, so it is surfaced rather than settled.** The
measurements that inform it are all above; nothing in the matrix distinguishes the two on
correctness, and both clear every threshold.

## 6. What this report does NOT claim

- **Platform.** Every number is `darwin/arm64` on one machine. The ubuntu and windows CI legs
  *compile* the spike and never run its benchmarks; no cross-platform number is claimed.
- **B8 is not adjudicated.** No threshold was predeclared for GC shape, so it is reported only.
- **B7** is read off `-benchmem` on B1/B2 and contributes no new runs, per the plan's §5.
- **No rerun fired**, so the tie/spread path is exercised by tests rather than by this data. That
  path was repaired in this milestone precisely because it had never been exercised: the driver
  flagged a rerun and never performed it, which would have banked a straddling clause as a FAIL.

## M-STDLIB-XML-WALK-PERF — Sprint Plan

**Sprint ID:** `M-STDLIB-XML-WALK-PERF`
**Design Doc:** [m-stdlib-xml-walk-perf.md](./m-stdlib-xml-walk-perf.md)
**Target Version:** v0.21.0
**Estimated Duration:** 2 days (~12 hours)
**Risk Level:** Low
**Total LOC Estimate:** ~470 LOC (impl + tests + example)

## Summary

Add four `std/xml` builtins to eliminate the dominant FFI/allocation cost of `XmlNode` tree walks, addressing the bottleneck profiled by `sunholo/ailang-parse` (21,651 function calls per 79 KB page, dominated by `getChildren`/`getTag`/`flatMap` round-trips). All four operate on the existing `XmlNode` ADT — `std/html` callers get the win for free.

**Deliverables:**
- `foldChildren`, `foldChildrenStep`, `getAttrMap`, `nodeKind` builtins
- `NodeKind` ADT in `std/xml`
- Benchmarks proving ≥30 % wallclock improvement on a representative walker
- Example, docs, CHANGELOG

## Current Status Analysis

**Velocity baseline (last 14 days):**
- M-STDLIB-HTML (v0.19.1, comparable scope: stdlib addition + tests + docs): ~590 LOC, shipped in one commit cycle, ~2 days.
- M-TYPECHECK-NO-AUTO-UNWRAP-RESULT (v0.20.0, currently in flight): typecheck-side work, milestone-by-milestone cadence.

**Confidence:** High. This sprint is shaped like M-STDLIB-HTML (additive stdlib + tests + docs, no compiler changes). 2-day estimate has direct precedent.

**Blockers:** None. All prior art exists in `internal/builtins/xml_fold.go` (FnCallerN wiring), `internal/builtins/xml.go:680` (getChildren reference), `internal/builtins/map.go:97` (MapValue construction).

## Proposed Milestones

### M1 — Builtins + Module Surface (~6 hours, ~190 LOC)

**Implementation:**
- [ ] `internal/builtins/xml_query.go`: add four register/impl pairs (`_xml_foldChildren`, `_xml_foldChildrenStep`, `_xml_getAttrMap`, `_xml_nodeKind`). Reuse FnCallerN wiring from `xml_fold.go`. Reuse `*eval.MapValue` from `map.go`. ~150 LOC.
- [ ] `std/xml.ail`: add `export type NodeKind = Element | Text | Comment`. Add four `export pure func` declarations. Import `FoldStep` from `std/iter`, `Map` from `std/map`. ~20 LOC.
- [ ] Wire all four `register*` calls into `xml_query.go`'s `init()`.
- [ ] `make build && make test` clean (existing tests must pass — no regression).

**Files touched:**
- `internal/builtins/xml_query.go` (+150 LOC, currently 309 → ~459 LOC, still under 800)
- `std/xml.ail` (+20 LOC)

**Acceptance criteria:**
- All four builtins registered in the builtin registry (`make build` shows no duplicate/missing).
- `std/xml` smoke test still passes.
- Manually: `ailang repl` can `import std/xml (foldChildren, getAttrMap, nodeKind, NodeKind)` without error.

**Risk:** Low. Direct copy from `xml_fold.go` plus a trivial map walker.

---

### M2 — Tests + Benchmarks (~4 hours, ~250 LOC)

**Implementation:**
- [ ] `internal/builtins/xml_walkperf_test.go` (new file, ~250 LOC).
- [ ] Unit tests:
  - `foldChildren` on Element / Text / Comment / empty-children / non-Element
  - `foldChildrenStep` early-stops on `Stop(a)`, runs to end on all `Continue(_)`
  - `getAttrMap` on Element with attrs / duplicate-name attrs (last-write-wins) / non-Element (empty map)
  - `nodeKind` returns correct constructor for each ctor
  - Handler error propagation through `FnCallerN`
- [ ] Benchmarks (same file):
  - `BenchmarkXmlWalk_Classic` (flatMap + getChildren) vs `BenchmarkXmlWalk_FoldChildren` over a synthetic 1,900-node tree
  - `BenchmarkAttr_PerAttr_7x100` vs `BenchmarkAttr_AttrMap_7x100`
- [ ] Run `go test ./internal/builtins -run XmlWalkPerf -bench Walkperf -benchmem` and capture numbers.

**Files touched:**
- `internal/builtins/xml_walkperf_test.go` (new, ~250 LOC)

**Acceptance criteria:**
- All unit tests pass.
- `BenchmarkXmlWalk_FoldChildren` shows ≥30 % wallclock improvement and lower B/op vs `BenchmarkXmlWalk_Classic`.
- `BenchmarkAttr_AttrMap_7x100` shows lower wallclock and B/op vs per-attr variant.
- Numbers captured for CHANGELOG.

**Risk:** Medium — if `FnCallerN` overhead dominates, `foldChildren` may not hit 30 %. Mitigation: if missed, profile the gap and document the actual speedup (still wins on allocations).

---

### M3 — Example + Docs + CHANGELOG (~2 hours, ~50 LOC + docs)

**Implementation:**
- [ ] `examples/runnable/xml_walk_perf.ail` (~50 LOC) — same walker written classic + foldChildren, prints both timings via `clock.now()`.
- [ ] `docs/docs/reference/stdlib/std-xml.md` — add four function entries + "Cost model" subsection.
- [ ] `CHANGELOG.md` — v0.21.0 entry quoting benchmark deltas.
- [ ] `ailang messages send` ack reply to inbox msg `cd45490b` (sunholo/ailang-parse) with one-liner rewrite pattern.

**Files touched:**
- `examples/runnable/xml_walk_perf.ail` (new)
- `docs/docs/reference/stdlib/std-xml.md` (+~50 LOC)
- `CHANGELOG.md` (+~30 LOC)

**Acceptance criteria:**
- `ailang run examples/runnable/xml_walk_perf.ail` runs end-to-end and prints both timings.
- Docs page mentions all four new functions, their cost models, and the foldChildren-vs-classic pattern.
- CHANGELOG entry includes the captured benchmark numbers.
- Ack reply sent.

**Risk:** Low. Pure additive work.

## Day-by-Day Breakdown

**Day 1 (~6 hours):**
- Morning: M1 (builtins + module surface). Reach `make build && make test` clean.
- Afternoon: M2 first half — unit tests for all four builtins. Cover edge cases.

**Day 2 (~6 hours):**
- Morning: M2 second half — benchmarks. Run and capture numbers. Iterate if speedup is below 30 %.
- Afternoon: M3 — example, docs, CHANGELOG, ack reply. Final `make ci` clean.

## Success Metrics

- [ ] All four builtins registered and importable from `std/xml`.
- [ ] `NodeKind` ADT importable and pattern-matchable.
- [ ] All unit tests pass (`make test`).
- [ ] Benchmark shows ≥30 % wallclock improvement on `foldChildren` walker.
- [ ] `getAttrMap` benchmark shows crossover at N≥2 attrs.
- [ ] `examples/runnable/xml_walk_perf.ail` runs and demonstrates the speedup.
- [ ] No regression in `std/xml`, `std/html`, or other `examples/runnable/*.ail`.
- [ ] CHANGELOG entry shipped with benchmark numbers.
- [ ] Docs page updated.
- [ ] Inbox ack sent to `cd45490b`.

## Dependencies

**None.** All required infrastructure exists:
- `FnCallerN` evaluator wiring — proven in `xml_fold.go`
- `*eval.MapValue` — used throughout `map.go`
- `*eval.TaggedValue` for ADTs — used throughout `xml.go`
- `FoldStep` — already imported in `xml_fold.go`'s callers

## Open Questions

None blocking. The design doc froze:
- `foldChildren` ships with `foldChildrenStep` companion
- `getAttrMap` returns `Map[string, string]` (last-write-wins for duplicates)
- `NodeKind` is a new ADT, not a stringly-typed builtin
- No `foldDescendants` in v1

## Risk Summary

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| `FnCallerN` overhead caps the speedup below 30 %. | Med | Profile if missed. Even at 20 % wallclock, the alloc savings make the builtin worthwhile. Adjust CHANGELOG claim to match measured numbers. |
| Map last-write-wins surprises a caller. | Low | Document explicitly. Existing `Element` ADT preserves source order for callers who need it. |
| File-size guidance pressure on `xml_query.go`. | Low | Lands at ~460 LOC, well under 800. If concerned, split walk-perf builtins into `xml_walkperf.go`. |

## Handoff

This plan is ready for sprint-executor. JSON progress file at `.ailang/state/sprints/sprint_M-STDLIB-XML-WALK-PERF.json`.

---

**Plan created:** 2026-05-14

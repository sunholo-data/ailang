# M-PARSEFOLD-EARLY-TERMINATION: Short-Circuit Support for `parseFold` / `zipXmlScanFold`

**Status**: Implemented (v0.11.3)
**Target**: v0.11.3 (or v0.12.0 if paired with broader XLSX perf work)
**Priority**: P1 (High — unlocks XLSX workloads where only a prefix is needed)
**Estimated**: 1 day (~6 hours implementation + testing)
**Dependencies**: M-STREAMING-ZIP-XML (implemented, v0.11.0)
**Bug Report**: ailang-parse msg `80f142f6` (2026-04-10)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Early termination is deterministic — same input, same stop point |
| A2: Replayability | 0 | Traces record exact elements folded; replay produces identical result |
| A3: Effect Legibility | +1 | `Stop(acc)` vs `Continue(acc)` makes the termination boundary explicit in the program text |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Termination contract (`maxElements`) is machine-checkable |
| A6: Safe Concurrency | 0 | Single-threaded fold unchanged |
| A7: Machines First | +2 | AI-generated code frequently needs "first N matching" semantics — currently must scan whole document |
| A8: Minimal Syntax | 0 | Library-level — no new syntax |
| A9: Cost Visibility | +3 | Enables O(prefix) cost vs O(document); matches the existing cost-visibility wins from M-STREAMING-ZIP-XML |
| A10: Composability | +1 | `Continue`/`Stop` composes with any accumulator type |
| A11: Structured Failure | 0 | Result propagation unchanged |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism
- [x] A3 (Effects): Termination boundary explicit
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Directly addresses an AI-agent pain point

## Problem Statement

After M-STREAMING-ZIP-XML shipped (v0.11.0), memory usage for large XML files dropped
from O(document) to O(element). However, **runtime is still O(document)** because the
fold cannot stop mid-scan.

### Concrete Repro (from msg 80f142f6)

ailang-parse's XLSX parser needs the first 5,000 rows of a 50,000-row sheet
(~7 MB of XML). Today:

```ailang
-- Scans ALL 50K rows; accumulator stops growing after 5K but fold continues.
let rows = zipXmlScanFold(zip, "xl/worksheets/sheet1.xml", "row", [],
  \acc, node.
    if length(acc) >= 5000 then acc            -- "stop" here, but we keep scanning
    else append(acc, parseRow(node))
  )
```

**Measured cost:** 5.5 seconds for a fold that *logically* should complete after
extracting the first 10–15% of elements. The handler is invoked on every remaining row
(the `length(acc) >= 5000` check runs 45,000 times after the logical stop point).

### Why the Current API Can't Short-Circuit

The handler signature is `(acc, node) -> acc`. There is no way for the handler to
signal "I'm done" — the driver calls the handler for every matching element
unconditionally.

### Affected Workloads

- Large XLSX sheets (primary use case, msg 80f142f6)
- DOCX bodies where only first N sections are needed (preview/summary)
- EPUB table-of-contents extraction (first few `<nav>` elements out of thousands)
- Log file parsing (first N error records)

This is **not** a niche optimization — "bounded prefix scan" is one of the most common
parsing patterns across AI-generated code.

## Goals

**Primary Goal:** Allow `parseFold` / `zipXmlScanFold` to terminate early without
scanning the rest of the document.

**Success Metrics:**
- First-5K-of-50K-rows workload from msg 80f142f6: **~0.55s** (vs. 5.5s today, 10x improvement)
- Memory: unchanged from M-STREAMING-ZIP-XML (still O(element))
- No regression on full-fold workloads (`sharedStrings`, full-sheet extraction)
- New `examples/bounded_xml_fold.ail` demonstrating both `Stop`/`Continue` and `maxElements` APIs

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **API: sentinel ADT (`Stop/Continue`) vs separate `*Limit` builtin** | Sentinel is more general (can stop on any condition); `*Limit` is simpler but covers only the count case | human | design | high |
| Introduce new ADT (`FoldStep`) vs reuse `Option`/`Result` | A dedicated ADT is clearer in program text; reusing `Option` muddles semantics | human | design | med |
| Whether to also provide `parseFoldChildren` (tied to m-perf6 P3) | Bundles two perf wins but expands scope | agent | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **API**: Sentinel ADT `FoldStep[a] = Continue(a) | Stop(a)`. Handler returns `FoldStep[Acc]`. Driver stops scanning when it sees `Stop(_)`. This is more expressive than a raw `maxElements` limit (users can stop on any predicate) and composes trivially with existing folds.
- [ ] **Also ship `parseFoldLimit(xml, tag, n, init, f)` as a convenience wrapper** — it's the most common case and saves users from wrapping `Continue`/`Stop` manually.
- [ ] **Do NOT break the existing `parseFold` signature.** Existing callers keep working unchanged. The new primitive is `parseFoldStep` (returns `FoldStep[Acc]`).
- [ ] `parseFoldChildren` is **out of scope** for this doc — tracked under m-perf6 Phase 4.

## Solution Design

### API Overview

Three surface APIs, layered:

```ailang
-- 1. NEW PRIMITIVE: sentinel-based early termination
export type FoldStep[a] = Continue(a) | Stop(a)

zipXmlScanFoldStep(
  zip: ZipReader,
  path: string,
  tag: string,
  init: a,
  f: (a, XmlNode) -> FoldStep[a]
) -> Result[a, string] ! {FS}

-- 2. CONVENIENCE: count-bounded wrapper
zipXmlScanFoldLimit(
  zip: ZipReader,
  path: string,
  tag: string,
  maxN: int,
  init: a,
  f: (a, XmlNode) -> a
) -> Result[a, string] ! {FS}

-- 3. UNCHANGED: existing full-scan fold (no early termination)
zipXmlScanFold(...)   -- existing signature preserved
```

Same three-layer pattern applies to the pure-string `parseFold` family:
`parseFold` → `parseFoldStep` + `parseFoldLimit`.

### Architecture

Both `zipXmlScanFoldStep` and `parseFoldStep` are Go builtins. Core logic:

```go
// Pseudocode
for element := range xmlDecoder.stream() {
  if element.matches(tag) {
    result := f(acc, element)
    switch result := result.(type) {
    case Continue:
      acc = result.value
    case Stop:
      return Ok(result.value)   // exit loop
    }
  }
}
return Ok(acc)
```

`zipXmlScanFoldLimit` is implemented as a thin wrapper:

```go
func zipXmlScanFoldLimit(zip, path, tag, maxN, init, f) {
  count := 0
  return zipXmlScanFoldStep(zip, path, tag, init, func(acc, node) FoldStep {
    if count >= maxN { return Stop(acc) }
    count++
    return Continue(f(acc, node))
  })
}
```

### Components

1. **`FoldStep[a]` ADT** (`stdlib/std/iter.ail`, ~10 LOC)
   - Two constructors: `Continue(a)` and `Stop(a)`
   - Exported from `std/iter` (new module) or added to `std/list`

2. **Builtin: `zipXmlScanFoldStep`** (`internal/builtins/xml.go`, ~60 LOC)
   - Adapt existing `zipXmlScanFold` implementation
   - Branch on handler return value; exit on `Stop`

3. **Builtin: `parseFoldStep`** (`internal/builtins/xml.go`, ~40 LOC)
   - Pure-string variant, same pattern

4. **Builtin wrappers: `zipXmlScanFoldLimit`, `parseFoldLimit`** (~30 LOC)

5. **Examples + tests** (~120 LOC)
   - `examples/bounded_xml_fold.ail` — first-N rows, sentinel predicate
   - `internal/builtins/xml_foldstep_test.go` — correctness, early exit, count-limit

### Implementation Plan

**Phase 1: ADT + primitive** (~3 hours)
- [ ] Add `FoldStep[a]` to `stdlib/std/iter.ail` (new file) or inline in `std/list`
- [ ] Implement `zipXmlScanFoldStep` builtin — adapt existing `zipXmlScanFold` loop
- [ ] Implement `parseFoldStep` builtin — pure-string variant
- [ ] Unit tests: early stop triggers after N elements, final `acc` value correct

**Phase 2: Convenience wrappers** (~1 hour)
- [ ] `zipXmlScanFoldLimit` and `parseFoldLimit`
- [ ] Unit tests: wraps correctly, counts match maxN

**Phase 3: Examples + benchmark** (~1 hour)
- [ ] `examples/bounded_xml_fold.ail` — 5K-of-50K rows pattern
- [ ] Benchmark: poi_many_merges.xlsx first-5K-rows with old API vs new API
- [ ] CHANGELOG under **v0.11.3 → Performance**

**Phase 4: Teaching prompt update** (~30 min)
- [ ] `prompts/` — document `FoldStep` pattern, when to use `*Limit` vs `*Step`
- [ ] Update `ailang docs std/iter` if new module

### Files to Modify/Create

**New:**
- `stdlib/std/iter.ail` or extension to `std/list.ail` (~10 LOC)
- `internal/builtins/xml.go` — four new builtins (~130 LOC)
- `internal/builtins/xml_foldstep_test.go` (~100 LOC)
- `examples/bounded_xml_fold.ail` (~40 LOC)

**Modified:**
- `internal/builtins/registry.go` — register four new builtins
- `CHANGELOG.md` — entry under v0.11.3
- `prompts/*` — teaching prompt note

**Estimated: ~290 LOC (including tests)**

## Examples

### Example 1: First-N Rows (primary use case)

```ailang
-- Concise wrapper version:
let firstRows = zipXmlScanFoldLimit(zip, "xl/worksheets/sheet1.xml", "row", 5000, [],
  \acc, node. append(acc, parseRow(node)))

-- Explicit sentinel version (equivalent):
let firstRows = zipXmlScanFoldStep(zip, "xl/worksheets/sheet1.xml", "row", [],
  \acc, node.
    if length(acc) >= 5000 then Stop(acc)
    else Continue(append(acc, parseRow(node))))
```

### Example 2: Stop on Predicate (can't express with `maxN`)

```ailang
-- Find the first </heading> block; stop immediately when found.
let firstHeading = parseFoldStep(xml, "h1", None,
  \acc, node.
    match acc {
      Some(_) => Stop(acc),               -- already found, stop scanning
      None    => Stop(Some(node))          -- first match; stop
    })
```

### Example 3: Budget-Bounded Accumulation

```ailang
-- Stop when accumulated text exceeds 100 KB (token budget for an LLM).
let preview = parseFoldStep(xml, "p", "",
  \acc, node.
    let next = acc ++ textOf(node)
    if length(next) > 100000 then Stop(acc)
    else Continue(next))
```

## Success Criteria

- [ ] `FoldStep[a]` ADT available in stdlib with `Continue`/`Stop` constructors
- [ ] `zipXmlScanFoldStep` and `parseFoldStep` builtins registered and tested
- [ ] `zipXmlScanFoldLimit` and `parseFoldLimit` convenience wrappers shipped
- [ ] 5K-of-50K-rows benchmark: **≤0.6s** (vs 5.5s today)
- [ ] Sentinel predicate example (no count limit) works end-to-end
- [ ] `make test`, `make verify-examples` pass
- [ ] CHANGELOG entry under **Performance**
- [ ] Teaching prompt updated
- [ ] ailang-parse agent can remove the "scan everything and truncate" workaround

## Testing Strategy

**Unit tests:**
- Early stop: handler returns `Stop(x)` on first match → result is `Ok(x)`
- Late stop: handler returns `Stop(x)` on Nth match → no further handler invocations
- Full fold: handler always returns `Continue` → semantics identical to `zipXmlScanFold`
- Count wrapper: exactly `maxN` elements processed

**Integration tests:**
- `examples/bounded_xml_fold.ail` via `make verify-examples`
- Benchmark on poi_many_merges.xlsx (if available); otherwise synthetic 50K-element XML

**Regression:**
- Existing `zipXmlScanFold` callers unchanged

## Deferred Decisions

- Whether `FoldStep` lives in `std/iter` (new module) or `std/list` (existing) —
  implementation detail, agent may resolve
- Whether to also add `foldStep` for plain lists (not just XML) — deferred; existing
  `takeMap`/`takeFlatMap` (m-eval-bounded-pipeline) cover the list-level case
- Whether `parseFoldChildren` (child-element fold without materializing list) ships
  here or in m-perf6 — **deferred to m-perf6 Phase 4**

## Non-Goals

- Parallel fold — single-threaded, in-order semantics preserved
- Fold over multiple tags simultaneously — out of scope
- Lazy list traversal in general (`takeMap`/`takeFlatMap` cover list case — see
  m-eval-bounded-pipeline.md)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `FoldStep` name collides with user types | Low | Namespaced under `std/iter` or `std/list`; users can alias |
| Streaming XML decoder can't cleanly abort mid-element | Low | Go's `xml.Decoder` supports clean early return via token loop exit |
| Benchmark doesn't show 10x gain | Medium | Profile before/after; if gain <5x, investigate whether XML parser itself is the bottleneck |
| Users misuse `Continue`/`Stop` (e.g., `Stop` inside `Continue` branch) | Low | Type system enforces `FoldStep[a]` return type; compile-time error on mismatch |

## Related Documents

**Implemented (prior art):**
- [design_docs/implemented/v0_11_0/m-streaming-zip-xml.md](design_docs/implemented/v0_11_0/m-streaming-zip-xml.md) — Streaming ZIP/XML fold primitive this extends
- [design_docs/implemented/v0_11_0/m-bytecode-xml-builtins.md](design_docs/implemented/v0_11_0/m-bytecode-xml-builtins.md) — XML builtin wiring in bytecode VM

**Planned (related):**
- [design_docs/planned/v0_11_0/m-eval-bounded-pipeline.md](design_docs/planned/v0_11_0/m-eval-bounded-pipeline.md) — `takeMap`/`takeFlatMap` list-level short-circuit (complementary, list vs XML)
- [design_docs/planned/m-perf6-runtime-hotspots.md](design_docs/planned/m-perf6-runtime-hotspots.md) — Phase 4 will add `parseFoldChildren` (avoids list materialization entirely)

## References

- Originating message: ailang-parse msg `80f142f6` (`ailang messages read 80f142f6-5cd8-48cc-8490-09accb28be80`)
- Related perf message: ailang-parse msg `e234c455` (XLSX overall perf, includes ask P3 for `parseFoldChildren`)

---

**Document created**: 2026-04-14
**Last updated**: 2026-04-14

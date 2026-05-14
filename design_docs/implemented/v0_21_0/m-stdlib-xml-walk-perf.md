## M-STDLIB-XML-WALK-PERF: `std/xml` — Cut FFI Cost of Tree Walks

**Status**: IMPLEMENTED
**Target**: v0.21.0
**Priority**: P1 (High — actively bottlenecking sunholo/ailang-parse; profile data attached)
**Estimated**: 2 days
**Dependencies**: None. Additive to existing `std/xml` builtins; `XmlNode` ADT unchanged.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure functions; same node → same fold/map output. |
| A2: Replayability | 0 | No trace surface change. |
| A3: Effect Legibility | +1 | All `pure func`; no hidden effects. |
| A4: Explicit Authority | 0 | No capabilities. |
| A5: Bounded Verification | 0 | No new bounds; existing tree depth limits apply. |
| A6: Safe Concurrency | 0 | Stateless, read-only over an immutable ADT. |
| A7: Machines First | +1 | One builtin call replaces a `getChildren → flatMap → closure` chain that accounts for ~30 % of every tree walk. Less code, less interpreter overhead. |
| A8: Minimal Syntax | +1 | No new syntax. Three new functions in `std/xml`. |
| A9: Cost Visibility | +1 | Each new function has an explicit cost model documented (`foldChildren`: O(direct children), zero intermediate allocs). |
| A10: Composability | +1 | Same `XmlNode` ADT as `parseFold`/`findAll`; immediately benefits `std/html` callers too. |
| A11: Structured Failure | 0 | No failure mode beyond runtime errors (handler exceptions propagate via `FnCallerN`). |
| A12: System Boundary | 0 | No FFI surface change; same builtin registration path. |

**Net Score: +6** → **Decision: Move forward.**

### Hard Violation Check

- [x] A1 (Determinism): pure operations
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): collapses the dominant tree-walk pattern into one call

## Problem Statement

`sunholo/ailang-parse` traced one full `std/html.parse` + walk of a 79 KB page (`www.sunholo.com`, ~1,900 nodes) and observed:

> **21,651 function calls** and **43,732 trace events** per page.

The dominant cost lines:

| Function | Calls | Notes |
|---|---:|---|
| `std/xml.getChildren` | 1,918 | Allocates a fresh `[XmlNode]` per node |
| `std/xml.getTag` | 1,869 | Per-node string equality at walk sites |
| `std/list.concat` | 1,735 | Glues `flatMap` results together |
| `flatMap` (cumulative) | 861 | 7.2 B ns wall — biggest bucket |
| Closure invocations | 1,037 | 562 μs/call × 1,037 = 583 ms |

The walk pattern is structural and unavoidable today: every AILANG node visitor looks like

```ailang
flatMap(processNode, getChildren(node))
```

Each `getChildren` call:
1. Crosses the AILANG↔Go FFI boundary.
2. Allocates a fresh `*eval.ListValue` containing the children.
3. Hands it to `flatMap`, which then walks it allocating intermediate lists.
4. Each child triggers another `getTag`/`getAttr`/… FFI call.

For a ~1,900-node tree, that is roughly **2,000 list allocations and ~6,000 FFI crossings per parse**. Image-heavy pages compound it: extracting 7 attributes from each `<img>` is 7 separate `getAttr` FFI calls.

**Current state:** `std/xml` is feature-complete for *correctness*. It was not designed for the FFI-cost dimension.

**Impact:**
- `ailang-parse` measured 2–3× slowdown attributable to this pattern.
- Any future caller doing structured tree walks (RAG ingestion, doc converters, e-commerce scrapers) hits the same wall.
- The fix is small (three stdlib functions, ~150 LOC of Go) and unblocks a measured external user.

## Goals

**Primary Goal:** Add three `std/xml` builtins that eliminate the dominant FFI/allocation cost of tree walks, without changing the `XmlNode` ADT or any existing behaviour.

**Success Metrics:**
- `foldChildren(node, init, f)` exists and visibly halves `std/xml.getChildren` call count on the ailang-parse profile.
- `getAttrMap(node)` exists; image-heavy pages see attribute FFI calls drop from N-per-node to 1-per-node.
- `nodeKind(node)` exists; callers can match on a small variant instead of string-equality on tag names.
- All three available from `std/html` too (free — they take `XmlNode`).
- `BenchmarkXmlWalk_5KB` in `internal/builtins/xml_test.go` shows ≥30 % wall-clock improvement when rewriting a representative walker to use `foldChildren`.
- No regression in any existing `std/xml` or `std/html` test.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `foldChildren` is fold-only vs. fold + early-stop (`foldChildrenStep`) | Determines whether bounded walks are ergonomic (short-circuit on first match) or have to be hacked around in the accumulator. | human | design | med |
| `getAttrMap` returns `Map[string, string]` vs. `List[{name, value}]` | Map gives O(1) lookups for the common multi-attr case but loses source ordering. List preserves ordering but the caller still pays O(n) for each lookup. | human | design | med |
| Introduce a new `NodeKind` ADT vs. reuse existing constructor names exposed via a builtin | New ADT = nicer pattern match in callers, but adds a stdlib type they have to import. Builtin returning a string = no new type, but loses pattern-match exhaustiveness. | human | design | med |
| Whether to also add `foldDescendants` (recursive variant) in v1 | `foldChildren` covers the *direct-child* case which is the reported bottleneck. `foldDescendants` would replace `findAll` + map but adds API surface. | agent | design | low |

### Design Freeze

- [ ] **`foldChildren` ships with both fold and step variants.** The bounded variant (`foldChildrenStep`) returning `FoldStep[a]` is cheap to add given the precedent in `parseFoldStep`. Skipping it would force every "find the first child matching X" caller back to allocating a list.
- [ ] **`getAttrMap` returns `Map[string, string]`.** O(1) lookups are the whole point. Source ordering is preserved separately by the existing attribute list on `Element`; callers that need ordering can keep using `findAllAttrs`/explicit access. Document the trade-off.
- [ ] **`NodeKind` is a new ADT in `std/xml`.** Defined as `type NodeKind = Element | Text | Comment`. Three constructors, no payload. Returned by `nodeKind(node)`. Pattern-matchable in callers, the linter will catch missed cases.
- [ ] **No `foldDescendants` in v1.** Deferred. `foldChildren` is the proven hot path; deeper recursion can compose `foldChildren` with `parseFold` (string-level) or call `findAll` for the rare cases that genuinely need every descendant.

## Solution Design

### Overview

Three new builtins in [internal/builtins/xml_query.go](../../../internal/builtins/xml_query.go) (the file is currently 309 LOC; adding ~150 LOC keeps it well under the 800-line guidance). Three new exports in [std/xml.ail](../../../std/xml.ail). All three operate on the existing `XmlNode` ADT — `std/html` callers get them for free since `std/html.parse` returns the same ADT.

### API Surface

```ailang
-- std/xml additions
export type NodeKind = Element | Text | Comment

-- Fold over direct children without materializing [XmlNode].
-- Replaces: foldl(f, init, getChildren(node)).
-- Cost: O(direct children) calls to f, zero intermediate list allocation.
export pure func foldChildren[a](node: XmlNode, init: a, f: (a, XmlNode) -> a) -> a
  = _xml_foldChildren(node, init, f)

-- Bounded variant. Handler returns Continue(a) | Stop(a).
-- Stops the iteration on Stop without visiting remaining children.
export pure func foldChildrenStep[a](node: XmlNode, init: a,
                                      f: (a, XmlNode) -> FoldStep[a]) -> a
  = _xml_foldChildrenStep(node, init, f)

-- All attributes of an Element as a Map. Non-Element nodes return empty map.
-- Replaces: N separate getAttr calls when extracting multiple attrs from one node.
-- Cost: 1 FFI call; O(attrs) build, O(1) lookup per use.
export pure func getAttrMap(node: XmlNode) -> Map[string, string]
  = _xml_getAttrMap(node)

-- Classify the node without string-equality on the tag.
-- Replaces: getTag(n) == "" tests, manual ADT pattern matching at FFI boundary.
-- Cost: 1 FFI call returning a small variant, pattern-matchable.
export pure func nodeKind(node: XmlNode) -> NodeKind
  = _xml_nodeKind(node)
```

### Architecture

**Components:**

1. **`internal/builtins/xml_query.go` additions** (~150 LOC):
   - `registerXmlFoldChildren()` — takes `(XmlNode, a, (a, XmlNode) -> a)`. Walks `tv.Fields[2]` (`*eval.ListValue`) once, calling `ctx.FnCallerN(handler, [acc, child])` per child. Returns the final accumulator. Non-Element → return `init` unchanged.
   - `registerXmlFoldChildrenStep()` — same as above but inspects the returned `TaggedValue`. If `CtorName == "Stop"`, stop early and return `Fields[0]`. If `Continue`, unwrap to `Fields[0]` and continue. Mirrors `xmlParseFoldStepImpl` exactly.
   - `registerXmlGetAttrMap()` — Element nodes: walk `tv.Fields[1]` (the attr list of `{name, value}` records), build a `*eval.MapValue` keyed by name. Non-Element → return empty `MapValue`. Duplicate attribute names: last write wins (matches `findAttrs` semantics).
   - `registerXmlNodeKind()` — read `tv.CtorName`, return a `*eval.TaggedValue` with `CtorName ∈ {"Element", "Text", "Comment"}` and no fields. Unknown ctor → return `Comment` (defensive — shouldn't happen since the ADT is closed).

2. **`std/xml.ail` additions** (~20 LOC):
   - `export type NodeKind = Element | Text | Comment`
   - Four new `export pure func`s as above. Import `FoldStep` from `std/iter`, `Map` from `std/map`.

3. **`internal/builtins/xml_walkperf_test.go`** (~250 LOC):
   - Unit tests on each builtin in isolation (Element / Text / Comment / non-Element inputs, empty attr list, duplicate attrs, Stop-on-first-child, Stop-after-N).
   - `BenchmarkXmlWalk_classic_vs_foldChildren` — wallclock comparison of `flatMap(_, getChildren(_))` vs `foldChildren(_, [], _)` over a 1,900-node tree (the ailang-parse target).
   - `BenchmarkImgAttrExtract_perAttr_vs_attrMap` — 7-attr extraction over 100 nodes, classic getAttr × 7 vs single getAttrMap.

**FnCallerN wiring (boilerplate, copy from `xml_fold.go`):**

```go
func xmlFoldChildrenImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    if ctx == nil || ctx.FnCallerN == nil {
        return nil, fmt.Errorf("_xml_foldChildren: FnCallerN not set (evaluator not wired)")
    }
    tv, ok := args[0].(*eval.TaggedValue)
    if !ok || tv.CtorName != "Element" {
        return args[1], nil // non-Element → init unchanged
    }
    children, ok := tv.Fields[2].(*eval.ListValue)
    if !ok {
        return args[1], nil
    }
    acc := args[1]
    handler := args[2]
    for _, child := range children.Elements {
        next, err := ctx.FnCallerN(handler, []eval.Value{acc, child})
        if err != nil {
            return nil, err
        }
        acc = next
    }
    return acc, nil
}
```

**Type signatures (Go side):**

```go
func makeXmlFoldChildrenType() types.Type {
    T := types.NewBuilder()
    a := T.Var("a")
    xmlNodeType := T.Con("XmlNode")
    fn := T.Func(a, xmlNodeType).Returns(a).Build()
    return T.Func(xmlNodeType, a, fn).Returns(a).Build()
}

func makeXmlGetAttrMapType() types.Type {
    T := types.NewBuilder()
    xmlNodeType := T.Con("XmlNode")
    return T.Func(xmlNodeType).Returns(T.Map(T.String(), T.String())).Build()
}

func makeXmlNodeKindType() types.Type {
    T := types.NewBuilder()
    return T.Func(T.Con("XmlNode")).Returns(T.Con("NodeKind")).Build()
}
```

### Implementation Plan

**Phase 1: Builtins (~5 hours)**
- [ ] Add `_xml_foldChildren` + `_xml_foldChildrenStep` + `_xml_getAttrMap` + `_xml_nodeKind` to `internal/builtins/xml_query.go`. Wire into `init()`.
- [ ] Reuse `*eval.MapValue` construction from `internal/builtins/map.go`.
- [ ] `make build && make test` clean.

**Phase 2: Module surface (~1 hour)**
- [ ] Add `NodeKind` ADT + 4 `pure func`s to `std/xml.ail`. Import `FoldStep`, `Map`.
- [ ] Verify type signatures match Go side.

**Phase 3: Tests + benchmarks (~4 hours)**
- [ ] `internal/builtins/xml_walkperf_test.go`: unit tests covering all four builtins, edge cases (Text/Comment/empty/duplicate attrs), Stop semantics.
- [ ] Benchmarks: classic-walk vs foldChildren-walk on a synthetic 1,900-node tree; per-attr vs attrMap on 100×7-attr inputs.
- [ ] Run `go test ./internal/builtins -run XmlWalkPerf -bench Walkperf -benchmem` and capture the numbers for the CHANGELOG.

**Phase 4: Example + docs (~2 hours)**
- [ ] `examples/runnable/xml_walk_perf.ail` — same walker written two ways, prints both timings via `clock.now()`.
- [ ] Update [docs/docs/reference/stdlib/std-xml.md](../../../docs/docs/reference/stdlib/std-xml.md) with the four new function entries + a "Cost model" subsection.
- [ ] CHANGELOG.md v0.21.0 entry quoting the benchmark deltas.
- [ ] Reply to inbox msg `cd45490b` (sunholo/ailang-parse) with ship confirmation and a one-liner showing the rewrite pattern.

### Files to Modify/Create

**New files:**
- `internal/builtins/xml_walkperf_test.go` — ~250 LOC (unit + bench).
- `examples/runnable/xml_walk_perf.ail` — ~50 LOC.

**Modified files:**
- `internal/builtins/xml_query.go` — +150 LOC (four new register/impl pairs). Lands at ~460 LOC, still well under 800.
- `std/xml.ail` — +20 LOC, one new type, four new functions.
- `docs/docs/reference/stdlib/std-xml.md` — +50 LOC.
- `CHANGELOG.md` — v0.21.0 entry.

## Examples

### Example 1: Classic walker → `foldChildren`

**Before:**
```ailang
func htmlProcessChildren(node: XmlNode) -> [Block] =
  flatMap(htmlProcessNode, getChildren(node))
```

**After:**
```ailang
func htmlProcessChildren(node: XmlNode) -> [Block] =
  foldChildren(node, [], \acc, child. append(acc, htmlProcessNode(child)))
```

The `append`-per-step variant is fine for short child lists. For long ones, accumulate into a builder type or use a chunked accumulator.

### Example 2: Image attribute extraction

**Before** (7 FFI calls):
```ailang
let src    = getAttr(img, "src")    in
let alt    = getAttr(img, "alt")    in
let width  = getAttr(img, "width")  in
let height = getAttr(img, "height") in
let srcset = getAttr(img, "srcset") in
let title  = getAttr(img, "title")  in
let loading = getAttr(img, "loading") in ...
```

**After** (1 FFI call):
```ailang
let attrs = getAttrMap(img) in
let src    = map.get(attrs, "src")    in
let alt    = map.get(attrs, "alt")    in
let width  = map.get(attrs, "width")  in ...
```

### Example 3: NodeKind pattern match

**Before:**
```ailang
if getTag(n) == ""
then -- assume text, sort of...
  processText(getText(n))
else
  processElement(n)
```

**After:**
```ailang
match nodeKind(n) {
  Element => processElement(n),
  Text    => processText(getText(n)),
  Comment => () -- skip
}
```

### Example 4: Early-stopping search

```ailang
import std/iter (FoldStep, Continue, Stop)
import std/option (Option, None, Some)

-- First child whose tag is "h1"
func firstH1(node: XmlNode) -> Option[XmlNode] =
  foldChildrenStep(node, None, \acc, c.
    if getTag(c) == "h1"
    then Stop(Some(c))
    else Continue(acc)
  )
```

## Success Criteria

- [ ] All four new builtins are registered, type-check, and pass unit tests.
- [ ] `NodeKind` ADT is importable from `std/xml`.
- [ ] `examples/runnable/xml_walk_perf.ail` runs and prints both the classic and `foldChildren` timings, with the latter measurably lower.
- [ ] Benchmark in `xml_walkperf_test.go` shows ≥30 % wall-clock improvement (and lower B/op) for the `foldChildren` rewrite vs `flatMap + getChildren`.
- [ ] `getAttrMap` benchmark shows N×getAttr vs 1×getAttrMap crossover (i.e. N≥2 wins).
- [ ] All existing `std/xml`, `std/html`, and `ailang-parse` examples still pass.
- [ ] CHANGELOG entry + docs page shipped.
- [ ] Inbox msg `cd45490b` acked with a ship reply.

## Testing Strategy

**Unit tests** (`internal/builtins/xml_walkperf_test.go`):
- `foldChildren` on Element → folds in document order.
- `foldChildren` on Text/Comment → returns `init` unchanged (no children).
- `foldChildren` on Element with no children → returns `init`.
- `foldChildrenStep` early-stops on `Stop(a)` and returns that `a`.
- `foldChildrenStep` runs to end if every step returns `Continue(_)`.
- `getAttrMap` on Element with attrs → map contains each (name → value).
- `getAttrMap` on Element with duplicate attr names → last write wins.
- `getAttrMap` on Text/Comment → empty map.
- `nodeKind` returns `Element`/`Text`/`Comment` per ctor.
- Handler panic / error propagates through `FnCallerN`.

**Benchmarks:**
- `BenchmarkXmlWalk_Classic` vs `BenchmarkXmlWalk_FoldChildren` on a synthetic 1,900-node tree mirroring the ailang-parse profile.
- `BenchmarkAttr_PerAttr_7x100` vs `BenchmarkAttr_AttrMap_7x100` — image extraction pattern.
- Capture B/op, allocs/op, ns/op. Quote in CHANGELOG.

**Integration test:**
- Build a representative AILANG walker on a 79 KB HTML fixture, rewrite it to use `foldChildren` + `getAttrMap` + `nodeKind`, assert the rewritten walker emits the same output and runs faster.

## Deferred Decisions

- **`foldDescendants`** — full recursive fold variant. Probably ship if `foldChildren` lands well and a caller asks. Don't speculate.
- **`getAttrMap` source-order preservation** — if a caller needs both O(1) lookup *and* source order, ship `getAttrEntries(node) -> [{name, value}]` later. Today, the existing `Element` ADT already exposes the ordered list.
- **`nodeKind` extension to CData / PI / Doctype** — neither `std/xml` nor `std/html` emits those today. If `parseDocument` (deferred from M-STDLIB-HTML) lands, extend `NodeKind` then.

## Non-Goals

- **Tail-call optimization, `@inline` hints, or trace-zero-cost build flag.** These are runtime/compiler concerns from the same ailang-parse profile but each is its own design doc and sprint. This doc is strictly stdlib.
- **DOM mutation.** `XmlNode` stays immutable.
- **CSS selectors.** Out of scope; tag-based queries are sufficient.
- **Changing the `XmlNode` ADT.** Every new function operates on the existing shape.

## Timeline

**Day 1 (~6 hours):**
- Phase 1 (builtins)
- Phase 2 (module surface)

**Day 2 (~6 hours):**
- Phase 3 (tests + benchmarks)
- Phase 4 (example + docs + CHANGELOG)
- Reply to inbox msg

**Total: ~12 hours / 2 days.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `FnCallerN` overhead is itself the dominant cost — `foldChildren` ends up no faster than `flatMap`. | Med | Benchmark *before* committing the docs message. If we don't see ≥30 % wallclock improvement, hold the doc, profile, and iterate. The walker still wins on allocations regardless. |
| Map insertion order with duplicate attrs surprises callers expecting source order. | Low | Document "last write wins" explicitly. Existing `Element` ADT preserves order for callers who need it. |
| `NodeKind` clashes with caller-defined types named `Element` / `Text` / `Comment`. | Low | Stdlib namespace; callers import qualified or rename. AILANG's import-with-rename already handles this. |
| Benchmark synthetic tree doesn't represent ailang-parse's actual workload, leading to overstated improvement. | Med | Use the ailang-parse trace as a seed for the synthetic tree shape (depth, fan-out, attr count). If feasible, drop a copy of the ailang-parse fixture into `internal/builtins/testdata/`. |
| `getAttrMap` allocates a map even when caller only reads one attr — hurts the "single attr" case. | Low | Document the crossover. `getAttr` stays — it's still the right call for N=1. |

## Related Documents

**Prior art — required reading before implementation:**
- [internal/builtins/xml.go:680-724](../../../internal/builtins/xml.go#L680) — `_xml_getChildren` reference impl.
- [internal/builtins/xml_fold.go:32-200](../../../internal/builtins/xml_fold.go#L32) — `_xml_parseFold` + `_xml_parseFoldStep` reference for `FnCallerN` wiring and `FoldStep` unwrap.
- [internal/builtins/map.go:97-100](../../../internal/builtins/map.go#L97) — `*eval.MapValue` construction.
- [design_docs/implemented/v0_19_1/m-stdlib-html.md](../../implemented/v0_19_1/m-stdlib-html.md) — confirms `std/html` shares `XmlNode`, so these builtins immediately benefit HTML callers.

**Sibling docs:**
- [m-stdlib-html-streaming.md](./m-stdlib-html-streaming.md) — addresses memory pressure at multi-MB input scale. Orthogonal axis; both can ship independently.

## References

- [Design Axioms](/docs/references/axioms)
- Inbox message: `cd45490b-...` (sunholo/ailang-parse, 2026-05-14) — profile data + 6-feature proposal. This doc covers features (1), (2), (5) of that message; features (3), (4), (6) are separate language/runtime tickets.
- Profile data source: `ailang run --emit-trace jsonl` on `www.sunholo.com` (79 KB HTML / ~1,900 nodes).

## Future Work

- **`foldDescendants`** — full-tree fold; ship if a caller asks.
- **`getAttrEntries(node) -> [{name, value}]`** — ordered counterpart to `getAttrMap`.
- **`@inline` hint, TCO, zero-cost trace** — separate runtime/compiler design docs; the ailang-parse message lists them but they don't belong here.

---

**Document created**: 2026-05-14
**Last updated**: 2026-05-14

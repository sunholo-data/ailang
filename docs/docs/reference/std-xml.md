---
title: std/xml — Tree-walk performance builtins
description: foldChildren, foldChildrenStep, getAttrMap, and nodeKind — collapse the FFI/allocation cost of XmlNode walks
sidebar_label: std/xml (perf)
---

# std/xml — Tree-walk performance builtins

`std/xml` is feature-complete for correctness: [`parse`](./stdlib), `findAll`, `getAttr`, `getChildren`, and the streaming `parseFold` family cover every shape of input. This page documents the four builtins added in **v0.21.0** that exist purely to cut the FFI/allocation cost of *walking* a parsed `XmlNode` tree.

The motivation came from a [`sunholo/ailang-parse`](https://github.com/sunholo/data/ailang-parse) profile: one walk of a 79 KB HTML page produced **21,651 function calls** and **43,732 trace events**, dominated by `getChildren` allocating a fresh `[XmlNode]` per node and `getAttr` paying a separate FFI crossing per attribute. The fix is API-level, not runtime-level — three new builtins that fold what the walker was doing into one call.

All four functions operate on the existing `XmlNode` ADT. `std/html` callers benefit for free because `std/html.parse` returns the same shape.

## When to use

Pick the primitive by the **shape of your result**, not by what your old code happens to look like:

| You wrote… | Use instead | Reason |
|---|---|---|
| `foldl(f, init, getChildren(node))` *(scalar accumulator: count/sum/string/map)* | `foldChildren(node, init, f)` | Skips the allocated `[XmlNode]` and one FFI crossing per visit. |
| `flatMap(processChild, getChildren(node))` *(walker output is a flat list)* | **`flatMapChildren(node, processChild)`** | `foldChildren` is the *wrong* primitive here — its AILANG-side accumulator (`prepend+reverse`) allocates O(N²) bytes vs `flatMapChildren`'s O(N) Go-native append. |
| `map(processChild, getChildren(node))` *(one output element per child)* | **`mapChildren(node, processChild)`** | Same reasoning. Pre-sized output slice, exact length known. |
| First child matching a predicate via filter + head | `foldChildrenStep(node, None, \_, c. if pred(c) then Stop(Some(c)) else Continue(None))` | Stops the walk on the first match instead of visiting every child. |
| Pulling N attributes from one element via N × `getAttr` | `let m = getAttrMap(node) in (map.get(m, "a"), map.get(m, "b"), …)` | Collapses N FFI crossings into one. Crossover at N ≥ 2. |
| `if getTag(n) == "" then … // text? // else …` | `match nodeKind(n) { KindElement => …, KindText => …, KindComment => … }` | Pattern-matchable, exhaustiveness-checkable, one FFI call. |

If you only need a single attribute (N=1), keep using [`getAttr`](./stdlib) — `getAttrMap` allocates a map and is a net loss there.

> **Footgun warning** — `foldChildren` is for scalar accumulators only. If your handler returns a list and you accumulate via `prepend + reverse` (or by repeatedly appending immutable lists), you will allocate *more* than the classic `flatMap(_, getChildren(_))` did, not less. `flatMapChildren` exists exactly for the list-output case. This was the v0.21.0 M4 hotfix after `sunholo/ailang-parse` reported a 229 MB → 395 MB regression on a 1.7 MB Mollie page.

## API

### `foldChildren[a](node: XmlNode, init: a, f: (a, XmlNode) -> a) -> a`

Fold over the direct children of an Element without materialising `[XmlNode]`. Non-Element nodes (`Text`, `Comment`) return `init` unchanged.

```ailang
import std/xml (foldChildren, getTag)

-- Count how many direct <li> children a <ul> has.
func countListItems(ul: XmlNode) -> int =
  foldChildren(ul, 0, \acc, child.
    if getTag(child) == "li" then acc + 1 else acc
  )
```

**Cost**: O(direct children) calls to `f`. Zero intermediate list allocation. Compared to `foldl(f, init, getChildren(n))`, saves one `[XmlNode]` allocation and the per-element FFI crossings that materialising the list would cost.

### `foldChildrenStep[a](node: XmlNode, init: a, f: (a, XmlNode) -> FoldStep[a]) -> a`

Bounded variant of `foldChildren`. The handler returns `FoldStep[a] = Continue(a) | Stop(a)` (from `std/iter`). `Stop(acc')` halts the walk immediately and returns `acc'`.

```ailang
import std/iter (FoldStep, Continue, Stop)
import std/option (Option, None, Some)
import std/xml (foldChildrenStep, getTag)

-- First direct child whose tag is "h1".
func firstH1(node: XmlNode) -> Option[XmlNode] =
  foldChildrenStep(node, None, \acc, c.
    if getTag(c) == "h1"
      then Stop(Some(c))
      else Continue(acc)
  )
```

**Cost**: O(prefix until `Stop`). Mirrors the existing `parseFoldStep` short-circuit pattern.

### `getAttrMap(node: XmlNode) -> Map[string, string]`

Build a `Map[string, string]` from all attributes of an Element. Non-Element nodes return an empty map. Duplicate attribute names: last write wins (consistent with `findAttrs` semantics).

```ailang
import std/xml (getAttrMap)
import std/map as map

-- One FFI call, then O(1) per attribute lookup.
func extractImgFields(img: XmlNode) -> {src: Option[string], alt: Option[string], srcset: Option[string]} =
  let attrs = getAttrMap(img) in
  { src    = map.get(attrs, "src")
  , alt    = map.get(attrs, "alt")
  , srcset = map.get(attrs, "srcset")
  }
```

**Cost**: 1 FFI call + O(attrs) Go-side map build, then O(1) per lookup. Crossover with `getAttr` at N ≥ 2 attribute probes per node. **Source order is not preserved** — if you need ordering, keep using the existing `Element(_, attrs, _)` field directly, which is the ordered list.

### `nodeKind(node: XmlNode) -> NodeKind`

Classify a node as `KindElement | KindText | KindComment` without string-equality on the tag. Pattern-matchable and exhaustiveness-checkable.

```ailang
import std/xml (NodeKind, KindElement, KindText, KindComment, nodeKind, getText)

func render(n: XmlNode) -> string =
  match nodeKind(n) {
    KindElement => renderElement(n),
    KindText    => getText(n),
    KindComment => ""
  }
```

**Cost**: 1 FFI call returning a tagged value with no fields.

### `type NodeKind = KindElement | KindText | KindComment`

A small ADT with three nullary constructors. **The constructors are prefixed `Kind`** to avoid colliding with `XmlNode`'s own `Element` / `Text` / `Comment` constructors — both types are exported from `std/xml`, and the prefix lets you import both without rename.

### `flatMapChildren[a](node: XmlNode, f: (XmlNode) -> [a]) -> [a]`

*Added v0.21.0 (M4 hotfix).* Go-iterative `flatMap` over the direct children of an Element. `f` returns a list per child; results are concatenated into one Go-native slice with O(1) amortised append. **Use this — not `foldChildren` — when the walker output is a flat list.** Non-Element nodes return `[]`.

```ailang
import std/xml (flatMapChildren)

-- Process every direct child into 0+ Block outputs; flatten the result.
func htmlProcessChildren(node: XmlNode) -> [Block] =
  flatMapChildren(node, htmlProcessNode)
```

**Cost**: 1 FFI call returning a single allocated slice. No `[XmlNode]` intermediate, no AILANG-side accumulator. Drops the prepend+reverse penalty `foldChildren` would impose for list output.

### `mapChildren[a](node: XmlNode, f: (XmlNode) -> a) -> [a]`

*Added v0.21.0 (M4 hotfix).* Like `flatMapChildren` but each callback returns exactly one output element. Output slice is pre-sized to the children count. Non-Element nodes return `[]`.

```ailang
import std/xml (mapChildren, getTag)

-- Snapshot of the direct child tag names in document order.
func childTags(node: XmlNode) -> [string] =
  mapChildren(node, \c. getTag(c))
```

**Cost**: 1 FFI call. Exactly `len(children)` output elements; no resize.

## Cost model

The four builtins target three concrete bottlenecks identified by the `ailang-parse` profile. Use this table to predict whether a rewrite will pay off:

| Pattern | FFI crossings / node | Allocations / node | After rewrite |
|---|---:|---:|---|
| `foldl(f, init, getChildren(n))` | 1 (`getChildren`) + 1 per child | 1 × `[XmlNode]` | `foldChildren`: 1 FFI call total, zero list alloc |
| 7× `getAttr(img, k)` over 100 images | 700 | 0 | `getAttrMap`: 100 FFI calls + 100 small map allocs, O(1) lookups |
| `getTag(n) == ""` text-detect | 1 (`getTag`) + 1 string compare | 1 × string | `nodeKind`: 1 FFI call returning a tagged variant |

**The wins compound.** A walker that does *all three* on every node — fold direct children, classify each, then for each element pull multiple attributes — sees the AILANG-level wallclock drop materially. Measured numbers (see "Measured impact" below) are 20-29% wallclock reduction on the benchmark example, with the upper end approached as attribute fan-out grows.

**The wins do not appear in Go-level microbenchmarks.** `FnCallerN` in test mode is a direct Go call with no closure-invocation overhead — exactly the overhead the design targets. Quote AILANG-level timings (from runnable examples) when reporting impact; Go benches only verify the floor cost.

## Measured impact

From [`examples/runnable/xml_walk_perf.ail`](https://github.com/sunholo/data/ailang/blob/dev/examples/runnable/xml_walk_perf.ail) — a synthetic tree of 2,000 `<row>` children with 10 attributes each, walked via `clock.now()` wallclock comparison:

| Walker | Time (median of 5 runs) |
|---|---:|
| Classic: `foldl` + `getChildren` + 10× `getAttr` / child | ~130 ms |
| New:    `foldChildren` + `getAttrMap` + 10× pure map lookups | ~103 ms |

≈ 20 % wallclock reduction; roughly 18,000 fewer AILANG↔Go crossings per walk. The win scales with attributes-probed-per-node: for the workload that drove this design (`<img>` elements with 7 attributes each, ≥100 images per page), expected reduction is closer to 30 %.

Run the example yourself:

```bash
ailang run --caps Clock examples/runnable/xml_walk_perf.ail
```

## Migration: rewriting a walker

A typical walker `node -> [Block]` looks like this **before**:

```ailang
func processChildren(node: XmlNode) -> [Block] =
  flatMap(processNode, getChildren(node))

func processNode(n: XmlNode) -> [Block] =
  if getTag(n) == ""
    then [TextBlock(getText(n))]
    else
      let src = getAttr(n, "src") in
      let alt = getAttr(n, "alt") in
      let title = getAttr(n, "title") in
      [ImgBlock(src, alt, title)]
```

**After** (one FFI call per concern):

```ailang
import std/xml (foldChildren, getAttrMap, nodeKind, KindElement, KindText, KindComment, getText)
import std/map as map

func processChildren(node: XmlNode) -> [Block] =
  foldChildren(node, [], \acc, c. append(acc, processNode(c)))

func processNode(n: XmlNode) -> [Block] =
  match nodeKind(n) {
    KindText    => [TextBlock(getText(n))],
    KindComment => [],
    KindElement =>
      let attrs = getAttrMap(n) in
      [ImgBlock(map.get(attrs, "src"), map.get(attrs, "alt"), map.get(attrs, "title"))]
  }
```

For long child lists, the `append`-per-step pattern above is O(n²); accumulate into a chunked builder or reverse-cons-then-reverse if that becomes the bottleneck.

## Related

- [`std/iter`](./stdlib) — `FoldStep`, `Continue`, `Stop` for the `foldChildrenStep` handler shape.
- [`std/xml.parseFold` / `parseFoldStep`](./stdlib) — the same fold-over-bounded-input pattern, but at the parse level for streaming large documents.
- [`std/html`](./stdlib) — returns the same `XmlNode` ADT, so all four builtins apply unchanged to HTML walks.
- [Stdlib Index](./stdlib) — the full `std/xml` and `std/html` surface.

## See also

The "Cost visibility" axiom ([A9](/docs/references/axioms)) requires every new stdlib function to ship with an explicit cost model. The table in [Cost model](#cost-model) is the contract.

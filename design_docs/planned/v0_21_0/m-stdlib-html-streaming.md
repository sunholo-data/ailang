## M-STDLIB-HTML-STREAMING: `std/html` — Streaming Parse for Large Documents

**Status**: Planned
**Target**: v0.21.0 (tentative — placeholder bucket; v0.20.0 is locked on M-TYPECHECK-NO-AUTO-UNWRAP-RESULT)
**Priority**: P2 (Medium — quality-of-life for memory-bound HTML ingestion; no blocker reported yet)
**Estimated**: 2 days
**Dependencies**: M-STDLIB-HTML (shipped v0.19.1). Mirrors M-STREAMING-ZIP-XML (shipped v0.11.0).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure functions over input string; same bytes → same outputs. |
| A2: Replayability | 0 | No trace surface change. |
| A3: Effect Legibility | +1 | `pure func` signatures throughout; no hidden effects vs `std/xml.parseFold`. |
| A4: Explicit Authority | 0 | No capabilities involved. |
| A5: Bounded Verification | +1 | Adds a `parseWithLimit` node-count cap and tightens memory bounds for `parseElements`/`parseFold` to O(largest matched element) instead of O(whole document). |
| A6: Safe Concurrency | 0 | Stateless; safe to call from any goroutine. |
| A7: Machines First | +1 | One-call replacements for "parse, walk, throw the tree away" pipelines. Less code for an LLM to assemble correctly. |
| A8: Minimal Syntax | +1 | No new syntax. Four new `pure func`s on the existing module. |
| A9: Cost Visibility | +1 | Cost model is the same as the `std/xml` streaming siblings — documented O(n) scan, O(1) memory per match. |
| A10: Composability | +1 | Returns the same `XmlNode` ADT — all `std/xml` query helpers chain after the streaming step. Same shape `std/xml.parseFold`/`parseFoldStep` already use. |
| A11: Structured Failure | +1 | All four entrypoints return `Result[..., string]`. |
| A12: System Boundary | 0 | No FFI surface change; reuses already-direct `golang.org/x/net/html` dep. |

**Net Score: +7** → **Decision: Move forward.**

### Hard Violation Check

- [x] A1 (Determinism): pure parse, no nondeterminism
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): four small, type-symmetric additions instead of a callable budget knob

## Problem Statement

`std/html.parse` (v0.19.1) is a single-shot parse: it materialises the entire `XmlNode` tree before the caller sees anything. The 50 MB input cap and 256 depth cap protect against pathological inputs, but the **expected** memory footprint for a normal-sized HTML document is roughly:

> `~5–10 × len(input)` resident bytes, plus one `XmlNode` ADT box per element / text / comment.

For an 8 MB HTML page (the upper end of CMS exports and scraped news pages) the converted tree comfortably exceeds 50 MB resident. For a 30 MB Wayback Machine snapshot it pushes 200 MB. The bigger memory ceiling for HTML is therefore not parsing per se — `golang.org/x/net/html` is fine — it is the **AILANG-side tree allocation**. Streaming the matched fragments out, the way `std/xml.parseElements` / `parseFold` already do for XML, would keep the working set bounded by the *largest matched element*, not the whole document.

**Current State:**
- `std/html.parse` returns `Result[XmlNode, string]` over the whole document. No early exit. No way to extract just `<article>` blocks from a 40 MB page without holding the rest of the tree in memory.
- `std/xml` already ships the streaming primitives we want to mirror:
  - `_xml_parseElements(xml, tag, maxResults)` — extract up to N matching elements ([internal/builtins/xml.go:194](../../../internal/builtins/xml.go#L194))
  - `_xml_parseFold(xml, tag, init, f)` — fold over matches with an accumulator ([internal/builtins/xml_fold.go:32](../../../internal/builtins/xml_fold.go#L32))
  - `_xml_parseFoldStep(xml, tag, init, f)` — bounded fold with `Continue(a) | Stop(a)`
  - `_xml_parseWithLimit(xml, maxNodes)` — full parse with a node-count safety valve
- Caller workaround today: split the HTML in AILANG using string heuristics, call `parse` on each chunk. Fragile and breaks on real-world unclosed tags.

**Impact:**
- Any project doing memory-bounded HTML ingestion (RAG over web archives, Word `.docx` (which is an HTML-ish XML), e-commerce scraping, large `.html` mail bodies) will hit this wall the moment input grows past a few MB.
- The downstream caller that prompted M-STDLIB-HTML (`sunholo/ailang-parse`) is already a candidate. Its next-up corpus is multi-MB HTML emails and full-page captures.

## Goals

**Primary Goal:** Extend `std/html` with the same streaming surface `std/xml` already offers, so HTML ingestion has the same memory cost model as XML ingestion.

**Success Metrics:**
- Parsing a 30 MB HTML document and folding the first 5 000 `<article>` elements into a list of titles runs to completion with resident memory bounded by `O(largest matched element)`, not `O(input size)`.
- `std/html.parseElements`, `parseFold`, `parseFoldStep`, `parseWithLimit` exist with signatures *identical in shape* to their `std/xml` siblings (only the module name differs).
- `ailang-parse` (or any caller) can replace `match parse(big) { Ok(t) => findAll(t, "article") | ... }` with `parseFold(big, "article", [], ...)` and observe a measurable peak-RSS drop on the same input.
- No regression in `std/html.parse` runtime or memory on small inputs (<1 MB).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Use `html.Tokenizer` (raw HTML5 token stream) vs. full `html.Parse` chunked | Tokenizer streams, Parse does not. But Tokenizer skips HTML5's auto-closing semantics, so `<p>a<p>b` is two siblings via Parse but one nested `<p>` via raw tokens. Affects the API contract. | human | design | high |
| Whether "matched element" means *syntactic* (raw tag-stack from tokenizer) or *post-fixup* (what `parse()` would have produced) | If syntactic, callers get cheap streaming but slight divergence from `parse()` for inline tags. If post-fixup, we have to reimplement enough of the HTML5 fragment-parsing algorithm to be wrong differently. | human | design | high |
| Reuse `_xml_parseFold`'s `FnCallerN` evaluator wiring as-is | Determines whether HTML fold can reuse XML fold's plumbing (no new evaluator changes) or needs its own callback layer. | agent | compile | low |
| `parseWithLimit` semantics: cap on node count vs. cap on bytes-after-conversion | Node count is what XML does; matching it keeps the surface symmetric. Bytes would be more precise but inconsistent. | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Tokenizer-based streaming for `parseElements`/`parseFold`/`parseFoldStep`.** Use `html.NewTokenizer(reader)`. Track a raw-tag stack; when an open tag matches the requested name, switch to "buffering" mode and collect children until the depth returns to the matched element's open level. Outside matches, advance without allocating subtrees.
- [ ] **Syntactic matching, with a documented caveat.** For block-level tags (`<article>`, `<section>`, `<div>`, `<table>`, `<tr>`, `<li>` *inside well-formed source*), syntactic depth matches `parse()`'s tree. For inline auto-closing tags (`<p>`, `<dt>`, `<dd>`, `<option>`, `<tr>` *inside malformed source*), the streaming variants may match a larger or smaller span than `parse()` would. Document this on the function and link to the WHATWG insertion-mode tables.
- [ ] **`parseWithLimit` does a full parse and bails on node count.** Same semantics as `_xml_parseWithLimit`. Cheap to implement; useful as a "I know my doc is small but want a hard cap" valve.
- [ ] **Reuse the `htmlNodeToXmlNode` converter.** When buffering a matched element, build a `*html.Node` subtree under a synthetic parent, then run the existing converter. Single source of truth for ADT shape.
- [ ] **Out of scope for v1:** `parseDocument` (full doc with doctype), streaming text-only extraction, CSS selectors. Defer until a caller surfaces them.

## Solution Design

### Overview

Add four new builtins in [internal/builtins/html.go](../../../internal/builtins/html.go), each mirroring one of the `std/xml` streaming siblings. The AILANG-facing surface adds four `pure func`s to [std/html.ail](../../../std/html.ail). The recursive `htmlNodeToXmlNode` converter is reused unchanged.

### Architecture

**Components:**

1. **`std/html.ail` additions** (~25 LOC):
   ```ailang
   import std/iter (FoldStep)

   export pure func parseElements(html: string, tag: string, maxResults: int)
     -> Result[[XmlNode], string] = _html_parseElements(html, tag, maxResults)

   export pure func parseFold[a](html: string, tag: string, init: a,
                                  f: (a, XmlNode) -> a)
     -> Result[a, string] = _html_parseFold(html, tag, init, f)

   export pure func parseFoldStep[a](html: string, tag: string, init: a,
                                      f: (a, XmlNode) -> FoldStep[a])
     -> Result[a, string] = _html_parseFoldStep(html, tag, init, f)

   export pure func parseWithLimit(html: string, maxNodes: int)
     -> Result[XmlNode, string] = _html_parseWithLimit(html, maxNodes)
   ```

2. **`internal/builtins/html.go` additions** (~250 LOC):
   - `registerHtmlParseElements()` — wraps `html.NewTokenizer` + a tag-stack walker, builds subtrees only for matches, stops at `maxResults`.
   - `registerHtmlParseFold()` — same walker, calls `ctx.FnCallerN(handler, [acc, node])` for each match. Mirrors `xmlParseFoldImpl`.
   - `registerHtmlParseFoldStep()` — fold with early termination on `Stop(a)`. Mirrors `xmlParseFoldStepImpl`.
   - `registerHtmlParseWithLimit()` — full `html.Parse`, run conversion, abort with `Err` if node count exceeds limit.
   - Shared helper `scanHtmlForElementsFold` for the three streaming variants (tokenizer + tag-stack + on-match callback).

3. **`internal/builtins/html_streaming_test.go`** (~250 LOC):
   - Memory bound test: `parseFold` on a 5 MB synthetic HTML doc with 50 000 `<article>` elements, asserting peak allocations are bounded relative to a sentinel.
   - Correctness parity with `parse() + findAll(_, tag)` on a curated set of well-formed inputs.
   - Documented-divergence tests for `<p>` / `<li>` auto-closing edge cases — locks the streaming behaviour in place so future "fixes" don't silently change it.
   - `parseFoldStep` early-stop test (Stop after first match in a 10k-element doc should not parse the rest).
   - `parseWithLimit` rejects when tree exceeds cap.

**Tokenizer walker (the only nontrivial logic):**

```go
// Pseudocode
tk := html.NewTokenizer(strings.NewReader(input))
stack := []string{} // raw open-tag names, lowercased
var buffer *html.Node // root of the currently-buffered match
var bufferDepth int  // depth at which we entered buffering

for {
  tt := tk.Next()
  switch tt {
  case html.ErrorToken:
    return // EOF or error — stop scanning
  case html.StartTagToken, html.SelfClosingTagToken:
    name, hasAttr := tk.TagName()
    lower := string(bytes.ToLower(name))
    stack = append(stack, lower)
    if buffer == nil && lower == tagName {
      // Enter buffering: synthesize a *html.Node subtree from tokens
      buffer = newElementNode(lower, collectAttrs(tk, hasAttr))
      bufferDepth = len(stack)
    } else if buffer != nil {
      appendChildElement(buffer, /* current path */, lower, ...)
    }
    if tt == html.SelfClosingTagToken {
      // Pop immediately — net/html convention
      stack = stack[:len(stack)-1]
      if buffer != nil && len(stack) < bufferDepth {
        emit(buffer); buffer = nil
        if results.length >= maxResults { return }
      }
    }
  case html.EndTagToken:
    if len(stack) == 0 { continue } // stray close — drop
    stack = stack[:len(stack)-1]
    if buffer != nil && len(stack) < bufferDepth {
      // Match closed — convert and emit
      node, err := htmlNodeToXmlNode(buffer, 0)
      if err == nil && node != nil { emit(node) }
      buffer = nil
      if results.length >= maxResults { return }
    }
  case html.TextToken, html.CommentToken:
    if buffer != nil { appendChildText/Comment(buffer, tk.Text()) }
  }
}
```

The buffer is a `*html.Node` subtree built incrementally from the token stream, so the existing `htmlNodeToXmlNode` converter (and its 256-depth guard) handles emission without duplicating logic.

**Guardrails:**
- 50 MB input cap (same as `_html_parse`, returned as `Err` before tokenizing).
- 256 depth cap inside the converter (already enforced).
- `parseWithLimit` adds an explicit `maxNodes` cap; the other three are naturally bounded by `maxResults` / `Stop(a)` / accumulator.

**HTML5-fidelity caveat (must be documented):**

The streaming walker uses **raw** start/end tags as the tokenizer emits them. It does **not** apply WHATWG insertion-mode fixups (auto-closing `<p>` before another `<p>`, foster-parenting around `<table>`, etc.). For block-level tags in well-formed HTML this matches `parse()` exactly. For inline auto-closing tags in malformed HTML it may not. Callers who need exact `parse()` parity should call `parse()` and `findAll()`.

The user-facing doc comment on each streaming function will say this in one sentence, with a link to a docs page documenting the precise divergence rules.

### Implementation Plan

**Phase 1: Builtins (~6 hours)**
- [ ] Tokenizer-driven walker in `internal/builtins/html.go`. Factor `scanHtmlForElementsFold(tk, tagName, acc, onMatch)` so all three streaming variants share it.
- [ ] `registerHtmlParseElements`, `registerHtmlParseFold`, `registerHtmlParseFoldStep`, `registerHtmlParseWithLimit`.
- [ ] Reuse `htmlNodeToXmlNode` for emission. Reuse `xmlMakeOk` / `xmlMakeErr` / `makeXml*` constructors.
- [ ] Wire `init()` to call the four new register functions.

**Phase 2: Module surface (~1 hour)**
- [ ] Add four `pure func` declarations to `std/html.ail`. Import `FoldStep` from `std/iter` for `parseFoldStep`.
- [ ] Verify `std/embed.go`'s `*.ail` glob picks up the changes.

**Phase 3: Tests + example (~5 hours)**
- [ ] `internal/builtins/html_streaming_test.go` — correctness, memory bounds, early-stop, divergence-from-parse() golden cases.
- [ ] `internal/builtins/html_streaming_bench_test.go` — `BenchmarkHtmlParseFold_5MB` next to `BenchmarkXmlParseFold_5MB` so the cost asymmetry is visible in CI.
- [ ] `examples/runnable/html_stream_articles.ail` — fold the first 100 `<article>` titles out of a fixture.

**Phase 4: Docs + announce (~2 hours)**
- [ ] Extend `docs/docs/reference/stdlib/std-html.md` with the four new functions.
- [ ] Cost-model section: side-by-side memory table for `parse` vs `parseFold` vs `parseElements`.
- [ ] CHANGELOG entry under v0.21.0 (or whatever lands first).
- [ ] If a caller filed this (registry-side request), reply with shipping confirmation.

### Files to Modify/Create

**New files:**
- `internal/builtins/html_streaming_test.go` — ~250 LOC.
- `internal/builtins/html_streaming_bench_test.go` — ~80 LOC.
- `examples/runnable/html_stream_articles.ail` — ~40 LOC.

**Modified files:**
- `internal/builtins/html.go` — +~250 LOC, four new register/impl pairs and the shared walker. Stays well under the 800-line guidance.
- `std/html.ail` — +25 LOC, four exports.
- `docs/docs/reference/stdlib/std-html.md` — +60 LOC.
- `CHANGELOG.md` — v0.21.0 entry.

## Examples

### Example 1: Memory-bounded extraction

```ailang
import std/html (parseFold)
import std/xml (findFirst, getText)
import std/option (getOrElse)

-- Extract just the <h1> text from each <article> in a 30 MB page
let titles = parseFold(bigHtml, "article", [], \acc, art.
  match findFirst(art, "h1") {
    Some(h) => append(acc, getText(h)),
    None => acc
  }
) in
match titles {
  Ok(ts) => println(show(length(ts))),
  Err(e) => println("html stream failed: " ++ e)
}
```

Peak memory: one `<article>` at a time, never the whole document.

### Example 2: Bounded prefix

```ailang
import std/html (parseFoldStep)
import std/iter (FoldStep, Continue, Stop)

-- Stop after collecting 50 product titles, even if the page has 5000
let firstFifty = parseFoldStep(productPage, "div", [], \acc, div.
  if length(acc) >= 50
  then Stop(acc)
  else Continue(append(acc, getText(div)))
) in ...
```

`parseFoldStep` short-circuits — the rest of the input is not tokenized.

### Example 3: Safety valve

```ailang
import std/html (parseWithLimit)

-- "I expect this to be a small fragment; reject anything weird."
match parseWithLimit(snippet, 10000) {
  Ok(tree) => render(tree),
  Err(_) => renderFallback()
}
```

## Success Criteria

- [ ] `std/html.parseElements`, `parseFold`, `parseFoldStep`, `parseWithLimit` all type-check and execute on the canonical examples.
- [ ] `BenchmarkHtmlParseFold_5MB` shows peak heap allocations within a factor of `BenchmarkXmlParseFold_5MB` on a comparable input.
- [ ] Documented-divergence tests pin the auto-closing behaviour so a future refactor cannot silently break it.
- [ ] All existing `std/html` and `std/xml` tests still pass.
- [ ] `examples/runnable/html_stream_articles.ail` runs end-to-end and prints the expected title list.
- [ ] CHANGELOG entry + docs page shipped.

## Testing Strategy

**Unit tests** (`internal/builtins/html_streaming_test.go`):
- Well-formed HTML: `parseElements(doc, "article", 100)` matches `findAll(parse(doc), "article")` element-for-element.
- Lenient case (documented divergence): `<p>a<p>b` under `parseElements(_, "p", _)` returns two siblings just like `parse()` does — this *does* work for `<p>` because the tokenizer emits two start tags without any close in between, and our walker treats each unclosed start as a depth bump until the implicit close.
- Lenient case (real divergence): `<table><tr><td>` without closing `</td>` — document and lock the exact behaviour.
- `parseFold` accumulator threading: `parseFold(doc, "tr", 0, \acc, _. acc + 1)` returns the row count.
- `parseFoldStep` with `Stop`: scan time linear in matched-prefix length, not document length.
- `parseWithLimit`: 10 001-node tree under limit 10 000 returns `Err`.
- 50 MB+ input returns `Err` before tokenizing.
- Empty string returns `Ok([])` for `parseElements`, `Ok(init)` for fold variants.

**Integration tests:**
- Run all four against a multi-MB fixture (e.g., a Wikipedia article dump). Assert tree depth ≤ 256, asserted titles non-empty.

**Manual testing:**
- Wire into `ailang-parse` on a branch, run its docparse regression suite on the biggest HTML fixture in the corpus, confirm parity and lower peak RSS.

## Deferred Decisions

- **`parseDocument` / doctype round-trip** — still deferred from v0.19.1; not unblocked by this milestone.
- **CSS selector queries** — out of scope; tag-based streaming is sufficient for the known use cases.
- **HTML5-spec-perfect streaming** (full insertion-mode state machine) — would require either upstream changes in `golang.org/x/net/html` to expose a true streaming Parser, or reimplementing the WHATWG state machine. Defer indefinitely; the syntactic-walker contract is good enough.
- **Effectful chunked input** (e.g., `parseFromReader` over a `Stream`) — `std/xml` doesn't have one either. Defer until both modules want it together (would mirror M-STREAMING-ZIP-XML's effectful variant).

## Non-Goals

- **Sanitization / XSS scrubbing** — same as M-STDLIB-HTML; out of scope.
- **DOM mutation** — `XmlNode` stays immutable.
- **Re-rendering HTML** — `std/xml.serialize` is good enough; HTML5-conformant serialization is a separate ask.
- **Selector languages (jQuery / CSS)** — `findAll` after streaming covers the same surface.

## Timeline

**Day 1 (~7 hours):**
- Phase 1 (builtins): tokenizer walker + four register/impl pairs
- Phase 2 (module surface): `std/html.ail` exports

**Day 2 (~7 hours):**
- Phase 3 (tests + benchmarks + example)
- Phase 4 (docs + CHANGELOG)
- `ailang-parse` integration sanity check

**Total: ~14 hours / 2 days.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Token-stream walker diverges from `parse()` on inline auto-closing tags in malformed HTML, surprises callers. | Med | Document the rule explicitly, ship golden tests, advise `parse() + findAll()` for spec-perfect parity. |
| `html.Tokenizer`'s `TagName()`/`TagAttr()` byte slices are reused on `Next()`. Buffering them without copy would mutate prior results. | High | Copy `[]byte` to `string` at the boundary. Already standard practice in net/html consumers; tests will catch a missed copy fast. |
| `parseFold` handler can panic or return non-Result values, mis-aligned with `FnCallerN` contract. | Low | Mirror `_xml_parseFold`'s error propagation exactly; the evaluator wiring is already proven. |
| Memory savings underperform on small inputs because tokenizer overhead is larger than tree build for tiny docs. | Low | Keep `parse()` as the recommended path for `<1 MB` inputs; document the crossover in the cost-model section. |
| `parseFoldStep` early stop leaks the underlying tokenizer (Go GC handles it, but lifetime confusion in tests). | Low | Walker function returns on `Stop`; tokenizer is local — `defer` not needed. |

## Related Documents

**Prior art — required reading before implementation:**
- [design_docs/implemented/v0_19_1/m-stdlib-html.md](../../implemented/v0_19_1/m-stdlib-html.md) — original `std/html` design; lists `parseStream` as Future Work.
- [design_docs/implemented/v0_11_0/m-streaming-zip-xml.md](../../implemented/v0_11_0/m-streaming-zip-xml.md) — the streaming pattern this doc mirrors. `parseFold` / `parseFoldStep` semantics are defined there.
- [internal/builtins/xml.go:189-303](../../../internal/builtins/xml.go#L189) — `_xml_parseElements` reference impl.
- [internal/builtins/xml_fold.go](../../../internal/builtins/xml_fold.go) — `_xml_parseFold` / `_xml_parseFoldStep` reference impls; HTML versions reuse the `FnCallerN` callback shape.
- [internal/builtins/html.go](../../../internal/builtins/html.go) — the file these builtins will extend.

## References

- [Design Axioms](/docs/references/axioms)
- Upstream: [golang.org/x/net/html#Tokenizer](https://pkg.go.dev/golang.org/x/net/html#Tokenizer) — the streaming primitive.
- WHATWG: [HTML Living Standard — Insertion modes](https://html.spec.whatwg.org/multipage/parsing.html#the-insertion-mode) — for documenting the syntactic-vs-spec divergence.
- Filed-from conversation: 2026-05-14, follow-up to v0.19.1's `std/html` ship.

## Future Work

- **`parseDocument`** — full doctype + root sum type, once a round-trip caller materialises.
- **Effectful chunked input** (`parseFoldFromStream`) — when both `std/xml` and `std/html` want a `Stream`-fed variant, design them together.
- **`querySelector` / CSS** — only if a caller surfaces a real use case.
- **Sanitization helper** (`std/html.sanitize`) — XSS-safe subset, only if a web-rendering use case lands.

---

**Document created**: 2026-05-14
**Last updated**: 2026-05-14

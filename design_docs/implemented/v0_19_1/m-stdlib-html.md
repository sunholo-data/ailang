# M-STDLIB-HTML: `std/html` — HTML5 Parser Module

**Status**: Planned
**Target**: v0.20.0
**Priority**: P1 (Medium — unblocks ongoing churn in dependent project)
**Estimated**: 2 days
**Dependencies**: Cross-module ADT import (shipped v0.19.0, confirms `std/html` can `import std/xml (XmlNode)`).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure function over input string; same bytes → same tree. No I/O, no clocks. |
| A2: Replayability | 0 | No trace surface change. |
| A3: Effect Legibility | +1 | `pure func` signatures; no hidden effects vs. `std/xml`. |
| A4: Explicit Authority | 0 | No capabilities involved (string in, ADT out). |
| A5: Bounded Verification | +1 | Reuses 256 depth / 50MB input guardrails from `std/xml`. |
| A6: Safe Concurrency | 0 | Stateless; safe to call from any goroutine. |
| A7: Machines First | +1 | Replaces 300 LOC of AILANG corner-case logic in dependents with one builtin call — measurably less for an LLM to read and reason about. |
| A8: Minimal Syntax | +1 | No new syntax. Two new top-level functions in a new module. |
| A9: Cost Visibility | 0 | Time/memory bounded by the same input-size limit as `std/xml`. |
| A10: Composability | +1 | Returns the existing `XmlNode` ADT — every `std/xml` query function (`findAll`, `findFirst`, `getText`, `getAttr`, `serialize`, …) immediately works on HTML trees. |
| A11: Structured Failure | +1 | Returns `Result[XmlNode, string]` — typed failure, same shape as `std/xml.parse`. |
| A12: System Boundary | 0 | No FFI surface change; uses an already-indirect Go dep. |

**Net Score: +7** → **Decision: Move forward.**

### Hard Violation Check

- [x] A1 (Determinism): pure parse, no nondeterminism
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): collapses 300 LOC of caller code into one builtin

## Problem Statement

Real-world HTML (Word exports, CMS output, hand-written marketing pages) is rarely well-formed XML. AILANG today only ships `std/xml`, which is strict by design. Downstream projects that need to extract content from HTML have to write their own tolerance layer in AILANG.

**Current State:**
- `sunholo/ailang-parse` v0.13.0 ships ~300 LOC of HTML5-tolerance shims in [docparse/services/html_parser.ail](https://github.com/sunholo-data/ailang-parse/blob/v0.13.0/docparse/services/html_parser.ail): boolean-attribute normalization for 23 HTML5 booleans, a tag-stack auto-closer for overlapping/unclosed tags, `<script>` stripping, `<!--[if IE]>` stripping, and HTML-comment stripping.
- Canonical regression: `data/test_files/sunholo_homepage.html` went from 1 error block → 13 structured blocks once the sanitizer was wired up. Works today, but each new real-world HTML source surfaces a new edge case (Word's `<o:p>` namespaces, mixed-case tags, attributes with embedded quotes, …).
- The WHATWG HTML5 spec is ~150 pages of state machine. Re-implementing it in AILANG is a category error.

**Impact:**
- Any AILANG project ingesting web content (RAG pipelines, scrapers, doc converters) will hit the same wall. Today the only available answer is "fork ailang-parse's sanitizer."
- The Go ecosystem already has the canonical answer: `golang.org/x/net/html` is a full WHATWG HTML5-compliant parser. It is **already in `go.mod` as an indirect dep** (v0.53.0). Promoting it to direct adds zero new transitive weight.

## Goals

**Primary Goal:** Ship a `std/html` module that returns the same `XmlNode` ADT as `std/xml`, so HTML and XML trees are interchangeable to downstream code.

**Success Metrics:**
- `ailang run examples/runnable/html_parser.ail` parses the ailang-parse canonical regression file and yields a non-error block tree.
- `ailang-parse` can delete its `htmlSanitize` pipeline (~300 LOC) and call `std/html.parse` directly, with its existing tests still passing.
- No new direct dependencies beyond promoting `golang.org/x/net` from indirect → direct in `go.mod`.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Reuse `std/xml`'s `XmlNode` ADT vs. define a new `HtmlNode` | Determines whether `findAll`/`getText`/etc. are shared or duplicated. Wrong choice doubles the API surface forever. | human | design | high |
| Drop or expose `DoctypeNode` and `DocumentNode` | Surface shape callers see at top level. Changing later is a breaking API change. | human | design | med |
| Where do `makeXmlElement`/`makeXmlText`/`makeXmlComment` live (xml.go exports vs. shared file) | Avoids duplicate ADT constructor logic drifting between two builtins files. | agent | compile | low |
| Skip `<script>`/`<style>` contents in the tree, or pass them through as Text children | API contract for callers. ailang-parse currently strips `<script>` itself. | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **ADT reuse:** `std/html` imports `XmlNode` from `std/xml`. Single source of truth for the tree shape. (Cross-module ADT import was fixed in v0.19.0 — see registry message 4ada0a1b.)
- [x] **Top-level shape:** Drop `DoctypeNode` and the synthetic `DocumentNode` root in v1. `parse` returns the `<html>` element directly (`Element("html", …, …)`). Rationale: every known use case is content extraction, not document round-trip. Re-add later as a `parseDocument` variant if needed.
- [x] **`<script>`/`<style>`:** Pass through as `Element` with Text children — net/html already does this. Stripping is a caller-side concern (one line of AILANG with `findAll`/filter).

## Solution Design

### Overview

Add a new builtin file `internal/builtins/html.go` that wraps `golang.org/x/net/html`. It mirrors `internal/builtins/xml.go`'s structure: a small init() registering builtins, two parse entrypoints, and a recursive converter from `*html.Node` to AILANG's `XmlNode` `TaggedValue`. The AILANG-facing surface is a 30-line `std/html.ail` file declaring two `pure func`s.

### Architecture

**Components:**
1. **`std/html.ail`** — Module declaration. `import std/xml (XmlNode)`, `import std/result (Result)`. Exports `parse` and `parseFragment` as `pure func`s bound to `_html_parse` / `_html_parseFragment` builtins.
2. **`internal/builtins/html.go`** — Two registered builtins; a recursive `htmlNodeToXmlNode(*html.Node) eval.Value` converter; reuse of xml.go's `makeXml*` constructors (promoted from package-private to package-internal — they already live in `internal/builtins` so this is a no-op rename to keep them accessible from `html.go`).
3. **`internal/builtins/html_test.go`** — Golden tests on real-world HTML fragments (unclosed tags, boolean attrs, `<script>` blocks, embedded comments, Word `<o:p>` namespace, mixed case).

**Converter rules** (the only nontrivial logic):
- `html.DocumentNode` → recurse into children, return the first `ElementNode` child (the `<html>` element).
- `html.ElementNode` → `makeXmlElement(node.Data, attrs, children)`. `attrs` flattens `html.Attribute` ignoring `Namespace` field (HTML5 doesn't really namespace). Tag name is lower-cased by net/html already.
- `html.TextNode` → `makeXmlText(node.Data)`.
- `html.CommentNode` → `makeXmlComment(node.Data)`.
- `html.DoctypeNode` → skip (do not emit).
- `parseFragment` uses `html.ParseFragment(reader, &html.Node{Type: html.ElementNode, DataAtom: atom.Body, Data: "body"})` and returns `Result[[XmlNode], string]` — the list of fragment roots, with text nodes preserved.

**Guardrails (reused from xml.go):**
- 50MB input-size limit (return `Err` before parsing).
- 256 depth limit enforced during conversion (panic-free; return `Err` if exceeded).
- Recursion is iterative-friendly: net/html builds the tree eagerly, so converter recursion depth is bounded by the parsed tree depth, which is bounded by the 256 limit.

### Implementation Plan

**Phase 1: Builtins (~4 hours)**
- [ ] Promote `golang.org/x/net` to direct dep in `go.mod` (`go get golang.org/x/net@latest && go mod tidy`).
- [ ] Create `internal/builtins/html.go` with `registerHtmlParse()` + `registerHtmlParseFragment()`.
- [ ] Implement `htmlNodeToXmlNode` converter reusing `makeXmlElement` / `makeXmlText` / `makeXmlComment` / `makeXmlAttr` from xml.go.
- [ ] Enforce 256 depth + 50MB size limits.
- [ ] Wire `init()` to call both register functions.

**Phase 2: Module surface (~1 hour)**
- [ ] Create `std/html.ail`: `module std/html`, `import std/xml (XmlNode)`, `import std/result (Result)`, two `pure func` declarations binding to `_html_parse` and `_html_parseFragment`.
- [ ] Verify `std/embed.go`'s `*.ail` glob picks it up (it should — no edits needed).

**Phase 3: Tests + example (~2 hours)**
- [ ] `internal/builtins/html_test.go`: golden tests for unclosed tags, `<br>` boolean attrs, `<script>` passthrough, Word `<o:p>`, mixed-case `<DIV>`, oversized input rejection, malformed surrogate handling.
- [ ] `examples/runnable/html_parser.ail` mirroring `examples/runnable/xml_parser.ail`.
- [ ] Optional: copy ailang-parse's `sunholo_homepage.html` into `internal/builtins/testdata/` as a fixture.

**Phase 4: Docs + announce (~1 hour)**
- [ ] `docs/docs/reference/stdlib/std-html.md` parallel to `std-xml.md`.
- [ ] Add `std/html` to the Stdlib Index page.
- [ ] CHANGELOG entry under v0.20.0.
- [ ] Reply to inbox msg 62a9b106 with ship confirmation.

### Files to Modify/Create

**New files:**
- `internal/builtins/html.go` — ~250–350 LOC. Two builtins + converter + guardrails.
- `internal/builtins/html_test.go` — ~200 LOC.
- `internal/builtins/testdata/sunholo_homepage.html` — fixture from ailang-parse.
- `std/html.ail` — ~30 LOC.
- `examples/runnable/html_parser.ail` — ~40 LOC.
- `docs/docs/reference/stdlib/std-html.md` — ~80 LOC.

**Modified files:**
- `go.mod` / `go.sum` — promote `golang.org/x/net` from indirect to direct.
- `CHANGELOG.md` — v0.20.0 entry.
- `docs/docs/reference/stdlib/index.md` — add `std/html` row.

## Examples

### Example 1: Parsing real-world HTML

**Before** (in ailang-parse v0.13.0 — abbreviated):
```ailang
-- ~300 LOC of htmlSanitize: boolean attrs, tag-stack auto-closer,
-- <script> strip, conditional comment strip, comment strip, …
let cleaned = htmlSanitize(raw) in
match xml.parse(cleaned) {
  Ok(tree) => extractBlocks(tree),
  Err(msg) => Err("html: " ++ msg)
}
```

**After:**
```ailang
import std/html (parse)

match parse(raw) {
  Ok(tree) => extractBlocks(tree),
  Err(msg) => Err("html: " ++ msg)
}
```

`extractBlocks` is unchanged — it operates on `XmlNode`, which `std/html.parse` returns just like `std/xml.parse`.

### Example 2: Fragment parsing

```ailang
import std/html (parseFragment)
import std/xml (findAll, getText)

match parseFragment("<p>hello <b>world</b></p><p>second</p>") {
  Ok(nodes) => {
    let paragraphs = list.flatMap(nodes, \n. findAll(n, "p")) in
    list.map(paragraphs, getText)
    -- => ["hello world", "second"]
  },
  Err(msg) => []
}
```

## Success Criteria

- [ ] `ailang run examples/runnable/html_parser.ail` parses the sunholo_homepage.html fixture and prints ≥10 structured blocks.
- [ ] `std/html.parse("<p>unclosed<p>second")` returns `Ok(...)` with two sibling `<p>` elements (lenient — not an error).
- [ ] `std/html.parse("<input type=text disabled>")` returns the boolean `disabled` attribute as `{name: "disabled", value: ""}` (net/html convention).
- [ ] 50MB+ input returns `Err`, doesn't OOM.
- [ ] All existing `std/xml` tests still pass (no regression — we only added).
- [ ] `go mod tidy` shows `golang.org/x/net` as a direct dep with no other module-graph changes.
- [ ] CHANGELOG entry + docs page shipped.

## Testing Strategy

**Unit tests** (`internal/builtins/html_test.go`):
- Unclosed `<p><p>` → two siblings, not nested.
- `<DIV>` → lower-cased to `div` (HTML5 convention from net/html).
- `<input disabled>` → attr with empty string value.
- `<!-- comment -->` → `Comment` node preserved.
- `<script>alert(1)</script>` → `Element("script", [], [Text("alert(1)")])` (passthrough; caller filters).
- Word's `<o:p>...</o:p>` → tag is `o:p` (net/html keeps the prefix in `Data`).
- Oversized input (50MB+1) → `Err`.
- Empty string → `Ok(Element("html", …, …))` (net/html synthesizes the document shell).

**Integration tests:**
- Parse the sunholo_homepage.html fixture, assert tree depth ≤ 256, assert ≥1 `<article>` or `<main>` element present.

**Manual testing:**
- Wire it into ailang-parse on a branch, run their docparse regression suite, confirm parity (or improvement) on the 13-block goal.

## Deferred Decisions

- **Whether to add `parseDocument`** returning the full doctype + document root for round-trip use cases — deferred until a caller actually needs it. Agent may choose not to ship in v1.
- **`Result[XmlNode, string]` vs. richer error type** — staying with `string` matches `std/xml`; revisit if/when `std/xml` upgrades.
- **Attribute ordering** — net/html preserves source order; we pass that through. If the spec ever pins canonical ordering, revisit.

## Non-Goals

- **HTML serialization** — `std/xml.serialize` already works on `XmlNode` and emits XML-ish output. If callers want HTML5-conformant serialization (e.g., self-closing `<br>`), file a follow-up.
- **DOM mutation API** — `XmlNode` is immutable. Use constructors (`xmlElement`/`xmlText`) the same way as `std/xml`.
- **CSS selector queries** — `findAll`/`findFirst` by tag name are sufficient for v1. Selector parsing belongs in a separate module if ever justified.
- **Sanitization / XSS scrubbing** — out of scope. The parser returns whatever is in the input; sanitization is a caller policy.

## Timeline

**Day 1 (~5 hours):**
- Phase 1 (builtins)
- Phase 2 (module surface)

**Day 2 (~3 hours):**
- Phase 3 (tests + example)
- Phase 4 (docs + CHANGELOG)
- ailang-parse integration sanity check

**Total: ~8 hours / 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `import std/xml (XmlNode)` from another stdlib module hits a regression. | High | Pre-flight: write a one-line test importing `XmlNode` into a throwaway module before starting Phase 1. v0.19.0 fixed cross-module ADT imports (msg 4ada0a1b), so this is expected to work — but verify. |
| net/html version pinning surfaces a transitive incompat. | Low | It's already indirect at v0.53.0 — promotion is a metadata change, not a version bump. |
| `htmlNodeToXmlNode` recursion blows the Go stack on adversarial inputs. | Med | The 256 depth limit caps recursion well before Go's default stack. Add an explicit depth counter, return `Err` on overflow. |
| Subtle divergence between `std/xml` and `std/html` over the same `XmlNode` shape (e.g., attribute namespace handling). | Med | Single converter codepath; cross-test by parsing identical well-formed XML through both modules and asserting equal trees. |

## Related Documents

**Prior art — required reading before implementation:**
- [design_docs/implemented/v0_7_3/m-stdlib-xml.md](../implemented/v0_7_3/m-stdlib-xml.md) — original `std/xml` design; the API and ADT shape this doc reuses.
- [design_docs/implemented/v0_9_4/m-stdlib-xml-improvements.md](../implemented/v0_9_4/m-stdlib-xml-improvements.md) — constructors (`xmlElement`/`xmlText`/`xmlComment`) and serialize API.
- [design_docs/implemented/v0_11_0/m-bytecode-xml-builtins.md](../implemented/v0_11_0/m-bytecode-xml-builtins.md) — wiring xml builtins through the bytecode VM. `std/html` must follow the same registration pattern.
- [design_docs/implemented/v0_11_0/m-streaming-zip-xml.md](../implemented/v0_11_0/m-streaming-zip-xml.md) — streaming pattern, for future `parseStream` follow-up.

**Auto-search results** (lower relevance, kept for completeness):
- [design_docs/implemented/v0_5_9/m-codegen-stdlib-math.md](../implemented/v0_5_9/m-codegen-stdlib-math.md)
- [design_docs/implemented/v0_11_0/m-std-string-perf.md](../implemented/v0_11_0/m-std-string-perf.md)
- [design_docs/implemented/v0_5_7/m-dx11-stdlib-discovery.md](../implemented/v0_5_7/m-dx11-stdlib-discovery.md)

## References

- [Design Axioms](/docs/references/axioms)
- Inbox message: `62a9b106-d4a4-4de1-a555-9bab9036e5ba` (sunholo/ailang-parse, 2026-05-13).
- Upstream: [golang.org/x/net/html](https://pkg.go.dev/golang.org/x/net/html) — WHATWG HTML5 parser.
- Real-world driver: [sunholo/ailang-parse v0.13.0 — html_parser.ail](https://github.com/sunholo-data/ailang-parse/blob/v0.13.0/docparse/services/html_parser.ail).

## Future Work

- `parseDocument` returning a `DocumentNode | XmlNode` sum if doctype round-trip becomes a real need.
- `parseStream` for incremental parsing of multi-megabyte HTML (mirror M-STREAMING-ZIP-XML).
- CSS selector queries (`querySelector`/`querySelectorAll`) — only if a caller surfaces a real use case; tag-based `findAll` covers ~90% today.
- Sanitization helper (`std/html.sanitize`) for an XSS-safe subset, only if a web-rendering use case lands.

---

**Document created**: 2026-05-13
**Last updated**: 2026-05-13

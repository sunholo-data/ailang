## M-STDLIB-XML-LENIENT: `std/xml` — Lenient Parsing for Real-World (Malformed) XML

**Status**: IMPLEMENTED — shipped in v0.24.0 (2026-06-04)
**Target**: v0.24.0
**Priority**: P1 (High — blocking sunholo/ailang-parse on real production documents)
**Estimated**: 1 day
**Dependencies**: None. Additive to existing `std/xml`; `XmlNode` ADT unchanged.

**Reported from**: ailang-parse (docparse) v0.20.2, AILANG v0.23.0
**Inbox refs**: `msg_20260602_203603_276a4d86` (+ duplicate `c60a9501`)

---

## Verdict: VALID — implement as an opt-in lenient path, never by mutating `parse`

The request is a real, reproducible limitation rooted in Go's `encoding/xml` being a
strict (XML 1.0 well-formedness) parser. It is correctly scoped (a `std/xml`-level
concern), additive, and the reporter already asked for the right shape (opt-in, not a
behavior change to `parse`). One hard constraint governs the design (see "No Silent
Fallbacks" below): **`parse` must stay strict**; leniency is a separate, named entry point.

---

## Problem Statement

`std/xml.parse` wraps Go's `encoding/xml` decoder ([internal/builtins/xml.go:174](../../internal/builtins/xml.go#L174)):

```go
decoder := xml.NewDecoder(strings.NewReader(input))
children, err := parseXmlChildren(decoder, 0, nil)
```

Go's decoder is strict: a bare/unescaped `&` (not starting a valid entity) aborts the
**entire** document with zero recovered content. Real-world XML — especially Office
(`content.xml`, DOCX/PPTX/XLSX) and HTML — is routinely emitted by non-reference tools
that fail to escape `&` → `&amp;`.

### Minimal repro

```ailang
import std/xml (parse)
parse("<r><p>Apex Consulting & Partners</p></r>")
-- => Err("XML parse error: XML syntax error on line 1: invalid character entity & (no semicolon)")
```

### Where it bites

A user submitted an ODT invoice (`apex-consulting-invoice-2026-103.odt`) via the Python
SDK. `odt_parser.ail → std/xml.parse(content.xml)` failed:

```
ODT XML parse error: XML syntax error on line 36: invalid character entity & (no semicolon)
```

The `content.xml` carried a bare `&` (a company name "Apex Consulting & Partners", an
"R&D" line item, or a URL query string). The file is technically invalid XML but opens
fine in lenient consumers (LibreOffice, browsers).

**Not ODT-specific.** This is a whole-class failure across every XML-backed docparse
parser — DOCX, PPTX, XLSX, ODT, ODP, ODS, HTML — whenever an upstream tool emits a stray
`&`. Same family as the previously-noted HTML strict-XML limitation.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Sanitization is a pure, total string→string rewrite with a fully-specified rule set; same input → same output. |
| A2: Replayability | 0 | No trace surface change. |
| A3: Effect Legibility | +1 | All new functions are `pure func`; no hidden effects. |
| A4: Explicit Authority | 0 | No capabilities. |
| A5: Bounded Verification | 0 | Reuses existing 50 MB / 256-depth bounds. |
| A6: Safe Concurrency | 0 | Stateless, read-only. |
| A7: Machines First | +1 | One opt-in call lets every docparse parser recover instead of each hand-rolling escape logic. |
| A8: Minimal Syntax | +1 | No new syntax; 2 functions added to `std/xml`. |
| A9: Cost Visibility | +1 | Cost model documented: `sanitizeXml` is O(n) single-pass; `parseLenient` = `sanitizeXml` + `parse`. |
| A10: Composability | +1 | Same `XmlNode` ADT; `sanitizeXml` composes with `parse`, `parseFold`, `parseElements`, and benefits `std/html`. |
| A11: Structured Failure | +1 | Returns `Result`; lenient path still surfaces unrecoverable errors instead of guessing. |
| A12: System Boundary | 0 | No FFI surface change; same builtin registration path. |

**Net Score: +7** → **Decision: Move forward.**

### Hard Violation Check

- [x] A1 (Determinism): rewrite rules are explicit and total.
- [x] A3 (Effects): pure.
- [x] A4 (Authority): none.
- [x] **No Silent Fallbacks (CLAUDE.md §2)**: `parse` is unchanged and stays strict.
      Leniency is opt-in via a distinct, self-describing name. Callers cannot accidentally
      get repaired data; they must ask for it. The repair is a documented, inspectable
      transform (`sanitizeXml` is exported so callers can see exactly what changed), not a
      hidden swallow of malformed input.

---

## Proposed API

Add to [std/xml.ail](../../std/xml.ail):

```ailang
-- Repair common well-formedness offenders so a strict parser accepts the input.
-- Currently: escapes a bare '&' that does not begin a valid entity reference
-- (&amp; &#123; &#xAB; &name; are left intact). Pure, single-pass, total.
export pure func sanitizeXml(xml: string) -> string = _xml_sanitize(xml)

-- Lenient parse: sanitizeXml(xml) then parse. For real-world XML from
-- non-reference generators (Office content.xml, HTML5). Strict callers should
-- keep using `parse`.
export pure func parseLenient(xml: string) -> Result[XmlNode, string] =
  parse(sanitizeXml(xml))
```

`parseLenient` can be pure AILANG (composition of `sanitizeXml` + `parse`), so only **one**
new Go builtin is required: `_xml_sanitize`.

### Why expose `sanitizeXml` separately

1. docparse parsers that use `parseFold` / `parseElements` (streaming) need to sanitize
   the string *before* their own parse call — `parseLenient` alone wouldn't help them.
2. Keeps the transform inspectable/testable in isolation (A1, No-Silent-Fallbacks).

---

## The repair rule (the one subtle part)

Naive "replace every `&` with `&amp;`" is **wrong** — it double-escapes valid entities
(`&amp;` → `&amp;amp;`). The repair must escape a `&` only when it does **not** begin a
valid entity reference. The well-known negative-lookahead pattern:

```
&(?!#[0-9]+;|#x[0-9a-fA-F]+;|[A-Za-z][A-Za-z0-9]*;)   ->   &amp;
```

Go's `regexp` (RE2) has **no lookahead**, so implement the scan manually in
`_xml_sanitize` (single pass): on each `&`, peek ahead for `#NNN;`, `#xHH;`, or `name;`;
if none matches, emit `&amp;`, else copy through verbatim. This reuses the entity-name
grammar already implied by `_escapeXml` ([xml_serialize.go:237](../../internal/builtins/xml_serialize.go#L237)).

### Scope decision: v1 handles bare `&` only

The reporter's "(optionally)" items are deliberately **deferred**:

| Offender | v1? | Rationale |
|----------|-----|-----------|
| Bare `&` not a valid entity | ✅ Yes | The actual reported failure; safe, well-defined, single-pass. |
| Unknown named entities (`&nbsp;`) | ⚠️ Maybe | Real for HTML-ish input. Cleaner fix: register an entity map on the decoder, not string rewriting. Track separately. |
| Stray `<` in text | ❌ No | Genuinely ambiguous (is `<` a tag or text?). Heuristic repair risks corrupting structure → violates A1's "total & predictable". Out of scope. |

Shipping the bare-`&` repair alone resolves the reported production failures. Over-reaching
into `<` recovery is where lenient parsers become unpredictable.

---

## Implementation Plan

1. **Go builtin** `_xml_sanitize` in `internal/builtins/xml.go` (or a new
   `xml_sanitize.go`): `string -> string`, manual single-pass bare-`&` escaper. Register
   as a pure builtin (mirror `_escapeXml` registration in
   [xml_serialize.go](../../internal/builtins/xml_serialize.go)).
2. **stdlib** add `sanitizeXml` + `parseLenient` to [std/xml.ail](../../std/xml.ail) and
   export them.
3. **Tests** (`internal/builtins/xml_test.go`): bare `&`; `&` before `amp;`/`#123;`/`#xAB;`
   (no double-escape); trailing `&` at EOF; `&` inside attribute values; idempotence
   (`sanitizeXml(sanitizeXml(x)) == sanitizeXml(x)`); the reporter's exact repro round-trips
   through `parseLenient`.
4. **Example** `examples/runnable/xml_lenient.ail` demonstrating `parseLenient` on dirty
   input (per coding-standards: every feature needs an example).
5. **Docs**: CHANGELOG entry; `ailang prompt` / stdlib docs for `std/xml`.

## Acceptance Criteria

- [ ] `parseLenient("<r><p>Apex Consulting & Partners</p></r>")` → `Ok(...)` with text
      `Apex Consulting & Partners`.
- [ ] `sanitizeXml` does not alter input that is already well-formed (no double-escaping of
      `&amp;`, `&#123;`, `&#xAB;`, `&lt;`).
- [ ] `sanitizeXml` is idempotent.
- [ ] `parse` (strict) behavior is **unchanged** — existing tests pass untouched.
- [ ] Round-trips the reporter's ODT-class input.

## Non-Goals

- Changing `parse` to be lenient by default.
- Stray-`<` recovery / full HTML5 tag-soup parsing.
- Unknown named-entity resolution (tracked as a possible follow-up via decoder entity map).

## Out-of-band Notes

- Two identical inbox copies were received (`276a4d86`, `c60a9501`); ack both.
- Relationship to the prior HTML strict-XML limitation: same root cause (strict
  `encoding/xml`). `sanitizeXml` is the shared mitigation `std/html` can also adopt.

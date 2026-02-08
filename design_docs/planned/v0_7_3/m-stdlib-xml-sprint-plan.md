# Sprint Plan: M-STDLIB-XML — XML Parsing Standard Library

## Summary
Implement XML parsing and querying as pure Go builtins under `std/xml`, providing an `XmlNode` ADT with tree construction and query functions for pattern matching on XML document structures.

**Duration:** 2.5 days (~17 hours)
**Dependencies:** None — uses Go stdlib `encoding/xml`. Companion to M-STDLIB-ZIP but independent.
**Risk Level:** Medium (namespace handling adds complexity; new ADT type registration)
**Design Doc:** [m-stdlib-xml.md](m-stdlib-xml.md)

## Current Status Analysis

### Completed Recently (reference velocity)
- ✅ M-STDLIB-DATETIME: ~530 LOC impl + ~390 LOC tests in ~2 days (12 builtins)
- ✅ std/json decode: ~380 LOC (tree ADT builder — closest pattern match for XML)
- ✅ std/json encode: ~264 LOC
- ✅ json_decode_test: ~517 LOC (tree ADT tests — reference for XML tests)

### Velocity
- Recent average: ~300-500 LOC/day (implementation + tests)
- Tree ADT pattern (json_decode): ~380 LOC for parser + ADT constructors
- Query functions: ~30-50 LOC each (simple recursive traversals)
- Test coverage: ~1.3x test LOC vs implementation LOC for ADT parsers

### Remaining from Design Doc
- ⏳ 7 builtins: `_xml_parse`, `_xml_findAll`, `_xml_findFirst`, `_xml_getText`, `_xml_getAttr`, `_xml_getChildren`, `_xml_getTag`
- ⏳ XmlNode ADT: `Element | Text | CData | Comment` (4 constructors)
- ⏳ Namespace prefix handling for OOXML compatibility
- ⏳ Security: depth limit, size limit
- ⏳ Tests + example file

## Proposed Milestones

### Milestone 1: XmlNode ADT + Parser Builtin
**Goal:** Implement `_xml_parse` that converts an XML string into an `XmlNode` tagged value tree, plus register all 7 builtin specs.
**Estimated:** ~250 LOC implementation + ~0 LOC tests = ~250 LOC
**Duration:** 5 hours

**Tasks:**
1. Create `internal/builtins/xml.go` — register all 7 `BuiltinSpec` entries with full metadata
2. Add `makeXmlElement`, `makeXmlText`, `makeXmlCData`, `makeXmlComment` constructors (pattern: `makeJString`, `makeJNumber` from json_decode.go)
3. Add shared `makeOk` / `makeErr` Result helpers (or reuse from M-STDLIB-ZIP if done first)
4. Implement `xmlParseImpl` using `encoding/xml` token decoder with recursive `parseChildren`
5. Handle namespace prefixes: track `xmlns:*` attributes, reverse-map URIs to prefixes in tag names
6. Verify `ailang doctor builtins` passes, `make build` succeeds

**Acceptance Criteria:**
- [ ] 7 BuiltinSpec entries registered with Module `"std/xml"`
- [ ] `_xml_parse("<root/>")` returns `Ok(Element("root", [], []))`
- [ ] `_xml_parse("<p>hello</p>")` returns `Ok(Element("p", [], [Text("hello")]))`
- [ ] `_xml_parse("not xml")` returns `Err("XML parse error: ...")`
- [ ] Namespace prefixes preserved: `<w:p>` → `Element("w:p", ...)`
- [ ] `ailang doctor builtins` passes
- [ ] Linting clean

**Risks:**
- Go's `encoding/xml` resolves namespace prefixes to full URIs — need reverse mapping. Mitigation: fallback to local name only if reverse mapping too complex
- Recursive parsing may need depth guard for large documents — addressed in M3

### Milestone 2: Query Functions
**Goal:** Implement 6 query functions that operate on the XmlNode tagged value tree.
**Estimated:** ~200 LOC implementation + ~250 LOC tests = ~450 LOC
**Duration:** 5 hours

**Tasks:**
1. Create `internal/builtins/xml_query.go` with all 6 query implementations
2. `xmlFindAllImpl` — recursive depth-first search matching tag name, returns `[XmlNode]`
3. `xmlFindFirstImpl` — same but returns `Option[XmlNode]` (Some/None)
4. `xmlGetTextImpl` — recursive text collection across Element/Text/CData nodes
5. `xmlGetAttrImpl` — linear scan of attributes list, returns `Option[string]`
6. `xmlGetChildrenImpl` — extract children field from Element, return `[]` for non-Element
7. `xmlGetTagImpl` — extract tag field from Element, return `""` for non-Element
8. Create `internal/builtins/xml_test.go` with test fixture helpers and tests for parse + all queries

**Acceptance Criteria:**
- [ ] `findAll(root, "item")` returns all matching descendant elements
- [ ] `findFirst(root, "title")` returns `Some(Element(...))` or `None`
- [ ] `getText(Element("p", [], [Text("hello "), Element("b", [], [Text("world")])]))` returns `"hello world"`
- [ ] `getAttr(element, "class")` returns `Some("main")` for matching attribute
- [ ] `getAttr(element, "missing")` returns `None`
- [ ] `getChildren(element)` returns direct children list
- [ ] `getTag(Element("div", ...))` returns `"div"`, `getTag(Text(...))` returns `""`
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- TaggedValue field access by index (Fields[0], Fields[2]) is fragile — Mitigation: add clear comments documenting field order, consider helper accessors

### Milestone 3: Security Hardening + Namespace Edge Cases
**Goal:** Add depth limits, size limits, and handle namespace edge cases for OOXML.
**Estimated:** ~80 LOC implementation + ~100 LOC tests = ~180 LOC
**Duration:** 3 hours

**Tasks:**
1. Add `maxDepth = 256` guard to recursive parser (return error if exceeded)
2. Add `maxInputSize = 50MB` check before parsing (configurable via `AILANG_XML_MAX_SIZE`)
3. Test OOXML fragments: `<w:p><w:r><w:t>Hello</w:t></w:r></w:p>` with proper prefix resolution
4. Handle edge cases: self-closing elements, empty elements, CDATA sections, comments
5. Test mixed content: `<p>text <b>bold</b> more text</p>`
6. Write security-focused tests: deeply nested XML, huge input, malformed XML

**Acceptance Criteria:**
- [ ] XML nested >256 levels returns `Err("XML parse error: maximum depth exceeded")`
- [ ] Input >50MB returns `Err("XML input too large: ...")`
- [ ] OOXML fragment `<w:t>Hello</w:t>` parses with prefix-form tag names
- [ ] CDATA sections produce `CData(content)` nodes
- [ ] Comments produce `Comment(content)` nodes
- [ ] Mixed content preserves text ordering
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Go's xml.Decoder may handle some edge cases differently than expected — Mitigation: test with real OOXML excerpts

### Milestone 4: Integration Test + Example + Docs
**Goal:** End-to-end test combining ZIP + XML (if ZIP available), working example, CHANGELOG.
**Estimated:** ~40 LOC implementation + ~60 LOC tests + ~30 LOC example = ~130 LOC
**Duration:** 2 hours

**Tasks:**
1. Create integration test: parse a multi-element XML document, query with findAll + getText
2. Create `examples/xml_parser.ail` demonstrating parseXml, findAll, getText, getAttr
3. If M-STDLIB-ZIP is done: create `examples/docparse_demo.ail` combining ZIP + XML
4. Verify example works: `ailang run --entry main examples/xml_parser.ail`
5. Update CHANGELOG.md with new `std/xml` module
6. Run `make verify-examples` to confirm examples pass

**Acceptance Criteria:**
- [ ] Integration test parses complex XML and extracts data via queries
- [ ] `examples/xml_parser.ail` exists and runs successfully
- [ ] `make verify-examples` passes (new example included)
- [ ] CHANGELOG.md updated
- [ ] `make test` passes (all tests)
- [ ] `make lint` passes

**Risks:**
- Example may need `std/xml` module path recognized by module loader — same risk as ZIP, same mitigation

## Success Metrics
- Test coverage: >90% for `internal/builtins/xml.go` and `xml_query.go`
- New tests: ~25-30 test cases across `xml_test.go`
- Examples passing: `examples/xml_parser.ail` verified working
- Documentation: CHANGELOG.md updated
- All tests passing: `make test` ✅
- All linting passing: `make lint` ✅
- Total new LOC: ~570 implementation + ~410 tests = ~980 LOC

## Dependencies
- None — Go `encoding/xml` is a stdlib package
- M-STDLIB-ZIP is a companion but NOT a blocker. If both complete, a combined demo is a bonus.
- Shared `makeOk`/`makeErr` helpers: if ZIP is implemented first, reuse; otherwise create in xml.go and extract later

## Open Questions
- Should `getText` include whitespace-only text nodes? Design doc trims whitespace. Leaning: trim by default, matches typical XML processing expectations.
- Should we register `XmlNode` as a known type in the type system (like `Json`)? Probably yes for pattern matching — investigate during M1.

## Notes
- Closest implementation reference: `internal/builtins/json_decode.go` (tree ADT from token stream)
- XML is more complex than JSON: attributes, namespaces, mixed content, CDATA, comments
- All 7 functions are pure (no effects) — simpler registration than ZIP (no FS effect wiring)
- The `XmlNode` ADT uses `TaggedValue` with `ModulePath: "std/xml"` (same pattern as `Json` ADT)

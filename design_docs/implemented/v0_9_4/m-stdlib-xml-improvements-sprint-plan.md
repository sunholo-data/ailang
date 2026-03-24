# Sprint Plan: M-STDLIB-XML — XmlNode Constructors, writeEntryBytes, escapeXml Tests

## Summary
Add XmlNode constructor builtins (`xmlElement`, `xmlText`, `xmlComment`) to enable programmatic XML building, add `writeEntryBytes` for binary ZIP entries (images in DOCX/PPTX), and backfill tests/examples for the already-implemented `escapeXml`.

**Duration:** 1.5 days (~5-6 hours)
**Dependencies:** None (all prerequisites are in place — `BytesValue` exists, ZIP builtins exist)
**Risk Level:** Low
**Design Doc:** `design_docs/planned/v0_9_4/m-stdlib-xml-improvements.md`

## Current Status Analysis

### Completed Recently
- ✅ `escapeXml` builtin registered (`internal/builtins/xml_serialize.go:206-241`)
- ✅ `escapeXml` exported in `std/xml.ail:86`
- ✅ Internal Go helpers exist: `makeXmlElement`, `makeXmlText`, `makeXmlComment` (`internal/builtins/xml.go:38-68`)
- ✅ XML serialize + serializeWithDecl builtins working

### Velocity
- Recent average: ~200-400 LOC/day for stdlib builtins (M-STDLIB-CRYPTO was ~350 LOC)
- This sprint: ~430 LOC estimated (80 + 180 + 140 + 30)

### Remaining from Design Doc
- ⏳ Phase 1 completion: Tests + example for `escapeXml` (~50 LOC)
- ⏳ Phase 2: XmlNode constructor builtins (~150 LOC)
- ⏳ Phase 3: `writeEntryBytes` for binary ZIP entries (~120 LOC)
  - `BytesValue` already exists in `internal/eval/value.go:47`
  - `_zip_readEntryBytes` already returns base64 — `writeEntryBytes` is the write counterpart
  - `_zip_createArchive` currently only handles string content (line 375) — needs bytes path

## Proposed Milestones

### Milestone 1: M1_ESCAPEXML_TESTS — escapeXml Tests & Example
**Goal:** Backfill tests and example file for the already-implemented `escapeXml` builtin
**Estimated:** 30 LOC implementation + 50 LOC tests = ~80 LOC
**Duration:** 30 minutes

**Tasks:**
- Add Go unit tests in `internal/builtins/xml_serialize_test.go` for `escapeXml` edge cases
  - `<`, `>`, `&`, `"`, `'` escaping
  - Already-escaped text (double-escape check)
  - Empty string
  - Mixed content
- Create `examples/xml_escape.ail` demonstrating `escapeXml` usage
- Run `make test` and `make verify-examples`

**Acceptance Criteria:**
- [ ] Go unit tests for `escapeXml` pass with edge cases
- [ ] `examples/xml_escape.ail` runs successfully
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:** None — function already works, just needs test coverage.

### Milestone 2: M2_XMLNODE_CONSTRUCTORS — XmlNode Constructor Builtins
**Goal:** Expose `xmlElement`, `xmlText`, `xmlComment` as AILANG builtins so users can build XML programmatically instead of string concatenation
**Estimated:** 120 LOC implementation + 60 LOC tests = ~180 LOC
**Duration:** 1.5 hours

**Tasks:**
- Register 3 new builtins in `internal/builtins/xml_serialize.go` (or new file `xml_constructors.go`):
  - `_xmlElement(tag: string, attrs: [{name: string, value: string}], children: [XmlNode]) -> XmlNode`
  - `_xmlText(content: string) -> XmlNode`
  - `_xmlComment(content: string) -> XmlNode`
- Each wraps existing `makeXmlElement`, `makeXmlText`, `makeXmlComment` from `xml.go`
- Export in `std/xml.ail`:
  ```ailang
  export pure func xmlElement(tag: string, attrs: [{name: string, value: string}], children: [XmlNode]) -> XmlNode = _xmlElement(tag, attrs, children)
  export pure func xmlText(content: string) -> XmlNode = _xmlText(content)
  export pure func xmlComment(content: string) -> XmlNode = _xmlComment(content)
  ```
- Add Go unit tests: construct nodes → serialize → verify output
- Create `examples/xml_build.ail` — build a small XML document programmatically
- Update `std/xml.ail` module header comment with new exports

**Acceptance Criteria:**
- [ ] `xmlElement`, `xmlText`, `xmlComment` builtins registered and callable
- [ ] Round-trip test: construct XmlNode → serialize → matches expected XML string
- [ ] `examples/xml_build.ail` runs and produces correct XML output
- [ ] `make test` passes
- [ ] `make lint` clean
- [ ] CHANGELOG.md updated

**Risks:**
- Type signatures for list/record args need to match AILANG's type system — low risk, same pattern used by `serialize`

### Milestone 3: M3_WRITE_ENTRY_BYTES — Binary ZIP Entry Writing
**Goal:** Add `writeEntryBytes` to write binary data (images, media) into ZIP archives — needed for DOCX/PPTX with embedded images
**Estimated:** 80 LOC implementation + 60 LOC tests = ~140 LOC
**Duration:** 1.5 hours

**Tasks:**
- Register new builtin `_zip_writeEntryBytes` or extend `_zip_createArchive` to accept bytes content
  - Option A: New standalone builtin `_zip_writeEntryBytes(path: string, entryName: string, data: string) -> Result[(), string] ! {FS}` where `data` is base64-encoded (mirrors `_zip_readEntryBytes` pattern)
  - Option B: Modify `_zip_createArchive` to accept `{name: string, content: string, encoding: string}` where `encoding` = "base64" triggers binary decode
  - **Recommended: Option A** — simpler, consistent with existing `readEntryBytes`
- Export in `std/zip.ail` (or wherever zip builtins are exported)
- Add Go unit tests: write base64-encoded PNG data → read back → verify round-trip
- Create `examples/zip_binary.ail` demonstrating binary entry writing

**Acceptance Criteria:**
- [ ] `writeEntryBytes` builtin registered and callable
- [ ] Round-trip test: write base64 bytes → read back with `readEntryBytes` → matches
- [ ] Path traversal rejected (e.g. `../etc/passwd`)
- [ ] Size limit enforced
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- Need to decide append-to-existing-zip vs create-new — Go's `archive/zip` doesn't natively support appending. Recommend creating new archives with both text and binary entries.

### Milestone 4: M4_DOCS_CHANGELOG — Documentation & Cleanup
**Goal:** Update docs, design doc status, CHANGELOG
**Estimated:** ~30 LOC
**Duration:** 15 minutes

**Tasks:**
- Update CHANGELOG.md with all new builtins
- Update design doc status (Phase 1: ✅, Phase 2: ✅, Phase 3: ✅)
- Update `std/xml.ail` header comment to list new exports
- Move design doc to `implemented/v0_9_4/` if all phases complete

**Acceptance Criteria:**
- [ ] CHANGELOG.md updated
- [ ] Design doc status reflects implementation
- [ ] `std/xml.ail` module comment lists all exports

## Success Metrics
- Test coverage: All new builtins have unit tests
- Examples passing: `xml_escape.ail` + `xml_build.ail` + `zip_binary.ail` verified
- Documentation: CHANGELOG + design doc updated
- All tests passing: ✅
- All linting passing: ✅

## Notes
- Go helper functions `makeXmlElement`/`makeXmlText`/`makeXmlComment` already exist in `internal/builtins/xml.go` — the builtins just wrap them
- `html.EscapeString()` is used throughout — consistent with existing serialize behavior
- Phase 1 & 2: Pure functions (no effects) — simplest registration path
- Phase 3: `writeEntryBytes` needs `FS` effect — same as `_zip_createArchive`
- `BytesValue` exists in `internal/eval/value.go:47` with `[]byte`, `Filename`, `MimeType` fields
- `_zip_readEntryBytes` returns base64-encoded strings — `writeEntryBytes` should accept the same format for symmetry

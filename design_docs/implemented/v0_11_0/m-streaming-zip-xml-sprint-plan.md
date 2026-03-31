# Sprint Plan: M-STREAMING-ZIP-XML

## Summary
Add streaming ZIP+XML fold builtins so XLSX parsing can process shared strings without materializing the entire decompressed XML in memory. Solves the 8.7 MB XLSX OOM.

**Duration:** 2 days
**Dependencies:** None (std/array `empty`/`append` from M-STD-MAP nice-to-have but not blocking — fold can return lists)
**Risk Level:** Low-Medium

## Current Status Analysis

### Completed Recently
- Fix multipart BytesValue crash: ~200 LOC in 1 day
- Add cons (::) expression: ~350 LOC in 1 day
- Add fileData/fileUri Gemini support: ~150 LOC in 1 day

### Velocity
- Recent average: ~300-400 LOC/day (implementation + tests)
- Estimated capacity: ~600-800 LOC for this 2-day sprint

### Infrastructure Already in Place
- `scanForElements()` in xml.go:268 — token-streaming XML scanner (fold variant is ~30 line diff)
- `ctx.FnCallerN()` — callback infrastructure used by `_list_foldl`, `_str_foldChars`
- `_list_foldl` in list_iterative.go:202 — exact pattern to follow for fold type signature
- `zip.File.Open()` returns `io.ReadCloser` — streaming already at Go level

## Proposed Milestones

### M1_XML_PARSE_FOLD: XML parseFold builtin (pure)
**Goal:** Fold variant of `parseElements` — calls handler per matching element instead of collecting into list
**Estimated:** ~120 LOC implementation + ~80 LOC tests = ~200 LOC
**Duration:** 0.5 day

**Tasks:**
1. Add `scanForElementsFold()` helper to `xml.go` (adapt `scanForElements`)
2. Register `_xml_parseFold` builtin — type: `(string, string, a, (a, XmlNode) -> a) -> Result[a, string]`
3. Add `parseFold` to `std/xml.ail`
4. Write Go unit tests for fold (empty XML, multiple matches, handler error propagation)
5. Create `examples/runnable/xml_fold.ail`

**Acceptance Criteria:**
- `_xml_parseFold` registered and callable from AILANG
- Fold produces same results as `parseElements` + manual fold
- Handler errors propagate as `Err(...)` Result
- `make test` passes
- `make verify-examples` passes

**Risks:**
- Type inference for polymorphic accumulator `a` — Mitigation: follow `_list_foldl` type pattern exactly

### M2_ZIP_XML_SCAN_FOLD: Combined ZIP+XML streaming fold
**Goal:** Pipe `zip.Open()` stream directly into `xml.NewDecoder()` with fold callback. Zero-copy — decompressed XML never materializes as a Go string.
**Estimated:** ~150 LOC implementation + ~100 LOC tests = ~250 LOC
**Duration:** 0.5 day

**Tasks:**
1. Create `internal/builtins/zip_xml.go` (or add to zip.go)
2. Register `_zip_xml_scanFold` builtin — type: `(string, string, string, a, (a, XmlNode) -> a) -> Result[a, string] ! {FS}`
3. Reuse `scanForElementsFold` from M1 with `io.ReadCloser` input
4. Add `scanFold` to `std/zip.ail` (imports XmlNode from std/xml)
5. Write Go unit tests with small test ZIP containing XML
6. Create `examples/runnable/zip_xml_fold.ail`

**Acceptance Criteria:**
- `_zip_xml_scanFold` registered and callable from AILANG
- Opens ZIP entry as stream, never calls `io.ReadAll()`
- XML tokens parsed from stream, fold callback invoked per matching element
- Proper cleanup: ZIP archive + entry reader closed on all code paths
- Handler errors propagate cleanly as `Err(...)` Result
- `make test` passes
- `make verify-examples` passes

**Risks:**
- Cross-module type dependency (std/zip importing XmlNode from std/xml) — Mitigation: XmlNode is in the type universe, no circular import
- Resource cleanup on handler panic — Mitigation: defer Close() on all readers

### M3_INTEGRATION_BENCHMARK: End-to-end test + memory validation
**Goal:** Prove the OOM fix works by testing with realistic XML data
**Estimated:** ~80 LOC tests + ~70 LOC example = ~150 LOC
**Duration:** 0.5 day

**Tasks:**
1. Create Go benchmark test: `parseFold` vs `parseElements` on 10 MB XML string
2. Create Go benchmark test: `zipXmlScanFold` vs `readEntry+parseElements` on ZIP with large XML
3. Verify memory profile: fold variant doesn't allocate O(document) memory
4. Update design doc status to implemented
5. Update CHANGELOG.md

**Acceptance Criteria:**
- Benchmark shows constant memory usage for fold (not proportional to document size)
- `scanFold` on ZIP with 10 MB XML entry completes without OOM
- `make test` and `make lint` clean
- Design doc moved to `implemented/`
- CHANGELOG updated

**Risks:**
- Generating large test fixtures — Mitigation: generate XML programmatically in test

### M4_DOCS_CLEANUP: Documentation and agent notification
**Goal:** Notify ailang-parse agent, update docs
**Estimated:** ~50 LOC
**Duration:** 0.25 day

**Tasks:**
1. Send completion message to ailang-parse agent
2. Acknowledge original agent messages
3. Update std/xml.ail and std/zip.ail header comments
4. Verify `make ci` passes

**Acceptance Criteria:**
- Agent message acknowledged
- Completion notification sent
- All CI checks pass

## Success Metrics
- Test coverage: xml_fold and zip_xml tests with >80% coverage of new code
- Examples passing: `xml_fold.ail` and `zip_xml_fold.ail` verified
- Documentation: CHANGELOG.md, std/xml.ail, std/zip.ail updated
- All tests passing
- All linting clean
- Memory: fold variant uses O(element) not O(document)

## Dependencies
- None blocking. M-STD-MAP array gaps (`empty`/`append`) are nice-to-have for fold accumulators but not required — can fold into lists.

## Notes
- Total estimated: ~650 LOC across 2 days
- This sprint is self-contained — no type system changes, no new value types
- Pattern follows established `_list_foldl` callback infrastructure exactly
- The combined `_zip_xml_scanFold` is the high-value builtin; `_xml_parseFold` is a useful building block

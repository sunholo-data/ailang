# Sprint Plan: M-DOCPARSE-DX Stdlib & DX Improvements

## Summary

Close the DX trust gap identified by DocParse by fixing a curried lambda type-check bug, adding set operations to std/list, adding listDir to std/fs, adding string splitting functions, and adding XML serialization + ZIP write capabilities.

**Duration:** 4 days (focused — defer XLSX perf investigation to follow-up)
**Dependencies:** None — all items are independent
**Risk Level:** Medium — M1 (curried lambda fix) touches typechecker
**Design Doc:** [m-docparse-dx.md](m-docparse-dx.md)

## Current Status Analysis

### Completed Recently
- M-BRAIN: 4 milestones (~5000 LOC) in ~5 days
- M-PERF5: 3 milestones (~350 LOC) in 2 days — bulk XML ops, string join, dtree investigation
- M-CLOUD-PROGRESS-TRACKING: Cloud execution visibility in 1 day

### Velocity
- Recent average: ~200-300 LOC/day for builtin work
- Estimated capacity: ~800-1200 LOC for 4 days
- Sprint estimate: ~670 LOC total (within capacity)

### Remaining from Design Doc
- ⏳ M1: Curried lambda type-check fix (~30 LOC + 100 LOC tests)
- ⏳ M2: std/list set operations (~140 LOC Go + AILANG)
- ⏳ M3: std/fs listDir (~35 LOC)
- ⏳ M4: String splitting words/splitAny (~30 LOC)
- ⏳ M5: std/xml serialization (~95 LOC)
- ⏳ M6: std/zip write (~115 LOC)
- 📋 M7: XLSX merged-cell perf (deferred — needs profiling, may be DocParse-side)

## Proposed Milestones

### Milestone 1: Curried Lambda Type-Check Fix
**Goal:** Fix type unification so `\a. \b. expr` unifies with `(a, b) -> c`, allowing curried lambdas as HOF arguments.

**Estimated:** 30 LOC implementation + 100 LOC tests = 130 LOC
**Duration:** 1 day (typechecker work requires careful investigation)

**Tasks:**
1. Reproduce: confirm `foldl(\acc. \cell. acc + cell, 0, [1,2,3])` fails
2. Trace through `internal/types/typechecker.go` unification to find where arity mismatch fails
3. Add uncurrying logic: when unifying `TFunc2{Params:[a], Return:TFunc2{Params:[b], Return:c}}` with `TFunc2{Params:[a,b], Return:c}`, flatten the curried form
4. Handle nested currying: `\a. \b. \c. expr` should unify with 3-arity function
5. Regression tests:
   - Curried lambda in foldl, map, filter
   - Multi-param lambda still works
   - Mixed: some params curried, some not
   - Nested currying (3+ levels)
6. Run `make test` and `make eval-all` to verify no regressions

**Acceptance Criteria:**
- [ ] `foldl(\acc. \cell. acc + cell, 0, [1,2,3])` type-checks and returns `6`
- [ ] `map(\x. \y. x + y)` type-checks (partial application)
- [ ] Existing multi-param lambdas (`\x y. x + y`) unchanged
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Full eval suite shows no regressions

**Risks:**
- Uncurrying logic may interfere with partial application semantics — Mitigation: only uncurry during unification with a known multi-param target, not globally
- May need changes in `internal/elaborate/` instead of `internal/types/` — Mitigation: investigate both paths before coding

**Pause Point:** After M1 completes, run full eval suite. If regressions appear, stop and fix before proceeding.

### Milestone 2: std/list Set Operations
**Goal:** Add `member`, `dedup`, `intersect`, `union`, `difference` to std/list using Go builtins with `reflect.DeepEqual`.

**Estimated:** 120 LOC Go + 20 LOC AILANG + 150 LOC tests = 290 LOC
**Duration:** 0.5 day

**Tasks:**
1. Add Go builtins in `internal/builtins/list.go`:
   - `_list_member(elem, list)` → bool
   - `_list_dedup(list)` → list (preserve first occurrence)
   - `_list_intersect(a, b)` → list
   - `_list_union(a, b)` → list
   - `_list_difference(a, b)` → list
2. Register all 5 with `RegisterEffectBuiltin` (IsPure: true)
3. Add type functions using `types.Builder` — all are `a -> [a] -> ...` or `[a] -> [a] -> [a]`
4. Add AILANG wrappers in `std/list.ail`:
   ```ailang
   export pure func member(elem: a, xs: [a]) -> bool = _list_member(elem, xs)
   export pure func dedup(xs: [a]) -> [a] = _list_dedup(xs)
   export pure func intersect(xs: [a], ys: [a]) -> [a] = _list_intersect(xs, ys)
   export pure func union(xs: [a], ys: [a]) -> [a] = _list_union(xs, ys)
   export pure func difference(xs: [a], ys: [a]) -> [a] = _list_difference(xs, ys)
   ```
5. Tests: empty lists, single element, duplicates, no overlap, full overlap, strings, ints, records
6. Create `examples/set_operations.ail`

**Acceptance Criteria:**
- [ ] `member(3, [1,2,3])` returns `true`
- [ ] `dedup([1,2,1,3,2])` returns `[1,2,3]`
- [ ] `intersect([1,2,3], [2,3,4])` returns `[2,3]`
- [ ] `union([1,2], [2,3])` returns `[1,2,3]`
- [ ] `difference([1,2,3], [2])` returns `[1,3]`
- [ ] `examples/set_operations.ail` runs successfully
- [ ] `make test` and `make lint` pass

**Risks:**
- `reflect.DeepEqual` on records/ADTs may have edge cases — Mitigation: test with records and ADT values explicitly

### Milestone 3: std/fs listDir + String Splitting
**Goal:** Add directory listing and whitespace-aware string splitting.

**Estimated:** 35 LOC listDir + 30 LOC strings + 10 LOC AILANG + 80 LOC tests = 155 LOC
**Duration:** 0.5 day

**Tasks:**
1. Add `_fs_listDir(path)` in `internal/builtins/fs.go`:
   - Use `os.ReadDir()`, return sorted list of entry names
   - Respect sandbox restrictions (if any)
   - Return error for nonexistent directories
2. Add wrapper in `std/fs.ail`:
   ```ailang
   export func listDir(path: string) -> [string] ! {FS} = _fs_listDir(path)
   ```
3. Add `_str_words(s)` in `internal/builtins/string.go`:
   - Use `strings.Fields()` — splits on any whitespace, drops empties
4. Add `_str_splitAny(s, delimiters)` in `internal/builtins/string.go`:
   - Use `strings.FieldsFunc()` with custom delimiter check
5. Register builtins with proper types
6. Tests: listDir on `examples/`, empty dir, nonexistent dir
7. Tests: words with tabs, newlines, multiple spaces, empty string
8. Tests: splitAny with multiple delimiters
9. Create `examples/directory_listing.ail`

**Acceptance Criteria:**
- [ ] `listDir("examples/")` returns sorted list of filenames
- [ ] `listDir` returns error for nonexistent directory
- [ ] `words("hello  world\tfoo")` returns `["hello", "world", "foo"]`
- [ ] `splitAny("a,b;c", [",", ";"])` returns `["a", "b", "c"]`
- [ ] `examples/directory_listing.ail` runs successfully
- [ ] `make test` and `make lint` pass

**Risks:**
- Sandbox may restrict `os.ReadDir` — Mitigation: check how existing FS builtins handle sandbox, follow same pattern

### Milestone 4: XML Serialization + ZIP Write + Docs
**Goal:** Add XML serialization (tree→string) and ZIP archive write capabilities. Update CHANGELOG and examples.

**Estimated:** 80 LOC XML + 100 LOC ZIP + 30 LOC AILANG + 120 LOC tests + docs = 330 LOC
**Duration:** 1 day

**Tasks:**
1. Add `_xml_serialize(node)` in `internal/builtins/xml.go`:
   - Walk XmlNode tree, emit XML using `strings.Builder`
   - Handle Element, Text, Comment node types
   - Properly escape attribute values and text content
2. Add `_xml_serializeWithDecl(node)` — same but prepends `<?xml version="1.0" encoding="UTF-8"?>`
3. Verify `element()` and `textNode()` constructors exist (XmlNode ADT variants) — if not, add Go builtins
4. Add AILANG wrappers in `std/xml.ail`:
   ```ailang
   export pure func serialize(node: XmlNode) -> string = _xml_serialize(node)
   export pure func serializeWithDecl(node: XmlNode) -> string = _xml_serializeWithDecl(node)
   ```
5. Add `_zip_createArchive(path)` in `internal/builtins/zip.go`:
   - Create file + `zip.NewWriter`, return opaque handle
6. Add `_zip_writeEntry(writer, name, content)` — string content
7. Add `_zip_writeEntryBytes(writer, name, base64data)` — binary content
8. Add `_zip_closeArchive(writer)` — flush and close
9. Add AILANG wrappers in `std/zip.ail` with `! {FS}` effects
10. Tests: XML roundtrip (parse → serialize → parse → compare)
11. Tests: ZIP roundtrip (create → write entries → read back → compare)
12. Update CHANGELOG.md with all M-DOCPARSE-DX changes
13. Send completion message to docparse inbox

**Acceptance Criteria:**
- [ ] `serialize(element("root", [], [textNode("hello")]))` returns `<root>hello</root>`
- [ ] `serializeWithDecl(node)` includes XML declaration
- [ ] XML roundtrip: `parse(serialize(tree))` produces equivalent tree
- [ ] ZIP roundtrip: create archive, write entries, read back, entries match
- [ ] `closeArchive` properly flushes and closes ZIP
- [ ] CHANGELOG.md updated
- [ ] `make test` and `make lint` pass
- [ ] `make verify-examples` passes

**Risks:**
- ZipWriter handle lifecycle — Mitigation: document that `closeArchive` must be called; consider adding `withArchive` bracket helper in future
- XML namespace serialization complexity — Mitigation: start with simple namespace-free serialization, add namespace support as follow-up if needed

## Success Metrics
- All 4 milestones complete
- 11 new stdlib functions across 4 modules
- Test coverage maintained (no regression)
- All examples passing: `make verify-examples`
- Full eval suite: no regressions from M1 typechecker change
- CHANGELOG.md updated
- DocParse inbox notified of changes

## Dependencies
- None — all milestones are independent and can be reordered
- M1 is highest priority (bug fix) and should go first
- M4 is lowest risk and can be dropped if time pressure

## Open Questions
- Should `listDir` return just names or include file metadata (size, isDir)? Design doc says names only — confirm with user if needed.
- Should `member` return `Option[a]` (found element) or `bool`? Design doc says `bool` — simpler and sufficient for set operations.

## Notes
- M7 (XLSX merged-cell performance) deliberately deferred — needs profiling first, may be DocParse-side issue
- Follow M-PERF5 pattern for builtin registration: `RegisterEffectBuiltin` + `types.Builder`
- Reference `_str_join` implementation (M-PERF5 M2) as template for new string builtins
- Reference `_xml_findAllTexts` (M-PERF5 M1) as template for new XML builtins

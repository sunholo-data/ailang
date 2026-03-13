# M-DOCPARSE-DX: Stdlib & DX Improvements from DocParse Feedback

**Status**: Planned
**Target**: v0.9.3
**Priority**: P1 (High — unblocks real-world project adoption)
**Estimated**: 5 days (20h implementation + 8h testing + 2h docs)
**Dependencies**: None (all items are independent stdlib/parser/typechecker work)
**Milestone ID**: M-DOCPARSE-DX
**Created**: 2026-03-13
**Source**: DocParse project agent messages (6 messages, 2026-03-12/13)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | M1 fix: curried lambdas produce same result as uncurried (removes surprising failure). All new stdlib functions are pure or deterministic with same inputs |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | +1 | listDir and ZIP write explicitly require `! {FS}`; XML serialize is pure (no effect needed). No hidden effects |
| A4: Explicit Authority | +1 | All I/O functions reuse existing FS capability + sandbox; no ambient access |
| A5: Bounded Verification | +1 | M1 fix enables local type-checking of curried lambdas that currently fails |
| A6: Safe Concurrency | 0 | No concurrency impact — all functions are stateless per-call (except ZipWriter handle) |
| A7: Machines First | +1 | Reduces token cost: `dedup(xs)` vs 15-line recursive implementation; AI agents write fewer lines |
| A8: Minimal Syntax | 0 | No new syntax — all additions are builtins/stdlib functions |
| A9: Cost Visibility | 0 | No new cost model |
| A10: Composability | +1 | XML serialize + ZIP write compose into document generation pipeline; set operations compose with existing list functions |
| A11: Structured Failure | 0 | Uses existing error patterns (Option for member, direct return for others) |
| A12: System Boundary | 0 | FS/IO boundaries already explicit |

**Net Score: +6** → **Decision: Accept**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — all functions are deterministic
- [x] A3 (Effects): All I/O operations explicitly require FS/IO capabilities
- [x] A4 (Authority): No ambient access — reuses existing FS capability and sandbox
- [x] A7 (Machines First): Reduces token cost and agent turns for common operations

---

## Related Documents

- [m-stdlib-zip.md](../../implemented/v0_7_3/m-stdlib-zip.md) — Original ZIP read-only implementation (v0.7.3). M6 extends this with write capabilities
- [m-stdlib-gaps.md](../../implemented/v0_7_4/m-stdlib-gaps.md) — Prior stdlib gap analysis from eval results (v0.7.4). Same pattern: agent feedback → stdlib additions
- [m-dx11-string-split-builtin.md](../../implemented/v0_4_6/m-dx11-string-split-builtin.md) — Original string split implementation. M4 extends with whitespace-aware splitting
- [m-gap6-stdlib-maximum.md](../../implemented/v0_7_0/m-gap6-stdlib-maximum.md) — std/list maximum/minimum additions. M2 follows same pattern for set operations
- [m-dx-match-in-hof-block-lambda.md](../v0_10_0/m-dx-match-in-hof-block-lambda.md) — Related parser issue with lambdas in HOF args. M1 is a typechecker cousin of this parser bug

---

## Systemic Audit

**Question: Is this a one-off or part of a pattern?**

Yes — this is the **third round** of stdlib gap closure driven by real-world agent feedback:
1. **v0.6.5** (M-STDLIB-GAPS): Eval harness agents needed `nth`, `last`, `join` → added to std/list
2. **v0.7.3** (M-STDLIB-ZIP): DocParse agent needed ZIP read → added std/zip
3. **v0.9.3** (M-DOCPARSE-DX): DocParse agent needs set ops, listDir, XML/ZIP write, string splitting

**Pattern**: Each round follows the same shape — agents reimplement common operations because stdlib is missing them. The systemic fix would be a stdlib request pipeline where agent feedback auto-generates design docs. For now, we address the concrete gaps.

**Lambda/HOF issues** are also a pattern (M1 here + M-DX-MATCH-HOF). Both involve lambdas misbehaving when passed as HOF arguments. The root causes differ (typechecker vs parser) but should be investigated together.

---

## Problem Statement

DocParse — a real-world document parsing project built on AILANG — has identified concrete DX gaps that make AILANG harder to use than Python for equivalent tasks. Their eval.ail module (525 lines) mirrors eval_office.py (386 lines), but took 4x longer to write due to missing stdlib functions, a typechecker bug with curried lambdas, and missing I/O capabilities.

These are not theoretical concerns — they come from a production project that is actively choosing Python over AILANG for CI because of these gaps.

**Impact:**
- DocParse eval runs 106 ailang invocations instead of 1 (no listDir)
- Set operations require 40+ lines of O(n²) list code instead of 4 lines
- Curried lambdas in HOF args force extracting trivial closures to named functions
- XML/ZIP write operations impossible — blocks document generation use case
- XLSX parser hangs on merged-cells-heavy files

---

## Goals

**Primary Goal:** Close the DX trust gap so DocParse eval.ail replaces Python eval as the primary CI check.

**Success Metrics:**
- DocParse eval.ail line count: 525 → ~400 (−24%) by removing manual set operation implementations
- Process invocations per eval run: 106 → 1 (using listDir for single-process execution)
- Curried lambda workarounds: eliminate need to extract trivial lambdas to named functions
- New stdlib functions: +11 functions across 4 modules (list, fs, xml, zip)
- Zero regressions in existing test suite and eval benchmarks

---

## Verified Issues

All issues verified against current codebase (v0.9.1.1 / dev):

### M1: Curried Lambda in HOF Type-Check Failure (P0 — Bug)

**Problem**: `\acc. \cell. acc || f(cell)` passed to `foldl` fails with:
```
type unification failed: function arity mismatch: 2 vs 1
```

**Reproduction**:
```ailang
import std/list (foldl)
let isMerged = \cell. cell > 5
-- ❌ FAILS: curried lambda as HOF argument
let hasMerged = \cells. foldl(\acc. \cell. acc || isMerged(cell), false, cells)
-- ✅ WORKS: multi-param lambda
let hasMerged2 = \cells. foldl(\acc cell. acc || isMerged(cell), false, cells)
```

**Root cause**: The typechecker treats `\acc. \cell. body` as a 1-arity function returning a function, but `foldl` expects a 2-arity function. These should unify (currying equivalence) but don't.

**Location**: Likely `internal/types/typechecker.go` or `internal/elaborate/` — the unification logic doesn't flatten curried function types.

**Fix approach**: During unification of function types, when arity mismatches, attempt to uncurry/flatten the curried form before failing. E.g., `(a -> (b -> c))` should unify with `((a, b) -> c)`.

**Related**: [m-dx-match-in-hof-block-lambda.md](../v0_10_0/m-dx-match-in-hof-block-lambda.md) (different symptom, same area — lambda bodies in HOF args)

### M2: std/list Set Operations (P1 — Feature)

**Problem**: No `dedup`, `intersect`, `union`, or `member` in std/list. Docparse wrote 40+ lines of O(n²) recursive implementations for Jaccard similarity.

**Deliverables**:
```ailang
-- std/list additions
member : a -> [a] -> bool          -- O(n) linear membership test
dedup : [a] -> [a]                 -- Remove duplicates, preserve first occurrence
intersect : [a] -> [a] -> [a]     -- Elements in both lists
union : [a] -> [a] -> [a]         -- Elements in either list, no duplicates
difference : [a] -> [a] -> [a]    -- Elements in first but not second
```

**Implementation**: Go builtins using `reflect.DeepEqual` for equality comparison (consistent with existing list equality semantics — no `Eq` typeclass needed). Pure AILANG wrappers in `std/list.ail` calling `_list_member`, `_list_dedup`, etc.

**Location**: `internal/builtins/list.go` (new functions), `std/list.ail` (wrappers)

### M3: std/fs.listDir (P1 — Feature)

**Problem**: Cannot enumerate directory contents. DocParse eval must spawn 106 separate ailang processes (2 per file × 53 files) via bash wrapper because it can't iterate golden files.

**Deliverables**:
```ailang
-- std/fs addition
listDir : string -> [string] ! {FS}   -- List entries in directory
```

**Implementation**: Go builtin `_fs_listDir` using `os.ReadDir()`. Returns sorted list of entry names (not full paths). User concatenates with base path.

**Location**: `internal/builtins/fs.go` (new function), `std/fs.ail` (wrapper)

### M4: Basic String Splitting Enhancements (P2 — Feature)

**Problem**: Only literal delimiter `split(text, " ")` available. Can't split on whitespace runs, punctuation, or character classes. Docparse tokenization limited to single-space split.

**Deliverables**:
```ailang
-- std/string additions (or string builtins)
words : string -> [string]                      -- Split on whitespace runs, drop empties (Haskell convention)
splitAny : string -> [string] -> [string]       -- Split on any of given delimiters
```

**Implementation**: Go builtins using `strings.Fields()` for `words`, `strings.FieldsFunc()` for `splitAny`. No regex engine needed — these cover 90% of tokenization needs. Note: `splitOnWhitespace` omitted as `words` provides identical semantics via `strings.Fields()`.

**Location**: `internal/builtins/string.go` (new functions)

**Note**: Full regex is deferred to v0.10.0+ — adding a regex engine is significant scope. These string functions are the pragmatic 80/20 solution.

### M5: std/xml Serialization (P2 — Feature)

**Problem**: Can parse and query XML, but cannot serialize XmlNode trees back to strings. Blocks document generation (DOCX/PPTX/XLSX are XML-in-ZIP).

**Deliverables**:
```ailang
-- std/xml additions
element : string -> [(string, string)] -> [XmlNode] -> XmlNode
textNode : string -> XmlNode
serialize : XmlNode -> string              -- Pure: in-memory tree → string
serializeWithDecl : XmlNode -> string      -- Includes <?xml version="1.0"?>
```

**Implementation**: Go builtin `_xml_serialize` using `encoding/xml.Encoder` writing to `strings.Builder` (no I/O — pure in-memory conversion). `element` and `textNode` are constructors (may already exist as ADT variants — verify). `serialize` walks the XmlNode tree and emits XML string. No effect annotation needed since this is a pure transformation.

**Location**: `internal/builtins/xml.go` (new functions), `std/xml.ail` (wrappers)

### M6: std/zip Write Capabilities (P2 — Feature)

**Problem**: ZIP support is read-only (listEntries, readEntry, readEntryBytes). Cannot create archives. Blocks Office document generation.

**Deliverables**:
```ailang
-- std/zip additions
createArchive : string -> ZipWriter ! {FS}
writeEntry : ZipWriter -> string -> string -> () ! {FS}
writeEntryBytes : ZipWriter -> string -> string -> () ! {FS}  -- base64 content
closeArchive : ZipWriter -> () ! {FS}
```

**Implementation**: Go builtins using `archive/zip.Writer`. `ZipWriter` is an opaque handle (similar to existing file handles). Requires resource cleanup semantics — `closeArchive` must be called.

**Location**: `internal/builtins/zip.go` (new functions), `std/zip.ail` (wrappers)

**Risk**: Resource handle lifecycle — if user forgets `closeArchive`, archive is corrupted. Consider a `withArchive` bracket pattern.

### M7: XLSX Merged-Cell Performance (P2 — Bug/Performance)

**Problem**: `poi_many_merges.xlsx` (829KB, hundreds of merged cells) causes XLSX parser to hang (>60s timeout). Likely O(n²) in merged cell resolution or repeated std/xml.findAll on large trees.

**Investigation needed**: This may be a DocParse-side algorithm issue rather than an AILANG stdlib issue. Need profiling to determine whether the bottleneck is in AILANG interpreter overhead (known issue per XML performance message) or in the parser algorithm itself.

**Deliverables**:
- Profile the hang to identify hot path
- If AILANG-side: add bulk XML query operations or optimize interpreter loop
- If DocParse-side: provide guidance on algorithm optimization

**Location**: TBD after profiling

---

## Out of Scope

These items from the docparse messages are acknowledged but deferred:

| Item | Reason | Target |
|------|--------|--------|
| Full regex engine | Large scope, string functions cover 90% | v0.10.0+ |
| Entry point >1 param | Known limitation, getArgs() workaround exists | v0.10.0 |
| Process startup caching | Requires incremental compilation infrastructure | v0.10.0+ |
| Set type (first-class) | Requires type system changes; list functions sufficient | v1.0.0 |
| Module scoping collision | Partially fixed (M-MODULE-SCOPE, M-DX18) | Ongoing |
| Compiled mode | Architectural change, 100x perf but massive scope | v1.0.0+ |

---

## Implementation Plan

### Day 1: M1 — Curried Lambda Fix
- Diagnose exact location in typechecker where curried/uncurried unification fails
- Add uncurrying logic to function type unification
- Test: `foldl(\acc. \cell. expr, init, list)` type-checks and runs correctly
- Test: existing multi-param lambdas continue to work
- Regression tests in `internal/types/` or `internal/elaborate/`

### Day 2: M2 — std/list Set Operations
- Implement `_list_member`, `_list_dedup`, `_list_intersect`, `_list_union`, `_list_difference` in Go
- Add wrappers to `std/list.ail`
- Tests: unit tests for each function, including edge cases (empty lists, duplicates)
- Example: `examples/set_operations.ail`

### Day 3: M3 + M4 — listDir + String Splitting
- Implement `_fs_listDir` in Go, wrapper in `std/fs.ail`
- Implement `_str_words`, `_str_splitAny` in Go
- Tests for both
- Example: `examples/directory_listing.ail`

### Day 4: M5 + M6 — XML Serialize + ZIP Write
- Implement `_xml_serialize`, `_xml_serializeWithDecl` in Go
- Implement `_zip_createArchive`, `_zip_writeEntry`, `_zip_writeEntryBytes`, `_zip_closeArchive` in Go
- Add wrappers to `std/xml.ail` and `std/zip.ail`
- Tests: roundtrip (parse → modify → serialize → parse again)
- Tests: create archive, add entries, read back

### Day 5: M7 Investigation + Integration Testing + Docs
- Profile XLSX merged-cell hang
- Full integration test: run docparse eval.ail equivalent using new stdlib
- Update CHANGELOG.md
- Update examples/
- Send response to docparse inbox with release notes

---

## Files to Modify/Create

**Modified files:**
- `internal/types/typechecker.go` (~+30 LOC) — Curried function type unification
- `internal/builtins/list.go` (~+120 LOC) — Set operation builtins
- `internal/builtins/fs.go` (~+30 LOC) — listDir builtin
- `internal/builtins/string.go` (~+30 LOC) — words, splitAny builtins
- `internal/builtins/xml.go` (~+80 LOC) — serialize, serializeWithDecl builtins
- `internal/builtins/zip.go` (~+100 LOC) — createArchive, writeEntry, closeArchive builtins
- `std/list.ail` (~+20 LOC) — Wrappers for set operations
- `std/fs.ail` (~+5 LOC) — Wrapper for listDir
- `std/xml.ail` (~+15 LOC) — Wrappers for serialize functions
- `std/zip.ail` (~+15 LOC) — Wrappers for write functions

**New files:**
- `internal/types/curry_unify_test.go` (~100 LOC) — Curried lambda unification tests
- `internal/builtins/list_set_test.go` (~150 LOC) — Set operation tests
- `internal/builtins/fs_listdir_test.go` (~60 LOC) — listDir tests
- `internal/builtins/xml_serialize_test.go` (~100 LOC) — XML serialization tests
- `internal/builtins/zip_write_test.go` (~100 LOC) — ZIP write tests
- `examples/set_operations.ail` (~30 LOC) — Set operations example
- `examples/directory_listing.ail` (~20 LOC) — listDir example

---

## Testing Strategy

**Unit tests:**
- Curried lambda unification (all arity combinations, nested currying, edge cases)
- Each set operation (empty lists, duplicates, mixed types, large lists)
- listDir (valid dir, empty dir, nonexistent dir → error)
- words/splitAny (tabs, newlines, multiple spaces, empty input, custom delimiters)
- XML serialize (elements, attributes, nested trees, text nodes, namespaces)
- ZIP write (single entry, multiple entries, binary content, roundtrip)

**Integration tests:**
- End-to-end: parse XML → modify → serialize → parse again → verify
- End-to-end: create ZIP → write entries → read back → verify
- DocParse eval.ail runs as single process using listDir
- `make verify-examples` passes with new example files

**Regression tests:**
- Full eval suite (`make eval-all`) before and after M1 curried lambda fix
- `make test` passes after each milestone
- Existing std/list, std/fs, std/xml, std/zip tests still pass

---

## Success Criteria

- [ ] `foldl(\acc. \cell. acc + cell, 0, [1,2,3])` type-checks and returns 6
- [ ] `dedup([1,2,1,3,2])` returns `[1,2,3]`
- [ ] `intersect([1,2,3], [2,3,4])` returns `[2,3]`
- [ ] `listDir("examples/")` returns list of `.ail` files
- [ ] `words("hello  world\tfoo")` returns `["hello", "world", "foo"]`
- [ ] `serialize(element("root", [], [textNode("hello")]))` returns `<root>hello</root>`
- [ ] ZIP roundtrip: create archive → write entries → read back → entries match
- [ ] DocParse eval.ail can run as single process (using listDir instead of bash wrapper)
- [ ] All tests passing (`make test`)
- [ ] Examples passing (`make verify-examples`)
- [ ] Full eval suite passes (`make eval-all`) — no regressions
- [ ] CHANGELOG.md updated
- [ ] Documentation updated (std/list, std/fs, std/xml, std/zip)

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Curried lambda fix breaks existing type inference | Medium | High | Extensive test suite, run full eval suite before merge |
| ZipWriter handle leaks | Medium | Medium | Add `withArchive` bracket; document cleanup requirement |
| XML serialization namespace handling | Low | Medium | Start with simple elements, add namespace support later |
| XLSX hang is AILANG interpreter overhead | High | Low | Document as known limitation, recommend bulk operations |

---

## Messages Addressed

| Message ID | Title | Status |
|------------|-------|--------|
| msg_20260313_142716_c26791f9 | AILANG DX trust gap: eval.ail vs Python eval | Addressed (M1-M4) |
| msg_20260313_134017_6fc878e2 | XLSX parser hangs on many-merged-cells file | Addressed (M7) |
| msg_20260313_120219_c0a908c7 | RE: XML parse performance — Go-layer optimizations | Acknowledged (already applied) |
| msg_20260313_111526_d33b3b75 | FIXED: findAll nondeterminism | Acknowledged (already fixed) |
| msg_20260312_143522_2c981a35 | Feature request: std/xml serialization | Addressed (M5) |
| msg_20260312_143515_5a03395a | Feature request: std/zip write capabilities | Addressed (M6) |

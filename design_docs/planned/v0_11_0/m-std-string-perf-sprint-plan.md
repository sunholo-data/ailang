# Sprint Plan: M-STD-STRING-PERF

## Summary
Implement O(n) string processing primitives to unblock real-world document parsing. Lead with native `decodeQuotedPrintable` (guaranteed fix for the 37KB email repro), then add general-purpose primitives (`replaceMany`, `mapSlicesJoin`, ASCII fast-paths) that benefit all string-heavy workloads.

**Duration:** 3 days (5 milestones)
**Dependencies:** None
**Risk Level:** Low — all changes are additive Go builtins, no syntax or type system changes
**Design Doc:** [m-std-string-perf.md](m-std-string-perf.md)

## Current Status Analysis

### Completed Recently
- M-CODEGEN-STRATEGIC-REVIEW M1-M5: Statement IR architecture (3 days)
- v0.10.2 release (status codes, serve-api fixes)
- M-PERF7 (v0.9.3): `foldChars`, `charAt`, iterative `foldl` — already in production

### Velocity
- Recent average: ~200-400 LOC/day for builtin work (based on M-PERF5, M-PERF7 history)
- Estimated capacity: ~500 LOC for this sprint (3 days, focused builtin work)

### Remaining from Design Doc
- Track 1: `foldSlices` + `mapSlicesJoin` (~150 LOC)
- Track 2: `replaceMany` (~80 LOC)
- Track 3: `decodeQuotedPrintable` (~80 LOC)
- Track 4: ASCII fast-paths (~30 LOC)
- Track 5: `startsWithIgnoreCase` (~20 LOC)
- Tests + benchmarks (~200 LOC)
- Total: ~560 LOC

## Proposed Milestones

### Milestone 1: M1_NATIVE_QP_DECODE — Native Quoted-Printable Decoder
**Goal:** Implement `decodeQuotedPrintable` as a pure Go builtin. This is the proof-of-concept: if the 37KB email decodes in <1ms, it confirms the interpreter dispatch is the bottleneck and validates the entire sprint direction.

**Estimated:** 80 LOC implementation + 60 LOC tests = 140 LOC
**Duration:** 0.5 days

**Tasks:**
1. Create `internal/builtins/string_encoding.go` with `_str_decodeQP` implementation
2. Register builtin with `BuiltinSpec` (module `std/string`, pure, 1 arg)
3. Add type function: `string -> string`
4. Add to `std/string.ail`: `export pure func decodeQuotedPrintable(s: string) -> string = _str_decodeQP(s)`
5. Write Go unit tests: RFC 2045 test vectors (soft line breaks, `=XX` hex, invalid hex passthrough, `=` at end of string, `=\r\n` vs `=\n`)
6. Write Go benchmark: `BenchmarkDecodeQP_37KB` using content from the repro email
7. Validate: `make test && make lint`
8. Quick smoke test: write a 5-line `.ail` file that calls `decodeQuotedPrintable` on a QP string, run with `ailang run`

**Acceptance Criteria:**
- [ ] `decodeQuotedPrintable("hello=20world=0D=0A")` returns `"hello world\r\n"`
- [ ] Soft line break removal: `"line1=\r\nline2"` → `"line1line2"`
- [ ] Invalid hex passthrough: `"=GG"` → `"=GG"`
- [ ] `BenchmarkDecodeQP_37KB` completes in <1ms
- [ ] All existing tests pass (`make test`)
- [ ] Linting clean (`make lint`)

**Risks:**
- None — well-defined RFC, simple Go implementation, additive change

---

### Milestone 2: M2_REPLACE_MANY — Multi-Pattern String Replacement
**Goal:** Implement `replaceMany` using Go's `strings.NewReplacer` for single-pass multi-pattern replacement. Replaces 23 sequential `replace()` calls for HTML entity decoding.

**Estimated:** 80 LOC implementation + 50 LOC tests = 130 LOC
**Duration:** 0.5 days

**Tasks:**
1. Implement `_str_replaceMany` in `internal/builtins/string_ops.go`
2. Extract tuple pairs from AILANG list of tuples (handle `TupleValue` or `ListValue` with 2-element inner lists)
3. Register builtin: module `std/string`, pure, 2 args `(string, [(string, string)]) -> string`
4. Add to `std/string.ail`
5. Write Go unit tests: empty list, single pair, multiple pairs, overlapping patterns, Unicode, empty strings
6. Write Go benchmark: `BenchmarkReplaceMany_23Patterns_50KB`
7. Add example file: `examples/runnable/replace_many.ail`
8. Validate: `make test && make lint && make verify-examples`

**Acceptance Criteria:**
- [ ] `replaceMany("a&amp;b", [("&amp;", "&")])` returns `"a&b"`
- [ ] 23-pattern replacement on 50KB string is >10x faster than 23 sequential `replace` calls
- [ ] Empty replacement list returns input unchanged
- [ ] Example file runs successfully
- [ ] All tests pass, linting clean

**Risks:**
- Tuple extraction from AILANG values — need to handle how AILANG represents `(string, string)` tuples at the Value level. Check existing tuple handling patterns in builtins.

---

### Milestone 3: M3_MAP_SLICES_JOIN — Split-Transform-Join Without List Materialization
**Goal:** Implement `mapSlicesJoin` and `foldSlices` as Go builtins that iterate over split segments without creating an intermediate list, using `strings.Builder` to avoid O(n^2) accumulator concatenation.

**Estimated:** 120 LOC implementation + 60 LOC tests = 180 LOC
**Duration:** 0.5 days

**Tasks:**
1. Implement `_str_foldSlices(s, delim, acc, f)` in `internal/builtins/string_ops.go` — iterates segments via `strings.Index`, calls `FnCallerN` per segment
2. Implement `_str_mapSlicesJoin(s, delim, f)` — same iteration but uses `strings.Builder`, callback returns string
3. Register both builtins with proper polymorphic type signatures
4. Add to `std/string.ail`
5. Write Go unit tests: empty string, no delimiter found, single segment, delimiter at start/end, Unicode delimiters, empty segments between delimiters
6. Write Go benchmark: `BenchmarkMapSlicesJoin_3000Segs` vs `split+foldl`
7. Validate: `make test && make lint`

**Acceptance Criteria:**
- [ ] `mapSlicesJoin("a=b=c", "=", toUpper)` returns `"ABC"` (or appropriate transform)
- [ ] `foldSlices("a,b,c", ",", 0, \acc s -> acc + length(s))` returns `3`
- [ ] 3000-segment split-transform-join completes in <100ms (currently 20s+)
- [ ] Memory stays under 50MB for 3000 segments (currently 318MB+)
- [ ] All tests pass, linting clean

**Risks:**
- `FnCallerN` dispatch overhead may still be significant for 3000+ segments — benchmark will reveal. Even if slower than native QP decode, it should be much faster than `split+foldl` because it eliminates list allocation and O(n^2) accumulator.

---

### Milestone 4: M4_ASCII_FAST_PATH — ASCII Optimization for substring/find
**Goal:** Add ASCII fast-path to `_str_slice` and `_str_find` to skip `[]rune` conversion for ASCII-only strings (99% of email/HTML content).

**Estimated:** 30 LOC implementation + 40 LOC tests = 70 LOC
**Duration:** 0.25 days

**Tasks:**
1. Add `func isASCII(s string) bool` helper in `internal/builtins/string.go`
2. Add ASCII fast-path to `strSliceImpl`: if ASCII, use `str[start:end]` directly
3. Add ASCII fast-path to `strFindImpl`: if ASCII, skip `RuneCountInString` conversion
4. Write Go benchmarks: `BenchmarkStrSlice_ASCII_50KB`, `BenchmarkStrSlice_Unicode_50KB`
5. Verify all existing string tests still pass
6. Validate: `make test && make lint`

**Acceptance Criteria:**
- [ ] `BenchmarkStrSlice_ASCII_50KB` shows >5x improvement over rune conversion path
- [ ] Unicode strings still produce identical results (no behavior change)
- [ ] Mixed ASCII/Unicode strings detected correctly (falls back to rune path)
- [ ] All existing tests pass, linting clean

**Risks:**
- Low — `utf8.RuneCountInString(s) == len(s)` is a reliable ASCII check. Go strings are immutable so no TOCTOU risk.

---

### Milestone 5: M5_INTEGRATION — startsWithIgnoreCase + Integration Validation
**Goal:** Add `startsWithIgnoreCase` builtin, then validate the full sprint against the docparse 37KB email repro case. Update docs.

**Estimated:** 20 LOC implementation + 30 LOC tests + 20 LOC docs = 70 LOC
**Duration:** 0.25 days

**Tasks:**
1. Implement `_str_startsWithIC` using `strings.EqualFold` on prefix slice
2. Register builtin, add to `std/string.ail`
3. Write Go unit tests
4. Run docparse integration test with real 37KB email
5. Measure and record before/after: time, memory, for each track
6. Update `CHANGELOG.md` with all new builtins
7. Run full CI: `make test && make lint && make verify-examples`
8. Send results to docparse agent via `ailang messages send`

**Acceptance Criteria:**
- [ ] `startsWithIgnoreCase("Hello", "hel")` returns `true`
- [ ] 37KB QP email parses in <2s end-to-end (requires docparse to adopt `decodeQuotedPrintable`)
- [ ] All new builtins documented in CHANGELOG
- [ ] All tests pass, all examples verify, linting clean
- [ ] Results communicated back to docparse agent

**Risks:**
- End-to-end validation requires docparse to update their code to use new builtins. Sprint validates the AILANG side; docparse adoption is separate.

## Success Metrics
- 37KB QP decode: <1ms (native builtin)
- 23-pattern replace on 50KB: >10x faster than sequential
- 3000-segment mapSlicesJoin: <100ms, <50MB
- ASCII substring 50KB: >5x faster than rune path
- All existing tests passing: `make test`
- All examples verified: `make verify-examples`
- Linting clean: `make lint`
- CHANGELOG updated
- New example file added

## Dependencies
- None — all changes are additive Go builtins

## Open Questions
- Whether `decodeQuotedPrintable` should go in `std/string` or a new `std/encoding` module (deferred to implementer)
- Whether `toLowerASCII` should be a separate builtin (deferred — `startsWithIgnoreCase` covers the main use case)

## Notes
- Sprint ordered by confidence: M1 (100% guaranteed fix) → M2 (100%) → M3 (70% — callback dispatch may still be slow) → M4 (95%) → M5 (integration)
- If M1 alone fixes the docparse email, remaining milestones are still valuable for general string performance
- The test file is at `ailang-parse/data/test_files/challenge/challenge_real_html_qp.eml` (37KB, 2977 QP escapes)
- Total estimated LOC: ~560 across implementation, tests, benchmarks, and docs

# Sprint Plan: M-PERF5 Data-Intensive Workload Performance

## Summary

Optimize AILANG's data processing pipeline for large XML documents by adding bulk Go builtins and replacing O(n²) stdlib functions. Target: Moby Dick EPUB from 11.5s to <3s.

**Duration:** 2 days (focused — skip Env CoW and native list ops)
**Dependencies:** ddb52588 (String() fix, already merged)
**Risk Level:** Low — all changes are additive builtins, no interpreter changes
**Design Doc:** [m-perf5-data-intensive-workloads.md](m-perf5-data-intensive-workloads.md)

## Current Status Analysis

### Completed Recently
- ✅ String() → strings.Builder: 62s → 11.5s (5.4x) in ddb52588
- ✅ Pre-allocated slices in evalCoreList/Array/Tuple
- ✅ Zero-allocation whitespace check for CharData
- ✅ findAll nondeterminism fix (Go map iteration)

### Velocity
- Recent: ~150-200 LOC/day for builtin work (based on ddb52588: 111 LOC in 1 session)
- Estimated capacity: ~350 LOC for 2 days

### Remaining from Design Doc
- ⏳ Track 1: Bulk XML operations (~90 LOC Go + 10 LOC AILANG)
- ⏳ Track 3: String join builtin (~60 LOC Go + update std/string.ail)
- ⏳ Track 4: Decision tree investigation (~10 LOC change + investigation)
- 📋 Track 2: Native list ops (deferred — closure-calling architecture risk)
- 📋 Track 5: Env CoW (deferred — complex, M-PERF3 scope)

## Proposed Milestones

### Milestone 1: Bulk XML Operations
**Goal:** Add `findAllTexts` and `findAllAttrs` Go builtins that do N operations in 1 call, eliminating per-element interpreter round-trips.

**Estimated:** 90 LOC implementation + 60 LOC tests = 150 LOC
**Duration:** 0.5 day

**Tasks:**
1. Add `_xml_findAllTexts(node, tag)` in `internal/builtins/xml.go`
   - Combine `findAllRecursive` DFS with `collectText` string building
   - Returns `ListValue` of `StringValue`
2. Add `_xml_findAllAttrs(node, tag, attr)` in `internal/builtins/xml.go`
   - DFS for matching tag + extract named attribute
   - Returns `ListValue` of `StringValue`
3. Register both with `RegisterEffectBuiltin` (IsPure: true)
4. Add type functions using `types.Builder`
5. Add AILANG wrappers in `std/xml.ail`:
   ```ailang
   export pure func findAllTexts(node: XmlNode, tag: string) -> List[string] = _xml_findAllTexts(node, tag)
   export pure func findAllAttrs(node: XmlNode, tag: string, attr: string) -> List[string] = _xml_findAllAttrs(node, tag, attr)
   ```
6. Tests with `-count=20` for determinism (duplicate namespace XML)
7. Tests with realistic EPUB-like XML (20+ matching elements)

**Acceptance Criteria:**
- [ ] `findAllTexts(root, "p")` returns all text content of `<p>` elements in one call
- [ ] `findAllAttrs(root, "item", "href")` returns all href values of `<item>` elements
- [ ] Tests pass with `-count=20` (determinism verified)
- [ ] Realistic-complexity test: 20+ elements with namespaces
- [ ] `make test` and `make lint` pass

**Risks:**
- Type signature for generic return (`List[string]` vs `List[XmlNode]`) — Mitigation: return `List[string]` since the whole point is avoiding per-element getText/getAttr calls

### Milestone 2: String Join Builtin
**Goal:** Replace O(n²) recursive AILANG `join` with Go builtin using `strings.Join`. This is critical because DocParse builds strings from many XML elements.

**Estimated:** 60 LOC implementation + 40 LOC tests = 100 LOC
**Duration:** 0.5 day

**Key Discovery:** `std/string.ail` already has `join(delimiter, xs)` but it's:
```ailang
export pure func join(delimiter: string, xs: [string]) -> string {
  match xs {
    [] => "",
    [x] => x,
    [x, ...rest] => x ++ delimiter ++ join(delimiter, rest)
  }
}
```
This is O(n²) — each `++` copies the growing accumulator. For 1000 items of 100 chars, that's ~50MB of string copies.

**Tasks:**
1. Add `_str_join(parts, separator)` in `internal/builtins/string.go`
   - Uses `strings.Join` (single allocation)
   - Validate all elements are strings, return error if not
2. Register with `RegisterEffectBuiltin` (IsPure: true)
3. Add type function: `(List[string], string) -> string`
4. **Update** `std/string.ail` to use the Go builtin:
   ```ailang
   export pure func join(delimiter: string, xs: List[string]) -> string = _str_join(xs, delimiter)
   ```
   Note: argument order swap — Go builtin takes `(list, sep)` but AILANG API keeps `(delimiter, list)` for pipeline ergonomics
5. Tests: empty list, single element, many elements, empty separator
6. Determinism test with `-count=20`

**Acceptance Criteria:**
- [ ] `join(", ", ["a", "b", "c"])` returns `"a, b, c"`
- [ ] `join("", strings)` works for large lists (1000+ elements)
- [ ] Existing code using `join` continues to work (API compatible)
- [ ] Tests pass with `-count=20`
- [ ] `make test` and `make lint` pass

**Risks:**
- Argument order mismatch (AILANG: `join(sep, list)` vs Go convention `join(list, sep)`) — Mitigation: wrapper handles the swap

### Milestone 3: Decision Tree Pattern Matching Investigation
**Goal:** Investigate why dtree is disabled, enable if safe, benchmark impact.

**Estimated:** 10 LOC change + 50 LOC investigation/test = 60 LOC
**Duration:** 0.5 day

**Key Discovery:** `internal/dtree/` is fully implemented (380 lines) and tested. It's disabled by a single line in `eval_patterns.go`:
```go
useDecisionTree := false
```

**Tasks:**
1. Read dtree tests — verify they still pass
2. Read eval `evalDecisionTree` implementation (158 lines in `eval/decision_tree.go`)
3. Check for guard evaluation concerns (guards might need sequential ordering)
4. Enable via environment variable: `AILANG_DTREE=1`
5. Benchmark: run DocParse with/without dtree
6. If safe and beneficial: enable by default

**Acceptance Criteria:**
- [ ] Understand WHY dtree was disabled (document finding)
- [ ] If enabled: all existing tests pass
- [ ] If enabled: `make verify-examples` passes
- [ ] Performance delta measured and documented
- [ ] Flag-gated: `AILANG_DTREE=1` to enable (conservative)

**Risks:**
- Guard evaluation order semantics — Mitigation: test with guard-heavy match expressions, compare output with/without dtree
- May be disabled for a good reason — Mitigation: investigate before enabling, keep flag-gated

### Milestone 4: Benchmark and Verify Target
**Goal:** Re-run DocParse benchmarks with all optimizations, verify <3s target for Moby Dick.

**Estimated:** 0 LOC (testing only)
**Duration:** 0.5 day

**Tasks:**
1. `make quick-install` to rebuild with all changes
2. Run DocParse benchmark suite (sample.docx, Alice, Moby Dick)
3. Compare before/after: document speedup per file
4. Verify linear scaling (4x size ≈ 4x time)
5. Update design doc with final benchmarks
6. Update CHANGELOG.md

**Acceptance Criteria:**
- [ ] Moby Dick EPUB: <3s (from 11.5s)
- [ ] Alice EPUB: <2s (from 3.8s)
- [ ] Linear scaling verified
- [ ] All tests pass
- [ ] CHANGELOG updated
- [ ] Design doc updated with results

## Success Metrics
- Moby Dick EPUB: 11.5s → <3s (target 4x improvement)
- All tests passing with `-count=20` (determinism verified)
- `make test` and `make lint` clean
- CHANGELOG.md updated
- Design doc updated with final benchmarks

## Dependencies
- ddb52588 (String() fix) — already merged ✅
- DocParse EPUB test files for benchmarking

## Deferred to Future Sprint
- **Track 2: Native list map/filter/fold** — Requires evaluator access from builtins package, possible circular dependency. Needs architecture investigation first.
- **Track 5: Environment CoW** — Complex refactor, belongs in M-PERF3 scope. Less impactful for data-processing use case where bulk builtins bypass most function calls.

## Notes
- All new builtins are `IsPure: true` — determinism tests mandatory
- Bulk XML ops are the highest-impact change — they eliminate thousands of interpreter round-trips
- String join replacement is also high-impact — O(n²) → O(n) for string building
- Decision tree is exploratory — may or may not help for this specific workload
- Total estimated: ~310 LOC across 2 days (conservative for a low-risk sprint)

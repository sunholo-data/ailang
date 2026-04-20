# M-STD-STRING-PERF: O(n) String Processing Primitives

**Status**: Planned
**Target**: v0.11.0
**Priority**: P0 (blocks document parsing at any real-world scale)
**Estimated**: 3-4 days
**Dependencies**: None
**Milestone ID**: M-STD-STRING-PERF
**Created**: 2026-04-03

## Context

The docparse agent (ailang-parse) reports that parsing real-world Gmail emails
(40-140KB with quoted-printable encoding and dense HTML) takes 16-30s each, and
a 37KB pure QP email never finishes (killed after 7min, 593MB RAM).

**Important**: The AILANG-level code has already been optimized (M-PERF7, v0.9.3).
The email parser uses split-based O(n) patterns, `foldl` accumulators, `charAt`,
and `join` — it no longer uses the naive recursive substring approach.

**However, the optimized code is still too slow.** A 33KB QP email split on "="
produces 3345 parts. `foldl` over 3345 elements with string-producing callbacks
still runs at 318MB after 20s before being killed. The problem is **two-layered**:

**Layer 1 — Interpreter overhead per `foldl` step:**
Each `foldl` iteration calls `ctx.FnCallerN(fn, []eval.Value{acc, elem})` — a
full evaluator dispatch through the tree-walking interpreter. With 3345 elements,
that's 3345 round-trips through pattern matching, env binding, and recursion
checking. The callback itself does string operations (`charAt`, `substring`, `++`),
each of which allocates.

**Layer 2 — String primitive allocation:**
- `substring` / `_str_slice`: converts entire string to `[]rune`, copies substring
- `toLower` / `toUpper`: allocates new string even when only checking a prefix
- `replace` called N times sequentially: N full-string scans + allocations
- `find` + `substring` pair: `find` computes byte→rune index via `RuneCountInString`,
  then `substring` re-converts the whole string to `[]rune` — double traversal
- `split()` on a 33KB string materializes a 3345-element `[]Value` list

**Layer 1 is the dominant cost.** Even with zero-allocation string primitives,
3345 evaluator round-trips with callback dispatch would still be slow. The
highest-impact fix is to move entire string-processing algorithms into Go
builtins, eliminating interpreter round-trips entirely.

The docparse agent has already identified the practical solution: native
builtins for well-defined string transformations (QP decode, multi-pattern
replace) that avoid the split→foldl→join pattern altogether.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Same semantics, same results — pure performance optimization |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | 0 | All functions remain pure |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | Type signatures unchanged |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Enables AI agents to process real documents (email, HTML) at production scale |
| A8: Minimal Syntax | +1 | `replaceMany` is new but minimal — single function, no syntax changes |
| A9: Cost Visibility | +1 | Faster = more predictable costs; removes hidden O(n^2) traps |
| A10: Composability | +1 | `replaceMany` composes with existing string pipeline patterns |
| A11: Structured Failure | 0 | Error handling unchanged |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** -> **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Pure performance optimization — identical outputs
- [x] A3 (Effects): All functions remain pure, no hidden side effects
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Primary motivation is enabling machine document processing

## Problem Statement

**Observed performance on a 37KB QP-encoded email (2977 `=XX` escapes, 630 lines):**

| Approach | Result | Why |
|----------|--------|-----|
| Recursive substring (original) | OOM at 593MB after 7min | O(n^2) substring copies |
| Split+foldl+join (M-PERF7 optimized) | 318MB after 20s, killed | 3345 evaluator round-trips |
| Line-by-line foldl (~10 escapes/line) | 1.8GB, never finishes | foldl dispatch overhead compounds across 558 lines |

The split+foldl pattern, which is the *idiomatic* AILANG approach, cannot
handle >1000 elements with string-producing callbacks. This is a fundamental
runtime limitation, not a user code issue.

**Cost breakdown for `foldl` over 3345 parts:**

| Cost | Per-step | Total (3345 steps) |
|------|----------|-------------------|
| `FnCallerN` evaluator dispatch | ~10μs (env bind, pattern match) | ~33ms |
| String `++` in callback (growing accumulator) | O(acc_len) copy | O(n^2) total |
| `charAt` / `substring` rune conversion | O(part_len) | O(n) |
| `[]eval.Value` allocation per call | 2 values | 6690 allocations |
| GC pressure from intermediate strings | varies | 318MB observed |

**The string accumulator is the killer.** Each `foldl` step does `acc ++ decoded_part`,
which copies the entire accumulated string. For a 33KB result, the last 1000
steps each copy >30KB. This is O(n^2) at the concat level, even though the
algorithm is O(n) at the element level.

**Test case**: `ailang-parse/data/test_files/challenge/challenge_real_html_qp.eml`
— a real 37KB Gmail message (`text/html`, `Content-Transfer-Encoding: quoted-printable`).
630 lines, 2977 `=XX` escapes (only 26 unique patterns), ~440 lines containing `=`.

**Repro command:**
```bash
cd /path/to/ailang-parse
ailang run --entry main --caps IO,FS,Env --max-recursion-depth 50000 \
  docparse/main.ail data/test_files/challenge/challenge_real_html_qp.eml
```

**What docparse tried (all still hang at 1.8GB):**
1. Recursive `emlDecodeQpString` with `substring(s, eqPos+3, length(s))` per escape
2. `split("=")` + `foldl` over 3345 chunks with record accumulator
3. Line-by-line `foldl` (558 lines, ~10 escapes per line) — still hangs

Even the line-by-line approach (keeping each `split("=")` to ~5 parts) cannot
finish. This is critical: 558 `foldl` iterations with ~10 inner operations each
should be fast, but it still OOMs at 1.8GB. This suggests either:
- **String accumulator growth** is the dominant cost (each outer foldl step
  appends a decoded line to the growing result via `++`, copying the entire
  accumulated output each time — classic Schlemiel the Painter)
- **Nested foldl** (outer over lines, inner over QP segments) compounds the
  dispatch overhead multiplicatively: 558 × 10 = ~5580 FnCallerN calls
- **Memory leak** in the evaluator's value retention — old accumulators may
  not be GC'd promptly

The `++` accumulator hypothesis is most likely: if the result is 33KB, the last
100 foldl steps each copy >30KB. Total copies ≈ 558 × 16KB average ≈ 9MB of
string data, but with Go's GC holding old StringValues, peak RSS balloons.

**This makes Track 3 (native `decodeQuotedPrintable`) the most important fix** —
it bypasses ALL interpreter overhead for the single most expensive operation.

## Goals

**Primary Goal:** Make document parsing of 37KB emails complete in <2s (currently 318MB/20s+ with optimized code, OOM with naive code).

**Success Metrics:**
- 37KB QP email: from never-finishes to <2s
- 40-140KB mixed emails: from 16-30s to <3s each
- `foldl` over 3000+ string elements: from 318MB/20s to <1s/<50MB
- No semantic changes — all existing tests pass unchanged
- Benchmarkable: add Go benchmarks for new builtins on 50KB inputs

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `foldSlices` vs native QP decode as primary fix | Determines whether we solve the general case or the specific case first | human | design | med |
| `replaceMany` API shape (list of tuples vs record) | Public API, hard to change after release | human | design | med |
| Whether `decodeQuotedPrintable` is stdlib or builtin | Builtin = fast but opinionated; stdlib = flexible but slow | human | design | med |
| ASCII fast-path in `_str_slice` | Avoids rune conversion for 99% of real content | agent | compile | low |
| Whether to expose `toLowerASCII` as separate builtin or auto-detected | Affects stdlib surface area | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] `replaceMany` takes `[(string, string)]` — list of (old, new) pairs
- [x] Both `foldSlices` (general) AND `decodeQuotedPrintable` (specific) — general primitive first, native QP as fast-follow
- [ ] Whether to expose `toLowerASCII` as separate builtin or keep `toLower` with auto-detect

## Solution Design

### Overview

Five tracks, ordered by expected impact. Tracks 1-2 are the **highest priority** —
they eliminate the interpreter round-trip problem entirely for the two main
string-processing patterns (split-transform-join, multi-replace).

1. **Track 1: `foldSlices` builtin** — iterate over split segments in Go, no list materialization
2. **Track 2: `replaceMany` builtin** — single-pass multi-pattern replacement
3. **Track 3: Native `decodeQuotedPrintable`** — RFC 2045 §6.7 in pure Go
4. **Track 4: ASCII fast-path for substring/find** — avoid `[]rune` conversion
5. **Track 5: `startsWithIgnoreCase`** — avoid `toLower` allocation for prefix checks

### Track 1: `foldSlices` — Iterate Split Segments Without List Materialization (~6 hours)

**This is the highest-impact change.** It solves the general problem: any
split→process→join pattern over large strings.

**Problem**: `split(s, "=")` on a 33KB string creates a 3345-element `ListValue`,
then `foldl` over it does 3345 `FnCallerN` evaluator dispatches, each creating
intermediate `StringValue` allocations. The accumulated string grows quadratically
because each `++` copies the entire accumulator.

**Solution**: A Go-native builtin that splits and folds in one pass, using a
`strings.Builder` internally to avoid quadratic concatenation:

```go
func strFoldSlicesImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    s, _ := SafeAsString(args[0])
    delim, _ := SafeAsString(args[1])
    acc := args[2]
    fn := args[3]

    if ctx == nil || ctx.FnCallerN == nil {
        return nil, fmt.Errorf("_str_foldSlices: FnCallerN not set")
    }

    // Iterate over segments without allocating a full []string slice
    for {
        idx := strings.Index(s, delim)
        var segment string
        if idx == -1 {
            segment = s
        } else {
            segment = s[:idx]
        }

        segVal := &eval.StringValue{Value: segment}
        var err error
        acc, err = ctx.FnCallerN(fn, []eval.Value{acc, segVal})
        if err != nil {
            return nil, fmt.Errorf("_str_foldSlices: callback error: %w", err)
        }

        if idx == -1 {
            break
        }
        s = s[idx+len(delim):]
    }
    return acc, nil
}
```

**Key difference from `split` + `foldl`:**
- No `ListValue` with 3345 elements allocated
- No intermediate `[]eval.Value` per element
- Segments are Go string slices (zero-copy from source)
- Still calls `FnCallerN` per segment (unavoidable with user callback), but
  avoids the list overhead

**But the accumulator `++` problem remains.** To fully solve this, we need a
companion: `foldSlicesJoin` that assumes the callback returns strings and uses
a `strings.Builder` internally:

```go
func strFoldSlicesJoinImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    s, _ := SafeAsString(args[0])
    delim, _ := SafeAsString(args[1])
    fn := args[2]  // (segment: string) -> string

    var builder strings.Builder
    builder.Grow(len(s))  // pre-allocate output buffer

    first := true
    for {
        idx := strings.Index(s, delim)
        var segment string
        if idx == -1 {
            segment = s
        } else {
            segment = s[:idx]
        }

        segVal := &eval.StringValue{Value: segment}
        result, err := ctx.FnCallerN(fn, []eval.Value{segVal})
        if err != nil {
            return nil, fmt.Errorf("_str_foldSlicesJoin: callback error: %w", err)
        }

        resultStr, _ := SafeAsString(result)
        if !first {
            // Optionally write a separator — or omit for direct concatenation
        }
        builder.WriteString(resultStr)
        first = false

        if idx == -1 {
            break
        }
        s = s[idx+len(delim):]
    }
    return &eval.StringValue{Value: builder.String()}, nil
}
```

**This eliminates the O(n^2) accumulator problem.** The `strings.Builder` grows
amortized O(1) per append, so the total is O(n) regardless of segment count.

**AILANG API**:
```ailang
-- std/string

-- Fold over split segments without materializing the list.
-- Equivalent to foldl(f, acc, split(s, delim)) but O(n) memory.
export pure func foldSlices(s: string, delim: string, acc: a, f: (a, string) -> a) -> a
  = _str_foldSlices(s, delim, acc, f)

-- Split, transform each segment, and join results.
-- Equivalent to join("", map(f, split(s, delim))) but O(n) total with strings.Builder.
-- This is the recommended pattern for any split→transform→concatenate workflow.
export pure func mapSlicesJoin(s: string, delim: string, f: (string) -> string) -> string
  = _str_mapSlicesJoin(s, delim, f)
```

**Usage (QP decode rewritten):**
```ailang
pure func emlDecodeQpSplit(s: string) -> string {
  -- Instead of: join("", map(decodeChunk, split(s, "=")))
  -- Use: mapSlicesJoin — no list allocation, O(n) builder
  let parts = split(s, "=");
  match parts {
    [] => "",
    first :: rest => {
      -- Each chunk after first "=" has hex prefix to decode
      first ++ mapSlicesJoin(join("=", rest), "=", emlDecodeQpChunk)
    }
  }
}
```

Or more directly with `foldSlices`:
```ailang
pure func emlDecodeQpDirect(s: string) -> string {
  let result = foldSlices(s, "=", { first: true, chunks: [] }, func(acc, seg) {
    if acc.first then { first: false, chunks: [seg] }
    else { first: false, chunks: concat(acc.chunks, [emlDecodeHexPrefix(seg)]) }
  });
  join("", result.chunks)
}
```

### Track 2: `replaceMany` Builtin (~4 hours)

**Problem**: HTML entity decoding calls `replace()` 23 times sequentially:
```ailang
let s1 = replace(s, "&amp;", "&")
let s2 = replace(s1, "&lt;", "<")
let s3 = replace(s2, "&gt;", ">")
-- ... 20 more
```

Each call scans the full string. For a 50KB string, that's 23 * 50KB = 1.15MB of scanning.

**Solution**: Single-pass using Go's `strings.NewReplacer` (Aho-Corasick internally):

```go
func strReplaceManyImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
    s, _ := SafeAsString(args[0])
    pairs := args[1] // list of (old, new) tuples

    replacements := extractTuplePairs(pairs)
    oldNew := make([]string, 0, len(replacements)*2)
    for _, pair := range replacements {
        oldNew = append(oldNew, pair.old, pair.new)
    }
    replacer := strings.NewReplacer(oldNew...)
    return &eval.StringValue{Value: replacer.Replace(s)}, nil
}
```

**AILANG API**:
```ailang
export pure func replaceMany(s: string, replacements: [(string, string)]) -> string
  = _str_replaceMany(s, replacements)
```

**Usage**:
```ailang
let decoded = replaceMany(raw, [
  ("&amp;", "&"), ("&lt;", "<"), ("&gt;", ">"),
  ("&quot;", "\""), ("&#39;", "'")
])
```

### Track 3: Native `decodeQuotedPrintable` (~3 hours)

**Rationale**: QP encoding is a well-defined RFC (2045 §6.7). The rules are:
- `=XX` → byte with hex value XX
- `=\n` (soft line break) → removed
- Everything else → literal

This is a single-pass O(n) transformation with no ambiguity. Implementing it
in Go eliminates ALL interpreter overhead for the single most expensive
operation in email parsing.

```go
func strDecodeQPImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
    s, _ := SafeAsString(args[0])

    var buf strings.Builder
    buf.Grow(len(s))

    i := 0
    for i < len(s) {
        if s[i] == '=' {
            if i+2 < len(s) {
                if s[i+1] == '\r' || s[i+1] == '\n' {
                    // Soft line break — skip =\r\n or =\n
                    i += 2
                    if i < len(s) && s[i] == '\n' { i++ }
                    continue
                }
                b, err := hex.DecodeString(s[i+1 : i+3])
                if err == nil {
                    buf.Write(b)
                    i += 3
                    continue
                }
            }
        }
        buf.WriteByte(s[i])
        i++
    }
    return &eval.StringValue{Value: buf.String()}, nil
}
```

**AILANG API**:
```ailang
-- std/encoding (new module) or std/string
export pure func decodeQuotedPrintable(s: string) -> string
  = _str_decodeQP(s)
```

**Impact**: The 37KB email with 3345 `=` signs would be processed in a single
Go function call — no split, no foldl, no intermediate list, no 3345 callbacks.
Expected time: <1ms for 37KB.

### Track 4: ASCII Fast-Path for `_str_slice` and `_str_find` (~4 hours)

**Root cause**: `_str_slice` always converts to `[]rune` (line 315 of string.go),
even for pure ASCII strings where byte index == rune index.

**Fix**: Check if string is ASCII-only. For ASCII, use direct byte slicing (zero allocation):

```go
func strSliceImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    str, _ := SafeAsString(args[0])
    start, _ := SafeAsInt(args[1])
    end, _ := SafeAsInt(args[2])

    length := len(str)

    // ASCII fast path: byte index == rune index
    if utf8.RuneCountInString(str) == length {
        if start < 0 { start = 0 }
        if end > length { end = length }
        if start > end { start = end }
        return &eval.StringValue{Value: str[start:end]}, nil
    }

    // Unicode slow path (existing code)
    runes := []rune(str)
    // ... existing clamping and return
}
```

Same pattern for `_str_find` — skip `RuneCountInString(haystack[:byteIdx])` when
ASCII-only, since byte offset == rune offset.

### Track 5: `startsWithIgnoreCase` (~2 hours)

**Problem**: `htmlFixVoidFragment` calls `toLower(fragment)` on every fragment
just to check if it starts with a void tag name. Allocates a full copy.

**Solution**: Prefix-only comparison using `strings.EqualFold`:

```go
func strStartsWithICImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
    s, _ := SafeAsString(args[0])
    prefix, _ := SafeAsString(args[1])
    if len(s) < len(prefix) {
        return eval.FalseValue, nil
    }
    return eval.BoolValue(strings.EqualFold(s[:len(prefix)], prefix)), nil
}
```

**AILANG API**:
```ailang
export pure func startsWithIgnoreCase(s: string, prefix: string) -> bool
  = _str_startsWithIC(s, prefix)
```

### Implementation Plan

**Phase 1: `foldSlices` + `mapSlicesJoin`** (~6 hours) — HIGHEST IMPACT
- [ ] Implement `_str_foldSlices` in `internal/builtins/string_ops.go`
- [ ] Implement `_str_mapSlicesJoin` with `strings.Builder` accumulation
- [ ] Register both builtins with proper type signatures
- [ ] Add to `std/string.ail`
- [ ] Add Go benchmarks: `BenchmarkFoldSlices_3000Segments` vs `split+foldl`
- [ ] Add unit tests (empty string, no delimiters, single segment, Unicode delimiters)

**Phase 2: `replaceMany`** (~4 hours)
- [ ] Implement `_str_replaceMany` using `strings.NewReplacer`
- [ ] Register in `internal/builtins/string_ops.go`
- [ ] Add to `std/string.ail`
- [ ] Add unit tests and benchmark
- [ ] Add example file `examples/runnable/replace_many.ail`

**Phase 3: `decodeQuotedPrintable`** (~3 hours)
- [ ] Implement `_str_decodeQP` with single-pass Go scanner
- [ ] Register in `internal/builtins/string_ops.go` (or new `string_encoding.go`)
- [ ] Add to `std/string.ail` or new `std/encoding.ail`
- [ ] Test against RFC 2045 §6.7 test vectors
- [ ] Benchmark: 37KB QP string decode time

**Phase 4: ASCII fast-paths** (~4 hours)
- [ ] Add ASCII check helper: `func isASCII(s string) bool`
- [ ] Optimize `_str_slice` with ASCII fast-path
- [ ] Optimize `_str_find` to skip rune counting for ASCII
- [ ] Add Go benchmarks: `BenchmarkStrSlice_ASCII_50KB`, `BenchmarkStrSlice_Unicode_50KB`
- [ ] Verify all existing string tests pass

**Phase 5: `startsWithIgnoreCase` + integration validation** (~3 hours)
- [ ] Implement `_str_startsWithIC`
- [ ] Add to `std/string.ail`
- [ ] Run docparse email benchmark with 37KB QP test email
- [ ] Measure before/after timing for each track independently
- [ ] Run `make test && make verify-examples`
- [ ] Update CHANGELOG.md

### Files to Modify/Create

**Modified files:**
- `internal/builtins/string.go` — ASCII fast-path for `_str_slice`, `_str_find` (~30 LOC)
- `internal/builtins/string_ops.go` — `_str_replaceMany`, `_str_startsWithIC`, `_str_foldSlices`, `_str_mapSlicesJoin` (~200 LOC)
- `std/string.ail` — export new functions (~15 LOC)

**New files:**
- `internal/builtins/string_encoding.go` — `_str_decodeQP` (~80 LOC)
- `internal/builtins/string_bench_test.go` — Go benchmarks for all new builtins (~150 LOC)
- `examples/runnable/replace_many.ail` — Example file (~15 LOC)

## Examples

### Example 1: Quoted-Printable Decode (native builtin vs AILANG)

**Before (split+foldl — 318MB, 20s+, killed):**
```ailang
pure func emlDecodeQpSplit(s: string) -> string {
  let parts = split(s, "=");
  match parts {
    [] => "",
    first :: rest => {
      let decoded = foldl(func(acc, chunk) {
        acc ++ emlDecodeHexPrefix(chunk)
      }, first, rest);
      decoded
    }
  }
}
```

**After (native builtin — <1ms, O(n)):**
```ailang
import std/string (decodeQuotedPrintable)

pure func emlDecodeBody(text: string) -> string {
  decodeQuotedPrintable(text)
}
```

### Example 2: Generic Split-Transform-Join (mapSlicesJoin)

**Before (materializes 3345-element list):**
```ailang
let result = join("", map(transform, split(content, delim)))
```

**After (zero list allocation, O(n) builder):**
```ailang
let result = mapSlicesJoin(content, delim, transform)
```

### Example 3: HTML Entity Decoding (replaceMany)

**Before (23 sequential full-string scans):**
```ailang
let s1 = replace(s, "&amp;", "&")
let s2 = replace(s1, "&lt;", "<")
let s3 = replace(s2, "&gt;", ">")
-- ... 20 more
```

**After (single pass, Aho-Corasick):**
```ailang
let decoded = replaceMany(raw, [
  ("&amp;", "&"), ("&lt;", "<"), ("&gt;", ">"),
  ("&quot;", "\""), ("&#39;", "'"),
  ("&nbsp;", " "), ("&mdash;", "-"), ("&ndash;", "-")
])
```

### Example 4: Case-Insensitive Tag Matching

**Before (allocates lowered copy of entire fragment):**
```ailang
pure func htmlFixVoidFragment(fragment: string) -> string {
  let lower = toLower(fragment)
  let voidTag = htmlMatchVoidTag(lower)
  -- ...
}
```

**After (no allocation, checks prefix only):**
```ailang
pure func htmlMatchVoidTag(fragment: string) -> string {
  if startsWithIgnoreCase(fragment, "br") then "br"
  else if startsWithIgnoreCase(fragment, "hr") then "hr"
  else if startsWithIgnoreCase(fragment, "img") then "img"
  -- ...
  else ""
}
```

## Success Criteria

- [ ] 37KB QP email parses in <2s (currently 318MB/20s+ with optimized code)
- [ ] 140KB mixed email parses in <5s (currently 30s)
- [ ] `mapSlicesJoin` over 3000 segments completes in <100ms (currently 20s+ via split+foldl)
- [ ] `decodeQuotedPrintable` processes 37KB in <1ms
- [ ] `replaceMany` with 23 patterns on 50KB string is >10x faster than 23 sequential `replace`
- [ ] `BenchmarkStrSlice_ASCII_50KB` shows >5x improvement over rune path
- [ ] All existing tests pass (`make test && make verify-examples`)
- [ ] No semantic changes — output identical for all inputs
- [ ] Documentation updated (CHANGELOG, examples)

## Testing Strategy

**Go benchmarks (new):**
- `BenchmarkFoldSlices_3000Segs` — vs `split` + `foldl` over same data
- `BenchmarkMapSlicesJoin_3000Segs` — vs `join("", map(f, split(s, d)))`
- `BenchmarkDecodeQP_37KB` — native QP decode on real email content
- `BenchmarkReplaceMany_23Patterns_50KB` — vs 23 sequential replace calls
- `BenchmarkStrSlice_ASCII_50KB` — ASCII fast-path vs rune conversion
- `BenchmarkStrSlice_Unicode_50KB` — verify Unicode path still works
- `BenchmarkStartsWithIC` — vs toLower + startsWith

**Unit tests:**
- `foldSlices`: empty string, no delimiter found, single segment, Unicode delimiters, delimiter at start/end
- `mapSlicesJoin`: identity transform, segment expansion, empty segments
- `decodeQuotedPrintable`: RFC 2045 test vectors, soft line breaks (`=\r\n`), invalid hex (`=GG`), `=` at end of string, UTF-8 encoded via QP
- `replaceMany`: empty list, single pair, overlapping patterns, Unicode patterns
- `startsWithIgnoreCase`: ASCII, Unicode, empty strings, exact match
- ASCII fast-path: verify ASCII detection, boundary cases (last char non-ASCII)

**Integration test (repro file from docparse):**
- File: `ailang-parse/data/test_files/challenge/challenge_real_html_qp.eml` (37KB, 2977 QP escapes)
- Repro: `ailang run --entry main --caps IO,FS,Env --max-recursion-depth 50000 docparse/main.ail <file>`
- Parse end-to-end, compare output with known-good decoded result
- Measure time: must complete in <2s (currently never finishes)
- Measure memory: must stay under 50MB (currently 1.8GB)
- Test each track independently to measure per-track improvement

## Deferred Decisions

The following are intentionally left open for the implementer:

- Whether to cache ASCII-ness on StringValue (add a `isASCII *bool` field) — agent may choose based on benchmarks
- Internal algorithm for `replaceMany` (Go's `strings.NewReplacer` vs custom) — agent may choose
- Whether `startsWithIgnoreCase` uses `EqualFold` or byte-level comparison — agent may choose
- Whether `decodeQuotedPrintable` goes in `std/string` or a new `std/encoding` module — agent may choose
- Whether `foldSlices` callback receives segment index as a third argument — agent may choose based on usage patterns

## Non-Goals

**Not attempted in this feature:**
- Rope/persistent data structure for strings — too complex for the gain, revisit in v1.0
- Zero-copy string views at the Value level — would require GC changes and lifetime tracking
- Regex support — separate feature, separate design doc
- Tail-call optimization — compiler-level change, tracked separately
- `foldChars` improvements — already implemented in M-PERF7, working correctly
- General `foldl` performance — the interpreter dispatch overhead affects all builtins, not just strings; addressed by M-PERF4 (bytecode interpreter)
- `--profile` flag — valuable but separate concern, would benefit all AILANG programs not just string processing

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `foldSlices` callback still slow due to evaluator dispatch | Med | Even with list overhead removed, 3000+ FnCallerN calls may be slow. Track 3 (native QP) is the fallback. Benchmark both. |
| `mapSlicesJoin` assumes callback returns string | Low | Type signature enforces `(string) -> string`; runtime check on return value |
| `decodeQuotedPrintable` scope creep (charset decoding, MIME boundaries) | Med | Strictly RFC 2045 §6.7 — byte-level decode only, no charset interpretation |
| `strings.NewReplacer` pattern overlap behavior differs from sequential replace | Med | Document that `replaceMany` uses first-match-wins semantics; add tests for overlapping patterns |
| ASCII fast-path incorrect for edge cases | Low | Go strings are immutable — no TOCTOU risk |
| Docparse bottleneck is elsewhere (XML parsing, not strings) | Med | Benchmark each track independently; profile end-to-end |

## Related Documents

**Implemented (informs design):**
- [m-perf5-data-intensive-workloads.md](../../implemented/v0_9_2/m-perf5-data-intensive-workloads.md) — Bulk XML ops, similar "reduce interpreter round-trips" pattern
- [m-dx11-string-split-builtin.md](../../implemented/v0_4_6/m-dx11-string-split-builtin.md) — `split` builtin that docparse now relies on
- [m-gap5-stdlib-repeat.md](../../implemented/v0_7_0/m-gap5-stdlib-repeat.md) — String repeat builtin

**Planned (check for overlap):**
- [m-perf4-bytecode-interpreter.md](../../planned/v1_0_0/m-perf4-bytecode-interpreter.md) — Would make all builtins faster via reduced dispatch overhead
- [m-std-map-and-array-gaps.md](m-std-map-and-array-gaps.md) — Similar stdlib gap-filling work
- [m-concat-disambiguation.md](../../planned/v0_10_1/m-concat-disambiguation.md) — **Synergistic**: removing `++` for strings and pushing users toward `join(parts)` naturally eliminates the O(n²) accumulator anti-pattern. String interpolation (`"${acc}${part}"`) has the same O(n²) cost as `acc ++ part` — both copy the growing accumulator. But `join("", parts)` is O(n) via `strings.Builder`. The concat disambiguation redesign aligns with this perf work by discouraging accumulators at the language level.

**Source messages:**
- Inbox `bb6055e2`: Bug: O(n^2) string performance — substring/find/toLower copy instead of slice
- Inbox `f066ef0d`: Feature: String builder / foldString / replaceMany for O(n) string processing

## References

- [Design Axioms](/docs/references/axioms)
- Go `strings.NewReplacer` — uses Aho-Corasick for multi-pattern replacement
- `internal/builtins/string.go:301-330` — current `_str_slice` implementation
- `internal/builtins/string_ops.go:721-735` — current `_str_replace` implementation
- `internal/builtins/string_char.go:69-91` — existing `foldChars` implementation

## Future Work

- **`--profile` flag**: `ailang run --profile` to show where time is spent — would help docparse and all users identify bottlenecks without guessing
- **More native encodings**: `decodeBase64`, `decodeURLEncoded`, `decodeMIMEHeader` — same pattern as QP, well-defined RFCs
- **String interning**: Cache frequently-used strings (tag names, entity names) to avoid repeated allocation
- **Compiled replacer reuse**: If the same replacement list is used repeatedly, cache the `NewReplacer` across calls
- **Bytecode interpreter (M-PERF4)**: Would make `foldSlices` callbacks ~10x faster by eliminating tree-walk dispatch

---

**Document created**: 2026-04-03
**Last updated**: 2026-04-03

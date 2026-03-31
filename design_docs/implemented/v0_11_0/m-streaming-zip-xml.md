# M-STREAMING-ZIP-XML: Streaming ZIP Entry + XML Fold

**Status**: Implemented
**Target**: v0.11.0
**Priority**: P1 (High — blocks XLSX >5 MB, OOMs at 8.7 MB)
**Estimated**: 2-3 days
**Dependencies**: M-STD-MAP (for `std/array` gaps: `empty`, `append`)
**Requested by**: ailang-parse agent (2026-03-31, msg 99e3dcd6)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Fold order is document order (deterministic) |
| A2: Replayability | 0 | Same input produces same fold result |
| A3: Effect Legibility | 0 | ZIP reading declares {FS}; XML fold is pure |
| A4: Explicit Authority | 0 | No new capabilities |
| A5: Bounded Verification | 0 | No new type system nodes — uses existing XmlNode + function types |
| A6: Safe Concurrency | 0 | Single-threaded fold, no shared state |
| A7: Machines First | +2 | Enables AI agents to process real XLSX documents at scale |
| A8: Minimal Syntax | 0 | No new syntax — library-level builtins only |
| A9: Cost Visibility | +3 | O(largest element) memory vs O(entire document); 100MB -> ~1KB per element |
| A10: Composability | +1 | Fold composes with any accumulator type |
| A11: Structured Failure | 0 | Returns Result — errors propagated cleanly |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +6** -> **Decision: Move forward**

## Problem Statement

XLSX parsing OOMs on an 8.7 MB file because `readZipEntry` decompresses the entire
`xl/sharedStrings.xml` (100 MB decompressed) into a Go string via `io.ReadAll()`,
then `parseElements` scans that 100 MB string.

The M-STD-MAP design doc addresses the O(n) lookup bottleneck (shared string
resolution via list traversal). This doc addresses the memory bottleneck: holding
100 MB of raw XML in memory.

### Memory timeline (current)

```
readZipEntry("xl/sharedStrings.xml")
  -> io.ReadAll() allocates ~100 MB string              [100 MB]
  -> parseElements scans the string, builds XmlNode list [100 MB + result list]
  -> Total peak: ~200 MB for just shared strings
  -> Multiply by worksheet XML: OOM
```

### Memory timeline (proposed)

```
zipXmlScanFold("xl/sharedStrings.xml", "si", [], handler)
  -> zip.File.Open() returns io.ReadCloser (streaming)   [0 MB — streamed]
  -> xml.NewDecoder(reader) reads token-by-token          [~4 KB buffer]
  -> scanForElements builds one XmlNode at a time         [~1 KB per element]
  -> handler called, element GC'd before next             [accumulator only]
  -> Total peak: ~4 KB + accumulator size
```

### Impact

| File type | Current | With streaming |
|-----------|---------|----------------|
| DOCX 50 MB | 5s, OK | No change needed |
| PPTX 50 MB | 9s, OK | No change needed |
| XLSX 8.7 MB | OOM (2 GB) -> SIGKILL | Expected: <10s, <100 MB |
| XLSX 50 MB | Impossible | Expected: <60s, <200 MB |

## Goals

**Primary Goal:** Process XLSX files up to 50 MB without OOM.

**Success Metrics:**
- ZIP entry streaming: reads decompressed data without `io.ReadAll()`
- XML fold: processes elements via callback without building full result list
- Combined ZIP+XML fold: pipes Go streams internally, zero-copy to AILANG
- XLSX 8.7 MB: completes without OOM, <10s
- All existing ZIP and XML tests still pass

## Key Design Insight

The ailang-parse agent requested two separate features: raw byte streaming from ZIP
entries and SAX-style XML parsing. But we can solve the problem more elegantly by
observing that:

1. Go's `zip.File.Open()` already returns an `io.ReadCloser` (a stream)
2. Go's `xml.NewDecoder()` already accepts an `io.Reader` (a stream)
3. Our `scanForElements()` already walks the token stream efficiently

We just need to **connect them inside Go** without materializing the decompressed
bytes into an AILANG string. The AILANG callback only sees parsed `XmlNode` values,
never raw bytes. This is simpler, safer, and more memory-efficient than exposing
raw byte streaming to AILANG.

## Current State

### What exists

| Component | Status | Notes |
|-----------|--------|-------|
| `readZipEntry()` | Full materialization | `io.ReadAll()` on line 567 of zip.go |
| `scanForElements()` | Token-streaming | Already skips non-matching elements efficiently |
| `ctx.FnCaller()` | Working | Used by list builtins and stream builtins |
| `parseElements` builtin | Collects into list | Returns `List[XmlNode]` — all elements in memory |

### What's missing

| Component | Description |
|-----------|-------------|
| ZIP entry stream reader | Open ZIP entry without `io.ReadAll()` |
| XML fold builtin | `scanForElements` variant that calls handler instead of collecting |
| Combined ZIP+XML fold | Pipe `zip.Open()` -> `xml.Decoder` -> fold |

## Solution Design

### Track 1: `_xml_parseFold` — XML fold over string (Pure)

A fold variant of `parseElements` that calls a handler for each matching element
instead of collecting them into a list.

```go
// _xml_parseFold(xml: string, tag: string, init: a, handler: (a, XmlNode) -> a) -> Result[a, string]
func xmlParseFoldImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    xmlStr := args[0].(*eval.StringValue).Value
    tag := args[1].(*eval.StringValue).Value
    acc := args[2]          // initial accumulator
    handler := args[3]      // fold function: (acc, node) -> acc

    decoder := xml.NewDecoder(strings.NewReader(xmlStr))
    var foldErr error
    acc, foldErr = scanForElementsFold(decoder, tag, acc, func(node eval.Value) (eval.Value, error) {
        return ctx.FnCallerN(handler, []eval.Value{acc, node})
    })
    if foldErr != nil {
        return xmlMakeErr(foldErr.Error()), nil
    }
    return xmlMakeOk(acc), nil
}
```

**AILANG signature:**
```ailang
export pure func parseFold[a](xml: string, tag: string, init: a, f: (a, XmlNode) -> a) -> Result[a, string]
```

**Why pure?** The fold function is pure — it takes a string and returns a value.
No FS effect needed because the XML is already in memory.

### Track 2: `_zip_xml_scanFold` — Combined ZIP + XML fold (Effectful)

The key builtin: opens a ZIP entry as a stream, pipes it into an XML decoder,
and folds over matching elements. The decompressed XML never exists as a
contiguous string in memory.

```go
// _zip_xml_scanFold(zipPath, entryName, tag, init, handler) -> Result[a, string] ! {FS}
func zipXmlScanFoldImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    zipPath := args[0].(*eval.StringValue).Value
    entryName := args[1].(*eval.StringValue).Value
    tag := args[2].(*eval.StringValue).Value
    acc := args[3]
    handler := args[4]

    // 1. Open ZIP archive
    archive, err := zip.OpenReader(zipPath)
    if err != nil {
        return zipMakeErr(err.Error()), nil
    }
    defer archive.Close()

    // 2. Find entry
    var entry *zip.File
    for _, f := range archive.File {
        if f.Name == entryName {
            entry = f
            break
        }
    }
    if entry == nil {
        return zipMakeErr(fmt.Sprintf("entry not found: %s", entryName)), nil
    }

    // 3. Open decompressed stream (NOT io.ReadAll!)
    rc, err := entry.Open()
    if err != nil {
        return zipMakeErr(err.Error()), nil
    }
    defer rc.Close()

    // 4. Pipe stream directly into XML decoder
    decoder := xml.NewDecoder(rc)  // <-- KEY: reads from stream, not string

    // 5. Fold over matching elements
    var foldErr error
    scanForElementsFold(decoder, tag, &acc, func(node eval.Value) error {
        newAcc, err := ctx.FnCallerN(handler, []eval.Value{acc, node})
        if err != nil {
            return err
        }
        acc = newAcc
        return nil
    })
    if foldErr != nil {
        return zipMakeErr(foldErr.Error()), nil
    }

    return zipMakeOk(acc), nil
}
```

**AILANG signature:**
```ailang
export func scanFold[a](zipPath: string, entryName: string, tag: string,
                        init: a, f: (a, XmlNode) -> a) -> Result[a, string] ! {FS}
```

**Memory profile:** The `xml.Decoder` reads from the `io.ReadCloser` in small
internal buffers (~4 KB). Each matching element is built, passed to the handler,
and then eligible for GC. Peak memory is O(single element + accumulator).

### Track 3: `_zip_readEntryStream` — General streaming ZIP (Optional)

For non-XML use cases, a general-purpose streaming ZIP reader that calls a handler
with chunks of decompressed data.

```go
// _zip_readEntryStream(zipPath, entryName, chunkSize, init, handler) -> Result[a, string] ! {FS}
```

**AILANG signature:**
```ailang
export func readEntryStream[a](path: string, entry: string, chunkSize: int,
                               init: a, f: (a, string) -> a) -> Result[a, string] ! {FS}
```

**Deferred**: Not needed for XLSX. Only implement if other file formats need raw
streaming. Keep the design slot open but don't build it in v0.11.0.

### Core helper: `scanForElementsFold`

Adapts the existing `scanForElements` to use a fold callback instead of list
accumulation. Minimal diff from the existing function.

```go
// scanForElementsFold walks the XML token stream, calling handler for each
// element matching tagName. Returns the final accumulator value.
// This is the fold variant of scanForElements.
func scanForElementsFold(
    decoder *xml.Decoder,
    tagName string,
    acc *eval.Value,
    handler func(node eval.Value) error,
) error {
    for {
        tok, err := decoder.Token()
        if err != nil {
            return nil // EOF — normal termination
        }

        switch t := tok.(type) {
        case xml.StartElement:
            resolvedName := resolveTagName(t.Name, nil)
            if resolvedName == tagName {
                localPM := extractPrefixMap(t, nil)
                attrs := buildAttrs(t, localPM)
                childNodes, err := parseXmlChildren(decoder, 1, localPM)
                if err != nil {
                    return err
                }
                finalTag := resolveTagName(t.Name, localPM)
                node := makeXmlElement(finalTag, attrs, childNodes)

                // Call handler instead of appending to list
                if err := handler(node); err != nil {
                    return err
                }
            } else {
                if err := scanForElementsFold(decoder, tagName, acc, handler); err != nil {
                    return err
                }
            }
        case xml.EndElement:
            return nil
        }
    }
}
```

### Stdlib Wrappers

**`std/xml.ail` additions:**

```ailang
-- Fold over XML elements matching a tag. Calls handler for each match
-- with an accumulator, returning the final accumulated value.
-- Memory: O(largest element + accumulator), not O(document).
--
-- COST MODEL:
--   parseFold(xml, tag, init, f)  O(n) scan, O(1) memory per element
export pure func parseFold[a](xml: string, tag: string, init: a, f: (a, XmlNode) -> a) -> Result[a, string] {
    _xml_parseFold(xml, tag, init, f)
}
```

**`std/zip.ail` additions:**

```ailang
-- Stream a ZIP entry through an XML parser, folding over matching elements.
-- The decompressed XML is never held entirely in memory.
--
-- COST MODEL:
--   scanFold(path, entry, tag, init, f)  O(n) scan, O(1) memory per element
--   Peak memory: ~4 KB XML buffer + accumulator size
export func scanFold[a](path: string, entry: string, tag: string,
                        init: a, f: (a, XmlNode) -> a) -> Result[a, string] ! {FS} {
    _zip_xml_scanFold(path, entry, tag, init, f)
}
```

**Alternative: new `std/zip/xml.ail` module?** No — `scanFold` naturally belongs
in `std/zip` since it operates on ZIP files. The XmlNode type dependency is fine
since it's already in the type universe via `std/xml`.

### Usage Example (XLSX shared string extraction)

```ailang
import std/zip (scanFold)
import std/array (empty, append, length)
import std/xml (getText, getChildren)

-- Extract shared strings from XLSX without OOM
-- Memory: O(array size) not O(decompressed XML size)
func extractSharedStrings(xlsxPath: string) -> Result[Array[string], string] ! {FS} {
    scanFold(xlsxPath, "xl/sharedStrings.xml", "si", empty(), \acc, si.
        let text = si |> getChildren |> findText
        append(acc, text)
    )
}
```

## Implementation Plan

### Milestone 1: `scanForElementsFold` helper (Day 1, 2-3 hours)
- Add `scanForElementsFold()` to `internal/builtins/xml.go`
- Reuse all existing helpers: `resolveTagName`, `extractPrefixMap`, `buildAttrs`, `parseXmlChildren`, `makeXmlElement`
- Unit test: fold over XML string, verify accumulator result matches `parseElements` output
- `make test` passes

### Milestone 2: `_xml_parseFold` builtin (Day 1, 2-3 hours)
- Register new pure builtin in `xml.go`
- Type: `(string, string, a, (a, XmlNode) -> a) -> Result[a, string]`
- Uses `ctx.FnCallerN` for the fold callback
- Add `parseFold` to `std/xml.ail`
- Create `examples/runnable/xml_fold.ail`
- `make verify-examples` passes

### Milestone 3: `_zip_xml_scanFold` builtin (Day 2, 3-4 hours)
- Register new effectful builtin in `zip.go` (or new `zip_xml.go`)
- Type: `(string, string, string, a, (a, XmlNode) -> a) -> Result[a, string] ! {FS}`
- Opens ZIP entry stream, pipes to `xml.NewDecoder`, calls `scanForElementsFold`
- Add `scanFold` to `std/zip.ail`
- Create `examples/runnable/zip_xml_fold.ail`
- `make verify-examples` passes

### Milestone 4: Integration test + benchmarks (Day 2-3, 2-3 hours)
- Test with real XLSX files (8.7 MB problem file)
- Verify no OOM, measure peak memory
- Benchmark: compare `readEntry + parseElements` vs `scanFold`
- Update XLSX parsing code in ailang-parse to use `scanFold`

## Complexity Assessment

**Overall: MEDIUM (2-3 days)**

This is significantly simpler than M-STD-MAP because:

| Dimension | M-STD-MAP | M-STREAMING-ZIP-XML |
|-----------|-----------|---------------------|
| New type system nodes | Yes (TMap, 14 files) | No |
| New eval value types | Yes (MapValue) | No |
| New builtins | 10 | 2-3 |
| Type switch updates | 14 files | 0 |
| Callback infrastructure | N/A | Already exists (`ctx.FnCaller`) |
| Core change | `scanForElements` -> fold variant | ~30 lines diff |

The heaviest part is the combined ZIP+XML builtin, which is straightforward Go
plumbing (pipe `io.ReadCloser` to `xml.NewDecoder`).

## Alternatives Considered

### 1. Expose raw byte streaming to AILANG
The agent requested `readZipEntryStream(path, entry, chunkSize, handler)` for
raw byte chunks. This would require AILANG code to reassemble partial XML across
chunk boundaries — extremely error-prone and unnecessary since Go can pipe streams
internally.
**Rejected:** Leaks implementation complexity to AILANG level.

### 2. SAX token API (full SAX events to AILANG)
Expose `StartElement`, `EndElement`, `Text`, etc. as individual callbacks. More
flexible but requires AILANG code to track state, handle nesting, build elements.
**Deferred:** `parseFold` covers 95% of use cases. SAX can be added later if needed.

### 3. Iterator/generator protocol
A lazy iterator that yields elements on demand. Would require new runtime
infrastructure (coroutines or continuations) that AILANG doesn't have.
**Rejected:** Massive runtime complexity for marginal benefit over fold.

### 4. Increase memory limits
Just raise `zipMaxDecompressedSize` and hope for the best.
**Rejected:** Doesn't scale. 50 MB XLSX = 500 MB+ decompressed XML.

### 5. Separate `_zip_readEntryToDecoder` + `_xml_decoderFold`
Expose an opaque `Decoder` value that can be passed between builtins. More
composable but requires a new opaque value type and lifecycle management.
**Deferred:** Adds complexity. The combined builtin is sufficient for now.

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Callback error propagation | Medium | Medium | Test handler errors in fold; ensure proper cleanup of ZIP/XML resources |
| Handler panics | Low | High | Recover in Go, return error Result |
| XML entities across buffer boundaries | None | N/A | `xml.Decoder` handles buffering internally |
| Partial element at EOF | Low | Low | `xml.Decoder` returns proper error, fold stops cleanly |
| Type inference for fold accumulator | Medium | Medium | Follow `_list_takeMap` pattern — already solved |
| Large accumulator (200K strings) | Low | Low | Array append is O(n) copy; acceptable for this scale |

## Non-Goals

- Raw byte streaming from ZIP entries to AILANG (deferred)
- Full SAX event API (deferred)
- Iterator/generator protocol (out of scope)
- Mutable accumulator optimization (copy-on-write is sufficient)
- XLSX-specific builtins (keep using generic ZIP+XML)
- Parallel element processing (single-threaded fold is correct)

## Related Documents

- Agent message: `99e3dcd6` (streaming ZIP + SAX XML request from ailang-parse)
- `design_docs/planned/v0_11_0/m-std-map-and-array-gaps.md` — Companion: Map type + array gaps
- `internal/builtins/xml.go:268` — `scanForElements()` (base for fold variant)
- `internal/builtins/zip.go:554` — `readZipEntry()` (the `io.ReadAll` bottleneck)
- `internal/builtins/list_bounded.go` — Callback pattern via `ctx.FnCaller`

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-03-31 | Pipe Go streams internally, don't expose bytes to AILANG | Simpler, safer, no partial-XML handling in AILANG |
| 2026-03-31 | Fold over elements, not SAX tokens | Covers XLSX use case; SAX adds complexity without benefit |
| 2026-03-31 | Combined ZIP+XML builtin | Avoids need for opaque Decoder value type |
| 2026-03-31 | Defer `readEntryStream` | Raw byte streaming not needed for XLSX |
| 2026-03-31 | Put `scanFold` in std/zip, not new module | Natural home since it operates on ZIP files |

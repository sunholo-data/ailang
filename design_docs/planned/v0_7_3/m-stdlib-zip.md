# M-STDLIB-ZIP: ZIP Archive Standard Library

**Status**: Planned
**Target**: v0.7.3
**Priority**: P2 (Medium)
**Estimated**: 1-2 days
**Dependencies**: None (Go `archive/zip` stdlib)
**Author**: Claude (Opus 4.6)
**Date**: 2026-02-08
**Requested by**: docparse-demo agent (message 62662386)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Same file → same ZIP contents (deterministic) |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | +1 | ZIP ops explicitly require `! {FS}` — no hidden I/O |
| A4: Explicit Authority | +1 | Reuses FS capability + sandbox; no ambient access |
| A5: Bounded Verification | 0 | No new verification concerns |
| A6: Safe Concurrency | 0 | Stateless per-call; no shared handles |
| A7: Machines First | +1 | Enables AI agents to process document formats (DOCX, XLSX) |
| A8: Minimal Syntax | 0 | No new syntax — just builtins |
| A9: Cost Visibility | 0 | No new cost model |
| A10: Composability | +1 | Composes with std/xml for document parsing pipelines |
| A11: Structured Failure | +1 | Returns `Result[T, string]` — typed error handling |
| A12: System Boundary | 0 | FS boundary already explicit |

**Net Score: +5** → **Decision: Accept**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): ZIP reading explicitly requires FS effect
- [x] A4 (Authority): No ambient access — reuses FS capability and sandbox
- [x] A7 (Machines First): Improves AI agent capabilities for document processing

## Related Documents

- [m-stdlib-xml.md](m-stdlib-xml.md) — Companion module for parsing XML inside ZIPs
- [m-stdlib-datetime.md](../../implemented/v0_7_0/m-stdlib-datetime.md) — Similar stdlib pattern (effects split, pure/effectful separation)
- [m-stdlib-gaps.md](../v0_7_2/m-stdlib-gaps.md) — Stdlib gap analysis from eval results
- [m-wasm-stdlib.md](../v0_7_2/m-wasm-stdlib.md) — WASM stdlib embedding (new modules need WASM inclusion)

## Problem Statement

AILANG has no way to read ZIP archives. Many document formats (DOCX, PPTX, XLSX, EPUB, JAR) are ZIP containers with XML/JSON payloads. Without ZIP support, AILANG cannot parse any of these formats.

The immediate use case is a DocParse demo that showcases AILANG's pattern matching on document structures. DOCX files are ZIP archives containing XML — we need `std/zip` to open them and `std/xml` (separate design doc) to parse the contents.

### Motivating Example

```ailang
module demos/docparse

import std/zip (listEntries, readEntry)
import std/xml (parseXml, findAll, getText)

-- Extract text from a DOCX file
func extractDocxText(path: string) -> string ! {FS} =
  let xmlContent = readEntry(path, "word/document.xml")
  in match parseXml(xmlContent) with
    | Ok(root) ->
      let paragraphs = findAll(root, "w:p")
      in _list_join(
        _list_map(\p. getText(p), paragraphs),
        "\n"
      )
    | Err(msg) -> "Parse error: " ++ msg
```

## Goals

**Primary Goal:** Read-only ZIP archive access for document parsing use cases.

**Success Metrics:**
- List entries in a ZIP archive
- Read text entries by name
- Read binary entries as base64-encoded strings
- All operations use `! {FS}` effect (ZIP reading = filesystem access)
- Works with DOCX, XLSX, PPTX archives

**Non-Goals (v0.7.3):**
- ZIP creation / writing (future: v0.8+)
- Streaming/incremental decompression
- Password-protected ZIPs
- ZIP64 extensions (files > 4GB)

## Solution Design

### Effect Classification

ZIP reading is filesystem I/O — it requires the `FS` capability, same as `_fs_readFile`.

| Function | Effect | Rationale |
|----------|--------|-----------|
| `listEntries(path)` | `! {FS}` | Opens and reads ZIP file from disk |
| `readEntry(path, name)` | `! {FS}` | Opens ZIP and reads specific entry |
| `readEntryBytes(path, name)` | `! {FS}` | Opens ZIP and reads binary entry |

No new effect type needed — reuses existing `FS` effect and sandbox.

### API Design

#### std/zip Module

```ailang
module std/zip

-- List all file paths in a ZIP archive
-- Returns: Result[[string], string]
-- Example: listEntries("doc.docx") => Ok(["word/document.xml", "[Content_Types].xml", ...])
export func listEntries(path: string) -> Result[[string], string] ! {FS}

-- Read a text entry from a ZIP archive as a UTF-8 string
-- Returns: Result[string, string]
-- Example: readEntry("doc.docx", "word/document.xml") => Ok("<w:document>...")
export func readEntry(path: string, entryName: string) -> Result[string, string] ! {FS}

-- Read a binary entry from a ZIP archive as a base64-encoded string
-- Returns: Result[string, string]
-- Example: readEntryBytes("doc.docx", "word/media/image1.png") => Ok("iVBORw0KGgo...")
export func readEntryBytes(path: string, entryName: string) -> Result[string, string] ! {FS}
```

**Design decisions:**
- All functions return `Result[T, string]` for safe error handling (consistent with `std/json`)
- `readEntryBytes` returns base64 because AILANG strings are UTF-8; raw bytes would corrupt
- Each call opens the ZIP independently — no handle/session state needed for MVP
- Path argument is validated against `AILANG_FS_SANDBOX` (same as `_fs_readFile`)

### Implementation

#### New Files

| File | LOC (est.) | Purpose |
|------|-----------|---------|
| `internal/builtins/zip.go` | ~200 | Builtin registration + Go implementations |
| `internal/builtins/zip_test.go` | ~250 | Unit tests with test ZIP fixtures |
| `internal/effects/fs.go` (modify) | +15 | Register ZIP operations under FS effect |

#### Builtin Specifications

```go
// internal/builtins/zip.go

// _zip_listEntries: string -> Result[[string], string] ! {FS}
BuiltinSpec{
    Module: "std/zip", Name: "_zip_listEntries", NumArgs: 1,
    IsPure: false, Effect: "FS",
    Type: func() types.Type {
        T := types.NewBuilder()
        return T.Func(T.String()).Returns(
            T.App("Result", T.List(T.String()), T.String()),
        ).Effects("FS")
    },
}

// _zip_readEntry: string -> string -> Result[string, string] ! {FS}
BuiltinSpec{
    Module: "std/zip", Name: "_zip_readEntry", NumArgs: 2,
    IsPure: false, Effect: "FS",
    Type: func() types.Type {
        T := types.NewBuilder()
        return T.Func(T.String(), T.String()).Returns(
            T.App("Result", T.String(), T.String()),
        ).Effects("FS")
    },
}

// _zip_readEntryBytes: string -> string -> Result[string, string] ! {FS}
BuiltinSpec{
    Module: "std/zip", Name: "_zip_readEntryBytes", NumArgs: 2,
    IsPure: false, Effect: "FS",
    Type: func() types.Type {
        T := types.NewBuilder()
        return T.Func(T.String(), T.String()).Returns(
            T.App("Result", T.String(), T.String()),
        ).Effects("FS")
    },
}
```

#### Go Implementation (sketch)

```go
func zipListEntriesImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    path := args[0].(*eval.StringValue).Value

    r, err := zip.OpenReader(path)
    if err != nil {
        return makeErr(fmt.Sprintf("cannot open ZIP: %v", err)), nil
    }
    defer r.Close()

    entries := make([]eval.Value, len(r.File))
    for i, f := range r.File {
        entries[i] = &eval.StringValue{Value: f.Name}
    }

    return makeOk(&eval.ListValue{Elements: entries}), nil
}

func zipReadEntryImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    path := args[0].(*eval.StringValue).Value
    entryName := args[1].(*eval.StringValue).Value

    r, err := zip.OpenReader(path)
    if err != nil {
        return makeErr(fmt.Sprintf("cannot open ZIP: %v", err)), nil
    }
    defer r.Close()

    for _, f := range r.File {
        if f.Name == entryName {
            rc, err := f.Open()
            if err != nil {
                return makeErr(fmt.Sprintf("cannot read entry: %v", err)), nil
            }
            defer rc.Close()

            data, err := io.ReadAll(rc)
            if err != nil {
                return makeErr(fmt.Sprintf("read error: %v", err)), nil
            }
            return makeOk(&eval.StringValue{Value: string(data)}), nil
        }
    }

    return makeErr(fmt.Sprintf("entry not found: %s", entryName)), nil
}
```

### Security Considerations

1. **Sandbox enforcement**: All paths checked against `AILANG_FS_SANDBOX` before opening
2. **Zip bomb protection**: Limit decompressed entry size (default: 100MB per entry)
3. **Path traversal**: Entry names with `../` are rejected
4. **Resource limits**: Maximum 10,000 entries per archive (prevent DoS)

### Testing Strategy

1. **Unit tests**: Test ZIP fixture with known entries (create in test setup)
2. **Error paths**: Missing file, missing entry, corrupt ZIP, sandbox violation
3. **Security**: Zip bomb detection, path traversal rejection
4. **Integration**: Create a test DOCX-like ZIP and verify roundtrip

### Test Fixture

```go
// Create test ZIP in test setup
func createTestZip(t *testing.T, path string) {
    f, _ := os.Create(path)
    w := zip.NewWriter(f)

    entry, _ := w.Create("hello.txt")
    entry.Write([]byte("Hello, World!"))

    entry2, _ := w.Create("data/config.xml")
    entry2.Write([]byte("<config><name>test</name></config>"))

    w.Close()
    f.Close()
}
```

## Milestones

| # | Task | Est. |
|---|------|------|
| 1 | Register FS ops + builtin specs for 3 functions | 2h |
| 2 | Implement `_zip_listEntries` with tests | 2h |
| 3 | Implement `_zip_readEntry` + `_zip_readEntryBytes` with tests | 3h |
| 4 | Security hardening (sandbox, zip bombs, path traversal) | 2h |
| 5 | Integration test with DOCX-like fixture | 1h |
| 6 | Example file + documentation | 1h |

**Total: ~11 hours (1.5 days)**

## Alternatives Considered

### A: New `Zip` effect type
Rejected. ZIP reading is fundamentally filesystem I/O. Adding a separate effect would fragment the capability model without adding real security value. Users already need `FS` to read the ZIP file path.

### B: Handle-based API (open → read → close)
Rejected for MVP. Stateful handles add complexity (resource leaks, lifetime management) that AILANG's functional model handles poorly. Each function call reopens the ZIP — acceptable for document parsing where you read 5-20 entries. Consider handles in v0.8+ if performance profiling shows reopening is a bottleneck.

### C: Return raw bytes instead of base64
Rejected. AILANG strings are UTF-8. Returning raw bytes would corrupt binary content. Base64 is the standard interchange format, and `_bytes_from_base64` (std/bytes) can decode it when needed.

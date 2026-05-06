# std/deflate: raw zlib/deflate primitives for PDF FlateDecode and wire protocols

**Status**: IMPLEMENTED (2026-05-06)
**Target**: v0.16.0
**Priority**: P1 (Medium — unblocked ailang-parse PDF annotation extractor)
**Estimated**: 1 day (~6 hours) — actual ~2 hours
**Dependencies**: None (sits alongside existing std/gzip and std/zip)

**Source**: Feature request from `cli` (msg_20260506_074820_1f185b21) — blocking PDF annotation extraction in ailang-parse.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure functions; deflate/inflate are bit-exact deterministic |
| A2: Replayability | 0 | Pure functions emit no spans; no impact on traces |
| A3: Effect Legibility | +1 | Pure (no `! {…}`); contrasts cleanly with `decompressFile` which carries `! {FS}` |
| A4: Explicit Authority | +1 | No FS/Net/IO; takes a base64 string in, returns base64 string out |
| A5: Bounded Verification | +1 | Result[string, string] makes failure typed and locally checkable |
| A6: Safe Concurrency | 0 | Stateless; safe to call concurrently |
| A7: Machines First | +1 | Mirrors std/gzip shape exactly — agents can pattern-match the API |
| A8: Minimal Syntax | +1 | No new syntax; 4 builtins behind 4 stdlib wrappers |
| A9: Cost Visibility | +1 | 100MB output cap (matches std/gzip) prevents zip-bomb surprises |
| A10: Composability | +1 | Composes with std/bytes (base64) and std/zip (entry bytes are deflate streams) |
| A11: Structured Failure | +1 | All four functions return Result[string, string]; no panics |
| A12: System Boundary | +1 | base64 string in / base64 string out — boundary is explicit |

**Net Score: +10** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Deflate/inflate are deterministic by construction
- [x] A3 (Effects): All four functions are `pure func` (no effect row)
- [x] A4 (Authority): No ambient access; data is passed by value as base64
- [x] A7 (Machines First): API shape is a verbatim copy of std/gzip — minimal new surface for agents to learn

## Problem Statement

While building a PDF annotation extractor in **ailang-parse** (extracting highlights/comments without invoking the multimodal AI path), we hit a hard stdlib gap: **PDF object streams (`/ObjStm`) and compressed dictionary entries use FlateDecode, which is raw zlib (RFC 1950 — deflate body + 2-byte zlib header + adler32 trailer).**

Modern PDFs (PDF 1.5+, anything "optimized for web") bundle small objects — including annotations — into `/ObjStm`. Any AILANG module that wants to read PDF metadata without shelling out to an external tool needs raw inflate.

**Current State:**
- `std/zip` exposes ZIP-archive APIs only (`listEntries`, `readEntry`, `readEntryBytes`); no way to feed it a raw deflate or zlib byte string
- `std/gzip.decompress` requires the gzip wrapper (RFC 1952) — a different framing format, incompatible with raw zlib
- No `std/deflate` module exists
- ailang-parse is shipping a pure-string PDF annotation scanner that works on simple PDFs (Quartz `PDFContext` output, no `ObjStm`) but **silently misses annotations on any PDF that compresses objects** — the feature is fragile until this primitive lands

**Impact:**
- **ailang-parse** (`cli` agent) — blocks robust PDF annotation extraction; the workaround silently degrades on PDFs from non-Apple producers (most modern PDFs)
- **Future stdlib work** — any AILANG code that touches HTTP `Content-Encoding: deflate`, WebSocket `permessage-deflate`, PNG `IDAT` chunks, or custom binary formats with embedded zlib payloads
- **AI agents writing AILANG** — agents currently have no path from "I have a deflate-compressed byte string" to "I have the decompressed bytes" without reaching for a ZIP container they don't have

## Goals

**Primary Goal:** Expose raw deflate and zlib (RFC 1950) inflate/deflate primitives so AILANG modules can decompress PDF FlateDecode streams and other zlib-wrapped payloads without going through a ZIP archive container.

**Success Metrics:**
- ailang-parse PDF annotation extractor handles `/ObjStm`-bearing PDFs (PDF 1.5+) without shelling out
- Four new builtins (`_deflate_inflate`, `_deflate_inflateZlib`, `_deflate_deflate`, `_deflate_deflateZlib`) pass round-trip tests
- API mirrors std/gzip exactly (base64-in / base64-out, 100MB output cap, Result-typed failures)
- Single `examples/std_deflate_*.ail` runnable demonstrating PDF FlateDecode-style usage

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| New `std/deflate` module vs. extending `std/zip` | Once shipped, the import path is permanent — moving it is a breaking change | human | design | high |
| 100MB decompressed-size cap (matches std/gzip) | Defines the zip-bomb threshold; raising it later is easy, lowering it breaks callers | human | design | med |
| Compression level signature: `int` with `-1` = default (matches std/gzip) | Locked in by std/gzip parallel; diverging would surprise agents | agent | design | low |
| Base64 string boundary (no raw `bytes` type) | Consistent with std/gzip and std/zip.readEntryBytes | compiler | design | high (would touch all three modules) |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **Module placement**: `std/deflate` (new module) — confirmed 2026-05-06
- [x] **Output cap**: 100MB (matches std/gzip) — confirmed 2026-05-06

## Solution Design

### Overview

Add a new stdlib module `std/deflate` exposing four pure functions backed by Go's `compress/flate` and `compress/zlib` standard library. The boundary contract is identical to `std/gzip`: input is a base64-encoded byte string, output is `Result[string, string]` carrying base64 of the (de)compressed bytes or an error message.

**Why a new module rather than extending std/zip:**
- `std/zip` is the **archive container** abstraction (entries, central directory, file metadata)
- `std/gzip` is the **single-file wrapper** abstraction (gzip header + deflate body + CRC trailer)
- `std/deflate` is the **primitive compression algorithm** (raw deflate, optionally with the zlib header/trailer)

Bundling primitives into the archive module muddies the mental model and makes the docs harder for agents to navigate. Three small, single-purpose modules pattern-match better than one grab-bag.

### Architecture

**Components:**

1. **`std/deflate.ail`** — AILANG-side stdlib wrapper (~30 LOC, mirrors `std/gzip.ail`)
2. **`internal/builtins/deflate.go`** — Go-side builtins backed by `compress/flate` and `compress/zlib` (~250 LOC, mirrors `gzip.go` structure)
3. **`internal/builtins/deflate_test.go`** — Round-trip + boundary + error tests (~150 LOC)
4. **`examples/std_deflate_pdf_objstm.ail`** — Runnable example showing PDF FlateDecode-style decompression

**Boundary contract (identical to std/gzip):**
- Input: base64-encoded byte string
- Output: `Ok(base64 of result)` or `Err(message)`
- Output cap: 100MB decompressed (rejects with `Err` if exceeded)
- All four functions are `pure func` — no effect row

### Implementation Plan

**Phase 1: Builtins** (~3 hours)
- [ ] Create `internal/builtins/deflate.go` modeled on `gzip.go`
- [ ] Register `_deflate_inflate` (raw deflate, `flate.NewReader`)
- [ ] Register `_deflate_inflateZlib` (zlib-wrapped, `zlib.NewReader`)
- [ ] Register `_deflate_deflate(level)` (raw deflate, `flate.NewWriter`)
- [ ] Register `_deflate_deflateZlib(level)` (zlib-wrapped, `zlib.NewWriterLevel`)
- [ ] Apply 100MB output cap via `io.LimitReader`-style guard (mirror gzip.go)

**Phase 2: Stdlib wrapper + types** (~1 hour)
- [ ] Create `std/deflate.ail` with four exported `pure func` declarations
- [ ] Wire into stdlib loader (search for how `std/gzip` is registered — likely zero changes if discovery is glob-based)

**Phase 3: Tests + example** (~2 hours)
- [ ] Round-trip tests for all four functions (deflate→inflate, deflateZlib→inflateZlib)
- [ ] Cross-verification: `std/gzip.compress` body (strip 10-byte header + 8-byte trailer) round-trips through `inflate`
- [ ] Boundary tests: empty input, single-byte input, malformed deflate stream
- [ ] Output-cap test: synthesize a stream that decompresses past 100MB → expect `Err`
- [ ] Compression-level test: `deflate(input, 9)` produces ≤ `deflate(input, 1)` size for non-trivial input
- [ ] `examples/std_deflate_pdf_objstm.ail` — decode a hand-rolled FlateDecode payload representing a PDF object stream

### Files to Modify/Create

**New files:**
- `internal/builtins/deflate.go` — Four builtin registrations, ~250 LOC
- `internal/builtins/deflate_test.go` — Unit tests, ~150 LOC
- `std/deflate.ail` — Stdlib wrapper, ~30 LOC
- `examples/std_deflate_pdf_objstm.ail` — Runnable example, ~30 LOC

**Modified files:**
- `internal/builtins/init.go` (or wherever `RegisterAll` lives) — call `registerDeflateInflate()` etc. (~6 LOC)
- `CHANGELOG.md` — entry under v0.16.0
- `prompts/v0.16.md` (or current teaching prompt) — add a one-line "use std/deflate for raw zlib" note alongside std/gzip

## Examples

### Example 1: PDF FlateDecode object stream

**Before** (today, ailang-parse workaround):
```ailang
-- Pure-string PDF annotation scanner — silently misses /ObjStm-compressed annotations
func extractAnnotations(pdfText: string) -> List[string] = ...
```

**After** (with std/deflate):
```ailang
module benchmark/solution
import std/deflate (inflateZlib)
import std/bytes (from_base64, to_utf8)
import std/result (Result, Ok, Err)

-- Decode a PDF FlateDecode object stream (base64 of the raw zlib bytes)
func decodeObjStm(b64: string) -> Result[string, string] =
  match inflateZlib(b64) {
    Ok(decompressedB64) => match from_base64(decompressedB64) {
      Ok(bytes) => to_utf8(bytes),
      Err(e)    => Err("from_base64: " ++ e)
    },
    Err(msg) => Err("inflate: " ++ msg)
  }

export func main() -> () ! {IO} = ...
```

### Example 2: Round-trip a deflate stream

```ailang
import std/deflate (deflate, inflate)
import std/result (Result, Ok)

-- compressed.size() < raw.size() for compressible input
let raw      = "..."  -- base64 of some text
let comp     = deflate(raw, 6)
let restored = match comp { Ok(c) => inflate(c), Err(e) => Err(e) }
```

### Example 3: HTTP Content-Encoding: deflate (future use case)

```ailang
-- HTTP servers send "Content-Encoding: deflate" as raw zlib (RFC 1950)
-- regardless of the misleading name. Use inflateZlib, not inflate.
match inflateZlib(httpBodyB64) {
  Ok(decoded) => ...,
  Err(msg)    => log("deflate decode failed: " ++ msg)
}
```

## Success Criteria

- [ ] Four builtins registered and discoverable via `ailang prompt`
- [ ] `std/deflate.ail` wrapper exports `inflate`, `inflateZlib`, `deflate`, `deflateZlib`
- [ ] Round-trip tests pass for all four functions
- [ ] Cross-verification test: gzip body (header/trailer stripped) inflates correctly via raw `inflate`
- [ ] 100MB output cap enforced and tested with synthesized over-cap stream
- [ ] `examples/std_deflate_pdf_objstm.ail` runs to completion via `make verify-examples`
- [ ] ailang-parse PDF annotation extractor migrates to `inflateZlib` and handles `/ObjStm` PDFs (verified out-of-tree by `cli` agent)
- [ ] All tests passing (`make test`)
- [ ] CHANGELOG.md updated
- [ ] Doc moved to `design_docs/implemented/v0_16_x/`

## Testing Strategy

**Unit tests** (`internal/builtins/deflate_test.go`):
- Round-trip: `inflate(deflate(x, level))` == x for level ∈ {0, 1, 6, 9, -1}
- Round-trip: `inflateZlib(deflateZlib(x, level))` == x
- Cross-format negative: `inflate(deflateZlib(x, 6))` returns `Err` (header bytes confuse raw deflate)
- Boundary: empty string, single null byte, 1MB random bytes
- Malformed input: random base64 garbage → `Err` (no panic)
- Output cap: hand-built deflate stream that expands past 100MB → `Err`

**Integration tests:**
- Pull a small real-world FlateDecode payload extracted from a PDF (commit a fixture under `internal/builtins/testdata/pdf_objstm.b64`)
- Verify `inflateZlib` on the fixture matches the expected plaintext byte-for-byte

**Manual testing:**
- ailang-parse migrates and confirms a PDF that previously hid annotations now exposes them
- `ailang run examples/std_deflate_pdf_objstm.ail` produces expected output

## Deferred Decisions

The following are intentionally left open for the implementer:

- **File-based variants** (`inflateFile`, `deflateFile`) — agent may add if symmetry with `std/gzip.decompressFile` is preferred; not strictly needed for the PDF use case since PDF readers already have the bytes in memory
- **Streaming API** (incremental inflate for large payloads) — agent may defer; the 100MB cap makes one-shot acceptable for the v0.16.0 use cases
- **Default compression level constant** — agent may export `defaultLevel = -1` as a named constant or just document the magic number; no preference

## Non-Goals

- **bzip2, lzma, brotli, zstd** — out of scope; deflate/zlib is the immediate need (PDF, HTTP, PNG, WebSocket all use deflate)
- **Streaming inflate** — single-shot only for v0.16.0; revisit if real workloads exceed the 100MB cap
- **Native `bytes` type** — not in this sprint; sticking with the established base64-string boundary used by std/gzip and std/zip
- **Replacing std/gzip or std/zip internals with std/deflate calls** — internal refactor, not behavior-visible; can happen later if maintenance warrants it

## Timeline

**Single sprint** (~1 day, 6 hours):
- Hours 0–3: Phase 1 (builtins + registration)
- Hours 3–4: Phase 2 (stdlib wrapper)
- Hours 4–6: Phase 3 (tests, example, CHANGELOG)

**Total: ~6 hours, single session**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Naming clash: `deflate` is both the algorithm and one of the four function names | Low (docs confusion only) | Module doc-comment explicitly explains: `inflate`/`deflate` = raw RFC 1951; `inflateZlib`/`deflateZlib` = RFC 1950 wrapped |
| Agent confusion: writes `inflate` when they need `inflateZlib` (or vice versa) | Med (silent garbage output) | All four functions return `Err` on format mismatch — header byte validation catches the wrong-variant case immediately |
| 100MB cap rejects legitimate large PDF object streams | Low | Cap matches std/gzip precedent; if a real PDF exceeds it we revisit, but 100MB is already 100x the largest realistic ObjStm |
| Stdlib loader doesn't auto-discover the new module | Low | Mirror exactly how `std/gzip` was added (recent precedent in v0.12.0); same registration path |
| Adler-32 checksum mismatch on `inflateZlib` produces cryptic error | Low | Wrap Go's error with the message `"zlib: checksum mismatch — input may be raw deflate (try inflate instead)"` |

## Related Documents

<!-- Auto-populated by Ollama neural search on "std deflate zlib" -->

**Implemented (informs design — same boundary patterns):**
- [design_docs/implemented/v0_7_3/m-stdlib-zip.md](../../implemented/v0_7_3/m-stdlib-zip.md) (0.47) — original `std/zip` archive API; established the base64-string boundary
- [design_docs/implemented/v0_12_0/m-stdlib-tar-gzip.md](../../implemented/v0_12_0/m-stdlib-tar-gzip.md) (0.46) — closest sibling; this doc deliberately mirrors its module shape
- [design_docs/implemented/v0_11_0/m-streaming-zip-xml.md](../../implemented/v0_11_0/m-streaming-zip-xml.md) (0.43) — streaming patterns; informs the deferred streaming decision

**Planned (no overlap, listed for completeness):**
- [design_docs/planned/v1_0_0/m-perf4-bytecode-interpreter.md](../../planned/v1_0_0/m-perf4-bytecode-interpreter.md) (0.34)
- [design_docs/planned/v0_13_0/m-error-propagation.md](../../planned/v0_13_0/m-error-propagation.md) (0.33)
- [design_docs/planned/v0_13_0/m-eval-bounded-pipeline.md](../../planned/v0_13_0/m-eval-bounded-pipeline.md) (0.31)

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- **Source request**: `ailang messages read 1f185b21-82d6-4f44-81b1-cac772df1de3`
- **RFC 1950**: ZLIB Compressed Data Format — https://www.rfc-editor.org/rfc/rfc1950
- **RFC 1951**: DEFLATE Compressed Data Format — https://www.rfc-editor.org/rfc/rfc1951
- **PDF 1.5 spec, §7.5.7**: Object Streams — explains `/ObjStm` and FlateDecode usage
- **Go stdlib**: `compress/flate` (raw deflate), `compress/zlib` (RFC 1950 wrapper)
- **Existing implementations**: `internal/builtins/gzip.go`, `internal/builtins/zip_xml.go`, `std/gzip.ail`

## Future Work

- **Streaming inflate/deflate**: incremental processing for payloads > 100MB; pairs with a future native `bytes` type
- **brotli/zstd primitives**: same module shape, separate modules (`std/brotli`, `std/zstd`) once a real use case emerges
- **PDF stdlib module**: with raw inflate available, a `std/pdf` module covering metadata extraction (annotations, outlines, form fields) becomes feasible without multimodal AI

---

**Document created**: 2026-05-06
**Last updated**: 2026-05-06 (moved to implemented)

---

## Implementation Report (2026-05-06)

### What was built

All three milestones shipped in a single ~2-hour session against the 1-day plan estimate:

| Milestone | Estimated LOC | Actual LOC | Status |
|-----------|---------------|------------|--------|
| M1: Builtins + tests | 400 | ~700 (363 impl + 340 tests) | ✅ |
| M2: Stdlib wrapper | 30 | ~60 | ✅ |
| M3: Example + CHANGELOG + doc move | 50 | ~70 | ✅ |
| **Total** | **460** | **~830** | ✅ |

The 80% LOC overage came entirely from richer test coverage than the original sketch — see "Test coverage" below. The architecture itself was a clean mirror of the v0.12.0 `std/gzip` template.

### Files created

- [`internal/builtins/deflate.go`](../../../internal/builtins/deflate.go) — 4 builtins backed by `compress/flate` and `compress/zlib`, 100MB output cap via `io.LimitReader`
- [`internal/builtins/deflate_test.go`](../../../internal/builtins/deflate_test.go) — 13 test functions
- [`std/deflate.ail`](../../../std/deflate.ail) — Stdlib wrapper exporting 4 `pure func`
- [`examples/runnable/std_deflate_pdf_objstm.ail`](../../../examples/runnable/std_deflate_pdf_objstm.ail) — Worked PDF FlateDecode example

### Files modified

- [`internal/pipeline/testdata/builtin_types.golden`](../../../internal/pipeline/testdata/builtin_types.golden) — Regenerated for 4 new signatures (288 total)
- [`changelogs/v0.10-current.md`](../../../changelogs/v0.10-current.md) — Added M-STD-DEFLATE-ZLIB entry under [Unreleased]
- This design doc + sprint plan moved from `design_docs/planned/v0_16_0/` → `design_docs/implemented/v0_16_x/`

### Test coverage

13 test functions covering:

- **Round-trip × 2 variants × 5 levels** (`TestDeflateRawRoundTrip`, `TestDeflateZlibRoundTrip`) — empty input, small ASCII, repeated text, levels 0/1/6/9/-1, unicode
- **Cross-format negative** (`TestDeflate_CrossFormat_ZlibToRawInflate`, `TestDeflate_CrossFormat_RawToZlibInflate`) — confirms zlib-wrapped data fed to raw `inflate` does not silently round-trip; raw deflate fed to `inflateZlib` errors with a hint to try `inflate`
- **Malformed input** (4 tests) — invalid base64, non-zlib bytes, bad compression levels (-2, 10, 99)
- **100MB output cap** (`TestDeflateInflate_BombRejected`, `TestDeflateInflateZlib_BombRejected`) — synthesizes 150MB-of-zeros stream that compresses to KB; verifies cap rejection
- **Level monotonicity** (`TestDeflate_LevelMonotonicity`) — guards against silent ignoring of the `level` argument
- **Determinism** (`TestDeflate_Deterministic`) — 20-iteration loop per pure variant
- **RFC 1951 wire compatibility** (`TestDeflate_GzipBodyCrossDecode`) — strips 10-byte gzip header + 8-byte trailer, confirms the deflate body decompresses via raw `inflate`. Proves we're genuinely RFC 1951 compatible, not just self-consistent.

All 18 sub-tests pass; `make test` and `make lint` clean.

### Deviations from plan

- **No deviations from API or architecture.** Both design-freeze items resolved before implementation: new `std/deflate` module, 100MB cap.
- **Test scope expanded**: added the gzip-body cross-decode test (M1 acceptance criteria did not require this, but it's a load-bearing wire-compat proof — without it, `inflate` could be self-consistent but incompatible with real-world deflate streams).
- **Stdlib loader**: zero changes needed. The existing `//go:embed *.ail` glob in [`std/embed.go`](../../../std/embed.go) picked up `std/deflate.ail` automatically.

### External validation

- **GitHub issue #223** (linked to this sprint) can be closed once a release containing this work ships
- **ailang-parse migration** (out-of-tree, post-merge): `cli` agent to replace its pure-string PDF annotation scanner with `inflateZlib`-based decoder; will close the silent-degradation-on-`/ObjStm` gap that motivated the request

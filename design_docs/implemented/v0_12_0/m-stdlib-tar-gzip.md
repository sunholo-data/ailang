# M-STDLIB-TAR-GZIP: std/tar + std/gzip Native Archive Support

**Status**: IMPLEMENTED
**Target**: v0.12.0
**Priority**: P2 (Low — workarounds exist in AILANG Parse)
**Estimated**: 3 days
**Dependencies**: std/zip (v0.7.3, pattern reference only)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure native Go decoders; no subprocess env leakage (vs. shelling out to `tar`/`gzip`) |
| A2: Replayability | 0 | No new trace surface; FS effect already traced |
| A3: Effect Legibility | +1 | FS effect declared on every entry point; no ambient IO |
| A4: Explicit Authority | +1 | All read/write goes through FS capability + AILANG_FS_SANDBOX; tarbomb (`../`) rejected at stdlib layer |
| A5: Bounded Verification | +1 | Result[…, string] signatures — type checker verifies error handling at call sites |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Stable typed API beats parsing `tar` stderr; uniform composition with std/zip |
| A8: Minimal Syntax | +1 | Reuses existing record types and Result; no new syntax |
| A9: Cost Visibility | +1 | Document size/entry caps up front (mirror std/zip: 10K entries, 100MB decompressed) |
| A10: Composability | +1 | format_router.ail can handle zip-office, zip-odf, tar-gz uniformly |
| A11: Structured Failure | +1 | Typed Result[_, string] errors, no panics |
| A12: System Boundary | +1 | Every FS crossing gated by capability + sandbox |

**Net Score: +10** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Pure Go stdlib (`archive/tar`, `compress/gzip`), no nondeterminism
- [x] A3 (Effects): FS effect declared on all entry points that touch disk
- [x] A4 (Authority): No ambient access; tarbomb defence enforced before write
- [x] A7 (Machines First): API shape mirrors std/zip — one pattern for AI/humans to learn

## Problem Statement

AILANG Parse (v0.15.0) landed LaTeX/arXiv parsing. The natural next input is **arXiv source bundles**, which ship as `.tar.gz`. Current workaround: the user or the arxivbench corpus downloader extracts manually (Python-side) and points the parser at the inner `main.tex`.

**Current State:**
- std/zip handles 7 of 15 supported input formats in AILANG Parse (DOCX, PPTX, XLSX, ODT, ODP, ODS, EPUB)
- No native tar or gzip support → arXiv bundles require out-of-band extraction
- Shelling out (`tar -xzf`) breaks the no-shell-out story: Windows-flaky, subprocess env leakage, can't enforce path-traversal sandboxing, exit codes don't compose with `Result[_, _]`

**Impact:**
- AILANG Parse users working with arXiv corpora hit a papercut — the whole point of arXiv bulk access is "one .tar.gz per paper"
- Adding tar.gz completes the "no shell-out for archives" story started by std/zip
- Low-severity: workarounds exist (Python pre-extraction, `std/process` shell-out)

## Goals

**Primary Goal:** Provide native std/gzip and std/tar stdlib modules, modelled on std/zip, so AILANG Parse (and other consumers) can read .tar and .tar.gz archives without shelling out.

**Success Metrics:**
- arxivbench parses a `.tar.gz` source bundle end-to-end via AILANG stdlib (no Python, no `std/process`)
- format_router.ail treats `zip-*` and `tar-gz` uniformly (single dispatch pattern)
- Tarbomb payload (entry named `../../etc/passwd`) rejected with `Err("path traversal rejected: ../../etc/passwd")` — verified by unit test
- Decompressed size cap enforced (100MB, mirror std/zip) — gzip bomb rejected
- Zero new system dependencies; builds on Go stdlib `archive/tar` + `compress/gzip`

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Module split: separate `std/gzip` + `std/tar` vs. combined `std/archive` | Affects every consumer's import line; precedent sets naming for future `std/bzip2`, `std/zstd` | human | design | med |
| `readFromGzip(path, entry)` convenience: include in v1 or defer | This is the ONLY shape that matters for arxivbench (per requester); omitting it means temp-file workaround | human | design | low |
| Binary entry encoding: base64 string (match std/zip) vs. raw bytes type | Consistency with std/zip vs. efficiency for large binary entries | human | design | med |
| Size/entry caps: identical to std/zip (10K entries, 100MB decompressed) or different for tar's streaming nature | tar is typically used for larger bundles than zip-office; too-low caps break arxiv use case | human | design | low |
| Gzip `compress(level)` API: include writer in v1 or read-only first | AILANG Parse only needs read; write adds scope but completes the symmetry with std/zip `createArchive` | agent (default: read-only v1) | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Module split**: proposal = separate `std/gzip` and `std/tar` (mirrors Go stdlib, composable, matches requester's sketch)
- [ ] **`readFromGzip` convenience**: proposal = **include in v1** (requester says it's the only shape that matters)
- [ ] **Binary encoding**: proposal = base64 string, same as `std/zip.readEntryBytes` (consistency > efficiency for v1)
- [ ] **Caps**: proposal = 10K entries / 100MB decompressed (identical to std/zip); revisit if arxiv bundles exceed

## Solution Design

### Overview

Two new stdlib modules, each backed by Go-side builtins in `internal/builtins/`:

- **std/gzip** — thin wrapper around `compress/gzip`: pure `decompress`/`compress` on strings + file-based `decompressFile`
- **std/tar** — thin wrapper around `archive/tar`: `listEntries`/`readEntry`/`readEntryBytes`/`extractAll` + the composed `readFromGzip` convenience

Security model mirrors std/zip exactly: FS capability gating, `AILANG_FS_SANDBOX` respect, `../` rejection, 10K entry cap, 100MB decompressed size cap.

### Architecture

**Components:**
1. **`std/gzip.ail`** (~30 LOC) — AILANG surface, `_gzip_*` builtin dispatch
2. **`std/tar.ail`** (~50 LOC) — AILANG surface, `_tar_*` builtin dispatch
3. **`internal/builtins/gzip.go`** (~150 LOC) — gzip read/write + decompressFile, bounded via `io.LimitReader`
4. **`internal/builtins/tar.go`** (~300 LOC) — tar reader, entry listing, extractAll with tarbomb defence (`filepath.IsLocal` + join-and-verify)
5. **Test fixtures** — small `.tar`, `.tar.gz`, a crafted tarbomb, a gzip bomb

### AILANG Surface

```ailang
-- std/gzip
export func decompress(input: string) -> Result[string, string]
export func compress(input: string, level: int) -> Result[string, string]
export func decompressFile(path: string) -> Result[string, string] ! {FS}

-- std/tar
type TarEntry = {name: string, size: int, isDir: bool}
export func listEntries(path: string) -> Result[List[TarEntry], string] ! {FS}
export func readEntry(path: string, entry: string) -> Result[string, string] ! {FS}
export func readEntryBytes(path: string, entry: string) -> Result[string, string] ! {FS}
export func extractAll(path: string, destDir: string) -> Result[List[string], string] ! {FS}
export func readFromGzip(path: string, entry: string) -> Result[string, string] ! {FS}
```

### Implementation Plan

**Phase 1: std/gzip** (~4 hours)
- [ ] `internal/builtins/gzip.go`: `_gzip_decompress`, `_gzip_compress`, `_gzip_decompressFile`
- [ ] `std/gzip.ail` surface
- [ ] Unit tests: round-trip, empty input, gzip bomb (enforce 100MB decompressed cap via `io.LimitReader`)
- [ ] Golden test: compress at level 6, verify byte-stable output

**Phase 2: std/tar (read)** (~8 hours)
- [ ] `internal/builtins/tar.go`: `_tar_listEntries`, `_tar_readEntry`, `_tar_readEntryBytes`
- [ ] Tarbomb defence utility (reject `..`, absolute paths, symlinks pointing outside destDir)
- [ ] `std/tar.ail` surface (read-only subset)
- [ ] Unit tests: normal tar, tar with directories, empty tar, 10K entry cap

**Phase 3: std/tar (extractAll + gzip composition)** (~6 hours)
- [ ] `_tar_extractAll` with sandboxed writes, returns list of written paths
- [ ] `_tar_readFromGzip` — stream-decompress into in-memory tar reader, extract one entry without temp file
- [ ] Integration test: arxiv-style bundle (`paper.tar.gz` containing `main.tex`, `refs.bib`, figures)
- [ ] Tarbomb regression test: `evil.tar` with `../../etc/passwd` entry → `Err`

**Phase 4: Docs + example** (~2 hours)
- [ ] `examples/runnable/tar_gzip_reader.ail` (arxiv bundle walkthrough)
- [ ] CHANGELOG entry
- [ ] Update AILANG Parse integration note (optional: ack message to cli agent)

### Files to Modify/Create

**New files:**
- `std/gzip.ail` (~30 LOC)
- `std/tar.ail` (~50 LOC)
- `internal/builtins/gzip.go` (~150 LOC)
- `internal/builtins/tar.go` (~300 LOC)
- `internal/builtins/tar_test.go` (~200 LOC)
- `internal/builtins/gzip_test.go` (~120 LOC)
- `examples/runnable/tar_gzip_reader.ail` (~40 LOC)
- Test fixtures: `internal/builtins/testdata/sample.tar`, `sample.tar.gz`, `evil-tarbomb.tar`

**Modified files:**
- `internal/builtins/init.go` or equivalent — wire new `init()` registrations (~4 LOC)
- `CHANGELOG.md` — v0.12.0 entry

## Examples

### Example 1: arXiv source bundle (the motivating use case)

**Before (current workaround):**
```bash
# Python-side, out of band:
tar -xzf 2401.12345.tar.gz -C /tmp/paper/
ailang run parse-latex.ail -- /tmp/paper/main.tex
```

**After:**
```ailang
import std/tar (readFromGzip)
import std/result (Result)

export func main() -> () ! {FS, IO} {
  match readFromGzip("2401.12345.tar.gz", "main.tex") {
    Ok(tex) => println(parse_latex(tex)),
    Err(msg) => println("failed: " ++ msg)
  }
}
```

### Example 2: Tarbomb defence

**Input:** `evil.tar` containing one entry named `../../../etc/passwd`

```ailang
extractAll("evil.tar", "/tmp/sandbox")
-- => Err("path traversal rejected: ../../../etc/passwd")
```

### Example 3: Uniform format dispatch

```ailang
-- format_router.ail — one pattern for all archive-backed formats
match classify(path) {
  ZipOffice(_)  => handle_zip(path),
  ZipOdf(_)     => handle_zip(path),
  TarGz(_)      => handle_tar_gz(path),  -- now native!
  _             => read_plain(path)
}
```

## Success Criteria

- [ ] `std/gzip.decompress`/`compress`/`decompressFile` pass unit tests (round-trip + gzip bomb)
- [ ] `std/tar.listEntries`/`readEntry`/`readEntryBytes` pass unit tests
- [ ] `std/tar.extractAll` rejects tarbomb payload with typed `Err`
- [ ] `std/tar.readFromGzip` extracts a single entry from `.tar.gz` without temp file
- [ ] `examples/runnable/tar_gzip_reader.ail` runs green in `make verify-examples`
- [ ] AILANG Parse can swap its Python pre-extraction for native stdlib (adapter code from requester)
- [ ] All tests passing (`make test`)
- [ ] CHANGELOG.md + design doc moved to `implemented/v0_12_0/`
- [ ] `std/tar.ail` and `std/gzip.ail` documented in the prompts file

## Testing Strategy

**Unit tests:**
- gzip round-trip, level boundaries (0, 6, 9), empty input, gzip bomb (>100MB decompressed → Err)
- tar listEntries with directories + regular files + symlinks, 10K+ entries → Err
- tar readEntry on missing entry → Err, on binary entry via readEntryBytes
- tarbomb regression: `../`, absolute path, symlink-escape
- extractAll returns list of written paths in entry order

**Integration tests:**
- `readFromGzip` on a real arxiv bundle fixture (~50KB)
- Capability check: running without `--caps FS` → Err (effect rejection)
- Sandbox check: `AILANG_FS_SANDBOX=/tmp/safe` blocks writes outside sandbox

**Manual testing:**
- Build AILANG Parse with `std/tar` backend, run arxivbench on a 10-paper corpus
- Cross-platform smoke: tar.gz produced on macOS readable on Linux builds

## Deferred Decisions

- **Streaming tar fold** (analogous to `std/zip.scanFold`) — defer to v0.13+ if consumers ask. AILANG Parse's arxiv use case is single-entry lookups, not tag-matching scans.
- **Compression level default** — agent may choose (proposal: 6, matching gzip default)
- **Error message wording** — agent may choose, as long as tarbomb errors include the offending path for debuggability
- **bzip2/zstd support** — explicitly deferred; this doc covers gzip only

## Non-Goals

- **tar writing (createArchive)** — out of scope for v1. AILANG Parse only needs read. Can be added later mirroring `std/zip.createArchive`.
- **Streaming XML fold on tar** (analogous to `std/zip.scanFold`) — no consumer has asked; tar entries tend to be whole documents, not XML shards
- **Other compression codecs** (bzip2, xz, zstd, lz4) — gzip covers >95% of `.tar.*` in the wild
- **Incremental/append-mode tar writes** — not in any roadmap

## Timeline

**Week 1** (20 hours):
- Phase 1: std/gzip (4h)
- Phase 2: std/tar read (8h)
- Phase 3: extractAll + readFromGzip (6h)
- Phase 4: docs + example (2h)

**Total: ~20 hours across 1 week**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Tarbomb defence has subtle edge case (e.g., symlink-to-symlink escape) | High | Adopt Go 1.20+ `filepath.IsLocal` + after-join realpath check; port std/zip's defence suite |
| Gzip bomb DoS (100:1 compression ratio) | High | Bound decompressed reads with `io.LimitReader` at 100MB (match std/zip) |
| tar format variants (GNU, POSIX, old-ustar) misparsed | Med | Rely on Go stdlib `archive/tar` — already handles major variants |
| Large arxiv bundles exceed 100MB cap | Low | Raise cap per-call via env var if users hit it; revisit in v0.13 |
| API drift from std/zip pattern | Low | Code review checkpoint: diff std/tar.ail against std/zip.ail for naming consistency |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_7_3/m-stdlib-xml.md](/Users/mark/dev/sunholo/ailang/design_docs/implemented/v0_7_3/m-stdlib-xml.md) — stdlib module pattern reference
- [design_docs/implemented/v0_7_4/m-stdlib-gaps.md](/Users/mark/dev/sunholo/ailang/design_docs/implemented/v0_7_4/m-stdlib-gaps.md) (neural 0.40)
- [design_docs/implemented/v0_11_0/m-streaming-zip-xml.md](/Users/mark/dev/sunholo/ailang/design_docs/implemented/v0_11_0/m-streaming-zip-xml.md) — std/zip scanFold (future model for std/tar streaming)
- [design_docs/implemented/v0_9_2/m-docparse-dx.md](/Users/mark/dev/sunholo/ailang/design_docs/implemented/v0_9_2/m-docparse-dx.md) — std/zip write APIs (createArchive pattern)

**Planned (check for overlap):**
- None — no active doc proposes tar/gzip support

## References

- [Design Axioms](/docs/references/axioms)
- Requester message: `ailang messages read fb09cfae-d288-4139-9dad-e0e35e12ebe7` (cli / AILANG Parse v0.15.0)
- Existing pattern: [std/zip.ail](/Users/mark/dev/sunholo/ailang/std/zip.ail), [internal/builtins/zip.go](/Users/mark/dev/sunholo/ailang/internal/builtins/zip.go)
- Go stdlib: `archive/tar`, `compress/gzip`

## Future Work

- `std/tar.scanFold` — streaming fold over tar entries (analogous to std/zip scanFold), if a consumer requests it
- `std/tar.createArchive` — tar writing, mirroring `std/zip.createArchive`
- `std/bzip2`, `std/zstd` — additional codecs, composable with std/tar via a `readFromCodec(path, entry, codec)` pattern
- Generalised `std/archive` facade that dispatches zip/tar/tar.gz uniformly (only if call sites actually want this — format_router.ail currently prefers explicit dispatch)

---

**Document created**: 2026-04-14
**Last updated**: 2026-04-14
**Source request**: AILANG message `fb09cfae-d288-4139-9dad-e0e35e12ebe7` from cli / AILANG Parse v0.15.0

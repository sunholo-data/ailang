# M-STDLIB-TAR-GZIP Sprint Plan

**Sprint ID**: M-STDLIB-TAR-GZIP
**Design doc**: [m-stdlib-tar-gzip.md](m-stdlib-tar-gzip.md)
**Target**: v0.12.0
**Duration**: 4 days (~22 hours)
**Risk**: Low
**Source**: AILANG Parse v0.15.0 request (msg `fb09cfae`)

## Goal

Ship native `std/gzip` and `std/tar` stdlib modules — modelled on `std/zip` — so AILANG Parse can read arXiv `.tar.gz` source bundles without shelling out or Python pre-extraction.

## Design Freeze (resolved with proposed defaults)

- [x] **Module split**: two separate modules — `std/gzip` and `std/tar`
- [x] **`readFromGzip(path, entry)` convenience**: included in v1 (the shape arxivbench actually needs)
- [x] **Binary encoding**: base64-encoded string (consistency with `std/zip.readEntryBytes`)
- [x] **Caps**: 10K entries, 100MB decompressed (identical to std/zip)
- [x] **Gzip write API**: include `compress(input, level)` in v1 for API symmetry with zip
- [x] **tar write API (createArchive)**: NOT in v1 — read-only tar is sufficient for AILANG Parse

## Velocity Calibration

Reference data from existing std/zip (v0.7.3 + v0.9.2):
- `internal/builtins/zip.go` = 576 LOC for 5 builtins (~115 LOC/builtin avg)
- `std/zip.ail` = 80 LOC surface
- This sprint: 3 gzip builtins + 5 tar builtins = 8 total → estimate ~700–900 LOC Go + ~80 LOC .ail

## Milestones

### M1: std/gzip (Day 1, ~4h, ~280 LOC)

**Deliverables:**
- `internal/builtins/gzip.go` — registers `_gzip_decompress`, `_gzip_compress`, `_gzip_decompressFile`
- `internal/builtins/gzip_test.go` — round-trip, empty input, gzip bomb defence
- `std/gzip.ail` — public surface (~25 LOC)

**Acceptance:**
- `make test` green for `internal/builtins` package
- gzip bomb (>100MB decompressed from tiny input) rejected with typed `Err`
- Compress round-trip: `decompress(compress(x, 6)) == Ok(x)` for text + base64 binary

**Implementation notes:**
- Use `compress/gzip` from Go stdlib
- Guard `decompress` with `io.LimitReader` at `zipMaxDecompressedSize` (reuse constant from zip.go → promote to shared pkg const)
- `compress` returns raw bytes encoded as base64 (match `std/zip.readEntryBytes` convention) — call this out in docstring

---

### M2: std/tar read (Day 2, ~7h, ~350 LOC)

**Deliverables:**
- `internal/builtins/tar.go` — registers `_tar_listEntries`, `_tar_readEntry`, `_tar_readEntryBytes`
- `internal/builtins/tar_test.go` — list/read fixtures, entry cap, unicode names
- `std/tar.ail` — surface with `TarEntry` record type + 3 exports (~30 LOC)
- Test fixtures: `testdata/sample.tar` (3 entries: file, dir, nested file)

**Acceptance:**
- `listEntries(path)` returns `Ok([{name, size, isDir}])` matching fixture
- `readEntry` on missing entry → typed `Err`
- 10K+ entry tar → `Err("too many entries")`
- Unicode entry names round-trip correctly
- FS capability required — running without `--caps FS` rejects

**Implementation notes:**
- Use `archive/tar` from Go stdlib
- Model on `zipListEntries`/`zipReadEntry` in `internal/builtins/zip.go`
- Build `TarEntry` as `eval.RecordValue` — see existing record construction in zip.go

---

### M3: extractAll + readFromGzip (Day 3, ~7h, ~280 LOC)

**Deliverables:**
- `_tar_extractAll` with **tarbomb defence** (reject `..`, absolute, symlink escape)
- `_tar_readFromGzip` — compose gzip+tar without temp file (stream decompress → in-memory tar reader)
- `tar_test.go` additions: tarbomb rejection, symlink escape, `.tar.gz` extraction
- Test fixture: `testdata/evil-tarbomb.tar` with `../../etc/passwd` entry
- Test fixture: `testdata/sample.tar.gz` (e.g., fake arxiv bundle: main.tex + refs.bib)

**Acceptance:**
- Tarbomb payload → `Err` containing the offending path (for debuggability)
- Symlink pointing outside destDir → `Err`
- `readFromGzip(path, entry)` extracts single entry from .tar.gz without creating temp files
- `extractAll` returns ordered list of written absolute paths
- `AILANG_FS_SANDBOX=/tmp/safe` blocks extraction outside sandbox

**Implementation notes:**
- Tarbomb defence: use `filepath.IsLocal` (Go 1.20+) + post-join realpath check
- `readFromGzip`: `os.Open` → `gzip.NewReader` → `tar.NewReader`, seek to entry, read
- Reuse sandbox check helper from existing FS builtins (grep for `AILANG_FS_SANDBOX`)

---

### M4: Docs, example, release prep (Day 4, ~4h, ~80 LOC)

**Deliverables:**
- `examples/runnable/tar_gzip_reader.ail` — arxiv-style bundle walkthrough (~40 LOC)
- CHANGELOG.md entry under v0.12.0
- `prompts/` update — document new stdlib modules in active teaching prompt
- Update `docs/docs/reference/stdlib-reference.md` (if exists) or equivalent guide
- Move design doc `planned/v0_12_0/` → `implemented/v0_12_0/` (at release time, not now)

**Acceptance:**
- `make verify-examples` passes on new example
- `make ci` green (lint + test + verify-examples + check-file-sizes)
- Example demonstrates all 3 key APIs: `listEntries`, `readFromGzip`, tarbomb rejection
- Teaching prompt mentions std/tar + std/gzip

## Day-by-Day Task List

**Day 1 (Mon)** — M1: std/gzip
- [ ] 09:00 Scaffold `internal/builtins/gzip.go`, register 3 builtins
- [ ] 10:30 Implement `_gzip_decompress` with bounded reader
- [ ] 11:30 Implement `_gzip_compress` (base64 out)
- [ ] 13:00 Implement `_gzip_decompressFile` (FS effect)
- [ ] 14:00 Write `gzip_test.go` — round-trip + bomb + empty
- [ ] 15:30 Write `std/gzip.ail` surface
- [ ] 16:00 `make quick-install && make test` — milestone checkpoint

**Day 2 (Tue)** — M2: tar read
- [ ] 09:00 Create tar fixtures (`testdata/sample.tar`)
- [ ] 10:00 Implement `_tar_listEntries` with entry cap
- [ ] 11:30 Build `TarEntry` record helper
- [ ] 13:00 Implement `_tar_readEntry` and `_tar_readEntryBytes`
- [ ] 14:30 Write `std/tar.ail` surface
- [ ] 15:30 Write `tar_test.go` — list/read/caps/unicode
- [ ] 17:00 Milestone checkpoint

**Day 3 (Wed)** — M3: extractAll + readFromGzip
- [ ] 09:00 Port tarbomb defence helper (`filepath.IsLocal` + realpath)
- [ ] 10:30 Implement `_tar_extractAll` with sandbox check
- [ ] 12:00 Create `evil-tarbomb.tar` fixture + symlink-escape fixture
- [ ] 13:30 Implement `_tar_readFromGzip` (compose, no temp file)
- [ ] 15:00 Create `sample.tar.gz` fixture
- [ ] 15:30 Extend `tar_test.go`: tarbomb/symlink/gzip composition
- [ ] 17:00 Milestone checkpoint

**Day 4 (Thu)** — M4: docs + release prep
- [ ] 09:00 Write `examples/runnable/tar_gzip_reader.ail`
- [ ] 10:30 Run `make verify-examples`; fix any issues
- [ ] 11:30 CHANGELOG.md + prompts update
- [ ] 13:00 Update stdlib reference docs
- [ ] 14:30 `make ci` full pass
- [ ] 15:30 Send ack message to cli agent (AILANG Parse) with API confirmation
- [ ] 16:00 Sprint retrospective, prep for sprint-evaluator handoff

## Success Metrics

- [ ] All 4 milestones complete with acceptance criteria met
- [ ] Test coverage for `internal/builtins/gzip.go` and `tar.go` ≥ 85%
- [ ] `examples/runnable/tar_gzip_reader.ail` verified working
- [ ] No new file exceeds 800 LOC (`make check-file-sizes`)
- [ ] Axiom compliance verified (no hard violations on A1/A3/A4/A7)
- [ ] CHANGELOG.md updated under v0.12.0
- [ ] AILANG Parse (cli agent) acked with API confirmation message

## Files to Create/Modify

**Create:**
- `std/gzip.ail` (~25 LOC)
- `std/tar.ail` (~30 LOC)
- `internal/builtins/gzip.go` (~180 LOC)
- `internal/builtins/gzip_test.go` (~120 LOC)
- `internal/builtins/tar.go` (~400 LOC)
- `internal/builtins/tar_test.go` (~220 LOC)
- `internal/builtins/testdata/sample.tar`
- `internal/builtins/testdata/sample.tar.gz`
- `internal/builtins/testdata/evil-tarbomb.tar`
- `examples/runnable/tar_gzip_reader.ail` (~40 LOC)

**Modify:**
- `CHANGELOG.md` — v0.12.0 entry
- `prompts/` current teaching prompt — document new modules
- Builtin registration (wherever `init()` chain is rooted)

**Total estimate:** ~1,035 LOC production + fixtures (under the 1,200 LOC "must split" threshold per file)

## Risks & Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Tarbomb defence misses subtle symlink escape | Med | Port std/zip's existing defence suite verbatim; add explicit symlink-escape fixture |
| Gzip bomb bypasses cap via streaming | Low | `io.LimitReader` wrapping `gzip.Reader` before reading any bytes |
| Fixture generation flaky across platforms | Low | Check fixtures into repo as binary artifacts, don't generate at test-time |
| Go `archive/tar` quirk on exotic formats (GNU long names, sparse files) | Low | Document limitations; reject exotic formats with typed Err rather than panic |
| AILANG Parse's adapter code expects slightly different API shape | Low | Cli agent committed to writing adapter against whatever shape we pick — confirm via ack message at M4 |

## Dependencies

- None. All work is additive within `internal/builtins/` and `std/`.
- Does NOT block or get blocked by M-PERF5 (different area of codebase).

## Open Questions

None — Design Freeze resolved with proposed defaults.

## Handoff

On completion, sprint-executor hands to sprint-evaluator for acceptance review.

---

**Created**: 2026-04-14
**Source design doc**: [design_docs/planned/v0_12_0/m-stdlib-tar-gzip.md](m-stdlib-tar-gzip.md)
**Upstream request**: AILANG message `fb09cfae-d288-4139-9dad-e0e35e12ebe7` (cli / AILANG Parse v0.15.0)

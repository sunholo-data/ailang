# Sprint Plan: M-STDLIB-ZIP — ZIP Archive Standard Library

## Summary
Implement read-only ZIP archive support as Go builtins under `std/zip`, enabling AILANG programs to list and read entries from ZIP-based document formats (DOCX, XLSX, PPTX).

**Duration:** 1.5 days (~11 hours)
**Dependencies:** None — uses Go stdlib `archive/zip`
**Risk Level:** Low (well-scoped, follows established builtin patterns)
**Design Doc:** [m-stdlib-zip.md](m-stdlib-zip.md)

## Current Status Analysis

### Completed Recently (reference velocity)
- ✅ M-STDLIB-DATETIME: ~530 LOC impl + ~390 LOC tests in ~2 days (12 builtins)
- ✅ M-DX15 bytes builtins: ~300 LOC impl + ~310 LOC tests in ~1 day (5 builtins)
- ✅ Chain metric capture + executor fix: ~630 LOC across 13 files in ~1 day

### Velocity
- Recent average: ~300-500 LOC/day (implementation + tests)
- Builtin registration pattern: ~40 LOC per builtin (spec + metadata)
- Effect handler: ~40-50 LOC per operation
- Test coverage: ~1.0-1.5x test LOC vs implementation LOC

### Remaining from Design Doc
- ⏳ 3 builtins: `_zip_listEntries`, `_zip_readEntry`, `_zip_readEntryBytes`
- ⏳ 3 FS effect operations: `zipListEntries`, `zipReadEntry`, `zipReadEntryBytes`
- ⏳ Security hardening: sandbox, zip bombs, path traversal, entry limits
- ⏳ Test fixtures + integration tests
- ⏳ Example file

## Proposed Milestones

### Milestone 1: Effect Operations + Builtin Specs
**Goal:** Register 3 ZIP operations under the FS effect and register 3 builtin specs with full metadata.
**Estimated:** ~120 LOC implementation + ~0 LOC tests = ~120 LOC
**Duration:** 2 hours

**Tasks:**
1. Add `zipListEntries`, `zipReadEntry`, `zipReadEntryBytes` to `internal/effects/fs.go` via `RegisterOp("FS", ...)`
2. Create `internal/builtins/zip.go` with 3 `BuiltinSpec` registrations (Module: `"std/zip"`)
3. Add `makeOk` / `makeErr` helpers for `Result[T, string]` return values
4. Verify `ailang doctor builtins` passes with new builtins

**Acceptance Criteria:**
- [ ] 3 new FS operations registered (`RegisterOp`)
- [ ] 3 `BuiltinSpec` entries with full `Metadata` (description, params, examples, tags)
- [ ] `ailang doctor builtins` passes
- [ ] `make build` succeeds
- [ ] Linting clean

**Risks:**
- Type builder may not support `T.List(T.String())` inside `T.App("Result", ...)` — Mitigation: check datetime/json_decode for precedent

### Milestone 2: Core ZIP Implementations
**Goal:** Implement the 3 ZIP reading functions with full error handling.
**Estimated:** ~130 LOC implementation + ~200 LOC tests = ~330 LOC
**Duration:** 4 hours

**Tasks:**
1. Implement `zipListEntriesImpl` — open ZIP, iterate `r.File`, return `Ok([string])` or `Err(string)`
2. Implement `zipReadEntryImpl` — open ZIP, find entry by name, read as UTF-8 string
3. Implement `zipReadEntryBytesImpl` — same as above but base64-encode via `encoding/base64`
4. Create `internal/builtins/zip_test.go` with test fixture helper (`createTestZip`)
5. Write tests: happy path (list, read text, read binary), missing file, missing entry, empty ZIP

**Acceptance Criteria:**
- [ ] `_zip_listEntries("test.zip")` returns `Ok(["hello.txt", "data/config.xml"])`
- [ ] `_zip_readEntry("test.zip", "hello.txt")` returns `Ok("Hello, World!")`
- [ ] `_zip_readEntryBytes("test.zip", "binary.dat")` returns `Ok(<base64 string>)`
- [ ] Missing file returns `Err("cannot open ZIP: ...")`
- [ ] Missing entry returns `Err("entry not found: ...")`
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- `defer rc.Close()` inside loop — Mitigation: extract to helper function to avoid deferred close accumulation

### Milestone 3: Security Hardening
**Goal:** Add sandbox enforcement, zip bomb protection, path traversal rejection, and entry count limits.
**Estimated:** ~60 LOC implementation + ~80 LOC tests = ~140 LOC
**Duration:** 2 hours

**Tasks:**
1. Validate path against `AILANG_FS_SANDBOX` before `zip.OpenReader` (reuse `internal/effects/fs.go` sandbox logic)
2. Add `maxDecompressedSize` check (100MB default) — read up to limit, error if exceeded
3. Reject entry names containing `../` (path traversal)
4. Limit archive to 10,000 entries maximum
5. Write security-specific tests for each check

**Acceptance Criteria:**
- [ ] Sandbox violation returns clear error
- [ ] Entry >100MB returns `Err("entry too large: ...")`
- [ ] Entry name `"../../etc/passwd"` is rejected
- [ ] Archive with >10,000 entries returns `Err("too many entries: ...")`
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Sandbox logic may be internal to `effects/fs.go` — Mitigation: extract reusable `validatePath` helper or call `effects.Call` which does the check

### Milestone 4: Integration Test + Example + Docs
**Goal:** End-to-end test with a DOCX-like ZIP fixture, working example file, and CHANGELOG update.
**Estimated:** ~40 LOC implementation + ~60 LOC tests + ~30 LOC example = ~130 LOC
**Duration:** 2 hours

**Tasks:**
1. Create integration test: build a mini DOCX-like ZIP (with `word/document.xml` entry), read it back
2. Create `examples/zip_reader.ail` demonstrating all 3 functions
3. Verify example works: `ailang run --caps FS --entry main examples/zip_reader.ail`
4. Update CHANGELOG.md with new `std/zip` module
5. Run `make verify-examples` to confirm example passes

**Acceptance Criteria:**
- [ ] Integration test creates ZIP with XML content and reads it back
- [ ] `examples/zip_reader.ail` exists and runs successfully
- [ ] `make verify-examples` passes (new example included)
- [ ] CHANGELOG.md updated
- [ ] `make test` passes (all tests)
- [ ] `make lint` passes

**Risks:**
- Example may hit module system issues if `std/zip` module path isn't recognized — Mitigation: check how `std/fs` is wired in the module loader

## Success Metrics
- Test coverage: >90% for `internal/builtins/zip.go`
- New tests: ~15-20 test cases across `zip_test.go`
- Examples passing: `examples/zip_reader.ail` verified working
- Documentation: CHANGELOG.md updated
- All tests passing: `make test` ✅
- All linting passing: `make lint` ✅
- Total new LOC: ~465 implementation + ~340 tests = ~720 LOC

## Dependencies
- None — Go `archive/zip` and `encoding/base64` are stdlib packages
- M-STDLIB-XML is a companion feature but NOT a dependency

## Open Questions
- Should `readEntry` preserve whitespace exactly (including BOM)? Leaning yes for XML fidelity.
- Should we expose `CompressedSize`/`UncompressedSize` metadata per entry? Deferred for MVP.

## Notes
- Pattern follows `internal/builtins/fs.go` (3 FS-effectful builtins) + `internal/builtins/bytes.go` (result helpers)
- The `makeOk`/`makeErr` helpers can be shared with std/xml later — consider placing in a shared `result_helpers.go`
- ZIP operations reopen the archive each call (no stateful handles) — acceptable for document parsing workloads

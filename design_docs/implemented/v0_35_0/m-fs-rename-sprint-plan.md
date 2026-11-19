# Sprint Plan: M-FS-RENAME — std/fs rename/move (atomic file publish)

## Summary
Ship `renameFile` + `renameFileResult` for `std/fs`, closing issue #897: pure AILANG gets atomic file publish (write temp → rename) inside the FS capability and sandbox.

**Design doc:** [m-fs-rename.md](m-fs-rename.md) (ratified 2026-08-28; quorum 2 rounds, all objections addressed)
**Duration:** 1 session (~4h)
**Dependencies:** None
**Risk Level:** Low (additive, follows the proven 4-layer std/fs pattern end to end)

## Current Status Analysis

### Completed Recently
- ✅ std/fs 4-layer pattern proven by `removeFile`/`removeFileResult` (v0.16.0 convention, read end-to-end)
- ✅ Parked WIP in `internal/effects/fs_dir.go:159-253` — behavior verified against design semantics (V10)

### Velocity
- Recent comparable: builtin additions land as single-session commits (e.g. gitexec #954)
- Sprint scope ≈ 270 LOC (implementation ~145 + tests ~125) — well within a single session

### Remaining from Design Doc
- ⏳ Effect ops + registration (V10-verified WIP to adopt)
- ⏳ Builtins + stdlib wrappers + codegen stub row
- ⏳ Tests, example verification, CHANGELOG

## Proposed Milestones

### M1: Effect layer — adopt WIP + register ops (~1h)
**Estimated:** 55 impl LOC (mostly adopted from verified WIP) + 2 registration lines

**Tasks:**
- Adopt `fsRenameFile`/`fsRenameFileResult` from parked WIP (V10-verified: both-path sandbox, no fallback)
- `RegisterOp("FS", "renameFile"/"renameFileResult", …)` in `internal/effects/fs.go`

**Acceptance criteria:**
- [ ] `Call(ctx, "FS", "renameFile", …)` renames within sandbox
- [ ] Sandbox escape rejected on BOTH oldPath and newPath (nothing renamed)

### M2: Builtin + stdlib + codegen (~1h)
**Estimated:** 70 LOC registrations + 20 LOC std/fs.ail + 1 codegen row

**Tasks:**
- `_fs_rename` / `_fs_renameResult` in `internal/builtins/fs.go` (full metadata, Since v0.35.0)
- `_fs_rename` stub row in `registry_codegen_io.go` (sibling-consistent with `_fs_removeFile`; `_fs_renameResult` follows the `*Result` omission per design V9)
- `renameFile` / `renameFileResult` exports + doc comments in `std/fs.ail`

**Acceptance criteria:**
- [ ] `ailang builtins list` shows both with complete metadata
- [ ] `ailang doctor builtins` clean
- [ ] `stdlib_codegen_resolution_test.go` passes unmodified

### M3: Tests + verification + docs (~2h)
**Estimated:** ~125 test LOC

**Tasks:**
- `internal/effects/fs_test.go`: 6 cases — success, rename dir in sandbox, missing source, sandbox escape ×2 (oldPath/newPath), Result variant Ok/Err
- Regression surface: `TestFSExists_Success`, `TestFSSandbox_AbsolutePathOutsideSandbox` pass unmodified; `ailang run examples/runnable/directory_ops.ail`, `effects_fs_io.ail`, `examples/tests/test_effect_fs.ail` unchanged
- Issue #897 repro compiles + runs
- CHANGELOG entry under v0.35.0 (Stdlib section)

**Acceptance criteria:**
- [ ] Issue #897 repro passes `ailang check` and runs
- [ ] `make test` green; `make fmt` clean
- [ ] All 5 design-doc regression fixtures pass

## Success Metrics
- 50 → 52 builtins load; `ailang builtins list` complete
- Tests: 6 new, 0 regressions
- Issue #897 closable (`Fixes #897` in final commit)

## Out of Scope (per design doc)
Symlink-aware sandbox hardening (systemic follow-up doc), copy fallback, `fs.move` alias, Windows atomicity.
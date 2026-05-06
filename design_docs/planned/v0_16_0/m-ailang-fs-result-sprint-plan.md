# Sprint Plan — M-AILANG-FS-RESULT

**Sprint ID**: `M-AILANG-FS-RESULT`
**Target version**: v0.16.0
**Estimated**: 0.5 day (single session)
**Risk**: Low (additive only, no existing API changes)
**Dependencies**: None
**Design doc**: [m-ailang-fs-result.md](m-ailang-fs-result.md)

## Goal

Add 5 new `Result`-returning std/fs builtins so agent runtimes can
recover from fs syscall failures instead of crashing the entire
agent process. Migrate motoko_agent's tool dispatcher to use them.

## Milestones

### M1 — `readFileResult` (~120 LOC)

**Files**:
- `internal/effects/fs.go` — add `fsReadFileResult` handler + register
- `internal/builtins/fs.go` — add `_fs_readFileResult` registration
- `std/fs.ail` — add `readFileResult` export
- `internal/effects/fs_test.go` — add Ok + Err path tests

**Acceptance criteria**:
- `_fs_readFileResult("existing/file.txt")` returns `Ok("file contents")`
- `_fs_readFileResult("does/not/exist")` returns `Err("cannot read file: ...")`
- `ailang builtins doc _fs_readFileResult` shows the metadata
- Go tests for both paths pass with `-count=20`

### M2 — `writeFileResult` (~120 LOC)

**Files**: same set as M1, with `writeFile` analogues

**Acceptance criteria**:
- `_fs_writeFileResult("/tmp/ok.txt", "hi")` returns `Ok(())`
- `_fs_writeFileResult("/this/parent/does/not/exist/foo.txt", "x")` returns
  `Err("cannot write file: ...")` (no panic)
- Go tests for both paths pass with `-count=20`

### M3 — `appendFileResult`, `removeFileResult`, `mkdirAllResult` (~150 LOC)

**Files**: same set, three new handlers each following the M1 pattern

**Acceptance criteria**:
- All three new builtins registered; each has Ok+Err tests
- `make build && make test` green for the three test packages

### M4 — motoko_agent migration (~50 LOC, in motoko repo)

**Files**:
- `motoko_agent/src/core/tool_runtime.ail` — change `run_read_file`,
  `run_write_file`, `run_append_file` (if applicable) to use the new
  Result-returning builtins. On `Err`, return `ToolErrorResult{...}`.
- `motoko_agent/scripts/smoke_v2_writefile_missing_parent.ail` — new
  smoke that verifies a WriteFile to `nonexistent/dir/foo.txt`
  returns a tool-role error envelope (no agent crash).

**Acceptance criteria**:
- `make check_core` green (motoko side)
- New smoke passes
- Existing `smoke_v2_*` smokes still pass
- M9 provider matrix re-run: 25/25 green

### M5 — docs + CHANGELOG

**Files**:
- `changelogs/v0.10-current.md` — entry under "Added" with the 5 new
  builtins listed
- Move design doc from `planned/v0_16_0/` to `implemented/v0_16_0/` on
  completion

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| AILANG `Result` constructor naming differs from expectation | Low | `fsMakeOk` / `fsMakeErr` already work for `_fs_readFileBytes`; reuse them. |
| motoko's std/fs import set changes break dependent modules | Low | Additive only; existing imports stay valid. |
| Sandbox-respecting tests are flaky | Low | Use `t.TempDir()` instead of relying on `AILANG_FS_SANDBOX` env. |

## Success metrics

- 5 new builtins, 10+ new tests (Ok+Err per builtin), 0 regressions in
  existing fs tests.
- motoko's tool dispatcher: any user-supplied path that fails to
  read/write produces a tool-role envelope, no runtime panic.
- M9 provider matrix still 25/25 after motoko migration.

## Execution mode

**Sequential** — each milestone depends on the previous (M3 reuses
the Go-side helper pattern from M1+M2; M4 depends on M3 being
shipped). Single Claude Code session, no need for parallel sub-
agents.

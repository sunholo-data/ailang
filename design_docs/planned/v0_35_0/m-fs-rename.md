# M-FS-RENAME — std/fs rename/move: atomic file publish for pure AILANG

**Status**: Planned
**Target**: v0.35.0
**Priority**: P1
**Estimated**: ~4 hours (single sprint session)
**Dependencies**: None
**Tracking**: GitHub issue #897 (filed by motoko_agent)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact — os.Rename on a given filesystem state is deterministic for same-fs targets |
| A2: Replayability | 0 | Trace records the op + args as with other FS ops; no new nondeterminism |
| A3: Effect Legibility | +1 | Makes an FS mutation explicit in `! {FS}` signatures instead of hiding it in a Process call |
| A4: Explicit Authority | +1 | FS capability already gates it; BOTH paths sandbox-resolved (a one-sided resolve would be an escape primitive) |
| A5: Bounded Verification | +1 | `ailang check` fully verifies calls; no new type machinery |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Agents publishing `run.json`/state files get torn-read-free writes without shelling out to `mv` |
| A8: Minimal Syntax | +1 | Pure stdlib + effect addition; zero new syntax |
| A9: Cost Visibility | 0 | No resource metering impact |
| A10: Composability | +1 | Composes with existing FS ops; Result variant composes with `?`/match flows |
| A11: Structured Failure | +1 | Result-returning variant makes fs failure a value, not an escape |
| A12: System Boundary | +1 | Filesystem boundary crossing stays inside the declared FS effect |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects — effect is declared, capability-gated
- [x] A4 (Authority): No ambient access — FS capability + sandbox enforced on both paths
- [x] A7 (Machines First): Removes a Process shell-out, a machine-analysis win

## Verification Log

Every load-bearing claim in this doc, verified 2026-08-28 at `007184b7b` (v0.34.0+58):

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | No rename/move op exists in the FS effect, builtins, or stdlib | `grep -rn "os.Rename\|\"rename\"" internal/effects/ internal/builtins/ std/*.ail` (excluding parked WIP file) | Empty — confirmed absent |
| V2 | FS effect has 18 registered ops, none named `rename` | `grep -c "RegisterOp(\"FS\"" internal/effects/fs.go` → 18; `grep -n rename internal/effects/fs.go` → empty | Confirmed |
| V3 | Sandbox path resolver exists and is the canonical gate | Read `resolveSandboxPath` at `internal/effects/fs.go:55`; used by every FS op | Confirmed |
| V4 | Parked WIP exists and is UNREGISTERED (inert) | `grep -n "fsRenameFile" internal/effects/*.go` → only definitions in `fs_dir.go:159-230`; no `RegisterOp` entries, no builtin registrations, no stdlib wrappers | Confirmed — see "Parked WIP" below |
| V5 | `IMP010: symbol 'renameFile' not exported by 'std/fs'` reproduces the issue's claim | Read `std/fs.ail` export list — no rename/move export | Confirmed (matches issue #897 repro) |
| V6 | No new error code is proposed; failures are Go-wrapped runtime messages like sibling ops | Read `fsRemoveFile` / `fsRemoveFileResult` — `fmt.Errorf("removeFile: %w", err)` pattern | Confirmed (nothing to grep-allocate) |
| V7 | stdlib.md needs no change (module-level table only) | Read `docs/docs/reference/stdlib.md` I/O section — lists modules, not functions | Confirmed |

## Problem Statement

`std/fs` has no rename/move primitive. Pure AILANG cannot publish a file atomically:
any workflow that writes a file another process watches (agents writing `run.json`,
mission state files, package manifests) must either accept torn reads or shell out to
`mv` via `std/process` — paying process-spawn cost, losing the FS capability boundary,
and dragging `Process` into an effect signature that should only need `FS`.

**Current State** (verified V1, V5): 18 FS ops exist (read/write/append/exists/listDir/
mkdir/isDir/isFile/removeFile + 5 Result variants); none is rename/move.

**Impact:** Every autonomous agent loop in this repo's ecosystem that does
write-temp-then-publish currently has no pure-AILANG path. Issue #897 requests exactly
`renameFile("run.json.tmp", "run.json")`.

## Goals

**Primary Goal:** Pure AILANG can atomically publish a file (write temp → rename over target) within the FS capability and sandbox.

**Success Metrics:**
- The issue #897 repro (`import std/fs (renameFile)`) passes `ailang check`
- Sandbox: renaming across the sandbox boundary is rejected for BOTH source and destination
- `make test` green with new unit tests covering success, missing source, sandbox escape ×2, Result variant
- `ailang builtins list` shows `_fs_rename` / `_fs_renameResult` with complete metadata

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Names: `renameFile` / `renameFileResult` | Stdlib API surface, agents learn it from prompts; matches `removeFile` convention and issue #897's expected name | agent | design | low |
| Sandbox applies to BOTH paths | Security semantics — one-sided resolution makes rename a sandbox-escape primitive (rename `../../etc/cron.d/x` in) | agent (following A4; reviewer may veto) | design | med |
| Directory renames allowed (Go `os.Rename` semantics, same filesystem) | Matches Go; atomic publish use case only needs files, but excluding dirs is artificial | agent | design | low |
| Cross-filesystem rename: surface `EXDEV` error, no fallback copy | Silent copy+delete fallback is a torn-write footgun — violates no-silent-fallback principle | human (confirm) | design | low |

### Design Freeze

- [ ] Confirm sandbox-on-both-paths semantics (A4-critical)
- [ ] Confirm cross-filesystem = loud error (no copy fallback)

## Solution Design

### Overview

Follow the established 4-layer pattern used by every std/fs operation (verified against
`removeFile`/`removeFileResult` end to end):

1. **Effect ops** in `internal/effects/fs_dir.go`: `fsRenameFile`, `fsRenameFileResult`
2. **Op registration** in `internal/effects/fs.go`: `RegisterOp("FS", "renameFile", ...)` / `renameFileResult`
3. **Builtins** in `internal/builtins/fs.go`: `_fs_rename` / `_fs_renameResult` with full `BuiltinMetadata` (Since: v0.35.0)
4. **Stdlib wrappers** in `std/fs.ail`: `renameFile` / `renameFileResult` with doc comments in house style

### Key Semantics

```
renameFile(oldPath, newPath) -> () ! {FS}
renameFileResult(oldPath, newPath) -> Result[(), string] ! {FS}
```

- Sandbox: when `AILANG_FS_SANDBOX` is set, **both** paths pass through
  `resolveSandboxPath` (`internal/effects/fs.go:55`); either escaping → error, nothing renamed.
- `os.Rename` atomicity: readers of `newPath` see either old or new content, never partial.
- Errors: missing source, permission denied, cross-device link (EXDEV), non-empty-dir
  constraints per Go `os.Rename` — surfaced as `Err("cannot rename file: …")` in the Result variant,
  runtime error in the panicking variant (matches `removeFile` pair, V6).

### Parked WIP (provenance + disposition)

`internal/effects/fs_dir.go` already contains unregistered `fsRenameFile` (line 163) and
`fsRenameFileResult` (line 230), written 2026-08-28 in a session that violated the work-routing
gate and was parked uncommitted per AGENTS.md. **Sprint-executor decides: adopt (register +
wrap + test), adapt, or rewrite from the doc.** The WIP is inert (V4) and uncommitted; nothing
else may build on it until this doc is approved. The doc's design supersedes the WIP text where they differ.

### Implementation Plan

**Phase 1: Effect layer** (~1h)
- [ ] Adopt/adapt parked `fsRenameFile` + `fsRenameFileResult` in `internal/effects/fs_dir.go`
- [ ] `RegisterOp("FS", "renameFile"/"renameFileResult", ...)` in `internal/effects/fs.go`

**Phase 2: Builtin + stdlib layer** (~1h)
- [ ] `_fs_rename` / `_fs_renameResult` registrations in `internal/builtins/fs.go` (full metadata, `Since: v0.35.0`)
- [ ] `_fs_rename` effect-stub entry in `internal/builtins/registry_codegen_io.go` (next to `_fs_removeFile`; Result variants follow the existing table's precedent of omission)
- [ ] `renameFile` / `renameFileResult` exports + doc comments in `std/fs.ail`

**Phase 3: Tests + validation** (~2h)
- [ ] `internal/effects/fs_test.go`: success (file), rename dir within sandbox, missing source, sandbox escape on oldPath, sandbox escape on newPath, Result variant Ok/Err
- [ ] `internal/builtins/registry_codegen_test.go` / `stdlib_codegen_resolution_test.go` pass unchanged (they enumerate; new export resolves to `_fs_` family automatically)
- [ ] Issue #897 repro passes `ailang check` and runs
- [ ] `ailang doctor builtins` clean; `make test` green

### Files to Modify/Create

**Modified files:**
- `internal/effects/fs_dir.go` — parked WIP adopted/adapted (+~55 LOC net)
- `internal/effects/fs.go` — +2 RegisterOp lines
- `internal/builtins/fs.go` — +2 registrations (~70 LOC)
- `internal/builtins/registry_codegen_io.go` — +1 stub-table entry
- `std/fs.ail` — +2 exports with doc comments (~20 LOC)
- `internal/effects/fs_test.go` — +6 test cases (~120 LOC)

## Conflict Surface

**Trigger:** touches `internal/effects/` — required per design-doc-creator. No parser/lexer/AST/typechecker positions are touched; the positions are the FS *op namespace*, the *builtin registry namespace*, and the *std/fs export surface*.

### Positions touched

- Adds two entries to the FS op table registered in `internal/effects/fs.go` `init()` (`RegisterOp("FS", "renameFile"/"renameFileResult", …)`)
- Adds two builtins to the `_fs_*` namespace in `internal/builtins/fs.go`
- Adds two exports to `std/fs.ail`'s module surface
- Adds one row to the codegen effect-stub table in `internal/builtins/registry_codegen_io.go`

**Effect-row algebra is NOT changed**: no new capability, no change to the `FS` effect set, no change to how `! {FS}` is inferred. The change is invisible to the type checker beyond two new typed constants.

### What else lives in these positions

| Position | Existing valid form | Shape |
|----------|--------------------|-------|
| FS op namespace (`RegisterOp("FS", …)`) | 18 ops (verified V2): `readFile`, `readFileBytes`, `writeFile`, `writeFileBytes`, `appendFile`, `appendFileBytes`, `exists`, `listDir`, `mkdir`, `mkdirAll`, `isDir`, `isFile`, `removeFile` + 5 `*Result` variants | `("<opName>", <func>)` string-keyed; op names unique strings |
| `_fs_*` builtin namespace | 12 registered `_fs_*` builtins (fs.go) | `RegisterEffectBuiltin(BuiltinSpec{Module: "std/fs", Name: "_fs_<name>"…})` |
| `std/fs` export surface | 19 exported functions incl. `removeFile` / `removeFileResult` (read, verified V5) | `export func <camelCase>(…) -> … ! {FS} = _fs_<name>(…)` |
| Codegen stub table | 9 `_fs_*` rows (read, registry_codegen_io.go:34–42); `*Result` variants deliberately absent | `{"_fs_name", "std/fs", "stdlibName", "GoFuncName", "FS", arity}` |

**Disambiguation:** none required — no grammar changes; op/builtin/export names are collision-free by V1/V2 (greps empty). The only name-resolution path is the stdlib export → `_fs_` family mapping, which `stdlib_codegen_resolution_test.go` enforces mechanically.

### Programs that MUST still work (regression fixtures — all verified to exist and read)

1. `examples/runnable/directory_ops.ail` — uses `mkdir, mkdirAll, isDir, isFile, fileExists, writeFile, removeFile` with `! {IO, FS}`; must run unchanged
2. `examples/runnable/effects_fs_io.ail` — `fileExists/readFile/writeFile` under `! {IO, FS}`; must run unchanged
3. `examples/tests/test_effect_fs.ail` — FS effect test program
4. `TestFSExists_Success` (internal/effects/fs_test.go:118) — asserts `exists` returns `BoolValue` true for an existing temp file and false for `/nonexistent/file.txt`
5. `TestFSSandbox_AbsolutePathOutsideSandbox` (fs_test.go:303) — asserts sandbox rejection of absolute paths escaping the sandbox

### Regression-surface tests (required)

Fixtures 4–5 are the pinned existing tests — they must keep passing unmodified (failures are regressions, not churn). Fixtures 1–3 are `ailang run` smoke-checked in Phase 3. New-surface tests (rename success/missing/escape×2/Result) are additive and listed in the Testing Strategy.

### Deliberately changes

Nothing. Purely additive: no existing op, builtin, export, or test behavior changes.

## Examples

### Example 1: Atomic publish (the issue #897 use case)

**Before** (impossible in pure AILANG — had to shell out):
```
process run ["mv", "run.json.tmp", "run.json"]   // pulls in Process capability + spawn cost
```

**After:**
```ailang
module repro
import std/fs (renameFile)

export func main() -> () ! {FS} {
  renameFile("run.json.tmp", "run.json")
}
```

### Example 2: Recoverable variant

```ailang
match renameFileResult("run.json.tmp", "run.json") {
  Ok(()) => ()
  Err(msg) => () -- log and retry next cycle
}
```

## Success Criteria

- [ ] Issue #897 repro compiles (`ailang check`) and runs (acceptance: exact program from the issue)
- [ ] Sandbox: rename with `oldPath` outside sandbox → error, nothing changed (acceptance: unit test)
- [ ] Sandbox: rename with `newPath` outside sandbox → error, nothing changed (acceptance: unit test)
- [ ] `renameFileResult` returns `Err` on missing source, `Ok(())` on success (acceptance: unit test)
- [ ] `ailang doctor builtins` reports no issues for the two new builtins
- [ ] All tests passing; `make test` green
- [ ] CHANGELOG entry added under v0.35.0 (Stdlib section)

## Testing Strategy

**Unit tests (internal/effects/fs_test.go):** six cases listed in Phase 3; sandbox tests follow
the `ctx.Env.Sandbox = <tmpdir>` pattern already used by `TestFSReadFile_Sandbox*`.

**Regression-surface (Conflict Surface fixtures):**
- `TestFSExists_Success` + `TestFSSandbox_AbsolutePathOutsideSandbox` pass **unmodified**
- `ailang run examples/runnable/directory_ops.ail`, `effects_fs_io.ail`, `examples/tests/test_effect_fs.ail` — output unchanged

**Registry tests:** existing `stdlib_codegen_resolution_test.go` must pass unmodified — proves the
new stdlib export resolves through the `_fs_` family.

**Manual testing:** run the issue repro under `AILANG_FS_SANDBOX=$(mktemp -d)` and confirm in-sandbox
success + out-of-sandbox rejection with a readable error.

## Deferred Decisions

- Whether a `copyFile` sibling follows later — agent may propose a separate doc; explicitly not decided here.
- Exact wording of the EXDEV error string — agent may choose, must contain `renameFile:` prefix.

## Non-Goals

- **Copy-based move fallback** — silently degrades atomicity; no-silent-fallback principle.
- **`fs.move` alias** — one name per concept; `renameFile` is the issue-requested name.
- **Windows reserved-name / long-path handling** — Go stdlib behavior inherited as-is, same as sibling ops.

## Timeline

**Single session** (~4h): Phase 1 → 2 → 3 in one sprint session, tests written alongside each layer.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| One-sided sandbox resolution escapes the sandbox | High (security) | Both paths resolved; two dedicated escape tests block regression |
| Rename used as partial-write vector | Low | Doc comment prescribes write-temp-then-rename pattern; EXDEV surfaces loudly |
| Parked WIP drifts from approved doc | Low | Sprint-executor explicitly re-reviews WIP against this doc before adopting |

## Related Documents

<!-- Auto-populated by Ollama neural search on "fs rename"; duplicate gate passed (max neural 0.40, unrelated) -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_9_11/m-dx-app-package-adoption.md](design_docs/implemented/v0_9_11/m-dx-app-package-adoption.md) (0.38) — package file-layout conventions
- [design_docs/implemented/v0_3/M-R5b_record_extension.md](design_docs/implemented/v0_3/M-R5b_record_extension.md) (0.35) — additive-extension precedent

**Planned (check for overlap):**
- [design_docs/planned/v0_29_0/m-eval-results-folder-structure.md](design_docs/planned/v0_29_0/m-eval-results-folder-structure.md) (0.40) — result-folder writes, a rename consumer
- [design_docs/planned/m-dynamic-data-runtime-plane.md](design_docs/planned/m-dynamic-data-runtime-plane.md) (0.33) — unrelated runtime plane

## References

- GitHub issue #897 (repro, measurement, rationale)
- `M-AILANG-FS-RESULT` (v0.16.0) — the Result-variant convention this doc extends
- `internal/effects/fs.go` — `resolveSandboxPath` semantics (V3)

## Future Work

- `std/fs.copyFile` (separate doc, own atomicity semantics)
- Directory-tree move helper, if agent workloads demand it

---

**Document created**: 2026-08-28
**Last updated**: 2026-08-28
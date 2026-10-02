# M-NAMEDTEST-TEMPFILE-LEAK — Interrupted `ailang test` leaves `_namedtest_body_*.ail` in the package; pkg quality/publish/git ship it

**Status**: Implemented (2026-10-02) — via a different mechanism than designed; see note

> **Implementation note (2026-10-02).** The design's premise — that the temp body must live in
> the source dir for sibling imports and manifest discovery — did not hold at HEAD: the module
> loader's base dir and the package search both come from `pipeline.Config.PackageDir`, which the
> harness already sets, not from the root file's directory. So the body is now written to a
> private `os.MkdirTemp` dir (`internal/testing/executor.go`), and **no kill of any kind can leave
> it in the package** — which removes the need for Layer A (PID-named files, signal handler,
> dead-PID sweep). A new `pipeline.Config.TransientRoot` marks the root as a harness copy: MOD010
> is skipped for it (the Phase 3 warning item, done), it is never compile-cached, and the cache
> for the other modules stays in `PackageDir`. Regression test: the package dir is made
> read-only for the run (`internal/testing/named_test_tempfile_test.go`).
>
> Partly done from Layer B: `pkg.IsNamedTestBodyFile` (`internal/pkg/testartifact.go`) is the
> one predicate, and both `ailang test` walks skip and name leftover debris from older binaries
> (fixes the V6 doubled-test-count symptom). **Not done:** the PUB024 quality gate and the
> tarball / hasher / staging / `check` walk exclusions for debris written by pre-fix binaries,
> the `pkg init` `.gitignore` scaffold, and the package-authoring guide note. New debris can no
> longer be produced, so these only matter for packages that already carry a leftover file.
**Target**: v0.51.1
**Priority**: P1 (Medium-High) — blocks a clean `publish` for any maintainer whose CI timed out mid-test; manual workaround exists (delete the file)
**Estimated**: 2 days (~12h: 4h consumer immunity + 4h producer hardening + 4h tests/docs)
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Publish tarball and content hash become independent of transient debris; a stale body no longer silently re-executes in `ailang test <dir>` (120 tests instead of 60, measured — V6) |
| A2: Replayability | 0 | No trace/replay changes |
| A3: Effect Legibility | 0 | No effect-row changes; the temp file is tooling, not language |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | A local pre-publish gate (PUB024) catches debris before the registry round-trip, next to the existing PUB gates |
| A6: Safe Concurrency | +1 | PID-liveness-checked sweep cannot delete a concurrent live run's temp body (the naive sweep would) |
| A7: Machines First | +1 | Debris surfaces as a machine-readable gate code (PUB024) instead of a human noticing "compile: 19 files" |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | 0 | No resource-budget changes |
| A10: Composability | +1 | One predicate reused by 7 scanning surfaces; one gate reused by `pkg quality`, `publish`, and the registry validator (the "ONE quality report" doctrine, quality.go:9-23) |
| A11: Structured Failure | +1 | Signal cleanup preserves conventional exit codes (128+signal); refusal names the files and the remedy instead of silently shipping them |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — the opposite: hash/tarball stop depending on debris
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Gate code + structured message, not human-only output

## Problem Statement

Killing `ailang test --package .` mid-run (CI timeout → SIGTERM, or Ctrl-C) leaves a
`_namedtest_body_<digits>.ail` file (mode 0600) in the package root. It is a full copy of the
test module — same `module ..._test` line plus a folded named-test body — because the named-test
executor materialises the test body as a real file next to the original source so that relative
imports (`./types`, `./engine`) and the package manifest (`ailang.toml` / `ailang.lock`) resolve
(executor.go:259-267, V1). The only cleanup is `defer os.Remove(pipelineFilename)`
(executor.go:288, V2), and Go defers do not run when the process dies from a signal's default
disposition.

**Current State** (report: sunholo/relativity 0.5.0, sunholo-data/ailang-packages branch
`relativity-0.5.0-m1`; reproduced on v0.51.0, commit b99dd25 binary + code-read at 4460d91b):

- `ailang pkg quality .` reports `compile: 19 files` instead of 18 — the debris carries a
  `module ..._test` declaration, so `check.DiscoverPackageSources` classifies it as a *source*
  file (V3, V5).
- `ailang publish --dry-run` ships it: tarball 49,824 bytes vs 47,372 clean (report), because
  `CreateTarball` includes every `*.ail` (V7).
- `git add -A` stages it (mode 0600, not gitignored) — it can land in the source repo, not just
  the tarball.
- **New symptom found during verification:** plain `ailang test .` *re-runs* the stale body as a
  test file — 120 tests instead of 60 in the reproduction (V6). A stale body silently doubles
  the next test run and can turn a passing suite red.
- The debris also enters the smoke staging workspace (V8) and changes the published content hash
  (V9), so two publishes from "the same" tree can hash differently.

**Impact:** every package maintainer whose CI kills a slow test run (timeout) then publishes.
Severity: publish-blocking contamination with a manual workaround; the git side (add -A) has no
workaround except remembering.

## Goals

**Primary Goal:** An interrupted `ailang test` never contaminates `pkg quality`, `publish`,
`git add -A`, or the next test run — debris is either prevented (signal cleanup + sweep) or
refused loudly (PUB024), never silently shipped.

**Success Metrics:**
- SIGTERM mid-run leaves zero `_namedtest_body_*` files (subprocess signal test)
- After an *unhandleable* kill (SIGKILL): `pkg quality` exits 2 with PUB024 naming the file;
  `publish` refuses; tarball bytes and content hash are identical to the clean tree; the next
  `ailang test --package .` sweeps the debris and runs the correct test count
- `ailang test .` on a debris-carrying dir runs 60 tests, not 120 (regression test)
- All 7 `.ail`-scanning surfaces skip the pattern via ONE predicate (no duplicated literals)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Refuse, not silently ignore**: PUB024 is a hard gate (quality exit 2, publish refuses) while tarball/hash/staging exclude the pattern as defense-in-depth | "Ignore" cures the tarball but leaves the `git add -A` contamination undetected; refusal forces the delete and keeps debris a visible anomaly. Exclusion still guarantees hash/tarball determinism (A1) if a gate is ever bypassed | human | design | med |
| Temp body stays in the **source dir** (not `os.TempDir`) | Relative sibling imports + manifest discovery require it (executor.go:259-267, V1); moving it means a virtual-module-loading refactor of the loader — out of scope (Non-Goal) | compiler (mechanism) | design | high |
| Embed the **creating PID** in the filename (`_namedtest_body_<pid>_<rand>.ail`) and sweep only dead-PID files at test start | Makes the startup sweep safe against concurrent live runs (A6); unconditional sweep would delete a sibling run's in-flight body | human | design | med |
| Shared predicate lives in `internal/pkg` (`NamedTestBodyPrefix` + `IsNamedTestBodyFile`); `internal/testing` imports it for creation | One literal, seven consumers. Import direction verified acyclic: `internal/pipeline` already imports `internal/pkg` (package_resolver.go:11) and `internal/pkg` imports neither pipeline nor testing (quality.go:21 comment, V14) | human | design | low |
| Signal handler lives in the **`ailang test` command** (cmd layer), backed by a process-wide in-flight registry in `internal/testing`; exit 128+signal after cleanup | The library cannot invent an exit policy; the CLI owns process lifecycle. Keeps `go test`-embedded use of `internal/testing` handler-free | agent | design | low |

### Design Freeze

- [x] Refuse-not-ignore (PUB024 hard gate) — chosen above
- [x] Temp body remains source-dir-local — mechanism constraint, virtual loading deferred
- [x] PID-in-name + dead-PID sweep (Windows falls back to mtime age, see Phase 2)
- [x] Predicate home: `internal/pkg`; `internal/testing` → `internal/pkg` import is new and acyclic (V14)
- [x] Error code PUB024 verified unallocated (V13)

## Solution Design

### Overview

Two independent layers, each sufficient for its half of the problem:

**Layer A — producer hardening** (make the leak rare): the executor embeds its PID in the temp
body name and registers the in-flight path in a process-wide registry; the `ailang test` command
installs a SIGINT/SIGTERM/SIGHUP handler that drains the registry and exits conventionally; both
test runners sweep dead-PID debris at startup.

**Layer B — consumer immunity** (make leftover debris harmless *and* loud): one predicate in
`internal/pkg` classifies the pattern; every `.ail`-scanning surface skips it; `BuildQualityReport`
gains a hard PUB024 gate that fires when debris is present — which automatically makes
`pkg quality` red, `publish` refuse, and the registry validator refuse, because all three call
the same report builder (quality.go:9-23 doctrine).

Layer B is mandatory even with Layer A: SIGKILL, CI `kill -9`, OOM, and power loss run no Go
code at all — no defer, no handler. Tooling must not trust process cleanup for files it ships.

### Architecture

**Components:**

1. **`internal/pkg/testartifact.go`** (new): `const NamedTestBodyPrefix = "_namedtest_body_"`
   and `func IsNamedTestBodyFile(base string) bool` (base-name match: prefix + optional
   `<pid>_<digits>` or legacy `<digits>` + `.ail` suffix). Plus
   `FindTransientTestArtifacts(dir string) []string` (glob, sorted) used by the quality input
   assembly. The executor's create-pattern uses the same exported prefix via a
   `NamedTestBodyPattern(pid int)` helper so creator and classifiers can never drift.
2. **`internal/testing/tempbody.go`** (new): mutex-guarded in-flight registry
   (`RegisterTempBody`, `UnregisterTempBody`, `CleanupInFlightBodies()`) and
   `SweepStaleTempBodies(dir string) (removed, skipped []string, err error)` implementing the
   PID-liveness rule (Unix `kill(pid, 0)` → dead ⇒ remove, alive ⇒ skip and report; no-PID
   legacy shape or non-Unix ⇒ remove only when mtime older than 1 hour).
3. **`internal/testing/executor.go`**: create with `os.CreateTemp(sourceDir,
   pkg.NamedTestBodyPattern(os.Getpid()))`; register before write, unregister in the existing
   defer (which keeps its `os.Remove`).
4. **`cmd/ailang/test.go` + `cmd/ailang/commands_language.go`**: `handleTest` installs the
   signal handler before dispatch (V16: dispatch site confirmed at commands_language.go:307-309);
   `runTestsV2` and `runPackageTests` call `SweepStaleTempBodies` first and print what was
   swept; both walks skip `pkg.IsNamedTestBodyFile`.
5. **Consumer exclusions**: `internal/pkg/discover.go` (walk skip + `TransientFiles` field),
   `internal/pkg/tarball.go`, `internal/pkg/hasher.go`, `internal/pkg/publish_validator.go`
   (`shouldStageFile`), `internal/check/package.go` (`DiscoverPackageSources` walk),
   `cmd/ailang/check.go` (dir-mode walk).
6. **PUB024 gate**: `QualityInputs.TransientFiles []string`; badge level `gate`
   ("stale named-test body file(s) present — delete before publish (left by an interrupted
   `ailang test` run)") — one implementation covers `pkg quality`, `publish --dry-run`, and the
   registry validator.

### Walk Inventory (conflict-surface equivalent)

This change touches no parser/typechecker/codegen position — its "conflict surface" is the set of
directory walks that scan `*.ail` and how each treats the pattern after the fix:

| Surface | Today (v0.51.0) | After |
|---------|-----------------|-------|
| `check.DiscoverPackageSources` (internal/check/package.go:243) — backs `pkg quality` compile count, `check --package`, `verify --package` | classifies debris as **source** (`HasModuleDeclaration` true — V3) | skip; count unchanged by debris |
| `pkg.DiscoverPackageSources` (internal/pkg/discover.go:76) | debris → `OrphanFiles` | skip + collect into `TransientFiles` |
| `pkg.CreateTarball` (internal/pkg/tarball.go:63) | includes debris (any `*.ail`) | exclude (defense-in-depth under the gate) |
| `pkg` content hasher (internal/pkg/hasher.go:31) | hashes debris → hash differs from clean tree | exclude → hash debris-independent |
| `shouldStageFile` (internal/pkg/publish_validator.go:178) | stages debris into smoke workspace | exclude |
| `cmd/ailang/test.go:31` `runTestsV2` | **re-runs** debris as a test file (V6) | skip (plus startup sweep removes it) |
| `cmd/ailang/test.go:153` `runPackageTests` | counts debris as a source module in the preamble | skip |
| `cmd/ailang/check.go:545` dir mode | compiles debris | skip |

Deliberately unchanged: `docs.go`, `examples.go`, `compile.go` walks (repo-level tooling, not the
publish path — noted in Non-Goals), and `*_test.ail` discovery (debris does not match that glob).

### Implementation Plan

**Phase 1: Consumer immunity** (~4 hours)
- [ ] `internal/pkg/testartifact.go`: prefix const, `NamedTestBodyPattern(pid)`,
      `IsNamedTestBodyFile`, `FindTransientTestArtifacts` + unit tests
- [ ] Exclude the pattern in all 7 walks above (one predicate, no new literals)
- [ ] `QualityInputs.TransientFiles` + PUB024 badge at gate level in `BuildQualityReport`;
      wire from `pkg_quality.go` and `pkg_publish.go`
- [ ] Regression tests: planted `_namedtest_body_9.ail` → quality exit 2 with PUB024;
      `CreateTarball` output byte-identical clean vs debris; hash identical clean vs debris

**Phase 2: Producer hardening** (~4 hours)
- [ ] `internal/testing/tempbody.go`: in-flight registry + `SweepStaleTempBodies`
      (PID liveness; legacy-shape/mtime fallback) + unit tests
- [ ] `executor.go`: PID-embedded pattern + register/unregister around the existing defer
- [ ] `handleTest` signal handler (SIGINT/SIGTERM/SIGHUP): drain registry, print, `os.Exit(128+sig)`
- [ ] Startup sweep in `runTestsV2` + `runPackageTests`, result printed in the preamble
- [ ] Subprocess test: start `ailang test --package .` against a slow suite, SIGTERM mid-body,
      assert no `_namedtest_body_*` remains (reuse the planted-window technique from V6)

**Phase 3: Polish + docs** (~4 hours)
- [ ] Suppress the spurious `MOD010` temp-path warning for transient body filenames
      (emitter: internal/pipeline/mod010.go:46) — every named-test run in a relaxed/temp dir
      currently prints a warning naming the ephemeral file (observed in V6 runs)
- [ ] `ailang pkg init` scaffolds `.gitignore` containing `_namedtest_body_*.ail` (and notes
      `.ailang/`) for new packages
- [ ] Document the invariant in `cmd/ailang/guides/package-authoring.md`: debris is refused by
      quality, swept by the next test run, never published
- [ ] `make fmt`, `make test-core`, `make check-boundaries` (new testing→pkg import)

### Files to Modify/Create

**New files:**
- `internal/pkg/testartifact.go` (~50 LOC) — the ONE predicate + pattern helpers
- `internal/pkg/testartifact_test.go` (~80 LOC)
- `internal/testing/tempbody.go` (~120 LOC) — registry, sweep, liveness probe
- `internal/testing/tempbody_test.go` (~150 LOC)

**Modified files:**
- `internal/testing/executor.go` (+15/-3) — PID pattern, register/unregister
- `cmd/ailang/commands_language.go` (+12) — install signal handler in `handleTest`
- `cmd/ailang/test.go` (+30) — sweep calls + walk skips (both runners)
- `cmd/ailang/check.go` (+4) — dir-walk skip
- `internal/pkg/discover.go` (+10) — walk skip, `TransientFiles`
- `internal/pkg/tarball.go` (+4) — exclude
- `internal/pkg/hasher.go` (+4) — exclude
- `internal/pkg/publish_validator.go` (+3) — `shouldStageFile` exclude
- `internal/pkg/quality.go` (+18) — `TransientFiles` input + PUB024 gate badge
- `cmd/ailang/pkg_quality.go` (+6) — wire input; surface swept/ignored count
- `cmd/ailang/pkg_publish.go` (+4) — wire input
- `internal/pipeline/mod010.go` (+3) — suppress warning for transient filenames
- `cmd/ailang/pkg_init.go` (+6) — `.gitignore` scaffold

## Examples

### Example 1: CI timeout, then publish (the reported scenario)

**Before:**
```
$ ailang test --package .        # CI 10-minute timeout → SIGTERM
$ ls
_namedtest_body_3176816492.ail   # 0600, full copy of the test module
$ ailang pkg quality .
  compile:   ✓ 19 files          # 18 real + 1 debris
$ ailang publish --dry-run       # tarball 49,824 B — debris inside
$ git add -A                     # debris staged
```

**After:**
```
$ ailang test --package .        # CI timeout → SIGTERM
interrupted — removed 1 in-flight test body file (_namedtest_body_4821_…); exit 143
$ ls                             # clean
# Even after kill -9 (no Go code runs):
$ ailang pkg quality .
  ✗ PUB024 stale named-test body file(s) present: _namedtest_body_3176816492.ail —
       delete before publish (left by an interrupted `ailang test` run)
Exit: 2
$ ailang test --package .        # next run sweeps it
  swept 1 stale test body file (no live owner)
```

### Example 2: concurrent runs must not eat each other

Two `ailang test --package .` in the same checkout (e.g. a matrix CI sharing a workspace):
run B's startup sweep sees `_namedtest_body_4821_*.ail`, probes PID 4821 — alive — skips it.
Run A's temp body survives until A finishes. The sweep reports `skipped 1 (live owner)`.

## Success Criteria

- [ ] Subprocess SIGTERM test: no `_namedtest_body_*` remains after an interrupted run
- [ ] SIGKILL-leaves-debris scenario: `pkg quality` exit 2 with PUB024 naming the file; `publish` refuses with the same gate; tarball + hash byte-identical to clean tree
- [ ] `ailang test .` on a debris dir runs 60 tests, not 120 (regression test)
- [ ] Startup sweep removes dead-PID debris; skips live-PID debris (unit tests with a real child PID)
- [ ] `compile:` count in `pkg quality` identical with and without planted debris
- [ ] All tests passing (`make test-core` + full `make test`); `make check-boundaries` clean with the new testing→pkg import
- [ ] Documentation updated (package-authoring guide, `pkg init` scaffold)

## Testing Strategy

**Unit tests:**
- Predicate: prefix/PID/legacy shapes, non-matching names (`_smoke.ail`, `x_test.ail`, `_namedtest_body_readme.md`)
- Registry: concurrent register/unregister/cleanup (race detector)
- Sweep: dead PID removed, live PID skipped, legacy shape removed only when old (mtime), mtime fallback on non-Unix
- Discover/tarball/hash/staging: planted debris excluded (byte-level tarball assertion)
- PUB024: fires at gate level with files listed; silent when clean; registry-validator mode unaffected (tarballs contain no debris)

**Integration tests:**
- Subprocess signal test (Phase 2): spawn the built CLI on a slow suite, SIGTERM mid-body, assert clean dir and exit 143; SIGINT variant asserts 130
- `ailang test .` debris regression (60 vs 120, the V6 scenario, planted file)

**Manual testing:**
- Kill -9 mid-run in a scratch package → observe PUB024 in `pkg quality` and `publish --dry-run` refusal → next `ailang test --package .` sweeps

**Regression-surface tests:** one per Walk Inventory row — each walk's output must be identical
clean vs debris.

## Deferred Decisions

- Exact PUB024 badge wording and whether `pkg quality --json` carries the file list — agent may choose (keep the code + remedy; list at most ~5 names, then a count)
- Registry/GC for the `.ailang/cache/compile/modules/_namedtest_body_*` cache entries (found in V10): they are hidden-dir cache artifacts, never shipped — leave to the cache's own lifecycle; agent may add the pattern to cache eviction if trivial
- Whether `docs.go`/`examples.go`/`compile.go` repo-level walks also skip the pattern — agent may add the predicate where cheap, but it is NOT required for this fix
- Signal-handler granularity (single handler for the whole `test` command vs per-runner) — agent may choose, one handler is expected

## Non-Goals

- **Virtual / in-memory module loading** to avoid the temp body entirely — requires reworking the loader's relative-import and manifest discovery path; large, separate effort (Future Work)
- **Handling SIGKILL / power loss in the producer** — impossible by definition; that is why Layer B exists
- **Other tools' temp patterns** (`.ailang-fmt-*`, `.activation-*`, `.review-bundle-*`) — dot-prefixed, git-ignored, and not `*.ail`-suffixed, so they neither stage in git nor enter tarballs (V11); no action
- **Changing named-test semantics** — FoldTestBody, assert lowering, and the pipeline contract are untouched

## Timeline

**Week 1** (12 hours):
- Phase 1: consumer immunity + PUB024 (4h)
- Phase 2: producer hardening (4h)
- Phase 3: signal subprocess test, MOD010 polish, scaffold + docs (4h)

**Total: ~12 hours across 1 week** (2× the naive 6h estimate per project convention)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| New `internal/testing → internal/pkg` import trips a layer rule | Medium | Verified acyclic now (V14); `make check-boundaries` gates it; predicate could move to a leaf package if the checker objects (cheap, single file) |
| PID-liveness probe false-negative (PID reused by another process) keeps debris unswept | Low | Not a correctness issue: PUB024 still refuses at publish; the 1h mtime fallback eventually collects it |
| Sweep deletes a file a user intentionally created matching the pattern | Low | The pattern is reserved tooling namespace (underscore-prefased transient); PUB024 message names the deletion path; sweep prints what it removed |
| PUB024 hard-gates a fleet CI that currently publishes with debris | Medium | Intended (reporter asked "ignore or refuse"); remedy is one `rm`; gate text says exactly that; exclusion keeps tarball/hash deterministic if someone bypasses the gate |
| Signal handler adds latency or deadlocks in JSON output mode | Low | Handler only drains a mutex-guarded list and exits; no I/O beyond one stderr line |

## Related Documents

**Implemented (may inform design):**
- `design_docs/implemented/v0_10_0/m-dx-package-test.md` — package-level test runner (`--package` mode) that the named-test executor backs
- `design_docs/implemented/v0_40_0/m-pkg-quality-ladder-sprint-plan.md` — the PUB-gate ladder and the "ONE quality report behind pkg quality / publish / validator" doctrine this fix extends
- `design_docs/implemented/v0_30_0/m-ailang-fmt.md` (V19 survey) — repo lore that ad-hoc temp-file+rename is the common pattern; this doc adds the missing "signal-safe removal" leg

**Planned (check for overlap):**
- `design_docs/planned/ailang-core-triage/pkg-quality-smoke-output-and-staging.md` — adjacent quality work (smoke output surfacing, staging of non-`assets/` data files). Distinct: that doc is about what quality *shows* for legitimate files; this one is about refusing *illegitimate transient* files. No coverage overlap (duplicate gate: search + read, V12).

## References

- [Design Axioms](/docs/references/axioms)
- Report context: sunholo/relativity 0.5.0 prep, sunholo-data/ailang-packages branch `relativity-0.5.0-m1`; binary v0.51.0-24-g4460d91b9-dirty
- `internal/testing/executor.go:259-288` — producer mechanism and its documented source-dir constraint

## Verification Log

Probes run 2026-10-02 at workspace HEAD `4460d91b` (the reporter's commit) with the installed
`ailang` v0.51.0 binary (`b99dd25`) on linux. Code claims are code-reads at HEAD; consumer-surface
claims are live reproductions with a planted
`_namedtest_body_1234567890.ail` (copy of the test module, mode 0600) in a scratch package
`repro/pkg` (`ailang.toml` + `core.ail` + `core_test.ail` with 60 named tests).

**V1 — Producer mechanism (code-read, executor.go:282):** `os.CreateTemp(sourceDir,
"_namedtest_body_*.ail")` with `sourceDir = filepath.Dir(e.modulePath)` when a module path exists.
The in-code comment (executor.go:259-267) states WHY the file must sit next to the source:
relative sibling imports and `ailang.toml`/`ailang.lock` discovery. "Move to os.TempDir" would
break package tests with intra-package imports.

**V2 — Cleanup is defer-only (code-read):** the single cleanup is `defer os.Remove(pipelineFilename)`
(executor.go:288; `defer os.RemoveAll(sourceDir)` at :277 covers only the no-module-path fallback).
No `signal.Notify` exists anywhere in the test command path — `grep -rn "signal.Notify" internal/ cmd/`
returns 11 sites (daemon, coordinator, lsp, eval_suite, missions, chains, apiserver, rig_gate,
microrag), none reached by `ailang test`. Go defers do not run on default SIGTERM/SIGINT death,
and no Go code runs on SIGKILL.

**V3 — Debris counts as a source file (code-read + live):** the body file is written as
`baseSource + "{ " + folded + " }"` (executor.go:246-258), so it carries the original
`module …_test` line; `check.HasModuleDeclaration` (check/package.go:263) then returns true and
`check.DiscoverPackageSources` (check/package.go:228) files it under sources.

**V4 — Mode 0600 (code-read):** `os.CreateTemp` creates 0600; the subsequent
`os.WriteFile(…, 0644)` cannot change the mode of an existing file — matches the report's
"mode 0600".

**V5 — `pkg quality` count inflated (live repro):** planted debris → `compile: ✓ 3 files`;
without → `compile: ✓ 2 files`. Same mechanism as the report's 19-vs-18.

**V6 — `ailang test .` re-runs debris (live repro, NEW symptom):** with debris planted,
`ailang test .` reported `120 tests: 0 passed, 120 failed` vs `60` clean (runTestsV2 walks every
`*.ail`, test.go:31). Also observed: every named-test run in a relaxed dir prints a spurious
`WARNING MOD010 … canonical path '_namedtest_body_<n>'` naming the ephemeral file → Phase 3
polish item.

**V7 — Tarball ships debris (code-read + report):** `CreateTarball` includes any path with
`.ail` suffix (tarball.go:58-63); no exclusion for the pattern exists. Reporter measured
49,824 B vs 47,372 B for sunholo/relativity 0.5.0.

**V8 — Smoke staging ships debris (code-read):** `shouldStageFile` returns true for any
`filepath.Ext(rel) == ".ail"` (publish_validator.go:178-192).

**V9 — Content hash depends on debris (code-read):** the publish hasher hashes every `*.ail`
under the dir (hasher.go:24-37), so debris changes the provenance hash of an otherwise identical
tree.

**V10 — Compile-cache side effect (observed):** `.ailang/cache/compile/modules/_namedtest_body_<n>`
entries accumulate under the package's hidden `.ailang/` cache dir. Hidden-dir, no `.ail` suffix
→ never staged by git-default tooling nor included in tarballs; left to cache lifecycle (Deferred).

**V11 — Other temp patterns are not exposed on this surface (negative-existence, grep):**
`grep -rn "CreateTemp" internal/ cmd/` — the only `*.ail`-suffixed temp file created inside a
source dir is executor.go:282. All other patterns (`.ailang-fmt-*`, `.activation-*`,
`.review-bundle-*`, `.tmp-`) are dot-prefixed and/or live in `os.TempDir`, so they are git-ignored
and cannot match the `*.ail` tarball rule.

**V12 — Duplicate/coverage gate (docs search):** `ailang docs search` (SimHash + neural-fallback)
on "namedtest tempfile leak", "test runner temp file cleanup", "pkg quality tarball publish
package files" — no doc covers transient test bodies or signal cleanup;
`pkg-quality-smoke-output-and-staging.md` read and confirmed distinct (smoke output surfacing /
unstaged data dirs). `grep -rln "temp file\|SIGTERM\|signal handler" design_docs/` — no coverage.

**V13 — Error code PUB024 unallocated (grep):** `grep -rn "PUB024" internal/ cmd/ docs/
design_docs/` → empty. Allocated ceiling is PUB023 (quality.go:315-320: PUB022 = AGENT.md
overlap warn, PUB023 = export-overlap info).

**V14 — Import direction for the shared predicate (code-read):** `internal/pipeline` imports
`internal/pkg` (package_resolver.go:11, canonical_json.go:11, effect_ceiling.go:10);
`internal/pkg` imports neither `internal/pipeline` (quality.go:21 comment exists precisely to keep
that cycle out) nor `internal/testing` (grep of its imports: config, iface, proctree, schema,
strutil, testutil). `internal/testing` already imports `internal/pipeline` (executor.go:15-22), so
`internal/testing → internal/pkg` is acyclic and adds no new layer.

**V15 — Dispatch site for the signal handler (code-read):** `handleTest` dispatches to
`runPackageTests` / `runTestsV2` at commands_language.go:307-309, after config assembly and before
any body runs — the correct install point; both runners `os.Exit` on completion, so the handler
must be installed before dispatch.

**Note on producer-side live repro:** the installed binary is `b99dd25` (older than the
reporter's `4460d91b9-dirty` build); in our runs its named-test path did not leave an observable
package-dir temp body (polling ~900k listdir samples during a killed run showed nothing, and the
MOD010 warnings indicate its pipeline used the same `_namedtest_body_*` module id). The
source-dir temp body at executor.go:282 is present at the workspace commit `4460d91b`, which is
what the reporter built; the mechanism is therefore verified by code-read at that commit plus the
reporter's first-hand repro, and all consumer symptoms were reproduced live on the binary.

## Future Work

- Virtual module loading for named-test bodies (eliminate the temp file entirely — loader refactor;
  would also remove the MOD010 noise class and the cache-id churn of V10)
- A general "transient artifact" namespace reservation doc for tooling that writes next to user
  sources (pattern: `<toolprefix>_<pid>_`, dot-hidden, never `*.ail`) so this class of bug stops
  being rediscovered
- `ailang pkg doctor`-style one-command hygiene sweep (stale bodies, empty unstaged dirs, cache size)

---

**Document created**: 2026-10-02
**Last updated**: 2026-10-02

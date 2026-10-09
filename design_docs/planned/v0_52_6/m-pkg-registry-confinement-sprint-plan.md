# Sprint Plan: M-PKG-REGISTRY-CONFINEMENT

Refs #1607 #1608

**Design:** [m-pkg-registry-confinement.md](m-pkg-registry-confinement.md)
**Target:** v0.52.6 · **Priority:** P0 · **Duration:** 3 days (24 engineering hours)
**Status:** Implementation complete; independent evaluation pending. D3 and D5 ruled by Mark on 2026-10-08 via PR #1697.
**Sprint ID:** M-PKG-REGISTRY-CONFINEMENT

## Summary

Close the registry network/cache-write escape under `AILANG_AGENT_POLICY`, add an operator-provisioned read-only `AILANG_PACKAGE_ROOT`, and detect registry content drift against the lock. Preserve unconfined package workflows. Track only existing issues #1607 and #1608; include the exact text `Refs #1607 #1608` in the implementation PR body. Do not create another issue.

## Current status and capacity

Read both issue bodies and comments through GitHub's API on 2026-10-08: both open, zero comments. The handoff records live verification on origin/dev `658ff76a3`. This checkout is on `coordinator/task-83f42c57`, clean before planning, at `35c4dfb3`, version v0.52.5. Code reads confirm `pkg_docs` fetches/extracts on miss, `RegistryCacheDir` creates directories, the lock validator skips registry sources, and the pipeline emits only a warning on hash failure. No implementation milestone is complete.

The skill's seven-day velocity analysis found one reachable design-document commit and no usable LOC metrics. The repository is shallow; historical throughput and completion rates cannot be measured reliably here. Do not treat old changelog LOC examples as recent velocity. Use the approved design's three-day estimate (already twice its naive estimate), with 600 planned changed lines: 230 implementation, 320 tests, 50 documentation. Planning capacity is 200 LOC/day, an assumption rather than a measured velocity. Security fixtures and full validation, rather than LOC, determine completion.

## Design freeze and implementation assumptions

The handoff approves previous design work, but the design still has unchecked human decisions D3 and D5. Planning can proceed; execution must record their rulings instead of silently self-approving them.

- D3: proposed behavior is exclusive root resolution when `AILANG_PACKAGE_ROOT` is set. No HOME registry-cache fallback, even when the root is missing or incomplete. Apply this consistently to docs, loader, resolver, and hashing. The design's “root read → HOME read” wording must be interpreted as alternatives based on whether the variable is set.
- D5: the estimate uses D4's existing warning behavior for hash drift. A hard failure requires the human D5 ruling and tests at the pipeline caller. Until that ruling is recorded, M3's severity acceptance remains pending. Warning-only verification detects poisoning but does not prevent compilation against changed content; do not claim otherwise.
- Provisioned `lock` must succeed without an index fetch. Verify manifest/index needs for transitive registry dependencies before editing; if offline resolution cannot meet A2, stop and update the design rather than introducing a hidden fetch or fallback.
- A3's no-write claim applies to package reads and HOME registry state; separately account for declared lock output and policy-tool's private compile/prompt cache. Test read-only roots with tree snapshots, since chmod alone does not prove absence of writes when tests run as root.

## Registry reuse audit

Executed `ailang pkg search confinement`, `registry`, and `content-hash`. Confinement and content-hash returned no packages; registry returned `sunholo/registry_validator@0.1.2`. Inspected it with `ailang pkg info` and `ailang pkg docs`: it validates packages for publishing and generates metadata, using AILANG IO/FS effects. It cannot enforce the Go CLI/library's authority boundary before network access or change Go package resolution. CLI warned the binary may be stale; the search is discovery evidence, not proof about current source behavior.

| Milestone | Decision | Reason |
|---|---|---|
| M1 | none | Core Go registry network/cache authority must be guarded in its owning library. |
| M2 | none | Host configuration and compiler package resolution require core Go changes. |
| M3 | none | Reuse existing Go ContentHash and lock validation; publishing validator is not the compile-time trust boundary. |
| M4 | none | Regression checks and repository documentation use existing local tools. |

## Milestones

### M1 ✅: Registry confinement gate (~280 LOC)

**Estimate:** 140 implementation + 140 tests; Day 1, 8 hours.
**Dependencies:** design freeze before execution.
**Files:** new `internal/pkg/confinement.go`, `internal/pkg/confinement_test.go`, `cmd/ailang/pkg_docs_confinement_test.go`; update `internal/pkg/registry.go`, `internal/pkg/registry_test.go`, `internal/pkg/resolver.go`, `internal/pkg/resolver_test.go`, `cmd/ailang/pkg_docs.go`, `cmd/ailang/pkg_install.go`.

Implement `Confined` and canonical refusal. Guard FetchIndex/FetchPackage before HTTP or in-memory index reuse. Make RegistryCacheDir pure, add gated EnsureRegistryCacheDir, and migrate every extraction-to-cache writer. Add named docs/resolver misses. Audit all callers, including operator tooling, for reliance on directory creation.

**Examples/fixtures:** temp HOME, restricted policy, locally served index/tarball, and provisioned cache fixtures in the test files; reuse working package fixtures rather than inventing syntax.

- [x] A1: confined uncached pkg-docs refuses, makes zero HTTP requests, and does not create HOME/.ailang.
- [x] A2 negative: confined lock requiring download or transitive index refuses without registry writes or network calls.
- [x] Both network methods and EnsureRegistryCacheDir refuse independently; pure path computation never creates directories.
- [x] A5: unconfined pkg-docs, install, and lock still fetch/extract successfully against a local HTTP fixture.
- [x] Error text names confinement and operator provisioning; focused pkg/CLI tests pass.

**Risk:** missed writers bypass the helper. Mitigate with a complete caller/extraction sweep and explicit zero-network counters.

### M2 ✅: Read-only package root (~150 LOC)

**Estimate:** 60 implementation + 90 tests; Day 2, first 6 hours.
**Dependencies:** M1 and D3 ruling.
**Files:** `internal/config/paths.go` and config path tests; `internal/pkg/registry.go`, `internal/pkg/loader.go`, `internal/pkg/resolver.go`, loader/resolver tests; `cmd/ailang/pkg_docs.go` and confinement tests; `internal/policytool/cli_ops_test.go` if needed for child environment propagation.

Register EnvPackageRoot/PackageRoot and pathVars. Add PackageDir; route all registry reads, including resolver reads for provisioned lock, through the same exclusive root selection. Docs enumerates versions under the selected root without creating it. Verify existing child environment propagation preserves host-set root.

**Examples/fixtures:** 0555 operator root with vendor/name/version packages and AGENT.md, plus a different HOME-cache copy that would succeed if incorrectly used.

- [x] A3: confined check and docs read the provisioned root successfully; root and HOME tree snapshots are unchanged.
- [x] Configured missing root/package refuses loudly despite a valid HOME cache; no fallback and no root creation.
- [x] Unset root preserves HOME-cache reads and existing git-cache behavior.
- [x] A2 positive: fully provisioned confined lock succeeds offline, with only its declared lock output.
- [x] Loader tests cover the consumers used by check/ai-check/test/iface/tree and child environment retains the root.

**Risk:** resolver still fetches an index for transitive dependencies. Cover a transitive fixture and surface any design gap before proceeding.

### M3 ✅: Registry content verification (~120 LOC)

**Estimate:** 30 implementation + 90 tests; Day 2 last 2 hours and Day 3 first 2 hours.
**Dependencies:** M2 and recorded D5 ruling.
**Files:** `internal/pkg/lockfile.go`, `internal/pkg/lockfile_test.go`; `internal/pipeline/package_resolver.go` and pipeline package-resolution tests as required by D5.

Extend ValidateContentHashesFrom to hash registry entries using PackageDir; retain path dependency behavior and exclude git sources. Reuse ContentHash; no new hashing algorithm. Emit package identity and locked/current hashes safely, including malformed short hash input without panic. Apply exactly the recorded D5 severity at the pipeline consumer.

**Examples/fixtures:** matching provisioned lock, mutated .ail source, missing registry directory, empty and short hash cases in existing lockfile/pipeline test fixtures.

- [x] A4: tampering with cached or root-provisioned registry source is detected at check time.
- [x] Matching registry entries pass; missing directory and malformed hashes produce useful errors without panic; empty hash validation remains rejected.
- [x] D5-selected stderr and exit status are asserted through check, not only the helper.
- [x] Existing path-dependency behavior remains covered and root-only hashing cannot select the HOME copy.

**Risk:** warning detection is confused with rejection. Record the chosen severity and its limits in tests, documentation, and the implementation report.

### M4 ✅: Documentation and security regression sweep (~50 LOC)

**Estimate:** 50 documentation; Day 3 final 6 hours.
**Dependencies:** M1, M2, M3.
**Files:** `docs/docs/reference/env-vars.md`, `docs/docs/reference/std-package.md`, `SECURITY.md`, current-version changelog per repository convention. Verify CLI/help documentation against existing commands; no new command surface. Record results in the sprint state JSON and the implementation PR.

Document root layout, exclusive resolution, confinement refusal, provisioning, and D5 severity. Correct the design's example path inconsistency by using one operator root path throughout runnable instructions. Provisioning examples are shell commands and existing verified package fixtures; no new language feature examples are needed.

- [x] Document the environment variable via existing config/docs tooling and verify the generated row matches pathVars.
- [x] Build the binary and rerun #1607's policy-tool repro with empty HOME; record refusal, no HTTP, no persistent registry creation.
- [x] Run operator-root, root-miss/no-fallback, tamper, provisioned-lock, and unconfined smoke scenarios on the built binary.
- [x] Run the requested focused Go tests, make test-core, make fmt, make lint, make check-boundaries, and make check-file-sizes; record environment failures explicitly. Full make test is delegated to CI because this executor has RAM-backed temporary storage.
- [x] All A1–A5 results and D3/D5 rulings appear in the sprint report; implementation PR body contains `Refs #1607 #1608`.

**Risk:** chmod is ineffective under root or global tests contact live services. Use content/tree snapshots and local HTTP fixtures; live repro should not need an actual registry response after confinement is implemented.

## Execution and evaluation handoff

Progress file: `.ailang/state/sprints/sprint_M-PKG-REGISTRY-CONFINEMENT.json`. All milestones start uncompleted. Use focused Go package tests while developing, followed by the repository gates once the implementation is complete. Cover every added security branch; no repository-wide coverage percentage is claimed without measurement.

This is a planning-only coordinator stage. Route these artifacts to sprint-executor after plan approval, the D3/D5 rulings, and execution authorization; then sprint-evaluator assesses the design and A1–A5. No executor was started during planning. No issue was created or closed. Preserve `Refs #1607 #1608` in the eventual PR body.

## Tooling notes

`gh` and `jq` are unavailable here. Issue bodies/comments were read using Python's standard-library GitHub API client. The existing sprint JSON generator and validator were inspected; both depend on jq. The populated JSON is therefore written and structurally validated with Python instead of modifying those tools or claiming their checks ran. No implementation tests were run for this documentation-only planning stage.

## Execution outcome

All four milestones completed with local milestone commits. Mark’s D3 root-only and D5 immediate hard-fail rulings were applied. Historical planning assumptions above describe the original planning stage; current evidence is in the sprint state (`.ailang/state/sprints/sprint_M-PKG-REGISTRY-CONFINEMENT.json`) and the implementation PR. SQLite/CGO test failures are environment limitations, not regressions.

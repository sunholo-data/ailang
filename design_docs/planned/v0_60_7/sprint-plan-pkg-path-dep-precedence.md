# Sprint plan for M-PKG-PATH-DEP-PRECEDENCE

Implement the [approved design](m-pkg-path-dep-precedence.md) for v0.60.7 (task-207a73de). An equal-version transitive registry pin must resolve to the direct path or git source. A different exact version must fail with a structured conflict that identifies the actual requesting package. Keep the manifest and lock formats unchanged.

**Estimate** — 3 working days, about 18 hours including a 4-hour buffer, and about 460 lines changed including tests, deletions, and docs. Medium risk. External audit access and D6 review turnaround are outside the estimate.

## Baseline and schedule

The checked-out resolver sorts dependencies and deduplicates by name and version. Only registry pins enter the root direct-version map. `pkgLockCommand` saves the lock before warning about a shadowed path. The design’s V1–V3 already reproduce the problem in a mock registry. Current sizes are 425 lines for `resolver.go`, 546 for `resolver_test.go`, and 671 for `pkg_commands.go`. Add focused companion tests in `internal/pkg/resolver_local_claims_test.go` to keep the fixtures together. Use `internal/pkg/resolver_local_claims.go` if helper growth warrants a companion file.

The seven-day velocity script saw one documentation commit in this shallow checkout and could not calculate a diff. Changelog LOC examples are historical. The estimate uses the approved design’s phase breakdown and the current code surface, with roughly 150 changed lines per planned day as a planning figure. No coverage baseline is available here because Go is absent.

Day 1 covers M1, day 2 covers M2 and starts M3, and day 3 finishes M3 and M4. Within each day, work through the numbered milestones in order and checkpoint the resolver, CLI, and audit work separately.

## Registry reuse

The search gate applies to package-like capabilities. All four milestones modify Go resolver or CLI behavior, exercise it with fixtures, or document and audit its rollout. There are no registry search terms or candidate packages to inspect.

| Milestone | Decision | Reason |
|---|---|---|
| M1 | none | Reuse the Go resolver, manifest loader, git resolver, and mock registry harness. |
| M2 | none | Reuse the Go conflict type, resolver entry point, and CLI lock tests. |
| M3 | none | Reuse checked-in lock, git confinement, staging, and validator fixtures. |
| M4 | none | Use the repository’s packages guide, changelog fragments, and committed manifests and locks. |

## Milestones

### M1: Local claim resolution (~155 LOC)

**Dependencies:** None

Allow about 5 hours. Preload and validate path and git claims before descending into sibling subtrees, using the existing sorted walk and retaining loaded manifests and git metadata for owner resolution. A compatible edge must use the claimed source even if it encounters the claim early. Preserve child-before-parent ordering and cycle detection when satisfying pending edges, including an edge back to a claim in progress. Track the owner and source through materialization so later equal-version local claims reuse the winner and incompatible claims still conflict. Keep legacy registry tarball path declarations on their existing path-to-registry conversion route, then apply the claim check to the version obtained from the index. Reuse `resolveGitSource` with the previous lock so preloading respects confinement and `AILANG_PACKAGE_ROOT`.

- [ ] `TestResolve_DirectPathDepWinsOverTransitiveRegistryPin` proves the local hash and portable path are used in both sibling sort orders and the registry copy is never fetched.
- [ ] `TestResolve_PathClaimSurvivesSiblingSubtree` covers two path siblings and a transitive registry pin from the earlier subtree.
- [ ] Equal local claims produce one owner entry, unequal claims return a structured conflict, and cycles through pending claims are detected. All claimed owners are materialized on success.

### M2: Conflict attribution and resolver invariant (~110 LOC)

**Dependencies:** M1

Allow about 3 hours. Extend `VersionConflictError` with `DirectSource`. Populate local root version and source for errors both before and after claim materialization, and cite the carrier as the requesting package. Add the typed `ErrShadowedLocalDep` check at `ResolveDependenciesFrom` for absent or wrong-source root path and git entries. Remove the post-save call in `pkg_commands.go` and delete `pkg_lock_warnings.go`. Update the CLI warning tests to exercise the new contract and retain existing lock-preservation assertions.

- [ ] `TestResolve_PathDepMismatch_AttributedConflict` checks the root clone at 0.4.1 against a carrier pin at 0.4.0, correctly attributed fields and source text, and both remediation suggestions.
- [ ] Invariant unit checks reject missing and wrong-source entries for root path and git dependencies. Command integration checks failure exits nonzero, creates no new lock, and preserves existing lock bytes.
- [ ] A successful equal-version path lock reports the path source without the old warning.

### M3: Lock and caller regression checks (~140 LOC)

**Dependencies:** M2

Allow about 4 hours. Add the direct-git case using existing fixture conventions, the legacy committed-lock migration in `pkg_lock_check_test.go`, and repeated-resolution assertions. Check publish smoke staging and registry validation through their existing entry points.

- [ ] A direct git dep wins against an equal transitive registry pin, with the expected hash and revision. Confinement and package-root carry-forward tests pass.
- [ ] A previously shadowed lock reports source drift with `ailang lock --check` and stays byte-identical. `ailang lock` rewrites it to the path source, and the next check succeeds.
- [ ] Ten repeated resolutions produce identical results and lock bytes. Existing legacy conversion, registry-only exact conflicts and deduplication, relative-path portability, publish staging, and validator checks pass.

### M4: Documentation and rollout audit (~55 LOC)

**Dependencies:** M3

Allow about 2 hours plus review turnaround. Update `docs/docs/guides/packages.md` with a manifest and CLI example for testing a patched clone at the carrier’s pinned version, plus the mismatch and one-time drift behavior. Add a dated `changelogs/unreleased/YYYY-MM-DD-pkg-path-dep-precedence.md` fragment beginning with a level-three heading. Use the mock-registry test topology as the runnable example. The work writes no AILANG modules.

Count committed manifest/lock pairs and direct path declarations with matching-name non-path lock entries in `ailang-packages` and stapledons-godot `sim/`. Record audited revisions, denominators, counts, affected packages, and any access gaps in a checked-in audit report alongside this plan. Use read-only inspection and report inaccessible repositories as unknown.

- [ ] The packages guide and changelog explain precedence, exact-version conflicts, and refreshing affected locks.
- [ ] The audit report provides reproducible counts or explicitly records missing evidence and is available for the D6 ruling.
- [ ] Run the focused validation below and record results before the D6 gate.

## Validation

Run `go test ./internal/pkg ./cmd/ailang ./cmd/registry-validator` for the relevant milestones and `make test-changed` against the implementation base. Format touched Go files and run `make lint`, `make check-boundaries`, `make check-file-sizes`, and `make check-changelog`. CI runs the full suite. Replay V1–V3 against the newly built binary with a mock registry and an isolated cache, checking output, status, source, and hash. Coverage is measured by the regression cases above and the existing CI coverage gate.

## Decisions and handoff

D1–D5 carry forward from the approved design. Equal versions use the local source, exact version mismatches fail, the manifest surface stays the same, claims precede descent, and the invariant runs before lock writes. Cross-version overrides remain deferred.

D6 remains open in the design’s **High-Impact Decisions → Design Freeze**, which explicitly calls for audit-informed sign-off on the one-time `lock --check` drift. After M4, attach the audit evidence and check for a recorded ruling. If sign-off is still open, **STOP: commit current work, report `BLOCKED: D6 sign-off on one-time lock drift`, and end the run**. Coordinator plan approval governs the transition to execution.

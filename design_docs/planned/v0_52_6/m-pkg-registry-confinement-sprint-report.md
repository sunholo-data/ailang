# M-PKG-REGISTRY-CONFINEMENT — implementation report

Refs #1607 #1608

Implementation complete on `coordinator/task-d86575b1`; independent coordinator evaluation pending. No push or PR creation was attempted.

## Authorization and plan

Mark ruled D3 and D5 on 2026-10-08 via PR #1697: the configured package root is exclusive, and registry content mismatches fail check immediately. The user authorized execution with focused checks instead of full `make test`.

The named planning branch was deleted upstream. Its PR head was fetched as `origin/plan-1685` from `refs/pull/1685/head`; both supplied plan artifacts were already present and identical to that head. The populated four-milestone JSON was validated with Python because jq is absent. Session/checkpoint scripts were replaced by their safe equivalent checks because they invoke the prohibited full suite.

## Milestones and evidence

- M1 (`e10c4166e`): registry index/package/metadata fetches refuse before HTTP or index reuse. RegistryCacheDir computes paths without creating them. All three extraction-to-cache writers call the guarded EnsureRegistryCacheDir. The confinement regression failed before implementation; subsequent package and local-server docs/install/lock tests passed.
- M2 (`faa74682a`): config registers AILANG_PACKAGE_ROOT; loader, resolver and docs use the root exclusively. HOME and legacy lock Path fallbacks cannot override it. Provisioned versioned transitive dependencies lock offline. Legacy registry path dependencies that require an index refuse. Tests failed before implementation and passed afterward.
- M3 (`d991da23e`): registry source is hashed before path dependency validation. Typed registry trust errors stop the pipeline; path drift still warns. Full hashes avoid malformed-short-hash panics. The real CLI tamper test exited 0 before implementation and exits 1 afterward.
- M4: generated environment docs, package/security documentation, changelog, child-root propagation and traversal tests, live smoke scenarios, and requested validation gates completed. The root snapshot test verifies unchanged file bytes, modes, and tree structure.

## A1–A5 built-binary smoke results

Built with `make build`; exercised against a local HTTP fixture and temporary HOME/project/package roots. No live registry was contacted.

| Acceptance | Result |
|---|---|
| A1 | Real policy-tool pkg_docs dispatch under empty network allowance and deny-write policy returns failure naming AILANG_AGENT_POLICY; HTTP request counter stays zero; HOME/.ailang remains absent. |
| A2 negative | Confined lock on an uncached version refuses; no HTTP request or registry write. Transitive index refusal also has a focused resolver regression. |
| A2 positive | Fully provisioned confined lock succeeds offline and writes ailang.lock; HOME registry state remains absent. Unit fixture includes versioned transitive dependencies. |
| A3 | Check importing a registry module and docs both succeed from a chmod-0555 root; root bytes and HOME are unchanged. The CLI regression additionally compares the complete root tree and modes. |
| D3 root miss | Check fails for a missing configured root and never creates it. Loader/docs/hash tests also supply valid HOME copies and verify no fallback; loader tests cover a legacy lock Path. |
| A4 | Mutating a registry .ail source makes real check exit 1, naming the dependency and locked/current hashes. HOME-cache mutation, root mutation, missing content, empty/short hashes and path-warning masking are covered. |
| A5 | Unconfined docs, install --no-bin and lock succeed against a local server and extract into the HOME registry cache. Existing install-with-bin/shim registry tests also pass in the requested selection. |

## Validation and environment limitations

| Check | Result |
|---|---|
| Requested `go test ./internal/pkg/... ./internal/config/... ./cmd/ailang/ -run 'Pkg\|Registry\|Confin\|PackageRoot\|ContentHash'` | pkg/config pass; CLI selection fails only in the three coordinator SQLite tests below. All implementation tests pass, including the built-CLI check and local-server workflows. |
| `make test-core` | All selected packages except internal/effects pass; eight SQLite brain-store tests fail because CGO is unavailable. |
| `make lint` | Pass, zero issues. One earlier run was killed with exit 137; bounded retry passes. |
| `make fmt` | Pass; no unrelated formatting changes. |
| `make check-boundaries` | Pass. |
| `make check-file-sizes` | Pass. |
| Child environment regression | `go test ./internal/policytool -run '^TestPolicyTool_ChildEnvCarriesPolicy$'` passes, including host root propagation. |
| Full `make test` | Deliberately not run; CI owns this gate under the user's RAM-backed /tmp constraint. |

Go 1.26.8 was already installed but absent from PATH. Make and the repository-pinned golangci-lint binary were extracted into a temporary tools directory. No compiler was installed. CGO_ENABLED=0 and no C toolchain means go-sqlite3 is a stub; these failures are environment limitations, not regressions:

- CLI: TestCheckRegistryCanDispatchSurfacesLoadError, TestCheckRegistryCanDispatchRejectsEmptyRegistry, TestCheckRegistryCanDispatchAllowsRejectionOfAnOrphanedAgent.
- Core: TestNewBrainStore_Unregistered, TestBrainStore_TwoTier, TestBrainStore_Promote, TestBrainStore_Stats, TestBrainStore_NilTier, TestBrainStore_WithEmbedder, TestBrainStore_SearchThreeTier, TestBrainStore_SearchByEmbedding.

Resource-limited runs used GOFLAGS=-p=2, GOMAXPROCS=2 and GOMEMLIMIT=768MiB or 1GiB. A concurrent CLI linker was killed once; its isolated retry passed. Abandoned temporary outputs from cancelled duplicate baseline builds were removed to recover RAM-backed storage.

## Scope and follow-up

FetchMetadata receives the same guard as the two specified network entry points, closing the remaining direct registry HTTP route. PackageDir rejects traversal components in configured-root identities. Git dependencies and std/package.assetPath runtime asset lookup retain their existing behavior. Operators must protect the root and lock from agent writes; a lock digest does not authenticate a lock the agent can modify.

The implementation PR body is saved in [m-pkg-registry-confinement-pr-body.md](m-pkg-registry-confinement-pr-body.md). Full-suite CI and independent sprint evaluation remain downstream coordinator checks. The design and plan stay together in planned until that evaluation.

## Evaluation handoff

The required sprint-executor inbox handoff was attempted with the built CLI but refused because no live coordinator agent serves the sprint-evaluator inbox. The coordinator must trigger independent evaluation from the final completion markers; no evaluation verdict is claimed.

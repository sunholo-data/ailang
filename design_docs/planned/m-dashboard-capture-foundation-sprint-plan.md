# M-DASHBOARD-CAPTURE-FOUNDATION — exporter recovery and honest initialization

**Status:** Implementation complete; formal evaluation/promotion held by the unfiltered test gate
**Authorized:** Mark, 2026-09-08: “please continue to sprint plan then execute,” following the production audit and its proposed first repair.
**Design basis:** [Live audit F1](../dashboard-live-audit-2026-09-08.md), [recovery plan](../dashboard-recovery-plan-2026-09-08.md).
**Branch/worktree:** `sprint/dashboard-capture-foundation`, isolated from unrelated work on `dev`.
**Target:** next patch after v0.35.2; no release/deployment performed by this plan.
**Planning allowance:** 2 engineering days, approximately 650 implementation/test LOC. The velocity script was reviewed/run; its historical changelog matches and last-commit statistic do not establish current engineering throughput. This is a scope estimate, not a measured productivity claim.

## Contract and scope

An HTTPS OTLP destination without an explicit port must be handled by the configured HTTP exporter. A collector unavailable during process initialization must not disable that destination for the lifetime of the coordinator. Initializer and daemon logs must distinguish configured destinations, registered exporters, and unverified delivery. A fresh task-linked span must survive the production OTLP conversion/storage path in an isolated integration fixture.

Remove the startup TCP reachability gate instead of maintaining a second URL/port/availability implementation beside the SDK. OTLP HTTP exporters initialize without a network handshake and already own endpoint parsing, retry and transport. Enforce a three-second export-cycle bound and preserve the two-second shutdown bound. This enables later batches to recover after an outage; it does not promise durable replay of previously failed batches. Durable spool/fleet capture remains the next recovery sprint.

Add an explicit initialization report returned by `InitWithStatus`; keep existing public initializer signatures as compatibility wrappers. Report exporter registration, warnings and delivery-unverified semantics. Coordinator and server startup use the returned report instead of recomputing success from environment variables. GCP initialization timeout is a named degraded state, not successful registration. Test GCP constructor behavior through a local injected factory rather than cloud credentials. Initialization failure must clean up owned resources; shutdown ownership belongs to providers, avoiding duplicate exporter shutdown.

This sprint does not migrate databases, change language semantics, backfill records, dispatch agents, approve work, alter UI routes, or certify global fleet coverage. Production rollout and fresh live-cohort acceptance follow review of the concrete changes. They are tracked separately from the local implementation milestones so local tests cannot be mislabeled production proof.

## Verification log

| Premise | Evidence |
|---|---|
| HTTPS destination is skipped at startup | Live audit F1: coordinator logs on Sept 8 at 05:26, 05:04, 03:47 and 03:11 UTC |
| All OTLP initialization paths share the bad check | `rg isOTLPReachable internal/telemetry`: OTLP init plus trace/metric dual-export gates |
| Startup success is inferred from config | `cmd/ailang/coordinator_lifecycle.go` and `server.go` use `IsDualExportEnabled/IsGoogleCloudEnabled/IsEnabled` after init |
| SDK owns URL/TLS/path parsing | Pinned `otlptracehttp` v1.46.0 module `options.go`; `WithEndpointURL` and environment handling |
| Existing bounds must survive | `internal/telemetry/otel.go`: three-second exporter timeout and two-second combined shutdown deadline |
| Independent existing init API consumers exist | `rg telemetry.Init cmd/ailang`: compiler/check/repl/eval/messages callers; preserve compatibility |

## Milestones

### M1: Recover OTLP destinations after startup outages (~190 LOC)

**Dependencies:** None.
**Files:** `internal/telemetry/otel.go`, `internal/telemetry/otel_recovery_test.go`.

- [x] Regression test fails on the original startup gate, then passes after its removal.
- [x] Both OTLP-only and dual-export initialization retain the configured HTTP exporter without a reachability preflight.
- [x] A collector started after initialization receives a later span; initialization does not contact the collector.
- [x] HTTPS URL without explicit port follows HTTP transport semantics in a deterministic test.
- [x] Export and shutdown remain bounded; invalid transport configuration cannot be reported as successful delivery.

### M2: Report actual exporter registration (~280 LOC)

**Dependencies:** M1.
**Files:** `internal/telemetry/otel.go`, initialization report/helpers/tests under `internal/telemetry/`, `cmd/ailang/coordinator_lifecycle.go`, `cmd/ailang/server.go`.

- [x] An explicit initialization report distinguishes disabled, registered and degraded destinations; registration is labeled delivery-unverified.
- [x] Both daemon startup consumers use that report rather than environment-based success inference.
- [x] Tests cover GCP timeout/failure and successful registration alongside OTLP without real cloud credentials.
- [x] Failed initialization and late GCP constructor completion clean up owned resources; successful shutdown does not double-close exporters.
- [x] Existing `Init`, `InitOTLP`, `InitDual`, `InitGoogleCloudTrace` signatures remain compatible.

### M3: Verify linked delivery and document rollout (~180 LOC)

**Dependencies:** M1, M2.
**Files:** integration tests under `internal/telemetry/` or `internal/observatory/`, `docs/docs/guides/telemetry.md`, `changelogs/v0.32-current.md`, this plan, sprint/evaluation JSON.

- [x] Real OTLP HTTP protobuf emission reaches the production receiver/conversion path and stores trace/task/chain/stage identities in a temporary database.
- [x] Recovery and failure tests use isolated loopback fixtures, no production writes and no external model calls.
- [x] Relevant tests, race checks, lint/build and architecture boundaries pass; any full-suite environmental/baseline failure is recorded rather than hidden.
- [x] Documentation distinguishes registration from receipt, explains outage/data-loss limits, and gives a concrete staged production verification procedure.
- [x] Independent sprint evaluation records acceptance evidence and any remaining rollout gate.

## Evidence-driven implementation adjustments

The regression tests exposed two defects beyond the startup preflight, both within
this sprint’s acceptance contract:

- The pinned SDK’s HTTP timeout covers individual requests; a retrying batch lasted
  about 30 seconds. Configure the batch processor and periodic metric reader with
  three-second export deadlines as well as the existing transport timeout.
- The SQLite backend’s task-linked aggregation transaction omitted explicit chain
  and stage columns. The real receiver roundtrip decoded those IDs correctly but
  read them back empty. Preserve those two columns in that transaction. This does
  not claim Firestore parity or change its implementation.

Independent review also found the coordinator discarded its shutdown callback.
Normal coordinator exit now flushes providers, with trace flush taking priority
inside the shared deadline. These changes are necessary to meet M1/M3; no dashboard
UI work or database migration was added.

## Day-by-day execution

Day 1: baseline targeted checks; red/green startup-recovery tests; remove preflight; implement honest initialization report and resource lifecycle handling. Day 2: receiver roundtrip, race/regression tests, lint/build/boundaries, documentation and independent evaluation. Continue sequentially under the user's execution instruction without re-asking at each milestone.

## Validation and operations

Use existing Go/Make targets. The sprint JSON creation/prerequisite scripts unconditionally import GitHub inbox messages and invoke the CLI's local retention routine; the user explicitly asked to leave the inbox untouched. Construct JSON from the documented schema and run the existing validator; perform their relevant checks directly. Do not run message-import side effects.

Baseline telemetry package first; tests that invoke the CLI must remain sandboxed to protect the real home-directory observatory from its unrelated auto-retention startup behavior. Use narrowly scoped escalation for loopback/cache permissions when required. No new `.ail` code or compiler/effect changes, so language fixture and conflict-surface sections are not applicable.

## Rollout gate (separate from local completion)

- [ ] Review built binary/commit and deployment diff; then deploy a canary/approved revision using existing infrastructure.
- [ ] Confirm startup lists only actually registered exporters and says delivery unverified before evidence.
- [ ] Observe a fresh authorized coordinator task: emitted trace and resource/span IDs match the persisted cloud task/chain/stage. Distinguish no matching record from failed retrieval.
- [ ] Observe restart and collector recovery with bounded failure reporting; inspect export errors and receiving-side timestamps.
- [ ] Record revision/digest, query IDs, latency and result. Continue broader capture/API/approval work under the recovery plan; no claim of global reliability yet.

## Axiom review

A1 0, A2 +1, A3 0, A4 0, A5 +1, A6 +1, A7 +1, A8 0, A9 0, A10 +1, A11 +1, A12 +1 = +7. No language semantics or authority changes. Gains come from explicit failure/registration state, bounded verification, safe lifecycle handling and preserving existing initializer composition.

## Execution results (2026-09-08)

Implementation commit: `935c5934d` on `sprint/dashboard-capture-foundation`.
No release, merge, push or production deployment was performed. The original
checkout and inbox were left untouched. Added implementation/test lines: 755;
this is diff size, not a measured engineering velocity.

| Check | Result |
|---|---|
| Original startup regression | Red: HTTPS registration and both late-collector cases failed before removal of the gate |
| Initial integrated tests | Exposed retry-cycle timeout and SQLite provenance loss; corrected and rerun |
| `go test ./internal/telemetry ./internal/observatory -count=1` | Pass: 5.426s / 1.531s |
| Same packages with `-race -count=1` | Pass: 7.522s / 3.590s |
| Independent focused review/test run | Pass; no remaining concrete implementation blocker |
| `make lint` | Pass, zero issues |
| `make check-boundaries` | Pass |
| Build and pi extension tests | Pass through `make test` |
| Unfiltered repository suite | Not green; four environment-sensitive failures remain (listed below) |
| Repository suite with explicit four-test exclusions | Pass, exit 0 on implementation commit |
| Sprint JSON validator and `git diff --check` | Pass |

The full test command runs under a nested macOS sandbox denying writes to the
real `/Users/voightkampff/.ailang`, with `AILANG_STORAGE=local` and
`AILANG_MESSAGES_STORE=local`. This protects production-like local observatory
data from the CLI’s unrelated startup auto-retention. The four exclusions are:

- `TestMemWatchdogKillsAllocator`
- `TestMemWatchdogKillsGrandchild`
- `TestProcessGroupRSSSamplesGroup`
- `TestPackageDir_RegistryRuntimeResolution`

The first three require process-group memory visibility unavailable in this run;
the fourth writes and removes a fixture under the actual home package cache, which
is deliberately protected. Their source paths are unchanged from the base commit.
They are recorded as environment-sensitive failures, not proven passing tests.
The correctly quoted exclusion expression was
`Test(MemWatchdogKills|ProcessGroupRSSSamplesGroup|PackageDir_RegistryRuntimeResolution)`.

Logs are local, not committed: `/tmp/dashboard-sprint-red.log`,
`/tmp/dashboard-sprint-packages-v2.log`, `/tmp/dashboard-sprint-race-v2.log`,
`/tmp/dashboard-sprint-lint-v2.log`, `/tmp/dashboard-sprint-boundaries.log`,
`/tmp/dashboard-sprint-full-test.log`,
`/tmp/dashboard-sprint-full-filtered.log`, and
`/tmp/dashboard-sprint-full-filtered-v2.log`.
The first full run predates the final fixes. The second full run on the commit
(`full-filtered.log`) had an ineffective exclusion expression, excluded no tests,
and confirms exactly the four unchanged failures; telemetry and observatory pass.
The final correctly quoted filtered run exits 0.

The independent evaluation artifact is
`.ailang/state/evaluations/eval_M-DASHBOARD-CAPTURE-FOUNDATION_round_1.json`.
Its strict rubric requires unfiltered `make test` exit 0; filtered success does
not satisfy that requirement. Therefore the implementation milestones are locally
met but sprint promotion remains paused. Re-run unfiltered tests in an isolated
runner with normal process inspection and disposable user data before promotion.
The separate production rollout checklist above remains entirely pending.

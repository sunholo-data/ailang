# Sprint: M-MAINTENANCE-EVAL-BROKER

**Status:** In progress; execution approved by Mark on 2026-10-26 (“execute sprint”).
M1 host feasibility is the active stop gate.
**Design:** [host-side confined tools](m-maintenance-eval-broker.md).
**Planning request:** Mark, 2026-10-26: “ok lets try to continue”.
**Base:** `40fbb90d5` on `origin/dev`, including merged core PR 1940
(`d8e47684cb0b8bb0d72392d10d801fb2df0a30dc`). Companion package PR 162
is merged at `9ac50f6a25d587fc3a6a935a87991bbaecdd0fc1`.
**Duration:** 3–5 engineering days if feasibility passes; initial planning envelope.
**Risk:** High. A working socket does not establish child ownership or oracle isolation.

## Outcome and boundary

Qualify a real, no-model pi SDK session that uses native filesystem tools inside
the outer kernel boundary and calls a host broker for usable no-network bash.
The same qualification must cover native candidate execution, bounded evidence
and cleanup before retained-source checkpoints are finalized.

This sprint prepares a usable protected instrument. It does not authorize
inference, a priced campaign, registry publication, a release, or changes to the
standing eval rotation. The existing unconditional live-dispatch refusal remains
in place throughout this sprint. Qualifying a no-model chain and enabling a
model lane are separate decisions; a later launch design must specify the
admission binding and actual model/provider/budget configuration.

No language semantics or motoko core changes. The implementation belongs in the
existing pi extension/executor seams and maintenance adapter. Use the locked
sandbox runtime 0.0.71 and actual pi SDK 0.85.1. Do not update personal/global pi.

## Current evidence and remaining uncertainty

- The adapter sprint has 7/12 milestones complete and remains in progress.
  Its independent evaluation is 81/100 with a full-sprint FAIL/blocked verdict.
- Native tool and direct-host sandbox controls pass, including positive controls
  for owned application work. Nested macOS sandbox initialization fails with
  `sandbox_apply: Operation not permitted`. This is the broker's specific target.
- `proctree.Configure` owns a process group; `ReapWorkspace` finds processes by
  cwd. Neither establishes ownership of a descendant after `setsid` plus a cwd
  change. Do not treat repeated process-table polling as an ownership guarantee.
- `maintenanceInvokeGrade` currently uses `CombinedOutput` and reads its report
  without a byte ceiling. Checkpoint reads also need enforced per-file and total
  limits. Bounds must apply before allocation, including raced/growing files.
- The merged CI gates pass. Advisory Sonar coverage was 57.5% on new code:
  opt-in native paths and embedded JS/TS coverage are not currently imported.
  Coverage reporting needs to measure those actual tests; exclusions are not a fix.

The first milestone is a stop gate. If the tested host primitives cannot provide
both a usable sibling launch and escaped-descendant ownership, record the failed
qualification and stop. Do not build the remaining transport or expand into a
VM/container/platform project under this plan.

## Registry reuse and code audit

On 2026-10-26, `ailang pkg search` was run for `maintenance`, `sandbox`,
`executor`, and `process`. `pkg info` and `pkg docs sunholo/maintenance_bench`
were read. The published benchmark is 0.1.0; the merged taskpack/ledger source
must therefore be pinned separately. The `sandbox` result (`social_dynamics`)
does not implement host isolation. Process results included `duckdb` and
`external_backend`; their documented role is subprocess/JSON adaptation, not
stage capability transport or descendant ownership. They are not dependencies
for this host-side Node/Go work.

M1–M5 and M7: **none** — reuse existing pi/SDK/proctree primitives, extending
their maintenance seams only. M6: **contribute** to the existing
`sunholo/maintenance_bench` companion research assets if protocol controls need
changes; keep analysis in native AILANG. No new benchmark runner or package.

The following files were read for this plan:
`internal/executor/pi/containment.go`, `containment_test.go`,
`internal/executor/eval_containment.go`, `internal/proctree/proctree.go`,
`reap_cwd_unix.go`, `internal/eval_harness/maintenance_grade.go`,
`maintenance_runner_stage.go`, `maintenance_workspace.go`, the maintenance
Node tests, `make/eval.mk`, and the maintenance workflow.
The installed locked SDK's macOS code was also read: specific Unix socket grants
emit `subpath` filters, and process fork/exec are allowed. These are code-path
observations, not proof that the proposed chain or cleanup works.

## Frozen control limits

M1 records a versioned fixture with these initial limits. A change requires a
new protocol/qualification identity and re-running affected controls.

| Surface | Limit and required outcome |
|---|---|
| Active stages/commands | One stage and one active command per broker |
| Stage requests | At most 64 accepted bash requests; bounded rejection count (8) then close |
| Input frame / command | 64 KiB UTF-8 frame; 32 KiB UTF-8 command; enforce while reading |
| Output | 16 KiB chunks, 2 MiB combined stdout/stderr per command |
| Command wall time | At most 120 seconds, also capped by remaining stage/campaign time |
| Startup / cleanup | 5 seconds each; expiry is a named infrastructure failure |
| Connections | One authenticated client; bounded unauthenticated handshake with 5-second deadline |
| Local socket | Short owned path; validate actual platform byte limit before launch |
| Candidate stdout/stderr | 2 MiB combined; overflow terminates owned execution and invalidates grading |
| Grade JSON | 1 MiB; over-limit report is an infrastructure failure, not candidate failure |
| Snapshot | 2 MiB per declared file, 16 MiB total, at most 256 declared paths |
| Control manifest/receipt input | 1 MiB per document; explicit unavailable/error on overflow |

Existing smaller stage/campaign limits take precedence. No implicit retries,
unbounded backlog, report repair or successful empty-output fallback. Returned
bytes may be a bounded prefix, but overflow is explicit and never graded as a
complete successful command. Tests cover exactly-at-limit and one-byte-over.
These application/output bounds do not claim a general host CPU/memory quota.

## Milestones

Each milestone targets one focused executor session (60–90 minutes). Estimates
are change-size budgets, not measured implementation speed. Split a milestone
before execution if the actual mechanism cannot fit that budget.

### M1: Actual host feasibility and stop gate (~280 LOC)

**Dependencies:** None. **Estimate:** 80 source/fixture + 200 test LOC.
**Ownership:** `tools/pi-extensions/maintenance-broker-feasibility.test.mjs`,
a small feasibility helper only if needed, and new validation artifacts under
`design_docs/validation/maintenance-broker/`. Reuse existing owned application
fixtures; no new `.ail` modules in this planning slice.

- [ ] Run an actual credential-free outer process using pi SDK 0.85.1 and the locked kernel wrapper; exchange a bounded request over the single intended socket and execute a sibling no-network SDK child.
- [ ] Positive controls build/run the existing owned AILANG and Python fixtures. Protected file/network controls fail; a nearby ungranted socket is unreachable.
- [ ] A controlled descendant attempts `setsid`, changes cwd, closes inherited stdio and outlives its leader. Prove mechanical escape prevention or a concrete ownership mechanism that still terminates and awaits it on normal exit, timeout and controller death. Protect an unrelated bystander process.
- [ ] Record host/OS/architecture, Node/pi/SDK/runtime identities, wrapper/policy/source hashes, exact commands and exit evidence. No skipped control can qualify a platform.
- [ ] On any failed property, write `qualified:false` with the named blocker and leave M2–M7 unstarted; no general reaper rewrite or host platform expansion.

### M2: Strict stage protocol (~270 LOC)

**Dependencies:** M1 passed. **Estimate:** 130 source + 140 tests.
**Ownership:** `tools/pi-extensions/maintenance-broker-protocol.mjs` and its test;
embedded copies follow the existing asset verification mechanism.

- [ ] Bind schema/version, stage ID, policy/workspace identities and random stage capability; all controller values are immutable after startup.
- [ ] Reject duplicate JSON keys, unknown fields, invalid UTF-8, oversized/incomplete frames, replay, cross-stage IDs, wrong capabilities and non-monotonic/duplicate request IDs before starting a child.
- [ ] Only `bash` is supported; requests cannot choose cwd, environment, policy/runtime paths, executable transport or network authority.
- [ ] Apply the frozen request/connection/rejection limits and typed exit/timeout/cancel/overflow/transport results. Boundary and fragmentation tests pass.

### M3: Bounded host command owner (~320 LOC)

**Dependencies:** M1, M2 passed. **Estimate:** 150 source + 170 tests.
**Ownership:** `tools/pi-extensions/maintenance-broker.mjs`,
`maintenance-broker.test.mjs`, and a separate child-owner module if required.

- [ ] Use the exact lifetime mechanism qualified in M1. A serial command uses the immutable stage cwd and existing tools-mode SDK policy as a sibling of outer pi.
- [ ] Enforce command time/output bounds during streaming; kill, join and observe owned execution before another request or socket cleanup.
- [ ] Tool/candidate environment is explicitly allowlisted; no provider credential, loader hook, capability or controller proxy setting is inherited.
- [ ] Parent death, leader exit, broken client, broker loss, malformed transport, overflow and cancellation produce named outcomes without surviving owned children or affecting the bystander.

### M4: Trusted pi transport and stage lifecycle (~260 LOC)

**Dependencies:** M2, M3 passed. **Estimate:** 120 source + 140 tests.
**Ownership:** `tools/pi-extensions/maintenance-fence.ts`, new broker client
module, `internal/executor/pi/containment_broker.go` and focused tests;
keep `containment.go` and `pi.go` below the repository file-size limit.

- [ ] Real SDK bash uses the authenticated broker transport; native read/write/edit/search guards and the outer kernel boundary stay active.
- [ ] Ready/version/policy handshake occurs before the no-model outer SDK process starts. The capability is absent from model-visible tool results, candidate files, child environment and process argv.
- [ ] The socket/policy directory cannot be replaced by candidate code; tool/candidate processes cannot use the broker socket or inference proxy. Check actual SDK-generated socket grants.
- [ ] Nil-containment executor behavior remains unchanged, and both Execute paths still reject live containment before model resolution, CLI probing or inference.

### M5: Bounded grading and retained evidence (~300 LOC)

**Dependencies:** M1, M3 passed. **Estimate:** 140 source + 160 tests.
**Ownership:** new `internal/eval_harness/maintenance_limits.go` and focused
tests, existing grade/workspace/JSON helpers, companion `candidate_grade.py`
and its controls only where necessary. Reuse the same child ownership policy.

- [ ] Bound controller subprocess output while reading, grade JSON before allocation/parsing, and source snapshots by file/count/total budget; include growth and symlink-race cases.
- [ ] Both native candidate languages run under the qualified owner. Their entire owned execution is joined before scratch deletion or snapshot finalization.
- [ ] Output/report/snapshot overflow stays distinguishable from candidate compile/test failure. Partial diagnostic output is labeled, and spent attempts remain visible.
- [ ] Carry actual failed source into the next fresh session; never restore a gold starter after a grading or execution failure.

### M6: Real full-chain no-model qualification (~240 LOC)

**Dependencies:** M4, M5 passed. **Estimate:** 60 fixture/source + 180 tests.
**Ownership:** split maintenance integration/OS tests, companion
`research/real-world-maintenance/taskpacks/v1/` controls, and a new dated
qualification record. Do not edit historical qualification records.

- [ ] Actual outer SDK/native tools → broker → sibling tool/candidate chain passes normal application, protected oracle, native path-race, hardlink, recursive-search and live reference-server controls.
- [ ] Try credential/process-metadata reads, other-stage sockets, broker replay, inference-proxy reuse and all failure/cancellation paths with independent positive controls.
- [ ] Run both families/languages and the existing eight-trajectory/24-stage retained-source control under the qualified no-model chain. Compare identities/verdicts with the native AILANG ledger; provider usage/economics remain explicitly unavailable.
- [ ] Bind every qualification to the exact executed inventory, including known reference copies/Git stores, runtime exceptions, policy, tool sources and compiler. A missing or drifted item fails qualification.

### M7: Evidence, coverage and independent review (~180 LOC)

**Dependencies:** M6 passed. **Estimate:** 60 CI/docs + 120 tests/reporting.
**Ownership:** maintenance CI/make target, test coverage artifact mapping,
validation README, changelog fragment, this sprint state and adapter state.

- [ ] Import actual native Go and JS/TS coverage into the appropriate reports, preserving source/embedded-asset identity. Report measured coverage and exclusions accurately; do not weaken the 80% advisory target or change branch protection.
- [ ] Focused touched-package tests, core tests, lint, embedded asset drift, boundaries, file-size/changelog gates and no-model instrument controls pass. All supported-platform required PR checks pass.
- [ ] Independent sprint evaluation checks the actual ownership and authority contract; a score cannot override a failed containment control. Record unavailable review rather than inventing a pass.
- [ ] Reassess the original adapter M3/M5/M8/M11 criteria individually with linked evidence. Leave any unmet criterion incomplete; qualification never automatically opens inference.
- [ ] Deliver source/review artifacts and a proposed next-launch checklist: actual model/provider, held-out cases, independent usage completeness, finite cost/request/token bounds and explicit launch approval. No campaign runs in this sprint.

## Schedule, estimates and stop behavior

| Day | Planned work | Change-size budget |
|---|---|---|
| 1 | M1 feasibility; stop immediately if ownership/topology fails | 280 LOC |
| 2 | M2 protocol and M3 host command owner | 590 LOC |
| 3 | M4 transport and M5 evidence bounds | 560 LOC |
| 4 | M6 full chain and M7 delivery/review | 420 LOC |
| 5 | Qualification/review buffer only | Included above |

Total estimate: **1,850 LOC**, including source, tests, fixtures and CI/docs;
3–5 days, with no observed LOC/day claim. The velocity helper was tried and
exited 141 after historical changelog output, so it supplies no usable rate.
The prior adapter diff was 4,833 LOC across 31 implementation/test files in this
slice; its repeated CI/qualification corrections support a buffer, not a promise
that the smaller broker estimate is proven. Re-estimate after M1.

## Validation and examples

Use `make test-changed` for touched Go packages plus the core inner loop, and
the existing `make maintenance-adapter-check` with explicit package/SDK paths.
Extend that target with actual broker controls; never substitute a stub SDK or
skip an unsupported OS. Existing direct-host and known nested-failure controls
remain useful regressions even when the brokered topology works.

Run `make verify-maintenance-assets`, `make verify-pi-assets`,
`make check-boundaries`, `make check-file-sizes` and `make check-changelog` as
appropriate. CI owns the full suite; repeat focused checks only after changes or
new failures. Coverage uses real execution, including native integration.

Examples are the existing taskpack candidate fixtures and no-model SDK controls.
This plan adds no `.ail` code. If a later implementation change requires a new
AILANG example, load `ailang prompt` and record the active version before
writing it, then specify contracts, effects and inline tests for that module.

## Authorization and handoff

Planning and implementation of this concrete sprint are authorized by Mark
(2026-10-26: “execute sprint”). No coordinator handoff has occurred. The existing
approved adapter work does not silently authorize a new host service or an
inference campaign. Original unrelated working-tree changes remain parked.

Sprint state: `.ailang/state/sprints/sprint_M-MAINTENANCE-EVAL-BROKER.json`.
Keep the original adapter state separate; do not reset its progress or declare
the broader maintenance study complete when this infrastructure sprint ends.

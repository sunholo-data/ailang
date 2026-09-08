# M-MISSION-RUNTIME-CONTRACT: Durable Projects, Measured Workflows

**Status**: Approved project direction (Mark, attended 2026-09-07); first-slice planning/execution authorized; fleet cutover and experiment runs not started
**Created**: 2026-09-07
**Target**: staged delivery; release assignment at sprint planning
**Priority**: P0 — Mark identifies this as the foundation for downstream projects
**Estimated**: four independently reviewable slices; first-slice estimate at sprint planning, whole-project estimate after its recovery trial
**Owner**: Codex leads design, implementation coordination, verification, and progress reporting; Mark owns project policy and deployment approval
**Dependencies**: existing mission workbench, executor/coordinator interfaces, quota rollout, and communications workstream

## Problem Statement

A mission should turn an objective into accepted project outcomes over many sessions, providers,
and eventually machines. Today, controllers still translate role names into provider-specific
spawn recipes, reconcile differing instruction copies, and reconstruct interrupted work from
logs and worktrees. Each repair can add more instructions to interpret next time.

The first review found a concrete contradiction: the driver prompt requires the Agent tool,
while the current cross-provider recipe prohibits it for provider-qualified role pins. V1
iteration 340 records a Pi model rejected by the Agent model enum and a wrapper-mediated retry.
World inbox reports independently describe the same mismatch. These observations support a
shared dispatch contract, not another model-specific exception.

The quota ledger is newly deployed. Its empty local result is rollout evidence, NOT evidence
that the new guard is defective. The quota sprint explicitly distinguishes a landed gate from
active enforcement with provider capacity data.

Two questions must be separated:
1. Which guarantees must a durable runtime enforce regardless of model?
2. Which planning/review workflow buys enough quality to justify its cost for a particular task?

A written project contract answers the first. The second is an empirical question; this project
must not encode today's five-role process as the only possible workflow.

## Goals

1. A project can adopt missions through the binary without copying the AILANG shell harness.
2. Every accepted artifact has traceable inputs, actual routing, verification, and authority.
3. Restart, provider failure, and duplicate delivery resume valid work without repeating accepted
   transitions or silently discarding in-flight artifacts.
4. Operators can see progress, blockers, budget state, and the next valid action in one CLI view.
5. Workflow complexity is earned by measured outcomes, with a simpler workflow available for
   evaluation before any change to production policy.

Success is accepted outcomes per unit time and cost, plus human interventions and harness-repair
share. Iteration counts, number of roles invoked, and evaluator scores alone are not success.

## High-Impact Decisions

D1–D4 approved by Mark in the attended 2026-09-07 response: “great please git commit so we start ona clean tree then lets get this important work up and running”. This authorizes first-slice planning and execution. D5 provider-backed experiment activation remains separate.

| ID | Decision | Recommendation | Chosen by | Change cost |
|---|---|---|---|---|
| D1 | Runtime versus model responsibilities | Binary owns transitions, dispatch, deadlines, resource admission, and evidence receipts; models own task reasoning and artifacts | Mark at contract review | High |
| D2 | Workflow policy | Support full and compact workflows as explicit versioned profiles; retain production policy until comparison is reviewed | Mark | Medium |
| D3 | State ownership | Extend coordinator persistence for execution; keep existing human-decision authority until its separately approved migration | Mark | High |
| D4 | Migration | Prove one isolated work item, then one canary mission, then the fleet; each step can return scheduling to the old driver | Mark | Medium |
| D5 | Experiment authorization | Approve protocol separately from activating a production default; initial pilot caps proposed below | Mark | Low |

### Design Freeze

- [x] Approve D1–D4 project direction (Mark, attended 2026-09-07).
- [x] First slice freezes request, receipt, cancellation and adapter contracts in [the role-dispatch sprint](m-mission-role-dispatch-sprint-plan.md). Coordinator store transactions and verified artifacts remain slice 2; the local journal does not claim those guarantees.
- [ ] Approve D5 before provider-backed trial runs; freeze task corpus, route assignments, and evaluator rules.

This is the project-level design. It deliberately leaves future slices at architecture level;
the first implementation slice is expanded in the linked sprint and implemented as an opt-in command. This avoids pretending the
entire migration is already specified or repeatedly redesigning unrelated future work.

## Solution Design

### Runtime contract

A mission definition names its objective, acceptance criteria, project root, verification commands,
scope, workflow profile, approval policy, and references to routing/resource profiles. Model
assignments remain owned by the existing model-registry workstream; the mission definition does
not introduce a competing model catalog.

Each work item consists of versioned stage attempts. A stage request carries mission/work-item/
stage/attempt IDs; immutable input revision and instruction digest; role and required capabilities;
artifact expectations; deadline; resource reservation; and authority policy reference. A result
carries actual executor/model/vendor/transport/billing identity, outcome, artifact revision,
verification evidence, usage with known/unknown provenance, and causal error classification.

The coordinator persists attempts and advances them conditionally. Proposed outcomes distinguish
completed, verification-failed, transport-failed, awaiting-decision, awaiting-capacity, cancelled,
and ambiguous-external-effect. A transport failure consumes its retry allowance, not an evaluation
judgment round. A provider switch cannot erase the previous attempt or inherit a stale PASS.

The implementation must reuse or extend existing coordinator mechanisms after a transaction audit;
this document does not claim existing finalization/CAS support alone provides end-to-end exactly-once
execution. External actions can be delivered more than once. Stable idempotency keys, reconciliation,
and an explicit unknown outcome are required where transactional guarantees stop.

### Provider-independent roles

The controller requests a role through the runtime. It does not manufacture an Agent-tool wrapper
to reach a provider absent from that tool's model enum. Reuse internal/executor's interface and
capability declarations, adapting gaps only where confirmed by the first-slice audit.

Resolve capability, authority, availability, budget, then preference. Check independence against
actual model identity/vendor after fallback. Hosting the same model through another service is
transport redundancy, not an independent judge. Never silently weaken the approved independence
policy to keep work moving.

One versioned role contract is distributed to every executor. Provider-specific syntax belongs in
adapters. Long incident narratives remain reference material; executable obligations and acceptance
criteria are concise, structured, and checked by the runtime where possible.

### Durable execution and messages

Use leases with generation/fencing information for claims and renewal. An expired worker may not
publish a result over its replacement. Heartbeats indicate execution liveness; artifact and tool
receipts indicate progress. Cancellation covers the owned process tree, not processes matched by
name. Interrupted work is retained and resumed or explicitly superseded.

Messages transport requests and notifications; an unread flag alone is not a work lease or proof
of completion. Persist an input before acknowledging consumption. Use a durable outbox with bounded
delivery and stable event IDs. GitHub reports and dashboards are projections of recorded state.
Human directives retain the existing authenticated provenance and decision semantics until the
communications workstream's migration is approved. Do not create a second decision writer.

Cloud Run implementation remains in the separate thread. This project defines the IDs, artifact
references, claim/result semantics, and executor boundary that remote execution must preserve.

### Resource admission

Consume the quota workstream rather than replacing it. Distinguish warming-up, unknown-capacity,
actively-enforced, exhausted, and unavailable measurement. Runtime results feed usage directly;
post-hoc model-authored bookkeeping is not the intended long-term accounting path.

Reserve resources before concurrent dispatch and reconcile after each attempt. Keep metered dollars,
subscription usage, local compute, and unknown values separate. Do not assume token counts map
linearly to provider quota units. Admission rules for unknown measurements must be explicit in the
approved resource policy. Preserve a controller/recovery reserve and bound concurrency fleet-wide.

### Proposed CLI experience

These commands are design examples, not claims that the current binary supports them:

```text
ailang mission add . --name my-project
ailang mission start my-project
ailang mission status my-project --explain
ailang mission pause my-project
ailang mission resume my-project
```

`add` creates a project-local definition and registers its location; it does not silently start
work. `start` validates tools, role compatibility, verification, authority, and resource policy.
Folders use explicit artifact snapshots; repositories additionally support revision/worktree and
merge checks. A scheduler invokes the same bounded runtime entry point on any supported host.

Separate portable project definition from machine-local installation and secrets. Reconcile this
with the existing fleet registry in the onboarding slice; do not silently move ratified registry
or role configuration in the first implementation slice.

## Workflow Experiment

**Hypothesis:** For bounded tasks, an integrated agent that investigates, plans, and implements,
followed by independent evaluation, can match the quality of separate designer/planner/executor
sessions with lower wall time and fewer handoffs. This is unproven; stronger models do not establish it.

| Arm | Workflow | Shared protections |
|---|---|---|
| A: full | Controller + separate designer, planner, executor, independent evaluator | Frozen brief, scope, acceptance battery, authority, isolation, budget |
| B: compact | Controller + one agent investigating/planning/implementing, independent evaluator | Identical protections and evaluator contract |

The compact arm still records intent, assumptions, and verification. It removes mandatory separate
sessions/documents per task; it does not remove project requirements or independent review.
A third adaptive arm is deferred until this comparison has interpretable results.

### Initial pilot protocol

- Six frozen task briefs: two bounded defects, two multi-file changes, two operational/recovery tasks.
  Include a small held-out task outside the mission harness. Choose tasks and hidden acceptance
  checks before either arm runs; no live mission owns these experimental workspaces.
- Start each arm from the same immutable base in separate disposable workspaces. Exclude artifacts,
  tests, answers, and transcripts created by the other arm. Historical tasks require removal of
  solution-bearing history/docs from the supplied workspace; otherwise label them contaminated.
- First compare orchestration with the SAME primary model across A's authoring roles and B's
  integrated role; same harness/tool access and settings. Independent evaluator identity and
  acceptance tests are fixed across both. Mixed-model fleet comparisons come later and are
  reported separately, since changing models and workflow together confounds attribution.
- Randomize arm order within each pair and distribute pairs across available capacity windows.
  Record model/version, instruction digest, quota state, tool availability, and realized route.
- Evaluate anonymized artifacts against a frozen functional and adversarial battery. Record
  evaluator judgment separately from deterministic checks. Tests do not substitute for judgment,
  and a high score cannot override a failed hard acceptance condition.
- Include handoff/retry/repair time and ALL roles' costs. Record first-pass acceptance and final
  acceptance after the same bounded repair allowance (one revision). Transport failures remain
  in intention-to-treat totals; a capability mismatch is separately classified, not quietly dropped.
- Proposed initial caps: $20 total metered spend, 90 minutes per arm, one arm active at a time,
  and existing approved subscription ration/reserve rules. Unknown quota measurements do not
  authorize unrestricted subscription use. Stop at any resource cap and report incomplete pairs.
- No automatic publishing, merging, fleet reconfiguration, or outbound reports in trial workspaces.
  Unexpected mutation outside the sandbox or acceptance of an invalid artifact suspends the pilot.

### Decision rule

The six-pair pilot checks feasibility and exposes failure modes; it cannot establish general quality
parity. Report paired successes/failures, intervention counts, end-to-end wall time, metered cost,
quota usage, and missing measurements. Reuse `ailang eval-paired` where its input schema fits;
first prove the mapping preserves task/trial identity rather than forcing mission data into it.

A larger confirmatory sample and a quality non-inferiority margin must be agreed from pilot variance
and product risk before promoting a default. Do not choose thresholds after seeing the confirmatory
result. Any simpler default is scoped to task class/model family, not declared universally better.
Revalidate after substantive model, harness, or workflow changes. Retain full design review for
irreversible authority/storage/protocol decisions unless evidence supports a separately approved change.

## Delivery Slices and Acceptance

1. **Role dispatch contract:** one isolated work item reaches the required external evaluator without
   a controller-invented wrapper; actual routing and versioned instructions are recorded. Deterministic
   fixtures exercise primary success, incompatible capabilities, fallback, and independence collision.
2. **Durable lifecycle:** kill/restart at every stage boundary, duplicate completion, stale worker,
   approval-during-retry, and ambiguous remote effect preserve accepted progress and decision authority.
3. **Resources and communication:** integrate quota measurements/reservations and bounded durable
   delivery; simulate capacity drought and message outage without losing work or storming retries.
4. **Onboarding and canary:** add a scratch folder and an unrelated repo through the CLI; run one
   mission canary, prove rollback and in-flight compatibility, then adopt the remaining fleet.

Run the workflow pilot as an independently reported evaluation track after its authorization; it
need not wait for all runtime slices. It must not turn this project's delivery into an unbounded
benchmark exercise. Model dispatch and persistence guarantees apply to both experiment arms.

### Files and reuse boundaries

| Surface | Intended ownership |
|---|---|
| internal/mission/ and cmd/ailang/mission_cmd.go | Mission contract, CLI, registry integration; exact files/LOC at first-slice planning |
| internal/executor/executor.go and provider adapters | Existing dispatch abstraction; targeted contract gaps only |
| internal/coordinator/ | Existing task persistence, conditional transitions, finalization, worker heartbeat, remote dispatch |
| internal/modelreg/ | Consume routing identities; coordinate with ongoing model-registry changes |
| tools/launchd/mission-control.sh and mission skill resources | Compatibility callers reduced as each binary capability replaces them |
| quota/comms designs linked below | Their existing owners retain accounting and human-channel migration scope |

## Conflict Surface

This project contract changes no language semantics and proposes no changes to cmd/ailang/exec.go
in the first slice. That file already has unrelated uncommitted work; leave it parked. If a slice
requires it or compiler/runtime semantic packages, expand that slice's conflict analysis first.

Operational compatibility is load-bearing: current role pins and authority policy, queued coordinator
tasks, in-flight mission worktrees, message receipts, and generated launchd installations must remain
interpretable across rollout. Schema changes require explicit migration and old-worker behavior.
No active mission is silently reassigned to the new runtime.

## Verification Log and Coverage Gate

Observed in the attended review on 2026-09-07; live fleet state can advance independently.

| Claim | Evidence / limitation |
|---|---|
| Agent-tool versus provider-recipe contradiction | tools/launchd/mission-control.sh prompt and .claude/skills/mission-control/resources/gate-3-route.md read; V1 iteration 340 and canonical World messages corroborate |
| Provider-facing skill copies differ | diff of .agents/skills/mission-control/SKILL.md and .claude/skills/mission-control/SKILL.md; differences include namespace and route instructions |
| Coordinator reuse exists | Read internal/coordinator/task_status_cas.go, finalization_ledger_store.go, cloud_dispatcher.go, heartbeat.go, and agent_registry.go; not an audit of all crash boundaries |
| Executor reuse exists | Read internal/executor/executor.go; Execute/ExecuteStreaming, capabilities, task environment, deadlines, and budget fields present |
| Quota rollout is incomplete by design | quota sprint M2/M4 landed; M3 capacity and M5 reserve remain pending in reviewed plan; local CLI returned empty usage |
| Existing mission package baseline | go test ./internal/mission -timeout 60s passed in the preceding review; no end-to-end/provider run claimed |
| Installed topology check | ailang mission doctor returned four missions/no drift; does not establish workflow or budget health |
| Distinct from workbench/comms | Both documents explicitly exclude the rest of the driver runtime; this contract owns that excluded lifecycle/dispatch boundary |
| Distinct from generic workflows | Implemented configurable invoke/trigger/approval support is reused, not reimplemented; this project adds mission recovery contract and measured profile selection |
| Related search | CLI results were unrelated; direct neural search reported 0 embeddings and model=fallback-simhash. Neural duplicate thresholds do not apply to these fallback ranks; inspected scope boundaries above establish the distinction |

## Related Documents

- [Mission loop workbench](v0_36_0/m-mission-loop-workbench.md): registry/deployment machinery reused; runtime explicitly excluded there.
- [Mission communications](v0_36_0/m-mission-comms-into-the-binary.md): bounded channels and decision projections; preserve its authority decisions.
- [Quota rationing](m-quota-rationing-routing.md): resource policy and new measurement; do not rebuild.
- [Mission ELO routing](m-mission-elo-routing.md): model assignment from evidence; this experiment compares workflow structure, a separate variable.
- [Generic coordinator workflows](../implemented/v0_7_0/m-coord-generic-workflows.md): existing configurable workflow foundation.
- [Unified telemetry](../implemented/v0_33_2/m-mission-loop-unified-telemetry.md): existing mission evidence substrate.

## Axiom Compliance

Scores assess the proposed design, not proof of implemented guarantees.

| Axiom | Score | Reason |
|---|---|---|
| A1 Determinism | +1 | Recorded inputs and conditional transitions; model outputs remain explicit external observations |
| A2 Replayability | +1 | Attempts and artifact revisions survive restart |
| A3 Effect Legibility | +1 | Dispatch and external actions have receipts |
| A4 Explicit Authority | +1 | Policy and decision provenance survive retries |
| A5 Bounded Verification | +1 | Acceptance, retry, and experiment limits are explicit |
| A6 Safe Concurrency | +1 | Claims, fencing, and admission precede execution |
| A7 Machines First | +1 | Structured state replaces narrative reconstruction |
| A8 Minimal Syntax | 0 | No language syntax change |
| A9 Cost Visibility | +1 | Realized usage and unknown measurement are distinct |
| A10 Composability | +1 | Workflow profiles use shared executor/coordinator contracts |
| A11 Structured Failure | +1 | Transport, judgment, resource, and authority outcomes differ |
| A12 System Boundary | +1 | Portable requests/results define local and future remote execution |

**Net: +11.** No proposed negative on A1/A3/A4/A7. Approval and implementation evidence remain required.

## Success Criteria

- [ ] First isolated lifecycle passes its acceptance and fault-injection battery.
- [ ] Every accepted artifact resolves to its actual route, exact input, and independent verification.
- [ ] Duplicate/stale delivery cannot regress approval or accept an obsolete artifact.
- [ ] CLI distinguishes actual project progress, human decisions, transport failure, and quota warmup.
- [ ] Pilot reports paired evidence and uncertainty without an unsupported general superiority claim.
- [ ] Scratch project onboarding and canary rollback verified before fleet cutover.
- [ ] Relevant tests, architecture-boundary checks, and documentation pass for each implemented slice.

## Non-Goals and Deferred Decisions

Cloud deployment, changing language semantics, repricing models, autonomously relaxing existing
approval rules, and a wholesale rewrite of the coordinator are outside this project contract.
A general DAG editor and learned scheduler are deferred until a simpler runtime earns their need.

The implementer may choose private helper names, fixture organization, CLI presentation, and bounded
internal retry timings within the approved contract. Durable schema, authority migration, evaluator
independence relaxation, fleet activation, and extra experiment spend require explicit decisions.

## Risks and Delivery Discipline

The primary risk is reproducing shell complexity in Go. Each slice must retire a concrete repeated
interpretation or recovery burden and demonstrate that removal. Merely adding a CLI wrapper is not
acceptance. Existing P0 product incidents can still be handled by their owners; this project does
not commandeer unrelated dirty work or running missions.

Detailed sprint estimation follows first-slice contract approval and repository velocity review.
No whole-project delivery date or productivity improvement is claimed before those measurements.


### Recovery increment — 2026-09-07

[Recovery sprint](m-mission-recovery-sprint-plan.md) implements repository-safe driver pinning
and opt-in coordinator-backed attempt admission/fencing. It preserves the four live loops and
weekly-thread authority. The full durable lifecycle still requires artifact acceptance,
reconciliation resolution, quota reservations, and canary adoption; this increment is not
an assertion that those parent delivery criteria are complete.

## Next increment proposal — binary-owned work-item iteration (7 September 2026)

**Status:** Scope and additive-schema direction approved by Mark, attended 2026-09-07:
“yep please proceed”, in response to the design approval request. Sprint planning authorized;
execution subsequently authorized by “lets go! spritn execute”. **Implementation:** M1–M5 implemented and verified at `77dc7287e`; M6 remains partial pending a concrete approved Docs item. Live canary activation remains separate. See [verification evidence](../verification/mission-iteration/checks.md). **Priority:** P0. **Target:** next runtime
increment; release assignment at sprint planning. **Estimate:** 5–8 engineering days including
fault tests and canary preparation, provisional until sprint decomposition.

### Outcome and scope

One explicit, authorized work item progresses through its remaining design/planning/execution/
evaluation stages under binary control. A restart resumes from recorded evidence; it never
guesses that an interrupted provider did nothing. Completion means a validated candidate ready
for the project's existing landing process. It does not mean merged or released.

This is a bounded work-item iteration, not yet the entire autonomous mission loop. Backlog
observation/selection, publication and retrospective reporting remain at the existing boundary.
The canary starts from an approved sprint so it can exercise execution and independent evaluation
without spending its first run generating another infrastructure design. Future work can supply
the same input through an autonomous selector.

Keep this specification in the parent contract: it elaborates slices 2–4 rather than creating
another competing mission-runtime design. General `mission add`, non-Git folder snapshots,
distributed workers, fleet-wide reservations, workflow A/B trials, and notification outboxes
remain subsequent work. The first implementation targets a local Git repository and one active
work item per mission, using one registered coordinator SQLite database.

### Current behavior and replacement boundary

| Responsibility | Inspected implementation | Next increment |
|---|---|---|
| Schedule, driver pin and project root | Registry/render/apply; shell pin helper | Retain scheduler and pin selection; validate the work repository independently |
| Mission preflight and pick | Mission-control skill Gates 0–2, interpreted by controller | Explicit work-item input with frozen source revision, scope and authority references |
| Role selection and provider probes | Shell role ladders and controller dispatch | Consume model-registry routes; freeze ordered candidates and record actual selection |
| Role transitions | Controller interprets Gate 3 and invokes skills | Binary advances only when prerequisite evidence and authority are present |
| Quota admission | Shell invokes quota CLI before probes | Shared binary policy runs before candidate health/probe and again before execution |
| Execution lease and receipt | `mission_role_state.go`, coordinator `mission_attempts` | Reuse attempt keys/fences and move orchestration glue behind a shared service |
| Failure retry | Shell can rerun a controller after transient/quota signatures | No controller-wide retry on this path; pre-execution fallback only |
| Accepted artifact | Role report explicitly leaves `ArtifactVerified` false | Separate immutable evidence and conditional stage acceptance |
| Heartbeat, watchdog, slot verdict | Shell process/heartbeat/log inspection | Persist phase/liveness, bounded stage deadlines, owned-process cancellation |
| Landing, decisions and external reports | Existing mission policy and communications workstream | Keep existing authority; return a candidate plus local structured summary |

For a canary invocation, the compatibility driver must branch before legacy role probes and
controller launch, invoke the binary once, and return its result without legacy transient retry.
The legacy path remains available for missions not opted in. The new branch retires its use of
controller-authored spawn recipes, role verdict text parsing and whole-controller replay.
It must not run the new dispatcher inside a legacy controller and call that lifecycle adoption.

### Decisions proposed for this slice

| Decision | Recommendation | Authority / change cost |
|---|---|---|
| State ownership | Add work-item and acceptance records in the existing coordinator SQLite store; retain attempt state meanings | Mark at design approval / high |
| Smallest usable runtime | One explicit work item, sequential remaining stages, existing full-workflow policy | Mark at design approval / medium |
| First canary | One unowned, approved Docs mission item; stage it for a single controlled invocation | Mark at canary review / low |
| Acceptance boundary | Validated candidate; existing landing process retains merge/CI authority | Mark at design approval / high |
| Interrupted execution | Preserve and require reconciliation; no automatic retry after dispatch | Existing parent decision / high |
| Helper layout and diagnostic rendering | Implementer chooses within these contracts | Agent / low |

Design freeze: [x] approve this bounded scope and additive schema; [x] specify the work-item,
stage-result and acceptance schemas in [the iteration sprint plan](m-mission-iteration-sprint-plan.md)
(pending sprint execution approval). Canary activation additionally
requires a named task/base/route/caps/rollback record; current fleet state must be refreshed then.
No new permission is inferred from an old message or from an executor's PASS.

### Work-item input and CLI

Commands implemented by M-MISSION-ITERATION (live activation remains pending):

```text
ailang mission iterate --work-item FILE --dry-run
ailang mission iterate --work-item FILE
ailang mission status NAME --work-item ID --json
ailang mission resume NAME --work-item ID
ailang mission cancel NAME --work-item ID --version N
```

The versioned input names the registered mission, stable work-item ID, expected repository
identity, base commit, approved brief and scope, workflow/profile version, remaining stages,
prerequisite artifact digests, authority references, verification argv lists with timeouts,
and iteration/stage resource ceilings. Resolve role assignments through `internal/modelreg`;
persist the resolved registry digest, ordered candidates and instruction bytes before dispatch.
Do not introduce a second role/model catalog in the mission manifest.

Canonicalize project/workspace paths and verify the base exists in the expected repository.
Use an isolated clean worktree from that base. Reject dirty/shared workspaces and changed inputs
for an existing ID. Keep DB, receipts and verification evidence outside the author worktree.
Resolve the DB through machine-local mission configuration; `iterate` must not accept arbitrary
alternate DB paths as a way to escape an existing work-item fence. The lower-level opt-in
`role-run --state-db` API retains its documented local scope.

`--dry-run` validates and resolves without creating a worktree, receipt, DB record, probing a
provider or claiming quota is available. `status` does not create a missing database or reconcile
state as a side effect. It exposes the phase, blocker, evidence references, lease age, deadline,
actual route, resource provenance and next valid action; never ownership credentials.

### Persistence and recovery contract

Add `mission_work_items` keyed by (mission_id, work_item_id), with input digest/body, workflow
version, phase, next stage, lease/owner generation, semantic version and blocker details.
Admission serializes at mission scope in the same transaction so different work-item IDs cannot
run concurrently for one mission. Existing `mission_attempts` remains authoritative for whether
a role was dispatched. Add immutable stage-acceptance records keyed by work item and stage;
each binds request/outcome digests, artifact commit/tree, verification receipts and authority refs.

Acquire the work-item lease before creating/claiming a role attempt. Start a child attempt only
while its parent generation is current and runnable, checked transactionally at dispatch.
Acceptance and the next-stage pointer advance in one conditional transaction that checks both
parent generation and child outcome digest. Parent cancellation atomically fences advancement
and active child attempts. A stale parent cannot launch another role after cancellation.

The work-item phase is `ready`, `running`, `validating`, `waiting`, `needs_reconciliation`,
`completed`, `failed` or `cancelled`. `waiting` has a typed reason (`quota`, `availability`,
`decision`) and optional known retry time; these are work-item states, not renamed attempt states.
Status shows these as quota-blocked, provider-blocked or decision-blocked. Never invent a reset
time when the provider observation does not contain one.

| Interruption / observation | Resume behavior |
|---|---|
| Before any child dispatch | Reclaim expired parent/prepared ownership, same immutable request; create a new receipt generation and preserve earlier receipts |
| Child still has a live lease | Report running; do not steal or launch another child |
| Child running with expired lease | Reconcile to `needs_reconciliation`; retain workspace and receipts, zero new provider calls |
| Child completed durably, acceptance absent | Validate the recorded artifact without executing that role again |
| Acceptance and next pointer committed | Continue at the recorded next stage; never repeat the accepted stage |
| Receipt says finished but DB completion is absent | Treat the result as ambiguous; filesystem evidence cannot override the DB fence |
| Validation interrupted | Rerun only declared repeatable local checks in a fresh verification worktree |
| Cancel races with completion/acceptance | Versioned transaction chooses the winner; cancellation cannot be overwritten by a late worker |

All-candidates-blocked must leave the work item waiting without consuming its permanent execution
slot. Add a recorded preflight phase before claiming a child; recheck quota after claim and just
before dispatch. If admission changes in that interval, retain an unstarted prepared attempt for
same-request recovery rather than completing it as `execution_failed`. The existing terminal
attempt contract stays closed to automatic reruns. Known execution failure ends this increment's
automatic processing; a revised work item requires explicit review and a supersedes reference.
`resume` cannot resolve ambiguous execution or promote a discarded receipt to accepted work.

One new orchestrator may reclaim an expired parent while a child remains live, but can only
observe/wait or cancel that child, not dispatch its successor. Expired-parent recovery must
inspect the child before deciding whether progress is safe. Lease renewal and actual artifact
progress remain separate signals. Use stage and iteration deadlines; do not infer a hung model
from an old coarse gate stamp. Require process-tree cancellation tests for each admitted adapter
before canary use. Cross-machine locking and power-loss durability are not claimed.

### Artifact and authority checks

Each role must produce a bounded structured result naming its request digest, input revision,
output commit, expected artifact paths and outcome. The runtime reads Git objects itself:
the commit must exist in the expected repository, descend from the stage base, include the
required changes/artifacts and stay within approved changed paths. A clean working tree with
commits is valid work; a clean tree alone is neither success nor failure. Unexpected dirty files
block acceptance and are preserved. Explicit no-change results require policy approval.

Persist result bytes and hashes outside the author's workspace. Validate in a separate worktree
at the exact output commit, running the approved argv/timeouts from the frozen input. The role
cannot replace verification commands with ones from its own output. Log exit status, bounded
output/digest, duration and the checked tree; a failed hard check cannot be overridden by a score.
Verification commands must be repeatable local checks; deployment and other external mutations
belong to later explicitly authorized stages. Path checks are acceptance checks, not an OS sandbox.

Designer/planner handoffs require the project's existing review/approval references before
advancement. A reference includes canonical artifact revision, decision location and content
digest; missing, changed or unrecognized authority blocks. The canary uses attended, frozen
approval references from its approved plan. Automatic import/authentication of new human decisions
remains with the existing authority workstream; models cannot write themselves approval.

The evaluator reads a fresh worktree at the executor's recorded commit. Populate author identity
from accepted stage receipts (including prior-stage provenance), not the evaluator request's
self-declaration. Apply the existing dispatch vendor exclusions after fallback; if historical
author provenance is unavailable, block rather than guess. The evaluator result binds that exact
commit and the frozen criteria; reject missing criteria, invalid JSON, wrong revision or blocking
findings. A changed candidate invalidates prior verification and evaluation. Persist deterministic
checks and evaluator judgment separately, then mark the work item completed/ready-for-landing.

### Quota and resource boundary

Factor admission policy for both role-run and iteration use, sharing the existing Codex/Ollama
observers and ledger policy used by the quota CLI. Put the hook before executor construction or
health checks that might spend quota, then recheck immediately before ExecuteStreaming. Required
admission wiring cannot default to allow when missing. Tests inject observations, not an allow
fallback. Classify the actual resolved route using registry/wire identity, including Ollama Cloud
through Pi; retain the current local-GPU exclusion.

Codex keeps the existing fresh provider observation/ration rules; Ollama keeps the corrected
session/weekly gauge and 95% cutoff. Unknown/exhausted protected quota blocks that candidate,
including explicit pins. Record observation timestamp, policy version and reason without secrets.
Fallback is permitted only before execution, within frozen candidates and independence policy.
Once dispatch starts, transport errors and quota exhaustion preserve the attempt and never trigger
an automatic provider switch on the same work.

Use the existing per-role token/cost guards and an iteration deadline; allocate each subsequent
stage only the remaining configured allowance. Unknown metered cost blocks further metered work;
subscription usage and imputed dollar cost remain separate. This is admission and bounded execution,
not a reservation of provider account capacity: attended sessions and legacy missions can consume
the same account between observations. Do not claim a fleet-wide hard subscription cap.

### Acceptance and canary plan

Hermetic acceptance battery, using temporary Git repositories/DBs and counting fake providers:

- [ ] Concurrent identical starts and different work-item IDs admit only one active item per mission.
- [ ] Changed input/alternate receipt path cannot bypass admission; missing DB status creates nothing.
- [ ] Process death at each boundary in the recovery table yields the specified state and call count.
- [ ] Parent lease loss/cancellation fences both child dispatch and stage acceptance; late results lose.
- [ ] Quota unknown/over and pin overrides produce zero protected provider health/inference calls;
      quota changing between preflight and dispatch waits and can later resume without a new task ID.
- [ ] Post-dispatch errors never switch provider; fallback evaluator identity remains independent.
- [ ] Committed clean-tree work passes; empty output, wrong commit, out-of-scope changes, mutated
      verification policy, dirty artifacts, stale evaluation and failed hard checks cannot complete.
- [ ] Foreign-origin project fixtures stay in their own repositories through the driver entry point.
- [ ] Completion survives restart, launches no duplicate role and returns the same evidence references.
- [ ] CLI help/docs, bounded package/race tests, architecture checks and relevant Bash 3.2 tests pass.

After these pass, prepare a concrete one-shot Docs canary record: chosen unowned task and approved
plan, base SHA, tested binary hash, routes and independent judge, worktree/DB paths, rollback and
existing quota policy. Propose at most 60 minutes, two remaining roles, at most 30 minutes per role,
and at most $5 total metered spend; freeze token caps from the chosen task before activation.
No model inference is required to prepare this record. Schedule only after the old Docs iteration
is idle; suspend its next fire during the controlled run to avoid cross-runtime duplication.

Canary passes only when it produces a useful validated project artifact, the DB/receipts prove
binary-owned dispatch and acceptance, duplicate invocation does no extra work, and status tells
the truth. Record total elapsed time, actual routes, metered/unknown/subscription usage, interventions
and duplicate provider calls. Compare with the nearest comparable legacy item descriptively;
one canary does not establish statistical productivity improvement.

Rollback disables new binary admissions, cancels or reconciles in-flight work, preserves its
worktree/DB/receipts and restores scheduling. Do not resubmit that item to the old loop while its
outcome is ambiguous. Other missions retain their current path. A successful canary earns a
separate fleet-adoption review; it does not silently enable all four loops.

### Implementation shape, risks and verified premises

Estimated ownership: `internal/coordinator/mission_work_item*.go` and acceptance persistence
(~600–900 implementation lines); shared service in `internal/mission/iteration/` plus dispatch
admission integration (~700–1,000); mission CLI/status/help (~250–400); compatibility driver branch
(~50–100); comparable fault/integration fixture volume. Reuse registry/config rendering and
existing SQLite attempt APIs. Move CLI-only durable dispatch glue into the service without
introducing a coordinator-to-mission dependency. Exact file split and estimates belong in planning.

Primary risks are scope growth, receipt/DB disagreement, and accidental policy changes during
extraction. Mitigations are the explicit work-item boundary, conservative reconciliation and the
side-by-side behavior table above. A local adapter is trusted code with its existing filesystem
permissions; acceptance validation does not guarantee containment of a malicious executor.

Source audit on 2026-09-07 (HEAD observed `6c0bf448d`; concurrent unrelated edits preserved):

| Premise | Evidence read / limitation |
|---|---|
| Installed surface is opt-in role execution, not an iterate command | `mission role-run --help`, `mission attempt status --help`; `mission_cmd.go` switch has list/doctor/install/apply/rotate-log/normalize/attempt/role-run/quota |
| Dispatch does not currently invoke quota observers | Read `mission_role_cmd.go`, `mission_role_state.go`, dispatch `run.go`; quota/Observe search across those paths has no observer calls; shell `_mc_load_ration` invokes quota CLI |
| Fallback stops at first executor invocation | dispatch `Runner.Run`: return through finish after ExecuteStreaming, with fallback loop only before it |
| Artifact acceptance is not implemented by role dispatch | `Report.ArtifactVerified` defaults false; `executionError` checks execution result/budgets/nonempty text, not Git artifacts |
| Terminal stages cannot be retried with a new receipt | `ClaimMissionAttempt` only reclaims matching expired prepared rows; read `TestMissionDurableRoleRejectsSecondReceipt` and `TestMissionCompletionAndCancellation` |
| Existing crash evidence is process-level | Read `TestMissionAbruptProcessRecovery`: subprocess exits without defers, reopens DB, distinguishes prepared/running; not a whole-iteration or power-loss test |
| Heartbeat cancellation is already exercised | Read `TestMissionHeartbeatRenewsThenCancelsWorker`: waits for actual lease renewal, cancels, asserts context cancellation; does not prove every external adapter kills descendants |
| Legacy retry is broader than role retry | Read `_mc_run_once` and following retry loop: controller process re-launched after matching transient/runtime quota output |
| Registry and role ownership must remain separate | `internal/mission/registry.go` rejects `[roles]`; workbench scope correction and model-registry dependency agree |
| Communications has its own approved boundary | Read communications design status/scope: charter authority and projections stay there; this slice does not add a decision writer/outbox |

No language-support claims, new diagnostic codes, live canary, test run or independent review are
claimed by this source audit. The dated handover supplies historical deployment observations;
those observations must be refreshed before activation.

### Axiom compliance for this increment

| Axiom | Score | Reason |
|---|---|---|
| A1 Determinism | +1 | Conditional transitions bind immutable inputs and observed external outcomes |
| A2 Replayability | +1 | Restart follows persisted receipts and acceptance without repeating execution |
| A3 Effect legibility | +1 | Dispatch, verification and cancellation become recorded boundaries |
| A4 Explicit authority | +1 | Acceptance checks frozen authority; no new approval writer |
| A5 Bounded verification | +1 | Frozen local checks and fault fixtures have deadlines |
| A6 Safe concurrency | +1 | Parent/child fencing prevents stale advancement |
| A7 Machines first | +1 | Structured status/results replace interpretation of controller prose |
| A8 Minimal syntax | 0 | No language syntax changes |
| A9 Cost visibility | +1 | Admission evidence and actual/unknown usage are separate |
| A10 Composability | +1 | Existing registry, executor and coordinator boundaries are reused |
| A11 Structured failure | +1 | Waiting and ambiguity are actionable states |
| A12 System boundary | +1 | Provider execution and candidate acceptance remain distinct |

Net +11; design intent has no negative A1/A3/A4/A7 score. This is a design assessment, not
implementation evidence or independent evaluator approval.

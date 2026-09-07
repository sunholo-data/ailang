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

# M-MAINTENANCE-EVAL-TASKPACK: Real Maintenance Chains for Roadmap Decisions

**Status:** Planned; requested by Mark on 2026-10-23. Design proposal, not implementation or model-spend approval.
**Target:** Opt-in maintenance eval; package release proposed after implementation. No compiler release or existing rotation commitment.
**Priority:** P1. **Estimated:** 4–6 engineering days for qualified task assets, a thin staged adapter and dry-run evidence, excluding runs; planning envelope, not measured velocity.
**Dependencies:** maintenance_bench 0.1.0; [evidence ledger](m-maintenance-bench-evidence-ledger.md); lifecycle protocol and existing executors below.

## Problem and measured starting point

[PR #157](https://github.com/sunholo-data/ailang-packages/pull/157) merged at `683046911d27e10ba5ccd44aae19988a62010e80`. Its [mature study](https://github.com/sunholo-data/ailang-packages/blob/683046911d27e10ba5ccd44aae19988a62010e80/research/real-world-maintenance/mature-2026-10-23/README.md) supplies reproducible observations, not a language win:

- Three real automated AILANG maintenance jobs recorded 1,114,042 input-plus-output tokens. Cache/model/billing reconciliation and later repair cost remain unknown.
- A real resource change introduced image-file aliasing; both historical sources type/effect-check on the pinned modern runtime, but the ownership oracle fails before and passes after repair about 12 days later.
- A Rust text-trimming regression passes its original focused tests and fails two new semantic checks; repaired production passes both.
- The 30-day later-touch proxy is 3/3 AILANG, 9/12 Go and 8/12 Rust. Those are different project/task mixes, not comparative bug rates. PR sampling misses direct-to-main work.

We need a prospective instrument where an agent changes its own retained program, old and new obligations are checked independently, and all attempts have costs. Historical known bugs are useful calibration assets; repeating them indefinitely cannot supply independent evidence of future improvements.

## Coverage decision and ownership

This is a concrete real-source child of the existing [real-world study](m-real-world-language-maintenance-study.md) and [lifecycle protocol](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/m-lifecycle-eval-pilot.md). The parent owns lifecycle inclusion, continued failure and cost-per-success principles. This child owns task packs, staged-execution integration, oracle qualification and roadmap promotion. It replaces the parent's provisional synthetic seed topics for this workstream; it does not start another general runner, dashboard, rating model or default benchmark rotation.

The companion ledger owns versioned index/observation separation, usage normalisation, semantic joins and evidence cards. Build identity/cohort provenance remains with the [existing provenance design](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/m-cohort-manifest-build-provenance.md); this proposal consumes truthful identities without assuming that design has shipped.

Scaffold and explicit SimHash/neural searches were run. The nominal neural search reported zero embeddings and fallback-SimHash; its scores are not a semantic duplicate verdict. Manual reads established the ownership boundary above. Existing eval YAML guidance on appending task_prompt is reused where applicable; a lifecycle is not reduced to one expected_stdout task.

## Goals and claim boundaries

1. Ship two qualified real-requirement task families with successive feature/repair/migration obligations and independently checked retained invariants.
2. Produce an offline no-model run that validates every stage, negative control, identity, failure state and usage aggregation.
3. Run an explicitly approved pilot of **2 families × 2 languages × 2 repeats = 8 trajectories, each with 3 changes = 24 scheduled maintenance episodes**. This estimates feasibility and friction, not population significance or a representative working day. A later corpus needs a prospective source-task sample with declared project, task-size and change-type strata, including ordinary successful work rather than only interesting bugs.
4. Produce evidence-linked roadmap candidates whose claims are no stronger than their reproduction and coverage.

The first pilot is **maintenance-only**: competent native starters are prepared and qualified before agent execution. It measures changes to existing programs. A separate generation-plus-maintenance cohort may add a model-built initial application, retain failed initial builds and charge their construction cost; it must have its own manifest and counts. Do not label starter preparation as measured model generation.

No change to compiler/types/effect semantics, no new core release gate, no generic workflow engine and no retrospective declaration that public cases are sealed holdouts. A neutral or negative AILANG result is a valid outcome.

## High-impact decisions

| Decision | Proposed choice | Owner / deadline | Change cost |
|---|---|---|---|
| Unit and comparison | Whole 3-change trajectory; AILANG/Python first; native tooling allowed and frozen | Mark / design approval | High |
| Continuity | Fresh agent session at every stage, carrying the agent's own declared app artifacts and handover notes | Mark / design approval | High |
| Task identity | Real-requirement component adaptations, distinguished from exact historical replay | Task author / before qualification | High |
| Execution surface | Thin extension of existing agent eval/executor path, not a second runner | Implementer / dry-run acceptance | Medium |
| Oracle exposure | Independent stage/final oracles outside agent workspace; no hidden failure feedback during primary trajectory | Oracle owner / protocol freeze | High |
| Run budget/model | Exact model/provider/harness, token accounting, caps and prices frozen in launch manifest | Mark / before any model call | High |

### Design freeze

- [ ] Approve these family contracts, the maintenance-only stratum and first comparator.
- [ ] Approve the small staged adapter scope and ledger dependency for sprint planning.
- [ ] Both native starters pass the same external semantic preconditions; equal responsibility and allowed dependencies are reviewed.
- [ ] Oracle reference/no-op/mutation evidence and artifact-isolation dry run are reviewed.
- [ ] Exact launch identity and a current priced/capped run plan are approved separately.

Implementation may choose internal filenames, fixture encodings and report layout. It may not change primary outcomes, inclusion, stage schedule, budget policy or public/held-out status after observing model results.

## Task families and stages

All entries below are proposed task assets, not claimed implemented. Historical sources are seeds for independently specified requirements. Creating a competent counterpart is recorded starter-authoring work; it is not passed off as an existing historical Python repair. If equivalent starters cannot be qualified, keep that family as an unpaired development task and refuse its language delta.

### F1 — Image resource ownership through a storage migration

Source: Parse [#46](https://github.com/sunholo-data/ailang-parse/pull/46) and [#77](https://github.com/sunholo-data/ailang-parse/pull/77), with the exact accepted component probes already merged. Use a bounded image-extraction component rather than porting a whole document platform. Both native starters correctly preserve image bytes before changes.

| Stage | Real maintenance request | Old obligations that remain checked |
|---|---|---|
| S1 | Move retained inline image bytes to file-backed resources, keeping the consumer API's byte meaning | A returned image reference continues to resolve to its own bytes after another document is processed |
| S2 | Add a batch operation and per-document decoded-byte cap with explicit failure | Earlier successful results remain correct; a rejected document cannot corrupt another document's resources |
| S3 | Add release/cleanup for one completed document | Unreleased documents, including shared-content cases, remain readable; cleanup obeys the declared lifetime contract |

Check repeated basenames, different bytes, duplicate content, operation order, cap boundaries and retry after partial failure. The ownership obligation exists from the starter, so losing it after S1 is a regression, not a newly invented S2 requirement. Cleanup deliberately ends only the released document's lifetime; old tests do not demand released resources live forever.

Pass/fail uses bytes, ownership and explicit error results. Peak memory and verification time are recorded; an RSS pass threshold is introduced only after platform-specific calibration, not chosen to favour one runtime. The first version uses sequential interleavings as in the reproduced defect; it does not claim physical race coverage.

### F2 — Text fidelity through parser and interface changes

Source: Calamine [#671](https://github.com/tafia/calamine/pull/671)/[#673](https://github.com/tafia/calamine/pull/673), with exact parent/repaired controls. Both native starter slices expose the same cell-text contract and correctly preserve logical text at baseline.

| Stage | Real maintenance request | Old obligations that remain checked |
|---|---|---|
| S1 | Honour explicit preserve-space and default edge-ASCII trimming | Internal spaces, decoded character references and non-ASCII whitespace retain their specified meaning |
| S2 | Adapt to an event/streaming reader or interface revision | Chunk boundaries, reference events and empty values cannot silently change logical content or error meaning |
| S3 | Extend the documented structured-text cases and output record | Existing consumers retain every required field and distinguish malformed input from a valid empty result |

The exact S2 interface and S3 case set are authored from real documented use before oracle freeze; both languages must receive equivalent responsibility. Candidates include CDATA/rich-text mixtures and existing output-schema consumers. Qualification rejects an arbitrary extra difficulty in one arm. This is a proposed semantic contract, not a claim that any particular current stdlib API lacks support.

### Development, validation and confirmation

The two historical reproductions and everything exposed in PR #157 are **development/calibration** cases. They can test oracle sensitivity and reveal roadmap friction; their scores cannot confirm a language improvement. Variants must represent the same real operational requirements, not be renamed old eval tasks.

A future confirmation batch uses new real requirements/tickets and independently frozen oracles, split by issue family, subsystem and shared lineage. Freeze digests before execution, score once, retain results, and retire exposed failures from future confirmation. Use new confirmation cases after a roadmap fix. Saturated development tasks remain useful regressions but lose ranking weight; they are not removed retrospectively to improve a headline score. Existing language-development evals remain outside this primary corpus.

## Execution integration

Propose `ailang eval-suite --agent --maintenance-manifest <file>` as an **unimplemented opt-in flag**. It identifies every planned episode explicitly, conflicts with ordinary benchmark discovery/tier/confidence selection, and never enters the standing rotation automatically. Standard one-shot mode refuses this manifest. Ordinary invocations remain byte-for-byte compatible.

Own the adapter near `internal/eval_harness/` with only CLI plumbing in `cmd/ailang/eval_suite.go`. It validates a finite stage list, constructs existing executor.Task values and invokes the existing executor/result parsers. Reuse provider preflight, rig admission, model registry, credentials policy and available budget mechanisms. Do not reimplement Motoko's loop or parse its logs a second time in the package. If a new execution helper is needed, keep it scoped to this manifest contract and cover it with a scripted executor; no general workflow engine.

The existing agent workspace-grading path preserves declared solution files and reseeds fixed inputs, but its GradeEntrypoint path is not a proven cross-language lifecycle grader. The adapter must preserve that integrity property while running neutral external oracles on each language's retained component. Hidden oracle sources/results are outside the model's writable/readable task tree. A launch cannot claim a protected holdout unless the actual worker/tool boundary enforces it; otherwise it is development-only.

**Fresh sessions are intentional.** The inspected Motoko executor advertises streaming/local workspace/structured output, not CapSessionResume; it creates a session for each call. The first pilot therefore makes no continuous-context or resumed-session uplift claim. It carries only app source, dependency locks, public tests and model-authored handover notes. Logs, hidden tests, provider journals and old conversations are not reintroduced accidentally. A continuous-session comparison is a separately qualified executor/treatment.

At every stage:

1. Verify manifest, prior checkpoint, native toolchain and oracle identity before dispatch.
2. Restore that trajectory's own declared artifacts; seed immutable developer checks and the new requirement. Do not replace failing code with a reference solution.
3. Invoke the frozen model/harness with native tooling. Include teaching/retrieved context and handover work in usage. Capture every attempted change/check and its feedback.
4. Freeze the produced artifact and run external old/new invariant checks in an isolated build. The agent does not receive hidden verdicts during the primary trajectory.
5. Record terminal state, costs and next checkpoint. Proceed on the fixed schedule even after a failed stage where execution is possible; otherwise record an explicit blocked continuation.

Use per-trajectory workspaces and per-source-identity build products; the merged Rust reproduction demonstrated why a shared stale target can create a false pass. Retain failed snapshots and spent cost. External bridges are declared adapters; neither arm may outsource the target implementation wholesale to the other language.

## Measurement and analysis

Primary: unconditional whole-trajectory independent success with every required retained invariant preserved. Report all eight scheduled trajectories and all 24 scheduled changes, including failed initial changes and blocked continuations. Show started, completed, invalid, budget-stopped and not-run counts separately. A model is not declared incorrect merely because its job was never launched.

Economics: total generation/diagnosis/repair/verification model cost and tokens, including failed attempts, across the declared cohort; cost per independently successful trajectory; stage distributions and human intervention. Zero successes is unavailable, never zero cost per success. Missing usage prevents the affected economic comparison while correctness may remain interpretable. Local GPU time/hardware/energy proxies remain separate from metered dollars and subscription list-price equivalents.

Record visible-check acceptance separately from the external oracle. A visible compiler/test failure followed by a repair is an observed interception; it is not automatically a prevented production defect. A visible-green change violating a retained hidden requirement is an escaped tooling regression in this experiment. Feature work, migrations, semantic repairs, infrastructure repairs and inconclusive causes stay separate. File touches and local edit attempts are separate measures.

Use the ledger's semantic pair key while preserving language labels. Report delivered language-plus-native-toolchain deltas; do not relabel Lang to use the existing same-language PairArms join. Two task families and repeated trials are clustered; the pilot publishes paired counts/descriptive ranges, not a population p-value. A causal claim specifically about early type/verification feedback would require a separate within-language feedback intervention with final correctness checks unchanged.

Proposed pilot ceilings for a later launch review: 40 agent turns and 600,000 accounted tokens per change (the ledger's fresh + cache-read + cache-write + visible-output + disjoint-reasoning measure, requiring complete source accounting); 10 minutes per change for a cloud pilot; one active trajectory; 6 hours campaign wall time; a proposed $25 aggregate metered cap. These are limits, not a price estimate or authorisation. Local hard timeouts must be separately calibrated; do not copy a cloud deadline onto a slow local model. Freeze exact definitions/prices and preflight the worst next request before dispatch. Existing suite dollar/wall caps stop scheduling after banked cost and can overshoot with in-flight work; they are not represented as a real-time hard meter. If the chosen executor cannot enforce the proposed bound, disclose the gap and revise the approved launch policy before calls.

Stop on the frozen rule, never when a favourable delta appears. At most one separately linked infrastructure replacement is proposed; original spend remains. Ordinary model/task failures receive no special replacement. A partial budget-stopped batch is coverage-limited and cannot be reported as completed confirmation.

## From results to roadmap designs

The ledger emits deterministic evidence cards; a maintainer applies design-doc-creator to qualified cards. A card contains the exact failed obligation, source/model/harness/tool versions, before/after artifact hashes, attempted checks, oracle/mutation result, usage coverage, affected consumers, recurrence and the strongest alternative explanation. It links the frozen input and run, not just a screenshot or a fix-title count.

| Evidence | First routing action | Design gate |
|---|---|---|
| File aliasing after a resource migration | Application/package ownership helper + regression oracle; check existing content-addressed helpers first | A broader stdlib need is claimed only after reuse/consumer audit |
| Repeated quadratic assembly or retention work | Profile actual paths; consider language authoring diagnostics or harness guidance | Source pattern alone is not proof of the current runtime's bottleneck |
| Context/handover/diagnostic loops consuming measured work | Motoko extension or existing harness integration per PROGRAM | Frozen core stays frozen unless a floor defect is reproduced |
| Dropped logical text despite native checks | Exact semantic oracle and application correction; audit a reusable primitive | Type-check success is not substituted for the missing requirement |
| Unknown/duplicated token attribution or stale builds | Instrument repair | Affected rows do not support a language claim |

Recurring cost/severity across independent consumers prioritises normal roadmap work. A single severe reproduced defect can warrant action with an explicit exception; do not multiply related packages into independent evidence. Cards can draft proposals, not approve implementations, merge patches, dispatch agents or publish packages. A design can legitimately recommend no language change because an application fix already solves the problem.

## Delivery and acceptance

| Milestone | Deliverable | Required failure control |
|---|---|---|
| T1 — Qualification | Two native starter pairs, semantic contracts, public checks and external oracles | Reference passes; pre-fix/no-op and weakened-oracle mutations fail; unsupported pairing refuses |
| T2 — Staged adapter | Finite plan over existing executors; fresh sessions with owned artifacts | Scripted agent returns a broken first stage; no gold restoration; all 24 scheduled rows accounted |
| T3 — Identity/accounting | Ledger receipts, separate scopes and exact semantic pairs | Duplicate attempt, altered prompt/provider, dropped costly failure, missing cache/billing and unmatched arm are surfaced |
| T4 — Offline package release candidate | Native smoke, bounded no-model replay, docs and report/card examples | Hidden tests cannot be weakened by editing agent files; stale build and skipped oracle cannot look successful |
| T5 — Optional pilot | Approved launch manifest, estimate and stopped/completed report | This milestone cannot run on design approval alone; actual campaign authorisation and caps required |

Keep task manifests/oracle assets under the package repository, with public development assets eligible for a later package tarball and confirmation oracles in a separately controlled artifact bank. Task inputs and metric schemas are versioned; old 0.1.0 study results remain replayable. The AILANG package handles analysis; the existing harness owns model execution.

Success means a trustworthy negative, neutral or positive report with actionable, correctly routed evidence. It does not require AILANG to win. No implementation or paid run accompanies this document.

## Verification log, conflict surface and risks

| Claim | Evidence inspected on 2026-10-23 | Limit |
|---|---|---|
| Real causal probes and usage exist | Merged mature study, verified-oracles.json, source hashes and replay scripts at 6830469 | Component probes on pinned modern runtimes; usage is first-party attestation |
| Existing workspace/task execution can be reused | executor.Task fields; agent_runner_multi.go executeWithBrowser; spec.go GradeEntrypoint/SolutionFiles | Persistent cross-stage grading is new adapter work, not claimed shipped |
| Existing CLI has agent, dry-run, trials and suite caps | cmd/ailang/eval_suite.go flag definitions | New maintenance-manifest flag search returned no allocation; proposed only |
| Continuous Motoko executor resume is not assumed | Task.ResumeSessionID exists; motoko.Capabilities excludes CapSessionResume; Execute starts a session | Motoko core durable sessions in README are a different capability surface |
| Standard pair key is unsuitable directly | internal/eval_analysis/paired.go pairKey includes ID/Lang/Trial | Ledger must validate semantic cross-language pairing |
| Cost/cache distinctions exist | executor.Result/CostProvenance, RunMetrics.CacheAccounted/FreshTokens | Legacy chain totals cannot be upgraded to a known split by inference |

Core sources read at observed HEAD 19e33da; actual launches must pin their own revisions. MOTOKO.md identifies mk-20261019 at 907dd78e, also verified read-only. This is inventory, not a permanent path/version requirement.

Conflict surface: proposed changes affect eval scheduling/fixtures and package analysis, not parser/typechecker/evaluator/VM/effect semantics or cmd/ailang/exec.go. Preserve ordinary suite selection, standard grading, existing agent workspaces and result readers. If implementation expands into language semantics, add the mandatory conflict analysis and obtain that scope's approval.

Risks: mismatched starter difficulty, public-answer contamination, hidden-oracle leakage, stale build products, censored/unknown outcomes, diagnostic rather than semantic successes, and accounting drift. Qualification, explicit strata, isolation controls, immutable sidecars and unavailable states address these; the pilot cannot remove small-family uncertainty.

## Axiom compliance

Canonical reference: [Design Axioms](https://ailang.sunholo.com/docs/references/axioms).

| Axiom | Score | Constraint |
|---|---:|---|
| A1 Determinism | +1 | Seeded task inputs/oracles; stochastic model identity and settings recorded |
| A2 Replayability | +1 | Frozen artifacts, sidecars and source-specific builds |
| A3 Effect legibility | +1 | Pure analysis; execution authority stays in the explicit harness boundary |
| A4 Explicit authority | +1 | A finding/card cannot authorise a run or implementation |
| A5 Bounded verification | +1 | Finite stages, budgets, oracle counters and negative controls |
| A6 Safe concurrency | 0 | First pilot serial; no new language concurrency claim |
| A7 Machines first | +1 | Structured outcomes and identity-bound diagnostics |
| A8 Minimal syntax | +1 | Existing syntax/executor APIs only |
| A9 Cost visibility | +1 | Failed work, cache, verification and missing usage remain visible |
| A10 Composability | +1 | Existing lifecycle protocol, executors and package analysis |
| A11 Structured failure | +1 | Invalid, blocked, not-run and failed remain distinct |
| A12 System boundary | +1 | Observations, oracles, planning and authority are separate |

Net **+11**; no negative on A1/A3/A4/A7. Directional design assessment, not implementation approval or proof of language superiority.

### Hard-violation check

- [x] A1: no implicit model nondeterminism is hidden in deterministic replay claims.
- [x] A3: analysis and model/tool execution keep explicit effect/authority boundaries.
- [x] A4: generated findings never grant run, merge or implementation permission.
- [x] A7: machine-readable states preserve unknowns, invalidity and identity.

## 2026-10-23 foundation implementation

Mark approved the concrete foundation sprint with “execute sprint”. The native package now implements explicit index/workload scopes, strict neutral receipt accounting, retained stage ledger validation, matched-arm economics, exact edit bounds and draft reproduction cards. Both native resource/text development taskpacks qualified against positive, semantic-negative and weakened-oracle controls. Source v0.2.0 is a release candidate; registry publication has not occurred.

Review amendments before model execution: headline token economics uses fresh input plus generated output, with fresh input as a separate diagnostic; unknown output cannot support the headline. Standalone roadmap cards cannot qualify economic claims from a known label; those need identity-bound controlled reports. Card identities and JSON output types are strict. Follow-up windows and observed rename/delete/copy history remain explicit. See the foundation sprint's actual verification and review log.

The complete lifecycle design stays planned: existing-executor staged integration, actual worker/oracle protection, live source receipt qualification and a priced model campaign are later scope. Exposed development controls do not confirm a general language improvement.

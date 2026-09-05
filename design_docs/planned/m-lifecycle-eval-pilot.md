# M-LIFECYCLE-EVAL-PILOT: Measure Repair, Change and Resumption

**Status:** Proposed experiment/instrument design — not implementation-approved; quorum not run.
**Date / author:** 2026-09-05 / Astra, requested by Mark.
**Target:** Opt-in v1.x evidence; no new v1.0 threshold or default-rotation change.
**Priority:** P1 candidate.
**Estimated:** 3–5 engineering days for corpus/protocol/instrument, then separately budgeted runs.
**Dependencies:** [Repair packet](m-semantic-repair-packet.md) for that treatment; World integration for later World arms.
**Traces to:** [Astra vision](m-astra-vision.md), [eval validity](m-eval-validity-discipline.md), [measurement contract](../implemented/v0_31_0/m-eval-measurement-contract.md).

## Problem and coverage decision

A language may help models recover and preserve constraints even when initial pass rates are similar. Conversely, short success-only tasks may conceal the cost of repairs, stale evidence and resumed work. This proposal tests those hypotheses; it claims no measured AILANG advantage.

Existing validity/cohort designs own provenance, error attribution and comparable cohorts. This child adds a lifecycle task protocol and a small opt-in corpus. It does not create another dashboard, change ELO, or replace the current cost-per-success KPI.

## Goals / non-goals

Measure independently verified task completion after a controlled requirement change or restart, along with total cost, repair attempts and policy violations. Distinguish language effects from context/harness effects.

Non-goals: public language-preference claims; training a model; changing release gates; calling paid providers during design; physical external actions; treating a synthetic UI drill as World's three REAL provenance questions.

## Protocol

Each task family contains a frozen starter repository and an independently authored evaluator with three phases:

1. **Initial task:** implement a bounded behavior; evaluation records correctness, cost and artifact identity.
2. **Perturbation:** reveal a predeclared requirement change or controlled defect. Include a preserved invariant and a forbidden shortcut. Reveal on the fixed phase schedule, not only after initial success.
3. **Continuation:** optionally restart the resident with a specified retained-artifact set. Evaluate final behavior, retained invariants, permitted actions and explanation provenance.

Unsuccessful initial phases remain in the denominator. If continuation is impossible, record that terminal outcome and its reason; do not silently replace the task with a successful starter. A distinct repair-only stratum may start all arms from the same seeded defective program, with a separate cohort ID.

Candidate families to author: parser configuration amendment, state-machine rule change, data-pipeline schema revision, and bounded code-patch repair. Existing `benchmarks/{ast_patch_roundtrip,state_machine_elevator,pipeline,contract_sorted_merge}.yml` are seed topics, not already-authored lifecycle fixtures or causal evidence. Use independently written hidden checks, held out from the generator's context.

## Arms — vary one thing at a time

| Study | Control | Treatment | Interpretation |
|---|---|---|---|
| First pilot | Existing full-context repair under a frozen harness | Repair packet under the same harness/model | Context-tool uplift within AILANG |
| Later language study | Python with competent native tooling | AILANG with competent native tooling | Delivered language/toolchain comparison; not pure syntax effect |
| Later World study | Same resident using shell and persisted files | Same resident through World | Substrate cost and operational outcomes; World owns its release floor |

Do not combine all three changes in one headline comparison. Select models by documented baseline ability/operational availability, not by treatment results. Freeze exact model/provider/settings; “small” means an observed ability tier, not merely a vendor label. Report tiers individually. Count any stronger model used to prepare task decomposition.

## Manifest and result schema

Extend the existing cohort/measurement instruments with a versioned lifecycle sidecar. Proposed fields: task-family ID, variant, phase protocol hash, starter hash, hidden-evaluator hash, perturbation hash, retained-artifact policy, arm, repeat, treatment version, source/dependency/compiler/prompt identities, actual model/provider/config, harness revision, budget/cache/retry policy, and randomization seed.

A result records phase outcomes, final outcome, input/output artifact identities, declared versus observed tool invocations, policy violations, consumed budgets, model costs, verification costs, elapsed time and infrastructure validity. Unknown prices remain unknown; local inference reports time/hardware separately rather than pretending it is free. A valid refusal to perform a forbidden action can be success when the oracle explicitly requires refusal.

First implementation should use checked sidecars attached to existing run IDs, avoiding a breaking change to `BenchmarkResult` and its readers. The adapter refuses unmatched identities or duplicate arm keys before analysis. Do not repurpose stdout success as lifecycle success without an explicit documented conversion.

## Pairing and statistical discipline

`internal/eval_analysis/paired.go` currently pairs on `(ID, Lang, Trial)` and defines success through compile/runtime/stdout checks. Therefore:

- For a same-language packet experiment, a validated adapter can map whole-episode oracle success into the existing paired input, preserving the raw phase records separately.
- Do not feed cross-language arms directly into that join and assume they pair; `Lang` is part of its key. A later language study needs an explicit semantic task-pair mapping and analysis approval.
- Reject duplicates before `PairArms`; its map assignment otherwise cannot certify uniqueness of supplied keys.
- Existing McNemar/headroom output is useful descriptively. Repeated variants/trials within a family are clustered, so do not treat every episode as an independent family in inferential claims. Predeclare family-level summaries and use family-cluster uncertainty in a confirmation study.

Exploratory first batch: 8 task families × 2 arms × 3 repeats × 3 ability tiers = 144 episodes. Bank all rows, including invalids and failures. No superiority conclusion from this size by default. Estimate discordance and variance, then pre-register confirmation size and minimum worthwhile effect.

Primary metric: total paid generation/repair/verification cost across the cohort divided by independently verified final successes. Zero successes is undefined/infinite, not zero. Publish human time and p50/p95 latency separately. Also report invariant failures, forbidden actions, repair attempts and success conditioned on initial success as secondary statistics ONLY; unconditional success remains primary.

Stop when the frozen budget or episode cap is reached, never when the comparison first looks favorable. Instrument-invalid rows retain spent cost and are separately reported. At most one replacement retry per infrastructure-invalid episode in the proposed pilot, both linked and counted operationally; no retry for ordinary model failure. Missing price/config invalidates the relevant economic claim, not necessarily the correctness observation.

## Implementation plan

| Phase | Likely scope | Estimate |
|---|---|---|
| M1 | New opt-in lifecycle fixture directory and manifest/oracle specification under `benchmarks/`; use existing harness runner mechanisms | 1–2 days |
| M2 | Sidecar validator and episode adapter near `internal/eval_harness/` and `internal/eval_analysis/`; no leaderboard mutation | 1–2 days |
| M3 | Deterministic dry run, report template and cost estimate; only then run approved model batch | 1 day plus run time |

Before planning M2, inventory the harness's existing staged/agent fixture support and pick the smallest adapter; no new generic workflow engine. Detailed file split and LOC estimate belong to the sprint plan after that inventory.

## Acceptance criteria / failure controls

| ID | Acceptance | Required counterexample |
|---|---|---|
| LE1 | Every scheduled episode accounted for, including initial failures and incomplete continuations | Drop a failed initial episode; accounting check fails |
| LE2 | Same task/protocol/model/harness/budget except the declared treatment | Alter prompt or provider identity in one arm; pairing refuses |
| LE3 | Hidden checks catch weakening the specification and breaking a preserved invariant | Seed both mutations; neither may pass the final oracle |
| LE4 | Restart retains exactly the declared artifacts | Plant a forbidden transcript artifact; environment/manifest check catches it |
| LE5 | Costs include unsuccessful attempts and replacement retries; zero-success output honest | Insert costly failed/retried row and zero-success cohort |
| LE6 | No duplicate keys, silent unmatched pairs or false cross-language join | Duplicate one trial; replace language; leave unmatched arm |
| LE7 | Dry-run test validates all phases without a model or external service | Empty family set refuses; planted perturbation is observed |
| LE8 | Report distinguishes pilot, confirmation, invalidity and oracle coverage | All-pass tiny cohort cannot generate an unsupported significance claim |

Verification for instrument implementation: existing paired-analysis tests plus the new sidecar/dry-run tests and repository required checks. Real-model results are a later artifact, not an acceptance claim made by this doc.

## High-impact decisions / design freeze

- [ ] Mark approves first intervention and opt-in status; no release-bar change.
- [ ] Task families, independent oracle owner and model-tier selection frozen before runs.
- [ ] Dollar/time caps, cache treatment, retry rule and stopping rule accepted.
- [ ] Harness adapter inventory completed; source identities can be captured truthfully.

Designer may choose fixture organization and report layout. Any change to the primary metric, episode inclusion or arm identity requires a new manifest/cohort, not an in-place revision after results are visible.

## Verification and related documents

Read `internal/eval_analysis/paired.go` to establish the key and success predicate; read the measurement-contract and cohort-provenance designs. The four named seed benchmark paths exist; no claim is made that their bodies already meet this protocol. Scaffold searches attempted; fallback search scores are not neural duplicate evidence. Manual coverage review assigns this child the lifecycle corpus/protocol only.

Related: [cohort provenance](m-cohort-manifest-build-provenance.md), [validity discipline](m-eval-validity-discipline.md), [repair packets](m-semantic-repair-packet.md). Risks: contaminated holdout, asymmetric tooling, task clustering, small-model overhead and optimistic cost accounting; each is controlled above. No code or paid runs accompany this design.


## Axiom compliance

Directional design assessment, not implementation approval. No hard violation proposed on A1/A3/A4/A7.

| Axiom | Score | Design constraint |
|---|---:|---|
| A1 Determinism | +1 | Identity-bound inputs; deterministic mechanical results |
| A2 Replayability | +1 | Preserve inputs and outcomes needed to reproduce the decision |
| A3 Effect legibility | 0 | Existing effect semantics unchanged |
| A4 Explicit authority | +1 | Metadata and model output never mint execution authority |
| A5 Bounded verification | +1 | Explicit limits and checks with named refusal outcomes |
| A6 Safe concurrency | 0 | Sequential first version; snapshot identity checked |
| A7 Machines first | +1 | Structured, versioned artifacts with explicit unavailable states |
| A8 Minimal syntax | +1 | No new language syntax |
| A9 Cost visibility | +1 | Record tool/model costs and failure overhead |
| A10 Composability | +1 | Reuse existing compiler, evidence and protocol boundaries |
| A11 Structured failure | +1 | Unknown, incomplete and stale cannot masquerade as success |
| A12 System boundary | +1 | Separate claims, verification, permission and action |

**Net +10.** Re-score if implementation scope changes.

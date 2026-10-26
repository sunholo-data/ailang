# M-MAINTENANCE-BENCH-EVIDENCE-LEDGER: Correct Cohorts, Run Costs and Roadmap Cards

**Status:** Planned; design requested by Mark on 2026-10-23. Not implementation or registry-publication approval.
**Target:** Proposed sunholo/maintenance_bench 0.2.0; opt-in metric/schema versions, with 0.1.0 evidence preserved.
**Priority:** P1, prerequisite for priced lifecycle claims. **Estimated:** 2–4 engineering days after adapter inventory; not a measured velocity forecast.
**Dependencies:** Released maintenance_bench 0.1.0; existing executor/RunMetrics accounting; [task pack](m-maintenance-eval-taskpack.md).

## Problem and verified evidence

The [merged mature study](https://github.com/sunholo-data/ailang-packages/blob/683046911d27e10ba5ccd44aae19988a62010e80/research/real-world-maintenance/mature-2026-10-23/README.md) found real information but also specific measurement gaps:

1. The first PR selector produced no mature denominator; merged-window selection produced only three mature functional AILANG PRs, while direct-main history contained other work. Absence from a query is not absence of maintenance.
2. v0.1 Manifest selects repositories/authorship/canonical rows, not an explicit set of index episode IDs. Follow-up observations are needed to find repairs but must not automatically enlarge a sampled index denominator.
3. v0.1 metrics.summarize selects eligible/mature counts from the cohort but workload/usage/outcome rows from allRows in the group. The first pilot therefore reports six eligible Parse changes but seven unknown outcomes and one mechanical change, including an excluded smoke. Scope must be explicit rather than inferred by the reader.
4. Three task-ID lookups recover 1,114,042 recorded tokens despite empty repo-filtered chain queries. Model/cache/billing reconciliation and later repair usage remain unestablished. A scalar is not an accounting contract.
5. A same-file touch is a candidate relationship. Exact source probes confirm one AILANG introducing-change link; other overlaps and the separate Rust case cannot be quietly made additional primary events.

The package already provides strict codecs, native tests, lineage, semantic task pairing and deterministic reports. Extend those boundaries; do not create a dataframe engine or replace the executor's provider parsers.

## Coverage and ownership

This child owns analysis schemas, scoped denominators, usage receipts, semantic joins and evidence cards in ailang-packages. [Task pack](m-maintenance-eval-taskpack.md) owns real task contracts, staged scheduling and external-oracle qualification. [Real-world study](m-real-world-language-maintenance-study.md) owns observational sampling and interpretation. [Lifecycle pilot](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/m-lifecycle-eval-pilot.md) owns continuation/failure principles. [Build provenance](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/m-cohort-manifest-build-provenance.md) owns core build identity mechanisms; consume identities without reimplementing or presuming that proposal has landed.

The scaffold and explicit searches ran; neural output used fallback-SimHash with zero embeddings. Manual review, not its numeric similarity scores, established this distinct scope. No new compiler syntax, runtime effect behavior, generic runner, dashboard, model selector or default rotation is proposed.

## Goals

- Make every numerator/denominator and every usage total name its population and evidence coverage.
- Analyse a frozen sampled index while retaining later observations and excluded workload without counting them as new index changes.
- Import existing run/task receipts without losing cache/reasoning/cost provenance or charging parent and child summaries twice.
- Produce identity-bound evidence cards that can support a reviewed roadmap design, including negative/inconclusive results.

## High-impact decisions and freeze

| Decision | Proposed choice | Owner / deadline | Change cost |
|---|---|---|---|
| Data model | Separate historical source observations, selected index, controlled trajectories and usage receipts | Mark / design approval | High |
| Metric compatibility | New metric/report schema; preserve immutable 0.1.0 evidence and its semantics | Mark / design approval | High |
| Money/token claims | Source-specific accounting contract, explicit known/missing quantities and currency/provenance | Mark / launch protocol | High |
| Execution boundary | Existing harness parsers produce receipts; pure AILANG validates/aggregates them | Implementer / adapter qualification | Medium |
| Evidence-to-design | Deterministic card/draft only; existing design approval/execution gate remains | Mark / design approval | High |

- [ ] Approve schema separation and explicitly named metric populations.
- [ ] Approve versioned report semantics and preservation of old study artifacts.
- [ ] Qualify exact executor-source token/cost adapters with independent fixtures.
- [ ] Approve roadmap-card routing and review rules before autonomous use.

The implementer may choose internal record/ADT decomposition, helper names and JSON ordering. Publication timing, primary outcomes, claim eligibility and rounding policy are protocol decisions, not invisible implementation fallbacks.

## Schema boundaries (proposed)

These are new versioned artifacts, not claimed current exports or CLI commands:

| Artifact | Responsibility |
|---|---|
| Index cohort | Frozen source frame, selection rule/hash, explicit unique index IDs, exclusion reasons, cluster IDs, cutoff/window and acceptance-evidence policy |
| Observation snapshot | Canonical changes plus later repairs, migrations, mirrors and context; original source and capture identities retained |
| Trajectory manifest | Family/variant, native starters, three-stage schedule, retained-artifact policy, language arms, repeat/model/harness/config, oracles and budgets |
| Run ledger | Scheduled/started/terminal stage states, attempt IDs, input/output artifacts, checks, diagnostic events, validity and links to task/session records |
| Usage receipt | Original source contract, leaf request/attempt identity, quantities and knownness, cache/reasoning semantics, cost currency/provenance and source digest |
| Evidence card | Reproducer, affected obligation, scope, source/run identities, baseline/fix/mutation outcomes, cost coverage, recurrence and proposed routing |

Use closed states in native AILANG; strict boundary decoding rejects unknown schema versions, duplicate keys/IDs, invalid enums and invalid numeric lexemes. IDs/hashes stay text. Raw high-precision monetary values are decimal text with currency/provenance; arithmetic uses a declared fixed-point scale and rounding policy or returns unavailable on unsupported precision/overflow. No lossy float-to-integer conversion can turn an unreported amount into zero. Audit existing numeric/package capabilities before proposing stdlib work.

### Historical index versus follow-up versus workload

An index ID belongs to exactly one frozen eligible/excluded decision. A follow-up can be outside that index, and can repair several index artifacts or one defect across several files, without becoming another denominator entry. The library validates same-project/language lineage, chronology and report/capture dates. One defect, five changed files and three visible corrective rounds remain different quantities.

Maintain separate named scopes:

- **index:** sampled eligible accepted changes; mature/censored/unresolved acceptance partitions account for every selected row;
- **followup:** observed corrections/touches within the named index-specific window, with supported/confirmed/unknown links;
- **canonical workload:** all declared canonical-source work in the observation frame;
- **total workload:** canonical, dependency, mirror, tests and infrastructure work, charged as observed rather than hidden by source-only filters;
- **controlled trajectory:** all scheduled stages and owned artifacts, with attempts and usage; it does not borrow a historical 30-day denominator.

Every reported metric includes scope, numerator/denominator or known-value count, missing count, status/reason and metric version. Zero denominator is unavailable. Zero known cost rows may have an empty known-value sum, but the display/summary must state unreported total expenditure. Supported repairs are descriptive counts; confirmed introduced-defect incidence requires a supported causal link and the relevant oracle/evidence policy. Unknown attribution cannot be treated as a clean index change.

Direct main commits retain commit-proxy time unless reviewed evidence establishes acceptance. A release can establish a latest-known acceptance bound, not an invented exact push time. v0.2 does not compute an exact-window rate when that interval makes inclusion ambiguous; keep it in an availability/uncertain stratum. A future interval-aware analysis needs its own definition. Exhausted PR pagination and full defect ascertainment are separate completeness flags.

### Controlled states and actual checks

Record scheduled, started, candidate-written, visible-check outcome, independent-oracle outcome and terminal reason. Failed, blocked, invalid, budget-stopped and never-started remain distinct. Preserve the model's source even when wrong; the controller cannot heal it with gold code.

Check events distinguish parser/type/effect/contract checks, visible application tests, independent semantic oracles and infrastructure failures. A count of verified unrelated contracts is not a successful semantic oracle. A compiler diagnostic plus a subsequent edit can demonstrate interception of that candidate; it does not prove a causal population reduction in production bugs.

One accepted PR, one tool edit, a failed verification, a model request and an independent repair round are separate records. Edits/checks can be incomplete observations; no local-attempt count is reconstructed from Git commit count.

### Change intent, bug attribution and touches

Classify against the requirement at the time of the change, using the issue/request and behavior diff: feature adds a requested capability; migration changes an interface/dependency while preserving required behavior; semantic repair restores an already required behavior; mechanical/infrastructure work changes neither application obligation. Mixed changes retain multiple labels and a reviewed primary intent; unknown intent remains unknown. A feature can introduce a bug and later receive a semantic repair: intent and outcome are independent fields.

A later same-file edit is a touch candidate, not a defect. Link each supported repair to its earlier artifact/obligation, with evidence strength and reviewer disposition. Exclude feature-only follow-ups from repair incidence, retain their workload, and keep ambiguous fixes in an unknown stratum. For a reported sample, review change intent without seeing the language comparison totals, independently double-review a declared subset and publish disagreements. Git history cannot recover every local generated-file edit; measure those only where captured tool traces permit it, with explicit trace coverage.

## Usage adapter and cost accounting

Prefer existing executor.Result / RunMetrics and captured raw usage events over a new CLI-log parser. Each source adapter is tied to an actual version/contract and source digest. Empty repo-filtered queries trigger coverage investigation, including the existing task-ID lookup; they do not synthesise a missing-run or zero-cost row. No raw database queries or second observatory store is introduced.

Record quantities with presence/knownness, not bare default integers:

- fresh input, cache-read input, cache-write input;
- visible output and reasoning, or a known combined generated total with an explicitly unknown split;
- reported original input/total, original inclusive/exclusive scheme and reconciliation status;
- price source/effective version, cost currency, metered/list-price-equivalent/free-local/unknown provenance, and model/provider/harness/settings identities.

Executor and harness shapes are not interchangeable: inspected RunMetrics.InputTokens is cache-inclusive and FreshTokens subtracts recorded cache buckets only when CacheAccounted is true. Motoko's executor normalises cache-inclusive OpenAI-like input to its executor contract and preserves the original in ProviderData; reasoning must remain disjoint from visible output. The adapter reads its actual path, not a universal field-name assumption. A source-zero reason_tokens value without reporting evidence is unknown reasoning, not a proven zero.

Required independent arithmetic fixture:

| Source form | Original counters | Canonical interpretation |
|---|---|---|
| Inclusive input | input 150, cache read 80, cache write 20, visible output 7, reasoning 3 | fresh 50; accounted total 160; fresh+generated KPI 60 |
| Exclusive input | fresh input 50, cache read 80, cache write 20, visible output 7, reasoning 3 | Same accounted total 160 and KPI 60, with no second cache charge |
| Generated total includes reasoning | generated total 10, reported reasoning 3 | visible output 7, reasoning 3; never 13 generated |
| Legacy chain scalar only | input+output known; cache/model/billing not established | Recorded scalar only; no inferred fresh-token or metered-cost delta |

If cache buckets exceed an inclusive input, refuse that reconciliation instead of clamping into an apparently valid result. A known combined generated total can remain usable while its split is unavailable; do not pretend a split is known merely to populate columns.

Charge each leaf request/attempt once. Parent/stage totals can be integrity checks, not additional expenditure. Deduplicate the same provider request only with identity evidence; when two sources may overlap but lack a join key, report overlap_unknown rather than assume their totals are additive. Keep infrastructure replacements, timeouts and failed attempts charged. Oracle authoring, starter construction and model-assisted study preparation have explicit overhead buckets; amortised summaries state the assumed reuse count.

Budget-token measure, headline fresh-token measure and billing-weighted money are separate. The launch manifest defines the former before calls; a package cannot enforce a live budget solely by aggregating receipts after execution. Local runs retain GPU/wall/verification time and hardware context separately from money. Metered and subscription/list-price-equivalent values never silently pool into one dollar-saving claim.

## Semantic pairs and correctness economics

Controlled pairing key: task-family/variant, stage protocol, repeat, model settings, harness revision/config, retention policy and declared treatment. Language remains an arm field. Language-specific starter, dependency and native-tool hashes are pinned and explicitly permitted to differ; the shared comparison contract must match. Reject duplicate arm keys and unexpected config differences, and surface unmatched arms.

Reuse the package's existing semantic-pair validation ideas. Do not feed cross-language records directly into core PairArms, whose key includes Lang, or relabel them into a fake single language. Existing legacy episode pairing does not by itself certify a three-stage trajectory.

Report:

- scheduled/started/completed/blocked/invalid/not-run coverage;
- independent chain/stage success, visible-green but hidden-regressed changes, preserved invariants and failed-check attempt counts;
- total work and cost per independent successful trajectory, with unavailable zero-success/missingness states;
- per-family and per-language differences over matched arms, and unmatched/invalid spent overhead separately;
- historical repair incidence and controlled chain outcomes as separate analyses.

Two repeated families in the pilot do not become 24 independent task families. Population inference is withheld; a later confirmation protocol must specify family-cluster uncertainty and sample sizing. Fast failures cannot win an economic ranking by omitting their unsuccessful work.

## Evidence cards and design promotion

A deterministic card/draft records:

1. Frozen source requirement and failed obligation; exact baseline/fix/oracle/config hashes.
2. Actual diagnostic/check sequence and independent positive/negative/mutation controls.
3. Intended and observed scope: real versus constructed starter, public development versus held-out batch, family/project cluster and related consumers.
4. Attempts, usage/cost coverage and source accounting contract, including missing values and alternate explanations.
5. Reproduction command, expected exit/assertion and actual result; no claim based only on a compiler error keyword.
6. Recurrence/severity, existing package/stdlib/extension reuse audit, one proposed lane and a draft design destination.
7. Qualification state: discovery, reproduced, candidate, design_ready, landed, remeasured; human disposition recorded separately from generated recommendation.

The operator or an explicitly authorised design workflow may turn design_ready into a design-doc-creator draft. A card cannot self-approve a sprint, send messages, dispatch a coordinator task, merge, or publish. Ordinary prioritisation considers independent recurrence and wasted work; a severe single reproduced defect may justify a documented exception. Missing profiling can justify an investigation design, not a confident performance fix.

Initial cards can cite the merged image ownership and whitespace probes, token-coverage gaps and resource-authoring work. The image application repair already shipped: a new stdlib feature is not presumed necessary. The route may be an application helper, reusable test, authoring diagnostic or Motoko extension. Follow PROGRAM's frozen-core boundary. Exposed examples guide development; remeasurement of a claimed general improvement needs new held-out real tasks.

## Implementation boundaries and compatibility

Proposed package modules: index cohort types/validation, run-ledger codec/validation, source-specific receipt normalisation, scoped metrics/reporting and evidence-card rendering. Names/file splits are implementer choices. Pure functions own analysis; read-only collection/import adapters own FS/Process; a package ceiling never grants model execution authority. The task-pack adapter uses existing executors to produce receipts. No provider parser is copied into native AILANG.

Suggested new CLI entry points are explicit run-ledger import/analyse and card rendering, to be finalised in planning; do not claim they exist in 0.1.0 help. Keep ordinary v1 snapshot/report meanings readable. v2 report identity binds actual package/compiler/metric version, inputs, index selection, accounting contract, pair direction and protocol. New reports must not forge a 0.1.0 package identity to obtain old bytes.

Original study reports remain reproducible with their original locked 0.1.0 consumer/compiler. Do not rewrite frozen snapshots, release metadata or prior baseline totals. A new metric definition gets a new series; do not pool an index-scoped v2 number with an all-observation v1 number. Publish 0.2.0 only through the separately authorised normal package workflow after implementation checks.

## Delivery and acceptance

| Milestone | Acceptance | Required negative control |
|---|---|---|
| E1 — Scoped indices | Unique explicit index membership and full mature/censored/excluded accounting while follow-up rows stay available | Add an out-of-index follow-up or excluded smoke: index denominator/outcomes stay fixed; total workload changes in its labelled scope |
| E2 — Strict receipts | Known/missing, scale, cache/reasoning scheme and provenance validated | Inclusive/exclusive fixtures agree; impossible cache split, duplicate JSON keys, overflow and missing accounting flags fail or become unavailable |
| E3 — Attribution/pairs | Leaf-only usage and semantic pairs preserve every arm/attempt | Duplicate parent+leaf, overlapping imports, model/config drift, wrong language relabel and unmatched arm cannot silently aggregate |
| E4 — Reports/cards | Native deterministic reports and evidence-card drafts with counters and source hashes | Zero success, no known costs, all-pass tiny sample, unsupported causal link and a weakened oracle cannot emit a qualified win/design-ready card |
| E5 — Compatibility/release candidate | Offline named tests/properties, meaningful contract outcomes, independent expectations, old consumer replay and source/dry-run evidence | No zero-discovered-test pass; no vacuous contracts or hidden strict warning waiver; old immutable assets remain unchanged |

Native properties cover partition accounting, ordering invariance and receipt deduplication idempotence under a declared source identity. Independent hand-counted fixtures check arithmetic, rather than recording the implementation's first output. Keep all normal tests offline and bounded. Existing live token records are replay fixtures with missing provenance, not a replacement for source-contract fixtures.

## Verification log and conflict surface

| Claim | Evidence read on 2026-10-23 | Consequence |
|---|---|---|
| v0.1 lacks an explicit index-ID selector and has scalar usage | packages/maintenance-bench/types.ail Manifest/Episode at merged 6830469 | Add separate versioned boundaries; do not reinterpret scalars |
| Current metric scopes differ | metrics.ail summarize uses c.eligible/c.mature versus group-filtered allRows; pilot's 6/7 counters | Explicit scope is a measured need, not a guessed bug category |
| Existing ledger primitives can be reused | codec.ail strict boundary helpers; compare.ail semantic keys; snapshot_io.ail new-path output | Reuse rather than duplicate frameworks |
| Cache/cost/reasoning contracts already exist | executor.Result, executor/motoko/{README.md,cost.go}, RunMetrics.CacheAccounted/FreshTokens at observed core HEAD 19e33da | New adapter must preserve the inspected version's semantics |
| Same-language PairArms includes Lang | internal/eval_analysis/paired.go pairKey | A direct cross-language join would be wrong |
| Task lookup can recover repo-query misses | Mature study's three task-ID chains versus empty repo-filter lists | Missing query coverage is not evidence of zero work |
| Qualified source probes exist | Merged study verified-oracles.json and independent replay script | Cards can cite one confirmed AILANG introduction; broader incidence remains unascertained |

This is package/instrument work, not a parser/typechecker/AST/evaluator/VM/effects change. Core implementation beyond receipt plumbing is not implicitly authorised. Preserve ordinary result readers and the current release/rotation gates. If scope crosses a language boundary, add the required conflict-surface analysis and explicit review before execution.

Risks include false precision, unknown source accounting, changed metric populations, correlated task families, incomplete direct-main coverage and automatic roadmap overreach. New schemas, exact identity, independent controls, explicit unavailable states and review gates are the controls; none makes the small pilot a population study.

## Axiom compliance

Canonical reference: [Design Axioms](https://ailang.sunholo.com/docs/references/axioms).

| Axiom | Score | Constraint |
|---|---:|---|
| A1 Determinism | +1 | Canonical identities and fixed accounting definitions |
| A2 Replayability | +1 | Frozen receipts/inputs and versioned outputs |
| A3 Effect legibility | +1 | Pure analytical core and explicit import effects |
| A4 Explicit authority | +1 | Evidence cards confer no execution authority |
| A5 Bounded verification | +1 | Exact bounds, mutation controls and finite offline probes |
| A6 Safe concurrency | 0 | Immutable inputs; no new concurrent mutation protocol |
| A7 Machines first | +1 | Typed states and explicit scope/missingness |
| A8 Minimal syntax | +1 | Existing language/package surface only |
| A9 Cost visibility | +1 | Failed work, cache, overhead and cost provenance retained |
| A10 Composability | +1 | Existing executors, native package and lifecycle protocol |
| A11 Structured failure | +1 | Invalid, ambiguous and unavailable do not become zero/success |
| A12 System boundary | +1 | Separate observations, accounting, claims and approval |

Net **+11**; no negative on A1/A3/A4/A7. This is a design assessment, not proof or permission to implement.

### Hard-violation check

- [x] A1: no implicit model nondeterminism is hidden in deterministic replay claims.
- [x] A3: analysis and model/tool execution keep explicit effect/authority boundaries.
- [x] A4: generated findings never grant run, merge or implementation permission.
- [x] A7: machine-readable states preserve unknowns, invalidity and identity.

## 2026-10-23 foundation implementation

Mark approved the concrete foundation sprint with “execute sprint”. The native package now implements explicit index/workload scopes, strict neutral receipt accounting, retained stage ledger validation, matched-arm economics, exact edit bounds and draft reproduction cards. Both native resource/text development taskpacks qualified against positive, semantic-negative and weakened-oracle controls. Source v0.2.0 is a release candidate; registry publication has not occurred.

Review amendments before model execution: headline token economics uses fresh input plus generated output, with fresh input as a separate diagnostic; unknown output cannot support the headline. Standalone roadmap cards cannot qualify economic claims from a known label; those need identity-bound controlled reports. Card identities and JSON output types are strict. Follow-up windows and observed rename/delete/copy history remain explicit. See the foundation sprint's actual verification and review log.

The complete lifecycle design stays planned: existing-executor staged integration, actual worker/oracle protection, live source receipt qualification and a priced model campaign are later scope. Exposed development controls do not confirm a general language improvement.

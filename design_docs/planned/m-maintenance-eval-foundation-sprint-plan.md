# Sprint Plan: M-MAINTENANCE-EVAL-FOUNDATION

**Status:** Foundation implemented and independently evaluated PASS90/100 on2026-10-23; source PR#160. Registry publication and model campaign remain later work.
**Goal:** Build the repeatable native AILANG evidence ledger and qualify two real-requirement maintenance task families offline, ready for later executor integration.
**Repository:** `sunholo-data/ailang-packages`; package `packages/maintenance-bench`; task assets in `research/real-world-maintenance/taskpacks/v1/`.
**Designs:** [Evidence ledger](m-maintenance-bench-evidence-ledger.md), [task pack](m-maintenance-eval-taskpack.md), [parent study](m-real-world-language-maintenance-study.md). Design review: [PR #159](https://github.com/sunholo-data/ailang-packages/pull/159), head `2a9e41edb5e4b45f7a34d0aea81a71ab1d520869`.
**Duration:** 6–8 engineering days; budget eight with qualification/review buffer. **Risk:** Medium/high, principally cross-language oracle equivalence and accounting provenance.
**Estimated scope:** 3,700 source/test/config lines across 14 bounded milestones, excluding acquired evidence. Estimates are planning envelopes, not a performance commitment.
**AILANG prompt version loaded:** v0.16.7, obtained from `ailang prompt` and `ailang prompt --version-active`; installed runtime v0.54.18 (`cd33d050b2276633c82624453309ebf7fcea9e2f`). Reload at implementation start and pin actual tool identities.

## Current implementation and velocity

- Source v0.1.0 merged in #156, research in #157. Registry lookup confirms `sunholo/maintenance_bench@0.1.0` is available. The two new designs contain no implementation.
- Live baseline `make verify` passed in the isolated package checkout on v0.54.18: 84 native tests, 200 generated property cases, 35 compiled files, smoke, four executed offline examples, lock check and independent expected report. Non-exported tests/examples are compiler inventory notices, not API exports.
- The completed v0.1 sprint records 1,544 physical AILANG lines (1,204 production/example, 340 test) against 2,160 planned, plus 5,249 seconds elapsed attended-agent session. Neither session time nor a single same-day squash establishes engineering LOC/day. Actual productive-day velocity is unavailable; the new capacity allowance is about 463 estimated lines/day, not a measured rate.
- The standard velocity script exited 141 in its piped recent-log display before computing a summary. Scoped git history and the completed sprint JSON were read directly; no rate was invented from its partial output.
- Reuse `codec.ail`, `strict_json.ail`, `cohort.ail`, `lineage.ail`, `compare.ail` and deterministic output/FS boundaries. Current `metrics.summarize` has intentionally different eligible/workload populations; v2 introduces explicit scopes rather than changing frozen v1 numbers.
- Executor/provider parsing remains outside this sprint. We define and validate receipts over existing parsed results; actual source adapters need independently verified version contracts. Nothing infers missing cache/billing fields from a historical scalar.

## Scope and sequence

This sprint covers ledger E1–E5 and taskpack T1 qualification. It creates no new model loop or core eval CLI flag. The next sprint will add the finite stage adapter through existing executors and prove artifact/oracle isolation with a scripted executor. A later launch manifest names model/provider/prices/caps and authorizes the 8-trajectory/24-change campaign; its design ceilings are not a spend estimate.

The pilot remains maintenance-only with competent starters and fresh sessions carrying owned application artifacts. Initial generation plus maintenance, continuous sessions, representative-workday inference, other languages and sealed confirmation are separate cohorts. Exposed historical cases and these adaptations are development evidence. New real requirements are needed to confirm general improvements after tuning.

## Registry reuse audit

Searches executed 2026-10-23: `ailang pkg search maintenance`, `json`, `testing`, `resource`, `ledger`. Maintenance found the existing package; testing found `sunholo/testing_utils@0.1.1`; JSON included `sunholo/agui@0.2.1` and process-backed clients. Resource/ledger matches were unrelated simulation/physics packages. Inspected `pkg info` and `pkg docs` for maintenance_bench, testing_utils and agui.

| Milestones | Decision | Reason |
|---|---|---|
| M1–M8, M13–M14 | contribute: sunholo/maintenance_bench | Extend the existing repository package, reusing its codec/metric/report boundaries; no second analysis package, dependency on itself or provider parser |
| M9–M12 | none | Ownership/text qualification is specific to the sourced requirements; shared stdlib/native tools suffice until equivalent starter qualification demonstrates otherwise |

AG-UI events omit this study's accounting/cohort contract and are not a lossless usage proxy. Native test declarations already express required assertions; no assertion-wrapper dependency is needed, and no claim of current testing_utils incompatibility is inferred from its old gap note. No new dependency is added speculatively. Recheck relevant registry capabilities if implementation discovers a reusable need; do not silently broaden effects or outsource an arm wholesale to another language. No agent inbox handoff is sent by writing this plan.

## Milestones

All paths in M1–M8 are relative to `packages/maintenance-bench/`. Native taskpack packages have their own manifests, effect ceilings, tests and locks; their host oracle is independent qualification glue, while analysis remains AILANG. New names below are proposed, not existing APIs. Each milestone is intended as a 60–90 minute focused coding slice plus validation/review; split one before execution if it cannot fit. Keep modules below 800 lines and prefer short codec helpers over deeply nested binders.

### M1: Explicit index and reviewed change intent (~280 LOC)

**ID:** `M1_INDEX_BOUNDARY`. **Dependencies:** none. **Estimate:** 170 implementation/config + 110 tests/controls.

**Files/examples:** `index_types.ail`, `index_codec.ail`, `index_test.ail`, `SCHEMA.md`; example `examples/scoped_index.ail`.

Reuse strict_json/codec boundary helpers. Add a distinct versioned index with selected IDs, source/cutoff/selection identity and per-row acceptance evidence. Represent primary intent, mixed labels and attribution separately. Preserve legacy Manifest/Episode decoders.

- [x] Reject duplicate or absent selected IDs, duplicate keys, invalid enums, fractional/exponent counts and unknown schemas; explicit null remains unknown.
- [x] A feature that introduces a confirmed defect remains feature-intent with a separate causal outcome. A feature-only follow-up is not a repair.
- [x] Commit-proxy or interval-ambiguous acceptance cannot silently enter an exact-window mature stratum. Reviewed disposition and provenance survive round trip.

### M2: Scoped historical counts and workload (~260 LOC)

**ID:** `M2_SCOPED_METRICS`. **Dependencies:** `M1_INDEX_BOUNDARY`. **Estimate:** 150 implementation/config + 110 tests/controls.

**Files/examples:** `scoped_metrics.ail`, `scoped_metrics_test.ail`, hand-counted index fixtures; finish `examples/scoped_index.ail`.

Compute index/follow-up/canonical-workload/total-workload views without changing legacy metrics.summarize. Keep supported and confirmed repair links separate and include unknown ascertainment/trace coverage.

- [x] Selected index partitions account for every selected row; an out-of-index follow-up or excluded smoke cannot change index denominator/outcomes.
- [x] The same added row changes the correctly labelled workload count and known usage subtotal, with missingness retained.
- [x] A touch, five files and three repair rounds do not become equivalent defect counts; zero denominator is unavailable. Order-invariance and partition properties execute.

### M3: Versioned strict usage receipts (~280 LOC)

**ID:** `M3_USAGE_BOUNDARY`. **Dependencies:** none. **Estimate:** 170 implementation/config + 110 tests/controls.

**Files/examples:** `usage_types.ail`, `usage_codec.ail`, `usage_codec_test.ail`, source-contract fixtures; example `examples/usage_accounting.ail`.

Accept a documented neutral receipt over already parsed executor/harness results, not provider logs. Store source/version/digest, provider request and attempt identity, leaf/summary role, accounting presence, cache/reasoning scheme and price/currency provenance.

- [x] A legacy chain scalar retains only its recorded known total; missing split/model/billing cannot become zero or a fresh-token claim.
- [x] Wrong types, duplicate IDs/keys, invalid or negative counts, unknown contracts and unsupported precision are rejected or explicitly unavailable under the field policy.
- [x] Money remains decimal text. Exact USD micro-unit aggregation uses scale 6 without rounding; excess nonzero precision/overflow is unavailable, trailing zero precision is accepted. Other currencies remain recorded but cannot silently join USD.

### M4: Token reconciliation and leaf-only costs (~300 LOC)

**ID:** `M4_USAGE_RECONCILE`. **Dependencies:** `M3_USAGE_BOUNDARY`. **Estimate:** 180 implementation/config + 120 tests/controls.

**Files/examples:** `usage_normalize.ail`, `usage_aggregate.ail`, `usage_test.ail`; finish `examples/usage_accounting.ail`.

Implement source-qualified inclusive/exclusive input and combined/separate output normalization. Deduplicate only proven request identities; treat parent summaries as integrity checks. Retain failed/retried/infrastructure and preparation overhead buckets.

- [x] Independent fixtures: inclusive input 150 minus cache 80+20 gives fresh 50; visible 7 plus reasoning 3 gives accounted total 160 and fresh+generated 60. Exclusive-input fixture agrees.
- [x] Combined generated 10 with reasoning 3 yields visible 7, never generated 13. Impossible cache splits fail; unknown reason/cache reporting never becomes proven zero.
- [x] Parent plus leaf costs are charged once; conflicting duplicate identities fail; possible cross-source overlap is unavailable. Costly failed attempts remain charged.
- [x] Mixed metered/list-price/local/unknown cost bases are separated. Known subtotal and coverage are distinct from total expenditure and cost per success.

### M5: Scheduled trajectories, checks and continuity (~260 LOC)

**ID:** `M5_LEDGER_STATES`. **Dependencies:** `M3_USAGE_BOUNDARY`. **Estimate:** 150 implementation/config + 110 tests/controls.

**Files/examples:** `run_types.ail`, `run_codec.ail`, `run_test.ail`, staged ledger fixtures; example `examples/maintenance_trajectory.ail`.

Add a finite offline trajectory/stage/attempt/check ledger with immutable source identities and declared retained artifacts. No model executor calls. Keep visible checks, independent oracle results, edits and repair rounds distinct.

- [x] A synthetic 8-trajectory/24-stage schedule accounts for started, completed, failed, blocked, invalid, budget-stopped and never-started rows without counting non-launch as a model error.
- [x] Check prior/output checkpoint identities and the fixed stage schedule; a broken S1 remains the trajectory source for S2 rather than receiving a gold replacement.
- [x] Missing stage, duplicate attempt, wrong checkpoint and forged oracle provenance fail. Usage stays attached to spent attempts even if the terminal artifact fails.

### M6: Semantic pairs and trajectory economics (~240 LOC)

**ID:** `M6_LEDGER_PAIRS`. **Dependencies:** `M4_USAGE_RECONCILE`, `M5_LEDGER_STATES`. **Estimate:** 140 implementation/config + 100 tests/controls.

**Files/examples:** `run_compare.ail`, `run_metrics.ail`, `run_metrics_test.ail`; finish `examples/maintenance_trajectory.ail`.

Implement cross-language keys over family/variant/protocol/repeat/model/harness/settings/retention/treatment. Keep language as an arm and permit pinned language-native tool/starter hashes to differ.

- [x] Reject duplicate arm keys and unintended config changes; preserve unmatched and invalid arms plus their spend. Never relabel language to call core PairArms.
- [x] Chain success requires every mandatory stage and retained invariant to pass independent checks. Visible-green hidden-regression is reported separately.
- [x] All failed work contributes to matched-cohort economic totals; unmatched/invalid overhead remains separately visible. Zero success, incomplete oracle outcomes or missing usage makes affected economic comparisons unavailable.
- [x] Two repeated families remain two clusters. Reports publish descriptive paired counts, never an unsupported population significance or representative-working-day claim.

### M7: Qualified roadmap evidence drafts (~260 LOC)

**ID:** `M7_EVIDENCE_CARDS`. **Dependencies:** `M2_SCOPED_METRICS`, `M6_LEDGER_PAIRS`. **Estimate:** 160 implementation/config + 100 tests/controls.

**Files/examples:** `evidence_card.ail`, `evidence_card_test.ail`, small card fixtures; example `examples/roadmap_card.ail`.

Render deterministic discovery/reproduced/candidate/design_ready/landed/remeasured cards with failed obligation, source/run/oracle hashes, positive/negative/mutation outcomes, usage coverage, alternative explanation, recurrence and reuse audit. Human disposition is separate.

- [x] A missing reproducer, weakened oracle, unsupported causal link or all-pass tiny sample cannot emit a qualified language win or design_ready recommendation.
- [x] A reproduction can support an investigation card with unknown cost, while an economic claim requires its own coverage. A single severe case requires an explicit severity exception.
- [x] Seed discovery/reproduced cards cite the merged ownership/text cases; current application fixes remain recorded. No automatic stdlib deficiency, agent message, dispatch, approval or implementation is generated.

### M8: Deterministic offline reports and command (~260 LOC)

**ID:** `M8_REPORT_CLI`. **Dependencies:** `M7_EVIDENCE_CARDS`. **Estimate:** 160 implementation/config + 100 tests/controls.

**Files/examples:** `ledger_report.ail`, `ledger_cli.ail`, `ledger_report_test.ail`, `ailang.toml`; examples from M2/M4/M6/M7.

Add a separate offline maintenance-ledger [bin] entry with validate/analyse/cards subcommands over explicit files. Keep maintenance-bench/maintenance-collect behavior and v1 readers stable. Version new report series and bind actual declared package/compiler/protocol/accounting identities.

- [x] Same frozen inputs give identical JSON/Markdown/card bytes. Changed index, receipt, protocol or comparison direction changes the report identity.
- [x] Invalid input exits nonzero with field/context; no fallback to an empty successful cohort. Missing costs and observational/controlled evidence remain separate.
- [x] The installed offline command runs outside the source checkout under IO/FS/Env only; it does not request Process, Net or AI. Do not forge a 0.1.0 identity in 0.2.0 output.

### M9: Resource ownership starter pair and contract (~260 LOC)

**ID:** `M9_F1_STARTER_ORACLE`. **Dependencies:** none. **Estimate:** 160 implementation/config + 100 tests/controls.

**Files/examples:** Repository `research/real-world-maintenance/taskpacks/v1/resource-ownership/`: README, source attribution, native AILANG/Python starter packages, public checks, shared neutral fixtures and external oracle; qualification example `examples/qualify_f1.ail` in the starter package.

Freeze the bounded image extraction command/API and file-lifetime contract from Parse #46/#77. Keep competent native inline-resource starters, identical byte obligations and a language-neutral protocol; no whole-platform port. Independent oracle checks outputs/resources, not implementation text.

- [x] Both native starters pass the same baseline cases, including different bytes with repeated names, duplicate content and reordered documents. Record source/runtime/dependency identities.
- [x] External checks detect planted byte corruption and cross-document aliasing while public checks remain unchanged. A nonzero exit before successful compilation is infrastructure evidence, not the desired semantic negative control.
- [x] Attribute the seed and adaptation honestly; all exposed controls are development assets. Baseline native responsibility and allowed dependencies are documented before stage qualification.

### M10: Resource stages and negative controls (~300 LOC)

**ID:** `M10_F1_STAGE_CONTROLS`. **Dependencies:** `M9_F1_STARTER_ORACLE`, `M5_LEDGER_STATES`. **Estimate:** 190 implementation/config + 110 tests/controls.

**Files/examples:** Same F1 directory: `stages/`, native qualified reference revisions/mutations, oracle fixtures and qualification receipts; extend starter-package qualification example.

Provide three successive requirements and native reference checkpoints: file backing, batch/per-document decoded-byte cap, then release/cleanup. References qualify assets, not repair agent trajectories. Keep original bytes/lifetime checks active throughout.

- [x] Both languages pass each stage reference and all still-required older obligations. No-op/pre-fix and targeted weakened-oracle/implementation mutations fail actual assertions.
- [x] Rejected or partially processed documents cannot corrupt earlier successes; cap boundaries, retry, shared content and release/unreleased ownership are exercised.
- [x] Use per-source build/output directories and an explicit qualification script outside agent workspace. Record byte checks and timing; no uncalibrated RSS or physical-race pass claim.

### M11: Text fidelity starter pair and contract (~260 LOC)

**ID:** `M11_F2_STARTER_ORACLE`. **Dependencies:** none. **Estimate:** 160 implementation/config + 100 tests/controls.

**Files/examples:** Repository `research/real-world-maintenance/taskpacks/v1/text-fidelity/`: README, native starter packages, shared input/result fixtures and external oracle; qualification example `examples/qualify_f2.ail` in the starter package.

Freeze the bounded cell-text adapter and native dependency responsibility from Calamine #671/#673. Both native starters implement the same correct initial logical-text contract. Write the exact streaming/interface and structured-record stage requests before references or model observations.

- [x] Both starters preserve baseline logical text, internal/non-ASCII whitespace, decoded character references and empty/error distinctions on shared cases.
- [x] Specify identical semantic responsibility for S2 producer/interface events and S3 structured-text/output fields; provide both arms equally competent native producers/dependencies.
- [x] Independent oracle catches a planted semantic loss after a successful build. Reject unmatched responsibility rather than publishing an AILANG/Python delta; do not present Python assets as historical repairs.

### M12: Text stages and negative controls (~300 LOC)

**ID:** `M12_F2_STAGE_CONTROLS`. **Dependencies:** `M11_F2_STARTER_ORACLE`, `M5_LEDGER_STATES`. **Estimate:** 190 implementation/config + 110 tests/controls.

**Files/examples:** Same F2 directory: `stages/`, native reference revisions/mutations, text/chunk/reference fixtures and qualification receipts; extend starter-package qualification example.

Qualify preserve-space/default edge-ASCII trimming, an exact event/interface revision, and a documented structured-text/output extension. Preserve all earlier logical-text/consumer obligations.

- [x] Both native reference chains pass the identical frozen stage expectations, including event chunk boundaries and relevant CDATA/rich-text/schema cases selected before execution.
- [x] No-op/pre-fix controls and targeted mutants fail specific old/new assertions; oracle weakening is detected by the independent expected-failure ledger.
- [x] Malformed versus empty output and missing consumer fields cannot pass. Fresh build identities and exact source requirements are included in qualification receipts.

### M13: Independent replay and frozen-v1 compatibility (~220 LOC)

**ID:** `M13_REPLAY_COMPAT`. **Dependencies:** `M8_REPORT_CLI`, `M10_F1_STAGE_CONTROLS`, `M12_F2_STAGE_CONTROLS`. **Estimate:** 100 implementation/config + 120 tests/controls.

**Files/examples:** `tests/ledger_integration.sh`, analytical ledger expectations, taskpack `qualify.sh`, package `_smoke.ail`/Makefile additions and a new unfrozen qualification-result directory.

Run bounded no-model native qualification plus scripted ledger ingestion. Combine hand-counted instrument fixtures with actual qualification receipts without presenting either as a model result. Re-run original pinned consumer separately and preserve all frozen study bytes.

- [x] Account for the complete synthetic 8/24 schedule with forced failures, blocked continuation and costly retries; label its instrument-control role visibly.
- [x] Run F1/F2 native starter/reference/no-op/mutation checks independently, record pass/fail counts and assert their intended failure causes. A skipped/weak oracle or stale build cannot qualify a family.
- [x] Replay the old consumer with its actual 0.1.0 lock/compiler and compare original reports byte-for-byte. New v2 reports use a separate series; do not edit frozen pilot/mature studies.

### M14: Release candidate, focused CI and evaluation (~220 LOC)

**ID:** `M14_QUALITY_DOCS_CI`. **Dependencies:** `M13_REPLAY_COMPAT`. **Estimate:** 160 implementation/config + 60 tests/controls.

**Files/examples:** `README.md`, `AGENT.md`, `SCHEMA.md`, `CHANGELOG.md`, `VALIDATION.md`, `DATA_SCIENCE_GAPS.md`, manifest/lock and `.github/workflows/maintenance-bench.yml`; evaluation JSON.

Prepare 0.2.0 feature release-candidate metadata and exports/assets, add explicit offline CI paths/gates for taskpacks and ledger, run focused quality/dry-run and independent sprint evaluation. Keep source PR reviewable via gh; no registry upload or model launch.

- [x] Record actual named/inline/property counts, compile/effects/interface/release/docs badges, smoke and SMT proved/counterexample/unknown/skipped outcomes. No undiscovered-test or vacuous-contract pass.
- [x] Keep baseline 75 uncontracted-export PUB011 limit from growing: add meaningful contracts to new exports or keep helpers internal. Disclose encoding skips as runtime assertions; no widened warning allowlist.
- [x] Verify clean dependency resolution and installed commands; provide scoped source PR with exact tested revision. Do not claim main/registry availability or CI success before verified.
- [x] Run sprint-evaluator against both designs and acceptance controls; resolve material failures before calling the sprint complete. No core compiler/harness edits or default rotation changes belong to this sprint.

## Day-by-day schedule and exit points

| Day | Work and evidence |
|---|---|
| 1 | M1/M2: freeze index/intent rules, hand-count scoped metrics; verify excluded/follow-up controls |
| 2 | M3/M4: receipt boundaries, exact money policy, normalization and deduplication; hand-audit arithmetic |
| 3 | M5/M6: staged states, semantic pairs and unconditional economics over instrument controls |
| 4 | M7/M8: evidence-card qualification, offline report/command and installed example |
| 5 | M9/M10: qualify F1 native starter and three reference stages with mutation receipts |
| 6 | M11/M12: qualify F2 contracts/native arms and three reference stages with mutation receipts |
| 7 | M13: full offline qualification/ledger replay and separately pinned v1 consumer replay |
| 8 | M14: focused CI, quality, documentation, source PR and sprint evaluation; buffer |

A family failing equivalence remains explicitly unpaired/unqualified; do not replace it after seeing language scores. Escalate a genuine missing stdlib/compiler capability with a minimal reproduction and a separate scoped design rather than patching core during package work. Split the delivery if controls expose materially larger work; preserve completed receipts and report the unmet acceptance rather than declare success.

## Example syntax/evidence checklist

Rows describe planned APIs/behavior, not verified current signatures. Contracts on orchestration wrappers are skipped because they delegate; meaningful laws belong to their pure owner functions and native tests/properties.

| Showcase | Contracts | Effects | Native inline tests/properties |
|---|---|---|---|
| `scoped_index.ail` | include in pure owner: count partition and nonnegative bounded maturity predicate; wrapper skip: orchestration only | include: main `! {IO}`; index/scoped helpers pure | include in owner: partition, order invariance and out-of-index negative case; wrapper skip: no new pure logic |
| `usage_accounting.ail` | include in pure owner: bounded exact addition/subtraction preconditions and knownness rules; wrapper skip | include: main `! {IO}`; normalization pure | include in owner: independent 150/80/20/7/3 fixture, overlap/overflow/unknown cases; wrapper skip |
| `maintenance_trajectory.ail` | include in pure owner: scheduled partition/count invariants where encodable; wrapper skip | include: main `! {IO}`; ledger/compare pure | include in owner: prior checkpoint, retained-failure, semantic-key and incomplete-usage controls; wrapper skip |
| `roadmap_card.ail` | include in pure owner: qualification cannot precede mandatory evidence; report unsupported SMT encodings; wrapper skip | include: main `! {IO}`; card rendering pure | include in owner: missing reproduction/weak oracle/coverage blocks qualified claim; wrapper skip |
| F1 `qualify_f1.ail` | include for pure decoded-byte/cap/lifetime predicates where applicable; FS orchestration skip: independent resource checks required | include: native component/stage `! {FS}` as needed, main `! {IO, FS}`; no Net/AI | include: native byte/cap/ownership boundary tests plus independent host assertions; no claim that unit tests replace oracle |
| F2 `qualify_f2.ail` | include for exact edge-trim/preserve/consumer invariants where encodable; orchestration skip | include: parsing helpers pure; main `! {IO}` with FS only if frozen fixture interface needs it | include: Unicode/internal-space/chunk/error cases in owners plus independent host oracle; report actual generator skips |

## Validation and scientific integrity

- Per milestone run `ailang check` for changed modules and targeted native tests. Run source-module inline tables explicitly; package discovery alone is insufficient. Use `ailang fmt --check` after type checking and short helpers to bound formatter work. Reproduce any old formatter waiver on the actual pinned runtime before relying on it; do not automatically expand the four legacy exclusions.
- Final package gates: `make verify`, `make integration`, the new ledger/taskpack qualification targets, and bounded `make scale` when affected algorithms change. `ailang lock --check`, `ailang pkg quality --json .`, `ailang pkg quality --strict --json .` and `ailang publish --dry-run` provide separate actual evidence. Native references get equivalent compile/public-test/oracle execution in both languages.
- Verify meaningful pure contracts with the installed solver and bounded supported flags discovered through `ailang verify --help`. Record verified, refuted, unknown, skipped and PUB016 separately. A fixture pass, runtime assertion and SMT proof are different results.
- Existing CI pins v0.54.17. Qualify the changed sources there as well as v0.54.18, or deliberately amend minimum/lock/CI to the tested released toolchain and record the reason. Do not silently claim v0.54.17 compatibility from a v0.54.18 run.
- Keep strict PUB011's existing numeric ceiling at 75; new public APIs require meaningful contracts, not `ensures true`, and unencodable contracts remain explicit runtime coverage. Other quality gates must pass without a new allowlist. No percentage-of-coverage claim without an actual coverage instrument.
- Host oracles build/read both arm outputs and compare shared semantic requirements. They do not trust submitted test code or classify a compiler failure as the intended semantic assertion failure. No hidden-oracle protection is claimed until the later executor stage proves its actual worker boundary.
- Retain baseline source hashes, exact tool versions and immutable expectation files. Expected outputs are derived independently, including mutation failures; never bless the implementation's first output as ground truth. Original study directories stay frozen.
- This is package/research work: the full Go suite is outside scope. Future harness edits require focused `make test-changed`/`make test-core` and relevant boundaries, not an unbounded full-suite executor gate.

## Approval, state and handoff

Progress state: `.ailang/state/sprints/sprint_M-MAINTENANCE-EVAL-FOUNDATION.json` in the isolated package checkout. It was populated and validated during planning with execution unapproved. Mark subsequently approved execution; current JSON has `execution_authorized=true`, completed implementation milestones and final review/CI closeout state. Do not overwrite the core repository's active sprint or dispatch remote agents. The standard creator was inspected; it syncs/imports GitHub issues and reads messages, so this plan uses its documented JSON schema directly without unrelated inbox mutations, as in the previous package sprint.

Mark's “ok continue” approves progressing from the designs into this concrete plan. The repository routing gate is `design-doc-creator → user approval → sprint-planner → user says "execute sprint" → sprint-executor → sprint-evaluator`. Planning approval is not silently recorded as approval of a plan that had not been written. After the user approves execution, load sprint-executor and execute this sprint locally; handoff can occur in-session without messaging or delegation. Prepare a clean implementation branch in a separate temporary worktree at the reviewed design revision, then copy the approved plan/state into it. Do not switch the dirty original package checkout or add implementation commits to design PR #159. Commit/push the plan with the approved scope, then create a scoped source PR using gh when checks pass. Registry publication and a model campaign are not part of this sprint.

## Execution log

- 2026-10-23: Mark explicitly approved execution. Isolated implementation branch created; original working trees and active core sprint preserved. Independent taskpack/usage work is assigned to local workers under sprint-executor Phase 2B; integration and all shared manifest/CI changes stay with the controller.

- Integration checkpoint: M1–M13 passed. Main185 native tests plus500 generated cases; both native task families and amended host controls pass; old six-report replay byte-identical. M14 independent evaluation passed90/100 and implementation-head CI passed; final docs/state closeout receives its own CI. Source0.2 remains a release candidate.
- Review amendments: immutable/distinct card IDs, standalone economic cards refused; fresh+generated headline versus fresh-input diagnostic; bounded follow-ups and rename lineage; typed JSON oracle equality; create-only evidence; exact edit sums. All pre-model changes and targeted controls are documented in VALIDATION.
- Wrapper evidence amendment: scoped_index/usage_accounting use IO; trajectory/card examples read explicit checked-in fixtures under IO/FS. Pure owner modules hold contracts/native laws; wrappers contain no new pure logic. No vacuous wrapper contracts were added.

## Foundation completion

All14 milestones and45 substantive acceptance criteria passed. Independent evaluation:90/100 at162db525; implementation source40e1e74932a6e6860030d8fca5d78bba2b49248f, CI39971346946 successful. Full report is committed under .ailang/state/evaluations/. Main185 native tests and500 generated cases pass;3/20 SMT contracts proved,17unsupported, strictPUB011/75 explicitly retained. No new formatter waiver. Source/lock/runtime provenance binds the real metric-v2 replay.

Actual net code is4,653 physical .ail/.py/.sh lines against3,700 planned (different scope units exclude docs/JSON/config). Human productive-day velocity and development billing are unmeasured. See docs/sprint-retros/M-MAINTENANCE-EVAL-FOUNDATION-retro.md. Both broad design docs and this companion remain in planned/ because staged executor integration, protected worker/oracle boundaries and live campaign qualification are still pending. Final exact-head CI is checked after this documentation-only closeout; no package source changes accompany it.

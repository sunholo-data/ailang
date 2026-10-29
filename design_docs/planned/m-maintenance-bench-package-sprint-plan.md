# Sprint Plan: M-MAINTENANCE-BENCH-PACKAGE

**Design:** [Real-world language maintenance study — package addition](m-real-world-language-maintenance-study.md#repeatable-ailang-package--attended-scope-addition-2026-10-23)
**Sprint ID:** M-MAINTENANCE-BENCH-PACKAGE
**Status:** Planned; execution not authorised. Study direction and source destination requested/confirmed by Mark, 2026-10-23.
**Source target:** `sunholo-data/ailang-packages`, `packages/maintenance-bench/`, package `sunholo/maintenance_bench@0.1.0`.
**Planned duration:** 4–6 focused days including fixture review, package validation and buffer. Separate from the larger survey's 6–10 research days and any model campaign.
**Estimated total:** 2,160 LOC including meaningful tests; approximate planning quantity, not an empirical velocity forecast.
**Risk:** Medium: evidence semantics, snapshot/API completeness, exact counts and unfamiliar data-analysis workloads.
**AILANG prompt version loaded: v0.16.7** (`ailang prompt`, followed by `ailang prompt --version-active`).

## Goal and Scope

Deliver an installable AILANG library/command that collects bounded public Git/GitHub metadata using Git and `gh`, imports reviewed annotations, and reproduces offline exploratory maintenance comparisons from frozen inputs. All analysis is AILANG. First release includes real-source examples, independent analytical expectations, gap logging and quality evidence. It does not execute the full research survey or controlled model campaign, infer bugs from file touches, or claim AILANG superiority.

The source destination was explicitly confirmed in-session. Repository source publication is the requested release target. A registry release is a separate action; prepare and verify the tarball, but do not upload without that release being authorised. No scheduler, service, dashboard, LLM labeler or new generic runner in this sprint. The existing lifecycle proposal owns later model task execution.

## Planning Baseline (before execution)

- The research design exists; no package source, fixtures or pipeline implementation has been created.
- The installed binary reports `v0.54.17-dirty` at `8f58117f265471584176e2aeb45996c2c9822577`. Pin a clean compiler for reproducible release evidence and record the prompt/compiler distinction.
- Core checkout HEAD observed `731fa1b95`; the target packages checkout is on `build/v0.4.0/decisions-use-case-map`, with inherited Discord source/docs and lockfile edits. Preserve them. Do not implement on that unrelated dirty branch or switch it; use a clean isolated worktree after execution is authorised.
- The velocity helper was run. Its changelog scan returned historical/global entries, not a usable package-specific throughput estimate. The target package checkout has no commits in the 2026-10-16–23 window and its visible tip is September 19. **No defensible measured LOC/day is available.** Estimate by bounded tasks, include buffer, and revise from actual implementation measurements.
- Existing shadow-executor sprint is in progress; do not alter its status or start this one automatically.

## Registry Reuse Audit

Commands run on 2026-10-23: `ailang search statistics`, `stats`, `dataframe`, `csv`, `github`, `benchmark` returned no matches; `analysis` returned `sunholo/audio@0.1.0`; `testing` returned `sunholo/testing_utils@0.1.1`. Read `pkg info` and `pkg-docs` for audio and testing_utils. Audio/DSP analysis is not a cohort/statistics engine. Reuse native std/json, std/list, std/datetime, std/fs, std/process, std/env, std/io and std/clock where applicable; recheck actual exports before writing code. Installed docs for JSON/list/process/FS/env/clock/package authoring were inspected. Read std/datetime before implementing exposure logic.

| Milestone | Decision | Package/capability and rationale |
|---|---|---|
| M1_SCHEMA | none (revised after reproduction) | Published testing_utils 0.1.1 failed on clean v0.54.17; native test expressions suffice; see execution amendments |
| M2_GIT | none | No GitHub/benchmark package match; a narrow read-only Git acquisition adapter is study-specific |
| M3_GITHUB | none | Use requested `gh` subprocess GET acquisition; no connector or new HTTP client package |
| M4_COHORT | none | Study-specific annotations, lineage, censoring and accounting; stdlib supplies primitives |
| M5_METRICS | none | No relevant statistics package found; bounded descriptive operations, not a new general dataframe library |
| M6_REPORT | none | Stable JSON/Markdown reporting over study records; std/json and std/list |
| M7_REPEATABILITY | none (revised after reproduction) | Native package/test tools; independent fixtures/oracles belong to this package |
| M8_SOURCE_RELEASE | none | Existing package quality, [bin], Git/gh and CI mechanisms; no release utility package |

The absence of a search match is dated registry evidence, not proof that no relevant future package can exist. If a suitable package appears before execution, inspect and revise the dependency decision rather than duplicate it.

## Milestones

Each milestone should fit a 60–90 minute coding slice plus its validation/review; split further if it cannot. Every path below is relative to `packages/maintenance-bench/` in the target repository unless prefixed otherwise. Source LOC budgets include tests but exclude large acquired data. No file should approach the 800-line limit.

### ✅ M1: Schema and strict codecs (~300 LOC)

**ID:** M1_SCHEMA. **Dependencies:** none. **Estimate:** 180 source + 120 tests.

Files: `ailang.toml`, `types.ail`, `manifest.ail`, `codec.ail`, `codec_test.ail`; example `examples/offline_report.ail` starts as a documented input-contract example and becomes runnable by M6. Declare exported boundary types and minimal effect ceiling. Begin metadata/docs required by quality; pin testing_utils 0.1.1, release kind feature and experimental stability.

- [ ] Snapshot/annotation/manifest/report schema versions, provenance and explicit unavailable states are documented and decoded strictly.
- [ ] IDs/hashes stay strings; fractional/out-of-range counts, wrong types, duplicate episode IDs and unknown schema versions fail with field context.
- [ ] Missing, false, zero and empty are distinguishable; decoder never repairs malformed research input silently.
- [ ] Native valid/invalid/boundary tests and a codec round-trip property execute with recorded totals; no test_* naming substitute for native declarations.

### ✅ M2: Bounded Git acquisition (~260 LOC)

**ID:** M2_GIT. **Dependencies:** M1_SCHEMA. **Estimate:** 160 source + 100 tests.

Files: `collect_git.ail`, `snapshot_io.ail`, `collect_git_test.ail`; Git history fixtures and example `examples/git_snapshot.ail`. Use argument arrays and explicit refs/cutoffs/canonical path rules. Do not invoke a shell or mutate source repositories. Use a tiny disposable history fixture for execution checks.

- [ ] Capture source repo/ref identities and canonical file changes, including additions, modifications, renames and derived-path classification.
- [ ] Merge/squash/cherry-pick policy is explicit; a rename or mirrored copy does not become a second defect by itself.
- [ ] Non-zero process exit, truncated output, missing executable and incomplete history are errors/incomplete observations even if exec returned Ok.
- [ ] Git output parsing handles spaces/unicode and record delimiters; a bounded test fixture verifies no mutation of the source repo.

### ✅ M3: Bounded GitHub acquisition through gh (~300 LOC)

**ID:** M3_GITHUB. **Dependencies:** M1_SCHEMA, M2_GIT. **Estimate:** 180 source + 120 tests.

Files: `collect_github.ail`, `collect_github_test.ail`; captured small metadata fixtures, example `examples/github_snapshot.ail`. Only allow validated repo identifiers and supported endpoint templates with `gh api --method GET`. GitHub connector use is prohibited by the user's preference.

- [ ] Acquire PR/issue/change/review evidence with page/row/output limits and preserve endpoint/ref/page provenance; include review submissions and inline comments, not just issue comments.
- [ ] Pagination exhaustion and limit/rate/auth failures cannot yield a complete empty cohort; snapshot completeness is explicit.
- [ ] Offline replay of captured responses covers multi-page, missing-data, rejected/unmerged PR and non-zero exit cases without a network dependency.
- [ ] Acquisition outputs immutable snapshot identity; repeated collection never overwrites an earlier source snapshot in place.

### ✅ M4: Reviewed annotations, lineage and cohorts (~320 LOC)

**ID:** M4_COHORT. **Dependencies:** M1_SCHEMA, M2_GIT, M3_GITHUB. **Estimate:** 180 source + 140 tests.

Files: `annotation.ail`, `lineage.ail`, `cohort.ail`, `cohort_test.ail`; example `examples/build_cohort.ail`. Read std/datetime documentation before implementing UTC cutoff/exposure arithmetic. Human-reviewed annotations are separate versioned input.

- [ ] Features, confirmed repairs, mixed/uncertain work, migrations, compiler repairs and mirror updates remain separate; titles/keywords do not create verified labels.
- [ ] One defect/five files counts one defect with breadth five; five visible attempts count five rounds. Public committed-only observations carry unknown local-attempt coverage.
- [ ] 30/90-day cutoff boundaries, unknown introductions, unlabelled rows and censored episodes are explicit and account for every eligible episode.
- [ ] Unknown authorship is not human/AI; copied code and package consumers retain lineage/cluster identity.
- [ ] Partition-accounting and row-order-invariance properties plus adversarial duplicate/missing-link fixtures execute.

### ✅ M5: Descriptive metrics and explicit deltas (~340 LOC)

**ID:** M5_METRICS. **Dependencies:** M4_COHORT. **Estimate:** 180 source + 160 tests.

Files: `metrics.ail`, `compare.ail`, `metrics_test.ail`, `compare_test.ail`; example `examples/compare_projects.ail`. Definitions are fixed in a versioned metric specification, with hand-calculated fixtures.

- [ ] Compute observed linked-repair incidence, repair burden/rounds, touched-file breadth, missingness, censoring and declared-rule medians with explicit denominators.
- [ ] AILANG-minus-comparator percentage-point deltas and ratios preserve sign/direction; zero denominators and unknown values are named states, not NaN/zero.
- [ ] Semantic paired keys preserve language/model/harness/repeat identity, reject duplicates and surface unmatched arms. Do not pass cross-language rows into the existing Lang-keyed PairArms join.
- [ ] Per-project/domain/agent strata remain visible; tiny related cohorts cannot emit population-significance or unsupported confidence claims. Inferential results say not_computed with reason in v0.1.
- [ ] Independent fixtures cover equal rates, negative deltas, both-zero denominators, mixed known/unknown usage, costly failed attempts and zero successes.

### ✅ M6: Stable reports and installable CLI (~280 LOC)

**ID:** M6_REPORT. **Dependencies:** M5_METRICS. **Estimate:** 180 source + 100 tests.

Files: `report.ail`, `cli.ail`, `report_test.ail`; finish examples `examples/offline_report.ail` and `examples/compare_projects.ail`. Declare `[bin]` command `maintenance-bench` instead of a checkout-dependent shell launcher. Proposed subcommands: collect-git, collect-github, validate, analyse, compare, report, verify-snapshot, benchmark; final flag spelling is an implementation choice documented in AGENT.md.

- [ ] Offline commands run with no network/LLM/subprocess requirement; live collection requires an explicit subcommand and Process authority.
- [ ] Canonical JSON and Markdown have stable ordering and include snapshot, annotations, manifest, metric and package/compiler identities; timing goes to a separate record.
- [ ] The same frozen inputs yield byte-identical reports; changed annotations yield a different report identity.
- [ ] Non-zero CLI exit and typed diagnostics distinguish invalid schema, incomplete acquisition and unsupported analysis; a successful empty result never hides a failure.
- [ ] The installed [bin] command reproduces an example without requiring the original source checkout's working directory.

### ✅ M7: Repeatability fixtures and data-science gap ledger (~240 LOC)

**ID:** M7_REPEATABILITY. **Dependencies:** M6_REPORT. **Estimate:** 140 source + 100 tests.

Files: `benchmark.ail`, `_smoke.ail`, `benchmark_test.ail`, `DATA_SCIENCE_GAPS.md`, `fixtures/README.md`, versioned real-source/analytical/scale fixtures; example `examples/scale_bench.ail`.

- [ ] Ship a small source-attributed real-case metadata/annotation fixture and independently calculated analytical expectations; synthetic mutation/scale data has a visibly different role.
- [ ] verify-snapshot rejects planted semantic errors, dropped rows, duplicate repair links and weakened checks; it does not merely compare the program with its own first output.
- [ ] Bounded 1k/10k-row deterministic scale cases emit identical semantic summaries where expected and separate timing/environment records. Measure behaviour, not a promised performance threshold.
- [ ] Gap entries include exact toolchain, reproducible operation/dataset size, observed behaviour, workaround and cost, proposed package/stdlib/compiler ownership, and follow-up status. No speculative language limitation or automatic core change.
- [ ] Smoke and normal tests remain offline and bounded; [bin] benchmark mode requests Clock explicitly. No paid/model runs or full study campaign.

### ✅ M8: Package quality and repository source release (~120 LOC)

**ID:** M8_SOURCE_RELEASE. **Dependencies:** M7_REPEATABILITY. **Estimate:** 120 docs/config; full validation of existing tests rather than superficial new tests.

Files: `README.md`, `AGENT.md`, `CHANGELOG.md` with 0.1.0 entry, package metadata/lock, `VALIDATION.md`; deliberate package index and targeted CI edits in repository root. Add docs on source availability versus registry publication. Avoid unrelated workflow rewrites.

- [ ] Clean isolated clone/worktree resolves the portable lock, installs/runs [bin], and reproduces the independent offline report and smoke result.
- [ ] Record actual compile, named/inline test, property and SMT verified/unknown/skipped totals; run pkg quality --strict and publish --dry-run, reporting gates honestly.
- [ ] Repository metadata points to the package's actual GitHub tree path; changelog/release kind/docs/exports/effects are complete; no credentials or private transcripts in fixtures.
- [ ] Verify gh account before push, push only scoped changes to the requested repository, and create a reviewable PR using gh with exact validation evidence. Do not claim main-branch availability while only a PR exists.
- [ ] Registry upload is not performed without separate release authorisation; dry-run is not registry acceptance. Independent sprint evaluation reviews design and evidence before completion.

## Example Evidence Checklist (Syntax Gate)

Each planned .ail showcase declares effects and actual native tests; implementation may split examples but must retain the evidence rows. These are planned behaviours, not signatures claimed to compile today.

| Example | Contracts | Effects | Inline tests/properties |
|---|---|---|---|
| offline_report.ail | skip: effectful file/report orchestration is outside a pure SMT claim; metrics module owns count/ratio contracts | include: main IO/FS; pure summary helpers no effects | include: empty/unknown evidence yields explicit unavailable output, not a zero rate |
| git_snapshot.ail | skip: external Git acquisition requires integration evidence | include: Process/FS/IO, Env only if reading arguments | include: pure path-role decoding, canonical versus mirror case |
| github_snapshot.ail | skip: API completeness is an acquired fact, not a solver theorem | include: Process/FS/IO, Env for arguments | include: repo-identifier rejection and incomplete-page decoding |
| build_cohort.ail | include: pure exposure predicate/count bounds where encodable; report solver skips | include: main IO/FS; cohort predicates pure | include: exact 30/90-day boundary and unknown-introduction partition |
| compare_projects.ail | include: pure numerator bounds and positive denominator requirements | include: main IO/FS; comparisons pure | include: equal/negative delta and zero-denominator cases |
| scale_bench.ail | include: bounded fixture accounting where encodable; no runtime-speed proof | include: IO/FS/Clock; pure data construction | include: order invariance and population accounting on a small scale case |

## Validation and Data Integrity

From the package root, run the installed documented tools: `ailang lock --check`, `ailang check --package .`, `ailang test --package .`, explicit `ailang test <source/example>` for modules with inline tests, `ailang verify` on the relevant pure modules, `ailang pkg quality --strict .`, and `ailang publish --dry-run`. Use fmt/check commands on changed .ail modules after type checking. Inspect counts rather than accept an exit-zero test run with zero discovered tests. Fix or disclose quality warnings without vacuous contracts.

Package-specific unit/integration/snapshot tests are the executor gates. The core compiler is not modified here; do not run the full Go suite or add a compiler change to satisfy a package failure. Reproduce a suspected compiler/stdlib gap in a scratch artifact, record it, and route a separate scoped design. No claims about unsupported syntax are accepted without live check/reference evidence.

Acquisition tests use deterministic captured pages and a disposable local Git history, never live GitHub as a unit-test dependency. Optional one-repository read-only gh smoke is separately reported; it cannot replace the captured pagination/error tests. Quality smoke must not unexpectedly access the network. Scientific oracle verification includes hand-counted cases and mutations, not just golden-output agreement.

## Proposed Day Breakdown

- Day 1: clean worktree and pinned toolchain; M1 + M2.
- Day 2: M3 + M4, including evidence/exposure fixture review.
- Day 3: M5 and independent arithmetic/denominator review.
- Day 4: M6 + M7; repeatability and growth measurements.
- Day 5: M8, clean-clone check, quality evidence and source PR.
- Day 6: buffer for package/toolchain friction and independent sprint evaluation.

Dependencies are sequential for v0.1; no delegation is requested. Statistical inference, full surveys, automatic labelling, real maintenance model campaigns and registry upload remain separately planned work. The 24-change controlled feasibility campaign in the parent design is not launched by building this package.

## Handoff and Progress State

Progress JSON: `.ailang/state/sprints/sprint_M-MAINTENANCE-BENCH-PACKAGE.json`. Status not_started; every passes field starts null, and execution_authorized is false. The current active sprint is left unchanged. The standard JSON creator was inspected; it imports GitHub messages and can mark message reads, so the equivalent documented schema is written directly without unrelated inbox mutations. Validate using the existing sprint-executor validator.

**Next execution gate:** Mark's explicit `execute sprint` instruction, as required by the user-provided AGENTS.md work-routing table. No autonomous dispatch or implementation starts merely because this plan/JSON exists. Repository source publication is within the requested destination after implementation checks; immutable registry upload is a separate release decision.

## Execution decisions and evidence amendments

- Mark authorised execution in-session: “great yes - sprint plan, then execute”. Source destination remains the existing package monorepo.
- The originally planned registry testing_utils 0.1.1 dependency was reproduced incompatible with clean v0.54.17 (string ++ in assertions.ail). The package uses native assertions, has no external dependencies, and the registry reuse rows now record none with the reproduction. No duplicated assertion library or unrelated registry release.
- Offline `maintenance-bench`, Process-enabled `maintenance-collect`, and Clock-enabled `maintenance-scale` are separate [bin] surfaces so offline analysis does not require Process. This refines the provisional single-command layout.
- Showcase main functions delegate to tested owner modules. Contracts: skip for orchestration wrappers; meaningful maturity contract lives in cohort. Inline tests/properties: skip in wrappers because they contain no new pure logic; codec/cohort/metrics/compare/report/lineage/benchmark owner modules supply native tests and generated properties. `make verify` executes the four offline examples; integration executes acquisition through disposable Git and replayed gh. These actual skips replace the earlier planned include rows, avoiding duplicate tests solely to satisfy a table.
- Four valid native-test ASTs trigger pinned formatter nil-expression failures. Only codec_test.ail, cohort_test.ail, lineage_test.ail and metrics_test.ail receive a formatter waiver; all remain compiled and executed. Other source/examples must pass fmt. Deep binder/interpolation formatting was mitigated by decomposition and join/concat. Reproductions and actual quality/SMT coverage are recorded in package docs.
- Git commit timestamps are explicitly proxy data; accepted-exposure analysis requires reviewed acceptance timestamps. Real AILANG sample merge times were verified through gh PR112/PR228. Artifact metrics now include structured rename/delete/copy history and touch/feature/confirmed-repair frequencies.

## Package sprint completion

All eight milestones passed. [Source PR #156](https://github.com/sunholo-data/ailang-packages/pull/156) contains the package, fixtures, documentation, CI and retrospective. [Implementation CI](https://github.com/sunholo-data/ailang-packages/actions/runs/39930266136) passed at d9699d5; the independent evaluation is in `.ailang/state/evaluations/eval_M-MAINTENANCE-BENCH-PACKAGE_round_1.json`. Later completion commits update documentation/state only; their CI status is visible on the PR.

Actual source: 35 AILANG files / 1,544 physical lines, including 340 native-test lines. 84 native tests and 200 generated property cases passed. One SMT contract verified; 75 uncontracted exports keep strict PUB011 failing, explicitly documented. No registry upload, main-branch merge, broad survey or model campaign is claimed. Parent study and companion plan stay together in planned/ until its later research phases are complete.

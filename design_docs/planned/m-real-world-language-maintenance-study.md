# M-REAL-WORLD-LANGUAGE-MAINTENANCE-STUDY: Real Project Repair and Lifetime Cost

**Status:** Study direction approved by Mark in the attended 2026-10-23 session. Repeatable AILANG package requested; implementation awaits sprint execution. No model campaign approved.
**Created:** 2026-10-23, from Mark's attended research request.
**Target:** Opt-in research, independent of release versions and the existing eval rotation.
**Priority:** P1 candidate; prioritisation follows the feasibility pilot.
**Estimated:** 6–10 engineering/research days for a first observational report; controlled runs separately scoped and budgeted. A 90-day prospective outcome needs 90 days of follow-up.
**Dependencies:** Read access to selected histories; actual agent provenance and usage where available; reproducible application tests; existing lifecycle protocol for controlled replay.

## Problem Statement

We want to know whether AI-written AILANG applications need fewer corrective edits and consume less cumulative development effort than comparable applications in other languages. Initial generation success does not answer this question. Our existing eval suite is repeatedly used to improve the language and tooling; it is a development instrument, not an independent real-world comparison corpus. Saturated tasks add little information about ordinary maintenance.

This study starts from actual applications, accepted changes, user-reported defects, and dependency evolution. It distinguishes a touch to existing source from a repair, and a compiler-intercepted mistake from a defect found after acceptance or deployment. It also separates AI-written canonical source, derived mirrors, application-generated documents, and dependencies.

The survey is exploratory. A few AILANG repositories owned by one organisation cannot establish a population-wide causal language effect, regardless of how many files they contain. Public GitHub histories generally do not expose token usage or every local edit. Missing data remains unknown.

## Goals and Claim Boundaries

| Question | Evidence | Claim permitted |
|---|---|---|
| Do real applications show different repair patterns? | Validated histories, issues, PRs, tests and follow-up | Project/cohort association, conditional on coverage and matching |
| Does AI-written code need repeated repairs? | Evidence-linked authorship plus source lineage and corrective episodes | Association within the observed agent-authored stratum |
| Which mistakes do checks intercept? | Actual failed check and subsequent repair, or an executable historical reproduction | Observed interception; hypothetical prevention separately labelled |
| Is maintenance cheaper in AILANG? | Same real requirements, comparable starters, frozen agent setup and complete traces | Delivered language/toolchain delta for those task families |
| Does an optional check cause an improvement? | Within-language randomised intervention on that check's feedback | Effect of that intervention in the tested environment |

Success is an auditable estimate, including a negative or inconclusive result. No success criterion requires AILANG to win.

## Related Work and Coverage Decision

- [Lifecycle eval pilot](m-lifecycle-eval-pilot.md) already owns staged repair/change/restart protocols, episode accounting, cost-per-success and experimental validity. **Reuse that protocol.** This companion owns real-project sampling, public-history annotation, artifact lineage and semantic cross-language mappings. Its real-source pilot replaces synthetic seed topics for this study; it does not create a second generic lifecycle runner or promote the existing repair-packet intervention without its own decision.
- [Semantic repair packet](m-semantic-repair-packet.md) is a possible later treatment, not a presumed explanation for observed friction.
- [Measurement contract](../../docs/docs/guides/evaluation/measurement-contract.md) supplies validity and pairing principles. [Cohort provenance](m-cohort-manifest-build-provenance.md) is related design work, not assumed to be fully implemented.
- [PROGRAM](../PROGRAM.md) owns routing of reproduced friction. This plan does not edit the mission queue or release bar.
- [Docparse DX history](../implemented/v0_9_2/m-docparse-dx.md) documents real-project feedback informing language work. Historical assertions there are not fresh measurements or proof of present limitations.

Coverage search: both `ailang docs search` and the scaffold's nominal neural search were attempted. Neural output reported fallback-SimHash, so its scores are not semantic duplicate evidence. Manual review found the lifecycle pilot and repair-packet overlap above. This document extends evidence sourcing and comparison, rather than duplicating their implementation plans.

## High-Impact Decisions

| Decision | Proposed choice | Authority before execution | Change cost |
|---|---|---|---|
| Main evidence set | Real application histories and tickets; existing evals only explain development history | Mark approves study scope | High |
| Interpretation | Observational associations separated from controlled language/toolchain deltas | Protocol freeze | High |
| Unit and follow-up | Accepted change episode, defect episode and canonical artifact lineage; fixed 30/90-day windows | Protocol freeze | High |
| Pilot languages | Survey several languages; first controlled pair AILANG/Python, typed comparator next | Mark chooses after feasibility | Medium |
| Model runs | Explicit episode/time/spend caps; same frozen harness and model within each pair | Mark approves concrete run manifest | Medium |
| Data access | Public reads through `gh`; local Git and existing chains/transcript tools | Already authorised read-only planning scope | Low |

### Design Freeze for Execution

- [ ] Approve the cohort manifest, rubric, primary outcomes and minimum worthwhile effects before analysing outcomes.
- [ ] Confirm source visibility, authorship evidence, historical tests and actual token coverage; publish exclusions.
- [ ] Freeze real-task families, evaluators, semantic mappings, splits, models, tools and limits before model runs.
- [ ] Approve the separately costed controlled pilot; follow normal design approval → sprint planning → execute sprint workflow for tooling changes.

Planning and small read-only source spot checks are authorised now. No approval of implementation, paid runs, scheduling, publication, or messages to maintainers is implied.

## Repeatable AILANG Package — Attended Scope Addition, 2026-10-23

Mark requested that this study become a repeatable AILANG package, published as source in the existing `sunholo-data/ailang-packages` repository, and serve as an exploration of AILANG data science. Source destination was explicitly confirmed in-session. Proposed identity: **`sunholo/maintenance_bench`**, version **0.1.0**, directory **`packages/maintenance-bench/`**. Registry publication is a separate release action; repository availability, packaging dry-run and registry acceptance must never be described as equivalent.

The first implementation is a reusable analysis instrument plus an explicit read-only acquisition adapter. It is not the full research campaign and does not launch lifecycle/model trials. All domain decoding, annotations, cohort selection, lineage, aggregations, comparisons and reports are AILANG. Git and `gh` are external acquisition tools, invoked with argument arrays; no GitHub connector, Python/pandas/R analysis engine, or LLM dependency is required to run frozen reports. An optional future language comparator may have an independently implemented statistical oracle, but it is not a runtime dependency of the package.

### Version 0.1 Boundaries

- **Offline analysis:** read a versioned snapshot plus reviewed annotations and cohort manifest; validate identities, construct exposure/cohorts, compute per-project descriptive measures and explicit deltas; emit deterministic JSON and Markdown.
- **Live collection:** explicit `collect-git` and `collect-github` commands capture bounded read-only responses and provenance to a new snapshot. Offline analysis never silently contacts a service. Capture pagination, truncation, non-zero process exits, partial records and missing history as named outcomes.
- **Repeatability benchmark:** `verify-snapshot` checks known independent expectations; `benchmark` exercises the analysis pipeline on frozen real-source fixtures and declared deterministic scale fixtures. It measures this instrument's correctness/time and growth behaviour, not whether AILANG has fewer application bugs. Keep timing records separate from canonical report bytes.
- **Exploratory statistics:** counts, grouped summaries, repair rounds, attributed and unattributed repair burden, censoring/missingness, medians with a declared even-sample rule, observed 30/90-day linked-repair rates, and descriptive paired deltas where mappings exist. No significance/p-values or population intervals until project-cluster methods have their own reviewed specification and independent oracle. Output `not_computed` with a reason rather than suggest inferential support.
- **Manual evidence remains essential:** import reviewed purpose/root-cause/authorship/repair links as data. Keywords may suggest annotation candidates but never certify bugs, AI provenance or introducing changes. No unattended classifier in version 0.1.

### Data and Module Design

Public pure modules proposed: `types`, `manifest`, `codec`, `lineage`, `cohort`, `metrics`, `compare`, `report`; effectful modules: `collect_git`, `collect_github`, `snapshot_io`, `cli`, `benchmark`. Use relative sibling imports, exported boundary types, native Result/Option and closed states for absent/unknown/censored/unmatched/invalid observations. Use module paths `sunholo/maintenance_bench/...` and explicit `[exports]`.

A snapshot records schema version, source/repo/ref identities, collection cutoff, canonical path rules, source response hashes, episode/defect/artifact identifiers, pagination completeness, provenance level, and separately versioned reviewed labels. Preserve unlabelled and ambiguous records. Raw API IDs and hashes are strings; do not coerce them through floating-point JSON numbers. Reject fractional/out-of-range count values, duplicate keys and unsupported schema versions. Ratios preserve numerator/denominator and zero-denominator status before rendering; missing values never become zeros or empty arrays by default.

A report identity binds package version, compiler identity, snapshot digest, annotation digest, manifest digest and metric-definition version. Stable ordering and explicit cutoffs make repeat reports byte-identical apart from a separately emitted timing/acquisition record. New annotations or fixes produce a new report identity instead of rewriting old evidence. Validate semantic task/chain mapping for cross-language runs; never rewrite language labels to reuse an incompatible join.

The manifest declares version 0.1.0, experimental stability, `[release] kind = "feature"`, source repository metadata and an installable `[bin]` command provisionally `maintenance-bench`. Library analytics remain pure. Adapters require explicit FS/Process; CLI arguments/output require Env/IO; only timing requires Clock. Proposed package ceiling is `FS, Process, Env, IO, Clock`; it grants no execution authority. `collect-github` permits fixed read-only `gh api --method GET` endpoint templates, validates repository identifiers, and never accepts a user-supplied shell command. `exec` completion is not success: inspect exit code and truncation even on `Ok` (installed `ailang docs std/process`). Missing tools/authentication/rate-limit pages produce incomplete/error records, not successful empty cohorts.

### Native Data-Science Exploration

Keep `DATA_SCIENCE_GAPS.md` as a structured gap ledger. Each entry includes the exact compiler/prompt identity, smallest reproducer, requested operation, observed diagnostics/limits, dataset size, workaround and measured cost, and proposed ownership: package helper, contribute to reusable package, stdlib, compiler/runtime, or instrumentation. Distinguish a bug from unfamiliar syntax or a missing API name. Reproduce against the installed toolchain and inspect CLI docs/registry before claiming a language gap. Future features/stdlib additions go through their own design gate rather than silently expanding this sprint.

The exercise should test actual data work: strict JSONL ingestion; typed missingness; joins/group-by; file lineage; timestamp/exposure logic; numerical summaries; exact count handling; stable tabular reporting; and bounded growth on larger snapshots. Add deterministic stress fixtures to expose scaling independently of the research corpus. Benchmark the instrument before and after an implementation change on the same snapshot and pinned compiler; report semantic equivalence and time together. Do not let successful implementation of this package count as independent evidence of AILANG superiority.

### Fixtures, Quality and Source Release

Ship small redistributable metadata/annotation examples linked to actual cases plus hand-calculated analytical fixtures. Keep real-source examples and synthetic mutation/scale cases visibly distinct. Expected results must come from reviewed definitions or an independent calculation, not solely from this package's first output. Public source contains no credentials or private agent transcripts. A larger raw survey snapshot can be separately fetched and content-addressed without bloating the package.

Meaningful contracts should express real count bounds, exposure predicates and ratio preconditions where encodable. Run native named/inline tests and properties for partition accounting, codec round-trips and order invariance; preserve actual verified/unknown/skipped counts. Use package quality and dry-run publication checks, not a contract count as proof. The smoke probe is offline, bounded and uses the shipped fixture.

Source release acceptance: a clean isolated checkout of `ailang-packages` installs the package command, reproduces the frozen expected report, passes package-specific checks, and includes README, AGENT.md, CHANGELOG, gap ledger, manifests, source attribution, examples and CI evidence. Push only this package and deliberate index/workflow edits to a reviewable repository branch/PR after the execution gate; do not alter inherited dirty work. Record that source was pushed, distinguish PR/main/registry states, and do not claim a registry release without its result.

See [the package sprint plan](m-maintenance-bench-package-sprint-plan.md). The existing research protocol and held-out-evidence rules above remain binding. Package test fixtures and exploratory roadmap cases cannot silently become sealed confirmation evidence.

### Package Planning Verification

On 2026-10-23: `ailang --version` returned `v0.54.17-dirty`, commit `8f58117f265471584176e2aeb45996c2c9822577`; `ailang prompt` was loaded and `ailang prompt --version-active` returned **v0.16.7**. Execution must pin an actual clean compiler/build identity rather than use the dirty string as reproducibility evidence. Read installed `ailang docs package-authoring`, `std/process`, `std/json`, `std/list`, `std/fs`, `std/env`, `std/clock`, the package repo's AGENTS.md and package manifests. These reference pages support the API/effect claims here; no new syntax or unsupported-construct claim is needed for this design.

Registry searches: `statistics`, `stats`, `dataframe`, `csv`, `github`, `benchmark` found no matches. `analysis` matched `sunholo/audio@0.1.0`; `pkg info` and package docs show audio/DSP analysis, not cohort statistics, so do not depend on it. `testing` matched `sunholo/testing_utils@0.1.1`; inspect/pin it for assertion reuse where helpful while keeping native test declarations. `pkg info sunholo/maintenance_bench` returned not found; recheck before first publication. This is a dated reuse audit, not a permanent claim that the ecosystem lacks statistical packages.

## Study A: Survey Real Repository Histories

### Candidate Cohort

Initial AILANG subjects are `ailang-parse` (deterministic parser/generator modules), `docparse` (hosted application), and `ailang-packages` (only modules demonstrably used by applications). Treat the package monorepo as a project cluster, not each package as an independent organisation. Shared code and dependencies also cluster Parse and its consumers. A benchmark-derived package qualifies through subsequent real use, not its origin alone.

Candidate external projects were spot-checked with `gh api` on 2026-10-23:

| Repository | Reported main language | Comparable scope | Survey role |
|---|---|---|---|
| [python-openxml/python-docx](https://github.com/python-openxml/python-docx) | Python | DOCX structure and generation | Candidate after history/test feasibility |
| [mwilliamson/mammoth.js](https://github.com/mwilliamson/mammoth.js) | JavaScript | DOCX conversion and structure | Candidate; do not label it TypeScript |
| [qax-os/excelize](https://github.com/qax-os/excelize) | Go | Spreadsheet parsing, generation and formula boundaries | Candidate with recent merged repair and feature examples |
| [tafia/calamine](https://github.com/tafia/calamine) | Rust | Spreadsheet decoding and malformed inputs | Candidate with recent merged semantic fixes |
| [dotnet/Open-XML-SDK](https://github.com/dotnet/Open-XML-SDK) | C# | OOXML parts and relationships | Candidate; SDK abstraction differs from an end-user parser |

These are candidates, not matched controls or a random sample of their languages. Metadata confirmed they were not archived; last push alone does not establish maintenance quality. `exceljs/exceljs` and `scanny/python-pptx` were considered but had last pushes in January 2025 and August 2024 respectively, so they are historical reserves rather than automatically included current-working-day controls. The original `python-openxml/python-pptx` URL returned 404; the resolved repository is `scanny/python-pptx`.

For broader claims, expand beyond document processing: find real consumers of policy, authentication and dependency packages with equivalent responsibilities. The discovery target is at least three independent external repositories per reported language/domain stratum, subject to coverage. With fewer, report named project case studies rather than language-wide rates. Additional AILANG files do not solve the small-independent-project problem.

### Inclusion, Sampling and Follow-Up

1. Register eligibility before counting defects: runnable historical tests, inspectable patches and linked requirements, usable source licence, meaningful source history, identifiable canonical paths, and comparable subsystem responsibilities. Record excluded projects and why.
2. Start with **25 randomly selected accepted functional source-change episodes per project**, up to eight project clusters (approximately 200 episodes). Sample all eligible changes, not just `fix:` commits or bug-labelled PRs. Trace their later corrections separately. Oversampled categories need sampling weights and a separate denominator.
3. Snapshot repository refs and collection cutoff. For the mature retrospective 90-day cohort, sample index changes from the preceding 12 months only when 90 days of follow-up exist. Match actual overlapping project ages/calendar periods where possible. Recent changes enter a separate censored cohort; no observed repair is not proof of no bug.
4. Match or stratify by domain, pre-change subsystem size/age, activity, contributor pattern, task category, CI/check configuration, and known agent/harness era. Report support gaps. Stars, file count and language name alone are inadequate controls.
5. Keep all-language, confirmed-agent and unknown-authorship strata separate. If these projects have too few confirmed agent contributions, use the AIDev frame to discover additional domain-matched projects, or proceed with project-history observations only. Do not silently turn mostly human/unknown histories into an AI-language result.

### Evidence Collection and Authorship

Use `gh api`/`gh pr view` and local `git log`, `git show`, rename-aware history and diffs. Follow pagination. Distinguish merged, rejected, abandoned and open work. Fetch issue discussion, PR review submissions, inline comments, later linked fixes, and available CI evidence. Preserve canonical URLs, SHAs and collection timestamps. Missing public workflow logs may have expired; do not infer their former result.

Reuse [AIDev's primary documentation](https://github.com/SAILResearch/AI_Teammates_in_SE3) as an external agent-authorship sampling frame. The README inspected at commit `1ea6cd4f07a12bb7a2d2b59a7ebcfa34c329cd57` names v5 and a March 31, 2026 dataset cutoff; pin the actual dataset revision and schema before use. Its older paper and newer releases describe different populations. Existing PR task labels are candidate labels, not validated defect introductions. Stream relevant columns/rows rather than downloading the full dataset by default.

Authorship evidence levels: (1) source/session trace establishing the generation and repair; (2) explicit agent-authored PR/session metadata; (3) co-author/trailer assertion; (4) unknown. Preserve the level, human edits and mixed authorship. Agent PR authorship does not prove every changed line was generated by that agent. A bot account does not prove LLM authorship, and a human account does not prove unaided work. Do not identify AI code by style.

### Annotation Rubric

| Axis | Allowed values / rule |
|---|---|
| Change intent | Defect repair; feature; refactor; performance; dependency/compiler migration; security maintenance; mechanical sync; tests/docs only; mixed; uncertain |
| Root cause | Application semantics; missing/incorrect requirement; type/interface; effects/authority; dependency; compiler/runtime; harness/infrastructure; build/deployment; unknown |
| Detection stage | Local compiler/verifier; local tests; CI; review; post-acceptance; confirmed deployed/user report; unknown |
| Artifact role | Canonical application source; derived mirror/vendor/generated source; generated product output; tests/fixtures; dependency; harness |
| Evidence strength | Confirmed by reproduction/regression test; directly supported by issue and diff; inferred; unknown |
| Relationship | Repairs a named earlier episode; follow-up attempt for same defect; regression/revert; feature extension; unrelated touch; unresolved linkage |

Define a bug as violation of an already-existing documented/observable requirement. New behaviour is a feature; clarification or a missing earlier specification can remain uncertain. Read the diff and evidence, not just the title. Split mixed changes into supported sub-episodes where possible; otherwise retain mixed and run sensitivity analysis. A security dependency upgrade is not automatically a newly introduced application bug.

Calibrate on 30 episodes, independently annotated twice by a human and a separate reviewer where feasible. Record disagreements, raw agreement and chance-adjusted agreement, then adjudicate. Aim for at least 85% raw agreement on repair-versus-feature; if missed, revise the rubric and re-label calibration before scaling. The classifier, if used later, proposes evidence-linked labels. Blind project/language identity where practical and double-review a random 20% plus all uncertain primary-outcome links. No sub-agents or annotation campaign are launched by this plan.

### Counting Touches and Repair Episodes

One defect touching five files is one defect episode with five-file breadth. Five separate attempts on that defect are five repair rounds if visible. A commit count is not an edit-attempt count: squashed, uncommitted, and reverted work may be invisible. Recover within-session attempts only from actual transcripts; mark public-only counts as a lower bound with unknown missing volume.

Track canonical file and, where reliably recoverable, function lineage from introduction through accepted edits and corrections. Record renames/splits, mirrors, cherry-picks and duplicated dependencies to prevent counting one repair repeatedly. A copied browser mirror is a derived update: count its real maintenance effort, but not another source defect. A generated DOCX/XLSX is product output: its regeneration is not automatically a source repair. AI-written `.ail` source remains canonical source even if an agent originally generated it.

Trace candidate introducing changes using links, regression evidence and history. Blame/SZZ can propose candidates, never establish cause alone. Inspect fixes outside the originally edited lines too. If introduction cannot be established, count a confirmed repair burden but leave introduced-defect attribution unknown. [Empirical SZZ research](https://doi.org/10.1007/s10664-021-10092-4) motivates this distinction.

## Measures and Delta Reporting

| Measure | Definition and comparison |
|---|---|
| Primary observational signal | Fraction of accepted source-change episodes linked to at least one confirmed introduced-defect repair within 30/90 days. Report AILANG minus comparator in percentage points; separately show uncertain attribution and censored episodes. This is observed linked-repair incidence, not true bug prevalence. |
| Repair burden | Confirmed corrective episodes and repair rounds per 100 accepted functional changes, plus linked repair-only churn/breadth. Show application defects, toolchain migrations and mechanical maintenance separately and as total workload. |
| Artifact stability | Fraction of eligible canonical files/functions requiring confirmed corrective edits over a fixed exposure period; repeated repairs and post-acceptance touches by category. Quiet/unexercised code is not assumed correct. |
| Controlled correctness | Unconditional chain completion with all old and newly required behaviour preserved; independent hidden-check failures and escaped regressions per scheduled change. Report paired absolute deltas. |
| Controlled economics | All generation, diagnosis, failed attempts, repair and verification cost divided by independent successes, plus total cost per scheduled chain and AILANG/comparator ratios. Zero successes is undefined/infinite, never zero. Report token categories, human time and latency separately. |

Publish numerators, denominators, uncertainty, missingness and per-project/task distributions. Use project/family-cluster uncertainty; repeated trials, related packages and shared histories are not independent samples. With too few clusters, use descriptive ranges and explicitly withhold population significance claims. Predeclare the primary comparator/outcome and treat other language/category comparisons as exploratory. A larger confirmation sample is sized from pilot event rates/discordance and a pre-agreed worthwhile effect, not an arbitrary number of files.

Public histories do not provide credible token costs or human hours by default. Review duration is not labour time. Never infer token consumption from diff length, commit count or merge time. Track verified defect severities and user exposure where available; raw issue counts and fixes/commits are not language defect rates.

## Study B: Replay Real Maintenance on the Rig

### Source Tasks and Comparability

Select actual, independently documented user issues and requirement changes from the survey, retaining source issue, parent revision, acceptance criteria and later regression checks. Include fixes, features, dependency/interface changes, malformed input, cross-module work and fresh-session handovers. The selection frame should approximate the sampled working-day task distribution; report any deliberate hard-case oversampling separately.

Use bounded existing application slices or real package consumers with equivalent external contracts. Do not port whole mature libraries merely to create a comparison, or compare extending a mature Python library with writing an AILANG replacement from scratch. Validate each starter's common behaviour before tasks begin. Where no comparable starting application exists, mark the task unmatched instead of hiding the construction asymmetry.

Both arms use competent native tools, idiomatic code, existing packages and normal tests. Record exact check configuration and dependencies. AILANG-only application logic is an optional declared stratum; external adapters, SDKs and compiler/runtime work remain visible costs. Prohibit delegating the target task wholesale to the other language in one arm. Keep necessary adapters symmetrical or report the asymmetry. First comparison estimates the delivered language-plus-toolchain ecosystem, not a pure syntax effect.

Real-source examples already found, still requiring executable oracle review:

- Parse list-numbering correction `d7f56bb`: repeated and nested lists must preserve intended numbering. Inspect the pre-fix generator and independent OOXML/render checks.
- Docparse filename disclosure correction `272f43d`: API output must expose the display filename rather than the temporary server path; the commit identifies a prior-code failing check.
- [Excelize #2456](https://github.com/qax-os/excelize/pull/2456): invalid formula ranges should return an error rather than panic. The PR describes pre-fix reproduction and 49 regression cases.
- [Calamine #749](https://github.com/tafia/calamine/pull/749): spreadsheet boolean decoding must distinguish valid values and reject malformed ones. The PR links the prior defect and changes tests.

These sources demonstrate rubric feasibility, not matched AILANG superiority. The two external examples were inspected through `gh`; neither author's identity establishes AI authorship. Recent examples lack 90-day follow-up and belong in task discovery or the censored cohort, not the mature 90-day rate.

### Campaign Shape and Controls

Start with a dry-run feasibility batch: **2 real task chains × 3 successive changes × 2 languages × 1 model × 2 repeats = 8 trajectories / 24 scheduled change episodes**. No language-superiority conclusion from this batch. If oracles and instrumentation pass, propose **6 chains × 5 changes × 2 languages × 2 model tiers × 3 repeats = 72 trajectories / 360 scheduled change episodes**, priced and time-capped before approval. First pair: AILANG/Python; add a typed comparator after feasibility rather than a full factorial expansion immediately.

Carry the agent's own artifacts forward within a chain. Keep initial failure and incomplete chains in the denominator. Reveal later requirements on the predeclared schedule where executable; record blocked continuations instead of replacing bad artifacts with gold code. A fresh-session handover retains only the declared artifacts. Do not conflate uninterrupted long-context work and resumed work: report both conditions separately.

Freeze model/provider/version/settings, harness/tool versions, context/retry budgets and package policy. Same model/harness within each pair; randomise paired run order and balance cache conditions and rig load. Tool outputs, teaching prompts and verification messages count toward consumption. Record any model-assisted task preparation independently. Local inference reports GPU/wall time and hardware context; subscription, metered and list-price-equivalent dollars remain distinct.

Count all within-task attempts, including checks that fail. Capture fresh input, cache reads/writes, generated output and reasoning under the executor's actual accounting contract, avoiding double counts. Distinguish unreported from zero. Store checks run, diagnostics, file edits, model calls, human intervention, artifact hashes and independent verdicts. Count CPU verification time separately. Separate development cost from explicit runtime AI calls; ordinary deterministic execution does not automatically consume LLM tokens in either language.

Use maintenance-only cohorts with equivalent existing starters, and separately report cohorts that include initial application construction. For newly extracted reusable packages, record extraction cost and observed consumer reuse; any amortised result names its reuse count. Existing library availability is part of practical ecosystem performance in both arms.

### Oracle and Holdout Integrity

A task's reference implementation must pass its evaluator and the pre-fix/no-op baseline must fail the targeted requirement. Preserve old tests, independent spec-derived hidden cases, boundary/adversarial inputs and relevant output rendering. Add oracle mutations for dropped fields, wrong errors, weakened assertions and bypassed functionality. Agreement with a language's own goldens is insufficient, as Parse's earlier positioning audit illustrates.

Public reference patches can already be in model training data; block future history and answer retrieval during execution, use recent/private prospective requirements when available, and report residual contamination. Newly drawn variants must reflect real requirements and be independently checked; merely renaming a known benchmark is not a holdout.

Split by issue family/subsystem/shared source lineage before outcomes: development, validation and sealed confirmation. Keep closely related fixtures and package consumers together. Freeze one confirmation batch before run exposure. Development failures may guide the roadmap; observed confirmation cases are retired from independent confirmation after use, while their results remain immutable. Refresh with future real tickets. A private test is not independent once its failures have repeatedly driven development.

Use SWE-EVO and SWE-Milestone for methods, not as ready-made cross-language controls. [SWE-EVO](https://arxiv.org/abs/2512.18470) studies evolution from real project histories. [SWE-Milestone](https://proceedings.mlr.press/v306/deng26d.html) studies continuous milestone sequences and preservation of previous behaviour. [SWE-bench Pro v2](https://labs.scale.com/blog/swe-bench-pro-v2) documents reference/no-op gates and leakage controls worth reusing. Their scores cannot establish an AILANG delta because AILANG is not a matched arm in those measurements.

### Reuse Existing Instruments

Follow the lifecycle pilot's sidecar/manifests rather than introducing a generic runner. First inventory existing staged execution and use `ailang chains chat`, chain data and original session JSONL where present. Existing `RunMetrics` stores validity, verification counts, token/cache accounting and cost provenance; verify each proposed executor actually populates them.

`internal/eval_analysis/paired.go` keys pairs on `(ID, Lang, Trial)`. It cannot directly pair AILANG against Python. Specify and validate a semantic task/chain mapping that preserves original language labels, model/harness/config and repeat identity; never relabel a language to force a join. Reject duplicates, surface unmatched arms and report invalidity/spent overhead. Reuse statistical primitives only after reviewing their assumptions for clustered chain outcomes. No required adapter is claimed implemented by this design.

## Roadmap Feedback

Each development-cohort finding produces an evidence card: real source requirement; reproducer; classification; check that intercepted it or missed it; repair rounds/tokens; affected consumers; matched comparator where available; confidence and limits.

| Observed recurring burden | Candidate route after reproduction |
|---|---|
| Repeated unnecessary source repairs due to diagnostics, interfaces or compiler behaviour | AILANG language/toolchain design |
| Application logic defect or missing domain invariant | Application/package fix and regression check |
| Repeated reimplementation of an operation available in comparator libraries | Package/stdlib candidate with reuse and maintenance-cost case |
| Correct code repeatedly revisited due to lost context or handover | Existing harness extension/context strategy lane |
| Derived mirrors and dependency drift dominate costs | Packaging/build/release tooling, with cost retained in the total |
| Oracle, configuration or accounting failure | Measurement repair; no language claim from affected rows |

Rank by observed recurrence, independent consumers, severity and cumulative wasted effort, with uncertainty visible. A one-off serious security defect still merits action. Ship through the existing design/approval/sprint process. Re-measure targeted changes on development/validation cases, then use new sealed real tasks to test generalisation. Retain regressions and negative results. Do not tune the task mix to restore a favourable average or update a previously published cohort in place.

## Delivery Plan and Acceptance

| Stage | Deliverable | Approximate effort |
|---|---|---|
| 0: Protocol and availability | Frozen candidate/exclusion manifest, authorship/token coverage matrix, 30-episode calibration and rubric | 1–2 days |
| 1: Public/first-party history pilot | Up to 200 evidence-linked episodes, canonical lineage, 30/90-day follow-up and descriptive deltas | 3–5 days |
| 2: Interpretation and task selection | Audited case studies, uncertainty/missingness report, roadmap evidence cards, real-task manifest and run estimate | 2–3 days |
| 3: Controlled feasibility | Implement only necessary approved adapters/oracles, then the separately approved 24-change batch | Separate sprint after instrument inventory |
| 4: Confirmation/prospective follow-up | Sample size from pilot variance; frozen real-task campaign and later 30/90-day outcomes | Separately budgeted; calendar follow-up explicit |

Planned research artifacts: `research/real-world-maintenance/README.md`, versioned protocol/cohort manifest, evidence-linked episode/lineage tables, classification audit, availability report, semantic task mappings, immutable run manifests/results, analysis and roadmap evidence cards. The approved package sprint now implements the repeatable instrument in source PR #156; research artifacts listed here belong to the later survey. Package implementation and validation do not complete that survey or a controlled campaign. No compiler, type-system or runtime semantics are changed by this proposal; a compiler conflict-surface analysis is therefore not applicable here.

Acceptance criteria:

- [ ] Corpus contains real project work with provenance, not repackaged existing eval tasks.
- [ ] Feature/repair/mixed/uncertain labels are calibrated and auditable; agent authorship is not guessed.
- [ ] File touches, repairs, mirror regeneration and committed versus observed local attempts remain distinct.
- [ ] Introduced-defect attribution is validated or unknown; recent/censored changes never masquerade as mature bug-free exposure.
- [ ] Reports distinguish observational associations, controlled ecosystem deltas and optional-check interventions.
- [ ] Full costs, failures, missing token data, mismatches and zero-success cohorts are represented honestly.
- [ ] Real-task oracles reject no-op and seeded regressions; comparative starters meet the same initial contract.
- [ ] No false cross-language join; every scheduled chain/phase accounted for.
- [ ] Independent confirmation survives roadmap feedback via family-level split, freezing and refresh.
- [ ] Documentation and source references checked; future instrument changes pass appropriate existing checks and new validity tests. No implementation tests are claimed run for this documentation-only plan.

Stop or narrow if provenance is too sparse, controls are unmatched, or the oracle cannot distinguish correct behaviour. Expand annotation only after calibration; expand model runs only after accounting/oracle feasibility. Stop at predeclared caps, never when a result first looks favourable.

## Verification Log

| ID | Claim checked during planning | Evidence/result |
|---|---|---|
| V1 | Existing lifecycle experiment overlaps controlled replay | Read `m-lifecycle-eval-pilot.md`; this companion owns real-source cohort/rubric/mapping, reuses its protocol |
| V2 | Candidate projects/languages/activity/licences | `gh api repos/<owner>/<repo>` on 2026-10-23; main-language metadata and stale reserves recorded above; not a full eligibility audit |
| V3 | External histories contain meaningful semantic repairs | `gh api repos/qax-os/excelize/pulls/2456` and `repos/tafia/calamine/pulls/749`; read bodies, merge timestamps and file counts; executable reproduction still pending |
| V4 | Public agent sampling frame exists | Read AIDev primary README via `gh`, pinned commit above; dataset schema/coverage/labels still need validation |
| V5 | Historical local source edits have distinct purposes | Read Parse `2891068`, `d7f56bb`, `0931946` and Docparse `272f43d` via local `git show`; type repair, behaviour repair and mirror update differ |
| V6 | Cross-language pairing needs explicit adaptation | Read `internal/eval_analysis/paired.go`: language is part of join key; read lifecycle pilot's same warning |
| V7 | Token provenance has explicit missingness | Read `internal/eval_harness/metrics.go`: cache-accounted and cost-provenance fields; actual historical/source-session coverage not yet inventoried |
| V8 | Scope does not duplicate old generation analysis | Manual search and linked doc review; search announced fallback-SimHash rather than real neural scores |
| V9 | External methodological precedents | Read primary SWE-EVO, SWE-Milestone, SWE-bench Pro v2 and SZZ research references; no external leaderboard treated as a language comparison |

## Axiom Compliance

Research/instrument design only; scores describe protocol constraints, not demonstrated outcome gains.

| Axiom | Score | Justification |
|---|---:|---|
| A1 Determinism | 0 | Language semantics unchanged; stochastic repetitions explicitly recorded |
| A2 Replayability | +1 | Frozen source, protocol, evaluator and run identities |
| A3 Effect legibility | 0 | Existing language/effect behaviour unchanged |
| A4 Explicit authority | 0 | Read-only planning; runs and tooling have separate approval scope |
| A5 Bounded verification | +1 | Independent oracle controls, bounded cohorts and explicit stopping rules |
| A6 Safe concurrency | 0 | No concurrency semantics or rig scheduling changes |
| A7 Machines first | +1 | Structured evidence, artifact lineage and unknown states |
| A8 Minimal syntax | 0 | No language syntax changes |
| A9 Cost visibility | +1 | Failed attempts, cache, metered/local provenance and human effort remain visible |
| A10 Composability | +1 | Reuse lifecycle protocol and existing chain/measurement tools |
| A11 Structured failure | +1 | Invalid, uncertain, unmatched and censored observations remain explicit |
| A12 System boundary | +1 | Separate application, dependency, runtime, harness and inference boundaries |

**Net +7; no hard violations proposed on A1/A3/A4/A7.** Proceed to review of this plan, not implementation. Any later compiler/runtime or permission change needs its own scoped design and verification.

## 2026-10-23: Merged evidence and concrete eval proposals

The package shipped as `sunholo/maintenance_bench@0.1.0`. [Research PR #157](https://github.com/sunholo-data/ailang-packages/pull/157) is merged at `683046911d27e10ba5ccd44aae19988a62010e80`, including the initial availability and mature-history studies, three historical task-usage receipts and public-source regression reproductions. This is evidence of specific maintenance work and one confirmed AILANG introduction, not completion of the broader comparative research or a language-advantage result.

Mark requested merging those results and designing a useful repeatable eval. The new [maintenance task-pack proposal](m-maintenance-eval-taskpack.md) owns two real-requirement families, three successive changes and existing-executor integration; the [evidence-ledger proposal](m-maintenance-bench-evidence-ledger.md) owns sampled index/follow-up scopes, cost accounting and roadmap cards. They reuse the lifecycle protocol and preserve this study's eligibility/claim rules. Both are planned, not implementation-approved.

Recommended sequence: scope/identity ledger and offline oracle qualification; then the staged adapter; then a separately priced and approved eight-trajectory / 24-maintenance-episode feasibility pilot. No model call, registry release or compiler change is authorised merely by these proposal files.

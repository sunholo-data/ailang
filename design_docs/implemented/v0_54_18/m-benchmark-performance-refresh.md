# M-BENCHMARK-PERFORMANCE-REFRESH

**Status:** Implemented and independently evaluated (95/100); private review ready, production publication pending.
**Target:** Next documentation deployment; current language release v0.54.18
**Priority:** P1
**Estimated:** Two focused development days including validation and contingency
**Date:** 2026-10-23
**Dependencies:** Existing benchmark exporter, rolling bank, documentation theme and saved history

## Problem and Goal

Replace `/docs/benchmarks/performance` with the reviewed A design: an evidence
table, overall trend plots and progressive disclosure of the underlying metrics.
The current page repeats radar plots, trends, performance bars, token charts,
definitions and diagnostic sections. It also hides AILANG-only models through
paired-language filters. The first attempted repair published Argon's data but
verified JSON rather than its visibility in the page.

The page answers both “which models work well for AILANG?” and “how is AILANG
doing compared with Python?”, with model comparison as the main view. Retain
ELO, cost per solved task, timing, outcomes, task detail and historical evidence;
change their presentation rather than deleting research data.

Frequent language releases must not cause redundant paid evaluations. Latest
available evidence is a rolling projection of saved observations. A release
snapshot is a publication, not a claim that every model was rerun on that version.

## Accepted Direction

- A: table with ELO/status, AILANG and optional Python results, cost per solved
  task, timing and measurement provenance.
- Overall AILANG/Python pass-rate and model-ELO plots on the main page. Full and
  recent history, a shared snapshot inspector and selectable ELO model identities.
- Direct-model and coding-agent modes remain separate. Harness identity is shown.
- Undefeated models with no numeric ELO appear first in a visibly separate group;
  complete Core coverage comes before smaller provisional samples. Numeric ELO
  rows then follow in descending order. Other sort choices retain their semantics.
- Preserve missing measurements as unavailable. Genuine measured zero stays zero.
- Use the AILANG documentation theme and approved original logo.

Reviewed reference: `design_docs/mockups/benchmark-redesign/`, A revision 3,
served privately at `https://voights-mac-studio.tail97eda0.ts.net:8450/?view=table&rev=3`.
The HTML is a design reference, not production code to copy wholesale.

## Verification Log

| ID | Source checked | Finding |
|---|---|---|
| V1 | Published JSON and recovered bank files | Standard Argon: 49/49 AILANG final passes, 48/49 first attempts; all 23 current Core IDs covered. Astra: 3/3 standard AILANG passes, zero current Core IDs. Agent Argon: two smoke passes, distinct identity/mode |
| V2 | Four filters in ModelChart, ModelComparisonTable, ModelTokenChart and ModelRadarComparison | All require AILANG and Python data, hiding single-language measurements |
| V3 | `ratings_export.go:fitLeaderboard` | Anchored standard fits omit undefeated/winless numeric ratings. This is not a low score; the UI sorted null last |
| V4 | `types.go:DashboardJSON`, `ModelDimensionStats`, `HistoryEntry` | Existing schema contains mode/language summaries, ELO and version snapshots, but not a complete per-model/mode/language/tier provenance and cost projection |
| V5 | `export_json_models.go`, `efficiency.go` | Some legacy per-model metrics pool languages/modes; timing can fall back from SuccessAtMs to DurationMs |
| V6 | `eval_harness/metrics.go` | CostProvenance already distinguishes metered, list-price-equivalent, free-local and unknown; LLMWallMs and SuccessAtMs exist. DurationMs includes startup/compile/execution, not necessarily full generation latency |
| V7 | `loader.go:LoadResultsFromDirs` | Existing LoadRows adapter owns validity and newest-per-slot dedup. Slot includes model, benchmark, language, seed, mode and trial; this sprint reuses it |
| V8 | Existing history + mockup projection | 61 saved release snapshots; 55 retain model-rating history. The snapshot inspected during design has no fixed-panel direction_index values. No fabricated index is permitted |
| V9 | `docs/src/css/custom.css` and `docs/static/img/ailang-logo.svg` | Inter body, Montserrat headings, original SVG, Sunholo red/teal and existing light/dark tokens. Marketing-site-only style rules do not replace the docs theme |
| V10 | Prototype checks | All three candidate interactions tested against saved records; A trend projection checked at 300/520px chart widths; missing values break paths, zero remains zero. A real browser screenshot was unavailable in the prototype environment |
| V11 | Current publish script and runtime fetch | `publish-unified-dashboard.sh` reads banked results; `benchmarkFetchWithSource` reads live JSON with a visible static-copy fallback. Existing Argon backfill is already published |

Related designs were searched by repository paths and read before this design.
No model calls, neural search, API probes or eval-suite runs were used.

## High-Impact Decisions

| Decision | Authority / state | Consequence |
|---|---|---|
| Table-first A plus overall trend plots | Human, accepted | Replaces the existing performance-page composition |
| Undefeated-first presentation with provisional sample labels | Human, accepted | No synthetic finite ELO or medal; display grouping is distinct from a fitted ranking |
| Rolling evidence across measured language versions | Human, requested | Publication version and measured versions must stay separate |
| Preserve all metrics through detail views/downloads | Human, requested | No deletion of old evidence or neighboring research views |
| Add a typed, additive export projection | Proposed implementation detail | Produces honest scoped metrics; preserves existing consumers and grading |

### Design Freeze

- [x] Layout A, main-page trends and AILANG theme.
- [x] Saved results only; no redundant evaluations or paid review calls.
- [x] Undefeated group first; Astra's small sample must remain explicit.
- [x] Per-row provenance and historical snapshots retained.
- [ ] Authorize production sprint execution after reviewing the plan.

## Solution Design

### 1. Add a scoped evidence projection

Add optional `benchmarkEvidence` to DashboardJSON with its own schema version.
Build it in companion files from the validated results already passed to
ExportBenchmarkJSON, after the existing loader has selected current observations.
Do not add a second dedup, grading, pricing or ELO implementation.

The projection groups model identity, mode, harness and language, with separate
all-task and tier summaries. Each summary includes exact run/pass/first-attempt
counts where retained, repair use, task IDs and Core-set coverage, token summaries,
cost and timing summaries with provenance, and outcome counts. Stable ordering
keeps artifact changes reviewable.

Provenance includes measurement date range, latest reported language version,
all contributing reported versions and counts, prompt versions when retained,
and latest observation date. A mixed-version rollup must say that it spans versions;
the newest version is not relabeled as the version of every contributing run.
Missing/empty provenance stays unavailable and is not guessed from a release tag.

Cost per solved task = contributing saved cost / successes in the same scope.
Retain cost-provenance categories; unknown billing provenance does not become
metered spending. Never blend list-price-equivalent dollars with metered dollars
under a label claiming actual billed cost. Zero marginal local cost is explicit;
unavailable cost and a zero-success denominator have separate states.

Timing names its boundary: generation wall time, agent time-to-success, or legacy
recorded duration. Unknown boundaries must not become an end-to-end speed claim.
Retain available legacy durations in detail, with their label. No inferred
instantaneous timing from absent/zero fields.

Export numeric ELO by existing mode/language fit. A separate evidence status
records undefeated/winless/rated/unavailable as supported by saved counts. The
fitter remains unchanged. Complete Core coverage uses actual required task IDs,
not the number of repeated runs or a comparison with the largest model's count.

### 2. Replace the performance-page composition

Introduce a modular `BenchmarkPerformance` React component and narrowly scoped
projection helpers. Fetch latest JSON once through benchmarkFetchWithSource and
reuse it throughout the page; avoid child components independently mixing snapshots.

The hierarchy is: title and publication provenance; compact overall trends;
comparison controls; one sortable evidence table; selected-model detail; concise
definitions and links to Tasks, Progress, ELO and Value detail. Remove the radar,
duplicate bar/table encodings, repeated filter rows and unrelated axiom scorecard
from this page. Their source evidence and other pages remain available.

Default to latest saved evidence, AILANG with optional Python reference, all
measured tasks and capability ordering. Keep the trend task selector independently
labeled because it is historical direct-model evidence; an agent table must not
silently relabel that history as agent history. Unsupported historical modes have
an explicit unavailable state, never copied standard results.

Model eligibility requires a measurement in the applicable selected language or
visible comparison reference, not both languages. Missing cells show “Not measured”;
no cross-language delta is calculated over different task sets. Search and sort
must not make status/coverage badges disappear. A filtered empty state explains
which scope has no measurements.

For capability ordering, group unrated undefeated records first. Within that
group, complete required Core coverage precedes incomplete samples, followed by
distinct observed-task count and deterministic name ordering. Label the grouping
as undefeated on observed tasks, not a statistically established joint rank.
Numeric ELO rows follow by ELO; other unavailable ratings are last. A numerically
rated agent does not move into the unrated undefeated group. Cost/name/pass-rate
sorts do not secretly use the capability grouping.

### 3. Preserve and clarify historical plots

Use saved HistoryEntry values, with an explicit release-snapshot axis. Pass-rate
plots show AILANG/Python on the same selected task tier where those historic
values exist. Missing measurements break lines. ELO plots use retained anchored
model identities; do not connect different aliases or bridge an anchor change.
Expose suite events, model/task count changes and raw sample counts in inspection.

Label changing historical cohorts as descriptive. Display fixed-panel direction
indices only when recorded, with their panel/anchor and direction of improvement.
An absent index is unavailable, not zero. Do not invent an upward trend or obtain
one by paying to rerun old releases. Keep old snapshots inspectable without
rewriting them using today's model set or ratings.

### 4. Retain detail and compatibility

Model detail exposes first-attempt/final outcomes, failures/refusals/provider
issues, repair, tokens, cost categories and timing boundary. Links lead to the
existing task gallery and research pages; a snapshot download retains all
published fields, not only visible columns.

For older JSON without benchmarkEvidence, a read-only frontend adapter exposes
only compatible known fields. It must display missing scoped metrics explicitly,
not borrow pooled cost, timing or coverage. No forced full-baseline recomputation.
Publication date and measurement date use distinct labels. A static fallback is
identified as a fallback copy, without declaring it stale solely because it was
baked into the site.

## Scope and Boundaries

Change the performance route and the additive evidence export, associated tests,
benchmark overview text and a changelog fragment. Existing ELO/Value/Explorer and
OS pages remain linked and compatible; migrating all of them is a separate task.
This supersedes `m-eval-measured-model-visibility.md` for the performance page;
the broader omission findings remain relevant to later page work.

No compiler/runtime change, no AILANG program, new benchmark, registry/model
configuration, grading policy, billing rate, baseline run, observatory write,
rolling-bank rewrite, ingestion scheduler change or production data deletion.
The parked exact-set global ELO redesign is not pulled into this UI sprint.
Keep unrelated uncommitted work parked.

## Acceptance Criteria

- [ ] Production route uses reviewed A structure and AILANG theme; trends remain
  on the main view, not only a secondary tab.
- [ ] Argon is discoverable at the top undefeated group with 49/49 final and
  48/49 first-attempt counts; Astra shows 3/3 and provisional, 0/23 current Core
  coverage. These fixtures describe the saved design snapshot, not immutable
  assertions about future live data.
- [ ] No fake Python zero, synthetic ELO, inflated rank, or cross-mode/tier cost.
- [ ] Exact task-ID coverage and repeated trials are tested separately.
- [ ] Per-row measurement dates/versions, mixed versions and publication date
  are distinct. Old evidence remains usable after a release with no new evals.
- [ ] Overall pass-rate and ELO charts support full/recent history, missing
  observations, anchor boundaries and a keyboard-accessible snapshot inspector.
- [ ] Existing 61 snapshot fixtures round-trip; the 55 retained rating snapshots
  remain available without fabricating values for the other six.
- [ ] All metric families remain reachable through rows, detail or download;
  absent and zero values are visibly different.
- [ ] Targeted Go/JS tests, relevant repo checks and a production docs build pass.
- [ ] Real rendered-page verification covers desktop/mobile, light/dark, Argon
  visibility, provisional Astra, sorting, filters, chart interaction and fallback.
  Fetching JSON or a successful deployment alone does not satisfy this criterion.
- [ ] No model/evaluation calls were made. Independent sprint evaluation runs
  before publication, followed by concrete final review/deployment authorization.

## Validation and Risks

Use small fixtures extracted from the saved snapshot, rather than embedding the
2.4 MB preview JSON in a test. Test the additive exporter without mutating the
live rolling bank. Keep new code split into files well below 800 lines. Reuse
existing Recharts and docs theme; avoid a new UI/chart framework.

Main risks: missing provenance in imported chains, legacy pooled metrics, absent
historical data, and publication/measurement version confusion. All require
explicit unavailable states. The prototype's raw cost field lacks complete
billing provenance and its legacy duration is not generation latency; those
display shortcuts must not survive the production implementation.

## Axiom Compliance

| Axiom | Score | Reason |
|---|---|---|
| A1 Determinism | +1 | Deterministic projections over saved evidence |
| A2 Replayability | +1 | Retained versions, fixtures and snapshots |
| A3 Effect Legibility | 0 | Explicit read-only fetch and separate publication |
| A4 Explicit Authority | 0 | No new capability or unattended approval |
| A5 Bounded Verification | +1 | Targeted state/metric invariants and rendered checks |
| A6 Safe Concurrency | 0 | Single-page snapshot reuse; no concurrency redesign |
| A7 Machines First | +1 | Missingness and identity survive in structured evidence |
| A8 Minimal Syntax | 0 | No language syntax changes |
| A9 Cost Visibility | +1 | Scoped costs with provenance; no redundant model calls |
| A10 Composability | +1 | Additive export, existing loader and chart library |
| A11 Structured Failure | +1 | Explicit unavailable metrics and failure categories |
| A12 System Boundary | 0 | Compiler/runtime untouched |

Net +7; no hard violations on A1/A3/A4/A7.

## Related Documents

- [Sprint plan](m-benchmark-performance-refresh-sprint-plan.md)
- [Measured-model visibility](../../planned/m-eval-measured-model-visibility.md)
- [Dashboard redesign](../v0_26_0/m-eval-dashboard-redesign.md)
- [Rolling ELO](../../planned/v0_35_0/m-eval-rolling-elo.md)
- [Validity discipline](../../planned/m-eval-validity-discipline.md)

## Implementation Verification

Implemented 2026-10-23 under the user’s explicit “execute sprint” instruction.
Independent sprint evaluation passed 95/100. See the companion
[verification and publication record](m-benchmark-performance-refresh-verification.md).
No evaluation/model inference calls were rerun.

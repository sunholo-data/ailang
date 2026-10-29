# M-EVAL-MEASURED-MODEL-VISIBILITY

**Status**: Performance-page scope superseded by [M-BENCHMARK-PERFORMANCE-REFRESH](../implemented/v0_54_18/m-benchmark-performance-refresh.md), following the attended A mockup review. Other-page omission findings remain recorded here; no execution is authorized by this status note.
**Target**: Next documentation deployment (current release v0.54.18)
**Priority**: P1
**Estimated**: One bounded frontend sprint
**Dependencies**: Existing published results; no new inference or eval runs
**Date**: 2026-10-23

## Problem Statement

The Argon backfill and deployment succeeded, but publishing the data did not make
the model visible in the main comparison views. Verification checked JSON rather
than the rendered product. The user reports absence across benchmark pages.

Live evidence contains `gemini-4-argon` with 49 AILANG standard results (49 final
passes, 48 first-attempt passes), including 23 Core results, and
`gemini-agents-argon` with two passing AILANG smoke results. Neither has Python
measurements. These are distinct model/mode identities and must remain so.

## Verification Log

| ID | Evidence / method | Result |
|---|---|---|
| V1 | Read runtime `/benchmarks/latest.json` on 2026-10-23; inspect `models`, `agentModels`, `tiers.core.model_stats` | Both Argon identities present; languages contain only `ailang`; Core standard count is 23 |
| V2 | Evaluate the predicate from `ModelChart.jsx`, `ModelComparisonTable.jsx`, `ModelTokenChart.jsx` against V1 | All require both AILANG and Python; Argon is excluded |
| V3 | Read `ModelRadarComparison/index.jsx` | Same paired-language filter, then a 12-model cap |
| V4 | Read `internal/eval_analysis/ratings_export.go:fitLeaderboard` | Anchored standard fits omit undefeated and winless models; Argon's standard rating is absent |
| V5 | Read `QualityScatter.jsx` and `ValueScoreTable.jsx` | Models without ELO are omitted; minimum-run gates additionally omit sparse agent results |
| V6 | Read `BenchmarkExplorer/index.jsx:mergeOSData` and model selection | Agent-only models merge into the working model map; standard-only models are deliberately excluded. Live harness data includes `gemini_agents` and its two Argon runs. Browser visibility has not been verified |
| V7 | Deployment 39983989620 succeeded; compare deployed static JSON with validated publication artifact | Static data matches byte-for-byte; another data-only deploy cannot fix V2–V5 |
| V8 | Search `design_docs` for dashboard, ELO and validity designs; read related docs below | Existing coverage-annotation principle applies; parked exact-set ELO redesign remains separate |

No language-support claims are made. Related-doc discovery used repository search
instead of the scaffold's neural search to honor the request for no model calls.

## Goals

Every measured model is discoverable on its applicable benchmark pages, while
unmeasured languages, unavailable ratings, and sparse samples remain explicit.
Fix the shared omission pattern for all models rather than special-casing Argon.

## High-Impact Decisions

| Decision | Why it matters | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| Include single-language measurements | Changes which rows appear in comparisons | human | design | medium |
| Show unavailable ELO as an unranked summary | Preserves visibility without inventing a finite rating | human | design | medium |
| Retain mode, language, tier and coverage boundaries | Prevents comparing unrelated evidence | agent, within this design | implementation | medium |

### Design Freeze

- [ ] Approve single-language rows with missing-language cells and deltas marked unavailable.
- [ ] Approve unranked/omitted-measurement summaries on rating-dependent pages.
- [x] Reuse saved results exclusively; no model calls, new evals or invented Python/ELO values.

## Solution Design

1. Add a small shared frontend projection for measured language evidence. A
   language is measured only when its applicable mode has positive run counts.
   Missing values remain null; measured zero passes remain numeric zero.
2. Performance table and charts accept models with either measured language.
   Render absent table values as `—` with “Not measured”; chart series use null
   for absent values. Gap and token ratio require both languages and a valid
   denominator. Token averages use measured-language counts, rather than halving
   a single-language value. Sorting puts unavailable values last.
3. Radar remains a paired-language comparison. Add a visible summary of measured
   models omitted for missing paired evidence or the display cap, including an
   anchor to their performance rows. Never draw an unmeasured series as zero.
4. ELO page adds a separate, unranked measured-model summary sourced from the
   applicable `models` / `agentModels` evidence when the selected rating block
   has no finite rating for that identity/language. Show measured pass/run counts
   and “ELO unavailable”; identify an undefeated sample only when the actual
   per-benchmark evidence proves it. Exclude these rows from medals, rating bars,
   ranking ranges and ELO deltas. Leave the fitter and its omission rule intact.
5. Value page lists measured models omitted from its rating-based charts/table,
   with explicit reasons such as unavailable ELO or fewer than the minimum runs.
   Do not assign a Value Score or Pareto-frontier status without its inputs.
6. Verify Explorer's existing agent path using the exact deployed JSON. Its
   default/all-tier view must expose `gemini-agents-argon`, `gemini_agents`, two
   smoke runs and the small-sample annotation. Under a Core filter these smoke
   measurements must not become Core results. Standard Argon belongs on standard
   views; do not relabel it as an agent model to make it appear here.

## Scope and Deferred Decisions

Frontend scope: the four components in V2/V3, `EloLeaderboard`, `ValueDashboard`
and their directly used helpers/components as needed. Explorer changes are
limited to a demonstrated display defect or explanatory mode/filter labeling.
The implementer may choose helper names and a compact summary layout.

Raw results, pass grading, ELO fitting, coverage policy, history and model registry
remain authoritative. This does not implement the parked exact-set cohort redesign
or automate observatory-to-bank ingestion. Existing unrelated uncommitted work
stays parked. No benchmark runs, API probes or paid design reviews are required.

## Acceptance and Validation

- [ ] Using the published fixture, default Core performance shows Gemini 4 Argon,
  23/23 AILANG, unavailable Python and unavailable cross-language deltas.
- [ ] All-tier standard evidence shows 49/49 final passes and 48/49 first-attempt
  passes without conflating the two metrics.
- [ ] Missing Python never becomes 0%; a genuine measured 0% remains visible.
- [ ] Single-language token summaries do not divide their measurements by two.
- [ ] ELO and Value pages visibly explain standard Argon's unavailable rating;
  neither fabricates an ELO, rank or Value Score.
- [ ] Agent Explorer exposes the two Argon smoke runs under all tiers, labels
  the limited sample, and does not mix them with the 49 standard runs.
- [ ] Radar accounts visibly for omitted measured models.
- [ ] Existing paired-language rows retain their measured values. Mode and tier
  toggles do not borrow evidence from another scope.
- [ ] Focused tests exercise missing versus zero values, mode separation and
  absent ratings. Run the docs build and verify rendered pages against fixtures;
  a successful JSON fetch alone does not satisfy acceptance.
- [ ] After deployment, verify the visible model and missing-value labels on the
  actual applicable pages. Record any unavailable browser verification explicitly.

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | +1 | Same saved evidence gives the same visible rows |
| A2 Replayability | +1 | Saved publication fixture reproduces the omission and fix |
| A3 Effect Legibility | 0 | Display projection only |
| A4 Explicit Authority | 0 | No new capability or publication permissions |
| A5 Bounded Verification | +1 | Focused fixtures establish missing/zero and scope invariants |
| A6 Safe Concurrency | 0 | No concurrency change |
| A7 Machines First | +1 | Missing measurements stay distinguishable from failures |
| A8 Minimal Syntax | 0 | No language syntax change |
| A9 Cost Visibility | +1 | Saved runs suffice; unavailable value metrics remain explicit |
| A10 Composability | +1 | Shared evidence projection replaces duplicated eligibility rules |
| A11 Structured Failure | +1 | Explains unavailable metrics and omitted samples |
| A12 System Boundary | 0 | Frontend remains a consumer of published evidence |

Net +7. No hard violations on A1/A3/A4/A7.

## Related Documents

- [Dashboard redesign](../implemented/v0_26_0/m-eval-dashboard-redesign.md)
- [Validity discipline](m-eval-validity-discipline.md) — annotation instead of hiding;
  its separately parked exact-set cohort work is not authorized by this proposal.
- [Rolling ELO](v0_35_0/m-eval-rolling-elo.md)

## Risks

Removing a filter without changing numeric fallbacks would manufacture 0% Python
scores. Adding null-ELO rows to the ranked dataset would create false ranks or
NaN bars. Both are specifically excluded by the design and require regression
coverage. Browser verification remains a delivery requirement, not an inference
from deployment success.

# M-EVAL-BENCHMARK-UI-CONSOLIDATION: A coherent benchmark/dashboard information architecture

**Status**: Planned (supersedes the display portion of M-EVAL-DASHBOARD-REDESIGN, which delivered phase 1 — the ELO/regrade `latest.json` schema — already on dev)
**Target**: v0.26.0
**Priority**: P1 (the public benchmark site accreted metrics over many releases; standard/agent mix, stale pages, and an empty local-model story make "actual performance" hard to read)
**Estimated**: 3–4 days (phases A–D below)
**Dependencies**: ELO ratings in `latest.json` (M-EVAL-DASHBOARD-REDESIGN phase 1 — done), `eval-publish` (local rotations), the local-Ollama eval path.

> **📊 CONTEXT (2026-06-12):** Today's fixes already landed on dev — regrade (Python grading
> artifact), dual-mode dedup, the **standard/agent phantom-zero fix** (agent-only models were
> polluting the standard Model Leaderboard), the ELO page added to the sidebar, and the
> OS/local-model section moved to Benchmarks with the 7 stale v0.23.0 snapshots retired. This doc
> is the **consolidation** on top of that: prune the accreted metrics, finish the standard/agent
> split everywhere, give the OS/local cross-language + harness + longitudinal story a real
> (zero-server-cost, static) home, and make the whole section coherent.

## Problem statement

The benchmark site answers too many questions at once and conflates two different audiences:
1. **Cloud frontier** — how do the best API models do on **AILANG vs Python**? (capability, ELO, value.)
2. **OS / local models** — how do open models do across **languages (incl. JS/Go) and harnesses**, **longitudinally**, run cheaply on a local rig?

Symptoms the user flagged: standard and agent numbers mixed in some plots; zeros in sections
(agent-only models on the standard board — now fixed); metrics "built up over time with ideas"
that need pruning; the local-model story has no home; overlapping pages (Explorer vs OS/Local).

## Target information architecture

Two top-level axes under **Benchmarks**, each with a small, fixed set of views:

```
Benchmarks
├─ Overview (NEW landing)        — what we measure, the two axes, links out
├─ Cloud frontier (AILANG + Python)
│  ├─ Model Leaderboard          — regraded AILANG-vs-Python, per-model, one trend chart (PRUNED)
│  ├─ ELO Ratings & Difficulty   — capability + difficulty bands, saturation, artifact flags
│  └─ Value Score                — cost / quality / speed
└─ OS / Local models (cross-language, cross-harness, longitudinal)
   ├─ OS / Local Leaderboard     — JS/Go included, harness comparison, N≥3 trials (static, 0-cost)
   └─ Agent Harness Explorer     — agent cross-harness × language (folds in here)

Codebase Statistics stays as a standalone utility page.
```

**Invariant (the "always split" rule):** every chart/table is explicitly **standard** OR **agent**
— never blended. Where both are shown, they are separate, labeled series/columns. A model appears
on a board only for modes it actually ran.

## Phases

### Phase A — Data & split correctness (finish the pass) · ~0.5d
- Audit every chart in `BenchmarkDashboard/` (ModelChart, LanguageChart, radar, scatter, tables)
  for standard/agent blending. Each must read the correct map (`models` = standard, `agentModels`
  = agent) and label the mode. No model with 0 runs in a mode shows on that mode's view.
- Add a Go test over `ExportBenchmarkJSON` asserting: `models` ⊆ ran-standard, `agentModels` ⊆
  ran-agent, no overlap of empty entries (pins today's phantom-zero fix).
- **Done when:** no zero/empty rows from mode-mixing; spot-check each chart labeled by mode.

### Phase B — Prune the cloud dashboard + dead code · ~1d
- Reduce the Model Leaderboard to a focused headline: regraded AILANG-vs-Python gap, per-model
  table (pass-rate + ELO link), one success-over-time chart. Demote secondary metrics behind a
  "more detail" toggle rather than always-on.
- Delete the 4 confirmed-orphaned components (`LanguageLeaderboard`, `HarnessComparisonTable`,
  `DimensionSelector`, `BenchmarkOverview`). (`BenchmarkMini` is used — keep.)
- **Done when:** the page shows the headline numbers without scroll-fatigue; orphans gone; build green.

### Phase C — OS/Local section with real (static, 0-cost) data · ~2d
- Define the static rotation JSON the OS/Local leaderboard reads (per (benchmark, model, lang,
  harness): pass-rate at N trials + per-release trend), emitted by `ailang eval-publish` into
  `docs/static/benchmarks/os/`.
- A React component (reuse `EloLeaderboard`/`LanguageLeaderboard` patterns) that renders
  cross-language (AILANG/Python/JS/Go) × harness, with a release-trend toggle. Degrades to the
  "refreshing" placeholder when no data.
- Wire it to the local-Ollama rotation output; first dataset comes from a local-rig run (offline,
  no server/API cost).
- **Done when:** a local rotation publishes to the OS/Local page and renders JS/Go + harness columns.

### Phase D — IA coherence (landing + nav) · ~0.5d
- New **Benchmarks Overview** landing page framing the two axes and linking the views; group the
  sidebar into "Cloud frontier" and "OS / Local" sub-sections.
- Fold the Agent Harness Explorer under OS/Local (it's a cross-harness view).
- **Done when:** a first-time visitor can tell, in one screen, what's measured and where to look.

## Non-goals
- No change to how evals are *run* (suites, tiers) — this is presentation + the static publish path.
- No live server / API at view time — everything stays static JSON (0 server cost).
- Cloud stays AILANG+Python; multi-language is an OS/local concern ([[ailang-eval-language-split]]).

## Axiom Compliance
| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | +1 | Static, regenerated-from-results data; the split rule removes mode-mixing ambiguity. |
| A7 Machines First | +2 | The public signal is currently hard to read / partly wrong; coherence + correctness is the point. |
| A9 Cost Visibility | +1 | OS/local longitudinal evals served at zero server cost; surfaces saturation/value. |
| A10 Composability | +1 | Reuses the existing component set + `latest.json`/`eval-publish` plumbing. |
| A11 Structured Failure | +1 | "Always split" + no-empty-rows invariant makes a mode-mix a test failure, not a silent zero. |

## Acceptance
- [ ] Phase A: split-correctness test green; no mode-mixed zeros on any chart.
- [ ] Phase B: pruned Model Leaderboard; 4 orphan components deleted; build green.
- [ ] Phase C: OS/Local page renders a published local rotation with JS/Go + harness.
- [ ] Phase D: Benchmarks Overview landing + grouped nav; Explorer folded under OS/Local.
- [ ] Each phase committed separately on dev; docusaurus-deploy green after each.

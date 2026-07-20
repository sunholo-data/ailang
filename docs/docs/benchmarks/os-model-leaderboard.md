---
title: OS / Local-model leaderboard
sidebar_position: 5
description: Open-source / locally-hosted model evals — cross-language (incl. JS & Go), cross-harness, and longitudinal trends, run continuously on a local rig at zero server cost.
---

import OSLocalLeaderboard from '@site/src/components/OSLocalLeaderboard';
import OSReleaseTrend from '@site/src/components/OSReleaseTrend';

# OS / Local-model leaderboard

Pass-rate and trend data for the **open-source, locally-hosted models** that AILANG's eval rig
runs continuously on dedicated hardware (opencode + local Ollama on a 128 GB Apple-silicon rig).

This is the home for the **cross-language** and **cross-harness** comparison: the cloud
leaderboards ([Model Leaderboard](/docs/benchmarks/performance), [ELO](/docs/benchmarks/elo))
cover **AILANG + Python**, while the multi-language story (**JavaScript, Go**) and the
**harness** comparison (opencode / claude / codex / pi) live here — because running N-trial,
multi-language sweeps on pay-per-token cloud APIs is expensive, and the local rig does it for
**zero server cost**.

## Why a separate section

Pay-per-token cloud APIs make N-trial evaluation expensive — you usually see "one shot, one
model, one benchmark." Running on the local rig lets the project publish, **at no per-token cost**:

- **N≥3 trials per (model, benchmark)** so variance is visible, not hidden behind a single roll.
- **Cross-language** — the same benchmark in AILANG, Python, JavaScript, and Go.
- **Cross-harness** — the same model through different agentic CLIs.
- **Longitudinal trend deltas across releases** — same benchmark + model, last release vs. this.

The numbers feed the language-design feedback loop: a benchmark that fails persistently across N
trials becomes a candidate for a stdlib/prompt/syntax fix, then re-measured at the next release.

## How the data is published

Local-rig rotations are published as **static data** (no live server) via `ailang eval-publish`,
which walks a rotation directory for `summary.json` files (produced by `make eval-smoke … --trials N`),
merges per `(benchmark, model, lang)`, and emits a per-release snapshot with an optional
trend-delta section vs. the previous release:

```bash
ailang eval-publish <version> \
  --rotation eval_results/rotation/<date> \
  --prev     eval_results/rotation/<prev-date> \
  --prev-tag <prev-version>
```

See [the local-Ollama evaluation guide](/docs/guides/evaluation/local-ollama) for how the rotation
directories get populated.

## Current data (latest release)

<OSLocalLeaderboard />

The headline version is the **AILANG release the runs executed against** (results are banked
per-version on the rig, so rows never mix releases); the `rolling-YYYYMMDD` tag is just the
rotation snapshot that published them. The table populates from
`/benchmarks/os/latest.json`. Cloud AILANG-vs-Python leaderboards are on the
[Model Leaderboard](/docs/benchmarks/performance) and [ELO Ratings](/docs/benchmarks/elo) pages.

## Evolution across releases

How each AILANG release changed local-model performance, from the per-release snapshots in
`/benchmarks/os/history.json`:

<OSReleaseTrend />

Two things to keep in mind when reading release-over-release change:

- **Compare within a tier, not the overall number.** Rotations differ in which tiers they
  covered and how many trials they ran (the coverage line under the chart shows both), so the
  overall pass rate moves with the *mix* of benchmarks, not just model performance.
- **The AILANG − Python gap is the language-improvement metric.** Python runs as a control on
  the same model, harness and benchmarks; if AILANG fixes are landing, the gap closes toward
  0pp independent of how strong the underlying model is.

The same per-release local data also appears as the "Local agent" lines on the
[Model Leaderboard](/docs/benchmarks/performance) trend charts, alongside the cloud models.

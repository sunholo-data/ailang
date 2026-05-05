---
sidebar_position: 4
title: Value Score
description: Cost vs quality vs speed analysis — Pareto frontier, weighted value scores, industry-standard "score vs cost" scatter plots
last_updated: 2026-05-05
---

import ValueDashboard from '@site/src/components/ValueDashboard';

# Value Score

Cost-quality-speed analysis for the AILANG eval suite. **Which model gives you the most pass-rate per dollar — or per second?**

:::tip Three lenses on the same data
- **Score vs Cost** — LMArena-style: where does each model sit on the cost / accuracy plane?
- **Score vs Speed** — interactive vs batch tradeoff
- **Value Score Table** — weighted ranking with N=1..4 quality weighting, plus Pareto-frontier flags
:::

## Methodology

Each model's value score combines three orthogonal axes:

```
score = pass_rate^N / (cost_per_success × (1 + median_TTS_seconds / 60))
```

- **`pass_rate`** — fraction of benchmarks the model solved correctly (final, with self-repair)
- **`cost_per_success`** — total dollars spent ÷ number of successful runs
- **`median_TTS_seconds`** — wall-clock time to a passing solution (post-v0.15.1; older runs fall back to total duration)
- **`N`** — quality weighting:
  - `N=1` — pure cost efficiency (raw $/success)
  - `N=2` — balanced (default recommendation)
  - `N=3` — quality matters more than savings
  - `N=4` — quality dominates

The **Pareto frontier** is the set of models where no other model is both cheaper *and* higher pass-rate. Picking a model off the frontier means you're paying more for less. Models on the frontier represent genuine engineering tradeoffs.

<ValueDashboard />

## Why this matters for AILANG

AILANG's [eval suite](/docs/guides/evaluation/) measures real-world AI code generation across multiple harnesses. Without a unified score, comparing models requires juggling pass-rate tables, cost spreadsheets, and latency dashboards.

The score formula here is intentionally simple — multiplicative, transparent, and tunable via the N exponent. It rewards models that are simultaneously cheap, accurate, and fast.

For the underlying definitions and methodology behind cost-as-primary-gate eval semantics (introduced in v0.15.1), see the [Cost-and-Speed Budgets guide](/docs/guides/evaluation/cost-and-speed-budgets).

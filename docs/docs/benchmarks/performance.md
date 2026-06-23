---
sidebar_position: 3
title: Model Leaderboard
description: AI model leaderboard for AILANG code generation — radar comparison, trend charts, and 0-shot / self-repair results across 8 models
last_updated: 2026-04-23
---

import BenchmarkDashboard from '@site/src/components/BenchmarkDashboard';
import ModelRadarComparison from '@site/src/components/ModelRadarComparison';

# Model Leaderboard

Real-world performance metrics for AILANG and Python across multiple AI models, harnesses, and languages.

:::tip Explore and browse
- **[Benchmark Explorer →](/docs/benchmarks/explorer)** — filter by language, harness, and model; cross-harness comparison table
- **[Benchmark Gallery →](/docs/benchmarks/gallery)** — browse all benchmark tasks with pass rates and code samples
- **[Value Score →](/docs/benchmarks/value)** — cost vs quality vs speed analysis with Pareto frontier
:::

## Evaluation Modes

This page shows results from three complementary evaluation approaches:

| Mode | What it tests | Metric |
|------|--------------|--------|
| **Standard API (0-shot)** | Direct model API call — does the model produce correct code on the first attempt? | `zeroShotSuccess` |
| **Self-repair** | One additional attempt after failure — does error feedback help? | `finalSuccess` |
| **Agent mode** | Agentic harness (Claude Code / opencode / Codex / Managed Agents) with multi-turn iteration — real-world developer workflow | `agentSuccessRate` |

Agent mode results are also shown in the [Benchmark Explorer](/docs/benchmarks/explorer) broken down by language and harness.

## Model Comparison

Compare AI model performance across multiple dimensions:

<ModelRadarComparison />

<BenchmarkDashboard showGallery={false} />

## What These Numbers Mean

Our benchmark suite tests AI models' ability to generate correct, working code across 4 languages.

### Success Metrics

- **0-Shot Success**: Code works on first try (no repairs)
- **Final Success**: Code works after M-EVAL-LOOP self-repair
- **Agent Success**: Code works via multi-turn agentic iteration

### Why This Matters

These benchmarks demonstrate:

1. **Type Safety Works**: AILANG's type system catches errors early
2. **Effects Are Clear**: Explicit effect annotations help AI models
3. **Patterns Are Learnable**: AI models understand functional programming
4. **Room to Grow**: Benchmarks identify language gaps and guide development

## How Benchmarks Guide Development

The M-EVAL-LOOP system uses these benchmarks to:

1. **Identify Bugs**: Failing benchmarks reveal language issues
2. **Validate Fixes**: Compare before/after to confirm improvements
3. **Track Progress**: Historical data shows language evolution
4. **Prioritize Features**: High-impact failures guide roadmap

### Case Study: Float Equality Bug

The `adt_option` benchmark caught a critical bug where float comparisons with variables called `eq_Int` instead of `eq_Float`. The benchmark suite detected it, guided the fix, and validated the solution.

**Result**: Benchmark went from runtime_error → PASSING ✅

## Try It Yourself

Want to see AILANG in action?

- **[Interactive REPL](/docs/reference/repl-commands)** - Try AILANG in your browser
- **[Code Examples](https://github.com/sunholo-data/ailang/tree/main/examples)** - 48+ working examples
- **[Getting Started](/docs/guides/getting-started)** - Install and run locally

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/docs/guides/evaluation/eval-loop)

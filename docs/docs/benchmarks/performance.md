---
sidebar_position: 6
title: Benchmark Performance
description: Real-world AI code generation performance metrics for AILANG
last_updated: 2025-11-01
---

import BenchmarkDashboard from '@site/src/components/BenchmarkDashboard';
import ModelRadarComparison from '@site/src/components/ModelRadarComparison';

# AI Code Generation Benchmarks

Real-world performance metrics for AILANG vs Python across multiple AI models.

## Model Comparison

Compare AI model performance across multiple dimensions:

<ModelRadarComparison />

<BenchmarkDashboard />

## What These Numbers Mean

Our benchmark suite tests AI models' ability to generate correct, working code in both AILANG and Python.

### Success Metrics

- **0-Shot Success**: Code works on first try (no repairs)
- **Final Success**: Code works after M-EVAL-LOOP self-repair
- **Token Efficiency**: Lower tokens = more concise code

### Why This Matters

These benchmarks demonstrate:

1. **Type Safety Works**: AILANG's type system catches errors early
2. **Effects Are Clear**: Explicit effect annotations help AI models
3. **Patterns Are Learnable**: AI models understand functional programming
4. **Room to Grow**: Benchmarks identify language gaps and guide development

## Where AILANG Shines

AILANG excels at these problem types:

- **Nested Records**: 100.0% success rate
- **Fizzbuzz**: 100.0% success rate
- **Higher Order Functions**: 100.0% success rate
- **Recursion Factorial**: 100.0% success rate
- **Adt Option**: 100.0% success rate

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

- **[Interactive REPL](/ailang/docs/reference/repl-commands)** - Try AILANG in your browser
- **[Code Examples](https://github.com/sunholo-data/ailang/tree/main/examples)** - 48+ working examples
- **[Getting Started](/ailang/docs/guides/getting-started)** - Install and run locally

## Technical Details

**Version**: v0.4.0

**Total Runs**: 480

**Generated**: 2025-11-01 10:58:09

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |
|-------|------|--------|-------|------------|----------|----------|
| Claude Sonnet 4.5 | 80 | 57.5% | 62.5% | 6445 | $0.0214 | v0.4.0 |
| gpt5-mini | 80 | 57.5% | 61.3% | 5588 | $0.0018 | v0.4.0 |
| gemini-2-5-flash | 80 | 56.2% | 58.8% | 6175 | $0.0025 | v0.4.0 |
| Gemini 2.5 Pro | 80 | 56.2% | 57.5% | 5825 | $0.0100 | v0.4.0 |
| claude-haiku-4-5 | 80 | 43.8% | 55.0% | 6542 | $0.0077 | v0.4.0 |
| gpt5 | 80 | 46.2% | 52.5% | 5770 | $0.0095 | v0.4.0 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Adt Option | 100.0% | 253 | ailang, python |
| ✅ Fizzbuzz | 100.0% | 145 | ailang, python |
| ✅ Higher Order Functions | 100.0% | 270 | ailang, python |
| ✅ Nested Records | 100.0% | 208 | ailang, python |
| ✅ Pattern Matching Complex | 100.0% | 432 | ailang, python |
| ✅ Print With Show | 100.0% | 50 | ailang, python |
| ✅ Records Person | 100.0% | 134 | ailang, python |
| ✅ Recursion Factorial | 100.0% | 103 | ailang, python |
| ✅ Type Safe Record Access | 100.0% | 139 | ailang, python |
| ⚠️ Effect Composition | 91.7% | 272 | ailang, python |
| ⚠️ Explicit State Threading | 91.7% | 263 | ailang, python |
| ⚠️ Immutable Data Structures | 91.7% | 301 | ailang, python |
| ⚠️ Recursion Fibonacci | 91.7% | 123 | ailang, python |
| ⚠️ Referential Transparency | 91.7% | 160 | ailang, python |
| ⚠️ Simple Print | 91.7% | 39 | ailang, python |
| ⚠️ String Manipulation | 91.7% | 121 | ailang, python |
| ⚠️ Effect Tracking Io Fs | 83.3% | 367 | ailang, python |
| ⚠️ Error Handling | 75.0% | 462 | ailang, python |
| ⚠️ Print Missing Effect | 66.7% | 40 | ailang |
| ⚠️ Targeted Repair Test | 66.7% | 46 | ailang |
| ⚠️ Csv To Json Converter | 58.3% | 820 | ailang, python |
| ⚠️ Canonical Normalization | 50.0% | 185 | ailang, python |
| ⚠️ Config File Parser | 50.0% | 867 | ailang, python |
| ⚠️ Deterministic List Transform | 50.0% | 179 | ailang, python |
| ⚠️ List Operations | 50.0% | 232 | ailang, python |
| ⚠️ Tree Transformation Pipeline | 50.0% | 757 | ailang, python |
| ❌ Effect Pure Separation | 41.7% | 290 | ailang, python |
| ❌ Exhaustive Pattern Matching | 41.7% | 171 | ailang, python |
| ❌ No Runtime Crashes Option | 41.7% | 484 | ailang, python |
| ❌ List Comprehension | 33.3% | 364 | ailang, python |
| ❌ Log File Analyzer | 33.3% | 968 | ailang, python |
| ❌ Record Update | 33.3% | 176 | ailang, python |
| ❌ Float Eq | 8.3% | 397 | ailang, python |
| ❌ Multi Module Imports | 8.3% | 400 | ailang, python |
| ❌ Api Call Json | 0.0% | 332 | ailang, python |
| ❌ Cli Args | 0.0% | 419 | ailang, python |
| ❌ Json Encode | 0.0% | 426 | ailang, python |
| ❌ Json Parse | 0.0% | 355 | ailang, python |
| ❌ Numeric Modulo | 0.0% | 332 | ailang, python |
| ❌ Pipeline | 0.0% | 323 | ailang, python |
| ❌ State Machine Traffic Light | 0.0% | 549 | ailang, python |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

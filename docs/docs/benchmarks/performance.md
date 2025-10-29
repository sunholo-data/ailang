---
sidebar_position: 6
title: Benchmark Performance
description: Real-world AI code generation performance metrics for AILANG
last_updated: 2025-10-29
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

- **Explicit State Threading**: 100.0% success rate
- **Fizzbuzz**: 100.0% success rate
- **Type Safe Record Access**: 100.0% success rate
- **Recursion Fibonacci**: 100.0% success rate
- **String Manipulation**: 100.0% success rate

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

**Version**: v0.3.24

**Total Runs**: 458

**Generated**: 2025-10-29 18:56:41

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |
|-------|------|--------|-------|------------|----------|----------|
| Gemini 2.5 Pro | 76 | 53.9% | 55.3% | 5401 | $0.0096 | v0.3.24 |
| claude-haiku-4-5 | 77 | 45.5% | 54.5% | 5880 | $0.0070 | v0.3.24 |
| gpt5 | 75 | 45.3% | 52.0% | 5361 | $0.0092 | v0.3.24 |
| gpt5-mini | 76 | 48.7% | 51.3% | 4921 | $0.0017 | v0.3.24 |
| Claude Sonnet 4.5 | 76 | 50.0% | 51.3% | 5582 | $0.0199 | v0.3.24 |
| gemini-2-5-flash | 78 | 48.7% | 51.3% | 5453 | $0.0024 | v0.3.24 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Adt Option | 100.0% | 275 | ailang, python |
| ✅ Explicit State Threading | 100.0% | 246 | ailang, python |
| ✅ Fizzbuzz | 100.0% | 148 | ailang, python |
| ✅ Recursion Fibonacci | 100.0% | 121 | ailang, python |
| ✅ String Manipulation | 100.0% | 120 | ailang, python |
| ✅ Type Safe Record Access | 100.0% | 159 | ailang, python |
| ⚠️ Nested Records | 91.7% | 174 | ailang, python |
| ⚠️ Print With Show | 91.7% | 50 | ailang, python |
| ⚠️ Referential Transparency | 91.7% | 148 | ailang, python |
| ⚠️ Pattern Matching Complex | 90.0% | 377 | ailang, python |
| ⚠️ Records Person | 88.9% | 133 | ailang, python |
| ⚠️ Tree Transformation Pipeline | 85.7% | 695 | ailang, python |
| ⚠️ Error Handling | 83.3% | 668 | ailang, python |
| ⚠️ Higher Order Functions | 75.0% | 231 | ailang, python |
| ⚠️ Immutable Data Structures | 75.0% | 248 | ailang, python |
| ⚠️ Simple Print | 75.0% | 43 | ailang, python |
| ⚠️ Recursion Factorial | 70.0% | 109 | ailang, python |
| ⚠️ Targeted Repair Test | 66.7% | 39 | ailang |
| ⚠️ List Operations | 60.0% | 205 | ailang, python |
| ⚠️ Effect Composition | 58.3% | 307 | ailang, python |
| ⚠️ Canonical Normalization | 50.0% | 185 | ailang, python |
| ⚠️ Config File Parser | 50.0% | 918 | ailang, python |
| ⚠️ Deterministic List Transform | 50.0% | 175 | ailang, python |
| ⚠️ Effect Pure Separation | 50.0% | 297 | ailang, python |
| ⚠️ Effect Tracking Io Fs | 50.0% | 308 | ailang, python |
| ⚠️ Log File Analyzer | 50.0% | 1181 | ailang, python |
| ⚠️ Print Missing Effect | 50.0% | 35 | ailang |
| ❌ Csv To Json Converter | 41.7% | 766 | ailang, python |
| ❌ No Runtime Crashes Option | 41.7% | 478 | ailang, python |
| ❌ Exhaustive Pattern Matching | 33.3% | 172 | ailang, python |
| ❌ List Comprehension | 33.3% | 265 | ailang, python |
| ❌ Record Update | 8.3% | 175 | ailang, python |
| ❌ Api Call Json | 0.0% | 449 | ailang, python |
| ❌ Cli Args | 0.0% | 358 | ailang, python |
| ❌ Float Eq | 0.0% | 395 | ailang, python |
| ❌ Json Encode | 0.0% | 523 | ailang, python |
| ❌ Json Parse | 0.0% | 440 | ailang, python |
| ❌ Multi Module Imports | 0.0% | 357 | ailang, python |
| ❌ Numeric Modulo | 0.0% | 474 | ailang, python |
| ❌ Pipeline | 0.0% | 442 | ailang, python |
| ❌ State Machine Traffic Light | 0.0% | 544 | ailang, python |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

---
sidebar_position: 6
title: Benchmark Performance
description: Real-world AI code generation performance metrics for AILANG
last_updated: 2025-10-28
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

- **Fizzbuzz**: 100.0% success rate
- **Referential Transparency**: 100.0% success rate
- **Records Person**: 100.0% success rate
- **String Manipulation**: 100.0% success rate
- **Higher Order Functions**: 100.0% success rate

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

**Version**: v0.3.24_clean

**Total Runs**: 204

**Generated**: 2025-10-28 10:29:55

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |
|-------|------|--------|-------|------------|----------|----------|
| claude-haiku-4-5 | 68 | 55.9% | 66.2% | 7000 | $0.0074 | v0.3.24_clean |
| gpt5-mini | 68 | 61.8% | 64.7% | 5647 | $0.0017 | v0.3.24_clean |
| gemini-2-5-flash | 68 | 60.3% | 60.3% | 6056 | $0.0022 | v0.3.24_clean |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Adt Option | 100.0% | 215 | ailang, python |
| ✅ Fizzbuzz | 100.0% | 114 | ailang, python |
| ✅ Higher Order Functions | 100.0% | 302 | ailang, python |
| ✅ Immutable Data Structures | 100.0% | 160 | ailang, python |
| ✅ Nested Records | 100.0% | 132 | ailang, python |
| ✅ Records Person | 100.0% | 141 | ailang, python |
| ✅ Recursion Fibonacci | 100.0% | 83 | ailang, python |
| ✅ Referential Transparency | 100.0% | 116 | ailang, python |
| ✅ String Manipulation | 100.0% | 99 | ailang, python |
| ✅ Type Safe Record Access | 100.0% | 115 | ailang, python |
| ⚠️ Explicit State Threading | 83.3% | 212 | ailang, python |
| ⚠️ Pattern Matching Complex | 83.3% | 347 | ailang, python |
| ⚠️ Print With Show | 83.3% | 35 | ailang, python |
| ⚠️ Simple Print | 83.3% | 25 | ailang, python |
| ⚠️ Numeric Modulo | 66.7% | 153 | ailang, python |
| ⚠️ Targeted Repair Test | 66.7% | 45 | ailang |
| ⚠️ Api Call Json | 50.0% | 150 | ailang, python |
| ⚠️ Canonical Normalization | 50.0% | 306 | ailang, python |
| ⚠️ Deterministic List Transform | 50.0% | 189 | ailang, python |
| ⚠️ Effect Composition | 50.0% | 461 | ailang, python |
| ⚠️ Effect Pure Separation | 50.0% | 290 | ailang, python |
| ⚠️ Effect Tracking Io Fs | 50.0% | 245 | ailang, python |
| ⚠️ Error Handling | 50.0% | 540 | ailang, python |
| ⚠️ Exhaustive Pattern Matching | 50.0% | 149 | ailang, python |
| ⚠️ Json Parse | 50.0% | 122 | ailang, python |
| ⚠️ List Operations | 50.0% | 222 | ailang, python |
| ⚠️ No Runtime Crashes Option | 50.0% | 393 | ailang, python |
| ⚠️ Recursion Factorial | 50.0% | 72 | ailang, python |
| ❌ Float Eq | 33.3% | 225 | ailang, python |
| ❌ Json Encode | 33.3% | 151 | ailang, python |
| ❌ Print Missing Effect | 33.3% | 31 | ailang |
| ❌ Record Update | 33.3% | 156 | ailang, python |
| ❌ List Comprehension | 16.7% | 309 | ailang, python |
| ❌ Cli Args | 0.0% | 231 | ailang, python |
| ❌ Pipeline | 0.0% | 355 | ailang, python |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

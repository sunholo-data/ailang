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

- **Fizzbuzz**: 100.0% success rate
- **Nested Records**: 100.0% success rate
- **Type Safe Record Access**: 100.0% success rate
- **Records Person**: 100.0% success rate
- **Immutable Data Structures**: 100.0% success rate

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

**Version**: v0.3.23

**Total Runs**: 272

**Generated**: 2025-10-29 18:03:34

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |
|-------|------|--------|-------|------------|----------|----------|
| Claude Sonnet 4.5 | 68 | 58.8% | 63.2% | 5696 | $0.0182 | v0.3.23 |
| claude-haiku-4-5 | 68 | 44.1% | 58.8% | 5922 | $0.0063 | v0.3.23 |
| gemini-2-5-flash | 68 | 52.9% | 57.4% | 5495 | $0.0021 | v0.3.23 |
| gpt5-mini | 68 | 52.9% | 57.4% | 4946 | $0.0016 | v0.3.23 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Adt Option | 100.0% | 303 | ailang, python |
| ✅ Fizzbuzz | 100.0% | 180 | ailang, python |
| ✅ Immutable Data Structures | 100.0% | 235 | ailang, python |
| ✅ Nested Records | 100.0% | 229 | ailang, python |
| ✅ Print With Show | 100.0% | 50 | ailang, python |
| ✅ Records Person | 100.0% | 197 | ailang, python |
| ✅ String Manipulation | 100.0% | 132 | ailang, python |
| ✅ Type Safe Record Access | 100.0% | 139 | ailang, python |
| ⚠️ Error Handling | 87.5% | 577 | ailang, python |
| ⚠️ Explicit State Threading | 87.5% | 248 | ailang, python |
| ⚠️ Higher Order Functions | 87.5% | 279 | ailang, python |
| ⚠️ Pattern Matching Complex | 87.5% | 402 | ailang, python |
| ⚠️ Recursion Factorial | 87.5% | 111 | ailang, python |
| ⚠️ Recursion Fibonacci | 87.5% | 119 | ailang, python |
| ⚠️ Referential Transparency | 87.5% | 164 | ailang, python |
| ⚠️ Simple Print | 75.0% | 79 | ailang, python |
| ⚠️ Targeted Repair Test | 75.0% | 41 | ailang |
| ⚠️ Effect Tracking Io Fs | 62.5% | 416 | ailang, python |
| ⚠️ Canonical Normalization | 50.0% | 193 | ailang, python |
| ⚠️ Effect Composition | 50.0% | 278 | ailang, python |
| ⚠️ Effect Pure Separation | 50.0% | 300 | ailang, python |
| ⚠️ List Operations | 50.0% | 185 | ailang, python |
| ⚠️ No Runtime Crashes Option | 50.0% | 543 | ailang, python |
| ⚠️ Print Missing Effect | 50.0% | 39 | ailang |
| ❌ Deterministic List Transform | 37.5% | 160 | ailang, python |
| ❌ Exhaustive Pattern Matching | 37.5% | 234 | ailang, python |
| ❌ List Comprehension | 37.5% | 278 | ailang, python |
| ❌ Record Update | 25.0% | 205 | ailang, python |
| ❌ Numeric Modulo | 12.5% | 290 | ailang, python |
| ❌ Api Call Json | 0.0% | 317 | ailang, python |
| ❌ Cli Args | 0.0% | 308 | ailang, python |
| ❌ Float Eq | 0.0% | 365 | ailang, python |
| ❌ Json Encode | 0.0% | 265 | ailang, python |
| ❌ Json Parse | 0.0% | 326 | ailang, python |
| ❌ Pipeline | 0.0% | 248 | ailang, python |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

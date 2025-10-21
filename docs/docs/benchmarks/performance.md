---
sidebar_position: 6
title: Benchmark Performance
description: Real-world AI code generation performance metrics for AILANG
last_updated: 2025-10-21
---

import BenchmarkDashboard from '@site/src/components/BenchmarkDashboard';

# AI Code Generation Benchmarks

Real-world performance metrics for AILANG vs Python across multiple AI models.

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

- **Pattern Matching Complex**: 100.0% success rate
- **Referential Transparency**: 100.0% success rate
- **Type Safe Record Access**: 100.0% success rate
- **Simple Print**: 100.0% success rate
- **Recursion Fibonacci**: 100.0% success rate

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

**Version**: 0.3.15

**Total Runs**: 399

**Generated**: 2025-10-21 18:58:30

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |
|-------|------|--------|-------|------------|----------|----------|
| Claude Sonnet 4.5 | 68 | 52.9% | 64.7% | 4703 | $0.0143 | 0.3.15 |
| gemini-2-5-flash | 68 | 60.3% | 63.2% | 4075 | $0.0015 | 0.3.15 |
| claude-haiku-4-5 | 68 | 45.6% | 58.8% | 4435 | $0.0052 | 0.3.15 |
| gpt5 | 68 | 54.4% | 57.4% | 3826 | $0.0056 | 0.3.15 |
| Gemini 2.5 Pro | 59 | 50.8% | 55.9% | 4100 | $0.0065 | 0.3.15 |
| gpt5-mini | 68 | 52.9% | 54.4% | 3571 | $0.0011 | 0.3.15 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Pattern Matching Complex | 100.0% | 375 | ailang, python |
| ✅ Recursion Fibonacci | 100.0% | 79 | ailang, python |
| ✅ Referential Transparency | 100.0% | 127 | ailang, python |
| ✅ Simple Print | 100.0% | 47 | ailang, python |
| ✅ Type Safe Record Access | 100.0% | 176 | ailang, python |
| ⚠️ Immutable Data Structures | 91.7% | 148 | ailang, python |
| ⚠️ Print With Show | 83.3% | 42 | ailang, python |
| ⚠️ Nested Records | 81.8% | 190 | ailang, python |
| ⚠️ Fizzbuzz | 75.0% | 119 | ailang, python |
| ⚠️ Records Person | 75.0% | 123 | ailang, python |
| ⚠️ Recursion Factorial | 75.0% | 75 | ailang, python |
| ⚠️ String Manipulation | 70.0% | 103 | ailang, python |
| ⚠️ Adt Option | 66.7% | 288 | ailang, python |
| ⚠️ Effect Composition | 66.7% | 375 | ailang, python |
| ⚠️ Error Handling | 66.7% | 543 | ailang, python |
| ⚠️ Explicit State Threading | 66.7% | 257 | ailang, python |
| ⚠️ Targeted Repair Test | 60.0% | 40 | ailang |
| ⚠️ Effect Tracking Io Fs | 58.3% | 299 | ailang, python |
| ⚠️ Higher Order Functions | 58.3% | 195 | ailang, python |
| ⚠️ Canonical Normalization | 50.0% | 187 | ailang, python |
| ⚠️ Deterministic List Transform | 50.0% | 149 | ailang, python |
| ⚠️ Effect Pure Separation | 50.0% | 222 | ailang, python |
| ⚠️ Json Parse | 50.0% | 84 | ailang, python |
| ⚠️ No Runtime Crashes Option | 50.0% | 481 | ailang, python |
| ⚠️ Numeric Modulo | 50.0% | 24 | ailang, python |
| ❌ List Operations | 45.5% | 169 | ailang, python |
| ❌ Exhaustive Pattern Matching | 41.7% | 158 | ailang, python |
| ❌ Json Encode | 41.7% | 103 | ailang, python |
| ❌ Api Call Json | 33.3% | 160 | ailang, python |
| ❌ Float Eq | 33.3% | 27 | ailang, python |
| ❌ List Comprehension | 27.3% | 296 | ailang, python |
| ❌ Record Update | 25.0% | 170 | ailang, python |
| ❌ Cli Args | 0.0% | 131 | ailang, python |
| ❌ Pipeline | 0.0% | 60 | ailang, python |
| ❌ Print Missing Effect | 0.0% | 30 | ailang |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

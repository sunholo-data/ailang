---
sidebar_position: 6
title: Benchmark Performance
description: Real-world AI code generation performance metrics for AILANG
last_updated: 2025-10-10
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

- **Fizzbuzz**: 100.0% success rate
- **String Manipulation**: 100.0% success rate
- **Records Person**: 100.0% success rate
- **Recursion Fibonacci**: 90.0% success rate
- **Adt Option**: 90.0% success rate

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

**Version**: 0.3.3-11-5models

**Total Runs**: 180

**Generated**: 2025-10-10 20:40:02

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run |
|-------|------|--------|-------|------------|---------|
| gpt-5 | 36 | 63.9% | 63.9% | 2936 | $0.0881 |
| gpt-5-mini | 36 | 63.9% | 63.9% | 2599 | $0.0780 |
| Claude Sonnet 4.5 | 36 | 58.3% | 58.3% | 2383 | $0.0715 |
| gemini-2.5-pro | 36 | 55.6% | 55.6% | 2184 | $0.0655 |
| gemini-2.5-flash | 36 | 55.6% | 55.6% | 2185 | $0.0655 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Fizzbuzz | 100.0% | 275 | ailang, python |
| ✅ Records Person | 100.0% | 236 | ailang, python |
| ✅ String Manipulation | 100.0% | 269 | ailang, python |
| ⚠️ Adt Option | 90.0% | 520 | ailang, python |
| ⚠️ Nested Records | 90.0% | 281 | ailang, python |
| ⚠️ Recursion Fibonacci | 90.0% | 214 | ailang, python |
| ⚠️ Recursion Factorial | 80.0% | 168 | ailang, python |
| ⚠️ Error Handling | 70.0% | 1000 | ailang, python |
| ⚠️ List Operations | 70.0% | 546 | ailang, python |
| ⚠️ Higher Order Functions | 50.0% | 590 | ailang, python |
| ⚠️ Json Parse | 50.0% | 310 | ailang, python |
| ⚠️ Numeric Modulo | 50.0% | 156 | ailang, python |
| ❌ Pattern Matching Complex | 40.0% | 1111 | ailang, python |
| ❌ Float Eq | 30.0% | 219 | ailang, python |
| ❌ List Comprehension | 30.0% | 765 | ailang, python |
| ❌ Record Update | 30.0% | 400 | ailang, python |
| ❌ Cli Args | 0.0% | 688 | ailang, python |
| ❌ Pipeline | 0.0% | 353 | ailang, python |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

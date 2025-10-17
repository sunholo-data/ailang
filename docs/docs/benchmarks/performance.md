---
sidebar_position: 6
title: Benchmark Performance
description: Real-world AI code generation performance metrics for AILANG
last_updated: 2025-10-17
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

- **String Manipulation**: 100.0% success rate
- **Pattern Matching Complex**: 100.0% success rate
- **Nested Records**: 100.0% success rate
- **Fizzbuzz**: 100.0% success rate
- **Records Person**: 100.0% success rate

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

**Version**: v0.3.13

**Total Runs**: 180

**Generated**: 2025-10-17 15:56:44

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |
|-------|------|--------|-------|------------|----------|----------|
| gemini-2-5-flash | 3 | 100.0% | 100.0% | 6251 | $0.0021 | v0.3.13 |
| Claude Sonnet 4.5 | 42 | 64.3% | 71.4% | 2521 | $0.0090 | v0.3.13 |
| gpt5 | 42 | 66.7% | 66.7% | 2106 | $0.0037 | v0.3.13 |
| gpt5-mini | 42 | 64.3% | 64.3% | 2118 | $0.0008 | v0.3.13 |
| Gemini 2.5 Pro | 14 | 42.9% | 57.1% | 2999 | $0.0039 | v0.3.13 |
| claude-haiku-4-5 | 37 | 51.4% | 54.1% | 2523 | $0.0033 | v0.3.13 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Adt Option | 100.0% | 310 | ailang, python |
| ✅ Fizzbuzz | 100.0% | 121 | ailang, python |
| ✅ Nested Records | 100.0% | 140 | ailang, python |
| ✅ Pattern Matching Complex | 100.0% | 413 | ailang, python |
| ✅ Records Person | 100.0% | 118 | ailang, python |
| ✅ Recursion Factorial | 100.0% | 80 | ailang, python |
| ✅ Simple Print | 100.0% | 22 | python |
| ✅ String Manipulation | 100.0% | 106 | ailang, python |
| ⚠️ Record Update | 87.5% | 157 | ailang, python |
| ⚠️ Recursion Fibonacci | 83.3% | 82 | ailang, python |
| ⚠️ Targeted Repair Test | 80.0% | 46 | ailang |
| ⚠️ Error Handling | 75.0% | 458 | ailang, python |
| ⚠️ Higher Order Functions | 50.0% | 166 | ailang, python |
| ⚠️ Json Encode | 50.0% | 104 | ailang, python |
| ⚠️ Json Parse | 50.0% | 82 | ailang, python |
| ⚠️ Numeric Modulo | 50.0% | 17 | ailang, python |
| ❌ List Operations | 44.4% | 174 | ailang, python |
| ❌ Float Eq | 37.5% | 29 | ailang, python |
| ❌ List Comprehension | 25.0% | 275 | ailang, python |
| ❌ Api Call Json | 20.0% | 125 | ailang, python |
| ❌ Cli Args | 0.0% | 118 | ailang, python |
| ❌ Pipeline | 0.0% | 63 | ailang, python |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

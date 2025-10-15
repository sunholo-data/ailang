---
sidebar_position: 6
title: Benchmark Performance
description: Real-world AI code generation performance metrics for AILANG
last_updated: 2025-10-15
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

- **Simple Print**: 100.0% success rate
- **Records Person**: 100.0% success rate
- **Recursion Fibonacci**: 100.0% success rate
- **Nested Records**: 94.4% success rate
- **Fizzbuzz**: 94.1% success rate

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

**Version**: v0.3.6-21-g56b7984-dirty

**Total Runs**: 332

**Generated**: 2025-10-15 07:51:06

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |
|-------|------|--------|-------|------------|----------|----------|
| gpt5-mini | 38 | 65.8% | 65.8% | 1645 | $0.0007 | v0.3.6-cost-fix-mini |
| Gemini 2.5 Pro | 14 | 64.3% | 64.3% | 1408 | $0.0029 | v0.3.6-cost-fix |
| gpt-5-mini | 36 | 63.9% | 63.9% | 2599 | $0.0780 | v0.3.3-11-5models |
| gpt-5 | 37 | 62.2% | 62.2% | 1972 | $0.0591 | v0.3.5-8-g2e48915 |
| Claude Sonnet 4.5 | 26 | 61.5% | 61.5% | 1967 | $0.0171 | v0.3.6-cost-fix |
| Claude Sonnet 4.5 | 38 | 60.5% | 60.5% | 1922 | $0.0577 | v0.3.5-8-g2e48915 |
| gpt5 | 32 | 59.4% | 59.4% | 1555 | $0.0032 | v0.3.6-cost-fix |
| gemini-2-5-flash | 38 | 57.9% | 57.9% | 1784 | $0.0009 | v0.3.6-cost-fix-mini |
| gemini-2.5-flash | 36 | 55.6% | 55.6% | 2185 | $0.0655 | v0.3.3-11-5models |
| gemini-2.5-pro | 37 | 48.6% | 48.6% | 1823 | $0.0547 | v0.3.5-8-g2e48915 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Records Person | 100.0% | 155 | ailang, python |
| ✅ Recursion Fibonacci | 100.0% | 124 | ailang, python |
| ✅ Simple Print | 100.0% | 18 | python |
| ⚠️ Nested Records | 94.4% | 190 | ailang, python |
| ⚠️ Fizzbuzz | 94.1% | 174 | ailang, python |
| ⚠️ String Manipulation | 92.9% | 165 | ailang, python |
| ⚠️ Adt Option | 88.2% | 325 | ailang, python |
| ⚠️ Recursion Factorial | 84.2% | 107 | ailang, python |
| ⚠️ Pattern Matching Complex | 70.0% | 606 | ailang, python |
| ⚠️ Error Handling | 56.2% | 721 | ailang, python |
| ⚠️ List Operations | 55.6% | 270 | ailang, python |
| ⚠️ Higher Order Functions | 50.0% | 288 | ailang, python |
| ⚠️ Json Parse | 50.0% | 137 | ailang, python |
| ⚠️ Numeric Modulo | 50.0% | 49 | ailang, python |
| ⚠️ Targeted Repair Test | 50.0% | 43 | ailang |
| ❌ List Comprehension | 33.3% | 373 | ailang, python |
| ❌ Record Update | 31.6% | 252 | ailang, python |
| ❌ Float Eq | 29.4% | 97 | ailang, python |
| ❌ Cli Args | 0.0% | 318 | ailang, python |
| ❌ Pipeline | 0.0% | 141 | ailang, python |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

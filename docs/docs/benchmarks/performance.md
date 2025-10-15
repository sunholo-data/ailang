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

- **Fizzbuzz**: 100.0% success rate
- **Simple Print**: 100.0% success rate
- **Records Person**: 100.0% success rate
- **Recursion Factorial**: 90.0% success rate
- **Nested Records**: 90.0% success rate

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

**Version**: v0.3.7-2-g7ba8de7-dirty

**Total Runs**: 190

**Generated**: 2025-10-15 08:39:22

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |
|-------|------|--------|-------|------------|----------|----------|
| gpt5-mini | 38 | 65.8% | 65.8% | 1645 | $0.0007 | v0.3.6-24-mini |
| Claude Sonnet 4.5 | 38 | 63.2% | 63.2% | 1932 | $0.0077 | v0.3.7-1-gd24a7dc |
| gemini-2-5-flash | 38 | 57.9% | 57.9% | 1784 | $0.0009 | v0.3.6-24-mini |
| gpt5 | 38 | 57.9% | 57.9% | 1630 | $0.0031 | v0.3.7-1-gd24a7dc |
| Gemini 2.5 Pro | 38 | 55.3% | 55.3% | 1784 | $0.0036 | v0.3.7-1-gd24a7dc |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Fizzbuzz | 100.0% | 127 | ailang, python |
| ✅ Records Person | 100.0% | 114 | ailang, python |
| ✅ Simple Print | 100.0% | 19 | python |
| ⚠️ Nested Records | 90.0% | 136 | ailang, python |
| ⚠️ Recursion Factorial | 90.0% | 70 | ailang, python |
| ⚠️ Recursion Fibonacci | 90.0% | 83 | ailang, python |
| ⚠️ String Manipulation | 90.0% | 102 | ailang, python |
| ⚠️ Adt Option | 70.0% | 193 | ailang, python |
| ⚠️ Pattern Matching Complex | 60.0% | 320 | ailang, python |
| ⚠️ Record Update | 60.0% | 168 | ailang, python |
| ⚠️ Error Handling | 50.0% | 536 | ailang, python |
| ⚠️ Higher Order Functions | 50.0% | 166 | ailang, python |
| ⚠️ Json Parse | 50.0% | 82 | ailang, python |
| ⚠️ List Operations | 50.0% | 179 | ailang, python |
| ⚠️ Numeric Modulo | 50.0% | 14 | ailang, python |
| ❌ List Comprehension | 40.0% | 278 | ailang, python |
| ❌ Targeted Repair Test | 40.0% | 40 | ailang |
| ❌ Float Eq | 30.0% | 28 | ailang, python |
| ❌ Cli Args | 0.0% | 120 | ailang, python |
| ❌ Pipeline | 0.0% | 51 | ailang, python |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

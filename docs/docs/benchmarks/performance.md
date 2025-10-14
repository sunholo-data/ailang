---
sidebar_position: 6
title: Benchmark Performance
description: Real-world AI code generation performance metrics for AILANG
last_updated: 2025-10-14
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

- **Recursion Fibonacci**: 100.0% success rate
- **Targeted Repair Test**: 100.0% success rate
- **Fizzbuzz**: 100.0% success rate
- **Records Person**: 100.0% success rate
- **Pattern Matching Complex**: 100.0% success rate

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

**Version**: 0.3.5-15-g542d20f

**Total Runs**: 38

**Generated**: 2025-10-14 20:58:36

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run |
|-------|------|--------|-------|------------|---------|
| Claude Sonnet 4.5 | 38 | 68.4% | 68.4% | 1931 | $0.0579 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Fizzbuzz | 100.0% | 140 | ailang, python |
| ✅ Nested Records | 100.0% | 135 | ailang, python |
| ✅ Pattern Matching Complex | 100.0% | 298 | ailang, python |
| ✅ Record Update | 100.0% | 186 | ailang, python |
| ✅ Records Person | 100.0% | 122 | ailang, python |
| ✅ Recursion Factorial | 100.0% | 73 | ailang, python |
| ✅ Recursion Fibonacci | 100.0% | 89 | ailang, python |
| ✅ Simple Print | 100.0% | 22 | python |
| ✅ String Manipulation | 100.0% | 109 | ailang, python |
| ✅ Targeted Repair Test | 100.0% | 55 | ailang |
| ⚠️ Adt Option | 50.0% | 177 | ailang, python |
| ⚠️ Error Handling | 50.0% | 642 | ailang, python |
| ⚠️ Float Eq | 50.0% | 36 | ailang, python |
| ⚠️ Higher Order Functions | 50.0% | 170 | ailang, python |
| ⚠️ Json Parse | 50.0% | 88 | ailang, python |
| ⚠️ List Comprehension | 50.0% | 286 | ailang, python |
| ⚠️ List Operations | 50.0% | 190 | ailang, python |
| ⚠️ Numeric Modulo | 50.0% | 18 | ailang, python |
| ❌ Cli Args | 0.0% | 110 | ailang, python |
| ❌ Pipeline | 0.0% | 38 | ailang, python |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

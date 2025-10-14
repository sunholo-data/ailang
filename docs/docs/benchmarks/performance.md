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

- **Adt Option**: 100.0% success rate
- **Recursion Fibonacci**: 100.0% success rate
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

**Version**: 0.3.5-8-g2e48915

**Total Runs**: 57

**Generated**: 2025-10-14 14:15:29

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run |
|-------|------|--------|-------|------------|---------|
| gpt-5 | 19 | 42.1% | 42.1% | 3005 | $0.0901 |
| Claude Sonnet 4.5 | 19 | 36.8% | 36.8% | 3563 | $0.1069 |
| gemini-2.5-pro | 19 | 26.3% | 26.3% | 3261 | $0.0978 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Adt Option | 100.0% | 197 | ailang |
| ✅ Records Person | 100.0% | 127 | ailang |
| ✅ Recursion Fibonacci | 100.0% | 91 | ailang |
| ⚠️ Fizzbuzz | 66.7% | 169 | ailang |
| ⚠️ Nested Records | 66.7% | 120 | ailang |
| ⚠️ Recursion Factorial | 66.7% | 85 | ailang |
| ⚠️ String Manipulation | 66.7% | 121 | ailang |
| ⚠️ Targeted Repair Test | 66.7% | 47 | ailang |
| ❌ Error Handling | 33.3% | 450 | ailang |
| ❌ Cli Args | 0.0% | 114 | ailang |
| ❌ Float Eq | 0.0% | 36 | ailang |
| ❌ Higher Order Functions | 0.0% | 219 | ailang |
| ❌ Json Parse | 0.0% | 83 | ailang |
| ❌ List Comprehension | 0.0% | 460 | ailang |
| ❌ List Operations | 0.0% | 215 | ailang |
| ❌ Numeric Modulo | 0.0% | 14 | ailang |
| ❌ Pattern Matching Complex | 0.0% | 337 | ailang |
| ❌ Pipeline | 0.0% | 57 | ailang |
| ❌ Record Update | 0.0% | 176 | ailang |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

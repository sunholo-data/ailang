---
sidebar_position: 6
title: Benchmark Performance
description: Real-world AI code generation performance metrics for AILANG
last_updated: 2025-10-16
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

- **Recursion Factorial**: 100.0% success rate
- **String Manipulation**: 100.0% success rate
- **Records Person**: 100.0% success rate
- **Recursion Fibonacci**: 100.0% success rate
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

**Version**: v0.3.9

**Total Runs**: 126

**Generated**: 2025-10-16 01:00:40

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |
|-------|------|--------|-------|------------|----------|----------|
| gpt5-mini | 42 | 69.0% | 69.0% | 1978 | $0.0007 | v0.3.9 |
| gemini-2-5-flash | 42 | 54.8% | 54.8% | 2130 | $0.0010 | v0.3.9 |
| claude-haiku-4-5 | 42 | 52.4% | 52.4% | 2362 | $0.0032 | v0.3.9 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Pattern Matching Complex | 100.0% | 334 | ailang, python |
| ✅ Records Person | 100.0% | 122 | ailang, python |
| ✅ Recursion Factorial | 100.0% | 84 | ailang, python |
| ✅ Recursion Fibonacci | 100.0% | 91 | ailang, python |
| ✅ Simple Print | 100.0% | 20 | python |
| ✅ String Manipulation | 100.0% | 103 | ailang, python |
| ⚠️ Adt Option | 83.3% | 269 | ailang, python |
| ⚠️ Fizzbuzz | 83.3% | 116 | ailang, python |
| ⚠️ Nested Records | 83.3% | 242 | ailang, python |
| ⚠️ Record Update | 66.7% | 159 | ailang, python |
| ⚠️ Targeted Repair Test | 66.7% | 48 | ailang |
| ⚠️ Higher Order Functions | 50.0% | 170 | ailang, python |
| ⚠️ Json Parse | 50.0% | 80 | ailang, python |
| ⚠️ List Operations | 50.0% | 182 | ailang, python |
| ⚠️ Numeric Modulo | 50.0% | 82 | ailang, python |
| ❌ Error Handling | 33.3% | 541 | ailang, python |
| ❌ Float Eq | 33.3% | 22 | ailang, python |
| ❌ Json Encode | 33.3% | 132 | ailang, python |
| ❌ Api Call Json | 16.7% | 106 | ailang, python |
| ❌ List Comprehension | 16.7% | 279 | ailang, python |
| ❌ Cli Args | 0.0% | 234 | ailang, python |
| ❌ Pipeline | 0.0% | 61 | ailang, python |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

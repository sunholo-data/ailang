Loading results from eval_results/baselines/0.3.14...
Generating performance matrix...
Generating docusaurus report...

---
sidebar_position: 6
title: Benchmark Performance
description: Real-world AI code generation performance metrics for AILANG
last_updated: 2025-10-18
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
- **String Manipulation**: 100.0% success rate
- **Fizzbuzz**: 100.0% success rate
- **Records Person**: 100.0% success rate
- **Nested Records**: 100.0% success rate

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

**Version**: 0.3.14

**Total Runs**: 227

**Generated**: 2025-10-18 22:42:40

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |
|-------|------|--------|-------|------------|----------|----------|
| Claude Sonnet 4.5 | 42 | 64.3% | 71.4% | 2523 | $0.0090 | 0.3.14 |
| gpt5-mini | 31 | 71.0% | 71.0% | 2396 | $0.0008 | 0.3.14 |
| gpt5 | 42 | 59.5% | 61.9% | 2267 | $0.0037 | 0.3.14 |
| Gemini 2.5 Pro | 39 | 61.5% | 61.5% | 2150 | $0.0041 | 0.3.14 |
| claude-haiku-4-5 | 42 | 50.0% | 59.5% | 2633 | $0.0033 | 0.3.14 |
| gemini-2-5-flash | 31 | 51.6% | 58.1% | 2676 | $0.0010 | 0.3.14 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Fizzbuzz | 100.0% | 129 | ailang, python |
| ✅ Nested Records | 100.0% | 211 | ailang, python |
| ✅ Records Person | 100.0% | 120 | ailang, python |
| ✅ Simple Print | 100.0% | 20 | python |
| ✅ String Manipulation | 100.0% | 101 | ailang, python |
| ⚠️ Adt Option | 91.7% | 273 | ailang, python |
| ⚠️ Pattern Matching Complex | 91.7% | 378 | ailang, python |
| ⚠️ Recursion Fibonacci | 91.7% | 84 | ailang, python |
| ⚠️ Recursion Factorial | 83.3% | 80 | ailang, python |
| ⚠️ Error Handling | 80.0% | 552 | ailang, python |
| ⚠️ Targeted Repair Test | 80.0% | 53 | ailang |
| ⚠️ Record Update | 58.3% | 160 | ailang, python |
| ⚠️ Higher Order Functions | 55.6% | 171 | ailang, python |
| ⚠️ Json Encode | 50.0% | 95 | ailang, python |
| ⚠️ Json Parse | 50.0% | 88 | ailang, python |
| ⚠️ List Operations | 50.0% | 243 | ailang, python |
| ❌ Numeric Modulo | 45.5% | 30 | ailang, python |
| ❌ Api Call Json | 33.3% | 131 | ailang, python |
| ❌ List Comprehension | 33.3% | 296 | ailang, python |
| ❌ Float Eq | 18.2% | 23 | ailang, python |
| ❌ Cli Args | 0.0% | 128 | ailang, python |
| ❌ Pipeline | 0.0% | 52 | ailang, python |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

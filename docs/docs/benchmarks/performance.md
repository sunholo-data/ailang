---
sidebar_position: 6
title: 🚀 Benchmark Performance
description: Real-world AI code generation performance metrics for AILANG
last_updated: 2025-10-10
---

import BenchmarkDashboard from '@site/src/components/BenchmarkDashboard';

# 🚀 AI Code Generation Benchmarks

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

## 🎯 Where AILANG Shines

AILANG excels at these problem types:

- **Records Person**: 100.0% success rate
- **Adt Option**: 100.0% success rate
- **Fizzbuzz**: 100.0% success rate
- **Recursion Fibonacci**: 100.0% success rate

## 🛠️ How Benchmarks Guide Development

The M-EVAL-LOOP system uses these benchmarks to:

1. **Identify Bugs**: Failing benchmarks reveal language issues
2. **Validate Fixes**: Compare before/after to confirm improvements
3. **Track Progress**: Historical data shows language evolution
4. **Prioritize Features**: High-impact failures guide roadmap

### Case Study: Float Equality Bug

The `adt_option` benchmark caught a critical bug where float comparisons with variables called `eq_Int` instead of `eq_Float`. The benchmark suite detected it, guided the fix, and validated the solution.

**Result**: Benchmark went from runtime_error → PASSING ✅

## 🎮 Try It Yourself

Want to see AILANG in action?

- **[Interactive REPL](/ailang/docs/reference/repl-commands)** - Try AILANG in your browser
- **[Code Examples](https://github.com/sunholo-data/ailang/tree/main/examples)** - 48+ working examples
- **[Getting Started](/ailang/docs/guides/getting-started)** - Install and run locally

## 📊 Technical Details

**Version**: 0.3.2-19-g4f42cf4

**Total Runs**: 10

**Generated**: 2025-10-10 17:53:42

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run |
|-------|------|--------|-------|------------|---------|
| Claude Sonnet 4.5 | 10 | 40.0% | 40.0% | 3076 | $0.0923 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Adt Option | 100.0% | 6100 | ailang |
| ✅ Fizzbuzz | 100.0% | 6042 | ailang |
| ✅ Records Person | 100.0% | 6009 | ailang |
| ✅ Recursion Fibonacci | 100.0% | 5986 | ailang |
| ❌ Cli Args | 0.0% | 181 | ailang |
| ❌ Float Eq | 0.0% | 105 | ailang |
| ❌ Json Parse | 0.0% | 206 | ailang |
| ❌ Numeric Modulo | 0.0% | 62 | ailang |
| ❌ Pipeline | 0.0% | 145 | ailang |
| ❌ Recursion Factorial | 0.0% | 5921 | ailang |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

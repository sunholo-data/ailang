---
sidebar_position: 6
title: Benchmark Performance
description: Real-world AI code generation performance metrics for AILANG
last_updated: 2025-10-27
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

- **Referential Transparency**: 100.0% success rate
- **Type Safe Record Access**: 100.0% success rate
- **Recursion Fibonacci**: 100.0% success rate
- **Simple Print**: 100.0% success rate
- **Explicit State Threading**: 91.7% success rate

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

**Version**: v0.3.21

**Total Runs**: 376

**Generated**: 2025-10-27 13:31:30

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |
|-------|------|--------|-------|------------|----------|----------|
| gpt5 | 68 | 60.3% | 61.8% | 4322 | $0.0065 | v0.3.21 |
| gpt5-mini | 57 | 57.9% | 57.9% | 5122 | $0.0015 | v0.3.21 |
| claude-haiku-4-5 | 58 | 50.0% | 56.9% | 6045 | $0.0069 | v0.3.21 |
| gemini-2-5-flash | 57 | 52.6% | 56.1% | 5878 | $0.0021 | v0.3.21 |
| Claude Sonnet 4.5 | 68 | 55.9% | 55.9% | 5045 | $0.0170 | v0.3.21 |
| Gemini 2.5 Pro | 68 | 52.9% | 52.9% | 4784 | $0.0075 | v0.3.21 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Recursion Fibonacci | 100.0% | 85 | ailang, python |
| ✅ Referential Transparency | 100.0% | 136 | ailang, python |
| ✅ Simple Print | 100.0% | 27 | ailang, python |
| ✅ Type Safe Record Access | 100.0% | 121 | ailang, python |
| ⚠️ Explicit State Threading | 91.7% | 217 | ailang, python |
| ⚠️ Print With Show | 88.9% | 36 | ailang, python |
| ⚠️ Records Person | 88.9% | 128 | ailang, python |
| ⚠️ Recursion Factorial | 88.9% | 73 | ailang, python |
| ⚠️ Pattern Matching Complex | 83.3% | 325 | ailang, python |
| ⚠️ Immutable Data Structures | 81.8% | 162 | ailang, python |
| ⚠️ String Manipulation | 77.8% | 108 | ailang, python |
| ⚠️ Higher Order Functions | 75.0% | 178 | ailang, python |
| ⚠️ Fizzbuzz | 70.0% | 135 | ailang, python |
| ⚠️ Error Handling | 66.7% | 445 | ailang, python |
| ⚠️ Nested Records | 66.7% | 129 | ailang, python |
| ⚠️ Adt Option | 58.3% | 267 | ailang, python |
| ⚠️ Effect Composition | 58.3% | 437 | ailang, python |
| ⚠️ Canonical Normalization | 50.0% | 203 | ailang, python |
| ⚠️ Effect Pure Separation | 50.0% | 227 | ailang, python |
| ⚠️ Effect Tracking Io Fs | 50.0% | 254 | ailang, python |
| ⚠️ List Operations | 50.0% | 172 | ailang, python |
| ⚠️ Numeric Modulo | 50.0% | 28 | ailang, python |
| ⚠️ Targeted Repair Test | 50.0% | 42 | ailang |
| ❌ Json Encode | 45.5% | 96 | ailang, python |
| ❌ Json Parse | 45.5% | 101 | ailang, python |
| ❌ Deterministic List Transform | 41.7% | 142 | ailang, python |
| ❌ Exhaustive Pattern Matching | 41.7% | 167 | ailang, python |
| ❌ Float Eq | 41.7% | 28 | ailang, python |
| ❌ No Runtime Crashes Option | 41.7% | 486 | ailang, python |
| ❌ List Comprehension | 33.3% | 312 | ailang, python |
| ❌ Api Call Json | 16.7% | 118 | ailang, python |
| ❌ Print Missing Effect | 16.7% | 30 | ailang |
| ❌ Record Update | 11.1% | 159 | ailang, python |
| ❌ Cli Args | 0.0% | 154 | ailang, python |
| ❌ Pipeline | 0.0% | 58 | ailang, python |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

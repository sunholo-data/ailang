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

- **Print With Show**: 100.0% success rate
- **Higher Order Functions**: 100.0% success rate
- **Recursion Fibonacci**: 100.0% success rate
- **Simple Print**: 100.0% success rate
- **Type Safe Record Access**: 100.0% success rate

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

**Total Runs**: 210

**Generated**: 2025-10-27 16:53:02

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |
|-------|------|--------|-------|------------|----------|----------|
| claude-haiku-4-5 | 35 | 45.7% | 48.6% | 12123 | $0.0125 | v0.3.21 |
| gpt5-mini | 35 | 37.1% | 42.9% | 10725 | $0.0029 | v0.3.21 |
| gpt5 | 35 | 37.1% | 42.9% | 10791 | $0.0147 | v0.3.21 |
| gemini-2-5-flash | 35 | 37.1% | 40.0% | 11484 | $0.0038 | v0.3.21 |
| Gemini 2.5 Pro | 35 | 37.1% | 37.1% | 11330 | $0.0177 | v0.3.21 |
| Claude Sonnet 4.5 | 35 | 28.6% | 34.3% | 12528 | $0.0379 | v0.3.21 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Higher Order Functions | 100.0% | 317 | ailang |
| ✅ Print With Show | 100.0% | 44 | ailang |
| ✅ Recursion Fibonacci | 100.0% | 92 | ailang |
| ✅ Referential Transparency | 100.0% | 131 | ailang |
| ✅ Simple Print | 100.0% | 32 | ailang |
| ✅ Type Safe Record Access | 100.0% | 109 | ailang |
| ⚠️ Adt Option | 83.3% | 192 | ailang |
| ⚠️ Explicit State Threading | 83.3% | 269 | ailang |
| ⚠️ Immutable Data Structures | 83.3% | 162 | ailang |
| ⚠️ Records Person | 83.3% | 138 | ailang |
| ⚠️ Recursion Factorial | 83.3% | 80 | ailang |
| ⚠️ Targeted Repair Test | 83.3% | 51 | ailang |
| ⚠️ Nested Records | 66.7% | 126 | ailang |
| ⚠️ Pattern Matching Complex | 66.7% | 299 | ailang |
| ⚠️ String Manipulation | 66.7% | 122 | ailang |
| ⚠️ Fizzbuzz | 50.0% | 162 | ailang |
| ❌ Error Handling | 33.3% | 389 | ailang |
| ❌ Numeric Modulo | 33.3% | 517 | ailang |
| ❌ Effect Composition | 16.7% | 381 | ailang |
| ❌ Api Call Json | 0.0% | 280 | ailang |
| ❌ Canonical Normalization | 0.0% | 244 | ailang |
| ❌ Cli Args | 0.0% | 388 | ailang |
| ❌ Deterministic List Transform | 0.0% | 232 | ailang |
| ❌ Effect Pure Separation | 0.0% | 169 | ailang |
| ❌ Effect Tracking Io Fs | 0.0% | 199 | ailang |
| ❌ Exhaustive Pattern Matching | 0.0% | 145 | ailang |
| ❌ Float Eq | 0.0% | 1185 | ailang |
| ❌ Json Encode | 0.0% | 328 | ailang |
| ❌ Json Parse | 0.0% | 326 | ailang |
| ❌ List Comprehension | 0.0% | 408 | ailang |
| ❌ List Operations | 0.0% | 188 | ailang |
| ❌ No Runtime Crashes Option | 0.0% | 392 | ailang |
| ❌ Pipeline | 0.0% | 443 | ailang |
| ❌ Print Missing Effect | 0.0% | 30 | ailang |
| ❌ Record Update | 0.0% | 154 | ailang |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

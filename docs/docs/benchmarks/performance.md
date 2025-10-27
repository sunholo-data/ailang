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
- **Higher Order Functions**: 100.0% success rate
- **Print With Show**: 100.0% success rate
- **Simple Print**: 100.0% success rate

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

**Version**: v0.3.22

**Total Runs**: 408

**Generated**: 2025-10-27 18:16:13

### Model Performance Details

| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |
|-------|------|--------|-------|------------|----------|----------|
| claude-haiku-4-5 | 68 | 51.5% | 63.2% | 5453 | $0.0061 | v0.3.22 |
| Claude Sonnet 4.5 | 68 | 54.4% | 61.8% | 5456 | $0.0170 | v0.3.22 |
| gemini-2-5-flash | 68 | 60.3% | 61.8% | 4968 | $0.0018 | v0.3.22 |
| Gemini 2.5 Pro | 68 | 58.8% | 61.8% | 4816 | $0.0075 | v0.3.22 |
| gpt5 | 68 | 58.8% | 60.3% | 4488 | $0.0065 | v0.3.22 |
| gpt5-mini | 68 | 58.8% | 58.8% | 4339 | $0.0013 | v0.3.22 |

### Benchmark Details

| Benchmark | Success Rate | Avg Tokens | Languages |
|-----------|--------------|------------|-----------|
| ✅ Higher Order Functions | 100.0% | 192 | ailang, python |
| ✅ Print With Show | 100.0% | 35 | ailang, python |
| ✅ Referential Transparency | 100.0% | 122 | ailang, python |
| ✅ Simple Print | 100.0% | 41 | ailang, python |
| ✅ Type Safe Record Access | 100.0% | 131 | ailang, python |
| ⚠️ Explicit State Threading | 91.7% | 230 | ailang, python |
| ⚠️ Immutable Data Structures | 91.7% | 199 | ailang, python |
| ⚠️ Recursion Fibonacci | 91.7% | 80 | ailang, python |
| ⚠️ Adt Option | 83.3% | 342 | ailang, python |
| ⚠️ Fizzbuzz | 83.3% | 121 | ailang, python |
| ⚠️ Nested Records | 83.3% | 198 | ailang, python |
| ⚠️ Pattern Matching Complex | 83.3% | 338 | ailang, python |
| ⚠️ Records Person | 83.3% | 121 | ailang, python |
| ⚠️ Recursion Factorial | 83.3% | 72 | ailang, python |
| ⚠️ String Manipulation | 75.0% | 102 | ailang, python |
| ⚠️ Effect Composition | 66.7% | 306 | ailang, python |
| ⚠️ Error Handling | 66.7% | 514 | ailang, python |
| ⚠️ Canonical Normalization | 50.0% | 201 | ailang, python |
| ⚠️ Deterministic List Transform | 50.0% | 170 | ailang, python |
| ⚠️ Effect Pure Separation | 50.0% | 210 | ailang, python |
| ⚠️ Effect Tracking Io Fs | 50.0% | 347 | ailang, python |
| ⚠️ Json Parse | 50.0% | 83 | ailang, python |
| ⚠️ List Operations | 50.0% | 178 | ailang, python |
| ⚠️ Numeric Modulo | 50.0% | 29 | ailang, python |
| ⚠️ Targeted Repair Test | 50.0% | 41 | ailang |
| ❌ Json Encode | 41.7% | 127 | ailang, python |
| ❌ Api Call Json | 33.3% | 136 | ailang, python |
| ❌ Exhaustive Pattern Matching | 33.3% | 203 | ailang, python |
| ❌ Float Eq | 33.3% | 27 | ailang, python |
| ❌ List Comprehension | 33.3% | 291 | ailang, python |
| ❌ No Runtime Crashes Option | 33.3% | 477 | ailang, python |
| ❌ Record Update | 16.7% | 160 | ailang, python |
| ❌ Cli Args | 0.0% | 127 | ailang, python |
| ❌ Pipeline | 0.0% | 86 | ailang, python |
| ❌ Print Missing Effect | 0.0% | 30 | ailang |

---

**Methodology**: Benchmarks use deterministic seeds across multiple AI models. Each benchmark tests code generation, compilation, and execution. The M-EVAL-LOOP system provides structured error feedback for automatic repair.

**Learn More**: [M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | [Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)

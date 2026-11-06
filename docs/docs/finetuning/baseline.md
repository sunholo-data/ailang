---
sidebar_position: 3
title: Baseline (Phase 0)
description: Size ladder of open-weight models on AILANG, agent-mode failure analysis, rig speed and parallel-decoding measurements.
reviewed: 2026-11-06
reviewBy: 2027-02-06
---

# Baseline (Phase 0)

Measured on 2026-11-05 at AILANG v0.57.1, before any training. The question: which model size
already knows enough AILANG to be worth fine-tuning, and what does it get wrong?

## Standard mode: one-shot with the full teaching prompt

Core (23) + stretch (28) benchmarks, three trials, self-repair on. Local rungs ran MXFP8 on
Ollama's MLX runner; cloud rungs ran on OpenRouter (open-weight models only).

| Model | Where | Pass | Core | Stretch |
|---|---|---|---|---|
| qwen3.8-27b (reference) | cloud | **94%** | 100% | 89% |
| gemma-4-31b | cloud | 58% | 81% | 39% |
| **gemma4-12b** | local | 48% (**54%** excluding 7 harness timeouts) | 65% | 31% |
| gemma-4-26b-a4b (MoE) | cloud | 46% | 61% | 35% |
| ministral-14b | cloud | 33% | 38% | 30% |
| ministral-8b | cloud | 30% | 45% | 18% |
| qwen3.5-9b | cloud | 21% | 30% | 13% |
| gemma4-e4b | local | 13% | 14% | 11% |

**The gap is language knowledge.** For every model below 27B, 71–89% of failures are compile
errors. That is the failure fine-tuning addresses.

## Agent mode: the pi harness, 24 benchmarks

| Model | Pass | Failures |
|---|---|---|
| gemma4-e4b | 0/24 | 13 never called a tool; 11 runtime errors. Ruled out |
| **gemma4-12b** | **11/24**, all on the first attempt | 6 never called a tool; 6 left the template untouched and answered in chat; **1** genuine compile error |

**12 of 12B's 13 agent failures are tool-protocol misses.** When it wrote a solution file, the
solution was right 11 times out of 12. "Finish by writing the file and running it" is exactly
what trajectory fine-tuning teaches.

## Speed on the rig (Mac Studio M4 Max, 128 GB, ~546 GB/s)

- Decode is memory-bandwidth bound: E4B ≈ 65 tok/s, 12B ≈ 46 tok/s.
- Ollama's prefix cache works: a cached 33k-token prompt costs ~1 s; a cold one costs ~70 s on 12B.
- Per agent pass, 12B is ~1.6× faster than qwen3.8-27b, but it thinks ~1.6× longer. At its
  pass rate it is slower **per verified success** (~20 min against ~12 min). So the fine-tune is
  measured on **output tokens per pass** and **wall time per verified success**, not pass rate
  alone.

## Parallel decoding (mlx-lm batching, 12B 4-bit)

| Context | Streams | Total tok/s | Gain |
|---|---|---|---|
| short | 1 | 58.7 | 1.0× |
| short | 4 | 162.6 | 2.77× |
| short | 8 | 177.7 | 3.03× |
| 33k (the agent prompt) | 1 | 51.3 | 1.0× |
| 33k | 4 | 64.4 | **1.26×** |

At a real agent context, batching collapses: attention over the long prompt dominates each step.
**Dropping the teaching prompt is what makes parallel agents pay off.** That is a second,
independent reason to fine-tune.

## Tooling check

Stock mlx-lm 0.32.2 trains Gemma 4 12B LoRA at ~150 tok/s with a 13.4 GB peak (40 steps in
177 s); 31B needs 25.7 GB. The fused model serves through `ollama create -q mxfp8` once the
fused config's audio section is dropped (see [Pilot-1](./pilot-1.md) for the traps).

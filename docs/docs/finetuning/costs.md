---
sidebar_position: 7
title: Serving costs
description: What it costs to serve a fine-tuned 12B AILANG model on the rig, Cloud Run GPUs, Vertex AI or a per-token multi-LoRA host.
reviewed: 2026-11-06
reviewBy: 2027-02-06
---

# Serving costs

List prices found on 2026-11-05, in USD. Verify them before budgeting. Per-run figures use the
measured 12B agent profile: about 0.7–2M input tokens per benchmark (mostly a cacheable prefix)
and about 10–35k output tokens.

The agent loop itself needs only CPU. Only model inference needs a GPU.

| Option | Billing | Rate | ≈ per agent run | Serves our fine-tune? |
|---|---|---|---|---|
| **Rig (Mac Studio M4 Max)** | electricity; hardware already owned | ~100 W under GPU load | **≈ €0.005** | yes; private; one box (~4–8 batched streams) |
| **Cloud Run + NVIDIA L4** | per second while up; scales to zero | ≈ $0.67/h GPU + 4 vCPU / 16 GB ≈ **$1/h** | **≈ $0.04** batched ×4; $0.15–0.30 single stream | yes (vLLM + adapter); best overflow option |
| **Vertex AI dedicated endpoint** (g2-standard-4, 1× L4) | per node-hour, including idle | $0.81/h ≈ **$585/month** always on | ≈ $0.04 when busy | yes; pays only at steady load |
| **Vertex managed Gemma 4** | per token | 26B: $0.15/M in, $0.60/M out, $0.015/M cached | ≈ $0.03–0.05 if the cache hits | **no**: no 12B, no custom LoRA |
| **Fireworks serverless multi-LoRA** | per token, adapter on a shared base | 4–16B tier ≈ $0.20/M (third-party figure) | ≈ $0.20 | maybe; Gemma 4 12B support unconfirmed |

Notes:
- **An L4 has about half the rig's memory bandwidth** (~300 GB/s against ~546 GB/s), so a single
  stream is slower in the cloud. Batching is what makes Cloud Run pay.
- **About 98% of our tokens are input**, so prompt-cache pricing dominates every per-token
  option. We have measured provider billing at ~4× the traced cost on cached runs, so test any
  per-token option with a real bill first.
- **Recommended shape: rig first, cloud overflow.** Routine AILANG work runs batched on the rig at
  about $0; the same adapter runs on a scale-to-zero Cloud Run L4 service for bursts.

Sources: [Cloud Run pricing](https://cloud.google.com/run/pricing) ·
[Cloud Run GPUs](https://docs.cloud.google.com/run/docs/configuring/services/gpu) ·
[Vertex AI pricing](https://cloud.google.com/vertex-ai/pricing) ·
[Model Garden pricing](https://cloud.google.com/model-garden/pricing)

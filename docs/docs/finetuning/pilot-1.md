---
sidebar_position: 5
title: Pilot-1 (Gemma 4 12B LoRA)
description: The first AILANG fine-tune — recipe, training curve, the traps we hit, and the held-out A/B against the base model.
reviewed: 2026-11-06
reviewBy: 2027-02-06
---

# Pilot-1: Gemma 4 12B LoRA

Weekend run on the rig, started 2026-11-05 19:42Z, with the GPU reserved.

## Recipe

| Setting | Value |
|---|---|
| Base | `mlx-community/gemma-4-12B-it-4bit` |
| Trainer | mlx-lm 0.32.2 `lora`, through a wrapper that fixes prompt masking (below) |
| Adapter | LoRA rank 16, scale 20, dropout 0.05, all linear layers in the top 16 blocks (21.9M trainable parameters, 0.18%) |
| Schedule | 1 epoch, batch 1 × 4 gradient accumulation, learning rate 1e-4 cosine with 25 warmup steps, max sequence 8,192 |
| Data | the [pilot-1 mix](./data.md#pilot-1-mix): 2,183 samples / 4.79M tokens |
| Serving | `mlx_lm.fuse --dequantize`, then `ollama create -q mxfp8` (12 GB) |

## Training curve

Training took 6 h 25 min (2,183 steps at ~5.7 steps a minute). Validation loss fell at every
checkpoint, with shrinking steps and no sign of overfitting:

| Step | 0 | 250 | 500 | 750 | 1000 | 1250 | 1750 | 2000 | 2183 |
|---|---|---|---|---|---|---|---|---|---|
| Val loss | 1.971 | 1.025 | 0.897 | 0.851 | 0.829 | 0.807 | 0.782 | 0.777 | **0.775** |

## Traps (each cost a debugging round)

- **Gemma 4's chat template adds an empty thinking block to the generation prompt, but not to a
  finished turn.** So stock `--mask-prompt` hid the first 4 target tokens on 1,497 of 2,193
  samples, including the tool-call token. Our wrapper trains on exactly the inference-time prompt
  plus the target; a pre-training gate checks every sample.
- **`--mask-prompt` trains only the last message.** Trajectories must therefore be split into one
  sample per target turn.
- **The fused model failed to load in Ollama at first.** Fusing drops the audio and vision
  weights but keeps their config, and the runner refuses a declared audio embedder with no
  weights. Dropping `audio_config` from the fused config fixes it. `q8_0` is not supported on
  this path; `mxfp8` is.
- **The imported fine-tune has no speculative-decoding draft model**, so it decodes at ~34 tok/s
  against the library model's ~46. Compare speed on output tokens per pass, not wall time.
- **The imported fine-tune has no sampling defaults.** The base ships temperature 1, top_k 64,
  top_p 0.95; set them at import.
- **Agent mode ignored `--prompt-version`**, and the prompt loader silently falls back to the full
  prompt on a bad version. Both fixed; every arm now checks the prompt version it banked.

## Held-out A/B: result

Pilot-1 is a **negative result with known causes**. On the 24 held-out benchmarks:

| Arm | Model | Prompt | Mode | Pass |
|---|---|---|---|---|
| A | base 12B | full teaching prompt | standard (one-shot) | 16/31 (52%; core 15/18, stretch 1/13) |
| B | pilot-1 | 2.4 KB slim prompt | standard | 0/15 |
| C | pilot-1 | none | standard | 1/26 |
| **A** | **base 12B** | **full prompt** | **agent (pi)** | **17/24 (71%; core 12/12, stretch 5/12)** |
| B | pilot-1 | slim prompt | agent | 0/24 |
| C | pilot-1 | none | agent | 0/24 |

**The kill criterion (B ≥ A) fails in both modes.** The standard arms ran under 2.5-hour time
caps; B and C hit theirs after 15 and 26 runs because of repetition loops.

**Why the fine-tune failed**, in the order we found the causes:

1. **The first agent arms were void.** The pi session-protocol gate, a guard for attended dev
   sessions, refused every eval `write`, `edit` and mutating `bash` until the model called
   `session_protocol_ack`.
   - Base 12B had learned to ack: every one of its earlier passes did.
   - The fine-tune never acked.
   - Fixed in `5dc3ba42b` ([D-FT-6](./decisions.md)). Every agent arm, base included, was re-run
     with the fix; the table shows the re-runs.
2. **Thinking-format mismatch.** Pilot-1 was trained in thinking-off format, but Ollama serves it
   thinking-on. A live transcript showed its sensible tool calls (`ailang run …`,
   `ailang_check …`) landing in the reasoning channel, so pi saw no tool call and ended the run.
   This alone explains the 0/24 ([D-FT-8](./decisions.md)).
3. **Context mismatch.** It trained on a one-line system prompt and was served under pi's
   7,338-token default prefix ([D-FT-9](./decisions.md)).
4. **Standard mode, which does not depend on tool-call format:** one light epoch on 16 blocks did
   not teach enough syntax to replace the teaching prompt.

**The fair base-12B agent baseline is 17/24 (71%).** That is far above Phase 0's 11/24, which ran
with the gate armed: the gate had been suppressing the base model too.

Pilot-2 addresses each cause. It trains in the served format (thinking on, the lean pi profile,
whole episodes), on habit-simulated episodes with real tool output, across all transformer blocks.

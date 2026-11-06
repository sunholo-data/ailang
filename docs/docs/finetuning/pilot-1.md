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

## Held-out A/B (in progress)

Arms on the 24 held-out benchmarks:

| Arm | Model | Prompt | Standard (one-shot) | Agent (pi) |
|---|---|---|---|---|
| A | base 12B | full teaching prompt | 48% (Phase 0); re-run pending | 11/24 (Phase 0) |
| B | pilot-1 | 2.4 KB slim prompt | **0/15** | running |
| C | pilot-1 | none | **1/26** | pending |

**The first agent arms are void.** Both B_agent and C_agent scored 0/24, but every run stopped
after about two tool calls with the template untouched. Capturing pi's wire traffic against a
scripted model showed why:
- The pi session-protocol gate, a guard for attended dev sessions, was armed in every eval. It
  refused each `write`, `edit` and mutating `bash` until the model called `session_protocol_ack`.
- Base 12B had learned to ack; every one of its Phase 0 passes did. The fine-tune never acked, so
  every write was refused.
- Fixed in `5dc3ba42b`: evals now disarm the gate ([D-FT-6](./decisions.md)). All three agent arms,
  base included, re-run under the same condition.

**The re-run with the gate fixed: B_agent2 0/24, and no run called the ack.** A live transcript,
taken through the cooperative rig handoff, showed the second cause. After a tool result, the
fine-tune's sensible tool calls (`ailang run …`, `ailang_check …`) came back inside the
*thinking* channel, so pi saw no tool call and ended the run.
- Pilot-1 was trained in thinking-off format: the generation prompt pre-fills an empty, closed
  thought.
- Ollama serves Gemma 4 with thinking on, and its parser then files the output as reasoning.
- Replaying one turn with `think: false` did not fix it reliably, so the mismatch is in the
  training format, not a serving switch.
- Pilot-2 trains every step as thought → action in the served format
  ([D-FT-8](./decisions.md)).

The capture also showed two kinds of noise in every eval tool result: an OpenTelemetry
deprecation warning on each `ailang` command, and an injected "brain context" block on each
`read`.

**Standard mode came out clearly worse than the base model with its full prompt.**
- Both arms hit their 2.5-hour time caps. Six of the 15 B runs used the full 32k output tokens on
  repetition loops.
- The runs that finished normally failed on invented stdlib functions, wrong module names and
  statement-style `x = …` assignments.

**What we take from it** ([D-FT-3](./decisions.md)):
- A light single-epoch LoRA on mostly-trajectory data does not teach enough syntax to replace
  the teaching prompt in one-shot mode.
- The agent arms are the real test, because they match what the data mostly contained.

Pilot-2 changes the data and the recipe in response: habit-weighted trajectories, canonical
examples, staleness pairs, 2 epochs, LoRA on all layers, and sampling defaults set at import. It
also adds a base + slim prompt arm, so that "the fine-tune didn't help" can be told apart from
"the slim prompt is too thin".

*This page is updated as the arms finish.*

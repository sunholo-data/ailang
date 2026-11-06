---
sidebar_position: 1
title: Fine-tuning AILANG models
description: Research log for training small, local AILANG-expert models — what we measured, what we decided, and what we publish.
reviewed: 2026-11-06
reviewBy: 2027-02-06
---

# Fine-tuning AILANG models

This section is the open research log for a question we started on 2026-11-05: **can a small
open-weight model, fine-tuned on AILANG, do useful agent work locally at $0, without the
26k-token teaching prompt?** We record measurements, decisions and dead ends as they happen,
including the ones that go against us.

## Why

- **The rig is saturated.** A 27B model at a 131K context occupies ~49 GB, and Ollama serves one
  model at a time. Smaller models free memory for parallel lanes.
- **The teaching prompt is the bulk of every agent context.** At ~33k tokens, parallel decoding on
  the rig gains only ~1.26× from batching. At short contexts it gains ~2.8×. A model that does
  not need the prompt makes parallel agents pay off. See [Baseline](./baseline.md).
- **AILANG was designed for AI self-discovery.** Its compiler errors name the fix, and `ailang docs`,
  `ailang examples` and `ailang iface` expose the language at runtime. A fine-tune that learns to
  *use* those tools should stay useful as the language moves.

## The approach in one paragraph

Fine-tune for **habits, not memorisation**. The target model should:
- write its solution through the tool and never answer in chat;
- check every write and apply the fix the compiler names;
- look up a stdlib name before guessing it;
- trust the compiler over its own memory;
- stop once the program is verified.

Only the core syntax is memorised, from canonical examples. Everything else is discovered at run
time. That should make a fine-tune decay slowly as AILANG changes, and we plan to measure that
decay. See [Training data](./data.md).

## Status

| Stage | State |
|---|---|
| [Baseline](./baseline.md) (size ladder, speed, parallelism) | done 2026-11-05; target picked: **Gemma 4 12B** |
| [Training data](./data.md) (corpus, rules, mix) | pilot-1 mix built; examples expansion under way |
| [Pilot-1](./pilot-1.md) (12B LoRA, weekend run on the rig) | trained 2026-11-06; held-out A/B running |
| Pilot-2 (habit-weighted data, deeper LoRA) | planned |
| Published fine-tunes | per release, from an artifact bucket linked here ([D-FT-2](./decisions.md)) |

## Pages

- [Decision log](./decisions.md): every decision with its date, owner and reason.
- [Baseline](./baseline.md): which model size knows enough AILANG to be worth tuning.
- [Training data](./data.md): where the data comes from, the rules, and coverage gaps.
- [Pilot-1](./pilot-1.md): the recipe, the training curve, the traps, and the A/B result.
- [Serving costs](./costs.md): rig against Cloud Run, Vertex and per-token hosts.

The working design doc, with the full detail, is
`design_docs/planned/v1_1_0/m-eval-finetuning-data-pipeline.md` in the repository. The tooling is
in `tools/finetune/`.

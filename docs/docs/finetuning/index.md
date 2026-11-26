---
sidebar_position: 1
title: Fine-tuning AILANG models
description: Research log for training small, local AILANG-expert models — what we measured, what we decided, and what we publish.
reviewed: 2026-11-25
reviewBy: 2027-02-25
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

## The approach (two acts)

We began by fine-tuning for **agent habits, not memorisation** ([Pilots 1–4](./pilot-4.md)):
- write its solution through the tool and never answer in chat;
- check every write and apply the fix the compiler names;
- look up a stdlib name before guessing it;
- trust the compiler over its own memory;
- stop once the program is verified.

Four pilots proved that trajectory SFT fixes tool-protocol discipline (0 protocol misses, 3× fewer
tokens on passes), **but habits cannot substitute for knowing the language**: an untrained Gemma 4
12B given the same 2.4 KB slim prompt matched our best habit fine-tune (`9–10/23` vs `10/23`,
[D-FT-18](./decisions.md)), while the full teaching prompt lifted the untrained model to `16/23`.
[Pilot-5](./pilot-5.md) ([D-FT-19](./decisions.md)) therefore reverses the order: **teach the
language first** from verified repository and benchmark programs (measured on a fast single-shot
gate), while preserving the model's test-time reasoning, and layer agent habits back in only once
the language gate passes. See [Training data](./data.md).

## Status

| Stage | State |
|---|---|
| [Baseline](./baseline.md) (size ladder, speed, parallelism) | done 2026-11-05; target picked: **Gemma 4 12B** |
| [Training data](./data.md) (corpus, rules, mix) | 63 stdlib/core examples landed (95% stdlib coverage); Pilot 1–4 habit/repair mixes + Pilot-5 near-deduped repo descriptions (1,753) and verified benchmark solutions (491) built; Stage 1 expansion next |
| [Pilot-1](./pilot-1.md) (12B LoRA) | negative, causes found (session gate, thinking format, context) |
| [Pilot-2](./pilot-2.md) (served format, habit episodes) | agent protocol fixed (0 protocol misses), 3× fewer tokens, but 4/24 against base's 6/13 |
| [Pilot-3](./pilot-3.md) (its own mistakes, repaired) | **10/24** with the slim prompt (pilot-2: 4/24); 6 newly pass, none newly fail. Left behind: dropped runaway writes, long loops |
| [Pilot-4](./pilot-4.md) (second round, short writes, loop-breaking) | **5/23**, below pilot-3. Two diagnostic arms: the untrained model with the slim prompt scores 9/23 (10/23 with a working output limit), level with the best fine-tune (10/23); base with the full prompt scores 16/23 |
| On-policy repair line | **stopped 2026-11-16** ([D-FT-18](./decisions.md)): three rounds of training added nothing measurable over the slim prompt alone. Fixed `max_tokens` enforcement on local rig models |
| [Pilot-5](./pilot-5.md) (teach the language first) | **running** ([D-FT-19](./decisions.md)). Single-shot baseline: untrained 12B passes 0% (no prompt), 3% (slim) and **39%** (full prompt, 44% compile). Run a (1,504 repo programs) learned repo modules (34%/39% compile vs 0%/12.5% untrained) but did not transfer to benchmark tasks (0%/6% pass). Runs b & b2 (rank 128) diverged. Run c (+491 benchmark solutions, 2,004 total) reaches **44% slim-prompt compile** (matching full-prompt base!), yet passes **11% / 17% / 18%** (no / slim / full) vs the 39% gate because code-only targets suppressed reasoning (~430 tokens vs 3k–6k). Next: restore thinking in the examples and scale Stage 1 data |
| Published fine-tunes | per release, from an artifact bucket linked here ([D-FT-2](./decisions.md)) |

## Failed paths, and when to walk them again

A reader asked: *do you store failed paths somehow, so they are not walked again, or walked with a
better model?* Yes. Every path that was stopped is written down in three places:

- The page for that pilot holds a dated log of each step and its measured result, including the
  ones that went wrong.
- [Decisions](./decisions.md) records each choice with a number (D-FT-n), who made it and the
  measurement behind it. A stopped line gets a decision of its own.
- [Field notes](./field-notes.md) turns each failure into a general lesson.

The data is kept as well as the write-up. Each eval arm's results and request traces stay on disk,
so a later question can often be answered from an old run without repeating it. Pilot-4's
one-hour timeouts were explained that way, from traces recorded days earlier.

A failed path is a result under specific conditions: this model, this data, this harness. The
table lists each stopped path with what would make it worth another try. The last column is our
judgement, not a measurement.

| Path | What the measurement showed | Recorded in | Worth walking again when |
|---|---|---|---|
| Train on agent runs from frontier models (pilot-1) | Negative, with the causes found: a harness gate, context, and a thinking format in training that differed from the one served | [Pilot-1](./pilot-1.md) | Not as it was. Training in the served format is now a rule for every run |
| Train agent habits with a slim prompt (pilot-2) | The agent protocol was fixed and tokens fell three times, but it passed fewer tasks than the untrained model | [Pilot-2](./pilot-2.md) | The model already knows the language and only its tool use is weak |
| Repair the model's own mistakes and train on the repairs (pilots 3 and 4) | Three rounds added nothing measurable over the slim prompt alone. Stopped by D-FT-18 | [Pilot-3](./pilot-3.md), [Pilot-4](./pilot-4.md) | A model that passes the language gate still fails as an agent. The 7,860 passing agent runs are kept for that case |
| Give a fine-tune the full teaching prompt as a fallback | Worse than no training, twice: 1 of 23 in pilot-3, and 0% against 39% in pilot-5 run a | [Pilot-4](./pilot-4.md), [Pilot-5](./pilot-5.md) | The full prompt is part of the training examples, which needs a trainer that can take 27,000-token sequences |
| Retry a dropped response in the agent harness | Not built. Five of six dropped runs were the model repeating itself, which a retry would not fix | [Pilot-4](./pilot-4.md) | Dropped responses are common in a model that otherwise writes correct programs |
| Raise LoRA rank from 32 to 128 on 12B (pilot-5 runs b and b2) | Diverged twice: at step 160 (unscaled `scale=10`) and at steps 880–980 even with `scale=2.5` (`scale × rank = 320`), with no validation-loss advantage over rank 32 before breaking | [Pilot-5](./pilot-5.md) | Lower learning rate (`≤1e-5`) or QDoRA / gradient clipping if rank 32 saturates on the full 10k+ dataset |
| Train code-only targets under `enable_thinking=True` with no `reasoning` field (pilot-5 runs a and c) | Reached 44% slim-prompt compile rate (matching full-prompt base), but trained an empty `<\|channel>thought\n<channel\|>` span that suppressed test-time reasoning (~430 tokens vs 3k–6k) and dropped pass-given-compile from 88% to 38% | [Pilot-5](./pilot-5.md) | Never for multi-step algorithmic tasks; always populate `reasoning` or mask the empty thought prefix |
| A larger base model on the rig | Not trained. A 27B model takes 49 GB and the rig loads one model at a time, so it cannot run parallel agents | [Why](#why), above | The 12B cannot hold the language at any adapter size, or the rig can run several models at once |

Two habits keep an old path from being walked again by accident. The untrained model is measured
under the same conditions as every fine-tune, so a result is always read against a control. And
every eval input is frozen (compiler, standard library, tasks, prompts), so an old number can
still be compared with a new one.

## Pages

- [Field notes](./field-notes.md): what fine-tuning is like day to day, drawn from what happened here.
- [Decision log](./decisions.md): every decision with its date, owner and reason.
- [Baseline](./baseline.md): which model size knows enough AILANG to be worth tuning.
- [Training data](./data.md): where the data comes from, the rules, and coverage gaps.
- [Pilot-1](./pilot-1.md): the recipe, the training curve, the traps, and the A/B result.
- [Pilot-2](./pilot-2.md): training in the served format, and what it fixed and did not.
- [Pilot-3](./pilot-3.md): training on the model's own mistakes, and the habit of looking things up.
- [Pilot-4](./pilot-4.md): a second round, short writes, and breaking loops.
- [Pilot-5](./pilot-5.md): teach the language first, from our own verified code.
- [Serving costs](./costs.md): rig against Cloud Run, Vertex and per-token hosts.

The working design doc, with the full detail, is
`design_docs/planned/v1_1_0/m-eval-finetuning-data-pipeline.md` in the repository. The tooling is
in `tools/finetune/`.

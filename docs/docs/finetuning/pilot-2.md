---
sidebar_position: 6
title: Pilot-2 (served format, habit episodes)
description: The second AILANG fine-tune — trained in the format it is served in, on habit-simulated agent episodes — and its held-out result against base Gemma 4 12B.
reviewed: 2026-11-09
reviewBy: 2027-02-09
---

# Pilot-2: served format, habit episodes

Weekend run on the rig, 2026-11-07/08. Pilot-2 kept the base model and the held-out set, and
fixed what [pilot-1](./pilot-1.md) got wrong.

## What changed from pilot-1

| | Pilot-1 | Pilot-2 |
|---|---|---|
| Training format | thinking **off**, one-line system prompt | thinking **on** with a thought before every action; the exact lean pi wire format ([D-FT-8](./decisions.md), [D-FT-9](./decisions.md)) |
| Agent harness at eval | pi default (7,338-token prefix) | lean pi profile (~910 tokens) |
| Data | trajectories from other harnesses (55% with stale paths), per-turn samples | 3,011 habit-simulated episodes with real tool output: solve, repair chains, examples ([Training data](./data.md)) |
| Samples | one per assistant turn | one per episode, with loss on every assistant turn |
| LoRA | top 16 blocks, rank 16, 1 epoch | all blocks, rank 32, 2 epochs |

## Training

- **The first attempt diverged.**
  - Settings: learning rate 1e-4, LoRA scale 20.
  - Train loss jumped from 0.31 to 10.2 at step 80, as warmup ended. Validation went from
    0.59 to 6.76.
- **The re-run trained cleanly.**
  - Settings: learning rate 2e-5, LoRA scale 10, warmup 100.
  - It took ~21 h for 2,368 steps.

| Step | 0 | 200 | 1000 | 1200 | 2368 |
|---|---|---|---|---|---|
| Val loss | 0.592 | 0.191 | 0.176 | 0.169 | 0.175 |

- Served with the base model's sampling defaults (temperature 1, top_k 64, top_p 0.95).

## Held-out A/B (lean profile, agent mode)

Each arm had a 5-hour wall-clock cap; A and C hit it before finishing all 24 benchmarks. The
base + slim-prompt arm (A2) did not run before the weekend window closed.

| Arm | Model | Prompt | Pass | Core | Stretch | Failures | Median output tokens per pass |
|---|---|---|---|---|---|---|---|
| A | base 12B | full teaching prompt | 6/13 (46%) | 6/10 | 0/3 | 3 compile, 3 timeout, 1 runtime | 9,869 |
| B | pilot-2 | 2.4 KB slim prompt | 4/24 (17%) | 3/12 | 1/12 | 16 compile, 3 logic, 1 runtime | 3,097 |
| C | pilot-2 | none | 4/19 (21%) | 2/9 | 2/10 | 11 compile, 2 logic, 1 runtime, 1 timeout | 2,185 |

On the 10 benchmarks all three arms reached, A passed 5, B passed 1 and C passed 2.

**The kill criterion (B ≥ A) fails.**

## What it shows

- **The agent protocol is fixed.** Pilot-1 failed every agent run on protocol: no file written,
  tool calls lost in the thinking channel. Pilot-2 has **zero protocol misses**. Every failure is
  a real program that does not compile or prints the wrong output.
- **Concise only when it succeeds.** On its passes the fine-tune used about a third of base's
  output tokens. Per run it is the other way round, because failing runs loop:

  | Arm | Median output tokens per run | Median turns | Runs ≥ 50 turns | Final error is a parse error |
  |---|---|---|---|---|
  | A (base) | 9,728 | 4.5 | 0/10 | 1/10 |
  | B (ft + slim) | 17,168 | 57.5 | 13/24 | 14/24 |
  | C (ft, no prompt) | 12,055 | 29.5 | 8/18 | 9/18 |

  A's 3 other rows hit pi's 1-hour hard timeout with no output.

## Diagnosis (2026-11-09)

The banked rows (final code, final error, tool histogram, turns) are enough to see what went
wrong:

- **The fine-tune's syntax is wrong.** Two thirds of its failures end in a **parse** error, and the
  errors are borrowed syntax:
  - `as` casts and `->` match arms;
  - bare assignment without `let`;
  - `;` in expression-bodied functions, and `?` inside interpolation;
  - a missing `{`.

  Base's failures are type and effect errors in programs that parse.
- **It loops instead of looking things up.** Typical failing runs are 40–80 write → check cycles
  that rewrite the whole file and never converge (merge_sort: 408 turns). It almost never reads
  docs or examples once an error appears.
- **The training data did not cover this.** Of 1,671 repair episodes:
  - 618 fix `undefined variable` and 202 fix exhaustiveness;
  - **26 fix a parse error**;
  - no episode shows the "two failed fixes, so look it up" move. The 241 episodes with an
    `ailang docs` or `ailang examples search` lookup all do it *before* the first write.

  The repair chains came from staleness pairs: programs that once compiled and later broke on
  checker changes. That source cannot produce the syntax mistakes a model makes on its own.

So the result is not a ceiling. The model learned the shape of the loop, but the data never showed
it its own kinds of mistakes, or the habit of looking up after a failed fix.

## Open questions for pilot-3

These are not yet decided.

- **On-policy repair data (best supported by the diagnosis).** Generate episodes from the
  fine-tune's *own* failures:
  1. Run pilot-2 on the training benchmarks only, never the held-out 24.
  2. Keep its real parse and type errors.
  3. Repair each one with the existing iterative `stale_fix.py` loop, verified against the
     expected output.
  4. Render the repairs as episodes. After a second failed fix, the episode does a lookup:
     `ailang docs`, `ailang examples search`, or `ailang prompt` for the syntax.

  Separately, cap or penalise turn counts over ~20 in the episode mix.
- **Fine-tune + the full teaching prompt.** A fine-tune that carries the habits, given the prompt
  for language knowledge, might beat base while staying concise. This arm was never run.
- **More language data.** The 2,733 verified repo programs and the eval passes were unused as agent
  episodes. Adding them would widen the language the model has seen beyond the 57 benchmarks.
- **Self-distillation.** Base 12B's own verified thinking would give richer reasoning than the
  fact-based thoughts.
- **A2 and complete arms.** Every arm should finish all 24 benchmarks; the time caps cut A and C
  short.

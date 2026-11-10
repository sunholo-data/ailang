---
sidebar_position: 2
title: Field notes (what fine-tuning is like day to day)
description: Lessons from running AILANG fine-tunes, each drawn from something that actually happened - where the time goes, why the score lies, and the habits that kept the work honest.
reviewed: 2026-11-10
reviewBy: 2027-02-10
---

# Field notes: what fine-tuning is like day to day

The pilot pages tell each run's story in order. This page pulls out what those stories teach. Every
lesson comes from something that actually happened in this project, with the numbers we measured,
and links to the page where it happened.

The short version: **training is the easy part.** Most of the work is building data you can trust,
and finding out what a score actually means.

## 1. Most of the time is not training

One full cycle of [pilot-3](./pilot-3.md), from its own log:

| Step | Took | Uses |
|---|---|---|
| Run the previous model on the 57 training tasks, recording every request | 8 h 38 min | GPU |
| Turn the recording into 349 broken-file items | 4 min | CPU |
| Have Opus repair each one, keeping only verified fixes | 29 min | Claude subscription |
| Build episodes, thoughts and the training mix | 6 min | CPU |
| Train (784 samples, one epoch) | 7 h 58 min | GPU |
| Package the model for serving | 75 s | GPU |
| One held-out evaluation run (24 tasks) | 5 h 39 min | GPU |

Training is a third of the GPU time. Collecting and evaluating take the rest, and the hours in
between go on reading transcripts and fixing what the numbers turned out to hide.

## 2. The score is a symptom: read what the model actually did

A pass rate says *that* something failed, never *why*. Three times, the obvious reading was wrong:

- **Pilot-1 scored 0/24** in agent mode. The model was fine: a session gate in the agent harness
  refused every file write until the model sent an acknowledgement it was never trained to send
  ([pilot-1](./pilot-1.md), [D-FT-6](./decisions.md)).
- **Pilot-1's tool calls vanished at random.** It was trained with thinking off and served with
  thinking on, so the server sometimes filed its tool calls as "thoughts"
  ([D-FT-8](./decisions.md)).
- **Pilot-3 "read the file, then stopped"** in 4 of 24 runs. It had not stopped. It wrote
  thousands of tokens of program inside a `write` call that ran away or ended malformed, the
  server's parser dropped it, and the harness read the empty reply as the end
  ([pilot-4](./pilot-4.md)).

**The habit:** record the raw requests and responses of every eval, and read a few failures before
theorising. Every held-out run is now traced, and dropped tool calls are counted automatically.

## 3. Your first theory is often wrong, so test it cheaply

For pilot-3's early stops, we had a neat explanation: in 38% of training samples, the step after
reading the file was masked from training, so the model never learned "after reading, write".
It fit the numbers. A 10-minute probe replayed the exact moment, after reading, on 16 training tasks.
Both models wrote every time, so the theory was wrong.

**The habit:** before acting on an explanation, find the smallest experiment that could prove it
wrong, and run that first.

## 4. Train in exactly the format you serve

The model learns the exact token sequence it is shown. Small mismatches between training and serving
become behaviour:

- Gemma 4's chat template adds an empty thinking block when asking for a reply, but not inside a
  finished conversation. Stock prompt-masking therefore hid the first 4 target tokens, including the
  tool-call token, on 1,497 of 2,193 pilot-1 samples ([pilot-1](./pilot-1.md)).
- Training cuts over-long samples **from the end**. In pilot-3's on-policy samples the end is the
  verified fix, the only part with a loss. 17 samples would have trained on nothing
  ([pilot-3](./pilot-3.md)).

**The habit:** render training samples with the same code path and tokenizer as training itself.
A character-count estimate flagged 22 too-long samples; the real tokenizer found 16.

## 5. Train on the model's own mistakes

Pilot-2 learned the agent protocol (0 protocol misses) but scored 4/24. Its failures were mostly
syntax borrowed from other languages, and its repair data barely covered them: 26 parse-error
episodes out of 1,671. The data showed someone else's mistakes, not the model's.

Pilot-3 ran pilot-2 on the training tasks, kept every file it wrote, had Opus repair the broken
ones, and trained on those repairs. The model's own wrong attempt stays in each episode as
context, with no training loss, so it sees the mistake without learning to make it. Result: **4/24 →
10/24**, six tasks newly passing and none newly failing. Pilot-4 repeats this with pilot-3's
mistakes, pooled with the first round.

## 6. Never train on what you test

24 benchmarks were set aside before any data was built ([D-FT-H](./decisions.md)). Each tool refuses
to collect, extract or probe them. When we needed transcripts of held-out failures to diagnose the
early stops, that run went to a scratch folder that never feeds training.

## 7. Verify every example, even from the best model

Training data comes from the strongest models available ([D-FT-7](./decisions.md)), but nothing is
kept on trust. An Opus repair is kept only if it compiles **and** prints exactly the expected output:
348 of 349 passed in pilot-3. Even the best model needs managing as a tool. Asked to fix code with
its tools switched off, Opus spent 15 minutes writing pretend tool calls as text. Telling it
plainly that it had no tools fixed that: 7.5 seconds per repair.

## 8. Harness bugs look exactly like model results

Some of the biggest swings came from the plumbing, not the model:

- A deprecated logging library printed a notice above **every** tool result in every agent eval, on
  every model. Fixed in [#2151](https://github.com/sunholo-data/ailang/issues/2151); pilot-2's base
  numbers saw it, so base was re-run from scratch.
- The prompt loader silently fell back to the full teaching prompt on a bad version name, so an arm
  could run the wrong prompt without anyone knowing. Every arm now checks the prompt version it
  banked.
- A collection script replaced the rig lock's release hook with its own, leaving the GPU "taken" by
  a process that had exited.

**The habit:** before trusting a comparison, check the arms differ only in what you meant to change.

## 9. One try per task is noisy: count the flips

Each held-out task runs once, so a single pass or fail is partly luck. A total like 10/24 against
4/24 hides which tasks changed. Comparing task by task, six flipped from fail to pass and none the
other way; by chance, a six-to-none split has a probability of about 3%. That kind of comparison
says something is real, even when the exact size of the gain is uncertain.

## 10. Shared hardware is a team sport

The GPU rig also serves nightly evals and other agents. Jobs take a lock, hand it over politely, and
a gateway refuses long GPU calls from anyone not holding it. During pilot-3, the gateway correctly
refused one of our own diagnostic probes while the eval held the lock; the probe ran later, under its
own lock. The rules are only useful if they are followed when they are inconvenient.

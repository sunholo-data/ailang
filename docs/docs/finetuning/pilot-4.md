---
sidebar_position: 9
title: Pilot-4 (a second round, short writes, breaking loops)
description: The fourth AILANG fine-tune, in progress. A second round of the model's own mistakes, pooled with the first, plus fixes for the two failures pilot-3 left behind - runaway writes and long loops.
reviewed: 2026-11-10
reviewBy: 2027-02-10
---

# Pilot-4: a second round, short writes, breaking loops

**Status: in progress.** Like [pilot-3](./pilot-3.md), this page is written as the work happens.

## Where pilot-3 left us

Training on the model's own mistakes worked. On the 24 held-out benchmarks with the slim prompt,
pilot-3 passed **10/24 against pilot-2's 4/24**. Six benchmarks newly passed and none newly failed.
Compile errors fell from 16 to 6, and the median run from 57 turns to 12.

It also left two failures behind, and finding out what they really were took some care.

### "Read, then stop" was a lost tool call

Four runs read the starter file and ended without writing anything. It looked like the model giving
up. Our first theory was that the training data rarely showed a trained write after the first read,
because on-policy episodes mask the model's own first attempt. A probe disproved that: on 16
training benchmarks, pilot-3 and pilot-2 wrote after reading every time.

The trace told the real story. In the run that stopped, the model generated 4,340 tokens and pi
received about 80 of them. The rest was a `write` call holding the whole program. Some such calls
**run away**: one repeated the same function definition 225 times. Others end malformed. Ollama
cannot parse either kind, returns an empty "stop", and the agent harness reads that as the end of
the run.

| | Responses dropped this way |
|---|---|
| pilot-2 (4,106 requests) | 2 (0.05%) |
| pilot-3 (223 requests) | 4 (1.8%) |

Every drop ends the run, and in the diagnosis run that was half the failures.

### Loops

Three runs hit the 1-hour limit, and others ran to 89 and 146 turns. In round 1's trace, 14
benchmarks had the same kind of error fail four or more times in a row: fix after fix of one
mistake, never stepping back.

## The plan

| Step | What | Why |
|---|---|---|
| Collect round 2 | Run pilot-3 on the 57 training benchmarks, one trial, every request traced | Its mistakes are different now; this also measures it on its own training set (pilot-2: 30/57) |
| Repair | Opus fixes every failing file, kept only if it compiles and prints exactly the expected output | 348/349 verified in round 1 |
| Pool | Round-1 + round-2 repairs, plus a replay sample of earlier data | Aggregating rounds keeps everything learned (DAgger-style) |
| Short writes | Drop trained writes over 6,000 characters; add **build-up** episodes | Long writes are what run away |
| Break loops | Episodes ending a loop of 3+ failed fixes: read the reference, rewrite the failing part | Targets the timeouts |
| Train | From the **pilot-2** adapter, 1 epoch, learning rate 1e-5 | Not stacked on pilot-3, whose adapter learned the runaway writes |
| Gate | 30 minutes on training benchmarks: act after reading? dropped-call rate ≤ 1%? | Catch a regression before 12 hours of evals |
| Evaluate | Held-out 24: fine-tune + slim prompt, and base 12B once, both traced | The banked row cannot tell a dropped call from a real stop |

The decisions are D-FT-13 to D-FT-16 on the [decisions page](./decisions.md).

### Build-up episodes

A long program is written in two steps instead of one large `write`:

1. **write** the helpers, with `main` as a stub whose body is `()`,
2. **`ailang check`** that part,
3. **edit** the stub `main` into the remaining helpers and the real `main`,
4. **run** it.

Each episode is verified: the first part must type-check, and the edit must reproduce the full
program exactly, which must print the expected output. The stub form was checked with
`ailang check` before any episode used it.

### Loop-break episodes

The extractor now records each failing file's **history**: up to three earlier attempts that failed
with the same kind of error, oldest first. When there are two or more, the episode replays them all
as context (no loss), then teaches the step out:

> That is 3 fixes in a row for the same kind of error, and it still fails: … Patching again will not
> help. Read the "Pattern Matching" section of the AILANG reference and rewrite the failing part the
> way it shows.

Round 1's 348 repairs were matched back to their history by code hash, so no repair was bought
twice: 110 of them end a loop of 3 or 4 failed attempts.

### The compiler's own hint can feed a loop

`MOD010` says: *module 'rle/decode' doesn't match file path 'benchmark/solution'. Fix: use --relax-modules flag or set AILANG_RELAX_MODULES=1*. An agent cannot change how its program is run, so the only fix it can make is the module line, which the hint does not mention. A model that follows error messages faithfully, which is the habit we train, gets pointed the wrong way. The training episodes now read the reference rule instead; the message itself is a candidate AILANG fix.

### Measuring what the banked row cannot see

- `tools/finetune/trace_dropped.py` counts dropped tool calls in any traced run, and which runs
  ended on one. Checked against the known traces: 4/223 for pilot-3's diagnosis run, 2/4,106 for
  pilot-2's collection.
- `tools/finetune/probe_after_read.py` replays the real "after reading the template" requests from
  a training trace and records the first action. It refuses held-out benchmarks.
- Every held-out arm runs through the trace proxy, so each eval keeps its transcripts.

## Progress log

| Date | Step | Result |
|---|---|---|
| 2026-11-10 | Paused pilot-3's remaining evals; designed pilot-4 with Mark | Decisions D-FT-13 to D-FT-16 |
| 2026-11-10 | Started round-2 collection: pilot-3 on the 57 training benchmarks, traced | Same lane budgets as round 1 (25 min, 1.5M tokens per benchmark) |
| 2026-11-10 | Added attempt history to the extractor; matched it onto round 1's repairs | 348/348 matched; 110 end a loop of 3+ failed attempts |
| 2026-11-10 | Added build-up and loop-break episodes, the write-length cap, the dropped-call counter, the probe and `pilot4_run.sh` | Counter reproduces the known drop rates |
| 2026-11-10 | Test-built pilot-4 episodes from round 1 alone | 110 loop-break episodes, but only 60 had a lookup: the error-to-section map covered parse errors only. Added `MOD010` → "Required Program Structure" (its rule 1), Option/Result mix-ups → "Constructing Option Values", and a signature lookup for type mismatches in calls to stdlib functions: **83/110**. Errors with no section that teaches the fix (bool ordering, unclosed braces) get none, rather than a wrong one. 119 build-up episodes verified out of 131 long programs |

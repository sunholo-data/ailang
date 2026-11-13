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
| 2026-11-10 | Round-2 collection finished (5 h 50 min) | Pilot-3 passed **35/57** of its training benchmarks (pilot-2: 30/57). **13 of 1,539 responses were dropped tool calls (0.84%)**, and every one ended its run: the runaway write shows up on training tasks too, not only on held-out ones. 374 failing file states, 70 of them at the end of a loop |
| 2026-11-10 | Opus repaired round 2 (43 min, subscription) | 372/374 verified (2 printed the wrong output and were dropped) |
| 2026-11-10 | Pooled rounds 1 + 2 into the mix | 719 on-policy episodes (179 loop-break, 300 with a reference lookup). Mix: 1,062 training samples, 5.6M tokens: 555 on-policy, 204 solve, 166 repair, 104 examples, 33 build-up. The write cap removed 106 samples, the token limit 46 (all long loop-break episodes) |
| 2026-11-10 | Paused the overnight model rotation and started training: 265 updates from the pilot-2 adapter | Training jobs and eval rotations share one GPU lock. The rotation had taken it while the data was being built; Mark gave the fine-tune priority |
| 2026-11-11 | Training finished (12 h 20 min) | Validation loss 0.222 → 0.075 by update 50, flat near 0.065 after that, 0.056 at the end |
| 2026-11-11 | **Gate passed** | The probe: pilot-4 acted after reading 16/16. Eight traced training runs: **1 of 1,075 responses dropped (0.09%)**, against pilot-3's 0.84% on the same kind of task. Six of eight passed, but these are training benchmarks, so that number is not a score |
| 2026-11-11 | **Arm B: pilot-4 + slim prompt on the held-out 24: 5/24** | A regression from pilot-3's 10/24. Paired: pilot-4 alone passed 1, pilot-3 alone passed 6 (sign test p = 0.125, so not yet significant, but the direction is clear). **13 of 24 runs ended on a dropped tool call**, including 4 of the 6 that pilot-3 passed. The response rate looks harmless (13/2,516, 0.52%); the drops land at the moments that end runs |
| 2026-11-11 | Where the drops fall | After a reference lookup: **6 of 350 responses (1.7%)**. After anything else: 7 of 2,142 (0.33%). The gate saw only 34 post-lookup states, all on training tasks, and none dropped. The training data after a lookup looks normal (writes median 1,387 characters, no code fences), so the next step is to replay those requests and read the raw text the model produced |
| 2026-11-12 | Arm A: base 12B + full teaching prompt on the held-out 24 | **17/24**, the same as pilot-1's arm A. 1 of 397 responses dropped (0.25%), at about 17 responses a task against pilot-4's 105. The base model with the full prompt is still the bar the fine-tunes have to reach |
| 2026-11-12 | Raw replay of the 6 dropped post-lookup turns, 2 samples each | **12 of 12 came back as complete, well-formed tool calls**: 9 writes or edits, 3 more lookups, quotes balanced, no runaway, no stray markdown. The drop does not reproduce on demand, so it is not a fixed habit the training taught. It is a small per-response chance of a call ollama cannot parse, and pilot-4's runs are long (105 responses a task against the base model's 17), so most runs meet one. A per-run probability, 1 − (1 − 0.005)^105, is about 40% |
| 2026-11-12 | Two diagnostic arms started, to separate what training adds from what the prompt adds | A2: base 12B + slim prompt (no training at all). D: pilot-3 + full teaching prompt. Neither had been run in any pilot |
| 2026-11-12 | The stdlib changed under the running arm | The suite ran from the repo checkout, and scoring loads `std/` from the working directory when one is there. A merge renamed stdlib functions, the checkout fast-forwarded, and the installed binary could no longer parse the new files. Two results were lost and re-run; none of the others had reached a stdlib load. The arms now run from a directory holding a snapshot of `benchmarks/` and no `std/`. An eval that takes a day needs its inputs frozen, not just its binary |
| 2026-11-12 | **Arm A2: base 12B + slim prompt: 9/23** (the swapped `run_length_encode` task is left out of every comparison from here, see D-FT-17) | On the same 23 tasks pilot-3 passed 10, pilot-4 passed 5 and base + full prompt passed 16. Paired against pilot-3: A2 alone passed 1, pilot-3 alone passed 2. **Training has not added anything measurable over the slim prompt on an untrained model.** 11 of A2's 14 failures were one-hour timeouts; 3 of 983 responses were dropped tool calls |
| 2026-11-12 | What the full prompt is worth to the base model | Base + full prompt alone passed 7 tasks that base + slim prompt failed, and the reverse never happened (sign test p = 0.016). That gap of 7 tasks is the knowledge a fine-tune has to carry for the slim prompt to be enough, and three pilots have not moved it |
| 2026-11-13 | **Arm D: pilot-3 + full prompt: 1/23** | Pilot-3 with the slim prompt alone passed 9 tasks that this arm failed, and the reverse never happened (p = 0.004). 11 runs ended on a dropped tool call and most of the rest handed in code with parse errors after a few minutes. The fine-tune was trained with the slim prompt and does not work with the full one, so adding the prompt back is not a way to rescue it |

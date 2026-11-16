---
sidebar_position: 10
title: Pilot-5 plan (teach the language first)
description: A proposed plan for the fifth AILANG fine-tune. It trains on the verified AILANG programs in our own repositories, which earlier pilots left unused, as single-turn examples, and it measures language knowledge with a fast single-shot test before any agent eval.
reviewed: 2026-11-16
reviewBy: 2027-02-16
---

# Pilot-5 plan: teach the language first

**Status: approved by Mark on 2026-11-16 ([D-FT-19](./decisions.md)). Stage 0 is running.**

## Why a new plan

Four pilots trained a Gemma 4 12B adapter to behave well as an agent: check every write, read the
error, look things up. On the 23 held-out tasks, none of them beat the untrained model given the
same slim prompt ([pilot-4](./pilot-4.md), [D-FT-18](./decisions.md)).

The same measurements showed where the gap is:

| Arm | Passes out of 23 |
|---|---|
| Untrained model + full teaching prompt | 16 |
| Untrained model + slim prompt | 9 (10 with a working output limit) |
| Best fine-tune (pilot-3) + slim prompt | 10 |

The full prompt is worth six or seven tasks, and it contains language knowledge, not agent habits.
When the slim-prompt runs fail, they fail on compile errors, wrong output, and long loops over
syntax the model does not know.

The pilots also used very little of the AILANG we have. Pilots 2 to 4 each trained on about 1,000
samples from roughly 165 benchmark tasks, plus 100 to 200 files from `examples/`. They used none of
the programs in the packages, the motoko checkouts, the demos or the downstream applications.

So pilot-5 changes the target. It teaches the language directly, from that unused code, and adds
agent behaviour only if the language knowledge shows up first.

## What we have to train on

Counted on 2026-11-16 from the extraction of 2026-11-06. Every program here passes `ailang check`
on the compiler of that date, and none matches a held-out solution.

| Source | Verified and available | Used by pilots 2 to 4 |
|---|---|---|
| Programs in our repositories | 2,733 unique programs, 594,000 lines | none |
| Passing eval solutions | 6,299 programs over 75 training tasks | 200 to 600 per pilot |
| `examples/` | 560 files | 100 to 200 per pilot |
| Failed eval programs with their compiler error | 4,171 | as agent repair episodes only |

Three facts about the repository programs shape the plan:

- **They have no task description.** A training example needs a prompt. Pilot-1 dropped 508
  programs for this reason.
- **Half are short and a third are long.** The median is 82 lines; 1,795 are 150 lines or fewer and
  938 are longer. One program of 2,000 lines is a poor single example.
- **Coverage of the standard library is uneven.** They import 52 `std` modules. `std/list`,
  `std/string`, `std/result` and `std/option` each appear in more than 1,100 programs; 17 modules
  appear in fewer than five.

## The plan

### Stage 0: a fast test of language knowledge

The agent eval takes 12 hours per arm and gives one noisy trial per task. Pilot-5 needs a test that
runs in minutes, so that every data and training choice can be checked.

- **The test:** give the model a task description and ask for one complete program, with no tools.
  Score whether it compiles and whether its output is right.
- **The tasks:** the held-out tasks, three samples each, and about 150 repository programs set
  aside before any training data is built, so the test also covers code that looks like our real
  applications.
- **The baseline:** the untrained model with no prompt, with the slim prompt and with the full
  prompt. The distance between the first and the last of those is what training has to close.

This stage needs no training. It also yields the first result worth recording: how much AILANG the
untrained model knows with nothing in its context.

### Stage 1: build the training set

All of it is single-turn: a prompt, then a program or part of one.

| Part | How it is made | Rough size |
|---|---|---|
| Whole programs | A strong model writes the task description for each repository program of 150 lines or fewer. The answer already exists and is already verified | 1,800 |
| Parts of long programs | Each long program is split by function. The prompt gives the module so far and a description of the next function | 4,000 to 6,000 |
| `examples/` | Every file, with a description | 560 |
| Eval solutions | At most five per task, from different models | 375 |
| Repairs | A broken program and the real compiler error, answered by the fixed program. Sources: the 4,171 failed eval programs, and verified programs broken on purpose in the ways models most often break them | 2,000 |
| Standard library reference | For each exported function: its signature and one short verified use | 1,000 |
| New problems | Written only for modules and syntax the rest does not cover. Each is kept only if it compiles and passes its test | 500 to 1,000 |

That gives 10,000 to 13,000 examples, about ten times the size of any earlier mix.

Four rules apply to all of it:

1. **Descriptions come from the best model available** (Opus through the Claude subscription,
   [D-FT-7](./decisions.md)). A description must not quote the solution.
2. **Near-duplicates are removed before splitting.** The repositories hold many copies of the same
   package in different worktrees. Exact copies are already removed by hash; similar copies are
   not, and they would let test programs leak into training.
3. **Everything is re-checked with one pinned compiler** on the day the set is built. The standard
   library has changed since the extraction.
4. **Each example is rendered with one of three prompts:** none, slim or full. Pilot-3 was trained
   with the slim prompt only and fell to 1 of 23 when given the full one.

### Stage 2: a small training run first

Train a fresh adapter from the untrained model on about 2,000 examples, which is a few hours of
GPU. Score it with the Stage 0 test.

- If the compile rate with no prompt rises clearly over the untrained model, go on to Stage 3.
- If it does not move, the full run would not either. Stop and change the training setup before
  spending days of GPU: a larger adapter first, then full fine-tuning or a larger base model.

This run is also where the adapter size is chosen. Two short runs, at rank 32 and rank 128, scored
on the same test, will show whether size matters at this scale.

### Stage 3: the full run and the bar

Train on the full set and score with the Stage 0 test, then with the agent eval.

| Gate | Measure | Bar |
|---|---|---|
| Language | Single-shot pass rate, fine-tune + slim prompt | At least the untrained model + full prompt |
| Agent | 23 held-out agent tasks, fine-tune + slim prompt | At least 16, the untrained model + full prompt |

The agent gate is the kill criterion from the original design. Both arms run from the frozen eval
root with the output limit in place.

### Stage 4: agent runs, only if needed

If the language gate passes and the agent gate fails, the model knows the language but handles the
tools badly. Only then add agent runs back, as a small share of the mix, drawn from the 7,860
passing episodes we already hold.

## What could go wrong

- **The adapter may be too small to hold a language.** Low-rank adapters are well proven for
  changing habits and less so for adding this much knowledge. Stage 2 exists to find this out for
  a few hours of GPU.
- **The code may teach dated style.** Everything compiles, but some of it was written by agents
  for older versions. The canonical `examples/` and the reference entries are there to anchor
  current style, and the mix can weight them up.
- **Descriptions may be poor.** A vague description teaches the model to guess. A sample of 50 will
  be read by hand before the rest are generated.
- **Twenty-three tasks is a small test.** A difference of two or three passes is noise. The
  single-shot test, with several samples per task and 150 extra programs, is the more sensitive
  instrument, and the agent result will be reported with its paired comparison.

## What I expect

These are estimates made before any measurement.

- A clear rise in the single-shot compile rate with no prompt is likely. The model will have seen
  about ten times more AILANG, covering far more of the language.
- Reaching 16 of 23 on the agent eval in the first full run is about even odds. The language gate
  could pass and the agent gate still fail, which is the case Stage 4 covers.

## Time and cost

| Stage | Work | Time | GPU |
|---|---|---|---|
| 0 | Build the single-shot test, run three baseline arms | 1 day | about 6 hours |
| 1 | Descriptions, splitting, repairs, de-duplication, checks | 3 days | none |
| 2 | Small run, two adapter sizes | 1 day | 6 to 10 hours |
| 3 | Full run, both gates | 3 to 4 days | 2 to 3 days |
| 4 | Only if needed | 2 days | 1 day |

The GPU time for Stage 3 is scaled from pilot-4, which trained on 5.6 million tokens in 12 hours
20 minutes. A 30-minute timing run at the start of Stage 2 will replace that estimate.

Money cost is zero: descriptions use the Claude subscription and training runs on the rig. The real
costs are subscription quota for 3,000 to 4,000 description requests and the rig's GPU for about
three days, during which other local evals wait.

## Log

| Date | Step | Result |
|---|---|---|
| 2026-11-16 | Single-shot test built | It is the existing standard mode of `ailang eval-suite` with self-repair off, run from the frozen eval folder with the pinned compiler (`tools/finetune/pilot5_baseline.sh`). No new harness was needed |
| 2026-11-16 | Time per sample measured | One sample takes 1.5 to 2 minutes, because the model thinks for 3,000 to 6,000 tokens before it answers. The plan's "about an hour" for the baseline was wrong. Three samples per task over three arms is about 6 hours, so the test uses three samples, not five |
| 2026-11-16 | Baseline started | Untrained Gemma 4 12B, 24 held-out tasks, three samples each, with no prompt, the slim prompt and the full prompt |
| 2026-11-16 | Near-duplicates grouped | The 2,733 repository programs fall into 2,076 groups of similar code (half or more of their five-token sequences shared). 1,028 programs sit in a group with at least one sibling. Splits and caps are made by group (`tools/finetune/neardup.py`) |
| 2026-11-16 | Test programs set aside | 150 groups of short programs with a `main`, chosen at random with a fixed seed, are held back from training before any training data is built |
| 2026-11-16 | **Single-shot baseline, untrained model** | No prompt: 0 of 50 samples pass and 2 compile (4%). Slim prompt: 2 of 71 pass (3%), 11 compile (15%). Full prompt: 28 of 72 pass (39%), 32 compile (44%), and 14 of the 24 tasks pass at least once. The model knows almost no AILANG on its own, and the full prompt takes it from 0% to 39%. That range is what training has to cover; the language gate is 39% |
| 2026-11-16 | What the baseline says about failures | With the full prompt, 28 of the 32 programs that compiled also gave the right output. Nearly every failure is a program that does not compile, which is the failure this pilot trains against |
| 2026-11-16 | Open defect in the test | The no-prompt arm banked 50 samples, not 72: for some tasks the second or third sample was skipped. The cause is not yet found. The arm's result does not depend on it (no sample passed), but it must be fixed before a fine-tune is scored |

---
sidebar_position: 7
title: Pilot-3 (learning from its own mistakes)
description: The third AILANG fine-tune, in progress. It trains on the pilot-2 model's own failing attempts, repaired and verified, and on the habit of looking things up after a fix fails.
reviewed: 2026-11-09
reviewBy: 2027-02-09
---

# Pilot-3: learning from its own mistakes

**Status: in progress.** This page is written as the work happens, so it records each step
and why we took it, not only the result.

## Where pilot-2 left us

[Pilot-2](./pilot-2.md) fixed the agent protocol, but it failed the held-out comparison. Base
Gemma 4 12B passed 6 of 13; the fine-tune passed 4 of 24. The banked runs showed why:

- **Most failures were syntax errors.** Two thirds of the fine-tune's failing runs ended in a
  parse error, almost all of them syntax borrowed from other languages: `->` match arms, `as`
  casts, a missing `let`.
- **When a fix failed, it retried blindly.** Failing runs went through 40–80 write → check cycles
  without reading the docs or examples.
- **The training data never showed either situation.** Only 26 of its 1,671 repair episodes fixed a
  parse error. No episode showed a fix failing and the model looking something up afterwards.

The repair episodes came from *staleness pairs*: programs that once compiled and broke when the
language changed. Those pairs teach real mistakes, but not the mistakes **this model** makes.

## The idea: on-policy repair data

*On-policy* data comes from the model being trained, not from some other source. The steps:

1. **Collect.** Run the pilot-2 model as an agent on the **training** benchmarks, through a
   logging proxy that records every request and response. Every file version it writes is a
   real example of its own mistakes.
2. **Re-check.** Replay the recorded writes and edits to rebuild each file version, then run
   `ailang check` and `ailang run` on each one ourselves. We don't trust what the transcript says
   the tools returned.
3. **Repair.** A stronger model (Claude Opus, through the Claude Code subscription,
   [D-FT-7](./decisions.md)) fixes each failing version. It gets the task, the expected output
   and the AILANG agent reference, and loops on compiler feedback. A fix is kept only if it
   compiles **and** prints exactly the expected output.
4. **Build episodes.** Each training episode starts from the model's own broken program and the
   real error, then shows the verified fix. When the model's own first fix failed, the episode
   shows the habit we want: look up the syntax, then fix (see below).
5. **Mask the mistakes.** The model's broken writes stay in the episode as context, but they
   carry **no training loss**. The model learns *what to do after* a mistake, not to repeat it.

### The new habit: read the reference after a failed fix

Which lookup should the episodes teach? We tried three options against real errors:

- **`ailang examples search`** returns only example *names*, so it would take a second step to
  see any code.
- **`ailang examples show <name>`** shows code, but no single example covers each construct
  cleanly.
- **A section of the teaching prompt** works best. It is authoritative, it already has
  "wrong → right" tables, and the error message tells you which section to read. For example:

  ```bash
  ailang prompt | grep -A45 "^## Pattern Matching"
  ```

So the episodes map error shapes to sections:

| Error | Section read |
|---|---|
| `PAR_MATCH_ARROW` (`->` in a match arm) | Pattern Matching |
| `PAR015` (no `let`), `PAR017` (`;` in an expression body), a missing `{` or `else` | Let Bindings: Block Style vs Expression Style |
| bad lambda parameters | Syntax Reference |
| broken `${...}` | String Interpolation |
| `as` casts, float division on ints | Number Conversions |
| `++` on strings | String and List Concatenation |
| any other parse error | What AILANG Does NOT Have |

A broken file often has several errors. The lookup is tied to one specific error line, and the
thought before it quotes that line. That way the model learns "this error → this section",
not just "look something up".

Thoughts remain **built from facts**, as in pilot-2: the real error line and the fix it names.
Here is an episode for one of the fine-tune's real fizzbuzz attempts:

| Loss | Action | Thought |
|---|---|---|
| ✓ | read the template | *The task names the solution file; I'll read it first…* |
| — | write its own broken program | (the model's mistake: context only) |
| ✓ | `ailang check` | *Type-check it before running.* |
| ✓ | `ailang prompt \| grep -A42 "^## Let Bindings…"` | *The checker says: PAR017 … ';' is only valid inside `{ ... }` block bodies… Check the "Let Bindings" section before fixing it.* |
| ✓ | write the fix | *The reference shows the correct form. So I'll use one style: `let x = e in ...` in an expression body, or a `{ ...; ... }` block body…* |
| ✓ | `ailang run` | *Run it and compare the output with the expected output exactly.* |

Every syntax claim in these thoughts was checked with `ailang check` before use.

### How "no loss" is implemented

A broken step gets `"weight": 0` on its assistant message. The trainer
(`tools/finetune/train2.py`) skips those messages when it builds the loss mask. Two checks:

- **The key is invisible to the model.** The Gemma 4 chat template ignores it; the rendered text
  is byte-identical with and without it.
- **The mask shrinks as expected.** Marking two steps as no-loss in a pilot-2 sample left the
  tokens unchanged and cut the trained tokens from 775 to 586.

### Guardrails

- **Held-out tasks are never used.** The collection script and the extractor both refuse a
  held-out benchmark ([data spec](./data.md)), so the before/after comparison stays clean.
- **Every tool output in an episode is real.** It comes from running the current `ailang` on that
  exact file.
- **Runaway runs are cut short.** A token budget per benchmark (1.5M) stops the 400-turn loops
  pilot-2 fell into.
- **Long episodes are dropped.** The training mix leaves out episodes over 24 steps, so the model
  never learns that long loops are normal.

## What we learned along the way

### A repair model with no tools still wants to test its code

The repairer runs as `claude -p` with every tool disabled. Our own loop compiles and runs each
fix, then sends any new error back. On the first test, Opus didn't answer for 15 minutes.
Instead it kept writing `<invoke name="Bash">…` blocks as plain text, trying to run its fix
before replying. Two changes fixed this:

- The prompt now says so outright: *you have no tools… the program will be compiled and run for
  you, and any new error sent back*.
- The repairer runs at `--effort medium`. Hard thinking adds little when a compiler checks every
  answer anyway.

The next run took 7.5 s and the fix compiled and printed exactly the expected output in one
round.

### Not every failed tool call changes the file

The first live session showed the fine-tune often sends an `edit` that pi rejects ("Could not
find the exact text"). The extractor first treated that as losing track of the file and stopped.
Now it skips edits pi rejected. On one benchmark that raised the recovered file versions from 1
to 17, including 9 cases where the model's own fix failed with the same kind of error. Those are
exactly the moments habit 5 is about.

### The eval's tool output wasn't the tool output we trained on

Every `ailang check` and `ailang run` result in an agent eval started with a log line from a
deprecated Google Cloud tracing library. The harness sets `GOOGLE_CLOUD_PROJECT` for agent
children, and any `ailang` command run with that variable printed the notice. Training episodes
never had it, so evaluation showed the model slightly different output from training. It also
affected every agent eval on every model, not only the fine-tune.

- **Fixed** in `d76958190` ([#2151](https://github.com/sunholo-data/ailang/issues/2151)). The
  deprecation line is filtered from the global logger.
- **Before pilot-3's evals**, the fixed binary gets installed.
- **The base arm is re-run from scratch.** Pilot-2's base rows saw the notice, so mixing them with
  clean runs would confound the comparison.

### A habit we did not expect: inventing module names

The second most common error was `MOD010`: the model starts its file with a made-up module name (`module rle/decode`, `module fizzbuzz/main`) instead of the one the file path dictates (`module benchmark/solution`). This is a habit carried over from other languages, where a file can be named anything. Each of the 46 items now shows the error, then the one-line fix, which is cheap to learn and was costing whole runs.

### Long samples lose the part you train on

On-policy episodes are longer than pilot-2's: they carry the model's own wrong file, its failed fix, the reference section it looks up, and then the verified fix. Measured with the training tokenizer, 16 of 806 training samples and 1 validation sample were over the 12,288-token limit, the largest at 45,010 tokens. Pilot-2 had none.

Training **truncates from the end**, so an over-long on-policy sample keeps the no-loss context (its own mistakes) and loses the fix, the only part with a loss. The sample would teach nothing, and its length would cost the most compute of any.

- `build_mix2.py --tokenizer --max-tokens` now measures every sample with the same render function `train2.py` uses, one implementation rather than an estimate.
- Over-long samples are dropped **before** they take a per-kind slot, so another episode of the same benchmark and error kind fills the slot.
- A character-count estimate had flagged 22 samples, not 16; it would have dropped some good ones.

## Training and evaluation plan

`tools/finetune/pilot3_run.sh` runs each stage and can resume from any of them:

| Stage | What | Where |
|---|---|---|
| pairs | trace → failing file versions (≤10 per benchmark and error kind; failed fixes first) | CPU |
| repair | Opus fixes each one; kept only if verified | Claude subscription |
| data | episodes → thoughts → mix (on-policy, plus a smaller replay of pilot-2's kinds; episodes over 24 steps dropped) | CPU |
| train | **warm start from the pilot-2 adapter**: 1 epoch, learning rate 1e-5 (half of pilot-2's) | GPU |
| serve | fuse, strip the audio config, `ollama create gemma4-12b-ailang-ft3` | GPU |
| eval | held-out 24, lean profile, agent mode, **no wall-clock cap** | GPU |

**Why warm-start?** Pilot-2 already learned the protocol and the basic loop; what it lacks is
covered by the new data. Starting from its adapter saves the ~21 hours a retrain from base
would take. The replay sample of pilot-2's data guards against forgetting what it learned.

**Eval arms**, all on the same 24 held-out benchmarks:

| Arm | Model | Prompt | Question it answers |
|---|---|---|---|
| B | pilot-3 | slim | did on-policy data close the gap? (pilot-2: 4/24) |
| D | pilot-3 | **full teaching prompt** | do the habits from training plus the prompt's language knowledge beat base? |
| A | base 12B | full teaching prompt | the reference, now on all 24 |
| C | pilot-3 | none | how much language does the model itself carry? |

## Progress log

| Date | Step | Result |
|---|---|---|
| 2026-11-09 | Diagnosed pilot-2 from its banked rows | Parse errors dominate; repair data had 26 parse-error episodes ([pilot-2 §Diagnosis](./pilot-2.md#diagnosis-2026-11-09)) |
| 2026-11-09 | Built the collection run (`tools/finetune/collect_onpolicy.sh`) | Dry run: 57 training benchmarks, 1 trial, lean pi profile, slim prompt |
| 2026-11-09 | Built the extractor (`tools/finetune/onpolicy_pairs.py`) | Tested on an older trace: rebuilds file versions and re-checks them; refuses held-out tasks |
| 2026-11-09 | Added on-policy repair to `tools/finetune/stale_fix.py` (`--prompt onpolicy`) | First smoke test **timed out after 15 min**: with tools disabled, Opus wrote fake tool calls as text to test its code. With an explicit "you have no tools" and `--effort medium`, it took 7.5 s and the fix passed in one round (see below) |
| 2026-11-09 | Started the on-policy collection (rig idle, lock free) | ~9 min per benchmark, so ~9–10 h; the trace proxy records every request |
| 2026-11-09 | Fixed #2151 part 1: the Cloud Trace notice in every eval tool result | Verified against the old binary as a control |
| 2026-11-09 | Built `pilot3_run.sh` and `pilot3.yaml`; registered the `pi-gemma4-12b-ft3` lane | Waiting for the collection |
| 2026-11-09 | Added the `onpolicy` episode kind (`synth_episodes.py`), habit-5 thoughts (`thoughts.py`), no-loss steps (`build_mix2.py`, `train2.py`) | Rendered a test episode; template and mask checks pass. Thoughts now quote the specific error line, not the "module loading error" wrapper above it (this also improves pilot-2's repair thoughts) |
| 2026-11-09 | Collection finished: 57 benchmarks in 8 h 38 min, 5,059 requests traced | The pilot-2 model passed **30/57** of its own *training* tasks, so it had not mastered even the tasks it trained on. A trial extraction on the first 38 benchmarks found 268 failing file versions, 121 of them failed fixes |
| 2026-11-09 | Extracted repair items from the full trace (`pilot3_run.sh pairs`) | 752 file versions → **349 items** from 44 benchmarks (132 already passed, 179 duplicates, 92 over the per-kind cap). 163 are failed fixes. Top kinds: unexpected-token parse errors 98, `MOD010` 46, wrong output 32, type mismatch 30, undefined variable 30 |
| 2026-11-09 | Opus repaired every item (`pilot3_run.sh repair`, 29 min, 4 workers) | **348/349 verified**: each fix compiles and prints exactly the expected output. 285 passed in one round, 55 in two, 9 in three or four; one wrong-output item failed all four rounds and is dropped. About 0.8 M output tokens on the subscription |
| 2026-11-09 | Built the training mix (`pilot3_run.sh data`, 6 min) | 3,361 episodes → 784 training samples: 297 on-policy (all with no-loss context), 210 solve, 176 repair, 101 canonical examples. 30 validation samples, 8 of them on-policy. 3.54 M tokens. No held-out benchmark appears |
| 2026-11-09 | Caught and fixed over-long samples before training | 17 samples were over the 12,288-token limit and would have lost their fix to truncation; now measured with the training tokenizer and dropped before the per-kind cap (see above) |
| 2026-11-09 | Started training (16:47 UTC): 784 samples, 1 epoch, 196 optimizer updates, warm start from pilot-2 | It waited first on a rig lock our own collection had left behind: the collection script set its own `trap ... EXIT` after taking the lock and so **replaced** the lock's release trap. Fixed to add to it; tested old vs new with a throwaway lock directory |
| 2026-11-10 | Training finished (00:45 UTC, 7 h 58 min) | Validation loss **0.170 → 0.154 → 0.127 → 0.120 → 0.117** at iterations 1, 200, 400, 600, 784, measured on held-back training-distribution samples, 8 of them on-policy. Train loss about 0.09. A falling validation loss says the data was learned; only the held-out evals say whether it transfers |
| 2026-11-10 | Installed the #2151 fix in both binaries the evals use (`~/go/bin/ailang` for the agent's tool calls, `ailang-ft3` for eval-suite) | Checked with `GOOGLE_CLOUD_PROJECT` set: the old binary prints the deprecation notice above the program's output, the new ones print only the output. Both had been built before the fix, so the install was needed |
| 2026-11-10 | Started serve → evals B, D, A, C | Waits its turn for the GPU behind a daneel job |

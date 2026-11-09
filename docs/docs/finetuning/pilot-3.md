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

## Progress log

| Date | Step | Result |
|---|---|---|
| 2026-11-09 | Diagnosed pilot-2 from its banked rows | Parse errors dominate; repair data had 26 parse-error episodes ([pilot-2 §Diagnosis](./pilot-2.md#diagnosis-2026-11-09)) |
| 2026-11-09 | Built the collection run (`tools/finetune/collect_onpolicy.sh`) | Dry run: 57 training benchmarks, 1 trial, lean pi profile, slim prompt |
| 2026-11-09 | Built the extractor (`tools/finetune/onpolicy_pairs.py`) | Tested on an older trace: rebuilds file versions and re-checks them; refuses held-out tasks |
| 2026-11-09 | Added on-policy repair to `tools/finetune/stale_fix.py` (`--prompt onpolicy`) | Smoke test running |
| 2026-11-09 | Added the `onpolicy` episode kind (`synth_episodes.py`), habit-5 thoughts (`thoughts.py`), no-loss steps (`build_mix2.py`, `train2.py`) | Rendered a test episode; template and mask checks pass. Thoughts now quote the specific error line, not the "module loading error" wrapper above it (this also improves pilot-2's repair thoughts) |

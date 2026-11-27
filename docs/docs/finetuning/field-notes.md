---
sidebar_position: 2
title: Field notes (what fine-tuning is like day to day)
description: Lessons from running AILANG fine-tunes, each drawn from something that actually happened - where the time goes, why the score lies, and the habits that kept the work honest.
reviewed: 2026-11-26
reviewBy: 2027-02-26
tier: internal
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

## 11. Run the no-training control first

For three pilots we compared each fine-tune with the one before it, and with the base model on the
full teaching prompt. We never ran the base model on the slim prompt the fine-tunes used. When we
did, it passed 9 of 23 held-out tasks, level with our best fine-tune at 10. Three rounds of data
collection and training had added nothing we could measure. That control costs
one day of GPU and no training, and it belonged before pilot-1.

## 12. A quick check can measure the wrong thing

Pilot-4 passed a 30-minute behaviour check before its evals: only 1 response in 1,075 was a dropped
tool call. It then scored 5 of 24, with 13 runs ended by a dropped call. The check counted drops per
response, on training tasks. What ends a run is a drop per run, and pilot-4's runs were about 100
responses long, so a 0.5% chance per response became roughly 40% per run. Check the thing that
fails, in the unit it fails in.

## 13. Freeze every input of a long eval

An eval arm takes most of a day, and the repository kept moving under it. Midway through one arm a
merge renamed standard-library functions, the checkout updated itself, and scoring began loading
library files the installed compiler could not parse. Pinning the binary had not been enough: the
library and the benchmark files were read from the live checkout. The arms now run from a
directory holding a snapshot of both.

## 14. A negative result is a result

The stop rule was written before the first pilot: a fine-tune with the slim prompt must match the
base model with the full prompt. It failed every time, and the control in note 11 removed the reason
to expect another round to pass it, so the line was stopped. What it bought is specific: the gap is
language knowledge the full prompt carries, worth 7 tasks, and most of the untrained model's
failures are single runaway responses that the harness could cut off.

## 15. Separate "did not learn" from "wrong task shape"

Pilot-5's first run trained on 1,504 verified programs from our repositories and scored 0% and 6% on
the held-out benchmarks ([pilot-5](./pilot-5.md)). That looked like a capacity ceiling until an
in-distribution check on 108 held-back repository programs showed compile rates jumping from 0% and
12.5% untrained to 34% and 39%. Repository libraries and `module benchmark/solution` stdin/stdout
programs are different genres. Adding 491 training-benchmark solutions in Run c lifted slim-prompt
benchmark compilation to 44% — matching the untrained model with the full teaching prompt.

## 16. Training without reasoning teaches a thinking model not to think

Run c matched the full-prompt base model's 44% compile rate on held-out benchmarks, yet passed only
17% against 39% ([pilot-5](./pilot-5.md)). Why? When the untrained model compiles a benchmark, 88%
of those programs also produce the right output, because it spends 3,000 to 6,000 thinking tokens
working out the algorithm first. Run c's examples had no `reasoning` field while rendering under
`enable_thinking=True`, which trained an empty `<|channel>thought\n<channel|>` block as part of the
target. The fine-tune learned to skip thinking and emit code cold in ~430 tokens — getting the
syntax right and the algorithm wrong.

## 17. Check library scaling conventions, and supervise unattended guards

Raising LoRA rank from 32 to 128 in Pilot-5 Run b diverged at iteration 160 (`mlx-lm` multiplies by
`scale` directly without dividing by `rank`, so keeping `scale=10` quadrupled the effective update)
and sat at loss 7 for six hours. We fixed `scale × rank = 320` and added an automatic loss-above-3
stop guard — which rightly killed Run b2 at iteration 980, after which **nothing ran for five and a
half days** because the watcher process had been reaped under memory pressure. A guard that stops a
run cleanly still needs an alert when no human is looking.

## 18. Decode the unmasked token IDs: chat templates can silently strip your targets

To fix note 16, Pilot-5 Run d generated 2,907 planning scratchpads (median 253 words) and added a
mix-time gate requiring at least 50 words of `reasoning` on every single-turn sample. Every JSONL row
had a rich thought — yet after 22 hours of GPU training, Run d still answered single-shot benchmarks
in ~423 tokens with zero thinking ([pilot-5](./pilot-5.md)).

Decoding the exact token IDs produced by `tokenizer.apply_chat_template(..., enable_thinking=True)`
showed why. Gemma 4's stock HuggingFace Jinja template is written to prune thoughts from past turns
at inference time, and guards the thought block with:

```text
if thinking_text and loop.index0 > ns_turn.last_user_idx and message.get('tool_calls')
```

Whenever an assistant turn has no `tool_calls` — every single-turn program, every single-turn
repair, and every agent episode's final reply — Jinja silently threw `message['reasoning']` away
(and `strip_thinking()` removed `<|channel>...<channel|>` from `content` too). Checking the input
JSON proved only what we *sent* to the template, not what reached the loss function. `train2.py` now
patches that template guard in memory and decodes the unmasked target slice (`mask == 1`) of every
training turn, raising an error if `<|channel>thought\n...\n<channel|>` is missing.

## 19. Aim a small model at its real tier (Core vs Stretch), and watch thought style

A headline pass rate across mixed tiers hides what a 12B model is actually good at. Even with the
full 26k-token teaching prompt, untrained Gemma 4 12B passes **12 of 12 (`100%`) Core** benchmarks in
agent mode, and **5 of 12 (`42%`) Stretch** competitive-programming puzzles ([pilot-1](./pilot-1.md)).
Fine-tuning 12B is therefore not about turning it into a frontier puzzle solver on Stretch; it is
about removing the 26k prompt tax (which collapses 4-stream batching from `2.77×` to `1.26×`) so
parallel $0 local workers can handle routine `Core`-grade engineering — module edits, inline `tests`,
compiler-guided refactors, and CLI/package maintenance — while escalating hard algorithmic design.

Thought *shape* matters too. Teacher-distilled plans (`200–450` words) are clean and 6× shorter than
the base model's rambling `3,000–6,000`-token scratchpads, which helps speed and routine tasks. If
harder multi-branch tasks turn out to need trial-and-error self-correction inside `<|channel>thought`
rather than a neat forward plan, the next lever is rejection-sampled self-thinking (STaR): sample the
fine-tune several times on training tasks, keep only runs that pass `ailang run`, and train on the
model's own winning `<|channel>thought` traces.

## 20. Short dropped tool calls end agent loops silently (don't only look above 500 tokens)

Our dropped-call detector (`tools/finetune/trace_dropped.py`, note 13) defaulted to `--min-tokens 500`
because Pilot-3's failures were runaway `write` calls. On Pilot-5 Run d's early agent gate, that
threshold reported only 1 dropped call in 111 turns (`0.9%`), yet all 4 completed benchmarks had
exited early with broken code still on disk.

Inspecting the raw SSE chunks showed `completion_tokens` of `52`, `114`, `154`, and `907` on the
final turn of those 4 runs, with `content: ""`, `tool_calls: []`, and `finish_reason: "stop"`. Short
malformed `bash`, `read`, and `edit` calls were being dropped by Ollama's tool parser just like long
`write` calls — and a single empty `stop` turn makes `pi` treat the task as finished. Re-scoring at
`--min-tokens 25` showed the real damage across runs:

- **Untrained 12B (`A`):** `2/397` (`0.50%` per turn) → **2 of 24** runs ended on a drop.
- **Pilot-4 (`A2cap`):** `6/546` (`1.10%` per turn) → **6 of 24 (`25%`)** runs ended on a drop.
- **Pilot-5 Run d:** `4/114` (`3.51%` per turn, after training 83% of steps with `tools=None`) →
  **4 of 4 (`100%`)** finished runs ended on a drop.

Even a `3.5%` per-turn parser drop rate kills `1 - 0.965^25 ≈ 59%` of 25-turn episodes. Run e
logs both `>=500` and `>=25` token drops, raises `--agent-cap` to 700, and renders 55% of compiler
repairs as lean-`pi` `check → thought → write(fixed)` tool calls (`--agent-repair-frac 0.55`) so
tool-call formatting stays sharp alongside single-turn language training.



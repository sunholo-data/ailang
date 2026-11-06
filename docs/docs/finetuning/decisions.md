---
sidebar_position: 2
title: Decision log
description: Dated decisions for the AILANG fine-tuning programme — who decided, what, and why.
reviewed: 2026-11-06
reviewBy: 2027-02-06
---

# Decision log

Newest last. Each entry gives the date, who decided, the decision and the evidence behind it.
"Mark" is the project lead; "measured" means the decision follows directly from a measurement
on the [Baseline](./baseline.md) or [Pilot-1](./pilot-1.md) page.

| ID | Date | By | Decision | Why |
|---|---|---|---|---|
| R1 | 2026-11-05 | analysis | **Shrink the teaching prompt, don't remove it** | 84 release tags in 90 days. A prompt-free fine-tune freezes one snapshot of the language and writes the old syntax with confidence. Superseded in spirit by D-FT-4: habits plus runtime discovery are the correction mechanism |
| R3 | 2026-11-05 | measured | **Drop the "nothing under 13B" rule and measure instead** | The rule had no source. The size ladder showed the gap is language knowledge (71–89% of small-model failures are compile errors), which fine-tuning addresses |
| D-FT-0 | 2026-11-05 | Mark + measured | **Target Gemma 4 12B dense**; 31B as the quality arm; qwen3.8-27b as a teacher only | 12B: 54% one-shot (excluding harness timeouts), and 12 of its 13 agent failures were tool-protocol misses, not wrong code. Gemma 4 is Mark's preferred family; Apache 2.0 |
| D-FT-1 | 2026-11-05 | Mark | **Frontier outputs may train the model** (Claude and OpenAI code and trajectories) | Our repos hold ~225k lines of frontier-authored AILANG and thousands of verified agent trajectories. Generating data with a local 27B would be slower and worse |
| D-FT-K | 2026-11-05 | analysis | **Kill criterion:** fine-tune + slim prompt must at least match base + full prompt on the held-out 24 | Keeps the experiment honest; a negative result is recorded, not buried |
| D-FT-H | 2026-11-05 | analysis | **Hold out 24 benchmarks before extracting any data** | Frontier eval results are solutions to the scored benchmarks. Contamination is checked by exact and normalised-code hashes, plus benchmark-id matching |
| D-FT-3 | 2026-11-06 | Mark | **Agent mode is the target; standard one-shot is a prompt problem** | The model's job is fleet agent work. One-shot results inform, but do not decide |
| D-FT-4 | 2026-11-06 | Mark | **Train habits, memorise only the core** | "The whole point of the language has been to help AI self discover." Runtime discovery (check → read error → fix, `docs`, `examples`) also counters staleness |
| D-FT-5 | 2026-11-06 | Mark | **Expand `examples/` into a representative training set** | Measured: 25 of ~397 passing examples reached pilot-1's data pool, and 1 reached the mix. 96 of 496 exported stdlib functions appear in no example. The examples are needed for the docs anyway |
| D-FT-2 | 2026-11-06 | Mark | **Publish fine-tunes outside the rig**, through an artifact bucket linked from this site | Weekend fine-tunes per language release, versioned by AILANG version. Base-model licence (Apache 2.0) to be confirmed against the model card at publish time |
| D-FT-6 | 2026-11-06 | Mark | **Agent evals run without the attended-session gate; re-run pilot-1's three agent arms** | A wire capture against a scripted model showed the pi session-protocol gate refused every eval write until the model called `session_protocol_ack`. Every Phase 0 base-12B pass acked first; the fine-tune never did, so its 0/24 was the harness, not the model. Fixed in `5dc3ba42b`; agent results before it sit on the other side of a baseline boundary |
| D-FT-7 | 2026-11-06 | Mark | **Generate training data with the best models available** (frontier: Opus via the Claude CLI subscription, Codex), not the cheapest | "To generate data we can use good models, so we can present best data for fine tuning." Verification keeps generated data correct whichever model writes it, but a weaker generator skews the kept set towards easy cases: gemma-4-31b fixed 49% of stale programs and failed mostly on the harder ones. Cheap models stay for bulk mechanical work only |
| D-FT-8 | 2026-11-06 | Mark + measured | **Keep thinking ON: train every step as thought → action, in the format Ollama serves; thoughts are built from facts in each episode, and the agent habits are simulated with real tool output** | A live transcript showed pilot-1 emitting correct tool calls inside the thinking channel. It was trained in thinking-off format (an empty thought pre-filled), Ollama serves thinking-on, and the parser intermittently filed the call as reasoning. Switching thinking off at serve time was rejected: it costs reasoning on hard tasks, and the parse failure was random either way. Thoughts state the real error, diff, imports and task; richer reasoning comes from the base model's own verified thinking (self-distillation) |

## Open

- **Weekly cadence and promotion gate.** Proposal: a new fine-tune must match or beat the previous
  one, and base + full prompt, on the held-out agent set before it is published.
- **Staleness measurement.** Run the previous fine-tune against the new language each release; slow
  decay is the evidence that the habits work.

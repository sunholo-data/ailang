# Evaluator route/model probe — a negative result, and what it redirected to

**Date:** 2026-09-08 (attended). **Cost:** ~13k metered tokens across two OpenRouter arms;
the third arm is flat-rate ollama. **No lineage was spent and no frozen state was touched.**

## Question

The reliability canary's evaluator failed 3/3, always on `pi-or-deepseek-v4-flash`
(`deepseek-v4-flash-0731:floor`). `:floor` is OpenRouter's `provider.sort=price` and was
DROPPED from the mission EXECUTOR lane on 2026-08-18 because its two cheapest hosts carry
negative health status. Was the evaluator failing for the same reason — or is it the model?

Established first, from `mission_attempts` in the canary runtime DB:

| work item | stage | state |
|---|---|---|
| `docs-canary-work-item-guide-1` | executor | **execution_completed** |
| `docs-canary-work-item-guide-1` | evaluator | execution_failed |
| `docs-canary-guide-review-2` | evaluator | execution_failed |
| `docs-canary-guide-review-3` | evaluator | execution_failed |

The executor (`claude-sonnet-5`) completed. Only the evaluator failed. The work item's
`reason_code` prose says "executor did not finish successfully", which is generic stage
wording and NOT the failing stage — misreading it as literal cost one wrong turn here.

## Design

Three arms against a miniature of the real contract: a candidate file, a bound deterministic
validator the agent is told to run, and a strict `stage-result.json` to emit, with explicit
instructions not to explore.

| Arm | Route | Model | Isolates |
|---|---|---|---|
| floor | `deepseek-v4-flash-0731:floor` | deepseek | baseline (known-failing) |
| bare | `deepseek-v4-flash-0731` | deepseek | route |
| glm | `ollama/glm-5.3-flash:cloud` | glm-5.3-flash | model (also changes provider) |

## Result — the probe did not discriminate

```
floor  turns=4 tools=3 tokens=6,569  pass   evidence quotes the validator
bare   turns=4 tools=3 tokens=6,579  pass   evidence quotes the validator
glm    turns=4 tools=3 tokens=2,706  pass   evidence quotes the validator
```

**The control passed.** `:floor` is the known-failing condition and it converged in four
turns with a flawless result. When the baseline does not reproduce the failure, the other
arms say nothing about it. This is a negative result about the PROBE — it does not clear
either the route or the model.

## What it did establish

| | turns | tools | tokens | outcome |
|---|---|---|---|---|
| probe (all three arms) | 4 | 3 | ~6.5k | pass |
| real evaluator | 19 | 26 | 100,024 | thrash_aborted |

A 15x token gap on the same shape of task, with three different lanes behaving identically.
That points at the task and its environment rather than the lane.

## The hypothesis it redirected to — UNTESTED

The frozen evaluator instructions say:

> "Read **CLAUDE.md** first. … Read exact candidate diff against `ad1bf98d3…` and **the
> production spec/authority/CLI code**."

In this repository `CLAUDE.md` pulls in path-scoped rules files, and "the production CLI
code" is unbounded. So the contract instructs the evaluator to explore a large repository
while M1 was making the PACKET focused: the packet was bounded and the instructions were not.
The trial's own observation — the agent "repeatedly list[ed] directories, inspect[ed] Git
status/log/HEAD … with different command strings" — is what that instruction invites.

If correct, this explains why a route swap was never going to help, and why the previous
session's "this trial does not support another token-cap increase" was right for a cause
nobody had isolated.

**Discriminating test (cheap, no lineage):** re-run the SAME real contract with the
exploration mandate removed — drop "read CLAUDE.md first" and "the production spec/authority/
CLI code", keep every criterion, the packet and the bound validator. Convergence would
implicate the contract. Not run; awaiting authorization.

## Reusable assets

`internal/modelreg` gained `pi-or-deepseek-v4-flash-bare`, an A/B control identical to the
`:floor` row but for the suffix. It could not be used on the existing lineage: `--evaluator`
resolves against the work item's FROZEN model snapshot (`retry_review.go:255`), not the live
registry — the immutability guarantee working as designed. Within that snapshot every bare-id
deepseek option also changes the harness, so a single-variable route test on that lineage is
impossible without re-freezing.

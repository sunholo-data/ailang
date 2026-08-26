---
title: The Mission Model Fleet
description: Which model runs which role in an autonomous mission loop, why, and how each role survives a provider outage or an exhausted quota
---

# The mission model fleet

An [autonomous mission loop](./mission-bootstrap.md) is not one model doing everything. It is five
distinct roles, each with different demands, each pinned to a different model — and each with a
fallback chain that crosses providers so a single outage or spent quota cannot wedge the loop.

This page documents the fleet that runs AILANG's own missions: what each role does, which model is
assigned to it, and — most importantly — **how those assignments were decided**. The specific model
names will age. The method should not.

## The five roles

| Role | Work it does | Dominant demand | Volume |
|---|---|---|---|
| **Controller** | Triage the inbox, pick the top backlog item, record the outcome, run the retro. Spawns every other role. | Long-context judgement | Low output, high input |
| **Designer** | Author a design document *as a file*, verify its premises with real commands, survive adversarial review | Deep reasoning + skeptical self-check | Low |
| **Planner** | Turn the design into milestones and a day-by-day plan | Decomposition | Low |
| **Executor** | Implement the plan: edit files, run tests, iterate for up to a hundred steps | Tool-call stamina — *not derailing* | **High** |
| **Evaluator** | Score the implementation against a rubric, re-run the tests, verify each acceptance criterion | Judgement + verification | Medium |

A sixth role sits outside the loop: the **design quorum**, a panel of independent reviewers that
must pass a design before any implementation begins.

## The principle: spend where decisions are made

Designer, planner and quorum are all **low-volume and high-leverage**. One run per iteration, and a
mistake there propagates through everything downstream — a bad premise sends the executor to build
the wrong thing correctly. That is where an expensive model pays for itself.

The executor is the inverse: **high-volume, and what matters is not derailing over a long agentic
loop.** Raw reasoning is less decisive there than reliable tool-calling. A cheaper, steadier model
is the better fit, and it is cheaper precisely where the volume is.

## Different roles need different evidence

This is the part most easily got wrong. Two instruments measure different things, and using the
wrong one misroutes a model:

- **Harness benchmarks** — a model's pass rate driving *your own* agent loop. If your benchmarks
  are easier than the public suites, a high score does not mean "smart", it means **"drives the
  loop without derailing"**. That is exactly the *executor* demand.
- **Public benchmarks** (SWE-bench, Terminal-Bench, and similar) measure raw capability. That is
  the right lens for *designer, planner and evaluator*.

A model can top one and be mediocre on the other. Score the role, not the model.

### A third number that decides more than it looks

**Output headroom** — the model's maximum completion length — is a first-class selection criterion
for any reasoning model, not a footnote.

We learned this expensively. A model recorded a mediocre score in our own evaluations for weeks,
and the verdict was later retracted: the harness had been truncating its *thinking* tokens. The
model was fine; the budget was not. It turned out to be one of the strongest open-weight coding
models available.

The lesson generalises: **a starved thinking budget reads as a capability failure.** When a
reasoning model underperforms, check headroom before concluding anything about the model. And when
assigning a role, prefer generous headroom for anything that must reason over a large context.

## The fleet

*Current as of August 2026. Model names age fast — the chain **shapes** are the durable part.*

| Role | Chain (primary → fallbacks) | Vendors |
|---|---|---|
| Controller | `opus-5` → `fable-5` → `gpt-5.6-sol` | Anthropic, OpenAI |
| Designer | `fable-5` ↔ `kimi-k3` (rotating) | Anthropic, Moonshot |
| Planner | `gpt-5.6-sol` → `kimi-k3` *(flat)* → `kimi-k3` *(metered)* → `opus` | OpenAI, Moonshot, Anthropic |
| Executor | `gpt-5.6-sol` → `deepseek-v4-flash` *(flat)* → *(metered)* → `opus` | OpenAI, DeepSeek, Anthropic |
| Evaluator | `sonnet` → `minimax-m3` *(flat)* → `minimax-m3` *(metered)* | Anthropic, MiniMax |
| Quorum | `gpt-5.6-sol` + `gemini-3-1-pro` + `glm-5.2`, in parallel | OpenAI, Google, Z-AI |

Every role names **at least two vendors**. That is a hard rule, not an aspiration: a role reliant
on one provider is a role that stops when that provider does.

### Route fallback vs model fallback

Notice the planner, executor and evaluator each have **two consecutive rungs of the same model**.
That is deliberate.

Those models are reachable two ways: a flat-rate subscription route and a metered pay-per-token
route. The subscription is cheaper but its quota has an **undisclosed limit** — the usage API
reports consumption but never the denominator, so exhaustion cannot be predicted, only survived.

Putting the metered twin directly behind the flat-rate lane means running out of quota degrades the
**route**, not the **model**. The loop keeps the same capability and only changes how it is billed.
Only when both routes are gone does the chain fall back to a different model.

```
codex → ollama/deepseek-v4-flash → openrouter/deepseek-v4-flash → opus
        └── flat-rate ──────────┘  └── metered ───────────────┘  └ different model
```

## Two independence rules

Provider diversity is about *availability*. These two rules are about *correctness* — and both are
enforced, not merely documented.

**Generator ≠ judge.** The evaluator must not share a vendor with the executor. Not just a
different model — a different **vendor**. Two models from the same lab and generation are different
models, but they plausibly share a systematic blind spot, and a judge that cannot see what the
generator missed is not a judge.

> We got this wrong in review and were caught by our own test. The first pick had the best headroom
> and benchmark scores, but was the same vendor and generation as the executor. The test failed, and
> it was right to.

**Designer ≠ reviewer.** A model that authors a design must never be one of the reviewers judging
it. This sounds obvious and is easy to violate by accident: a rotation entry and a reviewer default
can name the same model without anyone noticing, and it is worst on a revision pass — where the
objection being answered is the reviewer's own.

The failure mode here is instructive. Our authoring rotation nominally had three entries, but one
could not write files at all (it ran in a server-side sandbox, so its edits never reached the
working tree) and another *was* one of the reviewers. That left **one usable lane** — so any design
needing a revision exceeded its model's budget by construction, not by carelessness. Nobody noticed
for months, because the rotation *looked* three deep.

**Audit the effective chain, not the declared one.** An entry that cannot perform the role is not a
fallback.

## Failure handling

- **Probe before use.** Each lane is checked with a one-token request before the iteration commits
  to it. A lane that cannot answer is skipped and the chain advances.
- **Degrade loudly.** When a reviewer is unreachable, the quorum drops to N−1 and **names the absent
  reviewer and why**. It never silently passes with fewer votes than it claims.
- **A probe is not proof.** A cheap probe can return success against an exhausted quota — the
  failure appears only on the real run. Treat probe success as necessary, not sufficient, and make
  the in-iteration path handle exhaustion too.
- **Beware the silent no-op.** Our planner accepted only one provider prefix; any other value fell
  through to a default. A pin naming a different lane would have *read as applied in the logs while
  something else ran*. If you add a lane, test that a pin actually reaches it.

## Applying this to your own loop

1. **List your roles and their real demands.** Volume and failure mode matter more than a
   leaderboard position.
2. **Spend on the upfront decisions.** Design and planning are low-volume; a better model there is
   cheap in absolute terms and compounds downstream.
3. **Pick the right instrument per role.** Harness pass rates for the executor; public benchmarks
   for the reasoning roles; headroom for anything that thinks at length.
4. **Give every role two vendors.** Then check the chain is *effective* — every entry must actually
   be able to do the job.
5. **Enforce independence in a test.** Generator ≠ judge and designer ≠ reviewer are properties a
   test can assert. Ours caught a real mistake; taste alone would not have.

## See also

- [Bootstrapping a Mission Loop](./mission-bootstrap.md) — pointing a loop at your own repository
- [Agent Workflows](./agent-workflows.mdx) — the design → plan → execute → evaluate inner loop
- [Evaluation](./evaluation/README.mdx) — how the harness benchmarks referenced above are produced

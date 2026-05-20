---
title: Three Camps Self-Audit
sidebar_label: Three Camps Self-Audit
description: AILANG's own results on 14 gap benchmarks that probe the three AI-native-language camps' core hypotheses.
---

# Three Camps Self-Audit

The companion [Three Camps Comparison](./three-camps-comparison) page surveys 16 AI-native languages and identifies ~14 **gap benchmarks** — each one a testable hypothesis about why a particular camp exists. This page is AILANG running against those benchmarks. Honest scoreboard. The failures are the most informative part.

:::tip Methodology in one sentence
We wrote 14 benchmarks designed to probe each camp's core claim, then ran AILANG and Python (baseline) under the same teaching-prompt eval methodology AILANG already uses on itself.
:::

## Setup

- **Run date**: 2026-05-20
- **Model**: `claude-haiku-4-5` (cheap-model floor; multi-model run deferred)
- **Languages**: AILANG, Python (baseline)
- **Benchmarks**: 11 of 14 (3 vision-tier benchmarks requiring real `std/ai` provider deferred to mainline eval)
- **Self-repair**: enabled (LLM gets one retry on compile errors)
- **Duration**: 11s wall-clock for 22 runs
- **Cost**: under $0.50

The 3 vision-tier benchmarks deferred from this run are `multi_agent_handoff`, `ai_effect_summarize`, `ai_effect_json_schema` — all require a real AI provider configured. Their type-level structure is verified but execution is left to the next mainline eval cycle.

## Aggregate Results

| Language | Pass Rate | One-shot | Self-repair pass | Fail |
|----------|-----------|----------|-------------------|------|
| AILANG | **8/11 (73%)** | 5 | 3 | 3 |
| Python | **10/11 (91%)** | 9 | 1 | 1 |

"One-shot" = passed without using the self-repair retry. "Self-repair pass" = first attempt produced a compile error but the LLM corrected it on retry (~2× token cost).

## The Hypothesis Map

For each gap benchmark, the table below records:
- **Pass/Fail** for AILANG and Python
- The camp's underlying claim
- What the result tells us about that claim

| Benchmark | AILANG | Python | Camp's claim | Result |
|-----------|--------|--------|--------------|--------|
| `dense_operator_program` | ✅ | ✅ | **NERD**: ambiguous operators hurt LLM pass rate | **Refuted (locally).** Both languages pass operator-heavy code one-shot. Tokenizer ambiguity is not the bottleneck for this model on this task. |
| `shadowing_heavy_contract` | ✅ | ✅ | **Vera**: named identifiers break under shadowing | **Refuted (locally).** AILANG's HM type inference resolves shadowed names correctly. Vera's De Bruijn slot-ref machinery is not necessary for this case. |
| `explicit_dataflow_ssa` | ✅ | ✅ | **Magpie**: SSA-shaped code is easier to reason about | **Neutral.** Both pass; we don't yet have a non-SSA counterpart to measure the delta. |
| `canonical_convergence` | ✅ (after self-repair) | ✅ | **Zero**: one canonical form helps LLMs converge | **Weak signal.** AILANG passed but needed self-repair; full N=20 convergence test deferred to a future eval. |
| `intent_annotated_solver` | ✅ (after self-repair) | ✅ | **Pact**: intent annotations improve pass rate | **Inconclusive without delta.** We need a paired benchmark *without* the `@intent` annotation to measure the lift. Filed as follow-up. |
| `typed_stream_pipeline` | ✅ | ✅ | **Plumbing**: typed streaming pipelines catch wiring errors | **Neutral.** Both pass on a 3-stage filter→map→fold pipeline. Static well-formedness is verified by AILANG's type system at compile time. |
| `parallel_independent_subtasks` | ✅ (after self-repair) | ✅ | **Quasar**: explicit independence exposes parallelism | **Neutral.** Structural independence is achievable in both. AILANG self-repair needed, suggesting the prompt-to-AILANG path has friction. |
| `parallel_map_reduce` | ✅ | ✅ | AILANG-strength: HOF polymorphism | **Confirmed.** AILANG handles polymorphic map_reduce one-shot. |
| `ast_patch_roundtrip` | ❌ logic | ❌ WRONG_LANG | **X07**: structural diffs beat free text | **Both fail — but for different reasons.** AILANG fails on output formatting; Python fails because the LLM wrote AILANG syntax even when asked for Python (eval-environment contamination). X07's hypothesis isn't testable from these results without a structural-edit variant. |
| `audit_chain_replay` | ❌ PAR_001 | ✅ | **Boruna**: replayability + audit chains | **AILANG syntax gap surfaced.** The LLM produced AILANG that the parser rejected (`PAR_001`) even after self-repair. Filed as follow-up issue. |
| `decision_block_capture` | ❌ PAR_001 | ✅ | **Aver**: structured rationale alongside implementation | **AILANG syntax gap surfaced.** Same `PAR_001` pattern. The "implementation + structured side-channel" pattern is hard to express in AILANG idiom. Filed as follow-up. |

## What the Failures Tell Us

The three AILANG failures cluster on a common pattern: **the LLM struggles to produce AILANG code that contains a non-functional auxiliary output alongside the main computation.** Two of three (`audit_chain_replay`, `decision_block_capture`) hit `PAR_001` parser errors that self-repair couldn't fix. The third (`ast_patch_roundtrip`) is a different category — both languages failed it, suggesting a stdlib/prompting gap on JSON manipulation.

Concrete follow-up improvements suggested by these failures:

1. **Teaching prompt gap on multi-line `print` patterns** — `audit_chain_replay` requires printing the same value twice. The LLM tried syntactic shortcuts that the parser rejected. Worth a prompt example.
2. **Teaching prompt gap on string-prefixed structured output** — `decision_block_capture` requires `print("CHOICE: " ++ ...)` alongside the main result. Same issue.
3. **JSON manipulation gap in stdlib coverage** — `ast_patch_roundtrip` failed on both languages; for AILANG specifically, the `std/json` decode → mutate → encode round-trip is under-documented in the agent prompt.

## What the Hypotheses Map Tells Us

Two camp-defining hypotheses are **locally refuted** by this run:

1. **NERD's tokenizer-ambiguity claim** — AILANG's operator-heavy code passes one-shot with no special tokenization treatment. The bet that ambiguous operators are the bottleneck doesn't hold up here.
2. **Vera's named-reference-breakdown claim** — AILANG handles heavy shadowing correctly. De Bruijn slot references are not necessary; HM type inference is sufficient.

Three hypotheses are **inconclusive** without further work:

- **Pact's intent-annotation lift** needs a paired benchmark (with/without `@intent`) to measure the delta. Filed.
- **Zero's canonical-form convergence** needs the N=20 multi-run harness extension that was deferred. Filed.
- **X07's structural-diff superiority** can't be measured without a structural-edit variant of `ast_patch_roundtrip`. Filed.

The remaining hypotheses (Magpie SSA, Plumbing typed streams, Quasar parallelism, Boruna audit, Aver decision blocks) show **AILANG either passes or surfaces a fixable gap**. None of them suggest AILANG needs a fundamental design change.

## Self-Repair Cost

A subtle finding: **3 of AILANG's 8 passes needed self-repair** (vs 1 of Python's 10). On those runs, token usage roughly doubles (~55k vs ~27k). This is real cost the talk should surface. The implications:

- AILANG's teaching prompt is doing most of the work, but specific patterns (multi-output, structured side-channels) still trip the LLM.
- Closing the self-repair-rate gap is a concrete agenda item: identify the patterns that need retries and add them to the teaching prompt.
- The eval-harness self-repair feature is doing what it should — catching one-shot mistakes — but every retry is a sign of a teaching-prompt gap.

## Limitations of This Run

This is **one model, one date, one run per benchmark, no multi-model variance, no N=20 convergence measurement**. It's a starting point, not a conclusion. Stronger results would come from:

- Multi-model runs (claude-sonnet, gpt-5, gemini-3) for variance
- N=20 generations of `canonical_convergence` to actually measure convergence
- Vision-tier execution with a real AI provider for the `ai_effect_*` benchmarks
- A paired (`with`/`without`) variant of `intent_annotated_solver` to measure Pact's hypothesized lift

These are tracked in the [planned design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_22_0/m-three-camps-language-survey.md) for the post-talk Phase 4.

## See Also

- [Three Camps Comparison](./three-camps-comparison) — the survey + gap-analysis context for this self-audit
- [Original Negroni post](https://negroniventurestudios.com/2026/05/20/three-camps-alike-in-dignity/) — the survey that triggered this work
- AILANG eval harness — how AILANG measures itself (see `docs/docs/guides/evaluation/`)
- Raw eval results: `eval_results/three-camps-self-audit/`

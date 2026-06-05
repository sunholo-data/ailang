---
sidebar_position: 9
title: Model Capability Threshold
description: How model capability correlates with AILANG performance — and why frontier models score higher on AILANG than Python
---

# Model Capability Threshold

One of the most striking patterns in AILANG's eval data is that **above a certain capability level, models score *higher* on AILANG than on Python** for the same coding tasks. Below that threshold, the gap reverses sharply.

This page documents the threshold, what predicts it, and what it means for AILANG's design goals.

---

## The Pattern

Running the same 50+ benchmark tasks in both AILANG and Python, across all models in the eval suite, reveals a clear capability threshold:

| Model tier | AILANG score | Python score | Gap |
|---|---|---|---|
| **Frontier** (Claude Opus/Sonnet 4.x, GPT-5.2-codex, GPT-5.4, Gemini 2.5) | 80–90% | 78–93% | **AILANG ≥ Python** |
| **Strong** (Claude Haiku 4.5, Gemini 3-flash) | 67–75% | 65–72% | Roughly equal |
| **Mid** (Gemini 3-pro, GPT-5.1-instant, GPT-5.1) | 53–66% | 69–80% | Python +13–24 pts |
| **Weak** (GPT-5-mini, GPT-5) | 14–15% | 69–74% | Python +55–59 pts |
| **Local (small MoE)** (Qwen 3.5, Gemma 4, local) | 59–73% | 87–88% | Python +14–29 pts |

The crossover happens around **80% AILANG score** — once a model clears that bar, it performs at least as well on AILANG as Python, and often better.

---

## Why This Happens

**AILANG is harder to fake.** A model can write working Python by pattern-matching billions of Python examples in its training data. AILANG has very little training data — every solve requires genuine understanding of the spec, the type system, and the effects model.

This means AILANG functions as a **capability discriminator**: it rewards compositional reasoning and type-directed programming rather than token-completion of familiar patterns.

Frontier models handle this well because they can:
1. **Read and apply the teaching prompt** — the 23k-token AILANG spec explains the language fully; frontier models follow it faithfully
2. **Reason about types** — explicit effect signatures and Hindley-Milner inference reward models that think about types, not just output syntax
3. **Generalise from examples** — the μRAG context injection surfaces relevant patterns at inference time; stronger models extract more signal from them

Weaker models fall back to Python idioms even when instructed otherwise, producing `None` instead of `import std/option (Option, None)`, `when` guards instead of `if-then-else` inside match arms, or `list[string]` where a `string` is expected.

---

## Regression Analysis

We ran Pearson correlation between AILANG performance and two external benchmarks across 14 models (June 2026):

| Relationship | r | Interpretation |
|---|---|---|
| AILANG% ~ SWE-bench Verified | **0.70** | Moderate positive |
| AILANG-vs-Python Δ ~ SWE-bench Verified | **0.60** | Moderate positive |
| AILANG% ~ SWE-bench Pro | **0.46** | Weak |
| AILANG-vs-Python Δ ~ SWE-bench Pro | **−0.03** | No correlation |

**The surprising finding:** AILANG correlates *more strongly* with SWE-bench Verified (the contaminated benchmark) than with SWE-bench Pro (the cleaner one). This is counterintuitive but explainable: the contamination in SWE-bench Verified is correlated with the specific capability AILANG measures — spec-following and generalisation to novel typed language patterns. Models that memorised SWE-bench gold patches are generally also the same models that carefully read system prompts and follow unfamiliar type systems.

**More importantly:** the AILANG-vs-Python delta (the ATT signal) correlates r=0.60 with SWE-bench Verified but r=−0.03 with SWE-bench Pro — essentially zero. This means **no current external benchmark cleanly predicts whether a model will match Python on AILANG**. AILANG evals are capturing something the benchmarks miss.

### What AILANG evals actually measure (that benchmarks don't)

The outlier analysis reveals two distinct failure modes:

**Models that underperform on AILANG vs SWE expectations** (SWE-bench score overstates AILANG capability):
- `gpt5-1` (−15 pts): likely SWE-bench contamination without genuine spec-following
- `gemini-3-pro` (−21 pts): reliability/context limits at agentic depth
- `gpt5-mini` (−20 pts): insufficient capacity for novel language reasoning

**Models that overperform on AILANG vs SWE expectations** (AILANG reveals capability SWE misses):
- `gemini-2-5-flash` (+24 pts), `gemini-2-5-pro` (+18 pts): Google's smaller models read the teaching prompt carefully and generalise well — SWE-bench undervalues this because SWE tasks reward tool-use scaffolding more than spec comprehension
- `claude-sonnet-4-5` (+15 pts): strong spec-follower despite being a slightly older model

The ATT is therefore best expressed in AILANG-score terms directly, not via external benchmark proxies.

## The Correlation with External Benchmarks

*(SWE-bench Verified scores fetched June 2026 from llm-stats.com, benchlm.ai, and vendor pages — see sources)*

| Model | AILANG % | SWE-bench Verified | AILANG vs Python |
|---|---|---|---|
| claude-opus-4-7 | 84% | **87.6%** | AILANG +1pt |
| gpt5-2-codex | 84% | **85.0%** | AILANG +3pts |
| claude-opus-4-6 | 84% | **80.8%** | AILANG +2pts |
| gpt5-4 | 86% | **~80%** | AILANG +2pts |
| gemini-3-1-pro | 66% | **80.6%** | Python +13pts ⚠️ |
| claude-sonnet-4-6 | 83% | **79.6%** | AILANG +2pts |
| gemini-3-flash | 67% | **78.0%** | AILANG +2pts |
| claude-sonnet-4-5 | 90% | **77.2%** | AILANG −3pts |
| gpt5-1 | 59% | **76.3%** | Python +21pts |
| gemini-3-pro | 53% | **76.2%** | Python +16pts |
| claude-haiku-4-5 | 75% | **73.3%** | AILANG +3pts |
| gemini-2-5-pro | 79% | **63.8%** | AILANG +4pts |
| gemini-2-5-flash | 75% | **54%** | AILANG +8pts |
| gpt5-mini | 15% | **~38%** | Python +59pts |

> ⚠️ **Important caveat:** SWE-bench Verified scores above ~77% are flagged for potential training-data contamination (OpenAI's internal audit found frontier models reproducing gold patches verbatim; SWE-bench Pro is the more reliable benchmark at the frontier). AILANG scores are measured on our eval harness (standard mode, v0.23.0).

The headline correlation holds (r=0.70 with SWE-bench Verified) but no external benchmark cleanly separates the AILANG≥Python from Python-wins tier. Key anomalies:

- **Gemini 3.1 Pro**: 80.6% SWE-Verified, 54.2% SWE-Pro, but **66% AILANG / Python wins**. The SWE-Verified score is likely inflated (contamination + scaffolding effects); SWE-Pro's 54.2% better reflects its actual novel-language capability.
- **GPT-5.1**: 76.3% SWE-Verified but only 59% AILANG. Similar pattern.
- **Gemini 2.5 Flash**: 54% SWE-Verified but **75% AILANG / +8pts vs Python**. Google's flash models are exceptional at prompt-following; SWE-bench undervalues this.

**Working ATT estimate: AILANG score ≥ 70% = model will likely match or beat Python.** External benchmarks are useful pre-screening (don't bother testing a model under 50% SWE-bench Verified) but the AILANG score itself is the ground truth — it has no contamination by construction, since almost no AILANG code exists in any model's training data.

> As AILANG becomes more widely adopted and training data accumulates, this contamination advantage will erode — but by then, AILANG will have entered enough codebases that models will have been trained on real usage, which is a better outcome anyway.

---

## The AILANG Teachability Threshold (ATT)

The crossover point has a name: the **AILANG Teachability Threshold** — the minimum external coding capability score (SWE-bench Verified or equivalent) at which a model achieves AILANG ≥ Python parity on the core eval tier.

**Current ATT: ~55% SWE-bench Verified** *(as of v0.23.0, June 2026)*

This is a first-class KPI for AILANG development. A declining ATT is direct evidence the language is becoming more AI-tractable. We track it per release.

### Why ATT matters strategically

ATT benefits from **two independent improvement trends simultaneously**:

```
AILANG improves        → ATT drops (55% → 45% → 35%)
Models improve         → More models cross the current ATT
Compound effect        → Every ATT drop instantly enfranchises all models
                         already above the new threshold
```

If AILANG improvements lower ATT from 55% to 40%, every model currently in the 40–55% SWE-bench range — dozens of capable, widely-deployed models — crosses into "AILANG ≥ Python" territory with zero changes on their side. AILANG's improvement agenda directly expands its addressable model cohort.

The chart to watch over releases:
- x-axis: SWE-bench Verified score
- y-axis: AILANG score − Python score (positive = AILANG wins)
- The crossover x-intercept is ATT. Track it declining over versions.

## Frontier Model Failures — What Blocks the Last 15%?

Even the best models (Claude Opus 4.7, GPT-5.2-Codex, Claude Sonnet 4.6) fail ~8–9 AILANG benchmarks in standard mode. Analysing *what* they fail on reveals two distinct failure classes:

### Class 1: Language gaps (compile_error) — fixable

These fail in multiple frontier models with `compile_error`, meaning the model generated syntactically or type-invalid AILANG. The spec is clear but the model generated something AILANG rejects:

| Benchmark | Models failing | Pattern |
|---|---|---|
| `log_file_analyzer` | 9/9 frontier models | Complex string parsing — model invents methods not in std/string |
| `multi_module_imports` | 4 models (all compile) | Module path resolution idioms |
| `polymorphic_ord_defaulting` | 5 models | Polymorphic comparisons — model uses wrong syntax |
| `contract_roman_numeral` | 5 models (80% compile) | Contract annotation syntax |
| `type_unify` | 4 models (100% compile) | Type-level unification patterns |
| `config_file_parser` | 3 models (100% compile) | Match arm syntax (the guard-clause gap) |
| `pipeline`, `run_length_encode`, `red_black_tree` | 2–5 models | Various compile patterns |

**These are fixable.** Each is a prompt/stdlib improvement in the design doc queue.

### Class 2: Capability ceiling (logic_error) — harder

These fail with correct AILANG syntax but wrong output — the model understood the language but got the algorithm wrong. Notably, `contract_rle_roundtrip` fails as **logic_error in all three top frontier models simultaneously** — this is the single confirmed hard capability ceiling in v0.23.0.

Other shared logic failures: `contract_sorted_merge`, `lambda_calc`, `contract_matrix_determinant`, `binary_tree_sum`. These require either model improvements or new AILANG stdlib primitives that make the task expressible more naturally.

### Breakeven projections

| Tier | Current AILANG% | After fixing compile gaps | Hard ceiling |
|---|---|---|---|
| Frontier (Claude Opus 4.7) | 84% | **~88%** | ~90% (4 logic-error benchmarks) |
| Mid-strong (Claude Haiku 4.5) | 75% | **~83%** | — |
| Mid-tier CPR (Qwen 3.5 local) | 64% CPR | **~82% CPR** | — |

The frontier ceiling of ~88–90% represents tasks genuinely difficult to express in any typed functional language at this model tier. Pushing past it requires either the ecosystem growing (more AILANG in training data), new stdlib primitives, or models improving on structured reasoning tasks generally.

## Implications for AILANG's Design Goals

**Short term:** The 3 most common failure categories for mid-tier models are already documented as prompt improvements:
- `None`/`Some` without `import std/option` → [m-prompt-option-none-idiom](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_24_0/m-prompt-option-none-idiom.md)
- Match guard syntax (`when` clauses) → [m-prompt-match-guard-syntax](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_24_0/m-prompt-match-guard-syntax.md)
- `split()` returning `list[string]` → [m-prompt-split-list-operations](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_24_0/m-prompt-split-list-operations.md)

Fixing these 3 gaps is expected to push a model like Qwen 3.5 (currently 59% AILANG) to ~73% — potentially crossing the frontier threshold into "AILANG ≥ Python" territory.

**Long term:** As AILANG matures and more examples accumulate in model training data, the threshold will lower. The goal is a world where even mid-tier models score ≥ Python on AILANG — making AILANG the path of least resistance for AI-assisted programming, not an expert-only tool.

**The μRAG effect:** Our A/B experiments show microRAG knowledge injection (injecting relevant syntax chunks at inference time) moves **+4 benchmarks** on the smoke tier — from 30/34 to 34/34. This is particularly valuable for mid-tier models where the static teaching prompt isn't enough: targeted context injection bridges the gap between "has read the spec" and "can recall the right idiom."

---

## The right KPI to optimise: Conditional Pass Rate (CPR)

The raw AILANG score is misleading for tracking progress — it conflates model capability with language accessibility. A better metric is:

**CPR = P(AILANG passes | Python passes)**
> "When the model can solve a task in Python, what % can it also solve in AILANG?"

This strips out model capability and asks only: *does AILANG impose extra friction vs Python?*

**Empirical values (Qwen 3.5 mxfp8, core tier, agent mode, v0.23.0):**
- CPR = **64.3%** — 18 of 28 Python-solvable tasks also pass in AILANG
- 10 blockers where Python works but AILANG fails: **5 compile_errors** (fixable) + 4 logic + 1 timeout
- Projected CPR after fixing 5 language gaps: **~82%**

**The optimisation target:** CPR → 100% for mid-tier models. This means: every task a Qwen/Haiku-class model can do in Python, it can also do in AILANG. That's the "AILANG as natural as Python" milestone.

| Milestone | CPR target | What achieves it |
|---|---|---|
| Today (v0.23.0) | 64% | Baseline |
| Fix all compile_error gaps | ~82% | Prompt + stdlib improvements |
| Reach Python parity | 100% | Full prompt coverage + ecosystem growth |

**Important note on saturation:** the raw delta (AILANG%−Python%) has r=0.92 correlation with the AILANG score level itself — it's mostly a ceiling-effect artefact. Never use raw delta as a KPI; use CPR or the AILANG/Python ratio instead (r=0.59 with SWE-bench Verified, much more stable).

## Monitoring

The nightly eval rotation (running on the M4 Max Studio at 03:00) tracks the smoke tier continuously. Each Monday it runs `--microrag on` and `--microrag off` for an A/B comparison (current A/B result: microRAG +4 benchmarks on the smoke tier — 34/34 vs 30/34). Results broadcast to the `controlplane` inbox.

To check current standing:
```bash
ailang messages list --compact | grep nightly
ailang eval-sweet-spot eval_results/baselines/<latest>
```

---

## Data Sources

- AILANG eval results: `eval_results/baselines/` (this repo)
- External benchmarks: vendor model cards, [Artificial Analysis](https://artificialanalysis.ai), [LMSYS Chatbot Arena](https://chat.lmsys.org), community SWE-bench leaderboards
- Correlation analysis: June 2026, n=19 models with ≥5 benchmarks in both AILANG and Python

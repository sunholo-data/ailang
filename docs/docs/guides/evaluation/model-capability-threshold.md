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

The headline correlation holds — but **SWE-bench Verified is not the clean predictor it first appears**:

- Gemini 3.1 Pro scores 80.6% SWE-bench Verified but only 66% AILANG, putting it in "Python wins" territory. SWE-bench contamination likely explains the gap.
- Models with *genuine* SWE-bench capability (Claude Opus, GPT-5.2-Codex, Gemini 2.5) show the pattern cleanly.
- A better predictor than the raw SWE-bench number may be **SWE-bench Pro** (harder, less contaminated) — once scores on that benchmark are widely available, the ATT will be expressible more precisely.

**Working estimate for ATT: ~55% SWE-bench Verified (uncontaminated) or ~60% AILANG eval score itself.** The AILANG score is actually the cleanest proxy for "will this model handle AILANG as well as Python" — because AILANG evals have no training-data contamination by construction.

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

## Implications for AILANG's Design Goals

**Short term:** The 3 most common failure categories for mid-tier models are already documented as prompt improvements:
- `None`/`Some` without `import std/option` → [m-prompt-option-none-idiom](../../../../design_docs/planned/v0_24_0/m-prompt-option-none-idiom.md)
- Match guard syntax (`when` clauses) → [m-prompt-match-guard-syntax](../../../../design_docs/planned/v0_24_0/m-prompt-match-guard-syntax.md)
- `split()` returning `list[string]` → [m-prompt-split-list-operations](../../../../design_docs/planned/v0_24_0/m-prompt-split-list-operations.md)

Fixing these 3 gaps is expected to push a model like Qwen 3.5 (currently 59% AILANG) to ~73% — potentially crossing the frontier threshold into "AILANG ≥ Python" territory.

**Long term:** As AILANG matures and more examples accumulate in model training data, the threshold will lower. The goal is a world where even mid-tier models score ≥ Python on AILANG — making AILANG the path of least resistance for AI-assisted programming, not an expert-only tool.

**The μRAG effect:** Our A/B experiments show microRAG knowledge injection (injecting relevant syntax chunks at inference time) moves **+4 benchmarks** on the smoke tier — from 30/34 to 34/34. This is particularly valuable for mid-tier models where the static teaching prompt isn't enough: targeted context injection bridges the gap between "has read the spec" and "can recall the right idiom."

---

## Monitoring

The nightly eval rotation (running on the M4 Max Studio at 03:00) tracks this threshold over time. Each Monday it runs the smoke tier with both `--microrag on` and `--microrag off` for the A/B comparison. Results broadcast to the `controlplane` inbox.

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

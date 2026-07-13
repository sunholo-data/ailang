---
sidebar_position: 0
title: Benchmarks Overview
description: A guide to every AILANG benchmark view — which one answers your question, what each measures, and how to read the numbers honestly.
---

# Benchmarks Overview

AILANG's thesis is one question: **can AI coding agents write AILANG as well as they write mainstream languages?** To answer it, we run the same tasks through many models — in AILANG and, for comparison, Python (plus JavaScript and Go on the local rig) — continuously, and publish everything here, including where AILANG falls behind.

This page is the map. Each view answers a different question.

## Which view do you want?

| Your question | Start here |
|---|---|
| Which models are best at AILANG right now? | [Model Leaderboard](/docs/benchmarks/performance) |
| How hard is a given benchmark, and which models beat it? | [ELO &amp; Difficulty](/docs/benchmarks/elo) |
| Does wrapping a model in an agent loop actually help? | [Agent Harness Explorer](/docs/benchmarks/explorer) |
| Which model is the best *value* — cost vs quality vs speed? | [Value Score](/docs/benchmarks/value) |
| How does a free, on-device model compare to the cloud frontier? | [OS / Local Leaderboard](/docs/benchmarks/os-model-leaderboard) |
| What does a benchmark actually ask, and how did models solve it? | [Benchmark Gallery](/docs/benchmarks/gallery) |
| How is the AILANG codebase itself growing? | [Codebase Statistics](/docs/benchmarks/codebase-stats) |

## The views

### [Model Leaderboard](/docs/benchmarks/performance)
The headline: per-model AILANG-vs-Python pass rates, the success-rate trend over releases, a sortable comparison table (with a highlighted **on-device "Local GPU agent"** row so you can see the free option against the frontier), and a failure-mode breakdown that separates real capability failures from budget limits, provider noise, and safety refusals.

### [ELO &amp; Difficulty](/docs/benchmarks/elo)
Two ratings, both *derived* from head-to-head results rather than hand-assigned: a **capability** rating per model and a **difficulty** rating per benchmark. Split by mode (standard vs agent); the on-device GPU agents are ranked and highlighted here in agent mode. Click any benchmark to jump straight to its spec sheet in the Gallery.

### [Agent Harness Explorer](/docs/benchmarks/explorer)
Agent mode only — the same model run through different agentic CLIs (Claude, Codex, opencode, Pi, motoko). Includes the like-for-like **agent-uplift** table (how much a model gains from a multi-turn loop vs a plain 0-shot call) and per-executor agent performance.

### [Value Score](/docs/benchmarks/value)
Cost vs quality vs speed. The Pareto frontier of "cheapest model that still passes," weighted value scores, and a score-vs-cost scatter.

### [OS / Local Leaderboard](/docs/benchmarks/os-model-leaderboard)
Open-source / locally-hosted models, run continuously on a Mac Studio rig at **~$0/run**. This is where the **multi-language** (JavaScript, Go) and **cross-harness** comparisons live, with N≥3 trials and per-release trends.

### [Benchmark Gallery](/docs/benchmarks/gallery)
Every benchmark as a **spec sheet**: the task prompt and expected output, pass rates by language, by model, and by agent harness, plus a browsable sample solution for each language. Search and filter by tier and feature-area tag. This is the place to understand *what* is being tested.

### [Codebase Statistics](/docs/benchmarks/codebase-stats)
AILANG repository growth and AI-assisted development metrics over time.

## How to read the numbers

- **Standard and agent never mix.** *Standard* = 0-shot generation + self-repair via the API; *agent* = a multi-turn agentic CLI. Every chart is one or the other, labeled — a model only appears on a view for the modes it actually ran.
- **Tiers.** Benchmarks are grouped smoke · core · stretch · frontier · vision. On harder tiers fewer models have full coverage, so partial-coverage models are marked **provisional** (dimmed, with a coverage badge) and don't earn a rank until coverage fills in — a 6-benchmark score can't be misread as beating a 55-benchmark one.
- **Regraded for formatting parity.** A correct answer isn't marked wrong over `True` vs `true` or `7.50` vs `7.5`.
- **Refusals are counted, but flagged.** A safety refusal (the model declines the prompt) still counts as a non-pass, but carries a "⚠ N% refused" note so a decline-driven number isn't misread as "can't code."
- **On-device is a first-class option.** The local GPU agents (a local Qwen run through motoko / opencode / Pi) are slow and cost ~$0/run; they're highlighted in cyan wherever they appear so the "free local" story is easy to spot.
- **Cloud is AILANG + Python; multi-language is a local-rig concern.** Running N-trial JavaScript/Go sweeps on pay-per-token cloud APIs is expensive; the on-device rig does it for free.

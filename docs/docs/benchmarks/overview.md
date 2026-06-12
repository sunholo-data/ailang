---
sidebar_position: 0
title: Benchmarks Overview
description: How AILANG measures coding-agent performance — two axes (cloud frontier vs OS/local) and where to find each view.
---

# Benchmarks Overview

AILANG continuously evaluates how well coding agents synthesize **correct AILANG** (and, for
comparison, other languages). The dashboards split along **two axes** — pick the one that
answers your question:

## 1. Cloud frontier — AILANG vs Python

The best pay-per-token API models, on **AILANG and Python**. This is the "how capable is the
frontier on AILANG?" view. Numbers are **regraded** (formatting parity — a correct answer isn't
marked wrong over `True` vs `true` or `7.50` vs `7.5`).

- **[Model Leaderboard](/docs/benchmarks/performance)** — per-model pass rates, AILANG-vs-Python gap, trend over releases.
- **[ELO Ratings & Difficulty](/docs/benchmarks/elo)** — capability rating per model, difficulty rating per benchmark (derived, not hand-assigned), saturation + grader-artifact flags. Split by mode (standard vs agent).
- **[Value Score](/docs/benchmarks/value)** — cost vs quality vs speed.

## 2. OS / Local models — cross-language, cross-harness, longitudinal

Open-source / locally-hosted models, run continuously on a local rig at **zero server cost**.
This is where the **multi-language** (JavaScript, Go) and **harness** (opencode / claude / codex)
comparisons live, with **N≥3 trials** and **per-release trends**.

- **[OS / Local Leaderboard](/docs/benchmarks/os-model-leaderboard)** — cross-language × harness, longitudinal.
- **[Agent Harness Explorer](/docs/benchmarks/explorer)** — agent-mode cross-harness, by language.

## Two rules that keep the numbers honest

- **Standard and agent never mix.** *Standard* = 0-shot + self-repair via the API; *agent* =
  multi-turn agentic CLI. Every chart is one or the other, labeled — a model only appears on a
  view for the modes it actually ran.
- **Cloud is AILANG+Python; multi-language is an OS/local concern.** Running N-trial JS/Go sweeps
  on cloud APIs is expensive; the local rig does it for free.

## Also

- **[Benchmark Gallery](/docs/benchmarks/gallery)** — browse the benchmark suite by tier (prompts, expected output, per-language pass rates).
- **[Codebase Statistics](/docs/benchmarks/codebase-stats)** — AILANG repository growth over time.

---
sidebar_position: 1
title: Agent Harness Explorer
description: Compare coding agent performance across harnesses (Claude CLI, opencode, Codex, and the hosted Managed Agents API) by language — agent mode only
---

import BenchmarkExplorer from '@site/src/components/BenchmarkExplorer';
import AgentUpliftTable from '@site/src/components/AgentUpliftTable';

# Agent Harness Explorer

**Agent mode only** — these results come from multi-turn agentic coding sessions (Claude CLI, opencode, Codex, Managed Agents). Pre-v0.22 runs used the now-retired Gemini CLI; those rows are kept as historical data. For 0-shot API and self-repair results, see [Benchmarks](/docs/benchmarks/performance).

Browse by language, harness, and model. The cross-harness comparison shows what happens when the same underlying model runs through a different CLI.

This view aggregates **two sources**: the cloud release/nightly baseline, and the continuous **on-device** rotation (local Ollama models on opencode and Pi, $0/run) which refreshes between releases. On-device models are tagged `on-device` and let you compare, for example, the same local Qwen across the opencode and Pi harnesses.

<BenchmarkExplorer />

## What does agent mode add?

AILANG is built for AI coding *with harnesses*, so the question that matters is: for a given model, how much does wrapping it in an agentic loop (multi-turn, tool-using) improve its AILANG pass rate over a plain 0-shot API call? This table answers it **like-for-like** — each model compared to *itself*, over only the benchmarks both modes ran. A negative delta means the agent loop actually *hurt* that model.

<AgentUpliftTable />

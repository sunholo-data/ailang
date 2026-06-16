---
sidebar_position: 1
title: Agent Harness Explorer
description: Compare coding agent performance across harnesses (Claude CLI, Gemini CLI, opencode, Codex) by language — agent mode only
---

import BenchmarkExplorer from '@site/src/components/BenchmarkExplorer';
import OSVersionTrend from '@site/src/components/OSVersionTrend';

# Agent Harness Explorer

**Agent mode only** — these results come from multi-turn agentic coding sessions (Claude CLI, Gemini CLI, opencode, Codex). For 0-shot API and self-repair results, see [Benchmarks](/docs/benchmarks/performance).

Browse by language, harness, and model. The cross-harness comparison shows what happens when the same underlying model runs through a different CLI.

This view aggregates **two sources**: the cloud release/nightly baseline, and the continuous **on-device** rotation (local Ollama models on opencode and Pi, $0/run) which refreshes between releases. On-device models are tagged `on-device` and let you compare, for example, the same local Qwen across the opencode and Pi harnesses.

<BenchmarkExplorer />

## Local-rig version trend

Does each AILANG release move the needle for **local models**? This tracks the on-device rotation (opencode + Pi + motoko on local Qwen, $0/run) **per AILANG release** — columns are AILANG versions, newest on the right. Retired models freeze at their last version; active models re-run each release.

<OSVersionTrend />

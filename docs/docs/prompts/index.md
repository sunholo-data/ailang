---
title: AI Prompts
sidebar_position: 0
reviewed: 2026-11-26
reviewBy: 2027-02-26
---

# AI Prompts for AILANG

These prompts teach AI models how to write correct AILANG code without relying on training-data priors.

## Current Prompt

The **[Current Teaching Prompt](/docs/prompts/current)** is automatically synced from `prompts/versions.json` at build time so the website and `ailang prompt` always match.

:::tip Recommended
Use `ailang prompt` via the CLI or `prompt_get` via [Hosted Docs MCP](/docs/guides/agent-mcp) so your session uses the exact compiler-matched prompt.
:::

## Using the CLI

```bash
# Get current/active prompt
ailang prompt

# Get a specific historical version
ailang prompt --version v0.16.20

# List all available versions
ailang prompt --list

# Save to file
ailang prompt > syntax.md
```

## Example Request to AI

```text
Using AILANG (see teaching prompt below), write a program that:
- Reads a list of numbers from user input
- Filters out even numbers
- Returns the sum of remaining odd numbers

[Paste output from: ailang prompt]
```

## Historical Versions & Metadata

Only the active prompt is published on the website (`current.md`) so site search never surfaces obsolete syntax rules. Historical snapshots (`prompts/v*.md`) and their SHA-256 hashes, tags (`production`, `latest`), and release notes remain tracked in [`prompts/versions.json`](https://github.com/sunholo-data/ailang/blob/dev/prompts/versions.json) and accessible locally via `ailang prompt --version <tag>`.

## Learn More

- **[AI Prompt Guide](/docs/guides/ai-prompt-guide)** — Detailed usage instructions and eval workflow
- **[Agent Integration Guide](/docs/guides/agent-integration)** — CLI + MCP routing table and repair loop
- **[AI Effect System](/docs/guides/ai-effect)** — Calling LLMs from inside AILANG via `std/ai`

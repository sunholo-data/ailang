---
title: AI Prompts
sidebar_position: 0
---

# AI Prompts for AILANG

These prompts teach AI models how to write correct AILANG code.

## Current Prompt

The **[Current Teaching Prompt](/docs/prompts/current)** is automatically synced from the AILANG source repository at build time. It always reflects the active prompt version used in production.

:::tip Recommended
Use `ailang prompt` via the CLI for the most accurate, version-locked prompt.
:::

## Using the CLI (Recommended)

The `ailang` CLI provides built-in access to all prompt versions:

```bash
# Get current/active prompt
ailang prompt

# Get specific version
ailang prompt --version v0.5.11

# List all available versions
ailang prompt --list

# Save to file
ailang prompt > syntax.md
```

**Install AILANG:**
```bash
# Download binary (macOS Apple Silicon)
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/ailang-darwin-arm64.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# Or build from source
git clone https://github.com/sunholo-data/ailang && cd ailang && make install
```

## Example Request to AI

When asking an AI model (Claude, GPT, Gemini) to write AILANG code:

```
Using AILANG (see teaching prompt below), write a program that:
- Reads a list of numbers from user input
- Filters out even numbers
- Returns the sum of remaining odd numbers

[Paste output from: ailang prompt]
```

## Version Management

Prompt versions are tracked in [`prompts/versions.json`](https://github.com/sunholo-data/ailang/blob/main/prompts/versions.json). Each version includes:

- **Hash**: SHA256 of the prompt content for integrity
- **Description**: What changed in this version
- **Tags**: `production`, `latest`, `testing`
- **Notes**: Implementation details and known issues

The website automatically syncs the active prompt during CI/CD builds.

## Learn More

- **[AI Prompt Guide](/docs/guides/ai-prompt-guide)** - Detailed usage instructions and best practices
- **[AI Effect System](/docs/guides/ai-effect)** - How AILANG tracks AI interactions

---
title: AI Prompts
sidebar_position: 4
---

# AI Prompts for AILANG

These prompts teach AI models how to write correct AILANG code.

## Current Prompts

> **Active Version**: See [`prompts/versions.json`](https://github.com/sunholo-data/ailang/blob/main/prompts/versions.json) in the repo for the current active prompt version.

### Latest Teaching Prompts

> **Note**: The website shows recent production prompts. For the complete list of all versions, use `ailang prompt --list`.

Prompts are synced from [`prompts/versions.json`](https://github.com/sunholo-data/ailang/blob/main/prompts/versions.json) automatically. The sync script (`make sync-prompts`) copies all prompts tagged with `production` or `latest`.

**Browse recent versions:**
- Navigate using the sidebar on the left
- Or use `ailang prompt --version <version>` to get any version

**Python comparison:**
- **[Python Syntax Guide](/docs/prompts/python)** - AILANG vs Python side-by-side

## Using the Prompts

### Via CLI (Recommended)

The `ailang` CLI provides built-in access to all prompt versions:

```bash
# Get current/active prompt
ailang prompt

# Get specific version
ailang prompt --version v0.3.24

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

### Via Website (Browse)

The website shows recent prompt versions for browsing. For the complete list, use `ailang prompt --list` or see [`prompts/versions.json`](https://github.com/sunholo-data/ailang/blob/main/prompts/versions.json).

### Example Request to AI

When asking an AI model (Claude, GPT, Gemini) to write AILANG code:

```
Using AILANG (see teaching prompt below), write a program that:
- Reads a list of numbers from user input
- Filters out even numbers
- Returns the sum of remaining odd numbers

[Paste output from: ailang prompt]
```

## Core Features (Latest)

**Current implementation** (v0.4.x):
- ✅ Syntactic sugar: `::` cons, `->` function types, `f()` zero-arg calls
- ✅ Multi-line ADTs: `type Tree = | Leaf | Node`
- ✅ Record updates: `{base | field: value}`
- ✅ Auto-import prelude (no imports for comparisons)
- ✅ Full module system with effects (IO, FS, Clock, Net, Env)
- ✅ Pattern matching, recursion, type classes

See the [AI Prompt Guide](/docs/guides/ai-prompt-guide) for detailed usage instructions and best practices.

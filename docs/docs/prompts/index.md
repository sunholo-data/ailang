---
title: AI Prompts
sidebar_position: 4
---

# AI Prompts for AILANG

These prompts teach AI models how to write correct AILANG code.

## Current Prompts

- **[AILANG v0.3.6](/docs/prompts/v0.3.6)** - Current AILANG teaching prompt
  - Record updates, auto-import prelude, anonymous functions
  - Recursion, pattern matching, effects, type classes
  - Updated for latest language features

- **[Python Comparison](/docs/prompts/python)** - AILANG vs Python syntax guide
  - Side-by-side syntax comparison
  - Common patterns and idioms
  - Migration tips

## Using the Prompts

When asking an AI model (Claude, GPT, Gemini) to write AILANG code:

1. **Include the full prompt** - Copy the entire v0.3.6 prompt content
2. **Be specific** - Describe what you want the code to do
3. **Mention version** - Reference "AILANG v0.3.6" to ensure correct syntax

### Example Request

```
Using AILANG v0.3.6, write a program that:
- Reads a list of numbers from user input
- Filters out even numbers
- Returns the sum of remaining odd numbers

[Include full v0.3.6 prompt here]
```

## Features by Version

**v0.3.6 (Current)**
- ✅ Record updates: `{base | field: value}`
- ✅ Auto-import prelude (no imports for comparisons)
- ✅ Anonymous functions: `func(x: int) -> int { x * 2 }`
- ✅ Numeric conversions: `intToFloat`, `floatToInt`
- ✅ Full module system with effects

See the [AI Prompt Guide](/docs/guides/ai-prompt-guide) for detailed usage instructions and best practices.

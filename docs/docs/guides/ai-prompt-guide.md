# AI Prompt Guide: Teaching AILANG to Language Models

**Purpose**: This document points to the canonical AILANG teaching prompt for AI models.

**KPI**: One of AILANG's key success metrics is **"teachability to AI"** - how easily can an LLM learn to write correct AILANG code from a single prompt?

---

## Canonical Prompt (v0.3.6)

**The official AILANG teaching prompt is maintained at**:

### 📖 [prompts/v0.3.6.md](../prompts/v0.3.6)

This prompt is:
- ✅ **Validated through eval benchmarks** - Tested across GPT-5, Gemini 2.5 Pro, Claude Sonnet 4.5
- ✅ **Up-to-date with v0.3.6 features** - Record updates, auto-import prelude, anonymous functions
- ✅ **Versioned with SHA-256 hashing** - Reproducible eval results
- ✅ **Actively maintained** - Updated as language evolves

---

## Quick Reference

**Current version**: v0.3.6 (AI Usability Improvements)

**What works in v0.3.6**:
- ✅ Module execution with effects
- ✅ Recursion (self-recursive and mutually-recursive)
- ✅ Block expressions (`{ stmt1; stmt2; result }`)
- ✅ Records (literals + field access + **updates**)
- ✅ **Record update syntax** `{base | field: value}` - NEW!
- ✅ **Auto-import std/prelude** - No imports needed for comparisons - NEW!
- ✅ **Anonymous functions** `func(x: int) -> int { x * 2 }` - NEW!
- ✅ **Numeric conversions** `intToFloat`, `floatToInt` - NEW!
- ✅ Effects: IO, FS, Clock, Net
- ✅ Type classes, ADTs, pattern matching
- ✅ REPL with full type checking

**What doesn't work yet**:
- ❌ Pattern guards (parsed but not evaluated)
- ❌ Error propagation operator `?`
- ❌ Deep let nesting (4+ levels)
- ❌ Typed quasiquotes
- ❌ CSP concurrency

**For complete details**, see [prompts/v0.3.6.md](/docs/prompts/v0.3.6)

---

## Using the Prompt

### For AI Code Generation

When asking an AI model (Claude, GPT, Gemini) to write AILANG code, provide the full prompt from [prompts/v0.3.6.md](/docs/prompts/v0.3.6).

**Example usage**:
```
I need you to write AILANG code to solve this problem: [problem description]

First, read this AILANG syntax guide:
[paste contents of prompts/v0.3.6.md]

Now write the code.
```

### For Eval Benchmarks

The eval harness automatically loads the correct prompt version:

```yaml
# benchmarks/example.yml
id: example_task
languages: ["ailang", "python"]
prompt_files:
  ailang: "prompts/v0.3.6.md"
  python: "prompts/python.md"
task_prompt: |
  Write a program that [task description]
```

See [benchmarks/README.md](https://github.com/sunholo-data/ailang/tree/main/benchmarks) for details.

---

## Current Prompt

**Version**: v0.3.6 - [View full prompt](/docs/prompts/v0.3.6)

**Features**:
- Record updates: `{base | field: value}`
- Auto-import std/prelude
- Anonymous functions: `func(x: int) -> int { x * 2 }`
- Numeric conversions: `intToFloat`, `floatToInt`
- Full module system with effects

**Why prompt quality matters**:
- Better AI code generation
- Reproducible eval results
- Consistent teaching across models

---

## Eval Results

**Current success rates** (v0.3.6 prompt on v0.3.8):
- **Overall**: 49.1% AILANG success rate (vs 82.5% Python baseline)
- **Claude Sonnet 4.5**: 68.4% best performer
- **Gemini 2.5 Pro**: 65.8%
- **GPT-5**: 63.2%

**Improvement trajectory**:
- v0.3.7: 38.6% → v0.3.8: 49.1% (+10.5% improvement)
- Fixed benchmarks: pattern_matching_complex, adt_option, error_handling, and more

See [Benchmark Dashboard](/docs/benchmarks/performance) for detailed metrics.

---

## Contributing Improvements

If you find ways to improve the AILANG teaching prompt:

1. **Test your changes** with the eval harness:
   ```bash
   ailang eval --benchmark all --model gpt-4o-mini
   ```

2. **Measure impact**:
   ```bash
   tools/compare_prompts.sh old_version new_version
   ```

3. **Update the prompt** at `prompts/v0.3.6.md` (or create new version)

4. **Document changes** in `prompts/versions.json` (future enhancement)

---

## See Also

- **[CLAUDE.md](https://github.com/sunholo-data/ailang/blob/main/CLAUDE.md)** - Instructions for AI assistants working on AILANG development
- **[examples/](https://github.com/sunholo-data/ailang/tree/main/examples)** - Working AILANG code examples
- **[Language Reference](/docs/reference/language-syntax)** - Complete AILANG syntax guide
- **[benchmarks/](https://github.com/sunholo-data/ailang/tree/main/benchmarks)** - Eval harness benchmark suite

---

*Last updated: October 15, 2025 for v0.3.6/v0.3.8*

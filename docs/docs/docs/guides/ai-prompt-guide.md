# AI Prompt Guide: Teaching AILANG to Language Models

**Purpose**: This document points to the canonical AILANG teaching prompt for AI models.

**KPI**: One of AILANG's key success metrics is **"teachability to AI"** - how easily can an LLM learn to write correct AILANG code from a single prompt?

---

## Canonical Prompt (v0.3.0)

**The official AILANG teaching prompt is maintained at**:

### 📖 [prompts/v0.3.0.md](../prompts/v0.3.0)

This prompt is:
- ✅ **Validated through eval benchmarks** - Tested across GPT-4o-mini, Gemini 2.0, Claude Sonnet 4.5
- ✅ **Up-to-date with v0.3.0 features** - Recursion, blocks, records, Clock/Net effects
- ✅ **Versioned with SHA-256 hashing** - Reproducible eval results
- ✅ **Actively maintained** - Updated as language evolves

---

## Quick Reference

**Current version**: v0.3.0 (Clock & Net Effects + Type System Fixes)

**What works in v0.3.0**:
- ✅ Module execution with effects
- ✅ Recursion (self-recursive and mutually-recursive)
- ✅ Block expressions (`{ stmt1; stmt2; result }`)
- ✅ Records (literals + field access)
- ✅ Effects: IO, FS, Clock, Net
- ✅ Type classes, ADTs, pattern matching
- ✅ REPL with full type checking

**What doesn't work yet**:
- ❌ Record update syntax `{r | field: val}`
- ❌ Pattern guards (parsed but not evaluated)
- ❌ Error propagation operator `?`
- ❌ Deep let nesting (4+ levels)
- ❌ Typed quasiquotes
- ❌ CSP concurrency

**For complete details**, see [prompts/v0.3.0.md](../prompts/v0.3.0)

---

## Using the Prompt

### For AI Code Generation

When asking an AI model (Claude, GPT, Gemini) to write AILANG code, provide the full prompt from `prompts/v0.3.0.md`.

**Example usage**:
```
I need you to write AILANG code to solve this problem: [problem description]

First, read this AILANG syntax guide:
[paste contents of prompts/v0.3.0.md]

Now write the code.
```

### For Eval Benchmarks

The eval harness automatically loads the correct prompt version:

```yaml
# benchmarks/example.yml
id: example_task
languages: ["ailang", "python"]
prompt_files:
  ailang: "prompts/v0.3.0.md"
  python: "prompts/python.md"
task_prompt: |
  Write a program that [task description]
```

See [benchmarks/README.md](https://github.com/sunholo-data/ailang/tree/main/benchmarks) for details.

---

## Prompt Versioning

AILANG teaching prompts are versioned alongside language releases:

| Version | File | Features |
|---------|------|----------|
| v0.2.0 | `prompts/v0.2.0.md` | Module execution, effects (IO, FS) |
| v0.3.0 | `prompts/v0.3.0.md` | + Recursion, blocks, records, Clock/Net |
| v0.4.0 | `prompts/v0.4.0.md` | (Future: record updates, pattern guards) |

**Why versioning matters**:
- Reproducible eval results (hash verification)
- A/B testing of teaching strategies
- Track prompt evolution over time

---

## Eval Results

**Current success rates** (v0.3.0 prompt):
- **GPT-4o-mini**: 100% success on recursion/blocks/records benchmarks
- **Gemini 2.0 Flash**: 100% success on recursion/blocks/records benchmarks
- **Claude Sonnet 4.5**: 100% success on recursion/blocks/records benchmarks

**Token efficiency**:
- AILANG generates **8-15% fewer output tokens** than Python for equivalent tasks
- Prompt tokens are higher (teaching overhead), but will be reduced via fine-tuning

See [eval_results/](https://github.com/sunholo-data/ailang/tree/main/eval_results) for detailed reports.

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

3. **Update the prompt** at `prompts/v0.3.0.md` (or create new version)

4. **Document changes** in `prompts/versions.json` (future enhancement)

---

## See Also

- **[CLAUDE.md](https://github.com/sunholo-data/ailang/blob/main/CLAUDE.md)** - Instructions for AI assistants working on AILANG development
- **[examples/](https://github.com/sunholo-data/ailang/tree/main/examples)** - Working AILANG code examples
- **[LIMITATIONS.md](./limitations)** - Current limitations and workarounds
- **[benchmarks/](https://github.com/sunholo-data/ailang/tree/main/benchmarks)** - Eval harness benchmark suite

---

*Last updated: October 5, 2025 for v0.3.0*

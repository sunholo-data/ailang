# AI Usability Benchmarking Guide

## Overview

AILANG includes a built-in benchmarking system (M-EVAL) to empirically measure AI efficiency when generating code in AILANG vs Python. This helps validate AILANG's core value proposition: reducing AI token usage and improving code generation success rates.

**⚠️ Current Phase: Baseline Single-Shot (M-EVAL Phase 1)**

This guide covers **single-shot code generation** — one prompt, one response, no iteration. This provides baseline data on:
- Initial token efficiency
- Syntax familiarity
- First-attempt success rate

**Multi-turn agentic evaluation** (M-EVAL2) is planned for v0.3.0 and will measure:
- Total effort across multiple iterations
- Debugging/fix cycles
- Real-world AI coding workflows

See [M-EVAL2 Design Doc](https://github.com/sunholo-data/ailang/blob/main/design_docs/20251002/m_eval2_agentic.md) for details.

## Quick Start

### Running a Benchmark (Mock Mode)

```bash
# Run FizzBuzz benchmark with mock AI agent
ailang eval --benchmark fizzbuzz --mock

# Run with both Python and AILANG
ailang eval --benchmark fizzbuzz --mock --langs python,ailang

# Generate report
make eval-report
```

### Running with Real AI Models

```bash
# Set API keys
export OPENAI_API_KEY="your-key-here"
# or
export ANTHROPIC_API_KEY="your-key-here"

# Run benchmark with GPT-4
ailang eval --benchmark fizzbuzz --model gpt-4 --seed 42

# Run with Claude
ailang eval --benchmark adt_option --model claude-3 --seed 42
```

## Available Benchmarks

| ID | Description | Difficulty | Focus Area |
|----|-------------|------------|------------|
| `fizzbuzz` | Classic FizzBuzz (1-100) | Easy | Control flow |
| `json_parse` | Parse JSON, filter, output | Medium | Data parsing |
| `pipeline` | stdin → transform → stdout | Medium | IO + lists |
| `cli_args` | Read file, process, sum | Hard | IO + FS |
| `adt_option` | Option/Maybe monad ops | Medium | Algebraic types |

## Latest Benchmark Results (v0.3.5)

**Model:** Claude Sonnet 4.5 (claude-sonnet-4-5-20250929)
**Date:** October 13, 2025
**Success Rate:** 10/19 (52.6%)

| Benchmark | Compile | Runtime | Output | Status |
|-----------|---------|---------|--------|--------|
| adt_option | ✅ | ✅ | ✅ | **PASS** |
| cli_args | ❌ | ❌ | ❌ | FAIL |
| error_handling | ✅ | ✅ | ✅ | **PASS** |
| fizzbuzz | ✅ | ✅ | ✅ | **PASS** |
| float_eq | ❌ | ❌ | ❌ | FAIL |
| higher_order_functions | ✅ | ✅ | ✅ | **PASS** |
| json_parse | ❌ | ❌ | ❌ | FAIL |
| list_comprehension | ✅ | ✅ | ❌ | FAIL |
| list_operations | ✅ | ✅ | ✅ | **PASS** |
| nested_records | ✅ | ✅ | ✅ | **PASS** |
| numeric_modulo | ✅ | ❌ | ❌ | FAIL |
| pattern_matching_complex | ❌ | ❌ | ❌ | FAIL |
| pipeline | ❌ | ❌ | ❌ | FAIL |
| record_update | ✅ | ✅ | ✅ | **PASS** |
| records_person | ✅ | ✅ | ✅ | **PASS** |
| recursion_factorial | ✅ | ✅ | ❌ | FAIL |
| recursion_fibonacci | ❌ | ❌ | ❌ | FAIL |
| string_manipulation | ✅ | ✅ | ✅ | **PASS** |
| targeted_repair_test | ✅ | ✅ | ✅ | **PASS** |

**Key Insights:**
- **Strong areas:** Records, ADTs, basic recursion, string operations
- **Needs improvement:** File I/O, JSON parsing, complex pattern matching, numeric operations
- **Compile success:** 16/19 (84.2%) - AI understands AILANG syntax well
- **Runtime success:** 14/19 (73.7%) - Most syntax issues resolved
- **Output correctness:** 10/19 (52.6%) - Logic errors in complex scenarios

## Command-Line Options

```bash
ailang eval [options]

Required:
  --benchmark <id>     Benchmark ID to run (e.g., fizzbuzz)

Optional:
  --langs <list>       Comma-separated languages (default: python,ailang)
  --model <name>       LLM model (default: gpt-4)
  --seed <n>           Random seed for reproducibility (default: 42)
  --output <dir>       Output directory (default: eval_results)
  --timeout <dur>      Execution timeout (default: 30s)
  --mock               Use mock AI agent (for testing)
```

## Understanding Results

### JSON Metrics

Each run produces a JSON file in `eval_results/` with:

```json
{
  "id": "fizzbuzz",
  "lang": "python",
  "model": "gpt-4",
  "seed": 42,
  "tokens": 290,
  "cost_usd": 0.0087,
  "compile_ok": true,
  "runtime_ok": true,
  "stdout_ok": true,
  "duration_ms": 120,
  "error_category": "none",
  "timestamp": "2025-10-02T12:34:56Z",
  "code": "..."
}
```

### Error Categories

- **`none`**: All checks passed
- **`compile_error`**: Syntax or type error
- **`runtime_error`**: Crashed during execution
- **`logic_error`**: Wrong output (compile/runtime OK)

### Reports

The `make eval-report` command generates:

**CSV** (`eval_results/summary.csv`):
```csv
benchmark,lang,model,seed,tokens,cost_usd,compile,runtime,stdout,duration_ms,error_type
fizzbuzz,python,gpt-4,42,290,0.0087,true,true,true,120,none
fizzbuzz,ailang,gpt-4,42,180,0.0054,true,true,true,80,none
```

**Markdown** (`eval_results/leaderboard.md`):
```markdown
# AILANG vs Python Benchmark Results

| Benchmark | Lang | Tokens | Cost | Compile | Run | Pass | Duration |
|-----------|------|--------|------|---------|-----|------|----------|
| fizzbuzz | python | 290 | $0.0087 | ✅ | ✅ | ✅ | 0.12s |
| fizzbuzz | ailang | 180 | $0.0054 | ✅ | ✅ | ✅ | 0.08s |

## Summary
- **Avg Token Reduction:** 37.9%
- **AILANG Success Rate:** 100% (1/1)
- **Python Success Rate:** 100% (1/1)
```

## Creating Custom Benchmarks

### 1. Create YAML Spec

Create `benchmarks/my_benchmark.yml`:

```yaml
id: my_benchmark
description: "My custom benchmark"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO"]
difficulty: "medium"
expected_gain: "high"
prompt: |
  Write a program in <LANG> that:
  1. Does something interesting
  2. Produces specific output

  Output only the code, no explanations.
expected_stdout: |
  expected
  output
  here
```

### 2. Run It

```bash
ailang eval --benchmark my_benchmark --mock
```

### 3. Validate Output

The benchmark passes if:
- Code compiles without errors
- Executes without crashing
- Output matches `expected_stdout` (exact string match after trimming whitespace)

## Reproducibility

### Deterministic Runs

Use `--seed` for reproducible results:

```bash
# Same seed → same model output (when supported)
ailang eval --benchmark fizzbuzz --model gpt-4 --seed 123
ailang eval --benchmark fizzbuzz --model gpt-4 --seed 123
```

### Comparing Across Models

```bash
# Run same benchmark with different models
ailang eval --benchmark adt_option --model gpt-4 --seed 42
ailang eval --benchmark adt_option --model gpt-3.5-turbo --seed 42
ailang eval --benchmark adt_option --model claude-3 --seed 42

# Generate unified report
make eval-report
```

## Best Practices

### 1. Start with Mock Mode
```bash
# Verify benchmark works before using real API
ailang eval --benchmark my_benchmark --mock
```

### 2. Use Consistent Seeds
```bash
# Always use same seed for fair comparisons
ailang eval --benchmark fizzbuzz --model gpt-4 --seed 42
```

### 3. Test Both Languages
```bash
# Always compare AILANG vs Python
ailang eval --benchmark fizzbuzz --langs python,ailang
```

### 4. Clean Between Runs
```bash
# Avoid confusion from stale results
make eval-clean
```

## Interpreting Results

### Token Reduction

**Good**: 30-50% reduction
- Indicates AILANG's concise syntax is effective
- Lower API costs
- Faster generation

**Neutral**: 0-30% reduction
- May indicate benchmark doesn't stress AILANG's strengths
- Consider more ADT-heavy or effect-heavy tasks

**Bad**: Negative reduction (AILANG uses more tokens)
- Investigate: Is AILANG verbose for this task?
- May indicate need for language improvements

### Success Rate

**Target**: 80%+ for both languages

If AILANG < 80%:
- Check runtime errors (capability issues? stdlib gaps?)
- Review generated code for common patterns
- May indicate missing language features

If Python < 80%:
- Check if task is too hard for AI
- Simplify prompt or expected output

### Error Patterns

Monitor `error_category` distribution:
- High `compile_error`: AI doesn't understand syntax
- High `runtime_error`: Missing builtins or capabilities
- High `logic_error`: AI misunderstood requirements

## Troubleshooting

### "OPENAI_API_KEY environment variable not set"

```bash
export OPENAI_API_KEY="sk-..."
```

### "ailang binary not found"

```bash
make install
# Ensure $GOPATH/bin is in PATH
```

### "Benchmark not found"

```bash
ls benchmarks/
# Check that <id>.yml exists
```

### Mock code doesn't match expected output

This is expected! Mock code is for testing the harness, not for real evaluation. Use `--model gpt-4` for actual benchmarks.

## Makefile Targets

```bash
make eval            # Run FizzBuzz with mock agent
make eval-report     # Generate CSV + Markdown reports
make eval-clean      # Remove all eval_results files
```

## Future Extensions

**v0.3.0:**
- Net/Clock benchmarks (once effects land)
- Auto-retry on failure
- Cost tracking across vendors

**v0.4.0:**
- Concurrency benchmarks (spawn, channels)
- Performance metrics
- Memory usage tracking

**v0.5.0:**
- Web leaderboard
- Historical trending
- CI integration

## Contributing

When adding benchmarks:
1. Use neutral prompts (no language bias)
2. Test with `--mock` first
3. Verify expected output is achievable
4. Document difficulty and expected gain
5. Submit PR with results from at least 2 models

---

**Next Steps:**
- Run your first benchmark: `ailang eval --benchmark fizzbuzz --mock`
- Create a custom benchmark for your domain
- Share results to help guide AILANG development

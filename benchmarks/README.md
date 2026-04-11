# AILANG Evaluation Benchmarks

This directory contains benchmark specifications for measuring AI code generation efficiency in AILANG vs Python.

## 🎯 Vision-Aligned Benchmarks

**NEW**: Benchmarks specifically designed to test AILANG's vision goals and differentiation from Python.

See **[VISION_BENCHMARKS.md](VISION_BENCHMARKS.md)** for detailed documentation on:
- How benchmarks map to vision goals
- Current vs future evaluation capabilities
- Metrics tracking and success criteria

| Benchmark | Vision Goal | Status |
|-----------|-------------|--------|
| `explicit_state_threading` | Explicit state vs implicit globals | 🆕 Ready |
| `deterministic_list_transform` | One canonical form vs multiple ways | 🆕 Ready |
| `effect_tracking_io_fs` | Explicit effect tracking (!: IO, FS) | 🆕 Ready |
| `effect_pure_separation` | Pure vs effectful separation | 🆕 Ready |
| `effect_composition` | Effect propagation in signatures | 🆕 Ready |
| `exhaustive_pattern_matching` | Total functions, no crashes | 🆕 Ready |
| `type_safe_record_access` | Static type safety vs runtime errors | 🆕 Ready |
| `referential_transparency` | Same input → same output | 🆕 Ready |
| `canonical_normalization` | Idiomatic, canonical code structure | 🆕 Ready |
| `no_runtime_crashes_option` | Option types prevent null errors | 🆕 Ready |
| `immutable_data_structures` | Immutable updates vs mutations | 🆕 Ready |

**Key Differentiators**:
- 🚀 **Explicit Effects**: AILANG forces effect declarations, Python doesn't track them
- 🎯 **Determinism**: AILANG has one way to do things, Python has many
- 🛡️ **Type Safety**: AILANG catches errors at compile time, Python at runtime
- ✅ **Totality**: AILANG requires exhaustive patterns, Python allows missing cases

## Available Benchmarks

### Core Benchmarks (Existing)

| ID | Description | Difficulty | Focus Area | Expected Gain | Status |
|----|-------------|------------|------------|---------------|--------|
| `fizzbuzz` | Classic FizzBuzz (1-100) | Easy | Control flow | Low | ✅ Passing |
| `recursion_factorial` | Factorial via recursion | Easy | Recursion | Low | ✅ Passing |
| `recursion_fibonacci` | Fibonacci via recursion | Easy | Recursion | Low | ✅ Passing |
| `records_person` | Record types with field access | Medium | Records | High | ✅ Passing |
| `adt_option` | Option/Maybe monad operations | Medium | Algebraic types | Very High | ✅ Passing |
| `float_eq` | Float equality comparisons | Easy | Type system | Low | ⚠️ AI variance |
| `numeric_modulo` | Modulo operator (%) | Easy | Operators | Low | ❌ Failing |
| `json_parse` | Parse JSON, filter, output | Medium | Data parsing | Medium | ❌ Failing |
| `pipeline` | stdin → transform → stdout | Medium | IO + lists | High | ❌ Failing |
| `cli_args` | Read file, process, sum | Hard | IO + FS | High | ❌ Failing |

### New Benchmarks (Testing Current Features)

| ID | Description | Difficulty | Focus Area | Expected Gain | Expected Status |
|----|-------------|------------|------------|---------------|-----------------|
| `list_operations` | List construction, pattern matching, recursion | Easy | Lists + recursion | High | Should Pass ✅ |
| `string_manipulation` | String concat, show(), comparisons | Easy | Strings | Medium | Should Pass ✅ |
| `nested_records` | Nested record access | Medium | Records | High | Should Pass ✅ |
| `higher_order_functions` | Compose, map, filter, currying | Medium | Functional programming | High | May Pass ⚠️ |
| `pattern_matching_complex` | Nested patterns, guards, Tree ADT | Hard | Pattern matching | Medium | Should Pass ✅ |

### New Benchmarks (Identifying Gaps)

| ID | Description | Difficulty | Focus Area | Expected Gain | Expected Status |
|----|-------------|------------|------------|---------------|-----------------|
| `record_update` | Record update syntax `{r \| field: value}` | Medium | Records | Very High | Will Fail ❌ (M-R5b) |
| `list_comprehension` | Map/filter/fold on lists | Hard | Stdlib | Very High | Will Fail ❌ (no stdlib) |
| `error_handling` | Result/Either type for errors | Medium | Error handling | High | Will Fail ❌ (no Result) |

## Quick Start

### Mock Mode (No API Key Required)

Test the harness with pre-written mock code:

```bash
# Single benchmark
ailang eval --benchmark fizzbuzz --mock

# Both languages
ailang eval --benchmark fizzbuzz --mock --langs python,ailang

# All existing benchmarks (mock)
for bench in fizzbuzz recursion_factorial recursion_fibonacci records_person adt_option \
             float_eq numeric_modulo json_parse pipeline cli_args; do
    ailang eval --benchmark $bench --mock --langs python,ailang
done

# All new benchmarks (mock)
for bench in list_operations string_manipulation nested_records higher_order_functions \
             pattern_matching_complex record_update list_comprehension error_handling; do
    ailang eval --benchmark $bench --mock --langs python,ailang
done

# Generate report
make eval-report
```

### Real API Mode

Requires API key from OpenAI or Anthropic:

```bash
# Set API key
export OPENAI_API_KEY="sk-..."
# or
export ANTHROPIC_API_KEY="..."

# Run single benchmark
ailang eval --benchmark fizzbuzz --model gpt-4 --seed 42

# Run all benchmarks (this will cost API credits!)
# Existing benchmarks
for bench in fizzbuzz recursion_factorial recursion_fibonacci records_person adt_option \
             float_eq numeric_modulo json_parse pipeline cli_args; do
    ailang eval --benchmark $bench --model gpt-4 --seed 42 --langs python,ailang
done

# New benchmarks (testing current features)
for bench in list_operations string_manipulation nested_records higher_order_functions \
             pattern_matching_complex; do
    ailang eval --benchmark $bench --model gpt-4 --seed 42 --langs python,ailang
done

# New benchmarks (expected to fail - identifies gaps)
for bench in record_update list_comprehension error_handling; do
    ailang eval --benchmark $bench --model gpt-4 --seed 42 --langs python,ailang
done

# Generate report
make eval-report
```

## Benchmark Spec Format

Each benchmark is a YAML file with this structure:

```yaml
id: my_benchmark
description: "Short description"
languages: ["python", "ailang"]  # Supported languages
entrypoint: "main"               # Entry function name
caps: ["IO", "FS"]               # Required capabilities
difficulty: "medium"             # easy | medium | hard
expected_gain: "high"            # low | medium | high | very_high
prompt: |
  Task description with <LANG> placeholder.

  Requirements:
  - Specific requirement 1
  - Specific requirement 2

  Output only the code, no explanations.
expected_stdout: |
  expected
  output
  here
```

## Creating Custom Benchmarks

1. **Create YAML file**: `benchmarks/my_benchmark.yml`
2. **Test with mock**: `ailang eval --benchmark my_benchmark --mock`
3. **Verify output**: Check `eval_results/*.json`
4. **Test with real API**: `ailang eval --benchmark my_benchmark --model gpt-4`
5. **Generate report**: `make eval-report`

## Prompt Engineering Tips

### Neutral Language

❌ **Bad** (biased):
```yaml
prompt: "Write Python code (the most popular language) that..."
```

✅ **Good** (neutral):
```yaml
prompt: "Write a program in <LANG> that..."
```

### Clear Requirements

❌ **Bad** (vague):
```yaml
prompt: "Do FizzBuzz"
```

✅ **Good** (specific):
```yaml
prompt: |
  Write a program in <LANG> that prints numbers 1-100:
  - Multiples of 3: print "Fizz"
  - Multiples of 5: print "Buzz"
  - Multiples of 15: print "FizzBuzz"
  - Others: print the number
```

### AILANG Hints (Phase 2)

After baseline tests reveal common errors, add AILANG-specific hints:

```yaml
prompt: |
  Write a program in <LANG> that...

  <LANG=AILANG> Additional context:
  - Use `let x = value in body` syntax
  - Import from stdlib/std/* for standard types
  - Effects declared with ! syntax: func() -> T ! {IO}
```

## Expected Results

### Phase 1 (Baseline Single-Shot)

**Goal**: Measure first-attempt quality

Expected outcomes:
- **Python**: Higher success rate (familiar to AI)
- **AILANG**: Lower success rate (unfamiliar syntax)
- **AILANG**: Fewer tokens per attempt (concise syntax)

**What we learn**:
- Which AILANG syntax confuses AI
- What documentation is missing
- How to improve prompts for Phase 2

### Phase 2 (Multi-Turn - Coming v0.3.0)

**Goal**: Measure total effort with iteration

Expected outcomes:
- **Python**: Fewer turns to success
- **AILANG**: More turns initially (learning curve)
- **AILANG**: Lower total tokens (after prompt improvements)

## Directory Structure

`benchmarks/` houses **three sibling suites** that measure different things and
should not be conflated:

```
benchmarks/
  README.md              # This file
  *.yml                  # AI eval specs (suite 1) — measure code-gen quality
  cross-language/        #   ↳ shared specs run against AILANG and Python
  runtime/               # Runtime micro-benchmarks (suite 2) — measure interpreter cost
  workloads/             # Latency-budget canonical workloads (suite 3) — measure user-visible p95

eval_results/            # AI-eval output (git-ignored)
  .gitignore
  *.json                 # Individual run results
  summary.csv            # Aggregated results
  leaderboard.md         # Human-readable report
```

| Suite | Question it answers | Measured by | Owner |
|-------|---------------------|-------------|-------|
| `*.yml` AI evals | "How well do AI models generate AILANG?" | `ailang eval-suite` | M-EVAL |
| `runtime/*.ail` | "Did this commit slow down a specific evaluator path?" | `make bench` (Go benchmarks) | perf-reviewer skill |
| `workloads/*.ail` | "Did this release blow the latency budget on realistic user workloads?" | `tools/bench_workloads.sh` | M-LAT-BUDGET |

The three suites are deliberately separate. AI evals stress code-gen quality
(can the model write the program at all?). Runtime micro-benchmarks isolate
single hot-loop interpreter changes. Latency-budget workloads run end-to-end
programs that mirror real user shapes — they are the only suite whose p95 is
treated as a release-gating SLO.

See [`workloads/README.md`](workloads/README.md) for the canonical workload
catalog and the latency-budget process.


## Cost Estimates

Approximate API costs (as of 2025):

| Model | Tokens/Benchmark | Cost/Benchmark | Full Suite (5) |
|-------|------------------|----------------|----------------|
| GPT-4 | 300 | $0.009 | $0.09 |
| GPT-3.5 | 300 | $0.0003 | $0.003 |
| Claude-3 | 300 | $0.0045 | $0.045 |

**Note**: Multi-turn evaluation (Phase 2) will cost 2-5x more due to iteration.

## Troubleshooting

### "OPENAI_API_KEY environment variable not set"

```bash
export OPENAI_API_KEY="sk-..."
```

### "Benchmark not found"

Check the benchmark ID matches the filename (without `.yml`):

```bash
ls benchmarks/
# Should see: fizzbuzz.yml, json_parse.yml, etc.
```

### Mock code doesn't pass tests

This is expected! Mock code is for testing the harness, not for real benchmarks. Use `--model gpt-4` for actual evaluation.

## Contributing

When adding benchmarks:

1. ✅ Use neutral prompts (no language bias)
2. ✅ Test with `--mock` first
3. ✅ Specify clear expected output
4. ✅ Document difficulty and expected gain
5. ✅ Test with at least 2 models
6. ✅ Submit PR with results

## Automated Design Doc Generation

**NEW** ✨ Automatically generate design documents from eval failures!

### Quick Start

```bash
# Full workflow: run evals → analyze → generate design docs
make eval-to-design

# Or run steps individually:
make eval-suite          # Run benchmarks
make eval-analyze        # Analyze failures, generate design docs
```

### How It Works

1. **Run Evals**: Benchmarks execute and save results to `eval_results/*.json`
2. **Analyze Patterns**: `ailang eval-analyze` groups failures by error pattern
3. **Generate Designs**: GPT-5 analyzes failures and creates design documents in `design_docs/planned/`

### Example Workflow

```bash
# Run eval suite with multiple models
make eval-suite

# Analyze results (dry-run to see issues first)
ailang eval-analyze --results eval_results/ --dry-run

# Generate design docs
ailang eval-analyze --results eval_results/ \
    --model gpt5 \
    --output design_docs/planned/ \
    --min-frequency 2

# Review generated designs
ls -lh design_docs/planned/
cat design_docs/planned/EVAL_ANALYSIS_*.md
```

### Options

```bash
ailang eval-analyze [options]

Options:
  --results <dir>         Directory with eval results (default: eval_results)
  --output <dir>          Output directory for design docs (default: design_docs/planned)
  --model <name>          LLM model for analysis (default: gpt5)
  --min-frequency <n>     Minimum failure count to report (default: 2)
  --categories <list>     Filter by category (compile_error,runtime_error,logic_error)
  --dry-run               Show issues without generating design docs
  --generate=false        Skip design doc generation (analysis only)
```

### What Gets Generated

For each issue pattern discovered:

1. **Design Document** (`YYYYMMDD_category_issue_name.md`)
   - Problem statement synthesized from error patterns
   - Root cause analysis
   - Proposed solution with implementation plan
   - Testing strategy
   - Success criteria
   - Estimated LOC and time

2. **Summary Report** (`EVAL_ANALYSIS_YYYYMMDD.md`)
   - Overview of all issues by impact
   - Links to generated design docs
   - Next steps for implementation

3. **Analysis Data** (`analysis_YYYYMMDD_HHMMSS.json`)
   - Machine-readable issue data
   - For further processing/tracking

### Cost Estimate

- ~$0.10-0.50 per design doc (GPT-5)
- Typical analysis: 1-3 design docs
- Total cost: $0.10-$1.50 per analysis run

## Next Steps

1. **Run mock tests**: `make eval`
2. **Set up API key**: `export OPENAI_API_KEY="..."`
3. **Run baseline suite**: See "Real API Mode" above
4. **Analyze results**: `make eval-report`
5. **Generate design docs**: `make eval-analyze` ✨ NEW
6. **Share findings**: Help improve AILANG documentation

---

**Documentation**: See [docs/guides/benchmarking.md](../docs/guides/benchmarking.md)
**Design Doc**: See [design_docs/20251002/m_eval_ai_benchmarking.md](../design_docs/20251002/m_eval_ai_benchmarking.md)
**Phase 2**: See [design_docs/20251002/m_eval2_agentic.md](../design_docs/20251002/m_eval2_agentic.md)

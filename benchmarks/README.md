# AILANG Evaluation Benchmarks

This directory contains benchmark specifications for measuring AI code generation efficiency in AILANG vs Python.

## Available Benchmarks

| ID | Description | Difficulty | Focus Area | Expected Gain |
|----|-------------|------------|------------|---------------|
| `fizzbuzz` | Classic FizzBuzz (1-100) | Easy | Control flow | Low |
| `json_parse` | Parse JSON, filter, output | Medium | Data parsing | Medium |
| `pipeline` | stdin → transform → stdout | Medium | IO + lists | High |
| `cli_args` | Read file, process, sum | Hard | IO + FS | High |
| `adt_option` | Option/Maybe monad operations | Medium | Algebraic types | Very High |

## Quick Start

### Mock Mode (No API Key Required)

Test the harness with pre-written mock code:

```bash
# Single benchmark
ailang eval --benchmark fizzbuzz --mock

# Both languages
ailang eval --benchmark fizzbuzz --mock --langs python,ailang

# All benchmarks (mock)
for bench in fizzbuzz json_parse pipeline cli_args adt_option; do
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
for bench in fizzbuzz json_parse pipeline cli_args adt_option; do
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

```
benchmarks/
  README.md              # This file
  fizzbuzz.yml           # Control flow benchmark
  json_parse.yml         # Data parsing benchmark
  pipeline.yml           # IO + lists benchmark
  cli_args.yml           # IO + FS benchmark
  adt_option.yml         # Algebraic types benchmark

eval_results/            # Output directory (git-ignored)
  .gitignore
  *.json                 # Individual run results
  summary.csv            # Aggregated results
  leaderboard.md         # Human-readable report
```

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

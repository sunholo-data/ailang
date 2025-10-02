# AI Evaluation Framework (M-EVAL)

This directory contains documentation for the AILANG AI evaluation framework, which measures how well AI models can generate AILANG code compared to Python.

## Overview

M-EVAL is designed to empirically measure the "AI teachability" of AILANG - one of the project's key success metrics. It compares AI code generation efficiency for AILANG vs Python across multiple benchmarks and models.

## Documentation

- **[baseline-tests.md](baseline-tests.md)** - Complete guide for running your first baseline tests
- **[model-configuration.md](model-configuration.md)** - How to configure and manage AI models
- **[benchmarking.md](../benchmarking.md)** - Technical details of the benchmarking system

## Quick Start

### Prerequisites

You need at least one of these:

1. **Anthropic Claude** (recommended)
   ```bash
   export ANTHROPIC_API_KEY="sk-ant-..."
   ```

2. **OpenAI GPT**
   ```bash
   export OPENAI_API_KEY="sk-proj-..."
   ```

3. **Google Gemini** (via Vertex AI)
   ```bash
   gcloud auth application-default login
   gcloud config set project YOUR_PROJECT_ID
   ```

### Run Baseline Tests

```bash
# Quick test with single model
ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5 --seed 42

# Full benchmark suite with all 3 models
./tools/run_benchmark_suite.sh

# Generate report
make eval-report
cat eval_results/leaderboard.md
```

## Available Models (October 2025)

All three models tested and working:

- ✅ **Claude Sonnet 4.5** (Anthropic) - Recommended for coding
- ✅ **GPT-5** (OpenAI) - Latest reasoning model
- ✅ **Gemini 2.5 Pro** (Google Vertex AI) - Multimodal capabilities

See [model-configuration.md](model-configuration.md) for details.

## Benchmarks

Current benchmark suite (5 benchmarks):

1. **fizzbuzz** - Control flow (if/else, loops)
2. **json_parse** - Data parsing and filtering
3. **pipeline** - IO + list operations
4. **cli_args** - Command-line argument handling (IO + FS)
5. **adt_option** - Algebraic data types (Option monad)

## Results Location

After running benchmarks:

- **JSON**: `eval_results/*.json` - Full details for each run
- **CSV**: `eval_results/summary.csv` - Aggregated data
- **Markdown**: `eval_results/leaderboard.md` - Human-readable report

## Key Metrics

The framework tracks:

- **Token efficiency** - Tokens used per attempt
- **Success rate** - % of attempts that compile and run correctly
- **Error categories** - compile_error, runtime_error, logic_error
- **Cost** - Estimated API cost per run

## Target KPIs

- **AI Teachability**: 80%+ success rate on simple benchmarks
- **Token Efficiency**: AILANG should use ≤ Python tokens (concise syntax)
- **Error Quality**: Clear error categories to identify documentation gaps

## Next Steps

1. Run baseline tests following [baseline-tests.md](baseline-tests.md)
2. Analyze results to identify common AI errors
3. Update AI prompt guide based on findings
4. Re-run benchmarks to measure improvement

## Phase 2: M-EVAL2 (Future)

The current framework (M-EVAL Phase 1) does single-shot evaluation. Phase 2 will add:

- Multi-turn evaluation with error feedback
- Integration with Claude Code and Gemini CLI
- Retry loops with corrective hints
- Success rate tracking across iterations

See design docs for details.

---

**Last Updated**: October 2, 2025 (v0.2.0-rc1)
**Framework Status**: ✅ Complete and operational
**Models Tested**: 3/3 working

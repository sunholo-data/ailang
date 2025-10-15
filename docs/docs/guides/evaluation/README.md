# AI Evaluation Framework (M-EVAL-LOOP v2.0)

This directory contains documentation for AILANG's AI evaluation framework, which measures how well AI models can generate AILANG code and provides automated feedback loops for continuous improvement.

## Overview

M-EVAL-LOOP is designed to empirically measure the "AI teachability" of AILANG - one of the project's key success metrics. It:
- Compares AI code generation across AILANG vs Python
- Tracks performance across multiple models and benchmarks
- Provides automated analysis and fix suggestions
- Validates fixes and measures improvements

## Quick Start

### Cost-Conscious Development (Recommended)

Use cheaper/faster models for daily development:

```bash
# Quick dev check (gpt5-mini, gemini-2-5-flash)
ailang eval-suite

# Create baseline
make eval-baseline

# Validate a fix
ailang eval-validate float_eq
```

**Cost**: ~$0.0003-0.002 per benchmark (5-10x cheaper than full suite)

### Full Comprehensive Testing

Use expensive models for release validation:

```bash
# Full suite (gpt5, claude-sonnet-4-5, gemini-2-5-pro)
ailang eval-suite --full

# Full baseline
FULL=true make eval-baseline

# Compare results
ailang eval-compare baselines/v0.3.0 current
```

**Cost**: ~$0.003-0.015 per benchmark

### Custom Model Selection

```bash
# Test specific models
ailang eval-suite --models gpt5,claude-sonnet-4-5

# With self-repair
ailang eval-suite --models gpt5 --self-repair
```

## Documentation

### Core Guides
- **[architecture.md](architecture.md)** - System architecture and command reference
- **[eval-loop.md](eval-loop.md)** - Automated evaluation and improvement workflow
- **[model-configuration.md](model-configuration.md)** - Model setup and pricing

### Implementation Details
- **[go-implementation.md](go-implementation.md)** - Native Go implementation guide
- **[migration-guide.md](migration-guide.md)** - Migration from bash to Go
- **[baseline-tests.md](baseline-tests.md)** - Running baseline tests

## Available Models (October 2025)

### Production Models (--full)
- ✅ **Claude Sonnet 4.5** (Anthropic) - $3/$15 per 1M tokens
- ✅ **GPT-5** (OpenAI) - $1.25/$10 per 1M tokens
- ✅ **Gemini 2.5 Pro** (Google) - $1.25/$10 per 1M tokens

### Development Models (default)
- ✅ **GPT-5 Mini** (OpenAI) - $0.25/$2 per 1M tokens (~1/5 price)
- ✅ **Gemini 2.5 Flash** (Google) - $0.30/$2.50 per 1M tokens (~1/4 price)

See [model-configuration.md](model-configuration.md) for setup details.

## Prerequisites

Set up at least one model's API key:

```bash
# Anthropic Claude (recommended for coding)
export ANTHROPIC_API_KEY="sk-ant-..."

# OpenAI GPT
export OPENAI_API_KEY="sk-proj-..."

# Google Gemini (Application Default Credentials)
gcloud auth application-default login
gcloud config set project YOUR_PROJECT_ID
# OR set API key
export GOOGLE_API_KEY="..."
```

## Commands Overview

| Command | Purpose | Example |
|---------|---------|---------|
| `ailang eval-suite` | Run benchmarks | `ailang eval-suite --full` |
| `ailang eval-compare` | Compare two runs | `ailang eval-compare baseline current` |
| `ailang eval-validate` | Validate specific fix | `ailang eval-validate records_person` |
| `ailang eval-matrix` | Generate performance matrix | `ailang eval-matrix results/ v0.3.0` |
| `ailang eval-summary` | Export to JSONL | `ailang eval-summary results/` |
| `ailang eval-report` | Generate reports | `ailang eval-report results/ v0.3.0` |

See [architecture.md](architecture.md) for detailed command reference.

## Benchmarks

Current benchmark suite (20 benchmarks):

**Core Language Features:**
- `fizzbuzz` - Control flow (if/else, loops)
- `recursion_factorial`, `recursion_fibonacci` - Recursion
- `higher_order_functions` - Functions as first-class values
- `list_operations`, `list_comprehension` - List manipulation
- `string_manipulation` - String operations

**Type System:**
- `records_person`, `nested_records` - Record types
- `record_update` - Record update syntax
- `adt_option` - Algebraic data types (Option)
- `pattern_matching_complex` - Pattern matching

**Effects & Capabilities:**
- `simple_print` - IO effects
- `cli_args` - Command-line arguments (IO + FS)
- `json_parse` - JSON parsing
- `error_handling` - Error propagation
- `pipeline` - Effect composition

**Edge Cases:**
- `float_eq` - Floating-point comparisons
- `numeric_modulo` - Modulo operations
- `targeted_repair_test` - Self-repair validation

## Results Location

After running benchmarks:

- **JSON**: `eval_results/baselines/VERSION/*.json` - Full details per run
- **Matrix**: `eval_results/baselines/VERSION/matrix.json` - Aggregated stats
- **Dashboard**: `docs/docs/benchmarks/performance.md` - Live leaderboard

## Key Metrics

The framework tracks:

- **Success Rate**: % of attempts that compile, run, and produce correct output
- **Token Efficiency**: Input/output tokens used per attempt
- **Cost**: Actual API cost based on model pricing
- **Error Categories**: compile_error, runtime_error, logic_error
- **Self-Repair**: First attempt success vs. repair success

## Typical Workflow

### 1. Create Baseline
```bash
make eval-baseline              # Quick baseline (dev models)
# OR
FULL=true make eval-baseline    # Full baseline (all models)
```

### 2. Make Changes
```bash
# Edit code, update prompts, etc.
```

### 3. Validate Fix
```bash
ailang eval-validate float_eq   # Check specific benchmark
```

### 4. Compare Results
```bash
ailang eval-compare baselines/v0.3.6 current
```

### 5. Update Dashboard
```bash
make benchmark-dashboard
```

## Natural Language Interface

You can also use natural language with Claude Code:

```
✅ "validate my fix for records"
✅ "how is AILANG performing?"
✅ "compare baseline to current"
✅ "generate an HTML report for v0.3.6"
```

The eval-orchestrator agent automatically routes to the correct commands.

## Architecture

M-EVAL-LOOP uses a two-tier architecture:

1. **Native Go Commands** - Fast, type-safe execution (90%+ test coverage)
2. **Smart Agents** - Natural language interface and workflow automation

See [architecture.md](architecture.md) for complete details.

## Target KPIs

- **AI Teachability**: 80%+ success rate on all benchmarks
- **Token Efficiency**: AILANG should use ≤ Python tokens (concise syntax)
- **Cost Efficiency**: Dev models viable for daily development
- **Error Quality**: Clear categorization for targeted improvements

## Development Cycle

```
1. make eval-baseline           # Store current state
2. <make changes>               # Implement features/fixes
3. ailang eval-validate BENCH   # Check specific fix
4. ailang eval-compare ...      # Full comparison
5. make benchmark-dashboard     # Update public dashboard
6. Repeat!
```

---

**Version**: 2.0
**Last Updated**: October 15, 2025
**Framework Status**: ✅ Production Ready
**Models Tested**: 5/5 working (3 production + 2 dev)
**Test Coverage**: 90%+ for eval analysis tools

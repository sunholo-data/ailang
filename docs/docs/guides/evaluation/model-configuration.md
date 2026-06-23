# Model Configuration Guide

## Overview

AILANG evaluation system supports the latest AI models from OpenAI, Anthropic, and Google. Model configurations are centralized in [`internal/eval_harness/models.yml`](https://github.com/sunholo-data/ailang/blob/main/internal/eval_harness/models.yml) for easy updates when new versions are released.

---

## Current Models (October 2025)

### Default: Claude Sonnet 4.5 (Anthropic)

**Why Claude Sonnet 4.5 is the default:**
- Released September 29, 2025
- **Best coding model in the world** (Anthropic's claim)
- Optimized for complex agents and autonomous coding
- 1M token context window
- Competitive pricing: $3/$15 per million tokens

### Recommended Benchmark Suite

For comprehensive evaluation, run all three:

1. **GPT-5** (OpenAI)
   - Released: August 7, 2025
   - API Name: `gpt-5`
   - Strength: Reasoning, general intelligence
   - Pricing: ~$30/$60 per million (estimated)

2. **Claude Sonnet 4.5** (Anthropic) ⭐ **Default**
   - Released: September 29, 2025
   - API Name: `claude-sonnet-4-5-20250929`
   - Strength: Coding, agents, computer use
   - Pricing: $3/$15 per million

3. **Gemini 2.5 Pro** (Google)
   - Released: March 2025
   - API Name: `gemini-2.5-pro`
   - Strength: Math, science, reasoning
   - Pricing: ~$1/$2 per million (estimated)

### Development/Testing Models (Cheaper, Faster)

For rapid iteration and cost-conscious development:

1. **GPT-5 Mini** (OpenAI)
   - API Name: `gpt-5-mini`
   - ~1/5 the price of GPT-5
   - Pricing: $0.25/$2 per million

2. **Claude Haiku 4.5** (Anthropic) 🆕
   - Released: October 1, 2025
   - API Name: `claude-haiku-4-5-20251001`
   - Fastest and most cost-effective Claude model
   - ~3x cheaper than Sonnet
   - Pricing: $1/$5 per million

3. **Gemini 2.5 Flash** (Google)
   - API Name: `gemini-2.5-flash`
   - 4x cheaper than Gemini Pro
   - Pricing: $0.30/$2.50 per million

---

## Quick Start

### 1. Set API Keys

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Anthropic (recommended)
export ANTHROPIC_API_KEY="sk-ant-..."

# Google (Gemini) — the eval harness uses Vertex AI Application Default Credentials.
# Authenticate once with gcloud; GOOGLE_API_KEY is NOT used by `ailang eval`.
gcloud auth application-default login
gcloud config set project <your-gcp-project>

# (GOOGLE_API_KEY is only honored by the in-language AI builtin / serve-api,
# which supports AI Studio as an alternative to Vertex. Leave it unset for eval.)
```

### 2. List Available Models

```bash
make eval-models
# or
ailang eval --list-models
```

### 3. Run Single Benchmark

```bash
# With default model (Claude Sonnet 4.5)
ailang eval --benchmark fizzbuzz

# With specific models
ailang eval --benchmark fizzbuzz --model gpt5
ailang eval --benchmark fizzbuzz --model gemini-2-5-pro

# With cheaper/faster models (for development)
ailang eval --benchmark fizzbuzz --model claude-haiku-4-5  # 🆕 Fast & cheap
ailang eval --benchmark fizzbuzz --model gpt5-mini
ailang eval --benchmark fizzbuzz --model gemini-2-5-flash
```

### 4. Run Full Suite (All Models)

```bash
make eval-suite
# or
./tools/run_benchmark_suite.sh
```

This runs all 5 benchmarks (fizzbuzz, json_parse, pipeline, cli_args, adt_option) with all 3 models.

**Expected cost**: ~$0.15-0.30 total (5 benchmarks × 3 models × 2 languages)
**Expected time**: ~15-20 minutes (with rate limiting)

---

## Configuration File

Models are configured in [`internal/eval_harness/models.yml`](https://github.com/sunholo-data/ailang/blob/main/internal/eval_harness/models.yml):

```yaml
models:
  claude-sonnet-4-5:
    api_name: "claude-sonnet-4-5-20250929"
    provider: "anthropic"
    description: "Claude Sonnet 4.5 - best for coding"
    env_var: "ANTHROPIC_API_KEY"
    pricing:
      input_per_1k: 0.003
      output_per_1k: 0.015
```

### When to Update

**Update `internal/eval_harness/models.yml` when:**
- New model versions release (e.g., GPT-6, Claude 5)
- Pricing changes
- API names change (e.g., `gpt-5-2026-01-01`)

**How to update:**
1. Edit `internal/eval_harness/models.yml`
2. Add new model entry
3. Update `default:` if needed
4. Update `benchmark_suite:` list
5. Test with `ailang eval --list-models`

---

## Model Selection Strategy

### For Development/Testing
```bash
# Use GPT-5 mini (fastest, cheapest)
ailang eval --benchmark fizzbuzz --model gpt5-mini --mock
```

### For Baseline Data
```bash
# Use Claude Sonnet 4.5 (best balance)
ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5 --seed 42
```

### For Comprehensive Comparison
```bash
# Run all 3 models
make eval-suite
```

### For Multi-Harness Cross-Comparison (v0.15.0+)

`agent_suite` routes the same benchmarks through the supported CLI-subprocess
harnesses — Claude Code, Codex, and opencode — using the executor factory. One
command, identical result schema. (Gemini-family agent evals run through the
hosted [Managed Agents API](./harness-setup.md#managed-agents-api-managed_agents)
— it has no local CLI and is opt-in rather than part of the routine suite.)

```bash
# Preview routing without running (--dry-run):
ailang eval-suite --models agent_suite --benchmarks fizzbuzz --dry-run
# claude-sonnet-4-6  → claude executor
# gpt5-4             → codex executor
# opencode-haiku     → opencode executor

# Run the full cross-harness comparison:
ailang eval-suite --models agent_suite --benchmarks fizzbuzz

# Add μRAG A/B toggle:
ailang eval-suite --models agent_suite --benchmarks fizzbuzz --microrag=on
ailang eval-suite --models agent_suite --benchmarks fizzbuzz --microrag=off
```

**Harness-specific prerequisites:**
| Harness | Install | Auth |
|---------|---------|------|
| claude | `npm i -g @anthropic-ai/claude-code` | `ANTHROPIC_API_KEY` |
| managed_agents | (HTTP API, no CLI) | ADC (`gcloud auth application-default login`) |
| codex | `npm i -g @openai/codex` | `OPENAI_API_KEY` |
| opencode | `npm i -g opencode-ai` | `ANTHROPIC_API_KEY` (or provider-specific key) |

Harnesses without the CLI binary installed are **skipped gracefully** — the
eval harness logs a `SKIP` row rather than failing the suite.

**Local Ollama models with opencode:**
```bash
# Start Ollama server
ollama serve
ollama pull gemma4:latest

# Configure opencode (~/.config/opencode/opencode.jsonc):
# { "provider": { "ollama": { "npm": "@ai-sdk/openai-compatible",
#     "options": { "baseURL": "http://localhost:11434/v1" },
#     "models": { "gemma4:latest": { "name": "Gemma 4" } } } } }

# Run benchmark with local model:
ailang eval-suite --models opencode-haiku --benchmarks fizzbuzz \
  --model-override ollama/gemma4:latest
```

See `internal/executor/opencode/testdata/opencode_ollama_config.jsonc` for
the full Ollama provider config template.

### For Budget-Conscious Testing
```bash
# Start with mock mode (free)
ailang eval --benchmark fizzbuzz --mock

# Then run one model
ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5
```

---

## Pricing Comparison (October 2025)

| Model | Input (per 1K) | Output (per 1K) | Full Suite Cost |
|-------|----------------|-----------------|-----------------|
| GPT-5 | $0.03 | $0.06 | ~$0.15 |
| GPT-5 mini | $0.01 | $0.02 | ~$0.05 |
| Claude Sonnet 4.5 | $0.003 | $0.015 | ~$0.03 |
| Gemini 2.5 Pro | $0.001 | $0.002 | ~$0.01 |

**Full suite (all 3 models)**: ~$0.20-0.30

*Note: Prices are estimates. Check official documentation for current rates.*

---

## Model Capabilities

### GPT-5 (OpenAI)
- ✅ Reasoning with "minimal" mode
- ✅ Verbosity parameter
- ✅ Code generation
- ✅ Broad knowledge
- ⚠️ Most expensive

**Best for**: General-purpose benchmarks, reasoning tasks

### Claude Sonnet 4.5 (Anthropic)
- ✅ **Best coding model**
- ✅ Computer use (CLI/tool use)
- ✅ 30-hour autonomous operation
- ✅ 1M context (2M coming)
- ✅ Great price/performance

**Best for**: Coding benchmarks (⭐ **recommended**)

### Gemini 2.5 Pro (Google)
- ✅ Thinking/reasoning mode
- ✅ Strong in math/science
- ✅ 1M context (2M coming)
- ✅ Cheapest option
- ⚠️ Less proven in coding

**Best for**: Budget testing, math/science benchmarks

---

## Troubleshooting

### "Model not found"

```bash
# Check if model is in config
make eval-models

# If not, add to internal/eval_harness/models.yml
```

### "API key not set"

```bash
# Check which key is needed
ailang eval --list-models

# Set the appropriate key
export ANTHROPIC_API_KEY="sk-ant-..."
```

### "Rate limit exceeded"

```bash
# Add delays between runs (done automatically in suite script)
ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5
sleep 10
ailang eval --benchmark json_parse --model claude-sonnet-4-5
```

### Cost tracking

```bash
# Check cost in results
cat eval_results/summary.csv | awk -F, '{sum+=$6} END {print "Total: $" sum}'
```

---

## Adding New Models

When new models release, update `internal/eval_harness/models.yml`:

```yaml
# Example: GPT-6 release
gpt6:
  api_name: "gpt-6-2026-01-01"
  provider: "openai"
  description: "GPT-6 - next generation"
  env_var: "OPENAI_API_KEY"
  pricing:
    input_per_1k: 0.05
    output_per_1k: 0.10
  notes: |
    Released January 2026.
    Improved reasoning and coding.
```

Then rebuild and test:
```bash
make build
ailang eval --list-models
ailang eval --benchmark fizzbuzz --model gpt6 --seed 42
```

---

## Best Practices

1. **Always use `--seed 42`** for reproducible results
2. **Start with `--mock`** to test harness before using API credits
3. **Use `eval-suite`** for comprehensive model comparison
4. **Check `--list-models`** to see current configuration
5. **Update `models.yml`** when new versions release
6. **Track costs** with `summary.csv`

---

## Quick Commands Reference

```bash
# List models
make eval-models
ailang eval --list-models

# Single benchmark
ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5

# Full suite (all models, all benchmarks)
make eval-suite

# Generate report
make eval-report

# Clean results
make eval-clean
```

---

**Last Updated**: October 2, 2025
**Default Model**: Claude Sonnet 4.5 (Anthropic)
**Configuration**: [internal/eval_harness/models.yml](https://github.com/sunholo-data/ailang/blob/main/internal/eval_harness/models.yml)

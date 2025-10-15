# M-EVAL-LOOP Architecture (v2.0)

## Quick Reference

### Running Evaluations

```bash
# Quick dev eval (default: gpt5-mini, gemini-2-5-flash)
ailang eval-suite

# Full comprehensive eval (gpt5, claude-sonnet-4-5, gemini-2-5-pro)
ailang eval-suite --full

# Custom models
ailang eval-suite --models gpt5,claude-sonnet-4-5

# Create baseline
make eval-baseline                # Quick baseline (dev models)
FULL=true make eval-baseline      # Full baseline (all models)

# Compare results
ailang eval-compare eval_results/baselines/v0.3.0 eval_results/current

# Validate specific fix
ailang eval-validate float_eq
```

### Available Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `ailang eval-suite` | Run full benchmark suite | `ailang eval-suite --full` |
| `ailang eval-compare` | Compare two eval runs | `ailang eval-compare baseline current` |
| `ailang eval-validate` | Validate specific fix | `ailang eval-validate records_person` |
| `ailang eval-matrix` | Generate performance matrix | `ailang eval-matrix results/ v0.3.0` |
| `ailang eval-summary` | Export to JSONL | `ailang eval-summary results/` |
| `ailang eval-report` | Generate reports | `ailang eval-report results/ v0.3.0 --format=html` |

## Architecture Overview

```
User Input ("validate my fix")
    ↓
Smart Agent (eval-orchestrator)  ← Natural language interface
    ↓
Native Go Commands (ailang eval-*)  ← Fast, type-safe execution
    ↓
Results + Interpretation
```

## Tier 1: Native Go Commands

Location: `internal/eval_analysis/` + `internal/eval_harness/` + `cmd/ailang/`

### eval-suite: Full Benchmark Execution

**Key Flags:**
- `--full`: Use expensive models (gpt5, claude-sonnet-4-5, gemini-2-5-pro)
- `--models X,Y,Z`: Custom model list (default: gpt5-mini, gemini-2-5-flash)
- `--benchmarks X,Y,Z`: Specific tests (default: all)
- `--langs X,Y`: Target languages (default: python,ailang)
- `--parallel N`: Concurrent API calls (default: 5)
- `--self-repair`: Enable self-repair on errors
- `--output DIR`: Output directory (default: eval_results)

**Examples:**
```bash
# Quick dev check (cheap/fast)
ailang eval-suite

# Full validation (expensive)
ailang eval-suite --full

# Single model + self-repair
ailang eval-suite --models gpt5 --self-repair

# Custom subset
ailang eval-suite --models gpt5 --benchmarks fizzbuzz,json_parse
```

**Model Cost Comparison:**
- Dev models (default): ~$0.0003-0.002 per benchmark
- Full models (--full): ~$0.003-0.015 per benchmark
- **5-10x cheaper for day-to-day development**

### eval-compare: Diff Two Runs

```bash
ailang eval-compare baseline/ current/
```

Shows:
- Success rate changes
- Newly passing/failing tests
- Token usage deltas
- Cost differences

### eval-validate: Check Specific Fix

```bash
ailang eval-validate records_person [version]
```

Validates that a fix works by comparing current implementation against baseline.

### Other Commands

```bash
ailang eval-matrix results/ v0.3.0     # Aggregate stats
ailang eval-summary results/           # Export JSONL
ailang eval-report results/ v0.3.0 -f html  # Generate report
```

## Tier 2: Smart Agents

### eval-orchestrator (.claude/agents/eval-orchestrator.md)

Interprets natural language and routes to correct commands:

```
User: "validate the float_eq fix"
Agent: → ailang eval-validate float_eq
      → Interprets results
      → Suggests next steps
```

### eval-fix-implementer (.claude/agents/eval-fix-implementer.md)

Automates fix implementation from design docs:

```
User: "implement the float_eq fix"
Agent: → Reads design_docs/planned/EVAL_ANALYSIS_float_eq.md
      → Implements fix
      → Runs tests
      → Validates with ailang eval-validate
      → Reports metrics
```

## Make Targets (Convenience)

```bash
make eval-baseline              # Quick baseline
FULL=true make eval-baseline    # Full baseline
MODELS=X,Y make eval-baseline   # Custom baseline

make eval-suite                 # Run benchmarks
make eval-analyze               # Generate design docs
make eval-diff BASELINE=X NEW=Y # Compare runs
```

## User Experience

### Natural Language (Recommended)
```
✅ "validate my fix for records"
✅ "how is AILANG performing?"
✅ "compare baseline to current"
✅ "generate a release report"
```

### Direct Commands (Power Users)
```bash
ailang eval-validate records_person
ailang eval-compare baselines/v0.3.0 current
ailang eval-report results/ v0.3.0 --format=html
```

## Model Selection Strategy

### Default: Cheap & Fast (Dev Models)
- **Models**: gpt5-mini, gemini-2-5-flash
- **Cost**: ~1/5 of full suite
- **Use for**: Daily development, rapid iteration, CI checks
- **Command**: `ailang eval-suite` (no flags needed)

### Full Suite: Comprehensive (Production Models)
- **Models**: gpt5, claude-sonnet-4-5, gemini-2-5-pro
- **Cost**: Full price
- **Use for**: Release validation, final QA, baseline creation
- **Command**: `ailang eval-suite --full`

### Custom: Mix & Match
- **Models**: Your choice
- **Cost**: Varies
- **Use for**: Targeted testing, specific model evaluation
- **Command**: `ailang eval-suite --models X,Y,Z`

## File Organization

```
.claude/
  agents/
    eval-orchestrator.md          # Smart workflow router
    eval-fix-implementer.md       # Automated fix implementation
  EVAL_ARCHITECTURE.md            # This file

internal/
  eval_analysis/                  # Native Go implementation (2,070 LOC)
    types.go, loader.go, comparison.go, matrix.go, formatter.go,
    validate.go, export.go, *_test.go (90%+ coverage)
  eval_harness/                   # Benchmark execution
    models.yml                    # Model configurations

cmd/ailang/
  eval_suite.go                   # eval-suite command
  eval_tools.go                   # Other eval commands
  main.go                         # Command routing

tools/
  eval_baseline.sh                # Baseline creation helper

Makefile                          # Convenience targets
```

## Design Principles

1. **Native Go First**: Fast, type-safe, testable
2. **Smart Agents Layer**: Add intelligence without forcing syntax
3. **Cost-Conscious Defaults**: Cheap models for dev, expensive for release
4. **Flexible**: Natural language, direct commands, or make targets
5. **Separation of Concerns**: Execution vs. interpretation

## Why This Architecture?

### Performance
- Native Go = 5-10x faster than old bash scripts
- No jq/sed/awk overhead
- Parallel execution built-in

### Reliability
- 90%+ test coverage
- Type-safe (no division by zero!)
- Proper error handling

### Usability
- Natural language interface (agents)
- Power user direct commands
- Cost-conscious defaults

### Maintainability
- Clear layer boundaries
- Easy to test and debug
- No brittle bash scripts

---

**Version**: 2.1
**Updated**: 2025-10-15
**Status**: Production Ready

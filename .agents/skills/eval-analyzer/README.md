# Eval Analyzer Skill

Comprehensive analysis toolkit for AILANG evaluation baselines.

## Overview

This skill provides tools to analyze benchmark results, identify failure patterns, compare model performance, and generate actionable insights from eval baselines.

## Files

- **SKILL.md** - Main skill documentation with workflow and common commands
- **scripts/** - Helper scripts for analysis
  - `quick_summary.sh` - Fast overview using eval-matrix
  - `analyze_failures.sh` - Detailed failure categorization
  - `compare_models.sh` - Model performance comparison
  - `examine_code.sh` - Inspect generated code from failures
  - `examine_prompts.sh` - View prompts for specific benchmarks
  - `verify_prompt_accuracy.sh` - Check prompt vs implementation mismatches
- **resources/** - Reference materials
  - `jq_queries.md` - Extended library of jq query patterns
  - `failure_analysis_v0.3.16.md` - Example comprehensive analysis

## Quick Start

```bash
# Get high-level overview
.claude/skills/eval-analyzer/scripts/quick_summary.sh eval_results/baselines/v0.3.16

# Analyze failures
.claude/skills/eval-analyzer/scripts/analyze_failures.sh eval_results/baselines/v0.3.16

# Compare models
.claude/skills/eval-analyzer/scripts/compare_models.sh eval_results/baselines/v0.3.16

# Check for prompt bugs
.claude/skills/eval-analyzer/scripts/verify_prompt_accuracy.sh v0.3.16
```

## Key Features

- Automated failure categorization (compile_error, logic_error, runtime_error)
- Model performance comparison with cost analysis
- Prompt accuracy verification (catches false limitations)
- Generated code inspection for debugging
- Comprehensive jq query library for custom analysis

## When to Use

- After running eval baselines
- When investigating performance regressions
- Before creating new prompt versions (use verify_prompt_accuracy.sh!)
- When user asks "what's failing?" or "why did performance drop?"

## Integration

Works with:
- `post-release` skill (runs baselines)
- `ailang eval-*` commands (matrix, analyze, compare, summary)
- `prompts/versions.json` (prompt version tracking)

## Maintenance

All scripts use `set -euo pipefail` for robustness and include proper error handling.
SKILL.md kept under 300 lines for progressive disclosure.

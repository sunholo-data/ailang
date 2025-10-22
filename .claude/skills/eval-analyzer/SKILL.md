---
name: Eval Analyzer
description: Analyze evaluation baseline results, identify failure patterns, and generate actionable insights. Use after running eval baselines or when user asks to analyze eval results, check benchmarks, investigate failures, or understand what's failing.
---

# Eval Analyzer

Analyze AILANG evaluation baseline results to identify failure patterns, compare model performance, and generate actionable insights.

## Quick Start

**Most common usage:**
```bash
# User says: "Analyze the v0.3.16 eval results"
# This skill will:
# 1. Run eval-analyze to categorize failures
# 2. Generate summary with jq queries
# 3. Identify top failing benchmarks
# 4. Show model performance comparison
# 5. Provide actionable recommendations
```

## When to Use This Skill

Invoke this skill when:
- User asks to "analyze eval results", "check benchmarks", "what's failing"
- After running an eval baseline
- When investigating why benchmark performance changed
- User wants to understand failure patterns or model performance
- Comparing two versions of AILANG

## Key Eval Commands

All commands work on baseline directories like `eval_results/baselines/v0.3.16/`.

### 1. Quick Overview - `eval-matrix`

Shows comprehensive statistics with model/language breakdowns.

```bash
ailang eval-matrix eval_results/baselines/v0.3.16 0.3.16 | head -60
```

**Shows**: Overall stats, per-model performance, per-language breakdown, top error codes.

### 2. Detailed Analysis - `eval-analyze`

Categorizes failures and can generate design docs for issues.

```bash
# Dry run (no design docs, just analysis)
ailang eval-analyze -results eval_results/baselines/v0.3.16 -dry-run

# Full analysis with design doc generation
ailang eval-analyze -results eval_results/baselines/v0.3.16
```

**⚠️ CRITICAL**: Must use `-results` flag, NOT positional argument!

**Output**: Categorized failures (compile_error, logic_error, runtime_error) with frequency, affected benchmarks, models, and sample errors.

### 3. Query-Friendly Summary - `eval-summary`

Generates JSONL for easy querying with jq.

```bash
ailang eval-summary eval_results/baselines/v0.3.16
```

**Output**: `eval_results/baselines/v0.3.16/summary.jsonl`

### 4. Compare Versions - `eval-compare`

Shows what changed between two versions.

```bash
ailang eval-compare eval_results/baselines/v0.3.15 eval_results/baselines/v0.3.16
```

## Workflow

### Step 1: Get High-Level Overview

```bash
# Show overall statistics
ailang eval-matrix eval_results/baselines/v0.3.16 0.3.16 | head -60
```

**Look for:**
- Overall success rate (target: >60%)
- AILANG vs Python gap (current: ~54%)
- Model performance variance
- Top error codes

### Step 2: Identify Problem Areas

```bash
# Categorize all failures
ailang eval-analyze -results eval_results/baselines/v0.3.16 -dry-run
```

**Key metrics:**
- compile_error frequency (parse/syntax issues)
- logic_error frequency (wrong output)
- runtime_error frequency (crashes)
- Which benchmarks fail most

### Step 3: Deep Dive with jq Queries

```bash
# Generate queryable summary
ailang eval-summary eval_results/baselines/v0.3.16

# Set variable for convenience
SUMMARY=eval_results/baselines/v0.3.16/summary.jsonl

# Error code distribution
jq -s 'group_by(.err_code) | map({code: .[0].err_code, count: length}) | sort_by(-.count)' $SUMMARY

# Top 20 failing AILANG benchmarks
jq -s 'map(select(.lang == "ailang" and .stdout_ok == false)) | group_by(.id) | map({benchmark: .[0].id, failures: length, success_rate: ((6 - length) * 100 / 6 | floor)}) | sort_by(.success_rate) | .[:20]' $SUMMARY

# Model performance on AILANG
jq -s 'map(select(.lang == "ailang")) | group_by(.model) | map({model: .[0].model, success: map(select(.stdout_ok)) | length, total: length, rate: (map(select(.stdout_ok)) | length) / length * 100 | round}) | sort_by(-.rate)' $SUMMARY

# Cost analysis by model
jq -s 'group_by(.model) | map({model: .[0].model, total_cost: (map(.cost_usd) | add | . * 100 | round / 100), runs: length, avg_cost: ((map(.cost_usd) | add) / length * 1000 | round / 1000)}) | sort_by(-.total_cost)' $SUMMARY
```

### Step 4: Compare with Previous Version

```bash
# Show regressions and improvements
ailang eval-compare eval_results/baselines/v0.3.15 eval_results/baselines/v0.3.16
```

### Step 5: Generate Insights

Based on the data, identify:

1. **Systemic Issues**: Categories with >50 failures
2. **Model Patterns**: Which models struggle with which features
3. **Benchmark Hotspots**: Benchmarks with 100% failure rate
4. **Cost Efficiency**: Which models give best success/cost ratio
5. **Trends**: Improvements or regressions vs previous version

## Common Queries

**All queries assume:**
```bash
SUMMARY=eval_results/baselines/v0.3.16/summary.jsonl
```

**Error categories for AILANG:**
```bash
jq -s 'map(select(.lang == "ailang" and .stdout_ok == false)) | group_by(.error_category) | map({category: .[0].error_category, count: length}) | sort_by(-.count)' $SUMMARY
```

**Which models fail most on AILANG:**
```bash
jq -s 'map(select(.lang == "ailang")) | group_by(.model) | map({model: .[0].model, fail_rate: (map(select(.stdout_ok == false)) | length) / length * 100 | floor}) | sort_by(-.fail_rate)' $SUMMARY
```

**Benchmarks where all models fail:**
```bash
jq -s 'map(select(.lang == "ailang" and .stdout_ok == false)) | group_by(.id) | map({benchmark: .[0].id, failures: length}) | map(select(.failures == 6))' $SUMMARY
```

**Average tokens by model:**
```bash
jq -s 'group_by(.model) | map({model: .[0].model, avg_tokens: (map(.total_tokens) | add / length | floor)})' $SUMMARY
```

**First-attempt success rate:**
```bash
jq -s 'map(select(.lang == "ailang")) | {total: length, first_ok: map(select(.first_attempt_ok)) | length, rate: (map(select(.first_attempt_ok)) | length) / length * 100}' $SUMMARY
```

**Repair effectiveness:**
```bash
jq -s 'map(select(.repair_used == true)) | {total_repairs: length, successful: map(select(.repair_ok)) | length, rate: (map(select(.repair_ok)) | length) / length * 100}' $SUMMARY
```

## Key Metrics to Track

1. **Overall Success Rate**: AILANG vs Python gap (target: reduce below 50%)
2. **Error Code Distribution**:
   - PAR_001 (parse errors) - indicates prompt/syntax issues
   - WRONG_LANG - models writing Python instead of AILANG
   - IMPERATIVE - models using imperative patterns
3. **Model Performance**: Which models work best with AILANG
4. **Benchmark-Level**: Which benchmarks consistently fail
5. **Cost Efficiency**: Success rate per dollar spent
6. **Repair Success**: Is self-repair helping? (currently low)

## Common Issues

### Issue 1: "Total Runs: 6" instead of 408

**Symptom**: eval-analyze only finds 6 results

**Cause**: Used positional argument instead of `-results` flag

**Solution**:
```bash
# ❌ WRONG
ailang eval-analyze eval_results/baselines/v0.3.16

# ✅ CORRECT
ailang eval-analyze -results eval_results/baselines/v0.3.16
```

### Issue 2: Summary file not found

**Symptom**: jq queries fail with "file not found"

**Cause**: Need to run eval-summary first

**Solution**:
```bash
ailang eval-summary eval_results/baselines/v0.3.16
```

### Issue 3: Design docs not generated

**Symptom**: eval-analyze shows issues but doesn't create docs

**Cause**: Using `-dry-run` flag

**Solution**: Run without `-dry-run` to generate design docs

## Resources

### Common jq Patterns
See [`resources/jq_queries.md`](resources/jq_queries.md) for more query examples and patterns.

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (workflow + common commands)
2. **Execute as needed**: `ailang eval-*` commands (analysis happens externally)
3. **Load on demand**: `resources/jq_queries.md` (extended query library)

## Notes

- All eval commands work offline (no API calls for analysis)
- `eval-analyze` generates design docs using LLM (default: gpt5)
- Summary JSONL format is stable and queryable
- Use `-dry-run` to preview before generating design docs
- baseline directories typically at `eval_results/baselines/vX.X.X/`
- This skill complements `post-release` skill (which runs baselines)

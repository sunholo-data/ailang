---
name: eval-analyzer
description: Identify AILANG language gaps from agent struggles, analyze eval baselines, and generate actionable insights. PRIMARY PURPOSE is finding what stdlib/prompt improvements would help agents succeed. Use when analyzing eval results, checking benchmarks, or investigating failures.
---

# Eval Analyzer

**Primary goal**: Identify AILANG language gaps from what agents struggle with → drives stdlib additions and prompt improvements.

Secondary: Analyze eval baseline results, compare model performance, track success rates.

## Quick Start

**Language gap analysis** (most valuable):
```bash
# Find what stdlib/prompt improvements would help agents
.claude/skills/eval-analyzer/scripts/find_language_gaps.sh eval_results/baselines/v0.6.2

# Output shows:
# - Functions agents searched for but couldn't find
# - Undefined variable errors (hallucinated functions)
# - Type confusion patterns
# - Benchmarks with stuck loops (high turn count)
# - Mapping of hallucinated names to actual builtins
```

**Standard analysis:**
```bash
# User says: "Analyze the v0.3.24 eval results"
# This skill will:
# 1. Run find_language_gaps.sh to identify AILANG improvements needed
# 2. Run eval-analyze to categorize failures
# 3. Run agent KPIs to analyze efficiency
# 4. Identify top failing benchmarks
# 5. Generate actionable recommendations
```

**For agent evaluation analysis** (NEW - optimization focus):
```bash
# Step 1: Get efficiency metrics (turns, tokens, cost)
.claude/skills/eval-analyzer/scripts/agent_kpis.sh eval_results/baselines/v0.3.24

# Step 2: Investigate expensive benchmarks
.claude/skills/eval-analyzer/scripts/agent_transcripts.sh eval_results/baselines/v0.3.24 simple_print

# Step 3: Compare Python vs AILANG
./tools/compare_agents.sh eval_results/baselines/v0.3.24
```

**See [`resources/agent_optimization_guide.md`](resources/agent_optimization_guide.md) for complete optimization strategies.**

**Tier- and tag-aware analysis (v0.14.0+):**
```bash
# Per-tier pass rates + per-tag deltas + AILANG-only wins (one shot)
.claude/skills/eval-analyzer/scripts/category_analysis.sh eval_results/baselines/v0.13.0

# Saturation + refusal detection + tier-promotion signal
.claude/skills/eval-analyzer/scripts/benchmark_health.sh eval_results/baselines/v0.13.0

# The primitives above wrap these CLI flags (added in M-EVAL-SUITE-PREP):
ailang eval-matrix <dir> <ver> --by-tags         # per-tag AILANG vs Python delta
ailang eval-matrix <dir> <ver> --show-saturated  # benchmarks at 100% everywhere
ailang eval-matrix <dir> <ver> --ailang-wins     # AILANG passes, Python fails
```

Use these when:
- **`--by-tags`**: deciding whether a regression is a language-wide issue or
  concentrated in a single feature (e.g. `contracts` drops 15pp while
  `records` is flat → it's a contract-prover issue, not a general regression).
- **`--show-saturated`**: rotating out benchmarks that no longer
  discriminate; see [`benchmarks/CURATION.md`](../../../benchmarks/CURATION.md) §4.
- **`--ailang-wins`**: tracking AILANG's differentiation over time; a
  shrinking list means the language-advantage story is eroding.

**Tier definitions** (from `benchmarks/CURATION.md`):

| Tier | Expected pass | Run cost | Role |
|------|---------------|----------|------|
| `smoke` | ≥ 95% | seconds | Regression gate |
| `core` | 70–95% | minutes | **Headline metric** |
| `stretch` | 30–70% | minutes | Headroom / differentiation |
| `vision` | 0–50% | variable | Aspirational |

Always quote the **Core tier** pass rate when reporting release numbers — it
is what the dashboard shows as the headline and what promotion/demotion
rules are calibrated against.

## Language Gap Analysis (PRIMARY GOAL)

**The most valuable output of eval analysis is identifying AILANG language gaps** - what agents struggle with that reveals missing stdlib functions, undocumented features, or prompt gaps.

### Why This Matters

When an agent fails after many turns, the transcript reveals what it was *trying* to do:

```
[config_file_parser - 22 turns]
Turn 14: "Perfect! There's floatToInt in std/prelude" -- HALLUCINATED!
Turn 15: "It looks like floatToInt is a builtin, not from a module"
Turn 18: "Let me add a floatToInt using basic arithmetic" -- broken workaround
```

**Insight**: Agent knows what it needs (`floatToInt`) but can't find it → **stdlib gap**.

### Language Gap Workflow

```bash
# Step 1: Find stuck loops - agents searching for functions
cat eval_results/baselines/v0.6.2/agent/*_ailang_*.json | \
  jq -r 'select(.stdout_ok == false) | .stderr' | \
  grep -i "let me check\|what function\|is available\|undefined variable" | head -20

# Step 2: Check if builtin exists for hallucinated function
ailang builtins list | grep -i "float\|int"

# Step 3: Check if stdlib wrapper exists
grep -i "floatToInt" std/*.ail

# Step 4: If builtin exists but wrapper doesn't → Add stdlib wrapper
# Step 5: If wrapper exists but agent didn't know → Update prompt
```

### Gap Pattern Categories

| Agent Behavior | Gap Type | Fix |
|----------------|----------|-----|
| "Let me check what X is available" then fails | Missing stdlib wrapper | Add wrapper to std/ |
| Uses function that exists but wrong name | Undocumented | Document in `ailang prompt` |
| Tries Python syntax in AILANG | Prompt gap | Add AILANG examples to `ailang prompt` |
| 10+ turns on same type error | Type confusion | Add type examples to `ailang prompt` |
| `undefined variable: floatToInt` | Missing wrapper | Add `floatToInt = _float_to_int` |
| Doesn't know about debugging tools | Toolchain gap | Update `ailang devtools-prompt` |

**Two prompt types to check when diagnosing gaps:**
- `ailang prompt` — language syntax (how to write .ail files)
- `ailang devtools-prompt` — toolchain (how to debug, trace, test, monitor)

### Example Gap Report

After analysis, produce actionable output:

```markdown
## Missing Wrappers (builtin exists, wrapper doesn't)
| Function | Builtin | Add to | Impact |
|----------|---------|--------|--------|
| floatToInt | _float_to_int | std/math | 3 benchmarks |
| intToFloat | _int_to_float | std/math | 2 benchmarks |

## Undocumented (exists but agents don't know)
| Function | Module | Agents looked for |
|----------|--------|-------------------|
| substring | std/string | stringSlice, slice |
| contains | std/string | includes, has |

## Prompt Gaps (syntax confusion)
| Issue | Example | Add to prompt |
|-------|---------|---------------|
| Json vs string | `get(str, key)` fails | Json type examples |
| String not list | `match s { [a,b] => }` | String handling section |
```

**See [design_docs/planned/v0_6_5/m-eval-gap-analysis.md](../../../design_docs/planned/v0_6_5/m-eval-gap-analysis.md) for full analysis example.**

## When to Use This Skill

Invoke this skill when:
- User asks to "analyze eval results", "check benchmarks", "what's failing"
- After running an eval baseline
- When investigating why benchmark performance changed
- User wants to understand failure patterns or model performance
- Comparing two versions of AILANG
- **Identifying AILANG language gaps** from what agents struggle with

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

### 5. Fair Comparison (RECOMMENDED) - `fair_comparison.py`

**Use this for accurate version comparisons!** The `eval-compare` command may include duplicates or different model sets. This script normalizes data for apple-to-apples comparison.

```bash
.claude/skills/eval-analyzer/scripts/fair_comparison.py
```

**What it does:**
- Deduplicates runs (keeps last run per benchmark+model)
- Filters to dev models only (gpt5-mini, claude-haiku-4-5, gemini-2-5-flash)
- AILANG only (ignores Python results)
- Shows net fixes vs regressions
- Per-model breakdown

**Output:**
```
v0.4.0: 56/123 = 45.5%
v0.4.2: 59/123 = 48.0%
Delta:  +3 (+2.4pp)

✅ Fixed:   11 benchmarks
❌ Broken:  8 benchmarks
NET:       +3 benchmarks
```

**When to use:** Before making decisions based on eval results (e.g., reverting changes, merging PRs).

### 6. Validate Results - `validate_eval_results.py`

**Check for output corruption and race conditions** in eval results.

```bash
python3 tools/validate_eval_results.py eval_results/baselines/v0.4.2
```

**Checks:**
- Output corruption (fibonacci outputting "All results equal", etc.)
- Duplicate runs for same benchmark+model
- Code hash validation (if available)
- Success rate statistics

**When to use:** After running eval baselines, especially if results look suspicious.

## Agent Eval Analysis (v0.8.0+ - Chain-Based)

Agent eval results are stored as chains in `observatory.db`. Use `ailang eval-chains` for analysis:

### Chain-Based Analysis (Recommended)

```bash
# List recent agent eval chains
ailang eval-chains list

# View per-benchmark assessment (compile/runtime/stdout)
ailang eval-chains view <chain-id>

# Pass rate breakdown by model, language, benchmark
ailang eval-chains stats <chain-id>

# Show only failures with error details
ailang eval-chains failures <chain-id>

# Full chain with session/tool data
ailang chains view <chain-id> --spans

# Generate report from chain (same format as file-based)
ailang eval-report --from-chain <chain-id> v0.8.0 --format=json

# Compare two agent eval runs
ailang eval-compare --chain <id1> --chain <id2>
```

### Chat/Tool Data (Per-Executor)

Agent eval captures full tool/chat data per executor:

| Executor | Query | Data |
|----------|-------|------|
| **Claude** | `ailang chains view <id> --spans` | Full conversation via JSONL import (tool inputs, thinking, tool results) |
| **Gemini** | `ailang chains view <id> --spans` | Streaming tool capture (tool names, inputs, outputs) |

### Legacy File-Based Agent Analysis

For older agent eval results stored as JSON files (pre-v0.8.0):

### 1. Agent KPIs - Minimize Tokens & Turns

Shows efficiency metrics for agent runs - **key for optimizing language and prompts**.

```bash
.claude/skills/eval-analyzer/scripts/agent_kpis.sh eval_results/WITH_ALL_FIXES
```

**Goal**: Minimize agent turns and tokens → indicates clearer prompts and simpler language.

### 2. Agent Transcripts - View AILANG Conversations

```bash
.claude/skills/eval-analyzer/scripts/agent_transcripts.sh eval_results/WITH_ALL_FIXES
.claude/skills/eval-analyzer/scripts/agent_transcripts.sh eval_results/WITH_ALL_FIXES --failed-only
.claude/skills/eval-analyzer/scripts/agent_transcripts.sh eval_results/WITH_ALL_FIXES fizzbuzz
```

### 3. Python vs AILANG Comparison

```bash
./tools/compare_agents.sh eval_results/WITH_ALL_FIXES
```

## Standard Eval Workflow (Non-Agent)

### Step 1: Get High-Level Overview

```bash
# Show overall statistics
ailang eval-matrix eval_results/baselines/v0.3.16 0.3.16 | head -60
```

**Look for:**
- Core tier pass rate (the headline metric in v0.14.0+)
- AILANG vs Python gap on Core specifically — not on the full suite
- Model performance variance within dev_models vs extended_suite
- Top error codes and top refused benchmarks (see `DetectRefusal`)

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

### Step 3: Query with jq (Custom Analysis)

Use jq queries on summary.jsonl for custom analysis:

```bash
# Ensure summary exists
ailang eval-summary eval_results/baselines/v0.3.20

# AILANG-only success rate (all models)
jq -s 'map(select(.lang == "ailang")) |
  {total: length, success: (map(select(.stdout_ok == true)) | length),
   rate: ((map(select(.stdout_ok == true)) | length) * 100.0 / length)}' \
  eval_results/baselines/v0.3.20/summary.jsonl

# Dev models only (useful for prompt testing)
# Dev models are sourced from internal/eval_harness/models.yml — DO NOT hardcode.
DEV_MODELS_JQ=$(. .claude/skills/_shared/scripts/eval_lib.sh && \
  dev_models_csv | awk -F, '{for(i=1;i<=NF;i++) printf "\"%s\"%s", $i, (i<NF?",":"")}')
jq -s --argjson devs "[$DEV_MODELS_JQ]" \
  'map(select(.lang == "ailang" and (.model as $m | $devs | index($m)))) |
   {total: length, success: (map(select(.stdout_ok == true)) | length),
    rate: ((map(select(.stdout_ok == true)) | length) * 100.0 / length)}' \
  eval_results/baselines/$VERSION/summary.jsonl

# Check specific benchmark across all models
jq -s 'map(select(.benchmark == "explicit_state_threading" and .lang == "ailang")) |
  map({model, success: .stdout_ok, error: .error_category})' \
  eval_results/baselines/$VERSION/summary.jsonl

```

**For tier/tag aggregates, prefer the CLI** — `summary.jsonl` rows have no `tier` or
`tags` fields, so jq alone can't filter by tier. Use one of these instead:

```bash
# Per-tier pass rate (headline Core number + stretch/vision breakdown)
jq '.tiers' eval_results/baselines/$VERSION/latest.json  # dashboard JSON

# Tier-aware analysis surface (v0.14.0+)
ailang eval-matrix --by-tags          # pass rate grouped by the 12-tag taxonomy
ailang eval-matrix --show-saturated   # benchmarks where all langs ≥ 95% (demote candidates)
ailang eval-matrix --ailang-wins      # benchmarks where AILANG ≥ Python by ≥ 10pp (keep)
```

For more tier/tag jq recipes see [`resources/jq_queries.md`](resources/jq_queries.md) —
the "Tier- and tag-aware queries (v0.14.0+)" section.

**For more jq patterns**, see [`resources/jq_queries.md`](resources/jq_queries.md)

### Step 4: Deep Dive with Helper Scripts

Use the provided helper scripts for detailed code inspection:

```bash
# Failure analysis with error categorization
.claude/skills/eval-analyzer/scripts/analyze_failures.sh eval_results/baselines/v0.3.16

# Model performance comparison
.claude/skills/eval-analyzer/scripts/compare_models.sh eval_results/baselines/v0.3.16

# Examine specific benchmark failures
.claude/skills/eval-analyzer/scripts/examine_code.sh eval_results/baselines/v0.3.16 api_call_json
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

## Key Metrics to Track

1. **Core-tier pass rate** (v0.14.0+ headline metric) — the AILANG vs Python comparison
   is computed on the Core tier only. Don't average the full suite; it mixes smoke (trivial),
   vision (research-grade) and Core (real signal) and the number becomes meaningless.
2. **Error Code Distribution**:
   - PAR_001 (parse errors) — prompt/syntax issues
   - WRONG_LANG — models writing Python instead of AILANG
   - IMPERATIVE — models using imperative patterns
   - **Refusals** — use `DetectRefusal` (`internal/eval_analysis/refusal.go`); important to
     distinguish "the model refused" from "the model attempted but failed"
3. **Per-tag gaps** — run `ailang eval-matrix --by-tags` to see which areas of the 12-tag
   taxonomy we're losing in. Target the largest negative gaps first.
4. **Saturated benchmarks** — `--show-saturated` shows benchmarks where *every* language
   passes consistently. These add no signal; demote or rotate out.
5. **AILANG-only wins** — `--ailang-wins` shows benchmarks where AILANG beats Python. Keep
   these as the "why AILANG" evidence set even if other metrics shift.
6. **Cost Efficiency** — success rate per dollar spent (separate dev vs extended models).
7. **Repair Success** — is agent mode's self-repair helping? (track delta: first-attempt vs final).

## Benchmark Rotation Philosophy

The eval suite is **curated, not accumulated**. The goal is high-signal benchmarks — ones
that tell us something about relative performance across AILANG, Python, and agent harnesses.
When reviewing a baseline, we ask three questions in order:

1. **Is a benchmark saturated across all langs?** (`--show-saturated`) → demote or retire.
2. **Is a benchmark an AILANG-only win?** (`--ailang-wins`) → keep as a value proxy, even
   if the Python rate changes later.
3. **Is a tag under-covered?** (`--by-tags` with low benchmark counts) → add targeted benchmarks.

Time-series continuity matters less than relative signal. Keeping a saturated benchmark
purely for regression defense has a cost (noise, runtime, agent budget) — do it sparingly,
and only where a past bug could plausibly regress.

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

## Helper Scripts

The skill includes helper scripts in `scripts/` directory:

### quick_summary.sh

Fast overview using eval-matrix.

```bash
.claude/skills/eval-analyzer/scripts/quick_summary.sh eval_results/baselines/v0.3.16
```

**Output**: Overall stats, model performance, language breakdown, top error codes.

### analyze_failures.sh

Detailed failure analysis with error categorization.

```bash
.claude/skills/eval-analyzer/scripts/analyze_failures.sh eval_results/baselines/v0.3.16 ailang
```

**Output**: Overall statistics, error categories, top failing benchmarks, model performance, error codes.

### compare_models.sh

Model-by-model performance comparison.

```bash
.claude/skills/eval-analyzer/scripts/compare_models.sh eval_results/baselines/v0.3.16
```

**Output**: Success rates, first-attempt vs final, cost analysis, token usage, best model per benchmark.

### examine_code.sh

Inspect generated code from specific benchmarks.

```bash
.claude/skills/eval-analyzer/scripts/examine_code.sh eval_results/baselines/v0.3.16 api_call_json
.claude/skills/eval-analyzer/scripts/examine_code.sh eval_results/baselines/v0.3.16 api_call_json gpt5
```

**Output**: Generated code, compiler errors, success status, error codes for each model run.

### examine_prompts.sh

View prompts used for specific benchmarks.

```bash
.claude/skills/eval-analyzer/scripts/examine_prompts.sh eval_results/baselines/v0.3.16 api_call_json
```

**Output**: System prompt, user prompt, success status for benchmark runs.

### verify_prompt_accuracy.sh

Check if prompt documentation matches actual implementation.

```bash
.claude/skills/eval-analyzer/scripts/verify_prompt_accuracy.sh v0.3.16
```

**Output**: Reports false limitations, undocumented features, and prompt-code mismatches.

**Use this**: After creating new prompt versions to catch documentation bugs!

## Resources

### Analysis Documents

- [`resources/failure_analysis_v0.3.16.md`](resources/failure_analysis_v0.3.16.md) - Comprehensive analysis of v0.3.16 eval results with root cause analysis

### Common jq Patterns

See [`resources/jq_queries.md`](resources/jq_queries.md) for more query examples and patterns.

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (workflow + commands + scripts)
2. **Execute as needed**: `ailang eval-*` commands and helper scripts
3. **Load on demand**: `resources/jq_queries.md`, analysis documents

## Notes

- All eval commands work offline (no API calls for analysis)
- `eval-analyze` generates design docs using LLM (default: gpt5)
- Summary JSONL format is stable and queryable
- Use `-dry-run` to preview before generating design docs
- baseline directories typically at `eval_results/baselines/vX.X.X/`
- This skill complements `post-release` skill (which runs baselines)

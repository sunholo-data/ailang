---
description: Run M-EVAL-LOOP validation and analysis workflows for AI code generation
allowed-tools:
  - Read
  - Bash(make:*)
  - Bash(bin/ailang:*)
  - Bash(./tools/*:*)
---

# Eval Loop Command

Execute M-EVAL-LOOP workflows for validating fixes, A/B testing prompts, and analyzing AI code generation results.

**Usage:** `/eval-loop <workflow> [options]`

**Available Workflows:**
- `baseline` - Store current results as baseline
- `validate <benchmark-id>` - Validate a specific fix
- `diff <baseline-dir> <new-dir>` - Compare two runs
- `prompt-ab <version-a> <version-b>` - A/B test prompts
- `summary <results-dir>` - Generate AI-friendly JSONL
- `matrix <results-dir> <version>` - Generate performance matrix
- `help` - Show this help

## Examples

```
/eval-loop baseline
/eval-loop validate float_eq
/eval-loop diff baselines/v0.3.0 after_fix
/eval-loop prompt-ab v0.3.0-baseline v0.3.0-hints
/eval-loop summary eval_results/baseline
/eval-loop matrix eval_results/baseline v0.3.0-alpha5
```

## Workflow Descriptions

### 1. Baseline Workflow

**Command:** `/eval-loop baseline`

**What it does:**
1. Runs full benchmark suite with current code
2. Stores results in `eval_results/baselines/<version>/`
3. Generates performance matrix
4. Creates baseline metadata with git commit info

**When to use:**
- Before starting work on a fix
- After completing a major milestone
- When you want to validate improvements

**Output:**
- Baseline directory with all results
- Performance matrix JSON
- Metadata file with git info

### 2. Validate Workflow

**Command:** `/eval-loop validate <benchmark-id>`

**What it does:**
1. Checks if baseline exists for the benchmark
2. Runs the benchmark with current code
3. Compares results and shows if fix worked
4. Returns exit code 0 if validated, 1 if failed

**When to use:**
- After fixing a specific bug
- To prove a fix works before committing
- During code review to show validation

**Possible outcomes:**
- ✅ **FIX VALIDATED**: Was failing, now passing
- ✗ **REGRESSION**: Was passing, now failing
- ⚠ **STILL FAILING**: Remains broken
- ℹ **NO CHANGE**: Was already passing

**Example:**
```bash
User: "I fixed the float comparison bug"
Assistant: Let me validate that fix
/eval-loop validate float_eq
# Output: "✓ FIX VALIDATED: Benchmark now passing!"
```

### 3. Diff Workflow

**Command:** `/eval-loop diff <baseline-dir> <new-dir>`

**What it does:**
1. Compares all benchmarks between two runs
2. Shows which benchmarks were fixed or broken
3. Calculates success rate deltas
4. Displays color-coded summary

**When to use:**
- After making changes to see overall impact
- To compare different versions
- To detect regressions

**Output:**
- List of fixed benchmarks
- List of broken benchmarks
- Success rate comparison
- Delta percentage

**Example:**
```bash
/eval-loop diff eval_results/baselines/v0.3.0 eval_results/after_fix
# Shows: Fixed (3), Broken (0), Success: 85% → 95% (+10%)
```

### 4. Prompt A/B Testing

**Command:** `/eval-loop prompt-ab <version-a> <version-b>`

**What it does:**
1. Runs all benchmarks with prompt version A
2. Runs all benchmarks with prompt version B
3. Compares success rates, token usage, repair effectiveness
4. Recommends which prompt to use

**When to use:**
- Testing a new teaching strategy
- Optimizing prompt for lower token cost
- Comparing error hint effectiveness

**Available prompt versions:**
- `v0.3.0-baseline`: Original teaching prompt (3,674 tokens)
- `v0.3.0-hints`: Enhanced with error patterns (4,538 tokens)

**Output:**
- 0-shot success rates for both
- Final success rates (with repair)
- Token usage comparison
- Cost comparison
- Recommendation based on delta

**Example:**
```bash
/eval-loop prompt-ab v0.3.0-baseline v0.3.0-hints
# Shows: v0.3.0-hints has +7% better 0-shot success
# Recommendation: "v0.3.0-hints shows significant improvement"
```

### 5. Summary Generation

**Command:** `/eval-loop summary <results-dir>`

**What it does:**
1. Converts all JSON results to JSONL format
2. One JSON object per line for easy streaming
3. Includes key metrics for AI analysis

**When to use:**
- Preparing data for AI analysis
- Querying with jq or other tools
- Exporting for external analysis

**Output fields:**
- id, lang, model, seed
- first_attempt_ok, repair_used, repair_ok
- err_code, error_category
- input_tokens, output_tokens, cost_usd
- timestamp, stderr

**Example queries:**
```bash
# Count successes
jq -s 'map(select(.stdout_ok == true)) | length' summary.jsonl

# Error distribution
jq -s 'group_by(.err_code) | map({code: .[0].err_code, count: length})' summary.jsonl

# Repair effectiveness
jq -s 'map(select(.repair_used == true)) | {total: length, success: map(select(.repair_ok == true)) | length}' summary.jsonl
```

### 6. Matrix Generation

**Command:** `/eval-loop matrix <results-dir> <version>`

**What it does:**
1. Generates performance matrix JSON with aggregates
2. Groups by model, benchmark, error code, language, prompt version
3. Tracks 0-shot vs 1-shot success rates
4. Includes token costs and efficiency metrics

**When to use:**
- After completing a milestone
- For historical performance tracking
- Comparing different versions

**Output structure:**
```json
{
  "version": "v0.3.0-alpha5",
  "total_runs": 50,
  "aggregates": {
    "0-shot_success": 0.85,
    "final_success": 0.92,
    "repair_success_rate": 0.70
  },
  "models": {...},
  "benchmarks": {...},
  "error_codes": [...]
}
```

## Command Implementation

When user invokes `/eval-loop <workflow>`, the assistant should:

1. **Parse the workflow and options**
2. **Validate prerequisites** (check if required directories/files exist)
3. **Execute the appropriate Makefile target or script**
4. **Parse and summarize the output**
5. **Provide actionable recommendations**

## Integration with Development Workflow

### Typical Development Cycle

```
1. User: "Let's fix the float comparison bug"
   Assistant: /eval-loop baseline

2. User: "I've made the changes"
   Assistant: /eval-loop validate float_eq
   # Output: ✓ FIX VALIDATED

3. User: "What else changed?"
   Assistant: /eval-loop diff baselines/v0.3.0 after_fix
   # Shows: Fixed (1), Still passing (48), No regressions

4. User: "Update the performance matrix"
   Assistant: /eval-loop matrix after_fix v0.3.0-alpha5
   # Generates: performance_tables/v0.3.0-alpha5.json
```

### Prompt Optimization Cycle

```
1. User: "Let's test if error hints help"
   Assistant: /eval-loop prompt-ab v0.3.0-baseline v0.3.0-hints

2. Output: "+7% improvement with hints prompt"
   Assistant: "The enhanced prompt significantly improves first-attempt success. Recommend adopting v0.3.0-hints as the new baseline."

3. User: "Update the active version"
   Assistant: [Edits prompts/versions.json to set active: "v0.3.0-hints"]
```

## Error Handling

If a workflow fails:
1. Show the error message clearly
2. Suggest troubleshooting steps
3. Provide fallback commands

**Common errors:**

- **"No baseline found"**
  - Solution: Run `/eval-loop baseline` first

- **"Benchmark not found"**
  - Solution: List available benchmarks with `ls benchmarks/*.yml`

- **"Directory not found"**
  - Solution: Provide correct path or create the directory

## Best Practices

1. **Store baseline before making changes** - Always validate against known state
2. **Use validate for specific fixes** - Faster than full diff
3. **Use diff for comprehensive review** - See everything that changed
4. **A/B test prompt changes** - Data-driven teaching improvements
5. **Generate matrix after milestones** - Track historical progress
6. **Export summary for analysis** - Enable AI-powered insights

## References

- [Eval Loop Guide](https://ailang-docs.vercel.app/guides/evaluation/eval-loop)
- [M-EVAL-LOOP Design Doc](../design_docs/planned/M-EVAL-LOOP_self_improving_feedback.md)
- [Makefile Targets](../../Makefile) (search for "eval-")

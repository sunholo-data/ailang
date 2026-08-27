# M-EVAL-LOOP: Self-Improving AI Feedback Loop

The M-EVAL-LOOP system transforms the AILANG eval harness from passive benchmarking into a **self-improving feedback loop** that teaches AI models and validates language improvements.

## Overview

**Status**: ✅ COMPLETE (v0.3.0-alpha5)

The eval loop closes the development cycle:
1. **Eval** → Run benchmarks, collect failures
2. **Analyze** → Generate design docs from patterns
3. **Iterate** → Review with multiple AI vendors
4. **Implement** → Fix language/compiler/stdlib
5. **Validate** → Re-run benchmarks, measure improvement
6. **Track** → Update performance tables

## Key Features

### 1. AI Self-Repair (Milestone 1)

AI models can retry failed code generation with error-specific guidance:

```bash
# Enable self-repair (single-shot retry)
ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5 --self-repair
```

**Error Taxonomy**: 6 error codes with repair hints
- `PAR_001`: Parse errors (missing semicolons)
- `TC_REC_001`: Record field not found
- `TC_INT_001`: Modulo on floats
- `EQ_001`: Wrong Eq dictionary
- `CAP_001`: Missing capability
- `MOD_001`: Undefined module/entrypoint

**Metrics Tracked**:
- `first_attempt_ok`: Did it work without error feedback?
- `repair_used`: Did self-repair trigger?
- `repair_ok`: Did self-repair succeed?
- `err_code`: Which error pattern matched?

### 2. Prompt A/B Testing (Milestone 2)

Compare different teaching strategies across AI models:

```bash
# Use specific prompt version
ailang eval --benchmark fizzbuzz --prompt-version v0.3.0-hints

# A/B comparison (full automation)
make eval-prompt-ab A=v0.3.0-baseline B=v0.3.0-hints

# List available versions
make eval-prompt-list
```

**Prompt Versions**:
- `v0.3.0-baseline`: Original teaching prompt (3,674 tokens)
- `v0.3.0-hints`: Enhanced with error pattern warnings (4,538 tokens)

**Hash Verification**: SHA256 prevents accidental modification mid-experiment

Frozen prompt versions are also protected by the CI gate `make check-prompt-freeze`. The gate
compares versions frozen at the branch's `origin/dev` merge-base with the working tree, rejecting
changes to their registry `file` or `hash` fields and to their Markdown bytes. It also requires the
frozen source entries and prompt bytes to agree with `cmd/ailang/prompts/`, which is embedded into
agent-mode binaries.

### 3. Fix Validation (Milestone 3)

Prove fixes work before committing:

```bash
# Store baseline
make eval-baseline

# Make code changes...
vim internal/eval/builtins.go

# Analyze results
ailang eval-analyze -results eval_results/current -dry-run
# Shows categorized failures and improvement suggestions

# Compare all changes
make eval-diff BASELINE=baselines/v0.3.0 NEW=after_fix
```

### 4. AI-Friendly Formats

Export results in formats optimized for AI analysis:

```bash
# JSONL (one JSON per line)
make eval-summary DIR=eval_results/baseline OUTPUT=summary.jsonl

# Performance matrix
make eval-matrix DIR=eval_results/baseline VERSION=v0.3.0-alpha5
```

**Query with jq**:
```bash
# Count successes
jq -s 'map(select(.stdout_ok == true)) | length' summary.jsonl

# Error distribution
jq -s 'group_by(.err_code) | map({code: .[0].err_code, count: length})' summary.jsonl

# Repair effectiveness
jq -s 'map(select(.repair_used == true)) | {total: length, success: map(select(.repair_ok == true)) | length}' summary.jsonl
```

## Complete Workflow

### Step 1: Store Baseline

Before making changes, store current results:

```bash
make eval-baseline
```

This runs all benchmarks and stores:
- Individual result JSON files
- Performance matrix with aggregates
- Baseline metadata with git commit

### Step 2: A/B Test Prompts (Optional)

Test if a new teaching strategy helps:

```bash
make eval-prompt-ab A=v0.3.0-baseline B=v0.3.0-hints
```

Output shows success rate comparison:
```
0-shot Success    85%           92%          +7%
Final Success     90%           95%          +5%
```

### Step 3: Implement Fix

Make code changes to fix identified issues:

```bash
vim internal/eval/builtins.go
make test
```

### Step 4: Analyze Results

Analyze failures and identify patterns:

```bash
ailang eval-analyze -results eval_results/current -dry-run
```

**Output shows**:
- Categorized failures (compile_error, runtime_error, logic_error)
- Affected benchmarks and models
- Frequency of each issue
- Sample error messages

### Step 5: Compare All Changes

See what else changed:

```bash
make eval-diff BASELINE=baselines/v0.3.0 NEW=after_fix
```

Output shows:
- ✓ Fixed benchmarks (3)
- ✗ Broken benchmarks (0)
- → Still passing (45)
- ⚠ Still failing (2)
- Success rate: 85% → 95% (+10%)

### Step 6: Update Performance Matrix

Track progress over time:

```bash
make eval-matrix DIR=after_fix VERSION=v0.3.0-alpha5
```

Generates `performance_tables/v0.3.0-alpha5.json` with:
- Aggregates by model, benchmark, error code
- 0-shot vs 1-shot success rates
- Token costs and efficiency
- Historical tracking

## Makefile Targets

### Self-Repair

```bash
make eval                    # Run single benchmark (mock)
make eval-suite              # Full suite, all models
make eval-suite-repair       # Full suite with self-repair
```

### Prompt Versioning

```bash
make eval-prompt-list        # Show available versions
make eval-prompt-hash        # Compute SHA256 hashes
make eval-prompt-ab A=X B=Y  # A/B comparison
```

### Validation Workflow

```bash
make eval-baseline                    # Store baseline
ailang eval-analyze -results X -dry-run  # Analyze failures
make eval-diff BASELINE=X NEW=Y      # Compare runs
make eval-summary DIR=<dir>          # Generate JSONL
make eval-matrix DIR=<dir> VERSION=X # Generate matrix
```

### Analysis

```bash
make eval-analyze            # Generate design docs from failures
make eval-analyze-fresh      # Force new docs (no dedup)
make eval-to-design          # Full workflow: eval → analyze
```

## Performance Metrics

The system tracks:

**0-shot metrics** (no error feedback):
- First attempt success rate
- Error distribution
- Token efficiency

**1-shot metrics** (with self-repair):
- Final success rate after repair
- Repair trigger rate
- Repair success rate

**Cost metrics**:
- Input/output tokens
- USD cost per benchmark
- Cost efficiency by model

**Time metrics**:
- Compilation time
- Execution time
- Total duration

## AI Agent Integration

### For Research

```bash
# Export for analysis
make eval-summary DIR=results OUTPUT=summary.jsonl

# Load into your tool
import jsonlines
with jsonlines.open('summary.jsonl') as reader:
    results = list(reader)

# Analyze
errors = [r for r in results if not r['stdout_ok']]
print(f"Error distribution: {Counter(e['err_code'] for e in errors)}")
```

### For Automation

```bash
# CI/CD integration - run targeted eval suite
ailang eval-suite --benchmarks float_eq --models gpt5-mini
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
  echo "Benchmark passing, safe to merge"
else
  echo "Benchmark failing"
  exit 1
fi
```

### For Historical Tracking

```bash
# Store matrix for each version
make eval-matrix DIR=results VERSION=v0.3.0-alpha5

# Compare versions
jq -s '[.[] | {version: .version, success: .aggregates."final_success"}]' \
  performance_tables/*.json
```

## Best Practices

1. **Store baseline before every fix** - Enables validation
2. **Run self-repair by default** - Measures teachability
3. **A/B test prompt changes** - Isolate what works
4. **Update performance tables after validation** - Track progress
5. **Review uncategorized errors monthly** - Expand taxonomy
6. **Keep benchmarks up-to-date** - Add new test cases

## Implementation Details

- **Total LOC**: ~2,960 (implementation + tests + scripts)
- **Development Time**: ~7 hours (3 milestones)
- **Files Modified**: 25+
- **Test Coverage**: 100% for new code
- **All tests passing**: ✅

## Automated Fix Implementation (NEW! 🚀)

**Milestone 4** adds fully automated fix implementation:

```bash
# Dry-run (preview what would be done)
make eval-auto-improve

# Actually implement the fix
make eval-auto-improve-apply
```

**How it works:**
1. Runs benchmarks (or uses recent results)
2. Analyzes failures → generates design docs
3. **AI agent reads design doc and implements fix** ⬅ NEW!
4. Runs tests to verify
5. Re-runs affected benchmarks
6. Shows before/after comparison

**Example workflow:**
```bash
# Preview
make eval-auto-improve
# Shows: Design doc preview, what would be done

# Apply
make eval-auto-improve-apply
# AI agent implements the fix automatically

# Analyze and compare
ailang eval-analyze -results eval_results/current -dry-run
make eval-diff
```

**Agent Integration:**
- Uses Claude Code Task agent (general-purpose)
- Pluggable design for future CLI/API agents
- Task file generated: `.eval_auto_improve_task.md`

**Safety:**
- Dry-run by default
- Tests must pass before accepting fix
- Human review before commit
- Automatic rollback on test failures

## Freezing a baseline cohort

The **cost-per-verified-success** KPI (dollars of benchmark spend per *verified*
success) is computed over a **frozen cohort**: the set of runs banked under one
immutable baseline id. Freezing and querying are two commands that share that id.

```bash
# 1. Freeze + run the cohort (METERED — see the cost caveat below)
ailang eval-suite --agent --langs ailang --verify \
  --models agent_suite \
  --benchmarks contract_bst_validate,contract_leap_year,contract_matrix_determinant,contract_rle_roundtrip,contract_roman_numeral,contract_sorted_merge,prompt_injection \
  --baseline v1.0 --seed 42 --no-rig-lock

# 2. Reproduce the published KPI from the banked data
ailang chains stats --cost-per-verified-success --baseline v1.0 --json --strict
```

Both commands take the **same** `--baseline` value. `--baseline` makes the run write
`chains.source_ref` as `<baseline>/<taskID>/<mode>[/<cond>]`, and the query selects
the cohort by that `<baseline>/` prefix.

### Rules

- **`--baseline` requires `--verify`.** Without verification a cohort can only ever
  report `zero_denominator` (the verified-success predicate needs
  `verify_verified > 0`), so `eval-suite` refuses *before* spending anything.
- **Allowed ids**: `^[A-Za-z0-9][A-Za-z0-9.-]*$` — letters, digits, `.` and `-`, not
  starting with `.` or `-`. `_` and `%` are rejected because the id is used as a SQL
  `LIKE` prefix where they are wildcards, and `/` is the `source_ref` separator.
  Good: `v1.0`, `v1.0-rc1`, `os-rolling.2`. Rejected: `v1_0`, `50%`, `v1.0/x`.
- **Without `--baseline` nothing changes.** The default `source_ref` and output are
  byte-identical to a normal run, and no manifest is written.

### The cohort manifest

A freeze run writes `<output-dir>/cohort_manifest.json` (the absolute path is
printed) and fills in its `run_window.completed_at` when the run finishes. It records
the **resolved** cohort — `models[]` comes from `agent_suite` in
[`internal/eval_harness/models.yml`](https://github.com/sunholo-data/ailang/blob/dev/internal/eval_harness/models.yml),
never a list hardcoded in Go — plus the resolved `benchmarks[]`, `seed`,
`prompt_version`, `trials`, `verify`, `verify_timeout`, the per-model `executors[]`,
the AILANG version and git commit, the `chain_id`, and a `cohort_hash`.

`cohort_hash` identifies the **cohort**, not the run: it covers the sorted
models/benchmarks/languages/conditions plus mode, seed, prompt version and trials,
and excludes timestamps, `git_commit` and `chain_id`. Consequences:

- Re-running the same cohort produces the **same** hash.
- Editing `agent_suite` in `models.yml` produces a **different** hash, so cohort drift
  is visible in the artifact instead of silent.
- **Re-freeze under a new id** (`v1.0-rc2`, …) rather than rewriting a published one.

### Cost-provenance caveat (known limitation)

The KPI's numerator is **not uniformly metered dollars**. Cost classification treats
any non-zero reported stage cost as authoritative, but the **subscription** `claude`
CLI reports a non-zero `total_cost_usd` even when nothing is billed — so on a rig with
no `ANTHROPIC_API_KEY`, `claude`-executor rows are **list-price equivalents, not
metered spend**, while the OpenRouter lanes (`opencode`, `codex`) are real money. A
cohort spanning both blends them under one `reported` label.

This is a **known, unaddressed limitation**, pending a product decision on whether a
subscription lane belongs in a metered KPI at all. Until then, read the manifest's
per-model `executors[]` to see which rows are subscription lanes before quoting a
number.

## Next Steps

Future enhancements could include:

- **Multi-agent coordination**: Multiple agents working on related fixes
- **Multi-shot repair**: Allow more than one retry
- **Error pattern learning**: Auto-generate repair hints from manual fixes
- **Cross-model comparison**: Compare GPT vs Claude vs Gemini on same benchmarks
- **Prompt evolution tracking**: Automated prompt optimization
- **Performance dashboards**: Web UI for historical trends

## References

- [Design Document](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/M-EVAL-LOOP_self_improving_feedback.md)
- [CHANGELOG Entry](https://github.com/sunholo-data/ailang/blob/dev/CHANGELOG.md)
- [Benchmarking Guide](../benchmarking.md)

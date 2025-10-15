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

### 3. Fix Validation (Milestone 3)

Prove fixes work before committing:

```bash
# Store baseline
make eval-baseline

# Make code changes...
vim internal/eval/builtins.go

# Validate fix
make eval-validate-fix BENCH=float_eq
# Output: "✓ FIX VALIDATED: Benchmark now passing!"

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

### Step 4: Validate Fix

Prove the fix works for specific benchmarks:

```bash
make eval-validate-fix BENCH=float_eq
```

**Possible outcomes**:
- ✅ **FIX VALIDATED**: Was failing, now passing
- ✗ **REGRESSION**: Was passing, now failing
- ⚠ **STILL FAILING**: Remains broken
- ℹ **NO CHANGE**: Was already passing

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
make eval-validate-fix BENCH=<id>    # Validate fix
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
# CI/CD integration
make eval-validate-fix BENCH=float_eq
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
  echo "Fix validated, safe to merge"
else
  echo "Fix failed or caused regression"
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

# Validate
make eval-validate-fix BENCH=<benchmark-id>
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
- [Evaluation Guide](./README.md)
- [Benchmarking Guide](../benchmarking.md)

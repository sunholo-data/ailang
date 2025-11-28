---
name: AILANG Post-Release Tasks
description: Run automated post-release workflow (eval baselines, dashboard, docs) for AILANG releases. Executes 46-benchmark suite (medium/hard/stretch) + full standard eval with validation and progress reporting. Use when user says "post-release tasks for vX.X.X" or "update dashboard". Fully autonomous with pre-flight checks.
---

# AILANG Post-Release Tasks

Run post-release tasks for an AILANG release: evaluation baselines, dashboard updates, and documentation.

## Quick Start

**Most common usage:**
```bash
# User says: "Run post-release tasks for v0.3.14"
# This skill will:
# 1. Run eval baseline (ALL 6 PRODUCTION MODELS, both languages) - ALWAYS USE --full FOR RELEASES
# 2. Update website dashboard (JSON with history preservation)
# 3. Extract metrics and UPDATE CHANGELOG.md automatically
# 4. Move design docs from planned/ to implemented/
# 5. Commit all changes to git
```

**🚨 CRITICAL: For releases, ALWAYS use --full flag by default**
- Dev models (without --full) are only for quick testing/validation, NOT releases
- Users expect full benchmark results when they say "post-release" or "update dashboard"
- Never start with dev models and then try to add production models later

## When to Use This Skill

Invoke this skill when:
- User says "post-release tasks", "update dashboard", "run benchmarks"
- After successful release (once GitHub release is published)
- User asks about eval baselines or benchmark results
- User wants to update documentation after a release

## Available Scripts

### `scripts/run_eval_baseline.sh <version> [--full]`
Run evaluation baseline for a release version.

**🚨 CRITICAL: ALWAYS use --full for releases!**

**Usage:**
```bash
# ✅ CORRECT - For releases (ALL 6 production models)
.claude/skills/post-release/scripts/run_eval_baseline.sh 0.3.14 --full
.claude/skills/post-release/scripts/run_eval_baseline.sh v0.3.14 --full

# ❌ WRONG - Only use without --full for quick testing/validation
.claude/skills/post-release/scripts/run_eval_baseline.sh 0.3.14
```

**Output:**
```
Running eval baseline for 0.3.14...
Mode: FULL (all 6 production models)
Expected cost: ~$0.50-1.00
Expected time: ~15-20 minutes

[Running benchmarks...]

✓ Baseline complete
  Results: eval_results/baselines/0.3.14
  Files: 264 result files
```

**What it does:**
- **Step 1**: Runs `make eval-baseline` (standard 0-shot + repair evaluation)
  - Tests both AILANG and Python implementations
  - Uses all 6 production models (--full) or 3 dev models (default)
  - Tests all benchmarks in benchmarks/ directory
- **Step 2**: Runs agent eval on full benchmark suite
  - **Current suite** (v0.4.8+): 46 benchmarks (trimmed from 56)
    - Removed: trivial benchmarks (print tests), most easy benchmarks
    - Kept: fizzbuzz (1 easy for validation), all medium/hard/stretch benchmarks
    - **Stretch goals** (6 new): symbolic_diff, mini_interpreter, lambda_calc,
      graph_bfs, type_unify, red_black_tree
    - Expected: ~55-70% success rate with haiku, higher with sonnet/opus
  - Uses haiku+sonnet (--full) or haiku only (default)
  - Tests both AILANG and Python implementations
- Saves combined results to eval_results/baselines/X.X.X/
- Accepts version with or without 'v' prefix

### `scripts/update_dashboard.sh <version>`
Update website benchmark dashboard with new release data.

**Usage:**
```bash
.claude/skills/post-release/scripts/update_dashboard.sh 0.3.14
```

**Output:**
```
Updating dashboard for 0.3.14...

1/5 Generating Docusaurus markdown...
  ✓ Written to docs/docs/benchmarks/performance.md

2/5 Generating dashboard JSON with history...
  ✓ Written to docs/static/benchmarks/latest.json (history preserved)

3/5 Validating JSON...
  ✓ Version: 0.3.14
  ✓ Success rate: 0.627

4/5 Clearing Docusaurus cache...
  ✓ Cache cleared

5/5 Summary
  ✓ Dashboard updated for 0.3.14
  ✓ Markdown: docs/docs/benchmarks/performance.md
  ✓ JSON: docs/static/benchmarks/latest.json

Next steps:
  1. Test locally: cd docs && npm start
  2. Visit: http://localhost:3000/ailang/docs/benchmarks/performance
  3. Verify timeline shows 0.3.14
  4. Commit: git add docs/docs/benchmarks/performance.md docs/static/benchmarks/latest.json
  5. Commit: git commit -m 'Update benchmark dashboard for 0.3.14'
  6. Push: git push
```

**What it does:**
- Generates Docusaurus-formatted markdown
- Updates dashboard JSON with history preservation
- Validates JSON structure (version matches input exactly)
- Clears Docusaurus build cache
- Provides next steps for testing and committing
- Accepts version with or without 'v' prefix

### `scripts/extract_changelog_metrics.sh [json_file]`
Extract benchmark metrics from dashboard JSON for CHANGELOG.

**Usage:**
```bash
.claude/skills/post-release/scripts/extract_changelog_metrics.sh
# Or specify JSON file:
.claude/skills/post-release/scripts/extract_changelog_metrics.sh docs/static/benchmarks/latest.json
```

**Output:**
```
Extracting metrics from docs/static/benchmarks/latest.json...

=== CHANGELOG.md Template ===

### Benchmark Results (M-EVAL)

**Overall Performance**: 59.1% success rate (399 total runs)

**By Language:**
- **AILANG**: 33.0% - New language, learning curve
- **Python**: 87.0% - Baseline for comparison
- **Gap: 54.0 percentage points (expected for new language)

**Comparison**: -15.2% AILANG regression from 0.3.14 (48.2% → 33.0%)

=== End Template ===

Use this template in CHANGELOG.md for 0.3.15
```

**What it does:**
- Parses dashboard JSON for metrics
- Calculates percentages and gap between AILANG/Python
- **Auto-compares with previous version from history**
- **Formats comparison text automatically** (+X% improvement or -X% regression)
- Generates ready-to-paste CHANGELOG template with no manual work needed

## Post-Release Workflow

### 1. Verify Release Exists

```bash
git tag -l vX.X.X
gh release view vX.X.X
```

If release doesn't exist, run `release-manager` skill first.

### 2. Run Eval Baseline

**🚨 CRITICAL: ALWAYS use --full for releases!**

**Correct workflow:**
```bash
# ✅ ALWAYS do this for releases - ONE command, let it complete
.claude/skills/post-release/scripts/run_eval_baseline.sh X.X.X --full
```

This runs all 6 production models with both AILANG and Python.

**Cost**: ~$0.50-1.00
**Time**: ~15-20 minutes

**❌ WRONG workflow (what happened with v0.3.22):**
```bash
# DON'T do this for releases!
.claude/skills/post-release/scripts/run_eval_baseline.sh X.X.X  # Missing --full!
# Then try to add production models later with --skip-existing
# Result: Confusion, multiple processes, incomplete baseline
```

**If baseline times out or is interrupted:**
```bash
# Resume with ALL 6 models (maintains --full semantics)
ailang eval-suite --full --langs python,ailang --parallel 5 \
  --output eval_results/baselines/X.X.X --skip-existing
```

The `--skip-existing` flag skips benchmarks that already have result files, allowing resumption of interrupted runs. But ONLY use this for recovery, not as a strategy to "add more models later".

### 3. Update Website Dashboard

**Use the automation script:**
```bash
.claude/skills/post-release/scripts/update_dashboard.sh X.X.X
```

**IMPORTANT**: This script automatically:
- Generates Docusaurus markdown (docs/docs/benchmarks/performance.md)
- Updates JSON with history preservation (docs/static/benchmarks/latest.json)
- Does NOT overwrite historical data - merges new version into existing history
- Validates JSON structure before writing
- Clears Docusaurus cache to prevent webpack errors

**Test locally (optional but recommended):**
```bash
cd docs && npm start
# Visit: http://localhost:3000/ailang/docs/benchmarks/performance
```

Verify:
- Timeline shows vX.X.X
- Success rate matches eval results
- No errors or warnings

**Commit dashboard updates:**
```bash
git add docs/docs/benchmarks/performance.md docs/static/benchmarks/latest.json
git commit -m "Update benchmark dashboard for vX.X.X"
git push
```

### 4. Extract Metrics for CHANGELOG

**Generate metrics template:**
```bash
.claude/skills/post-release/scripts/extract_changelog_metrics.sh X.X.X
```

This outputs a formatted template with:
- Overall success rate
- Standard eval metrics (0-shot, final with repair, repair effectiveness)
- Agent eval metrics by language
- **Automatic comparison with previous version** (no manual work!)

**Update CHANGELOG.md automatically:**
1. Run the script to generate the template
2. Insert the "Benchmark Results (M-EVAL)" section into CHANGELOG.md
3. Place it after the feature/fix sections and before the next version
4. Add context about specific improvements or regressions in "Key Findings"
5. Update any design doc references from `planned/` to `implemented/`

**Example section to insert:**
```markdown
### Benchmark Results (M-EVAL)

**Overall Performance**: 68.1% success rate (480 total runs)

**Standard Eval (0-shot + self-repair):**

| Metric | 0.4.4 | 0.4.5 | Change |
|--------|--------|--------|--------|
| **0-shot (first attempt)** | 55.6% | 64.0% (182/284) | **+8.4%** |
| **Final (with repair)** | 60.5% | 68.6% (195/284) | **+8.1%** |
| **Repair effectiveness** | +4.9pp | +4.6pp | -.3pp |
| **Python (final)** | 73.1% | 76.4% (208/272) | +3.3% |

**Agent Eval (multi-turn iterative problem solving):**

| Language | 0.4.4 | 0.4.5 | Change |
|----------|--------|--------|--------|
| **AILANG** | 92.1% | 100.0% (38/38) | **+7.9%** |
| **Python** | 100.0% | 100.0% (38/38) | 0% |

**Key Findings:**
- Major improvement in 0-shot performance
- Perfect agent eval score achieved
```

**IMPORTANT**: Don't just output the template - actually edit CHANGELOG.md to insert it!

### 4a. Analyze Agent Evaluation Results (NEW!)

**Check agent efficiency metrics:**
```bash
# Get KPIs (turns, tokens, cost by language)
.claude/skills/eval-analyzer/scripts/agent_kpis.sh eval_results/baselines/X.X.X
```

**Key metrics to track:**
- **Avg Turns**: AILANG vs Python (target: ≤1.5x gap)
- **Avg Tokens**: AILANG vs Python (target: ≤2.0x gap)
- **Success Rate**: Both should be 100% (agent mode corrects mistakes)

**If AILANG significantly worse than Python:**
1. Identify expensive benchmarks (from "Most Expensive" section)
2. View transcripts to understand issues: `.claude/skills/eval-analyzer/scripts/agent_transcripts.sh eval_results/baselines/X.X.X <benchmark>`
3. File optimization tasks for next release
4. See [eval-analyzer skill's agent_optimization_guide.md](../.claude/skills/eval-analyzer/resources/agent_optimization_guide.md) for strategies

**Example good result:**
```
🐍 Python: 10.6 avg turns, 58k tokens, 100% success
🔷 AILANG: 12.0 avg turns, 72k tokens, 100% success
Gap: 1.13x turns, 1.24x tokens ✅ (within target!)
```

**Example needs work:**
```
🐍 Python: 10.6 avg turns, 58k tokens, 100% success
🔷 AILANG: 18.0 avg turns, 178k tokens, 100% success
Gap: 1.7x turns, 3.0x tokens ⚠️ (needs optimization!)
```

### 5. Update Design Docs

- Move completed design docs from `design_docs/planned/` to `design_docs/implemented/vX_Y/`
- Update design docs with what was actually implemented (vs planned)
- Create new design docs for deferred features

### 6. Update Public Documentation

- Update `prompts/` with latest AILANG syntax
- Update website docs (`docs/`) with latest features
- Remove outdated examples or references
- Add new examples to website
- Update `docs/guides/evaluation/` if significant benchmark improvements
- **Update `docs/LIMITATIONS.md`**:
  - Remove limitations that were fixed in this release
  - Add new known limitations discovered during development/testing
  - Update workarounds if they changed
  - Update version numbers in "Since" and "Fixed in" fields
  - **Test examples**: Verify that limitations listed still exist and workarounds still work
    ```bash
    # Test examples from LIMITATIONS.md
    # Example: Test Y-combinator still fails (should fail)
    echo 'let Y = \f. (\x. f(x(x)))(\x. f(x(x))) in Y' | ailang repl

    # Example: Test named recursion works (should succeed)
    ailang run examples/factorial.ail

    # Example: Test polymorphic operator limitation (should panic with floats)
    # Create test file and verify behavior matches documentation
    ```
  - Commit changes: `git add docs/LIMITATIONS.md && git commit -m "Update LIMITATIONS.md for vX.X.X"`

## Resources

### Post-Release Checklist
See [`resources/post_release_checklist.md`](resources/post_release_checklist.md) for complete step-by-step checklist.

## Prerequisites

- Release vX.X.X completed successfully
- Git tag vX.X.X exists
- GitHub release published with all binaries
- `ailang` binary installed (for eval baseline)
- Node.js/npm installed (for dashboard, optional)

## Common Issues

### Eval Baseline Times Out
**Solution**: Use `--skip-existing` flag to resume:
```bash
bin/ailang eval-suite --full --skip-existing --output eval_results/baselines/vX.X.X
```

### Dashboard Shows "null" for Aggregates
**Cause**: Wrong JSON file (performance matrix vs dashboard JSON)
**Solution**: Use `update_dashboard.sh` script, not manual file copying

### Webpack/Cache Errors in Docusaurus
**Cause**: Stale build cache
**Solution**: Run `cd docs && npm run clear && rm -rf .docusaurus build`

### Dashboard Shows Old Version
**Cause**: Didn't run update_dashboard.sh with correct version
**Solution**: Re-run `update_dashboard.sh X.X.X` with correct version

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (YAML frontmatter + workflow overview)
2. **Execute as needed**: Scripts in `scripts/` directory (automation)
3. **Load on demand**: `resources/post_release_checklist.md` (detailed checklist)

Scripts execute without loading into context window, saving tokens while providing powerful automation.

## Recent Improvements (v0.3.15)

### Version Format Handling
All scripts now accept version with or without 'v' prefix:
- ✅ `0.3.15` works
- ✅ `v0.3.15` works
- Scripts pass version to underlying tools exactly as given
- No more "directory not found" errors due to version format mismatch

### Auto-Comparison in Metrics
The `extract_changelog_metrics.sh` script now:
- Automatically finds previous version from history
- Calculates AILANG performance difference
- Formats comparison text ("+X% improvement" or "-X% regression")
- Shows before/after percentages: "(48.2% → 33.0%)"
- **Zero manual jq queries needed!**

### Eliminated Manual Work
Before v0.3.15, you needed to:
- Run 15+ manual jq queries to extract metrics
- Manually compare with previous version
- Format comparison text yourself
- Handle version prefix inconsistencies

After v0.3.15:
- **3 scripts do everything automatically**
- No jq queries needed
- No manual calculations
- Version format doesn't matter

## Notes

- This skill follows Anthropic's Agent Skills specification (Oct 2025)
- Scripts handle 100% of automation (eval baseline, dashboard, metrics extraction)
- Can be run hours or even days after release
- Dashboard JSON preserves history - never overwrites historical data
- Always use `--full` flag for release baselines (all production models)
- Design doc migration requires manual review

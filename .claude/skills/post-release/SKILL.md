---
name: AILANG Post-Release Tasks
description: Run post-release tasks including eval baselines, dashboard updates, and documentation. Use after successful release to update benchmarks and website. Invoke when user says "post-release tasks", "update dashboard", or after completing a release.
---

# AILANG Post-Release Tasks

Run post-release tasks for an AILANG release: evaluation baselines, dashboard updates, and documentation.

## Quick Start

**Most common usage:**
```bash
# User says: "Run post-release tasks for v0.3.14"
# This skill will:
# 1. Run eval baseline (all models, both languages)
# 2. Update website dashboard (markdown + JSON with history)
# 3. Extract metrics for CHANGELOG
# 4. Guide through design doc and public doc updates
```

## When to Use This Skill

Invoke this skill when:
- User says "post-release tasks", "update dashboard", "run benchmarks"
- After successful release (once GitHub release is published)
- User asks about eval baselines or benchmark results
- User wants to update documentation after a release

## Available Scripts

### `scripts/run_eval_baseline.sh <version> [--full]`
Run evaluation baseline for a release version.

**Usage:**
```bash
# Dev models only (fast, cheap)
.claude/skills/post-release/scripts/run_eval_baseline.sh 0.3.14

# All production models (for releases)
.claude/skills/post-release/scripts/run_eval_baseline.sh 0.3.14 --full
```

**Output:**
```
Running eval baseline for v0.3.14...
Mode: FULL (all 6 production models)
Expected cost: ~$0.50-1.00
Expected time: ~15-20 minutes

[Running benchmarks...]

✓ Baseline complete
  Results: eval_results/baselines/v0.3.14
  Files: 264 result files
```

**What it does:**
- Runs `make eval-baseline` with appropriate flags
- Tests both AILANG and Python implementations
- Uses all 6 production models (--full) or 3 dev models (default)
- Saves results to eval_results/baselines/vX.X.X/

### `scripts/update_dashboard.sh <version>`
Update website benchmark dashboard with new release data.

**Usage:**
```bash
.claude/skills/post-release/scripts/update_dashboard.sh 0.3.14
```

**Output:**
```
Updating dashboard for v0.3.14...

1/5 Generating Docusaurus markdown...
  ✓ Written to docs/docs/benchmarks/performance.md

2/5 Generating dashboard JSON with history...
  ✓ Written to docs/static/benchmarks/latest.json (history preserved)

3/5 Validating JSON...
  ✓ Version: v0.3.14
  ✓ Success rate: 0.627

4/5 Clearing Docusaurus cache...
  ✓ Cache cleared

5/5 Summary
  ✓ Dashboard updated for v0.3.14
  ✓ Markdown: docs/docs/benchmarks/performance.md
  ✓ JSON: docs/static/benchmarks/latest.json

Next steps:
  1. Test locally: cd docs && npm start
  2. Visit: http://localhost:3000/ailang/docs/benchmarks/performance
  3. Verify timeline shows v0.3.14
  4. Commit: git add docs/docs/benchmarks/performance.md docs/static/benchmarks/latest.json
  5. Commit: git commit -m 'Update benchmark dashboard for v0.3.14'
  6. Push: git push
```

**What it does:**
- Generates Docusaurus-formatted markdown
- Updates dashboard JSON with history preservation
- Validates JSON structure
- Clears Docusaurus build cache
- Provides next steps for testing and committing

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

**Overall Performance**: 62.7% success rate (264 total runs)

**By Language:**
- **AILANG**: 42.1% - New language, learning curve
- **Python**: 83.3% - Baseline for comparison
- **Gap: 41.2 percentage points (expected for new language)**

**Comparison**: [Add comparison to previous version, e.g., "+3.5% AILANG improvement from v0.3.X"]

=== End Template ===

Use this template in CHANGELOG.md for v0.3.14
```

**What it does:**
- Parses dashboard JSON for metrics
- Calculates percentages and gap between AILANG/Python
- Generates ready-to-paste CHANGELOG template

## Post-Release Workflow

### 1. Verify Release Exists

```bash
git tag -l vX.X.X
gh release view vX.X.X
```

If release doesn't exist, run `release-manager` skill first.

### 2. Run Eval Baseline

**For releases (recommended):**
```bash
.claude/skills/post-release/scripts/run_eval_baseline.sh X.X.X --full
```

This runs all 6 production models with both AILANG and Python.

**Cost**: ~$0.50-1.00
**Time**: ~15-20 minutes

**If baseline times out or is interrupted:**
```bash
bin/ailang eval-suite --full --langs python,ailang --parallel 5 \
  --output ./eval_results/baselines/vX.X.X --self-repair --skip-existing
```

The `--skip-existing` flag skips benchmarks that already have result files, allowing resumption of interrupted runs.

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

**Use the automation script:**
```bash
.claude/skills/post-release/scripts/extract_changelog_metrics.sh
```

This outputs a ready-to-paste template with:
- Overall success rate
- AILANG-only rate (important!)
- Python baseline rate
- Gap analysis

**Update CHANGELOG.md:**
- Paste template into CHANGELOG.md under version section
- Add comparison to previous version (e.g., "+3.5% AILANG improvement")
- List specific improvements or regressions

**Example CHANGELOG entry:**
```markdown
### Benchmark Results (M-EVAL)

**Overall Performance**: 58.8% success rate (67/114 runs)

**By Language:**
- **AILANG**: 38.6% (22/57) - New language, learning curve
- **Python**: 78.9% (45/57) - Baseline for comparison
- **Gap**: 40.3 percentage points (expected for new language)

**Comparison**: +3.5% AILANG improvement from v0.3.7 (38.6% → 42.1%)
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

## Notes

- This skill follows Anthropic's Agent Skills specification (Oct 2025)
- Scripts handle most automation (eval baseline, dashboard, metrics extraction)
- Can be run hours or even days after release
- Dashboard JSON preserves history - never overwrites historical data
- Always use `--full` flag for release baselines (all production models)
- Design doc migration requires manual review

---
description: Run post-release tasks (benchmarks, dashboard, docs) after a release
allowed-tools:
  - Bash(git:*)
  - Bash(make:*)
  - Bash(gh:*)
  - Bash(jq:*)
  - Bash(ailang:*)
  - Bash(npm:*)
  - Bash(cd:*)
  - Edit
  - Read
---

# Post-Release Command

Run post-release tasks for an AILANG release: benchmarks, dashboard updates, and documentation.

**Usage:** `/post-release <version>`

**Example:** `/post-release 0.3.14`

**Prerequisites:** Must have already run `/release <version>` successfully.

## Steps to perform:

1. **Verify release exists:**
   - Check that v$1 tag exists: `git tag -l v$1`
   - Verify release on GitHub: `gh release view v$1`
   - If release doesn't exist, run `/release $1` first

2. **Update eval benchmarks** (M-EVAL-LOOP)
   - Run eval baseline to capture current performance: `make eval-baseline EVAL_VERSION=$1 FULL=true`
     - **IMPORTANT**: Use `FULL=true` for releases (runs all production models)
     - This ensures complete baseline data for dashboard charts and language comparison
     - Will run: claude-sonnet-4-5, claude-haiku-4-5, gpt5, gpt5-mini, gemini-2-5-flash, gemini-2-5-pro
     - Both languages: AILANG and Python (default, DO NOT override with LANGS=ailang)
     - Cost: ~$0.50-1.00 for full suite
   - **If eval baseline times out or is interrupted**, resume with:
     ```bash
     bin/ailang eval-suite --full --langs python,ailang --parallel 5 --output ./eval_results/baselines/$1 --self-repair --skip-existing
     ```
     - The `--skip-existing` flag skips benchmarks that already have result files
     - This allows resuming long-running eval baselines without losing progress
     - Checks for existing result files before running each benchmark
     - Added in v0.3.14 to handle timeout issues on slower machines
   - Compare to previous version: `ailang eval-compare eval_results/baselines/v<prev> eval_results/baselines/v$1`
   - **CRITICAL**: Calculate AILANG-only and combined metrics correctly:
     ```bash
     # Count AILANG-only results
     AILANG_PASS=$(jq -s 'map(select(.lang == "ailang" and .compile_ok and .runtime_ok and .stdout_ok)) | length' eval_results/baselines/v$1/*ailang*.json)
     AILANG_TOTAL=$(ls eval_results/baselines/v$1/*ailang*.json | wc -l)

     # Count combined results
     TOTAL_PASS=$(jq -s 'map(select(.compile_ok and .runtime_ok and .stdout_ok)) | length' eval_results/baselines/v$1/*.json)
     TOTAL_RUNS=$(ls eval_results/baselines/v$1/*.json | grep -v baseline.json | wc -l)

     # Count Python baseline (for comparison)
     PYTHON_PASS=$(jq -s 'map(select(.lang == "python" and .compile_ok and .runtime_ok and .stdout_ok)) | length' eval_results/baselines/v$1/*python*.json)
     PYTHON_TOTAL=$(ls eval_results/baselines/v$1/*python*.json | wc -l)
     ```
   - Update CHANGELOG.md with **SEPARATE AILANG and COMBINED metrics**:
     - Add section "### Benchmark Results (M-EVAL)"
     - **AILANG-only rate**: "AILANG: X/Y (Z%) - New language, learning curve"
     - **Python baseline**: "Python: X/Y (Z%) - Baseline for comparison"
     - **Combined rate**: "Overall: X/Y (Z%) - Combined Python+AILANG"
     - **Gap analysis**: "Gap: Z percentage points (AILANG vs Python)"
     - List improvements: "✓ Fixed: <benchmark_ids>"
     - List regressions: "✗ Regressed: <benchmark_ids>" (if any)
     - Show comparison: "+X% AILANG improvement" (not combined!)
   - **Example CHANGELOG format**:
     ```markdown
     ### Benchmark Results (M-EVAL)

     **Overall Performance**: 58.8% success rate (67/114 runs across 3 models × 20 benchmarks × 2 languages)

     **By Language:**
     - **AILANG**: 38.6% (22/57) - New language, learning curve
     - **Python**: 78.9% (45/57) - Baseline for comparison
     - **Gap**: 40.3 percentage points (expected for new language)

     **By Model:**
     - claude-sonnet-4-5: 63.2% (best performer)
     - gpt5: 57.9%
     - gemini-2-5-pro: 55.3%

     **Comparison**: +3.5% AILANG improvement from v0.3.7 (38.6% → 42.1%)
     ```
   - Store baseline: Already saved to `eval_results/baselines/v$1/` by make target
   - Commit baseline results: `git add eval_results/baselines/v$1/ && git commit -m "Add eval baseline for v$1" && git push`

3. **Update website dashboard** (CRITICAL - often forgotten!)
   - **Generate dashboard files** (markdown + JSON with history preservation):
     ```bash
     ailang eval-report eval_results/baselines/v$1 v$1 --format=docusaurus > docs/docs/benchmarks/performance.md
     ailang eval-report eval_results/baselines/v$1 v$1 --format=json
     ```
   - **IMPORTANT**: The `--format=json` output is shown on stdout but the tool ALSO writes to `docs/static/benchmarks/latest.json` with history preservation. Do NOT redirect to file (bypasses history logic).
   - **Verify JSON is valid**:
     ```bash
     jq -r '.version, .aggregates.finalSuccess' docs/static/benchmarks/latest.json
     # Should show: v$1 and success rate (e.g., 0.627 = 62.7%)
     ```
   - **Clear Docusaurus cache** (prevents webpack errors):
     ```bash
     cd docs && npm run clear
     ```
   - **Test locally** (optional but recommended):
     ```bash
     cd docs && npm start
     # Visit: http://localhost:3000/ailang/docs/benchmarks/performance
     # Verify: Timeline shows v$1, success rate matches, no errors
     ```
   - **Commit dashboard updates**:
     ```bash
     git add docs/docs/benchmarks/performance.md docs/static/benchmarks/latest.json
     git commit -m "Update benchmark dashboard for v$1"
     git push
     ```

4. **Update design docs**
   - Move design docs used into design_docs/implemented/
   - Update design docs used with what was implemented
   - If any features were missed or pushed to a future release, ensure they have new design_docs ready in design_docs/planned/

5. **Update public docs**
   - Ensure prompt in prompts/ reflects latest changes for AILANG syntax instruction
   - Ensure website docs in docs/ includes latest changes and is updated to remove any old references
   - Make sure latest examples are reflected on the website
   - Update docs/guides/evaluation/ with new benchmark results if significant improvements

6. **Final verification checklist**
   - [ ] Eval baseline complete for v$1
   - [ ] CHANGELOG.md updated with benchmark results (AILANG-only + combined)
   - [ ] Website dashboard shows v$1 as latest (http://localhost:3000/ailang/docs/benchmarks/performance)
   - [ ] Dashboard timeline chart includes v$1 data point
   - [ ] No webpack/cache errors when viewing dashboard
   - [ ] Dashboard JSON preserves history (check `docs/static/benchmarks/latest.json` has multiple versions)
   - [ ] Design docs moved to implemented/
   - [ ] Public docs updated with latest syntax/examples
   - [ ] All changes committed and pushed

## Notes

- This command can be run hours or even days after `/release`
- Eval baselines take ~10-15 minutes to run (cost: ~$0.50-1.00)
- Dashboard updates require Node.js/npm to be installed
- Design doc migration is manual review - take your time

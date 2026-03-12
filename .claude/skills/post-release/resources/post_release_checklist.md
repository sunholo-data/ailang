# Post-Release Checklist

## Prerequisites

- [ ] Release vX.X.X completed successfully
- [ ] Git tag vX.X.X exists
- [ ] GitHub release published with all binaries

## 1. Run Eval Baseline

Use the automation script:
```bash
.claude/skills/post-release/scripts/run_eval_baseline.sh X.X.X --full
```

**Manual alternative:**
```bash
make eval-baseline EVAL_VERSION=X.X.X FULL=true
```

**If baseline times out or is interrupted:**
```bash
bin/ailang eval-suite --full --langs python,ailang --parallel 5 \
  --output ./eval_results/baselines/vX.X.X --self-repair --skip-existing
```

- [ ] Eval baseline complete (eval_results/baselines/vX.X.X/)
- [ ] Results include both AILANG and Python
- [ ] All models run successfully

## 2. Verify Codebase Statistics

The release workflow auto-commits updated stats to `docs/static/codebase_stats.json`.
Verify it ran, or generate manually:

```bash
# Check if stats were auto-updated by CI
git pull
cat docs/static/codebase_stats.json | python3 -c "import json,sys; d=json.load(sys.stdin); print('Current:', d['current']['version'])"

# If stale, generate manually:
AILANG_VERSION=vX.X.X bash tools/generate_codebase_stats.sh
git add docs/static/codebase_stats.json
git commit -m "Update codebase statistics for vX.X.X"
git push
```

- [ ] `codebase_stats.json` shows vX.X.X as current
- [ ] History includes vX.X.X entry

## 3. Update Website Dashboard

Use the automation script:
```bash
.claude/skills/post-release/scripts/update_dashboard.sh X.X.X
```

This will:
- Generate docs/docs/benchmarks/performance.md
- Update docs/static/benchmarks/latest.json (preserves history)
- Validate JSON output
- Clear Docusaurus cache

- [ ] Markdown generated (docs/docs/benchmarks/performance.md)
- [ ] JSON updated (docs/static/benchmarks/latest.json)
- [ ] JSON validation passed
- [ ] Cache cleared

## 4. Test Dashboard Locally (Optional)

```bash
cd docs && npm start
# Visit: http://localhost:3000/ailang/docs/benchmarks/performance
```

- [ ] Timeline shows vX.X.X
- [ ] Success rate matches eval results
- [ ] No webpack/cache errors

## 5. Update Axiom Scorecard (if applicable)

Review axiom compliance if the release includes features that affect design axioms:

```bash
# View current scorecard
ailang axioms

# Edit scorecard JSON if needed
# File: docs/static/benchmarks/axiom_scorecard.json
```

Update if:
- A partial implementation became complete (+1 → +2)
- New feature aligns with an axiom (add evidence)
- Gaps were fixed (remove from gaps array)

Always add history entry:
```json
{
  "version": "vX.X.X",
  "date": "YYYY-MM-DD",
  "score": 18,
  "maxScore": 24,
  "percentage": 75.0,
  "notes": "Brief description of changes"
}
```

- [ ] Axiom scorecard reviewed
- [ ] Scores updated if applicable
- [ ] History entry added

## 6. Extract Metrics for CHANGELOG

Use the automation script:
```bash
.claude/skills/post-release/scripts/extract_changelog_metrics.sh
```

This outputs a CHANGELOG.md template with:
- Overall success rate
- AILANG-only rate
- Python baseline rate
- Gap analysis

- [ ] Metrics extracted
- [ ] CHANGELOG.md updated with benchmark results

## 7. Commit Dashboard Updates

```bash
git add docs/docs/benchmarks/performance.md docs/static/benchmarks/latest.json
git commit -m "Update benchmark dashboard for vX.X.X"
git push
```

- [ ] Files staged
- [ ] Committed
- [ ] Pushed

## 8. Verify Sprint JSON Tracking

Check that sprint state JSON files are properly completed:

```bash
# List sprint JSON files in project
ls -la .ailang/state/sprints/

# Verify sprint completion for this release
cat .ailang/state/sprints/sprint_<MILESTONE>.json | jq '.'
```

**For each sprint JSON file related to this release:**

- [ ] Sprint status is `"completed"` (not `"in_progress"`)
- [ ] All milestones marked with `"passes": true`
- [ ] `completed` timestamp is filled out for each milestone
- [ ] `actual_loc` values are present (not 0)
- [ ] `velocity` section has final metrics calculated
- [ ] `completion_summary` section exists with:
  - [ ] Total milestones count
  - [ ] Key deliverable counts (tests, files, functions, etc.)
  - [ ] Phase 2 message ID if applicable
  - [ ] Any important metrics specific to the sprint

**If sprint JSON is incomplete:**

1. Review sprint completion documents (e.g., `M-<MILESTONE>-SPRINT-COMPLETE.md`)
2. Update sprint JSON with correct values
3. Create tracked completion summary if missing (sprint JSONs are gitignored)

**Common issues:**
- Sprint left in `"in_progress"` status
- Milestones missing completion timestamps
- `actual_loc` not calculated
- `velocity.efficiency` not computed
- `completion_summary` section missing

## 9. Update Design Docs

- [ ] Move completed design docs to design_docs/implemented/vX_Y/
- [ ] Update design docs with what was actually implemented
- [ ] Create new design docs in design_docs/planned/ for deferred features

## 10. Update Public Documentation

- [ ] Ensure prompts/ reflects latest AILANG syntax
- [ ] Update website docs (docs/) with latest features
- [ ] Remove old references or outdated examples
- [ ] Add new examples to website if applicable
- [ ] Update docs/guides/evaluation/ with significant improvements
- [ ] **Update docs/LIMITATIONS.md**:
  - [ ] Remove limitations fixed in this release
  - [ ] Add new limitations discovered during development
  - [ ] Update workarounds if they changed
  - [ ] Update version numbers ("Since", "Fixed in" fields)
  - [ ] Test examples in LIMITATIONS.md still work/fail as documented
  - [ ] Commit: `git add docs/LIMITATIONS.md && git commit -m "Update LIMITATIONS.md for vX.X.X"`

## 11. Run Documentation Sync Check

Use the docs-sync skill to verify website accuracy:

```bash
# Check version constants match git tag
.claude/skills/docs-sync/scripts/check_versions.sh

# Audit design docs vs website claims
.claude/skills/docs-sync/scripts/audit_design_docs.sh

# Generate full sync report
.claude/skills/docs-sync/scripts/generate_report.sh
```

- [ ] Version constants match git tag (docs/src/constants/version.js)
- [ ] Teaching prompt references point to latest version
- [ ] Architecture pages have PLANNED banners for unimplemented features
- [ ] No unimplemented features claimed as current
- [ ] Examples referenced in website actually work

**If issues found:**
- [ ] Update version.js if stale
- [ ] Add PLANNED banners to theoretical feature pages
- [ ] Move implemented features from roadmap to current sections
- [ ] Fix broken example references
- [ ] Commit: `git add docs/ && git commit -m "docs: sync website with vX.X.X"`

## Final Verification

- [ ] Eval baseline complete for vX.X.X
- [ ] CHANGELOG.md has benchmark results (AILANG + combined)
- [ ] Website dashboard shows vX.X.X as latest
- [ ] Dashboard timeline includes vX.X.X data point
- [ ] Dashboard JSON preserves history (multiple versions)
- [ ] **Axiom scorecard reviewed and updated if applicable**
- [ ] **Axiom history entry added for this version**
- [ ] Design docs moved to implemented/
- [ ] Public docs updated
- [ ] **docs/LIMITATIONS.md updated and tested**
- [ ] **docs-sync report shows no critical issues**
- [ ] All changes committed and pushed

## Notes

- Can be run hours or days after release
- Eval baselines cost ~\$0.50-1.00 (full) or ~\$0.10-0.20 (dev)
- Dashboard requires Node.js/npm installed
- Design doc migration is manual review

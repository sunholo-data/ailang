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

## 2. Update Website Dashboard

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

## 3. Test Dashboard Locally (Optional)

```bash
cd docs && npm start
# Visit: http://localhost:3000/ailang/docs/benchmarks/performance
```

- [ ] Timeline shows vX.X.X
- [ ] Success rate matches eval results
- [ ] No webpack/cache errors

## 4. Extract Metrics for CHANGELOG

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

## 5. Commit Dashboard Updates

```bash
git add docs/docs/benchmarks/performance.md docs/static/benchmarks/latest.json
git commit -m "Update benchmark dashboard for vX.X.X"
git push
```

- [ ] Files staged
- [ ] Committed
- [ ] Pushed

## 6. Update Design Docs

- [ ] Move completed design docs to design_docs/implemented/vX_Y/
- [ ] Update design docs with what was actually implemented
- [ ] Create new design docs in design_docs/planned/ for deferred features

## 7. Update Public Documentation

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

## Final Verification

- [ ] Eval baseline complete for vX.X.X
- [ ] CHANGELOG.md has benchmark results (AILANG + combined)
- [ ] Website dashboard shows vX.X.X as latest
- [ ] Dashboard timeline includes vX.X.X data point
- [ ] Dashboard JSON preserves history (multiple versions)
- [ ] Design docs moved to implemented/
- [ ] Public docs updated
- [ ] **docs/LIMITATIONS.md updated and tested**
- [ ] All changes committed and pushed

## Notes

- Can be run hours or days after release
- Eval baselines cost ~\$0.50-1.00 (full) or ~\$0.10-0.20 (dev)
- Dashboard requires Node.js/npm installed
- Design doc migration is manual review

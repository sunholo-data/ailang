# Markdown File Audit & Organization Recommendations

**Date**: 2025-11-01
**Purpose**: Identify temporary status files and organize documentation structure

---

## Executive Summary

**Total .md files found**: 316 files (excluding node_modules)

**Key findings**:
- **37 root-level .md files** - Many are temporary status/completion reports
- **Duplicate content** - prompts/ duplicated in docs/prompts/ (with front matter)
- **Mixed concerns** - Some design docs in root should be in design_docs/
- **Stale summaries** - Multiple completion reports that are now historical

**Recommendation**: Archive 30+ temporary files, consolidate duplicates, maintain only living documentation at root level.

---

## Root-Level Files Analysis

### ✅ KEEP (Core Living Documentation)
These are actively maintained and belong at project root:

```
README.md              # Main project README (just cleaned up)
CHANGELOG.md           # Version history (actively maintained)
CLAUDE.md              # AI assistant instructions (maintained)
```

### 🗄️ ARCHIVE (Temporary Status/Completion Reports)
Move to `archive/completion_reports/YYYY-MM/`:

**M-DX1 Reports (October 2025)**:
- M-DX1-COMPLETION-ANNOUNCEMENT.md
- M-DX1-FINAL-SUMMARY.md
- M-DX1-PROGRESS-REPORT.md
- M-DX1-SESSION-SUMMARY.md

**M-DX4 Reports (October 2025)**:
- M-DX4-AUDIT-FINDINGS.md
- M-DX4-COMPLETION-SUMMARY.md
- M-DX4-IMPLEMENTATION-REPORT.md

**M-DX8 Reports (October 2025)**:
- M-DX8-PHASE1-QUICK-WINS.md

**M-POLY Reports (October 2025)**:
- M-POLY-A-DAY5-FINDINGS.md
- M-POLY-A-TEST-SUMMARY.md
- M-POLY-B-PHASE1-COMPLETE.md
- M-POLY-B-PHASE1-COMPLETION-REPORT.md
- M-POLY-B-PHASE1-STABILITY-REPORT.md

**Milestone Reports (October 2025)**:
- MILESTONE_1_COMPLETE.md
- MILESTONE_2_COMPLETE.md
- MILESTONE_3_COMPLETE.md

**Eval/Benchmark Analysis (October-November 2025)**:
- AGENT_BENCHMARK_SOLUTIONS.md
- AGENT_BENCHMARK_WORK_SUMMARY.md
- AGENT_EVAL_BENCHMARKS.md
- AGENT_EVAL_MODEL_FILTERING.md
- AGENT_EVAL_VERIFICATION.md
- API_CALL_JSON_FAILURE_ANALYSIS.md
- BENCHMARK_AUDIT_ANALYSIS.md
- M-EVAL-HTTP-FIX-FINDINGS.md
- NEW_BENCHMARK_TEST_RESULTS.md
- PER_BENCHMARK_TIMEOUT_RESULTS.md
- POST_RELEASE_WORKFLOW_ANALYSIS.md
- VISION_BENCHMARK_RESULTS.md

**Bug/Fix Reports (October-November 2025)**:
- LANG_PLACEHOLDER_FIX.md
- V0.4.0_AILANG_FAILURE_ANALYSIS.md
- V0.4.0_FAILURE_ROOT_CAUSE_ANALYSIS.md
- SELF_REPAIR_DISCOVERY.md

**Total to archive**: 30 files

### 🤔 EVALUATE

**gemini.md** (42KB, last updated Oct 25):
- Near-duplicate of CLAUDE.md but for Gemini model
- **Recommendation**: Move to `.claude/gemini.md` if still used, or delete if CLAUDE.md is sufficient
- Check git history to see if actively maintained

**WASM_DEPLOYMENT.md** (Oct 25):
- Deployment guide for WebAssembly
- **Recommendation**: Move to `docs/guides/wasm-deployment.md` (already exists as docs/docs/guides/wasm-integration.md)
- Check for duplication before moving

**examples_status.md** (Oct 26):
- Auto-generated from CI
- **Recommendation**: Keep if CI actively updates it, else delete (info is in examples/STATUS.md)

---

## docs/ Directory Analysis

### ✅ KEEP (Living Documentation)
```
docs/VISION.md                  # Project vision (referenced from README)
docs/LIMITATIONS.md             # Known limitations (actively maintained)
docs/TESTING.md                 # Testing guide (actively maintained)
docs/README.md                  # Docusaurus index
docs/architecture/              # Architecture docs (ANF.md, types.md)
docs/guides/                    # Development guides (maintained)
```

### 🗄️ ARCHIVE (Temporary Status)
Move to `archive/completion_reports/2025-10/`:

```
docs/FINAL_SUMMARY.md                    # Oct 10 - M-EVAL-LOOP v2.0 completion
docs/DOCUMENTATION_CLEANUP_SUMMARY.md    # Oct 25 - Documentation cleanup status
```

### 📚 KEEP BUT OUTDATED (Move to Archive)
```
docs/BENCHMARK_COMPARISON.md             # Oct 16 - Eval report for v0.3.9
                                         # NOTE: Auto-generated, should be in docs/static/benchmarks/
                                         # Check if still referenced
```

### 🤔 EVALUATE (Setup Guides)
```
docs/CLAUDE_CODE_SETUP.md                # Setup guide for Claude Code integration
docs/MULTI_PROVIDER_AGENT_SETUP.md       # Multi-provider agent eval setup
```
- **Recommendation**: Move to `docs/guides/` or `.claude/` if internal tooling docs
- These look like internal setup docs, not user-facing docs

---

## Duplicate Content Analysis

### prompts/ vs docs/prompts/

**Pattern**: All files in `prompts/` are duplicated in `docs/prompts/` with front matter added.

**Source**: `prompts/` (21 versioned prompt files)
**Docusaurus copy**: `docs/prompts/` (subset, with YAML front matter)

**Files in prompts/**:
```
CHANGELOG_v0.3.15.md
python.md
testing_guide_ai.md
v0.2.0.md through v0.4.1.md (21 version files)
```

**Files in docs/prompts/**:
```
Same files, subset, with front matter:
---
layout: page
title: AI Prompt - vX.X.X
parent: AI Prompts
nav_order: 1
---
```

**Recommendation**:
1. **Keep** `prompts/` as source of truth (actively updated)
2. **Keep** `docs/prompts/` for website (Docusaurus build)
3. **Add** build step to sync prompts/ → docs/prompts/ with front matter
4. **Document** in docs/README.md that prompts/ is source, docs/prompts/ is generated

---

## design_docs/ Analysis

### Current Structure (Well-Organized!)
```
design_docs/
├── archive/          (23 files - completed/obsolete design docs)
├── implemented/      (93 files - completed features by version)
├── planned/          (59 files - future work)
└── README.md         (index)
```

**Status**: ✅ Already well-organized!

**Recommendations**:
1. Check if any root-level analysis files should move to `design_docs/archive/`
2. Verify that `design_docs/PLANNED_DOCS_REVIEW.md` is still relevant (move to archive if complete)

---

## Recommended Archive Structure

Create this structure for archived temporary files:

```
archive/
├── 2025-10/
│   ├── completion_reports/
│   │   ├── M-DX1-COMPLETION-ANNOUNCEMENT.md
│   │   ├── M-DX1-FINAL-SUMMARY.md
│   │   ├── M-DX4-COMPLETION-SUMMARY.md
│   │   ├── MILESTONE_1_COMPLETE.md
│   │   └── ... (all completion reports)
│   ├── analysis/
│   │   ├── AGENT_BENCHMARK_SOLUTIONS.md
│   │   ├── API_CALL_JSON_FAILURE_ANALYSIS.md
│   │   ├── BENCHMARK_AUDIT_ANALYSIS.md
│   │   └── ... (all analysis docs)
│   └── bug_fixes/
│       ├── LANG_PLACEHOLDER_FIX.md
│       ├── SELF_REPAIR_DISCOVERY.md
│       └── V0.4.0_FAILURE_ROOT_CAUSE_ANALYSIS.md
└── 2025-11/
    └── ... (future archives)
```

---

## Action Plan

### Phase 1: Archive Root-Level Temporary Files
```bash
# Create archive structure
mkdir -p archive/2025-10/{completion_reports,analysis,bug_fixes}
mkdir -p archive/2025-11/analysis

# Move completion reports (October)
git mv M-DX*-*.md archive/2025-10/completion_reports/
git mv M-POLY-*.md archive/2025-10/completion_reports/
git mv MILESTONE_*.md archive/2025-10/completion_reports/

# Move analysis docs (October)
git mv AGENT_*.md archive/2025-10/analysis/
git mv BENCHMARK_*.md archive/2025-10/analysis/
git mv M-EVAL-HTTP-FIX-FINDINGS.md archive/2025-10/analysis/
git mv NEW_BENCHMARK_TEST_RESULTS.md archive/2025-10/analysis/
git mv PER_BENCHMARK_TIMEOUT_RESULTS.md archive/2025-10/analysis/
git mv POST_RELEASE_WORKFLOW_ANALYSIS.md archive/2025-10/analysis/
git mv VISION_BENCHMARK_RESULTS.md archive/2025-10/analysis/

# Move bug/fix reports (October-November)
git mv LANG_PLACEHOLDER_FIX.md archive/2025-10/bug_fixes/
git mv SELF_REPAIR_DISCOVERY.md archive/2025-10/bug_fixes/

# Move v0.4.0 analysis (November)
git mv V0.4.0_*.md archive/2025-11/analysis/
git mv API_CALL_JSON_FAILURE_ANALYSIS.md archive/2025-11/analysis/
```

### Phase 2: Clean Up docs/ Directory
```bash
# Move temporary summaries
git mv docs/FINAL_SUMMARY.md archive/2025-10/completion_reports/
git mv docs/DOCUMENTATION_CLEANUP_SUMMARY.md archive/2025-10/completion_reports/

# Evaluate BENCHMARK_COMPARISON.md
# (Check if still referenced, might be auto-generated)

# Move setup guides to .claude/ or docs/guides/
git mv docs/CLAUDE_CODE_SETUP.md .claude/SETUP_GUIDE.md
git mv docs/MULTI_PROVIDER_AGENT_SETUP.md .claude/MULTI_PROVIDER_SETUP.md
```

### Phase 3: Handle Edge Cases
```bash
# Evaluate gemini.md
# Option 1: Move to .claude/ if still used
git mv gemini.md .claude/gemini.md
# Option 2: Delete if CLAUDE.md is sufficient
git rm gemini.md

# Evaluate WASM_DEPLOYMENT.md
# Check for duplication with docs/docs/guides/wasm-integration.md
# If duplicate, delete root version
git rm WASM_DEPLOYMENT.md

# Evaluate examples_status.md
# Check if CI still updates it
# If not, delete (info in examples/STATUS.md)
git rm examples_status.md
```

### Phase 4: Update References
```bash
# Update any references to archived files
grep -r "M-DX1-COMPLETION" --include="*.md" .
grep -r "MILESTONE_1_COMPLETE" --include="*.md" .
# Update links to point to archive/

# Update README.md if it references any archived files
# (Already done in previous cleanup)
```

### Phase 5: Add Archive README
Create `archive/README.md` with explanation of structure.

---

## Summary Statistics

### Before Cleanup
```
Root-level .md files:     37
docs/ root .md files:      9
Total active files:       46
```

### After Cleanup
```
Root-level .md files:      3 (README, CHANGELOG, CLAUDE)
docs/ root .md files:      6 (VISION, LIMITATIONS, TESTING, README, architecture/, guides/)
Archived files:           32
Total active files:        9 (80% reduction!)
```

### Benefits
- ✅ **Cleaner root directory** - Only living documentation
- ✅ **Historical preservation** - All work preserved in archive/
- ✅ **Better discoverability** - Clear separation of active vs historical
- ✅ **Easier maintenance** - Fewer files to keep updated
- ✅ **Git history preserved** - All moves tracked via git mv

---

## Next Steps

1. **Review this audit** with team/maintainers
2. **Execute Phase 1** - Archive root-level files
3. **Execute Phase 2** - Clean up docs/
4. **Execute Phase 3** - Handle edge cases
5. **Execute Phase 4** - Update references
6. **Execute Phase 5** - Add archive README
7. **Update CLAUDE.md** - Document new archive/ structure
8. **Commit changes** - Single commit with clear message

**Estimated time**: 30-45 minutes

**Risk**: Low (all git mv operations preserve history)

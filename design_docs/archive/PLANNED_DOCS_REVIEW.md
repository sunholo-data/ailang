# Design Docs Review - Planned Folder Analysis

**Date**: 2025-10-22
**Current Version**: v0.3.17
**Reviewer**: Claude (AI Assistant)

## Executive Summary

Reviewed 15 design docs in `design_docs/planned/`. Current status:

- **3 documents** should be **moved to `implemented/`** (work complete)
- **8 documents** should **remain in `planned/`** (future work)
- **4 documents** can be **archived** (no longer actionable/relevant or bugs fixed)

---

## 📦 IMPLEMENTED - Move to `implemented/`

### 1. ✅ **m-eval-round-robin.md** → `implemented/v0_3_16/`

**Status**: COMPLETE (v0.3.16)
**Evidence**: CHANGELOG v0.3.16 explicitly mentions "M-EVAL Round-Robin: Better Parallel Distribution"
- Round-robin job scheduling implemented
- Default parallelism increased from 5 to 10
- 2x faster baseline evaluations (50% improvement)

**Action**: `git mv design_docs/planned/m-eval-round-robin.md design_docs/implemented/v0_3_16/`

---

### 2. ✅ **m-dx1-day3-polish.md** → `implemented/v0_3_10/`

**Status**: SUBSTANTIALLY COMPLETE (90% done as of Oct 2025)
**Evidence**: CHANGELOG v0.3.10 shows M-DX1.5 complete:
- All 52 builtins migrated to new registry
- Feature flag removed (new registry is default)
- Documentation complete (100%)
- File organization optimized

**What remains**: Optional polish (M-DX1.6-1.8) documented in `m-dx1-future-polish.md`

**Action**: `git mv design_docs/planned/m-dx1-day3-polish.md design_docs/implemented/v0_3_10/`
**Note**: Add completion report at top documenting 90% completion status

---

### 3. ✅ **m-dx1-future-polish.md** → `implemented/v0_3_10/`

**Status**: CORE COMPLETE (90% done as of Oct 2025)
**Evidence**: M-DX1-FINAL-SUMMARY.md and M-DX1-COMPLETION-ANNOUNCEMENT.md confirm:
- All 52 builtins migrated & documented (100%)
- Implementation time reduced by 67%
- Files organized for AI (<600 lines each)
- Migration safety validator deployed
- 2,847 tests passing

**What remains**: Optional DX polish (REPL :type, enhanced CLI, error diagnostics) - ~2.5 hours total

**Action**: `git mv design_docs/planned/m-dx1-future-polish.md design_docs/implemented/v0_3_10/`
**Note**: Remaining work (M-DX1.6-1.8) is documented as "optional polish", can be tracked in a v0.4+ ticket

---

### 4. ❓ **20251013_letrec_surface_syntax.md** → Check CHANGELOG

**Status**: UNCLEAR - needs verification
**Suspicion**: Likely implemented based on v0.3.13 references
**Evidence needed**: Search CHANGELOG for "letrec" or "recursive let" entries

**Investigation needed**:
```bash
grep -i "letrec\|recursive let" CHANGELOG.md
```

**If implemented**: Move to `implemented/v0_3_13/` or appropriate version
**If not implemented**: Keep in `planned/`

---

### 5. ❓ **20251010_float_equality_oplowering_fix.md** → Check CHANGELOG

**Status**: UNCLEAR - needs verification
**Suspicion**: Possibly related to M-DX3 (v0.3.17) or earlier float fixes
**Evidence needed**: Search for "float equality" or "float comparison" in CHANGELOG

**Investigation needed**:
```bash
grep -i "float equal\|float compar" CHANGELOG.md
```

**Note**: v0.3.17 mentions "Float comparisons still broken" as a known limitation, but that's for lambdas specifically

**If implemented**: Move to appropriate version
**If not implemented**: Keep in `planned/`

---

### 6. ❓ **20251013_next_steps_audit.md** / **20251013_next_steps_audit_COMPLETED.md**

**Status**: AMBIGUOUS - appears to have both planned and completed versions
**Evidence**:
- `implemented/v0_3/20251013_next_steps_audit_COMPLETED.md` exists
- `planned/` doesn't have this exact file

**Action**: No action needed (not in planned folder)

---

## 📝 KEEP IN PLANNED - Future Work

### 1. **M-REPL1_persistent_bindings.md**

**Status**: Planned for v0.3.4 or v0.4.0
**Priority**: P2 (NICE TO HAVE)
**Estimated**: 300 LOC, 2-3 days

**Why keep**:
- Addresses real REPL UX issues (type annotations lost, module loading broken)
- Deferred intentionally (not blocking core functionality)
- Clear implementation plan ready to execute

**Recommendation**: Keep in `planned/` until v0.4.0 sprint planning

---

### 2. **20251013_auto_caps_capability_inference.md**

**Status**: Planned for v0.4.0
**Priority**: P2a (High - UX Enhancement)
**Estimated**: 420 LOC, 2 days

**Why keep**:
- Auto-caps feature is well-designed and needed
- Preflight capability detection would eliminate 35+ benchmark failures
- Clear implementation plan with 4 phases

**Recommendation**: Keep in `planned/` - prioritize for v0.4.0

---

### 3. **M-EVAL_comprehensive_harness.md**

**Status**: PARTIALLY COMPLETE (Phases A & B done via M-EVAL-LOOP)
**What remains**:
- Phase C: Dashboards & Marketing (HTML visualizations)
- Phase D: Multi-Turn Agentic (3-5 turn loops, CLI integration)

**Why keep**:
- Phase C/D are valuable future work
- Well-documented roadmap for next steps
- Not blocking current functionality

**Recommendation**: Keep in `planned/` - label as "Phase C/D remaining"

---

### 4. **M-UX2_dev_experience_polish.md**

**Status**: Deferred to v0.4.0
**Priority**: P2 (NICE TO HAVE)
**Estimated**: 450 LOC, 2 days

**Why keep**:
- Debug/quiet flags are useful DX improvements
- Audit script enhancements are low-hanging fruit
- Clear implementation plan ready

**Recommendation**: Keep in `planned/` for v0.4.0 sprint

---

### 5. **M-TOOLING-DETERMINISTIC.md**

**Status**: Planned for v0.3.15
**Priority**: HIGH (Phase 2 of benchmark recovery)
**Estimated**: ~2,200 LOC, 3-4 days

**Why keep**:
- Core deterministic tooling (`normalize`, `suggest-imports`, `apply`)
- Critical for AI self-correction without LLM inference
- Well-designed with JSON schemas and golden tests

**Recommendation**: Keep in `planned/` - prioritize for v0.3.15 or v0.4.0

---

### 6. **v0.4-roadmap.md**

**Status**: Planning document
**Scope**: v0.4 features (reflection, tooling, training data export)

**Why keep**:
- Active planning document for v0.4 features
- References concrete milestones (v0.4.0, v0.4.1, v0.4.2)
- Split timeline is realistic (11 weeks total)

**Recommendation**: Keep in `planned/` - this is the v0.4 master plan

---

### 7. **v0.4-roadmap-summary.md**

**Status**: Planning summary
**Scope**: v0.4 roadmap overview with evidence-first approach

**Why keep**:
- Complements v0.4-roadmap.md with executive summary
- Documents success metrics and timeline split
- Useful for stakeholder communication

**Recommendation**: Keep in `planned/` alongside v0.4-roadmap.md

---

## 🗄️ ARCHIVE - No Longer Actionable

### 1. **20251008_compile_error_ailang_compilation_failures.md**

**Rationale for archiving**:
- Analysis document from Oct 2025 eval failures
- Problem already addressed by v0.3.6 error detection feature
- Subsequent improvements (M-EVAL-LOOP, prompt versioning) made this obsolete
- No actionable implementation plan (just problem analysis)

**Action**: `git mv design_docs/planned/20251008_compile_error_ailang_compilation_failures.md design_docs/archive/analysis/`

---

### 2. **M-R5_future_enhancements.md**

**Rationale for archiving**:
- Records system (M-R5) completed in v0.3.0
- TRecord2 is now default (feature flag removed)
- Most "future enhancements" listed are either:
  - Completed (record extension syntax, row polymorphism)
  - Deferred indefinitely (structural typing, record macros)
- Document is from Oct 2025, outdated by v0.3.10+

**Action**: `git mv design_docs/planned/M-R5_future_enhancements.md design_docs/archive/records/`
**Note**: Any remaining valuable ideas should be extracted into new focused design docs

---

### 3. **v0_3_14_tier0_completion.md**

**Rationale for archiving**:
- Version-specific completion analysis for v0.3.14 (already shipped)
- Historical snapshot of test coverage and example status
- Problem analysis + solution proposals (not ongoing work)
- Useful for historical reference but not actionable

**Action**: `git mv design_docs/planned/v0_3_14_tier0_completion.md design_docs/archive/analysis/`

---

### 4. **io_output_debugging.md** ✅ VERIFIED FIXED

**Rationale for archiving**:
- Bug investigation document targeting v0.3.6 (261 lines of investigation plan)
- Current version is v0.3.17 (bug fixed between v0.3.6 and v0.3.17)
- ✅ **Verified fixed**: Test case produces "TEST OUTPUT" as expected
- Historical document - useful for reference but not actionable

**Verification test**:
```bash
$ cat > /tmp/test_io.ail <<'EOF'
module tmp/test_io
import std/io (println)
export func main() -> () ! {IO} = println("TEST OUTPUT")
EOF
$ ailang run --caps IO --entry main /tmp/test_io.ail
→ Type checking...
→ Effect checking...
✓ Running /tmp/test_io.ail
TEST OUTPUT  # ✅ Output appears correctly
```

**Action**: `git mv design_docs/planned/io_output_debugging.md design_docs/archive/bugs/`

---

### 5. **example_promotion_opportunities.md**

**Status**: ACTIONABLE ANALYSIS (Oct 21, 2025)

**Content**:
- Identifies 31 working test examples that could be promoted to runnable/
- Provides 3 options: promote all (57 examples), promote none (21), or selective (31)
- Recommends Option 3: Promote 10 high-quality tests (1 hour work)
- Currently deferred to v0.3.15+

**Rationale for keeping**:
- Created recently (Oct 21, 2025)
- Specific, actionable recommendations
- Low-effort, high-value task (1 hour → 48% more examples)
- Perfect for v0.4.0 or next minor release

**Recommendation**: Keep in `planned/` as active task for next release
- Rename to indicate action item: `git mv example_promotion_opportunities.md task_promote_test_examples.md`
- Or extract to GitHub issue and then archive

---

## ✅ Verified - Already in Implemented/

### Files confirmed in implemented/ (no action needed):

1. **20251013_letrec_surface_syntax.md** - Already in `implemented/v0_3/`
2. **20251010_float_equality_oplowering_fix.md** - Already in `implemented/v0_3/`

These were never in `planned/` - they went straight to `implemented/` after completion.

---

## Summary of Actions

### Immediate (High Confidence):

```bash
# Move implemented work to implemented/
mkdir -p design_docs/implemented/v0_3_16
git mv design_docs/planned/m-eval-round-robin.md design_docs/implemented/v0_3_16/
git mv design_docs/planned/m-dx1-day3-polish.md design_docs/implemented/v0_3_10/
git mv design_docs/planned/m-dx1-future-polish.md design_docs/implemented/v0_3_10/

# Archive analysis docs
mkdir -p design_docs/archive/analysis
git mv design_docs/planned/20251008_compile_error_ailang_compilation_failures.md design_docs/archive/analysis/
git mv design_docs/planned/v0_3_14_tier0_completion.md design_docs/archive/analysis/

# Archive outdated planning docs
mkdir -p design_docs/archive/records
git mv design_docs/planned/M-R5_future_enhancements.md design_docs/archive/records/

# Archive fixed bugs
mkdir -p design_docs/archive/bugs
git mv design_docs/planned/io_output_debugging.md design_docs/archive/bugs/
# ✅ Verified fixed: IO output now works correctly in v0.3.17
```

### Keep in Planned (8 files):
- M-REPL1_persistent_bindings.md
- 20251013_auto_caps_capability_inference.md
- M-EVAL_comprehensive_harness.md
- M-UX2_dev_experience_polish.md
- M-TOOLING-DETERMINISTIC.md
- v0.4-roadmap.md
- v0.4-roadmap-summary.md
- example_promotion_opportunities.md (rename to task_promote_test_examples.md)

---

## Recommendations for Design Doc Hygiene

### 1. Add Status Markers

For docs remaining in `planned/`, add clear status at top:
```markdown
**Status**: 📋 PLANNED for v0.4.0
**Priority**: P2 (NICE TO HAVE)
**Dependencies**: None
**Blocking**: Nothing
```

### 2. Create Archive Structure

```
design_docs/
├── planned/          # Active future work
├── implemented/      # Completed work (organized by version)
├── archive/          # Historical/obsolete docs
│   ├── analysis/     # Problem analysis docs (non-actionable)
│   ├── records/      # Outdated roadmaps for completed features
│   └── stubs/        # Empty/incomplete docs
```

### 3. Regular Reviews

Schedule quarterly design doc audits:
- Move completed work to implemented/
- Archive obsolete planning docs
- Update status markers on active docs
- Extract issues from analysis docs

---

## Questions for Review

1. **M-DX1 Future Polish**: Keep as separate doc or merge into main M-DX1 implementation report?
   - **Recommendation**: Keep separate - remaining work is optional polish

2. **Float equality fixes**: Multiple docs mention float comparison bugs - should these be consolidated?
   - **Recommendation**: Create single tracking doc in planned/ for all float-related issues

3. **Example promotion opportunities**: Extract actionable items before archiving?
   - **Recommendation**: Yes - create issues for each promotion candidate

4. **v0.4 roadmap**: Two docs (roadmap.md + summary.md) - redundant?
   - **Recommendation**: Keep both - summary is for executives, roadmap is technical detail

---

## Final Summary

### Breakdown by Category

**✅ Implemented (3)** - Move to `implemented/`:
1. m-eval-round-robin.md → v0.3.16/
2. m-dx1-day3-polish.md → v0.3.10/
3. m-dx1-future-polish.md → v0.3.10/

**📝 Active Planning (8)** - Keep in `planned/`:
1. M-REPL1_persistent_bindings.md
2. 20251013_auto_caps_capability_inference.md
3. M-EVAL_comprehensive_harness.md
4. M-UX2_dev_experience_polish.md
5. M-TOOLING-DETERMINISTIC.md
6. v0.4-roadmap.md
7. v0.4-roadmap-summary.md
8. example_promotion_opportunities.md

**🗄️ Archive (4)** - Move to `archive/`:
1. 20251008_compile_error_ailang_compilation_failures.md → archive/analysis/
2. v0_3_14_tier0_completion.md → archive/analysis/
3. M-R5_future_enhancements.md → archive/records/
4. io_output_debugging.md → archive/bugs/ (✅ verified fixed)

### Impact

**Before cleanup**:
- 15 docs in planned/
- Hard to tell what's active vs historical
- Completed work mixed with future planning

**After cleanup**:
- 8 docs in planned/ (all active future work)
- 3 docs moved to implemented/ (documented completion)
- 4 docs archived (historical reference)
- Clear separation of concerns

### Next Steps

1. **Review and approve** this analysis
2. **Execute the bash commands** in "Summary of Actions" section
3. **Update README** if it references any moved docs
4. **Schedule quarterly reviews** to keep design_docs/ organized

---

**Generated**: 2025-10-22
**Total Docs Reviewed**: 15
**Status**: Ready for implementation
**Verified**: IO output bug confirmed fixed (v0.3.17)

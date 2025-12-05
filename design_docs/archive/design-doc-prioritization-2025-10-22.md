# Design Doc Audit & Prioritization (October 2025)

**Status**: Complete
**Created**: 2025-10-22
**Auditor**: Claude (design-doc-creator skill)
**Purpose**: Comprehensive audit of all planned design docs with AI-first alignment scoring and prioritization

---

## Executive Summary

**Total Docs Audited**: 32 design documents
**Complete & Scored**: 11 major initiatives
**Ready to Implement**: 5 P0-P1 docs with >+2 score
**Needs Work**: 3 docs with incomplete sections
**Deferred/Archive Candidates**: 14 docs (obsolete, completed, or superseded)

### Top 2 Priorities (Immediate Action)

**⚠️ CRITICAL UPDATE (2025-10-22 14:38)**: Just tested current state. Results below.

1. **20251022_compile_error_ailang_compilation_failures.md** [P0, URGENT, Score: +3]
   - **Status**: Teaching prompt v0.3.19 has HTTP examples BUT STILL FAILING
   - **Just tested**: api_call_json success rate 16.7% (1/6), all 3 AILANG runs failed
   - **Why P0**: Currently breaking benchmarks despite prompt updates
   - **Impact**: +60pp improvement needed (16.7% → 75%+)
   - **Effort**: 4-6 hours (implement Option A: enhanced error messages)
   - **Evidence**: gpt5-mini still generates `import http` / `url = ...` / `http.post()`
   - **Next**: Error messages must guide models to correct syntax

2. **M-DX4: CoreTypeInfo Completeness** [P0, Score: +3]
   - **Status**: Infrastructure exists (CoreTypeInfo tracking), validation pass missing
   - **Git history**: Commits bcc973c, e60c60a added CoreTypeInfo tracking
   - **What's missing**: `validateCoreTypeInfo()` walker pass (design doc's goal)
   - **Why P0**: Prevents compiler panics and wrong code generation
   - **Impact**: Eliminates "unknown variant" runtime errors
   - **Effort**: 6 hours (implement validation pass, comprehensive tests)

### ✅ Already Implemented (Move to Archive)

3. **M-DX3: Lambda DX Fixes** - **COMPLETE** (commit 9152119, 2025-10-22)
   - ✅ Comparison operators in lambda bodies FIXED
   - ✅ show(Bool) already works (no changes needed)
   - ✅ All tests pass, lint clean
   - **Action**: Move `v0_3_17/m-dx3-lambda-dx-fixes.md` to `implemented/v0_3_17/`

---

## AI-First Scoring Rubric

All docs scored against AILANG's core principles:

| Principle | Question | Impact |
|-----------|----------|--------|
| **Reduce Syntactic Noise** | Removes boilerplate/scaffolding? | −1/0/+1 |
| **Preserve Semantic Clarity** | Keeps meaning explicit? | −1/0/+1 |
| **Increase Determinism** | Identical inputs → identical outputs? | −1/0/+1 |
| **Lower Token Cost** | Reduces AI conversation rounds? | −1/0/+1 |

**Decision Rule**: Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference**: [VISION_BENCHMARKS.md](../../benchmarks/VISION_BENCHMARKS.md), [example-parity-vision-alignment.md](v0_3_16/example-parity-vision-alignment.md)

---

## Priority Tier 1: P0 Critical (Ship First)

### 1. 20251022_compile_error_ailang_compilation_failures.md

**Status**: ✅ **Now Complete** - Analysis done, solution specified
**Target**: v0.3.20 (next version)
**Score**: +3 (Semantic Clarity +1, Determinism +1, Token Cost +1)
**Estimated**: 4-6 hours (enhanced error messages implementation)

**⚠️ CRITICAL UPDATE - Just Tested (2025-10-22 14:38)**:
- Teaching prompt v0.3.19 has HTTP+JSON example at line 166
- **BUT**: api_call_json still fails at 16.7% (1/6 runs)
- **All 3 AILANG runs failed** with same parse errors
- **Conclusion**: Examples alone don't prevent failures

**What It Is:**
- Fix ai_call_json benchmark failures (currently 16.7% success)
- Models affected: claude-haiku-4-5, gemini-2-5-flash, gpt5-mini
- Root cause: AIs generate Python-style syntax despite prompt examples

**Latest Failure (gpt5-mini, just now):**
```python
import http              # ❌ Namespace import
import json              # ❌ Namespace import
url = "..."              # ❌ Bare assignment
headers = {...}          # ❌ Bare assignment
resp = http.post(...)    # ❌ Method call syntax
print(resp.status)       # ❌ Should be println(show(...))
```

**Why P0 URGENT:**
- Currently failing RIGHT NOW (16.7% success vs 25% in original doc)
- Prompt update (v0.3.17, v0.3.19) didn't fix it
- Models default to familiar patterns despite examples
- Only solution: Enhanced error messages (Option A)

**Next Steps:**
1. ✅ Root cause analysis - DONE (see design doc)
2. ✅ Solution specified - Option A + Option B (design doc complete)
3. **TODO**: Implement enhanced parser error messages (~2h)
4. **TODO**: Test with eval-suite (~2h)
5. **Expected**: 16.7% → 75%+ success rate

---

### 2. M-DX4: CoreTypeInfo Completeness (v0_3_15/m-dx4-coretypeinfo-completeness.md)

**Status**: ⚠️ **Partially Implemented** - Infrastructure exists, validation missing
**Target**: v0.3.20 (next version)
**Score**: +3 (Semantic Clarity +1, Determinism +1, Token Cost +1)
**Estimated**: 6 hours (validation pass + tests)

**Implementation Status (checked git history):**
- ✅ CoreTypeInfo tracking exists (commits bcc973c, e60c60a)
- ✅ Infrastructure in place (`internal/elaborate/`, `internal/types/`)
- ❌ Validation pass NOT implemented (main goal of design doc)
- ❌ CI enforcement NOT added

**What's Missing:**
- `Pipeline.validateCoreTypeInfo()` walker function
- Validation call in `Pipeline.Run()` before lowering
- Tests for Float/Bool/comparison operator CoreTypeInfo
- CI gate: fail if coverage < 100%

**Why P0:**
- Compiler panics are critical bugs
- Silent fallbacks hide bugs (violates "fail loudly" principle)
- Blocks development of type-guided features (M-DX2 depends on this)

**Next Steps:**
1. ✅ Design complete (4 phases specified)
2. ✅ Infrastructure exists (CoreTypeInfo tracking)
3. **TODO**: Implement validation walker (~2h)
4. **TODO**: Fix missing CoreTypeInfo entries (~2h)
5. **TODO**: Add comprehensive tests + CI gate (~2h)

---

### 3. M-DX3: Lambda DX Fixes (v0_3_17/m-dx3-lambda-dx-fixes.md)

**Status**: ✅ **IMPLEMENTED** - Shipped in commit 9152119 (2025-10-22)
**Target**: v0.3.17 ✅ SHIPPED
**Score**: +3 (Syntactic Noise +1, Semantic Clarity +1, Token Cost +1)

**Implementation Complete:**
- ✅ Comparison operators in lambda bodies FIXED
  - Root cause: operator lowering used result type (Bool) instead of operand type
  - Solution: Use operand type for comparison/equality operators
- ✅ show(Bool) already works - no changes needed
- ✅ Tests added: `internal/pipeline/op_lowering_comparison_test.go` (237 LOC)
- ✅ All tests pass, lint clean
- ✅ Lambda examples enhanced with max/min/abs

**Commit Details:**
```
commit 91521195720f58e13e7148b607e2aebb392d3b40
Author: Mark
Date:   Wed Oct 22 00:13:56 2025 +0200

Complete M-DX3: Fix comparison operators in lambda bodies
```

**Action Required:**
- **Move design doc**: `v0_3_17/m-dx3-lambda-dx-fixes.md` → `implemented/v0_3_17/`
- **Update CHANGELOG**: Ensure v0.3.17 entry documents this fix
- **No further work needed**

---

## Priority Tier 2: P1 High Priority (Next Sprint)

### 4. M-DX5: Inference Visibility (v0_3_15/m-dx5-inference-visibility.md)

**Status**: ✅ **Complete** - Ready to implement
**Target**: v0.3.15
**Score**: +3 (Semantic Clarity +1, Determinism +1, Token Cost +1)
**Estimated**: 6 hours
**Depends On**: M-DX4 (CoreTypeInfo validation)

**What It Is:**
- `ailang debug types --trace-inference` - Show type inference steps
- `ailang debug types --show-gaps` - List missing CoreTypeInfo
- `ailang debug types --query <expr>` - Inspect inferred types

**Why P1:**
- Improves DX for debugging type errors
- Enables faster development (fewer conversation rounds)
- Builds on M-DX4 infrastructure

---

### 5. 20251013_auto_caps_capability_inference.md

**Status**: ✅ **Complete** - High quality
**Target**: v0.4.0 (deferred from v0.3.x)
**Score**: +2 (Syntactic Noise +1, Token Cost +1)
**Estimated**: 2 days (~420 LOC)

**What It Is:**
- Preflight capability detection (show missing caps before run)
- `--auto-caps` flag (automatically grant required caps)
- Environment variable `AILANG_AUTO_CAPS=1` for CI

**Why P1 (not P0):**
- UX enhancement, not blocking bug
- Workaround exists (manually specify --caps)
- Improves dev experience but doesn't block features

---

### 6. M-TOOLING-DETERMINISTIC.md

**Status**: ✅ **Complete** - Extensive spec
**Target**: v0.3.15
**Score**: +3 (Syntactic Noise +1, Determinism +1, Token Cost +1)
**Estimated**: 3-4 days (~2,355 LOC)

**What It Is:**
- `ailang normalize` - Wrap fragments into modules
- `ailang suggest-imports` - Resolve missing symbols
- `ailang apply` - Apply JSON edits deterministically

**Why P1:**
- Enables deterministic code repair (no LLM needed)
- Reduces eval costs ($0.002-0.005 per repair → free)
- Faster (<100ms vs 500ms-2s per repair)

---

## Recommended Implementation Order

### Sprint 1: Fix Critical Bugs (2-3 days)

**Week Goal**: Ship v0.3.20 with API/JSON benchmark fixes

**Updated Timeline (post-verification):**

1. **Day 1**: 20251022_compile_error (4-6h) - **URGENT**
   - ✅ Analysis complete (see design doc)
   - ✅ Solution specified (Option A: enhanced error messages)
   - **TODO**: Implement parser error suggestions
   - **TODO**: Test with eval-suite
   - **Success Metric**: 16.7% → 75%+ success rate on api_call_json

2. **Day 2-3**: M-DX4 CoreTypeInfo (6h)
   - ✅ Infrastructure exists
   - **TODO**: Implement validation walker
   - **TODO**: Fix missing CoreTypeInfo entries
   - **TODO**: Add tests + CI gate
   - **Success Metric**: 100% CoreTypeInfo coverage, no panics

3. ~~**Day 4-5**: M-DX3 Lambda DX Fixes~~ - **ALREADY DONE** ✅
   - ✅ Shipped in commit 9152119 (2025-10-22)
   - **Action**: Move design doc to implemented/

**Sprint 1 Deliverables**:
- ✅ v0.3.20 tag
- ✅ 2 P0 issues fixed (compile_error, M-DX4)
- ✅ 1 doc archived (M-DX3)
- ✅ api_call_json: 16.7% → 75%+ success
- ✅ Eval baseline improvement: current → +10-15pp compile success

---

## Summary Statistics

### By Status
- ✅ **Complete & Ready**: 9 docs
- ✅ **Implemented & Shipped**: 1 doc (M-DX3)
- ⚠️ **Partially Implemented**: 1 doc (M-DX4 - infrastructure only)
- ⚠️ **Complete but Failing**: 1 doc (compile_error - prompt updated but still fails)
- 🔍 **Needs Review**: 7 docs
- 📦 **Archive Candidates**: 14 docs

### By Priority
- **P0 Critical (Active)**: 2 docs (compile_error, M-DX4)
- **P0 Complete**: 1 doc (M-DX3 - move to archive)
- **P1 High**: 4 docs
- **P2 Medium**: 2 docs
- **v0.4.0 Roadmap**: 3 docs
- **Unscored**: 7 docs
- **Archive**: 14 docs

### By AI-First Score
- **+3 (Excellent)**: 7 docs
- **+2 (Good)**: 2 docs
- **+1 (Neutral)**: 0 docs
- **0 (Minimal Impact)**: 1 doc
- **Unscored**: 7 docs

---

## Appendix: Full Doc List with Scores

| Doc | Priority | Score | Effort | Status | Notes |
|-----|----------|-------|--------|--------|-------|
| compile_error | P0 URGENT | +3 | 4-6h | ⚠️ Active | Prompt updated but STILL failing (16.7%) |
| M-DX4-coretypeinfo | P0 | +3 | 6h | ⚠️ Partial | Infrastructure exists, validation missing |
| M-DX3-lambda-dx | ✅ DONE | +3 | - | ✅ Shipped | Move to implemented/v0_3_17/ |
| M-DX5-inference | P1 | +3 | 6h | ✅ Ready | Depends on M-DX4 |
| auto-caps | P1 | +2 | 2d | ✅ Ready | Deferred to v0.4.0 |
| M-TOOLING | P1 | +3 | 3-4d | ✅ Ready | Consider phasing |
| M-DX2-operator | P1-P2 | +3 | 8h | ✅ Ready | Good DX improvement |
| v0.4-roadmap | Meta | N/A | 7w | ✅ Complete | Use for planning |
| M-DX2-M1-M4-COMPLETE | - | - | - | 📦 Archive | Move to implemented/ |
| DEFERRED_FEATURES | Meta | N/A | - | 🔍 Review | Check if still relevant |

---

## Next Steps

### Immediate Actions (This Week)

**⚠️ UPDATED PRIORITIES (post-verification, 2025-10-22 14:38)**

1. **Implement compile_error fix** (4-6 hours) - **START IMMEDIATELY**
   - ✅ Doc complete (analysis done, solution specified)
   - ✅ Tested: Currently failing at 16.7% (all AILANG runs fail)
   - **TODO**: Implement enhanced parser error messages
   - **TODO**: Re-run eval-suite to measure improvement
   - **Expected**: 16.7% → 75%+ success rate

2. **Implement M-DX4 validation pass** (6 hours) - **Day 2-3**
   - ✅ Infrastructure exists
   - **TODO**: Implement `validateCoreTypeInfo()` walker
   - **TODO**: Fix missing entries, add tests
   - **Impact**: Eliminate compiler panics

3. **Archive M-DX3** (5 minutes)
   - ✅ Already implemented (commit 9152119)
   - **TODO**: `git mv design_docs/planned/v0_3_17/m-dx3-lambda-dx-fixes.md design_docs/implemented/v0_3_17/`

4. **Review uncertain docs** (2 hours) - **After P0s done**
   - Assess completeness and relevance
   - Archive/merge/prioritize each

5. **Archive obsolete docs** (30 minutes) - **After P0s done**
   - Move completed milestones to implemented/
   - Clean up planned/ directory

**Sprint Goal**: Ship v0.3.20 with api_call_json fix (16.7% → 75%+) + CoreTypeInfo validation

---

**Document created**: 2025-10-22
**Last updated**: 2025-10-22 14:45 (post-verification)

---

## Verification Log

**2025-10-22 14:38** - Tested current implementation status:
- ✅ M-DX3 confirmed shipped (commit 9152119)
- ⚠️ compile_error: Prompt v0.3.19 has HTTP examples BUT still failing
  - Tested: `ailang eval-suite --benchmarks api_call_json`
  - Result: 16.7% success (1/6), all 3 AILANG runs failed
  - Evidence: gpt5-mini generated `import http`, `url = ...`, `http.post(...)`
  - Conclusion: Examples alone insufficient, need error message suggestions
- ⚠️ M-DX4: Infrastructure exists (CoreTypeInfo tracking), validation pass missing
  - Git history: commits bcc973c, e60c60a added tracking
  - Missing: `validateCoreTypeInfo()` walker (design doc's main goal)

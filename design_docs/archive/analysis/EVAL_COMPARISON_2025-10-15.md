# M-EVAL Comparison: v0.3.6 vs v0.3.7

**Analysis Date**: 2025-10-15
**Tool**: `ailang eval-compare`
**Baselines**: v0.3.6-24 → v0.3.7-1-gd24a7dc

---

## Executive Summary

**Overall Success Rate**: 61.8% → 58.8% (**-3.0% regression**)
- v0.3.6: 42/68 runs passing (61.8%)
- v0.3.7: 67/114 runs passing (58.8%)

**By Language**:
- **AILANG**: 22/57 passing (38.6%) ← **This is the real AILANG success rate**
- **Python**: 45/57 passing (78.9%) ← Baseline comparison

**Key Finding**: CHANGELOG's "58.8%" is **combined Python+AILANG**, not AILANG-only!

---

## Regression Analysis

### ✅ Fixed Benchmarks (1)
- `record_update` (python, gemini-2-5-pro) ← Python improvement, not AILANG

### ❌ Broken Benchmarks (5 regressions)

#### 1. pattern_matching_complex (ailang, gemini-2-5-pro) - compile_error
**Root Cause**: **Multi-line ADT syntax not supported**

**v0.3.6 Code** (✅ Works):
```ailang
type Tree = Leaf(int) | Node(Tree, int, Tree)
```

**v0.3.7 Code** (❌ Fails):
```ailang
type Tree =
  | Leaf(int)
  | Node(Tree, int, Tree)
```

**Error**:
```
parse errors: [PAR_TYPE_BODY_EXPECTED at line 6:3: expected type definition
PAR_NO_PREFIX_PARSE at line 7:3: unexpected token in expression: |]
```

**Impact**: AI models prefer multi-line syntax for readability
**Solution**: Add multi-line ADT parsing support (v0.4.0 parser enhancement)

---

#### 2. pattern_matching_complex (ailang, gpt5) - compile_error
**Same root cause** as #1 (multi-line ADT syntax)

---

#### 3. pattern_matching_complex (python, claude-sonnet-4-5) - runtime_error
**Not relevant** to AILANG (Python benchmark regression)

---

#### 4. adt_option (ailang, gpt5) - runtime_error
**v0.3.6**: ✅ passing
**v0.3.7**: ❌ runtime_error (compiles OK, runtime fails)

**Need to investigate**: What changed between v0.3.6 and v0.3.7 that breaks ADT runtime?

**Candidates**:
- Type system changes (unlikely - v0.3.7 was just cleanup)
- Evaluator changes (check CHANGELOG v0.3.7)
- Pattern matching changes

**Action**: Compare generated code between v0.3.6 and v0.3.7 for same seed

---

#### 5. nested_records (python, claude-sonnet-4-5) - runtime_error
**Not relevant** to AILANG (Python benchmark regression)

---

## Success Rate Breakdown (Corrected)

### v0.3.7 Success Rates

| Metric | Count | Percentage |
|--------|-------|------------|
| **Total runs** | 114 | 100% |
| **AILANG runs** | 57 | 50% |
| **Python runs** | 57 | 50% |
| **AILANG passing** | 22 | **38.6%** ← Real AILANG rate |
| **Python passing** | 45 | **78.9%** ← Baseline |
| **Combined passing** | 67 | 58.8% ← CHANGELOG number |

### Corrected Analysis

**CHANGELOG claim**: "58.8% success rate (67/114 runs)"
- ✅ **Technically correct** for combined Python+AILANG
- ❌ **Misleading** - implies AILANG is at 58.8%
- ✅ **Actual AILANG**: 38.6% (22/57)

**AILANG vs Python Gap**: 78.9% - 38.6% = **40.3% gap**
- Python is 2x more successful than AILANG
- This is expected (Python is mature, well-known)
- Goal: Close gap to ~15-20% by v0.4.0

---

## New Benchmarks Added (46 total)

Between v0.3.6 (68 runs) and v0.3.7 (114 runs), **46 new benchmark runs** were added.

**Passing new benchmarks** (AILANG):
- `recursion_fibonacci` ✅
- `list_comprehension` ✅
- `fizzbuzz` ✅
- `adt_option` ✅
- `recursion_factorial` ✅
- `simple_print` ✅
- `targeted_repair_test` ✅
- `nested_records` ✅
- `list_operations` ✅
- `error_handling` ✅
- `string_manipulation` ✅
- `higher_order_functions` ✅
- `float_eq` ✅
- `record_update` ✅
- `records_person` ✅

**Failing new benchmarks** (AILANG):
- `cli_args` ❌ (no CLI args support)
- `json_parse` ❌ (no JSON stdlib)
- `list_comprehension` ❌ (no comprehension syntax)
- `list_operations` ❌ (ADT lists too verbose)
- `higher_order_functions` ❌ (compilation issues)
- `error_handling` ❌ (verbose Result handling)
- `float_eq` ❌ (floating point precision?)
- `string_manipulation` ❌ (no string methods)
- `targeted_repair_test` ❌ (test-specific)

---

## Root Cause: Prompt Changes?

**Hypothesis**: Did the teaching prompt change between v0.3.6 and v0.3.7?

### Check Prompt Versions

```bash
# v0.3.6 baseline
grep "prompt_version" eval_results/baselines/v0.3.6-24/*.json | head -1

# v0.3.7 baseline
grep "prompt_version" eval_results/baselines/v0.3.7-1-gd24a7dc/*.json | head -1
```

**Need to investigate**:
1. What prompt version was used in v0.3.6?
2. What prompt version was used in v0.3.7?
3. Did prompt emphasize multi-line ADT syntax?

**Likely cause**: Prompt updated to show "better" formatting (multi-line ADTs), but parser doesn't support it yet!

---

## Key Insights

### 1. Multi-Line ADT Syntax is a Parser Bug

**Problem**: Parser requires all constructors on same line
```ailang
-- ✅ Works:
type Tree = Leaf(int) | Node(Tree, int, Tree)

-- ❌ Fails:
type Tree =
  | Leaf(int)
  | Node(Tree, int, Tree)
```

**Impact**: 2 AILANG benchmarks broke (gemini, gpt5 for pattern_matching_complex)

**Solution**: Add multi-line parsing support
- Estimate: ~100 LOC in parser
- Priority: P1 (causes 3.5% of AILANG failures)
- Difficulty: Medium (need to handle newlines in type declarations)

---

### 2. ADT Runtime Regression (adt_option/gpt5)

**Problem**: Something broke between v0.3.6 and v0.3.7 causing runtime errors

**Evidence**:
- Same benchmark (adt_option)
- Same model (gpt5)
- v0.3.6: ✅ passing
- v0.3.7: ❌ runtime_error

**Need to investigate**:
- Compare generated code (same seed should produce same code)
- Check CHANGELOG v0.3.7 for evaluator changes
- Run both versions with DEBUG logging

---

### 3. AILANG Success Rate Interpretation

**Correct metrics for v0.3.7**:
- **AILANG-only**: 38.6% (22/57) ← Use this for AILANG progress tracking
- **Python baseline**: 78.9% (45/57) ← Use this to measure language gap
- **Combined**: 58.8% (67/114) ← Use this for overall eval health

**Gap analysis**:
- AILANG is **40.3 percentage points** behind Python
- This gap represents missing features (CLI args, JSON, list syntax, etc.)
- v0.4.0 goal: Reduce gap to ~20 points (AILANG 60%, Python 80%)

---

## Recommendations

### Immediate Actions

#### 1. ⚠️ Add Multi-Line ADT Syntax Support (P1)
**Priority**: High (fixes 2 regressions)
**Estimate**: ~100 LOC parser changes, 2-3 hours
**File**: `internal/parser/parser.go` (type declaration parsing)

**Implementation**:
```go
// Allow newlines and optional leading | in type constructors
parseTypeDecl() {
  // After type name and =:
  skipNewlines()
  parseConstructor() // First constructor (no | needed)
  for peek() == PIPE {
    consume(PIPE)
    skipNewlines()
    parseConstructor()
  }
}
```

#### 2. 🔍 Investigate adt_option Runtime Regression (P0)
**Priority**: Critical (unexpected regression)
**Action**: Compare v0.3.6 vs v0.3.7 generated code for adt_option/gpt5/seed=42

```bash
# Extract and compare
jq '.code' eval_results/baselines/v0.3.6-24/adt_option_ailang_gpt5*.json > /tmp/v0.3.6.ail
jq '.code' eval_results/baselines/v0.3.7-1-gd24a7dc/adt_option_ailang_gpt5*.json > /tmp/v0.3.7.ail
diff /tmp/v0.3.6.ail /tmp/v0.3.7.ail
```

#### 3. ✅ Update CHANGELOG Metrics (Documentation)
**Change**: Clarify that 58.8% is combined Python+AILANG
**Add**: AILANG-only success rate (38.6%)
**Add**: Python baseline (78.9%)

**Suggested wording**:
```markdown
## Benchmark Results (M-EVAL)

**Overall Performance**: 58.8% success rate (67/114 runs across 3 models × 20 benchmarks × 2 languages)

**By Language:**
- **Python**: 78.9% (45/57) - Baseline for comparison
- **AILANG**: 38.6% (22/57) - New language, learning curve

**Gap**: 40.3 percentage points (expected for new language)
```

#### 4. 📝 Update Teaching Prompt (P2)
**Problem**: Prompt may encourage multi-line ADT syntax
**Action**: Check prompt examples, update to single-line syntax until parser supports multi-line

**Current (prompts/v0.3.0.md or later)**:
```ailang
-- ❌ Don't show this until parser supports it:
type Tree =
  | Leaf(int)
  | Node(Tree, int, Tree)

-- ✅ Show this instead:
type Tree = Leaf(int) | Node(Tree, int, Tree)
```

---

## v0.4.0 Roadmap Updates

### New High-Priority Item

**P1: Multi-Line ADT Syntax Support**
- **Estimate**: ~100 LOC, 2-3 hours
- **Impact**: Fixes 2 regressions, improves readability
- **Priority**: Should be done in Week 4 (Bug Fixes) or Week 1 (prep work)

**Updated Week 4 Plan**:
```
Week 4: Bug Fixes + Polish
- Days 1-2: Multi-line ADT syntax support (NEW)
- Days 2-3: Investigate ADT runtime regression (NEW)
- Days 4-5: Investigate modulo operator
- Days 6-7: REPL improvements, error messages
```

### Clarified Success Metrics

**Current (v0.3.7)**:
- AILANG: 38.6% (22/57)
- Python: 78.9% (45/57)
- Gap: 40.3 points

**Target (v0.4.0)**:
- AILANG: **60%+** (34/57 runs)
- Python: 80%+ (46/57 runs) ← maintain baseline
- Gap: **20 points** ← halve the gap

**Stretch Goal (v0.4.0)**:
- AILANG: **70%+** (40/57 runs)
- Gap: **15 points**

---

## References

- **Eval Results**: `eval_results/baselines/v0.3.7-1-gd24a7dc/`
- **Previous Baseline**: `eval_results/baselines/v0.3.6-24/`
- **Comparison Tool**: `ailang eval-compare`
- **CHANGELOG**: [CHANGELOG.md](../CHANGELOG.md)
- **Previous Analysis**: [EVAL_ANALYSIS_2025-10-15.md](EVAL_ANALYSIS_2025-10-15.md)

---

**Generated**: 2025-10-15
**Tool**: `ailang eval-compare` + manual analysis
**Auditor**: Claude Code

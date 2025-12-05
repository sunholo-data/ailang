# v0.4.3 Action Plan - ACTUAL Status (Corrected)

**Date**: 2025-11-04
**Based on**: v0.4.2 eval analysis + code audit
**Priority**: P0 (Addresses 33% of failures)

---

## ✅ What's ALREADY Implemented (v0.4.2)

### 1. Surface Sugar Pack - **COMPLETE** ✅
- **S-CALL0**: `f()` → `f(())` ✅ (statement + expression level)
- **S-CONS**: `x :: xs` → `::(x, xs)` ⚠️ **EXPRESSIONS ONLY** (not in patterns!)
- **S-ARROWTYPE**: `T -> U` → `funcType T U` ✅
- **Flag**: `--strict-syntax` to disable ✅
- **Status**: Implemented in v0.4.1, documented in prompt v0.4.1

**Code evidence**:
- `internal/parser/parser.go`: All three sugars registered
- `internal/parser/sugar_test.go`: Comprehensive test coverage
- `CHANGELOG.md` v0.4.2: S-CALL0 fix for zero-arg builtins

### 2. Comment Syntax - **COMPLETE** ✅
- Both `--` and `//` supported ✅
- Works inline and standalone ✅
- **Status**: Implemented, but may not be prominent in prompt

**Code evidence**:
- `internal/lexer/lexer.go:104-107`: Both comment styles supported

---

## 🚨 What's MISSING or BROKEN (v0.4.2)

### Critical Gap 1: S-CONS Pattern Limitation (36 PAR_001 errors)

**Problem**: `x :: xs` only works in EXPRESSIONS, not in PATTERNS

**Current behavior**:
```ailang
-- ✅ WORKS (expression)
let list = x :: xs :: [] in ...

-- ❌ FAILS (pattern) - PAR_001 error
match list {
  x :: xs => ...  -- Parse error!
}

-- ✅ MUST USE (pattern)
match list {
  ::(x, xs) => ...  -- Canonical form
}
```

**Impact**: 36 PAR_001 failures (12% of failures)

**Root cause**: Parser implementation is expression-only (design limitation?)

**Solutions**:
- **Option A**: Extend S-CONS to patterns (parser work needed)
- **Option B**: Clarify prompt to emphasize "expressions only" more prominently
- **Option C**: Both (extend parser + update prompt)

### Critical Gap 2: MOD_001 Errors (64 failures, 21%!)

**Problem**: Despite prompt updates, models still make module errors

**Common patterns** (need to analyze actual errors):
- Wrong effect syntax?
- Missing exports?
- Import syntax errors?
- Module path errors?

**Action needed**:
- [ ] Extract actual MOD_001 error messages from eval results
- [ ] Categorize by error type
- [ ] Determine if prompt issue or language issue

### Critical Gap 3: Agent Inefficiency

**Problem**: Even simple benchmarks take many turns

**Evidence**:
- `simple_print`: 21 turns (should be 3-5!)
- `effect_tracking_io_fs`: 28 turns
- `effect_composition`: 22 turns

**Root causes** (hypothesis):
- Prompt clarity issues
- Missing architecture docs (PIPELINE.md)
- No JSON AST inspection
- Benchmark hints have wrong syntax

---

## 📋 v0.4.3 Action Plan (Revised)

### Priority 0: Audit & Analysis (0.5 days)

**Before making ANY changes, we need data:**

1. **Audit prompts/v0.4.1.md**:
   - [ ] Is comment syntax in Quick Reference?
   - [ ] Is :: limitation emphasized enough?
   - [ ] Is effect syntax consistent throughout?
   - [ ] Are let bindings emphasized?

2. **Analyze MOD_001 errors**:
   ```bash
   # Extract actual error messages
   find eval_results/baselines/v0.4.2 -name "*.json" -type f \
     | xargs jq -r 'select(.lang == "ailang" and .err_code == "MOD_001") | {benchmark, stderr}' \
     | jq -s 'group_by(.stderr) | map({error: .[0].stderr, count: length, benchmarks: [.[].benchmark] | unique})' \
     > mod_001_analysis.json
   ```

3. **Analyze PAR_001 errors**:
   ```bash
   # Check if errors are :: in patterns
   find eval_results/baselines/v0.4.2 -name "*.json" -type f \
     | xargs jq -r 'select(.lang == "ailang" and .err_code == "PAR_001") | {benchmark, code: .generated_code, error: .stderr}' \
     | jq -s '.[] | select(.error | contains("::"))' \
     > par_001_cons_analysis.json
   ```

4. **Audit benchmark hints**:
   ```bash
   # Check for wrong AILANG syntax in benchmark prompts
   grep -r "!:" benchmarks/*.yml  # Wrong effect syntax
   grep -r "func.*() ->" benchmarks/*.yml  # Check zero-arg syntax
   ```

### Priority 1: Targeted Fixes (Based on Analysis)

**After analysis, prioritize:**

#### If PAR_001 is mostly :: in patterns:
- **Option A**: Extend parser to support :: in patterns (1-2 days)
- **Option B**: Emphasize limitation in prompt (0.5 days)
- **Decision**: Depends on parser complexity

#### If MOD_001 is syntax confusion:
- **Fix**: Update prompt with clearer module checklist (0.5 days)
- **Fix**: Update benchmark hints with correct syntax (0.5 days)

#### If MOD_001 is semantic errors:
- **Research**: May need language simplification (v0.5.0)

### Priority 2: DX Improvements (v0.4.4+)

- [ ] PIPELINE.md (2-3 hours) - IF agent turn analysis shows confusion
- [ ] JSON AST inspection (3-4 hours) - IF needed
- [ ] Module simplification design (TBD) - IF MOD_001 persists

---

## 🎯 Expected Impact (Realistic)

### If we fix :: in patterns (parser + prompt):
- PAR_001: 36 → ~5-10 (eliminate cons errors)
- AILANG success: 54.2% → ~58-60%

### If we fix MOD_001 (prompt clarity + benchmarks):
- MOD_001: 64 → ~30-40 (reduce confusion, not eliminate)
- AILANG success: 54.2% → ~60-62%

### Combined (optimistic):
- PAR_001: 36 → ~5
- MOD_001: 64 → ~30
- **AILANG success: 54.2% → 62-65%**
- **Gap vs Python: 21.9pp → ~15-18pp**

---

## ⚠️ Key Learnings

1. **Don't assume - verify**: Surface sugar WAS implemented, I assumed it wasn't
2. **Read the code**: CHANGELOG and grep are essential
3. **Check prompt versions**: versions.json tells you what's active
4. **Partial features are tricky**: :: works in expressions but not patterns - subtle!

---

## 🔍 Next Steps (Immediate)

**For you to decide:**

1. **Should we extend :: to patterns?** (parser work)
   - Pro: Eliminates 12% of failures
   - Con: 1-2 days work, may have edge cases
   - Alternative: Just clarify prompt (0.5 days)

2. **What's causing MOD_001 errors?** (need analysis)
   - Extract actual error messages
   - Categorize by type
   - Determine if fixable with prompt or needs language changes

3. **Which to prioritize: prompt fixes or parser extensions?**
   - Prompt: Fast, safe, low ROI
   - Parser: Slower, some risk, higher ROI

---

## 📚 Files to Reference

**Prompt**:
- `prompts/v0.4.1.md` - Current active prompt
- `prompts/versions.json` - Prompt version tracking

**Parser Sugar**:
- `internal/parser/parser.go` - Sugar registration
- `internal/parser/sugar_test.go` - Test coverage
- `internal/parser/parser_expr.go` - Expression parsing

**Analysis**:
- `eval_results/baselines/v0.4.2/summary.jsonl` - Eval summary
- Design doc: `m-prompt-improvements-module-clarity.md` (needs updating!)


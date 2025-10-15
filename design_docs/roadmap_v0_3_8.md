# AILANG v0.3.8 Roadmap - Critical Bug Fixes

**Status**: Planning
**Target Date**: Week of 2025-10-21 (1 week sprint)
**Focus**: Regression fixes and parser improvements
**Type**: Patch release

---

## Version Context

**Current Version**: v0.3.7 (October 15, 2025)
- M-EVAL success: 38.6% AILANG (22/57 runs)
- 5 regressions identified from v0.3.6 → v0.3.7
- Multi-line ADT syntax breaks parser
- ADT runtime regression in adt_option benchmark

**Next Patch**: v0.3.8 (This roadmap)
- Focus: Fix critical regressions
- Goal: Restore v0.3.6 success rate (42/68 = 61.8% for common benchmarks)
- Timeline: 3-4 days of bug fixes

**Next Major**: v0.4.0 (Q1 2026)
- Focus: Language completeness & developer experience
- See [roadmap_v0_4_0.md](roadmap_v0_4_0.md)

---

## Objectives for v0.3.8

### Primary Goals

1. **Fix v0.3.7 regressions** (5 broken benchmarks)
2. **Add multi-line ADT syntax support** (parser improvement)
3. **Restore eval success rate** to v0.3.6 levels or better
4. **Update teaching prompts** to prevent future regressions

### Success Criteria

**M-EVAL Targets**:
- Fix 2 pattern_matching_complex failures (multi-line ADT)
- Fix 1 adt_option failure (runtime regression)
- AILANG success: 38.6% → **42%+** (restore v0.3.6 parity)
- No new regressions

**Quality Targets**:
- All v0.3.6 passing benchmarks still pass
- Multi-line ADT syntax supported (readability improvement)
- Teaching prompt updated with correct syntax

---

## Critical Issues (P0)

### 1. 🐛 Multi-Line ADT Syntax Not Supported

**Priority**: P0 (Blocks 2 AILANG benchmarks)
**Discovered**: 2025-10-15 (eval comparison v0.3.6 → v0.3.7)
**Impact**: 3.5% of AILANG failures (2/57 runs)

#### Problem Description

AI models generate "prettier" multi-line ADT syntax, but parser rejects it:

```ailang
-- ❌ CURRENT: Parser fails
type Tree =
  | Leaf(int)
  | Node(Tree, int, Tree)

-- Error: PAR_TYPE_BODY_EXPECTED at line 2:3: expected type definition

-- ✅ WORKAROUND: Single line works
type Tree = Leaf(int) | Node(Tree, int, Tree)
```

**Affected Benchmarks**:
- `pattern_matching_complex` (ailang, gemini-2.5-pro)
- `pattern_matching_complex` (ailang, gpt5)

**Root Cause**: Parser expects all constructors on same line after `=`

#### Implementation Plan

**File**: `internal/parser/parser.go` (type declaration parsing)

**Current Code** (approximate):
```go
func (p *Parser) parseTypeDecl() (*ast.TypeDecl, error) {
    // ... parse "type Name ="

    // Parse first constructor
    ctor1 := p.parseConstructor()

    // Parse additional constructors
    constructors := []ast.Constructor{ctor1}
    for p.peek() == PIPE {
        p.consume(PIPE)
        ctor := p.parseConstructor()
        constructors = append(constructors, ctor)
    }
}
```

**Updated Code** (with newline support):
```go
func (p *Parser) parseTypeDecl() (*ast.TypeDecl, error) {
    // ... parse "type Name ="

    // Allow newlines after =
    p.skipNewlinesAndComments()

    // Optional leading | for first constructor (Haskell-style)
    if p.peek() == PIPE {
        p.consume(PIPE)
        p.skipNewlinesAndComments()
    }

    // Parse first constructor
    ctor1 := p.parseConstructor()
    constructors := []ast.Constructor{ctor1}

    // Parse additional constructors
    for p.peek() == PIPE {
        p.consume(PIPE)
        p.skipNewlinesAndComments()  // NEW: Allow newlines after |
        ctor := p.parseConstructor()
        constructors = append(constructors, ctor)
    }
}
```

**New Helper Function**:
```go
func (p *Parser) skipNewlinesAndComments() {
    for p.peek() == NEWLINE || p.peek() == COMMENT {
        p.advance()
    }
}
```

**Estimate**: ~100-150 LOC changes
**Duration**: 2-3 hours (including tests)
**Tests**: Add to `internal/parser/parser_test.go`

**Test Cases**:
```go
func TestMultiLineADT(t *testing.T) {
    tests := []struct {
        name string
        input string
        want string
    }{
        {
            name: "Single line (existing)",
            input: "type Option[a] = Some(a) | None",
            want: "Option with 2 constructors",
        },
        {
            name: "Multi-line with leading |",
            input: `type Tree =
  | Leaf(int)
  | Node(Tree, int, Tree)`,
            want: "Tree with 2 constructors",
        },
        {
            name: "Multi-line without leading |",
            input: `type Result[a, e] =
  Ok(a)
  | Err(e)`,
            want: "Result with 2 constructors",
        },
    }
    // ... test implementation
}
```

#### Expected Results

**Before Fix**:
- v0.3.7 AILANG: 22/57 passing (38.6%)
- pattern_matching_complex failures: 2

**After Fix**:
- v0.3.8 AILANG: 24/57 passing (42.1%)
- pattern_matching_complex failures: 0
- All single-line ADT syntax still works (backwards compatible)

---

### 2. 🐛 ADT Runtime Regression (adt_option)

**Priority**: P0 (Unexpected regression)
**Discovered**: 2025-10-15 (eval comparison v0.3.6 → v0.3.7)
**Impact**: 1.8% of AILANG failures (1/57 runs)

#### Problem Description

**Benchmark**: `adt_option` (ailang, gpt5)
- **v0.3.6**: ✅ Compile OK, Runtime OK, Output OK
- **v0.3.7**: ✅ Compile OK, ❌ Runtime Error, ❌ Output mismatch

**Error**: Runtime error (details TBD - need to extract from eval results)

**Root Cause**: Unknown - needs investigation

#### Investigation Plan

**Step 1: Compare Generated Code**

```bash
# Extract v0.3.6 generated code
jq -r '.code' eval_results/baselines/v0.3.6-24/adt_option_ailang_gpt5*.json > /tmp/v0.3.6_adt_option.ail

# Extract v0.3.7 generated code
jq -r '.code' eval_results/baselines/v0.3.7-1-gd24a7dc/adt_option_ailang_gpt5*.json > /tmp/v0.3.7_adt_option.ail

# Compare
diff /tmp/v0.3.6_adt_option.ail /tmp/v0.3.7_adt_option.ail
```

**Expected Outcome**:
- If code is **identical**: Runtime evaluator bug in v0.3.7
- If code is **different**: Prompt change caused different generation

**Step 2: Test Both Versions**

```bash
# Test v0.3.6 code with v0.3.7 interpreter
ailang run /tmp/v0.3.6_adt_option.ail

# Test v0.3.7 code with v0.3.7 interpreter
ailang run /tmp/v0.3.7_adt_option.ail
```

**Step 3: Check v0.3.7 CHANGELOG**

Review changes in v0.3.7:
- Evaluator changes?
- Type system changes?
- Pattern matching changes?
- Cost calculation cleanup (may have affected runtime?)

**Step 4: Git Bisect (if needed)**

```bash
git bisect start
git bisect bad v0.3.7        # Runtime error
git bisect good v0.3.6       # Works
# Test each commit with: ailang run /tmp/v0.3.6_adt_option.ail
```

#### Potential Root Causes

1. **Cost Calculation Cleanup** (v0.3.7)
   - CHANGELOG: "Removed unused `CalculateCost` function"
   - Could have affected runtime evaluation?
   - Check: `internal/eval_harness/metrics.go`

2. **Float Display Formatting** (v0.3.3)
   - Already fixed in v0.3.3, unlikely cause
   - But check if related to show() function

3. **Type System Changes**
   - Unlikely - v0.3.7 was just cleanup
   - But check git diff between v0.3.6 and v0.3.7

4. **Pattern Matching Changes**
   - Check if ADT pattern matching evaluator changed

#### Implementation Plan

**Duration**: 2-4 hours investigation + 1-2 hours fix
**Estimate**: ~50-100 LOC fix (depending on root cause)

**Once root cause identified**:
- Add regression test for adt_option
- Fix evaluator/runtime bug
- Verify all ADT examples still work

---

## Secondary Issues (P1)

### 3. 📝 Update Teaching Prompt

**Priority**: P1 (Prevents future regressions)
**Impact**: Reduces wrong syntax generation

#### Problem Description

Teaching prompt may show multi-line ADT syntax that doesn't parse (until we fix the parser), or may not emphasize limitations clearly enough.

**Current prompt** (check `prompts/v0.3.0.md` or later):
```ailang
-- May show this (doesn't work until v0.3.8):
type Tree =
  | Leaf(int)
  | Node(Tree, int, Tree)
```

**Updated prompt** (for v0.3.8):
```ailang
-- Single-line syntax (current requirement):
type Option[a] = Some(a) | None
type Tree = Leaf(int) | Node(Tree, int, Tree)

-- Multi-line syntax (SUPPORTED in v0.3.8+):
type Result[a, e] =
  | Ok(a)
  | Err(e)
```

#### Implementation Plan

**Files to Update**:
1. `prompts/v0.3.8.md` (new version)
2. `prompts/versions.json` (add v0.3.8 entry, set as active)
3. `CLAUDE.md` (reference updated prompt)

**Changes Needed**:
- ✅ Add multi-line ADT examples (after parser fix)
- ✅ Clarify that both single-line and multi-line work
- ✅ Emphasize tested syntax over "pretty" syntax
- ✅ Add note: "Prefer single-line for simple types, multi-line for complex types"

**Estimate**: 1-2 hours
**Impact**: Prevents future syntax regressions

---

## Sprint Plan (v0.3.8 Development)

### Timeline: 3-4 Days

**Day 1: Investigation & Planning**
- Morning: Investigate ADT runtime regression (adt_option/gpt5)
  - Extract and compare generated code
  - Test both versions
  - Identify root cause
- Afternoon: Create minimal reproduction cases
  - ADT runtime bug
  - Multi-line ADT syntax
- Deliverable: Root cause analysis document

**Day 2: Parser Fix**
- Morning: Implement multi-line ADT syntax support
  - Update `parseTypeDecl()` in `internal/parser/parser.go`
  - Add `skipNewlinesAndComments()` helper
  - Handle optional leading `|`
- Afternoon: Write parser tests
  - Test single-line (backwards compat)
  - Test multi-line with leading `|`
  - Test multi-line without leading `|`
  - Test mixed newlines and comments
- Evening: Run full test suite
- Deliverable: Parser supports multi-line ADT syntax

**Day 3: Runtime Fix & Validation**
- Morning: Fix ADT runtime regression
  - Implement fix based on Day 1 investigation
  - Add regression test for adt_option
- Afternoon: Run M-EVAL partial baseline
  - Test affected benchmarks only (pattern_matching_complex, adt_option)
  - Verify no new regressions
- Evening: Run full M-EVAL if partial tests pass
- Deliverable: All v0.3.7 regressions fixed

**Day 4: Documentation & Release**
- Morning: Update teaching prompt (prompts/v0.3.8.md)
  - Add multi-line ADT examples
  - Update versions.json
- Afternoon: Update CHANGELOG.md
  - Document bug fixes
  - Add eval results
  - Note backwards compatibility
- Evening: Run final M-EVAL baseline
- Deliverable: v0.3.8 ready for release

---

## Testing Strategy

### Unit Tests

**Parser Tests** (`internal/parser/parser_test.go`):
```go
func TestMultiLineADTSyntax(t *testing.T) {
    // Single line (backwards compat)
    // Multi-line with leading |
    // Multi-line without leading |
    // Mixed with comments
}
```

**Evaluator Tests** (based on ADT regression):
```go
func TestADTRuntimeRegression(t *testing.T) {
    // Minimal reproduction of adt_option bug
    // Ensure pattern matching works
    // Ensure ADT values evaluate correctly
}
```

### Integration Tests

**Run affected benchmarks**:
```bash
# Pattern matching
ailang eval-suite --benchmark=pattern_matching_complex --model=gemini-2-5-pro --lang=ailang
ailang eval-suite --benchmark=pattern_matching_complex --model=gpt5 --lang=ailang

# ADT runtime
ailang eval-suite --benchmark=adt_option --model=gpt5 --lang=ailang
```

**Run all ADT examples**:
```bash
make verify-examples
# Check: adt_simple.ail, adt_option.ail, demos/adt_pipeline.ail
```

### Regression Prevention

**M-EVAL Baseline Comparison**:
```bash
# Create v0.3.8 baseline
ailang eval-suite --full --models=claude-sonnet-4-5,gemini-2-5-pro,gpt5

# Compare with v0.3.7
ailang eval-compare eval_results/baselines/v0.3.7-1-gd24a7dc eval_results/baselines/v0.3.8-X-YYYYYY

# Expect:
# - 2 fixed: pattern_matching_complex (gemini, gpt5)
# - 1 fixed: adt_option (gpt5)
# - 0 broken: No new regressions
```

---

## Success Metrics

### Quantitative Goals

| Metric | v0.3.7 (Current) | v0.3.8 (Target) | Status |
|--------|-----------------|----------------|--------|
| AILANG Success | 22/57 (38.6%) | **24/57 (42.1%)** | +2 runs |
| Broken from v0.3.6 | 5 | **0** | All fixed |
| Multi-line ADT | ❌ Not supported | ✅ Supported | Feature |
| Parser Coverage | 70.8% | 72%+ | +1.2% |

### Qualitative Goals

✅ **Backwards Compatibility**: All v0.3.7 working code still works
✅ **Readability**: Multi-line ADT syntax improves code readability
✅ **AI Codegen**: Models can generate "prettier" code without breaking
✅ **Regression Prevention**: Teaching prompt updated to prevent future issues

---

## Risk Assessment

### Low Risk Items

1. **Multi-line ADT Parser** - Isolated change, easy to test
2. **Teaching Prompt Update** - Documentation only, no code impact

### Medium Risk Items

3. **ADT Runtime Fix** - Depends on root cause complexity
   - Mitigation: Thorough investigation on Day 1
   - Fallback: Revert problematic v0.3.7 changes if needed

### High Risk Items

None identified (v0.3.8 is a focused bug fix release)

---

## Post-Release Actions

### Immediate (Day 1 after release)

1. **Run full M-EVAL baseline**
   - Compare v0.3.8 with v0.3.7
   - Verify 2-3 fixes, 0 regressions

2. **Update CHANGELOG.md**
   - Add v0.3.8 results
   - Document success rate improvement

3. **Update roadmap_v0_4_0.md**
   - Remove "investigate multi-line ADT" (done in v0.3.8)
   - Remove "investigate ADT runtime bug" (done in v0.3.8)

### Follow-up (Week after release)

4. **Monitor benchmark stability**
   - Run daily M-EVAL for 1 week
   - Check for any latent regressions

5. **Community feedback**
   - Check GitHub issues for bug reports
   - Monitor eval results from external users

---

## Changelog Template (v0.3.8)

```markdown
## [v0.3.8] - 2025-10-21 - Bug Fix Release

### Fixed - Multi-Line ADT Syntax Support

**Parser Enhancement**: Added support for multi-line algebraic data type declarations.

**Before (v0.3.7)**:
```ailang
-- ❌ Only this worked:
type Tree = Leaf(int) | Node(Tree, int, Tree)

-- ❌ This failed:
type Tree =
  | Leaf(int)
  | Node(Tree, int, Tree)
```

**After (v0.3.8)**:
```ailang
-- ✅ Both work:
type Tree = Leaf(int) | Node(Tree, int, Tree)

type Tree =
  | Leaf(int)
  | Node(Tree, int, Tree)
```

**Impact**: Fixes 2 M-EVAL benchmark failures (pattern_matching_complex)
- AI models can now generate more readable multi-line ADT definitions
- Backwards compatible: Single-line syntax still works

**Implementation**:
- `internal/parser/parser.go` - Updated `parseTypeDecl()` to handle newlines
- Added `skipNewlinesAndComments()` helper
- Supports optional leading `|` (Haskell-style)

**Files Modified**:
- `internal/parser/parser.go` (~100 LOC)
- `internal/parser/parser_test.go` (~80 LOC tests)

---

### Fixed - ADT Runtime Regression

**Issue**: `adt_option` benchmark passed in v0.3.6 but failed at runtime in v0.3.7.

**Root Cause**: [TO BE FILLED IN AFTER INVESTIGATION]

**Fix**: [TO BE FILLED IN AFTER IMPLEMENTATION]

**Impact**: Restores runtime evaluation for ADT pattern matching
- `adt_option` benchmark now passes
- All ADT examples work correctly

**Files Modified**:
- [TO BE FILLED IN]

---

### Updated - Teaching Prompt (v0.3.8)

**Prompt Version**: v0.3.8
**Active**: Yes (set in `prompts/versions.json`)

**Changes**:
- ✅ Added multi-line ADT syntax examples
- ✅ Clarified single-line vs multi-line usage
- ✅ Emphasized tested syntax patterns
- ✅ Added note: "Prefer single-line for simple types, multi-line for complex types"

**Files Modified**:
- `prompts/v0.3.8.md` (new version)
- `prompts/versions.json` (updated active)

---

### Benchmark Results (M-EVAL)

**Overall Performance**: 42.1% AILANG success (24/57 runs)
**Improvement**: +3.5% vs v0.3.7 (38.6% → 42.1%)

**Fixed Benchmarks** (3):
- ✅ `pattern_matching_complex` (ailang, gemini-2.5-pro)
- ✅ `pattern_matching_complex` (ailang, gpt5)
- ✅ `adt_option` (ailang, gpt5)

**Regression Status**: 0 new regressions (all v0.3.7 passing tests still pass)

**Comparison with v0.3.6**:
- v0.3.6: 42/68 (61.8%) - smaller benchmark set
- v0.3.8: 24/57 (42.1%) - larger benchmark set, different baselines
- Result: Parity restored for common benchmarks
```

---

## References

- **Eval Analysis**: [EVAL_ANALYSIS_2025-10-15.md](EVAL_ANALYSIS_2025-10-15.md)
- **Eval Comparison**: [EVAL_COMPARISON_2025-10-15.md](EVAL_COMPARISON_2025-10-15.md)
- **v0.3.7 Results**: `eval_results/baselines/v0.3.7-1-gd24a7dc/`
- **v0.3.6 Results**: `eval_results/baselines/v0.3.6-24/`
- **v0.4.0 Roadmap**: [roadmap_v0_4_0.md](roadmap_v0_4_0.md)

---

**Document Status**: ✅ **READY FOR IMPLEMENTATION**
**Target Start**: 2025-10-21
**Estimated Completion**: 2025-10-24 (4 days)
**Owner**: Core Development Team

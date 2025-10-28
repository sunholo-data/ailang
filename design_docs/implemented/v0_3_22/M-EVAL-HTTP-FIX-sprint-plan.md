# Sprint Plan: M-EVAL-HTTP - Fix HTTP/JSON Syntax Errors (75% AILANG Failure Rate)

## Status: ✅ IMPLEMENTED (v0.3.17 + v0.3.22) | ❌ SUCCESS METRICS NOT ACHIEVED

**Completed**: October 2025 (Milestones 1-2)
**Validation**: v0.3.22 baseline run (Milestone 3)
**Result**: Implementation complete, but AIs still failing at same rate
**Actual Duration**: ~8 hours (Milestone 1: 6h, Milestone 2: 2h)

## Summary
Reduce AI-generated HTTP/JSON syntax errors from 75% to <10% by enhancing parser error messages and improving prompt placement. The v0.3.21 prompt **already has HTTP examples** (line 218), but they appear too late for AIs to notice them before generating code.

**Original Duration:** 2-3 days
**Dependencies:** None - can start immediately
**Risk Level:** Low (no language changes, only error messages + prompt reorganization)

## Current Status Analysis

### Problem Identified (eval-analyzer findings)
- **api_call_json benchmark**: 75% failure rate (3/4 runs failing)
- **All failures**: Parse errors (not runtime/logic)
- **Root cause**: AIs generate Python/JS syntax despite HTTP example existing in prompt

### Evidence
```ailang
-- ❌ AIs generate (wrong):
import http from 'http'          // JavaScript ES6
const URL = "..."                // JavaScript const
url = "..."                      // Python bare assignment
http.post(url, headers=headers)  // Python OOP

-- ✅ Prompt shows (correct, but AIs miss it):
import std/net (httpRequest)
let jsonBody = encode(jo([...])) in
httpRequest("POST", url, headers, jsonBody)
```

### Current Prompt Status (v0.3.21.md)
- ✅ HTTP POST example exists at line 218
- ❌ Example appears **after** 200+ lines of other syntax
- ❌ No anti-pattern warnings in critical section (lines 1-100)
- ❌ Error messages don't guide to correct syntax

### Recent Velocity (last 14 days)
- **v0.3.21**: Parser bug fix (~60 LOC implementation + 425 LOC tests) in 1 day
- **v0.3.20**: Property-based testing (~5,750 LOC) in 10 days = 575 LOC/day
- **Average**: ~150-200 LOC/day for focused bugfixes
- **Estimate**: This sprint should be ~230 LOC total (well within capacity)

## Root Cause Analysis

### Why HTTP Example Doesn't Help (Despite Existing)

1. **Placement Issue**: HTTP example is at line 218 in a 1,096-line prompt
2. **AI Attention**: Models focus on first 100-200 lines, then skip to bottom
3. **No Anti-Patterns**: Prompt doesn't warn against `import http from` or `const`
4. **Error Messages**: Parser errors don't suggest correct syntax

### Failure Pattern Breakdown

| Pattern | Frequency | AI Behavior |
|---------|-----------|-------------|
| `import http from 'http'` | 3/3 | Default to JavaScript ES6 |
| `const URL = "..."` | 1/3 | Assume const exists |
| `url = "..."` (bare assignment) | 2/3 | Python-style assignment |
| `http.post(...)` (OOP) | 3/3 | Assume OOP library |

## Proposed Milestones

### Milestone 1: Enhanced Parser Error Messages (Priority 1)
**Goal:** When AIs generate wrong syntax, guide them to correct syntax
**Estimated:** ~80 LOC implementation + ~100 LOC tests = 180 LOC
**Duration:** 1 day

**Tasks:**
- **Hour 1-2**: Add `SuggestionError` type to `internal/errors/parser_errors.go`
  - Fields: `Code`, `Message`, `Suggestions []string`, `HelpURL`
  - Example format for error output

- **Hour 3-4**: Detect `import X from Y` pattern in `parseImport()`
  - Check for `FROM` token after identifier
  - Return `SuggestionError` with:
    - `import std/net (httpRequest)` for HTTP
    - `import std/json (encode, decode)` for JSON
    - Link: `https://sunholo-data.github.io/ailang/docs/reference/language-syntax#imports`

- **Hour 5-6**: Detect `const` keyword in `parseStatement()`
  - Check for `CONST` token (add to lexer if missing)
  - Suggest: `Use: let name = value in ...`
  - Note: "All bindings immutable by default"

- **Hour 7-8**: Enhance bare assignment errors in `parseExpression()`
  - Detect `IDENT ASSIGN` without `let`
  - Suggest: `let x = value in ...`
  - Link: `https://sunholo-data.github.io/ailang/docs/guides/getting-started#let-bindings`

**Acceptance Criteria:**
- [ ] `echo 'import http from "http"' | ailang check -` shows suggestion for `std/net`
- [ ] `echo 'const x = 5' | ailang check -` shows suggestion for `let`
- [ ] `echo 'x = 5' | ailang check -` shows suggestion for `let ... in`
- [ ] All 3 error messages include working URL links
- [ ] Unit tests cover all 3 patterns
- [ ] All existing tests still pass

**Risks:**
- Lexer may not have `CONST` token → Add if missing (~10 LOC in lexer)
- Bare assignment already has error → Enhance, don't replace

**Files to Modify:**
- `internal/errors/parser_errors.go` (~30 LOC new)
- `internal/parser/parser.go` (~50 LOC modifications)
- `internal/parser/parser_errors_test.go` (~100 LOC tests)
- `internal/lexer/token.go` (~5 LOC if CONST missing)
- `internal/lexer/lexer.go` (~5 LOC if CONST missing)

---

### Milestone 2: Prompt Reorganization (Priority 2)
**Goal:** Move HTTP example to early position + add anti-pattern warnings
**Estimated:** ~50 LOC modifications (moving text, not adding much)
**Duration:** 0.5 days (4 hours)

**Tasks:**
- **Hour 1**: Analyze prompt structure
  - Current order: Basics → Syntax → (200 lines) → HTTP → (800 lines)
  - Proposed: Basics → **Anti-Patterns** → HTTP Example → Syntax → Advanced

- **Hour 2**: Add anti-pattern section (lines 50-80)
  ```markdown
  ## ❌ Common Mistakes (Do NOT Use These)

  ### JavaScript/Python Imports (WRONG)
  ```javascript
  import http from 'http'  // ❌ JavaScript ES6
  import http              // ❌ Python style
  ```

  ### AILANG Imports (CORRECT)
  ```ailang
  import std/net (httpRequest)  // ✅ Selective import
  import std/json (encode)      // ✅ Explicit symbols
  ```

  ### Other Anti-Patterns
  - ❌ `const URL = "..."` → Use `let url = "..."` (no const keyword!)
  - ❌ `url = "..."` → Use `let url = "..." in` (no bare assignment!)
  - ❌ `http.post(...)` → Use `httpRequest("POST", ...)` (no OOP!)
  ```

- **Hour 3**: Move HTTP example forward (after anti-patterns)
  - Cut lines 218-260 (HTTP POST example)
  - Paste at line 80 (right after anti-patterns)
  - Update line numbers in links

- **Hour 4**: Test prompt with all 3 dev models
  - Create test benchmark: `benchmarks/http_syntax_test.yml`
  - Run: `ailang eval-suite --benchmarks http_syntax_test --models gpt5-mini,claude-haiku-4-5,gemini-2-5-flash`
  - Verify: 3/3 generate correct syntax

**Acceptance Criteria:**
- [ ] Anti-patterns section appears in first 80 lines
- [ ] HTTP example appears in first 100 lines
- [ ] All internal links still work
- [ ] Prompt validates: `ailang doctor prompts`
- [ ] Hash updated in `prompts/versions.json`
- [ ] Test run shows ≥2/3 models generate correct syntax

**Risks:**
- Moving text may break cross-references → Check all links after move
- Prompt size increase ~50 tokens → Acceptable (<2% increase)

**Files to Modify:**
- `prompts/v0.3.22.md` (new version, copy from v0.3.21.md, ~50 LOC changes)
- `prompts/versions.json` (~5 LOC, add v0.3.22 entry with hash)

---

### Milestone 3: Validation & Eval Baseline (Priority 3)
**Goal:** Measure improvement on actual benchmarks
**Estimated:** No new code, just testing time
**Duration:** 0.5 days (4 hours)

**Tasks:**
- **Hour 1**: Re-run `api_call_json` with dev models
  ```bash
  ailang eval-suite \
    --benchmarks api_call_json \
    --models gpt5-mini,claude-haiku-4-5,gemini-2-5-flash \
    --prompt-version v0.3.22 \
    --output eval_results/verification/v0.3.22_http_fix
  ```

- **Hour 2**: Compare before/after
  ```bash
  # Before (v0.3.21): 3/4 failures (75%)
  # After  (v0.3.22): Target <1/4 failures (<25%)

  ailang eval-compare \
    eval_results/baselines/v0.3.21 \
    eval_results/verification/v0.3.22_http_fix
  ```

- **Hour 3**: Analyze any remaining failures
  - If still failing → Check which pattern (namespace, const, assignment)
  - If new failure mode → Document and add to error messages
  - If success → Document victory!

- **Hour 4**: Update design doc with results
  - Actual success rate improvement
  - Remaining failure patterns (if any)
  - Recommendations for future work

**Acceptance Criteria:**
- [ ] api_call_json success rate: ≥75% (target: 100%)
- [ ] Failure breakdown by pattern documented
- [ ] Design doc updated with actual metrics
- [ ] Commit results to git

**Risks:**
- AIs may find new creative ways to fail → Document and defer to v0.3.23
- 100% success unlikely (API variance) → 75%+ is acceptable

**Files to Modify:**
- `design_docs/planned/20251022_compile_error_ailang_compilation_failures.md` (update with results)
- `CHANGELOG.md` (add v0.3.22 entry with improvement metrics)

## Success Metrics

**Quantitative:**
- [ ] api_call_json failure rate: 75% → <25% (target: <10%)
- [ ] Enhanced error messages: 3/3 patterns detected
- [ ] Test coverage: 100% for new error handling code
- [ ] All unit tests passing: ✅
- [ ] Linting clean: ✅

**Qualitative:**
- [ ] Error messages guide AIs to correct syntax
- [ ] Prompt anti-patterns reduce initial errors
- [ ] HTTP example is prominent and findable
- [ ] No regressions in other benchmarks

**Documentation:**
- [ ] `CHANGELOG.md` updated with v0.3.22 entry
- [ ] `prompts/versions.json` has v0.3.22 entry
- [ ] Design doc updated with actual results
- [ ] URLs in error messages verified working

## Implementation Order

**Day 1 (8 hours): Milestone 1 - Enhanced Errors**
1. Morning: Implement SuggestionError type + namespace import detection
2. Afternoon: Add const and bare assignment detection + tests
3. End of day: All error message tests passing

**Day 2 (4 hours): Milestone 2 - Prompt Reorganization**
1. Morning: Add anti-patterns section + move HTTP example forward
2. Afternoon: Test with dev models, update versions.json

**Day 2-3 (4 hours): Milestone 3 - Validation**
1. Run eval baseline with v0.3.22 prompt
2. Compare results, analyze failures
3. Update design doc + CHANGELOG

**Total: 16 hours (~2 days)**

## Dependencies

**None** - This work can start immediately:
- No blocking features
- No external dependencies
- No coordination required

## Open Questions

1. **Should we add `const` keyword to lexer?**
   - Current: Lexer may not recognize `const` as keyword
   - Option A: Add as keyword (parsed but rejected)
   - Option B: Treat as IDENT, error in parser
   - **Recommendation**: Option B (simpler, no lexer changes)

2. **What about TypeScript imports (`import {x} from 'y'`)?**
   - Not seen in eval failures yet
   - **Recommendation**: Defer until we see evidence of this pattern

3. **Should error URLs point to main site or localhost during dev?**
   - **Recommendation**: Use production URLs (`https://sunholo-data.github.io/ailang/...`)
   - Verify URLs exist before committing

## Notes

### Key Insight from Analysis
The HTTP example **already exists** in v0.3.21 prompt (line 218), but:
1. It's too far down (after 200 lines)
2. No anti-patterns warning AIs away from wrong syntax
3. Parser errors don't guide to correct syntax

**Solution is reorganization + error enhancement, not adding new content.**

### Velocity Assumptions
- ~80 LOC/hour for focused error handling work
- ~30 minutes to reorganize/move existing text
- ~2 hours eval validation per test run
- Total: 16 hours = 2 working days

### Conservative Estimate
Design doc estimated 4-6 hours, but includes:
- Analyzing existing prompt (already done)
- Testing multiple prompt versions (already done in previous evals)

**Realistic timeline: 2-3 days** to be conservative with:
- Unexpected test failures
- Eval baseline runtime (long-running)
- Documentation polish

### Success Threshold
- **Minimum acceptable**: 50% → 75% (25pp improvement)
- **Target**: 75% → 90% (15% residual failure acceptable)
- **Stretch**: 75% → 100% (perfect, but unlikely due to API variance)

---

## Implementation Summary

### What Was Built

**Milestone 1: Enhanced Parser Errors** ✅ (v0.3.17, M-COMPILE-ERROR)
- Enhanced ParserError with Suggestions field
- Detection for 3 patterns:
  1. JavaScript namespace imports: `import X from 'Y'` → IMP012_UNSUPPORTED_NAMESPACE
  2. Const keyword: `const x = y` → PAR014
  3. Bare assignment: `x = y` → PAR015
- Files: `internal/parser/parser_error.go` (+30 LOC), `parser_decl.go` (+50 LOC)
- Tests: 470 LOC (100% passing)
- **Completed**: v0.3.17 (Oct 2025)

**Milestone 2: Prompt Reorganization** ✅ (v0.3.22)
- HTTP POST example moved from line 218 → line 107 (early position!)
- Prompt title: "AI Teaching Prompt (Enhanced Error Messages + HTTP Early)"
- Anti-pattern warnings in "Critical: What AILANG is NOT" section (lines 21-83)
- Files: `prompts/v0.3.22.md` (new version)
- **Completed**: Oct 27, 2025 (commit 1a8d322)

**Milestone 3: Validation & Eval Baseline** ⏳ PARTIAL (v0.3.22)
- Eval baseline run on Oct 27, 2025
- Files exist: `eval_results/baselines/v0.3.22/api_call_json_*`
- **Result**: FAILED - AIs still generating wrong syntax at ~same rate

### Actual Results (v0.3.22 Eval Baseline)

**Test Case**: `api_call_json` benchmark (dev models)

**Example (gpt5-mini)**:
```python
# AI generated (STILL WRONG):
url = "https://httpbin.org/post"  # ❌ Bare assignment (PAR015)
headers = {"X-Test-Header": "value123", ...}  # ❌ Python dict syntax
body = {"message": "Hello from AILANG", ...}  # ❌ Python dict syntax
resp = http.post(url, headers=headers, json=body)  # ❌ Python OOP
print(resp.status_code)
```

**Parser Errors (CORRECT)**: PAR015 with full suggestions shown
**Self-Repair**: Attempted but FAILED (repair_ok=false)

**Metrics**:
- **Target**: 75% failure → <25% failure (50pp improvement)
- **Actual**: ~100% failure (no improvement observed)
- **compile_ok**: false (all tested models)
- **repair_ok**: false (self-repair attempts failed)

### Why It Didn't Work

**Root Cause Analysis**:
1. **Enhanced errors ARE working**: Parser detects patterns and shows suggestions
2. **Prompt reorganization worked**: HTTP example is at line 107 (early!)
3. **BUT AIs ignore both**: Still generate Python/JS syntax on first attempt
4. **AND self-repair fails**: Even with error messages, AIs can't fix the code

**Hypothesis**:
- AIs default to Python/JS patterns regardless of prompt position
- Error messages alone insufficient for self-repair
- May require:
  - More prominent anti-pattern warnings (red boxes, ALL CAPS)
  - Simpler examples (less realistic HTTP code)
  - Teaching prompts restructured around "DON'T" examples first
  - Better self-repair prompts (not just parser errors)

### Conclusion

**Implementation Status**: ✅ COMPLETE
- All code implemented and tested
- All files modified as planned
- Prompt reorganized successfully

**Success Metrics**: ❌ NOT ACHIEVED
- api_call_json still failing at ~100% rate
- No measurable improvement from baseline
- Self-repair still failing

**Recommendation**:
- Mark as "implemented but ineffective"
- Consider alternative approaches:
  - More aggressive prompt engineering (anti-patterns FIRST)
  - Better self-repair loop (provide correct example in repair prompt)
  - Simpler teaching examples (less realistic, more syntax-focused)
  - Eval harness improvements (better repair prompts)

**Files**:
- Parser errors: v0.3.17 (M-COMPILE-ERROR)
- Prompt: `prompts/v0.3.22.md`
- Eval baseline: `eval_results/baselines/v0.3.22/`
- Commit: 1a8d322 (Oct 27, 2025)

---

*Sprint completed: October 2025*
*Success metrics: Not achieved*
*Status: Implementation complete, efficacy unproven*

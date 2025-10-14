ok good, lets up# AI Usability Improvements for v0.3.6

**Status**: Planned
**Priority**: P0 (Critical for AI adoption)
**Created**: 2025-10-14
**Based on**: M-EVAL benchmark analysis (v0.3.5-8-g2e48915)

## Problem Statement

Analysis of 57 benchmark runs across 3 models (Claude, GPT-5, Gemini) with v0.3.5 prompt revealed **33.3% success rate** (20/60 passing). Error analysis shows clear patterns where AI models consistently generate invalid AILANG code despite explicit prompt instructions.

### Error Breakdown (37 failures analyzed)

| Pattern | Count | % of Failures | Severity |
|---------|-------|---------------|----------|
| **Wrong Language** (non-AILANG code) | 9 | 24% | **CRITICAL** |
| **Imperative Syntax** (loop/break) | 8 | 22% | **CRITICAL** |
| **Missing Typeclass Imports** | 4 | 11% | High |
| **Record Update Workaround** | 2 | 5% | Medium |
| **Logic Errors** | 2 | 5% | Low (not language issue) |
| **Other Parse/Type Errors** | 12 | 32% | Varied |

**Key Insight**: **46% of failures** (17/37) are due to AI models ignoring core AILANG syntax rules despite clear prompt warnings.

## Evidence: Actual AI-Generated Code

### Pattern 1: Wrong Language (9 failures)

**Benchmark**: `numeric_modulo`
**Model**: Claude Sonnet 4.5

```
PRINT 5 % 3
```

**Error**: `undefined variable: PRINT`

**Analysis**: AI generated a single-line statement with no module declaration, no function, completely wrong syntax. This is **not AILANG code at all**.

---

### Pattern 2: Imperative Syntax (8 failures)

**Benchmark**: `pipeline`
**Model**: Claude Sonnet 4.5

```python
loop {
  input_line = read();
  if (input_line == "") {
    break;
  }
  num = int(input_line);
  doubled = num * 2;
  print(doubled);
}
```

**Error**: `expected ; or }, got = PAR_NO_PREFIX_PARSE`

**Analysis**: AI generated **imperative Python-style code** with:
- `loop` construct (doesn't exist)
- `break` statement (doesn't exist)
- Assignment statements `=` (not valid in expressions)
- Mutable variable reassignment

Despite prompt explicitly stating: **"NO `for`, `while`, `var`, `const`, `let mut`, or any imperative constructs"**

---

### Pattern 3: Missing Typeclass Imports (4 failures)

**Benchmark**: `fizzbuzz`
**Model**: Gemini 2.5 Pro

```ailang
module benchmark/solution

import std/io (println)

export func fizzbuzz(n: int) -> string {
  if n <= 0  // ERROR: No instance for Ord
  then "0"
  else ...
}
```

**Error**: `No instance for Ord[α9] in scope. Import std/prelude or define instance`

**Analysis**: AI used `<=` operator but forgot to `import std/prelude`. Error message explicitly tells them to import, but they don't learn for next attempt.

---

### Pattern 4: Record Update Syntax Missing (2 failures)

**Benchmark**: `record_update`
**Model**: Claude Sonnet 4.5

```ailang
export func main() -> () ! {IO} {
  let person = {name: "Alice", age: 30, city: "NYC"};
  let older = {name: person.name, age: person.age + 1, city: person.city};
  let moved = {name: person.name, age: person.age, city: "SF"};  // BUG!

  println(moved.name ++ ", " ++ show(moved.age) ++ ", " ++ moved.city)
}
```

**Output**: `"Alice, 30, SF"` (expected `"Alice, 31, SF"`)

**Analysis**: AI wanted to write `{older | city: "SF"}` but AILANG doesn't support record update syntax. Forced to manually copy all fields, and **forgot to copy the updated age field**. This is a **logic error caused by missing language feature**.

## Root Cause Analysis

### Why AI Models Fail

1. **Cognitive Load**: Remembering all syntax rules while also solving the algorithmic problem
2. **Conflicting Training Data**: Models trained on Python/Rust/JS override AILANG-specific rules
3. **Weak Prompt Adherence**: Current prompt warnings are insufficient
4. **Missing Language Features**: Forcing verbose workarounds that cause errors

### What Works vs What Doesn't

**✅ High Success Benchmarks** (80-100% success):
- `adt_option` - ADT syntax is clear and familiar
- `records_person` - Simple record syntax
- `fizzbuzz` - When they remember to import std/prelude
- `recursion_fibonacci` - Recursive functions work well

**❌ Low Success Benchmarks** (0-40% success):
- `pipeline` - IO + stdin loops → imperative code
- `json_parse` - String processing → wrong language
- `list_comprehension` - No list comprehension syntax → imperative loops
- `record_update` - Missing syntax → manual copy errors

## Proposed Solutions

### Priority 1: Prompt Improvements (v0.3.6-prompt)

**Goal**: Reduce "wrong language" and "imperative syntax" failures by 50%

#### Changes to `prompts/v0.3.5.md`

**1. Add Strong Anti-Pattern Section at Top**

```markdown
## ⚠️ CRITICAL: What AILANG is NOT

**DO NOT generate code like this:**

❌ **WRONG - Imperative style:**
```
loop {
  x = read();
  if (x == "") { break; }
}
```

❌ **WRONG - Single-line statements:**
```
PRINT 5 % 3
```

❌ **WRONG - Python/JavaScript syntax:**
```
for i in range(10):
    print(i)
```

**✅ CORRECT - Functional AILANG:**
```ailang
module benchmark/solution

import std/io (println)

export func processLoop(acc: int) -> () ! {IO} {
  if acc > 10
  then ()
  else {
    println(show(acc));
    processLoop(acc + 1)
  }
}
```
```

**2. Emphasize Module Requirement**

```markdown
**EVERY AILANG program MUST:**
1. Start with `module path/name`
2. Import capabilities: `import std/io (println)`
3. Define `export func main() -> () ! {IO}`
```

**3. Add Typeclass Import Checklist**

```markdown
**Before using operators, check imports:**
- `<`, `>`, `<=`, `>=` → `import std/prelude` (Ord typeclass)
- `==`, `!=` → `import std/prelude` (Eq typeclass)
- `show` → builtin, no import needed ✅
```

### Priority 2: Language Features (v0.4.0)

**Goal**: Reduce workaround-induced errors

#### Feature 1: Record Update Syntax

**Syntax**: `{record | field: newValue, field2: newValue2}`

**Example**:
```ailang
let person = {name: "Alice", age: 30};
let older = {person | age: 31};  // Only update age, keep name
```

**Benefit**: Eliminates manual field copying that causes logic errors

**Implementation**: Desugar to record construction in elaboration phase

---

#### Feature 2: Auto-Import std/prelude

**Change**: Make `Ord` and `Eq` instances available by default (no import needed)

**Rationale**:
- Every language needs comparison operators
- Requiring imports for `<`, `==` is confusing to AI models
- Reduces cognitive load

**Implementation**: Add prelude types to initial environment in type checker

---

#### Feature 3: List Comprehension Syntax (Optional)

**Syntax**: `[expr | x <- list, guard]`

**Example**:
```ailang
let evens = [x * 2 | x <- [1, 2, 3, 4], x % 2 == 0]
// Desugars to: map(\x. x * 2, filter(\x. x % 2 == 0, [1, 2, 3, 4]))
```

**Benefit**: Prevents AI from generating imperative loops

**Priority**: Low (can use `map`/`filter` for now)

### Priority 3: Better Error Messages (v0.3.6)

**Goal**: Help AI models self-correct during repair phase

#### Current vs Improved

**Current**:
```
Error: expected ; or }, got = PAR_NO_PREFIX_PARSE
```

**Improved**:
```
Error: Assignment statements are not allowed in AILANG.
  at benchmark/solution.ail:2:14: 'input_line = read()'

  AILANG is a pure functional language. Use 'let' bindings instead:

  Hint: let input_line = read() in ...
```

## Success Metrics

**Prompt Improvements (v0.3.6-prompt):**
- "Wrong language" failures: 9 → <5 (44% reduction)
- "Imperative syntax" failures: 8 → <4 (50% reduction)
- Overall success rate: 33% → 45%+ (12% improvement)

**Language Features (v0.4.0):**
- Record update failures: 2 → 0 (100% fix)
- Typeclass import errors: 4 → 0 (100% fix)
- Overall success rate: 45% → 60%+ (15% improvement)

## Implementation Plan

### Phase 1: Auto-Import std/prelude ✅ COMPLETE

**Status**: Implemented and tested (2025-10-14)

**Changes Made**:
1. Modified `internal/types/typechecker_core.go`:
   - `NewCoreTypeChecker()` now calls `LoadBuiltinInstances()` by default
   - Added `AILANG_NO_PRELUDE=1` env var to disable for tests
2. Created `internal/types/auto_import_test.go` with unit tests
3. **Bug Fix**: Fixed `isGround()` to recognize `TVar2` as non-ground
   - **Root Cause**: `isGround()` only checked for `*TVar` (old type system), not `*TVar2` (new type system)
   - **Symptom**: Type variables like `α4` were treated as ground types, causing premature instance lookup before defaulting
   - **Impact**: `let x = 10; if x < 10` would fail with "No instance for Ord[α4]" even though instances were loaded
   - **Fix**: Added `case *TVar2: return false` to `isGround()` function

**Test Results**:
- ✅ Simple case: `if 5 < 10` works (integer literals default immediately)
- ✅ Variable case: `let x = 10; if x < 10` works (type variables defer to defaulting)
- ✅ All existing tests pass
- ✅ New test `TestAutoImportWithVariables` validates the fix

**Metrics**:
- Lines changed: ~15 (core fix was 2 lines)
- Tests added: 3 test functions
- Files modified: 2 (typechecker_core.go, auto_import_test.go)

### Phase 2: Error Taxonomy + Self-Repair (2 days) - PENDING

1. Create `internal/eval_harness/errors.go` with pattern detection
2. Create `internal/eval_harness/repair.go` with retry logic
3. Add repair metrics tracking

### Phase 3: Record Update Syntax (3 days) - PENDING

1. Add AST node for `RecordUpdate`
2. Parser: `{expr | field: value}`
3. Elaborate: Desugar to full record construction
4. Type checker: Verify field existence
5. Tests + examples

### Phase 3: Auto-Import Prelude (1 day)

1. Modify type checker initial environment
2. Add Ord/Eq instances automatically
3. Update docs to note auto-import
4. Tests

### Phase 4: Error Messages (2 days)

1. Add error context to parser errors
2. Include hints for common mistakes
3. Add "did you mean?" suggestions
4. Tests

**Total Estimate**: 8 days for v0.3.6 + v0.4.0 features

## Security Considerations

- Record update syntax: No security implications (pure syntactic sugar)
- Auto-import prelude: Verify no capability leaks
- Error messages: Don't expose internal paths/data

## Alternatives Considered

### Alternative 1: Keep Current Approach

**Pros**: No implementation work
**Cons**: AI success rate stays at 33%, adoption blocked

### Alternative 2: Multi-Shot Repair

**Pros**: Might fix some errors
**Cons**: Expensive (more tokens), doesn't address root cause

### Alternative 3: Simpler Syntax

**Pros**: Easier for AI
**Cons**: Lose functional programming benefits, not our goal

## Open Questions

1. Should we add list comprehension syntax now, or wait for v0.5.0?
2. Should auto-import include other common modules (std/io)?
3. How to measure prompt effectiveness across different model versions?

## References

- M-EVAL Results: `eval_results/baselines/v0.3.5-8-g2e48915/`
- Current Prompt: `prompts/v0.3.5.md`
- Benchmark Specs: `benchmarks/*.yml`
- Error Analysis: This document's evidence section

---

**Next Steps**:
1. Review this design doc with team
2. Prioritize: prompt improvements (quick win) vs language features (longer term)
3. Run A/B test with improved prompt
4. Implement record update syntax for v0.4.0

---

## Iteration 1 Results (v0.3.6 Prompt Test - FAILED)

**Date**: 2025-10-14
**Models Tested**: Claude Sonnet 4.5, Gemini 2.5 Flash
**Test Duration**: 45 seconds

### Results

| Prompt Version | Claude Success | Gemini Success | Average |
|----------------|----------------|----------------|---------|
| v0.3.5 (baseline) | 7/19 (36.8%) | 5/19 (26.3%) | 31.6% |
| v0.3.6 (with anti-patterns) | 6/19 (31.6%) | 5/19 (26.3%) | 28.9% |

**Outcome**: ❌ **v0.3.6 performed WORSE (-5.2% for Claude, -2.7% overall)**

### What Went Wrong

**1. Import Syntax Bug** - Broke previously-working code
```ailang
# Prompt suggested:
import std/prelude  // Wrong! Namespace import not supported

# AI generated:
import std/prelude  # ← Compile error!

# Should have been:
import std/prelude (Ord, Eq)  # ← Correct
```

**Result**: `fizzbuzz` benchmark failed despite correct logic

**2. Prompt Too Long** - Diluted key instructions
- v0.3.5: ~400 lines
- v0.3.6: ~450 lines (+12.5%)
- More text → more confusion, not less

**3. Anti-Patterns Didn't Work** - AI still generated wrong code
```
# AI still generated this despite explicit warnings:
READ A 5
MOD C A B
PRINT C
```

**Analysis**: Models ignore warnings in favor of training data patterns

### Key Insights

1. **Prompt improvements alone are insufficient** for this problem
2. **Explicit warnings don't work** - AI reverts to training data
3. **Import syntax is too complex** - even with checklist, AI gets it wrong
4. **Longer prompts hurt performance** - information overload

### Revised Strategy

**❌ Don't**: Try to teach AI correct syntax through prompts
**✅ Do**: Change the language to eliminate wrong choices

**Priority 1: Auto-Import std/prelude (v0.3.6 language change)**
- Makes import errors impossible
- Reduces prompt complexity
- Expected impact: +10-15% success rate

**Priority 2: Record Update Syntax (v0.4.0)**
- Eliminates manual field copying
- Expected impact: +5% success rate

**Priority 3: Better Error Messages (v0.4.0)**
- Help AI self-correct in repair phase
- Expected impact: +5-10% with self-repair enabled

### Next Steps

1. ❌ ~~Iterate on prompt~~ - demonstrated ineffective
2. ✅ Implement auto-import std/prelude
3. ✅ Run new baseline with v0.3.6 language + simpler prompt
4. ✅ Measure improvement


## Iteration 2 Results (v0.3.6 Fixed Import Syntax - STILL FAILED)

**Date**: 2025-10-14
**Fix Applied**: Corrected import syntax to `import std/prelude (Ord, Eq)`
**Test Duration**: 44 seconds

### Results

| Prompt Version | Claude Success | Gemini Success | Average |
|----------------|----------------|----------------|---------|
| v0.3.5 (baseline) | 7/19 (36.8%) | 5/19 (26.3%) | 31.6% |
| v0.3.6-test1 (bad import) | 6/19 (31.6%) | 5/19 (26.3%) | 28.9% |
| v0.3.6-test2 (fixed import) | 5/19 (26.3%) | 5/19 (26.3%) | 26.3% |

**Outcome**: ❌ **Even WORSE (-10.5% for Claude, -5.3% overall)**

### Analysis

**Progressive degradation with more instructions:**
1. v0.3.5: Shortest prompt → Best performance (36.8% Claude)
2. v0.3.6-test1: Added anti-patterns + wrong import → Worse (31.6%)
3. v0.3.6-test2: Fixed import syntax → Even worse (26.3%)

**Hypothesis**: Longer prompts cause information overload
- v0.3.5: ~400 lines
- v0.3.6: ~450 lines (+12.5%)
- More examples → More confusion, not less

### Final Conclusion

**Prompt engineering is COUNTER-PRODUCTIVE for this problem.**

Three iterations proved:
1. ❌ Anti-pattern warnings → ignored by AI
2. ❌ Import checklists → cause more errors  
3. ❌ Longer, more detailed prompts → worse performance

**The fundamental issue**: AI models default to training data patterns regardless of explicit instructions. Telling them "don't do X" doesn't work when their training heavily weights X.

### Recommendation: STOP Prompt Iteration

**Instead, prioritize language changes:**

1. **Auto-import std/prelude** (P0 - Critical)
   - Eliminates 11% of failures (4/37 typeclass errors)
   - Removes cognitive load
   - Implementation: 1 day

2. **Record update syntax** (P1 - High)
   - Eliminates 5% of failures (2/37 record copy errors)
   - Cleaner syntax reduces mistakes
   - Implementation: 3 days

3. **Better error messages** (P2 - Medium)
   - Help AI self-correct in repair phase
   - Target: +10% success rate with self-repair
   - Implementation: 2 days

**Expected cumulative impact**: 31.6% → 50%+ success rate


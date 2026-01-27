# M-DX24: Developer Experience Improvements - BigQuery Connector Feedback

**Status:** Planned
**Target:** v0.7.0
**Priority:** P1 (High - blocks real-world usage)
**Estimated:** 2 weeks (documentation, fixes, error handling)
**Dependencies:** None

**Created:** 2026-01-27
**Last Modified:** 2026-01-27

---

## Problem Statement

A developer built a real-world BigQuery connector for Google Cloud Platform authentication and querying, demonstrating significant AILANG capabilities. However, the experience revealed 6 critical DX issues that make the language harder to use than necessary:

### Current Pain Points

**1. Reserved Words Not Documented**
- `exists` is a reserved keyword but not mentioned anywhere in docs
- Cryptic parse error: `PAR_UNEXPECTED_TOKEN: expected next token to be IDENT, got exists instead`
- Forces code refactoring to discover the restriction
- **Impact:** Wasted developer time, frustration with error messages

**2. Option Pattern Matching Broken at Runtime** ⚠️ CRITICAL
- Code type-checks but fails at runtime: "no pattern matched in match expression"
- Workaround: Use `isSome`/`getOrElse` instead of pattern matching
- `match getEnv("HOME") { Some(h) => h, None => "/tmp" }` fails despite valid syntax
- **Impact:** Language feature doesn't work; fundamentally confusing

**3. Stdlib Version Mismatch Warnings** ⚠️ CRITICAL
- Every run shows: `Warning: stdlib version mismatch: expected dev, found v0.7.0`
- No guidance on fixing it
- May be causing Option pattern matching issues
- **Impact:** Unclear whether warnings indicate real problems

**4. Module Import Transitivity Unclear**
- Module A imports B which uses `std/fs`
- Module A fails: `failed to resolve global std/fs.fileExists: module std/fs not imported`
- Implicit vs. explicit imports not documented
- **Impact:** Non-obvious error; requires trial/error to understand design

**5. Multi-Statement If-Then-Else Blocks Unsupported**
- Cannot use `let` bindings inside if branches
- Workaround: Extract into separate functions (verbose)
- **Impact:** Code readability; increased boilerplate

**6. Record Construction in Result Types Fragile**
- Direct `Ok({ field: value })` causes type unification errors
- Workaround: Create helper function (adds indirection)
- **Impact:** Simple operations require extra ceremony

### Metrics

**Developer Experience Impact:**
- ~60% of time spent fighting runtime errors that passed type-checking
- ~30% actual implementation
- ~10% documentation lookup
- **Frustration level:** High (type-checking didn't catch real issues)

**Real-World Test Case:**
- Files: `gcp_auth.ail`, `bigquery.ail`, `bigquery_demo.ail`
- Total LOC: ~400 lines of production-quality AILANG
- Issues encountered: 6 distinct DX problems
- Recovery path: Manual workarounds, code duplication

---

## Goals

### Primary Goal
Improve developer ergonomics for real-world AILANG programs by fixing runtime failures, improving error messages, and documenting implicit behaviors.

### Success Metrics
1. ✅ All reserved keywords documented and discoverable
2. ✅ Option pattern matching works correctly at runtime
3. ✅ Stdlib version mismatch resolved or clearly explained
4. ✅ Module import rules documented with clear examples
5. ✅ Error messages include file location and expected types
6. ✅ Record construction in Result types "just works"

---

## Root Cause Analysis

### Issue 1: Reserved Keywords Not Documented

**Root Cause:** Keywords defined in `internal/lexer/token.go` but no user-facing documentation.

**Current State:**
- Keywords map: 43 reserved words (func, let, if, match, type, exists, forall, etc.)
- Contextual keywords: test, tests, properties (can be used as identifiers in some contexts)
- Error message: Generic `PAR_UNEXPECTED_TOKEN` without explanation

**Gap:** No doc page listing reserved words. `ailang docs` doesn't include keyword reference.

**Solution:**
1. Create `docs/reference/reserved-keywords.md` with complete list
2. Improve parser error message: "Keyword 'exists' is reserved - cannot use as identifier"
3. Add reserved word check in error suggestions
4. Mention common pitfalls in teaching prompt

---

### Issue 2: Option Pattern Matching Broken at Runtime ⚠️ CRITICAL

**Root Cause:** Pattern matching elaboration doesn't correctly generate code for ADT constructors.

**Evidence:**
- Code type-checks (ADT pattern is valid)
- Runtime fails with "no pattern matched"
- Workaround works: `if isSome(opt) then ...`

**Likely Cause:**
- Elaborator generates incorrect Core for `match` expressions with ADT patterns
- Decision tree generation fails for `Some(h)` pattern
- Fallback path is not hit (otherwise would use default/error pattern)

**Impact on real code:**
```ailang
-- Type-checks, FAILS at runtime:
match getEnv("HOME") {
  Some(h) => h,
  None => "/tmp"
}

-- Works (workaround):
if isSome(opt) then getOrElse(opt, "") else "/tmp"
```

**Investigation Required:**
1. Check `internal/elaborate/pattern.go` - Pattern elaboration for ADTs
2. Check `internal/dtree/dtree.go` - Decision tree generation
3. Add debug output for pattern matching failures
4. Test with simple ADT patterns

**Fix Strategy:**
- Audit pattern matching elaboration
- Add test case: `match Some(1) { Some(x) => x, None => 0 }`
- Ensure decision tree covers all cases
- Add "no pattern matched" error with source location

---

### Issue 3: Stdlib Version Mismatch ⚠️ CRITICAL

**Root Cause:** Version checking logic doesn't align with development mode.

**Current Behavior:**
```
Warning: stdlib version mismatch: expected dev, found v0.7.0
```

**Root Cause Hypothesis:**
- AILANG expects stdlib version "dev" (development build)
- Installed stdlib is v0.7.0 (released)
- Version check is too strict during development
- May break Option pattern matching if type definitions don't match

**Investigation Required:**
1. Check `internal/loader/loader.go` or manifest checking code
2. Find version check logic
3. Determine if version mismatch actually causes runtime issues

**Fix Strategy:**
1. **Option A:** Relax dev version check (allow semver mismatch)
2. **Option B:** Add `--skip-version-check` flag for dev/testing
3. **Option C:** Clearly document version expectations
4. **Preferred:** Investigate if version mismatch causes Option pattern failure, fix root cause

---

### Issue 4: Module Import Transitivity Unclear

**Root Cause:** No documentation of implicit vs. explicit imports; error message unhelpful.

**Current Behavior:**
```
Module A imports B
Module B imports std/fs, uses fs.fileExists
Module A fails: failed to resolve global std/fs.fileExists: module std/fs not imported
```

**Design Decision:** AILANG requires explicit imports (no transitive visibility).
- Similar to Python's "explicit is better than implicit"
- Prevents surprises when module B changes
- But not documented

**Fix Strategy:**
1. Document in `docs/guides/modules.md`: "Imports are not transitive - you must explicitly import all modules you use"
2. Improve error message: "Module 'std/fs' is not imported. Either import it in module A or access via module B."
3. Add example showing correct usage
4. Update teaching prompt with module example

---

### Issue 5: Multi-Statement If-Then-Else Blocks

**Root Cause:** Parser or elaborator requires single expression, doesn't support blocks in conditions.

**Current Limitation:**
```ailang
-- FAILS:
if condition then {
  let x = foo();
  doSomething(x)
} else ...

-- WORKS:
if condition then helperFunc() else ...
```

**Root Cause:** Block expressions may not be fully supported in if-then branches, or `let` binding isn't recognized inside blocks.

**Investigation Required:**
1. Check if `{ let x = ...; expr }` works as standalone block
2. Check if blocks work in if-then position
3. Test: `if true then { let x = 1; x + 2 } else 0`

**Fix Strategy:**
1. Enable block expressions in if-then-else branches
2. Add test case covering multiple statements in both branches
3. Document pattern in teaching prompt

---

### Issue 6: Record Construction in Result Types Fragile

**Root Cause:** Type unification doesn't handle record inference in ADT constructor context.

**Current Workaround:**
```ailang
-- FAILS with type error:
let result: Result[ADCCredentials, string] = Ok({ clientId: "...", ... });

-- WORKS:
let creds: ADCCredentials = { clientId: "...", ... };
Ok(creds)
```

**Root Cause Hypothesis:**
- Type inference for record literals doesn't propagate from `Ok()`'s type parameter
- Needs explicit intermediate binding to help type checker

**Fix Strategy:**
1. Improve record literal type inference in ADT constructor contexts
2. Add error message suggesting intermediate binding workaround
3. Test with nested type parameters

---

## Solution Design

### Overview

Six focused improvements targeting documentation, error messages, and runtime fixes:

1. **Reserved Keywords Documentation** - Add public reference doc + improve error messages
2. **Option Pattern Matching Fix** - Debug and fix elaboration or evaluation
3. **Stdlib Version Handling** - Either relax checking or investigate root cause
4. **Module Import Documentation** - Document design decision + improve error messages
5. **If-Then-Else Blocks** - Enable or document limitation
6. **Record Inference** - Improve type inference or add better error guidance

### Implementation Plan

#### Phase 1: Documentation & Error Messages (3-4 days)

**1.1: Create Reserved Keywords Reference**
- File: `docs/reference/reserved-keywords.md`
- Content:
  - Complete list of 43 keywords
  - Which are contextual (test, tests, properties)
  - Why each exists
  - Common mistakes and workarounds
- LOC: ~150 lines
- Add to `docs/SUMMARY.md` under "Reference"

**1.2: Improve Parser Error Messages**
- File: `internal/errors/errors.go`
- Changes:
  - Detect when identifier is reserved keyword
  - Provide helpful message: "Cannot use reserved keyword 'X' as identifier"
  - Suggest alternative naming
- LOC: ~50 lines of new code
- Tests: Add case for each contextual keyword

**1.3: Update Teaching Prompt**
- File: `prompts/v0.7.0.md`
- Add section on reserved keywords with examples
- Document module import rules
- LOC: ~100 lines

**1.4: Document Module Import Rules**
- File: `docs/guides/modules.md`
- New section: "Import Transitivity"
- Example showing when/why explicit imports required
- Error message reference
- LOC: ~80 lines

---

#### Phase 2: Runtime Fixes (5-7 days)

**2.1: Fix Option Pattern Matching** ⚠️ CRITICAL
- Investigate: `internal/elaborate/pattern.go`
- Likely fixes:
  - Correct decision tree generation for ADT patterns
  - Ensure `Some(x)` pattern matches `Some` constructor
  - Add pattern matching test suite
- Files to modify:
  - `internal/elaborate/pattern.go` (~50-100 lines added)
  - `internal/dtree/dtree.go` (debug + fix)
  - Add test: `examples/pattern_matching_adt.ail`
- Tests: Comprehensive ADT pattern test suite
- LOC: ~150-200 lines

**2.2: Debug Stdlib Version Mismatch**
- Investigate: Find version check code
- Likely in: `internal/loader/`, `internal/manifest/`
- Decision:
  - **Option A:** Relax check for development
  - **Option B:** Add `--skip-version-check` flag
  - **Option C:** Fix root cause affecting Option patterns
- Files: Likely 1-2 files, ~30-50 lines modified
- Tests: Version mismatch test case

---

#### Phase 3: Language Features (5-7 days)

**3.1: Enable If-Then-Else Block Expressions**
- Files: `internal/parser/parser.go`, `internal/elaborate/elaborate.go`
- Changes:
  - Allow block expressions in if-then-else branches
  - Ensure type checking propagates correctly
  - Test: `if true then { let x = 1; x + 2 } else 0` → `2`
- LOC: ~50 lines
- Tests: Block expression test suite

**3.2: Improve Record Inference in ADT Contexts**
- Files: `internal/types/unify.go`, `internal/elaborate/elaborate.go`
- Changes:
  - Propagate type expectations to record literals
  - Better error messages when inference fails
  - Test: `let _: Result[Rec, str] = Ok({ a: 1 })`
- LOC: ~100 lines
- Tests: Record type inference test suite

---

### Files to Modify/Create

**Documentation (New):**
- `docs/reference/reserved-keywords.md` (~150 lines)
- `docs/guides/modules.md` (add ~80 lines)

**Implementation (Modified):**
- `internal/errors/errors.go` (~50 lines)
- `internal/elaborate/pattern.go` (~150-200 lines)
- `internal/parser/parser.go` (~50 lines)
- `internal/types/unify.go` (~100 lines)

**Tests (New):**
- `internal/elaborate/pattern_test.go` (~100 lines)
- `examples/pattern_matching_adt.ail` (~20 lines)
- `examples/if_then_else_blocks.ail` (~20 lines)

**Teaching Prompt:**
- `prompts/v0.7.0.md` (~100 lines added)

---

## Examples

### Reserved Keywords Documentation

```markdown
## Reserved Keywords

AILANG reserves 43 keywords. You cannot use these as variable or function names.

### Complete List

**Control Flow:** if, then, else, match, with, select, timeout
**Definitions:** func, let, letrec, type, class, instance, import, export, extern, pure
**Modules:** module, import, export, as
**Types:** forall, exists
**Testing:** test, tests, property, properties, assert
**Concurrency:** spawn, parallel, channel, send, recv
**Verification:** requires, ensures, invariant (M-VERIFY)
**Boolean:** true, false, and, or, not

### Contextual Keywords

The following can sometimes be used as identifiers, but we recommend avoiding them:
- `test`, `tests`, `property`, `properties` - Reserved after `func` declarations

### Common Mistake

```ailang
-- ❌ WRONG - 'exists' is reserved
let exists = fileExists(path);

-- ✅ CORRECT - Use alternative name
let found = fileExists(path);
```
```

### Module Import Documentation

```markdown
## Module Imports Are Not Transitive

Unlike some languages, AILANG requires **explicit imports** for every module you use.

### Why?

When module A imports module B, A does **not** automatically get B's imports.

This prevents:
- Hidden dependencies (you see exactly what each module uses)
- Breakage when module B changes its imports
- Confusion about where functions come from

### Example

**WRONG - Implicit import:**
```ailang
module myapp
import ecommerce/services/bigquery  -- Uses std/fs inside

-- ERROR: failed to resolve global std/fs.fileExists
let path = getCredentialsPath()
```

**CORRECT - Explicit import:**
```ailang
module myapp
import std/fs (fileExists)  -- Explicitly import what you use
import ecommerce/services/bigquery

let path = getCredentialsPath()  -- Now works
```

The rule: **Import everything you directly use in your module.**
```

### Option Pattern Matching Example

**After Fix - This Will Work:**
```ailang
import std/option (Some, None)

-- Pattern matching will work correctly:
func getHomeOrDefault() -> string ! {FS} {
  match getEnv("HOME") {
    Some(h) => h,
    None => "/tmp"
  }
}
```

---

## Success Criteria

- [ ] Reserved keywords documented (`docs/reference/reserved-keywords.md` created)
- [ ] Parser error message mentions reserved keywords
- [ ] Module import rules documented (`docs/guides/modules.md` updated)
- [ ] Option pattern matching test passes
- [ ] If-then-else with blocks works in tests
- [ ] Record inference in ADT contexts improved
- [ ] Teaching prompt updated with examples
- [ ] BigQuery demo code works without workarounds
- [ ] All tests passing
- [ ] No regression in existing functionality

---

## Timeline

### Week 1: Documentation & Error Messages
- **Mon-Tue:** Create reserved keywords reference, update teaching prompt (~6h)
- **Wed:** Improve error messages for keyword detection (~4h)
- **Thu:** Document module import rules, add examples (~4h)
- **Fri:** Testing and iteration (~4h)

### Week 2: Runtime Fixes & Language Features
- **Mon-Tue:** Debug and fix Option pattern matching (~8h)
- **Wed:** Debug stdlib version mismatch (~4h)
- **Thu:** Enable if-then-else blocks (~6h)
- **Fri:** Improve record type inference, testing (~6h)

---

## Related Documents

- [M-DX1: Developer Experience](../implemented/v0_3_10/m-dx1-developer-experience.md) - Builtin registry improvements
- [M-DX4: CoreTypeInfo Validation](../implemented/v0_3_10/M-DX4-CORETYPEINFO-VALIDATION.md) - Type system improvements
- [Teaching Prompt System](../implemented/v0_5_0/prompt-system.md) - Documentation of prompt versioning
- [Parser Developer Experience](../planned/v0_3_15/m-dx9-parser-developer-experience.md) - Parser conventions

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| **A1: Determinism** | +1 | Better error messages don't affect determinism; fixes improve reliability |
| **A2: Replayability** | +1 | Clear error messages aid debugging and reproduction |
| **A3: Effect Legibility** | 0 | No effect-related changes |
| **A4: Explicit Authority** | 0 | Module import improvements clarify explicit import design |
| **A5: Bounded Verification** | +1 | Improved error messages aid local verification |
| **A6: Safe Concurrency** | 0 | No concurrency changes |
| **A7: Machines First** | +1 | Better structured errors are machine-parseable |
| **A8: Minimal Syntax** | 0 | No syntax additions |
| **A9: Cost Visibility** | 0 | No resource-related changes |
| **A10: Composability** | +1 | Pattern matching fixes improve function composition |
| **A11: Structured Failure** | +1 | Better error messages with location info |
| **A12: System Boundary** | +1 | Explicit imports strengthen module boundaries |

**Net Score: +7** ✅ Strong alignment with core axioms

---

## Known Issues & Limitations

### Not In Scope

The following issues are real but deferred to future releases:
- **Nullary function calls** (M-DX10) - Cannot call zero-arg functions from AILANG
- **Better performance** - No optimization work in this DX pass
- **Syntax improvements** - Minimal changes per Axiom A8

### Risks

1. **Option pattern matching fix may introduce regressions** - Comprehensive test suite required
2. **Module import changes may affect existing code** - Only documentation, no breaking changes
3. **Type inference improvements may have subtle effects** - Test edge cases

---

## Success Stories from Real-World Usage

The BigQuery connector demonstrates AILANG's strengths:
- ✅ Type-checking caught many errors at compile-time
- ✅ Effect system clearly showed required capabilities
- ✅ JSON handling comprehensive (`std/json` has all needed functions)
- ✅ Pattern matching and recursion work well
- ✅ Module system enables code organization

Addressing these 6 DX issues will let developers focus on the language's strengths rather than working around rough edges.

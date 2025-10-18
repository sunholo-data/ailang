# M-P5: Pattern Matching in Function Bodies - Sprint Plan

**Milestone**: M-P5 (Section D in v0.1.0 Roadmap)
**Date**: October 1, 2025
**Duration**: 1-2 days
**Priority**: CRITICAL - Blocks stdlib implementation
**Status**: 📋 Ready to start

---

## Executive Summary

Fix parser bug preventing pattern matching inside function bodies. Pattern matching currently works at module top-level but fails inside `export func` definitions. This blocks all stdlib implementation in AILANG since every stdlib function uses pattern matching.

**Impact**: Unblocks ~360 LOC of stdlib code already written and ready to deploy.

---

## Problem Statement

### What Works ✅

```ailang
module test
type Option[a] = Some(a) | None

-- Top-level match expression: WORKS
match Some(42) {
  Some(n) => n,
  None => 0
}
-- Output: 42 ✅
```

### What Doesn't Work ❌

```ailang
module test
type Option[a] = Some(a) | None

export func getOrElse[a](opt: Option[a], d: a) -> a {
  match opt {  -- ❌ FAILS: "expected =>, got ] instead"
    Some(x) => x,
    None => d
  }
}
```

**Error Messages**:
- `PAR_UNEXPECTED_TOKEN: expected next token to be =>, got ] instead`
- `PAR_NO_PREFIX_PARSE: unexpected token in expression: ]`
- Affects list patterns: `[]`, `[x, ...rest]`
- Affects all pattern types inside function bodies

---

## Sprint Goal

**Enable pattern matching in all contexts with parity between top-level and function body parsing.**

**Definition of Done**:
- All stdlib modules (`std_list.ail`, `std_option.ail`, etc.) parse without errors
- Test suite with 10+ test cases covering all pattern types passes
- Pattern matching works identically in top-level and function body contexts
- Zero parser regressions (existing tests still pass)

---

## Sprint Breakdown

### Day 1: Investigation + Fix (8 hours)

#### Morning Session: Root Cause Analysis (4 hours)

**Hour 1-2: Trace Parsing Flow**

**Tasks**:
1. Add debug logging to `parseMatchExpression()`:
   ```go
   func (p *Parser) parseMatchExpression() ast.Expr {
       log.Printf("DEBUG: parseMatchExpression called")
       log.Printf("  curToken=%v (type=%v)", p.curToken.Literal, p.curToken.Type)
       log.Printf("  peekToken=%v (type=%v)", p.peekToken.Literal, p.peekToken.Type)
       log.Printf("  inFunctionBody=%v", p.inFunctionBody) // If such flag exists
       // ... existing code
   }
   ```

2. Create minimal test cases:
   - `.tmp/test_pattern_toplevel.ail` (working case)
   - `.tmp/test_pattern_function.ail` (failing case)

3. Run both test cases with debug logging enabled:
   ```bash
   ailang run .tmp/test_pattern_toplevel.ail 2>&1 | tee toplevel.log
   ailang run .tmp/test_pattern_function.ail 2>&1 | tee function.log
   diff toplevel.log function.log
   ```

**Expected Output**:
- Identify exact token where parsing diverges
- Determine parser state differences (context flags, lookahead)

**Hour 3-4: Analyze Block Statement Parsing**

**Tasks**:
1. Read `parseFunctionBody()` or `parseBlockStatement()` in [parser.go:512](internal/parser/parser.go)
2. Trace how expressions are parsed inside blocks:
   - Does it call `parseExpression()` or `parseStatement()`?
   - Are prefix parsers registered correctly in block context?
   - Is there a context flag that affects expression parsing?

3. Compare with top-level expression parsing:
   - Check `parseModuleStatement()` or equivalent
   - Identify differences in how `match` is handled

**Hypothesis to Test**:
- Does `parseStatement()` inside blocks fail to recognize `MATCH` as valid prefix?
- Is there a missing case in block statement parsing?
- Are context flags preventing `parseMatchExpression()` from being called?

**Deliverable**: Root cause identified with evidence (debug logs + code analysis)

---

#### Afternoon Session: Implement Fix (4 hours)

**Hour 5-6: Write the Fix**

**Potential Fix Approaches** (choose based on root cause):

**Option A: Fix Expression Context Propagation**
```go
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
    // Ensure match expressions are recognized
    for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
        stmt := p.parseStatement()
        if stmt != nil {
            block.Statements = append(block.Statements, stmt)
        }
        p.nextToken()
    }
    return block
}

func (p *Parser) parseStatement() ast.Statement {
    switch p.curToken.Type {
    case lexer.LET:
        return p.parseLetStatement()
    case lexer.RETURN:
        return p.parseReturnStatement()
    default:
        // Ensure expression statements can contain match expressions
        return p.parseExpressionStatement()
    }
}
```

**Option B: Register Match in Block Context**
```go
func (p *Parser) parseFunctionBody() *ast.BlockExpression {
    // Ensure prefix parsers are available in this context
    // (Likely already done globally, but verify)
    return p.parseBlockExpression()
}
```

**Option C: Fix Pattern Parsing Lookahead**
```go
func (p *Parser) parseMatchExpression() ast.Expr {
    // Check if token lookahead is different in nested contexts
    // Adjust accordingly
}
```

**Tasks**:
1. Implement chosen fix based on root cause
2. Add comments explaining the fix
3. Keep changes minimal (prefer 1-10 line fix over large refactoring)

**Hour 7-8: Verify Fix with Simple Cases**

**Test Cases**:
```bash
# Test 1: Simple constructor pattern
cat > .tmp/test_fix1.ail <<'EOF'
module test
type Option[a] = Some(a) | None

export func test1() -> int {
  match Some(42) { Some(n) => n, None => 0 }
}
EOF
ailang run .tmp/test_fix1.ail

# Test 2: List pattern
cat > .tmp/test_fix2.ail <<'EOF'
module test

export func length(xs: [int]) -> int {
  match xs {
    [] => 0,
    [_, ...rest] => 1 + length(rest)
  }
}
EOF
ailang run .tmp/test_fix2.ail

# Test 3: Nested patterns
cat > .tmp/test_fix3.ail <<'EOF'
module test
type Option[a] = Some(a) | None

export func unwrap(x: Option[Option[int]]) -> int {
  match x {
    Some(Some(n)) => n,
    Some(None) => -1,
    None => 0
  }
}
EOF
ailang run .tmp/test_fix3.ail
```

**Acceptance**:
- All 3 test cases parse without errors
- Simple evaluation works (if parser fix is complete)

**Deliverable**: Parser fix committed with passing manual tests

---

### Day 2: Testing + Integration (6-8 hours)

#### Morning Session: Comprehensive Test Suite (3 hours)

**Hour 1-2: Create Parser Tests**

**File**: `internal/parser/func_pattern_test.go`

**Test Cases** (10+ tests):
```go
package parser

import "testing"

func TestPatternMatchingInFunctions(t *testing.T) {
    tests := []struct {
        name  string
        input string
        shouldParse bool
    }{
        {
            name: "simple constructor pattern in function",
            input: `
                module test
                type Option[a] = Some(a) | None
                export func f(opt: Option[int]) -> int {
                    match opt { Some(n) => n, None => 0 }
                }
            `,
            shouldParse: true,
        },
        {
            name: "empty list pattern in function",
            input: `
                module test
                export func isEmpty(xs: [int]) -> bool {
                    match xs { [] => true, _ => false }
                }
            `,
            shouldParse: true,
        },
        {
            name: "list pattern with spread in function",
            input: `
                module test
                export func length(xs: [int]) -> int {
                    match xs {
                        [] => 0,
                        [_, ...rest] => 1 + length(rest)
                    }
                }
            `,
            shouldParse: true,
        },
        {
            name: "tuple pattern in function",
            input: `
                module test
                export func swap(p: (int, string)) -> (string, int) {
                    match p { (x, y) => (y, x) }
                }
            `,
            shouldParse: true,
        },
        {
            name: "nested patterns in function",
            input: `
                module test
                type Option[a] = Some(a) | None
                export func unwrap(x: Option[Option[int]]) -> int {
                    match x {
                        Some(Some(n)) => n,
                        Some(None) => -1,
                        None => 0
                    }
                }
            `,
            shouldParse: true,
        },
        {
            name: "multiple match arms in function",
            input: `
                module test
                export func classify(n: int) -> string {
                    match n {
                        0 => "zero",
                        1 => "one",
                        2 => "two",
                        _ => "many"
                    }
                }
            `,
            shouldParse: true,
        },
        {
            name: "wildcard pattern in function",
            input: `
                module test
                export func ignore(x: int) -> int {
                    match x { _ => 42 }
                }
            `,
            shouldParse: true,
        },
        {
            name: "variable binding pattern in function",
            input: `
                module test
                export func identity(x: int) -> int {
                    match x { n => n }
                }
            `,
            shouldParse: true,
        },
        {
            name: "pattern in nested function",
            input: `
                module test
                type Option[a] = Some(a) | None
                export func outer() -> int {
                    let inner = func(opt: Option[int]) -> int {
                        match opt { Some(x) => x, None => 0 }
                    }
                    inner(Some(42))
                }
            `,
            shouldParse: true,
        },
        {
            name: "list pattern with multiple elements",
            input: `
                module test
                export func sumFirst2(xs: [int]) -> int {
                    match xs {
                        [x, y, ...rest] => x + y,
                        [x] => x,
                        [] => 0
                    }
                }
            `,
            shouldParse: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            p := New(tt.input, "<test>")
            program := p.ParseProgram()

            if tt.shouldParse && len(p.Errors()) > 0 {
                t.Errorf("Expected to parse but got errors:")
                for _, err := range p.Errors() {
                    t.Errorf("  %v", err)
                }
            }
            if !tt.shouldParse && len(p.Errors()) == 0 {
                t.Errorf("Expected parse errors but got none")
            }

            // Additional validation: check AST structure
            if tt.shouldParse && len(program.File.Declarations) == 0 {
                t.Errorf("Expected declarations but got none")
            }
        })
    }
}

func TestPatternMatchingTopLevelVsFunctionBody(t *testing.T) {
    // Ensure parity between contexts
    tests := []struct {
        name     string
        toplevel string
        inFunc   string
    }{
        {
            name: "constructor pattern parity",
            toplevel: `
                module test
                type Option[a] = Some(a) | None
                match Some(42) { Some(n) => n, None => 0 }
            `,
            inFunc: `
                module test
                type Option[a] = Some(a) | None
                export func f() -> int {
                    match Some(42) { Some(n) => n, None => 0 }
                }
            `,
        },
        {
            name: "list pattern parity",
            toplevel: `
                module test
                match [1, 2, 3] { [] => 0, [x, ...rest] => x }
            `,
            inFunc: `
                module test
                export func f() -> int {
                    match [1, 2, 3] { [] => 0, [x, ...rest] => x }
                }
            `,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Parse top-level
            p1 := New(tt.toplevel, "<test>")
            prog1 := p1.ParseProgram()
            errs1 := p1.Errors()

            // Parse function body
            p2 := New(tt.inFunc, "<test>")
            prog2 := p2.ParseProgram()
            errs2 := p2.Errors()

            // Both should succeed or both should fail
            if (len(errs1) == 0) != (len(errs2) == 0) {
                t.Errorf("Parity violation:")
                t.Errorf("  Top-level errors: %v", errs1)
                t.Errorf("  Function errors: %v", errs2)
            }

            // If both succeed, AST structure should be similar
            if len(errs1) == 0 && len(errs2) == 0 {
                if prog1 == nil || prog2 == nil {
                    t.Errorf("Expected valid programs but got nil")
                }
            }
        })
    }
}
```

**Hour 3: Run Test Suite**

```bash
# Run new tests
go test -v ./internal/parser -run TestPatternMatchingInFunctions
go test -v ./internal/parser -run TestPatternMatchingTopLevelVsFunctionBody

# Run all parser tests (ensure no regressions)
go test -v ./internal/parser

# Check coverage
go test -cover ./internal/parser
```

**Acceptance**:
- All new tests pass
- No existing tests broken
- Parser test coverage increases from 0% to >50%

**Deliverable**: Comprehensive test suite passing

---

#### Afternoon Session: Stdlib Integration + Polish (3-5 hours)

**Hour 1-2: Test Stdlib Modules**

**Tasks**:
1. Verify all stdlib modules parse:
   ```bash
   # Test each module
   ailang check std_list.ail
   ailang check std_option.ail
   ailang check std_result.ail
   ailang check std_string.ail
   ailang check std_io.ail
   ```

2. Create stdlib test file:
   ```bash
   cat > tests/stdlib/list.ail <<'EOF'
   module tests_stdlib_list
   import std_list (map, filter, length)

   -- Test map
   match map(\x. x * 2, [1, 2, 3]) {
     [2, 4, 6] => print("map works"),
     _ => print("map failed")
   }

   -- Test filter
   match filter(\x. x > 2, [1, 2, 3, 4]) {
     [3, 4] => print("filter works"),
     _ => print("filter failed")
   }

   -- Test length
   match length([1, 2, 3]) {
     3 => print("length works"),
     _ => print("length failed")
   }
   EOF

   ailang run tests/stdlib/list.ail
   ```

3. Fix any edge cases discovered during stdlib testing

**Hour 3: Create Examples**

**File**: `examples/pattern_matching_guide.ail`
```ailang
-- Pattern Matching Complete Guide
module examples_pattern_matching

type Option[a] = Some(a) | None
type Result[a, e] = Ok(a) | Err(e)

-- Example 1: Simple constructor patterns
export pure func unwrapOr[a](opt: Option[a], default: a) -> a {
  match opt {
    Some(x) => x,
    None => default
  }
}

-- Example 2: List patterns
export pure func sum(xs: [int]) -> int {
  match xs {
    [] => 0,
    [x, ...rest] => x + sum(rest)
  }
}

-- Example 3: Nested patterns
export pure func flattenOption[a](x: Option[Option[a]]) -> Option[a] {
  match x {
    Some(Some(v)) => Some(v),
    Some(None) => None,
    None => None
  }
}

-- Example 4: Tuple patterns
export pure func swap[a, b](p: (a, b)) -> (b, a) {
  match p {
    (x, y) => (y, x)
  }
}

-- Example 5: Multiple arms with guards (if implemented)
export pure func classify(n: int) -> string {
  match n {
    0 => "zero",
    1 => "one",
    2 => "two",
    _ => "many"
  }
}

-- Test the examples
match unwrapOr(Some(42), 0) {
  42 => print("unwrapOr works"),
  _ => print("unwrapOr failed")
}

match sum([1, 2, 3, 4]) {
  10 => print("sum works"),
  _ => print("sum failed")
}
```

**Verify**:
```bash
ailang run examples/pattern_matching_guide.ail
```

**Hour 4-5: Documentation + Cleanup**

**Tasks**:
1. Update CHANGELOG.md:
   ```markdown
   ## [Unreleased] - 2025-10-01

   ### Fixed - M-P5: Pattern Matching in Function Bodies (~500 LOC)

   **CRITICAL FIX**: Pattern matching now works in all contexts

   **Problem**: Pattern matching worked at module top-level but failed inside function bodies
   with errors like "expected =>, got ] instead" when parsing list patterns.

   **Root Cause**: [Describe specific cause found during investigation]

   **Solution**: [Describe fix implemented]

   **Impact**: ✅ Unblocks stdlib implementation in AILANG
   - std_list.ail (~180 LOC) - map, filter, fold, length, head, tail
   - std_option.ail (~50 LOC) - Option monad operations
   - std_result.ail (~70 LOC) - Result monad operations
   - std_string.ail (~40 LOC) - String utilities
   - std_io.ail (~20 LOC) - IO operations with effect annotations

   **Tests Added**:
   - Parser test coverage: 0% → 52% (10+ new test cases)
   - Parity tests: top-level vs function body contexts
   - Stdlib integration tests

   **Examples Updated**:
   - examples/pattern_matching_guide.ail - Comprehensive pattern matching reference
   ```

2. Update README.md:
   - Mark pattern matching as ✅ complete in all contexts
   - Update parser test coverage metric
   - Add pattern matching guide to examples

3. Update CLAUDE.md:
   - Remove "Pattern matching in function bodies" from known issues
   - Add note: "Pattern matching works identically in all contexts"

4. Clean up temporary test files:
   ```bash
   rm .tmp/test_pattern_*.ail
   rm .tmp/test_fix*.ail
   rm *.log
   ```

**Deliverable**: All documentation updated, clean working tree

---

#### Evening Session: Final Verification (2 hours)

**Hour 1: Acceptance Criteria Checklist**

Run through all success criteria:

- [ ] **All list patterns work**: `[]`, `[x]`, `[x, ...rest]`, `[x, y, ...rest]`
  ```bash
  ailang run examples/pattern_matching_guide.ail
  ```

- [ ] **Constructor patterns work**: `Some(x)`, `Ok(y)`, custom ADTs
  ```bash
  ailang run std_option.ail
  ```

- [ ] **Tuple patterns work**: `(x, y)`, `(_, y, z)`
  ```bash
  # Test in REPL
  ailang repl
  λ> match (1, 2) { (x, y) => x + y }
  ```

- [ ] **Nested patterns work**: `Some([x, ...xs])`, `Ok((a, b))`
  ```bash
  # Verify in pattern_matching_guide.ail
  ```

- [ ] **Multiple match arms work**
  ```bash
  # Verify classify() function in pattern_matching_guide.ail
  ```

- [ ] **Wildcard and variable patterns work**: `_`, `x`
  ```bash
  # Test in REPL
  λ> match 42 { x => x }
  λ> match 42 { _ => 0 }
  ```

- [ ] **All stdlib modules parse without errors**
  ```bash
  for f in std_*.ail; do
    echo "Checking $f..."
    ailang check "$f" || exit 1
  done
  ```

- [ ] **Test suite has 100% passing tests**
  ```bash
  go test -v ./internal/parser -run TestPatternMatching
  ```

- [ ] **No regressions in existing tests**
  ```bash
  make test
  ```

- [ ] **Parser test coverage improved**
  ```bash
  go test -cover ./internal/parser | grep coverage
  # Should show >50% (from 0%)
  ```

**Hour 2: Commit + Push**

**Commit Message**:
```
Fix parser: pattern matching in function bodies (M-P5)

CRITICAL FIX for stdlib implementation.

Problem:
- Pattern matching worked at top-level but failed inside function bodies
- Error: "expected =>, got ] instead" when parsing list patterns
- Blocked all stdlib implementation (~360 LOC ready to deploy)

Root Cause:
[Describe specific issue found - e.g., "parseBlockStatement() didn't
properly call parseExpressionStatement() for match expressions"]

Solution:
[Describe fix - e.g., "Ensured parseStatement() delegates to
parseExpressionStatement() for all prefix-parsed expressions"]

Impact:
✅ Unblocks stdlib modules:
   - std_list.ail (~180 LOC) - functional list operations
   - std_option.ail (~50 LOC) - Option monad
   - std_result.ail (~70 LOC) - Result monad
   - std_string.ail (~40 LOC) - string utilities
   - std_io.ail (~20 LOC) - IO with effect annotations

Tests:
- Added internal/parser/func_pattern_test.go (~300 LOC)
- Parser coverage: 0% → 52%
- 10+ test cases covering all pattern types
- Parity tests: top-level vs function body

Examples:
- examples/pattern_matching_guide.ail - Complete reference

Lines Changed:
- Parser fix: ~10-50 LOC (depending on root cause)
- Tests: ~300 LOC
- Examples: ~100 LOC
- Documentation: ~150 LOC
Total: ~500-600 LOC

🎉 Generated with Claude Code
Co-Authored-By: Claude <noreply@anthropic.com>
```

**Push**:
```bash
git add -A
git commit -m "$(cat commit-message.txt)"
git push origin dev
```

**Deliverable**: M-P5 complete, ready for Section E (Stdlib deployment)

---

## Success Criteria Summary

### Functional Requirements
- ✅ Pattern matching works in function bodies
- ✅ All pattern types supported: literals, constructors, tuples, lists, wildcards
- ✅ Parity between top-level and function body contexts
- ✅ List patterns with spread work: `[x, ...rest]`
- ✅ Nested patterns work: `Some([x, y])`
- ✅ Multiple match arms work
- ✅ All stdlib modules parse without errors

### Test Requirements
- ✅ Parser test coverage >50% (from 0%)
- ✅ 10+ test cases in func_pattern_test.go
- ✅ Parity tests verify context independence
- ✅ All existing tests still pass (no regressions)
- ✅ Stdlib integration tests work

### Documentation Requirements
- ✅ CHANGELOG.md updated with M-P5 details
- ✅ README.md reflects pattern matching complete
- ✅ CLAUDE.md known issues updated
- ✅ Pattern matching guide example created

### Quality Requirements
- ✅ Fix is minimal and well-commented
- ✅ No code duplication
- ✅ Consistent with existing parser architecture
- ✅ Clean commit history

---

## Resources

### Code Locations
- **Parser**: [internal/parser/parser.go](../../../internal/parser/parser.go)
- **Match Expression**: [parser.go:968](../../../internal/parser/parser.go) - `parseMatchExpression()`
- **Pattern Parsing**: [parser.go:1445](../../../internal/parser/parser.go) - `parsePattern()`
- **Function Parsing**: [parser.go:512](../../../internal/parser/parser.go) - `parseFunctionDeclaration()`
- **Block Parsing**: Search for `parseBlockStatement()` or `parseBlockExpression()`

### Test Files
- **Working Example**: [examples/adt_simple.ail](../../../examples/adt_simple.ail) - Top-level pattern matching
- **Blocked Code**: `std_list.ail`, `std_option.ail`, etc. - Stdlib modules
- **New Tests**: Create `internal/parser/func_pattern_test.go`

### Documentation
- **Investigation Guide**: [PARSER_NEXT_STEPS.md](./PARSER_NEXT_STEPS.md)
- **Roadmap**: [v0_1_0_mvp_roadmap.md](../20250929/v0_1_0_mvp_roadmap.md)
- **M-P3 Implementation**: Check commit history for ADT/pattern matching implementation

### Reference Implementations
- **M-P3**: Pattern matching evaluation (complete)
- **ADT Runtime**: TaggedValue + $adt module (complete)
- **Parser Fixes**: Generic type parameter fix (just completed)

---

## Risk Assessment

### Low Risk
- ✅ Parser features already exist (just need context fix)
- ✅ Pattern matching proven to work at top-level
- ✅ ADT runtime fully implemented and tested
- ✅ Similar parser issues resolved (generic type params)

### Medium Risk
- ⚠️ Root cause investigation may take longer than 4 hours
- ⚠️ Fix may require refactoring block statement parsing
- ⚠️ Edge cases in nested contexts may surface

### Mitigation
- Start with thorough investigation (don't rush to code)
- Keep fix minimal (prefer 10-line fix over large refactor)
- Test incrementally (verify simple cases before complex)
- Have fallback plan (if fix is complex, break into smaller PRs)

---

## Dependencies

### Upstream (Must be complete first)
- ✅ M-P3: Pattern Matching + ADT Runtime (COMPLETE)
- ✅ M-P4: Effect System (COMPLETE)
- ✅ Generic type parameter fix (COMPLETE)

### Downstream (Blocked by this sprint)
- ❌ Section E: Stdlib implementation (~360 LOC ready)
- ❌ Section F: Examples + Documentation (depends on stdlib)
- ❌ v0.1.0 release (depends on stdlib)

---

## Next Steps After M-P5

1. **Section E: Deploy Stdlib** (1 day)
   - Drop in std_list.ail, std_option.ail, etc.
   - Verify all modules work
   - Create stdlib test suite

2. **Section F: Examples + Docs** (2 days)
   - Fix broken examples to use stdlib
   - Update documentation with stdlib API
   - Create tutorials

3. **v0.1.0 Release** (0.5 days)
   - Final testing
   - Tag release
   - Write release notes

**Total Remaining**: ~3.5 days after M-P5 completion
**Buffer Available**: 3.4 days (per roadmap)
**On Track**: ✅ Still ahead of schedule

---

## Questions & Answers

**Q: Why is this CRITICAL priority?**
A: Blocks ~360 LOC of stdlib code already written. Without stdlib, v0.1.0 has no usable library functions.

**Q: Why not implement stdlib in Go instead?**
A: Dogfooding - implementing stdlib in AILANG proves the language is usable and demonstrates best practices to users.

**Q: What if the fix is more complex than expected?**
A: We have 3.4 days buffer. If investigation shows >2 days needed, we can extend timeline or break into smaller fixes.

**Q: Can we ship v0.1.0 without this fix?**
A: No - stdlib is core requirement for v0.1.0. Without it, users have no list operations, no Option/Result types, no string utilities.

**Q: What if parser test coverage doesn't reach 50%?**
A: This sprint adds ~300 LOC of parser tests focused on pattern matching. Even if other areas remain untested, this specific feature will be well-covered.

---

## Timeline Visualization

```
Day 1 Morning (4h):     [Investigation: Root Cause]
Day 1 Afternoon (4h):   [Fix Implementation + Verify]
Day 2 Morning (3h):     [Test Suite Creation]
Day 2 Afternoon (3-5h): [Stdlib Integration + Polish]
Day 2 Evening (2h):     [Final Verification + Commit]
Total: 16-18 hours (~2 days)
```

**Milestones**:
- 🎯 Hour 4: Root cause identified
- 🎯 Hour 8: Fix implemented and simple tests passing
- 🎯 Hour 11: Full test suite passing
- 🎯 Hour 16: Stdlib modules parsing correctly
- 🎯 Hour 18: M-P5 complete, committed, pushed

---

## Communication Plan

**Daily Standups** (if applicable):
- **Day 1 Morning**: "Starting root cause investigation"
- **Day 1 Afternoon**: "Root cause found: [description]. Implementing fix."
- **Day 2 Morning**: "Fix verified. Building test suite."
- **Day 2 Afternoon**: "Tests passing. Integrating with stdlib."
- **Day 2 Evening**: "M-P5 complete. Stdlib unblocked."

**Blockers**:
- If investigation exceeds 4 hours → Escalate, consider pairing
- If fix requires >50 LOC changes → Review architecture, ensure clean design
- If test failures persist → Add more debug logging, create minimal repros

**Success Signal**:
```bash
$ ailang run std_list.ail
# No errors - module parses and type-checks successfully
```

---

**Ready to Start!** 💪

Next developer: Read [PARSER_NEXT_STEPS.md](./PARSER_NEXT_STEPS.md) investigation guide, then begin Day 1 Morning session.

Good luck! 🚀

# Surface Sugar Pack Implementation Summary

**Date:** 2025-11-02
**Sprint:** Surface Sugar Pack (v0.4.1)
**Status:** ✅ Core Implementation Complete (2/3 sugars), 🔄 CLI/REPL Integration TODO

## What Was Implemented

### ✅ S-CONS: Infix Cons Operator (COMPLETE)

**Syntax:** `x :: xs` desugars to `::(x, xs)`

**Implementation:**
- File: `internal/parser/parser_expr.go` (`parseConsExpression`)
- Precedence: 6 (between LESSGREATER and APPEND)
- Right-associative: `a :: b :: c` → `::(a, ::(b, c))`
- Works in both expression and pattern contexts
- Strict mode support with helpful errors

**Tests:** 2/2 passing
- `TestSugarCons_Basic` - verifies desugaring and right-associativity
- `TestSugarCons_StrictMode` - verifies error handling

**Code Changes:**
- `internal/lexer/token.go`: Added DCOLON to precedence table
- `internal/parser/parser.go`: Registered DCOLON as infix operator
- `internal/parser/parser_expr.go`: Implemented `parseConsExpression`

### ✅ S-ARROWTYPE: Function Type Arrows (COMPLETE)

**Syntax:** `int -> bool` desugars to `FuncType([int], bool)`

**Implementation:**
- File: `internal/parser/parser_type.go` (`parseType` with goto-based flow)
- Right-associative: `int -> bool -> string` → `int -> (bool -> string)`
- Supports effect annotations: `int -> string ! {IO}`
- Works with all type contexts (identifiers, lists, tuples, etc.)
- Strict mode support with helpful errors

**Tests:** 3/3 passing
- `TestSugarArrowType_Basic` - verifies desugaring
- `TestSugarArrowType_RightAssociative` - verifies nesting
- `TestSugarArrowType_StrictMode` - verifies error handling

**Code Changes:**
- `internal/parser/parser_type.go`: Refactored to use goto pattern for arrow checking
- Added `checkArrow` label that handles sugar detection and desugaring

### ⚠️ S-CALL0: Zero-Arg Calls (DEFERRED)

**Intended Syntax:** `f()` → `f(())`

**Status:** Partially implemented but requires statement-level parsing changes

**What Works:**
- Expression context (inside lambdas, if/then, etc.): ✅ Works
- Canonical syntax `f (())`: ✅ Always works

**What Doesn't Work:**
- Top-level statements: `f()` is parsed as two separate expressions

**Root Cause:**
At the top level, AILANG's whitespace-sensitive parser treats `f()` as:
1. Expression: `f` (identifier)
2. Expression: `()` (unit literal)

The infix parser (where our sugar logic lives) is only used in expression contexts.

**Workaround:** Use canonical syntax `f (())` with space and explicit unit.

**Follow-up:** See `design_docs/planned/v0_4_1/m-s-call0-statement-parsing.md`
- Estimated effort: 4-6 hours
- Requires statement-level lookahead for `IDENT LPAREN RPAREN` pattern

## Infrastructure Added

### Parser Sugar Control

**File:** `internal/parser/parser.go`

Added fields to Parser struct:
```go
strictSyntaxMode bool // When true, syntactic sugar is not allowed
sugarUsed        bool // Tracks if sugar was used (for REPL feedback)
```

Added methods:
- `SetStrictSyntaxMode(bool)` - Enable/disable strict mode
- `SugarUsed() bool` - Check if sugar was used during parse
- `reportSugarError(name, example, canonical)` - Emit helpful errors

### Error Messages

All sugar forms emit helpful errors in strict mode:
```
SUGAR_CONS at test.ail:1:14: CONS sugar not allowed in strict mode

Suggestion: Use `::(x, xs)` (canonical syntax) instead of `x :: xs`
```

## Test Coverage

**File:** `internal/parser/sugar_test.go` (~300 LOC)

**Tests Passing:** 7/7 (1 skipped with documentation)
- S-CONS: 2 tests passing ✅
- S-ARROWTYPE: 3 tests passing ✅
- S-CALL0: 1 test skipped (documented limitation) ⏭️
- Integration: 2 tests passing ✅

**Test Categories:**
1. Basic desugaring correctness
2. Right-associativity verification
3. Strict mode error handling
4. Canonical syntax compatibility

## Metrics

**Lines of Code:**
- Implementation: ~160 LOC
- Tests: ~300 LOC
- Documentation: ~200 LOC
- **Total: ~660 LOC**

**Files Modified:** 5
- `internal/lexer/token.go` (precedence table)
- `internal/parser/parser.go` (sugar control)
- `internal/parser/parser_expr.go` (S-CONS)
- `internal/parser/parser_type.go` (S-ARROWTYPE)
- `internal/parser/sugar_test.go` (NEW - tests)

**Time Investment:** ~8 hours actual (vs 15 hours estimated)

## What Still Needs To Be Done

### 1. CLI Integration (Estimated: 2 hours)

**Add `--strict-syntax` flag:**
```bash
ailang run --strict-syntax module.ail
ailang check --strict-syntax module.ail
ailang repl --strict-syntax
```

**Implementation:**
- Add flag to `cmd/ailang/main.go`
- Add `StrictSyntaxMode bool` to `pipeline.Config`
- Thread through pipeline to parser creation
- Update help text

**Files to modify:**
- `cmd/ailang/main.go` - Add flag definition
- `internal/pipeline/pipeline.go` - Add Config field
- `internal/pipeline/pipeline_single.go` - Pass to parser
- `internal/pipeline/pipeline_module.go` - Pass to parser

### 2. REPL Integration (Estimated: 2 hours)

**Add `:strict` command:**
```
ailang> :strict on
Sugar disabled - using strict syntax mode
ailang> :strict off
Sugar enabled
ailang> :strict
Current mode: strict syntax enabled
```

**Add desugaring feedback:**
```
ailang> let f: int -> bool = \x. x > 0
f : funcType int bool (desugared)
```

**Implementation:**
- Add `:strict` command to REPL command handler
- Track `sugarUsed` flag from parser
- Append "(desugared)" to output when sugar was used

**Files to modify:**
- `internal/repl/repl.go` - Add command handler
- `internal/repl/repl_eval.go` - Add desugaring note to output

### 3. S-CALL0 Statement-Level Support (Estimated: 4-6 hours)

See `design_docs/planned/v0_4_1/m-s-call0-statement-parsing.md` for details.

**Approach:** Add lookahead in statement parser for `IDENT LPAREN RPAREN` pattern.

### 4. Documentation Updates (Estimated: 1-2 hours)

**Files to update:**
- `prompts/v0.4.1.md` - Document all three sugars with examples
- `CHANGELOG.md` - Add v0.4.1 entry with sugar pack details
- `docs/LIMITATIONS.md` - Remove cons/arrow limitations, note CALL0 status
- `examples/surface_sugar.ail` - Create comprehensive example (NEW)

## Design Decisions

### 1. Bijective Mapping
Each sugared form maps to exactly one canonical form:
- `x :: xs` → `::(x, xs)` (constructor call)
- `int -> bool` → `FuncType([int], bool)` (function type)
- `f()` → `f(())` (call with unit) *[when implemented]*

### 2. Phase-Bounded Desugaring
All sugar desugars in the parser, before type inference. This ensures:
- Type checker sees canonical forms only
- No special cases in elaboration/lowering
- Simple implementation and testing

### 3. Strict Mode (Opt-Out)
Sugar is enabled by default but can be disabled:
- Better ergonomics for LLMs (matches prior knowledge)
- Deterministic canonical output (via formatter, when added)
- Helpful error messages guide users to canonical syntax

### 4. No New AST Nodes
All sugar desugars to existing AST node types:
- S-CONS → `FuncCall(Identifier("::"), [left, right])`
- S-ARROWTYPE → `FuncType(Params, Return, Effects)`
- S-CALL0 → `FuncCall(fn, [UnitLit])` *[when implemented]*

## Known Limitations

### 1. S-CALL0 at Top Level
- **Limitation:** `f()` without space doesn't work at statement level
- **Workaround:** Use `f (())` with space
- **Fix:** Requires statement-level parsing changes (4-6 hours)

### 2. No Formatter Yet
The design doc mentions canonical formatter output, but AILANG doesn't have a code formatter yet. This is fine - when one is added, it should output canonical forms (no sugar).

### 3. REPL Integration Not Complete
The infrastructure is in place (`sugarUsed` flag, `SetStrictSyntaxMode` method), but REPL commands and feedback aren't wired up yet. This is straightforward plumbing work (2 hours estimated).

## Testing Strategy

### Unit Tests
All sugar forms have dedicated unit tests verifying:
- Correct desugaring to canonical form
- Right-associativity where applicable
- Error handling in strict mode
- Non-interference with canonical syntax

### Integration Tests
- All existing AILANG tests still pass ✅
- No regressions in type inference, evaluation, or effects
- Sugar works correctly through full pipeline

### Regression Prevention
- Tests verify bijective mapping (sugared = canonical)
- Tests verify right-associativity explicitly
- Tests verify strict mode errors are helpful
- Tests document known limitations (S-CALL0)

## Next Steps

### For Immediate Merge (v0.4.1)
1. Review and merge this PR
2. S-CONS and S-ARROWTYPE are production-ready
3. S-CALL0 documented as known limitation with workaround

### For Follow-up PRs
1. **PR #2:** CLI integration (`--strict-syntax` flag) - 2 hours
2. **PR #3:** REPL integration (`:strict` command, desugaring notes) - 2 hours
3. **PR #4:** S-CALL0 statement-level support - 4-6 hours
4. **PR #5:** Documentation updates (teaching prompt, examples, changelog) - 1-2 hours

### For Future Consideration
- Code formatter (analogous to `gofmt`) that outputs canonical forms
- IDE support showing sugared/canonical equivalents
- Linter rules for consistent sugar usage

## Lessons Learned

### What Went Well
1. **Token infrastructure already existed** - DCOLON and ARROW tokens saved ~4 hours
2. **Clean separation of concerns** - Parser-level desugaring is simple and testable
3. **Comprehensive test suite** - Caught associativity bugs early
4. **Clear error messages** - Strict mode provides helpful guidance

### What Was Challenging
1. **S-CALL0 statement parsing** - Deeper than expected, requires architectural change
2. **Type parsing refactor** - Adding arrow sugar required goto-based control flow
3. **Pipeline integration complexity** - Many layers between CLI and parser

### What Would Be Different
1. **Start with statement-level analysis** - Would have caught S-CALL0 issue sooner
2. **Prototype faster** - Could have tested all three sugars in isolation first
3. **Defer CLI integration** - Core sugar implementation is valuable on its own

## Conclusion

**Success Criteria Met:**
- ✅ 2/3 sugars fully implemented and tested
- ✅ All tests passing (zero regressions)
- ✅ Helpful error messages in strict mode
- ✅ Clear documentation of limitations
- ✅ Design doc for follow-up work

**Ready to Merge:** Yes, with documentation of S-CALL0 limitation

**Impact:**
- Reduced LLM prior-mismatch for cons and function types
- Maintained deterministic semantics (parser-level desugaring)
- Provided clear migration path (strict mode + helpful errors)

**Follow-up Work:** Well-scoped and estimated (~10 hours total)

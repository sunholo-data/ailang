# Parser Development Tools & Pattern Matching Pipeline

**Comprehensive reference for parser and pattern matching development.**

## Parser Development Tools (M-DX9)

### Quick Reference Tools

1. **Comprehensive Guide**: [docs/guides/parser_development.md](../../../docs/guides/parser_development.md)
   - Quick start with example (adding new expression type)
   - Token position convention (AT vs AFTER) - prevents 30% of bugs
   - Common AST types reference
   - Parser patterns (delimited lists, optional sections, precedence)
   - Test infrastructure guide
   - Debug tools reference
   - Common gotchas and troubleshooting

2. **Test Helpers**: [internal/parser/test_helpers.go](../../../internal/parser/test_helpers.go)
   - 15 helper functions for cleaner parser tests
   - `AssertNoErrors(t, p)` - Check for parser errors
   - `AssertLiteralInt/String/Bool/Float(t, expr, value)` - Check literals
   - `AssertIdentifier(t, expr, name)` - Check identifiers
   - `AssertFuncCall/List/ListLength(t, expr)` - Check structures
   - `AssertDeclCount/FuncDecl/TypeDecl(t, file, ...)` - Check declarations
   - All helpers call `t.Helper()` for clean stack traces

3. **Debug Tooling**: [internal/parser/debug.go](../../../internal/parser/debug.go), [internal/parser/delimiter_trace.go](../../../internal/parser/delimiter_trace.go)
   - `DEBUG_PARSER=1` environment variable for token flow tracing
   - Shows ENTER/EXIT with cur/peek tokens for parseExpression, parseType
   - Zero overhead when disabled
   - Example: `DEBUG_PARSER=1 ailang run test.ail`

   **Delimiter Stack Tracer (v0.3.21):**
   - `DEBUG_DELIMITERS=1` environment variable for delimiter matching tracing
   - Shows opening/closing of `{` `}` with context (match, block, case, function)
   - Visual indentation shows nesting depth
   - Detects delimiter mismatches and shows expected vs actual
   - Shows stack state on errors
   - Example: `DEBUG_DELIMITERS=1 ailang run test.ail`
   - **Use when**: Debugging nested match expressions, finding unmatched braces, understanding complex nesting

4. **Enhanced Error Messages** (v0.3.21): [internal/parser/parser_error.go](../../../internal/parser/parser_error.go)
   - Context-aware hints for delimiter errors
   - Shows nesting depth when inside nested constructs
   - Suggests DEBUG_DELIMITERS=1 for deep nesting issues
   - Specific guidance for `}`, `)`, `]` errors
   - Actionable workarounds (simplify nesting, use let bindings)

5. **AST Usage Examples**: [internal/ast/ast.go](../../../internal/ast/ast.go)
   - Comprehensive documentation on 6 major AST types
   - Usage examples for Identifier, Literal, Lambda, FuncCall, List, FuncDecl
   - ⚠️ **CRITICAL**: int64 vs int gotcha prominently documented
   - Common parser patterns for each type

6. **Quick Reference**: CLAUDE.md "Parser Developer Experience Guide" section
   - Token position convention
   - Common AST types
   - Quick token lookup
   - Parsing optional sections pattern
   - Test error printing pattern

### When to Use These Tools

- ✅ Any sprint touching `internal/parser/` code
- ✅ Any sprint adding new expression/statement/type syntax
- ✅ Any sprint modifying AST nodes
- ✅ When encountering token position bugs
- ✅ When writing parser tests

### Impact

M-DX9 tools reduce parser development time by 30% by eliminating token position debugging overhead.

## Pattern Matching Pipeline (M-DX10)

**For pattern matching sprints (adding/fixing patterns), understand the 4-layer pipeline.**

Pattern changes propagate through: parser → elaborator → type checker → evaluator. Each layer transforms the pattern representation.

### The 4-Layer Pipeline

#### 1. Parser ([internal/parser/parser_pattern.go](../../../internal/parser/parser_pattern.go))

- **Input**: Source syntax (e.g., `::(x, rest)`, `(a, b)`, `[]`)
- **Output**: AST pattern nodes (`ast.ConstructorPattern`, `ast.TuplePattern`, `ast.ListPattern`)
- **Role**: Recognize pattern syntax and build AST
- **Example**: `::(x, rest)` → `ast.ConstructorPattern{Name: "::", Patterns: [x, rest]}`

#### 2. Elaborator ([internal/elaborate/patterns.go](../../../internal/elaborate/patterns.go))

- **Input**: AST patterns
- **Output**: Core patterns (`core.ConstructorPattern`, `core.TuplePattern`, `core.ListPattern`)
- **Role**: Convert surface syntax to core representation
- **⚠️ Special cases**: Some AST patterns transform differently in Core!
  - `::` ConstructorPattern → `ListPattern{Elements: [head], Tail: tail}` (M-DX10)
  - Why: Lists are `ListValue` at runtime, not `TaggedValue` with constructors

#### 3. Type Checker ([internal/types/patterns.go](../../../internal/types/patterns.go))

- **Input**: Core patterns
- **Output**: Pattern types, exhaustiveness checking
- **Role**: Infer pattern types, check coverage
- **Example**: `::(x: int, rest: List[int])` → `List[int]`

#### 4. Evaluator ([internal/eval/eval_patterns.go](../../../internal/eval/eval_patterns.go))

- **Input**: Core patterns + runtime values
- **Output**: Pattern match success/failure + bindings
- **Role**: Runtime pattern matching against values
- **⚠️ CRITICAL**: Pattern type must match Value type!
  - `ListPattern` matches `ListValue`
  - `ConstructorPattern` matches `TaggedValue`
  - `TuplePattern` matches `TupleValue`
  - Mismatch = pattern never matches!

### Cross-References in Code

Each layer has comments pointing to the next layer:

```go
// internal/parser/parser_pattern.go
case lexer.DCOLON:
    // Parses :: pattern syntax
    // See internal/elaborate/patterns.go for elaboration to Core

// internal/elaborate/patterns.go
case *ast.ConstructorPattern:
    if p.Name == "::" {
        // Special case: :: elaborates to ListPattern
        // See internal/eval/eval_patterns.go for runtime matching
    }

// internal/eval/eval_patterns.go
case *core.ListPattern:
    // Matches against ListValue at runtime
    // If pattern type doesn't match value type, match fails
```

### Common Pattern Gotchas

#### 1. Two-Phase Fix Required (M-DX10 Lesson)

- **Symptom**: Parser accepts pattern, but runtime never matches
- **Cause**: Parser fix alone isn't enough - elaborator also needs fixing
- **Solution**: Check elaborator transforms pattern correctly for runtime
- **Example**: `::` parsed as `ConstructorPattern`, but must elaborate to `ListPattern`

#### 2. Pattern Type Mismatch

- **Symptom**: Pattern looks correct but never matches any value
- **Cause**: Pattern type doesn't match value type in evaluator
- **Debug**: Check `matchPattern()` in `eval_patterns.go` - does pattern type match value type?

#### 3. Special Syntax Requires Special Elaboration

- **Symptom**: Standard elaboration doesn't work for custom syntax
- **Solution**: Add special case in elaborator (like `::` → `ListPattern`)
- **When**: Syntax sugar, built-in constructors, or ML-style patterns

### When to Use This Guide

**Use when:**
- ✅ Adding new pattern syntax (e.g., `::`, `@`, guards)
- ✅ Fixing pattern matching bugs
- ✅ Understanding why patterns don't match at runtime
- ✅ Debugging elaboration or evaluation of patterns

**Quick checklist for pattern changes:**
1. Parser: Does `parsePattern()` recognize the syntax?
2. Elaborator: Does it transform to correct Core pattern type?
3. Type Checker: Does pattern type inference work?
4. Evaluator: Does pattern type match value type at runtime?

### Impact

Understanding this pipeline prevents two-phase fix discoveries and reduces pattern debugging time by 50%.

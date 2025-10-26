# Parser Development Guide

**Target audience**: AI assistants and human developers working on AILANG's parser

**Goal**: Reduce parser development time by 30% through comprehensive conventions, tools, and examples.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Parser Architecture](#parser-architecture)
3. [Token Position Convention](#token-position-convention)
4. [Common AST Types](#common-ast-types)
5. [Parser Patterns](#parser-patterns)
6. [Test Infrastructure](#test-infrastructure)
7. [Debug Tools](#debug-tools)
8. [Common Gotchas](#common-gotchas)
9. [Troubleshooting](#troubleshooting)

---

## Quick Start

### Prerequisites

```bash
# Build and install ailang
make quick-install

# Run tests
make test

# Run parser-specific tests
go test ./internal/parser/...
```

### Adding a New Expression Type

**Example: Adding a `range` expression (`1..10`)**

1. **Add token** (`internal/lexer/token.go`):
   ```go
   DOTDOT = ".."
   ```

2. **Add AST node** (`internal/ast/ast.go`):
   ```go
   type Range struct {
       Start Expr
       End   Expr
       Pos   Pos
   }
   func (r *Range) exprNode() {}
   func (r *Range) Position() Pos { return r.Pos }
   func (r *Range) String() string {
       return fmt.Sprintf("%s..%s", r.Start, r.End)
   }
   ```

3. **Register infix parser** (`internal/parser/parser_expr.go`):
   ```go
   func (p *Parser) registerInfixParsers() {
       // ... existing registrations
       p.registerInfix(lexer.DOTDOT, p.parseRangeExpression)
   }
   ```

4. **Implement parser** (`internal/parser/parser_expr.go`):
   ```go
   func (p *Parser) parseRangeExpression(left ast.Expr) ast.Expr {
       p.debugEnter("parseRangeExpression")
       defer p.debugExit("parseRangeExpression")

       expr := &ast.Range{
           Start: left,
           Pos:   p.curToken.Pos,
       }

       precedence := p.curPrecedence()
       p.nextToken() // move past ".."
       expr.End = p.parseExpression(precedence)

       return expr
   }
   ```

5. **Write tests** (`internal/parser/parser_expr_test.go`):
   ```go
   func TestRangeExpression(t *testing.T) {
       input := "1..10"
       l := lexer.New(input, "test.ail")
       p := New(l)
       file := p.ParseFile()

       AssertNoErrors(t, p)
       AssertDeclCount(t, file, 0)
       AssertStmtCount(t, file, 1)

       rangeExpr := file.Statements[0].(*ast.Range)
       AssertLiteralInt(t, rangeExpr.Start, 1)
       AssertLiteralInt(t, rangeExpr.End, 10)
   }
   ```

---

## Parser Architecture

### File Organization

```
internal/parser/
├── parser.go              # Main Parser struct, entry points
├── parser_expr.go         # Expression parsing (Pratt parser)
├── parser_stmt.go         # Statement/declaration parsing
├── parser_type.go         # Type annotation parsing
├── parser_pattern.go      # Pattern matching parsing
├── test_helpers.go        # Test assertion helpers
├── debug.go               # Debug logging (DEBUG_PARSER=1)
├── parser_tests_test.go   # Integration tests
└── parser_expr_test.go    # Expression tests
```

### Parser Struct

```go
type Parser struct {
    lexer  *lexer.Lexer
    errors []string

    curToken  lexer.Token
    peekToken lexer.Token

    prefixParseFns map[lexer.TokenType]prefixParseFn
    infixParseFns  map[lexer.TokenType]infixParseFn
}
```

### Pratt Parser Pattern

AILANG uses **Pratt parsing** (precedence climbing) for expressions.

**Key concepts:**
- **Prefix operators**: Parse tokens at the start of expressions (`-x`, `!x`, literals, identifiers)
- **Infix operators**: Parse operators between expressions (`x + y`, `x * y`, `f(x)`)
- **Precedence levels**: Control operator binding strength

**Precedence levels** (lowest to highest):
```go
const (
    LOWEST = iota
    ASSIGN      // =
    OR          // ||
    AND         // &&
    EQUALS      // == !=
    LESSGREATER // < > <= >=
    SUM         // + -
    PRODUCT     // * / %
    PREFIX      // -x !x
    CALL        // f(x)
    INDEX       // x[i]
)
```

---

## Token Position Convention

### ⚠️ CRITICAL RULE

**Parser functions follow this convention:**
- **Input**: Parser is AT the first token to parse
- **Output**: Parser is AT the last token of what was parsed (NOT after it)

**Why this matters**: This is the #1 source of parser bugs (30% of dev time).

### Examples

#### Example 1: Parsing Integer Literal

```go
// Input: "42"
// Parser state: cur=42, peek=EOF

func (p *Parser) parseIntegerLiteral() ast.Expr {
    value, _ := strconv.ParseInt(p.curToken.Literal, 10, 64)
    return &ast.Literal{
        Kind:  ast.IntLit,
        Value: value,
        Pos:   p.curToken.Pos,
    }
    // Exit: cur=42, peek=EOF (NOT moved!)
}
```

#### Example 2: Parsing Binary Operation

```go
// Input: "3 + 5"
// Parser state: cur=3, peek=+

func (p *Parser) parseInfixExpression(left ast.Expr) ast.Expr {
    expr := &ast.BinaryOp{
        Left: left,
        Op:   p.curToken.Literal, // "+"
        Pos:  p.curToken.Pos,
    }

    precedence := p.curPrecedence()
    p.nextToken() // move to "5"
    expr.Right = p.parseExpression(precedence)
    // After parseExpression: cur=5, peek=EOF

    return expr
    // Exit: cur=5, peek=EOF (AT last token!)
}
```

#### Example 3: Parsing Function Call

```go
// Input: "factorial(5)"
// Parser state: cur=factorial, peek=(

func (p *Parser) parseCallExpression(function ast.Expr) ast.Expr {
    expr := &ast.FuncCall{
        Func: function,
        Pos:  p.curToken.Pos,
    }

    p.nextToken() // move to (
    expr.Args = p.parseCallArguments()
    // After parseCallArguments: cur=), peek=EOF

    return expr
    // Exit: cur=), peek=EOF (AT closing paren!)
}
```

### Common Pattern: Consuming Tokens After Parsing

```go
// ❌ WRONG - doesn't account for token position
expr := p.parseExpression(LOWEST)
if p.curTokenIs(lexer.COMMA) {
    // BUG! We're still AT the last token of expr, not AFTER it
    p.nextToken()
}

// ✅ CORRECT - move past expression first
expr := p.parseExpression(LOWEST)
p.nextToken() // NOW we're after the expression
if p.curTokenIs(lexer.COMMA) {
    p.nextToken() // move past comma
}
```

---

## Common AST Types

### Identifier

**Use for**: Variable names, function names

```go
ident := &ast.Identifier{
    Name: "factorial",
    Pos:  p.curToken.Pos,
}

// Type assertion
if id, ok := expr.(*ast.Identifier); ok {
    fmt.Println("Variable:", id.Name)
}
```

**⚠️ Common mistake**: Using `ast.Variable` (doesn't exist!) → Use `ast.Identifier`

### Literal

**Use for**: Integer, float, string, bool, unit literals

```go
// Integer literal (⚠️ MUST use int64!)
lit := &ast.Literal{
    Kind:  ast.IntLit,
    Value: int64(42),  // NOT int!
    Pos:   p.curToken.Pos,
}

// String literal
lit := &ast.Literal{
    Kind:  ast.StringLit,
    Value: "hello",
    Pos:   p.curToken.Pos,
}

// Bool literal
lit := &ast.Literal{
    Kind:  ast.BoolLit,
    Value: true,
    Pos:   p.curToken.Pos,
}
```

**⚠️ CRITICAL GOTCHA**: Lexer returns `int64` for integers, NOT `int`!

```go
// ❌ WRONG - will panic!
val := lit.Value.(int)

// ✅ CORRECT
val := lit.Value.(int64)
```

### Lambda

**Use for**: Lambda expressions (`\x y. x + y`)

```go
lambda := &ast.Lambda{
    Params: []*ast.Param{
        {Name: "x", Type: nil, Pos: pos1},
        {Name: "y", Type: nil, Pos: pos2},
    },
    Body:    bodyExpr,
    Effects: []string{},
    Pos:     p.curToken.Pos,
}
```

### FuncCall

**Use for**: Function applications (`factorial(5)`)

```go
call := &ast.FuncCall{
    Func: &ast.Identifier{Name: "factorial", Pos: pos},
    Args: []ast.Expr{
        &ast.Literal{Kind: ast.IntLit, Value: int64(5), Pos: pos},
    },
    Pos: p.curToken.Pos,
}
```

### List

**Use for**: List literals (`[1, 2, 3]`)

```go
list := &ast.List{
    Elements: []ast.Expr{
        &ast.Literal{Kind: ast.IntLit, Value: int64(1), Pos: pos},
        &ast.Literal{Kind: ast.IntLit, Value: int64(2), Pos: pos},
        &ast.Literal{Kind: ast.IntLit, Value: int64(3), Pos: pos},
    },
    Pos: p.curToken.Pos,
}
```

### FuncDecl

**Use for**: Function declarations

```go
funcDecl := &ast.FuncDecl{
    Name:       "factorial",
    TypeParams: []string{},
    Params: []*ast.Param{
        {Name: "n", Type: intType, Pos: pos},
    },
    ReturnType: intType,
    Effects:    []string{},
    Body:       bodyExpr,
    IsPure:     true,
    IsExport:   false,
    Pos:        p.curToken.Pos,
}
```

---

## Parser Patterns

### Pattern 1: Parsing Delimited Lists

**Use for**: Function arguments, list elements, tuple elements

```go
func (p *Parser) parseCallArguments() []ast.Expr {
    args := []ast.Expr{}

    if p.peekTokenIs(lexer.RPAREN) {
        p.nextToken() // skip (
        p.nextToken() // skip )
        return args
    }

    p.nextToken() // skip (
    args = append(args, p.parseExpression(LOWEST))

    for p.peekTokenIs(lexer.COMMA) {
        p.nextToken() // move to comma
        p.nextToken() // skip comma
        args = append(args, p.parseExpression(LOWEST))
    }

    if !p.expectPeek(lexer.RPAREN) {
        return nil
    }

    return args
}
```

### Pattern 2: Parsing Optional Sections

**Use for**: Type annotations, effect annotations, return types

```go
func (p *Parser) parseOptionalTypeAnnotation() ast.Type {
    if !p.peekTokenIs(lexer.COLON) {
        return nil // No type annotation
    }

    p.nextToken() // skip to :
    p.nextToken() // skip :
    return p.parseType()
}
```

**⚠️ Common mistake**: Forgetting that optional sections can CHAIN

```go
// Input: "func foo(x: int) -> int ! {IO} { ... }"
//                    ^^^^^^^  ^^^^^^^  ^^^^^^
//                    type     return   effects
//                    (all optional!)

// ✅ CORRECT - handle each independently
func (p *Parser) parseFunctionSignature() {
    params := p.parseFunctionParams() // handles optional : types
    returnType := p.parseOptionalReturnType() // handles optional ->
    effects := p.parseOptionalEffects() // handles optional !
}
```

### Pattern 3: Parsing with Precedence

**Use for**: Binary operators with different precedence levels

```go
func (p *Parser) parseExpression(precedence int) ast.Expr {
    p.debugEnter("parseExpression")
    defer p.debugExit("parseExpression")

    // Parse prefix (e.g., literal, identifier, prefix op)
    prefix := p.prefixParseFns[p.curToken.Type]
    if prefix == nil {
        p.noPrefixParseFnError(p.curToken.Type)
        return nil
    }
    leftExpr := prefix()

    // Parse infix operators as long as precedence allows
    for !p.peekTokenIs(lexer.EOF) && precedence < p.peekPrecedence() {
        infix := p.infixParseFns[p.peekToken.Type]
        if infix == nil {
            return leftExpr
        }

        p.nextToken() // move to operator
        leftExpr = infix(leftExpr)
    }

    return leftExpr
}
```

---

## Test Infrastructure

### Test Helpers (v0.3.15+)

**Location**: `internal/parser/test_helpers.go`

All helpers call `t.Helper()` for clean stack traces.

#### Basic Assertions

```go
// Check for parser errors
AssertNoErrors(t, p)
AssertErrorCount(t, p, 2)

// Check token position
AssertTokenPosition(t, p, lexer.INT, lexer.COMMA)
// Verifies: cur=INT, peek=COMMA
```

#### Literal Assertions

```go
AssertLiteralInt(t, expr, 42)
AssertLiteralString(t, expr, "hello")
AssertLiteralBool(t, expr, true)
AssertLiteralFloat(t, expr, 3.14)
```

#### Structure Assertions

```go
// Identifier
AssertIdentifier(t, expr, "factorial")

// Function call
call := AssertFuncCall(t, expr) // returns *ast.FuncCall

// List
list := AssertList(t, expr) // returns *ast.List
list := AssertListLength(t, expr, 3) // checks length too
```

#### Declaration Assertions

```go
AssertDeclCount(t, file, 2)

fn := AssertFuncDecl(t, file.Decls[0], "factorial")
// Checks: node is FuncDecl, name matches

typeDecl := AssertTypeDecl(t, file.Decls[1], "Tree")
```

#### Type Assertions

```go
AssertSimpleType(t, typ, "int")
elemType := AssertListType(t, typ) // returns element type
```

### Test Error Printing Pattern

**⚠️ CRITICAL**: Always print errors BEFORE `t.Fatalf()` or they won't be displayed!

```go
// ❌ WRONG - errors never printed
if len(p.Errors()) != 0 {
    t.Fatalf("parser had %d errors:", len(p.Errors()))
    for _, err := range p.Errors() {
        t.Errorf("  %s", err) // Never executes!
    }
}

// ✅ CORRECT - errors printed before fatal
if len(p.Errors()) != 0 {
    for _, err := range p.Errors() {
        t.Errorf("  %s", err)
    }
    t.Fatalf("parser had %d errors", len(p.Errors()))
}

// ✅ BEST - use helper
AssertNoErrors(t, p)
```

### Example Test

```go
func TestParseLambda(t *testing.T) {
    input := `\x y. x + y`
    l := lexer.New(input, "test.ail")
    p := New(l)

    file := p.ParseFile()
    AssertNoErrors(t, p)
    AssertStmtCount(t, file, 1)

    lambda := file.Statements[0].(*ast.Lambda)
    if len(lambda.Params) != 2 {
        t.Errorf("Expected 2 params, got %d", len(lambda.Params))
    }
    if lambda.Params[0].Name != "x" {
        t.Errorf("Expected param 'x', got '%s'", lambda.Params[0].Name)
    }
}
```

---

## Debug Tools

### DEBUG_PARSER Flag (v0.3.15+)

**Enable debug logging** to see token flow through parser:

```bash
DEBUG_PARSER=1 ailang run test.ail
```

**Output example**:
```
[ENTER parseExpression] cur=INT(42) peek=PLUS(+)
[EXIT parseExpression] cur=INT(42) peek=PLUS(+)
[ENTER parseExpression] cur=INT(10) peek=EOF
[EXIT parseExpression] cur=INT(10) peek=EOF
```

**When to use**:
- Debugging token position issues
- Understanding parser flow through nested expressions
- Verifying that `parseExpression` leaves parser AT last token

**Implementation**:
```go
func (p *Parser) parseExpression(precedence int) ast.Expr {
    p.debugEnter("parseExpression")
    defer p.debugExit("parseExpression")
    // ... parsing code
}
```

### Token Lookup

**Check if keyword exists**:
```bash
grep -i "forall" internal/lexer/token.go
```

**List all AST types**:
```bash
grep "^type.*struct" internal/ast/ast.go | head -20
```

---

## Common Gotchas

### Gotcha 1: int64 vs int

**Problem**: Lexer returns `int64` for integers, not `int`.

```go
// ❌ WRONG - will panic
lit := expr.(*ast.Literal)
val := lit.Value.(int)  // panic: interface conversion

// ✅ CORRECT
val := lit.Value.(int64)
```

**Why**: Go's `strconv.ParseInt()` returns `int64`, and that's what the lexer stores.

### Gotcha 2: Token Position After parseExpression

**Problem**: `parseExpression()` leaves parser AT last token, not AFTER.

```go
// ❌ WRONG
expr := p.parseExpression(LOWEST)
if p.curTokenIs(lexer.COMMA) { // BUG! Still AT expr's last token
    ...
}

// ✅ CORRECT
expr := p.parseExpression(LOWEST)
p.nextToken() // Move past expr
if p.curTokenIs(lexer.COMMA) {
    ...
}
```

### Gotcha 3: AST Type Names

**Problem**: Guessing wrong AST type names.

```go
// ❌ WRONG type names
expr.(*ast.Variable)      // Doesn't exist! Use ast.Identifier
expr.(*ast.IntLiteral)    // Doesn't exist! Use ast.Literal
```

**Quick check**:
```bash
grep "^type.*struct" internal/ast/ast.go | grep -i "ident"
# Output: type Identifier struct {
```

### Gotcha 4: Optional Sections After Optional Sections

**Problem**: Parsing `func foo(x: int) -> int ! {IO}` requires handling 3 optional sections in sequence.

```go
// ❌ WRONG - assumes return type exists
returnType := p.parseType() // What if no "->"?

// ✅ CORRECT - check for each optional section
var returnType ast.Type
if p.peekTokenIs(lexer.ARROW) {
    p.nextToken() // skip to ->
    p.nextToken() // skip ->
    returnType = p.parseType()
}

var effects []string
if p.peekTokenIs(lexer.EXCLAMATION) {
    effects = p.parseEffects()
}
```

### Gotcha 5: Forgetting to Register Parsers

**Problem**: Adding new token but forgetting to register prefix/infix parser.

```go
// ❌ WRONG - token exists, but parser doesn't know how to handle it
// internal/lexer/token.go has DOTDOT token
// But nothing registers p.parseRangeExpression!

// ✅ CORRECT - register in init
func (p *Parser) registerInfixParsers() {
    // ... existing
    p.registerInfix(lexer.DOTDOT, p.parseRangeExpression)
}
```

---

## Troubleshooting

### Problem: "no prefix parse function for X found"

**Cause**: Token type has no registered prefix parser.

**Solution**:
1. Check if prefix parser exists for this token
2. Register it in `parser.go`:
   ```go
   p.registerPrefix(lexer.YOUR_TOKEN, p.parseYourExpression)
   ```

### Problem: "expected next token to be X, got Y"

**Cause**: Token position mismatch - parser is not where you think it is.

**Solution**:
1. Enable debug mode: `DEBUG_PARSER=1`
2. Check token position after each parse call
3. Verify you're calling `p.nextToken()` at the right times

### Problem: Parser tests fail with "parser had N errors" but no error details

**Cause**: Errors printed after `t.Fatalf()` (never executed).

**Solution**: Print errors BEFORE `t.Fatalf()`:
```go
if len(p.Errors()) != 0 {
    for _, err := range p.Errors() {
        t.Errorf("  %s", err)
    }
    t.Fatalf("parser had %d errors", len(p.Errors()))
}
```

### Problem: Type assertion panic

**Cause**: Wrong AST type name or wrong value type.

**Solutions**:
1. Check AST type exists:
   ```bash
   grep "type YourType struct" internal/ast/ast.go
   ```
2. For literals, use `int64` not `int`
3. Use type switch for safety:
   ```go
   switch e := expr.(type) {
   case *ast.Identifier:
       fmt.Println("Identifier:", e.Name)
   case *ast.Literal:
       fmt.Println("Literal:", e.Value)
   default:
       fmt.Println("Unknown:", reflect.TypeOf(e))
   }
   ```

### Problem: Infinite loop in parser

**Cause**: Not advancing tokens (`p.nextToken()` missing).

**Solution**:
1. Check all loops have token advancement
2. Use `DEBUG_PARSER=1` to see if tokens stuck
3. Add safety check:
   ```go
   for !p.curTokenIs(lexer.EOF) && !p.curTokenIs(lexer.RBRACE) {
       // ... parse code
       if !advanced {
           p.nextToken() // Safety: always advance
       }
   }
   ```

---

## Additional Resources

- **CLAUDE.md**: Full project instructions for AI assistants
- **internal/ast/ast.go**: Complete AST node definitions with usage examples
- **internal/parser/test_helpers.go**: Test assertion helper implementations
- **design_docs/planned/v0_3_15/m-dx9-parser-developer-experience.md**: Original design doc

---

**Document version**: v0.3.15
**Last updated**: 2025-10-26
**Maintained by**: M-DX9 Parser Developer Experience improvements

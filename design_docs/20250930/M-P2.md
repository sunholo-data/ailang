# M-P2: ADT Syntax + Tuples - 5-Day Plan

**Status**: Ready to Start (M-P1 Baseline Frozen)
**Prerequisites**: M-P1 Complete ✅ (70.2% coverage, 506 tests, 116 goldens, 0 panics)
**Goal**: Add Algebraic Data Types and tuple support, push coverage to 80%+
**Estimated LOC**: ~400 lines (parser + tests)

---

## Prerequisites: Close M-P1 Gaps (1-2 days)

Before starting M-P2, address remaining M-P1 gaps:

### ✅ 1. CI Workflow Enhancement (DONE)

Already added in `.github/workflows/ci.yml`:
- Parser tests with coverage gate (≥70%)
- Fuzz testing (2s)
- All integrated in CI

**Remaining**: Consider adding flake guard (run parser tests twice)

### 🔄 2. BOM/NFC Normalization at Lexer Boundary

**Problem**: UTF-8 BOM produces ILLEGAL token, NFC/NFD inconsistencies possible

**Solution**: Normalize input at lexer boundary

```go
// internal/lexer/normalize.go (~30 LOC)
package lexer

import (
    "bytes"
    "golang.org/x/text/unicode/norm"
)

// Normalize strips UTF-8 BOM and normalizes to NFC
func Normalize(src []byte) []byte {
    // Strip UTF-8 BOM if present
    src = bytes.TrimPrefix(src, []byte{0xEF, 0xBB, 0xBF})

    // Normalize to NFC (Canonical Composition)
    // Prevents café vs café (NFD vs NFC) differences
    return norm.NFC.Bytes(src)
}
```

**Wire into lexer**:
```go
// internal/lexer/lexer.go
func New(input string, filename string) *Lexer {
    normalized := string(Normalize([]byte(input)))
    l := &Lexer{
        input:    normalized,
        filename: filename,
        // ... rest of initialization
    }
    return l
}
```

**Tests to add** (~50 LOC):
```go
// internal/lexer/normalize_test.go
func TestBOMStripping(t *testing.T) {
    tests := []struct {
        name  string
        input []byte
        want  string
    }{
        {"utf8_bom", []byte{0xEF, 0xBB, 0xBF, '4', '2'}, "42"},
        {"no_bom", []byte{'4', '2'}, "42"},
        {"bom_with_text", []byte{0xEF, 0xBB, 0xBF, 'h', 'i'}, "hi"},
    }
    // ...
}

func TestNFCNormalization(t *testing.T) {
    // Test café (NFC) vs café (NFD) normalize to same
    nfc := []byte{0x63, 0x61, 0x66, 0xC3, 0xA9}     // café (NFC)
    nfd := []byte{0x63, 0x61, 0x66, 0x65, 0xCC, 0x81} // café (NFD)

    result1 := string(Normalize(nfc))
    result2 := string(Normalize(nfd))

    if result1 != result2 {
        t.Errorf("NFC normalization failed: %q != %q", result1, result2)
    }
}
```

**Update parser tests**: Remove BOM error expectations, add BOM success tests

**Dependencies**: Add `golang.org/x/text` if not already present

### 🔄 3. Structured Error Migration (Gradual)

**Current State**: 9 `fmt.Errorf` vs 10 `NewParserError`

**Strategy**: Add shim for gradual migration

```go
// internal/parser/errors.go (~20 LOC)
package parser

import (
    "github.com/sunholo/ailang/internal/errors"
    "github.com/sunholo/ailang/internal/lexer"
)

// report creates a structured error with code, position, and fix suggestion
func (p *Parser) report(code, message, fixSuggestion string) error {
    return NewParserError(
        code,
        p.curPos(),
        p.curToken,
        message,
        nil,
        fixSuggestion,
    )
}

// reportExpected creates an error for unexpected token with expected types
func (p *Parser) reportExpected(expected []lexer.TokenType, got lexer.TokenType) error {
    return NewParserError(
        "PAR001_UNEXPECTED_TOKEN",
        p.curPos(),
        p.curToken,
        fmt.Sprintf("expected %v, got %s", expected, got),
        expected,
        fmt.Sprintf("Use %s instead", expected[0]),
    )
}
```

**Migration approach**:
1. Replace `peekError()` and `noPrefixParseFnError()` with structured versions
2. Migrate high-traffic paths first (literals, expressions)
3. Leave low-frequency paths for opportunistic migration
4. Target: 80%+ structured errors by end of M-P2

### ✅ 4. Documentation (DONE)

- ✅ `docs/parser-guarantees.md` created (333 lines)
- ✅ README.md updated with parser stats
- ✅ M-P1.md comprehensive summary

---

## M-P2: ADT + Tuple Implementation

### Scope

**In Scope**:
- ✅ Sum types: `type Option[a] = Some(a) | None`
- ✅ Product types: `type Point = {x: float, y: float}`
- ✅ Recursive types: `type Tree[a] = Leaf(a) | Branch(Tree[a], Tree[a])`
- ✅ Tuple types: `(int, string, bool)`
- ✅ Tuple literals: `(1, "hello", true)`
- ✅ Type aliases: `type UserId = int`

**Out of Scope** (Deferred to M-P3):
- ❌ Class declarations: `class Show a where...`
- ❌ Instance declarations: `instance Show Int where...`
- ❌ Pattern matching on ADTs (parser only, semantics in type checker)
- ❌ GADT syntax
- ❌ Type class constraints in ADTs

**Why defer class/instance?**
- Type declarations are simpler (no method signatures)
- Can achieve 80% coverage with types alone
- Class/instance parsing more complex, better as separate milestone

---

## Day 1: AST Nodes + Printers (≈150 LOC)

**Goal**: Define AST structure and deterministic printing BEFORE implementing parser

**Why printers first?**
- Golden files depend on stable printer output
- Easier to review AST design when you can see serialized form
- Prevents "implement then rewrite printer" cycle

### Tasks

#### 1. AST Type Declarations (~80 LOC)

```go
// internal/ast/types.go (additions)

// TypeDecl represents a type declaration
type TypeDecl struct {
    Name       string        // e.g., "Option"
    TypeParams []string      // e.g., ["a"]
    Body       TypeDeclBody  // Sum, Product, or Alias
    IsExport   bool
    Pos        Pos
    Origin     string
}

// TypeDeclBody is the RHS of a type declaration
type TypeDeclBody interface {
    typeDeclBody()
}

// TypeAlias: type UserId = int
type TypeAlias struct {
    Type Type
}

// TypeSum: type Option[a] = Some(a) | None
type TypeSum struct {
    Variants []*TypeVariant
}

type TypeVariant struct {
    Name   string
    Fields []Type  // Constructor arguments
    Pos    Pos
}

// TypeProduct: type Point = {x: float, y: float}
type TypeProduct struct {
    Fields []*TypeField
}

type TypeField struct {
    Name string
    Type Type
    Pos  Pos
}

// TupleType: (int, string, bool)
type TupleType struct {
    Elements []Type
    Pos      Pos
}

// TupleLiteral: (1, "hello", true)
type TupleLiteral struct {
    Elements []Expr
    Pos      Pos
}
```

#### 2. Update AST Printer (~70 LOC)

```go
// internal/ast/print.go (additions)

func (p *astPrinter) printTypeDecl(td *TypeDecl) interface{} {
    result := map[string]interface{}{
        "type": "TypeDecl",
        "name": td.Name,
    }

    if len(td.TypeParams) > 0 {
        result["type_params"] = td.TypeParams
    }

    if td.IsExport {
        result["export"] = true
    }

    result["body"] = p.printTypeDeclBody(td.Body)
    return result
}

func (p *astPrinter) printTypeDeclBody(body TypeDeclBody) interface{} {
    switch b := body.(type) {
    case *TypeAlias:
        return map[string]interface{}{
            "type": "TypeAlias",
            "rhs":  p.printType(b.Type),
        }
    case *TypeSum:
        variants := make([]interface{}, len(b.Variants))
        for i, v := range b.Variants {
            variants[i] = p.printTypeVariant(v)
        }
        return map[string]interface{}{
            "type":     "TypeSum",
            "variants": variants,
        }
    case *TypeProduct:
        fields := make([]interface{}, len(b.Fields))
        for i, f := range b.Fields {
            fields[i] = map[string]interface{}{
                "name": f.Name,
                "type": p.printType(f.Type),
            }
        }
        return map[string]interface{}{
            "type":   "TypeProduct",
            "fields": fields,
        }
    }
    return nil
}

// Add to expression printer
func (p *astPrinter) printTupleLiteral(tl *TupleLiteral) interface{} {
    elements := make([]interface{}, len(tl.Elements))
    for i, e := range tl.Elements {
        elements[i] = p.simplify(e)
    }
    return map[string]interface{}{
        "type":     "TupleLiteral",
        "elements": elements,
    }
}
```

### Deliverables

- ✅ AST nodes defined
- ✅ Printer updated
- ✅ Manual smoke test (create AST node, print it, verify JSON)
- ✅ No parser changes yet

---

## Day 2: Type Declaration Parsing (≈100 LOC)

**Goal**: Implement `parseTypeDeclaration()` with golden tests

### Parser Implementation (~60 LOC)

```go
// internal/parser/parser.go

func (p *Parser) parseTypeDeclaration() ast.Node {
    startPos := p.curPos()
    isExport := false

    // Handle export prefix
    if p.curTokenIs(lexer.EXPORT) {
        isExport = true
        p.nextToken()
    }

    // Expect 'type'
    if !p.curTokenIs(lexer.TYPE) {
        p.reportExpected([]lexer.TokenType{lexer.TYPE}, p.curToken.Type)
        return nil
    }

    p.nextToken() // consume 'type'

    // Type name
    if !p.curTokenIs(lexer.IDENT) {
        return p.report("PAR_TYPE_NO_NAME", "type declaration requires a name", "Add a type name")
    }

    typeName := p.curToken.Literal
    p.nextToken()

    // Type parameters [a, b]
    var typeParams []string
    if p.curTokenIs(lexer.LBRACKET) {
        typeParams = p.parseTypeParams()
    }

    // Expect '='
    if !p.curTokenIs(lexer.ASSIGN) {
        return p.report("PAR_TYPE_NO_BODY", "type declaration requires '='", "Add '= <type>'")
    }
    p.nextToken()

    // Parse body (alias, sum, or product)
    body := p.parseTypeDeclBody()

    return &ast.TypeDecl{
        Name:       typeName,
        TypeParams: typeParams,
        Body:       body,
        IsExport:   isExport,
        Pos:        startPos,
        Origin:     "type_decl",
    }
}

func (p *Parser) parseTypeDeclBody() ast.TypeDeclBody {
    // Record type: { fields }
    if p.curTokenIs(lexer.LBRACE) {
        return p.parseProductType()
    }

    // Parse first type/variant
    first := p.parseType()

    // Check for | (sum type)
    if p.curTokenIs(lexer.PIPE) {
        return p.parseSumType(first)
    }

    // Type alias
    return &ast.TypeAlias{Type: first}
}
```

### Golden Tests (~40 LOC)

```go
// internal/parser/type_test.go (update existing)

func TestTypeDeclarations(t *testing.T) {
    // Unmark t.Skip() - types now implemented!

    tests := []struct {
        name   string
        input  string
        golden string
    }{
        {"simple_alias", "type UserId = int", "type/simple_alias"},
        {"generic_alias", "type Box[a] = a", "type/generic_alias"},
        {"sum_type", "type Option[a] = Some(a) | None", "type/sum_type"},
        {"product_type", "type Point = {x: int, y: int}", "type/product_type"},
        {"recursive", "type List[a] = Nil | Cons(a, List[a])", "type/recursive"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            output := parseAndPrint(t, tt.input)
            goldenCompare(t, tt.golden, output)
        })
    }
}
```

### Deliverables

- ✅ `parseTypeDeclaration()` implemented
- ✅ Alias, sum, and product types working
- ✅ ~20 golden tests passing
- ✅ Coverage: ~72-75%

---

## Day 3: Tuple Types + Literals (≈80 LOC)

**Goal**: Add tuple support and update existing tests

### Parser Updates (~50 LOC)

```go
// Tuple type parsing
func (p *Parser) parseTupleOrType() ast.Type {
    if !p.curTokenIs(lexer.LPAREN) {
        return p.parseType()
    }

    startPos := p.curPos()
    p.nextToken() // consume '('

    // Empty tuple: ()
    if p.curTokenIs(lexer.RPAREN) {
        p.nextToken()
        return &ast.UnitType{Pos: startPos}
    }

    // Parse first element
    first := p.parseType()

    // Check for comma (tuple) or closing paren (grouped type)
    if !p.curTokenIs(lexer.COMMA) {
        p.expectPeek(lexer.RPAREN)
        return first // Grouped type: (int)
    }

    // Tuple type
    elements := []ast.Type{first}
    for p.curTokenIs(lexer.COMMA) {
        p.nextToken()
        if p.curTokenIs(lexer.RPAREN) {
            break // Trailing comma
        }
        elements = append(elements, p.parseType())
    }

    p.expectPeek(lexer.RPAREN)
    return &ast.TupleType{Elements: elements, Pos: startPos}
}

// Tuple literal parsing (update parseGroupedExpression)
func (p *Parser) parseGroupedOrTuple() ast.Expr {
    startPos := p.curPos()
    p.nextToken() // consume '('

    // Empty: ()
    if p.curTokenIs(lexer.RPAREN) {
        p.nextToken()
        return &ast.Literal{Kind: "Unit", Pos: startPos}
    }

    // Parse first element
    first := p.parseExpression(LOWEST)

    // Single expression
    if !p.curTokenIs(lexer.COMMA) {
        p.expectPeek(lexer.RPAREN)
        return first // Grouped: (expr)
    }

    // Tuple
    elements := []ast.Expr{first}
    for p.curTokenIs(lexer.COMMA) {
        p.nextToken()
        if p.curTokenIs(lexer.RPAREN) {
            break // Trailing comma
        }
        elements = append(elements, p.parseExpression(LOWEST))
    }

    p.expectPeek(lexer.RPAREN)
    return &ast.TupleLiteral{Elements: elements, Pos: startPos}
}
```

### Golden Tests (~30 LOC)

```go
func TestTuples(t *testing.T) {
    tests := []struct {
        name   string
        input  string
        golden string
    }{
        {"tuple_literal_2", "(1, 2)", "tuple/literal_2"},
        {"tuple_literal_3", "(1, \"hi\", true)", "tuple/literal_3"},
        {"nested_tuple", "((1, 2), (3, 4))", "tuple/nested"},
        {"tuple_type", "(int, string)", "tuple/type_2"},
        {"triple_type", "(int, string, bool)", "tuple/type_3"},
        {"empty_tuple", "()", "tuple/empty"},
    }
    // ...
}
```

### Deliverables

- ✅ Tuple types and literals working
- ✅ ~15 golden tests added
- ✅ Coverage: ~75-78%

---

## Day 4: Error Recovery + Negative Tests (≈50 LOC)

**Goal**: Add error handling for malformed type declarations

### Error Tests (~50 LOC)

```go
func TestTypeErrors(t *testing.T) {
    tests := []struct {
        name  string
        input string
    }{
        {"type_no_name", "type = int"},
        {"type_no_body", "type Foo"},
        {"type_trailing_pipe", "type Color = Red | Green |"},
        {"type_empty_sum", "type Empty = | |"},
        {"type_invalid_variant", "type Foo = 123"},
        {"export_underscore_type", "export type _Private = int"},
        {"tuple_no_close", "(1, 2"},
        {"tuple_type_no_close", "(int, string"},
        {"nested_type_error", "type Foo = Bar[Baz["},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            errs := mustParseError(t, tt.input)
            if len(errs) == 0 {
                t.Error("Expected parse errors but got none")
            }
        })
    }
}

func TestTypeRecovery(t *testing.T) {
    // Parser should recover and report multiple type errors
    input := `
        type BadType1 =
        type GoodType = int
        type BadType2 = |
    `

    errs := mustParseError(t, input)
    if len(errs) < 2 {
        t.Errorf("Expected multiple errors, got %d", len(errs))
    }
}
```

### Deliverables

- ✅ ~15 error tests added
- ✅ Error recovery validated
- ✅ All parser tests still passing

---

## Day 5: Coverage Push + Polish (≈20 LOC + polish)

**Goal**: Push coverage to 80%+, final polish

### Tasks

#### 1. Coverage Analysis

```bash
make cover-branch  # Open HTML report
```

Identify uncovered branches:
- Type parameter edge cases
- Nested type parsing
- Error recovery paths

#### 2. Add Targeted Tests (~20 LOC)

```go
func TestTypeEdgeCases(t *testing.T) {
    tests := []struct {
        name   string
        input  string
        golden string
    }{
        {"deeply_nested", "type T = (((int)))", "type/deeply_nested"},
        {"many_type_params", "type Foo[a,b,c,d] = a", "type/many_params"},
        {"long_variant_list", "type X = A|B|C|D|E|F", "type/long_variants"},
        {"mixed_constructors", "type T = A | B(int) | C(int, string)", "type/mixed"},
    }
    // ...
}
```

#### 3. Update CI Coverage Gate

```yaml
# .github/workflows/ci.yml (update)
- name: Check parser coverage
  run: |
    COVERAGE=$(make cover-lines)
    echo "Parser coverage: $COVERAGE"
    COVERAGE_NUM=$(echo $COVERAGE | sed 's/%//')
    if (( $(echo "$COVERAGE_NUM < 80.0" | bc -l) )); then
      echo "❌ Parser coverage $COVERAGE is below 80% threshold"
      exit 1
    fi
    echo "✅ Parser coverage $COVERAGE meets 80% threshold"
```

#### 4. Documentation Updates

Update `docs/parser-guarantees.md`:
- Add ADT syntax to "What the Parser Guarantees"
- Update coverage numbers
- Add tuple documentation
- Update "What's NOT Guaranteed" to remove ADTs

Update `README.md`:
- Update parser coverage to 80%+
- Update test count (~550+ tests)
- Add ADT support to feature list

### Deliverables

- ✅ Coverage: ≥80%
- ✅ All tests passing
- ✅ Documentation updated
- ✅ CI gate raised to 80%

---

## Success Criteria

### Quantitative

- [ ] **Coverage**: ≥80% line coverage (up from 70.2%)
- [ ] **Tests**: ~550+ tests (up from 506)
- [ ] **Golden Files**: ~140+ golden files (up from 116)
- [ ] **Fuzz**: Still 0 panics on 52k+ executions
- [ ] **CI**: Green with 80% coverage gate

### Qualitative

- [ ] **Types Work**: All ADT forms parse correctly
- [ ] **Tuples Work**: Types and literals both parse
- [ ] **Errors Structured**: 80%+ errors use `NewParserError`
- [ ] **No Regressions**: All M-P1 tests still pass
- [ ] **Documentation**: Parser guarantees updated

---

## Feature Flags (Optional)

If you want to land incrementally:

```go
// internal/parser/parser.go
var enableADT = os.Getenv("AILANG_ENABLE_ADT") == "true"

func (p *Parser) parseTypeDeclaration() ast.Node {
    if !enableADT {
        return p.report("PAR_FEATURE_DISABLED",
            "type declarations require --feature adt",
            "Set AILANG_ENABLE_ADT=true")
    }
    // ... rest of implementation
}
```

Benefits:
- Land code without breaking golden tests
- Test in isolation
- Gradual rollout

Downsides:
- More complexity
- Need to remember to remove flag

**Recommendation**: Don't use feature flags for M-P2. M-P1 baseline is solid enough.

---

## Risks & Mitigations

### Risk 1: Golden File Churn

**Problem**: 35+ new type test files might conflict with existing tests

**Mitigation**:
- Create `testdata/parser/types/` subdirectory
- Separate type tests from expression tests
- Use `-update` flag carefully

### Risk 2: Coverage Not Reaching 80%

**Problem**: Type parsing might not add enough coverage

**Mitigation**:
- Target coverage explicitly on Day 5
- Add edge case tests for uncovered branches
- Accept 78-79% if necessary (document why)

### Risk 3: Parser Performance

**Problem**: Tuple/type parsing might slow down parser

**Mitigation**:
- Benchmark before/after: `go test -bench=. ./internal/parser`
- Expect <5% slowdown
- Profile if >5%: `go test -cpuprofile=cpu.out`

---

## Post-M-P2: What's Next (M-P3)

After M-P2, consider:

### M-P3: Class + Instance Declarations

**Scope**:
- `class Show a where show :: a -> string`
- `instance Show Int where show = showInt`
- Method signatures
- Superclass constraints

**Estimated**: 5 days, ~300 LOC

### M-P4: Pattern Matching

**Scope**:
- Match expressions with ADT patterns
- Exhaustiveness checking (warning, not error)
- Nested patterns
- Guards

**Estimated**: 7 days, ~500 LOC

### M-P5: Effect System Parsing

**Scope**:
- Effect declarations: `effect IO { ... }`
- Handler syntax
- Effect polymorphism

**Estimated**: 5 days, ~400 LOC

---

## Summary

M-P2 takes the solid M-P1 baseline (70.2%, 506 tests) and adds:
- Algebraic Data Types (sum, product, recursive)
- Tuple types and literals
- Structured error migration (80%+)
- BOM/NFC normalization
- Coverage push to 80%+

**Total Effort**: 5 days, ~400 LOC
**Expected Outcome**: Production-ready type system parsing with 80%+ coverage

**Key Principle**: Printers first, parser second, tests throughout. Keep CI green.

---

**Status**: 📋 Ready to Execute
**Prerequisites**: ✅ M-P1 Complete (baseline frozen)
**Next Step**: Close M-P1 gaps (BOM normalization, error migration)
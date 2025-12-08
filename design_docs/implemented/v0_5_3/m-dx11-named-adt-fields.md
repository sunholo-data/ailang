# M-DX11: Named ADT Constructor Fields for Go Codegen

## Status
**Planned** for v0.5.3

## Problem Statement

When AILANG generates Go code for ADT variants with payloads, it uses positional field names (`Value0`, `Value1`, etc.) instead of the names from the AILANG source. This hurts developer experience significantly.

**AILANG source:**
```ailang
type DrawCmd =
    | Rect(x: float, y: float, w: float, h: float, color: int, z: int)
    | Sprite(id: int, x: float, y: float, z: int)
```

**Current generated Go:**
```go
type DrawCmdRect struct {
    Value0 float64  // x
    Value1 float64  // y
    Value2 float64  // w
    Value3 float64  // h
    Value4 int64    // color
    Value5 int64    // z
}
```

**Desired generated Go:**
```go
type DrawCmdRect struct {
    X     float64
    Y     float64
    W     float64
    H     float64
    Color int64
    Z     int64
}
```

**Impact:**
- Current: `draw(c.Value0, c.Value1, c.Value2, c.Value3)`
- Desired: `draw(c.X, c.Y, c.W, c.H)`

## Source

DX feedback from `stapledons_voyage` agent (2025-12-03).

## Root Cause Analysis

1. **AST doesn't capture field names**: `ast.Constructor.Fields` is `[]Type`, not `[]*ConstructorField`
2. **Parser doesn't parse named syntax**: `parseVariant()` calls `parseType()` directly without handling `name: type` syntax
3. **Codegen uses positional names**: `adt.go` generates `Value0`, `Value1`, etc.

## Design

### Phase 1: AST Changes (~30 LOC)

Add named field support to the AST:

```go
// internal/ast/ast_decl.go

// ConstructorField represents a named field in an ADT constructor.
type ConstructorField struct {
    Name string  // Field name (may be empty for positional)
    Type Type    // Field type
    Pos  Pos
}

type Constructor struct {
    Name   string
    Fields []*ConstructorField  // Changed from []Type
    Pos    Pos
}
```

### Phase 2: Parser Changes (~50 LOC)

Modify `parseVariant()` to handle both syntaxes:

```go
// internal/parser/parser_type.go

func (p *Parser) parseConstructorField() *ast.ConstructorField {
    pos := p.curPos()

    // Check for named field syntax: name: type
    if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON) {
        name := p.curToken.Literal
        p.nextToken() // consume IDENT
        p.nextToken() // consume COLON
        typ := p.parseType()
        return &ast.ConstructorField{Name: name, Type: typ, Pos: pos}
    }

    // Positional field (type only)
    typ := p.parseType()
    return &ast.ConstructorField{Name: "", Type: typ, Pos: pos}
}
```

### Phase 3: Codegen Changes (~40 LOC)

Modify `adt.go` to use field names when available:

```go
// internal/gen/golang/adt.go

func (g *ADTGenerator) generateVariantFields(ctor *ast.Constructor) {
    for i, field := range ctor.Fields {
        var name string
        if field.Name != "" {
            name = ToPascalCase(field.Name)  // Use named field
        } else {
            name = fmt.Sprintf("Value%d", i)  // Fallback to positional
        }
        goType := g.mapASTType(field.Type)
        g.writef("%s %s\n", name, goType)
    }
}
```

### Phase 4: Pattern Matching Updates (~30 LOC)

Update `codegen_match.go` to use field names in pattern matching:

```go
// internal/gen/golang/codegen_match.go

func (g *Generator) getFieldAccess(ctor *ast.Constructor, idx int) string {
    if idx < len(ctor.Fields) && ctor.Fields[idx].Name != "" {
        return ToPascalCase(ctor.Fields[idx].Name)
    }
    return fmt.Sprintf("Value%d", idx)
}
```

## Backwards Compatibility

- **Named syntax is opt-in**: Existing `Rect(int, int)` syntax continues to work with `Value0`, `Value1`
- **No runtime changes**: Only affects Go code generation
- **Gradual adoption**: Users can add names to their ADTs incrementally

## Estimated Effort

| Phase | LOC | Time |
|-------|-----|------|
| AST changes | ~30 | 20min |
| Parser changes | ~50 | 40min |
| Codegen changes | ~40 | 30min |
| Pattern matching | ~30 | 30min |
| Tests | ~100 | 40min |
| **Total** | ~250 | 2.5h |

## Acceptance Criteria

1. [ ] Named constructor syntax parses: `Rect(x: float, y: float)`
2. [ ] Generated Go uses field names: `X float64`, `Y float64`
3. [ ] Pattern matching works with named fields
4. [ ] Positional syntax still works (backwards compatible)
5. [ ] All existing tests pass
6. [ ] New tests for named field syntax

## Test Cases

```ailang
-- Named fields
type Point = { x: int, y: int }
type DrawCmd =
    | Rect(x: float, y: float, w: float, h: float)
    | Circle(cx: float, cy: float, r: float)
    | Text(msg: string, x: float, y: float)

-- Should generate:
-- DrawCmdRect { X, Y, W, H float64 }
-- DrawCmdCircle { Cx, Cy, R float64 }
-- DrawCmdText { Msg string; X, Y float64 }
```

## Related Issues

**Not addressed in this design doc:**
- Issue 2: List fields in records generate as `interface{}` instead of typed slices
  - This requires runtime changes (evaluator uses `[]interface{}` internally)
  - Planned for future milestone (M-GAME-C or later)

## Files to Modify

1. `internal/ast/ast_decl.go` - Add ConstructorField struct
2. `internal/parser/parser_type.go` - Parse named field syntax
3. `internal/gen/golang/adt.go` - Generate named field names
4. `internal/gen/golang/codegen_match.go` - Use names in pattern matching
5. `internal/gen/golang/adt_test.go` - Add tests for named fields

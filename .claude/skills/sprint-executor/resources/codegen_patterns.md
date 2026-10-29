# Code Generation Patterns

**Legacy reference for the Go source backend. Not required for new work.**

## Status: generated Go is not a requirement

Go source emission (`ailang compile --emit-go`) is demoted to a diagnostic projection and emit-go-v2 is frozen (Mark, 2026-07-14; see `design_docs/v1-mission.md`).

- New builtins, stdlib functions and language features do **not** need a `GoCodegenSpec`,
  a `registry_codegen_*.go` entry, or a generated-Go parity test.
- Engine parity means the **evaluator and the strict VM** agree. Do not add
  "generated Go agrees" acceptance criteria to sprint plans, and do not build emit-go
  harnesses to satisfy one.
- Use this file only when the sprint explicitly scopes a fix to existing codegen
  (`internal/gen/*`) or the contracts projection (`--verify-contracts`).

## File Organization

```
internal/gen/golang/
├── codegen.go             # Main generator, Generate(), ADT/record registration
├── codegen_decl.go        # Declaration generation (functions, let bindings)
├── codegen_expr.go        # Main expression dispatch (generateExpr)
├── codegen_expr_simple.go # Literals, Var, VarGlobal, Lambda, builtin mappings
├── codegen_expr_app.go    # Function application (App)
├── codegen_expr_let.go    # Let/LetRec bindings
├── codegen_expr_control.go# If expressions, block expressions
├── codegen_ops.go         # BinOp, UnOp, Record, List, Tuple, DictAbs
├── codegen_match.go       # Pattern matching (Match expressions)
├── codegen_block.go       # Block expression handling
├── codegen_runtime.go     # Entry point for runtime helpers
├── codegen_runtime_arith.go    # Arithmetic helpers (Add, Sub, etc.)
├── codegen_runtime_collections.go # List/array helpers
├── codegen_runtime_records.go  # Record helpers (FieldGet, RecordUpdate)
├── codegen_runtime_misc.go     # Function calling, misc helpers
├── types.go               # TypeMapper, AILANG → Go type conversions
├── naming.go              # Go naming conventions (ToGoVarName, ToPascalCase)
├── effects.go             # Effect handler registration
├── adt.go                 # ADT struct generation
```

## Common Workflows

### 1. Adding a Builtin Function Mapping

**When:** Adding a new builtin that compiles to Go code (not interpreted).

**Location:** `codegen_expr_simple.go`

**Example - Adding a math function:**

```go
// In mapPureMathBuiltin() function:
func (g *Generator) mapPureMathBuiltin(name string) string {
    mathMappings := map[string]string{
        // ... existing mappings ...

        // NEW: Add hyperbolic functions
        "_math_sinh": "math.Sinh",
        "_math_cosh": "math.Cosh",
        "_math_tanh": "math.Tanh",
        "sinh":       "math.Sinh",  // stdlib wrapper
        "cosh":       "math.Cosh",
        "tanh":       "math.Tanh",
    }

    if expr, ok := mathMappings[name]; ok {
        g.needsMathImport = true  // Mark math import needed
        return expr
    }
    return ""
}
```

**Example - Adding an effect builtin:**

```go
// In mapEffectBuiltinToHandler() function:
func mapEffectBuiltinToHandler(name string) string {
    effectMappings := map[string]string{
        // ... existing mappings ...

        // NEW: Add database effect
        "_db_query":   "requireDB().Query",
        "_db_execute": "requireDB().Execute",
        "db_query":    "requireDB().Query",   // stdlib wrapper
        "db_execute":  "requireDB().Execute",
    }
    return effectMappings[name]
}
```

**Checklist:**
- [ ] Add mapping in appropriate function (`mapPureMathBuiltin` or `mapEffectBuiltinToHandler`)
- [ ] Add both `_builtin_name` (internal) and `builtin_name` (stdlib wrapper) variants
- [ ] Set import flags if needed (`g.needsMathImport = true`)
- [ ] Add tests in `codegen_*_test.go`

### 2. Adding a New Core Expression Type

**When:** New AST node type that needs code generation.

**Location:** `codegen_expr.go` (dispatch) + dedicated handler file

**Step 1: Add case to generateExpr dispatch:**

```go
// In codegen_expr.go:
func (g *Generator) generateExpr(expr core.Expr) error {
    switch e := expr.(type) {
    // ... existing cases ...

    case *core.NewExprType:
        return g.generateNewExprType(e)

    default:
        return fmt.Errorf("unsupported expression type: %T", expr)
    }
}
```

**Step 2: Implement the handler:**

```go
// In appropriate file (codegen_expr_*.go):
func (g *Generator) generateNewExprType(e *core.NewExprType) error {
    // Generate Go code for this expression
    g.writef("/* generated code for NewExprType */")
    return nil
}
```

**Checklist:**
- [ ] Add case to `generateExpr()` switch
- [ ] Implement handler function
- [ ] Handle nested expressions recursively (call `g.generateExpr()`)
- [ ] Manage indentation (`g.indent++`, `g.indent--`)
- [ ] Add comprehensive tests

### 3. Adding Runtime Helpers

**When:** Generated code needs supporting Go functions at runtime.

**Location:** `codegen_runtime_*.go` files

**Step 1: Add to writeRuntimeHelpers dispatch:**

```go
// In codegen_runtime.go:
func (g *Generator) writeRuntimeHelpers() {
    g.writeRuntimeRecordHelpers()
    g.writeRuntimeListHelpers()
    g.writeRuntimeArithmeticHelpers()
    g.writeRuntimeMiscHelpers()
    g.writeRuntimeSliceConverters()
    g.writeArrayRuntimeFunctions()
    g.writeADTSliceConverters()

    // NEW: Add custom helpers
    g.writeNewFeatureHelpers()
}
```

**Step 2: Implement helper generator:**

```go
// In codegen_runtime_misc.go (or new file):
func (g *Generator) writeNewFeatureHelpers() {
    g.writef(`
// NewFeatureHelper does something useful
func NewFeatureHelper(x interface{}) interface{} {
    // Implementation
    return x
}
`)
}
```

**Checklist:**
- [ ] Add call in `writeRuntimeHelpers()`
- [ ] Implement helper generator function
- [ ] Use `g.writef()` with backtick strings for multi-line
- [ ] Consider `skipRuntimeHelpers` flag for multi-file compilation
- [ ] Add tests that verify generated helper code

### 4. Adding Type Mappings

**When:** New AILANG type needs Go representation.

**Location:** `types.go`

**Example:**

```go
// In TypeMapper.MapType():
func (tm *TypeMapper) MapType(t types.Type) string {
    switch ty := t.(type) {
    // ... existing cases ...

    case *types.TNewType:
        return "GoNewType"

    default:
        return "interface{}"
    }
}
```

**Checklist:**
- [ ] Add case to `MapType()` switch
- [ ] Consider pointer vs value semantics
- [ ] Update `GoTypeToAILANG()` for reverse mapping if needed
- [ ] Add tests in `types_test.go`

## Testing Patterns

### Basic Codegen Test

```go
// In codegen_*_test.go:
func TestGenerateNewFeature(t *testing.T) {
    // Build Core AST
    prog := &core.Program{
        Decls: []core.Decl{
            &core.FuncDecl{
                Name: "testFunc",
                Body: &core.NewExprType{...},
            },
        },
    }

    // Generate code
    gen := New("main")
    code, err := gen.Generate(prog)
    if err != nil {
        t.Fatalf("Generate failed: %v", err)
    }

    // Verify output contains expected Go code
    codeStr := string(code)
    if !strings.Contains(codeStr, "expectedGoCode") {
        t.Errorf("Missing expected code.\nGot:\n%s", codeStr)
    }
}
```

### Integration Test (Compile + Run)

```go
func TestNewFeatureIntegration(t *testing.T) {
    src := `
module test
let result = newFeature(42)
`
    // Parse → Elaborate → TypeCheck → Generate → Compile → Run
    // See existing integration tests for patterns
}
```

## Common Pitfalls

### 1. Forgetting Interface{} Type Assertions

**Problem:** Generated code uses `interface{}` but Go operations need concrete types.

```go
// ❌ WRONG - Won't compile
func Add(a, b interface{}) interface{} {
    return a + b  // Can't add interface{} values!
}

// ✅ CORRECT - Type assert first
func Add(a, b interface{}) interface{} {
    return a.(int64) + b.(int64)
}
```

### 2. Missing Import Tracking

**Problem:** Generated code uses stdlib but import not included.

```go
// ❌ WRONG - Forgets to track import
func (g *Generator) mapSomething(name string) string {
    return "math.Sqrt"  // But math not imported!
}

// ✅ CORRECT - Track import
func (g *Generator) mapSomething(name string) string {
    g.needsMathImport = true  // Mark import needed
    return "math.Sqrt"
}
```

### 3. Indentation Mismatch

**Problem:** Generated code has wrong indentation.

```go
// ❌ WRONG - Forgot to decrement
g.indent++
g.generateExpr(body)
// Missing g.indent--!

// ✅ CORRECT - Always balance
g.indent++
g.generateExpr(body)
g.indent--
```

### 4. Not Handling Nested Expressions

**Problem:** Only generates outer expression, not inner.

```go
// ❌ WRONG - Just writes literal
func (g *Generator) generateIf(e *core.If) error {
    g.writef("if %v { ... }", e.Cond)  // Cond not generated!
}

// ✅ CORRECT - Generate nested expression
func (g *Generator) generateIf(e *core.If) error {
    g.writef("if ")
    if err := g.generateExpr(e.Cond); err != nil {
        return err
    }
    g.writef(" { ... }")
}
```

## Quick Reference: Key Functions

| Function | Purpose | Location |
|----------|---------|----------|
| `generateExpr()` | Main expression dispatch | codegen_expr.go |
| `generateDecl()` | Declaration generation | codegen_decl.go |
| `mapEffectBuiltinToHandler()` | Effect builtin → handler | codegen_expr_simple.go |
| `mapPureMathBuiltin()` | Math builtin → Go math | codegen_expr_simple.go |
| `writeRuntimeHelpers()` | Runtime helper dispatch | codegen_runtime.go |
| `MapType()` | AILANG type → Go type | types.go |
| `ToGoVarName()` | AILANG name → Go var name | naming.go |
| `ToPascalCase()` | Convert to exported name | naming.go |

## Debugging

```bash
# View generated code without formatting
go test -v -run TestMyFeature ./internal/gen/golang/

# Check type mapper
make doc PKG=internal/gen/golang | grep -A5 "TypeMapper"

# Trace generation with verbose output
# Add debug prints in generateExpr() temporarily
```

## Related Resources

- **Builtin Developer Skill**: [`../.claude/skills/builtin-developer/SKILL.md`](../../builtin-developer/SKILL.md) - For adding builtins to interpreter
- **Parser Patterns**: [`parser_patterns.md`](parser_patterns.md) - For AST changes
- **API Patterns**: [`api_patterns.md`](api_patterns.md) - Common constructor signatures

# M-R5b: Record Extension & Update Syntax

**Status**: 📋 Planned for v0.3.1
**Created**: October 5, 2025
**Priority**: HIGH
**Depends on**: M-R5 Records & Row Polymorphism (✅ v0.3.0-alpha3)
**Estimated Effort**: 4-5 days (~850 LOC)

## Overview

Add syntactic sugar for common record operations: extension (add fields), restriction (remove fields), and update (change field values). These operations are common in functional programming and make record manipulation more ergonomic.

## Motivation

### Current Pain Point
```ailang
-- Want to add a field to a record
let person = {name: "Alice", age: 30}
-- Currently: Must manually create new record with all fields
let employee = {name: person.name, age: person.age, id: 100, dept: "Engineering"}
-- This is verbose, error-prone, and doesn't scale
```

### With Extension Syntax
```ailang
-- Much cleaner!
let employee = {person | id: 100, dept: "Engineering"}
-- Result: {name: "Alice", age: 30, id: 100, dept: "Engineering"}
```

### Use Cases

1. **Adding metadata** - Add tracking fields to existing records
2. **Configuration merging** - Combine default config with overrides
3. **API responses** - Add computed fields to database records
4. **Type narrowing** - Add fields to satisfy stricter type requirements
5. **Incremental construction** - Build records step by step

## Syntax Design

### 1. Record Extension (Add Fields)

**Syntax**: `{record | field1: value1, field2: value2, ...}`

```ailang
let base = {x: 1, y: 2}
let extended = {base | z: 3, w: 4}
-- Result: {x: 1, y: 2, z: 3, w: 4}

-- With shadowing (new value overrides)
let updated = {base | x: 10}
-- Result: {x: 10, y: 2}  -- x is overridden
```

**Type Rule**:
```
Γ ⊢ r : {τ₁, ..., τₙ | ρ}
Γ ⊢ v₁ : σ₁, ..., Γ ⊢ vₘ : σₘ
fields(r) ∩ {f₁, ..., fₘ} = ∅  (no overlap, or override allowed)
────────────────────────────────────────────────────────
Γ ⊢ {r | f₁: v₁, ..., fₘ: vₘ} : {τ₁, ..., τₙ, f₁: σ₁, ..., fₘ: σₘ | ρ}
```

**Edge Cases**:
- **Duplicate fields**: `{r | x: 1}` where `r` has `x` → Override allowed (like record update)
- **Empty extension**: `{r | }` → Just returns `r` (identity)
- **Multiple extensions**: `{r | x: 1 | y: 2}` → Syntax error (use `{r | x: 1, y: 2}`)

### 2. Record Restriction (Remove Fields)

**Syntax**: `{record - field1, field2, ...}`

```ailang
let person = {name: "Alice", age: 30, ssn: "123-45-6789"}
let public = {person - ssn}
-- Result: {name: "Alice", age: 30}

-- Remove multiple fields
let minimal = {person - age, ssn}
-- Result: {name: "Alice"}
```

**Type Rule**:
```
Γ ⊢ r : {f₁: τ₁, ..., fₙ: τₙ, g₁: σ₁, ..., gₘ: σₘ | ρ}
{f₁, ..., fₙ} ⊆ fields(r)
────────────────────────────────────────────────────────
Γ ⊢ {r - f₁, ..., fₙ} : {g₁: σ₁, ..., gₘ: σₘ | ρ}
```

**Edge Cases**:
- **Missing field**: `{r - nonexistent}` → Type error TC_REC_001
- **Remove all fields**: `{r - x, y}` where r = {x, y} → `{}` (empty record)
- **Restriction on open record**: `{r | ρ} - x` → `{r | ρ}` (cannot remove from unknown fields)

### 3. Record Update (Change Field Values)

**Syntax**: `{record with field1: value1, field2: value2, ...}`

```ailang
let person = {name: "Alice", age: 30}
let older = {person with age: 31}
-- Result: {name: "Alice", age: 31}

-- Update multiple fields
let renamed = {person with name: "Alicia", age: 31}
-- Result: {name: "Alicia", age: 31}
```

**Type Rule**:
```
Γ ⊢ r : {f₁: τ₁, ..., fₙ: τₙ | ρ}
Γ ⊢ v₁ : τ₁, ..., Γ ⊢ vₘ : τₘ
{f₁, ..., fₘ} ⊆ fields(r)
────────────────────────────────────────────────────────
Γ ⊢ {r with f₁: v₁, ..., fₘ: vₘ} : {f₁: τ₁, ..., fₙ: τₙ | ρ}
```

**Key Difference from Extension**:
- **Update**: Field *must* exist, type must match
- **Extension**: Field *may* exist (override), or be new

**Edge Cases**:
- **Missing field**: `{r with nonexistent: val}` → Type error TC_REC_001
- **Type mismatch**: `{r with age: "thirty"}` where age: int → Type error TC_REC_004
- **Empty update**: `{r with }` → Syntax error

### Comparison Table

| Operation | Syntax | Field Must Exist? | Type Must Match? | Result Type |
|-----------|--------|-------------------|------------------|-------------|
| Extension | `{r \| f: v}` | No | N/A (new field) | `{r, f: τ}` |
| Restriction | `{r - f}` | Yes | N/A | `{r without f}` |
| Update | `{r with f: v}` | Yes | Yes | `{r}` (same) |

## Implementation Plan

### Phase 1: Parser & AST (Day 1, ~150 LOC, 6 hours)

**1.1: Add Keywords** (~10 LOC)
- Add `with` to keywords (extension/update already use existing `|` and `-`)
- Update lexer token tests

**1.2: AST Nodes** (~50 LOC)
```go
// internal/ast/ast.go

type RecordExtension struct {
    BaseNode
    Record Expr
    Fields map[string]Expr  // New/override fields
}

type RecordRestriction struct {
    BaseNode
    Record Expr
    Remove []string  // Fields to remove
}

type RecordUpdate struct {
    BaseNode
    Record Expr
    Updates map[string]Expr  // Fields to update (must exist)
}
```

**1.3: Parser Rules** (~90 LOC)
```go
// internal/parser/parser.go

// Primary expression parsing
func (p *Parser) parsePrimaryExpression() ast.Expr {
    // ... existing cases ...

    case token.LBRACE:
        p.advance()

        // Check for record operations
        if p.peek().Type == token.IDENT {
            next := p.peekAhead(1)

            switch next.Type {
            case token.PIPE:
                // Record extension: {r | x: 1}
                return p.parseRecordExtension()

            case token.MINUS:
                // Record restriction: {r - x}
                return p.parseRecordRestriction()

            case token.WITH:
                // Record update: {r with x: 1}
                return p.parseRecordUpdate()
            }
        }

        // Otherwise, regular record literal
        return p.parseRecordLiteral()
}

func (p *Parser) parseRecordExtension() *ast.RecordExtension {
    record := p.parseIdentifier()
    p.expectAndAdvance(token.PIPE)

    fields := make(map[string]ast.Expr)
    for {
        name := p.expectAndAdvance(token.IDENT).Literal
        p.expectAndAdvance(token.COLON)
        value := p.parseExpression()
        fields[name] = value

        if p.peek().Type != token.COMMA {
            break
        }
        p.advance() // consume comma
    }

    p.expectAndAdvance(token.RBRACE)
    return &ast.RecordExtension{Record: record, Fields: fields}
}

// Similar for parseRecordRestriction() and parseRecordUpdate()
```

**Tests**: Add 10-15 parser tests for each operation

### Phase 2: Elaboration (Day 2, ~200 LOC, 8 hours)

**2.1: Core AST Representation** (~50 LOC)
```go
// internal/core/core.go

type RecordExtension struct {
    CoreNode
    Record CoreExpr
    Fields map[string]CoreExpr
}

type RecordRestriction struct {
    CoreNode
    Record CoreExpr
    Remove []string
}

type RecordUpdate struct {
    CoreNode
    Record CoreExpr
    Updates map[string]CoreExpr
}
```

**2.2: Elaboration Logic** (~150 LOC)
```go
// internal/elaborate/elaborate.go

func (e *Elaborator) elaborateRecordExtension(ext *ast.RecordExtension) (core.CoreExpr, error) {
    // Elaborate base record
    recordCore, err := e.elaborate(ext.Record)
    if err != nil {
        return nil, err
    }

    // Elaborate extension fields
    fieldsCore := make(map[string]core.CoreExpr)
    for name, expr := range ext.Fields {
        fieldCore, err := e.elaborate(expr)
        if err != nil {
            return nil, err
        }
        fieldsCore[name] = fieldCore
    }

    return &core.RecordExtension{
        Record: recordCore,
        Fields: fieldsCore,
    }, nil
}

// Similar for restriction and update
```

**Tests**: Add 8-12 elaboration tests

### Phase 3: Type Checking (Day 2-3, ~250 LOC, 10 hours)

**3.1: Type Inference for Extension** (~100 LOC)
```go
// internal/types/typechecker_core.go

func (tc *CoreTypeChecker) inferRecordExtension(ctx *InferenceContext, ext *core.RecordExtension) (*typedast.TypedRecordExtension, *TypeEnv, error) {
    // Infer base record type
    recordNode, _, err := tc.inferCore(ctx, ext.Record)
    if err != nil {
        return nil, ctx.env, err
    }

    baseType := getType(recordNode)

    // Get base record fields and row variable
    var baseFields map[string]Type
    var rowVar Type

    switch bt := baseType.(type) {
    case *TRecord:
        baseFields = bt.Fields
        rowVar = bt.Row
    case *TRecord2:
        if bt.Row != nil {
            baseFields = bt.Row.Labels
            rowVar = bt.Row.Tail
        }
    case *TRecordOpen:
        baseFields = bt.Fields
        rowVar = bt.Row
    default:
        return nil, ctx.env, fmt.Errorf("cannot extend non-record type: %s", baseType.String())
    }

    // Infer extension field types
    extensionFields := make(map[string]Type)
    extensionNodes := make(map[string]typedast.TypedExpr)

    for name, fieldExpr := range ext.Fields {
        fieldNode, _, err := tc.inferCore(ctx, fieldExpr)
        if err != nil {
            return nil, ctx.env, err
        }

        extensionFields[name] = getType(fieldNode)
        extensionNodes[name] = fieldNode

        // Check for override (field exists in base)
        if baseType, exists := baseFields[name]; exists {
            // Override: new type must unify with old type
            var unifyErr error
            _, unifyErr = tc.unifier.Unify(extensionFields[name], baseType, make(Substitution))
            if unifyErr != nil {
                return nil, ctx.env, NewFieldTypeMismatchError(
                    name, baseType, extensionFields[name], ext.Span().String(),
                )
            }
        }
    }

    // Combine base and extension fields
    resultFields := make(map[string]Type)
    for name, typ := range baseFields {
        resultFields[name] = typ
    }
    for name, typ := range extensionFields {
        resultFields[name] = typ  // Override if exists
    }

    // Create result type
    var resultType Type
    if tc.useRecordsV2 {
        resultType = &TRecord2{
            Row: &Row{
                Kind:   RecordRow,
                Labels: resultFields,
                Tail:   rowVar,
            },
        }
    } else {
        resultType = &TRecord{
            Fields: resultFields,
            Row:    rowVar,
        }
    }

    return &typedast.TypedRecordExtension{
        TypedExpr: typedast.TypedExpr{
            NodeID:    ext.ID(),
            Span:      ext.Span(),
            Type:      resultType,
            EffectRow: getEffectRow(recordNode),
            Core:      ext,
        },
        Record: recordNode,
        Fields: extensionNodes,
    }, ctx.env, nil
}
```

**3.2: Type Inference for Restriction** (~80 LOC)
```go
func (tc *CoreTypeChecker) inferRecordRestriction(ctx *InferenceContext, rest *core.RecordRestriction) (*typedast.TypedRecordRestriction, *TypeEnv, error) {
    // Infer base record
    recordNode, _, err := tc.inferCore(ctx, rest.Record)
    if err != nil {
        return nil, ctx.env, err
    }

    baseType := getType(recordNode)

    // Get fields
    baseFields := getRecordFieldsFromType(baseType)

    // Check all removed fields exist
    for _, fieldName := range rest.Remove {
        if _, exists := baseFields[fieldName]; !exists {
            return nil, ctx.env, NewMissingFieldError(fieldName, baseType, rest.Span().String())
        }
    }

    // Create result fields (all except removed)
    resultFields := make(map[string]Type)
    for name, typ := range baseFields {
        if !contains(rest.Remove, name) {
            resultFields[name] = typ
        }
    }

    // Preserve row variable (can't remove from unknown fields)
    rowVar := getRowVariable(baseType)

    // Build result type
    var resultType Type
    if tc.useRecordsV2 {
        resultType = &TRecord2{Row: &Row{Kind: RecordRow, Labels: resultFields, Tail: rowVar}}
    } else {
        resultType = &TRecord{Fields: resultFields, Row: rowVar}
    }

    return &typedast.TypedRecordRestriction{
        TypedExpr: typedast.TypedExpr{
            NodeID:    rest.ID(),
            Span:      rest.Span(),
            Type:      resultType,
            EffectRow: getEffectRow(recordNode),
            Core:      rest,
        },
        Record: recordNode,
        Remove: rest.Remove,
    }, ctx.env, nil
}
```

**3.3: Type Inference for Update** (~70 LOC)
```go
func (tc *CoreTypeChecker) inferRecordUpdate(ctx *InferenceContext, upd *core.RecordUpdate) (*typedast.TypedRecordUpdate, *TypeEnv, error) {
    // Similar to extension, but:
    // 1. All updated fields MUST exist in base
    // 2. Types MUST match exactly (unify with existing type)
    // 3. Result type is same as base type

    // ... implementation similar to extension but stricter checks ...
}
```

**Tests**: Add 15-20 type checking tests

### Phase 4: Runtime Evaluation (Day 3-4, ~150 LOC, 6 hours)

**4.1: Record Value Operations** (~50 LOC)
```go
// internal/eval/value.go

// Extend creates new record with additional fields
func (r *RecordValue) Extend(fields map[string]Value) *RecordValue {
    result := make(map[string]Value)

    // Copy existing fields
    for name, val := range r.Fields {
        result[name] = val
    }

    // Add/override with new fields
    for name, val := range fields {
        result[name] = val
    }

    return &RecordValue{Fields: result}
}

// Restrict creates new record with fields removed
func (r *RecordValue) Restrict(remove []string) *RecordValue {
    result := make(map[string]Value)

    for name, val := range r.Fields {
        if !contains(remove, name) {
            result[name] = val
        }
    }

    return &RecordValue{Fields: result}
}

// Update creates new record with fields updated
func (r *RecordValue) Update(updates map[string]Value) *RecordValue {
    result := make(map[string]Value)

    // Copy all fields
    for name, val := range r.Fields {
        result[name] = val
    }

    // Override updated fields
    for name, val := range updates {
        result[name] = val  // Type checker ensures field exists
    }

    return &RecordValue{Fields: result}
}
```

**4.2: Evaluator Cases** (~100 LOC)
```go
// internal/eval/eval.go

func (e *Evaluator) Eval(expr Expr, env *Env) (Value, error) {
    switch ex := expr.(type) {
    // ... existing cases ...

    case *RecordExtension:
        // Evaluate base record
        recordVal, err := e.Eval(ex.Record, env)
        if err != nil {
            return nil, err
        }

        record, ok := recordVal.(*RecordValue)
        if !ok {
            return nil, fmt.Errorf("cannot extend non-record: %T", recordVal)
        }

        // Evaluate extension fields
        fields := make(map[string]Value)
        for name, fieldExpr := range ex.Fields {
            val, err := e.Eval(fieldExpr, env)
            if err != nil {
                return nil, err
            }
            fields[name] = val
        }

        // Return extended record
        return record.Extend(fields), nil

    case *RecordRestriction:
        // Evaluate base record
        recordVal, err := e.Eval(ex.Record, env)
        if err != nil {
            return nil, err
        }

        record, ok := recordVal.(*RecordValue)
        if !ok {
            return nil, fmt.Errorf("cannot restrict non-record: %T", recordVal)
        }

        // Return restricted record
        return record.Restrict(ex.Remove), nil

    case *RecordUpdate:
        // Evaluate base record
        recordVal, err := e.Eval(ex.Record, env)
        if err != nil {
            return nil, err
        }

        record, ok := recordVal.(*RecordValue)
        if !ok {
            return nil, fmt.Errorf("cannot update non-record: %T", recordVal)
        }

        // Evaluate update fields
        updates := make(map[string]Value)
        for name, fieldExpr := range ex.Updates {
            val, err := e.Eval(fieldExpr, env)
            if err != nil {
                return nil, err
            }
            updates[name] = val
        }

        // Return updated record
        return record.Update(updates), nil
    }
}
```

**Tests**: Add 12-15 runtime evaluation tests

### Phase 5: Examples & Documentation (Day 4, ~100 LOC, 4 hours)

**5.1: Example Files** (~60 LOC)

**`examples/record_extension.ail`**:
```ailang
module examples/record_extension

import std/io (println)

-- Example 1: Basic extension
export func example1() -> () ! {IO} {
  let person = {name: "Alice", age: 30};
  let employee = {person | id: 100, dept: "Engineering"};

  println("Employee: " ++ employee.name);
  println("ID: " ++ show(employee.id))
}

-- Example 2: Configuration merging
export func example2() -> () ! {IO} {
  let defaults = {port: 8080, host: "localhost", debug: false};
  let config = {defaults | debug: true, timeout: 30};

  println("Debug mode: " ++ show(config.debug));
  println("Timeout: " ++ show(config.timeout))
}

-- Example 3: Override existing field
export func example3() -> () ! {IO} {
  let original = {x: 1, y: 2};
  let modified = {original | x: 10};  -- Override x

  println("Modified x: " ++ show(modified.x))
}

export func main() -> () ! {IO} {
  example1();
  example2();
  example3()
}
```

**`examples/record_operations.ail`**:
```ailang
module examples/record_operations

import std/io (println)

type Person = {name: string, age: int, ssn: string}

-- Remove sensitive field
export func makePublic(person: Person) -> {name: string, age: int} {
  {person - ssn}
}

-- Update age
export func birthday(person: Person) -> Person {
  {person with age: person.age + 1}
}

-- Add employee fields
export func promote(person: Person, id: int) -> {Person | id: int, dept: string} {
  {person | id: id, dept: "Engineering"}
}

export func main() -> () ! {IO} {
  let alice = {name: "Alice", age: 30, ssn: "123-45-6789"};

  let public = makePublic(alice);
  println("Public: " ++ public.name);

  let older = birthday(alice);
  println("Age: " ++ show(older.age));

  let employee = promote(alice, 100);
  println("Employee ID: " ++ show(employee.id))
}
```

**5.2: Documentation** (~40 LOC)
- Update CLAUDE.md with record operations syntax
- Update language guide with examples
- Add to CHANGELOG.md for v0.3.1
- Update README with record operations

**5.3: Tests** (~50 LOC)
- Integration tests with all three operations
- Edge case tests (empty records, chaining, etc.)
- Type error tests (missing fields, type mismatches)

## Error Codes

Reuse existing TC_REC codes:

- **TC_REC_001**: Missing field in restriction/update
  - `{r - nonexistent}` → "Field 'nonexistent' not found in record {x, y}"
  - `{r with nonexistent: val}` → "Field 'nonexistent' not found in record {x, y}"

- **TC_REC_004**: Field type mismatch in update
  - `{r with age: "thirty"}` where age: int → "Field 'age' type mismatch: expected Int, found String"

## Type System Properties

### Soundness

**Extension**:
```
If Γ ⊢ {r | f: v} : τ and Γ ⊢ r : σ, then σ <: τ (subtype)
```

**Restriction**:
```
If Γ ⊢ {r - f} : τ and Γ ⊢ r : σ, then τ <: σ (result is subtype of input)
```

**Update**:
```
If Γ ⊢ {r with f: v} : τ and Γ ⊢ r : σ, then τ = σ (same type)
```

### Row Polymorphism Interaction

```ailang
-- Extension preserves row variable
func extend[ρ](r: {x: int | ρ}) -> {x: int, y: bool | ρ} {
  {r | y: true}
}

-- Restriction preserves row variable
func restrict[ρ](r: {x: int, y: bool | ρ}) -> {y: bool | ρ} {
  {r - x}
}

-- Update preserves row variable and type
func update[ρ](r: {x: int | ρ}) -> {x: int | ρ} {
  {r with x: r.x + 1}
}
```

## Performance Considerations

### Memory

All operations create **new** records (immutable semantics):
```go
// Extension copies all fields from base + new fields
result := make(map[string]Value, len(base.Fields) + len(newFields))
```

**Optimization** (Phase 4, v0.4.0):
- Structural sharing via persistent data structures
- Only copy modified portions
- Share unchanged field references

### Time Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Extension | O(n + m) | O(n + m) |
| Restriction | O(n) | O(n - m) |
| Update | O(n) | O(n) |

Where:
- n = number of fields in base record
- m = number of fields added/removed/updated

## Testing Strategy

### Unit Tests (~50 tests total)

**Parser Tests** (15 tests):
- Valid syntax parsing
- Error recovery
- Edge cases (empty, nested, etc.)

**Type Checking Tests** (20 tests):
- Extension with new fields
- Extension with override
- Restriction of existing fields
- Restriction errors (missing field)
- Update with matching types
- Update errors (missing field, type mismatch)
- Row polymorphism preservation
- Nested operations

**Runtime Tests** (15 tests):
- Extension evaluation
- Restriction evaluation
- Update evaluation
- Chaining operations
- Integration with pattern matching

### Integration Tests

**Example Files** (2 files):
- `examples/record_extension.ail` - All three operations
- `examples/record_operations.ail` - Real-world use cases

### Property Tests

```ailang
-- Extension then restriction is identity (if no overlap)
property "extend_then_restrict" {
  forall(r: Record, f: Field, v: Value) =>
    {r | f: v} - f == r  (if f not in r)
}

-- Update is idempotent with same value
property "update_idempotent" {
  forall(r: Record, f: Field) =>
    {r with f: r.f} == r
}

-- Restriction then extension is not identity (order matters)
property "restrict_then_extend" {
  forall(r: Record, f: Field, v: Value) =>
    {{r - f} | f: v} == {r with f: v}  (if f in r)
}
```

## Migration & Compatibility

### v0.3.0-alpha3 → v0.3.1

**Before** (manual record construction):
```ailang
let employee = {
  name: person.name,
  age: person.age,
  id: 100,
  dept: "Engineering"
}
```

**After** (extension syntax):
```ailang
let employee = {person | id: 100, dept: "Engineering"}
```

**Compatibility**: Fully backward compatible
- New syntax is additive (doesn't change existing code)
- Manual construction still works
- No breaking changes

### Future: Deprecate Manual Construction?

**No** - Manual construction is still useful for:
- Reordering fields
- Renaming fields
- Complex transformations

Extension syntax is **sugar**, not replacement.

## Implementation Checklist

### Day 1: Parser & AST
- [ ] Add `with` keyword to lexer
- [ ] Add AST nodes (RecordExtension, RecordRestriction, RecordUpdate)
- [ ] Implement parser rules
- [ ] Add 15 parser unit tests
- [ ] Verify parsing with example files

### Day 2: Elaboration & Type Checking (Part 1)
- [ ] Add Core AST nodes
- [ ] Implement elaboration for all three operations
- [ ] Add 10 elaboration tests
- [ ] Implement type inference for extension
- [ ] Add 8 type checking tests for extension

### Day 3: Type Checking (Part 2) & Runtime
- [ ] Implement type inference for restriction
- [ ] Implement type inference for update
- [ ] Add 12 type checking tests for restriction/update
- [ ] Add RecordValue methods (Extend, Restrict, Update)
- [ ] Implement evaluator cases
- [ ] Add 15 runtime tests

### Day 4: Examples & Documentation
- [ ] Create `examples/record_extension.ail`
- [ ] Create `examples/record_operations.ail`
- [ ] Update CLAUDE.md with syntax
- [ ] Update CHANGELOG.md for v0.3.1
- [ ] Update README.md with examples
- [ ] Run full test suite (expect 52+ examples passing)
- [ ] Update examples/STATUS.md

### Day 5: Polish & Review (if needed)
- [ ] Add property-based tests
- [ ] Performance benchmarks
- [ ] Error message review
- [ ] Documentation review
- [ ] Code review & refactoring

## Success Criteria

### Must Have (v0.3.1 Release)
- ✅ All three operations (extension, restriction, update) work
- ✅ Type checking correctly infers types
- ✅ Runtime evaluation produces correct values
- ✅ All unit tests pass (50+ tests)
- ✅ Example files work
- ✅ Documentation updated

### Nice to Have
- ✅ Property-based tests
- ✅ Performance benchmarks
- ✅ Nested operation support
- ✅ Row polymorphism integration

### Out of Scope (v0.4.0+)
- ❌ Structural sharing optimization
- ❌ Record comprehensions
- ❌ Field renaming syntax
- ❌ Spread operator (if different from extension)

## Estimated Timeline

**Total**: 4-5 days (~850 LOC)

| Day | Tasks | LOC | Hours |
|-----|-------|-----|-------|
| 1 | Parser & AST | 150 | 6 |
| 2 | Elaboration & Type Checking (Part 1) | 200 | 8 |
| 3 | Type Checking (Part 2) & Runtime | 250 | 10 |
| 4 | Examples & Documentation | 150 | 6 |
| 5 | Polish (if needed) | 100 | 4 |

**Buffer**: 1 day for unforeseen issues

## Risks & Mitigations

### Risk 1: Parser Ambiguity
**Problem**: `{r | x: 1}` vs `{x: 1 | ...}` (record literal with row)

**Mitigation**:
- Lookahead parsing: if first token after `{` is identifier followed by `|`, it's extension
- Otherwise, it's record literal
- Clear error messages for syntax errors

### Risk 2: Type Inference Complexity
**Problem**: Row variables with extension/restriction may be complex

**Mitigation**:
- Start with closed records (no row variables)
- Add row variable support incrementally
- Extensive unit tests for edge cases

### Risk 3: Runtime Performance
**Problem**: Creating new records on every operation

**Mitigation**:
- Accept O(n) copy for v0.3.1
- Document performance characteristics
- Plan structural sharing for v0.4.0
- Benchmark to establish baseline

## References

**Prior Art**:
- **PureScript**: Record extension/restriction syntax
- **Elm**: Record update syntax `{r | x = 1}`
- **OCaml**: Record update `{r with x = 1}`
- **TypeScript**: Spread operator `{...r, x: 1}`
- **Haskell**: Record update via lens libraries

**AILANG Docs**:
- M-R5 Core: `design_docs/implemented/v0_3_0/M-R5_records.md`
- Future Enhancements: `design_docs/planned/M-R5_future_enhancements.md`
- CHANGELOG: v0.3.0-alpha3

**Papers**:
- Rémy, D. (1994). "Type Inference for Records in Natural Extension of ML"
- Gaster, B. R., & Jones, M. P. (1996). "A Polymorphic Type System for Extensible Records and Variants"

# Type Classes Implementation in AILANG v2.0

## Overview

This document describes the implementation of type classes with explicit dictionary passing in AILANG v2.0. The system transforms high-level type class constraints into explicit dictionary calls during compilation, making all type class operations transparent and deterministic.

## Architecture

### Core Components

#### 1. Instance Registry (`internal/types/instances.go`)
- **ClassInstance**: Represents a type class instance with method dictionaries
- **InstanceEnv**: Manages coherent instance lookup with superclass provision
- **LoadBuiltinInstances()**: Loads standard instances (Num, Eq, Ord, Show)

```go
type ClassInstance struct {
    ClassName string
    TypeHead  Type
    Dict      Dict  // map[string]string for method identifiers
    Super     []string
}
```

#### 2. Constraint Resolution (`internal/types/typechecker_core.go`)
- **ResolvedConstraint**: Tracks resolved type class constraints without runtime payloads
- **resolveGroundConstraints()**: Maps operators to dictionary method calls
- **fillOperatorMethods()**: Associates operator AST nodes with method names

```go
type ResolvedConstraint struct {
    NodeID    uint64  // Links to AST node
    ClassName string  // e.g., "Num"
    Type      Type    // Ground type, e.g., TInt
    Method    string  // e.g., "add"
}
```

#### 3. Dictionary Elaboration (`internal/elaborate/elaborate.go`)
- **ElaborateWithDictionaries()**: Transforms operators to ANF-bound dictionary calls
- Converts `2 + 3` into:
```
let dict1 = lookupDict("prelude", "Num", "Int", "add") in
dict1(2, 3)
```

#### 4. Numeric Defaulting (`internal/types/defaulting.go`)
- **DefaultingConfig**: Module-scoped defaulting rules
- **applyNumericDefaulting()**: Defaults ambiguous Num constraints to Int
- **DefaultingTrace**: Logs all defaulting decisions for transparency

#### 5. Type Normalization (`internal/types/normalize.go`)
- **NormalizeTypeName()**: Creates stable dictionary keys
- **MakeDictionaryKey()**: Generates fully-qualified method identifiers
- **ParseDictionaryKey()**: Decomposes dictionary keys into components

Format: `module.class.type.method` (e.g., `"prelude.Num.Int.add"`)

## Key Features

### 1. Explicit Dictionary Passing
All type class operations are transformed into explicit dictionary lookups:

**Source:**
```ailang
2 + 3
```

**Elaborated ANF:**
```ailang
let dict_add_1 = lookupDict("prelude", "Num", "Int", "add") in
dict_add_1(2, 3)
```

### 2. Superclass Provision
Ord automatically provides Eq using lawful definitions:

```ailang
-- If only Ord[T] exists, Eq[T] is derived as:
eq(x, y) = ¬lt(x, y) ∧ ¬lt(y, x)
neq(x, y) = lt(x, y) ∨ lt(y, x)
```

### 3. Coherent Instance Resolution
- No overlapping instances allowed
- Deterministic lookup with early error detection
- Module-scoped instance visibility

### 4. Numeric Literal Defaulting
Ambiguous numeric literals default to `Int`:

```ailang
let x = 42        -- Defaults to Int
let y: Float = 42 -- Constrained to Float
```

### 5. Lawful Float Semantics
Float equality implements equivalence relation:
- `NaN == NaN` returns `true` (reflexivity)
- Total ordering for comparisons

## Operator Transformations

| Operator | Type Class | Method | Example |
|----------|------------|--------|---------|
| `+`      | Num        | add    | `a + b` → `Num[T].add(a, b)` |
| `-`      | Num        | sub    | `a - b` → `Num[T].sub(a, b)` |
| `*`      | Num        | mul    | `a * b` → `Num[T].mul(a, b)` |
| `/`      | Num        | div    | `a / b` → `Num[T].div(a, b)` |
| `==`     | Eq         | eq     | `a == b` → `Eq[T].eq(a, b)` |
| `!=`     | Eq         | neq    | `a != b` → `Eq[T].neq(a, b)` |
| `<`      | Ord        | lt     | `a < b` → `Ord[T].lt(a, b)` |
| `<=`     | Ord        | lte    | `a <= b` → `Ord[T].lte(a, b)` |
| `>`      | Ord        | gt     | `a > b` → `Ord[T].gt(a, b)` |
| `>=`     | Ord        | gte    | `a >= b` → `Ord[T].gte(a, b)` |

## Built-in Instances

### Num Class
- `Num[Int]`: Integer arithmetic
- `Num[Float]`: Floating-point arithmetic with IEEE 754 semantics

### Eq Class  
- `Eq[Int]`: Integer equality
- `Eq[Float]`: Lawful float equality (NaN == NaN)
- `Eq[String]`: String equality
- `Eq[Bool]`: Boolean equality

### Ord Class
- `Ord[Int]`: Integer ordering
- `Ord[Float]`: Total float ordering (NaN as maximum)
- `Ord[String]`: Lexicographic string ordering

### Show Class
- `Show[Int]`: Integer to string conversion
- `Show[Float]`: Float to string conversion  
- `Show[String]`: String identity
- `Show[Bool]`: Boolean to string conversion

## Type Inference Integration

### Constraint Collection
During type inference, constraints are collected and partitioned:

```go
func (tc *CoreTypeChecker) partitionConstraints(constraints []ClassConstraint) (ground, nonGround []ClassConstraint)
```

- **Ground constraints**: Have concrete types (e.g., `Num[Int]`)
- **Non-ground constraints**: Have type variables (e.g., `Num[α]`)

### Resolution Process
1. **Unification**: Solve type equations
2. **Defaulting**: Apply numeric literal defaults  
3. **Ground Resolution**: Resolve constraints with concrete types
4. **Generalization**: Preserve non-ground constraints in type schemes

### Qualified Type Schemes
Non-ground constraints are preserved in polymorphic type schemes:

```go
type QualifiedScheme struct {
    Constraints []ClassConstraint  // e.g., [Num[α]]
    Scheme      *Scheme           // ∀α. α → α → α
}
```

Example: `∀α. Num[α] ⇒ α → α → α` for the polymorphic addition function.

## Testing Strategy

### Unit Tests (`internal/types/*_test.go`)
- ✅ Instance lookup and coherence
- ✅ Superclass provision  
- ✅ Type normalization
- ✅ Dictionary key handling
- ✅ Operator method mapping
- ✅ Builtin instance loading

### Integration Tests (`cmd/test_integration/`)
- ✅ Complete pipeline from source to dictionary elaboration
- ✅ Pure and constrained polymorphism
- ✅ Instance resolution
- ⚠️ Numeric defaulting (known issue)

### Core Typechecker Tests (`cmd/test_typechecker_core/`)
- ✅ Basic type inference
- ✅ Let polymorphism
- ⚠️ Numeric literal handling (defaulting issue)

## Known Issues

### Defaulting Integration
Current issue: Numeric literals remain as unconstrained type variables instead of defaulting to `Int` during type checking. This suggests the defaulting logic runs too late in the pipeline or has a bug in application.

**Symptoms:**
- `2 + 3` fails with "No instance for Num[α2]" instead of defaulting to `Num[Int]`
- Pure functions without numeric literals work correctly
- Instance lookup system works correctly

**Next Steps:**
1. Debug defaulting timing in the type checking pipeline
2. Ensure defaulting runs before constraint resolution
3. Verify defaulting config is properly propagated

## Examples

See the following example files:
- `examples/type_classes.ail`: Basic type class usage
- `examples/dictionary_passing.ail`: ANF transformation examples
- `examples/defaulting_trace.ail`: Numeric literal defaulting

## Design Compliance

This implementation follows AILANG v2.0 specifications:
- ✅ Explicit dictionary passing (no implicit resolution)
- ✅ Deterministic execution
- ✅ Machine-decidable compilation  
- ✅ Lawful type class semantics
- ✅ Coherent instance resolution
- ✅ A-Normal Form output with let-bound operations

The system successfully transforms high-level type class operations into explicit, traceable dictionary calls, making AILANG v2.0 ideal for AI training data generation and deterministic execution.
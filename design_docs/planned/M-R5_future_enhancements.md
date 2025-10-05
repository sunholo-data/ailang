# M-R5 Future Enhancements

**Status**: 📋 Planned for v0.3.1+
**Created**: October 5, 2025
**Depends on**: M-R5 Records (✅ v0.3.0-alpha3)

## Overview

This document outlines future enhancements to AILANG's record system beyond the core M-R5 implementation. These features build on the foundation of row polymorphism and subsumption delivered in v0.3.0-alpha3.

## Phase 1: TRecord2 Migration (v0.3.1, ~2-3 days)

### Goals
- Make TRecord2 the default record type
- Deprecate TRecord with clear migration path
- Remove compatibility shims

### Tasks

**1. Default AILANG_RECORDS_V2=1** (~50 LOC, 2 hours)
- Change default in typechecker constructors
- Add `AILANG_RECORDS_V1=1` flag for backwards compat
- Update all documentation
- Update CHANGELOG with migration guide

**2. Deprecation Warnings** (~100 LOC, 4 hours)
- Add warning when TRecord is used
- Suggest migration to TRecord2
- Log deprecation notices in type checker
- Update error messages to reference new types

**3. Example Migration** (~200 LOC, 6 hours)
- Convert all examples to use TRecord2
- Remove AILANG_RECORDS_V2 flags from tests
- Update README examples
- Verify all 66 examples pass with TRecord2

**4. Documentation Update** (~2 hours)
- Update CLAUDE.md with TRecord2 as default
- Update language guide with row polymorphism
- Create migration guide for users
- Update CHANGELOG

**Total**: ~350 LOC, ~14 hours (2 days)

## Phase 2: Record Extension Syntax (v0.3.1, ~3-4 days)

### Syntax Design

**Record Extension** - Add fields to existing record:
```ailang
let person = {name: "Alice", age: 30}
let employee = {person | id: 100, dept: "Engineering"}
-- Result: {name: "Alice", age: 30, id: 100, dept: "Engineering"}
```

**Record Restriction** - Remove fields from record:
```ailang
let detailed = {id: 42, name: "Alice", secret: "password"}
let public = {detailed - secret}
-- Result: {id: 42, name: "Alice"}
```

**Record Update** - Change field values:
```ailang
let person = {name: "Alice", age: 30}
let older = {person with age: 31}
-- Result: {name: "Alice", age: 31}
```

### Implementation Plan

**1. Lexer & Parser** (~150 LOC, 6 hours)
- Add `with` keyword
- Parse `{record | field: value, ...}` syntax
- Parse `{record - field, ...}` syntax
- Parse `{record with field: value, ...}` syntax
- Update AST with RecordExtension, RecordRestriction, RecordUpdate nodes

**2. Elaboration** (~200 LOC, 8 hours)
- Elaborate extension to functional merge
- Elaborate restriction to functional filter
- Elaborate update to extension with override
- Generate Core AST for runtime

**3. Type Checking** (~250 LOC, 10 hours)
- Infer extension: `{r | ρ} + {x: τ} → {r, x: τ | ρ}`
- Infer restriction: `{r, x: τ | ρ} - x → {r | ρ}`
- Infer update: `{r, x: τ₁ | ρ} with x: τ₂ → {r, x: τ₂ | ρ}` (requires τ₁ ~ τ₂)
- Handle duplicate field errors
- Row variable management

**4. Runtime** (~100 LOC, 4 hours)
- Implement record merge in eval
- Implement record filter in eval
- Preserve field order (deterministic)
- Efficient implementation (no copying when possible)

**5. Tests & Examples** (~150 LOC, 6 hours)
- Unit tests for extension (10 cases)
- Unit tests for restriction (8 cases)
- Unit tests for update (8 cases)
- Create `examples/record_extension.ail`
- Create `examples/record_operations.ail`

**Total**: ~850 LOC, ~34 hours (4-5 days)

## Phase 3: Advanced Row Features (v0.3.2+, ~5-7 days)

### 1. Row Kinds Enforcement (~300 LOC, 2 days)

**Goal**: Prevent mixing record and effect rows

```ailang
-- Should fail type checking:
type Bad = {x: int | EffectRow}  -- ❌ Kind error
```

**Implementation**:
- Add kind checking to row unification
- Error code: TC_REC_005 (row kind mismatch)
- Update unifyRows() with kind guards
- Add 5-8 unit tests

### 2. Duplicate Field Detection (~200 LOC, 1 day)

**Goal**: Catch duplicate fields in record literals at parse time

```ailang
-- Should fail during parsing:
{name: "Alice", age: 30, name: "Bob"}  -- ❌ TC_REC_002
```

**Implementation**:
- Add field tracking during parsing
- Emit TC_REC_002 with both positions
- Show "first defined at line X, column Y"
- Add 3-5 unit tests

### 3. Record Pattern Matching (~400 LOC, 2-3 days)

**Goal**: Pattern match on record structure

```ailang
match person {
  {name: n, age: a} => println(n),
  {id: i} => println(show(i)),
  _ => println("unknown")
}
```

**Implementation**:
- Parse record patterns
- Elaborate to pattern matching AST
- Type check with row polymorphism
- Implement in evaluator
- Add 10-15 unit tests

### 4. Record Comprehensions (~500 LOC, 2-3 days)

**Goal**: Map over record fields

```ailang
-- Transform all fields
{f: toUpper(v) for f:v in record}

-- Filter fields
{f: v for f:v in record if f != "secret"}
```

**Implementation**:
- Parser support for comprehension syntax
- Elaborate to map/filter operations
- Type inference with row polymorphism
- Runtime implementation
- Add 8-12 unit tests

**Total Phase 3**: ~1,400 LOC, ~7-10 days

## Phase 4: Performance Optimizations (v0.4.0+)

### 1. Record Sharing (~200 LOC, 2 days)

**Goal**: Share unchanged fields between records

```ailang
let base = {x: 1, y: 2, z: 3}
let modified = {base with x: 10}
-- modified.y and modified.z share memory with base
```

**Implementation**:
- Immutable record internals
- Structural sharing via persistent data structures
- Benchmark suite for record operations
- Verify no observable behavior changes

### 2. Field Access Optimization (~150 LOC, 1-2 days)

**Goal**: Compile field access to array offsets

**Current**: Linear scan of field map
**Optimized**: Direct array access with compile-time offset

**Implementation**:
- Record layout computation during elaboration
- Store field offsets in TypedRecordAccess
- Use offsets in runtime for O(1) access
- Benchmark showing 5-10x speedup

### 3. Record Literal Optimization (~100 LOC, 1 day)

**Goal**: Constant folding for record literals

```ailang
-- Compile time:
let config = {port: 8080, host: "localhost"}

-- Runtime:
-- Single pre-allocated RecordValue, no field-by-field construction
```

**Implementation**:
- Detect constant record literals
- Pre-allocate in compiled module
- Reference pre-allocated value at runtime
- Reduce allocation overhead

**Total Phase 4**: ~450 LOC, ~4-5 days

## Phase 5: Advanced Type Features (v0.4.0+)

### 1. Record Type Aliases (~150 LOC, 1 day)

```ailang
type Person = {name: string, age: int}
type Employee = {Person | id: int, dept: string}
```

### 2. Anonymous Record Types (~200 LOC, 1-2 days)

```ailang
-- Inline record types in function signatures
func greet(person: {name: string}) -> string {
  "Hello, " ++ person.name
}
```

### 3. Recursive Record Types (~300 LOC, 2-3 days)

```ailang
type TreeNode = {
  value: int,
  left: Option[TreeNode],
  right: Option[TreeNode]
}
```

### 4. Record Constraints in Type Classes (~400 LOC, 3-4 days)

```ailang
class Identifiable[r | {id: int | ρ}] {
  func getId(r: r) -> int
}

instance Identifiable[{id: int, name: string}] {
  func getId(r) { r.id }
}
```

**Total Phase 5**: ~1,050 LOC, ~7-10 days

## Timeline & Priorities

### v0.3.1 (Next Release, ~2-3 weeks)
**Priority: HIGH**
- ✅ Phase 1: TRecord2 Migration (2 days)
- ✅ Phase 2: Record Extension Syntax (4-5 days)
- 📊 **Total**: ~1,200 LOC, ~6-7 days

**Goals**:
- TRecord2 as default
- Full record extension/restriction/update syntax
- 60+ examples passing
- Production ready

### v0.3.2 (Follow-up, ~2-3 weeks)
**Priority: MEDIUM**
- ✅ Phase 3.1: Row Kinds Enforcement (2 days)
- ✅ Phase 3.2: Duplicate Field Detection (1 day)
- ✅ Phase 3.3: Record Pattern Matching (2-3 days)
- 📊 **Total**: ~900 LOC, ~5-6 days

**Goals**:
- Better error messages
- Pattern matching over records
- Robust type system

### v0.4.0 (Major Release, ~1-2 months)
**Priority: LOW/NICE-TO-HAVE**
- ⚡ Phase 4: Performance Optimizations (4-5 days)
- 🎯 Phase 5: Advanced Type Features (7-10 days)
- 🗑️ Remove TRecord completely
- 🗑️ Remove all compatibility shims
- 📊 **Total**: ~1,500 LOC, ~11-15 days

**Goals**:
- 10x faster field access
- Advanced type features
- Production optimized

## Deferred Features (v0.5.0+)

### Row Polymorphic Effects
Unify record rows and effect rows under single system.

```ailang
func doThings[e | {IO, FS | ε}](action: () -> () ! {e}) -> () ! {e} {
  action()
}
```

**Complexity**: HIGH
**Timeline**: v0.5.0+
**Effort**: ~2-3 weeks

### Record Macros
Compile-time record generation.

```ailang
@derive[Show, Eq]
type Person = {name: string, age: int}
```

**Complexity**: HIGH
**Timeline**: v0.6.0+ (requires macro system)
**Effort**: ~3-4 weeks

### Structural Typing
Full structural subtyping beyond subsumption.

```ailang
-- Any record with at least {x: int, y: int} is a Point
type Point = structural {x: int, y: int}
```

**Complexity**: VERY HIGH
**Timeline**: v1.0.0+
**Effort**: ~4-6 weeks

## Success Metrics

### v0.3.1 Success Criteria
- ✅ 100% of examples use TRecord2
- ✅ Extension/restriction/update syntax works
- ✅ No performance regression
- ✅ 65+ examples passing
- ✅ All unit tests pass

### v0.3.2 Success Criteria
- ✅ Zero row kind violations in test suite
- ✅ All duplicate fields caught at parse time
- ✅ Pattern matching over records works
- ✅ 70+ examples passing

### v0.4.0 Success Criteria
- ✅ Field access <100ns (vs ~500ns current)
- ✅ Zero TRecord usage in codebase
- ✅ Advanced type features working
- ✅ 75+ examples passing
- ✅ Ready for production use

## References

- **M-R5 Core**: `design_docs/implemented/v0_3_0/M-R5_records.md`
- **CHANGELOG**: v0.3.0-alpha3
- **Prior Art**:
  - PureScript record extension syntax
  - OCaml row polymorphism
  - Elm record updates
  - TypeScript mapped types

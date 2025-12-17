# M-REFLECT: Structural Reflection & User-Defined Type Classes

**Status**: Planned
**Target**: v0.7.0
**Priority**: P1 (Medium) - Foundational for extensibility
**Estimated**: ~2 weeks (reduced from 3-4 weeks after infrastructure audit)
**Dependencies**: Monomorphization (v0.4.0), CoreTypeInfo validation (M-DX4)

## Infrastructure Audit (2025-12-17)

**Significant infrastructure already exists.** This audit revised estimates downward.

| Component | Status | Completion | Key Files |
|-----------|--------|------------|-----------|
| **Type Classes** | Partial | **65%** | `internal/types/dictionaries.go`, `internal/link/linker.go` |
| **Reflection** | Partial | **25%** | `internal/types/typeinfo.go`, `internal/repl/repl_commands.go` |
| **Schema Registry** | Working | **60%** | `internal/schema/registry.go`, `internal/schema/plan.go` |

### What Already Exists

**Type Classes (65% complete):**
- ✅ `CLASS` and `INSTANCE` tokens in lexer (`internal/lexer/token.go` lines 33-34)
- ✅ Core AST nodes: `DictAbs`, `DictApp`, `DictRef`, `DictParam`, `DictValue` (`internal/core/core.go`)
- ✅ `DictionaryRegistry` with Num/Eq/Ord instances (~467 LOC in `internal/types/dictionaries.go`)
- ✅ Linker support for dictionary validation (`internal/link/linker.go`)
- ✅ `:instances` REPL command working (`internal/repl/repl_commands.go` lines 354-378)
- ✅ Working examples: `examples/runnable/type_classes.ail`, `examples/runnable/typeclasses.ail`
- ❌ **Missing:** Parser grammar for `class`/`instance` syntax

**Reflection (25% complete):**
- ✅ `types.TypeInfo` and `types.CoreTypeInfo` structures (`internal/types/typeinfo.go`)
- ✅ Type checker populates TypeInfo throughout pipeline
- ✅ `:type <expr>` and `:effects <expr>` REPL commands working
- ✅ Substitution application with cycle detection
- ❌ **Missing:** `reflectType()`/`reflectEffect()` builtins, JSON serialization, `:reflect` command

**Schema Registry (60% complete):**
- ✅ `internal/schema/` package with versioned schemas (~6 files)
- ✅ `MarshalDeterministic()` for reproducible JSON output
- ✅ Plan schema with full CRUD operations
- ❌ **Missing:** TypeInfo/EffectRow schemas, full validation

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Removes need for hardcoded special-casing; one mechanism for all type classes |
| Preserve Semantic Clarity | + | +1 | Types self-describe their capabilities; explicit instance declarations |
| Increase Determinism | 0 | 0 | Reflection is read-only; no runtime modification of types |
| Lower Token Cost | + | +1 | AI can query type structure instead of guessing; structured JSON responses |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

## Problem Statement

**AILANG currently has hardcoded type class instances** (Num, Eq, Ord, Show) that cannot be extended by users. This limits the language's expressiveness and requires compiler changes to add new abstractions.

**Current State:**
- Type classes are hardcoded in `internal/link/linker.go` (lines 99-108)
- Only `Num`, `Eq`, `Ord` are recognized; `Show` mentioned but not consistently implemented
- No runtime type introspection - AIs cannot inspect types to self-debug
- Adding a new type class requires modifying Go code in multiple places
- No way for users to define `Functor`, `Monad`, `Semigroup`, etc.

**Impact:**
- **AI code generation**: Cannot reason about type capabilities at runtime
- **Language extensibility**: Users cannot define domain-specific abstractions
- **Self-debugging**: AIs cannot inspect why type errors occur
- **Documentation drift**: Website mentions "planned" features that don't exist

**Referenced in:**
- [CLAUDE.md](../../../CLAUDE.md) lines 60-70: "Planned: Structural reflection for user-defined type classes"
- [v0.4-roadmap.md](../../archive/v0.4-roadmap.md) lines 265-396: v0.4.1 Reflection APIs
- [m-verify-arc-verification-policy-mode.md](../v0_6_1/m-verify-arc-verification-policy-mode.md) line 74

## Goals

**Primary Goal:** Enable runtime type introspection and user-defined type classes through structural reflection.

**Success Metrics:**
- AIs can query any value's type structure via `reflectType(x)`
- Users can define custom type classes (e.g., `class Functor f where ...`)
- Remove hardcoded type class switch statements from linker
- 100% accuracy for monomorphic type reflection
- JSON schema for type representation consumable by AI models

## Solution Design

### Overview

Structural reflection provides a **read-only view** of AILANG's type system at runtime. Unlike traditional reflection (which can modify behavior), AILANG reflection is purely observational - it returns structured data about types, enabling AI self-debugging without introducing non-determinism.

User-defined type classes build on reflection by allowing users to declare **interfaces** that types can implement through **instance declarations**.

### Architecture

**Components:**

1. **Type Reflection API** (`internal/reflect/`)
   - `reflectType : forall a. a -> TypeInfo` - Get type structure
   - `reflectEffect : forall a. a -> EffectRow` - Get effect signature
   - JSON-serializable `TypeInfo` and `EffectRow` types

2. **Type Class System** (`internal/typeclass/`)
   - Class declarations: `class Functor f where map : (a -> b) -> f a -> f b`
   - Instance declarations: `instance Functor List where map = list_map`
   - Instance resolution via structural matching

3. **Schema Registry** (`internal/schema/`)
   - Machine-readable type/class definitions
   - Versioned schemas for stability
   - Export for AI training data

### Type Reflection Design

**TypeInfo Structure:**
```go
// internal/reflect/typeinfo.go
type TypeInfo struct {
    Kind       string     `json:"kind"`       // "con", "var", "func", "forall", "app", "record", "effect"
    Name       string     `json:"name,omitempty"`
    TypeParams []string   `json:"typeParams,omitempty"`
    Params     []TypeInfo `json:"params,omitempty"`
    Return     *TypeInfo  `json:"return,omitempty"`
    Fields     []Field    `json:"fields,omitempty"`
    Effects    []string   `json:"effects,omitempty"`
}

type Field struct {
    Name string   `json:"name"`
    Type TypeInfo `json:"type"`
}
```

**AILANG Surface Syntax:**
```ailang
-- Get type information (monomorphic values only in v0.7.0)
let info = reflectType(42)
-- Returns: {"kind": "con", "name": "int"}

let listInfo = reflectType([1, 2, 3])
-- Returns: {"kind": "app", "name": "List", "params": [{"kind": "con", "name": "int"}]}

-- Check effects on a function
let eff = reflectEffect(println)
-- Returns: {"effects": ["IO"]}
```

**REPL Integration:**
```
> :reflect fold
{
  "kind": "forall",
  "typeParams": ["a", "b"],
  "params": [
    {"kind": "func", "params": [{"kind": "var", "name": "b"}, {"kind": "var", "name": "a"}], "return": {"kind": "var", "name": "b"}},
    {"kind": "var", "name": "b"},
    {"kind": "app", "name": "List", "params": [{"kind": "var", "name": "a"}]}
  ],
  "return": {"kind": "var", "name": "b"},
  "effects": []
}
```

### User-Defined Type Classes Design

**Class Declaration Syntax:**
```ailang
-- Define a type class
class Functor f where
    map : forall a b. (a -> b) -> f a -> f b

class Eq a where
    eq : a -> a -> bool
    neq : a -> a -> bool
    -- Default implementation
    neq = \x y. not (eq x y)

class Ord a extends Eq where
    lt : a -> a -> bool
    lte : a -> a -> bool
    gt : a -> a -> bool
    gte : a -> a -> bool
```

**Instance Declaration Syntax:**
```ailang
-- Implement type class for a type
instance Functor List where
    map = list_map

instance Eq int where
    eq = _int_eq

instance Ord int where
    lt = _int_lt
    lte = \x y. lt(x, y) || eq(x, y)
    gt = \x y. lt(y, x)
    gte = \x y. lte(y, x)

-- Derived instances (structural)
instance Eq (a, b) given (Eq a, Eq b) where
    eq = \(x1, y1) (x2, y2). eq(x1, x2) && eq(y1, y2)
```

**Resolution Algorithm:**
1. Parse class/instance declarations during elaboration
2. Build instance registry mapping (ClassName, Type) → Implementation
3. At dictionary reference sites, look up registry
4. For polymorphic contexts, pass dictionary as implicit parameter
5. Fail with clear error if no instance found

### Implementation Plan

**Phase 1: Type Reflection** (~2-3 days) - 25% exists
- [x] ~~Define `TypeInfo` and `CoreTypeInfo` Go structs~~ (exists in `internal/types/typeinfo.go`)
- [x] ~~Type checker populates TypeInfo~~ (working)
- [x] ~~`:type` and `:effects` REPL commands~~ (working)
- [ ] Create `internal/reflect/` package for JSON serialization
- [ ] Convert `types.Type` → JSON-serializable `reflect.TypeInfo`
- [ ] Add `reflectType` builtin function
- [ ] Add `reflectEffect` builtin function
- [ ] Add `:reflect` REPL command (JSON output)
- [ ] Unit tests for all primitive types

**Phase 2: Class/Instance Parsing** (~1 week) - Tokens exist, grammar missing
- [x] ~~Add `CLASS` and `INSTANCE` tokens to lexer~~ (exists at lines 33-34)
- [ ] Add `ClassDecl` and `InstanceDecl` AST nodes
- [ ] Add `parseClassDecl()` to parser
- [ ] Add `parseInstanceDecl()` to parser
- [ ] Parse method signatures in class body
- [ ] Parse method implementations in instance body
- [ ] Parse `given` constraints for conditional instances
- [ ] Extend module system to export classes/instances

**Phase 3: Instance Resolution** (~3-4 days) - Registry exists, needs extension
- [x] ~~`DictionaryRegistry` for instance storage~~ (exists in `internal/types/dictionaries.go`)
- [x] ~~Instance lookup by (class, type) pair~~ (working via `Lookup()`, `LookupMethod()`)
- [x] ~~Core AST nodes for dictionaries~~ (`DictAbs`, `DictApp`, `DictRef`, etc.)
- [x] ~~Linker validates dictionary references~~ (working)
- [ ] Extend registry to accept user-defined classes
- [ ] Extend registry to accept user-defined instances
- [ ] Handle superclass constraints (`extends`)
- [ ] Handle conditional instances (`given`)
- [ ] Remove hardcoded class switch in `internal/link/linker.go` lines 99-108
- [ ] Wire parsed instances to `DictionaryRegistry`

**Phase 4: Standard Library Migration** (~1-2 days) - Instances exist, need syntax
- [x] ~~Num instances for int, float~~ (7 methods each, working)
- [x] ~~Eq instances for int, float, bool, string~~ (working)
- [x] ~~Ord instances for int, float, string~~ (6 methods each, working)
- [ ] Convert hardcoded Go instances to AILANG `instance` declarations
- [ ] Define `Num`, `Eq`, `Ord`, `Show` as AILANG `class` declarations
- [ ] Add `Functor`, `Applicative`, `Monad` classes
- [ ] Update stdlib to use new syntax

### Files to Modify/Create

**New files:**
- `internal/reflect/typeinfo.go` - JSON-serializable TypeInfo (~150 LOC)
- `internal/reflect/serialize.go` - Convert `types.Type` → JSON (~200 LOC)
- `internal/reflect/builtins.go` - reflectType/reflectEffect implementations (~100 LOC)
- `stdlib/typeclass.ail` - Standard type class declarations in AILANG (~150 LOC)

**Modified files (already have infrastructure):**
- `internal/ast/ast.go` - Add `ClassDecl`, `InstanceDecl` nodes (~80 LOC)
- `internal/parser/parser.go` - Add `parseClassDecl()`, `parseInstanceDecl()` (~200 LOC)
- `internal/types/dictionaries.go` - Extend to accept user-defined classes/instances (~100 LOC)
- `internal/link/linker.go` - Remove hardcoded switch at lines 99-108 (~-20 LOC)
- `internal/builtins/spec.go` - Register reflectType, reflectEffect (~30 LOC)
- `internal/repl/repl_commands.go` - Add `:reflect` command (~50 LOC)
- `internal/schema/` - Add TypeInfo schema version (~50 LOC)

**Files that DON'T need changes (already working):**
- ✅ `internal/lexer/token.go` - CLASS, INSTANCE tokens exist
- ✅ `internal/types/typeinfo.go` - TypeInfo/CoreTypeInfo structures exist
- ✅ `internal/core/core.go` - DictAbs, DictApp, DictRef nodes exist
- ✅ `internal/repl/repl_commands.go` - `:instances`, `:type`, `:effects` commands exist

## Examples

### Example 1: AI Self-Debugging with Reflection

**Current (no reflection):**
```ailang
-- AI generates code with missing effect
fn greet(name: string) -> () =
    println("Hello, " ++ name)
-- ERROR: Missing effect annotation !: IO
-- AI has no way to introspect what went wrong
```

**After (with reflection):**
```ailang
fn checkEffects(f) !: IO =
    let declared = reflectSignature(f).effects
    let actual = reflectEffect(f)
    if actual != declared then
        println("Effect mismatch!")
        println("Declared: " ++ show(declared))
        println("Actual: " ++ show(actual))
        println("Add !: " ++ show(actual) ++ " to function signature")

-- AI can self-diagnose:
checkEffects(greet)
-- Output: Effect mismatch!
--         Declared: []
--         Actual: ["IO"]
--         Add !: IO to function signature
```

### Example 2: User-Defined Type Class

**Current (impossible):**
```ailang
-- Users cannot define Functor - it's not in the hardcoded list
-- They must wait for compiler updates
```

**After (user-defined):**
```ailang
-- Define Functor
class Functor f where
    map : forall a b. (a -> b) -> f a -> f b

-- Implement for List
instance Functor List where
    map = \f xs. match xs {
        [] -> [],
        [h | t] -> [f(h) | map(f, t)]
    }

-- Implement for Maybe
instance Functor Maybe where
    map = \f m. match m {
        Nothing -> Nothing,
        Just(x) -> Just(f(x))
    }

-- Use polymorphically
fn double_all[F: Functor](container: F int) -> F int =
    map(\x. x * 2, container)

double_all([1, 2, 3])        -- [2, 4, 6]
double_all(Just(5))          -- Just(10)
```

### Example 3: REPL Type Inspection

```
ailang> :reflect map
{
  "kind": "forall",
  "typeParams": ["f", "a", "b"],
  "constraints": [{"class": "Functor", "type": "f"}],
  "params": [
    {"kind": "func", "params": [{"kind": "var", "name": "a"}], "return": {"kind": "var", "name": "b"}},
    {"kind": "app", "name": "f", "params": [{"kind": "var", "name": "a"}]}
  ],
  "return": {"kind": "app", "name": "f", "params": [{"kind": "var", "name": "b"}]},
  "effects": []
}

ailang> :instances Functor
Functor List
Functor Maybe
Functor Result
Functor IO

ailang> :class Ord
class Ord a extends Eq where
    lt  : a -> a -> bool
    lte : a -> a -> bool
    gt  : a -> a -> bool
    gte : a -> a -> bool
```

## Success Criteria

- [ ] `reflectType(42)` returns valid JSON with `{"kind": "con", "name": "int"}`
- [ ] `reflectType([1,2,3])` correctly shows `List[int]` structure
- [ ] `:reflect` REPL command works for all stdlib functions
- [ ] User can define and use custom `Functor` class
- [ ] Hardcoded Num/Eq/Ord switch removed from linker.go
- [ ] All existing tests pass (no regression)
- [ ] New type class tests: 20+ test cases
- [ ] Documentation updated with type class guide
- [ ] Examples added to `examples/` directory

## Testing Strategy

**Unit tests:**
- TypeInfo JSON serialization round-trip
- All primitive types reflect correctly
- Function types include effect information
- Record types include all fields
- Class parsing produces correct AST
- Instance parsing with constraints

**Integration tests:**
- Reflection works through full pipeline
- Type classes resolve in multi-module programs
- Default method implementations work
- Superclass constraints enforced
- Conditional instances resolve correctly

**Manual testing:**
- REPL `:reflect` on complex polymorphic functions
- AI model consumes TypeInfo JSON successfully
- User-defined type class in real program

## Non-Goals

**Not in v0.7.0:**
- **Polymorphic reflection** - Only monomorphic values can be reflected; `forall` types return schema without instantiation. Deferred to v0.8.0 after research.
- **Runtime type modification** - Reflection is read-only; no `setType` or dynamic dispatch changes
- **Automatic deriving** - `deriving (Eq, Show)` syntax; requires more inference work
- **Multi-parameter type classes** - `class Convert a b where ...`; adds complexity
- **Functional dependencies** - `class Collection c e | c -> e`; advanced feature

**Why deferred:**
- Polymorphic reflection requires research on type erasure vs reification
- Automatic deriving needs structural analysis of ADTs
- Multi-param classes and fundeps are advanced Haskell features rarely needed

## Timeline (Revised After Audit)

**Week 1** (20-25 hours):
- Phase 1: Type Reflection (2-3 days)
  - Create `internal/reflect/` package
  - JSON serialization for existing TypeInfo
  - reflectType/reflectEffect builtins
  - `:reflect` REPL command
- Phase 2 Start: Class/Instance Parsing
  - AST node definitions
  - Begin parser implementation

**Week 2** (20-25 hours):
- Phase 2 Complete: Class/Instance Parsing
  - Finish parser for class/instance syntax
  - Tests for parsing
- Phase 3: Instance Resolution (3-4 days)
  - Extend existing DictionaryRegistry
  - Wire parsed instances to registry
  - Remove hardcoded switch
- Phase 4: Standard Library Migration (1-2 days)
  - Convert Go instances to AILANG syntax
  - Documentation and examples

**Total: ~40-50 hours across 2 weeks** (reduced from 55-75 hours)

**Effort saved by existing infrastructure:**
- ~15 hours: TypeInfo structures already exist
- ~5 hours: Lexer tokens already exist
- ~10 hours: DictionaryRegistry and Core AST already exist
- ~5 hours: REPL commands (`:type`, `:effects`, `:instances`) already exist

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Type erasure complicates reflection | High | Limit v0.7.0 to monomorphic types only; research reification for v0.8.0 |
| Instance resolution ambiguity | Medium | Require explicit instances; no overlapping instances in v0.7.0 |
| Performance overhead | Low | Reflection is opt-in; lazy computation of TypeInfo |
| Breaking existing code | Medium | Keep hardcoded paths as fallback during migration; deprecate gradually |

## References

**Internal (existing infrastructure):**
- [internal/types/dictionaries.go](../../../internal/types/dictionaries.go) - DictionaryRegistry (~467 LOC)
- [internal/types/typeinfo.go](../../../internal/types/typeinfo.go) - TypeInfo/CoreTypeInfo (~134 LOC)
- [internal/core/core.go](../../../internal/core/core.go) - DictAbs, DictApp, DictRef nodes (lines 420-488)
- [internal/link/linker.go](../../../internal/link/linker.go) - Dictionary linking, hardcoded switch (lines 99-108)
- [internal/repl/repl_commands.go](../../../internal/repl/repl_commands.go) - `:instances`, `:type`, `:effects` commands
- [internal/schema/](../../../internal/schema/) - Schema registry infrastructure
- [examples/runnable/type_classes.ail](../../../examples/runnable/type_classes.ail) - Working type class examples

**Design docs:**
- [v0.4-roadmap.md](../../archive/v0.4-roadmap.md) - Original reflection planning (v0.4.1-v0.4.2)

**External prior art:**
- [Haskell Type Classes](https://www.haskell.org/tutorial/classes.html) - Prior art
- [Rust Traits](https://doc.rust-lang.org/book/ch10-02-traits.html) - Alternative approach
- [TypeScript Structural Typing](https://www.typescriptlang.org/docs/handbook/type-compatibility.html) - Structural vs nominal

## Future Work

**v0.8.0+:**
- Polymorphic reflection with type reification
- Automatic `deriving` for Eq, Ord, Show
- Multi-parameter type classes
- Associated types (`class Container c where type Elem c`)

**v0.9.0+:**
- Capability budgets (`!: {IO @limit=2}`)
- Effect class constraints
- Higher-kinded polymorphism improvements

---

**Document created**: 2025-12-17
**Last updated**: 2025-12-17 (infrastructure audit added)

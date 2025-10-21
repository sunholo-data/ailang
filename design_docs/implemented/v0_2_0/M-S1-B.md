# M-S1B: Fix Export System & Builtin Visibility

## Goal
Fix the two critical blockers preventing stdlib from working:
1. **ADT Constructor Exports** - Make type names and constructors importable
2. **Builtin Visibility** - Add builtin type signatures to type environment

## Timeline
**Estimated**: 3-5 hours total
- Part A (Export System): 2-3 hours
- Part B (Builtin Signatures): 1-2 hours  
- Part C (extern keyword - optional): 30-45 min

## Part A: Export ADT Constructors & Type Names (2-3 hours)

### Changes Required

#### 1. Interface Builder - Add Types & Constructors (~60 LOC)
**File**: `internal/iface/builder.go`

Add fields to Interface:
```go
type Interface struct {
    // existing fields...
    Types        map[string]*TypeExport
    Constructors map[string]*ConstructorScheme
}

type TypeExport struct {
    Name   string
    Arity  int
}

type ConstructorScheme struct {
    TypeName   string
    CtorName   string
    FieldTypes []Type
    ResultType Type
}
```

Modify `BuildInterfaceWithConstructors()`:
- Extract type declarations from compile unit
- Add types to `itf.Types` map
- Add type names to `itf.Exports` (Kind: ExportType)
- Add constructors from elaborator to `itf.Constructors`
- Add constructor names to `itf.Exports` (Kind: ExportCtor)
- **CRITICAL**: Deterministic ordering (sort keys)

#### 2. Interface Serialization - Include in Digest (~30 LOC)
**File**: `internal/iface/iface.go`

Update `Freeze()` method:
- Serialize Types in sorted order (by name)
- Serialize Constructors in sorted order (by ctor name)
- Include in SHA256 digest for stability checking

#### 3. Import Resolver - Handle Type/Ctor Imports (~40 LOC)
**File**: `internal/link/resolver.go`

When resolving `import std/option (Option, Some, None)`:
- Look in `itf.Types["Option"]` - bind as type alias
- Look in `itf.Constructors["Some"]` - bind as value constructor
- Look in `itf.Constructors["None"]` - bind as value constructor

Add export kind checking:
- `ExportType` - goes to type namespace
- `ExportCtor` - goes to value namespace (like functions)
- `ExportValue` - existing function exports

### Tests
1. **Unit test**: `internal/iface/builder_test.go`
   - Test type export extraction
   - Test constructor export extraction
   - Verify deterministic ordering

2. **Integration test**: Create `tests/import_adt.ail`
   ```ailang
   module tests/import_adt
   import stdlib/std/option (Option, Some, None, getOrElse)
   
   export pure func demo() -> int {
     getOrElse(Some(7), 0)
   }
   ```
   Expected: Type-checks and returns 7

3. **Golden test**: Regenerate stdlib interface digests
   ```bash
   mkdir -p goldens/stdlib
   for m in option result list; do
     ailang iface --module stdlib/std/$m --json | shasum -a 256 > goldens/stdlib/$m.sha256
   done
   ```

### Acceptance Criteria
- [ ] `import stdlib/std/option (Option, Some, None)` type-checks
- [ ] stdlib/std/list.ail compiles (imports Option from option.ail)
- [ ] examples/option_demo.ail runs and outputs correct result
- [ ] `make test-stdlib-freeze` target works (SHA256 matching)
- [ ] No regressions in existing tests

---

## Part B: Seed Builtin Signatures into Type Env (1-2 hours)

### Changes Required

#### 1. Builtin Type Registry (~100 LOC)
**File**: `internal/link/builtin_module.go` (or new file `internal/types/builtins.go`)

Create function to expose builtin signatures:
```go
func BuiltinTypes() map[string]types.Type {
    m := map[string]types.Type{}
    
    // Pure string builtins
    m["_str_len"] = types.TFunc2{
        Params: []types.Type{types.TString{}},
        Result: types.TInt{},
        EffectRow: nil, // pure
    }
    m["_str_slice"] = types.TFunc2{
        Params: []types.Type{types.TString{}, types.TInt{}, types.TInt{}},
        Result: types.TString{},
        EffectRow: nil,
    }
    // ... _str_upper, _str_lower, _str_trim, _str_compare, _str_find
    
    // IO builtins (effectful)
    ioRow := &types.Row{
        Kind: types.EffectRow,
        Labels: map[string]types.Type{"IO": types.TUnit{}},
        Tail: nil, // closed
    }
    m["_io_print"] = types.TFunc2{
        Params: []types.Type{types.TString{}},
        Result: types.TUnit{},
        EffectRow: ioRow,
    }
    m["_io_println"] = types.TFunc2{
        Params: []types.Type{types.TString{}},
        Result: types.TUnit{},
        EffectRow: ioRow,
    }
    m["_io_readLine"] = types.TFunc2{
        Params: nil,
        Result: types.TString{},
        EffectRow: ioRow,
    }
    
    return m
}
```

#### 2. Seed Type Environment (~20 LOC)
**File**: `internal/pipeline/pipeline.go`

Before type-checking modules:
```go
func buildInitialTypeEnv() *types.TypeEnv {
    env := types.NewTypeEnv()
    
    // Add builtins to global scope
    for name, typ := range link.BuiltinTypes() {
        env.Bind(name, typ)
    }
    
    return env
}

// Use this env as parent for all module type-checking
```

#### 3. Update Pipeline Integration (~10 LOC)
Ensure every module's type-checking session starts with builtin env.

### Tests
1. **Unit test**: `internal/types/builtins_test.go`
   - Verify all 10 builtins have correct signatures
   - Verify IO builtins have effect rows
   - Verify string builtins are pure (nil effect row)

2. **Integration test**: `tests/string_wrappers.ail`
   ```ailang
   module tests/string_wrappers
   
   export pure func myLength(s: string) -> int { 
     _str_len(s) 
   }
   ```
   Expected: Type-checks successfully

3. **Integration test**: `tests/io_effects.ail`
   ```ailang
   module tests/io_effects
   
   export func greet() -> () ! {IO} {
     _io_println("hello")
   }
   ```
   Expected: Type-checks with IO effect

### Acceptance Criteria
- [ ] stdlib/std/string.ail compiles (calls _str_len, _str_slice, etc.)
- [ ] stdlib/std/io.ail compiles (calls _io_print with ! {IO})
- [ ] examples/effects_basic.ail runs and prints "hello effects"
- [ ] REPL `:type _str_len` shows `string -> int`
- [ ] REPL `:type _io_println` shows `string -> () ! {IO}`
- [ ] No regressions in existing tests

---

## Part C: Add `extern` Keyword (Optional, 30-45 min)

### Changes Required

#### 1. Lexer - Add EXTERN Token (~5 LOC)
**File**: `internal/lexer/token.go`
```go
EXTERN = "EXTERN"
```

**File**: `internal/lexer/lexer.go`
```go
keywords["extern"] = token.EXTERN
```

#### 2. Parser - Parse extern Declarations (~40 LOC)
**File**: `internal/parser/parser.go`

Allow:
```ailang
extern pure func _str_len(s: string) -> int
extern func _io_println(s: string) -> () ! {IO}
```

Parse as function declaration with:
- No body (nil)
- Mark as "external" (new flag in AST)

#### 3. Elaboration - Handle extern (~10 LOC)
Treat extern declarations as type-only bindings.
They introduce symbols in the type environment but don't elaborate to Core (runtime looks up in builtins).

### Tests
1. **Parser test**: `internal/parser/extern_test.go`
   - Parse `extern pure func foo() -> int`
   - Parse `extern func bar() -> () ! {IO}`
   - Verify AST structure

2. **Integration**: Update stdlib/std/string.ail to use extern
   ```ailang
   extern pure func _str_len(s: string) -> int
   export pure func length(s: string) -> int { _str_len(s) }
   ```

### Acceptance Criteria
- [ ] `extern` keyword recognized by lexer
- [ ] extern function declarations parse without errors
- [ ] stdlib modules can use extern for documentation
- [ ] Runtime still resolves to Go builtins correctly

---

## Implementation Order

### Hour 1-2: Part A - Export System
1. Add TypeExport and ConstructorScheme structs to interface
2. Modify BuildInterface to extract types and constructors
3. Update Freeze() to serialize types/constructors
4. Test with stdlib/std/option.ail

### Hour 3-4: Part A - Import Resolution  
1. Update resolver to handle ExportType and ExportCtor
2. Wire type name imports to type namespace
3. Wire constructor imports to value namespace
4. Test cross-module import: list.ail importing Option

### Hour 5: Part B - Builtin Signatures
1. Create BuiltinTypes() registry
2. Seed type environment before module checking
3. Test stdlib/std/string.ail and stdlib/std/io.ail

### Hour 6 (Optional): Part C - extern Keyword
1. Add EXTERN token to lexer
2. Parse extern declarations
3. Update stdlib modules to use extern

---

## End-to-End Verification

### Run Full Stdlib Test Suite
```bash
# Type-check all stdlib modules
ailang check stdlib/std/option.ail
ailang check stdlib/std/result.ail
ailang check stdlib/std/list.ail    # Now works! (imports Option)
ailang check stdlib/std/string.ail  # Now works! (calls _str_*)
ailang check stdlib/std/io.ail      # Now works! (calls _io_*)

# Run examples
ailang run examples/option_demo.ail    # Output: 183 (42+99+42)
ailang run examples/effects_basic.ail  # Output: hello effects
ailang run examples/stdlib_demo.ail    # Output: SUM=20\nUP=HI\nOPT=42

# Verify golden outputs match
bash scripts/verify-examples.sh

# Check interface stability
make test-stdlib-freeze
```

### Success Metrics
- [ ] All 5 stdlib modules type-check ✅
- [ ] All 4 example programs run with correct output ✅
- [ ] Golden file tests pass ✅
- [ ] Interface freeze tests pass ✅
- [ ] No test regressions ✅
- [ ] REPL shows effect annotations for IO functions ✅

---

## Risks & Mitigations

**Risk 1**: Name collisions (user defines `Some` locally)
- **Mitigation**: Resolver should prefer local scope over imports (standard shadowing)

**Risk 2**: Digest instability
- **Mitigation**: Sort all keys (Types, Constructors) before serialization
- **Test**: Double-freeze equality test

**Risk 3**: Effect row mismatch
- **Mitigation**: Ensure IO builtins use closed row (nil Tail)
- **Test**: Verify ! {IO} appears in type signatures

**Risk 4**: Breaking existing examples
- **Mitigation**: Run full test suite after each part
- **Rollback**: Git commits for each part separately

---

## Commit Strategy

1. `feat: add type and constructor exports to interface builder`
2. `feat: wire import resolver to handle type/ctor imports`  
3. `feat: seed builtin signatures into type environment`
4. `feat(optional): add extern keyword for external declarations`
5. `feat: add stdlib modules (option, result, list, string, io)`
6. `test: add examples and golden files for stdlib`
7. `ci: add stdlib interface freeze checks to Makefile`

---

## Ready to Execute?

All code locations identified, patches sketched, tests planned. Estimated 3-5 hours to complete both blockers and ship working stdlib.
# M-S1: Stdlib Implementation Sprint Plan

## Status: ✅ Parts A & B COMPLETE (2025-10-01)

**Time**: 1 day (8 hours) - On schedule
**Code Added**: ~700 LOC across 7 files
**Tests**: All passing, no regressions

## Summary
Implement 5 core stdlib modules in native AILANG (~400 LOC .ail + ~600 LOC infrastructure/tests) over 1-2 days to dogfood the language and prove the parser/type system work end-to-end.

## ✅ COMPLETED: Part A - Export System for Types and Constructors

### What Was Fixed
The import system now correctly resolves types, constructors, and functions from modules.

**Implementation** (~400 LOC):
1. **Loader Enhancement** ([internal/loader/loader.go](../../internal/loader/loader.go))
   - Added `Types` and `Constructors` maps to `LoadedModule` struct
   - Created `buildTypes()` to extract type declarations from AST
   - Updated `GetExport()` to recognize types and constructors (returns `nil, nil` for non-functions)

2. **Elaborator Updates** ([internal/elaborate/elaborate.go](../../internal/elaborate/elaborate.go))
   - Modified import resolution to skip types/constructors (handled later in pipeline)
   - Added `AddBuiltinsToGlobalEnv()` to inject builtin functions into elaborator scope

3. **Interface Builder** ([internal/iface/iface.go](../../internal/iface/iface.go), [internal/iface/builder.go](../../internal/iface/builder.go))
   - Added `Types` map to `Iface` struct with `TypeExport` entries
   - Enhanced `BuildInterfaceWithTypesAndConstructors()` to extract types from AST
   - Constructors properly added to interface from `AlgebraicType.Constructors` field

4. **Pipeline Integration** ([internal/pipeline/pipeline.go](../../internal/pipeline/pipeline.go))
   - Updated import resolution to check `GetType()` and `GetConstructor()`
   - Auto-inject `$builtin` module exports into all modules' `externalTypes`
   - Added `AddBuiltinsToGlobalEnv()` calls for both REPL and module compilation

5. **Module Linker** ([internal/link/module_linker.go](../../internal/link/module_linker.go))
   - Enhanced `BuildGlobalEnv()` to handle types and constructors
   - Added separate code paths for types (skip env), constructors (`$adt` mapping), and functions
   - Improved error reporting to list available types and constructors

**Test Results**:
- ✅ Function imports work ([examples/test_import_func.ail](../../examples/test_import_func.ail))
- ✅ Constructor imports work ([examples/test_import_ctor.ail](../../examples/test_import_ctor.ail))
- ✅ Type imports work (type-checked successfully)
- ✅ All existing tests pass (no regressions)

**Remaining Work**:
- ⏳ Constructor evaluation (`$adt` runtime implementation - deferred until full stdlib)
- ⏳ `Freeze()` serialization of types/constructors (deferred until evaluation works)

---

## ✅ COMPLETED: Part B - Builtin Type Visibility

### What Was Fixed
String and IO primitive builtins (`_str_*`, `_io_*`) are now visible to all modules.

**Implementation** (~300 LOC):
1. **Builtin Module Enhancement** ([internal/link/builtin_module.go](../../internal/link/builtin_module.go))
   - Added `handleStringPrimitive()` for: `_str_len`, `_str_slice`, `_str_compare`, `_str_find`, `_str_upper`, `_str_lower`, `_str_trim`
   - Added `handleIOBuiltin()` for: `_io_print`, `_io_println`, `_io_readLine`
   - Proper effect row handling with `! {IO}` for effectful operations

2. **Pipeline Integration** ([internal/pipeline/pipeline.go](../../internal/pipeline/pipeline.go))
   - `$builtin` module automatically injected into all modules' `externalTypes`
   - Builtins available in global environment during elaboration
   - Works for both REPL and file compilation modes

**Test Results**:
- ✅ `stdlib/std/string.ail` type-checks successfully
- ✅ All 7 string functions exported correctly
- ⏳ `stdlib/std/io.ail` has parse errors (inline function syntax not yet supported)
- ✅ All existing tests pass (no regressions)

---

## Day 1: Core Stdlib Modules (6-8 hours) - 📋 IN PROGRESS

### Prerequisites ✅ COMPLETE
- ✅ **Part A**: Type/constructor imports work
- ✅ **Part B**: Builtin primitives visible

### Remaining Tasks
1. **stdlib/std/option.ail** (~50 LOC): map, flatMap, getOrElse, isSome, filter
2. **stdlib/std/result.ail** (~70 LOC): map, mapErr, flatMap, isOk, unwrap
3. **stdlib/std/io.ail** (~20 LOC): print, println, readLine, debug with `! {IO}` effects
4. **stdlib/std/list.ail** (~180 LOC): length, head, tail, reverse, concat, zip (+ optional map/filter/fold)

**Blockers Resolved**:
- ✅ Import system works end-to-end
- ✅ Builtins available globally
- ⏳ Need `$adt` runtime for constructor evaluation (can defer to Part C)

## Day 2: Infrastructure & Polish (6-8 hours) - 📋 TODO
1. **stdlib/std/string.ail** (~40 LOC): ✅ Type-checks successfully (ready to use)
2. **Verify builtins** (1-2 hours): ✅ String and IO primitives complete
3. **CI infrastructure** (2-3 hours): `make test-stdlib-freeze`, `make verify-examples`
4. **Tests & examples** (2-3 hours): Golden files, stdlib_demo.ail, effects_basic.ail

## Timeline
- **Day 1 (Blockers A & B)**: ✅ COMPLETE (8 hours actual)
- **Day 2 (Stdlib modules)**: 📋 6-8 hours remaining
- **Day 3 (Infrastructure)**: 📋 6-8 hours remaining
- **Total**: 1 day complete, 1.5-2 days remaining

## Success Criteria
- ✅ ~~Type and constructor imports work~~ **DONE**
- ✅ ~~Builtin primitives accessible~~ **DONE**
- ⏳ All 5 stdlib modules parse and type-check
- ⏳ `import std/option (Some, None)` works from external files
- ⏳ Pattern matching in functions proven across all modules
- ✅ Effect annotations work: `! {IO}` displays in type signatures **DONE**
- ⏳ At least 5 golden test files passing
- ⏳ Examples passing count increases to 35+ (from 23)

## Files Modified (Parts A & B)
1. [internal/loader/loader.go](../../internal/loader/loader.go) - Type/constructor extraction from modules
2. [internal/elaborate/elaborate.go](../../internal/elaborate/elaborate.go) - Builtin environment setup
3. [internal/iface/iface.go](../../internal/iface/iface.go) - Type export data structures
4. [internal/iface/builder.go](../../internal/iface/builder.go) - AST type extraction
5. [internal/pipeline/pipeline.go](../../internal/pipeline/pipeline.go) - Builtin injection and import resolution
6. [internal/link/module_linker.go](../../internal/link/module_linker.go) - Type/constructor linking
7. [internal/link/builtin_module.go](../../internal/link/builtin_module.go) - String/IO builtin type signatures

## Test Files Created
- [examples/test_import_ctor.ail](../../examples/test_import_ctor.ail) - Constructor import test
- [examples/test_import_func.ail](../../examples/test_import_func.ail) - Function import test
- [examples/test_use_constructor.ail](../../examples/test_use_constructor.ail) - Constructor usage test

## Risk Mitigation
- ✅ Import system proven working before stdlib implementation
- ✅ Builtins verified accessible
- ⏳ Start with option.ail (smallest, no effects)
- ⏳ Test incrementally with `ailang check`
- ⏳ Document any limitations (e.g., missing `extern` support)

## Ready for Day 2?
✅ Yes! All blockers resolved. Import system works, builtins are accessible. Ready to implement stdlib modules in native AILANG.
# AILANG v0.1.0 MVP Roadmap

## Executive Summary

This document synthesizes feedback from Claude Sonnet 4.5 and GPT-5, assesses current implementation status (v0.0.7), and defines a focused MVP for v0.1.0 that proves AILANG's "one-shot + secure by construction" thesis.

**Primary Goal**: Run single `.ail` files hermetically with explicit effects, resource budgets, and reproducible artifacts.

---

## Current Implementation Status (v0.0.9 + M-P3 + M-P4 + M-S1 Parts A & B + Blockers Fixed)

### 🆕 Recent Progress (October 1, 2025)

**✅ M-S1 BLOCKERS FIXED: Cross-Module Constructors & Multi-Statement Functions (~224 LOC)**
- ✅ **Blocker 1 FIXED**: Cross-module constructor resolution (~74 LOC, 2 hours)
  - Constructor factory types now added to `externalTypes` during import
  - `examples/option_demo.ail` now type-checks successfully
  - Can use imported constructors: `import std/option (Some, None)`
- ✅ **Blocker 2 FIXED**: Multi-statement function bodies (~150 LOC, 3 hours)
  - Added `Block` AST node for semicolon-separated expressions
  - Parser now supports: `{ let x = 1; let y = 2; x + y }`
  - Elaboration converts blocks to nested `Let` expressions
  - `examples/block_demo.ail` demonstrates working multi-statement functions
- ✅ **5 files modified**: pipeline, ast, parser, elaborate, examples
- ✅ **All tests passing**: No regressions, both blockers resolved
- ✅ **READY FOR STDLIB**: All prerequisites complete! Can now implement realistic stdlib modules

**✅ M-S1 Parts A & B COMPLETE: Import System & Builtin Visibility (~700 LOC)**
- ✅ **Type/constructor imports**: `import std/option (Option, Some, None)` now works
- ✅ **Dual import resolution**: Elaboration phase + linking phase both handle types/constructors
- ✅ **Builtin visibility**: String (_str_*) and IO (_io_*) primitives globally available
- ✅ **Effect tracking**: IO builtins properly annotated with `! {IO}` in type system
- ✅ **7 files modified**: loader, elaborator, interface builder, pipeline, linker (2 files)
- ✅ **stdlib/std/string.ail**: Type-checks successfully with all 7 exports
- ✅ **All tests passing**: No regressions, import system proven working

**✅ M-P4 COMPLETE: Effect System (~1,060 LOC)**
- ✅ **Effect syntax parsing**: `func f() -> int ! {IO, FS}` works
- ✅ **Lambda effects**: `\x. body ! {IO}` syntax supported
- ✅ **Effect validation**: 8 canonical effects (IO, FS, Net, Clock, Rand, DB, Trace, Async)
- ✅ **Effect elaboration**: AST strings → typed effect rows with deterministic sorting
- ✅ **Type checking integration**: Effect annotations thread to TFunc2.EffectRow
- ✅ **46 tests passing**: 17 parser tests + 29 elaboration tests
- ✅ **Foundation complete**: Ready for runtime effect enforcement in v0.2.0
- 📝 **Deferred**: Examples, REPL display formatting, pure function verification (polish items)

**✅ TYPE SYSTEM CONSOLIDATION COMPLETE (~1 hour)**
- ✅ **Unified type system**: All old types (TFunc, TVar) migrated to new system (TFunc2, TVar2)
- ✅ **Builtin functions**: Converted 8 TFunc → TFunc2 with `EffectRow: nil` for pure operations
- ✅ **Type variables**: Converted 4 TVar → TVar2 with `Kind: Star` for proper kind tracking
- ✅ **Unifier cleanup**: Removed compatibility fallback code (unification.go lines 174-182)
- ✅ **All tests passing**: Full test suite verified after migration
- ✅ **ADT example working**: `examples/adt_simple.ail` outputs `42` correctly
- ✅ **Clean foundation**: Ready for M-P4 (Effect System) with consistent TFunc2/TVar2 usage

**✅ M-P3 COMPLETE: Pattern Matching + ADT Runtime (~600 LOC)**
- ✅ **TaggedValue runtime**: Constructor representation with TypeName, CtorName, Fields
- ✅ **$adt synthetic module**: Factory functions auto-generated from type declarations
- ✅ **Type declaration elaboration**: `type Option[a] = Some(a) | None` works end-to-end
- ✅ **Constructor expressions**: Both nullary (`None`) and non-nullary (`Some(42)`) work
- ✅ **Constructor pattern matching**: Full destructuring with variable binding
- ✅ **Pipeline integration**: TFunc2/TVar2 for type checking, monomorphic result types
- ✅ **Working examples**: `examples/adt_simple.ail` demonstrates Option type with pattern matching

**Example that works**:
```ailang
type Option[a] = Some(a) | None

match Some(42) {
  Some(n) => n,
  None => 0
}
-- Output: 42 ✅
```

### 🆕 Recent Progress (September 30, 2025)

**REPL Fixed**:
- ✅ Fixed "Empty expression" bug by updating `Elaborate()` to handle `prog.File.Statements`
- ✅ Added `Intrinsic` support to ANF verifier for arithmetic operators
- ✅ Integrated `OpLowering` pass into REPL pipeline
- ✅ All basic expressions now work: `42`, `1 + 2`, `"hello" ++ "world"`, etc.

**Module System Verified**:
- ✅ `func` declarations work in files (proven by test_export_func.ail)
- ✅ `module`/`import` statements work for basic cases
- ✅ Export/import mechanism functional

**Metrics Updated**:
- Corrected test coverage from inflated 31.3% to actual 24.9%
- Updated LOC count from 7,860 to accurate 23,384
- Now ~24,000 LOC with M-P3 additions
- Identified critical gaps: parser (0% tests), eval (14.9%), types (15.4%)

### ✅ What We Have (Working)

**Currently at ~25% test coverage with ~24,000 LOC** *(Updated 2025-10-01)*

1. **Pattern Matching + ADTs** (M-P3 Complete - Oct 2025)
   - ✅ Type declarations: `type Option[a] = Some(a) | None`
   - ✅ Constructor expressions: `Some(42)`, `None`
   - ✅ Pattern matching: literals, tuples, constructors, variables, wildcards
   - ✅ TaggedValue runtime representation
   - ✅ $adt synthetic module with factory functions
   - ✅ Full pipeline integration (parsing → elaboration → type checking → evaluation)
   - ⚠️ Known limitation: Monomorphic result types (Option vs Option[Int])
   - ⚠️ Missing: Exhaustiveness checking, guard evaluation

2. **Type System** (Foundation - 15.4% coverage)
   - Hindley-Milner inference with let-polymorphism (~6,815 LOC)
   - Type classes: Num, Eq, Ord, Show with dictionary-passing
   - Row-polymorphic records with principal row unification
   - Value restriction for sound polymorphism
   - Kind system (Effect, Record, Row)
   - Linear capability capture analysis
   - ✅ **Unified type system**: TFunc2/TVar2 consistently used (Oct 1, 2025)

3. **Module System** (v0.0.6-v0.0.7)
   - Path resolution (relative, stdlib, project) (~405 LOC)
   - Dependency management with cycle detection (~607 LOC)
   - Module caching (thread-safe, concurrent)
   - Import conflict detection (IMP011)
   - **Structured error reporting** (v0.0.7, ~680 LOC)
   - **Golden file testing** (byte-for-byte reproducibility)
   - **CLI JSON output** (`--json`, `--compact` flags)

3. **Evaluation** (Working - 14.9% coverage)
   - Tree-walking interpreter (~3,362 LOC)
   - Lambda expressions with closures
   - Arithmetic, strings, conditionals, let bindings
   - Records (creation + field access)
   - Lists, built-ins (print, show, toText)

4. **REPL** (NOW Operational - Fixed v0.0.8)
   - Professional interactive REPL (~1,351 LOC)
   - **Recent fix**: Elaborate() now handles bare expressions, Intrinsic support added, OpLowering pass integrated
   - Arrow key history, tab completion, persistent history
   - Type class resolution with dictionary-passing
   - Module import system
   - Rich diagnostic commands (`:type`, `:instances`, `:dump-core`)
   - Auto-imports std/prelude

5. **Parser** (Nearly Complete - 0% test coverage ⚠️)
   - Recursive descent + Pratt parsing (~1,436 LOC)
   - ✅ Expressions, let bindings, if-then-else
   - ✅ Binary/unary operators (spec-compliant precedence)
   - ✅ Lambda expressions (`\x.` syntax, currying)
   - ✅ Record field access (correct precedence)
   - ✅ Module declarations, import statements
   - ✅ **Pattern matching** (M-P3: parsed AND evaluated)
   - ✅ **Type declarations** (M-P3: ADT syntax working)
   - ✅ **Tuples** (M-P3: tuple expressions and patterns)
   - ❌ `?` operator, effect handlers not implemented yet

6. **AI-First Features** (v0.0.4-v0.0.7)
   - Schema registry (versioned JSON, ~176 LOC, 88.5% coverage)
   - Error JSON encoder (~192 LOC, 50.0% coverage ⚠️)
   - Test reporter (~95 LOC, 95.7% coverage)
   - Effects inspector stub (~41 LOC)
   - Golden test framework (~310 LOC)
   - **Note**: Coverage ranges 50-95.7%, not 100%

7. **Infrastructure**
   - Lexer (~935 LOC, 57.0% coverage)
   - Error taxonomy (60+ error codes)
   - Manifest system (~390 LOC)
   - CI/CD with automated testing
   - Example verification system

### ⚠️ What's Broken/Missing

**Parser Issues** (MOSTLY FIXED in v0.0.8-0.0.9, M-P5 Oct 1):
- ✅ `func` declarations work in files (test_export_func.ail passes)
- ✅ `module`/`import` statements work (basic cases proven)
- ✅ **`type` definitions work** (M-P3: ADTs fully supported)
- ✅ **Generic type parameters** work: `func map[a, b]` syntax fixed (Oct 1, 2025)
- ✅ **Pattern matching in function bodies** - FIXED (M-P5, Oct 1, 2025)
- ❌ Test/property syntax broken
- ❌ `?` operator not implemented

**Parser Discovery & Fixes (October 1, 2025)** - Two Issues Found and Fixed:
1. ✅ **FIXED**: Generic type parameter bug - `parseTypeParams()` left parser at wrong position
   - **Fix**: Adjusted token positioning after type param parsing (lines 554-582 in parser.go)
   - **Impact**: `export func map[a, b](f: (a) -> b, xs: [a]) -> [b]` now parses correctly
   - **Time**: 5 minutes to fix, verified with test case

2. ✅ **FIXED**: Pattern matching in function bodies (M-P5)
   - **Root Cause**: `parseListPattern()` was unimplemented stub returning nil
   - **Fix**: Implemented full list pattern parsing (~260 LOC total)
   - **Impact**: List patterns now work: `[]`, `[x]`, `[x, ...rest]`
   - **Time**: ~6 hours (investigation + implementation + tests)
   - **Tests**: Added `internal/parser/func_pattern_test.go` with 4 test cases, all passing

**Pattern Matching Limitations** (M-P3):
- ⚠️ Let bindings with constructors have elaboration bug
- ⚠️ No exhaustiveness checking yet
- ⚠️ Guard evaluation not implemented (parsed but ignored)

**Not Started**:
- ❌ Effect system (no tracking/inference) - **Next milestone**
- ❌ Quasiquotes
- ❌ CSP/channels

### 📊 Current Metrics (v0.0.9 + M-P3 - Updated 2025-10-01)
- **Test Coverage**: ~25% (slightly up with M-P3 additions)
- **Examples**: ~23 passing (adt_simple.ail now works), ~37 failing (60 total)
- **Production Code**: ~24,000 lines (~600 LOC added in M-P3)
- **Well-tested**: test (95.7%), manifest (89.9%), schema (88.5%), module (67.7%)
- **Needs tests**: parser (0%), eval (14.9%), types (15.4%), errors (50.0%)
- **New working**: ADT runtime, pattern matching, type declarations

---

## Feature Analysis: What Both AIs Agreed On

### Priority Matrix

| Feature | V4.0 Rating | GPT-5 Priority | Current Status | v0.1.0 MVP? |
|---------|-------------|----------------|----------------|-------------|
| **Effect System** | ⭐⭐⭐⭐⭐ | Critical | ✅ **Type-level complete** | ✅ **Core** |
| **Capability Budgets** | ⭐⭐⭐⭐⭐ | Critical | ❌ None | ✅ **Core** |
| **@oneshot Runner** | N/A | Critical | ❌ None | ✅ **Core** |
| **Refinement Types** | ⭐⭐⭐⭐⭐ | High | ❌ None | ✅ **Starter set** |
| **Effect Composition** | ⭐⭐⭐⭐ | High | ❌ None | ✅ **Basic** |
| **Linear/Affine Types** | N/A | High | ❌ None | ⬜ v0.2.0 |
| **Info-Flow Labels** | N/A | High | ❌ None | ⬜ v0.2.0 |
| **Semantic Annotations** | ⭐⭐⭐⭐⭐ | Medium | ❌ None | ⬜ v0.2.0 |
| **Session Types** | N/A | Medium | ❌ None | ⬜ v0.3.0 |
| **Policy DSL** | N/A | Medium | ❌ None | ⬜ v0.2.0 |
| **Gradual Typing** | ⭐⭐⭐⭐ | Low | ❌ None | ⬜ v0.3.0 |

---

## v0.1.0 MVP Scope (REVISED - Conservative & Achievable)

### Design Philosophy (Updated 2025-09-30)

**Revised Goal**: Build a **solid foundation** with comprehensive testing rather than rushing all features.

**Core Principle**: Ship features that are **well-tested and production-ready**, not features that are "mostly done."

**What v0.1.0 Proves**:
1. **Parser is robust** - 80%+ test coverage, no "REPL vs file" discrepancies
2. **Effects are tracked** - Type system enforces effect discipline
3. **Type system is complete** - ADTs, pattern matching, type classes all work
4. **Ready for v0.2.0** - Solid foundation for runtime features

### What's In (Must-Have)

#### A. Parser Testing & Fixes (HIGHEST PRIORITY - 5 days)

**Status**: ⚠️ **PARTIALLY COMPLETE** - ADT support done in M-P3, parser tests still needed

**Remaining Tasks**:
1. **Write 100+ parser tests** (3 days) - **STILL NEEDED**
   - Expression parsing (arithmetic, lambdas, let, if-then-else)
   - Module/import parsing
   - Function declarations
   - Type definitions
   - Pattern matching syntax
   - Error recovery

2. ~~**Add ADT support**~~ ✅ **COMPLETE in M-P3**
   - ✅ Sum types: `type Option[a] = Some(a) | None`
   - ✅ Product types (via records)
   - ✅ Recursive types
   - ✅ Tuple syntax: `(1, "hello", true)` with type `(int, string, bool)`

**Lines**: ~500 new (tests) still needed
**Acceptance**:
- ⚠️ Parser test coverage still 0% - **HIGH PRIORITY**
- ✅ ADT examples parse and type-check correctly (M-P3)
- ✅ No more "works in REPL, fails in files" (M-P3)

#### B. Pattern Matching Evaluation (3 days)

**Status**: ✅ **COMPLETE in M-P3**

~~**Tasks**:~~
1. ~~**Implement evaluation**~~ ✅ **DONE**
   - ✅ Literal patterns: `42`, `"hello"`, `true`
   - ✅ Constructor patterns: `Some(x)`, `Cons(head, tail)`
   - ✅ Tuple patterns: `(x, y, z)`
   - ✅ Wildcard/variable patterns: `_`, `x`

2. **Exhaustiveness checking** (1 day) - ⚠️ **DEFERRED to polish phase**
   - Warn on non-exhaustive matches
   - Suggest missing patterns

**Lines**: ~600 delivered in M-P3 (eval + ADT runtime)
**Acceptance**:
```ailang
type Option[a] = Some(a) | None

match option {
  Some(x) => x,
  None => 0
}
```
✅ Works correctly! (exhaustiveness warnings still TODO)

#### C. Effect System - Type Level Only (4 days) - ✅ **COMPLETE**

**Goal**: Track effects in types, **NO runtime enforcement yet** ✅ ACHIEVED

**Core Effects**:
```ailang
type Effect = IO | FS | Net | Clock | Rand
```

**Function Signatures**:
```ailang
func readFile(path: string) -> Result[string, string] ! {FS}
func httpGet(url: string) -> Result[string, string] ! {Net}
func print(s: string) -> () ! {IO}
```

**Effect Inference**:
```ailang
// Compiler infers {FS, Net} from body
func process(path: string) -> Result[string] ! {FS, Net} {
  let config = readFile(path)?
  let data = httpGet(config)?
  Ok(data)
}
```

**Implementation**:
- Effect syntax in parser (~100 LOC)
- Effect tracking in type checker (~300 LOC)
- Effect propagation (~200 LOC)
- Export signature enforcement (~100 LOC)

**Lines**: ~1,060 total (700 LOC code + 360 LOC tests)
**Acceptance**: ✅ ALL ACHIEVED
- ✅ Effect syntax parses correctly
- ✅ Effect annotations thread through compilation pipeline
- ✅ Type checker integrates effect rows in TFunc2
- ✅ 46 tests passing (17 parser + 29 elaboration)
- ✅ **NO runtime enforcement** (correctly deferred to v0.2.0)

#### D. Parser Enhancement: Pattern Matching in Function Bodies (1-2 days) - ✅ **COMPLETE**

**Status**: ✅ **FIXED** (Oct 1, 2025 - completed in ~6 hours)

**Discovery Summary**:
- ✅ **Fixed**: Generic type parameter parsing (`func map[a, b]`) - 1-line fix COMPLETE
- ✅ **Fixed**: Pattern matching now works inside function bodies
- ✅ **Works**: Pattern matching at top-level (proven by `adt_simple.ail`)
- ✅ **Works**: Pattern matching inside `export func` bodies

**Root Cause**: `parseListPattern()` was a stub returning `nil`. When `parsePattern()` encountered `[` in a match expression, it called this unimplemented function, causing parser failures.

**Solution**: Implemented complete list pattern parsing:
1. **Parser** (~75 LOC): Full `parseListPattern()` implementation
   - Handles empty lists: `[]`
   - Handles fixed-length lists: `[x, y, z]`
   - Handles spread patterns: `[x, ...rest]`
   - Error handling for spread without identifier

2. **Elaboration** (~20 LOC): List pattern to Core AST transformation
   - Maps `ast.ListPattern` → `core.ListPattern`
   - Handles elements and tail (spread) patterns

3. **Type Checking** (~60 LOC): List pattern type inference
   - Extracts element type from list scrutinee
   - Type checks each element pattern
   - Type checks tail pattern as full list type

4. **TypedAST** (~15 LOC): Added `TypedListPattern` node

**Tests Added** (~90 LOC):
- `internal/parser/func_pattern_test.go` created
- 3 main test cases + 1 error case
- All tests passing ✅
- No regressions in existing tests ✅

**Lines**: ~260 LOC total (75 parser + 20 elaboration + 60 types + 15 typedast + 90 tests)

**Acceptance**: ✅ ALL CRITERIA MET
- ✅ All list patterns parse inside functions: `[]`, `[x]`, `[x, ...rest]`, `[x, y, ...rest]`
- ✅ Simple stdlib modules parse without errors (isEmpty, head, length verified)
- ✅ Pattern matching parity: top-level == function body
- ✅ All tests pass (parser + full suite)
- ✅ Error handling for malformed patterns (spread without ident)

---

#### E. Minimal Stdlib in AILANG (1 day) - ✅ **PREREQUISITES COMPLETE** (M-S1 A & B)

**Goal**: Dogfood AILANG by implementing stdlib in .ail files (NOT Go builtins)

**Prerequisites**: ✅ ALL COMPLETE (October 1, 2025)
- ✅ Pattern matching in functions (M-P5)
- ✅ Type/constructor imports work (M-S1 Part A)
- ✅ Builtin primitives visible (M-S1 Part B)

**Modules** (code already written, ready to drop in):
```ailang
std_list       -- map, filter, fold, length, head, tail (~180 LOC)
std_string     -- length, join, substring (~40 LOC, uses builtins)
std_option     -- Option[a], map, flatMap, getOrElse (~50 LOC)
std_result     -- Result[a,e], map, flatMap, isOk, unwrap (~70 LOC)
std_io         -- print, println, debug with ! {IO} effects (~20 LOC)
```

**Builtins Already Implemented** (Oct 1, 2025):
- ✅ String primitives: `_str_len`, `_str_slice`, `_str_compare`, `_str_find`, `_str_upper`, `_str_lower`, `_str_trim` (~150 LOC)
- ✅ IO primitives: `_io_print`, `_io_println`, `_io_readLine` with `IsPure: false`
- ✅ All builtins compile and are ready to use

**Implementation**: ~360 LOC stdlib + ~200 LOC tests = ~560 LOC total
**Acceptance**:
- [ ] All stdlib modules parse and type-check
- [ ] `import std_list (map, filter)` works
- [ ] Effect annotations verified: `println` has `! {IO}`
- [ ] Example programs use stdlib functions

#### E. Fix Examples & Documentation (2 days)

**Goal**: Make existing examples work and document accurately

**Tasks**:
1. **Fix broken examples** (~1 day)
   - Update to use new ADT syntax
   - Add pattern matching examples
   - Add effect-annotated examples

2. **Update documentation** (~1 day)
   - README.md with accurate metrics
   - CLAUDE.md with current status
   - Example comments explaining effects

**Lines**: ~500 modified (examples + docs)
**Acceptance**: >35 examples passing (up from 22)

---

### 🚫 EXPLICITLY DEFERRED to v0.2.0

**These features are OUT OF SCOPE for v0.1.0:**

#### Refinement Types
- **Why defer**: Requires SMT solver or extensive runtime guards
- **Complexity**: ~400-800 LOC + external dependencies
- **Not essential** for core thesis proof

#### Capability Budgets
- **Why defer**: Requires runtime instrumentation
- **Complexity**: ~600-1000 LOC + testing overhead
- **Dependencies**: Needs effect runtime first

#### Effect Composition (Runtime)
- **Why defer**: Needs runtime effect handlers (retry, timeout)
- **Note**: Syntax can be added in parser, implementation deferred

#### @oneshot Hermetic Bundler
- **Why defer**: Complex tooling (bundling, SBOM, signing)
- **For v0.1.0**: Basic `ailang run file.ail` is sufficient
- **For v0.2.0**: Add hermetic execution, signing, SBOM generation

---

### Timeline Summary

**Total Time**: ~~13 days~~ ~~10 days~~ ~~9.5 days~~ ~~6.5 days~~ ~~8 days~~ ~~6 days~~ **~3 days remaining** (~0.75 weeks) - Way ahead of schedule!

| Week | Task | Days | Status |
|------|------|------|--------|
| ~~Week 1~~ | ~~Parser tests + ADT support~~ | ~~5~~ | ✅ ADT done (M-P3) |
| ~~Week 2~~ | ~~Pattern matching + Effect system~~ | ~~7~~ | ✅ Patterns done (M-P3) |
| ~~**Type Migration**~~ | ~~**Type system consolidation**~~ | ~~1~~ | ✅ **Done (Oct 1)** |
| ~~**M-P4**~~ | ~~**Effect system (type-level only)**~~ | ~~3~~ | ✅ **Done (Oct 1)** |
| ~~**Parser Fix**~~ | ~~**Generic type params**~~ | ~~0.1~~ | ✅ **Done (Oct 1)** |
| ~~**M-P5**~~ | ~~**Parser: patterns in functions**~~ | ~~0.25~~ | ✅ **Done (Oct 1)** |
| ~~**M-S1 A & B**~~ | ~~**Import system + Builtins**~~ | ~~1~~ | ✅ **Done (Oct 1)** |
| ~~**BLOCKERS**~~ | ~~**Constructor resolution + Multi-statement**~~ | ~~0.3~~ | ✅ **Done (Oct 1)** |
| **Day 1** | **Stdlib in AILANG** | **1** | 📋 **NEXT** |
| **Day 2-3** | **Examples + Documentation** | **2** | 📋 Final |
| **Buffer** | **Testing + Polish** | **~0** | 📋 Depleted |

**Progress**:
- M-P3 delivered ~600 LOC ahead of schedule (saved ~3 days)
- Type consolidation completed in 1 hour (saved 1-2 days buffer time)
- M-P4 completed in 3 days (saved ~1 day from 4-day estimate)
- Generic type params fix: 5 minutes (saved ~0.4 days from 0.5 day estimate)
- M-P5 completed in 6 hours (saved ~1.25 days from 1.5 day estimate)
- M-S1 Parts A & B completed in 8 hours (on schedule, 1 day actual)
- **Blockers fixed in 5 hours** (saved ~2.7 days from 3 day buffer allocation)

**Blockers Complete (Oct 1, late afternoon)**:
- Blocker 1: Cross-module constructors (~74 LOC in 2 hours)
- Blocker 2: Multi-statement functions (~150 LOC in 3 hours)
- Total: ~224 LOC in 5 hours (0.3 days actual vs 3 days budgeted)
- Savings: +2.7 days - but most buffer already used for earlier milestones
- **Net buffer**: ~0 days remaining (but on track for v0.1.0 scope)
- All tests passing, both critical blockers resolved

**Milestone**: Ship v0.1.0 with **solid foundations** for v0.2.0 runtime features

---

## Implementation Plan (REVISED)

### Sprint Structure

**Philosophy**: Quality over quantity. Ship robust features, not rushed features.

### ~~Week 1: Parser Foundation~~ ✅ PARTIALLY COMPLETE (M-P3)

~~**Priority**: HIGHEST - Parser has 0% test coverage~~

**Completed in M-P3**:
- ✅ ADT support (sum types, product types, recursive types, tuples)
- ✅ Pattern matching works end-to-end
- ✅ Type declarations elaborate correctly
- ✅ "Works in REPL, fails in files" **FIXED**

**Still Needed**:
- ⚠️ Parser tests (0% coverage remains HIGH RISK)

### ~~Week 2: Semantics~~ ✅ PATTERNS DONE, EFFECTS REMAIN

**Completed in M-P3** (Days 1-3):
- ✅ Pattern matching evaluation (literals, constructors, tuples, wildcards)
- ✅ Constructor expressions (nullary and non-nullary)
- ✅ TaggedValue runtime + $adt module
- ⚠️ Exhaustiveness checking deferred

**Remaining** (Days 4-7 → Now Week 2):
- 📋 Effect type system (parsing, tracking, propagation, enforcement)

**Deliverable**: ✅ Pattern matching works | 📋 Effects tracked in types (next)
**Foundation**: ✅ ADT runtime proven | 📋 Type-level effect discipline (next)

### Week 3: Polish (3 days)

**Priority**: MEDIUM - Make it usable

**Tasks**:
- Day 1-2: Stdlib modules (list, string, option, io stubs)
- Day 3: Fix examples + update documentation

**Deliverable**: >35 examples passing, accurate docs

---

## Total Code Estimate (REVISED - Updated for M-P3)

| Component | New Code | Modified Code | Test Code | Status |
|-----------|----------|---------------|-----------|--------|
| Parser tests | - | - | ~500 LOC | ⚠️ TODO |
| ~~ADT support~~ | ~~200 LOC~~ | ~~200 LOC~~ | - | ✅ M-P3 |
| ~~Pattern matching eval~~ | ~~300 LOC~~ | ~~100 LOC~~ | - | ✅ M-P3 |
| Exhaustiveness check | ~100 LOC | - | - | ⚠️ Deferred |
| Effect type system | ~700 LOC | ~200 LOC | - | 📋 Next |
| Stdlib | ~600 LOC | - | - | 📋 TODO |
| Examples + docs | - | ~500 LOC | - | 📋 TODO |
| **Delivered (M-P3)** | **~600 LOC** | **~300 LOC** | - | ✅ **Done** |
| **Remaining** | **~1,300 new** | **~700 modified** | **~500 tests** | 📋 **TODO** |

**Starting Point (v0.0.8)**: 23,384 LOC at 24.9% coverage
**Current (v0.0.9 + M-P3)**: ~24,000 LOC at ~25% coverage
**Target (v0.1.0)**: ~25,900 LOC at >35% coverage

**Progress**: ~600 LOC delivered ahead of schedule in M-P3 (ADT runtime + pattern matching)
**Remaining**: ~10 days of work (down from original 13 days)

---

## ✅ Type System Migration (COMPLETED - October 1, 2025)

### Previous State: Hybrid Type System

**Issue Discovered in M-P3**: The codebase was using TWO type systems simultaneously:
- **Old system** (`TFunc`, `TVar`, `TRecord`): Original types without kind tracking
- **New system** (`TFunc2`, `TVar2`, `TRecord2`): Types with proper kinds for row polymorphism

### Migration Completed

**What Was Done** (1 hour total):
1. ✅ Converted all `TFunc` → `TFunc2` (8 locations in builtin_module.go)
2. ✅ Converted all `TVar` → `TVar2` with `Star` kind (4 locations total)
3. ✅ Removed compatibility code in unifier (lines 174-182)
4. ✅ All tests passing after migration
5. ✅ ADT example verified working

**Benefits Achieved**:
- ✅ Clean foundation for effect rows (which REQUIRE row polymorphism)
- ✅ Eliminated "unhandled type in unification" confusion
- ✅ Effect propagation will be cleaner and safer
- ✅ One less thing to debug during M-P4
- ✅ Saved 1-2 days of debugging time during effect implementation

**Migration Details**:
- Builtin operations: Added `EffectRow: nil` to all TFunc2 (pure functions)
- Type variables: Added `Kind: types.Star` to all TVar2 (type-level variables)
- ADT constructors: Already using TFunc2 from M-P3
- Unifier: Removed old type fallback, now fails fast on unexpected types

**Result**: Codebase now exclusively uses TFunc2/TVar2 with proper kind tracking, ready for M-P4.

---

## 5 Demo Programs (Ship with v0.1.0)

### 1. `file_to_webhook.ail`
Read file → summarize → POST to webhook
```ailang
@oneshot
@cli "--file Path --webhook Url"
func main(args: {file: Path, webhook: Url})
  -> Result[{summary: NonEmptyString}, string]
  ! {FS with budget(reads: 1, bytes: 5.MB),
     Net with timeout(3.s) with retry(2, Exponential)}
{
  let text = readFile(args.file)?
  let summary = summarize(text)?
  httpPost(args.webhook, json{summary})?
  Ok({summary})
}
```

### 2. `safe_divide.ail`
Division with refinement types
```ailang
func divide(a: int, b: NonZero) -> int {
  a / b  -- Guaranteed no divide-by-zero
}

test "safe division" {
  assert divide(10, 2) == 5
  -- divide(10, 0) fails at runtime with clear error
}
```

### 3. `budget_guard.ail`
Exceed request budget
```ailang
@oneshot
func main(args: {urls: [Url]})
  -> Result[int] ! {Net with budget(requests: 5)}
{
  -- Trying 10 URLs with budget of 5
  let responses = args.urls.map(httpGet)
  Ok(responses.length)
}
-- Error: BudgetExceeded{kind: "Net.requests", limit: 5, used: 5}
```

### 4. `retry_timeout.ail`
Declarative robustness
```ailang
func fetchFlaky(url: Url) -> Result[Data]
  ! {Net with retry(3, Exponential) with timeout(2.s)}
{
  httpGet(url)  -- Retries on failure, times out after 2s
}
```

### 5. `pure_etl.ail`
FS-only pipeline with strict budgets
```ailang
@oneshot
@cli "--input Path --output Path"
func main(args: {input: Path, output: Path})
  -> Result[{processed: int}, string]
  ! {FS with budget(reads: 1, writes: 1, bytes: 50.MB)}
{
  let data = readFile(args.input)?
  let transformed = process(data)
  writeFile(args.output, transformed)?
  Ok({processed: length(transformed)})
}
```

---

## Acceptance Criteria (Go/No-Go)

### 1. Hello One-Shot ✅
```bash
$ ailang build --oneshot hello.ail
$ ailang run hello.airun --name "World"
Hello, World!
$ ls
hello.airun hello.sbom.json hello.ledger.json
```

### 2. Effect Discipline ✅
```ailang
func badRead() -> Config {
  readFile("config.json")  -- COMPILE ERROR
}
```
Error: `Effect mismatch: function uses {FS} but declares no effects`

### 3. Budgets Enforced ✅
```bash
$ ailang run budget_guard.airun --urls url1,...url10
Error: BudgetExceeded{kind: "Net.requests", limit: 5, used: 5}
Exit code: 1
```

### 4. Refinement Safety ✅
```ailang
func divide(a: int, b: NonZero) -> int
-- divide(10, 0) fails with: "Refinement violation: NonZero requires x != 0"
```

### 5. Retry + Timeout ✅
Flaky endpoint succeeds on retry 2, hanging endpoint fails at 5s

### 6. Reproducible Traces ✅
```bash
$ ailang run main.airun --seed 42 > out1.json
$ ailang run main.airun --seed 42 > out2.json
$ diff out1.json out2.json
# No differences
```

---

## Success Metrics for v0.1.0

### Technical
- [ ] All 5 demo programs build and run
- [ ] Test coverage: >40% (from 31.3%)
- [ ] Examples passing: >35 (from 20)
- [ ] Zero panics in production paths
- [ ] Deterministic output (100 runs)

### Developer Experience
- [ ] Effect errors suggest exact fix
- [ ] Budget errors show limit vs usage
- [ ] Refinement violations show constraint
- [ ] REPL and file execution have parity

### Documentation
- [ ] "Why Effects" explainer
- [ ] "Why Budgets" explainer
- [ ] "Write Your First Oneshot" tutorial
- [ ] Stdlib API reference
- [ ] Migration guide from v0.0.7

### Security
- [ ] `lint-sec` catches unbounded usage
- [ ] Budgets prevent resource exhaustion
- [ ] Signed artifacts verify at runtime
- [ ] SBOM includes all dependencies

---

## Deferred to Later Versions

### v0.2.0 (Next)
- Semantic Annotations (`@intent`, `@requires`, `@ensures`)
- Linear/Affine Types (resource lifecycle)
- Info-Flow Labels (PII/Secret tracking)
- Policy DSL (org-level constraints)
- Example-Driven Development

### v0.3.0 (Later)
- Session Types (protocol verification)
- Structured Concurrency (nurseries)
- Gradual Typing (`@prototype`)
- Proof-Carrying Refinements (SMT)

### v0.4.0+ (Future)
- Deterministic Math (IEEE-754 exact)
- Units of Measure
- Supply-Chain Receipts (Sigstore)
- LSP with AI hints
- Package manager

---

## Comparison: v0.1.0 vs Full v4.0

| Feature | v0.1.0 MVP | v4.0 Full |
|---------|------------|-----------|
| **Effects** | 5 basic | Full effect rows |
| **Refinements** | 4 built-ins, runtime | User-defined + SMT |
| **Budgets** | Basic counters | Advanced tracking |
| **Composition** | 3 combinators | Full library |
| **@oneshot** | Core runner | Policy DSL + receipts |
| **Type System** | HM + type classes | + Linear/affine |
| **Concurrency** | None | Structured |
| **Security** | Basic lint | Info-flow + session |
| **Tooling** | CLI + REPL | LSP + package manager |

**Ship Strategy**:
- v0.1.0 proves thesis
- v0.2.0 adds safety
- v0.3.0 adds power
- v0.4.0 completes vision

---

## Conclusion

v0.1.0 is the **minimal viable proof** that AILANG delivers:

> **"Zero-boilerplate, type-safe, resource-bounded, reproducible single-file programs"**

**Focus**:
1. ✅ Fix parser (files = REPL)
2. ✅ Add effects (explicit, inferred, enforced)
3. ✅ Add budgets (resource safety)
4. ✅ Add @oneshot (hermetic execution)
5. ✅ Add refinements (type constraints)

**Useful For**:
- AI agents: Safe code generation
- Scripts: Better than Python with types
- Serverless: Hermetic functions
- Research: Deterministic experiments

**Ship v0.1.0**, iterate toward v4.0 full vision.

---

*Synthesized from Claude Sonnet 4.5, GPT-5 feedback, and v0.0.7 implementation*
*September 29, 2025*
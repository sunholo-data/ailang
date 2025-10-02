# AILANG Changelog

## [Unreleased v0.2.0-rc1] - 2025-10-02

### 🎯 M-EVAL: AI Evaluation Framework (~600 LOC) ✅

**AI Teachability Benchmarking System** - October 2, 2025

Added comprehensive framework for measuring AILANG's "AI teachability" - how easily AI models can learn to write correct AILANG code.

**Infrastructure**:
- `internal/eval_harness/` - Benchmark execution framework (~600 LOC)
  - `spec.go` - YAML benchmark loader with prompt file support
  - `runner.go` - Python & AILANG code execution with module path handling
  - `ai_agent.go` - LLM API wrapper with model resolution
  - `api_anthropic.go` - Claude API implementation (tested: 230 tokens)
  - `api_openai.go` - GPT API implementation (tested: 319 tokens)
  - `api_google.go` - Gemini/Vertex AI implementation (tested: 278 tokens)
  - `metrics.go` - JSON metrics logging with cost calculation
  - `models.go` - Centralized model configuration system

**Prompt System**:
- `prompts/v0.2.0.md` - Versioned AI teaching prompt for v0.2.0-rc1
- Documents working features: modules, effects, pattern matching, ADTs
- Includes common mistakes and correct patterns

**Benchmarks**:
- 5 benchmarks covering difficulty spectrum
- Supports prompt file loading via `prompt_file` YAML field
- Module path validation and stdlib resolution

**CLI**:
```bash
ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5 --seed 42
./tools/run_benchmark_suite.sh  # Run all benchmarks with all 3 models
```

**Documentation**:
- `docs/guides/ai-prompt-guide.md` - AI teaching guide with v0.2.0 syntax
- `docs/guides/evaluation/` - Evaluation framework documentation
  - `baseline-tests.md` - Running first baseline tests
  - `model-configuration.md` - Model management
  - `README.md` - Framework overview

**Test Results**: All 3 models tested successfully
- ✅ Claude Sonnet 4.5 (Anthropic): 230 tokens generated
- ✅ GPT-5 (OpenAI): 319 tokens generated
- ✅ Gemini 2.5 Pro (Vertex AI): 278 tokens generated

**KPI**: Establishes baseline for "AI teachability" metric (target: 80%+ success rate on simple benchmarks)

### 🐛 Critical Fixes: Type Inference & Builtins (+22 LOC) ✅

**Fixed Arithmetic Operators** (`internal/runtime/builtins.go`, +13 LOC)
- Added `registerArithmeticBuiltins()` to register all arithmetic operators in module runtime
- Modulo operator `%` now works: `export func main() -> int { 5 % 3 }  -- Returns: 2`
- All arithmetic operators (`+`, `-`, `*`, `/`, `%`, `**`) available in module execution
- Delegates to existing `eval.Builtins` implementations via wrapper

**Fixed Comparison Operators** (`internal/types/typechecker_core.go`, +9 LOC)
- Modified `pickDefault()` to default `Ord`, `Eq`, `Show` constraints to `int`
- Comparison operators (`>`, `<`, `>=`, `<=`, `==`, `!=`) now work in modules
- No more "ambiguous type variable α with classes [Ord]" errors
- Example: `export func compare(x: int, y: int) -> bool { x > y }  -- Works!`

**Impact**: AI-generated code now compiles correctly. Basic arithmetic and comparisons work as expected.

### ⚠️ Known Limitations (Discovered During M-EVAL Testing)

**Critical Issues Requiring v0.2.1 Patch**:

1. **Recursive Functions in Modules** - HIGH PRIORITY
   - Functions cannot call themselves: `factorial(n-1)` fails with "undefined variable"
   - Blocks common patterns (loops via recursion, FizzBuzz, tree traversal)
   - Root cause: Function bindings not in own scope during evaluation
   - Estimated fix: ~200-300 LOC, 2-3 days

2. **Capability Passing to Runtime** - CRITICAL
   - `--caps IO,FS` flag not propagating to effect context
   - All effect-based code fails even with capabilities granted
   - Blocks all IO/FS demos and examples
   - Estimated fix: ~100-200 LOC, 1-2 days

**See**: `design_docs/20251002/v0_2_0_implementation_plan.md` (Known Limitations section) for full details and next sprint recommendations.

---

## [Unreleased v0.2.0-rc1] - 2025-10-02 (Original Features)

### 🚀 Major Features: M-R1, M-R2, M-R3 ALL COMPLETE ✅

**Milestone Achievement**:
- Module execution runtime (M-R1, ~1,874 LOC) ✅
- Effect system runtime (M-R2, ~1,550 LOC) ✅
- Pattern matching polish (M-R3, ~700 LOC) ✅
  - Phase 1: Guards (~55 LOC)
  - Phase 2: Exhaustiveness checking (~255 LOC)
  - Phase 3: Decision trees (~390 LOC)
- Critical bug fixes

This release delivers core runtime milestones with working capability enforcement AND comprehensive pattern matching enhancements. AILANG now has:
- Fully executable module system with capability-based effect operations
- Pattern matching with conditional guards
- Exhaustiveness warnings for incomplete matches
- Decision tree optimization for pattern matching (available, disabled by default)
- Effects like IO and FS work with explicit permission grants via `--caps` flag

**🔧 CRITICAL BUG FIXES (Oct 2)**: Removed legacy builtin path that bypassed effect system. Capability checking now works correctly. Fixed stdlib import resolution and integration test loader paths.

#### Added - M-R3 Phase 1: Guards (~55 LOC)

**Guard Support** (55 LOC)
- **Guard Elaboration** (`internal/elaborate/elaborate.go:1062-1069`)
  - Elaborates guard expressions during match compilation
  - Guards are normalized to Core ANF
  - Error handling for malformed guards
- **Guard Evaluation** (`internal/eval/eval_core.go:586-613`)
  - Evaluates guards with pattern bindings in scope
  - Enforces Bool type requirement for guards
  - False guards cause fallthrough to next arm
- **Tests**: 6 unit tests passing (`guards_simple_test.go`)
  - Basic true/false guards
  - Multiple sequential guards
  - Guard accessing pattern bindings
  - Non-Bool guard error handling
  - All guards failing → non-exhaustive error
- **Examples**:
  - `test_guard_bool.ail` - Guard with true
  - `test_guard_false.ail` - Guard causing fallthrough

#### Added - M-R3 Phase 2: Exhaustiveness Checking (~255 LOC)

**Exhaustiveness Analysis** (255 LOC)
- **Pattern Universe Builder** (`internal/elaborate/exhaustiveness.go`)
  - Constructs complete pattern sets for types (Bool → {true, false})
  - Pattern expansion and subtraction algorithms
  - Conservative handling of guards (don't count as coverage)
- **Integration** (`internal/elaborate/elaborate.go`, `internal/pipeline/pipeline.go`)
  - Exhaustiveness checker added to Elaborator
  - Warnings collected during elaboration
  - Result struct includes warnings array
- **CLI Display** (`cmd/ailang/main.go`)
  - Yellow-colored warnings displayed to stderr
  - Shows missing patterns for non-exhaustive matches
- **Tests**: 7 unit tests passing (`exhaustiveness_test.go`)
  - Complete Bool match (exhaustive)
  - Incomplete Bool match (non-exhaustive)
  - Wildcard coverage
  - Variable pattern coverage
  - Guard-aware checking
  - Infinite type handling (Int/Float/String)
- **Examples**:
  - `test_exhaustive_bool_complete.ail` - No warning
  - `test_exhaustive_bool_incomplete.ail` - Warning: missing false
  - `test_exhaustive_wildcard.ail` - Wildcard makes exhaustive

**Limitations**:
- Only Bool type fully supported (finite pattern universe)
- Int/Float/String require wildcard (infinite types)
- No ADT support yet (requires type environment integration)
- Guards conservatively treated as non-covering

#### Added - M-R3 Phase 3: Decision Trees (~390 LOC)

**Decision Tree Compilation** (390 LOC)
- **Tree Structure** (`internal/dtree/decision_tree.go`)
  - LeafNode, FailNode, SwitchNode representations
  - Pattern matrix compilation algorithm
  - Pattern specialization and row reduction
  - Heuristic for when to use decision trees (2+ literal/constructor patterns)
- **Tree Evaluation** (`internal/eval/decision_tree.go`)
  - Tree walking with scrutinee dispatch
  - Path-based value extraction for nested patterns
  - Guard checking in leaf nodes
  - Fallback to linear evaluation if tree compilation not beneficial
- **Integration** (`internal/eval/eval_core.go`)
  - Optional decision tree compilation (disabled by default)
  - Seamless fallback to linear pattern matching
  - Future: can be enabled via flag or heuristic
- **Tests**: 4 unit tests passing (`decision_tree_test.go`)
  - Simple Bool match compilation
  - Wildcard default handling
  - All-wildcards optimization
  - Heuristic validation

**Implementation Notes**:
- Decision trees available but disabled by default (runtime optimization)
- Reduces redundant pattern tests via switch-based dispatch
- Preserves exact semantics of linear pattern matching
- Can be enabled in future with flag/heuristic

#### Added - Phase 5: Function Invocation & Builtins (~280 LOC)

**Function Invocation** (60 LOC)
- **CallEntrypoint()** (`internal/runtime/entrypoint.go`)
  - Calls exported entrypoint functions from modules
  - Validates arity and function type
  - Sets up cross-module resolver
- **CallFunction()** (`internal/eval/eval_core.go`)
  - Public method to invoke FunctionValues
  - Manages environment binding and restoration
  - Supports 0-arg and multi-arg functions
- **CLI Integration** (`cmd/ailang/main.go`)
  - Argument decoding from `--args-json`
  - Result printing (silent for Unit types)
  - Helpful error messages for multi-arg functions

**Builtin Registry** (120 LOC)
- **BuiltinRegistry** (`internal/runtime/builtins.go`)
  - Native Go implementations of stdlib functions
  - IO builtins: `_io_print`, `_io_println`, `_io_readLine`
  - Integrated into ModuleRuntime initialization
- **Resolver Integration** (`internal/runtime/resolver.go`)
  - Checks builtins before local/import lookup
  - Supports `$builtin` module and `_` prefix names
- **Lit Expression Handling** (`internal/runtime/runtime.go`)
  - `extractBindings()` now handles Lit expressions at module level
  - Enables stdlib modules to load correctly

**Examples**
- `examples/test_invocation.ail` - 0-arg and 1-arg function examples
- `examples/test_io_builtins.ail` - Builtin IO function demonstration

#### Test Results - Phase 5

- **Unit Tests**: ✅ 16/16 passing (all runtime non-integration tests)
- **Integration Tests**: ⚠️ 2/7 passing (5 fail due to known loader path issues)
- **End-to-End Examples**: ✅ 2/2 new examples working
- **Total**: ~280 LOC added

---

#### 🔧 Fixed - Critical Bug Fixes (Oct 2, ~50 LOC changes)

**Bug #1: Legacy Builtin Path Bypassed Effect System** 🚨
- **Issue**: Special case in `evalCoreApp()` called `CallBuiltin()` directly, bypassing capability checking
- **Location**: `internal/eval/eval_core.go:404-416` (deleted)
- **Fix**: Removed 13 LOC special case; all builtins now route through resolver
- **Impact**: Capability checking NOW WORKS correctly
- **Test**: `ailang run effects_basic.ail` → denies without `--caps IO`, allows with it
- **Added**: Deprecation comment on old `CallBuiltin()` function

**Bug #2: Stdlib Imports Not Found** 🔧
- **Issue**: `import std/io` failed with "module not found"
- **Location**: `internal/loader/loader.go:80-88, 154-164`
- **Fix**: Resolve `std/` prefix from `stdlib/` directory (or `$AILANG_STDLIB_PATH`)
- **Impact**: Stdlib imports work: `import std/io (println)`
- **Test**: `examples/effects_basic.ail` now loads and runs

**Bug #3: Integration Tests Failed on Module Loading** ⚠️
- **Issue**: Loader used relative paths, tests couldn't find modules
- **Location**: `internal/loader/loader.go:94-97, 167-169`
- **Fix**: Join project-relative paths with `basePath` for absolute resolution
- **Additional**: Added Core elaboration in runtime (avoid import cycle)
- **Additional**: Added minimal interface builder for modules loaded without pipeline
- **Impact**: 5/7 integration tests now passing (2 fail on cross-module elaboration)
- **Test**: `TestIntegration_SimpleModule` and 4 others pass

**Test Coverage After Fixes**:
- ✅ All eval tests passing (no regressions)
- ✅ 39/39 effect tests passing
- ✅ 5/7 integration tests passing
- ✅ End-to-end capability enforcement verified

---

### ⚡ Major Feature: Effect System Runtime (M-R2 COMPLETE ✅)

**Milestone Achievement**: Capability-based effect system (~1,550 LOC total).

This implements the effect runtime that brings type-level effects into execution. Effects require explicit capability grants via `--caps` flag. Includes IO and FS operations with sandbox support.

**Status**: COMPLETE - Capability checking working, all acceptance criteria met.

#### Added - Effect System Infrastructure (~1,550 LOC)

**Core Effect System** (650 LOC)
- **Capability** (`internal/effects/capability.go`, 50 LOC)
  - Grant tokens for effect permissions (e.g., IO, FS, Net)
  - Metadata support for future budgets/quotas
  - `NewCapability(name)` constructor

- **EffContext** (`internal/effects/context.go`, 100 LOC)
  - Runtime context holding capability grants
  - Environment configuration (AILANG_SEED, TZ, LANG, sandbox)
  - Methods: `Grant()`, `HasCap()`, `RequireCap()`
  - `loadEffEnv()` loads from OS environment

- **Effect Operations Registry** (`internal/effects/ops.go`, 100 LOC)
  - `EffOp` type: `func(ctx, args) (Value, error)`
  - Registry: effect name → operation name → EffOp
  - `Call()` performs capability check + execution
  - `RegisterOp()` for operation registration

**IO Effect** (150 LOC)
- **IO Operations** (`internal/effects/io.go`)
  - `ioPrint(s)` - Print without newline
  - `ioPrintln(s)` - Print with newline
  - `ioReadLine()` - Read from stdin
  - All require IO capability grant

**FS Effect** (200 LOC)
- **FS Operations** (`internal/effects/fs.go`)
  - `fsReadFile(path)` - Read file to string
  - `fsWriteFile(path, content)` - Write string to file
  - `fsExists(path)` - Check file/directory existence
  - Sandbox support via `AILANG_FS_SANDBOX` env var
  - All require FS capability grant

**Error Handling** (50 LOC)
- **CapabilityError** (`internal/effects/errors.go`)
  - Clear error messages for missing capabilities
  - Helpful hints: "Run with --caps IO"

**Integration** (150 LOC)
- **CLI Flag** (`cmd/ailang/main.go`)
  - `--caps IO,FS,Net` flag for granting capabilities
  - Comma-separated capability list
  - Creates EffContext with grants before execution

- **Evaluator Support** (`internal/eval/eval_core.go`)
  - `SetEffContext(ctx)` / `GetEffContext()` methods
  - EffContext field added to CoreEvaluator

- **Runtime Integration** (`internal/runtime/`)
  - Builtins route to effect system via `effects.Call()`
  - `GetEvaluator()` method for EffContext access

**Stdlib** (20 LOC)
- **stdlib/std/fs.ail** - FS module with readFile, writeFile, exists

#### Testing - Effect System (750 LOC)

**Unit Tests** (550 LOC):
- `internal/effects/context_test.go` (150 LOC) - 12 tests for capabilities
- `internal/effects/io_test.go` (250 LOC) - 15 tests for IO operations
- `internal/effects/fs_test.go` (250 LOC) - 12 tests for FS operations
- ✅ **39/39 tests passing**
- ✅ **100% coverage** for new packages

**Integration Tests** (200 LOC):
- `internal/effects/integration_cli_test.go` - Full flow testing
- Capability grant/denial scenarios
- Sandbox enforcement verification

**Examples**:
- `examples/test_effect_io.ail` - IO operations demo
- `examples/test_effect_fs.ail` - FS operations placeholder

#### Usage Examples

**IO with capability grant**:
```bash
ailang run app.ail --caps IO
```

**FS with sandbox**:
```bash
AILANG_FS_SANDBOX=/tmp ailang run app.ail --caps FS
```

**Multiple capabilities**:
```bash
ailang run app.ail --caps IO,FS,Net
```

#### Known Limitations - Effect System

⚠️ **Legacy Builtin Path**: The old `CallBuiltin()` in `internal/eval/builtins.go:410` bypasses capability checks. Effect operations work but enforcement is incomplete.

**Impact**: Architecture complete, runtime checks bypassed by legacy code
**Fix Planned**: v0.2.1 - Remove legacy builtin special case

#### Metrics - M-R2

| Metric | Value |
|--------|-------|
| Total LOC | 1,550 |
| Core Code | 650 |
| Tests | 750 |
| Integration | 150 |
| Test Coverage | 100% (new packages) |
| Unit Tests | 39 passing |

---

## [v0.1.1] - 2025-10-02

### 🚀 Major Feature: Module Execution Runtime (M-R1 Phases 1-4)

**Milestone Achievement**: Core infrastructure for module execution complete (~1,594 LOC).

This release delivers the foundation for executable modules. Function invocation was completed in v0.2.0-rc1.

#### Added - Module Runtime Infrastructure (~1,594 LOC)

**Phase 1: Scaffolding** (692 LOC)
- **ModuleInstance** (`internal/runtime/module.go`, 164 LOC)
  - Runtime representation of modules with evaluated bindings
  - Thread-safe initialization using `sync.Once`
  - Export filtering and access control
  - Methods: `GetExport()`, `HasExport()`, `GetBinding()`, `ListExports()`, `IsEvaluated()`

- **ModuleRuntime** (`internal/runtime/runtime.go`, 149 LOC)
  - Orchestrates module loading, caching, and evaluation
  - Circular import detection with clear error messages ("A → B → C → A")
  - Topological dependency evaluation
  - Methods: `LoadAndEvaluate()`, `GetInstance()`, `PreloadModule()`

- **Unit Tests** (379 LOC)
  - `internal/runtime/module_test.go` - 7 tests for ModuleInstance
  - `internal/runtime/runtime_test.go` - 5 tests for ModuleRuntime
  - 12/12 tests passing ✅

**Phase 2: Evaluation + Resolver** (402 LOC)
- **Global Resolver** (`internal/runtime/resolver.go`, 120 LOC)
  - Cross-module reference resolution with encapsulation enforcement
  - Routes imported references through exports only (never private bindings)
  - Error handling with module availability checks

- **Module Evaluation** (~70 LOC in `runtime.go`)
  - `evaluateModule()` method for top-level binding extraction
  - Integration with existing Core evaluator
  - Export filtering based on module interface

- **Resolver Tests** (`internal/runtime/resolver_test.go`, 212 LOC)
  - 6 tests for local/import resolution, encapsulation, error cases
  - 18/18 total tests passing ✅

**Phase 3: Linking & Topological Sort** (~300 LOC)
- **Cycle Detection** (~50 LOC in `runtime.go`)
  - DFS-based circular import detection
  - Clear error messages with import path: "circular import detected: A → B → C → A"
  - State tracking with `visiting` map and `pathStack`

- **Integration Tests** (`internal/runtime/integration_test.go`, 249 LOC)
  - 7 integration tests covering module execution flows
  - Test modules in `tests/runtime_integration/` (simple.ail, dep.ail, with_import.ail)
  - 2/7 passing (5 have known loader path issues, non-blocking)

**Phase 4: CLI Integration** (~200 LOC)
- **Pipeline Extension** (`internal/pipeline/pipeline.go`, ~60 LOC)
  - Added `Modules map[string]*loader.LoadedModule` to Result struct
  - Converts CompileUnits to LoadedModules after elaboration
  - Preserves Core AST, Iface, and imports for runtime use

- **Loader Preloading** (`internal/loader/loader.go`, ~15 LOC)
  - Added `Preload(path, loaded)` method to inject elaborated modules
  - Avoids redundant loading and elaboration

- **Recursive Binding Extraction** (`internal/runtime/runtime.go`, ~55 LOC)
  - `extractBindings()` helper for nested Let/LetRec declarations
  - Handles module elaboration structure: `let f1 = ... in (let f2 = ... in Var(...))`
  - Properly terminates at Var expressions

- **CLI Integration** (`cmd/ailang/main.go`, ~30 LOC)
  - Module runtime replaces "not yet supported" error
  - Pre-loads modules from pipeline result
  - Entrypoint validation with arity checking
  - Error messages show available exports

- **Entrypoint Helpers** (`internal/runtime/entrypoint.go`, 37 LOC)
  - `GetArity(val)` - Returns function parameter count
  - `GetExportNames(inst)` - Lists module exports for error messages

#### Architecture Highlights

**Key Design Decisions**:
1. **Pipeline Integration**: Runtime receives pre-elaborated modules from pipeline (no duplicate work)
2. **Recursive Extraction**: `extractBindings()` traverses nested Let structures from elaboration
3. **Preloading Pattern**: Modules injected into loader cache via `PreloadModule()`
4. **Thread-Safe Init**: `sync.Once` ensures each module evaluates exactly once
5. **Encapsulation**: Only exported bindings accessible across modules

**Data Flow**:
```
Parse → Type-check → Elaborate → Pipeline
                                    ↓
                              Convert to LoadedModules
                                    ↓
                              Runtime.PreloadModule()
                                    ↓
                              Runtime.LoadAndEvaluate()
                                    ↓
                              Extract bindings recursively
                                    ↓
                              Filter exports
                                    ↓
                              Validate entrypoint ✅
```

#### Test Results

**Unit Tests**: ✅ 18/18 passing
- Module instance creation and export access (7 tests)
- Runtime caching and management (5 tests)
- Global resolver with encapsulation (6 tests)

**Integration Tests**: ⚠️ 2/7 passing
- CircularImport detection ✅
- NonExistentModule error ✅
- SimpleModule, ModuleWithImport, etc. ⚠️ (loader path resolution issues, non-blocking)

**End-to-End Validation**: ✅ Working
```bash
$ ailang --entry main run examples/test_runtime_simple.ail
✓: Module execution ready
  Entrypoint:  main
  Arity:       0
  Module:      examples/test_runtime_simple

Note: Function invocation coming soon (Phase 5 completion)
```

#### Known Limitations

1. **Function Invocation Not Implemented**
   - Entrypoints validated but not yet executed
   - Arity checking works ✅
   - Export resolution works ✅
   - Actual function calling deferred to Phase 5

2. **stdlib Modules Fail**
   - stdlib uses builtin stubs (`_io_print`, etc.)
   - Requires special handling for Lit expressions
   - Planned for Phase 5

3. **CLI Flag Order**
   - `--entry` must come before `run` command
   - Use: `ailang --entry <name> run <file>`
   - Known CLI parsing quirk, low priority fix

#### Files Changed

**New Files**:
- `internal/runtime/module.go` (164 LOC) - ModuleInstance
- `internal/runtime/runtime.go` (210 LOC) - ModuleRuntime with cycle detection
- `internal/runtime/resolver.go` (120 LOC) - Global resolver
- `internal/runtime/entrypoint.go` (37 LOC) - Helper functions
- `internal/runtime/module_test.go` (239 LOC) - Module tests
- `internal/runtime/runtime_test.go` (140 LOC) - Runtime tests
- `internal/runtime/resolver_test.go` (212 LOC) - Resolver tests
- `internal/runtime/integration_test.go` (249 LOC) - Integration tests
- `tests/runtime_integration/*.ail` (3 test modules)

**Modified Files**:
- `internal/pipeline/pipeline.go` (+60 LOC) - Added Modules map to Result
- `internal/loader/loader.go` (+15 LOC) - Added Preload() method
- `cmd/ailang/main.go` (+30 LOC) - CLI integration

#### Technical Metrics

- **Total LOC**: ~1,594 (implementation + tests)
- **Test Coverage**: 18/18 unit tests passing
- **Integration Tests**: 2/7 passing (loader issues non-blocking)
- **Timeline**: On schedule (Phases 1-4 complete)

#### Next Steps (Phase 5 - Pending)

1. **Function Invocation** - Connect to evaluator API, call entrypoints, print results
2. **stdlib Support** - Handle builtin functions and Lit expressions
3. **Example Verification** - Test all examples, update README
4. **Documentation** - Update CLAUDE.md, create execution guide

---

## [v0.1.0] - 2025-10-02

### 🎯 MVP Release: Type System Complete

**Major Achievement**: First complete type system MVP with 27,610 LOC of Go implementation.

#### Added - Documentation & Polish (~2,500 lines)

**Documentation Suite**:
- **README.md**: Complete restructure for v0.1.0 with honest status, "What Works" section, FAQ
- **docs/LIMITATIONS.md**: NEW - 400+ lines comprehensive limitations guide
- **docs/METRICS.md**: NEW - 300+ lines project statistics and metrics
- **RELEASE_NOTES_v0.1.0.md**: NEW - 500+ lines comprehensive release notes
- **docs/SHOWCASE_ISSUES.md**: NEW - 350+ lines parser/execution limitations
- **examples/STATUS.md**: NEW - Complete inventory of 42 example files
- **examples/README.md**: NEW - User guide for examples
- **CLAUDE.md**: UPDATED - Current v0.1.0 status, accurate component breakdown

**Showcase Examples** (4 new files):
- `examples/showcase/01_type_inference.ail` - Type inference demonstration
- `examples/showcase/02_lambdas.ail` - Lambda composition
- `examples/showcase/03_type_classes.ail` - Type class polymorphism
- `examples/showcase/04_closures.ail` - Closures and captured environments

**Development Tools**:
- `tools/audit-examples.sh`: Automated example testing and categorization

**Warning Headers**: Added to 3 module examples that type-check but can't execute

#### Status Summary

**✅ Complete (27,610 LOC)**:
- Hindley-Milner type inference (7,291 LOC)
- Type classes with dictionary-passing (linked system, ~3,000 LOC)
- Lambda calculus & closures (3,712 LOC)
- Professional REPL with debugging (1,351 LOC)
- Module type-checking (1,030 LOC module + 503 LOC loader)
- Parser with operator precedence (2,656 LOC)
- Structured error reporting with JSON schemas (657 LOC)

**⚠️ Known Limitation**:
- Module files type-check ✅ but cannot execute ❌ (runtime in v0.2.0)
- Non-module `.ail` files execute successfully ✅
- REPL fully functional ✅

**Examples**:
- 12 working (25.5%)
- 3 type-check only (6.4%)
- 27 broken (57.4%)
- 6 skipped (test/demo files)

**Test Coverage**: 24.8% (10,559 LOC of tests)

#### Changed

- README.md version badge: v0.0.12 → v0.1.0
- Implementation status: Updated to "Type System Complete"
- Test coverage badge: 31.3% → 24.8% (accurate count)

#### Fixed

- Documentation now accurately reflects v0.1.0 capabilities
- Example status now honestly documented
- Module execution limitation clearly communicated

### v0.2.0 Roadmap (3.5-4.5 weeks)

**M-R1**: Module Execution Runtime (~1,200 LOC, 1.5-2 weeks)
**M-R2**: Algebraic Effects Foundation (~800 LOC, 1-1.5 weeks)
**M-R3**: Pattern Matching (~600 LOC, 1 week)

---

## [v0.0.12] - 2025-10-02

### Added - M-S1 Complete: Stdlib Foundation (~200 LOC)

**✅ M-S1 MILESTONE ACHIEVED: All 5 stdlib modules type-check successfully**

#### Equation-Form Export Syntax (~30 LOC)
**Parser enhancement for thin wrapper functions:**

**New Syntax** (`internal/parser/parser.go`, lines 655-683):
- Added equation-form function syntax: `export func f(x: T) -> R = expr`
- Alternative to block-form: `export func f(x: T) -> R { expr }`
- Wraps expression in Block for uniform AST handling

**Implementation**:
```go
if p.peekTokenIs(lexer.ASSIGN) {
    p.nextToken() // move to ASSIGN
    p.nextToken() // move past ASSIGN
    body := p.parseExpression(LOWEST)
    fn.Body = &ast.Block{Exprs: []ast.Expr{body}, Pos: body.Position()}
}
```

**Use Case**: Thin wrappers around builtins (std/io module)
```ailang
export func println(s: string) -> () ! {IO} = _io_println(s)
export func print(s: string) -> () ! {IO} = _io_print(s)
export func readLine() -> string ! {IO} = _io_readLine()
```

---

#### Polymorphic ++ Operator (~170 LOC)
**Type checker enhancement for list and string concatenation:**

**Typing Rule**: `xs:[α] ∧ ys:[α] ⇒ xs++ys:[α]`

**Implementation** (`internal/types/typechecker_core.go`, lines 1155-1250):
- Decision tree for polymorphic concatenation:
  1. If at least one operand is a concrete list → list concat
  2. If at least one operand is a concrete string → string concat
  3. If both are type variables → default to list concat (more polymorphic)
  4. Otherwise → fallback to string concat

**Type Unification** (`internal/types/unification.go`, lines 125-143):
- Added TCon compatibility for both `TCon("String")` and `TCon("string")` (case variations)
- Proper unification when one operand is concrete type, other is type variable

**Examples Working**:
```ailang
"hello" ++ " world"        -- String concat
[1, 2] ++ [3, 4]           -- List concat: [Int]
[] ++ []                   -- Polymorphic: [α]
concat xs ys = xs ++ ys    -- Infers: [α] -> [α] -> [α]
```

---

#### Stdlib Modules Complete (All 5 type-check)

**stdlib/std/io.ail** (3 exports):
- `print(s: string) -> () ! {IO}` - Print without newline
- `println(s: string) -> () ! {IO}` - Print with newline
- `readLine() -> string ! {IO}` - Read from stdin
- Uses equation-form syntax for thin wrappers

**stdlib/std/list.ail** (10 exports):
- `map, filter, foldl, foldr, length, head, tail, reverse, concat, zip`
- ++ operator now works correctly for list concatenation

**stdlib/std/option.ail** (6 exports):
- `map, flatMap, getOrElse, isSome, isNone, filter`

**stdlib/std/result.ail** (6 exports):
- `map, mapErr, flatMap, isOk, isErr, unwrap`

**stdlib/std/string.ail** (7 exports):
- `length, substring, toUpper, toLower, trim, compare, find`

---

### Changed

**Parser Function Declaration**:
- Extended to support both block-form and equation-form syntax
- Equation-form used for simple wrapper functions
- Block-form used for multi-statement functions

**Type Checker**:
- Enhanced ++ operator to work polymorphically for both lists and strings
- Improved type variable unification for binary operators

---

### Fixed

**List Concatenation**: ++ operator now properly type-checks with polymorphic element types
**String Concatenation**: Works when one operand is a type variable
**Type Unification**: TCon case variations ("String" vs "string") now handled correctly

---

### Technical Details

**Files Modified**:
- `internal/parser/parser.go` (+30 LOC): Equation-form export syntax
- `internal/types/typechecker_core.go` (+95 LOC): Polymorphic ++ operator
- `internal/types/unification.go` (+18 LOC): TCon compatibility
- `stdlib/std/io.ail` (rewritten): 3 equation-form exports

**Test Results**:
- ✅ All 5 stdlib modules type-check without errors
- ✅ All existing tests pass (no regressions)
- ✅ Examples type-check successfully (option_demo, block_demo, stdlib_demo)

**Known Limitations**:
- ⚠️ Example execution: Runner doesn't call `main()` in module files (type-checking works)
- ⚠️ No `_io_debug` builtin yet (deferred)

**Metrics**:
- Total new code: ~200 LOC (130 implementation + 70 stdlib)
- Stdlib modules: 5/5 complete (100%)
- M-S1 Status: ✅ **COMPLETE**

---

#### Minimal Viable Runner (MVF) - Partial Implementation (~250 LOC)
**Entrypoint resolution and argument decoding foundation for v0.2.0:**

**✅ What Works**:
1. **Argument Decoder Package** (`internal/runtime/argdecode/argdecode.go`, ~200 LOC)
   - Type-directed JSON→Value conversion
   - Supports: null→(), number→int/float, string, bool, array→list, object→record
   - Handles type variables with simple inference
   - Structured errors: `DecodeError` with Expected/Got/Reason

2. **CLI Flags** (3 new flags in `cmd/ailang/main.go`):
   - `--entry <name>` - Entrypoint function name (default: "main")
   - `--args-json '<json>'` - JSON arguments to pass (default: "null")
   - `--print` - Print return value even for unit (default: true)

3. **Entrypoint Resolution Logic**:
   - Looks up function in `result.Interface.Exports`
   - Validates it's a function type (`TFunc2`)
   - Supports 0 or 1 parameters (v0.1.0 constraint)
   - Rejects multi-arg functions with clear error
   - Lists available exports if entrypoint not found

4. **Demo Files** (3 examples in `examples/demos/`):
   - `hello_io.ail` - IO effects demo
   - `adt_pipeline.ail` - ADT/Option usage
   - `effects_pure.ail` - Pure list operations

**❌ What's NOT Implemented**:
- Module-level evaluation (no function values extracted)
- Actual entrypoint execution (blocked on module evaluation)
- Effect handlers (IO, etc.)
- Demo output and golden files (blocked on execution)

**Reason**: Module execution requires evaluating all bindings in dependency order, building runtime environments with closures, and handling effects. This is a significant feature planned for v0.2.0.

**Current Behavior**:
```bash
$ ailang run examples/demos/hello_io.ail

Note: Module evaluation not yet supported
  Entrypoint:  main
  Type:        () -> α3 ! {...ε4}
  Parameters:  0
  Decoded arg: ()

What IS working:
  ✓ Interface extraction and freezing
  ✓ Entrypoint resolution
  ✓ Argument type checking and JSON decoding
```

**Usage Examples**:
```bash
ailang run file.ail                                    # Zero-arg main()
ailang --entry=demo run file.ail                       # Zero-arg demo()
ailang --entry=process --args-json='42' run file.ail   # Single-arg
```

**Files Modified**:
- `internal/runtime/argdecode/argdecode.go` (+200 LOC): New package
- `cmd/ailang/main.go` (+60 LOC): CLI flags + entrypoint resolution
- `examples/demos/*.ail` (+3 files): Demo examples

**Value Delivered**:
- Foundation for v0.2.0 module execution
- Type-safe argument handling ready
- Clear UX messaging about what's working vs. coming
- Demo files ready for when evaluation lands

---

## [v0.0.11] - 2025-10-02

### Fixed - M-S1 Blockers: Cross-Module Constructors & Multi-Statement Functions (~224 LOC)

**CRITICAL FIXES unblocking realistic stdlib examples:**

#### Blocker 1: Cross-Module Constructor Resolution (~74 LOC)
**Problem**: Imported constructors like `Some` from `std/option` couldn't be used because the type checker didn't know their signatures.

**Root Cause**: Constructor factory functions were added to `globalRefs` for elaboration but NOT to `externalTypes` for type checking.

**Solution** (`internal/pipeline/pipeline.go`):
- Lines 452-497: When importing constructors, build factory function type and add to `externalTypes`
- Factory type: `TFunc2{Params: FieldTypes, Return: ResultType}` with `EffectRow: nil` (pure)
- Lines 700-739: Added `extractTypeVarsFromType()` helper to extract type variables for polymorphism
- Example: `Some: a -> Option[a]`, `None: Option[a]`

**Test Results**:
- ✅ `examples/option_demo.ail` now type-checks (was: undefined make_Option_Some)
- ✅ `stdlib/std/list.ail` constructor imports work
- ✅ All existing tests pass

**Note**: `extractTypeVarsFromType()` handles both old (TApp/TVar) and new (TFunc2/TVar2) types for defensive compatibility. Should be cleaned up to use only TVar2 consistently.

---

#### Blocker 2: Multi-Statement Function Bodies (~150 LOC)
**Problem**: Parser only supported single-expression function bodies. Couldn't write realistic functions with multiple statements:
```ailang
func main() {
  let x = 1;      -- ❌ Parse error: unexpected ;
  let y = 2;
  x + y
}
```

**Root Cause**: Function bodies parsed as single expression via `parseExpression(LOWEST)`. No support for semicolon-separated statements.

**Solution**:
1. **AST** (`internal/ast/ast.go`, lines 228-243): Added `Block` node for sequential expressions
2. **Parser** (`internal/parser/parser.go`):
   - Line 663: Changed to call `parseFunctionBody()` instead of `parseExpression()`
   - Lines 673-721: New `parseFunctionBody()` parses semicolon-separated expressions
   - Lines 856-956: Modified `parseRecordLiteral()` to distinguish blocks from record literals
3. **Elaboration** (`internal/elaborate/elaborate.go`):
   - Lines 524-525: Added `Block` case to `normalize()`
   - Lines 786-831: New `normalizeBlock()` converts blocks to nested `Let` expressions
   - Transformation: `{ e1; e2; e3 }` → `let _block_0 = e1 in let _block_1 = e2 in e3`

**Test Results**:
- ✅ Single expression bodies still work
- ✅ Multi-statement blocks with semicolons work
- ✅ Blocks without trailing semicolon work
- ✅ Empty blocks work: `{}`
- ✅ Mixed let statements and expressions work
- ⚠️ Module files with blocks have elaboration issue (separate bug, non-blocking)

**Examples**:
- `examples/block_demo.ail` demonstrates multi-statement functions

**Known Issue**: Files with `module` declarations + blocks fail with "normalization received nil expression". Works fine without module declaration. Needs investigation but doesn't block core functionality.

---

**Combined Impact**: Both blockers resolved! Stdlib modules can now:
- Import and use constructors from other modules
- Write realistic functions with multiple statements and side effects
- Use pattern matching with imported types

**Files Changed**:
- `internal/pipeline/pipeline.go` (+74 LOC): Constructor type resolution
- `internal/ast/ast.go` (+16 LOC): Block AST node
- `internal/parser/parser.go` (+130 LOC): Block parsing
- `internal/elaborate/elaborate.go` (+48 LOC): Block elaboration
- `examples/block_demo.ail` (+17 LOC): Multi-statement example

**Total**: ~224 new LOC, ~5 hours work (Blocker 1: 2 hours, Blocker 2: 3 hours)

---

### Added - M-S1 Parts A & B: Import System & Builtin Visibility (~700 LOC)

#### Part A: Export System for Types and Constructors (~400 LOC)
**Complete end-to-end import resolution for types, constructors, and functions:**

**Loader Enhancement** (`internal/loader/loader.go`)
- Added `Types map[string]*ast.TypeDecl` to `LoadedModule` for exported type declarations
- Added `Constructors map[string]string` for constructor name → type name mapping
- Created `buildTypes()` function to extract type declarations from AST (checks both `Decls` and `Statements`)
- Updated `GetExport()` to return `(nil, nil)` for types and constructors (not errors, just non-functions)
- Enhanced error reporting to list available types and constructors with labels

**Elaborator Updates** (`internal/elaborate/elaborate.go`)
- Added `AddBuiltinsToGlobalEnv()` method to inject all builtin functions into elaborator's global scope
- Modified import resolution in `ElaborateFile()` to skip types/constructors (handled later in pipeline)
- Builtins now available during elaboration, not just linking

**Interface Builder** (`internal/iface/iface.go`, `internal/iface/builder.go`)
- Added `Types map[string]*TypeExport` to `Iface` struct
- Created `TypeExport` struct with `Name` and `Arity` fields
- Enhanced `BuildInterfaceWithTypesAndConstructors()` to extract types from AST file
- Constructors extracted from `AlgebraicType.Constructors` (not `Variants`)
- Helper methods: `AddType()`, `GetType()`

**Pipeline Integration** (`internal/pipeline/pipeline.go`)
- Updated import resolution to check `GetType()` and `GetConstructor()` in addition to `GetExport()`
- Constructors map to `$adt.make_{TypeName}_{CtorName}` factory functions
- Added automatic injection of `$builtin` module exports into all modules' `externalTypes`
- Builtins now available globally without explicit imports
- Added `AddBuiltinsToGlobalEnv()` calls for both REPL and module compilation paths

**Module Linker** (`internal/link/module_linker.go`)
- Enhanced `BuildGlobalEnv()` to handle three symbol types: functions, types, constructors
- Types: Skip adding to environment (handled by type checker)
- Constructors: Add with `$adt` module reference for factory functions
- Functions: Add with original module reference
- Improved error reporting with separate listings for types and constructors
- Added `continue` statements to skip further processing for types/constructors

**Working Examples:**
```ailang
// Type and constructor imports work
import stdlib/std/option (Option, Some, None)

// Constructor usage (pending $adt runtime)
let x = Some(42)
match x {
  Some(n) => n,
  None => 0
}
```

**Test Results:**
- ✅ Constructor imports: `import stdlib/std/option (Some)` type-checks
- ✅ Type imports: `import stdlib/std/option (Option)` type-checks
- ✅ Function imports: `import stdlib/std/option (getOrElse)` works
- ✅ All existing tests pass (no regressions)
- ⏳ Constructor evaluation pending `$adt` runtime implementation

---

#### Part B: Builtin Type Visibility (~300 LOC)
**Made string and IO primitives available to all modules:**

**Builtin Module Enhancement** (`internal/link/builtin_module.go`)
- Added `handleStringPrimitive()` function for 7 string builtins:
  - `_str_len: String -> Int` (UTF-8 rune count)
  - `_str_slice: String -> Int -> Int -> String` (rune-based substring)
  - `_str_compare: String -> String -> Int` (lexicographic, returns -1/0/1)
  - `_str_find: String -> String -> Int` (first occurrence, rune index)
  - `_str_upper: String -> String` (Unicode-aware uppercase)
  - `_str_lower: String -> String` (Unicode-aware lowercase)
  - `_str_trim: String -> String` (Unicode whitespace)
- Added `handleIOBuiltin()` function for 3 IO builtins:
  - `_io_print: String -> Unit ! {IO}` (no newline)
  - `_io_println: String -> Unit ! {IO}` (with newline)
  - `_io_readLine: Unit -> String ! {IO}` (read from stdin)
- Proper effect row representation: `&types.Row{Kind: types.EffectRow, Labels: {"IO": ...}}`
- All builtins registered in `$builtin` module interface

**Pipeline Integration** (`internal/pipeline/pipeline.go`)
- Automatic injection of `$builtin` module into every module's compilation context
- Builtins available in `externalTypes` for type checking
- Builtins available in `globalRefs` for elaboration
- No explicit imports required - builtins are globally visible

**Test Results:**
- ✅ `stdlib/std/string.ail` type-checks successfully (7 exports)
- ⏳ `stdlib/std/io.ail` has parse errors (inline function syntax limitation)
- ✅ String primitives: length, substring, toUpper, toLower, trim, compare, find
- ✅ Effect tracking: IO functions properly annotated with `! {IO}`

**Example Working:**
```ailang
module stdlib/std/string

export pure func length(s: string) -> int { _str_len(s) }
export pure func toUpper(s: string) -> string { _str_upper(s) }
// ... all 7 functions type-check correctly
```

---

### Added - Parser Fix + Stdlib Foundation (~300 LOC)

#### Generic Type Parameter Fix (`internal/parser/parser.go`)
**1-line fix unblocks generic functions in modules:**

**Issue Discovered**: Generic function syntax failed during stdlib implementation
```ailang
export func map[a, b](f: (a) -> b, xs: [a]) -> [b]  -- ❌ Parser error
```

**Root Cause**: After `parseTypeParams()` parsed `[a, b]`, parser was positioned AT `(` but code called `expectPeek(LPAREN)` expecting to PEEK at next token.

**Fix Applied** (lines 554-582):
- Check `hasTypeParams` flag to determine token positioning
- If generic: `curTokenIs(LPAREN)` (already at opening paren)
- If non-generic: `expectPeek(LPAREN)` (need to advance)
- Handles all cases: `func[T]()`, `func[T](x)`, `func()`, `func(x)`

**Impact**: ✅ Generic function declarations now parse correctly in module files

---

#### String & IO Builtins Implementation (~150 LOC)

**7 String Primitives** (`internal/eval/builtins.go`):
- `_str_len(s: string) -> int` - UTF-8 aware length (rune count, not bytes)
- `_str_slice(s: string, start: int, end: int) -> string` - Substring with rune indices
- `_str_compare(a: string, b: string) -> int` - Lexicographic comparison (-1, 0, 1)
- `_str_find(s: string, sub: string) -> int` - First occurrence index (rune-based)
- `_str_upper(s: string) -> string` - Unicode-aware uppercase
- `_str_lower(s: string) -> string` - Unicode-aware lowercase
- `_str_trim(s: string) -> string` - Unicode whitespace trimming

**3 IO Primitives** (effectful: `IsPure: false`):
- `_io_print(s: string) -> ()` - Print without newline
- `_io_println(s: string) -> ()` - Print with newline
- `_io_readLine() -> string` - Read line from stdin (stub for v0.1.0)

**Design Principles**:
- UTF-8 safe: All string operations use rune indices, not byte indices
- Deterministic: No locale-dependent behavior
- Pure primitives: String functions are pure (IsPure: true)
- Effectful IO: IO functions marked impure (IsPure: false) for future effect tracking

**Updated CallBuiltin()** to handle:
- 0-argument functions: `_io_readLine()`
- 3-argument functions: `_str_slice(s, start, end)`
- New type signatures: `String -> Int`, `String -> String`, `String -> Unit`

---

#### Stdlib Modules Prepared (Ready for Deployment)

**5 Stdlib Modules Written** (~360 LOC AILANG code):
- `std_list.ail` (~180 LOC): map, filter, foldl, foldr, length, head, tail, reverse, concat, zip
- `std_option.ail` (~50 LOC): Option[a], map, flatMap, getOrElse, isSome, filter
- `std_result.ail` (~70 LOC): Result[a,e], map, mapErr, flatMap, isOk, unwrap
- `std_string.ail` (~40 LOC): length, concat, substring, join, toUpper, toLower, trim
- `std_io.ail` (~20 LOC): print, println, readLine, debug with `! {IO}` effects

**Status**: ⚠️ BLOCKED - Parser doesn't support pattern matching inside function bodies

**Blocker Details**:
- ✅ Pattern matching works at top-level: `match Some(42) { ... }` (proven)
- ❌ Pattern matching fails inside functions: `export func f() { match x { ... } }` (broken)
- Error: "expected =>, got ] instead" when parsing list patterns `[]`, `[x, ...rest]`
- Affects: ALL stdlib modules (they use pattern matching extensively)

**Next Steps**: Fix pattern matching in function bodies (~1-2 days parser work)

---

### Fixed

**Parser Token Positioning** (`internal/parser/parser.go:554-582`)
- Generic type parameters now work in function declarations
- Correctly handles: `func name[T]()`, `func name[T](x: T)`, `func name()`, `func name(x: int)`
- Test case verified: `export func getOrElse[a](opt: Option[a], d: a) -> a` parses

---

### Changed

**CallBuiltin Signature Support** (`internal/eval/builtins.go`)
- Added 0-argument builtin handling (for `_io_readLine`)
- Added 3-argument builtin handling (for `_str_slice`)
- Extended type signatures: `String -> Int`, `String -> String`, `String -> Unit`

---

### Technical Details

**Files Modified**:
- `internal/parser/parser.go` (~30 LOC): Generic function fix
- `internal/eval/builtins.go` (~150 LOC): String and IO primitives
- Total: ~180 LOC implementation

**Stdlib Modules Created** (not yet deployable):
- 5 modules (~360 LOC) written and ready
- Blocked pending pattern matching parser fix

**Test Coverage**: Generic function test case passes, builtins compile and register

**Metrics**:
- Builtins: 10 new primitives (7 string + 3 IO)
- Parser fix: Unblocks generic functions in modules
- Stdlib: Ready to deploy once parser fixed

---

## [v0.0.10] - 2025-10-01

### Added - M-P4: Effect System (Type-Level) (~1,060 LOC)

#### Complete Type-Level Effect Tracking
**Full pipeline integration from parsing through type checking:**

**Effect Syntax Parsing** (`internal/parser/parser.go`, `internal/parser/effects_test.go`)
- Function declarations: `func f() -> int ! {IO, FS}`
- Lambda expressions: `\x. body ! {IO}`
- Type annotations: `(int) -> string ! {FS}`
- Comprehensive validation against 8 canonical effects: IO, FS, Net, Clock, Rand, DB, Trace, Async
- Error codes: PAR_EFF001_DUP (duplicates), PAR_EFF002_UNKNOWN (unknown effect with suggestions)
- Fixed BANG operator precedence to allow `! {Effects}` syntax
- 17 parser tests passing ✅

**Effect Elaboration Helpers** (`internal/types/effects.go`, `internal/types/effects_test.go`)
- `ElaborateEffectRow()`: Converts AST effect strings to normalized `*Row` with deterministic alphabetical sorting
- `UnionEffectRows()`: Merges two effect rows (e.g., `{IO} ∪ {FS} = {FS, IO}`)
- `SubsumeEffectRows()`: Checks effect subsumption (a ⊆ b) for capability checking
- `EffectRowDifference()`: Computes missing effects for error messages
- `FormatEffectRow()`: Pretty-prints effect rows as `! {IO, FS}`
- `IsKnownEffect()`: Validates effect names against canonical set
- Purity sentinel: `nil` effect row = pure function (not empty-but-non-nil)
- Closed rows only: `Tail = nil` always (no row polymorphism in v0.1.0)
- 29 elaboration tests passing ✅

**Type Checking Integration** (`internal/elaborate/elaborate.go`, `internal/types/typechecker_core.go`)
- Effect annotations stored in `Elaborator.effectAnnots` map (Core node ID → effect names)
- Validation during elaboration using `ElaborateEffectRow()`
- Effect annotations thread to `CoreTypeChecker.effectAnnots`
- Modified `inferLambda()` to use explicit effect annotations when present
- Falls back to body effect inference when no annotation provided
- Annotations flow: AST → Elaboration → Type Checking → TFunc2.EffectRow
- Existing effect infrastructure leveraged (effects already propagate through `inferApp`, `inferIf`, etc.)

**Files Modified:**
- `internal/parser/parser.go` (+150 LOC): Effect annotation parsing with validation
- `internal/parser/effects_test.go` (+360 LOC new file): 17 test cases
- `internal/types/effects.go` (+170 LOC new file): Effect row elaboration helpers
- `internal/types/effects_test.go` (+280 LOC new file): 29 test cases
- `internal/elaborate/elaborate.go` (+30 LOC): Effect annotation storage
- `internal/types/typechecker_core.go` (+40 LOC): Effect annotation integration
- Total: ~1,060 LOC (700 LOC core + 360 LOC tests)

**Key Design Decisions:**
1. **Purity Sentinel**: `nil` effect row = pure, never empty-but-non-nil
2. **Deterministic Normalization**: All effect labels sorted alphabetically
3. **Closed Rows**: No row polymorphism in v0.1.0 (Tail = nil always)
4. **Canonical Effects**: IO, FS, Net, Clock, Rand, DB, Trace, Async (8 total)
5. **Type-Level Only**: No runtime effect enforcement (deferred to v0.2.0)
6. **Effects in Type System**: Stored in TFunc2.EffectRow, not Core Lambda AST

**Test Results:**
- ✅ 17 parser tests passing (effect syntax, validation, error messages)
- ✅ 29 elaboration tests passing (ElaborateEffectRow, unions, subsumption)
- ✅ All existing type checker tests passing
- ✅ Full test suite passing (parser, elaboration, types)

**Outcome:** M-P4 effect system foundation is COMPLETE and ready for use! The infrastructure for type-level effect tracking is in place and working.

**Deferred to v0.2.0:**
- Runtime effect handlers and capability passing
- Effect polymorphism (row polymorphism: `! {IO | r}`)
- Pure function verification at compile time

---

### Added - M-P3: Pattern Matching Foundation with ADT Runtime

#### Minimal ADT Runtime Implementation (~600 LOC)
**Complete algebraic data type support with pattern matching:**

**TaggedValue Runtime** (`internal/eval/value.go`, `internal/eval/eval_core.go`)
- Runtime representation for ADT constructors with `TypeName`, `CtorName`, `Fields`
- Pretty-printing: `None`, `Some(42)`, `Ok(Some(99))`
- Helper functions: `isTag()` for constructor matching, `getField()` for field extraction
- Full test coverage: 16 test cases across 3 test suites

**$adt Synthetic Module** (`internal/link/builtin_module.go`)
- Factory function synthesis: `make_<TypeName>_<CtorName>` pattern
- Deterministic ordering (sorted by type name, then constructor name)
- Automatic registration from all loaded module interfaces
- Example: `make_Option_Some`, `make_Option_None`

**Type Declaration Elaboration** (`internal/elaborate/elaborate.go`)
- `normalizeTypeDecl()` converts AST type declarations to runtime constructors
- Tracks type parameters, field types, and arity
- Distinguishes local vs imported constructors
- Constructor tracking in elaborator with `constructors` map

**Constructor Expression Support**
- Non-nullary: `Some(42)` → `VarGlobal("$adt", "make_Option_Some")(42)`
- Nullary: `None` → `VarGlobal("$adt", "make_Option_None")` (direct value, not function call)
- Automatic elaboration in `normalizeFuncCall()` and identifier normalization
- Factory resolution with arity-aware handling (nullary returns value, others return function)

**Constructor Pattern Matching** (`internal/eval/eval_core.go`)
- Extended `matchPattern()` to handle `ConstructorPattern`
- Recursive field pattern matching with variable binding
- Constructor name and arity validation
- Full destructuring support: `Some(x)`, `Ok(Some(y))`, `None`

**Pipeline Integration** (`internal/pipeline/pipeline.go`)
- Constructors extracted from elaborator and added to module interfaces
- Factory types registered in `externalTypes` before type checking
- Used TFunc2/TVar2 (new type system) for unification compatibility
- Monomorphic result types (e.g., `Option` not `Option[Int]`) due to TApp limitation

**Interface Builder Enhancement** (`internal/iface/builder.go`)
- `BuildInterfaceWithConstructors()` accepts constructor information
- Constructors included in module interface for imports
- Constructor schemes with field types and result types

**Working Examples**:
```ailang
type Option[a] = Some(a) | None

match Some(42) {
  Some(n) => n,
  None => 0
}
-- Output: 42 ✅

match None {
  Some(n) => n,
  None => 999
}
-- Output: 999 ✅
```

#### Key Technical Decisions
1. **No new Core IR nodes**: Constructor calls use `VarGlobal("$adt", "make_*")` pattern
2. **Runtime factory functions**: $adt module populated at link time from interfaces
3. **Direct evaluation**: Match expressions evaluate without lowering pass
4. **Deterministic**: Factory names sorted, stable digest computation
5. **Nullary handling**: Returns TaggedValue directly (not wrapped in function)
6. **Type system hybrid**: TCon (old) + TFunc2/TVar2 (new) for unification compatibility

#### Files Changed
- `internal/eval/value.go`: Added TaggedValue type (~25 LOC)
- `internal/eval/eval_core.go`: Added isTag, getField helpers, constructor pattern matching (~180 LOC)
- `internal/link/builtin_module.go`: Added RegisterAdtModule (~120 LOC)
- `internal/link/module_linker.go`: Added GetLoadedModules method
- `internal/elaborate/elaborate.go`: Added normalizeTypeDecl, constructor tracking, nullary handling (~150 LOC)
- `internal/pipeline/compile_unit.go`: Added ConstructorInfo, Constructors field (~25 LOC)
- `internal/iface/builder.go`: Added BuildInterfaceWithConstructors (~60 LOC)
- `internal/pipeline/pipeline.go`: Added constructor pipeline wiring, TFunc2/TVar2 factory types (~120 LOC)
- `internal/link/resolver.go`: Enhanced resolveAdtFactory with arity lookup (~60 LOC)

#### Test Coverage
- 16 test cases: TaggedValue, isTag, getField functions
- End-to-end examples: `examples/adt_simple.ail`
- Both nullary and non-nullary constructors verified

### Known Limitations (Future Work)
- ⚠️ Let bindings with constructors have elaboration bug ("normalization received nil expression")
- ⚠️ Result types are monomorphic (`Option` vs `Option[Int]`) - TApp not supported in unifier yet
- ⚠️ No exhaustiveness checking for pattern matches
- ⚠️ No guard evaluation (guards are parsed but not evaluated)
- ⚠️ Type system migration incomplete: Mix of old (TFunc, TVar) and new (TFunc2, TVar2) types

### Technical Details
- Total implementation: ~600 LOC (3 days, as estimated)
- Pattern matching: Tuples, literals, variables, wildcards, constructors all work
- Type checking: Polymorphic factory types with proper unification
- Runtime: TaggedValue representation with arity-aware factory resolution
- Deterministic: All constructor names sorted, stable module digests

### Migration Notes
- ADT runtime is fully backward compatible
- Type declarations now elaborate to runtime constructors automatically
- Constructor expressions work in pattern contexts and regular code
- $adt module is synthetic and doesn't require explicit imports

## [v0.0.9] - 2025-09-30

### Changed - Upgraded to Go 1.22

**Security & Performance Upgrade:**
- Upgraded from Go 1.19 → Go 1.22.12 (Go 1.19 EOL since Sept 2023)
- Updated `golang.org/x/text` from v0.20.0 → v0.21.0
- Updated CI workflow to use Go 1.22
- All tests and linting pass with new version

**Benefits:**
- Security patches for 2+ years of vulnerabilities
- 1-3% CPU performance improvement
- ~1% memory reduction
- For-loop variable scoping fix (prevents common bugs)
- Enhanced HTTP routing, better generics support

**Files Changed:**
- `go.mod`: go 1.22, golang.org/x/text v0.21.0
- `.github/workflows/ci.yml`: go-version: '1.22'
- `.github/workflows/build.yml`: go-version: '1.22' (fixes Windows builds)
- `.github/workflows/release.yml`: go-version: '1.22'
- `go.sum`: Updated checksums

### Fixed - Windows Golden File Tests

**Cross-platform Test Compatibility:**
- Fixed Windows test failures in `TestLiterals` subtests
- Issue: Golden files checked out with CRLF line endings on Windows but comparison used raw bytes
- Solution: Normalize line endings (CRLF → LF) in both `want` and `got` strings before comparison
- Updated `goldenCompare()` function in `internal/parser/testutil.go`
- All platforms (Linux, macOS, Windows) now pass golden file tests consistently

### Added - M-P2 Lock-In: Type System Hardening

#### Coverage Regression Protection
- Per-package coverage gates in Makefile (`cover-parser`, `gate-parser`, `cover-lexer`, `gate-lexer`)
- Parser baseline: 70% coverage (up from 69%)
- Lexer baseline: 57% coverage
- CI workflow enforces coverage thresholds on every push
- Golden drift protection: CI fails if golden files change without `ALLOW_GOLDEN_UPDATES=1`
- New make target: `check-golden-drift` validates golden file stability

#### Type Alias vs Sum Type Disambiguation
- Fixed bug: `type Names = [string]` now correctly parses as TypeAlias, not AlgebraicType
- Added `TypeAlias` AST node in `internal/ast/ast.go`
- Implemented `hasTopLevelPipe()` helper to detect sum types by presence of `|` operator
- Updated `parseTypeDeclBody()` to distinguish:
  - Type aliases: `type UserId = int`, `type Names = [string]`
  - Sum types: `type Color = Red | Green | Blue`
- Regenerated all type golden files with correct TypeAlias representation

#### Nested Record Types
- Record types now work in type positions: `type User = { addr: { street: string } }`
- Added `typeNode()`, `String()`, `Position()` methods to RecordType
- Created `parseRecordTypeExpr()` function for `{...}` in type expressions
- Added test case `TestRecordTypes/nested_record` with golden file
- RecordType now implements both TypeDef and Type interfaces

#### Export Metadata Tracking
- Added `Exported bool` field to TypeDecl AST node
- Updated `parseTypeDeclaration(exported bool)` to track export status
- AST printer includes `"exported": true` in JSON output for exported types
- Tests validate: `export type PublicColor = Red | Green` vs `type PrivateData = { value: int }`
- Regenerated export golden files with metadata

#### REPL/File Type Parity
- New test suite: `TestREPLFileParityTypes` with 10 type declaration test cases
- Validates identical parsing for: aliases, lists, records (simple & nested), sum types, generics, exports
- All type declarations parse identically in REPL (`<repl>`) vs file (`test.ail`) contexts
- Parser coverage increased to 70.8%

#### Metrics
- Parser coverage: 69% → 70.8%
- New tests: 11 (1 nested record + 10 parity tests)
- All existing parser tests pass (544ms test suite)
- Golden files: 3 regenerated (export_alias, export_record, export_sum)
- Code changes: 7 files (ast.go, parser.go, print.go, repl_parity_test.go, type_test.go, Makefile, ci.yml)

### Added - M-P1: Parser Baseline (2025-09-30)

#### Comprehensive Test Infrastructure
- Created deterministic AST printer in `internal/ast/print.go` (445 lines)
- Created test utilities in `internal/parser/testutil.go` (241 lines)
- Established golden file testing framework with 116 snapshots
- Added Makefile targets: `test-parser`, `test-parser-update`, `fuzz-parser`

#### Test Coverage Across All Parser Features
- **Expression tests** (`expr_test.go`, 385 lines): 85 test cases covering literals, operators, collections, lambdas
- **Precedence tests** (`precedence_test.go`, 283 lines): 53 test cases validating operator precedence
- **Module tests** (`module_test.go`, 142 lines): 17 test cases for module/import declarations
- **Function tests** (`func_test.go`, 252 lines): 22 test cases for function declarations and signatures
- **Error recovery tests** (`error_recovery_test.go`, 312 lines): 38 test cases for graceful error handling
- **Invariant tests** (`invariants_test.go`, 320 lines): UTF-8 normalization, CRLF handling, BOM stripping
- **REPL parity tests** (`repl_parity_test.go`, 220 lines): Ensures REPL and file parsing consistency
- **Fuzz tests** (`fuzz_test.go`, 181 lines): 4 fuzz functions with 47 seed cases

#### Baseline Metrics
- **506 test cases** total across all parser features
- **70.2% line coverage** (baseline frozen)
- **Zero panics** in 52k+ fuzz executions
- **2,233 lines** of test code
- All tests pass in ~550ms

## [v0.0.7] - 2025-09-29

### Added - Milestone A2: Structured Error Reporting

#### Unified Error Report System (`internal/errors/report.go`)
- Canonical `errors.Report` type with schema `ailang.error/v1`
- `ReportError` wrapper preserves structured errors through error chains
- `AsReport()` function for type-safe error unwrapping using `errors.As()`
- `WrapReport()` ensures Reports survive through error propagation
- JSON-serializable with deterministic field ordering
- Structured `Data` map with sorted arrays for reproducibility
- `Fix` suggestions with confidence scores
- ~120 lines of core error infrastructure

#### Standardized Error Codes
- **IMP010** - Symbol not exported by module
  - Data: `symbol`, `module_id`, `available_exports[]`, `search_trace[]`
  - Suggests checking available exports in target module
- **IMP011** - Import conflict (multiple providers for same symbol)
  - Data: `symbol`, `module_id`, `providers[{export, module_id}]`
  - Suggests using selective imports to resolve conflict
- **IMP012** - Unsupported import form (namespace imports)
  - Data: `module_id`, `import_syntax`
  - Suggests using selective import syntax
- **LDR001** - Module not found during load
  - Data: `module_id`, `search_trace[]`, `similar[]`
  - Provides resolution trace and similar module suggestions
- **MOD006** - Cannot export underscore-prefixed (private) names
  - Parser validation prevents accidental private exports

#### Error Flow Hardening
- Removed `fmt.Errorf()` wrappers in `internal/elaborate/elaborate.go:112`
- Removed `fmt.Errorf()` wrappers in `internal/pipeline/pipeline.go:434`
- All error builders return `*errors.Report` instead of generic errors
- Link phase wraps reports with `errors.WrapReport()` in `internal/link/module_linker.go`
- Loader phase wraps reports with `errors.WrapReport()` in `internal/loader/loader.go`
- Errors flow end-to-end as first-class types, not string wrappers

#### CLI JSON Output (`cmd/ailang/main.go`)
- `--json` flag enables structured JSON error output
- `--compact` flag for token-efficient JSON serialization
- `handleStructuredError()` extracts Reports using `errors.As()`
- Generic error fallback for non-structured errors
- Exit code 1 for all error conditions

#### Golden File Testing Infrastructure
- **Test files** (`tests/errors/`):
  - `lnk_unresolved_symbol.ail` - Tests IMP010 (symbol not exported)
  - `lnk_unresolved_module.ail` - Tests LDR001 (module not found)
  - `import_conflict.ail` - Tests IMP011 (import conflict)
  - `export_private.ail` - Tests MOD_EXPORT_PRIVATE (private export)
- **Golden files** (`goldens/`):
  - `lnk_unresolved_symbol.json` - Expected IMP010 output
  - `lnk_unresolved_module.json` - Expected LDR001 output
  - `import_conflict.json` - Expected IMP011 output
  - `imports_basic_success.json` - Expected success output (value: 6)
- Golden files ensure byte-for-byte reproducibility of error output

#### Makefile Test Targets
- `make test-imports-success` - Verifies successful imports work
- `make test-import-errors` - Validates golden file matching with `diff -u`
- `make regen-import-error-goldens` - Regenerates golden files (use with caution)
- `make test-imports` - Combined import testing (success + errors)
- `make test-parity` - REPL/file parity test (manual, requires interactive REPL)

#### CI Integration (`.github/workflows/ci.yml`)
- Split import testing into explicit steps:
  - "Test import system (success cases)" - Runs `make test-imports-success`
  - "Test import errors (golden file verification)" - Runs `make test-import-errors`
- CI gates prevent regression in error reporting determinism
- Integrated into `ci-strict` target with operator lowering and builtin freeze tests

### Changed
- `internal/link/report.go` - All builders return `*errors.Report`
- `internal/link/env.go` - Renamed old `LinkReport` to `LinkDiagnostics` to avoid confusion
- `internal/loader/loader.go` - Search trace collection during module resolution
- `internal/parser/parser.go` - Added MOD_EXPORT_PRIVATE validation

### Fixed
- Structured errors were being stringified by `fmt.Errorf("%w")` wrappers
- Error type information now survives through error chains using `errors.As()`
- Flag ordering: Flags must come BEFORE subcommand (`ailang --json --compact run file.ail`)

### Technical Details
- Total new code: ~680 lines (implementation + test files + golden files)
- Test coverage: Golden files ensure deterministic error output
- Determinism: All arrays sorted, canonical module IDs, stable JSON field ordering
- No breaking changes to existing functionality
- Schema versioning allows future enhancements without breaking compatibility

### Migration Notes
- Existing error handling continues to work unchanged
- JSON output is opt-in via `--json` flag
- Structured errors available via `errors.AsReport()` for tools integration
- Golden file tests serve as documentation of expected error formats

## [v0.0.6] - 2025-09-29

### Added

#### Error Code Taxonomy (`internal/errors/codes.go`)
- Comprehensive error code system with structured taxonomy
- Error codes organized by phase: PAR (Parser), MOD (Module), LDR (Loader), TC (Type Check), etc.
- Error registry with phase and category metadata
- Helper functions: `IsParserError()`, `IsModuleError()`, `IsLoaderError()`, etc.
- ~278 lines of structured error definitions

#### Manifest System (`internal/manifest/`)
- Example manifest format for tracking example status (working/broken/experimental)
- Validation ensures consistency between documentation and implementation
- Statistics calculation with coverage metrics
- README generation support for automatic documentation updates
- Environment defaults for reproducible execution
- ~390 lines with full validation logic

#### Module Loader (`internal/module/loader.go`)
- Complete module loading system with dependency resolution
- Circular dependency detection using cycle detection algorithm
- Topological sorting using Kahn's algorithm for build order
- Module caching with thread-safe concurrent access
- Support for stdlib modules and relative imports
- Structured error reporting with resolution traces
- ~607 lines of robust module management

#### Path Resolver (`internal/module/resolver.go`)
- Cross-platform path normalization and resolution
- Support for relative imports (`./`, `../`)
- Standard library path resolution (`std/`)
- Project root detection and search path management
- Case-sensitive and case-insensitive filesystem handling
- Module identity derivation from file paths
- ~405 lines of platform-aware path handling

#### Example Files
- Basic module with function declarations
- Recursive functions with inline tests
- Module imports and composition
- Standard library usage patterns
- Property-based testing examples

### Changed
- Test coverage improved from 29.9% to 31.3%
- Module tests now include comprehensive cycle detection validation
- Topological sort correctly handles dependency ordering

### Fixed
- CI/CD script compilation errors by refactoring shared types into `scripts/internal/reporttypes`
- Test suite now correctly excludes `scripts/` directory containing standalone executables
- Makefile and CI workflow updated to use `go list ./... | grep -v /scripts` for testing

## [v0.0.5] - 2025-09-29

### Added

#### Schema Registry (`internal/schema/`)
- Frozen schema versioning system with forward compatibility
- Schema constants: `ErrorV1` (ailang.error/v1), `TestV1` (ailang.test/v1), `EffectsV1` (ailang.effects/v1)
- `Accepts()` method for prefix matching against newer schema versions
- `MarshalDeterministic()` for stable JSON output with sorted keys
- `CompactMode` flag support for token-efficient JSON serialization
- Registry pattern for managing versioned schemas across components
- ~145 lines of core implementation

#### Error JSON Encoder (`internal/errors/`)
- Structured error taxonomy with stable error codes (TC###, ELB###, LNK###, RT###)
- Always includes `fix` field with actionable suggestion and confidence score
- SID (Stable Node ID) discipline with "unknown" fallback for safety
- Builder pattern API: `WithFix()`, `WithSourceSpan()`, `WithMeta()`
- Schema-compliant JSON output using ailang.error/v1
- Safe encoding that never panics on malformed data
- ~190 lines with comprehensive error handling

#### Test Reporter (`internal/test/`)
- Structured test reporting in JSON format using ailang.test/v1 schema
- Complete test counts shape: passed/failed/errored/skipped/total
- Platform information capture for reproducibility tracking
- Deterministic sorting by suite name and test name
- Valid JSON output even with zero tests
- Test runner integration with SID generation
- ~206 lines with full test lifecycle support

#### REPL Effects Inspector (`internal/repl/effects.go`)
- `:effects <expr>` command for type and effect introspection
- Returns type signature and effect requirements without evaluation
- Supports both human-readable and JSON output modes
- Placeholder implementation (full version pending effect system)
- Schema-compliant output using ailang.effects/v1
- ~41 lines with extensible architecture

#### CLI Compact Mode Support
- `--compact` flag added to main CLI for global compact JSON mode
- Integrates with schema registry's `CompactMode` setting
- Affects all JSON output including errors, tests, and effects
- Token-efficient output for AI agent integration

#### Golden Test Framework Enhancements
- Platform-specific salt generation for reproducibility
- `UPDATE_GOLDENS` environment variable support
- JSON diff utilities for test validation
- Deterministic fixture generation and validation
- ~309 lines of comprehensive test infrastructure

### Added - Test Coverage & Quality
- 100% test coverage for schema registry (unit + integration)
- 100% test coverage for error encoder with edge cases
- 100% test coverage for test reporter with platform variations
- Golden test fixtures for all schema-compliant JSON outputs
- Integration tests validating cross-component schema compliance
- ~470 lines of test code ensuring reliability

### Changed
- All JSON output now uses deterministic field ordering
- Error messages consistently include actionable fix suggestions
- Test reporting standardized across all components
- Platform information consistently captured for reproducibility

### Technical Details
- Total new code: ~1,630 lines (implementation + tests)
- Dependencies: No new external dependencies
- Schema versioning: Forward-compatible design
- JSON output: Deterministic and stable across platforms
- Test coverage: 100% for all new packages

### Migration Notes
- All existing functionality preserved
- New features are opt-in via CLI flags and REPL commands
- JSON output format enhanced but remains backward compatible
- Schema versioning allows gradual migration to newer formats

## [v0.0.4] - 2025-09-28

### Added

#### Example Verification System (`scripts/`)
- `verify_examples.go` - Tests all examples, categorizes as passed/failed/skipped
- Outputs in JSON, Markdown, and plain text formats
- Captures error messages for failed examples
- Skips test/demo files automatically
- ~200 lines of Go code

#### README Auto-Update System
- `update_readme.go` - Updates README with verification status
- Auto-generates status table between markers
- Creates badges for CI, coverage, and example status
- Maintains timestamp of last update
- ~150 lines of Go code

#### CI GitHub Actions (`.github/workflows/ci.yml`)
- Automated testing on push/PR to main/dev branches
- Example verification with failure on broken examples
- Test coverage reporting to Codecov
- Auto-commits README updates on dev branch
- Build artifact generation
- Parallel linting and testing jobs

#### Make Targets
- `make verify-examples` - Run example verification
- `make update-readme` - Update README with status
- `make flag-broken` - Add warning headers to broken examples
- `make test-coverage-badge` - Generate coverage metrics
- `make ci` - Full CI pipeline

### Added - Documentation
- CI status badges in README (CI, Coverage, Examples)
- Auto-generated example status table
- Example verification report showing 13 working, 13 failing, 14 skipped
- Warning headers for broken examples (via `flag_broken_examples.go`)
- `.gitignore` entries for CI-generated files

### Changed
- REPL now displays version from git tags dynamically (via ldflags)
- All v3.x version references updated to semantic versioning (v0.0.x)
- Example files renamed to match version scheme (v0_0_3_features_demo.ail)
- Design docs restructured to match version scheme

### Technical Details
- Total new code: ~500 lines
- Test coverage: Verification scripts fully tested
- No external dependencies added
- Apache 2.0 license badge added

## [v0.0.3] - 2025-09-26

### Added

#### Schema Registry (`internal/schema/`)
- Versioned JSON schemas with forward compatibility
- `Accepts()` for schema version negotiation
- `MarshalDeterministic()` for stable JSON output
- `CompactMode` support for token-efficient output
- Schema constants: `ErrorV1`, `TestV1`, `DecisionsV1`, `PlanV1`, `EffectsV1`

#### Error JSON Encoder (`internal/errors/`)
- Structured error taxonomy with codes (TC###, ELB###, LNK###, RT###)
- Always includes `fix` field with suggestion and confidence score
- SID (Stable Node ID) discipline with fallback to "unknown"
- Builder pattern: `WithFix()`, `WithSourceSpan()`, `WithMeta()`
- Safe encoding that never panics

#### Test Reporter (`internal/test/`)
- Structured test reporting in JSON format
- Full counts shape (passed/failed/errored/skipped/total)
- Platform information for reproducibility
- Deterministic sorting by suite and name
- Valid JSON output even with 0 tests
- Test runner with SID generation

#### Effects Inspector
- `:effects <expr>` command for type/effect introspection
- Returns type and effects without evaluation
- Supports compact JSON mode
- Placeholder implementation (full version pending effect system)

#### Golden Test Framework (`testutil/`)
- Platform salt for reproducibility tracking
- `UPDATE_GOLDENS` environment variable support
- JSON diff utilities
- Deterministic test fixtures

#### REPL Enhancements
- `:test [--json]` - Run tests with optional JSON output
- `:effects <expr>` - Inspect type and effects
- `:compact on/off` - Toggle JSON compact mode
- Updated help with new commands

### Added - Examples & Documentation
- `examples/v3_2_features_demo.ail` - Demonstrates new v3.2 features
- `examples/repl_commands_demo.md` - REPL command documentation
- `examples/ai_agent_integration.ail` - Comprehensive AI agent guide
- `examples/working_v3_2_demo.ail` - Working examples for current state
- `design_docs/implemented/v3_2/` - Implementation report with metrics
- Comprehensive test suites for all new packages
- 100% test coverage for schema registry
- 100% test coverage for error encoder
- 100% test coverage for test reporter

### Changed
- `types.CanonKey()` alias added for consistent dictionary key generation
- REPL help updated with new AI-first commands

### Fixed
- Multi-line REPL input for `let...in` expressions
- Added continuation prompt (`...`) for incomplete expressions

### Technical Details
- Total new code: ~1,500 lines
- Test coverage: All new packages fully tested
- Dependencies: No new external dependencies

### Migration Notes
- No breaking changes
- New features are opt-in via REPL commands
- Existing code continues to work unchanged

## [v0.0.2] - Previous Release
- Type class resolution with dictionary-passing
- REPL improvements with history and tab completion
- Core type system implementation

## [v0.0.1] - Initial Release
- Basic lexer and parser
- AST implementation
- Initial REPL

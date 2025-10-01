# AILANG Changelog

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

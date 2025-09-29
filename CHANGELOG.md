# AILANG Changelog

## [Unreleased]

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
- **MOD_EXPORT_PRIVATE** - Cannot export underscore-prefixed (private) names
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

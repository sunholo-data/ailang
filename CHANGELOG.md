# AILANG Changelog

## [Unreleased]

### Fixed
- CI/CD script compilation errors by refactoring shared types into `scripts/internal/reporttypes`
- Test suite now correctly excludes `scripts/` directory containing standalone executables
- Makefile and CI workflow updated to use `go list ./... | grep -v /scripts` for testing

## [v0.0.4] - 2025-09-28

### Added - Schema & Compact Mode Baseline

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

### Added - CI/CD Infrastructure

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

### Added - AI-First Features

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

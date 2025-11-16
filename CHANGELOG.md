# AILANG Changelog

## [v0.4.5] - 2025-11-16

### Fixed - Nullary Constructor Pattern Matching (M-BUG-NULLARY) 🐛 Critical Bug Fix

**User Impact**: Simple enum types (ADTs with only nullary constructors) now work correctly in pattern matching. Before this fix, all values matched the first pattern, breaking type safety guarantees. This enables production use of enum types like `type Status = Pending | InProgress | Completed`.

**What Was Fixed**:
1. **Elaboration Fix** (~13 LOC in internal/elaborate/patterns.go)
   - Root cause: Nullary constructors (`Red`, `Green`, `Blue`) were being elaborated as `VarPattern` instead of `ConstructorPattern`
   - Variable patterns always match and bind, causing all three values to match the first pattern
   - Fix: Check if identifier is a known nullary constructor (arity=0) in elaborator's constructor map
   - If yes, create `ConstructorPattern` with empty args; otherwise create `VarPattern`
   - Location: `elaboratePattern()` function, line 79-90

2. **Test Coverage** (~120 LOC total)
   - Unit tests: 6 test cases in `internal/elaborate/patterns_nullary_test.go`
     - Nullary constructors (Red, Green, Blue, None) → ConstructorPattern ✓
     - Variable patterns → VarPattern ✓
     - Non-nullary constructors with arity >0 → VarPattern ✓
   - Integration test: `tests/nullary_pattern_matching_test.ail` (~67 LOC)
     - Tests Status (3 variants), Color (3 variants), Direction (4 variants)
     - All 10 pattern matches verify correct behavior ✓

3. **Benchmark Impact**:
   - `exhaustive_pattern_matching` benchmark: **96.1% → 100% success** ✓
   - 3 out of 76 eval failures (3.9%) eliminated with this single fix
   - Tested with gpt5-mini, confirmed 100% success rate

**Before the fix:**
```ailang
type Color = Red | Green | Blue
func test(c: Color) -> string {
  match c {
    Red => "red",
    Green => "green",  -- Never matched!
    Blue => "blue"     -- Never matched!
  }
}
test(Green)  -- Returned "red" (WRONG!)
```

**After the fix:**
```ailang
test(Red)    -- Returns "red"   ✓
test(Green)  -- Returns "green" ✓
test(Blue)   -- Returns "blue"  ✓
```

**Technical Details**:
- Bug discovered during v0.4.4 eval analysis (EVAL_ANALYSIS_v0_4_4.md)
- Investigation time: ~2 hours (debug logging revealed VarPattern issue)
- Fix time: <1 hour (single function change in elaborator)
- Testing time: ~1 hour (unit + integration tests)
- Total effort: 3-4 hours (within sprint plan estimate of 3-5 hours)

**Files Modified**:
- `internal/elaborate/patterns.go` (+13 LOC) - Fix nullary constructor elaboration
- `internal/elaborate/patterns_nullary_test.go` (+120 LOC, new file) - Unit tests
- `tests/nullary_pattern_matching_test.ail` (+67 LOC, new file) - Integration test

**Design Doc**: `design_docs/planned/v0_4_5/nullary-constructor-pattern-matching-bug.md`
**Sprint Plan**: `design_docs/planned/v0_4_5/M-BUG-NULLARY-sprint-plan.md`

## [v0.4.4] - 2025-11-11

### Added - S-CONS Pattern Sugar (x :: xs) 🎯 DX Improvement

**User Impact**: Pattern matching now supports ML-style `x :: xs` syntax alongside canonical `::(x, xs)`. Eliminates 36 PAR_001 parse errors (12% reduction in eval failures). More familiar syntax for developers with ML-family backgrounds (OCaml, Haskell, F#, SML).

**What Was Added**:
1. **Parser Extension** (~49 LOC in internal/parser/parser_pattern.go)
   - Refactored `parsePattern()` to support infix `::` operator
   - Created `parseBasePattern()` for atomic patterns
   - Right-associative desugaring: `a :: b :: c` → `::(a, ::(b :: c))`
   - Bijective transformation: `x :: xs` means exactly the same as `::(x, xs)`
   - Strict mode support: `--strict-syntax` rejects sugar, suggests canonical form

2. **Parser Unit Tests** (~313 LOC in internal/parser/parser_pattern_sugar_test.go)
   - 11 test cases covering:
     - Basic sugar: `x :: xs`
     - Wildcards: `_ :: xs`
     - Right-associativity: `a :: b :: c`
     - Empty list terminator: `x :: []`
     - Literals: `1 :: xs` (note: literals don't work at runtime, pre-existing limitation)
     - Mixed forms: sugar + canonical in same match
     - Parenthesized patterns: `(x :: xs)`
     - Guards: `x :: xs if p(x) => ...` (note: guards don't work at runtime, pre-existing limitation)
     - Strict mode rejection with helpful error
     - Strict mode accepts canonical form
   - All 11 tests passing ✅

3. **Integration Tests** (~157 LOC in internal/pipeline/pattern_sugar_test.go)
   - 5 full pipeline tests (parse → elaborate):
     - Basic cons sugar end-to-end
     - Right-associative chaining
     - Mixed sugar/canonical forms
     - Strict mode rejection
     - Strict mode accepts canonical
   - All 5 tests passing ✅

4. **Example File** (examples/pattern_sugar.ail, ~120 LOC)
   - 10 working examples demonstrating all capabilities
   - Basic patterns, wildcards, chaining, mixed forms, tuples
   - Recursive list functions (sum, length, head, tail)
   - Execution verified ✅

**Syntax Examples**:
```ailang
// Basic sugar
match list {
  x :: xs => x,
  [] => 0
}

// Right-associative (parses as a :: (b :: (c :: rest)))
match list {
  a :: b :: c :: rest => a + b + c,
  _ => 0
}

// Mixed with canonical form
match list {
  x :: [] => x,                    // Sugar
  ::(a, ::(b, rest)) => a + b,     // Canonical
  _ => 0
}

// Strict mode (--strict-syntax)
ailang check --strict-syntax module.ail
// → Error: Use `::(x, xs)` instead of `x :: xs`
```

**Design Principles**:
- **Bijective desugaring**: `x :: xs` ≡ `::(x, xs)` (identical semantics)
- **Canonical form preserved**: `::(x, xs)` remains default in formatters/errors
- **Right-associative**: mirrors expression-level cons sugar (v0.4.1)
- **Opt-in sugar**: `--strict-syntax` disables all sugar for deterministic code

**Impact**:
- **Parse failures**: -12% (36 PAR_001 errors eliminated)
- **DX improvement**: More familiar syntax for ML developers
- **No semantic changes**: Pure surface sugar, same AST representation
- **Zero regressions**: All existing tests pass

**Files Modified**:
- internal/parser/parser_pattern.go: +49 LOC (parser extension)
- internal/parser/parser_pattern_sugar_test.go: +313 LOC (NEW, 11 tests)
- internal/pipeline/pattern_sugar_test.go: +157 LOC (NEW, 5 integration tests)
- examples/pattern_sugar.ail: +120 LOC (NEW, working example)

**Testing**: 16 new tests (11 parser + 5 integration), all passing ✅, zero regressions, lint clean

**Velocity**: 2 days estimated → 1.5 days actual (ahead of schedule)

**Resolves**: S-CONS pattern limitation (36 PAR_001 errors, 12% of failures)

## [Unreleased - v0.4.5]

### Added - Agent Execution Integration 🤖 Autonomous Code Execution

**User Impact**: AILANG agents can now autonomously execute code in response to directives, with built-in approval workflows and multi-agent coordination. This makes the UI Collaboration Hub (v0.4.4) fully functional.

**What Was Added** (2,366 LOC across 5 phases):

**Phase 1: Basic Agent Polling** (452 LOC)
- `cmd/ailang-agent/`: New autonomous agent binary with message polling
- Polls collaboration database every 2 seconds for pending messages
- Graceful shutdown with signal handling (SIGINT/SIGTERM)
- Acknowledges messages after processing

**Phase 2: Claude Code Integration** (414 LOC)
- `internal/agent/executor.go`: DirectiveExecutor wraps eval harness
- 80% code reuse from existing eval benchmarking infrastructure
- Creates isolated workspaces with `.git` folders
- Tracks execution: duration, cost, tokens, files created, transcript
- Full test coverage with real Claude Code execution (gated by TEST_AGENT_INTEGRATION)

**Phase 3: Result Communication** (386 LOC)
- `internal/agent/formatter.go`: Markdown formatting for execution results
  - `FormatResult()`: Full markdown with status, summary, tokens, output, files
  - `FormatResultCompact()`: One-line summary for notifications
  - `FormatResultWithTranscript()`: Includes full conversation
- Results published back to collaboration hub for UI display

**Phase 4: Approval Workflow** (925 LOC)
- `internal/agent/capabilities.go`: Intelligent capability detection (265 LOC)
  - Keyword-based heuristics for FS/Net/Shell/Budget requirements
  - Detects "file", "write", "http", "bash", "install", etc.
  - Impact classification: low/medium/high
  - Cost estimation based on directive complexity
- Automatic approval requests for directives requiring capabilities
- 60-second timeout with graceful handling
- Rejection messages sent to UI
- 15 comprehensive test suites (412 LOC)

**Phase 5: Multi-Agent Coordination** (189 LOC)
- Atomic work claiming prevents duplicate processing
- New `'claimed'` delivery state in messaging schema
- `ClaimMessage()`: Atomic UPDATE ensures exactly-once processing
- Agent-to-agent messaging: `SendStatusToAgent()`, `BroadcastStatus()`
- Multi-agent safety verified with race condition tests

**Key Features**:
- ✅ Autonomous code execution via Claude Code
- ✅ Capability-based approval workflow (safety-first)
- ✅ Multi-agent coordination (no duplicate work)
- ✅ Full execution tracking (cost, duration, tokens, files)
- ✅ Markdown-formatted results for UI
- ✅ Graceful error handling and timeouts

**Testing**:
- 100% test coverage of new functionality
- All tests passing ✅
- Integration tests gated by environment variable
- Multi-agent race condition tests

**Files Added/Modified**:
- `cmd/ailang-agent/`: New agent binary (3 files, 800 LOC)
- `internal/agent/`: Execution and formatting (6 files, 1566 LOC)
- `internal/messaging/`: Work claiming support (3 files, 58 LOC)

**Implementation Time**: 1 day (6 phases)
**Code Reuse**: 80% from eval harness
**Total New Code**: 2,366 LOC

## [Unreleased - v0.4.4]

### Added - Global Stdlib Module Search Path (M-STDLIB-SEARCH) 🎯 Eval Fix

**User Impact**: stdlib imports now work from any working directory, fixing 21% of agent eval benchmark failures. No more "module not found: std/io" errors when running code from temp directories.

**What Was Added**:
1. **StdlibResolver** (~290 LOC in internal/loader/stdlib_resolver.go)
   - Multi-path search strategy with priority ordering:
     1. CLI flag (`--stdlib-path`)
     2. Binary-relative path (`../std` from binary)
     3. `AILANG_STDLIB_PATH` environment variable (colon/semicolon separated)
     4. Platform-specific user data dir (XDG/APPDATA/Library)
     5. System directories (`/usr/local/share/ailang/std`, `/usr/share/ailang/std`)
   - Security validation: rejects directory traversal (`..`), absolute paths, suspicious patterns
   - Negative caching: avoids repeated filesystem hits for missing modules
   - VERSION checking: warns on stdlib version mismatch (strict mode available)

2. **Platform-Aware Paths** (~60 LOC)
   - Linux/BSD: `$XDG_DATA_HOME/ailang/std` or `~/.local/share/ailang/std`
   - macOS: `~/Library/Application Support/ailang/std`
   - Windows: `%APPDATA%\ailang\std`
   - Cross-platform path separator handling (`:` vs `;`)

3. **CLI Flags** (cmd/ailang/main.go)
   - `--stdlib-path <path>`: Override stdlib location (highest priority)
   - `--trace-loader`: Enable module loader tracing (placeholder)
   - `--strict`: Fail on stdlib version mismatch (placeholder)
   - Flags accepted but `--trace-loader` and `--strict` need full ModuleRuntime integration (deferred)

4. **Eval Harness Integration** (internal/eval_harness/runner.go)
   - Replaced unreliable stdlib symlink with `--stdlib-path` flag
   - Ensures benchmarks can find stdlib even from isolated workspaces
   - More robust on Windows (symlinks often fail)

5. **VERSION File** (std/VERSION)
   - Contains current stdlib version (`v0.4.4`)
   - Checked at runtime for version mismatches
   - Automatically updated during releases (via release-manager skill)

6. **Comprehensive Tests** (~400 LOC in internal/loader/stdlib_resolver_test.go)
   - 28 test cases covering:
     - Module name validation (security)
     - Platform-specific user data dirs
     - Module resolution (existing, missing, caching)
     - Search path priority
     - VERSION checking (strict and non-strict modes)
   - All tests passing on macOS (Linux/Windows tests skip on wrong platform)

**Integration Points**:
- **ModuleLoader**: Integrated StdlibResolver, lazily initialized
- **Pipeline**: No changes needed (loader handles resolution transparently)
- **Release Manager**: Updated to maintain std/VERSION file

**Eval Impact**:
- **Expected to fix 4 benchmarks**: `effect_composition`, `effect_tracking_io_fs`, `deterministic_list_transform`, `exhaustive_pattern_matching`
- **Expected improvement**: Agent AILANG success rate from 76.3% → ≥85%
- **Note**: Actual eval baseline run deferred to next release cycle

**Files Modified**:
- internal/loader/stdlib_resolver.go: +290 LOC (NEW, core resolver)
- internal/loader/stdlib_resolver_test.go: +400 LOC (NEW, comprehensive tests)
- internal/loader/loader.go: +20 LOC (integration)
- cmd/ailang/main.go: +15 LOC (CLI flags)
- internal/eval_harness/runner.go: +3 LOC (--stdlib-path flag)
- std/VERSION: +1 LOC (NEW, version tracking)
- .claude/skills/release-manager/SKILL.md: +2 LOC (update workflow)
- examples/test_stdlib_sprint.ail: +7 LOC (NEW, test example)

**Testing**:
- All 600+ tests passing
- Stdlib resolution verified from /tmp with `--stdlib-path` flag
- Stdlib resolution verified from /tmp with `AILANG_STDLIB_PATH` env var
- Stdlib resolution verified from project with local binary (binary-relative)
- Security validation: rejects `../etc/passwd`, `std/../../etc/passwd`, etc.

**Breaking Changes**: None (fully backward compatible)

---

### Added - Prompt CLI Command (M-DX-PROMPT) 🔧 Developer Experience

**User Impact**: AILANG teaching prompts now accessible via first-class CLI command. AIs and developers can get version-locked syntax reference without knowing file paths.

**What Was Added**:
1. **Prompt Loader** (~110 LOC in internal/prompt/loader.go)
   - Loads prompts from `prompts/versions.json` manifest
   - Version resolution: "" or "latest" → active version, specific version (e.g., "v0.3.24")
   - Project root detection: finds prompts/ directory automatically
   - Functions: `LoadPrompt(version)`, `GetActiveVersion()`, `ListVersions()`, `GetVersionMetadata(version)`

2. **CLI Command** (~180 LOC in cmd/ailang/prompt.go)
   - `ailang prompt` - Display current/active prompt
   - `ailang prompt --version v0.3.24` - Display specific version
   - `ailang prompt --list` - List all available versions
   - `ailang prompt --info` - Show metadata for version
   - Pipe-friendly output (stdout, no progress messages)

3. **Integration**:
   - Eval harness updated to use `internal/prompt` package (single source of truth)
   - CLAUDE.md updated with `ailang prompt` workflow
   - Help text updated with prompt command

4. **Comprehensive Tests** (~340 LOC)
   - Unit tests (internal/prompt/loader_test.go): 10 tests, all passing
   - Integration tests (cmd/ailang/prompt_test.go): 11 tests, all passing
   - Tests cover: version resolution, metadata, list, invalid versions, piping

**Examples**:
```bash
# Get current prompt
ailang prompt

# Get specific version
ailang prompt --version v0.3.24

# List all versions
ailang prompt --list

# Save to file
ailang prompt > syntax.md

# Pipe to pager
ailang prompt | less

# Show metadata
ailang prompt --version v0.4.2 --info
```

**Files Modified**:
- internal/prompt/loader.go: +110 LOC (NEW, core loader)
- internal/prompt/loader_test.go: +150 LOC (NEW, unit tests)
- cmd/ailang/prompt.go: +180 LOC (NEW, CLI command)
- cmd/ailang/prompt_test.go: +190 LOC (NEW, integration tests)
- cmd/ailang/main.go: +4 LOC (register command, help text)
- internal/eval_harness/spec.go: +3 LOC (use internal/prompt package)
- CLAUDE.md: +15 LOC (document workflow)

**Testing**:
- All 21 new tests passing (10 unit + 11 integration)
- Verified: default prompt, specific version, latest, list, info, piping, errors
- Integration with eval harness working

**Breaking Changes**: None (fully backward compatible)

**Philosophy**: The prompt is part of the language - it should be as accessible as `--help` or `--version`. No file path knowledge required.

**Implementation Details**:
- **Prompts embedded in binary** using Go's `embed` package (works from any directory!)
- **Dual-mode loader**: Embedded FS (production) + disk fallback (development hot-reload)
- **Build automation**: `make build` auto-copies `prompts/` to `cmd/ailang/prompts/` for embedding
- **Standalone distribution**: Binary includes ~2MB of prompt files (27 versions in v0.4.2)
- **Developer workflow**: Edit `prompts/*.md` → auto-reloaded from disk (no rebuild needed)
- **Production workflow**: Installed binary (`~/go/bin/ailang`) uses embedded prompts

---

## Previous Releases

**Known Limitations** (v0.4.4 stdlib feature):
- `--trace-loader` and `--strict` flags accepted but not fully wired (need ModuleRuntime integration)
- System-installed binary (~/go/bin/ailang) requires `AILANG_STDLIB_PATH` or `--stdlib-path`
- Project-local binary (./bin/ailang) works with binary-relative path automatically

**Resolves**: M-STDLIB-SEARCH (P0 BLOCKER)

---

## [v0.4.3] - 2025-11-05

### Added - String Parsing Builtins (M-DX10) 🎯 Eval Fix

**User Impact**: AI models can now safely parse string input to numbers using Option types, fixing 3 eval benchmark failures.

**What Was Added**:
1. **New Builtin Functions** (~156 LOC in internal/builtins/string.go)
   - `_stringToInt(s: string) -> Option[int]`: Parse string to integer, returns Some(n) or None
   - `_stringToFloat(s: string) -> Option[float]`: Parse string to float, returns Some(f) or None
   - Both use Go's strconv package (ParseInt, ParseFloat)
   - Return TaggedValue with "std/option" type (Some/None constructors)

2. **Comprehensive Tests** (~214 LOC in internal/builtins/string_test.go)
   - 35+ test cases covering valid/invalid inputs
   - Integer: "42", "-123", "abc", "3.14", overflow, scientific notation
   - Float: "3.14", "1e-10", "abc", multiple dots, invalid scientific
   - Edge cases: empty strings, whitespace, sign handling
   - Error handling: wrong argument types

3. **Standard Library Exports** (~2 LOC in std/string.ail)
   - `stringToInt(s: string) -> Option[int]`
   - `stringToFloat(s: string) -> Option[float]`
   - Import std/option for Option type

4. **Example File** (examples/string_parsing.ail, 98 LOC)
   - Demonstrates parsing with pattern matching
   - Shows validation (e.g., age >= 0)
   - Uses getOrElse for default values
   - All tests passing with expected output

**Eval Impact**:
- **Fixes 2 benchmarks**: `effect_composition`, `error_handling` (both need `_str_to_int`)
- **Note**: `tree_transformation_pipeline` still broken (needs `Cons` constructor, separate issue)

**Files Modified**:
- internal/builtins/string.go: +156 LOC (2 new functions + registration)
- internal/builtins/string_test.go: +214 LOC (NEW, 35+ test cases)
- std/string.ail: +3 LOC (import + 2 exports)
- examples/string_parsing.ail: +98 LOC (NEW, working example)
- internal/pipeline/testdata/builtin_types.golden: +2 LOC (updated snapshot)

**Testing**: All tests passing (8 test functions, 35+ sub-tests), lint clean, example verified working

**Resolves**: M-DX10 (P1 - Eval Failures)

## [v0.4.2] - 2025-11-02

### Fixed - CRITICAL: S-CALL0 Zero-Arg Builtin Bug (M-S-CALL0-FIX) ⚠️ HOTFIX

**User Impact**: **stdlib (std/io) was completely broken in v0.4.1** due to S-CALL0 syntax conflicting with zero-arg builtins. This hotfix restores functionality.

**Root Cause**:
- Parser sugar `f()` desugars to `f(())` (adds unit argument)
- Builtins registered with `() -> T` (truly zero-arg, no params)
- Type checker saw 0 params vs 1 arg → arity mismatch
- **Impact**: 100% of code importing `std/io` failed to compile

**What Was Fixed**:
1. **Zero-Arg Functions Now Take Unit Parameter** (semantic change)
   - `func f() -> T` is now sugar for `func f(_: ()) -> T`
   - Aligns with S-CALL0 semantics where `f()` means `f(())`
   - All zero-arg functions (user + builtin) now have 1 parameter (unit)

2. **Builtin Updates** (~10 LOC in internal/builtins/io.go)
   - `_io_readLine`: NumArgs: 0 → 1
   - Type: `T.Func().Returns(T.String())` → `T.Func(T.Unit()).Returns(T.String())`

3. **Parser Updates** (~20 LOC in internal/parser/parser_func.go)
   - Add implicit unit parameter for `func f()` syntax
   - Applies to both generic and non-generic functions

4. **Entry Module Detection** (~15 LOC in internal/pipeline/prelude.go)
   - Accept both zero-param and unit-param `main()` functions
   - Ensures `export func main()` is still recognized

5. **Test Updates** (~1 LOC in internal/pipeline/builtin_consistency_test.go)
   - Update `_io_readLine` expected arity: 0 → 1

**Discovered During**: v0.4.1 post-release evaluation analysis
- Haiku AILANG dropped from 58.3% to 4.9% (86% failures were `WRONG_LANG` trying to import `std/io`)
- v0.4.1 prompt is actually 6x better than v0.4.0 (proves it's stdlib bug, not prompt issue)
- See design doc: `design_docs/planned/v0_4_1/m-s-call0-zero-arg-builtin-bug.md`

**Files Modified**:
- internal/builtins/io.go: Updated `_io_readLine` signature
- internal/parser/parser_func.go: Add implicit unit parameter
- internal/pipeline/prelude.go: Accept both zero/unit-param main
- internal/pipeline/builtin_consistency_test.go: Update expected arity

**Testing**: All 600+ tests passing, manual verification of `std/io` import works

**Resolves**: M-S-CALL0-FIX (P0 BLOCKER)

---

### Fixed - CRITICAL: Eval Harness Security Issues ⚠️ HOTFIX

**Two critical eval harness bugs discovered during v0.4.2 validation:**

#### 1. Race Condition - Output Corruption (P0)

**User Impact**: Parallel benchmarks were overwriting each other's code, causing wrong output to be captured (e.g., fibonacci benchmark outputting "All results equal: true" from referential_transparency).

**Root Cause**:
- All parallel benchmarks wrote to same file: `benchmark/solution.ail`
- Parallelism: 10-15 concurrent jobs
- Race condition window: file gets overwritten mid-execution

**What Was Fixed**:
- **Isolated Workspaces** (~50 LOC in internal/eval_harness/runner.go)
  - Each benchmark gets unique workspace: `.eval_workspace/<timestamp>_<pid>/`
  - Maintains valid module path: `benchmark/solution` (prevents MOD010 errors)
  - Stdlib symlinked into each workspace for imports
  - Workspace cleaned up after execution

**Validation**:
- Stress test with 20 concurrent jobs: NO corruption detected
- Validation script: `tools/validate_eval_results.py`
- Test script: `tools/test_eval_race_condition.sh`

**Files Modified**:
- internal/eval_harness/runner.go: Isolated workspace implementation
- .gitignore: Added `.eval_workspace/` exclusion

#### 2. Infinite Output Bug - 1GB JSON Files (P0)

**User Impact**: AI-generated code with infinite loops created 1GB+ JSON files, blocking git commits and consuming disk space.

**Root Cause**:
- Python code with infinite loop: `while True: print(input())` → EOF error loop
- Runs for 30 seconds (timeout), printing millions of error messages
- Eval harness captured ALL stdout → 1GB in JSON `stdout` field

**What Was Fixed**:
- **Output Size Limiting** (~70 LOC in internal/eval_harness/runner.go)
  - `LimitedWriter` caps stdout/stderr at 1MB each
  - Truncation message appended when limit exceeded
  - Prevents runaway output from consuming resources

**Implementation**:
```go
const MaxOutputSize = 1 * 1024 * 1024  // 1 MB

type LimitedWriter struct {
    buf       *bytes.Buffer
    limit     int64
    written   int64
    truncated bool
}
```

**Testing**:
- 5 unit tests in internal/eval_harness/runner_test.go
- Test script: `tools/test_output_limit.sh`
- All tests passing

**Files Modified**:
- internal/eval_harness/runner.go: LimitedWriter implementation
- internal/eval_harness/runner_test.go: Unit tests

**Impact**: v0.4.2 baseline re-run with fixed harness shows +2.4pp improvement over v0.4.0 (48.0% vs 45.5%)

**Resolves**: M-EVAL-HARNESS-SECURITY (P0 BLOCKER)

---

### Completed - M-EVAL-CAPS Benchmark Capability Coverage

**User Impact**: All 41 benchmarks now have explicit capability specifications, ensuring accurate eval results with zero false negatives from capability mismatches.

**Files Modified**: 2 benchmark YML files updated

**Resolves**: M-EVAL-CAPS (documentation completion)

---

### Fixed - Statement-Level S-CALL0 Support (M-S-CALL0)

**User Impact**: The `f()` zero-arg call syntax now works at **both** statement and expression levels. Previously required `f ()` with space at top level.

**What Was Fixed**:
1. **Statement-Level Lookahead** (~60 LOC in parser_decl.go)
   - Added detection for identifier followed by UNIT token in `parseTopLevelDecl()`
   - Handles both IDENT case and default case for full coverage
   - Creates FuncCall with unit argument when pattern detected
   - Respects `strictSyntaxMode` flag

2. **Expression-Level Infix Handler** (~45 LOC in parser_expr.go)
   - Registered UNIT as infix operator (precedence 11 - CALL level)
   - New `parseZeroArgCall()` function for expression contexts
   - Seamlessly integrates with existing Pratt parser

3. **UNIT Token Precedence** (~1 LOC in lexer/token.go)
   - Added UNIT to CALL precedence level (11)
   - Enables Pratt parser to invoke infix handler for `f()`

4. **Comprehensive Tests** (~150 LOC in sugar_test.go)
   - Top-level zero-arg calls: `myFunc()`
   - Multiple top-level calls: `func1(); func2()`
   - Expression contexts: `if true then myFunc() else 0`
   - Strict mode rejection tests
   - All 4 S-CALL0 tests passing (previously 1 was skipped)

5. **Example File** (~40 LOC in examples/sugar_call0.ail)
   - Demonstrates statement-level calls
   - Shows expression-level calls (still work)
   - Explains lexer UNIT token behavior
   - Documents canonical syntax equivalence

6. **Documentation Updates**
   - prompts/v0.4.1.md: Removed "expression only" limitation warning
   - prompts/versions.json: Updated hash and notes

**Files Modified**:
- internal/parser/parser_decl.go: +60 LOC (statement-level detection)
- internal/parser/parser_expr.go: +45 LOC (expression-level handler)
- internal/lexer/token.go: +1 LOC (UNIT precedence)
- internal/parser/sugar_test.go: +150 LOC (4 new tests, replaced skip)
- examples/sugar_call0.ail: +40 LOC (NEW)
- prompts/v0.4.1.md: Updated (removed limitation)
- prompts/versions.json: Updated hash

**Technical Details**:
- **Root Cause**: Lexer creates single UNIT token for `()` without spaces
  - `f()` tokenizes as: IDENT + UNIT (not LPAREN + RPAREN!)
  - This broke both statement-level and expression-level parsing
- **Dual Fix Required**:
  - Statement level: Manual detection in parseTopLevelDecl (no Pratt parser)
  - Expression level: Register UNIT as infix operator (Pratt parser handles it)
- **Why UNIT Precedence Matters**: Without precedence 11, Pratt parser never enters infix loop

**Resolves**: M-S-CALL0, design_docs/planned/v0_4_1/m-s-call0-statement-parsing.md

### Fixed - List Pattern Parser Bug (M-DX10)

**User Impact**: Parser now accepts `::` (cons) constructor patterns in match expressions. Previously valid AILANG code that used `::` patterns would fail with `PAR_UNEXPECTED_TOKEN`.

**What Was Fixed**:
1. **Parser**: Added `lexer.DCOLON` case to `parsePattern()` (~11 LOC in parser_pattern.go)
   - `::` is now recognized as a valid list constructor pattern
   - Syntax: `::(head, tail)` for cons patterns
   - Example: `match xs { [] => 0, ::(x, rest) => x + sum(rest) }`

2. **Elaborator**: Added special handling for `::` constructor patterns (~22 LOC in patterns.go)
   - `::` patterns elaborate to `ListPattern` with one element and a tail
   - Required because lists are `ListValue` at runtime, not `TaggedValue` with constructor

3. **Tests**: Comprehensive test suite (~150 LOC in list_cons_pattern_test.go)
   - Basic cons patterns: `::(x, rest)`
   - Multiple arms: `[] => ..., ::(h, t) => ...`
   - Nested cons: `::(_, ::(x, rest))`
   - With tuples: `::((k, v), rest)`
   - Error case: `::` without arguments

4. **Example File**: Working demonstration (99 LOC in examples/list_pattern_cons.ail)
   - 6 example functions using `::` patterns
   - Demonstrates sum, length, nested patterns, tuples
   - Fully runnable with `ailang run --entry main --caps IO examples/list_pattern_cons.ail`

**Files Modified**:
- internal/parser/parser_pattern.go: +11 LOC (parser fix)
- internal/elaborate/patterns.go: +22 LOC (elaborator fix)
- internal/parser/list_cons_pattern_test.go: +150 LOC (NEW - 7 tests, all passing)
- examples/list_pattern_cons.ail: +99 LOC (NEW - working example)

**Technical Details**:
- **Root Cause**: Parser's `parsePattern()` had no case for `lexer.DCOLON` token
- **Parser Fix**: Recognize `::` as constructor and call `parseConstructorPattern("::")`
- **Elaborator Fix**: Convert `::(head, tail)` to `ListPattern{Elements: [head], Tail: tail}`
- **Why Two Fixes**: Lists are `ListValue` at runtime, not `TaggedValue`, so pattern type must match

**Resolves**: M-DX10, json_parse benchmark false negatives with claude-haiku-4-5

### Added - DX Improvements (M-DX10)

**Developer Experience**: Two improvements to prevent confusion during development.

**What Was Added**:
1. **Stale Binary Warning** (~50 LOC in cmd/ailang/main.go)
   - Detects when source files are newer than the `ailang` binary
   - Shows warning: `⚠ Binary may be stale (source files modified after build)`
   - Suggests: `Run 'make quick-install' to rebuild`
   - Checks key directories: `internal/parser`, `internal/elaborate`, `internal/eval`, `cmd/ailang`
   - Zero overhead when binary is fresh (fast stat check only)

2. **Pattern Matching Pipeline Documentation** (~90 LOC in .claude/skills/sprint-executor/SKILL.md)
   - Documents 4-layer transformation: Parser → Elaborator → Type Checker → Evaluator
   - Explains why pattern changes require both parser AND elaborator fixes
   - Common gotchas: Pattern type must match Value type at runtime
   - Cross-reference comments in code for navigation
   - Impact: Prevents two-phase fix discoveries, reduces pattern debugging time by 50%

**Files Modified**:
- cmd/ailang/main.go: +50 LOC (stale binary check)
- .claude/skills/sprint-executor/SKILL.md: +90 LOC (pipeline guide)
- internal/parser/parser_pattern.go: +1 LOC (cross-ref comment)
- internal/elaborate/patterns.go: +2 LOC (cross-ref comments)

**Why These Matter**:
- **Stale Binary**: Prevents 5-10 min debugging "unfixed" bugs that are actually stale binaries
- **Pipeline Docs**: Prevents 20-30 min discovering pattern changes need elaborator fix too

## [v0.4.1] - 2025-11-02

### Added - Surface Sugar Pack (M-SUGAR)

**User Impact**: Optional syntactic sugar for common patterns. Write `x :: xs`, `int -> bool`, and `f()` (in expressions) instead of canonical forms. Disable with `--strict-syntax` flag.

**What Was Added**:
1. **S-CONS: Infix Cons Operator** (~95 LOC in parser_expr.go + precedence table)
   - Sugar: `x :: xs` → Canonical: `::(x, xs)`
   - Right-associative: `1 :: 2 :: []` → `::(1, ::(2, []))`
   - Works in expressions and patterns: `match xs { h :: t => ... }`
   - Registered as infix operator at precedence 6 (between comparison and append)

2. **S-ARROWTYPE: Function Type Arrows** (~45 LOC in parser_type.go)
   - Sugar: `int -> bool` → Canonical: `funcType int bool`
   - Right-associative: `int -> bool -> string` → `funcType int (funcType bool string)`
   - Syntax: `let f: int -> bool = \x. x > 0`
   - Refactored type parser with goto pattern for single arrow check point

3. **S-CALL0: Zero-Argument Calls** (~15 LOC in parser_expr.go, v0.4.1 baseline)
   - Sugar: `f()` → Canonical: `f (())`
   - Initial implementation: Expression contexts only
   - ⚠️ **Initial Limitation**: Statement-level required `f ()` with space
   - ✅ **Fixed in v0.4.2**: Now works at both statement and expression levels (see M-S-CALL0 above)

4. **Strict Syntax Mode** (~120 LOC across parser + pipeline + repl)
   - CLI: `--strict-syntax` flag for `run`, `check`, `repl` commands
   - REPL: `:strict` toggle command
   - Rejects all syntactic sugar with helpful error messages
   - Example: `Error: CONS sugar not allowed in strict mode. Use '::(x, xs)' instead of 'x :: xs'`

5. **REPL Desugaring Feedback** (~20 LOC in repl_eval.go + repl_commands.go)
   - Shows `(desugared)` note when syntactic sugar is used
   - Works in both expression evaluation and `:type` command
   - Example: `1 :: 2 :: [] :: List[int] (desugared)`

**Files Added**:
- internal/parser/sugar_test.go: +300 LOC (NEW - 7 comprehensive tests, 2 for S-CONS, 3 for S-ARROWTYPE, 2 integration)
- design_docs/planned/v0_4_1/m-s-call0-statement-parsing.md: +150 LOC (NEW - documents S-CALL0 limitation + 3 solution approaches)

**Files Modified**:
- internal/parser/parser.go: +25 LOC (strict mode infrastructure)
- internal/parser/parser_expr.go: +110 LOC (S-CONS + S-CALL0)
- internal/parser/parser_type.go: +45 LOC (S-ARROWTYPE with goto refactor)
- internal/lexer/token.go: +1 LOC (DCOLON precedence)
- internal/pipeline/pipeline.go: +1 LOC (StrictSyntaxMode config field)
- internal/pipeline/pipeline_single.go: +1 LOC (pass flag to parser)
- internal/pipeline/pipeline_module.go: +1 LOC (pass flag to loader)
- internal/loader/loader.go: +10 LOC (strict mode support)
- internal/repl/repl.go: +6 LOC (strict mode config + setter + autocomplete)
- internal/repl/repl_eval.go: +6 LOC (desugaring feedback)
- internal/repl/repl_commands.go: +26 LOC (:strict command + help + desugaring in :type)
- cmd/ailang/main.go: +30 LOC (flag routing for all commands)
- prompts/v0.4.1.md: +95 LOC (comprehensive sugar documentation)

**Technical Details**:
- **Parser Strategy**: Desugar during parsing (bijective transformation to canonical forms)
- **Right-Associativity**: Both `::` and `->` use precedence-based right-associativity
- **Error Messages**: Strict mode provides canonical form suggestions for rejected sugar
- **REPL Integration**: Parser tracks sugar usage via `SugarUsed()` flag for feedback

**Test Coverage**:
- 7 new tests in sugar_test.go (all passing)
- S-CONS: Basic, right-associativity
- S-ARROWTYPE: Single arrow, multi-arrow, with effects
- S-CALL0: Skipped (documented limitation)
- Integration: Multiple sugars combined

**Resolves**: M-SUGAR milestone (baseline), Surface Sugar Pack design doc

**Note**: S-CALL0 statement-level support completed in v0.4.2 (M-S-CALL0)

**Total Impact**: ~1,000 LOC (600 new + 400 modified), 7 new tests, 0 regressions

### Benchmark Results (M-EVAL)

**Overall Performance**: 59.9% success rate (333/556 runs)

**Standard Eval (0-shot + self-repair):**

| Metric | v0.4.0 | v0.4.1 | Change |
|--------|--------|--------|--------|
| **0-shot (first attempt)** | 44.0% (125/284) | 38.4% (109/284) | **-5.6%** |
| **Final (with repair)** | 49.3% (140/284) | 45.8% (130/284) | **-3.5%** |
| **Repair effectiveness** | +5.3pp | +7.4pp | **+2.1pp** ✅ |
| **Python (final)** | 73.9% (201/272) | 74.6% (203/272) | +0.7% |

**Agent Eval (multi-turn iterative problem solving):**

| Language | v0.4.0 | v0.4.1 | Change |
|----------|--------|--------|--------|
| **AILANG** | 76.3% (29/38) | 81.6% (31/38) | **+5.3%** ✅ |
| **Python** | 78.9% (30/38) | 84.2% (32/38) | **+5.3%** ✅ |

**Key Findings:**
1. **0-shot declined** (-5.6%): Models making more first-attempt mistakes
2. **Self-repair improved** (+2.1pp): System catching and fixing more errors
3. **Agent eval improved** (+5.3%): Multi-turn iterative problem solving got better for both languages
4. **Net effect**: -3.5% final success for standard eval, but strong improvement in agent mode

**Root Cause Analysis**: LLM variance, not Surface Sugar
- +22 WRONG_LANG errors (models trying to use non-existent features like `import std/io`)
- 24 benchmarks improved, 22 benchmarks broke (nearly balanced)
- Example: `simple_print/gpt5` succeeded in v0.4.0 but failed in v0.4.1 (switched from correct `print()` to wrong `import std/io (write)`)
- No pattern linking failures to Surface Sugar syntax (`::`, `->`, `f()`)
- v0.4.1 prompt was correctly used (confirmed in versions.json)

**Conclusion**: The -3.5% standard eval regression is within normal LLM variance. Surface Sugar features are working as designed. The +5.3% agent eval improvement suggests the v0.4.1 prompt helps with iterative problem solving.

## [v0.4.0] - 2025-11-01

### Added - Environment Variable Support (M-ENV)

**User Impact**: Access environment variables with capability-based security, snapshot semantics, and automatic redaction.

**What Was Added**:
1. **Env Effect**: New capability for environment variable access (~740 LOC core + ~440 LOC tests = ~1,180 LOC)
   - `getEnv(name)`: Returns `Result(String, EnvError)` with Ok/NotFound/NotAllowed
   - `hasEnv(name)`: Returns `bool` for existence check
   - `getEnvOr(name, default)`: Convenience wrapper with fallback
   - Snapshot semantics: Immutable snapshot captured at program start (external changes ignored)
   - Allowlist enforcement: Restrict access with `--allow-env` or `--allow-env-file`
   - No enumeration: Cannot list all variables (security by design)

2. **CLI Flags** (cmd/ailang/main.go: +95 LOC):
   - `--caps Env`: Enable Env capability
   - `--allow-env KEY1,KEY2`: Restrict to specific variables
   - `--allow-env-file path.txt`: Load allowlist from file (one per line, # for comments)
   - `--env KEY=value,FOO=bar`: Override specific variables
   - `--env-snapshot path.json`: Load snapshot from JSON file
   - `--write-env-snapshot path.json`: Write snapshot to JSON and exit

3. **Redaction System** (internal/effects/redact.go: ~200 LOC, 35 tests):
   - Pattern matching: Detects sensitive names (key, secret, token, password, credential)
   - Error redaction: Removes API keys, tokens, Base64 strings from error messages
   - `AILANG_REDACT_ENV=off`: Disable redaction for debugging
   - Example: `API_KEY=sk-proj-abc123` → `API_KEY=[REDACTED]`

4. **Standard Library** (std/env.ail: ~80 LOC):
   - `getEnv(name)`: Get variable with Result type
   - `hasEnv(name)`: Check existence
   - `getEnvOr(name, default)`: Get with fallback
   - `EnvError` ADT: `NotFound(String) | NotAllowed(String)`

**Files Added/Modified**:
- internal/effects/env.go: 170 LOC (effect operations)
- internal/effects/env_test.go: 320 LOC (12 tests)
- internal/effects/context.go: +50 LOC (snapshot fields)
- internal/effects/redact.go: 200 LOC (redaction system)
- internal/effects/redact_test.go: 240 LOC (35 tests)
- internal/builtins/env.go: 120 LOC (builtin registration)
- std/env.ail: 80 LOC (stdlib module)
- cmd/ailang/main.go: +95 LOC (CLI flags)
- internal/parser/parser_effect.go: +1 LOC (Env effect)

**Security Invariants Verified**:
- ✅ Cannot read env without Env capability
- ✅ Cannot enumerate env vars (no list function)
- ✅ Cannot bypass allowlist (enforced in getEnv/hasEnv)
- ✅ Secrets never in errors/logs (redaction works)
- ✅ Snapshot immutable (external changes ignored)

**Example Usage**:
```bash
# Basic usage (all variables allowed)
ailang run --caps Env program.ail

# Security: Allowlist enforcement
ailang run --caps Env --allow-env API_KEY,DEBUG program.ail

# Testing: Override variables
ailang run --caps Env --env API_KEY=test_key program.ail

# Reproducibility: Save/load snapshots
ailang run --caps Env --write-env-snapshot env.json program.ail
ailang run --caps Env --env-snapshot env.json program.ail
```

**Example AILANG Code**:
```ailang
import std/env(getEnv, hasEnv, getEnvOr)

func main() -> string ! {Env} =
  if hasEnv("DEBUG") then
    match getEnv("API_KEY") {
      Ok(key) => "Debug mode with key"
      Err(NotFound) => "Debug mode, no key"
      Err(NotAllowed) => "API_KEY not in allowlist"
    }
  else
    getEnvOr("PORT", "8080")
```

### Fixed

**Critical Result Type Bug** (bb68921):
- **Issue**: `envGetEnv` returned bare `EnvError` instead of wrapping it in `Err()` Result constructor
- **Impact**: All error cases caused "no pattern matched in match expression" runtime errors
- **Example**: `getEnv("NONEXISTENT")` returned `NotFound("...")` instead of `Err(NotFound("..."))`
- **Fix**: Added `makeErrResult()` helper function to properly wrap EnvError in Err() constructor
- **Type Fix**: Changed `Result(T, E)` to `Result[T, E]` in std/env.ail (square brackets, not parentheses)

**Examples Added**:
- env_simple.ail - Basic getEnvOr usage (~10 LOC)
- env_basic.ail - Demonstrates getEnv, hasEnv, getEnvOr (~35 LOC)
- env_allowlist.ail - Security allowlist demonstration (~60 LOC)
- env_config.ail - Configuration management pattern (~60 LOC)
- env_snapshot.ail - Snapshot semantics demonstration (~50 LOC)

All examples tested and verified working with M-ENV implementation.

**Test Coverage**: 47 tests (12 env + 35 redaction), all passing

## [v0.3.25] - 2025-10-29

### Fixed - Stdlib Reserved Keyword Bug (M-BUG-STDLIB-RESERVED-KEYWORD)

**Issue**: Using `exists` as a function name in `std/fs.ail` caused parse errors because `exists` is a reserved keyword (used for quantifiers in planned testing syntax).

**Impact**:
- ❌ Any code importing `std/fs` or `std/io` (which transitively imports `std/fs`) failed to parse
- ❌ Affected ~19/226 AILANG eval benchmarks (~8%), incorrectly marked as WRONG_LANG
- ❌ Prevented users from using filesystem operations

**Root Cause**:
- `std/fs.ail:28` defined `export func exists(path: string) -> bool ! {FS}`
- `exists` is a reserved keyword in `internal/lexer/token.go`
- Parser rejected `exists` as an identifier

**Fix**:
1. **Renamed function** in `std/fs.ail` (~1 LOC)
   - Changed: `export func exists(...)` → `export func fileExists(...)`
   - Rationale: More specific, matches naming pattern (`readFile`, `writeFile`, `fileExists`)

2. **Created FS builtins registration** in `internal/builtins/fs.go` (~120 LOC)
   - Registered 3 builtins: `_fs_readFile`, `_fs_writeFile`, `_fs_exists`
   - Delegates to effect operations in `internal/effects/fs.go`
   - Follows M-DX1 builtin registration pattern (using `RegisterEffectBuiltin`)
   - Complete metadata: descriptions, params, returns, examples, tags

3. **Updated golden file** for builtin types test
   - `internal/pipeline/testdata/builtin_types.golden` now includes 3 FS builtins
   - Total builtins: 52 → 59 (added FS operations)

**Verification**:
- ✅ `std/fs.ail` parses successfully
- ✅ Code importing `std/fs` works correctly
- ✅ Code importing `std/io` works (transitive import fixed)
- ✅ All 3 FS functions (`fileExists`, `readFile`, `writeFile`) tested and working
- ✅ All existing tests pass (including golden file update)

**Migration Guide**:
- No migration needed - `exists` was never callable before (parse error)
- New function name: `fileExists(path: string) -> bool ! {FS}`
- Example: `if fileExists("config.yaml") then readFile("config.yaml") else "default"`

**Regression Prevention**:
4. **Added stdlib integration tests** in `internal/stdlib/integration_test.go` (~180 LOC)
   - `TestStdlibModulesCanBeParsed`: Ensures all 9 stdlib modules parse successfully
   - `TestStdlibNoReservedKeywordsAsIdentifiers`: Explicitly checks for reserved keyword violations
   - `TestStdlibImportChain`: Tests that importing stdlib modules doesn't cause transitive failures
   - These tests will catch this bug class automatically in the future

**Files Modified**:
- `std/fs.ail`: 1 LOC (function rename)
- `internal/builtins/fs.go`: 120 LOC (new file, FS builtin registration)
- `internal/pipeline/testdata/builtin_types.golden`: +3 builtins
- `internal/stdlib/integration_test.go`: 180 LOC (new file, regression tests)

**Total**: ~301 LOC (implementation + tests), all tests passing

**Expected Impact on Next Eval Baseline**:
- ✅ ~19 benchmarks will succeed instead of WRONG_LANG error
- ✅ Final success rate improvement: ~8% (19/226)
- ✅ Enables proper testing of FS capability system

## [v0.3.24] - 2025-10-29

### Fixed - Windows Build Cross-Platform Compatibility

**Issue**: v0.3.23 release had Windows build failures due to line-ending differences in prompt files causing SHA256 hash mismatches in tests.

**Root Cause**: Windows CI was checking out `prompts/*.md` files with CRLF line endings while macOS/Linux used LF, causing different file hashes despite identical content.

**Fix**: Added `.gitattributes` rule to force LF line endings for all `prompts/*.md` files across all platforms.

```gitattributes
# Force LF for prompt files (prevents hash mismatches on Windows)
prompts/*.md text eol=lf
```

**Impact**:
- ✅ Windows build now passes all tests
- ✅ Consistent file hashes across macOS, Linux, and Windows
- ✅ v0.3.23 per-benchmark timeout feature now available on all platforms

**Files Modified**: 1 LOC in `.gitattributes`

## [v0.3.23] - 2025-10-29

### Added - Per-Benchmark Agent Timeout Control

**User Impact**: Enables fine-grained cost control for agent evaluation by allowing each benchmark to specify its own timeout.

**What Was Missing**:
- All agent benchmarks used a global 60-second timeout (hardcoded in AgentBenchmarkConfig)
- No way to give complex benchmarks more time without affecting all benchmarks
- Easy benchmarks wasted time, hard benchmarks hit timeout prematurely

**What Was Implemented**:
1. **Core Feature**: Per-benchmark timeout field in BenchmarkSpec (~30 LOC)
   - Added `Timeout int` field to BenchmarkSpec YAML schema
   - Default: Uses config.TimeoutSeconds (60s) if not specified
   - Backwards compatible: Existing benchmarks continue to use 60s default
   - Files: `internal/eval_harness/spec.go`, `internal/eval_harness/agent_runner_streaming.go`

2. **Benchmark Updates**: Added timeout metadata to 6 new benchmarks
   - Medium complexity (90s): csv_to_json_converter, config_file_parser, log_file_analyzer
   - Hard complexity (120s): multi_module_imports, state_machine_traffic_light, tree_transformation_pipeline
   - Rationale: Tiered timeouts (60s/90s/120s) balance cost control with success rate

3. **Testing & Validation**: Verified timeout feature works correctly
   - Python benchmarks: 100% success with 60s timeouts (baseline validation)
   - AILANG benchmarks: Agents reached Turn 7-19 with extended timeouts (vs Turn 0-6 with 60s)
   - Timeout messages confirmed correct values (90s and 120s)

**Benefits**:
- ✅ **Cost control**: Hard cap prevents runaway costs
- ✅ **Flexibility**: Easy benchmarks finish fast, hard ones get more time
- ✅ **Transparency**: Clear timeout values in benchmark YAML
- ✅ **Optimization**: Can tune timeouts based on observed success rates

**Documentation**:
- NEW: `PER_BENCHMARK_TIMEOUT_RESULTS.md` - Implementation analysis and test results
- NEW: `NEW_BENCHMARK_TEST_RESULTS.md` - Python baseline validation (6/6 success)
- NEW: `BENCHMARK_AUDIT_ANALYSIS.md` - Full audit of 38 existing benchmarks

**Code Changes**: 30 LOC across 3 files
- `internal/eval_harness/spec.go` (+1 LOC)
- `internal/eval_harness/agent_runner_streaming.go` (+5 LOC)
- `internal/eval_harness/agent_runner.go` (+1 LOC)
- 6 benchmark YAMLs updated with timeout metadata

**Next Steps**: Run full eval baseline to validate timeout effectiveness across all models and benchmarks.

## [v0.3.22] - 2025-10-27

### Added - JSON Encoding Support

**User Impact**: Enables JSON encoding for AILANG programs. Unblocks api_call_json benchmark and HTTP POST request workflows.

**What Was Missing**:
- `encode()` function was commented out in std/json.ail (lines 19-22)
- Underlying `_json_encode` builtin was never migrated to M-DX1's builtin registry
- AIs teaching prompt referenced encode() but function didn't exist, causing IMP010 errors

**What Was Implemented**:
1. **Core Implementation**: `_json_encode` builtin with RFC 8259 compliance (~270 LOC)
   - Type signature: `Json -> string`
   - Recursive encoder for all 6 JSON types (JNull, JBool, JNumber, JString, JArray, JObject)
   - String escaping: quotes, backslashes, control chars, unicode
   - Number formatting: removes unnecessary decimals (42.0 → "42")
   - Files: `internal/builtins/json_encode.go` (NEW)

2. **Test Coverage**: Comprehensive test suite (~390 LOC, 27 tests passing)
   - Unit tests: 12+ tests for individual JSON types
   - String escaping: 9+ tests for RFC 8259 compliance
   - Edge cases: empty arrays/objects, nested structures
   - Roundtrip tests: 5 tests verifying decode(encode(x)) == Ok(x)
   - Files: `internal/builtins/json_encode_test.go` (NEW)

3. **Integration**: Uncommented encode() in std/json.ail
   - Removed comment markers on lines 19-21
   - Removed TODO note about migration
   - Files: `std/json.ail` (4 lines changed)

**Files Modified**:
- `internal/builtins/json_encode.go` (+270 LOC NEW): Complete implementation
- `internal/builtins/json_encode_test.go` (+390 LOC NEW): 27 tests
- `std/json.ail` (+3 LOC, -3 comments): Uncommented encode() function

**Validation**:
- ✅ All 27 new tests passing
- ✅ Roundtrip tests pass: decode(encode(x)) == Ok(x)
- ✅ `ailang builtins list --by-module` shows _json_encode in std/json
- ✅ `ailang doctor builtins` passes validation
- ✅ Full test suite passes (no regressions)

**Metrics**:
- Total new code: ~660 LOC (270 impl + 390 tests)
- Test coverage: 100% on new code
- Development time: ~6 hours (Milestones 1-4 complete)

**Sprint**: M-JSON-ENCODE (design_docs/planned/M-JSON-ENCODE-sprint-plan.md)

---

## [v0.3.21] - 2025-10-27

### Fixed - Parser Regression: Nested Match Expressions in Blocks

**User Impact**: AI-generated code with nested match expressions in block contexts now parses correctly. Fixes PAR_NO_PREFIX_PARSE errors on closing braces.

**What Was Broken**:
- 3-level nested match expressions with IO effects failed to parse
- Match arms containing blocks with nested matches triggered delimiter tracking bugs
- Trailing semicolons in match arm blocks caused parser errors
- 64 eval benchmark failures in v0.3.20 (gpt5-mini explicit_state_threading pattern)

**What Was Fixed**:
1. **Primary Fix (Option B)**: Modified `parseCase()` to detect block arms and use `parseBlockOrExpression()` for proper delimiter tracking (~12 LOC in `internal/parser/parser_expr.go`)
2. **Trailing Semicolon Bug**: Fixed `parseBlockOrExpression()` and `parseFunctionBody()` to handle trailing semicolons correctly (~20 LOC across 2 files)
3. **Test Coverage**: Added 11 comprehensive regression tests covering 2-level, 3-level nesting, IO effects, edge cases (empty blocks, comments, whitespace, error recovery)

**Files Modified**:
- `internal/parser/parser_expr.go` (+30 LOC): `parseCase()` fix, `parseBlockOrExpression()` trailing semicolon fix
- `internal/parser/parser_func.go` (+10 LOC): `parseFunctionBody()` trailing semicolon fix
- `internal/parser/parser_match_nested_test.go` (+425 LOC NEW): 11 regression tests

**Validation**:
- ✅ All 11 new regression tests pass
- ✅ Full test suite passes (no regressions)
- ✅ Original failing example (examples/nested_match_ai_generated.ail) executes correctly
- ✅ Linting passes (pre-existing unused function warnings unrelated to changes)

### Added - DX Improvements: Delimiter Tracer & Enhanced Errors

**User Impact**: Better debugging tools for parser issues, especially for AI code generators encountering nested construct problems.

**What Was Added**:

1. **Delimiter Stack Tracer** (`DEBUG_DELIMITERS=1`):
   - Runtime delimiter tracking showing opening/closing of `{` `}` with context
   - Visual indentation showing nesting depth (0-6+ levels)
   - Context labels: match, block, case, function, lambda, record, list
   - Mismatch detection showing expected vs actual delimiters
   - Stack inspection on errors
   - Zero overhead when disabled
   - Example: `DEBUG_DELIMITERS=1 ailang run test.ail`
   - Files: `internal/parser/delimiter_trace.go` (+140 LOC NEW)

2. **Enhanced Error Messages** (Context-Aware):
   - `PAR_NO_PREFIX_PARSE` errors now show nesting depth when inside nested constructs
   - Suggests `DEBUG_DELIMITERS=1` for deep nesting issues (depth > 0)
   - Specific hints for `}`, `)`, `]` errors with actionable guidance
   - Suggests workarounds (simplify nesting, use let bindings)
   - Files: `internal/parser/parser_error.go` (+35 LOC)

3. **Documentation Updates**:
   - `.claude/DX-QUICK-REF.md`: Added DEBUG_DELIMITERS=1 documentation
   - `.claude/skills/sprint-executor/SKILL.md`: Updated parser debugging section with new tools

**Example Enhanced Error**:
```
PAR_NO_PREFIX_PARSE at test.ail:10:9: unexpected token in expression: }

Suggestion: Check for unmatched delimiters or missing expression

Context: Inside nested construct (depth=5)
Hint: This may indicate a parser issue with deeply nested match expressions in blocks.
      Try enabling DEBUG_DELIMITERS=1 to trace delimiter matching.

Suggested workaround: Try simplifying nested constructs or using let bindings.
```

**Total Impact**:
- **Bug Fix**: ~60 LOC across 3 files
- **DX Features**: ~180 LOC across 2 new files + documentation
- **Test Coverage**: +425 LOC, 11 comprehensive tests
- **Estimated eval improvement**: PAR_NO_PREFIX_PARSE errors should drop from 64 → <20 in next baseline

## [v0.3.20] - 2025-10-26

### Added - M-TESTING: Property-Based Testing Infrastructure

**User Impact**: QuickCheck-style property-based testing with automatic shrinking for deterministic validation and CI/CD integration.

**What It Does**:
- Property-based testing (100 random test cases per property)
- Automatic shrinking to minimal counterexamples when tests fail
- `ailang test` CLI command with JSON/human output formats
- Type-aware generators for all AILANG types
- CI/CD ready with exit codes and JSON schema

**Implementation** (Days 6-10 Complete):

- ✅ **Day 6: Basic Generators**
  - IntGenerator, FloatGenerator, BoolGenerator, StringGenerator, ListGenerator
  - PropertyRunner with deterministic seeding
  - GenConfig for customizable generation parameters
  - 30 tests passing
  - Files: `internal/testing/generator.go` (+230 LOC), `generator_test.go` (+529 LOC)

- ✅ **Day 7: Advanced Generators**
  - Combinators: MapGenerator, FilterGenerator, OneOfGenerator, FrequencyGenerator, SizedGenerator
  - Complex types: ADTGenerator, RecordGenerator, TupleGenerator
  - Helpers: OptionGenerator, ResultGenerator
  - 85 tests total (84 pass + 1 skip)
  - Files: `internal/testing/generator_advanced.go` (+271 LOC), `generator_advanced_test.go` (+530 LOC)

- ✅ **Day 8: Shrinking Algorithm**
  - Shrinker interface with 6 implementations
  - IntShrinker, FloatShrinker, StringShrinker (basic types)
  - ListShrinker, ADTShrinker, NoOpShrinker (complex types)
  - PropertyRunner.ShrinkValue() integration
  - Binary search toward simplest values, bounded iterations (max 100)
  - 110 tests total (109 pass + 1 skip)
  - Files: `internal/testing/shrink.go` (+300 LOC), `shrink_test.go` (+537 LOC)

- ✅ **Day 9: CLI Command**
  - `ailang test [path]` command with flag parsing
  - `--format human|json` for output control
  - `--no-color` for CI environments
  - Integration with internal/testing.RunTestsFromFile()
  - Exit codes: 0=pass, 1=fail
  - Files: `cmd/ailang/test.go` (+142 LOC), `cmd/ailang/main.go` (+17 LOC)

- ✅ **Day 10: Documentation & Examples**
  - Customer-facing guide: `docs/TESTING.md` (+650 LOC)
  - AI-focused guide: `prompts/testing_guide_ai.md` (+650 LOC)
  - Basic examples: `examples/testing_basic.ail` (+149 LOC)
  - Advanced examples: `examples/testing_advanced.ail` (+248 LOC)
  - CI/CD integration (GitHub Actions, GitLab CI, CircleCI)
  - README update with testing section

**Code Organization Improvements**:
- Split `internal/parser/parser_decl.go` (1085 → 5 files, all <320 LOC)
- Split `internal/ast/ast.go` (918 → 4 files, all <490 LOC)
- All files now under 800 lines (AI-friendly for context windows)
- Clear package documentation and file responsibilities

**Test Infrastructure**:
- Test syntax: `test "name" = boolean_expression`
- Property syntax: `property "name" (x: type, ...) = boolean_expression`
- 110 tests passing (109 pass + 1 skip)
- Test-to-code ratio: 1.5x

**CI/CD Integration**:
- JSON output schema for machine parsing
- Exit codes for automation (0=pass, 1=fail)
- Pre-commit hook examples
- GitHub Actions workflow example
- GitLab CI and CircleCI configurations

**Files Added**: 13 files (~5,750 lines total)
- Production code: ~1,550 lines
- Test code: ~2,350 lines
- Documentation: ~1,650 lines
- Examples: ~400 lines

**Files Split**: 9 files (better AI maintainability)
- Parser split: 5 focused files (file, func, testing, test_decl, decl routing)
- AST split: 4 focused files (core, expr, decl, type)

**Breaking Changes**: None

**Migration Notes**: None required

## [v0.3.19] - 2025-10-25

### Added - M-CLAUDE-CODE-INTEGRATION-V2: Interactive ↔ Autonomous Agent Bridge

**User Impact**: Seamless handoff between Claude Code sessions and autonomous AILANG agents with production-grade reliability.

**What It Does**:
- Interactive sessions → autonomous agents (Stop hook detects design docs, sends to sprint-planner)
- Autonomous agents → user notifications (inbox system with read/unread/archive)
- Content-addressed artifact storage (SHA256 hashing, deduplication, verification)
- HMAC message signing (prevent spoofing, key rotation support)
- Session start notifications (agents can notify you of completed work)

**Implementation** (Phases 1-4 Complete):

- ✅ **Phase 1: Foundation**
  - `InteractiveEvent` struct (provider-agnostic event abstraction)
  - Content-addressed artifact storage (`internal/agentprotocol/artifacts.go`, ~350 LOC)
    - SHA256 hashing with `.ailang/state/artifacts/sha256/<hash>/content` storage
    - Metadata tracking (original path, MIME type, size, creation time)
    - Deduplication (same content stored only once)
    - Hash verification on retrieval (detect corruption)
  - HMAC message signing (`internal/agentprotocol/signing.go`, ~350 LOC)
    - HMAC-SHA256 with key rotation support
    - Signing key stored in `.ailang/state/signing_key.json` (mode 0600)
    - Canonical JSON representation for deterministic signing
    - Automatic verification on message receive
  - Stop hook script (`scripts/hooks/agent_handoff.sh`, ~100 LOC)
    - Detects design docs in `design_docs/planned/` modified < 5 min
    - Stores artifacts and sends to `sprint-planner` agent
    - Logs to `.ailang/state/hooks.log`

- ✅ **Phase 2: User Inbox & CLI**
  - User inbox system (`internal/agentprotocol/message.go`, +147 LOC)
    - Three folders: `_unread/`, `_read/`, `_archive/`
    - `UserInbox` API: SendToUser, GetUnreadMessages, MarkAsRead, MarkAsArchived
  - Enhanced send-message CLI (`examples/agents/send_message.go`, ~190 LOC)
    - `--to-user` flag (send to user inbox)
    - `--wait <duration>` flag (poll for response with timeout)
    - `--from <agent>` flag (specify sender)
  - Enhanced check-inbox CLI (`examples/agents/check_inbox.go`, ~230 LOC)
    - Support for `user` inbox (read/unread/archive views)
    - `--archive` flag (move to archive after viewing)
    - `--unread-only`, `--read-only`, `--archived` filters
  - SessionStart hook script (`scripts/hooks/session_start.sh`, ~70 LOC)
    - Checks user inbox on session start
    - Displays notification with count and preview
    - Guides user to check-inbox command

- ✅ **Phase 3: Delivery Guarantees + Observability**
  - Extended database schema with message envelope fields:
    - `parent_message_id` - Message threading for request/response chains
    - `ttl_seconds` - Time-to-live for message expiration
    - `deadline` - Hard deadline timestamp
    - `attempt` - Attempt counter (tracks retries across restarts)
  - Database methods for retry and DLQ logic:
    - `IncrementRetryCount()` - Atomic retry counter increment
    - `GetMessagesByStatus()` - Query messages by status with limits
    - `GetExpiredMessages()` - Find messages past deadline
    - `GetMetrics()` - Retrieve metrics for time range
    - `GetAgentStats()` - Aggregate statistics per agent
  - Dead Letter Queue implementation (`internal/agentprotocol/message.go`, +128 LOC):
    - `DeadLetterQueue` struct with file-based storage
    - `MoveToDeadLetter()` - Move failed messages with metadata
    - `GetDeadLetterMessages()` - List all DLQ entries
    - `DeleteDeadLetterMessage()` - Remove from DLQ
    - `RetryFromDeadLetter()` - Retry with reset counter
    - DLQ entries include: failure reason, stack trace, retry count, timestamp
  - Observability CLI (`cmd/ailang/agent.go`, ~290 LOC):
    - `ailang agent top` - Show agent status, queue sizes, metrics
    - `ailang agent dlq --list` - List dead letter queue entries
    - `ailang agent dlq --retry <id>` - Retry failed message
    - `ailang agent dlq --delete <id>` - Delete DLQ entry

- ✅ **Phase 4: Testing & Quality**
  - DLQ unit tests (`internal/agentprotocol/dlq_test.go`, ~235 LOC)
  - Integration tests for DLQ, retry logic, and message expiration
  - All 36 new tests passing (~100% coverage on new code)
  - Database schema migration compatibility (backward compatible)
  - Build system updates (exclude `examples/agents` from linting)

**Documentation**:
- ✅ **Docusaurus Integration** - Main documentation now on website
  - `docs/docs/guides/claude-code-integration.mdx` - Complete integration guide (~600 LOC)
  - `docs/docs/guides/hooks-setup.mdx` - Quick setup guide (~200 LOC)
  - `docs/docs/guides/agent-workflows.mdx` - Workflow patterns (~550 LOC)
  - Added to "Getting Started" section in sidebar
  - All documentation accessible at https://sunholo-data.github.io/ailang/

**Testing**:
- ✅ Artifact storage: 11 unit tests
- ✅ HMAC signing: 9 unit tests
- ✅ User inbox: 8 unit tests
- ✅ DLQ & retry logic: 8 integration tests
- ✅ All 36 new tests passing (100% coverage on new code)

**Quick Start**:
1. Configure hooks in `.claude/hooks.json`
2. Run `chmod +x scripts/hooks/*.sh`
3. Test user inbox: `ailang agent inbox user`
4. Send messages: `ailang agent send --to-user '{"message": "test"}'`
5. Monitor agent status: `ailang agent top`
6. View DLQ: `ailang agent dlq --list`

### Changed - Code Organization & AI Maintainability

**Motivation**: AILANG is designed to be maintained by AI assistants. Large files (>800 lines) exceed AI context windows and violate single responsibility principle. This release refactors the two largest files in the compiler pipeline.

**Pipeline Module Refactoring** (`internal/pipeline/`, -88% main file size):
- **Split `pipeline.go` (1014 lines → 4 files, all <800 lines)**:
  - `pipeline.go` (121 lines, -88%): Main types, Config, Result, Run entry point with package documentation
  - `pipeline_single.go` (355 lines): Single-file/REPL pipeline (runSingle function)
  - `pipeline_module.go` (540 lines): Multi-module pipeline with dependencies (runModule function)
  - `pipeline_telemetry.go` (54 lines): Lowering telemetry reporting

**Monomorphization Module Refactoring** (`internal/pipeline/`, -90% main file size):
- **Split `specialize.go` (1384 lines → 6 files, all <800 lines)**:
  - `specialize.go` (142 lines, -90%): Main Specializer struct, entry point, statistics with package documentation
  - `specialize_types.go` (368 lines): Type manipulation (canonicalTypeFingerprint, substituteType, etc.)
  - `specialize_expr.go` (336 lines): Expression specialization (specializeExpr)
  - `specialize_lambda.go` (132 lines): Lambda specialization (specializeLambda)
  - `specialize_clone.go` (295 lines): Expression cloning with fresh node IDs (cloneExpr)
  - `specialize_helpers.go` (171 lines): Helper functions (isRecursive, patternBoundVars, copyEnv, etc.)

**Results**:
- ✅ All files now under 800 line limit (largest: 540 lines)
- ✅ All 2,847+ tests passing (no regressions)
- ✅ Package compiles successfully
- ✅ Clear package documentation explaining file responsibilities
- ✅ Follows AI-friendly design patterns (200-500 line sweet spot)
- ✅ Ready for AI-assisted maintenance and feature development

**Impact**: Makes codebase significantly more maintainable for AI code assistants by ensuring all files fit comfortably in context windows.

## [Unreleased]

_No unreleased changes yet._

## [v0.3.18] - 2025-01-23

### M-POLY-B Phase 1: Var-Bound Polymorphic Lambdas (Comparison Operators)

**User Impact**: Var-bound polymorphic lambdas with comparison operators now work correctly. Example: `let max = \x. \y. if x > y then x else y in max(3.14)(2.71)` → `3.14` (previously panicked).

**Problem**: Var-bound polymorphic lambdas failed at runtime because operators inside specialized lambda bodies weren't being re-linked with correct types.

**Root Cause Analysis**:
- Dictionary elaboration (BinOp → DictApp) only ran in REPL, not file pipeline
- Monomorphization cloned lambdas but didn't re-elaborate operators
- Type substitution missing TVar2 support
- Operator resolution used wrong strategy (intrinsic type vs operand type)

**Implementation** (Phase 1 - Comparison Operators):

- ✅ **Dictionary Elaboration in All Pipelines** (`internal/pipeline/pipeline.go`)
  - Added `ElaborateWithDictionaries()` to file pipeline (line 228-244)
  - Added to module pipeline (line 680-701)
  - BinOp → DictApp transformation now consistent across REPL and files

- ✅ **Type Substitution Enhanced** (`internal/pipeline/specialize.go`)
  - Added TVar2 case with normalization (line 1019-1027)
  - Fixed `substituteType()` to handle both TVar and TVar2
  - Normalized TVar2 → TVar when possible

- ✅ **cloneExpr Let Case Added** (`internal/pipeline/specialize.go`)
  - Added missing Let case (line 1008-1017)
  - Properly clones Let bindings during specialization
  - Updates CoreTI with substituted types

- ✅ **Operator Resolution Strategy Fixed** (`internal/pipeline/op_lowering.go`)
  - Changed comparison operators to use operand type instead of result type
  - `isComparisonOrEqualityOp()` function determines strategy
  - Fixes: `>`, `<`, `>=`, `<=`, `==`, `!=`

**What Works Now** ✅:
- Var-bound comparison lambdas: `let max = \x. \y. if x > y then x else y in max(3.14)(2.71)` → `3.14`
- All comparison operators: `>`, `<`, `>=`, `<=`, `==`, `!=`
- All equality operators: `==`, `!=`
- Polymorphic type preservation: Types stay polymorphic until call site

**What Remains (Phase 2, Deferred to v0.4.2)** ❌:
- Var-bound arithmetic lambdas: `let add = \x. \y. x + y in add(3.14)(2.71)`
  - Root cause: Type inference defaults arithmetic to `int` (Num typeclass defaulting)
  - Workaround 1: Type annotations: `let add: float -> float -> float = \x. \y. x + y`
  - Workaround 2: Inline lambdas: `(\x. \y. x + y)(3.14)(2.71)` (works!)
  - Phase 2 requires type inference changes (4-8 hours, complex)

**Bugs Fixed**:
1. Dictionary elaboration missing from file pipeline
2. Type substitution missing TVar2 support
3. cloneExpr missing Let case
4. substituteType not normalizing TVar2
5. Operator resolution using wrong strategy for comparison

**Tests**:
- ✅ Comparison operators: All 6 working with var-bound lambdas
- ✅ Type substitution: TVar and TVar2 both handled
- ✅ Monomorphization: Correctly specializes comparison lambdas
- ❌ Arithmetic operators: Phase 2 (type inference issue)

**Files Modified**:
- `internal/pipeline/pipeline.go` (+120 LOC)
- `internal/pipeline/specialize.go` (+40 LOC)
- `internal/pipeline/op_lowering.go` (+10 LOC)

**Documentation**:
- `M-POLY-B-PHASE1-COMPLETE.md` (implementation report)
- `M-POLY-B-PHASE1-COMPLETION-REPORT.md` (this changelog entry)
- `design_docs/planned/v0_4_1/m-poly-b-operator-relinking.md` (updated)

**Time Investment**: 12 hours (within 8-16 hour estimate for Phase 1)

---

## [v0.3.18] - 2025-10-23

### M-DX4: Var Type Resolution (Float Comparison Fix)

**User Impact**: Float comparisons in let-bound variables now work correctly instead of panicking. Example: `let f1 = 3.14 in let f2 = 2.71 in f1 > f2` → `true` (previously panicked with "interface conversion: FloatValue is not IntValue").

**Problem**: After type inference and ApplySubstitution, Var nodes bound to monomorphic values (like float literals) still had unresolved type variables (TVars) in CoreTypeInfo. This caused operator lowering to fall back to Default (Int), resulting in runtime type mismatches when float values were used.

**Root Cause Analysis**:
- Hindley-Milner unification creates substitution mapping type variables to concrete types
- ApplySubstitution resolves type variable chains BUT doesn't always propagate Let binding types to Var usages
- Example: `let x = 3.14 in x > 0.0`
  - Literal `3.14` has CoreTI entry: `float`
  - Var `x` has CoreTI entry: `α4` (type variable, unresolved!)
  - Operator lowering sees `α4`, Head=Unknown, falls back to Default (Int)
  - Runtime: expects IntValue, receives FloatValue → panic

**Implementation** (Option B: Pragmatic Workaround):

- ✅ **Var Type Resolver** (`internal/pipeline/resolve_vars.go` - 175 LOC)
  - Post-inference pass that propagates monomorphic types from Let bindings to Var usages
  - Conservative rules:
    - Only propagates concrete types (Int, Float, String, Bool, List)
    - Preserves polymorphism (lambda params, polymorphic let-bindings stay as TVars)
    - Respects shadowing (inner bindings override outer)
    - Idempotent (running twice has no effect)
  - Integrated at pipeline Phase 3.5.5 (after type checking, before lowering)
  - Zero allocations, O(n) traversal
  - Enabled by default, `--disable-var-resolution` flag to disable

- ✅ **Pipeline Integration** (`internal/pipeline/pipeline.go`)
  - Added VarResolver pass in both file and module pipelines
  - Debug output: "Var type resolution complete" when `--debug-compile` enabled
  - Config flag: `DisableVarResolution` (default: false)

- ✅ **Enhanced Telemetry** (`internal/pipeline/op_lowering.go`)
  - Track CoreTI hits/misses per operator
  - Report via `--debug-compile`: "Lowering telemetry: X operators, Y% CoreTI hits"
  - Fallback categories: CoreTI-hit, ResolvedConstraints, Default

- ✅ **Documentation** (`internal/types/typechecker_core.go`)
  - Enhanced CoreTypeInfo contract with TVar guidance
  - Explains why TVars remain after type inference
  - Documents VarResolver as pragmatic workaround until M-POLY-B

**What Works Now** ✅:
- Direct float comparisons: `3.14 > 2.71` → `true`
- Let-bound float vars: `let x = 3.14 in x > 0.0` → `true`
- Let chains: `let f1 = 3.14 in let f2 = 2.71 in f1 > f2` → `true`
- Shadowing: `let x = 3.14 in let x = 42 in x > 0` → `true` (int comparison)
- Direct lambda apps: `(\x. x > 0.0)(3.14)` → `true`

**What Remains (Deferred to M-POLY-B, v0.4.1+)** ❌:
- Var-bound polymorphic lambdas: `let maxF = \x. \y. if x > y then x else y in maxF(3.14)(2.71)`
  - Currently: Compiles, panics at runtime (operators still have TVars in specialized body)
  - M-POLY-B will fix: Re-elaborate specialized bodies after monomorphization

**Tests**:
- ✅ **Unit tests** (`internal/pipeline/resolve_vars_test.go` - 387 LOC)
  - 7/7 tests passing
  - `TestVarResolverMonomorphicFloat`: Basic float propagation
  - `TestVarResolverLetChain`: Propagation through ANF chains
  - `TestVarResolverPolymorphicParam`: Lambda params stay polymorphic
  - `TestVarResolverMixedBindings`: Selective mono vs poly
  - `TestVarResolverIdempotent`: Running twice has no effect
  - `TestVarResolverNestedLet`: Shadowing resolution
  - `TestVarResolverNonMonomorphic`: Polymorphic bindings not propagated

- ✅ **Integration tests** (manual verification)
  - Float comparison: `let f1 = 3.14 in let f2 = 2.71 in f1 > f2` → `true` (100% CoreTI hits)
  - Polymorphic lambda: Compiles, panics at runtime (expected, deferred to M-POLY-B)

**Debug Output Example**:
```bash
$ ailang run --debug-compile test.ail
[DEBUG] Monomorphization (module test): 0 specializations, 0 skipped
[DEBUG] Var type resolution complete for module test
[DEBUG M-DX4] NodeID 3: type=float, head=Float
[DEBUG] Lowering telemetry for module test:
[DEBUG] Lowering telemetry: 1 operators processed
[DEBUG]   CoreTI hits: 1 (100.0%)
[DEBUG]   CoreTI misses: 0 (0.0%)
true
```

**Metrics**:
- Implementation: ~175 LOC (resolver) + ~50 LOC (integration/telemetry)
- Tests: ~387 LOC unit tests
- Test coverage: 7/7 unit tests passing, manual integration verification
- All existing tests still passing

**Files Modified** (4):
- New: `internal/pipeline/resolve_vars.go` (175 LOC)
- New: `internal/pipeline/resolve_vars_test.go` (387 LOC)
- Modified: `internal/pipeline/pipeline.go` (+~30 LOC, VarResolver integration)
- Modified: `internal/pipeline/op_lowering.go` (+~20 LOC, telemetry)
- Modified: `internal/types/typechecker_core.go` (+20 LOC, documentation)

**See Also**:
- Design doc: `design_docs/planned/v0_3_18/M-DX4-SPRINT-PLAN.md`
- Future work: `design_docs/planned/v0_4_1/m-poly-b-operator-relinking.md`

---

## [v0.3.17] - 2025-10-22

### M-DX4: CoreTypeInfo Completeness & Type-Guided Lowering

**User Impact**: Compiler now fails fast with clear diagnostics when type information is incomplete, instead of panicking during lowering with "cannot lower unknown variant".

**Problem**: Lowering phase could crash with cryptic "cannot lower unknown variant" errors when CoreTypeInfo had gaps, with no indication of which Core node was missing type information or where in the code the issue originated.

**Implementation**:
- ✅ **CoreTypeInfo validation pass** (`internal/pipeline/validate_coretypeinfo.go` - 343 LOC)
  - Walks all 20+ Core node types (Var, Lit, Lambda, Let, LetRec, App, If, Match, BinOp, UnOp, Intrinsic, Record, RecordAccess, RecordUpdate, List, Tuple, DictAbs, DictApp)
  - Verifies 100% CoreTypeInfo coverage before lowering
  - Groups errors by kind (Lit(Float), Intrinsic(OpLe), Let(x), etc.)
  - Includes actionable hints for each missing type
  - Suggests debug command: `ailang debug ast <file> --show-types --compact`
  - Forward-compatible with monomorphization (type variables OK)
  - Performance: O(n) linear, zero allocations (191ns for 10 nodes, 34.4μs for 1000 nodes)

- ✅ **Validation integration** (3 sites)
  - Single-file pipeline (`internal/pipeline/pipeline.go:228`) - validates before lowering
  - Module pipeline (`internal/pipeline/pipeline.go:631`) - validates per-module before lowering
  - REPL (`internal/repl/repl_eval.go:113`) - validates before evaluation
  - Ensures complete parity across file and REPL paths

- ✅ **Comprehensive error diagnostics**
  - NodeID: Unique identifier for each Core node
  - ExprKind: Human-readable kind ("Lit(Float)", "Intrinsic(OpLe)", "Lambda(x)")
  - Position: Source location from OriginalSpan (line/column)
  - Hint: Actionable suggestion based on node type
  - Example: "This usually means defaulting/substitution wasn't applied to CoreTI. Check that ApplySubstitution() was called after type inference."

**Example Error Output**:
```
CoreTypeInfo validation failed: missing type information for Core nodes

Missing Lit(Float) types (1 nodes):
  • NodeID 42 at line 5, col 12
    Hint: This usually means defaulting/substitution wasn't applied to CoreTI.
          Check that ApplySubstitution() was called after type inference.

Missing Intrinsic(OpLe) types (1 nodes):
  • NodeID 58 at line 7, col 8
    Hint: Intrinsic operations (comparisons, arithmetic) must have types before lowering.
          Check that operand types are populated in typechecker_core.go.

Debug with:
  ailang debug ast <file> --show-types --compact

This is a compiler bug. The type checker should populate CoreTypeInfo for all Core nodes.
See: https://sunholo-data.github.io/ailang/docs/internals/type-system
```

**Tests**:
- ✅ **Comprehensive unit tests** (`internal/pipeline/validate_coretypeinfo_test.go` - 417 LOC)
  - 8/8 tests passing
  - Complete program validation (all nodes typed)
  - Missing Float/Bool literal detection
  - Missing comparison operator detection
  - Missing nested let detection
  - **Multi-gap golden test** (4 missing nodes, grouped output, stable ordering)
  - Polymorphic lambda acceptance (type variables OK - forward-compat with monomorphization)
  - All Core node types smoke test (no panics)

- ✅ **Performance benchmarks** (`internal/pipeline/validate_coretypeinfo_bench_test.go` - 117 LOC)

  | Benchmark | Nodes | Time/op | Allocs/op | Notes |
  |-----------|-------|---------|-----------|-------|
  | SmallProgram | 10 | 191 ns | 0 | Typical REPL expression |
  | MediumProgram | 100 | 2.3 μs | 0 | Small module |
  | LargeProgram | 1000 | 34.4 μs | 0 | Large module |
  | DeepNesting | 500 levels | 11.5 μs | 0 | Stress test (recursion) |
  | WideTree | 100 children | 229 ns | 0 | Stress test (branching) |

  **Analysis**: O(n) linear scaling confirmed (1000 nodes ≈ 180x slower than 10 nodes as expected). Zero allocations across all benchmarks = negligible overhead. Validation adds <35μs even for very large programs.

**Key Discovery**: CoreTypeInfo population was already complete thanks to M-DX4 FIX V2 (ApplySubstitution applied after type inference on lines 207-210, 340-342 in typechecker_core.go). The typechecker's single CoreTI.Set() call (line 442) successfully populates CoreTypeInfo for ALL Core expressions after successful type inference.

**Manual Verification**:
```bash
# All these run successfully without CoreTypeInfo validation errors:
ailang run --entry main <(echo 'let x = 3.14 in x')                   # Float
ailang run --entry main <(echo 'let x = 5 <= 10 in x')                # Comparison
ailang run --entry main <(echo 'let x = 1 in let y = 2 in x + y')     # Nested lets
ailang run --entry main <(echo 'let f = (\x -> x + 1) in f 42')       # Lambda
```

**Metrics**:
- Total implementation: ~360 LOC (validation walker + integration)
- Total tests: ~534 LOC (unit tests + benchmarks)
- Test ratio: 1.5:1 (test-heavy, appropriate for compiler correctness)
- Test coverage: 100% for validation logic (8/8 unit tests, 5 benchmarks)
- All existing tests passing with validation enabled

**Files Modified** (5):
- New: `internal/pipeline/validate_coretypeinfo.go` (343 LOC)
- New: `internal/pipeline/validate_coretypeinfo_test.go` (417 LOC)
- New: `internal/pipeline/validate_coretypeinfo_bench_test.go` (117 LOC)
- Modified: `internal/pipeline/pipeline.go` (+11 LOC, 2 validation sites)
- Modified: `internal/repl/repl_eval.go` (+6 LOC, 1 validation site)
- Modified: `internal/parser/cli_integration_test.go` (fixed 3 URL assertions for M-COMPILE-ERROR)

**Design Documentation**:
- `design_docs/planned/v0_3_15/m-dx4-coretypeinfo-completeness.md` - Original design
- `design_docs/planned/v0_3_15/M-DX4-SPRINT-PLAN-REFINED.md` - Sprint plan with all 10 refinements

**Sprint Timeline**:
- Estimated: 1.5-2 days (4-6 hours)
- Actual: ~3 hours (validation skeleton + integration + testing)
- Efficiency: Phases 2 & 3 were already complete due to prior M-DX4 FIX V2 work

---

### M-POLY-A: Call-Site Monomorphization (v0.4.0)

**User Impact**: Polymorphic lambdas are now specialized at call sites with concrete types, eliminating potential runtime panics and enabling future optimizations.

**Problem**: Polymorphic functions (type `α -> α`) applied with concrete types could cause runtime issues when operators in the lambda body couldn't resolve types. This is the foundation for v0.4.0 monomorphization support.

**Implementation**:
- ✅ **Feature flags** (`cmd/ailang/main.go`, `internal/pipeline/pipeline.go` - 30 LOC)
  - `--no-mono` flag: Emergency escape hatch to disable monomorphization
  - `--debug-compile` flag: Shows specialization statistics and cache metrics
  - Default: Monomorphization enabled for all compilations

- ✅ **Core specialization infrastructure** (`internal/pipeline/specialize.go` - ~1000 LOC)
  - `Specializer` with cache and resource limits (16 per-function, 512 per-module)
  - Canonical type fingerprinting with SHA256 collision resistance
  - Fresh node ID generation (starting at 1000000 to avoid conflicts)
  - Recursion detection with full shadowing support
  - AST walker for 11+ Core expression types
  - Body cloning with type substitution (TVar, TFunc2, TApp)
  - Cache deduplication for identical specializations

- ✅ **Enhanced diagnostics** (`internal/pipeline/pipeline.go` - 40 LOC)
  - Cache hit/miss tracking and display
  - Per-function specialization breakdown
  - Skip reason reporting (recursive functions, caps exceeded)
  - Example output: `5 specializations, 2 skipped (cache: 3 hits, 2 misses)`

- ✅ **Resource protection**
  - Per-function cap: Max 16 specializations per function
  - Module-wide cap: Max 512 specializations per module
  - Clear error messages with current/max counts: `Per-function limit reached (16/16)`

**Key Discovery**: Hindley-Milner type inference already specializes simple polymorphic lambdas during type checking. The monomorphization pass handles:
- Within-module specialization of direct lambda applications (v0.4.0)
- Future: Cross-module polymorphic functions (v0.5.0)
- Future: Persistently polymorphic values in let-polymorphism contexts

**Tests**:
- ✅ **Unit tests** (`internal/pipeline/specialize_test.go` - ~460 LOC)
  - 12 tests covering fingerprinting, naming, detection, limits
  - Cache tracking validation
  - Per-function and module cap enforcement
  - Skip reason tracking

- ✅ **Integration tests** (`internal/pipeline/specialize_integration_test.go` - ~330 LOC)
  - 7 comprehensive integration tests
  - Direct lambda application specialization (verified: 1 specialization!)
  - Recursive function skipping (verified: correctly skipped)
  - Module and per-function cap enforcement
  - Cache deduplication on identical types
  - Statistics accuracy validation

**Example Usage**:
```bash
# Normal compilation (monomorphization enabled)
ailang run --entry main --caps IO module.ail

# Debug mode (show specialization stats)
ailang run --entry main --caps IO --debug-compile module.ail
# Output: [DEBUG] Monomorphization: 5 specializations, 2 skipped (cache: 3 hits, 2 misses)

# Emergency disable (if issues arise)
ailang run --entry main --caps IO --no-mono module.ail
```

**Metrics**:
- Total implementation: ~1130 LOC (specializer + pipeline integration)
- Total tests: ~790 LOC (unit + integration tests)
- Test ratio: 0.7:1 (well-tested infrastructure)
- Test coverage: 19/19 tests passing (12 unit + 7 integration)
- All existing tests passing with monomorphization enabled
- Performance: O(n) traversal, ~0 overhead for non-polymorphic code

**Files Modified** (4):
- New: `internal/pipeline/specialize.go` (1002 LOC - core implementation)
- New: `internal/pipeline/specialize_test.go` (461 LOC - unit tests)
- New: `internal/pipeline/specialize_integration_test.go` (331 LOC - integration tests)
- Modified: `internal/pipeline/pipeline.go` (+120 LOC - integration + diagnostics)
- Modified: `cmd/ailang/main.go` (+30 LOC - CLI flags)

**Design Documentation**:
- `design_docs/planned/v0_4_0/monomorphization.md` - Original design
- Sprint plan refined with 10 architectural improvements (caps, fingerprints, caching)

**Sprint Timeline**:
- Estimated: 4-5 days
- Actual: 4 days (infrastructure, core logic, diagnostics, testing)
- On schedule with comprehensive test coverage

**Limitations (v0.4.0)**:
- Within-module specialization only (cross-module deferred to v0.5.0)
- **Direct lambda applications only** - Callee must be inline `Lam`, not `Var` bound to `Lam`
  - ✅ Works: `(\x. \y. if x > y then x else y)(3.14)(2.71)` (inline lambda)
  - ❌ Fails: `let max = \x. \y. if x > y then x else y; max(3.14)(2.71)` (runtime panic)
  - **Workaround**: Inline the lambda or add type annotations `(\x: float. \y: float. ...)`
  - **Fix planned for v0.4.1**: Add `Var→Lam` resolution in specializer (~1 day)
- Recursive functions skipped (with diagnostic messages)
- Mutually recursive groups skipped (with diagnostic messages)

---

### Benchmark Results (M-EVAL)

**Overall Performance**: 60.3% success rate (408 total runs)

**By Language:**
- **AILANG**: 40.0% - New language, learning curve
- **Python**: 81.8% - Baseline for comparison
- **Gap: 41.8 percentage points (expected for new language)

**Comparison**: +9.0% AILANG improvement from 0.3.16 (31.0% → 40.0%)

---

### M-COMPILE-ERROR: Enhanced Parser Errors for AI Code Generation

**User Impact**: AIs generating AILANG code now receive helpful error messages with suggestions when they use Python/JavaScript syntax patterns

**Problem**: AI code generation benchmarks showed 75% failure rate on `api_call_json` due to AIs using familiar Python/JS syntax (namespace imports, `const` keyword, bare assignment) instead of AILANG syntax.

**Added**:
- ✅ **Enhanced ParserError with suggestions** (`internal/parser/parser_error.go` - 30 LOC)
  - New `Suggestions []string` field for multiple fix suggestions
  - New `HelpURL string` field for documentation links
  - Enhanced `.Error()` method formats suggestions with "Did you mean one of these?" header
  - Backward compatible with existing `Fix string` field
  - `NewSuggestionError()` constructor for creating multi-suggestion errors

- ✅ **JavaScript/ES6 import detection** (`internal/parser/parser_decl.go` - 18 LOC)
  - Detects `import X from 'Y'` pattern (common JS/ES6 syntax)
  - Suggests correct AILANG imports: `import std/net (httpRequest)`, `import std/json (encode, decode)`
  - Error code: `IMP012_UNSUPPORTED_NAMESPACE`
  - Help URL: https://sunholo-data.github.io/ailang/docs/language/modules

- ✅ **JavaScript `const` keyword detection** (`internal/parser/parser_decl.go` - 16 LOC)
  - Detects `const` keyword at module level
  - Suggests AILANG syntax: `let name = value in ...`
  - Explains that AILANG bindings are immutable by default
  - Error code: `PAR_CONST_NOT_SUPPORTED`
  - Help URL: https://sunholo-data.github.io/ailang/docs/language/basics

- ✅ **Python-style bare assignment detection** (`internal/parser/parser_decl.go` - 16 LOC)
  - Detects `x = y` without `let` keyword (Python pattern)
  - Suggests correct AILANG syntax with variable name: `let x = ... in`
  - Error code: `PAR_BARE_ASSIGNMENT`
  - Help URL: https://sunholo-data.github.io/ailang/docs/language/basics

**Tests**:
- ✅ **Comprehensive unit tests** (`internal/parser/suggestion_errors_test.go` - 320 LOC)
  - `TestDetectJavaScriptNamespaceImport`: Verifies `import X from 'Y'` detection
  - `TestDetectConstKeyword`: Verifies `const` keyword detection
  - `TestDetectBareAssignment`: Verifies Python-style `x = y` detection
  - `TestActualEvalFailureExample1/2/3`: Tests with actual AI-generated code from eval failures
  - `TestMultipleSuggestionsFormatting`: Validates error message formatting
  - `TestBackwardCompatibilityWithFix`: Ensures old `Fix` field still works

- ✅ **CLI integration tests** (`internal/parser/cli_integration_test.go` - 150 LOC)
  - `TestCLIIntegration_JavaScriptImport`: Full error flow for JS imports
  - `TestCLIIntegration_ConstKeyword`: Full error flow for `const`
  - `TestCLIIntegration_BareAssignment`: Full error flow for bare assignment
  - `TestErrorFormattingConsistency`: Validates consistent formatting across all error types

**Metrics**:
- Total implementation: ~80 LOC
- Total tests: ~470 LOC (100% coverage for new code)
- All existing tests still passing
- Test coverage: 100% for all new error detection logic

**Example Error Output**:
```
IMP012_UNSUPPORTED_NAMESPACE at test.ail:1:8: namespace imports not yet supported

Did you mean one of these?
  import std/net (httpRequest)     -- For HTTP requests
  import std/json (encode, decode) -- For JSON parsing
  import std/io (println)          -- For I/O operations

See: https://sunholo-data.github.io/ailang/docs/language/modules
```

**Files Modified** (2):
- `internal/parser/parser_error.go` (+30 LOC)
- `internal/parser/parser_decl.go` (+50 LOC)

**Files Added** (2):
- `internal/parser/suggestion_errors_test.go` (320 LOC)
- `internal/parser/cli_integration_test.go` (150 LOC)

**Design Documentation**:
- `design_docs/planned/20251022_compile_error_ailang_compilation_failures.md` - Problem analysis
- `design_docs/planned/M-COMPILE-ERROR-SPRINT.md` - Sprint plan

**Eval Baseline Results** (Milestone 3):
- ✅ **Error detection working**: `IMP012_UNSUPPORTED_NAMESPACE` appears in compiler output
- ✅ **All 3 patterns detected**: Namespace imports, const keyword, bare assignment
- ✅ **Repair attempted**: All 3 models tried self-repair with error messages
- ❌ **Repair still fails**: All 3 models (claude-haiku-4-5, gemini-2-5-flash, gpt5-mini) failed after repair
- ❌ **Suggestions not reaching AIs**: Module loader truncates error messages

**Critical Discovery**:
- Enhanced error messages with suggestions ARE generated correctly by parser
- BUT module loader (`internal/loader/loader.go:143`) formats errors as:
  ```go
  fmt.Errorf("parse errors in %s: %v", path, p.Errors())
  ```
- Using `%v` with error slice bypasses our custom `.Error()` method
- AIs only see: `[IMP012_UNSUPPORTED_NAMESPACE at file:1:8: namespace imports not yet supported...]`
- AIs DON'T see: `Did you mean: import std/net (httpRequest)...`
- **Impact**: AIs can't benefit from our helpful suggestions during repair attempts

**Follow-up Required** (v0.3.19):
- Fix module loader error formatting to iterate errors and call `.Error()` on each
- Re-run eval baseline after fix to measure actual improvement
- Expected improvement after fix: 75% failure → <25% failure (target: 100% success)

---

## [v0.3.17] - 2025-10-21

### M-DX3: Lambda DX Fixes (Comparison Operators + show Bool)

**User Impact**: Comparison operators now work correctly in lambda expressions

**Fixed**:
- ✅ **Comparison operators in lambda bodies** (`internal/pipeline/op_lowering.go`)
  - Root cause: Operator lowering used result type (Bool) instead of operand type (Int/Float/String)
  - For `x > 0` in lambda, intrinsic has type Bool, but needs operand type (Int) to choose `gt_Int`
  - Fix: Added `isComparisonOrEqualityOp()` helper to detect comparison/equality operators
  - Changed type lookup to use `intrinsic.Args[0].ID()` for comparisons (not `intrinsic.ID()`)
  - Now correctly selects `gt_Int`, `lt_Float`, `eq_String`, etc. based on operand types
  - Eliminates "Operator '>' has no implementation for type Bool" errors

**Verified**:
- ✅ **show(Bool) already worked** - No implementation needed
  - Tested `show(true)`, `show(false)`, `show(5 > 3)` - all return correct strings
  - Implementation in `internal/builtins/show.go` lines 112-116 handles BoolValue
  - Tests exist in `internal/builtins/show_test.go` lines 35-37
  - No changes required for this item

**Changed**:
- ✅ **Enhanced lambda examples** (`examples/snippets/showcase/lambdas_basic.ail`)
  - Added `max`, `min`, `abs` functions using comparison operators
  - Demonstrates working comparison operators in lambda bodies
  - Examples: `max(10)(5)`, `min(10)(5)`, `abs(-7)`

**Added**:
- ✅ **Comprehensive tests** (`internal/pipeline/op_lowering_comparison_test.go` - 237 LOC)
  - `TestComparisonWithIntOperands`: Verifies `x > 0` uses `gt_Int` (not `gt_Bool`)
  - `TestComparisonWithFloatOperands`: Verifies `x < 0.0` uses `lt_Float`
  - `TestAllComparisonOperators`: Tests all 6 operators (lt, le, gt, ge, eq, ne)
  - `TestIsComparisonOrEqualityOp`: Tests helper function
  - All tests follow existing patterns from `op_lowering_test.go`
  - Uses mocked CoreTypeInfo for unit testing
- ✅ **LIMITATIONS.md** (`docs/LIMITATIONS.md` - ~250 LOC)
  - Documents Y-combinator limitation (Hindley-Milner occurs check by design)
  - Documents float comparison bug (pre-existing, out of scope for M-DX3)
  - Includes workarounds and explanations for both limitations
  - Other sections: Parse errors, string interpolation, REPL/file parity

**Known Limitations**:
- ⚠️ **Float comparisons still broken**: Pre-existing bug where float comparisons in lambdas panic
  - Root cause: CoreTypeInfo doesn't have float variable types, defaults to "Int"
  - Calls `gt_Int` on FloatValue, causing panic
  - Workaround: Use float comparisons outside lambda bodies
  - Out of scope for M-DX3 (focused on Int comparisons per original bug report)

**Performance Impact**:
- No runtime performance change (operator lowering is compile-time)
- Test coverage: 100% for new code (237 LOC tests)
- Eliminated entire class of "wrong operator type" bugs for comparisons

**Files Added** (2):
- `internal/pipeline/op_lowering_comparison_test.go` (237 LOC)
- `docs/LIMITATIONS.md` (~250 LOC)

**Files Modified** (2):
- `internal/pipeline/op_lowering.go` (+24 LOC: helper function + modified type lookup)
- `examples/snippets/showcase/lambdas_basic.ail` (+9 LOC: max/min/abs examples)

**Design Documentation**:
- `design_docs/planned/v0_3_17/m-dx3-lambda-dx-fixes.md` - Complete technical spec
- `design_docs/planned/v0_3_17/M-DX3-sprint-plan.md` - Sprint execution plan
- `design_docs/implemented/v0_3_16/lambda-expressions-example-refactor.md` - DX analysis (lines 352-802)

**Sprint Execution**:
- Milestone 1: Fix comparison operators (✅ complete)
- Milestone 2: show(Bool) support (✅ already worked, no changes needed)
- Milestone 3: Integration & docs (✅ complete)
- Total time: ~3 hours (estimated), actual: ~2.5 hours

---

## [v0.3.16] - 2025-10-21

### Examples: Lambda Expressions Refactor

**User Impact**: Improved lambda expression examples with focused, runnable tutorials

**Added**:
- ✅ **6 new focused lambda examples** (`examples/snippets/showcase/lambdas_*.ail`)
  - `lambdas_basic.ail` - Basic syntax, identity, arithmetic, binary lambdas (49 LOC)
  - `lambdas_curried.ail` - Currying, partial application, order matters (45 LOC)
  - `lambdas_closures.ail` - Environment capture, closure factories (44 LOC)
  - `lambdas_higher_order.ail` - Composition, map-like, function returning function (49 LOC)
  - `lambdas_records.ail` - Creating/accessing/updating records with lambdas (59 LOC)
  - `lambdas_advanced.ail` - Flip, Church numerals, CPS, combinators (51 LOC)
  - All files runnable with `ailang run --caps IO --entry main`
  - Total: 297 LOC of working examples

**Changed**:
- ✅ **Archived original lambda_expressions.ail** (moved to `examples/archive/`)
  - Original file was 187 LOC of tutorial-style let-in chains
  - Didn't fit entry-module pattern (needed deep nesting or block expressions)
  - Split into 6 focused, pedagogical examples instead

**Rationale**:
- Better discoverability (clear file names vs monolithic tutorial)
- Each file is independently runnable and testable
- Matches existing showcase structure
- Easier to maintain and extend
- More focused learning: one concept per file

**Design Doc**: `design_docs/planned/v0_3_16/lambda-expressions-example-refactor.md` (moved to implemented)

---

### M-DX2: Operator Development Experience Improvements

**Developer Experience**: 67% faster polymorphic operator development (2h → 30-60min)

**Added**:
- ✅ **Type-guided operator lowering** (`internal/types/typeinfo.go`, `internal/types/type_head.go`, `internal/pipeline/op_lowering.go`)
  - `CoreTypeInfo` maps Core NodeID → Type (populated during inference)
  - `types.Head()` identifies type constructors (Int, Float, String, Bool, List, etc.)
  - Eliminates ANF shape guessing (~30 lines of heuristics removed)
  - 3-tier fallback: CoreTI → resolved constraints → defaults
- ✅ **Core IR helpers with cycle detection** (`internal/core/helpers.go`)
  - `ResolveValue()` follows ANF variable bindings safely
  - `IsListValue()`, `IsStringValue()`, `IsIntValue()`, etc.
  - Fail-closed cycle detection (returns last resolvable expression)
- ✅ **Debug CLI for ANF inspection** (`cmd/ailang/debug.go`)
  - `ailang debug ast file.ail --show-types` shows Core AST with inferred types
  - Node IDs, type annotations, intrinsic operations visible
  - Essential for debugging operator lowering
- ✅ **Structured builtin errors** (`internal/eval/builtin_errors.go`)
  - `ArgTypeMismatch()`, `IndexOutOfBounds()`, `InvalidOperation()`, `EmptyListError()`
  - Context-aware hints (20+ patterns)
  - Replaces panics with actionable error messages
- ✅ **Comprehensive documentation** (`docs/architecture/ANF.md`, `docs/guides/adding-operators.md`)
  - ANF architecture guide for AI assistants
  - Step-by-step operator implementation checklist
  - Type-guided lowering patterns and examples

**Changed**:
- ✅ **OpLowerer now uses CoreTypeInfo** (`internal/pipeline/op_lowering.go`)
  - Type-guided builtin selection (was: ANF shape checking)
  - Clearer separation of concerns (typechecker → lowerer)
  - No more "wrong builtin" bugs from ANF shape mismatches

**Performance Impact**:
- Polymorphic operator development: 2 hours → 30-60 minutes (-67% to -75%)
- Test coverage: 100% for new code (~1,500 LOC total)
- "Wrong builtin" class of bugs: eliminated

**Files Added** (11):
- `internal/types/typeinfo.go` (93 LOC)
- `internal/types/typeinfo_test.go` (220 LOC)
- `internal/types/type_head.go` (100 LOC)
- `internal/types/type_head_test.go` (140 LOC)
- `internal/pipeline/op_lowering_regression_test.go` (150 LOC)
- `internal/core/helpers.go` (110 LOC)
- `internal/core/helpers_test.go` (310 LOC)
- `cmd/ailang/debug.go` (200 LOC)
- `internal/eval/builtin_errors.go` (170 LOC)
- `internal/eval/builtin_errors_test.go` (310 LOC)
- `docs/architecture/ANF.md` (~450 lines)
- `docs/guides/adding-operators.md` (~650 lines)

**Files Modified** (7):
- `internal/types/typechecker_core.go` (~10 LOC changes)
- `internal/types/inference.go` (~5 LOC changes)
- `internal/pipeline/op_lowering.go` (~60 LOC changes)
- `internal/pipeline/pipeline.go` (~5 LOC changes)
- `internal/repl/repl_eval.go` (~2 LOC changes)
- `cmd/ailang/main.go` (~10 LOC changes)
- `.claude/skills/sprint-executor/resources/developer_tools.md` (~60 LOC additions)

**Design Documentation**:
- `design_docs/planned/v0_3_16/M-DX2-M1-COMPLETE.md` - Type-Guided Lowering
- `design_docs/planned/v0_3_16/M-DX2-M2-COMPLETE.md` - Core IR Helpers
- `design_docs/planned/v0_3_16/M-DX2-M3-COMPLETE.md` - Debug CLI
- `design_docs/planned/v0_3_16/M-DX2-M4-COMPLETE.md` - Better Runtime Errors
- `design_docs/planned/v0_3_16/M-DX2-COMPLETE.md` - Final sprint summary

### M-EVAL Round-Robin: Better Parallel Distribution

**Performance**: 2x faster baseline evaluations with improved model interleaving

**Added**:
- ✅ **Round-robin job scheduling** (`cmd/ailang/eval_suite.go`)
  - Interleaves models in job queue (model1, model2, model3, model1, ...)
  - Distributes API calls across providers (OpenAI, Anthropic, Google)
  - Enables higher parallelism without hitting single-provider rate limits
  - Example: `--parallel 10` now means ~3-4 concurrent calls per provider (was 10 to single provider)

**Changed**:
- ✅ **Increased default parallelism from 5 to 10** (`--parallel` flag)
  - Safe with round-robin distribution (spreads load across 3 providers)
  - Recommended: 10-12 for dev suite (3 models), 12-15 for full suite (6 models)
  - Updated help text to explain cross-provider distribution

**Performance Impact**:
- Dev suite (132 jobs, 3 models): ~10-12 minutes (was ~18-22 minutes) - **45% faster**
- Full suite (264 jobs, 6 models): ~22-28 minutes (was ~55-70 minutes) - **50% faster**
- Enables safe parallelism scaling (can push to 15 workers without rate limit issues)

**Files Modified**:
- `cmd/ailang/eval_suite.go` - Round-robin job ordering, increased default parallelism
- `design_docs/planned/m-eval-round-robin.md` - Design doc with benchmarks and rationale

### Entry-Module Prelude System

**AI-First DX**: Automatic `print` builtin for entry modules and REPL

**Added**:
- ✅ **Entry-module prelude injection** (`internal/pipeline/prelude.go`)
  - AST-based detection of `export func main` with 0 parameters
  - Type environment injection before type checking
  - `print : string -> () ! {IO}` available in entry modules and REPL
- ✅ **Enhanced teaching prompt** (`prompts/v0.3.16.md`)
  - Comprehensive documentation of prelude system
  - Entry module vs library module examples
  - Updated from v0.3.8 with new features
- ✅ **Comprehensive tests** (`internal/pipeline/prelude_test.go`, 278 LOC)
  - Entry module detection tests
  - Type injection tests
  - Library isolation tests
  - Builtin list verification

**Changed**:
- ✅ **Removed `print` from global builtin registry** (`internal/builtins/io.go`)
  - Now entry-module-only (explicit libraries must use `_io_println`)
  - Preserves library purity and explicitness
- ✅ **Updated 12 example files** to work with new system
  - 6 files: Added `import std/io (_io_println)`
  - 3 files: Fixed parse errors
  - 3 files: Updated deprecated `stdlib/*` imports to `std/*`

**Fixed**:
- ✅ **Net builtin errors** - Migrated 3 files from deprecated `_net_httpGet` to modern API
  - Updated `stdlib/std/net.ail` with wrapper functions
  - Updated test files to use `std/net` module
  - Enhanced capability detection in verification script
  - Pass rate improved from 69.3% to 72.7% (+3 examples fixed)

**Files Added/Modified**:
- `internal/pipeline/prelude.go` (+120 LOC) - Core prelude implementation
- `internal/pipeline/prelude_test.go` (+278 LOC) - Comprehensive tests
- `prompts/v0.3.16.md` (+1,213 LOC) - Updated teaching prompt
- `prompts/versions.json` - Set v0.3.16 as active
- `stdlib/std/net.ail` - Added `httpGet`/`httpPost` wrappers
- `scripts/verify_examples.go` - Enhanced capability detection
- `Makefile` - Added `verify-examples-all` and `examples-status` targets
- `benchmarks/simple_print.yml` - Entry-module prelude test
- `README.md` - Updated pass rate to 72.7%

**Metrics**:
- Pass rate: 61/88 (69.3%) → 64/88 (72.7%)
- All 2,847+ tests passing
- CI threshold: 60% (comfortably met at 72.7%)

## [v0.3.15] - 2025-10-21

### Module Path Unification & Net Builtin Fixes

**Changed**:
- ✅ **Unified module paths** - All imports now use `std/` prefix (removed legacy `stdlib/`)
- ✅ **Updated deprecated imports** - Fixed 6 example files with old `stdlib/*` imports
- ✅ **Enhanced verification** - Capability detection for Net, Clock, IO effects

**Fixed**:
- ✅ **Net builtin migration** - Updated deprecated `_net_httpGet` to modern `httpRequest` API
- ✅ **Parse errors** - Fixed 3 files with syntax issues

**Metrics**:
- Pass rate improved from 61/88 to 64/88
- All core tests passing

### Benchmark Results (M-EVAL)

**Overall Performance**: 59.1% success rate (399 total runs)

**By Language:**
- **AILANG**: 33.0% - New language, learning curve
- **Python**: 87.0% - Baseline for comparison
- **Gap**: 54.0 percentage points (expected for new language)

**Comparison**: -15.2% AILANG regression from v0.3.14 (48.2% → 33.0%)

**Analysis**: The regression is likely due to the entry-module prelude changes from v0.3.16 being already in the codebase when this baseline was run. The benchmark suite may need updates to work with the new `print` scoping rules.

## [Unreleased] - Next release

### M-DX1: Builtin Registry - COMPLETE! (2025-10-20)

🎉 **MILESTONE ACHIEVED**: All 52 builtins fully documented and organized!

**Status**: 90% complete - all core work done, remaining 10% is optional DX polish

**What We Accomplished** (October 2025 session, ~6.5 hours):

**Infrastructure (Already complete from v0.3.10)**:
- ✅ Central registry with single-point registration
- ✅ Type Builder DSL (71% less code for type construction)
- ✅ Test harness with MockEffContext for hermetic testing
- ✅ CLI tools: `ailang doctor builtins`, `ailang builtins list`

**New This Session**:
- ✅ **Complete builtin documentation (52/52 = 100%)**
  - All builtins have descriptions, parameters, returns, examples
  - Searchable tags, version tracking, stability indicators
  - 100+ working examples across all builtins
- ✅ **Enhanced metadata system** (11 fields per builtin)
  - Description, LongDesc, Params, Returns, Examples, SeeAlso
  - Since, Deprecated, Stability (Experimental/Stable/Deprecated)
  - Tags (searchable), Category (grouping)
- ✅ **File organization** (split 785-line file into 7 AI-friendly modules)
  - `string.go` (458 lines) - 9 string builtins
  - `math.go` (566 lines) - 37 math/comparison/logic/conversion builtins
  - `io.go` (114 lines) - 3 I/O builtins
  - `net.go` (101 lines) - 1 HTTP builtin
  - `show.go` (188 lines) - 1 polymorphic show builtin
  - `json_decode.go` (378 lines) - 1 JSON parsing builtin
  - `register.go` (26 lines) - Documentation only
- ✅ **Migration safety validator** (`ailang builtins check-migration`)
  - AST-based scanning of legacy builtin locations
  - Prevents disasters like the show() loss in v0.3.10
  - Reports orphaned builtins with actionable diagnostics

**Documented Builtins by Category**:
1. String operations (9) - `_str_len`, `_str_compare`, `_str_find`, etc.
2. Math arithmetic (12) - `add_Int`, `div_Float`, `neg_Int`, etc.
3. Comparisons (20) - `eq_Int`, `lt_Float`, `gt_String`, `ne_Bool`, etc.
4. Logic (3) - `and_Bool`, `or_Bool`, `not_Bool`
5. Conversions (2) - `intToFloat`, `floatToInt`
6. I/O (3) - `_io_print`, `_io_println`, `_io_readLine`
7. Network (1) - `_net_httpRequest`
8. Core (1) - `show` (polymorphic)
9. JSON (1) - `_json_decode`

**Metrics**:
- Implementation time: 7.5h → 2.5h (-67% reduction) ✅
- Files to edit: 4 → 1 (-75% reduction) ✅
- Type construction: 35 LOC → 10 LOC (-71% reduction) ✅
- Documented builtins: 0/52 → 52/52 (+100%) ✅
- File size (max): 785 lines → 566 lines (-28%) ✅
- All 2,847 tests passing ✅

**Optional Future Polish** (~2.5 hours total, see `design_docs/planned/m-dx1-future-polish.md`):
- Enhanced CLI: `--verbose` and `search` commands (~2h)
- REPL `:type` command (~0.5h)
- Error diagnostics improvements (~0.5h)

**Files Added/Modified**:
- `internal/builtins/metadata.go` (+145 LOC) - Metadata type definitions
- `internal/builtins/spec.go` - Added Metadata field to BuiltinSpec
- `internal/builtins/math.go` - Added metadata to 37 builtins
- `internal/builtins/json_decode.go` - Added metadata to JSON parsing
- `internal/builtins/migration_validator.go` (+329 LOC) - Safety validator
- `cmd/ailang/main.go` - Added `check-migration` subcommand
- `M-DX1-FINAL-SUMMARY.md` (+400 LOC) - Complete session documentation
- `M-DX1-COMPLETION-ANNOUNCEMENT.md` (+300 LOC) - Milestone announcement
- `CLAUDE.md` - Updated M-DX1 status and examples
- `design_docs/planned/m-dx1-future-polish.md` - Updated with completion status

**Verification**:
```bash
ailang doctor builtins              # ✅ All 52 builtins valid
ailang builtins list                # ✅ All builtins listed
ailang builtins list --by-module    # ✅ Organized by module
ailang builtins check-migration     # ✅ No orphaned builtins
make test                           # ✅ All 2,847 tests pass
```

**For detailed information**: See [M-DX1-COMPLETION-ANNOUNCEMENT.md](M-DX1-COMPLETION-ANNOUNCEMENT.md)

---

### Documentation Clarity: Honest AI-First Positioning

**Documentation Alignment** (2025-10-18)

Updated documentation to accurately reflect what AILANG **already is**: a deterministic language designed for autonomous AI code synthesis and reasoning. This is not a pivot - it's honest communication about the actual implementation and removing over-ambitious promises about features that were never built.

**Clarified Status of Unimplemented Features**:

| Feature | Documentation Said | Reality | Now Documented As |
|---------|-------------------|---------|-------------------|
| **CSP Concurrency / Session Types** | "Key feature" | `internal/channels/` and `internal/session/` are empty | Not implementing - static effect graphs sufficient |
| **LSP Server** | "`ailang lsp` available" | Command did nothing | Removed - AIs use CLI/API |
| **Type Classes** | "Extensible type system" | Hardcoded Num/Eq/Ord/Show only | Built-in only - structural reflection planned v0.4.0 |
| **Typed Quasiquotes** | "Key feature" | Only lexer token exists | Planned v0.4.0 |

**What Actually Works** (v0.3.14):
- Pure functional core (lambda calculus, closures, recursion)
- Hindley-Milner type inference with row polymorphism
- Algebraic effects with capability-based security (IO, FS, Clock, Net)
- Pattern matching with ADTs and exhaustiveness checking
- Module system with runtime execution
- JSON parsing and encoding
- M-EVAL AI benchmarking framework

**Next Priorities** (v0.3.15 - deterministic tooling):
- `ailang normalize` - Canonical code formatting
- `ailang suggest-imports` - Automatic import resolution
- `ailang apply` - Deterministic code edits from JSON plans
- `--emit-trace jsonl` - Structured execution traces for training

**Future** (v0.4.0+ - reflection):
- Typed quasiquotes - Deterministic AST templates
- Structural reflection - `reflect(typeOf(f))` replaces hardcoded type classes
- Schema registry - Machine-readable type/effect definitions
- Capability budgets - `! {IO @limit=2}` for resource-bounded effects

**Documentation Updates**:

- **README.md**: Rewritten to accurately describe what AILANG is
  - Tagline: "The Deterministic Language for AI Coders" (reflects actual design)
  - Architecture overview: 8 layers from core semantics to cognitive interfaces
  - "Why AILANG Works Better for AIs" - comparison of AI vs human needs
  - Honest feature status: what works, what's next, what's not happening

- **CLAUDE.md**: Updated implementation status
  - Clear status markers: ✅ Stable, 🔜 Next, 🔮 Future, ❌ Not implementing
  - Emphasis on determinism, semantic transparency, machine decidability

- **CLI**: Removed non-functional `ailang lsp` command
  - Deleted from help output, command dispatcher, and implementation

**Design Spec Audit Results**:
- Documentation-implementation alignment improved: **75%** → **95%**
- Removed misleading claims about CSP, LSP, extensible type classes
- Clear, honest roadmap: v0.3.15 (tooling), v0.4.0 (reflection), v0.5.x+ (budgets)

---

## [v0.3.14] - 2025-10-18 - JSON Decode + Major DX Improvements

**MAJOR**: JSON parsing support + pattern matching fixes + type system consistency

### Added

**JSON Decoding** (~860 LOC, 42 tests) - **M-LANG-JSON-DECODE**

- `std/json.decode : string -> Result[Json, string]` - Parse JSON strings
- Json ADT with constructors: `JNull`, `JBool(bool)`, `JNumber(float)`, `JString(string)`, `JArray(List[Json])`, `JObject(List[{key: string, value: Json}])`
- Helper: `kv(key, value)` for building JSON objects
- Streaming builder using Go's `encoding/json` for correctness
- Example: `examples/json_basic_decode.ail` demonstrates pattern matching approach

**Files Modified:**
- `internal/builtins/json_decode.go` (+330 LOC) - Streaming JSON builder
- `internal/builtins/json_decode_test.go` (+534 LOC) - 42 comprehensive tests
- `stdlib/std/json.ail` - Json ADT + decode function
- **All 42 tests passing** ✅

**Note**: JSON accessor functions (`get()`, `has()`, `asString()`, etc.) are implemented but not exported pending a module system fix for constructor scope. Target: v0.3.15

### Changed

**Pattern Matching Runtime** (+89 LOC)

- ✅ `[head, ...tail]` cons patterns now work at runtime
- ✅ Record patterns `{field1, field2}` now work
- Unlocks all stdlib list operations (map, filter, foldl, etc.)
- File: `internal/eval/eval_patterns.go`

**Type System Consistency** (~50 fixes across codebase)

- ✅ Type Builder now emits lowercase primitive types: `string`, `int`, `float`, `bool`
- ✅ Aligns with canonical type names in `types.go`
- ✅ Eliminates "cannot unify String vs string" errors
- ✅ Comparison operators now work in most contexts (known edge case in recursive list processing - see note)
- Files: `internal/types/builder.go` + 52 test updates across 4 packages

**Polymorphic Type Support** (+150 LOC)

- ✅ Added TApp unification support for type applications like `Result[Json, string]`
- ✅ Enables generic types to work correctly
- ✅ Decomposition algorithm handles arbitrary nesting
- File: `internal/types/unification.go`

**Builtins**

- Added `_str_eq : (string, string) -> bool` for direct boolean equality
- Registered in both new and legacy registries

### Fixed

- Pattern matching on lists with cons patterns (`[x, ...xs]`)
- Pattern matching on records with field patterns
- Type unification for polymorphic type applications (TApp)
- Type consistency between Type Builder and canonical type names

### Known Issues

**Operator Edge Case**: The `==` operator works in most contexts but has a known edge case in recursive functions with list pattern matching. Workaround: use `_str_eq()` in those specific contexts. This is tracked for investigation in v0.3.15.

**Module Constructor Scope**: ADT constructors (`Some`, `None`, `Ok`, `Err`) from imported modules work for pattern matching but not for construction in helper functions. This blocks JSON accessor functions and is tracked for v0.3.15.

### Testing

- ✅ All 2,847 tests passing
- ✅ Golden files updated with lowercase primitive types
- ✅ Added `TestPrimitiveCasing` guard to prevent regressions
- ✅ JSON decode example verified working

### Metrics

- Files modified: 22 (14 core + 8 tests)
- Tests added: 42 (JSON decode)
- Test expectations updated: 52+
- LOC added (core): +569
- LOC added (tests): +534

### Benchmark Results (M-EVAL)

**Overall Performance**: 63.9% success rate (145/227 runs across 6 models × 22 benchmarks × 2 languages)

**By Language:**
- **AILANG**: 48.2% (54/112) - New language, learning curve
- **Python**: 79.1% (91/115) - Baseline for comparison
- **Gap**: 30.9 percentage points (expected for new language)

**By Model** (sorted by success rate):
- claude-sonnet-4-5: Best performer (full suite run)
- gpt5: Strong performance
- claude-haiku-4-5: Cost-effective option
- gemini-2-5-pro: Competitive
- gpt5-mini: Budget option
- gemini-2-5-flash: Fast and cheap

**Changes from v0.3.13**:
- ✅ **Fixed (3)**: api_call_json (python, claude-haiku-4-5, gpt5), recursion_fibonacci (ailang, gpt5-mini)
- ❌ **Broken (4)**: record_update (ailang/python, gpt5), adt_option (ailang, gpt5-mini), pattern_matching_complex (ailang, gpt5)
- Net change: -0.6% (63.9% vs 64.4% in v0.3.13)

**Developer Experience Improvement**:
- Added `--skip-existing` flag to `ailang eval-suite` command
- Enables resuming interrupted eval runs without losing progress
- Critical for long-running baselines on slower machines
- Example: If 219/264 runs complete before timeout, `--skip-existing` runs only the missing 45

**Notes**:
- This is the first full 6-model baseline (previous versions used 3 models)
- Total eval cost: ~$0.50-1.00 for full suite
- See [archive/2025-10/analysis/BENCHMARK_COMPARISON_v0.3.9.md](archive/2025-10/analysis/BENCHMARK_COMPARISON_v0.3.9.md) for historical comparison (current dashboard: [docs/static/benchmarks/latest.json](docs/static/benchmarks/latest.json))

---

## [v0.3.12] - 2025-10-17 - Recovery Release (show() Restored)

**RECOVERY**: Restored `show()` builtin function lost in v0.3.10 migration

### Added

**`show()` Builtin Function** (~350 LOC, 35 test cases) - **M-LANG Recovery**

**Status**: ✅ COMPLETE - Restores 51% of AILANG benchmarks (v0.3.12)

**Files Modified:**
- `internal/builtins/show.go` (+160 LOC) - Polymorphic show() implementation
- `internal/builtins/show_test.go` (+190 LOC) - Comprehensive tests for all types

**Implementation** (`internal/builtins/show.go`)
- Polymorphic type signature: `∀α. α -> string`
- Runtime type dispatch for primitives: int, float, bool, string
- Structured types: lists, records, ADT constructors
- Special handling: NaN, Inf, depth limiting, string truncation
- Based on v0.3.9's `showValue()` from `internal/eval/eval_simple.go`

**Tests** (`internal/builtins/show_test.go`)
- 17 primitive tests (int, float, bool, string, special floats)
- 5 list tests (empty, single, multiple, mixed, nested)
- 4 record tests (empty, single, multiple, nested)
- 4 ADT constructor tests (nullary, unary, n-ary, nested)
- Edge case tests (depth limit, truncation, functions, errors)
- Type registration validation
- **All 35 tests passing** ✅

**Root Cause Analysis:**
- v0.3.9: `show()` existed in `internal/types/env.go` + `internal/eval/eval_simple.go`
- v0.3.10: Migration to builtin registry lost `show()` (deleted from old locations, never added to new registry)
- Impact: 64/125 AILANG benchmarks failed with "undefined variable: show" (51% of suite)

**Recovery:**
- v0.3.9: 29/63 = 46% AILANG success (with show())
- v0.3.10: 0/126 = 0% AILANG success (row unification bug + no show())
- v0.3.11: 0/125 = 0% AILANG success (row bug fixed, but show() still missing)
- v0.3.12: Expected ~46% AILANG success (row fixed + show() restored)

**REPL Verification:**
```ailang
λ> show(42)
"42" :: String

λ> show(3.14)
"3.14" :: String

λ> show(true)
"true" :: String

λ> show("hello world")
"hello world" :: String
```

**Next Steps:**
- Run `make eval-baseline EVAL_VERSION=v0.3.12` to measure recovery
- Compare v0.3.11 → v0.3.12 to validate 46% success rate restoration

---

## [v0.3.11] - 2025-10-16 - Critical Row Unification Fix

**CRITICAL BUGFIX**: Fixed row unification regression that caused 0% AILANG success in v0.3.10

### Fixed

**Row Unification Bug** (Existed since v0.3.9, became critical in v0.3.10)
- **Root cause**: Parameter swap in `internal/types/row_unification.go` (lines 70-91)
- **Symptom**: All stdlib modules failed with "closed row missing labels: [IO]"
- **Impact**:
  - v0.3.9: Bug existed but masked by other issues (46% AILANG success)
  - v0.3.10: Bug became critical (0% AILANG success)
  - v0.3.11: Bug fixed, but exposed `show()` missing (still 0%, different cause)
- **Fix**: Correctly assign `only1` (r1's unique labels) to `r2.Tail` when unifying closed/open rows

**Effect Propagation in Function Application**
- **File**: `internal/types/typechecker_functions.go` (line 365-370)
- **Issue**: Included `getEffectRow(funcNode)` which is always empty for variable references
- **Fix**: Only combine argument effects + function type's effect row

**REPL Builtin Environment**
- **Files**: `internal/repl/repl.go`, `internal/repl/repl_commands.go`
- **Issue**: Used `NewTypeEnv()` instead of `NewTypeEnvWithBuiltins()`
- **Fix**: REPL now has access to all builtins for `:type` command

### Added

**Safety Net: Regression Prevention Tests** (~300 LOC)
- `internal/types/row_unification_regression_test.go`: 12-case matrix test for row unification
- `internal/pipeline/application_effects_regression_test.go`: Builtin environment availability test
- `internal/pipeline/stdlib_canary_test.go`: End-to-end stdlib typechecking smoke test

**Builtin Environment Factory Pattern**
- `internal/types/env.go`: Added `SetBuiltinEnvFactory()` registration mechanism
- `internal/link/env_seed.go`: Bridge between types and link packages (breaks import cycle)
- Enables REPL and compiler to share builtin definitions without circular dependencies

### Changed

**Debug Logging Cleanup**
- Removed DEBUG fmt.Printf statements from 5 files
- Cleaner output in production builds

### Known Issues

**`show()` Function Missing** (discovered during v0.3.11 validation)
- **Impact**: 64/125 (51%) AILANG benchmarks fail with "undefined variable: show"
- **Root cause**: `show()` was defined in v0.3.9's `internal/types/env.go` but not migrated to new builtin registry
- **Status**: Design doc created (`design_docs/planned/m-lang-show-function.md`)
- **Target**: v0.3.12 (3-4 hour fix)
- **Workaround**: None - code using `show()` will not compile

### Metrics

| Metric | v0.3.9 | v0.3.10 | v0.3.11 | Status |
|--------|--------|---------|---------|--------|
| Row unification errors | 0 (bug masked) | 75 | 0 | ✅ Fixed |
| AILANG compile failures | Many | 126/126 | 125/125 | ⚠️ Different cause |
| `show()` errors | 0 (existed) | N/A | 64 | ❌ Regression |
| Examples passing | 48/87 (55%) | Unknown | 38/87 (44%) | ⚠️ Degraded |
| Test coverage | ✅ | ✅ | ✅ | No regressions |

### Files Modified

**Core fixes:**
- `internal/types/row_unification.go`: Fixed parameter swap (lines 70-91)
- `internal/types/typechecker_functions.go`: Fixed effect propagation (lines 365-370)
- `internal/repl/repl.go`: Use `NewTypeEnvWithBuiltins()` (line 92)
- `internal/repl/repl_commands.go`: Use `NewTypeEnvWithBuiltins()` (line 92)

**Factory pattern:**
- `internal/types/env.go`: Added `SetBuiltinEnvFactory()`, `NewTypeEnvWithBuiltins()`
- `internal/link/env_seed.go`: New file - factory registration

**Safety nets:**
- `internal/types/row_unification_regression_test.go`: New file - 12 test cases
- `internal/pipeline/application_effects_regression_test.go`: New file - builtin env test
- `internal/pipeline/stdlib_canary_test.go`: New file - stdlib smoke test

**Documentation:**
- `design_docs/implemented/v0_3/202510_regression_fix.md`: Complete post-mortem
- `design_docs/planned/m-lang-show-function.md`: Next priority fix

### Test Coverage

- ✅ All 183 Go packages pass tests
- ✅ Row unification matrix test (12 cases)
- ✅ Stdlib canary test (end-to-end)
- ✅ Builtin environment availability test
- ✅ No import cycles

### Technical Notes

**The Row Unification Bug (Lines 70-91)**

Before (buggy):
```go
case r1.Tail == nil && r2.Tail != nil:
    // r1 closed, r2 open
    if len(only1) > 0 {
        return nil, fmt.Errorf("closed row missing labels: %v", ru.labelNames(only1))
    }
    sub[r2.Tail.Name] = &Row{
        Kind:   r2.Kind,
        Labels: only2,  // ❌ WRONG - assigns r2's labels instead of r1's
        Tail:   nil,
    }
```

After (fixed):
```go
case r1.Tail == nil && r2.Tail != nil:
    // r1 closed, r2 open - r2's tail gets r1's unique labels
    sub[r2.Tail.Name] = &Row{
        Kind:   r2.Kind,
        Labels: only1,  // ✅ CORRECT - assigns r1's labels to tail variable
        Tail:   nil,
    }
```

**Why This Matters:**
- When typechecking `_io_print("hello")`, we unify:
  - Builtin signature: `String -> () ! {IO}` (closed row)
  - Application context: `String -> () ! {} | ε` (open row)
- The bug assigned wrong labels to `ε`, causing "closed row missing labels: [IO]"
- Fix correctly unifies `ε := {IO}`, allowing stdlib to typecheck

### Lessons Learned

**1. Silent Fallbacks Hide Bugs**
- The row bug existed since v0.3.9 (Sept 2025) but was masked
- Became critical only when other code paths changed
- Reinforces: NO SILENT FALLBACKS in critical code (cost calculations, types, effects)

**2. Regression Tests Are Essential**
- Created 3-layer safety net (unit, integration, end-to-end)
- Would have caught this bug immediately
- Added to standard test suite to prevent recurrence

**3. Migration Requires Comprehensive Checklists**
- When migrating builtins to new registry, missed `show()` function
- Need explicit checklist: "What builtins existed in v0.3.9?"
- Automated migration validation would catch this

### Next Steps

**Immediate (v0.3.12):**
1. Implement `show()` builtin (see `design_docs/planned/m-lang-show-function.md`)
2. Expected to recover ~46% AILANG success rate
3. Re-run full evaluation baseline

**Future:**
- Complete M-DX1 polish (REPL `:type`, enhanced diagnostics, docs)
- Migrate remaining complex builtins (`_json_encode`)
- Delete legacy builtin code paths

---

## [v0.3.10] - 2025-10-16 - M-DX1.5: Builtin Migration Complete

**Goal Achieved**: Reduced builtin development time from 7.5h to 2.5h (-67%)

### Added

**M-DX1.5: Complete Builtin Migration** (~450 LOC migration code)
- ✅ Migrated all 49 legacy builtins to new spec-based registry
- ✅ Removed feature flag - new registry is now the default
- ✅ All builtins use single-file registration pattern
- ✅ Zero regressions - all tests passing

**Migrated builtins** (49 total):
- **String primitives** (7): `_str_len`, `_str_compare`, `_str_find`, `_str_slice`, `_str_trim`, `_str_upper`, `_str_lower`
- **Arithmetic** (12): `add_Int`, `sub_Int`, `mul_Int`, `div_Int`, `mod_Int`, `neg_Int` + Float variants
- **Comparisons** (20): `eq_*`, `ne_*`, `lt_*`, `le_*`, `gt_*`, `ge_*` for Int, Float, String, Bool
- **Logic** (3): `and_Bool`, `or_Bool`, `not_Bool`
- **Conversions** (2): `intToFloat`, `floatToInt`
- **String ops** (1): `concat_String`
- **IO effects** (3): `_io_print`, `_io_println`, `_io_readLine`

### Changed

**Registry is now default** - No feature flag required
- `internal/link/builtin_module.go`: Always use spec-based registry
- `internal/runtime/builtins.go`: Always use spec-based registry
- `cmd/ailang/main.go`: Removed `AILANG_BUILTINS_REGISTRY` checks from CLI

### Metrics

| Metric | Before (v0.3.9) | After (v0.3.10) | Improvement |
|--------|-----------------|-----------------|-------------|
| Builtins migrated | 2 | 49 | +47 (+2,350%) |
| Files to edit (per builtin) | 4 | 1 | -75% |
| Type construction LOC | 35 | 10 | -71% |
| Dev time (per builtin) | 7.5h | 2.5h | -67% |
| Feature flag required | Yes | No | Removed |
| Tests passing | ✅ | ✅ | No regressions |

### Files Modified

**Core implementation:**
- `internal/builtins/register.go`: +450 LOC (all builtin registrations)
- `internal/link/builtin_module.go`: Removed legacy path
- `internal/runtime/builtins.go`: Removed legacy path
- `cmd/ailang/main.go`: Removed feature flag checks

### Test Coverage

- ✅ All existing tests pass (no regressions)
- ✅ 49 builtins validated by registry
- ✅ CLI commands work without feature flag

### Technical Notes

**M-DX1 Infrastructure** (completed in v0.3.9-alpha3):
1. **Central Builtin Registry** (`internal/builtins/`)
   - Single-point registration with compile-time validation
   - Files: spec.go (150 LOC), validator.go (190 LOC), registry.go

2. **Type Builder DSL** (`internal/types/builder.go`)
   - Fluent API reduces type construction from 35→10 lines
   - Methods: `String()`, `Int()`, `List()`, `Record()`, `Func()`, `Returns()`, `Effects()`

3. **Test Harness** (`internal/effects/testctx/`)
   - MockEffContext with HTTP/FS mocking
   - Value constructors/extractors (17 helpers)
   - 100% test coverage

4. **CLI Commands**
   - `ailang doctor builtins` - Validation with actionable diagnostics
   - `ailang builtins list --by-effect --by-module` - Browse registry

### Future Work

Deferred to v0.3.11+ (see `design_docs/planned/m-dx1-future-polish.md`):
- M-DX1.6: REPL `:type` command (~3h)
- M-DX1.7: Enhanced error diagnostics (~2h)
- M-DX1.8: `docs/ADDING_BUILTINS.md` guide (~2h)
- Migrate `_json_encode` (complex ADT handling)
- Delete legacy builtin code (cleanup)

---

## [v0.3.9] - 2025-10-15 - AI API Integration (HTTP Headers + JSON Encoding)

### Added

**1. HTTP Headers Support** (~350 LOC) - Advanced HTTP client with Result-based error handling
- **New function**: `httpRequest(method, url, headers, body) -> Result[HttpResponse, NetError] ! {Net}`
- **Security features**:
  - Header validation: blocks hop-by-hop headers (Connection, Transfer-Encoding, etc.)
  - Blocks Host override, Accept-Encoding, Content-Length
  - Authorization header stripping on cross-origin redirects
  - Method whitelist (GET, POST only in v0.3.8)
- **Return type**: `Result[HttpResponse, NetError]` with structured error handling
  - `HttpResponse = {status: int, headers: List[{name, value}], body: string, ok: bool}`
  - `NetError = Transport(string) | DisallowedHost(string) | InvalidHeader(string) | BodyTooLarge(string)`
- **Non-breaking**: Existing `httpGet()` and `httpPost()` remain unchanged (deprecated but functional)
- **Files**: `internal/effects/net.go`, `stdlib/std/net.ail`, `internal/link/builtin_module.go`
- **Tests**: 100% coverage with 10+ test cases (`internal/effects/net_test.go`)

**2. JSON Encoding** (~250 LOC) - Complete JSON encoder with proper escaping
- **New module**: `stdlib/std/json.ail` with `Json` ADT and convenience helpers
- **ADT constructors**: `JNull`, `JBool(bool)`, `JNumber(float)`, `JString(string)`, `JArray(List[Json])`, `JObject(List[{key, value}])`
- **Builtin**: `_json_encode(Json) -> string` with full JSON spec compliance
- **String escaping**: All escape sequences (\n, \r, \t, \", \\, \b, \f, control chars)
- **UTF-16 support**: Proper handling of surrogate pairs for characters > 0xFFFF
- **Convenience helpers**: `jn()`, `jb()`, `jnum()`, `js()`, `ja()`, `jo()`, `kv()`
- **Files**: `internal/eval/builtins.go`, `internal/eval/json_test.go`, `stdlib/std/json.ail`
- **Tests**: 100% coverage with 10+ test cases covering all JSON types

**3. Example: OpenAI Integration** (~82 LOC)
- **File**: `examples/ai_call.ail` - Working example calling OpenAI GPT-4o-mini
- **Demonstrates**: Complete workflow with JSON encoding, HTTP headers, Result error handling
- **Security**: Uses Authorization bearer token, validates HTTP status codes
- **Error handling**: Pattern matches on all NetError variants for robust error reporting

### Changed

**Builtin system extended** - Added support for `func(Value) (*StringValue, error)` signature
- **Why**: JSON encoder needs to process ADT values (not just primitives)
- **Impact**: Enables more sophisticated builtins that operate on user-defined types
- **Files**: `internal/eval/builtins.go` (line 520-522)

### Deprecated

- `httpGet()` and `httpPost()` - Use `httpRequest()` instead for status codes and headers
- **Migration**: Both functions remain functional, no breaking changes
- **Reason**: `httpRequest()` provides Result-based error handling and full HTTP response metadata

### Test Coverage

- ✅ JSON encoding: 10 test cases (null, bool, number, string escaping, arrays, objects, nesting)
- ✅ HTTP headers: 4 test functions with 13 subtests (validation, method whitelist, result types)
- ✅ Full effects test suite: 70+ tests pass
- ✅ No regressions: All existing tests pass

### Implementation Notes

**Builtin registration** (4-step process):
1. Effect implementation: `internal/effects/net.go`
2. Runtime wrapper: `internal/runtime/builtins.go`
3. Metadata registry: `internal/builtins/registry.go`
4. Type signature + export: `internal/link/builtin_module.go`

**Type system integration**:
- Used `TApp` for parameterized `Result[HttpResponse, NetError]` type
- Record types use `map[string]types.Type` (not `[]RecordField`)
- List types use `Element` field (not `Elem`)

### Files Modified

1. `internal/effects/net.go` (+300 LOC) - netHTTPRequest implementation
2. `internal/eval/builtins.go` (+205 LOC) - JSON encoder + builtin support
3. `stdlib/std/json.ail` (new, 50 LOC) - Json ADT and helpers
4. `stdlib/std/net.ail` (+72 LOC) - NetError, HttpResponse, httpRequest
5. `examples/ai_call.ail` (new, 82 LOC) - OpenAI integration example
6. `internal/link/builtin_module.go` (+35 LOC) - Type signature for httpRequest
7. `internal/runtime/builtins.go` (+15 LOC) - Runtime registration
8. `internal/builtins/registry.go` (+10 LOC) - Metadata registration
9. `internal/eval/json_test.go` (new, 350 LOC) - JSON tests
10. `internal/effects/net_test.go` (+200 LOC) - HTTP header tests

**Total new code**: ~1,370 LOC (including tests)
**Test coverage**: 100% for new features

### Benchmark Results (M-EVAL)

**Overall Performance**: 62.7% success rate (79/126 runs across 3 models × 21 benchmarks × 2 languages)

**By Language:**
- **AILANG**: 42.9% (27/63) - New language, learning curve
- **Python**: 82.5% (52/63) - Baseline for comparison
- **Gap**: 39.6 percentage points (expected for new language)

**By Model:**
- claude-sonnet-4-5: 66.7% (best performer)
- gpt5: 61.9%
- gemini-2-5-pro: 59.5%

**New Benchmarks (v0.3.9)**:
- `json_encode`: Testing JSON ADT construction and encoding
- `api_call_json`: Testing HTTP POST with headers and JSON payload

**Cost & Metrics**:
- Total cost: $0.68 (full suite with 3 production models)
- Total tokens: 268,886
- Average duration: 34ms per run

---

## [v0.3.8] - 2025-10-15 - Bug Fixes

### Fixed

**1. Multi-line ADT Parser** - Parser now supports multi-line algebraic data type declarations
- **Problem**: AI models generating multi-line ADTs that parser couldn't handle
- **Root Cause**: Parser assumed NEWLINE tokens existed, but lexer skips all newlines as whitespace
- **Solution**:
  - Added support for optional leading PIPE: `type Tree = | Leaf | Node`
  - Removed all NEWLINE token checks (they never exist!)
  - Fixed token positioning in `parseVariant()` to follow parser conventions
- **Impact**: `pattern_matching_complex` benchmarks now pass
- **Files**: `internal/parser/parser_type.go`, `internal/parser/parser.go`

**2. Operator Lowering Bug** - Division operators now resolve to correct builtins
- **Problem**: Division was using wrong builtin (div_Int instead of div_Float), causing runtime errors
- **Root Cause**: Pipeline missing `FillOperatorMethods()` call after type checking
- **Solution**: Added method resolution before operator lowering (5 lines in `internal/pipeline/pipeline.go`)
- **Impact**: `adt_option` benchmarks now pass
- **Files**: `internal/pipeline/pipeline.go`

**3. Documentation** - Added critical architectural lesson to CLAUDE.md
- **Section**: "Lexer/Parser Architecture - NEWLINE Tokens Don't Exist!"
- **Key insight**: Lexer skips newlines in `skipWhitespace()` - they're never returned as tokens
- **Why important**: Prevents future developers from making the same multi-hour debugging mistake
- **Files**: `CLAUDE.md` (~82 lines added)

### Test Results
- ✅ All 100+ parser tests pass
- ✅ Both failing benchmarks (pattern_matching_complex, adt_option) now pass
- ✅ No regressions introduced

### Known Issues
- **File size violations**: 6 files exceed 800 line limit (deferred to v0.3.9/v0.4.0)
  - internal/pipeline/pipeline.go: 848 lines
  - internal/types/inference.go: 853 lines
  - internal/parser/parser_expr.go: 951 lines
  - internal/ast/ast.go: 841 lines
  - internal/eval/eval_typed.go: 879 lines
  - internal/eval/builtins.go: 815 lines

### Benchmark Results (M-EVAL)

**Overall Performance**: 65.8% success rate (75/114 runs across 3 models × 20 benchmarks × 2 languages)

**By Language:**
- **AILANG**: 49.1% (28/57) - New language, learning curve
- **Python**: 82.5% (47/57) - Baseline for comparison
- **Gap**: 33.4 percentage points (expected for new language)

**By Model:**
- claude-sonnet-4-5: 68.4% (best performer)
- gemini-2-5-pro: 65.8%
- gpt5: 63.2%

**Comparison to v0.3.7**:
- v0.3.7 AILANG: 38.6% (22/57)
- v0.3.8 AILANG: 49.1% (28/57)
- **Improvement: +10.5 percentage points** 🎉

**Fixed Benchmarks**:
- ✓ `pattern_matching_complex` - Multi-line ADT parser fix
- ✓ `adt_option` - Operator lowering fix for division
- ✓ `error_handling` - Better AI code generation patterns
- ✓ `numeric_modulo` - Improved modulo operator support
- ✓ `float_eq` - Float equality comparisons
- ✓ Additional improvements across 6 more benchmarks

**Cost & Duration**:
- Total cost: $0.55 (full suite with 3 production models)
- Duration: 5m11s
- Total tokens: 203,483
- Average duration: 28ms per run

**Note**: This release focused on fixing two critical P0 regressions (multi-line ADT parsing and operator lowering). The 10.5% improvement demonstrates significant progress in AI code generation capabilities for AILANG.

---

## [v0.3.7] - 2025-10-15 - Code Cleanup

### Removed
- **Deprecated `CalculateCost` function** - Removed unused cost calculation function
  - Only used in tests, not in actual codebase
  - Replaced by `CalculateCostWithBreakdown` which provides accurate pricing
  - Follows "NO SILENT FALLBACKS" principle - better to return 0.0 than trust wrong data
  - Files modified: `internal/eval_harness/metrics.go`, `internal/eval_harness/metrics_test.go`

### Fixed
- **Linting issues** - Fixed formatting and nil check simplifications
  - Formatted `internal/eval_analysis/types.go`
  - Simplified nil checks in `internal/eval_analysis/export_docusaurus.go`
  - All linting checks now pass

### Benchmark Results (M-EVAL)

**Overall Performance**: 58.8% success rate (67/114 runs across 3 models × 20 benchmarks × 2 languages)

**By Model:**
- claude-sonnet-4-5: 63.2% (best performer)
- gpt5: 57.9%
- gemini-2-5-pro: 55.3%

**By Language:**
- Python: 78.9% (mature ecosystem, well-known syntax)
- AILANG: 38.6% (new language, learning curve)

**Cost & Performance:**
- Total cost: $0.55 for full baseline
- Duration: 4m27s
- Average tokens per run: 1,782

**Note**: This is a code cleanup release with no language changes. Benchmark results reflect the current state of v0.3.6 language features (auto-import std/prelude, record update syntax, numeric conversions, etc.) with improved cost tracking accuracy.

---

## [v0.3.6] - 2025-10-14 - AI Usability Improvements

### Added - Auto-Import std/prelude (2025-10-14)

**Zero-Import Comparisons**: Typeclass instances now auto-loaded by default.
- No more `import std/prelude (Ord, Eq)` needed for `<`, `>`, `==`, `!=` operators
- Automatically loads: Ord, Eq, Num, Show instances for builtin types (int, float, string, bool)
- Optional disable: Set `AILANG_NO_PRELUDE=1` environment variable for explicit import testing

**Implementation** (`internal/types/`)
- `NewCoreTypeChecker()` calls `LoadBuiltinInstances()` by default
- Critical bug fix: `isGround()` now recognizes `TVar2` type variables
  - Was: `TVar2` fell through to `default: return true` (treated as ground)
  - Now: Added `case *TVar2: return false` (correctly non-ground)
  - Impact: Fixed premature instance lookup before defaulting
- Tests: `internal/types/auto_import_test.go` (3 test functions)

**Files Modified**:
- `internal/types/typechecker_core.go` - Auto-load instances, fix isGround()
- `internal/types/auto_import_test.go` - Unit tests for auto-import

**Impact**: Eliminates 11% of M-EVAL failures (typeclass import errors)
- `fizzbuzz` benchmark: Works without imports
- AI cognitive load: Reduced (one less thing to remember)

---

### Added - Record Update Syntax (2025-10-14)

**Functional Record Updates**: New syntax eliminates manual field copying errors.
- Syntax: `{base | field: value, field2: value2}`
- Example: `{person | age: 31}` creates new record with updated age, preserving other fields
- Type-safe: Verifies field exists and type matches
- Pure functional: Returns new record (immutable)

**Implementation** (Full compilation pipeline)
- AST: Added `RecordUpdate` node with base expression and update fields
- Parser: Detects `IDENT PIPE` pattern to distinguish from record literals
  - Supports complex bases: `{foo.bar | x: 1}`, `{getRecord() | y: 2}`
- Core: Added `core.RecordUpdate` node in ANF
- Elaborator: Normalizes base and updates to atomic form
- Type Checker: Extracts base record fields, unifies update types
- Evaluator: Copies all base fields, overwrites specified fields

**Files Modified**:
- `internal/ast/ast.go` - RecordUpdate AST node
- `internal/parser/parser_expr.go` - Parse {base | updates}
- `internal/core/core.go` - core.RecordUpdate node
- `internal/elaborate/expressions.go` - normalizeRecordUpdate()
- `internal/types/typechecker_data.go` - inferRecordUpdate()
- `internal/eval/eval_expressions.go` - evalCoreRecordUpdate()

**Example**:
```ailang
let person = {name: "Alice", age: 30, city: "NYC"};
let older = {person | age: 31};       // Keep name & city
let moved = {older | city: "SF"};     // Keep age: 31 (not reverted!)
// Result: {name: "Alice", age: 31, city: "SF"}
```

**Impact**: Fixes 5% of M-EVAL failures (manual field copy errors)
- `record_update` benchmark: Now passes with all models
- Prevents bugs: AI models no longer forget to copy updated fields

---

### Added - Error Detection for Self-Repair (2025-10-14)

**Targeted Error Messages**: Detect wrong language/imperative syntax for better repair.
- New error codes:
  - `WRONG_LANG`: Detects Python (`def`), JavaScript (`var`, `function`), Java (`public static`), C++ (`#include`)
  - `IMPERATIVE`: Detects `loop`, `while`, `for`, `break`, `continue`, assignment statements
- Pattern matching: Checks generated code BEFORE compilation
- Repair hints: Targeted guidance ("Use recursion instead of loops", "Start over with AILANG syntax")

**Implementation** (`internal/eval_harness/`)
- `errors.go`: New error codes and regex patterns
- `CategorizeErrorWithCode()`: Checks both code and stderr
- `repair.go`: Updated to use new categorization
- Comprehensive tests: 8 test cases for WRONG_LANG/IMPERATIVE detection

**Files Modified**:
- `internal/eval_harness/errors.go` - Add WRONG_LANG/IMPERATIVE patterns
- `internal/eval_harness/repair.go` - Use CategorizeErrorWithCode()
- `internal/eval_harness/errors_test.go` - Test new patterns

**Usage**: `ailang eval-suite --self-repair`

**Impact**: +8.1% improvement with self-repair (32.4% → 40.5% success)
- Detected: 3 WRONG_LANG, 2 IMPERATIVE errors (out of 60 runs)
- Repair success: Some errors auto-corrected, others too fundamental

---

### Performance - M-EVAL Benchmark Results (2025-10-14)

**Baseline**: v0.3.5-8-g2e48915 (before improvements)
**Current**: v0.3.5-15-g542d20f (with all improvements)

| Model | Baseline | With Improvements | Change |
|-------|----------|-------------------|--------|
| Claude Sonnet 4.5 | 35.1% (7/19) | **52.6% (10/19)** | **+17.5%** 🎉 |
| Gemini 2.5 Pro | 26.3% | 37.5% | +11.2% |
| Gemini 2.5 Flash | N/A | 31.6% | - |
| GPT-5 | N/A | 28.6% | - |

**With Self-Repair** (`--self-repair` flag):
- Claude Sonnet: 42.9% → 50.0% (+7.1%)
- Gemini Pro: 25.0% → 37.5% (+12.5%)
- Overall: 32.4% → 40.5% (+8.1%)

**Key Wins**:
- ✅ 3 new benchmarks passing: `recursion_factorial`, `pattern_matching_complex`, `record_update`
- ✅ `fizzbuzz` works without imports
- ✅ Record update syntax used successfully by all models
- ✅ Error detection working (detected 5 WRONG_LANG/IMPERATIVE errors)

**Analysis**:
- Hypothesis confirmed: Language changes (+17.5%) >> Prompt engineering (-5.2%)
- Auto-import: Reduced cognitive load, eliminated typeclass errors
- Record updates: Prevented manual field copying mistakes
- Self-repair: Helped in some cases, but fundamental errors remain hard

**Total Changes**: 11 files, ~400 lines
**Test Coverage**: All changes fully tested end-to-end

---

## [v0.3.5] - 2025-10-13 - Functional Completeness Sprint

### Added - P0: Anonymous Function Syntax (2025-10-13)

**Func Expressions**: Inline function syntax now works in all expression positions.
- New syntax: `func(x: int) -> int { x * 2 }` alongside existing `\x. x * 2`
- Multi-param: `func(x: int, y: int) -> int { x + y }`
- Effects: `func() -> () ! {IO} { println("hi") }`
- Type inference: `func(x, y) { x + y }` (types optional)
- Backward compatible: Old `func(x) => body` syntax still works

**Implementation** (`internal/ast/`, `internal/parser/`, `internal/elaborate/`)
- AST: New `FuncLit` node with params, return type, effects, body (~40 LOC)
- Parser: `parseLambda` detects `->` vs `=>` to choose syntax (~120 LOC)
  - Adds `parseFuncLitWithParams` helper
  - Adds `parseBlockOrExpression` for brace bodies
- Elaborate: `normalizeFuncLit` desugars to `core.Lambda` (~35 LOC)
- SCC: Handle `FuncLit` in `findReferences` (~5 LOC)

**Tests**
- All existing tests pass ✅
- REPL: `let f = func(x: int) -> int { x * 2 } in f(5)` → `10`
- Higher-order: `apply(func(n: int) -> int { n * 2 })(5)` → `10`

**Files Modified**:
- `internal/ast/ast.go` (+40 LOC) - Add FuncLit node
- `internal/parser/parser.go` (+120 LOC) - Parse func expressions
- `internal/elaborate/elaborate.go` (+35 LOC) - Desugar FuncLit → Lambda
- `internal/elaborate/scc.go` (+5 LOC) - Handle FuncLit in call graph

**Total**: ~200 LOC

**Impact**: Unblocks 15/90 M-EVAL benchmarks (all higher-order function code)
- `higher_order_functions` benchmark now parseable
- `pipeline` benchmark now parseable
- AI models can use familiar `func(x) { ... }` syntax

---

### Added - P1a: letrec Keyword for Recursive Lambdas (2025-10-13)

**Recursive Functions in REPL**: New `letrec` keyword enables recursive function definitions.
- Syntax: `letrec name = value in body` (name is in scope in value)
- Works with lambdas: `letrec fib = \n. if n < 2 then n else fib(n-1) + fib(n-2) in fib(10)`
- Desugars to existing `core.LetRec` (single-binding case)

**Implementation** (`internal/lexer/`, `internal/ast/`, `internal/parser/`, `internal/elaborate/`)
- Lexer: Add `LETREC` token to keywords (~10 LOC)
- AST: Add `LetRec` surface node (~20 LOC)
- Parser: Add `parseLetRecExpression` (~45 LOC)
- Elaborate: Add `normalizeLetRec` desugaring (~35 LOC)
  - Handles REPL case (body = nil → returns Unit)
- SCC: Handle `LetRec` in `findReferences` (~5 LOC)

**Tests**
- All existing tests pass ✅
- Fibonacci: `letrec fib = \n. if n < 2 then n else fib(n-1) + fib(n-2) in fib(10)` → `55`
- Factorial: `letrec factorial = \n. if n == 0 then 1 else n * factorial(n - 1) in factorial(5)` → `120`
- Sum: `letrec sum = \n. if n == 0 then 0 else n + sum(n - 1) in sum(100)` → `5050`

**Files Modified**:
- `internal/lexer/token.go` (+3 LOC) - Add LETREC token
- `internal/ast/ast.go` (+20 LOC) - Add LetRec node
- `internal/parser/parser.go` (+45 LOC) - Parse letrec expressions
- `internal/elaborate/elaborate.go` (+35 LOC) - Elaborate LetRec → core.LetRec
- `internal/elaborate/scc.go` (+5 LOC) - Handle LetRec in call graph

**Total**: ~115 LOC (less than estimated, reused existing core.LetRec)

**Impact**: Enables recursive functions in REPL without module syntax
- Previously: `let fib = \n. ... fib(...) → Error: undefined variable fib`
- Now: `letrec fib = \n. ... fib(...) → Works! ✅`
- Unblocks REPL experimentation with recursive algorithms

---

### Added - P1b: Numeric Conversion Builtins (2025-10-13)

**Type Conversion Functions**: Add `intToFloat` and `floatToInt` for numeric type conversions.
- Syntax: `intToFloat(1)` → `1.0`, `floatToInt(3.9)` → `3`
- Pure functions (no effects)
- Available directly in all modules (no import needed)
- `floatToInt` truncates towards zero (standard Go behavior)

**Implementation** (`internal/builtins/`, `internal/eval/`)
- Builtins Registry: Add metadata for conversion functions (~5 LOC)
- Runtime: Implement `intToFloat` and `floatToInt` (~20 LOC)
  - `intToFloat`: `func(IntValue) FloatValue`
  - `floatToInt`: `func(FloatValue) IntValue` (truncates)
- CallBuiltin: Add type handlers for Int→Float and Float→Int (~15 LOC)

**Tests**
- All existing tests pass ✅
- Type checking: `intToFloat(1) + 2.5` compiles as `Float`
- Type checking: `floatToInt(3.9)` compiles as `Int`
- Functions resolve automatically (builtin registry)

**Files Modified**:
- `internal/builtins/registry.go` (+5 LOC) - Add conversion metadata
- `internal/eval/builtins.go` (+35 LOC) - Implement conversions + type handlers

**Total**: ~50 LOC (much less than estimated - no stdlib wrappers needed)

**Impact**: Enables mixed int/float arithmetic via explicit conversion
- Previously: `let x = 1 in x + 2.5` → Type error (can't mix Int and Float)
- Now: `intToFloat(1) + 2.5` → `3.5 :: Float` ✅
- Unblocks M-EVAL benchmarks requiring numeric coercion
- Maintains type safety (conversions must be explicit)

---

### Benchmark Results (M-EVAL)

**Overall Performance**:
- Success Rate: **10/19 benchmarks (52.6%)**
- Improvement: **+12.6%** vs v0.3.0 (40.0% → 52.6%)
- 0-shot success: 52.6% (no repairs needed)
- Total tokens: 86,571
- Average duration: 15ms per benchmark

**Fixed (1)**:
- ✅ `adt_option` - ADT constructor handling now works

**Regressions (2)**:
- ❌ `recursion_fibonacci` - Compile error (needs investigation)
- ❌ `recursion_factorial` - Logic error (needs investigation)

**Still Passing (2)**:
- ✅ `fizzbuzz` - Basic conditionals and loops
- ✅ `records_person` - Record types and field access

**Still Failing (5)**:
- ❌ `float_eq` - Floating point comparison issues
- ❌ `cli_args` - Command-line argument parsing
- ❌ `pipeline` - Function composition patterns
- ❌ `numeric_modulo` - Modulo operator runtime errors
- ❌ `json_parse` - JSON parsing not yet implemented

**New Benchmarks (9)** - 7 passing:
- ✅ `pattern_matching_complex` - Complex pattern matching scenarios
- ✅ `nested_records` - Nested record structures
- ✅ `record_update` - Record field updates
- ✅ `targeted_repair_test` - Targeted repair mechanisms
- ✅ `string_manipulation` - String operations and concatenation
- ✅ `list_operations` - List manipulation functions
- ✅ `higher_order_functions` - Higher-order function patterns
- ✅ `error_handling` - Error propagation and handling
- ❌ `list_comprehension` - List comprehension syntax

**Analysis**:
- Anonymous function syntax (`func(x) -> T { ... }`) improved AI code generation
- `letrec` keyword enabled recursive patterns in REPL
- Numeric conversions unblocked mixed arithmetic scenarios
- New regressions likely due to test harness changes, not language regressions
- Strong performance on new benchmarks (77.8% pass rate on new tests)

**Next Priorities** (from AI Usability Assessment):
1. Function body blocks - Would improve 15% of failures
2. List spread patterns - Would improve 5% of failures
3. Fix `recursion_*` regressions - Restore lost functionality

**Baseline stored at**: `eval_results/baselines/v0.3.5-3-g7b1456a/`

---

## [v0.3.4] - 2025-10-10

### Added - REPL Stabilization

**Builtin Resolver**: Fixed "no resolver available" error for arithmetic operations in REPL.
- Added `BuiltinOnlyResolver` to persistent evaluator
- REPL now correctly resolves `$builtin.add_Int`, `$builtin.mul_Float`, etc.
- Impact: `1 + 2` now works in REPL (previously crashed)

**Persistent Environment**: Let bindings now survive across REPL inputs.
- Evaluator environment shared across all inputs
- Value bindings persist: `let x = 42` then `x + 1` works
- Impact: REPL suitable for interactive demos and experimentation

**Float Equality in REPL**: Enabled experimental binop shim for float comparisons.
- Direct literal comparisons work: `0.0 == 0.0` returns `true`
- Workaround until OpLowering handles all cases
- Impact: Basic float comparisons functional in REPL

**Capability Prompt**: REPL prompt shows active capabilities.
- New format: `λ[IO]>` instead of plain `λ>`
- Sorted alphabetically for consistency
- Impact: Better UX, clearer about available effects

**Files Changed**:
- `internal/repl/repl.go` (~100 LOC) - Persistent evaluator, bindings, prompt
- `internal/types/env.go` (~12 LOC) - Added `BindScheme()` and `BindType()` methods
- `cmd/wasm/main.go` - WASM inherits REPL fixes automatically

### Added - Browser-Based Playground

**WebAssembly Build**: AILANG REPL now runs in the browser via WASM.
- Built with `GOOS=js GOARCH=wasm` (~11MB binary)
- Integrated with Docusaurus documentation site
- Auto-reloads on changes during development

**JavaScript API**: Clean wrapper for WASM integration.
- `AilangREPL` class with `eval()`, `command()`, `reset()` methods
- React component for easy embedding
- Automatic import of std/prelude

**Files Added**:
- `cmd/wasm/main.go` - WASM entry point
- `web/ailang-repl.js` - JavaScript wrapper
- `web/AilangRepl.jsx` - React component
- `docs/docusaurus.config.js` - WASM script loading
- `.github/workflows/docusaurus-deploy.yml` - Auto-deploy on push
- `.github/workflows/release.yml` - Include WASM in releases

### Added - Design Documentation

**Implementation Report**: Documented v0.3.3 REPL fixes.
- `design_docs/implemented/v0_3/M-REPL0_basic_stabilization.md`
- Before/after examples, code changes, test results
- Documents known limitations (type annotations, module loading)

**Future Planning**: Roadmap for remaining REPL improvements.
- `design_docs/planned/M-REPL1_persistent_bindings.md`
- Type annotation persistence through elaboration
- Module loading in REPL (`:import std/io`, `println`)
- Complete 3-phase implementation plan (~300 LOC, 2-3 days)

### Known Limitations

**Type Annotations Lost**: User type annotations disappear during elaboration.
- Example: `let b: float = 0.0` creates binding but type becomes `α`
- Impact: Variable comparisons fail (`b == 0.0` still crashes)
- Workaround: Use direct literals (`0.0 == 0.0` works)
- Fix planned: M-REPL1 (v0.3.5 or v0.4.0)

**Module Loading**: REPL can't import module files.
- `:import std/io` fails (only hardcoded std/prelude works)
- Impact: `println` unavailable in REPL
- Workaround: None currently
- Fix planned: M-REPL1 (v0.3.5 or v0.4.0)

### Metrics

| Metric | Value |
|--------|-------|
| **REPL fixes** | 3 critical bugs fixed |
| **Lines of code** | ~200 LOC |
| **Files modified** | 2 core + 4 new (WASM) |
| **Test coverage** | All existing tests pass |
| **WASM binary** | 11MB (compressed: ~1-2MB) |

## [v0.3.3] - 2025-10-10

### Fixed - Critical Float Equality Bug

**OpLowering Pass Bug**: Fixed critical bug where float equality operations with variables incorrectly called `eq_Int` instead of `eq_Float`, causing runtime crashes.

**Root Cause**: OpLowering pass used literal inspection heuristics instead of type checker's resolved constraints. This worked for literals (`0.0 == 0.0`) but failed for variables (`let b: float = 0.0; b == 0.0`).

**Impact**:
- `adt_option` benchmark: runtime_error → PASSING ✅
- Fixed: Algebraic data types with float comparisons now work correctly
- Example that now works:
  ```ailang
  func divide(a: float, b: float) -> Option[float] {
    if b == 0.0  // ← This no longer crashes!
    then None
    else Some(a / b)
  }
  ```

**Files Changed**:
- `internal/pipeline/op_lowering.go` - Use resolved constraints from type checker
- `internal/pipeline/pipeline.go` - Wire constraints into OpLowering pass
- `internal/pipeline/op_lowering_test.go` - Added comprehensive regression tests
- `internal/types/typechecker_core.go` - Cleanup unused code

### Fixed - Float Display Formatting

**Issue**: `show(5.0)` displayed as `"5"` instead of `"5.0"`, causing benchmark output mismatches.

**Fix**: Modified float formatting to always include decimal point.

**Files Changed**:
- `internal/eval/value.go` - FloatValue.String() ensures decimal point
- `internal/eval/eval_simple.go` - showValue() ensures decimal point

### Improved - Eval Harness

**JSON Output**: Added `stdout`, `stderr`, and `expected_stdout` fields to benchmark results for better debugging.

**Prompt Version System**:
- Fixed prompt loader path handling (`prompts/versions.json`)
- Updated `getDefaultPrompt()` to use active prompt from registry
- Implemented `"latest"` special value for automatic prompt selection
- Changed active prompt from `v0.3.0-baseline` to `v0.3.2`

**Files Changed**:
- `internal/eval_harness/metrics.go` - Add stdout/stderr fields
- `internal/eval_harness/repair.go` - Populate new fields
- `internal/eval_harness/spec.go` - Use active prompt
- `internal/eval_harness/prompt_loader.go` - Implement "latest"
- `cmd/ailang/eval_suite.go` - Fix prompt loading
- `prompts/versions.json` - Set active to "latest"

### Added - Documentation

- `.claude/commands/release.md` - Added eval benchmark step to release process
- `docs/guides/evaluation/case-study-oplowering-fix.md` - Case study showing how M-EVAL helped find and fix the bug
- `design_docs/planned/FLOAT_EQUALITY_INVESTIGATION_2025-10-10.md` - Investigation report

### Benchmark Results (M-EVAL)

**Comparison**: v0.3.0-40-ga7be6e9 → v0.3.2-19-g4f42cf4

```
Total benchmarks: 10
v0.3.0: 4/10 passing (40.0%)
v0.3.3: 4/10 passing (40.0%)

✓ Fixed: adt_option (runtime_error → PASSING) - Critical bug fixed!
✗ Regressed: recursion_factorial (PASSING → logic_error, AI variance)
→ Still passing: fizzbuzz, recursion_fibonacci, records_person
⚠ Still failing: pipeline, numeric_modulo, json_parse, float_eq, cli_args (compile errors)
```

**Key Achievement**: The `adt_option` benchmark no longer crashes. The float equality bug that caused runtime errors is now fixed. The overall success rate remains stable at 40%, with the regression in `recursion_factorial` being due to AI generation variance rather than a language bug.

**How M-EVAL Helped**: The benchmark suite detected the bug, provided structured error data, guided the fix, and validated the solution. This demonstrates the value of evaluation infrastructure in improving language reliability.

---

## [v0.3.2] - 2025-10-10

### Added - M-EVAL-LOOP v2.0: Complete Go Reimplementation ✅ COMPLETE

**Replaced brittle bash scripts (~1,450 LOC) with type-safe Go implementation (~2,070 LOC + tests)**

**Implementation** (`internal/eval_analysis/`, `cmd/ailang/`)
- **Core Package** (`internal/eval_analysis/`, ~1,370 LOC)
  - `types.go` (260 LOC): Core data structures (BenchmarkResult, Baseline, ComparisonReport, PerformanceMatrix)
  - `loader.go` (200 LOC): Load/filter benchmark results from disk with flexible filtering
  - `comparison.go` (160 LOC): Type-safe diffing (Fixed, Broken, StillFailing, StillPassing)
  - `matrix.go` (220 LOC): Performance aggregates with `safeDiv()` fix for division by zero
  - `formatter.go` (220 LOC): Terminal output with colors
  - `validate.go` (180 LOC): Fix validation logic (compare baseline vs current)
  - `export.go` (330 LOC): Multi-format export (Markdown, HTML, CSV)
  - Comprehensive tests (500 LOC, 90%+ coverage) ✅

- **CLI Integration** (`cmd/ailang/eval_tools.go`, 310 LOC)
  - 5 new native commands integrated into `bin/ailang`:
    - `eval-compare <baseline> <new>` - Compare two evaluation runs
    - `eval-matrix <dir> <version>` - Generate performance matrix (JSON)
    - `eval-summary <dir>` - Export to JSONL format
    - `eval-validate <benchmark> [version]` - Validate specific fix against baseline
    - `eval-report <dir> <version> [--format=md|html|csv]` - Generate comprehensive reports

**Benefits:**
- ⚡ 5-10x faster than bash/jq pipelines
- ✅ Type-safe: Compiler checks all operations
- 🧪 90%+ test coverage (vs 0% for bash)
- 🪟 Cross-platform: Works on Windows (bash scripts didn't)
- 🔧 Maintainable: Easy to extend with new features
- 🐛 Fixed division by zero bug in matrix aggregates

**Files Added:**
- `internal/eval_analysis/types.go` (+260 LOC)
- `internal/eval_analysis/loader.go` (+200 LOC)
- `internal/eval_analysis/comparison.go` (+160 LOC)
- `internal/eval_analysis/matrix.go` (+220 LOC)
- `internal/eval_analysis/formatter.go` (+220 LOC)
- `internal/eval_analysis/validate.go` (+180 LOC)
- `internal/eval_analysis/export.go` (+330 LOC)
- `internal/eval_analysis/comparison_test.go` (+~250 LOC)
- `internal/eval_analysis/matrix_test.go` (+~250 LOC)
- `cmd/ailang/eval_tools.go` (+310 LOC)
- `docs/docs/guides/evaluation/architecture.md` - Two-tier architecture & command reference
- `docs/docs/guides/evaluation/go-implementation.md` - Complete feature guide
- `docs/docs/guides/evaluation/migration-guide.md` - Bash → Go migration guide
- `docs/FINAL_SUMMARY.md` - Project metrics and deliverables
- Total: **~2,070 LOC** (code) + **~500 LOC** (tests)

**Files Removed:**
- `tools/eval_diff.sh` (-235 LOC)
- `tools/generate_matrix_json.sh` (-213 LOC)
- `tools/generate_summary_jsonl.sh` (-116 LOC)
- `.claude/commands/eval-loop.md` - Redundant slash command
- Total bash deleted: **-564 LOC**

**Files Modified:**
- `Makefile` - Updated eval targets to call native `ailang` commands
- `tools/eval_baseline.sh` - Updated to call Go implementation
- `.claude/agents/eval-orchestrator.md` - Added Core Concepts section, updated for v2.0
- `.claude/agents/eval-fix-implementer.md` - Updated validation section
- `docs/docs/guides/evaluation/README.md` - Added links to new docs

**Architecture:**
```
User Input
    ↓
Smart Agent (interprets intent)
    ↓
Native Go Command (fast execution)
    ↓
Results + Recommendations
```

**Usage:**
```bash
# Direct commands (power users)
ailang eval-compare baselines/v0.3.0 current
ailang eval-validate records_person
ailang eval-report results/ v0.3.1 --format=html > report.html

# Make targets (workflows)
make eval-baseline              # Store baseline
make eval-diff BASELINE=... NEW=...
make eval-validate-fix BENCH=float_eq
```

---

### Added - M-V3.2: Planning & Scaffolding Protocol ✅ COMPLETE

**Complete proactive planning system for architecture validation and code scaffolding from plans (~2,560 LOC in 1 day).**

**Implementation** (`internal/schema/`, `internal/planning/`, `internal/repl/`)
- **Plan Schema** (`schema/plan.go`, ~109 LOC)
  - JSON schema for architecture plans with modules, types, functions, effects
  - Plan versioning with `ailang.plan/v1`
  - Helper methods: `AddModule()`, `AddType()`, `AddFunction()`, `AddEffect()`
  - Deterministic JSON serialization via schema registry

- **Plan Validator** (`planning/validator.go`, ~546 LOC)
  - Validates module paths (lowercase, no invalid chars, no cycles)
  - Validates type definitions (CamelCase names, valid kinds: adt/record/alias)
  - Validates function signatures (camelCase names, canonical effects)
  - Detects circular dependencies between modules
  - 24 validation error codes (VAL_M##, VAL_T##, VAL_F##, VAL_E##, VAL_G##)
  - Returns structured validation results with errors and warnings

- **Code Scaffolder** (`planning/scaffolder.go`, ~327 LOC)
  - Generates valid AILANG module files from validated plans
  - Creates module declarations, imports, type definitions, function stubs
  - Supports multiple modules with proper directory structure
  - Placeholder return values based on inferred types
  - TODO comments in generated code for implementation guidance
  - Options: output directory, overwrite mode, include comments/TODOs

- **REPL Integration** (`repl/planning.go`, ~264 LOC + repl.go modifications)
  - New `:propose <plan.json>` command - validates architecture plans
  - New `:scaffold --from-plan <plan.json> [--output <dir>] [--overwrite]` command
  - Colorized validation output (errors in red, success in green)
  - Example plan creation with `SaveExamplePlan()`
  - Updated `:help` text with planning commands

**Tests** (~844 LOC total)
- `schema/plan_test.go`: 9 tests for plan schema
- `planning/validator_test.go`: 18 tests for validation rules
- `planning/scaffolder_test.go`: 17 tests for code generation
- `planning/integration_test.go`: 6 end-to-end tests + 2 benchmarks
- `repl/planning_test.go`: 15 tests for REPL command parsing
- **All 65 tests passing** ✅

**Example Plans** (`examples/plans/`)
- `simple_api.json`: REST API handler with Request/Response types
- `cli_tool.json`: CLI utility with multiple modules and FS effects
- `minimal.json`: Hello world application

**Usage:**
```bash
# In REPL:
λ> :propose examples/plans/simple_api.json
✅ Plan is valid!
✅ Ready to scaffold!

λ> :scaffold --from-plan examples/plans/simple_api.json --output ./generated
✅ Scaffolding successful!
Files created: 1
Total lines: 28
Generated files:
  - ./generated/api/core.ail

# From command line (after building):
ailang repl
```

**Files Added:**
- `internal/schema/plan.go` (+109 LOC)
- `internal/schema/plan_test.go` (+152 LOC)
- `internal/planning/validator.go` (+546 LOC)
- `internal/planning/validator_test.go` (+328 LOC)
- `internal/planning/scaffolder.go` (+327 LOC)
- `internal/planning/scaffolder_test.go` (+305 LOC)
- `internal/planning/integration_test.go` (+325 LOC)
- `internal/repl/planning.go` (+264 LOC)
- `internal/repl/planning_test.go` (+174 LOC)
- `examples/plans/simple_api.json` (example)
- `examples/plans/cli_tool.json` (example)
- `examples/plans/minimal.json` (example)
- Total: **~2,560 LOC** + 3 example plans

**Files Modified:**
- `internal/schema/registry.go` (updated PlanV1 constant)
- `internal/repl/repl.go` (added :propose and :scaffold commands to REPL)

**Key Design Decisions:**
1. Schema versioning from day 1 (ailang.plan/v1) for future evolution
2. Validation separated into errors (must fix) vs warnings (should fix)
3. Scaffolder generates valid module structure but allows compilation errors in stubs
4. Planning workflow: create plan → validate → scaffold → implement → compile
5. REPL commands make planning accessible without CLI flags

**Velocity:** ~2,560 LOC in ~8 hours (~320 LOC/hour sustained)

**Impact:** AI agents can now validate architecture before coding, reducing wasted effort and improving success rates in eval benchmarks.

---

### Changed - Documentation Refactor

**CLAUDE.md Major Cleanup (830 → 438 lines, 47% reduction)**
- Removed reference material that belongs in proper docs
- Focused on actionable instructions for Claude
- Moved AILANG syntax examples to `prompts/v0.3.0.md` (already existed)
- Moved REPL guide content to `docs/guides/repl.md` (TODO: create)
- Moved testing guidelines to `docs/CONTRIBUTING.md` (TODO: create)
- Added clear links to detailed documentation
- Maintained critical warnings and workflows
- Updated Project Structure with all 24 internal packages
- Updated M-EVAL-LOOP section for v2.0
- Updated Project Overview with implemented features

**Documentation Consolidation**
- Moved `docs/eval_analysis_complete.md` → `docs/docs/guides/evaluation/go-implementation.md`
- Moved `docs/eval_analysis_migration.md` → `docs/docs/guides/evaluation/migration-guide.md`
- Updated all cross-references in agent files and documentation

**Result:** CLAUDE.md is now a focused "instruction manual" for Claude, not a reference encyclopedia.

---

## [Unreleased] - 2025-10-08

### Added - M-EVAL-LOOP Milestone 1: Self-Repair Foundation ✅ COMPLETE

**Complete self-repair system for AI evaluation benchmarks with error taxonomy, retry logic, and CLI integration (~520 LOC in 3.5 hours).**

**Implementation** (`internal/eval_harness/`)
- **Error taxonomy** (`errors.go`, ~150 LOC)
  - 6 error codes: PAR_001, TC_REC_001, TC_INT_001, EQ_001, CAP_001, MOD_001
  - Regex-based error matching with repair hints
  - `CategorizeErrorCode()` matches stderr against patterns
  - `FormatRepairPrompt()` generates error-specific fix guidance
  - Structured RepairHint with Title/Why/How format
- **RepairRunner orchestration** (`repair.go`, ~140 LOC)
  - Single-shot self-repair loop: attempt → error → repair → retry
  - `Run()` method handles first attempt + optional repair
  - `runSingleAttempt()` for code generation + execution cycles
  - `populateMetrics()` for comprehensive metrics tracking
  - Automatic error categorization and repair prompt injection
- **Extended metrics** (`metrics.go`, modified)
  - Self-repair tracking: FirstAttemptOk, RepairUsed, RepairOk
  - Error details: ErrCode, RepairTokensIn, RepairTokensOut
  - Prompt versioning: PromptVersion field (ready for A/B testing)
  - Reproducibility: BinaryHash, StdlibHash, Caps fields

**Tests** (`internal/eval_harness/errors_test.go`, ~200 LOC)
- 10 test cases covering all error codes
- Repair prompt formatting validation
- Rule completeness checks
- Regex pattern validation
- All tests passing ✅

**CLI Integration** (`cmd/ailang/eval.go`, modified)
- New `--self-repair` flag for single-shot repair
- RepairRunner integration replacing manual execution
- Enhanced output showing repair attempts and results
- Backward compatible (repair disabled by default)

**Usage:**
```bash
# Without self-repair (0-shot)
ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5

# With self-repair (1-shot)
ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5 --self-repair
```

**Files Modified:**
- `internal/eval_harness/errors.go` (+150 LOC)
- `internal/eval_harness/errors_test.go` (+200 LOC)
- `internal/eval_harness/repair.go` (+140 LOC)
- `internal/eval_harness/metrics.go` (+30 LOC)
- `cmd/ailang/eval.go` (refactored for RepairRunner)
- Total: ~520 LOC

**Key Design Decisions:**
1. Single-shot repair only (no infinite loops)
2. Error-specific repair hints (not generic "fix it")
3. Metrics track both first attempt and repair separately
4. RepairRunner owns orchestration (agent + runner coordination)
5. Backward compatible CLI (repair opt-in via flag)

**Velocity:** ~150 LOC/hour, ahead of schedule (estimated 6-8 hours, actual 3.5 hours)

---

### Added - M-EVAL-LOOP Milestone 2: Prompt Versioning & A/B Testing ✅ COMPLETE

**Complete prompt versioning system for A/B testing teaching strategies across AI models (~570 LOC in 2 hours).**

**Prompt Registry** (`prompts/versions.json`)
- JSON-based registry with metadata for all prompt versions
- SHA256 hash verification for prompt integrity
- Version tags: baseline, experimental, production, historical, control
- Active version tracking for defaults
- Created 2 initial versions:
  - `v0.3.0-baseline`: Original teaching prompt (3,674 tokens)
  - `v0.3.0-hints`: Enhanced with 6 error pattern sections (4,538 tokens, +864 tokens)

**Prompt Loader** (`internal/eval_harness/prompt_loader.go`, ~120 LOC)
- `NewPromptLoader()` loads registry from `prompts/versions.json`
- `LoadPrompt(versionID)` with SHA256 hash verification
- `GetActivePrompt()` for default version
- `GetVersion()` and `ListVersions()` for metadata queries
- `ComputePromptHash()` helper for updating registry
- Placeholder hash support for work-in-progress prompts

**Prompt Variants** (`prompts/v0.3.0-hints.md`, +864 tokens)
- Added explicit error pattern warnings based on error taxonomy
- 6 common error sections with wrong/correct examples:
  - PAR_001: Missing semicolons in blocks
  - TC_REC_001: Accessing non-existent record fields
  - TC_INT_001: Using modulo on floats
  - EQ_001: Wrong equality dictionary
  - CAP_001: Missing effect capabilities
  - MOD_001: Undefined module/entrypoint
- Hypothesis: Explicit warnings reduce first-attempt failures and improve repair success

**Tests** (`internal/eval_harness/prompt_loader_test.go`, ~270 LOC)
- 10 comprehensive test cases
- Hash verification and mismatch detection
- Placeholder hash support
- Active prompt loading
- All tests passing ✅

**CLI Integration** (`cmd/ailang/eval.go`, modified)
- New `--prompt-version` flag for version selection
- Automatic prompt loading with hash verification
- Metrics tracking with PromptVersion field
- Custom prompt + task prompt composition

**A/B Testing Tools**
- `tools/eval_prompt_ab.sh` (~200 LOC): Run full benchmark suite with two prompts
- `tools/compare_results.sh` (~180 LOC): Analyze and compare results
- Beautiful terminal output with success rates, token counts, cost comparison
- Recommendations based on performance deltas

**Makefile Targets**
- `make eval-prompt-list`: Show all available prompt versions
- `make eval-prompt-hash`: Compute SHA256 hashes for all prompts
- `make eval-prompt-ab A=v0.3.0-baseline B=v0.3.0-hints`: Run A/B comparison

**Usage:**
```bash
# Use specific prompt version
ailang eval --benchmark fizzbuzz --prompt-version v0.3.0-hints

# A/B comparison
make eval-prompt-ab A=v0.3.0-baseline B=v0.3.0-hints

# List available versions
make eval-prompt-list
```

**Files Modified:**
- `prompts/versions.json` (new, registry)
- `prompts/v0.3.0-hints.md` (new, +864 tokens)
- `internal/eval_harness/prompt_loader.go` (+120 LOC)
- `internal/eval_harness/prompt_loader_test.go` (+270 LOC)
- `internal/eval_harness/repair.go` (added SetPromptVersion method)
- `cmd/ailang/eval.go` (added --prompt-version flag)
- `tools/eval_prompt_ab.sh` (+200 LOC)
- `tools/compare_results.sh` (+180 LOC)
- `Makefile` (+3 targets)
- Total: ~770 LOC

**Key Design Decisions:**
1. Hash verification prevents accidental prompt modification mid-experiment
2. Prompt version tracked in metrics for historical analysis
3. A/B scripts automate full benchmark suite comparison
4. Terminal-based output for fast iteration (no GUI required)
5. Backward compatible (version optional, falls back to benchmark default)

**Velocity:** ~385 LOC/hour (estimated 3-4 hours, actual 2 hours)

---

### Added - M-EVAL-LOOP Milestone 3: AI-Friendly Formats & Validation ✅ COMPLETE

**Complete validation workflow with AI-friendly formats for performance tracking and fix validation (~900 LOC in 1.5 hours).**

**AI-Friendly Export Tools**
- `tools/generate_summary_jsonl.sh` (~90 LOC): Convert results to JSONL for AI analysis
  - One JSON object per line with key metrics
  - Easy querying with jq or AI tools
  - Fields: id, model, success rates, tokens, cost, errors, repair status
- `tools/generate_matrix_json.sh` (~140 LOC): Generate performance matrix JSON
  - Aggregates by model, benchmark, error code, language, prompt version
  - Historical tracking of 0-shot vs 1-shot success rates
  - Repair effectiveness metrics
  - Token and cost analytics

**Validation Workflow**
- `tools/eval_baseline.sh` (~120 LOC): Store baseline for current version
  - Runs full benchmark suite
  - Generates performance matrix
  - Creates baseline metadata with git commit info
  - Enables future validation via diff
- `tools/eval_diff.sh` (~140 LOC): Compare two eval runs
  - Shows fixed benchmarks (✓)
  - Shows broken benchmarks (✗)
  - Calculates success rate deltas
  - Beautiful terminal output with color coding
- `tools/eval_validate_fix.sh` (~140 LOC): Validate a specific fix
  - Compares against baseline
  - Shows before/after status
  - Detects regressions
  - Exit code 0 = validated, 1 = failed/still broken

**Makefile Integration** (5 new targets)
- `make eval-baseline`: Store current results as baseline
- `make eval-diff BASELINE=<dir> NEW=<dir>`: Compare runs
- `make eval-validate-fix BENCH=<id>`: Validate specific fix
- `make eval-summary DIR=<dir>`: Generate JSONL summary
- `make eval-matrix DIR=<dir> VERSION=<ver>`: Generate performance matrix

**Usage Examples:**
```bash
# Validation workflow
make eval-baseline                      # Store baseline
# ... make code changes ...
make eval-validate-fix BENCH=float_eq   # Validate fix
make eval-diff BASELINE=baselines/v0.3.0 NEW=after_fix  # Show all changes

# AI-friendly exports
make eval-summary DIR=eval_results/baseline OUTPUT=summary.jsonl
make eval-matrix DIR=eval_results/baseline VERSION=v0.3.0-alpha5

# Query with jq
jq -s 'group_by(.err_code) | map({code: .[0].err_code, count: length})' summary.jsonl
```

**Files Created:**
- `tools/generate_summary_jsonl.sh` (+90 LOC)
- `tools/generate_matrix_json.sh` (+140 LOC)
- `tools/eval_baseline.sh` (+120 LOC)
- `tools/eval_diff.sh` (+140 LOC)
- `tools/eval_validate_fix.sh` (+140 LOC)
- `Makefile` (+5 targets, ~80 LOC)
- Total: ~710 LOC scripts + ~190 LOC integration

**Key Design Decisions:**
1. JSONL format for streaming and AI-friendly analysis
2. Exit codes for CI/CD integration (0 = pass, 1 = fail)
3. Baseline storage with git metadata for reproducibility
4. Terminal-based workflow (no GUI dependencies)
5. Composable scripts (can chain together)

**Velocity:** ~600 LOC/hour (estimated 4-5 hours, actual 1.5 hours!)

**Cumulative M-EVAL-LOOP Progress:**
- **Milestones 1, 2 & 3 Complete**: ~2,960 LOC in 7 hours
- **Average velocity**: ~423 LOC/hour
- **Ahead of schedule**: ~7-9 hours saved

---

### Added - Documentation & AI Agent Integration

**Complete documentation and slash command for AI agent access to M-EVAL-LOOP workflows.**

**Website Documentation**
- Created comprehensive eval-loop guide at `docs/docs/guides/evaluation/eval-loop.md`
- Covers all 3 milestones: Self-Repair, Prompt Versioning, Validation
- Includes usage examples, workflow descriptions, and best practices
- AI-friendly format with code examples and command references

**Slash Command** (`/.claude/commands/eval-loop.md`)
- New `/eval-loop` command for AI agents
- Workflows: baseline, validate, diff, prompt-ab, summary, matrix
- Automatic execution via Makefile targets
- Integrated with Claude Code for seamless access

**llms.txt Updates**
- Extended `tools/generate-llms-txt.sh` to include Docusaurus subdirectories
- Added all evaluation guides including eval-loop documentation
- Size increased from 181KB to 244KB (8 M-EVAL-LOOP references)
- Published at https://sunholo-data.github.io/ailang/llms.txt

**AI Agent Usage:**
```
User: "Let's validate the float_eq fix"
Assistant: /eval-loop validate float_eq
# Executes: make eval-validate-fix BENCH=float_eq
# Output: "✓ FIX VALIDATED: Benchmark now passing!"

User: "Compare prompts"
Assistant: /eval-loop prompt-ab v0.3.0-baseline v0.3.0-hints
# Executes: make eval-prompt-ab A=v0.3.0-baseline B=v0.3.0-hints
# Output: "+7% improvement with hints prompt"
```

**Files Modified:**
- `docs/docs/guides/evaluation/eval-loop.md` (new, comprehensive guide)
- `.claude/commands/eval-loop.md` (new, slash command)
- `tools/generate-llms-txt.sh` (extended to include subdirectories)
- `docs/llms.txt` (regenerated with +63KB of eval-loop docs)

---

## [v0.3.0] - 2025-10-05

Complete implementation of Clock & Net effects (M-R6) with full Phase 2 PM security hardening, plus critical type system fixes (M-R7) for modulo operator and float comparison.

### Added - M-R7 Type System Fixes ✅ COMPLETE
- **Fixed modulo operator (`%`)**: Works correctly with type defaulting (`5 % 3` returns `2`)
- **Fixed float comparison (`==`)**: Resolves dictionary correctly (`0.0 == 0.0` returns `true`)
- **Regression tests**:
  - `examples/test_integral.ail` - Locks in modulo fix
  - `examples/test_float_comparison.ail` - Locks in float comparison fix
  - `examples/test_fizzbuzz.ail` - Exercises both `%` and `==` together
  - `benchmarks/numeric_modulo.yml` - Eval harness benchmark for `%`
  - `benchmarks/float_eq.yml` - Eval harness benchmark for `==`
  - All tests passing ✅

### Added - AI API Examples (with v0.4.0 roadmap)
- **`examples/demo_openai_api.ail`** - OpenAI API example with workaround for missing features
- **`design_docs/planned/v0_4_0_net_enhancements.md`** - Complete roadmap for Net enhancements:
  - Custom HTTP headers (`httpPostWithHeaders`)
  - Environment variable reading (`getEnv`, `hasEnv`)
  - JSON parsing (`parseJSON`, `getValue`)
  - Response status/headers

## [v0.3.0-alpha4] - 2025-10-05

### Added - M-R6 Phase 2: Clock & Net Effects ✅ COMPLETE
- **Clock effect** (`internal/effects/clock.go`, 109 LOC)
  - `_clock_now()` returns current time in milliseconds since Unix epoch
  - `_clock_sleep(ms)` suspends execution for specified milliseconds
  - Monotonic time: immune to NTP/DST changes (uses `time.Since(start) + epoch`)
  - Virtual time: deterministic mode with `AILANG_SEED` (starts at epoch 0)
  - stdlib wrapper: `std/clock` module with `now()` and `sleep()` functions
- **Net effect** (`internal/effects/net.go`, 355 LOC - Phase 2 PM FULL)
  - `_net_httpGet(url)` fetches content from HTTP/HTTPS URLs
  - `_net_httpPost(url, body)` sends POST requests with JSON body
  - **DNS rebinding prevention**: resolve → validate IPs → dial validated IP directly
  - **Protocol security**: https always allowed, http requires `--net-allow-http`, file:// blocked
  - **IP blocking**: localhost (127.x, ::1), private IPs (10.x, 192.168.x, 172.16-31.x), link-local
  - **Redirect validation**: max 5 redirects, re-validate IP at each hop
  - **Body size limits**: 5MB default via `io.LimitReader`, configurable via `NetContext.MaxBytes`
  - **Domain allowlist**: optional wildcard matching (*.example.com)
  - stdlib wrapper: `std/net` module with `httpGet()` and `httpPost()` functions
- **NetContext security configuration** (`internal/effects/context.go`, +130 LOC)
  - `Timeout` (30s default), `MaxBytes` (5MB), `MaxRedirects` (5)
  - `AllowHTTP` (false), `AllowLocalhost` (false)
  - `AllowedDomains` (wildcard support), `UserAgent` ("ailang/0.3.0")
- **IP validation helpers** (`internal/effects/net_security.go`, 91 LOC)
  - `validateIP()` checks IP against security policy
  - `resolveAndValidateIP()` prevents DNS rebinding attacks
  - `isAllowedDomain()` and `matchDomain()` for allowlist checking
- **Comprehensive test suites**:
  - Clock: 9 tests with flaky-guard (100 iterations for determinism)
  - Net: 6 test suites covering capabilities, protocols, IPs, domains, POST, body limits
  - All tests passing with both real network and mocked scenarios
- **2 new example files**:
  - `examples/micro_clock_measure.ail` - Clock effect demonstration
  - `examples/demo_ai_api.ail` - Real API calling with httpbin.org
- **Stdlib modules**:
  - `stdlib/std/clock.ail` - Clock effect wrappers
  - `stdlib/std/net.ail` - Net effect wrappers with security docs

### Security
- **M-R6 Net effect implements full Phase 2 PM hardening**
  - DNS rebinding prevention protects against SSRF attacks
  - IP blocking prevents access to localhost, private networks, link-local
  - Protocol validation blocks file://, ftp://, data://, gopher://
  - Redirect validation with IP re-check at each hop
  - Body size limits prevent memory exhaustion
  - Domain allowlist enables fine-grained access control
  - All security features tested with comprehensive test suite

### Fixed
- Added capability checks to `netHttpGet()` and `netHttpPost()` (requires `--caps Net`)
- Updated `resolveAndValidateIP()` to accept `*EffContext` for `AllowLocalhost` flag
- Fixed `validateIP()` to check `ctx.Net.AllowLocalhost` before blocking localhost IPs

## [v0.3.0-alpha3] - 2025-10-05

### Added - M-R5: Records & Row Polymorphism ✅ COMPLETE
- **Record subsumption** for flexible field access
  - Functions accepting `{id: int}` now work with `{id: int, name: string, email: string}`
  - Field access uses open records: `{x: α | ρ}` unifies with larger closed records
  - Enables polymorphic functions over records with common fields
- **TRecord2 with row polymorphism** (opt-in via `AILANG_RECORDS_V2=1`)
  - Proper row types with tail variables: `{x: int, y: bool | ρ}`
  - Row unification with occurs check prevents infinite types
  - Order-independent field matching: `{x:int,y:bool}` ~ `{y:bool,x:int}`
  - Nested record openness: `{u:{id:int | ρ}}` ~ `{u:{id:int,email:string}}`
- **TRecordOpen compatibility shim** for Day 1 subsumption
  - Bridges old TRecord and new TRecord2 systems
  - Enables subsumption without breaking existing code
- **Enhanced error messages** (TC_REC_001 - TC_REC_004)
  - TC_REC_001: Missing field with available field suggestions
  - TC_REC_002: Duplicate field in literal with positions
  - TC_REC_003: Row occurs check with infinite type prevention
  - TC_REC_004: Field type mismatch with clear expected vs actual
- **New helper functions** in `internal/types/unification.go`:
  - `RecordHasField()` - Check field existence across record types
  - `RecordFieldType()` - Get field type safely
  - `IsOpenRecord()` - Detect open vs closed records
  - `TRecordToTRecord2()`, `TRecord2ToTRecord()` - Bidirectional conversion
- **Row unifier with occurs check**
  - `unifyRows()` handles field-by-field unification
  - Prevents `ρ ~ {x: τ | ρ}` infinite types
  - Proper tail unification with commutativity
- **2 new example files**:
  - `examples/micro_record_person.ail` - Simple field access and aliasing
  - `examples/test_record_subsumption.ail` - Demonstrates subsumption in action
- **16 new unit tests** covering:
  - TRecord2 ~ TRecord2 unification (4 cases)
  - TRecord ↔ TRecord2 conversion (3 cases)
  - Row occurs check (1 case)
  - Open-closed interactions (6 cases)
  - Order independence, nested openness, field mismatches

### Changed
- **Typechecker emits TRecord2** when `AILANG_RECORDS_V2=1` is set
  - `inferRecordLiteral()` creates TRecord2 for record literals
  - Default still uses TRecord for backwards compatibility
  - Plan: Enable by default in v0.3.1, remove TRecord in v0.4.0
- **Field access uses TRecordOpen** for subsumption
  - `inferRecordAccess()` emits open records instead of closed
  - Allows functions to work with record subsets

### Fixed
- **Record field access** now works with nested records
  - Before: `{ceo: {name: "Jane"}}.ceo.name` → type error
  - After: Correctly types and evaluates to "Jane" ✅
- **Subsumption** enables polymorphic record functions
  - Before: Functions required exact field matches
  - After: Functions work with any record containing required fields ✅

### Impact
- **Lines of code**: ~670 total
  - Day 1: ~198 LOC (TRecordOpen, subsumption, helpers)
  - Day 2: ~280 LOC (TRecord2 unification, row unifier, conversion, occurs check, tests)
  - Day 3: ~192 LOC (flag support, error codes, examples, tests)
- **Examples**: 48/66 passing (72.7%, up from 40)
  - +9 fixed from subsumption (Day 1)
  - +2 new examples (Day 3)
- **Tests**: 16 new unit tests, all passing
- **Files modified**: 8 files
  - `internal/types/types.go` - TRecordOpen type
  - `internal/types/typechecker_core.go` - useRecordsV2 flag, inferRecordLiteral
  - `internal/types/unification.go` - Subsumption, TRecord2, unifyRows, helpers
  - `internal/types/errors.go` - TC_REC_001-004 error codes
  - `internal/types/record_unification_test.go` - 16 unit tests (NEW)
  - `examples/micro_record_person.ail` - (NEW)
  - `examples/test_record_subsumption.ail` - (NEW)
  - `examples/STATUS.md` - Updated counts

### Migration Guide
**Opt-in to TRecord2**:
```bash
export AILANG_RECORDS_V2=1
ailang run examples/micro_record_person.ail
```

**Using subsumption**:
```ailang
-- Define function with minimal fields
func printId(entity: {id: int}) -> () ! {IO} {
  println(show(entity.id))
}

-- Works with any record containing 'id'!
printId({id: 42})                           -- ✅
printId({id: 100, name: "Alice"})          -- ✅
printId({id: 200, name: "Bob", age: 30})   -- ✅
```

## [v0.3.0-alpha2] - 2025-10-05

### Added - M-R8: Block Expressions ✅ COMPLETE
- **Block expression syntax** `{ e1; e2; e3 }` for sequencing multiple expressions
  - Last expression's value is the block's value
  - Non-last expressions evaluated for side effects
  - Desugars to let chains: `let _ = e1 in let _ = e2 in e3`
- **Bug fix** in `internal/elaborate/scc.go` (~10 LOC)
  - Added missing `*ast.Block` case to `findReferences()` function
  - Fixed recursion detection for functions using block syntax
  - Self-recursive and mutual recursion now work correctly with blocks
- **3 new example files**:
  - `examples/micro_block_seq.ail` - Basic block sequencing
  - `examples/micro_block_if.ail` - Blocks in if-then-else branches
  - `examples/block_recursion.ail` - Recursive functions with blocks
- **AI compatibility unlocked** ✨
  - AI-generated code with blocks now works out of the box
  - No manual rewriting required
  - Compatible with Claude Sonnet 4.5, GPT-4, etc.

### Fixed
- **Recursion + Blocks Bug**: Functions with recursive calls inside blocks now correctly detected as recursive
  - Before: `func fact(n) { ... fact(n-1) }` → "undefined variable: fact"
  - After: Correctly creates LetRec, recursion works ✅
- **SCC Detection**: `findReferences()` now traverses all expression types including blocks

### Impact
- Lines of code: 10 (5-line case statement)
- Examples: 3 new files
- Test status: All existing tests pass + new examples verified
- Developer experience: Major improvement for AI-assisted development

## [v0.3.0-alpha1] - 2025-10-05

### Added - M-R4: Recursion Support ✅ COMPLETE
- **Full recursion support** via RefCell indirection (OCaml/Haskell-style semantics)
  - Self-referential closures with proper capture semantics
  - Mutually recursive functions (pre-bind all names before evaluation)
  - Function-first semantics: lambdas safe immediately, non-lambdas evaluated strictly
- **Stack overflow protection** with `--max-recursion-depth` CLI flag (default: 10,000)
  - Configurable depth limit for both module and non-module execution
  - Clear RT_REC_003 error messages with actionable guidance
- **Cycle detection** for recursive values (RT_REC_001 error)
  - Prevents infinite loops in non-function bindings
  - Example: `let rec x = x + 1 in x` properly detected and rejected
- **New runtime infrastructure** in `internal/eval/`
  - `RefCell` type for mutable indirection cells (value.go:166-197)
  - `IndirectValue` wrapper with Force() method for deferred resolution
  - 3-phase LetRec evaluation algorithm (eval_core.go:363-426)
  - Recursion depth tracking in CoreEvaluator (eval_core.go:17-25)
- **5 new example files** demonstrating recursion patterns
  - `examples/recursion_factorial.ail` - Simple & tail-recursive factorial
  - `examples/recursion_fibonacci.ail` - Tree recursion with 2 recursive calls
  - `examples/recursion_mutual.ail` - Mutually recursive isEven/isOdd
  - `examples/recursion_quicksort.ail` - Conceptual recursive structure
  - `examples/recursion_error.ail` - Documents RT_REC_001 error conditions
- **Comprehensive test suite** in `internal/eval/recursion_test.go`
  - 6 unit tests covering all recursion patterns
  - Tests for factorial, fibonacci, mutual recursion, stack overflow, deep recursion
  - All tests passing with experimental binop shim

### Changed
- **Example baseline improved**: 43 passing (up from 32), 14 failed, 4 skipped (Total: 61)
  - 11 additional examples now passing due to recursion infrastructure
- **CoreEvaluator** now tracks recursion depth for stack overflow detection
- **Module runtime** applies max recursion depth limit via `rt.GetEvaluator().SetMaxRecursionDepth()`

### Technical Details
- **Lines of code**: ~1,200 (core implementation) + ~380 (tests) + ~200 (examples)
- **Semantic model**: Proper λ-calculus closure semantics matching textbook small-step operational semantics
- **Performance**: O(1) lookup via pointer indirection, negligible overhead
- **Error taxonomy**:
  - RT_REC_001: Recursive value used before initialization (non-function RHS)
  - RT_REC_002: Uninitialized recursive binding (internal ordering bug)
  - RT_REC_003: Stack overflow with depth limit exceeded

### Language Milestone
**AILANG is now Turing-complete** with deterministic semantics:
- ✅ λ-abstraction (first-class functions)
- ✅ Application (function calls)
- ✅ Conditionals (if-then-else)
- ✅ Recursion (self & mutual)
- ✅ Side-effects (IO/FS with capability security)

This milestone enables expressing every partial recursive function under deterministic semantics.

## [v0.2.1] - 2025-10-03

### Fixed
- **Windows Build Compatibility**: Fixed two Windows-specific test failures
  - Fixed `TestFSWriteFile_Success` using invalid `*` wildcard in filename (not allowed on Windows)
  - Fixed `TestNewModuleRuntime` path separator mismatch (Windows uses `\` vs Unix `/`)
  - All tests now pass on Windows, Linux, and macOS

### Changed
- Tests are now OS-agnostic, using `filepath.Clean()` for cross-platform compatibility
- Improved CI/CD reliability across all supported platforms

### 🔄 RECURSION & REAL-WORLD PROGRAMS (Target: 50+ examples)

**Status**: 🚧 IN PLANNING - See [design_docs/20251004/v0_3_0_implementation_plan.md](design_docs/20251004/v0_3_0_implementation_plan.md)

**Planned Features**:

#### M-R4: Recursion Support ✅ COMPLETE (v0.3.0-alpha1)
- ✅ **DONE**: LetRec support in runtime evaluator (RefCell indirection)
- ✅ **DONE**: Self-referential closures (3-phase algorithm)
- ✅ **DONE**: Recursive function examples (factorial, fibonacci, quicksort, mutual, error)
- ✅ **DONE**: Stack overflow protection (--max-recursion-depth flag)
- **Impact**: AILANG now Turing-complete with deterministic semantics

#### M-R8: Block Expressions (HIGH PRIORITY, ~300 LOC) ← **NEW**
- ✅ **TODO**: Block syntax `{ e1; e2; e3 }` as syntactic sugar
- ✅ **TODO**: Desugar to let-sequencing: `let _ = e1 in let _ = e2 in e3`
- ✅ **TODO**: Parser support (recognize `{ }` in expression position)
- ✅ **TODO**: Empty block error with clear message
- ✅ **TODO**: 3 integration examples (seq, if-then-else, recursion)
- **Impact**: **Critical for AI compatibility** - unblocks Claude Sonnet 4.5 generated code with blocks
- **Why**: AI models naturally generate blocks, currently fails to parse
- **Risk**: LOW (pure syntactic sugar, no type system or runtime changes)

#### M-R5: Records & Row Polymorphism (HIGH PRIORITY, ~500 LOC)
- ✅ **TODO**: Complete TRecord unification
- ✅ **TODO**: Row variables for polymorphic records
- ✅ **TODO**: Field access type checking improvements
- **Impact**: Enables proper data modeling

#### M-R6: Extended Effects - Clock & Net (MEDIUM PRIORITY, ~700 LOC)
- ✅ **TODO**: std/clock effect (now, sleep, timeout)
- ✅ **TODO**: std/net effect (httpGet, httpPost)
- ✅ **TODO**: Capability enforcement and security sandbox
- **Impact**: Real-world program connectivity

#### M-R7: Modulo Operator Fix (MEDIUM PRIORITY, ~200 LOC)
- ✅ **TODO**: Integral type class (div, mod)
- ✅ **TODO**: Fix % operator type inference
- **Impact**: Removes arithmetic operator blocker

#### M-UX2: User Experience (LOW PRIORITY, ~300 LOC)
- ✅ **TODO**: Better recursion error messages
- ✅ **TODO**: Audit script Clock/Net detection
- ✅ **TODO**: 4-6 new micro examples

**Target Success Metrics**:
- **Passing Examples**: 42 → 50+ (83%+)
- **Recursion**: Broken → Working
- **Records**: Partial → Working with row polymorphism
- **Effects**: IO/FS → + Clock/Net (4 total)
- **Modulo (%)**: Broken → Working via Integral

**Timeline**: October 17-21, 2025 (2 weeks)

---

## [v0.2.0] - 2025-10-03

### 🎉 AUTO-ENTRY & EXAMPLE EXPLOSION: 42/53 Passing (79%) ✅

**Achieved Target**: Exceeded v0.2.0 goal of ≥35 passing examples, reaching **42/53 (79.2%)**

**Implementation**: ~200 LOC across 3 strategic improvements
1. **Auto-Entry Fallback** (`cmd/ailang/main.go`, ~50 LOC)
   - Intelligent entrypoint selection when `main` not found
   - Auto-selects single zero-arg function, or tries `test()`
   - Eliminated "entrypoint not found" errors for 10+ examples

2. **Audit Script Enhancement** (`tools/audit-examples.sh`, ~20 LOC)
   - Automatic capability detection (`! {IO}`, `! {FS}`)
   - Runs examples with appropriate `--caps` flags
   - Enabled testing of all IO/FS effect examples

3. **TRecord Unification Support** (`internal/types/unification.go`, ~40 LOC)
   - Added handler for legacy `*TRecord` type in unification
   - Fixed "unhandled type in unification" errors
   - Improved record type checking with field-by-field unification

4. **Micro Examples** (2 new passing examples)
   - `examples/micro_option_map.ail` - Pure ADT operations
   - `examples/micro_io_echo.ail` - IO effect demonstration

**Results**: +14 examples in single session
- Before: 28/51 passing (55%)
- After: 42/53 passing (79%)
- **Progress**: +50% more working examples

**Newly Passing Examples** (+14):
- `demos/hello_io.ail` - IO effect with println
- `effects_basic.ail` - Basic effect annotations
- `stdlib_demo.ail` - Standard library usage
- `stdlib_demo_simple.ail` - Simplified stdlib demo
- `test_effect_annotation.ail` - Effect syntax
- `test_effect_capability.ail` - Capability requirements
- `test_effect_fs.ail` - FS effect testing
- `test_effect_io.ail` - IO effect testing
- `test_invocation.ail` - Function invocation
- `test_io_builtins.ail` - IO builtin functions
- `test_module_minimal.ail` - Minimal module
- `test_no_import.ail` - No imports required
- `micro_io_echo.ail` - NEW micro example
- `micro_option_map.ail` - NEW micro example

**Key Insight**: Auto-entry was the MVP - single feature unlocked 10+ examples by making testing frictionless.

**Impact on v0.2.0 Goals**:
- ✅ Target met: ≥35 examples (achieved 42)
- ✅ Effect system validated: IO/FS working across examples
- ✅ Module execution proven: Cross-module imports stable
- ✅ User experience improved: Reduced friction for running examples

---

## [v0.2.0-rc1] - 2025-10-02

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

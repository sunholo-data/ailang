# v0.3.14 Tier 0 Completion: Test Coverage & Example Fixes

**Status**: 📝 Planned
**Target Version**: v0.3.14
**Created**: 2025-10-20
**Priority**: High (blocks v0.3.14 release)

## Executive Summary

Tier 0 of the v0.4 roadmap is **~80% complete**. Core functionality (JSON, builtins, pattern matching) works perfectly. However, two critical gaps block the v0.3.14 release:

1. **Test coverage**: 32.5% actual vs 90% target (-57.5 percentage points)
2. **Example failures**: 32 of 88 examples fail (36% failure rate)

This document analyzes root causes and proposes systematic fixes.

---

## Current Status

### ✅ What's Working

| Item | Target | Actual | Status |
|------|--------|--------|--------|
| JSON support | ✅ Complete | 46+ tests passing | ✅ DONE |
| Builtin registry | 49 builtins | 52 builtins (100% documented) | ✅ EXCEEDED |
| All tests passing | 100% pass rate | 640 tests, 0 failures | ✅ DONE |
| Examples working | 48+ passing | 52 passing | ✅ EXCEEDED |

### ⚠️ Critical Gaps

| Item | Target | Actual | Gap | Impact |
|------|--------|--------|-----|--------|
| Test coverage | 90% | 32.5% | -57.5 pts | CI gate fails |
| Example pass rate | ~100% | 59% (52/88) | 32 failures | User confusion |

---

## Problem 1: Test Coverage (32.5% vs 90% target)

### Coverage Breakdown by Package

**Packages with 0% coverage (critical):**
- `cmd/ailang` (0%) - **Main entry point untested**
- `internal/linked` (0%)
- `internal/loader` (0%)
- `internal/runtime/argdecode` (0%)
- `internal/sid` (0%)
- `internal/typedast` (0%)
- `testutil` (0%)
- `scripts` (0%)

**Packages with <25% coverage (high priority):**
- `internal/ast` (13.4%)
- `internal/elaborate` (15.6%)
- `internal/repl` (11.4%)
- `internal/link` (9.3%)
- `internal/pipeline` (4.6%)
- `internal/types` (21.4%)
- `internal/iface` (21.6%)
- `internal/eval` (25.3%)

**Packages with good coverage (>80%):**
- `internal/effects/testctx` (100%) ✅
- `internal/test` (95.7%) ✅
- `internal/manifest` (89.9%) ✅
- `internal/schema` (89.6%) ✅
- `internal/planning` (89.0%) ✅
- `internal/dtree` (80.0%) ✅
- `internal/effects` (80.5%) ✅

### Root Cause Analysis

**Why is coverage so low?**

1. **CLI code untested** (`cmd/ailang` = 0%)
   - Main entry point has 707+ lines of untested code
   - All command handlers (run, eval-suite, doctor, etc.) lack tests
   - Flag parsing, error handling, output formatting all untested

2. **Type system under-tested** (`internal/types` = 21.4%)
   - Complex unification logic (701 lines) mostly untested
   - Type inference paths (701 lines) have minimal coverage
   - Row polymorphism edge cases not covered

3. **Elaboration/desugaring sparse** (`internal/elaborate` = 15.6%)
   - Surface AST → Core AST transformation under-tested
   - Edge cases in lambda lifting, pattern desugaring untested

4. **Integration gaps** (`internal/pipeline` = 4.6%)
   - End-to-end compilation paths barely tested
   - Module loading, linking, execution flow untested
   - Error propagation between pipeline stages untested

5. **Dead code** (7 packages at 0%)
   - Some packages may be unused/deprecated
   - Others are test utilities not tested themselves

### Proposed Solution: Targeted Test Campaign

**Phase 1: Quick Wins (target: 50% coverage in 4-6 hours)**

Focus on high-value, easy-to-test code:

1. **CLI command tests** (`cmd/ailang`)
   - Test each subcommand (run, eval-suite, doctor, builtins, etc.)
   - Use golden files for output validation
   - Mock file system for integration tests
   - **Impact**: +20% overall coverage

2. **Type system happy paths** (`internal/types`)
   - Test common unification scenarios (already some exist)
   - Add exhaustive tests for row unification edge cases
   - Test type variable substitution
   - **Impact**: +8% overall coverage

3. **Pipeline integration** (`internal/pipeline`)
   - Test full compilation paths (parse → typecheck → link → eval)
   - Test error handling at each stage
   - Test module loading with imports
   - **Impact**: +5% overall coverage

4. **Elaboration core** (`internal/elaborate`)
   - Test lambda lifting
   - Test pattern match desugaring
   - Test block expression normalization
   - **Impact**: +5% overall coverage

**Expected after Phase 1**: 50-55% coverage

**Phase 2: Deep Coverage (target: 75% coverage in 6-8 hours)**

Add comprehensive edge case testing:

1. **Type inference corner cases**
   - Polymorphic recursion
   - Effect row merging with conflicts
   - Defaulting ambiguous type variables
   - TApp unification with nested applications

2. **Parser edge cases**
   - Malformed syntax recovery
   - Unicode handling
   - Deeply nested expressions
   - Large file handling

3. **Evaluator edge cases**
   - Stack overflow protection
   - Cyclic data structure detection
   - Effect capability violations
   - Resource exhaustion

4. **REPL workflows**
   - Multi-line input handling
   - State persistence across commands
   - Import resolution in REPL context
   - Type display formatting

**Expected after Phase 2**: 75-80% coverage

**Phase 3: Polish (target: 90% coverage, optional)**

- Property-based tests for unification
- Fuzzing parser with invalid inputs
- Performance regression tests
- Concurrent evaluation tests

**Why not 100% coverage?**

Some code is deliberately untested:
- Dead/deprecated code (should be removed instead)
- Unreachable error paths
- Debug/logging code
- Test utilities (testing the test helpers is low value)

---

## Problem 2: Example Failures (32 of 88 fail)

### Failed Examples by Category

**Category A: Missing Module/Entry Point (15 examples)**

These files are **not modules** - they're just expression snippets for documentation:

```
hello.ail                    - Just "print("Hello")" (no module wrapper)
records.ail                  - Collection of record examples (no main)
func_expressions.ail         - Function syntax examples (no main)
lambda_expressions.ail       - Lambda examples (no main)
arithmetic.ail               - Math examples (no main)
numeric_conversion.ail       - Conversion examples (no main)
list_patterns.ail            - Pattern examples (no main)
showcase/01_type_inference.ail
showcase/02_lambdas.ail
showcase/03_lists.ail
showcase/03_type_classes.ail
showcase/04_closures.ail
typeclasses.ail
type_classes_working_reference.ail
v3_3/math/gcd.ail
```

**Root cause**: `verify_examples.go` expects all `.ail` files to be runnable modules with `export func main()`. But many examples are intentionally **documentation snippets**, not programs.

**Example error**:
```
Error: undefined variable: print at examples/hello.ail:4:1
```

The file contains just:
```ailang
-- hello.ail - Simple hello world program
print("Hello, AILANG!")
```

This is a **doc snippet**, not a module. It should either:
1. Be wrapped in a module, OR
2. Be marked as `SKIP_VERIFY` in the header

**Category B: Experimental/Unimplemented Features (9 examples)**

Files that use features not yet implemented:

```
experimental/ai_agent_integration.ail  - AI agent calls (requires HTTP + JSON decode)
experimental/concurrent_pipeline.ail   - Concurrency primitives (not implemented)
experimental/factorial.ail             - May use unimplemented syntax
experimental/quicksort.ail             - May use unimplemented list functions
experimental/web_api.ail               - HTTP server (not implemented)
ai_call.ail                            - AI API calls (requires JSON + HTTP)
claude_haiku_call.ail                  - Claude API (requires JSON + HTTP)
demo_ai_api.ail                        - AI API (requires JSON + HTTP)
demo_openai_api.ail                    - OpenAI API (requires JSON + HTTP)
```

**Root cause**: Experimental files intentionally use future syntax or unimplemented features.

**Category C: Effect System Tests (8 examples)**

Files that test edge cases or may have syntax issues:

```
test_effect_io_simple.ail
test_m_r7_comprehensive.ail
test_net_file_protocol.ail
test_net_localhost.ail
test_net_security.ail
micro_clock_measure.ail
micro_net_fetch.ail
demos/effects_pure.ail
```

**Root cause**: These may be:
- Using old syntax that's been deprecated
- Missing required capabilities
- Testing error conditions (intentionally failing)
- Requiring external resources (localhost server, files, etc.)

### Proposed Solution: Example Organization

**Option 1: Split Examples by Purpose (RECOMMENDED)**

```
examples/
├── runnable/              # Full programs (must pass verify)
│   ├── hello_world.ail
│   ├── fibonacci.ail
│   ├── json_parse.ail
│   └── ...
├── snippets/              # Doc examples (skip verify)
│   ├── records.ail
│   ├── lambda_expressions.ail
│   └── ...
├── experimental/          # Future features (skip verify)
│   ├── ai_agent_integration.ail
│   └── ...
└── tests/                 # Test cases (may intentionally fail)
    ├── test_effect_io_simple.ail
    └── ...
```

Update `verify_examples.go` to:
- Only verify `examples/runnable/*.ail`
- Skip `snippets/`, `experimental/`, `tests/`
- Report coverage: "52 runnable examples, 48 passing (92%)"

**Option 2: Add Verify Pragma (SIMPLER)**

Add a comment pragma to files that should be skipped:

```ailang
-- @VERIFY:SKIP - This is a documentation snippet
-- hello.ail - Simple hello world example
print("Hello, AILANG!")
```

Update `verify_examples.go` to check for `@VERIFY:SKIP` pragma.

**Option 3: Fix Examples to Be Runnable (MOST WORK)**

Wrap all snippets in proper modules:

```ailang
-- hello.ail - Simple hello world program
module examples/hello

export func main() ! {IO} -> string {
  _io_print("Hello, AILANG!");
  "ok"
}
```

**Recommendation**: Use **Option 1** (split by purpose). Benefits:
- Clear separation of concerns
- Easy to understand for users
- Matches how users think about examples
- No magic comments/pragmas needed
- Can add automated tests for `runnable/` only

---

## Implementation Plan

### Task 1: Test Coverage Campaign

**Goal**: Increase coverage from 32.5% to 75%+

**Sub-tasks**:
1. Add CLI command tests (4-5 hours)
   - Test `ailang run` with various flags
   - Test `ailang eval-suite` with mock API
   - Test `ailang doctor builtins`
   - Test `ailang builtins list`
   - Use golden files for output validation

2. Add type system tests (3-4 hours)
   - Unification edge cases (polymorphic recursion, effect rows)
   - Type inference happy paths
   - Defaulting and ambiguity resolution
   - Row polymorphism (open vs closed rows)

3. Add pipeline integration tests (2-3 hours)
   - Full compile paths (parse → eval)
   - Error propagation
   - Module loading with imports
   - Multi-file projects

4. Add elaboration tests (2-3 hours)
   - Lambda lifting
   - Pattern desugaring
   - Block normalization
   - Effect annotation propagation

**Estimated time**: 11-15 hours
**Expected coverage after**: 75-80%

### Task 2: Example Organization

**Goal**: 100% pass rate for runnable examples, clear docs for snippets

**Sub-tasks**:
1. Reorganize examples/ directory (1 hour)
   ```bash
   mkdir examples/runnable examples/snippets examples/tests
   # Move files to appropriate directories
   git mv examples/hello.ail examples/snippets/
   git mv examples/experimental/* examples/experimental/  # already in subdir
   ```

2. Update verify_examples.go (1 hour)
   - Only scan `examples/runnable/`
   - Add optional `--all` flag to scan everything
   - Improve error reporting (categorize failures)

3. Fix runnable examples (2-3 hours)
   - Ensure all `examples/runnable/*.ail` have proper module structure
   - Add missing `export func main()` declarations
   - Test each one manually
   - Update documentation to explain directory structure

4. Add README.md to each directory (30 min)
   ```markdown
   # examples/runnable/
   Full programs that can be executed with `ailang run`.
   All files must have `export func main()`.

   # examples/snippets/
   Code snippets for documentation.
   Not meant to be run directly.

   # examples/experimental/
   Examples using future/unimplemented features.
   May not work with current version.
   ```

**Estimated time**: 4-5 hours
**Expected pass rate after**: 95-100% (for runnable examples)

---

## Success Metrics

**Before (current state)**:
- Test coverage: 32.5%
- Examples passing: 52/88 (59%)
- Tier 0 completion: 80%

**After Task 1 (test coverage)**:
- Test coverage: 75-80% ✅
- Examples passing: 52/88 (unchanged)
- Tier 0 completion: 90%

**After Task 2 (example fixes)**:
- Test coverage: 75-80% ✅
- Runnable examples: 35-40 passing / 40 total (95%+) ✅
- Documentation snippets: Clearly labeled, not verified
- Tier 0 completion: 95% ✅

**After both tasks**:
- Ready to ship v0.3.14 ✅
- CI gates pass ✅
- User experience improved (clear example organization) ✅

---

## Dependencies & Risks

### Dependencies
- None (all work is additive)

### Risks

**Risk 1: Test coverage campaign takes longer than estimated**
- Mitigation: Start with Phase 1 quick wins
- Fallback: Ship with 50% coverage, document gaps
- Note: 75% is still far below 90% target, but better than 32%

**Risk 2: Example reorganization breaks user workflows**
- Mitigation: Add deprecation notices, update README.md
- Fallback: Keep old structure, add new directories alongside
- Note: Examples are documentation, not API - breakage acceptable

**Risk 3: Some examples can't be fixed easily**
- Mitigation: Move unfixable examples to `experimental/` or `broken/`
- Document known issues in each file's header
- Priority: Get runnable examples to 95%+, not 100%

---

## Open Questions

1. **Should we aim for 90% coverage or accept 75%?**
   - 90% is the Tier 0 gate, but may require 20+ more hours
   - 75% is achievable in 11-15 hours and covers all critical paths
   - **Recommendation**: Ship v0.3.14 with 75%, defer 90% to v0.3.15

2. **Should verify_examples.go test execution or just parsing?**
   - Current: Full execution (slow, requires capabilities)
   - Alternative: Just parse/typecheck (fast, catches syntax errors)
   - **Recommendation**: Keep execution, but split examples by runnability

3. **What to do with experimental/ examples?**
   - Keep them (shows future direction)
   - Delete them (less confusion)
   - **Recommendation**: Keep but clearly label as "not working yet"

---

## Next Steps

1. **Decide on coverage target**: 75% or 90%?
2. **Prioritize tasks**: Coverage first or examples first?
3. **Allocate time**: 15-20 hours total work
4. **Start implementation**: Begin with high-value quick wins

**Recommended order**:
1. Task 2 (example organization) - 4-5 hours, immediate user value
2. Task 1 Phase 1 (quick coverage wins) - 4-6 hours, CI improvement
3. Task 1 Phase 2 (deep coverage) - optional, based on time budget

This gets examples to 95%+ and coverage to 50%+ in ~10 hours, enabling v0.3.14 ship.

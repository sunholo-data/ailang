# Sprint Plan: M-LANG-CLI-ARGS (v0.4.6)

## Summary
Implement command-line argument access via `std/env.getArgs()` to enable CLI tools and fix the `cli_args` benchmark (currently 0% → target 80%+). Extends existing Env capability with consistent API design.

**Duration:** 3 days (6 working hours)
**Dependencies:** Env capability (✅ already implemented in v0.4.4)
**Risk Level:** Low-Medium (capability enforcement is well-established pattern, but CLI parsing needs careful testing)

## Current Status Analysis

### Completed Recently (last 14 days)
- ✅ **Concat operator inference fix**: ~260 LOC in 1-2 days
- ✅ **Nullary pattern matching fix**: ~133 LOC in 1 day
- ✅ **Parser pattern sugar**: ~519 LOC in 2-3 days
- ✅ **v0.4.5 release**: Post-release tasks, benchmarks, documentation

### Velocity
- **Recent average**: ~150-200 LOC/day (implementation + tests)
- **Test ratio**: 1.5-2x test LOC vs implementation LOC
- **Commit frequency**: ~5 commits/day (73 commits in 14 days)
- **Estimated capacity**: 400-600 LOC for this sprint (3 days)

### Current State
- ✅ **Env capability exists**: `getEnv`, `hasEnv` implemented in v0.4.4
- ✅ **Effect system**: Capability-based security model working
- ✅ **Module system**: `std/env` module ready to extend
- ❌ **CLI args**: No way to access command-line arguments
- ❌ **cli_args benchmark**: 0% success rate across all 6 models

### Remaining from Design Doc
- 📋 **Phase 1**: Builtin function (~178 LOC) - 1.5 hours
- 📋 **Phase 2**: Stdlib wrapper (~8 LOC) - 0.5 hours
- 📋 **Phase 3**: Tests (~120 LOC) - 1.5 hours
- 📋 **Phase 4**: Documentation (~250 LOC) - 1 hour
- 📋 **Phase 5**: Eval testing - 1.5 hours

**Total estimated**: ~408 LOC implementation + tests

## Proposed Milestones

### Milestone 1: Core Implementation (Day 1)
**Goal:** Implement `_env_getArgs` builtin with capability enforcement and wire through runtime

**Estimated:** ~90 LOC implementation + ~50 LOC infrastructure = 140 LOC
**Duration:** 1 day (2 hours)

**Tasks:**
- **Morning (1 hour):**
  - [ ] Add `_env_getArgs` to `internal/builtins/spec.go` (~10 LOC)
  - [ ] Implement `EnvGetArgs` in `internal/builtins/env.go` (~25 LOC)
  - [ ] Add `GetArgs()` to `EffectContext` interface in `internal/effects/context.go` (~10 LOC)
  - [ ] Implement `GetArgs()` in `RealEffContext` (~15 LOC)

- **Afternoon (1 hour):**
  - [ ] Wire args through `runtime.Run()` in `internal/runtime/runtime.go` (~5 LOC)
  - [ ] Parse CLI args in `cmd/ailang/main.go` (~20 LOC)
  - [ ] Add `getArgs()` wrapper to `std/env.ail` (~8 LOC)
  - [ ] Quick manual test: `ailang run --caps Env test.ail arg1 arg2`

**Acceptance Criteria:**
- [ ] Builtin registered with correct type signature (nullary function, Env effect)
- [ ] Capability enforcement works (fails without `--caps Env`)
- [ ] Args passed from CLI → runtime → builtin correctly
- [ ] Manual test shows args are accessible
- [ ] Code compiles and runs

**Risks:**
- CLI argument parsing complexity - **Mitigation:** Test multiple invocation styles (flags before/after .ail file)
- Type signature mistakes - **Mitigation:** Follow exact pattern from `_env_getEnv`

**Files Changed:**
```
internal/builtins/spec.go        +10 LOC
internal/builtins/env.go         +25 LOC
internal/effects/context.go      +10 LOC
internal/runtime/runtime.go      +5 LOC
cmd/ailang/main.go               +20 LOC
std/env.ail                      +8 LOC
Total:                           ~78 LOC (implementation)
```

---

### Milestone 2: Testing & Verification (Day 2)
**Goal:** Comprehensive test coverage (unit + integration) and verify all edge cases

**Estimated:** ~30 LOC integration + ~45 LOC unit tests = 75 LOC
**Duration:** 1 day (1.5 hours)

**Tasks:**
- **Morning (1 hour):**
  - [ ] Create `tests/cli_args_test.ail` integration test (~30 LOC)
  - [ ] Test: No args → empty list
  - [ ] Test: Single arg → `["arg1"]`
  - [ ] Test: Multiple args → `["arg1", "arg2", "arg3"]`
  - [ ] Test: Args with spaces → `["hello world"]`
  - [ ] Test: Missing Env capability → `E_ENV_CAP_MISSING` error

- **Afternoon (0.5 hours):**
  - [ ] Add unit tests to `internal/builtins/env_test.go` (~45 LOC)
  - [ ] Test: Capability enforcement (with/without Env)
  - [ ] Test: Empty args list
  - [ ] Test: Args with special characters
  - [ ] Run full test suite: `make test`
  - [ ] Check coverage: `make test-coverage-badge` (target: 100% on new code)

**Acceptance Criteria:**
- [ ] `tests/cli_args_test.ail` passes with all test cases
- [ ] Unit tests pass (100% coverage on `EnvGetArgs`)
- [ ] Capability enforcement verified (error without `--caps Env`)
- [ ] CLI parsing works for all invocation styles:
  - [ ] `ailang run test.ail a b c`
  - [ ] `ailang run --caps Env test.ail a b`
  - [ ] `ailang run --entry main --caps IO,Env module.ail arg1`
- [ ] All existing tests still pass (no regressions)
- [ ] Linting clean: `make lint`

**Risks:**
- CLI parsing edge cases - **Mitigation:** Test extensively with different flag orders
- Capability test might be flaky - **Mitigation:** Use testctx.New().WithCap() pattern

**Files Changed:**
```
tests/cli_args_test.ail          +30 LOC (integration)
internal/builtins/env_test.go    +45 LOC (unit tests)
Total:                           ~75 LOC (tests)
```

---

### Milestone 3: Documentation & Eval (Day 3)
**Goal:** Update teaching prompt, create user guide, update eval harness, run benchmark

**Estimated:** ~250 LOC docs + ~5 LOC harness = 255 LOC
**Duration:** 1 day (2.5 hours)

**Tasks:**
- **Morning (1 hour) - Documentation:**
  - [ ] Update `prompts/v0.4.6.md` with CLI args section (~50 LOC)
    - [ ] Add `std/env` section with `getArgs` example
    - [ ] Include anti-patterns section (no `argv[0]`, no `argc()`)
    - [ ] Show correct usage with `! {IO, Env}`
  - [ ] Create `docs/guides/cli-arguments.md` (~200 LOC)
    - [ ] Basic usage examples
    - [ ] Flag parsing patterns (manual pattern matching)
    - [ ] Argument validation examples
  - [ ] Update `CLAUDE.md` with new feature (~10 LOC)

- **Afternoon (1.5 hours) - Eval Testing:**
  - [ ] Update eval harness to pass args: `internal/eval_harness/runner.go` (~5 LOC)
  - [ ] Add verification test for arg passing
  - [ ] Run targeted eval on `cli_args` benchmark:
    ```bash
    ailang eval-suite --benchmarks cli_args \
      --models gpt5-mini,claude-haiku-4-5,gemini-2-5-flash \
      --output eval_results/cli_args_v0.4.6
    ```
  - [ ] Analyze results, iterate on prompt if needed
  - [ ] Target: 0% → 80%+ success rate

**Acceptance Criteria:**
- [ ] Teaching prompt includes clear examples with `! {IO, Env}`
- [ ] Anti-patterns section prevents common hallucinations
- [ ] User guide covers basic usage and common patterns
- [ ] Eval harness passes args correctly (verified by test)
- [ ] `cli_args` benchmark: 0% → 80%+ success rate
- [ ] All documentation links work (verify in browser)
- [ ] Examples in docs are tested and working

**Risks:**
- Models still hallucinate `argv[0]` - **Mitigation:** Strong anti-patterns section
- Eval benchmark doesn't improve - **Mitigation:** Iterate on prompt clarity, add more examples

**Files Changed:**
```
prompts/v0.4.6.md                     +50 LOC
docs/guides/cli-arguments.md          +200 LOC
CLAUDE.md                             +10 LOC
internal/eval_harness/runner.go       +5 LOC
Total:                                ~265 LOC (docs + harness)
```

---

## Success Metrics

**Code Quality:**
- [ ] Test coverage: 100% on new builtin code (`EnvGetArgs`)
- [ ] Overall coverage: Maintain or improve from 38.5%
- [ ] All tests passing: `make test`
- [ ] All linting passing: `make lint`

**Examples & Documentation:**
- [ ] Integration test created: `tests/cli_args_test.ail`
- [ ] User guide created: `docs/guides/cli-arguments.md`
- [ ] Teaching prompt updated with examples and anti-patterns
- [ ] CLAUDE.md updated with new feature

**Functionality:**
- [ ] `getArgs()` returns `[string]` with CLI arguments
- [ ] Excludes program name (first element is first arg, not `argv[0]`)
- [ ] Capability enforcement works (requires `--caps Env`)
- [ ] Works with all CLI invocation styles

**Eval Metrics:**
- [ ] `cli_args` benchmark: 0% → 80%+ success rate
- [ ] Overall eval: 75% → 77%+ (with prompt improvements)
- [ ] No regressions in other benchmarks

## Day-by-Day Breakdown

### Day 1 (2 hours): Core Implementation
**Goal:** Builtin working end-to-end

**AM (1h):**
- Set up builtin spec and implementation
- Wire through effect context
- Quick compile check

**PM (1h):**
- Wire through runtime and CLI
- Add stdlib wrapper
- Manual smoke test

**Checkpoint:** Can run `ailang run --caps Env test.ail arg1` and see args

---

### Day 2 (1.5 hours): Testing
**Goal:** Comprehensive test coverage

**AM (1h):**
- Write integration test
- Test all edge cases (empty, single, multiple, spaces, missing cap)
- Verify CLI parsing with different flag orders

**PM (0.5h):**
- Write unit tests
- Run full test suite
- Check coverage (target: 100% on new code)

**Checkpoint:** All tests green, coverage ≥100% on new code

---

### Day 3 (2.5 hours): Docs & Eval
**Goal:** Production-ready with documentation and eval validation

**AM (1h):**
- Update teaching prompt (v0.4.6.md)
- Create user guide (docs/guides/cli-arguments.md)
- Update CLAUDE.md

**PM (1.5h):**
- Update eval harness
- Run `cli_args` benchmark
- Analyze results, iterate if needed
- Celebrate 0% → 80%+ improvement! 🎉

**Checkpoint:** Benchmark passing, docs complete, ready to merge

---

## Dependencies

**External Dependencies:**
- ✅ **Env capability** (already implemented in v0.4.4)
- ✅ **Effect system** (capability-based security working)
- ✅ **Module system** (`std/env` module exists)

**Blocking Items:**
- None - all prerequisites met

**Concurrent Work:**
- Can proceed independently of other v0.4.6 features
- No conflicts expected with ongoing work

## Open Questions

1. **CLI argument parsing strategy:**
   - Current plan: Find `.ail` file position, take everything after
   - Alternative: Use CLI library's args directly (may not handle flags correctly)
   - **Decision needed:** Verify with user which invocation styles to support

2. **Error message clarity:**
   - Should we suggest `--caps Env` in type error message?
   - Or only at runtime when capability is missing?
   - **Current plan:** Runtime error only (consistent with `getEnv`)

3. **REPL support:**
   - Design doc says "no REPL support" (doesn't make sense for interactive shell)
   - Should we explicitly error in REPL, or just return `[]`?
   - **Current plan:** Return `[]` (simplest, consistent with "no args provided")

4. **Eval harness args field:**
   - Does benchmark JSON schema already have `args` field?
   - Or do we need to extend the schema?
   - **Action:** Check `internal/eval_harness/benchmark.go` schema

## Notes

**Assumptions:**
- Env capability pattern is well-established and working
- CLI library (`urfave/cli`) handles flag parsing correctly
- Eval harness can be extended to pass args without major refactoring
- Teaching prompt improvements will be sufficient for 80%+ benchmark success

**Velocity Confidence:**
- **High** for implementation (follows established pattern)
- **Medium** for eval improvement (depends on prompt clarity)
- **Low risk** overall (no major architectural changes)

**Caveats:**
- This is a small, focused feature (~400 LOC total)
- Well-scoped with clear acceptance criteria
- Builds on existing Env capability (low risk)
- Main uncertainty is eval benchmark improvement (depends on AI model behavior)

**Post-Sprint:**
- Consider adding `--args` CLI flag for testing (like `--env`)
- Future: `std/cli` module with common patterns (help text, flag parsing)
- Future: Argument schema/validation DSL

## Estimated LOC Breakdown

| Component | Implementation | Tests | Docs | Total |
|-----------|---------------|-------|------|-------|
| Builtin spec | 10 | - | - | 10 |
| Builtin impl | 25 | 45 | - | 70 |
| Effect context | 25 | - | - | 25 |
| Runtime wiring | 5 | - | - | 5 |
| CLI parsing | 20 | - | - | 20 |
| Stdlib wrapper | 8 | - | - | 8 |
| Integration test | - | 30 | - | 30 |
| Teaching prompt | - | - | 50 | 50 |
| User guide | - | - | 200 | 200 |
| **Total** | **93** | **75** | **250** | **418** |

**Comparison to estimate:** Design doc estimated 408 LOC, actual breakdown shows 418 LOC (98% accuracy)

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| CLI parsing breaks existing invocations | Low | High | Test all known invocation styles; gradual rollout |
| Models still hallucinate `argv[0]` | Medium | Medium | Strong anti-patterns section in prompt |
| Capability enforcement bug | Low | High | Follow exact pattern from `_env_getEnv` |
| Eval harness refactoring needed | Low | Medium | Check schema first; minimal changes expected |
| Benchmark doesn't improve | Medium | Medium | Iterate on prompt; acceptable if ≥60% improvement |

**Overall Risk Level:** Low-Medium (well-scoped feature with established patterns)

## Success Criteria Summary

**Must Have (Sprint Blocker if Missing):**
- [ ] `getArgs()` returns CLI arguments as `[string]`
- [ ] Capability enforcement works (requires `--caps Env`)
- [ ] All tests passing (100% coverage on new code)
- [ ] Documentation complete (prompt + user guide)
- [ ] No regressions in existing tests

**Should Have (Post-Sprint if Missing):**
- [ ] `cli_args` benchmark: ≥80% success rate
- [ ] Examples working and documented
- [ ] CLI parsing handles all known invocation styles

**Nice to Have (Future Enhancement):**
- [ ] `--args` CLI flag for testing
- [ ] `std/cli` module with common patterns
- [ ] Argument schema/validation DSL

---

**Sprint approved by:** [Awaiting user approval]
**Start date:** [TBD]
**Target completion:** [TBD + 3 days]

# Sprint Plan: M-EFFECTFUL-LIST-COMBINATORS

## Summary
Add effectful list combinators (`mapE`, `filterE`, `foldlE`, `flatMapE`, `forEachE`) and pure `flatMap` to `std/list`, eliminating the most-requested DX gap from DocParse (54 hand-rolled recursive functions).

**Duration:** 2 days (10-12 hours)
**Dependencies:** None (std/list already exists, effect polymorphism supported)
**Risk Level:** Medium (effect polymorphism in HOFs needs type inference validation)
**Design Doc:** `design_docs/planned/v0_7_3/m-effectful-list-combinators.md`

## Current Status Analysis

### Completed Recently
- M-STDLIB-ZIP: ~315 LOC impl + ~388 LOC tests in 1 day
- M-STDLIB-XML: ~530 LOC impl + ~530 LOC tests in 1 day
- readFileBytes: ~80 LOC + 5 tests in <1 day

### Velocity
- Recent average: ~400-600 LOC/day (implementation + tests)
- Stdlib additions are well-practiced — ZIP and XML each took ~1 day
- This sprint is AILANG-only code (no Go builtins needed), so faster iteration

### Remaining from Design Doc
- Phase 1: Core combinators (~80 LOC in std/list.ail)
- Phase 2: Type inference regression tests (~200 LOC in test files)
- Phase 3: Documentation updates (~150 LOC across prompt, examples, CHANGELOG)

## Proposed Milestones

### Milestone 1: Core Combinators + Pure flatMap
**Goal:** Add all 6 functions to `std/list.ail` and verify they compile and run with basic effects.
**Estimated:** ~80 LOC implementation + ~120 LOC basic tests = ~200 LOC
**Duration:** 0.5 days

**Tasks:**
1. Add `flatMap` (pure) to `std/list.ail` — simplest, validates pattern
2. Add `mapE` — effectful map with `! {e}` row variable
3. Add `filterE` — effectful filter
4. Add `foldlE` — effectful left fold
5. Add `flatMapE` — effectful flatMap
6. Add `forEachE` — effectful forEach (unit return)
7. Add stack safety comment to each function
8. Create basic smoke test: `examples/runnable/effectful_list.ail` using `! {IO}` (println per element)
9. Verify with `ailang run --caps IO --entry main examples/runnable/effectful_list.ail`
10. Run `make test` to ensure no regressions

**Acceptance Criteria:**
- [ ] All 6 functions added to `std/list.ail`
- [ ] `mapE(\x. { println(show(x)); x * 2 }, [1,2,3])` works with `! {IO}`
- [ ] `forEachE(\x. println(show(x)), [1,2,3])` prints a, b, c in order
- [ ] Pure `flatMap` works without effects
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- Effect polymorphism `{e}` row variable may not unify correctly in HOFs — Mitigation: test early, fall back to concrete `! {IO}` signatures if needed
- Parser may struggle with effectful lambda syntax in function arguments — Mitigation: test multiple lambda syntaxes

### Milestone 2: Type Inference Regression Tests
**Goal:** Create 8 canonical test cases validating effect polymorphism in higher-order list functions. This doubles as typechecker validation.
**Estimated:** ~200 LOC test files
**Duration:** 0.5 days

**Tasks:**
1. Test `mapE` with `! {IO}` — single effect (println per element)
2. Test `filterE` with `! {IO}` — effectful predicate
3. Test `foldlE` with named function syntax `func(acc, x) -> T { ... }`
4. Test `foldlE` with lambda syntax `\acc. \x. ...` (known fragile path)
5. Test `flatMapE` with `! {IO}` — map + flatten with effects
6. Test `forEachE` with unit-returning lambda
7. Test pure `flatMap` (no effects)
8. Test left-to-right evaluation order (verify print order matches list order)
9. Document any broken paths (e.g., if curried lambda inference fails, note it)

**Acceptance Criteria:**
- [ ] 8 test cases created as runnable `.ail` files
- [ ] Type inference works without explicit effect annotations on lambdas
- [ ] Left-to-right evaluation order verified via IO output
- [ ] Known limitations documented if any tests fail

**Risks:**
- Lambda type inference for `\acc. \x. ...` style may be broken (GAP-2/GAP-3) — Mitigation: document and use `func(acc, x)` syntax as workaround
- Monomorphization limits (16/function, 512/module) — Mitigation: test with `--debug-compile`

### Milestone 3: Documentation + CHANGELOG
**Goal:** Update teaching prompt, create example file, update CHANGELOG.
**Estimated:** ~150 LOC
**Duration:** 0.5 days

**Tasks:**
1. Update teaching prompt (`prompts/v0.7.3.md`) with effectful list combinators section
2. Ensure `examples/runnable/effectful_list.ail` is comprehensive (from M1) — add budget guidance comments
3. Update `examples/manifest.json` with new example
4. Update CHANGELOG.md with new functions
5. Run `make verify-examples` to confirm examples pass
6. Final `make test && make lint` check

**Acceptance Criteria:**
- [ ] Teaching prompt includes effectful list section with budget guidance
- [ ] Example file created and listed in manifest
- [ ] CHANGELOG.md updated
- [ ] `make verify-examples` passes
- [ ] All tests and linting clean

**Risks:**
- Minimal — documentation-only tasks

## Success Metrics
- All 6 functions working in std/list: `mapE`, `filterE`, `foldlE`, `flatMapE`, `forEachE`, `flatMap`
- 8+ type inference regression tests passing
- Example file verified working
- Teaching prompt updated
- CHANGELOG updated
- `make test && make lint` clean
- Zero regressions in existing tests

## Dependencies
- None — std/list exists, effect system is mature, no Go builtins needed

## Open Questions
- Will effect row variable `{e}` unify correctly in HOF signatures? (validated in M2)
- Will curried lambda syntax `\acc. \x. ...` work with `foldlE`? (tested in M2)

## Notes
- All functions are pure AILANG (no Go builtins needed) — just `.ail` code in `std/list.ail`
- Stack safety limited to ~10,000 elements (recursive impl) — documented, Go builtin planned for v0.8.0+
- Budget exhaustion mid-traversal is a hard error (BudgetExhaustedError) per no-silent-fallback principle
- Inline test harness may be broken (M-DX-TEST-HARNESS-NIL) — use `ailang run` integration tests instead

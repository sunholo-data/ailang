# Sprint Plan: M-MOTOKO-INLINE-TEST-HARNESS

## Summary

Fix two bugs in `internal/testing/` that cause `ailang test` to fail on common patterns: (1) local helper functions that call imported stdlib functions, and (2) aliased imports when two modules export the same function name. Unblocks `make test_core` in `sunholo-data/motoko_agent`.

**Duration:** 1 day (~6 hours)
**Design Doc:** [design_docs/planned/v0_16_x/m-motoko-inline-test-harness.md](m-motoko-inline-test-harness.md)
**Target Version:** v0.16.3
**Dependencies:** None — pure bug fix in `internal/testing/`
**Risk Level:** Low (well-understood root cause, minimal LOC change, no API surface change)

---

## Current Status Analysis

### Completed Recently (last 7 days)
- ✅ M-EXTERNAL-CONSUMER-DX M3 — error_codes.json release artifact
- ✅ M-EXTERNAL-CONSUMER-DX M2 — effect-row error names
- ✅ M-EXTERNAL-CONSUMER-DX M1 — MOD013 module prefix overlap diagnostic
- ✅ v0.16.2 release

### Velocity
- Recent average: ~150-200 LOC/day on bug-fix milestones
- This sprint: ~90 LOC total (tiny; 6 hours estimated)
- Buffer: none needed — root cause fully diagnosed with working reproducers

### Remaining from Design Doc
- ⏳ M1: Bug 1 fix — cluster evaluation missing module imports (~10 LOC)
- ⏳ M2: Bug 2 fix — aliased import collision in CombinedResolver (~20 LOC)
- ⏳ M3: Tests + verification (~60 LOC tests)

---

## Proposed Milestones

### M1: Fix cluster evaluation missing module imports

**Goal:** `EvaluateInlineTestsWithCluster` gains access to module-imported functions by caching `result.Modules` in `ExtractPureClusterForFunction` and switching to `CombinedResolver`.

**Estimated:** ~10 LOC implementation
**Duration:** ~1.5 hours

**Tasks:**
1. In `internal/testing/executor.go`, `ExtractPureClusterForFunction` (line 387-388): add `e.modules = result.Modules` after `result, err := pipeline.Run(...)`
2. In `EvaluateInlineTestsWithCluster` (lines 349-352): replace:
   ```go
   resolver := runtime.NewBuiltinOnlyResolver(builtinRegistry)
   evaluator.SetGlobalResolver(resolver)
   ```
   with:
   ```go
   env := evaluator.Env()
   e.injectModuleBindings(evaluator, env)
   resolver := &CombinedResolver{
       Builtins: builtinRegistry,
       Env:      env,
       Modules:  e.modules,
   }
   evaluator.SetGlobalResolver(resolver)
   ```
3. Verify `e.injectADTConstructors(evaluator)` call remains (line 353 — already present)
4. Run `ailang test /tmp/test_import_in_helper.ail` — confirm passes
5. Run `make test` — confirm no regression

**Acceptance Criteria:**
- [ ] `ailang test /tmp/test_import_in_helper.ail` → all 1 tests pass
- [ ] `ailang test /tmp/test_local_helper.ail` (5-test dirname reproducer) → all 5 pass
- [ ] `make test ./internal/testing/...` all green

---

### M2: Fix aliased import collision in CombinedResolver

**Goal:** `CombinedResolver` correctly resolves module-qualified references by using module-qualified keys in the env, preventing same-named functions from different modules from overwriting each other.

**Estimated:** ~20 LOC implementation
**Duration:** ~2 hours

**Tasks:**
1. In `internal/testing/executor_helpers.go`, update `PendingLambdaBinding` struct to include `modulePath string`
2. In `injectModuleBindings` Pass 1, capture the module path alongside the lambda:
   ```go
   for modulePath, mod := range e.modules {
       // ...
       pendingLambdas = append(pendingLambdas, PendingLambdaBinding{
           name:       d.Name,
           lambda:     lambda,
           modulePath: modulePath,
       })
   }
   ```
3. In `injectModuleBindings` Pass 2, after `env.Set(pending.name, funcVal)`, also set:
   ```go
   if pending.modulePath != "" {
       env.Set(pending.modulePath+"."+pending.name, funcVal)
   }
   ```
4. In `CombinedResolver.ResolveValue` Case 2 (module-qualified reference), before `r.Env.Get(ref.Name)`, try:
   ```go
   qualifiedKey := ref.Module + "." + ref.Name
   if val, ok := r.Env.Get(qualifiedKey); ok {
       return val, nil
   }
   ```
5. Run `ailang test /tmp/test_alias_collision.ail` — confirm passes
6. Run `make test` — confirm no regression

**Acceptance Criteria:**
- [ ] `ailang test /tmp/test_alias_collision.ail` → all 1 tests pass
- [ ] `make test ./internal/testing/...` all green

---

### M3: Tests, cross-repo verification, CHANGELOG

**Goal:** Two new Go tests to lock in both fixes; cross-repo verification against motoko_agent; CHANGELOG entry.

**Estimated:** ~60 LOC tests + docs
**Duration:** ~2.5 hours

**Tasks:**
1. Add `TestClusterEvalWithImportedHelper` in `internal/testing/executor_test.go`:
   - Write the `test_import_in_helper.ail` minimal reproducer as a temp file test
   - Call `RunTestsFromFile` on it
   - Assert 0 failures, 1 pass
2. Add `TestAliasImportCollision` in `internal/testing/executor_test.go`:
   - Write the `test_alias_collision.ail` minimal reproducer as a temp file test
   - Call `RunTestsFromFile` on it
   - Assert 0 failures, 1 pass
3. Run `make test ./internal/testing/...` — all green
4. Run `make verify-examples` — no regression on existing examples
5. Cross-repo: `cd /Users/mark/dev/sunholo/motoko_agent && make test_core` — 0 harness failures
6. Add CHANGELOG entry under v0.16.3:
   ```markdown
   ### Bug Fixes
   - **fix(testing): inline tests for local helpers using stdlib imports** — `ailang test` no longer fails with `cannot apply non-function value: <nil>` when a function under test calls a local helper that uses imported functions. Root cause: `EvaluateInlineTestsWithCluster` used `BuiltinOnlyResolver` and `ExtractPureClusterForFunction` did not cache loaded modules. (#M-MOTOKO-INLINE-TEST-HARNESS M1)
   - **fix(testing): aliased import collision in CombinedResolver** — `ailang test` no longer fails with `_list_length: expected List, got *eval.StringValue` when a module imports the same function name from two stdlib packages with an alias. Root cause: flat env injection allowed last-loaded module to overwrite earlier bindings; CombinedResolver now prefers module-qualified env keys. (#M-MOTOKO-INLINE-TEST-HARNESS M2)
   ```
7. Move design doc to `design_docs/implemented/v0_16_3/m-motoko-inline-test-harness.md`
8. Run `.claude/skills/sprint-executor/scripts/milestone_checkpoint.sh M3_CHANGELOG_DOCS`

**Acceptance Criteria:**
- [ ] `TestClusterEvalWithImportedHelper` passes in `go test ./internal/testing/...`
- [ ] `TestAliasImportCollision` passes in `go test ./internal/testing/...`
- [ ] `make test_core` in `sunholo-data/motoko_agent` → `All core runtime module tests passed!` (0 harness failures)
- [ ] `make test` full suite passes
- [ ] CHANGELOG.md updated under v0.16.3
- [ ] Design doc moved to `design_docs/implemented/v0_16_3/`

---

## Success Metrics

- **All tests:** `make test` green
- **Inline test regression:** `make verify-examples` green
- **Cross-repo:** `make test_core` in motoko_agent — `0 harness failures`
- **New tests:** 2 regression tests added to `internal/testing/executor_test.go`
- **LOC change:** ~90 LOC total (10 impl + 20 impl + 60 tests)

---

## Files to Modify

| File | Change | ΔLoC |
|------|--------|------|
| `internal/testing/executor.go` | M1: add module caching + switch to CombinedResolver in cluster path | +10 |
| `internal/testing/executor_helpers.go` | M2: module-qualified env keys in `PendingLambdaBinding` + `injectModuleBindings` + `CombinedResolver` | +20 |
| `internal/testing/executor_test.go` | M3: two new regression tests | +60 |
| `changelogs/v0.10-current.md` | M3: CHANGELOG entry | +10 |

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Module-qualified key format mismatch | Low | Use same format as `loader.LoadedModule` map keys (verify with `fmt.Println(range e.modules)` in test) |
| Backward compat: existing tests rely on flat-key env | Low | Flat keys still set; qualified keys added additionally |
| motoko_agent AILANG version mismatch | Low | motoko is on current dev branch; run `ailang --version` in motoko to confirm |

---

## Open Questions

None — root cause confirmed with reproducers, fix approach approved in design doc.

---

**Sprint created:** 2026-05-06
**Estimated duration:** 1 day (~6 hours)
**Sprint ID:** M-MOTOKO-INLINE-TEST-HARNESS

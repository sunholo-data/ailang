# M-MOTOKO-INLINE-TEST-HARNESS: Fix inline test harness for import-dependent helpers and aliased import collisions

**Status**: Planned
**Target**: v0.16.3
**Priority**: P1 (High — blocks motoko_agent `make test_core`; affects any module that imports two stdlib packages with overlapping function names)
**Estimated**: 1 day (~6 hours)
**Dependencies**: None
**Author**: Claude + Mark
**Created**: 2026-05-06

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Bug fix; no semantic change |
| A2: Replayability | +1 | Inline tests produce more traces (previously failed silently) |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | More functions can be verified locally via `ailang test` |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Agents (motoko) can rely on inline tests as self-checking contracts |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | 0 | No composition changes |
| A11: Structured Failure | +1 | Errors now surface meaningful test failures rather than `<nil>` application errors |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine-readable test contracts

---

## Problem Statement

The AILANG inline-test harness (`ailang test <file.ail>`) fails for two related but distinct patterns that are common in production AILANG modules. Both failures produce unhelpful runtime errors that obscure the actual root cause.

**Bug 1: Cluster evaluation missing module imports**

When a function under test calls a local helper that itself calls an imported stdlib function, the cluster harness evaluates with a `BuiltinOnlyResolver` that cannot resolve module-imported functions.

Error: `cluster harness evaluation failed: cannot apply non-function value: <nil>`

**Bug 2: Import alias collision**

When a module imports the same function name from two different modules (one with an alias), the flat environment injection overwrites the earlier binding, causing the wrong function to be called.

Error: `harness evaluation failed: _list_length: expected List, got *eval.StringValue`

**Real-world impact (motoko_agent):**

`make test_core` in `sunholo-data/motoko_agent` reports failures across 5+ modules:
- `src/core/agents_md.ail` — `dirname` (Bug 1), `is_root` (Bug 2)
- `src/core/context_usage.ail` — Bug 1
- `src/core/compaction.ail` — Bug 1
- `src/core/ext/runtime.ail` — Bug 1
- `src/core/config.ail` — Bug 1

**Production code is correct** — `ailang check` passes for all modules. Failures are isolated to the test runner.

---

## Root Cause Analysis

### Bug 1: `BuiltinOnlyResolver` in cluster evaluation path

In `runner.go:76-95`, when a function has cross-function dependencies:
1. `ExtractPureClusterForFunction` (executor.go:368-422) runs the module pipeline and gets `result.Modules`, but **never caches them** into `e.modules`
2. `EvaluateInlineTestsWithCluster` (executor.go:351) creates `runtime.NewBuiltinOnlyResolver(builtinRegistry)` — this resolves only AILANG builtins (`_substring`, `_list_filter`, etc.), not imported module functions

When `find_char` (a local helper) calls `substring` from `std/string`, the resolver sees a `VarGlobal{module:"std/string", name:"substring"}` reference and returns nil — hence `cannot apply non-function value: <nil>`.

Contrast with `EvaluateInlineTestsWithHarness` (executor.go:97-143) which correctly uses a `CombinedResolver` + `injectModuleBindings`. The cluster path was implemented without porting this fix.

### Bug 2: Flat env injection overwrites aliased imports

`injectModuleBindings` injects all module functions into a flat `env` keyed by bare function name. When `std/string.length` and `std/list.length` are both loaded, the last-injected one wins in the env under the key `"length"`.

`CombinedResolver.ResolveValue` Case 2 (module-qualified references like `{module:"std/string", name:"length"}`) correctly identifies the right module, but then calls `r.Env.Get("length")` — returning whichever `length` was injected last, regardless of the module qualifier.

---

## Goals

**Primary Goal:** Fix both bugs so `ailang test` correctly evaluates inline tests for all pure functions, regardless of local helper chains or aliased imports.

**Success Metrics:**
1. `make test_core` in `sunholo-data/motoko_agent` passes with 0 cluster harness failures
2. Minimal reproducer for Bug 1 (local helper calling imported function) passes
3. Minimal reproducer for Bug 2 (aliased import collision) passes
4. `make test` in `ailang` repo passes (no regression)
5. All existing inline tests that pass today continue to pass

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| How to fix alias collision in CombinedResolver | Two valid approaches: module-qualified env keys vs direct module lookup; choice affects env size and resolver complexity | agent | design | low |
| Whether to fix Bug 1 by changing cluster path or unifying it with harness path | Unified path = simpler long-term; split path = smaller diff | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Bug 1 fix approach: add module caching to `ExtractPureClusterForFunction` + switch `EvaluateInlineTestsWithCluster` to `CombinedResolver` (mirrors the existing harness path)
- [x] Bug 2 fix approach: in `CombinedResolver.ResolveValue` Case 2, use `r.Env.Get(ref.Module + "." + ref.Name)` and store with module-qualified key in `injectModuleBindings`

---

## Solution Design

### Overview

Two targeted fixes in `internal/testing/`, each under 15 LOC:

**Fix 1 (Bug 1, two sub-changes):**
- `ExtractPureClusterForFunction`: add `e.modules = result.Modules` after pipeline run
- `EvaluateInlineTestsWithCluster`: replace `BuiltinOnlyResolver` with `CombinedResolver` + `injectModuleBindings` call (exact same pattern as `EvaluateInlineTestsWithHarness`)

**Fix 2 (Bug 2, two sub-changes):**
- `injectModuleBindings`: inject lambdas under both `"name"` (backward compat) and `"module/path.name"` qualified key
- `CombinedResolver.ResolveValue` Case 2: look up `ref.Module + "." + ref.Name` in env before falling back to bare `ref.Name`

### Architecture

The two bugs share a common theme: the cluster evaluation path was not updated when the harness path gained module injection support (M-DX25). This is the pattern fix, not a case-by-case patch.

**Execution paths for inline tests:**
```
ailang test file.ail
  → runner.runTest(testCase)
      → ExtractPureClusterForFunction()  ← Bug 1: no module caching
          → if cluster.HasDependencies():
              → EvaluateInlineTestsWithCluster()  ← Bug 1: BuiltinOnlyResolver
          → else:
              → EvaluateInlineTestsWithHarness()  ← Already works (CombinedResolver)
                  → CombinedResolver.ResolveValue()  ← Bug 2: flat name lookup
```

### Implementation Plan

**Phase 1: Bug 1 fix** (~2 hours)

- [ ] In `ExtractPureClusterForFunction` (executor.go:387-388), add `e.modules = result.Modules` after `result, err := pipeline.Run(...)`
- [ ] In `EvaluateInlineTestsWithCluster` (executor.go:349-352), replace:
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
- [ ] Add `e.injectADTConstructors(evaluator)` call (already present; verify it stays)
- [ ] Add `TestClusterEvalWithImportedHelper` test in `executor_test.go`

**Phase 2: Bug 2 fix** (~2 hours)

- [ ] In `injectModuleBindings` (executor_helpers.go:286-347), in Pass 2 where FunctionValues are set, also bind under the module-qualified key:
  ```go
  env.Set(pending.name, funcVal)
  if pending.modulePath != "" {
      env.Set(pending.modulePath + "." + pending.name, funcVal)
  }
  ```
  This requires threading the module path through `PendingLambdaBinding`.
- [ ] In `CombinedResolver.ResolveValue` Case 2 (executor_helpers.go:41-70), prefer the qualified lookup:
  ```go
  qualifiedKey := ref.Module + "." + ref.Name
  if val, ok := r.Env.Get(qualifiedKey); ok {
      return val, nil
  }
  // fall back to bare name (backward compat)
  if val, ok := r.Env.Get(ref.Name); ok {
      return val, nil
  }
  ```
- [ ] Add `TestAliasImportCollision` test in `executor_test.go`

**Phase 3: Verification** (~2 hours)

- [ ] Run `make test` in ailang repo — all green
- [ ] Run `ailang test /tmp/test_import_in_helper.ail` — passes
- [ ] Run `ailang test /tmp/test_alias_collision.ail` — passes
- [ ] Run `make test_core` in motoko_agent — 0 harness failures

### Files to Modify

**Modified files:**

| File | Changes | ΔLoC |
|------|---------|------|
| `internal/testing/executor.go` | Add module caching in `ExtractPureClusterForFunction`; switch `EvaluateInlineTestsWithCluster` to `CombinedResolver` | +10 |
| `internal/testing/executor_helpers.go` | Thread module path in `PendingLambdaBinding`; dual-key binding in Pass 2; qualified lookup in `CombinedResolver` | +20 |
| `internal/testing/executor_test.go` | Two new test functions for the minimal reproducers | +60 |

Total: ~90 LOC change, ~60 new test LOC.

---

## Examples

### Bug 1 — Before (fails) / After (passes)

```ailang
module test_import_in_helper

import std/string (substring)

func find_char(i: int, s: string) -> int =
  if substring(s, i, i + 1) == "/" then i else find_char(i - 1, s)

func uses_it(s: string) -> int
  tests [("/home/user", 5)]
  { find_char(10, s) }
```

**Before:** `cluster harness evaluation failed: cannot apply non-function value: <nil>`  
**After:** `✓ uses_it_test_1 — 1 passed`

### Bug 2 — Before (fails) / After (passes)

```ailang
module test_alias_collision

import std/string (length as str_len)
import std/list (length)

func check_str_len(s: string) -> int
  tests [("hello", 5)]
  { str_len(s) }
```

**Before:** `harness evaluation failed: _list_length: expected List, got *eval.StringValue`  
**After:** `✓ check_str_len_test_1 — 1 passed`

### Real-world fix (motoko_agent)

```
$ cd motoko_agent && make test_core
Running src/core/agents_md.ail tests...
✓ dirname_test_1 ... dirname_test_5  [5 passed]
✓ is_root_test_1 ... is_root_test_6  [6 passed]
...
All core runtime module tests passed!
```

---

## Success Criteria

- [ ] `ailang test /tmp/test_import_in_helper.ail` → all tests pass (Bug 1 reproducer)
- [ ] `ailang test /tmp/test_alias_collision.ail` → all tests pass (Bug 2 reproducer)
- [ ] `make test_core` in `sunholo-data/motoko_agent` → `0 harness failures`
- [ ] `make test` in ailang repo → all green (no regression)
- [ ] `TestClusterEvalWithImportedHelper` and `TestAliasImportCollision` added to `executor_test.go`
- [ ] CHANGELOG.md updated under v0.16.3

---

## Testing Strategy

**Unit tests (executor_test.go):**
- `TestClusterEvalWithImportedHelper`: write the minimal reproducer as an in-process test using a temp file; assert it passes
- `TestAliasImportCollision`: write the alias collision reproducer as an in-process test; assert it passes
- Verify all existing `executor_test.go` tests still pass

**Integration tests:**
- `make verify-examples` — all example inline tests still pass
- `ailang test examples/factorial.ail` — baseline smoke test

**Cross-repo verification:**
- `make test_core` in `sunholo-data/motoko_agent` — the primary consumer that reported both bugs

---

## Deferred Decisions

- Whether to remove the now-redundant `EvaluateInlineTestsWithCluster` function and unify fully with `EvaluateInlineTestsWithHarness` — agent may choose; the minimal fix keeps both paths to reduce diff size
- Whether to backport `TestAliasCollision` to also cover triple-import collisions (three modules with same function name) — agent may choose

---

## Non-Goals

- Changing inline test syntax — no syntax changes
- Supporting `tests [...]` on expression-style (`=`) functions — separate issue (M-DX23 scope)
- Supporting file-level `test "name" { }` blocks — separate issue
- Cross-module inline tests — functions under test must be in the file being tested

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Module-qualified env keys cause memory bloat | Low | Only pure function lambdas are injected; stdlib has ~200 functions max |
| Backward compat: code relying on `BuiltinOnlyResolver` behavior | Low | Only the cluster path used it; single-binding path already uses CombinedResolver |
| New module caching in cluster path causes stale data between tests | Low | `e.modules` is reset on each `ExtractFunctionBinding` / `ExtractPureClusterForFunction` call |

---

## Related Documents

**Implemented (directly relevant):**
- [design_docs/implemented/v0_4_7/m-testing-inline-core-evaluation.md](../../../implemented/v0_4_7/m-testing-inline-core-evaluation.md) — original inline test harness build
- [design_docs/implemented/v0_7_4/m-dx25-inline-tests-import-resolution.md](../../../implemented/v0_7_4/m-dx25-inline-tests-import-resolution.md) — prior import resolution fix (M-DX25) that introduced `CombinedResolver`; cluster path was not updated at the same time
- [design_docs/implemented/v0_9_2/m-bug-inline-test-extraction.md](../../../implemented/v0_9_2/m-bug-inline-test-extraction.md) — module-less file and absolute path fix

**Planned (context):**
- [design_docs/planned/v0_13_0/m-dx23-inline-tests-documentation.md](../../v0_13_0/m-dx23-inline-tests-documentation.md) — documentation for inline tests (depends on harness being correct)
- [design_docs/planned/v0_17_0/m-bench-motoko-executor.md](../../v0_17_0/m-bench-motoko-executor.md) — motoko as benchmark executor (requires `make test_core` passing)

**Motoko-side design doc:**
- `sunholo-data/motoko_agent` — `design_docs/planned/m-motoko-inline-test-harness.md` (identifies symptom; this doc is the AILANG-side fix)

---

**Document created**: 2026-05-06
**Last updated**: 2026-05-06

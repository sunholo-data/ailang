# M-DX-SERVE-API-COERCION: Fix serve-api type coercion in cross-package function calls

**Status**: Planned (blocked — cannot reproduce locally)
**Target**: v0.11.0
**Priority**: P1 (blocks serve-api deployments with cross-package deps)
**Estimated**: 2-3 days
**Dependencies**: None
**Milestone ID**: M-DX-SERVE-API-COERCION
**Created**: 2026-04-02
**Source**: DocParse agent messages (multiple threads, 2026-04-02)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Bug is non-deterministic (works in ailang run, fails in serve-api) — fix restores determinism |
| A2: Replayability | +1 | Consistent function results regardless of execution path |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Type-correct function returns enable local verification |
| A6: Safe Concurrency | +1 | Evaluator Fork correctness affects concurrent request handling |
| A7: Machines First | +1 | AI agents building serve-api integrations hit this silently |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Function composition breaks when inner functions return wrong types |
| A11: Structured Failure | +1 | concat_String error is misleading — real problem is upstream |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): This IS a determinism fix — same code gives different results in run vs serve-api
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Improves machine predictability

## Problem Statement

Functions like `intToStr(x)` and `show(x)` return raw `IntValue` instead of `StringValue` when:
1. Running through `serve-api` (works correctly with `ailang run`)
2. Handler imports packages with diamond dependency patterns (e.g., `billing_store → firestore/fields`)
3. The transitive dependency also calls the same function (`show()` / string operations)

The immediate symptom: `concat_String: arg 1 - expected string, got int` — the function call passes through the raw value instead of applying the conversion.

**Current State:**
- Confirmed by docparse in production with `billing_service_api@0.5.4`
- Cannot reproduce locally even with same binary (5e61986d) and same package
- `CallPreserveFloats → Call` fix resolved the float variant but int variant persists in production
- `DEBUG_EVAL_APP=1` tracing shows correct resolution locally

**Impact:**
- Blocks production billing serve-api deployment
- Affects any serve-api deployment with cross-package diamond dependencies
- Error is deeply confusing: appears as `concat_String` failure, actual cause is upstream

## Goals

**Primary Goal:** Ensure function application returns correct results in serve-api regardless of package dependency structure.

**Success Metrics:**
- `billing_service_api@0.5.4` entitlements endpoint returns all string values correctly in serve-api
- Zero `concat_String` type mismatch errors in production
- Regression test that exercises the serve-api Call path with cross-package diamond deps
- `DEBUG_EVAL_APP` tracing available as `--debug-eval` CLI flag for serve-api

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Whether evaluator Fork copies module bindings or creates fresh env | Controls whether closures captured during module eval survive Fork | agent | design | high |
| Whether shared evaluator state (resolver, env) gets modified during concurrent module loads | Race condition if module loading mutates shared state | agent | design | high |
| Whether to add integration test for serve-api Call path | Prevents regression but requires test infrastructure | agent | compile | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Reproduce the bug — need exact environment delta vs local (docparse binary hash check pending)
- [ ] Confirm whether bug is in evaluator Fork, module resolver, or compilation context

## Solution Design

### Overview

Three parallel investigation tracks, then targeted fix:

### Architecture

**Track A: Reproduce the Bug**
The bug cannot be fixed without reproduction. Possible causes of local non-reproduction:
1. Different binary (docparse may be running stale binary)
2. Different Firestore data (real vs fallback path)
3. Race condition that only manifests under specific timing
4. Module cache state that differs between environments

**Track B: Audit Evaluator Fork for State Leakage**
The serve-api `CallEntrypoint` path:
1. `rt.evaluator.Fork()` — creates fresh env with builtins, copies effContext
2. `newModuleGlobalResolver(inst, rt)` — resolver bound to entry module
3. `reqEval.CallFunction(fn, args)` — evaluates function on forked evaluator

Potential issues:
- `Fork()` creates `NewEnvironment()` + `registerBuiltins()` — but closures captured during module eval reference the ORIGINAL evaluator's env via `fn.Env`. If `fn.Env`'s parent chain was corrupted during module loading, the Fork inherits corrupted closures.
- `evaluateModule()` sets `rt.evaluator.SetGlobalResolver(resolver)` per module — if modules load concurrently, this is a race.
- `LetRec` phase 2.5 propagates bindings to parent env — if two modules share the same parent, last-write-wins.

**Track C: Audit the `concat_String` Call Chain**
`concat_String` is a pure builtin `(string, string) -> string`. It receives args from the `++` operator lowering. If `intToStr(x)` returned `IntValue` instead of `StringValue`, then the expression `"prefix" ++ intToStr(x)` would pass `IntValue` to `concat_String`.

The question: is `intToStr` being called and returning wrong result, or is the call being bypassed entirely?

### Implementation Plan

**Phase 1: Reproduce** (~2 hours)
- [ ] Get docparse's binary hash and compare
- [ ] Add `--debug-eval` CLI flag to serve-api (always-available tracing to file)
- [ ] If same binary: test with real Firestore data (not fallback path)
- [ ] If different binary: rebuild and retest in docparse's environment
- [ ] Create minimal reproduction test case that runs in CI

**Phase 2: Root Cause** (~4 hours)
- [ ] Instrument `evalCoreApp` to log when a FunctionValue's body evaluates to a value whose type doesn't match expected return type
- [ ] Audit `evaluateModule` for shared state mutation (resolver, env parent chain)
- [ ] Check if `LetRec` phase 2.5 parent propagation causes cross-module name collisions
- [ ] Verify `FallbackResolver` chain is correct when calling cross-package functions from serve-api
- [ ] Test with `DEBUG_CONCURRENCY=1` to check for race conditions

**Phase 3: Fix & Test** (~4 hours)
- [ ] Apply targeted fix based on root cause
- [ ] Write integration test: serve-api with diamond deps calling intToStr
- [ ] Run full billing_service_api@0.5.4 through serve-api in CI
- [ ] Remove temporary debug logging or consolidate behind `--debug-eval`

### Files to Modify/Create

**Key files to audit:**
- `internal/runtime/runtime.go` — `evaluateModule()`, `extractBindings()` shared evaluator state
- `internal/runtime/entrypoint.go` — `CallEntrypoint()` Fork + resolver setup
- `internal/runtime/resolver.go` — `moduleGlobalResolver` resolution chain
- `internal/eval/eval_evaluator.go` — `Fork()` state copying
- `internal/eval/eval_operations.go` — `evalCoreApp()` function dispatch
- `internal/eval/eval_expressions.go` — `evalCoreLetRec()` binding propagation

**Likely fix locations (hypotheses):**
- `runtime.go:265-268` — Module env isolation may not be sufficient
- `runtime.go:312-317` — LetRec parent propagation may cause cross-module collisions
- `entrypoint.go:103-107` — Fork may need to carry module bindings, not start fresh

**New files:**
- `internal/apiserver/serve_api_coercion_test.go` — Integration test (~100 LOC)

## Examples

### Example 1: The Bug

**AILANG code (entitlements_handler.ail):**
```ailang
import std/string (intToStr)
import std/json (encode, jo, js, kv)
import pkg/sunholo/billing_store/entitlements_repo (getEntitlements)
import pkg/sunholo/billing_store/usage_repo (getUsage)

@route("GET", "/billing/me/entitlements")
export func handleGetEntitlements(principalId: string, period: string) -> Result[string, string] ! {Net, FS, Env} =
  let ent = ... in
  Ok(encode(jo([kv("maxFileSizeMb", js(intToStr(ent.maxFileSizeMb)))])))
```

**Expected (ailang run):**
```json
{"maxFileSizeMb": "10"}
```

**Actual (serve-api, intermittent):**
```
concat_String: arg 1 - expected string, got int. Use intToStr() to convert int to string
```

### Example 2: The Diamond Dependency Pattern

```
entitlements_handler.ail
  ├── pkg/billing_store/entitlements_repo
  │     └── pkg/firestore/fields (uses show(value) in intVal)
  ├── pkg/billing_store/usage_repo        ← adding this triggers the bug
  │     ├── pkg/firestore/fields          ← diamond: shared dep
  │     └── pkg/billing_entitlements/usage_policy
  └── pkg/billing_entitlements/plan
        └── pkg/billing_entitlements/quota_policy
```

## Success Criteria

- [ ] `billing_service_api@0.5.4` /billing/me/entitlements returns correct JSON in serve-api
- [ ] No `concat_String` type mismatch errors with diamond dependency patterns
- [ ] Integration test in CI that exercises serve-api + cross-package intToStr
- [ ] `--debug-eval` flag available for serve-api troubleshooting
- [ ] All existing tests passing
- [ ] Design doc moved to implemented/

## Testing Strategy

**Unit tests:**
- `embed_test.go` — FromGo/FromGoPreserveFloats coercion (done)
- `safe_cast_test.go` — Error message format with AILANG types (done)

**Integration tests:**
- New: serve-api handler calling intToStr through cross-package dependency
- New: serve-api handler with diamond dependency pattern (two packages sharing a transitive dep)

**Manual testing:**
- Run `billing_service_api@0.5.4` through serve-api
- Hit /billing/me/entitlements with real Firestore data (not fallback)

## Deferred Decisions

- Exact debug tracing format for `--debug-eval` — agent may choose
- Whether to add trace-level logging to resolver chain — agent may choose
- Whether `evaluateModule` needs locking for concurrent module loads — agent decides based on analysis

## Non-Goals

- Changing the type class / dictionary-passing system — too large
- Adding polymorphic dispatch for `show()` — separate feature
- Fixing `println` output (separate bug reported by docparse)
- List pattern matching (separate feature request)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Cannot reproduce the bug | High | Get docparse's binary hash; add --debug-eval flag; instrument production |
| Race condition in module loading | High | Add DEBUG_CONCURRENCY tracing; test under concurrent load |
| Fix breaks other serve-api users | Med | Comprehensive test suite; feature flag for new behavior |
| Root cause is in compilation, not evaluation | Med | Add DEBUG_CODEGEN tracing; compare Core AST between run and serve-api |

## Related Documents

**Prior investigation (superseded by this doc):**
- [m-investigate-xpkg-int-float-coercion.md](m-investigate-xpkg-int-float-coercion.md) — Initial investigation, partial fixes applied

**Related serve-api docs:**
- [m-serve-api-agent-enhancements.md](../../implemented/v0_10_0/m-serve-api-agent-enhancements.md)
- [m-hot-reload-serve-api.md](../../implemented/v0_7_1/m-hot-reload-serve-api.md)
- [m-serve-api-get-args.md](../../implemented/v0_10_0/m-serve-api-get-args.md)

**Related type coercion:**
- [m-dx-json-bool-coercion.md](m-dx-json-bool-coercion.md) — Similar JSON type issue
- M-DX-XPKG-RESOLVE (v0.9.11) — Prior cross-package resolver fix

## Future Work

- Polymorphic `show()` dispatch via dictionary-passing (currently monomorphic builtin)
- Serve-api compilation context sharing across hot-reloads
- Module loading concurrency safety audit

---

**Document created**: 2026-04-02
**Last updated**: 2026-04-02

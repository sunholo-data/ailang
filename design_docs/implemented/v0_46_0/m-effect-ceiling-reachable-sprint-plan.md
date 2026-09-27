# Sprint Plan: M-EFFECT-CEILING-REACHABLE

**Design doc**: [m-effect-ceiling-reachable.md](m-effect-ceiling-reachable.md)
**Duration**: 1 day (single session) · **Risk**: medium (security-relevant semantics) · **Est. LOC**: ~300 impl + ~350 tests

No sprint JSON / executor handoff message: this sprint ran attended in one isolated worktree
with the message plane off-limits; the milestones below were executed TDD-first in order.

## Milestones

| ID | Milestone | LOC | Acceptance | Status |
|----|-----------|-----|------------|--------|
| M1 | Failing tests first: exact repro + soundness controls (direct, private helper, closure, HOF, record, stored local, sibling via `./`, bare, `pkg/<self>/`) | 250 test | Repro FAILS on base with the std/stream message; controls assert *ceiling* violation naming Process and not attributed to std | ✅ |
| M2 | `isOwnPackageModule` + move ceiling to `effect_ceiling.go`; run before `ValidateEffects` | 80 | Repro passes; `pkg/<self>` checked | ✅ |
| M3 | Authority-entry walker (exhaustive over Core; VarGlobal-to-non-own, Intrinsic, DictRef; fail loudly on missing CoreTI) | 180 | Closure/HOF/record/stored controls fail the ceiling independently of the effect checker | ✅ |
| M4 | Corpus regression run (104 packages) — revealed the every-node-type variant broke 7 motoko-ext packages; narrowed to entry points; precision controls added | 60 test | 0 new failures; 4 false positives cleared | ✅ |
| M5 | Cache soundness: ceiling in own modules' cache key + test proving the cache served the module first | 10 + 45 test | Narrowed max re-checked after a cached wide pass | ✅ |
| M6 | Mutation tests, CHANGELOG, packages guide, `make test/lint/verify-examples`, file sizes | docs | All mutations caught (table below) | ✅ |

## Mutation results

| Mutation | Caught by |
|----------|-----------|
| M1 drop authority walk (b) | 4 closure/HOF/record/stored controls |
| M2 `pkg/<self>` treated as a dependency | relative + `pkg/<self>` sibling controls, `TestIsOwnPackageModule` |
| M3 std checked as own (the bug) | repro, import-without-reference, all controls (std attribution) |
| M4a VarGlobal never an entry point | the 4 closure/HOF/record/stored controls |
| M4b own VarGlobals charged too | own-sibling precision control (motoko `_smoke` shape) |
| M5 ceiling dropped from cache key | cache test (5/5 runs) |
| M6 charge every node's type | not caught by unit tests (CoreTI rows of local lambdas are not resolved); caught by the corpus run (7 motoko-ext regressions) |

# M-TEST-HARNESS-MODULE-SCOPED-ENVS: Module-scoped private names in the test harness

**Status**: Implemented (2026-10-02; also fixes #1516)

> **Implementation note (2026-10-02).** Shipped as designed (Option A, per-module child
> environments) in `internal/testing/module_scope.go`, with three deviations:
> (1) the root module is identified by **Core-program identity** (`LoadedModule.Core ==
> pipeline.Result.Artifacts.Core`), not by `ResolveModuleIdentity` — that identity is the seed
> identity, and the pipeline's module keys derive from file paths (temp copy, relaxed modules),
> so a string compare would not match; (2) re-exports are written to the owning module's env and
> its qualified key, and to the shared env only for the root — writing every module's re-exports
> to the shared bare namespace would re-open the leak; the old deferred pass also never worked
> (it called `evaluator.Eval` before any resolver was set, and dropped the error), so re-exports
> are now resolved directly to a fixpoint; (3) the same root-vs-stdlib collision was reported
> separately as #1516 (private `isErr`/`words` replaced by std/result / std/string exports when
> the root module sorted before `std/`) and is fixed by the same rule. The four harness paths now
> share one `newHarnessEvaluator`. Tests: `internal/testing/module_scope_test.go` (both sort
> orders, named + inline paths, direct root reference, resolver fallback), each mutation-tested.
**Target**: v0.50.2
**Priority**: P1 (High — `ailang test` reports false failures for any multi-module package where two modules define same-named private functions; real consumer hit: stapledons-godot, protocol.reject vs ship.reject)
**Estimated**: 2 days (~12 hours: 5h implementation + 4h tests + 2h verification/docs + buffer)
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Restores cross-tool determinism: the same program now evaluates identically under `ailang run` and `ailang test`; today `test` silently depends on lexicographic module order, not program semantics |
| A2: Replayability | 0 | No trace/replay changes |
| A3: Effect Legibility | 0 | No effect-row or capability changes; the fix is orthogonal to effects (private helpers may be pure or effectful — scoping applies equally) |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Test results become trustworthy local verification: a passing/failing test reflects program semantics, not a harness namespace accident |
| A6: Safe Concurrency | 0 | No concurrency changes (per-module envs are used by one evaluator per harness run, same as today's shared env) |
| A7: Machines First | +1 | AI-generated packages idiomatically reuse helper names per module (`helper`, `validate`, `reject`); today models must invent globally-unique private names as a workaround — an invisible constraint the language never declares |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No resource-cost changes |
| A10: Composability | +1 | Modules compose in tests exactly as they compose in `run`; the test/tool boundary stops being a semantic boundary |
| A11: Structured Failure | 0 | The residual fallback path already fails loudly; the fix removes the silent wrong-function dispatch in the common path |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — module iteration stays sorted (deterministic); the fix removes an order-*dependent* outcome, it does not add ordering
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Removes a human-invisible workaround constraint, not a machine convenience

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

`ailang test` (single file, or `--package` discovery) evaluates named-test bodies,
inline-test harnesses, cluster harnesses, and contract (requires/ensures) property
harnesses through a private module-evaluation mechanism in `internal/testing/` that
**flattens every loaded module's functions into one shared `eval.Environment` under
their bare names**. When two modules each define a private (non-exported) function
with the same name, the module processed last overwrites the other's bare-name
binding. Since intra-module calls elaborate to bare `core.Var` references (see
Verification Log V1), a function body then calls the *other module's* private
function — a silent cross-module type confusion that surfaces as an opaque
evaluation error.

**Reproduction (v0.50.1, verified in-session):** package with `a.ail`
(`pure func helper(r: {n: int}) -> int`, `export pure func fa`), `b.ail`
(`pure func helper(r: {s: string}) -> string`, `export pure func fb`), and
`t_test.ail` importing both:

```
$ ailang test --package .
  ✗ both
      evaluation error: record has no field: s        ← a's fa called b's helper

$ ailang test t_test.ail          # same failure — the bug is in the harness,
  ✗ both                             not in --package discovery

$ ailang run -entry both main.ail        → true   (interpreter)
$ ailang run -entry both -bytecode main.ail → true   (bytecode VM)
```

The error is deterministic but *order*-deterministic: renaming `a.ail` to `z.ail`
(so its module sorts *after* `b`'s) flips the failure to `record has no field: n`
(verified in-session, V2) — the last module in `sort.Strings` order wins. Neither
outcome is the program's semantics.

**Why the previous fix could not cover this:** M-MOTOKO-INLINE-TEST-HARNESS M2
(v0.16.3) fixed the *exported*-name collision (two stdlib modules both exporting
`length`) by binding module-qualified env keys (`"std/string.length"`) and having
`CombinedResolver` prefer them for module-qualified `VarGlobal` references. That
works only when a qualified reference exists. A private function has **no**
`VarGlobal` — the elaborator only emits `VarGlobal` for symbols in the module's
import table (`internal/elaborate/expressions.go`, V1) — so a private `helper` is
reachable *only* through its bare env name, and the qualified-key mechanism has
nothing to latch onto. The bare-name writes were left in place, with a comment
asserting the only concern is "ordering, which is now deterministic"
(`internal/testing/executor_helpers.go:585-588`, V3) — deterministic, and wrong.

**Impact:**
- **Who:** any multi-module package tested with `ailang test` — including every
  package-style consumer (stapledons-godot hit `protocol.reject` vs `ship.reject`
  and renamed a function as a workaround, which hides the bug and pollutes the
  package's design). AI-generated packages are structurally prone to it (per-module
  `helper`/`validate` idioms).
- **How significant:** blocker for affected packages (false test failures with an
  error message pointing at the *victim* module, nowhere near the cause); silent
  wrong-function dispatch (no diagnostic) in the general case. `ailang run` is
  correct, so the failure looks like a "test-only" flake and misleads debugging.
- **Also affected:** the same shared-env mechanism is used by all four harness
  paths (V4), so inline tests, cluster tests, and contract property tests inherit
  the same corruption whenever a helper name collides across imported modules —
  including two stdlib modules that happen to share a private helper name.

## Goals

**Primary Goal:** A module's private names resolve within that module's own
environment in every `ailang test` evaluation path, so test outcomes match
`ailang run` for any multi-module package.

**Success Metrics:**
- The 3-file reproducer (a/b/t_test above) passes `ailang test --package .` and
  `ailang test t_test.ail`, under *both* module-naming orders (determinism check).
- All existing harness regression tests stay green: `TestAliasImportCollision`,
  `TestADTConstructorFromImportedModule`, `TestADTConstructorInCluster`,
  `TestClusterEvalWithImportedHelper`, `TestFunctionlessNamedTestsResolveStdlibBuiltins`
  (V5), plus `make test-core`.
- Zero new syntax, zero new error codes, no change to `ailang run` behavior
  (production paths in `internal/link/resolver.go` untouched).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Scope private names via per-module child environments inside `injectModuleBindings` (not by converging the harness on `link.Resolver`) | Localized fix keeps the existing lazy-FunctionValue architecture; full resolver convergence deletes ~200 LOC of mirror semantics but requires linker/iface assembly and re-validates all four harness paths at once | human | design | med |
| Root (module-under-test) keeps bare names in the main evaluation env | Test bodies and harness expressions are elaborated in the root module's scope and reference its functions by bare `core.Var`; dropping root injection breaks all inline/cluster/ensures paths | compiler | design | med |
| Module-qualified keys for exports stay in the shared env (unchanged) | `CombinedResolver` Case 2 resolves cross-module `VarGlobal` references through them; M2's behavior must be preserved | compiler | design | low |
| When a bare-name lookup now misses (previously a silent leak to another module's function), fail loudly with the existing `undefined reference` error | CLAUDE.md §2 no-silent-fallbacks; a loud failure is the only honest outcome for a program that could never have been elaborated | human | design | low |
| No new error codes; no changes outside `internal/testing` | The bug and its fix live entirely in the test harness; production resolution (`internal/link/resolver.go`) is already correct | compiler | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Per-module child environments (Option A) vs converging on `link.Resolver` (Option B) — A chosen; B documented under Non-Goals and Future Work
- [x] Root-module exception exists and is identified by the resolved module identity (canonical ID), threaded from the runner, not by guessing from file paths
- [x] Residual bare-name lookups fail loudly instead of leaking across modules

## Solution Design

### Overview

The test harness (`internal/testing`) evaluates test bodies by running the module
pipeline on the (stripped) test file, caching the loaded modules
(`pipeline.Result.Modules`), and then **injecting** every module's functions into
one shared `eval.Environment` so that function bodies — which reference their own
module's private helpers as bare `core.Var` — can resolve them at call time.
Production `ailang run` never does this: `link.Resolver` evaluates each module in
its *own* evaluator environment, so bare names are naturally module-scoped.

The fix gives the harness the same scoping rule, without replacing the harness's
lazy-injection architecture:

> **Scoping rule:** a module's bare names are bound only in that module's own
> environment (a child of the shared env). Cross-module references resolve
> exclusively through module-qualified keys. The module under test ("root")
> additionally exposes its bare names in the main evaluation env, because test
> bodies and harness expressions are elaborated in the root module's scope.

This is one rule covering all four harness evaluation paths, private and exported
names alike — not a special case per collision.

### Architecture

**Components:**

1. **Per-module environments in `injectModuleBindings`** (`internal/testing/executor_helpers.go`).
   For each module (iteration order stays `sort.Strings` — deterministic):
   - Create `moduleEnv := env.NewChildEnvironment()` (child of the shared env, so
     qualified keys, builtins and re-exported names remain reachable through the
     parent chain — strictly fewer lookups change than under the flatten).
   - `core.Let` lambdas: keep the qualified key write to the shared env
     (`modulePath + "." + name`, unchanged); move the bare-name write from the
     shared env to `moduleEnv`; set `FunctionValue.Env = moduleEnv`.
   - `core.LetRec` groups: `recEnv := moduleEnv.NewChildEnvironment()` with the
     existing `IndirectValue` cells (self/mutual recursion), bare bindings in
     `moduleEnv`, qualified key in the shared env. Cross-SCC references within the
     same module resolve through the `recEnv → moduleEnv` parent chain; Core decls
     are in dependency (topo) order, so earlier groups are already bound.
   - **Root exception:** the module whose canonical ID equals the executor's
     resolved module identity *also* writes bare names into the shared env
     (exactly today's behavior for that one module). One root per executor (each
     test file gets a fresh Runner/Executor — verified, V6), so no root-vs-root
     collision is possible.
   - The executor records `moduleEnvs map[string]*eval.Environment` (module
     canonical ID → its env), rebuilt whenever `e.modules` is repopulated.

2. **Per-module deferred re-exports.** The `let concat = VarGlobal{pkg/std/list, concat}`
   re-export pass currently defers globally and writes the bare name into the
   shared env. Keep the *global deferral* (a re-export's target qualified key may
   belong to a module processed later — sort order is lexical, not dependency
   order), but write the resolved value into (a) the owning module's `moduleEnv`
   — so the module's own private references find it — and (b) the shared env, as
   today. (b) preserves M2's cross-module fallback semantics; two modules
   re-exporting the same bare name remain last-writer-deterministic in the shared
   env, unchanged from today — a genuine author-owned name clash.

3. **`CombinedResolver` fallback repair** (`internal/testing/executor_helpers.go`).
   Case 2 (module-qualified reference) falls back to bare-name `Env.Get` today for
   "modules whose path wasn't captured". That fallback must consult
   `moduleEnvs[ref.Module]` (the owning module's env) instead of the shared env,
   so a module-less or pathless module still resolves while non-root module bare
   names are no longer globally visible. Case 0 (`$adt`), Case 1 (`$builtin`) and
   Case 3 (unqualified → root/harness names) are unchanged.

4. **Root identity plumbing.** `RunTestsFromFileWithConfig` already resolves the
   module identity (`ResolveModuleIdentity`, `internal/testing/runner.go:587`);
   thread it into the `Executor` (constructor arg or setter alongside
   `SetSourceFile`) so `injectModuleBindings` can identify the root module by
   canonical ID rather than by comparing file paths (fragile under temp-file
   renaming, symlinks, and the `_namedtest_body_*.ail` temp module). If no module
   in `e.modules` matches the identity, no root exception applies (module-less
   snippets — the synthetic `_test/<base>` module path is compared identically).

### Implementation Plan

**Phase 1: Scoping core** (~4 hours)
- [ ] Thread resolved module identity into `Executor`; store on the struct
- [ ] Restructure `injectModuleBindings` into per-module env creation; move bare-name writes (Let and LetRec) into `moduleEnv`; keep qualified-key writes unchanged
- [ ] Root exception: bare names of the identity-matched module also go to the shared env
- [ ] Record `moduleEnvs`; rebuild on `e.modules` refresh
- [ ] Unit test: synthetic `e.modules` with two modules defining same-named private lambdas; assert each resolves its own through the main env's qualified lookups and through each other's bodies

**Phase 2: Re-exports and resolver fallback** (~3 hours)
- [ ] Per-module deferred `VarGlobal` re-exports (owning `moduleEnv` + shared env)
- [ ] `CombinedResolver` Case 2 bare fallback → `moduleEnvs[ref.Module]`
- [ ] Run `TestAliasImportCollision` — it is the canonical consumer of the fallback path (V5)

**Phase 3: Regression tests + verification** (~5 hours)
- [ ] In-process regression test for this bug: temp package dir (`ailang.toml` + `a.ail` + `b.ail` + `t_test.ail`, exact reproducer) run through `RunTestsFromFile`; assert pass
- [ ] Same test with module names swapped so the other sort order wins (the V2 determinism experiment, generalized)
- [ ] Inline-test-path variant of the reproducer (function with `tests [...]` block calling both imported functions)
- [ ] Full `make test-core`; manual CLI verification with the reproducer package (`test --package`, `test <file>`, `run` interpreter + `--bytecode` parity)

### Files to Modify/Create

**New files:**
- None (tests go into existing `internal/testing/*_test.go` files, following the
  package's layout conventions)

**Modified files:**
- `internal/testing/executor_helpers.go` (+~60/−~25 LOC) — per-module envs in `injectModuleBindings`, per-module re-export deferral, `CombinedResolver` Case 2 fallback, update the PASS 1 comment block (its "NOT the bare name" claim finally becomes true)
- `internal/testing/executor.go` (+~15 LOC) — root-identity field + constructor/setter plumbing; `moduleEnvs` field; pass identity into the four `injectModuleBindings` call sites (they all live in this file)
- `internal/testing/runner.go` (+~3 LOC) — pass the already-resolved identity to the executor in `RunTestsFromFileWithConfig`
- `internal/testing/executor_regression_test.go` (+~120 LOC) — reproducer tests (named-test path, swapped-order determinism, inline-path variant)
- `internal/testing/executor_helpers_test.go` or a new focused test file (+~60 LOC) — unit test of the scoping rule on synthetic module maps

## Examples

### Example 1: The reported failure, before/after

**Before (v0.50.1, verified in-session):**
```
$ ailang test --package .
  ✗ both
      evaluation error: record has no field: s
```

**After (expected):**
```
$ ailang test --package .
  ✓ both (1 tests: 1 passed, 0 failed, 0 skipped)
```

…and the stapledons-godot workaround (`protocol.reject` renamed away from
`ship.reject`) can be reverted; the package's intended names work under both
`run` and `test`.

### Example 2: What changes inside the harness

**Before — one shared env, bare names overwritten (`executor_helpers.go:585-588`):**
```go
// Bare name: set unconditionally.  For Let (non-recursive)
// lambdas there is no self-reference issue; the only concern is
// ordering, which is now deterministic (sorted paths above).
env.Set(d.Name, funcVal)          // x/pkg/b's helper silently overwrites x/pkg/a's
```

**After — module-scoped bare names, qualified keys unchanged:**
```go
moduleEnv := moduleEnvs[modulePath]            // per-module child env
funcVal := &eval.FunctionValue{..., Env: moduleEnv}
env.Set(modulePath+"."+d.Name, funcVal)        // shared env: qualified only
moduleEnv.Set(d.Name, funcVal)                 // bare name: module-scoped
if modulePath == rootModuleID {                // root exception
    env.Set(d.Name, funcVal)
}
```

## Success Criteria

- [ ] Reproducer package (3 modules, same-named private `helper`) passes `ailang test --package .` — with both lexicographic orders of the two helper-defining module names
- [ ] `ailang test t_test.ail` (no `--package`) passes for the same package — the bug is not discovery-mode-specific
- [ ] `ailang run` behavior unchanged: reproducer's entrypoint still evaluates `true` on interpreter and `--bytecode`
- [ ] All harness regression tests green: `TestAliasImportCollision`, `TestADTConstructorFromImportedModule`, `TestADTConstructorInCluster`, `TestClusterEvalWithImportedHelper`, `TestFunctionlessNamedTestsResolveStdlibBuiltins`, full `internal/testing` suite, `make test-core`
- [ ] A previously-leaking lookup (bare name defined only in a non-root module) now fails loudly with the existing `undefined reference: <name> (module: <path>)` error — verified by a negative test
- [ ] All tests passing
- [ ] CHANGELOG entry under v0.50.2 (changelogs/v0.32-current.md), citing this doc and the regression tests
- [ ] No files outside `internal/testing` modified (production resolution untouched)

## Testing Strategy

**Unit tests:**
- Synthetic `e.modules` map with two modules, each with a `core.Let` lambda named
  `helper` (different bodies); assert each module's body resolves its own `helper`
  when invoked through the shared env's qualified entries
- `CombinedResolver` Case 2 fallback: module-qualified ref whose qualified key is
  absent resolves via the owning module's env; a ref to a name absent from the
  owning module errors loudly

**Integration tests:**
- Reproducer package in a `t.TempDir()` (manifest + a.ail + b.ail + t_test.ail),
  run via `RunTestsFromFile` — named-test path (`test "both" { checkBoth() }`)
- Swapped module-name variant (both sort orders) — pins the determinism repair
- Inline-test variant: a function with a `tests [...]` block calling both imported
  functions — covers `EvaluateInlineTestsWithHarness`/cluster path with colliding
  private names

**Regression-surface tests (one per Conflict Surface "must still work" entry):**
- `TestAliasImportCollision` (executor_regression_test.go:66) — aliased + plain
  import of same-named *exported* functions (M2 behavior)
- `TestADTConstructorFromImportedModule` / `TestADTConstructorInCluster`
  (executor_regression_test.go:93/124) — `$adt` resolution in harness and cluster
- `TestClusterEvalWithImportedHelper` (executor_regression_test.go:37) — root
  helper calling an imported stdlib function
- `TestFunctionlessNamedTestsResolveStdlibBuiltins` (executor_builtin_resolution_test.go:7) — named-test-only module resolving stdlib builtins
- CLI-level fixtures: `ailang test tests/record_update_regression_test.ail`,
  `ailang test tests/m_rt1_imported_constructor_pattern_test.ail`,
  `ailang test std/trace_test.ail` (all exist, V7)

**Manual testing:**
- Reproducer package from this doc, exercised through the CLI in all four
  combinations: `test --package .`, `test t_test.ail`, `run` interpreter, `run --bytecode`
- Re-run the stapledons-godot package (un-renamed) if available locally

## Conflict Surface

This change does not touch `internal/parser|lexer|ast|types|elaborate|iface|codegen|eval|vm|effects`
or `cmd/ailang/exec.go`; the "grammar" here is the **key space of the shared
evaluation environment** consumed by the four harness evaluation paths, plus
`CombinedResolver`'s lookup cases. That surface is enumerated below with the same
discipline, because the failure mode is identical: an existing consumer of a
position we are changing.

### Syntactic positions touched

- Bare-name writes in `injectModuleBindings` Pass 1 (`executor_helpers.go:588` for
  `core.Let`, `:643` for `core.LetRec` bindings): previously the *shared* env, now
  the owning module's child env (plus the shared env for the root module only)
- `FunctionValue.Env` for injected module lambdas: shared env → owning module env
- Deferred `VarGlobal` re-export writes: shared env → owning `moduleEnv` + shared env
- `CombinedResolver.ResolveValue` Case 2 bare-name fallback: shared env → owning module env
- Module iteration order and per-module decl order: unchanged (`sort.Strings`, decl order)

### What else lives here (consumers of the shared env key space)

| Consumer | Existing valid form | Effect of change |
|----------|--------------------|-------------------|
| Root-module function bodies + test/harness expressions | bare `core.Var` lookups against main env (root decls bound by `EvalCoreProgram` and by root injection) | Unchanged — root exception preserves root bare names in the shared env |
| Cross-module `VarGlobal` refs (imports, incl. stdlib) | qualified key `module.path.name` in shared env | Unchanged — qualified writes untouched |
| M2 aliased-import fallback | Case 2 qualified lookup, bare fallback for uncaptured paths | Fallback retargeted to owning module env; same results for valid programs |
| Re-exports (`let concat = VarGlobal{...}`) | deferred bare write to shared env | Also written to owning module env; shared-env write kept |
| Imported ADT constructors in test bodies | `$adt` refs → `r.Modules[*].Iface.Constructors` | Unchanged |
| Root-module ADT constructors (`injectADTConstructors`) | bare ctor names in main env from `sourceFile.Decls` | Unchanged |
| Builtin shadowing (CombinedResolver Case 1 env fallback) | `Env.Get` after builtin registry miss | Unchanged for root names; cross-module private shadowing was never elaborable |

### Disambiguation strategy

The root module is identified by canonical module ID (threaded from
`ResolveModuleIdentity`), not by name shape or file-path heuristics — there is
exactly one root per Executor instance (fresh Runner/Executor per test file,
verified V6), so "which module keeps shared bare names" is unambiguous by
construction. All other modules' bare names are unreachable from the shared env,
so no ordering, guessing, or name-shape disambiguation exists at all: a bare name
in the shared env is either root-owned or absent.

### Programs that MUST still work

1. `internal/testing/executor_regression_test.go` — `TestAliasImportCollision` (M2 exported-name collision), `TestADTConstructorFromImportedModule`, `TestADTConstructorInCluster`, `TestClusterEvalWithImportedHelper`
2. `internal/testing/executor_builtin_resolution_test.go` — `TestFunctionlessNamedTestsResolveStdlibBuiltins`
3. `tests/m_rt1_imported_constructor_pattern_test.ail` — imported constructor + pattern matching through the test CLI
4. `tests/record_update_regression_test.ail`, `std/trace_test.ail` — standard `ailang test <file>` fixtures
5. The stapledons-godot package shape (un-renamed `protocol.reject` / `ship.reject`) — the reported consumer

### What deliberately changes

- A bare-name lookup that previously (and only ever wrongly) resolved *into a
  different module's private function* now fails with the existing
  `undefined reference: <name> (module: <path>)` / `function <module>.<name> not yet
  evaluated in environment` errors. No valid program could elaborate such a
  reference (cross-module private references are unrepresentable, V1), so any
  program hitting this was already broken — now it is broken loudly (CLAUDE.md §2).
- Deterministic last-writer behavior between two non-root modules sharing a bare
  name is *removed* (there is no longer a shared slot to win). Programs that
  depended on the winner being called were mis-evaluating; both modules now get
  their own.
- Nothing in `ailang run`, the REPL, serve-api, or the production resolvers changes.
  No syntax, manifest, or CLI surface changes.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact `moduleEnvs` lifecycle (rebuild per `injectModuleBindings` call vs on `e.modules` refresh) — agent may choose; correctness requires only that envs are rebuilt no later than the modules map
- Whether the root-exception plumbing lands as a constructor argument or a setter on `Executor` — agent may choose (setter mirrors `SetSourceFile`; constructor arg is cleaner)
- Whether the root module's *LetRec* bare bindings need the shared-env write in addition to `moduleEnv` (the named-test path re-binds root decls via `EvalCoreProgram`; the harness paths do not) — implement conservatively: write both, as today
- Unit-test file placement (extend `executor_regression_test.go` vs a new `executor_scoping_test.go`) — agent may choose
- Whether to also add the reproducer as an `eval_projects/` fixture — agent may choose; not required by success criteria

## Non-Goals

**Not in this feature:**
- Converging the harness on `link.Resolver` (production per-module evaluation) — the deeper unification; see Future Work. This doc deliberately keeps the lazy-injection architecture to bound the blast radius
- The MOD010 relaxed-path warning noise from the `_namedtest_body_*.ail` temp file (`module 'x/pkg/t_test' does not match canonical path '_namedtest_body_...'`) — pre-existing, cosmetic, observed in the repro; unrelated to the bug
- Determinism of `CombinedResolver.resolveAdtFactory`'s map iteration when two loaded modules define the same type/constructor name — pre-existing latent hazard, noted in Future Work
- Any change to module-linking semantics, exports, or `ailang run`
- Backporting to releases before v0.50.2

## Timeline

**Week 1** (12 hours):
- Phase 1: scoping core (4h)
- Phase 2: re-exports + resolver fallback (3h)
- Phase 3: regression tests + verification (5h)

**Total: ~12 hours across 1 week** (estimate already 2x'd from the naive 5-6h)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A harness path silently relies on a non-root module's bare name (leak) | Medium | Sweep: grep harness builders for bare `core.Var` emission (`harness.go`, `pure_cluster.go`, `contract_domain.go` — all reference root-module bindings, V8); negative unit test makes any residual leak fail loudly instead of silently |
| Re-export edge cases (`engine.ail` pattern: module-let re-export chains) regress | Medium | Keep global deferral order and shared-env write (identical to today); `make test-core` covers stdlib re-export consumers |
| Root-identity mismatch for module-less test snippets (synthetic `_test/<base>` module) | Low | Identity is resolved by the same code path the runner already uses; unit test with a module-less snippet; fallback = no root exception (loud failure, not silent) |
| Per-module child envs change memory/latency profile of harness runs | Low | One extra env per module (≤ ~200 stdlib modules today); lazily populated, same values as before |
| Hidden consumers of `e.modules` outside `internal/testing` | Low | Verified none exist (V9) |

## Related Documents

**Implemented (may inform design):**

- [m-motoko-inline-test-harness.md](../../implemented/v0_16_3/m-motoko-inline-test-harness.md) — closest prior art: fixed the *exported*-name collision via module-qualified env keys (M2). This doc extends the scoping to *private* names, which have no qualified reference to latch onto — distinct bug class, same mechanism. Its PASS-1 rationale ("only concern is ordering") is the defect being repaired here.
- [m-testing-adt-import-resolution.md](../../implemented/v0_16_4/m-testing-adt-import-resolution.md) — second prior instance of the harness resolver diverging from production; its $adt fix mirrors `internal/runtime/resolver.go`, illustrating why the divergence is a bug farm.
- [m-dx-package-test.md](../../implemented/v0_10_0/m-dx-package-test.md) — the `--package` discovery mode the user hit; discovery is unaffected by this bug (single-file mode fails identically).
- [m-testing-inline-core-evaluation.md](../../implemented/v0_4_7/m-testing-inline-core-evaluation.md) — origin of the module-scope elaboration approach used by named-test bodies.

**Planned (check for overlap):**

- (none found — SimHash scan of `design_docs/planned/` for test-harness/module-scope topics returned no overlaps; the auto-search neural pass was unavailable in this environment, noted in Verification Log V10)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Task report: "ailang test: same-named private functions in different modules collide (ailang run is correct)" — coordinator task task-c8b74e6c, v0.50.0 binary (v0.50.1-6-g021c469 verified in-session)
- [Modules reference — Exports](/docs/docs/reference/modules.md): "Not exported: private to this module"
- Production contrast mechanism: `internal/link/resolver.go` (`Resolver.ResolveValue` — per-module evaluator + memo)
- Changelog lineage: M-MOTOKO-INLINE-TEST-HARNESS M2 (v0.16.3), M-TESTING-ADT-IMPORT-RESOLUTION (v0.16.4) in [changelogs/v0.10-v0.17-bytecode-vm.md](../../../changelogs/v0.10-v0.17-bytecode-vm.md)

## Verification Log

Every load-bearing claim above, with its check (per design-doc-creator hard gate):

| # | Claim | Check | Result |
|---|-------|-------|--------|
| V1 | Private (same-module, non-imported) function calls elaborate to bare `core.Var`, not `VarGlobal` — so a private name is resolvable *only* through its env | Read `internal/elaborate/expressions.go` `normalize` → `*ast.Identifier` case: `VarGlobal` only `if ref, ok := e.globalEnv[ex.Name]`; `globalEnv` holds imports (built via `BuildGlobalEnv(imports)`, `internal/elaborate/file.go:210-249`) and builtins, never a module's own private functions | Confirmed |
| V1b | Non-exported functions are module-private by language design | `docs/docs/reference/modules.md` §Exports ("Not exported: private to this module… Can use private functions internally"); semantics also confirmed live: `ailang run` resolves each `helper` correctly | Confirmed |
| V2 | The collision is deterministic and last-sorted-module wins | Live experiment (v0.50.1): package x with a.ail/b.ail/t_test.ail → `record has no field: s`; renaming `a.ail`→`z.ail` (module `x/pkg/z`, now sorting *after* `x/pkg/b`) → `record has no field: n`. Error flips with sort order | Confirmed |
| V3 | The bare-name overwrite is explicit in code, with a comment claiming ordering is the only concern | `internal/testing/executor_helpers.go:585-588` (`env.Set(d.Name, funcVal)` "Bare name: set unconditionally… ordering, which is now deterministic") and `:643` (LetRec `env.Set(binding.Name, funcVal)`); sort at `:559` | Confirmed |
| V4 | All four harness evaluation paths use the shared-env injection | `grep injectModuleBindings internal/testing/*.go`: call sites at executor.go:160 (ensures/requires harness), :329 (named-test bodies), :409 (inline harness), :622 (cluster harness) — all in `internal/testing` | Confirmed |
| V5 | Named regression tests exist and pin what the doc says | Read bodies: `TestFunctionlessNamedTestsResolveStdlibBuiltins` (executor_builtin_resolution_test.go:7, asserts 3 named-test statuses), `TestAliasImportCollision` (executor_regression_test.go:66, asserts 0 failures + ≥1 pass), `TestADTConstructorFromImportedModule` (:93), `TestADTConstructorInCluster` (:124), `TestClusterEvalWithImportedHelper` (:37, pre-fix error text in comment) | Confirmed |
| V6 | One root module per executor (no root-vs-root collision in a multi-file package run) | Read `cmd/ailang/test.go` `runPackageTests` → per-file `runTestFile` → `RunTestsFromFileWithConfig` → `NewRunnerWithConfig` (fresh Runner/Executor per file, runner.go:570-620) | Confirmed |
| V7 | CLI-level "must still work" fixtures exist | `ls`: `tests/m_rt1_imported_constructor_pattern_test.ail`, `tests/record_update_regression_test.ail`, `std/trace_test.ail` all present | Confirmed |
| V8 | Harness builders reference only root-module bindings by bare `core.Var` | `grep core.Var internal/testing/harness.go pure_cluster.go contract_domain.go`: all bare Vars are test-arg binders or the tested function's binding name (root scope); no import-module bare references | Confirmed |
| V9 | No consumer of `pipeline.Result.Modules` / `injectModuleBindings` outside `internal/testing` | `grep -rn "Result.Modules" internal/ cmd/ serveapi/` → only `internal/testing/executor.go`; `grep injectModuleBindings` → only executor.go/executor_helpers.go | Confirmed |
| V10 | No duplicate/coverage conflict in planned docs | `ailang docs search --stream planned/implemented` for "test harness module scoped envs", "private function collision test runner module scope", "name collision CombinedResolver" — no planned doc covers it; nearest implemented docs cited in Related Documents with the distinction stated | Confirmed |
| V11 | Reproducer behaves as reported on the current tree | Live (v0.50.1, this workspace): `ailang test --package .` → `evaluation error: record has no field: s`; `ailang test t_test.ail` → same; `ailang run -entry both main.ail` → `true`; `ailang run -entry both -bytecode main.ail` → `true` | Confirmed |
| V12 | Negative-existence: no existing test already covers same-named private functions across modules | `grep -rn "SameNamed\|same name\|same-name\|private" internal/testing/*_test.go` → no hits; `TestAliasImportCollision` covers only exported names | Confirmed |

Note on tooling (not a claim about the language): the create-doc script's
related-doc search returned empty in this environment — its SimHash result filter
(`grep -E "^\d+\."`) does not match on GNU grep and its `merge_results` pipeline
aborts the script under `set -e -o pipefail` when no results match. The scaffold
was reproduced from the script's template verbatim; the manual searches above
(V10) stand in for the automated coverage gate. Flagged for a `.pi/extensions`
or skill-script fix if it bites again.

## Future Work

- **Converge the test harness on `link.Resolver`** (production per-module
  evaluation) instead of the mirror implementation: deletes the whole
  qualified-key/re-export/fallback patch surface this doc extends. Trigger:
  the next harness-resolution divergence bug (this is instance #3 of the class —
  M-DX25 v0.7.4 flatten, M2 v0.16.3 qualified keys, this doc module-scoped envs).
- Deterministic ordering for `CombinedResolver.resolveAdtFactory` when two loaded
  modules define the same type/constructor name (currently map-iteration order).
- Quiet the MOD010 relaxed-path warning for the harness's own `_namedtest_body_*.ail`
  temp file (process-level dedup exists; the warning still prints once per run).

---

**Document created**: 2026-10-01
**Last updated**: 2026-10-01

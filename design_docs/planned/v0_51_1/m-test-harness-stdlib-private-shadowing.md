# M-TEST-HARNESS-STDLIB-PRIVATE-SHADOWING: `ailang test` replaces a module's non-recursive private functions with same-named, never-imported stdlib exports

**Status**: Planned
**Target**: v0.51.1
**Priority**: P0 (High — `ailang test` mis-evaluates the module under test whenever a non-recursive private helper shares a name with any export of a stdlib module the package imports *anything* from; consumer hit: stapledons-godot AI.4, `ai/stub_test.ail` private `isErr`, `ai/provider.ail` private `words`, worked around by renaming)
**Estimated**: 1.5 days (~9 hours: 3h implementation — landed jointly with the companion doc's sprint — 4h tests + 2h verification/docs; the scoping core itself is specified by the companion doc, this doc contributes the stdlib-side requirements, the root-identity mechanism, and acceptance fixtures)
**Dependencies**: [m-test-harness-module-scoped-envs.md](../../planned/v0_50_2/m-test-harness-module-scoped-envs.md) — same root cause, same function (`injectModuleBindings`); the two docs must land as one change (see Relationship below)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Today the same program passes or fails `ailang test` depending on whether the CLI path was spelled relative or absolute, where the stdlib root resolves, and the source directory's name — all invisible, semantics-free accidents of `sort.Strings` over module keys (verified live, V4/V5/V6). The fix removes the ordering dependency entirely |
| A2: Replayability | 0 | No trace/replay changes |
| A3: Effect Legibility | 0 | No effect-row or capability changes; the substituted functions here are pure — the failure is a value-level type confusion, orthogonal to effects |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Test outcomes become trustworthy local verification: a red test points at the program, not at a namespace accident in the harness; today the failure's error text (`no pattern matched in match expression`) points at the *victim* caller and looks like a bug in the user's code |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | AI-generated modules idiomatically reuse stdlib vocabulary for private helpers (`isErr`, `words`, `map`, `get`); today the model must invent names absent from the entire stdlib surface — an invisible, unteachable constraint (the reported consumer hit exactly this and renamed to `errIs`/`digitWords`) |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No resource-cost changes |
| A10: Composability | +1 | Importing one symbol from a module must not import that module's whole export surface into the evaluator's namespace; `test` and `run` stop being different languages |
| A11: Structured Failure | 0 | The fix removes a silent wrong-function dispatch; residual lookups already fail loudly |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — module iteration stays sorted; the fix deletes an order-*dependent* outcome rather than adding ordering
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Removes a human-invisible naming constraint, not a machine convenience

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

`ailang test` evaluates named-test bodies through a private harness in
`internal/testing/` that flattens **every loaded module's** functions — including
every stdlib module transitively pulled in by *any* import — into one shared
`eval.Environment` under their **bare names** (`injectModuleBindings`,
`internal/testing/executor_helpers.go:588` for `core.Let`, `:643` for
`core.LetRec`), in `sort.Strings` key order, last writer wins. A module-private
function that is elaborated as a `core.Let` (every **non-recursive** top-level
function; `internal/elaborate/file.go:316` emits singleton non-self-recursive
SCCs as `Let`) never binds into the shared environment during
`EvalCoreProgram` (`evalCoreLet` binds in a child env and restores,
`internal/eval/eval_expressions.go:354-370`), so the test body — and the bodies
of the module's own other functions — resolve it through the injected closure,
whose captured `Env` is the shared environment. Whatever module sorted last owns
the name.

When the last-sorted module is a **stdlib** module, the module-under-test's
private function is silently replaced by the stdlib function at evaluation time.
The type checker accepts the program (elaboration resolves the call as a local
`core.Var`; `ailang run` is correct on both the interpreter and the bytecode
VM), so the substitution is invisible until the stdlib closure receives the
caller's argument and fails with an error that blames the caller:

**Reproduction (v0.51.0 b99dd25, verified in-session, V1–V3):**

```
module shadow
import std/result (Result, Ok, Err)
import std/string (join)
type Box = { n: int }
pure func isErr(b: Box) -> bool = b.n < 0
pure func words(xs: [string]) -> [string] = xs
pure func checkIt() -> bool = !isErr({ n: 1 }) && join(",", words(["a", "b"])) == "a,b"
export func main() -> () ! {IO} = println(if checkIt() then "ok" else "BAD")
test "local helpers are not replaced by non-imported stdlib names" { checkIt() }
```

```
$ ailang run --caps IO --entry main shadow.ail            → ok
$ ailang run --caps IO --entry main --bytecode shadow.ail → ok
$ ailang check shadow.ail                                 → ✓ No errors found!
$ ailang test shadow.ail
  ✗ evaluation error: no pattern matched in match expression   ← std/result.isErr ran on a Box record
  (words-only variant: evaluation error: _str_words: expected String, got *eval.ListValue
                        ← std/string.words, a delegation to builtin _str_words, ran on a list)
```

Neither `isErr` nor `words` was **imported** — the shadowing side of the
collision is the stdlib module's *whole export surface*, loaded because the
module imported *anything* (`Ok`, `join`) from it.

**Why the last writer is (almost always) the stdlib — verified live (V4, V5, V6):**
the named-test harness re-elaborates the module through a temp file
`_namedtest_body_<rand>.ail` written next to the source
(`internal/testing/executor.go:282`), and the root unit's canonical ID is
`CanonicalModuleID` of that temp *path*. With a relative CLI path the temp file
is created in `.` and the key is just `_namedtest_body_<rand>` — an underscore
sorts before every stdlib key (`std/...`, `workspace/.../std/...`, even
`<embedded>/std/...`), so **the stdlib always wins**. With an absolute CLI path
the key carries the source directory, and whether the user's module or the
stdlib wins is decided by the directory name versus the stdlib root path. Same
file, same directory, only the path spelling differs:

```
$ cd /workspace/task-e0fdc621/zztest && ailang test shadow.ail
  ✗ evaluation error: no pattern matched in match expression
$ ailang test /workspace/task-e0fdc621/zztest/shadow.ail
  ✓ All tests passed!        ← the *user's* isErr overwrote the stdlib's, by accident
```

**Recursion-shape dependence (verified live, V7):** a *self-recursive* private
function is elaborated as `core.LetRec`, whose Phase 2.5 re-binds the local into
the shared env *after* injection (`eval_expressions.go:417-424`), overwriting
the stdlib's bare name — so the bug only bites **non-recursive** private
functions, and a refactor from recursive to non-recursive (or the reverse)
silently flips which function a green test was calling.

**Impact:**
- **Who:** every package tested with `ailang test` whose private helpers share
  a name with any export of any stdlib module it imports anything from —
  AI-generated code hits this structurally (private `isErr`, `words`, `map`,
  `get` are idiomatic per-module names). Real consumer: stapledons-godot AI.4
  (`ai/stub_test.ail`, `ai/provider.ail`), worked around by renaming
  (`errIs`, `digitWords`) — which hides the bug and pollutes the package's names.
- **How significant:** false test failures whose error text blames the caller
  ("no pattern matched in match expression" in a function that contains no
  match), or — in the other sort order — silently *green* tests that were never
  calling the code under test. `ailang run` is correct, so the failure reads as
  a "test-only" flake and misleads debugging (exactly what the reporter
  concluded: "looks like a bug in the caller").
- **All four harness paths inherit it** (`injectModuleBindings` is called from
  the ensures/requires harness, named-test bodies, inline harness, and cluster
  harness — executor.go:173/342/422/635, V8).

## Goals

**Primary Goal:** In every `ailang test` evaluation path, a module's private
functions resolve to that module's own functions regardless of what any loaded
stdlib module exports, of import lists, of CLI path spelling, of recursion
shape, and of module-name sort order — matching `ailang run`.

**Success Metrics:**
- The reported repro (shadow.ail above) passes `ailang test` under **both**
  CLI path spellings (relative and absolute) and in a source directory whose
  name sorts both before and after the stdlib root path — four orderings, one
  outcome (V4/V5/V6 are the pre-fix evidence that these orderings differ today).
- The `words`-only variant (delegation-builtin signature) passes identically.
- A **non-recursive** and a **self-recursive** private `isErr`, in otherwise
  identical modules, both resolve to the local function (V7 is the pre-fix
  evidence that they differ today).
- A module that imports **only** `Ok` from `std/result` never sees
  `std/result.isErr` in any lookup it can perform (V2: importing one symbol is
  today sufficient to lose your own `isErr`).
- All existing harness regression tests stay green (list under Conflict
  Surface), plus `make test-core`.
- `ailang run` (interpreter and `--bytecode`) unchanged; no files outside
  `internal/testing` modified.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Land jointly with [m-test-harness-module-scoped-envs](../../planned/v0_50_2/m-test-harness-module-scoped-envs.md) as one change set — this doc supplies requirements + fixtures, not a competing design | The root cause is one function (`injectModuleBindings`); two partial fixes touching it in different orders would each re-break the other's repro | human | design | med |
| Root identity must be captured **per pipeline run** as `loader.CanonicalModuleID(<the filename given to that run>)` — the executor creates the `_namedtest_body_<rand>.ail` temp file and knows its name (executor.go:282) — NOT the runner's pre-resolved `ResolveModuleIdentity` of the original file | The companion doc's root plumbing (thread the original file's identity) cannot match the temp module: in the named-test path the root unit in `e.modules` is keyed by the temp basename (MOD010 prints it: `canonical path '_namedtest_body_754825368'`), which matches neither the original file's canonical ID nor its module decl. As specified there, the named-test path — the companion doc's *own* repro path — would get no root exception and every named test body would die with `undefined reference` (or the bug survives if an implementer "fixes" that by falling back to today's injection) | compiler | design | low |
| The module-under-test's non-recursive (`core.Let`) private functions are covered by the same root exception as `LetRec` groups — bare names of the root module in the shared env, closures capturing the shared env | The Let-vs-LetRec asymmetry (V7) is the *mechanism* of this bug: under the companion doc's per-module-env rule alone, a root `Let` closure capturing only a `moduleEnv` would be indistinguishable from a non-root module's; the body statement decl and cross-decl references must still see root Lets | compiler | design | low |
| Stdlib delegation exports (e.g. `std/string.words` = `export pure func words(s: string) -> [string] = _str_words(s)`) get a dedicated regression fixture | The failure signature routes through a builtin (`_str_words: expected String, got *eval.ListValue`) and would otherwise be mistaken for a builtin-arity bug; the fixture pins that the *delegation closure itself* must never be reachable from a non-importing module | compiler | design | low |
| No new error codes; no changes outside `internal/testing` | Production resolution (`internal/link/resolver.go`) is already correct — `ailang run` proves it; the bug and fix live entirely in the harness | compiler | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Joint landing with the companion doc (one sprint, merged acceptance criteria from both docs; the companion doc's scoping rule is the mechanism)
- [x] Root identity = per-pipeline-run canonical ID of the run's root filename (temp file included), recorded on the Executor when `e.modules` is (re)populated — not threaded from the original file's identity
- [x] Root exception covers both `Let` and `LetRec` decls of the root module
- [x] Acceptance matrix includes: both path spellings × both directory-sort orders × non-recursive and self-recursive shadowed helpers × Core-function (isErr) and delegation-builtin (words) stdlib exports

## Solution Design

### Overview

The scoping mechanism is specified by the companion doc
([m-test-harness-module-scoped-envs](../../planned/v0_50_2/m-test-harness-module-scoped-envs.md)):
per-module child environments; bare names bound only in the owning module's env,
plus a **root exception** (the module under test also exposes bare names in the
shared evaluation env); module-qualified keys unchanged. This doc does not
re-specify that rule. It contributes what that doc's analysis could not see —
because this manifestation exercises paths its repro does not:

1. **The root of the named-test path is the temp module** — root identity must
   come from the pipeline run itself (see High-Impact Decisions), or the
   companion fix breaks loudly on its own repro.
2. **The shadowing side can be a module the user imported nothing from by
   name** — the whole stdlib export surface is loaded by any selective import
   (V2), so acceptance fixtures must use never-imported names.
3. **The Let/LetRec recursion-shape asymmetry** (V7) — a scoping fix that only
   repairs `LetRec` groups (where Phase 2.5 already accidentally self-heals
   today) would leave this exact bug alive for non-recursive helpers.
4. **Order-independence must be asserted, not assumed** — the pre-fix outcome
   flips on CLI path spelling and stdlib root location (V4-V6); the fix's
   acceptance matrix must pin all orderings.

### Architecture

Changes layered on the companion doc's Phase 1-3 plan:

1. **Root identity, per pipeline run** (`internal/testing/executor.go`).
   Wherever `e.modules = <pipeline result>.Modules` is assigned (executor.go:335
   and the cluster/inline paths), also record
   `e.rootModuleID = loader.CanonicalModuleID(<filename passed to that pipeline run>)`.
   For `EvaluateNamedTestBodyExprs` that is the temp file it created at :282;
   for `ExtractFunctionBinding`/`ExtractPureClusterForFunction` it is
   `e.modulePath` (or the synthetic temp path when the source had no module
   decl). `injectModuleBindings` treats the module whose key equals the recorded
   root ID as the root exception. If no module matches, no root exception —
   loud failure, never silent (CLAUDE.md §2). This *replaces* the companion
   doc's "thread `ResolveModuleIdentity` from `RunTestsFromFileWithConfig`"
   plumbing for the executor; the runner's identity continues to serve the seed
   and package purposes it already has.

2. **Root exception covers `Let` decls explicitly** (companion doc Phase 1).
   The root module's `core.Let` lambdas keep bare-name writes to the shared env
   and `FunctionValue.Env = <shared env>` — exactly today's behavior for that
   one module. The comment at executor.go:338 ("This ensures function bindings
   are in scope when the body expression is evaluated") is true only via this
   injection today (a `Let`'s binding never reaches the shared env through
   `EvalCoreProgram`); after the fix it stays true for the root and becomes
   *honestly false* for non-root modules. Rewrite the comment to say so.

3. **Stdlib exports stop entering the shared env under bare names** — direct
   consequence of the companion doc's rule; this doc's fixtures make it
   non-regressable:
   - `TestStdlibPrivateShadowing_CoreFunc`: import `std/result (Ok)` only;
     private non-recursive `isErr`; body asserts the local one ran.
   - `TestStdlibPrivateShadowing_DelegationBuiltin`: import `std/string (join)`
     only; private non-recursive `words`; asserts `join(",", words([...]))`.
   - `TestStdlibPrivateShadowing_SelfRecursive`: same as the first with a
     self-recursive `isErr` (pins the V7 asymmetry as fixed — both shapes
     resolve locally).
   - `TestStdlibPrivateShadowing_PathSpellingAndOrderIndependent`: the shadow
     repro invoked with a relative and an absolute path, from source
     directories whose canonical IDs sort before and after the stdlib root's
     key (assert all four outcomes identical and green).

4. **No `CombinedResolver` changes beyond the companion doc's Case-2 fallback
   repair.** This bug's references are bare `core.Var`s resolved purely through
   the env chain (`evalCoreVar`, `eval_expressions.go:167-182` — env-only, no
   resolver); the resolver is not on the failing path (V9).

### Implementation Plan

**Phase 0 — joint planning (1 hour)**
- [ ] Merge this doc's acceptance matrix into the companion doc's sprint plan
  as its Phase 3 additions; the sprint executes once against both docs

**Phase 1 — root identity (2 hours)**
- [ ] Record `rootModuleID` on every `e.modules` population site; delete any
  reliance on file-path heuristics
- [ ] `injectModuleBindings` root exception keyed on the recorded ID, covering
  `Let` and `LetRec`
- [ ] Unit test: synthetic `e.modules` containing a root keyed
  `_namedtest_body_test` + a `std/result`-like module exporting `isErr`;
  assert the shared env resolves the root's `isErr` and the std module's env
  resolves its own

**Phase 2 — fixtures (3 hours)**
- [ ] The four `TestStdlibPrivateShadowing_*` tests above, in-process through
  `RunTestsFromFile` on `t.TempDir()` layouts
- [ ] CLI-level check: the reporter's exact repro (shadow.ail) under the four
  orderings; `ailang run` interpreter + `--bytecode` parity before/after

**Phase 3 — verification & docs (3 hours)**
- [ ] Companion doc's full regression list + `make test-core`
- [ ] CHANGELOG entry citing both docs; note in the companion doc's
  Verification Log that its root-identity plumbing was superseded by the
  per-run rule (this doc, V10/V11)

### Files to Modify/Create

**New files:**
- None (fixtures go into `internal/testing/executor_regression_test.go` or a new
  focused `executor_stdlib_shadowing_test.go` — implementer's choice, mirroring
  the companion doc's deferred decision on placement)

**Modified files:**
- `internal/testing/executor.go` (+~20 LOC) — `rootModuleID` recording at each
  `e.modules` assignment; temp-file root key at :282/:335; comment rewrite at :338
- `internal/testing/executor_helpers.go` (+~10/−~5 LOC on top of the companion
  doc's changes) — root exception consults the recorded ID; no other new logic
- `internal/testing/*_test.go` (+~150 LOC) — the four fixtures
- Companion doc `m-test-harness-module-scoped-envs.md` (−1 plumbing decision, +1
  note) — its "Root identity plumbing" bullet is replaced by the per-run rule

## Examples

### Example 1: The reported failure, before/after

**Before (v0.51.0, verified in-session, V1):** see Problem Statement —
`run` ok, `check` ok, `test` fails with an error that names no stdlib symbol.

**After (expected):**
```
$ ailang test shadow.ail          # relative path
  ✓ local helpers are not replaced by non-imported stdlib names
$ ailang test /abs/path/shadow.ail   # absolute path — same result
  ✓ local helpers are not replaced by non-imported stdlib names
```
…and the stapledons-godot workarounds (`errIs`, `digitWords`) can be reverted;
the intended names work under both `run` and `test`.

### Example 2: What the harness does differently, in one table

| Lookup | Today (shared env, sort.Strings last-writer-wins) | After (per-module envs + root exception) |
|---|---|---|
| Body: `checkIt()` (root `Let`) | injected root closure; survives by luck of sort order | root closure — root exception, order-free |
| `checkIt` body: `isErr({n:1})` (root `Let`, non-recursive) | **last-sorted module's `isErr` — std/result's, on a Box record** | root's `isErr`; std/result's is bound only in `std/result`'s own env |
| `checkIt` body: `words([...])` (root `Let`) | **std/string's delegation closure → `_str_words` gets a list** | root's `words` |
| Self-recursive private `isErr` (root `LetRec`) | root's (Phase 2.5 rebinds after injection — accidental) | root's (by rule, not accident) |
| Cross-module import `join(...)` | qualified key `std/string.join` (no collision today) | unchanged |

## Success Criteria

- [ ] Reporter's repro passes `ailang test` in all four orderings (relative/absolute × dir-before/dir-after stdlib key) — the four pre-fix outcomes are documented in V4/V5/V6
- [ ] `words`-only (delegation builtin) variant passes
- [ ] Non-recursive and self-recursive shadowed helpers both resolve locally (V7 asymmetry eliminated)
- [ ] Importing only `Ok` from `std/result` does not make `std/result.isErr` reachable from the module under test
- [ ] Companion doc's regression list green: `TestAliasImportCollision`, `TestADTConstructorFromImportedModule`, `TestADTConstructorInCluster`, `TestClusterEvalWithImportedHelper`, `TestFunctionlessNamedTestsResolveStdlibBuiltins`, full `internal/testing` suite, `make test-core`
- [ ] `ailang run` interpreter and `--bytecode` unchanged on the repro (ok both before and after)
- [ ] A bare-name lookup that only ever wrongly resolved into a stdlib (non-root) module's function now fails loudly with the existing `undefined reference: <name> (module: <path>)` error — negative test
- [ ] All tests passing; CHANGELOG entry citing both docs; companion doc's root-identity plumbing note updated

## Testing Strategy

**Unit tests:**
- Synthetic `e.modules` + recorded root ID: root keyed like the temp basename;
  std module exporting the same bare names; assert shared-env lookups return the
  root's and the std module's own env returns its own (per companion doc's
  Phase 1 unit test, extended with a std-shaped module)

**Integration tests (the four fixtures, Architecture §3):**
- Core-function stdlib export (`std/result.isErr`), never imported
- Delegation-builtin stdlib export (`std/string.words` → `_str_words`), never imported
- Self-recursive variant (V7)
- Path-spelling × directory-order matrix (V4/V5/V6)

**Regression-surface tests:** companion doc's list (TestAliasImportCollision et al.) — these are the consumers of the shared-env key space that must not change behavior for valid programs.

**Manual testing:**
- Reporter's repro through the CLI in all four orderings + both run modes
- The stapledons-godot package (un-renamed `isErr`/`words`) if available

## Conflict Surface

This change (with the companion doc) touches `internal/testing` only — not
`internal/parser|lexer|ast|types|elaborate|iface|codegen|eval|vm|effects` or
`cmd/ailang/exec.go`. The surface here is the **key space of the shared
evaluation environment** and the **root-identity contract of the Executor**, so
the enumeration discipline is applied to those, not to grammar.

### Positions touched

- Bare-name writes in `injectModuleBindings` (executor_helpers.go:588 Let, :643 LetRec) — companion doc's rule, with this doc's root-identity amendment
- `FunctionValue.Env` for injected closures — owning module env (root: shared env)
- Root-identity recording at each `e.modules` assignment (executor.go:335 + cluster/inline sites)
- The `_namedtest_body_<rand>.ail` temp file's module identity (executor.go:282) — now load-bearing: its canonical ID *is* the root ID; no longer an accident of `sort.Strings`

### What else lives here (consumers of the shared-env key space)

| Consumer | Existing valid form | Effect |
|---|---|---|
| Named-test body statements (trailing App/expr decls) | bare `core.Var` lookups for root functions via injection | Unchanged for root (root exception); stdlib bare names no longer reachable |
| Root `LetRec` groups (self/mutual recursion) | Phase 2.5 rebind + injection | Unchanged — now by rule instead of by ordering accident |
| Inline/cluster/ensures harness expressions | bare `core.Var` refs to root bindings | Unchanged (root exception); companion doc V8 already verified these reference root scope only |
| Cross-module `VarGlobal` (imports incl. stdlib) | qualified keys `std/string.join` | Unchanged |
| `$adt` constructor factories | `r.Modules[*].Iface.Constructors` | Unchanged |
| Builtin references (`$builtin`, `_`-prefixed) | CombinedResolver Case 1 | Unchanged |
| **Entry-module implicit prelude** (`std/option`+`std/result` auto-imported for entry modules, loader.go:305-310) | makes Result/Ok/Err available in `ailang run` entry modules | Unaffected — constructor-only surface; the stripped temp file is not an entry module (no exported main), and even where the prelude loads the modules, their function exports are not in GlobalRefs (V2: explicit selective import of `Ok` was needed to lose `isErr` in the repro) |

### Disambiguation strategy

The root is identified by the canonical ID of the filename given to the
pipeline run that populated `e.modules` — a value the Executor itself
constructed. No name-shape, path-shape, or ordering heuristic; a module key
either equals the recorded root ID or it does not.

### Programs that MUST still work

1. Reporter's repro (this doc, V1) — under all four orderings
2. Companion doc's 3-module package repro (a.ail/b.ail/t_test.ail, both sort orders)
3. `internal/testing/executor_regression_test.go` — `TestAliasImportCollision`, `TestADTConstructorFromImportedModule`, `TestADTConstructorInCluster`, `TestClusterEvalWithImportedHelper` (read, V12)
4. `internal/testing/executor_builtin_resolution_test.go` — `TestFunctionlessNamedTestsResolveStdlibBuiltins` (read, V12) — the *positive* half of stdlib resolution: a named test that legitimately uses a stdlib import (`startsWith`) must keep working; the fix removes only the never-imported path
5. `internal/testing/named_test_env_test.go` — #906 contract: "a named test body sees exactly what the enclosing module sees — its imports, and the builtins" (read, V13)
6. CLI fixtures: `tests/m_rt1_imported_constructor_pattern_test.ail`, `tests/record_update_regression_test.ail`, `std/trace_test.ail` (exist, V14)

### What deliberately changes

- A lookup that previously (and only ever wrongly) resolved into a **never-imported stdlib export** now resolves to the module's own function, or fails loudly with the existing `undefined reference` error when no such function exists. No valid program could elaborate a reference to a non-imported stdlib function, so any program relying on the old behavior was already mis-evaluating.
- The recursion-shape dependence (V7) disappears: recursive and non-recursive private functions resolve identically.
- Path-spelling and directory-name dependence (V4/V5/V6) disappears.
- Nothing in `ailang run`, the REPL, serve-api, or production resolvers changes. No syntax, manifest, or CLI changes.

## Deferred Decisions

- Fixture file placement (extend `executor_regression_test.go` vs new `executor_stdlib_shadowing_test.go`) — implementer's choice, mirroring the companion doc's placement deferral
- Whether the four CLI orderings are also pinned as a shell-level test under `tests/` or only in-process — implementer's choice
- Whether `e.rootModuleID` is also asserted in a debug/doctor surface — not required
- Whether to add an `eval_projects/` fixture of the shadow repro — not required by success criteria

## Non-Goals

**Not in this feature:**
- Re-specifying the per-module-env scoping rule — that is [m-test-harness-module-scoped-envs](../../planned/v0_50_2/m-test-harness-module-scoped-envs.md); this doc amends it, it does not compete with it
- Converging the harness on `link.Resolver` (production per-module evaluation) — the companion doc's Future Work; the trigger condition (instance #3 of the divergence class) is now met twice over
- The MOD010 relaxed-path warning noise from the harness's temp file — pre-existing, cosmetic (companion doc Non-Goal)
- The user-vs-user module collision (protocol.reject vs ship.reject) — the companion doc's own manifestation; fixed by the same joint change
- Any change to stdlib delegation (builtin `_str_*` bindings) — the delegation is correct; only its harness visibility is wrong
- Backporting before v0.51.1

## Timeline

**Week 1** (~9 hours, riding the companion doc's sprint):
- Phase 0: joint planning (1h)
- Phase 1: root identity per pipeline run (2h)
- Phase 2: four fixtures + CLI matrix (3h)
- Phase 3: verification, CHANGELOG, companion-doc note (3h)

(Estimate already 2x'd from the naive 4-5h.)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| The companion sprint lands its scoping without this doc's root-identity amendment → named-test path breaks loudly (`undefined reference: checkIt`) or the bug survives via a fallback | High | Joint landing is a Design Freeze item; Phase 0 merges acceptance matrices before any code |
| A harness path reaches a stdlib function by bare name today because a module *forgot* to import it and the harness silently supplied it (accidental dependency) | Medium | Negative tests make such lookups fail loudly; companion doc's sweep of harness builders (its V8) plus this doc's fixtures cover the std side; any real dependency surfaces as an immediate, fixable import error |
| Root identity recorded from the wrong pipeline run (stale `e.modules` + new root) | Medium | Record root ID at the same statement as each `e.modules` assignment; unit test with two sequential runs |
| Delegation-builtin fixtures drift if stdlib delegation surface changes | Low | Fixture asserts behavior (`join(",", words([...])) == "a,b"`), not the delegation mechanism |
| Hidden consumers of `e.modules` outside `internal/testing` | Low | Verified none (companion doc V9; re-checked in-session, V15) |

## Related Documents

**Planned (overlap stated explicitly — duplicate gate):**

- [m-test-harness-module-scoped-envs.md](../../planned/v0_50_2/m-test-harness-module-scoped-envs.md) — **closest prior art, same root cause, same function** (`injectModuleBindings` bare-name flatten). Its manifestation is two *user* modules' same-named private functions; this doc's manifestation is the *module under test* vs a **never-imported stdlib export**. Distinct in what it exposes: (a) selective imports load whole std modules whose non-imported exports then shadow the root — import lists stop protecting anything; (b) the root of the named-test path is the `_namedtest_body_<rand>.ail` temp module, which that doc's root-identity plumbing cannot identify (this doc replaces it); (c) the recursion-shape (Let vs LetRec) asymmetry; (d) delegation-builtin failure signatures. **The two docs land as one change**; that doc is the mechanism, this doc is the requirements + acceptance amendment. Not a duplicate: its repro, risk table, and plumbing do not cover (a)-(d).

**Triage (the class call):**

- [test-executor-module-env-miswiring.md](../ailang-core-triage/test-executor-module-env-miswiring.md) — the triage that recommended the design-doc for this harness's env divergence; this bug is the stdlib-side member of that class (its "bare-name last-writer-wins between modules" prediction, realized on the stdlib side)

**Implemented (prior instances of the divergence class):**

- [m-motoko-inline-test-harness.md](../../implemented/v0_16_3/m-motoko-inline-test-harness.md) — M2 fixed exported-name collisions via qualified keys; private names had nothing to latch onto (this bug and the companion doc's are the two private-name members)
- [m-testing-adt-import-resolution.md](../../implemented/v0_16_4/m-testing-adt-import-resolution.md) — harness resolver diverged from production on `$adt`
- [m-named-test-blocks.md](../../implemented/v0_29_0/m-named-test-blocks.md) — origin of the named-test body path this repro uses
- [m-testing-inline-core-evaluation.md](../../implemented/v0_4_7/m-testing-inline-core-evaluation.md) — origin of the module-scope elaboration approach

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Task report: "test runner: a private helper resolves to a NON-imported stdlib export of the same name (isErr, words)" — coordinator task task-e0fdc621, v0.51.0 (b99dd25), consumer: stapledons-godot AI.4
- Issue #1461 — same-named private functions across modules (the user-vs-user sibling; companion doc's original report)
- [Modules reference — Exports](/docs/docs/reference/modules.md): "Not exported: private to this module"
- Mechanism sites: `internal/testing/executor_helpers.go:518-660` (injection), `internal/testing/executor.go:204-362` (named-test path), `internal/eval/eval_expressions.go:354-370` (Let, no parent propagation) and `:417-424` (LetRec Phase 2.5), `internal/elaborate/file.go:316-341` (singleton SCC → Let), `internal/loader/loader.go:375-391` (CanonicalModuleID)

## Verification Log

Every load-bearing claim above, with its check (per design-doc-creator hard gate). All live checks ran on the shipped binary v0.51.0 (b99dd25) at `/usr/local/bin/ailang`, workspace `/workspace/task-e0fdc621`, AILANG_STDLIB_PATH on disk at `/workspace/task-e0fdc621/std`.

| # | Claim | Check | Result |
|---|---|---|---|
| V1 | The reporter's repro behaves exactly as reported | Live: `ailang run --caps IO --entry main shadow.ail` → `ok`; `--bytecode` → `ok`; `ailang check` → `✓ No errors found!`; `ailang test shadow.ail` → `✗ evaluation error: no pattern matched in match expression` at shadow.ail:9:1 | Confirmed |
| V2 | Importing ONE symbol (`Ok`) from std/result suffices to lose a private `isErr`; importing nothing leaves it intact | Live: variant with `import std/result (Ok)` only → `✗ no pattern matched`; variant with no imports at all → `✓` pass. Read `resolveModuleImports` (pipeline_module_imports.go: selective symbols only → GlobalRefs; but the whole dep module is compiled and lands in `Result.Modules`, whose Let-lambda bare names all get `env.Set`) | Confirmed |
| V3 | The `words` variant fails with the delegation-builtin signature; std/string.words delegates | Live: words-only repro → `✗ _str_words: expected String, got *eval.ListValue`; `run` → ok. Read `std/string.ail:124`: `export pure func words(s: string) -> [string] = _str_words(s)`; `ailang builtins list` shows `_str_words [pure] std/string`; `std/result.ail:41` `isErr` is a Core match (hence "no pattern matched") | Confirmed |
| V4 | The winner is decided by `sort.Strings` over module keys; with a relative CLI path the root temp module's key is the bare basename `_namedtest_body_<rand>`, which sorts before every stdlib key, so std always overwrites the root's private names | Live: `cd /tmp/repro && ailang test shadow.ail` → FAIL; `cd /workspace/task-e0fdc621/zztest && ailang test shadow.ail` → FAIL. Read: executor.go:282 `os.CreateTemp(sourceDir, "_namedtest_body_*.ail")` with `sourceDir = filepath.Dir("shadow.ail") = "."` → relative basename; `loader.CanonicalModuleID` keeps the shape; `sort.Strings` (executor_helpers.go:559); bare writes :588/:643 unconditional | Confirmed |
| V5 | The same file in the same directory flips from FAIL to PASS on path spelling alone (absolute path → root key carries the dir and sorts after the std key in that layout) | Live: `cd /workspace/task-e0fdc621/zztest && ailang test shadow.ail` → `✗`; `ailang test /workspace/task-e0fdc621/zztest/shadow.ail` (absolute) → `✓ All tests passed!`; also `ailang test /tmp/repro/shadow.ail` → `✓`, with and without AILANG_STDLIB_PATH (embedded root) | Confirmed |
| V6 | With an absolute path under a directory sorting before the std key the user module still wins (order flip both ways) | Live: `ailang test /tmp/repro/shadow.ail` → `✓` (root `tmp/repro/_namedtest_body_N` sorts after `workspace/.../std/result`); relative in the same dir → `✗` (V4). Together with V5: all four orderings differ pre-fix | Confirmed |
| V7 | The bug bites non-recursive private functions only; self-recursive ones escape via LetRec Phase 2.5 re-binding | Live: identical module with self-recursive private `isErr` + `import std/result (Ok)` → `✓` pass, while the non-recursive variant (V2) fails. Read: file.go:316 singleton non-self-recursive SCC → `core.Let`; eval_expressions.go:354-370 `evalCoreLet` binds in a child env (no parent write); :417-424 Phase 2.5 propagates LetRec bindings to the parent — after injection, overwriting the stdlib bare name | Confirmed |
| V8 | All four harness paths use the shared-env injection | `grep -n "injectModuleBindings(evaluator" internal/testing/executor.go` → :173, :342, :422, :635 | Confirmed |
| V9 | Bare `core.Var` resolution is env-chain-only; the CombinedResolver is not on the failing path | Read `evalCoreVar` (eval_expressions.go:167-182): `e.env.Get` → error, no resolver fallback; `VarGlobal` is the only resolver consumer. The repro's calls are bare Vars (elaboration local — `ailang run` correct proves it; `normalize` in expressions.go:56-64 emits VarGlobal only for `globalEnv` names, which never contain non-imported stdlib functions — V2) | Confirmed |
| V10 | The root unit in `e.modules` is keyed by the temp file's canonical ID, not the original module's | Live: MOD010 warning on every temp-file run prints `canonical path '_namedtest_body_<rand>'` — no directory, no relation to the module decl. Read: pipeline_module_phases.go:223 `rootCanonical = CanonicalModuleID(src.Filename)`; assembleModuleResult includes the root unit under that key | Confirmed |
| V11 | `pipeline.Result` exposes no root-identity field; the executor must derive it from the filename it ran | Read `pipeline.Result` (pipeline.go:121-135): Value/Type/Artifacts/Interface/Modules/… — no root-ID field; grep `RootCanonical` in pipeline.go → none | Confirmed |
| V12 | Named regression tests exist and pin what the doc says | Read bodies: `TestClusterEvalWithImportedHelper` (executor_regression_test.go:37 — root helper calling an imported stdlib function), `TestAliasImportCollision` (:66), `TestADTConstructorFromImportedModule` (:93), `TestADTConstructorInCluster` (:124) | Confirmed |
| V13 | The #906 named-test contract ("body sees exactly what the enclosing module sees — its imports, and the builtins") is the positive half this fix must preserve | Read `internal/testing/named_test_env_test.go:1-24` header + `TestNamedTestOnly_FileHasBuiltinsAndStdlib` (named tests using `not` and `startsWith` — an *imported* stdlib function) | Confirmed |
| V14 | CLI-level fixtures exist | `ls`: `tests/m_rt1_imported_constructor_pattern_test.ail`, `tests/record_update_regression_test.ail`, `std/trace_test.ail` present | Confirmed |
| V15 | No consumer of `pipeline.Result.Modules` / `injectModuleBindings` outside `internal/testing` | `grep -rn "Result.Modules" internal/ cmd/ serveapi/` → executor.go only (companion doc V9, re-verified: only `internal/testing` hits) | Confirmed |
| V16 | No existing test covers private-vs-never-imported-stdlib shadowing | `grep -rln "isErr\|_str_words" internal/testing/*_test.go` → none; no shadowing fixtures exist in the package | Confirmed |
| V17 | The companion doc's root-identity plumbing cannot match the temp module (the amendment's premise) | Read companion doc §Architecture item 4 ("thread it into the Executor" from `ResolveModuleIdentity` at runner.go:587 — the ORIGINAL file) against V10: the temp key matches neither the original file's canonical ID nor its module decl; the companion doc's risk table covers only module-less snippets | Confirmed |
| V18 | The skill's related-doc search friction (empty results + script abort) is real and now fixed | The create script aborted on this machine (GNU grep `^\d+\.` never matches + unguarded grep in `merge_results` under `set -euo pipefail`) — the same failure the companion doc's tooling note recorded 2026-10-01 (its second-bite flag). Fixed in-session per the friction→extension doctrine: 4× `^\d+\.` → `^[0-9]+\.` and `|| true` on the merge pipeline (matches the live `ailang docs search` output format, verified). Search now returns results and the scaffold was created | Confirmed |

## Future Work

- Converge the test harness on `link.Resolver` (production per-module evaluation) — deletes the whole mirror-injection surface; this doc is instance #4 of the divergence class (M-DX25 flatten, M2 qualified keys, module-scoped envs, stdlib-side shadowing)
- Quiet the MOD010 relaxed-path warning for the harness's own temp file (now also load-bearing for root identity — the two changes should land together)
- Teach `ailang doctor` to assert harness/run parity on a shadow-shaped probe module (cheap, catches the next divergence class member)

---

**Document created**: 2026-10-02
**Last updated**: 2026-10-02

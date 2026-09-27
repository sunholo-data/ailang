# M-EFFECT-CEILING-REACHABLE: compute the package effect ceiling over what the package reaches

**Status**: Planned (implemented in the same PR, pending ratification)
**Target**: v0.44.2
**Priority**: P1 (a security-relevant declaration is currently forced to say something false)
**Estimated**: 1 day
**Dependencies**: None
**Bug Report**: Daneel, 2026-09-26 (via the ailang-core lane; verbatim repro below)

## Problem Statement

A package's `ailang.toml` `[effects].max` is its declared authority: "no code in this
package can reach an effect outside this list". The check that enforces it
(`validateEffectCeiling`, then in `internal/pipeline/package_resolver.go`) ran on **every
module compiled in the run whose ID did not start with `pkg/`** — which includes every
imported **stdlib** module. It checked each function *declared in that module* against the
root package's ceiling.

Daneel's repro (v0.44.0, reproduced on this branch's base `5f1be97ea`):

```toml
[effects]
max = ["Stream"]
```
```ailang
module m
import std/stream (disconnect, StreamConn)
export func bye(c: StreamConn) -> unit ! {Stream} = disconnect(c)
```
```
Error: in function asyncExecProcess in std/stream: effect ceiling violation in package t/a:
effects [Process] not in max [Stream]
```

`asyncExecProcess` is neither imported nor called. The error fires while compiling
`std/stream` itself. To type-check, Daneel's WebSocket route package (holding a live Vertex
session) had to add `Process` to its ceiling: runtime `--caps` still refuses Process, but the
manifest now claims authority the package does not have.

**Measured blast radius** (corpus of 104 packages with a ceiling across ailang-packages,
mk-main/packages, daneel, ailang-fleet, ailang-motoko and this repo, copied to scratch and
checked with the base and fixed binaries): 5 packages fail the ceiling today; 4 of them are
this exact false positive (`sunholo/ollama_stream` — a published package — plus
`examples/configdriven_provider_demo` in three repos, which blames `std/ai.callImage`). The
fifth (`daneel/tools`) is masked by the same bug and hides a genuine violation (below).

## Audit (principle 3) — related paths

| Path | Finding | Action |
|------|---------|--------|
| Ceiling on `std/` modules | The reported bug | Fixed: only own modules are checked |
| Ceiling on `pkg/<self>/...` modules | **Skipped** (`HasPrefix(modID,"pkg/")`), yet `import ./sibling` and `import pkg/<self>/x` both normalise to `pkg/<self>/...` (verified: `TestIntraPackageImports_*`, and the relative-import test here reports modID `pkg/t/a/exec`). A sibling module's own code escaped the ceiling | Fixed: `pkg/<self>/` is own |
| Closures / HOFs / stored lambdas | The declared-row check alone never saw a pure function returning or storing a Process closure; today these are caught only because the effect checker over-charges lambda bodies to the enclosing function — which `ailang-core-triage/stored-lambda-effects-charge-enclosing-fn.md` plans to stop | Fixed: the ceiling now walks the module's Core itself (rule (b)) and runs *before* `ValidateEffects` |
| Compile cache | **Pre-existing soundness hole**: the module cache key (`ModuleCacheKey`) does not include the manifest, so narrowing `max` served the cached verdict of the wider ceiling. Measured on the base binary: wide → pass, narrow → pass (cached), narrow + `AILANG_NO_CACHE=1` → violation | Fixed: own modules' key carries `pkg:[sorted max]` |
| `internal/pipeline/core_sanity.go` `walkCore` | Not exhaustive (no `RecordUpdate`, `Tuple`, `Array`) | Not reused; new walker is exhaustive and errors on unknown node kinds. `walkCore` left as-is (out of scope, noted) |
| Runtime `--caps` | Independent runtime gate; unaffected | None |
| `ailang iface` / publish manifest (`pkg_publish.go`, `registry-validator`) | Carry `Effects.Max` verbatim as metadata; they never recompute it | None; the validator runs `ailang check --package`, so it inherits the fix |
| `CheckEffectCeiling` (internal/pkg) | Set comparison, lowercase ≤2-char row-var skip, Debug ghost skip | Reused unchanged |

## The rule

The ceiling is checked on every module that **belongs to the package** —
`isOwnPackageModule(modID)`: not `$builtin`/`$adt`, not `std/...`, not `pkg/<dep>/...`;
`pkg/<self>/...` and every other (local) ID are own. Unexpected IDs default to *own*
(checked), never silently skipped.

For each own module the charged effect set is the union of:

- **(a)** every function's **declared** effect row (the pre-existing, documented check);
- **(b)** every concrete effect label anywhere in the inferred type (from CoreTypeInfo) of
  every **authority entry point** in the module's Core, wherever it occurs (called or not,
  inside lambdas, record fields, arguments): a `VarGlobal` whose module is **not** own
  (std, deps, `$builtin`, `$adt`), every `Intrinsic`, every `DictRef`.

Imported modules' bodies are never charged. Importing a name without referencing it charges
nothing.

### Soundness argument

Effectful behaviour enters a program only via a builtin or a function defined somewhere. Any
name used in an own module is either:

1. **a local** (parameter / let binding): its value came from the caller (the caller's
   authority — and *performing* it requires the enclosing function to declare the effect,
   which (a) checks; row-polymorphic callbacks carry a row variable, instantiated concretely
   at the caller's own reference) or from another expression of this module, which is
   walked;
2. **a top-level function of an own module**: its body is walked and its declared row
   checked when *that* module is checked. Own modules imported by the entry are compiled in
   the same run, or served from a cache entry whose key now includes this exact ceiling;
3. **anything else**: the elaborator resolves imports *and* builtins to `VarGlobal`
   (`AddBuiltinsToGlobalEnv`; `expressions.go` `normalize` for `ast.Identifier`), and
   every Core node has a CoreTypeInfo entry (M-DX4 invariant, enforced by
   `ValidateCoreTypeInfo` immediately before the ceiling check; the walker also fails loudly
   if an entry point lacks one). A VarGlobal's recorded type is its scheme's instantiation,
   so the concrete labels of the referenced signature are present at that node. (b) charges
   it.

So every concrete effect the package's code can reach is charged: (b) over-approximates
reachability at the granularity of "referenced anywhere" (a never-called stored closure still
counts), which is the sound direction.

### Why not the obvious alternatives

- **Exact call-graph reachability from exports**: unsound or fragile for functions passed as
  values, stored in records, and returned closures; also would *un-check* private helpers
  that nothing calls yet, which the package still ships. Rejected.
- **Charge every Core node's type** (first implementation attempt): sound, but the corpus run
  showed **7 newly failing packages** (`motoko-ext-ailang-docs`, `-context-mode`,
  `-decision-framework`, `-exa-search`, `-mcp`, `-omnigraph`, `-test-dummy`, and a new
  ceiling message on the already-failing `-a2a`) — because the motoko
  ABI fixes each hook's row (`{AI, Clock, IO, Net, SharedMem, Stream}`) even when the body
  uses only `{Process, FS, Env}`. Types of the package's own values are not authority.
  Rejected; entry-point charging has 0 regressions on the same corpus.
- **Rely on the effect checker's declared rows only** (just skip std): makes the repro pass,
  but regresses soundness for closures relative to today the moment the stored-lambda
  attribution fix lands, and leaves the `pkg/<self>` and cache holes. Rejected.

### Known limitations (stated, not hidden)

- **Modules not in the compile graph** are not checked (true before and after): `ailang check
  m.ail` checks `m` and what it imports. `ailang check --package .` is the package-wide gate.
- **User-defined typeclass instances with effectful methods**: dictionary nodes (`DictRef`)
  are charged by type; instance method bodies in own modules are walked if elaborated into
  the module's `Decls`. Not separately fixture-tested (std has no user `instance` decls to
  mirror); open question below.
- The ceiling is checked before `ValidateEffects`, so a function that both under-declares and
  exceeds the ceiling now reports the ceiling violation first.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Deterministic walk (sorted record fields); no runtime change |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | +1 | The manifest can state the package's true authority again |
| A4: Explicit Authority | +1 | Closes three silent under-checks (pkg/<self>, closures independent of the effect checker, cache) while removing the false one |
| A5: Bounded Verification | +1 | Per-module local check; no whole-program analysis |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | Violation now names the function and the exact reference (`reference to std/stream.asyncExecProcess at m.ail:5:72`) |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | — |
| A10: Composability | +1 | A package's ceiling no longer depends on which unrelated functions its imports happen to contain |
| A11: Structured Failure | 0 | Same error shape, more precise |
| A12: System Boundary | +1 | Package boundary = own modules, stated in one predicate |

**Net Score: +7** → Proceed. No −1 on A1/A3/A4/A7.

## Conflict Surface (touches `internal/pipeline` effect/ceiling ordering)

1. **Position extended**: the ceiling now reads Core + CoreTypeInfo, not just surface
   signatures; it runs before `ValidateEffects` instead of after.
2. **Other constructs in that position**: `ValidateEffects` (declared-vs-required rows),
   `validateLambdaAnnotations` (#386), the compile cache.
3. **Disambiguation**: independent checks; both must pass. Order only changes which error is
   reported first when both fail.
4. **Must still work**: `examples/intra_package_imports/` (own ceiling), every
   `TestIntraPackageImports_*`, the 104-package corpus (0 new failures), the full
   `make test` / `make verify-examples`.
5. **Deliberately changes**: (i) std/dependency function bodies are no longer charged;
   (ii) `pkg/<self>/` modules are now checked; (iii) a stored/returned closure referencing an
   out-of-ceiling function is a *ceiling* violation (previously an effect-check error);
   (iv) narrowing `max` invalidates own modules' cache entries.

## Verification Log

| # | Claim | Evidence |
|---|-------|----------|
| V1 | Repro fails on base | `ailang check m.ail` (base `5f1be97ea`) → `in function asyncExecProcess in std/stream ...` |
| V2 | Ceiling was skipped for `pkg/` only | `package_resolver.go` (pre-change) `if strings.HasPrefix(modID, "pkg/") { return nil }` |
| V3 | Relative sibling imports normalise to `pkg/<self>/...` | test log: `in function execP in pkg/t/a/exec` for `import ./exec` |
| V4 | Imports and builtins elaborate to `VarGlobal` | `elaborate/expressions.go` (`globalEnv` lookup → `VarGlobal`), `elaborate/core.go` `AddBuiltinsToGlobalEnv` |
| V5 | Every Core node has CoreTI before the check | `ValidateCoreTypeInfo` runs first in `runPostTypeCheckPhases` (strict) |
| V6 | Cache key excluded the manifest | `cache_key.go` `ModuleCacheKey(identity, source, depDigests)`; measured narrow→pass-from-cache on base |
| V7 | `walkCore` is not exhaustive | `core_sanity.go` walkCore switch has no RecordUpdate/Tuple/Array |
| V8 | Effect-checker charges lambda bodies today (so closures failed before, as effect errors) | mutation M1: with (b) removed, closure cases fail with `effect checking failed ... 'mk'` |
| V9 | Zero-arg lambda `\. e` and `func() ...` do not parse | `PAR_UNEXPECTED_TOKEN` in the first test draft; fixtures use `\c. ...` |
| V10 | tools/video genuinely uses Process | `daneel/tools/video/*.ail:320` `exec("/usr/bin/avconvert", ...)` |

## Quorum

Trigger 1 fires (the rule redefines a security-relevant declaration and needs Mark's
ratification). Quorum was **not** run in this session by instruction (no evals/quorum); the
ruling is left to Mark (see Open Questions).

## Success Criteria

- [x] Exact Daneel repro passes with `max = ["Stream"]`
- [x] Soundness controls fail the ceiling (not just "some error"): direct call, private
      helper never called, returned closure, HOF pass-through, record field, closure stored
      in a local and never called, sibling via `./`, bare canonical, and `pkg/<self>/` imports
- [x] Precision controls pass: import-without-reference, hook typed wider than its body,
      caller-supplied callback stored, own-sibling reference whose type carries wider rows
- [x] Narrowed ceiling is re-checked despite the compile cache
- [x] Mutation-tested (see sprint plan)
- [x] Corpus: 0 new failures, 4 false positives cleared
- [x] `make test`, `make lint`, `make verify-examples`, file sizes
- [x] Docs (`docs/docs/guides/packages.md`) + CHANGELOG

## Related Documents

- `design_docs/implemented/v0_9_5/m-pkg-package-system.md` — origin of `[effects].max` (A4)
- `design_docs/planned/ailang-core-triage/stored-lambda-effects-charge-enclosing-fn.md` — the
  effect-checker change this design must stay sound under
- `design_docs/planned/ailang-core-triage/effect-checker-let-shadowing.md` — name-keyed
  resolution over-charges; over-charging is sound for the ceiling
- `design_docs/planned/v1_1_0/m-effect-handlers.md` (search match, unrelated)

## Open Questions (for Mark)

1. **Ratify the rule**: "declared rows of own functions + effects of every reference to a
   non-own global, wherever it occurs". Callbacks supplied by callers are *not* charged
   unless performed.
2. **`daneel/tools`** (`sunholo/daneel_tools`, max without `Stream`): after the fix it
   reports a *genuine* violation — `serve/daneel_serve.ail:249 notice(...) ! {Stream}` —
   previously hidden behind the std/stream false positive. Daneel should either add `Stream`
   or keep `serve/` out of that package's graph.
3. Typeclass instances with effectful methods: is that expressible today? If so, add a
   fixture.
4. The module cache has an occasional spurious MISS for an unchanged module (the test tolerates
   it); a key-determinism bug worth its own triage row.

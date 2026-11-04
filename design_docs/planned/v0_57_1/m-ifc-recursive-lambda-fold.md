# M-IFC-RECURSIVE-LAMBDA-FOLD: finite summaries for lambdas that reach themselves through recursion

**Status:** Implemented (branch `fix/ifc-callable-blowup`)
**Priority:** P0: it blocks registry publishes. The validator refuses `sunholo/specstore@0.3.0` on memory.
**Created:** 2026-11-04
**Introduced by:** `3484c6789` "fix(ifc): complete callable propagation and wrapper inference without silent cutoffs (refs #1719)". The first release with it is v0.56.0.
**Scope:** `internal/types` IFC pass only (`ifc_callable.go`, `ifc_summary_local.go`, `ifc_analysis_limits.go`, new `ifc_callable_fold.go`).
**Quorum:** none of the four attended triggers fires:
- no design-freeze items;
- no "override" row in the conflict surface;
- no cost, KPI or banked-data surface;
- every premise was verified in this repository (V1–V12).

## Problem

The registry validator stopped `ailang check --package .` of specstore 0.3.0 at 903 MiB, against its 896 MiB cap, twice (V1). The publisher measured 4.5 s and 74 MB locally and blamed the registry. Their local binary predates v0.56.0 (V2). On v0.56.1 and v0.57.0 the same source costs 1.4 GB and ends in an `information-flow analysis limit exceeded (summary work budget)` error. So the package is refused on every current compiler whatever memory it is given.

One 239-line module, `audit.ail`, accounts for all of it (V3). The IFC pass's module fixed point (`summariseModule`) never converges (V5). Its doc comment says summaries "are drawn from a finite set (labels and callables written in the module), so the loop terminates". That premise is false for lambdas, for three reasons (V6, V7):

- **Inline keys.** A lambda alternative (`fnLambda`) carries its result callable and deferred applications inline, and `fnKey` content-addresses them.
- **The fallback adds the lambda to the result.** The fallback for an imported or unknown callee (`applyOpaque`, design F2) adds every callable argument to the call's result. So `flatMap(\b. …, xs)` "may return" the lambda.
- **Recursion nests it.** When the lambda's body recurses, `headingsIn(xs) = flatMap(\b. … headingsIn(…) …, xs)`, round *n*'s λ has round *n−1*'s λ in its result. The fixed point is an infinite ascending chain:

  ```
  round 2: L:λ1{…}->[L:λ1{…}->[O|P:λ1#0^0]|O|P:λ1#0^0]
  round 3: L:λ1{…}->[L:λ1{…}->[L:λ1{…}->[O|…]|…]|…]
  ```

Each round deepens the key by about 150 bytes. 726 rounds later the 1e6-step budget trips, having kept every key: 1.4 GB, 60 s. The work budget counts steps, not bytes, so it cannot bound memory.

The pattern is ordinary code: recursion over a tree through `flatMap` or `map`, as in `tablesIn` and `headingsIn` (V6). Any module that also has IFC requirements and recurses through a lambda hits it.

## Design

### 1. Fold a lambda's self-reference (the fix)

`fnSelf` is a new callable kind meaning "lambda X itself", valid only inside an instance of lambda X.

- **`foldSelf(n)` normalises every fresh instance.** It runs where `lambdaVal` builds one, and where `substCallable` rebuilds one, because substitution can bring an instance of X inside it.
  - Every instance of X nested anywhere in `n.result` or `n.deferred` is merged into `n`: its deferred applications are unioned in and its result alternatives joined in.
  - The nested instance is then replaced by `fnSelf(X)`.
- **`resolveSelf(cl, inst)` replaces `fnSelf(X)` with `inst` wherever a lambda's contents are read.** There are three such places:
  - `applyCallable` reads the result;
  - `replay` reads the deferred applications (it gains a `self` parameter, `nil` for module functions);
  - `resultN` reads the result.
- **An unresolved `fnSelf` reaching an application is a programming error and panics.** The IFC guard does not recover that panic.

**Soundness.** A callable is a set of alternatives, and applying it checks every alternative's demands. Merging two instances of the same literal into one whose deferred set is the union replays every demand of both, so it checks at least as strictly as keeping both. Folding therefore never drops a demand (test 3 pins the union). In practice the merged instance adds nothing: a nested instance is an earlier fixed-point round of the same lambda, so its demands are a subset of the outer one's. That is why the end-to-end tests cannot catch a broken merge and the unit test does.

**Precision.** A self-reference resolves to the instance with its own parameters still symbolic. This is the lambda value itself, and it is applied later with its own arguments. The unfolded tree substituted the *outer* activation's arguments into the inner instance, so resolution is never less precise than before.

**Termination.** After folding, no instance of X occurs inside X. Lambda nesting is bounded by the number of lambda literals, labels and declared demands are finite, and the chain is gone (V5 re-measured: specstore converges in a handful of rounds).

### 2. Bound summary size, not just steps (defence in depth)

`summariseModule` now fails closed with `information-flow analysis limit exceeded (summary size)` when one function's canonical summary exceeds 64 KiB (`ifcSummaryBytes`). The largest real summary seen is 3.5 KiB (specstore `audit1`). Any future non-converging chain, such as the probe `res` counter, now fails in seconds at a few MB instead of 1.4 GB. With the fold disabled, both repro tests now fail in under 3 s instead of hanging (R2).

## Conflict surface

**Position extended:** the IFC callable domain (`ifcFn` kinds). No syntax, HM type, CoreTI, iface or codegen change.

**Other constructs living there:**
- Module recursion by name (`fnTop`, looked up through `c.summaries`): unchanged, and it already had indirection.
- Closures capturing a labelled variable: their `ρ` of other owners is substituted as before; the fold only cuts instances of the *same* literal.
- Lambdas returned or stored in records and lists: unchanged unless they reach themselves.
- The exported-summary path (`producerCallableResult`): it reads alternative kinds only. `fnSelf` falls to the conservative `joinLabels` default and in practice never reaches it (V11).

**Must still work:**
- the IFC test corpus (`ifc_*_test.go`, including `ifc_function_value_corpus_test.go`, `ifc_closure_test.go` and `ifc_function_value_depth_test.go`);
- every `examples/` and `std/` module;
- every package in `ailang-packages`.

All three pass, with output identical to the base binary (R3).

**Intentional change:** a module that used to stop at "analysis limit exceeded" now finishes and can report real violations the limit hid. specstore's `audit.ail:150` is one (V9).

## Axioms

- **A1 (determinism):** improved. A module that never converged now does, with stable keys.
- **A3 (explicit effects):** untouched.
- **A4 / A7 (soundness, fail loudly):** preserved. The fold is a join, and the new size limit fails closed with a diagnostic, never a partial summary.
- **Machine decidability:** improved. The check terminates on ordinary recursive code.
- **No hard violations.**

## Verification Log

| # | Claim | Evidence |
|---|---|---|
| V1 | The validator refused specstore@0.3.0 on memory, running v0.57.0 | Cloud Run logs 2026-11-04 12:56:51 (revision `01127-zj7`, image `validator:v0.57.0`): "memory budget (896 MiB; observed 903 MiB)" |
| V2 | The publisher's 74 MB is a pre-v0.56.0 compiler | Same source (`origin/specstore-0.3` = PR #178 head `0ec210d`). Cold `check --package .`: `3484c6789^` 4.75 s / 81 MB / clean; v0.56.1 139.5 s / 1.45 GB / 4.2e12 instr; v0.57.0 63 s / 1.37 GB / 1.59e12 instr |
| V3 | `audit.ail` alone carries the cost | v0.57.0 single-file checks: `rules.ail` 50 MB, `model.ail` 60 MB, `audit.ail` 59.7 s / 1.43 GB |
| V4 | The hot path is the callable substitution | macOS `sample` at t = 25 s: `ifcChecker.substCallable` in 43,973 frames, the next IFC frame 545; the rest is GC and memmove |
| V5 | The fixed point does not converge | Temporary per-round trace in `summariseModule`: 26,124 function walks over 726 rounds. `tablesIn`'s fingerprint reaches 106 KB and `headingsIn`'s 56 KB, growing every round; every other function is stable from round 2 |
| V6 | An 11-line module reproduces it | `headingsIn(blocks) = flatMap(\b. … headingsIn(objs(b,"blocks")) …, blocks)` plus a `{not client}` sink: the fingerprint nests one more `L:λ1{…}->[…]` per round (transcript in the session notes) |
| V7 | Mechanism: the F2 fallback adds callable arguments to the result, and lambda keys inline their result | `applyOpaque`: `result = joinCallable(result, a.fn)`; `fnKey` for `fnLambda` ends in `callableKey(f.result)` |
| V8 | The regression entered in v0.56.0 | `git tag --contains 3484c6789` → v0.56.0, v0.56.1, v0.57.0 |
| V9 | The line-150 violation is genuine and independent of the fold | `audit.ail` with both recursive calls replaced by `[]`: the unfixed binary reports the same `audit.ail:150:28 … <client> ⊔ <structure> reaches parameter "raw" of present` in 0.27 s |
| V10 | specstore's other error (`map` with an `FS` lambda, `model.ail:91`) is not this bug | It is the documented breaking soundness change in v0.57.0 (#1958, `changelogs/v0.32-current.md` line 9, migrate to `mapE`). With `mapE`, 10/12 modules pass and only `audit.ail` (and `main.ail`, which imports it) remain, on V9 |
| V11 | Nothing else reads a lambda's result or deferred list (negative existence) | `grep -n '\.result\b\|\.deferred\b\|fnLambda' internal/types/*.go`: only `fnKey`, `applyCallable`, `replay`, `substCallable`, `resultN`, `lambdaVal`. `ifc_summary.go:241` `producerCallableResult` switches on `alt.imported`/`fnTop` and defaults to `joinLabels` |
| V12 | #2078 is unrelated | #2078 is an effect-validator bug in `internal/elaborate/scc.go` (design `M-SCC-SHADOW-AWARE-CALLGRAPH`), not IFC |

## Results

| # | Result |
|---|---|
| R1 | `ifc_callable_fold_test.go`, each on a 30 s deadline so a regression fails instead of hanging:<br>(1) the specstore shape converges in < 2,000 work units;<br>(2) a `{not client}` demand inside the recursive lambda is still reported when client data reaches it;<br>(3) `foldSelf` unions a nested instance's deferred applications, and `resolveSelf` restores the instance |
| R2 | Mutants, made by editing a scratchpad copy:<br>• fold disabled → (1) and (2) fail in 2.8 s and 1.9 s on the new size limit (before the limit, the suite hung for more than 5 min);<br>• merge dropped → (3) fails |
| R3 | No behaviour change on converging code. Base `c96b71802` vs fix:<br>• all 530 `.ail` files under `examples/` and `std/`: identical `check` output;<br>• all 45 `ailang-packages` packages: identical once line order is normalised (4 differed only in the order of parallel-emitted warnings);<br>• `go test ./internal/types/ ./internal/pipeline/ ./internal/iface/` green |
| R4 | specstore 0.3.0 cold `check --package .`: **0.55 s, 72 MB, 1.23e10 instructions**, from 63 s / 1.37 GB / 1.59e12 (−99%). The remaining errors are the package's own (V9, V10) |

## Not in scope

- **Whether `<structure>` should be ⊑ `<client>`.** That would let a declassified shape flow back into a client-scoped declassifier. It is a labelling-policy question for the IFC design, not a performance bug, and specstore can restructure `pickHeader` instead.
- **Making the work budget memory-aware in general.** The size limit covers the measured failure.

# M-WASM-TYPECHECK-FLOAT-DIVERGENCE — native CLI accepts code that WASM type-checker rejects with "float vs int"

**Status**: ✅ IMPLEMENTED (v0.22.x, 2026-05-21) — root cause turned out to be **missing imported-type-alias and parameter/return annotation propagation** in the WASM `ModuleRegistry.LoadModule` path (`internal/repl/module_registry_load.go`), not any of the four pre-sprint hypotheses (H1-H4). The CLI pipeline propagates both (`pipeline_module_compile.go:201-219`); the WASM path didn't, so declared signatures like `currentLeader(st: CommonsState) -> Persona` weren't constraining inference — the type-checker inferred open-record signatures from usage and free type variables leaked across modules. Adding a `Fractional` constraint (e.g. `if g > 0.5 then ...`) pushed the solver into binding the leaked vars to the wrong concrete type (`int` in browser, `string` in Go-test). **Fix**: ~50 LOC mirroring the CLI pipeline. **Regression test**: `internal/repl/wasm_float_divergence_test.go`. See changelog entry _M-WASM-TYPECHECK-FLOAT-DIVERGENCE_ in `changelogs/v0.10-current.md` and `internal/types/testdata/wasm_float_divergence/DIAGNOSIS.md` for the full forensic trail.
**Target**: v0.22.x
**Priority**: P1 — surfaces a soundness gap between the native and WASM type-checkers; affects any browser demo that uses cross-module imports with float-returning helpers + Result match arms with float literals
**Estimated**: 2–4 days (investigation + minimal fix; isolation of MWE is the main unknown)
**Risk**: Medium — touches the type-checker hot path; mitigated by feature-flag if needed
**Source**: 2026-05-21 cognitive_commons restoration session — discovered when restoring `compose_user_prompt` conversation dynamics broke the WASM harness on an unrelated `jnum(sx)` call in commons_browser.ail.

## Problem

The native CLI type-checker accepts code that the WASM-compiled type-checker rejects with a "float vs int" unification error — even though both are built from the same source commit. Reproduces deterministically on `v0.20.1-71-g110b5d92-dirty` (both targets) and is not a budget-exceeded case; the error fires within ~150ms of starting `loadModule()`.

### Specific symptom

The WASM-side error message:
```
type error in speakJson: type unification failed at
[function application at cognitive_commons/services/commons_browser.ail:201:43]:
failed to unify parameter 0: cannot unify type constructors: float vs int
```

Line 201 col 43 is `jnum(sx)` where `sx` is bound by:
```ailang
let sx = match score_result { Ok(s) => s.x, Err(_) => 0.0 };
```

`s.x` is `float` (from `JudgeScore = {x: float, y: float}` declared in persuasion.ail).
The `Err` arm's `0.0` is a float literal.
`jnum` expects a float parameter.
On the CLI: `sx: float`, no error.
On WASM: `sx: int`, fails to unify with `jnum`'s float parameter.

### Trigger

The bug is **not** in commons_browser.ail itself — its source is unchanged. It only manifests when **another module in the dependency chain (citizen.ail) is "enriched"** with a particular shape:
- Multiple new `pure func` helpers that take or return `float`
- A larger `compose_user_prompt` function body that uses those helpers
- An additional type import (`Point`) from a sibling module

Reverting *any* of those changes (or all of them) eliminates the error.

The minimal-attempt repro at `/tmp/wasm_float_repro/` (a two-module example mimicking the shape) does NOT reproduce — so the trigger involves either a specific identifier-count threshold, a specific pattern in `compose_user_prompt`'s body, or an interaction with `compose`/`speak`'s effect-row constraints. **Isolating the MWE is the first deliverable** (see Implementation M1 below).

### In-the-wild reproduction (frozen for analysis)

The demos repo `sunholo-data/ailang-demos` preserves the failing shape:
- Failing commits attempted (then reverted): around `5a2ea73` and predecessors on `main`
- The two-attempt sequence is documented inline in `cognitive_commons/services/citizen.ail` (TODO comment) and `cognitive_commons/services/commons_browser.ail` (similar note)
- The full demos-repo file pair plus this design doc + the harness gives a reproducible test setup:
  ```bash
  cd /Users/mark/dev/sunholo/demos
  git log --oneline cognitive_commons/services/citizen.ail | head -5
  # → look at the commits where my prompt-enrichment attempts were applied; cherry-pick to see the failure
  node scripts/wasm-loadmodule-harness.js   # would fail with the float-vs-int error
  ```

## Why this matters

1. **Soundness asymmetry**: AILANG is supposed to have a single type system. Native and WASM diverging means the language has two semantics. This breaks the trust contract — code that `ailang check` blesses can fail in the browser, and the failure message is misleading (points at unrelated callsites).

2. **Blocks our just-shipped WASM-Friendly Patterns guide**: the Pattern 2 refactor ("destructure tagged unions once into a record") is the *recommended* shape we documented in `ailang prompt | grep -A 200 "Idiomatic AILANG"`. We then hit this bug while trying to apply Pattern 2 to a real demo and had to revert. Embarrassing: the bug makes our own prevention guide unusable.

3. **Forces business logic JS-side**: the cognitive_commons demo currently builds its rich AI-author prompt in JavaScript and passes it through AILANG as an opaque string, sidestepping the AILANG type-checker. That's the wrong architecture — prompt-building is exactly the kind of logic AILANG's effect-typed system is *for*. The bug pushes work back to the untyped layer.

## Root cause hypotheses (ranked)

### H1 — Literal-defaulting in match arms diverges on WASM

The native CLI defaults `0.0` to `float` based on context; the WASM build may apply defaulting at a different unification step where the `Err(_) => 0.0` arm is type-checked before the `Ok(s) => s.x` arm provides the float constraint. If the arm-order is non-deterministic (e.g., goroutine ordering in the constraint solver), native happens to do Ok-then-Err and gets float, WASM happens to do Err-then-Ok and gets int.

Investigation entry point: `internal/types/typechecker_literals.go::inferLit` — check for OS-conditional logic, map iteration over arms (Go map iteration is non-deterministic), or platform-specific float-default config.

### H2 — Cross-module constraint propagation truncated by WASM budget

The M-WASM-TYPECHECK-LIMITS budget (2 seconds wall-clock) might truncate constraint solving mid-pass on WASM. If citizen.ail's larger body pushes commons_browser's import-resolution-and-constraint phase past some internal checkpoint, the solver returns a partial state where `sx` is bound to `int` (a placeholder before defaulting). Native has no budget, completes the pass, sx becomes float.

Investigation: instrument `internal/types/unification_core.go::Unify` to log when budget-exceeded happens; check whether the failing call site is downstream of a budget trip.

### H3 — Iface / scheme serialization

`internal/iface/builder.go` writes module schemes to disk on the CLI but holds them in memory on WASM. Maybe the in-memory representation drops float annotations or interns floats differently. This was the source of the May 2026 M-SCHEME-IMPORT-PRESERVE-ADT-HEAD bug.

Investigation: dump the iface JSON for citizen.ail on CLI and the in-memory iface on WASM (add a `--dump-iface` flag if needed). Diff them.

### H4 — Iteration-order non-determinism in constraint solving

The constraint solver may iterate over a map of pending constraints. Go map iteration order is randomized per process. If the solver depends on insertion order for soundness, native and WASM can diverge. This was the failure mode for several prior type-checker fixes (M-POLY-ORD, M-AI-EFFECT-MODES M2).

Investigation: search for `range *map*` in `internal/types/inference.go` and the unification code; convert to sorted iteration where order matters.

## Implementation milestones

### M1 — Isolate the MWE (1 day, ~50 LOC test fixture)

Write a self-contained two-module example that reproduces the failure. Likely needs to be larger than my current `/tmp/wasm_float_repro` attempt (~50 lines per module). Bisect the working citizen.ail vs the failing variant from this session.

**Acceptance**: a `internal/types/testdata/wasm_float_divergence/` directory with two `.ail` files where `ailang check` passes both but `node scripts/wasm-loadmodule-harness.js` (or equivalent Go-test harness running the WASM binary) returns exit 4 with the "float vs int" error.

### M2 — Diagnose (1–2 days)

Walk through H1–H4 in priority order. The MWE from M1 should make this tractable — add `DEBUG_TYPES=1` logging, compare native vs WASM execution traces of `Unify`/`inferLit`/`inferMatch` on the MWE.

**Acceptance**: a written diagnosis identifying the exact code path responsible, with a one-line explanation of why native and WASM diverge.

### M3 — Fix (1 day, ~100 LOC)

The fix depends on M2's diagnosis. Likely a sort+stabilize pass on map iteration, or a literal-defaulting reorder, or a WASM-specific iface-loading bypass. Behind a feature flag if risky (`AILANG_WASM_TYPECHECK_DETERMINISTIC=1` to opt in until confidence builds).

**Acceptance**:
1. MWE from M1 passes the WASM harness.
2. Original demos-repo commit that surfaced this (the prompt-enrichment attempts in `cognitive_commons/services/citizen.ail`) can be reapplied without breaking the harness.
3. `make test` clean.
4. No regression in `internal/types/` benchmarks (existing baseline from M2 of M-WASM-TYPECHECK-ITERATIVE design).

### M4 — Roll back the JS-side workaround in cognitive_commons (0.5 day)

Once M3 lands and we've verified the WASM type-checker matches CLI semantics, undo the `ailangSpeak()` JS-side prompt enrichment (commit `5a2ea73` in `sunholo-data/ailang-demos`) and restore the AILANG-side `compose_user_prompt` enrichment that we documented and committed-then-reverted this session.

**Acceptance**: cognitive_commons builds the rich prompt entirely in AILANG, `recent_dialogue` parameter only carries dialogue lines (its original semantic).

## Acceptance gate

```bash
# In the ailang repo:
node internal/types/testdata/wasm_float_divergence/run.js
# → exit 0; both modules load without "float vs int" errors

# In the sister demos repo, after restoring the AILANG-side prompt enrichment:
node scripts/wasm-loadmodule-harness.js
# → exit 0; all 5 cognitive_commons modules load cleanly
```

## Workarounds (in effect today)

1. **Build the rich prompt JS-side** and pass it via the existing `recent_dialogue` parameter — current cognitive_commons demo shape (commit `5a2ea73`).
2. **Don't apply Pattern 2 (unpack-helper) across modules** in WASM-targeted code. Inline three back-to-back matches on the same Result instead, even though they're ugly. The cliff is real but the helper triggers the bug.
3. **Avoid adding float-returning helpers in modules that are imported by browser entrypoints** until M3 lands.

## Prevention infrastructure (already in place)

- **Headless WASM harness** (`demos/scripts/wasm-loadmodule-harness.js`) — caught this bug. Would catch a regression of the fix too.
- **Boot-banner diagnostic** in `cognitive_commons/index.html` — surfaces budget/type errors to the user without needing DevTools.
- **`make build-wasm`** + the freshness checker (`demos/scripts/check-wasm-freshness.sh`) — eliminates the "stale wasm" red herring.

What's missing: a CI gate in the ailang repo that runs the harness on every PR touching `internal/types/`. Should land alongside M1 — see `cmd/wasm/testdata/` for analogous harness setups.

## Related work

- **M-WASM-TYPECHECK-LIMITS** (v0.22.x, shipped) — the wall-clock budget guard. Same diagnostic toolchain that enables this investigation.
- **M-WASM-TYPECHECK-ITERATIVE** (deferred to v1.1.x+) — the long-term fix for WASM type-checker scale. This (M-WASM-TYPECHECK-FLOAT-DIVERGENCE) is the *correctness* counterpart to that *scale* sprint.
- **M-SCHEME-IMPORT-PRESERVE-ADT-HEAD** (v0.22.0, shipped, commit 3325d39f) — last cross-module type-info propagation bug. The diagnosis pattern (iface dump diff, exhaustive type-switch audit) is reusable here.

## Open questions

1. Does this also affect non-`float` types? E.g., does a `string` literal in `Err(_)` arm + cross-module helper cause analogous "string vs int" failure? Or is it `float` specifically (matching the Num defaulting machinery)?
2. Does **adding** a helper trigger it, or does **removing** a helper trigger it? My session was add-helpers-fail. Worth bisecting the other direction.
3. Is the bug related to `Iceland` ([the brittle WASM iface serialization]) — could the WASM iface for citizen.ail be losing the float type and falling back to a default int?
4. Does `--source mcp` vs `--source embedded` (prompt-loading mode) affect the WASM type-checker's behaviour? (Unlikely but cheap to verify.)

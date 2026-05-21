# Sprint Plan: M-WASM-TYPECHECK-FLOAT-DIVERGENCE

**Sprint ID**: M-WASM-TYPECHECK-FLOAT-DIVERGENCE  
**Design Doc**: [m-wasm-typecheck-float-divergence.md](m-wasm-typecheck-float-divergence.md)  
**Created**: 2026-05-21  
**Target**: v0.22.x  
**Duration**: 2.5 days  
**Risk**: Medium — type-checker hot path; bounded by WASM-only scope  
**Total LOC Estimate**: ~280 (debug instrumentation + fix + test fixture + demo restore)

---

## Sprint Goal

Fix the WASM type-checker so that `float`-typed values in `match` arms over cross-module ADTs are inferred correctly. The trigger is deterministic: adding any `float`-parameterised helper to `citizen.ail` causes `commons_browser.ail` to fail with "float vs int" — even though the CLI passes both. After this sprint, restoring the reverted `compose_user_prompt` enrichment in the demos repo should work end-to-end.

---

## Pre-Sprint Analysis

### Root-cause investigation summary (done before sprint start)

The investigation before sprint creation narrows the root cause to **two suspects** in `internal/repl/module_registry_load.go`:

**Suspect A — Unqualified typeEnv injection (lines 281–290)**

```go
for _, mod := range mr.modules {           // map iteration — non-deterministic order
    for exportName, export := range mod.Exports {
        typeEnv = typeEnv.ExtendScheme(exportName, export.Scheme) // bare name, no module prefix
    }
}
```

ALL exports from ALL previously-loaded modules are injected into every subsequent module's typeEnv under their **bare, unqualified names**. On the CLI, the equivalent data flows through disk-serialised ifaces under qualified names (`citizen.gap_force`). The bare injection is invisible most of the time (commons_browser only references its own VarGlobals via qualified keys in `tc.globalTypes`), but it puts the schemes — with their free `TVar2` nodes — into the same environment chain as the current module's fresh type variables. Under certain conditions (still to be confirmed in M2) this creates spurious variable sharing.

**Suspect B — `elabCtors` / `mr.constructors` double-registration (lines 253–263)**

```go
// First: register imported constructors
for ctorName, ctor := range mr.constructors {    // maps — non-deterministic order
    registerAdtFactory(...)
}
// Then: register "local" constructors — but elabCtors includes ALL elaborator-known ctors
// = prev-module ctors (already registered above) PLUS current module's ctors
for ctorName, ctorInfo := range elabCtors {      // map — non-deterministic order
    registerAdtFactory(...)
}
```

`elabCtors` from `elaborator.GetConstructors()` contains everything the elaborator was seeded with (lines 41–52), which is the full `mr.constructors` set PLUS anything new in the current module. So every imported constructor's factory scheme is built **twice**, but with a freshly-allocated `TVar2{Name: "t0"}` on each call. If the second call produces a structurally identical but pointer-distinct object, the unification solver may treat them as different type variables.

This is the more likely direct cause of the float-vs-int divergence. The `Ok(a) -> Result[a, e]` scheme for `Ok` gets registered twice: once from `mr.constructors` (from the original module that defined the `Result` type), and once from `elabCtors` when citizen.ail is loaded with its new float helper. Adding the float helper changes the elaboration state enough that `elabCtors` includes a different-typed reference for `JudgeScore`-related constructors, subtly altering the second factory registration.

### Key files

| File | Lines of interest |
|------|-------------------|
| `internal/repl/module_registry_load.go` | 41–52 (ctor seeding), 253–263 (double registration), 281–290 (typeEnv injection) |
| `internal/types/typechecker_patterns.go` | 27–91 (`inferMatch` — arm order and resultType capture) |
| `internal/types/typechecker_defaulting.go` | 107–181 (`defaultAmbiguitiesTopLevel` — Fractional → float) |
| `internal/types/typechecker_core.go` | 376–431 (`InferWithConstraints` — freshCounter persistence) |

---

## Milestones

### M1 — Reproduce: add float helper and run WASM harness (0.5 day, ~30 LOC)

**Goal**: Confirm the bug is live and create the canonical repro.

**Tasks**:

1. Add minimal float helper to citizen.ail in demos repo:
   ```ailang
   pure func gap_force(gap: float) -> string =
     if gap > 0.5 then "high" else "low"
   ```
2. Rebuild WASM: `make build-wasm && cp bin/ailang.wasm /Users/mark/dev/sunholo/demos/`
3. Run harness: `cd /Users/mark/dev/sunholo/demos && node scripts/wasm-loadmodule-harness.js`
4. Confirm exit ≠ 0 with "float vs int" error at commons_browser.ail
5. Create a self-contained Go test fixture in `internal/types/testdata/wasm_float_divergence/`:
   - `two_modules_test.go` — a Go test that calls `LoadModule` sequentially with the trigger pattern
   - Aims to reproduce via Go (not Node.js) for faster iteration in M2/M3

**Acceptance**:
- WASM harness exits non-zero with the documented error after adding the helper
- Go test either reproduces (ideal) or confirms the repro requires the full multi-module WASM path (still useful)

**LOC**: ~30 (Go test fixture, rest is demos-repo edits)

---

### M2 — Diagnose: add targeted debug probes (1 day, ~80 LOC)

**Goal**: Identify the exact code path that diverges between native and WASM.

**Tasks** (work through H1→H4 with the trigger code live):

1. **Probe B — constructor double-registration** (most likely based on pre-sprint analysis)
   - Add a `debugRegisterAdtFactory` flag to `LoadModule`; when set, log each `registerAdtFactory` call with the factory key and the addresses of the `TVar2` nodes created
   - Load `citizen` with and without `gap_force`; diff the logs
   - Check: does the factory for `Ok` or a JudgeScore-related ctor differ between the two runs?

2. **Probe A — typeEnv injection inspection**
   - Add a `dumpTypeEnvKeys` helper; call it after the injection at line 290 when loading commons_browser
   - Check: does the set of injected names change when citizen.ail has the float helper?

3. **Probe H1 — match arm ordering / Fractional defaulting**
   - Add `tc.debugMode = true` before type-checking commons_browser's `speakJson` declaration
   - Run with the trigger; capture the defaulting trace for the `0.0` literal in the `Err` arm
   - Check: is the `Fractional[tN]` constraint present, and does tN default to float or int?

4. **Write diagnosis** — one-paragraph description of the exact code path responsible, with the specific line numbers and why native/WASM diverge.

**Acceptance**:
- A written diagnosis (in a `DIAGNOSIS.md` alongside the testdata or as a comment in the test) confirming which hypothesis is correct and naming the specific lines/variables involved

**LOC**: ~80 (debug instrumentation, mostly in `module_registry_load.go`)

---

### M3 — Fix (1 day, ~140 LOC)

**Fix strategy depends on M2; two likely paths:**

**Path B-fix (constructor double-registration)**:

Deduplicate the `elabCtors` loop: before calling `registerAdtFactory` in the "local constructors" pass, check if the constructor was already registered in the "imported constructors" pass and skip it:

```go
// Register $adt factory types for LOCAL constructors (from this module ONLY)
importedCtorNames := make(map[string]bool)
mr.mu.RLock()
for ctorName := range mr.constructors {
    importedCtorNames[ctorName] = true
}
mr.mu.RUnlock()

for ctorName, ctorInfo := range elabCtors {
    if importedCtorNames[ctorName] {
        continue  // already registered above — don't overwrite with new TVar2 instances
    }
    registerAdtFactory(ctorName, ctorInfo.TypeName, ctorInfo.Arity, ctorInfo.TypeParamCount, ctorInfo.TypeParamNames, ctorInfo.FieldTypes)
}
```

**Path A-fix (typeEnv injection)**:

Qualify injected names with module prefix to avoid shadowing / unintended typeEnv lookups:

```go
for modName, mod := range mr.modules {
    for exportName, export := range mod.Exports {
        if export.Scheme != nil {
            qualifiedName := fmt.Sprintf("%s.%s", modName, exportName)
            typeEnv = typeEnv.ExtendScheme(qualifiedName, export.Scheme)
        }
    }
}
```

(This also fixes potential name-shadowing bugs for any module that exports the same short name as another.)

**Tasks**:
1. Apply the fix for the confirmed root cause
2. `make test` — full suite
3. WASM rebuild + harness: `node scripts/wasm-loadmodule-harness.js` must exit 0
4. Verify CLI `ailang check` still passes both files
5. Ensure no regression in `internal/types/` benchmarks (use `make bench`)
6. Add a regression test in `internal/types/testdata/wasm_float_divergence/` that will catch reintroduction

**Acceptance**:
1. WASM harness exits 0 with `gap_force` present in citizen.ail
2. Original demos-repo prompt-enrichment commits (around `5a2ea73`) can be reapplied without breaking the harness
3. `make test` clean
4. No regression in `make bench` for internal/types (< 5% slowdown)

**LOC**: ~140 (fix + regression test)

---

### M4 — Restore demos (0.5 day, ~30 LOC AILANG)

**Goal**: Undo the JS workaround and restore AILANG-side prompt building.

**Tasks** (in `sunholo-data/ailang-demos`):
1. Restore the `compose_user_prompt` enrichment in `citizen.ail`: the address-by-name / react-to-last-move / gap-force helpers documented in the TODO comment
2. Remove the `ailangSpeak()` JS-side prompt enrichment from `cognitive_commons/index.html` (commit `5a2ea73` equivalent)
3. Restore the original `recent_dialogue` semantic (dialogue lines only, not full prompt)
4. Run harness — all 5 cognitive_commons modules load cleanly
5. Smoke-test the demo in a browser

**Acceptance**:
- `node scripts/wasm-loadmodule-harness.js` exits 0 with all 5 modules loading
- `compose_user_prompt` in citizen.ail uses float-typed helpers without WASM errors
- No JS-side prompt assembly

**LOC**: ~30 AILANG (restored helpers), net-negative JS (removal)

---

## Day-by-day breakdown

| Day | Task |
|-----|------|
| Day 1 morning | M1: Add trigger, rebuild WASM, confirm repro, create Go test fixture |
| Day 1 afternoon | M2: Add debug probes, run comparison logs, identify root cause |
| Day 2 | M3: Implement fix, run make test, WASM rebuild + harness validation, regression test |
| Day 3 morning | M4: Restore demos, browser smoke test, commit |
| Day 3 afternoon | Buffer: handle unexpected M2 findings if H1/H4 is more complex than suspected |

---

## Success Metrics

| Metric | Target |
|--------|--------|
| WASM harness exit code | 0 (all 5 cognitive_commons modules) |
| `make test` | All passing |
| `make bench` regression | < 5% |
| AILANG-side prompt building | Restored in demos |
| CI gate for `internal/types/` PRs | Planned for M1 (Go test), CI hook in M3 |

---

## Open questions going into M2

1. Does the double-registration bug apply only to `float`-parameterised constructors, or any constructor? (If the latter, we have a latent correctness bug beyond the float symptom.)
2. Does the typeEnv bare-injection cause any actual name collision in the current demos setup, or is it a separate latent bug?
3. Does the fix for Path B require map-iteration sort stabilisation (determinism), or just deduplication?

---

## Dependencies

- `make build-wasm` must work (verified: in Makefile)
- Demos repo accessible at `/Users/mark/dev/sunholo/demos` (verified)
- M-WASM-TYPECHECK-LIMITS (shipped) — provides the depth budget guard we can piggy-back on for debugging

## Related

- [m-wasm-typecheck-limits.md](m-wasm-typecheck-limits.md) — the scale counterpart (shipped)
- [m-wasm-typecheck-float-divergence.md](m-wasm-typecheck-float-divergence.md) — the design doc this plan executes
